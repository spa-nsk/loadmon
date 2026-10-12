package zvirt

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/zvirt/zvirtmock"
)

func newClient(t *testing.T, m *zvirtmock.Server, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("ZVIRT_TEST_PASSWORD", password)
	c, err := New(connector.Config{Code: "zv-test", Kind: Kind, Settings: map[string]any{
		"base_url": srv.URL, "username": m.User, "password_env": "ZVIRT_TEST_PASSWORD", "manager_name": "zvirt-mgr-01",
	}})
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

func TestExtract_MapsClassesAndRelations(t *testing.T) {
	m := zvirtmock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// менеджер + кластер + 2 хоста + 3 ВМ + 1 data-домен (iso пропущен)
	if len(items) != 8 {
		t.Fatalf("записей = %d, want 8", len(items))
	}
	recs := byExt(items)

	mgr := recs["manager:zv-test"]
	if mgr.Class != "virt_manager" || mgr.SourceName != "zvirt-mgr-01" || mgr.Attrs["version"] != "4.5.5.1" {
		t.Errorf("менеджер = %+v", mgr)
	}
	if len(mgr.Relations) != 1 || mgr.Relations[0].Type != "contains" || mgr.Relations[0].TargetExternalID != "cluster:"+m.Clusters[0].ID {
		t.Errorf("связи менеджера = %+v", mgr.Relations)
	}

	host := recs["host:"+m.Hosts[0].ID]
	if host.Class != "hypervisor_host" || host.SerialNumber != "SN-H1-0123456789" || host.Identifiers["serial_number"] != "SN-H1-0123456789" {
		t.Errorf("хост = %+v", host)
	}

	if host.Attrs["bios_uuid"] != m.Hosts[0].UUID {
		t.Errorf("bios_uuid хоста = %v, want %s (нижний регистр, мост к серверу по UUID)", host.Attrs["bios_uuid"], m.Hosts[0].UUID)
	}

	vm := recs["vm:"+m.VMs[0].ID]
	if vm.Attrs["ram_gb"] != 8 || vm.Attrs["vcpu"] != 4 || vm.Identifiers["instance_uuid"] != m.VMs[0].ID {
		t.Errorf("ВМ = %+v", vm)
	}
	if len(vm.Relations) != 2 {
		t.Errorf("связи включённой ВМ = %+v, want runs_on + member_of", vm.Relations)
	}
	if off := recs["vm:"+m.VMs[2].ID]; len(off.Relations) != 1 || off.Relations[0].Type != "member_of" {
		t.Errorf("выключенная ВМ не должна иметь runs_on: %+v", off.Relations)
	}

	ds := recs["sd:"+m.Domains[0].ID]
	if ds.Attrs["capacity_gb"] != 500.0 || ds.Attrs["ds_type"] != "nfs" {
		t.Errorf("датастор = %+v", ds)
	}
	if _, ok := recs["sd:"+m.Domains[1].ID]; ok {
		t.Error("iso-домен не должен попадать в датасторы")
	}
}

func TestTest_AuthAndFailure(t *testing.T) {
	m := zvirtmock.Default("0123456789")
	if err := newClient(t, m, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test с верным паролем: %v", err)
	}
	if err := newClient(t, m, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: err = %v, want HTTP 401", err)
	}
}

func TestExtract_SourceFailureIsWholeSnapshotError(t *testing.T) {
	m := zvirtmock.Default("0123456789")
	m.Fail = true
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err == nil || items != nil {
		t.Fatalf("Extract при сбое: items=%d err=%v, want ошибка без частичного снимка", len(items), err)
	}
}

func TestNew_RequiresSettings(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без base_url должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"base_url": "http://h", "username": "u", "password_env": "ZVIRT_UNSET_VAR_XYZ"}}); err == nil {
		t.Error("New без переменной окружения с паролем должен вернуть ошибку")
	}
}

// ds_type в БД допускает только vmfs/nfs/vsan: fcp/iscsi не должны попадать туда как есть (CHECK
// отклонил бы запись), а исходное значение сохраняется в вендорских сведениях.
func TestExtract_DatastoreTypeMappedToAllowedValues(t *testing.T) {
	m := zvirtmock.Default("0123456789")
	m.Do(func(s *zvirtmock.Server) {
		s.Domains = append(s.Domains, zvirtmock.Domain{ID: "d1000000-0000-0000-0000-999999999999", Name: "data-fc", Type: "data", Storage: "fcp"})
	})
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	recs := byExt(items)
	fc := recs["sd:d1000000-0000-0000-0000-999999999999"]
	if fc.Attrs["ds_type"] != nil || fc.Vendor["storage_type"] != "fcp" {
		t.Errorf("FC-домен = attrs %+v vendor %+v, want ds_type=nil и storage_type=fcp", fc.Attrs, fc.Vendor)
	}
	if nfs := recs["sd:"+m.Domains[0].ID]; nfs.Attrs["ds_type"] != "nfs" {
		t.Errorf("NFS-домен ds_type = %v", nfs.Attrs["ds_type"])
	}
}
