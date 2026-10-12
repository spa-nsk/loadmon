package vcenter

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/vmware/govmomi/simulator"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
)

// startSim поднимает симулятор vSphere (govmomi vcsim): 1 ЦОД, кластер из 3 хостов, одиночный
// хост, 4 ВМ, 1 датастор.
func startSim(t *testing.T) (baseURL, user, pass string) {
	t.Helper()
	model := simulator.VPX()
	if err := model.Create(); err != nil {
		t.Fatalf("vcsim: %v", err)
	}
	t.Cleanup(model.Remove)
	// свои учётные данные: с DefaultLogin симулятор принимает любой непустой пароль
	model.Service.Listen = &url.URL{User: url.UserPassword("vcuser", "vcpass")}
	srv := model.Service.NewServer()
	t.Cleanup(srv.Close)
	pw, _ := srv.URL.User.Password()
	return (&url.URL{Scheme: srv.URL.Scheme, Host: srv.URL.Host}).String(), srv.URL.User.Username(), pw
}

func newClient(t *testing.T, base, user, pass string) connector.Connector {
	t.Helper()
	t.Setenv("VCENTER_TEST_PASSWORD", pass)
	c, err := New(connector.Config{Code: "vc-test", Kind: Kind, Settings: map[string]any{
		"base_url": base, "username": user, "password_env": "VCENTER_TEST_PASSWORD", "manager_name": "vc-mgr", "insecure_skip_verify": true,
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func count(items []canon.Item, class string) int {
	n := 0
	for _, it := range items {
		if it.Record.Class == class {
			n++
		}
	}
	return n
}

func TestExtract_AgainstSimulator(t *testing.T) {
	base, user, pass := startSim(t)
	c := newClient(t, base, user, pass)
	if err := c.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	items, err := c.Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if count(items, "virt_manager") != 1 || count(items, "cluster") != 1 || count(items, "hypervisor_host") != 4 ||
		count(items, "vm") != 4 || count(items, "datastore") != 1 {
		t.Fatalf("состав снимка: manager=%d cluster=%d host=%d vm=%d ds=%d, want 1/1/4/4/1",
			count(items, "virt_manager"), count(items, "cluster"), count(items, "hypervisor_host"), count(items, "vm"), count(items, "datastore"))
	}

	byExt := map[string]canon.Record{}
	for _, it := range items {
		byExt[it.Record.ExternalID] = it.Record
	}
	mgr := byExt["manager:vc-test"]
	if len(mgr.Relations) != 2 { // кластер + одиночный хост
		t.Errorf("менеджер содержит %d объектов, want 2 (кластер и одиночный хост): %+v", len(mgr.Relations), mgr.Relations)
	}
	if !strings.Contains(mgr.Attrs["product"].(string), "vCenter") || !strings.Contains(mgr.Attrs["version"].(string), "build") {
		t.Errorf("менеджер = %+v", mgr.Attrs)
	}

	for _, it := range items {
		switch it.Record.Class {
		case "cluster":
			// vsan_enabled=false — не заглушка: реальный вызов VsanClusterGetConfig против vcsim
			// (симулятор не включает vSAN ни на одном кластере), см. TestBuildRecords_VsanAndFreeSpace
			// для случая vsan_enabled=true, который на симуляторе не воспроизвести.
			if it.Record.Attrs["vsan_enabled"] != false {
				t.Errorf("кластер %s: vsan_enabled = %v, want false (vcsim не включает vSAN)", it.Record.SourceName, it.Record.Attrs["vsan_enabled"])
			}
		case "datastore":
			if fg, ok := it.Record.Attrs["free_gb"].(float64); !ok || fg <= 0 {
				t.Errorf("датастор %s: free_gb = %v, want > 0", it.Record.SourceName, it.Record.Attrs["free_gb"])
			}
		}
	}

	var inCluster, standalone int
	for _, it := range items {
		r := it.Record
		switch r.Class {
		case "hypervisor_host":
			if r.SerialNumber == "" || r.Identifiers["serial_number"] != r.SerialNumber {
				t.Errorf("хост %s без серийного номера (ServiceTag)", r.SourceName)
			}
			if r.Attrs["hv_type"] != "esxi" || !strings.HasPrefix(r.Attrs["hv_version"].(string), "8.") {
				t.Errorf("хост %s = %+v", r.SourceName, r.Attrs)
			}
			if len(r.Relations) == 1 {
				inCluster++
			} else {
				standalone++
			}
		case "vm":
			if r.Identifiers["instance_uuid"] == "" || r.Identifiers["bios_uuid"] == "" || r.Attrs["vcpu"] != 1 {
				t.Errorf("ВМ %s = ids %+v attrs %+v", r.SourceName, r.Identifiers, r.Attrs)
			}
			types := map[string]int{}
			for _, rel := range r.Relations {
				types[rel.Type]++
			}
			if types["runs_on"] != 1 || types["uses_storage"] != 1 {
				t.Errorf("связи ВМ %s = %+v, want runs_on и uses_storage", r.SourceName, r.Relations)
			}
		}
	}
	if inCluster != 3 || standalone != 1 {
		t.Errorf("хостов в кластере %d, одиночных %d, want 3/1", inCluster, standalone)
	}
}

// TestExtract_AgainstStandaloneESXi — plan.md E8.3: тот же коннектор без изменений в коде против
// одиночного хоста ESXi напрямую, без vCenter/VCF (govmomi обслуживает оба случая одним и тем же
// API — «vmware» из плана не требует отдельного коннектора, только другой endpoint в config).
// simulator.ESX() — официальная заготовка govmomi именно для этого случая: 1 хост, без кластера,
// без ЦОД-специфичной структуры vCenter.
func TestExtract_AgainstStandaloneESXi(t *testing.T) {
	model := simulator.ESX()
	if err := model.Create(); err != nil {
		t.Fatalf("vcsim (ESX): %v", err)
	}
	t.Cleanup(model.Remove)
	model.Service.Listen = &url.URL{User: url.UserPassword("root", "esxpass")}
	srv := model.Service.NewServer()
	t.Cleanup(srv.Close)
	pw, _ := srv.URL.User.Password()
	base := (&url.URL{Scheme: srv.URL.Scheme, Host: srv.URL.Host}).String()

	c := newClient(t, base, srv.URL.User.Username(), pw)
	if err := c.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	items, err := c.Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if count(items, "cluster") != 0 {
		t.Errorf("у одиночного ESXi не должно быть кластеров: %d", count(items, "cluster"))
	}
	if count(items, "hypervisor_host") != 1 {
		t.Fatalf("хостов = %d, want 1", count(items, "hypervisor_host"))
	}

	byExt := map[string]canon.Record{}
	for _, it := range items {
		byExt[it.Record.ExternalID] = it.Record
	}
	mgr := byExt["manager:vc-test"]
	if strings.Contains(mgr.Attrs["product"].(string), "vCenter") {
		t.Errorf("product для отдельного ESXi = %q, не должен содержать vCenter (нет centralized-менеджера)", mgr.Attrs["product"])
	}
	if !strings.Contains(strings.ToLower(mgr.Attrs["product"].(string)), "esxi") {
		t.Errorf("product для отдельного ESXi = %q, want содержит esxi", mgr.Attrs["product"])
	}
	if len(mgr.Relations) != 1 || mgr.Relations[0].Type != "contains" {
		t.Errorf("менеджер должен содержать одиночный хост: %+v", mgr.Relations)
	}
}

func TestExtract_BadCredentials(t *testing.T) {
	base, user, _ := startSim(t)
	if err := newClient(t, base, user, "wrong").Test(context.Background()); err == nil {
		t.Fatal("Test с неверным паролем должен вернуть ошибку")
	}
	if items, err := newClient(t, base, user, "wrong").Extract(context.Background()); err == nil || items != nil {
		t.Fatalf("Extract с неверным паролем: items=%d err=%v", len(items), err)
	}
}

// Дубликаты UUID (клоны) не должны становиться идентификаторами — иначе две ВМ схлопнутся в одну КЕ.
func TestBuildRecords_DuplicateUUIDsAreNotIdentifiers(t *testing.T) {
	snap := snapshot{
		Hosts: []hostInfo{{ID: "host-1", Name: "esx1"}},
		VMs: []vmInfo{
			{ID: "vm-1", Name: "web-1", UUID: "aaaa", InstanceUUID: "i-1", HostID: "host-1", CPU: 2, MemoryMB: 4096},
			{ID: "vm-2", Name: "web-1-clone", UUID: "aaaa", InstanceUUID: "i-2", HostID: "host-1", CPU: 2, MemoryMB: 4096},
			{ID: "vm-3", Name: "db", UUID: "bbbb", InstanceUUID: "i-3", HostID: "host-1", CPU: 4, MemoryMB: 512},
		},
	}
	byName := map[string]canon.Record{}
	for _, r := range buildRecords(snap, "vc", "mgr") {
		byName[r.SourceName] = r
	}
	for _, n := range []string{"web-1", "web-1-clone"} {
		r := byName[n]
		if _, has := r.Identifiers["bios_uuid"]; has {
			t.Errorf("%s: дубликат BIOS UUID не должен быть идентификатором", n)
		}
		if r.Identifiers["instance_uuid"] == "" {
			t.Errorf("%s: уникальный instance UUID должен остаться идентификатором", n)
		}
		if r.Vendor["duplicate_uuid"] != true {
			t.Errorf("%s: дубликат UUID должен быть отмечен в вендорских сведениях", n)
		}
	}
	db := byName["db"]
	if db.Identifiers["bios_uuid"] != "bbbb" || db.Vendor != nil {
		t.Errorf("db = %+v", db)
	}
	if db.Attrs["ram_gb"] != 1 { // 512 МиБ -> не 0
		t.Errorf("ram_gb для 512 МиБ = %v, want 1 (ненулевая память не сводится к 0)", db.Attrs["ram_gb"])
	}
}

func TestNew_RequiresSettings(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без base_url должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"base_url": "https://h", "username": "u", "password_env": "VCENTER_UNSET_XYZ"}}); err == nil {
		t.Error("New без переменной окружения с паролем должен вернуть ошибку")
	}
}

// E5.5: WWPN FC-адаптеров хоста становятся интерфейсами (мост к презентации томов), NAA extent'ов
// VMFS — атрибутом датастора (мост датастор -> том), SMBIOS UUID — атрибутом хоста.
func TestBuildRecords_LinkingIdentifiers(t *testing.T) {
	snap := snapshot{
		Hosts: []hostInfo{{ID: "host-1", Name: "esx1", UUID: "aabbccdd-1111-2222-3333-444455556666", Serial: "SN1",
			HBAs: []hbaInfo{{Device: "vmhba2", WWPN: "21000e1e1a2b3c4d"}, {Device: "vmhba3", WWPN: "garbage"}}}},
		Datastores: []dsInfo{{ID: "ds-1", Name: "vmfs1", Type: "VMFS", NAA: []string{"600a09800000abcd"}}, {ID: "ds-2", Name: "nfs1", Type: "NFS"}},
	}
	byName := map[string]canon.Record{}
	for _, r := range buildRecords(snap, "vc", "mgr") {
		byName[r.SourceName] = r
	}
	h := byName["esx1"]
	if h.Attrs["bios_uuid"] != "aabbccdd-1111-2222-3333-444455556666" {
		t.Errorf("bios_uuid хоста = %v", h.Attrs["bios_uuid"])
	}
	if len(h.Interfaces) != 1 || h.Interfaces[0].WWPN != "21:00:0e:1e:1a:2b:3c:4d" || h.Interfaces[0].IfType != "fc" || h.Interfaces[0].Name != "vmhba2" {
		t.Errorf("интерфейсы хоста = %+v, want один FC-порт (некорректный WWPN отброшен)", h.Interfaces)
	}
	if naa, _ := byName["vmfs1"].Attrs["naa"].([]string); len(naa) != 1 || naa[0] != "600a09800000abcd" {
		t.Errorf("naa датастора VMFS = %v", byName["vmfs1"].Attrs["naa"])
	}
	if naa, ok := byName["nfs1"].Attrs["naa"].([]string); !ok || len(naa) != 0 {
		t.Errorf("naa датастора NFS = %#v, want пустой список (достоверно: нет устройств)", byName["nfs1"].Attrs["naa"])
	}
}

// E8.3: vSAN представлен как обычный кластер (признак vsan_enabled) и обычный датастор
// (free_gb — свободная ёмкость) — случай VsanEnabled=true не воспроизвести на vcsim (govmomi не
// умеет включать vSAN в симуляторе), поэтому здесь проверяется только сведение snapshot -> canon
// без обращения к govmomi/vsan.
func TestBuildRecords_VsanAndFreeSpace(t *testing.T) {
	snap := snapshot{
		Clusters:   []clusterInfo{{ID: "cl-1", Name: "vsan-cluster", VsanEnabled: true}, {ID: "cl-2", Name: "plain-cluster"}},
		Datastores: []dsInfo{{ID: "ds-1", Name: "vsanDatastore", Type: "vsan", CapacityBytes: 10 << 30, FreeBytes: 4 << 30}},
	}
	byName := map[string]canon.Record{}
	for _, r := range buildRecords(snap, "vc", "mgr") {
		byName[r.SourceName] = r
	}
	if byName["vsan-cluster"].Attrs["vsan_enabled"] != true {
		t.Errorf("vsan-cluster: vsan_enabled = %v, want true", byName["vsan-cluster"].Attrs["vsan_enabled"])
	}
	if byName["plain-cluster"].Attrs["vsan_enabled"] != false {
		t.Errorf("plain-cluster: vsan_enabled = %v, want false", byName["plain-cluster"].Attrs["vsan_enabled"])
	}
	ds := byName["vsanDatastore"]
	if ds.Attrs["ds_type"] != "vsan" || ds.Attrs["free_gb"] != 4.0 || ds.Attrs["capacity_gb"] != 10.0 {
		t.Errorf("vsanDatastore = %+v", ds.Attrs)
	}
}
