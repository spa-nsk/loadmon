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
//     "Размещение в шкафу/стойке", т.е. extra["rack_num"] и так был бы
//     пустым).
//     Порядок обработки места установки - СНАЧАЛА стандартный разбор
//     "Местоположение" в DatacenterSite/RackCabinet (тот же код, что и в
//     ImportItiliumServer - "аналогично обычным серверам", как просил
//     пользователь), ЗАТЕМ, если блейд-корзина найдена - её RackCabinetId
//     имеет приоритет (см. шаг "Привязка к блейд-корзине" ниже). Если
//     корзина не найдена (например, ещё не импортирована через
//     /admin/importblades) - используется результат стандартного разбора,
//     событие логируется как предупреждение.
//
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
			Name:       r.Extra["location_room_text"] + " " + r.Extra["location_room_code"],
			Annotation: "ЦОД по адресу: " + r.Extra["location_room_text"] + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"],
			Id_User:    models.UserSourceManual,
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

		// --- Шкаф (models.RackCabinet) - см. выше, extra["rack_num"] в этой
		// выгрузке обычно пуст (нет колонки "Размещение в шкафу/стойке"),
		// поэтому find-or-create по пустому имени намеренно пропускается
		// (иначе завели бы мусорную запись RackCabinet с пустым Name) -
		// реальная привязка к шкафу приходит ниже, через блейд-корзину.
		if strings.TrimSpace(r.Extra["rack_num"]) != "" {
			// Резолвится ПАРОЙ (Name, DatacenterSiteId), см.
			// resolveRackCabinet: одинаковое имя шкафа допустимо на разных
			// площадках.
			//
			// ИЗМЕНЕНО (2026-09-11, тот же баг, что исправлен в
			// importblades.go/importstorages.go): "Размещение в
			// шкафу/стойке" может указывать colocation-площадку ПЕРЕД "/",
			// отличную от "Местоположение" строки - см.
			// resolvecabinet.go:splitColocationCabinetName.
			colocationDc, cabinetCode := splitColocationCabinetName(r.Extra["rack_num"])
			rackSiteId := datacenterId
			rackAnnotation := "Шкаф по адресу: " + r.Extra["location_room_text"] + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"]
			if colocationDc != "" {
				var errSite error
				rackSiteId, errSite = resolveDatacenterSiteByName(trn, colocationDc)
				if errSite != nil {
					log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка резолва colocation ЦОД", colocationDc, ":", errSite)
					trn.Rollback()
					continue
				}
				rackAnnotation = "Шкаф на colocation-площадке \"" + colocationDc + "\" (по данным импорта из Итиллиум)"
			}
			rackId, errRack := resolveRackCabinet(trn, cabinetCode, rackSiteId, rackAnnotation)
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
				// (FK). Текстовые DatacenterRoom/RackNum у ServerHardware
				// также удалены (2026-10-01) - их текст сохраняется только
				// в r.Extra["location_room_text"]/r.Extra["rack_num"].
				if bs.RackCabinetId != 0 {
					r.Hw.RackCabinetId = bs.RackCabinetId
				}
			} else {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - блейд-корзина", bladeRefName, "не найдена (см. /admin/importblades) - RackCabinetId сервера взят из разбора \"Местоположение\" этой строки, если получилось")
			}
		}

		// --- Статус/питание ----------------------------------------------
		if r.Hw.SerialNum != "" {
			r.Spr.SerialNumber = r.Hw.SerialNum
		} else {
			r.Spr.SerialNumber = r.Hw.TechnicalNumberIT
			r.Hw.SerialNum = r.Hw.TechnicalNumberIT
		}

		if r.Spr.Stat == 1 || r.Spr.Active == 1 {
			r.Spr.PowerState = 1
			r.Hw.PowerState = 1
			r.Spr.Stat = 1
			r.Spr.Active = 1
			r.Hw.Active = 1
			r.Spr.ActivatorUser = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Spr.SourceUpdateType = models.SourceUpdateManual //вставлен из итилиум, значит его источник - ручной импорт (см. cmdb/updatehwsprhp.go)
		} else {
			r.Spr.PowerState = 2
			r.Hw.PowerState = 2
			r.Spr.Stat = models.ServerSprStatUsed // 2026-10-02: Stat=0 ставит только человек; "не активен" = Stat=1, Active=2
			r.Spr.Active = 2
			r.Hw.Active = 2
			r.Spr.TimeDeActiveState = time.Now().Local()
			r.Spr.DeActivatorUser = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
		}

		// --- ServerHardware ------------------------------------------------
		var hw models.ServerHardware
		hw = models.ServerHardware{SerialNum: r.Hw.SerialNum}
		hwFound := trn.Read(&hw, "SerialNum") == nil
		if !hwFound {
			r.Hw.FormFactor = 2                  // блейд (1-rack, 2-blade, см. ServerHardware.FormFactor)
			r.Hw.TimeCreate = time.Now().Local() //фиксируем времяя создания и автора
			r.Hw.CreatorUser = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Hw.Time_Mfy = time.Now().Local()
			r.Hw.Id_User = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Hw.SourceUpdateType = models.SourceUpdateManual

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

			if hw.SourceUpdateType == models.SourceUpdateManual { //если источник обновления - ручной импорт, то обновляем состояние питания и активность
				hw.PowerState = r.Hw.PowerState
				hw.Active = r.Hw.Active
				if _, err := trn.Update(&hw, "RackCabinetId", "BladeSystemId", "BladeNum", "FormFactor", "PowerState", "SourceUpdateType", "Active"); err != nil {
					log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка обновления hw:", err)
					trn.Rollback()
					continue
				}
			} else {
				if _, err := trn.Update(&hw, "RackCabinetId"); err != nil {
					log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка обновления hw:", err)
					trn.Rollback()
					continue
				}
			}
		}

		// --- ServerSprRecords -----------------------------------------------
		r.Spr.HWSprId = r.Hw.Id

		var sprsrv models.ServerSprRecords
		sprFound := hw.ServerId != 0
		if sprFound {
			sprsrv = models.ServerSprRecords{Id: hw.ServerId}
			sprFound = trn.Read(&sprsrv) == nil && !models.IsServerSprDestroyed(&sprsrv) // запись Stat=666 считается отсутствующей
		}
		if !sprFound {
			r.Spr.TimeCreate = time.Now().Local() //фиксируем времяя создания и автора
			r.Spr.CreatorUser = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Spr.Time_Mfy = time.Now().Local()
			r.Spr.Id_User = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Spr.SourceUpdateType = models.SourceUpdateManual
			sprId, err := trn.Insert(&r.Spr)
			if err != nil {
				log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка вставки sprsrv:", err)
				trn.Rollback()
				continue
			}
			r.Spr.Id = sprId
			if !hwFound { //если hwFound == false, то r.Hw.Id уже содержит Id вставленной записи, иначе hw.Id содержит Id существующей записи
				r.Hw.ServerId = sprId
				if _, err := trn.Update(&r.Hw, "ServerId"); err != nil {
					log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления r.Hw.ServerId:", err)
					trn.Rollback()
					continue
				}
			} else {
				hw.ServerId = sprId
				if _, err := trn.Update(&hw, "ServerId"); err != nil {
					log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления hw.ServerId:", err)
					trn.Rollback()
					continue
				}
			}
		} else {
			r.Spr.Id = sprsrv.Id
			sprsrv.HWSprId = r.Spr.HWSprId
			if sprsrv.SourceUpdateType == models.SourceUpdateManual {
				sprsrv.Stat = r.Spr.Stat
				sprsrv.Active = r.Spr.Active
				sprsrv.PowerState = r.Spr.PowerState
				if _, err := trn.Update(&sprsrv, "HWSprId", "Stat", "Active", "PowerState", "SourceUpdateType"); err != nil {
					log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка обновления sprsrv:", err)
					trn.Rollback()
					continue
				}
			} else {
				if _, err := trn.Update(&sprsrv, "HWSprId"); err != nil {
					log.Println("itilliumimport: ImportItiliumBladeServer(", name, ") - ошибка обновления sprsrv:", err)
					trn.Rollback()
					continue
				}
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
