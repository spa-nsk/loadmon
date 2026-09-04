package cmdb

import (
	"fmt"
	"log"
	"math"
	"time"

	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// SaveStorageMetricRecord агрегирует поминутные метрики СХД (storage_metric)
// в суточные записи storage_metric_record с max/min/avg и временем extrema.
func SaveStorageMetricRecord(currentTime time.Time) {
	dayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
	dayEnd := currentTime

	o := orm.NewOrm()
	var rows []*models.StorageMetric
	qs := o.QueryTable(new(models.StorageMetric))
	qs = qs.Filter("CollectedAt__gte", dayStart).Filter("CollectedAt__lt", dayEnd)
	qs = qs.OrderBy("StorageSystemsId", "CollectedAt")
	num, err := qs.All(&rows)
	if err != nil {
		fmt.Println("cmdb.SaveStorageMetricRecord: ошибка чтения storage_metric:", err)
		return
	}
	if num == 0 {
		return
	}

	byStorage := make(map[int64][]*models.StorageMetric)
	storageOrder := make([]int64, 0)
	for _, r := range rows {
		if r == nil {
			continue
		}
		if _, ok := byStorage[r.StorageSystemsId]; !ok {
			storageOrder = append(storageOrder, r.StorageSystemsId)
		}
		byStorage[r.StorageSystemsId] = append(byStorage[r.StorageSystemsId], r)
	}

	for _, storageID := range storageOrder {
		samples := byStorage[storageID]
		rec := aggregateStorageMetricRecord(storageID, samples)
		rec.Date = dayStart
		rec.Time_Mfy = time.Now().Local()

		if len(samples) > 0 && samples[0] != nil {
			rec.ShortName = samples[0].ShortName
		}
		shd := models.StorageSystems{Id: storageID}
		if err := o.Read(&shd); err == nil && rec.ShortName == "" {
			rec.ShortName = shd.ShortName
		}

		existing := models.StorageMetricRecord{}
		existing.StorageSystemsId = storageID
		existing.Date = dayStart
		if err := o.Read(&existing, "StorageSystemsId", "Date"); err == nil {
			rec.Id = existing.Id
			if _, err := o.Update(rec); err != nil {
				log.Println("cmdb.SaveStorageMetricRecord: ошибка обновления storage_metric_record StorageSystemsId=", storageID, err)
			}
		} else {
			if _, err := o.Insert(rec); err != nil {
				log.Println("cmdb.SaveStorageMetricRecord: ошибка вставки storage_metric_record StorageSystemsId=", storageID, err)
			}
		}
	}
}

func aggregateStorageMetricRecord(storageID int64, samples []*models.StorageMetric) *models.StorageMetricRecord {
	rec := &models.StorageMetricRecord{StorageSystemsId: storageID, PerfCount: int64(len(samples))}
	if len(samples) == 0 {
		return rec
	}

	type acc struct {
		max, avgSum, min float64
		maxTime, minTime time.Time
	}
	newAcc := func() acc { return acc{max: -math.MaxFloat64, min: math.MaxFloat64} }
	upd := func(a *acc, v float64, t time.Time) {
		a.avgSum += v
		if v > a.max {
			a.max = v
			a.maxTime = t
		}
		if v < a.min {
			a.min = v
			a.minTime = t
		}
	}

	cpu, used := newAcc(), newAcc()
	readIOPS, writeIOPS, totalIOPS := newAcc(), newAcc(), newAcc()
	readMBps, writeMBps := newAcc(), newAcc()
	readLat, writeLat := newAcc(), newAcc()

	for _, s := range samples {
		if s == nil {
			continue
		}
		t := s.CollectedAt
		upd(&cpu, s.CPUUtilPercent, t)
		upd(&used, s.UsedPercent, t)
		upd(&readIOPS, s.ReadIOPS, t)
		upd(&writeIOPS, s.WriteIOPS, t)
		upd(&totalIOPS, s.TotalIOPS, t)
		upd(&readMBps, s.ReadMBps, t)
		upd(&writeMBps, s.WriteMBps, t)
		upd(&readLat, s.ReadLatencyMs, t)
		upd(&writeLat, s.WriteLatencyMs, t)
	}

	n := float64(len(samples))

	rec.CPUUtilMaximum, rec.CPUUtilMaximumTime = cpu.max, cpu.maxTime
	rec.CPUUtilMinimum, rec.CPUUtilMinimumTime = cpu.min, cpu.minTime
	rec.CPUUtilAverage = cpu.avgSum / n

	rec.UsedPercentMaximum, rec.UsedPercentMaximumTime = used.max, used.maxTime
	rec.UsedPercentMinimum, rec.UsedPercentMinimumTime = used.min, used.minTime
	rec.UsedPercentAverage = used.avgSum / n

	rec.ReadIOPSMaximum, rec.ReadIOPSMaximumTime = readIOPS.max, readIOPS.maxTime
	rec.ReadIOPSMinimum, rec.ReadIOPSMinimumTime = readIOPS.min, readIOPS.minTime
	rec.ReadIOPSAverage = readIOPS.avgSum / n

	rec.WriteIOPSMaximum, rec.WriteIOPSMaximumTime = writeIOPS.max, writeIOPS.maxTime
	rec.WriteIOPSMinimum, rec.WriteIOPSMinimumTime = writeIOPS.min, writeIOPS.minTime
	rec.WriteIOPSAverage = writeIOPS.avgSum / n

	rec.TotalIOPSMaximum, rec.TotalIOPSMaximumTime = totalIOPS.max, totalIOPS.maxTime
	rec.TotalIOPSMinimum, rec.TotalIOPSMinimumTime = totalIOPS.min, totalIOPS.minTime
	rec.TotalIOPSAverage = totalIOPS.avgSum / n

	rec.ReadMBpsMaximum, rec.ReadMBpsMaximumTime = readMBps.max, readMBps.maxTime
	rec.ReadMBpsMinimum, rec.ReadMBpsMinimumTime = readMBps.min, readMBps.minTime
	rec.ReadMBpsAverage = readMBps.avgSum / n

	rec.WriteMBpsMaximum, rec.WriteMBpsMaximumTime = writeMBps.max, writeMBps.maxTime
	rec.WriteMBpsMinimum, rec.WriteMBpsMinimumTime = writeMBps.min, writeMBps.minTime
	rec.WriteMBpsAverage = writeMBps.avgSum / n

	rec.ReadLatencyMsMaximum, rec.ReadLatencyMsMaximumTime = readLat.max, readLat.maxTime
	rec.ReadLatencyMsMinimum, rec.ReadLatencyMsMinimumTime = readLat.min, readLat.minTime
	rec.ReadLatencyMsAverage = readLat.avgSum / n

	rec.WriteLatencyMsMaximum, rec.WriteLatencyMsMaximumTime = writeLat.max, writeLat.maxTime
	rec.WriteLatencyMsMinimum, rec.WriteLatencyMsMinimumTime = writeLat.min, writeLat.minTime
	rec.WriteLatencyMsAverage = writeLat.avgSum / n

	return rec
}
