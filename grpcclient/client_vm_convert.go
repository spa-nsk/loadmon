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
	// VMClearPerfomance обнуляет в том числе *Billing - а они приходят от
	// сервера готовыми (pb.VMPoint.*Billing), агрегатами по Perf не являются и
	// пересчёту не подлежат. Без восстановления биллинг терялся у всех
	// вызывающих (GetVM, GetVMByKeyWithPerf, LoadVMByKeyRecs и т.п.).
	out.CPUBilling = in.CpuBilling
	out.MemBilling = in.MemBilling
	out.DiskBilling = in.DiskBilling
	out.NetBilling = in.NetBilling
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
		if out.DiskTotalReadLatencyMaximum < perf.DiskTotalReadLatency {
			out.DiskTotalReadLatencyMaximum = perf.DiskTotalReadLatency
			out.DiskTotalReadLatencyMaximumTime = perf.Start
		}
		if out.DiskTotalReadLatencyMinimum > perf.DiskTotalReadLatency {
			out.DiskTotalReadLatencyMinimum = perf.DiskTotalReadLatency
			out.DiskTotalReadLatencyMinimumTime = perf.Start
		}
		out.DiskTotalReadLatencyAverage += perf.DiskTotalReadLatency

		perf.DiskTotalWriteLatency = p.DiskTotalWriteLatency
		if out.DiskTotalWriteLatencyMaximum < perf.DiskTotalWriteLatency {
			out.DiskTotalWriteLatencyMaximum = perf.DiskTotalWriteLatency
			out.DiskTotalWriteLatencyMaximumTime = perf.Start
		}
		if out.DiskTotalWriteLatencyMinimum > perf.DiskTotalWriteLatency {
			out.DiskTotalWriteLatencyMinimum = perf.DiskTotalWriteLatency
			out.DiskTotalWriteLatencyMinimumTime = perf.Start
		}
		out.DiskTotalWriteLatencyAverage += perf.DiskTotalWriteLatency
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
		out.DiskTotalReadLatencyAverage /= float64(out.PerfCount)
		out.DiskTotalWriteLatencyAverage /= float64(out.PerfCount)
	}
}

// GetVMPerfRPC конвертирует gRPC-сообщение pb.VMPerfSlice в срез доменных точек
// vsphere.VMPerf (время переводится в локальную зону, как в CopyAndReCalcVMPerformance).
// При in == nil возвращает пустой (не nil) срез; элементы nil пропускаются.
func GetVMPerfRPC(in *pb.VMPerfSlice) []*vsphere.VMPerf {
	out := make([]*vsphere.VMPerf, 0)
	if in == nil {
		return out
	}
	for _, p := range in.Perf {
		if p == nil {
			continue
		}
		out = append(out, &vsphere.VMPerf{
			ID:                    p.Id,
			Start:                 p.Start.AsTime().Local(),
			CPUUsage:              p.CpuUsage,
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
	return out
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
	vm.EsxHost = in.EsxHost // реальный ESXi-хост, см. protocol/vm.proto (2026-09-15)
	vm.PathName = in.PathName
	vm.PowerState = in.PowerState
	vm.CPUMhz = in.CpuMhz
	vm.NumCPU = in.NumCPU
	vm.NumCores = in.NumCores
	vm.MemorySizeGB = in.MemorySizeGB
	vm.Network = in.Network
	vm.IPAddress = in.IpAddress
	vm.MacAddress = in.MacAddress
	// Interfaces - структурированные сетевые интерфейсы (Stage 1 доработки
	// хранения сетевых интерфейсов серверов, см. CHANGELOG.md), аддитивно
	// к IPAddress/MacAddress/Network выше.
	vm.Interfaces = netInterfacesRPC(in.Interfaces)
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

	vm.Files = append(vm.Files, VMFilesRPC(in.VMFiles)...)
	vm.Disk = append(vm.Disk, VMDisksRPC(in.VMDisk)...)
	vm.UsageDatastore = append(vm.UsageDatastore, VMUsageOnDatastoreRPC(in.UsageDatastore)...)
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

// VMFilesRPC конвертирует список файлов ВМ pb.VMFile в доменную модель vsphere.VMFile (nil-элементы пропускаются).
func VMFilesRPC(in []*pb.VMFile) []vsphere.VMFile {
	out := make([]vsphere.VMFile, 0, len(in))
	for _, f := range in {
		if f == nil {
			continue
		}
		out = append(out, vsphere.VMFile{Type: f.Type, Name: f.Name, Size: f.Size, UniqueSize: f.UniqueSize})
	}
	return out
}

// VMDisksRPC конвертирует список дисков гостевой ОС pb.GuestDiskInfo в доменную модель vsphere.GuestDiskInfo (nil-элементы пропускаются).
func VMDisksRPC(in []*pb.GuestDiskInfo) []vsphere.GuestDiskInfo {
	out := make([]vsphere.GuestDiskInfo, 0, len(in))
	for _, d := range in {
		if d == nil {
			continue
		}
		out = append(out, vsphere.GuestDiskInfo{DiskPath: d.DiskPath, FileSystemType: d.FileSystemType, CapacityGb: d.CapacityGb, FreeSpaceGb: d.FreeSpaceGb, PercentUsed: d.PercentUsed})
	}
	return out
}

// VMUsageOnDatastoreRPC конвертирует список использования хранилищ ВМ pb.VMUsageOnDatastore в доменную модель vsphere.VMUsageOnDatastore (nil-элементы пропускаются).
func VMUsageOnDatastoreRPC(in []*pb.VMUsageOnDatastore) []vsphere.VMUsageOnDatastore {
	out := make([]vsphere.VMUsageOnDatastore, 0, len(in))
	for _, d := range in {
		if d == nil {
			continue
		}
		out = append(out, vsphere.VMUsageOnDatastore{
			ID:                     d.Id,
			UID:                    d.UID,
			Vcenter:                d.Vcenter,
			Datacenter:             d.Datacenter,
			Cluster:                d.Cluster,
			Datastore:              d.Datastore,
			StorageName:            d.StorageName,
			CommittedGb:            d.CommittedGb,
			UncommittedGb:          d.UncommittedGb,
			UnsharedGb:             d.UnsharedGb,
			PercentAllocateStorage: d.PercentAllocateStorage,
		})
	}
	return out
}

// GetVMPointShortRPC конвертирует укороченное gRPC-сообщение pb.VMPointShort в доменную модель vsphere.VMPoint.
// В отличие от GetVMPointRPC в сообщении нет временного ряда Perf, поэтому агрегаты (min/max/average и времена
// экстремумов) берутся из самого сообщения как есть (average уже поделено сервером на PerfCount). При in == nil
// возвращает нулевое значение vsphere.VMPoint.
func GetVMPointShortRPC(in *pb.VMPointShort) vsphere.VMPoint {
	var vm vsphere.VMPoint
	if in == nil {
		return vm
	}
	vm.ID = in.Id
	vm.Name = in.Name
	vm.GuestFullName = in.GuestFullName
	vm.HostName = in.HostName
	vm.PathName = in.PathName
	vm.PowerState = in.PowerState
	vm.NumCPU = in.NumCPU
	vm.NumCores = in.NumCores
	vm.MemorySizeGB = in.MemorySizeGB
	vm.IPAddress = in.IpAddress
	vm.CapacityGb = in.CapacityGb
	vm.FreeSpaceGb = in.FreeSpaceGb
	vm.PercentUsed = in.PercentUsed
	vm.PerfCount = in.PerfCount
	vm.TagName = in.TagName
	vm.TagDescription = in.TagDescription
	vm.CPUMhz = in.CpuMhz
	vm.Network = in.Network
	vm.WorkStation = in.Workstation
	vm.Template = in.Template
	vm.ClusterID = in.ClusterId
	vm.TypeWork = int(in.TypeWork)
	vm.MacAddress = in.MacAddress
	vm.GuestDetailedData = in.GuestDetailedData
	vm.ToolsStatus = in.ToolsStatus
	vm.DomainName = in.DomainName
	vm.Vcenter = in.Vcenter
	vm.Datacenter = in.Datacenter
	vm.Cluster = in.Cluster
	vm.KeyName = in.KeyName
	vm.UUID = in.Uuid
	vm.GuestID = in.GuestId
	vm.Annotation = in.Annotation
	vm.NumEthernetCards = in.NumEthernetCards
	vm.NumVirtualDisks = in.NumVirtualDisks
	vm.DiskCommittedGB = in.DiskCommittedGB
	vm.DiskUncommittedGB = in.DiskUncommittedGB
	vm.EsxHost = in.EsxHost
	vm.Interfaces = netInterfacesRPC(in.Interfaces)
	vm.CPUBilling = in.CpuBilling
	vm.MemBilling = in.MemBilling
	vm.DiskBilling = in.DiskBilling
	vm.NetBilling = in.NetBilling

	vm.DiskTotalReadLatencyMaximum = in.DiskTotalReadLatencyMaximum
	vm.DiskTotalReadLatencyMaximumTime = in.DiskTotalReadLatencyMaximumTime.AsTime()
	vm.DiskTotalReadLatencyAverage = in.DiskTotalReadLatencyAverage
	vm.DiskTotalReadLatencyMinimum = in.DiskTotalReadLatencyMinimum
	vm.DiskTotalReadLatencyMinimumTime = in.DiskTotalReadLatencyMinimumTime.AsTime()
	vm.DiskTotalWriteLatencyMaximum = in.DiskTotalWriteLatencyMaximum
	vm.DiskTotalWriteLatencyMaximumTime = in.DiskTotalWriteLatencyMaximumTime.AsTime()
	vm.DiskTotalWriteLatencyAverage = in.DiskTotalWriteLatencyAverage
	vm.DiskTotalWriteLatencyMinimum = in.DiskTotalWriteLatencyMinimum
	vm.DiskTotalWriteLatencyMinimumTime = in.DiskTotalWriteLatencyMinimumTime.AsTime()

	vm.CPUUsageMaximum = in.CpuUsageMaximum
	vm.CPUUsageMaximumTime = in.CpuUsageMaximumTime.AsTime()
	vm.CPUUsageAverage = in.CpuUsageAverage
	vm.CPUUsageMinimum = in.CpuUsageMinimum
	vm.CPUUsageMinimumTime = in.CpuUsageMinimumTime.AsTime()
	vm.MemUsageMaximum = in.MemUsageMaximum
	vm.MemUsageMaximumTime = in.MemUsageMaximumTime.AsTime()
	vm.MemUsageAverage = in.MemUsageAverage
	vm.MemUsageMinimum = in.MemUsageMinimum
	vm.MemUsageMinimumTime = in.MemUsageMinimumTime.AsTime()
	vm.DiskUsageMaximum = in.DiskUsageMaximum
	vm.DiskUsageMaximumTime = in.DiskUsageMaximumTime.AsTime()
	vm.DiskUsageAverage = in.DiskUsageAverage
	vm.DiskUsageMinimum = in.DiskUsageMinimum
	vm.DiskUsageMinimumTime = in.DiskUsageMinimumTime.AsTime()
	vm.DiskWriteMaximum = in.DiskWriteMaximum
	vm.DiskWriteMaximumTime = in.DiskWriteMaximumTime.AsTime()
	vm.DiskWriteAverage = in.DiskWriteAverage
	vm.DiskWriteMinimum = in.DiskWriteMinimum
	vm.DiskWriteMinimumTime = in.DiskWriteMinimumTime.AsTime()
	vm.DiskReadMaximum = in.DiskReadMaximum
	vm.DiskReadMaximumTime = in.DiskReadMaximumTime.AsTime()
	vm.DiskReadAverage = in.DiskReadAverage
	vm.DiskReadMinimum = in.DiskReadMinimum
	vm.DiskReadMinimumTime = in.DiskReadMinimumTime.AsTime()
	vm.NetUsageMaximum = in.NetUsageMaximum
	vm.NetUsageMaximumTime = in.NetUsageMaximumTime.AsTime()
	vm.NetUsageAverage = in.NetUsageAverage
	vm.NetUsageMinimum = in.NetUsageMinimum
	vm.NetUsageMinimumTime = in.NetUsageMinimumTime.AsTime()
	return vm
}

// HSPerfRPC конвертирует точки метрик хоста pb.HSPerf в доменную модель vsphere.HSPerf (время в локальной зоне, nil-элементы пропускаются).
func HSPerfRPC(in []*pb.HSPerf) []*vsphere.HSPerf {
	out := make([]*vsphere.HSPerf, 0, len(in))
	for _, p := range in {
		if p == nil {
			continue
		}
		hp := vsphere.HSPerf{}
		MessageToHostMetric(p, &hp)
		hp.Start = hp.Start.Local()
		out = append(out, &hp)
	}
	return out
}

// CLPerfRPC конвертирует точки метрик кластера pb.CLPerf в доменную модель vsphere.CLPerf (время в локальной зоне, nil-элементы пропускаются).
func CLPerfRPC(in []*pb.CLPerf) []*vsphere.CLPerf {
	out := make([]*vsphere.CLPerf, 0, len(in))
	for _, p := range in {
		if p == nil {
			continue
		}
		cp := vsphere.CLPerf{}
		MessageToClusterMetric(p, &cp)
		cp.Start = cp.Start.Local()
		out = append(out, &cp)
	}
	return out
}

// netInterfacesRPC конвертирует []*pb.NetInterface в []vsphere.NetInterfacePoint (nil-элементы пропускаются). Для пустого входа возвращает nil.
func netInterfacesRPC(in []*pb.NetInterface) []vsphere.NetInterfacePoint {
	if len(in) == 0 {
		return nil
	}
	out := make([]vsphere.NetInterfacePoint, 0, len(in))
	for _, iface := range in {
		if iface == nil {
			continue
		}
		out = append(out, vsphere.NetInterfacePoint{
			MacAddress:  iface.MacAddress,
			IPAddress:   iface.IPAddress,
			NetworkName: iface.NetworkName,
			SubnetMask:  iface.SubnetMask,
		})
	}
	return out
}
