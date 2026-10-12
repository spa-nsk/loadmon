package huawei

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/huawei/huaweimock"
	"cmdb/internal/importer/connectors/storage"
)

func newClient(t *testing.T, m *huaweimock.Server, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("HUAWEI_TEST_PASSWORD", password)
	c, err := New(connector.Config{Code: "hw-test", Kind: Kind, Settings: map[string]any{
		"base_url": srv.URL, "username": m.User, "password_env": "HUAWEI_TEST_PASSWORD", "page_size": 2}})
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

func TestExtract_MapsArrayDisksPoolsAndLUNs(t *testing.T) {
	m := huaweimock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 5 { // массив + пул + 3 LUN
		t.Fatalf("записей = %d, want 5", len(items))
	}
	recs := byExt(items)
	a := recs["array:hw-test"]
	if a.SerialNumber != m.DeviceID || a.SourceName != m.Name || a.Attrs["controllers"] != 2 || a.Attrs["microcode"] != "6.1.6 SPH100" {
		t.Errorf("массив = %+v", a)
	}
	// 2 контроллера + 2 полки + 3 диска (пагинация по 2: диски приходят двумя блоками)
	if len(a.Components) != 7 {
		t.Fatalf("компонентов = %d, want 7 (range-постраничность)", len(a.Components))
	}
	types := map[string]int{}
	var hdd float64
	for _, c := range a.Components {
		types[c.Type]++
		if c.Type == "disk" && c.CapacityValue != nil {
			hdd = *c.CapacityValue
		}
	}
	if types["nvme"] != 1 || types["ssd"] != 1 || types["disk"] != 1 || types["controller"] != 2 || types["enclosure"] != 2 {
		t.Errorf("типы компонентов = %v (DISKTYPE 14 -> nvme, 3 -> ssd, 1 -> disk)", types)
	}
	if hdd != 600 { // 1 153 846 153 секторов × 520 Б — не ×512 (harness/03, п. 4)
		t.Errorf("ёмкость HDD = %v ГБ, want 600 (SECTORS × SECTORSIZE, SECTORSIZE=520)", hdd)
	}
	pool := recs["pool:0"]
	if pool.Attrs["capacity_tb"] != 20.0 {
		t.Errorf("ёмкость пула = %v, want 20 ТБ", pool.Attrs["capacity_tb"])
	}
	lun := recs["vol:0"]
	if lun.Attrs["thin"] != true || lun.Attrs["lun_id"] != 0 || lun.Attrs["size_gb"] != 107.37 || len(lun.Identifiers["wwn"]) != 32 {
		t.Errorf("LUN = %+v", lun)
	}
	if recs["vol:1"].Attrs["thin"] != false {
		t.Error("ALLOCTYPE=0 -> thin=false")
	}
	if len(pool.Relations) != 3 {
		t.Errorf("пул содержит %d LUN, want 3 (PARENTID — id пула)", len(pool.Relations))
	}
}

// Логаут обязателен на любой ветке (лимит сессий ~32): 10 опросов при лимите 4, в том числе с ошибкой.
func TestExtract_AlwaysLogsOut(t *testing.T) {
	m := huaweimock.Default("0123456789")
	c := newClient(t, m, m.Password)
	for i := 0; i < 5; i++ {
		if _, err := c.Extract(context.Background()); err != nil {
			t.Fatalf("Extract #%d: %v (утечка сессий?)", i+1, err)
		}
	}
	m.Do(func(s *huaweimock.Server) { s.Fail = true })
	for i := 0; i < 5; i++ {
		if items, err := c.Extract(context.Background()); err == nil || items != nil {
			t.Fatalf("Extract при сбое: items=%d err=%v", len(items), err)
		}
	}
	if m.Sessions != 0 || m.Logins != m.Logouts {
		t.Errorf("открытых сессий %d, входов %d, выходов %d, want 0 и равенство", m.Sessions, m.Logins, m.Logouts)
	}
}

func TestTest_ErrorCodeInEnvelope(t *testing.T) {
	m := huaweimock.Default("0123456789")
	if err := newClient(t, m, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if err := newClient(t, m, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "1077949061") {
		t.Fatalf("Test с неверным паролем: %v, want ошибка из error.code (HTTP 200)", err)
	}
}

func TestDiskType(t *testing.T) {
	for code, want := range map[string]string{"14": "nvme", "16": "nvme", "21": "nvme", "3": "ssd", "6": "ssd", "1": "disk", "4": "disk", "19": "disk"} {
		if got := diskType(code); got != want {
			t.Errorf("diskType(%s) = %s, want %s", code, got, want)
		}
	}
}

func wwpns(r canon.Record) []string {
	w, _ := r.Attrs["mapped_wwpns"].([]string)
	return w
}

func TestExtract_HostAssociatedLUNsAndInitiators(t *testing.T) {
	m := huaweimock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	recs := byExt(items)
	if got := wwpns(recs["vol:0"]); len(got) != 1 || got[0] != storage.SampleWWPN("0123456789", 4) {
		t.Errorf("mapped_wwpns LUN 0 = %v, want WWPN инициатора хоста (ID инициатора без разделителей)", got)
	}
	if got := wwpns(recs["vol:1"]); len(got) != 0 {
		t.Errorf("LUN 1 не презентован: %v", got)
	}
}
