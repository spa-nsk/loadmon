// Package grpcclient — gRPC-клиент пакета loadmon. Обращается к серверу
// пакета grpcserver, который отдаёт метрики VMware vSphere (сервис Loadmon,
// адрес config.GetCliApiAddr()) и данные из БД (сервис LoadmonDB, адрес
// config.GetCliDBAddr()), а также к серверу HPE OneView за инвентарём
// оборудования.
//
// Общий стиль функций пакета:
//   - на каждый вызов открывается короткоживущее соединение через dialAPI()/
//     dialDB()/dialAPIAddr() с таймаутом 30 секунд на весь запрос (включая
//     чтение потоковых ответов), соединение закрывается через defer сразу
//     после установления;
//   - при ошибке подключения или вызова RPC в лог пишется причина
//     (log.Printf), а вызывающему возвращается "пустой" результат (nil/nil-
//     срез/false) и, где это предусмотрено сигнатурой, сама ошибка;
//   - при потоковом чтении (stream.Recv()) признаком конца потока служит
//     io.EOF, любая другая ошибка прерывает или пропускает элемент в
//     зависимости от функции (см. комментарии на местах).
package grpcclient

import (
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
)

func CopyAndReCalcVMPerformance(out *vsphere.VMPoint, in *pb.VMPoint) {
	if out == nil || in == nil {
		return
	}
	out.VMClearPerfomance()
	for _, p := range in.Perf {
		perf := vsphere.VMPerf{}
		perf.ID = p.Id
		perf.Start = p.Start.AsTime().Local()
		perf.CPUUsage = p.CpuUsage
		if out.CPUUsageMaximum < perf.CPUUsage {
			out.CPUUsageMaximum = perf.CPUUsage
			out.CPUUsageMaximumTime = perf.Start
		}
		if out.CPUUsageMinimum > perf.CPUUsage {
			out.CPUUsageMinimum = perf.CPUUsage
			out.CPUUsageMinimumTime = perf.Start
		}
		out.CPUUsageAverage += perf.CPUUsage

		perf.MemUsage = p.MemUsage
		if out.MemUsageMaximum < perf.MemUsage {
			out.MemUsageMaximum = perf.MemUsage
			out.MemUsageMaximumTime = perf.Start
		}
		if out.MemUsageMinimum > perf.MemUsage {
			out.MemUsageMinimum = perf.MemUsage
			out.MemUsageMinimumTime = perf.Start
		}
		out.MemUsageAverage += perf.MemUsage

		perf.DiskUsage = p.DiskUsage
		if out.DiskUsageMaximum < perf.DiskUsage {
			out.DiskUsageMaximum = perf.DiskUsage
			out.DiskUsageMaximumTime = perf.Start
		}
		if out.DiskUsageMinimum > perf.DiskUsage {
			out.DiskUsageMinimum = perf.DiskUsage
			out.DiskUsageMinimumTime = perf.Start
		}
		out.DiskUsageAverage += perf.DiskUsage

		perf.DiskRead = p.DiskRead
		if out.DiskReadMaximum < perf.DiskRead {
			out.DiskReadMaximum = perf.DiskRead
			out.DiskReadMaximumTime = perf.Start
		}
		if out.DiskReadMinimum > perf.DiskRead {
			out.DiskReadMinimum = perf.DiskRead
			out.DiskReadMinimumTime = perf.Start
		}
		out.DiskReadAverage += perf.DiskRead

		perf.DiskWrite = p.DiskWrite
		if out.DiskWriteMaximum < perf.DiskWrite {
			out.DiskWriteMaximum = perf.DiskWrite
			out.DiskWriteMaximumTime = perf.Start
		}
		if out.DiskWriteMinimum > perf.DiskWrite {
			out.DiskWriteMinimum = perf.DiskWrite
			out.DiskWriteMinimumTime = perf.Start
		}
		out.DiskWriteAverage += perf.DiskWrite

		perf.NetUsage = p.NetUsage
		if out.NetUsageMaximum < perf.NetUsage {
			out.NetUsageMaximum = perf.NetUsage
			out.NetUsageMaximumTime = perf.Start
		}
		if out.NetUsageMinimum > perf.NetUsage {
			out.NetUsageMinimum = perf.NetUsage
			out.NetUsageMinimumTime = perf.Start
		}
		out.NetUsageAverage += perf.NetUsage

		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		perf.DiskTotalReadLatency = p.DiskTotalReadLatency
		perf.DiskTotalWriteLatency = p.DiskTotalWriteLatency
		out.Perf = append(out.Perf, &perf)
		out.PerfCount++
	}

	if out.PerfCount > 0 {
		out.CPUUsageAverage /= float64(out.PerfCount)
		out.MemUsageAverage /= float64(out.PerfCount)
		out.DiskUsageAverage /= float64(out.PerfCount)
		out.DiskReadAverage /= float64(out.PerfCount)
		out.DiskWriteAverage /= float64(out.PerfCount)
		out.NetUsageAverage /= float64(out.PerfCount)
	}
}

// GetVMPointRPC конвертирует gRPC-сообщение pb.VMPoint в доменную модель
// vsphere.VMPoint (основные поля, метрики, файлы VM, диски, использование
// хранилищ). При in == nil возвращает нулевое значение vsphere.VMPoint.
func GetVMPointRPC(in *pb.VMPoint) vsphere.VMPoint {
	var vm vsphere.VMPoint
	if in == nil {
		return vm
	}

	vm.WorkStation = in.Workstation
	vm.TypeWork = int(in.TypeWork)
	vm.Template = in.Template
	vm.TagName = in.TagName
	vm.TagDescription = in.TagDescription
	vm.ID = in.Id

	vm.Name = in.Name
	vm.DomainName = in.DomainName
	vm.GuestDetailedData = in.GuestDetailedData
	vm.ToolsStatus = in.ToolsStatus

	vm.GuestFullName = in.GuestFullName
	vm.HostName = in.HostName
	vm.PathName = in.PathName
	vm.PowerState = in.PowerState
	vm.CPUMhz = in.CpuMhz
	vm.NumCPU = in.NumCPU
	vm.NumCores = in.NumCores
	vm.MemorySizeGB = in.MemorySizeGB
	vm.Network = in.Network
	vm.IPAddress = in.IpAddress
	vm.MacAddress = in.MacAddress
	vm.CapacityGb = in.CapacityGb
	vm.FreeSpaceGb = in.FreeSpaceGb
	vm.PercentUsed = in.PercentUsed

	// PerfCount копируется независимо от того, вызывается ли отдельно
	// CopyAndReCalcVMPerformance (которая заполняет сам срез vm.Perf) - так
	// вызывающий код может понять, есть ли у ВМ метрики для отображения
	// (например, включать/выключать пункты меню), не запрашивая весь Perf.
	// Если CopyAndReCalcVMPerformance всё же вызывается следом, она
	// пересчитывает vm.PerfCount заново по факту полученных точек - значения
	// должны совпадать, если сервер передаёт консистентные данные.
	vm.PerfCount = in.PerfCount

	// Метрики производительности (CPUUsage*, MemUsage*, Disk*, NetUsage*)
	// заполняются внутри CopyAndReCalcVMPerfomance выше — на основе временного
	// ряда in.Perf, а не готовых агрегатов из сообщения.
	vm.Vcenter = in.Vcenter
	vm.Datacenter = in.Datacenter
	vm.Cluster = in.Cluster
	vm.ClusterID = in.ClusterId

	vm.KeyName = in.KeyName
	vm.UUID = in.Uuid
	vm.GuestID = in.GuestId
	vm.Annotation = in.Annotation
	vm.NumEthernetCards = in.NumEthernetCards
	vm.NumVirtualDisks = in.NumVirtualDisks
	vm.DiskCommittedGB = in.DiskCommittedGB
	vm.DiskUncommittedGB = in.DiskUncommittedGB
	vm.CPUBilling = in.CpuBilling
	vm.MemBilling = in.MemBilling
	vm.DiskBilling = in.DiskBilling
	vm.NetBilling = in.NetBilling

	// vm.Perf и vm.PerfCount заполняются внутри CopyAndReCalcVMPerfomance выше.

	for _, f := range in.VMFiles {
		fn := vsphere.VMFile{}
		fn.Type = f.Type
		fn.Name = f.Name
		fn.Size = f.Size
		fn.UniqueSize = f.UniqueSize
		vm.Files = append(vm.Files, fn)
	}
	for _, d := range in.VMDisk {
		dn := vsphere.GuestDiskInfo{}
		dn.CapacityGb = d.CapacityGb
		dn.DiskPath = d.DiskPath
		dn.FreeSpaceGb = d.FreeSpaceGb
		dn.PercentUsed = d.PercentUsed
		vm.Disk = append(vm.Disk, dn)
	}
	for _, d := range in.UsageDatastore {
		dn := vsphere.VMUsageOnDatastore{}
		dn.ID = d.Id
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
		vm.UsageDatastore = append(vm.UsageDatastore, dn)
	}
	return vm
}

// CopyAndReCalcVMPerfomance — устаревшее имя-обёртка над
// CopyAndReCalcVMPerformance (оставлено для обратной совместимости из-за
// опечатки в исходном названии).
func CopyAndReCalcVMPerfomance(out *vsphere.VMPoint, in *pb.VMPoint) {
	CopyAndReCalcVMPerformance(out, in)
}

// GetVMByKey возвращает одну VM (первую найденную) по заданным
// vcenter/datacenter/cluster/vmname. Возвращает ошибку, если подключиться
// к серверу не удалось или данные по ключу не найдены.
