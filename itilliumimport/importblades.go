package itilliumimport

import (
	"loadmon/grpcclient"
	"loadmon/models"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// mappingConfigFileNameBlade - имя файла соответствий для импорта
// блейд-корзин внутри каталога, заданного ключом "yaml" в app.conf (см.
// mappingConfigDir() в importrackservers.go - переиспользуется как есть).
const mappingConfigFileNameBlade = "blade_systems.yaml"

// mappingConfigPathBlade - полный путь к yaml-файлу соответствий, читаемому
// в ImportItiliumBlade. Сам файл нужно положить в каталог, указанный в
// app.conf (ключ "yaml"), под именем mappingConfigFileNameBlade - см.
// itilliumimport/mappingblade.yaml как образец.
func mappingConfigPathBlade() string {
	return mappingConfigDir() + "/" + mappingConfigFileNameBlade
}

// StartImportItiliumBlade регистрирует новое задание импорта блейд-корзин
// под ключом jobId (см. ImportProgress в progress.go) и запускает сам разбор
// и сохранение файла (runImportItiliumBlade) В ОТДЕЛЬНОЙ горутине, сразу
// возвращая указатель на прогресс вызывающему коду (controllers/
// adminimportblades.go) - 2026-09-08, по запросу пользователя "отображать
// процесс импорта в виде градуированного линейного индикатора с процентами".
// Job регистрируется СИНХРОННО, до запуска горутины - иначе страница могла
// бы успеть опросить статус раньше, чем задание появится в реестре
// (GetImportJob вернул бы ok=false).
func StartImportItiliumBlade(name string) *ImportProgress {
	job := newImportJob(name)
	go runImportItiliumBlade(name, job)
	return job
}

// ImportItiliumBlade - синхронная версия импорта блейд-корзин (без отдельной
// горутины и без прогресса) - оставлена для прямых вызовов вне HTTP-хендлера
// (например, из будущих неинтерактивных сценариев/тестов), где отслеживать
// прогресс не нужно и вызывающий код готов дождаться завершения. Текущий
// HTTP-хендлер использует StartImportItiliumBlade выше.
func ImportItiliumBlade(name string) {
	job := &ImportProgress{}
	runImportItiliumBlade(name, job)
}

// runImportItiliumBlade читает xlsx-файл name (внутри каталога "./import",
// см. controllers/adminimportblades.go), конвертирует его строки через
// ConvertFileBlade/ConvertRowBlade (rack_blade.yaml) и создаёт/обновляет
// записи models.DatacenterSite -> models.RackCabinet -> models.BladeSystems
// в БД - полный аналог ImportItiliumServer (importrackservers.go) для
// блейд-корзин, с одним ключевым отличием: поиск существующей записи
// BladeSystems выполняется по Name (короткое мнемоническое имя вида
// "C7000-01", "C7000-02" - именно так это принято в справочнике блейд-корзин
// в отличие от серверов, для которых поиск идёт по серийному номеру
// (S/N у шасси часто не заполняется в Итиллиум так же аккуратно, как у
// серверов, а Name - устойчивый, заранее согласованный идентификатор).
//
// job (см. progress.go) отражает ход выполнения для страницы-опросчика -
// job.finish(...) вызывается ЛЮБЫМ путём выхода из функции (defer), поэтому
// индикатор прогресса на странице не может "зависнуть", даже если импорт не
// смог даже начаться (ошибка загрузки конфигурации/чтения файла).
func runImportItiliumBlade(name string, job *ImportProgress) {
	var startErr string
	defer func() { job.finish(startErr) }()

	cfg, err := LoadMapping(mappingConfigPathBlade())
	if err != nil {
		log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка загрузки конфигурации соответствий:", err)
		startErr = err.Error()
		return
	}

	results, err := ConvertFileBlade(cfg, "./import/"+name)
	if err != nil {
		log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка чтения/разбора файла:", err)
		startErr = err.Error()
		return
	}
	job.setFileRows(len(results))

	o := orm.NewOrm()

	for _, r := range results {
		job.tickProcessed()

		// r.Bs       - models.BladeSystems (частично заполнена)
		// r.Extra    - map[string]string: поля без прямого соответствия в
		//              схеме + диагностические "сырые" значения
		// r.Warnings - []string: на что стоит посмотреть глазами перед сохранением
		if r.Bs.Name == "" {
			log.Println("itilliumimport: ImportItiliumBlade(", name, ") - пропущена строка без имени блейд-шасси (BladeSystems.Name), см. колонку \"Имя блейд-шасси\" в rack_blade.yaml")
			continue
		}

		r.Bs.PowerState = 1 //по умолчанию считаем, что корзина включена (PowerState=1), если в выгрузке не указано обратное
		r.Bs.Active = 1     //по умолчанию считаем, что корзина активна (Active=1), если в выгрузке не указано обратное

		if r.Bs.EndUnitPos >= r.Bs.StartUnitPos && r.Bs.StartUnitPos > 0 {
			r.Bs.SizeUnit = r.Bs.EndUnitPos - r.Bs.StartUnitPos + 1
		}
		flNotWorking := false

		if strings.Contains(r.Extra["datacenter_room"], "списан") || strings.Contains(r.Extra["datacenter_room"], "продаж") || strings.Contains(r.Extra["datacenter_room"], "утер") {
			flNotWorking = true
		}
		trn, err := o.Begin()
		if err != nil {
			log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка начала транзакции:", err)
			continue
		}

		datacenterRoom := r.Extra["datacenter_room"]
		rackId := int64(0)
		if !flNotWorking {
			// --- ЦОД (models.DatacenterSite) --------------------------------
			datacenter := models.DatacenterSite{
				Name:       datacenterRoom + " " + r.Extra["location_room_code"],
				Annotation: "ЦОД по адресу: " + datacenterRoom + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"],
				Id_User:    9999,
				Time_Mfy:   time.Now().Local(),
			}
			if err := trn.Read(&datacenter, "Name"); err != nil {
				if _, err := trn.Insert(&datacenter); err != nil {
					log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка вставки datacenter:", err)
					trn.Rollback()
					continue
				}
			}
			datacenterId := datacenter.Id

			// --- Шкаф (models.RackCabinet) - резолвится ПАРОЙ (Name,
			// DatacenterSiteId), см. resolveRackCabinet: одинаковое имя
			// шкафа допустимо на разных площадках -------------------------
			var errRack error
			rackId, errRack = resolveRackCabinet(trn, r.Extra["rack_num"], datacenterId,
				"Шкаф по адресу: "+datacenterRoom+" этаж:"+r.Extra["location_zone_code"]+" комната: "+r.Extra["location_room_code"])
			if errRack != nil {
				log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка резолва rack:", errRack)
				trn.Rollback()
				continue
			}
		}
		r.Bs.RackCabinetId = rackId

		rackNum := strings.TrimSpace(r.Extra["rack_num"])
		r.Bs.Anotation = strings.TrimSpace(r.Extra["datacenter_room"])
		if rackNum == "" || r.Bs.RackCabinetId == 0 || flNotWorking {
			// Размещение в шкафу не указано в выгрузке - корзина считается
			// выключенной/выведенной из эксплуатации, пока это не будет
			// скорректировано вручную или последующим импортом с заполненным
			// размещением.
			r.Bs.PowerState = 2
			r.Bs.Active = 2
		}

		// --- BladeSystems -------------------------------------------------
		// Поиск существующей записи ПО ИМЕНИ (не по серийному номеру, как у
		// ServerHardware) - см. комментарий у ImportItiliumBlade выше.
		bs := models.BladeSystems{Name: r.Bs.Name}
		bsFound := trn.Read(&bs, "Name") == nil
		var bsId int64 // Id сохранённой записи - нужен ниже для refreshBladeSystemFromOneView

		if !bsFound {
			// Новая запись - вставляем указатель на r.Bs (не значение!),
			// иначе автосгенерированный Id не запишется обратно (см.
			// beego/v2/client/orm/orm.go:InsertWithCtx - Insert требует
			// указатель, чтобы установить Id через reflect после вставки, -
			// та же ошибка была найдена и исправлена в ImportItiliumServer,
			// см. CHANGELOG.md).
			r.Bs.Id_User = 9999
			r.Bs.Time_Mfy = time.Now().Local()
			if _, err := trn.Insert(&r.Bs); err != nil {
				log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка вставки bs:", err)
				trn.Rollback()
				continue
			}
			bsId = r.Bs.Id
		} else {
			// Запись уже существует - обновляем только поля, которые
			// однозначно должен приносить импорт (место в стойке, статус по
			// наличию размещения, примечание с сырым значением размещения),
			// НЕ перезаписывая остальные поля существующей записи данными из
			// файла - на случай, если они были вручную скорректированы в
			// интерфейсе после прошлой синхронизации (тот же принцип
			// приоритета ручных правок, что и в ImportItiliumServer - см.
			// kontext/05-conventions-and-gotchas.md). Если нужно
			// перезаписывать больше полей при повторном импорте -
			// перечислите их здесь и передайте именами в trn.Update.
			bs.RackCabinetId = r.Bs.RackCabinetId
			bs.PowerState = r.Bs.PowerState
			bs.Active = r.Bs.Active
			bs.Anotation = r.Bs.Anotation
			bs.InventoryNum = r.Bs.InventoryNum
			bs.InventoryNumBuh = r.Bs.InventoryNumBuh
			bs.TechnicalNumberIT = r.Bs.TechnicalNumberIT
			// ИСПРАВЛЕНО (2026-09-08): StartUnitPos/EndUnitPos/SizeUnit
			// вычислялись и присваивались (частично) и раньше, но не
			// попадали в список колонок trn.Update ниже - при повторном
			// импорте существующей записи они молча не сохранялись.
			bs.StartUnitPos = r.Bs.StartUnitPos
			bs.EndUnitPos = r.Bs.EndUnitPos
			bs.SizeUnit = r.Bs.SizeUnit
			bs.Id_User = 9999
			bs.Time_Mfy = time.Now().Local()
			if _, err := trn.Update(&bs, "RackCabinetId", "PowerState", "Active", "Anotation", "StartUnitPos", "EndUnitPos", "SizeUnit", "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "Id_User", "Time_Mfy"); err != nil {
				log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка обновления bs:", err)
				trn.Rollback()
				continue
			}
			bsId = bs.Id
		}

		// TODO: r.Extra содержит поля без прямого соответствия в схеме
		// (location_zone_code/location_room_code и любые extra.* из
		// rack_blade.yaml) - на данный момент они нигде не сохраняются (ни
		// в БД, ни в лог) - решить, нужно ли и куда их пристраивать.
		for _, w := range r.Warnings {
			log.Println("itilliumimport:", name, w)
		}

		if err := trn.Commit(); err != nil {
			log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка коммита транзакции:", err)
			continue
		}

		// --- Актуализация PowerState/Active/Used по факту из HPE OneView --
		// (2026-09-08, по запросу пользователя "актуализация состояния блэйд
		// корзин после импорта данных из итиллиум через grpc по серийному
		// номеру") - см. комментарий у refreshBladeSystemFromOneView ниже.
		// finalActive - лучшее известное на данный момент значение Active
		// (из OneView, если корзина там найдена и state пришёл определённый,
		// иначе - как проставлено импортом из Итиллиум выше) - используется
		// только для статистики "ушло в неактивные" (job.addResult ниже).
		finalActive := r.Bs.Active
		if refreshedActive, ok := refreshBladeSystemFromOneView(name, bsId, r.Bs.SerialNum); ok {
			finalActive = refreshedActive
		}

		// --- Статистика для страницы-опросчика (progress.go) --------------
		// (2026-09-08, по запросу пользователя "сколько всего импортировано,
		// сколько новых, сколько обновлено, какое количество ушло в
		// неактивные").
		job.addResult(!bsFound, finalActive == 2)
	}
}

// refreshBladeSystemFromOneView актуализирует поля PowerState/Active/Used
// только что сохранённой записи models.BladeSystems (bsId) фактическим
// состоянием оборудования - запрашивает корзину по серийному номеру serialNum
// через gRPC (grpcclient.GetBladeEnclosureBySN, см. protocol/vm.proto:
// GetBladeEnclosureBySN и grpcserver/handler_hardware.go) у сервиса,
// агрегирующего данные HPE OneView (см. oneview2.Infrastructure.EnclosureBySN).
//
// Смысл: выгрузка Итиллиум не знает фактического состояния оборудования -
// PowerState/Active, проставленные выше в ImportItiliumBlade, это лишь
// допущение по наличию размещения в шкафу/признаку списания в тексте
// местоположения (см. комментарии там), а не реальный статус. Если корзина с
// таким серийным номером заведена в HPE OneView, эта функция перезаписывает
// PowerState/Active РЕАЛЬНЫМ health-статусом/состоянием ресурса (см.
// bladePowerActiveFromEnclosure) и Used - фактическим количеством занятых
// отсеков (по числу известных серверов в enc.ByBay, см.
// messageToEnclosureSummary в grpcclient/client_oneview.go).
//
// Если корзина в OneView не найдена (не-HPE оборудование, чисто ручной учёт
// и т.п. - обычный случай для части блейд-корзин) или у неё пока нет
// серийного номера в выгрузке Итиллиум - функция ничего не меняет, оставляя
// значения из ImportItiliumBlade единственным источником. Ошибки самого
// gRPC-вызова (сервис недоступен и т.п.) также не прерывают импорт - только
// логируются, см. grpcclient.GetBladeEnclosureBySN.
//
// Возвращает (active, true), если Active реально был обновлён по данным
// OneView (нужно вызывающему коду runImportItiliumBlade только для
// статистики "ушло в неактивные" - job.addResult) - (0, false), если
// обновления не было (корзина не найдена/нет серийного номера/state не
// пришёл определённым).
func refreshBladeSystemFromOneView(importName string, bsId int64, serialNum string) (active int, updated bool) {
	serialNum = strings.TrimSpace(serialNum)
	if bsId == 0 || serialNum == "" {
		return 0, false
	}

	enc, ok := grpcclient.GetBladeEnclosureBySN(serialNum)
	if !ok || enc == nil {
		log.Println("itilliumimport: ImportItiliumBlade(", importName, ") - корзина с S/N", serialNum, "не найдена в HPE OneView, PowerState/Active/Used оставлены как после импорта из Итиллиум")
		return 0, false
	}

	powerState, enclosureActive := bladePowerActiveFromEnclosure(enc.Status, enc.State)
	used := 0
	for _, srv := range enc.ByBay {
		if srv == nil {
			continue
		}
		if strings.TrimSpace(srv.Identity.SerialNumber) != "" || strings.TrimSpace(srv.Identity.Name) != "" {
			used++
		}
	}

	bs := models.BladeSystems{Id: bsId}
	o2 := orm.NewOrm()
	if err := o2.Read(&bs); err != nil {
		log.Println("itilliumimport: ImportItiliumBlade(", importName, ") - ошибка чтения bs Id=", bsId, "для актуализации по OneView:", err)
		return 0, false
	}

	// Used обновляется всегда (0 - тоже достоверный факт: корзина заведена в
	// OneView, но в ней нет ни одного известного сервера) - PowerState/Active
	// затираются, только если по ним пришло определённое значение (см.
	// bladePowerActiveFromEnclosure - 0 означает "неизвестно", такой нуль
	// затирать уже проставленное ImportItiliumBlade значение не должен).
	cols := []string{"Used", "Time_Mfy"}
	bs.Used = used
	bs.Time_Mfy = time.Now().Local()
	if powerState != 0 {
		bs.PowerState = powerState
		cols = append(cols, "PowerState")
	}
	if enclosureActive != 0 {
		bs.Active = enclosureActive
		cols = append(cols, "Active")
	}

	if _, err := o2.Update(&bs, cols...); err != nil {
		log.Println("itilliumimport: ImportItiliumBlade(", importName, ") - ошибка обновления bs Id=", bsId, "по данным OneView:", err)
		return 0, false
	}

	if enclosureActive != 0 {
		return enclosureActive, true
	}
	return 0, false
}

// bladePowerActiveFromEnclosure - тот же приём, что
// cmdb/initrec.go:bladeSystemPowerActiveFromEnclosure (не экспортируется
// оттуда, поэтому продублирован здесь небольшой независимой функцией, а не
// импортом пакета cmdb - см. README.md про зависимости пакета itilliumimport).
// Выводит числовые коды BladeSystems.PowerState/Active (0 не указано,
// PowerState 1 включен/2 выключен, Active 1 активен/2 выведен из
// эксплуатации) из health-статуса/state ресурса HPE OneView. Пустое значение
// status/state (GetEnclosure не отработал ни для одного сервера корзины)
// даёт 0 ("не указано") - вызывающий код не должен затирать этим нулём уже
// известное значение.
func bladePowerActiveFromEnclosure(status, state string) (powerState, active int) {
	if status != "" {
		if status == "OK" {
			powerState = 1
		} else {
			powerState = 2
		}
	}
	if state != "" {
		if state == "Removed" {
			active = 2
		} else {
			active = 1
		}
	}
	return powerState, active
}
