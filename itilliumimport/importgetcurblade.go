package itilliumimport

import (
	"log"
	"strconv"
	"strings"
	"time"

	"loadmon/itilium"
	"loadmon/locality"
	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// ImportBladeServersFromCU — КЕ getCU cuClass=142 → ServerHardware (FormFactor=2) + ServerSprRecords.
func ImportBladeServersFromCU(rows []itilium.CURow) ImportCUStats {
	var st ImportCUStats
	st.Total = len(rows)
	o := orm.NewOrm()
	rs := locality.NewResolver()
	for i, row := range rows {
		aggregateCUImportOne(&st, importOneCUBlade(o, rs, row, i))
	}
	log.Printf("itilliumimport: ImportBladeServersFromCU: total=%d inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Total, st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}

func importOneCUBlade(o orm.Ormer, rs *locality.Resolver, row itilium.CURow, idx int) cuImportOne {
	var out cuImportOne
	tech := cuTechFromRow(row)
	if tech == "" {
		log.Printf("itilliumimport: getCU blade[%d]: пропуск — нет code/uid", idx)
		out.skipped = true
		out.message = "нет code/uid"
		return out
	}
	out.key = tech

	statusLabel := strings.TrimSpace(row.Status.Label)
	mapped, known := MapCUStatusLabel(statusLabel)
	if !known {
		log.Printf("itilliumimport: getCU blade %s: неизвестный статус %q", tech, statusLabel)
		out.unknownSt = true
	}

	sn := cuTrim(row.SN)
	if sn == "" {
		sn = tech
	}
	hostname := cuTrim(row.Hostname)
	ip := cuTrim(row.IP)
	inv := cuTrim(row.Name)
	where := cuTrim(row.Where)
	links := cuLinksUnit(row)
	model := cuBladeModel(row)
	owner := cuTrim(row.User)
	details := cuDetails(row)
	role := cuTrim(row.Role)
	contractDoc := cuTrim(row.Docs)
	provider := cuTrim(row.Provider)
	buhInv := cuTrim(row.BuhCode)
	if buhInv == "" {
		buhInv = cuTrim(row.BuhInv)
	}
	docDate, hasDocDate := cuParseDateRU(cuTrim(row.DataDocs))
	descOnly := cuTrim(row.Description)
	statusHist, hasStatusHist := cuStatusHistoryTime(row)
	bladeRefName := parseBladeBucketName(cuTrim(row.Bucket))
	bladeNum := cuIntField(row.BayNumber)

	skipLoc := locationUndefinedForCU(where, mapped.ItiliumStatus)
	if skipLoc {
		out.noLoc = true
	}

	trn, err := o.Begin()
	if err != nil {
		log.Printf("itilliumimport: getCU blade %s: Begin: %v", tech, err)
		out.err = true
		return out
	}

	var hw models.ServerHardware
	var spr models.ServerSprRecords
	hw.TechnicalNumberIT = tech
	hw.InventoryNum = inv
	hw.InventoryNumBuh = buhInv
	hw.SerialNum = sn
	hw.Model = model
	hw.ContractDoc = contractDoc
	hw.Counterparty = provider
	hw.Description = descOnly
	hw.BladeNum = bladeNum
	if hasDocDate {
		hw.DateContractDoc = docDate
	}
	if hasStatusHist {
		hw.StatusHistory = statusHist
	}
	hw.FormFactor = 2
	hw.ItiliumStatus = mapped.ItiliumStatus
	hw.PowerState = mapped.HwPower
	hw.Active = mapped.HwActive
	hw.SourceUpdateType = models.SourceUpdateManual
	hw.Id_User = models.UserSourceManual
	hw.Time_Mfy = time.Now().Local()

	spr.Name = hostname
	spr.NameFQDN = hostname
	spr.IPAddress = ip
	spr.SerialNumber = sn
	spr.Owner = owner
	spr.Role = role
	spr.Details = details
	spr.Stat = mapped.Stat
	spr.PowerState = mapped.Power
	spr.Active = mapped.Active
	spr.SourceUpdateType = models.SourceUpdateManual
	spr.Id_User = models.UserSourceManual
	spr.Time_Mfy = time.Now().Local()
	if mapped.Active == 1 {
		spr.ActivatorUser = models.UserSourceManual
	} else if known {
		spr.TimeDeActiveState = time.Now().Local()
		spr.DeActivatorUser = models.UserSourceManual
		out.inactive = true
	}

	if !skipLoc && where != "" {
		site, errSite := resolveSite(trn, rs, where, models.UserSourceManual)
		if errSite != nil {
			log.Printf("itilliumimport: getCU blade %s: resolveSite: %v", tech, errSite)
			trn.Rollback()
			out.err = true
			return out
		}
		if site.Loc != nil && site.Loc.Service {
			out.noLoc = true
			hw.RackCabinetId, hw.StartUnitPos, hw.SizeUnit = 0, 0, 0
		} else {
			datacenterId := site.Id
			pl := locality.ParsePlacement(links)
			rackNum := ""
			startU, sizeU := 0, 0
			if pl != nil {
				rackNum = pl.Cabinet
				if pl.Pos != nil {
					startU = pl.Pos.Start
					sizeU = pl.Pos.End - pl.Pos.Start + 1
					if sizeU < 1 {
						sizeU = 1
					}
				}
			}
			cabinetCode, cabinetAlt, locPos := cabinetFromPlacement(rackNum, site.Loc)
			if cabinetCode != "" {
				rackId, errRack := resolveRackCabinet(trn, cabinetCode, cabinetAlt, datacenterId,
					"Шкаф по данным getCU blade, площадка: "+site.Name)
				if errRack != nil {
					log.Printf("itilliumimport: getCU blade %s: resolveRack: %v", tech, errRack)
					trn.Rollback()
					out.err = true
					return out
				}
				hw.RackCabinetId = rackId
				if startU > 0 {
					hw.StartUnitPos, hw.SizeUnit = startU, sizeU
				} else if locPos != nil {
					hw.StartUnitPos = locPos.Start
					hw.SizeUnit = locPos.End - locPos.Start + 1
					if hw.SizeUnit < 1 {
						hw.SizeUnit = 1
					}
				}
			} else if datacenterId > 0 {
				outsideId, errOut := resolveOutsideCabinet(trn, datacenterId)
				if errOut != nil {
					log.Printf("itilliumimport: getCU blade %s: outside: %v", tech, errOut)
					trn.Rollback()
					out.err = true
					return out
				}
				hw.RackCabinetId = outsideId
			}
		}
	} else {
		hw.RackCabinetId, hw.StartUnitPos, hw.SizeUnit = 0, 0, 0
	}

	if bladeRefName != "" {
		bs := models.BladeSystems{Name: bladeRefName}
		if trn.Read(&bs, "Name") == nil {
			hw.BladeSystemId = bs.Id
			if bs.RackCabinetId != 0 {
				hw.RackCabinetId = bs.RackCabinetId
			}
		} else {
			log.Printf("itilliumimport: getCU blade %s: корзина %q не найдена", tech, bladeRefName)
		}
	}

	if hw.SizeUnit == 0 {
		if u := strings.TrimSpace(row.Units.ValueString()); u != "" {
			if n, err := strconv.Atoi(u); err == nil && n > 0 {
				hw.SizeUnit = n
			}
		}
	}

	var exist models.ServerHardware
	exist.SerialNum = sn
	hwFound := trn.Read(&exist, "SerialNum") == nil

	if !hwFound {
		hw.TimeCreate = time.Now().Local()
		hw.CreatorUser = models.UserSourceManual
		id, errIns := trn.Insert(&hw)
		if errIns != nil {
			log.Printf("itilliumimport: getCU blade %s: insert hw: %v", tech, errIns)
			trn.Rollback()
			out.err = true
			return out
		}
		hw.Id = id
		out.inserted = true
	} else {
		hw.Id = exist.Id
		exist.FormFactor = 2
		exist.ItiliumStatus = hw.ItiliumStatus
		cols := []string{"FormFactor", "ItiliumStatus"}
		keepChassis := hw.BladeSystemId == 0 && exist.BladeSystemId > 0
		if keepChassis {
			bs := models.BladeSystems{Id: exist.BladeSystemId}
			if trn.Read(&bs) == nil && bs.RackCabinetId != 0 && bs.RackCabinetId != exist.RackCabinetId {
				exist.RackCabinetId = bs.RackCabinetId
				cols = append(cols, "RackCabinetId")
			}
		} else {
			exist.RackCabinetId = hw.RackCabinetId
			cols = append(cols, "RackCabinetId")
			if hw.BladeSystemId > 0 {
				exist.BladeSystemId = hw.BladeSystemId
				exist.BladeNum = hw.BladeNum
				cols = append(cols, "BladeSystemId", "BladeNum")
			} else if hw.BladeNum > 0 {
				exist.BladeNum = hw.BladeNum
				cols = append(cols, "BladeNum")
			}
		}
		if exist.SourceUpdateType == models.SourceUpdateManual {
			exist.PowerState = hw.PowerState
			exist.Active = hw.Active
			cols = append(cols, "PowerState", "Active")
		}
		if fillEmpty(&exist.TechnicalNumberIT, tech) {
			cols = append(cols, "TechnicalNumberIT")
		}
		if fillEmpty(&exist.InventoryNum, inv) {
			cols = append(cols, "InventoryNum")
		}
		if fillEmpty(&exist.Model, model) {
			cols = append(cols, "Model")
		}
		if fillEmpty(&exist.ContractDoc, contractDoc) {
			cols = append(cols, "ContractDoc")
		}
		if fillEmpty(&exist.Counterparty, provider) {
			cols = append(cols, "Counterparty")
		}
		if fillEmpty(&exist.InventoryNumBuh, buhInv) {
			cols = append(cols, "InventoryNumBuh")
		}
		if exist.DateContractDoc.IsZero() && hasDocDate {
			exist.DateContractDoc = docDate
			cols = append(cols, "DateContractDoc")
		}
		if descOnly != "" {
			exist.Description = descOnly
			cols = append(cols, "Description")
		}
		if hasStatusHist {
			exist.StatusHistory = statusHist
			cols = append(cols, "StatusHistory")
		}
		exist.Time_Mfy = time.Now().Local()
		exist.Id_User = models.UserSourceManual
		cols = append(cols, "Time_Mfy", "Id_User")
		if _, errUp := trn.Update(&exist, cols...); errUp != nil {
			log.Printf("itilliumimport: getCU blade %s: update hw: %v", tech, errUp)
			trn.Rollback()
			out.err = true
			return out
		}
		hw = exist
	}

	if hw.BladeSystemId == 0 && hw.Active == 1 {
		log.Printf("itilliumimport: getCU blade %s: активный блейд без корзины (sn=%s, bucket=%q)", tech, sn, bladeRefName)
	}

	spr.HWSprId = hw.Id
	var sprsrv models.ServerSprRecords
	sprFound := hw.ServerId != 0
	if sprFound {
		sprsrv = models.ServerSprRecords{Id: hw.ServerId}
		sprFound = trn.Read(&sprsrv) == nil && !models.IsServerSprDestroyed(&sprsrv)
	}
	if !sprFound {
		if strings.TrimSpace(spr.Name) == "" {
			if err := trn.Commit(); err != nil {
				log.Printf("itilliumimport: getCU blade %s: commit (hw only): %v", tech, err)
				out.err = true
			}
			return out
		}
		spr.TimeCreate = time.Now().Local()
		spr.CreatorUser = models.UserSourceManual
		spr.TimeActiveState = time.Now().Local()
		sprId, errIns := trn.Insert(&spr)
		if errIns != nil {
			log.Printf("itilliumimport: getCU blade %s: insert spr: %v", tech, errIns)
			trn.Rollback()
			out.err = true
			return out
		}
		hw.ServerId = sprId
		if _, errUp := trn.Update(&hw, "ServerId"); errUp != nil {
			log.Printf("itilliumimport: getCU blade %s: hw.ServerId: %v", tech, errUp)
			trn.Rollback()
			out.err = true
			return out
		}
		if !hwFound {
			out.inserted = true
		}
	} else {
		sprsrv.HWSprId = hw.Id
		cols := []string{"HWSprId"}
		if sprsrv.SourceUpdateType == models.SourceUpdateManual {
			sprsrv.Stat = spr.Stat
			sprsrv.Active = spr.Active
			sprsrv.PowerState = spr.PowerState
			cols = append(cols, "Stat", "Active", "PowerState")
		}
		if fillEmpty(&sprsrv.Role, role) {
			cols = append(cols, "Role")
		}
		if fillEmpty(&sprsrv.Owner, owner) {
			cols = append(cols, "Owner")
		}
		if fillEmpty(&sprsrv.IPAddress, ip) {
			cols = append(cols, "IPAddress")
		}
		if fillEmpty(&sprsrv.Details, details) {
			cols = append(cols, "Details")
		}
		if _, errUp := trn.Update(&sprsrv, cols...); errUp != nil {
			trn.Rollback()
			out.err = true
			return out
		}
	}

	if err := trn.Commit(); err != nil {
		log.Printf("itilliumimport: getCU blade %s: commit: %v", tech, err)
		out.err = true
		return out
	}
	return out
}