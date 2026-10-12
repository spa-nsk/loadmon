package unity

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/storage"
	"cmdb/internal/importer/connectors/unity/unitymock"
)

func newClient(t *testing.T, m *unitymock.Server, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("UNITY_TEST_PASSWORD", password)
	c, err := New(connector.Config{Code: "unity-test", Kind: Kind, Settings: map[string]any{
		"base_url": srv.URL, "username": m.User, "password_env": "UNITY_TEST_PASSWORD"}})
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

func TestExtract_MapsArrayPoolsLUNsAndFileSystems(t *testing.T) {
	m := unitymock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// массив + 2 пула + NAS-сервер + 3 LUN + ФС + 2 ресурса
	if len(items) != 10 {
		t.Fatalf("записей = %d, want 10", len(items))
	}
	if m.Logins != 1 || m.Logouts != 1 {
		t.Errorf("сессий открыто %d, закрыто %d, want 1/1 (cookie удерживает одну сессию, выход с CSRF-токеном)", m.Logins, m.Logouts)
	}
	recs := byExt(items)

	a := recs["array:unity-test"]
	if a.SerialNumber != "UNSN0123456789" || a.Attrs["array_family"] != "Dell Unity Unity 480F" || a.Attrs["microcode"] != "5.3.0.0.5.120" ||
		a.Attrs["controllers"] != 2 || a.Attrs["mgmt_ip"] != "127.0.0.1" {
		t.Errorf("массив = %+v", a)
	}
	// 2 SP + DPE + DAE + 3 диска; raw = 1900+1900+4000 = 7.8 ТБ
	if len(a.Components) != 7 || a.Attrs["raw_tb"] != 7.8 {
		t.Errorf("компонентов %d (want 7), raw_tb %v (want 7.8)", len(a.Components), a.Attrs["raw_tb"])
	}
	types := map[string]int{}
	for _, c := range a.Components {
		types[c.Type]++
	}
	if types["controller"] != 2 || types["enclosure"] != 2 || types["ssd"] != 2 || types["disk"] != 1 {
		t.Errorf("типы компонентов = %v", types)
	}

	lun := recs["vol:lun:sv_1"]
	if lun.Identifiers["wwn"] == "" || strings.Contains(lun.Identifiers["wwn"], ":") || lun.Attrs["thin"] != true {
		t.Errorf("LUN = %+v", lun)
	}
	if recs["vol:lun:sv_2"].Attrs["thin"] != false {
		t.Error("isThinEnabled=false должен дать thin=false, а не nil")
	}
	if len(recs["pool:pool_1"].Relations) != 2 { // sv_1, sv_2
		t.Errorf("пул 1 содержит %d томов, want 2", len(recs["pool:pool_1"].Relations))
	}
	fs := recs["vol:fs:fs_1"]
	if fs.Attrs["protocol"] != "nas" {
		t.Errorf("ФС = %+v", fs)
	}
	if recs["share:smb:SMBShare_1"].Attrs["share_type"] != "smb" || recs["share:nfs:NFSShare_1"].Attrs["share_type"] != "nfs" {
		t.Error("типы файловых ресурсов")
	}
}

func TestExtract_AuthAndFailure(t *testing.T) {
	m := unitymock.Default("0123456789")
	if err := newClient(t, m, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if err := newClient(t, m, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v", err)
	}
	m.Do(func(s *unitymock.Server) { s.Fail = true })
	if items, err := newClient(t, m, m.Password).Extract(context.Background()); err == nil || items != nil {
		t.Fatalf("Extract при сбое: items=%d err=%v", len(items), err)
	}
	if m.Logouts != m.Logins {
		t.Errorf("сессия должна закрываться и при ошибке снимка: входов %d, выходов %d", m.Logins, m.Logouts)
	}
}

func wwpns(r canon.Record) []string {
	w, _ := r.Attrs["mapped_wwpns"].([]string)
	return w
}

func TestExtract_HostLUNMappingUsesFCInitiatorPort(t *testing.T) {
	m := unitymock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	recs := byExt(items)
	if got := wwpns(recs["vol:lun:sv_1"]); len(got) != 1 || got[0] != storage.SampleWWPN("0123456789", 3) {
		t.Errorf("mapped_wwpns sv_1 = %v, want порт из «WWNN:WWPN» без IQN", got)
	}
	if got := wwpns(recs["vol:lun:sv_2"]); len(got) != 0 {
		t.Errorf("sv_2 не презентован: %v", got)
	}
}
