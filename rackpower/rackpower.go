// Package rackpower — агрегация мощности шкафа/ЦОД (вариант C, без double-count блейдов).
package rackpower

import (
	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// RackCabinetPowerVariantC — rack ServerHardware (FormFactor=1) + целые BladeSystems.
func RackCabinetPowerVariantC(o orm.Ormer, rackCabinetId int64) oneview.PowerWatts {
	if o == nil || rackCabinetId <= 0 {
		return oneview.PowerWatts{}
	}
	m := RackCabinetPowerTotalsBatch(o, []int64{rackCabinetId})
	return m[rackCabinetId]
}

// RackCabinetPowerTotalsBatch считает мощность для набора шкафов.
func RackCabinetPowerTotalsBatch(o orm.Ormer, rackCabinetIds []int64) map[int64]oneview.PowerWatts {
	out := make(map[int64]oneview.PowerWatts, len(rackCabinetIds))
	for _, id := range rackCabinetIds {
		if id > 0 {
			out[id] = oneview.PowerWatts{}
		}
	}
	if len(out) == 0 {
		return out
	}
	ids := make([]int64, 0, len(out))
	for id := range out {
		ids = append(ids, id)
	}

	var servers []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).
		Filter("RackCabinetId__in", ids).
		Filter("FormFactor", 1).
		All(&servers); err == nil {
		for _, h := range servers {
			p := out[h.RackCabinetId]
			p.Avg24hW += h.PowerAvg24hW
			p.Peak24hW += h.PowerPeak24hW
			p.RatedMaxW += h.PowerRatedMaxW
			out[h.RackCabinetId] = p
		}
	}

	var chassis []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).
		Filter("RackCabinetId__in", ids).
		All(&chassis); err == nil {
		for _, b := range chassis {
			p := out[b.RackCabinetId]
			p.Avg24hW += b.PowerAvg24hW
			p.Peak24hW += b.PowerPeak24hW
			p.RatedMaxW += b.PowerRatedMaxW
			out[b.RackCabinetId] = p
		}
	}
	return out
}

// DatacenterSitePowerTotalsBatch — сумма по шкафам площадки.
func DatacenterSitePowerTotalsBatch(o orm.Ormer, siteIds []int64) map[int64]oneview.PowerWatts {
	out := make(map[int64]oneview.PowerWatts, len(siteIds))
	for _, id := range siteIds {
		if id > 0 {
			out[id] = oneview.PowerWatts{}
		}
	}
	if len(out) == 0 {
		return out
	}
	var racks []models.RackCabinet
	if _, err := o.QueryTable(new(models.RackCabinet)).
		Filter("DatacenterSiteId__in", siteIds).
		All(&racks); err != nil {
		return out
	}
	rackIds := make([]int64, 0, len(racks))
	rackToSite := make(map[int64]int64, len(racks))
	for _, r := range racks {
		if r.Id <= 0 {
			continue
		}
		rackIds = append(rackIds, r.Id)
		rackToSite[r.Id] = r.DatacenterSiteId
	}
	rackPower := RackCabinetPowerTotalsBatch(o, rackIds)
	for rackId, p := range rackPower {
		siteId := rackToSite[rackId]
		if siteId <= 0 {
			continue
		}
		s := out[siteId]
		s.Avg24hW += p.Avg24hW
		s.Peak24hW += p.Peak24hW
		s.RatedMaxW += p.RatedMaxW
		out[siteId] = s
	}
	return out
}
