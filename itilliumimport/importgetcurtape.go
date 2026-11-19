package itilliumimport

import (
	"log"
	"strings"
	"time"

	"loadmon/itilium"
	"loadmon/locality"
	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// ImportTapeLibrariesFromCU — cuClass=198 → TapeLibrary.
func ImportTapeLibrariesFromCU(rows []itilium.CURow) ImportCUStats {
	var st ImportCUStats
	st.Total = len(rows)
	o := orm.NewOrm()
	rs := locality.NewResolver()
	for i, row := range rows {
		aggregateCUImportOne(&st, importOneCUTape(o, rs, row, i))
	}
	log.Printf("itilliumimport: ImportTapeLibrariesFromCU: total=%d inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Total, st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}

func importOneCUTape(o orm.Ormer, rs *locality.Resolver, row itilium.CURow, idx int) cuImportOne {
	var out cuImportOne
	tech := cuTechFromRow(row)
	sn := cuTrim(row.SN)
	if sn == "" {
		sn = tech
	}
	if sn == "" {
		log.Printf("itilliumimport: getCU tape[%d]: пропуск — нет sn/code", idx)
		out.skipped = true
		return out
	}

	statusLabel := strings.TrimSpace(row.Status.Label)
	mapped, known := MapCUStatusLabel(statusLabel)
	if !known {
		log.Printf("itilliumimport: getCU tape %s: неизвестный статус %q", sn, statusLabel)
		out.unknownSt = true
	}

	itiliumSt := 0
	if known {
		itiliumSt = mapped.ItiliumStatus
	}
	where := cuTrim(row.Where)
	if cuSkipPlacementLikeNetwork(where, itiliumSt) {
		out.noLoc = true
	}
	if itiliumSt == 2 || itiliumSt == 3 {
		out.inactive = true
	}

	name := firstNonEmptyStr(cuTrim(row.Name), cuTrim(row.Description))
	model := cuTrim(row.ModelTape)
	ip := cuTrim(row.IP)
	hostname := cuTrim(row.Hostname)
	desc := cuTrim(row.Description)
	statusHist, hasStatusHist := cuStatusHistoryTime(row)
	owner := cuTrim(row.User)
	provider := cuTrim(row.Provider)
	contractDoc := cuTrim(row.Docs)
	inv := cuTrim(row.Name)
	buhInv := cuTrim(row.BuhCode)
	if buhInv == "" {
		buhInv = cuTrim(row.BuhInv)
	}
	docDate, hasDocDate := cuParseDateRU(cuTrim(row.DataDocs))
	sizeU := cuIntField(row.Units)

	var tl models.TapeLibrary
	tl.SerialNum = sn
	tl.TechnicalNumberIT = tech
	tl.Name = name
	tl.Model = model
	tl.IPAddress = ip
	tl.PcName = hostname
	tl.ItiliumStatus = itiliumSt
	tl.Description = desc
	tl.Owner = owner
	tl.Counterparty = provider
	tl.ContractDoc = contractDoc
	tl.InventoryNum = inv
	tl.InventoryNumBuh = buhInv
	tl.SizeUnit = sizeU
	if hasStatusHist {
		tl.TimeActiveState = statusHist
		tl.StatusHistory = statusHist
	}
	if hasDocDate {
		tl.DateContractDoc = docDate
	}

	trn, err := o.Begin()
	if err != nil {
		out.err = true
		return out
	}

	if !cuSkipPlacementLikeNetwork(where, itiliumSt) && where != "" {
		site, errSite := resolveSite(trn, rs, where, models.UserSourceManual)
		if errSite != nil {
			log.Printf("itilliumimport: getCU tape %s: resolveSite: %v", sn, errSite)
			trn.Rollback()
			out.err = true
			return out
		}
		if site.Id > 0 && site.Loc != nil && !site.Loc.Service {
			cabinetCode, cabinetAlt, locPos := cabinetFromPlacement("", site.Loc)
			var rackId int64
			if cabinetCode != "" {
				rackId, err = resolveRackCabinet(trn, cabinetCode, cabinetAlt, site.Id,
					"Шкаф по данным getCU tape, площадка: "+site.Name)
				if err == nil && locPos != nil {
					tl.StartUnitPos = locPos.Start
					if tl.SizeUnit == 0 && locPos.End >= locPos.Start && locPos.Start > 0 {
						tl.SizeUnit = locPos.End - locPos.Start + 1
					}
				}
			} else {
				rackId, err = resolveOutsideCabinet(trn, site.Id)
			}
			if err != nil {
				trn.Rollback()
				out.err = true
				return out
			}
			tl.RackCabinetId = rackId
		}
	}

	serverId, err := FindServerByPcName(trn, tl.PcName)
	if err != nil {
		log.Printf("itilliumimport: getCU tape %s: FindServerByPcName: %v", sn, err)
		trn.Rollback()
		out.err = true
		return out
	}
	tl.ServerId = serverId

	exist := models.TapeLibrary{SerialNum: sn}
	found := trn.Read(&exist, "SerialNum") == nil
	now := time.Now().Local()
	if !found {
		tl.TimeCreate = now
		tl.CreatorUser = models.UserSourceManual
		tl.Id_User = models.UserSourceManual
		tl.Time_Mfy = now
		tl.SourceUpdateType = models.SourceUpdateManual
		if _, err := trn.Insert(&tl); err != nil {
			log.Printf("itilliumimport: getCU tape %s: insert: %v", sn, err)
			trn.Rollback()
			out.err = true
			return out
		}
		out.inserted = true
	} else {
		exist.ItiliumStatus = tl.ItiliumStatus
		exist.TimeActiveState = tl.TimeActiveState
		exist.RackCabinetId = tl.RackCabinetId
		exist.StartUnitPos = tl.StartUnitPos
		exist.SizeUnit = tl.SizeUnit
		exist.PcName = tl.PcName
		exist.IPAddress = tl.IPAddress
		exist.InventoryNum = tl.InventoryNum
		exist.InventoryNumBuh = tl.InventoryNumBuh
		fillEmpty(&exist.TechnicalNumberIT, tech)
		fillEmpty(&exist.Name, name)
		fillEmpty(&exist.Model, model)
		fillEmpty(&exist.Owner, owner)
		fillEmpty(&exist.Counterparty, provider)
		fillEmpty(&exist.ContractDoc, contractDoc)
		if desc != "" {
			exist.Description = desc
		}
		if hasStatusHist {
			exist.StatusHistory = statusHist
		}
		if exist.DateContractDoc.IsZero() && hasDocDate {
			exist.DateContractDoc = docDate
		}
		exist.Id_User = models.UserSourceManual
		exist.Time_Mfy = now
		cols := []string{"ItiliumStatus", "TimeActiveState", "RackCabinetId", "StartUnitPos", "SizeUnit",
			"PcName", "IPAddress", "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "Name", "Model",
			"Owner", "Counterparty", "ContractDoc", "Description", "StatusHistory", "DateContractDoc",
			"Id_User", "Time_Mfy"}
		if serverId > 0 {
			exist.ServerId = serverId
			cols = append(cols, "ServerId")
		}
		if _, err := trn.Update(&exist, cols...); err != nil {
			trn.Rollback()
			out.err = true
			return out
		}
	}

	if err := trn.Commit(); err != nil {
		out.err = true
		return out
	}
	return out
}