package grpcclient

// Файл message_cli.go содержит функции-конвертеры (маппинг) между доменными
// моделями пакетов vspheremetrics/models/oneview и сгенерированными gRPC-
// сообщениями пакета protocol. Функции вида MessageToXxx конвертируют
// gRPC-сообщение в доменную модель (приём данных от сервера), а функции вида
// XxxToMessage — доменную модель в gRPC-сообщение (отправка данных на сервер).
// Все функции симметрично защищены от nil-указателей на входе и просто
// ничего не делают в этом случае, ошибок не возвращают (это чистое
// копирование полей, а не операции, которые могут завершиться неудачей).

import (
	"fmt"
	"loadmon/models"
	"loadmon/oneview"
	pb "loadmon/protocol"
	"strconv"
	vsphere "loadmon/vspheremetrics"

	"github.com/HewlettPackard/oneview-golang/utils"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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
func MessageToServerSprRecod(in *pb.ServerSprRecords, out *models.ServerSprRecords) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                                  //int64
	out.OldId = in.OldId                            //int64      ссылка на историческую запись после переименования. 0 если такой нет. Может быть цепочка записей.
	out.Type = int(in.Type)                          //int        0 - физический сервер 1 - виртуальный 2-host vmware
	out.TypeMicroservice = int(in.TypeMicroservice) //int        0 не указано принадлежность к микросервисам, роль в этом пространстве
	out.RoleType = int(in.RoleType)                  //int        0-не указано 1-APP 2-DB
	out.TypeWork = int(in.TypeWork)                  //int        0 - не указано 1 - продуктовая среда 2- тестовая
	out.CriticalLevel = int(in.CriticalLevel)        //int        контур 0 не указано 1-Vital 2-Critical 3-Secondary 4-Non Priority
	out.VMSprId = in.VMSprId                        //int64      ссылка на справочник VM если Type = 1 иначе null
	out.Stat = int(in.Stat)                          //int        0 историческая запись 1 используется 2 помечено для удаления 3-архивная запись после переименования 911 - новая запись,неполные
	out.PowerState = int(in.PowerState)              //int        0 не указано 1 включен 2 выключен (последнее состояние) при отображении переопределять текущим состоянием если есть
	out.Active = int(in.Active)                      //int        1 активен 2 выведен из эксплуатации
	out.TimeActiveState = in.TimeActiveState.AsTime() //time.Time  время изменения статуса активности введения в эксплуатацию
	out.ActivatorUser = in.ActivatorUser            //int64      автор записи
	out.TimeDeActiveState = in.TimeDeActiveState.AsTime() //time.Time время выведения из эксплуатации
	out.DeActivatorUser = in.DeActivatorUser        //int64      автор записи
	out.TimeCreate = in.TimeCreate.AsTime()          //time.Time  время создания записи
	out.CreatorUser = in.CreatorUser                //int64      автор записи
	out.BusinesPrivilage = in.BusinesPrivilage      //int64      ограничение по редактированию если указано то только пользователь из этой группы может редактировать данные
	out.Name = in.Name                              //string
	out.NameFQDN = in.NameFQDN                        //string     полное имя сервера
	out.NameInfo = in.NameInfo                      //string     альтернативное имя сервера
	out.IPAddress = in.IPAddress                    //string     список IP адресов сервера при наличие информации подгружается из конфигурации VM или вводится вручную
	out.MacAddress = in.MacAddress                  //
	out.IPSharedVIP = in.IPSharedVIP                //string     разделяемый IP вводится вручную
	out.Network = in.Network                        //string     сеть vlan144
	out.NetworkName = in.NetworkName                //string     наименование сети сеть DMZ Prom
	out.CPUCount = in.CPUCount                      //float64    Количество физических процессоров
	out.CPULogical = in.CPULogical                  //float64    количество логических CPU
	out.MemSize = in.MemSize                           //float64  размер оперативной памяти сервера
	out.DiskSize = in.DiskSize                         //float64  суммарный размер всех дисков
	out.Service = in.Service                           //string   сервис в котором задействован сервер
	out.ServiceId = in.ServiceId                       //int64    ссылка на базу сервисов
	out.ITSystemKey = in.ITSystemKey                   //string   ссылка на ИТ систему из jira https://jira.bcs.ru/rest/assets/latest/aql/objects?qlQuery=objectTypeID=21&includeAttributes=true&p
	out.OS = in.OS                                     //string   операционная система
	out.Component = in.Component                       //string   компонента Kafka, service BD ....
	out.Owner = in.Owner                               //string   владелец сервера (отвественный за сервер)
	out.BusinessOwner = in.BusinessOwner               //string   владелец от бизнеса
	out.ITOwner = in.ITOwner                           //string   владелец со стороны ИТ
	out.ProductOwner = in.ProductOwner                 //string   Владелец продукта
	out.Support = in.Support                           //string   ответственные за поддержку
	out.SupportTeam = in.SupportTeam                   //string   Подразделение поддержки
	out.SystemType = in.SystemType                     //string   тип системы техническая ...
	out.LicensingType = in.LicensingType               //string   тип лицензирования
	out.SolutionType = in.SolutionType                 //string   вариант решения (коробочной)
	out.Users = in.Users                               //string   пользователи системы
	out.Allocation = in.Allocation                     //string   аллокация (подразделение)
	out.Anotation = in.Anotation                       //string   описане сервера
	out.Role = in.Role                                 //string   роль сервера (сервер баз данных, сервер приложений, хост)
	out.Details = in.Details                           //string   детали по серверу в свободном формате (примечание)
	out.DPCAddress = int(in.DPCAddress)                //int      ЦОД по справочнику адрес
	out.DPCOffice = int(in.DPCOffice)                  //int      ЦОД по справочнику комната с адресом
	out.SZHistory = in.SZhistory                       //string   номера сз через пробел в хронологическом порядке
	out.Warning = in.Warning                           //string
	out.Level = in.Level                               //float64
	out.Monitoring = int(in.Monitoring)                //int      0- не указано 1-включен мониторинг 2-отключен мониторинг
	out.Backuped = int(in.Backuped)                    //int      0 - не указано 1-включено резервное копирование 2-отключено резервное копирование
	out.TimeActiveMonitoring = in.TimeActiveMonitoring.AsTime() //time.Time время постановки на мониторинг
	out.MonitoringSZ = in.MonitoringSZ                 //string   основание постановки на мониторинг
	out.Id_User = in.Id_User                           //int64
	out.Time_Mfy = in.Time_Mfy.AsTime()                //time.Time
	out.Privilage = int(in.Privilage)                  //int      уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привилегии пользователя)
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Cluster = in.Cluster
	out.ClusterId = in.ClusterId
}

// MessageToVMSprRecod конвертирует gRPC-сообщение pb.VMSprRecord (запись справочника виртуальных машин) в доменную модель models.VMSprRecord. При in == nil или out == nil ничего не делает.
func MessageToVMSprRecod(in *pb.VMSprRecord, out *models.VMSprRecord) {
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
}

// ServerHardwareToMessage конвертирует доменную модель oneview.ServerHardware (данные сервера HPE OneView) в gRPC-сообщение pb.ServerHardware для передачи по сети. При in == nil или out == nil ничего не делает.
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

	vsn := in.Base.VirtualSerialNumber.String()

	if vsn == "null" {
		out.Base.VirtualSerialNumber = ""
	} else {
		out.Base.VirtualSerialNumber = vsn
	}

	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = in.Base.UUID.String()
	out.Base.FormFactor = in.Base.FormFactor
	out.Base.OperatingSystem = in.OperatingSystem

	out.Memory = &pb.ServerHardwareMemory{}

	out.Memory.Modified = timestamppb.New(in.Memory.Modified)
	out.Memory.Count = int32(in.Memory.Count)
	out.Memory.Name = in.Memory.Name

	for _, p := range in.Memory.Data {
		m := pb.MemoryModule{}
		m.BaseModuleType = p.BaseModuleType
		m.CapacityMiB = int32(p.CapacityMiB)
		m.DeviceLocator = p.DeviceLocator
		m.Manufacturer = p.Manufacturer
		m.MemoryDeviceType = p.MemoryDeviceType
		m.Name = p.Name
		m.OperatingSpeedMhz = int32(p.OperatingSpeedMhz)
		m.PartNumber = p.PartNumber
		m.RankCount = int32(p.RankCount)
		m.Status = &pb.MemoryStatus{Healt: p.Status.Health, State: p.Status.State}
		out.Memory.Data = append(out.Memory.Data, &m)
	}

	out.Storage = &pb.ServerHardwareLocalStorage{}

	out.Storage.Modified = timestamppb.New(in.Storage.Modified)
	out.Storage.Count = int32(in.Storage.Count)
	out.Storage.Name = in.Storage.Name
	for _, p := range in.Storage.Data {
		s := pb.LocalStorage{}

		s.AdapterType = p.AdapterType                                   //тип адаптера "SmartArray"
		s.CacheMemorySizeMiB = int32(p.CacheMemorySizeMiB)              //размер кэша контроллера в мегабайтах
		s.CacheModuleSerialNumber = p.CacheModuleSerialNumber                                                                   //серийный номер контроллера
		s.EncryptionCspTestPassed = p.EncryptionCspTestPassed                                                                   //результат тестового прохода
		s.EncryptionEnabled = p.EncryptionEnabled                                                                               //шифрование включено?
		s.EncryptionSelfTestPassed = p.EncryptionSelfTestPassed                                                                 //результата теста шифрования
		s.Location = p.Location                                                                                                 //расположение "Slot 3"
		s.Model = p.Model                                                                                                       //модель "Smart array p840 Controller"
		s.Name = p.Name                                                                                                         //наименование "Controller in Slot 3"
		s.SerialNumber = p.SerialNumber                                                                                         //серийный номер
		s.Status = &pb.Status{Health: p.Status.Health, State: p.Status.State}                                                   //состояние
		s.CacheModuleStatus = &pb.CacheModuleStatus{Health: p.CacheModuleStatus.Health}                                         //состояние кэша
		s.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: p.FirmwareVersion.Current.VersionString}}    //прошивка
		s.ControllerBoard = &pb.ControllerBoard{Status: &pb.ControllerSatus{Health: p.ControllerBoard.Status.Health, State: p.ControllerBoard.Status.State}} // p.ControllerBoard      *ControllerBo
		for _, l := range p.LogicalDrives {                                                                                     //томы (логические диски)
			nl := pb.LocalLogicalDrive{}
			nl.CapacityMiB = l.CapacityMiB
			nl.LogicalDriveEncryption = l.LogicalDriveEncryption
			nl.LogicalDriveNumber = int32(l.LogicalDriveNumber)
			nl.LogicalDriveType = l.LogicalDriveType
			nl.Raid = l.Raid
			nl.Status = &pb.Status{Health: l.Status.Health, State: l.Status.State}
			for _, d := range l.DataDrives {
				ld := pb.LogicalDataDrive{}
				ld.CapacityMiB = d.CapacityMiB                 //float64               //емкость диска в мегабайтах
				for _, ps := range d.DiskDriveStatusReasons { //[]string              //статусы "None"
					ld.DiskDriveStatusReasons = append(ld.DiskDriveStatusReasons, ps)
				}
				ld.DiskDriveUse = d.DiskDriveUse                                                                                       //string           //использование диска в томе "Data"
				ld.EncryptedDrive = d.EncryptedDrive                                                                                   //bool             //включено ли шифрование true, false
				ld.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: d.FirmwareVersion.Current.VersionString}} //*FirmwareVersion //текущая версия прошивки
				ld.Location = d.Location                                                                                               //string           //расположение "1I:1:7"
				ld.LocationFormat = d.LocationFormat                                                                                   //string           //формат строки расположения "Control
				ld.MediaType = d.MediaType                                                                                             //string           //тип диска "HDD", ...
				ld.Model = d.Model                                                                                                     //string           //модель "MB4000JVYZQ"
				ld.SerialNumber = d.SerialNumber                                                                                       //string           //серийный номер "ZC17LE2X"
				ld.Status = &pb.Status{}                                                                                               //d.Status         //*Status            //состояние

				nl.DataDrives = append(nl.DataDrives, &ld)
			}

			s.LogicalDrives = append(s.LogicalDrives, &nl)
		}
		for _, pd := range p.PhysicalDrives {
			d := pb.LocalPhysicalDrive{}
			d.CapacityMiB = pd.CapacityMiB
			for _, ds := range pd.DiskDriveStatusReasons {
				d.DiskDriveStatusReasons = append(d.DiskDriveStatusReasons, ds)
			}
			d.DiskDriveUse = pd.DiskDriveUse
			d.EncryptedDrive = pd.EncryptedDrive
			d.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
			d.Location = pd.Location
			d.LocationFormat = pd.LocationFormat
			d.MediaType = pd.MediaType
			d.Model = pd.Model
			d.SerialNumber = pd.SerialNumber
			d.Status = &pb.Status{Health: pd.Status.Health, State: pd.Status.State}
			s.PhysicalDrives = append(s.PhysicalDrives, &d)
		}
		for _, se := range p.StorageEnclosures {
			enc := pb.LocalStorageEnclosure{}
			enc.DriveBayCount = int32(se.DriveBayCount)
			enc.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: se.FirmwareVersion.Current.VersionString}}
			enc.Location = se.Location
			enc.LocationFormat = se.LocationFormat
			enc.Status = &pb.Status{Health: se.Status.Health, State: se.Status.State}
			s.StorageEnclosures = append(s.StorageEnclosures, &enc)
		}
		out.Storage.Data = append(out.Storage.Data, &s)
	}
	out.Datacenter = &pb.DatacenterInfo{}
	out.Datacenter.Width = int32(in.Datacenter.Width)
	out.Datacenter.Depth = int32(in.Datacenter.Depth)
	out.Datacenter.Currency = in.Datacenter.Currency
	out.Datacenter.DeratingType = in.Datacenter.DeratingType
	out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
	out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
	for _, ct := range in.Datacenter.Contents {
		out.Datacenter.Contents = append(out.Datacenter.Contents, &pb.Contents{X: ct.X, Y: ct.Y, Rotation: ct.Rotation})
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
	out.EnclosureInfo.Type = in.EnclosureInfo.Type                       //string
	out.EnclosureInfo.Category = in.EnclosureInfo.Category               //string
	out.EnclosureInfo.Created = timestamppb.New(in.EnclosureInfo.Created)   //*timestamppb.Timestamp
	out.EnclosureInfo.Modified = timestamppb.New(in.EnclosureInfo.Modified) //*timestamppb.Timestamp
	out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState       //string
	out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason         //string
	out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType     //string
	out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel   //string
	out.EnclosureInfo.UUID = in.EnclosureInfo.UUID                       //string
	out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber       //string
	out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber           //string
	out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState //string
	out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent //string
	out.EnclosureInfo.DeviceBayCount = int32(in.EnclosureInfo.DeviceBayCount) //int32

	//out.EnclosureInfo.DeviceBays = in.EnclosureInfo.DeviceBays             //[]*DeviceBayInfo
	for _, dev := range in.EnclosureInfo.DeviceBays {
		nd := pb.DeviceBayInfo{}
		nd.Type = dev.Type                                               //string
		nd.BayNumber = int32(dev.BayNumber)                              //int32
		nd.DevicePresence = dev.DevicePresence                           //string
		nd.CoveredByDevice = dev.CoveredByDevice                         //string
		nd.URI = dev.URI                                                 //string
		nd.Category = dev.Category                                       //string
		nd.AvailableForHalfHeightProfile = dev.AvailableForHalfHeightProfile //bool
		nd.AvailableForFullHeightProfile = dev.AvailableForFullHeightProfile //bool
		nd.DeviceBayType = dev.DeviceBayType                             //string
		nd.DeviceFormFactor = dev.DeviceFormFactor                       //string
		nd.BayPowerState = dev.BayPowerState                             //string
		nd.ChangeState = dev.ChangeState                                 //string
		nd.AvailableForHalfHeightDoubleWideProfile = dev.AvailableForHalfHeightDoubleWideProfile //bool
		nd.AvailableForFullHeightDoubleWideProfile = dev.AvailableForFullHeightDoubleWideProfile //bool

		out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, &nd)
	}

	out.EnclosureInfo.InterconnectBayCount = int32(in.EnclosureInfo.InterconnectBayCount) //int32

	//out.EnclosureInfo.InterconnectBays = in.EnclosureInfo.InterconnectBays //[]*InterconnectBayInfo
	for _, con := range in.EnclosureInfo.InterconnectBays {
		ncon := pb.InterconnectBayInfo{}

		ncon.BayNumber = int32(con.BayNumber)           //int32
		ncon.InterconnectModel = con.InterconnectModel ////string
		ncon.SerialNumber = con.SerialNumber           //string
		ncon.InterconnectBayType = con.InterconnectBayType //string
		ncon.ChangeState = con.ChangeState             //string
		ncon.BayPowerState = con.BayPowerState         //string

		out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, &ncon)
	}

	out.EnclosureInfo.FanBayCount = int32(in.EnclosureInfo.FanBayCount) //int32

	//out.EnclosureInfo.FanBays = in.EnclosureInfo.FanBays //[]*FanBayInfo
	for _, fan := range in.EnclosureInfo.FanBays {
		nfan := pb.FanBayInfo{}
		nfan.BayNumber = int32(fan.BayNumber)        //int32
		nfan.DevicePresence = fan.DevicePresence     //string
		nfan.DeviceRequired = fan.DeviceRequired     //bool
		nfan.Status = fan.Status                     //string
		nfan.Model = fan.Model                       //string
		nfan.PartNumber = fan.PartNumber             //string
		nfan.SparePartNumber = fan.SparePartNumber   //string
		nfan.FanBayType = fan.FanBayType             //string
		nfan.ChangeState = fan.ChangeState           //string
		nfan.State = fan.State                       //string
		out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, &nfan)
	}

	out.EnclosureInfo.PowerSupplyBayCount = int32(in.EnclosureInfo.PowerSupplyBayCount) //int32

	//out.EnclosureInfo.PowerSupplyBays = in.EnclosureInfo.PowerSupplyBays //[]*PowerSupplyBayInfo
	for _, pwr := range in.EnclosureInfo.PowerSupplyBays {
		npwr := pb.PowerSupplyBayInfo{}
		npwr.BayNumber = int32(pwr.BayNumber)           //int32
		npwr.DevicePresence = pwr.DevicePresence       //string
		npwr.Status = pwr.Status                       //string
		npwr.Model = pwr.Model                         //string
		npwr.SerialNumber = pwr.SerialNumber           //string
		npwr.PartNumber = pwr.PartNumber               //string
		npwr.SparePartNumber = pwr.SparePartNumber     //string
		npwr.PowerSupplyBayType = pwr.PowerSupplyBayType //string
		npwr.ChangeState = pwr.ChangeState             //string
		out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, &npwr)
	}

	out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged               //bool
	out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware //bool

	//out.EnclosureInfo.ManagerBays = in.EnclosureInfo.ManagerBays //[]*ManagerBayInfo
	for _, man := range in.EnclosureInfo.ManagerBays {
		nman := pb.ManagerBayInfo{}

		nman.BayNumber = int32(man.BayNumber)    //int32
		nman.ManagerType = man.ManagerType       //string
		nman.BayPowerState = man.BayPowerState   //string
		nman.FwVersion = man.FwVersion           //string
		nman.DevicePresence = man.DevicePresence //string
		nman.Role = man.Role                     //string
		nman.IPAddress = man.IPAddress           //string
		nman.ChangeState = man.ChangeState       //string
		nman.FwBuildDate = man.FwBuildDate       //string
		nman.DhcpIpv6Enable = man.DhcpIpv6Enable //bool
		nman.FqdnHostName = man.FqdnHostName     //string
		nman.DhcpEnable = man.DhcpEnable         //bool

		out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, &nman)
	}

	out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState                             //string
	out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState   //string
	out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType     //string
	out.EnclosureInfo.Status = in.EnclosureInfo.Status                                         //string
	out.EnclosureInfo.Name = in.EnclosureInfo.Name                                             //string
	out.EnclosureInfo.State = in.EnclosureInfo.State                                           //string
	out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP             //string
	out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP               //string
	out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag                                     //string
	out.EnclosureInfo.RackName = in.EnclosureInfo.RackName                                     //string
	out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID                               //string
	out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName                           //string
	out.EnclosureInfo.VcmMode = fmt.Sprint(in.EnclosureInfo.VcmMode) //bool
	out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL                                         //string
	out.EnclosureInfo.OaBays = int32(in.EnclosureInfo.OaBays)                                 //int32
	out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState                         //string

	out.BayInfo = &pb.DeviceBayInfo{}

	out.BayInfo.Type = in.BayInfo.Type                                 //string
	out.BayInfo.BayNumber = int32(in.BayInfo.BayNumber)               //int32
	out.BayInfo.DevicePresence = in.BayInfo.DevicePresence             //string
	out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice                    //string
	out.BayInfo.URI = in.BayInfo.URI                                            //string
	out.BayInfo.Category = in.BayInfo.Category                                  //string
	out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile //bool
	out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile //bool
	out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType                        //string
	out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor                  //string
	out.BayInfo.BayPowerState = in.BayInfo.BayPowerState                        //string
	out.BayInfo.ChangeState = in.BayInfo.ChangeState                            //string
	out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile //bool
	out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile //bool
}

// MessageToServerHardware конвертирует gRPC-сообщение pb.ServerHardware обратно в доменную модель oneview.ServerHardware. При in == nil или out == nil ничего не делает.
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

	vsn := in.Base.VirtualSerialNumber

	if vsn == "null" {
		out.Base.VirtualSerialNumber = utils.Nstring("")
	} else {
		out.Base.VirtualSerialNumber = utils.Nstring(vsn)
	}

	out.Base.FormFactor = in.Base.FormFactor

	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = utils.Nstring(in.Base.UUID)

	out.OperatingSystem = in.Base.OperatingSystem

	out.Memory.Modified = in.Memory.Modified.AsTime()
	out.Memory.Count = int(in.Memory.Count)
	out.Memory.Name = in.Memory.Name

	for _, p := range in.Memory.Data {
		m := oneview.MemoryModule{}
		m.BaseModuleType = p.BaseModuleType
		m.CapacityMiB = int(p.CapacityMiB)
		m.DeviceLocator = p.DeviceLocator
		m.Manufacturer = p.Manufacturer
		m.MemoryDeviceType = p.MemoryDeviceType
		m.Name = p.Name
		m.OperatingSpeedMhz = int(p.OperatingSpeedMhz)
		m.PartNumber = p.PartNumber
		m.RankCount = int(p.RankCount)
		m.Status = oneview.MemoryStatus{Health: p.Status.Healt, State: p.Status.State}
		out.Memory.Data = append(out.Memory.Data, m)
	}

	out.Storage.Modified = in.Storage.Modified.AsTime()
	out.Storage.Count = int(in.Storage.Count)
	out.Storage.Name = in.Storage.Name
	for _, p := range in.Storage.Data {
		s := oneview.LocalStorage{}

		s.AdapterType = p.AdapterType                                                                                     //тип адаптера "SmartArray"
		s.CacheMemorySizeMiB = int(p.CacheMemorySizeMiB)                                                                  //размер кэша контроллера в мегабайтах
		s.CacheModuleSerialNumber = p.CacheModuleSerialNumber                                                              //серийный номер контроллера
		s.EncryptionCspTestPassed = p.EncryptionCspTestPassed                                                             //тезультат тестового прохода
		s.EncryptionEnabled = p.EncryptionEnabled                                                                         //шифрование включено?
		s.EncryptionSelfTestPassed = p.EncryptionSelfTestPassed                                                           //результата теста шифрования
		s.Location = p.Location                                                                                           //расположение "Slot 3"
		s.Model = p.Model                                                                                                 //модель "Smart array p840 Controlle
		s.Name = p.Name                                                                                                   //наименование "Controller in Slot 3
		s.SerialNumber = p.SerialNumber                                                                                   //серийный номер
		s.Status = oneview.Status{Health: p.Status.Health, State: p.Status.State}                                          //состояние
		s.CacheModuleStatus = oneview.CacheModuleStatus{Health: p.CacheModuleStatus.Health}                                //состояние кэша
		s.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: p.FirmwareVersion.Current.VersionString}} //прошивка
		s.ControllerBoard = oneview.ControllerBoard{Status: oneview.ControllerSatus{Health: p.ControllerBoard.Status.Health, State: p.ControllerBoard.Status.State}} //  p.ControllerBoard           *Cont
		for _, l := range p.LogicalDrives {                                                                               //томы (логические диски)
			nl := oneview.LocalLogicalDrive{}
			nl.CapacityMiB = l.CapacityMiB
			nl.LogicalDriveEncryption = l.LogicalDriveEncryption
			nl.LogicalDriveNumber = int(l.LogicalDriveNumber)
			nl.LogicalDriveType = l.LogicalDriveType
			nl.Raid = l.Raid
			nl.Status = oneview.Status{Health: l.Status.Health, State: l.Status.State} //
			for _, d := range l.DataDrives {
				ld := oneview.LogicalDataDrive{}
				ld.CapacityMiB = d.CapacityMiB                 //float64		//емкость диска в мегабайтах
				for _, ps := range d.DiskDriveStatusReasons { //[]string		//стаатусы "None"
					ld.DiskDriveStatusReasons = append(ld.DiskDriveStatusReasons, ps)
				}
				ld.DiskDriveUse = d.DiskDriveUse                                                                  //string		//использование диска в томе "
				ld.EncryptedDrive = d.EncryptedDrive                                                              //bool		//включено ли шифрование true,
				ld.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: d.FirmwareVersion.Current.VersionString}} //*FirmwareVersion //текущая версия прошивки
				ld.Location = d.Location                                                                          //string		//расположение "1I:1:7"
				ld.LocationFormat = d.LocationFormat                                                              //string		//формат строки расположения
				ld.MediaType = d.MediaType                                                                        //string		//тип диска "HDD", ...
				ld.Model = d.Model                                                                                //string		//модель "MB4000JVYZQ"
				ld.SerialNumber = d.SerialNumber                                                                  //string		//серийный номер "ZC17LE2X"
				ld.Status = oneview.Status{}                                                                      //d.Status		//*Status		//состояние

				nl.DataDrives = append(nl.DataDrives, ld)
			}

			s.LogicalDrives = append(s.LogicalDrives, nl)
		}
		for _, pd := range p.PhysicalDrives {
			d := oneview.LocalPhysicalDrive{}
			d.CapacityMiB = pd.CapacityMiB
			for _, ds := range pd.DiskDriveStatusReasons {
				d.DiskDriveStatusReasons = append(d.DiskDriveStatusReasons, ds)
			}
			d.DiskDriveUse = pd.DiskDriveUse
			d.EncryptedDrive = pd.EncryptedDrive
			d.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
			d.Location = pd.Location
			d.LocationFormat = pd.LocationFormat
			d.MediaType = pd.MediaType
			d.Model = pd.Model
			d.SerialNumber = pd.SerialNumber
			d.Status = oneview.Status{Health: pd.Status.Health, State: pd.Status.State}
			s.PhysicalDrives = append(s.PhysicalDrives, d)
		}
		for _, se := range p.StorageEnclosures {
			enc := oneview.LocalStorageEnclosure{}
			enc.DriveBayCount = int(se.DriveBayCount)
			enc.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: se.FirmwareVersion.Current.VersionString}}
			enc.Location = se.Location
			enc.LocationFormat = se.LocationFormat
			enc.Status = oneview.Status{Health: se.Status.Health, State: se.Status.State}
			s.StorageEnclosures = append(s.StorageEnclosures, enc)
		}
		out.Storage.Data = append(out.Storage.Data, s)
	}
	out.Datacenter.Width = int(in.Datacenter.Width)
	out.Datacenter.Depth = int(in.Datacenter.Depth)
	out.Datacenter.Currency = in.Datacenter.Currency
	out.Datacenter.DeratingType = in.Datacenter.DeratingType
	out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
	out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
	for _, ct := range in.Datacenter.Contents {
		out.Datacenter.Contents = append(out.Datacenter.Contents, oneview.Content{X: ct.X, Y: ct.Y, Rotation: ct.Rotation})
	}

	out.Datacenter.Category = in.Datacenter.Category
	out.Datacenter.Created = in.Datacenter.Created.AsTime()
	out.Datacenter.Modified = in.Datacenter.Modified.AsTime()
	out.Datacenter.ID = in.Datacenter.ID
	out.Datacenter.UUID = in.Datacenter.UUID
	out.Datacenter.DefaultPowerLineVoltage = int(in.Datacenter.DefaultPowerLineVoltage)
	out.Datacenter.Name = in.Datacenter.Name
	out.Datacenter.State = in.Datacenter.State
	out.Datacenter.Status = in.Datacenter.Status

	out.Rack.UHeight = int(in.Rack.UHeight)
	out.Rack.Width = int(in.Rack.Width)
	out.Rack.Height = int(in.Rack.Height)
	out.Rack.Depth = int(in.Rack.Depth)
	out.Rack.SerialNumber = in.Rack.SerialNumber
	out.Rack.Created = in.Rack.Created.AsTime()
	out.Rack.Modified = in.Rack.Modified.AsTime()
	out.Rack.ID = in.Rack.ID
	out.Rack.UUID = in.Rack.UUID
	out.Rack.Name = in.Rack.Name
	out.Rack.State = in.Rack.State
	out.Rack.Status = in.Rack.Status

	out.EnclosureInfo.Type = in.EnclosureInfo.Type                      //string
	out.EnclosureInfo.Category = in.EnclosureInfo.Category              //string
	out.EnclosureInfo.Created = in.EnclosureInfo.Created.AsTime()      //*timestamppb.Timestamp
	out.EnclosureInfo.Modified = in.EnclosureInfo.Modified.AsTime()    //*timestamppb.Timestamp
	out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState      //string
	out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason        //string
	out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType    //string
	out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel  //string
	out.EnclosureInfo.UUID = in.EnclosureInfo.UUID                      //string
	out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber      //string
	out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber          //string
	out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState //string
	out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent //string
	out.EnclosureInfo.DeviceBayCount = int(in.EnclosureInfo.DeviceBayCount) //int32

	//out.EnclosureInfo.DeviceBays = in.EnclosureInfo.DeviceBays //[]*DeviceBayInfo
	for _, dev := range in.EnclosureInfo.DeviceBays {
		nd := oneview.DeviceBayInfo{}
		nd.Type = dev.Type //string
		nd.BayNumber = int(dev.BayNumber) //int32
		nd.DevicePresence = dev.DevicePresence //string
		nd.CoveredByDevice = dev.CoveredByDevice //string
		nd.URI = dev.URI //string
		nd.Category = dev.Category //string
		nd.AvailableForHalfHeightProfile = dev.AvailableForHalfHeightProfile //bool
		nd.AvailableForFullHeightProfile = dev.AvailableForFullHeightProfile //bool
		nd.DeviceBayType = dev.DeviceBayType //string
		nd.DeviceFormFactor = dev.DeviceFormFactor //string
		nd.BayPowerState = dev.BayPowerState //string
		nd.ChangeState = dev.ChangeState //string
		nd.AvailableForHalfHeightDoubleWideProfile = dev.AvailableForHalfHeightDoubleWideProfile //bool
		nd.AvailableForFullHeightDoubleWideProfile = dev.AvailableForFullHeightDoubleWideProfile //bool

		out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, nd)
	}

	out.EnclosureInfo.InterconnectBayCount = int(in.EnclosureInfo.InterconnectBayCount) //int32

	//out.EnclosureInfo.InterconnectBays = in.EnclosureInfo.InterconnectBays //[]*InterconnectBayInfo
	for _, con := range in.EnclosureInfo.InterconnectBays {
		ncon := oneview.InterconnectBayInfo{}

		ncon.BayNumber = int(con.BayNumber) //int32
		ncon.InterconnectModel = con.InterconnectModel //string
		ncon.SerialNumber = con.SerialNumber //string
		ncon.InterconnectBayType = con.InterconnectBayType //string
		ncon.ChangeState = con.ChangeState //string
		ncon.BayPowerState = con.BayPowerState //string

		out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, ncon)
	}

	out.EnclosureInfo.FanBayCount = int(in.EnclosureInfo.FanBayCount) //int32

	//out.EnclosureInfo.FanBays = in.EnclosureInfo.FanBays //[]*FanBayInfo
	for _, fan := range in.EnclosureInfo.FanBays {
		nfan := oneview.FanBayInfo{}
		nfan.BayNumber = int(fan.BayNumber) //int32
		nfan.DevicePresence = fan.DevicePresence //string
		nfan.DeviceRequired = fan.DeviceRequired //bool
		nfan.Status = fan.Status //string
		nfan.Model = fan.Model //string
		nfan.PartNumber = fan.PartNumber //string
		nfan.SparePartNumber = fan.SparePartNumber //string
		nfan.FanBayType = fan.FanBayType //string
		nfan.ChangeState = fan.ChangeState //string
		nfan.State = fan.State //string
		out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, nfan)
	}

	out.EnclosureInfo.PowerSupplyBayCount = int(in.EnclosureInfo.PowerSupplyBayCount) //int32

	//out.EnclosureInfo.PowerSupplyBays = in.EnclosureInfo.PowerSupplyBays //[]*PowerSupplyBayInfo
	for _, pwr := range in.EnclosureInfo.PowerSupplyBays {
		npwr := oneview.PowerSupplyBayInfo{}
		npwr.BayNumber = int(pwr.BayNumber) //int32
		npwr.DevicePresence = pwr.DevicePresence //string
		npwr.Status = pwr.Status //string
		npwr.Model = pwr.Model //string
		npwr.SerialNumber = pwr.SerialNumber //string
		npwr.PartNumber = pwr.PartNumber //string
		npwr.SparePartNumber = pwr.SparePartNumber //string
		npwr.PowerSupplyBayType = pwr.PowerSupplyBayType //string
		npwr.ChangeState = pwr.ChangeState //string
		out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, npwr)
	}

	out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged //bool
	out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware //bool

	//out.EnclosureInfo.ManagerBays = in.EnclosureInfo.ManagerBays //[]*ManagerBayInfo
	for _, man := range in.EnclosureInfo.ManagerBays {
		nman := oneview.ManagerBayInfo{}

		nman.BayNumber = int(man.BayNumber) //int32
		nman.ManagerType = man.ManagerType //string
		nman.BayPowerState = man.BayPowerState //string
		nman.FwVersion = man.FwVersion //string
		nman.DevicePresence = man.DevicePresence //string
		nman.Role = man.Role //string
		nman.IPAddress = man.IPAddress //string
		nman.ChangeState = man.ChangeState //string
		nman.FwBuildDate = man.FwBuildDate //string
		nman.DhcpIpv6Enable = man.DhcpIpv6Enable //bool
		nman.FqdnHostName = man.FqdnHostName //string
		nman.DhcpEnable = man.DhcpEnable //bool

		out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, nman)
	}

	out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState //string
	out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState //string
	out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType //string
	out.EnclosureInfo.Status = in.EnclosureInfo.Status //string
	out.EnclosureInfo.Name = in.EnclosureInfo.Name //string
	out.EnclosureInfo.State = in.EnclosureInfo.State //string
	out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP //string
	out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP //string
	out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag //string
	out.EnclosureInfo.RackName = in.EnclosureInfo.RackName //string
	out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID //string
	out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName //string
	if b, err := strconv.ParseBool(in.EnclosureInfo.VcmMode); err == nil {
		out.EnclosureInfo.VcmMode = b //bool
	}
	out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL //string
	out.EnclosureInfo.OaBays = int(in.EnclosureInfo.OaBays) //int32
	out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState //string

	out.BayInfo.Type = in.BayInfo.Type //string
	out.BayInfo.BayNumber = int(in.BayInfo.BayNumber) //int32
	out.BayInfo.DevicePresence = in.BayInfo.DevicePresence //string
	out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice //string
	out.BayInfo.URI = in.BayInfo.URI //string
	out.BayInfo.Category = in.BayInfo.Category //string
	out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile //bool
	out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile //bool
	out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType //string
	out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor //string
	out.BayInfo.BayPowerState = in.BayInfo.BayPowerState //string
	out.BayInfo.ChangeState = in.BayInfo.ChangeState //string
	out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile //bool
	out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile //bool
}

// MessageToClusterSprRecords конвертирует gRPC-сообщение pb.ClusterSprRecord (запись справочника кластеров) в доменную модель models.ClusterSprRecords. При in == nil или out == nil ничего не делает.
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

// ClusterSprRecordsToMessage конвертирует доменную модель models.ClusterSprRecords в gRPC-сообщение pb.ClusterSprRecord для передачи по сети. При in == nil или out == nil ничего не делает.
func ClusterSprRecordsToMessage(in *models.ClusterSprRecords, out *pb.ClusterSprRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Name = in.Name
	out.ClusterId = in.ClusterId
	out.TypeWork = int32(in.TypeWork)
	out.Stat = int32(in.Stat)
	out.Annotation = in.Annotation
	//out.Production = int32(in.Production) //признак продуктового кластера 0-не заполнено 1-продуктовый с бизнес приложениями 2-тестовые среды
	out.Id_User = in.Id_User
	out.Time_Mfy = timestamppb.New(in.Time_Mfy)
}

// ClusterRecordToMessage конвертирует доменную модель models.ClusterRecord в gRPC-сообщение pb.ClusterRecord. При in == nil или out == nil ничего не делает.
func ClusterRecordToMessage(in *models.ClusterRecord, out *pb.ClusterRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                                      //int64
	out.ClusterId = in.ClusterId                        //int64
	out.Date = timestamppb.New(in.Date)                  //time.Time
	out.Vcenter = in.Vcenter                            //string
	out.Datacenter = in.Datacenter                      //string
	out.Name = in.Name                                  //string
	out.CpuMhzAvg = in.CpuMhzAvg                        //int32
	out.AllNumCpuPkgs = int32(in.AllNumCpuPkgs)          //int16
	out.AllNumCpuCores = int32(in.AllNumCpuCores)        //int16
	out.AllNumCpuThreads = int32(in.AllNumCpuThreads)    //int16
	out.AllMemorySizeGB = in.AllMemorySizeGB             //float64
	out.AllCapacityGb = in.AllCapacityGb                //float64
	out.AllFreeSpaceGb = in.AllFreeSpaceGb              //float64
	out.AllPercentUsed = in.AllPercentUsed              //float64
	out.CpuUsageMaximum = in.Cpu_usage_maximum          //float64
	out.CpuUsageMaximumTime = timestamppb.New(in.Cpu_usage_maximum_time) //time.Time
	out.CpuUsageAverage = in.Cpu_usage_average          //float64
	out.CpuUsageMinimum = in.Cpu_usage_minimum          //float64
	out.CpuUsageMinimumTime = timestamppb.New(in.Cpu_usage_minimum_time) ////time.Time
	out.MemUsageMaximum = in.Mem_usage_maximum                         //float64
	out.MemUsageMaximumTime = timestamppb.New(in.Mem_usage_maximum_time) ////time.Time
	out.MemUsageAverage = in.Mem_usage_average                         //float64
	out.MemUsageMinimum = in.Mem_usage_minimum                         //float64
	out.MemUsageMinimumTime = timestamppb.New(in.Mem_usage_minimum_time) ////time.Time
	out.DiskUsageMaximum = in.Disk_usage_maximum                       //float64
	out.DiskUsageMaximumTime = timestamppb.New(in.Disk_usage_maximum_time) ////time.Time
	out.DiskUsageAverage = in.Disk_usage_average                       //float64
	out.DiskUsageMinimum = in.Disk_usage_minimum                       //float64
	out.DiskUsageMinimumTime = timestamppb.New(in.Disk_usage_minimum_time) ////time.Time
	out.DiskTotalReadLatencyMaximum = in.Disk_totalReadLatency_maximum //float64
	out.DiskTotalReadLatencyMaximumTime = timestamppb.New(in.Disk_totalReadLatency_maximum_time) ////time.Time
	out.DiskTotalReadLatencyAverage = in.Disk_totalReadLatency_average //float64
	out.DiskTotalReadLatencyMinimum = in.Disk_totalReadLatency_minimum //float64
	out.DiskTotalReadLatencyMinimumTime = timestamppb.New(in.Disk_totalReadLatency_minimum_time) ////time.Time
	out.DiskTotalWriteLatencyMaximum = in.Disk_totalWriteLatency_maximum //float64
	out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(in.Disk_totalWriteLatency_maximum_time) ////time.Time
	out.DiskTotalWriteLatencyAverage = in.Disk_totalWriteLatency_average //float64
	out.DiskTotalWriteLatencyMinimum = in.Disk_totalWriteLatency_minimum //float64
	out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(in.Disk_totalWriteLatency_minimum_time) ////time.Time
	out.DiskWriteMaximum = in.Disk_write_maximum                       //float64
	out.DiskWriteMaximumTime = timestamppb.New(in.Disk_write_maximum_time) ////time.Time
	out.DiskWriteAverage = in.Disk_write_average                       //float64
	out.DiskWriteMinimum = in.Disk_write_minimum                       //float64
	out.DiskWriteMinimumTime = timestamppb.New(in.Disk_write_minimum_time) ////time.Time
	out.DiskReadMaximum = in.Disk_read_maximum                         //float64
	out.DiskReadMaximumTime = timestamppb.New(in.Disk_read_maximum_time) ////time.Time
	out.DiskReadAverage = in.Disk_read_average                         //float64
	out.DiskReadMinimum = in.Disk_read_minimum                         //float64
	out.DiskReadMinimumTime = timestamppb.New(in.Disk_read_minimum_time) ////time.Time
	out.NetUsageMaximum = in.Net_usage_maximum                         //float64
	out.NetUsageMaximumTime = timestamppb.New(in.Net_usage_maximum_time) ////time.Time
	out.NetUsageAverage = in.Net_usage_average                         //float64
	out.NetUsageMinimum = in.Net_usage_minimum                         //float64
	out.NetUsageMinimumTime = timestamppb.New(in.Net_usage_minimum_time) ////time.Time
	out.CpuBilling = in.CpuBilling                                     //float64
	out.DiskBilling = in.DiskBilling                                   //float64
	out.MemBilling = in.MemBilling                                     //float64
	out.NetBilling = in.NetBilling                                     //float64
	out.HostCount = in.HostCount                                       //int64
	out.VMCount = in.VMCount                                           //int64
	out.StorageCount = in.StorageCount                                 //int64
}

// MessageToClusterRecord конвертирует gRPC-сообщение pb.ClusterRecord в доменную модель models.ClusterRecord. При in == nil или out == nil ничего не делает.
func MessageToClusterRecord(in *pb.ClusterRecord, out *models.ClusterRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                                                         //int64
	out.ClusterId = in.ClusterId                                           //int64
	out.Date = in.Date.AsTime()                                            //time.Time
	out.Vcenter = in.Vcenter                                               //string
	out.Datacenter = in.Datacenter                                         //string
	out.Name = in.Name                                                     //string
	out.CpuMhzAvg = in.CpuMhzAvg                                           //int32
	out.AllNumCpuPkgs = int16(in.AllNumCpuPkgs)                            //int16
	out.AllNumCpuCores = int16(in.AllNumCpuCores)                          //int16
	out.AllNumCpuThreads = int16(in.AllNumCpuThreads)                      //int16
	out.AllMemorySizeGB = in.AllMemorySizeGB                               //float64
	out.AllCapacityGb = in.AllCapacityGb                                   //float64
	out.AllFreeSpaceGb = in.AllFreeSpaceGb                                 //float64
	out.AllPercentUsed = in.AllPercentUsed                                 //float64
	out.Cpu_usage_maximum = in.CpuUsageMaximum                             //float64
	out.Cpu_usage_maximum_time = in.CpuUsageMaximumTime.AsTime()           //time.Time
	out.Cpu_usage_average = in.CpuUsageAverage                             //float64
	out.Cpu_usage_minimum = in.CpuUsageMinimum                             //float64
	out.Cpu_usage_minimum_time = in.CpuUsageMinimumTime.AsTime()           //time.Time
	out.Mem_usage_maximum = in.MemUsageMaximum                             //float64
	out.Mem_usage_maximum_time = in.MemUsageMaximumTime.AsTime()           //time.Time
	out.Mem_usage_average = in.MemUsageAverage                             //float64
	out.Mem_usage_minimum = in.MemUsageMinimum                             //float64
	out.Mem_usage_minimum_time = in.MemUsageMinimumTime.AsTime()           //time.Time
	out.Disk_usage_maximum = in.DiskUsageMaximum                           //float64
	out.Disk_usage_maximum_time = in.DiskUsageMaximumTime.AsTime()         //time.Time
	out.Disk_usage_average = in.DiskUsageAverage                           //float64
	out.Disk_usage_minimum = in.DiskUsageMinimum                           //float64
	out.Disk_usage_minimum_time = in.DiskUsageMinimumTime.AsTime()         //time.Time
	out.Disk_totalReadLatency_maximum = in.DiskTotalReadLatencyMaximum     //float64
	out.Disk_totalReadLatency_maximum_time = in.DiskTotalReadLatencyMaximumTime.AsTime() //time.Time
	out.Disk_totalReadLatency_average = in.DiskTotalReadLatencyAverage       //float64
	out.Disk_totalReadLatency_minimum = in.DiskTotalReadLatencyMinimum       //float64
	out.Disk_totalReadLatency_minimum_time = in.DiskTotalReadLatencyMinimumTime.AsTime() //time.Time
	out.Disk_totalWriteLatency_maximum = in.DiskTotalWriteLatencyMaximum     //float64
	out.Disk_totalWriteLatency_maximum_time = in.DiskTotalWriteLatencyMaximumTime.AsTime() //time.Time
	out.Disk_totalWriteLatency_average = in.DiskTotalWriteLatencyAverage     //float64
	out.Disk_totalWriteLatency_minimum = in.DiskTotalWriteLatencyMinimum     //float64
	out.Disk_totalWriteLatency_minimum_time = in.DiskTotalWriteLatencyMinimumTime.AsTime() //time.Time
	out.Disk_write_maximum = in.DiskWriteMaximum                           //float64
	out.Disk_write_maximum_time = in.DiskWriteMaximumTime.AsTime()         //time.Time
	out.Disk_write_average = in.DiskWriteAverage                           //float64
	out.Disk_write_minimum = in.DiskWriteMinimum                           //float64
	out.Disk_write_minimum_time = in.DiskWriteMinimumTime.AsTime()         //time.Time
	out.Disk_read_maximum = in.DiskReadMaximum                             //float64
	out.Disk_read_maximum_time = in.DiskReadMaximumTime.AsTime()           //time.Time
	out.Disk_read_average = in.DiskReadAverage                             //float64
	out.Disk_read_minimum = in.DiskReadMinimum                             //float64
	out.Disk_read_minimum_time = in.DiskReadMinimumTime.AsTime()           //time.Time
	out.Net_usage_maximum = in.NetUsageMaximum                             //float64
	out.Net_usage_maximum_time = in.NetUsageMaximumTime.AsTime()           //time.Time
	out.Net_usage_average = in.NetUsageAverage                             //float64
	out.Net_usage_minimum = in.NetUsageMinimum                             //float64
	out.Net_usage_minimum_time = in.NetUsageMinimumTime.AsTime()           //time.Time
	out.CpuBilling = in.CpuBilling                                         //float64
	out.DiskBilling = in.DiskBilling                                       //float64
	out.MemBilling = in.MemBilling                                         //float64
	out.NetBilling = in.NetBilling                                         //float64
	out.HostCount = in.HostCount                                           //int64
	out.VMCount = in.VMCount                                               //int64
	out.StorageCount = in.StorageCount                                     //int64
}

// HostRecordToMessage конвертирует доменную модель models.HostRecord в gRPC-сообщение pb.HostRecord. При in == nil или out == nil ничего не делает.
func HostRecordToMessage(in *models.HostRecord, out *pb.HostRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                                    //int64
	out.ServerId = in.ServerId                                        //int64
	out.ClusterId = in.ClusterId                                      //int64
	out.Date = timestamppb.New(in.Date)                                //time.Time
	out.Vcenter = in.Vcenter                                          //string
	out.Datacenter = in.Datacenter                                    //string
	out.Cluster = in.Cluster                                          //string
	out.Name = in.Name                                                //string
	out.Vendor = in.Vendor                                            //string
	out.Model = in.Model                                              //string
	out.CpuModel = in.CpuModel                                        //string
	out.CpuMhz = in.CpuMhz                                            //int32
	out.NumCpuPkgs = int32(in.NumCpuPkgs)                             //int16
	out.NumCpuCores = int32(in.NumCpuCores)                           //int16
	out.NumCpuThreads = int32(in.NumCpuThreads)                       //int16
	out.NumNics = in.NumNics                                          //int32
	out.NumHBAs = in.NumHBAs                                          //int32
	out.MemorySizeGB = in.MemorySizeGB                                //float64
	out.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity    //float64
	out.CpuUsageMaximum = in.Cpu_usage_maximum                        //float64
	out.CpuUsageMaximumTime = timestamppb.New(in.Cpu_usage_maximum_time) //time.Time
	out.CpuUsageAverage = in.Cpu_usage_average                               //float64
	out.CpuUsageMinimum = in.Cpu_usage_minimum                               //float64
	out.CpuUsageMinimumTime = timestamppb.New(in.Cpu_usage_minimum_time)     //time.Time
	out.MemUsageMaximum = in.Mem_usage_maximum                               //float64
	out.MemUsageMaximumTime = timestamppb.New(in.Mem_usage_maximum_time)     //time.Time
	out.MemUsageAverage = in.Mem_usage_average                               //float64
	out.MemUsageMinimum = in.Mem_usage_minimum                               //float64
	out.MemUsageMinimumTime = timestamppb.New(in.Mem_usage_minimum_time)     //time.Time
	out.DiskUsageMaximum = in.Disk_usage_maximum                             //float64
	out.DiskUsageMaximumTime = timestamppb.New(in.Disk_usage_maximum_time)   //time.Time
	out.DiskUsageAverage = in.Disk_usage_average                             //float64
	out.DiskUsageMinimum = in.Disk_usage_minimum                             //float64
	out.DiskUsageMinimumTime = timestamppb.New(in.Disk_usage_minimum_time)   //time.Time
	out.DiskTotalReadLatencyMaximum = in.Disk_totalReadLatency_maximum       //float64
	out.DiskTotalReadLatencyMaximumTime = timestamppb.New(in.Disk_totalReadLatency_maximum_time) //time.Time
	out.DiskTotalReadLatencyAverage = in.Disk_totalReadLatency_average       //float64
	out.DiskTotalReadLatencyMinimum = in.Disk_totalReadLatency_minimum       //float64
	out.DiskTotalReadLatencyMinimumTime = timestamppb.New(in.Disk_totalReadLatency_minimum_time) //time.Time
	out.DiskTotalWriteLatencyMaximum = in.Disk_totalWriteLatency_maximum     //float64
	out.DiskTotalWriteLatencyMaximumTime = timestamppb.New(in.Disk_totalWriteLatency_maximum_time) //time.Time
	out.DiskTotalWriteLatencyAverage = in.Disk_totalWriteLatency_average     //float64
	out.DiskTotalWriteLatencyMinimum = in.Disk_totalWriteLatency_minimum     //float64
	out.DiskTotalWriteLatencyMinimumTime = timestamppb.New(in.Disk_totalWriteLatency_minimum_time) //time.Time
	out.DiskWriteMaximum = in.Disk_write_maximum                             //float64
	out.DiskWriteMaximumTime = timestamppb.New(in.Disk_write_maximum_time)   //time.Time
	out.DiskWriteAverage = in.Disk_write_average                             //float64
	out.DiskWriteMinimum = in.Disk_write_minimum                             //float64
	out.DiskWriteMinimumTime = timestamppb.New(in.Disk_write_minimum_time)   //time.Time
	out.DiskReadMaximum = in.Disk_read_maximum                               //float64
	out.DiskReadMaximumTime = timestamppb.New(in.Disk_read_maximum_time)     //time.Time
	out.DiskReadAverage = in.Disk_read_average                               //float64
	out.DiskReadMinimum = in.Disk_read_minimum                               //float64
	out.DiskReadMinimumTime = timestamppb.New(in.Disk_read_minimum_time)     //time.Time
	out.NetUsageMaximum = in.Net_usage_maximum                               //float64
	out.NetUsageMaximumTime = timestamppb.New(in.Net_usage_maximum_time)     //time.Time
	out.NetUsageAverage = in.Net_usage_average                               //float64
	out.NetUsageMinimum = in.Net_usage_minimum                               //float64
	out.NetUsageMinimumTime = timestamppb.New(in.Net_usage_minimum_time)     //time.Time
	out.CpuBilling = in.CpuBilling                                           //float64
	out.MemBilling = in.MemBilling                                           //float64
	out.DiskBilling = in.DiskBilling                                         //float64
	out.NetBilling = in.NetBilling                                           //float64
}

// MessageToHostRecord конвертирует gRPC-сообщение pb.HostRecord в доменную модель models.HostRecord. При in == nil или out == nil ничего не делает.
func MessageToHostRecord(in *pb.HostRecord, out *models.HostRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                         //int64
	out.ServerId = in.ServerId             //int64
	out.ClusterId = in.ClusterId           //int64
	out.Date = in.Date.AsTime()            //time.Time
	out.Vcenter = in.Vcenter               //string
	out.Datacenter = in.Datacenter         //string
	out.Cluster = in.Cluster               //string
	out.Name = in.Name                     //string
	out.Vendor = in.Vendor                 //string
	out.Model = in.Model
	out.CpuModel = in.CpuModel             //string
	out.CpuMhz = in.CpuMhz                 //int32
	out.NumCpuPkgs = int16(in.NumCpuPkgs)  //int16
	out.NumCpuCores = int16(in.NumCpuCores) //int16
	out.NumCpuThreads = int16(in.NumCpuThreads) //int16
	out.NumNics = in.NumNics               //int32
	out.NumHBAs = in.NumHBAs               //int32
	out.MemorySizeGB = in.MemorySizeGB     //float64
	out.HostMaxVirtualDiskCapacity = in.HostMaxVirtualDiskCapacity //float64
	out.Cpu_usage_maximum = in.CpuUsageMaximum //float64
	out.Cpu_usage_maximum_time = in.CpuUsageMaximumTime.AsTime() //time.Time
	out.Cpu_usage_average = in.CpuUsageAverage //float64
	out.Cpu_usage_minimum = in.CpuUsageMinimum //float64
	out.Cpu_usage_minimum_time = in.CpuUsageMinimumTime.AsTime() //time.Time
	out.Mem_usage_maximum = in.MemUsageMaximum //float64
	out.Mem_usage_maximum_time = in.MemUsageMaximumTime.AsTime() //time.Time
	out.Mem_usage_average = in.MemUsageAverage //float64
	out.Mem_usage_minimum = in.MemUsageMinimum //float64
	out.Mem_usage_minimum_time = in.MemUsageMinimumTime.AsTime() //time.Time
	out.Disk_usage_maximum = in.DiskUsageMaximum //float64
	out.Disk_usage_maximum_time = in.DiskUsageMaximumTime.AsTime() //time.Time
	out.Disk_usage_average = in.DiskUsageAverage //float64
	out.Disk_usage_minimum = in.DiskUsageMinimum //float64
	out.Disk_usage_minimum_time = in.DiskUsageMinimumTime.AsTime() //time.Time
	out.Disk_totalReadLatency_maximum = in.DiskTotalReadLatencyMaximum //float64
	out.Disk_totalReadLatency_maximum_time = in.DiskTotalReadLatencyMaximumTime.AsTime() //time.Time
	out.Disk_totalReadLatency_average = in.DiskTotalReadLatencyAverage //float64
	out.Disk_totalReadLatency_minimum = in.DiskTotalReadLatencyMinimum //float64
	out.Disk_totalReadLatency_minimum_time = in.DiskTotalReadLatencyMinimumTime.AsTime() //time.Time
	out.Disk_totalWriteLatency_maximum = in.DiskTotalWriteLatencyMaximum //float64
	out.Disk_totalWriteLatency_maximum_time = in.DiskTotalWriteLatencyMaximumTime.AsTime() //time.Time
	out.Disk_totalWriteLatency_average = in.DiskTotalWriteLatencyAverage //float64
	out.Disk_totalWriteLatency_minimum = in.DiskTotalWriteLatencyMinimum //float64
	out.Disk_totalWriteLatency_minimum_time = in.DiskTotalWriteLatencyMinimumTime.AsTime() //time.Time
	out.Disk_write_maximum = in.DiskWriteMaximum //float64
	out.Disk_write_maximum_time = in.DiskWriteMaximumTime.AsTime() //time.Time
	out.Disk_write_average = in.DiskWriteAverage //float64
	out.Disk_write_minimum = in.DiskWriteMinimum //float64
	out.Disk_write_minimum_time = in.DiskWriteMinimumTime.AsTime() //time.Time
	out.Disk_read_maximum = in.DiskReadMaximum //float64
	out.Disk_read_maximum_time = in.DiskReadMaximumTime.AsTime() //time.Time
	out.Disk_read_average = in.DiskReadAverage //float64
	out.Disk_read_minimum = in.DiskReadMinimum //float64
	out.Disk_read_minimum_time = in.DiskReadMinimumTime.AsTime() //time.Time
	out.Net_usage_maximum = in.NetUsageMaximum //float64
	out.Net_usage_maximum_time = in.NetUsageMaximumTime.AsTime() //time.Time
	out.Net_usage_average = in.NetUsageAverage //float64
	out.Net_usage_minimum = in.NetUsageMinimum //float64
	out.Net_usage_minimum_time = in.NetUsageMinimumTime.AsTime() //time.Time
	out.CpuBilling = in.CpuBilling //float64
	out.MemBilling = in.MemBilling //float64
	out.DiskBilling = in.DiskBilling //float64
	out.NetBilling = in.NetBilling //float64
}

// VMRecordToMessage конвертирует доменную модель models.VMRecord в gRPC-сообщение pb.VMRecord. При in == nil или out == nil ничего не делает.
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

// MessageToVMRecord конвертирует gRPC-сообщение pb.VMRecord в доменную модель models.VMRecord. При in == nil или out == nil ничего не делает.
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

// ClusterPerfToMessage конвертирует доменную модель models.ClusterPerf (метрика кластера для БД) в gRPC-сообщение pb.ClusterPerf. При in == nil или out == nil ничего не делает.
func ClusterPerfToMessage(in *models.ClusterPerf, out *pb.ClusterPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                                      //int64
	out.ClusterId = in.ClusterId                        //int64
	out.Start = timestamppb.New(in.Start)               //time.Time
	out.CpuUsage = in.Cpu_usage                         //float64
	out.MemUsage = in.Mem_usage                         //float64
	out.DiskUsage = in.Disk_usage                       //float64
	out.DiskTotalReadLatency = in.Disk_totalReadLatency //float64
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency //float64
	out.DiskWrite = in.Disk_write                       //float64
	out.DiskRead = in.Disk_read                         //float64
	out.NetUsage = in.Net_usage                         //float64
	out.NetReceived = in.Net_received                   //float64
	out.NetTransmitted = in.Net_transmitted             //float64
}

// MessageToClusterPerf конвертирует gRPC-сообщение pb.ClusterPerf в доменную модель models.ClusterPerf. При in == nil или out == nil ничего не делает.
func MessageToClusterPerf(in *pb.ClusterPerf, out *models.ClusterPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                              //int64
	out.ClusterId = in.ClusterId                //int64
	out.Start = in.Start.AsTime()               //time.Time
	out.Cpu_usage = in.CpuUsage                 //float64
	out.Mem_usage = in.MemUsage                 //float64
	out.Disk_usage = in.DiskUsage               //float64
	out.Disk_totalReadLatency = in.DiskTotalReadLatency //float64
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency //float64
	out.Disk_write = in.DiskWrite               //float64
	out.Disk_read = in.DiskRead                 //float64
	out.Net_usage = in.NetUsage                 //float64
	out.Net_received = in.NetReceived           //float64
	out.Net_transmitted = in.NetTransmitted     //float64

}

// HostPerfToMessage конвертирует доменную модель models.HostPerf в gRPC-сообщение pb.HostPerf. При in == nil или out == nil ничего не делает.
func HostPerfToMessage(in *models.HostPerf, out *pb.HostPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                              //int64
	out.HostId = in.HostId                      //int64
	out.Start = timestamppb.New(in.Start)       //time.Time
	out.CpuUsage = in.Cpu_usage                 //float64
	out.MemUsage = in.Mem_usage                 //float64
	out.DiskUsage = in.Disk_usage               //float64
	out.DiskTotalReadLatency = in.Disk_totalReadLatency //float64
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency //float64
	out.DiskWrite = in.Disk_write               //float64
	out.DiskRead = in.Disk_read                 //float64
	out.NetUsage = in.Net_usage                 //float64
	out.NetReceived = in.Net_received           //float64
	out.NetTransmitted = in.Net_transmitted     //float64

}

// MessageToHostPerf конвертирует gRPC-сообщение pb.HostPerf в доменную модель models.HostPerf. При in == nil или out == nil ничего не делает.
func MessageToHostPerf(in *pb.HostPerf, out *models.HostPerf) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                               //int64
	out.HostId = in.HostId                       //int64
	out.Start = in.Start.AsTime()                //time.Time
	out.Cpu_usage = in.CpuUsage                  //float64
	out.Mem_usage = in.MemUsage                  //float64
	out.Disk_usage = in.DiskUsage                //float64
	out.Disk_totalReadLatency = in.DiskTotalReadLatency //float64
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency //float64
	out.Disk_write = in.DiskWrite                //float64
	out.Disk_read = in.DiskRead                  //float64
	out.Net_usage = in.NetUsage                  //float64
	out.Net_received = in.NetReceived            //float64
	out.Net_transmitted = in.NetTransmitted      //float64

}

// VMPerfRecordToMessage конвертирует доменную модель models.VMPerfRecord в gRPC-сообщение pb.VMPerfRecord. При in == nil или out == nil ничего не делает.
func VMPerfRecordToMessage(in *models.VMPerfRecord, out *pb.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                               //int64
	out.VMID = in.VMId                           //int64
	out.Start = timestamppb.New(in.Start)        //time.Time
	out.CpuUsage = in.Cpu_usage                  //float64
	out.MemUsage = in.Mem_usage                  //float64
	out.DiskUsage = in.Disk_usage                //float64
	out.DiskTotalReadLatency = in.Disk_totalReadLatency //float64
	out.DiskTotalWriteLatency = in.Disk_totalWriteLatency //float64
	out.DiskWrite = in.Disk_write                //float64
	out.DiskRead = in.Disk_read                  //float64
	out.NetUsage = in.Net_usage                  //float64
	out.NetReceived = in.Net_received            //float64
	out.NetTransmitted = in.Net_transmitted      //float64

}

// MessageToVMPerfRecord конвертирует gRPC-сообщение pb.VMPerfRecord в доменную модель models.VMPerfRecord. При in == nil или out == nil ничего не делает.
func MessageToVMPerfRecord(in *pb.VMPerfRecord, out *models.VMPerfRecord) {
	if in == nil || out == nil {
		return
	}

	out.Id = in.Id                               //int64
	out.VMId = in.VMID                           //int64
	out.Start = in.Start.AsTime()             //time.Time
	out.Cpu_usage = in.CpuUsage               //float64
	out.Mem_usage = in.MemUsage                //float64
	out.Disk_usage = in.DiskUsage             //float64
	out.Disk_totalReadLatency = in.DiskTotalReadLatency //float64
	out.Disk_totalWriteLatency = in.DiskTotalWriteLatency //float64
	out.Disk_write = in.DiskWrite             //float64
	out.Disk_read = in.DiskRead               //float64
	out.Net_usage = in.NetUsage               //float64
	out.Net_received = in.NetReceived         //float64
	out.Net_transmitted = in.NetTransmitted   //float64

}
