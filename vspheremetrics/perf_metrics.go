package vspheremetrics

import (
	"math"

	"github.com/vmware/govmomi/performance"
)

func CheckAndCorrectHSPerf(perf *HSPerf) {
	if perf == nil {
		return
	}

	if perf.CPUUsage > 100 {
		perf.CPUUsage = 100.0
	}
	if perf.MemUsage > 100 {
		perf.MemUsage = 100.0

	}
	if math.IsNaN(perf.CPUUsage) {
		perf.CPUUsage = 0.0
	}
	if math.IsNaN(perf.MemUsage) {
		perf.MemUsage = 0.0
	}
	if math.IsNaN(perf.DiskUsage) {
		perf.DiskUsage = 0.0
	}
	if math.IsNaN(perf.NetUsage) {
		perf.NetUsage = 0.0
	}
	if math.IsNaN(perf.DiskTotalReadLatency) {
		perf.DiskTotalReadLatency = 0.0
	}
	if math.IsNaN(perf.DiskTotalWriteLatency) {
		perf.DiskTotalWriteLatency = 0.0
	}
	if math.IsNaN(perf.DiskWrite) {
		perf.DiskWrite = 0.0
	}
	if math.IsNaN(perf.DiskRead) {
		perf.DiskRead = 0.0
	}
}

// SetHostPerfMetrics заполняет структуру perf значениями из среза метрик
// производительности хоста metric, полученного через govmomi/performance.
func SetHostPerfMetrics(perf *HSPerf, metric performance.EntityMetric) {
	if perf == nil {
		return
	}

	for _, v := range metric.Value {
		if len(v.Value) > 0 {
			//fmt.Printf("1 - %s\n", v.Name)
			switch v.Name {
			case "net.transmitted.average":
				if v.Instance == "" {
					perf.NetTransmitted = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				}
				//perf.NetTransmittedMap = append(perf.NetTransmittedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
			case "net.received.average":
				if v.Instance == "" {
					perf.NetReceived = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				}
				//perf.NetReceivedMap = append(perf.NetReceivedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
			case "cpu.usage.average":
				if v.Instance == "" {
					perf.CPUUsage = float64(maxValue(v.Value)) / 100.0
				} else {
					//perf.CPUUsageMap = append(perf.CPUUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 100.0})
				}
			case "cpu.utilization.average":
				if v.Instance == "" {
					perf.CPUUtilization = float64(maxValue(v.Value)) / 100.0
				}
			case "mem.usage.average":
				if v.Instance == "" {
					perf.MemUsage = float64(maxValue(v.Value)) / 100.0
				}
			case "disk.usage.average":
				if v.Instance == "" {
					perf.DiskUsage = float64(maxValue(v.Value)) / 1024.0
				}
			case "net.usage.average": //может быть несколько сетевых карт с именем в Instance
				if v.Instance == "4000" {
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
				if v.Instance == "" {
					perf.NetUsage = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "disk.deviceReadLatency.average":
				if v.Instance == "" {
					perf.DiskTotalReadLatency = float64(maxValue(v.Value)) //ms
				} else {
					//perf.DiskTotalReadLatencyMap = append(perf.DiskTotalReadLatencyMap, Metric{v.Instance, float64(maxValue(v.Value))}) //ms
				}
			case "disk.deviceWriteLatency.average":
				if v.Instance == "" {
					perf.DiskTotalWriteLatency = float64(maxValue(v.Value)) //ms
				} else {
					//perf.DiskTotalWriteLatencyMap = append(perf.DiskTotalWriteLatencyMap, Metric{v.Instance, float64(maxValue(v.Value))}) //ms
				}
			case "disk.write.average":
				if v.Instance == "" {
					perf.DiskWrite = float64(maxValue(v.Value)) / 1024.0
				} else {
					//perf.DiskWriteMap = append(perf.DiskWriteMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0})
				}
			case "disk.read.average":
				if v.Instance == "" {
					perf.DiskRead = float64(maxValue(v.Value)) / 1024.0
				} else {
					//perf.DiskReadMap = append(perf.DiskReadMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0})
				}
			}
		}
	}
}

// SetHostMetric добавляет снимок метрик perf к истории хоста hi и обновляет его агрегированные показатели (min/max/average).
func SetHostMetric(hi *HostInfo, perf HSPerf) {
	if hi == nil {
		return
	}

	hi.LockObject()
	defer hi.UnLockObject()

	hi.CPUBilling = hi.CPUBilling + float64(perf.CPUUsage)*float64(hi.CPUMhz)*float64(hi.NumCPUCores)/180.0 //утилизация процессора частота в мегагерцах умноженная процент утилизации и промежуток времени
	hi.MemBilling = hi.MemBilling + float64(perf.MemUsage)*hi.MemorySizeGB*1024.0/180.0                     // утилизация памяти мегабайт часах

	if hi.CPUUsageMaximum < perf.CPUUsage {
		hi.CPUUsageMaximum = perf.CPUUsage
		hi.CPUUsageMaximumTime = perf.Start
	}
	if hi.CPUUsageMinimum > perf.CPUUsage {
		hi.CPUUsageMinimum = perf.CPUUsage
		hi.CPUUsageMinimumTime = perf.Start
	}
	hi.CPUUsageAverage = hi.CPUUsageAverage + perf.CPUUsage

	if hi.MemUsageMaximum < perf.MemUsage {
		hi.MemUsageMaximum = perf.MemUsage
		hi.MemUsageMaximumTime = perf.Start
	}
	if hi.MemUsageMinimum > perf.MemUsage {
		hi.MemUsageMinimum = perf.MemUsage
		hi.MemUsageMinimumTime = perf.Start
	}
	hi.MemUsageAverage = hi.MemUsageAverage + perf.MemUsage

	if hi.NetUsageMaximum < perf.NetUsage {
		hi.NetUsageMaximum = perf.NetUsage
		hi.NetUsageMaximumTime = perf.Start
	}
	if hi.NetUsageMinimum > perf.NetUsage {
		hi.NetUsageMinimum = perf.NetUsage
		hi.NetUsageMinimumTime = perf.Start
	}
	hi.NetUsageAverage = hi.NetUsageAverage + perf.NetUsage

	if hi.DiskUsageMaximum < perf.DiskUsage {
		hi.DiskUsageMaximum = perf.DiskUsage
		hi.DiskUsageMaximumTime = perf.Start
	}
	if hi.DiskUsageMinimum > perf.DiskUsage {
		hi.DiskUsageMinimum = perf.DiskUsage
		hi.DiskUsageMinimumTime = perf.Start
	}
	hi.DiskUsageAverage = hi.DiskUsageAverage + perf.DiskUsage

	if hi.DiskReadMaximum < perf.DiskRead {
		hi.DiskReadMaximum = perf.DiskRead
		hi.DiskReadMaximumTime = perf.Start
	}
	if hi.DiskReadMinimum > perf.DiskRead {
		hi.DiskReadMinimum = perf.DiskRead
		hi.DiskReadMinimumTime = perf.Start
	}
	hi.DiskReadAverage = hi.DiskReadAverage + perf.DiskRead

	if hi.DiskWriteMaximum < perf.DiskWrite {
		hi.DiskWriteMaximum = perf.DiskWrite
		hi.DiskWriteMaximumTime = perf.Start
	}
	if hi.DiskWriteMinimum > perf.DiskWrite {
		hi.DiskWriteMinimum = perf.DiskWrite
		hi.DiskWriteMinimumTime = perf.Start
	}
	hi.DiskWriteAverage = hi.DiskWriteAverage + perf.DiskWrite

	if hi.DiskTotalReadLatencyMaximum < perf.DiskTotalReadLatency {
		hi.DiskTotalReadLatencyMaximum = perf.DiskTotalReadLatency
		hi.DiskTotalReadLatencyMaximumTime = perf.Start
	}
	if hi.DiskTotalReadLatencyMinimum > perf.DiskTotalReadLatency {
		hi.DiskTotalReadLatencyMinimum = perf.DiskTotalReadLatency
		hi.DiskTotalReadLatencyMinimumTime = perf.Start
	}
	hi.DiskTotalReadLatencyAverage = hi.DiskTotalReadLatencyAverage + perf.DiskTotalReadLatency

	if hi.DiskTotalWriteLatencyMaximum < perf.DiskTotalWriteLatency {
		hi.DiskTotalWriteLatencyMaximum = perf.DiskTotalWriteLatency
		hi.DiskTotalWriteLatencyMaximumTime = perf.Start
	}
	if hi.DiskTotalWriteLatencyMinimum > perf.DiskTotalWriteLatency {
		hi.DiskTotalWriteLatencyMinimum = perf.DiskTotalWriteLatency
		hi.DiskTotalWriteLatencyMinimumTime = perf.Start
	}
	hi.DiskTotalWriteLatencyAverage = hi.DiskTotalWriteLatencyAverage + perf.DiskTotalWriteLatency
}

// SetVMPerfMetrics заполняет структуру perf значениями из среза метрик
// производительности виртуальной машины metric, полученного через govmomi/performance.
func SetVMPerfMetrics(perf *VMPerf, metric performance.EntityMetric) {
	if perf == nil {
		return
	}

	for _, v := range metric.Value {
		if len(v.Value) > 0 {
			//fmt.Printf("1 - %s\n", v.Name)
			switch v.Name {
			case "net.transmitted.average":
				if v.Instance == "" {
					perf.NetTransmitted = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				}
				if v.Instance == "4000" {
					//perf.NetTransmittedMap = append(perf.NetTransmittedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "net.received.average":
				perf.NetReceived = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				if v.Instance == "4000" {
					//perf.NetReceivedMap = append(perf.NetReceivedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "cpu.usage.average":
				if v.Instance == "" {
					perf.CPUUsage = float64(maxValue(v.Value)) / 100.0 //unit:"percent"
				} else {
					//perf.CPUUsageMap = append(perf.CPUUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 100.0}) //unit:"percent"
				}
			case "mem.usage.average":
				if v.Instance == "" {
					perf.MemUsage = float64(maxValue(v.Value)) / 100.0 //unit:"percent"
				}
			case "disk.usage.average":
				if v.Instance == "" {
					perf.DiskUsage = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				} else {
					//perf.DiskUsageMap = append(perf.DiskUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "net.usage.average": //может быть несколько сетевых карт с именем в Instance
				if v.Instance == "4000" {
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
				if v.Instance == "" {
					perf.NetUsage = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "virtualDisk.totalReadLatency.average":
				perf.DiskTotalReadLatency = float64(maxValue(v.Value)) //unit:"millisecond"
			case "virtualDisk.totalWriteLatency.average":
				perf.DiskTotalWriteLatency = float64(maxValue(v.Value)) //unit:"millisecond"
			case "virtualDisk.write.average":
				perf.DiskWrite = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
			case "virtualDisk.read.average":
				perf.DiskRead = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
			}
		}
	}
}

// CheckAndCorrectVMPerf проверяет и корректирует аномальные (отрицательные, NaN
// или превышающие разумные пределы) значения в снимке метрик виртуальной машины perf.
func CheckAndCorrectVMPerf(perf *VMPerf) {
	if perf == nil {
		return
	}

	//	perf.LockGetMetric()
	//	defer perf.UnLockGetMetric()

	if perf.CPUUsage > 100.0 {
		perf.CPUUsage = 100.0
	}
	if perf.MemUsage > 100.0 {
		perf.MemUsage = 100.0
	}
	if math.IsNaN(perf.CPUUsage) {
		perf.CPUUsage = 0.0
	}
	if math.IsNaN(perf.MemUsage) {
		perf.MemUsage = 0.0
	}
	if math.IsNaN(perf.DiskUsage) {
		perf.DiskUsage = 0.0
	}
	if math.IsNaN(perf.NetUsage) {
		perf.NetUsage = 0.0
	}
	if math.IsNaN(perf.DiskTotalReadLatency) {
		perf.DiskTotalReadLatency = 0.0
	}
	if math.IsNaN(perf.DiskTotalWriteLatency) {
		perf.DiskTotalWriteLatency = 0.0
	}
	if math.IsNaN(perf.DiskWrite) {
		perf.DiskWrite = 0.0
	}
	if math.IsNaN(perf.DiskRead) {
		perf.DiskRead = 0.0
	}
}

// SetVMMetric обновляет агрегированные показатели виртуальной машины vmP новым
// снимком метрик perf, используя cpuMhz для расчёта загрузки CPU в мегагерцах.
func SetVMMetric(vmP *VMPoint, perf VMPerf, cpuMhz float64) {
	if vmP == nil {
		return
	}
	vmP.LockObject()
	defer vmP.UnLockObject()
	vmP.CPUMhz = cpuMhz
	vmP.CPUBilling = vmP.CPUBilling + float64(perf.CPUUsage)*float64(cpuMhz)*float64(vmP.NumCores)/180.0 //утилизация процессора частота в мегагерцах умноженная процент утилизации и промежуток времени
	vmP.MemBilling = vmP.MemBilling + float64(perf.MemUsage)*vmP.MemorySizeGB*1024.0/180.0               // утилизация памяти мегабайт часах

	if vmP.CPUUsageMaximum < perf.CPUUsage {
		vmP.CPUUsageMaximum = perf.CPUUsage
		vmP.CPUUsageMaximumTime = perf.Start
	}
	if vmP.CPUUsageMinimum > perf.CPUUsage {
		vmP.CPUUsageMinimum = perf.CPUUsage
		vmP.CPUUsageMinimumTime = perf.Start
	}
	vmP.CPUUsageAverage = vmP.CPUUsageAverage + perf.CPUUsage

	if vmP.MemUsageMaximum < perf.MemUsage {
		vmP.MemUsageMaximum = perf.MemUsage
		vmP.MemUsageMaximumTime = perf.Start
	}
	if vmP.MemUsageMinimum > perf.MemUsage {
		vmP.MemUsageMinimum = perf.MemUsage
		vmP.MemUsageMinimumTime = perf.Start
	}
	vmP.MemUsageAverage = vmP.MemUsageAverage + perf.MemUsage

	if vmP.NetUsageMaximum < perf.NetUsage {
		vmP.NetUsageMaximum = perf.NetUsage
		vmP.NetUsageMaximumTime = perf.Start
	}
	if vmP.NetUsageMinimum > perf.NetUsage {
		vmP.NetUsageMinimum = perf.NetUsage
		vmP.NetUsageMinimumTime = perf.Start
	}
	vmP.NetUsageAverage = vmP.NetUsageAverage + perf.NetUsage

	if vmP.DiskUsageMaximum < perf.DiskUsage {
		vmP.DiskUsageMaximum = perf.DiskUsage
		vmP.DiskUsageMaximumTime = perf.Start
	}
	if vmP.DiskUsageMinimum > perf.DiskUsage {
		vmP.DiskUsageMinimum = perf.DiskUsage
		vmP.DiskUsageMinimumTime = perf.Start
	}
	vmP.DiskUsageAverage = vmP.DiskUsageAverage + perf.DiskUsage

	if vmP.DiskReadMaximum < perf.DiskRead {
		vmP.DiskReadMaximum = perf.DiskRead
		vmP.DiskReadMaximumTime = perf.Start
	}
	if vmP.DiskReadMinimum > perf.DiskRead {
		vmP.DiskReadMinimum = perf.DiskRead
		vmP.DiskReadMinimumTime = perf.Start
	}
	vmP.DiskReadAverage = vmP.DiskReadAverage + perf.DiskRead

	if vmP.DiskWriteMaximum < perf.DiskWrite {
		vmP.DiskWriteMaximum = perf.DiskWrite
		vmP.DiskWriteMaximumTime = perf.Start
	}
	if vmP.DiskWriteMinimum > perf.DiskWrite {
		vmP.DiskWriteMinimum = perf.DiskWrite
		vmP.DiskWriteMinimumTime = perf.Start
	}
	vmP.DiskWriteAverage = vmP.DiskWriteAverage + perf.DiskWrite

	if vmP.DiskTotalReadLatencyMaximum < perf.DiskTotalReadLatency {
		vmP.DiskTotalReadLatencyMaximum = perf.DiskTotalReadLatency
		vmP.DiskTotalReadLatencyMaximumTime = perf.Start
	}
	if vmP.DiskTotalReadLatencyMinimum > perf.DiskTotalReadLatency {
		vmP.DiskTotalReadLatencyMinimum = perf.DiskTotalReadLatency
		vmP.DiskTotalReadLatencyMinimumTime = perf.Start
	}
	vmP.DiskTotalReadLatencyAverage = vmP.DiskTotalReadLatencyAverage + perf.DiskTotalReadLatency

	if vmP.DiskTotalWriteLatencyMaximum < perf.DiskTotalWriteLatency {
		vmP.DiskTotalWriteLatencyMaximum = perf.DiskTotalWriteLatency
		vmP.DiskTotalWriteLatencyMaximumTime = perf.Start
	}
	if vmP.DiskTotalWriteLatencyMinimum > perf.DiskTotalWriteLatency {
		vmP.DiskTotalWriteLatencyMinimum = perf.DiskTotalWriteLatency
		vmP.DiskTotalWriteLatencyMinimumTime = perf.Start
	}
	vmP.DiskTotalWriteLatencyAverage = vmP.DiskTotalWriteLatencyAverage + perf.DiskTotalWriteLatency
}

// VmfsStrLunToUID преобразует часть канонического имени LUN VMFS (NAA) в UID/WWN
// в формате с разделителями-двоеточиями между байтами.
