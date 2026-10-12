package vnx

import (
	"context"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/storage"
	"cmdb/internal/importer/connectors/vnx/vnxmock"
)

func setup(t *testing.T) (*vnxmock.Array, connector.Connector) {
	t.Helper()
	a := vnxmock.Default(t.TempDir(), "0123456789")
	if err := a.Write(); err != nil {
		t.Fatalf("mock: %v", err)
	}
	c, err := New(connector.Config{Code: "vnx-test", Kind: Kind, Settings: map[string]any{
		"host": "10.0.0.5", "binary": a.Script(), "secfile_path": "/etc/cmdb/vnx.sec"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return a, c
}

func byExt(items []canon.Item) map[string]canon.Record {
	m := map[string]canon.Record{}
	for _, it := range items {
		m[it.Record.ExternalID] = it.Record
	}
	return m
}

func TestExtract_RunsRealProcessAndParsesOutput(t *testing.T) {
	a, c := setup(t)
	items, err := c.Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 5 { // массив + пул + 3 LUN
		t.Fatalf("записей = %d, want 5", len(items))
	}
	recs := byExt(items)
	arr := recs["array:vnx-test"]
	if arr.SourceName != a.Name || arr.SerialNumber != a.Serial || arr.Attrs["array_family"] != "Dell EMC VNX5300" || arr.Attrs["microcode"] != "05.33.009.5.155" {
		t.Errorf("массив = %+v", arr)
	}
	types := map[string]int{}
	for _, c := range arr.Components {
		types[c.Type]++
	}
	// полка + 3 диска (пустой слот Removed пропущен): SAS, NL SAS -> disk, Flash -> ssd
	if types["enclosure"] != 1 || types["disk"] != 2 || types["ssd"] != 1 || len(arr.Components) != 4 {
		t.Errorf("компоненты = %v, want 1 полка, 2 disk, 1 ssd (пустой слот пропущен)", types)
	}
	for _, c := range arr.Components {
		if c.Type == "ssd" && (c.ParentKey != "enc:b0e0" || c.CapacityValue == nil || *c.CapacityValue != 799) { // 762000 МиБ ≈ 799 ГБ
			t.Errorf("SSD = %+v", c)
		}
	}
	lun := recs["vol:0"]
	if lun.Attrs["lun_id"] != 0 || lun.Attrs["thin"] != true || strings.Contains(lun.Identifiers["wwn"], ":") || len(lun.Identifiers["wwn"]) != 32 {
		t.Errorf("LUN = %+v", lun)
	}
	if lun.Attrs["size_gb"] != 107.37 { // 100 «GBs» VNX = ГиБ
		t.Errorf("size_gb = %v, want 107.37 (GiB -> ГБ)", lun.Attrs["size_gb"])
	}
	if len(recs["pool:0"].Relations) != 2 { // два LUN в пуле
		t.Errorf("пул содержит %d LUN, want 2", len(recs["pool:0"].Relations))
	}
	rgContained := false
	for _, r := range arr.Relations {
		if r.TargetExternalID == "vol:2" {
			rgContained = true // LUN RAID-группы (Pool Name N/A) принадлежит массиву напрямую
		}
	}
	if !rgContained {
		t.Errorf("LUN вне пула должен принадлежать массиву: %+v", arr.Relations)
	}
	calls := strings.Join(a.Calls(), "\n")
	if !strings.Contains(calls, "-h 10.0.0.5 -secfilepath /etc/cmdb/vnx.sec") || strings.Contains(calls, "-Password") {
		t.Errorf("вызовы = %q, want -h и -secfilepath без пароля в аргументах", calls)
	}
}

func TestExtract_CommandFailureIsSnapshotError(t *testing.T) {
	a, c := setup(t)
	a.Fail = true
	if err := a.Write(); err != nil {
		t.Fatal(err)
	}
	items, err := c.Extract(context.Background())
	if err == nil || items != nil || !strings.Contains(err.Error(), "could not connect") {
		t.Fatalf("Extract при сбое: items=%d err=%v, want ошибка со stderr команды", len(items), err)
	}
	if err := c.Test(context.Background()); err == nil {
		t.Error("Test при недоступном массиве должен вернуть ошибку")
	}
}

func TestNew_Validation(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без host должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"host": "h"}}); err == nil {
		t.Error("New без secfile_path и username должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"host": "h", "username": "u", "password_env": "VNX_UNSET_XYZ"}}); err == nil {
		t.Error("New без переменной окружения с паролем должен вернуть ошибку")
	}
}

func TestParseBlocks(t *testing.T) {
	b := parseBlocks([]byte("Bus 1 Enclosure 2  Disk B3\nVendor Id:  X\r\nSerial Number:  A:B\n\n\nLOGICAL UNIT NUMBER 7\nName:  n\n"))
	if len(b) != 2 || b[0].header != "Bus 1 Enclosure 2  Disk B3" || b[0].get("serial number") != "A:B" || b[1].header != "LOGICAL UNIT NUMBER 7" {
		t.Errorf("blocks = %+v", b)
	}
}

func wwpns(r canon.Record) []string {
	w, _ := r.Attrs["mapped_wwpns"].([]string)
	return w
}

func TestExtract_StorageGroupsGiveHBAsPerLUN(t *testing.T) {
	_, c := setup(t)
	items, err := c.Extract(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	recs := byExt(items)
	for _, id := range []string{"vol:0", "vol:1"} { // ALU 0 и 1 в группе
		if got := wwpns(recs[id]); len(got) != 1 || got[0] != storage.SampleWWPN("0123456789", 6) {
			t.Errorf("mapped_wwpns %s = %v, want порт HBA из «WWNN:WWPN»", id, got)
		}
	}
	if got := wwpns(recs["vol:2"]); len(got) != 0 {
		t.Errorf("LUN вне группы: %v", got)
	}
}
