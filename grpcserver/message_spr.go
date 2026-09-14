package grpcserver

import (
	"loadmon/models"

	pb "loadmon/protocol"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func ServerSprRecodToMessage(in *models.ServerSprRecords, out *pb.ServerSprRecords) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id            //int64
	out.OldId = in.OldId      //int64
	out.Type = int32(in.Type) //int
	out.TypeMicroservice = int32(in.TypeMicroservice)
	out.RoleType = int32(in.RoleType)
	out.TypeWork = int32(in.TypeWork)
	out.CriticalLevel = int32(in.CriticalLevel)
	out.VMSprId = in.VMSprId //int64
	out.Stat = int32(in.Stat)
	out.PowerState = int32(in.PowerState)
	out.Active = int32(in.Active)
	out.TimeActiveState = timestamppb.New(in.TimeActiveState)
	out.ActivatorUser = in.ActivatorUser
	out.TimeDeActiveState = timestamppb.New(in.TimeDeActiveState)
	out.DeActivatorUser = in.DeActivatorUser
	out.TimeCreate = timestamppb.New(in.TimeCreate)
	out.CreatorUser = in.CreatorUser
	out.BusinesPrivilage = in.BusinesPrivilage
	out.Name = in.Name
	out.NameFQDN = in.NameFQDN
	out.NameInfo = in.NameInfo
	out.IPAddress = in.IPAddress
	out.IPSharedVIP = in.IPSharedVIP
	out.Network = in.Network
	out.NetworkName = in.NetworkName
	out.CPUCount = in.CPUCount
	out.CPULogical = in.CPULogical
	out.MemSize = in.MemSize
	out.DiskSize = in.DiskSize
	out.Service = in.Service
	out.ServiceId = in.ServiceId
	out.ITSystemKey = in.ITSystemKey
	out.OS = in.OS
	out.Component = in.Component
	out.Owner = in.Owner                                                //string    владелец сервера (ответственный за сервер)
	out.BusinessOwner = in.BusinessOwner                                //string    владелец от бизнеса
	out.ITOwner = in.ITOwner                                            //string    владелец со стороны ИТ
	out.ProductOwner = in.ProductOwner                                  //string    Владелец продукта
	out.Support = in.Support                                            //string    ответственные за поддержку
	out.SupportTeam = in.SupportTeam                                    //string    Подразделение поддержки
	out.SystemType = in.SystemType                                      //string    тип системы техническая ...
	out.LicensingType = in.LicensingType                                //string    тип лицензирования
	out.SolutionType = in.SolutionType                                  //string    вариант решения (коробочной)
	out.Users = in.Users                                                //string    пользователи системы
	out.Allocation = in.Allocation                                      //string    аллокация (подразделение)
	out.Anotation = in.Anotation                                        //string    описане сервера
	out.Role = in.Role                                                  //string    роль сервера (сервер баз данных, сервер приложений, хост)
	out.Details = in.Details                                            //string    детали по серверу в свободном формате (примечание)
	out.SZhistory = in.SZHistory                                        //string    номера сз через пробел в хронологическом порядке
	out.Warning = in.Warning                                            //string
	out.Level = in.Level                                                //float64
	out.Monitoring = int32(in.Monitoring)                               //int       0- не указано 1-включен мониторинг 2-отключен мониторинг
	out.Backuped = int32(in.Backuped)                                   //int       0 - не указано 1-включено резервное копирование 2-отключено резервное копирование
	out.TimeActiveMonitoring = timestamppb.New(in.TimeActiveMonitoring) //time.Time время постановки на мониторинг
	out.MonitoringSZ = in.MonitoringSZ                                  //string    основание постановки на мониторинг
	out.Id_User = in.Id_User                                            //int64
	out.Time_Mfy = timestamppb.New(in.Time_Mfy)                         //time.Time
	out.Privilage = int32(in.Privilage)                                 //int       уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привелегии пользователя)
	out.ClusterId = in.ClusterId
	out.Cluster = in.Cluster
	out.Datacenter = in.Datacenter
	out.Vcenter = in.Vcenter
}

// VMSprRecodToMessage конвертирует запись справочника ВМ models.VMSprRecord в gRPC-сообщение pb.VMSprRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func VMSprRecodToMessage(in *models.VMSprRecord, out *pb.VMSprRecord) {
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
	out.Vm_HostName = in.Vm_HostName // TODO: screenshot shows Vm_Hostname
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
	out.CpuUsageMaximum = in.Cpu_usage_maximum
	out.MemUsageMaximum = in.Mem_usage_maximum
	out.DiskUsageMaximum = in.Disk_usage_maximum
	out.NetUsageMaximum = in.Net_usage_maximum
}

// ClusterSprRecordsToMessage конвертирует запись справочника кластеров models.ClusterSprRecords в gRPC-сообщение pb.ClusterSprRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
func ClusterSprRecordsToMessage(in *models.ClusterSprRecords, out *pb.ClusterSprRecord) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id
	out.Vcenter = in.Vcenter
	out.Datacenter = in.Datacenter
	out.Name = in.Name
	out.ClusterId = in.ClusterId
	out.Annotation = in.Annotation
	//out.Production = int32(in.Production) //признак продуктового кластера 0-не заполнено 1-продуктовый с бизнес приложениями 2-тестовые среды
	out.TypeWork = int32(in.TypeWork)
	out.Stat = int32(in.Stat)
	out.Id_User = in.Id_User
	out.Time_Mfy = timestamppb.New(in.Time_Mfy)
}

// MessageToClusterSprRecords выполняет обратную конвертацию pb.ClusterSprRecord в models.ClusterSprRecords.
// Принимает in и out; при in == nil или out == nil ничего не делает.
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

// ClusterRecordToMessage конвертирует models.ClusterRecord в gRPC-сообщение pb.ClusterRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
