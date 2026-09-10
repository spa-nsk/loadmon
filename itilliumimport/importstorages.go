package itilliumimport

import (
	"loadmon/models"
	"log"
	"strconv"
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
//     Текущая версия rack_storage.yaml разбирает РОВНО ОДНО размещение на
//     строку файла (колонка "Размещение в шкафу/стойке") - пользователь
//     предупредил, что пришлёт отдельно xlsx-пример с СХД в НЕСКОЛЬКИХ
//     шкафах; когда он появится, эту функцию нужно будет научить проходить
//     по нескольким сегментам размещения и вызывать
//     upsertStorageSystemPlacement для каждого (сама функция уже написана
//     так, что вызывать её несколько раз для одной СХД безопасно - см. её
//     комментарий).
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
					log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка вставки datacenter:", err)
					trn.Rollback()
					continue
				}
			}
			datacenterId := datacenter.Id

			// --- Шкаф (models.RackCabinet) - резолвится ПАРОЙ (Name,
			// DatacenterSiteId), см. resolveRackCabinet: одинаковое имя
			// шкафа допустимо на разных площадках -----------------------
			var errRack error
			rackId, errRack = resolveRackCabinet(trn, r.Extra["rack_num"], datacenterId,
				"Шкаф по адресу: "+datacenterRoom+" этаж:"+r.Extra["location_zone_code"]+" комната: "+r.Extra["location_room_code"])
			if errRack != nil {
				log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка резолва rack:", errRack)
				trn.Rollback()
				continue
			}
		}

		// Юнит начала/конца - разобраны composite rack_unit_split
		// (rack_storage.yaml) в extra.start_unit_pos/extra.end_unit_pos (НЕ
		// в поля StorageSystems - см. комментарий у runImportItiliumStorage
		// выше про переход на models.StorageSystemPlacement).
		startU, _ := strconv.Atoi(strings.TrimSpace(r.Extra["start_unit_pos"]))
		endU, _ := strconv.Atoi(strings.TrimSpace(r.Extra["end_unit_pos"]))

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

		// --- Размещение (models.StorageSystemPlacement) --------------------
		if rackId > 0 {
			if err := upsertStorageSystemPlacement(trn, st.Id, rackId, startU, endU); err != nil {
				log.Println("itilliumimport: ImportItiliumStorage(", name, ") - ошибка сохранения размещения (StorageSystemsId=", st.Id, ", RackCabinetId=", rackId, "):", err)
				trn.Rollback()
				continue
			}
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
