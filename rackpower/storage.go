package rackpower

import (
	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// storageShare — доля мощности СХД, приходящаяся на одно размещение.
type storageShare struct {
	rackId, siteId int64
	avg, peak      float64
}

// storagePlacementPower возвращает мощность СХД по размещениям, выбранным
// условием cond. СХД может стоять в нескольких шкафах (StorageSystemPlacement),
// поэтому её мощность делится между всеми размещениями пропорционально
// SizeUnit (если юниты не заполнены - поровну). Мощность СХД -
// StorageSystems.PowerAvg24hW/PowerPeak24hW (пакет storagepower).
func storagePlacementPower(o orm.Ormer, cond *orm.Condition) []storageShare {
	var selected []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).SetCond(cond).All(&selected); err != nil || len(selected) == 0 {
		return nil
	}
	sysIds := make([]int64, 0, len(selected))
	seen := map[int64]bool{}
	for _, p := range selected {
		if !seen[p.StorageSystemsId] {
			seen[p.StorageSystemsId] = true
			sysIds = append(sysIds, p.StorageSystemsId)
		}
	}

	var systems []models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Id__in", sysIds).All(&systems); err != nil {
		return nil
	}
	power := make(map[int64]*models.StorageSystems, len(systems))
	for i := range systems {
		power[systems[i].Id] = &systems[i]
	}

	// вес размещений считаем по ВСЕМ размещениям СХД, а не только выбранным
	var all []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId__in", sysIds).All(&all); err != nil {
		return nil
	}
	total := map[int64]float64{}
	count := map[int64]float64{}
	for _, p := range all {
		total[p.StorageSystemsId] += float64(p.SizeUnit)
		count[p.StorageSystemsId]++
	}
	weight := func(p models.StorageSystemPlacement) float64 {
		if total[p.StorageSystemsId] > 0 {
			return float64(p.SizeUnit) / total[p.StorageSystemsId]
		}
		return 1 / count[p.StorageSystemsId]
	}

	out := make([]storageShare, 0, len(selected))
	for _, p := range selected {
		s := power[p.StorageSystemsId]
		if s == nil {
			continue
		}
		w := weight(p)
		out = append(out, storageShare{
			rackId: p.RackCabinetId,
			siteId: p.DatacenterSiteId,
			avg:    s.PowerAvg24hW * w,
			peak:   s.PowerPeak24hW * w,
		})
	}
	return out
}
