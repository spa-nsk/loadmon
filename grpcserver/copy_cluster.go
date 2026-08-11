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

func CopyClusterFull(pbcl *pb.ClusterFull) *pb.ClusterFull {
	pbclf := pb.ClusterFull{
		Id:                    pbcl.Id,
		Date:                  pbcl.Date,
		Vcenter:               pbcl.Vcenter,
		Datacenter:            pbcl.Datacenter,
		Name:                  pbcl.Name,
		CpuMhzAvg:             pbcl.CpuMhzAvg,
		AllNumCpuPkgs:         pbcl.AllNumCpuPkgs,
		AllNumCpuCores:        pbcl.AllNumCpuCores,
		AllNumCpuThreads:      pbcl.AllNumCpuThreads,
		AllMemorySizeGB:       pbcl.AllMemorySizeGB,
		AllCapacityGb:         pbcl.AllCapacityGb,
		AllFreeSpaceGb:        pbcl.AllFreeSpaceGb,
		AllPercentUsed:        pbcl.AllPercentUsed,
		AllocatedCoreCount:    pbcl.AllocatedCoreCount,
		AllocatedMemorySizeGb: pbcl.AllocatedMemorySizeGb,
		CoreTransferRate:      pbcl.CoreTransferRate,
		MemoryTransferRate:    pbcl.MemoryTransferRate,
		CpuUsageMaximum:       pbcl.CpuUsageMaximum,
		CpuUsageMaximumTime:   pbcl.CpuUsageMaximumTime,
		CpuUsageAverage:       pbcl.CpuUsageAverage,
		CpuUsageMinimum:       pbcl.CpuUsageMinimum,
		CpuUsageMinimumTime:   pbcl.CpuUsageMinimumTime,
		MemUsageMaximum:       pbcl.MemUsageMaximum,
		MemUsageMaximumTime:   pbcl.MemUsageMaximumTime,
		MemUsageAverage:       pbcl.MemUsageAverage,
		MemUsageMinimum:       pbcl.MemUsageMinimum,
		MemUsageMinimumTime:   pbcl.MemUsageMinimumTime,
		DiskUsageMaximum:      pbcl.DiskUsageMaximum,
		DiskUsageMaximumTime:  pbcl.DiskUsageMaximumTime,
		DiskUsageAverage:      pbcl.DiskUsageAverage,
		DiskUsageMinimum:      pbcl.DiskUsageMinimum,
		DiskUsageMinimumTime:  pbcl.DiskUsageMinimumTime,
		DiskWriteMaximum:      pbcl.DiskWriteMaximum,
		DiskWriteMaximumTime:  pbcl.DiskWriteMaximumTime,
		DiskWriteAverage:      pbcl.DiskWriteAverage,
		DiskWriteMinimum:      pbcl.DiskWriteMinimum,
		DiskWriteMinimumTime:  pbcl.DiskWriteMinimumTime,
		DiskReadMaximum:       pbcl.DiskReadMaximum,
		DiskReadMaximumTime:   pbcl.DiskReadMaximumTime,
		DiskReadAverage:       pbcl.DiskReadAverage,
		DiskReadMinimum:       pbcl.DiskReadMinimum,
		DiskReadMinimumTime:   pbcl.DiskReadMinimumTime,
		NetUsageMaximum:       pbcl.NetUsageMaximum,
		NetUsageMaximumTime:   pbcl.NetUsageMaximumTime,
		NetUsageAverage:       pbcl.NetUsageAverage,
		NetUsageMinimum:       pbcl.NetUsageMinimum,
		NetUsageMinimumTime:   pbcl.NetUsageMinimumTime,
		PerfCount:             pbcl.PerfCount,
		HostFlag:              pbcl.HostFlag,
		HostCount:             pbcl.HostCount,
		VMCount:               pbcl.VMCount,
		Perf:                  make([]*pb.CLPerf, len(pbcl.Perf)),
		Storages:              make([]*pb.Datastore, len(pbcl.Storages))}

	for i, perfcl := range pbcl.Perf {
		perf := pb.CLPerf{
			Id:             perfcl.Id,
			Start:          perfcl.Start,
			CpuUsage:       perfcl.CpuUsage,
			CpuUtilization: perfcl.CpuUtilization,
			MemUsage:       perfcl.MemUsage,
			DiskUsage:      perfcl.DiskUsage,
			DiskWrite:      perfcl.DiskWrite,
			DiskRead:       perfcl.DiskRead,
			NetUsage:       perfcl.NetUsage,
			NetReceived:    perfcl.NetReceived,
			NetTransmitted: perfcl.NetTransmitted}
		pbclf.Perf[i] = &perf
	}
	for i, st := range pbcl.Storages {
		stn := pb.Datastore{
			Id:          st.Id,
			UID:         st.UID,
			StorageName: st.StorageName,
			CapacityGb:  st.CapacityGb,
			FreeSpaceGb: st.FreeSpaceGb,
			PercentUsed: st.PercentUsed}
		stn.VmKeys.Count = st.VmKeys.Count
		for _, key := range st.VmKeys.KeyRecs {
			stn.VmKeys.KeyRecs = append(stn.VmKeys.KeyRecs, key)
		}
		pbclf.Storages[i] = &stn
	}
	return &pbclf
}

// CopyClusterStorages создаёт независимую копию сообщения pb.Datastore, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту.
func CopyClusterStorages(st *pb.Datastore) *pb.Datastore {
	stn := pb.Datastore{
		Id:          st.Id,
		UID:         st.UID,
		StorageName: st.StorageName,
		CapacityGb:  st.CapacityGb,
		FreeSpaceGb: st.FreeSpaceGb,
		PercentUsed: st.PercentUsed}
	stn.VmKeys.Count = st.VmKeys.Count
	for _, key := range st.VmKeys.KeyRecs {
		stn.VmKeys.KeyRecs = append(stn.VmKeys.KeyRecs, key)
	}
	return &stn
}

// GetClusterMetrics отдаёт клиенту потоком историю метрик производительности конкретного кластера (предварительно пересчитывая агрегаты через cl.CalcPerfomance()). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
// Возвращает ошибку, если кластер с указанными vCenter/датацентром/именем не найден.
func CopyCLPerf(in *pb.CLPerf) *pb.CLPerf {
	clperf := pb.CLPerf{
		Id:             in.Id,
		Start:          in.Start,
		CpuUsage:       in.CpuUsage,
		CpuUtilization: in.CpuUtilization,
		MemUsage:       in.MemUsage,
		DiskUsage:      in.DiskUsage,
		DiskWrite:      in.DiskWrite,
		DiskRead:       in.DiskRead,
		NetUsage:       in.NetUsage,
		NetReceived:    in.NetReceived,
		NetTransmitted: in.NetTransmitted}
	return &clperf
}

// GetClusterMedium отдаёт клиенту потоком информацию о кластерах с агрегированными метриками производительности (без списка хостов/хранилищ). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func CopyClusterMedium(pbcl *pb.Cluster) *pb.Cluster {
	pbclf := pb.Cluster{
		Id:                    pbcl.Id,
		Date:                  pbcl.Date,
		Vcenter:               pbcl.Vcenter,
		Datacenter:            pbcl.Datacenter,
		Name:                  pbcl.Name,
		CpuMhzAvg:             pbcl.CpuMhzAvg,
		AllNumCpuPkgs:         pbcl.AllNumCpuPkgs,
		AllNumCpuCores:        pbcl.AllNumCpuCores,
		AllNumCpuThreads:      pbcl.AllNumCpuThreads,
		AllMemorySizeGB:       pbcl.AllMemorySizeGB,
		AllCapacityGb:         pbcl.AllCapacityGb,
		AllFreeSpaceGb:        pbcl.AllFreeSpaceGb,
		AllPercentUsed:        pbcl.AllPercentUsed,
		AllocatedCoreCount:    pbcl.AllocatedCoreCount,
		AllocatedMemorySizeGb: pbcl.AllocatedMemorySizeGb,
		CoreTransferRate:      pbcl.CoreTransferRate,
		MemoryTransferRate:    pbcl.MemoryTransferRate,
		CpuUsageMaximum:       pbcl.CpuUsageMaximum,
		CpuUsageMaximumTime:   pbcl.CpuUsageMaximumTime,
		CpuUsageAverage:       pbcl.CpuUsageAverage,
		CpuUsageMinimum:       pbcl.CpuUsageMinimum,
		CpuUsageMinimumTime:   pbcl.CpuUsageMinimumTime,
		MemUsageMaximum:       pbcl.MemUsageMaximum,
		MemUsageMaximumTime:   pbcl.MemUsageMaximumTime,
		MemUsageAverage:       pbcl.MemUsageAverage,
		MemUsageMinimum:       pbcl.MemUsageMinimum,
		MemUsageMinimumTime:   pbcl.MemUsageMinimumTime,
		DiskUsageMaximum:      pbcl.DiskUsageMaximum,
		DiskUsageMaximumTime:  pbcl.DiskUsageMaximumTime,
		DiskUsageAverage:      pbcl.DiskUsageAverage,
		DiskUsageMinimum:      pbcl.DiskUsageMinimum,
		DiskUsageMinimumTime:  pbcl.DiskUsageMinimumTime,
		DiskWriteMaximum:      pbcl.DiskWriteMaximum,
		DiskWriteMaximumTime:  pbcl.DiskWriteMaximumTime,
		DiskWriteAverage:      pbcl.DiskWriteAverage,
		DiskWriteMinimum:      pbcl.DiskWriteMinimum,
		DiskWriteMinimumTime:  pbcl.DiskWriteMinimumTime,
		DiskReadMaximum:       pbcl.DiskReadMaximum,
		DiskReadMaximumTime:   pbcl.DiskReadMaximumTime,
		DiskReadAverage:       pbcl.DiskReadAverage,
		DiskReadMinimum:       pbcl.DiskReadMinimum,
		DiskReadMinimumTime:   pbcl.DiskReadMinimumTime,
		NetUsageMaximum:       pbcl.NetUsageMaximum,
		NetUsageMaximumTime:   pbcl.NetUsageMaximumTime,
		NetUsageAverage:       pbcl.NetUsageAverage,
		NetUsageMinimum:       pbcl.NetUsageMinimum,
		NetUsageMinimumTime:   pbcl.NetUsageMinimumTime,
		PerfCount:             pbcl.PerfCount,
		HostFlag:              pbcl.HostFlag,
		HostCount:             pbcl.HostCount,
		VMCount:               pbcl.VMCount}

	return &pbclf
}

// CopyDatastore создаёт независимую копию сообщения pb.Datastore вместе со списком ключей связанных ВМ (VmKeys).
func CopyDatastore(in *pb.Datastore) *pb.Datastore {
	out := pb.Datastore{
		Id:          in.Id,
		UID:         in.UID,
		StorageName: in.StorageName,
		CapacityGb:  in.CapacityGb,
		FreeSpaceGb: in.FreeSpaceGb,
		PercentUsed: in.PercentUsed,
	}

	out.VmKeys.Count = in.VmKeys.Count
	for _, key := range in.VmKeys.KeyRecs {
		out.VmKeys.KeyRecs = append(out.VmKeys.KeyRecs, key)
	}
	return &out
}

// GetHostFull отдаёт клиенту потоком полную информацию о хостах, отфильтрованную по vCenter/датацентру/кластеру/имени, с пересчитанными агрегированными метриками. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
