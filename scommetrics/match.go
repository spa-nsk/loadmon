package scommetrics

import "strings"

// Классификация счётчиков производительности Windows (как их отдаёт SCOM:
// объект \ счётчик \ экземпляр) по колонкам models.ServerMetric.
//
// Имена - стандартные счётчики Windows Performance Monitor. Какие из них
// реально собираются правилами в конкретной инсталляции SCOM, зависит от
// management pack'ов и overrides - проверяется дампом
// (tools/scomdump -counters <имя компьютера>), см. kontext/13-scom-metrics.md.
// Списки ниже - единственное место, которое нужно поправить, если имена в
// инсталляции окажутся другими.

type fieldKind int

const (
	fieldUnknown fieldKind = iota
	fieldCPU
	fieldRAMPercent
	fieldRAMCommitted // запасной вариант для RAM, если нет fieldRAMPercent
	fieldDiskReadBps
	fieldDiskWriteBps
	fieldDiskReadSec
	fieldDiskWriteSec
	fieldDiskReadQueue
	fieldDiskWriteQueue
	fieldNetInBps
	fieldNetOutBps
)

type counterRule struct {
	objects  []string // нормализованные имена объектов (normName)
	counters []string // нормализованные имена счётчиков
	kind     fieldKind
}

var counterRules = []counterRule{
	{[]string{"processorinformation", "processor"}, []string{"processortime"}, fieldCPU},
	{[]string{"memory"}, []string{"percentmemoryused", "memoryused"}, fieldRAMPercent},
	{[]string{"memory"}, []string{"committedbytesinuse"}, fieldRAMCommitted},
	{[]string{"logicaldisk", "physicaldisk"}, []string{"diskreadbytessec"}, fieldDiskReadBps},
	{[]string{"logicaldisk", "physicaldisk"}, []string{"diskwritebytessec"}, fieldDiskWriteBps},
	{[]string{"logicaldisk", "physicaldisk"}, []string{"avgdisksecread"}, fieldDiskReadSec},
	{[]string{"logicaldisk", "physicaldisk"}, []string{"avgdisksecwrite"}, fieldDiskWriteSec},
	{[]string{"logicaldisk", "physicaldisk"}, []string{"avgdiskreadqueuelength"}, fieldDiskReadQueue},
	{[]string{"logicaldisk", "physicaldisk"}, []string{"avgdiskwritequeuelength"}, fieldDiskWriteQueue},
	{[]string{"networkadapter", "networkinterface"}, []string{"bytesreceivedsec"}, fieldNetInBps},
	{[]string{"networkadapter", "networkinterface"}, []string{"bytessentsec"}, fieldNetOutBps},
}

// classify определяет колонку для счётчика; fieldUnknown - не нужен.
func classify(objectName, counterName string) fieldKind {
	o, c := normName(objectName), normName(counterName)
	for _, r := range counterRules {
		if !inList(o, r.objects) {
			continue
		}
		if inList(c, r.counters) {
			return r.kind
		}
	}
	return fieldUnknown
}

func inList(s string, list []string) bool {
	for _, x := range list {
		if s == x {
			return true
		}
	}
	return false
}

// Экземпляры, которые не суммируются как отдельные диски/адаптеры - те же
// принципы, что в zabbixmetrics (diskSkip*/netSkip*).
var netSkipContains = []string{
	"loopback", "isatap", "teredo", "6to4", "pseudo-interface", "wan miniport",
	"bluetooth", "kernel debug", "npcap",
}

func isTotalInstance(inst string) bool {
	return strings.EqualFold(strings.TrimSpace(inst), "_total")
}

func skipNetInstance(inst string) bool {
	l := strings.ToLower(inst)
	for _, s := range netSkipContains {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

// skipDiskInstance - служебные тома без буквы (HarddiskVolumeN) не
// суммируются, как loop/ram в zabbixmetrics.
func skipDiskInstance(inst string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(inst)), "harddiskvolume")
}

// NeededCounters оставляет из списка счётчиков объекта только те, что
// нужны для server_metric (classify != fieldUnknown).
func NeededCounters(all []Counter) []Counter {
	var need []Counter
	for _, ct := range all {
		if classify(ct.ObjectName, ct.CounterName) != fieldUnknown {
			need = append(need, ct)
		}
	}
	return need
}
