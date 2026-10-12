package asof

import (
	"context"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	if _, ok := Parse(""); ok {
		t.Error(`Parse("") = ok, want not ok`)
	}
	if _, ok := Parse("не дата"); ok {
		t.Error(`Parse("не дата") = ok, want not ok`)
	}
	tm, ok := Parse("2026-01-15T10:00:00Z")
	if !ok || !tm.Equal(time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("Parse(RFC3339) = %v, %v", tm, ok)
	}
	tm, ok = Parse("2026-01-15")
	if !ok || !tm.Equal(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Parse(date) = %v, %v", tm, ok)
	}
}

func TestWithTime_FromContext(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Error("FromContext(пустой ctx) = ok, want not ok")
	}
	want := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	ctx := WithTime(context.Background(), want)
	got, ok := FromContext(ctx)
	if !ok || !got.Equal(want) {
		t.Errorf("FromContext = %v, %v, want %v, true", got, ok, want)
	}
}
