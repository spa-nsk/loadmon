package cmdb

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"loadmon/lenovostorage"
	models "loadmon/models"
	"loadmon/storages"
	datamap "loadmon/utilities"

	"github.com/beego/beego/v2/client/orm"
	"github.com/lovi-cloud/go-dorado-sdk/dorado"
)

// storageMetricRetention — срок хранения поминутных метрик СХД (storage_metric).
const storageMetricRetention = time.Hour * 24 * 3

// CollectStorageMetricsOnly — если не пусто, CollectStorageMetrics опрашивает
// ТОЛЬКО одну СХД с этим ShortName (без учёта регистра, точное совпадение) -
// используется отдельной отладочной утилитой tools/storagemetrics (по
// аналогии с cmdb.UpdateVolumeStorageOnly у tools/storagesync), чтобы можно
// было итеративно отлаживать опрос конкретной СХД без полного цикла
// metrics_grpc. Пустая строка (по умолчанию, обычный режим metrics_grpc) -
// опрашиваются все СХД с Power=1, как раньше.
var CollectStorageMetricsOnly string

// collectStorageMetricsRunning - признак выполняющегося цикла
// CollectStorageMetrics. metrics_grpc запускает его каждую минуту в новой
// горутине, а цикл по всем СХД длится дольше минуты (один опрос Unity с
// сотнями LUN - 10+ пачек real-time запросов по ~11 с). Без этой защиты
// циклы накладывались, и к одной Unity одновременно шло несколько
// real-time запросов из разных циклов - отсюда HTTP 422 "The system has
// reached the maximum number of real-time subscriptions" (0x7d14024,
// 2026-10-02). Новый цикл, пока предыдущий не закончился, пропускается.
var collectStorageMetricsRunning int32

// storageMetricWorkers - сколько СХД опрашивается параллельно в одном цикле.
// Разные СХД не делят лимит real-time подписок (он у каждой Unity свой),
// а одну СХД в цикле опрашивает ровно один воркер.
const storageMetricWorkers = 4

// CollectStorageMetrics собирает поминутные метрики всех активных СХД -
// обязательно по КАЖДОЙ записи справочника storage_systems (Power==1),
// вне зависимости от вендора/ApiType (см. collectStorageMetricForSystem -
// явная ветка на каждый ApiType 1-5, без "default: ничего не делаем"). Если
// задан CollectStorageMetricsOnly - опрашивается только одна СХД с этим
// ShortName.
func CollectStorageMetrics() {
	if !atomic.CompareAndSwapInt32(&collectStorageMetricsRunning, 0, 1) {
		log.Println("cmdb.CollectStorageMetrics: предыдущий цикл опроса СХД ещё выполняется - этот запуск пропущен")
		return
	}
	defer atomic.StoreInt32(&collectStorageMetricsRunning, 0)
	started := time.Now()

	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Power", 1).All(&systems); err != nil {
		log.Println("cmdb.CollectStorageMetrics: ошибка чтения storage_systems:", err)
		return
	}
	if CollectStorageMetricsOnly != "" {
		filtered := systems[:0]
		for _, shd := range systems {
			if shd != nil && strings.EqualFold(shd.ShortName, CollectStorageMetricsOnly) {
				filtered = append(filtered, shd)
			}
		}
		systems = filtered
		if len(systems) == 0 {
			log.Printf("cmdb.CollectStorageMetrics: СХД с ShortName=%q (Power=1) не найдена - опрос пропущен", CollectStorageMetricsOnly)
			return
		}
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
	var (
		mu   sync.Mutex
		wg   sync.WaitGroup
		jobs = make(chan *models.StorageSystems)
	)
	for w := 0; w < storageMetricWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for shd := range jobs {
				row := safeCollectStorageMetricForSystem(shd, collectedAt)
				if row != nil {
					mu.Lock()
					rows = append(rows, row)
					mu.Unlock()
				}
			}
		}()
	}
	for _, shd := range systems {
		if shd == nil || shd.Id <= 0 {
			continue
		}
		jobs <- shd
	}
	close(jobs)
	wg.Wait()
	if len(rows) == 0 {
		// (2026-09-22) Аналогично случаю "нет СХД с Power=1" выше - раньше
		// здесь тоже был молчаливый return. По жалобе пользователя
		// "метрики не собираются, storage_metric пустая", при этом ни
		// строка "ошибка записи storage_metric", ни эта не находились в
		// логе - т.е. неясно было, на каком именно шаге теряются данные
		// (пустой rows, ошибка записи, или сбор вовсе не запускается).
		// Теперь оба исхода явно логируются, чтобы при следующем
		// воспроизведении сразу было видно, где именно рвётся цепочка.
		log.Printf("cmdb.CollectStorageMetrics: из %d СХД (Power=1) не собрано ни одной строки метрик - storage_metric не пополнится в этом цикле", len(systems))
		return
	}
	if err := models.ReplaceStorageMetrics(context.Background(), rows); err != nil {
		log.Println("cmdb.CollectStorageMetrics: ошибка записи storage_metric:", err)
		return
	}
	log.Printf("cmdb.CollectStorageMetrics: записано %d строк в storage_metric за %s (СХД с Power=1: %d), цикл занял %s", len(rows), collectedAt.Format("2006-01-02 15:04:05"), len(systems), time.Since(started).Round(time.Second))
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

	case 2: // Unity REST.
		// (2026-09-22) Раньше здесь опрашивались 8 метрик через
		// storages.GetUnityKpiAverage/GetUnityKpiSum по путям вида
		// "kpi.sp.*.utilization"/"kpi.pool.*.percentUsed"/"kpi.sp.*.ioRate.*"/
		// "kpi.sp.*.throughput.*"/"kpi.sp.*.responseTime.*" - на реальном
		// оборудовании это давало 100% "частичный сбой" по ВСЕМ 8 метрикам на
		// КАЖДОЙ Unity СХД без единого раза успеха (жалоба пользователя "по
		// Unity везде частичная ошибка, на всех СХД"). Разбор показал два
		// независимых бага:
		//   1) сама ошибка глушилась молча (unityKpiLatestPerEntry отбрасывал
		//      err, наружу уходило только ok=false) - поэтому не было видно,
		//      что реально не так;
		//   2) REST-эндпоинт /api/types/kpiValue/instances, который
		//      использовался под капотом (emcstorage.GetkpiValue), не
		//      существует в Unity/Unisphere REST API вовсе - подтверждённые
		//      документацией Dell типы для метрик - metricValue (история) и
		//      metricRealTimeQuery/metricQueryResult (реальное время), а не
		//      kpiValue. Поэтому опрос 100% падал на каждой СХД, вне
		//      зависимости от прошивки/модели/сети.
		// UsedPercent теперь считается тем же проверенным способом, что и у
		// Huawei/Lenovo (collectHuaweiUsedPercent/collectLenovoUsedPercent) -
		// через уже рабочий (используется в syncstoragehardware.go)
		// storages.GetUnityPools -> SizeUsed/SizeTotal, а не через
		// несуществующий kpi-путь.
		//
		// 2026-09-23: IOPS/throughput/latency/CPUUtilPercent РЕАЛИЗОВАНЫ -
		// пользователь снял реальный каталог метрик массива
		// (`-dumpmetricsonly`, после починки бага с `fields`/пагинацией,
		// см. CHANGELOG) - 2862 подтверждённых путей, включая нужные
		// `kpi.lun.+.sp.+.*IOPS`/`*Bandwidth`/`responseTime` (real-time,
		// суммируются по всем LUN) и `sp.*.cpu.summary.busyTicks`/
		// `idleTicks` (для CPU% через дельту). См.
		// storages.GetUnityLiveKpi/collectUnityKpi ниже и
		// kontext/12-unity-rest-api.md - подробности схемы, единиц
		// измерения и оставшихся неподтверждённых нюансов (единица
		// responseTime в real-time-варианте пути).
		log.Println("cmdb.collectStorageMetricForSystem: опрос пулов Unity", shd.ShortName, shd.IpAddress)
		if v, err := collectUnityUsedPercent(shd); err == nil {
			row.UsedPercent = v
		} else {
			row.PollOK = false
			row.PollError = "UsedPercent (пулы): " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса пулов Unity", shd.ShortName, shd.IpAddress, "-", err)
		}
		log.Println("cmdb.collectStorageMetricForSystem: опрос живых KPI Unity", shd.ShortName, shd.IpAddress)
		if err := collectUnityKpi(shd, row); err != nil {
			row.PollOK = false
			if row.PollError != "" {
				row.PollError += "; "
			}
			row.PollError += "KPI: " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса KPI Unity", shd.ShortName, shd.IpAddress, "-", err)
		}

	case 3, 4: // Huawei Dorado / OceanStor REST.
		log.Println("cmdb.collectStorageMetricForSystem: опрос KPI Huawei", shd.ShortName, shd.IpAddress)
		if v, err := collectHuaweiUsedPercent(shd); err == nil {
			row.UsedPercent = v
		} else {
			row.PollOK = false
			row.PollError = "UsedPercent (пулы): " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса пулов Huawei", shd.ShortName, shd.IpAddress, "-", err)
		}
		// 2026-09-23: живые IOPS/throughput/latency/CPU через
		// dorado.Device.GetControllers (CPUUSAGE) и GetCurStatisticData
		// (performace_statistic) - добавлены в SDK этим же запросом
		// ("переделай storagemetrics исходя из последних данных") после
		// того, как пользователь предоставил официальный REST Interface
		// Reference (huawei-api.md), подтвердивший реальную схему (см.
		// kontext/10-huawei-dorado-rest-api.md, раздел "Источник:
		// huawei-api.md"). Раньше здесь не было вообще ничего, кроме
		// UsedPercent - см. комментарий у collectHuaweiKpi ниже.
		if err := collectHuaweiKpi(shd, row); err != nil {
			row.PollOK = false
			if row.PollError != "" {
				row.PollError += "; "
			}
			row.PollError += "KPI: " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса KPI Huawei", shd.ShortName, shd.IpAddress, "-", err)
		}

	case 5: // Lenovo (SANtricity REST).
		if v, err := collectLenovoUsedPercent(shd); err == nil {
			row.UsedPercent = v
		} else {
			row.PollOK = false
			row.PollError = "UsedPercent (пулы): " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса пулов Lenovo", shd.ShortName, shd.IpAddress, "-", err)
		}
		// 2026-09-22: живые KPI/IOPS/throughput/latency через
		// lenovostorage.GetAnalysedVolumeStatistics/GetAnalysedControllerStatistics
		// (добавлены в SDK этим же запросом - "вернёмся к обработке метрик
		// из СХД... добавь функции в SDK для СХД, при потребности"). Раньше
		// этих методов не было вовсе - см. комментарий выше у case 5 в
		// прежней редакции.
		if err := collectLenovoKpi(shd, row); err != nil {
			row.PollOK = false
			if row.PollError != "" {
				row.PollError += "; "
			}
			row.PollError += "KPI: " + err.Error()
			log.Println("cmdb.collectStorageMetricForSystem: ошибка опроса KPI Lenovo", shd.ShortName, shd.IpAddress, "-", err)
		}

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
		// 2026-09-24: НЕ ошибка - см. тот же комментарий у
		// collectUnityUsedPercent (подтверждено пользователем на реальных
		// СХД: "не размечены, пулов нет и это не ошибка").
		log.Printf("cmdb.collectHuaweiUsedPercent(%s): пулы не найдены (СХД, вероятно, ещё не размечена) - UsedPercent недоступен, это не ошибка", shd.IpAddress)
		return 0, nil
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

// huaweiPerfIndicatorReadMBps/... - коды индикаторов производительности
// Huawei DeviceManager REST API (`CMO_STATISTIC_DATA_ID_LIST`), подтверждённые
// официальным REST Interface Reference. IOPS/throughput (23/25/26/28) -
// изначально из `huawei-api.md`, см. kontext/10-huawei-dorado-rest-api.md.
// Раздельные read/write индикаторы задержки (384/385) - НАЙДЕНЫ 2026-09-23 в
// официальных документах, добавленных пользователем loadmon в этот же день
// ("добавил свежие описания по работе с СХД Huawei и Dell EMC"):
// doc-API-hardware/OceanStor Dorado 6.1.6 REST Interface Reference-ru.md и
// .../OceanStor Dorado V700R001C10 REST Interface Reference-ru.md, раздел
// "4.3.1 Block Storage Performance Indicators" (приложение) - в ОБЕИХ версиях
// документа коды идентичны: 384 = "Avg. Read I/O Response Time (us)", 385 =
// "Avg. Write I/O Response Time (us)", оба подтверждены для объекта
// Controller (TYPE=207, тот же, что используется в collectHuaweiKpi). НЕ
// угаданы - взяты дословно из таблицы индикаторов. Заменяют ранее
// использовавшийся общий индикатор 370 (Avg. I/O Response Time) - см. историю
// ниже у collectHuaweiKpi.
const (
	huaweiPerfIndicatorReadMBps      = "23"  // Read Bandwidth (MB/s)
	huaweiPerfIndicatorReadIOPS      = "25"  // Read IOPS
	huaweiPerfIndicatorWriteMBps     = "26"  // Write Bandwidth (MB/s)
	huaweiPerfIndicatorWriteIOPS     = "28"  // Write IOPS
	huaweiPerfIndicatorReadLatencyUs  = "384" // Avg. Read I/O Response Time (us)
	huaweiPerfIndicatorWriteLatencyUs = "385" // Avg. Write I/O Response Time (us)
)

// collectHuaweiKpi заполняет row живыми KPI Huawei (IOPS/throughput/
// latency/CPU) через новые методы SDK dorado.Device.GetControllers
// (CPUUSAGE - готовое поле объекта "controller", без отдельного
// performance-запроса) и GetCurStatisticData (performace_statistic -
// IOPS/throughput/latency). Добавлено 2026-09-23 по запросу пользователя
// "переделай storagemetrics исходя из последних данных" - после того как
// пользователь предоставил официальный REST Interface Reference
// (huawei-api.md), подтвердивший точную схему обоих запросов (см.
// kontext/10-huawei-dorado-rest-api.md, раздел "Источник: huawei-api.md").
// Тот же принцип агрегации по контроллерам, что и у collectLenovoKpi -
// суммируются IOPS/throughput по всем контроллерам, CPU и latency
// усредняются по контроллерам, от которых удалось получить значение.
//
// ⚠️ Не подтверждено на реальном оборудовании этого проекта (в отличие,
// например, от DISKTYPE/SECTORS) - схема взята из документации, но ни разу
// не проверена сырым ответом реального массива (см.
// tools/storagemetrics -huaweidumpraw для такой проверки). При первом
// реальном сбое стоит свериться именно так, а не гадать дальше.
//
// 2026-09-23: раньше ReadLatencyMs/WriteLatencyMs заполнялись ОДНИМ общим
// значением индикатора 370 (Avg. I/O Response Time, без разделения на
// read/write) - других кодов на тот момент в доступной документации не
// нашлось. После того как пользователь добавил официальные REST Interface
// Reference (Dorado 6.1.6 и V700R001C10, см. doc-API-hardware/), нашлись
// подтверждённые раздельные индикаторы 384 (Avg. Read I/O Response Time) и
// 385 (Avg. Write I/O Response Time) - см. комментарий у констант выше.
// Теперь ReadLatencyMs/WriteLatencyMs считаются раздельно, как у Unity/Lenovo.
func collectHuaweiKpi(shd *models.StorageSystems, row *models.StorageMetric) error {
	key := "spaview"
	keypass := datamap.GetKeyData(key)
	remoteIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}
	localIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}

	client, err := dorado.NewClient(localIps, remoteIps, key, keypass, "", nil)
	if err != nil {
		return fmt.Errorf("подключение: %w", err)
	}

	ctx := context.Background()
	controllers, err := client.LocalDevice.GetControllers(ctx)
	if err != nil {
		return fmt.Errorf("список контроллеров: %w", err)
	}
	if len(controllers) == 0 {
		return fmt.Errorf("СХД не вернула ни одного контроллера")
	}

	var errs []string

	var cpuSum float64
	var cpuN int
	for _, c := range controllers {
		if v, ok := c.CPUUsagePercent(); ok {
			cpuSum += v
			cpuN++
		}
	}
	if cpuN > 0 {
		row.CPUUtilPercent = cpuSum / float64(cpuN)
	} else {
		errs = append(errs, "CPUUSAGE: не удалось разобрать ни у одного контроллера")
	}

	var readIOPS, writeIOPS, readMBps, writeMBps, readLatencyUsSum, writeLatencyUsSum float64
	var readLatencyN, writeLatencyN int
	var statErrs int
	dataIDList := strings.Join([]string{
		huaweiPerfIndicatorReadMBps,
		huaweiPerfIndicatorReadIOPS,
		huaweiPerfIndicatorWriteMBps,
		huaweiPerfIndicatorWriteIOPS,
		huaweiPerfIndicatorReadLatencyUs,
		huaweiPerfIndicatorWriteLatencyUs,
	}, ",")
	for _, c := range controllers {
		if c.ID == "" {
			continue
		}
		uuid := "207:" + c.ID // 207 = TYPE контроллера, подтверждено huawei-api.md
		entries, serr := client.LocalDevice.GetCurStatisticData(ctx, uuid, dataIDList)
		if serr != nil {
			statErrs++
			continue
		}
		for _, e := range entries {
			vals := e.Values()
			if v, ok := vals[huaweiPerfIndicatorReadMBps]; ok {
				readMBps += v
			}
			if v, ok := vals[huaweiPerfIndicatorReadIOPS]; ok {
				readIOPS += v
			}
			if v, ok := vals[huaweiPerfIndicatorWriteMBps]; ok {
				writeMBps += v
			}
			if v, ok := vals[huaweiPerfIndicatorWriteIOPS]; ok {
				writeIOPS += v
			}
			if v, ok := vals[huaweiPerfIndicatorReadLatencyUs]; ok {
				readLatencyUsSum += v
				readLatencyN++
			}
			if v, ok := vals[huaweiPerfIndicatorWriteLatencyUs]; ok {
				writeLatencyUsSum += v
				writeLatencyN++
			}
		}
	}
	if statErrs > 0 && statErrs == len(controllers) {
		errs = append(errs, fmt.Sprintf("performace_statistic: не удалось получить данные ни у одного из %d контроллеров", len(controllers)))
	} else {
		row.ReadIOPS = readIOPS
		row.WriteIOPS = writeIOPS
		row.TotalIOPS = readIOPS + writeIOPS
		row.ReadMBps = readMBps
		row.WriteMBps = writeMBps
		if readLatencyN > 0 {
			row.ReadLatencyMs = (readLatencyUsSum / float64(readLatencyN)) / 1000 // us -> ms
		}
		if writeLatencyN > 0 {
			row.WriteLatencyMs = (writeLatencyUsSum / float64(writeLatencyN)) / 1000 // us -> ms
		}
		if statErrs > 0 {
			errs = append(errs, fmt.Sprintf("performace_statistic: не удалось получить данные у %d из %d контроллеров (учтены только успешные)", statErrs, len(controllers)))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// DumpHuaweiRawRequest выполняет произвольный GET/POST-запрос к REST API
// Huawei Dorado/OceanStor (та же авторизация/порт/схема, что и у
// collectHuaweiUsedPercent выше - ключ "spaview", https://<ip>:8088) и
// сохраняет СЫРОЙ ответ устройства в файл dumpPath через
// dorado.Device.RawRequestDump (go-dorado-sdk/dorado/rawrequest.go).
// Экспортирована (в отличие от остальных функций этого файла) для
// использования из tools/storagemetrics (-huaweidumpraw), см. комментарий
// там же. Добавлено 2026-09-22 по запросу пользователя loadmon "сделай
// сохранение сырых запросов по huawei" - для проверки реального формата
// /performance_statistic/... (или любого другого не реализованного пока в
// SDK эндпоинта) перед тем, как писать код разбора ответа - "не гадать", тот
// же принцип, что и у Unity (kontext/11-storage-metrics.md). rawQuery -
// СЫРАЯ query-строка без ведущего "?" (см. предупреждение в комментарии
// dorado.Device.RawRequestDump, 2026-09-23 - раньше query нужно было класть
// прямо в path, что было скрытым багом).
func DumpHuaweiRawRequest(shd *models.StorageSystems, method, path, rawQuery string, body []byte, dumpPath string) error {
	key := "spaview"
	keypass := datamap.GetKeyData(key)
	remoteIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}
	localIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}

	client, err := dorado.NewClient(localIps, remoteIps, key, keypass, "", nil)
	if err != nil {
		return fmt.Errorf("подключение: %w", err)
	}
	return client.LocalDevice.RawRequestDump(context.Background(), method, path, rawQuery, body, dumpPath)
}

// collectUnityUsedPercent возвращает % использования СХД Unity как
// SizeUsed/SizeTotal, просуммированные по всем storage-pools
// (storages.GetUnityPools -> emcstorage.PoolContent, уже используется и
// проверен в syncstoragehardware.go при синхронизации конфигурации СХД -
// тот же принцип, что и collectHuaweiUsedPercent/collectLenovoUsedPercent
// ниже). Добавлено 2026-09-22 взамен опроса через несуществующий Unity
// REST-эндпоинт /api/types/kpiValue/instances (path "kpi.pool.*.percentUsed")
// - см. комментарий у case 2 в collectStorageMetricForSystem.
func collectUnityUsedPercent(shd *models.StorageSystems) (float64, error) {
	pools, err := storages.GetUnityPools(shd.IpAddress)
	if err != nil {
		return 0, fmt.Errorf("список пулов: %w", err)
	}
	if len(pools) == 0 {
		// 2026-09-24: НЕ ошибка - подтверждено пользователем ("есть такие
		// СХД, он пока просто не размечены поэтому пулов нет и это не
		// ошибка. Просто и метрик по ним ждать нет смысла") - СХД может
		// быть заведена в справочник, но ещё не размечена (пулы не
		// созданы/не презентованы) - раньше это считалось ошибкой опроса
		// (PollOK=false на каждом цикле), хотя это ожидаемое состояние для
		// таких систем. UsedPercent остаётся 0 (честно - данных нет), но
		// PollOK не портится из-за этого.
		log.Printf("cmdb.collectUnityUsedPercent(%s): пулы не найдены (СХД, вероятно, ещё не размечена) - UsedPercent недоступен, это не ошибка", shd.IpAddress)
		return 0, nil
	}
	var usedSum, totalSum float64
	for _, p := range pools {
		total := float64(p.SizeTotal)
		used := float64(p.SizeUsed)
		if total <= 0 {
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

// collectUnityKpi заполняет row живыми KPI Unity (IOPS/throughput/latency/
// CPU) через storages.GetUnityLiveKpi - один real-time REST-запрос
// (metricRealTimeQuery/metricQueryResult), агрегирующий значения по ВСЕМ
// LUN и SP массива. Добавлено 2026-09-23 на основе РЕАЛЬНОГО каталога
// метрик, снятого пользователем (`-dumpmetricsonly` после починки бага с
// `fields`/пагинацией - см. CHANGELOG) на массиве 10.2.19.106 - 2862
// подтверждённых путей, включая нужные (см. таблицу и подробности в
// kontext/12-unity-rest-api.md):
//   - `kpi.lun.+.sp.+.readIOPS`/`writeIOPS`/`totalIOPS`
//   - `kpi.lun.+.sp.+.readBandwidth`/`writeBandwidth`/`totalBandwidth` (KB/s)
//   - `kpi.lun.+.sp.+.responseTime` (единый показатель, не раздельно read/write)
//   - `sp.*.cpu.summary.busyTicks`/`idleTicks` (счётчики тиков - CPU% через
//     дельту первого и последнего отсчёта, а не одно значение)
//
// Взамен прежнего "не реализовано, не гадать" (см. историю у case 2 в
// collectStorageMetricForSystem) - раньше не было подтверждённого
// документацией списка путей, теперь есть.
//
// ⚠️ Не полностью подтверждено на реальном оборудовании - пути метрик
// подтверждены каталогом устройства, но сама АГРЕГАЦИЯ (сумма по всем LUN,
// дельта тиков CPU) на момент реализации не сверена с независимым
// источником (например, Unisphere UI). Единица `responseTime` в
// real-time-варианте пути в каталоге не указана (`unitDisplayString`
// пусто) - использовано предположение "микросекунды" по аналогии с
// историческим вариантом ТОГО ЖЕ пути/ID, который документирован как
// микросекунды (см. storages.GetUnityLiveKpi) - если после первого
// реального цикла сбора значения задержки будут выглядеть неправдоподобно
// (на 3 порядка больше/меньше остальных вендоров) - стоит свериться с
// Unisphere UI, а не подстраивать код вслепую дальше.
//
// ⚠️ Добавляет ~11 секунд (unityLiveKpiWait) к опросу КАЖДОЙ Unity СХД в
// цикле - нужно минимум 2 отсчёта тиков CPU для дельты. При большом числе
// Unity-массивов это стоит учитывать (общее время цикла
// cmdb.CollectStorageMetrics растёт линейно с числом Unity СХД).
func collectUnityKpi(shd *models.StorageSystems, row *models.StorageMetric) error {
	// 2026-09-23: GetUnityLiveKpi теперь делает ДВА независимых real-time
	// запроса (LUN IOPS/bandwidth/responseTime и CPU busyTicks/idleTicks
	// отдельно, см. комментарий там же - были объединены в один запрос,
	// что давало HTTP 500 на реальном оборудовании) - поэтому она может
	// вернуть И частично заполненный kpi (то, что удалось), И ошибку
	// (что не удалось) ОДНОВРЕМЕННО. kpi нужно использовать в любом
	// случае, когда он не nil - ошибка ниже лишь ДОБАВЛЯЕТСЯ к тексту.
	kpi, err := storages.GetUnityLiveKpi(shd.IpAddress)
	if kpi == nil {
		return fmt.Errorf("real-time KPI: %w", err)
	}

	row.ReadIOPS = kpi.ReadIOPS
	row.WriteIOPS = kpi.WriteIOPS
	row.TotalIOPS = kpi.TotalIOPS
	row.ReadMBps = kpi.ReadMBps
	row.WriteMBps = kpi.WriteMBps
	if kpi.ResponseTimeOK {
		latencyMs := kpi.ResponseTimeUs / 1000 // предположение "микросекунды", см. комментарий у storages.GetUnityLiveKpi
		row.ReadLatencyMs = latencyMs
		row.WriteLatencyMs = latencyMs
	}
	if kpi.CPUUtilOK {
		row.CPUUtilPercent = kpi.CPUUtilPercent
	}

	if err != nil {
		return err
	}
	return nil
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
		// 2026-09-24: НЕ ошибка - см. тот же комментарий у
		// collectUnityUsedPercent (подтверждено пользователем на реальных
		// СХД: "не размечены, пулов нет и это не ошибка").
		log.Printf("cmdb.collectLenovoUsedPercent(%s): пулы не найдены (СХД, вероятно, ещё не размечена) - UsedPercent недоступен, это не ошибка", shd.IpAddress)
		return 0, nil
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

// collectLenovoKpi заполняет row живыми KPI Lenovo (IOPS/throughput/
// latency/CPU) через новые методы SDK lenovostorage.GetAnalysedVolumeStatistics
// (суммируется по всем томам - общая картина нагрузки СХД, тот же принцип,
// что и у Unity KPI выше по функции - kpi.sp.*) и
// GetAnalysedControllerStatistics (усредняется по контроллерам - CPU и
// задержка отклика). Добавлено 2026-09-22 по запросу пользователя "вернёмся
// к обработке метрик из СХД... добавь функции в SDK для СХД, при
// потребности" - раньше в пакете loadmon/lenovostorage не было ни одного
// метода, дающего живые KPI, только статический % использования пулов
// (collectLenovoUsedPercent).
func collectLenovoKpi(shd *models.StorageSystems, row *models.StorageMetric) error {
	key := "monitor"
	session, err := lenovostorage.NewSession(shd.IpAddress, true, key, datamap.GetKeyData(key))
	if err != nil {
		return fmt.Errorf("авторизация: %w", err)
	}

	var errs []string

	if vols, verr := session.GetAnalysedVolumeStatistics(); verr == nil {
		var readIOPS, writeIOPS, readBps, writeBps, readLatSum, writeLatSum float64
		var readLatN, writeLatN int
		for _, v := range vols {
			readIOPS += v.ReadIOps
			writeIOPS += v.WriteIOps
			readBps += v.ReadThroughput
			writeBps += v.WriteThroughput
			if v.ReadIOps > 0 { // средняя задержка взвешивается только по томам с реальной нагрузкой - том без чтений/записей не характеризует латентность СХД
				readLatSum += v.ReadResponseTime
				readLatN++
			}
			if v.WriteIOps > 0 {
				writeLatSum += v.WriteResponseTime
				writeLatN++
			}
		}
		row.ReadIOPS = readIOPS
		row.WriteIOPS = writeIOPS
		row.TotalIOPS = readIOPS + writeIOPS
		row.ReadMBps = readBps / (1024 * 1024)
		row.WriteMBps = writeBps / (1024 * 1024)
		if readLatN > 0 {
			row.ReadLatencyMs = readLatSum / float64(readLatN)
		}
		if writeLatN > 0 {
			row.WriteLatencyMs = writeLatSum / float64(writeLatN)
		}
	} else {
		errs = append(errs, "AnalysedVolumeStatistics: "+verr.Error())
	}

	if ctrls, cerr := session.GetAnalysedControllerStatistics(); cerr == nil {
		var cpuSum float64
		var cpuN int
		for _, c := range ctrls {
			cpuSum += c.CpuAvgUtilization
			cpuN++
		}
		if cpuN > 0 {
			row.CPUUtilPercent = cpuSum / float64(cpuN)
		}
	} else {
		errs = append(errs, "AnalysedControllerStatistics: "+cerr.Error())
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
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
