package models

import "time"

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

func (m *StorageEnclosure) TableIndex() [][]string {
	return [][]string{
		{"StorageSystemsId"},
		{"EnclosureType"},
		{"SerialNum"},
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

func (m *StorageDisk) TableIndex() [][]string {
	return [][]string{
		{"StorageSystemsId"},
		{"StorageEnclosureId"},
		{"SerialNum"},
	}
}
