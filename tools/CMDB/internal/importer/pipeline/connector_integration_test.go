// E5.2a/E4.8: API-коннектор через весь конвейер на реальном PostgreSQL. Источник — mock-сервер
// oVirt (zvirtmock), состояние которого тест меняет между прогонами.
package pipeline_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/connectors/oneview/oneviewmock"
	"cmdb/internal/importer/connectors/zabbix/zabbixmock"
	"cmdb/internal/importer/connectors/zvirt/zvirtmock"
	"cmdb/internal/importer/pipeline"
	"cmdb/internal/testutil"
)

// newZvirtConnector регистрирует коннектор zvirt, направленный на mock-сервер, и возвращает его код.
func newZvirtConnector(t *testing.T, db *sql.DB, m *zvirtmock.Server) string {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("CMDB_ZVIRT_TEST_PW", m.Password)

	code := "zv-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "CMDB_ZVIRT_TEST_PW", "manager_name": "mgr-" + code})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('zvirt', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'zvirt', 'Тестовый zVirt', 'zvirt', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func runConn(t *testing.T, code string) pipeline.Summary {
	t.Helper()
	sum, err := pipeline.RunConnector(context.Background(), code, pipeline.Options{UserID: 1})
	if err != nil {
		t.Fatalf("RunConnector: %v", err)
	}
	return sum
}

func ciID(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM cmdb.ci WHERE name = $1 AND lifecycle_stage NOT IN ('Retired', 'Disposed')`, name).Scan(&id); err != nil {
		t.Fatalf("КЕ %q: %v", name, err)
	}
	return id
}

func openRels(t *testing.T, db *sql.DB, src int64, relType string) []int64 {
	t.Helper()
	rows, err := db.Query(`SELECT dst_ci_id FROM cmdb.ci_rel WHERE src_ci_id = $1 AND rel_type = $2 AND upper_inf(valid_period) ORDER BY dst_ci_id`, src, relType)
	if err != nil {
		t.Fatalf("связи: %v", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		out = append(out, id)
	}
	return out
}

func TestRunConnector_SnapshotAndIdempotency(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := zvirtmock.Default(tag)
	code := newZvirtConnector(t, db, m)

	first := runConn(t, code)
	// менеджер + кластер + 2 хоста + 3 ВМ + 1 датастор
	if first.Created != 8 || first.Errors != 0 || first.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=8", first)
	}
	// contains(1) + host member_of(2) + vm member_of(3) + vm runs_on(2)
	if first.RelCreated != 8 || first.ChangeSetID == 0 {
		t.Fatalf("первый прогон = %+v, want RelCreated=8 и один change_set", first)
	}

	vm := ciID(t, db, "app-01-"+tag)
	host := ciID(t, db, "zv-host-01-"+tag)
	if got := openRels(t, db, vm, "runs_on"); len(got) != 1 || got[0] != host {
		t.Fatalf("runs_on ВМ app-01 = %v, want [%d]", got, host)
	}
	var ram, vcpu int
	if err := db.QueryRow(`SELECT ram_gb, vcpu FROM cmdb.ci_vm WHERE ci_id = $1`, vm).Scan(&ram, &vcpu); err != nil {
		t.Fatalf("ci_vm: %v", err)
	}
	if ram != 8 || vcpu != 4 {
		t.Errorf("ci_vm ram_gb=%d vcpu=%d, want 8/4", ram, vcpu)
	}

	second := runConn(t, code)
	if second.Created != 0 || second.Updated != 0 || second.Unchanged != 8 || second.RelCreated != 0 || second.RelClosed != 0 {
		t.Fatalf("повторный прогон = %+v, want 8 Unchanged без связей (идемпотентность)", second)
	}
}

func TestRunConnector_VMMigrationMovesRelation(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := zvirtmock.Default(tag)
	code := newZvirtConnector(t, db, m)
	runConn(t, code)

	m.Do(func(s *zvirtmock.Server) { s.VMs[0].HostID = s.Hosts[1].ID })
	sum := runConn(t, code)
	if sum.RelClosed != 1 || sum.RelCreated != 1 || sum.Created != 0 {
		t.Fatalf("прогон после переезда = %+v, want закрыта 1 и создана 1 связь", sum)
	}

	vm := ciID(t, db, "app-01-"+tag)
	host2 := ciID(t, db, "zv-host-02-"+tag)
	if got := openRels(t, db, vm, "runs_on"); len(got) != 1 || got[0] != host2 {
		t.Fatalf("после переезда runs_on = %v, want ровно [%d]", got, host2)
	}

	// ВМ выключили — связи runs_on больше нет
	m.Do(func(s *zvirtmock.Server) { s.VMs[0].HostID = "" })
	if sum := runConn(t, code); sum.RelClosed != 1 || sum.RelCreated != 0 {
		t.Fatalf("прогон после выключения ВМ = %+v, want закрыта 1 связь", sum)
	}
	if got := openRels(t, db, vm, "runs_on"); len(got) != 0 {
		t.Fatalf("runs_on выключенной ВМ = %v, want нет", got)
	}
}

func TestRunConnector_SourceFailureChangesNothing(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := zvirtmock.Default(tag)
	code := newZvirtConnector(t, db, m)
	runConn(t, code)

	m.Do(func(s *zvirtmock.Server) { s.Fail = true })
	for i := 0; i < 4; i++ { // больше порога notDiscoveredAfterMisses
		sum, err := pipeline.RunConnector(context.Background(), code, pipeline.Options{UserID: 1})
		if err == nil {
			t.Fatal("RunConnector при сбое источника должен вернуть ошибку")
		}
		var status string
		if e := db.QueryRow(`SELECT status FROM imp.run WHERE id = $1`, sum.RunID).Scan(&status); e != nil || status != "failed" {
			t.Fatalf("статус прогона = %q (err=%v), want failed", status, e)
		}
	}

	var maxMissed int
	var notDiscovered int
	db.QueryRow(`SELECT COALESCE(max(missed_runs), 0) FROM imp.match m JOIN imp.connector c ON c.id = m.connector_id WHERE c.connector_code = $1`, code).Scan(&maxMissed)
	db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE lifecycle_status = 'not_discovered' AND name LIKE '%' || $1`, tag).Scan(&notDiscovered)
	if maxMissed != 0 || notDiscovered != 0 {
		t.Fatalf("после сбоев источника missed_runs=%d not_discovered=%d, want 0/0 (сбой не «списывает» КЕ)", maxMissed, notDiscovered)
	}
}

func TestRunConnector_RemovedVMBecomesNotDiscovered(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := zvirtmock.Default(tag)
	code := newZvirtConnector(t, db, m)
	runConn(t, code)

	m.Do(func(s *zvirtmock.Server) { s.VMs = s.VMs[:2] }) // test-off удалена в источнике
	for i := 0; i < 3; i++ {
		runConn(t, code)
	}
	var status sql.NullString
	if err := db.QueryRow(`SELECT lifecycle_status FROM cmdb.ci WHERE name = $1`, "test-off-"+tag).Scan(&status); err != nil {
		t.Fatalf("КЕ: %v", err)
	}
	if status.String != "not_discovered" {
		t.Fatalf("lifecycle_status удалённой в источнике ВМ = %q, want not_discovered (КЕ не удаляется)", status.String)
	}
}

func TestRunConnector_UnknownConnectorAndKind(t *testing.T) {
	testutil.DB(t)
	if _, err := pipeline.RunConnector(context.Background(), "no-such-"+testutil.Suffix(), pipeline.Options{}); err == nil {
		t.Error("RunConnector для несуществующего коннектора должен вернуть ошибку")
	}
}

func newOneViewConnector(t *testing.T, db *sql.DB, m *oneviewmock.Server) string {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("CMDB_ONEVIEW_TEST_PW", m.Password)
	code := "ov-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "CMDB_ONEVIEW_TEST_PW"})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('oneview', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'oneview', 'Тестовый OneView', 'oneview', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func TestRunConnector_OneViewServersAndChassis(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := oneviewmock.Default(tag)
	code := newOneViewConnector(t, db, m)

	first := runConn(t, code)
	// корзина + 3 сервера; корзина содержит 2 блейда
	if first.Created != 4 || first.Errors != 0 || first.RelCreated != 2 {
		t.Fatalf("первый прогон = %+v, want Created=4, RelCreated=2", first)
	}

	blade := ciID(t, db, "esx-blade-01-"+tag)
	var ram, cores int
	var bay, formFactor string
	if err := db.QueryRow(`SELECT ram_gb, core_count, bay, form_factor FROM cmdb.ci_server WHERE ci_id = $1`, blade).Scan(&ram, &cores, &bay, &formFactor); err != nil {
		t.Fatalf("ci_server: %v", err)
	}
	if ram != 128 || cores != 24 || bay != "1" || formFactor != "blade" {
		t.Errorf("ci_server ram=%d cores=%d bay=%q ff=%q, want 128/24/1/blade", ram, cores, bay, formFactor)
	}
	var model string
	if err := db.QueryRow(`SELECT attrs -> 'oneview' ->> 'model' FROM cmdb.ci WHERE id = $1`, blade).Scan(&model); err != nil || model != "ProLiant BL460c Gen9" {
		t.Errorf("вендорские сведения oneview.model = %q (err=%v)", model, err)
	}
	if got := openRels(t, db, ciID(t, db, "Enc-"+tag), "contains"); len(got) != 2 {
		t.Errorf("корзина содержит %v, want 2 блейда", got)
	}

	second := runConn(t, code)
	if second.Created != 0 || second.Updated != 0 || second.Unchanged != 4 || second.RelCreated != 0 || second.RelClosed != 0 {
		t.Fatalf("повторный прогон = %+v, want 4 Unchanged (идемпотентность, в т.ч. вендорские сведения)", second)
	}
}

// E5.5: хост гипервизора из zVirt и сервер из OneView с одним серийным номером связываются
// выведенной связью runs_on; связь следует за данными и закрывается, когда совпадение пропало.
func TestRunConnector_BridgeHypervisorToServerBySerial(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	ov := oneviewmock.Default(tag)
	zv := zvirtmock.Default(tag)
	zv.Do(func(s *zvirtmock.Server) { s.Hosts[0].Serial = "SN-SRV1-" + tag }) // тот же серийный, что у блейда OneView
	ovCode := newOneViewConnector(t, db, ov)
	zvCode := newZvirtConnector(t, db, zv)

	runConn(t, ovCode)
	sum := runConn(t, zvCode)
	if sum.Errors != 0 {
		t.Fatalf("прогон zVirt = %+v", sum)
	}

	host := ciID(t, db, "zv-host-01-"+tag)
	server := ciID(t, db, "esx-blade-01-"+tag)
	got := openRels(t, db, host, "runs_on")
	if len(got) != 1 || got[0] != server {
		t.Fatalf("runs_on хоста = %v, want [%d] (сервер с тем же серийным номером)", got, server)
	}
	if !derivedRel(t, db, host, server) {
		t.Error("связь должна вестись от имени источника derived")
	}

	// повторный прогон — без изменений связей
	if again := runConn(t, zvCode); again.RelCreated != 0 || again.RelClosed != 0 {
		t.Fatalf("повторный прогон = %+v, want связи без изменений", again)
	}

	// серийный номер хоста изменился — совпадения нет, связь закрывается
	zv.Do(func(s *zvirtmock.Server) { s.Hosts[0].Serial = "SN-OTHER-" + tag })
	runConn(t, zvCode)
	if got := openRels(t, db, host, "runs_on"); len(got) != 0 {
		t.Fatalf("после смены серийного номера runs_on = %v, want связь закрыта", got)
	}
}

func derivedRel(t *testing.T, db *sql.DB, src, dst int64) bool {
	t.Helper()
	var n int
	db.QueryRow(`SELECT count(*) FROM cmdb.ci_rel WHERE src_ci_id = $1 AND dst_ci_id = $2 AND rel_type = 'runs_on'
		AND source_system_code = 'derived' AND upper_inf(valid_period)`, src, dst).Scan(&n)
	return n == 1
}

func newZabbixConnector(t *testing.T, db *sql.DB, m *zabbixmock.Server) string {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("CMDB_ZABBIX_TEST_TOKEN", m.Token)
	code := "zbx-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": srv.URL, "token_env": "CMDB_ZABBIX_TEST_TOKEN"})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('zabbix', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'zabbix', 'Тестовый Zabbix', 'zabbix', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

// E5.4: Zabbix только дополняет существующие КЕ — не создаёт, не перезаписывает, не «списывает».
func TestRunConnector_ZabbixSupplementsExistingOnly(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	tag := randHex(10)
	// КЕ «a» (найдётся по короткому имени FQDN-хоста, серийного нет) и «b» (найдётся по имени, серийный уже есть)
	if s := run(t, ctx, buildCSV(t, []map[string]string{
		{"name": "zbx-a-" + tag, "location_code": testutil.TestLocation},
		{"name": "zbx-b-" + tag, "location_code": testutil.TestLocation, "serial_number": "SN-EXISTING-B-" + tag},
	}), false); s.Created != 2 {
		t.Fatalf("фикстура = %+v, want Created=2", s)
	}
	m := zabbixmock.Default(tag)
	code := newZabbixConnector(t, db, m)

	first := runConn(t, code)
	if first.Created != 0 || first.Updated != 2 || first.Skipped != 1 || first.Errors != 0 || first.NotDiscovered != 0 {
		t.Fatalf("первый прогон = %+v, want Updated=2, Skipped=1, Created=0", first)
	}

	a, b := ciID(t, db, "zbx-a-"+tag), ciID(t, db, "zbx-b-"+tag)
	var serialA, serialB, hostID string
	var groups int
	db.QueryRow(`SELECT serial_number FROM cmdb.ci WHERE id = $1`, a).Scan(&serialA)
	db.QueryRow(`SELECT serial_number FROM cmdb.ci WHERE id = $1`, b).Scan(&serialB)
	db.QueryRow(`SELECT attrs -> 'zabbix' ->> 'hostid', jsonb_array_length(attrs -> 'zabbix' -> 'groups') FROM cmdb.ci WHERE id = $1`, b).Scan(&hostID, &groups)
	if serialA != "SN-ZBX-A-"+tag {
		t.Errorf("серийный номер A = %q, want заполнен из inventory (пустое поле дополняется)", serialA)
	}
	if serialB != "SN-EXISTING-B-"+tag {
		t.Errorf("серийный номер B = %q, want прежний (непустое поле не перезаписывается)", serialB)
	}
	if hostID != "10002" || groups != 2 {
		t.Errorf("attrs.zabbix B: hostid=%q groups=%d, want 10002/2", hostID, groups)
	}
	var lastSeen sql.NullTime
	db.QueryRow(`SELECT last_seen_at FROM cmdb.ci WHERE id = $1`, b).Scan(&lastSeen)
	if !lastSeen.Valid {
		t.Error("last_seen_at должен быть заполнен подтверждением источника")
	}
	var created int
	db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE name LIKE 'unknown-%' AND name LIKE '%' || $1`, tag).Scan(&created)
	if created != 0 {
		t.Errorf("хост без КЕ создал %d КЕ, want 0", created)
	}
	var reason string
	db.QueryRow(`SELECT COALESCE(error_message, '') FROM imp.norm_record WHERE run_id = $1 AND status = 'skipped'`, first.RunID).Scan(&reason)
	if reason == "" {
		t.Error("пропущенная запись должна нести причину (не пропускать молча)")
	}

	if second := runConn(t, code); second.Updated != 0 || second.Unchanged != 2 || second.Skipped != 1 {
		t.Fatalf("повторный прогон = %+v, want Unchanged=2, Skipped=1 (идемпотентность)", second)
	}

	// хосты пропали из Zabbix: дополнение не владеет КЕ — они не должны считаться пропавшими
	m.Do(func(s *zabbixmock.Server) { s.Hosts = nil })
	for i := 0; i < 4; i++ {
		runConn(t, code)
	}
	var marked int
	db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE id IN ($1, $2) AND lifecycle_status = 'not_discovered'`, a, b).Scan(&marked)
	if marked != 0 {
		t.Fatalf("not_discovered у %d КЕ после исчезновения хостов из Zabbix, want 0", marked)
	}
}

// Нарушение ограничения БД данными одной записи (здесь — две ВМ с одним именем: уникальность
// активного имени в классе) откатывает только эту запись; остальные применяются, прогон успешен.
func TestRunConnector_BadRecordDoesNotFailWholeRun(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := zvirtmock.Default(tag)
	m.Do(func(s *zvirtmock.Server) {
		dup := s.VMs[0]
		dup.ID = dup.ID[:len(dup.ID)-2] + "99" // другой id, то же имя
		s.VMs = append(s.VMs, dup)
	})
	code := newZvirtConnector(t, db, m)

	sum := runConn(t, code)
	// 8 обычных записей + дубликат: он падает на уникальности имени, остальные создаются
	if sum.Created != 8 || sum.Errors != 1 {
		t.Fatalf("прогон = %+v, want Created=8, Errors=1 (плохая запись не роняет прогон)", sum)
	}
	var status, msg string
	db.QueryRow(`SELECT status FROM imp.run WHERE id = $1`, sum.RunID).Scan(&status)
	db.QueryRow(`SELECT COALESCE(error_message, '') FROM imp.norm_record WHERE run_id = $1 AND status = 'error'`, sum.RunID).Scan(&msg)
	if status != "succeeded" || msg == "" {
		t.Errorf("статус прогона = %q, сообщение ошибки записи = %q", status, msg)
	}
	var conflicts int
	db.QueryRow(`SELECT count(*) FROM imp.conflict WHERE run_id = $1 AND kind = 'apply_error'`, sum.RunID).Scan(&conflicts)
	if conflicts != 1 {
		t.Errorf("конфликтов apply_error = %d, want 1 (ошибка видна в очереди конфликтов)", conflicts)
	}
	// связи остальных записей на месте
	vm := ciID(t, db, "app-01-"+tag)
	if got := openRels(t, db, vm, "runs_on"); len(got) != 1 {
		t.Errorf("runs_on первой ВМ = %v, want 1 связь", got)
	}
}

// Два объекта источника с общим идентификатором (одинаковый серийный номер у разных хостов) —
// конфликт, а не слияние в одну КЕ (E4.6: без автослияния).
func TestRunConnector_SharedSerialIsConflictNotMerge(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := zvirtmock.Default(tag)
	m.Do(func(s *zvirtmock.Server) { s.Hosts[1].Serial = s.Hosts[0].Serial })
	code := newZvirtConnector(t, db, m)

	sum := runConn(t, code)
	if sum.Conflicts != 1 || sum.Created != 7 {
		t.Fatalf("прогон = %+v, want Conflicts=1 (второй хост с тем же серийным номером), Created=7", sum)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE name = $1`, "zv-host-02-"+tag).Scan(&n)
	if n != 0 {
		t.Errorf("хост-дубликат по серийному номеру создал КЕ: %d", n)
	}
	// повторный прогон устойчив: первый хост не «перескакивает» на второго
	if again := runConn(t, code); again.Updated != 0 || again.Conflicts != 1 {
		t.Fatalf("повторный прогон = %+v, want без изменений и тот же 1 конфликт", again)
	}
}
