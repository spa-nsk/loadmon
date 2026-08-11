package grpcclient

import (
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
)

// MessageToHostInfoFull конвертирует полное gRPC-сообщение pb.HostFull (хост vSphere со всеми метриками) в доменную модель vsphere.HostInfo. При in == nil или hs == nil ничего не делает.
func MessageToHostInfoFull(in *pb.HostFull, hs *vsphere.HostInfo) {
	if in == nil || hs == nil {
		return
	}
	hs.TagName = in.TagName
	hs.TagDescription = in.TagDescription
	hs.ID = in.Id
	hs.Name = in.Name

	hs.ProductFullName = in.ProductFullName

	hs.Vcenter = in.Vcenter
	hs.Datacenter = in.Datacenter
	hs.Cluster = in.Cluster
	hs.ClusterID = in.ClusterId

	hs.KeyName = in.KeyName
	hs.Vendor = in.Vendor
	hs.Model = in.Model
	hs.SerialNumber = in.SerialNumber
	hs.VirtualSerialNumber = in.VirtualSerialNumber
	hs.EnclosureSerialNumber = in.EnclosureSerialNumber
	hs.UUID = in.Uuid
	hs.PowerState = in.PowerState
	hs.InMaintenanceMode = in.InMaintenanceMode
	hs.CPUModel = in.CpuModel
	hs.CPUMhz = in.CpuMhz
	hs.NumCPUPkgs = int16(in.NumCpuPkgs)
	hs.NumCPUCores = int16(in.NumCpuCores)
	hs.NumCPUThreads = int16(in.NumCpuThreads)
	hs.NumHBAs = in.NumHBAs
	hs.NumNics = in.NumNics
	hs.IPAddress = in.IpAddress
	hs.MacAddress = in.MacAddress
	hs.MemorySizeGB = in.MemorySizeGB
	hs.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity

	hs.CPUUsageMaximum = in.CpuUsageMaximum
	hs.CPUUsageMaximumTime = in.CpuUsageMaximumTime.AsTime()
	hs.CPUUsageAverage = in.CpuUsageAverage
	hs.CPUUsageMinimum = in.CpuUsageMinimum
	hs.CPUUsageMinimumTime = in.CpuUsageMinimumTime.AsTime()
	hs.MemUsageMaximum = in.MemUsageMaximum
	hs.MemUsageMaximumTime = in.MemUsageMaximumTime.AsTime()
	hs.MemUsageAverage = in.MemUsageAverage
	hs.MemUsageMinimum = in.MemUsageMinimum
	hs.MemUsageMinimumTime = in.MemUsageMinimumTime.AsTime()
	hs.DiskUsageMaximum = in.DiskUsageMaximum
	hs.DiskUsageMaximumTime = in.DiskUsageMaximumTime.AsTime()
	hs.DiskUsageAverage = in.DiskUsageAverage
	hs.DiskUsageMinimum = in.DiskUsageMinimum
	hs.DiskUsageMinimumTime = in.DiskUsageMinimumTime.AsTime()
	hs.DiskWriteMaximum = in.DiskWriteMaximum
	hs.DiskWriteMaximumTime = in.DiskWriteMaximumTime.AsTime()
	hs.DiskWriteAverage = in.DiskWriteAverage
	hs.DiskWriteMinimum = in.DiskWriteMinimum
	hs.DiskWriteMinimumTime = in.DiskWriteMinimumTime.AsTime()
	hs.DiskReadMaximum = in.DiskReadMaximum
	hs.DiskReadMaximumTime = in.DiskReadMaximumTime.AsTime()
	hs.DiskReadAverage = in.DiskReadAverage
	hs.DiskReadMinimum = in.DiskReadMinimum
	hs.DiskReadMinimumTime = in.DiskReadMinimumTime.AsTime()
	hs.NetUsageMaximum = in.NetUsageMaximum
	hs.NetUsageMaximumTime = in.NetUsageMaximumTime.AsTime()
	hs.NetUsageAverage = in.NetUsageAverage
	hs.NetUsageMinimum = in.NetUsageMinimum
	hs.NetUsageMinimumTime = in.NetUsageMinimumTime.AsTime()
	hs.PerfCount = in.PerfCount

	for _, p := range in.Perf {
		perf := vsphere.HSPerf{}
		perf.ID = p.Id
		perf.Start = p.Start.AsTime()
		perf.CPUUsage = p.CpuUsage
		perf.CPUUtilization = p.CpuUtilization
		perf.MemUsage = p.MemUsage
		perf.DiskUsage = p.DiskUsage
		perf.DiskRead = p.DiskRead
		perf.DiskWrite = p.DiskWrite
		perf.NetUsage = p.NetUsage
		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		perf.DiskTotalReadLatency = p.DiskTotalReadLatency
		perf.DiskTotalWriteLatency = p.DiskTotalWriteLatency
		hs.Perf = append(hs.Perf, &perf)
	}
}

// проверить !!! на предмет средних значений
// MessageToHostInfo конвертирует краткое gRPC-сообщение pb.Host в доменную модель vsphere.HostInfo. При in == nil или hs == nil ничего не делает.
func MessageToHostInfo(in *pb.Host, hs *vsphere.HostInfo) {
	if in == nil || hs == nil {
		return
	}
	hs.TagName = in.TagName
	hs.TagDescription = in.TagDescription
	hs.ID = in.Id
	hs.Name = in.Name

	hs.Vcenter = in.Vcenter
	hs.Datacenter = in.Datacenter
	hs.Cluster = in.Cluster
	hs.ClusterID = in.ClusterId

	hs.KeyName = in.KeyName
	hs.Vendor = in.Vendor
	hs.Model = in.Model

	hs.SerialNumber = in.SerialNumber
	hs.ProductFullName = in.ProductFullName

	hs.VirtualSerialNumber = in.VirtualSerialNumber
	hs.EnclosureSerialNumber = in.EnclosureSerialNumber
	hs.UUID = in.Uuid
	hs.PowerState = in.PowerState
	hs.InMaintenanceMode = in.InMaintenanceMode
	hs.CPUModel = in.CpuModel
	hs.CPUMhz = in.CpuMhz
	hs.NumCPUPkgs = int16(in.NumCpuPkgs)
	hs.NumCPUCores = int16(in.NumCpuCores)
	hs.NumCPUThreads = int16(in.NumCpuThreads)
	hs.NumHBAs = in.NumHBAs
	hs.NumNics = in.NumNics
	hs.IPAddress = in.IpAddress
	hs.MacAddress = in.MacAddress
	hs.MemorySizeGB = in.MemorySizeGB
	hs.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity

	hs.CPUUsageMaximum = in.CpuUsageMaximum
	hs.CPUUsageMaximumTime = in.CpuUsageMaximumTime.AsTime()

	hs.CPUUsageMinimum = in.CpuUsageMinimum
	hs.CPUUsageMinimumTime = in.CpuUsageMinimumTime.AsTime()
	hs.MemUsageMaximum = in.MemUsageMaximum
	hs.MemUsageMaximumTime = in.MemUsageMaximumTime.AsTime()

	hs.MemUsageMinimum = in.MemUsageMinimum
	hs.MemUsageMinimumTime = in.MemUsageMinimumTime.AsTime()
	hs.DiskUsageMaximum = in.DiskUsageMaximum
	hs.DiskUsageMaximumTime = in.DiskUsageMaximumTime.AsTime()

	hs.DiskUsageMinimum = in.DiskUsageMinimum
	hs.DiskUsageMinimumTime = in.DiskUsageMinimumTime.AsTime()
	hs.DiskWriteMaximum = in.DiskWriteMaximum
	hs.DiskWriteMaximumTime = in.DiskWriteMaximumTime.AsTime()

	hs.DiskWriteMinimum = in.DiskWriteMinimum
	hs.DiskWriteMinimumTime = in.DiskWriteMinimumTime.AsTime()
	hs.DiskReadMaximum = in.DiskReadMaximum
	hs.DiskReadMaximumTime = in.DiskReadMaximumTime.AsTime()

	hs.DiskReadMinimum = in.DiskReadMinimum
	hs.DiskReadMinimumTime = in.DiskReadMinimumTime.AsTime()
	hs.NetUsageMaximum = in.NetUsageMaximum
	hs.NetUsageMaximumTime = in.NetUsageMaximumTime.AsTime()

	hs.NetUsageMinimum = in.NetUsageMinimum
	hs.NetUsageMinimumTime = in.NetUsageMinimumTime.AsTime()

	hs.PerfCount = in.PerfCount
	//count := in.PerfCount
	//if in.PerfCount == 0 {
	//count = 1
	//}
	//count := 1
	//hs.CPUUsageAverage = in.CpuUsageAverage / float64(count)
	//hs.MemUsageAverage = in.MemUsageAverage / float64(count)
	//hs.DiskUsageAverage = in.DiskUsageAverage / float64(count)
	//hs.DiskWriteAverage = in.DiskWriteAverage / float64(count)
	//hs.DiskReadAverage = in.DiskReadAverage / float64(count)
	//hs.NetUsageAverage = in.NetUsageAverage / float64(count)
	hs.CPUUsageAverage = in.CpuUsageAverage
	hs.MemUsageAverage = in.MemUsageAverage
	hs.DiskUsageAverage = in.DiskUsageAverage
	hs.DiskWriteAverage = in.DiskWriteAverage
	hs.DiskReadAverage = in.DiskReadAverage
	hs.NetUsageAverage = in.NetUsageAverage

}

// MessageToClusterMedium конвертирует gRPC-сообщение pb.Cluster (кластер vSphere среднего уровня детализации) в доменную модель vsphere.Cluster. При in == nil или cl == nil ничего не делает.
func MessageToClusterMedium(in *pb.Cluster, cl *vsphere.Cluster) {
	if in == nil || cl == nil {
		return
	}
	cl.HostFlag = in.HostFlag
	cl.ID = in.Id
	cl.Date = in.Date.AsTime()
	cl.Name = in.Name
	cl.ClusterID = in.ClusterId
	cl.TypeWork = int(in.TypeWork)
	cl.Vcenter = in.Vcenter
	cl.Datacenter = in.Datacenter
	cl.CPUMhzAvg = in.CpuMhzAvg
	cl.AllNumCPUPkgs = int16(in.AllNumCpuPkgs)
	cl.AllNumCPUCores = int16(in.AllNumCpuCores)
	cl.AllNumCPUThreads = int16(in.AllNumCpuThreads)
	cl.AllMemorySizeGB = in.AllMemorySizeGB
	cl.AllCapacityGb = in.AllCapacityGb
	cl.AllFreeSpaceGb = in.AllFreeSpaceGb
	cl.AllPercentUsed = in.AllPercentUsed
	cl.AllocatedCoreCount = int(in.AllocatedCoreCount)
	cl.AllocatedMemorySizeGb = int(in.AllocatedMemorySizeGb)
	cl.CoreTransferRate = in.CoreTransferRate
	cl.MemoryTransferRate = in.CoreTransferRate
	cl.CPUUsageMaximum = in.CpuUsageMaximum
	cl.CPUUsageMaximumTime = in.CpuUsageMaximumTime.AsTime()
	cl.CPUUsageAverage = in.CpuUsageAverage
	cl.CPUUsageMinimum = in.CpuUsageMinimum
	cl.CPUUsageMinimumTime = in.CpuUsageMinimumTime.AsTime()
	cl.MemUsageMaximum = in.MemUsageMaximum
	cl.MemUsageMaximumTime = in.MemUsageMaximumTime.AsTime()
	cl.MemUsageAverage = in.MemUsageAverage
	cl.MemUsageMinimum = in.MemUsageMinimum
	cl.MemUsageMinimumTime = in.MemUsageMinimumTime.AsTime()
	cl.DiskUsageMaximum = in.DiskUsageMaximum
	cl.DiskUsageMaximumTime = in.DiskUsageMaximumTime.AsTime()
	cl.DiskUsageAverage = in.DiskUsageAverage
	cl.DiskUsageMinimum = in.DiskUsageMinimum
	cl.DiskUsageMinimumTime = in.DiskUsageMinimumTime.AsTime()
	cl.DiskWriteMaximum = in.DiskWriteMaximum
	cl.DiskWriteMaximumTime = in.DiskWriteMaximumTime.AsTime()
	cl.DiskWriteAverage = in.DiskWriteAverage
	cl.DiskWriteMinimum = in.DiskWriteMinimum
	cl.DiskWriteMinimumTime = in.DiskWriteMinimumTime.AsTime()
	cl.DiskReadMaximum = in.DiskReadMaximum
	cl.DiskReadMaximumTime = in.DiskReadMaximumTime.AsTime()
	cl.DiskReadAverage = in.DiskReadAverage
	cl.DiskReadMinimum = in.DiskReadMinimum
	cl.DiskReadMinimumTime = in.DiskReadMinimumTime.AsTime()
	cl.NetUsageMaximum = in.NetUsageMaximum
	cl.NetUsageMaximumTime = in.NetUsageMaximumTime.AsTime()
	cl.NetUsageAverage = in.NetUsageAverage
	cl.NetUsageMinimum = in.NetUsageMinimum
	cl.NetUsageMinimumTime = in.NetUsageMinimumTime.AsTime()
	cl.HostCount = in.HostCount
	cl.VMCount = in.VMCount
	cl.PerfCount = in.PerfCount
}

// MessageToClusterFull конвертирует полное gRPC-сообщение pb.ClusterFull (кластер со списком метрик и хранилищ) в доменную модель vsphere.Cluster. При in == nil или cl == nil ничего не делает.
func MessageToClusterFull(in *pb.ClusterFull, cl *vsphere.Cluster) {
	if in == nil || cl == nil {
		return
	}
	cl.HostFlag = in.HostFlag
	cl.ID = in.Id
	cl.Date = in.Date.AsTime()
	cl.Name = in.Name
	cl.ClusterID = in.ClusterId

	cl.TypeWork = int(in.TypeWork)
	cl.Vcenter = in.Vcenter
	cl.Datacenter = in.Datacenter
	cl.CPUMhzAvg = in.CpuMhzAvg
	cl.AllNumCPUPkgs = int16(in.AllNumCpuPkgs)
	cl.AllNumCPUCores = int16(in.AllNumCpuCores)
	cl.AllNumCPUThreads = int16(in.AllNumCpuThreads)
	cl.AllMemorySizeGB = in.AllMemorySizeGB
	cl.AllCapacityGb = in.AllCapacityGb
	cl.AllFreeSpaceGb = in.AllFreeSpaceGb
	cl.AllPercentUsed = in.AllPercentUsed
	cl.AllocatedCoreCount = int(in.AllocatedCoreCount)
	cl.AllocatedMemorySizeGb = int(in.AllocatedMemorySizeGb)
	cl.CoreTransferRate = in.CoreTransferRate
	cl.MemoryTransferRate = in.MemoryTransferRate
	cl.CPUUsageMaximum = in.CpuUsageMaximum
	cl.CPUUsageMaximumTime = in.CpuUsageMaximumTime.AsTime()
	cl.CPUUsageAverage = in.CpuUsageAverage
	cl.CPUUsageMinimum = in.CpuUsageMinimum
	cl.CPUUsageMinimumTime = in.CpuUsageMinimumTime.AsTime()
	cl.MemUsageMaximum = in.MemUsageMaximum
	cl.MemUsageMaximumTime = in.MemUsageMaximumTime.AsTime()
	cl.MemUsageAverage = in.MemUsageAverage
	cl.MemUsageMinimum = in.MemUsageMinimum
	cl.MemUsageMinimumTime = in.MemUsageMinimumTime.AsTime()
	cl.DiskUsageMaximum = in.DiskUsageMaximum
	cl.DiskUsageMaximumTime = in.DiskUsageMaximumTime.AsTime()
	cl.DiskUsageAverage = in.DiskUsageAverage
	cl.DiskUsageMinimum = in.DiskUsageMinimum
	cl.DiskUsageMinimumTime = in.DiskUsageMinimumTime.AsTime()
	cl.DiskWriteMaximum = in.DiskWriteMaximum
	cl.DiskWriteMaximumTime = in.DiskWriteMaximumTime.AsTime()
	cl.DiskWriteAverage = in.DiskWriteAverage
	cl.DiskWriteMinimum = in.DiskWriteMinimum
	cl.DiskWriteMinimumTime = in.DiskWriteMinimumTime.AsTime()
	cl.DiskReadMaximum = in.DiskReadMaximum
	cl.DiskReadMaximumTime = in.DiskReadMaximumTime.AsTime()
	cl.DiskReadAverage = in.DiskReadAverage
	cl.DiskReadMinimum = in.DiskReadMinimum
	cl.DiskReadMinimumTime = in.DiskReadMinimumTime.AsTime()
	cl.NetUsageMaximum = in.NetUsageMaximum
	cl.NetUsageMaximumTime = in.NetUsageMaximumTime.AsTime()
	cl.NetUsageAverage = in.NetUsageAverage
	cl.NetUsageMinimum = in.NetUsageMinimum
	cl.NetUsageMinimumTime = in.NetUsageMinimumTime.AsTime()
	cl.HostCount = in.HostCount
	cl.VMCount = in.VMCount
	cl.PerfCount = in.PerfCount
	for _, p := range in.Perf {
		perf := vsphere.CLPerf{}
		perf.ID = p.Id
		perf.Start = p.Start.AsTime()
		perf.CPUUsage = p.CpuUsage
		perf.CPUUtilization = p.CpuUtilization
		perf.MemUsage = p.MemUsage
		perf.DiskUsage = p.DiskUsage
		perf.DiskRead = p.DiskRead
		perf.DiskWrite = p.DiskWrite
		perf.NetUsage = p.NetUsage
		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		cl.Perf = append(cl.Perf, &perf)
	}
	for _, s := range in.Storages {
		ds := vsphere.Datastore{ID: s.Id,
			UID:         s.UID,
			StorageName: s.StorageName,
			CapacityGb:  s.CapacityGb,
			FreeSpaceGb: s.FreeSpaceGb,
			PercentUsed: s.PercentUsed}
		for _, vmk := range s.VmKeys.KeyRecs {
			vm := vsphere.VMPoint{Name: vmk.VMName, KeyName: vmk.Key}
			ds.VMCount++
			ds.VM = append(ds.VM, &vm)
		}
		cl.StorageCount++
		cl.Storage = append(cl.Storage, &ds)
	}
}

// MessageToClusterStorages конвертирует gRPC-сообщение pb.Datastore в доменную модель vsphere.Datastore, включая связанные с ним VM по ключам. При in == nil или out == nil ничего не делает.
func MessageToClusterStorages(in *pb.Datastore, out *vsphere.Datastore) {
	if in == nil || out == nil {
		return
	}
	out.ID = in.Id
	out.UID = in.UID
	out.StorageName = in.StorageName
	out.CapacityGb = in.CapacityGb
	out.FreeSpaceGb = in.FreeSpaceGb
	out.PercentUsed = in.PercentUsed
	for _, vmk := range in.VmKeys.KeyRecs {
		vm := vsphere.VMPoint{Name: vmk.VMName, KeyName: vmk.Key}
		out.VMCount++
		out.VM = append(out.VM, &vm)
	}
}

// MessageToClusterMetric конвертирует одну точку метрик кластера pb.CLPerf в доменную модель vsphere.CLPerf. При in == nil или clp == nil ничего не делает.
func MessageToClusterMetric(in *pb.CLPerf, clp *vsphere.CLPerf) {
	if in == nil || clp == nil {
		return
	}
	clp.ID = in.Id
	clp.Start = in.Start.AsTime()
	clp.CPUUsage = in.CpuUsage
	clp.CPUUtilization = in.CpuUtilization
	clp.MemUsage = in.MemUsage
	clp.DiskUsage = in.DiskUsage
	clp.DiskRead = in.DiskRead
	clp.DiskWrite = in.DiskWrite
	clp.NetUsage = in.NetUsage
	clp.NetReceived = in.NetReceived
	clp.NetTransmitted = in.NetTransmitted
}

// MessageToHostMetric конвертирует одну точку метрик хоста pb.HSPerf в доменную модель vsphere.HSPerf. При in == nil или clp == nil ничего не делает.
func MessageToHostMetric(in *pb.HSPerf, clp *vsphere.HSPerf) {
	if in == nil || clp == nil {
		return
	}
	clp.ID = in.Id
	clp.Start = in.Start.AsTime()
	clp.CPUUsage = in.CpuUsage
	clp.CPUUtilization = in.CpuUtilization
	clp.MemUsage = in.MemUsage
	clp.DiskUsage = in.DiskUsage
	clp.DiskRead = in.DiskRead
	clp.DiskWrite = in.DiskWrite
	clp.NetUsage = in.NetUsage
	clp.NetReceived = in.NetReceived
	clp.NetTransmitted = in.NetTransmitted
}

// MessageToServerSprRecod конвертирует gRPC-сообщение pb.ServerSprRecords (запись справочника серверов) в доменную модель models.ServerSprRecords. При in == nil или out == nil ничего не делает.
