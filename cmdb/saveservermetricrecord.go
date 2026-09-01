package cmdb

import (
	"fmt"
	"log"
	"math"
	"time"

	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// SaveServerMetricRecord строит суточную агрегированную запись
// models.ServerMetricRecord (таблица server_metric_record) из поминутных
// "сырых" снимков models.ServerMetric (таблица server_metric, заполняется
// пакетом zabbixmetrics, см. kontext/03-data-model.md) - по аналогии с тем,
// как cmdb.SaveVMRecord строит суточный models.VMRecord (vm_record) из
// "сырых" замеров models.VMPerfRecord (vm_perf).
//
// Вызывается из cmdb.SaveMetricData (см. metrics_grpc/metrics_grpc.go,
// планируется раз в сутки около 23:59:59) - агрегирует данные ТЕКУЩИХ
// суток (от начала суток currentTime до самого currentTime), т.к. на этот
// момент почти весь день уже собран. currentTime - то же время, что
// SaveMetricData передаёт в SaveClusterData/SaveVMRecord/SaveHostRecord -
// единое время "среза" для всех суточных записей одного цикла записи.
func SaveServerMetricRecord(currentTime time.Time) {
	dayStart := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
	dayEnd := currentTime

	o := orm.NewOrm()

	var rows []*models.ServerMetric
	qs := o.QueryTable(new(models.ServerMetric))
	qs = qs.Filter("CollectedAt__gte", dayStart).Filter("CollectedAt__lt", dayEnd)
	qs = qs.OrderBy("HostId", "CollectedAt")
	num, err := qs.All(&rows)
	if err != nil {
		fmt.Println("cmdb.SaveServerMetricRecord: ошибка чтения server_metric за", dayStart, "-", dayEnd, ":", err)
		return
	}
	if num == 0 {
		return // за сутки не было ни одного снимка - агрегировать нечего (например, zabbixmetrics ещё не собирал данные)
	}

	// Группируем по HostId, сохраняя порядок появления - агрегируем в один
	// проход по отсортированному (HostId, CollectedAt) списку.
	byHost := make(map[int64][]*models.ServerMetric)
	hostOrder := make([]int64, 0)
	for _, r := range rows {
		if r == nil {
			continue
		}
		if _, ok := byHost[r.HostId]; !ok {
			hostOrder = append(hostOrder, r.HostId)
		}
		byHost[r.HostId] = append(byHost[r.HostId], r)
	}

	for _, hostId := range hostOrder {
		samples := byHost[hostId]
		rec := aggregateServerMetricRecord(hostId, samples)
		rec.Date = dayStart
		rec.Time_Mfy = time.Now().Local()

		// ServerName/ServerId - через справочник zabbix_host (models.ZabbixHost),
		// на который ссылается ServerMetric.HostId (см. её комментарий).
		zh := models.ZabbixHost{Id: hostId}
		if err := o.Read(&zh); err == nil {
			rec.ServerName = zh.Name
			if rec.ServerName == "" {
				rec.ServerName = zh.Host
			}
			rec.ServerId = zh.ServerId
		} else {
			// Хост мог быть удалён из справочника между сбором сырых метрик и
			// формированием суточной записи - не блокируем агрегацию, просто
			// оставляем ServerName/ServerId пустыми (роднит с обработкой
			// отсутствующих связей в остальном cmdb, см. например
			// cmdb.CheckServerSprRecordsExistence).
			if len(samples) > 0 {
				rec.ServerName = samples[0].ServerName
			}
		}

		// Идемпотентный upsert по (HostId, Date) - как models.ReplaceMetrics
		// для "сырой" таблицы server_metric, чтобы повторный вызов
		// SaveMetricData за те же сутки не плодил дублирующихся суточных
		// записей (TableUnique у ServerMetricRecord тоже (HostId, Date), но
		// полагаться только на ошибку уникального индекса означало бы терять
		// обновлённые за более поздний вызов агрегаты).
		existing := models.ServerMetricRecord{}
		existing.HostId = hostId
		existing.Date = dayStart
		if err := o.Read(&existing, "HostId", "Date"); err == nil {
			rec.Id = existing.Id
			if _, err := o.Update(rec); err != nil {
				log.Println("cmdb.SaveServerMetricRecord: ошибка обновления записи server_metric_record HostId=", hostId, err)
			}
		} else {
			if _, err := o.Insert(rec); err != nil {
				log.Println("cmdb.SaveServerMetricRecord: ошибка вставки записи server_metric_record HostId=", hostId, err)
			}
		}
	}
}

// aggregateServerMetricRecord считает Max/Average/Min (+время достижения
// Max/Min) по всем полям-метрикам models.ServerMetric среди samples
// (поминутные снимки одного хоста за сутки, отсортированные по
// CollectedAt) и возвращает готовую (кроме Date/ServerName/ServerId/
// Time_Mfy - заполняются вызывающим кодом) models.ServerMetricRecord.
func aggregateServerMetricRecord(hostId int64, samples []*models.ServerMetric) *models.ServerMetricRecord {
	rec := &models.ServerMetricRecord{HostId: hostId, PerfCount: int64(len(samples))}
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

	cpu, ram := newAcc(), newAcc()
	diskRead, diskWrite := newAcc(), newAcc()
	diskReadLat, diskWriteLat := newAcc(), newAcc()
	diskReadQ, diskWriteQ := newAcc(), newAcc()
	netRecv, netSent := newAcc(), newAcc()

	for _, s := range samples {
		if s == nil {
			continue
		}
		upd(&cpu, s.CPUUtilPercent, s.CollectedAt)
		upd(&ram, s.RAMUtilPercent, s.CollectedAt)
		upd(&diskRead, s.DiskReadMBps, s.CollectedAt)
		upd(&diskWrite, s.DiskWriteMBps, s.CollectedAt)
		upd(&diskReadLat, s.DiskReadLatencyMs, s.CollectedAt)
		upd(&diskWriteLat, s.DiskWriteLatencyMs, s.CollectedAt)
		upd(&diskReadQ, s.DiskReadQueue, s.CollectedAt)
		upd(&diskWriteQ, s.DiskWriteQueue, s.CollectedAt)
		upd(&netRecv, s.NetRecvMBps, s.CollectedAt)
		upd(&netSent, s.NetSentMBps, s.CollectedAt)
	}

	n := float64(len(samples))

	rec.CPUUtilMaximum, rec.CPUUtilMaximumTime = cpu.max, cpu.maxTime
	rec.CPUUtilMinimum, rec.CPUUtilMinimumTime = cpu.min, cpu.minTime
	rec.CPUUtilAverage = cpu.avgSum / n

	rec.RAMUtilMaximum, rec.RAMUtilMaximumTime = ram.max, ram.maxTime
	rec.RAMUtilMinimum, rec.RAMUtilMinimumTime = ram.min, ram.minTime
	rec.RAMUtilAverage = ram.avgSum / n

	rec.DiskReadMBpsMaximum, rec.DiskReadMBpsMaximumTime = diskRead.max, diskRead.maxTime
	rec.DiskReadMBpsMinimum, rec.DiskReadMBpsMinimumTime = diskRead.min, diskRead.minTime
	rec.DiskReadMBpsAverage = diskRead.avgSum / n

	rec.DiskWriteMBpsMaximum, rec.DiskWriteMBpsMaximumTime = diskWrite.max, diskWrite.maxTime
	rec.DiskWriteMBpsMinimum, rec.DiskWriteMBpsMinimumTime = diskWrite.min, diskWrite.minTime
	rec.DiskWriteMBpsAverage = diskWrite.avgSum / n

	rec.DiskReadLatencyMsMaximum, rec.DiskReadLatencyMsMaximumTime = diskReadLat.max, diskReadLat.maxTime
	rec.DiskReadLatencyMsMinimum, rec.DiskReadLatencyMsMinimumTime = diskReadLat.min, diskReadLat.minTime
	rec.DiskReadLatencyMsAverage = diskReadLat.avgSum / n

	rec.DiskWriteLatencyMsMaximum, rec.DiskWriteLatencyMsMaximumTime = diskWriteLat.max, diskWriteLat.maxTime
	rec.DiskWriteLatencyMsMinimum, rec.DiskWriteLatencyMsMinimumTime = diskWriteLat.min, diskWriteLat.minTime
	rec.DiskWriteLatencyMsAverage = diskWriteLat.avgSum / n

	rec.DiskReadQueueMaximum, rec.DiskReadQueueMaximumTime = diskReadQ.max, diskReadQ.maxTime
	rec.DiskReadQueueMinimum, rec.DiskReadQueueMinimumTime = diskReadQ.min, diskReadQ.minTime
	rec.DiskReadQueueAverage = diskReadQ.avgSum / n

	rec.DiskWriteQueueMaximum, rec.DiskWriteQueueMaximumTime = diskWriteQ.max, diskWriteQ.maxTime
	rec.DiskWriteQueueMinimum, rec.DiskWriteQueueMinimumTime = diskWriteQ.min, diskWriteQ.minTime
	rec.DiskWriteQueueAverage = diskWriteQ.avgSum / n

	rec.NetRecvMBpsMaximum, rec.NetRecvMBpsMaximumTime = netRecv.max, netRecv.maxTime
	rec.NetRecvMBpsMinimum, rec.NetRecvMBpsMinimumTime = netRecv.min, netRecv.minTime
	rec.NetRecvMBpsAverage = netRecv.avgSum / n

	rec.NetSentMBpsMaximum, rec.NetSentMBpsMaximumTime = netSent.max, netSent.maxTime
	rec.NetSentMBpsMinimum, rec.NetSentMBpsMinimumTime = netSent.min, netSent.minTime
	rec.NetSentMBpsAverage = netSent.avgSum / n

	return rec
}
