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

	// Пошаговое логирование запуска (2026-10-12): по жалобе "web подвисает
	// при запуске после сообщения про режим авторизации" - всё, что идёт
	// между этим сообщением и стартом HTTP-сервера, происходит здесь
	// (подключение к БД, RunSyncdb и миграции с DDL-блокировками), и без
	// логов не было видно, на каком шаге именно ожидание.
	dbInitStep := func(name string, f func()) {
		t0 := time.Now()
		log.Printf("DBInit: %s - начало", name)
		// "пульс" раз в 10 с: шаг долгий, но программа не зависла - видно,
		// на каком именно шаге идёт работа
		done := make(chan struct{})
		go func() {
			tk := time.NewTicker(10 * time.Second)
			defer tk.Stop()
			for {
				select {
				case <-done:
					return
				case <-tk.C:
					log.Printf("DBInit: %s - всё ещё выполняется (%s)...", name, time.Since(t0).Round(time.Second))
				}
			}
		}()
		f()
		close(done)
		log.Printf("DBInit: %s - готово за %s", name, time.Since(t0).Round(time.Millisecond))
	}

	orm.ResetModelCache()
	orm.DefaultTimeLoc = time.Local
	orm.RegisterDriver("postgres", orm.DRPostgres)
	orm.RegisterDriver("sqlite3", orm.DRSqlite)

	alias, driver, dataSource := buildDataSource(ConfigData.DriverName, ConfigData.UserName, ConfigData.Password,
		ConfigData.DBName, ConfigData.DBFile, ConfigData.HostName, ConfigData.Port, ConfigData.SSLmode)
	dbInitStep("подключение к БД "+alias, func() { orm.RegisterDataBase(alias, driver, dataSource) })

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
		dbInitStep("подключение к БД default-add", func() { orm.RegisterDataBase("default-add", "sqlite3", sqliteDataSource(dbFileAdd)) })

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
		dbInitStep("подключение к БД default-add (postgres)", func() { orm.RegisterDataBase("default-add", "postgres", dataSourceAdd) })
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
	orm.RegisterModel(new(models.StorageSystemPlacement))
	orm.RegisterModel(new(models.TapeLibrary))
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
	orm.RegisterModel(new(models.Locality))
	orm.RegisterModel(new(models.RackCabinet))
	orm.RegisterModel(new(models.NetworkSprRecords))
	orm.RegisterModel(new(models.SwitchSprRecords))
	orm.RegisterModel(new(models.SwitchNetworkLink))
	orm.RegisterModel(new(models.IPAddressSprRecords))
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
	orm.RegisterModel(new(models.ServerPowerMetric))  // поминутная мощность серверов (пакет zabbixpower), хранится 3 суток
	orm.RegisterModel(new(models.ServerPowerRecord))  // суточная агрегация мощности
	orm.RegisterModel(new(models.StoragePowerMetric)) // поминутная мощность СХД (пакет storagepower), хранится 3 суток
	orm.RegisterModel(new(models.StoragePowerRecord)) // суточная агрегация мощности СХД
	orm.RegisterModel(new(models.BladePowerRecord))   // суточные снимки мощности блейд-корзин (OneView, только по корзине в целом)
	orm.RegisterModel(new(models.RackPowerRecord))    // суточная мощность шкафа (пакет rackpower)
	orm.RegisterModel(new(models.SitePowerRecord))    // суточная мощность ЦОД/площадки (пакет rackpower)
	orm.RegisterModel(new(models.ScomHost))           // справочник компьютеров Microsoft SCOM (пакет scommetrics), см. kontext/13-scom-metrics.md
	orm.RegisterModel(new(models.StorageEnclosure))
	orm.RegisterModel(new(models.StorageDisk))
	orm.RegisterModel(new(models.StoragePool))
	orm.RegisterModel(new(models.StorageMetric))
	orm.RegisterModel(new(models.StorageMetricRecord))
	orm.RegisterModel(new(models.ServerNetworkInterface)) // Stage 1 хранения "распределённых" сетевых интерфейсов серверов, см. CHANGELOG.md

	// Одноразовая дедупликация RackCabinet ДО создания композитного
	// уникального индекса (Name, DatacenterSiteId) - см. комментарий у
	// RackCabinet в models_add.go и dedupeDuplicateRackCabinets ниже.
	// Должна выполняться ДО RunSyncdb("default", ...), иначе ADD UNIQUE
	// INDEX упадёт на уже существующих дублях шкафов с одинаковым Name
	// на разных площадках (баг импорта СХД/блейд-корзин/rack-серверов,
	// искавших шкаф только по Name, без учёта DatacenterSiteId).
	// Преобразования базы (RunSyncdb, миграции, триггеры) выполняет ТОЛЬКО
	// metrics_grpc (WorkMode=api) - web-интерфейс и утилиты (WorkMode=web)
	// лишь регистрируют модели и работают с уже подготовленной базой, не
	// задерживая старт на DDL-операциях (2026-10-12, по жалобе "web
	// долго стартует, делая непонятно что"). Принудительный запуск из
	// любого режима - переменная окружения LOADMON_DB_MIGRATE=1 (например,
	// для tools/seeddata на пустой базе).
	if ConfigData.WorkMode != "api" && os.Getenv("LOADMON_DB_MIGRATE") != "1" {
		log.Printf("DBInit: WorkMode=%q - синхронизация схемы и миграции БД выполняет metrics_grpc (WorkMode=api), здесь пропущены", ConfigData.WorkMode)
		return
	}

	dbInitStep("дедупликация шкафов", dedupeDuplicateRackCabinets)

	var err error
	dbInitStep("RunSyncdb default", func() {
		err = orm.RunSyncdb("default", false, false)
		logSyncdbErr("default", err)
	})
	dbInitStep("RunSyncdb default-add", func() {
		err = orm.RunSyncdb("default-add", false, false)
		logSyncdbErr("default-add", err)
	})

	// Одноразовый перенос старых данных размещения СХД (RackCabinetId/
	// StartUnitPos/EndUnitPos/SizeUnit) из уже "осиротевших" колонок
	// storage_systems в новую таблицу StorageSystemPlacement - см. её
	// комментарий в models_add.go и migrateStorageSystemsPlacement ниже.
	// Вызывается ПОСЛЕ RunSyncdb, т.к. таблица storage_system_placement
	// должна уже существовать.
	dbInitStep("перенос размещения СХД", migrateStorageSystemsPlacement)

	// Перенос уникальности server_metric/server_metric_record с (HostId, ...)
	// на (Source, HostId, ...) на уже существующих базах - см.
	// config/dbinit_servermetric.go. Тоже ПОСЛЕ RunSyncdb: колонку source
	// добавляет именно он.
	addDriver := "postgres"
	if ConfigData.DriverName == "sqlite3" {
		addDriver = "sqlite3"
	}
	aliasDrivers := map[string]string{alias: driver, "default-add": addDriver}
	dbInitStep("уникальность server_metric", func() { migrateServerMetricSourceUnique(aliasDrivers) })

	// Записи server_spr_records со Stat=666 ("к уничтожению") не должны
	// меняться никаким кодом - защита триггером БД, см.
	// config/dbinit_serverspr_destroy.go.
	dbInitStep("триггер server_spr_records (блокировка таблицы)", func() { ensureServerSprDestroyTrigger(aliasDrivers) })

	// Одноразовые миграции данных (таблица loadmon_migration), см.
	// config/dbinit_migrations.go. Справочник серверов живёт в "default".
	dbInitStep("одноразовые миграции данных", func() { runDataMigrations(alias) })

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

// migrateStorageSystemsPlacement переносит одноразово данные старой схемы
// размещения СХД (одиночный FK на шкаф прямо на StorageSystems) в новую
// таблицу-связку models.StorageSystemPlacement - по одной записи на каждую
// СХД, у которой в старых колонках было проставлено размещение,
// IsPrimary=true (см. подробный комментарий у полей StorageSystems и у
// StorageSystemPlacement в models_add.go, запись 2026-09-09 в CHANGELOG.md).
//
// Колонки rack_cabinet_id/start_unit_pos/end_unit_pos/size_unit таблицы
// storage_systems больше не описаны в Go-структуре models.StorageSystems
// (RunSyncdb удаляет только отсутствующее - ДОБАВЛЯЕТ недостающие
// таблицы/колонки, но никогда не убирает лишние), поэтому они читаются
// сырым SQL через o.Raw(...), а не через ORM-запрос по структуре.
//
// Идемпотентна и безопасна для многократного вызова при каждом старте
// приложения: переносится только та StorageSystems.Id, для которой в
// storage_system_placement ЕЩЁ НЕТ ни одной записи - повторный перенос
// новых записей второй раз не создаёт. Если старых колонок в БД уже нет
// (например, на свежей базе, где storage_systems создана заново уже без
// них) - o.Raw(...).QueryRows вернёт ошибку, которая тихо игнорируется:
// переносить в этом случае просто нечего.
func migrateStorageSystemsPlacement() {
	o := orm.NewOrm()

	type legacyStoragePlacement struct {
		Id            int64
		RackCabinetId int64
		StartUnitPos  int
		EndUnitPos    int
		SizeUnit      int
	}
	var rows []legacyStoragePlacement
	_, err := o.Raw("SELECT id, rack_cabinet_id, start_unit_pos, end_unit_pos, size_unit FROM storage_systems WHERE rack_cabinet_id IS NOT NULL AND rack_cabinet_id > 0").QueryRows(&rows)
	if err != nil {
		// Старых колонок уже нет в БД либо иная ошибка чтения - переносить
		// нечего, это не считается сбоем инициализации.
		return
	}
	if len(rows) == 0 {
		return
	}

	migrated := 0
	for _, r := range rows {
		// ИСПРАВЛЕНО: в используемой версии beego/v2/client/orm
		// QuerySeter.Exist() возвращает только bool (без error) - в отличие
		// от некоторых других методов ORM здесь нечего проверять на ошибку.
		exists := o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId", r.Id).Exist()
		if exists {
			// Для этой СХД размещение уже перенесено (или заведено заново
			// вручную через новую форму) - не трогаем.
			continue
		}
		p := models.StorageSystemPlacement{
			StorageSystemsId: r.Id,
			RackCabinetId:    r.RackCabinetId,
			StartUnitPos:     r.StartUnitPos,
			EndUnitPos:       r.EndUnitPos,
			SizeUnit:         r.SizeUnit,
			IsPrimary:        true,
			Id_User:          9999,
			Time_Mfy:         time.Now().Local(),
		}
		if _, err := o.Insert(&p); err != nil {
			log.Println("config.migrateStorageSystemsPlacement: ошибка вставки размещения для StorageSystems.Id=", r.Id, ":", err)
			continue
		}
		migrated++
	}
	if migrated > 0 {
		log.Println("config.migrateStorageSystemsPlacement: перенесено размещений СХД из старой схемы:", migrated)
	}
}

// dedupeDuplicateRackCabinets устраняет дубли RackCabinet ПЕРЕД тем, как
// RunSyncdb попытается создать композитный уникальный индекс
// (Name, DatacenterSiteId) - см. комментарий у RackCabinet в
// models_add.go.
//
// Причина дублей: импорт из Итиллиум (importstorages.go/importrackservers.go
// /importblades.go) до этого исправления искал существующий шкаф методом
// trn.Read(&rack, "Name") - строго по имени, БЕЗ учёта DatacenterSiteId.
// Так как имена шкафов ("3.1-A-08" и т.п.) переиспользуются на разных
// площадках, первый найденный шкаф с таким Name считался "тем самым" и его
// DatacenterSiteId тихо перезаписывался на площадку из последней
// импортированной строки - вместо того чтобы завести второй шкаф с тем же
// именем на другой площадке. Прямых дублей строк (Name, DatacenterSiteId)
// это не создавало, зато создавало ситуацию, когда один и тот же
// RackCabinet.Id к моменту исправления бага может быть "перепривязан" не к
// той площадке, к которой относилось оборудование, размещённое в нём
// раньше - см. предупреждение в конце этой функции.
//
// Сама дедупликация нужна на случай штатных дублей: если ДО этого фикса
// пользователь вручную завёл (или тот же импорт успел завести, например, в
// параллельных горутинах/повторных запусках) две записи RackCabinet с
// одинаковой парой (Name, DatacenterSiteId) - без неё ADD UNIQUE INDEX в
// RunSyncdb упадёт с ошибкой и приложение не сможет синхронизировать схему.
//
// Идемпотентна: при повторном запуске дублей уже не будет (уникальный
// индекс их не позволит создать), функция просто ничего не найдёт и сразу
// вернётся.
func dedupeDuplicateRackCabinets() {
	o := orm.NewOrm()

	type cabinetRow struct {
		Id               int64
		Name             string
		DatacenterSiteId int64
	}
	var rows []cabinetRow
	_, err := o.Raw("SELECT id, name, datacenter_site_id FROM cmdb_rack_cabinet ORDER BY id").QueryRows(&rows)
	if err != nil {
		// Таблицы ещё нет (первый запуск на чистой БД) либо иная ошибка
		// чтения - дедуплицировать нечего, RunSyncdb сам создаст таблицу
		// сразу с уникальным индексом.
		return
	}

	// Группируем по (Name, DatacenterSiteId); первый увиденный Id в группе
	// (наименьший, т.к. rows отсортированы по id) остаётся "каноническим",
	// остальные - дубли на удаление с переносом ссылок на канонический Id.
	type groupKey struct {
		Name             string
		DatacenterSiteId int64
	}
	canonical := map[groupKey]int64{}
	duplicates := map[int64]int64{} // dupId -> canonicalId

	for _, r := range rows {
		key := groupKey{Name: r.Name, DatacenterSiteId: r.DatacenterSiteId}
		if canonicalId, ok := canonical[key]; ok {
			duplicates[r.Id] = canonicalId
			continue
		}
		canonical[key] = r.Id
	}

	if len(duplicates) == 0 {
		return
	}

	log.Println("config.dedupeDuplicateRackCabinets: найдено дублей RackCabinet (одинаковые Name+DatacenterSiteId):", len(duplicates), "- переносим ссылки и удаляем дубли")

	for dupId, canonicalId := range duplicates {
		// Переносим ссылки во всех таблицах, где есть RackCabinetId FK -
		// storage_system_placement, blade_systems, server_hardware (см.
		// комментарий в models_add.go у RackCabinet: "Связывается с
		// BladeSystems (BladeSystems.RackCabinetId) и ServerHardware
		// (ServerHardware.RackCabinetId)").
		for _, table := range []string{"storage_system_placement", "blade_systems", "server_hardware"} {
			sql := "UPDATE " + table + " SET rack_cabinet_id = ? WHERE rack_cabinet_id = ?"
			if _, err := o.Raw(sql, canonicalId, dupId).Exec(); err != nil {
				log.Println("config.dedupeDuplicateRackCabinets: ошибка переноса ссылок из", table, "с dup Id=", dupId, "на canonical Id=", canonicalId, ":", err)
			}
		}
		if _, err := o.Raw("DELETE FROM cmdb_rack_cabinet WHERE id = ?", dupId).Exec(); err != nil {
			log.Println("config.dedupeDuplicateRackCabinets: ошибка удаления дубля RackCabinet.Id=", dupId, ":", err)
		}
	}

	log.Println("config.dedupeDuplicateRackCabinets: дедупликация RackCabinet завершена. ВНИМАНИЕ: из-за исторического бага поиска шкафа только по Name (без DatacenterSiteId) часть оборудования, ранее размещённого в 'схлопнутых' шкафах, могла быть привязана не к той площадке - при необходимости сверьте размещение СХД/блейд-корзин/rack-серверов вручную через справочники.")
}
