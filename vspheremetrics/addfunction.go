package vspheremetrics

import (
	//"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"strings"
	"sync"

	cmap "github.com/orcaman/concurrent-map/v2"
	//"github.com/vmware/govmomi"
	//"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vapi/rest"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	//	"github.com/vmware/govmomi/units"
	"golang.org/x/net/context"
	//"os"
	"time"
	//"text/tabwriter"
)

// SetPerfInterval подключается к vCenter, выводит в лог доступные интервалы
// сбора статистики производительности (PerfInterval) и включает первый из них.
func (infra *VMInfrastructure) SetPerfInterval(ctx context.Context, vc VCenter) error {
	c, err := vc.Connect()
	if err != nil {
		log.Printf("Ошибка подключения: %s\n", err.Error())
		return err
	}
	defer c.Logout(ctx)

	m2 := view.NewManager(c.Client)

	v2, err := m2.CreateContainerView(ctx, c.Client.ServiceContent.RootFolder, nil, true)
	if err != nil {
		fmt.Println("CreateContainerView:", err)
		return err
	}

	defer v2.Destroy(ctx)
	perfManager := performance.NewManager(c.Client)

	intervals, err := perfManager.HistoricalInterval(ctx) //[]types.PerfInterval
	if err != nil {
		fmt.Println("HistoricalInterval:", err)
		return err
	}

	for _, i := range intervals {
		period := (time.Duration(i.SamplingPeriod) * time.Second).String()
		period = strings.TrimSuffix(period, "0s")
		if strings.Contains(period, "h") {
			period = strings.TrimSuffix(period, "0m")
		}
		samples := i.Length / i.SamplingPeriod

		fmt.Printf("ID:\t%d\n", i.SamplingPeriod)
		fmt.Printf("  Enabled:\t%t\n", i.Enabled)
		fmt.Printf("  Interval:\t%s\n", period)
		fmt.Printf("  Available Samples:\t%d\n", samples)
		fmt.Printf("  Name:\t%s\n", i.Name)
		fmt.Printf("  Level:\t%d\n", i.Level)
	}
	current := types.PerfInterval{}

	if len(intervals) > 0 {
		current = intervals[0]
	}

	current.Enabled = true

	_, err = methods.UpdatePerfInterval(ctx, c.Client, &types.UpdatePerfInterval{
		This:     perfManager.Reference(),
		Interval: current,
	})
	return nil
}

// LoadVmMetric (устаревшая версия опроса метрик ВМ, см. также LoadVmMetricV4)
// подключается к vCenter, получает свойства и метрики производительности всех
// включённых серверных виртуальных машин и обновляет соответствующие записи
// VMPoint в глобальной инфраструктуре (создавая новые при необходимости).
func (infra *VMInfrastructure) LoadVmMetric(ctx context.Context, vc VCenter) error {

	if infra.IsLockGetMetric() {
		return nil
	}

	begin := time.Now().Local()

	c, err := vc.Connect()
	if err != nil {
		log.Printf("Ошибка подключения: %s\n", err.Error())
		return err
	}
	defer c.Logout(ctx)

	cc := rest.NewClient(c.Client)
	err = cc.Login(ctx, url.UserPassword(vc.Username, vc.Password))
	t := tags.NewManager(cc)

	m2 := view.NewManager(c.Client)

	v2, err := m2.CreateContainerView(ctx, c.Client.ServiceContent.RootFolder, nil, true)
	if err != nil {
		fmt.Println("CreateContainerView:", err)
		return err
	}

	defer v2.Destroy(ctx)

	vmsRefs, err := v2.Find(ctx, []string{"VirtualMachine"}, nil)
	if err != nil {
		fmt.Println("Find VirtualMachine:", err)
		//continue
		return err
	}

	var wg sync.WaitGroup

	metricPool := cmap.New[[]performance.EntityMetric]()
	wg.Add(1)
	go func() error {
		defer wg.Done()
		// Create a PerfManager
		perfManager := performance.NewManager(c.Client)

		// Create PerfQuerySpec
		spec := types.PerfQuerySpec{
			MaxSample:  2,
			MetricId:   []types.PerfMetricId{{Instance: "*"}},
			IntervalId: 20,
		}

		sample, err := perfManager.SampleByName(ctx, spec, []string{"cpu.usage.average", "mem.usage.average",
			"virtualDisk.totalReadLatency.average", "virtualDisk.totalWriteLatency.average",
			"virtualDisk.write.average", "virtualDisk.read.average", "net.usage.average",
			"disk.usage.average", "net.transmitted.average", "net.received.average"}, vmsRefs) //метрики по всем виртуальным машинам
		if err != nil {
			fmt.Println("perfManager.SampleByName VirtualMachine:", err)
			//continue
			return err
		}

		result, err := perfManager.ToMetricSeries(ctx, sample)
		if err != nil {
			fmt.Println("perfManager.ToMetricSeries VirtualMachine:", err)
			//continue
			return err
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
		return nil
	}()

	vmPool := cmap.New[mo.VirtualMachine]()

	wg.Add(1)
	go func() error {
		defer wg.Done()
		var vmt []mo.VirtualMachine
		pc := property.DefaultCollector(c.Client)

		//		err = pc.Retrieve(ctx, refs, []string{"name", "config", "summary","hardware","network", "datastore", "storage","guest", "guest.ipAddress", "config.extraConfig"}
		err = pc.Retrieve(ctx, vmsRefs, nil, &vmt) //получаем данные по по всем виртуальным машинам
		if err != nil {
			fmt.Println("Поиск свойств всех виртуальных машин", err)
			//infra.VM.Unlock()
			//continue
			return err
		}
		for _, vm := range vmt {
			ref := vm.Reference()
			key := ref.Value
			vmPool.Set(key, vm)
		}
		return nil
	}()

	count := 0

	for _, vmRef := range vmsRefs {

		wg.Wait() //ждем завершения
		if count == 0 {
			count = metricPool.Count()
		}
		key := vmRef.Value
		vm, ok := vmPool.Get(key)
		if !ok {
			continue
		}
		if vm.Summary.Runtime.PowerState == "poweredOff" { //пропускаем выключенные машины
			continue
		}

		osUp := strings.ToUpper(vm.Summary.Config.GuestFullName)
		if strings.Contains(osUp, "WINDOWS") {
			if !strings.Contains(osUp, "SERVER") {
				continue // это рабочая станция windows
			}
		}

		resultvm, ok := metricPool.Get(key)

		if !ok {
			continue
		} //так то аномалия

		for _, metric := range resultvm { //проходим по пулу метрик виртуальных машин
			var cl *Cluster = nil
			var hi *HostInfo = nil
			var ok bool

			if vm.Summary.Runtime.Host == nil { //на каком host размещена vm
				continue
			}
			hi, ok = infra.HostMapList.Get(vc.Hostname + "/" + vm.Summary.Runtime.Host.Value) //есть ли информация по этому host
			if !ok {
				continue
			}
			cpuMhz := 0.0
			notCluster := "VM не в кластере"
			notDatacenter := "нет данных"
			clname := notCluster
			dcname := notDatacenter

			cpuMhz = float64(hi.CPUMhz)

			if hi.Datacenter != "" { //если хост не входит в датацентр
				dcname = hi.Datacenter
			}

			if dcname == notDatacenter { //если нет информации по датацентру - игнорируем
				continue
			}

			if hi.Cluster != "" { //если хост входит в кластер
				clname = hi.Cluster
			}

			if clname != notCluster {
				ok := false
				key := fmt.Sprintf("%s/%s/%s", vc.Hostname, dcname, clname)
				cl, ok = infra.ClusterMapList.Get(key)
				if !ok {

				}
			}

			if cl == nil { //если нет информации по кластеру - игнорируем
				continue
			}

			refs := make([]mo.Reference, 0)
			refs = append(refs, vm.Reference())
			attachedTags, errt := t.GetAttachedTagsOnObjects(ctx, refs)

			keyName := fmt.Sprintf("%s/%s/%s/%s", vc.Hostname, dcname, clname, vm.Name)

			if errt == nil {
				tvm, ok := infra.TagsVM.Get(keyName)
				if !ok {
					infra.TagsVM.Set(keyName, attachedTags)
				} else {
					if len(tvm) > 0 && len(attachedTags) > 0 {
						if len(tvm[0].Tags) > 0 && len(attachedTags[0].Tags) > 0 {
							if tvm[0].Tags[0].Name != attachedTags[0].Tags[0].Name {
								infra.TagsVM.Set(keyName, attachedTags)
							}
						}
					}
				}
			}
			//ошибка если cluster не найден
			vmP, ok := infra.VMMapList.Get(keyName) //найти виртуальную машину

			if !ok { //если новая виртуальная машина, добавляем её
				idVM := int64(infra.VMMapList.Count())

				vmP = &VMPoint{ID: idVM, Vcenter: vc.Hostname, Datacenter: dcname, Cluster: clname}
				if len(attachedTags) > 0 {
					if len(attachedTags[0].Tags) > 0 {
						vmP.TagName = attachedTags[0].Tags[0].Name
						vmP.TagDescription = attachedTags[0].Tags[0].Description
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

				vmP.KeyName = vc.Hostname + "/" + dcname + "/" + clname + "/" + vm.Name
				cl.VM = append(cl.VM, vmP)
				cl.VMCount++
				cl.AllocatedCoreCount += int(vmP.NumCores)
				cl.AllocatedMemorySizeGb += int(vmP.MemorySizeGB)
				infra.VMMapList.Set(vmP.KeyName, vmP)
				fmt.Printf("Добавлена виртуальная машина %s %s %s %s %s\n", vmRef, vmP.Name, vmP.Cluster, vmP.Datacenter, vmP.Vcenter)
			} else {
				vmP.ImportData(vm, infra, vc.Hostname, dcname, clname) //обновляем данные
				vmP.EsxHost = hi.Name
				if len(attachedTags) > 0 {
					if len(attachedTags[0].Tags) > 0 {
						vmP.TagName = attachedTags[0].Tags[0].Name
						vmP.TagDescription = attachedTags[0].Tags[0].Description
					}
				}
			}
			perf := VMPerf{}
			perf.Init()
			if len(metric.SampleInfo) > 0 {
				perf.Start = metric.SampleInfo[0].Timestamp
			} // время среза метрик и интервал по первой записи
			if len(vmP.Perf) == 0 {
				vmP.Perf = make([]*VMPerf, 0)
			}
			SetVMPerfMetrics(&perf, metric)
			CheckAndCorrectVMPerf(&perf)

			perf.ID = vmP.ID //Id виртуалки в метрики

			SetVMMetric(vmP, perf, cpuMhz)

			vmP.Perf = append(vmP.Perf, &perf) //добавить метрики
			vmP.PerfCount++
		}
	}
	log.Printf("%-30s %s ( %s ) : количество метрик по виртуальным машинам: %d\n", time.Now().Local().Format("2006-01-02 15:04:05"), vc.Hostname, time.Now().Local().Sub(begin).String(), count)
	return nil
}
