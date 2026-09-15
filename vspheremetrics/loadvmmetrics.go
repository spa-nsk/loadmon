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
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"time"

	"golang.org/x/net/context"
)

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
// LoadVmMetricV4 — актуальная версия опроса метрик виртуальных машин: подключается
// к vCenter vc, пакетно получает свойства и метрики производительности всех
// подходящих (включённых, не рабочих станций) ВМ и обновляет их представление
// (VMPoint) в глобальной инфраструктуре.
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
	if errmetric != nil {
		log.Printf("LoadVmMetricV4 %s: ошибка сбора perf: %v", vc.Hostname, errmetric)
	}

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

		if vm.Config == nil { //vm.Config - указатель, у "битых"/orphaned ВМ или ВМ без прав на чтение конфигурации vSphere отдаёт его nil - без этой проверки обращение к vm.Config.Template ниже приводило к панике (nil pointer dereference, segfault signal 0xc0000005)
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
		// ИЗМЕНЕНО (2026-09-15): сохраняем имя РЕАЛЬНОГО хоста ESXi (hi -
		// уже отрезолвлен выше по vm.Summary.Runtime.Host) - раньше не
		// сохранялось никуда, из-за чего фильтр размещения VM ошибочно
		// сравнивал ГОСТЕВОЕ имя VM (vmp.HostName) с именем хоста.

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
				infra.VMUUIDMapList.Remove(keyNameUUID2)                    //
				vmP.UUID = vm.Config.Uuid                                   //
				infra.VMUUIDMapList.Set(keyNameUUID, vmP)                   //
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
			vmP.EsxHost = hi.Name

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
			vmP.EsxHost = hi.Name
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
	totalThreads := atomic.LoadInt64(&infra.CountThread)
	vcThreads := int64(0)
	if vcp, ok := infra.VCenterMapList.Get(vc.Hostname); ok {
		vcThreads = atomic.LoadInt64(&vcp.CountThread)
	}
	log.Printf("%-30s %s ( %s ) : количество метрик по виртуальным машинам: %d всего потоков %d по Vcenter %d\n", time.Now().Local().Format("2006-01-02 15:04:05"), vc.Hostname, time.Now().Local().Sub(begin).String(), metricPool.Count(), totalThreads, vcThreads)
	atomic.AddUint64(&infra.CountMetrics, uint64(metricPool.Count()))
	//setMetric()
	return nil
}
