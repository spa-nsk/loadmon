package vspheremetrics

import (
	"fmt"
	"log"
	"strings"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"

	"golang.org/x/net/context"
)

// PruneStaleObjects удаляет из закэшированного в памяти пула
// (VMInfrastructure) виртуальные машины, хосты и кластеры vCenter vc,
// которые больше не существуют в самой VMware.
//
// Зачем это нужно: metrics_grpc работает без перезапуска, а
// UpdateListCluster/UpdateHosts/LoadVmMetricV4 (см. loadinventory.go,
// loadvmmetrics.go) только ДОБАВЛЯЮТ/ОБНОВЛЯЮТ записи в
// VMMapList/VMUUIDMapList/HostMapList/ClusterMapList - объект, удалённый,
// перемещённый за пределы наблюдаемых кластеров или переименованный (со
// стороны VMware) в VMware, без отдельной сверки навсегда остаётся в
// памяти сборщика. PruneStaleObjects вызывается после каждого опроса
// конкретного vCenter (см. config.pollVCenter) и приводит содержимое
// пула в соответствие с фактическим состоянием этого vCenter прямо
// сейчас.
//
// Подход: для vc строится "живой" набор идентификаторов (по отдельному
// govmomi-соединению и container view, т.е. без использования кэша) для
// ВМ (UUID), хостов (MOID) и кластеров (Datacenter/Cluster), после чего
// из соответствующих карт убираются все записи этого vCenter, ключ/UUID
// которых не попал в "живой" набор. Если построить "живой" набор не
// удалось (ошибка соединения/запроса) - сверка для этого vCenter в этом
// цикле пропускается целиком, чтобы не удалить актуальные данные из-за
// временной недоступности vCenter.
func (infra *VMInfrastructure) PruneStaleObjects(ctx context.Context, vc VCenter) error {
	if infra.IsLockGetMetric() {
		return nil
	}

	c, err := vc.Connect()
	if err != nil {
		log.Printf("PruneStaleObjects: ошибка подключения %s: %v", vc.Hostname, err)
		return err
	}
	defer c.Logout(ctx)

	m := view.NewManager(c.Client)
	v, err := m.CreateContainerView(ctx, c.Client.ServiceContent.RootFolder, nil, true)
	if err != nil {
		log.Printf("PruneStaleObjects: ошибка CreateContainerView %s: %v", vc.Hostname, err)
		return err
	}
	defer v.Destroy(ctx)

	pc := property.DefaultCollector(c.Client)

	// ---- живой набор хостов (ключ совпадает с ключом HostMapList) ----
	hostRefs, err := v.Find(ctx, []string{"HostSystem"}, nil)
	if err != nil {
		log.Printf("PruneStaleObjects: ошибка списка хостов %s: %v", vc.Hostname, err)
		return err
	}
	liveHostKeys := make(map[string]struct{}, len(hostRefs))
	for _, ref := range hostRefs {
		liveHostKeys[vc.Hostname+"/"+ref.Value] = struct{}{}
	}

	// ---- живой набор ВМ (ключ совпадает с ключом VMUUIDMapList) ----
	vmRefs, err := v.Find(ctx, []string{"VirtualMachine"}, nil)
	if err != nil {
		log.Printf("PruneStaleObjects: ошибка списка ВМ %s: %v", vc.Hostname, err)
		return err
	}
	liveVMUUIDKeys := make(map[string]struct{}, len(vmRefs))
	if len(vmRefs) > 0 {
		var vms []mo.VirtualMachine
		if err := pc.Retrieve(ctx, vmRefs, []string{"config.uuid"}, &vms); err != nil {
			log.Printf("PruneStaleObjects: ошибка чтения UUID ВМ %s: %v", vc.Hostname, err)
			return err
		}
		for _, vm := range vms {
			if vm.Config == nil || vm.Config.Uuid == "" {
				continue
			}
			liveVMUUIDKeys[vc.Hostname+"/"+vm.Config.Uuid] = struct{}{}
		}
	}

	// ---- живой набор кластеров (ключ совпадает с ключом ClusterMapList) ----
	liveClusterKeys, err := buildLiveClusterKeySet(ctx, c.Client, vc)
	if err != nil {
		log.Printf("PruneStaleObjects: ошибка списка кластеров %s: %v", vc.Hostname, err)
		return err
	}

	prefix := vc.Hostname + "/"

	// удаляем хосты, отсутствующие в VMware
	for _, key := range infra.HostMapList.Keys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if _, ok := liveHostKeys[key]; ok {
			continue
		}
		hi, ok := infra.HostMapList.Get(key)
		infra.HostMapList.Remove(key)
		if ok && hi != nil && hi.SerialNumber != "" {
			infra.HostBySNMapList.Remove(hi.SerialNumber)
		}
		log.Println("PruneStaleObjects: удалён отсутствующий в VMware host:", key)
	}

	// удаляем ВМ, отсутствующие в VMware
	for _, key := range infra.VMMapList.Keys() {
		vmP, ok := infra.VMMapList.Get(key)
		if !ok || vmP == nil || vmP.Vcenter != vc.Hostname {
			continue
		}
		uuidKey := vc.Hostname + "/" + vmP.UUID
		if _, ok := liveVMUUIDKeys[uuidKey]; ok {
			continue
		}
		infra.VMMapList.Remove(key)
		infra.VMUUIDMapList.Remove(uuidKey)
		log.Println("PruneStaleObjects: удалена отсутствующая в VMware ВМ:", key)
	}
	// подчищаем VMUUIDMapList отдельно - на случай если ключ VMMapList уже
	// был обновлён (переименование) до того, как объект пропал из VMware
	for _, key := range infra.VMUUIDMapList.Keys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if _, ok := liveVMUUIDKeys[key]; ok {
			continue
		}
		infra.VMUUIDMapList.Remove(key)
	}

	// удаляем кластеры, отсутствующие в VMware
	for _, key := range infra.ClusterMapList.Keys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if _, ok := liveClusterKeys[key]; ok {
			continue
		}
		infra.ClusterMapList.Remove(key)
		log.Println("PruneStaleObjects: удалён отсутствующий в VMware кластер:", key)
	}

	return nil
}

// buildLiveClusterKeySet обходит все датацентры vc и возвращает набор
// ключей вида "vCenter/Датацентр/Кластер" для всех ClusterComputeResource
// (обычных кластеров) - в том же формате, что использует UpdateClusterInfo
// при добавлении в ClusterMapList (одиночные хосты, hostFlag==true, туда
// не попадают - см. комментарий "не вносим в кластеры одиночные хосты" в
// loadinventory.go, поэтому и здесь не учитываются).
func buildLiveClusterKeySet(ctx context.Context, gc *vim25.Client, vc VCenter) (map[string]struct{}, error) {
	f := find.NewFinder(gc, true)

	dclist, err := f.DatacenterList(ctx, "*")
	if err != nil {
		return nil, err
	}

	live := make(map[string]struct{})
	for _, dc := range dclist {
		dcname, err := dc.ObjectName(ctx)
		if err != nil {
			continue
		}
		f.SetDatacenter(dc)

		clusterList, err := f.ClusterComputeResourceList(ctx, "*")
		if err != nil {
			// в датацентре нет ни одного кластера (только одиночные хосты) - не ошибка
			continue
		}
		for _, cluster := range clusterList {
			clname, err := cluster.ObjectName(ctx)
			if err != nil {
				continue
			}
			live[fmt.Sprintf("%s/%s/%s", vc.Hostname, dcname, clname)] = struct{}{}
		}
	}
	return live, nil
}
