package snmpnet

import (
	"context"
	"testing"

	"github.com/gosnmp/gosnmp"

	"cmdb/internal/importer/connector"
)

// fakeFetcher — заглушка fetcher для теста: протокольный уровень SNMP (USM, ASN.1) не эмулируется,
// проверяется только сведение PDU -> canon (см. комментарий пакета snmpnet.go).
type fakeFetcher struct {
	byOID map[string][]gosnmp.SnmpPDU
	err   map[string]error
}

func (f *fakeFetcher) Connect() error { return nil }
func (f *fakeFetcher) Close() error   { return nil }
func (f *fakeFetcher) WalkAll(oid string) ([]gosnmp.SnmpPDU, error) {
	if err := f.err[oid]; err != nil {
		return nil, err
	}
	return f.byOID[oid], nil
}

func defaultFixture() *fakeFetcher {
	return &fakeFetcher{err: map[string]error{}, byOID: map[string][]gosnmp.SnmpPDU{
		oidSystem: {
			{Name: ".1.3.6.1.2.1.1.5.0", Value: []byte("sw1-test")},
			{Name: ".1.3.6.1.2.1.1.1.0", Value: []byte("Eltex MES3324 v3.4.2")},
		},
		oidEntity: {
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.5.1", Value: 3}, // chassis
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.11.1", Value: []byte("SN12345")},
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.13.1", Value: []byte("MES3324")},
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.5.2", Value: 6}, // PSU
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.11.2", Value: []byte("PSU001")},
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.7.2", Value: []byte("PSU 1")},
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.5.3", Value: 7}, // fan
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.11.3", Value: []byte("FAN001")},
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.5.4", Value: 9}, // module, без серийного номера
			{Name: ".1.3.6.1.2.1.47.1.1.1.1.13.4", Value: []byte("SFP-1G")},
		},
		oidIfTable: {
			{Name: ".1.3.6.1.2.1.2.2.1.2.1", Value: []byte("Gi0/1")},
			{Name: ".1.3.6.1.2.1.2.2.1.5.1", Value: 1000000000},
			{Name: ".1.3.6.1.2.1.2.2.1.6.1", Value: []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}},
			{Name: ".1.3.6.1.2.1.2.2.1.8.1", Value: 1}, // up
			{Name: ".1.3.6.1.2.1.2.2.1.2.2", Value: []byte("Gi0/2")},
			{Name: ".1.3.6.1.2.1.2.2.1.5.2", Value: 100000000},
			{Name: ".1.3.6.1.2.1.2.2.1.8.2", Value: 2}, // down
		},
		oidLldpRem: {
			{Name: ".1.0.8802.1.1.2.1.4.1.1.9.0.1.1", Value: []byte("core-sw")},
			{Name: ".1.0.8802.1.1.2.1.4.1.1.7.0.1.1", Value: []byte("Gi0/24")},
		},
	}}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()
	t.Setenv("SNMP_TEST_AUTH", "authpass123")
	t.Setenv("SNMP_TEST_PRIV", "privpass123")
	c, err := New(connector.Config{Code: "snmp-test", Kind: Kind, Settings: map[string]any{
		"host": "10.0.0.1", "username": "monitor", "auth_pass_env": "SNMP_TEST_AUTH", "priv_pass_env": "SNMP_TEST_PRIV",
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c.(*Client)
}

func TestExtract_MapsDeviceComponentsInterfacesAndLLDP(t *testing.T) {
	c := newTestClient(t)
	f := defaultFixture()
	items, err := c.extract(f)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("записей = %d, want 1 (устройство)", len(items))
	}
	rec := items[0].Record
	if rec.Class != "net_switch" || rec.SourceName != "sw1-test" || rec.ExternalID != "device:snmp-test" {
		t.Errorf("устройство = %+v", rec)
	}
	if rec.SerialNumber != "SN12345" || rec.Identifiers["serial_number"] != "SN12345" {
		t.Errorf("серийный номер (chassis) = %q", rec.SerialNumber)
	}
	if rec.Attrs["os_version"] != "Eltex MES3324 v3.4.2" {
		t.Errorf("os_version = %v", rec.Attrs["os_version"])
	}
	if rec.Vendor["model"] != "MES3324" {
		t.Errorf("model = %v", rec.Vendor["model"])
	}

	if len(rec.Components) != 3 {
		t.Fatalf("компонентов = %d, want 3 (БП, вентилятор, модуль)", len(rec.Components))
	}
	types := map[string]int{}
	for _, comp := range rec.Components {
		types[comp.Type]++
	}
	if types["psu"] != 1 || types["fan"] != 1 || types["module"] != 1 {
		t.Errorf("типы компонентов = %v", types)
	}

	if len(rec.Interfaces) != 2 {
		t.Fatalf("интерфейсов = %d, want 2", len(rec.Interfaces))
	}
	byName := map[string]int{}
	for i, intf := range rec.Interfaces {
		byName[intf.Name] = i
	}
	up := rec.Interfaces[byName["Gi0/1"]]
	if up.MAC != "aa:bb:cc:dd:ee:ff" || up.SpeedMbps != 1000 || !up.Enabled {
		t.Errorf("Gi0/1 = %+v", up)
	}
	down := rec.Interfaces[byName["Gi0/2"]]
	if down.Enabled {
		t.Errorf("Gi0/2 должен быть выключен: %+v", down)
	}

	neighbors, _ := rec.Vendor["lldp_neighbors"].([]map[string]any)
	if len(neighbors) != 1 || neighbors[0]["remote_system"] != "core-sw" || neighbors[0]["remote_port"] != "Gi0/24" {
		t.Errorf("lldp_neighbors = %+v", neighbors)
	}
}

func TestNew_RequiresSettings(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без host должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"host": "10.0.0.1"}}); err == nil {
		t.Error("New без username должен вернуть ошибку")
	}
	t.Setenv("SNMP_TEST_AUTH2", "a")
	t.Setenv("SNMP_TEST_PRIV2", "p")
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{
		"host": "10.0.0.1", "username": "u", "auth_pass_env": "SNMP_TEST_AUTH2", "priv_pass_env": "SNMP_TEST_PRIV2", "device_class": "vm",
	}}); err == nil {
		t.Error("New с недопустимым device_class должен вернуть ошибку")
	}
}

func TestExtract_SourceFailureIsWholeSnapshotError(t *testing.T) {
	c := newTestClient(t)
	f := defaultFixture()
	f.err[oidIfTable] = context.DeadlineExceeded
	if items, err := c.extract(f); err == nil || items != nil {
		t.Fatalf("extract при сбое: items=%d err=%v, want ошибка без частичного снимка", len(items), err)
	}
}
