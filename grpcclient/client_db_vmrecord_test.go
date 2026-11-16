package grpcclient

import (
	pb "loadmon/protocol"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestGetDBVMRecordBadArgs(t *testing.T) {
	now := time.Now()
	if recs, err := GetDBVMRecord("", now.Add(-time.Hour), now); err == nil || recs != nil {
		t.Fatalf("пустой ключ: ожидалась ошибка, получено %v, %v", recs, err)
	}
	if recs, err := GetDBVMRecord("a/b/c/d", now, now.Add(-time.Hour)); err == nil || recs != nil {
		t.Fatalf("end < start: ожидалась ошибка, получено %v, %v", recs, err)
	}
}

func TestNewVMRecordPeriodReq(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	req := newVMRecordPeriodReq("a/b/c/d", start, time.Time{})
	if req.Key != "a/b/c/d" || req.Start == nil || !req.Start.AsTime().Equal(start) || req.End != nil {
		t.Fatalf("неверный запрос: %v", req)
	}
	if req := newVMRecordPeriodReq("k", time.Time{}, time.Time{}); req.Start != nil || req.End != nil {
		t.Fatalf("нулевые границы должны быть nil: %v", req)
	}
}

func TestMessageToVMRecords(t *testing.T) {
	if got := messageToVMRecords(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil-ответ должен давать пустой не nil срез, получено %v", got)
	}
	date := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	in := &pb.VMRecordSlice{Count: 2, Records: []*pb.VMRecord{
		{Id: 1, VMSprId: 7, Date: timestamppb.New(date), CpuUsageMaximum: 90.5},
		nil,
		{Id: 2, VMSprId: 7, Date: timestamppb.New(date.Add(24 * time.Hour))},
	}}
	got := messageToVMRecords(in)
	if len(got) != 2 {
		t.Fatalf("ожидалось 2 записи, получено %d", len(got))
	}
	if got[0].Id != 1 || got[0].VMSprId != 7 || !got[0].Date.Equal(date) || got[0].Cpu_usage_maximum != 90.5 {
		t.Fatalf("неверное преобразование: %+v", got[0])
	}
	if got[1].Id != 2 {
		t.Fatalf("неверный порядок: %+v", got[1])
	}
}
