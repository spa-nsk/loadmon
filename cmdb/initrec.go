package cmdb

import (
	"fmt"

	// "loadmon/filefunction"
	models "loadmon/models"
	"loadmon/oneview"
	_ "loadmon/routers"
	vsphere "loadmon/vspheremetrics"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func CheckHypervisorOS(srv *models.ServerSprRecords) bool {
	if srv.OS != "" {
		if strings.Contains(strings.ToUpper(srv.OS), "ESXI") {
			return true
		}
	}
	return false
}

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

func UpdateVmToSprSrv(from vsphere.VMPoint, to *models.ServerSprRecords) {
	if to == nil {
		return
	}

	to.TagName = from.TagName
	to.TagDescription = from.TagDescription

	to.Type = 1
	if from.PowerState {
		to.PowerState = 1
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

func CopyHostToSprSrv(from vsphere.HostInfo, to *models.ServerSprRecords) {
	if to == nil {
		return
	}
	to.Type = 2
	if from.PowerState /*|| from.InMaintenanceMode*/ {
		to.PowerState = 1
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

	if from.CPUUsageMinimum == 777777777.0 {
		from.CPUUsageMinimum = 0.0
	}
	if from.MemUsageMinimum == 777777777.0 {
		from.MemUsageMinimum = 0.0
	}
	if from.DiskUsageMinimum == 777777777.0 {
		from.DiskUsageMinimum = 0.0
	}
	if from.DiskTotalReadLatencyMinimum == 777777777.0 {
		from.DiskTotalReadLatencyMinimum = 0.0
	}
	if from.DiskTotalWriteLatencyMinimum == 777777777.0 {
		from.DiskTotalWriteLatencyMinimum = 0.0
	}
	if from.DiskWriteMinimum == 777777777.0 {
		from.DiskWriteMinimum = 0.0
	}
	if from.DiskReadMinimum == 777777777.0 {
		from.DiskReadMinimum = 0.0
	}
	if from.NetUsageMinimum == 777777777.0 {
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

	if from.CPUUsageMinimum == 7777777777.0 {
		from.CPUUsageMinimum = 0.0
	}
	if from.MemUsageMinimum == 7777777777.0 {
		from.MemUsageMinimum = 0.0
	}
	if from.DiskUsageMinimum == 7777777777.0 {
		from.DiskUsageMinimum = 0.0
	}
	if from.DiskTotalReadLatencyMinimum == 7777777777.0 {
		from.DiskTotalReadLatencyMinimum = 0.0
	}
	if from.DiskTotalWriteLatencyMinimum == 7777777777.0 {
		from.DiskTotalWriteLatencyMinimum = 0.0
	}
	if from.DiskWriteMinimum == 7777777777.0 {
		from.DiskWriteMinimum = 0.0
	}
	if from.DiskReadMinimum == 7777777777.0 {
		from.DiskReadMinimum = 0.0
	}
	if from.NetUsageMinimum == 7777777777.0 {
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

func CopyReportRecToPerfSCOM(from *models.ReportServerRecord, to *models.ServerPerfSCOM) {
	if from == nil || to == nil {
		return
	}

	to.Date = from.Time
	if from.Virtual {
		to.Type = 1
	} else {
		to.Type = 0
	}
	to.NameFQDN = from.ServerName
	to.CPUCount = from.CPUCount
	to.CPULogical = from.CoreCount
	to.MemSize = from.MemSize
	to.SysDiskSize = from.SysDiskSize
	to.WorkDiskSize = from.WorkDiskSize
	to.CPULoadAvg = from.CPULoadAvg
	to.CPULoadMin = from.CPULoadMin
	to.CPULoadMax = from.CPULoadMax
	to.MemUseAvg = from.MemUseAvg
	to.MemUseMin = from.MemUseMin
	to.MemUseMax = from.MemUseMax
	to.SysDiskFreePercentAvg = from.SysDiskFreePercentAvg
	to.SysDiskFreePercentMin = from.SysDiskFreePercentMin
	to.SysDiskFreePercentMax = from.SysDiskFreePercentMax
	to.SysDiskIdleTimeAvg = from.SysDiskIdleTimeAvg
	to.SysDiskIdleTimeMin = from.SysDiskIdleTimeMin
	to.SysDiskIdleTimeMax = from.SysDiskIdleTimeMax
	to.SysDiskQueueAvg = from.SysDiskQueueAvg
	to.SysDiskQueueMin = from.SysDiskQueueMin
	to.SysDiskQueueMax = from.SysDiskQueueMax
	to.WorkDiskFreePercentAvg = from.WorkDiskFreePercentAvg
	to.WorkDiskFreePercentMin = from.WorkDiskFreePercentMin
	to.WorkDiskFreePercentMax = from.WorkDiskFreePercentMax
	to.WorkDiskIdleTimeAvg = from.WorkDiskIdleTimeAvg
	to.WorkDiskIdleTimeMin = from.WorkDiskIdleTimeMin
	to.WorkDiskIdleTimeMax = from.WorkDiskIdleTimeMax
	to.WorkDiskQueueAvg = from.WorkDiskQueueAvg
	to.WorkDiskQueueMin = from.WorkDiskQueueMin
	to.WorkDiskQueueMax = from.WorkDiskQueueMax
	to.SysDiskIOAvg = from.SysDiskIOAvg
	to.SysDiskIOMin = from.SysDiskIOMin
	to.SysDiskIOMax = from.SysDiskIOMax
	to.SysDiskFreeMbAvg = from.SysDiskFreeMbAvg
	to.SysDiskFreeMbMin = from.SysDiskFreeMbMin
	to.SysDiskFreeMbMax = from.SysDiskFreeMbMax
	to.WorkDiskIOAvg = from.WorkDiskIOAvg
	to.WorkDiskIOMin = from.WorkDiskIOMin
	to.WorkDiskIOMax = from.WorkDiskIOMax
	to.WorkDiskFreeMbAvg = from.WorkDiskFreeMbAvg
	to.WorkDiskFreeMbMin = from.WorkDiskFreeMbMin
	to.WorkDiskFreeMbMax = from.WorkDiskFreeMbMax
	if from.UseWorkDisk {
		to.UseWorkDisk = 1
	} else {
		to.UseWorkDisk = 0
	}
}

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
	to.TimeActiveState = from.TimeActiveState
	to.ActivatorUser = from.ActivatorUser
	to.TimeDeActiveState = from.TimeActiveState
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
	if to.DPCAddress == 0 {
		to.DPCAddress = from.DPCAddress
	}
	if to.DPCOffice == 0 {
		to.DPCOffice = from.DPCOffice
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

func ImportHWSprRecord(h *models.ServerHardware, srv *oneview.ServerHardware, ov *oneview.OVInfrastructure) {
	if h == nil || srv == nil || ov == nil {
		return
	}

	//ov := oneview.GetGlobalOVInfrastructure()
	o := orm.NewOrm()

	server := srv.Base
	h.SerialNum = srv.Base.SerialNumber.String()
	vsn := srv.Base.VirtualSerialNumber.String()

	if vsn == "null" || vsn == "" {
		h.VirtualSerialNum = ""
	} else {
		h.VirtualSerialNum = vsn
	}

	if srv.Base.PowerState == "on" || srv.Base.PowerState == "On" {
		h.PowerState = 1
	} else {
		h.PowerState = 2
	}

	h.DatacenterName = srv.Datacenter.Name

	h.RackName = srv.Rack.Name

	h.Vendor = "HPE"
	h.Model = server.Model
	h.FormFactor = 1

	if server.Position > 0 {
		h.FormFactor = 2
		h.BladeNum = server.Position

		BladeSystem := ""
		enc, ok := ov.GetEnclosureByURI(server.LocationURI.String())

		if ok {
			BladeSystem = enc.Name
		}

		if devbay, ok := ov.GetDeviceBayInfo(server.URI.String()); ok {
			fmt.Println(devbay.BayNumber, server.Position)
		}
		s := strings.Split(BladeSystem, "-")
		if len(s) == 2 {
			num, _ := strconv.Atoi(s[1])
			BladeSystem = fmt.Sprintf("%s-%02d", s[0], num)
		}
		bls := models.BladeSystems{Name: BladeSystem}
		if err := o.Read(&bls, "Name"); err == nil {
			h.BladeSystemId = bls.Id
			if h.DatacenterRoom == "" { //если у сервера не указан датацентр то берем из данных корзины
				h.DatacenterRoom = bls.DatacenterRoom
			}
			if h.RackNum == "" { //если у сервера не указан номер стойки то берем из данных корзины
				h.RackNum = bls.RackNum
			}
			h.StartUnit = bls.StartUnit
		} else { //добавить запись в справочник корзин
			if ok {
				bls.SerialNum = enc.SerialNumber
				bls.PartNum = enc.PartNumber
				bls.Model = enc.EnclosureModel
				bls.Size = enc.DeviceBayCount
				bls.Used = 0
				for _, bay := range enc.DeviceBays {
					if bay.DevicePresence == "Present" {
						bls.Used++
					}
				}
				bls.Time_Mfy = time.Now().Local()
				bls.Anotation = fmt.Sprintf("%s%s", enc.SerialNumber, " Данные добавлены из HP One View")
				if _, err := o.Insert(&bls); err != nil {
					fmt.Println("Ошибка записи данных в справочник корзин.", err)
				}
			}
		}
	}
	h.CPUModel = server.ProcessorType
	h.CPUCount = server.ProcessorCount
	h.CorePerCPU = server.ProcessorCoreCount
	h.CoreMHz = server.ProcessorSpeedMhz
	h.MemoryMb = server.MemoryMb //размер памяти в мб
	h.PartNum = server.PartNumber
	h.Time_Mfy = time.Now().Local()
}

func ImportServerSprFromHW(srvspr *models.ServerSprRecords, srv *oneview.ServerHardware) {
	if srvspr == nil || srv == nil {
		return
	}
	srvspr.CPUCount = float64(srv.Base.ProcessorCount)
	srvspr.CPULogical = float64(srv.Base.ProcessorCount) * float64(srv.Base.ProcessorCoreCount)
	srvspr.MemSize = float64(srv.Base.MemoryMb) / 1024.0
	if srvspr.Type == 0 { //физический сервер
		srvspr.DiskSize = 0.0
		srvspr.SerialNumber = string(srv.Base.SerialNumber)
		for _, storage := range srv.Storage.Data {
			for _, logical := range storage.LogicalDrives {
				srvspr.DiskSize += logical.CapacityMiB / 1024 //добавляем емкость всех Raid
			}
		}
	}
	if srv.Base.PowerState == "on" || srv.Base.PowerState == "On" {
		srvspr.PowerState = 1
	} else {
		srvspr.PowerState = 2
	}

	if srvspr.OS == "Unknown" {
		srvspr.OS = ""
	}

	up := strings.ToUpper(srvspr.OS)
	if strings.Contains(up, "ESXI") || srvspr.Type == 2 {
		//это гипервизор и обновление будет в другом месте
	} else {
		if srv.OperatingSystem == "" || srv.OperatingSystem == "Unknown" {
			//srvspr.OS = "" не трогаем данные введенные руками
		} else {
			srvspr.OS = srv.OperatingSystem
		}
	}
	srvspr.Time_Mfy = time.Now().Local()
}
