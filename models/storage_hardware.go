package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Типы полок СХД (StorageEnclosure.EnclosureType).
const (
	StorageEnclosureTypeUnknown    = 0
	StorageEnclosureTypeManagement = 1 // DPE / control shelf / полка управления
	StorageEnclosureTypeData       = 2 // DAE / disk shelf / полка с данными
)

// Статусы записей hardware-слоя СХД.
const (
	StorageHWStatusRemoved = 0
	StorageHWStatusActive  = 1
)

// StorageEnclosure — полка СХД (управления или с данными), синхронизируется
// из API/CLI массива (Unity DPE/DAE, VNX enclosure и т.д.).
type StorageEnclosure struct {
	Id               int64     `form:"Id" orm:"auto;pk"`
	StorageSystemsId int64     `form:"StorageSystemsId" orm:"index"`
	EnclosureType    int       `form:"EnclosureType"` // StorageEnclosureType*
	ExternalKey      string    `form:"ExternalKey" orm:"size(128);index"` // стабильный ключ в рамках СХД (dpe:..., dae:..., vnx:enc:N)
	Name             string    `form:"Name" orm:"size(128)"`
	Vendor           string    `form:"Vendor" orm:"size(64)"`
	Model            string    `form:"Model" orm:"size(128)"`
	SerialNum        string    `form:"SerialNum" orm:"size(64);index"`
	SlotNumber       int       `form:"SlotNumber"`
	BusId            int       `form:"BusId"`
	HealthStatus     int       `form:"HealthStatus"` // код health API, 0 если неизвестно
	Status           int       `form:"Status"`       // StorageHWStatus*
	SourceUpdateType int       `form:"SourceUpdateType"`
	Time_Mfy         time.Time `form:"Time_Mfy"`
}

func (m *StorageEnclosure) TableName() string { return "storage_enclosure" }

func (m *StorageEnclosure) TableUnique() [][]string {
	return [][]string{{"StorageSystemsId", "ExternalKey"}}
}

// TableIndex() оставляет только EnclosureType - у StorageSystemsId и
// SerialNum уже есть тег `orm:"...;index"` на поле (см. выше), повторение
// тех же колонок здесь давало два одинаковых индекса и падение RunSyncdb с
// "index ... already exists" на чистой базе (см. models/storage_metric.go
// и CHANGELOG за подробным объяснением этого паттерна).
func (m *StorageEnclosure) TableIndex() [][]string {
	return [][]string{
		{"EnclosureType"},
	}
}

// StorageDisk — физический диск СХД с типом и моделью.
type StorageDisk struct {
	Id                 int64     `form:"Id" orm:"auto;pk"`
	StorageSystemsId   int64     `form:"StorageSystemsId" orm:"index"`
	StorageEnclosureId int64     `form:"StorageEnclosureId" orm:"index"`
	ExternalKey        string    `form:"ExternalKey" orm:"size(128);index"`
	SlotNumber         int       `form:"SlotNumber"`
	BusId              int       `form:"BusId"`
	EnclosureNum       int       `form:"EnclosureNum"`
	Vendor             string    `form:"Vendor" orm:"size(64)"`
	Model              string    `form:"Model" orm:"size(128)"`
	DiskType           string    `form:"DiskType" orm:"size(64)"` // SAS/SSD/NLSAS/…
	SerialNum          string    `form:"SerialNum" orm:"size(64);index"`
	CapacityGb         float64   `form:"CapacityGb"`
	RaidGroupId        string    `form:"RaidGroupId" orm:"size(64)"`
	Status             int       `form:"Status"` // 1=in use, 2=spare, 0=removed
	SourceUpdateType   int       `form:"SourceUpdateType"`
	Time_Mfy           time.Time `form:"Time_Mfy"`
}

func (m *StorageDisk) TableName() string { return "storage_disk" }

func (m *StorageDisk) TableUnique() [][]string {
	return [][]string{{"StorageSystemsId", "ExternalKey"}}
}

// TypeLabel возвращает DiskType в читаемом виде для отображения в карточках
// (server_web_new/views/sprsrv/storagesystemconfigview.tpl и т.п.).
//
// У всех вендоров, кроме Huawei, DiskType уже сохраняется читаемой строкой
// (unityDiskTechnologyLabel, VNX DriveType и т.д.) - для них TypeLabel просто
// возвращает поле как есть. У Huawei DISKTYPE декодируется в читаемую метку
// на этапе синхронизации (huaweiDiskTypeLabel в cmdb/syncstoragehardware.go,
// делегирует сюда, в HuaweiDiskTypeLabel), но записи, синхронизированные ДО
// этого исправления (2026-09-22), в БД остались с "сырым" числовым кодом
// ("14", "21" и т.п.) - пока СХД не пройдёт очередной цикл опроса.
// TypeLabel - подстраховка на это переходное время: если DiskType состоит
// только из цифр (что у других вендоров не встречается), расшифровывает его
// по той же таблице, не дожидаясь пересинхронизации.
func (m *StorageDisk) TypeLabel() string {
	if m.DiskType == "" {
		return ""
	}
	if isDigitsOnly(m.DiskType) {
		return HuaweiDiskTypeLabel(m.DiskType)
	}
	return m.DiskType
}

func isDigitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// HuaweiDiskTypeLabel переводит числовой код DISKTYPE Huawei DeviceManager
// REST API (Dorado/OceanStor, поле disk.DISKTYPE, приходит строкой) в
// читаемую метку. Источник таблицы - официальный MIB
// HUAWEI-STORAGE-HARDWARE-MIB, hwInfoDiskType (коды 0-18), + код 21,
// добавленный позже в V700R001C01 для Capacity-Optimized/QLC SSD (см.
// kontext/10-huawei-dorado-rest-api.md). Коды 19/20/22 в открытом MIB не
// задокументированы - для них, как и для любого другого нераспознанного
// кода, возвращается "Код N", ничего не домысливая.
func HuaweiDiskTypeLabel(code string) string {
	code = strings.TrimSpace(code)
	n, err := strconv.Atoi(code)
	if err != nil {
		// на случай прошивки, которая всё же прислала не число, а готовую
		// строку (как в CLI) - возвращаем как есть, ничего не портим
		return code
	}
	switch n {
	case 0:
		return "FC (HDD Fibre Channel)"
	case 1:
		return "SAS (HDD)"
	case 2:
		return "SATA (HDD)"
	case 3:
		return "SAS SSD"
	case 4:
		return "NL-SAS (HDD)"
	case 5:
		return "SLC SSD"
	case 6:
		return "MLC SSD"
	case 7:
		return "FC SED"
	case 8:
		return "SAS SED (HDD)"
	case 9:
		return "SATA SED (HDD)"
	case 10:
		return "SAS SSD SED"
	case 11:
		return "NL-SAS SED (HDD)"
	case 12:
		return "SLC SSD SED"
	case 13:
		return "MLC SSD SED"
	case 14:
		return "NVMe SSD"
	case 15:
		return "SMR SATA (HDD)"
	case 16:
		return "NVMe SSD SED"
	case 17:
		return "SCM SSD (Storage Class Memory)"
	case 18:
		return "SCM SED"
	case 21:
		return "Capacity-Optimized NVMe SSD (QLC)"
	default:
		return fmt.Sprintf("Код %d", n)
	}
}

// TableIndex() намеренно не объявлен - StorageSystemsId/StorageEnclosureId/
// SerialNum уже проиндексированы тегом `orm:"...;index"` на самих полях
// (см. выше); повтор тех же колонок здесь давал дублирующиеся индексы и
// падение RunSyncdb на чистой базе (см. models/storage_metric.go и
// CHANGELOG).

// StoragePool — логический пул хранения СХД (Huawei Dorado/OceanStor
// storagepool, Lenovo storage-pool и т.п. - модель намеренно вендор-
// независимая, как StorageEnclosure/StorageDisk). Добавлено 2026-09-16 по
// прямому запросу пользователя loadmon "добавь обработку пулов и обязательно
// с полем description" (продолжение задачи по расширению go-dorado-sdk
// дисками/полками - см. cmdb.syncHuaweiHardware) - Description ОБЯЗАТЕЛЬНО
// заполняется из вендорского поля DESCRIPTION (у Huawei StoragePools.
// DESCRIPTION приходит пустой строкой, если пул не аннотирован
// администратором СХД - это нормально, не ошибка синхронизации).
type StoragePool struct {
	Id               int64     `form:"Id" orm:"auto;pk"`
	StorageSystemsId int64     `form:"StorageSystemsId" orm:"index"`
	ExternalKey      string    `form:"ExternalKey" orm:"size(128);index"` // стабильный ключ в рамках СХД (huawei:pool:<ID> и т.п.)
	Name             string    `form:"Name" orm:"size(128)"`
	Description      string    `form:"Description" orm:"size(255)"` // DESCRIPTION у вендорского API, может быть пустым
	HealthStatus     int       `form:"HealthStatus"`                // код health API, 0 если неизвестно
	TotalCapacityGb  float64   `form:"TotalCapacityGb"`
	FreeCapacityGb   float64   `form:"FreeCapacityGb"`
	UsedCapacityGb   float64   `form:"UsedCapacityGb"`
	UsedPercent      float64   `form:"UsedPercent"`
	Status           int       `form:"Status"` // StorageHWStatus*
	SourceUpdateType int       `form:"SourceUpdateType"`
	Time_Mfy         time.Time `form:"Time_Mfy"`
}

func (m *StoragePool) TableName() string { return "storage_pool" }

func (m *StoragePool) TableUnique() [][]string {
	return [][]string{{"StorageSystemsId", "ExternalKey"}}
}
