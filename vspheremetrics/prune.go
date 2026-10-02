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
// ВМ, хостов (MOID) и кластеров (Datacenter/Cluster), после чего из
// соответствующих карт убираются все записи этого vCenter, ключ которых не
// попал в "живой" набор. Если построить "живой" набор не удалось (ошибка
// соединения/запроса) - сверка для этого vCenter в этом цикле пропускается
// целиком, чтобы не удалить актуальные данные из-за временной
// недоступности vCenter.
//
// ⚠️ ВМ сверяются по составному ключу "vCenter/Датацентр/Кластер/Имя" (тот
// же формат, что использует VMMapList), а НЕ по UUID: в исходных данных
// VMware встречаются виртуальные машины с дублирующимся UUID (известная,
// пока не устранённая проблема данных), из-за чего UUID не может служить
// надёжным уникальным идентификатором для сверки существования - при
// коллизии UUID можно было бы случайно удалить ещё существующую ВМ только
// потому, что другая ВМ с тем же UUID пропала. Уникальным для этой сверки
// считается именно vCenter+Датацентр+Кластер+Имя.
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

	// ---- живой набор ВМ (ключ вида "vCenter/Датацентр/Кластер/Имя",
	// совпадает с ключом VMMapList - см. комментарий к функции выше про
	// дубли UUID) ----
	vmRefs, err := v.Find(ctx, []string{"VirtualMachine"}, nil)
	if err != nil {
		log.Printf("PruneStaleObjects: ошибка списка ВМ %s: %v", vc.Hostname, err)
		return err
	}
	liveVMNameKeys := make(map[string]struct{}, len(vmRefs))
	if len(vmRefs) > 0 {
		var vms []mo.VirtualMachine
		// "name" и "summary.runtime.host" достаточно, чтобы построить тот же
		// ключ, что и при сборе метрик (см. loadvmmetrics.go: dcname/clname
		// определяются не прямым обходом иерархии, а через уже
		// заполненный на этом же цикле опроса infra.HostMapList по хосту
		// ВМ) - используем тот же приём здесь, чтобы ключи гарантированно
		// совпадали по формату.
		if err := pc.Retrieve(ctx, vmRefs, []string{"name", "summary.runtime.host"}, &vms); err != nil {
			log.Printf("PruneStaleObjects: ошибка чтения имён ВМ %s: %v", vc.Hostname, err)
			return err
		}
		for _, vm := range vms {
			if vm.Summary.Runtime.Host == nil {
				continue // без хоста не определить датацентр/кластер - как и при сборе метрик, такую ВМ пропускаем
			}
			hi, ok := infra.HostMapList.Get(vc.Hostname + "/" + vm.Summary.Runtime.Host.Value)
			if !ok || hi == nil {
				continue // хост этой ВМ ещё не в пуле - не можем построить ключ, пропускаем (сверка по ней в этом цикле не делается)
			}
			liveVMNameKeys[fmt.Sprintf("%s/%s/%s/%s", vc.Hostname, hi.Datacenter, hi.Cluster, vm.Name)] = struct{}{}
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

	// удаляем ВМ, отсутствующие в VMware - сверка по имени
	// (vCenter/Датацентр/Кластер/Имя), не по UUID, см. комментарий к функции.
	for _, key := range infra.VMMapList.Keys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if _, ok := liveVMNameKeys[key]; ok {
			continue
		}
		vmP, ok := infra.VMMapList.Get(key)
		infra.VMMapList.Remove(key)
		if ok && vmP != nil {
			uuidKey := vc.Hostname + "/" + vmP.UUID
			// Удаляем запись в VMUUIDMapList, только если она всё ещё
			// указывает именно на эту (устаревшую) ВМ - при дублирующихся
			// UUID тот же ключ прямо сейчас может принадлежать другой,
			// реально существующей ВМ (см. комментарий к функции выше), и
			// её задевать нельзя.
			if cur, curOk := infra.VMUUIDMapList.Get(uuidKey); curOk && cur == vmP {
				infra.VMUUIDMapList.Remove(uuidKey)
			}
		}
		log.Println("PruneStaleObjects: удалена отсутствующая в VMware ВМ:", key)
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

	// Пул по этому vCenter теперь достоверен - см. vspheremetrics/syncstate.go.
	MarkVCenterSynced(vc.Hostname)
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
