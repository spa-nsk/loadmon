package grpcserver

import (
	"loadmon/protocol"

	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"

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

	// PerfCount передаётся независимо от того, заполняется ли сам срез
	// Perf (см. vm.PerfCount ниже) - используется на клиенте, чтобы понять,
	// есть ли метрики для отображения, не запрашивая их массив целиком.
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
