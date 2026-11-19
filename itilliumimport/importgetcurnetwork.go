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

// ImportNetworkEquipmentFromCU — cuClass=146 → SwitchSprRecords.
func ImportNetworkEquipmentFromCU(rows []itilium.CURow) ImportCUStats {
	var st ImportCUStats
	st.Total = len(rows)
	o := orm.NewOrm()
	rs := locality.NewResolver()
	typeCache := map[string]int64{}
	for i, row := range rows {
		aggregateCUImportOne(&st, importOneCUNetwork(o, rs, typeCache, row, i))
	}
	log.Printf("itilliumimport: ImportNetworkEquipmentFromCU: total=%d inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Total, st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}

func importOneCUNetwork(o orm.Ormer, rs *locality.Resolver, typeCache map[string]int64, row itilium.CURow, idx int) cuImportOne {
	var out cuImportOne
	tech := cuTechFromRow(row)
	sn := cuTrim(row.SN)
	if tech == "" && sn == "" {
		log.Printf("itilliumimport: getCU network[%d]: пропуск — нет code/sn", idx)
		out.skipped = true
		return out
	}

	statusLabel := strings.TrimSpace(row.Status.Label)
	mapped, known := MapCUStatusLabel(statusLabel)
	if !known {
		log.Printf("itilliumimport: getCU network %s: неизвестный статус %q", tech, statusLabel)
		out.unknownSt = true
	}

	itiliumSt := 0
	if known {
		itiliumSt = mapped.ItiliumStatus
	}
	if itiliumSt == 2 || itiliumSt == 3 {
		out.inactive = true
	}

	where := cuTrim(row.Where)
	links := cuLinksUnit(row)
	hostname := cuTrim(row.Hostname)
	desc := cuTrim(row.Description)
	model := cuTrim(row.ModelNet)
	mac := cuTrim(row.Mac)
	ip := cuTrim(row.IP)
	owner := cuTrim(row.User)
	provider := cuTrim(row.Provider)
	contractDoc := cuTrim(row.Docs)
	inv := cuTrim(row.Name)
	buhInv := cuTrim(row.BuhCode)
	if buhInv == "" {
		buhInv = cuTrim(row.BuhInv)
	}
	docDate, hasDocDate := cuParseDateRU(cuTrim(row.DataDocs))
	statusHist, hasStatusHist := cuStatusHistoryTime(row)
	sizeU := cuIntField(row.Units)

	if cuSkipPlacementLikeNetwork(where, itiliumSt) {
		out.noLoc = true
	}

	eq := models.SwitchSprRecords{
		TechnicalNumberIT: tech,
		SerialNum:         sn,
		Model:             model,
		MacAddress:        mac,
		ManagementIp:      ip,
		ItiliumStatus:     itiliumSt,
		InventoryNum:      inv,
		InventoryNumBuh:   buhInv,
		Owner:             owner,
		Counterparty:      provider,
		ContractDoc:       contractDoc,
		Description:       desc,
		SizeUnit:          sizeU,
	}
	eq.Name = firstNonEmptyStr(hostname, desc, tech)
	if len(eq.Name) > 200 {
		eq.Name = eq.Name[:200]
	}
	if strings.TrimSpace(eq.Annotation) == "" && desc != "" {
		eq.Annotation = desc
	}
	if hasStatusHist {
		eq.TimeActiveState = statusHist
		eq.StatusHistory = statusHist
	}
	if hasDocDate {
		eq.DateContractDoc = docDate
	}

	trn, err := o.Begin()
	if err != nil {
		out.err = true
		return out
	}

	typeId, err := resolveNetworkEquipmentType(trn, typeCache, cuTrim(row.TypeNet))
	if err != nil {
		log.Printf("itilliumimport: getCU network %s: type: %v", tech, err)
		trn.Rollback()
		out.err = true
		return out
	}
	eq.TypeId = typeId

	if !cuSkipPlacementLikeNetwork(where, itiliumSt) && where != "" {
		site, errSite := resolveSite(trn, rs, where, models.UserSourceManual)
		if errSite != nil {
			log.Printf("itilliumimport: getCU network %s: resolveSite: %v", tech, errSite)
			trn.Rollback()
			out.err = true
			return out
		}
		eq.DatacenterSiteId = site.Id
		if site.Id > 0 && (site.Loc == nil || !site.Loc.Service) {
			cabinetCode, cabinetAlt, locPos := cabinetFromPlacement(links, site.Loc)
			if cabinetCode != "" {
				eq.RackCabinetId, err = resolveRackCabinet(trn, cabinetCode, cabinetAlt, site.Id,
					"Шкаф по данным getCU network, площадка: "+site.Name)
				if err == nil && locPos != nil && eq.StartUnitPos == 0 {
					eq.StartUnitPos = locPos.Start
					if sz := locPos.End - locPos.Start + 1; sz > 0 {
						eq.SizeUnit = sz
					} else if eq.SizeUnit == 0 {
						eq.SizeUnit = 1
					}
				}
			} else {
				eq.RackCabinetId, err = resolveOutsideCabinet(trn, site.Id)
				eq.StartUnitPos = 0
				if eq.SizeUnit == 0 {
					eq.SizeUnit = 0
				}
			}
			if err != nil {
				trn.Rollback()
				out.err = true
				return out
			}
		}
	}

	found := false
	var cur models.SwitchSprRecords
	if tech != "" {
		found = trn.QueryTable(new(models.SwitchSprRecords)).Filter("TechnicalNumberIT", tech).One(&cur) == nil
	}
	if !found && sn != "" {
		found = trn.QueryTable(new(models.SwitchSprRecords)).Filter("SerialNum", sn).One(&cur) == nil
	}

	now := time.Now().Local()
	if !found {
		eq.Id_User = models.UserSourceManual
		eq.Time_Mfy = now
		eq.SourceUpdateType = models.SourceUpdateManual
		if _, err := trn.Insert(&eq); err != nil {
			log.Printf("itilliumimport: getCU network %s: insert: %v", tech, err)
			trn.Rollback()
			out.err = true
			return out
		}
		out.inserted = true
	} else {
		cur.TypeId = eq.TypeId
		cur.ItiliumStatus = eq.ItiliumStatus
		cur.TimeActiveState = eq.TimeActiveState
		cur.DatacenterSiteId = eq.DatacenterSiteId
		cur.RackCabinetId = eq.RackCabinetId
		cur.StartUnitPos = eq.StartUnitPos
		if eq.SizeUnit > 0 {
			cur.SizeUnit = eq.SizeUnit
		}
		cur.InventoryNum = eq.InventoryNum
		cur.InventoryNumBuh = eq.InventoryNumBuh
		fillEmpty(&cur.Name, eq.Name)
		fillEmpty(&cur.Model, model)
		fillEmpty(&cur.MacAddress, mac)
		fillEmpty(&cur.ManagementIp, ip)
		fillEmpty(&cur.Owner, owner)
		fillEmpty(&cur.Counterparty, provider)
		fillEmpty(&cur.ContractDoc, contractDoc)
		if desc != "" {
			cur.Description = desc
		}
		if strings.TrimSpace(cur.Annotation) == "" && desc != "" {
			cur.Annotation = desc
		}
		if hasStatusHist {
			cur.StatusHistory = statusHist
		}
		if cur.DateContractDoc.IsZero() && hasDocDate {
			cur.DateContractDoc = docDate
		}
		if cur.TechnicalNumberIT == "" && tech != "" {
			cur.TechnicalNumberIT = tech
		}
		if cur.SerialNum == "" && sn != "" {
			cur.SerialNum = sn
		}
		cur.Id_User = models.UserSourceManual
		cur.Time_Mfy = now
		cols := []string{"TypeId", "ItiliumStatus", "TimeActiveState", "DatacenterSiteId", "RackCabinetId",
			"StartUnitPos", "SizeUnit", "InventoryNum", "InventoryNumBuh", "Name", "Model", "MacAddress",
			"ManagementIp", "Owner", "Counterparty", "ContractDoc", "Description", "Annotation",
			"StatusHistory", "DateContractDoc", "TechnicalNumberIT", "SerialNum", "Id_User", "Time_Mfy"}
		if _, err := trn.Update(&cur, cols...); err != nil {
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