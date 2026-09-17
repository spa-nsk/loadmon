package cmdb

import (
	"context"
	"fmt"
	"loadmon/emcstorage"
	"loadmon/lenovostorage"
	"log"
	models "loadmon/models"
	"loadmon/storages"
	datamap "loadmon/utilities"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/lovi-cloud/go-dorado-sdk/dorado"
)

// SyncStorageHardwareForSystem синхронизирует полки, диски и пулы хранения
// одной СХД после успешного опроса томов (вызывается из UpdateVolumeStorage)
// - обязательно по КАЖДОМУ вендору, представленному в справочнике СХД
// (2026-09-03):
//   - VNX (1) и Unity (2) - полная инвентаризация полок и дисков, плюс пулы
//     (2026-09-16, "для VNX/Unity/Lenovo тоже нужно" - см.
//     storages.GetStoragePoolsCmd/GetUnityPools).
//   - Lenovo (5) - полная инвентаризация полок/дисков через GetGraph
//     (Tray/Drive), плюс пулы через уже существовавший
//     session.GetStoragePools() (2026-09-16, тот же запрос), см.
//     syncLenovoHardware.
//   - Dorado/OceanStor (3, 4) - полная инвентаризация полок и дисков через
//     GetEnclosures()/GetDisks() (2026-09-16, методы добавлены в вендоренный
//     go-dorado-sdk по запросу пользователя loadmon - см.
//     go-dorado-sdk/dorado/enclosure.go и disk.go), плюс пулы через уже
//     существовавший GetStoragePools(), см. syncHuaweiHardware. Сводная
//     сырая ёмкость/UsedPercent из GetSystem (syncHuaweiSystemTotals)
//     используется как основа/резерв на случай, если листинг дисков не
//     поддержан конкретной прошивкой.
//
// Во всех четырёх вендорах пулы хранятся в единой вендор-независимой
// models.StoragePool (см. её комментарий) - поле Description заполняется из
// вендорского API ОБЯЗАТЕЛЬНО там, где оно доступно (Huawei DESCRIPTION,
// Unity description, VNX Description из naviseccli); у Lenovo SANtricity API
// отдельного поля description для дисковых групп/пулов нет, там Description
// остаётся пустой строкой - ограничение API, не ошибка синхронизации.
func SyncStorageHardwareForSystem(o orm.Ormer, shd *models.StorageSystems) (err error) {
	if shd == nil || shd.Id <= 0 || shd.Power != 1 {
		return nil
	}
	// defer/recover (2026-09-03) - см. подробный комментарий у
	// safeCollectStorageMetricForSystem в cmdb/collectstoragemetrics.go:
	// та же категория риска (сторонние SDK, сетевые вызовы), тот же эффект
	// без recover - паника в вызывающей горутине UpdateVolumeStorage убивает
	// весь процесс metrics_grpc, а не только опрос одной СХД.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("cmdb.SyncStorageHardwareForSystem: ПАНИКА при синхронизации hardware СХД Id=%d ShortName=%s ApiType=%d: %v", shd.Id, shd.ShortName, shd.ApiType, r)
			err = fmt.Errorf("паника при синхронизации hardware: %v", r)
		}
	}()
	// ВРЕМЕННАЯ ДИАГНОСТИКА (2026-09-16, "по всем СХД не собираются данные по
	// дискам и полкам, хотя запросы проходят" - без ошибок в логе и без
	// доступа к БД со стороны пользователя единственный способ понять, на
	// каком шаге теряются данные - счётчики в лог сразу после каждого
	// вендорского API-вызова, до upsert; удалить после диагностики).
	log.Printf("[HW-sync] СХД Id=%d ShortName=%s ApiType=%d Power=%d - запуск синхронизации hardware", shd.Id, shd.ShortName, shd.ApiType, shd.Power)
	switch shd.ApiType {
	case 1:
		return syncVNXHardware(o, shd)
	case 2:
		return syncUnityHardware(o, shd)
	case 3, 4:
		return syncHuaweiHardware(o, shd)
	case 5:
		return syncLenovoHardware(o, shd)
	default:
		log.Printf("[HW-sync] СХД Id=%d: неизвестный ApiType=%d, hardware не синхронизируется", shd.Id, shd.ApiType)
		return nil
	}
}

func syncUnityHardware(o orm.Ormer, shd *models.StorageSystems) error {
	encByKey := map[string]int64{}
	touchedEnc := make([]int64, 0)
	touchedDisk := make([]int64, 0)

	// GetUnityHardwareBundle - ОДНА REST-сессия на полки+диски+пулы (2026-09-17,
	// см. её комментарий в storages/unity_hardware.go) - до этой правки здесь
	// было четыре отдельных login/logout-цикла подряд (GetUnityDPE/DAE/Disks/
	// Pools по отдельности), что при и без того нестабильном REST-интерфейсе
	// конкретных Unity-массивов (см. CHANGELOG, жалоба "диски/полки/пулы не
	// появились" - широкие HTTP 422 на СОВЕРШЕННО другие запросы того же
	// массива) увеличивало нагрузку без необходимости.
	bundle, err := storages.GetUnityHardwareBundle(shd.IpAddress)
	if err != nil {
		return err
	}
	dpeList := bundle.DPE
	log.Printf("[HW-sync] Unity Id=%d: GetUnityDPE вернул %d записей", shd.Id, len(dpeList))
	for _, dpe := range dpeList {
		key := "dpe:" + dpe.ID
		id, err := upsertStorageEnclosure(o, shd.Id, models.StorageEnclosure{
			EnclosureType:    models.StorageEnclosureTypeManagement,
			ExternalKey:      key,
			Name:             firstNonEmptyStr(dpe.Name, "DPE"),
			Vendor:           dpe.Manufacturer,
			Model:            dpe.Model,
			SerialNum:        firstNonEmptyStr(dpe.EmcSerialNumber, dpe.VendorSerialNumber),
			SlotNumber:       int(dpe.SlotNumber),
			BusId:            int(dpe.BusID),
			HealthStatus:     dpe.Health.Value,
			Status:           models.StorageHWStatusActive,
			SourceUpdateType: models.StorageSourceUpdateStorageAPI,
			Time_Mfy:         time.Now().Local(),
		})
		if err != nil {
			return err
		}
		encByKey[key] = id
		touchedEnc = append(touchedEnc, id)
	}

	daeList := bundle.DAE
	log.Printf("[HW-sync] Unity Id=%d: GetUnityDAE вернул %d записей", shd.Id, len(daeList))
	for _, dae := range daeList {
		key := "dae:" + dae.ID
		id, err := upsertStorageEnclosure(o, shd.Id, models.StorageEnclosure{
			EnclosureType:    models.StorageEnclosureTypeData,
			ExternalKey:      key,
			Name:             firstNonEmptyStr(dae.Name, "DAE"),
			Vendor:           dae.Manufacturer,
			Model:            dae.Model,
			SerialNum:        firstNonEmptyStr(dae.EmcSerialNumber, dae.VendorSerialNumber),
			SlotNumber:       int(dae.SlotNumber),
			BusId:            int(dae.BusID),
			HealthStatus:     dae.Health.Value,
			Status:           models.StorageHWStatusActive,
			SourceUpdateType: models.StorageSourceUpdateStorageAPI,
			Time_Mfy:         time.Now().Local(),
		})
		if err != nil {
			return err
		}
		encByKey[key] = id
		touchedEnc = append(touchedEnc, id)
	}

	disks := bundle.Disks
	log.Printf("[HW-sync] Unity Id=%d: GetUnityDisks вернул %d записей", shd.Id, len(disks))
	inUse := 0
	var totalCapacityGb float64
	for _, d := range disks {
		key := "disk:" + firstNonEmptyStr(d.WWN, d.EmcSerialNumber, d.ID)
		status := models.StorageHWStatusActive
		if !d.IsInUse {
			status = 2
		} else {
			inUse++
		}
		capGb := float64(d.Size) / (1024 * 1024 * 1024)
		totalCapacityGb += capGb
		// StorageEnclosureId - ИСПРАВЛЕНО 2026-09-17 по замечанию пользователя
		// "по Unity полки с данными не связаны с дисками, диски без привязки к
		// полке" - раньше это поле здесь вообще не заполнялось (не ошибка
		// значения, а полностью пропущенное присваивание), поэтому ВСЕ диски
		// Unity попадали в StorageEnclosureId=0 независимо от реальной полки.
		// В REST API Unity нет отдельного поля-ссылки "какой DAE/DPE" у
		// объекта disk - принадлежность кодируется в самом id диска по тому
		// же соглашению именования, что и у DPE/DAE (см. dpeList/daeList
		// выше в этой функции): "dpe_disk_N" для дисков в DPE, "dae_<bus>_
		// <slot>_disk_N" для дисков в DAE - unityDiskParentEnclosureKey
		// отрезает суффикс "_disk_N" и добавляет тот же префикс "dpe:"/"dae:",
		// которым проиндексирован encByKey чуть выше.
		encId := encByKey[unityDiskParentEnclosureKey(d.ID)]
		id, err := upsertStorageDisk(o, shd.Id, models.StorageDisk{
			StorageEnclosureId: encId,
			ExternalKey:        key,
			SlotNumber:         int(d.SlotNumber),
			BusId:              int(d.BusId),
			Vendor:             d.Manufacturer,
			Model:              d.Model,
			DiskType:           unityDiskTechnologyLabel(d.DiskTechnology),
			SerialNum:          d.EmcSerialNumber,
			CapacityGb:         capGb,
			Status:             status,
			SourceUpdateType:   models.StorageSourceUpdateStorageAPI,
			Time_Mfy:           time.Now().Local(),
		})
		if err != nil {
			return err
		}
		touchedDisk = append(touchedDisk, id)
	}

	touchedPool := make([]int64, 0)
	pools, poolsErr := bundle.Pools, bundle.PoolsErr
	if poolsErr != nil {
		// GetPool (внутри общей сессии bundle) - не фатально для остального
		// hardware-инвентаря (полки и диски уже собраны выше); при ошибке НЕ
		// вызываем deactivateAbsentStoragePools ниже, чтобы транзитная ошибка
		// запроса не деактивировала все реально существующие пулы
		// (2026-09-16, "для VNX/Unity/Lenovo тоже нужно" - обработка пулов).
		log.Printf("[HW-sync] Unity Id=%d: GetUnityPools вернул ошибку (не прерывает синхронизацию hardware): %v", shd.Id, poolsErr)
	} else {
		log.Printf("[HW-sync] Unity Id=%d: GetUnityPools вернул %d записей", shd.Id, len(pools))
		for _, p := range pools {
			totalGb := float64(p.SizeTotal) / (1024 * 1024 * 1024)
			freeGb := float64(p.SizeFree) / (1024 * 1024 * 1024)
			usedGb := float64(p.SizeUsed) / (1024 * 1024 * 1024)
			usedPercent := 0.0
			if p.SizeTotal > 0 {
				usedPercent = float64(p.SizeUsed) / float64(p.SizeTotal) * 100
			}
			id, err := upsertStoragePool(o, shd.Id, models.StoragePool{
				ExternalKey: "unity:pool:" + p.ID,
				Name:        firstNonEmptyStr(p.Name, "Pool "+p.ID),
				// Description ОБЯЗАТЕЛЬНО заполняется полем description Unity
				// REST API (по прямому запросу пользователя loadmon) - пустая
				// строка, если пул не аннотирован администратором СХД, это не
				// ошибка.
				Description:      p.Description,
				HealthStatus:      p.Health.Value,
				TotalCapacityGb:   totalGb,
				FreeCapacityGb:    freeGb,
				UsedCapacityGb:    usedGb,
				UsedPercent:       usedPercent,
				Status:            models.StorageHWStatusActive,
				SourceUpdateType:  models.StorageSourceUpdateStorageAPI,
				Time_Mfy:          time.Now().Local(),
			})
			if err != nil {
				return err
			}
			touchedPool = append(touchedPool, id)
		}
		deactivateAbsentStoragePools(o, shd.Id, touchedPool)
	}

	deactivateAbsentStorageEnclosures(o, shd.Id, touchedEnc)
	deactivateAbsentStorageDisks(o, shd.Id, touchedDisk)
	updateStorageSystemsDiskStats(o, shd, len(disks), inUse, totalCapacityGb)
	log.Printf("[HW-sync] Unity Id=%d: сохранено полок=%d дисков=%d пулов=%d", shd.Id, len(touchedEnc), len(touchedDisk), len(touchedPool))
	return nil
}

func syncVNXHardware(o orm.Ormer, shd *models.StorageSystems) error {
	disks, err := storages.GetDisksCmd(shd.IpAddress)
	if err != nil {
		return err
	}
	log.Printf("[HW-sync] VNX Id=%d: GetDisksCmd вернул %d записей", shd.Id, len(disks))
	encByNum := map[uint32]int64{}
	touchedEnc := make([]int64, 0)
	touchedDisk := make([]int64, 0)
	inUse := 0
	var totalCapacityGb float64

	for _, d := range disks {
		if d == nil {
			continue
		}
		encNum := d.Enclosure
		if encNum > 0 {
			if _, ok := encByNum[encNum]; !ok {
				key := fmt.Sprintf("vnx:enc:%d", encNum)
				id, err := upsertStorageEnclosure(o, shd.Id, models.StorageEnclosure{
					EnclosureType:    models.StorageEnclosureTypeData,
					ExternalKey:      key,
					Name:             fmt.Sprintf("Enclosure %d", encNum),
					Vendor:           d.VendorId,
					Model:            d.ProductId,
					SerialNum:        "",
					SlotNumber:       int(encNum),
					BusId:            int(d.Bus),
					Status:           models.StorageHWStatusActive,
					SourceUpdateType: models.StorageSourceUpdateStorageAPI,
					Time_Mfy:         time.Now().Local(),
				})
				if err != nil {
					return err
				}
				encByNum[encNum] = id
				touchedEnc = append(touchedEnc, id)
			}
		}
		key := fmt.Sprintf("vnx:%d:%d:%d", d.Bus, d.Enclosure, d.Disk)
		encId := encByNum[encNum]
		status := models.StorageHWStatusActive
		if d.RaidGroupId != "" && d.RaidGroupId != "None" {
			inUse++
		} else {
			status = 2
		}
		capGb := float64(d.Capacity) / (1024 * 1024 * 1024)
		totalCapacityGb += capGb
		id, err := upsertStorageDisk(o, shd.Id, models.StorageDisk{
			StorageEnclosureId: encId,
			ExternalKey:        key,
			SlotNumber:           int(d.Disk),
			BusId:                int(d.Bus),
			EnclosureNum:         int(d.Enclosure),
			Vendor:               strings.TrimSpace(d.VendorId),
			Model:                strings.TrimSpace(d.ProductId),
			DiskType:             strings.TrimSpace(d.DriveType),
			SerialNum:            strings.TrimSpace(d.SerialNumber),
			CapacityGb:           capGb,
			RaidGroupId:          d.RaidGroupId,
			Status:               status,
			SourceUpdateType:     models.StorageSourceUpdateStorageAPI,
			Time_Mfy:             time.Now().Local(),
		})
		if err != nil {
			return err
		}
		touchedDisk = append(touchedDisk, id)
	}

	touchedPool := make([]int64, 0)
	poolsList, poolsErr := storages.GetStoragePoolsCmd(shd.IpAddress)
	if poolsErr != nil {
		// naviseccli storagepool -list - не фатально для остального
		// hardware-инвентаря (полки/диски уже собраны выше); при ошибке НЕ
		// деактивируем существующие пулы, см. тот же приём у syncUnityHardware
		// (2026-09-16, "для VNX/Unity/Lenovo тоже нужно" - обработка пулов).
		log.Printf("[HW-sync] VNX Id=%d: GetStoragePoolsCmd вернул ошибку (не прерывает синхронизацию hardware): %v", shd.Id, poolsErr)
	} else {
		log.Printf("[HW-sync] VNX Id=%d: GetStoragePoolsCmd вернул %d записей", shd.Id, len(poolsList))
		for _, p := range poolsList {
			if p == nil {
				continue
			}
			usedPercent := 0.0
			if p.UserCapacityGBs > 0 {
				usedPercent = p.ConsumedCapacityGBs / p.UserCapacityGBs * 100
			}
			id, err := upsertStoragePool(o, shd.Id, models.StoragePool{
				ExternalKey: fmt.Sprintf("vnx:pool:%d", p.Id),
				Name:        firstNonEmptyStr(p.Name, fmt.Sprintf("Pool %d", p.Id)),
				// Description ОБЯЗАТЕЛЬНО заполняется полем Description из
				// naviseccli storagepool -list -description (по прямому
				// запросу пользователя loadmon) - у VNX часто приходит пустой
				// строкой, если пул не аннотирован администратором, это не
				// ошибка.
				Description:      p.Description,
				TotalCapacityGb:  p.UserCapacityGBs,
				FreeCapacityGb:   p.AvailableCapacityGBs,
				UsedCapacityGb:   p.ConsumedCapacityGBs,
				UsedPercent:      usedPercent,
				Status:           models.StorageHWStatusActive,
				SourceUpdateType: models.StorageSourceUpdateStorageAPI,
				Time_Mfy:         time.Now().Local(),
			})
			if err != nil {
				return err
			}
			touchedPool = append(touchedPool, id)
		}
		deactivateAbsentStoragePools(o, shd.Id, touchedPool)
	}

	deactivateAbsentStorageEnclosures(o, shd.Id, touchedEnc)
	deactivateAbsentStorageDisks(o, shd.Id, touchedDisk)
	updateStorageSystemsDiskStats(o, shd, len(disks), inUse, totalCapacityGb)
	log.Printf("[HW-sync] VNX Id=%d: сохранено полок=%d дисков=%d пулов=%d", shd.Id, len(touchedEnc), len(touchedDisk), len(touchedPool))
	return nil
}

func upsertStorageEnclosure(o orm.Ormer, storageID int64, row models.StorageEnclosure) (int64, error) {
	row.StorageSystemsId = storageID
	existing := models.StorageEnclosure{}
	err := o.QueryTable(new(models.StorageEnclosure)).
		Filter("StorageSystemsId", storageID).
		Filter("ExternalKey", row.ExternalKey).
		One(&existing)
	if err == nil {
		row.Id = existing.Id
		if _, err := o.Update(&row); err != nil {
			return 0, err
		}
		return row.Id, nil
	}
	id, err := o.Insert(&row)
	return id, err
}

func upsertStorageDisk(o orm.Ormer, storageID int64, row models.StorageDisk) (int64, error) {
	row.StorageSystemsId = storageID
	existing := models.StorageDisk{}
	err := o.QueryTable(new(models.StorageDisk)).
		Filter("StorageSystemsId", storageID).
		Filter("ExternalKey", row.ExternalKey).
		One(&existing)
	if err == nil {
		row.Id = existing.Id
		if _, err := o.Update(&row); err != nil {
			return 0, err
		}
		return row.Id, nil
	}
	id, err := o.Insert(&row)
	return id, err
}

// upsertStoragePool - тот же upsert-по-ExternalKey паттерн, что и у
// upsertStorageEnclosure/upsertStorageDisk (2026-09-16, добавлено по
// запросу пользователя loadmon "добавь обработку пулов и обязательно с
// полем description" - см. syncHuaweiHardware и models.StoragePool).
func upsertStoragePool(o orm.Ormer, storageID int64, row models.StoragePool) (int64, error) {
	row.StorageSystemsId = storageID
	existing := models.StoragePool{}
	err := o.QueryTable(new(models.StoragePool)).
		Filter("StorageSystemsId", storageID).
		Filter("ExternalKey", row.ExternalKey).
		One(&existing)
	if err == nil {
		row.Id = existing.Id
		if _, err := o.Update(&row); err != nil {
			return 0, err
		}
		return row.Id, nil
	}
	id, err := o.Insert(&row)
	return id, err
}

func deactivateAbsentStoragePools(o orm.Ormer, storageID int64, touched []int64) {
	t := map[int64]bool{}
	for _, id := range touched {
		t[id] = true
	}
	var rows []*models.StoragePool
	_, _ = o.QueryTable(new(models.StoragePool)).
		Filter("StorageSystemsId", storageID).
		Filter("SourceUpdateType", models.StorageSourceUpdateStorageAPI).
		Filter("Status", models.StorageHWStatusActive).
		All(&rows)
	for _, r := range rows {
		if r == nil || t[r.Id] {
			continue
		}
		r.Status = models.StorageHWStatusRemoved
		_, _ = o.Update(r, "Status")
	}
}

func deactivateAbsentStorageEnclosures(o orm.Ormer, storageID int64, touched []int64) {
	t := map[int64]bool{}
	for _, id := range touched {
		t[id] = true
	}
	var rows []*models.StorageEnclosure
	_, _ = o.QueryTable(new(models.StorageEnclosure)).
		Filter("StorageSystemsId", storageID).
		Filter("SourceUpdateType", models.StorageSourceUpdateStorageAPI).
		Filter("Status", models.StorageHWStatusActive).
		All(&rows)
	for _, r := range rows {
		if r == nil || t[r.Id] {
			continue
		}
		r.Status = models.StorageHWStatusRemoved
		_, _ = o.Update(r, "Status")
	}
}

func deactivateAbsentStorageDisks(o orm.Ormer, storageID int64, touched []int64) {
	t := map[int64]bool{}
	for _, id := range touched {
		t[id] = true
	}
	var rows []*models.StorageDisk
	_, _ = o.QueryTable(new(models.StorageDisk)).
		Filter("StorageSystemsId", storageID).
		Filter("SourceUpdateType", models.StorageSourceUpdateStorageAPI).
		Filter("Status__gt", 0).
		All(&rows)
	for _, r := range rows {
		if r == nil || t[r.Id] {
			continue
		}
		r.Status = models.StorageHWStatusRemoved
		_, _ = o.Update(r, "Status")
	}
}

// updateStorageSystemsDiskStats записывает в запись справочника СХД
// (models.StorageSystems) фактические данные, собранные при синхронизации
// физических дисков: количество дисков (Size), число задействованных
// (Used) и, по прямому запросу пользователя (2026-09-03), суммарную СЫРУЮ
// ёмкость всех собранных дисков (TotalSizeRAW, Гб) - раньше это поле
// заполнялось только вручную через /sprsrv/storagesystemsedit, теперь при
// успешной синхронизации дисков (VNX/Unity/Lenovo) автоматически
// проставляется суммой CapacityGb по факту опроса СХД - "установка данных
// в справочник по итоговому сырому месту".
func updateStorageSystemsDiskStats(o orm.Ormer, shd *models.StorageSystems, total, inUse int, totalCapacityGb float64) {
	if total <= 0 {
		return
	}
	shd.Size = total
	shd.Used = inUse
	shd.TotalSizeRAW = totalCapacityGb
	_, _ = o.Update(shd, "Size", "Used", "TotalSizeRAW")
}

// syncHuaweiSystemTotals обновляет сводную сырую ёмкость и % использования
// СХД Huawei Dorado/OceanStor из GetSystem(). Изначально (до 2026-09-16) это
// был единственный доступный для этого вендора источник данных без per-
// диск/per-полка листинга; теперь используется как первый шаг внутри
// syncHuaweiHardware - "базовое" значение TotalSizeRAW/UsedPercent, которое
// при успешном опросе GetDisks()/GetEnclosures() перезаписывается точными
// суммами по факту собранных физических дисков (как у VNX/Unity/Lenovo), а
// при ошибке листинга дисков (например, старая прошивка без этих ручек)
// остаётся единственным источником - см. updateStorageSystemsDiskStats.
// TOTALCAPACITY у Huawei API приходит в Мб (см. GetDoradoLun/GetHuaweyVolume
// - там же встречается тот же паттерн перевода capacity/sectorsize в Гб),
// поэтому переводится в Гб делением на 1024.
func syncHuaweiSystemTotals(o orm.Ormer, shd *models.StorageSystems) error {
	key := "spaview"
	keypass := datamap.GetKeyData(key)
	remoteIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}
	localIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}

	client, err := dorado.NewClient(localIps, remoteIps, key, keypass, "", nil)
	if err != nil {
		return err
	}

	sys, err := client.LocalDevice.GetSystem(context.Background())
	if err != nil {
		return err
	}
	if sys == nil {
		return fmt.Errorf("СХД %s не вернула данные GetSystem", shd.ShortName)
	}

	totalMb, tErr := strconv.ParseFloat(strings.TrimSpace(sys.TOTALCAPACITY), 64)
	usedMb, uErr := strconv.ParseFloat(strings.TrimSpace(sys.USEDCAPACITY), 64)
	if tErr != nil {
		return fmt.Errorf("не удалось разобрать TOTALCAPACITY=%q: %w", sys.TOTALCAPACITY, tErr)
	}

	shd.TotalSizeRAW = totalMb / 1024
	if uErr == nil && totalMb > 0 {
		shd.UsedPercent = usedMb / totalMb * 100
		_, err = o.Update(shd, "TotalSizeRAW", "UsedPercent")
	} else {
		_, err = o.Update(shd, "TotalSizeRAW")
	}
	return err
}

// syncHuaweiHardware синхронизирует полки, диски и пулы хранения СХД Huawei
// Dorado/OceanStor через GetEnclosures()/GetDisks() (2026-09-16, методы
// добавлены в вендоренный go-dorado-sdk по запросу пользователя loadmon
// "расширить SDK для сбора дисков/полок" - см.
// go-dorado-sdk/dorado/enclosure.go и disk.go) и уже существовавший
// GetStoragePools() (см. models.StoragePool - добавлена в тот же день по
// отдельному запросу "добавь обработку пулов и обязательно с полем
// description"). Тот же mark-and-sweep паттерн (upsert по ExternalKey +
// деактивация нетронутых), что и
// syncVNXHardware/syncUnityHardware/syncLenovoHardware.
//
// ⚠️ Имена JSON-полей у Disk/Enclosure взяты из общей документации Huawei
// DeviceManager REST API, а не проверены на реальном устройстве этого
// пользователя - при первом реальном запуске возможны 1-2 несовпадения
// (например, если конкретная прошивка называет поле иначе или возвращает
// пустую строку там, где ожидалось число) - тогда диски/полки будут
// сохраняться с нулевыми/пустыми отдельными полями, но само наличие записей
// в справочнике (см. [HW-sync] лог ниже) уже покажет, что запрос дошёл и
// список получен - поля можно поправить по факту без изменения структуры
// синхронизации.
//
// Сначала обновляются базовые TotalSizeRAW/UsedPercent через
// syncHuaweiSystemTotals (см. её комментарий) - если GetEnclosures/GetDisks
// не поддержаны прошивкой (ErrEnclosureNotFound/ErrDiskNotFound - пустой
// список, не ошибка транспорта), эти базовые значения останутся
// единственным источником и функция завершится без ошибки.
func syncHuaweiHardware(o orm.Ormer, shd *models.StorageSystems) error {
	if err := syncHuaweiSystemTotals(o, shd); err != nil {
		log.Printf("[HW-sync] Huawei Id=%d: syncHuaweiSystemTotals вернул ошибку (не прерывает синхронизацию hardware): %v", shd.Id, err)
	}

	key := "spaview"
	keypass := datamap.GetKeyData(key)
	remoteIps := []string{fmt.Sprintf("https://%s:8088", shd.IpAddress)}
	localIps := remoteIps

	client, err := dorado.NewClient(localIps, remoteIps, key, keypass, "", nil)
	if err != nil {
		return err
	}

	enclosures, err := client.LocalDevice.GetEnclosures(context.Background(), nil)
	if err != nil && err != dorado.ErrEnclosureNotFound {
		return err
	}
	log.Printf("[HW-sync] Huawei Id=%d: GetEnclosures вернул %d записей", shd.Id, len(enclosures))

	encByID := map[string]int64{}
	touchedEnc := make([]int64, 0, len(enclosures))
	for _, enc := range enclosures {
		// ИСПРАВЛЕНО 2026-09-17: enc.TYPE у реального оборудования оказался
		// числовым кодом (см. комментарий у Enclosure.TYPE в
		// go-dorado-sdk/dorado/enclosure.go), а не строкой "CTRL"/… как
		// предполагалось изначально - классификация управления/данных
		// делается по факту NAME/LOCATION (единственный текстовый признак,
		// доступный без знания конкретных числовых кодов TYPE на этой
		// прошивке).
		encType := models.StorageEnclosureTypeData
		if strings.Contains(strings.ToUpper(enc.NAME), "CTE") || strings.Contains(strings.ToUpper(enc.LOCATION), "CTE") {
			encType = models.StorageEnclosureTypeManagement
		}
		slot, _ := strconv.Atoi(strings.TrimSpace(enc.SLOTNUM))
		bus, _ := strconv.Atoi(strings.TrimSpace(enc.BUSID))
		id, err := upsertStorageEnclosure(o, shd.Id, models.StorageEnclosure{
			EnclosureType:    encType,
			ExternalKey:      "huawei:enc:" + enc.ID,
			Name:             firstNonEmptyStr(enc.NAME, enc.LOCATION, fmt.Sprintf("Enclosure %s", enc.ID)),
			Vendor:           firstNonEmptyStr(enc.MANUFACTURER, "Huawei"),
			Model:            enc.MODEL,
			SerialNum:        enc.SERIALNUMBER,
			SlotNumber:       slot,
			BusId:            bus,
			HealthStatus:     huaweiHealthStatusCode(enc.HEALTHSTATUS),
			Status:           models.StorageHWStatusActive,
			SourceUpdateType: models.StorageSourceUpdateStorageAPI,
			Time_Mfy:         time.Now().Local(),
		})
		if err != nil {
			return err
		}
		encByID[enc.ID] = id
		touchedEnc = append(touchedEnc, id)
	}

	// ИСПРАВЛЕНО 2026-09-17 по диагностике из (83): GetDisks(ctx, nil) БЕЗ
	// фильтра на реальном оборудовании пользователя возвращал диски
	// практически без атрибутов (DISKTYPE был ещё осмысленным кодом "14",
	// но CAPACITY/SLOTNUM/ENCLOSUREID были пустыми строками, а PARENTID -
	// "0" у ВСЕХ 36 дисков без исключения) - это известная особенность
	// DeviceManager REST API Huawei: полные атрибуты диска отдаются только
	// при запросе, ассоциированном с конкретной родительской полкой
	// (ASSOCIATEOBJTYPE=21 - TypeEnclosure, ASSOCIATEOBJID=<Enclosure.ID>,
	// см. go-dorado-sdk/dorado/query.go:SearchQuery.AssociateObjType/
	// AssociateObjID и константу TypeEnclosure в constants.go) - без этой
	// ассоциации API, судя по всему, отдаёт "облегчённый" список. Поэтому
	// теперь диски запрашиваются ОТДЕЛЬНО по каждой полке, а не одним
	// общим вызовом - как и GetHostAssociatedLUNs у томов (см.
	// cmdb/UpdateHuaweyVolume.go), тот же принцип API Huawei "список
	// association-объектов по родителю".
	var disks []dorado.Disk
	// diskParentEnc - соответствие Disk.ID -> Enclosure.ID полки, у которой
	// этот диск запрошен - используется НИЖЕ как основной источник
	// привязки к полке вместо d.ENCLOSUREID/d.PARENTID (которые по
	// диагностике из (83) на этом оборудовании остаются пустыми даже при
	// association-запросе - подстраховка, а не отмена самого фильтра выше:
	// сам факт association-запроса всё равно нужен, чтобы получить остальные
	// атрибуты диска - CAPACITY, DISKTYPE и т.п.).
	diskParentEnc := map[string]string{}
	for _, enc := range enclosures {
		encDisks, dErr := client.LocalDevice.GetDisks(context.Background(), &dorado.SearchQuery{
			AssociateObjType: strconv.Itoa(dorado.TypeEnclosure),
			AssociateObjID:   enc.ID,
		})
		if dErr != nil {
			if dErr != dorado.ErrDiskNotFound {
				log.Printf("[HW-sync] Huawei Id=%d: GetDisks по полке ID=%s вернул ошибку (не прерывает синхронизацию остальных полок): %v", shd.Id, enc.ID, dErr)
			}
			continue
		}
		for _, d := range encDisks {
			diskParentEnc[d.ID] = enc.ID
		}
		disks = append(disks, encDisks...)
	}
	if len(enclosures) == 0 {
		// Нет ни одной полки (маловероятно, но не исключено на другой
		// прошивке) - запасной вариант, старое поведение без фильтра, чтобы
		// хотя бы попытаться получить список дисков без привязки к полке.
		fallbackDisks, dErr := client.LocalDevice.GetDisks(context.Background(), nil)
		if dErr != nil && dErr != dorado.ErrDiskNotFound {
			return dErr
		}
		disks = fallbackDisks
	}
	log.Printf("[HW-sync] Huawei Id=%d: GetDisks (по полкам) вернул %d записей", shd.Id, len(disks))

	touchedDisk := make([]int64, 0, len(disks))
	inUse := 0
	var totalCapacityGb float64
	for _, d := range disks {
		// parentID - ИСПРАВЛЕНО 2026-09-17: до перехода на association-запрос
		// по полкам (см. комментарий выше у цикла formирования disks)
		// d.ENCLOSUREID/d.PARENTID были пустыми у ВСЕХ дисков на реальном
		// оборудовании (диагностика подтвердила: ENCLOSUREID="" PARENTID="0"
		// у 36/36 дисков) - теперь основной источник привязки это
		// diskParentEnc (Huawei Enclosure.ID полки, по которой диск был
		// запрошен), d.ENCLOSUREID/d.PARENTID используются только как
		// запасной вариант, если diskParentEnc почему-то не содержит этот
		// диск (например, попал в fallback-ветку без полок).
		parentID := diskParentEnc[d.ID]
		if parentID == "" {
			parentID = firstNonEmptyStr(d.ENCLOSUREID, d.PARENTID)
		}
		slot, _ := strconv.Atoi(strings.TrimSpace(d.SLOTNUM))
		status := models.StorageHWStatusActive
		if strings.TrimSpace(d.LOGICTYPE) == "1" {
			// hot spare - считается "свободным", тот же код статуса, что и у
			// VNX/Unity/Lenovo для незадействованных дисков.
			status = 2
		} else {
			inUse++
		}
		capGb := d.DiskCapacityGb()
		totalCapacityGb += capGb
		id, err := upsertStorageDisk(o, shd.Id, models.StorageDisk{
			StorageEnclosureId: encByID[parentID],
			ExternalKey:        "huawei:disk:" + firstNonEmptyStr(d.SERIALNUMBER, d.ID),
			SlotNumber:         slot,
			Vendor:             firstNonEmptyStr(d.MANUFACTURER, "Huawei"),
			Model:              d.MODEL,
			DiskType:           d.DISKTYPE,
			SerialNum:          d.SERIALNUMBER,
			CapacityGb:         capGb,
			Status:             status,
			SourceUpdateType:   models.StorageSourceUpdateStorageAPI,
			Time_Mfy:           time.Now().Local(),
		})
		if err != nil {
			return err
		}
		touchedDisk = append(touchedDisk, id)
	}

	pools, err := client.LocalDevice.GetStoragePools(context.Background(), nil)
	if err != nil && err != dorado.ErrStoragePoolNotFound {
		return err
	}
	log.Printf("[HW-sync] Huawei Id=%d: GetStoragePools вернул %d записей", shd.Id, len(pools))

	touchedPool := make([]int64, 0, len(pools))
	for _, p := range pools {
		totalGb := huaweiCapacityMbToGb(p.USERTOTALCAPACITY)
		freeGb := huaweiCapacityMbToGb(p.USERFREECAPACITY)
		usedGb := totalGb - freeGb
		if usedGb < 0 {
			usedGb = 0
		}
		usedPercent, pErr := strconv.ParseFloat(strings.TrimSpace(p.USERCONSUMEDCAPACITYPERCENTAGE), 64)
		if pErr != nil {
			usedPercent = 0
		}
		id, err := upsertStoragePool(o, shd.Id, models.StoragePool{
			ExternalKey: fmt.Sprintf("huawei:pool:%d", p.ID),
			Name:        firstNonEmptyStr(p.NAME, fmt.Sprintf("Pool %d", p.ID)),
			// Description ОБЯЗАТЕЛЬНО заполняется полем DESCRIPTION вендорского
			// API (по прямому запросу пользователя loadmon) - у Huawei может
			// приходить пустой строкой, если пул не аннотирован администратором
			// СХД, это ожидаемо и не является ошибкой синхронизации.
			Description:      p.DESCRIPTION,
			HealthStatus:      huaweiHealthStatusCode(p.HEALTHSTATUS),
			TotalCapacityGb:   totalGb,
			FreeCapacityGb:    freeGb,
			UsedCapacityGb:    usedGb,
			UsedPercent:       usedPercent,
			Status:            models.StorageHWStatusActive,
			SourceUpdateType:  models.StorageSourceUpdateStorageAPI,
			Time_Mfy:          time.Now().Local(),
		})
		if err != nil {
			return err
		}
		touchedPool = append(touchedPool, id)
	}
	deactivateAbsentStoragePools(o, shd.Id, touchedPool)

	deactivateAbsentStorageEnclosures(o, shd.Id, touchedEnc)
	deactivateAbsentStorageDisks(o, shd.Id, touchedDisk)
	updateStorageSystemsDiskStats(o, shd, len(disks), inUse, totalCapacityGb)
	log.Printf("[HW-sync] Huawei Id=%d: сохранено полок=%d дисков=%d пулов=%d", shd.Id, len(touchedEnc), len(touchedDisk), len(touchedPool))
	return nil
}

// huaweiCapacityMbToGb переводит объёмное поле Huawei StoragePools (строка,
// значение в Мб - тот же формат, что и TOTALCAPACITY/USEDCAPACITY у
// GetSystem, см. syncHuaweiSystemTotals) в Гб; при ошибке разбора
// возвращает 0, чтобы не прерывать синхронизацию пулов.
func huaweiCapacityMbToGb(raw string) float64 {
	mb, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0
	}
	return mb / 1024
}

// huaweiHealthStatusCode разбирает строковое поле HEALTHSTATUS Huawei
// DeviceManager REST API (см. models.StatusHealth=1 - тот же код "здоров",
// что и у остальных объектов этого SDK) в целое - при нераспознанном/пустом
// значении возвращает 0 (соответствует "неизвестно" по остальным вендорам).
func huaweiHealthStatusCode(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return v
}

// syncLenovoHardware синхронизирует полки (Graph.Tray) и диски
// (Graph.Drive) СХД Lenovo одним вызовом session.GetGraph() -
// единственным доступным в пакете loadmon/lenovostorage источником
// per-диск/per-полка данных (нет отдельных GetDrives()/GetTrays()). Тот же
// mark-and-sweep паттерн (upsert по ExternalKey + деактивация
// нетронутых), что и syncVNXHardware/syncUnityHardware.
func syncLenovoHardware(o orm.Ormer, shd *models.StorageSystems) error {
	key := "monitor"
	session, err := lenovostorage.NewSession(shd.IpAddress, true, key, datamap.GetKeyData(key))
	if err != nil {
		return err
	}

	graph, err := session.GetGraph()
	if err != nil {
		return err
	}
	log.Printf("[HW-sync] Lenovo Id=%d: GetGraph вернул Tray=%d Drive=%d", shd.Id, len(graph.Tray), len(graph.Drive))

	encByRef := map[string]int64{}
	touchedEnc := make([]int64, 0, len(graph.Tray))
	for _, tray := range graph.Tray {
		key := "lenovo:tray:" + tray.TrayRef
		id, err := upsertStorageEnclosure(o, shd.Id, models.StorageEnclosure{
			EnclosureType:    models.StorageEnclosureTypeData,
			ExternalKey:      key,
			Name:             firstNonEmptyStr(tray.PhysicalLocation.Label, fmt.Sprintf("Tray %d", tray.TrayId)),
			Vendor:           tray.VendorName,
			Model:            tray.PartNumber,
			SerialNum:        tray.SerialNumber,
			SlotNumber:       tray.PhysicalLocation.Slot,
			HealthStatus:     0,
			Status:           models.StorageHWStatusActive,
			SourceUpdateType: models.StorageSourceUpdateStorageAPI,
			Time_Mfy:         time.Now().Local(),
		})
		if err != nil {
			return err
		}
		encByRef[tray.TrayRef] = id
		touchedEnc = append(touchedEnc, id)
	}

	touchedDisk := make([]int64, 0, len(graph.Drive))
	inUse := 0
	var totalCapacityGb float64
	for _, d := range graph.Drive {
		key := "lenovo:drive:" + firstNonEmptyStr(d.WorldWideName, d.SerialNumber, d.DriveRef)
		status := models.StorageHWStatusActive
		if !d.Available && !d.HotSpare {
			// диск занят в volume group - "в использовании"
			inUse++
		} else {
			status = 2 // spare/свободен, тот же код, что и у VNX/Unity
		}
		capGb, cErr := strconv.ParseFloat(strings.TrimSpace(d.RawCapacity), 64)
		if cErr == nil {
			capGb = capGb / (1024 * 1024 * 1024)
		} else {
			capGb = 0
		}
		totalCapacityGb += capGb
		id, err := upsertStorageDisk(o, shd.Id, models.StorageDisk{
			StorageEnclosureId: encByRef[d.PhysicalLocation.TrayRef],
			ExternalKey:        key,
			SlotNumber:         d.PhysicalLocation.Slot,
			Vendor:             d.Manufacturer,
			Model:              d.ProductID,
			DiskType:           d.InterfaceType.DriveType,
			SerialNum:          d.SerialNumber,
			CapacityGb:         capGb,
			Status:             status,
			SourceUpdateType:   models.StorageSourceUpdateStorageAPI,
			Time_Mfy:           time.Now().Local(),
		})
		if err != nil {
			return err
		}
		touchedDisk = append(touchedDisk, id)
	}

	touchedPool := make([]int64, 0)
	pools, poolsErr := session.GetStoragePools()
	if poolsErr != nil {
		// session.GetStoragePools - не фатально для остального
		// hardware-инвентаря (полки/диски уже собраны через GetGraph выше);
		// при ошибке НЕ деактивируем существующие пулы, тот же приём, что у
		// syncUnityHardware/syncVNXHardware (2026-09-16, "для VNX/Unity/Lenovo
		// тоже нужно" - обработка пулов).
		log.Printf("[HW-sync] Lenovo Id=%d: GetStoragePools вернул ошибку (не прерывает синхронизацию hardware): %v", shd.Id, poolsErr)
	} else {
		log.Printf("[HW-sync] Lenovo Id=%d: GetStoragePools вернул %d записей", shd.Id, len(pools))
		for _, p := range pools {
			usedGb, uErr := strconv.ParseFloat(strings.TrimSpace(p.UsedSpace), 64)
			totalGb, tErr := strconv.ParseFloat(strings.TrimSpace(p.TotalRaidedSpace), 64)
			freeGb, fErr := strconv.ParseFloat(strings.TrimSpace(p.FreeSpace), 64)
			if uErr == nil {
				usedGb = usedGb / (1024 * 1024 * 1024)
			} else {
				usedGb = 0
			}
			if tErr == nil {
				totalGb = totalGb / (1024 * 1024 * 1024)
			} else {
				totalGb = 0
			}
			if fErr == nil {
				freeGb = freeGb / (1024 * 1024 * 1024)
			} else {
				freeGb = 0
			}
			usedPercent := 0.0
			if totalGb > 0 {
				usedPercent = usedGb / totalGb * 100
			}
			id, err := upsertStoragePool(o, shd.Id, models.StoragePool{
				ExternalKey: "lenovo:pool:" + p.ID,
				Name:        firstNonEmptyStr(p.Label, "Pool "+p.ID),
				// Description ОБЯЗАТЕЛЬНО заполняется полем модели
				// StoragePool (по прямому запросу пользователя loadmon), но у
				// Lenovo SANtricity API дисковые группы/пулы НЕ имеют
				// отдельного поля description (есть только Label - имя, уже
				// использовано выше как Name) - поэтому здесь Description
				// остаётся пустой строкой, это ограничение API, а не ошибка.
				Description:      "",
				TotalCapacityGb:  totalGb,
				FreeCapacityGb:   freeGb,
				UsedCapacityGb:   usedGb,
				UsedPercent:      usedPercent,
				Status:           models.StorageHWStatusActive,
				SourceUpdateType: models.StorageSourceUpdateStorageAPI,
				Time_Mfy:         time.Now().Local(),
			})
			if err != nil {
				return err
			}
			touchedPool = append(touchedPool, id)
		}
		deactivateAbsentStoragePools(o, shd.Id, touchedPool)
	}

	deactivateAbsentStorageEnclosures(o, shd.Id, touchedEnc)
	deactivateAbsentStorageDisks(o, shd.Id, touchedDisk)
	updateStorageSystemsDiskStats(o, shd, len(graph.Drive), inUse, totalCapacityGb)
	log.Printf("[HW-sync] Lenovo Id=%d: сохранено полок=%d дисков=%d пулов=%d", shd.Id, len(touchedEnc), len(touchedDisk), len(touchedPool))
	return nil
}

// unityDiskTechnologyLabel переводит код diskTechnology Unity REST API
// (объект disk, см. emcstorage.DiskContent.DiskTechnology) в читаемую
// метку. ИСПРАВЛЕНО 2026-09-17 по замечанию пользователя "есть неопознанный
// тип диска с кодом 7" - прежняя версия этой функции использовала
// самодельный маппинг 1-4 (SAS/NL-SAS/SSD/Flash), НЕ совпадающий с
// реальными числовыми кодами Unity DiskTechnologyEnum, которые уже были
// заведены именованными константами в emcstorage/types_add.go (TechSAS=1,
// TechNL_SAS=2, TechSAS_FLASH_2=6, TechSAS_FLASH_3=7, TechSAS_FLASH_4=8,
// TechSAS_FLASH_5=9, TechMixed=50, TechVrtual=99) ещё раньше, но эта функция
// их не использовала. Код 7 из жалобы пользователя - это
// emcstorage.TechSAS_FLASH_3.
func unityDiskTechnologyLabel(code int) string {
	switch code {
	case emcstorage.TechSAS:
		return "SAS"
	case emcstorage.TechNL_SAS:
		return "NL-SAS"
	case emcstorage.TechSAS_FLASH_2:
		return "SAS Flash 2 (SSD)"
	case emcstorage.TechSAS_FLASH_3:
		return "SAS Flash 3 (SSD)"
	case emcstorage.TechSAS_FLASH_4:
		return "SAS Flash 4 (SSD)"
	case emcstorage.TechSAS_FLASH_5:
		return "SAS Flash 5 (SSD)"
	case emcstorage.TechMixed:
		return "Смешанный"
	case emcstorage.TechVrtual:
		return "Виртуальный"
	default:
		return fmt.Sprintf("Код %d", code)
	}
}

// unityDiskParentEnclosureKey вычисляет ключ родительской полки диска Unity
// (для поиска в encByKey, куда сложены DPE/DAE, см. syncUnityHardware) по
// соглашению именования id диска в REST API Unity: id диска в DPE выглядит
// как "dpe_disk_N", id диска в DAE - как "dae_<bus>_<slot>_disk_N", то есть
// id родительской полки - это id диска с отрезанным суффиксом "_disk_N".
// Добавлено 2026-09-17 - раньше принадлежность диска полке для Unity не
// вычислялась вовсе (см. комментарий у вызова в syncUnityHardware).
//
// Возвращает пустую строку, если id диска не соответствует ожидаемому
// формату ("_disk_" не найден) - в этом случае вызывающий код получит "" в
// качестве ключа encByKey, что равносильно ненайденной полке (диск попадёт
// в "без привязки"), то есть не хуже прежнего поведения.
func unityDiskParentEnclosureKey(diskID string) string {
	idx := strings.LastIndex(diskID, "_disk_")
	if idx <= 0 {
		return ""
	}
	parent := diskID[:idx]
	if strings.HasPrefix(parent, "dpe") {
		return "dpe:" + parent
	}
	if strings.HasPrefix(parent, "dae") {
		return "dae:" + parent
	}
	return ""
}

func firstNonEmptyStr(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
