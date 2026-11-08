package controllers

import "testing"

func TestParseBladePSUBayAndStatus(t *testing.T) {
	cases := []struct {
		slot       string
		wantBay    int
		wantPower  int
		wantStatus string
	}{
		{"Bay 3, OK", 3, 1, "OK"},
		{"Bay 1, Present", 1, 1, "Present"},
		{"Bay 2, IP 172.18.1.11", 2, 1, "IP 172.18.1.11"},
		{"Bay 1, IP 10.0.0.1, Active", 1, 1, "IP 10.0.0.1, Active"},
		{"Bay 5,", 5, 0, ""},
		{"Bay5, Warning", 5, 1, "Warning"},
		{"Отсек 4, OK", 4, 1, "OK"},
		{"3", 3, 0, "3"},
		{"3, Present", 3, 1, "Present"},
		{"Bay 7, Absent", 7, 2, "Absent"},
		{"Bay 8, Failed", 8, 2, "Failed"},
		{"", 0, 0, ""},
	}
	for _, c := range cases {
		bay, power, status := parseBladePSUBayAndStatus(c.slot)
		if bay != c.wantBay || power != c.wantPower || status != c.wantStatus {
			t.Errorf("parseBladePSUBayAndStatus(%q) = (%d,%d,%q), want (%d,%d,%q)",
				c.slot, bay, power, status, c.wantBay, c.wantPower, c.wantStatus)
		}
	}
}

func TestHwTypicalFanCount(t *testing.T) {
	cases := []struct {
		u     int
		model string
		want  int
	}{
		{1, "ProLiant DL360 Gen10", 7},
		{2, "ProLiant DL380 Gen10", 6},
		{2, "PowerEdge R740", 6},
		{4, "Generic", 6},
		{0, "", 6},
	}
	for _, c := range cases {
		if got := hwTypicalFanCount(c.u, c.model); got != c.want {
			t.Errorf("hwTypicalFanCount(%d,%q)=%d, want %d", c.u, c.model, got, c.want)
		}
	}
}

func TestParseBladeCompIP(t *testing.T) {
	cases := []struct{ slot, status, want string }{
		{"Bay 1, IP 172.18.1.11", "IP 172.18.1.11", "172.18.1.11"},
		{"Bay 2, IP 10.0.0.1, Active", "IP 10.0.0.1, Active", "10.0.0.1"},
		{"Bay 3, OK", "OK", ""},
		{"", "172.18.1.5", "172.18.1.5"},
	}
	for _, c := range cases {
		if got := parseBladeCompIP(c.slot, c.status); got != c.want {
			t.Errorf("parseBladeCompIP(%q,%q) = %q, want %q", c.slot, c.status, got, c.want)
		}
	}
}
