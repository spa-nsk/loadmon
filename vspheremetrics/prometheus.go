package vspheremetrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cpu = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_num_cpu",
			Help: "VM CPU count",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)

	core = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm__num_core",
			Help: "VM CPU core count",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)

	mem = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_memory_size_gb",
			Help: "VM Memory size in gb",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)

	diskCapacity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_disk_capacity_gb",
			Help: "VM Disk size in gb",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)

	diskFree = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_disk_free_space_gb",
			Help: "VM Disk free in gb",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)

	diskPercent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_disk_percent",
			Help: "VM Disk used percent",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)

	cpuUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_cpu_usage_average_percent",
			Help: "VM CPU Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	memUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_mem_usage_average_percent",
			Help: "VM Memory Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	diskUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_disk_usage_average_mb_sec",
			Help: "VM Disk Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	diskReadUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_virtualDisk_read_average_mb_sec",
			Help: "VM Disk read Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	diskWriteUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_virtualDisk_write_average_mb_sec",
			Help: "VM Disk write Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	diskReadLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_virtualDisk_totalReadLatency_average_ms",
			Help: "VM Disk Read Latency Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	diskWriteLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_virtualDisk_totalWriteLatency_average_ms",
			Help: "VM Disk Write Latency Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	netUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_net_usage_average_mb_sec",
			Help: "VM Network Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	netReceived = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_net_received_average_mb_sec",
			Help: "VM Network Received Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
	netTransmitted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vm_net_transmite_average_mb_sec",
			Help: "VM Network Transmitted Perfonmance",
		},
		[]string{"Vcenter", "Datacenter", "Cluster", "Name", "HostName"},
	)
)

func init() {
	prometheus.MustRegister(cpu)
	prometheus.MustRegister(core)
	prometheus.MustRegister(mem)
	prometheus.MustRegister(diskCapacity)
	prometheus.MustRegister(diskFree)
	prometheus.MustRegister(diskPercent)
	prometheus.MustRegister(cpuUsage)
	prometheus.MustRegister(memUsage)
	prometheus.MustRegister(diskUsage)
	prometheus.MustRegister(diskReadUsage)
	prometheus.MustRegister(diskWriteUsage)
	prometheus.MustRegister(diskReadLatency)
	prometheus.MustRegister(diskWriteLatency)
	prometheus.MustRegister(netUsage)
	prometheus.MustRegister(netReceived)
	prometheus.MustRegister(netTransmitted)
}

// SetMetric публикует текущие значения показателей всех виртуальных машин
// глобальной инфраструктуры (ресурсы и последний снимок метрик производительности)
// в зарегистрированные Prometheus-гейджи для последующего скрейпа через /metrics.
func SetMetric() {
	if infrastructureGlobal == nil {
		return
	}
	if infrastructureGlobal.IsLockGetMetric() {
		return
	}
	for _, key := range infrastructureGlobal.VMMapList.Keys() {
		vm, ok := infrastructureGlobal.VMMapList.Get(key)
		if !ok || vm == nil {
			continue
		}
		if vm.LockObject() != nil {
			return
		}
		defer vm.UnLockObject()

		if len(vm.Perf) < 1 {
			continue
		}

		cpu.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(float64(vm.NumCPU))
		core.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(float64(vm.NumCores))
		mem.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(vm.MemorySizeGB)
		diskCapacity.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(vm.CapacityGb)
		diskFree.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(vm.FreeSpaceGb)
		diskPercent.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(vm.PercentUsed)
		ind := len(vm.Perf) - 1
		if ind >= 0 {
			if vm.Perf[ind] == nil {
				continue
			}
			perf := *vm.Perf[ind]
			cpuUsage.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.CPUUsage)
			memUsage.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.MemUsage)
			diskUsage.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.DiskUsage)
			diskReadLatency.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.DiskTotalReadLatency)
			diskWriteLatency.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.DiskTotalWriteLatency)
			diskWriteUsage.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.DiskWrite)
			diskReadUsage.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.DiskRead)
			netUsage.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.NetUsage)
			netReceived.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.NetReceived)
			netTransmitted.WithLabelValues(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name, vm.HostName).Set(perf.NetTransmitted)
		}
	}
}
