package brocade

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/brocade/brocademock"
)

func newClient(t *testing.T, m *brocademock.Server, password string, extra map[string]any) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("BROCADE_TEST_PASSWORD", password)
	settings := map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "BROCADE_TEST_PASSWORD"}
	for k, v := range extra {
		settings[k] = v
	}
	c, err := New(connector.Config{Code: "brocade-test", Kind: Kind, Settings: settings})
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

func TestExtract_MapsSwitchPortsAndActiveZones(t *testing.T) {
	m := brocademock.Default("0123456789")
	items, err := newClient(t, m, m.Password, nil).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// коммутатор + 1 активная зона
	if len(items) != 2 {
		t.Fatalf("записей = %d, want 2", len(items))
	}
	if m.Logins != 1 || m.Logouts != 1 {
		t.Errorf("сессия: входов %d, выходов %d, want 1/1", m.Logins, m.Logouts)
	}
	recs := byExt(items)

	sw := recs["switch:brocade-test"]
	if sw.Class != "fc_switch" || sw.SourceName != "sw-0123456789" || sw.SerialNumber != m.WWN {
		t.Errorf("коммутатор = %+v", sw)
	}
	if sw.Attrs["domain_id"] != 5 || sw.Attrs["ports_total"] != 2 {
		t.Errorf("атрибуты коммутатора = %+v", sw.Attrs)
	}
	// 2 порта заведено, но у одного пустой WWPN — недостоверный, должен быть отброшен
	if len(sw.Interfaces) != 1 || sw.Interfaces[0].IfType != "fc" || !sw.Interfaces[0].Enabled {
		t.Errorf("интерфейсы коммутатора = %+v, want один включённый FC-порт", sw.Interfaces)
	}
	if strings.Contains(sw.Interfaces[0].WWPN, ":") == false {
		t.Errorf("WWPN должен быть нормализован (с разделителями): %q", sw.Interfaces[0].WWPN)
	}

	zone := recs["zone:brocade-test:zone_0123456789"]
	if zone.Class != "san_zone" || zone.Attrs["zoneset"] != "prod_cfg" {
		t.Errorf("зона = %+v", zone)
	}
	members, _ := zone.Attrs["members"].([]string)
	if len(members) != 2 {
		t.Errorf("участников зоны = %d, want 2: %v", len(members), members)
	}
}

func TestExtract_FabricNameConfigurable(t *testing.T) {
	m := brocademock.Default("0123456789")
	items, err := newClient(t, m, m.Password, map[string]any{"fabric_name": "fab-prod"}).Extract(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	recs := byExt(items)
	if recs["switch:brocade-test"].Attrs["fabric"] != "fab-prod" {
		t.Errorf("fabric коммутатора = %v, want fab-prod", recs["switch:brocade-test"].Attrs["fabric"])
	}
	if recs["zone:brocade-test:zone_0123456789"].Attrs["fabric"] != "fab-prod" {
		t.Errorf("fabric зоны = %v, want fab-prod", recs["zone:brocade-test:zone_0123456789"].Attrs["fabric"])
	}
}

func TestExtract_AuthAndFailure(t *testing.T) {
	m := brocademock.Default("0123456789")
	if err := newClient(t, m, m.Password, nil).Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if err := newClient(t, m, "wrong", nil).Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v, want HTTP 401", err)
	}
	m.Do(func(s *brocademock.Server) { s.Fail = true })
	if items, err := newClient(t, m, m.Password, nil).Extract(context.Background()); err == nil || items != nil {
		t.Fatalf("Extract при сбое: items=%d err=%v, want ошибка без частичного снимка", len(items), err)
	}
}

func TestNew_RequiresSettings(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без base_url должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"base_url": "https://h"}}); err == nil {
		t.Error("New без username должен вернуть ошибку")
	}
}
