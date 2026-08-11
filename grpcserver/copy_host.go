// Package grpcserver реализует gRPC-серверную часть системы мониторинга
// loadmon: два gRPC-сервиса, Loadmon (сервер server, инвентарь и метрики
// VMware vSphere, оборудования HPE OneView и т.п. — контракт см. в
// protocol/*.proto) и LoadmonDB (сервер serverdb, доступ к данным,
// сохранённым в БД проекта). Сервисы принимают запросы, полученные клиентом
// пакета grpcclient, выполняют выборку/агрегацию данных из внутренних
// пакетов проекта (vspheremetrics, oneview, models и др.) и упаковывают
// результат в сгенерированные gRPC-сообщения пакета protocol. Файл
// serverfunc.go содержит запуск серверов и вспомогательные функции, файл
// message_srv.go — функции-конвертеры доменных моделей в gRPC-сообщения.
package grpcserver

import (
	pb "loadmon/protocol"
)

func CopyHost(hs *pb.Host) *pb.Host {
	out := pb.Host{}
	if hs == nil {
		return &out
	}
	out.TagName = hs.TagName
	out.TagDescription = hs.TagDescription
	out.Id = hs.Id
	out.Name = hs.Name

	out.Vcenter = hs.Vcenter
	out.Datacenter = hs.Datacenter
	out.Cluster = hs.Cluster

	out.KeyName = hs.KeyName
	out.Vendor = hs.Vendor
	out.Model = hs.Model

	out.ProductFullName = hs.ProductFullName
	out.SerialNumber = hs.SerialNumber

	out.VirtualSerialNumber = hs.VirtualSerialNumber
	out.EnclosureSerialNumber = hs.EnclosureSerialNumber
	out.Uuid = hs.Uuid
	out.PowerState = hs.PowerState
	out.InMaintenanceMode = hs.InMaintenanceMode
	out.CpuModel = hs.CpuModel
	out.CpuMhz = hs.CpuMhz
	out.NumCpuPkgs = hs.NumCpuPkgs
	out.NumCpuCores = hs.NumCpuCores
	out.NumCpuThreads = hs.NumCpuThreads
	out.NumHBAs = hs.NumHBAs
	out.NumNics = hs.NumNics
	out.IpAddress = hs.IpAddress
	out.MemorySizeGB = hs.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = hs.HostMaxVirtualDiskCapacity
	out.PerfCount = hs.PerfCount
	out.CpuUsageMaximum = hs.CpuUsageMaximum
	out.CpuUsageMaximumTime = hs.CpuUsageMaximumTime
	out.CpuUsageMinimum = hs.CpuUsageMinimum
	out.CpuUsageMinimumTime = hs.CpuUsageMinimumTime
	out.MemUsageMaximum = hs.MemUsageMaximum
	out.MemUsageMaximumTime = hs.MemUsageMaximumTime
	out.MemUsageMinimum = hs.MemUsageMinimum
	out.MemUsageMinimumTime = hs.MemUsageMinimumTime
	out.DiskUsageMaximum = hs.DiskUsageMaximum
	out.DiskUsageMaximumTime = hs.DiskUsageMaximumTime
	out.DiskUsageMinimum = hs.DiskUsageMinimum
	out.DiskUsageMinimumTime = hs.DiskUsageMinimumTime
	out.DiskWriteMaximum = hs.DiskWriteMaximum
	out.DiskWriteMaximumTime = hs.DiskWriteMaximumTime
	out.DiskWriteMinimum = hs.DiskWriteMinimum
	out.DiskWriteMinimumTime = hs.DiskWriteMinimumTime
	out.DiskReadMaximum = hs.DiskReadMaximum
	out.DiskReadMaximumTime = hs.DiskReadMaximumTime
	out.DiskReadMinimum = hs.DiskReadMinimum
	out.DiskReadMinimumTime = hs.DiskReadMinimumTime
	out.NetUsageMaximum = hs.NetUsageMaximum
	out.NetUsageMaximumTime = hs.NetUsageMaximumTime
	out.NetUsageMinimum = hs.NetUsageMinimum
	out.NetUsageMinimumTime = hs.NetUsageMinimumTime
	out.CpuUsageAverage = hs.CpuUsageAverage
	out.MemUsageAverage = hs.MemUsageAverage
	out.DiskUsageAverage = hs.DiskUsageAverage
	out.DiskWriteAverage = hs.DiskWriteAverage
	out.DiskReadAverage = hs.DiskReadAverage
	out.NetUsageAverage = hs.NetUsageAverage
	return &out
}

// CopyClusterFull создаёт независимую копию сообщения pb.ClusterFull, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту.
func CopyPbHostFull(in *pb.HostFull) *pb.HostFull {
	out := pb.HostFull{
		Id:                         in.Id,
		Vcenter:                    in.Vcenter,
		Datacenter:                 in.Datacenter,
		Cluster:                    in.Cluster,
		Name:                       in.Name,
		KeyName:                    in.KeyName,
		Vendor:                     in.Vendor,
		Model:                      in.Model,
		SerialNumber:               in.SerialNumber,
		VirtualSerialNumber:        in.VirtualSerialNumber,
		EnclosureSerialNumber:      in.EnclosureSerialNumber,
		Uuid:                       in.Uuid,
		PowerState:                 in.PowerState,
		InMaintenanceMode:          in.InMaintenanceMode,
		CpuModel:                   in.CpuModel,
		CpuMhz:                     in.CpuMhz,
		NumCpuPkgs:                 in.NumCpuPkgs,
		NumCpuCores:                in.NumCpuCores,
		NumCpuThreads:              in.NumCpuThreads,
		NumNics:                    in.NumNics,
		NumHBAs:                    in.NumHBAs,
		MemorySizeGB:               in.MemorySizeGB,
		HostMaxVirtualDiskCapacity: in.HostMaxVirtualDiskCapacity,
		CpuUsageMaximum:            in.CpuUsageMaximum,
		CpuUsageMaximumTime:        in.CpuUsageMaximumTime,
		CpuUsageAverage:            in.CpuUsageAverage,
		CpuUsageMinimum:            in.CpuUsageMinimum,
		CpuUsageMinimumTime:        in.CpuUsageMinimumTime,
		MemUsageMaximum:            in.MemUsageMaximum,
		MemUsageMaximumTime:        in.MemUsageMaximumTime,
		MemUsageAverage:            in.MemUsageAverage,
		MemUsageMinimum:            in.MemUsageMinimum,
		MemUsageMinimumTime:        in.MemUsageMinimumTime,
		DiskUsageMaximum:           in.DiskUsageMaximum,
		DiskUsageMaximumTime:       in.DiskUsageMaximumTime,
		DiskUsageAverage:           in.DiskUsageAverage,
		DiskUsageMinimum:           in.DiskUsageMinimum,
		DiskUsageMinimumTime:       in.DiskUsageMinimumTime,
		DiskWriteMaximum:           in.DiskWriteMaximum,
		DiskWriteMaximumTime:       in.DiskWriteMaximumTime,
		DiskWriteAverage:           in.DiskWriteAverage,
		DiskWriteMinimum:           in.DiskWriteMinimum,
		DiskWriteMinimumTime:       in.DiskWriteMinimumTime,
		DiskReadMaximum:            in.DiskReadMaximum,
		DiskReadMaximumTime:        in.DiskReadMaximumTime,
		DiskReadAverage:            in.DiskReadAverage,
		DiskReadMinimum:            in.DiskReadMinimum,
		DiskReadMinimumTime:        in.DiskReadMinimumTime,
		NetUsageMaximum:            in.NetUsageMaximum,
		NetUsageMaximumTime:        in.NetUsageMaximumTime,
		NetUsageAverage:            in.NetUsageAverage,
		NetUsageMinimum:            in.NetUsageMinimum,
		NetUsageMinimumTime:        in.NetUsageMinimumTime,
		PerfCount:                  in.PerfCount,
		TagName:                    in.TagName,
		TagDescription:             in.TagDescription,
	}
	for _, perf := range in.Perf {
		hsperf := pb.HSPerf{
			Id:             perf.Id,
			Start:          perf.Start,
			CpuUsage:       perf.CpuUsage,
			CpuUtilization: perf.CpuUtilization,
			MemUsage:       perf.MemUsage,
			DiskUsage:      perf.DiskUsage,
			DiskWrite:      perf.DiskWrite,
			DiskRead:       perf.DiskRead,
			NetUsage:       perf.NetUsage,
			NetReceived:    perf.NetReceived,
			NetTransmitted: perf.NetTransmitted}
		out.Perf = append(out.Perf, &hsperf)
	}
	return &out
}

// GetVMFiles возвращает список файлов виртуальной машины по её ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
// При обращении к удалённым эндпоинтам ошибки подключения к отдельным эндпоинтам не прерывают обход остальных (накопленный результат всё равно возвращается вызывающему).
