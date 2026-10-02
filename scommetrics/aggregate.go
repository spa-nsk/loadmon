package scommetrics

import (
	"strings"
	"time"

	"loadmon/models"
)

// BytesPerMB - делитель байт -> МБ (SI), как zabbixmetrics.BytesPerMB.
const BytesPerMB = 1000.0 * 1000.0

type sample struct {
	kind     fieldKind
	object   string // нормализованное имя объекта (logicaldisk/physicaldisk/...)
	instance string
	value    float64
}

// buildRow сводит значения счётчиков одного компьютера в одну строку
// server_metric (по тем же принципам, что zabbixmetrics.buildRow). nil -
// ни одного пригодного значения.
func buildRow(at time.Time, hostID int64, server string, values []PerfValue) *models.ServerMetric {
	var samples []sample
	for _, v := range values {
		if !v.HasData {
			continue
		}
		k := classify(v.ObjectName, v.CounterName)
		if k == fieldUnknown {
			continue
		}
		samples = append(samples, sample{kind: k, object: normName(v.ObjectName), instance: v.InstanceName, value: v.Last})
	}
	if len(samples) == 0 {
		return nil
	}

	// LogicalDisk и PhysicalDisk описывают одну и ту же нагрузку - при
	// наличии обоих берётся только LogicalDisk, чтобы не удвоить суммы.
	samples = preferDiskObject(samples)

	row := &models.ServerMetric{
		Source:      models.MetricSourceSCOM,
		HostId:      hostID,
		CollectedAt: at,
		ServerName:  server,
	}
	if v, ok := pickTotalOrAvg(samples, fieldCPU); ok {
		row.CPUUtilPercent = clampPercent(v)
	}
	if v, ok := pickTotalOrAvg(samples, fieldRAMPercent); ok {
		row.RAMUtilPercent = clampPercent(v)
	} else if v, ok := pickTotalOrAvg(samples, fieldRAMCommitted); ok {
		row.RAMUtilPercent = clampPercent(v)
	}
	row.DiskReadMBps = sumInstances(samples, fieldDiskReadBps, skipDiskInstance) / BytesPerMB
	row.DiskWriteMBps = sumInstances(samples, fieldDiskWriteBps, skipDiskInstance) / BytesPerMB
	row.DiskReadLatencyMs = weightedLatency(samples, fieldDiskReadSec, fieldDiskReadBps) * 1000
	row.DiskWriteLatencyMs = weightedLatency(samples, fieldDiskWriteSec, fieldDiskWriteBps) * 1000
	row.DiskReadQueue = sumInstances(samples, fieldDiskReadQueue, skipDiskInstance)
	row.DiskWriteQueue = sumInstances(samples, fieldDiskWriteQueue, skipDiskInstance)
	row.NetRecvMBps = sumInstances(samples, fieldNetInBps, skipNetInstance) / BytesPerMB
	row.NetSentMBps = sumInstances(samples, fieldNetOutBps, skipNetInstance) / BytesPerMB
	return row
}

func preferDiskObject(in []sample) []sample {
	hasLogical := false
	for _, s := range in {
		if s.object == "logicaldisk" {
			hasLogical = true
			break
		}
	}
	if !hasLogical {
		return in
	}
	out := in[:0:0]
	for _, s := range in {
		if s.object == "physicaldisk" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func ofKind(in []sample, k fieldKind) []sample {
	var out []sample
	for _, s := range in {
		if s.kind == k {
			out = append(out, s)
		}
	}
	return out
}

// pickTotalOrAvg - значение экземпляра _Total, иначе среднее по экземплярам.
func pickTotalOrAvg(in []sample, k fieldKind) (float64, bool) {
	list := ofKind(in, k)
	if len(list) == 0 {
		return 0, false
	}
	sum := 0.0
	for _, s := range list {
		if isTotalInstance(s.instance) {
			return s.value, true
		}
		sum += s.value
	}
	return sum / float64(len(list)), true
}

// instances - значения по экземплярам без _Total и без skip; если других
// экземпляров нет - только _Total (на случай, когда правило собирает лишь его).
func instances(in []sample, k fieldKind, skip func(string) bool) []sample {
	list := ofKind(in, k)
	var parts, totals []sample
	for _, s := range list {
		switch {
		case isTotalInstance(s.instance):
			totals = append(totals, s)
		case skip != nil && skip(s.instance):
		default:
			parts = append(parts, s)
		}
	}
	if len(parts) > 0 {
		return parts
	}
	return totals
}

func sumInstances(in []sample, k fieldKind, skip func(string) bool) float64 {
	sum := 0.0
	for _, s := range instances(in, k, skip) {
		sum += s.value
	}
	return sum
}

// weightedLatency - задержка (сек), средняя по дискам, взвешенная по
// потоку того же экземпляра (как aggregateDiskAvg в zabbixmetrics); если
// потоков нет - простое среднее.
func weightedLatency(in []sample, latKind, rateKind fieldKind) float64 {
	lats := instances(in, latKind, skipDiskInstance)
	if len(lats) == 0 {
		return 0
	}
	rates := map[string]float64{}
	for _, s := range instances(in, rateKind, skipDiskInstance) {
		rates[strings.ToLower(s.instance)] = s.value
	}
	wSum, vSum, plain := 0.0, 0.0, 0.0
	for _, s := range lats {
		plain += s.value
		if w := rates[strings.ToLower(s.instance)]; w > 0 {
			wSum += w
			vSum += s.value * w
		}
	}
	if wSum > 0 {
		return vSum / wSum
	}
	return plain / float64(len(lats))
}

func clampPercent(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
