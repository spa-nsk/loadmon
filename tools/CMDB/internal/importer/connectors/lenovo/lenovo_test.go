package lenovo

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/lenovo/lenovomock"
	"cmdb/internal/importer/connectors/storage"
)

func newClient(t *testing.T, m *lenovomock.Server, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("LENOVO_TEST_PASSWORD", password)
	c, err := New(connector.Config{Code: "lenovo-test", Kind: Kind, Settings: map[string]any{
		"base_url": srv.URL, "username": m.User, "password_env": "LENOVO_TEST_PASSWORD"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func byExt(items []canon.Item) map[string]canon.Record {
	m := map[string]canon.Record{}
	for _, it := range items {
		m[it.Record.ExternalID] = it.Record
	}
	return m
}

func TestExtract_OneGraphCallMapsEverything(t *testing.T) {
	m := lenovomock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// массив + группа томов + дисковый пул + 3 тома (2 обычных и 1 тонкий)
	if len(items) != 6 {
		t.Fatalf("записей = %d, want 6", len(items))
	}
	recs := byExt(items)
	a := recs["array:lenovo-test"]
	if a.SourceName != m.Label || a.SerialNumber != m.Serial || a.Attrs["array_family"] != "Lenovo ThinkSystem DE4000H" || a.Attrs["microcode"] != "08.74.00.00" || a.Attrs["controllers"] != 2 {
		t.Errorf("массив = %+v", a)
	}
	types := map[string]int{}
	var nvmeParent string
	for _, c := range a.Components {
		types[c.Type]++
		if c.Type == "nvme" {
			nvmeParent = c.ParentKey
		}
	}
	if types["controller"] != 2 || types["enclosure"] != 1 || types["disk"] != 1 || types["ssd"] != 1 || types["nvme"] != 1 {
		t.Errorf("типы компонентов = %v (media hdd -> disk, ssd -> ssd, interfaceType nvme -> nvme)", types)
	}
	if nvmeParent != "enc:0E00000000000000000000A1" {
		t.Errorf("диск должен быть вложен в полку по physicalLocation.trayRef, parent=%q", nvmeParent)
	}
	if a.Attrs["raw_tb"] != 6.32 { // 1.2 + 1.92 + 3.2 ТБ
		t.Errorf("raw_tb = %v, want 6.32", a.Attrs["raw_tb"])
	}
	vol := recs["vol:02000000600A0980001"]
	if vol.Attrs["size_gb"] != 107.37 || vol.Attrs["thin"] != false || vol.Attrs["lun_id"] != 0 || len(vol.Identifiers["wwn"]) != 32 {
		t.Errorf("том = %+v", vol)
	}
	if thin := recs["vol:02000000600A0980003"]; thin.Attrs["thin"] != true {
		t.Errorf("тонкий том = %+v", thin.Attrs)
	}
	if len(recs["pool:04000000600A098001"].Relations) != 2 { // vol-b и тонкий том, оба по ссылке пула
		t.Errorf("дисковый пул содержит %d томов, want 2", len(recs["pool:04000000600A098001"].Relations))
	}
}

func TestExtract_AuthAndFailure(t *testing.T) {
	m := lenovomock.Default("0123456789")
	if err := newClient(t, m, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if err := newClient(t, m, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v", err)
	}
	m.Do(func(s *lenovomock.Server) { s.Fail = true })
	if items, err := newClient(t, m, m.Password).Extract(context.Background()); err == nil || items != nil {
		t.Fatalf("Extract при сбое: items=%d err=%v", len(items), err)
	}
}

func wwpns(r canon.Record) []string {
	w, _ := r.Attrs["mapped_wwpns"].([]string)
	return w
}

func TestExtract_VolumeMappingsResolveHostInitiators(t *testing.T) {
	m := lenovomock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	recs := byExt(items)
	if got := wwpns(recs["vol:02000000600A0980001"]); len(got) != 1 || got[0] != storage.SampleWWPN("0123456789", 5) {
		t.Errorf("mapped_wwpns = %v, want WWPN хоста, которому смаплен том", got)
	}
	if got := wwpns(recs["vol:02000000600A0980002"]); len(got) != 0 {
		t.Errorf("том без маппинга: %v", got)
	}
}
