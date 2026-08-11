package vspheremetrics

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/types"

	"time"

	"golang.org/x/net/context"
)

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

	totalThreads := atomic.LoadInt64(&infra.CountThread)
	vcThreads := int64(0)
	if vcp, ok := infra.VCenterMapList.Get(vc.Hostname); ok {
		vcThreads = atomic.LoadInt64(&vcp.CountThread)
	}
	log.Printf("%-30s %s ( %s ) : количество метрик по хостам: %d всего потоков %d по Vcenter %d\n", time.Now().Local().Format("2006-01-02 15:04:05"), vc.Hostname, time.Now().Local().Sub(begin).String(), len(resultHosts), totalThreads, vcThreads)
	atomic.AddUint64(&infra.CountMetrics, uint64(len(resultHosts)))
	//setMetric()
	return nil
}

// активно используемая функция
// AppendMetricToVM преобразует срез метрик производительности metrics и
// добавляет их как новый снимок (VMPerf) в историю виртуальной машины vmP,
// обновляя её агрегированные показатели.
