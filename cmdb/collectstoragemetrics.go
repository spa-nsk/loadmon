package cmdb

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	models "loadmon/models"
	"loadmon/lenovostorage"
	"loadmon/storages"
	datamap "loadmon/utilities"

	"github.com/beego/beego/v2/client/orm"
	"github.com/lovi-cloud/go-dorado-sdk/dorado"
)

// storageMetricRetention — срок хранения поминутных метрик СХД (storage_metric).
const storageMetricRetention = time.Hour * 24 * 3

// CollectStorageMetrics собирает поминутные метрики всех активных СХД -
// обязательно по КАЖДОЙ записи справочника storage_systems (Power==1),
// вне зависимости от вендора/ApiType (см. collectStorageMetricForSystem -
// явная ветка на каждый ApiType 1-5, без "default: ничего не делаем").
func CollectStorageMetrics() {
	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Power", 1).All(&systems); err != nil {
		log.Println("cmdb.CollectStorageMetrics: ошибка чтения storage_systems:", err)
		return
	}
	if len(systems) == 0 {
		// (2026-09-04) Раньше здесь был молчаливый return - если ни у одной
		// СХД в справочнике не выставлен Power=1 (поле по умолчанию 0,
		// проставляется только вручную на форме редактирования, см.
		// controllers/sprstoragesystems.go), CollectStorageMetrics каждую
		// минуту ничего не находил и тихо выходил - ни одной ошибки в
		// логе, ни одной записи в storage_metric НИКОГДА, что выглядело
		// неотличимо от "опрос не работает". Теперь при пустой выборке
		// это явно логируется.
		log.Println("cmdb.CollectStorageMetrics: нет ни одной СХД с Power=1 в storage_systems - опрос метрик пропущен")
		return
	}

	collectedAt := time.Now().Local().Truncate(time.Minute)
	rows := make([]*models.StorageMetric, 0, len(systems))
	for _, shd := range systems {
		if shd == nil || shd.Id <= 0 {
			continue
		}
		row := safeCollectStorageMetricForSystem(shd, collectedAt)
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

// safeCollectStorageMetricForSystem оборачивает collectStorageMetricForSystem
// в defer/recover (2026-09-03). Опрос СХД делает сетевые вызовы через
// сторонние клиентские библиотеки (github.com/lovi-cloud/go-dorado-sdk для
// Huawei, loadmon/lenovostorage для Lenovo, emcstorage для Unity), качество
// обработки ошибок в которых мы не контролируем - при неожиданном/битом
// HTTP-ответе они могут паниковать (например, nil pointer при разборе
// пустого/HTML вместо JSON тела). CollectStorageMetrics вызывается из
// metrics_grpc.go через `go cmdb.CollectStorageMetrics()` - непойманная
// паника в горутине убивает ВЕСЬ процесс metrics_grpc целиком (Go не
// изолирует панику одной горутины от остальных - это и давало эффект
// "metrics_grpc падает по исключениям при опросе метрик СХД", тяжело
// диагностируемый, так как трассировка уходит вглубь чужой библиотеки).
// Теперь паника при опросе ОДНОЙ СХД лишь логируется (с Id/ShortName/
// ApiType и текстом паники в PollError) и не мешает опросу остальных.
func safeCollectStorageMetricForSystem(shd *models.StorageSystems, collectedAt time.Time) (row *models.StorageMetric) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("cmdb.CollectStorageMetrics: ПАНИКА при опросе СХД Id=%d ShortName=%s ApiType=%d: %v", shd.Id, shd.ShortName, shd.ApiType, r)
			row = &models.StorageMetric{
				StorageSystemsId: shd.Id,
				CollectedAt:      collectedAt,
				ShortName:        shd.ShortName,
				UsedPercent:      shd.UsedPercent,
				PollOK:           false,
				PollError:        fmt.Sprintf("паника при опросе: %v", r),
			}
		}
	}()
	return collectStorageMetricForSystem(shd, collectedAt)
}

// collectStorageMetricForSystem собирает метрики ОДНОЙ СХД по её ApiType.
// Явная ветка на каждый известный ApiType (1-5) - по требованию
// "обязательна обработка всех СХД разных вендоров, представленных в
// справочнике СХД" (2026-09-03): раньше был switch{case 2: Unity;
// default: ничего}, из-за чего VNX/Dorado/OceanStor/Lenovo молча
// пропускались без единой пометки. Теперь для каждого вендора либо
// собираются реальные данные там, где это позволяют доступные клиентские
// библиотеки (Unity KPI, Huawei/Lenovo - % использования пулов), либо явно
// фиксируется, что живого API метрик для этого вендора в проекте пока нет
// (row.PollError, row.PollOK=true - это не ошибка опроса, а
// задокументированное ограничение).
func collectStorageMetricForSystem(shd *models.StorageSystems, collectedAt time.Time) *models.StorageMetric {
	row := &models.StorageMetric{
		StorageSystemsId: shd.Id,
		CollectedAt:      collectedAt,
		ShortName:        shd.ShortName,
		UsedPercent:      shd.UsedPercent, // дефолт - статическое значение из справочника, ниже может быть заменено живым
		PollOK:           true,
	}

	switch shd.ApiType {
	case 1: // VNX CLI - в проекте нет клиента для VNX KPI/performance API,
		// живых метрик не существует технически (см. storages/vnxcli.go -
		// только листинг LUN/дисков/HBA, без perf-эндпоинтов). UsedPercent
		// остаётся статическим значением из справочника СХД (row.UsedPercent
		// выше) - это не сбой опроса, поэтому PollOK=true, PollError пуст.

	case 2: // Unity REST - живые KPI, см. storages/unity_hardware.go.
		var failed []string
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.sp.*.utilization"); ok {
			row.CPUUtilPercent = v
		} else {
			failed = append(failed, "CPUUtilPercent")
		}
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.pool.*.percentUsed"); ok {
			row.UsedPercent = v
		} else {
			failed = append(failed, "UsedPercent")
		}
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.ioRate.read"); ok {
			row.ReadIOPS = v
		} else {
			failed = append(failed, "ReadIOPS")
		}
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.ioRate.write"); ok {
			row.WriteIOPS = v
		} else {
			failed = append(failed, "WriteIOPS")
		}
		row.TotalIOPS = row.ReadIOPS + row.WriteIOPS
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.throughput.read"); ok {
			row.ReadMBps = v / (1024 * 1024)
		} else {
			failed = append(failed, "ReadMBps")
		}
		if v, ok := storages.GetUnityKpiSum(shd.IpAddress, "kpi.sp.*.throughput.write"); ok {
			row.WriteMBps = v / (1024 * 1024)
		} else {
			failed = append(failed, "WriteMBps")
		}
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.sp.*.responseTime.read"); ok {
			row.ReadLatencyMs = v
		} else {
			failed = append(failed, "ReadLatencyMs")
		}
		if v, ok := storages.GetUnityKpiAverage(shd.IpAddress, "kpi.sp.*.responseTime.write"); ok {
			row.WriteLatencyMs = v
		} else {
			failed = append(failed, "WriteLatencyMs")
		}
		if len(failed) > 0 {
			row.PollOK = false
			row.PollError = "не удалось получить: " + strings.Join(failed, ", ")
			log.Println("cmdb.collectStorageMetricForSystem: частичный сбой опроса KPI Unity", shd.ShortName, shd.IpAddress, "-", row.PollError)
		}

	case 3, 4: // Huawei Dorado / OceanStor REST.
		if v, err := collectHuaweiUsedPercent(shd); err == nil {
			row.UsedPercent = v
		} else {
			row.PollOK = false
			row.PollError = "UsedPercent (пулы): " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса пулов Huawei", shd.ShortName, shd.IpAddress, "-", err)
		}
		// IOPS/throughput/latency/CPU - недоступны: используемый SDK
		// (github.com/lovi-cloud/go-dorado-sdk) не содержит performance/KPI
		// эндпоинтов, только LUN/host/pool/system - см. ревью в этой сессии.

	case 5: // Lenovo (SANtricity REST).
		if v, err := collectLenovoUsedPercent(shd); err == nil {
			row.UsedPercent = v
		} else {
			row.PollOK = false
			row.PollError = "UsedPercent (пулы): " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса пулов Lenovo", shd.ShortName, shd.IpAddress, "-", err)
		}
		// IOPS/throughput/latency/CPU - недоступны: пакет loadmon/lenovostorage
		// не реализует statistics/analysed-performance эндпоинты SANtricity API.

	default:
		row.PollOK = false
		row.PollError = fmt.Sprintf("неизвестный ApiType=%d", shd.ApiType)
		log.Println("cmdb.collectStorageMetricForSystem: неизвестный ApiType у СХД", shd.ShortName, shd.ApiType)
	}

	return row
}

// collectHuaweiUsedPercent возвращает средний % использования пулов СХД
// Huawei Dorado/OceanStor (взвешенно по ёмкости пула, если она известна,
// иначе простое среднее) через client.LocalDevice.GetStoragePools -
// единственный доступный в подключённом SDK источник живой утилизации для
// этого вендора (без per-LUN/per-контроллер KPI).
func collectHuaweiUsedPercent(shd *models.StorageSystems) (float64, error) {
	key := "spaview"
	keypass := datamap.GetKeyData(key)
	remoteIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}
	localIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}

	client, err := dorado.NewClient(localIps, remoteIps, key, keypass, "", nil)
	if err != nil {
		return 0, fmt.Errorf("подключение: %w", err)
	}

	pools, err := client.LocalDevice.GetStoragePools(context.Background(), nil)
	if err != nil {
		return 0, fmt.Errorf("список пулов: %w", err)
	}
	if len(pools) == 0 {
		return 0, fmt.Errorf("СХД не вернула ни одного пула")
	}

	var usedSum, totalSum float64
	var haveCapacity bool
	var simpleSum float64
	var simpleCount int
	for _, p := range pools {
		pctStr := strings.TrimSpace(p.USERCONSUMEDCAPACITYPERCENTAGE)
		pct, perr := strconv.ParseFloat(pctStr, 64)
		if perr != nil {
			continue
		}
		simpleSum += pct
		simpleCount++

		totalStr := strings.TrimSpace(p.USERTOTALCAPACITY)
		total, terr := strconv.ParseFloat(totalStr, 64)
		if terr == nil && total > 0 {
			usedSum += pct * total
			totalSum += total
			haveCapacity = true
		}
	}
	if haveCapacity && totalSum > 0 {
		return usedSum / totalSum, nil
	}
	if simpleCount > 0 {
		return simpleSum / float64(simpleCount), nil
	}
	return 0, fmt.Errorf("не удалось разобрать процент использования ни одного пула")
}

// collectLenovoUsedPercent возвращает % использования СХД Lenovo как
// UsedSpace/TotalRaidedSpace, просуммированные по всем storage-pools
// (session.GetStoragePools - поля в байтах строками, см.
// lenovostorage/services.go:StoragePool).
func collectLenovoUsedPercent(shd *models.StorageSystems) (float64, error) {
	key := "monitor"
	session, err := lenovostorage.NewSession(shd.IpAddress, true, key, datamap.GetKeyData(key))
	if err != nil {
		return 0, fmt.Errorf("авторизация: %w", err)
	}

	pools, err := session.GetStoragePools()
	if err != nil {
		return 0, fmt.Errorf("список пулов: %w", err)
	}
	if len(pools) == 0 {
		return 0, fmt.Errorf("СХД не вернула ни одного пула")
	}

	var usedSum, totalSum float64
	for _, p := range pools {
		used, uerr := strconv.ParseFloat(strings.TrimSpace(p.UsedSpace), 64)
		total, terr := strconv.ParseFloat(strings.TrimSpace(p.TotalRaidedSpace), 64)
		if uerr != nil || terr != nil || total <= 0 {
			continue
		}
		usedSum += used
		totalSum += total
	}
	if totalSum <= 0 {
		return 0, fmt.Errorf("не удалось разобрать ёмкость ни одного пула")
	}
	return usedSum / totalSum * 100, nil
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
