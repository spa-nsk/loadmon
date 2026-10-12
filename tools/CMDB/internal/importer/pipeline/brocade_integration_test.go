// E8.4: коннектор Brocade FC SAN через весь конвейер на реальном PostgreSQL.
package pipeline_test

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/connectors/brocade/brocademock"
	"cmdb/internal/testutil"
)

func newBrocadeConnector(t *testing.T, db *sql.DB, m *brocademock.Server) string {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("CMDB_BROCADE_TEST_PW", m.Password)

	code := "brocade-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "CMDB_BROCADE_TEST_PW"})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind, trust_level) VALUES ('brocade', 'api', 60) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'brocade', 'Тестовый Brocade', 'brocade', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func TestRunConnector_BrocadeSwitchAndActiveZone(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := brocademock.Default(tag)
	code := newBrocadeConnector(t, db, m)

	sum := runConn(t, code)
	if sum.Created != 2 || sum.Errors != 0 || sum.Conflicts != 0 { // коммутатор + 1 активная зона
		t.Fatalf("первый прогон = %+v, want Created=2", sum)
	}

	sw := ciID(t, db, "sw-"+tag)
	var serial string
	db.QueryRow(`SELECT serial_number FROM cmdb.ci WHERE id = $1`, sw).Scan(&serial)
	if serial != m.WWN {
		t.Errorf("серийный номер коммутатора (WWN) = %q, want %q", serial, m.WWN)
	}
	var domainID, portsTotal int
	if err := db.QueryRow(`SELECT domain_id, ports_total FROM cmdb.ci_fc_switch WHERE ci_id = $1`, sw).Scan(&domainID, &portsTotal); err != nil {
		t.Fatalf("ci_fc_switch: %v", err)
	}
	if domainID != 5 || portsTotal != 2 {
		t.Errorf("ci_fc_switch domain_id=%d ports_total=%d, want 5/2", domainID, portsTotal)
	}

	var ifCount int
	db.QueryRow(`SELECT count(*) FROM cmdb.interface WHERE ci_id = $1`, sw).Scan(&ifCount)
	if ifCount != 1 {
		t.Errorf("интерфейсов коммутатора = %d, want 1 (порт без WWPN отброшен)", ifCount)
	}

	var zoneset, membersJoined string
	zoneCI := ciID(t, db, code+"/zone_"+tag)
	if err := db.QueryRow(`SELECT zoneset FROM cmdb.ci_san_zone WHERE ci_id = $1`, zoneCI).Scan(&zoneset); err != nil {
		t.Fatalf("ci_san_zone: %v", err)
	}
	if err := db.QueryRow(`SELECT array_to_string(members, ',') FROM cmdb.ci_san_zone WHERE ci_id = $1`, zoneCI).Scan(&membersJoined); err != nil {
		t.Fatalf("ci_san_zone.members: %v", err)
	}
	if zoneset != "prod_cfg" {
		t.Errorf("zoneset = %q, want prod_cfg", zoneset)
	}
	members := strings.Split(membersJoined, ",")
	if len(members) != 2 {
		t.Errorf("участников зоны в БД = %d, want 2: %v", len(members), members)
	}

	// повторный прогон без изменений на стороне источника — идемпотентен
	if second := runConn(t, code); second.Created != 0 || second.Updated != 0 || second.Unchanged != 2 {
		t.Errorf("повторный прогон = %+v, want 2 Unchanged", second)
	}
}
