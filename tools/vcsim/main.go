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

	vm, err := finder.VirtualMachine(ctx, "DC0_C0_RP0_VM0")
	if err != nil {
		vms, err2 := finder.VirtualMachineList(ctx, "*")
		if err2 != nil || len(vms) == 0 {
			return fmt.Errorf("не найдена VM для проверки perf: %w", err)
		}
		vm = vms[0]
	}

	name, err := vm.ObjectName(ctx)
	if err != nil {
		return err
	}

	pm := performance.NewManager(client.Client)
	spec := types.PerfQuerySpec{
		Format:     string(types.PerfFormatNormal),
		MaxSample:  2,
		IntervalId: 20,
		MetricId:   []types.PerfMetricId{{Instance: "*"}},
	}
	sample, err := pm.SampleByName(ctx, spec, vmPerfCounters, []types.ManagedObjectReference{vm.Reference()})
	if err != nil {
		return fmt.Errorf("SampleByName: %w", err)
	}
	series, err := pm.ToMetricSeries(ctx, sample)
	if err != nil {
		return fmt.Errorf("ToMetricSeries: %w", err)
	}
	if len(series) == 0 {
		return fmt.Errorf("perf-метрики пусты для %s", name)
	}

	log.Printf("perf OK для %s: %d counter series", name, len(series))
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

	// vcsim генерирует perf-метрики при QueryPerf; loadmon опрашивает их каждые Wait секунд.
	log.Println("метрики производительности доступны через PerfManager (IntervalId=20)")
	log.Println("нажмите Ctrl+C для остановки")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("остановка...")
}
