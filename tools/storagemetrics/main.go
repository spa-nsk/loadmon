// Команда storagemetrics — отдельное отладочное приложение loadmon, которое
// делает ТОЛЬКО одну вещь: опрос поминутных метрик производительности СХД
// (cmdb.CollectStorageMetrics/collectStorageMetricForSystem) - и больше
// ничего. Не поднимает gRPC-серверы, не синхронизирует конфигурацию/тома СХД
// (см. отдельную tools/storagesync для этого), не опрашивает vCenter/
// OneView/Zabbix/Jira.
//
// Добавлено 2026-09-22 по запросу пользователя loadmon "сделай отдельным
// приложением по аналогии с storagesync только опросы метрик и лог
// записывай в файл" - после того как в production (metrics_grpc на
// s-nsk01-grpc021) не удавалось однозначно понять по консольному выводу
// (смешанному с логом vCenter/Zabbix/ILO-экспорта в одном потоке), доходит
// ли цикл cmdb.CollectStorageMetrics до конца и что именно происходит с
// записью в storage_metric. См. также knowledge_base/CHANGELOG.md и
// kontext/11-storage-metrics.md.
//
// В отличие от storagesync, лог этой утилиты по умолчанию пишется В ФАЙЛ
// (не в stdout) - именно поэтому она и понадобилась отдельно: длинный
// непрерывный опрос (-interval) удобно смотреть/грепать в отдельном файле,
// не мешая другим процессам/консолям.
//
// Использует ТУ ЖЕ конфигурацию (conf/app2.json - тот же формат и тот же
// механизм поиска файла, что у metrics_grpc/server_web_new/storagesync, см.
// config.configFilePath: переменная окружения LOADMON_CONFIG, иначе
// conf/app2.json рядом с исполняемым файлом, иначе в текущем каталоге) и ТУ
// ЖЕ базу данных - никакого отдельного конфига/схемы не заводится.
//
// Примеры запуска:
//
//	# из каталога metrics_grpc (там уже есть conf/app2.json) - один проход по ВСЕМ СХД,
//	# лог пишется в storagemetrics.log рядом с рабочим каталогом:
//	cd metrics_grpc && go run ../tools/storagemetrics
//
//	# один проход только по одной СХД (быстрая итерация при отладке конкретного вендора):
//	go run ./tools/storagemetrics -only U450-MF-01
//
//	# непрерывный опрос раз в минуту (Ctrl+C для остановки) - имитирует поведение
//	# tickerStorageMetrics в metrics_grpc, но без всего остального и с логом в отдельном файле:
//	go run ./tools/storagemetrics -interval 1m -logfile /tmp/storagemetrics.log
//
//	# явно указать конфиг, если бинарник запускается не рядом с conf/app2.json:
//	LOADMON_CONFIG=/path/to/app2.json go run ./tools/storagemetrics
//
//	# снять сырой каталог доступных метрик Unity (для сверки реальных путей
//	# метрик перед реализацией живых IOPS/throughput/latency/CPU - см.
//	# kontext/11-storage-metrics.md) и выйти, ничего не опрашивая и никуда не записывая:
//	go run ./tools/storagemetrics -dumpmetricsonly U450-MF-01 -dumpmetricsfile u450-mf-01-metrics.json
//
//	# уточнить ID контроллеров Huawei (нужен для CMO_STATISTIC_UUID ниже) -
//	# GET /controller без query-параметров:
//	go run ./tools/storagemetrics -huaweidumpraw D6V6-01 -huaweidumppath /controller -huaweidumpquery "" -huaweidumpfile d6v6-01-controllers.json
//
//	# сохранить сырой ответ Huawei Dorado/OceanStor на текущие KPI контроллера
//	# "0A" (по умолчанию - Read/Write MBps/IOPS + раздельная read/write задержка,
//	# коды из kontext/10-huawei-dorado-rest-api.md) в файл и выйти, ничего не
//	# опрашивая и никуда не записывая:
//	go run ./tools/storagemetrics -huaweidumpraw D6V6-01 -huaweidumpfile d6v6-01-perfstat.json
//
//	# проверить подтверждённый документацией real-time REST-механизм Unity
//	# (metricRealTimeQuery/metricQueryResult, см. kontext/12-unity-rest-api.md)
//	# на примере CPU busy/idle ticks и сохранить сырой результат в файл, затем
//	# выйти, ничего не опрашивая и никуда не записывая:
//	go run ./tools/storagemetrics -unitydumprealtime U450-MF-01 -unitydumpfile u450-mf-01-realtime.json
//
//	# диагностика HTTP 500 ("Unknown internal metrics error") у путей
//	# "kpi.lun.+.sp.+.*" (подтверждено, что даже ОДИН такой путь падает сам
//	# по себе - см. storages.DumpUnityLunRealtimeByName/kontext/12-unity-rest-api.md):
//	# резолвит ID LUN по его ИМЕНИ (не нужно знать ID заранее) и пробует
//	# несколько вариантов подстановки вместо "+" (kpi.lun.<ID>.sp.+.readIOPS,
//	# kpi.lun.<ID>.sp.*.readIOPS, kpi.lun.<ID>.sp.spa.readIOPS):
//	go run ./tools/storagemetrics -unitydumplunname U480-MSK-02 -unitydumplunlun "имя_лана" -unitydumpfile unity-lun-test.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"loadmon/cmdb"
	"loadmon/config"
	"loadmon/models"
	"loadmon/storages"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	only := flag.String("only", "", "ограничить опрос одной СХД по ShortName (без учёта регистра, точное совпадение) - см. cmdb.CollectStorageMetricsOnly")
	interval := flag.Duration("interval", 0, "повторять опрос с этим периодом (например 1m, 6h); 0 (по умолчанию) - один проход и выход")
	logfile := flag.String("logfile", "storagemetrics.log", "путь к файлу лога (создаётся/дополняется); пусто (-logfile=\"\") - лог в stdout, как у storagesync")
	dumpMetricsOnly := flag.String("dumpmetricsonly", "", "не опрашивать метрики - вместо этого снять СЫРОЙ каталог доступных метрик СХД Unity (ShortName) через GET /api/types/metric/instances и сохранить в -dumpmetricsfile, затем выйти (см. kontext/11-storage-metrics.md)")
	dumpMetricsFile := flag.String("dumpmetricsfile", "unity-metric-catalog.json", "путь к файлу для -dumpmetricsonly")
	huaweiDumpRaw := flag.String("huaweidumpraw", "", "не опрашивать метрики - вместо этого выполнить произвольный запрос к REST API Huawei Dorado/OceanStor СХД (ShortName) и сохранить СЫРОЙ ответ в -huaweidumpfile, затем выйти (см. kontext/10-huawei-dorado-rest-api.md/11-storage-metrics.md). Путь задаётся -huaweidumppath, query-параметры - ОТДЕЛЬНО через -huaweidumpquery")
	huaweiDumpMethod := flag.String("huaweidumpmethod", "GET", "HTTP-метод для -huaweidumpraw (GET или POST)")
	// 2026-09-23: раньше был "/performance_statistic" (угаданный, неверный
	// путь - см. историю в CHANGELOG (111)/(112)) - теперь подтверждённый
	// официальной документацией (huawei-api.md) реальный путь для проверки
	// живых KPI (по умолчанию - для КОНТРОЛЛЕРА "0A", уточнить реальный ID
	// можно отдельным вызовом -huaweidumppath /controller без -huaweidumpquery).
	huaweiDumpPath := flag.String("huaweidumppath", "/performace_statistic/cur_statistic_data", "путь запроса (относительно /deviceManager/rest/<id>) для -huaweidumpraw - например /controller (список контроллеров с ID/CPUUSAGE) или /performace_statistic/cur_statistic_data (текущие KPI, требует -huaweidumpquery); опечатка \"performace\" - буквальный путь реального API, не ошибка")
	// 2026-09-23: коды 384/385 (раздельные read/write Avg I/O Response Time)
	// заменили ранее использовавшийся общий код 370 - см. комментарий у
	// huaweiPerfIndicatorReadLatencyUs/WriteLatencyUs в
	// cmdb/collectstoragemetrics.go (найдено в doc-API-hardware/OceanStor
	// Dorado 6.1.6 и V700R001C10 REST Interface Reference-ru.md).
	huaweiDumpQuery := flag.String("huaweidumpquery", "CMO_STATISTIC_UUID=207:0A&CMO_STATISTIC_DATA_ID_LIST=23,25,26,28,384,385", "сырая (уже готовая) query-строка без ведущего \"?\" для -huaweidumpraw - по умолчанию запрашивает Read/Write MBps/IOPS и раздельную read/write задержку (мкс) у контроллера с ID \"0A\"; ID контроллера уточнить через -huaweidumppath /controller -huaweidumpquery \"\"")
	huaweiDumpBody := flag.String("huaweidumpbody", "", "тело запроса (сырой JSON) для -huaweidumpraw при -huaweidumpmethod=POST; пусто - без тела")
	huaweiDumpFile := flag.String("huaweidumpfile", "huawei-raw-dump.json", "путь к файлу для -huaweidumpraw")
	// 2026-09-23: диагностика Unity real-time метрик (POST metricRealTimeQuery
	// + GET metricQueryResult + DELETE) - подтверждённый документацией механизм
	// (см. storages.DumpUnityRealTimeMetrics/kontext/12-unity-rest-api.md),
	// пришедший на смену несуществующему /api/types/kpiValue/instances.
	// Значения по умолчанию - подтверждённый документацией пример путей
	// (CPU busy/idle ticks); для других путей (IOPS/throughput/latency на
	// уровне SP/пула) сначала свериться с -dumpmetricsonly ("не гадать").
	unityDumpRealtime := flag.String("unitydumprealtime", "", "не опрашивать метрики - вместо этого выполнить real-time REST-запрос Unity (ShortName) через metricRealTimeQuery/metricQueryResult и сохранить сырой результат в -unitydumpfile, затем выйти (см. kontext/12-unity-rest-api.md)")
	unityDumpPaths := flag.String("unitydumppaths", "sp.*.cpu.summary.busyTicks,sp.*.cpu.summary.idleTicks", "список путей метрик через запятую для -unitydumprealtime - по умолчанию подтверждённый документацией пример (CPU busy/idle ticks); другие пути сначала проверить через -dumpmetricsonly")
	unityDumpInterval := flag.Uint("unitydumpinterval", 5, "интервал сэмплирования в секундах [5..300] для -unitydumprealtime (аргумент interval у metricRealTimeQuery)")
	unityDumpWait := flag.Duration("unitydumpwait", 6*time.Second, "сколько ждать перед тем, как забрать первый отсчёт, для -unitydumprealtime - должно быть не меньше -unitydumpinterval")
	unityDumpFile := flag.String("unitydumpfile", "unity-realtime-dump.json", "путь к файлу для -unitydumprealtime")
	// 2026-09-24: диагностика HTTP 500 ("Unknown internal metrics error")
	// у путей "kpi.lun.+.sp.+.*" в реальном сборе (collectUnityKpi) -
	// подтверждено, что даже ОДИН такой путь падает сам по себе (см.
	// storages.DumpUnityLunRealtimeByName). У пользователя нет под рукой
	// ID LUN, только имя - эта диагностика резолвит ID по имени сама и
	// пробует несколько вариантов подстановки вместо "+" (см. комментарий
	// у DumpUnityLunRealtimeByName).
	unityDumpLunShort := flag.String("unitydumplunname", "", "не опрашивать метрики - вместо этого резолвнуть ID LUN по имени (-unitydumplunlun) на СХД с этим ShortName и проверить несколько вариантов real-time-пути kpi.lun.<ID>.sp.*.readIOPS (взамен непроверенного kpi.lun.+.sp.+.*), сохранить результат в -unitydumpfile, затем выйти")
	unityDumpLunLun := flag.String("unitydumplunlun", "", "имя LUN (точное, как в Unisphere) для -unitydumplunname")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if *logfile != "" {
		f, err := os.OpenFile(*logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("storagemetrics: не удалось открыть файл лога %q: %v", *logfile, err)
		}
		defer f.Close()
		log.SetOutput(f)
		fmt.Printf("storagemetrics: лог пишется в файл %s\n", *logfile)
	}

	log.Println("storagemetrics: запуск (только опрос метрик СХД, ничего больше)")

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("storagemetrics: не удалось загрузить конфигурацию (conf/app2.json): %v", err)
	}
	if err := config.ConfigData.InitParams(); err != nil {
		// InitParams заодно настраивает список vCenter/OneView - для нашей
		// задачи (только метрики СХД) это не требуется, но вызываем тем же
		// способом, что и metrics_grpc/storagesync, чтобы не разъезжаться в
		// поведении инициализации конфигурации.
		log.Printf("storagemetrics: InitParams: %v", err)
	}

	// Тот же приём, что в metrics_grpc/metrics_grpc.go и tools/storagesync:
	// на время работы снимаем из окружения переменные, чьи ИМЕНА заданы в
	// HttpKey/HttpsKey (используются как ключи для расшифровки паролей через
	// utilities.GetKeyData, а не как HTTP(S)_PROXY) - чтобы сторонние
	// HTTP-клиенты СХД их точно не подхватили случайно, и восстанавливаем по
	// выходу.
	httpKey := config.ConfigData.HttpKey
	httpsKey := config.ConfigData.HttpsKey
	httpEnv := os.Getenv(httpKey)
	httpsEnv := os.Getenv(httpsKey)
	defer os.Setenv(httpKey, httpEnv)
	defer os.Setenv(httpsKey, httpsEnv)
	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	config.DBInit()

	if *dumpMetricsOnly != "" {
		dumpUnityMetricCatalog(*dumpMetricsOnly, *dumpMetricsFile)
		return
	}

	if *huaweiDumpRaw != "" {
		dumpHuaweiRaw(*huaweiDumpRaw, *huaweiDumpMethod, *huaweiDumpPath, *huaweiDumpQuery, *huaweiDumpBody, *huaweiDumpFile)
		return
	}

	if *unityDumpRealtime != "" {
		dumpUnityRealtime(*unityDumpRealtime, *unityDumpPaths, uint32(*unityDumpInterval), *unityDumpWait, *unityDumpFile)
		return
	}

	if *unityDumpLunShort != "" {
		if *unityDumpLunLun == "" {
			log.Fatalf("storagemetrics: -unitydumplunname требует также -unitydumplunlun (имя LUN)")
		}
		dumpUnityLunRealtime(*unityDumpLunShort, *unityDumpLunLun, uint32(*unityDumpInterval), *unityDumpWait, *unityDumpFile)
		return
	}

	if *only != "" {
		cmdb.CollectStorageMetricsOnly = *only
		log.Printf("storagemetrics: фильтр по СХД включён, ShortName=%q", *only)
	}

	runOnce := func() {
		started := time.Now()
		log.Println("storagemetrics: === начало цикла опроса метрик СХД ===")
		cmdb.CollectStorageMetrics()
		log.Printf("storagemetrics: === цикл завершён за %s ===", time.Since(started))
	}

	runOnce()

	if *interval <= 0 {
		log.Println("storagemetrics: -interval не задан, завершение после одного прохода")
		return
	}

	log.Printf("storagemetrics: переход в режим повторения каждые %s (Ctrl+C для остановки)", *interval)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			runOnce()
		case <-sigs:
			log.Println("storagemetrics: получен сигнал завершения, выход")
			return
		}
	}
}

// dumpUnityMetricCatalog ищет СХД по ShortName (без учёта регистра, как
// -only) в справочнике storage_systems, снимает через
// storages.DumpUnityMetricCatalog сырой каталог доступных Unity-метрик (GET
// /api/types/metric/instances) и сохраняет его (с отступами, для удобства
// чтения) в файл path. Ничего не опрашивает и не пишет в storage_metric -
// только диагностика. См. комментарий у case 2 в
// cmdb/collectstoragemetrics.go: цель - получить ДОСТОВЕРНЫЙ список путей
// метрик конкретного массива перед тем, как реализовывать живые IOPS/
// throughput/latency/CPU для Unity (взамен убранного опроса через
// несуществующий /api/types/kpiValue/instances).
func dumpUnityMetricCatalog(shortName, path string) {
	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).All(&systems); err != nil {
		log.Fatalf("storagemetrics: -dumpmetricsonly: ошибка чтения storage_systems: %v", err)
	}
	var shd *models.StorageSystems
	for _, s := range systems {
		if s != nil && strings.EqualFold(s.ShortName, shortName) {
			shd = s
			break
		}
	}
	if shd == nil {
		log.Fatalf("storagemetrics: -dumpmetricsonly: СХД с ShortName=%q не найдена в storage_systems", shortName)
	}
	if shd.ApiType != 2 {
		log.Fatalf("storagemetrics: -dumpmetricsonly: СХД %q имеет ApiType=%d, а не 2 (Unity) - каталог метрик /api/types/metric/instances специфичен для Unity REST API", shortName, shd.ApiType)
	}

	log.Printf("storagemetrics: -dumpmetricsonly: запрашиваю каталог метрик у %s (%s)...", shd.ShortName, shd.IpAddress)
	raw, err := storages.DumpUnityMetricCatalog(shd.IpAddress)
	if err != nil {
		log.Fatalf("storagemetrics: -dumpmetricsonly: ошибка запроса каталога метрик у %s (%s): %v", shd.ShortName, shd.IpAddress, err)
	}

	var pretty []byte
	if pretty, err = json.MarshalIndent(json.RawMessage(raw), "", "  "); err != nil {
		// На случай, если пришедший ответ не удалось переформатировать -
		// сохраняем как есть (сырьём), чтобы данные не потерялись.
		pretty = raw
	}
	if err := os.WriteFile(path, pretty, 0644); err != nil {
		log.Fatalf("storagemetrics: -dumpmetricsonly: не удалось записать файл %q: %v", path, err)
	}
	fmt.Printf("storagemetrics: каталог метрик %s (%s) сохранён в %s (%d байт)\n", shd.ShortName, shd.IpAddress, path, len(pretty))
	log.Printf("storagemetrics: -dumpmetricsonly: каталог метрик %s (%s) сохранён в %s (%d байт)", shd.ShortName, shd.IpAddress, path, len(pretty))
}

// dumpHuaweiRaw ищет СХД по ShortName (без учёта регистра, как -only) в
// справочнике storage_systems, проверяет, что это действительно Huawei
// (ApiType 3 - Dorado, 4 - OceanStor), и выполняет через
// cmdb.DumpHuaweiRawRequest произвольный запрос method/path (с телом body,
// если задано) к REST API устройства, сохраняя СЫРОЙ ответ в файл path.
// Ничего не опрашивает и не пишет в storage_metric - только диагностика.
// Добавлено 2026-09-22 по запросу пользователя loadmon "сделай сохранение
// сырых запросов по huawei" - см. комментарий у cmdb.DumpHuaweiRawRequest и
// kontext/10-huawei-dorado-rest-api.md/11-storage-metrics.md. 2026-09-23:
// добавлен отдельный параметр rawQuery (query-строка) - раньше её нужно
// было передавать прямо в -huaweidumppath, что было скрытым багом
// (newRequest не распознаёт "?" внутри пути, см. dorado.RawRequestDump).
//
// ⚠️ В отличие от -dumpmetricsonly (Unity, безопасный read-only GET каталога),
// -huaweidumpmethod POST может создавать состояние на СХД (например, подписку
// на объекты статистики, если путь на это рассчитан) - перед использованием
// POST на боевом массиве стоит свериться с официальной документацией Huawei
// на предмет побочных эффектов конкретного эндпоинта.
func dumpHuaweiRaw(shortName, method, reqPath, rawQuery, bodyStr, path string) {
	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).All(&systems); err != nil {
		log.Fatalf("storagemetrics: -huaweidumpraw: ошибка чтения storage_systems: %v", err)
	}
	var shd *models.StorageSystems
	for _, s := range systems {
		if s != nil && strings.EqualFold(s.ShortName, shortName) {
			shd = s
			break
		}
	}
	if shd == nil {
		log.Fatalf("storagemetrics: -huaweidumpraw: СХД с ShortName=%q не найдена в storage_systems", shortName)
	}
	if shd.ApiType != 3 && shd.ApiType != 4 {
		log.Fatalf("storagemetrics: -huaweidumpraw: СХД %q имеет ApiType=%d, а не 3/4 (Huawei Dorado/OceanStor)", shortName, shd.ApiType)
	}

	var body []byte
	if bodyStr != "" {
		body = []byte(bodyStr)
	}

	logPath := reqPath
	if rawQuery != "" {
		logPath += "?" + rawQuery
	}
	log.Printf("storagemetrics: -huaweidumpraw: %s %s у %s (%s)...", method, logPath, shd.ShortName, shd.IpAddress)
	if err := cmdb.DumpHuaweiRawRequest(shd, method, reqPath, rawQuery, body, path); err != nil {
		// Сырой ответ (если СХД вообще успела его прислать) decodeBodyDump
		// в go-dorado-sdk сохраняет ДО попытки разбора - см. комментарий
		// cmdb.DumpHuaweiRawRequest/dorado.Device.RawRequestDump - поэтому
		// даже при ошибке ниже стоит проверить файл path, он может быть
		// не пустым.
		log.Printf("storagemetrics: -huaweidumpraw: запрос завершился с ошибкой (проверьте файл %s - сырой ответ, если получен, сохранён туда независимо от этой ошибки): %v", path, err)
		fmt.Printf("storagemetrics: запрос завершился с ошибкой (см. лог); проверьте файл %s - сырой ответ мог всё равно сохраниться\n", path)
		return
	}
	fmt.Printf("storagemetrics: сырой ответ %s %s от %s (%s) сохранён в %s\n", method, logPath, shd.ShortName, shd.IpAddress, path)
	log.Printf("storagemetrics: -huaweidumpraw: сырой ответ сохранён в %s", path)
}

// dumpUnityRealtime ищет СХД по ShortName (без учёта регистра, как -only) в
// справочнике storage_systems, проверяет, что это Unity (ApiType 2), и
// выполняет через storages.DumpUnityRealTimeMetrics подтверждённый
// документацией цикл real-time-опроса (POST metricRealTimeQuery + GET
// metricQueryResult + DELETE) для заданных путей метрик, сохраняя сырой
// результат в файл. Ничего не опрашивает штатно и не пишет в storage_metric -
// только диагностика. Добавлено 2026-09-23 по запросу пользователя loadmon
// "добавил свежие описания по работе с СХД Huawei и Dell EMC... используй для
// доработок SDK" - см. комментарий у storages.DumpUnityRealTimeMetrics и
// kontext/12-unity-rest-api.md.
func dumpUnityRealtime(shortName, pathsCSV string, interval uint32, wait time.Duration, path string) {
	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).All(&systems); err != nil {
		log.Fatalf("storagemetrics: -unitydumprealtime: ошибка чтения storage_systems: %v", err)
	}
	var shd *models.StorageSystems
	for _, s := range systems {
		if s != nil && strings.EqualFold(s.ShortName, shortName) {
			shd = s
			break
		}
	}
	if shd == nil {
		log.Fatalf("storagemetrics: -unitydumprealtime: СХД с ShortName=%q не найдена в storage_systems", shortName)
	}
	if shd.ApiType != 2 {
		log.Fatalf("storagemetrics: -unitydumprealtime: СХД %q имеет ApiType=%d, а не 2 (Unity) - metricRealTimeQuery специфичен для Unity REST API", shortName, shd.ApiType)
	}

	paths := strings.Split(pathsCSV, ",")
	for i := range paths {
		paths[i] = strings.TrimSpace(paths[i])
	}

	log.Printf("storagemetrics: -unitydumprealtime: %v (interval=%ds, wait=%s) у %s (%s)...", paths, interval, wait, shd.ShortName, shd.IpAddress)
	result, err := storages.DumpUnityRealTimeMetrics(shd.IpAddress, paths, interval, wait)
	if err != nil {
		log.Fatalf("storagemetrics: -unitydumprealtime: ошибка запроса к %s (%s): %v", shd.ShortName, shd.IpAddress, err)
	}

	pretty, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("storagemetrics: -unitydumprealtime: ошибка сериализации результата: %v", err)
	}
	if err := os.WriteFile(path, pretty, 0644); err != nil {
		log.Fatalf("storagemetrics: -unitydumprealtime: не удалось записать файл %q: %v", path, err)
	}
	fmt.Printf("storagemetrics: real-time результат %s (%s) сохранён в %s (%d байт, %d записей)\n", shd.ShortName, shd.IpAddress, path, len(pretty), len(result.Entries))
	log.Printf("storagemetrics: -unitydumprealtime: результат сохранён в %s (%d записей)", path, len(result.Entries))
}

// dumpUnityLunRealtime ищет СХД по ShortName (без учёта регистра) в
// storage_systems, проверяет, что это Unity (ApiType 2), и вызывает
// storages.DumpUnityLunRealtimeByName - диагностику причины HTTP 500 у
// путей "kpi.lun.+.sp.+.*" (подтверждено, что даже ОДИН такой путь падает
// сам по себе - см. подробный комментарий у DumpUnityLunRealtimeByName).
// Добавлено 2026-09-24 по запросу пользователя loadmon "у меня нет LUN ID
// есть имя LUN сделай опрос сам" - резолвит ID LUN по имени сама (через
// storages.ResolveUnityLunID) и пробует несколько вариантов подстановки
// вместо "+" в одном показателе (readIOPS), сохраняя результат/ошибку по
// каждому варианту в файл. Ничего не опрашивает штатно и не пишет в
// storage_metric - только диагностика.
func dumpUnityLunRealtime(shortName, lunName string, interval uint32, wait time.Duration, path string) {
	o := orm.NewOrm()
	var systems []*models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).All(&systems); err != nil {
		log.Fatalf("storagemetrics: -unitydumplunname: ошибка чтения storage_systems: %v", err)
	}
	var shd *models.StorageSystems
	for _, s := range systems {
		if s != nil && strings.EqualFold(s.ShortName, shortName) {
			shd = s
			break
		}
	}
	if shd == nil {
		log.Fatalf("storagemetrics: -unitydumplunname: СХД с ShortName=%q не найдена в storage_systems", shortName)
	}
	if shd.ApiType != 2 {
		log.Fatalf("storagemetrics: -unitydumplunname: СХД %q имеет ApiType=%d, а не 2 (Unity)", shortName, shd.ApiType)
	}

	log.Printf("storagemetrics: -unitydumplunname: резолвлю LUN %q и проверяю варианты real-time-пути у %s (%s)...", lunName, shd.ShortName, shd.IpAddress)
	result, err := storages.DumpUnityLunRealtimeByName(shd.IpAddress, lunName, interval, wait)
	if err != nil {
		log.Fatalf("storagemetrics: -unitydumplunname: ошибка у %s (%s): %v", shd.ShortName, shd.IpAddress, err)
	}

	pretty, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("storagemetrics: -unitydumplunname: ошибка сериализации результата: %v", err)
	}
	if err := os.WriteFile(path, pretty, 0644); err != nil {
		log.Fatalf("storagemetrics: -unitydumplunname: не удалось записать файл %q: %v", path, err)
	}
	fmt.Printf("storagemetrics: результат диагностики LUN %q на %s (%s) сохранён в %s (%d байт) - смотрите поля id_plus_sp/id_star_sp/id_literal_spa\n", lunName, shd.ShortName, shd.IpAddress, path, len(pretty))
	log.Printf("storagemetrics: -unitydumplunname: результат сохранён в %s", path)
}
