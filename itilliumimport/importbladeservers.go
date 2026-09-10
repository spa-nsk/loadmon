package itilliumimport

import (
	"loadmon/models"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// mappingConfigFileNameBladeServer - имя файла соответствий для импорта
// блейд-серверов внутри каталога, заданного ключом "yaml" в app.conf (см.
// mappingConfigDir() в importrackservers.go - переиспользуется как есть).
const mappingConfigFileNameBladeServer = "blade_servers.yaml"

// mappingConfigPathBladeServer - полный путь к yaml-файлу соответствий,
// читаемому в ImportItiliumBladeServer. Сам файл нужно положить в каталог,
// указанный в app.conf (ключ "yaml"), под именем
// mappingConfigFileNameBladeServer - см. itilliumimport/mappingbladeservers.yaml
// как образец.
func mappingConfigPathBladeServer() string {
	return mappingConfigDir() + "/" + mappingConfigFileNameBladeServer
}

// ImportItiliumBladeServer читает xlsx-файл name (внутри каталога
// "./import", см. controllers/adminimportbladeservers.go), конвертирует
// его строки через ConvertFile/ConvertRow (rack_blade_server.yaml,
// mappingbladeservers.yaml) и создаёт/обновляет записи
// models.DatacenterSite -> models.RackCabinet -> models.ServerHardware ->
// models.ServerSprRecords в БД - полный аналог ImportItiliumServer
// (importrackservers.go), т.к. целевые структуры (ServerSprRecords/
// ServerHardware) те же самые.
//
// Отличия от обычных rack-серверов:
//  1. ServerHardware.FormFactor всегда проставляется в 2 (блейд, см.
//     комментарий у ServerHardware.FormFactor в models/models_add.go).
//  2. Дополнительная привязка к models.BladeSystems (блейд-корзина):
//     колонка "Корзина" ("(IT-0000003911) C7000-15") разбирается
//     composite'ом blade_ref_split (см. convert.go) в
//     extra["blade_ref_name"] - здесь по этому имени ищется запись
//     BladeSystems, и при успехе:
//     - ServerHardware.BladeSystemId = найденная запись;
//     - ServerHardware.RackCabinetId ПЕРЕОПРЕДЕЛЯЕТСЯ на RackCabinetId
//     корзины (она физически определяет размещение вложенного в неё
//     сервера - это надёжнее, чем повторно разбирать место установки из
//     строки самого сервера, тем более что в этой выгрузке нет колонки
//     "Размещение в шкафу/стойке", т.е. с) ServerHardware.RackNum и так
//     был бы пустым).
//     Порядок обработки места установки - СНАЧАЛА стандартный разбор
//     "Местоположение" в DatacenterSite/RackCabinet (тот же код, что и в
//     ImportItiliumServer - "аналогично обычным серверам", как просил
//     пользователь), ЗАТЕМ, если блейд-корзина найдена - её RackCabinetId
//     имеет приоритет (см. шаг "Привязка к блейд-корзине" ниже). Если
//     корзина не найдена (например, ещё не импортирована через
//     /admin/importblades) - используется результат стандартного разбора,
//     событие логируется как предупреждение.
// StartImportItiliumBladeServer регистрирует новое задание импорта
// блейд-серверов под ключом jobId (см. ImportProgress в progress.go) и
// запускает сам разбор и сохранение файла (runImportItiliumBladeServer) в
// отдельной горутине, сразу возвращая указатель на прогресс вызывающему коду
// (controllers/adminimportbladeservers.go) - 2026-09-08, по запросу
// пользователя "сделай аналогичный progress bar для импорта rack серверов и
// blade серверов" (то же решение, что для блейд-корзин, см. CHANGELOG.md
// запись (42), и для rack-серверов - StartImportItiliumServer). Job
// регистрируется СИНХРОННО, до запуска горутины.
func StartImportItiliumBladeServer(name string) *ImportProgress {
	job := newImportJob(name)
	go runImportItiliumBladeServer(name, job)
	return job
}

// ImportItiliumBladeServer - синхронная версия импорта блейд-серверов (без
// отдельной горутины и без прогресса) - оставлена для прямых вызовов вне
// HTTP-хендлера. Текущий HTTP-хендлер использует StartImportItiliumBladeServer
// выше.
func ImportItiliumBladeServer(name string) {
	job := &ImportProgress{}
	runImportItiliumBladeServer(name, job)
}

// runImportItiliumBladeServer читает xlsx-файл name (внутри каталога
// "./import", см. controllers/adminimportbladeservers.go) и создаёт/
// обновляет записи - см. подробный комментарий у ImportItiliumBladeServer
// выше. job (см. progress.go) отражает ход выполнения для
// страницы-опросчика - job.finish(...) вызывается ЛЮБЫМ путём выхода из
// функции (defer).
func runImportItiliumBladeServer(name string, job *ImportProgress) {
	var startErr string
	defer func() { job.finish(startErr) }()

	cfg, err := LoadMapping(mappingConfigPathBladeServer())
	if err != nil {
		log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка загрузки конфигурации соответствий:", err)
		startErr = err.Error()
		return
	}

	results, err := ConvertFile(cfg, "./import/"+name)
	if err != nil {
		log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка чтения/разбора файла:", err)
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
		//              схеме + диагностические "сырые" значения, включая
		//              blade_ref_name/blade_ref_tech_number (см. выше)
		// r.Warnings - []string: на что стоит посмотреть глазами перед сохранением
		trn, err := o.Begin()
		if err != nil {
			log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка начала транзакции:", err)
			continue
		}

		r.Hw.FormFactor = 2 // блейд (1-rack, 2-blade, см. ServerHardware.FormFactor)

		// --- ЦОД (models.DatacenterSite) - стандартный разбор "Местоположение",
		// как у ImportItiliumServer ("аналогично обычным серверам") -------
		datacenter := models.DatacenterSite{
			Name:       r.Hw.DatacenterRoom + " " + r.Extra["location_room_code"],
			Annotation: "ЦОД по адресу: " + r.Hw.DatacenterRoom + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"],
			Id_User:    9999,
			Time_Mfy:   time.Now().Local(),
		}
		if err := trn.Read(&datacenter, "Name"); err != nil {
			if _, err := trn.Insert(&datacenter); err != nil {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка вставки datacenter:", err)
				trn.Rollback()
				continue
			}
		}
		datacenterId := datacenter.Id

		// --- Шкаф (models.RackCabinet) - см. выше, RackNum в этой выгрузке
		// обычно пуст (нет колонки "Размещение в шкафу/стойке"), поэтому
		// find-or-create по пустому имени намеренно пропускается (иначе
		// завели бы мусорную запись RackCabinet с пустым Name) - реальная
		// привязка к шкафу приходит ниже, через блейд-корзину.
		if strings.TrimSpace(r.Hw.RackNum) != "" {
			// Резолвится ПАРОЙ (Name, DatacenterSiteId), см.
			// resolveRackCabinet: одинаковое имя шкафа допустимо на разных
			// площадках.
			rackId, errRack := resolveRackCabinet(trn, r.Hw.RackNum, datacenterId,
				"Шкаф по адресу: "+r.Hw.DatacenterRoom+" этаж:"+r.Extra["location_zone_code"]+" комната: "+r.Extra["location_room_code"])
			if errRack != nil {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка резолва rack:", errRack)
				trn.Rollback()
				continue
			}
			r.Hw.RackCabinetId = rackId
		}

		// --- Привязка к блейд-корзине (models.BladeSystems) - ДОПОЛНИТЕЛЬНО
		// к обработке места установки выше, по просьбе пользователя.
		// RackCabinetId корзины имеет приоритет над результатом стандартного
		// разбора места установки (см. комментарий у ImportItiliumBladeServer).
		if bladeRefName := strings.TrimSpace(r.Extra["blade_ref_name"]); bladeRefName != "" {
			bs := models.BladeSystems{Name: bladeRefName}
			if err := trn.Read(&bs, "Name"); err == nil {
				r.Hw.BladeSystemId = bs.Id
				// ИЗМЕНЕНО (2026-09-08): models.BladeSystems лишился
				// текстовых DatacenterRoom/RackNum - расположение корзины
				// теперь переносится на сервер только через RackCabinetId
				// (FK), текстовые поля ServerHardware.DatacenterRoom/RackNum
				// сервер получает исключительно из разбора "Местоположение"
				// собственной строки выгрузки (см. выше).
				if bs.RackCabinetId != 0 {
					r.Hw.RackCabinetId = bs.RackCabinetId
				}
			} else {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - блейд-корзина", bladeRefName, "не найдена (см. /admin/importblades) - RackCabinetId сервера взят из разбора \"Местоположение\" этой строки, если получилось")
			}
		}

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
			if _, err := trn.Insert(&r.Hw); err != nil {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка вставки hw:", err)
				trn.Rollback()
				continue
			}
		} else {
			// Запись уже существует - обновляем только поля, которые
			// однозначно должен приносить импорт (место в стойке, корзина,
			// слот, форм-фактор, состояние питания, источник обновления) -
			// см. тот же принцип приоритета ручных правок, что и в
			// ImportItiliumServer.
			r.Hw.Id = hw.Id
			hw.RackCabinetId = r.Hw.RackCabinetId
			hw.BladeSystemId = r.Hw.BladeSystemId
			hw.BladeNum = r.Hw.BladeNum
			hw.FormFactor = r.Hw.FormFactor
			hw.PowerState = r.Hw.PowerState
			hw.SourceUpdateType = r.Hw.SourceUpdateType
			if _, err := trn.Update(&hw, "RackCabinetId", "BladeSystemId", "BladeNum", "FormFactor", "PowerState", "SourceUpdateType"); err != nil {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка обновления hw:", err)
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
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка вставки sprsrv:", err)
				trn.Rollback()
				continue
			}
			r.Spr.Id = sprId
			r.Hw.ServerId = sprId
			if _, err := trn.Update(&r.Hw, "ServerId"); err != nil {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка обновления hw.ServerId:", err)
				trn.Rollback()
				continue
			}
		} else {
			r.Spr.Id = sprsrv.Id
			sprsrv.HWSprId = r.Spr.HWSprId
			sprsrv.Name = r.Spr.Name
			sprsrv.NameFQDN = r.Spr.NameFQDN
			sprsrv.Stat = r.Spr.Stat
			sprsrv.Active = r.Spr.Active
			sprsrv.PowerState = r.Spr.PowerState
			sprsrv.SourceUpdateType = r.Spr.SourceUpdateType
			if _, err := trn.Update(&sprsrv, "HWSprId", "Name", "NameFQDN", "Stat", "Active", "PowerState", "SourceUpdateType"); err != nil {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка обновления sprsrv:", err)
				trn.Rollback()
				continue
			}
		}

		for _, w := range r.Warnings {
			log.Println("itilliumimport:", name, w)
		}

		if err := trn.Commit(); err != nil {
			log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка коммита транзакции:", err)
			continue
		}

		// --- Статистика для страницы-опросчика (progress.go) --------------
		// (2026-09-08, тот же принцип, что для rack-серверов/блейд-корзин).
		job.addResult(!sprFound, r.Spr.Active == 2)
	}
}
