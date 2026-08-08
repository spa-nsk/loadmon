package vspheremetrics

import (
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"fmt"
	"strings"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

var TimeOut int32

func Init() {
	atomic.StoreInt32(&TimeOut, 10000)
}

type LockFlag struct {
	lockFlag int32
}

var (
	nilObject      string = "Указатель nil"
	lockCountLimit string = "Истекло время ожидания блокировки"
)

func (obj *LockFlag) LockObject() error {
	if obj == nil {
		return errors.New(nilObject)
	}
	for i := int32(0); !atomic.CompareAndSwapInt32(&obj.lockFlag, 0, 1); i++ { //цикл пока объект не разблокируется
		if i >= atomic.LoadInt32(&TimeOut) {
			return errors.New(lockCountLimit)
		}
		runtime.Gosched()
	}
	return nil
}

func (obj *LockFlag) UnLockObject() error {
	if obj == nil {
		return errors.New(nilObject)
	}
	atomic.StoreInt32(&obj.lockFlag, 0)
	return nil
}

func (obj *LockFlag) IsLockObject() bool {
	if obj == nil {
		return false
	}
	if atomic.LoadInt32(&obj.lockFlag) == 1 {
		return true
	} else {
		return false
	}
}

type VCenter struct {
	LockFlag
	Hostname    string `form:"Hostname"`
	Username    string `form:"Username"`
	Password    string `form:"Password"`
	Connected   bool   `form:"Connected"`
	C           *govmomi.Client
	Ticker      *time.Ticker
	//Done      chan bool
	DoneFl      int32 `form:"DoneFl"`
	Wait        int   `form:"Wait"`
	CountThread int64 `form:"CountThread"`
}

type Datacenter struct {
	LockFlag
	Vcenter string `form:"Vcenter"`
	Name    string `form:"Name"`
}

type ClusterShort struct {
	LockFlag
	Vcenter    string `form:"Vcenter"`
	Datacenter string `form:"Datacenter"`
	Name       string `form:"Name"`
}

type VMInfrastructure struct {
	LockFlag
	CountThread    int64
	CountMetrics   uint64
	MetricsPerHour uint64
	PerfLock       int32
	Ticker         *time.Ticker
	Done           chan bool
	//Initialized bool //загружена информация по кластерам, хостам, виртуальным машинам
	VCenterMapList  cmap.ConcurrentMap[string, *VCenter]
	ClusterMapList  cmap.ConcurrentMap[string, *Cluster]
	VMMapList       cmap.ConcurrentMap[string, *VMPoint]
	VMIDMapList     cmap.ConcurrentMap[string, *VMPoint]
	VMUUIDMapList   cmap.ConcurrentMap[string, *VMPoint]
	HostMapList     cmap.ConcurrentMap[string, *HostInfo]
	HostBySNMapList cmap.ConcurrentMap[string, *HostInfo]
	DataStoreMapList cmap.ConcurrentMap[string, *Datastore]
	LUNMapListUUID  cmap.ConcurrentMap[string, *ScsiLun]
	TagsVM          cmap.ConcurrentMap[string, []tags.AttachedTags]
	TagsDC          cmap.ConcurrentMap[string, []tags.AttachedTags]
}

type Info struct {
	VCList      []VCenter
	ClusterList []*Cluster
	VMList      map[string]*VMPoint
	HostList    map[string]*HostInfo
	DSList      map[string]*Datastore
	LUNListUUID map[string]*ScsiLun
}

type ScsiLun struct {
	UID           string `form:"UID"`           //UID/WWN Lun ("60:06:01:60:2B:40:45:00:5D:AD:03:5B:49:F9:20:1A")
	UUID          string `form:"UUID"`          //Uuid ("0200010000600601602b4045005dad035b49f9201a565241494420")
	CanonicalName string `form:"CanonicalName"` //("naa.600601602b4045005dad035b49f9201a")
	LunType       string `form:"LunType"`       //"disk"
	Vcenter       string `form:"Vcenter"`
	Datacenter    string `form:"Datacenter"`
	Cluster       string `form:"Cluster"`
	DisplayName   string //"DGC Fibre Channel Over Ethernet Disk (naa.600601602b4045005dad03...+11 more"
}

type Metric struct {
	Instance string
	Value    float64
}

type VMPerf struct {
	LockFlag
	ID                     int64     `form:"ID" csv:"-"`
	Start                  time.Time `form:"Start" csv:"Time"`
	CPUUsage               float64   `form:"CPUUsage" csv:"CPU usage (%)" format:"%.2f"`
	//CPUUsageMap          []Metric  `form:"CPUUsageMap"`
	MemUsage               float64   `form:"MemUsage" csv:"MEM usage (%)" format:"%.2f"`
	DiskUsage              float64   `form:"DiskUsage" csv:"Disk usage (MB/s)" format:"%f"` //скорость диска kbps
	//DiskUsageMap         []Metric  `form:"DiskUsageMap"`
	DiskTotalReadLatency   float64   `form:"DiskTotalReadLatency" csv:"Disk read latency (ms)" format:"%f"`   //Среднее время чтения с виртуального диска ms
	DiskTotalWriteLatency  float64   `form:"DiskTotalWriteLatency" csv:"Disk write latency (ms)" format:"%f"` //Среднее время записи на виртуальный диск ms
	DiskWrite              float64   `form:"DiskWrite" csv:"Disk write (MB/s)" format:"%f"`                   //Скорость записи данных на виртуальный диск kbps
	DiskRead               float64   `form:"DiskRead" csv:"Disk read (MB/s)" format:"%f"`                     //Скорость чтения данных с виртуального диска kbps
	NetUsage               float64   `form:"NetUsage" csv:"Net usage (MB/s)" format:"%f"`                     //Использование сети (комбинированные скорости передачи и скорости приема) в течение интервала kbps
	NetReceived            float64   `form:"NetReceived" csv:"Net received (MB/s)" format:"%f"`                //Использование сети (скорость приема) в течение интервала kbps
	NetTransmitted         float64   `form:"NetTransmitted" csv:"Net transmitted (MB/s)" format:"%f"`          //Использование сети (скорость передачи) в течение интервала kbps
	//NetUsageMap          []Metric  `form:"NetUsageMap"`
	//NetReceivedMap       []Metric  `form:"NetReceivedMap"`     //приём по сети kbps
	//NetTransmittedMap    []Metric  `form:"NetTransmittedMap"` //передача по сети kbps
}

func (s *VMPerf) Init() {
	//s.CPUUsageMap = make([]Metric, 0)
	//s.DiskUsageMap = make([]Metric, 0)
	//s.NetUsageMap = make([]Metric, 0)
	//s.NetReceivedMap = make([]Metric, 0)
	//s.NetTransmittedMap = make([]Metric, 0)
}

func (s *VMPerf) Import(in *VMPerf) {
	//for _, val := range in.CPUUsageMap {
	//	s.CPUUsageMap = append(s.CPUUsageMap, val)
	//}
	//for _, val := range in.DiskUsageMap {
	//	s.DiskUsageMap = append(s.DiskUsageMap, val)
	//}
	//for _, val := range in.NetUsageMap {
	//	s.NetUsageMap = append(s.NetUsageMap, val)
	//}
	//for _, val := range in.NetReceivedMap {
	//	s.NetReceivedMap = append(s.NetReceivedMap, val)
	//}
	//for _, val := range in.NetTransmittedMap {
	//	s.NetTransmittedMap = append(s.NetTransmittedMap, val)
	//}
}

type HSPerf struct {
	LockFlag
	ID                    int64     `form:"ID"`
	Start                 time.Time `form:"Start"`
	CPUUsage              float64   `form:"CPUUsage"`
	//CPUUsageMap         []Metric  `form:"CPUUsageMap"`
	CPUUtilization        float64   `form:"CPUUtilization"`
	MemUsage              float64   `form:"MemUsage"`
	DiskUsage             float64   `form:"DiskUsage"`             //скорость диска kbps
	DiskTotalReadLatency  float64   `form:"DiskTotalReadLatency"`  //Среднее время чтения с виртуального диска
	//DiskTotalReadLatencyMap  []Metric `form:"DiskTotalReadLatencyMap"`  //Среднее время чтения с виртуального диска
	DiskTotalWriteLatency float64   `form:"DiskTotalWriteLatency"` //Среднее время записи на виртуальный диск
	//DiskTotalWriteLatencyMap []Metric `form:"DiskTotalWriteLatencyMap"` //Среднее время записи на виртуальный диск
	DiskWrite             float64   `form:"DiskWrite"` //Скорость записи данных на виртуальный диск kbps
	DiskRead              float64   `form:"DiskRead"`  //Скорость чтения данных с виртуального диска kbps
	//DiskWriteMap        []Metric  `form:"DiskWriteMap"`
	//DiskReadMap         []Metric  `form:"DiskReadMap"`
	NetUsage              float64   `form:"NetUsage"`      //Использование сети (комбинированные скорости передачи и скорости приема) в течение интервала kbps
	NetReceived           float64   `form:"NetReceived"`   //Использование сети (скорость приема) в течение интервала kbps
	NetTransmitted        float64   `form:"NetTransmitted"` //Использование сети (скорость передачи) в течение интервала kbps
	//NetUsageMap         []Metric  `form:"NetUsageMap"`
	//NetReceivedMap      []Metric  `form:"NetReceivedMap"`     //приём по сети kbps
	//NetTransmittedMap   []Metric  `form:"NetTransmittedMap"` //передача по сети kbps
}

func (s *HSPerf) Init() {
	//s.CPUUsageMap = make([]Metric, 0)
	//s.DiskTotalReadLatencyMap = make([]Metric, 0)
	//s.DiskTotalWriteLatencyMap = make([]Metric, 0)
	//s.DiskReadMap = make([]Metric, 0)
	//s.DiskWriteMap = make([]Metric, 0)
	//s.NetUsageMap = make([]Metric, 0)
	//s.NetReceivedMap = make([]Metric, 0)
	//s.NetTransmittedMap = make([]Metric, 0)
}

func (s *HSPerf) Import(in *HSPerf) {
	//for _, val := range in.CPUUsageMap {
	//	s.CPUUsageMap = append(s.CPUUsageMap, val)
	//}
	//for _, val := range in.DiskTotalReadLatencyMap {
	//	s.DiskTotalReadLatencyMap = append(s.DiskTotalReadLatencyMap, val)
	//}
	//for _, val := range in.DiskTotalWriteLatencyMap {
	//	s.DiskTotalWriteLatencyMap = append(s.DiskTotalWriteLatencyMap, val)
	//}
	//for _, val := range in.DiskReadMap {
	//	s.DiskReadMap = append(s.DiskReadMap, val)
	//}
	//for _, val := range in.DiskWriteMap {
	//	s.DiskWriteMap = append(s.DiskWriteMap, val)
	//}
	//
	//for _, val := range in.NetUsageMap {
	//	s.NetUsageMap = append(s.NetUsageMap, val)
	//}
	//for _, val := range in.NetReceivedMap {
	//	s.NetReceivedMap = append(s.NetReceivedMap, val)
	//}
	//for _, val := range in.NetTransmittedMap {
	//	s.NetTransmittedMap = append(s.NetTransmittedMap, val)
	//}
}

type CLPerf struct {
	LockFlag
	ID                     int64     `form:"ID"`
	Start                  time.Time `form:"Start"`
	CPUUsage               float64   `form:"CPUUsage"`
	CPUUtilization         float64   `form:"CPUUtilization"`
	MemUsage               float64   `form:"MemUsage"`
	DiskUsage              float64   `form:"DiskUsage"`             //скорость диска kbps
	DiskTotalReadLatency   float64   `form:"DiskTotalReadLatency"`  //Среднее время чтения с виртуального диска
	DiskTotalWriteLatency  float64   `form:"DiskTotalWriteLatency"` //Среднее время записи на виртуальный диск
	DiskWrite              float64   `form:"DiskWrite"`             //Скорость записи данных на виртуальный диск kbps
	DiskRead               float64   `form:"DiskRead"`              //Скорость чтения данных с виртуального диска kbps
	NetUsage               float64   `form:"NetUsage"`              //Использование сети (комбинированные скорости передачи и скорости приема) в течение интервала kbps
	NetReceived            float64   `form:"NetReceived"`           //Использование сети (скорость приема) в течение интервала kbps
	NetTransmitted         float64   `form:"NetTransmitted"`        //Использование сети (скорость передачи) в течение интервала kbps
}

func (s *CLPerf) Init() {
}

type GuestDiskInfo struct {
	DiskPath       string  //путь диска
	FileSystemType string  //ext4
	CapacityGb     float64 //емкость байт
	FreeSpaceGb    float64 //свободно байт
	PercentUsed    float64 //
}

type VMUsageOnDatastore struct {
	ID                     int64   `form:"ID"`
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

type VMFile struct {
	Type       string  `form:"Type"` // "config", "extendedConfig", "nvram", "snapshotList", "diskDescriptor", "diskExtent", "diskDescriptor", "diskExtent", "snapshotData", "swap", "uwswap", "log",
	Name       string  `form:"Name"`
	Size       float64 `form:"Size"` //в kb
	UniqueSize float64 `form:"UniqueSize"`
}

type VMPoint struct {
	LockFlag
	ID                               int64                `form:"ID"`
	Vcenter                          string               `form:"Vcenter"`
	Datacenter                       string               `form:"Datacenter"`
	Cluster                          string               `form:"Cluster"`
	ClusterID                        string               `form:"ClusterID"` //ClusterId = s-msk-vc-01.global.bcs:domain-c2638
	KeyName                          string               `form:"KeyName"`
	WorkStation                      bool                 `form:"WorkStation"` //true если это рабочая станция windows
	Template                         bool                 `form:"Template"`    //шаблон виртуальной машины
	TypeWork                         int                  `form:"TypeWork"`
	Name                             string               `form:"Name"`
	HostName                         string               `form:"HostName"`
	DomainName                       string               `form:"DomainName"`
	ToolsStatus                      string               `form:"ToolsStatus"` // статус установки vmware tools
	GuestFullName                    string               `form:"GuestFullName"`      //гостевая операционная система
	GuestDetailedData                string               `form:"GuestDetailedData"`  //подробности по гостевой система
	PathName                         string               `form:"PathName"`
	PowerState                       bool                 `form:"PowerState"` //true - VM включена
	UUID                             string               `form:"UUID"`
	GuestID                          string               `form:"GuestID"`
	Annotation                       string               `form:"Annotation"`
	CPUMhz                           float64              `form:"CPUMhz"`
	NumCPU                           int32                `form:"NumCPU"`
	NumCores                         int32                `form:"NumCores"`
	MemorySizeGB                     float64              `form:"MemorySizeGB"`
	Network                          string               `form:"Network"`
	IPAddress                        string               `form:"IPAddress"`
	MacAddress                       string               `form:"MacAddress"`
	NumEthernetCards                 int32                `form:"NumEthernetCards"`
	NumVirtualDisks                  int32                `form:"NumVirtualDisks"`
	DiskCommittedGB                  float64              `form:"DiskCommittedGB"`
	DiskUncommittedGB                float64              `form:"DiskUncommittedGB"`
	CapacityGb                       float64              `form:"CapacityGb"`
	FreeSpaceGb                      float64              `form:"FreeSpaceGb"`
	PercentUsed                      float64              `form:"PercentUsed"`
	TagName                          string               `form:"TagName"`        //Имя 1 тэга для виртуальных машин (Код для справочника)
	TagDescription                   string               `form:"TagDescription"` //Описание 1 тэга для виртуальных машин
	CPUUsageMaximum                  float64              `form:"CPUUsageMaximum"`
	CPUUsageMaximumTime              time.Time            `form:"CPUUsageMaximumTime"`
	CPUUsageAverage                  float64              `form:"CPUUsageAverage"`
	CPUUsageMinimum                  float64              `form:"CPUUsageMinimum"`
	CPUUsageMinimumTime              time.Time            `form:"CPUUsageMinimumTime"`
	MemUsageMaximum                  float64              `form:"MemUsageMaximum"`
	MemUsageMaximumTime              time.Time            `form:"MemUsageMaximumTime"`
	MemUsageAverage                  float64              `form:"MemUsageAverage"`
	MemUsageMinimum                  float64              `form:"MemUsageMinimum"`
	MemUsageMinimumTime              time.Time            `form:"MemUsageMinimumTime"`
	DiskUsageMaximum                 float64              `form:"DiskUsageMaximum"`
	DiskUsageMaximumTime             time.Time            `form:"DiskUsageMaximumTime"`
	DiskUsageAverage                 float64              `form:"DiskUsageAverage"`
	DiskUsageMinimum                 float64              `form:"DiskUsageMinimum"`
	DiskUsageMinimumTime             time.Time            `form:"DiskUsageMinimumTime"`
	DiskTotalReadLatencyMaximum      float64              `form:"DiskTotalReadLatencyMaximum"`
	DiskTotalReadLatencyMaximumTime  time.Time            `form:"DiskTotalReadLatencyMaximumTime"`
	DiskTotalReadLatencyAverage      float64              `form:"DiskTotalReadLatencyAverage"`
	DiskTotalReadLatencyMinimum      float64              `form:"DiskTotalReadLatencyMinimum"`
	DiskTotalReadLatencyMinimumTime  time.Time            `form:"DiskTotalReadLatencyMinimumTime"`
	DiskTotalWriteLatencyMaximum     float64              `form:"DiskTotalWriteLatencyMaximum"`
	DiskTotalWriteLatencyMaximumTime time.Time            `form:"DiskTotalWriteLatencyMaximumTime"`
	DiskTotalWriteLatencyAverage     float64              `form:"DiskTotalWriteLatencyAverage"`
	DiskTotalWriteLatencyMinimum     float64              `form:"DiskTotalWriteLatencyMinimum"`
	DiskTotalWriteLatencyMinimumTime time.Time            `form:"DiskTotalWriteLatencyMinimumTime"`
	DiskWriteMaximum                 float64              `form:"DiskWriteMaximum"`
	DiskWriteMaximumTime             time.Time            `form:"DiskWriteMaximumTime"`
	DiskWriteAverage                 float64              `form:"DiskWriteAverage"`
	DiskWriteMinimum                 float64              `form:"DiskWriteMinimum"`
	DiskWriteMinimumTime             time.Time            `form:"DiskWriteMinimumTime"`
	DiskReadMaximum                  float64              `form:"DiskReadMaximum"`
	DiskReadMaximumTime              time.Time            `form:"DiskReadMaximumTime"`
	DiskReadAverage                  float64              `form:"DiskReadAverage"`
	DiskReadMinimum                  float64              `form:"DiskReadMinimum"`
	DiskReadMinimumTime              time.Time            `form:"DiskReadMinimumTime"`
	NetUsageMaximum                  float64              `form:"NetUsageMaximum"`
	NetUsageMaximumTime              time.Time            `form:"NetUsageMaximumTime"`
	NetUsageAverage                  float64              `form:"NetUsageAverage"`
	NetUsageMinimum                  float64              `form:"NetUsageMinimum"`
	NetUsageMinimumTime              time.Time            `form:"NetUsageMinimumTime"`
	CPUBilling                       float64              `form:"CPUBilling"`  // в мегагерц часах
	MemBilling                       float64              `form:"MemBilling"`  // в мегабайт часах
	DiskBilling                      float64              `form:"DiskBilling"` // по занимаемому месту в гигабайт часах
	NetBilling                       float64              `form:"NetBilling"`  // в мегабайт часах
	Disk                             []GuestDiskInfo      `form:"Disk"`
	Files                            []VMFile             `form:"Files"` //файлы виртуальной машины config, snapshot, log ...
	UsageDatastore                   []VMUsageOnDatastore `form:"UsageDatastore"`
	Perf                             []*VMPerf            `form:"Perf"` //14.08.2024
	PerfCount                        int64                `form:"PerfCount"`
}

type Datastore struct {
	LockFlag
	ID          int64                          `form:"ID"`
	Type        string                         `form:"Type"`
	UID         string                         `form:"UID"` //уникальный идентификатор LUN
	StorageName string                         `form:"StorageName"`
	CapacityGb  float64                        `form:"CapacityGb"`
	FreeSpaceGb float64                        `form:"FreeSpaceGb"`
	PercentUsed float64                        `form:"PercentUsed"`
	VM          []*VMPoint                     `form:"VM"`
	VMCount     int64                          `form:"VMCount"`
	vmnames     []types.ManagedObjectReference
}

type Cluster struct {
	LockFlag
	ID                                int64        `form:"ID"`
	Date                              time.Time    `form:"Date"`
	Vcenter                           string       `form:"Vcenter"`
	Datacenter                        string       `form:"Datacenter"`
	HostFlag                          bool         `form:"HostFlag"` //признак отдельного хоста true имитация структур кластера без внесения в справочник
	TypeWork                          int          `form:"TypeWork"` //0 - не указано 1 - продуктовая среда 2- внутренний тест 3 - внешний тест
	Name                              string       `form:"Name"`
	ClusterID                         string       `form:"ClusterID"` //ClusterId = s-msk-vc-01.global.bcs:domain-c2638
	CPUMhzAvg                         int32        `form:"CPUMhzAvg"`
	AllNumCPUPkgs                     int16        `form:"AllNumCPUPkgs"`
	AllNumCPUCores                    int16        `form:"AllNumCPUCores"`
	AllNumCPUThreads                  int16        `form:"AllNumCPUThreads"`
	AllMemorySizeGB                   float64      `form:"AllMemorySizeGB"`
	AllCapacityGb                     float64      `form:"AllCapacityGb"`
	AllFreeSpaceGb                    float64      `form:"AllFreeSpaceGb"`
	AllPercentUsed                    float64      `form:"AllPercentUsed"`
	AllocatedCoreCount                int          `form:"AllocatedCoreCount"`     //выделено ядер для виртуальных машин
	AllocatedMemorySizeGb             int          `form:"AllocatedMemorySizeGb"`  //выделено оперативной памяти для виртуальных машин
	CoreTransferRate                  float64      `form:"CoreTransferRate"`       //коэффициент перевыдачи процессорных ресурсов
	MemoryTransferRate                float64      `form:"MemoryTransferRate"`     //коэффициент перевыдачи процессорных ресурсов
	CPUUsageMaximum                   float64      `form:"CPUUsageMaximum"`
	CPUUsageMaximumTime               time.Time    `form:"CPUUsageMaximumTime"`
	CPUUsageAverage                   float64      `form:"CPUUsageAverage"`
	CPUUsageMinimum                   float64      `form:"CPUUsageMinimum"`
	CPUUsageMinimumTime               time.Time    `form:"CPUUsageMinimumTime"`
	MemUsageMaximum                   float64      `form:"MemUsageMaximum"`
	MemUsageMaximumTime               time.Time    `form:"MemUsageMaximumTime"`
	MemUsageAverage                   float64      `form:"MemUsageAverage"`
	MemUsageMinimum                   float64      `form:"MemUsageMinimum"`
	MemUsageMinimumTime               time.Time    `form:"MemUsageMinimumTime"`
	DiskUsageMaximum                  float64      `form:"DiskUsageMaximum"`
	DiskUsageMaximumTime              time.Time    `form:"DiskUsageMaximumTime"`
	DiskUsageAverage                  float64      `form:"DiskUsageAverage"`
	DiskUsageMinimum                  float64      `form:"DiskUsageMinimum"`
	DiskUsageMinimumTime              time.Time    `form:"DiskUsageMinimumTime"`
	DiskTotalReadLatencyMaximum       float64      `form:"DiskTotalReadLatencyMaximum"`
	DiskTotalReadLatencyMaximumTime   time.Time    `form:"DiskTotalReadLatencyMaximumTime"`
	DiskTotalReadLatencyAverage       float64      `form:"DiskTotalReadLatencyAverage"`
	DiskTotalReadLatencyMinimum       float64      `form:"DiskTotalReadLatencyMinimum"`
	DiskTotalReadLatencyMinimumTime   time.Time    `form:"DiskTotalReadLatencyMinimumTime"`
	DiskTotalWriteLatencyMaximum      float64      `form:"DiskTotalWriteLatencyMaximum"`
	DiskTotalWriteLatencyMaximumTime  time.Time    `form:"DiskTotalWriteLatencyMaximumTime"`
	DiskTotalWriteLatencyAverage      float64      `form:"DiskTotalWriteLatencyAverage"`
	DiskTotalWriteLatencyMinimum      float64      `form:"DiskTotalWriteLatencyMinimum"`
	DiskTotalWriteLatencyMinimumTime  time.Time    `form:"DiskTotalWriteLatencyMinimumTime"`
	DiskWriteMaximum                  float64      `form:"DiskWriteMaximum"`
	DiskWriteMaximumTime              time.Time    `form:"DiskWriteMaximumTime"`
	DiskWriteAverage                  float64      `form:"DiskWriteAverage"`
	DiskWriteMinimum                  float64      `form:"DiskWriteMinimum"`
	DiskWriteMinimumTime              time.Time    `form:"DiskWriteMinimumTime"`
	DiskReadMaximum                   float64      `form:"DiskReadMaximum"`
	DiskReadMaximumTime               time.Time    `form:"DiskReadMaximumTime"`
	DiskReadAverage                   float64      `form:"DiskReadAverage"`
	DiskReadMinimum                   float64      `form:"DiskReadMinimum"`
	DiskReadMinimumTime               time.Time    `form:"DiskReadMinimumTime"`
	NetUsageMaximum                   float64      `form:"NetUsageMaximum"`
	NetUsageMaximumTime               time.Time    `form:"NetUsageMaximumTime"`
	NetUsageAverage                   float64      `form:"NetUsageAverage"`
	NetUsageMinimum                   float64      `form:"NetUsageMinimum"`
	NetUsageMinimumTime               time.Time    `form:"NetUsageMinimumTime"`
	CPUBilling                        float64      `form:"CPUBilling"`  // в мегагерц часах
	DiskBilling                       float64      `form:"DiskBilling"` // по занимаемому месту в гигабайт часах
	MemBilling                        float64      `form:"MemBilling"`  // в мегабайт часах
	NetBilling                        float64      `form:"NetBilling"`  // в мегабайт часах
	Hosts                             []*HostInfo  `form:"Hosts"`
	HostCount                         int64        `form:"HostCount"`
	VM                                []*VMPoint   `form:"VM"`
	VMCount                           int64        `form:"VMCount"`
	Storage                           []*Datastore `form:"Storage"`
	StorageCount                      int64        `form:"StorageCount"`
	Perf                              []*CLPerf    `form:"Perf"`
	PerfCount                         int64        `form:"PerfCount"`
}

type ClusterListRecord struct {
	LockFlag
	ID         int64  `form:"ID"`
	Vcenter    string `form:"Vcenter"`
	Datacenter string `form:"Datacenter"`
	Name       string `form:"Name"`
}

type HostInfo struct {
	LockFlag
	ID                                int64            `form:"ID"`
	Vcenter                           string           `form:"Vcenter"`
	Datacenter                        string           `form:"Datacenter"`
	Cluster                           string           `form:"Cluster"`
	ClusterID                         string           `form:"ClusterID"` //ClusterId = s-msk-vc-01.global.bcs:domain-c2638
	Name                              string           `form:"Name"`
	KeyName                           string           `form:"KeyName"`
	Vendor                            string           `form:"Vendor"` //HostSystemInfo
	Model                             string           `form:"Model"`
	SerialNumber                      string           `form:"SerialNumber"`
	VirtualSerialNumber               string           `form:"VirtualSerialNumber"`
	EnclosureSerialNumber             string           `form:"EnclosureSerialNumber"`
	Product                           types.AboutInfo  `form:"Product"`         //подробная информация о esxi
	ProductFullName                   string           `form:"ProductFullName"` //строка с наименование esxi с версией и редакцией
	UUID                              string           `form:"UUID"`
	PowerState                        bool             `form:"PowerState"` //true host включен
	InMaintenanceMode                 bool             `form:"InMaintenanceMode"`
	CPUModel                          string           `form:"CPUModel"`
	CPUMhz                            int32            `form:"CPUMhz"`
	NumCPUPkgs                        int16            `form:"NumCPUPkgs"`
	NumCPUCores                       int16            `form:"NumCPUCores"`
	NumCPUThreads                     int16            `form:"NumCPUThreads"`
	NumNics                           int32            `form:"NumNics"`
	IPAddress                         string           `form:"IPAddress"`   //IP v4 адреса через разделитель ;
	MacAddress                        string           `form:"MacAddress"`  //Mac адреса через разделитель ;
	NumHBAs                           int32            `form:"NumHBAs"`
	MemorySizeGB                      float64          `form:"MemorySizeGB"`
	HostMaxVirtualDiskCapacity        float64          `form:"HostMaxVirtualDiskCapacity"`
	TagName                           string           `form:"TagName"`        //Имя 1 тэга для хоста (Код для справочника)
	TagDescription                    string           `form:"TagDescription"` //Описание 1 тэга для хоста
	CPUUsageMaximum                   float64          `form:"CPUUsageMaximum"`
	CPUUsageMaximumTime               time.Time `form:"CPUUsageMaximumTime"`
	CPUUsageAverage                   float64   `form:"CPUUsageAverage"`
	CPUUsageMinimum                   float64   `form:"CPUUsageMinimum"`
	CPUUsageMinimumTime               time.Time `form:"CPUUsageMinimumTime"`
	MemUsageMaximum                   float64   `form:"MemUsageMaximum"`
	MemUsageMaximumTime               time.Time `form:"MemUsageMaximumTime"`
	MemUsageAverage                   float64   `form:"MemUsageAverage"`
	MemUsageMinimum                   float64   `form:"MemUsageMinimum"`
	MemUsageMinimumTime               time.Time `form:"MemUsageMinimumTime"`
	DiskUsageMaximum                  float64   `form:"DiskUsageMaximum"`
	DiskUsageMaximumTime              time.Time `form:"DiskUsageMaximumTime"`
	DiskUsageAverage                  float64   `form:"DiskUsageAverage"`
	DiskUsageMinimum                  float64   `form:"DiskUsageMinimum"`
	DiskUsageMinimumTime              time.Time `form:"DiskUsageMinimumTime"`
	DiskTotalReadLatencyMaximum       float64   `form:"DiskTotalReadLatencyMaximum"`
	DiskTotalReadLatencyMaximumTime   time.Time `form:"DiskTotalReadLatencyMaximumTime"`
	DiskTotalReadLatencyAverage       float64   `form:"DiskTotalReadLatencyAverage"`
	DiskTotalReadLatencyMinimum       float64   `form:"DiskTotalReadLatencyMinimum"`
	DiskTotalReadLatencyMinimumTime   time.Time `form:"DiskTotalReadLatencyMinimumTime"`
	DiskTotalWriteLatencyMaximum      float64   `form:"DiskTotalWriteLatencyMaximum"`
	DiskTotalWriteLatencyMaximumTime  time.Time `form:"DiskTotalWriteLatencyMaximumTime"`
	DiskTotalWriteLatencyAverage      float64   `form:"DiskTotalWriteLatencyAverage"`
	DiskTotalWriteLatencyMinimum      float64   `form:"DiskTotalWriteLatencyMinimum"`
	DiskTotalWriteLatencyMinimumTime  time.Time `form:"DiskTotalWriteLatencyMinimumTime"`
	DiskWriteMaximum                  float64   `form:"DiskWriteMaximum"`
	DiskWriteMaximumTime              time.Time `form:"DiskWriteMaximumTime"`
	DiskWriteAverage                  float64   `form:"DiskWriteAverage"`
	DiskWriteMinimum                  float64   `form:"DiskWriteMinimum"`
	DiskWriteMinimumTime              time.Time `form:"DiskWriteMinimumTime"`
	DiskReadMaximum                   float64   `form:"DiskReadMaximum"`
	DiskReadMaximumTime               time.Time `form:"DiskReadMaximumTime"`
	DiskReadAverage                   float64   `form:"DiskReadAverage"`
	DiskReadMinimum                   float64   `form:"DiskReadMinimum"`
	DiskReadMinimumTime               time.Time `form:"DiskReadMinimumTime"`
	NetUsageMaximum                   float64   `form:"NetUsageMaximum"`
	NetUsageMaximumTime               time.Time `form:"NetUsageMaximumTime"`
	NetUsageAverage                   float64   `form:"NetUsageAverage"`
	NetUsageMinimum                   float64   `form:"NetUsageMinimum"`
	NetUsageMinimumTime               time.Time `form:"NetUsageMinimumTime"`
	CPUBilling                        float64   `form:"CPUBilling"`  // в мегагерц часах
	MemBilling                        float64   `form:"MemBilling"`  // в мегабайт часах
	DiskBilling                       float64   `form:"DiskBilling"` // по занимаемому месту в гигабайт часах
	NetBilling                        float64   `form:"NetBilling"`  // в мегабайт часах
	Perf                              []*HSPerf `form:"Perf"` //14.08.2024
	PerfCount                         int64     `form:"PerfCount"`
}

// для сортировки
type ByName []mo.VirtualMachine

func (n ByName) Len() int           { return len(n) }
func (n ByName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ByName) Less(i, j int) bool { return n[i].Name < n[j].Name }

type VMListID []VMPoint

func (n VMListID) Len() int           { return len(n) }
func (n VMListID) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n VMListID) Less(i, j int) bool { return n[i].ID < n[j].ID }

type VMPerfCPUorMEM []VMPoint

func (n VMPerfCPUorMEM) Len() int      { return len(n) }
func (n VMPerfCPUorMEM) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n VMPerfCPUorMEM) Less(i, j int) bool {
	jMax := 0.0
	if n[j].CPUUsageMaximum > n[j].MemUsageMaximum {
		jMax = n[j].CPUUsageMaximum
	} else {
		jMax = n[j].MemUsageMaximum
	}

	iMax := 0.0
	if n[i].CPUUsageMaximum > n[i].MemUsageMaximum {
		iMax = n[i].CPUUsageMaximum
	} else {
		iMax = n[i].MemUsageMaximum
	}

	jAverMax := 0.0
	if n[j].CPUUsageAverage/float64(n[j].PerfCount) > n[j].MemUsageAverage/float64(n[j].PerfCount) {
		jAverMax = n[j].CPUUsageAverage / float64(n[j].PerfCount)
	} else {
		jAverMax = n[j].MemUsageAverage / float64(n[j].PerfCount)
	}

	iAverMax := 0.0
	if n[i].CPUUsageAverage/float64(n[i].PerfCount) > n[i].MemUsageAverage/float64(n[i].PerfCount) {
		iAverMax = n[i].CPUUsageAverage / float64(n[i].PerfCount)
	} else {
		iAverMax = n[i].MemUsageAverage / float64(n[i].PerfCount)
	}
	if iMax == jMax {
		return iAverMax > jAverMax
	}
	return iMax > jMax
}

type VMListCluster []VMPoint

func (n VMListCluster) Len() int      { return len(n) }
func (n VMListCluster) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n VMListCluster) Less(i, j int) bool {
	if strings.Compare(n[i].Cluster, n[j].Cluster) < 0 {
		return true
	}
	return false
}

type VMListName []VMPoint

func (n VMListName) Len() int      { return len(n) }
func (n VMListName) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n VMListName) Less(i, j int) bool {
	if strings.Compare(n[i].Name, n[j].Name) < 0 {
		return true
	}
	return false
}

type HSListID []HostInfo

func (n HSListID) Len() int           { return len(n) }
func (n HSListID) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n HSListID) Less(i, j int) bool { return n[i].ID < n[j].ID }

type HSListName []*HostInfo

func (n HSListName) Len() int           { return len(n) }
func (n HSListName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n HSListName) Less(i, j int) bool { return n[i].Name < n[j].Name }

type ClusterName []*Cluster

func (n ClusterName) Len() int           { return len(n) }
func (n ClusterName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n ClusterName) Less(i, j int) bool { return n[i].Name < n[j].Name }

type ClusterCPUPerf []*Cluster

func (n ClusterCPUPerf) Len() int      { return len(n) }
func (n ClusterCPUPerf) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n ClusterCPUPerf) Less(i, j int) bool {
	var summI, summJ float64
	for _, hs := range n[i].Hosts {
		summI = summI + float64(hs.CPUMhz)*float64(hs.NumCPUCores)
	}
	for _, hs := range n[j].Hosts {
		summJ = summJ + float64(hs.CPUMhz)*float64(hs.NumCPUCores)
	}
	return !(summI < summJ) //сортировка по убыванию
}

type ClusterDate []*Cluster

func (n ClusterDate) Len() int      { return len(n) }
func (n ClusterDate) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n ClusterDate) Less(i, j int) bool {
	return (n[i].Date.Before(n[j].Date)) //сортировка сортировка по возрастанию
}

type ClusterVcenter []*Cluster

func (n ClusterVcenter) Len() int      { return len(n) }
func (n ClusterVcenter) Swap(i, j int) { n[i], n[j] = n[j], n[i] }
func (n ClusterVcenter) Less(i, j int) bool {
	a := fmt.Sprintf("%s%s%s", n[i].Vcenter, n[i].Datacenter, n[i].Name)
	b := fmt.Sprintf("%s%s%s", n[j].Vcenter, n[j].Datacenter, n[j].Name)
	return (a < b) //сортировка сортировка по возрастанию
}
