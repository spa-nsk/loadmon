package itilliumimport

import (
	"loadmon/locality"
	"loadmon/models"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// mappingConfigFileNameTape - имя yaml-файла соответствий для импорта ленточных
// библиотек в каталоге, заданном ключом "yaml" в app.conf (см. mappingConfigDir()).
const mappingConfigFileNameTape = "tape_library.yaml"

func mappingConfigPathTape() string {
	return mappingConfigDir() + "/" + mappingConfigFileNameTape
}

// StartImportItiliumTapeLibrary регистрирует задание импорта (см. ImportProgress)
// и запускает runImportItiliumTapeLibrary в отдельной горутине.
func StartImportItiliumTapeLibrary(name string) *ImportProgress {
	job := newImportJob(name)
	go runImportItiliumTapeLibrary(name, job)
	return job
}

// ImportItiliumTapeLibrary - синхронная версия импорта (без прогресса).
func ImportItiliumTapeLibrary(name string) {
	runImportItiliumTapeLibrary(name, &ImportProgress{})
}

// runImportItiliumTapeLibrary читает xlsx-файл name (каталог "./import"),
// конвертирует строки через ConvertFileTape (tape_library.yaml) и создаёт/
// обновляет models.DatacenterSite -> models.RackCabinet -> models.TapeLibrary.
//
//   - Запись ищется по серийному номеру (при пустом - по тех. номеру IT, он же
//     тогда пишется в SerialNum, как у rack-серверов).
//   - Адрес: площадка и шкаф берутся из "Местоположение" (resolveSite +
//     cabinetFromPlacement), начальный юнит - из "Unit N" того же поля, размер в
//     юнитах - из колонки units. Нет шкафа - служебный шкаф "вне шкафа" площадки.
//   - "Имя ПК" сопоставляется со справочником серверов (FindServerByPcName).
//   - Из существующей записи обновляются только поля, приносимые импортом
//     (статус, размещение, сервер, номера, ip); остальное - приоритет ручных правок.
func runImportItiliumTapeLibrary(name string, job *ImportProgress) {
	var startErr string
	defer func() { job.finish(startErr) }()

	cfg, err := LoadMapping(mappingConfigPathTape())
	if err != nil {
		log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка загрузки конфигурации соответствий:", err)
		startErr = err.Error()
		return
	}

	results, err := ConvertFileTape(cfg, "./import/"+name)
	if err != nil {
		log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка чтения/разбора файла:", err)
		startErr = err.Error()
		return
	}
	job.setFileRows(len(results))

	o := orm.NewOrm()
	rs := locality.NewResolver()

	for _, r := range results {
		job.tickProcessed()

		if r.Tl.SerialNum == "" {
			r.Tl.SerialNum = r.Tl.TechnicalNumberIT
		}
		if r.Tl.SerialNum == "" {
			log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - пропущена строка без S/N и тех. номера IT")
			continue
		}

		trn, err := o.Begin()
		if err != nil {
			log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка начала транзакции:", err)
			continue
		}

		// --- Площадка, шкаф, юнит из "Местоположение" ----------------------
		site, errSite := resolveSite(trn, rs, r.Extra["location_raw"], models.UserSourceManual)
		if errSite != nil {
			log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка резолва площадки:", errSite)
			trn.Rollback()
			continue
		}
		rackId, startU, endU := int64(0), 0, 0
		if site.Id > 0 && !site.Loc.Service {
			cabinetCode, cabinetAlt, locPos := cabinetFromPlacement("", site.Loc)
			if cabinetCode != "" {
				rackId, err = resolveRackCabinet(trn, cabinetCode, cabinetAlt, site.Id,
					"Шкаф по данным импорта ленточных библиотек из Итиллиум, площадка: "+site.Name)
				if err == nil && locPos != nil {
					startU, endU = locPos.Start, locPos.End
				}
			} else {
				rackId, err = resolveOutsideCabinet(trn, site.Id)
			}
			if err != nil {
				log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка резолва шкафа:", err)
				trn.Rollback()
				continue
			}
		}
		r.Tl.RackCabinetId = rackId
		r.Tl.StartUnitPos = startU
		if r.Tl.SizeUnit == 0 && endU >= startU && startU > 0 {
			r.Tl.SizeUnit = endU - startU + 1 // в выгрузке нет units - берём диапазон Unit N-M
		}
		if startU == 0 {
			r.Tl.SizeUnit = 0 // вне шкафа позиции и размера нет
		}

		// --- Сервер подключения по "Имя ПК" --------------------------------
		serverId, err := FindServerByPcName(trn, r.Tl.PcName)
		if err != nil {
			log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка поиска сервера по имени", r.Tl.PcName, ":", err)
			trn.Rollback()
			continue
		}
		r.Tl.ServerId = serverId

		// --- TapeLibrary ----------------------------------------------------
		tl := models.TapeLibrary{SerialNum: r.Tl.SerialNum}
		found := trn.Read(&tl, "SerialNum") == nil
		now := time.Now().Local()
		if !found {
			r.Tl.TimeCreate = now
			r.Tl.CreatorUser = models.UserSourceManual
			r.Tl.Id_User = models.UserSourceManual
			r.Tl.Time_Mfy = now
			r.Tl.SourceUpdateType = models.SourceUpdateManual
			if _, err := trn.Insert(&r.Tl); err != nil {
				log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка вставки:", err)
				trn.Rollback()
				continue
			}
		} else {
			tl.ItiliumStatus = r.Tl.ItiliumStatus
			tl.TimeActiveState = r.Tl.TimeActiveState
			tl.RackCabinetId = r.Tl.RackCabinetId
			tl.StartUnitPos = r.Tl.StartUnitPos
			tl.SizeUnit = r.Tl.SizeUnit
			tl.PcName = r.Tl.PcName
			tl.IPAddress = r.Tl.IPAddress
			tl.InventoryNum = r.Tl.InventoryNum
			tl.InventoryNumBuh = r.Tl.InventoryNumBuh
			tl.TechnicalNumberIT = r.Tl.TechnicalNumberIT
			tl.Id_User = models.UserSourceManual
			tl.Time_Mfy = now
			cols := []string{"ItiliumStatus", "TimeActiveState", "RackCabinetId", "StartUnitPos", "SizeUnit",
				"PcName", "IPAddress", "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "Id_User", "Time_Mfy"}
			if serverId > 0 { // найденная связь не затирается пустым результатом поиска
				tl.ServerId = serverId
				cols = append(cols, "ServerId")
			}
			if _, err := trn.Update(&tl, cols...); err != nil {
				log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка обновления:", err)
				trn.Rollback()
				continue
			}
		}

		for _, w := range r.Warnings {
			log.Println("itilliumimport:", name, w)
		}
		if err := trn.Commit(); err != nil {
			log.Println("itilliumimport: ImportItiliumTapeLibrary(", name, ") - ошибка коммита транзакции:", err)
			continue
		}
		// "Неактивные" - выведен (2) или списан (3).
		job.addResult(!found, r.Tl.ItiliumStatus == 2 || r.Tl.ItiliumStatus == 3)
	}
}

// FindServerByPcName ищет сервер в справочнике ServerSprRecords по "Имя ПК":
// сначала точное совпадение с Name или NameFQDN (без учёта регистра), затем по
// короткому имени (до первой точки). Записи Stat=666 пропускаются, при
// нескольких кандидатах предпочитается активный (Active=1). Не найден -> 0.
func FindServerByPcName(trn interface {
	QueryTable(ptrStructOrTableName interface{}) orm.QuerySeter
}, pcName string) (int64, error) {
	pcName = strings.TrimSpace(pcName)
	if pcName == "" {
		return 0, nil
	}
	candidates := []string{pcName}
	if i := strings.Index(pcName, "."); i > 0 {
		candidates = append(candidates, pcName[:i])
	}
	for _, field := range []string{"Name__iexact", "NameFQDN__iexact"} {
		for _, c := range candidates {
			var list []models.ServerSprRecords
			if _, err := trn.QueryTable(new(models.ServerSprRecords)).Filter(field, c).Limit(10).All(&list); err != nil {
				return 0, err
			}
			best := int64(0)
			for i := range list {
				if models.IsServerSprDestroyed(&list[i]) {
					continue
				}
				if list[i].Active == 1 {
					return list[i].Id, nil
				}
				if best == 0 {
					best = list[i].Id
				}
			}
			if best > 0 {
				return best, nil
			}
		}
	}
	return 0, nil
}
