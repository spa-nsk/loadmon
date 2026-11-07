package cmdb

import "testing"

func TestNormalizeDiskRole(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"Data":        DiskRoleData,
		"DataDrive":   DiskRoleData,
		"Spare":       DiskRoleSpare,
		" HotSpare ":  DiskRoleSpare,
		"spare drive": DiskRoleSpare,
		"Raw":         DiskRoleRaw,
		"Unassigned":  DiskRoleRaw,
		"Failed":      DiskRoleFailed,
		"Cache":       DiskRoleCache,
		"Weird":       "Weird",
	}
	for in, want := range cases {
		if got := normalizeDiskRole(in); got != want {
			t.Errorf("normalizeDiskRole(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHealthStatus(t *testing.T) {
	cases := []struct{ h, s, want string }{
		{"OK", "Enabled", "OK"},
		{"OK", "", "OK"},
		{"Warning", "Enabled", "Warning"},
		{"OK", "StandbySpare", "OK / StandbySpare"},
		{"", "Absent", "Absent"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := healthStatus(c.h, c.s); got != c.want {
			t.Errorf("healthStatus(%q,%q) = %q, want %q", c.h, c.s, got, c.want)
		}
	}
}
