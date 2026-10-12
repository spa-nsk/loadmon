package kubernetes

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/kubernetes/kubernetesmock"
)

func newClient(t *testing.T, m *kubernetesmock.Server, extra map[string]any) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("K8S_TEST_TOKEN", m.Token)
	settings := map[string]any{"base_url": srv.URL, "token_env": "K8S_TEST_TOKEN", "cluster_name": "test-cluster"}
	for k, v := range extra {
		settings[k] = v
	}
	c, err := New(connector.Config{Code: "k8s-test", Kind: Kind, Settings: settings})
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

func TestExtract_MapsClusterNodesAndNamespaces(t *testing.T) {
	m := kubernetesmock.Default("0123456789")
	items, err := newClient(t, m, nil).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// 1 менеджер + 1 кластер + 2 узла (постранично, PageCap=1) + 2 namespace (постранично)
	if len(items) != 6 {
		t.Fatalf("записей = %d, want 6", len(items))
	}
	recs := byExt(items)

	mgr := recs["manager:k8s-test"]
	if mgr.Class != "virt_manager" || mgr.Attrs["product"] != "Kubernetes" || mgr.Attrs["version"] != "v1.29.2" {
		t.Errorf("менеджер = %+v", mgr.Attrs)
	}
	if len(mgr.Relations) != 1 || mgr.Relations[0].TargetExternalID != "cluster:k8s-test" {
		t.Errorf("менеджер должен содержать кластер: %+v", mgr.Relations)
	}

	cluster := recs["cluster:k8s-test"]
	if cluster.Class != "cluster" || cluster.SourceName != "test-cluster" || cluster.Attrs["cluster_kind"] != "k8s" {
		t.Errorf("кластер = %+v", cluster)
	}

	cp := recs["node:node-cp-0123456789"]
	if cp.Class != "k8s_node" || cp.SourceName != "cp-0123456789" || cp.Attrs["role"] != "control-plane" || cp.Attrs["ready"] != true {
		t.Errorf("control-plane узел = %+v", cp)
	}
	if cp.Identifiers["bios_uuid"] != "aaaaaaaa-1111-2222-3333-000123456789" {
		t.Errorf("bios_uuid control-plane узла = %q", cp.Identifiers["bios_uuid"])
	}
	if len(cp.Relations) != 1 || cp.Relations[0].Type != "member_of" || cp.Relations[0].TargetExternalID != "cluster:k8s-test" {
		t.Errorf("узел должен входить в кластер: %+v", cp.Relations)
	}
	if cp.Vendor["internal_ip"] != "10.1.0.1" {
		t.Errorf("internal_ip control-plane узла = %v", cp.Vendor["internal_ip"])
	}

	wk := recs["node:node-wk-0123456789"]
	if wk.Attrs["role"] != "worker" {
		t.Errorf("рабочий узел: role = %v, want worker", wk.Attrs["role"])
	}

	ns := recs["ns:ns-default-0123456789"]
	if ns.Class != "k8s_namespace" || ns.SourceName != "default-0123456789" || ns.Attrs["status"] != "Active" {
		t.Errorf("namespace default = %+v", ns)
	}
	if len(ns.Relations) != 1 || ns.Relations[0].TargetExternalID != "cluster:k8s-test" {
		t.Errorf("namespace должен входить в кластер: %+v", ns.Relations)
	}
}

func TestTest_Auth(t *testing.T) {
	m := kubernetesmock.Default("0123456789")
	if err := newClient(t, m, nil).Test(context.Background()); err != nil {
		t.Fatalf("Test с верным токеном: %v", err)
	}
	m2 := kubernetesmock.Default("0123456789")
	srv := httptest.NewServer(m2)
	t.Cleanup(srv.Close)
	t.Setenv("K8S_TEST_TOKEN_WRONG", "wrong-token")
	c, err := New(connector.Config{Code: "k8s-test", Settings: map[string]any{"base_url": srv.URL, "token_env": "K8S_TEST_TOKEN_WRONG"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным токеном: %v, want HTTP 401", err)
	}
}

func TestExtract_FailureIsWholeSnapshotError(t *testing.T) {
	m := kubernetesmock.Default("0123456789")
	m.Fail = true
	items, err := newClient(t, m, nil).Extract(context.Background())
	if err == nil || items != nil {
		t.Fatalf("Extract при сбое: items=%d err=%v, want ошибка без частичного снимка", len(items), err)
	}
}

func TestNew_RequiresSettings(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{}}); err == nil {
		t.Error("New без base_url должен вернуть ошибку")
	}
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"base_url": "https://h", "token_env": "K8S_UNSET_XYZ"}}); err == nil {
		t.Error("New без переменной окружения с токеном должен вернуть ошибку")
	}
}

// E8.3: сведение snapshot -> canon без обращения к API — конвенция kubeadm для роли узла и
// Ready-условие среди прочих условий (не единичный флаг).
func TestBuildRecords_NodeRoleAndReadyConditions(t *testing.T) {
	nodes := []node{
		{Metadata: objectMeta{Name: "n1", UID: "u1", Labels: map[string]string{"node-role.kubernetes.io/master": ""}}},
		{Metadata: objectMeta{Name: "n2", UID: "u2"}},
	}
	nodes[0].Status.Conditions = []nodeCondition{{Type: "MemoryPressure", Status: "False"}, {Type: "Ready", Status: "True"}}
	nodes[1].Status.Conditions = []nodeCondition{{Type: "Ready", Status: "False"}}

	recs := buildRecords("v1.30.0", "c1", "cl1", nodes, nil)
	byName := map[string]canon.Record{}
	for _, r := range recs {
		byName[r.SourceName] = r
	}
	if byName["n1"].Attrs["role"] != "control-plane" || byName["n1"].Attrs["ready"] != true {
		t.Errorf("n1 (устаревший лейбл master) = %+v", byName["n1"].Attrs)
	}
	if byName["n2"].Attrs["role"] != "worker" || byName["n2"].Attrs["ready"] != false {
		t.Errorf("n2 = %+v", byName["n2"].Attrs)
	}
}
