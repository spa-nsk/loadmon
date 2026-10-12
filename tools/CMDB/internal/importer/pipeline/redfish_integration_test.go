// E5.2: коннектор Redfish через весь конвейер на PostgreSQL: несколько mock-BMC (redfishmock),
// компоненты и интерфейсы, недоступный BMC, приоритет источников, переезд BMC на другой адрес.
package pipeline_test

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/connectors/oneview/oneviewmock"
	"cmdb/internal/importer/connectors/redfish/redfishmock"
	"cmdb/internal/testutil"
)

func newRedfishConnector(t *testing.T, db *sql.DB, targets ...string) string {
	t.Helper()
	t.Setenv("CMDB_REDFISH_TEST_PW", "secret")
	code := "rf-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"targets": targets, "username": "root", "password_env": "CMDB_REDFISH_TEST_PW"})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind, trust_level) VALUES ('redfish', 'api', 70) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'redfish', 'Тестовые BMC', 'redfish', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func serve(t *testing.T, ms ...*redfishmock.Server) []string {
	t.Helper()
	var urls []string
	for _, m := range ms {
		srv := httptest.NewServer(m)
		t.Cleanup(srv.Close)
		urls = append(urls, srv.URL)
	}
	return urls
}

func compCount(t *testing.T, db *sql.DB, ci int64, status string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM cmdb.component WHERE ci_id = $1 AND status = $2`, ci, status).Scan(&n); err != nil {
		t.Fatalf("компоненты: %v", err)
	}
	return n
}

func TestRunConnector_RedfishSnapshotComponentsAndIdempotency(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m1, m2 := redfishmock.New(tag, 1), redfishmock.New(tag, 2)
	code := newRedfishConnector(t, db, serve(t, m1, m2)...)

	first := runConn(t, code)
	if first.Created != 2 || first.Errors != 0 || first.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=2", first)
	}
	srv1 := ciID(t, db, m1.HostName)
	// 2 CPU + 2 DIMM + RAID + 2 диска + 2 БП + 2 вентилятора + BMC
	if n := compCount(t, db, srv1, "installed"); n != 12 {
		t.Fatalf("компонентов = %d, want 12", n)
	}
	var ifaces int
	db.QueryRow(`SELECT count(*) FROM cmdb.interface WHERE ci_id = $1`, srv1).Scan(&ifaces)
	if ifaces != 3 {
		t.Errorf("интерфейсов = %d, want 3", ifaces)
	}
	var cores, ram int
	var bmcIP string
	db.QueryRow(`SELECT core_count, ram_gb, host(bmc_ip) FROM cmdb.ci_server WHERE ci_id = $1`, srv1).Scan(&cores, &ram, &bmcIP)
	if cores != 40 || ram != 64 || bmcIP != "127.0.0.1" {
		t.Errorf("ci_server cores=%d ram=%d bmc_ip=%q, want 40/64/127.0.0.1", cores, ram, bmcIP)
	}
	var diskParent sql.NullInt64
	db.QueryRow(`SELECT parent_component_id FROM cmdb.component WHERE ci_id = $1 AND component_type = 'nvme'`, srv1).Scan(&diskParent)
	if !diskParent.Valid {
		t.Error("диск должен быть вложен в RAID-контроллер (parent_component_id)")
	}

	second := runConn(t, code)
	if second.Created != 0 || second.Updated != 0 || second.Unchanged != 2 {
		t.Fatalf("повторный прогон = %+v, want 2 Unchanged (компоненты и интерфейсы без ложных изменений)", second)
	}
	var maxVer int
	db.QueryRow(`SELECT max(row_version) FROM cmdb.component WHERE ci_id = $1`, srv1).Scan(&maxVer)
	if maxVer != 1 {
		t.Errorf("row_version компонентов после повтора = %d, want 1 (нет лишних версий в hist)", maxVer)
	}
}

func TestRunConnector_RedfishComponentDiff(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := redfishmock.New(tag, 1)
	code := newRedfishConnector(t, db, serve(t, m)...)
	runConn(t, code)
	srv := ciID(t, db, m.HostName)

	// DIMM заменили (другой серийный номер), прошивка BMC обновилась
	m.Do(func(s *redfishmock.Server) {
		s.DIMMs[0].Serial = "DIMM-NEW-" + tag
		s.BMCFirmware = "iLO 5 v3.00"
	})
	sum := runConn(t, code)
	if sum.Updated != 1 || sum.Created != 0 {
		t.Fatalf("прогон после замены = %+v, want Updated=1", sum)
	}
	if n := compCount(t, db, srv, "installed"); n != 12 {
		t.Errorf("установленных компонентов = %d, want 12 (старый DIMM снят, новый добавлен)", n)
	}
	if n := compCount(t, db, srv, "removed"); n != 1 {
		t.Errorf("снятых компонентов = %d, want 1 (не удаляется — остаётся в истории)", n)
	}
	var fw string
	var ver int
	db.QueryRow(`SELECT firmware, row_version FROM cmdb.component WHERE ci_id = $1 AND component_type = 'bmc'`, srv).Scan(&fw, &ver)
	if fw != "iLO 5 v3.00" || ver != 2 {
		t.Errorf("BMC firmware=%q row_version=%d, want обновление на месте (v3.00, версия 2)", fw, ver)
	}
}

func TestRunConnector_RedfishUnreachableBMCKeepsServer(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m1, m2 := redfishmock.New(tag, 1), redfishmock.New(tag, 2)
	code := newRedfishConnector(t, db, serve(t, m1, m2)...)
	runConn(t, code)

	m2.Do(func(s *redfishmock.Server) { s.Fail = true })
	for i := 0; i < 4; i++ { // больше порога notDiscoveredAfterMisses
		sum := runConn(t, code)
		if sum.Errors != 1 || sum.NotDiscovered != 0 || sum.Unchanged != 1 {
			t.Fatalf("прогон %d при недоступном BMC = %+v, want Errors=1, Unchanged=1, NotDiscovered=0", i+1, sum)
		}
	}
	var status sql.NullString
	var maxMissed int
	db.QueryRow(`SELECT lifecycle_status FROM cmdb.ci WHERE name = $1`, m2.HostName).Scan(&status)
	db.QueryRow(`SELECT COALESCE(max(missed_runs), 0) FROM imp.match m JOIN imp.connector c ON c.id = m.connector_id WHERE c.connector_code = $1`, code).Scan(&maxMissed)
	if status.String == "not_discovered" || maxMissed != 0 {
		t.Fatalf("недоступный BMC: lifecycle_status=%q missed_runs=%d, want сервер не «пропал» (harness/04, урок 3)", status.String, maxMissed)
	}
}

// Два источника одного сервера (OneView доверие 80, Redfish 70): Redfish не перебивает значения
// OneView и не «дребезжит» между прогонами, но добавляет то, чего у OneView нет (компоненты).
func TestRunConnector_RedfishDoesNotOverrideHigherTrustSource(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	ov := oneviewmock.Default(tag)
	ovCode := newOneViewConnector(t, db, ov)
	runConn(t, ovCode)

	rf := redfishmock.New(tag, 3)
	rf.Do(func(s *redfishmock.Server) { s.Serial = "SN-SRV3-" + tag }) // тот же серийный, что у стоечного сервера OneView
	rfCode := newRedfishConnector(t, db, serve(t, rf)...)

	server := ciID(t, db, "rack-dl380-"+tag)
	sum := runConn(t, rfCode)
	if sum.Created != 0 || sum.Errors != 0 || sum.Conflicts != 0 {
		t.Fatalf("прогон Redfish = %+v, want сопоставление с сервером OneView без создания", sum)
	}

	var ram, cpu int
	db.QueryRow(`SELECT ram_gb, cpu_count FROM cmdb.ci_server WHERE ci_id = $1`, server).Scan(&ram, &cpu)
	if ram != 256 { // OneView: 262144 МиБ; Redfish отдал бы 64
		t.Fatalf("ram_gb = %d, want 256 — значение OneView (доверие 80) не должно быть перебито Redfish (70)", ram)
	}
	if n := compCount(t, db, server, "installed"); n != 12 {
		t.Errorf("компонентов от Redfish = %d, want 12 (у OneView их нет — дополняются)", n)
	}

	// повторные прогоны обоих источников не дребезжат
	if again := runConn(t, rfCode); again.Updated != 0 || again.Unchanged != 1 {
		t.Fatalf("повторный прогон Redfish = %+v, want Unchanged=1", again)
	}
	if again := runConn(t, ovCode); again.Updated != 0 {
		t.Fatalf("повторный прогон OneView = %+v, want без изменений", again)
	}

	// OneView (выше по доверию) свои значения обновляет, Redfish их не откатывает
	ov.Do(func(s *oneviewmock.Server) { s.Servers[2].MemoryMB = 524288 })
	runConn(t, ovCode)
	runConn(t, rfCode)
	db.QueryRow(`SELECT ram_gb FROM cmdb.ci_server WHERE ci_id = $1`, server).Scan(&ram)
	if ram != 512 {
		t.Fatalf("ram_gb = %d, want 512 (обновление OneView, Redfish не откатывает)", ram)
	}
}

// BMC сменил адрес (DHCP): новый внешний ID, но тот же MAC BMC — это тот же сервер, а не дубликат
// и не конфликт (E4.6: сервер → ... → bmc_mac).
func TestRunConnector_RedfishBMCAddressChangeMatchesByBMCMAC(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := redfishmock.New(tag, 1)
	urls := serve(t, m)
	code := newRedfishConnector(t, db, urls...)
	runConn(t, code)
	srv := ciID(t, db, m.HostName)

	// «тот же сервер» за новым адресом: другие серийный номер и UUID (замена материнской платы), тот же MAC BMC
	moved := redfishmock.New(tag, 1)
	moved.Do(func(s *redfishmock.Server) {
		s.Serial, s.UUID = "SN-NEWBOARD-"+tag, "4"+tag[1:8]+"-0000-4000-8000-"+tag+"01"
		s.BMCMAC = m.BMCMAC
	})
	newURL := serve(t, moved)[0]
	if _, err := db.Exec(`UPDATE imp.connector SET config = jsonb_set(config, '{targets}', to_jsonb($2::text[])) WHERE connector_code = $1`,
		code, "{"+newURL+"}"); err != nil {
		t.Fatalf("смена адреса: %v", err)
	}

	sum := runConn(t, code)
	if sum.Created != 0 || sum.Conflicts != 0 || sum.Errors != 0 {
		t.Fatalf("прогон после смены адреса = %+v, want сопоставление по bmc_mac без создания и конфликтов", sum)
	}
	var method string
	db.QueryRow(`SELECT match_method FROM imp.norm_record WHERE run_id = $1`, sum.RunID).Scan(&method)
	if method != "bmc_mac" {
		t.Errorf("метод сопоставления = %q, want bmc_mac", method)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE ci_class = 'server' AND id = $1`, srv).Scan(&n)
	if n != 1 {
		t.Error("исходная КЕ должна остаться единственной")
	}
}
