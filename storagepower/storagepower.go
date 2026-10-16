// Package storagepower собирает потребляемую мощность СХД по схеме пакета
// zabbixpower (серверы): поминутные замеры storage_power_metric (хранятся
// 3 суток), суточная агрегация storage_power_record в 00:00 и запись
// суточных итогов в StorageSystems.PowerAvg24hW/PowerPeak24hW.
//
// Источники: Unity - сумма currentPower DPE+DAE; Huawei Dorado/OceanStor -
// get_realtime_power_consumption. У Lenovo и VNX API мощности нет.
package storagepower

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/lovi-cloud/go-dorado-sdk/dorado"

	"loadmon/models"
	"loadmon/storages"
	"loadmon/utilities"
)

var (
	// Interval — период сбора.
	Interval = time.Minute
	// Retention — срок хранения поминутных замеров.
	Retention = 3 * 24 * time.Hour
	// Workers — сколько СХД опрашивается параллельно.
	Workers = 4
)

const (
	apiUnity       = 2
	apiDorado      = 3
	apiOceanStor   = 4
	huaweiLoginKey = "spaview"
)

// watts опрашивает одну СХД.
func watts(ctx context.Context, shd *models.StorageSystems) (float64, error) {
	switch shd.ApiType {
	case apiUnity:
		return storages.GetUnityPowerWatts(shd.IpAddress)
	case apiDorado, apiOceanStor:
		url := fmt.Sprintf("https://%s:8088", shd.IpAddress)
		client, err := dorado.NewClient([]string{url}, []string{url}, huaweiLoginKey, utilities.GetKeyData(huaweiLoginKey), "", nil)
		if err != nil {
			return 0, fmt.Errorf("подключение: %w", err)
		}
		return client.LocalDevice.GetRealtimePowerConsumption(ctx)
	}
	return 0, fmt.Errorf("ApiType=%d: мощность недоступна", shd.ApiType)
}

func safeWatts(ctx context.Context, shd *models.StorageSystems) (w float64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("паника: %v", r)
		}
	}()
	return watts(ctx, shd)
}

// Collect снимает мощность всех активных СХД (Power=1) за текущую минуту.
func Collect(ctx context.Context) error {
	now := time.Now().Local().Truncate(time.Minute)
	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Power", 1).
		Filter("ApiType__in", apiUnity, apiDorado, apiOceanStor).All(&systems); err != nil {
		return fmt.Errorf("storagepower: read storage_systems: %w", err)
	}

	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		rows []*models.StoragePowerMetric
		jobs = make(chan *models.StorageSystems)
	)
	for i := 0; i < Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for shd := range jobs {
				w, err := safeWatts(ctx, shd)
				if err != nil {
					log.Printf("storagepower: %s (%s): %v", shd.ShortName, shd.IpAddress, err)
					continue
				}
				mu.Lock()
				rows = append(rows, &models.StoragePowerMetric{StorageSystemsId: shd.Id, CollectedAt: now, Watts: w})
				mu.Unlock()
			}
		}()
	}
	for _, shd := range systems {
		if shd != nil && shd.Id > 0 {
			jobs <- shd
		}
	}
	close(jobs)
	wg.Wait()

	if err := models.ReplaceStoragePower(ctx, rows); err != nil {
		return fmt.Errorf("storagepower: store: %w", err)
	}
	initPower(o, systems, rows)
	log.Printf("storagepower: stored %d of %d at %s", len(rows), len(systems), now.Format("15:04"))
	return nil
}

// initPower: при первом замере (оба поля 0) пишет текущую мощность в
// PowerAvg24hW и PowerPeak24hW; дальше их ведёт суточная агрегация.
func initPower(o orm.Ormer, systems []*models.StorageSystems, rows []*models.StoragePowerMetric) {
	cur := make(map[int64]float64, len(rows))
	for _, r := range rows {
		cur[r.StorageSystemsId] = r.Watts
	}
	for _, s := range systems {
		w, ok := cur[s.Id]
		if !ok || w <= 0 || s.PowerAvg24hW != 0 || s.PowerPeak24hW != 0 {
			continue
		}
		if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Id", s.Id).
			Update(orm.Params{"PowerAvg24hW": w, "PowerPeak24hW": w}); err != nil {
			log.Printf("storagepower: init power storage=%d: %v", s.Id, err)
		}
	}
}

// RunAll запускает минутный сбор и суточное обслуживание до отмены ctx.
func RunAll(ctx context.Context) {
	go maintenance(ctx)
	if Interval <= 0 {
		Interval = time.Minute
	}
	var running sync.Mutex // цикл опроса может занять больше минуты - пересекающиеся запуски пропускаем
	tick := func() {
		if !running.TryLock() {
			return
		}
		defer running.Unlock()
		if err := Collect(ctx); err != nil {
			log.Printf("storagepower: collect: %v", err)
		}
	}
	tick()
	t := time.NewTicker(Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			go tick()
		}
	}
}

// maintenance: при старте догоняет агрегацию за дни хранения, затем каждые
// сутки в 00:00:05 агрегирует прошедший день и удаляет замеры старше Retention.
func maintenance(ctx context.Context) {
	run := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("storagepower: maintenance panic: %v", r)
			}
		}()
		today := dayStart(time.Now().Local())
		for d := int(Retention / (24 * time.Hour)); d >= 1; d-- {
			AggregateDay(today.AddDate(0, 0, -d), d == 1)
		}
		ClearOld(today.Add(-Retention))
	}
	run()
	for {
		now := time.Now().Local()
		next := dayStart(now).AddDate(0, 0, 1).Add(5 * time.Second)
		select {
		case <-ctx.Done():
			return
		case <-time.After(next.Sub(now)):
			run()
		}
	}
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// ClearOld удаляет минутные замеры старше before порциями.
func ClearOld(before time.Time) {
	o := orm.NewOrm()
	total := int64(0)
	for {
		res, err := o.Raw("DELETE FROM storage_power_metric WHERE id IN (SELECT id FROM storage_power_metric WHERE collected_at < ? LIMIT 1000)", before).Exec()
		if err != nil {
			log.Printf("storagepower: clear: %v", err)
			return
		}
		n, _ := res.RowsAffected()
		total += n
		if n == 0 {
			break
		}
	}
	log.Printf("storagepower: deleted %d rows older than %s", total, before.Format("2006-01-02"))
}

// AggregateDay строит (идемпотентно) суточные записи за сутки day; при setHW
// пишет среднее и максимум в StorageSystems.PowerAvg24hW/PowerPeak24hW.
func AggregateDay(day time.Time, setHW bool) {
	day = dayStart(day)
	end := day.AddDate(0, 0, 1)
	o := orm.NewOrm()

	var ids []orm.ParamsList
	if _, err := o.Raw("SELECT DISTINCT storage_systems_id FROM storage_power_metric WHERE collected_at >= ? AND collected_at < ?", day, end).ValuesList(&ids); err != nil {
		log.Printf("storagepower: aggregate %s: %v", day.Format("2006-01-02"), err)
		return
	}
	count := 0
	for _, row := range ids {
		var id int64
		switch v := row[0].(type) {
		case int64:
			id = v
		case string:
			fmt.Sscan(v, &id)
		}
		if id <= 0 {
			continue
		}
		var samples []*models.StoragePowerMetric
		if _, err := o.QueryTable(new(models.StoragePowerMetric)).Filter("StorageSystemsId", id).
			Filter("CollectedAt__gte", day).Filter("CollectedAt__lt", end).OrderBy("CollectedAt").All(&samples); err != nil || len(samples) == 0 {
			continue
		}
		rec := aggregate(id, day, samples)
		existing := models.StoragePowerRecord{StorageSystemsId: id, Date: day}
		if err := o.Read(&existing, "StorageSystemsId", "Date"); err == nil {
			rec.Id = existing.Id
			if _, err := o.Update(rec); err != nil {
				log.Printf("storagepower: update record storage=%d: %v", id, err)
			}
		} else if _, err := o.Insert(rec); err != nil {
			log.Printf("storagepower: insert record storage=%d: %v", id, err)
		}
		if setHW {
			if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Id", id).
				Update(orm.Params{"PowerAvg24hW": rec.WattsAvg, "PowerPeak24hW": rec.WattsMax}); err != nil {
				log.Printf("storagepower: set power storage=%d: %v", id, err)
			}
		}
		count++
	}
	log.Printf("storagepower: aggregated %d storages for %s", count, day.Format("2006-01-02"))
}

func aggregate(id int64, day time.Time, samples []*models.StoragePowerMetric) *models.StoragePowerRecord {
	rec := &models.StoragePowerRecord{StorageSystemsId: id, Date: day, PerfCount: int64(len(samples)), Time_Mfy: time.Now().Local()}
	max, min, sum := -math.MaxFloat64, math.MaxFloat64, 0.0
	for _, s := range samples {
		sum += s.Watts
		if s.Watts > max {
			max, rec.WattsMaxTime = s.Watts, s.CollectedAt
		}
		if s.Watts < min {
			min, rec.WattsMinTime = s.Watts, s.CollectedAt
		}
	}
	rec.WattsMax, rec.WattsMin = max, min
	rec.WattsAvg = sum / float64(len(samples))
	rec.EnergyKWh = sum / 60 / 1000
	return rec
}
