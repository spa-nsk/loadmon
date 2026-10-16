package rackpower

import (
	"context"
	"log"
	"time"

	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// SaveDaily пишет (идемпотентно) суточные записи мощности всех шкафов и
// площадок на дату day (rack_power_record, site_power_record), используя
// те же расчёты, что и экраны (RackCabinetPowerTotalsBatch /
// DatacenterSitePowerTotalsBatch): rack-серверы + корзины + СХД. Вызывать
// ПОСЛЕ ночной агрегации серверов и СХД (zabbixpower, storagepower), когда
// в PowerAvg24hW/PowerPeak24hW уже суточные итоги. Пик шкафа/площадки -
// сумма суточных пиков устройств (верхняя оценка: пики разнесены во времени).
func SaveDaily(day time.Time) {
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	o := orm.NewOrm()
	now := time.Now().Local()

	var racks []models.RackCabinet
	if _, err := o.QueryTable(new(models.RackCabinet)).All(&racks, "Id", "DatacenterSiteId"); err != nil {
		log.Printf("rackpower: read racks: %v", err)
		return
	}
	rackIds := make([]int64, 0, len(racks))
	for _, r := range racks {
		rackIds = append(rackIds, r.Id)
	}
	rackPower := RackCabinetPowerTotalsBatch(o, rackIds)
	nr := 0
	for id, p := range rackPower {
		if p.Avg24hW == 0 && p.Peak24hW == 0 {
			continue
		}
		rec := models.RackPowerRecord{RackCabinetId: id, Date: day}
		if err := o.Read(&rec, "RackCabinetId", "Date"); err != nil {
			rec = models.RackPowerRecord{RackCabinetId: id, Date: day}
		}
		rec.AvgW, rec.PeakW, rec.RatedMaxW, rec.Time_Mfy = p.Avg24hW, p.Peak24hW, p.RatedMaxW, now
		if err := upsert(o, &rec, rec.Id); err != nil {
			log.Printf("rackpower: save rack=%d: %v", id, err)
			continue
		}
		nr++
	}

	var sites []models.DatacenterSite
	if _, err := o.QueryTable(new(models.DatacenterSite)).All(&sites, "Id"); err != nil {
		log.Printf("rackpower: read sites: %v", err)
		return
	}
	siteIds := make([]int64, 0, len(sites))
	for _, s := range sites {
		siteIds = append(siteIds, s.Id)
	}
	sitePower := DatacenterSitePowerTotalsBatch(o, siteIds)
	ns := 0
	for id, p := range sitePower {
		if p.Avg24hW == 0 && p.Peak24hW == 0 {
			continue
		}
		rec := models.SitePowerRecord{DatacenterSiteId: id, Date: day}
		if err := o.Read(&rec, "DatacenterSiteId", "Date"); err != nil {
			rec = models.SitePowerRecord{DatacenterSiteId: id, Date: day}
		}
		rec.AvgW, rec.PeakW, rec.RatedMaxW, rec.Time_Mfy = p.Avg24hW, p.Peak24hW, p.RatedMaxW, now
		if err := upsert(o, &rec, rec.Id); err != nil {
			log.Printf("rackpower: save site=%d: %v", id, err)
			continue
		}
		ns++
	}
	log.Printf("rackpower: daily %s: racks %d, sites %d", day.Format("2006-01-02"), nr, ns)
}

func upsert(o orm.Ormer, rec interface{}, id int64) error {
	if id > 0 {
		_, err := o.Update(rec)
		return err
	}
	_, err := o.Insert(rec)
	return err
}

// RunDaily каждый день в 00:10 (после ночной агрегации серверов и СХД в
// 00:00:05) записывает суточные итоги шкафов и площадок за прошедшие сутки.
func RunDaily(ctx context.Context) {
	for {
		now := time.Now().Local()
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 10, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(next.Sub(now)):
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("rackpower: daily panic: %v", r)
					}
				}()
				SaveDaily(time.Now().Local().AddDate(0, 0, -1))
			}()
		}
	}
}
