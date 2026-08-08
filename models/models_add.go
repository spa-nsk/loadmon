package models

import (
	"time"
)

// блок для ролевой модели и типовых конфигураций

// описание модели-норматива все нормативные модели: системные блоки, мониторы, ноутбуки, телефоны ...
type ModelPersonalEquipment struct {
	Id          int64     `form:"Id"`          //уникальный идентификатор
	Code        string    `form:"Code"`        //идентификационный код модели-норматива
	Time        time.Time `form:"Time"`        //год актуализации модели-норматива
	ShortName   string    `form:"ShortName"`   //краткое наименование модели-норматива
	Description string    `form:"Description"` //текстовое описание конфигурации
	Money       float64   `form:"Money"`       //цена
	Currency    int       `form:"Currency"`    //валюта цены 0 -не указано 1-Российские рубли 2-USD 3-Юани 4-Евро
	//добавить детали конфигурации
}

func (a *ModelPersonalEquipment) TableName() string {
	return "model_personal_equipment"
}

func (u *ModelPersonalEquipment) TableIndex() [][]string {
	return [][]string{
		[]string{"Code"},
		[]string{"Code", "Time"},
		[]string{"ShortName", "Time"},
	}
}

// Описание роли для обеспечения оргтехникой: обобщенные потребности сотрудников использующие одинаковые наборы оборудования в работе
type RolePersonal struct { //
	Id           int64   `form:"Id"`           //уникальный идентификатор
	Code         string  `form:"Code"`         //идентификационный код роли
	CodeJira     string  `form:"CodeJira"`     //ссылка на профиль в jira
	BusinessWork string  `form:"BusinessWork"` //бизнес, цк ...
	FormatWork   int     `form:"FormatWork"`   //формат работы 0-не указано, 1-гибридный, 2-удаленный
	ShortName    string  `form:"ShortName"`    //наименование роли
	Description  string  `form:"Description"`  //подробное описание
	Status       int     `form:"Status"`       // 0- не проверена 1-подтверждена 2-отменена
	Jira         int     `form:"Jira"`         // 0- не указано 1-внесено в jira 2-требуется помощь
	MoneySumm    float64 `form:"MoneySumm"`    //сумма по роли
}

func (a *RolePersonal) TableName() string {
	return "role_personal"
}

func (u *RolePersonal) TableIndex() [][]string {
	return [][]string{
		[]string{"Code"},
		[]string{"CodeJira"},
		[]string{"BusinessWork"},
		[]string{"ShortName"},
	}
}

type RoleCompetenceCenter struct { // список ЦК или должностей для роли
	Id             int64  `form:"Id"`             // уникальный идентификатор
	CodeRole       string `form:"CodeRole"`       // идентификационный код роли
	CodeCompetence string `form:"CodeCompetence"` //
}

func (a *RoleCompetenceCenter) TableName() string {
	return "role_competence_center"
}

func (u *RoleCompetenceCenter) TableIndex() [][]string {
	return [][]string{
		[]string{"CodeRole"},
	}
}

type CompetenceCenter struct { // jira https://jira.bcs.ru/secure/ObjectSchema.jspa?id=3&typeId=214&view=list&objectId=3061492
	Id        int64  `form:"Id"`        // уникальный идентификатор
	Code      string `form:"Code"`      // идентификационный код для всех
	CodeJira  string `form:"CodeJira"`  // идентификационный код ЦК из jira CMDB-0000000 только ИТ
	Name      string `form:"Name"`      // Наименование
	ShortName string `form:"ShortName"` // мнемоническое наименование
	Status    int    `form:"Status"`    // 0-не указан 1-активен 2-не активен 3-требует действия
}

func (a *CompetenceCenter) TableName() string {
	return "competence_center"
}

func (u *CompetenceCenter) TableIndex() [][]string {
	return [][]string{
		[]string{"Code"},
		[]string{"CodeJira"},
	}
}

// привязка оборудования к роли с кодом CodeRole
type RolePersonalItem struct {
	Id            int64  `form:"Id"`            //уникальный идентификатор
	CodeRole      string `form:"CodeRole"`      //идентификационный код роли
	CodeEquipment string `form:"CodeEquipment"` //код модели оборудования
	Count         int    `form:"Count"`         //количество единиц оборудования
}

func (a *RolePersonalItem) TableName() string {
	return "role_personal_item"
}

func (u *RolePersonalItem) TableIndex() [][]string {
	return [][]string{
		[]string{"CodeRole"},
	}
}

// окончание блока ролевых моделей и типовых конфигураций
type ReportServerRecord struct {
	Id                     int64     `form:"Id"`
	Time                   time.Time `form:"Time"`
	Virtual                bool      `form:"Virtual"`
	ServerName             string    `form:"ServerName"`
	ServerId               int64     `form:"ServerId"` //ссылка на справочник
	Inventory              string    `form:"Inventory"`
	CPUCount               float64   `form:"CPUCount"`
	CoreCount              float64   `form:"CoreCount"`
	MemSize                float64   `form:"MemSize"`
	SysDiskSize            float64   `form:"SysDiskSize"`
	WorkDiskSize           float64   `form:"WorkDiskSize"`
	CPULoadAvg             float64   `form:"CPULoadAvg"`
	CPULoadMin             float64   `form:"CPULoadMin"`
	CPULoadMax             float64   `form:"CPULoadMax"`
	MemUseAvg              float64   `form:"MemUseAvg"`
	MemUseMin              float64   `form:"MemUseMin"`
	MemUseMax              float64   `form:"MemUseMax"`
	SysDiskFreePercentAvg  float64   `form:"SysDiskFreePercentAvg"`
	SysDiskFreePercentMin  float64   `form:"SysDiskFreePercentMin"`
	SysDiskFreePercentMax  float64   `form:"SysDiskFreePercentMax"`
	SysDiskIdleTimeAvg     float64   `form:"SysDiskIdleTimeAvg"`
	SysDiskIdleTimeMin     float64   `form:"SysDiskIdleTimeMin"`
	SysDiskIdleTimeMax     float64   `form:"SysDiskIdleTimeMax"`
	SysDiskQueueAvg        float64   `form:"SysDiskQueueAvg"`
	SysDiskQueueMin        float64   `form:"SysDiskQueueMin"`
	SysDiskQueueMax        float64   `form:"SysDiskQueueMax"`
	WorkDiskFreePercentAvg float64   `form:"WorkDiskFreePercentAvg"`
	WorkDiskFreePercentMin float64   `form:"WorkDiskFreePercentMin"`
	WorkDiskFreePercentMax float64   `form:"WorkDiskFreePercentMax"`
	WorkDiskIdleTimeAvg    float64   `form:"WorkDiskIdleTimeAvg"`
	WorkDiskIdleTimeMin    float64   `form:"WorkDiskIdleTimeMin"`
	WorkDiskIdleTimeMax    float64   `form:"WorkDiskIdleTimeMax"`
	WorkDiskQueueAvg       float64   `form:"WorkDiskQueueAvg"`
	WorkDiskQueueMin       float64   `form:"WorkDiskQueueMin"`
	WorkDiskQueueMax       float64   `form:"WorkDiskQueueMax"`
	SysDiskIOAvg           float64   `form:"SysDiskIOAvg"`
	SysDiskIOMin           float64   `form:"SysDiskIOMin"`
	SysDiskIOMax           float64   `form:"SysDiskIOMax"`
	SysDiskFreeMbAvg       float64   `form:"SysDiskFreeMbAvg"`
	SysDiskFreeMbMin       float64   `form:"SysDiskFreeMbMin"`
	SysDiskFreeMbMax       float64   `form:"SysDiskFreeMbMax"`
	WorkDiskIOAvg          float64   `form:"WorkDiskIOAvg"`
	WorkDiskIOMin          float64   `form:"WorkDiskIOMin"`
	WorkDiskIOMax          float64   `form:"WorkDiskIOMax"`
	WorkDiskFreeMbAvg      float64   `form:"WorkDiskFreeMbAvg"`
	WorkDiskFreeMbMin      float64   `form:"WorkDiskFreeMbMin"`
	WorkDiskFreeMbMax      float64   `form:"WorkDiskFreeMbMax"`
	Owner                  string    `form:"Owner"`
	Allocation             string    `form:"Allocation"`
	Anotation              string    `form:"Anotation"`
	Datacenter             string    `form:"Datacenter"`
	Claster                string    `form:"Claster"`
	Service                string    `form:"Service"`
	Busines                string    `form:"Busines"`
	BusinesLn              string    `form:"BusinesLn"`
	CenterMonney           string    `form:"CenterMonney"`
	UseWorkDisk            bool      `form:"-"`
}

type VMRecord struct {
	Id                   int64     `form:"Id" csv:"-"`
	ServerId             int64     `form:"ServerId" csv:"-"` //новая ссылка на справочник серверов. Добавить запись значений.
	VMsprId              int64     `form:"VMsprId" csv:"-"`  //ссылка на справочник VM
	ClusterId            int64     `form:"ClusterId" csv:"-"`
	Date                 time.Time `form:"Date" csv:"-"`
	Vm_NumCPU            int32     `form:"Vm_NumCPU" csv:"-"`
	Vm_NumCores          int32     `form:"Vm_NumCores" csv:"-"`
	Vm_MemorySizeGB      float64   `form:"Vm_MemorySizeGB" csv:"-"`
	Vm_IpAddress         string    `form:"Vm_IpAddress" csv:"-"`
	Vm_Network           string    `form:"Vm_Network" csv:"-"`
	Vm_NumEthernetCards  int32     `form:"Vm_NumEthernetCards" csv:"-"`
	Vm_NumVirtualDisks   int32     `form:"Vm_NumVirtualDisks" csv:"-"`
	Vm_DiskCommittedGB   float64   `form:"Vm_DiskCommittedGB" csv:"-"`
	Vm_DiskUncommittedGB float64   `form:"Vm_DiskUncommittedGB" csv:"-"`
	Vm_CapacityGb        float64   `form:"Vm_CapacityGb" csv:"VM DISK (Gb)"`
	Vm_FreeSpaceGb       float64   `form:"Vm_FreeSpaceGb" csv:"VM DISK Free (Gb)"`
	Vm_PercentUsed       float64   `form:"Vm_PercentUsed" csv:"VM DISK Used %"`
	Cpu_usage_maximum    float64   `form:"Cpu_usage_maximum" csv:"VM CPU max %"`
	Cpu_usage_maximum_time time.Time `orm:"null" form:"Cpu_usage_maximum_time" csv:"VM CPU max time"`
	Cpu_usage_average    float64   `form:"Cpu_usage_average" csv:"VM CPU avg %"`
	Cpu_usage_minimum    float64   `form:"Cpu_usage_minimum" csv:"VM CPU min %"`
	Cpu_usage_minimum_time time.Time `orm:"null" form:"Cpu_usage_minimum_time" csv:"VM CPU min time"`
	Mem_usage_maximum    float64   `form:"Mem_usage_maximum" csv:"VM MEM max %"`
	Mem_usage_maximum_time time.Time `orm:"null" form:"Mem_usage_maximum_time" csv:"VM MEM max time"`
	Mem_usage_average    float64   `form:"Mem_usage_average" csv:"VM MEM avg %"`
	Mem_usage_minimum    float64   `form:"Mem_usage_minimum" csv:"VM MEM min %"`
	Mem_usage_minimum_time time.Time `orm:"null" form:"Mem_usage_minimum_time" csv:"VM MEM min time"`
	Disk_usage_maximum   float64   `form:"Disk_usage_maximum" csv:"VM DISK Usage max %"`
	Disk_usage_maximum_time time.Time `orm:"null" form:"Disk_usage_maximum_time" csv:"VM DISK Usage max time"`
	Disk_usage_average   float64   `form:"Disk_usage_average" csv:"VM DISK Usage avg %"`
	Disk_usage_minimum   float64   `form:"Disk_usage_minimum" csv:"VM DISK Usage min %"`
	Disk_usage_minimum_time time.Time `orm:"null" form:"Disk_usage_minimum_time" csv:"VM DISK Usage min time"`
	Disk_totalReadLatency_maximum float64 `form:"Disk_totalReadLatency_maximum" csv:"-"`
	Disk_totalReadLatency_maximum_time time.Time `orm:"null" form:"Disk_totalReadLatency_maximum_time" csv:"-"`
	Disk_totalReadLatency_average float64 `form:"Disk_totalReadLatency_average" csv:"-"`
	Disk_totalReadLatency_minimum float64 `form:"Disk_totalReadLatency_minimum" csv:"-"`
	Disk_totalReadLatency_minimum_time time.Time `orm:"null" form:"Disk_totalReadLatency_minimum_time" csv:"-"`
	Disk_totalWriteLatency_maximum float64 `form:"Disk_totalWriteLatency_maximum" csv:"-"`
	Disk_totalWriteLatency_maximum_time time.Time `orm:"null" form:"Disk_totalWriteLatency_maximum_time" csv:"-"`
	Disk_totalWriteLatency_average float64 `form:"Disk_totalWriteLatency_average" csv:"-"`
	Disk_totalWriteLatency_minimum float64 `form:"Disk_totalWriteLatency_minimum" csv:"-"`
	Disk_totalWriteLatency_minimum_time time.Time `orm:"null" form:"Disk_totalWriteLatency_minimum_time" csv:"-"`
	Disk_write_maximum   float64   `form:"Disk_write_maximum" csv:"-"`
	Disk_write_maximum_time time.Time `orm:"null" form:"Disk_write_maximum_time" csv:"-"`
	Disk_write_average   float64   `form:"Disk_write_average" csv:"-"`
	Disk_write_minimum   float64   `form:"Disk_write_minimum" csv:"-"`
	Disk_write_minimum_time time.Time `orm:"null" form:"Disk_write_minimum_time" csv:"-"`
	Disk_read_maximum    float64   `form:"Disk_read_maximum" csv:"-"`
	Disk_read_maximum_time time.Time `orm:"null" form:"Disk_read_maximum_time" csv:"-"`
	Disk_read_average    float64   `form:"Disk_read_average" csv:"-"`
	Disk_read_minimum    float64   `form:"Disk_read_minimum" csv:"-"`
	Disk_read_minimum_time time.Time `orm:"null" form:"Disk_read_minimum_time" csv:"-"`
	Net_usage_maximum    float64   `form:"Net_usage_maximum" csv:"-"`
	Net_usage_maximum_time time.Time `orm:"null" form:"Net_usage_maximum_time" csv:"-"`
	Net_usage_average    float64   `form:"Net_usage_average" csv:"-"`
	Net_usage_minimum    float64   `form:"Net_usage_minimum" csv:"-"`
	Net_usage_minimum_time time.Time `orm:"null" form:"Net_usage_minimum_time" csv:"-"`
	CpuBilling           float64   `form:"CpuBilling" csv:"-"`  // в мегагерц часах
	MemBilling           float64   `form:"MemBilling" csv:"-"`  // в мегабайт часах
	DiskBilling          float64   `form:"DiskBilling" csv:"-"` // по занимаемому месту в гигабайт часах
	NetBilling           float64   `form:"NetBilling" csv:"-"`  // в мегабайт часах
}

func (a *VMRecord) TableName() string {
	return "vm_record"
}

func (u *VMRecord) TableIndex() [][]string {
	return [][]string{
		[]string{"Date"},
		[]string{"VMSprId"},
		[]string{"VMSprId", "Date"},
	}
}

type VMRecordUsageOnDatastore struct {
	Id                     int64   `form:"Id"`
	VMRecordId             int64   `form:"VMRecordId"`
	DatastoreId            int64   `form:"DatastoreId"`
	CommittedGb            float64 `form:"Committed"`
	UncommittedGb          float64 `form:"Uncommitted"`
	UnsharedGb             float64 `form:"Unshared"`
	PercentAllocateStorage float64 `form:"PercentAllocateStorage"` //процент выделения места под виртуальную магину на конкретном хранилище
}

func (a *VMRecordUsageOnDatastore) TableName() string {
	return "vm_record_usage_datastore"
}

func (u *VMRecordUsageOnDatastore) TableIndex() [][]string {
	return [][]string{
		[]string{"VMRecordId"},
		[]string{"DatastoreId"},
	}
}

type VMRecordGuestDiskInfo struct {
	Id          int64   `form:"Id"`
	VMRecordId  int64   `form:"VMRecordId"`
	DiskPath    string  `form:"DiskPath"`
	CapacityGb  float64 `form:"CapacityGb"`
	FreeSpaceGb float64 `form:"FreeSpaceGb"`
	PercentUsed float64 `form:"PercentUsed"`
}

func (a *VMRecordGuestDiskInfo) TableName() string {
	return "vm_record_guest_disk_info"
}

func (u *VMRecordGuestDiskInfo) TableIndex() [][]string {
	return [][]string{
		[]string{"VMRecordId"},
		[]string{"VMRecordId", "Id"},
	}
}

type VMSprRecordView struct {
	VMSprRecord
	Disks       []VMGuestDiskInfo `form:"Disks"`
	VMPerf      []VMPerfRecord    `form:"VMPerf"`
	CapacityGb  float64           `form:"CapacityGb"`
	PercentUsed float64           `form:"PercentUsed"`
}

type HostRecord struct {
	Id                         int64     `form:"Id" csv:"-"`
	ServerId                   int64     `form:"ServerId" csv:"-"` //ссылка на справочник серверов (хостов)
	ClusterId                  int64     `form:"ClusterId" csv:"-"`
	Date                       time.Time `form:"Date" csv:"-"`
	Vcenter                    string    `form:"Vcenter" csv:"-"`
	Datacenter                 string    `form:"Datacenter" csv:"-"`
	Cluster                    string    `form:"Cluster" csv:"-"`
	Name                       string    `form:"Name" csv:"-"`
	Vendor                     string    `form:"Vendor"` //HostSystemInfo
	Model                      string    `form:"Model"`
	CpuModel                   string    `form:"CpuModel"`
	CpuMhz                     int32     `form:"CpuMhz"`
	NumCpuPkgs                 int16     `form:"NumCpuPkgs"`
	NumCpuCores                int16     `form:"NumCpuCores"`
	NumCpuThreads              int16     `form:"NumCpuThreads"`
	NumNics                    int32     `form:"NumNics"`
	IpAddress                  string    `form:"IpAddress"`
	NumHBAs                    int32     `form:"NumHBAs"`
	MemorySizeGB               float64   `form:"MemorySizeGB"`
	HostMaxVirtualDiskCapacity float64   `form:"HostMaxVirtualDiskCapacity"`
	Cpu_usage_maximum          float64   `orm:"null" form:"Cpu_usage_maximum" csv:"HOST CPU Usage max %"`
	Cpu_usage_maximum_time     time.Time `orm:"null" form:"Cpu_usage_maximum_time" csv:"-"`
	Cpu_usage_average          float64   `orm:"null" form:"Cpu_usage_average" csv:"HOST CPU Usage avg %"`
	Cpu_usage_minimum          float64   `orm:"null" form:"Cpu_usage_minimum" csv:"HOST CPU Usage min %"`
	Cpu_usage_minimum_time     time.Time `orm:"null" form:"Cpu_usage_minimum_time" csv:"-"`
	Mem_usage_maximum          float64   `orm:"null" form:"Mem_usage_maximum" csv:"HOST MEM Usage max %"`
	Mem_usage_maximum_time     time.Time `orm:"null" form:"Mem_usage_maximum_time" csv:"-"`
	Mem_usage_average          float64   `orm:"null" form:"Mem_usage_average" csv:"HOST MEM Usage avg %"`
	Mem_usage_minimum          float64   `orm:"null" form:"Mem_usage_minimum" csv:"HOST MEM Usage min %"`
	Mem_usage_minimum_time     time.Time `orm:"null" form:"Mem_usage_minimum_time" csv:"-"`
	Disk_usage_maximum         float64   `orm:"null" form:"Disk_usage_maximum" csv:"-"`
	Disk_usage_maximum_time    time.Time `orm:"null" form:"Disk_usage_maximum_time" csv:"-"`
	Disk_usage_average         float64   `orm:"null" form:"Disk_usage_average" csv:"-"`
	Disk_usage_minimum         float64   `orm:"null" form:"Disk_usage_minimum" csv:"-"`
	Disk_usage_minimum_time    time.Time `orm:"null" form:"Disk_usage_minimum_time" csv:"-"`
	Disk_totalReadLatency_maximum float64 `orm:"null" form:"Disk_totalReadLatency_maximum" csv:"-"`
	Disk_totalReadLatency_maximum_time time.Time `orm:"null" form:"Disk_totalReadLatency_maximum_time" csv:"-"`
	Disk_totalReadLatency_average float64 `orm:"null" form:"Disk_totalReadLatency_average" csv:"-"`
	Disk_totalReadLatency_minimum float64 `orm:"null" form:"Disk_totalReadLatency_minimum" csv:"-"`
	Disk_totalReadLatency_minimum_time time.Time `orm:"null" form:"Disk_totalReadLatency_minimum_time" csv:"-"`
	Disk_totalWriteLatency_maximum float64 `orm:"null" form:"Disk_totalWriteLatency_maximum" csv:"-"`
	Disk_totalWriteLatency_maximum_time time.Time `orm:"null" form:"Disk_totalWriteLatency_maximum_time" csv:"-"`
	Disk_totalWriteLatency_average float64 `orm:"null" form:"Disk_totalWriteLatency_average" csv:"-"`
	Disk_totalWriteLatency_minimum float64 `orm:"null" form:"Disk_totalWriteLatency_minimum" csv:"-"`
	Disk_totalWriteLatency_minimum_time time.Time `orm:"null" form:"Disk_totalWriteLatency_minimum_time" csv:"-"`
	Disk_write_maximum         float64   `orm:"null" form:"Disk_write_maximum" csv:"-"`
	Disk_write_maximum_time    time.Time `orm:"null" form:"Disk_write_maximum_time" csv:"-"`
	Disk_write_average         float64   `orm:"null" form:"Disk_write_average" csv:"-"`
	Disk_write_minimum         float64   `orm:"null" form:"Disk_write_minimum" csv:"-"`
	Disk_write_minimum_time    time.Time `orm:"null" form:"Disk_write_minimum_time" csv:"-"`
	Disk_read_maximum          float64   `orm:"null" form:"Disk_read_maximun" csv:"-"`
	Disk_read_maximum_time     time.Time `orm:"null" form:"Disk_read_maximum_time" csv:"-"`
	Disk_read_average          float64   `orm:"null" form:"Disk_read_average" csv:"-"`
	Disk_read_minimum          float64   `orm:"null" form:"Disk_read_minimum" csv:"-"`
	Disk_read_minimum_time     time.Time `orm:"null" form:"Disk_read_minimum_time" csv:"-"`
	Net_usage_maximum          float64   `orm:"null" form:"Net_usage_maximun" csv:"-"`
	Net_usage_maximum_time     time.Time `orm:"null" form:"Net_usage_maximum_time" csv:"-"`
	Net_usage_average          float64   `orm:"null" form:"Net_usage_average" csv:"-"`
	Net_usage_minimum          float64   `orm:"null" form:"Net_usage_minimum" csv:"-"`
	Net_usage_minimum_time     time.Time `orm:"null" form:"Net_usage_minimum_time" csv:"-"`
	CpuBilling                 float64   `form:"CpuBilling" csv:"-"`  // в мегагерц часах
	MemBilling                 float64   `form:"MemBilling" csv:"-"`  // в мегабайт часах
	DiskBilling                float64   `form:"DiskBilling" csv:"-"` // по занимаемому месту в гигабайт часах
	NetBilling                 float64   `form:"NetBilling" csv:"-"`  // в мегабайт часах
}

func (a *HostRecord) TableName() string {
	return "host_record"
}

func (u *HostRecord) TableIndex() [][]string {
	return [][]string{
		[]string{"ServerId"},
		[]string{"Name"},
		[]string{"Date"},
	}
}

type ClusterRecord struct {
	Id                                 int64     `form:"Id"`
	ClusterId                          int64     `form:"ClusterId"`
	Date                               time.Time `form:"Date"`
	Vcenter                            string    `form:"Vcenter"`
	Datacenter                         string    `form:"Datacenter"`
	Name                               string    `form:"Name"`
	CpuMhzAvg                          int32     `form:"CpuMhzAvg"`
	AllNumCpuPkgs                      int16     `form:"AllNumCpuPkgs"`
	AllNumCpuCores                     int16     `form:"AllNumCpuCores"`
	AllNumCpuThreads                   int16     `form:"AllNumCpuThreads"`
	AllMemorySizeGB                    float64   `form:"AllMemorySizeGB"`
	AllCapacityGb                      float64   `form:"AllCapacityGb"`
	AllFreeSpaceGb                     float64   `form:"AllFreeSpaceGb"`
	AllPercentUsed                     float64   `form:"AllPercentUsed"`
	Cpu_usage_maximum                  float64   `orm:"null" form:"Cpu_usage_maximum"`
	Cpu_usage_maximum_time             time.Time `orm:"null" form:"Cpu_usage_maximum_time"`
	Cpu_usage_average                  float64   `orm:"null" form:"Cpu_usage_average"`
	Cpu_usage_minimum                  float64   `orm:"null" form:"Cpu_usage_minimum"`
	Cpu_usage_minimum_time             time.Time `orm:"null" form:"Cpu_usage_minimum_time"`
	Mem_usage_maximum                  float64   `orm:"null" form:"Mem_usage_maximum"`
	Mem_usage_maximum_time             time.Time `orm:"null" form:"Mem_usage_maximum_time"`
	Mem_usage_average                  float64   `orm:"null" form:"Mem_usage_average"`
	Mem_usage_minimum                  float64   `orm:"null" form:"Mem_usage_minimum"`
	Mem_usage_minimum_time             time.Time `orm:"null" form:"Mem_usage_minimum_time"`
	Disk_usage_maximum                 float64   `orm:"null" form:"Disk_usage_maximum"`
	Disk_usage_maximum_time            time.Time `orm:"null" form:"Disk_usage_maximum_time"`
	Disk_usage_average                 float64   `orm:"null" form:"Disk_usage_average"`
	Disk_usage_minimum                 float64   `orm:"null" form:"Disk_usage_minimum"`
	Disk_usage_minimum_time            time.Time `orm:"null" form:"Disk_usage_minimum_time"`
	Disk_totalReadLatency_maximum      float64   `orm:"null" form:"Disk_totalReadLatency_maximum"`
	Disk_totalReadLatency_maximum_time time.Time `orm:"null" form:"Disk_totalReadLatency_maximum_time"`
	Disk_totalReadLatency_average      float64   `orm:"null" form:"Disk_totalReadLatency_average"`
	Disk_totalReadLatency_minimum      float64   `orm:"null" form:"Disk_totalReadLatency_minimum"`
	Disk_totalReadLatency_minimum_time time.Time `orm:"null" form:"Disk_totalReadLatency_minimum_time"`
	Disk_totalWriteLatency_maximum     float64   `orm:"null" form:"Disk_totalWriteLatency_maximum"`
	Disk_totalWriteLatency_maximum_time time.Time `orm:"null" form:"Disk_totalWriteLatency_maximum_time"`
	Disk_totalWriteLatency_average     float64   `orm:"null" form:"Disk_totalWriteLatency_average"`
	Disk_totalWriteLatency_minimum     float64   `orm:"null" form:"Disk_totalWriteLatency_minimum"`
	Disk_totalWriteLatency_minimum_time time.Time `orm:"null" form:"Disk_totalWriteLatency_minimum_time"`
	Disk_write_maximum                 float64   `orm:"null" form:"Disk_write_maximum"`
	Disk_write_maximum_time            time.Time `orm:"null" form:"Disk_write_maximum_time"`
	Disk_write_average                 float64   `orm:"null" form:"Disk_write_average"`
	Disk_write_minimum                 float64   `orm:"null" form:"Disk_write_minimun"`
	Disk_write_minimum_time            time.Time `orm:"null" form:"Disk_write_minimum_time"`
	Disk_read_maximum                  float64   `orm:"null" form:"Disk_read_maximun"`
	Disk_read_maximum_time             time.Time `orm:"null" form:"Disk_read_maximum_time"`
	Disk_read_average                  float64   `orm:"null" form:"Disk_read_average"`
	Disk_read_minimum                  float64   `orm:"null" form:"Disk_read_minimun"`
	Disk_read_minimum_time             time.Time `orm:"null" form:"Disk_read_minimum_time"`
	Net_usage_maximum                  float64   `orm:"null" form:"Net_usage_maximun"`
	Net_usage_maximum_time             time.Time `orm:"null" form:"Net_usage_maximun_time"`
	Net_usage_average                  float64   `orm:"null" form:"Net_usage_average"`
	Net_usage_minimum                  float64   `orm:"null" form:"Net_usage_minimun"`
	Net_usage_minimum_time             time.Time `orm:"null" form:"Net_usage_minimum_time"`
	CpuBilling                         float64   `form:"CpuBilling"`  // в мегагерц часах
	DiskBilling                        float64   `form:"DiskBilling"` // по занимаемому месту в гигабайт часах
	MemBilling                         float64   `form:"MemBilling"`  // в мегабайт часах
	NetBilling                         float64   `form:"NetBilling"`  // в мегабайт часах
	HostCount                          int64     `form:"HostCount"`
	VMCount                            int64     `form:"VMCount"`
	StorageCount                       int64     `form:"StorageCount"`
}

func (a *ClusterRecord) TableName() string {
	return "cluster_record"
}

func (u *ClusterRecord) TableIndex() [][]string {
	return [][]string{
		[]string{"ClusterId"},
		[]string{"Name"},
		[]string{"Date"},
		[]string{"Date", "Name"},
	}
}

type ClusterSprRecords struct {
	Id         int64     `form:"Id"`
	Vcenter    string    `form:"Vcenter"`
	Datacenter string    `form:"Datacenter"`
	Name       string    `form:"Name"`
	ClusterId  string    `form:"ClusterId"` //ClusterId = s-msk-vc-01.global.bcs:domain-c2638
	Annotation string    `form:"Annotation"`
	//Production int         `form:"Production"` //признак продуктового кластера 0-не заполнено 1-продуктовый с бизнес приложениями 2-тестовые среды
	TypeWork   int       `form:"TypeWork"` //0 - не указано 1 - продуктовая среда 2 - внутренний тест 3 - внешний тест
	Stat       int       `form:"Stat"`     //0- не указано 1 - активен 2 выведен из работы
	Id_User    int64     `form:"Id_User"`
	Time_Mfy   time.Time `form:"Time_Mfy"`
}

func (a *ClusterSprRecords) TableName() string {
	return "cluster_spr_records"
}

func (u *ClusterSprRecords) TableIndex() [][]string {
	return [][]string{
		[]string{"Name"},
	}
}

type DatacenterSprRecords struct {
	Id         int64     `form:"Id"`
	Vcenter    string    `form:"Vcenter"`
	Name       string    `form:"Name"`
	Annotation string    `form:"Annotation"`
	Id_User    int64     `form:"Id_User"`
	Time_Mfy   time.Time `form:"Time_Mfy"`
}

func (a *DatacenterSprRecords) TableName() string {
	return "datacenter_spr_records"
}

func (u *DatacenterSprRecords) TableIndex() [][]string {
	return [][]string{
		[]string{"Name"},
	}
}

type VcenterSprRecords struct {
	Id         int64     `form:"Id"`
	Name       string    `form:"Name"`
	IpAddress  string    `form:"IpAddress"`
	Annotation string    `form:"Annotation"`
	Id_User    int64     `form:"Id_User"`
	Time_Mfy   time.Time `form:"Time_Mfy"`
}

func (a *VcenterSprRecords) TableName() string {
	return "vcenter_spr_records"
}

func (u *VcenterSprRecords) TableIndex() [][]string {
	return [][]string{
		[]string{"Name"},
	}
}

type VMSprRecord struct {
	Id                   int64   `form:"Id"`
	ServerId             int64   `form:"ServerId"` //новое поле. добавить запись ссылки на справочник серверов
	Vcenter              string  `form:"Vcenter"`
	Datacenter           string  `form:"Datacenter"`
	Cluster              string  `form:"Cluster"`
	ClusterId            string  `form:"ClusterId"` //ClusterId = s-msk-vc-01.global.bcs:domain-c2638
	Production           int     `form:"Production"` //среда 0-не заполнено 1-продуктовая с бизнес приложениями 2-тестовая
	WorkStation          bool    `form:"WorkStation"` //признак рабочей станции windows
	TypeWork             int     `form:"TypeWork"`
	Vm_Name              string  `form:"Vm_Name"`
	Vm_HostName          string  `form:"Vm_HostName"`
	Vm_DomainName        string  `form:"Vm_DomainName"` //
	Vm_ToolsStatus       string  `form:"Vm_ToolsStatus"` // статус установки vmware tools
	Vm_GuestFullName     string  `form:"Vm_GuestFullName"` //
	Vm_GuestDetailedData string  `form:"Vm_GuestDetailedData"` //подробности по гостевой система
	Vm_PathName          string  `form:"Vm_PathName"`
	Vm_PowerState        bool    `form:"Vm_PowerState"` //true - VM включена
	KeyName              string  `form:"KeyName"` // формат Vcenter/Datacenter/Cluster/Vm_Name
	Vm_Uuid              string  `form:"Vm_Uuid"`
	Vm_GuestId           string  `form:"Vm_GuestId"`
	Vm_Annotation        string  `form:"Vm_Annotation" csv:"-"`
	Vm_NumCPU            int32   `form:"Vm_NumCPU"`
	Vm_NumCores          int32   `form:"Vm_NumCores"`
	Vm_MemorySizeGB      float64 `form:"Vm_MemorySizeGB"`
	Vm_Network           string  `form:"Vm_Network"` //список через разделитель ;
	Vm_IpAddress         string  `form:"Vm_IpAddress"` //список через разделитель ;
	Vm_MacAddress        string  `form:"Vm_MacAddress"` //список через разделитель ;
	Vm_NumEthernetCards  int32   `form:"Vm_NumEthernetCards"`
	Vm_NumVirtualDisks   int32   `form:"Vm_NumVirtualDisks"`
	Vm_DiskCommittedGB   float64 `form:"Vm_DiskCommittedGB"`
	Vm_DiskUncommittedGB float64 `form:"Vm_DiskUncommittedGB"`
	Vm_CapacityGb        float64 `form:"Vm_CapacityGb"`
	Vm_FreeSpaceGb       float64 `form:"Vm_FreeSpaceGb"`
	Vm_PercentUsed       float64 `form:"Vm_PercentUsed"`
	Cpu_usage_maximum    float64 `form:"Cpu_usage_maximum"`
	Mem_usage_maximum    float64 `form:"Mem_usage_maximum"`
	Disk_usage_maximum   float64 `form:"Disk_usage_maximum"`
	Net_usage_maximum    float64 `form:"Net_usage_maximum"`
}

func (a *VMSprRecord) TableName() string {
	return "vm_spr_record"
}

func (u *VMSprRecord) TableIndex() [][]string {
	return [][]string{
		[]string{"Vm_Uuid"},
		[]string{"Vcenter", "Vm_Uuid"},
		[]string{"KeyName"},
		[]string{"Vm_HostName"},
		[]string{"Vm_GuestFullName"},
		[]string{"Vcenter", "Datacenter", "Cluster", "Vm_HostName"},
	}
}

type VMGuestDiskInfo struct {
	Id          int64   `form:"Id"`
	VMId        int64   `form:"VMId"`
	DiskPath    string  `form:"DiskPath"`
	CapacityGb  float64 `form:"CapacityGb"`
	FreeSpaceGb float64 `form:"FreeSpaceGb"`
	PercentUsed float64 `form:"PercentUsed"`
}

func (a *VMGuestDiskInfo) TableName() string {
	return "vm_guest_disk_info"
}

func (u *VMGuestDiskInfo) TableIndex() [][]string {
	return [][]string{
		[]string{"VMId", "Id"},
	}
}

type ClusterPerf struct {
	Id                     int64     `form:"Id"`
	ClusterId              int64     `form:"ClusterId"` //ссылка на запись в ClusterRecord
	Start                  time.Time `form:"Start"`
	Cpu_usage              float64   `form:"Cpu_usage"`
	Mem_usage              float64   `form:"Mem_usage"`
	Disk_usage             float64   `form:"Disk_usage"`             //скорость диска kbps
	Disk_totalReadLatency  float64   `form:"Disk_totalReadLatency"`  //Среднее время чтения с виртуального диска
	Disk_totalWriteLatency float64   `form:"Disk_totalWriteLatency"` //Среднее время записи на виртуальный диск
	Disk_write             float64   `form:"Disk_write"`             //Скорость записи данных на виртуальный диск kbps
	Disk_read              float64   `form:"Disk_read"`              //Скорость чтения данных с виртуального диска kbps
	Net_usage              float64   `form:"Net_usage"`              //Использование сети (комбинированные скорости передачи и скорости приема) в течение интервала kbps
	Net_received           float64   `form:"Net_received"`           //Использование сети (скорость приема) в течение интервала kbps
	Net_transmitted        float64   `form:"Net_transmitted"`        //Использование сети (скорость передачи) в течение интервала kbps
}

func (u *ClusterPerf) TableIndex() [][]string {
	return [][]string{
		[]string{"ClusterId", "Id"},
		[]string{"Start"},
	}
}

func (a *ClusterPerf) TableName() string {
	return "cluster_perf"
}

type HostPerf struct {
	Id                     int64     `form:"Id"`
	HostId                 int64     `form:"HostId"` //ссылка на HostRecord
	Start                  time.Time `form:"Start"`
	Cpu_usage              float64   `form:"Cpu_usage"`
	Mem_usage              float64   `form:"Mem_usage"`
	Disk_usage             float64   `form:"Disk_usage"`             //скорость диска kbps
	Disk_totalReadLatency  float64   `form:"Disk_totalReadLatency"`  //Среднее время чтения с виртуального диска
	Disk_totalWriteLatency float64   `form:"Disk_totalWriteLatency"` //Среднее время записи на виртуальный диск
	Disk_write             float64   `form:"Disk_write"`             //Скорость записи данных на виртуальный диск kbps
	Disk_read              float64   `form:"Disk_read"`              //Скорость чтения данных с виртуального диска kbps
	Net_usage              float64   `form:"Net_usage"`              //Использование сети (комбинированные скорости передачи и скорости приема) в течение интервала kbps
	Net_received           float64   `form:"Net_received"`           //Использование сети (скорость приема) в течение интервала kbps
	Net_transmitted        float64   `form:"Net_transmitted"`        //Использование сети (скорость передачи) в течение интервала kbps
}

func (u *HostPerf) TableIndex() [][]string {
	return [][]string{
		[]string{"HostId", "Id"},
		[]string{"Start"},
	}
}

func (a *HostPerf) TableName() string {
	return "host_perf"
}

type VMPerfRecord struct {
	Id                     int64     `form:"Id"`
	VMId                   int64     `form:"VMId"`
	Start                  time.Time `form:"Start"`
	Cpu_usage              float64   `form:"Cpu_usage"`
	Mem_usage              float64   `form:"Mem_usage"`
	Disk_usage             float64   `form:"Disk_usage"`             //скорость диска kbps
	Disk_totalReadLatency  float64   `form:"Disk_totalReadLatency"`  //Среднее время чтения с виртуального диска
	Disk_totalWriteLatency float64   `form:"Disk_totalWriteLatency"` //Среднее время записи на виртуальный диск
	Disk_write             float64   `form:"Disk_write"`             //Скорость записи данных на виртуальный диск kbps
	Disk_read              float64   `form:"Disk_read"`              //Скорость чтения данных с виртуального диска kbps
	Net_usage              float64   `form:"Net_usage"`              //Использование сети (комбинированные скорости передачи и скорости приема) в течение интервала kbps
	Net_received           float64   `form:"Net_received"`           //Использование сети (скорость приема) в течение интервала kbps
	Net_transmitted        float64   `form:"Net_transmitted"`        //Использование сети (скорость передачи) в течение интервала kbps
}

func (u *VMPerfRecord) TableIndex() [][]string {
	return [][]string{
		[]string{"VMId", "Id"},
		[]string{"Start"},
	}
}

type ServerSprRecords struct {
	Id               int64     `form:"Id" json:"Id"`
	OldId            int64     `form:"OldId" json:"OldId"` //ссылка на историческую запись после переименования. 0 если такой нет. Может быть цепочка записей.
	Type             int       `form:"Type" json:"Type"` //0 - физический сервер 1 - виртуальный 2-host vmware
	SerialNumber     string    `form:"SerialNumber" json:"SerialNumber"` //серийный номер для type 0 и 2
	Vcenter          string    `form:"Vcenter" json:"Vcenter"` //для type 1 и 2
	Datacenter       string    `form:"Datacenter" json:"Datacenter"` //для type 1 и 2
	Cluster          string    `form:"Cluster" json:"Cluster"` //для type 1 и 2
	ClusterId        string    `form:"ClusterId" json:"ClusterId"` //ClusterId = s-msk-vc-01.global.bcs:domain-c2638
	VMSprId          int64     `form:"VMSprId" json:"VMSprId"` //ссылка на справочник VM если Type = 1 иначе null
	HWSprId          int64     `form:"HWSprId" json:"HWSprId"` //ссылка на ServerHardware
	ZabbixHostId     int64     `form:"ZabbixHostId" json:"ZabbixHostId"` //ссылка на ZabbixHost.Id если есть данные из Zabbix по этому серверу
	TypeMicroservice int       `form:"TypeMicroservice" json:"TypeMicroservice"` // 0 не указано принадлежность к микросервисам, роль в этом пространстве
	RoleType         int       `form:"RoleType" json:"RoleType"` //не указано 1-APP 2-DB
	TypeWork         int       `form:"TypeWork" json:"TypeWork"` //0 - не указано 1 - продуктовая среда 2- внутренний тест 3 - внешний тест
	CriticalLevel    int       `form:"CriticalLevel" json:"CriticalLevel"` //контур 0 не указано 1-Vital 2-Critical 3-Secondary 4-Non Priority
	Stat             int       `form:"Stat" json:"Stat"` //0 историческая запись 1 используется 2 помечено для удаления 3-архивная запись после переименования
	PowerState       int       `form:"PowerState" json:"PowerState"` //0-не указано 1 включен 2 выключен (последнее состояние) при отображении переопределять текущим состоянием // TODO: comment truncated in screenshot
	Active           int       `form:"Active" json:"Active"` //1 активен 2 выведен из эксплуатации
	TimeActiveState  time.Time `orm:"null" form:"TimeActiveState" json:"TimeActiveState"` //время изменения статуса активности введения в эксплуатацию
	ActivatorUser    int64     `orm:"null" form:"ActivatorUser" json:"ActivatorUser"` //автор записи
	TimeDeActiveState time.Time `orm:"null" form:"TimeDeActiveState" json:"TimeDeActiveState"` //время выведения из эксплуатации
	DeActivatorUser  int64     `orm:"null" form:"DeActivatorUser" json:"DeActivatorUser"` //автор записи
	TimeCreate       time.Time `orm:"null" form:"TimeCreate" json:"TimeCreate"` //время создания записи
	CreatorUser      int64     `orm:"null" form:"CreatorUser" json:"CreatorUser"` //автор записи
	BusinesPrivilage int64     `orm:"null" form:"BusinesPrivilage" json:"BusinesPrivilage"` //ограничение по редактированию если указано 0 объект открыт // TODO: comment truncated in screenshot
	Name             string    `form:"Name,text,Name:" valid:"MinSize(0);MaxSize(256)" json:"Name"` //
	NameFQDN         string    `form:"NameFQDN" json:"NameFQDN"` //полное имя сервера
	NameInfo         string    `orm:"null" form:"NameInfo" json:"NameInfo"` //альтернативное имя сервера
	IPAddress        string    `orm:"null" form:"IPAddress" json:"IPAddress"` //список IP адресов сервера при наличии информации подгружается из конфигурации VM или вводится вручную // TODO: comment truncated in screenshot
	MacAddress       string    `form:"MacAddress"` //список через разделитель ;
	IPSharedVIP      string    `orm:"null" form:"IPSharedVIP" json:"IPSharedVIP"` //удалить разделяемый IP вводится вручную
	Network          string    `orm:"null" form:"Network" json:"Network"` //сеть vlan144
	NetworkName      string    `orm:"null" form:"NetworkName" json:"NetworkName"` //наименование сети сеть DMZ Prom
	CPUCount         float64   `orm:"null" form:"CPUCount" json:"CPUCount"` //Количество физических процессоров
	CPULogical       float64   `form:"CPULogical" json:"CPULogical"` //количество логических CPU
	MemSize          float64   `form:"MemSize" json:"MemSize"` //размер оперативной памяти сервера
	DiskSize         float64   `form:"DiskSize" json:"DiskSize"` //суммарный размер всех дисков
	Service          string    `form:"Service" json:"Service"` //удалить сервис в котором задействован сервер
	ServiceId        int64     `form:"ServiceId" json:"ServiceId"` //удалить ссылка на базу сервисов
	ITSystemKey      string    `form:"ITSystemKey" json:"ITSystemKey"` //ссылка на ИТ систему из jira https://jira.bcs.ru/rest/assets/latest/aql/objects?qlQuery=objectTypeId // TODO: URL truncated in screenshot
	OS               string    `form:"OS" json:"OS"` //операционная система
	Component        string    `form:"Component" json:"Component"` //удалить компонента Kafka, service BD ....
	Owner            string    `form:"Owner" json:"Owner" csv:"-"` //владелец сервера (ответственный за сервер)
	BusinessOwner    string    `form:"BusinessOwner" json:"BusinessOwner" csv:"-"` //владелец от бизнеса
	ITOwner          string    `form:"ITOwner" json:"ITOwner" csv:"-"` //владелец со стороны ИТ
	ProductOwner     string    `form:"ProductOwner" json:"ProductOwner" csv:"-"` //Владелец продукта
	Support          string    `form:"Support" json:"Support" csv:"-"` //ответственные за поддержку
	SupportTeam      string    `form:"SupportTeam" json:"SupportTeam" csv:"-"` //Подразделение поддержки
	SystemType       string    `form:"SystemType" json:"SystemType"` //тип системы техническая ...
	LicensingType    string    `form:"LicensingType" json:"LicensingType"` //тип лицензирования
	SolutionType     string    `form:"SolutionType" json:"SolutionType"` //вариант решения (коробочной)
	Users            string    `form:"Users" json:"Users" csv:"-"` //пользователи системы
	Allocation       string    `form:"Allocation" json:"Allocation"` //аллокация (подразделение)
	Anotation          string    `form:"Anotation" json:"Anotation" csv:"-"` //описание сервера
	TagName          string    `form:"TagName" json:"TagName"` //Имя тэга для виртуальных машин (Код для справочника)
	TagDescription   string    `form:"TagDescription" json:"TagDescription"` //Описание тэга для виртуальных машин
	Role             string    `form:"Role" json:"Role"` //роль сервера (сервер баз данных, сервер приложений, хост)
	Details          string    `form:"Details" json:"Details" csv:"-"` //детали по серверу в свободном формате (примечание)
	DPCAddress       int       `form:"DPCAddress" json:"DPCAddress"` //удалить ЦОД по справочнику адрес
	DPCOffice        int       `form:"DPCOffice" json:"DPCOffice"` //ЦОД по справочнику комната с адресом
	SZHistory        string    `form:"SZHistory" json:"SZHistory"` //номера сз через пробел в хронологическом порядке
	Warning          string    `form:"Warning" json:"Warning"` //предупреждение (CPU RAM)
	Level            float64   `form:"Level" json:"Level"` //уровень в процентах
	Monitoring       int       `form:"Monitoring" json:"Monitoring"` // 0- не указано 1-включен мониторинг 2-отключен мониторинг
	Backuped         int       `form:"Backuped" json:"Backuped"` // 0 - не указано 1-включено резервное копирование 2-отключено резервное копирование
	TimeActiveMonitoring time.Time `orm:"null" form:"TimeActiveMonitoring" json:"TimeActiveMonitoring"` //время постановки на мониторинг
	MonitoringSZ     string    `form:"MonitoringSZ" json:"MonitoringSZ"` //основание постановки на мониторинг
	Id_User          int64     `form:"Id_User" json:"Id_User"` //
	Time_Mfy         time.Time `form:"Time_Mfy" json:"Time_Mfy"` //
	Privilage        int       `form:"Privilage" json:"Privilage"` //уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привелегии пользователя) // TODO: comment truncated in screenshot
}

type ServerSprRecordsArch struct {
	ServerSprRecords
}

// структура с метриками из SCOM, усреднение за 1 день
type ServerPerfSCOM struct {
	Id                     int64     `form:"Id" csv:"-"` //уникальный идентификатор записи
	Date                   time.Time `form:"Date" csv:"-"` //дата сбора метрик
	NameFQDN               string    `form:"NameFQDN" csv:"-"` //полное имя сервера как пришло из SCOM
	ServerId               int64     `form:"ServerId" csv:"-"` //ссылка на справочник серверов
	Type                   int       `form:"Type" csv:"-"` //0 - физический сервер 1 - виртуальный как пришло из SCOM
	CPUCount               float64   `form:"CPUCount"` //Количество физических процессоров или vCPU
	CPULogical             float64   `form:"CPULogical"` //количество логических CPU включая hyper threading
	FoundAddInfo           int       `form:"FoundAddInfo"` //0 - не найдено 1 есть данные (для аппаратных серверов найдены данные конфигурации, для виртуальной инфраструктуры найден объект)
	MemSize                float64   `form:"MemSize"` //
	SysDiskSize            float64   `form:"SysDiskSize"` //
	WorkDiskSize           float64   `form:"WorkDiskSize"` //
	CPULoadAvg             float64   `form:"CPULoadAvg"` //
	CPULoadMin             float64   `form:"CPULoadMin"` //
	CPULoadMax             float64   `form:"CPULoadMax"` //
	MemUseAvg              float64   `form:"MemUseAvg"` //
	MemUseMin              float64   `form:"MemUseMin"` //
	MemUseMax              float64   `form:"MemUseMax"` //
	SysDiskFreePercentAvg  float64   `form:"SysDiskFreePercentAvg"` //
	SysDiskFreePercentMin  float64   `form:"SysDiskFreePercentMin"` //
	SysDiskFreePercentMax  float64   `form:"SysDiskFreePercentMax"` //
	SysDiskIdleTimeAvg     float64   `form:"SysDiskIdleTimeAvg"` //
	SysDiskIdleTimeMin     float64   `form:"SysDiskIdleTimeMin"` //
	SysDiskIdleTimeMax     float64   `form:"SysDiskIdleTimeMax"` //
	SysDiskQueueAvg        float64   `form:"SysDiskQueueAvg"` //
	SysDiskQueueMin        float64   `form:"SysDiskQueueMin"` //
	SysDiskQueueMax        float64   `form:"SysDiskQueueMax"` //
	WorkDiskFreePercentAvg float64   `form:"WorkDiskFreePercentAvg"` //
	WorkDiskFreePercentMin float64   `form:"WorkDiskFreePercentMin"` //
	WorkDiskFreePercentMax float64   `form:"WorkDiskFreePercentMax"` //
	WorkDiskIdleTimeAvg    float64   `form:"WorkDiskIdleTimeAvg"` //
	WorkDiskIdleTimeMin    float64   `form:"WorkDiskIdleTimeMin"` //
	WorkDiskIdleTimeMax    float64   `form:"WorkDiskIdleTimeMax"` //
	WorkDiskQueueAvg       float64   `form:"WorkDiskQueueAvg"` //
	WorkDiskQueueMin       float64   `form:"WorkDiskQueueMin"` //
	WorkDiskQueueMax       float64   `form:"WorkDiskQueueMax"` //
	SysDiskIOAvg           float64   `form:"SysDiskIOAvg"` //
	SysDiskIOMin           float64   `form:"SysDiskIOMin"` //
	SysDiskIOMax           float64   `form:"SysDiskIOMax"` //
	SysDiskFreeMbAvg       float64   `form:"SysDiskFreeMbAvg"` //
	SysDiskFreeMbMin       float64   `form:"SysDiskFreeMbMin"` //
	SysDiskFreeMbMax       float64   `form:"SysDiskFreeMbMax"` //
	WorkDiskIOAvg          float64   `form:"WorkDiskIOAvg"` //
	WorkDiskIOMin          float64   `form:"WorkDiskIOMin"` //
	WorkDiskIOMax          float64   `form:"WorkDiskIOMax"` //
	WorkDiskFreeMbAvg      float64   `form:"WorkDiskFreeMbAvg"` //
	WorkDiskFreeMbMin      float64   `form:"WorkDiskFreeMbMin"` //
	WorkDiskFreeMbMax      float64   `form:"WorkDiskFreeMbMax"` //
	UseWorkDisk            int       `form:"UseWorkDisk" csv:"-"` //
}

func (a *ServerPerfSCOM) TableName() string {
	return "server_perf_scom"
}

func (a *ServerPerfSCOM) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Date"},
		[]string{"ServerId"},
		[]string{"ServerId", "Date"},
		[]string{"NameFQDN", "Date"},
	}
}

// структура с метриками из Zabbix, усреднение за 1 день
type ServerPerfZabbix struct {
	Id                int64     `form:"Id" csv:"-"` //уникальный идентификатор записи
	Date              time.Time `form:"Date" csv:"-"` //дата сбора метрик
	NameFQDN          string    `form:"NameFQDN" csv:"-"` //полное имя сервера
	NameAdd           string    `form:"NameAdd" csv:"-"` //дополнение к имени после пробела как пришло из Zabbix
	ServerId          int64     `form:"ServerId" csv:"-"` //ссылка на справочник серверов
	Type              int       `form:"Type" csv:"-"` //0 - физический сервер 1 - виртуальный как пришло из Zabbix (наличие VC в дополнительной части имени признак виртуальной машины)
	CPUCount          float64   `form:"CPUCount" csv:"-"` //Количество физических процессоров или vCPU
	CPULogical        float64   `form:"CPULogical" csv:"-"` //Количество логических CPU включая hyper threading
	FoundAddInfo      int       `form:"FoundAddInfo" csv:"-"` //0 - не найдено 1 есть данные (для аппаратных серверов найдены данные конфигурации, для виртуальной инфраструктуры найден объект)
	MemSize           float64   `form:"MemSize"` //
	Cpu_usage_average float64   `orm:"null" form:"Cpu_usage_average"` //% Processor Time averagre
	Cpu_usage_maximum float64   `orm:"null" form:"Cpu_usage_maximum"` //% Processor Time max
	Cpu_usage_minimum float64   `orm:"null" form:"Cpu_usage_minimum"` //% Processor Time min
	Mem_usage_max     float64   `orm:"null" form:"Mem_usage_max"` //% Mem Usage max
	Mem_usage_average float64   `orm:"null" form:"Mem_usage_average"` //% Mem Usage average
	Mem_usage_minimum float64   `orm:"null" form:"Mem_usage_minimum"` //% Mem Usage min
}

// структура с метриками дисков из Zabbix, усреднение за 1 день
type ServerDiskPerfZabbix struct {
	Id                 int64   `form:"Id"` //уникальный идентификатор записи
	ZabbixPerfId       int64   `form:"ZabbixPerfId"` //идентификатор связанной записи ServerPerfZabbix
	Disk_name          string  `form:"Disk_name"` //имя диска windows или раздела unix
	Disk_size          float64 `form:"Disk_size"` //размер диска
	Disk_usage_maximum float64 `orm:"null" form:"Disk_usage_maximum"` //% Disk Usage max по занятости диска
	Disk_usage_average float64 `orm:"null" form:"Disk_usage_average"` //% Disk Usage average
	Disk_usage_minimum float64 `orm:"null" form:"Disk_usage_minimum"` //% Disk Usage min
}

type ServerPerfZabbixView struct {
	Base  ServerPerfZabbix
	Disks []*ServerDiskPerfZabbix `form:"Disks"` //диски
}

func (a *ServerPerfZabbix) TableName() string {
	return "server_perf_zabbix"
}

func (a *ServerPerfZabbix) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Date"},
		[]string{"ServerId"},
		[]string{"ServerId", "Date"},
		[]string{"NameFQDN", "Date"},
	}
}

func (a *ServerDiskPerfZabbix) TableName() string {
	return "server_disk_perf_zabbix"
}

func (a *ServerDiskPerfZabbix) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ZabbixPerfId"},
		[]string{"ZabbixPerfId", "Disk_name"},
	}
}

//------------------------------------------------------------------------------------------------------------------
type ZabbixHost struct {
	Id                int64     `form:"Id" orm:"auto"` //
	ServerId          int64     `form:"ServerId"` //ссылка на ServerSprRecords.Id
	Host              string    `form:"Host"` //Имя хоста 1
	Name              string    `form:"Name"` //Имя хоста 2
	NameFQDN          string    `form:"NameFQDN"` //Полное Имя хоста
	OperationalSystem string    `form:"OperationalSystem"` //Операционная система в своем формате
	OSArchitecture    string    `form:"OSArchitecture"` //архитектура операционной системы
	LogicalNumCPU     int       `form:"LogicalNumCPU"` //количество логических процессоров включая Intel Hyper-Threading
	MemoryTotal       float64   `form:"MemoryTotal"` //Общий объём оперативной памяти в байтах
	IpAddress         string    `form:"IpAddress"` //IP адреса всех интерфейсов через ;
	TimeMfy           time.Time `form:"TimeMfy"` //время последнего обновления
}

func (a *ZabbixHost) TableName() string {
	return "zabbix_host"
}

func (a *ZabbixHost) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ServerId"},
		[]string{"NameFQDN"},
	}
}

type ZabbixItem struct {
	Id       int64     `form:"Id" orm:"auto"` //
	HostId   int64     `form:"HostId"` //ссылка на ZabbixHost.Id
	KeyId    int64     `form:"KeyId"` //ссылка на ZabbixKey.Id
	KeyParam string    `form:"KeyParam"` //параметр из полного ключа значение между квадратных скобок []
	//KeyName  string    `form:"KeyName"`      //выбрать часть имени элемента
	Value    string    `form:"Value"` //значение в текстовом виде
	Clock    time.Time `form:"Clock"` //время значения
}

func (a *ZabbixItem) TableName() string {
	return "zabbix_item"
}

func (a *ZabbixItem) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"HostId"},
		[]string{"KeyId"},
		[]string{"HostId", "KeyId"},
	}
}

/*
Значения value_type
0 - Numeric float Число с плавающей точкой Пример: 0.15, 23.7
1 - Character Короткая строка Пример: "up", "OK"
2 - Log Лог-запись Используется для элементов, которые читают логи
3 - Numeric unsigned Целое без знака Пример: 0, 15, 1024
4 - Text Длинный текст Пример: большой JSON, текст ошибки, дамп строки
5 - Binary Бинарные данные

Значения поля Units
B - байты
Bps - байт/с
bps - бит/с
% - проценты
s - секунды
ms - миллисекунды
rpm - обороты в минуту
uptime - специальный формат времени работы
unixtime - Unix time
°C - градусы Цельсия
*/

type ZabbixKey struct {
	Id        int64  `form:"Id" orm:"auto"` //
	Key       string `form:"Key"` //
	Name      string `form:"Name"` //
	ValueType int    `form:"ValueType"` //0 - Numeric float, 1 - Character (короткий текст), 2 - Log, 3 - Numeric unsigned, 4 - Text (длинный текст),5 - Binary
	Units     string `form:"Units"` //единицы измерения
}

func (a *ZabbixKey) TableName() string {
	return "zabbix_key"
}

func (a *ZabbixKey) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Key"},
		[]string{"Name"},
	}
}

//---------------------------------------------------------

// прикладные сервисы, в которых задействованы серверы
type AppService struct {
	Id            int64  `form:"Id" orm:"auto"` //
	ServiceId     int64  `form:"ServiceId"` //идентификатор сервиса
	Code          string `form:"Code"` //1 код буквы и номер
	Type          int64  `form:"Type"` //тип сервиса (подумать над вариантами) 0-не указано 1-инфраструктурный 2 - бизнес
	Type2         int64  `form:"Type2"` //2 тип индивидуальная, индивидуальная / корпоративная, корпоративная, общего назначения
	Name          string `form:"Name"` //название сервиса
	Owner         string `form:"Owner"` //владелец сервиса ФИО (сотрудник отвественный за сервис) администратор информационны системы
	BusinesOwner  string `form:"BusinesOwner"` //заказчик от бизнеса
	Allocation    string `form:"Allocation"` //аллокация (подразделение) базовое значение (потребители)
	Anotation     string `form:"Anotation"` //описане сервиса
	Privilage     int    `form:"Privilage"` //уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привелегии пользователя)
	CriticalLevel int    `form:"CriticalLevel"` //контур 0 не указано 1-Vital 2-Critical 3-Secondary 4-Non Priority
}

func (a *AppService) TableName() string {
	return "app_service"
}

func (a *AppService) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ServiceId"},
		[]string{"Name"},
	}
}

// Учёт ресурсов администрирования сервиса (в разработке)
type ServiceAdminAllocation struct {
	Id        int64   `form:"Id" orm:"auto"` //
	ServiceId int64   `form:"ServiceId"` //идентификатор сервиса
	AdminId   int64   `form:"AdminId"` //идентификатор администратора системы
	Percent   float64 `form:"Percent"` //процент участия в обслуживании сервиса (доля в в конкретном сервисе)
	RoleType  int64   `form:"RoleType"` //категория роли в поддержке 0 не указано 1-аппаратная часть 2-системное ПО 3-прикладная
	Role      string  `form:"Role"` // зона ответственности в поддержке сервиса
}

// Юридические лица в рамках которых работают сервисы
type ServiceMember struct {
	Id         int64  `form:"Id" orm:"auto"` //
	ServiceId  int64  `form:"ServiceId"` //ссылка на справочник сервисов (может быть несколько записей с одним идентификатором сервиса)
	MemberFGId int64  `form:"MemberFGId"` //ссылка на справочник ЮЛ ФГ MembersFG
	Name       string `form:"Name"` //наименование ЮР лица
	Privilage  int    `form:"Privilage"` //уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привелегии пользователя)
}

func (a *ServiceMember) TableName() string {
	return "service_member"
}

func (a *ServiceMember) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ServiceId"},
		[]string{"MemberFGId"},
	}
}

type MembersFG struct {
	Id        int64  `form:"Id" orm:"auto"` //
	Name      string `form:"Name"` //Наименование
	InnUL     string `form:"InnUL"` // ИНН
	KppUL     string `form:"KppUL"` // КПП
	Privilage int    `form:"Privilage"` //уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привилегии пользователя)
}

func (a *MembersFG) TableName() string {
	return "members_fg"
}

func (u *MembersFG) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"InnUL"},
		[]string{"Name"},
	}
}

type Counterparty struct {
	Id        int64  `form:"Id" orm:"auto"` //
	Name      string `form:"Name"` //Наименование
	InnUL     string `form:"InnUL"` // ИНН
	KppUL     string `form:"KppUL"` // КПП
	Privilage int    `form:"Privilage"` //уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привилегии пользователя)
}

func (a *Counterparty) TableName() string {
	return "counterparty"
}

func (u *Counterparty) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"InnUL"},
		[]string{"Name"},
	}
}

type ServerSprView struct {
	PercentFilledField float64                `form:"PercentFilledField"` //процент заполнения полей в справочнике
	S                  ServerSprRecords       `form:"S"` //общая информация
	Hardware           *ServerHardwareView    `form:"Hardware"` //подробности для аппаратных серверов
	VM                 *VMSprRecordView       `form:"VM"` //подробная информация по VM
	VMPerf             *VMRecord              `form:"VMPerf"` //усредненные метрики производительности VM за период
	HostPerf           *HostRecord            `form:"HostPerf"` //усредненные метрики производительности Host за период
	SrvScomPerf        *ServerPerfSCOM        `form:"SrvScomPerf"` ////усредненные метрики производительности Srv за период из SCOM
	SrvZabbixPerf      *ServerPerfZabbixView  `form:"SrvZabbixPerf"` ////усредненные метрики производительности Srv за период из Zabbix + диски
	SrvPerf            *ReportServerRecord    `form:"SrvPerf"` //усредненные метрики производительности Srv за период
	CFO                []SprCFOAllocation     `form:"CFO"` //алокация по ЦФО
	Busines            []SprBusinesAllocation `form:"Busines"` //алокация по бизнесам
}

func (a *ServerSprRecords) TableName() string {
	return "server_spr_records"
}

func (u *ServerSprRecords) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"VMSprId"},
		[]string{"NameFQDN"},
		[]string{"Name"},
		[]string{"NameInfo"},
		[]string{"Active", "Name"},
		[]string{"SZHistory"},
		[]string{"Allocation"},
		[]string{"Service"},
		[]string{"ITSystemKey"},
		[]string{"Details"},
		[]string{"Anotation"},
		[]string{"Owner"},
	}
}

func (a *ServerSprRecordsArch) TableName() string {
	return "server_spr_records_arch"
}

func (u *ServerSprRecordsArch) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"VMSprId"},
		[]string{"NameFQDN"},
		[]string{"Name"},
		[]string{"NameInfo"},
		[]string{"Active", "Name"},
		[]string{"SZHistory"},
		[]string{"Allocation"},
		[]string{"Service"},
		[]string{"ITSystemKey"},
		[]string{"Details"},
		[]string{"Anotation"},
		[]string{"Owner"},
	}
}

type BladeSystems struct {
	Id              int64     `form:"Id"`
	Name            string    `form:"Name"` //короткое наименование
	Vendor          string    `form:"Vendor"` //производитель
	Model           string    `form:"Model"` //модель
	Type            string    `form:"Type"` //тип c7000 ...
	PartNum         string    `form:"PartNum"` //продуктовый номер
	Size            int       `form:"Size"` //количество слотов
	Used            int       `form:"Used"` //количество занятых слотов
	SerialNum       string    `form:"SerialNum"` //серийный номер
	Owner           string    `form:"Owner"` //владелец
	Allocation      string    `form:"Allocation"` //аллокация
	Anotation       string    `form:"Anotation"` //описане
	Description     string    `form:"Description"` //
	DatacenterRoom  string    `form:"DatacenterRoom"` //серверная
	RackNum         string    `form:"RackNum"` //номер стойки
	StartUnit       string    `form:"StartUnit"` //унит с которого начинается сервер или блэйд шасси, куда он установлен //!!! изменен тип
	StartUnitPos    int       `form:"StartUnitPos"` //
	SizeUnit        int       `form:"SizeUnit"` //
	ContractDoc     string    `form:"ContractDoc"` //договорной документ на покупку, название и номер дополнительного соглашения
	DateContractDoc time.Time `orm:"null" form:"DateContractDoc"` //дата документа
	Invoice         string    `form:"Invoice"` //счет фактура, накладная
	DateInvoice     time.Time `orm:"null" form:"DateInvoice"` //дата накладной
	InventoryNum    string    `form:"InventoryNum"` //Инвентарный номер
	InventoryNumBuh string    `form:"InventoryNumBuh"` //Инвентарный номер бухгалтерский
	Counterparty    string    `orm:"null" form:"Counterparty"` //контрагент
	DateService     time.Time `orm:"null" form:"DateService"` //дата до которой есть гарантия или сервис контракт
	ServiceContract string    `form:"ServiceContract"` //данные по сервисному контракту, уровень сервиса
	Id_User         int64     `form:"Id_User"` //
	Time_Mfy        time.Time `form:"Time_Mfy"` //
}

func (a *BladeSystems) TableName() string {
	return "blade_systems"
}

func (u *BladeSystems) TableIndex() [][]string {
	return [][]string{
		[]string{"Name"},
		[]string{"SerialNum"},
		[]string{"InventoryNum"},
	}
}

type SANSystem struct {
	Id              int64     `form:"Id"`
	ShortName       string    `form:"ShortName"` //мнемоническое имя для систем
	Vendor          string    `form:"Vendor"` //производитель
	Model           string    `form:"Model"` //модель
	PartNum         string    `form:"PartNum"` //код продукта у вендора
	SerialNum       string    `form:"SerialNum"` //серийный номер
	IpAddress       string    `form:"IpAddress"` //IP apec
	Fabric          string    `form:"Fabric"` //фабрика
	Firmware        string    `form:"Firmware"` //версия прошивки
	Owner           string    `form:"Owner"` //владелец
	Allocation      string    `form:"Allocation"` //аллокация
	Anotation       string    `form:"Anotation"` //Описание
	DatacenterRoom  string    `form:"DatacenterRoom"` //серверная
	RackNum         string    `form:"RackNum"` //номер стойки, если их несколько можно указать через запятую
	StartUnit       string    `form:"StartUnit"` //юнит с которого начинается СХД, куда она установлена, в случае нескольких стоек писать через запятую
	BladeSystemId   int64     `form:"BladeSystemId"` //ссылка блэйд-шасси в справоченке
	BladeNum        int       `form:"BladeNum"` //номер позиции в блэйд шасси
	ContractDoc     string    `form:"ContractDoc"` //договорной документ на покупку, название и номер дополнительного соглашения
	DateContractDoc time.Time `orm:"null" form:"DateContractDoc"` //дата документа
	Invoice         string    `form:"Invoice"` //счет фактура, накладная
	DateInvoice     time.Time `orm:"null" form:"DateInvoice"` //дата накладной
	Counterparty    string    `orm:"null" form:"Counterparty"` //контрагент
	InventoryNum    string    `form:"InventoryNum"` //Инвентарный номер
	InventoryNumBuh string    `form:"InventoryNumBuh"` //Инвентарный номер бухгалтерский
	DateService     time.Time `orm:"null" form:"DateService"` //дата до которой есть гарантия или сервис контракт
	ServiceContract string    `form:"ServiceContract"` //данные по сервисному контракту, уровень сервиса
	Id_User         int64     `form:"Id_User"` //
	Time_Mfy        time.Time `form:"Time_Mfy"` //
}

func (a *SANSystem) TableName() string {
	return "san_systems"
}

func (u *SANSystem) TableIndex() [][]string {
	return [][]string{
		[]string{"BladeSystemId"},
		[]string{"ShortName"},
		[]string{"InventoryNum"},
	}
}

type StorageSystems struct {
	Id              int64     `form:"Id"`
	UID             string    `form:"UID"` //уникальный идентификатор
	ShortName       string    `form:"ShortName"` //мнемоническое имя для систем
	TagName         string    `form:"TagName"` //список мнемонических имён встречающихся для этой СХД в именах LUN
	TypeWork        int       `form:"TypeWork" json:"TypeWork"` //0 - не указано 1 - продуктовая среда 2- тестовая 3 - бэкап
	//0 - не указано 1 - продуктовая среда 2- внутренний тест 3 - внешний тест
	Vendor          string    `form:"Vendor"` //производитель
	Model           string    `form:"Model"` //модель
	PartNum         string    `form:"PartNum"` //код продукта у вендора
	Status          int       `form:"Status"` //0-не указано 1-используется, 2-выключен, 3- не используется, 4 - тестирование, 5-приемка, 6- демонтирован,
	Size            int       `form:"Size"` //максимальное количество дисков
	Used            int       `form:"Used"` //общее количество установленных дисков
	IpAddress       string    `form:"IpAddress"` //IP apec
	IpAddressNext   string    `form:"IpAddressNext"` //IP apec 2
	TotalSizeRAW    float64   `form:"TotalSizeRAW"` //размер дисков СХД (сырой)
	UsedPercent     float64   `form:"UsedPercent"` //использовано в %
	WWNMask         string    `form:"WWNMask"` //WWN маска дисков
	Firmware        string    `form:"Firmware"` //версия прошивки
	SerialNum       string    `form:"SerialNum"` //серийный номер
	Power           int       `form:"Power"` //0 выключен 1 включен
	ApiType         int       `form:"ApiType"` //тип API 0 не указано 1 - VNX CLI, 2 -Unity Rest 3 - Dorado rest  4- OceanStor 5- Lenovo Rest
	Ad              bool      `form:"Ad"` //в AD
	Ldaps           bool      `form:"Ldaps"` //Ldaps
	AdCert          bool      `form:"AdCert"` //сертификат AD
	AdCertDate      time.Time `orm:"null" form:"AdCertDate"` //Дата истечения сертификата
	Owner           string    `form:"Owner"` //владелец
	Allocation      string    `form:"Allocation"` //аллокация
	Anotation       string    `form:"Anotation"` //Описание
	City            string    `form:"City"` //Город
	DatacenterRoom  string    `form:"DatacenterRoom"` //серверная
	RackNum         string    `form:"RackNum"` //номер стойки, если их несколько можно указать через запятую
	StartUnit       string    `form:"StartUnit"` //юнит с которого начинается СХД, куда она установлена, в случае нескольких стоек писать через запятую
	SizeUnit        string    `form:"SizeUnit"` //размер в юнитах
	ContractDoc     string    `form:"ContractDoc"` //договорной документ на покупку, название и номер дополнительного соглашения
	DateContractDoc time.Time `orm:"null" form:"DateContractDoc"` //дата документа
	Invoice         string    `form:"Invoice"` //счет фактура, накладная
	DateInvoice     time.Time `orm:"null" form:"DateInvoice"` //дата накладной
	Counterparty    string    `orm:"null" form:"Counterparty"` //контрагент
	InventoryNum    string    `form:"InventoryNum"` //Инвентарный номер
	InventoryNumBuh string    `form:"InventoryNumBuh"` //Инвентарный номер Бухгалтерский
	DateService     time.Time `orm:"null" form:"DateService"` //дата до которой есть гарантия или сервис контракт
	ServiceContract string    `form:"ServiceContract"` //данные по сервисному контракту, уровень сервиса
	Id_User         int64     `form:"Id_User"` //
	Time_Mfy        time.Time `form:"Time_Mfy"` //
}

func (a *StorageSystems) TableName() string {
	return "storage_systems"
}

func (u *StorageSystems) TableIndex() [][]string {
	return [][]string{
		[]string{"ShortName"},
		[]string{"InventoryNum"},
		[]string{"SerialNum"},
	}
}

type StorageVolumes struct {
	Id               int64     `form:"Id"`
	StorageSystemsId int64     `form:"StorageSystemsId"` //ID из StorageSystems
	Status           int       `form:"Status"` //0 - том выведен из работы или удален 1 том в виртуализации 2 все остальные случаи
	// HostId         int64   `form:"HostId"`           //если том не в виртуализации, то сервер к которому он подключен
	UID              string    `form:"UID"` //уникальный идентификатор тома (!!! для RDM дисков может возникнуть проблема будут записи с одинаковыми UID но разными VirtualName)
	Vcenter          string    `form:"Vcenter"` //
	Datacenter       string    `form:"Datacenter"` //
	Cluster          string    `form:"Cluster"` //
	Server           string    `form:"Server"` //Имя сервера к которому подключено хранилище (если их несколько то проблема будет один из них)
	VirtualName      string    `form:"VirtualName"` //Имя хранилища в vmware
	ShortName        string    `form:"ShortName"` //мнемоническое имя для систем
	DiskType         string    `form:"DiskType"` //тип дисков в lun
	StorageName      string    `form:"StorageName"` //краткое наименование СХД
	CapacityGb       float64   `form:"CapacityGb"` //емкость
	FreeSpaceGb      float64   `form:"FreeSpaceGb"` //свободно
	PercentUsed      float64   `form:"PercentUsed"` //процент использования
	Owner            string    `form:"Owner"` //владелец
	Allocation       string    `form:"Allocation"` //аллокация
	Annotation       string    `form:"Annotation"` //Описание
	Id_User          int64     `form:"Id_User"` //
	Time_Mfy         time.Time `form:"Time_Mfy"` //
}

func (a *StorageVolumes) TableName() string {
	return "storage_volumes"
}

func (u *StorageVolumes) TableIndex() [][]string {
	return [][]string{
		[]string{"StorageSystemsId"},
		[]string{"UID"},
		[]string{"VirtualName"},
		[]string{"StorageName"},
		[]string{"ShortName"},
	}
}

type Datastore struct {
	Id               int64   `form:"Id"`
	StorageSystemsId int64   `form:"StorageSystemsId"`
	ClusterId        int64   `form:"ClusterId"` //ссылка на кластер, к которому подключено хранилище
	ShortName        string  `form:"ShortName"` //мнемоническое имя для систем
	StorageName      string  `form:"StorageName"` //краткое наименование СХД
	CapacityGb       float64 `form:"CapacityGb"` //емкость
	FreeSpaceGb      float64 `form:"FreeSpaceGb"` //свободно
	PercentUsed      float64 `form:"PercentUsed"` //процент использования
}

func (a *Datastore) TableName() string {
	return "datastore"
}

func (u *Datastore) TableIndex() [][]string {
	return [][]string{
		[]string{"StorageSystemsId"},
		[]string{"ClusterId", "StorageName"},
	}
}

type VMUsageDatastore struct {
	Id                     int64   `form:"Id"`
	VMSprId                int64   `form:"VMSprId"` //ссылка на VMSprRecords.Id
	UID                    string  `form:"UID"`
	Vcenter                string  `form:"Vcenter"`
	Datacenter             string  `form:"Datacenter"`
	Cluster                string  `form:"Cluster"`
	Datastore              string  `form:"Datastore"`
	StorageName            string  `form:"StorageName"`
	CommittedGb            float64 `form:"Committed"`
	UncommittedGb          float64 `form:"Uncommitted"`
	UnsharedGb             float64 `form:"Unshared"`
	PercentAllocateStorage float64 `form:"PercentAllocateStorage"` //процент выделения места под виртуальную машину на конкретном хранилище
}

func (a *VMUsageDatastore) TableName() string {
	return "vm_usage_datastore"
}

func (u *VMUsageDatastore) TableIndex() [][]string {
	return [][]string{
		[]string{"VMSprId"},
		[]string{"VMSprId", "UID"},
		[]string{"UID"},
		[]string{"StorageName"},
	}
}

type ServerHardwareView struct {
	H     ServerHardware             `form:"H"`
	RAM   []ServerHardwareComponents `form:"RAM"`
	Disks []ServerHardwareComponents `form:"Disks"`
	Add   []ServerHardwareComponents `form:"Add"`
}

type ServerHardware struct {
	Id                int64     `form:"Id" csv:"%" json:"Id"`
	PowerState        int       `form:"PowerState" csv:"Состояние питания" json:"PowerState"` // 0 не указано 1 включен 2 выключен (последнее состояние) при отображении
	Active            int       `form:"Active" csv:"Статус" json:"Active"` // 0 не указано 1 активен 2 выведен из эксплуатации
	TimeActiveState   time.Time `orm:"null" form:"TimeActiveState" csv:"Дата изменения статуса" json:"TimeActiveState"` // время изменения статуса активности введения в эксплуатацию
	ActivatorUser     int64     `orm:"null" form:"ActivatorUser" csv:"Автор ввода в эксплуатацию" json:"ActivatorUser"` // автор записи
	TimeDeActiveState time.Time `orm:"null" form:"TimeDeActiveState" csv:"Дата выведения из эксплуатации" json:"TimeDeActiveState"` // время выведения из эксплуатации
	DeActivatorUser   int64     `orm:"null" form:"DeActivatorUser" csv:"Автор выведения из эксплуатации" json:"DeActivatorUser"` // автор записи
	DeActivateInfo    string    `orm:"null" form:"DeActivateInfo" csv:"Основание выведения из эксплуатации" json:"DeActivateInfo"` // Основание выведения из эксплуатации
	ServerId          int64     `form:"ServerId" csv:"Ссылка на справочник серверов" json:"ServerId"` // ссылка на сервер из справочника серверов
	Vendor            string    `form:"Vendor" csv:"Производитель" json:"Vendor"` // производитель
	Model             string    `form:"Model" csv:"Модель" json:"Model"` // модель
	PartNum           string    `form:"PartNum" csv:"Номер продукта" json:"PartNum"` // продуктовый номер
	FormFactor        int       `form:"FormFactor" csv:"Тип сервера" json:"FormFactor"` // тип сервера 1-rack или 2-blade
	SerialNum         string    `form:"SerialNum" csv:"Серийный номер" json:"SerialNum"` // серийный номер (для физических серверов и хостов виртуализации)
	VirtualSerialNum  string    `form:"VirtualSerialNum" csv:"Виртуальный серийный номер" json:"VirtualSerialNum"` // виртуальный серийный номер (только для хостов tsxi)
	CPUModel          string    `form:"CPUModel" csv:"Модель процессора" json:"CPUModel"` // модель процессоров
	CPUCount          int       `form:"CPUCount" csv:"Количество процессоров" json:"CPUCount"` // Количество физических процессоров
	CorePerCPU        int       `form:"CorePerCPU" csv:"Ядер в процессоре" json:"CorePerCPU"` // количество физических ядер на процессор
	CoreMHz           int       `form:"CoreMHz" csv:"Базовая частота процессора" json:"CoreMHz"` // базовая тактовая частота ядра
	MemoryMb          int       `form:"MemoryMb" csv:"Размер памяти в мб." json:"MemoryMb"` // размер оперативной памяти в мб
	BladeSystemId     int64     `form:"BladeSystemId" csv:"Ссылка на справочник корзин" json:"BladeSystemId"` // ссылка блэйд-шасси в справочнике
	BladeNum          int       `form:"BladeNum" csv:"Номер слота в корзине" json:"BladeNum"` // номер позиции в блэйд шасси
	DatacenterRoom    string    `form:"DatacenterRoom" csv:"ЦОД" json:"DatacenterRoom"` // серверная
	RackNum           string    `form:"RackNum" csv:"Номер стойки" json:"RackNum"` // номер стойки
	StartUnit         string    `form:"StartUnit" csv:"-" json:"StartUnit"` // юнит с которого начинается сервер или блэйд шасси, куда он установлен
	StartUnitPos      int       `form:"StartUnitPos" csv:"Номер юнита" json:"StartUnitPos"` // юнит
	SizeUnit          int       `form:"SizeUnit" csv:"Количество юнитов" json:"SizeUnit"` // количество занимаемых юнитов
	DatacenterName    string    `form:"DatacenterName" csv:"Серверная" json:"DatacenterName"` // Серверное помещение из HP One View
	RackName          string    `form:"RackName" csv:"Шкаф" json:"RackName"` // Шкаф из HP One View
	OwnerFromFGId     int64     `form:"OwnerFromFGId" csv:"-" json:"OwnerFromFGId"` // ЮЛ на которое покупается сервер из справочника MemberFG
	CounterpartyId    int64     `form:"CounterpartyId" csv:"-" json:"CounterpartyId"` // ЮЛ контрагент из справочника Counterparty
	ContractDoc       string    `form:"ContractDoc" csv:"Номер договора" json:"ContractDoc"` // договорной документ на покупку, название и номер дополнительного соглашения
	DateContractDoc   time.Time `orm:"null" form:"DateContractDoc" csv:"Дата договора" json:"DateContractDoc"` // дата документа
	Invoice           string    `form:"Invoice" csv:"Накладная" json:"Invoice"` // счет фактура, накладная
	DateInvoice       time.Time `orm:"null" form:"DateInvoice" csv:"Дата получения" json:"DateInvoice"` // дата накладной
	Counterparty      string    `orm:"null" form:"Counterparty" csv:"Контрагент" json:"Counterparty"` // контрагент
	InventoryNum      string    `form:"InventoryNum" csv:"Инвентарный номер" json:"InventoryNum"` // инвентарный номер
	InventoryNumBuh   string    `form:"InventoryNumBuh" csv:"Бухгалтерский номер" json:"InventoryNumBuh"` // инвентарный номер бухгалтерский
	TechnicalNumberIT string    `form:"TechnicalNumberIT" csv:"Технический номер" json:"TechnicalNumberIT"` // технический номер ИТ
	DateService       time.Time `orm:"null" form:"DateService" csv:"Дата окончания гарантии" json:"DateService"` // дата до которой есть гарантия или сервис контракт
	ServiceContract   string    `form:"ServiceContract" csv:"Сервисный контракт" json:"ServiceContract"` // данные по сервисному контракту, уровень сервиса
	Id_User           int64     `form:"Id_User" csv:"-" json:"Id_User"` //
	Time_Mfy          time.Time `form:"Time_Mfy" csv:"-" json:"Time_Mfy"` //
	SourceUpdate      string    `form:"SourceUpdate" csv:"Источник данных" json:"SourceUpdate"` //
}

// Блок описания для cmdb
// -----------------------------------------------------------------------------------------------------------------------------------------------------------------

type ServerSpr struct {
	Id               int64     `form:"Id" json:"Id"` //id записи
	Type             int       `form:"Type" json:"Type"` //0 - не указан 1 - виртуальный 2 - физический сервер
	PowerState       int       `form:"PowerState" json:"PowerState"` //0 - не указан 1 включен 2 выключен (последнее состояние)
	SerialNumber     string    `form:"SerialNumber" json:"SerialNumber"` //серийный номер для type 2
	Vcenter          string    `form:"Vcenter" json:"Vcenter"` //для type 1 и (type 2 и RoleType 1)
	Datacenter       string    `form:"Datacenter" json:"Datacenter"` //для type 1 и (type 2 и RoleType 1)
	Cluster          string    `form:"Cluster" json:"Cluster"` //для type 1 и (type 2 и RoleType 1)
	VMSprId          int64     `form:"VMSprId" json:"VMSprId"` //ссылка на дополнительные атрибуты для виртуального сервера - справочник VMSpr если Type = 1 иначе 0
	HWSprId          int64     `form:"HWSprId" json:"HWSprId"` //ссылка на дополнительные атрибуты справочник HWSpr для type 2 иначе 0
	Name             string    `form:"Name,text,Name" valid:"MinSize(0);MaxSize(256)" json:"Name"` //vm name для type 1, servername для type 2
	NameFQDN         string    `form:"NameFQDN" json:"NameFQDN"` //hostname для type 1, servername для type 2
	NameInfo         string    `orm:"null" form:"NameInfo" json:"NameInfo"` //альтернативное имя сервера (DNS имена через ;)
	TypeMicroservice int       `form:"TypeMicroservice" json:"TypeMicroservice"` //0 не указано принадлежность к микросервисам, роль в этом пространстве
	RoleType         int       `form:"RoleType" json:"RoleType"` //0-не указано 1- гипервизор vmware 2-APP 3-DB 4-app+DB
	TypeWork         int       `form:"TypeWork" json:"TypeWork"` //0- не указано 1 - продуктовая среда 2- внутренний тест 3 - внешний тест
	CriticalLevel    int       `form:"CriticalLevel" json:"CriticalLevel"` //уровень критичности 0 не указано 1-Vital 2-Critical 3-Secondary 4-Non Priority
	Stat             int       `form:"Stat" json:"Stat"` //0- историческая запись 911 новое добавлено автоматически (требуется правка) 1- не активное 2- в эксплуатации // TODO: comment truncated in screenshot
	Monitoring       int       `form:"Monitoring" json:"Monitoring"` //0 - не указано 1-включен мониторинг 2-отключен мониторинг
	Backuped         int       `form:"Backuped" json:"Backuped"` //0 - не указано 1-включено резервное копирование 2-отключено резервное копирование
	Network          string    `orm:"null" form:"Network" json:"Network"` //сети через ; vlan144
	IPAddress        string    `orm:"null" form:"IPAddress" json:"IPAddress"` //список IP адресов сервера через ; при наличие информации подгружается из конфигурации VM или вводится вручную // TODO: comment truncated in screenshot
	CPUCount         float64   `form:"CPUCount" json:"CPUCount"` //Количество физических процессоров (корпусов, гнёзд)
	CPULogical       float64   `form:"CPULogical" json:"CPULogical"` //общее количество логических CPU (ядер)
	MemSize          float64   `form:"MemSize" json:"MemSize"` //размер оперативной памяти сервера гигабайт
	DiskSize         float64   `form:"DiskSize" json:"DiskSize"` //суммарный размер всех дисков гигабайт
	OS               string    `form:"OS" json:"OS"` //операционная система
	ITSystemKey      string    `form:"ITSystemKey" json:"ITSystemKey"` //ссылка на ИТ систему из jira https://jira.bcs.ru/rest/assets/latest/aql/objects?qlQuery=objectTypeId // TODO: URL truncated in screenshot
	TimeCreate       time.Time `orm:"null" form:"TimeCreate" json:"TimeCreate"` //время создания записи
	CreatorUser      int64     `orm:"null" form:"CreatorUser" json:"CreatorUser"` //автор записи
	TimeActiveState  time.Time `orm:"null" form:"TimeActiveState" json:"TimeActiveState"` //дата ввода в эксплуатацию
	ActivatorUser    int64     `orm:"null" form:"ActivatorUser" json:"ActivatorUser"` //автор записи
	TimeDeActiveState time.Time `orm:"null" form:"TimeDeActiveState" json:"TimeDeActiveState"` //дата выведения из эксплуатации
	DeActivatorUser  int64     `orm:"null" form:"DeActivatorUser" json:"DeActivatorUser"` //автор записи
	TimeActiveMonitoring time.Time `orm:"null" form:"TimeActiveMonitoring" json:"TimeActiveMonitoring"` //время постановки на мониторинг
	Owner            string    `form:"Owner" json:"Owner" csv:"-"` //владелец сервера (отвественный за сервер)
	BusinessOwner    string    `form:"BusinessOwner" json:"BusinessOwner" csv:"-"` //владелец от бизнеса
	ITOwner          string    `form:"ITOwner" json:"ITOwner" csv:"-"` //владелец со стороны ИТ
	ProductOwner     string    `form:"ProductOwner" json:"ProductOwner" csv:"-"` //Владелец продукта
	Support          string    `form:"Support" json:"Support" csv:"-"` //ответственные за поддержку
	SupportTeam      string    `form:"SupportTeam" json:"SupportTeam" csv:"-"` //Подразделение поддержки
	SystemType       string    `form:"SystemType" json:"SystemType"` //тип системы техническая ...
	LicensingType    string    `form:"LicensingType" json:"LicensingType"` //тип лицензирования
	SolutionType     string    `form:"SolutionType" json:"SolutionType"` //вариант решения (коробочной)
	Users            string    `form:"Users" json:"Users" csv:"-"` //пользователи системы
	Allocation       string    `form:"Allocation" json:"Allocation"` //аллокация (подразделение)
	Anotation        string    `form:"Anotation" json:"Anotation" csv:"-"` //описане сервера
	TagName          string    `form:"TagName" json:"TagName"` //Имя тэга для виртуальных машин (Код для справочника)
	TagDescription   string    `form:"TagDescription" json:"TagDescription"` //Описание тэга для виртуальных машин
	MonitoringSZ     string    `form:"MonitoringSZ" json:"MonitoringSZ"` //основание постановки на мониторинг
	SzHistory        string    `form:"SzHistory" json:"SzHistory"` //номера сз через пробел в хронологическом порядке
	DPCOffice        int       `form:"DPCOffice" json:"DPCOffice"` //размещение в ЦОД по справочнику комната с адресом
	Id_User          int64     `form:"Id_User" json:"Id_User"` //автор изменений
	Time_Mfy         time.Time `form:"Time_Mfy" json:"Time_Mfy"` //дата последних изменений
}

func (a *ServerSpr) TableName() string {
	return "cmdb_server_spr"
}

func (u *ServerSpr) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"SerialNum"},
		[]string{"Vcenter", "Datacenter", "Cluster"},
		[]string{"VMSprId"},
		[]string{"HWSprId"},
		[]string{"ITSystemKey", "Name"},
		[]string{"PowerState", "Name"},
	}
}

type VMSpr struct {
	Id                int64     `form:"Id"` //id записи
	ServerId          int64     `form:"ServerId"` //ссылка на ServerSpr.Id
	Vcenter           string    `form:"Vcenter"` //управляющий центр vmware
	Datacenter        string    `form:"Datacenter"` //датацентр vmware
	Cluster           string    `form:"Cluster"` //кластер vmware
	Name              string    `form:"Name"` //имя виртуальной машины
	HostName          string    `form:"HostName"` //имя хоста (проверять на localhost и обозначать потребность исправления)
	GuestFullName     string    `form:"GuestFullName"` //операционная система (данные из описания виртуальной машины, может не совпадать с реальной)
	PathName          string    `form:"PathName"` //путь к файлам виртуальной машины на хранилище
	PowerState        bool      `form:"PowerState"` //состояние включения виртуальной машины true - включена
	Uuid              string    `form:"Uuid"` //уникальный в рамках vcenter идентификатор виртуальной машины
	Annotation        string    `form:"Annotation" csv:"-"` //описание виртуальной машины в свободном формате
	NumCPU            int32     `form:"NumCPU"` //количество vCPU
	NumCores          int32     `form:"NumCores"` //количество ядер
	MemorySizeGB      float64   `form:"MemorySizeGB"` //размер оперативной памяти в гигабайтах
	Network           string    `form:"Network"` //сети через разделитель ;
	IpAddress         string    `form:"IpAddress"` //IP адреса через разделитель ;
	NumEthernetCards  int32     `form:"NumEthernetCards"` //количество сетевых интерфейсов
	NumVirtualDisks   int32     `form:"NumVirtualDisks"` //количество виртуальных дисков
	DiskCommittedGB   float64   `form:"DiskCommittedGB"` //выделенное дисковое пространство с служебной информацией
	DiskUncommittedGB float64   `form:"DiskUncommittedGB"` //занятое дисковое пространство
	CapacityGb        float64   `form:"CapacityGb"` //форматная ёмкость дисков (при установленных vmware tools)
	FreeSpaceGb       float64   `form:"FreeSpaceGb"` //свободное место (при установленных vmware tools)
	PercentUsed       float64   `form:"PercentUsed"` //процент использования (при установленных vmware tools)
	Time_Mfy          time.Time `form:"Time_Mfy" json:"Time_Mfy"` //дата последних изменений (изменения вносятся в автоматическом режиме)
}

func (a *VMSpr) TableName() string {
	return "cmdb_vm_spr"
}

func (u *VMSpr) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Name"},
		[]string{"HostName"},
		[]string{"PowerState", "Name"},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name"},
		[]string{"ServerId"},
	}
}

type HWSpr struct {
	Id                int64     `form:"Id" csv:"%" json:"Id"`
	PowerState        int       `form:"PowerState" csv:"Состояние питания" json:"PowerState"` // 0 не указано 1 включен 2 выключен (последнее состояние)
	ServerId          int64     `form:"ServerId" csv:"Ссылка на справочник серверов" json:"ServerId"` //ссылка на ServerSpr.Id
	CUId              int64     `form:"CUId" csv:"-" json:"CUId"` //ссылка на ConfigurationUnit.Id
	Vendor            string    `form:"Vendor" csv:"Производитель" json:"Vendor"` //производитель
	Model             string    `form:"Model" csv:"Модель" json:"Model"` //модель
	PartNum           string    `form:"PartNum" csv:"Номер продукта" json:"PartNum"` //продуктовый номер
	Type              int       `form:"Type" csv:"Тип сервера" json:"Type"` //тип сервера 1-rack или 2-blade
	SerialNum         string    `form:"SerialNum" csv:"Серийный номер" json:"SerialNum"` //серийный номер (для физических серверов и хостов виртуализации)
	CPUModel          string    `form:"CPUModel" csv:"Модель процессора" json:"CPUModel"` //модель процессоров
	CPUCount          int       `form:"CPUCount" csv:"Количество процессоров" json:"CPUCount"` //Количество физических процессоров
	CorePerCPU        int       `form:"CorePerCPU" csv:"Ядер в процессоре" json:"CorePerCPU"` //Количество физических ядер на процессор
	CoreMHz           int       `form:"CoreMHz" csv:"Базовая частота процессора" json:"CoreMHz"` //базовая тактовая частота ядра
	MemoryMb          int       `form:"MemoryMb" csv:"Размер памяти в мб." json:"MemoryMb"` //размер оперативной памяти в мб
	BladeSystemId     int64     `form:"BladeSystemId" csv:"Ссылка на справочник корзин" json:"BladeSystemId"` //ссылка блэйд-шасси в справочнике
	BladeNum          int       `form:"BladeNum" csv:"Номер слота в корзине" json:"BladeNum"` //номер позиции в блэйд шасси
	DatacenterName    string    `form:"DatacenterName" csv:"Серверная" json:"DatacenterName"` //Серверное помещение из HP One View
	RackName          string    `form:"RackName" csv:"Шкаф" json:"RackName"` //Шкаф из HP One View
	StartUnitPos      int       `form:"StartUnitPos" csv:"Номер юнита" json:"StartUnitPos"` //юнит с которого начинается сервер или блэйд шасси, куда он установлен
	SizeUnit          int       `form:"SizeUnit" csv:"Количество юнитов" json:"SizeUnit"` //количество занимаемых unit
	InventoryNum      string    `form:"InventoryNum" csv:"Инвентарный номер" json:"InventoryNum"` //Инвентарный номер
	InventoryNumBuh   string    `form:"InventoryNumBuh" csv:"Бухгалтерский номер" json:"InventoryNumBuh"` //Инвентарный номер бухгалтерский
	TechnicalNumberIT string    `form:"TechnicalNumberIT" csv:"Технический номер" json:"TechnicalNumberIT"` //технический номер ИТ
	Id_User           int64     `form:"Id_User" csv:"-" json:"Id_User"` //
	Time_Mfy          time.Time `form:"Time_Mfy" csv:"-" json:"Time_Mfy"` //
}

func (a *HWSpr) TableName() string {
	return "cmdb_vm_spr"
}

func (u *HWSpr) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Name"},
		[]string{"HostName"},
		[]string{"PowerState", "Name"},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name"},
		[]string{"ServerId"},
	}
}

type ConfigurationUnit struct {
	Id                int64     `form:"Id" csv:"-" json:"Id"`
	StateWork         int       `form:"StateWork" csv:"Статус" json:"StateWork"` //0 не указано 1 активен 2 выведен из эксплуатации
	TimeActiveState   time.Time `orm:"null" form:"TimeActiveState" csv:"Дата изменения статуса" json:"TimeActiveState"` //время изменения статуса активности введения в эксплуатацию
	ActivatorUser     int64     `form:"ActivatorUser" csv:"Автор ввода в эксплуатацию" json:"ActivatorUser"` //автор записи
	TimeDeActiveState time.Time `orm:"null" form:"TimeDeActiveState" csv:"Дата выведения из эксплуатации" json:"TimeDeActiveState"` //время выведения из эксплуатации
	DeActivatorUser   int64     `form:"DeActivatorUser" csv:"Автор выведения из эксплуатации" json:"DeActivatorUser"` //автор записи
	ReasonEndLife     int64     `orm:"null" form:"ReasonEndLife" csv:"Основание выведения из эксплуатации" json:"ReasonEndLife"` //ReasonEndLife.Code Основание выведения из эксплуатации
	ServerId          int64     `form:"ServerId" csv:"Ссылка на справочник серверов" json:"ServerId"` //ссылка на сервер из справочника серверов
	NameFQDN          string    `form:"NameFQDN" csv:"Имя сервера" json:"NameFQDN"` //имя сервера
	SerialNum         string    `form:"SerialNum" csv:"Серийный номер" json:"SerialNum"` //серийный номер (для физических серверов и хостов виртуализации)
	ModelEquipment    int64     `form:"ModelEquipment" csv:"Модель норматив" json:"ModelEquipment"` //ModelEquipment.code модель-норматив по справочнику
	Datacenter        int64     `form:"Datacenter" csv:"ЦОД" json:"Datacenter"` //Datacenter.Code серверная из справочника
	Rack              int64     `form:"Rack" csv:"Номер стойки" json:"Rack"` //Rack.Code номер стойки из справочника для конкретного Datacenter
	StartUnit         int64     `form:"StartUnit" csv:"стартовый юнит" json:"StartUnit"` //юнит с которого начинается сервер или блэйд шасси, куда он установлен
	InventoryNum      string    `form:"InventoryNum" csv:"Инвентарный номер" json:"InventoryNum"` //Инвентарный номер
	InventoryNumBuh   string    `form:"InventoryNumBuh" csv:"Бухгалтерский номер" json:"InventoryNumBuh"` //Инвентарный номер бухгалтерский
	TechnicalNumberIT string    `form:"TechnicalNumberIT" csv:"Технический номер IT" json:"TechnicalNumberIT"` //технический номер ИТ
	Id_User           int64     `form:"Id_User" csv:"Автор изменений" json:"Id_User"`
	Time_Mfy          time.Time `orm:"null" form:"Time_Mfy" csv:"Дата последних изменений" json:"Time_Mfy"`
}

/*
// TODO: commented-out purchase/contract fields block truncated in screenshot
OwnerFromFGId int64
CounterpartyId int64
ContractDoc string
DateContractDoc time.Time
ContractAmount float64
ContractCurrency int64
Invoice string
DateInvoice time.Time
InvoiceAmount float64
InvoiceCurrency int64
DateService time.Time
ServiceContract string
SourceUpdate string
*/

func (a *ConfigurationUnit) TableName() string {
	return "cmdb_configuration_unit"
}

func (u *ConfigurationUnit) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ServerId"},
		[]string{"SerialNum"},
		[]string{"ModelEquipment"},
		[]string{"Datacenter"},
		[]string{"Datacenter", "Rack"},
	}
}

//выделить процессоры, память ... как компоненты и вычислять размер памяти или количество процессоров исходя из компонент

// таблица связей конфигурационных единиц как пример связи блэйд корзин и лезвий
type ConfigurationUnitLink struct {
	Id        int64 `form:"Id" orm:"auto"` //
	OwnerUnit int64 `form:"OwnerUnit"` //родительское устройство
	LinkUnit  int64 `form:"LinkUnit"` //дочернее устройство
}

func (a *ConfigurationUnitLink) TableName() string {
	return "cmdb_configuration_unit_link"
}

func (u *ConfigurationUnitLink) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"OwnerUnit"},
		[]string{"LinkUnit"},
	}
}

// цод
type Datacenter struct {
	Id    int64  `form:"Id" orm:"auto" csv:"%"` //
	Code  int64  `form:"Code" csv:"Код"` //уникальный код
	Name  string `form:"Name" csv:"Наименование"` //Наименование ЦОД
	Width int64  `form:"Width" csv:"Ширина"`
	Depth int64  `form:"Depth" csv:"Глубина"`
}

func (a *Datacenter) TableName() string {
	return "cmdb_datacenter"
}

func (u *Datacenter) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
	}
}

// стойка в ЦОД
type Rack struct {
	Id         int64  `form:"Id" orm:"auto" csv:"%"` //
	Code       int64  `form:"Code" csv:"Код"` //уникальный код
	Datacenter int64  `form:"Datacenter" csv:"ЦОД"` //Datacenter.Code ЦОД в котором стоит стойка
	Name       string `form:"Name" csv:"Наименование"` //Наименование стойки
	Width      int64  `form:"Width" csv:"Ширина"`
	Depth      int64  `form:"Depth" csv:"Глубина"`
	Height     int64  `form:"Height" csv:"Высота"`
	UHeight    int64  `form:"UHeight" csv:"Высота юниты"`
	//добавить размеры включая высоту в unit
}

func (a *Rack) TableName() string {
	return "cmdb_rack"
}

func (u *Rack) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
		[]string{"Datacenter"},
		[]string{"Datacenter", "Code"},
	}
}

// модель норматив для бюджетирования
type ModelEquipment struct {
	Id                   int64   `form:"Id" orm:"auto" csv:"%"` //
	ModelVendor          int64   `form:"ModelVendor" csv:"Модель вендора"` //модель у вендора
	Code                 int64   `form:"Code" csv:"Код норматива"` //код норматива
	Name                 string  `form:"Name" csv:"Наименование норматива"` //Наименование норматива
	PartNum              string  `form:"PartNum" csv:"Номер продукта из Oneview"` //номер продукта из Oneview
	PartNumSale          string  `form:"PartNumSale" csv:"Номер продукта для заказа"` //номер продукта для заказа
	Money                float64 `form:"Money" csv:"Цена для оценки"` //Цена для оценки
	Currency             int64   `form:"Currency" csv:"Валюта"` //валюта
	MoneyInfo            string  `form:"MoneyInfo" csv:"Обоснование цены"` //обоснование цены
	ChangeModelEquipment int64   `form:"ChangeModelEquipment" csv:"Норматив для замены"` //Норматив для замены
	ChangeMoney          float64 `form:"Money" csv:"Цена для оценки"` //Цена для оценки
	ChangeCurrency       int64   `form:"Currency" csv:"Валюта"` //валюта
	ChangeMoneyInfo      string  `form:"MoneyInfo" csv:"Обоснование цены"` //обоснование цены
	Detail               string  `form:"Detail" csv:"Подробное описание"` //подробное описание (текстовое описание конфигурации, помимо реальной конфигурации)
}

func (a *ModelEquipment) TableName() string {
	return "cmdb_model_equipment"
}

func (u *ModelEquipment) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
		[]string{"ModelVendor"},
		[]string{"Name"},
	}
}

type ModelVendor struct {
	Id                int64  `form:"Id" orm:"auto" csv:"%"` //
	Code              int64  `form:"Code" csv:"Код"` //уникальный код модели
	CategoryEquipment int64  `form:"CategoryEquipment" csv:"Категория оборудования"` //Категория оборудования из справочника CategoryEquipment.Code (серверное,сетевое, электропитание, оргтехника)
	Vendor            string `form:"Vendor" csv:"Производитель"` //производитель
	Model             string `form:"Model" csv:"Модель"` //Наименование модели
	Type              int64  `form:"Type" csv:"Тип"` //тип сервера данные из ServerType.Code (1-rack или 2-blade зарезервировано) //только для серверного оборудования
	Detail            string `form:"Detail" csv:"Подробное описание"` //подробное описание
	SizeUnit          int64  `form:"SizeUnit" csv:"Количество юнитов" json:"SizeUnit"` //количество занимаемых unit
}

func (a *ModelVendor) TableName() string {
	return "cmdb_model_vendor"
}

func (u *ModelVendor) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
		[]string{"CategoryEquipment"},
	}
}

// список компонент для конфигурационной единицы
type Components struct {
	Id                int64  `form:"Id" orm:"auto" csv:"%u"` //
	ConfigurationUnit int64  `form:"ConfigurationUnit" csv:"Конфигурационная единица"` //конфигурационная единица, содержащая компоненту
	CategoryEquipment int64  `form:"CategoryEquipment" csv:"Категория оборудования"` //Категория оборудования из справочника CategoryEquipment.Code (серверное,сетевое, электропитание, оргтехника)
	Type              int64  `form:"Type" csv:"Тип"` //тип компоненты ComponentType.Code (Процессор, память, SSD, HDD, видеокарта, сетевая карта, Raid контроллер, HBA ...)
	Component         int64  `form:"Component" csv:"Ссылка по типу"` //ссылка на ComponentsCPU.Code, ComponentsRAM.Code, и т. д. по типу из ComponentEquipment
	SerialNum         string `form:"SerialNum" csv:"Серийный номер"` //серийный номер
	Location          string `form:"Location" csv:"Размещение"` //размещение
}

func (a *Components) TableName() string {
	return "cmdb_components"
}

func (u *Components) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ConfigurationUnit"},
		[]string{"CategoryEquipment"},
		[]string{"Type"},
		[]string{"SerialNum"},
		[]string{"Component"},
	}
}

// справочник процессоров
type ComponentsCPU struct {
	Id                 int64  `form:"Id" orm:"auto" csv:"%u"` //
	Code               int64  `form:"Code" csv:"Код"` //код процессора
	ProcessorType      string `form:"ProcessorType" csv:"Наименование процессора"` //наименование процессора
	ProcessorCoreCount int    `form:"ProcessorCoreCount" csv:"Количество ядер"` //количество физических ядер на процессор
	ProcessorSpeedMhz  int    `form:"ProcessorSpeedMhz" csv:"Тактовая частота"` //базовая тактовая частота ядра
}

func (a *ComponentsCPU) TableName() string {
	return "cmdb_components_cpu"
}

func (u *ComponentsCPU) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
		[]string{"ProcessorType"},
	}
}

// Оперативная память
type ComponentsRAM struct {
	Id                int64  `form:"Id" orm:"auto" csv:"#"` //
	Code              int64  `form:"Code" csv:"Код"` //код памяти
	Model             string `form:"Model" csv:"Модель"` //fmt.Sprintf("%.0fGb", float64(mem.CapacityMiB)/1024.0) + " " + mem.BaseModuleType + " " + mem.MemoryDeviceType + " " + strconv.Itoa // TODO: comment truncated in screenshot
	CapacityMiB       int    `form:"CapacityMiB" csv:"Емкость Мб"` //объём в мегабайтах
	BaseModuleType    string `form:"BaseModuleType" csv:"Тип модуля"` //
	MemoryDeviceType  string `form:"MemoryDeviceType" csv:"Тип устройства"` //
	OperatingSpeedMhz int    `form:"OperatingSpeedMhz" csv:"Частота"` //
	Vendor            string `form:"Vendor" csv:"Производитель"` //производитель
	PartNumber        string `form:"PartNumber" csv:"Номер продукта One View"` //номер продукта из HP OneView
	PartNumberSale    string `form:"PartNumberSale" csv:"Номер для заказа"` //номер продукта для заказа
	RankCount         int    `form:"RankCount" csv:"Ранг"` //
}

func (a *ComponentsRAM) TableName() string {
	return "cmdb_components_ram"
}

func (u *ComponentsRAM) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
		[]string{"Model"},
	}
}

type ComponentType struct {
	Id   int64  `form:"Id" orm:"auto" csv:"#"` //
	Code int32  `form:"Code" csv:"Код"` //код компоненты
	Name string `form:"Name" csv:"Наименование"` //Наименование компоненты
}

func (a *ComponentType) TableName() string {
	return "cmdb_component_type"
}

func (u *ComponentType) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
	}
}

// причины окончания эксплуатации
type ModelsEndLife struct {
	Id               int64     `form:"Id" orm:"auto" csv:"%"` //
	ModelLink        int64     `form:"ModelLink" csv:"Ссылка на модель"` // ссылка на модель оборудования ModelEquipment.Id
	EndWork          time.Time `orm:"null" form:"EndWork" csv:"Дата окончания эксплуатации"` // Дата окончания эксплуатации по указанной причине
	EndWorkYearCount int64     `form:"EndWorkYearCount" csv:"Количество лет до окончания эксплуатации"` // количество лет до наступления вывода из эксплуатации с момента покупки
	ReasonEndLife    int64     `form:"ReasonEndLife" csv:"Причина вывода"` // причина вывода из эксплуатации ReasonEndLife.Code
}

func (a *ModelsEndLife) TableName() string {
	return "cmdb_models_end_life"
}

func (u *ModelsEndLife) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ModelLink"},
	}
}

// причина вывода из эксплуатации
type ReasonEndLife struct {
	Id   int64  `form:"Id" orm:"auto" csv:"%"` //
	Code int64  `form:"Code" csv:"Код"` // код причины вывода из эксплуатации
	Name string `form:"Name" csv:"Описание"` // текстовое описание причины для отображения
}

func (a *ReasonEndLife) TableName() string {
	return "cmdb_reason_end_life"
}

func (u *ReasonEndLife) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
	}
}

// Категории оборудования
type CategoryEquipment struct {
	Id       int64  `form:"Id" orm:"auto" csv:"%"`
	Code     int64  `form:"Code" csv:"Код"` //код категории оборудования (зарезервировано 1-серверное,2-сетевое, 3-электропитание, 4-оргтехника)
	Name     string `form:"Name" csv:"Наименование"` //Наименование
	LifeTime int64  `form:"LifeTime" csv:"Срок эксплуатации в годах"` //срок эксплуатации в годах время эксплуатации по правилам компании
}

func (a *CategoryEquipment) TableName() string {
	return "cmdb_category_equipment"
}

func (u *CategoryEquipment) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
	}
}

// время эксплуатации по правилам компании
/*
type LifeTime struct {
	Id   int64  `form:"Id" orm:"auto"` //
	Code int32  `form:"Code"`          //Срок жизни в годах
	Name string `form:"Name"`          //срок жизни строкой для отображения
}
*/

type ServerType struct {
	Id   int64  `form:"Id" orm:"auto"` //
	Code int64  `form:"Code"` //код типа сервера
	Type string `form:"Type"` //Наименование типа сервера
}

func (a *ServerType) TableName() string {
	return "cmdb_server_type"
}

func (u *ServerType) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
	}
}

// справочник операционных систем
type OperationalSystem struct {
	Id   int64  `form:"Id" orm:"auto" csv:"%"` //
	Code int64  `form:"Code" csv:"Код"` //код операционной системы
	Name string `form:"Name" csv:"Наименование"` //наименование операционной системы
}

func (a *OperationalSystem) TableName() string {
	return "cmdb_operational_system"
}

func (u *OperationalSystem) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Code"},
		[]string{"Name"},
	}
}

//---------------------------------------------------------

type ServerOwnershipHistory struct {
	Id              int64     `form:"Id" orm:"auto"` //
	Date            time.Time `orm:"null" form:"Date"` //дата операции внутренняя
	HardwareId      int64     `form:"HardwareId"` //ссылка на Id в ServerHardware
	Type            int       `form:"Type"` //тип операции 0 - не указано 1 - покупка у внешнего контрагента 2 внутренне перемещение внутри ФГ 3 продажа внешнему контрагенту
	OwnerFromFG1Id  int64     `form:"OwnerFromFG1Id"` //ЮЛ на которое покупается сервер из справочника MemberFG
	CounterpartyId  int64     `form:"CounterpartyId"` //ЮЛ контрагент из справочника Counterparty
	OwnerFromFG2Id  int64     `form:"OwnerFromFG2Id"` //ЮЛ с которого передается сервер, при внутреннем перемещении (из справочника MemberFG) или продаже внешнему
	ContractDoc     string    `form:"ContractDoc"` //договорной документ на покупку, название и номер дополнительного соглашения
	DateContractDoc time.Time `orm:"null" form:"DateContractDoc"` //дата документа
	Invoice         string    `form:"Invoice"` //счет фактура, накладная
	DateInvoice     time.Time `orm:"null" form:"DateInvoice"` //дата накладной
	Id_User         int64     `form:"Id_User"` //
	Time_Mfy        time.Time `form:"Time_Mfy"` //
}

func (a *ServerOwnershipHistory) TableName() string {
	return "server_ownership_history"
}

func (u *ServerOwnershipHistory) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"HardwareId"},
		[]string{"HardwareId", "Id"},
	}
}

func (a *ServerHardware) TableName() string {
	return "server_hardware"
}

func (u *ServerHardware) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"ServerId"},
		[]string{"InventoryNum"},
		[]string{"SerialNum"},
	}
}

type ServerHardwareComponents struct {
	Id              int64     `form:"Id"`
	ServerId        int64     `form:"ServerId"` //ссылка на сервер из справочника серверов (для компонент серверов)
	BladeSystemId   int64     `form:"BladeSystemId"` //ссылка блэйд-шасси в справочнике (для компонент блэйд шасси)
	StorageId       int64     `form:"StorageId"` //ссылка СХД в справочнике (для компонент СХД таких как диски ...)
	Type            int       `form:"Type"` //тип компонент 1 память, 2 диск, 3 сетевой контроллер, 4 FC контроллер, 5 RAID контроллер, 6 interconnect blade, 7 managment card blade, 8 // TODO: comment truncated in screenshot
	SubType         int       `form:"SubType"` //для дисков 1-SSD 2-SAS 3-SATA
	Vendor          string    `form:"Vendor"` //производитель
	Model           string    `form:"Model"` //модель
	SerialNum       string    `form:"SerialNum"` //серийный номер (там где есть, если нет наследуем от сервера с дополнением порядкового номера)
	CapacityGb      float64   `form:"CapacityGb"` //для дисков и памяти
	Slot            string    `form:"Slot"` //расположение в родительском устройстве
	ContractDoc     string    `form:"ContractDoc"` //договорной документ на покупку, название и номер дополнительного соглашения \
	DateContractDoc time.Time `orm:"null" form:"DateContractDoc"` //дата документа                                          \ только для отдельно приобретаемых комплектующих
	Invoice         string    `form:"Invoice"` //счет фактура, накладная                                       / возможно нужно учесть ЦФО с которого оплата
	DateInvoice     time.Time `orm:"null" form:"DateInvoice"` //дата накладной                                               /
	DateService     time.Time `orm:"null" form:"DateWork"` //дата до которой есть гарантия или сервис контракт
	ServiceContract string    `form:"ServiceContract"` //данные по сервисному контракту, уровень сервиса
	Id_User         int64     `form:"Id_User"` //
	Time_Mfy        time.Time `form:"Time_Mfy"` //
}

func (a *ServerHardwareComponents) TableName() string {
	return "server_hardware_components"
}

func (u *ServerHardwareComponents) TableIndex() [][]string {
	return [][]string{
		[]string{"ServerId"},
		[]string{"ServerId", "Type"},
		[]string{"BladeSystemId"},
		[]string{"StorageId"},
	}
}

type SprDiskInfo struct {
	Id          int64   `form:"Id"`
	ServerId    int64   `form:"ServerId"` //ссылка на сервер из справочника серверов
	DiskPath    string  `form:"DiskPath"`
	CapacityGb  float64 `form:"CapacityGb"`
	FreeSpaceGb float64 `form:"FreeSpaceGb"`
	PercentUsed float64 `form:"PercentUsed"`
}

func (a *SprDiskInfo) TableName() string {
	return "spr_disk_info"
}

func (u *SprDiskInfo) TableIndex() [][]string {
	return [][]string{
		[]string{"ServerId", "Id"},
	}
}

type SprCFOAllocation struct {
	Id            int64   `form:"Id"` //
	ServerId      int64   `form:"ServerId"` //ссылка на сервер из справочника серверов
	BladeSystemId int64   `form:"BladeSystemId"` //ссылка блэйд-шасси в справочнике (для аллокаций блэйд шасси)
	StorageId     int64   `form:"StorageId"` //ссылка СХД в справочнике (для компонент СХД таких как диски ...)
	ShareF        float64 `form:"ShareF"` //доля значение от 0.0 до 100.0 (проценты float) в аллокации по CFO, сумма долей по серверу должна быть 100%, на каждый сервер может быть несколько записей
	CFO           string  `form:"CFO"` //код CFO
	Name          string  `form:"Name"` //Наименование ЦФО
	Busines       string  `form:"Busines"` //Бизнес колонна
}

func (a *SprCFOAllocation) TableName() string {
	return "spr_cfo_allocation"
}

func (u *SprCFOAllocation) TableIndex() [][]string {
	return [][]string{
		[]string{"ServerId", "Id"},
		[]string{"BladeSystemId", "Id"},
		[]string{"StorageId", "Id"},
	}
}

type SprBusinesAllocation struct {
	Id            int64   `form:"Id"`
	ServerId      int64   `form:"ServerId"` //ссылка на сервер из справочника серверов
	BladeSystemId int64   `form:"BladeSystemId"` //ссылка блэйд-шасси в справочнике (для аллокаций блэйд шасси)
	StorageId     int64   `form:"StorageId"` //ссылка СХД в справочнике (для компонент СХД таких как диски ...)
	Share         int     `form:"Share"` //доля значение от 0 до 100 (проценты) в аллокации по бизнесам, сумма долей по серверу должна быть 100%, на каждый сервер может быть несколько записей
	ShareF        float64 `form:"ShareF"`
	Allocation    string  `form:"Allocation"`
	Busines       string  `form:"Busines"`
	Busines_ln    string  `form:"Busines_ln"`
	Center_money  string  `form:"Center_money"`
}

func (a *SprBusinesAllocation) TableName() string {
	return "spr_busines_allocation"
}

func (u *SprBusinesAllocation) TableIndex() [][]string {
	return [][]string{
		[]string{"ServerId", "Id"},
		[]string{"BladeSystemId", "Id"},
		[]string{"StorageId", "Id"},
	}
}

type LoadFilter struct {
	FromTime     time.Time `form:"FromTime"`
	ToTime       time.Time `form:"ToTime"`
	ServerName   string    `form:"ServerName,text,ServerName:" valid:"MinSize(0);MaxSize(256)"`
	Inventory    string    `form:"Inventory,text,Inventory:" valid:"MinSize(0);MaxSize(256)"`
	Owner        string    `form:"Owner,text,Owner:" valid:"MinSize(0);MaxSize(256)"`
	Allocation   string    `form:"Allocation,text,Allocation:" valid:"MinSize(0);MaxSize(256)"`
	Anotation    string    `form:"Anotation,text,Anotation:" valid:"MinSize(0);MaxSize(256)"`
	Datacenter   string    `form:"Datacenter,text,Datacenter:" valid:"MinSize(0);MaxSize(256)"`
	Claster      string    `form:"Claster,text,Claster:" valid:"MinSize(0);MaxSize(256)"`
	Service      string    `form:"Service,text,Service:" valid:"MinSize(0);MaxSize(256)"`
	Busines      string    `form:"Busines,text,Busines:" valid:"MinSize(0);MaxSize(256)"`
	Businesln    string    `form:"Businesln,text,Businesln:" valid:"MinSize(0);MaxSize(256)"`
	CenterMonney string    `form:"CenterMonney,text,CenterMoney:" valid:"MinSize(0);MaxSize(256)"`
	Map          map[string]interface{}
}

type ResourceMap struct {
	CountServers            float64 `form:"CountServers"`
	CPUCount                float64 `form:"CPUCount"`
	CoreCount               float64 `form:"CoreCount"`
	MemSize                 float64 `form:"MemSize"`
	SysDiskSize             float64 `form:"SysDiskSize"`
	WorkDiskSize            float64 `form:"WorkDiskSize"`
	CPULoadAvg              float64 `form:"CPULoadAvg"`
	CPULoadMin              float64 `form:"CPULoadMin"`
	CPULoadMax              float64 `form:"CPULoadMax"`
	MemUseAvg               float64 `form:"MemUseAvg"`
	MemUseMin               float64 `form:"MemUseMin"`
	MemUseMax               float64 `form:"MemUseMax"`
	SysDiskFreePercentAvg   float64 `form:"SysDiskFreePercentAvg"`
	SysDiskFreePercentMin   float64 `form:"SysDiskFreePercentMin"`
	SysDiskFreePercentMax   float64 `form:"SysDiskFreePercentMax"`
	WorkDiskFreePercentAvg  float64 `form:"WorkDiskFreePercentAvg"`
	WorkDiskFreePercentMin  float64 `form:"WorkDiskFreePercentMin"`
	WorkDiskFreePercentMax  float64 `form:"WorkDiskFreePercentMax"`
	Allocation              string  `form:"Allocation"`
	Busines                 string  `form:"Busines"`
	Count                   float64 `form:"Count"`
}

type AuthUser struct {
	Id        int       `form:"id"`
	Group     int64     `form:"Group"`
	First     string    `form:"first" valid:"Required"`
	Last      string    `form:"last"`
	Email     string    `form:"email" valid:"Email" orm:"unique"`
	Password  string    `form:"password"`
	Reg_key   string
	Reg_date  time.Time `orm:"auto_now_add;type(datetime)"`
	Reset_key string
	Priv      int       `form:"priv"`
	Podr      int       `form:"podr"`
}

type UserLog struct {
	Id      int64     `form:"id"` // id записи
	UserId  int64     `orm:"null" form:"userid"` // id пользователя
	Login   time.Time `orm:"null" form:"login"` // время регистрации
	Time    time.Time `orm:"null" form:"last"` // время операции
	Type    int       `form:"type"` // тип операции (1 авторизация, 2 выход, 3 сброс пароля, 4 обновление профиля, 5 чтение справочника, 6 изменение данных справочника, 7 чтение таблицы загрузки // TODO: comment truncated in screenshot
	Request string    `form:"request"` // текст запроса
}

func (a *UserLog) TableName() string {
	return "userlog"
}

func (a *AuthUser) TableName() string {
	return "users"
}

func (a *ReportServerRecord) TableName() string {
	return "report"
}

func (a *VMPerfRecord) TableName() string {
	return "vm_perf"
}

//Loadmon Rest API

type LoadmonRequest struct {
	Code int         `json:"Code"` //код запроса
	Data interface{} `json:"Data"` //данные для запроса
}

type LoadmonResponse struct {
	Code   int         `json:"Code"` //код запроса
	Status int         `json:"Status"` //код ответа 1 все хорошо, иначе код ошибки 0 - нет запрошенных данных, 2-ошибка в структуре запроса, 3-недопустимый код запроса
	Count  int         `json:"Count"` //количество записей в возврате
	Data   interface{} `json:"Data"` //данные для запроса
}

// jira CMDB https://jira.bcs.ru/rest/assets/latest/aql/objects?qlQuery=objectTypeId=21&includeAttributes=true&page=1
type ItSystemRecord struct {
	ID                           int    `form:"ID" json:"id"`
	ObjectKey                    string `form:"ObjectKey" json:"objectKey"`
	Label                        string `form:"Label" json:"label"`
	Name                         string `form:"Name" json:"name"`
	Key                          string `form:"Key" json:"key"` //0
	FullName                     string `form:"FullName" json:"fullName"` //1
	ShortName                    string `form:"ShortName" json:"shortName"` //2
	LevelCriticality             string `form:"LevelCriticality" json:"levelCriticality"` //3 "Критически важные (Vital),Критические для бизнеса (Critical),Важные для бизнеса (Secondary) // TODO: comment truncated in screenshot
	Description                  string `form:"Description" json:"description"` //4
	Status                       string `form:"Status" json:"status"` //5 "Планируется, Проектируется, Разрабатывается, Тестируется, Выведено на прод, Заморожено, В архиве // TODO: comment truncated in screenshot
	IsMonitored                  bool   `form:"IsMonitored" json:"isMonitored"` //6
	IsBackuped                   bool   `form:"IsBackuped" json:"isBackuped"` //7
	SystemType                   string `form:"SystemType" json:"systemType"` //8 "Бизнес, Техническая"
	LicensingType                string `form:"LicensingType" json:"licensingType"` //9 "Подписка, Единоразовая, Бесплатная"
	SolutionType                 string `form:"SolutionType" json:"solutionType"` //10
	DeploymentType               string `form:"DeploymentType" json:"deploymentType"` //11 "On-Cloud, On-Premise"
	ITSystemUsers                string `form:"ITSystemUsers" json:"itSystemUsers"` //12
	Documentation                string `form:"Documentation" json:"documentation"` //13
	CFO                          string `form:"CFO" json:"cfo"` //14
	Commissioning                string `form:"Commissioning" json:"commissioning"` //15
	Decommissioning              string `form:"Decommissioning" json:"decommissioning"` //16
	URL                          string `form:"URL" json:"url"` //17
	DeveloperTeam                string `form:"DeveloperTeam" json:"developerTeam"` //18
	SupportTeam                  string `form:"SupportTeam" json:"supportTeam"` //19
	JiraProject                  string `form:"JiraProject" json:"jiraProject"` //20
	AccountingName               string `form:"AccountingName" json:"accountingName"` //21 object key
	ReferencedApplication        string `form:"ReferencedApplication" json:"referencedApplication"` //22
	ItiliumItService             string `form:"ItiliumItService" json:"itiliumItService"` //23
	DataCenter                   string `form:"DataCenter" json:"dataCenter"` //24
	ReferencedDatabase           string `form:"ReferencedDatabase" json:"referencedDatabase"` //25
	ProjectGitlab                string `form:"ProjectGitlab" json:"projectGitlab"` //26
	LocationType                 string `form:"LocationType" json:"locationType"` //27
	IPAdressesV4                 string `form:"IPAdressesV4" json:"ipAdressesV4"` //28
	IPAdressesV6                 string `form:"IPAdressesV6" json:"ipAdressesV6"` //29
	DescriptionObject            string `form:"DescriptionObject" json:"descriptionObject"` //30
	AutonomousSystemNumber       string `form:"AutonomousSystemNumber" json:"autonomousSystemNumber"` //31
	DomainNameElectronicResource string `form:"DomainNameElectronicResource" json:"domainNameElectronicResource"` //32
	UIRI                         string `form:"UIRI" json:"uiri"` //33
	PhoneNumbers                 string `form:"PhoneNumbers" json:"phoneNumbers"` //34
	OperatorId                   string `form:"OperatorId" json:"operatorId"` //35
	FullCorporateName            string `form:"FullCorporateName" json:"fullCorporateName"` //36
	TaxpayerIdNumber             string `form:"TaxpayerIdNumber" json:"taxpayerIdNumber"` //37
	LinkService                  string `form:"LinkService" json:"linkService"` //40
	LactConfirmDate              string `form:"LactConfirmDate" json:"lactConfirmDate"` //41
	ItiliumITServiceNew          string `form:"ItiliumITServiceNew" json:"itiliumITServiceNew"` //42
	URLNew                       string `form:"URLNew" json:"urlnew"` //43
	SDNotificationDate           string `form:"SDNotificationDate" json:"sdnotificationDate"` //44
	BizTalkID                    string `form:"BizTalkID" json:"bizTalkID"` //45
	InboundIntegration           string `form:"InboundIntegration" json:"inboundIntegration"` //46
	OutboundIntegration          string `form:"OutboundIntegration" json:"outboundIntegration"` //47
	BusinessColumn               string `form:"BusinessColumn" json:"businessColumn"` //object key
	BusinessOwner                string `form:"BusinessOwner" json:"businessOwner"` //
	ITOwner                      string `form:"ITOwner" json:"itOwner"` //
	Vendor                       string `form:"Vendor" json:"Vendor"` //
	ProductOwner                 string `form:"ProductOwner" json:"productOwner"` //
	AnalystMethodologist         string `form:"AnalystMethodologist" json:"analystMethodologist"` //
	Created                      time.Time `form:"Created" json:"created"` //
	Updated                      time.Time `form:"Updated" json:"updated"` //
	Timestamp                    int64   `form:"Timestamp" json:"timestamp"` //
}

type InsightVm struct {
	Id                int64     `form:"Id"`
	Time              time.Time `orm:"null" form:"Time"` //время добавления записи (можно определить новые за период)
	Vcenter           string    `orm:"null" form:"Vcenter"`
	Datacenter        string    `orm:"null" form:"Datacenter"`
	Cluster           string    `orm:"null" form:"Cluster"`
	Name              string    `orm:"null" form:"Name"`
	HostName          string    `orm:"null" form:"HostName"`
	NumCPU            int32     `orm:"null" form:"NumCPU"`
	NumCores          int32     `orm:"null" form:"NumCores"`
	MemorySizeGB      float64   `orm:"null" form:"MemorySizeGB"`
	Os                string    `orm:"null" form:"Os"` //GuestFullName
	IPAddress         string    `orm:"null" form:"IPAddress"`
	Network           string    `orm:"null" form:"Network"`
	DiskCommittedGB   float64   `orm:"null" form:"DiskCommittedGB"`
	DiskUncommittedGB float64   `orm:"null" form:"DiskUncommittedGB"`
	PowerState        bool      `orm:"null" form:"PowerState"`
	ExpirationDate    time.Time `orm:"null" form:"ExpirationDate"`
	VMRole            string    `orm:"null" form:"VMRole"`
	Production        int       `form:"Production"` //наследуется из справочника кластеров
	WorkStation       bool      `form:"WorkStation"`
}

type InsightVmPerf struct {
	Id              int64     `form:"Id"`
	Time            time.Time `orm:"null" form:"Time"`
	Vcenter         string    `orm:"null" form:"Vcenter"`
	Datacenter      string    `orm:"null" form:"Datacenter"`
	Cluster         string    `orm:"null" form:"Cluster"`
	Name            string    `orm:"null" form:"Name"`
	HostName        string    `orm:"null" form:"HostName"`
	CpuUsage        float64   `orm:"null" form:"CpuUsage"`
	MemUsage        float64   `orm:"null" form:"MemUsage"`
	DiskUsage       float64   `orm:"null" form:"DiskUsage"`
	DiskCommittedGB float64   `orm:"null" form:"DiskCommittedGB"`
	CapacityGb      float64   `orm:"null" form:"CapacityGb"`
	FreeSpaceGb     float64   `orm:"null" form:"FreeSpaceGb"`
	PercentUsed     float64   `orm:"null" form:"PercentUsed"`
}

func (a *InsightVm) TableName() string {
	return "insight_vm"
}

func (u *InsightVm) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name"},
	}
}

func (a *InsightVmPerf) TableName() string {
	return "insight_vm_perf"
}

func (u *InsightVmPerf) TableIndex() [][]string {
	return [][]string{
		[]string{"Id"},
		[]string{"Time"},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name"},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "Time"},
	}
}

