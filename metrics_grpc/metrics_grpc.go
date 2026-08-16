// Package main (metrics_grpc) — исполняемый файл-демон системы мониторинга
// loadmon. Инициализирует конфигурацию и БД, поднимает gRPC-серверы пакета
// grpcserver (сервисы Loadmon и LoadmonDB, отдающие собранные данные), а
// также клиента grpcclient для запросов к вышестоящим/соседним узлам,
// запускает периодический опрос инфраструктуры (VMware vSphere через
// vspheremetrics, оборудование через oneview, CMDB через cmdb, itsystem,
// zabbix) и HTTP-эндпоинт с метриками Prometheus. Это точка сборки и запуска
// всей серверной части проекта.
package main

import (
	"fmt"
	"loadmon/cmdb"
	"loadmon/config"
	"loadmon/grpcclient"
	"loadmon/grpcserver"
	"loadmon/itsystem"
	_ "loadmon/routers"
	"loadmon/vspheremetrics"
	"loadmon/zabbix"

	"log"
	"os"
	"os/signal"

	"net/http"
	"runtime"
	"sync"
	"syscall"

	"time"

	oneview "github.com/spa-nsk/oneview2"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// var password string = ""
var baseLock sync.RWMutex
var baseLocked bool = false

// IsBaseLocked возвращает true, если БД в данный момент заблокирована для
// обновления (см. LockBase/UnlockBase).
func IsBaseLocked() bool {
	baseLock.RLock()
	defer baseLock.RUnlock()
	return baseLocked
}

// LockBase устанавливает флаг блокировки БД на время обновления данных,
// чтобы предотвратить конкурентный доступ.
func LockBase() {
	baseLock.Lock()
	defer baseLock.Unlock()
	baseLocked = true
}

// UnlockBase снимает флаг блокировки БД, установленный LockBase.
func UnlockBase() {
	baseLock.Lock()
	defer baseLock.Unlock()
	baseLocked = false
}

func init() {
	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Printf("config: %v", err)
	}
	if err := config.ConfigData.InitParams(); err != nil {
		log.Printf("InitParams: %v", err)
	}
}

func main() {

	config.DBInit()
	/*
		if len(os.Args) > 1 {
			if os.Args[1] == "convert" {
				fmt.Println("Конвертация данных начата")
				cmdb.ConvertDatabase("default", "default-add")
				fmt.Println("Конвертация данных завершена")
			}
		}
	*/
	//c := make(chan string)
	//shedulerExit := make(chan bool)

	httpKey := config.ConfigData.HttpKey
	httpsKey := config.ConfigData.HttpsKey
	httpEnv := os.Getenv(httpKey)
	defer os.Setenv(httpKey, httpEnv)

	httpsEnv := os.Getenv(httpsKey)
	defer os.Setenv(httpsKey, httpsEnv)

	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	s1 := os.Getenv(httpKey)
	s2 := os.Getenv(httpKey)

	_ = s1
	_ = s2

	ov := oneview.GetGlobalOVInfrastructure()
	log.Println("Загрузка данных по конфигурациям серверов")
	srv, err := ov.LoadServerHardwareList()
	if err != nil {
		log.Println("Ошибка загрузки данных по конфигурациям серверов: ", err)
	} else {
		log.Println("Загрузка данных по конфигурациям серверов завершена. Всего серверов: ", len(srv))
	}

	//	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("Используется : ", runtime.NumCPU(), " CPU", "Количество Goroutine: ", runtime.NumGoroutine()) //количество потоков на старте

	fmt.Println("Запуск в режиме: ", config.ConfigData.WorkMode)

	os.Setenv("GOMEMLIMIT", "6000000000")

	now := time.Now().Local()
	FirstTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local) //!!!!!!!!!!!!!!!!!! исправить на 23
	//FirstTime := time.Date(now.Year(), now.Month(), now.Day(), 13, 00, 0, 0, time.Local) //!!!!!!!!!! отладка

	//  FirstTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 55, 00, 0, time.Local) //!!!!!!!!!!!!!!!!! исправить на 23

	timewait := FirstTime.Sub(now)
	cleartime := timewait + time.Hour*3
	log.Println("Запуск сохранения кластеров через: ", timewait.String())
	timer := time.NewTimer(timewait)

	log.Println("Запуск очистки метрик через: ", cleartime.String())
	timer2 := time.NewTimer(cleartime)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Minute * time.Duration(60))
	defer ticker.Stop()

	//ticker2 := time.NewTicker(time.Minute * time.Duration(30))
	//defer ticker2.Stop()

	ticker4 := &time.Ticker{}
	defer ticker4.Stop() // сделано специально несмотря на недопустимое значение в текущий момент

	ticker9 := &time.Ticker{}
	defer ticker9.Stop() // сделано специально несмотря на недопустимое значение в текущий момент

	ticker8 := time.NewTicker(time.Hour * time.Duration(6)) //
	defer ticker8.Stop()

	ticker10 := time.NewTicker(time.Second * time.Duration(20)) //

	tickerZabbix := time.NewTicker(time.Minute * time.Duration(10))
	defer tickerZabbix.Stop()

	cmdb.ClearOldMetric()
	zabbix.Get10MinutesHistory()

	go func() {
		for {
			select {
			case <-sigs:
				//              ticker2.Stop()
				ticker4.Stop()
				ticker8.Stop()
				ticker9.Stop()
				ticker10.Stop()
				vspheremetrics.GetGlobalVMInfrastructure().SetDoneVcenter()
				//c <- "exit" // завершить обработчик сохранения кластеров
				log.Println("Получен сигнал на завершение loadmon")
				os.Exit(0)
			case <-ticker10.C:
				vspheremetrics.SetMetric()

			case <-ticker8.C:
				go func() {
					workStart := time.Now().Local()
					log.Println(workStart.Format("2006-01-02 15:04:05"), "Обновление данных конфигурации")
					config.ConfigData.LoadConfigData()
					config.ConfigData.InitParams()
					log.Println(workStart.Format("2006-01-02 15:04:05"), "Обновление данных конфигурации завершено")
					ov := oneview.GetGlobalOVInfrastructure()
					log.Println("Загрузка данных по конфигурациям серверов")
					ov.LoadServerHardwareList()
					log.Println(workStart.Format("2006-01-02 15:04:05"), "Обновление конфигураций серверов")
					cmdb.UpdateHwSprHP()
					log.Println(workStart.Format("2006-01-02 15:04:05"), "Обновление конфигураций серверов завершено")

					cmdb.UpdateVolumeStorage() //убираем до лучших времен. Нужно переопределить список СХД |||| вернул но надо проверить и дорабатывать
				}()

			case <-ticker4.C:
				//c <- "start" // запустить сохранение кластеров
				go func() {
					cmdb.ClearStatusVolumeStorage()
					cmdb.SaveMetricData()
				}()

			case <-timer.C:
				//c <- "start"                             // запустить сохранение кластеров
				ticker4 = time.NewTicker(time.Hour * time.Duration(24)) //инициировать запуск каждые 24 часа
				go func() {
					cmdb.ClearStatusVolumeStorage()
					cmdb.SaveMetricData()
				}()

			case <-ticker9.C:
				go func() {
					cmdb.ClearMetricData() // запустить очистку метрик
					cmdb.ClearOldMetric()  // и из базы
				}()

			case <-timer2.C:
				ticker9 = time.NewTicker(time.Hour * time.Duration(24)) //инициировать запуск каждые 24 часа
				go func() {
					cmdb.ClearMetricData() // запустить очистку метрик
					cmdb.ClearOldMetric()  // и из базы
				}()

			case <-tickerZabbix.C:
				zabbix.Get10MinutesHistory()

			case <-ticker.C:
				go func() {
					log.Println("START")
					log.Println("itsystem.LoadITSystemSpr() start")
					itsystem.LoadITSystemSpr()
					t := time.Now().Local()
					log.Println("cmdb.UpdateClusterSprRecords()")
					cmdb.UpdateClusterSprRecordsV2()
					cmdb.UpdatePowerStateForServers()
					if t.Hour() >= 4 && t.Hour() < 10 {
						//cmdb.ReportDownload()
						//cmdb.ReportImport()
					}
					log.Println("END")
				}()
			}
		}
	}()

	if config.ConfigData.WorkMode == "api" {
		go grpcserver.StartServerAddr(config.ConfigData.GRPCSrvApiAddr, config.ConfigData.GRPCSrvApiPort) //grpc api
		go grpcserver.StartDBServerAddr(config.ConfigData.GRPCSrvDBAddr, config.ConfigData.GRPCSrvDBPort) //grpc db

		go func() {
			itsystem.LoadITSystemSpr()
			log.Println("Обновление конфигураций серверов")
			cmdb.UpdateHwSprHP()
			log.Println("Обновление конфигураций серверов завершено")
			//cmdb.UpdateSprVM() //проверить по новой логике
			//cmdb.UpdateHWHostInfo() //проверить по новой логике
			//zabbix.ZabbixUpdate() //перенести на новые серверы
			cmdb.UpdateVolumeStorage() // вернул но надо проверить и дорабатывать
		}()

		if config.ConfigData.GRPCSwitchAddr != "" {
			if err := grpcclient.RegisterWorkProcess(&config.ConfigData); err != nil {
				fmt.Println("Регистрация обработчика в GRPCSwitch:", config.ConfigData.GetSwitchAddr(), "завершилась ошибкой: ", err)
				http.Handle("/metrics", promhttp.Handler())
				log.Println("Запущен http сервер для prometheus port 9090")
				http.ListenAndServe(":9090", nil)
			} else {
				fmt.Println("Успешная регистрация обработчика в GRPCSwitch:", config.ConfigData.GetSwitchAddr())
				for {
					time.Sleep(5 * time.Second)
					//runtime.Gosched()
				}
			}
		} else {
			http.Handle("/metrics", promhttp.Handler())
			log.Println("Запущен http сервер для prometheus port 9090")
			http.ListenAndServe(":9090", nil)
		}
	} else {
		fmt.Println("Попытка запуска api сервиса из папки с конфигурацией: ", config.ConfigData.WorkMode)
	}
}

/*
func saveClustersShedule(C chan string, shedulerExit chan bool) {
	for {
		switch <-C {
		case "start":
			cmdb.ClearStatusVolumeStorage()
			cmdb.SaveMetricData()
		case "exit":
			shedulerExit <- true
			os.Exit(3)
			return
		}
	}
}
*/
