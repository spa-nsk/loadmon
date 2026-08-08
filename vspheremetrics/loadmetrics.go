package vspheremetrics

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"strings"
	"sync/atomic"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"golang.org/x/net/context"
	"time"
)

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
func (infra *VMInfrastructure) LoadHostMetricN(ctx context.Context, vc VCenter) error {

	if infra.IsLockGetMetric() {
		return nil
	}

	begin := time.Now().Local()
	c, err := vc.Connect()
	if err != nil {
		log.Printf("Ошибка подключения: %s %s\n", err.Error(), vc.Hostname)
		return err
	}
	defer c.Logout(ctx)

	//infra.UpdateHostInfo(ctx, c, vc.Hostname)
	m2 := view.NewManager(c.Client)

	v2, err := m2.CreateContainerView(ctx, c.Client.ServiceContent.RootFolder, nil, true)
	if err != nil {
		fmt.Println("CreateContainerView:", err, vc.Hostname)
		return err
	}

	defer v2.Destroy(ctx)

	hostRefs, errh := v2.Find(ctx, []string{"HostSystem"}, nil)
	if errh != nil {
		fmt.Println("Find HostSystem:", errh, vc.Hostname)
		//continue
		return errh
	}

	flEnd := int32(0)

	go infra.UpdateHosts(ctx, c, hostRefs, vc, &flEnd)

	// Create a PerfManager
	perfManager := performance.NewManager(c.Client)

	// Create PerfQuerySpec
	spec := types.PerfQuerySpec{
		MaxSample:  2,
		MetricId:   []types.PerfMetricId{{Instance: "*"}},
		IntervalId: 20,
	}

	// Query metrics
	sampleHosts, errh := perfManager.SampleByName(ctx, spec, []string{"cpu.usage.average", "cpu.utilization.average", "mem.usage.average",
		"disk.deviceReadLatency.average", "disk.deviceWriteLatency.average",
		"disk.write.average", "disk.read.average", "net.usage.average",
		"disk.usage.average", "net.transmitted.average", "net.received.average"}, hostRefs)
	if errh != nil {
		fmt.Println("perfManager.SampleByName HostSystem:", errh, vc.Hostname)
		//continue
		return errh
	}

	resultHosts, errh := perfManager.ToMetricSeries(ctx, sampleHosts)
	if errh != nil {
		fmt.Println("perfManager.ToMetricSeries HostSystem:", errh, vc.Hostname)
		//continue
		return errh
	}

	for i := int32(0); !atomic.CompareAndSwapInt32(&flEnd, 1, 0); i++ { //ожидаем 1 в переменной flEnd
		if i >= atomic.LoadInt32(&TimeOut) {
			return errors.New(timeOutError)
		}
		time.Sleep(100 * time.Millisecond)
		//runtime.Gosched() //передаём управление планировщику
	}
	// хосты обновлены, можно работать дальше
	for _, metric := range resultHosts {
		hi, ok := infra.HostMapList.Get(vc.Hostname + "/" + metric.Entity.Value)
		if !ok { //если хост не обнаружен пропускаем (аномалия)
			hostref := make([]types.ManagedObjectReference, 0)
			flEnd := int32(0)
			hostref = append(hostref, metric.Entity)
			infra.UpdateHosts(ctx, c, hostref, vc, &flEnd)
			hi, ok = infra.HostMapList.Get(vc.Hostname + "/" + metric.Entity.Value)
			if !ok {
				continue
			}
			//continue
		}

		perf := &HSPerf{}

		if len(metric.SampleInfo) > 0 {
			perf.Start = metric.SampleInfo[0].Timestamp.Local()
		} // время среза метрик и интервал по первой записи
		if len(hi.Perf) == 0 {
			hi.Perf = make([]*HSPerf, 0)
			//hi.Perf = make([]*HSPerf, 4320) //3*60*24 - каждые 20 сек
		}
		SetHostPerfMetrics(perf, metric)
		CheckAndCorrectHSPerf(perf)
		perf.ID = hi.ID //id host
		SetHostMetric(hi, *perf)
		hi.Perf = append(hi.Perf, perf) //добавить метрики
		hi.PerfCount++
	}

	log.Printf("%-30s %s ( %s ) : количество метрик по хостам: %d всего потоков %d по Vcenter %d\n", time.Now().Local().Format("2006-01-02 15:04:05"), vc.Hostname, time.Now().Local().Sub(begin).String(), len(resultHosts), 0, 0)
	atomic.AddUint64(&infra.CountMetrics, uint64(len(resultHosts)))
	//setMetric()
	return nil
}

// активно используемая функция
func (infra *VMInfrastructure) AppendMetricToVM(vmP *VMPoint, metrics []performance.EntityMetric, cpuMhz float64) {
	for _, metric := range metrics {
		perf := VMPerf{}
		perf.Init()
		if len(metric.SampleInfo) > 0 {
			perf.Start = metric.SampleInfo[0].Timestamp
		} // время среза метрик и интервал по первой записи
		if len(vmP.Perf) == 0 {
			vmP.Perf = make([]*VMPerf, 0)
			//vmP.Perf = make([]*VMPerf, 4320) // 3*60*24 каждые 20 сек
		}
		SetVMPerfMetrics(&perf, metric)
		CheckAndCorrectVMPerf(&perf)

		perf.ID = vmP.ID //ID виртуалки в метрики

		SetVMMetric(vmP, perf, cpuMhz)
		vmP.LockObject()
		defer vmP.UnLockObject()

		vmP.Perf = append(vmP.Perf, &perf) //добавить метрики
		vmP.PerfCount++
	}
}

// активно используемая функция
func (infra *VMInfrastructure) LoadVmMetricV4(ctx context.Context, vc VCenter) error {

	if infra.IsLockGetMetric() {
		return nil
	}

	begin := time.Now().Local()

	c, err := vc.Connect()
	if err != nil {
		log.Printf("Ошибка подключения: %s %s\n", vc.Hostname, err.Error())
		return err
	}
	defer c.Logout(ctx)

	//запускаем поток для сбора данных по vm
	flDoneVM := int64(0)
	flVMResf := int64(0)

	var (
		vmsRefs []types.ManagedObjectReference
		errvm   error
	)
	vms := make([]mo.VirtualMachine, 0)

	go func() {
		var (
			m2 *view.Manager
			v2 *view.ContainerView
		)

		defer atomic.StoreInt64(&flDoneVM, 1)
		m2 = view.NewManager(c.Client)

		v2, errvm = m2.CreateContainerView(ctx, c.Client.ServiceContent.RootFolder, nil, true)
		if errvm != nil {
			//fmt.Println("Ошибка CreateContainerView:", vc.Hostname, errvm)
			return
		}

		defer v2.Destroy(ctx)

		//получаем список всех vm
		vmsRefs, errvm = v2.Find(ctx, []string{"VirtualMachine"}, nil)
		if errvm != nil {
			//fmt.Println("Ошибка Find VirtualMachine:", vc.Hostname, errvm)
			return
		}
		atomic.StoreInt64(&flVMResf, 1) //vmsRefs заполнена
		//получаем данные по всем vm
		pc := property.DefaultCollector(c.Client)

		errvm = pc.Retrieve(ctx, vmsRefs, nil, &vms)
		if errvm != nil {
			//fmt.Println("Ошибка Find VirtualMachine:", vc.Hostname, errvm)
			return
		}
		errvm = nil
	}()

	flDoneMetric := int64(0)
	metricPool := cmap.New[[]performance.EntityMetric]()
	errmetric := error(nil)

	go func() {
		var (
			sample []types.BasePerfEntityMetricBase
			result []performance.EntityMetric
		)
		defer atomic.StoreInt64(&flDoneMetric, 1)

		// Create a PerfManager
		perfManager := performance.NewManager(c.Client)

		// Create PerfQuerySpec
		spec := types.PerfQuerySpec{
			MaxSample:  2,
			MetricId:   []types.PerfMetricId{{Instance: "*"}},
			IntervalId: 20,
		}

		for i := int32(0); !atomic.CompareAndSwapInt64(&flVMResf, 1, 0); i++ { //ожидаем заполнения данных в vmsRefs
			if i >= atomic.LoadInt32(&TimeOut) {
				//fmt.Println("Ошибка получения VMResf:", vc.Hostname, timeOutError)
				return
			}
			time.Sleep(100 * time.Millisecond)
			//runtime.Gosched()
		}

		sample, errmetric = perfManager.SampleByName(ctx, spec, []string{"cpu.usage.average", "mem.usage.average",
			"virtualDisk.totalReadLatency.average", "virtualDisk.totalWriteLatency.average",
			"virtualDisk.write.average", "virtualDisk.read.average", "net.usage.average",
			"disk.usage.average", "net.transmitted.average", "net.received.average"}, vmsRefs) //метрики по всем виртуальным машинам
		if errmetric != nil {
			//fmt.Println("Ошибка perfManager.SampleByName VirtualMachine:", vc.Hostname, errmetric)
			return
		}

		result, errmetric = perfManager.ToMetricSeries(ctx, sample)
		if errmetric != nil {
			//fmt.Println("Ошибка perfManager.ToMetricSeries VirtualMachine:", vc.Hostname, errmetric)
			return
		}
		for _, metric := range result {
			key := metric.Entity.Value
			metrics, ok := metricPool.Get(key)
			if ok {
				metricPool.Set(key, append(metrics, metric))
			} else {
				metricPool.Set(key, []performance.EntityMetric{metric})
			}
		}

		errmetric = nil
	}()

	cc := rest.NewClient(c.Client)
	err = cc.Login(ctx, url.UserPassword(vc.Username, vc.Password))

	if err != nil {
		fmt.Println("Ошибка Login:", vc.Hostname, err)
		return err
	}
	defer cc.Logout(ctx)

	t := tags.NewManager(cc)

	//ждем окончания функции по сбору данных по vm
	for i := int32(0); !atomic.CompareAndSwapInt64(&flDoneVM, 1, 0); i++ {
		if i >= atomic.LoadInt32(&TimeOut) {
			return errors.New(timeOutError)
		}
		time.Sleep(100 * time.Millisecond)
		//runtime.Gosched()
	}

	//ждем окончания функции по сбору данных по metrics
	for i := int32(0); !atomic.CompareAndSwapInt64(&flDoneMetric, 1, 0); i++ {
		if i >= atomic.LoadInt32(&TimeOut) {
			return errors.New(timeOutError)
		}
		time.Sleep(100 * time.Millisecond)
		//runtime.Gosched()
	}
	_ = errvm
	_ = errmetric

	workstation := false
	skipNameUp1 := "CP-REPLICA"
	skipNameUp2 := "CP-TEMPLATE"
	for _, vm := range vms {
		if vm.Summary.Runtime.PowerState == "poweredOff" { //пропускаем выключенные машины - убрал для появления в учёте новых серверов но пока выключен
			//continue
		}
		nameUp := strings.ToUpper(vm.Name)
		if strings.Contains(nameUp, skipNameUp1) || strings.Contains(nameUp, skipNameUp2) { //игнорируем по шаблону
			continue
		}

		workstation = false
		osUp := strings.ToUpper(vm.Summary.Config.GuestFullName)
		if strings.Contains(osUp, "WINDOWS") {
			if !strings.Contains(osUp, "SERVER") {
				workstation = true // это рабочая станция windows
			}
		}
		notData := "нет данных"
		dcname := notData
		clname := notData

		if vm.Summary.Runtime.Host == nil {
			continue
		}

		if vm.Config.Template { //пропускаем шаблоны
			continue
		}

		if strings.Contains(vm.Name, "vCLS.") || strings.Contains(vm.Name, "tmpl-") { //пропускаем шаблоны
			continue
		}
		//данные сетевых карт
		/*
			finger := find.NewFinder(c.Client, true)
			dc, err := finger.DefaultDatacenter(ctx)
			if err != nil {
				continue
			}

			finger.SetDatacenter(dc)
			vm2, err := finger.VirtualMachine(ctx, vm.Name)
			devices, err := vm2.Device(ctx)
			if err != nil {
				continue
			}
			nics := devices.SelectByType((*types.VirtualEthernetCard)(nil))
			for _, nic := range nics {
				card := nic.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
				name := devices.Name(nic)
				primaryMAC := devices.PrimaryMacAddress()
				_ = name
				_ = card.MacAddress
				_ = primaryMAC
			}
		*/
		//
		hi, ok := infra.HostMapList.Get(vc.Hostname + "/" + vm.Summary.Runtime.Host.Value)

		if !ok { //если нет данных по хосту, пропускаем
			continue
		}

		dcname = hi.Datacenter
		clname = hi.Cluster

		key := fmt.Sprintf("%s/%s/%s", vc.Hostname, dcname, clname)

		if dcname == notData { //пропускаем все если нет данных по датацентру 14.10.2025
			continue
		}

		//заглушка для VDI
		//if vc.Hostname == "s-vc-03.global.bcs" && dcname == "Datacenter" && clname == "CL-VDI-CMN-01" {
		//	continue
		//}
		//

		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue //нет данных по кластеру, пропускаем
		}

		hi.LockObject()
		hi.ClusterID = cl.ClusterID
		hi.UnLockObject()

		refs := make([]mo.Reference, 0)
		refs = append(refs, vm.Reference()) //запрашиваем тэги
		tags, errt := t.GetAttachedTagsOnObjects(ctx, refs)

		keyName := fmt.Sprintf("%s/%s/%s/%s", vc.Hostname, dcname, clname, vm.Name)

		keyNameUUID := ""

		switch keyName {
		case "s-mf-vc-01.global.bcs/MF-DC-01/CL-MF-DITRB/s-mf-portal-ws-03":
			keyNameUUID = fmt.Sprintf("%s/%s", vc.Hostname, "420f697e-e446-e1ae-3f18-442e1403fb6f-s-mf-portal-ws-03")
		case "s-mf-vc-01.global.bcs/MF-DC-01/CL-MF-DITRB/s-mf-portal-ws-07":
			keyNameUUID = fmt.Sprintf("%s/%s", vc.Hostname, "420f697e-e446-e1ae-3f18-442e1403fb6f-s-mf-portal-ws-07")
		case "s-vc-02.global.bcs/BCS-Production/CL-PROD-CMN-03/s-rds-sh-45":
			keyNameUUID = fmt.Sprintf("%s/%s", vc.Hostname, "420f1855-aae8-5f6e-98e6-d9ea9adef0ca-s-rds-sh-45")
		case "s-vc-02.global.bcs/BCS-Production/CL-PROD-CMN-03/S-NSK01-SH00045":
			keyNameUUID = fmt.Sprintf("%s/%s", vc.Hostname, "420f1855-aae8-5f6e-98e6-d9ea9adef0ca-S-NSK01-SH00045")
		default:
			keyNameUUID = fmt.Sprintf("%s/%s", vc.Hostname, vm.Summary.Config.Uuid)
		}

		if errt == nil {
			tvm, ok := infra.TagsVM.Get(keyName)
			if !ok {
				infra.TagsVM.Set(keyName, tags)
			} else {
				if len(tvm) > 0 && len(tags) > 0 {
					if len(tvm[0].Tags) > 0 && len(tags[0].Tags) > 0 {
						if tvm[0].Tags[0].Name != tags[0].Tags[0].Name {
							infra.TagsVM.Set(keyName, tags)
						}
					}
				}
			}
		}
		//
		keyMetric := vm.Reference().Value
		resultVM, ok := metricPool.Get(keyMetric)

		if !ok {
			continue
		} //нет метрик по vm

		if dcname == notData || clname == notData { // пропускаем все vm если нет данных по кластеру или датацентру 14.10.2025
			continue
		}

		//--------------------- для анализа использования
		/*
			customValue := ""
			for val := range vm.Summary.CustomValue {
				customValue += fmt.Sprint(val, " ")
			}
			_ = customValue //дополнительные значения
			if vm.Guest != nil {
				if vm.Guest.ToolsStatus == "toolsOk" { //tools установлен корректно

				}
				if len(vm.Guest.IpStack) > 1 {
					fmt.Println("lrn(IpStack) > 1")
				}
				for _, ipSt := range vm.Guest.IpStack {
					if ipSt.DnsConfig != nil {
						_ = ipSt.DnsConfig.HostName     //имя хоста s-nskrt-mspki01
						_ = ipSt.DnsConfig.DomainName   //имя домена global.bcs
						for _, ipAddress := range ipSt.DnsConfig.IpAddress {
							_ = ipAddress //IP адреса
						}
					}
				}
			}
		*/
		//------------------------
		//vmP, ok := infra.VMMapList.Get(keyName) //найти виртуальную машину
		vmP, ok := infra.VMUUIDMapList.Get(keyNameUUID) //найти виртуальную машину по UUID уникальный идентификатор VM в рамках vcenter

		//обход ситуации с созданием новых машин в VDI с одинаковым именем
		if !ok {
			vmP, ok = infra.VMMapList.Get(keyName) //найти виртуальную машину по имени в обход
			if ok {
				keyNameUUID2 := fmt.Sprintf("%s/%s", vc.Hostname, vmP.UUID) //делаем подмену UUID
				infra.VMUUIDMapList.Remove(keyNameUUID2)                   //
				vmP.UUID = vm.Config.Uuid                                  //
				infra.VMUUIDMapList.Set(keyNameUUID, vmP)                  //
			}
		}
		//
		if !ok { //если новая виртуальная машина, добавляем её

			//idVM := int64(infra.VMMapList.Count())
			idVM := int64(infra.VMUUIDMapList.Count())

			vmP = &VMPoint{ID: idVM, Vcenter: vc.Hostname, Datacenter: dcname, Cluster: clname, WorkStation: workstation, ClusterID: cl.ClusterID, TypeWork: cl.TypeWork}
			if len(tags) > 0 {
				if len(tags[0].Tags) > 0 {
					vmP.TagName = tags[0].Tags[0].Name
					vmP.TagDescription = tags[0].Tags[0].Description
				}
			}
			vmP.CPUUsageMinimum = math.MaxFloat64
			vmP.MemUsageMinimum = math.MaxFloat64
			vmP.DiskUsageMinimum = math.MaxFloat64
			vmP.DiskTotalReadLatencyMinimum = math.MaxFloat64
			vmP.DiskTotalWriteLatencyMinimum = math.MaxFloat64
			vmP.DiskWriteMinimum = math.MaxFloat64
			vmP.DiskReadMinimum = math.MaxFloat64
			vmP.NetUsageMinimum = math.MaxFloat64

			vmP.ImportData(vm, infra, vc.Hostname, dcname, clname)

			//vmP.KeyName = vc.Hostname + "/" + dcname + "/" + clname + "/" + vm.Name
			vmP.KeyName = keyName
			cl.VM = append(cl.VM, vmP)
			cl.VMCount++
			cl.AllocatedCoreCount += int(vmP.NumCores)
			cl.AllocatedMemorySizeGb += int(vmP.MemorySizeGB)

			infra.AppendMetricToVM(vmP, resultVM, float64(hi.CPUMhz))

			infra.VMMapList.Set(vmP.KeyName, vmP)
			infra.VMUUIDMapList.Set(keyNameUUID, vmP) //запоминаем с ключом UUID

			fmt.Printf("Добавлена виртуальная машина %s %s %s %s %s\n", vm.Reference().Value, vmP.Name, vmP.Cluster, vmP.Datacenter, vmP.Vcenter)
		} else {
			if vmP.KeyName != keyName { //произошло переименование или перенос в другой датацентр или cluster
				fmt.Printf("Переименование или перенос:\n%s  UUID %s\n%s  UUID %s\n", vmP.KeyName, vmP.UUID, keyName, vm.Summary.Config.Uuid)
				infra.VMMapList.Remove(vmP.KeyName)   //удалить виртуальную машину по старому ключу
				vmP.KeyName = keyName                 //обновить ключ
				infra.VMMapList.Set(vmP.KeyName, vmP) //вставить по новому ключу решаем вопрос с переименованием виртуальных машин и переносами в другие места внутри vcenter

				vmP.Vcenter = vc.Hostname
				vmP.Datacenter = dcname
				vmP.Cluster = clname
				vmP.ClusterID = cl.ClusterID
			}
			vmP.ImportData(vm, infra, vc.Hostname, dcname, clname) //обновляем данные
			vmP.WorkStation = workstation
			vmP.ClusterID = cl.ClusterID
			infra.AppendMetricToVM(vmP, resultVM, float64(hi.CPUMhz))
			if len(tags) > 0 {
				if len(tags[0].Tags) > 0 {
					vmP.TagName = tags[0].Tags[0].Name
					vmP.TagDescription = tags[0].Tags[0].Description
				}
			}
		}
	}
	log.Printf("%-30s %s ( %s ) : количество метрик по виртуальным машинам: %d\n", time.Now().Local().Format("2006-01-02 15:04:05"), vc.Hostname, time.Now().Local().Sub(begin).String(), metricPool.Count())
	atomic.AddUint64(&infra.CountMetrics, uint64(metricPool.Count()))
	//setMetric()
	return nil
}
