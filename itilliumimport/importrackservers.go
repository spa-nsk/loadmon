package itilliumimport

import (
	"loadmon/locality"
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
	rs := locality.NewResolver() // справочник населённых пунктов (cmdb_locality)

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
		// Площадка определяется разбором "Местоположение" (locality.ParseLocation):
		// населённый пункт -> LocalityCode, описание -> AltName, улица+дом -> Address.
		site, errSite := resolveSite(trn, rs, r.Extra["location_raw"], models.UserSourceManual)
		if errSite != nil {
			log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка резолва площадки:", errSite)
			trn.Rollback()
			continue
		}
		datacenterId := site.Id

		// --- Шкаф (models.RackCabinet) - резолвится ПАРОЙ (Name,
		// DatacenterSiteId), см. resolveRackCabinet: одинаковое имя шкафа
		// допустимо на разных площадках --------------------------------
		//
		// ИЗМЕНЕНО (2026-09-11, тот же баг, что исправлен в
		// importblades.go/importbladeservers.go/importstorages.go):
		// "Размещение в шкафу/стойке" может указывать colocation-площадку
		// ПЕРЕД "/", отличную от "Местоположение" строки - см.
		// resolvecabinet.go:splitColocationCabinetName.
		//
		// ИЗМЕНЕНО (2026-10-19): часть до "/" - АЛЬТЕРНАТИВНОЕ наименование
		// шкафа (RackCabinet.AltName), а не другая площадка; в Name шкафа
		// пишется чистый код. Если "Размещение в шкафу/стойке" пусто, шкаф и
		// юниты берутся из "Местоположение" (Шкаф X | Unit N[-M]).
		cabinetCode, cabinetAlt, locPos := cabinetFromPlacement(r.Extra["rack_num"], site.Loc)
		if cabinetCode != "" {
			rackAnnotation := "Шкаф по данным импорта из Итиллиум, площадка: " + site.Name
			rackId, errRack := resolveRackCabinet(trn, cabinetCode, cabinetAlt, datacenterId, rackAnnotation)
			if errRack != nil {
				log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка резолва rack:", errRack)
				trn.Rollback()
				continue
			}
			r.Hw.RackCabinetId = rackId
			if locPos != nil && r.Hw.StartUnitPos == 0 {
				r.Hw.StartUnitPos = locPos.Start
				if sz := locPos.End - locPos.Start + 1; sz > 0 {
					r.Hw.SizeUnit = sz
				} else {
					r.Hw.SizeUnit = 1
				}
			}
		}

		// --- Статус/питание ----------------------------------------------
		if r.Hw.SerialNum != "" { //если серийный номер отсутствует делаем симуляцию в виде технического номера, чтобы не потерять запись в справочнике серверов
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
		hw = models.ServerHardware{SerialNum: r.Hw.SerialNum} //корректировка поля с проверкой выше
		hwFound := trn.Read(&hw, "SerialNum") == nil          //если запись найдена, то обновляем только поля, которые однозначно должен приносить импорт (место в стойке, состояние питания, источник обновления), НЕ перезаписывая остальные поля существующей записи данными из файла - на случай, если они были вручную скорректированы в интерфейсе после прошлой синхронизации (тот же принцип приоритета ручных правок, что и для СХД, см. kontext/05-conventions-and-gotchas.md). Если нужно перезаписывать больше полей при повторном импорте - перечислите их здесь и передайте именами в trn.Update (сейчас обновляются только явно перечисленные 3 колонки).

		if !hwFound {
			// Новая запись - вставляем указатель на r.Hw (не значение!),
			// иначе автосгенерированный Id не запишется обратно (см.
			// beego/v2/client/orm/orm.go:InsertWithCtx - Insert требует
			// указатель, чтобы установить Id через reflect после вставки).
			r.Hw.FormFactor = 1                  //rack-сервер
			r.Hw.TimeCreate = time.Now().Local() //фиксируем времяя создания и автора
			r.Hw.CreatorUser = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Hw.Time_Mfy = time.Now().Local()
			r.Hw.Id_User = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Hw.SourceUpdateType = models.SourceUpdateManual //вставлен из итилиум, значит его источник - ручной импорт (см. cmdb/updatehwsprhp.go)

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
			// (сейчас обновляются только явно перечисленные 4 колонки).
			r.Hw.Id = hw.Id
			hw.FormFactor = 1                                     //rack-сервер
			if r.Hw.RackCabinetId != 0 {                          //шкаф в выгрузке не указан - существующую привязку не сбрасываем
				hw.RackCabinetId = r.Hw.RackCabinetId
			}
			//обновляем при наличии шкафа, так как мастер система по размещению оборудования в стойках - итиллиум, и если там поменяли стойку, то нужно обновить и у нас
			if hw.SourceUpdateType == models.SourceUpdateManual { //если источник обновления - ручной импорт, то обновляем состояние питания и активность
				hw.PowerState = r.Hw.PowerState
				hw.Active = r.Hw.Active
				if _, err := trn.Update(&hw, "RackCabinetId", "PowerState", "Active"); err != nil {
					log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления hw:", err)
					trn.Rollback()
					continue
				}
			} else {
				if _, err := trn.Update(&hw, "RackCabinetId"); err != nil {
					log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления hw:", err)
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
			//новый сервер не обнаруженный системами сбора метрик и HP One View, значит его источник - ручной импорт (см. cmdb/updatehwsprhp.go)
			r.Spr.TimeCreate = time.Now().Local() //фиксируем времяя создания и автора
			r.Spr.CreatorUser = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Spr.Time_Mfy = time.Now().Local()
			r.Spr.Id_User = models.UserSourceManual //вставка из итиллиум - пользователь models.UserSourceManual=9999 (см. cmdb/updatehwsprhp.go)
			r.Spr.TimeActiveState = time.Now().Local() //фиксируем времяя активации и автора
			// Источник - ручной импорт ВСЕГДА для новой записи (и для статуса
			// "не активен" тоже, как у блейд-серверов в importbladeservers.go):
			// по SourceUpdateManual ночная проверка, почасовое обновление
			// питания и сверка не трогают записи, которых нет в HP OneView/VMware.
			r.Spr.SourceUpdateType = models.SourceUpdateManual

			// Запись справочника серверов без имени бесполезна (в списках
			// "стёртое имя"): при пустом "Имя (hostname)" берём технический
			// номер ИТ, затем серийный номер.
			if strings.TrimSpace(r.Spr.Name) == "" {
				fallback := strings.TrimSpace(r.Hw.TechnicalNumberIT)
				if fallback == "" {
					fallback = strings.TrimSpace(r.Hw.SerialNum)
				}
				r.Spr.Name = fallback
				log.Println("itilliumimport:", name, "- пустое имя сервера, использован", fallback)
			}
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
			//удалено обновление имен так как тут актуально только изменение статусов и питания, а имена могут быть изменены вручную в интерфейсе
			r.Spr.Id = sprsrv.Id
			sprsrv.HWSprId = r.Spr.HWSprId
			if sprsrv.SourceUpdateType == models.SourceUpdateManual {
				sprsrv.Stat = r.Spr.Stat
				sprsrv.Active = r.Spr.Active
				sprsrv.PowerState = r.Spr.PowerState
				if _, err := trn.Update(&sprsrv, "HWSprId", "Stat", "Active", "PowerState"); err != nil {
					log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления sprsrv:", err)
					trn.Rollback()
					continue
				}
			} else {
				if _, err := trn.Update(&sprsrv, "HWSprId"); err != nil {
					log.Println("itilliumimport: ImportItiliumServer(", name, ") - ошибка обновления sprsrv:", err)
					trn.Rollback()
					continue
				}
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
