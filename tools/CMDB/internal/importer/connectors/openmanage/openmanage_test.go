package openmanage

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/openmanage/openmanagemock"
)

func newClient(t *testing.T, m *openmanagemock.Server, extra map[string]any, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("OPENMANAGE_TEST_PASSWORD", password)
	settings := map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "OPENMANAGE_TEST_PASSWORD"}
	for k, v := range extra {
		settings[k] = v
	}
	c, err := New(connector.Config{Code: "ome-test", Kind: Kind, Settings: settings})
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

func TestExtract_MapsServersChassisAndPages(t *testing.T) {
	m := openmanagemock.Default("0123456789")
	items, err := newClient(t, m, nil, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 3 { // 1 шасси + 2 сервера; список отдаёт по 2 записи на страницу (PageCap)
		t.Fatalf("записей = %d, want 3", len(items))
	}
	if m.Logins != 1 || m.Logouts != 1 {
		t.Errorf("сессия: входов %d, выходов %d, want 1/1", m.Logins, m.Logouts)
	}
	recs := byExt(items)

	chassis := recs["chassis:CH-0123456789"]
	if chassis.Class != "blade_chassis" || chassis.SerialNumber != "CH-0123456789" {
		t.Errorf("шасси = %+v", chassis)
	}
	if len(chassis.Relations) != 1 || chassis.Relations[0].Type != "contains" || chassis.Relations[0].TargetExternalID != "srv:BL-0123456789" {
		t.Errorf("шасси должно содержать блейд: %+v", chassis.Relations)
	}

	blade := recs["srv:BL-0123456789"]
	if blade.Class != "server" || blade.SourceName != "blade-0123456789" || blade.SerialNumber != "BL-0123456789" {
		t.Errorf("блейд = %+v", blade)
	}
	if blade.Attrs["form_factor"] != "blade" || blade.Attrs["bay"] != "1" {
		t.Errorf("блейд: форм-фактор/слот = %+v", blade.Attrs)
	}
	if blade.Attrs["cpu_count"] != 2 || blade.Attrs["core_count"] != 64 || blade.Attrs["ram_gb"] != 32 || blade.Attrs["bmc_ip"] != "10.0.2.1" {
		t.Errorf("блейд: атрибуты = %+v", blade.Attrs)
	}
	if blade.Attrs["os_distro"] != "VMware ESXi 8.0" || blade.Attrs["hypervisor_role"] != "vmware_esxi" {
		t.Errorf("блейд: определение гипервизора по ОС = %+v", blade.Attrs)
	}
	if blade.Attrs["firmware"] != "2.10.2" {
		t.Errorf("блейд: версия BIOS = %v, want 2.10.2", blade.Attrs["firmware"])
	}
	wantComps := map[string]string{"cpu:1": "cpu", "cpu:2": "cpu", "dimm:MEM-0123456789-1": "dimm", "drive:DSK-0123456789-1": "ssd", "psu:PSU-0123456789-1": "psu"}
	if len(blade.Components) != len(wantComps) {
		t.Fatalf("компонентов блейда = %d, want %d: %+v", len(blade.Components), len(wantComps), blade.Components)
	}
	for _, c := range blade.Components {
		if wantComps[c.Key] != c.Type {
			t.Errorf("компонент %q: тип = %q, want %q", c.Key, c.Type, wantComps[c.Key])
		}
	}

	rack := recs["srv:RK-0123456789"]
	if rack.Attrs["form_factor"] != "rack" {
		t.Errorf("стоечный сервер: form_factor = %v, want rack", rack.Attrs["form_factor"])
	}
	if _, hasBay := rack.Attrs["bay"]; hasBay {
		t.Error("у стоечного сервера не должно быть bay")
	}
	if rack.Attrs["os_family"] != "Linux" {
		t.Errorf("стоечный сервер: os_family = %v, want Linux (Red Hat Enterprise Linux 9)", rack.Attrs["os_family"])
	}
	for _, c := range rack.Components {
		if c.Type == "disk" && c.CapacityValue != nil && *c.CapacityValue != 2000 {
			t.Errorf("диск: ёмкость = %v ГБ, want 2000", *c.CapacityValue)
		}
	}
}

func TestTest_Auth(t *testing.T) {
	m := openmanagemock.Default("0123456789")
	if err := newClient(t, m, nil, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test с верными данными: %v", err)
	}
	if err := newClient(t, m, nil, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v, want HTTP 401", err)
	}
}

func TestExtract_FailureIsWholeSnapshotError(t *testing.T) {
	m := openmanagemock.Default("0123456789")
	m.Fail = true
	items, err := newClient(t, m, nil, m.Password).Extract(context.Background())
	if err == nil || items != nil {
		t.Fatalf("Extract при сбое: items=%d err=%v, want ошибка без частичного снимка", len(items), err)
	}
	if m.Logouts != 1 {
		t.Errorf("сессия должна закрываться и при ошибке снимка: выходов %d", m.Logouts)
	}
}

func TestNew_RequiresSettings(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без base_url должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"base_url": "http://x"}}); err == nil {
		t.Error("New без username должен вернуть ошибку")
	}
}
