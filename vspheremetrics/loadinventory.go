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
			lun2, ok := infra.DataStoreMapList.Get(cl.Vcenter + "/" + ref.Value)
			if ok {
				for ind, lunTmp := range cl.Storage {
					if lunTmp.UID == lun.UID {
						cl.Storage[ind] = &lun
					}
				}
				infra.DataStoreMapList.Set(cl.Vcenter+"/"+ref.Value, &lun)
				cl.AllCapacityGb = cl.AllCapacityGb - lun2.CapacityGb + lun.CapacityGb
				cl.AllFreeSpaceGb = cl.AllFreeSpaceGb - lun2.FreeSpaceGb + lun2.FreeSpaceGb
				//cl.Storage = append(cl.Storage, &lun) найти и заменить иначе будет утечка памяти
			} else {
				cl.Storage = append(cl.Storage, &lun)
				infra.DataStoreMapList.Set(cl.Vcenter+"/"+ref.Value, &lun)
				cl.StorageCount++
				ids++
				cl.AllCapacityGb = cl.AllCapacityGb + lun.CapacityGb
				cl.AllFreeSpaceGb = cl.AllFreeSpaceGb + lun.FreeSpaceGb
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
	notData := "нет данных"
	defer atomic.StoreInt32(flEnd, 1) //устанавливаем флаг завершения функции при выходе

	hostref := make([]types.ManagedObjectReference, 0)
	//hostref := make([]types.ManagedObjectReference, 500)
	for _, ref := range hostRefs {
		_, ok := infra.HostMapList.Get(vc.Hostname + "/" + ref.Value)
		if !ok { //если хост не обнаружен пропускаем добавляем в обработку
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
	if err == nil {
		for _, hs := range hst {

			datacenter := notData
			cluster := notData
			clusterID := ""
			folder := notData
			hostFlag := false
			typeWork := 0
			phi, ok := infra.HostMapList.Get(vc.Hostname + "/" + hs.Reference().Value)
			hostFound := false
			if ok && phi != nil { //если хост уже есть
				hostFound = true
				//continue //пропускаем
			}
			crList := make([]mo.ComputeResource, 0)
			clList := make([]mo.ClusterComputeResource, 0)
			refs := make([]types.ManagedObjectReference, 0)
			refsdc := make([]types.ManagedObjectReference, 0)
			refsfolder := make([]types.ManagedObjectReference, 0)
			dcList := make([]mo.Datacenter, 0)
			folderList := make([]mo.Folder, 0)

			pComputeResource := (*mo.ComputeResource)(nil)
			refs = append(refs, *hs.Parent)
			var err error
			if hs.Parent.Type == "ClusterComputeResource" {
				err = pc.Retrieve(ctx, refs, nil, &clList) //получаем данные по родительскому объекту ClusterComputeResource
			} else {
				err = pc.Retrieve(ctx, refs, nil, &crList) //получаем данные по родительскому объекту ComputeResource
				hostFlag = true
			}

			if err == nil {
				if len(clList) > 0 { //если это кластер
					pComputeResource = (*mo.ComputeResource)(&clList[0].ComputeResource)
					cluster = clList[0].Name //Имя кластера
					clusterID = fmt.Sprintf("%s:%s", vc.Hostname, pComputeResource.Reference().Value)
					refsfolder = append(refsfolder, *clList[0].Parent)
				} else if len(crList) > 0 {
					pComputeResource = (*mo.ComputeResource)(&crList[0])
					cluster = crList[0].Name //Имя одиночного хоста
					clusterID = fmt.Sprintf("%s:%s", vc.Hostname, pComputeResource.Reference().Value)
					refsfolder = append(refsfolder, *crList[0].Parent)
				}
				err = pc.Retrieve(ctx, refsfolder, nil, &folderList)
				if err == nil && len(folderList) > 0 {
					if folderList[0].Parent.Type != "Datacenter" {
						for {
							ref := *(folderList[0].Parent)
							refsfolder = make([]types.ManagedObjectReference, 0)
							folderList = make([]mo.Folder, 0)

							refsfolder = append(refsfolder, ref)
							err = pc.Retrieve(ctx, refsfolder, nil, &folderList)
							if err == nil && len(folderList) > 0 {
								if folderList[0].Parent.Type == "Datacenter" {
									break
								}
							} else {
								break
							}
						}
					}
					folder = folderList[0].Name
					refsdc = make([]types.ManagedObjectReference, 0)
					refsdc = append(refsdc, *folderList[0].Parent)
					err = pc.Retrieve(ctx, refsdc, nil, &dcList)
					if err == nil && len(dcList) > 0 {
						datacenter = dcList[0].Name
						f := find.NewFinder(c.Client, false)
						dc, errdc := f.Datacenter(ctx, datacenter)
						if errdc == nil {
							// привязываем datacenter к finder на всякий случай
							f.SetDatacenter(dc)
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
						}
					}
				}
			}
			_ = cluster
			_ = clusterID
			_ = datacenter
			_ = folder

			if cluster == notData || datacenter == notData { //пропускаем все где не заполнен кластер и датацентр 14.10.2025
				continue
			}

			cl := infra.UpdateClusterInfo(ctx, vc, c, pComputeResource, datacenter, cluster, typeWork, hostFlag)

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
				phi.ClusterID = cl.ClusterID
				phi.UnLockObject()
			} else {
				idHS := cl.HostCount //проверить и продумать альтернативу

				hi := &HostInfo{ID: idHS, Vcenter: vc.Hostname, Datacenter: datacenter, Cluster: cluster, ClusterID: cl.ClusterID}

				hi.HSClearPerfomance()
				hi.ImportData(hs)
				infra.appendScsiLun(vc.Hostname, datacenter, cluster, hs)

				hi.HostMaxVirtualDiskCapacity = cl.AllCapacityGb
				hi.KeyName = fmt.Sprintf("%s/%s", vc.Hostname, hs.Summary.Host.Value)
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
