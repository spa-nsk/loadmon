// Команда vcsim — служебная CLI-утилита, запускающая эмулятор VMware vCenter
// (govmomi/simulator) с заданным инвентарём (датацентры, кластеры, хосты, ВМ)
// для локального тестирования сбора метрик loadmon без доступа к реальной
// инфраструктуре VMware.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vapi/rest"
	_ "github.com/vmware/govmomi/vapi/simulator" // REST /rest/com/vmware/cis/session и tags API
	"github.com/vmware/govmomi/vim25/types"
)

var vmPerfCounters = []string{
	"cpu.usage.average",
	"mem.usage.average",
	"virtualDisk.totalReadLatency.average",
	"virtualDisk.totalWriteLatency.average",
	"virtualDisk.write.average",
	"virtualDisk.read.average",
	"net.usage.average",
	"disk.usage.average",
	"net.transmitted.average",
	"net.received.average",
}

var hostPerfCounters = []string{
	"cpu.usage.average",
	"cpu.utilization.average",
	"mem.usage.average",
	"disk.deviceReadLatency.average",
	"disk.deviceWriteLatency.average",
	"disk.write.average",
	"disk.read.average",
	"net.usage.average",
	"disk.usage.average",
	"net.transmitted.average",
	"net.received.average",
}

var loadmonAggregateCounters = map[string]struct{}{
	"cpu.usage.average":                      {},
	"cpu.utilization.average":                {},
	"mem.usage.average":                      {},
	"disk.usage.average":                     {},
	"net.usage.average":                      {},
	"net.transmitted.average":                {},
	"net.received.average":                   {},
	"virtualDisk.totalReadLatency.average":   {},
	"virtualDisk.totalWriteLatency.average":  {},
	"virtualDisk.write.average":              {},
	"virtualDisk.read.average":               {},
	"disk.deviceReadLatency.average":         {},
	"disk.deviceWriteLatency.average":        {},
	"disk.write.average":                     {},
	"disk.read.average":                      {},
}

func applyModel(cfg VcsimConfig) *simulator.Model {
	model := simulator.VPX()
	inv := cfg.Inventory
	model.Autostart = cfg.Autostart
	model.Datacenter = inv.Datacenter
	model.Cluster = inv.Cluster
	model.ClusterHost = inv.ClusterHost
	model.Pool = inv.Pool
	model.Host = inv.Host
	model.Machine = inv.Machine
	model.Datastore = inv.Datastore
	model.Portgroup = inv.Portgroup
	model.App = inv.App
	return model
}

func verifyPerf(ctx context.Context, client *govmomi.Client) error {
	finder := find.NewFinder(client.Client)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		dcs, err2 := finder.DatacenterList(ctx, "*")
		if err2 != nil || len(dcs) == 0 {
			return fmt.Errorf("datacenter: %w", err)
		}
		dc = dcs[0]
	}
	finder.SetDatacenter(dc)

	vms, err := finder.VirtualMachineList(ctx, "*")
	if err != nil || len(vms) == 0 {
		return fmt.Errorf("VM: %w", err)
	}
	if err := verifyPerfForRef(ctx, client, "VM", vmPerfCounters, vms[0].Reference()); err != nil {
		return err
	}

	hosts, err := finder.HostSystemList(ctx, "*")
	if err != nil || len(hosts) == 0 {
		return fmt.Errorf("Host: %w", err)
	}
	return verifyPerfForRef(ctx, client, "Host", hostPerfCounters, hosts[0].Reference())
}

func verifyPerfForRef(ctx context.Context, client *govmomi.Client, label string, counters []string, ref types.ManagedObjectReference) error {
	pm := performance.NewManager(client.Client)
	spec := types.PerfQuerySpec{
		Format:     string(types.PerfFormatNormal),
		MaxSample:  2,
		IntervalId: 20,
		MetricId:   []types.PerfMetricId{{Instance: "*"}},
	}
	sample, err := pm.SampleByName(ctx, spec, counters, []types.ManagedObjectReference{ref})
	if err != nil {
		return fmt.Errorf("%s SampleByName: %w", label, err)
	}
	series, err := pm.ToMetricSeries(ctx, sample)
	if err != nil {
		return fmt.Errorf("%s ToMetricSeries: %w", label, err)
	}
	if len(series) == 0 {
		return fmt.Errorf("%s: perf-метрики пусты", label)
	}

	metrics := series[0]
	found := 0
	for _, v := range metrics.Value {
		if _, ok := loadmonAggregateCounters[v.Name]; !ok {
			continue
		}
		if v.Instance != "" {
			continue
		}
		if len(v.Value) == 0 || v.Value[len(v.Value)-1] <= 0 {
			return fmt.Errorf("%s: %s без значений", label, v.Name)
		}
		found++
	}
	if found < 3 {
		return fmt.Errorf("%s: недостаточно агрегированных метрик (%d)", label, found)
	}
	log.Printf("perf OK %s %s: %d/%d counters с ненулевыми значениями", label, ref.Value, found, len(counters))
	return nil
}

func verifyRestLogin(ctx context.Context, client *govmomi.Client, username, password string) error {
	rc := rest.NewClient(client.Client)
	if err := rc.Login(ctx, url.UserPassword(username, password)); err != nil {
		return fmt.Errorf("REST Login: %w", err)
	}
	defer rc.Logout(ctx)
	log.Printf("REST API OK: %s", rest.Path)
	return nil
}

func main() {
	configPath := flag.String("config", "tools/vcsim/conf/vcsim.json", "путь к конфигурации эмулятора")
	listen := flag.String("listen", "", "адрес прослушивания (host:port), переопределяет config")
	vmTarget := flag.Int("vm", 0, "целевое число ВМ (~10000); пересчитывает inventory.machine")
	verify := flag.Bool("verify", false, "проверить perf-метрики после старта")
	printLoadmon := flag.Bool("print-loadmon", false, "вывести фрагмент ConfigVCenters для app2.json и выйти")
	flag.Parse()

	cfgPath := resolveConfigPath(*configPath)
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Fatalf("config %s: %v", cfgPath, err)
	}
	if *listen != "" {
		cfg.ListenAddr = *listen
		for i := range cfg.Loadmon.ConfigVCenters {
			cfg.Loadmon.ConfigVCenters[i].Hostname = *listen
		}
	}
	cfg.applyVMTarget(*vmTarget)

	if *printLoadmon {
		printLoadmonSnippet(cfg)
		return
	}

	expected := cfg.Inventory.expectedVMs()
	log.Printf("конфиг: %s", cfgPath)
	log.Printf("инвентарь: dc=%d cluster=%d hosts/cluster=%d machine/pool=%d → ожидается ~%d VM",
		cfg.Inventory.Datacenter, cfg.Inventory.Cluster, cfg.Inventory.ClusterHost, cfg.Inventory.Machine, expected)

	model := applyModel(cfg)
	defer model.Remove()

	start := time.Now()
	log.Println("создание инвентаря vCenter (это может занять несколько минут)...")
	if err := model.Create(); err != nil {
		log.Fatalf("Create: %v", err)
	}
	count := model.Count()
	log.Printf("инвентарь создан за %s: VM=%d Host=%d Cluster=%d Datacenter=%d",
		time.Since(start).Round(time.Second), count.Machine, count.Host, count.Cluster, count.Datacenter)

	patchPerformanceManager(model)

	if cfg.TLS {
		model.Service.TLS = new(tls.Config)
	}
	model.Service.Listen = &url.URL{
		Host: cfg.ListenAddr,
		User: url.UserPassword(cfg.Username, cfg.Password),
	}
	model.Service.RegisterEndpoints = true

	s := model.Service.NewServer()
	defer s.Close()

	log.Printf("vCenter эмулятор: %s", s.URL.String())
	log.Printf("логин: %s / %s", cfg.Username, cfg.Password)
	printLoadmonSnippet(cfg)

	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, s.URL, true)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer client.Logout(ctx)

	if cfg.Perf.VerifyOnStart || *verify {
		if err := verifyPerf(ctx, client); err != nil {
			log.Fatalf("verify perf: %v", err)
		}
		if err := verifyRestLogin(ctx, client, cfg.Username, cfg.Password); err != nil {
			log.Fatalf("verify REST: %v", err)
		}
	}

	// loadmonPerfManager дополняет ответы PerfManager: Instance="" и синтетические значения.
	log.Println("метрики производительности VM/Host доступны через PerfManager (IntervalId=20)")
	log.Println("нажмите Ctrl+C для остановки")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("остановка...")
}
