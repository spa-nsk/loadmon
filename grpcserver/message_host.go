package grpcserver

import (
	"loadmon/protocol"
	"math"

	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

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
		// Делим именно на суммы, накопленные в цикле выше по актуальному
		// (возможно, урезанному после ClearMetricByKey/clearHSPerf) срезу
		// hs.Perf, а не на "сырые" hs.CPUUsageAverage/hs.MemUsageAverage -
		// те продолжают расти по всей истории хоста и не обрезаются вместе
		// с hs.Perf при частичной очистке метрик, из-за чего среднее
		// оказывалось завышено (иногда за 100%, особенно по памяти).
		out.CpuUsageAverage /= count
		out.MemUsageAverage /= count
		out.DiskUsageAverage /= count
		out.DiskWriteAverage /= count
		out.DiskReadAverage /= count
		out.NetUsageAverage /= count
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
		// см. аналогичный комментарий в RecalcHostToMessageFull выше - делим
		// суммы, накопленные в цикле по актуальному срезу hs.Perf, а не
		// "сырые" hs.CPUUsageAverage/hs.MemUsageAverage.
		out.CpuUsageAverage /= count
		out.MemUsageAverage /= count
		out.DiskUsageAverage /= count
		out.DiskWriteAverage /= count
		out.DiskReadAverage /= count
		out.NetUsageAverage /= count
	}
}

// clusterAssignMetrics копирует агрегированные метрики производительности кластера (cpu/mem/disk/net) из cl в out.
// Вспомогательная функция, используется внутри ClusterToMessage*.
