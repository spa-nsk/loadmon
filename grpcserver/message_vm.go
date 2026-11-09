package grpcserver

import (
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
	out.TypeWork = int32(vm.TypeWork)
	out.CpuBilling = vm.CPUBilling
	out.MemBilling = vm.MemBilling
	out.DiskBilling = vm.DiskBilling
	out.NetBilling = vm.NetBilling
	out.Vcenter = vm.Vcenter
	out.Datacenter = vm.Datacenter
	out.Cluster = vm.Cluster
	out.KeyName = vm.KeyName
	out.Uuid = vm.UUID
	out.GuestId = vm.GuestID
	out.Annotation = vm.Annotation
	out.NumEthernetCards = vm.NumEthernetCards
	out.NumVirtualDisks = vm.NumVirtualDisks
	out.DiskCommittedGB = vm.DiskCommittedGB
	out.DiskUncommittedGB = vm.DiskUncommittedGB
	out.EsxHost = vm.EsxHost
	out.Interfaces = hostInterfacesToMessage(vm.Interfaces)
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
		out.DiskTotalReadLatencyMaximum = vm.DiskTotalReadLatencyMaximum
		out.DiskTotalReadLatencyMaximumTime = timestamppb.New(vm.DiskTotalReadLatencyMaximumTime)
		out.DiskTotalReadLatencyMinimum = vm.DiskTotalReadLatencyMinimum
		out.DiskTotalReadLatencyMinimumTime = timestamppb.New(vm.DiskTotalReadLatencyMinimumTime)
		out.DiskTotalWriteLatencyMaximum = vm.DiskTotalWriteLatencyMaximum
		out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(vm.DiskTotalWriteLatencyMaximumTime)
		out.DiskTotalWriteLatencyMinimum = vm.DiskTotalWriteLatencyMinimum
		out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(vm.DiskTotalWriteLatencyMinimumTime)
		count := float64(vm.PerfCount)
		out.CpuUsageAverage = vm.CPUUsageAverage / count
		out.MemUsageAverage = vm.MemUsageAverage / count
		out.DiskUsageAverage = vm.DiskUsageAverage / count
		out.DiskWriteAverage = vm.DiskWriteAverage / count
		out.DiskReadAverage = vm.DiskReadAverage / count
		out.NetUsageAverage = vm.NetUsageAverage / count
		out.DiskTotalReadLatencyAverage = vm.DiskTotalReadLatencyAverage / count
		out.DiskTotalWriteLatencyAverage = vm.DiskTotalWriteLatencyAverage / count
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
	out.TypeWork = int32(vm.TypeWork)
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
	// EsxHost - реальный физический хост ESXi (в отличие от HostName выше,
	// которое остаётся гостевым именем VM) - см. комментарий у поля EsxHost
	// в protocol/vm.proto (2026-09-15).
	out.EsxHost = vm.EsxHost
	// Interfaces - структурированные сетевые интерфейсы (Stage 1 доработки
	// хранения сетевых интерфейсов серверов, см. CHANGELOG.md), аддитивно
	// к IPAddress/MacAddress/Network выше.
	out.Interfaces = hostInterfacesToMessage(vm.Interfaces)
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
		out.DiskTotalReadLatencyMaximum = vm.DiskTotalReadLatencyMaximum
		out.DiskTotalReadLatencyMaximumTime = timestamppb.New(vm.DiskTotalReadLatencyMaximumTime)
		out.DiskTotalReadLatencyMinimum = vm.DiskTotalReadLatencyMinimum
		out.DiskTotalReadLatencyMinimumTime = timestamppb.New(vm.DiskTotalReadLatencyMinimumTime)
		out.DiskTotalWriteLatencyMaximum = vm.DiskTotalWriteLatencyMaximum
		out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(vm.DiskTotalWriteLatencyMaximumTime)
		out.DiskTotalWriteLatencyMinimum = vm.DiskTotalWriteLatencyMinimum
		out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(vm.DiskTotalWriteLatencyMinimumTime)
		count := float64(vm.PerfCount)
		out.CpuUsageAverage = vm.CPUUsageAverage / count
		out.MemUsageAverage = vm.MemUsageAverage / count
		out.DiskUsageAverage = vm.DiskUsageAverage / count
		out.DiskWriteAverage = vm.DiskWriteAverage / count
		out.DiskReadAverage = vm.DiskReadAverage / count
		out.NetUsageAverage = vm.NetUsageAverage / count
		out.DiskTotalReadLatencyAverage = vm.DiskTotalReadLatencyAverage / count
		out.DiskTotalWriteLatencyAverage = vm.DiskTotalWriteLatencyAverage / count
	}

	// Конвертация точек метрик общая с GetVMPerfByKey (VMPerfToMessage) -
	// копирует в том числе DiskTotalReadLatency/DiskTotalWriteLatency.
	out.Perf = append(out.Perf, VMPerfToMessage(vm.Perf).Perf...)
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
		dn.FileSystemType = d.FileSystemType
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

// VMPerfToMessage конвертирует временной ряд метрик виртуальной машины (vsphere.VMPerf) в gRPC-сообщение
// pb.VMPerfSlice, включая задержки диска (DiskTotalReadLatency/DiskTotalWriteLatency). Элементы nil пропускаются.
// Результат всегда не nil, Count равен числу переданных точек.
func VMPerfToMessage(perf []*vsphere.VMPerf) *pb.VMPerfSlice {
	out := &pb.VMPerfSlice{Perf: make([]*pb.VMPerf, 0, len(perf))}
	for _, p := range perf {
		if p == nil {
			continue
		}
		out.Perf = append(out.Perf, &pb.VMPerf{
			Id:                    p.ID,
			Start:                 timestamppb.New(p.Start),
			CpuUsage:              p.CPUUsage,
			MemUsage:              p.MemUsage,
			DiskUsage:             p.DiskUsage,
			DiskWrite:             p.DiskWrite,
			DiskRead:              p.DiskRead,
			NetUsage:              p.NetUsage,
			NetReceived:           p.NetReceived,
			NetTransmitted:        p.NetTransmitted,
			DiskTotalReadLatency:  p.DiskTotalReadLatency,
			DiskTotalWriteLatency: p.DiskTotalWriteLatency,
		})
	}
	out.Count = int64(len(out.Perf))
	return out
}

// HostToMessageShort конвертирует внутреннюю структуру vsphere.HostInfo в укороченное gRPC-сообщение pb.Host без пересчёта агрегированных метрик.
// Принимает hs и out; при hs == nil или out == nil ничего не делает.
