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

// ImportStorageFromCU — cuClass=158 → StorageSystems + StorageSystemPlacement.
func ImportStorageFromCU(rows []itilium.CURow) ImportCUStats {
	var st ImportCUStats
	st.Total = len(rows)
	o := orm.NewOrm()
	rs := locality.NewResolver()
	for i, row := range rows {
		aggregateCUImportOne(&st, importOneCUStorage(o, rs, row, i))
	}
	log.Printf("itilliumimport: ImportStorageFromCU: total=%d inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Total, st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}

func importOneCUStorage(o orm.Ormer, rs *locality.Resolver, row itilium.CURow, idx int) cuImportOne {
	var out cuImportOne
	sn := cuTrim(row.SN)
	if sn == "" {
		log.Printf("itilliumimport: getCU storage[%d]: пропуск — нет sn", idx)
		out.skipped = true
		out.message = "нет sn"
		return out
	}
	out.key = sn

	statusLabel := strings.TrimSpace(row.Status.Label)
	mapped, known := MapCUStatusLabel(statusLabel)
	if !known {
		log.Printf("itilliumimport: getCU storage %s: неизвестный статус %q", sn, statusLabel)
		out.unknownSt = true
	}

	tech := cuTechFromRow(row)
	name := cuTrim(row.Name)
	desc := cuTrim(row.Description)
	where := cuTrim(row.Where)
	links := cuLinksUnit(row)
	owner := cuTrim(row.User)
	provider := cuTrim(row.Provider)
	contractDoc := cuTrim(row.Docs)
	buhInv := cuTrim(row.BuhCode)
	if buhInv == "" {
		buhInv = cuTrim(row.BuhInv)
	}
	docDate, hasDocDate := cuParseDateRU(cuTrim(row.DataDocs))
	statusHist, hasStatusHist := cuStatusHistoryTime(row)

	itiliumSt := 0
	if known {
		itiliumSt = mapped.ItiliumStatus
	}
	flNotWorking := cuWhereScrap(where) || locationUndefinedForCU(where, itiliumSt)
	if locationUndefinedForCU(where, itiliumSt) {
		out.noLoc = true
	}
	if flNotWorking {
		out.inactive = true
	}

	shortName := firstNonEmptyStr(name, desc, tech)
	model := cuTrim(row.ModelSHD)

	statusLabelTrim := strings.TrimSpace(row.Status.Label)
	anotation := statusLabelTrim
	if links != "" {
		if anotation != "" {
			anotation += "; " + links
		} else {
			anotation = links
		}
	}

	trn, err := o.Begin()
	if err != nil {
		log.Printf("itilliumimport: getCU storage %s: Begin: %v", sn, err)
		out.err = true
		return out
	}

	ownSiteId := int64(0)
	ownSiteName := ""
	var placements []struct {
		rackId int64
		startU int
		endU   int
	}
	if !flNotWorking && where != "" {
		site, errSite := resolveSite(trn, rs, where, models.UserSourceManual)
		if errSite != nil {
			log.Printf("itilliumimport: getCU storage %s: resolveSite: %v", sn, errSite)
			trn.Rollback()
			out.err = true
			return out
		}
		if site.Loc != nil && site.Loc.Service {
			out.noLoc = true
			flNotWorking = true
		} else {
			ownSiteId = site.Id
			ownSiteName = site.Name
			segments := parseStoragePlacementSegments(links)
			if len(segments) == 0 && links != "" {
				pl := locality.ParsePlacement(links)
				if pl != nil && pl.Cabinet != "" {
					seg := storagePlacementSegment{CabinetCode: pl.Cabinet}
					if pl.Pos != nil {
						seg.StartUnitPos, seg.EndUnitPos = pl.Pos.Start, pl.Pos.End
					}
					segments = append(segments, seg)
				}
			}
			if len(segments) == 0 {
				cabinetCode, _, locPos := cabinetFromPlacement("", site.Loc)
				if cabinetCode != "" {
					startU, endU := 0, 0
					if locPos != nil {
						startU, endU = locPos.Start, locPos.End
					}
					rackId, errRack := resolveRackCabinet(trn, cabinetCode, "", ownSiteId,
						"Шкаф по данным getCU СХД, площадка: "+ownSiteName)
					if errRack != nil {
						log.Printf("itilliumimport: getCU storage %s: resolveRack: %v", sn, errRack)
					} else {
						placements = append(placements, struct {
							rackId int64
							startU int
							endU   int
						}{rackId: rackId, startU: startU, endU: endU})
					}
				}
			}
			for _, seg := range segments {
				if ownSiteId <= 0 {
					continue
				}
				rackId, errRack := resolveRackCabinet(trn, seg.CabinetCode, seg.AltName, ownSiteId,
					"Шкаф по данным getCU СХД, площадка: "+ownSiteName)
				if errRack != nil {
					log.Printf("itilliumimport: getCU storage %s: segment %q: %v", sn, seg.CabinetCode, errRack)
					continue
				}
				placements = append(placements, struct {
					rackId int64
					startU int
					endU   int
				}{rackId: rackId, startU: seg.StartUnitPos, endU: seg.EndUnitPos})
			}
		}
	}

	st := models.StorageSystems{SerialNum: sn}
	stFound := trn.Read(&st, "SerialNum") == nil

	if !stFound {
		st.SerialNum = sn
		st.ShortName = shortName
		st.Model = model
		st.TechnicalNumberIT = tech
		st.InventoryNum = name
		st.InventoryNumBuh = buhInv
		st.Owner = owner
		st.Counterparty = provider
		st.ContractDoc = contractDoc
		st.ItiliumStatus = itiliumSt
		st.Anotation = anotation
		st.Description = desc
		if hasStatusHist {
			st.StatusHistory = statusHist
		}
		if hasDocDate {
			st.DateContractDoc = docDate
		}
		st.Id_User = models.UserSourceManual
		st.Time_Mfy = time.Now().Local()
		st.SourceUpdateType = models.StorageSourceUpdateManual
		if _, err := trn.Insert(&st); err != nil {
			log.Printf("itilliumimport: getCU storage %s: insert: %v", sn, err)
			trn.Rollback()
			out.err = true
			return out
		}
		out.inserted = true
	} else {
		st.InventoryNum = name
		st.InventoryNumBuh = buhInv
		st.ItiliumStatus = itiliumSt
		st.Anotation = anotation
		fillEmpty(&st.TechnicalNumberIT, tech)
		fillEmpty(&st.ShortName, shortName)
		fillEmpty(&st.Model, model)
		fillEmpty(&st.Owner, owner)
		fillEmpty(&st.Counterparty, provider)
		fillEmpty(&st.ContractDoc, contractDoc)
		if desc != "" {
			st.Description = desc
		}
		if hasStatusHist {
			st.StatusHistory = statusHist
		}
		if st.DateContractDoc.IsZero() && hasDocDate {
			st.DateContractDoc = docDate
		}
		st.Id_User = models.UserSourceManual
		st.Time_Mfy = time.Now().Local()
		cols := []string{"InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "ItiliumStatus", "Anotation",
			"ShortName", "Model", "Owner", "Counterparty", "ContractDoc", "Description", "StatusHistory",
			"DateContractDoc", "Id_User", "Time_Mfy"}
		if _, err := trn.Update(&st, cols...); err != nil {
			log.Printf("itilliumimport: getCU storage %s: update: %v", sn, err)
			trn.Rollback()
			out.err = true
			return out
		}
	}

	placementErr := false
	if !flNotWorking {
		if _, err := trn.QueryTable(new(models.StorageSystemPlacement)).
			Filter("StorageSystemsId", st.Id).Delete(); err != nil {
			log.Printf("itilliumimport: getCU storage %s: clear placements: %v", sn, err)
			trn.Rollback()
			out.err = true
			return out
		}
		for _, p := range placements {
			if p.rackId <= 0 {
				continue
			}
			if err := upsertStorageSystemPlacement(trn, st.Id, p.rackId, p.startU, p.endU); err != nil {
				log.Printf("itilliumimport: getCU storage %s: placement: %v", sn, err)
				placementErr = true
				break
			}
		}
		if !placementErr && len(placements) == 0 && ownSiteId > 0 {
			outsideId, errOut := resolveOutsideCabinet(trn, ownSiteId)
			if errOut != nil {
				placementErr = true
			} else if err := upsertStorageSystemPlacement(trn, st.Id, outsideId, 0, 0); err != nil {
				placementErr = true
			}
		}
	}
	if placementErr {
		trn.Rollback()
		out.err = true
		return out
	}

	if err := trn.Commit(); err != nil {
		log.Printf("itilliumimport: getCU storage %s: commit: %v", sn, err)
		out.err = true
		return out
	}
	return out
}