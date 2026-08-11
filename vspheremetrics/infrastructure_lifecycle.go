package vspheremetrics

import (
	"fmt"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/vmware/govmomi/vapi/tags"

	"time"

	"errors"
)

func (infra *VMInfrastructure) Init() {
	if infra == nil {
		fmt.Println("func (infra *VMInfrastructure) Init()", errors.New(vminfraNotInitialised))
		return
	}
	Init() // TimeOut для LockObject и циклов ожидания flEnd
	infra.VCenterMapList = cmap.New[*VCenter]()
	infra.ClusterMapList = cmap.New[*Cluster]()
	infra.VMMapList = cmap.New[*VMPoint]()
	infra.VMIDMapList = cmap.New[*VMPoint]()
	infra.VMUUIDMapList = cmap.New[*VMPoint]()
	infra.HostMapList = cmap.New[*HostInfo]()
	infra.HostBySNMapList = cmap.New[*HostInfo]()
	infra.DataStoreMapList = cmap.New[*Datastore]()

	infra.LUNMapListUUID = cmap.New[*ScsiLun]()
	infra.TagsVM = cmap.New[[]tags.AttachedTags]()
	infra.TagsDC = cmap.New[[]tags.AttachedTags]()
}

// Destroy полностью очищает все данные инфраструктуры (vCenter, кластеры, ВМ,
// хосты, хранилища, LUN, теги) и пересоздаёт карты для повторного использования.
func (infra *VMInfrastructure) Destroy() {

	if infra == nil {
		return
	}
	for _, key := range infra.VCenterMapList.Keys() {
		infra.VCenterMapList.Remove(key)
	}

	for _, key := range infra.ClusterMapList.Keys() {
		infra.ClusterMapList.Remove(key)
	}

	for _, key := range infra.VMMapList.Keys() {
		infra.VMMapList.Remove(key)
	}

	for _, key := range infra.VMIDMapList.Keys() {
		infra.VMIDMapList.Remove(key)
	}

	for _, key := range infra.VMUUIDMapList.Keys() {
		infra.VMUUIDMapList.Remove(key)
	}

	for _, key := range infra.HostMapList.Keys() {
		infra.HostMapList.Remove(key)
	}

	infra.HostBySNMapList.Clear()

	for _, key := range infra.DataStoreMapList.Keys() {
		infra.DataStoreMapList.Remove(key)
	}

	for _, key := range infra.LUNMapListUUID.Keys() {
		infra.LUNMapListUUID.Remove(key)
	}
	for _, key := range infra.TagsVM.Keys() {
		infra.TagsVM.Remove(key)
	}
	for _, key := range infra.TagsDC.Keys() {
		infra.TagsDC.Remove(key)
	}

	infra.ClusterMapList = cmap.New[*Cluster]()

	infra.VMMapList = cmap.New[*VMPoint]()
	infra.VMIDMapList = cmap.New[*VMPoint]()
	infra.VMUUIDMapList = cmap.New[*VMPoint]()

	infra.HostMapList = cmap.New[*HostInfo]()
	infra.HostBySNMapList = cmap.New[*HostInfo]()
	infra.DataStoreMapList = cmap.New[*Datastore]()

	infra.LUNMapListUUID = cmap.New[*ScsiLun]()
	infra.TagsVM = cmap.New[[]tags.AttachedTags]()
	infra.TagsDC = cmap.New[[]tags.AttachedTags]()
}

// DestroyData очищает данные инфраструктуры (кластеры, ВМ, хосты, хранилища,
// LUN, теги), сохраняя список vCenter (в отличие от Destroy).
func (infra *VMInfrastructure) DestroyData() {

	if infra == nil {
		return
	}

	for _, key := range infra.ClusterMapList.Keys() {

		infra.ClusterMapList.Remove(key)
	}

	for _, key := range infra.VMMapList.Keys() {
		infra.VMMapList.Remove(key)
	}

	for _, key := range infra.VMIDMapList.Keys() {
		infra.VMIDMapList.Remove(key)
	}

	for _, key := range infra.VMUUIDMapList.Keys() {
		infra.VMUUIDMapList.Remove(key)
	}

	for _, key := range infra.HostMapList.Keys() {
		infra.HostMapList.Remove(key)
	}

	infra.HostBySNMapList.Clear()

	for _, key := range infra.DataStoreMapList.Keys() {
		infra.DataStoreMapList.Remove(key)
	}

	for _, key := range infra.LUNMapListUUID.Keys() {
		infra.LUNMapListUUID.Remove(key)
	}
	for _, key := range infra.TagsVM.Keys() {
		infra.TagsVM.Remove(key)
	}
	for _, key := range infra.TagsDC.Keys() {
		infra.TagsDC.Remove(key)
	}
	infra.ClusterMapList = cmap.New[*Cluster]()

	infra.HostMapList = cmap.New[*HostInfo]()

	infra.HostBySNMapList = cmap.New[*HostInfo]()

	infra.DataStoreMapList = cmap.New[*Datastore]()

	infra.LUNMapListUUID = cmap.New[*ScsiLun]()
	infra.TagsVM = cmap.New[[]tags.AttachedTags]()
	infra.TagsDC = cmap.New[[]tags.AttachedTags]()
}

// DestroyPerfomance очищает накопленную историю метрик (Perf) у всех
// виртуальных машин, хостов и кластеров инфраструктуры.
func (infra *VMInfrastructure) DestroyPerfomance() {

	if infra == nil {
		return
	}
	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}
		vm.VMClearPerfomance()
		vm.Perf = make([]*VMPerf, 0)
	}

	for _, key := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		hs.HSClearPerfomance()
		hs.Perf = make([]*HSPerf, 0)
	}

	currentTime := time.Now().Local()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		cl.CLClearPerfomance()
		cl.Date = currentTime //инициализируем Дату и время в структуре кластера
	}
}

// AddVCenter регистрирует новый vCenter в инфраструктуре с указанными учётными
// данными и интервалом опроса (wait, сек.).
func (infra *VMInfrastructure) AddVCenter(hostname string, username string, password string, wait int) error {

	if infra == nil {
		return errors.New(vminfraNotInitialised)
	}

	vc := VCenter{Hostname: normalizeVCenterHost(hostname), Username: username, Password: password, Wait: wait, DoneFl: 0}

	infra.VCenterMapList.Set(vc.Hostname, &vc)
	return nil
}

// StartVcenterTimer запускает тикеры периодического опроса для каждого
// зарегистрированного vCenter (с интервалом vc.Wait) и общий часовой тикер инфраструктуры.
