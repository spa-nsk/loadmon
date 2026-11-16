package grpcclient

import (
	pb "loadmon/protocol"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetDBVMPerfEmptyKey(t *testing.T) {
	if recs, err := GetDBVMPerf(""); err == nil || recs != nil {
		t.Fatalf("пустой ключ: ожидалась ошибка без данных, получено %v, %v", recs, err)
	}
}

func TestMessageToVMPerfRecords(t *testing.T) {
	if got := messageToVMPerfRecords(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil-ответ должен давать пустой не nil срез, получено %v", got)
	}
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	in := &pb.VMPerfRecordSlice{Count: 2, Perf: []*pb.VMPerfRecord{
		{Id: 1, VMID: 7, Start: timestamppb.New(start), CpuUsage: 1.5, DiskTotalReadLatency: 2.5, DiskTotalWriteLatency: 3.5},
		nil,
		{Id: 2, VMID: 7, Start: timestamppb.New(start.Add(time.Minute))},
	}}
	got := messageToVMPerfRecords(in)
	if len(got) != 2 {
		t.Fatalf("ожидалось 2 записи, получено %d", len(got))
	}
	if got[0].Id != 1 || got[0].VMId != 7 || !got[0].Start.Equal(start) ||
		got[0].Cpu_usage != 1.5 || got[0].Disk_totalReadLatency != 2.5 || got[0].Disk_totalWriteLatency != 3.5 {
		t.Fatalf("неверное преобразование: %+v", got[0])
	}
	if got[1].Id != 2 {
		t.Fatalf("неверный порядок: %+v", got[1])
	}
}
