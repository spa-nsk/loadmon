package oneview

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/oneview/oneviewmock"
)

func newClient(t *testing.T, m *oneviewmock.Server, extra map[string]any, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("ONEVIEW_TEST_PASSWORD", password)
	settings := map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "ONEVIEW_TEST_PASSWORD"}
	for k, v := range extra {
		settings[k] = v
	}
	c, err := New(connector.Config{Code: "ov-test", Kind: Kind, Settings: settings})
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

func TestExtract_MapsServersEnclosuresAndPages(t *testing.T) {
	m := oneviewmock.Default("0123456789")
	items, err := newClient(t, m, nil, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 4 { // 1 корзина + 3 сервера; сервер отдаёт по 2 записи на страницу
		t.Fatalf("записей = %d, want 4", len(items))
	}
	if m.Logins != 1 || m.Logouts != 1 {
		t.Errorf("сессия: входов %d, выходов %d, want 1/1", m.Logins, m.Logouts)
	}
	recs := byExt(items)

	enc := recs["enc:enc1"]
	if enc.Class != "blade_chassis" || enc.Attrs["bays_total"] != 16 || enc.Attrs["bays_used"] != 2 || enc.Attrs["oa_ip"] != "10.0.0.5" {
		t.Errorf("корзина = %+v", enc)
	}
	if len(enc.Relations) != 2 || enc.Relations[0].Type != "contains" {
		t.Errorf("корзина должна содержать 2 блейда: %+v", enc.Relations)
	}

	blade := recs["sh:sh1"]
	if blade.SourceName != "esx-blade-01-0123456789" {
		t.Errorf("имя сервера с профилем = %q, want имя профиля", blade.SourceName)
	}
	if blade.Attrs["form_factor"] != "blade" || blade.Attrs["bay"] != "1" || blade.Attrs["ram_gb"] != 128 ||
		blade.Attrs["cpu_count"] != 2 || blade.Attrs["core_count"] != 24 || blade.Attrs["bmc_ip"] != "10.0.1.1" {
		t.Errorf("блейд = %+v", blade.Attrs)
	}
	if blade.Identifiers["bios_uuid"] != strings.ToLower(blade.Identifiers["bios_uuid"]) || blade.Identifiers["serial_number"] != "SN-SRV1-0123456789" {
		t.Errorf("идентификаторы блейда = %+v", blade.Identifiers)
	}
	if _, has := blade.Vendor["processor_speed"]; has {
		t.Error("частота процессора недостоверна (harness/02) и не должна попадать в вендорские сведения")
	}

	rack := recs["sh:sh3"]
	if rack.SourceName != "rack-dl380-0123456789" || rack.Attrs["form_factor"] != "rack" || rack.Attrs["bay"] != nil {
		t.Errorf("стоечный сервер = %+v", rack)
	}
}

func TestTest_VersionAndAuth(t *testing.T) {
	m := oneviewmock.Default("0123456789")
	if err := newClient(t, m, nil, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test с верными данными: %v", err)
	}
	if err := newClient(t, m, nil, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v, want HTTP 401", err)
	}
	if err := newClient(t, m, map[string]any{"api_version": "800"}, m.Password).Test(context.Background()); err == nil || !strings.Contains(err.Error(), "406") {
		t.Fatalf("Test с неподдерживаемой версией: %v, want HTTP 406", err)
	}
}

func TestExtract_FailureIsWholeSnapshotError(t *testing.T) {
	m := oneviewmock.Default("0123456789")
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
}
