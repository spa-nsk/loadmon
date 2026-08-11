package grpcserver

import (
	"loadmon/protocol"

	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

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
	// CalcPerfomance пересчитывает cl.Perf (по хостам кластера, cl.Hosts) и
	// все агрегаты Min/Avg/Max ниже - без этого вызова срез Perf остаётся
	// пустым (PerfCount=0), а метрики CPU/память/диск/сеть - нулевыми, даже
	// если у хостов кластера есть накопленные данные. ClusterToMessage и
	// ClusterToMessageMedium уже делают этот вызов - здесь он отсутствовал,
	// из-за чего /sprsrv/listclusterview и /sprsrv/clusterperfday (оба
	// используют GetClusterFull) получали кластеры без метрик.
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
