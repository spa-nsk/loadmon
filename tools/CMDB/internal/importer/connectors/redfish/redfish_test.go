package redfish

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/redfish/redfishmock"
)

// bmcs поднимает mock-BMC (по одному httptest-серверу на каждый) и возвращает адреса.
func bmcs(t *testing.T, mocks ...*redfishmock.Server) []any {
	t.Helper()
	var out []any
	for _, m := range mocks {
		srv := httptest.NewServer(m)
		t.Cleanup(srv.Close)
		out = append(out, srv.URL)
	}
	return out
}

func newClient(t *testing.T, targets []any, extra map[string]any) connector.Connector {
	t.Helper()
	t.Setenv("REDFISH_TEST_PASSWORD", "secret")
	cfg := map[string]any{"targets": targets, "username": "root", "password_env": "REDFISH_TEST_PASSWORD"}
	for k, v := range extra {
		cfg[k] = v
	}
	c, err := New(connector.Config{Code: "rf-test", Kind: Kind, Settings: cfg})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func types(comps []canon.Component) map[string]int {
	m := map[string]int{}
	for _, c := range comps {
		m[c.Type]++
	}
	return m
}

func TestExtract_MapsServerComponentsAndInterfaces(t *testing.T) {
	m := redfishmock.New("0123456789", 1)
	items, err := newClient(t, bmcs(t, m), nil).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 1 || items[0].Err != nil {
		t.Fatalf("items = %+v", items)
	}
	r := items[0].Record
	if r.Class != "server" || r.SourceName != m.HostName || r.SerialNumber != m.Serial || !strings.HasPrefix(r.ExternalID, "bmc:127.0.0.1:") {
		t.Errorf("запись = %+v", r)
	}
	if r.Identifiers["bios_uuid"] != m.UUID || r.Identifiers["serial_number"] != m.Serial || r.Identifiers["bmc_mac"] != m.BMCMAC {
		t.Errorf("идентификаторы = %+v, want bios_uuid/serial/bmc_mac", r.Identifiers)
	}
	a := r.Attrs
	if a["cpu_count"] != 2 || a["core_count"] != 40 || a["ram_gb"] != 64 || a["form_factor"] != "rack" || a["bmc_ip"] != "127.0.0.1" ||
		a["firmware"] != "U30 v2.72" || a["bmc_name"] != m.BMCHostName {
		t.Errorf("атрибуты = %+v", a)
	}

	got := types(r.Components)
	if got["cpu"] != 2 || got["dimm"] != 2 || got["raid_ctrl"] != 1 || got["disk"] != 1 || got["nvme"] != 1 || got["psu"] != 2 || got["fan"] != 2 || got["bmc"] != 1 {
		t.Errorf("типы компонентов = %v", got)
	}
	for _, c := range r.Components {
		if c.Type == "nvme" || c.Type == "disk" {
			if c.ParentKey != "raid:1" || c.CapacityUnit != "GB" {
				t.Errorf("диск %+v: родитель должен быть контроллером, ёмкость в GB", c)
			}
		}
		if c.Type == "dimm" && (c.CapacityValue == nil || *c.CapacityValue != 32 || c.SerialNumber == "") {
			t.Errorf("DIMM = %+v", c)
		}
	}
	if len(r.Interfaces) != 3 {
		t.Fatalf("интерфейсов %d, want 3 (2 порта + BMC)", len(r.Interfaces))
	}
	bmc := 0
	for _, i := range r.Interfaces {
		if i.Role == "bmc" {
			bmc++
			if i.MAC != strings.ToUpper(m.BMCMAC) && i.MAC != m.BMCMAC {
				t.Errorf("MAC BMC = %q", i.MAC)
			}
		}
	}
	if bmc != 1 {
		t.Errorf("интерфейсов BMC %d, want 1", bmc)
	}
}

func TestExtract_AbsentDIMMSkippedAndTemporaryName(t *testing.T) {
	m := redfishmock.New("0123456789", 1)
	m.Do(func(s *redfishmock.Server) {
		s.HostName = ""
		s.DIMMs[1].Absent = true
	})
	items, err := newClient(t, bmcs(t, m), nil).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	r := items[0].Record
	if !strings.HasPrefix(r.SourceName, "HPE-SN-RF-") {
		t.Errorf("имя сервера без HostName = %q, want временное по производителю и серийному номеру", r.SourceName)
	}
	if types(r.Components)["dimm"] != 1 {
		t.Errorf("пустой слот памяти не должен быть компонентом: %v", types(r.Components))
	}
}

func TestExtract_UnreachableBMCIsRetainedNotFatal(t *testing.T) {
	good, bad := redfishmock.New("0123456789", 1), redfishmock.New("0123456789", 2)
	bad.Fail = true
	items, err := newClient(t, bmcs(t, good, bad), nil).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 2 || items[0].Err != nil || items[1].Err == nil || !items[1].Retain || items[1].Record.ExternalID == "" {
		t.Fatalf("items = %+v, want второй BMC — ошибка с Retain и ExternalID", items)
	}
}

func TestExtract_AllBMCsDownIsSnapshotError(t *testing.T) {
	a, b := redfishmock.New("0123456789", 1), redfishmock.New("0123456789", 2)
	a.Fail, b.Fail = true, true
	items, err := newClient(t, bmcs(t, a, b), nil).Extract(context.Background())
	if err == nil || items != nil {
		t.Fatalf("Extract: items=%d err=%v, want ошибка снимка, когда не отвечает ни один BMC", len(items), err)
	}
}

// Сбой подресурса (не 404) делает список компонентов недостоверным: он не должен передаваться,
// иначе конвейер снял бы живые компоненты. 404 — вендор ресурс не реализует, это норма.
func TestExtract_SubresourceFailureWithholdsComponents(t *testing.T) {
	m := redfishmock.New("0123456789", 1)
	m.Do(func(s *redfishmock.Server) { s.FailPath = "/Memory" })
	items, err := newClient(t, bmcs(t, m), nil).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	r := items[0].Record
	if r.Components != nil || r.Interfaces != nil {
		t.Errorf("при сбое подресурса компоненты/интерфейсы должны быть nil: %d/%d", len(r.Components), len(r.Interfaces))
	}
	if r.Attrs["cpu_count"] != 2 {
		t.Errorf("сводные атрибуты при этом сохраняются: %+v", r.Attrs)
	}
}

func TestTest_AndConfigValidation(t *testing.T) {
	m := redfishmock.New("0123456789", 1)
	if err := newClient(t, bmcs(t, m), nil).Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	t.Setenv("REDFISH_TEST_PASSWORD", "wrong")
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	c, _ := New(connector.Config{Code: "x", Settings: map[string]any{"targets": []any{srv.URL}, "username": "root", "password_env": "REDFISH_TEST_PASSWORD"}})
	if err := c.Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v, want 401", err)
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без targets должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"targets": []any{"10.0.0.1", "10.0.0.1"}, "username": "u", "password_env": "REDFISH_TEST_PASSWORD"}}); err == nil {
		t.Error("New с повторяющейся целью должен вернуть ошибку")
	}
}
