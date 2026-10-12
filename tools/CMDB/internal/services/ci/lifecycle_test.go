package ci

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{StagePlanned, StageOrdered, true},
		{StageOrdered, StageInStock, true},
		{StageInStock, StageCommissioning, true},
		{StageCommissioning, StageOperational, true},
		{StageOperational, StageMaintenance, true},
		{StageMaintenance, StageOperational, true},
		{StageOperational, StageDecommissioning, true},
		{StageDecommissioning, StageRetired, true},
		{StageRetired, StageDisposed, true},
		{StageRetired, StageInStock, true},
		{StageOperational, StageOperational, true}, // смена только статуса
		{StageOperational, StageRetired, false},    // минуя Decommissioning
		{StagePlanned, StageOperational, false},    // минуя ввод в эксплуатацию
		{StageDisposed, StageInStock, false},       // конечная стадия
		{"Unknown", StageOperational, false},
		{StageOperational, "Unknown", false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%q, %q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestAllTargetsAreKnownStages(t *testing.T) {
	for from, targets := range allowedTransitions {
		for _, to := range targets {
			if !KnownStage(to) {
				t.Errorf("матрица: %q → неизвестная стадия %q", from, to)
			}
		}
	}
}
