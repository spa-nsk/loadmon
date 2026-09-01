package zabbixmetrics

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"loadmon/models"
)

type metricVal struct {
	kind  fieldKind
	item  zabbixItem
	value float64
}

// Collect снимает метрики группы HostGroupID за текущую минуту и пишет их в БД.
func Collect(ctx context.Context) error {
	if Endpoint == "" {
		return fmt.Errorf("zabbixmetrics: Endpoint is empty")
	}
	if Token == "" {
		return fmt.Errorf("zabbixmetrics: Token is empty")
	}
	if HostGroupID <= 0 {
		return fmt.Errorf("zabbixmetrics: HostGroupID is not set")
	}

	now := time.Now().Truncate(time.Minute)
	client := newZabbixClient()

	hosts, err := client.hostsByGroupID(ctx, HostGroupID)
	if err != nil {
		return err
	}
	if len(hosts) == 0 {
		log.Printf("zabbixmetrics: groupid %d has no enabled hosts", HostGroupID)
		return nil
	}

	hostIDs := make([]string, 0, len(hosts))
	byID := make(map[string]zabbixHost, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.HostID)
		byID[h.HostID] = h
	}

	items, err := client.itemsForHosts(ctx, hostIDs)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		log.Printf("zabbixmetrics: no matching items for groupid %d", HostGroupID)
		return nil
	}

	values, err := client.lastValues(ctx, items, now.Add(-HistoryWindow), now.Add(time.Minute))
	if err != nil {
		return err
	}

	itemsByHost := map[string][]zabbixItem{}
	for _, it := range items {
		itemsByHost[it.HostID] = append(itemsByHost[it.HostID], it)
	}

	resolver, err := models.NewZabbixHostResolver()
	if err != nil {
		return err
	}

	rows := make([]*models.ServerMetric, 0, len(hosts))
	for hostID, host := range byID {
		dbHostID, err := resolver.EnsureID(host.Host, host.Name)
		if err != nil {
			log.Printf("zabbixmetrics: skip host %s: %v", host.displayName(), err)
			continue
		}
		row := buildRow(now, dbHostID, host.displayName(), itemsByHost[hostID], values)
		if row == nil {
			continue
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		log.Printf("zabbixmetrics: no values in the last %s", HistoryWindow)
		return nil
	}
	if err := models.ReplaceMetrics(ctx, rows); err != nil {
		return fmt.Errorf("zabbixmetrics: store metrics: %w", err)
	}
	log.Printf("zabbixmetrics: stored %d host snapshots at %s", len(rows), now.Format(time.RFC3339))
	return nil
}

// Run собирает метрики сразу и затем каждые Interval, пока ctx не отменён.
func Run(ctx context.Context) error {
	if Interval <= 0 {
		Interval = time.Minute
	}
	if err := Collect(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := Collect(ctx); err != nil {
				log.Printf("zabbixmetrics: collect: %v", err)
			}
		}
	}
}

func buildRow(at time.Time, hostID int64, server string, items []zabbixItem, values map[string]float64) *models.ServerMetric {
	if len(items) == 0 {
		return nil
	}

	picked := make([]metricVal, 0, len(items))
	for _, it := range items {
		v, ok := values[it.ItemID]
		if !ok {
			continue
		}
		kind := classifyItem(it)
		if kind == fieldUnknown {
			continue
		}
		picked = append(picked, metricVal{kind: kind, item: it, value: v})
	}
	if len(picked) == 0 {
		return nil
	}

	cpu := pickCPU(picked)
	ram := pickRAM(picked)
	diskRead := aggregateDiskRate(picked, fieldDiskReadMBps)
	diskWrite := aggregateDiskRate(picked, fieldDiskWriteMBps)
	latRead := aggregateDiskAvg(picked, fieldDiskReadLatency, fieldDiskReadMBps)
	latWrite := aggregateDiskAvg(picked, fieldDiskWriteLatency, fieldDiskWriteMBps)
	qRead, qWrite := aggregateDiskQueues(picked)
	netIn := aggregateNet(picked, fieldNetIn)
	netOut := aggregateNet(picked, fieldNetOut)

	row := &models.ServerMetric{
		HostId:             hostID,
		CollectedAt:        at,
		ServerName:         server,
		DiskReadMBps:       diskRead,
		DiskWriteMBps:      diskWrite,
		DiskReadLatencyMs:  latRead,
		DiskWriteLatencyMs: latWrite,
		DiskReadQueue:      qRead,
		DiskWriteQueue:     qWrite,
		NetRecvMBps:        netIn,
		NetSentMBps:        netOut,
	}
	if cpu != nil {
		row.CPUUtilPercent = clampPercent(*cpu)
	}
	if ram != nil {
		row.RAMUtilPercent = clampPercent(*ram)
	}
	return row
}

func pickCPU(picked []metricVal) *float64 {
	var best *float64
	priority := 100
	for _, p := range picked {
		switch p.kind {
		case fieldCPU:
			pr := 1
			if p.item.Key == "system.cpu.util" || p.item.Key == "system.cpu.util[]" {
				pr = 0
			}
			if pr < priority {
				priority = pr
				v := p.value
				best = &v
			}
		case fieldCPUIdle:
			if priority > 2 {
				v := 100 - p.value
				priority = 2
				best = &v
			}
		}
	}
	return best
}

func pickRAM(picked []metricVal) *float64 {
	var best *float64
	priority := 100
	var used, total, avail *float64
	for _, p := range picked {
		v := p.value
		switch p.kind {
		case fieldRAM:
			pr := 1
			if p.item.Key == "vm.memory.util" || p.item.Key == "vm.memory.utilization" {
				pr = 0
			}
			if pr < priority {
				priority = pr
				best = &v
			}
		case fieldRAMPavailable:
			if priority > 2 {
				x := 100 - v
				priority = 2
				best = &x
			}
		case fieldRAMUsed:
			used = &v
		case fieldRAMTotal:
			total = &v
		case fieldRAMAvailable:
			avail = &v
		}
	}
	if best != nil {
		return best
	}
	if used != nil && total != nil && *total > 0 {
		x := *used / *total * 100
		return &x
	}
	if avail != nil && total != nil && *total > 0 {
		x := (*total - *avail) / *total * 100
		return &x
	}
	return nil
}

func aggregateDiskRate(picked []metricVal, kind fieldKind) float64 {
	items := selectDiskItems(picked, kind)
	var sum float64
	seen := map[string]struct{}{}
	for _, p := range items {
		id := diskDedupeKey(p)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		sum += toMBps(p.value, p.item.Units, p.item.Key)
	}
	return sum
}

func aggregateDiskAvg(picked []metricVal, kind, rateKind fieldKind) float64 {
	items := selectDiskItems(picked, kind)
	if len(items) == 0 {
		return 0
	}
	rates := selectDiskItems(picked, rateKind)
	rateByDev := map[string]float64{}
	for _, p := range rates {
		dev := diskDedupeKey(p)
		if _, ok := rateByDev[dev]; ok {
			continue
		}
		rateByDev[dev] = toMBps(p.value, p.item.Units, p.item.Key)
	}
	var sum, wsum float64
	seen := map[string]struct{}{}
	for _, p := range items {
		id := diskDedupeKey(p)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		w := 1.0
		if r := rateByDev[id]; r > 0 {
			w = r
		}
		sum += toMs(p.value, p.item.Units, p.item.Key) * w
		wsum += w
	}
	if wsum == 0 {
		return 0
	}
	return sum / wsum
}

func aggregateDiskQueues(picked []metricVal) (read, write float64) {
	rq := uniqueDiskSum(selectDiskItems(picked, fieldDiskReadQueue), false)
	wq := uniqueDiskSum(selectDiskItems(picked, fieldDiskWriteQueue), false)
	if len(selectDiskItems(picked, fieldDiskReadQueue)) > 0 || len(selectDiskItems(picked, fieldDiskWriteQueue)) > 0 {
		return rq, wq
	}
	all := uniqueDiskSum(selectDiskItems(picked, fieldDiskQueue), false)
	return all, all
}

func uniqueDiskSum(items []metricVal, asMBps bool) float64 {
	var sum float64
	seen := map[string]struct{}{}
	for _, p := range items {
		id := diskDedupeKey(p)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if asMBps {
			sum += toMBps(p.value, p.item.Units, p.item.Key)
		} else {
			sum += p.value
		}
	}
	return sum
}

func aggregateNet(picked []metricVal, kind fieldKind) float64 {
	var sum float64
	seen := map[string]struct{}{}
	for _, p := range picked {
		if p.kind != kind {
			continue
		}
		id := netDedupeKey(p)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		sum += toMBps(p.value, p.item.Units, p.item.Key)
	}
	return sum
}

func selectDiskItems(picked []metricVal, kind fieldKind) []metricVal {
	var all []metricVal
	for _, p := range picked {
		if p.kind == kind {
			all = append(all, p)
		}
	}
	if len(all) == 0 {
		return all
	}
	var totals []metricVal
	for _, p := range all {
		if isTotalInstance(diskInstance(p.item.Key)) {
			totals = append(totals, p)
		}
	}
	if len(totals) > 0 {
		return totals
	}
	hasWhole := false
	for _, p := range all {
		if !isLinuxPartition(diskInstance(p.item.Key)) {
			hasWhole = true
			break
		}
	}
	if !hasWhole {
		return all
	}
	out := all[:0]
	for _, p := range all {
		if !isLinuxPartition(diskInstance(p.item.Key)) {
			out = append(out, p)
		}
	}
	return out
}

func diskDedupeKey(p metricVal) string {
	inst := strings.ToLower(diskInstance(p.item.Key))
	if inst != "" {
		return inst
	}
	return p.item.ItemID
}

func netDedupeKey(p metricVal) string {
	if name := extractBracketArg(p.item.Key); name != "" {
		return strings.ToLower(name)
	}
	if _, inst, _, ok := parsePerfCounter(p.item.Key); ok && inst != "" {
		return strings.ToLower(inst)
	}
	return p.item.ItemID
}

func clampPercent(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
