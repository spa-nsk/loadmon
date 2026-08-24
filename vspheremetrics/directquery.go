package vspheremetrics

import (
	"fmt"
	"time"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"golang.org/x/net/context"
)

// directQueryTimeout - таймаут на подключение к vCenter и выполнение запроса
// в функциях GetVMDirect/GetHostDirect ниже (аналог dialTimeout в
// grpcclient, но для прямого govmomi-соединения, а не gRPC).
const directQueryTimeout = 30 * time.Second

// GetVMDirect ищет виртуальную машину vmName в датацентре datacenter
// vCenter vcenterHost НАПРЯМУЮ в VMware - устанавливает отдельное
// govmomi-соединение (VCenter.Connect(), тот же приём, что и в
// loadinventory.go/loadvmmetrics.go) и не читает закэшированные в памяти
// данные (VMInfrastructure.VMMapList и связанные карты глобальной
// инфраструктуры, см. infrastructure_global.go) - в отличие от остальных
// Get-функций пакета, которые отдают уже собранный ранее (возможно,
// устаревший на несколько минут - до следующего опроса по тикеру
// vc.Wait, см. vcenter_timer.go) снимок инфраструктуры.
//
// Используется там, где принципиально важно проверить самое текущее
// состояние объекта в vSphere прямо сейчас, а не то, что было в момент
// последнего опроса - например, cmdb.CheckServerSprRecordsExistence
// (проверка, не была ли ВМ удалена/переименована в VMware, прежде чем
// пометить соответствующую запись справочника как выведенную из
// эксплуатации).
//
// Параметр cluster в поиск не участвует (в vSphere ВМ ищется по имени в
// пределах датацентра, а не кластера) - используется только для заполнения
// одноимённого поля результата VMPoint.Cluster, как было передано вызывающей
// стороной. Единственная нужная для поиска связка - vcenterHost/datacenter/
// vmName; для целей проверки существования (переименована/удалена ли ВМ)
// этого достаточно - при желании уточнить принадлежность именно указанному
// кластеру вызывающая сторона может дополнительно сверить поле
// VMPoint.HostName/принадлежность к ресурс-пулу самостоятельно.
//
// Возвращает ошибку, если vcenterHost неизвестен приложению (не
// зарегистрирован в конфигурации), не удалось подключиться к vCenter, не
// найден датацентр или ВМ с таким именем не найдена.
func GetVMDirect(vcenterHost, datacenter, cluster, vmName string) (VMPoint, error) {
	vc, ok := GetGlobalVMInfrastructure().VCenterMapList.Get(vcenterHost)
	if !ok || vc == nil {
		return VMPoint{}, fmt.Errorf("vcenter %q не найден в конфигурации приложения", vcenterHost)
	}

	c, err := vc.Connect()
	if err != nil {
		return VMPoint{}, fmt.Errorf("не удалось подключиться к vcenter %q: %w", vcenterHost, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), directQueryTimeout)
	defer cancel()
	defer c.Logout(ctx)

	f := find.NewFinder(c.Client, true)
	dc, err := f.Datacenter(ctx, datacenter)
	if err != nil {
		return VMPoint{}, fmt.Errorf("датацентр %q не найден в vcenter %q: %w", datacenter, vcenterHost, err)
	}
	f.SetDatacenter(dc)

	vmObj, err := f.VirtualMachine(ctx, vmName)
	if err != nil {
		return VMPoint{}, fmt.Errorf("виртуальная машина %q не найдена в vcenter %q, датацентр %q: %w", vmName, vcenterHost, datacenter, err)
	}

	pc := property.DefaultCollector(c.Client)
	var mvms []mo.VirtualMachine
	if err := pc.Retrieve(ctx, []types.ManagedObjectReference{vmObj.Reference()}, nil, &mvms); err != nil || len(mvms) == 0 {
		return VMPoint{}, fmt.Errorf("не удалось прочитать свойства ВМ %q в vcenter %q: %w", vmName, vcenterHost, err)
	}
	mvm := mvms[0]

	point := VMPoint{
		Vcenter:    vcenterHost,
		Datacenter: datacenter,
		Cluster:    cluster,
		Name:       mvm.Name,
		// Сравнение со строковым литералом, а не с типизированной константой
		// types.VirtualMachinePowerStatePoweredOn - тот же приём, что уже
		// используется в этом пакете (см. loadvmmetrics.go/importdata.go,
		// vm.Summary.Runtime.PowerState == "poweredOff"), поле имеет строковый
		// тип, сравнение со строковым литералом компилируется без импорта
		// дополнительных типизированных констант.
		PowerState: string(mvm.Runtime.PowerState) == "poweredOn",
	}
	if mvm.Config != nil {
		point.Template = mvm.Config.Template
		point.UUID = mvm.Config.Uuid
		point.GuestID = mvm.Config.GuestId
	}
	return point, nil
}

// GetHostDirect ищет хост (гипервизор ESXi) hostName в датацентре
// datacenter vCenter vcenterHost НАПРЯМУЮ в VMware - аналог GetVMDirect
// (см. комментарий выше), но для хостов; параметр cluster так же не
// участвует в самом поиске (используется только для заполнения
// одноимённого поля результата).
func GetHostDirect(vcenterHost, datacenter, cluster, hostName string) (HostInfo, error) {
	vc, ok := GetGlobalVMInfrastructure().VCenterMapList.Get(vcenterHost)
	if !ok || vc == nil {
		return HostInfo{}, fmt.Errorf("vcenter %q не найден в конфигурации приложения", vcenterHost)
	}

	c, err := vc.Connect()
	if err != nil {
		return HostInfo{}, fmt.Errorf("не удалось подключиться к vcenter %q: %w", vcenterHost, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), directQueryTimeout)
	defer cancel()
	defer c.Logout(ctx)

	f := find.NewFinder(c.Client, true)
	dc, err := f.Datacenter(ctx, datacenter)
	if err != nil {
		return HostInfo{}, fmt.Errorf("датацентр %q не найден в vcenter %q: %w", datacenter, vcenterHost, err)
	}
	f.SetDatacenter(dc)

	hostObj, err := f.HostSystem(ctx, hostName)
	if err != nil {
		return HostInfo{}, fmt.Errorf("хост %q не найден в vcenter %q, датацентр %q: %w", hostName, vcenterHost, datacenter, err)
	}

	pc := property.DefaultCollector(c.Client)
	var mhosts []mo.HostSystem
	if err := pc.Retrieve(ctx, []types.ManagedObjectReference{hostObj.Reference()}, nil, &mhosts); err != nil || len(mhosts) == 0 {
		return HostInfo{}, fmt.Errorf("не удалось прочитать свойства хоста %q в vcenter %q: %w", hostName, vcenterHost, err)
	}
	mhost := mhosts[0]

	info := HostInfo{
		Vcenter:    vcenterHost,
		Datacenter: datacenter,
		Cluster:    cluster,
		Name:       mhost.Name,
		PowerState: string(mhost.Runtime.PowerState) == "poweredOn",
	}
	return info, nil
}
