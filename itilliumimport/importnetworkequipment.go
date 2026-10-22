package itilliumimport

import (
	"loadmon/locality"
	"loadmon/models"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// mappingConfigFileNameNet - имя yaml-файла соответствий для импорта сетевого
// оборудования в каталоге, заданном ключом "yaml" в app.conf.
const mappingConfigFileNameNet = "network_equipment.yaml"

func mappingConfigPathNet() string {
	return mappingConfigDir() + "/" + mappingConfigFileNameNet
}

// StartImportItiliumNetworkEquipment регистрирует задание импорта (см.
// ImportProgress) и запускает runImportItiliumNetworkEquipment в горутине.
func StartImportItiliumNetworkEquipment(name string) *ImportProgress {
	job := newImportJob(name)
	go runImportItiliumNetworkEquipment(name, job)
	return job
}

// ImportItiliumNetworkEquipment - синхронная версия импорта (без прогресса).
func ImportItiliumNetworkEquipment(name string) {
	runImportItiliumNetworkEquipment(name, &ImportProgress{})
}

// runImportItiliumNetworkEquipment читает xlsx name (каталог "./import"),
// конвертирует строки через ConvertFileNet (network_equipment.yaml) и создаёт/
// обновляет models.NetworkEquipmentType, models.DatacenterSite,
// models.RackCabinet и models.SwitchSprRecords (справочник сетевого
// оборудования).
//
//   - Тип оборудования ("Тип оборудования") ищется в NetworkEquipmentType по
//     имени, при отсутствии заводится.
//   - Запись ищется по тех. номеру IT, затем по s/n.
//   - Адрес - как у серверов (importrackservers.go): площадка из "Местоположение"
//     (resolveSite), шкаф/юнит из "Размещение в шкафу/стойке", а если оно пусто -
//     из "Шкаф X | Unit N" в "Местоположение" (cabinetFromPlacement). Нет шкафа -
//     служебный шкаф "вне шкафа". Служебные отметки ("! Покинуло ФГ ...")
//     площадку не создают.
//   - Из существующей записи обновляются только поля, приносимые импортом
//     (тип, статус, размещение, учётные номера); остальное - приоритет ручных правок.
func runImportItiliumNetworkEquipment(name string, job *ImportProgress) {
	var startErr string
	defer func() { job.finish(startErr) }()

	cfg, err := LoadMapping(mappingConfigPathNet())
	if err != nil {
		log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка загрузки конфигурации соответствий:", err)
		startErr = err.Error()
		return
	}

	results, err := ConvertFileNet(cfg, "./import/"+name)
	if err != nil {
		log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка чтения/разбора файла:", err)
		startErr = err.Error()
		return
	}
	job.setFileRows(len(results))

	o := orm.NewOrm()
	rs := locality.NewResolver()
	typeCache := map[string]int64{} // имя типа (в нижнем регистре) -> Id, чтобы не искать на каждой строке

	for _, r := range results {
		job.tickProcessed()

		eq := &r.Eq
		eq.TechnicalNumberIT = strings.TrimSpace(eq.TechnicalNumberIT)
		eq.SerialNum = strings.TrimSpace(eq.SerialNum)
		if eq.TechnicalNumberIT == "" && eq.SerialNum == "" {
			log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - пропущена строка без тех. номера IT и s/n")
			continue
		}

		trn, err := o.Begin()
		if err != nil {
			log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка начала транзакции:", err)
			continue
		}

		// --- Тип оборудования ----------------------------------------------
		typeId, err := resolveNetworkEquipmentType(trn, typeCache, r.Extra["type_name"])
		if err != nil {
			log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка резолва типа оборудования:", err)
			trn.Rollback()
			continue
		}
		eq.TypeId = typeId

		// --- Площадка, шкаф, юнит -------------------------------------------
		rawLocation := strings.TrimSpace(r.Extra["location_raw"])
		if !strings.HasPrefix(rawLocation, "!") { // "! Покинуло ФГ ..." - не адрес
			site, errSite := resolveSite(trn, rs, rawLocation, models.UserSourceManual)
			if errSite != nil {
				log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка резолва площадки:", errSite)
				trn.Rollback()
				continue
			}
			eq.DatacenterSiteId = site.Id
			if site.Id > 0 {
				cabinetCode, cabinetAlt, locPos := cabinetFromPlacement(r.Extra["rack_num"], site.Loc)
				if cabinetCode != "" {
					eq.RackCabinetId, err = resolveRackCabinet(trn, cabinetCode, cabinetAlt, site.Id,
						"Шкаф по данным импорта сетевого оборудования из Итиллиум, площадка: "+site.Name)
					if err == nil && locPos != nil && eq.StartUnitPos == 0 {
						eq.StartUnitPos = locPos.Start
						if sz := locPos.End - locPos.Start + 1; sz > 0 {
							eq.SizeUnit = sz
						} else {
							eq.SizeUnit = 1
						}
					}
				} else {
					// Адрес указан, шкафа нет - служебный шкаф "вне шкафа" площадки
					eq.RackCabinetId, err = resolveOutsideCabinet(trn, site.Id)
					eq.StartUnitPos, eq.SizeUnit = 0, 0
				}
				if err != nil {
					log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка резолва шкафа:", err)
					trn.Rollback()
					continue
				}
			}
		}

		// --- Имя записи: hostname -> описание -> тех. номер ----------------
		if strings.TrimSpace(eq.Name) == "" {
			eq.Name = firstNonEmptyStr(eq.Annotation, eq.TechnicalNumberIT, eq.SerialNum)
			if len(eq.Name) > 200 {
				eq.Name = eq.Name[:200]
			}
		}

		// --- Поиск существующей записи: тех. номер IT, затем s/n ------------
		found := false
		var cur models.SwitchSprRecords
		if eq.TechnicalNumberIT != "" {
			found = trn.QueryTable(new(models.SwitchSprRecords)).Filter("TechnicalNumberIT", eq.TechnicalNumberIT).One(&cur) == nil
		}
		if !found && eq.SerialNum != "" {
			found = trn.QueryTable(new(models.SwitchSprRecords)).Filter("SerialNum", eq.SerialNum).One(&cur) == nil
		}

		now := time.Now().Local()
		if !found {
			eq.Id_User = models.UserSourceManual
			eq.Time_Mfy = now
			eq.SourceUpdateType = models.SourceUpdateManual
			if _, err := trn.Insert(eq); err != nil {
				log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка вставки:", err)
				trn.Rollback()
				continue
			}
		} else {
			cur.TypeId = eq.TypeId
			cur.ItiliumStatus = eq.ItiliumStatus
			cur.TimeActiveState = eq.TimeActiveState
			cur.DatacenterSiteId = eq.DatacenterSiteId
			cur.RackCabinetId = eq.RackCabinetId
			cur.StartUnitPos = eq.StartUnitPos
			cur.SizeUnit = eq.SizeUnit
			cur.InventoryNum = eq.InventoryNum
			cur.InventoryNumBuh = eq.InventoryNumBuh
			cur.Id_User = models.UserSourceManual
			cur.Time_Mfy = now
			cols := []string{"TypeId", "ItiliumStatus", "TimeActiveState", "DatacenterSiteId", "RackCabinetId",
				"StartUnitPos", "SizeUnit", "InventoryNum", "InventoryNumBuh", "Id_User", "Time_Mfy"}
			if cur.TechnicalNumberIT == "" && eq.TechnicalNumberIT != "" {
				cur.TechnicalNumberIT = eq.TechnicalNumberIT
				cols = append(cols, "TechnicalNumberIT")
			}
			if cur.SerialNum == "" && eq.SerialNum != "" {
				cur.SerialNum = eq.SerialNum
				cols = append(cols, "SerialNum")
			}
			if _, err := trn.Update(&cur, cols...); err != nil {
				log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка обновления:", err)
				trn.Rollback()
				continue
			}
		}

		for _, w := range r.Warnings {
			log.Println("itilliumimport:", name, w)
		}
		if err := trn.Commit(); err != nil {
			log.Println("itilliumimport: ImportItiliumNetworkEquipment(", name, ") - ошибка коммита транзакции:", err)
			continue
		}
		// "Неактивные" - выведен (2) или списан (3).
		job.addResult(!found, eq.ItiliumStatus == 2 || eq.ItiliumStatus == 3)
	}
}

// resolveNetworkEquipmentType ищет тип по имени (без учёта регистра) в
// NetworkEquipmentType, при отсутствии заводит. Пустое имя -> 0.
func resolveNetworkEquipmentType(trn orm.TxOrmer, cache map[string]int64, typeName string) (int64, error) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return 0, nil
	}
	key := strings.ToLower(typeName)
	if id, ok := cache[key]; ok {
		return id, nil
	}
	var t models.NetworkEquipmentType
	if err := trn.QueryTable(new(models.NetworkEquipmentType)).Filter("Name__iexact", typeName).One(&t); err == nil {
		cache[key] = t.Id
		return t.Id, nil
	}
	t = models.NetworkEquipmentType{Name: typeName, Id_User: models.UserSourceManual, Time_Mfy: time.Now().Local()}
	if _, err := trn.Insert(&t); err != nil {
		return 0, err
	}
	// В кэш - только после коммита строки нельзя гарантировать, поэтому кэшируем
	// лишь найденные записи; созданный тип попадёт в кэш при следующем чтении.
	return t.Id, nil
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
