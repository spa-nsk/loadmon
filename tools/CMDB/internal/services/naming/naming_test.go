package naming

import (
	"strings"
	"testing"
)

func TestBuildHostName(t *testing.T) {
	reg := DefaultRegistry()

	cases := []struct {
		site string
		cls  string
		seq  int64
		want string
	}{
		{"OVBDC1", "S", 1001, "OVBDC1-S-01001"},
		{"OVBDC1", "V", 10234, "OVBDC1-V-10234"},
		{"OVBDC1", "ES", 1, "OVBDC1-ES-001"},
		{"OVBDC1", "CL", 4, "OVBDC1-CL-004"},
		{"OVBDC1", "SA", 3, "OVBDC1-SA-003"},
		{"ovbdc1", "s", 1001, "OVBDC1-S-01001"}, // регистр входных данных не важен
	}

	for _, c := range cases {
		got, err := BuildHostName(reg, c.site, c.cls, c.seq)
		if err != nil {
			t.Errorf("BuildHostName(%q,%q,%d) unexpected error: %v", c.site, c.cls, c.seq, err)
			continue
		}
		if got != c.want {
			t.Errorf("BuildHostName(%q,%q,%d) = %q, want %q", c.site, c.cls, c.seq, got, c.want)
		}
	}
}

func TestDNSName(t *testing.T) {
	if got := DNSName("OVBDC1-S-01001"); got != "ovbdc1-s-01001" {
		t.Errorf("DNSName = %q, want %q", got, "ovbdc1-s-01001")
	}
}

func TestBuildHostName_UnknownClass(t *testing.T) {
	reg := DefaultRegistry()
	if _, err := BuildHostName(reg, "OVBDC1", "ZZ", 1); err == nil {
		t.Fatal("expected error for unknown class code")
	}
}

func TestBuildHostName_DeprecatedClassRejectedWithSuggestion(t *testing.T) {
	// AC-11: попытка выдать новое имя с кодом deprecated (-FC-) отклоняется с подсказкой замены.
	reg := DefaultRegistry()
	_, err := BuildHostName(reg, "OVBDC1", "FC", 1)
	if err == nil {
		t.Fatal("expected error for deprecated class code FC")
	}
	if got := err.Error(); !strings.Contains(got, "CL") {
		t.Errorf("error %q does not mention replacement code CL", got)
	}
}

func TestBuildHostName_SeqTooLong(t *testing.T) {
	reg := DefaultRegistry()
	if _, err := BuildHostName(reg, "OVBDC1", "CL", 1000); err == nil {
		t.Fatal("expected error: 1000 does not fit in 3 digits")
	}
}

func TestBuildHostName_SiteTooLong(t *testing.T) {
	reg := DefaultRegistry()
	if _, err := BuildHostName(reg, "OVBDC1X", "S", 1); err == nil {
		t.Fatal("expected error: SITE longer than 6 chars")
	}
}

func TestParseHostName(t *testing.T) {
	p, err := ParseHostName("ovbdc1-s-01001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Site != "OVBDC1" || p.Cls != "S" || p.Seq != 1001 || p.SeqDigits != 5 {
		t.Errorf("unexpected parse result: %+v", p)
	}
}

func TestParseHostName_WrongShape(t *testing.T) {
	cases := []string{
		"ovbdc1-s-01001-bmc", // производное имя, не основной шаблон
		"ovbdc1s01001",
		"",
	}
	for _, name := range cases {
		if _, err := ParseHostName(name); err == nil {
			t.Errorf("ParseHostName(%q) expected error, got nil", name)
		}
	}
}

func TestValidate_Compliant(t *testing.T) {
	reg := DefaultRegistry()
	cases := []string{"OVBDC1-S-01001", "ovbdc1-v-10234", "OVBDC1-CL-004", "OVBDC1-SA-003"}
	for _, name := range cases {
		res := Validate(reg, name)
		if res.Status != StatusCompliant {
			t.Errorf("Validate(%q) = %+v, want compliant", name, res)
		}
	}
}

func TestValidate_UnknownClassIsNoncompliant(t *testing.T) {
	reg := DefaultRegistry()
	res := Validate(reg, "OVBDC1-ZZ-001")
	if res.Status != StatusNoncompliant {
		t.Errorf("Validate unknown class = %+v, want noncompliant", res)
	}
}

func TestValidate_DeprecatedClassSuggestsReplacement(t *testing.T) {
	// Имя с устаревшим кодом Guide 2019 (-FC-) распознаётся как legacy с подсказкой замены;
	// AC-11 (отказ при попытке ВЫДАТЬ новое имя с этим кодом) проверяется в BuildHostName.
	reg := DefaultRegistry()
	res := Validate(reg, "OVBDC1-FC-001")
	if res.Status != StatusLegacy {
		t.Fatalf("Validate deprecated class = %+v, want legacy", res)
	}
	if res.Suggestion != "OVBDC1-CL-001" {
		t.Errorf("Suggestion = %q, want %q", res.Suggestion, "OVBDC1-CL-001")
	}
}

func TestValidate_WrongSeqWidth(t *testing.T) {
	reg := DefaultRegistry()
	// CL требует 3 цифры (intent §5.3); "04" — 2 цифры.
	res := Validate(reg, "OVBDC1-CL-04")
	if res.Status != StatusNoncompliant {
		t.Errorf("Validate(%q) = %+v, want noncompliant (wrong seq width)", "OVBDC1-CL-04", res)
	}
}

func TestBuildBMCName(t *testing.T) {
	got, err := BuildBMCName("OVBDC1-S-01001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ovbdc1-s-01001-bmc"; got != want {
		t.Errorf("BuildBMCName = %q, want %q", got, want)
	}
}

func TestBuildStorageIfaceName(t *testing.T) {
	got, err := BuildStorageIfaceName("OVBDC1-H-00012", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ovbdc1-h-00012-st1"; got != want {
		t.Errorf("BuildStorageIfaceName = %q, want %q", got, want)
	}
}

func TestBuildClusterResourceName(t *testing.T) {
	got, err := BuildClusterResourceName("OVBDC1-CL-004", ClusterResourceListener, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ovbdc1-cl-004-l1"; got != want {
		t.Errorf("BuildClusterResourceName = %q, want %q", got, want)
	}
}

func TestBuildPortName(t *testing.T) {
	got, err := BuildPortName("OVBDC1-FS-002", 17)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ovbdc1-fs-002-p17"; got != want {
		t.Errorf("BuildPortName = %q, want %q", got, want)
	}
}

func TestBuildHBAPortName(t *testing.T) {
	got, err := BuildHBAPortName("OVBDC1-S-01001", 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ovbdc1-s-01001-hba1p2"; got != want {
		t.Errorf("BuildHBAPortName = %q, want %q", got, want)
	}
}

func TestBuildSANZoneName(t *testing.T) {
	got, err := BuildSANZoneName("OVBDC1-S-01001", 1, "OVBDC1-SA-003")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "z_ovbdc1-s-01001_hba1_ovbdc1-sa-003"; got != want {
		t.Errorf("BuildSANZoneName = %q, want %q", got, want)
	}
}

func TestBuildVolumeName(t *testing.T) {
	got, err := BuildVolumeName("OVBDC1-CL-004", "data", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ovbdc1-cl-004_data_001"; got != want {
		t.Errorf("BuildVolumeName = %q, want %q", got, want)
	}
}

func TestBuildSVMName(t *testing.T) {
	got, err := BuildSVMName("OVBDC1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "ovbdc1-svm-002"; got != want {
		t.Errorf("BuildSVMName = %q, want %q", got, want)
	}
}

func TestBuildK8sClusterName(t *testing.T) {
	got, err := BuildK8sClusterName("OVBDC1", "prod", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "k8s-ovbdc1-prod-01"; got != want {
		t.Errorf("BuildK8sClusterName = %q, want %q", got, want)
	}
}

func TestBuildK8sClusterName_RejectsInvalidRFC1123(t *testing.T) {
	// env содержащий "_" не должен пройти проверку RFC 1123.
	if _, err := BuildK8sClusterName("OVBDC1", "pr_od", 1); err == nil {
		t.Fatal("expected RFC 1123 validation error")
	}
}

func TestRegistry_ByClass(t *testing.T) {
	reg := DefaultRegistry()
	d, ok := reg.ByClass("server")
	if !ok || d.Code != "S" {
		t.Errorf("ByClass(server) = %+v, %v, want code S", d, ok)
	}
	if _, ok := reg.ByClass("no-such-class"); ok {
		t.Error("ByClass(no-such-class) expected ok=false")
	}
}

// TestBuildFunctions_RejectNegativeAndInvalid — ветки ошибок билдеров производных имён
// (suffix.go): отрицательные номера и недопустимые метки DNS.
func TestBuildFunctions_RejectNegativeAndInvalid(t *testing.T) {
	if _, err := BuildBMCName(""); err == nil {
		t.Error("BuildBMCName(\"\") expected error")
	}
	if _, err := BuildStorageIfaceName("OVBDC1-H-00012", -1); err == nil {
		t.Error("BuildStorageIfaceName negative n expected error")
	}
	if _, err := BuildClusterResourceName("OVBDC1-CL-004", ClusterResourceListener, -1); err == nil {
		t.Error("BuildClusterResourceName negative n expected error")
	}
	if _, err := BuildPortName("OVBDC1-FS-002", -1); err == nil {
		t.Error("BuildPortName negative port expected error")
	}
	if _, err := BuildHBAPortName("OVBDC1-S-01001", -1, 2); err == nil {
		t.Error("BuildHBAPortName negative hba expected error")
	}
	if _, err := BuildHBAPortName("OVBDC1-S-01001", 1, -2); err == nil {
		t.Error("BuildHBAPortName negative port expected error")
	}
	if _, err := BuildSANZoneName("OVBDC1-S-01001", -1, "OVBDC1-SA-003"); err == nil {
		t.Error("BuildSANZoneName negative hba expected error")
	}
	if _, err := BuildVolumeName("OVBDC1-CL-004", "data", 1000); err == nil {
		t.Error("BuildVolumeName n out of range expected error")
	}
	if _, err := BuildSVMName("OVBDC1", 1000); err == nil {
		t.Error("BuildSVMName n out of range expected error")
	}
	if _, err := BuildSVMName("OVBDC1X", 1); err == nil {
		t.Error("BuildSVMName invalid site expected error")
	}
	if _, err := BuildK8sClusterName("OVBDC1", "prod", 100); err == nil {
		t.Error("BuildK8sClusterName n out of range expected error")
	}
	if _, err := BuildK8sClusterName("OVBDC1X", "prod", 1); err == nil {
		t.Error("BuildK8sClusterName invalid site expected error")
	}
	long := strings.Repeat("a", 70)
	if err := checkDNSLabel(long); err == nil {
		t.Error("checkDNSLabel too-long expected error")
	}
}
