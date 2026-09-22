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
