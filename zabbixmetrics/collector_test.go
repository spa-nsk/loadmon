package zabbixmetrics

import (
	"math"
	"testing"
	"time"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestClassifyMixedOSKeys(t *testing.T) {
	cases := []struct {
		key, name string
		want      fieldKind
	}{
		{"system.cpu.util", "", fieldCPU},
		{"system.cpu.util[]", "", fieldCPU},
		{"system.cpu.util[all,avg,avg1]", "", fieldCPU},
		{"system.cpu.util[,idle]", "", fieldCPUIdle},
		{"system.cpu.util[all,idle,avg1]", "", fieldCPUIdle},
		{"system.cpu.util[,user]", "", fieldUnknown},
		{"system.cpu.util[0]", "", fieldUnknown},
		{"system.cpu.util[0,idle]", "", fieldUnknown},
		{`perf_counter_en["\Processor Information(_total)\% Processor Time"]`, "", fieldCPU},
		{`perf_counter["\Processor(_Total)\% Processor Time"]`, "", fieldCPU},
		{`perf_counter["\Processor(_Total)\% User Time"]`, "", fieldUnknown},

		{"vm.memory.util", "", fieldRAM},
		{"vm.memory.utilization", "", fieldRAM},
		{"vm.memory.size[pused]", "", fieldRAM},
		{"vm.memory.size[pavailable]", "", fieldRAMPavailable},
		{"vm.memory.size[available]", "", fieldRAMAvailable},
		{"vm.memory.size[total]", "", fieldRAMTotal},
		{"vm.memory.size[used]", "", fieldRAMUsed},

		{"vfs.dev.read.await[sda]", "", fieldDiskReadLatency},
		{"vfs.dev.write.await[sda]", "", fieldDiskWriteLatency},
		{"vfs.dev.queue_size[sda]", "", fieldDiskQueue},
		{"vfs.dev.read.rate[sda]", "", fieldUnknown},
		{"vfs.dev.read[sda,sps]", "", fieldDiskReadMBps},
		{"vfs.dev.write[sdb,sectors]", "", fieldDiskWriteMBps},
		{"vfs.dev.read[sda]", "", fieldUnknown},
		{"vfs.dev.read[sda,ops]", "", fieldUnknown},
		{"vfs.dev.read[loop0,sps]", "", fieldUnknown},

		{`perf_counter_en["\PhysicalDisk(_Total)\Disk Read Bytes/sec",60]`, "", fieldDiskReadMBps},
		{`perf_counter_en["\PhysicalDisk(_Total)\Disk Write Bytes/sec",60]`, "", fieldDiskWriteMBps},
		{`perf_counter_en["\PhysicalDisk(_Total)\Disk Reads/sec",60]`, "", fieldUnknown},
		{`perf_counter_en["\PhysicalDisk(0 C:)\Avg. Disk sec/Read",60]`, "", fieldDiskReadLatency},
		{`perf_counter_en["\PhysicalDisk(_Total)\Avg. Disk Read Queue Length",60]`, "", fieldDiskReadQueue},
		{`perf_counter["\234(_Total)\220"]`, "", fieldDiskReadMBps},
		{`perf_counter["\234(_Total)\208"]`, "", fieldDiskReadLatency},

		{`net.if.in["eth0"]`, "", fieldNetIn},
		{`net.if.in["eth0",bytes]`, "", fieldNetIn},
		{`net.if.in["eth0",dropped]`, "", fieldUnknown},
		{`net.if.out["lo"]`, "", fieldUnknown},
		{`net.if.in["Local Area Connection"]`, "", fieldNetIn},
		{`net.if.in["{12345678-AAAA-BBBB-CCCC-1234567890AB}"]`, "Interface Ethernet: Bits received", fieldNetIn},
		{`net.if.in["{12345678-AAAA-BBBB-CCCC-1234567890AB}"]`, "Interface Loopback Pseudo-Interface 1: Bits received", fieldUnknown},
		{`perf_counter_en["\Network Interface(Ethernet)\Bytes Received/sec"]`, "", fieldNetIn},
		{`perf_counter_en["\Network Interface(isatap.{abc})\Bytes Received/sec"]`, "", fieldUnknown},
	}
	for _, tc := range cases {
		got := classifyItem(zabbixItem{Key: tc.key, Name: tc.name})
		if got != tc.want {
			t.Errorf("classify(%q, %q)=%v want %v", tc.key, tc.name, got, tc.want)
		}
	}
}

func TestToMBps(t *testing.T) {
	BytesPerMB = 1_000_000
	if g := toMBps(8_000_000, "bps", `net.if.in["eth0"]`); !almostEqual(g, 1) {
		t.Errorf("net bps: got %v want 1", g)
	}
	if g := toMBps(2_000_000, "Bps", `perf_counter_en["\\PhysicalDisk(_Total)\\Disk Read Bytes/sec"]`); !almostEqual(g, 2) {
		t.Errorf("disk Bps: got %v want 2", g)
	}
	if g := toMBps(2000, "", "vfs.dev.read[sda,sps]"); !almostEqual(g, 2000*512/1_000_000.0) {
		t.Errorf("sectors: got %v", g)
	}
}

func TestToMs(t *testing.T) {
	if g := toMs(0.02, "s", `perf_counter_en["\PhysicalDisk(_Total)\Avg. Disk sec/Read"]`); !almostEqual(g, 20) {
		t.Errorf("sec to ms: got %v want 20", g)
	}
	if g := toMs(0.02, "", `perf_counter_en["\PhysicalDisk(_Total)\Avg. Disk sec/Read"]`); !almostEqual(g, 20) {
		t.Errorf("empty units windows latency: got %v want 20", g)
	}
	if g := toMs(12, "!ms", "vfs.dev.read.await[sda]"); !almostEqual(g, 12) {
		t.Errorf("ms: got %v want 12", g)
	}
}

func TestBuildRowLinuxLegacyAndModern(t *testing.T) {
	BytesPerMB = 1_000_000
	at := time.Date(2026, 8, 30, 12, 0, 0, 0, time.UTC)
	items := []zabbixItem{
		{ItemID: "1", Key: "system.cpu.util[,idle]", Units: "%"},
		{ItemID: "2", Key: "vm.memory.size[available]"},
		{ItemID: "3", Key: "vm.memory.size[total]"},
		{ItemID: "4", Key: "vfs.dev.read[sda,sps]"},
		{ItemID: "5", Key: "vfs.dev.read[sda1,sps]"},
		{ItemID: "6", Key: "vfs.dev.read.await[sda]", Units: "ms"},
		{ItemID: "7", Key: "vfs.dev.queue_size[sda]"},
		{ItemID: "8", Key: `net.if.in["eth0"]`, Units: "bps"},
		{ItemID: "9", Key: `net.if.out["eth0"]`, Units: "bps"},
		{ItemID: "10", Key: `net.if.in["lo"]`, Units: "bps"},
	}
	values := map[string]float64{
		"1":  58.5,
		"2":  4_000_000_000,
		"3":  8_000_000_000,
		"4":  2000,
		"5":  2000,
		"6":  3.5,
		"7":  0.4,
		"8":  8_000_000,
		"9":  16_000_000,
		"10": 99_000_000,
	}
	row := buildRow(at, 1, "web-01", items, values)
	if row == nil {
		t.Fatal("expected row")
	}
	if !almostEqual(row.CPUUtilPercent, 41.5) {
		t.Errorf("cpu from idle: %v", row.CPUUtilPercent)
	}
	if !almostEqual(row.RAMUtilPercent, 50) {
		t.Errorf("ram from available/total: %v", row.RAMUtilPercent)
	}
	if !almostEqual(row.DiskReadMBps, 2000*512/1_000_000.0) {
		t.Errorf("disk read should ignore partition sda1: %v", row.DiskReadMBps)
	}
	if !almostEqual(row.DiskReadLatencyMs, 3.5) {
		t.Errorf("latency: %v", row.DiskReadLatencyMs)
	}
	if !almostEqual(row.DiskReadQueue, 0.4) || !almostEqual(row.DiskWriteQueue, 0.4) {
		t.Errorf("queue fallback: read=%v write=%v", row.DiskReadQueue, row.DiskWriteQueue)
	}
	if !almostEqual(row.NetRecvMBps, 1) || !almostEqual(row.NetSentMBps, 2) {
		t.Errorf("net: recv=%v sent=%v", row.NetRecvMBps, row.NetSentMBps)
	}
}

func TestBuildRowWindowsPrefersTotalAndDoesNotDouble(t *testing.T) {
	BytesPerMB = 1_000_000
	at := time.Date(2026, 8, 30, 12, 1, 0, 0, time.UTC)
	items := []zabbixItem{
		{ItemID: "1", Key: "system.cpu.util", Units: "%"},
		{ItemID: "2", Key: "vm.memory.size[pused]", Units: "%"},
		{ItemID: "3", Key: `perf_counter_en["\PhysicalDisk(_Total)\Disk Read Bytes/sec",60]`, Units: "Bps"},
		{ItemID: "4", Key: `perf_counter_en["\PhysicalDisk(0 C:)\Disk Read Bytes/sec",60]`, Units: "Bps"},
		{ItemID: "5", Key: `perf_counter["\PhysicalDisk(_Total)\Disk Read Bytes/sec"]`, Units: "Bps"},
		{ItemID: "6", Key: `perf_counter_en["\PhysicalDisk(_Total)\Avg. Disk sec/Read",60]`, Units: "s"},
		{ItemID: "7", Key: `perf_counter_en["\PhysicalDisk(_Total)\Avg. Disk Read Queue Length",60]`},
		{ItemID: "8", Key: `perf_counter_en["\PhysicalDisk(_Total)\Avg. Disk Write Queue Length",60]`},
		{ItemID: "9", Key: `net.if.in["{AAA}"]`, Name: "Interface Ethernet: Bits received", Units: "bps"},
		{ItemID: "10", Key: `net.if.in["{BBB}"]`, Name: "Interface Loopback Pseudo-Interface 1: Bits received", Units: "bps"},
	}
	values := map[string]float64{
		"1":  22,
		"2":  71,
		"3":  3_000_000,
		"4":  3_000_000,
		"5":  3_000_000,
		"6":  0.015,
		"7":  0.2,
		"8":  0.3,
		"9":  8_000_000,
		"10": 8_000_000,
	}
	row := buildRow(at, 1, "win-01", items, values)
	if row == nil {
		t.Fatal("expected row")
	}
	if !almostEqual(row.CPUUtilPercent, 22) || !almostEqual(row.RAMUtilPercent, 71) {
		t.Errorf("cpu/ram: %+v", row)
	}
	if !almostEqual(row.DiskReadMBps, 3) {
		t.Errorf("disk _Total must not be summed with per-disk or duplicate template: %v", row.DiskReadMBps)
	}
	if !almostEqual(row.DiskReadLatencyMs, 15) {
		t.Errorf("windows latency ms: %v", row.DiskReadLatencyMs)
	}
	if !almostEqual(row.DiskReadQueue, 0.2) || !almostEqual(row.DiskWriteQueue, 0.3) {
		t.Errorf("windows queues: read=%v write=%v", row.DiskReadQueue, row.DiskWriteQueue)
	}
	if !almostEqual(row.NetRecvMBps, 1) {
		t.Errorf("net should skip loopback by name: %v", row.NetRecvMBps)
	}
}

func TestBuildRowPrefersUtilOverIdle(t *testing.T) {
	at := time.Date(2026, 8, 30, 12, 2, 0, 0, time.UTC)
	items := []zabbixItem{
		{ItemID: "1", Key: "system.cpu.util", Units: "%"},
		{ItemID: "2", Key: "system.cpu.util[,idle]", Units: "%"},
		{ItemID: "3", Key: "vm.memory.util", Units: "%"},
		{ItemID: "4", Key: "vm.memory.size[pused]", Units: "%"},
	}
	values := map[string]float64{"1": 40, "2": 10, "3": 55, "4": 90}
	row := buildRow(at, 1, "mix", items, values)
	if row == nil {
		t.Fatal("expected row")
	}
	if !almostEqual(row.CPUUtilPercent, 40) {
		t.Errorf("prefer system.cpu.util: %v", row.CPUUtilPercent)
	}
	if !almostEqual(row.RAMUtilPercent, 55) {
		t.Errorf("prefer vm.memory.util: %v", row.RAMUtilPercent)
	}
}
