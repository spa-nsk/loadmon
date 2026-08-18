package main

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/types"
)

func main() {
	u, _ := url.Parse("https://vcsim@local:pass@127.0.0.1:18989/sdk")
	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, u, true)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Logout(ctx)

	finder := find.NewFinder(client.Client)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		dcs, err2 := finder.DatacenterList(ctx, "*")
		if err2 != nil || len(dcs) == 0 {
			log.Fatalf("datacenter: %v", err)
		}
		dc = dcs[0]
	}
	finder.SetDatacenter(dc)

	vms, err := finder.VirtualMachineList(ctx, "*")
	if err != nil || len(vms) == 0 {
		log.Fatalf("VMs: %v len=%d", err, len(vms))
	}
	hosts, err := finder.HostSystemList(ctx, "*/*")
	if err != nil || len(hosts) == 0 {
		hosts, err = finder.HostSystemList(ctx, "*")
		if err != nil || len(hosts) == 0 {
			log.Fatalf("Hosts: %v len=%d", err, len(hosts))
		}
	}
	pm := performance.NewManager(client.Client)
	specStar := types.PerfQuerySpec{MaxSample: 2, IntervalId: 20, MetricId: []types.PerfMetricId{{Instance: "*"}}}
	specAgg := types.PerfQuerySpec{MaxSample: 2, IntervalId: 20, MetricId: []types.PerfMetricId{{Instance: ""}}}

	vmCounters := []string{"cpu.usage.average", "mem.usage.average", "virtualDisk.totalReadLatency.average", "virtualDisk.totalWriteLatency.average", "virtualDisk.write.average", "virtualDisk.read.average", "net.usage.average", "disk.usage.average", "net.transmitted.average", "net.received.average"}
	hostCounters := []string{"cpu.usage.average", "cpu.utilization.average", "mem.usage.average", "disk.deviceReadLatency.average", "disk.deviceWriteLatency.average", "disk.write.average", "disk.read.average", "net.usage.average", "disk.usage.average", "net.transmitted.average", "net.received.average"}

	tests := []struct {
		label    string
		refs     []types.ManagedObjectReference
		counters []string
	}{
		{"VM", []types.ManagedObjectReference{vms[0].Reference()}, vmCounters},
		{"Host", []types.ManagedObjectReference{hosts[0].Reference()}, hostCounters},
	}
	for _, t := range tests {
		for _, spec := range []types.PerfQuerySpec{specStar, specAgg} {
			sample, err := pm.SampleByName(ctx, spec, t.counters, t.refs)
			if err != nil {
				log.Fatalf("%s inst=%q: %v", t.label, spec.MetricId[0].Instance, err)
			}
			series, err := pm.ToMetricSeries(ctx, sample)
			if err != nil {
				log.Fatalf("%s series inst=%q: %v", t.label, spec.MetricId[0].Instance, err)
			}
			for _, s := range series {
				fmt.Printf("=== %s %s inst=%q: %d series ===\n", t.label, s.Entity.Value, spec.MetricId[0].Instance, len(s.Value))
				for _, v := range s.Value {
					fmt.Printf("  %q inst=%q vals=%v\n", v.Name, v.Instance, v.Value)
				}
			}
		}
	}
}
