package grpcclient

import (
	"loadmon/models"
	pb "loadmon/protocol"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func MessageToServerSprRecod(in *pb.ServerSprRecords, out *models.ServerSprRecords) {
	if in == nil || out == nil {
		return
	}
	out.Id = in.Id                                              //int64
	out.OldId = in.OldId                                        //int64      ссылка на историческую запись после переименования. 0 если такой нет. Может быть цепочка записей.
	out.Type = int(in.Type)                                     //int        0 - физический сервер 1 - виртуальный 2-host vmware
	out.TypeMicroservice = int(in.TypeMicroservice)             //int        0 не указано принадлежность к микросервисам, роль в этом пространстве
	out.RoleType = int(in.RoleType)                             //int        0-не указано 1-APP 2-DB
	out.TypeWork = int(in.TypeWork)                             //int        0 - не указано 1 - продуктовая среда 2- тестовая
	out.CriticalLevel = int(in.CriticalLevel)                   //int        контур 0 не указано 1-Vital 2-Critical 3-Secondary 4-Non Priority
	out.VMSprId = in.VMSprId                                    //int64      ссылка на справочник VM если Type = 1 иначе null
	out.Stat = int(in.Stat)                                     //int        0 историческая запись 1 используется 2 помечено для удаления 3-архивная запись после переименования 911 - новая запись,неполные
	out.PowerState = int(in.PowerState)                         //int        0 не указано 1 включен 2 выключен (последнее состояние) при отображении переопределять текущим состоянием если есть
	out.Active = int(in.Active)                                 //int        1 активен 2 выведен из эксплуатации
	out.TimeActiveState = in.TimeActiveState.AsTime()           //time.Time  время изменения статуса активности введения в эксплуатацию
	out.ActivatorUser = in.ActivatorUser                        //int64      автор записи
	out.TimeDeActiveState = in.TimeDeActiveState.AsTime()       //time.Time время выведения из эксплуатации
	out.DeActivatorUser = in.DeActivatorUser                    //int64      автор записи
	out.TimeCreate = in.TimeCreate.AsTime()                     //time.Time  время создания записи
	out.CreatorUser = in.CreatorUser                            //int64      автор записи
	out.BusinesPrivilage = in.BusinesPrivilage                  //int64      ограничение по редактированию если указано то только пользователь из этой группы может редактировать данные
	out.Name = in.Name                                          //string
	out.NameFQDN = in.NameFQDN                                  //string     полное имя сервера
	out.NameInfo = in.NameInfo                                  //string     альтернативное имя сервера
	out.IPAddress = in.IPAddress                                //string     список IP адресов сервера при наличие информации подгружается из конфигурации VM или вводится вручную
	out.MacAddress = in.MacAddress                              //
	out.IPSharedVIP = in.IPSharedVIP                            //string     разделяемый IP вводится вручную
	out.Network = in.Network                                    //string     сеть vlan144
	out.NetworkName = in.NetworkName                            //string     наименование сети сеть DMZ Prom
	out.CPUCount = in.CPUCount                                  //float64    Количество физических процессоров
	out.CPULogical = in.CPULogical                              //float64    количество логических CPU
	out.MemSize = in.MemSize                                    //float64  размер оперативной памяти сервера
	out.DiskSize = in.DiskSize                                  //float64  суммарный размер всех дисков
	out.Service = in.Service                                    //string   сервис в котором задействован сервер
	out.ServiceId = in.ServiceId                                //int64    ссылка на базу сервисов
	out.ITSystemKey = in.ITSystemKey                            //string   ссылка на ИТ систему из jira https://jira.bcs.ru/rest/assets/latest/aql/objects?qlQuery=objectTypeID=21&includeAttributes=true&p
	out.OS = in.OS                                              //string   операционная система
	out.Component = in.Component                                //string   компонента Kafka, service BD ....
	out.Owner = in.Owner                                        //string   владелец сервера (отвественный за сервер)
	out.BusinessOwner = in.BusinessOwner                        //string   владелец от бизнеса
	out.ITOwner = in.ITOwner                                    //string   владелец со стороны ИТ
	out.ProductOwner = in.ProductOwner                          //string   Владелец продукта
	out.Support = in.Support                                    //string   ответственные за поддержку
	out.SupportTeam = in.SupportTeam                            //string   Подразделение поддержки
	out.SystemType = in.SystemType                              //string   тип системы техническая ...
	out.LicensingType = in.LicensingType                        //string   тип лицензирования
	out.SolutionType = in.SolutionType                          //string   вариант решения (коробочной)
	out.Users = in.Users                                        //string   пользователи системы
	out.Allocation = in.Allocation                              //string   аллокация (подразделение)
	out.Anotation = in.Anotation                                //string   описане сервера
	out.Role = in.Role                                          //string   роль сервера (сервер баз данных, сервер приложений, хост)
	out.Details = in.Details                                    //string   детали по серверу в свободном формате (примечание)
	out.SZHistory = in.SZhistory                                //string   номера сз через пробел в хронологическом порядке
	out.Warning = in.Warning                                    //string
	out.Level = in.Level                                        //float64
	out.Monitoring = int(in.Monitoring)                         //int      0- не указано 1-включен мониторинг 2-отключен мониторинг
	out.Backuped = int(in.Backuped)                             //int      0 - не указано 1-включено резервное копирование 2-отключено резервное копирование
	out.TimeActiveMonitoring = in.TimeActiveMonitoring.AsTime() //time.Time время постановки на мониторинг
	out.MonitoringSZ = in.MonitoringSZ                          //string   основание постановки на мониторинг
	out.Id_User = in.Id_User                                    //int64
	out.Time_Mfy = in.Time_Mfy.AsTime()                         //time.Time
	out.Privilage = int(in.Privilage)                           //int      уровень привилегий (в формах должны отображаться сервисы с уровнем не выше чем текущие привилегии пользователя)
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
	out.DatacenterName = in.DatacenterName
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
	out.DatacenterName = in.DatacenterName
}

// ClusterRecordToMessage конвертирует доменную модель models.ClusterRecord в gRPC-сообщение pb.ClusterRecord. При in == nil или out == nil ничего не делает.
