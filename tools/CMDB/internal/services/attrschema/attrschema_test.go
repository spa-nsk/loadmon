package attrschema

import (
	"reflect"
	"testing"
)

func TestParse_Empty(t *testing.T) {
	for _, raw := range []string{"", "{}"} {
		s, err := Parse(raw)
		if err != nil {
			t.Fatalf("Parse(%q): %v", raw, err)
		}
		if len(s.RequiredByStage) != 0 || len(s.Properties) != 0 {
			t.Errorf("Parse(%q) = %+v, want empty schema", raw, s)
		}
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	if _, err := Parse("not json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRequired_OnlyOwnStageNotAccumulated(t *testing.T) {
	s := &Schema{RequiredByStage: map[string][]string{
		"Commissioning": {"cpu_count", "ram_gb"},
		"Operational":   {"bios_uuid"},
	}}

	// Стадия без собственных требований — ничего не требуется, даже если она позже в жизненном
	// цикле, чем Commissioning/Operational (например, прямой переход Planned → Retired без ввода
	// в эксплуатацию — см. комментарий у Schema.Required).
	if got := s.Required("Retired"); got != nil {
		t.Errorf("Required(Retired) = %v, want nil (нет собственных требований)", got)
	}
	if got := s.Required("Ordered"); got != nil {
		t.Errorf("Required(Ordered) = %v, want nil", got)
	}

	got := s.Required("Commissioning")
	want := []string{"cpu_count", "ram_gb"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Required(Commissioning) = %v, want %v", got, want)
	}

	// Operational не наследует требования Commissioning — только собственные.
	got = s.Required("Operational")
	want = []string{"bios_uuid"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Required(Operational) = %v, want %v", got, want)
	}
}

func TestMissing(t *testing.T) {
	required := []string{"cpu_count", "ram_gb", "os_family"}
	attrs := map[string]string{"cpu_count": "2", "ram_gb": "", "os_family": "  "}
	got := Missing(required, attrs)
	want := []string{"ram_gb", "os_family"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Missing = %v, want %v", got, want)
	}
}

func TestRequired_NilSchemaNoRequirements(t *testing.T) {
	var s *Schema
	if got := s.Required("Operational"); got != nil {
		t.Errorf("nil schema Required = %v, want nil", got)
	}
}
