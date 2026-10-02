package vspheremetrics

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"time"

	"golang.org/x/net/context"
)

// UpdateClusterInfo находит или создаёт запись Cluster в глобальной инфраструктуре
// по данным ComputeResource из govmomi и обновляет её основные параметры.
func (infra *VMInfrastructure) UpdateClusterInfo(ctx context.Context, vc VCenter, c *govmomi.Client, pComputeResource *mo.ComputeResource, dcname, clname string, typeWork int, hostFlag bool) *Cluster {

	pc := property.DefaultCollector(c.Client)

	clKey := fmt.Sprintf("%s/%s/%s", vc.Hostname, dcname, clname)
	cl, ok := infra.ClusterMapList.Get(clKey)

	newCluster := false
	clusterID := fmt.Sprintf("%s:%s", vc.Hostname, pComputeResource.Reference().Value)
	if !ok {
		cl = &Cluster{Date: time.Now().Local(), Vcenter: vc.Hostname, Datacenter: dcname, Name: clname, TypeWork: typeWork, ClusterID: clusterID}
		newCluster = true
	} else {
		cl.LockObject()
		cl.TypeWork = typeWork
		cl.ClusterID = clusterID
	}
	cl.Storage = make([]*Datastore, 0)
	dsList := make([]mo.Datastore, 0)
	err := pc.Retrieve(ctx, pComputeResource.Datastore, nil, &dsList)

	if err == nil {
		ids := 0
		for _, ds := range dsList {
			uid := ""

			m, ok := ds.Info.(*types.VmfsDatastoreInfo)
			if ok {
				if m.Vmfs != nil {
					if len(m.Vmfs.Extent) > 0 {
						diskName := m.Vmfs.Extent[0].DiskName
						sliceName := strings.Split(strings.ToUpper(diskName), ".")
						if sliceName[0] == "NAA" {
							uid = VmfsStrLunToUID(sliceName[1])
						} else {
							//fmt.Println(ds.Summary.Name, "diskName:", diskName)
						}
					}
				}
			} else {
				//
				nfs, ok2 := ds.Info.(*types.NasDatastoreInfo)
				if ok2 {
					if nfs.Nas != nil {
						uid = fmt.Sprintf("%s%s;%s", nfs.Nas.RemoteHost, nfs.Nas.RemotePath, nfs.Url)
					} else {
						uid = nfs.Url
					}
				}
				local, ok3 := ds.Info.(*types.LocalDatastoreInfo)
				_ = local
				_ = ok3

				mem, ok4 := ds.Info.(*types.PMemDatastoreInfo)
				_ = mem
				_ = ok4

				vs, ok5 := ds.Info.(*types.VsanDatastoreInfo)
				_ = vs
				_ = ok5

				vvol, ok6 := ds.Info.(*types.VvolDatastoreInfo)
				_ = vvol
				_ = ok6
			}

			lun := Datastore{ID: int64(ids), Type: ds.Summary.Type, UID: uid, StorageName: ds.Summary.Name,
				CapacityGb:  float64(ds.Summary.Capacity) / (1024.0 * 1024.0 * 1024.0),
				FreeSpaceGb: float64(ds.Summary.FreeSpace) / (1024.0 * 1024.0 * 1024.0),
				PercentUsed: 100.0 * (float64(ds.Summary.Capacity) - float64(ds.Summary.FreeSpace)) / float64(ds.Summary.Capacity)}
			for _, vmd := range ds.Vm {
				vmdNew := vmd
				lun.vmnames = append(lun.vmnames, vmdNew)
			}
			ref := ds.Reference()
			// lun2/ok - актуальное значение lun.datastore в ГЛОБАЛЬНОЙ карте
			// infra.DataStoreMapList (один и тот же datastore может быть
			// смонтирован хостам сразу нескольких кластеров - карта общая для
			// всех кластеров, а не только для текущего cl). Раньше ok==true
			// трактовалось как "datastore уже известен ЭТОМУ кластеру" -
			// из-за этого в cl.AllCapacityGb/AllFreeSpaceGb суммировалась
			// ёмкость shared-datastore для КАЖДОГО кластера, куда он
			// примонтирован, даже если в собственном cl.Storage этого
			// кластера записи о нём ещё не было (foundInCluster ниже отличает
			// эти два случая).
			lun2, ok := infra.DataStoreMapList.Get(cl.Vcenter + "/" + ref.Value)
			foundInCluster := false
			if ok {
				for ind, lunTmp := range cl.Storage {
					if lunTmp.UID == lun.UID {
						cl.Storage[ind] = &lun
						foundInCluster = true
					}
				}
			}
			infra.DataStoreMapList.Set(cl.Vcenter+"/"+ref.Value, &lun)
			if foundInCluster {
				cl.AllCapacityGb = cl.AllCapacityGb - lun2.CapacityGb + lun.CapacityGb
				// было lun2.FreeSpaceGb (то же старое значение ещё раз) -
				// разница всегда давала 0, и AllFreeSpaceGb переставал
				// обновляться после первого обнаружения datastore.
				cl.AllFreeSpaceGb = cl.AllFreeSpaceGb - lun2.FreeSpaceGb + lun.FreeSpaceGb
			} else {
				cl.Storage = append(cl.Storage, &lun)
				cl.StorageCount++
				ids++
				cl.AllCapacityGb = cl.AllCapacityGb + lun.CapacityGb
				cl.AllFreeSpaceGb = cl.AllFreeSpaceGb + lun.FreeSpaceGb
			}
			// AllPercentUsed раньше нигде не вычислялся (только копировался
			// между сообщениями) и всегда оставался нулевым - пересчитываем
			// его от актуальных суммарных значений при каждом обновлении.
			if cl.AllCapacityGb > 0 {
				cl.AllPercentUsed = 100.0 * (cl.AllCapacityGb - cl.AllFreeSpaceGb) / cl.AllCapacityGb
			}
		}
	}
	cl.HostFlag = hostFlag
	//infra.ClusterList = append(infra.ClusterList, cl)
	if !hostFlag { //не вносим в кластеры одиночные хосты 17.10.2025
		if newCluster {
			infra.ClusterMapList.Set(clKey, cl)
		}
	}
	if !newCluster {
		cl.UnLockObject()
	}
	return cl
}

// активно используемая функция
// UpdateHosts обновляет данные хостов ESXi (список hostRefs) в глобальной
// инфраструктуре: аппаратные характеристики, состояние, LUN и привязку к кластеру.
func (infra *VMInfrastructure) UpdateHosts(ctx context.Context, c *govmomi.Client, hostRefs []types.ManagedObjectReference, vc VCenter, flEnd *int32) {
	defer atomic.StoreInt32(flEnd, 1) //устанавливаем флаг завершения функции при выходе

	hostref := make([]types.ManagedObjectReference, 0)
	//hostref := make([]types.ManagedObjectReference, 500)
	for _, ref := range hostRefs {
		known, ok := infra.HostMapList.Get(vc.Hostname + "/" + ref.Value)
		// новый хост - в обработку; известный, но без определённого
		// датацентра/кластера (2026-10-02) - тоже, чтобы размещение
		// дозаполнилось, как только vCenter его отдаст.
		if !ok || known == nil || known.Datacenter == NoDatacenter || known.Cluster == NoCluster {
			hostref = append(hostref, ref)
		}
	}
	//блок для обработки тэгов
	cc := rest.NewClient(c.Client)
	errl := cc.Login(ctx, url.UserPassword(vc.Username, vc.Password))

	if errl != nil {
		fmt.Println("Ошибка Login:", vc.Hostname, errl)
		return
	}
	defer cc.Logout(ctx)

	t := tags.NewManager(cc)
	//--------

	pc := property.DefaultCollector(c.Client)

	//	err = pc.Retrieve(ctx, refs, []string{"name", "config", "summary","hardware","network", "datastore", "storage","guest", "guest.ipAddress", "config.extraConfig", "..."}
	var hst []mo.HostSystem
	err := pc.Retrieve(ctx, hostref, nil, &hst) //получаем данные по по всем хостам
	if err != nil && len(hostref) > 0 {
		log.Printf("UpdateHosts(%s): ошибка чтения %d хостов: %v - в этом цикле они не обработаны", vc.Hostname, len(hostref), err)
	}
	if err == nil {
		for _, hs := range hst {

			// Размещение хоста (2026-10-02, "мне важно собрать весь список
			// гипервизоров"): раньше хост, для которого не удалось подняться
			// по папкам до датацентра или прочитать родителя, молча
			// пропускался (continue) и не попадал ни в пул, ни в справочники.
			// Теперь:
			//   - датацентр - из полного списка предков хоста одним запросом
			//     (hostDatacenter / mo.Ancestors), без ручного обхода папок;
			//   - любая неудача только логируется, хост всё равно попадает в
			//     пул: датацентр/кластер подставляются NoDatacenter/NoCluster
			//     (в справочники пишутся пустыми - см. PlacementValue).
			datacenter := NoDatacenter
			cluster := NoCluster
			hostFlag := false
			typeWork := 0
			phi, ok := infra.HostMapList.Get(vc.Hostname + "/" + hs.Reference().Value)
			hostFound := false
			if ok && phi != nil { //если хост уже есть
				hostFound = true
			}
			hostName := strings.TrimSpace(hs.Name)
			if hostName == "" {
				log.Printf("UpdateHosts(%s): хост %s без имени - пропущен", vc.Hostname, hs.Reference().Value)
				continue
			}

			dcName, dcRef, dcErr := hostDatacenter(ctx, c, hs.Reference())
			if dcErr != nil {
				log.Printf("UpdateHosts(%s): хост %s: датацентр не определён (%v) - хост учитывается без датацентра", vc.Hostname, hostName, dcErr)
			} else {
				datacenter = dcName
			}

			// Родитель хоста: кластер или одиночный ComputeResource (хост вне
			// кластера - в том числе горячий резерв).
			pComputeResource := (*mo.ComputeResource)(nil)
			if hs.Parent == nil {
				log.Printf("UpdateHosts(%s): хост %s: у объекта нет родителя (кластер/ComputeResource) - хост учитывается без кластера", vc.Hostname, hostName)
			} else {
				refs := []types.ManagedObjectReference{*hs.Parent}
				var perr error
				if hs.Parent.Type == "ClusterComputeResource" {
					clList := make([]mo.ClusterComputeResource, 0)
					if perr = pc.Retrieve(ctx, refs, nil, &clList); perr == nil && len(clList) > 0 {
						pComputeResource = (*mo.ComputeResource)(&clList[0].ComputeResource)
						cluster = clList[0].Name //Имя кластера
					}
				} else {
					hostFlag = true
					crList := make([]mo.ComputeResource, 0)
					if perr = pc.Retrieve(ctx, refs, nil, &crList); perr == nil && len(crList) > 0 {
						pComputeResource = (*mo.ComputeResource)(&crList[0])
						cluster = crList[0].Name //Имя одиночного хоста (ComputeResource)
					}
				}
				if pComputeResource == nil {
					log.Printf("UpdateHosts(%s): хост %s: не удалось прочитать родителя %s (%v) - хост учитывается без кластера", vc.Hostname, hostName, hs.Parent.Value, perr)
				}
			}

			// Признак среды (тег BCS_ENV у датацентра).
			if dcRef != nil {
				if tagIDs, errt := t.ListAttachedTags(ctx, *dcRef); errt == nil {
					for _, idTag := range tagIDs {
						tag, errt := t.GetTag(ctx, idTag)
						if errt != nil {
							continue
						}
						cat, errc := t.GetCategory(ctx, tag.CategoryID)
						if errc == nil && cat.Name == "BCS_ENV" {
							switch tag.Name {
							case "PROD":
								typeWork = 1
							case "INTTEST":
								typeWork = 2
							case "EXTTEST":
								typeWork = 3
							default:
								typeWork = 0
							}
						}
					}
				}
			}

			var cl *Cluster
			if pComputeResource != nil {
				cl = infra.UpdateClusterInfo(ctx, vc, c, pComputeResource, datacenter, cluster, typeWork, hostFlag)
			} else {
				// Родитель неизвестен - временный объект кластера только для
				// заполнения HostInfo (в ClusterMapList не регистрируется,
				// как и одиночные хосты).
				cl = &Cluster{Date: time.Now().Local(), Vcenter: vc.Hostname, Datacenter: datacenter, Name: cluster, TypeWork: typeWork, HostFlag: true}
			}

			if hs.Hardware != nil {
				p := hs.Summary.Config.Product
				_ = p
				/* убрал пропуск выключенных гипервизоров и в режиме обслуживания
				if hs.Runtime.PowerState != "poweredOn" || hs.Runtime.InMaintenanceMode { //пропускаем выключенные и находящиеся в обслуживании
					continue
				}
				*/
				serialNumber := ""
				for _, info := range hs.Hardware.SystemInfo.OtherIdentifyingInfo {
					keyTag := info.IdentifierType.GetElementDescription().Key
					if keyTag == "SerialNumberTag" {
						serialNumber = info.IdentifierValue
					}
				}
				if hs.Hardware.SystemInfo.SerialNumber == "" && serialNumber == "" {
					//убираем пропуск при отсутствии серийного номера дальше будет заблокирована запись в базу без серийного номера
					//continue
				}
			} else { //если нет данных по оборудованию не обрабатываем
				//убираем отключение обработки если не указано оборудование так то видимо выключен
				//continue
			}

			if hostFound {
				phi.LockObject()
				phi.ImportData(hs)
				phi.Datacenter = datacenter
				phi.Cluster = cluster
				phi.ClusterID = cl.ClusterID
				phi.UnLockObject()
			} else {
				idHS := cl.HostCount //проверить и продумать альтернативу

				hi := &HostInfo{ID: idHS, Vcenter: vc.Hostname, Datacenter: datacenter, Cluster: cluster, ClusterID: cl.ClusterID}

				hi.HSClearPerfomance()
				hi.ImportData(hs)
				infra.appendScsiLun(vc.Hostname, datacenter, cluster, hs)

				hi.HostMaxVirtualDiskCapacity = cl.AllCapacityGb
				hi.KeyName = fmt.Sprintf("%s/%s", vc.Hostname, hs.Reference().Value) // = Summary.Host, но без паники у отключённого хоста с пустым Summary.Host
				infra.HostMapList.Set(hi.KeyName, hi)
				infra.HostBySNMapList.Set(hi.SerialNumber, hi)

				cl.LockObject()
				cl.Hosts = append(cl.Hosts, hi)
				cl.HostCount++
				cl.UnLockObject()

				fmt.Println("Добавлен host: ", vc.Hostname, datacenter, cluster, hi.Name)
			}
		}
	}
}

// обновление информации по кластерам для vcenter
// UpdateListCluster подключается к vCenter vc, обходит все датацентры и кластеры
// (ComputeResource) и обновляет их представление (Cluster) и входящие хосты в
// глобальной инфраструктуре.
func (infra *VMInfrastructure) UpdateListCluster(ctx context.Context, vc VCenter) error {
	if infra.IsLockGetMetric() {
		return nil
	}
	c, err := vc.Connect()
	if err != nil {
		log.Printf("Ошибка подключения: %s %s\n", err.Error(), vc.Hostname)
		return err
	}
	defer c.Logout(ctx)
	m2 := view.NewManager(c.Client)

	v2, err := m2.CreateContainerView(ctx, c.Client.ServiceContent.RootFolder, nil, true)
	if err != nil {
		fmt.Println("CreateContainerView:", err, vc.Hostname)
		return err
	}

	defer v2.Destroy(ctx)
	f := find.NewFinder(c.Client, true)

	dclist, err := f.DatacenterList(ctx, "*")
	if err != nil {
		fmt.Println("Список датацентров", err)
		return err
	}
	//блок для обработки тэгов
	cc := rest.NewClient(c.Client)
	errl := cc.Login(ctx, url.UserPassword(vc.Username, vc.Password))

	if errl != nil {
		fmt.Println("Ошибка Login:", vc.Hostname, errl)
		return errl
	}
	defer cc.Logout(ctx)

	t := tags.NewManager(cc)
	//--------

	for _, dc := range dclist {

		dcname, err := dc.ObjectName(ctx)
		if err != nil {
			fmt.Println("Запрос имени датацентра", err)
			return err
		}

		f.SetDatacenter(dc)

		clusterList, err := f.ClusterComputeResourceList(ctx, "*")
		if err != nil {
			log.Println("Список кластеров", err)
			continue
			//return err
		}

		typeWork := 0

		ref := dc.Reference()
		tagIDs, errt := t.ListAttachedTags(ctx, ref)
		if errt == nil {
			for _, idTag := range tagIDs {
				tag, errt := t.GetTag(ctx, idTag)
				if errt != nil {
					continue
				}
				cat, errc := t.GetCategory(ctx, tag.CategoryID)
				if errc == nil {
					if cat.Name == "BCS_ENV" {
						switch tag.Name {
						case "PROD":
							typeWork = 1
						case "INTTEST":
							typeWork = 2
						case "EXTTEST":
							typeWork = 3
						default:
							typeWork = 0
						}
					}
				}
			}
		}

		//object.ClusterComputeResource
		for _, cluster := range clusterList {
			clname, err := cluster.ObjectName(ctx)
			if err != nil {
				log.Println("Запрос имени кластера", err)
				return err
			}
			clObj := mo.ComputeResource{}
			clRef := cluster.Reference()
			err = c.RetrieveOne(ctx, clRef, nil, &clObj)

			if err == nil {
				infra.UpdateClusterInfo(ctx, vc, c, &clObj, dcname, clname, typeWork, false)
			}
		}
	}
	return nil
}

// активно используемая функция
// LoadHostMetricN подключается к vCenter vc, запрашивает метрики производительности
// всех хостов ESXi и обновляет их снимки (HSPerf) и агрегированные показатели в глобальной инфраструктуре.
