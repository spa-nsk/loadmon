// Команда storagesync — отдельное отладочное приложение loadmon, которое
// делает ТОЛЬКО одну вещь: синхронизацию данных СХД (тома + полки/диски/
// пулы через cmdb.UpdateVolumeStorage/cmdb.SyncStorageHardwareForSystem) - и
// больше ничего. Не поднимает gRPC-серверы, не опрашивает vCenter/OneView/
// Zabbix/Jira, не считает и не сохраняет метрики производительности.
//
// Добавлено 2026-09-17 по прямому запросу пользователя loadmon "сделай
// отдельное приложение с использованием конфигурации, которое будет делать
// синхронизацию данных СХД и больше ничего. На нём и отладим код" - после
// того как в реальном metrics_grpc обнаружились массовые ошибки опроса СХД
// (HTTP 422 у Unity, "Error start command" у VNX - см. запись в
// knowledge_base/CHANGELOG.md за 2026-09-17), а полный цикл metrics_grpc
// слишком долгий и шумный (опрашивает вообще всё) для быстрой итеративной
// отладки конкретно СХД-кода.
//
// Использует ТУ ЖЕ конфигурацию (conf/app2.json - тот же формат и тот же
// механизм поиска файла, что у metrics_grpc/server_web_new, см.
// config.configFilePath: переменная окружения LOADMON_CONFIG, иначе
// conf/app2.json рядом с исполняемым файлом, иначе в текущем каталоге) и ТУ
// ЖЕ базу данных - никакого отдельного конфига/схемы не заводится, чтобы
// отлаживать код на реальных данных о реальных СХД без риска разъехаться с
// production-конфигурацией.
//
// Примеры запуска:
//
//	# из каталога metrics_grpc (там уже есть conf/app2.json) - один проход по ВСЕМ СХД:
//	cd metrics_grpc && go run ../tools/storagesync
//
//	# один проход только по одной СХД (быстрая итерация при отладке конкретного вендора):
//	go run ./tools/storagesync -only U300-02
//
//	# непрерывный опрос раз в минуту (Ctrl+C для остановки) - имитирует поведение
//	# ticker8 в metrics_grpc, но без всего остального:
//	go run ./tools/storagesync -interval 1m
//
//	# явно указать конфиг, если бинарник запускается не рядом с conf/app2.json:
//	LOADMON_CONFIG=/path/to/app2.json go run ./tools/storagesync
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loadmon/cmdb"
	"loadmon/config"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	only := flag.String("only", "", "ограничить синхронизацию одной СХД по ShortName (без учёта регистра, точное совпадение) - см. cmdb.UpdateVolumeStorageOnly")
	interval := flag.Duration("interval", 0, "повторять синхронизацию с этим периодом (например 1m, 6h); 0 (по умолчанию) - один проход и выход")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("storagesync: запуск (только синхронизация СХД, ничего больше)")

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("storagesync: не удалось загрузить конфигурацию (conf/app2.json): %v", err)
	}
	if err := config.ConfigData.InitParams(); err != nil {
		// InitParams заодно настраивает список vCenter/OneView - для нашей
		// задачи (только СХД) это не требуется, но вызываем тем же способом,
		// что и metrics_grpc/server_web_new, чтобы не разъезжаться в
		// поведении инициализации конфигурации (например, тем же образом
		// применяются пароли из хранилища ключей для тех полей, что нужны).
		log.Printf("storagesync: InitParams: %v", err)
	}

	// Тот же приём, что в metrics_grpc/metrics_grpc.go: на время работы
	// снимаем из окружения переменные, чьи ИМЕНА заданы в HttpKey/HttpsKey
	// (используются как ключи для расшифровки паролей через
	// utilities.GetKeyData, а не как HTTP(S)_PROXY) - чтобы сторонние
	// библиотеки (HTTP-клиенты СХД в т.ч.) их точно не подхватили случайно,
	// и восстанавливаем по выходу.
	httpKey := config.ConfigData.HttpKey
	httpsKey := config.ConfigData.HttpsKey
	httpEnv := os.Getenv(httpKey)
	httpsEnv := os.Getenv(httpsKey)
	defer os.Setenv(httpKey, httpEnv)
	defer os.Setenv(httpsKey, httpsEnv)
	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	config.DBInit()

	if *only != "" {
		cmdb.UpdateVolumeStorageOnly = *only
		log.Printf("storagesync: фильтр по СХД включён, ShortName=%q", *only)
	}

	runOnce := func() {
		started := time.Now()
		log.Println("storagesync: === начало цикла синхронизации СХД ===")
		cmdb.UpdateVolumeStorage()
		log.Printf("storagesync: === цикл завершён за %s ===", time.Since(started))
	}

	runOnce()

	if *interval <= 0 {
		log.Println("storagesync: -interval не задан, завершение после одного прохода")
		return
	}

	log.Printf("storagesync: переход в режим повторения каждые %s (Ctrl+C для остановки)", *interval)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			runOnce()
		case <-sigs:
			log.Println("storagesync: получен сигнал завершения, выход")
			return
		}
	}
}
