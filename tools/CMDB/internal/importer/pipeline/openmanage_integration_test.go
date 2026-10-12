// E8.2: коннектор Dell OpenManage Enterprise через весь конвейер на реальном PostgreSQL.
package pipeline_test

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/connectors/openmanage/openmanagemock"
	"cmdb/internal/testutil"
)

func newOpenManageConnector(t *testing.T, db *sql.DB, m *openmanagemock.Server) string {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("CMDB_OME_TEST_PW", m.Password)

	code := "ome-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "CMDB_OME_TEST_PW"})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('openmanage', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'openmanage', 'Тестовый OpenManage Enterprise', 'openmanage', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func TestRunConnector_OpenManageServersChassisAndComponents(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := openmanagemock.Default(tag)
	code := newOpenManageConnector(t, db, m)

	sum := runConn(t, code)
	if sum.Created != 3 || sum.Errors != 0 || sum.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=3 (шасси + блейд + стоечный)", sum)
	}

	blade := ciID(t, db, "blade-"+tag)
	if n := compCount(t, db, blade, "installed"); n != 5 { // 2 CPU, 1 DIMM, 1 диск, 1 БП
		t.Errorf("компонентов блейда = %d, want 5", n)
	}
	if got := openRels(t, db, ciID(t, db, "MX7000-"+tag), "contains"); len(got) != 1 || got[0] != blade {
		t.Errorf("шасси должно содержать блейд: %v, want [%d]", got, blade)
	}

	var source string
	if err := db.QueryRow(`SELECT source_kind FROM audit.change_set cs JOIN audit.change_log cl ON cl.change_set_id = cs.id
		WHERE cl.ci_id = $1 AND cl.op = 'I' LIMIT 1`, blade).Scan(&source); err != nil {
		t.Fatalf("source_kind: %v", err)
	}
	if source != "import" {
		t.Errorf("source_kind создания блейда = %q, want import", source)
	}

	rack := ciID(t, db, "rack-"+tag)
	var osFamily string
	db.QueryRow(`SELECT COALESCE(os_family, '') FROM cmdb.ci_server WHERE ci_id = $1`, rack).Scan(&osFamily)
	if osFamily != "Linux" {
		t.Errorf("стоечный сервер: os_family = %q, want Linux", osFamily)
	}

	// повторный прогон без изменений на стороне источника — идемпотентен
	if second := runConn(t, code); second.Created != 0 || second.Updated != 0 || second.Unchanged != 3 {
		t.Errorf("повторный прогон = %+v, want 3 Unchanged", second)
	}
}
