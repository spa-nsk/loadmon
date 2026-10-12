package csv

import (
	"strings"
	"testing"

	"cmdb/internal/services/ci"
)

func serverSpec(t *testing.T) ci.ClassSpec {
	t.Helper()
	spec := ci.SpecFor("server")
	if spec == nil {
		t.Fatal("нет ClassSpec для server")
	}
	return *spec
}

func TestTemplate_HasExpectedColumns(t *testing.T) {
	spec := serverSpec(t)
	tpl := string(Template(spec))
	for _, col := range []string{"name", "location_code", "serial_number", "cpu_count", "ram_gb", "bios_uuid"} {
		if !strings.Contains(tpl, col) {
			t.Errorf("шаблон не содержит колонку %q: %s", col, tpl)
		}
	}
}

func TestParse_CSV_NormalizesAndValidates(t *testing.T) {
	spec := serverSpec(t)
	csvBody := "name,location_code,serial_number,cpu_count,ram_gb,os_family,bios_uuid\n" +
		"AMSDC1-S-01000,TST-QA1,SN-1,2,16384MB,Linux,4C4C4544-0043-3410-8035-C7C04F435931\n" +
		"AMSDC1-S-01001,TST-QA1,SN-2,4,32 GB,Windows,\n" +
		"BROKEN-ROW,TST-QA1,SN-3,not-a-number,16,Linux,\n"

	rows, err := Parse(strings.NewReader(csvBody), "csv", spec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("Parse вернул %d строк, want 3", len(rows))
	}

	r0 := rows[0]
	if r0.Err != nil {
		t.Fatalf("row 0: неожиданная ошибка: %v", r0.Err)
	}
	if r0.Record.SourceName != "AMSDC1-S-01000" {
		t.Errorf("row 0: SourceName = %q", r0.Record.SourceName)
	}
	if got := r0.Record.Attrs["ram_gb"]; got != "16" {
		t.Errorf("row 0: ram_gb = %v, want 16 (нормализовано из 16384MB)", got)
	}
	if got := r0.Record.Identifiers["bios_uuid"]; got != "4c4c4544-0043-3410-8035-c7c04f435931" {
		t.Errorf("row 0: bios_uuid identifier = %q, want нижний регистр", got)
	}
	if got := r0.Record.Identifiers["serial_number"]; got != "SN-1" {
		t.Errorf("row 0: serial_number identifier = %q", got)
	}

	r1 := rows[1]
	if r1.Err != nil {
		t.Fatalf("row 1: неожиданная ошибка: %v", r1.Err)
	}
	if got := r1.Record.Attrs["ram_gb"]; got != "32" {
		t.Errorf("row 1: ram_gb = %v, want 32 (нормализовано из \"32 GB\")", got)
	}

	r2 := rows[2]
	if r2.Err == nil {
		t.Error("row 2: ожидалась ошибка валидации cpu_count")
	}
}

func TestParse_EmptyName_IsError(t *testing.T) {
	spec := serverSpec(t)
	rows, err := Parse(strings.NewReader("name,location_code\n,TST-QA1\n"), "csv", spec)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(rows) != 1 || rows[0].Err == nil {
		t.Fatalf("ожидалась ошибка для строки без имени: %+v", rows)
	}
}

func TestParse_UnknownFormat(t *testing.T) {
	if _, err := Parse(strings.NewReader(""), "yaml", serverSpec(t)); err == nil {
		t.Error("ожидалась ошибка для неизвестного формата")
	}
}
