// Package cmdb реализует функции CMDB (Configuration Management Database) — базы данных
// управления конфигурациями — в составе системы мониторинга loadmon. Пакет обеспечивает
// синхронизацию и хранение сведений об инфраструктуре: виртуальных машинах и кластерах
// vSphere, физических серверах и их аппаратной конфигурации (HP OneView, Lenovo, СХД
// Huawei/EMC), а также метрик их производительности. Функции пакета преобразуют данные,
// полученные из внешних систем мониторинга и управления оборудованием, в записи справочников
// CMDB (ServerSprRecords, ServerHardware, VMSprRecord, ClusterSprRecords и др.), поддерживают
// их актуальность, устраняют дубликаты и восстанавливают связи между записями, а также
// выгружают показатели производительности в Zabbix.
package cmdb

import (
	"fmt"
	"math"

	// "loadmon/filefunction"
	models "loadmon/models"
	_ "loadmon/routers"
	vsphere "loadmon/vspheremetrics"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// CheckHypervisorOS определяет, является ли операционная система сервера гипервизором ESXi.
func CheckHypervisorOS(srv *models.ServerSprRecords) bool {
	if srv.OS != "" {
		if strings.Contains(strings.ToUpper(srv.OS), "ESXI") {
			return true
		}
	}
	return false
}

// GetSNFromServerSprRecords возвращает серийный номер сервера: явно указанный либо, если он
// отсутствует, извлечённый из текстового поля Details по маркеру "Добавлен из".
func GetSNFromServerSprRecords(srv *models.ServerSprRecords) string {
	if srv.SerialNumber != "" {
		return srv.SerialNumber
	}
	s := strings.Split(srv.Details, "Добавлен из")
	if len(s) > 1 {
		return strings.TrimSpace(s[0])
	}
	return ""
}

// CopyVmToSprSrv переносит данные виртуальной машины vSphere (VMPoint) в новую запись
// справочника серверов ServerSprRecords, включая параметры ресурсов и признаки перегрузки.
func CopyVmToSprSrv(from vsphere.VMPoint, to *models.ServerSprRecords) {
	if to == nil {
		return
	}

	to.TagName = from.TagName
	to.TagDescription = from.TagDescription

	to.Type = 1
	to.Stat = 1
	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Cluster = from.Cluster
	to.ClusterId = from.ClusterID
	to.MacAddress = from.MacAddress

	if from.PowerState {
		to.PowerState = 1
		// (2026-09-04) ВМ реально найдена в vSphere (иначе эта функция не
		// вызывалась бы) и включена - форсируем Active=1, иначе после
		// возврата из "выведена из эксплуатации" (Active=2) могла остаться
		// недопустимая комбинация Stat=1/PowerState=1/Active=2. Для
		// выключенной ВМ Active намеренно не трогаем (допустимы 0/1/2).
		// 2026-09-22: audit-helper вместо прямого присвоения, источник -
		// models.UserSourceVMware (данные из vSphere).
		models.SetActiveWithAudit(&to.Active, &to.TimeActiveState, &to.ActivatorUser, &to.TimeDeActiveState, &to.DeActivatorUser, 1, models.UserSourceVMware)
	} else {
		to.PowerState = 2
	}

	to.OS = from.GuestFullName

	to.Name = from.Name
	//tmpVmName := filefunction.MakeShortNameSrv(from.Name)
	//tmpVmHostName := filefunction.MakeShortNameSrv(from.HostName)
	/*
		tmpVmName, _ := SplitNameFQDN(from.Name)
		tmpVmHostName, _ := SplitNameFQDN(from.HostName)

		if !strings.EqualFold(tmpVmName, tmpVmHostName) {
			to.NameFQDN = from.Name
		} else {
			to.NameFQDN = from.HostName
		}
	*/
	/*
		fqdn, ip, err := GetFQDNNameAndIP(from.HostName)
		if err == nil {
			to.NameFQDN = fqdn
			if from.IPAddress == "" {
				to.IPAddress = ip
			} else {
				to.IPAddress = from.IPAddress
			}
		} else {
			to.NameFQDN = from.HostName
			to.IPAddress = from.IPAddress
		}
	*/
	to.NameFQDN = from.HostName
	to.IPAddress = from.IPAddress

	to.Network = from.Network
	to.CPUCount = float64(from.NumCPU)
	to.CPULogical = float64(from.NumCores)
	to.MemSize = from.MemorySizeGB
	to.DiskSize = from.CapacityGb

	to.Service = ""
	to.Owner = ""
	to.Allocation = ""
	to.Anotation = ""
	to.Role = ""
	if to.Details == "" {
		to.Details = from.Annotation
	}
	to.Level = from.CPUUsageMaximum
	to.Warning = ""
	if from.CPUUsageMaximum > 75 {
		to.Warning = "CPU "
	}
	if from.MemUsageMaximum > to.Level {
		to.Level = from.MemUsageMaximum
	}
	if from.MemUsageMaximum > 75 {
		to.Warning = to.Warning + " RAM "
	}
}

// UpdateVmToSprSrv обновляет существующую запись справочника серверов ServerSprRecords
// текущими данными виртуальной машины vSphere (VMPoint).
func UpdateVmToSprSrv(from vsphere.VMPoint, to *models.ServerSprRecords) {
	if to == nil {
		return
	}

	to.TagName = from.TagName
	to.TagDescription = from.TagDescription

	to.Type = 1
	// (2026-09-04) Раньше Stat здесь не проставлялся вовсе - вызывающий код
	// (cmdb/updatesprvm.go:UpdatePowerStateForServers-путь через
	// UpdateVmToSprSrv) применяет эту функцию и к УЖЕ СУЩЕСТВУЮЩИМ записям,
	// в т.ч. ранее помеченным отсутствующими (Stat=0/Active=2). Функция
	// вызывается только когда ВМ реально найдена в vSphere (см.
	// updatesprvm.go), поэтому Stat=1 корректно проставлять безусловно -
	// иначе повторно обнаруженная ВМ так и оставалась со старым Stat=0,
	// хотя PowerState ниже уже обновлялся на актуальное значение.
	to.Stat = 1
	// (2026-09-04) Раньше эта функция (в отличие от CopyVmToSprSrv для НОВЫХ
	// записей) не трогала Vcenter/Datacenter/Cluster/ClusterId вовсе - для
	// уже существующей записи это не затирало их, но и не давало
	// исправиться, если поля ранее оказались пустыми/устаревшими (миграция
	// ВМ между кластерами, или баг в cmdb/reconcileserversprtype.go, из-за
	// которого эти поля обнулялись на каждом цикле сверки - см. комментарий
	// в reconcileserversprtype.go, сверка размещения). Приводим к
	// единообразию с CopyVmToSprSrv.
	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Cluster = from.Cluster
	to.ClusterId = from.ClusterID
	if from.PowerState {
		to.PowerState = 1
		// См. аналогичный комментарий в CopyVmToSprSrv - форсируем Active=1
		// при найденной и включённой ВМ, иначе Active не трогаем.
		// 2026-09-22: audit-helper, источник - models.UserSourceVMware.
		models.SetActiveWithAudit(&to.Active, &to.TimeActiveState, &to.ActivatorUser, &to.TimeDeActiveState, &to.DeActivatorUser, 1, models.UserSourceVMware)
	} else {
		to.PowerState = 2
	}

	to.OS = from.GuestFullName

	to.Name = from.Name
	to.NameFQDN = from.HostName

	//tmpVmName := filefunction.MakeShortNameSrv(from.Name)
	//tmpVmHostName := filefunction.MakeShortNameSrv(from.HostName)

	//if !strings.EqualFold(tmpVmName, tmpVmHostName) {
	//	to.NameFQDN = from.Name
	//} else {
	//	to.NameFQDN = from.HostName
	//}

	to.IPAddress = from.IPAddress
	to.Network = from.Network
	to.CPUCount = float64(from.NumCPU)
	to.CPULogical = float64(from.NumCores)
	to.MemSize = from.MemorySizeGB
	to.DiskSize = from.CapacityGb

	if to.Details == "" {
		to.Details = from.Annotation
	}

	to.Level = from.CPUUsageMaximum
	to.Warning = ""
	if from.CPUUsageMaximum > 75 {
		to.Warning = "CPU "
	}
	if from.MemUsageMaximum > to.Level {
		to.Level = from.MemUsageMaximum
	}
	if from.MemUsageMaximum > 75 {
		to.Warning = to.Warning + " RAM "
	}
}

// CopyHostToSprHW переносит аппаратные характеристики хоста ESXi (HostInfo) в запись
// справочника конфигураций оборудования ServerHardware.
func CopyHostToSprHW(from vsphere.HostInfo, to *models.ServerHardware) {
	if to == nil {
		return
	}
	if from.PowerState {
		to.PowerState = 1
	} else {
		to.PowerState = 2
	}
	to.Vendor = from.Vendor
	to.Model = from.Model
	to.SerialNum = from.SerialNumber
	to.CPUModel = from.CPUModel
	to.CPUCount = int(from.NumCPUPkgs)
	to.CorePerCPU = int(from.NumCPUCores) //проверить може это общее количество ядер
	to.CoreMHz = int(from.CPUMhz)
	to.MemoryMb = int(from.MemorySizeGB * 1024)
	to.Time_Mfy = time.Now().Local()
}

// CopyHostToSprSrv переносит данные хоста ESXi (HostInfo) в запись справочника серверов
// ServerSprRecords как запись типа "гипервизор".
func CopyHostToSprSrv(from vsphere.HostInfo, to *models.ServerSprRecords) {
	if to == nil {
		return
	}
	to.Type = 2
	if from.PowerState /*|| from.InMaintenanceMode*/ {
		to.PowerState = 1
		// (2026-09-04) CopyHostToSprSrv вызывается только для хостов,
		// реально найденных в vSphere (см. вызовы в savedata.go/
		// savehostrecord.go) - Stat=1 проставляется отдельно вызывающей
		// стороной. Раньше Active здесь не трогался вовсе - если запись
		// раньше была выведена из эксплуатации (Active=2), а хост опять
		// стал включён, получалась недопустимая комбинация Stat=1/
		// PowerState=1/Active=2. По правилу "найден и включён" -> Active=1
		// обязательно; для выключенного хоста Active намеренно не
		// трогаем (допустимы 0/1/2).
		// 2026-09-22: audit-helper, источник - models.UserSourceVMware.
		models.SetActiveWithAudit(&to.Active, &to.TimeActiveState, &to.ActivatorUser, &to.TimeDeActiveState, &to.DeActivatorUser, 1, models.UserSourceVMware)
	} else {
		to.PowerState = 2
	}

	to.OS = from.ProductFullName //копируем точную версию esxi

	//to.Name = filefunction.MakeShortNameSrv(from.Name)
	to.Name, _ = SplitNameFQDN(from.Name) //убираем домен
	to.NameFQDN = from.Name

	to.OS = from.ProductFullName

	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Cluster = from.Cluster
	to.ClusterId = from.ClusterID

	to.CPUCount = float64(from.NumCPUPkgs)
	to.CPULogical = float64(from.NumCPUCores)
	to.MemSize = from.MemorySizeGB
	to.DiskSize = from.HostMaxVirtualDiskCapacity
	to.IPAddress = from.IPAddress
	to.MacAddress = from.MacAddress
	to.Level = from.CPUUsageMaximum
	to.Warning = ""
	if from.CPUUsageMaximum > 75 {
		to.Warning = "CPU "
	}
	if from.MemUsageMaximum > to.Level {
		to.Level = from.MemUsageMaximum
	}
	if from.MemUsageMaximum > 75 {
		to.Warning = to.Warning + " RAM "
	}
	to.Time_Mfy = time.Now().Local()
}

// CopyVmToSprVm переносит полные данные виртуальной машины vSphere (VMPoint) в запись
// справочника виртуальных машин VMSprRecord.
func CopyVmToSprVm(from vsphere.VMPoint, to *models.VMSprRecord) {
	if to == nil {
		return
	}
	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Cluster = from.Cluster
	to.ClusterId = from.ClusterID
	to.TypeWork = from.TypeWork
	to.Vm_MacAddress = from.MacAddress
	to.Vm_Name = from.Name
	to.Vm_GuestFullName = from.GuestFullName
	to.KeyName = from.KeyName
	to.WorkStation = from.WorkStation
	to.Vm_GuestDetailedData = from.GuestDetailedData
	to.Vm_ToolsStatus = from.ToolsStatus
	to.Vm_DomainName = from.DomainName
	/*
		tmpVmName := filefunction.MakeShortNameSrv(from.Name)
		tmpVmHostName := filefunction.MakeShortNameSrv(from.HostName)

		if !strings.EqualFold(tmpVmName, tmpVmHostName) {
			to.Vm_HostName = from.Name
		} else {
			to.Vm_HostName = from.HostName
		}
	*/
	to.Vm_Name = from.Name
	to.Vm_HostName = from.HostName
	to.Vm_EsxHost = from.EsxHost // реальный ESXi-хост, см. protocol/vm.proto (2026-09-15)

	to.Vm_PathName = from.PathName
	to.Vm_PowerState = from.PowerState
	to.Vm_Uuid = from.UUID
	to.Vm_GuestId = from.GuestID
	to.Vm_Annotation = from.Annotation
	to.Vm_NumCPU = from.NumCPU
	to.Vm_NumCores = from.NumCores
	to.Vm_MemorySizeGB = from.MemorySizeGB
	to.Vm_IpAddress = from.IPAddress
	to.Vm_Network = from.Network
	to.Vm_NumEthernetCards = from.NumEthernetCards
	to.Vm_NumVirtualDisks = from.NumVirtualDisks
	to.Vm_DiskCommittedGB = from.DiskCommittedGB
	to.Vm_DiskUncommittedGB = from.DiskUncommittedGB
	to.Vm_CapacityGb = from.CapacityGb
	to.Vm_FreeSpaceGb = from.FreeSpaceGb
	to.Vm_PercentUsed = from.PercentUsed
	to.Cpu_usage_maximum = from.CPUUsageMaximum
	to.Mem_usage_maximum = from.MemUsageMaximum
	to.Disk_usage_maximum = from.DiskUsageMaximum
	to.Net_usage_maximum = from.NetUsageMaximum
}

// CopySprVmToVm выполняет обратное преобразование: переносит данные из записи справочника
// виртуальных машин VMSprRecord в структуру vsphere.VMPoint.
func CopySprVmToVm(from models.VMSprRecord, to *vsphere.VMPoint) {
	if to == nil {
		return
	}
	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Cluster = from.Cluster
	to.Name = from.Vm_Name
	to.WorkStation = from.WorkStation
	to.KeyName = from.KeyName

	to.GuestFullName = from.Vm_GuestFullName
	to.HostName = from.Vm_HostName
	to.EsxHost = from.Vm_EsxHost
	to.PathName = from.Vm_PathName
	to.PowerState = from.Vm_PowerState
	to.UUID = from.Vm_Uuid
	to.GuestID = from.Vm_GuestId
	to.Annotation = from.Vm_Annotation
	to.NumCPU = from.Vm_NumCPU
	to.NumCores = from.Vm_NumCores
	to.MemorySizeGB = from.Vm_MemorySizeGB
	to.IPAddress = from.Vm_IpAddress
	to.Network = from.Vm_Network
	to.NumEthernetCards = from.Vm_NumEthernetCards
	to.NumVirtualDisks = from.Vm_NumVirtualDisks
	to.DiskCommittedGB = from.Vm_DiskCommittedGB
	to.DiskUncommittedGB = from.Vm_DiskUncommittedGB
	to.CapacityGb = from.Vm_CapacityGb
	to.FreeSpaceGb = from.Vm_FreeSpaceGb
	to.PercentUsed = from.Vm_PercentUsed
	to.CPUUsageMaximum = from.Cpu_usage_maximum
	to.MemUsageMaximum = from.Mem_usage_maximum
	to.DiskUsageMaximum = from.Disk_usage_maximum
	to.NetUsageMaximum = from.Net_usage_maximum
}

func importClusterRecord(to *models.ClusterRecord, from *vsphere.Cluster) {
	if from == nil || to == nil {
		return
	}
	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Name = from.Name
	to.CpuMhzAvg = from.CPUMhzAvg
	to.AllNumCpuPkgs = from.AllNumCPUPkgs
	to.AllNumCpuCores = from.AllNumCPUCores
	to.AllNumCpuThreads = from.AllNumCPUThreads
	to.AllMemorySizeGB = from.AllMemorySizeGB
	to.AllCapacityGb = from.AllCapacityGb
	to.AllFreeSpaceGb = from.AllFreeSpaceGb
	to.AllPercentUsed = from.AllPercentUsed
	to.Cpu_usage_maximum = from.CPUUsageMaximum
	to.Cpu_usage_maximum_time = from.CPUUsageMaximumTime
	to.Cpu_usage_minimum = from.CPUUsageMinimum
	to.Cpu_usage_minimum_time = from.CPUUsageMinimumTime
	to.Mem_usage_maximum = from.MemUsageMaximum
	to.Mem_usage_maximum_time = from.MemUsageMaximumTime
	to.Mem_usage_minimum = from.MemUsageMinimum
	to.Mem_usage_minimum_time = from.MemUsageMinimumTime
	to.Disk_usage_maximum = from.DiskUsageMaximum
	to.Disk_usage_maximum_time = from.DiskUsageMaximumTime
	to.Disk_usage_minimum = from.DiskUsageMinimum
	to.Disk_usage_minimum_time = from.DiskUsageMinimumTime
	to.Disk_totalReadLatency_maximum = from.DiskTotalReadLatencyMaximum
	to.Disk_totalReadLatency_maximum_time = from.DiskTotalReadLatencyMaximumTime
	to.Disk_totalReadLatency_minimum = from.DiskTotalReadLatencyMinimum
	to.Disk_totalReadLatency_minimum_time = from.DiskTotalReadLatencyMinimumTime
	to.Disk_totalWriteLatency_maximum = from.DiskTotalWriteLatencyMaximum
	to.Disk_totalWriteLatency_maximum_time = from.DiskTotalWriteLatencyMaximumTime
	to.Disk_totalWriteLatency_minimum = from.DiskTotalWriteLatencyMinimum
	to.Disk_totalWriteLatency_minimum_time = from.DiskTotalWriteLatencyMinimumTime
	to.Disk_write_maximum = from.DiskWriteMaximum
	to.Disk_write_maximum_time = from.DiskWriteMaximumTime
	to.Disk_write_minimum = from.DiskWriteMinimum
	to.Disk_write_minimum_time = from.DiskWriteMinimumTime
	to.Disk_read_maximum = from.DiskReadMaximum
	to.Disk_read_maximum_time = from.DiskReadMaximumTime
	to.Disk_read_minimum = from.DiskReadMinimum
	to.Disk_read_minimum_time = from.DiskReadMinimumTime
	to.Net_usage_maximum = from.NetUsageMaximum
	to.Net_usage_maximum_time = from.NetUsageMaximumTime
	to.Net_usage_minimum = from.NetUsageMinimum
	to.Net_usage_minimum_time = from.NetUsageMinimumTime
	to.CpuBilling = from.CPUBilling
	to.DiskBilling = from.DiskBilling
	to.MemBilling = from.MemBilling
	to.NetBilling = from.NetBilling
	to.HostCount = from.HostCount
	to.VMCount = from.VMCount
	to.StorageCount = from.StorageCount
	to.Cpu_usage_average = from.CPUUsageAverage
	to.Mem_usage_average = from.MemUsageAverage
	to.Disk_usage_average = from.DiskUsageAverage
	to.Disk_totalReadLatency_average = from.DiskTotalReadLatencyAverage
	to.Disk_totalWriteLatency_average = from.DiskTotalWriteLatencyAverage
	to.Disk_write_average = from.DiskWriteAverage
	to.Disk_read_average = from.DiskReadAverage
	to.Net_usage_average = from.NetUsageAverage
}

// ImportClusterPerf переносит показатели производительности кластера (vsphere.CLPerf) в
// запись метрик ClusterPerf.
func ImportClusterPerf(to *models.ClusterPerf, from *vsphere.CLPerf) {
	if from == nil || to == nil {
		return
	}

	to.Start = from.Start
	to.Cpu_usage = from.CPUUsage
	to.Mem_usage = from.MemUsage
	to.Disk_usage = from.DiskUsage
	to.Disk_totalReadLatency = from.DiskTotalReadLatency
	to.Disk_totalWriteLatency = from.DiskTotalWriteLatency
	to.Disk_write = from.DiskWrite
	to.Disk_read = from.DiskRead
	to.Net_usage = from.NetUsage
	to.Net_received = from.NetReceived
	to.Net_transmitted = from.NetTransmitted
}

// ImportHostPerf переносит показатели производительности хоста (vsphere.HSPerf) в
// запись метрик HostPerf.
func ImportHostPerf(to *models.HostPerf, from *vsphere.HSPerf) {
	if from == nil || to == nil {
		return
	}

	to.Start = from.Start
	to.Cpu_usage = from.CPUUsage
	to.Mem_usage = from.MemUsage
	to.Disk_usage = from.DiskUsage
	to.Disk_totalReadLatency = from.DiskTotalReadLatency
	to.Disk_totalWriteLatency = from.DiskTotalWriteLatency
	to.Disk_write = from.DiskWrite
	to.Disk_read = from.DiskRead
	to.Net_usage = from.NetUsage
	to.Net_received = from.NetReceived
	to.Net_transmitted = from.NetTransmitted
}

// ImportVMPerf переносит показатели производительности виртуальной машины (vsphere.VMPerf)
// в запись метрик VMPerfRecord.
func ImportVMPerf(to *models.VMPerfRecord, from *vsphere.VMPerf) {
	if from == nil || to == nil {
		return
	}

	to.Start = from.Start
	to.Cpu_usage = from.CPUUsage
	to.Mem_usage = from.MemUsage
	to.Disk_usage = from.DiskUsage
	to.Disk_totalReadLatency = from.DiskTotalReadLatency
	to.Disk_totalWriteLatency = from.DiskTotalWriteLatency
	to.Disk_write = from.DiskWrite
	to.Disk_read = from.DiskRead
	to.Net_usage = from.NetUsage
	to.Net_received = from.NetReceived
	to.Net_transmitted = from.NetTransmitted
}

// блок для Игоря Титова
func importInsightVM(from *vsphere.VMPoint, to *models.InsightVm) {
	if from == nil || to == nil {
		return
	}
	to.WorkStation = from.WorkStation
	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Cluster = from.Cluster
	to.Name = from.Name
	to.HostName = from.HostName
	to.NumCPU = from.NumCPU
	to.NumCores = from.NumCores
	to.MemorySizeGB = from.MemorySizeGB
	to.DiskCommittedGB = from.DiskCommittedGB
	to.DiskUncommittedGB = from.DiskUncommittedGB
	to.Os = from.GuestFullName
	to.IPAddress = from.IPAddress
	to.Network = from.Network
	to.PowerState = from.PowerState
}

func importInsightVMPerf(from *vsphere.VMPerf, to *models.InsightVmPerf) {
	if from == nil || to == nil {
		return
	}
	to.Time = from.Start
	to.CpuUsage = from.CPUUsage
	to.MemUsage = from.MemUsage
	to.DiskUsage = from.DiskUsage
}

//окончание блока для Игоря Титова

func importVMRecord(from *vsphere.VMPoint, to *models.VMRecord) {

	if from == nil || to == nil {
		return
	}

	from.VMClearPerfomance()
	for _, perf := range from.Perf {
		vsphere.ReCalcVMPerfomance(from, *perf)
	}
	from.PerfCount = (int64)(len(from.Perf))
	vsphere.CalcVMAveragePerfomance(from)

	// Минимумы инициализируются "бесконечностью" (math.MaxFloat64, см.
	// vspheremetrics/perf_clear.go, loadvmmetrics.go, addfunction.go) перед
	// проходом по Perf, чтобы первое же реальное значение через math.Min его
	// перекрыло; если у ВМ не было ни одного сэмпла Perf за период, минимум
	// так и останется на этом "бесконечном" значении - здесь оно приводится
	// к 0, чтобы не улетело в БД. ИСПРАВЛЕНО 2026-09-01: раньше сравнение шло
	// с литералом 777777777.0 - устаревшим "sentinel"-значением, которое
	// нигде в коде больше не присваивалось (актуальный sentinel уже давно
	// math.MaxFloat64) - из-за этого сравнение никогда не срабатывало, и
	// пустые ВМ по факту улетали в БД с математически бессмысленным
	// "минимумом" ~1.8e308 вместо 0.
	if from.CPUUsageMinimum == math.MaxFloat64 {
		from.CPUUsageMinimum = 0.0
	}
	if from.MemUsageMinimum == math.MaxFloat64 {
		from.MemUsageMinimum = 0.0
	}
	if from.DiskUsageMinimum == math.MaxFloat64 {
		from.DiskUsageMinimum = 0.0
	}
	if from.DiskTotalReadLatencyMinimum == math.MaxFloat64 {
		from.DiskTotalReadLatencyMinimum = 0.0
	}
	if from.DiskTotalWriteLatencyMinimum == math.MaxFloat64 {
		from.DiskTotalWriteLatencyMinimum = 0.0
	}
	if from.DiskWriteMinimum == math.MaxFloat64 {
		from.DiskWriteMinimum = 0.0
	}
	if from.DiskReadMinimum == math.MaxFloat64 {
		from.DiskReadMinimum = 0.0
	}
	if from.NetUsageMinimum == math.MaxFloat64 {
		from.NetUsageMinimum = 0.0
	}

	to.Vm_NumCPU = from.NumCPU
	to.Vm_NumCores = from.NumCores
	to.Vm_MemorySizeGB = from.MemorySizeGB
	to.Vm_IpAddress = from.IPAddress
	to.Vm_Network = from.Network
	to.Vm_NumEthernetCards = from.NumEthernetCards
	to.Vm_NumVirtualDisks = from.NumVirtualDisks
	to.Vm_DiskCommittedGB = from.DiskCommittedGB
	to.Vm_DiskUncommittedGB = from.DiskUncommittedGB
	to.Vm_CapacityGb = from.CapacityGb
	to.Vm_FreeSpaceGb = from.FreeSpaceGb
	to.Vm_PercentUsed = from.PercentUsed
	to.Cpu_usage_maximum = from.CPUUsageMaximum
	to.Cpu_usage_maximum_time = from.CPUUsageMaximumTime
	to.Cpu_usage_average = from.CPUUsageAverage
	to.Cpu_usage_minimum = from.CPUUsageMinimum
	to.Cpu_usage_minimum_time = from.CPUUsageMinimumTime
	to.Mem_usage_maximum = from.MemUsageMaximum
	to.Mem_usage_maximum_time = from.MemUsageMaximumTime
	to.Mem_usage_average = from.MemUsageAverage
	to.Mem_usage_minimum = from.MemUsageMinimum
	to.Mem_usage_minimum_time = from.MemUsageMinimumTime
	to.Disk_usage_maximum = from.DiskUsageMaximum
	to.Disk_usage_maximum_time = from.DiskUsageMaximumTime
	to.Disk_usage_average = from.DiskUsageAverage
	to.Disk_usage_minimum = from.DiskUsageMinimum
	to.Disk_usage_minimum_time = from.DiskUsageMinimumTime
	to.Disk_totalReadLatency_maximum = from.DiskTotalReadLatencyMaximum
	to.Disk_totalReadLatency_maximum_time = from.DiskTotalReadLatencyMaximumTime
	to.Disk_totalReadLatency_average = from.DiskTotalReadLatencyAverage
	to.Disk_totalReadLatency_minimum = from.DiskTotalReadLatencyMinimum
	to.Disk_totalReadLatency_minimum_time = from.DiskTotalReadLatencyMinimumTime
	to.Disk_totalWriteLatency_maximum = from.DiskTotalWriteLatencyMaximum
	to.Disk_totalWriteLatency_maximum_time = from.DiskTotalWriteLatencyMaximumTime
	to.Disk_totalWriteLatency_average = from.DiskTotalWriteLatencyAverage
	to.Disk_totalWriteLatency_minimum = from.DiskTotalWriteLatencyMinimum
	to.Disk_totalWriteLatency_minimum_time = from.DiskTotalWriteLatencyMinimumTime
	to.Disk_write_maximum = from.DiskWriteMaximum
	to.Disk_write_maximum_time = from.DiskWriteMaximumTime
	to.Disk_write_average = from.DiskWriteAverage
	to.Disk_write_minimum = from.DiskWriteMinimum
	to.Disk_write_minimum_time = from.DiskWriteMinimumTime
	to.Disk_read_maximum = from.DiskReadMaximum
	to.Disk_read_maximum_time = from.DiskReadMaximumTime
	to.Disk_read_average = from.DiskReadAverage
	to.Disk_read_minimum = from.DiskReadMinimum
	to.Disk_read_minimum_time = from.DiskReadMinimumTime
	to.Net_usage_maximum = from.NetUsageMaximum
	to.Net_usage_maximum_time = from.NetUsageMaximumTime
	to.Net_usage_average = from.NetUsageAverage
	to.Net_usage_minimum = from.NetUsageMinimum
	to.Net_usage_minimum_time = from.NetUsageMinimumTime
	to.CpuBilling = from.CPUBilling
	to.MemBilling = from.MemBilling
	to.DiskBilling = from.DiskBilling
	to.NetBilling = from.NetBilling
}

func importHostRecord(from *vsphere.HostInfo, to *models.HostRecord) {
	if from == nil || to == nil {
		return
	}

	// Тот же приём и тот же баг, что и в importVMRecord выше (см. её
	// комментарий) - только раньше здесь сравнивалось с 7777777777.0 (10
	// семёрок вместо 9), тоже никогда не совпадавшим с реальным sentinel'ом
	// math.MaxFloat64, устанавливаемым в vspheremetrics/perf_clear.go и
	// perf_metrics.go. ИСПРАВЛЕНО 2026-09-01.
	if from.CPUUsageMinimum == math.MaxFloat64 {
		from.CPUUsageMinimum = 0.0
	}
	if from.MemUsageMinimum == math.MaxFloat64 {
		from.MemUsageMinimum = 0.0
	}
	if from.DiskUsageMinimum == math.MaxFloat64 {
		from.DiskUsageMinimum = 0.0
	}
	if from.DiskTotalReadLatencyMinimum == math.MaxFloat64 {
		from.DiskTotalReadLatencyMinimum = 0.0
	}
	if from.DiskTotalWriteLatencyMinimum == math.MaxFloat64 {
		from.DiskTotalWriteLatencyMinimum = 0.0
	}
	if from.DiskWriteMinimum == math.MaxFloat64 {
		from.DiskWriteMinimum = 0.0
	}
	if from.DiskReadMinimum == math.MaxFloat64 {
		from.DiskReadMinimum = 0.0
	}
	if from.NetUsageMinimum == math.MaxFloat64 {
		from.NetUsageMinimum = 0.0
	}

	to.Vcenter = from.Vcenter
	to.Datacenter = from.Datacenter
	to.Cluster = from.Cluster
	to.Name = from.Name
	to.Vendor = from.Vendor
	to.Model = from.Model
	to.CpuModel = from.CPUModel
	to.CpuMhz = from.CPUMhz
	to.NumCpuPkgs = from.NumCPUPkgs
	to.NumCpuCores = from.NumCPUCores
	to.NumCpuThreads = from.NumCPUThreads
	to.NumNics = from.NumNics
	to.IpAddress = from.IPAddress
	to.NumHBAs = from.NumHBAs
	to.MemorySizeGB = from.MemorySizeGB
	to.HostMaxVirtualDiskCapacity = from.HostMaxVirtualDiskCapacity
	to.Cpu_usage_maximum = from.CPUUsageMaximum
	to.Cpu_usage_maximum_time = from.CPUUsageMaximumTime
	to.Cpu_usage_minimum = from.CPUUsageMinimum
	to.Cpu_usage_minimum_time = from.CPUUsageMinimumTime
	to.Mem_usage_maximum = from.MemUsageMaximum
	to.Mem_usage_maximum_time = from.MemUsageMaximumTime
	to.Mem_usage_minimum = from.MemUsageMinimum
	to.Mem_usage_minimum_time = from.MemUsageMinimumTime
	to.Disk_usage_maximum = from.DiskUsageMaximum
	to.Disk_usage_maximum_time = from.DiskUsageMaximumTime
	to.Disk_usage_minimum = from.DiskUsageMinimum
	to.Disk_usage_minimum_time = from.DiskUsageMinimumTime
	to.Disk_totalReadLatency_maximum = from.DiskTotalReadLatencyMaximum
	to.Disk_totalReadLatency_maximum_time = from.DiskTotalReadLatencyMaximumTime
	to.Disk_totalReadLatency_minimum = from.DiskTotalReadLatencyMinimum
	to.Disk_totalReadLatency_minimum_time = from.DiskTotalReadLatencyMinimumTime
	to.Disk_totalWriteLatency_maximum = from.DiskTotalWriteLatencyMaximum
	to.Disk_totalWriteLatency_maximum_time = from.DiskTotalWriteLatencyMaximumTime
	to.Disk_totalWriteLatency_minimum = from.DiskTotalWriteLatencyMinimum
	to.Disk_totalWriteLatency_minimum_time = from.DiskTotalWriteLatencyMinimumTime
	to.Disk_write_maximum = from.DiskWriteMaximum
	to.Disk_write_maximum_time = from.DiskWriteMaximumTime
	to.Disk_write_minimum = from.DiskWriteMinimum
	to.Disk_write_minimum_time = from.DiskWriteMinimumTime
	to.Disk_read_maximum = from.DiskReadMaximum
	to.Disk_read_maximum_time = from.DiskReadMaximumTime
	to.Disk_read_minimum = from.DiskReadMinimum
	to.Disk_read_minimum_time = from.DiskReadMinimumTime
	to.Net_usage_maximum = from.NetUsageMaximum
	to.Net_usage_maximum_time = from.NetUsageMaximumTime
	to.Net_usage_minimum = from.NetUsageMinimum
	to.Net_usage_minimum_time = from.NetUsageMinimumTime
	to.CpuBilling = from.CPUBilling
	to.MemBilling = from.MemBilling
	to.DiskBilling = from.DiskBilling
	to.NetBilling = from.NetBilling
	to.Cpu_usage_average = from.CPUUsageAverage
	to.Mem_usage_average = from.MemUsageAverage
	to.Disk_usage_average = from.DiskUsageAverage
	to.Disk_totalReadLatency_average = from.DiskTotalReadLatencyAverage
	to.Disk_totalWriteLatency_average = from.DiskTotalWriteLatencyAverage
	to.Disk_write_average = from.DiskWriteAverage
	to.Disk_read_average = from.DiskReadAverage
	to.Net_usage_average = from.NetUsageAverage
}

// CopyReportRecToPerfZabbix преобразует запись отчёта по серверу (ReportServerRecord) в
// формат метрик для выгрузки производительности в Zabbix (ServerPerfZabbix).
func CopyReportRecToPerfZabbix(from *models.ReportServerRecord, to *models.ServerPerfZabbix) {
	if from == nil || to == nil {
		return
	}
	if from.Virtual {
		to.Type = 1
	} else {
		to.Type = 0
	}
	to.Date = from.Time
	to.NameFQDN = from.ServerName
	to.CPUCount = from.CPUCount
	to.CPULogical = from.CoreCount
	to.MemSize = from.MemSize
	to.Cpu_usage_average = from.CPULoadAvg
	to.Cpu_usage_maximum = from.CPULoadMax
	to.Cpu_usage_minimum = from.CPULoadMin
	to.Mem_usage_average = from.MemUseAvg
	to.Mem_usage_max = from.MemUseMax
	to.Mem_usage_minimum = from.MemUseMin
}

// RestoreSprData переносит ранее заполненные вручную атрибуты записи ServerSprRecords
// (владелец, назначение, аннотации, признаки типа и т.п.) из старой версии записи (from)
// в новую (to), если соответствующие поля в новой записи ещё не заполнены.
func RestoreSprData(to *models.ServerSprRecords, from models.ServerSprRecords) {
	if to == nil {
		return
	}
	to.Type = from.Type
	if to.TypeMicroservice == 0 {
		to.TypeMicroservice = from.TypeMicroservice
	}
	if to.RoleType == 0 {
		to.RoleType = from.RoleType
	}
	if to.TypeWork == 0 {
		to.TypeWork = from.TypeWork
	}
	if to.CriticalLevel == 0 {
		to.CriticalLevel = from.CriticalLevel
	}
	//VMSprId
	//Stat
	//PowerState
	//to.Active
	// 2026-09-22: исправлена опечатка - TimeDeActiveState копировался из
	// from.TimeActiveState (даты активации) вместо from.TimeDeActiveState
	// (дата деактивации), из-за чего RestoreSprData затирала дату
	// деактивации датой последней активации.
	to.TimeActiveState = from.TimeActiveState
	to.ActivatorUser = from.ActivatorUser
	to.TimeDeActiveState = from.TimeDeActiveState
	to.DeActivatorUser = from.DeActivatorUser
	//TimeCreate
	//CreatorUser
	to.BusinesPrivilage = from.BusinesPrivilage
	if to.NameInfo == "" {
		to.NameInfo = from.NameInfo
	}
	if to.IPAddress == "" {
		to.IPAddress = from.IPAddress
	}
	if to.IPSharedVIP == "" {
		to.IPSharedVIP = from.IPSharedVIP
	}
	if to.Network == "" {
		to.Network = from.Network
	}
	if to.NetworkName == "" {
		to.NetworkName = from.NetworkName
	}
	if to.Service == "" {
		to.Service = from.Service
	}
	if to.ServiceId == 0 {
		to.ServiceId = from.ServiceId
	}
	if to.OS == "" {
		to.OS = from.OS
	}
	if to.Component == "" {
		to.Component = from.Component
	}
	if to.Owner == "" {
		to.Owner = from.Owner
	}
	if to.Allocation == "" {
		to.Allocation = from.Allocation
	}
	if to.Anotation == "" {
		to.Anotation = from.Anotation
	}
	if to.Role == "" {
		to.Role = from.Role
	}
	if to.Details == "" {
		to.Details = from.Details
	}
	if to.SZHistory == "" {
		to.SZHistory = from.SZHistory
	}
	if to.Monitoring == 0 {
		to.Monitoring = from.Monitoring
		to.TimeActiveMonitoring = from.TimeActiveMonitoring
		to.MonitoringSZ = from.MonitoringSZ
	}
	to.Privilage = from.Privilage
}

// ImportHWSprRecord переносит аппаратные характеристики сервера, полученные из HP OneView
// (oneview2.Server), в запись справочника конфигураций оборудования ServerHardware,
// включая определение блейд-корзины (BladeSystems) и её местоположения в стойке.
//
// Примечание про oneview2: в отличие от старого пакета loadmon/oneview, здесь не
// нужно отдельно запрашивать корзину через ov.GetEnclosureByURI/ov.GetDeviceBayInfo -
// oneview2.CollectServers уже разрешает размещение (датацентр/стойка/корзина/bay)
// на каждый Server (см. loadmon/oneview2/location.go), поэтому используется
// srv.EnclosureInfo/srv.BayInfo напрямую. Параметр ov сохранён в сигнатуре для
// совместимости вызывающего кода, хотя сейчас не используется.
//
// Параметр o - ORM-исполнитель (обычная *orm.ormBase ИЛИ активная транзакция
// orm.TxOrmer), который использует вызывающий код для сохранения самого h.
// ИСПРАВЛЕНО (2026-09-07, по жалобе "при обновлении данных по oneview блэйд
// корзины не связываются с блэйд серверами"): раньше эта функция ВСЕГДА
// открывала СВОЙ отдельный orm.NewOrm() для чтения/создания записи
// BladeSystems, независимо от того, что вызывающий код (например
// cmdb/updatehwsprhp.go:UpdateHwSprHPServersInfra) мог в этот момент уже
// держать открытой СВОЮ транзакцию (trn := o.Begin()) на всю обработку
// одного сервера. На SQLite (см. повсеместный retryOnDatabaseLocked в этом
// пакете - явный признак однопоточной блокировки записи) второе, отдельное
// соединение, пытающееся вставить строку в blade_systems, пока первая
// транзакция ещё не закоммичена, рискует получить "database is locked" -
// и эта ошибка только печаталась в лог (fmt.Println), после чего
// h.BladeSystemId так и оставался 0 НАВСЕГДА для этой корзины. Теперь
// функция переиспользует ТОТ ЖЕ ORM-исполнитель, что и вызывающий код -
// работа с BladeSystems становится частью той же транзакции, конфликта
// блокировок больше нет.
// bladeSystemPowerActiveFromEnclosure выводит числовые коды
// BladeSystems.PowerState/Active (см. models_add.go: 0 не указано,
// PowerState 1 включен/2 выключен, Active 1 активен/2 выведен из
// эксплуатации) из состояния корзины по данным HPE OneView (2026-09-08, по
// запросу пользователя "простановка статусов PowerState и Active на основе
// состояния блэйд корзины"). У oneview2.EnclosureInfo нет отдельного поля
// "включена ли корзина" (в отличие от identity.PowerState у сервера) -
// ближайшие доступные сигналы:
//   - status - health-статус ("OK"/"Warning"/"Critical"/...) -> PowerState
//     (по аналогии с h.PowerState: есть определённый статус и он "OK" ->
//     включена/исправна (1), любой другой непустой статус -> выключена/
//     неисправна (2, то же значение используется для отображения
//     "выключен последнее известное состояние", см. комментарий у поля).
//   - state - состояние ресурса в OneView ("Configured"/"Removed"/...) ->
//     Active (непустое, отличное от "Removed" -> активна (1), "Removed" ->
//     выведена из эксплуатации (2)).
//
// Если соответствующее значение не пришло (пустая строка - GetEnclosure не
// отработал ни для одного сервера корзины), возвращается 0 ("не указано") -
// вызывающий код (см. ImportHWSprRecord) не должен затирать этим нулём уже
// известное значение при обновлении существующей записи.
func bladeSystemPowerActiveFromEnclosure(status, state string) (powerState, active int) {
	if status != "" {
		if status == "OK" {
			powerState = 1
		} else {
			powerState = 2
		}
	}
	if state != "" {
		if state == "Removed" {
			active = 2
		} else {
			active = 1
		}
	}
	return powerState, active
}

func ImportHWSprRecord(o orm.QueryExecutor, h *models.ServerHardware, srv *oneview.Server, ov *oneview.Infrastructure) {
	if o == nil || h == nil || srv == nil || ov == nil {
		return
	}

	// prev - снимок ДО перезаписи полей ниже, нужен только чтобы после
	// заполнения залогировать реально изменившиеся поля в
	// hardware_event_log (см. cmdb/hardwareeventlog.go, 2026-09-04, по
	// запросу "фиксировать изменение конфигурации... в отдельном логе").
	// Для НОВОЙ записи (h.Id==0, ещё не читана из БД вызывающим кодом)
	// снимок не используется - иначе каждое поле только что созданного
	// сервера выглядело бы как "изменившееся с пустого", что не является
	// изменением конфигурации в смысле этой задачи.
	wasExisting := h.Id > 0
	prev := *h

	identity := srv.Identity
	h.SerialNum = identity.SerialNumber
	h.VirtualSerialNum = identity.VirtualSerialNumber

	if identity.PowerState == "on" || identity.PowerState == "On" {
		h.PowerState = 1
	} else {
		h.PowerState = 2
	}
	// (2026-09-22, по факту найденной пользователем недопустимой комбинации
	// "PowerState=включен + Active=выведен из эксплуатации" у гипервизора,
	// видимого и в VMware, и в HP OneView) - та же авто-коррекция, что
	// добавлена в cmdb/updatehwhostinfo.go и cmdb/updatepower.go: если
	// OneView сейчас видит сервер включённым, Active принудительно
	// возвращается в 1, даже если ранее был выставлен вручную в 2
	// (/sprsrv/hwconfedit, /sprsrv/hwconfbulkedit) и с тех пор не
	// откатывался. При выключенном сервере Active намеренно не трогаем.
	// 2026-09-22: audit-helper, источник - models.UserSourceOneView (эта
	// функция ImportHWSprRecord импортирует данные из HP OneView).
	if h.PowerState == 1 {
		models.SetActiveWithAudit(&h.Active, &h.TimeActiveState, &h.ActivatorUser, &h.TimeDeActiveState, &h.DeActivatorUser, 1, models.UserSourceOneView)
	}

	h.Vendor = "HPE"
	h.Model = identity.Model
	h.FormFactor = 1

	// Определение "это блейд-сервер, в какой корзине, в каком отсеке" -
	// ИЗМЕНЕНО (2026-09-07, по наблюдению пользователя): ПЕРВИЧНЫЙ источник
	// теперь - разбор srv.Identity.Name (oneview2.ParseBladeNameBay).
	// HPE OneView сам называет ресурс blade server-hardware в формате
	// "c7000-9, bay 15" - это поле приходит в ТОМ ЖЕ ответе
	// /rest/server-hardware, что список серверов, БЕЗ дополнительного
	// запроса, и поэтому не подвержено сбоям отдельного вызова
	// GetEnclosure(hw.LocationURI) (см. exportHardware в oneview2/export.go),
	// от которого зависят identity.Position/srv.EnclosureInfo - именно
	// пустой srv.EnclosureInfo из-за сбоя этого вызова был основной причиной
	// того, что часть блейд-серверов не привязывалась к корзине (см.
	// CHANGELOG (31)/(32) про историю этого разбора).
	//
	// ЗАПАСНОЙ источник (если Name почему-то не в этом формате, например
	// нестандартное именование ресурса на конкретной инсталляции) -
	// старая связка identity.Position + srv.EnclosureInfo.Name.
	chassisName, bladeNum, isBlade := oneview.ParseBladeNameBay(identity.Name)
	if !isBlade && identity.Position > 0 {
		isBlade = true
		bladeNum = identity.Position
		chassisName = oneview.NormalizeEnclosureName(srv.EnclosureInfo.Name)
	}

	if isBlade {
		h.FormFactor = 2
		h.BladeNum = bladeNum

		// Данные о состоянии корзины (health-статус/state ресурса) сначала
		// берём из агрегата ov.EnclosureByName (собран по ВСЕМ серверам
		// корзины, не зависит от того, отработал ли GetEnclosure именно для
		// ЭТОГО сервера), запасной источник - srv.EnclosureInfo этого
		// сервера. См. bladeSystemPowerActiveFromEnclosure ниже.
		encStatus := srv.EnclosureInfo.Status
		encState := srv.EnclosureInfo.State
		if ov != nil && ov.EnclosureByName != nil {
			if es, ok := ov.EnclosureByName[chassisName]; ok && es != nil {
				if es.Status != "" {
					encStatus = es.Status
				}
				if es.State != "" {
					encState = es.State
				}
			}
		}
		newPowerState, newActive := bladeSystemPowerActiveFromEnclosure(encStatus, encState)

		bls := models.BladeSystems{Name: chassisName}
		found := chassisName != "" && o.Read(&bls, "Name") == nil
		if found {
			h.BladeSystemId = bls.Id
			// ИЗМЕНЕНО (2026-09-08): models.BladeSystems лишился текстовых
			// DatacenterRoom/RackNum/StartUnit - расположение корзины теперь
			// описывается только RackCabinetId (FK на справочник шкафов) и
			// StartUnitPos (юнит), см. models_add.go. Переносим их на сервер
			// той же логикой "если у сервера не указано - берём из данных
			// корзины".
			if h.RackCabinetId == 0 {
				h.RackCabinetId = bls.RackCabinetId
			}
			h.StartUnitPos = bls.StartUnitPos

			// ДОБАВЛЕНО (2026-09-08, по запросу пользователя): простановка
			// PowerState/Active корзины по её состоянию в OneView при каждой
			// синхронизации. Пишем только когда получили ОПРЕДЕЛЁННОЕ
			// (ненулевое) значение - если OneView сейчас не отдал статус
			// (например, GetEnclosure не отработал ни для одного сервера
			// корзины), не затираем ранее известное/вручную выставленное
			// значение нулём ("не указано").
			if newPowerState != 0 || newActive != 0 {
				cols := make([]string, 0, 2)
				if newPowerState != 0 && bls.PowerState != newPowerState {
					bls.PowerState = newPowerState
					cols = append(cols, "PowerState")
				}
				if newActive != 0 && bls.Active != newActive {
					bls.Active = newActive
					cols = append(cols, "Active")
				}
				if len(cols) > 0 {
					if _, err := o.Update(&bls, cols...); err != nil {
						fmt.Println("Ошибка обновления статуса корзины.", err)
					}
				}
			}
			MaybeUpdateBladeSystemPowerFromServer(o, &bls, srv.ChassisPower)
		} else if chassisName != "" { //добавить запись в справочник корзин
			// Данные о самой корзине (серийный номер/модель/тип) сначала
			// берём из агрегата ov.EnclosureByName (Infrastructure, см.
			// oneview2/global.go) - он собран по ВСЕМ серверам корзины
			// сразу и не зависит от того, отработал ли GetEnclosure именно
			// для ЭТОГО сервера; srv.EnclosureInfo этого сервера -
			// запасной источник, если агрегата нет или он тоже пуст.
			serialNum := srv.EnclosureInfo.SerialNumber
			model := srv.EnclosureInfo.Model
			encType := srv.EnclosureInfo.EnclosureType
			if ov != nil && ov.EnclosureByName != nil {
				if es, ok := ov.EnclosureByName[chassisName]; ok && es != nil {
					if es.SerialNumber != "" {
						serialNum = es.SerialNumber
					}
					if es.Model != "" {
						model = es.Model
					}
					if es.EnclosureType != "" {
						encType = es.EnclosureType
					}
				}
			}
			bls.SerialNum = serialNum
			bls.PartNum = srv.EnclosureInfo.PartNumber
			bls.Model = model
			bls.Type = encType
			bls.Size = srv.EnclosureInfo.DeviceBayCount
			bls.PowerState = newPowerState
			bls.Active = newActive
			// Примечание про oneview2: EnclosureInfo на уровне отдельного
			// Server хранит только общее количество отсеков (DeviceBayCount),
			// а не список занятых/свободных отсеков, как старый
			// ov.Enclosure.DeviceBays - поэтому bls.Used посчитать сейчас
			// нельзя и он остаётся нулевым.
			bls.Used = 0
			bls.Time_Mfy = time.Now().Local()
			bls.Anotation = fmt.Sprintf("%s Данные добавлены из HP One View", serialNum)
			// Ссылка проставляется сразу по возвращённому Id вставленной
			// записи (см. фикс от 2026-09-07 в CHANGELOG (31)) - сервер, из-за
			// которого корзина заводится впервые, не остаётся без ссылки на
			// неё до следующего цикла синхронизации.
			if id, err := o.Insert(&bls); err != nil {
				fmt.Println("Ошибка записи данных в справочник корзин.", err)
			} else {
				bls.Id = id
				h.BladeSystemId = bls.Id
				MaybeUpdateBladeSystemPowerFromServer(o, &bls, srv.ChassisPower)
			}
		}
	}
	if len(srv.Processors) > 0 {
		p := srv.Processors[0]
		h.CPUModel = p.Model
		h.CorePerCPU = p.TotalCores
		speed := p.OperatingSpeedMHz
		if speed == 0 {
			speed = p.MaxSpeedMHz
		}
		h.CoreMHz = speed
	}
	h.CPUCount = len(srv.Processors)
	h.MemoryMb = oneview.TotalMemoryMb(srv) //размер памяти в мб
	h.PartNum = identity.PartNumber

	// (2026-09-04) Раньше эти поля identity читались только частично
	// (Serial/VirtualSerial/PowerState/Model/PartNum) - данные о состоянии
	// здоровья, версии BIOS, платформе и контроллере управления (iLO/OA) от
	// OneView были доступны, но нигде не сохранялись. См. новые поля в
	// models.ServerHardware (models/models_add.go).
	h.Status = identity.Status
	h.RomVersion = identity.RomVersion
	h.Platform = identity.Platform
	h.ShortModel = identity.ShortModel
	h.AssetTag = identity.AssetTag
	h.OperatingSystem = identity.OperatingSystem
	h.MpModel = identity.MpModel
	h.MpFirmwareVersion = identity.MpFirmwareVersion
	h.MpIPAddress = identity.MpIPAddress
	h.MpHostName = identity.MpHostName

	ApplyServerHardwarePowerFromOneView(h, srv.Power)

	h.Time_Mfy = time.Now().Local()

	if wasExisting {
		key := firstNonEmpty(h.SerialNum, prev.SerialNum)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "Vendor/Model", prev.Vendor+" "+prev.Model, h.Vendor+" "+h.Model)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "SerialNum", prev.SerialNum, h.SerialNum)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "Status", prev.Status, h.Status)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "RomVersion", prev.RomVersion, h.RomVersion)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "Platform", prev.Platform, h.Platform)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "AssetTag", prev.AssetTag, h.AssetTag)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "OperatingSystem", prev.OperatingSystem, h.OperatingSystem)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "MpModel", prev.MpModel, h.MpModel)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "MpFirmwareVersion", prev.MpFirmwareVersion, h.MpFirmwareVersion)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "MpIPAddress", prev.MpIPAddress, h.MpIPAddress)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "MpHostName", prev.MpHostName, h.MpHostName)
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "BladeSystemId/BladeNum", fmt.Sprintf("%d/%d", prev.BladeSystemId, prev.BladeNum), fmt.Sprintf("%d/%d", h.BladeSystemId, h.BladeNum))
		if prev.CPUModel != h.CPUModel || prev.CPUCount != h.CPUCount || prev.CorePerCPU != h.CorePerCPU || prev.CoreMHz != h.CoreMHz {
			logHardwareFieldChange(o, "Server", h.Id, 0, key, "CPU",
				fmt.Sprintf("%s x%d, %d ядер, %d МГц", prev.CPUModel, prev.CPUCount, prev.CorePerCPU, prev.CoreMHz),
				fmt.Sprintf("%s x%d, %d ядер, %d МГц", h.CPUModel, h.CPUCount, h.CorePerCPU, h.CoreMHz))
		}
		if prev.MemoryMb != h.MemoryMb {
			logHardwareFieldChange(o, "Server", h.Id, 0, key, "MemoryMb", strconv.Itoa(prev.MemoryMb), strconv.Itoa(h.MemoryMb))
		}
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "PowerAvg24hW", FormatPowerW(prev.PowerAvg24hW), FormatPowerW(h.PowerAvg24hW))
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "PowerPeak24hW", FormatPowerW(prev.PowerPeak24hW), FormatPowerW(h.PowerPeak24hW))
		logHardwareFieldChange(o, "Server", h.Id, 0, key, "PowerRatedMaxW", FormatPowerW(prev.PowerRatedMaxW), FormatPowerW(h.PowerRatedMaxW))
	}
}

// ImportServerSprFromHW переносит данные о процессорах, памяти, дисковой ёмкости,
// состоянии питания и операционной системе из данных HP OneView (oneview2.Server)
// в запись справочника серверов ServerSprRecords.
//
// Определение "гипервизор или нет" (передача владения записью между OneView и
// vSphere-стороной CMDB):
//
// Раньше признаком "это гипервизор, обновление OS - в другом месте" служило
// либо уже сохранённое srvspr.Type == 2, либо срабатывание проверки
// strings.Contains(OS, "ESXI") на уже сохранённом значении OS. Это не
// учитывало смену РОЛИ оборудования: если сервер, ранее бывший гипервизором,
// переустановили под другую задачу (например, ESXi снесли и поставили ОС для
// базы данных), OneView продолжал видеть Type==2/OS=="esxi" в уже
// сохранённой записи и молча пропускал её - Type, Vcenter, Datacenter,
// Cluster, ClusterId так и оставались от прежней роли навсегда (баг,
// зафиксированный в этом комментарии для истории).
//
// Теперь роль определяется не по сохранённым данным, а по факту: ищем хост с
// таким же серийным номером в актуальном закэшированном пуле vSphere
// (vsphere.GetGlobalVMInfrastructure().HostBySNMapList - cmdb выполняется в
// процессе metrics_grpc и имеет прямой доступ к пулу, поэтому используется
// он напрямую, а не grpcclient, см. kontext/01-architecture.md).
//   - Хост НАЙДЕН по серийному номеру - сервер сейчас является гипервизором:
//     Type=2, Vcenter/Datacenter/Cluster/ClusterId обновляются данными из
//     vSphere, OS не трогаем (её актуализирует сторона vSphere, см.
//     cmdb.UpdateHWHostInfo).
//   - Хост НЕ найден, но в записи ранее было Type==2 - "передача владения":
//     оборудование больше не гипервизор (либо переустановлено под другую
//     роль, либо снято с учёта вместе с vCenter/кластером) - демотируем
//     запись обратно в Type=0 (физический сервер) и очищаем
//     Vcenter/Datacenter/Cluster/ClusterId/OS, чтобы дальше по функции OS
//     заполнилась уже из данных OneView, а не осталась зависшей на "esxi".
func ImportServerSprFromHW(srvspr *models.ServerSprRecords, srv *oneview.Server) {
	if srvspr == nil || srv == nil {
		return
	}

	isHypervisorNow := false
	if infra := vsphere.GetGlobalVMInfrastructure(); infra != nil {
		if hs, ok := infra.HostBySNMapList.Get(srv.Identity.SerialNumber); ok && hs != nil {
			isHypervisorNow = true
			srvspr.Type = 2
			srvspr.Vcenter = hs.Vcenter
			srvspr.Datacenter = hs.Datacenter
			srvspr.Cluster = hs.Cluster
			srvspr.ClusterId = hs.ClusterID
		}
	}
	if !isHypervisorNow && srvspr.Type == 2 {
		// передача владения - см. комментарий к функции выше
		srvspr.Type = 0
		srvspr.Vcenter = ""
		srvspr.Datacenter = ""
		srvspr.Cluster = ""
		srvspr.ClusterId = ""
		srvspr.OS = ""
	}

	srvspr.CPUCount = float64(len(srv.Processors))
	if len(srv.Processors) > 0 {
		srvspr.CPULogical = float64(len(srv.Processors)) * float64(srv.Processors[0].TotalCores)
	}
	srvspr.MemSize = float64(oneview.TotalMemoryMb(srv)) / 1024.0
	if srvspr.Type == 0 { //физический сервер
		srvspr.DiskSize = 0.0
		srvspr.SerialNumber = srv.Identity.SerialNumber
		for _, storage := range srv.Controllers {
			for _, logical := range storage.Volumes {
				srvspr.DiskSize += float64(logical.CapacityMiB) / 1024 //добавляем емкость всех Raid
			}
		}
	}
	if srv.Identity.PowerState == "on" || srv.Identity.PowerState == "On" {
		srvspr.PowerState = 1
	} else {
		srvspr.PowerState = 2
	}

	if srvspr.OS == "Unknown" {
		srvspr.OS = ""
	}

	if !isHypervisorNow {
		if srv.Identity.OperatingSystem == "" || srv.Identity.OperatingSystem == "Unknown" {
			//srvspr.OS = "" не трогаем данные введенные руками
		} else {
			srvspr.OS = srv.Identity.OperatingSystem
		}
	}
	//else: гипервизор - OS обновляется по данным vSphere в другом месте (cmdb.UpdateHWHostInfo), OneView здесь её не трогает.

	srvspr.Time_Mfy = time.Now().Local()
}

// ImportServerSprFromHWInfra — самостоятельная переписанная версия
// ImportServerSprFromHW для использования в cmdb.UpdateServerInventory:
// принимает снапшот инфраструктуры vSphere (infra) явным параметром, а не
// берёт его самостоятельно через vsphere.GetGlobalVMInfrastructure().
// ImportServerSprFromHW выше НЕ вызывает эту функцию и не вызывается ею —
// это две независимые функции с одинаковой логикой, чтобы существующие
// вызывающие места ImportServerSprFromHW (cmdb/insertHWSpr.go,
// cmdb/checkserverspr.go) остались абсолютно нетронутыми.
//
// Зачем это понадобилось: при последовательном вызове HP OneView-стороны
// синхронизации (UpdateHwSprHPServersInfra, эта функция) и VMware-стороны
// (cmdb.UpdateHWHostInfoInfra) в рамках ОДНОГО цикла
// cmdb.UpdateServerInventory каждая функция раньше брала СВОЙ отдельный
// снапшот vsphere.GetGlobalVMInfrastructure() - если фоновый опрос vCenter
// успевал подменить глобальный снапшот между этими двумя вызовами (или даже
// в процессе перебора серверов HP OneView внутри одного вызова), решение о
// демоции роли ("сервер больше не гипервизор", принятое здесь по снапшоту
// T1) и последующее решение UpdateHWHostInfoInfra о промоции/подтверждении
// роли (принятое по снапшоту T2) могли молча разойтись в рамках одного и
// того же цикла синхронизации. Явная передача infra параметром гарантирует,
// что оба шага видят ОДНО согласованное состояние vSphere. См.
// cmdb.UpdateServerInventory, где снапшот берётся один раз и передаётся в
// обе стороны.
//
// ВАЖНО про порядок строк ниже: определение демоции (srvspr.Type = 0 и
// очистка Vcenter/Datacenter/Cluster/ClusterId/OS) выполняется в начале
// функции и намеренно ДО блока "if srvspr.Type == 0 { ... }" и блока
// заполнения OS - оба этих блока читают уже актуализированное здесь же
// значение srvspr.Type. Перестановка блоков между собой изменит поведение.
func ImportServerSprFromHWInfra(srvspr *models.ServerSprRecords, srv *oneview.Server, infra *vsphere.VMInfrastructure) {
	if srvspr == nil || srv == nil {
		return
	}

	isHypervisorNow := false
	if infra != nil {
		if hs, ok := infra.HostBySNMapList.Get(srv.Identity.SerialNumber); ok && hs != nil {
			isHypervisorNow = true
			srvspr.Type = 2
			srvspr.Vcenter = hs.Vcenter
			srvspr.Datacenter = hs.Datacenter
			srvspr.Cluster = hs.Cluster
			srvspr.ClusterId = hs.ClusterID
		}
	}
	if !isHypervisorNow && srvspr.Type == 2 {
		// передача владения - см. комментарий к функции выше
		srvspr.Type = 0
		srvspr.Vcenter = ""
		srvspr.Datacenter = ""
		srvspr.Cluster = ""
		srvspr.ClusterId = ""
		srvspr.OS = ""
	}

	srvspr.CPUCount = float64(len(srv.Processors))
	if len(srv.Processors) > 0 {
		srvspr.CPULogical = float64(len(srv.Processors)) * float64(srv.Processors[0].TotalCores)
	}
	srvspr.MemSize = float64(oneview.TotalMemoryMb(srv)) / 1024.0
	if srvspr.Type == 0 { //физический сервер
		srvspr.DiskSize = 0.0
		srvspr.SerialNumber = srv.Identity.SerialNumber
		for _, storage := range srv.Controllers {
			for _, logical := range storage.Volumes {
				srvspr.DiskSize += float64(logical.CapacityMiB) / 1024 //добавляем емкость всех Raid
			}
		}
	}
	if srv.Identity.PowerState == "on" || srv.Identity.PowerState == "On" {
		srvspr.PowerState = 1
	} else {
		srvspr.PowerState = 2
	}

	if srvspr.OS == "Unknown" {
		srvspr.OS = ""
	}

	if !isHypervisorNow {
		if srv.Identity.OperatingSystem == "" || srv.Identity.OperatingSystem == "Unknown" {
			//srvspr.OS = "" не трогаем данные введенные руками
		} else {
			srvspr.OS = srv.Identity.OperatingSystem
		}
	}
	//else: гипервизор - OS обновляется по данным vSphere в другом месте (cmdb.UpdateHWHostInfo), OneView здесь её не трогает.

	srvspr.Time_Mfy = time.Now().Local()
}
