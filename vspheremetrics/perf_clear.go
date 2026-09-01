package vspheremetrics

import (
	"math"
	"time"
)

func (vm *VMPoint) VMClearPerfomance() {
	if vm == nil {
		return
	}

	vm.CPUUsageMaximum = 0.0
	vm.CPUUsageMaximumTime = time.Time{}
	vm.CPUUsageAverage = 0.0
	vm.CPUUsageMinimum = math.MaxFloat64
	vm.CPUUsageMinimumTime = time.Time{}
	vm.MemUsageMaximum = 0.0
	vm.MemUsageMaximumTime = time.Time{}
	vm.MemUsageAverage = 0.0
	vm.MemUsageMinimum = math.MaxFloat64
	vm.MemUsageMinimumTime = time.Time{}
	vm.DiskUsageMaximum = 0.0
	vm.DiskUsageMaximumTime = time.Time{}
	vm.DiskUsageAverage = 0.0
	vm.DiskUsageMinimum = math.MaxFloat64
	vm.DiskUsageMinimumTime = time.Time{}
	vm.DiskTotalReadLatencyMaximum = 0.0
	vm.DiskTotalReadLatencyMaximumTime = time.Time{}
	vm.DiskTotalReadLatencyAverage = 0.0
	vm.DiskTotalReadLatencyMinimum = math.MaxFloat64
	vm.DiskTotalReadLatencyMinimumTime = time.Time{}
	vm.DiskTotalWriteLatencyMaximum = 0.0
	vm.DiskTotalWriteLatencyMaximumTime = time.Time{}
	vm.DiskTotalWriteLatencyAverage = 0.0
	vm.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	vm.DiskTotalWriteLatencyMinimumTime = time.Time{}
	vm.DiskWriteMaximum = 0.0
	vm.DiskWriteMaximumTime = time.Time{}
	vm.DiskWriteAverage = 0.0
	vm.DiskWriteMinimum = math.MaxFloat64
	vm.DiskWriteMinimumTime = time.Time{}
	vm.DiskReadMaximum = 0.0
	vm.DiskReadMaximumTime = time.Time{}
	vm.DiskReadAverage = 0.0
	vm.DiskReadMinimum = math.MaxFloat64
	vm.DiskReadMinimumTime = time.Time{}
	vm.NetUsageMaximum = 0.0
	vm.NetUsageMaximumTime = time.Time{}
	vm.NetUsageAverage = 0.0
	vm.NetUsageMinimum = math.MaxFloat64
	vm.NetUsageMinimumTime = time.Time{}
	vm.PerfCount = 0
	vm.CPUBilling = 0
	vm.MemBilling = 0
	vm.DiskBilling = 0
	vm.NetBilling = 0
}

// CLClearPerfomance сбрасывает агрегированные показатели производительности (min/max/average) кластера.
func (cl *Cluster) CLClearPerfomance() {
	if cl == nil {
		return
	}

	cl.CPUMhzAvg = 0
	cl.AllNumCPUPkgs = 0
	cl.AllNumCPUCores = 0
	cl.AllNumCPUThreads = 0
	cl.AllMemorySizeGB = 0.0
	cl.CPUUsageMaximum = 0.0
	cl.CPUUsageMaximumTime = time.Time{}
	cl.CPUUsageAverage = 0.0
	cl.CPUUsageMinimum = math.MaxFloat64
	cl.CPUUsageMinimumTime = time.Time{}
	cl.MemUsageMaximum = 0.0
	cl.MemUsageMaximumTime = time.Time{}
	cl.MemUsageAverage = 0.0
	cl.MemUsageMinimum = math.MaxFloat64
	cl.MemUsageMinimumTime = time.Time{}
	cl.DiskUsageMaximum = 0.0
	cl.DiskUsageMaximumTime = time.Time{}
	cl.DiskUsageAverage = 0.0
	cl.DiskUsageMinimum = math.MaxFloat64
	cl.DiskUsageMinimumTime = time.Time{}
	cl.DiskTotalReadLatencyMaximum = 0.0
	cl.DiskTotalReadLatencyMaximumTime = time.Time{}
	cl.DiskTotalReadLatencyAverage = 0.0
	cl.DiskTotalReadLatencyMinimum = math.MaxFloat64
	cl.DiskTotalReadLatencyMinimumTime = time.Time{}
	cl.DiskTotalWriteLatencyMaximum = 0.0
	cl.DiskTotalWriteLatencyMaximumTime = time.Time{}
	cl.DiskTotalWriteLatencyAverage = 0.0
	cl.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	cl.DiskTotalWriteLatencyMinimumTime = time.Time{}
	cl.DiskWriteMaximum = 0.0
	cl.DiskWriteMaximumTime = time.Time{}
	cl.DiskWriteAverage = 0.0
	cl.DiskWriteMinimum = math.MaxFloat64
	cl.DiskWriteMinimumTime = time.Time{}
	cl.DiskReadMaximum = 0.0
	cl.DiskReadMaximumTime = time.Time{}
	cl.DiskReadAverage = 0.0
	cl.DiskReadMinimum = math.MaxFloat64
	cl.DiskReadMinimumTime = time.Time{}
	cl.NetUsageMaximum = 0.0
	cl.NetUsageMaximumTime = time.Time{}
	cl.NetUsageAverage = 0.0
	cl.NetUsageMinimum = math.MaxFloat64
	cl.NetUsageMinimumTime = time.Time{}
	cl.CPUBilling = 0.0
	cl.DiskBilling = 0.0
	cl.MemBilling = 0.0
	cl.NetBilling = 0.0
}

// HSClearPerfomance сбрасывает агрегированные показатели производительности (min/max/average) хоста.
func (hs *HostInfo) HSClearPerfomance() {
	if hs == nil {
		return
	}

	hs.CPUUsageMaximum = 0.0
	hs.CPUUsageMaximumTime = time.Time{}
	hs.CPUUsageAverage = 0.0
	hs.CPUUsageMinimum = math.MaxFloat64
	hs.CPUUsageMinimumTime = time.Time{}
	hs.MemUsageMaximum = 0.0
	hs.MemUsageMaximumTime = time.Time{}
	hs.MemUsageAverage = 0.0
	hs.MemUsageMinimum = math.MaxFloat64
	hs.MemUsageMinimumTime = time.Time{}
	hs.DiskUsageMaximum = 0.0
	hs.DiskUsageMaximumTime = time.Time{}
	hs.DiskUsageAverage = 0.0
	hs.DiskUsageMinimum = math.MaxFloat64
	hs.DiskUsageMinimumTime = time.Time{}
	hs.DiskTotalReadLatencyMaximum = 0.0
	hs.DiskTotalReadLatencyMaximumTime = time.Time{}
	hs.DiskTotalReadLatencyAverage = 0.0
	hs.DiskTotalReadLatencyMinimum = math.MaxFloat64
	hs.DiskTotalReadLatencyMinimumTime = time.Time{}
	hs.DiskTotalWriteLatencyMaximum = 0.0
	hs.DiskTotalWriteLatencyMaximumTime = time.Time{}
	hs.DiskTotalWriteLatencyAverage = 0.0
	hs.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	hs.DiskTotalWriteLatencyMinimumTime = time.Time{}
	hs.DiskWriteMaximum = 0.0
	hs.DiskWriteMaximumTime = time.Time{}
	hs.DiskWriteAverage = 0.0
	hs.DiskWriteMinimum = math.MaxFloat64
	hs.DiskWriteMinimumTime = time.Time{}
	hs.DiskReadMaximum = 0.0
	hs.DiskReadMaximumTime = time.Time{}
	hs.DiskReadAverage = 0.0
	hs.DiskReadMinimum = math.MaxFloat64
	hs.DiskReadMinimumTime = time.Time{}
	hs.NetUsageMaximum = 0.0
	hs.NetUsageMaximumTime = time.Time{}
	hs.NetUsageAverage = 0.0
	hs.NetUsageMinimum = math.MaxFloat64
	hs.NetUsageMinimumTime = time.Time{}
	hs.Perf = nil
	hs.PerfCount = 0
	// ИСПРАВЛЕНО 2026-09-01: в отличие от VMClearPerfomance/CLClearPerfomance
	// (обе сбрасывают *Billing в 0), здесь сброс отсутствовал - Billing-поля
	// хоста накапливались (SetHostMetric делает hs.CPUBilling = hs.CPUBilling
	// + ...) без ограничения на протяжении всего времени жизни процесса,
	// через каждый цикл опроса vCenter (HSClearPerfomance вызывается перед
	// каждым циклом - см. loadinventory.go, infrastructure_lifecycle.go), а
	// не только один раз. См. kontext/05-conventions-and-gotchas.md.
	hs.CPUBilling = 0.0
	hs.MemBilling = 0.0
	hs.DiskBilling = 0.0
	hs.NetBilling = 0.0
}

// ImportData заполняет поля HostInfo данными, полученными из govmomi (mo.HostSystem):
// аппаратные характеристики, версию ESXi, состояние питания и режим обслуживания.
