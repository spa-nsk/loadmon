package powerstore

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/powerstore/powerstoremock"
	"cmdb/internal/importer/connectors/storage"
)

func newClient(t *testing.T, m *powerstoremock.Server, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("POWERSTORE_TEST_PASSWORD", password)
	c, err := New(connector.Config{Code: "powerstore-test", Kind: Kind, Settings: map[string]any{
		"base_url": srv.URL, "username": m.User, "password_env": "POWERSTORE_TEST_PASSWORD"}})
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

func TestExtract_MapsArrayVolumesAndDrives(t *testing.T) {
	m := powerstoremock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// массив + 2 тома (снимок отфильтрован type=eq.Primary)
	if len(items) != 3 {
		t.Fatalf("записей = %d, want 3", len(items))
	}
	recs := byExt(items)

	a := recs["array:powerstore-test"]
	if a.SerialNumber != "PSSN0123456789" || a.Attrs["array_family"] != "Dell PowerStore PowerStore 1000T" || a.Attrs["mgmt_ip"] != "127.0.0.1" {
		t.Errorf("массив = %+v", a)
	}
	// 2 диска: SAS_FLASH -> ssd, NVMe -> nvme; raw = 1920+3840 = 5760 ГБ = 5.76 ТБ
	if len(a.Components) != 2 || a.Attrs["raw_tb"] != 5.76 {
		t.Errorf("компонентов %d (want 2), raw_tb %v (want 5.76)", len(a.Components), a.Attrs["raw_tb"])
	}
	types := map[string]int{}
	for _, c := range a.Components {
		types[c.Type]++
	}
	if types["ssd"] != 1 || types["nvme"] != 1 {
		t.Errorf("типы компонентов = %v", types)
	}
	// без пулов: том привязан напрямую к массиву
	if len(a.Relations) != 2 {
		t.Errorf("массив должен содержать оба тома напрямую (нет пулов у PowerStore): %+v", a.Relations)
	}

	vol1 := recs["vol:vol1"]
	if vol1.Identifiers["wwn"] == "" || strings.Contains(vol1.Identifiers["wwn"], ":") || vol1.Attrs["thin"] != true || vol1.Attrs["protocol"] != "block" {
		t.Errorf("том 1 = %+v", vol1)
	}
	if _, hasSnap := recs["vol:vol1-snap"]; hasSnap {
		t.Error("снимок тома не должен попадать в снимок CMDB (type=eq.Primary)")
	}
}

func TestExtract_HostVolumeMappingUsesFCInitiatorPort(t *testing.T) {
	m := powerstoremock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	recs := byExt(items)
	got, _ := recs["vol:vol1"].Attrs["mapped_wwpns"].([]string)
	if len(got) != 1 || got[0] != storage.SampleWWPN("0123456789", 1) {
		t.Errorf("mapped_wwpns том 1 = %v, want [%s]", got, storage.SampleWWPN("0123456789", 1))
	}
	got2, _ := recs["vol:vol2"].Attrs["mapped_wwpns"].([]string)
	if len(got2) != 0 {
		t.Errorf("том 2 не презентован: %v", got2)
	}
}

func TestExtract_AuthAndFailure(t *testing.T) {
	m := powerstoremock.Default("0123456789")
	if err := newClient(t, m, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if err := newClient(t, m, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v", err)
	}
	m.Do(func(s *powerstoremock.Server) { s.Fail = true })
	if items, err := newClient(t, m, m.Password).Extract(context.Background()); err == nil || items != nil {
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
