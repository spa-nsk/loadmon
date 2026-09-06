package config

import (
	"fmt"
	"loadmon/models"
	"loadmon/utilities"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	// sqlite3-драйвер регистрируется здесь один раз для всех бинарников, использующих config.DBInit()
	_ "github.com/mattn/go-sqlite3"
)

// sqliteDataSource строит DSN для файловой базы sqlite3 на диске.
// Путь берётся из app2.json (DBFile / DBFileAdd); относительный путь разрешается от каталога conf.
func sqliteDataSource(dbFile string) string {
	if dbFile == "" {
		log.Fatal("sqlite: не задан путь к базе данных (DBFile / DBFileAdd в conf/app2.json)")
	}
	if strings.HasPrefix(dbFile, ":memory:") {
		log.Fatal("sqlite: :memory: не поддерживается, укажите путь к файлу в conf/app2.json")
	}
	if !filepath.IsAbs(dbFile) {
		dbFile = filepath.Join(confDir(), dbFile)
	}
	absPath, err := filepath.Abs(dbFile)
	if err != nil {
		log.Println("sqlite: не удалось получить абсолютный путь:", err)
		absPath = dbFile
	}
	if dir := filepath.Dir(absPath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Println("Ошибка создания каталога для базы sqlite:", dir, err)
		}
	}
	log.Println("sqlite: файл базы данных:", absPath)
	// WAL + sync: данные пишутся на диск, а не держатся только в памяти процесса.
	// _busy_timeout=5000 - критично для sqlite при нескольких горутинах,
	// пишущих в одну и ту же базу (см. metrics_grpc/metrics_grpc.go - там
	// несколько независимых time.Ticker'ов запускают запись в БД
	// одновременно, включая цикл обновления ServerHardware из
	// cmdb/updatehwsprhp.go): без этого параметра драйвер sqlite3 сразу
	// возвращает "database is locked" (SQLITE_BUSY) при столкновении двух
	// писателей, вместо того чтобы подождать, пока занятое соединение
	// освободится. См. также orm.SetMaxOpenConns(alias, 1) в DBInit() ниже -
	// вместе с busy_timeout это сводит конкурентные попытки записи к
	// последовательному ожиданию вместо ошибки.
	return absPath + "?_journal_mode=WAL&_sync=NORMAL&cache=shared&_busy_timeout=5000"
}

// logSyncdbErr выводит в лог результат orm.RunSyncdb, приглушая одну
// конкретную и совершенно безобидную ошибку sqlite - "index ... already
// exists"/"table ... already exists". orm.RunSyncdb(force=false) должен
// создавать только отсутствующие таблицы/индексы, но генератор DDL beego
// для sqlite не добавляет "IF NOT EXISTS" в CREATE INDEX (см.
// beego/v2/client/orm/ddl.go) - поэтому при КАЖДОМ перезапуске приложения,
// начиная со второго, RunSyncdb пытается создать уже существующие индексы
// заново и sqlite закономерно отвечает "already exists". Раньше это молча
// логировалось как обычная ошибка (`log.Println(err)`), что выглядело как
// поломка синхронизации схемы при каждом старте, хотя на самом деле это
// ожидаемый шум - таблицы и данные не страдают, RunSyncdb просто
// останавливается на первой такой "ошибке" и не проверяет остальные
// модели, но т.к. они уже были созданы в предыдущие разы, это не критично.
// Если сообщение НЕ про "already exists" (значит, реальная проблема со
// схемой БД) - оно по-прежнему выводится в лог как есть.
func logSyncdbErr(alias string, err error) {
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "already exists") {
		log.Printf("sqlite (%s): схема БД уже синхронизирована (таблица/индекс уже существует, это ожидаемо при повторном запуске): %v", alias, err)
		return
	}
	log.Printf("RunSyncdb(%q, false, false): %v", alias, err)
}

// buildDataSource формирует строку подключения (DSN) в зависимости от драйвера БД.
func buildDataSource(driverName, userName, password, dbName, dbFile, hostName string, port int, sslMode string) (alias, driver, dataSource string) {
	switch driverName {
	case "sqlite3":
		return ConfigData.AliasName, "sqlite3", sqliteDataSource(dbFile)
	case "":
		// драйвер не указан в конфигурации - используем резервное подключение к postgres по умолчанию
		return "default", "postgres", fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s",
			"work", utilities.GetKeyData("work"), "loadmonv3", "172.18.112.125", 5432, "disable")
	default:
		return ConfigData.AliasName, driverName, fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s",
			userName, password, dbName, hostName, port, sslMode)
	}
}

// DBInit инициализирует ORM beego: регистрирует драйверы БД (postgres, sqlite3),
// формирует строки подключения для основной базы ("default") и дополнительной
// ("default-add") по данным ConfigData, регистрирует все модели пакета models
// и выполняет синхронизацию схемы (RunSyncdb) для обеих баз.
func DBInit() {
	if ConfigData.DriverName == "" {
		log.Fatal("config.ConfigData.DriverName == \"\"")
	}

	orm.ResetModelCache()
	orm.DefaultTimeLoc = time.Local
	orm.RegisterDriver("postgres", orm.DRPostgres)
	orm.RegisterDriver("sqlite3", orm.DRSqlite)

	alias, driver, dataSource := buildDataSource(ConfigData.DriverName, ConfigData.UserName, ConfigData.Password,
		ConfigData.DBName, ConfigData.DBFile, ConfigData.HostName, ConfigData.Port, ConfigData.SSLmode)
	orm.RegisterDataBase(alias, driver, dataSource)

	// вторая база данных (default-add): для sqlite3 используем отдельный файл рядом с основным,
	// для остальных драйверов поведение сохранено прежним (отдельный postgres-сервер)
	if ConfigData.DriverName == "sqlite3" {
		dbFileAdd := ConfigData.DBFileAdd
		if dbFileAdd == "" {
			ext := filepath.Ext(ConfigData.DBFile)
			base := ConfigData.DBFile[:len(ConfigData.DBFile)-len(ext)]
			if base == "" {
				base = "loadmon"
			}
			dbFileAdd = base + "-add" + ext
		}
		orm.RegisterDataBase("default-add", "sqlite3", sqliteDataSource(dbFileAdd))

		// sqlite поддерживает только ОДНОГО писателя одновременно (даже в
		// режиме WAL) - пул из нескольких открытых на запись соединений не
		// ускоряет запись, а только увеличивает число конкурентных
		// попыток захватить файловую блокировку, что и приводило к
		// "database is locked" при одновременной работе нескольких
		// горутин из metrics_grpc/metrics_grpc.go. Ограничение до 1
		// открытого соединения на алиас заставляет database/sql
		// сериализовать запросы на уровне пула соединений Go, а не
		// полагаться только на busy_timeout драйвера.
		orm.SetMaxOpenConns(alias, 1)
		orm.SetMaxIdleConns(alias, 1)
		orm.SetMaxOpenConns("default-add", 1)
		orm.SetMaxIdleConns("default-add", 1)
	} else {
		dataSourceAdd := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s", "work", utilities.GetKeyData("work"), "loadmonv3", "172.18.112.125", 5432, "disable")
		orm.RegisterDataBase("default-add", "postgres", dataSourceAdd)
	}

	orm.Debug = ConfigData.ORMdebug

	orm.RegisterModel(new(models.ReportServerRecord))
	orm.RegisterModel(new(models.AuthUser))

	orm.RegisterModel(new(models.ServerSprRecords))
	orm.RegisterModel(new(models.ServerSprRecordsArch))

	orm.RegisterModel(new(models.SprDiskInfo))
	orm.RegisterModel(new(models.SprCFOAllocation))
	orm.RegisterModel(new(models.SprBusinesAllocation))
	orm.RegisterModel(new(models.BladeSystems))
	orm.RegisterModel(new(models.StorageSystems))
	orm.RegisterModel(new(models.Datastore))
	orm.RegisterModel(new(models.ServerHardware))
	orm.RegisterModel(new(models.ServerHardwareComponents))

	orm.RegisterModel(new(models.UserLog))

	orm.RegisterModel(new(models.VMRecord))
	orm.RegisterModel(new(models.VMSprRecord))

	orm.RegisterModel(new(models.VMGuestDiskInfo))
	orm.RegisterModel(new(models.VMRecordGuestDiskInfo))
	orm.RegisterModel(new(models.VMPerfRecord))
	orm.RegisterModel(new(models.HostRecord))
	orm.RegisterModel(new(models.HostPerf))
	orm.RegisterModel(new(models.VcenterSprRecords))
	orm.RegisterModel(new(models.DatacenterSprRecords))
	orm.RegisterModel(new(models.ClusterSprRecords))
	orm.RegisterModel(new(models.ClusterRecord))
	orm.RegisterModel(new(models.ClusterPerf))

	orm.RegisterModel(new(models.VMRecordUsageOnDatastore))
	orm.RegisterModel(new(models.SANSystem))
	orm.RegisterModel(new(models.StorageVolumes))
	orm.RegisterModel(new(models.StorageEventLog))
	orm.RegisterModel(new(models.HardwareEventLog))
	orm.RegisterModel(new(models.ServerSprEventLog))

	orm.RegisterModel(new(models.MembersFG))
	orm.RegisterModel(new(models.Counterparty))

	orm.RegisterModel(new(models.AppService))
	orm.RegisterModel(new(models.ServiceMember))

	orm.RegisterModel(new(models.ServerOwnershipHistory))

	orm.RegisterModel(new(models.ServerPerfSCOM))
	orm.RegisterModel(new(models.ServerPerfZabbix))
	orm.RegisterModel(new(models.ServerDiskPerfZabbix))
	orm.RegisterModel(new(models.InsightVm))
	orm.RegisterModel(new(models.InsightVmPerf))
	orm.RegisterModel(new(models.VMUsageDatastore))

	//CMDB
	//убраны для исключения из схемы, т.к. в CMDB они не используются и не нужны
	//orm.RegisterModel(new(models.ConfigurationUnit))
	//orm.RegisterModel(new(models.ConfigurationUnitLink))
	// models.Datacenter/models.Rack удалены полностью 2026-09-06 (мёртвый
	// легаси-код бюджетного нормативного контура, никогда не
	// регистрировались и не синхронизировались) - см. CHANGELOG.md.

	orm.RegisterModel(new(models.DatacenterSite))
	orm.RegisterModel(new(models.RackCabinet))
	orm.RegisterModel(new(models.NetworkSprRecords))
	orm.RegisterModel(new(models.SwitchSprRecords))
	orm.RegisterModel(new(models.SwitchNetworkLink))
	orm.RegisterModel(new(models.ModelEquipment))
	orm.RegisterModel(new(models.ModelVendor))
	orm.RegisterModel(new(models.Components))
	orm.RegisterModel(new(models.ComponentsCPU))
	orm.RegisterModel(new(models.ComponentsRAM))
	orm.RegisterModel(new(models.ComponentType))
	orm.RegisterModel(new(models.ReasonEndLife))
	orm.RegisterModel(new(models.CategoryEquipment))
	orm.RegisterModel(new(models.ServerType))
	orm.RegisterModel(new(models.OperationalSystem))
	orm.RegisterModel(new(models.ModelsEndLife))
	//
	orm.RegisterModel(new(models.ModelPersonalEquipment))
	orm.RegisterModel(new(models.RolePersonal))
	orm.RegisterModel(new(models.RolePersonalItem))
	orm.RegisterModel(new(models.CompetenceCenter))
	orm.RegisterModel(new(models.RoleCompetenceCenter))
	//zabbix
	orm.RegisterModel(new(models.ZabbixHost))
	orm.RegisterModel(new(models.ZabbixItem))
	orm.RegisterModel(new(models.ZabbixKey))
	orm.RegisterModel(new(models.ServerMetric))
	orm.RegisterModel(new(models.ServerMetricRecord))
	orm.RegisterModel(new(models.StorageEnclosure))
	orm.RegisterModel(new(models.StorageDisk))
	orm.RegisterModel(new(models.StorageMetric))
	orm.RegisterModel(new(models.StorageMetricRecord))

	err := orm.RunSyncdb("default", false, false)
	logSyncdbErr("default", err)

	err = orm.RunSyncdb("default-add", false, false)
	logSyncdbErr("default-add", err)

	/*
		name := "default"
		force := false
		verbose := false
		err := orm.RunSyncdb(name, force, verbose)
		if err != nil {
			log.Println(err)
		}
	*/
}
