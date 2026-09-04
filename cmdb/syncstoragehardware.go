package cmdb

import (
	"fmt"
	"loadmon/storages"
	models "loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// SyncStorageHardwareForSystem синхронизирует полки и диски одной СХД после
// успешного опроса томов (вызывается из UpdateVolumeStorage).
func SyncStorageHardwareForSystem(o orm.Ormer, shd *models.StorageSystems) error {
	if shd == nil || shd.Id <= 0 || shd.Power != 1 {
		return nil
	}
	switch shd.ApiType {
	case 1:
		return syncVNXHardware(o, shd)
	case 2:
		return syncUnityHardware(o, shd)
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
	for _, d := range disks {
		key := "disk:" + firstNonEmptyStr(d.WWN, d.EmcSerialNumber, d.ID)
		status := models.StorageHWStatusActive
		if !d.IsInUse {
			status = 2
		} else {
			inUse++
		}
		capGb := float64(d.Size) / (1024 * 1024 * 1024)
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
	updateStorageSystemsDiskStats(o, shd, len(disks), inUse)
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
	updateStorageSystemsDiskStats(o, shd, len(disks), inUse)
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

func updateStorageSystemsDiskStats(o orm.Ormer, shd *models.StorageSystems, total, inUse int) {
	if total <= 0 {
		return
	}
	shd.Size = total
	shd.Used = inUse
	_, _ = o.Update(shd, "Size", "Used")
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
