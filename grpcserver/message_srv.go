package grpcserver

import (
	"fmt"
	"loadmon/models"
	"loadmon/oneview"
	"loadmon/protocol"
	"math"
	"strconv"

	vsphere "loadmon/vspheremetrics"
	pb "loadmon/protocol"

	"github.com/HewlettPackard/oneview-golang/utils"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

// VMToMessageShort конвертирует внутреннюю структуру vsphere.VMPoint в укороченное gRPC-сообщение pb.VMPointShort.
// Принимает vm (доменные данные ВМ) и out (указатель на заполняемое сообщение). Если out == nil, функция ничего не делает.
func VMToMessageShort(vm vsphere.VMPoint, out *pb.VMPointShort) {
	if out == nil {
		return
	}
	out.ClusterId = vm.ClusterID
	out.Workstation = vm.WorkStation
	out.Template = vm.Template
	out.TagName = vm.TagName
	out.TagDescription = vm.TagDescription
	out.Id = vm.ID
	out.Name = vm.Name

	out.DomainName = vm.DomainName
	out.GuestDetailedData = vm.GuestDetailedData
	out.ToolsStatus = vm.ToolsStatus

	out.GuestFullName = vm.GuestFullName
	out.HostName = vm.HostName
	out.PathName = vm.PathName
	out.PowerState = vm.PowerState
	out.CpuMhz = vm.CPUMhz
	out.NumCPU = vm.NumCPU
	out.NumCores = vm.NumCores
	out.MemorySizeGB = vm.MemorySizeGB
	out.IpAddress = vm.IPAddress
	out.MacAddress = vm.MacAddress
	out.Network = vm.Network
	out.CapacityGb = vm.CapacityGb
	out.FreeSpaceGb = vm.FreeSpaceGb
	out.PercentUsed = vm.PercentUsed
	out.PerfCount = vm.PerfCount
	if vm.PerfCount > 0 {

		out.CpuUsageMaximum = vm.CPUUsageMaximum
		out.CpuUsageMaximumTime = timestamppb.New(vm.CPUUsageMaximumTime)
		out.CpuUsageMinimum = vm.CPUUsageMinimum
		out.CpuUsageMinimumTime = timestamppb.New(vm.CPUUsageMinimumTime)
		out.MemUsageMaximum = vm.MemUsageMaximum
		out.MemUsageMaximumTime = timestamppb.New(vm.MemUsageMaximumTime)
		out.MemUsageMinimum = vm.MemUsageMinimum
		out.MemUsageMinimumTime = timestamppb.New(vm.MemUsageMinimumTime)
		out.DiskUsageMaximum = vm.DiskUsageMaximum
		out.DiskUsageMaximumTime = timestamppb.New(vm.DiskUsageMaximumTime)
		out.DiskUsageMinimum = vm.DiskUsageMinimum
		out.DiskUsageMinimumTime = timestamppb.New(vm.DiskUsageMinimumTime)
		out.DiskWriteMaximum = vm.DiskWriteMaximum
		out.DiskWriteMaximumTime = timestamppb.New(vm.DiskWriteMaximumTime)
		out.DiskWriteMinimum = vm.DiskWriteMinimum
		out.DiskWriteMinimumTime = timestamppb.New(vm.DiskWriteMinimumTime)
		out.DiskReadMaximum = vm.DiskReadMaximum
		out.DiskReadMaximumTime = timestamppb.New(vm.DiskReadMaximumTime)
		out.DiskReadMinimum = vm.DiskReadMinimum
		out.DiskReadMinimumTime = timestamppb.New(vm.DiskReadMinimumTime)
		out.NetUsageMaximum = vm.NetUsageMaximum
		out.NetUsageMaximumTime = timestamppb.New(vm.NetUsageMaximumTime)
		out.NetUsageMinimum = vm.NetUsageMinimum
		out.NetUsageMinimumTime = timestamppb.New(vm.NetUsageMinimumTime)

		count := float64(vm.PerfCount)
		out.CpuUsageAverage = vm.CPUUsageAverage / count
		out.MemUsageAverage = vm.MemUsageAverage / count
		out.DiskUsageAverage = vm.DiskUsageAverage / count
		out.DiskWriteAverage = vm.DiskWriteAverage / count
		out.DiskReadAverage = vm.DiskReadAverage / count
		out.NetUsageAverage = vm.NetUsageAverage / count
	}
}

// VMToMessageFull конвертирует внутреннюю структуру vsphere.VMPoint в полное gRPC-сообщение pb.VMPoint.
// Принимает vm (доменные данные ВМ) и out (указатель на заполняемое сообщение). Если out == nil, функция ничего не делает.
func VMToMessageFull(vm vsphere.VMPoint, out *pb.VMPoint) {
	if out == nil {
		return
	}
	out.Workstation = vm.WorkStation
	out.Template = vm.Template
	out.TagName = vm.TagName
	out.TagDescription = vm.TagDescription
	out.Vcenter = vm.Vcenter
	out.Datacenter = vm.Datacenter
	out.Cluster = vm.Cluster
	out.ClusterId = vm.ClusterID
	out.KeyName = vm.KeyName
	out.Uuid = vm.UUID
	out.GuestId = vm.GuestID
	out.Annotation = vm.Annotation
	out.NumEthernetCards = vm.NumEthernetCards
	out.NumVirtualDisks = vm.NumVirtualDisks
	out.DiskCommittedGB = vm.DiskCommittedGB
	out.DiskUncommittedGB = vm.DiskUncommittedGB
	out.CpuBilling = vm.CPUBilling
	out.MemBilling = vm.MemBilling
	out.DiskBilling = vm.DiskBilling
	out.NetBilling = vm.NetBilling
	out.Id = vm.ID
	out.Name = vm.Name

	out.DomainName = vm.DomainName
	out.GuestDetailedData = vm.GuestDetailedData
	out.ToolsStatus = vm.ToolsStatus

	out.GuestFullName = vm.GuestFullName
	out.HostName = vm.HostName
	out.PathName = vm.PathName
	out.PowerState = vm.PowerState
	out.CpuMhz = vm.CPUMhz
	out.NumCPU = vm.NumCPU
	out.NumCores = vm.NumCores
	out.MemorySizeGB = vm.MemorySizeGB
	out.IpAddress = vm.IPAddress
	out.MacAddress = vm.MacAddress
	out.Network = vm.Network
	out.CapacityGb = vm.CapacityGb
	out.FreeSpaceGb = vm.FreeSpaceGb
	out.PercentUsed = vm.PercentUsed

	if vm.PerfCount > 0 {

		out.CpuUsageMaximum = vm.CPUUsageMaximum
		out.CpuUsageMaximumTime = timestamppb.New(vm.CPUUsageMaximumTime)
		out.CpuUsageMinimum = vm.CPUUsageMinimum
		out.CpuUsageMinimumTime = timestamppb.New(vm.CPUUsageMinimumTime)
		out.MemUsageMaximum = vm.MemUsageMaximum
		out.MemUsageMaximumTime = timestamppb.New(vm.MemUsageMaximumTime)
		out.MemUsageMinimum = vm.MemUsageMinimum
		out.MemUsageMinimumTime = timestamppb.New(vm.MemUsageMinimumTime)
		out.DiskUsageMaximum = vm.DiskUsageMaximum
		out.DiskUsageMaximumTime = timestamppb.New(vm.DiskUsageMaximumTime)
		out.DiskUsageMinimum = vm.DiskUsageMinimum
		out.DiskUsageMinimumTime = timestamppb.New(vm.DiskUsageMinimumTime)
		out.DiskWriteMaximum = vm.DiskWriteMaximum
		out.DiskWriteMaximumTime = timestamppb.New(vm.DiskWriteMaximumTime)
		out.DiskWriteMinimum = vm.DiskWriteMinimum
		out.DiskWriteMinimumTime = timestamppb.New(vm.DiskWriteMinimumTime)
		out.DiskReadMaximum = vm.DiskReadMaximum
		out.DiskReadMaximumTime = timestamppb.New(vm.DiskReadMaximumTime)
		out.DiskReadMinimum = vm.DiskReadMinimum
		out.DiskReadMinimumTime = timestamppb.New(vm.DiskReadMinimumTime)
		out.NetUsageMaximum = vm.NetUsageMaximum
		out.NetUsageMaximumTime = timestamppb.New(vm.NetUsageMaximumTime)
		out.NetUsageMinimum = vm.NetUsageMinimum
		out.NetUsageMinimumTime = timestamppb.New(vm.NetUsageMinimumTime)

		count := float64(vm.PerfCount)
		out.CpuUsageAverage = vm.CPUUsageAverage / count
		out.MemUsageAverage = vm.MemUsageAverage / count
		out.DiskUsageAverage = vm.DiskUsageAverage / count
		out.DiskWriteAverage = vm.DiskWriteAverage / count
		out.DiskReadAverage = vm.DiskReadAverage / count
		out.NetUsageAverage = vm.NetUsageAverage / count
	}

	for _, p := range vm.Perf {
		perf := protocol.VMPerf{}
		perf.Id = p.ID
		perf.Start = timestamppb.New(p.Start)
		perf.CpuUsage = p.CPUUsage
		perf.MemUsage = p.MemUsage
		perf.DiskUsage = p.DiskUsage
		perf.DiskRead = p.DiskRead
		perf.DiskWrite = p.DiskWrite
		perf.NetUsage = p.NetUsage
		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		out.Perf = append(out.Perf, &perf)
	}
	for _, f := range vm.Files {
		fn := pb.VMFile{}
		fn.Type = f.Type
		fn.Name = f.Name
		fn.Size = f.Size
		fn.UniqueSize = f.UniqueSize
		out.VMFiles = append(out.VMFiles, &fn)
	}
	for _, d := range vm.Disk {
		dn := pb.GuestDiskInfo{}
		dn.CapacityGb = d.CapacityGb
		dn.DiskPath = d.DiskPath
		dn.FreeSpaceGb = d.FreeSpaceGb
		dn.PercentUsed = d.PercentUsed
		out.VMDisk = append(out.VMDisk, &dn)
	}
	for _, d := range vm.UsageDatastore {
		dn := pb.VMUsageOnDatastore{}
		dn.Id = d.ID
		dn.UID = d.UID
		dn.Vcenter = d.Vcenter
		dn.Datacenter = d.Datacenter
		dn.Cluster = d.Cluster
		dn.CommittedGb = d.CommittedGb
		dn.UncommittedGb = d.UncommittedGb
		dn.UnsharedGb = d.UnsharedGb
		dn.PercentAllocateStorage = d.PercentAllocateStorage
		dn.StorageName = d.StorageName
		dn.Datastore = d.Datastore
		out.UsageDatastore = append(out.UsageDatastore, &dn)
	}
}

// HostToMessageShort конвертирует внутреннюю структуру vsphere.HostInfo в укороченное gRPC-сообщение pb.Host без пересчёта агрегированных метрик.
// Принимает hs и out; при hs == nil или out == nil ничего не делает.
func HostToMessageShort(hs *vsphere.HostInfo, out *pb.Host) {
	if out == nil || hs == nil {
		return
	}
	out.TagName = hs.TagName
	out.TagDescription = hs.TagDescription
	out.Id = hs.ID
	out.Name = hs.Name

	out.Vcenter = hs.Vcenter
	out.Datacenter = hs.Datacenter
	out.Cluster = hs.Cluster
	out.ClusterId = hs.ClusterID

	out.KeyName = hs.KeyName
	out.Vendor = hs.Vendor
	out.Model = hs.Model

	out.ProductFullName = hs.ProductFullName
	out.SerialNumber = hs.SerialNumber

	out.VirtualSerialNumber = hs.VirtualSerialNumber
	out.EnclosureSerialNumber = hs.EnclosureSerialNumber
	out.Uuid = hs.UUID
	out.PowerState = hs.PowerState
	out.InMaintenanceMode = hs.InMaintenanceMode
	out.CpuModel = hs.CPUModel
	out.CpuMhz = hs.CPUMhz
	out.NumCpuPkgs = int32(hs.NumCPUPkgs)
	out.NumCpuCores = int32(hs.NumCPUCores)
	out.NumCpuThreads = int32(hs.NumCPUThreads)
	out.NumHBAs = hs.NumHBAs
	out.NumNics = hs.NumNics
	out.IpAddress = hs.IPAddress
	out.MacAddress = hs.MacAddress
	out.MemorySizeGB = hs.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = hs.HostMaxVirtualDiskCapacity

	out.PerfCount = hs.PerfCount
	if hs.PerfCount > 0 {

		out.CpuUsageMaximum = hs.CPUUsageMaximum
		out.CpuUsageMaximumTime = timestamppb.New(hs.CPUUsageMaximumTime)
		out.CpuUsageMinimum = hs.CPUUsageMinimum
		out.CpuUsageMinimumTime = timestamppb.New(hs.CPUUsageMinimumTime)
		out.MemUsageMaximum = hs.MemUsageMaximum
		out.MemUsageMaximumTime = timestamppb.New(hs.MemUsageMaximumTime)
		out.MemUsageMinimum = hs.MemUsageMinimum
		out.MemUsageMinimumTime = timestamppb.New(hs.MemUsageMinimumTime)
		out.DiskUsageMaximum = hs.DiskUsageMaximum
		out.DiskUsageMaximumTime = timestamppb.New(hs.DiskUsageMaximumTime)
		out.DiskUsageMinimum = hs.DiskUsageMinimum
		out.DiskUsageMinimumTime = timestamppb.New(hs.DiskUsageMinimumTime)
		out.DiskWriteMaximum = hs.DiskWriteMaximum
		out.DiskWriteMaximumTime = timestamppb.New(hs.DiskWriteMaximumTime)
		out.DiskWriteMinimum = hs.DiskWriteMinimum
		out.DiskWriteMinimumTime = timestamppb.New(hs.DiskWriteMinimumTime)
		out.DiskReadMaximum = hs.DiskReadMaximum
		out.DiskReadMaximumTime = timestamppb.New(hs.DiskReadMaximumTime)
		out.DiskReadMinimum = hs.DiskReadMinimum
		out.DiskReadMinimumTime = timestamppb.New(hs.DiskReadMinimumTime)
		out.NetUsageMaximum = hs.NetUsageMaximum
		out.NetUsageMaximumTime = timestamppb.New(hs.NetUsageMaximumTime)
		out.NetUsageMinimum = hs.NetUsageMinimum
		out.NetUsageMinimumTime = timestamppb.New(hs.NetUsageMinimumTime)

		count := float64(hs.PerfCount)
		out.CpuUsageAverage = hs.CPUUsageAverage / count
		out.MemUsageAverage = hs.MemUsageAverage / count
		out.DiskUsageAverage = hs.DiskUsageAverage / count
		out.DiskWriteAverage = hs.DiskWriteAverage / count
		out.DiskReadAverage = hs.DiskReadAverage / count
		out.NetUsageAverage = hs.NetUsageAverage / count
	}
}

// HostToMessageFull конвертирует внутреннюю структуру vsphere.HostInfo в полное gRPC-сообщение pb.HostFull без пересчёта агрегированных метрик.
// Принимает hs и out; при hs == nil или out == nil ничего не делает.
func HostToMessageFull(hs *vsphere.HostInfo, out *pb.HostFull) {
	if out == nil || hs == nil {
		return
	}
	out.TagName = hs.TagName
	out.TagDescription = hs.TagDescription
	out.Id = hs.ID
	out.Name = hs.Name

	out.Vcenter = hs.Vcenter
	out.Datacenter = hs.Datacenter
	out.Cluster = hs.Cluster
	out.ClusterId = hs.ClusterID
	out.KeyName = hs.KeyName
	out.Vendor = hs.Vendor
	out.Model = hs.Model
	out.SerialNumber = hs.SerialNumber

	out.ProductFullName = hs.ProductFullName
	out.VirtualSerialNumber = hs.VirtualSerialNumber
	out.EnclosureSerialNumber = hs.EnclosureSerialNumber
	out.Uuid = hs.UUID
	out.PowerState = hs.PowerState
	out.InMaintenanceMode = hs.InMaintenanceMode
	out.CpuModel = hs.CPUModel
	out.CpuMhz = hs.CPUMhz
	out.NumCpuPkgs = int32(hs.NumCPUPkgs)
	out.NumCpuCores = int32(hs.NumCPUCores)
	out.NumCpuThreads = int32(hs.NumCPUThreads)
	out.NumHBAs = hs.NumHBAs
	out.NumNics = hs.NumNics
	out.IpAddress = hs.IPAddress
	out.MacAddress = hs.MacAddress
	out.MemorySizeGB = hs.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = hs.HostMaxVirtualDiskCapacity

	out.PerfCount = hs.PerfCount
	if hs.PerfCount > 0 {

		out.CpuUsageMaximum = hs.CPUUsageMaximum
		out.CpuUsageMaximumTime = timestamppb.New(hs.CPUUsageMaximumTime)
		out.CpuUsageMinimum = hs.CPUUsageMinimum
		out.CpuUsageMinimumTime = timestamppb.New(hs.CPUUsageMinimumTime)
		out.MemUsageMaximum = hs.MemUsageMaximum
		out.MemUsageMaximumTime = timestamppb.New(hs.MemUsageMaximumTime)
		out.MemUsageMinimum = hs.MemUsageMinimum
		out.MemUsageMinimumTime = timestamppb.New(hs.MemUsageMinimumTime)
		out.DiskUsageMaximum = hs.DiskUsageMaximum
		out.DiskUsageMaximumTime = timestamppb.New(hs.DiskUsageMaximumTime)
		out.DiskUsageMinimum = hs.DiskUsageMinimum
		out.DiskUsageMinimumTime = timestamppb.New(hs.DiskUsageMinimumTime)
		out.DiskWriteMaximum = hs.DiskWriteMaximum
		out.DiskWriteMaximumTime = timestamppb.New(hs.DiskWriteMaximumTime)
		out.DiskWriteMinimum = hs.DiskWriteMinimum
		out.DiskWriteMinimumTime = timestamppb.New(hs.DiskWriteMinimumTime)
		out.DiskReadMaximum = hs.DiskReadMaximum
		out.DiskReadMaximumTime = timestamppb.New(hs.DiskReadMaximumTime)
		out.DiskReadMinimum = hs.DiskReadMinimum
		out.DiskReadMinimumTime = timestamppb.New(hs.DiskReadMinimumTime)
		out.NetUsageMaximum = hs.NetUsageMaximum
		out.NetUsageMaximumTime = timestamppb.New(hs.NetUsageMaximumTime)
		out.NetUsageMinimum = hs.NetUsageMinimum
		out.NetUsageMinimumTime = timestamppb.New(hs.NetUsageMinimumTime)

		count := float64(hs.PerfCount)
		out.CpuUsageAverage = hs.CPUUsageAverage / count
		out.MemUsageAverage = hs.MemUsageAverage / count
		out.DiskUsageAverage = hs.DiskUsageAverage / count
		out.DiskWriteAverage = hs.DiskWriteAverage / count
		out.DiskReadAverage = hs.DiskReadAverage / count
		out.NetUsageAverage = hs.NetUsageAverage / count
	}

	for _, p := range hs.Perf {
		perf := protocol.HSPerf{}
		perf.Id = p.ID
		perf.Start = timestamppb.New(p.Start)
		perf.CpuUsage = p.CPUUsage
		perf.CpuUtilization = p.CPUUtilization
		perf.MemUsage = p.MemUsage
		perf.DiskUsage = p.DiskUsage
		perf.DiskRead = p.DiskRead
		perf.DiskWrite = p.DiskWrite
		perf.NetUsage = p.NetUsage
		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		out.Perf = append(out.Perf, &perf)
	}
}

// RecalcHostToMessageFull конвертирует vsphere.HostInfo в pb.HostFull и дополнительно пересчитывает агрегированные показатели
// (минимум/максимум/среднее по метрикам) на основе накопленных данных hs.Perf. Принимает hs и out; при hs == nil или out == nil ничего не делает.
func RecalcHostToMessageFull(hs *vsphere.HostInfo, out *pb.HostFull) {
	if out == nil || hs == nil {
		return
	}
	out.TagName = hs.TagName
	out.TagDescription = hs.TagDescription
	out.Id = hs.ID
	out.Name = hs.Name

	out.Vcenter = hs.Vcenter
	out.Datacenter = hs.Datacenter
	out.Cluster = hs.Cluster
	out.ClusterId = hs.ClusterID
	out.KeyName = hs.KeyName
	out.Vendor = hs.Vendor
	out.Model = hs.Model
	out.SerialNumber = hs.SerialNumber

	out.ProductFullName = hs.ProductFullName
	out.VirtualSerialNumber = hs.VirtualSerialNumber
	out.EnclosureSerialNumber = hs.EnclosureSerialNumber
	out.Uuid = hs.UUID
	out.PowerState = hs.PowerState
	out.InMaintenanceMode = hs.InMaintenanceMode
	out.CpuModel = hs.CPUModel
	out.CpuMhz = hs.CPUMhz
	out.NumCpuPkgs = int32(hs.NumCPUPkgs)
	out.NumCpuCores = int32(hs.NumCPUCores)
	out.NumCpuThreads = int32(hs.NumCPUThreads)
	out.NumHBAs = hs.NumHBAs
	out.NumNics = hs.NumNics
	out.IpAddress = hs.IPAddress
	out.MacAddress = hs.MacAddress
	out.MemorySizeGB = hs.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = hs.HostMaxVirtualDiskCapacity

	out.PerfCount = hs.PerfCount
	if hs.PerfCount > 0 {

		out.CpuUsageMinimum = math.MaxFloat64
		out.CpuUsageMaximum = 0.0
		out.MemUsageMinimum = math.MaxFloat64
		out.MemUsageMaximum = 0.0
		out.DiskUsageMinimum = math.MaxFloat64
		out.DiskUsageMaximum = 0.0
		out.DiskWriteMinimum = math.MaxFloat64
		out.DiskWriteMaximum = 0.0
		out.DiskReadMinimum = math.MaxFloat64
		out.DiskReadMaximum = 0.0
		out.NetUsageMinimum = math.MaxFloat64
		out.NetUsageMaximum = 0.0

		for _, p := range hs.Perf {
			perf := protocol.HSPerf{}
			perf.Id = p.ID
			perf.Start = timestamppb.New(p.Start)

			perf.CpuUsage = p.CPUUsage
			out.CpuUsageAverage += p.CPUUsage
			if out.CpuUsageMinimum > p.CPUUsage {
				out.CpuUsageMinimum = p.CPUUsage
				out.CpuUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.CpuUsageMaximum < p.CPUUsage {
				out.CpuUsageMaximum = p.CPUUsage
				out.CpuUsageMaximumTime = timestamppb.New(p.Start)
			}

			perf.CpuUtilization = p.CPUUtilization

			perf.MemUsage = p.MemUsage
			out.MemUsageAverage += p.MemUsage
			if out.MemUsageMinimum > p.MemUsage {
				out.MemUsageMinimum = p.MemUsage
				out.MemUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.MemUsageMaximum < p.MemUsage {
				out.MemUsageMaximum = p.MemUsage
				out.MemUsageMaximumTime = timestamppb.New(p.Start)
			}

			perf.DiskUsage = p.DiskUsage
			out.DiskUsageAverage += p.DiskUsage
			if out.DiskUsageMinimum > p.DiskUsage {
				out.DiskUsageMinimum = p.DiskUsage
				out.DiskUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.DiskUsageMaximum < p.DiskUsage {
				out.DiskUsageMaximum = p.DiskUsage
				out.DiskUsageMaximumTime = timestamppb.New(p.Start)
			}

			perf.DiskRead = p.DiskRead
			out.DiskReadAverage += p.DiskRead
			if out.DiskReadMinimum > p.DiskRead {
				out.DiskReadMinimum = p.DiskRead
				out.DiskReadMinimumTime = timestamppb.New(p.Start)
			}
			if out.DiskReadMaximum < p.DiskRead {
				out.DiskReadMaximum = p.DiskRead
				out.DiskReadMaximumTime = timestamppb.New(p.Start)
			}

			perf.DiskWrite = p.DiskWrite
			out.DiskWriteAverage += p.DiskWrite
			if out.DiskWriteMinimum > p.DiskWrite {
				out.DiskWriteMinimum = p.DiskWrite
				out.DiskWriteMinimumTime = timestamppb.New(p.Start)
			}
			if out.DiskWriteMaximum < p.DiskWrite {
				out.DiskWriteMaximum = p.DiskWrite
				out.DiskWriteMaximumTime = timestamppb.New(p.Start)
			}

			perf.NetUsage = p.NetUsage
			out.NetUsageAverage += p.NetUsage
			if out.NetUsageMinimum > p.NetUsage {
				out.NetUsageMinimum = p.NetUsage
				out.NetUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.NetUsageMaximum < p.NetUsage {
				out.NetUsageMaximum = p.NetUsage
				out.NetUsageMaximumTime = timestamppb.New(p.Start)
			}

			perf.NetReceived = p.NetReceived
			perf.NetTransmitted = p.NetTransmitted
			perf.DiskTotalReadLatency = p.DiskTotalReadLatency
			perf.DiskTotalWriteLatency = p.DiskTotalWriteLatency
			out.Perf = append(out.Perf, &perf)
		}
		out.PerfCount = (int64)(len(hs.Perf))
		count := float64(out.PerfCount)
		out.CpuUsageAverage = hs.CPUUsageAverage / count
		out.MemUsageAverage = hs.MemUsageAverage / count
		out.DiskUsageAverage = hs.DiskUsageAverage / count
		out.DiskWriteAverage = hs.DiskWriteAverage / count
		out.DiskReadAverage = hs.DiskReadAverage / count
		out.NetUsageAverage = hs.NetUsageAverage / count
	}
}

// RecalcHostToMessageShort конвертирует vsphere.HostInfo в укороченное pb.Host и дополнительно пересчитывает агрегированные показатели
// (минимум/максимум/среднее по метрикам) на основе накопленных данных hs.Perf. Принимает hs и out; при hs == nil или out == nil ничего не делает.
func RecalcHostToMessageShort(hs *vsphere.HostInfo, out *pb.Host) {
	if out == nil || hs == nil {
		return
	}
	out.TagName = hs.TagName
	out.TagDescription = hs.TagDescription
	out.Id = hs.ID
	out.Name = hs.Name

	out.Vcenter = hs.Vcenter
	out.Datacenter = hs.Datacenter
	out.Cluster = hs.Cluster
	out.ClusterId = hs.ClusterID
	out.KeyName = hs.KeyName
	out.Vendor = hs.Vendor
	out.Model = hs.Model
	out.SerialNumber = hs.SerialNumber

	out.ProductFullName = hs.ProductFullName
	out.VirtualSerialNumber = hs.VirtualSerialNumber
	out.EnclosureSerialNumber = hs.EnclosureSerialNumber
	out.Uuid = hs.UUID
	out.PowerState = hs.PowerState
	out.InMaintenanceMode = hs.InMaintenanceMode
	out.CpuModel = hs.CPUModel
	out.CpuMhz = hs.CPUMhz
	out.NumCpuPkgs = int32(hs.NumCPUPkgs)
	out.NumCpuCores = int32(hs.NumCPUCores)
	out.NumCpuThreads = int32(hs.NumCPUThreads)
	out.NumHBAs = hs.NumHBAs
	out.NumNics = hs.NumNics
	out.IpAddress = hs.IPAddress
	out.MacAddress = hs.MacAddress
	out.MemorySizeGB = hs.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = hs.HostMaxVirtualDiskCapacity

	out.PerfCount = hs.PerfCount
	if hs.PerfCount > 0 {

		out.CpuUsageMinimum = math.MaxFloat64
		out.CpuUsageMaximum = 0.0
		out.MemUsageMinimum = math.MaxFloat64
		out.MemUsageMaximum = 0.0
		out.DiskUsageMinimum = math.MaxFloat64
		out.DiskUsageMaximum = 0.0
		out.DiskWriteMinimum = math.MaxFloat64
		out.DiskWriteMaximum = 0.0
		out.DiskReadMinimum = math.MaxFloat64
		out.DiskReadMaximum = 0.0
		out.NetUsageMinimum = math.MaxFloat64
		out.NetUsageMaximum = 0.0

		for _, p := range hs.Perf {

			out.CpuUsageAverage += p.CPUUsage
			if out.CpuUsageMinimum > p.CPUUsage {
				out.CpuUsageMinimum = p.CPUUsage
				out.CpuUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.CpuUsageMaximum < p.CPUUsage {
				out.CpuUsageMaximum = p.CPUUsage
				out.CpuUsageMaximumTime = timestamppb.New(p.Start)
			}


			out.MemUsageAverage += p.MemUsage
			if out.MemUsageMinimum > p.MemUsage {
				out.MemUsageMinimum = p.MemUsage
				out.MemUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.MemUsageMaximum < p.MemUsage {
				out.MemUsageMaximum = p.MemUsage
				out.MemUsageMaximumTime = timestamppb.New(p.Start)
			}

			out.DiskUsageAverage += p.DiskUsage
			if out.DiskUsageMinimum > p.DiskUsage {
				out.DiskUsageMinimum = p.DiskUsage
				out.DiskUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.DiskUsageMaximum < p.DiskUsage {
				out.DiskUsageMaximum = p.DiskUsage
				out.DiskUsageMaximumTime = timestamppb.New(p.Start)
			}

			out.DiskReadAverage += p.DiskRead
			if out.DiskReadMinimum > p.DiskRead {
				out.DiskReadMinimum = p.DiskRead
				out.DiskReadMinimumTime = timestamppb.New(p.Start)
			}
			if out.DiskReadMaximum < p.DiskRead {
				out.DiskReadMaximum = p.DiskRead
				out.DiskReadMaximumTime = timestamppb.New(p.Start)
			}

			out.DiskWriteAverage += p.DiskWrite
			if out.DiskWriteMinimum > p.DiskWrite {
				out.DiskWriteMinimum = p.DiskWrite
				out.DiskWriteMinimumTime = timestamppb.New(p.Start)
			}
			if out.DiskWriteMaximum < p.DiskWrite {
				out.DiskWriteMaximum = p.DiskWrite
				out.DiskWriteMaximumTime = timestamppb.New(p.Start)
			}

			out.NetUsageAverage += p.NetUsage
			if out.NetUsageMinimum > p.NetUsage {
				out.NetUsageMinimum = p.NetUsage
				out.NetUsageMinimumTime = timestamppb.New(p.Start)
			}
			if out.NetUsageMaximum < p.NetUsage {
				out.NetUsageMaximum = p.NetUsage
				out.NetUsageMaximumTime = timestamppb.New(p.Start)
			}

		}
		out.PerfCount = (int64)(len(hs.Perf))
		count := float64(out.PerfCount)
		out.CpuUsageAverage = hs.CPUUsageAverage / count
		out.MemUsageAverage = hs.MemUsageAverage / count
		out.DiskUsageAverage = hs.DiskUsageAverage / count
		out.DiskWriteAverage = hs.DiskWriteAverage / count
		out.DiskReadAverage = hs.DiskReadAverage / count
		out.NetUsageAverage = hs.NetUsageAverage / count
	}
}

// clusterAssignMetrics копирует агрегированные метрики производительности кластера (cpu/mem/disk/net) из cl в out.
// Вспомогательная функция, используется внутри ClusterToMessage*.
func clusterAssignMetrics(cl vsphere.Cluster, out *pb.Cluster) {
	out.CpuUsageMaximum = cl.CPUUsageMaximum
	out.CpuUsageMaximumTime = timestamppb.New(cl.CPUUsageMaximumTime)
	out.CpuUsageAverage = cl.CPUUsageAverage
	out.CpuUsageMinimum = cl.CPUUsageMinimum
	out.CpuUsageMinimumTime = timestamppb.New(cl.CPUUsageMinimumTime)
	out.MemUsageMaximum = cl.MemUsageMaximum
	out.MemUsageMaximumTime = timestamppb.New(cl.MemUsageMaximumTime)
	out.MemUsageAverage = cl.MemUsageAverage
	out.MemUsageMinimum = cl.MemUsageMinimum
	out.MemUsageMinimumTime = timestamppb.New(cl.MemUsageMinimumTime)
	out.DiskUsageMaximum = cl.DiskUsageMaximum
	out.DiskUsageMaximumTime = timestamppb.New(cl.DiskUsageMaximumTime)
	out.DiskUsageAverage = cl.DiskUsageAverage
	out.DiskUsageMinimum = cl.DiskUsageMinimum
	out.DiskUsageMinimumTime = timestamppb.New(cl.DiskUsageMinimumTime)
	out.DiskWriteMaximum = cl.DiskWriteMaximum
	out.DiskWriteMaximumTime = timestamppb.New(cl.DiskWriteMaximumTime)
	out.DiskWriteAverage = cl.DiskWriteAverage
	out.DiskWriteMinimum = cl.DiskWriteMinimum
	out.DiskWriteMinimumTime = timestamppb.New(cl.DiskWriteMinimumTime)
	out.DiskReadMaximum = cl.DiskReadMaximum
	out.DiskReadMaximumTime = timestamppb.New(cl.DiskReadMaximumTime)
	out.DiskReadAverage = cl.DiskReadAverage
	out.DiskReadMinimum = cl.DiskReadMinimum
	out.DiskReadMinimumTime = timestamppb.New(cl.DiskReadMinimumTime)
	out.NetUsageMaximum = cl.NetUsageMaximum
	out.NetUsageMaximumTime = timestamppb.New(cl.NetUsageMaximumTime)
	out.NetUsageAverage = cl.NetUsageAverage
	out.NetUsageMinimum = cl.NetUsageMinimum
	out.NetUsageMinimumTime = timestamppb.New(cl.NetUsageMinimumTime)
}

// ClusterToMessage конвертирует vsphere.Cluster в базовое gRPC-сообщение pb.Cluster (без детальных метрик).
// Принимает cl (значение, не указатель) и out (указатель на заполняемое сообщение).
func ClusterToMessage(cl vsphere.Cluster, out *pb.Cluster) {
	if out == nil {
		return
	}
	cl.CalcPerfomance()
	out.HostFlag = cl.HostFlag
	out.Id = cl.ID
	out.Date = timestamppb.New(cl.Date)
	out.Name = cl.Name
	out.ClusterId = cl.ClusterID
	out.TypeWork = int32(cl.TypeWork)
	out.Vcenter = cl.Vcenter
	out.Datacenter = cl.Datacenter
	out.CpuMhzAvg = cl.CPUMhzAvg
	out.AllNumCpuPkgs = int32(cl.AllNumCPUPkgs)
	out.AllNumCpuCores = int32(cl.AllNumCPUCores)
	out.AllNumCpuThreads = int32(cl.AllNumCPUThreads)
	out.AllMemorySizeGB = cl.AllMemorySizeGB
	out.AllCapacityGb = cl.AllCapacityGb
	out.AllFreeSpaceGb = cl.AllFreeSpaceGb
	out.AllPercentUsed = cl.AllPercentUsed
	out.AllocatedCoreCount = int32(cl.AllocatedCoreCount)
	out.AllocatedMemorySizeGb = int32(cl.AllocatedMemorySizeGb)
	out.CoreTransferRate = cl.CoreTransferRate
	out.MemoryTransferRate = cl.MemoryTransferRate
	clusterAssignMetrics(cl, out)
	out.PerfCount = cl.PerfCount
	out.HostCount = cl.HostCount
	out.VMCount = cl.VMCount
}

// ClusterToMessageMedium конвертирует vsphere.Cluster в pb.Cluster с добавлением агрегированных метрик производительности.
// Принимает cl (значение, не указатель) и out (указатель на заполняемое сообщение).
func ClusterToMessageMedium(cl vsphere.Cluster, out *pb.Cluster) {
	if out == nil {
		return
	}
	cl.CalcPerfomance()
	out.HostFlag = cl.HostFlag
	out.Id = cl.ID
	out.Date = timestamppb.New(cl.Date)
	out.Name = cl.Name
	out.ClusterId = cl.ClusterID
	out.TypeWork = int32(cl.TypeWork)
	out.Vcenter = cl.Vcenter
	out.Datacenter = cl.Datacenter
	out.CpuMhzAvg = cl.CPUMhzAvg
	out.AllNumCpuPkgs = int32(cl.AllNumCPUPkgs)
	out.AllNumCpuCores = int32(cl.AllNumCPUCores)
	out.AllNumCpuThreads = int32(cl.AllNumCPUThreads)
	out.AllMemorySizeGB = cl.AllMemorySizeGB
	out.AllCapacityGb = cl.AllCapacityGb
	out.AllFreeSpaceGb = cl.AllFreeSpaceGb
	out.AllPercentUsed = cl.AllPercentUsed
	out.AllocatedCoreCount = int32(cl.AllocatedCoreCount)
	out.AllocatedMemorySizeGb = int32(cl.AllocatedMemorySizeGb)
	out.CoreTransferRate = cl.CoreTransferRate
	out.MemoryTransferRate = cl.MemoryTransferRate
	clusterAssignMetrics(cl, out)
	out.PerfCount = cl.PerfCount
	out.HostCount = cl.HostCount
	out.VMCount = cl.VMCount
}

// ClusterToMessageFull конвертирует vsphere.Cluster в развёрнутое gRPC-сообщение pb.ClusterFull, включая хосты и хранилища.
// Принимает cl (значение, не указатель) и out (указатель на заполняемое сообщение).
func ClusterToMessageFull(cl vsphere.Cluster, out *pb.ClusterFull) {
	if out == nil {
		return
	}
	cl.CalcPerfomance()
	out.HostFlag = cl.HostFlag
	out.Id = cl.ID
	out.Date = timestamppb.New(cl.Date)
	out.Name = cl.Name
	out.ClusterId = cl.ClusterID
	out.TypeWork = int32(cl.TypeWork)
	out.Vcenter = cl.Vcenter
	out.Datacenter = cl.Datacenter
	out.CpuMhzAvg = cl.CPUMhzAvg
	out.AllNumCpuPkgs = int32(cl.AllNumCPUPkgs)
	out.AllNumCpuCores = int32(cl.AllNumCPUCores)
	out.AllNumCpuThreads = int32(cl.AllNumCPUThreads)
	out.AllMemorySizeGB = cl.AllMemorySizeGB
	out.AllCapacityGb = cl.AllCapacityGb
	out.AllFreeSpaceGb = cl.AllFreeSpaceGb
	out.AllPercentUsed = cl.AllPercentUsed
	out.AllocatedCoreCount = int32(cl.AllocatedCoreCount)
	out.AllocatedMemorySizeGb = int32(cl.AllocatedMemorySizeGb)
	out.CoreTransferRate = cl.CoreTransferRate
	out.MemoryTransferRate = cl.MemoryTransferRate
	out.CpuUsageMaximum = cl.CPUUsageMaximum
	out.CpuUsageMaximumTime = timestamppb.New(cl.CPUUsageMaximumTime)
	out.CpuUsageAverage = cl.CPUUsageAverage
	out.CpuUsageMinimum = cl.CPUUsageMinimum
	out.CpuUsageMinimumTime = timestamppb.New(cl.CPUUsageMinimumTime)
	out.MemUsageMaximum = cl.MemUsageMaximum
	out.MemUsageMaximumTime = timestamppb.New(cl.MemUsageMaximumTime)
	out.MemUsageAverage = cl.MemUsageAverage
	out.MemUsageMinimum = cl.MemUsageMinimum
	out.MemUsageMinimumTime = timestamppb.New(cl.MemUsageMinimumTime)
	out.DiskUsageMaximum = cl.DiskUsageMaximum
	out.DiskUsageMaximumTime = timestamppb.New(cl.DiskUsageMaximumTime)
	out.DiskUsageAverage = cl.DiskUsageAverage
	out.DiskUsageMinimum = cl.DiskUsageMinimum
	out.DiskUsageMinimumTime = timestamppb.New(cl.DiskUsageMinimumTime)
	out.DiskWriteMaximum = cl.DiskWriteMaximum
	out.DiskWriteMaximumTime = timestamppb.New(cl.DiskWriteMaximumTime)
	out.DiskWriteAverage = cl.DiskWriteAverage
	out.DiskWriteMinimum = cl.DiskWriteMinimum
	out.DiskWriteMinimumTime = timestamppb.New(cl.DiskWriteMinimumTime)
	out.DiskReadMaximum = cl.DiskReadMaximum
	out.DiskReadMaximumTime = timestamppb.New(cl.DiskReadMaximumTime)
	out.DiskReadAverage = cl.DiskReadAverage
	out.DiskReadMinimum = cl.DiskReadMinimum
	out.DiskReadMinimumTime = timestamppb.New(cl.DiskReadMinimumTime)
	out.NetUsageMaximum = cl.NetUsageMaximum
	out.NetUsageMaximumTime = timestamppb.New(cl.NetUsageMaximumTime)
	out.NetUsageAverage = cl.NetUsageAverage
	out.NetUsageMinimum = cl.NetUsageMinimum
	out.NetUsageMinimumTime = timestamppb.New(cl.NetUsageMinimumTime)

	out.HostCount = cl.HostCount
	out.VMCount = cl.VMCount

	out.PerfCount = cl.PerfCount
	for _, p := range cl.Perf {
		perf := protocol.CLPerf{}
		perf.Id = p.ID
		perf.Start = timestamppb.New(p.Start)
		perf.CpuUsage = p.CPUUsage
		perf.CpuUtilization = p.CPUUtilization
		perf.MemUsage = p.MemUsage
		perf.DiskUsage = p.DiskUsage
		perf.DiskRead = p.DiskRead
		perf.DiskWrite = p.DiskWrite
		perf.NetUsage = p.NetUsage
		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		out.Perf = append(out.Perf, &perf)
	}
	for _, s := range cl.Storage {
		ds := pb.Datastore{Id: s.ID,
			UID:         s.UID,
			StorageName: s.StorageName,
			CapacityGb:  s.CapacityGb,
			FreeSpaceGb: s.FreeSpaceGb,
			PercentUsed: s.PercentUsed,
			VmKeys:      &pb.VMListKey{}}
		for _, vm := range s.VM {

			ds.VmKeys.Count++
			key := pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName}
			ds.VmKeys.KeyRecs = append(ds.VmKeys.KeyRecs, &key)
		}

		out.Storages = append(out.Storages, &ds)
	}
}

// ClusterStorageToMessage конвертирует vsphere.Datastore в gRPC-сообщение pb.Datastore.
// Принимает s и out; при s == nil или out == nil ничего не делает.
func ClusterStorageToMessage(s *vsphere.Datastore, out *pb.Datastore) {
	if s == nil || out == nil {
		return
	}

	out.Id = s.ID
	out.UID = s.UID
	out.StorageName = s.StorageName
	out.CapacityGb = s.CapacityGb
	out.FreeSpaceGb = s.FreeSpaceGb
	out.PercentUsed = s.PercentUsed
	out.VmKeys = &pb.VMListKey{}
	for _, vm := range s.VM {

		out.VmKeys.Count++
		key := pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName}
		out.VmKeys.KeyRecs = append(out.VmKeys.KeyRecs, &key)
	}
}

// ClusterMetricToMessage конвертирует одну точку метрик кластера vsphere.CLPerf в gRPC-сообщение pb.CLPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func ClusterMetricToMessage(in *vsphere.CLPerf, out *pb.CLPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.ID
	out.Start = timestamppb.New(in.Start)
	out.CpuUsage = in.CPUUsage
	out.CpuUtilization = in.CPUUtilization
	out.MemUsage = in.MemUsage
	out.DiskUsage = in.DiskUsage
	out.DiskRead = in.DiskRead
	out.DiskWrite = in.DiskWrite
	out.NetUsage = in.NetUsage
	out.NetReceived = in.NetReceived
	out.NetTransmitted = in.NetTransmitted
}

// ServerSprRecodToMessage конвертирует запись справочника серверов models.ServerSprRecords в gRPC-сообщение pb.ServerSprRecords.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func ServerSprRecodToMessage(in *models.ServerSprRecords, out *pb.ServerSprRecords) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                         //int64
	out.OldId = in.OldId                     //int64
	out.Type = int32(in.Type)                //int
	out.TypeMicroservice = int32(in.TypeMicroservice)
	out.RoleType = int32(in.RoleType)
	out.TypeWork = int32(in.TypeWork)
	out.CriticalLevel = int32(in.CriticalLevel)
	out.VMSprId = in.VMSprId                 //int64
	out.Stat = int32(in.Stat)
	out.PowerState = int32(in.PowerState)
	out.Active = int32(in.Active)
	out.TimeActiveState = timestamppb.New(in.TimeActiveState)
	out.ActivatorUser = in.ActivatorUser
	out.TimeDeActiveState = timestamppb.New(in.TimeDeActiveState)
	out.DeActivatorUser = in.DeActivatorUser
	out.TimeCreate = timestamppb.New(in.TimeCreate)
	out.CreatorUser = in.CreatorUser
	out.BusinesPrivilage = in.BusinesPrivilage
	out.Name = in.Name
	out.NameFQDN = in.NameFQDN
	out.NameInfo = in.NameInfo
	out.IPAddress = in.IPAddress
	out.IPSharedVIP = in.IPSharedVIP
	out.Network = in.Network
	out.NetworkName = in.NetworkName
	out.CPUCount = in.CPUCount
	out.CPULogical = in.CPULogical
	out.MemSize = in.MemSize
	out.DiskSize = in.DiskSize
	out.Service = in.Service
	out.ServiceId = in.ServiceId
	out.ITSystemKey = in.ITSystemKey
	out.OS = in.OS
	out.Component = in.Component
	out.Owner = in.Owner                         //string    владелец сервера (ответственный за сервер)
	out.BusinessOwner = in.BusinessOwner         //string    владелец от бизнеса
	out.ITOwner = in.ITOwner                     //string    владелец со стороны ИТ
	out.ProductOwner = in.ProductOwner           //string    Владелец продукта
	out.Support = in.Support                     //string    ответственные за поддержку
	out.SupportTeam = in.SupportTeam             //string    Подразделение поддержки
	out.SystemType = in.SystemType               //string    тип системы техническая ...
	out.LicensingType = in.LicensingType         //string    тип лицензирования
	out.SolutionType = in.SolutionType           //string    вариант решения (коробочной)
	out.Users = in.Users                         //string    пользователи системы
	out.Allocation = in.Allocation               //string    аллокация (подразделение)
	out.Anotation = in.Anotation                 //string    описане сервера
	out.Role = in.Role                           //string    роль сервера (сервер баз данных, сервер приложений, хост)
	out.Details = in.Details                     //string    детали по серверу в свободном формате (примечание)
	out.DPCAddress = int32(in.DPCAddress)        //int       ЦОД по справочнику адрес
	out.DPCOffice = int32(in.DPCOffice)          //int       ЦОД по справочнику комната с адресом
	out.SZhistory = in.SZHistory                 //string    номера сз через пробел в хронологическом порядке
	out.Warning = in.Warning                     //string
	out.Level = in.Level                         //float64
	out.Monitoring = int32(in.Monitoring)        //int       0- не указано 1-включен мониторинг 2-отключен мониторинг
	out.Backuped = int32(in.Backuped)            //int       0 - не указано 1-включено резервное копирование 2-отключено резервное копирование
	out.TimeActiveMonitoring = timestamppb.New(in.TimeActiveMonitoring) //time.Time время постановки на мониторинг
	out.MonitoringSZ = in.MonitoringSZ           //string    основание постановки на мониторинг
	out.Id_User = in.Id_User                     //int64
	out.Time_Mfy = timestamppb.New(in.Time_Mfy)   //time.Time
	out.Privilage = int32(in.Privilage)          //int       уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привелегии пользователя)
	out.ClusterId = in.ClusterId
	out.Cluster = in.Cluster
	out.Datacenter = in.Datacenter
	out.Vcenter = in.Vcenter
}

// VMSprRecodToMessage конвертирует запись справочника ВМ models.VMSprRecord в gRPC-сообщение pb.VMSprRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func VMSprRecodToMessage(in *models.VMSprRecord, out *pb.VMSprRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Cluster = in.Cluster
	out.ClusterId = in.ClusterId
	out.Vm_Name = in.Vm_Name
	out.Vm_GuestFullName = in.Vm_GuestFullName
	out.Vm_HostName = in.Vm_HostName // TODO: screenshot shows Vm_Hostname
	out.Vm_PathName = in.Vm_PathName
	out.Vm_PowerState = in.Vm_PowerState
	out.KeyName = in.KeyName
	out.Vm_Uuid = in.Vm_Uuid
	out.Vm_GuestId = in.Vm_GuestId
	out.Vm_Annotation = in.Vm_Annotation
	out.Vm_NumCPU = in.Vm_NumCPU
	out.Vm_NumCores = in.Vm_NumCores
	out.Vm_MemorySizeGB = in.Vm_MemorySizeGB
	out.Vm_IpAddress = in.Vm_IpAddress
	out.Vm_NumEthernetCards = in.Vm_NumEthernetCards
	out.Vm_NumVirtualDisks = in.Vm_NumVirtualDisks
	out.Vm_DiskCommittedGB = in.Vm_DiskCommittedGB
	out.Vm_DiskUncommittedGB = in.Vm_DiskUncommittedGB
	out.Vm_CapacityGb = in.Vm_CapacityGb
	out.Vm_FreeSpaceGb = in.Vm_FreeSpaceGb
	out.Vm_PercentUsed = in.Vm_PercentUsed
	out.CpuUsageMaximum = in.Cpu_usage_maximum
	out.MemUsageMaximum = in.Mem_usage_maximum
	out.DiskUsageMaximum = in.Disk_usage_maximum
	out.NetUsageMaximum = in.Net_usage_maximum
}

// ServerHardwareToMessage конвертирует доменную модель оборудования oneview.ServerHardware в gRPC-сообщение pb.ServerHardware.
// Принимает in и out; при in == nil или out == nil ничего не делает. Заполняет вложенные структуры (Base, ProcessorInfo, MemoryInfo и т.д.).
func ServerHardwareToMessage(in *oneview.ServerHardware, out *pb.ServerHardware) {
	if in == nil || out == nil {
		return
	}
	out.Base = &pb.OVServerHardware{}
	out.Base.Hostname = in.Base.Hostname
	out.Base.HostOsType = int32(in.Base.HostOsType)
	out.Base.MaintenanceMode = in.Base.MaintenanceMode
	out.Base.MemoryMb = int32(in.Base.MemoryMb)
	out.Base.Model = in.Base.Model
	out.Base.Name = in.Base.Name
	out.Base.PartNumber = in.Base.PartNumber
	out.Base.Position = int32(in.Base.Position)
	out.Base.PowerState = in.Base.PowerState
	out.Base.ProcessorCoreCount = int32(in.Base.ProcessorCoreCount)
	out.Base.ProcessorCount = int32(in.Base.ProcessorCount)
	out.Base.ProcessorSpeedMhz = int32(in.Base.ProcessorSpeedMhz)
	out.Base.ProcessorType = in.Base.ProcessorType
	out.Base.SerialNumber = in.Base.SerialNumber.String()
	out.Base.VirtualSerialNumber = in.Base.VirtualSerialNumber.String()
	out.Base.FormFactor = in.Base.FormFactor
	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = in.Base.UUID.String()
	out.Base.OperatingSystem = in.OperatingSystem

	out.Memory = &pb.ServerHardwareMemory{}
	out.Memory.URI = in.Memory.URI
	out.Memory.CollectionState = in.Memory.CollectionState
	out.Memory.Modified = timestamppb.New(in.Memory.Modified)
	out.Memory.Etag = in.Memory.Etag
	out.Memory.Count = int32(in.Memory.Count)
	out.Memory.Name = in.Memory.Name
	for _, m := range in.Memory.Data {
		mm := pb.MemoryModule{}
		mm.BaseModuleType = m.BaseModuleType
		mm.CapacityMiB = int32(m.CapacityMiB)
		mm.DeviceLocator = m.DeviceLocator
		mm.Manufacturer = m.Manufacturer
		mm.MemoryDeviceType = m.MemoryDeviceType
		mm.MemoryLocation = &pb.MemoryLocation{Slot: int32(m.MemoryLocation.Slot), Socket: int32(m.MemoryLocation.Socket)}
		mm.Name = m.Name
		mm.Oem = &pb.OemMemory{
			Hpe: &pb.Hpe{
				DIMMStatus:              m.Oem.Hpe.DIMMStatus,
				MinimumVoltageVoltsX10: int32(m.Oem.Hpe.MinimumVoltageVoltsX10),
			},
		}
		mm.OperatingSpeedMhz = int32(m.OperatingSpeedMhz)
		mm.PartNumber = m.PartNumber
		mm.RankCount = int32(m.RankCount)
		mm.Status = &pb.MemoryStatus{Healt: m.Status.Health, State: m.Status.State}
		out.Memory.Data = append(out.Memory.Data, &mm)
	}

	out.Storage = &pb.ServerHardwareLocalStorage{}
	out.Storage.URI = in.Storage.URI
	out.Storage.CollectionState = in.Storage.CollectionState
	out.Storage.Modified = timestamppb.New(in.Storage.Modified)
	out.Storage.Etag = in.Storage.Etag
	out.Storage.Count = int32(in.Storage.Count)
	out.Storage.Name = in.Storage.Name
	for _, st := range in.Storage.Data {
		ls := pb.LocalStorage{}
		ls.AdapterType = st.AdapterType
		ls.CacheMemorySizeMiB = int32(st.CacheMemorySizeMiB)
		ls.CacheModuleSerialNumber = st.CacheModuleSerialNumber
		ls.CacheModuleStatus = &pb.CacheModuleStatus{Health: st.CacheModuleStatus.Health}
		ls.ControllerBoard = &pb.ControllerBoard{Status: &pb.ControllerSatus{Health: st.ControllerBoard.Status.Health, State: st.ControllerBoard.Status.State}}
		ls.EncryptionCspTestPassed = st.EncryptionCspTestPassed
		ls.EncryptionEnabled = st.EncryptionEnabled
		ls.EncryptionSelfTestPassed = st.EncryptionSelfTestPassed
		ls.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: st.FirmwareVersion.Current.VersionString}}
		ls.Location = st.Location
		ls.Model = st.Model
		ls.Name = st.Name
		ls.SerialNumber = st.SerialNumber
		ls.Status = &pb.Status{Health: st.Status.Health, State: st.Status.State}
		for _, ld := range st.LogicalDrives {
			logical := pb.LocalLogicalDrive{}
			logical.CapacityMiB = ld.CapacityMiB
			logical.LogicalDriveEncryption = ld.LogicalDriveEncryption
			logical.LogicalDriveNumber = int32(ld.LogicalDriveNumber)
			logical.LogicalDriveType = ld.LogicalDriveType
			logical.Raid = ld.Raid
			logical.Status = &pb.Status{Health: ld.Status.Health, State: ld.Status.State}
			for _, dd := range ld.DataDrives {
				drive := pb.LogicalDataDrive{}
				drive.CapacityMiB = dd.CapacityMiB
				drive.DiskDriveStatusReasons = append(drive.DiskDriveStatusReasons, dd.DiskDriveStatusReasons...)
				drive.DiskDriveUse = dd.DiskDriveUse
				drive.EncryptedDrive = dd.EncryptedDrive
				drive.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: dd.FirmwareVersion.Current.VersionString}}
				drive.Location = dd.Location
				drive.LocationFormat = dd.LocationFormat
				drive.MediaType = dd.MediaType
				drive.Model = dd.Model
				drive.SerialNumber = dd.SerialNumber
				drive.Status = &pb.Status{Health: dd.Status.Health, State: dd.Status.State}
				logical.DataDrives = append(logical.DataDrives, &drive)
			}
			ls.LogicalDrives = append(ls.LogicalDrives, &logical)
		}
		for _, pd := range st.PhysicalDrives {
			physical := pb.LocalPhysicalDrive{}
			physical.CapacityMiB = pd.CapacityMiB
			physical.DiskDriveStatusReasons = append(physical.DiskDriveStatusReasons, pd.DiskDriveStatusReasons...)
			physical.DiskDriveUse = pd.DiskDriveUse
			physical.EncryptedDrive = pd.EncryptedDrive
			physical.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
			physical.Location = pd.Location
			physical.LocationFormat = pd.LocationFormat
			physical.MediaType = pd.MediaType
			physical.Model = pd.Model
			physical.SerialNumber = pd.SerialNumber
			physical.Status = &pb.Status{Health: pd.Status.Health, State: pd.Status.State}
			ls.PhysicalDrives = append(ls.PhysicalDrives, &physical)
		}
		for _, enc := range st.StorageEnclosures {
			enclosure := pb.LocalStorageEnclosure{}
			enclosure.DriveBayCount = int32(enc.DriveBayCount)
			enclosure.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: enc.FirmwareVersion.Current.VersionString}}
			enclosure.Location = enc.Location
			enclosure.LocationFormat = enc.LocationFormat
			enclosure.Status = &pb.Status{Health: enc.Status.Health, State: enc.Status.State}
			ls.StorageEnclosures = append(ls.StorageEnclosures, &enclosure)
		}
		out.Storage.Data = append(out.Storage.Data, &ls)
	}

	out.Datacenter = &pb.DatacenterInfo{}
	out.Datacenter.Width = int32(in.Datacenter.Width)
	out.Datacenter.Depth = int32(in.Datacenter.Depth)
	out.Datacenter.Currency = in.Datacenter.Currency
	out.Datacenter.DeratingType = in.Datacenter.DeratingType
	out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
	out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
	for _, c := range in.Datacenter.Contents {
		content := pb.Contents{
			X:        c.X,
			Y:        c.Y,
			Rotation: c.Rotation,
		}
		out.Datacenter.Contents = append(out.Datacenter.Contents, &content)
	}
	out.Datacenter.Category = in.Datacenter.Category
	out.Datacenter.Created = timestamppb.New(in.Datacenter.Created)
	out.Datacenter.Modified = timestamppb.New(in.Datacenter.Modified)
	out.Datacenter.ID = in.Datacenter.ID
	out.Datacenter.UUID = in.Datacenter.UUID
	out.Datacenter.DefaultPowerLineVoltage = int32(in.Datacenter.DefaultPowerLineVoltage)
	out.Datacenter.Name = in.Datacenter.Name
	out.Datacenter.State = in.Datacenter.State
	out.Datacenter.Status = in.Datacenter.Status

	out.Rack = &pb.Racks{}
	out.Rack.UHeight = int32(in.Rack.UHeight)
	out.Rack.Width = int32(in.Rack.Width)
	out.Rack.Height = int32(in.Rack.Height)
	out.Rack.Depth = int32(in.Rack.Depth)
	out.Rack.SerialNumber = in.Rack.SerialNumber
	out.Rack.Created = timestamppb.New(in.Rack.Created)
	out.Rack.Modified = timestamppb.New(in.Rack.Modified)
	out.Rack.ID = in.Rack.ID
	out.Rack.UUID = in.Rack.UUID
	out.Rack.Name = in.Rack.Name
	out.Rack.State = in.Rack.State
	out.Rack.Status = in.Rack.Status

	out.EnclosureInfo = &pb.Enclosure{}
	out.EnclosureInfo.Type = in.EnclosureInfo.Type
	out.EnclosureInfo.Category = in.EnclosureInfo.Category
	out.EnclosureInfo.Created = timestamppb.New(in.EnclosureInfo.Created)
	out.EnclosureInfo.Modified = timestamppb.New(in.EnclosureInfo.Modified)
	out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState
	out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason
	out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType
	out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel
	out.EnclosureInfo.UUID = in.EnclosureInfo.UUID
	out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber
	out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber
	out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState
	out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent
	out.EnclosureInfo.DeviceBayCount = int32(in.EnclosureInfo.DeviceBayCount)
	for _, bay := range in.EnclosureInfo.DeviceBays {
		db := pb.DeviceBayInfo{}
		db.Type = bay.Type
		db.BayNumber = int32(bay.BayNumber)
		db.DevicePresence = bay.DevicePresence
		db.CoveredByDevice = bay.CoveredByDevice
		db.URI = bay.URI
		db.Category = bay.Category
		db.DeviceBayType = bay.DeviceBayType
		db.DeviceFormFactor = bay.DeviceFormFactor
		db.BayPowerState = bay.BayPowerState
		db.ChangeState = bay.ChangeState
		out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, &db)
	}
	out.EnclosureInfo.InterconnectBayCount = int32(in.EnclosureInfo.InterconnectBayCount)
	for _, bay := range in.EnclosureInfo.InterconnectBays {
		ib := pb.InterconnectBayInfo{}
		ib.BayNumber = int32(bay.BayNumber)
		ib.InterconnectModel = bay.InterconnectModel
		ib.SerialNumber = bay.SerialNumber
		ib.InterconnectBayType = bay.InterconnectBayType
		ib.ChangeState = bay.ChangeState
		ib.BayPowerState = bay.BayPowerState
		out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, &ib)
	}
	out.EnclosureInfo.FanBayCount = int32(in.EnclosureInfo.FanBayCount)
	for _, bay := range in.EnclosureInfo.FanBays {
		fb := pb.FanBayInfo{}
		fb.BayNumber = int32(bay.BayNumber)
		fb.DevicePresence = bay.DevicePresence
		fb.DeviceRequired = bay.DeviceRequired
		fb.Status = bay.Status
		fb.Model = bay.Model
		fb.PartNumber = bay.PartNumber
		fb.SparePartNumber = bay.SparePartNumber
		fb.FanBayType = bay.FanBayType
		fb.ChangeState = bay.ChangeState
		fb.State = bay.State
		out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, &fb)
	}
	out.EnclosureInfo.PowerSupplyBayCount = int32(in.EnclosureInfo.PowerSupplyBayCount)
	for _, bay := range in.EnclosureInfo.PowerSupplyBays {
		ps := pb.PowerSupplyBayInfo{}
		ps.BayNumber = int32(bay.BayNumber)
		ps.DevicePresence = bay.DevicePresence
		ps.Status = bay.Status
		ps.Model = bay.Model
		ps.SerialNumber = bay.SerialNumber
		ps.PartNumber = bay.PartNumber
		ps.SparePartNumber = bay.SparePartNumber
		ps.PowerSupplyBayType = bay.PowerSupplyBayType
		ps.ChangeState = bay.ChangeState
		out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, &ps)
	}
	out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged
	out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware
	for _, bay := range in.EnclosureInfo.ManagerBays {
		mb := pb.ManagerBayInfo{}
		mb.BayNumber = int32(bay.BayNumber)
		mb.ManagerType = bay.ManagerType
		mb.BayPowerState = bay.BayPowerState
		mb.FwVersion = bay.FwVersion
		mb.DevicePresence = bay.DevicePresence
		mb.Role = bay.Role
		mb.IPAddress = bay.IPAddress
		mb.ChangeState = bay.ChangeState
		mb.FwBuildDate = bay.FwBuildDate
		mb.DhcpIpv6Enable = bay.DhcpIpv6Enable
		mb.FqdnHostName = bay.FqdnHostName
		mb.DhcpEnable = bay.DhcpEnable
		out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, &mb)
	}
	out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState
	out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState
	out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType
	out.EnclosureInfo.Status = in.EnclosureInfo.Status
	out.EnclosureInfo.Name = in.EnclosureInfo.Name
	out.EnclosureInfo.State = in.EnclosureInfo.State
	out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP
	out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP
	out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag
	out.EnclosureInfo.RackName = in.EnclosureInfo.RackName
	out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID
	out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName
	out.EnclosureInfo.VcmMode = fmt.Sprint(in.EnclosureInfo.VcmMode)
	out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL
	out.EnclosureInfo.OaBays = int32(in.EnclosureInfo.OaBays)
	out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState

	out.BayInfo = &pb.DeviceBayInfo{}
	out.BayInfo.Type = in.BayInfo.Type
	out.BayInfo.BayNumber = int32(in.BayInfo.BayNumber)
	out.BayInfo.DevicePresence = in.BayInfo.DevicePresence
	out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice
	out.BayInfo.URI = in.BayInfo.URI
	out.BayInfo.Category = in.BayInfo.Category
	out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType
	out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor
	out.BayInfo.BayPowerState = in.BayInfo.BayPowerState
	out.BayInfo.ChangeState = in.BayInfo.ChangeState
	out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile
	out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile
	out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile
	out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile
}

// MessageToServerHardware выполняет обратную конвертацию gRPC-сообщения pb.ServerHardware в доменную модель oneview.ServerHardware.
// Принимает in и out; при in == nil или out == nil ничего не делает.
// В отличие от ServerHardwareToMessage не выделяет вложенные структуры out.Base/out.ProcessorInfo и т.п. заново — вызывающий код должен
// гарантировать, что out и его вложенные поля уже инициализированы, иначе возможен nil pointer dereference.
// Примечание: в grpcserver эта функция сейчас не используется (симметричный конвертер есть в grpcclient), оставлена для возможного будущего использования на стороне сервера.
func MessageToServerHardware(in *pb.ServerHardware, out *oneview.ServerHardware) {
	if in == nil || out == nil {
		return
	}
	out.Base.Hostname = in.Base.Hostname
	out.Base.HostOsType = int(in.Base.HostOsType)
	out.Base.MaintenanceMode = in.Base.MaintenanceMode
	out.Base.MemoryMb = int(in.Base.MemoryMb)
	out.Base.Model = in.Base.Model
	out.Base.Name = in.Base.Name
	out.Base.PartNumber = in.Base.PartNumber
	out.Base.Position = int(in.Base.Position)
	out.Base.PowerState = in.Base.PowerState
	out.Base.ProcessorCoreCount = int(in.Base.ProcessorCoreCount)
	out.Base.ProcessorCount = int(in.Base.ProcessorCount)
	out.Base.ProcessorSpeedMhz = int(in.Base.ProcessorSpeedMhz)
	out.Base.ProcessorType = in.Base.ProcessorType
	out.Base.SerialNumber = utils.Nstring(in.Base.SerialNumber)
	out.Base.VirtualSerialNumber = utils.Nstring(in.Base.VirtualSerialNumber)
	out.Base.FormFactor = in.Base.FormFactor
	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = utils.Nstring(in.Base.UUID)
	out.OperatingSystem = in.Base.OperatingSystem

	if in.Memory != nil {
		out.Memory.URI = in.Memory.URI
		out.Memory.CollectionState = in.Memory.CollectionState
		out.Memory.Modified = in.Memory.Modified.AsTime()
		out.Memory.Etag = in.Memory.Etag
		out.Memory.Count = int(in.Memory.Count)
		out.Memory.Name = in.Memory.Name
		for _, m := range in.Memory.Data {
			mm := oneview.MemoryModule{}
			mm.BaseModuleType = m.BaseModuleType
			mm.CapacityMiB = int(m.CapacityMiB)
			mm.DeviceLocator = m.DeviceLocator
			mm.Manufacturer = m.Manufacturer
			mm.MemoryDeviceType = m.MemoryDeviceType
			if m.MemoryLocation != nil {
				mm.MemoryLocation = oneview.MemoryLocation{Slot: int(m.MemoryLocation.Slot), Socket: int(m.MemoryLocation.Socket)}
			}
			mm.Name = m.Name
			if m.Oem != nil && m.Oem.Hpe != nil {
				mm.Oem.Hpe.DIMMStatus = m.Oem.Hpe.DIMMStatus
				mm.Oem.Hpe.MinimumVoltageVoltsX10 = int(m.Oem.Hpe.MinimumVoltageVoltsX10)
			}
			mm.OperatingSpeedMhz = int(m.OperatingSpeedMhz)
			mm.PartNumber = m.PartNumber
			mm.RankCount = int(m.RankCount)
			if m.Status != nil {
				mm.Status = oneview.MemoryStatus{Health: m.Status.Healt, State: m.Status.State}
			}
			out.Memory.Data = append(out.Memory.Data, mm)
		}
	}

	if in.Storage != nil {
		out.Storage.URI = in.Storage.URI
		out.Storage.CollectionState = in.Storage.CollectionState
		out.Storage.Modified = in.Storage.Modified.AsTime()
		out.Storage.Etag = in.Storage.Etag
		out.Storage.Count = int(in.Storage.Count)
		out.Storage.Name = in.Storage.Name
		for _, st := range in.Storage.Data {
			ls := oneview.LocalStorage{}
			ls.AdapterType = st.AdapterType
			ls.CacheMemorySizeMiB = int(st.CacheMemorySizeMiB)
			ls.CacheModuleSerialNumber = st.CacheModuleSerialNumber
			if st.CacheModuleStatus != nil {
				ls.CacheModuleStatus = oneview.CacheModuleStatus{Health: st.CacheModuleStatus.Health}
			}
			if st.ControllerBoard != nil && st.ControllerBoard.Status != nil {
				ls.ControllerBoard = oneview.ControllerBoard{Status: oneview.ControllerSatus{Health: st.ControllerBoard.Status.Health, State: st.ControllerBoard.Status.State}}
			}
			ls.EncryptionCspTestPassed = st.EncryptionCspTestPassed
			ls.EncryptionEnabled = st.EncryptionEnabled
			ls.EncryptionSelfTestPassed = st.EncryptionSelfTestPassed
			if st.FirmwareVersion != nil && st.FirmwareVersion.Current != nil {
				ls.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: st.FirmwareVersion.Current.VersionString}}
			}
			ls.Location = st.Location
			ls.Model = st.Model
			ls.Name = st.Name
			ls.SerialNumber = st.SerialNumber
			if st.Status != nil {
				ls.Status = oneview.Status{Health: st.Status.Health, State: st.Status.State}
			}
			for _, ld := range st.LogicalDrives {
				logical := oneview.LocalLogicalDrive{}
				logical.CapacityMiB = ld.CapacityMiB
				logical.LogicalDriveEncryption = ld.LogicalDriveEncryption
				logical.LogicalDriveNumber = int(ld.LogicalDriveNumber)
				logical.LogicalDriveType = ld.LogicalDriveType
				logical.Raid = ld.Raid
				logical.Status = oneview.Status{}
				if ld.Status != nil {
					logical.Status = oneview.Status{Health: ld.Status.Health, State: ld.Status.State}
				}
				for _, dd := range ld.DataDrives {
					drive := oneview.LogicalDataDrive{}
					drive.CapacityMiB = dd.CapacityMiB
					drive.DiskDriveStatusReasons = append(drive.DiskDriveStatusReasons, dd.DiskDriveStatusReasons...)
					drive.DiskDriveUse = dd.DiskDriveUse
					drive.EncryptedDrive = dd.EncryptedDrive
					if dd.FirmwareVersion != nil && dd.FirmwareVersion.Current != nil {
						drive.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: dd.FirmwareVersion.Current.VersionString}}
					}
					drive.Location = dd.Location
					drive.LocationFormat = dd.LocationFormat
					drive.MediaType = dd.MediaType
					drive.Model = dd.Model
					drive.SerialNumber = dd.SerialNumber
					if dd.Status != nil {
						drive.Status = oneview.Status{Health: dd.Status.Health, State: dd.Status.State}
					}
					logical.DataDrives = append(logical.DataDrives, drive)
				}
				ls.LogicalDrives = append(ls.LogicalDrives, logical)
			}
			for _, pd := range st.PhysicalDrives {
				physical := oneview.LocalPhysicalDrive{}
				physical.CapacityMiB = pd.CapacityMiB
				physical.DiskDriveStatusReasons = append(physical.DiskDriveStatusReasons, pd.DiskDriveStatusReasons...)
				physical.DiskDriveUse = pd.DiskDriveUse
				physical.EncryptedDrive = pd.EncryptedDrive
				if pd.FirmwareVersion != nil && pd.FirmwareVersion.Current != nil {
					physical.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
				}
				physical.Location = pd.Location
				physical.LocationFormat = pd.LocationFormat
				physical.MediaType = pd.MediaType
				physical.Model = pd.Model
				physical.SerialNumber = pd.SerialNumber
				if pd.Status != nil {
					physical.Status = oneview.Status{Health: pd.Status.Health, State: pd.Status.State}
				}
				ls.PhysicalDrives = append(ls.PhysicalDrives, physical)
			}
			for _, enc := range st.StorageEnclosures {
				enclosure := oneview.LocalStorageEnclosure{}
				enclosure.DriveBayCount = int(enc.DriveBayCount)
				if enc.FirmwareVersion != nil && enc.FirmwareVersion.Current != nil {
					enclosure.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: enc.FirmwareVersion.Current.VersionString}}
				}
				enclosure.Location = enc.Location
				enclosure.LocationFormat = enc.LocationFormat
				if enc.Status != nil {
					enclosure.Status = oneview.Status{Health: enc.Status.Health, State: enc.Status.State}
				}
				ls.StorageEnclosures = append(ls.StorageEnclosures, enclosure)
			}
			out.Storage.Data = append(out.Storage.Data, ls)
		}
	}

	if in.Datacenter != nil {
		out.Datacenter.Width = int(in.Datacenter.Width)
		out.Datacenter.Depth = int(in.Datacenter.Depth)
		out.Datacenter.Currency = in.Datacenter.Currency
		out.Datacenter.DeratingType = in.Datacenter.DeratingType
		out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
		out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
		for _, c := range in.Datacenter.Contents {
			content := oneview.Content{
				X:        c.X,
				Y:        c.Y,
				Rotation: c.Rotation,
			}
			out.Datacenter.Contents = append(out.Datacenter.Contents, content)
		}
		out.Datacenter.Category = in.Datacenter.Category
		if in.Datacenter.Created != nil {
			out.Datacenter.Created = in.Datacenter.Created.AsTime()
		}
		if in.Datacenter.Modified != nil {
			out.Datacenter.Modified = in.Datacenter.Modified.AsTime()
		}
		out.Datacenter.ID = in.Datacenter.ID
		out.Datacenter.UUID = in.Datacenter.UUID
		out.Datacenter.DefaultPowerLineVoltage = int(in.Datacenter.DefaultPowerLineVoltage)
		out.Datacenter.Name = in.Datacenter.Name
		out.Datacenter.State = in.Datacenter.State
		out.Datacenter.Status = in.Datacenter.Status
	}

	if in.Rack != nil {
		out.Rack.UHeight = int(in.Rack.UHeight)
		out.Rack.Width = int(in.Rack.Width)
		out.Rack.Height = int(in.Rack.Height)
		out.Rack.Depth = int(in.Rack.Depth)
		out.Rack.SerialNumber = in.Rack.SerialNumber
		if in.Rack.Created != nil {
			out.Rack.Created = in.Rack.Created.AsTime()
		}
		if in.Rack.Modified != nil {
			out.Rack.Modified = in.Rack.Modified.AsTime()
		}
		out.Rack.ID = in.Rack.ID
		out.Rack.UUID = in.Rack.UUID
		out.Rack.Name = in.Rack.Name
		out.Rack.State = in.Rack.State
		out.Rack.Status = in.Rack.Status
	}

	if in.EnclosureInfo != nil {
		out.EnclosureInfo.Type = in.EnclosureInfo.Type
		out.EnclosureInfo.Category = in.EnclosureInfo.Category
		if in.EnclosureInfo.Created != nil {
			out.EnclosureInfo.Created = in.EnclosureInfo.Created.AsTime()
		}
		if in.EnclosureInfo.Modified != nil {
			out.EnclosureInfo.Modified = in.EnclosureInfo.Modified.AsTime()
		}
		out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState
		out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason
		out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType
		out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel
		out.EnclosureInfo.UUID = in.EnclosureInfo.UUID
		out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber
		out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber
		out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState
		out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent
		out.EnclosureInfo.DeviceBayCount = int(in.EnclosureInfo.DeviceBayCount)
		for _, bay := range in.EnclosureInfo.DeviceBays {
			db := oneview.DeviceBayInfo{}
			db.Type = bay.Type
			db.BayNumber = int(bay.BayNumber)
			db.DevicePresence = bay.DevicePresence
			db.CoveredByDevice = bay.CoveredByDevice
			db.URI = bay.URI
			db.Category = bay.Category
			db.DeviceBayType = bay.DeviceBayType
			db.DeviceFormFactor = bay.DeviceFormFactor
			db.BayPowerState = bay.BayPowerState
			db.ChangeState = bay.ChangeState
			out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, db)
		}
		out.EnclosureInfo.InterconnectBayCount = int(in.EnclosureInfo.InterconnectBayCount)
		for _, bay := range in.EnclosureInfo.InterconnectBays {
			ib := oneview.InterconnectBayInfo{}
			ib.BayNumber = int(bay.BayNumber)
			ib.InterconnectModel = bay.InterconnectModel
			ib.SerialNumber = bay.SerialNumber
			ib.InterconnectBayType = bay.InterconnectBayType
			ib.ChangeState = bay.ChangeState
			ib.BayPowerState = bay.BayPowerState
			out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, ib)
		}
		out.EnclosureInfo.FanBayCount = int(in.EnclosureInfo.FanBayCount)
		for _, bay := range in.EnclosureInfo.FanBays {
			fb := oneview.FanBayInfo{}
			fb.BayNumber = int(bay.BayNumber)
			fb.DevicePresence = bay.DevicePresence
			fb.DeviceRequired = bay.DeviceRequired
			fb.Status = bay.Status
			fb.Model = bay.Model
			fb.PartNumber = bay.PartNumber
			fb.SparePartNumber = bay.SparePartNumber
			fb.FanBayType = bay.FanBayType
			fb.ChangeState = bay.ChangeState
			fb.State = bay.State
			out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, fb)
		}
		out.EnclosureInfo.PowerSupplyBayCount = int(in.EnclosureInfo.PowerSupplyBayCount)
		for _, bay := range in.EnclosureInfo.PowerSupplyBays {
			ps := oneview.PowerSupplyBayInfo{}
			ps.BayNumber = int(bay.BayNumber)
			ps.DevicePresence = bay.DevicePresence
			ps.Status = bay.Status
			ps.Model = bay.Model
			ps.SerialNumber = bay.SerialNumber
			ps.PartNumber = bay.PartNumber
			ps.SparePartNumber = bay.SparePartNumber
			ps.PowerSupplyBayType = bay.PowerSupplyBayType
			ps.ChangeState = bay.ChangeState
			out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, ps)
		}
		out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged
		out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware
		for _, bay := range in.EnclosureInfo.ManagerBays {
			mb := oneview.ManagerBayInfo{}
			mb.BayNumber = int(bay.BayNumber)
			mb.ManagerType = bay.ManagerType
			mb.BayPowerState = bay.BayPowerState
			mb.FwVersion = bay.FwVersion
			mb.DevicePresence = bay.DevicePresence
			mb.Role = bay.Role
			mb.IPAddress = bay.IPAddress
			mb.ChangeState = bay.ChangeState
			mb.FwBuildDate = bay.FwBuildDate
			mb.DhcpIpv6Enable = bay.DhcpIpv6Enable
			mb.FqdnHostName = bay.FqdnHostName
			mb.DhcpEnable = bay.DhcpEnable
			out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, mb)
		}
		out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState
		out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState
		out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType
		out.EnclosureInfo.Status = in.EnclosureInfo.Status
		out.EnclosureInfo.Name = in.EnclosureInfo.Name
		out.EnclosureInfo.State = in.EnclosureInfo.State
		out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP
		out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP
		out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag
		out.EnclosureInfo.RackName = in.EnclosureInfo.RackName
		out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID
		out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName
		if b, err := strconv.ParseBool(in.EnclosureInfo.VcmMode); err == nil {
			out.EnclosureInfo.VcmMode = b
		}
		out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL
		out.EnclosureInfo.OaBays = int(in.EnclosureInfo.OaBays)
		out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState
	}

	if in.BayInfo != nil {
		out.BayInfo.Type = in.BayInfo.Type
		out.BayInfo.BayNumber = int(in.BayInfo.BayNumber)
		out.BayInfo.DevicePresence = in.BayInfo.DevicePresence
		out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice
		out.BayInfo.URI = in.BayInfo.URI
		out.BayInfo.Category = in.BayInfo.Category
		out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType
		out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor
		out.BayInfo.BayPowerState = in.BayInfo.BayPowerState
		out.BayInfo.ChangeState = in.BayInfo.ChangeState
		out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile
		out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile
		out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile
		out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile
	}
}

// ClusterSprRecordsToMessage конвертирует models.ClusterSprRecords в gRPC-сообщение pb.ClusterSprRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func ClusterSprRecordsToMessage(in *models.ClusterSprRecords, out *pb.ClusterSprRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Name = in.Name
	out.ClusterId = in.ClusterId
	out.Annotation = in.Annotation
	//out.Production = int32(in.Production) //признак продуктового кластера 0-не заполнено 1-продуктовый с бизнес приложениями 2-тестовые среды
	out.TypeWork = int32(in.TypeWork)
	out.Stat = int32(in.Stat)
	out.Id_User = in.Id_User
	out.Time_Mfy = timestamppb.New(in.Time_Mfy)
}

// MessageToClusterSprRecords выполняет обратную конвертацию pb.ClusterSprRecord в models.ClusterSprRecords.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToClusterSprRecords(in *pb.ClusterSprRecord, out *models.ClusterSprRecords) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Name = in.Name
	out.ClusterId = in.ClusterId
	out.Annotation = in.Annotation
	//out.Production = int(in.Production) //признак продуктового кластера 0-не заполнено 1-продуктовый с бизнес приложениями 2-тестовые среды
	out.TypeWork = int(in.TypeWork)
	out.Stat = int(in.Stat)
	out.Id_User = in.Id_User
	out.Time_Mfy = in.Time_Mfy.AsTime()
}

// ClusterRecordToMessage конвертирует models.ClusterRecord в gRPC-сообщение pb.ClusterRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func ClusterRecordToMessage(in *models.ClusterRecord, out *pb.ClusterRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                               //int64
	out.ClusterId = in.ClusterId               //int64
	out.Date = timestamppb.New(in.Date)          //time.Time
	out.Vcenter = in.Vcenter                     //string
	out.Datacenter = in.Datacenter               //string
	out.Name = in.Name                           //string
	out.CpuMhzAvg = in.CpuMhzAvg                 //int32
	out.AllNumCpuPkgs = int32(in.AllNumCpuPkgs)  //int16
	out.AllNumCpuCores = int32(in.AllNumCpuCores)
	out.AllNumCpuThreads = int32(in.AllNumCpuThreads)
	out.AllMemorySizeGB = in.AllMemorySizeGB     //float64
	out.AllCapacityGb = in.AllCapacityGb
	out.AllFreeSpaceGb = in.AllFreeSpaceGb
	out.AllPercentUsed = in.AllPercentUsed
	out.HostCount = in.HostCount
	out.VMCount = in.VMCount
	out.StorageCount = in.StorageCount
}

// MessageToClusterRecord выполняет обратную конвертацию pb.ClusterRecord в models.ClusterRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToClusterRecord(in *pb.ClusterRecord, out *models.ClusterRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ClusterId = in.ClusterId
	out.Date = in.Date.AsTime()
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Name = in.Name
	out.CpuMhzAvg = in.CpuMhzAvg
	out.AllNumCpuPkgs = int16(in.AllNumCpuPkgs)
	out.AllNumCpuCores = int16(in.AllNumCpuCores)
	out.AllNumCpuThreads = int16(in.AllNumCpuThreads)
	out.AllMemorySizeGB = in.AllMemorySizeGB
	out.AllCapacityGb = in.AllCapacityGb
	out.AllFreeSpaceGb = in.AllFreeSpaceGb
	out.AllPercentUsed = in.AllPercentUsed
	out.HostCount = in.HostCount
	out.VMCount = in.VMCount
	out.StorageCount = in.StorageCount
}

// HostRecordToMessage конвертирует models.HostRecord в gRPC-сообщение pb.HostRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func HostRecordToMessage(in *models.HostRecord, out *pb.HostRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.ClusterId = in.ClusterId
	out.Date = timestamppb.New(in.Date)
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Cluster = in.Cluster
	out.Name = in.Name
	out.Vendor = in.Vendor
	out.Model = in.Model
	out.CpuModel = in.CpuModel
	out.CpuMhz = in.CpuMhz
	out.NumCpuPkgs = int32(in.NumCpuPkgs)
	out.NumCpuCores = int32(in.NumCpuCores)
	out.NumCpuThreads = int32(in.NumCpuThreads)
	out.NumNics = in.NumNics
	out.NumHBAs = in.NumHBAs
	out.MemorySizeGB = in.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity
}

// MessageToHostRecord выполняет обратную конвертацию pb.HostRecord в models.HostRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToHostRecord(in *pb.HostRecord, out *models.HostRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.ClusterId = in.ClusterId
	out.Date = in.Date.AsTime()
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Cluster = in.Cluster
	out.Name = in.Name
	out.Vendor = in.Vendor
	out.Model = in.Model
	out.CpuModel = in.CpuModel
	out.CpuMhz = in.CpuMhz
	out.NumCpuPkgs = int16(in.NumCpuPkgs)
	out.NumCpuCores = int16(in.NumCpuCores)
	out.NumCpuThreads = int16(in.NumCpuThreads)
	out.NumNics = in.NumNics
	out.NumHBAs = in.NumHBAs
	out.MemorySizeGB = in.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity
}

// VMRecordToMessage конвертирует models.VMRecord в gRPC-сообщение pb.VMRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func VMRecordToMessage(in *models.VMRecord, out *pb.VMRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.VMSprId = in.VMsprId
	out.ClusterId = in.ClusterId
	out.Date = timestamppb.New(in.Date)
	out.Vm_NumCPU = in.Vm_NumCPU
	out.Vm_NumCores = in.Vm_NumCores
	out.Vm_MemorySizeGB = in.Vm_MemorySizeGB
	out.Vm_IpAddress = in.Vm_IpAddress
	out.Vm_NumEthernetCards = in.Vm_NumEthernetCards
	out.Vm_NumVirtualDisks = in.Vm_NumVirtualDisks
	out.Vm_DiskCommittedGB = in.Vm_DiskCommittedGB
	out.Vm_DiskUncommittedGB = in.Vm_DiskUncommittedGB
	out.Vm_CapacityGb = in.Vm_CapacityGb
	out.Vm_FreeSpaceGb = in.Vm_FreeSpaceGb
	out.Vm_PercentUsed = in.Vm_PercentUsed
	out.CpuUsageMaximum = in.Cpu_usage_maximum
	out.CpuUsageMaximumTime = timestamppb.New(in.Cpu_usage_maximum_time)
	out.CpuUsageAverage = in.Cpu_usage_average
	out.CpuUsageMinimum = in.Cpu_usage_minimum
	out.CpuUsageMinimumTime = timestamppb.New(in.Cpu_usage_minimum_time)
	out.MemUsageMaximum = in.Mem_usage_maximum
	out.MemUsageMaximumTime = timestamppb.New(in.Mem_usage_maximum_time)
	out.MemUsageAverage = in.Mem_usage_average
	out.MemUsageMinimum = in.Mem_usage_minimum
	out.MemUsageMinimumTime = timestamppb.New(in.Mem_usage_minimum_time)
	out.DiskUsageMaximum = in.Disk_usage_maximum
	out.DiskUsageMaximumTime = timestamppb.New(in.Disk_usage_maximum_time)
	out.DiskUsageAverage = in.Disk_usage_average
	out.DiskUsageMinimum = in.Disk_usage_minimum
	out.DiskUsageMinimumTime = timestamppb.New(in.Disk_usage_minimum_time)
	out.DiskTotalReadLatencyMaximum = in.Disk_totalReadLatency_maximum
	out.DiskTotalReadLatencyMaximumTime = timestamppb.New(in.Disk_totalReadLatency_maximum_time)
	out.DiskTotalReadLatencyAverage = in.Disk_totalReadLatency_average
	out.DiskTotalReadLatencyMinimum = in.Disk_totalReadLatency_minimum
	out.DiskTotalReadLatencyMinimumTime = timestamppb.New(in.Disk_totalReadLatency_minimum_time)
	out.DiskTotalWriteLatencyMaximum = in.Disk_totalWriteLatency_maximum
	out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(in.Disk_totalWriteLatency_maximum_time)
	out.DiskTotalWriteLatencyAverage = in.Disk_totalWriteLatency_average
	out.DiskTotalWriteLatencyMinimum = in.Disk_totalWriteLatency_minimum
	out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(in.Disk_totalWriteLatency_minimum_time)
	out.DiskWriteMaximum = in.Disk_write_maximum
	out.DiskWriteMaximumTime = timestamppb.New(in.Disk_write_maximum_time)
	out.DiskWriteAverage = in.Disk_write_average
	out.DiskWriteMinimum = in.Disk_write_minimum
	out.DiskWriteMinimumTime = timestamppb.New(in.Disk_write_minimum_time)
	out.DiskReadMaximum = in.Disk_read_maximum
	out.DiskReadMaximumTime = timestamppb.New(in.Disk_read_maximum_time)
	out.DiskReadAverage = in.Disk_read_average
	out.DiskReadMinimum = in.Disk_read_minimum
	out.DiskReadMinimumTime = timestamppb.New(in.Disk_read_minimum_time)
	out.NetUsageMaximum = in.Net_usage_maximum
	out.NetUsageMaximumTime = timestamppb.New(in.Net_usage_maximum_time)
	out.NetUsageAverage = in.Net_usage_average
	out.NetUsageMinimum = in.Net_usage_minimum
	out.NetUsageMinimumTime = timestamppb.New(in.Net_usage_minimum_time)
	out.CpuBilling = in.CpuBilling
	out.MemBilling = in.MemBilling
	out.DiskBilling = in.DiskBilling
	out.NetBilling = in.NetBilling
}

// MessageToVMRecord выполняет обратную конвертацию pb.VMRecord в models.VMRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToVMRecord(in *pb.VMRecord, out *models.VMRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.VMsprId = in.VMSprId
	out.ClusterId = in.ClusterId
	out.Date = in.Date.AsTime()
	out.Vm_NumCPU = in.Vm_NumCPU
	out.Vm_NumCores = in.Vm_NumCores
	out.Vm_MemorySizeGB = in.Vm_MemorySizeGB
	out.Vm_IpAddress = in.Vm_IpAddress
	out.Vm_NumEthernetCards = in.Vm_NumEthernetCards
	out.Vm_NumVirtualDisks = in.Vm_NumVirtualDisks
	out.Vm_DiskCommittedGB = in.Vm_DiskCommittedGB
	out.Vm_DiskUncommittedGB = in.Vm_DiskUncommittedGB
	out.Vm_CapacityGb = in.Vm_CapacityGb
	out.Vm_FreeSpaceGb = in.Vm_FreeSpaceGb
	out.Vm_PercentUsed = in.Vm_PercentUsed
	out.Cpu_usage_maximum = in.CpuUsageMaximum
	out.Cpu_usage_maximum_time = in.CpuUsageMaximumTime.AsTime()
	out.Cpu_usage_average = in.CpuUsageAverage
	out.Cpu_usage_minimum = in.CpuUsageMinimum
	out.Cpu_usage_minimum_time = in.CpuUsageMinimumTime.AsTime()
	out.Mem_usage_maximum = in.MemUsageMaximum
	out.Mem_usage_maximum_time = in.MemUsageMaximumTime.AsTime()
	out.Mem_usage_average = in.MemUsageAverage
	out.Mem_usage_minimum = in.MemUsageMinimum
	out.Mem_usage_minimum_time = in.MemUsageMinimumTime.AsTime()
	out.Disk_usage_maximum = in.DiskUsageMaximum
	out.Disk_usage_maximum_time = in.DiskUsageMaximumTime.AsTime()
	out.Disk_usage_average = in.DiskUsageAverage
	out.Disk_usage_minimum = in.DiskUsageMinimum
	out.Disk_usage_minimum_time = in.DiskUsageMinimumTime.AsTime()
	out.Disk_totalReadLatency_maximum = in.DiskTotalReadLatencyMaximum
	out.Disk_totalReadLatency_maximum_time = in.DiskTotalReadLatencyMaximumTime.AsTime()
	out.Disk_totalReadLatency_average = in.DiskTotalReadLatencyAverage
	out.Disk_totalReadLatency_minimum = in.DiskTotalReadLatencyMinimum
	out.Disk_totalReadLatency_minimum_time = in.DiskTotalReadLatencyMinimumTime.AsTime()
	out.Disk_totalWriteLatency_maximum = in.DiskTotalWriteLatencyMaximum
	out.Disk_totalWriteLatency_maximum_time = in.DiskTotalWriteLatencyMaximumTime.AsTime()
	out.Disk_totalWriteLatency_average = in.DiskTotalWriteLatencyAverage
	out.Disk_totalWriteLatency_minimum = in.DiskTotalWriteLatencyMinimum
	out.Disk_totalWriteLatency_minimum_time = in.DiskTotalWriteLatencyMinimumTime.AsTime()
	out.Disk_write_maximum = in.DiskWriteMaximum
	out.Disk_write_maximum_time = in.DiskWriteMaximumTime.AsTime()
	out.Disk_write_average = in.DiskWriteAverage
	out.Disk_write_minimum = in.DiskWriteMinimum
	out.Disk_write_minimum_time = in.DiskWriteMinimumTime.AsTime()
	out.Disk_read_maximum = in.DiskReadMaximum
	out.Disk_read_maximum_time = in.DiskReadMaximumTime.AsTime()
	out.Disk_read_average = in.DiskReadAverage
	out.Disk_read_minimum = in.DiskReadMinimum
	out.Disk_read_minimum_time = in.DiskReadMinimumTime.AsTime()
	out.Net_usage_maximum = in.NetUsageMaximum
	out.Net_usage_maximum_time = in.NetUsageMaximumTime.AsTime()
	out.Net_usage_average = in.NetUsageAverage
	out.Net_usage_minimum = in.NetUsageMinimum
	out.Net_usage_minimum_time = in.NetUsageMinimumTime.AsTime()
	out.CpuBilling = in.CpuBilling
	out.MemBilling = in.MemBilling
	out.DiskBilling = in.DiskBilling
	out.NetBilling = in.NetBilling
}

// ClusterPerfToMessage конвертирует models.ClusterPerf в gRPC-сообщение pb.ClusterPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func ClusterPerfToMessage(in *models.ClusterPerf, out *pb.ClusterPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ClusterId = in.ClusterId
	out.Start = timestamppb.New(in.Start)
	out.CpuUsage = in.Cpu_usage
	out.MemUsage = in.Mem_usage
	out.DiskUsage = in.Disk_usage
	out.DiskTotalReadLatency = in.Disk_totalReadLatency
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency
	out.DiskWrite = in.Disk_write
	out.DiskRead = in.Disk_read
	out.NetUsage = in.Net_usage
	out.NetReceived = in.Net_received
	out.NetTransmitted = in.Net_transmitted
}

// MessageToClusterPerf выполняет обратную конвертацию pb.ClusterPerf в models.ClusterPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToClusterPerf(in *pb.ClusterPerf, out *models.ClusterPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ClusterId = in.ClusterId
	out.Start = in.Start.AsTime()
	out.Cpu_usage = in.CpuUsage
	out.Mem_usage = in.MemUsage
	out.Disk_usage = in.DiskUsage
	out.Disk_totalReadLatency = in.DiskTotalReadLatency
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency
	out.Disk_write = in.DiskWrite
	out.Disk_read = in.DiskRead
	out.Net_usage = in.NetUsage
	out.Net_received = in.NetReceived
	out.Net_transmitted = in.NetTransmitted
}

// HostPerfToMessage конвертирует models.HostPerf в gRPC-сообщение pb.HostPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func HostPerfToMessage(in *models.HostPerf, out *pb.HostPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.HostId = in.HostId
	out.Start = timestamppb.New(in.Start)
	out.CpuUsage = in.Cpu_usage
	out.MemUsage = in.Mem_usage
	out.DiskUsage = in.Disk_usage
	out.DiskTotalReadLatency = in.Disk_totalReadLatency
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency
	out.DiskWrite = in.Disk_write
	out.DiskRead = in.Disk_read
	out.NetUsage = in.Net_usage
	out.NetReceived = in.Net_received
	out.NetTransmitted = in.Net_transmitted
}

// MessageToHostPerf выполняет обратную конвертацию pb.HostPerf в models.HostPerf.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToHostPerf(in *pb.HostPerf, out *models.HostPerf) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.HostId = in.HostId
	out.Start = in.Start.AsTime()
	out.Cpu_usage = in.CpuUsage
	out.Mem_usage = in.MemUsage
	out.Disk_usage = in.DiskUsage
	out.Disk_totalReadLatency = in.DiskTotalReadLatency
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency
	out.Disk_write = in.DiskWrite
	out.Disk_read = in.DiskRead
	out.Net_usage = in.NetUsage
	out.Net_received = in.NetReceived
	out.Net_transmitted = in.NetTransmitted
}

// VMPerfRecordToMessage конвертирует models.VMPerfRecord в gRPC-сообщение pb.VMPerfRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func VMPerfRecordToMessage(in *models.VMPerfRecord, out *pb.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.VMID = in.VMId
	out.Start = timestamppb.New(in.Start)
	out.CpuUsage = in.Cpu_usage
	out.MemUsage = in.Mem_usage
	out.DiskUsage = in.Disk_usage
	out.DiskTotalReadLatency = in.Disk_totalReadLatency
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency
	out.DiskWrite = in.Disk_write
	out.DiskRead = in.Disk_read
	out.NetUsage = in.Net_usage
	out.NetReceived = in.Net_received
	out.NetTransmitted = in.Net_transmitted
}

// MessageToVMPerfRecord выполняет обратную конвертацию pb.VMPerfRecord в models.VMPerfRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToVMPerfRecord(in *pb.VMPerfRecord, out *models.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.VMId = in.VMID
	out.Start = in.Start.AsTime()
	out.Cpu_usage = in.CpuUsage
	out.Mem_usage = in.MemUsage
	out.Disk_usage = in.DiskUsage
	out.Disk_totalReadLatency = in.DiskTotalReadLatency
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency
	out.Disk_write = in.DiskWrite
	out.Disk_read = in.DiskRead
	out.Net_usage = in.NetUsage
	out.Net_received = in.NetReceived
	out.Net_transmitted = in.NetTransmitted
}

// MessageToServerSprRecord выполняет обратную конвертацию pb.ServerSprRecords в models.ServerSprRecords.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToServerSprRecord(in *pb.ServerSprRecords, out *models.ServerSprRecords) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.OldId = in.OldId
	out.Type = int(in.Type)
	out.TypeMicroservice = int(in.TypeMicroservice)
	out.RoleType = int(in.RoleType)
	out.TypeWork = int(in.TypeWork)
	out.CriticalLevel = int(in.CriticalLevel)
	out.VMSprId = in.VMSprId
	out.Stat = int(in.Stat)
	out.PowerState = int(in.PowerState)
	out.Active = int(in.Active)
	if in.TimeActiveState != nil {
		out.TimeActiveState = in.TimeActiveState.AsTime()
	}
	out.ActivatorUser = in.ActivatorUser
	if in.TimeDeActiveState != nil {
		out.TimeDeActiveState = in.TimeDeActiveState.AsTime()
	}
	out.DeActivatorUser = in.DeActivatorUser
	if in.TimeCreate != nil {
		out.TimeCreate = in.TimeCreate.AsTime()
	}
	out.CreatorUser = in.CreatorUser
	out.BusinesPrivilage = in.BusinesPrivilage
	out.Name = in.Name
	out.NameFQDN = in.NameFQDN
	out.NameInfo = in.NameInfo
	out.IPAddress = in.IPAddress
	out.IPSharedVIP = in.IPSharedVIP
	out.Network = in.Network
	out.NetworkName = in.NetworkName
	out.CPUCount = in.CPUCount
	out.CPULogical = in.CPULogical
	out.MemSize = in.MemSize
	out.DiskSize = in.DiskSize
	out.Service = in.Service
	out.ServiceId = in.ServiceId
	out.ITSystemKey = in.ITSystemKey
	out.OS = in.OS
	out.Component = in.Component
	out.Owner = in.Owner
	out.BusinessOwner = in.BusinessOwner
	out.ITOwner = in.ITOwner
	out.ProductOwner = in.ProductOwner
	out.Support = in.Support
	out.SupportTeam = in.SupportTeam
	out.SystemType = in.SystemType
	out.LicensingType = in.LicensingType
	out.SolutionType = in.SolutionType
	out.Users = in.Users
	out.Allocation = in.Allocation
	out.Anotation = in.Anotation
	out.Role = in.Role
	out.Details = in.Details
	out.DPCAddress = int(in.DPCAddress)
	out.DPCOffice = int(in.DPCOffice)
	out.SZHistory = in.SZhistory
	out.Warning = in.Warning
	out.Level = in.Level
	out.Monitoring = int(in.Monitoring)
	out.Backuped = int(in.Backuped)
	if in.TimeActiveMonitoring != nil {
		out.TimeActiveMonitoring = in.TimeActiveMonitoring.AsTime()
	}
	out.MonitoringSZ = in.MonitoringSZ
	out.Id_User = in.Id_User
	if in.Time_Mfy != nil {
		out.Time_Mfy = in.Time_Mfy.AsTime()
	}
	out.Privilage = int(in.Privilage)
	out.ClusterId = in.ClusterId
	out.Cluster = in.Cluster
	out.Datacenter = in.Datacenter
	out.Vcenter = in.Vcenter
	out.MacAddress = in.MacAddress
}

// MessageToVMSprRecord выполняет обратную конвертацию pb.VMSprRecord в models.VMSprRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func MessageToVMSprRecord(in *pb.VMSprRecord, out *models.VMSprRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.ServerId = in.ServerId
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Cluster = in.Cluster
	out.ClusterId = in.ClusterId
	out.Vm_Name = in.Vm_Name
	out.Vm_GuestFullName = in.Vm_GuestFullName
	out.Vm_HostName = in.Vm_HostName
	out.Vm_PathName = in.Vm_PathName
	out.Vm_PowerState = in.Vm_PowerState
	out.KeyName = in.KeyName
	out.Vm_Uuid = in.Vm_Uuid
	out.Vm_GuestId = in.Vm_GuestId
	out.Vm_Annotation = in.Vm_Annotation
	out.Vm_NumCPU = in.Vm_NumCPU
	out.Vm_NumCores = in.Vm_NumCores
	out.Vm_MemorySizeGB = in.Vm_MemorySizeGB
	out.Vm_IpAddress = in.Vm_IpAddress
	out.Vm_NumEthernetCards = in.Vm_NumEthernetCards
	out.Vm_NumVirtualDisks = in.Vm_NumVirtualDisks
	out.Vm_DiskCommittedGB = in.Vm_DiskCommittedGB
	out.Vm_DiskUncommittedGB = in.Vm_DiskUncommittedGB
	out.Vm_CapacityGb = in.Vm_CapacityGb
	out.Vm_FreeSpaceGb = in.Vm_FreeSpaceGb
	out.Vm_PercentUsed = in.Vm_PercentUsed
	out.Cpu_usage_maximum = in.CpuUsageMaximum
	out.Mem_usage_maximum = in.MemUsageMaximum
	out.Disk_usage_maximum = in.DiskUsageMaximum
	out.Net_usage_maximum = in.NetUsageMaximum
	out.Vm_MacAddress = in.Vm_MacAddress
	out.Vm_GuestDetailedData = in.GuestDetailedData
	out.Vm_ToolsStatus = in.ToolsStatus
	out.Vm_DomainName = in.DomainName
}

// TODO: clusterAssignMetrics helper consolidates duplicated metric assignments not shown in screenshots
