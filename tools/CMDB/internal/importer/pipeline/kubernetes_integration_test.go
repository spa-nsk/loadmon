// E8.3: коннектор Kubernetes через весь конвейер на реальном PostgreSQL.
package pipeline_test

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/connectors/kubernetes/kubernetesmock"
	"cmdb/internal/testutil"
)

func newKubernetesConnector(t *testing.T, db *sql.DB, m *kubernetesmock.Server, tag string) string {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("CMDB_K8S_TEST_TOKEN", m.Token)

	code := "k8s-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": srv.URL, "token_env": "CMDB_K8S_TEST_TOKEN", "cluster_name": "cl-" + tag})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('kubernetes', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'kubernetes', 'Тестовый Kubernetes', 'kubernetes', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func TestRunConnector_KubernetesClusterNodesAndNamespaces(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := kubernetesmock.Default(tag)
	code := newKubernetesConnector(t, db, m, tag)

	sum := runConn(t, code)
	// менеджер + кластер + 2 узла + 2 namespace
	if sum.Created != 6 || sum.Errors != 0 || sum.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=6", sum)
	}

	cluster := ciID(t, db, "cl-"+tag)
	node := ciID(t, db, "cp-"+tag)
	if got := openRels(t, db, node, "member_of"); len(got) != 1 || got[0] != cluster {
		t.Errorf("узел должен входить в кластер: %v, want [%d]", got, cluster)
	}

	var role string
	var ready bool
	if err := db.QueryRow(`SELECT role, ready FROM cmdb.ci_k8s_node WHERE ci_id = $1`, node).Scan(&role, &ready); err != nil {
		t.Fatalf("ci_k8s_node: %v", err)
	}
	if role != "control-plane" || !ready {
		t.Errorf("control-plane узел: role=%q ready=%v, want control-plane/true", role, ready)
	}

	ns := ciID(t, db, "default-"+tag)
	var status string
	if err := db.QueryRow(`SELECT status FROM cmdb.ci_k8s_namespace WHERE ci_id = $1`, ns).Scan(&status); err != nil {
		t.Fatalf("ci_k8s_namespace: %v", err)
	}
	if status != "Active" {
		t.Errorf("namespace default: status = %q, want Active", status)
	}
	if got := openRels(t, db, ns, "member_of"); len(got) != 1 || got[0] != cluster {
		t.Errorf("namespace должен входить в кластер: %v, want [%d]", got, cluster)
	}

	var source string
	if err := db.QueryRow(`SELECT source_kind FROM audit.change_set cs JOIN audit.change_log cl ON cl.change_set_id = cs.id
		WHERE cl.ci_id = $1 AND cl.op = 'I' LIMIT 1`, node).Scan(&source); err != nil {
		t.Fatalf("source_kind: %v", err)
	}
	if source != "import" {
		t.Errorf("source_kind создания узла = %q, want import", source)
	}

	// повторный прогон без изменений на стороне источника — идемпотентен
	if second := runConn(t, code); second.Created != 0 || second.Updated != 0 || second.Unchanged != 6 {
		t.Errorf("повторный прогон = %+v, want 6 Unchanged", second)
	}
}
