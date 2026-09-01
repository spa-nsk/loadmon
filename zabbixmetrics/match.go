package zabbixmetrics

import (
	"regexp"
	"strings"
)

type fieldKind int

const (
	fieldUnknown fieldKind = iota
	fieldCPU
	fieldCPUIdle
	fieldRAM
	fieldRAMPavailable
	fieldRAMAvailable
	fieldRAMTotal
	fieldRAMUsed
	fieldDiskReadMBps
	fieldDiskWriteMBps
	fieldDiskReadLatency
	fieldDiskWriteLatency
	fieldDiskReadQueue
	fieldDiskWriteQueue
	fieldDiskQueue
	fieldNetIn
	fieldNetOut
)

var linuxPartitionRe = regexp.MustCompile(`(?i)^(?:(?:sd|vd|hd|xvd)[a-z]\d+|nvme\d+n\d+p\d+|mmcblk\d+p\d+)$`)
var loIfaceRe = regexp.MustCompile(`(?i)^lo\d*$`)

func classifyItem(it zabbixItem) fieldKind {
	key := it.Key
	if k := classifyCPUKey(key); k != fieldUnknown {
		return k
	}
	if k := classifyRAMKey(key); k != fieldUnknown {
		return k
	}
	if k := classifyDiskKey(key); k != fieldUnknown {
		return k
	}
	if k := classifyNetKey(key, it.Name); k != fieldUnknown {
		return k
	}
	return classifyByItemKeys(it)
}

func classifyCPUKey(key string) fieldKind {
	switch key {
	case "system.cpu.util", "system.cpu.util[]":
		return fieldCPU
	}
	if strings.HasPrefix(key, "system.cpu.util[") && strings.HasSuffix(key, "]") {
		parts := splitCSV(extractBracketInner(key))
		cpu, typ := "", ""
		if len(parts) > 0 {
			cpu = strings.Trim(parts[0], `"' `)
		}
		if len(parts) > 1 {
			typ = strings.ToLower(strings.Trim(parts[1], `"' `))
		}
		if cpu != "" && !strings.EqualFold(cpu, "all") {
			return fieldUnknown
		}
		switch typ {
		case "", "avg":
			return fieldCPU
		case "idle":
			return fieldCPUIdle
		default:
			return fieldUnknown
		}
	}
	obj, inst, counter, ok := parsePerfCounter(key)
	if !ok {
		return fieldUnknown
	}
	if isProcessorObject(obj) && isTotalInstance(inst) && isProcessorTime(counter) {
		return fieldCPU
	}
	return fieldUnknown
}

func classifyRAMKey(key string) fieldKind {
	switch key {
	case "vm.memory.util", "vm.memory.utilization", "vm.memory.util[]", "vm.memory.utilization[]":
		return fieldRAM
	}
	if !strings.HasPrefix(key, "vm.memory.size[") || !strings.HasSuffix(key, "]") {
		return fieldUnknown
	}
	param := strings.ToLower(strings.Trim(extractBracketInner(key), `"' `))
	switch param {
	case "pused":
		return fieldRAM
	case "pavailable", "pfree":
		return fieldRAMPavailable
	case "available", "free":
		return fieldRAMAvailable
	case "total":
		return fieldRAMTotal
	case "used":
		return fieldRAMUsed
	default:
		return fieldUnknown
	}
}

func classifyDiskKey(key string) fieldKind {
	if skipDiskDevice(key) {
		return fieldUnknown
	}
	if strings.HasPrefix(key, "vfs.dev.") {
		return classifyVfsDev(key)
	}
	obj, inst, counter, ok := parsePerfCounter(key)
	if !ok || !isPhysicalDiskObject(obj) {
		return fieldUnknown
	}
	if skipDiskInstance(inst) {
		return fieldUnknown
	}
	return classifyDiskCounter(counter)
}

func classifyVfsDev(key string) fieldKind {
	switch {
	case strings.HasPrefix(key, "vfs.dev.read.await"):
		return fieldDiskReadLatency
	case strings.HasPrefix(key, "vfs.dev.write.await"):
		return fieldDiskWriteLatency
	case strings.HasPrefix(key, "vfs.dev.queue_size") || strings.HasPrefix(key, "vfs.dev.queue.size"):
		return fieldDiskQueue
	case strings.Contains(key, ".read.bytes") || strings.HasPrefix(key, "vfs.dev.read.bytes"):
		return fieldDiskReadMBps
	case strings.Contains(key, ".write.bytes") || strings.HasPrefix(key, "vfs.dev.write.bytes"):
		return fieldDiskWriteMBps
	case strings.HasPrefix(key, "vfs.dev.read[") || key == "vfs.dev.read":
		if vfsDevIsBytesRate(key) {
			return fieldDiskReadMBps
		}
		return fieldUnknown
	case strings.HasPrefix(key, "vfs.dev.write[") || key == "vfs.dev.write":
		if vfsDevIsBytesRate(key) {
			return fieldDiskWriteMBps
		}
		return fieldUnknown
	default:
		return fieldUnknown
	}
}

func classifyDiskCounter(counter string) fieldKind {
	c := strings.ToLower(strings.TrimSpace(counter))
	switch c {
	case "disk read bytes/sec", "220":
		return fieldDiskReadMBps
	case "disk write bytes/sec", "222":
		return fieldDiskWriteMBps
	case "avg. disk sec/read", "208":
		return fieldDiskReadLatency
	case "avg. disk sec/write", "210":
		return fieldDiskWriteLatency
	case "avg. disk read queue length", "1402":
		return fieldDiskReadQueue
	case "avg. disk write queue length", "1404":
		return fieldDiskWriteQueue
	case "current disk queue length", "avg. disk queue length", "198", "1400":
		return fieldDiskQueue
	default:
		return fieldUnknown
	}
}

func classifyNetKey(key, name string) fieldKind {
	if skipInterface(key, name) {
		return fieldUnknown
	}
	if kind, ok := classifyNetIfKey(key); ok {
		return kind
	}
	obj, inst, counter, ok := parsePerfCounter(key)
	if !ok || !isNetObject(obj) {
		return fieldUnknown
	}
	if skipInterfaceName(inst) || skipInterfaceName(name) {
		return fieldUnknown
	}
	switch strings.ToLower(strings.TrimSpace(counter)) {
	case "bytes received/sec", "264":
		return fieldNetIn
	case "bytes sent/sec", "506":
		return fieldNetOut
	default:
		return fieldUnknown
	}
}

func classifyNetIfKey(key string) (fieldKind, bool) {
	var kind fieldKind
	switch {
	case strings.HasPrefix(key, "net.if.in["):
		kind = fieldNetIn
	case strings.HasPrefix(key, "net.if.out["):
		kind = fieldNetOut
	default:
		return fieldUnknown, false
	}
	inner := extractBracketInner(key)
	parts := splitCSV(inner)
	if len(parts) >= 2 {
		param := strings.ToLower(strings.Trim(parts[1], `"' `))
		if param != "" && param != "bytes" {
			return fieldUnknown, false
		}
	}
	return kind, true
}

func classifyByItemKeys(it zabbixItem) fieldKind {
	key := it.Key
	switch {
	case matchBare(key, ItemKeys.CPUUtil):
		return fieldCPU
	case matchBare(key, ItemKeys.RAMUtil):
		return fieldRAM
	case matchPrefixOrContains(key, ItemKeys.DiskReadMBps):
		return fieldDiskReadMBps
	case matchPrefixOrContains(key, ItemKeys.DiskWriteMBps):
		return fieldDiskWriteMBps
	case matchPrefixOrContains(key, ItemKeys.DiskReadLatency):
		return fieldDiskReadLatency
	case matchPrefixOrContains(key, ItemKeys.DiskWriteLatency):
		return fieldDiskWriteLatency
	case matchPrefixOrContains(key, ItemKeys.DiskReadQueue):
		return fieldDiskReadQueue
	case matchPrefixOrContains(key, ItemKeys.DiskWriteQueue):
		return fieldDiskWriteQueue
	case matchPrefixOrContains(key, ItemKeys.DiskQueue):
		return fieldDiskQueue
	case matchPrefixOrContains(key, ItemKeys.NetIn) && !skipInterface(key, it.Name):
		return fieldNetIn
	case matchPrefixOrContains(key, ItemKeys.NetOut) && !skipInterface(key, it.Name):
		return fieldNetOut
	default:
		return fieldUnknown
	}
}

func matchBare(key string, patterns []string) bool {
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if key == p {
			return true
		}
		if strings.HasPrefix(key, p+"[") && strings.HasSuffix(key, "]") {
			inner := strings.TrimSpace(key[len(p)+1 : len(key)-1])
			if inner == "" {
				return true
			}
		}
	}
	return false
}

func matchPrefixOrContains(key string, patterns []string) bool {
	for _, p := range patterns {
		if p == "" {
			continue
		}
		if key == p || strings.HasPrefix(key, p+"[") || strings.HasPrefix(key, p) {
			return true
		}
		if strings.Contains(p, " ") || strings.Contains(p, `\`) {
			if strings.Contains(key, p) {
				return true
			}
		}
	}
	return false
}

func vfsDevIsBytesRate(key string) bool {
	if strings.Contains(key, ".rate") || strings.Contains(key, ".await") || strings.Contains(key, ".time") {
		return false
	}
	_, param := splitDevParam(extractBracketInner(key))
	switch strings.ToLower(param) {
	case "sps", "sectors", "bytes", "bps":
		return true
	default:
		return false
	}
}

func parsePerfCounter(key string) (obj, inst, counter string, ok bool) {
	lk := strings.ToLower(key)
	if !strings.HasPrefix(lk, "perf_counter[") && !strings.HasPrefix(lk, "perf_counter_en[") {
		return "", "", "", false
	}
	inner := extractBracketInner(key)
	if inner == "" {
		return "", "", "", false
	}
	parts := splitCSV(inner)
	if len(parts) == 0 {
		return "", "", "", false
	}
	path := strings.Trim(parts[0], `"' `)
	path = strings.TrimPrefix(path, `\`)
	i := strings.IndexByte(path, '(')
	j := strings.IndexByte(path, ')')
	if i >= 0 && j > i {
		obj = path[:i]
		inst = path[i+1 : j]
		counter = strings.TrimPrefix(path[j+1:], `\`)
		return obj, inst, counter, obj != "" && counter != ""
	}
	k := strings.IndexByte(path, '\\')
	if k >= 0 {
		return path[:k], "", path[k+1:], true
	}
	return "", "", "", false
}

func isProcessorObject(obj string) bool {
	o := strings.ToLower(strings.TrimSpace(obj))
	return o == "processor" || o == "processor information" || o == "238" || o == "1848"
}

func isProcessorTime(counter string) bool {
	c := strings.ToLower(strings.TrimSpace(counter))
	return c == "% processor time" || c == "6"
}

func isPhysicalDiskObject(obj string) bool {
	o := strings.ToLower(strings.TrimSpace(obj))
	return o == "physicaldisk" || o == "234"
}

func isNetObject(obj string) bool {
	o := strings.ToLower(strings.TrimSpace(obj))
	return o == "network interface" || o == "510"
}

func isTotalInstance(inst string) bool {
	s := strings.ToLower(strings.TrimSpace(inst))
	return s == "_total" || s == "total"
}

func diskInstance(key string) string {
	if obj, inst, _, ok := parsePerfCounter(key); ok && isPhysicalDiskObject(obj) {
		return inst
	}
	if strings.HasPrefix(key, "vfs.dev.") {
		return extractBracketArg(key)
	}
	return extractBracketArg(key)
}

func skipDiskDevice(key string) bool {
	return skipDiskInstance(diskInstance(key))
}

func skipDiskInstance(inst string) bool {
	n := strings.ToLower(strings.Trim(inst, `"' `))
	if n == "" || isTotalInstance(n) {
		return false
	}
	for _, s := range diskSkipExact {
		if n == s {
			return true
		}
	}
	for _, s := range diskSkipPrefix {
		if strings.HasPrefix(n, s) {
			if len(n) == len(s) {
				return true
			}
			rest := n[len(s):]
			if rest == "" || isDigitStart(rest) {
				return true
			}
		}
	}
	return false
}

func skipInterface(key, name string) bool {
	if skipInterfaceName(name) {
		return true
	}
	if obj, inst, _, ok := parsePerfCounter(key); ok && isNetObject(obj) {
		return skipInterfaceName(inst)
	}
	if strings.HasPrefix(key, "net.if.") {
		return skipInterfaceName(extractBracketArg(key))
	}
	return false
}

func skipInterfaceName(name string) bool {
	n := strings.ToLower(strings.Trim(name, `"' `))
	if n == "" {
		return false
	}
	if loIfaceRe.MatchString(n) {
		return true
	}
	for _, s := range netSkipExact {
		if n == s {
			return true
		}
	}
	for _, s := range netSkipPrefix {
		if strings.HasPrefix(n, s) {
			return true
		}
	}
	for _, s := range netSkipContains {
		if strings.Contains(n, s) {
			return true
		}
	}
	return false
}

func isLinuxPartition(dev string) bool {
	return linuxPartitionRe.MatchString(strings.Trim(dev, `"' `))
}

func isDigitStart(s string) bool {
	if s == "" {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}

func extractBracketInner(key string) string {
	i := strings.IndexByte(key, '[')
	if i < 0 || !strings.HasSuffix(key, "]") {
		return ""
	}
	return key[i+1 : len(key)-1]
}

func extractBracketArg(key string) string {
	inner := extractBracketInner(key)
	if inner == "" {
		return ""
	}
	dev, _ := splitDevParam(inner)
	return strings.Trim(dev, `"' `)
}

func splitDevParam(inner string) (dev, param string) {
	parts := splitCSV(inner)
	if len(parts) == 0 {
		return "", ""
	}
	dev = strings.Trim(parts[0], `"' `)
	if len(parts) > 1 {
		param = strings.Trim(parts[1], `"' `)
	}
	return dev, param
}

func splitCSV(s string) []string {
	var parts []string
	var b strings.Builder
	inQuotes := false
	quote := rune(0)
	for _, r := range s {
		switch {
		case (r == '"' || r == '\'') && !inQuotes:
			inQuotes = true
			quote = r
		case r == quote && inQuotes:
			inQuotes = false
		case r == ',' && !inQuotes:
			parts = append(parts, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 || strings.HasSuffix(s, ",") {
		parts = append(parts, strings.TrimSpace(b.String()))
	}
	return parts
}

func vfsParam(key string) string {
	_, param := splitDevParam(extractBracketInner(key))
	return param
}

func toMBps(value float64, units, key string) float64 {
	if strings.EqualFold(vfsParam(key), "sps") || strings.EqualFold(vfsParam(key), "sectors") {
		return value * 512 / BytesPerMB
	}
	u := normalizeUnit(units)
	switch u {
	case "bps":
		return value / 8 / BytesPerMB
	case "kbps":
		return value * 1000 / 8 / BytesPerMB
	case "mbps":
		return value * 1_000_000 / 8 / BytesPerMB
	case "gbps":
		return value * 1_000_000_000 / 8 / BytesPerMB
	case "b", "bps_bytes":
		return value / BytesPerMB
	case "kbps_bytes":
		return value * 1000 / BytesPerMB
	case "mbps_bytes":
		return value
	case "gbps_bytes":
		return value * 1000
	default:
		if looksLikeBitsRate(key, units) {
			return value / 8 / BytesPerMB
		}
		return value / BytesPerMB
	}
}

func looksLikeBitsRate(key, units string) bool {
	if strings.Contains(strings.ToLower(units), "bit") {
		return true
	}
	return strings.Contains(strings.ToLower(key), "bits ")
}

func toMs(value float64, units, key string) float64 {
	u := normalizeUnit(units)
	if u == "" && (strings.Contains(key, "sec/Read") || strings.Contains(key, "sec/Write") || strings.Contains(key, "sec/Transfer")) {
		return value * 1000
	}
	switch u {
	case "s":
		return value * 1000
	case "us", "µs":
		return value / 1000
	default:
		return value
	}
}

func normalizeUnit(units string) string {
	u := strings.TrimSpace(units)
	u = strings.TrimPrefix(u, "!")
	switch u {
	case "Bps", "B":
		return "bps_bytes"
	case "KBps", "kBps", "KB":
		return "kbps_bytes"
	case "MBps", "MB":
		return "mbps_bytes"
	case "GBps", "GB":
		return "gbps_bytes"
	case "bps", "Kbps", "kbps", "Mbps", "mbps", "Gbps", "gbps", "s", "ms", "us", "µs", "%":
		return strings.ToLower(u)
	default:
		return strings.ToLower(u)
	}
}
