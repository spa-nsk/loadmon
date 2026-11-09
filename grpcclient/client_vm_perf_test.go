package grpcclient

import (
	"testing"
	"time"

	"loadmon/grpcserver"
	vsphere "loadmon/vspheremetrics"
)

// TestVMPerfRoundTrip проверяет, что серверный VMPerfToMessage и клиентский
// GetVMPerfRPC взаимно обратны (включая задержки диска) и корректно
// обрабатывают nil-элементы и пустой ввод.
func TestVMPerfRoundTrip(t *testing.T) {
	start := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
	in := []*vsphere.VMPerf{
		{ID: 1, Start: start, CPUUsage: 1.5, MemUsage: 2.5, DiskUsage: 3, DiskWrite: 4, DiskRead: 5,
			NetUsage: 6, NetReceived: 7, NetTransmitted: 8, DiskTotalReadLatency: 9, DiskTotalWriteLatency: 10},
		nil,
		{ID: 2, Start: start.Add(time.Minute)},
	}
	msg := grpcserver.VMPerfToMessage(in)
	if msg.Count != 2 || len(msg.Perf) != 2 {
		t.Fatalf("ожидалось 2 точки, получено Count=%d len=%d", msg.Count, len(msg.Perf))
	}
	out := GetVMPerfRPC(msg)
	if len(out) != 2 {
		t.Fatalf("ожидалось 2 точки после обратной конвертации, получено %d", len(out))
	}
	a, b := in[0], out[0]
	if a.ID != b.ID || !a.Start.Equal(b.Start) || a.CPUUsage != b.CPUUsage || a.MemUsage != b.MemUsage ||
		a.DiskUsage != b.DiskUsage || a.DiskWrite != b.DiskWrite || a.DiskRead != b.DiskRead ||
		a.NetUsage != b.NetUsage || a.NetReceived != b.NetReceived || a.NetTransmitted != b.NetTransmitted ||
		a.DiskTotalReadLatency != b.DiskTotalReadLatency || a.DiskTotalWriteLatency != b.DiskTotalWriteLatency {
		t.Fatalf("точка не совпала после round-trip: %+v != %+v", *a, *b)
	}
	if out[1].ID != 2 {
		t.Fatalf("порядок точек нарушен: %+v", *out[1])
	}
	if e := GetVMPerfRPC(nil); e == nil || len(e) != 0 {
		t.Fatalf("GetVMPerfRPC(nil) должен вернуть пустой не-nil срез")
	}
	if e := grpcserver.VMPerfToMessage(nil); e == nil || e.Count != 0 {
		t.Fatalf("VMPerfToMessage(nil) должен вернуть пустое сообщение")
	}
}
