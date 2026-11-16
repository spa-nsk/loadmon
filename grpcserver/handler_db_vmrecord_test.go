package grpcserver

import (
	"context"
	pb "loadmon/protocol"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVMRecordPeriod(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(48 * time.Hour)

	s, e, err := vmRecordPeriod(&pb.VMRecordPeriodReq{Start: timestamppb.New(t0), End: timestamppb.New(t1)})
	if err != nil || !s.Equal(t0) || !e.Equal(t1) {
		t.Fatalf("неверный период: %v %v %v", s, e, err)
	}
	s, e, err = vmRecordPeriod(&pb.VMRecordPeriodReq{})
	if err != nil || !s.IsZero() || !e.IsZero() {
		t.Fatalf("nil-границы должны давать нулевое время: %v %v %v", s, e, err)
	}
	if _, _, err := vmRecordPeriod(&pb.VMRecordPeriodReq{Start: timestamppb.New(t1), End: timestamppb.New(t0)}); err == nil {
		t.Fatal("end < start должно давать ошибку")
	}
}

func TestGetDBVMRecordBadRequest(t *testing.T) {
	s := &serverdb{}
	if res, err := s.GetDBVMRecord(context.Background(), &pb.VMRecordPeriodReq{}); err == nil || res == nil || res.Records == nil {
		t.Fatalf("пустой ключ: ожидалась ошибка и не nil ответ, получено %v, %v", res, err)
	}
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bad := &pb.VMRecordPeriodReq{Key: "a/b/c/d", Start: timestamppb.New(t0), End: timestamppb.New(t0.Add(-time.Hour))}
	if _, err := s.GetDBVMRecord(context.Background(), bad); err == nil {
		t.Fatal("end < start должно давать ошибку")
	}
}
