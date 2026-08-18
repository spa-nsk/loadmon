package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/simulator/vpx"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// loadmonPerfManager подменяет QueryPerf в vcsim: loadmon запрашивает Instance="*",
// а для агрегированных счётчиков ожидает Instance="". Также дополняются синтетические
// значения для счётчиков без sample data в govmomi/vpx.
type loadmonPerfManager struct {
	*simulator.PerformanceManager
}

func patchPerformanceManager(model *simulator.Model) {
	reg := model.Map()
	ref := *model.ServiceContent.PerfManager
	orig, ok := reg.Get(ref).(*simulator.PerformanceManager)
	if !ok || orig == nil {
		log.Printf("perf: PerformanceManager не найден")
		return
	}
	wrapped := &loadmonPerfManager{PerformanceManager: orig}
	reg.Put(wrapped)
	log.Printf("perf: симуляция метрик VM/Host для loadmon (%T)", reg.Get(ref))
}

func (p *loadmonPerfManager) QueryPerf(ctx *simulator.Context, req *types.QueryPerf) soap.HasFault {
	body := new(methods.QueryPerfBody)
	body.Res = new(types.QueryPerfResponse)
	body.Res.Returnval = make([]types.BasePerfEntityMetricBase, len(req.QuerySpec))

	for i, qs := range req.QuerySpec {
		metricData, ok := vpx.MetricData[qs.Entity.Type]
		if !ok {
			body.Fault_ = simulator.Fault("", &types.InvalidArgument{InvalidProperty: "Entity"})
			return body
		}

		spec := qs
		spec.MetricId = normalizeMetricIDs(qs.MetricId)

		start, end := perfWindow(&spec)
		interval := spec.IntervalId
		if interval <= 0 {
			interval = 20
		}
		n := sampleCount(start, end, interval, spec.MaxSample)

		metrics := &types.PerfEntityMetric{}
		metrics.Entity = spec.Entity
		metrics.SampleInfo = make([]types.PerfSampleInfo, n)
		for tick := int32(0); tick < n; tick++ {
			metrics.SampleInfo[tick] = types.PerfSampleInfo{
				Timestamp: end.Add(-time.Duration(interval*tick) * time.Second),
				Interval:  interval,
			}
		}

		metrics.Value = make([]types.BasePerfMetricSeries, len(spec.MetricId))
		offset := start.Unix() / int64(interval)

		for j, mid := range spec.MetricId {
			series := &types.PerfMetricIntSeries{Value: make([]int64, n)}
			series.Id = mid
			points := metricData[mid.CounterId]
			for tick := int32(0); tick < n; tick++ {
				val := int64(0)
				if len(points) > 0 {
					val = points[(offset+int64(tick))%int64(len(points))]
					if scale := val / 5; scale > 0 {
						val += int64(rand.NormFloat64() * float64(scale))
						if val < 0 {
							val = 0
						}
					}
				}
				if val == 0 {
					val = synthValue(mid, spec.Entity, int64(tick))
				}
				series.Value[tick] = val
			}
			metrics.Value[j] = series
		}

		if spec.Format == string(types.PerfFormatCsv) {
			body.Res.Returnval[i] = perfEntityMetricCSV(metrics)
		} else {
			body.Res.Returnval[i] = metrics
		}
	}
	return body
}

func normalizeMetricIDs(ids []types.PerfMetricId) []types.PerfMetricId {
	out := make([]types.PerfMetricId, len(ids))
	for i, mid := range ids {
		out[i] = mid
		if mid.Instance == "*" {
			out[i].Instance = ""
		}
	}
	return out
}

func perfWindow(qs *types.PerfQuerySpec) (start, end time.Time) {
	if qs.StartTime == nil {
		start = time.Now().Add(-365 * 24 * time.Hour)
	} else {
		start = *qs.StartTime
	}
	if qs.EndTime == nil {
		end = time.Now()
	} else {
		end = *qs.EndTime
	}
	return start, end
}

func sampleCount(start, end time.Time, interval, maxSample int32) int32 {
	n := int32(1) + int32(end.Sub(start).Seconds())/interval
	if maxSample > 0 && n > maxSample {
		n = maxSample
	}
	if n <= 0 {
		if maxSample > 0 {
			return maxSample
		}
		return 2
	}
	return n
}

func perfEntityMetricCSV(m *types.PerfEntityMetric) *types.PerfEntityMetricCSV {
	out := &types.PerfEntityMetricCSV{}
	out.Entity = m.Entity
	vals := make([]string, len(m.SampleInfo)*2)
	idx := 0
	for _, s := range m.SampleInfo {
		vals[idx] = fmt.Sprintf("%d", s.Interval)
		idx++
		vals[idx] = s.Timestamp.Format(time.RFC3339)
		idx++
	}
	out.SampleInfoCSV = strings.Join(vals, ",")
	out.Value = make([]types.PerfMetricSeriesCSV, len(m.Value))
	for j, raw := range m.Value {
		s := raw.(*types.PerfMetricIntSeries)
		csv := types.PerfMetricSeriesCSV{}
		csv.Id = s.Id
		parts := make([]string, len(s.Value))
		for k, v := range s.Value {
			parts[k] = fmt.Sprintf("%d", v)
		}
		csv.Value = strings.Join(parts, ",")
		out.Value[j] = csv
	}
	return out
}

func synthValue(id types.PerfMetricId, entity types.ManagedObjectReference, tick int64) int64 {
	name := counterName(id.CounterId)
	base, spread := valueRange(name, entity.Type)
	seed := hashSeed(entity.Value, id.CounterId, id.Instance, tick)
	rng := rand.New(rand.NewSource(seed))
	v := base + int64(rng.Float64()*float64(spread))
	v += int64(50 * math.Sin(float64(seed)/3.7))
	if v < 1 {
		v = 1
	}
	return v
}

func counterName(counterID int32) string {
	if name, ok := perfCounterSpecs[counterID]; ok {
		return name
	}
	return fmt.Sprintf("id:%d", counterID)
}

var perfCounterSpecs = map[int32]string{
	2:   "cpu.usage.average",
	6:   "cpu.usage.average",
	12:  "cpu.usage.average",
	14:  "mem.usage.average",
	24:  "cpu.utilization.average",
	130: "disk.read.average",
	131: "disk.read.average",
	173: "virtualDisk.read.average",
	174: "virtualDisk.write.average",
	175: "virtualDisk.totalReadLatency.average",
	176: "virtualDisk.totalWriteLatency.average",
	447: "disk.deviceReadLatency.average",
	448: "disk.deviceWriteLatency.average",
}

func valueRange(name, entityType string) (base, spread int64) {
	switch {
	case strings.HasPrefix(name, "cpu.usage"), strings.HasPrefix(name, "cpu.utilization"):
		return 1500, 5500
	case strings.HasPrefix(name, "mem.usage"):
		if entityType == "HostSystem" {
			return 5000, 9000
		}
		return 2500, 6500
	case strings.Contains(name, "Latency"):
		return 8, 35
	case strings.Contains(name, "read"), strings.Contains(name, "write"), strings.Contains(name, "usage.average"):
		if strings.HasPrefix(name, "net") {
			return 200, 1800
		}
		return 100, 3500
	default:
		return 500, 2500
	}
}

func hashSeed(parts ...interface{}) int64 {
	h := fnv.New64a()
	for _, p := range parts {
		fmt.Fprint(h, p)
		fmt.Fprint(h, "|")
	}
	return int64(h.Sum64())
}
