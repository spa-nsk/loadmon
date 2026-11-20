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

// ImportBladeEnclosuresFromCU — cuClass=159 → BladeSystems.
func ImportBladeEnclosuresFromCU(rows []itilium.CURow) ImportCUStats {
	var st ImportCUStats
	st.Total = len(rows)
	o := orm.NewOrm()
	rs := locality.NewResolver()
	for i, row := range rows {
		aggregateCUImportOne(&st, importOneCUBladeEnclosure(o, rs, row, i))
	}
	log.Printf("itilliumimport: ImportBladeEnclosuresFromCU: total=%d inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Total, st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}

func importOneCUBladeEnclosure(o orm.Ormer, rs *locality.Resolver, row itilium.CURow, idx int) cuImportOne {
	var out cuImportOne
	name := cuTrim(row.Name)
	if name == "" {
		log.Printf("itilliumimport: getCU enclosure[%d]: пропуск — нет name", idx)
		out.skipped = true
		out.message = "нет name"
		return out
	}
	out.key = name

	statusLabel := strings.TrimSpace(row.Status.Label)
	mapped, known := MapCUStatusLabel(statusLabel)
	if !known {
		log.Printf("itilliumimport: getCU enclosure %s: неизвестный статус %q", name, statusLabel)
		out.unknownSt = true
	}

	tech := cuTechFromRow(row)
	sn := cuTrim(row.SN)
	model := cuTrim(row.ModelBucke)
	where := cuTrim(row.Where)
	links := cuLinksUnit(row)
	owner := cuTrim(row.User)
	inv := cuTrim(row.Name)
	buhInv := cuTrim(row.BuhCode)
	if buhInv == "" {
		buhInv = cuTrim(row.BuhInv)
	}
	contractDoc := cuTrim(row.Docs)
	provider := cuTrim(row.Provider)
	docDate, hasDocDate := cuParseDateRU(cuTrim(row.DataDocs))
	descOnly := cuTrim(row.Description)
	statusHist, hasStatusHist := cuStatusHistoryTime(row)
	sizeSlots := cuIntField(row.BayCount)

	itiliumSt := 0
	if known {
		itiliumSt = mapped.ItiliumStatus
	}

	flNotWorking := cuWhereScrap(where) || locationUndefinedForCU(where, itiliumSt)
	if locationUndefinedForCU(where, itiliumSt) {
		out.noLoc = true
	}

	var bs models.BladeSystems
	bs.Name = name
	bs.TechnicalNumberIT = tech
	bs.SerialNum = sn
	bs.Model = model
	bs.Size = sizeSlots
	bs.ItiliumStatus = itiliumSt
	bs.PowerState = 1
	bs.Active = 1
	bs.Owner = owner
	bs.InventoryNum = inv
	bs.InventoryNumBuh = buhInv
	bs.ContractDoc = contractDoc
	bs.Counterparty = provider
	bs.Description = descOnly
	bs.Anotation = where
	if hasDocDate {
		bs.DateContractDoc = docDate
	}
	if hasStatusHist {
		bs.StatusHistory = statusHist
	}

	trn, err := o.Begin()
	if err != nil {
		log.Printf("itilliumimport: getCU enclosure %s: Begin: %v", name, err)
		out.err = true
		return out
	}

	rackId := int64(0)
	cabinetCode := ""
	if !flNotWorking && where != "" {
		site, errSite := resolveSite(trn, rs, where, models.UserSourceManual)
		if errSite != nil {
			log.Printf("itilliumimport: getCU enclosure %s: resolveSite: %v", name, errSite)
			trn.Rollback()
			out.err = true
			return out
		}
		if site.Loc != nil && site.Loc.Service {
			out.noLoc = true
			flNotWorking = true
		} else {
			pl := locality.ParsePlacement(links)
			rackNum := ""
			if pl != nil {
				rackNum = pl.Cabinet
			}
			var locPos *locality.Position
			var cabinetAlt string
			cabinetCode, cabinetAlt, locPos = cabinetFromPlacement(rackNum, site.Loc)
			if cabinetCode != "" {
				var errRack error
				rackId, errRack = resolveRackCabinet(trn, cabinetCode, cabinetAlt, site.Id,
					"Шкаф по данным getCU корзина, площадка: "+site.Name)
				if errRack != nil {
					log.Printf("itilliumimport: getCU enclosure %s: resolveRack: %v", name, errRack)
					trn.Rollback()
					out.err = true
					return out
				}
				if locPos != nil && bs.StartUnitPos == 0 {
					bs.StartUnitPos, bs.EndUnitPos = locPos.Start, locPos.End
				}
			} else if site.Id > 0 {
				rackId, err = resolveOutsideCabinet(trn, site.Id)
				if err != nil {
					trn.Rollback()
					out.err = true
					return out
				}
				bs.StartUnitPos, bs.EndUnitPos, bs.SizeUnit = 0, 0, 0
			}
			if pl != nil && pl.Pos != nil && bs.StartUnitPos == 0 {
				bs.StartUnitPos = pl.Pos.Start
				bs.EndUnitPos = pl.Pos.End
			}
		}
	}
	bs.RackCabinetId = rackId
	if bs.EndUnitPos >= bs.StartUnitPos && bs.StartUnitPos > 0 {
		bs.SizeUnit = bs.EndUnitPos - bs.StartUnitPos + 1
	}
	if bs.SizeUnit == 0 {
		if u := cuIntField(row.Units); u > 0 {
			bs.SizeUnit = u
		}
	}
	if cabinetCode == "" || rackId == 0 || flNotWorking {
		bs.PowerState = 2
		bs.Active = 2
		out.inactive = true
	}

	exist := models.BladeSystems{Name: name}
	bsFound := trn.Read(&exist, "Name") == nil
	var bsId int64

	if !bsFound {
		bs.Id_User = models.UserSourceManual
		bs.Time_Mfy = time.Now().Local()
		if _, err := trn.Insert(&bs); err != nil {
			log.Printf("itilliumimport: getCU enclosure %s: insert: %v", name, err)
			trn.Rollback()
			out.err = true
			return out
		}
		bsId = bs.Id
		out.inserted = true
	} else {
		exist.RackCabinetId = bs.RackCabinetId
		exist.PowerState = bs.PowerState
		exist.Active = bs.Active
		exist.ItiliumStatus = bs.ItiliumStatus
		exist.StartUnitPos = bs.StartUnitPos
		exist.EndUnitPos = bs.EndUnitPos
		exist.SizeUnit = bs.SizeUnit
		exist.Anotation = bs.Anotation
		exist.InventoryNum = bs.InventoryNum
		exist.InventoryNumBuh = bs.InventoryNumBuh
		fillEmpty(&exist.TechnicalNumberIT, tech)
		fillEmpty(&exist.SerialNum, sn)
		fillEmpty(&exist.Model, model)
		if exist.Size == 0 && sizeSlots > 0 {
			exist.Size = sizeSlots
		}
		if descOnly != "" {
			exist.Description = descOnly
		}
		if hasStatusHist {
			exist.StatusHistory = statusHist
		}
		fillEmpty(&exist.ContractDoc, contractDoc)
		fillEmpty(&exist.Counterparty, provider)
		fillEmpty(&exist.Owner, owner)
		if exist.DateContractDoc.IsZero() && hasDocDate {
			exist.DateContractDoc = docDate
		}
		exist.Id_User = models.UserSourceManual
		exist.Time_Mfy = time.Now().Local()
		cols := []string{"RackCabinetId", "PowerState", "Active", "ItiliumStatus", "Anotation",
			"StartUnitPos", "EndUnitPos", "SizeUnit", "InventoryNum", "InventoryNumBuh",
			"TechnicalNumberIT", "SerialNum", "Model", "Size", "Description", "StatusHistory",
			"ContractDoc", "Counterparty", "Owner", "Id_User", "Time_Mfy"}
		if hasDocDate && !exist.DateContractDoc.IsZero() {
			cols = append(cols, "DateContractDoc")
		}
		if _, err := trn.Update(&exist, cols...); err != nil {
			log.Printf("itilliumimport: getCU enclosure %s: update: %v", name, err)
			trn.Rollback()
			out.err = true
			return out
		}
		bsId = exist.Id
	}

	if err := trn.Commit(); err != nil {
		log.Printf("itilliumimport: getCU enclosure %s: commit: %v", name, err)
		out.err = true
		return out
	}

	if refreshedActive, ok := refreshBladeSystemFromOneView("getCU", bsId, sn); ok && refreshedActive == 2 {
		out.inactive = true
	}
	return out
}