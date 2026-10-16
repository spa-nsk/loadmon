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

// SaveBladeSnapshot пишет (идемпотентно) на дату day снимок мощности всех
// блейд-корзин: BladeSystems.PowerAvg24hW/PowerPeak24hW (OneView, среднее и
// пик за 24 часа). Поминутных данных по корзинам нет, это единственная
// история для графиков корзины.
func SaveBladeSnapshot(day time.Time) {
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	o := orm.NewOrm()
	var list []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).All(&list, "Id", "PowerAvg24hW", "PowerPeak24hW"); err != nil {
		log.Printf("rackpower: read blade systems: %v", err)
		return
	}
	now := time.Now().Local()
	n := 0
	for _, b := range list {
		if b.PowerAvg24hW <= 0 && b.PowerPeak24hW <= 0 {
			continue
		}
		rec := models.BladePowerRecord{BladeSystemId: b.Id, Date: day}
		if err := o.Read(&rec, "BladeSystemId", "Date"); err != nil {
			rec = models.BladePowerRecord{BladeSystemId: b.Id, Date: day}
		}
		rec.AvgW, rec.PeakW, rec.Time_Mfy = b.PowerAvg24hW, b.PowerPeak24hW, now
		if err := upsert(o, &rec, rec.Id); err != nil {
			log.Printf("rackpower: save blade=%d: %v", b.Id, err)
			continue
		}
		n++
	}
	log.Printf("rackpower: blade snapshot %s: %d", day.Format("2006-01-02"), n)
}

func upsert(o orm.Ormer, rec interface{}, id int64) error {
	if id > 0 {
		_, err := o.Update(rec)
		return err
	}
	_, err := o.Insert(rec)
	return err
}

// RunDaily: снимки мощности корзин - при старте и раз в час (значения
// OneView обновляются при инвентаризации), а каждый день в 00:10 (после
// ночной агрегации серверов и СХД в 00:00:05) - итоги шкафов и площадок и
// финальный снимок корзин за прошедшие сутки.
func RunDaily(ctx context.Context) {
	safe := func(f func()) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("rackpower: daily panic: %v", r)
			}
		}()
		f()
	}
	safe(func() { SaveBladeSnapshot(time.Now().Local()) })
	hourly := time.NewTicker(time.Hour)
	defer hourly.Stop()
	for {
		now := time.Now().Local()
		next := time.Date(now.Year(), now.Month(), now.Day(), 0, 10, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		select {
		case <-ctx.Done():
			return
		case <-hourly.C:
			safe(func() { SaveBladeSnapshot(time.Now().Local()) })
		case <-time.After(next.Sub(now)):
			safe(func() {
				y := time.Now().Local().AddDate(0, 0, -1)
				SaveBladeSnapshot(y)
				SaveDaily(y)
			})
		}
	}
}
