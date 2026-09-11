package itilliumimport

import (
	"loadmon/models"
	"log"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"

	"github.com/beego/beego/v2/client/orm"
)

// mappingConfigFileName - имя файла соответствий для импорта rack-серверов
// внутри каталога, заданного ключом "yaml" в app.conf.
const mappingConfigFileName = "rack_server.yaml"

// mappingConfigDir возвращает каталог с yaml-описаниями для импорта данных.
// Берётся из app.conf (ключ "yaml", см. server_web_new/conf/app.conf) - тот
// же приём, что и viewExportSQLDir() в controllers/adminviewexport.go. Если
// ключ не задан - запасной вариант "./conf/yaml" (совпадает со значением по
// умолчанию в app.conf).
func mappingConfigDir() string {
	if dir, err := beego.AppConfig.String("yaml"); err == nil && strings.TrimSpace(dir) != "" {
		return dir
	}
	return "./conf/yaml"
}

// mappingConfigPath - полный путь к yaml-файлу соответствий, читаемому в
// ImportItiliumServer. Сам файл нужно положить в каталог, указанный в
// app.conf (ключ "yaml"), под именем mappingConfigFileName - см.
// itilliumimport/mapping.yaml как образец (перенесите его туда же).
func mappingConfigPath() string {
	return mappingConfigDir() + "/" + mappingConfigFileName
}

// StartImportItiliumServer регистрирует новое задание импорта rack-серверов
// под ключом jobId (см. ImportProgress в progress.go) и запускает сам разбор
// и сохранение файла (runImportItiliumServer) в отдельной горутине, сразу
// возвращая указатель на прогресс вызывающему коду (controllers/
// adminimportrackservers.go) - 2026-09-08, по запросу пользователя "сделай
// аналогичный progress bar для импорта rack серверов и blade серверов" (см.
// itilliumimport/progress.go и то же решение для блейд-корзин, запись (42) в
// CHANGELOG.md). Job регистрируется СИНХРОННО, до запуска горутины - иначе
// страница могла бы успеть опросить статус раньше, чем задание появится в
// реестре (GetImportJob вернул бы ok=false).
func StartImportItiliumServer(name string) *ImportProgress {
	job := newImportJob(name)
	go runImportItiliumServer(name, job)
	return job
}

// ImportItiliumServer - синхронная версия импорта rack-серверов (без
// отдельной горутины и без прогресса) - оставлена для прямых вызовов вне
// HTTP-хендлера, где отслеживать прогресс не нужно. Текущий HTTP-хендлер
// использует StartImportItiliumServer выше.
func ImportItiliumServer(name string) {
	job := &ImportProgress{}
	runImportItiliumServer(name, job)
}

// runImportItiliumServer читает xlsx-файл name (внутри каталога "./import",
// см. controllers/adminimportrackservers.go), конвертирует его строки через
// ConvertFile/ConvertRow (mapping.yaml) и создаёт/обновляет записи
// models.DatacenterSite -> models.RackCabinet -> models.ServerHardware ->
// models.ServerSprRecords в БД. Одна строка исходного файла - одна
// транзакция (аналогично циклу cmdb/updatehwsprhp.go:UpdateHwSprHPServers) -
// ошибка в одной строке не прерывает импорт остальных.
//
// job (см. progress.go) отражает ход выполнения для страницы-опросчика -
// job.finish(...) вызывается ЛЮБЫМ путём выхода из функции (defer), поэтому
// индикатор прогресса на странице не может "зависнуть", даже если импорт не
// смог даже начаться (ошибка загрузки конфигурации/чтения файла).
func runImportItiliumServer(name string, job *ImportProgress) {
	var startErr string
	defer func() { job.finish(startErr) }()

	cfg, err := LoadMapping(mappingConfigPath())
	if err != nil {
		log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка загрузки конфигурации соответствий:", err)
		startErr = err.Error()
		return
	}

	results, err := ConvertFile(cfg, "./import/"+name)
	if err != nil {
		log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка чтения/разбора файла:", err)
		startErr = err.Error()
		return
	}
	job.setFileRows(len(results))

	o := orm.NewOrm()

	for _, r := range results {
		job.tickProcessed()

		// r.Spr  - models.ServerSprRecords (частично заполнена)
		// r.Hw   - models.ServerHardware (частично заполнена)
		// r.Extra    - map[string]string: поля без прямого соответствия в
		//              схеме + диагностические "сырые" значения
		// r.Warnings - []string: на что стоит посмотреть глазами перед сохранением
		trn, err := o.Begin()
		if err != nil {
			log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка начала транзакции:", err)
			continue
		}

		// --- ЦОД (models.DatacenterSite) --------------------------------
		datacenter := models.DatacenterSite{
			Name:       r.Hw.DatacenterRoom + " " + r.Extra["location_room_code"],
			Annotation: "ЦОД по адресу: " + r.Hw.DatacenterRoom + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"],
			Id_User:    9999,
			Time_Mfy:   time.Now().Local(),
		}
		if err := trn.Read(&datacenter, "Name"); err != nil {
			if _, err := trn.Insert(&datacenter); err != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка вставки datacenter:", err)
				trn.Rollback()
				continue
			}
		}
		datacenterId := datacenter.Id

		// --- Шкаф (models.RackCabinet) - резолвится ПАРОЙ (Name,
		// DatacenterSiteId), см. resolveRackCabinet: одинаковое имя шкафа
		// допустимо на разных площадках --------------------------------
		//
		// ИЗМЕНЕНО (2026-09-11, тот же баг, что исправлен в
		// importblades.go/importbladeservers.go/importstorages.go):
		// "Размещение в шкафу/стойке" может указывать colocation-площадку
		// ПЕРЕД "/", отличную от "Местоположение" строки - см.
		// resolvecabinet.go:splitColocationCabinetName.
		colocationDc, cabinetCode := splitColocationCabinetName(r.Hw.RackNum)
		rackSiteId := datacenterId
		rackAnnotation := "Шкаф по адресу: " + r.Hw.DatacenterRoom + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"]
		if colocationDc != "" {
			var errSite error
			rackSiteId, errSite = resolveDatacenterSiteByName(trn, colocationDc)
			if errSite != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка резолва colocation ЦОД", colocationDc, ":", errSite)
				trn.Rollback()
				continue
			}
			rackAnnotation = "Шкаф на colocation-площадке \"" + colocationDc + "\" (по данным импорта из Итиллиум)"
		}
		rackId, errRack := resolveRackCabinet(trn, cabinetCode, rackSiteId, rackAnnotation)
		if errRack != nil {
			log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка резолва rack:", errRack)
			trn.Rollback()
			continue
		}
		r.Hw.RackCabinetId = rackId

		// --- Статус/питание ----------------------------------------------
		r.Spr.SourceUpdateType = models.SourceUpdateManual
		r.Hw.SourceUpdateType = models.SourceUpdateManual
		if r.Hw.SerialNum != "" {
			r.Spr.SerialNumber = r.Hw.SerialNum
		} else {
			r.Spr.SerialNumber = r.Hw.TechnicalNumberIT
			r.Hw.SerialNum = r.Hw.TechnicalNumberIT
		}
		if r.Spr.Stat == 1 && r.Spr.Active == 1 {
			r.Spr.PowerState = 1
			r.Hw.PowerState = 1
		}

		// --- ServerHardware ------------------------------------------------
		var hw models.ServerHardware
		hwFound := r.Hw.SerialNum != ""
		if hwFound {
			hw = models.ServerHardware{SerialNum: r.Hw.SerialNum}
			hwFound = trn.Read(&hw, "SerialNum") == nil
		}
		if !hwFound {
			// Новая запись - вставляем указатель на r.Hw (не значение!),
			// иначе автосгенерированный Id не запишется обратно (см.
			// beego/v2/client/orm/orm.go:InsertWithCtx - Insert требует
			// указатель, чтобы установить Id через reflect после вставки).
			hwId, err := trn.Insert(&r.Hw)
			if err != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка вставки hw:", err)
				trn.Rollback()
				continue
			}
			r.Hw.Id = hwId
		} else {
			// Запись уже существует - обновляем только поля, которые
			// однозначно должен приносить импорт (место в стойке,
			// состояние питания, источник обновления), НЕ перезаписывая
			// остальные поля существующей записи данными из файла - на
			// случай, если они были вручную скорректированы в интерфейсе
			// после прошлой синхронизации (тот же принцип приоритета
			// ручных правок, что и для СХД, см.
			// kontext/05-conventions-and-gotchas.md). Если нужно
			// перезаписывать больше полей при повторном импорте -
			// перечислите их здесь и передайте именами в trn.Update
			// (сейчас обновляются только явно перечисленные 3 колонки).
			r.Hw.Id = hw.Id
			hw.RackCabinetId = r.Hw.RackCabinetId
			hw.PowerState = r.Hw.PowerState
			hw.SourceUpdateType = r.Hw.SourceUpdateType
			if _, err := trn.Update(&hw, "RackCabinetId", "PowerState", "SourceUpdateType"); err != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления hw:", err)
				trn.Rollback()
				continue
			}
		}

		// --- ServerSprRecords -----------------------------------------------
		r.Spr.HWSprId = r.Hw.Id

		var sprsrv models.ServerSprRecords
		sprFound := hw.ServerId != 0
		if sprFound {
			sprsrv = models.ServerSprRecords{Id: hw.ServerId}
			sprFound = trn.Read(&sprsrv) == nil
		}
		if !sprFound {
			sprId, err := trn.Insert(&r.Spr)
			if err != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка вставки sprsrv:", err)
				trn.Rollback()
				continue
			}
			r.Spr.Id = sprId
			// Обратная связь ServerHardware.ServerId -> новую запись
			// справочника серверов (иначе HWSprId в ServerSprRecords
			// указывает на ServerHardware, а обратной ссылки не будет,
			// как это устроено во всех остальных точках синхронизации,
			// см. cmdb/updatehwsprhp.go).
			r.Hw.ServerId = sprId
			if _, err := trn.Update(&r.Hw, "ServerId"); err != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления hw.ServerId:", err)
				trn.Rollback()
				continue
			}
		} else {
			r.Spr.Id = sprsrv.Id
			sprsrv.HWSprId = r.Spr.HWSprId
			// ИСПРАВЛЕНО (2026-09-10, баг стирания имён): Name/NameFQDN
			// перезаписываются значением из файла ТОЛЬКО если оно не пусто.
			// r.Spr.Name/NameFQDN приходят из ConvertRow (convert.go) через
			// setTarget, который вообще не трогает поле при пустой/"nan"/
			// "-"/"n/a" ячейке (см. cleanString) - то есть r.Spr.Name может
			// быть "" не потому что сервер реально безымянный, а просто
			// потому что в этой строке выгрузки Итиллиум ячейка была
			// пустой. Раньше это безусловно затирало уже сохранённое
			// непустое имя записи sprsrv (прочитанной из БД чуть выше) при
			// повторном импорте - ровно симптом "стёрлись имена серверов и
			// FQDN по части аппаратных серверов и гипервизоров". Тот же
			// принцип защиты уже применяется в cmdb/updatehwsprhp.go для
			// HP OneView-синхронизации.
			if r.Spr.Name != "" {
				sprsrv.Name = r.Spr.Name
			}
			if r.Spr.NameFQDN != "" {
				sprsrv.NameFQDN = r.Spr.NameFQDN
			}
			sprsrv.Stat = r.Spr.Stat
			sprsrv.Active = r.Spr.Active
			sprsrv.PowerState = r.Spr.PowerState
			sprsrv.SourceUpdateType = r.Spr.SourceUpdateType
			if _, err := trn.Update(&sprsrv, "HWSprId", "Name", "NameFQDN", "Stat", "Active", "PowerState", "SourceUpdateType"); err != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления sprsrv:", err)
				trn.Rollback()
				continue
			}
		}

		// TODO: r.Extra содержит поля без прямого соответствия в схеме
		// (см. itilliumimport/README.md, таблица "Что осталось в Extra и
		// почему") - на данный момент они нигде не сохраняются (ни в БД,
		// ни в лог) - решить, нужно ли и куда их пристраивать (например,
		// в ServerSprRecords.Details или отдельный лог импорта).
		for _, w := range r.Warnings {
			log.Println("itilliumimport:", name, w)
		}

		if err := trn.Commit(); err != nil {
			log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка коммита транзакции:", err)
			continue
		}

		// --- Статистика для страницы-опросчика (progress.go) --------------
		// (2026-09-08, по запросу пользователя "сделай аналогичный progress
		// bar для импорта rack серверов и blade серверов", тот же принцип,
		// что для блейд-корзин - "новый"/"обновлён" по !sprFound,
		// "неактивен" по итоговому r.Spr.Active == 2, см. models_add.go:
		// ServerSprRecords.Active).
		job.addResult(!sprFound, r.Spr.Active == 2)
	}
}
