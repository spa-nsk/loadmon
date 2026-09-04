package cmdb

import (
	"context"
	"fmt"
	"log"
	"time"

	models "loadmon/models"
	"loadmon/storages"

	"github.com/beego/beego/v2/client/orm"
)

// storageMetricRetention — срок хранения поминутных метрик СХД (storage_metric).
const storageMetricRetention = time.Hour * 24 * 3

// CollectStorageMetrics собирает поминутные метрики всех активных СХД.
func CollectStorageMetrics() {
	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Power", 1).All(&systems); err != nil {
		log.Println("cmdb.CollectStorageMetrics: ошибка чтения storage_systems:", err)
		return
	}
	if len(systems) == 0 {
		return
	}

	collectedAt := time.Now().Local().Truncate(time.Minute)
	rows := make([]*models.StorageMetric, 0, len(systems))
	for _, shd := range systems {
		if shd == nil || shd.Id <= 0 {
			continue
		}
		row := collectStorageMetricForSystem(shd, collectedAt)
		if row != nil {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return
	}
	if err := models.ReplaceStorageMetrics(context.Background(), rows); err != nil {
		log.Println("cmdb.CollectStorageMetrics: ошибка записи storage_metric:", err)
	}
}

func collectStorageMetricForSystem(shd *models.StorageSystems, collectedAt time.Time) *models.StorageMetric {
	row := &models.StorageMetric{
		StorageSystemsId: shd.Id,
		CollectedAt:      collectedAt,
		ShortName:        shd.ShortName,
		UsedPercent:      shd.UsedPercent,
	}

	switch shd.ApiType {
	case 2: // Unity REST
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.sp.*.utilization"); ok {
			row.CPUUtilPercent = v
		}
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.pool.*.percentUsed"); ok {
			row.UsedPercent = v
		}
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.ioRate.read"); ok {
			row.ReadIOPS = v
		}
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.ioRate.write"); ok {
			row.WriteIOPS = v
		}
		row.TotalIOPS = row.ReadIOPS + row.WriteIOPS
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.throughput.read"); ok {
			row.ReadMBps = v / (1024 * 1024)
		}
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.throughput.write"); ok {
			row.WriteMBps = v / (1024 * 1024)
		}
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.sp.*.responseTime.read"); ok {
			row.ReadLatencyMs = v
		}
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.sp.*.responseTime.write"); ok {
			row.WriteLatencyMs = v
		}
	default:
		// VNX/Huawei/Lenovo — пока только UsedPercent из справочника СХД.
	}

	return row
}

// ClearOldStorageMetric удаляет поминутные метрики СХД старше storageMetricRetention.
func ClearOldStorageMetric() {
	currentTime := time.Now().Local()
	currentTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.Local)
	cutoff := currentTime.Add(-storageMetricRetention)
	o := orm.NewOrm()
	count := int64(0)
	for {
		res, err := o.Raw(
			"DELETE FROM storage_metric WHERE id IN (SELECT id FROM storage_metric WHERE collected_at <= ? LIMIT 1000)",
			cutoff,
		).Exec()
		if err != nil {
			fmt.Println("Ошибка удаления старых записей из таблицы storage_metric", err)
			break
		}
		num, _ := res.RowsAffected()
		count += num
		if num == 0 {
			break
		}
	}
	if count > 0 {
		fmt.Println("Удалено:", count, "записей старее", storageMetricRetention.String(), "из таблицы storage_metric")
	}
}
