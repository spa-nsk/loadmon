package vspheremetrics

// ReCalcVMPerfomance пересчитывает агрегированные показатели (min/max/average)
// виртуальной машины vmP с учётом нового снимка метрик perf.
func ReCalcVMPerfomance(vmP *VMPoint, perf VMPerf) {
	if vmP == nil {
		return
	}

	vmP.CPUBilling = vmP.CPUBilling + float64(perf.CPUUsage)*float64(vmP.CPUMhz)*float64(vmP.NumCores)/180.0 //утилизация процессора частота в мегагерцах умноженная процент утилизации и промежуток времени
	vmP.MemBilling = vmP.MemBilling + float64(perf.MemUsage)*vmP.MemorySizeGB*1024.0/180.0                   // утилизация памяти мегабайт часах

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

// CalcVMAveragePerfomance пересчитывает средние значения метрик виртуальной
// машины vmP по накопленному количеству замеров (PerfCount).
func CalcVMAveragePerfomance(vmP *VMPoint) {
	if vmP == nil {
		return
	}

	count := len(vmP.Perf)
	if count == 0 {
		return
	}

	vmP.CPUUsageAverage = vmP.CPUUsageAverage / float64(count)

	vmP.MemUsageAverage = vmP.MemUsageAverage / float64(count)

	vmP.NetUsageAverage = vmP.NetUsageAverage / float64(count)

	vmP.DiskUsageAverage = vmP.DiskUsageAverage / float64(count)

	vmP.DiskReadAverage = vmP.DiskReadAverage / float64(count)

	vmP.DiskWriteAverage = vmP.DiskWriteAverage / float64(count)

	vmP.DiskTotalReadLatencyAverage = vmP.DiskTotalReadLatencyAverage / float64(count)

	vmP.DiskTotalWriteLatencyAverage = vmP.DiskTotalWriteLatencyAverage / float64(count)
}

// ImportData заполняет поля VMPoint данными, полученными из govmomi (mo.VirtualMachine):
// конфигурацию, гостевую ОС, сеть, диски, использование хранилищ и файлы ВМ.
