package vspheremetrics

import (
	"math"
	"time"
)

func (cl *Cluster) AddHostPerfomance(hi HostInfo) {
	if cl == nil {
		return
	}
	cl.LockObject()
	defer cl.UnLockObject()

	cl.CPUMhzAvg = cl.CPUMhzAvg + hi.CPUMhz
	cl.AllNumCPUPkgs = cl.AllNumCPUPkgs + hi.NumCPUPkgs
	cl.AllNumCPUCores = cl.AllNumCPUCores + hi.NumCPUCores
	cl.AllNumCPUThreads = cl.AllNumCPUThreads + hi.NumCPUThreads
	cl.AllMemorySizeGB = cl.AllMemorySizeGB + hi.MemorySizeGB
	cl.CPUUsageMaximum = cl.CPUUsageMaximum + hi.CPUUsageMaximum
	cl.CPUUsageAverage = cl.CPUUsageAverage + hi.CPUUsageAverage
	cl.CPUUsageMinimum = cl.CPUUsageMinimum + hi.CPUUsageMinimum
	cl.MemUsageMaximum = cl.MemUsageMaximum + hi.MemUsageMaximum
	cl.MemUsageAverage = cl.MemUsageAverage + hi.MemUsageAverage
	cl.MemUsageMinimum = cl.MemUsageMinimum + hi.MemUsageMinimum
	cl.DiskUsageMaximum = cl.DiskUsageMaximum + hi.DiskUsageMaximum
	cl.DiskUsageAverage = cl.DiskUsageAverage + hi.DiskUsageAverage
	cl.DiskUsageMinimum = cl.DiskUsageMinimum + hi.DiskUsageMinimum
	cl.DiskTotalReadLatencyMaximum = cl.DiskTotalReadLatencyMaximum + hi.DiskTotalReadLatencyMaximum
	cl.DiskTotalReadLatencyAverage = cl.DiskTotalReadLatencyAverage + hi.DiskTotalReadLatencyAverage
	cl.DiskTotalReadLatencyMinimum = cl.DiskTotalReadLatencyMinimum + hi.DiskTotalReadLatencyMinimum
	cl.DiskTotalWriteLatencyMaximum = cl.DiskTotalWriteLatencyMaximum + hi.DiskTotalWriteLatencyMaximum
	cl.DiskTotalWriteLatencyAverage = cl.DiskTotalWriteLatencyAverage + hi.DiskTotalWriteLatencyAverage
	cl.DiskTotalWriteLatencyMinimum = cl.DiskTotalWriteLatencyMinimum + hi.DiskTotalWriteLatencyMinimum
	cl.DiskWriteMaximum = cl.DiskWriteMaximum + hi.DiskWriteMaximum
	cl.DiskWriteAverage = cl.DiskWriteAverage + hi.DiskWriteAverage
	cl.DiskWriteMinimum = cl.DiskWriteMinimum + hi.DiskWriteMinimum
	cl.DiskReadMaximum = cl.DiskReadMaximum + hi.DiskReadMaximum
	cl.DiskReadAverage = cl.DiskReadAverage + hi.DiskReadAverage
	cl.DiskReadMinimum = cl.DiskReadMinimum + hi.DiskReadMinimum
	cl.NetUsageMaximum = cl.NetUsageMaximum + hi.NetUsageMaximum
	cl.NetUsageAverage = cl.NetUsageAverage + hi.NetUsageAverage
	cl.NetUsageMinimum = cl.NetUsageMinimum + hi.NetUsageMinimum
	cl.CPUBilling = cl.CPUBilling + hi.CPUBilling
	cl.DiskBilling = cl.DiskBilling + hi.DiskBilling
	cl.MemBilling = cl.DiskBilling + hi.MemBilling
	cl.NetBilling = cl.NetBilling + hi.NetBilling
	cl.CoreTransferRate = float64(cl.AllocatedCoreCount) / float64(cl.AllNumCPUCores)
	cl.MemoryTransferRate = float64(cl.AllocatedMemorySizeGb) / float64(cl.AllMemorySizeGB)
}

// CalcHostPerfomance пересчитывает суммарные ресурсы кластера по всем входящим в него хостам (Hosts).
func (cl *Cluster) CalcHostPerfomance() {
	if cl == nil {
		return
	}

	cl.LockObject()
	defer cl.UnLockObject()

	count := float64(len(cl.Hosts))
	if len(cl.Hosts) == 0 {
		return
	}

	cl.CPUMhzAvg = int32(float64(cl.CPUMhzAvg) / count)
	cl.CPUUsageMaximum = cl.CPUUsageMaximum / count
	cl.CPUUsageAverage = cl.CPUUsageAverage / count
	cl.CPUUsageMinimum = cl.CPUUsageMinimum / count
	cl.MemUsageMaximum = cl.MemUsageMaximum / count
	cl.MemUsageAverage = cl.MemUsageAverage / count
	cl.MemUsageMinimum = cl.MemUsageMinimum / count
}

// CalcPerfomance пересчитывает агрегированные метрики производительности кластера
// (CPU, память, диски, сеть) на основе метрик его хостов и виртуальных машин.
func (cl *Cluster) CalcPerfomance() {
	if cl == nil {
		return
	}

	//cl.LockObject()
	//defer cl.UnLockObject()

	if len(cl.Hosts) == 0 {
		return
	}
	maxCount := int64(0)

	// AllNumCPUPkgs/AllNumCPUCores/AllNumCPUThreads/AllMemorySizeGB/CPUMhzAvg
	// пересчитываются заново из cl.Hosts ниже через += - если не обнулить их
	// здесь, каждый повторный вызов CalcPerfomance для одного и того же
	// объекта Cluster (а при обращении через указатель, как в
	// GetClusterMetrics/GetClusterPerfByKey, это тот же объект, что лежит в
	// ClusterMapList) добавлял бы суммы хостов ПОВЕРХ уже накопленных -
	// значения росли бы без ограничений при каждом обновлении.
	cl.AllNumCPUPkgs = 0
	cl.AllNumCPUCores = 0
	cl.AllNumCPUThreads = 0
	cl.AllMemorySizeGB = 0
	cl.CPUMhzAvg = 0

	cl.CPUUsageMinimum = math.MaxFloat64
	cl.MemUsageMinimum = math.MaxFloat64
	cl.DiskUsageMinimum = math.MaxFloat64
	cl.DiskTotalReadLatencyMinimum = math.MaxFloat64
	cl.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	cl.DiskWriteMinimum = math.MaxFloat64
	cl.DiskReadMinimum = math.MaxFloat64
	cl.NetUsageMinimum = math.MaxFloat64

	for _, hs := range cl.Hosts {
		if hs.PerfCount > maxCount {
			maxCount = hs.PerfCount
		}

		cl.CPUMhzAvg += (hs.CPUMhz * int32(hs.NumCPUCores))
		cl.AllNumCPUPkgs += hs.NumCPUPkgs
		cl.AllNumCPUCores += hs.NumCPUCores
		cl.AllNumCPUThreads += hs.NumCPUThreads
		cl.AllMemorySizeGB += hs.MemorySizeGB

		if len(hs.Perf) == 0 {
			continue
		}

		cl.CPUBilling += hs.CPUBilling
		cl.DiskBilling += hs.DiskBilling
		cl.MemBilling += hs.MemBilling
		cl.NetBilling += hs.NetBilling
	}

	if cl.AllNumCPUCores > 0 {
		cl.CPUMhzAvg = cl.CPUMhzAvg / int32(cl.AllNumCPUCores)
	}

	cl.Perf = make([]*CLPerf, maxCount)

	for i := 0; i < int(maxCount); i++ {
		cl.Perf[i] = &CLPerf{}
		cl.Perf[i].Init()
		cl.Perf[i].ID = int64(i)
		cl.Perf[i].CPUUsage = 0.0
		cl.Perf[i].MemUsage = 0.0
		cl.Perf[i].DiskUsage = 0.0
		cl.Perf[i].DiskTotalReadLatency = 0.0
		cl.Perf[i].DiskTotalWriteLatency = 0.0
		cl.Perf[i].DiskWrite = 0.0
		cl.Perf[i].DiskRead = 0.0
		cl.Perf[i].NetUsage = 0.0
		cl.Perf[i].NetReceived = 0.0
		cl.Perf[i].NetTransmitted = 0.0

		for _, hs := range cl.Hosts {
			if i < len(hs.Perf) {
				if cl.Perf[i].Start.Equal(time.Time{}) {
					cl.Perf[i].Start = hs.Perf[i].Start // проверить
				}
				cl.Perf[i].CPUUsage += hs.Perf[i].CPUUsage
				cl.Perf[i].MemUsage += hs.Perf[i].MemUsage
				cl.Perf[i].DiskUsage += hs.Perf[i].DiskUsage
				cl.Perf[i].DiskTotalReadLatency += hs.Perf[i].DiskTotalReadLatency
				cl.Perf[i].DiskTotalWriteLatency += hs.Perf[i].DiskTotalWriteLatency
				cl.Perf[i].DiskWrite += hs.Perf[i].DiskWrite
				cl.Perf[i].DiskRead += hs.Perf[i].DiskRead
				cl.Perf[i].NetUsage += hs.Perf[i].NetUsage
				cl.Perf[i].NetReceived += hs.Perf[i].NetReceived
				cl.Perf[i].NetTransmitted += hs.Perf[i].NetTransmitted
			}
		}
		if len(cl.Hosts) > 0 {
			cl.Perf[i].CPUUsage /= float64(len(cl.Hosts))
			cl.Perf[i].MemUsage /= float64(len(cl.Hosts))
		}

		if cl.CPUUsageMaximum < cl.Perf[i].CPUUsage {
			cl.CPUUsageMaximum = cl.Perf[i].CPUUsage
			cl.CPUUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.CPUUsageMinimum > cl.Perf[i].CPUUsage {
			cl.CPUUsageMinimum = cl.Perf[i].CPUUsage
			cl.CPUUsageMinimumTime = cl.Perf[i].Start
		}
		cl.CPUUsageAverage += cl.Perf[i].CPUUsage

		if cl.MemUsageMaximum < cl.Perf[i].MemUsage {
			cl.MemUsageMaximum = cl.Perf[i].MemUsage
			cl.MemUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.MemUsageMinimum > cl.Perf[i].MemUsage {
			cl.MemUsageMinimum = cl.Perf[i].MemUsage
			cl.MemUsageMinimumTime = cl.Perf[i].Start
		}
		cl.MemUsageAverage += cl.Perf[i].MemUsage

		if cl.DiskUsageMaximum < cl.Perf[i].DiskUsage {
			cl.DiskUsageMaximum = cl.Perf[i].DiskUsage
			cl.DiskUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskUsageMinimum > cl.Perf[i].DiskUsage {
			cl.DiskUsageMinimum = cl.Perf[i].DiskUsage
			cl.DiskUsageMinimumTime = cl.Perf[i].Start
		}
		cl.DiskUsageAverage += cl.Perf[i].DiskUsage

		if cl.DiskWriteMaximum < cl.Perf[i].DiskWrite {
			cl.DiskWriteMaximum = cl.Perf[i].DiskWrite
			cl.DiskWriteMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskWriteMinimum > cl.Perf[i].DiskWrite {
			cl.DiskWriteMinimum = cl.Perf[i].DiskWrite
			cl.DiskWriteMinimumTime = cl.Perf[i].Start
		}
		cl.DiskWriteAverage += cl.Perf[i].DiskWrite

		if cl.DiskReadMaximum < cl.Perf[i].DiskRead {
			cl.DiskReadMaximum = cl.Perf[i].DiskRead
			cl.DiskReadMaximumTime = cl.Perf[i].Start
		}

		if cl.DiskReadMinimum > cl.Perf[i].DiskRead {
			cl.DiskReadMinimum = cl.Perf[i].DiskRead
			cl.DiskReadMinimumTime = cl.Perf[i].Start
		}
		cl.DiskReadAverage += cl.Perf[i].DiskRead

		if cl.DiskTotalReadLatencyMaximum < cl.Perf[i].DiskTotalReadLatency {
			cl.DiskTotalReadLatencyMaximum = cl.Perf[i].DiskTotalReadLatency
			cl.DiskTotalReadLatencyMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskTotalReadLatencyMinimum > cl.Perf[i].DiskTotalReadLatency {
			cl.DiskTotalReadLatencyMinimum = cl.Perf[i].DiskTotalReadLatency
			cl.DiskTotalReadLatencyMinimumTime = cl.Perf[i].Start
		}
		cl.DiskTotalReadLatencyAverage += cl.Perf[i].DiskTotalReadLatency

		if cl.DiskTotalWriteLatencyMaximum < cl.Perf[i].DiskTotalWriteLatency {
			cl.DiskTotalWriteLatencyMaximum = cl.Perf[i].DiskTotalWriteLatency
			cl.DiskTotalWriteLatencyMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskTotalWriteLatencyMinimum > cl.Perf[i].DiskTotalWriteLatency {
			cl.DiskTotalWriteLatencyMinimum = cl.Perf[i].DiskTotalWriteLatency
			cl.DiskTotalWriteLatencyMinimumTime = cl.Perf[i].Start
		}
		cl.DiskTotalWriteLatencyAverage += cl.Perf[i].DiskTotalWriteLatency

		if cl.NetUsageMaximum < cl.Perf[i].NetUsage {
			cl.NetUsageMaximum = cl.Perf[i].NetUsage
			cl.NetUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.NetUsageMinimum > cl.Perf[i].NetUsage {
			cl.NetUsageMinimum = cl.Perf[i].NetUsage
			cl.NetUsageMinimumTime = cl.Perf[i].Start
		}
		cl.NetUsageAverage += cl.Perf[i].NetUsage
	}
	switch {
	case maxCount > 0:
		cl.CPUUsageAverage /= float64(maxCount)
		cl.MemUsageAverage /= float64(maxCount)
		cl.DiskUsageAverage /= float64(maxCount)
		cl.DiskWriteAverage /= float64(maxCount)
		cl.DiskReadAverage /= float64(maxCount)
		cl.DiskTotalReadLatencyAverage /= float64(maxCount)
		cl.DiskTotalWriteLatencyAverage /= float64(maxCount)
		cl.NetUsageAverage /= float64(maxCount)
	default:
		cl.CPUUsageMaximum = 0.0
		cl.CPUUsageAverage = 0.0
		cl.CPUUsageMinimum = 0.0
		cl.MemUsageMaximum = 0.0
		cl.MemUsageAverage = 0.0
		cl.MemUsageMinimum = 0.0
		cl.DiskUsageMaximum = 0.0
		cl.DiskUsageAverage = 0.0
		cl.DiskUsageMinimum = 0.0
		cl.DiskTotalReadLatencyMaximum = 0.0
		cl.DiskTotalReadLatencyAverage = 0.0
		cl.DiskTotalReadLatencyMinimum = 0.0
		cl.DiskTotalWriteLatencyMaximum = 0.0
		cl.DiskTotalWriteLatencyAverage = 0.0
		cl.DiskTotalWriteLatencyMinimum = 0.0
		cl.DiskWriteMaximum = 0.0
		cl.DiskWriteAverage = 0.0
		cl.DiskWriteMinimum = 0.0
		cl.DiskReadMaximum = 0.0
		cl.DiskReadAverage = 0.0
		cl.DiskReadMinimum = 0.0
		cl.NetUsageMaximum = 0.0
		cl.NetUsageAverage = 0.0
		cl.NetUsageMinimum = 0.0
	}
	cl.PerfCount = (int64)(len(cl.Perf))
}

// CheckAndCorrectHSPerf проверяет и корректирует аномальные (отрицательные, NaN
// или превышающие разумные пределы) значения в снимке метрик хоста perf.
