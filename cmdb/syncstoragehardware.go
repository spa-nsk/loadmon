package cmdb

import (
	"context"
	"fmt"
	"loadmon/lenovostorage"
	models "loadmon/models"
	"loadmon/storages"
	datamap "loadmon/utilities"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/lovi-cloud/go-dorado-sdk/dorado"
)

// SyncStorageHardwareForSystem синхронизирует полки и диски одной СХД после
// успешного опроса томов (вызывается из UpdateVolumeStorage) - обязательно
// по КАЖДОМУ вендору, представленному в справочнике СХД (2026-09-03):
//   - VNX (1) и Unity (2) - полная инвентаризация полок и дисков.
//   - Lenovo (5) - полная инвентаризация полок/дисков через GetGraph
//     (Tray/Drive), см. syncLenovoHardware.
//   - Dorado/OceanStor (3, 4) - используемый SDK НЕ даёт листинг физических
//     дисков/полок (нет соответствующих методов - см. ревью в этой сессии),
//     поэтому вместо полного hardware-инвентаря обновляется только сводная
//     сырая ёмкость СХД (StorageSystems.TotalSizeRAW/UsedPercent) из
//     GetSystem - см. syncHuaweiSystemTotals. Это ограничение конкретного
//     SDK, а не пропуск вендора.
func SyncStorageHardwareForSystem(o orm.Ormer, shd *models.StorageSystems) error {
	if shd == nil || shd.Id <= 0 || shd.Power != 1 {
		return nil
	}
	switch shd.ApiType {
	case 1:
		return syncVNXHardware(o, shd)
	case 2:
		return syncUnityHardware(o, shd)
	case 3, 4:
		return syncHuaweiSystemTotals(o, shd)
	case 5:
		return syncLenovoHardware(o, shd)
	default:
		return nil
	}
}

func syncUnityHardware(o orm.Ormer, shd *models.StorageSystems) error {
	encByKey := map[string]int64{}
	touchedEnc := make([]int64, 0)
	touchedDisk := make([]int64, 0)

	dpeList, err := storages.GetUnityDPE(shd.IpAddress)
	if err != nil {
		return err
	}
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

	daeList, err := storages.GetUnityDAE(shd.IpAddress)
	if err != nil {
		return err
	}
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

	disks, err := storages.GetUnityDisks(shd.IpAddress)
	if err != nil {
		return err
	}
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
		id, err := upsertStorageDisk(o, shd.Id, models.StorageDisk{
			ExternalKey:      key,
			SlotNumber:       int(d.SlotNumber),
			BusId:            int(d.BusId),
			Vendor:           d.Manufacturer,
			Model:            d.Model,
			DiskType:         unityDiskTechnologyLabel(d.DiskTechnology),
			SerialNum:        d.EmcSerialNumber,
			CapacityGb:       capGb,
			Status:           status,
			SourceUpdateType: models.StorageSourceUpdateStorageAPI,
			Time_Mfy:         time.Now().Local(),
		})
		if err != nil {
			return err
		}
		touchedDisk = append(touchedDisk, id)
	}

	deactivateAbsentStorageEnclosures(o, shd.Id, touchedEnc)
	deactivateAbsentStorageDisks(o, shd.Id, touchedDisk)
	updateStorageSystemsDiskStats(o, shd, len(disks), inUse, totalCapacityGb)
	return nil
}

func syncVNXHardware(o orm.Ormer, shd *models.StorageSystems) error {
	disks, err := storages.GetDisksCmd(shd.IpAddress)
	if err != nil {
		return err
	}
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

	deactivateAbsentStorageEnclosures(o, shd.Id, touchedEnc)
	deactivateAbsentStorageDisks(o, shd.Id, touchedDisk)
	updateStorageSystemsDiskStats(o, shd, len(disks), inUse, totalCapacityGb)
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
// СХД Huawei Dorado/OceanStor из GetSystem() - единственное, что доступно
// для этого вендора в подключённом SDK без per-диск/per-полка листинга (см.
// комментарий у SyncStorageHardwareForSystem). TOTALCAPACITY у Huawei API
// приходит в Мб (см. GetDoradoLun/GetHuaweyVolume - там же встречается тот
// же паттерн перевода capacity/sectorsize в Гб), поэтому переводится в Гб
// делением на 1024.
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

	deactivateAbsentStorageEnclosures(o, shd.Id, touchedEnc)
	deactivateAbsentStorageDisks(o, shd.Id, touchedDisk)
	updateStorageSystemsDiskStats(o, shd, len(graph.Drive), inUse, totalCapacityGb)
	return nil
}

func unityDiskTechnologyLabel(code int) string {
	switch code {
	case 1:
		return "SAS"
	case 2:
		return "NL-SAS"
	case 3:
		return "SSD"
	case 4:
		return "Flash"
	default:
		return strconv.Itoa(code)
	}
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
