package itilliumimport

import (
	"loadmon/models"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// mappingConfigFileNameStorage - имя файла соответствий для импорта СХД
// внутри каталога, заданного ключом "yaml" в app.conf (см. mappingConfigDir()
// в importrackservers.go - переиспользуется как есть).
const mappingConfigFileNameStorage = "rack_storage.yaml"

// mappingConfigPathStorage - полный путь к yaml-файлу соответствий, читаемому
// в ImportItiliumStorage. Сам файл нужно положить в каталог, указанный в
// app.conf (ключ "yaml"), под именем mappingConfigFileNameStorage - см.
// server_web_new/conf/yaml/rack_storage.yaml как образец (он же реальный
// файл, если "yaml" в app.conf указывает именно на server_web_new/conf/yaml).
func mappingConfigPathStorage() string {
	return mappingConfigDir() + "/" + mappingConfigFileNameStorage
}

// StartImportItiliumStorage регистрирует новое задание импорта СХД под
// ключом jobId (см. ImportProgress в progress.go) и запускает сам разбор и
// сохранение файла (runImportItiliumStorage) в ОТДЕЛЬНОЙ горутине, сразу
// возвращая указатель на прогресс вызывающему коду (controllers/
// adminimportstorages.go) - тот же приём, что StartImportItiliumBlade
// (importblades.go). Job регистрируется СИНХРОННО, до запуска горутины -
// иначе страница могла бы успеть опросить статус раньше, чем задание
// появится в реестре (GetImportJob вернул бы ok=false).
//
// 2026-09-08, по запросу пользователя "добавь в меню Импорт пункт «Импорт
// СХД из Итилиум» и реализуй импорт по аналогии с блэйд корзинами".
func StartImportItiliumStorage(name string) *ImportProgress {
	job := newImportJob(name)
	go runImportItiliumStorage(name, job)
	return job
}

// ImportItiliumStorage - синхронная версия импорта СХД (без отдельной
// горутины и без прогресса) - оставлена для прямых вызовов вне HTTP-хендлера,
// где отслеживать прогресс не нужно. Текущий HTTP-хендлер использует
// StartImportItiliumStorage выше.
func ImportItiliumStorage(name string) {
	job := &ImportProgress{}
	runImportItiliumStorage(name, job)
}

// runImportItiliumStorage читает xlsx-файл name (внутри каталога "./import",
// см. controllers/adminimportstorages.go), конвертирует его строки через
// ConvertFileStorage/ConvertRowStorage (rack_storage.yaml) и создаёт/обновляет
// записи models.DatacenterSite -> models.RackCabinet -> models.StorageSystems
// -> models.StorageSystemPlacement в БД - полный аналог runImportItiliumBlade
// (importblades.go) для СХД, с двумя ключевыми отличиями:
//
//  1. Поиск существующей записи StorageSystems выполняется ПО СЕРИЙНОМУ
//     НОМЕРУ (models.StorageSystems.SerialNum, колонка "S/N"), а не по имени
//     - серийный номер у СХД в выгрузке Итиллиум заполняется значительно
//     надёжнее, чем короткое мнемоническое имя (см. комментарий в
//     rack_storage.yaml). Строки без S/N пропускаются - создавать
//     "безымянную" по серийнику запись СХД, которую потом нечем будет найти
//     при повторном импорте, смысла нет.
//  2. Размещение в шкафу (2026-09-09, по запросу пользователя "СХД может
//     размещаться в нескольких шкафах и схема со ссылкой
//     StorageSystems.RackCabinetId не позволяет реализовать учет
//     размещения") - НЕ поле самой StorageSystems, а отдельная запись
//     models.StorageSystemPlacement, см. upsertStorageSystemPlacement ниже.
//     ОДНА строка файла может содержать НЕСКОЛЬКО шкафов сразу в колонке
//     "Размещение в шкафу/стойке" (2026-09-10, по запросу пользователя "в
//     импорте СХД, если несколько шкафов то загрузка некорректна... Оно по
//     сути продолжение поля Местоположение") - все сегменты разбираются
//     parseStoragePlacementSegments (storageplacement.go) из сырой строки
//     колонки, и upsertStorageSystemPlacement вызывается по разу на каждый
//     разобранный шкаф (см. цикл по placements ниже).
//
// job (см. progress.go) отражает ход выполнения для страницы-опросчика -
// job.finish(...) вызывается ЛЮБЫМ путём выхода из функции (defer).
func runImportItiliumStorage(name string, job *ImportProgress) {
	var startErr string
	defer func() { job.finish(startErr) }()

	cfg, err := LoadMapping(mappingConfigPathStorage())
	if err != nil {
		log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка загрузки конфигурации соответствий:", err)
		startErr = err.Error()
		return
	}

	results, err := ConvertFileStorage(cfg, "./import/"+name)
	if err != nil {
		log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка чтения/разбора файла:", err)
		startErr = err.Error()
		return
	}
	job.setFileRows(len(results))

	o := orm.NewOrm()

	for _, r := range results {
		job.tickProcessed()

		// r.St       - models.StorageSystems (частично заполнена)
		// r.Extra    - map[string]string: поля без прямого соответствия в
		//              схеме + диагностические "сырые" значения (в т.ч.
		//              status/rack_placement_raw для Anotation,
		//              start_unit_pos/end_unit_pos для размещения ниже)
		// r.Warnings - []string: на что стоит посмотреть глазами перед сохранением
		serialNum := strings.TrimSpace(r.St.SerialNum)
		if serialNum == "" {
			log.Println("itilliumimport: ImportItiliumStorage(", name, ") - пропущена строка без серийного номера (StorageSystems.SerialNum), см. колонку \"S/N\" в rack_storage.yaml")
			continue
		}
		r.St.SerialNum = serialNum

		flNotWorking := false
		if strings.Contains(r.Extra["datacenter_room"], "списан") || strings.Contains(r.Extra["datacenter_room"], "продаж") || strings.Contains(r.Extra["datacenter_room"], "утер") {
			flNotWorking = true
		}

		trn, err := o.Begin()
		if err != nil {
			log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка начала транзакции:", err)
			continue
		}

		datacenterRoom := r.Extra["datacenter_room"]
		// ownSiteId - ЦОД, резолвленный по колонке "Местоположение"
		// (location_split) - используется как площадка ПО УМОЛЧАНИЮ для
		// сегментов размещения без собственного имени ЦОД перед "/" (см.
		// storagePlacementSegment.DatacenterSiteName и разбор ниже).
		ownSiteId := int64(0)
		if !flNotWorking {
			// --- ЦОД (models.DatacenterSite) по "Местоположение" ------------
			datacenter := models.DatacenterSite{
				Name:       datacenterRoom + " " + r.Extra["location_room_code"],
				Annotation: "ЦОД по адресу: " + datacenterRoom + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"],
				Id_User:    9999,
				Time_Mfy:   time.Now().Local(),
			}
			if err := trn.Read(&datacenter, "Name"); err != nil {
				if _, err := trn.Insert(&datacenter); err != nil {
					log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка вставки datacenter:", err)
					trn.Rollback()
					continue
				}
			}
			ownSiteId = datacenter.Id
		}

		// --- Размещение (models.StorageSystemPlacement) - ОДИН или
		// НЕСКОЛЬКО шкафов на одну СХД (2026-09-10, по запросу пользователя
		// "в импорте СХД, если несколько шкафов то загрузка некорректна...
		// не рассматривается поле Размещение в шкафу/стойке. Оно по сути
		// продолжение поля Местоположение") - разбирается напрямую из сырой
		// строки extra.rack_placement_raw (не из старого rack_unit_split,
		// который поддерживал ровно один шкаф на строку файла, см. подробный
		// комментарий у parseStoragePlacementSegments в storageplacement.go).
		var placements []struct {
			rackId int64
			startU int
			endU   int
		}
		if !flNotWorking {
			segments := parseStoragePlacementSegments(r.Extra["rack_placement_raw"])
			for _, seg := range segments {
				siteId := ownSiteId
				if seg.DatacenterSiteName != "" {
					// Colocation-сегмент со своим именем ЦОД перед "/" -
					// резолвится ОТДЕЛЬНО от "Местоположение" (см.
					// resolveDatacenterSiteByName и комментарий у
					// storagePlacementSegment.DatacenterSiteName - имя может
					// текстуально отличаться от ЦОД в "Местоположение" для
					// той же строки, это не ошибка данных, а два разных
					// идентификатора одной площадки).
					var errSite error
					siteId, errSite = resolveDatacenterSiteByName(trn, seg.DatacenterSiteName)
					if errSite != nil {
						log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка резолва ЦОД для сегмента размещения", seg.DatacenterSiteName, ":", errSite)
						continue
					}
				}
				if siteId <= 0 {
					continue
				}
				rackId, errRack := resolveRackCabinet(trn, seg.CabinetCode, siteId,
					"Шкаф по данным импорта СХД из Итиллиум (колонка \"Размещение в шкафу/стойке\")")
				if errRack != nil {
					log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка резолва rack для сегмента", seg.CabinetCode, ":", errRack)
					continue
				}
				placements = append(placements, struct {
					rackId int64
					startU int
					endU   int
				}{rackId: rackId, startU: seg.StartUnitPos, endU: seg.EndUnitPos})
			}
		}

		// --- Anotation: "Статус" (по запросу пользователя) В НАЧАЛЕ, затем
		// сырое значение колонки "Размещение в шкафу/стойке" (если было) -
		// тот же приём, что BladeSystems.Anotation в importblades.go, но с
		// текстовым статусом первым по явному требованию пользователя
		// ("поле «Статус» копируй в начало Anotation").
		status := strings.TrimSpace(r.Extra["status"])
		rawPlacement := strings.TrimSpace(r.Extra["rack_placement_raw"])
		switch {
		case status != "" && rawPlacement != "":
			r.St.Anotation = status + "; " + rawPlacement
		case status != "":
			r.St.Anotation = status
		default:
			r.St.Anotation = rawPlacement
		}

		// --- StorageSystems -------------------------------------------------
		// Поиск существующей записи ПО СЕРИЙНОМУ НОМЕРУ (см. комментарий у
		// runImportItiliumStorage выше и в rack_storage.yaml).
		st := models.StorageSystems{SerialNum: r.St.SerialNum}
		stFound := trn.Read(&st, "SerialNum") == nil

		if !stFound {
			// Новая запись - вставляем указатель на r.St (не значение!),
			// иначе автосгенерированный Id не запишется обратно (см.
			// beego/v2/client/orm/orm.go:InsertWithCtx - Insert требует
			// указатель, чтобы установить Id через reflect после вставки, -
			// та же ошибка была найдена и исправлена в ImportItiliumServer,
			// см. CHANGELOG.md).
			r.St.Id_User = 9999
			r.St.Time_Mfy = time.Now().Local()
			r.St.SourceUpdateType = models.StorageSourceUpdateManual
			if _, err := trn.Insert(&r.St); err != nil {
				log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка вставки st:", err)
				trn.Rollback()
				continue
			}
			st = r.St
		} else {
			// Запись уже существует - обновляем ТОЛЬКО поля, которые
			// однозначно должен приносить импорт (по прямому запросу
			// пользователя: InventoryNum/InventoryNumBuh/TechnicalNumberIT,
			// плюс Anotation со статусом в начале) - НЕ перезаписывая
			// остальные поля существующей записи данными из файла (тот же
			// принцип приоритета ручных правок, что и в ImportItiliumBlade/
			// ImportItiliumServer, см. kontext/05-conventions-and-gotchas.md).
			// Размещение в шкафу (RackCabinetId/StartUnitPos/EndUnitPos)
			// обновляется ОТДЕЛЬНО через upsertStorageSystemPlacement ниже -
			// это уже не поле StorageSystems.
			st.InventoryNum = r.St.InventoryNum
			st.InventoryNumBuh = r.St.InventoryNumBuh
			st.TechnicalNumberIT = r.St.TechnicalNumberIT
			st.Anotation = r.St.Anotation
			st.Id_User = 9999
			st.Time_Mfy = time.Now().Local()
			if _, err := trn.Update(&st, "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "Anotation", "Id_User", "Time_Mfy"); err != nil {
				log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка обновления st:", err)
				trn.Rollback()
				continue
			}
		}

		// --- Размещение (models.StorageSystemPlacement) - один вызов
		// upsertStorageSystemPlacement НА КАЖДЫЙ разобранный шкаф (см.
		// placements выше) - безопасно вызывать несколько раз подряд для
		// одной СХД, см. комментарий у upsertStorageSystemPlacement.
		placementErr := false
		for _, p := range placements {
			if p.rackId <= 0 {
				continue
			}
			if err := upsertStorageSystemPlacement(trn, st.Id, p.rackId, p.startU, p.endU); err != nil {
				log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка сохранения размещения (StorageSystemsId=", st.Id, ", RackCabinetId=", p.rackId, "):", err)
				placementErr = true
				break
			}
		}
		if placementErr {
			trn.Rollback()
			continue
		}

		// TODO: r.Extra содержит поля без прямого соответствия в схеме
		// (legal_entity/sz/repair_date/location_zone_code/location_room_code
		// и любые extra.* из rack_storage.yaml) - на данный момент они нигде
		// не сохраняются (ни в БД, ни в лог) - решить, нужно ли и куда их
		// пристраивать.
		for _, w := range r.Warnings {
			log.Println("itilliumimport:", name, w)
		}

		if err := trn.Commit(); err != nil {
			log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка коммита транзакции:", err)
			continue
		}

		// --- Статистика для страницы-опросчика (progress.go) --------------
		// "Неактивных" в терминах BladeSystems/ServerSprRecords у
		// StorageSystems нет отдельного bool-признака (Status - числовое
		// перечисление 0-6 без прямого соответствия импортируемому
		// текстовому статусу, см. rack_storage.yaml) - считаем "ушедшими в
		// неактивные" записи, у которых распознан явный признак списания/
		// вывода из эксплуатации по тексту "Местоположение" (flNotWorking) -
		// тот же критерий, что использован выше для решения не размещать
		// такую запись в шкафу.
		job.addResult(!stFound, flNotWorking)
	}
}

// upsertStorageSystemPlacement заводит НОВУЮ запись models.
// StorageSystemPlacement (StorageSystemsId, RackCabinetId, StartUnitPos,
// EndUnitPos, вычисленный SizeUnit) либо обновляет позицию (Start/End/
// SizeUnit) уже существующей - если у этой СХД уже есть размещение именно в
// этом шкафу (совпадение по паре StorageSystemsId+RackCabinetId).
//
// НЕ трогает размещения этой же СХД в ДРУГИХ шкафах - специально
// (2026-09-09, по запросу пользователя "СХД может размещаться в нескольких
// шкафах"): вызывать эту функцию несколько раз для одной СХД (по одному
// разу на каждый шкаф) безопасно и ничего не затирает, в отличие от
// "удалить всё старое размещение и вставить заново" (та схема используется
// в ручной форме - saveStorageSystemPlacements в controllers/
// sprstoragesystems.go, - где весь набор размещений приходит одним POST'ом
// целиком; здесь же импорт обрабатывает файл ПОСТРОЧНО, и у него нет
// целостной картины "весь набор шкафов этой СХД" за один вызов).
//
// IsPrimary=true проставляется ТОЛЬКО первому размещению этой СХД (когда до
// вызова у неё не было вообще ни одного) - при последующих вызовах (для той
// же СХД, но другого шкафа) новые записи заводятся с IsPrimary=false, чтобы
// не создавать несколько "основных" мест одновременно.
func upsertStorageSystemPlacement(trn orm.TxOrmer, storageId, rackCabinetId int64, startUnitPos, endUnitPos int) error {
	if rackCabinetId <= 0 {
		return nil
	}
	sizeUnit := 0
	if endUnitPos >= startUnitPos && startUnitPos > 0 {
		sizeUnit = endUnitPos - startUnitPos + 1
	}

	var existing models.StorageSystemPlacement
	err := trn.QueryTable(new(models.StorageSystemPlacement)).
		Filter("StorageSystemsId", storageId).
		Filter("RackCabinetId", rackCabinetId).
		One(&existing)
	if err == nil {
		existing.StartUnitPos = startUnitPos
		existing.EndUnitPos = endUnitPos
		existing.SizeUnit = sizeUnit
		existing.Time_Mfy = time.Now().Local()
		_, updErr := trn.Update(&existing, "StartUnitPos", "EndUnitPos", "SizeUnit", "Time_Mfy")
		return updErr
	}

	isPrimary := false
	if cnt, cErr := trn.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId", storageId).Count(); cErr == nil && cnt == 0 {
		isPrimary = true
	}
	p := models.StorageSystemPlacement{
		StorageSystemsId: storageId,
		RackCabinetId:    rackCabinetId,
		StartUnitPos:     startUnitPos,
		EndUnitPos:       endUnitPos,
		SizeUnit:         sizeUnit,
		IsPrimary:        isPrimary,
		Id_User:          9999,
		Time_Mfy:         time.Now().Local(),
	}
	_, insErr := trn.Insert(&p)
	return insErr
}
