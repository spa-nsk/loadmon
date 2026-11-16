package grpcserver

import (
	"context"
	"loadmon/models"
	pb "loadmon/protocol"
	"testing"
	"time"
)

func TestSplitVMKey(t *testing.T) {
	v, d, c, n, err := splitVMKey("vc1/dc/cl/my/vm")
	if err != nil || v != "vc1" || d != "dc" || c != "cl" || n != "my/vm" {
		t.Fatalf("неверный разбор: %q %q %q %q %v", v, d, c, n, err)
	}
	for _, bad := range []string{"", "a/b/c", "a//c/d", "/b/c/d", "a/b/c/"} {
		if _, _, _, _, err := splitVMKey(bad); err == nil {
			t.Errorf("ключ %q должен давать ошибку", bad)
		}
	}
}

func TestSortVMPerfByStart(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	recs := []*models.VMPerfRecord{
		{Id: 3, Start: t0.Add(2 * time.Hour)},
		{Id: 2, Start: t0},
		{Id: 1, Start: t0},
		{Id: 4, Start: t0.Add(time.Hour)},
	}
	sortVMPerfByStart(recs)
	want := []int64{1, 2, 4, 3}
	for i, r := range recs {
		if r.Id != want[i] {
			t.Fatalf("позиция %d: Id=%d, ожидался %d", i, r.Id, want[i])
		}
	}
}

func TestGetDBVMPerfBadKey(t *testing.T) {
	s := &serverdb{}
	if res, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{}); err == nil || res == nil || res.Perf == nil {
		t.Fatalf("пустой ключ: ожидалась ошибка и непустой (не nil) ответ, получено %v, %v", res, err)
	}
}
