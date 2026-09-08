package itilliumimport

import (
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

// ImportItiliumBlade читает xlsx-файл name (внутри каталога "./import", см.
// controllers/adminimportblades.go), конвертирует его строки через
// ConvertFileBlade/ConvertRowBlade (rack_blade.yaml) и создаёт/обновляет
// записи models.DatacenterSite -> models.RackCabinet -> models.BladeSystems
// в БД - полный аналог ImportItiliumServer (importrackservers.go) для
// блейд-корзин, с одним ключевым отличием: поиск существующей записи
// BladeSystems выполняется по Name (короткое мнемоническое имя вида
// "C7000-01", "C7000-02" - именно так это принято в справочнике блейд-корзин
// в отличие от серверов, для которых поиск идёт по серийному номеру
// (S/N у шасси часто не заполняется в Итиллиум так же аккуратно, как у
// серверов, а Name - устойчивый, заранее согласованный идентификатор).
func ImportItiliumBlade(name string) {
	cfg, err := LoadMapping(mappingConfigPathBlade())
	if err != nil {
		log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка загрузки конфигурации соответствий:", err)
		return
	}

	results, err := ConvertFileBlade(cfg, "./import/"+name)
	if err != nil {
		log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка чтения/разбора файла:", err)
		return
	}

	o := orm.NewOrm()

	for _, r := range results {
		// r.Bs       - models.BladeSystems (частично заполнена)
		// r.Extra    - map[string]string: поля без прямого соответствия в
		//              схеме + диагностические "сырые" значения
		// r.Warnings - []string: на что стоит посмотреть глазами перед сохранением
		if r.Bs.Name == "" {
			log.Println("itilliumimport: ImportItiliumBlade(", name, ") - пропущена строка без имени блейд-шасси (BladeSystems.Name), см. колонку \"Имя блейд-шасси\" в rack_blade.yaml")
			continue
		}

		// ДОБАВЛЕНО (2026-09-08, по запросу пользователя): SizeUnit больше не
		// приходит из выгрузки напрямую (см. rack_blade.yaml: composite
		// rack_unit_split теперь пишет EndUnitPos вместо готового SizeUnit) -
		// вычисляется здесь по StartUnitPos/EndUnitPos перед сохранением.
		// >= (не строго >), т.к. шасси на один юнит (StartUnitPos==EndUnitPos)
		// - валидный случай (SizeUnit=1), а не ошибка разбора.
		if r.Bs.EndUnitPos >= r.Bs.StartUnitPos && r.Bs.StartUnitPos > 0 {
			r.Bs.SizeUnit = r.Bs.EndUnitPos - r.Bs.StartUnitPos + 1
		}

		trn, err := o.Begin()
		if err != nil {
			log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка начала транзакции:", err)
			continue
		}

		// ИЗМЕНЕНО (2026-09-08): models.BladeSystems лишился текстовых
		// DatacenterRoom/RackNum - теперь эти значения попадают в r.Extra
		// (см. rack_blade.yaml: extra.datacenter_room/extra.rack_num), а не
		// в поля r.Bs напрямую.
		datacenterRoom := r.Extra["datacenter_room"]

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

		// --- Шкаф (models.RackCabinet) ----------------------------------
		rack := models.RackCabinet{
			Name:             r.Extra["rack_num"],
			DatacenterSiteId: datacenterId,
			Annotation:       "Шкаф по адресу: " + datacenterRoom + " этаж:" + r.Extra["location_zone_code"] + " комната: " + r.Extra["location_room_code"],
			Id_User:          9999,
			Time_Mfy:         time.Now().Local(),
		}
		if err := trn.Read(&rack, "Name"); err != nil {
			if _, err := trn.Insert(&rack); err != nil {
				log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка вставки rack:", err)
				trn.Rollback()
				continue
			}
		} else if rack.DatacenterSiteId != datacenterId {
			// Шкаф уже существовал (например, из более ранней выгрузки или
			// из импорта серверов) - подтягиваем актуальный ЦОД, если он
			// изменился.
			rack.DatacenterSiteId = datacenterId
			if _, err := trn.Update(&rack, "DatacenterSiteId"); err != nil {
				log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка обновления rack:", err)
				trn.Rollback()
				continue
			}
		}
		r.Bs.RackCabinetId = rack.Id

		// --- Статус по наличию размещения в шкафу (2026-09-08, по запросу
		// пользователя) ---------------------------------------------------
		// rack_placement_raw - сырое (нетронутое composite-разбором) значение
		// колонки "Размещение в шкафу/стойке" (см. extra.rack_placement_raw в
		// rack_blade.yaml) - используется здесь как признак "размещение в
		// шкафу указано" И как есть копируется в Anotation (текстовое поле
		// BladeSystems.RackNum, которое раньше хранило часть этого значения,
		// из схемы убрано - см. CHANGELOG 2026-09-08).
		rackPlacementRaw := strings.TrimSpace(r.Extra["rack_placement_raw"])
		r.Bs.Anotation = rackPlacementRaw
		if rackPlacementRaw == "" {
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
			// ИСПРАВЛЕНО (2026-09-08): StartUnitPos/EndUnitPos/SizeUnit
			// вычислялись и присваивались (частично) и раньше, но не
			// попадали в список колонок trn.Update ниже - при повторном
			// импорте существующей записи они молча не сохранялись.
			bs.StartUnitPos = r.Bs.StartUnitPos
			bs.EndUnitPos = r.Bs.EndUnitPos
			bs.SizeUnit = r.Bs.SizeUnit
			bs.Id_User = 9999
			bs.Time_Mfy = time.Now().Local()
			if _, err := trn.Update(&bs, "RackCabinetId", "PowerState", "Active", "Anotation", "StartUnitPos", "EndUnitPos", "SizeUnit", "Id_User", "Time_Mfy"); err != nil {
				log.Println("itilliumimport: ImportItiliumBlade(", name, ") - ошибка обновления bs:", err)
				trn.Rollback()
				continue
			}
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
		}
	}
}
