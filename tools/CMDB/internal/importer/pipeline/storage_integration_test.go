// E5.3: коннекторы СХД через весь конвейер на PostgreSQL (mock ONTAP и Unity).
package pipeline_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/connectors/huawei/huaweimock"
	"cmdb/internal/importer/connectors/lenovo/lenovomock"
	"cmdb/internal/importer/connectors/ontap/ontapmock"
	"cmdb/internal/importer/connectors/powerstore/powerstoremock"
	"cmdb/internal/importer/connectors/unity/unitymock"
	"cmdb/internal/importer/connectors/vnx/vnxmock"
	"cmdb/internal/importer/pipeline"
	"cmdb/internal/testutil"
)

func newStorageConnector(t *testing.T, db *sql.DB, kind, ssc, baseURL, pwEnv, user, pw string) string {
	t.Helper()
	t.Setenv(pwEnv, pw)
	code := kind + "-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": baseURL, "username": user, "password_env": pwEnv})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind, trust_level) VALUES ($1, 'api', 60) ON CONFLICT DO NOTHING`, ssc); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, $2, 'Тестовая СХД', $3, $4::jsonb)`, code, kind, ssc, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func TestRunConnector_ONTAPSnapshotAndSweep(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := ontapmock.Default(tag)
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	code := newStorageConnector(t, db, "ontap", "ontap", srv.URL, "CMDB_ONTAP_TEST_PW", m.User, m.Password)

	first := runConn(t, code)
	// массив + 2 агрегата + SVM + 2 тома + LUN + qtree + 2 ресурса
	if first.Created != 10 || first.Errors != 0 || first.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=10", first)
	}
	arr := ciID(t, db, m.Name)
	if n := compCount(t, db, arr, "installed"); n != 6 {
		t.Errorf("компонентов массива = %d, want 6 (2 узла, полка, 3 диска)", n)
	}
	var rawTB, controllers string
	db.QueryRow(`SELECT raw_tb::text, controllers::text FROM cmdb.ci_storage_array WHERE ci_id = $1`, arr).Scan(&rawTB, &controllers)
	if rawTB != "9.50" && rawTB != "9.5" || controllers != "2" {
		t.Errorf("ci_storage_array raw_tb=%s controllers=%s, want 9.5/2", rawTB, controllers)
	}
	vol := ciID(t, db, m.Name+"/svm1_"+tag+":vol_data_"+tag)
	if got := openRels(t, db, vol, "runs_on"); len(got) != 1 {
		t.Errorf("runs_on тома = %v, want агрегат", got)
	}
	var wwn string
	db.QueryRow(`SELECT wwn FROM cmdb.ci_volume WHERE ci_id = $1`, ciID(t, db, m.Name+"/svm1_"+tag+":/vol/vol_lun_"+tag+"/lun1")).Scan(&wwn)
	if len(wwn) < 8 || wwn[:8] != "600a0980" {
		t.Errorf("WWN LUN = %q, want NAA NetApp", wwn)
	}

	if second := runConn(t, code); second.Created != 0 || second.Updated != 0 || second.Unchanged != 10 || second.RelCreated != 0 || second.RelClosed != 0 {
		t.Fatalf("повторный прогон = %+v, want 10 Unchanged (идемпотентность)", second)
	}

	// сбой опроса СХД ничего не меняет и не списывает тома (harness/03, правило 2)
	m.Do(func(s *ontapmock.Server) { s.Fail = true })
	for i := 0; i < 4; i++ {
		if _, err := pipeline.RunConnector(context.Background(), code, pipeline.Options{UserID: 1}); err == nil {
			t.Fatal("сбой источника должен вернуть ошибку")
		}
	}
	var marked int
	db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE lifecycle_status = 'not_discovered' AND name LIKE $1`, m.Name+"/%").Scan(&marked)
	if marked != 0 {
		t.Fatalf("после сбоев опроса not_discovered у %d КЕ, want 0", marked)
	}

	// том удалён на СХД: после 3 успешных опросов — not_discovered
	m.Do(func(s *ontapmock.Server) { s.Fail = false; s.Volumes = s.Volumes[:1]; s.LUNs = nil })
	for i := 0; i < 3; i++ {
		runConn(t, code)
	}
	var status sql.NullString
	db.QueryRow(`SELECT lifecycle_status FROM cmdb.ci WHERE name = $1`, m.Name+"/svm1_"+tag+":vol_lun_"+tag).Scan(&status)
	if status.String != "not_discovered" {
		t.Fatalf("lifecycle_status удалённого тома = %q, want not_discovered", status.String)
	}
}

// Одинаковые имена LUN на разных массивах не сталкиваются (имя КЕ включает имя массива), а
// идентификация тома идёт по WWN.
func TestRunConnector_UnityTwoArraysSameLUNNames(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	a, b := unitymock.Default(tag), unitymock.Default(tag)
	b.Do(func(s *unitymock.Server) {
		s.Data["system"][0]["name"] = "unity-b-" + tag
		s.Data["system"][0]["serialNumber"] = "UNSNB" + tag
		// те же имена LUN, другие WWN
		for i := range s.Data["lun"] {
			s.Data["lun"][i]["wwn"] = "60:06:01:60:bb:" + tag[0:2] + ":" + tag[2:4] + ":" + tag[4:6] + ":00:00:00:0" + string(rune('1'+i))
		}
	})
	sa, sb := httptest.NewServer(a), httptest.NewServer(b)
	t.Cleanup(sa.Close)
	t.Cleanup(sb.Close)
	codeA := newStorageConnector(t, db, "unity", "unity", sa.URL, "CMDB_UNITY_TEST_PW", a.User, a.Password)
	codeB := newStorageConnector(t, db, "unity", "unity", sb.URL, "CMDB_UNITY_TEST_PW", b.User, b.Password)

	for _, code := range []string{codeA, codeB} {
		sum := runConn(t, code)
		if sum.Created != 10 || sum.Errors != 0 || sum.Conflicts != 0 {
			t.Fatalf("прогон %s = %+v, want Created=10 без ошибок и конфликтов", code, sum)
		}
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE ci_class = 'volume' AND name LIKE '%/lun-a-' || $1`, tag).Scan(&n)
	if n != 2 {
		t.Errorf("томов lun-a на двух массивах = %d, want 2 (по одному на массив)", n)
	}
	if again := runConn(t, codeA); again.Updated != 0 || again.Unchanged != 10 {
		t.Fatalf("повторный прогон = %+v, want 10 Unchanged", again)
	}
}

func TestRunConnector_HuaweiSnapshotIdempotencyAndSessions(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := huaweimock.Default(tag)
	m.PageCap = 100 // размер блока коннектора по умолчанию — 100 (постраничность проверена в unit-тесте)
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	code := newStorageConnector(t, db, "huawei", "huawei", srv.URL, "CMDB_HUAWEI_TEST_PW", m.User, m.Password)

	first := runConn(t, code)
	if first.Created != 5 || first.Errors != 0 || first.Conflicts != 0 { // массив + пул + 3 LUN
		t.Fatalf("первый прогон = %+v, want Created=5", first)
	}
	arr := ciID(t, db, m.Name)
	if n := compCount(t, db, arr, "installed"); n != 7 {
		t.Errorf("компонентов = %d, want 7 (2 контроллера, 2 полки, 3 диска)", n)
	}
	if again := runConn(t, code); again.Created != 0 || again.Updated != 0 || again.Unchanged != 5 {
		t.Fatalf("повторный прогон = %+v, want 5 Unchanged", again)
	}
	if m.Sessions != 0 || m.Logins != m.Logouts {
		t.Errorf("сессии DeviceManager: открыто %d, входов %d, выходов %d — логаут обязателен (лимит сессий)", m.Sessions, m.Logins, m.Logouts)
	}

	// сбой источника: логаут всё равно выполняется, данные не меняются
	m.Do(func(s *huaweimock.Server) { s.Fail = true })
	if _, err := pipeline.RunConnector(context.Background(), code, pipeline.Options{UserID: 1}); err == nil {
		t.Fatal("сбой источника должен вернуть ошибку")
	}
	if m.Sessions != 0 {
		t.Errorf("после сбоя открытых сессий %d, want 0", m.Sessions)
	}
}

func TestRunConnector_LenovoSnapshotAndIdempotency(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := lenovomock.Default(tag)
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	code := newStorageConnector(t, db, "lenovo", "lenovo", srv.URL, "CMDB_LENOVO_TEST_PW", m.User, m.Password)

	first := runConn(t, code)
	if first.Created != 6 || first.Errors != 0 || first.Conflicts != 0 { // массив + 2 пула + 3 тома
		t.Fatalf("первый прогон = %+v, want Created=6", first)
	}
	if n := compCount(t, db, ciID(t, db, m.Label), "installed"); n != 6 {
		t.Errorf("компонентов = %d, want 6 (2 контроллера, полка, 3 диска)", n)
	}
	if again := runConn(t, code); again.Created != 0 || again.Updated != 0 || again.Unchanged != 6 {
		t.Fatalf("повторный прогон = %+v, want 6 Unchanged", again)
	}
}

func TestRunConnector_VNXProcessSnapshotAndSweep(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	a := vnxmock.Default(t.TempDir(), tag)
	if err := a.Write(); err != nil {
		t.Fatalf("mock: %v", err)
	}
	code := "vnx-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"host": "10.0.0.5", "binary": a.Script(), "secfile_path": "/etc/cmdb/vnx.sec"})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind, trust_level) VALUES ('vnx', 'api', 60) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config) VALUES ($1, 'vnx', 'Тестовая VNX', 'vnx', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatal(err)
	}

	first := runConn(t, code)
	if first.Created != 5 || first.Errors != 0 { // массив + пул + 3 LUN
		t.Fatalf("первый прогон = %+v, want Created=5", first)
	}
	arr := ciID(t, db, a.Name)
	if n := compCount(t, db, arr, "installed"); n != 4 {
		t.Errorf("компонентов = %d, want 4 (полка и 3 диска; пустой слот пропущен)", n)
	}
	if again := runConn(t, code); again.Created != 0 || again.Updated != 0 || again.Unchanged != 5 {
		t.Fatalf("повторный прогон = %+v, want 5 Unchanged", again)
	}

	// диск заменён, LUN удалён
	a.Disks[1].Serial = "SNEW" + tag
	a.LUNs = a.LUNs[:2]
	if err := a.Write(); err != nil {
		t.Fatal(err)
	}
	upd := runConn(t, code)
	if upd.Updated != 1 { // массив: компоненты изменились
		t.Fatalf("прогон после замены диска = %+v, want Updated=1", upd)
	}
	if n := compCount(t, db, arr, "removed"); n != 1 {
		t.Errorf("снятых компонентов = %d, want 1 (старый диск)", n)
	}
	for i := 0; i < 2; i++ {
		runConn(t, code)
	}
	var status sql.NullString
	db.QueryRow(`SELECT lifecycle_status FROM cmdb.ci WHERE name = $1`, a.Name+"/lun-rg-"+tag).Scan(&status)
	if status.String != "not_discovered" {
		t.Fatalf("lifecycle_status удалённого LUN = %q, want not_discovered", status.String)
	}

	// массив недоступен: снимок не строится, ничего не меняется
	a.Fail = true
	if err := a.Write(); err != nil {
		t.Fatal(err)
	}
	if _, err := pipeline.RunConnector(context.Background(), code, pipeline.Options{UserID: 1}); err == nil {
		t.Fatal("недоступный массив должен дать ошибку прогона")
	}
}

func TestRunConnector_PowerStoreSnapshotAndIdempotency(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	m := powerstoremock.Default(tag)
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	code := newStorageConnector(t, db, "powerstore", "powerstore", srv.URL, "CMDB_POWERSTORE_TEST_PW", m.User, m.Password)

	first := runConn(t, code)
	if first.Created != 3 || first.Errors != 0 || first.Conflicts != 0 { // массив + 2 тома (снимок отфильтрован)
		t.Fatalf("первый прогон = %+v, want Created=3", first)
	}
	arr := ciID(t, db, "ps-"+tag)
	if n := compCount(t, db, arr, "installed"); n != 2 {
		t.Errorf("компонентов = %d, want 2 (2 диска; у PowerStore нет пулов)", n)
	}
	vol := ciID(t, db, "ps-"+tag+"/vol-a-"+tag)
	if got := openRels(t, db, vol, "contains"); len(got) != 0 {
		t.Errorf("том без пула должен быть в связи contains массива, не иметь исходящих contains: %v", got)
	}
	if got := openRels(t, db, arr, "contains"); len(got) != 2 {
		t.Errorf("массив должен содержать оба тома напрямую (нет пулов): %v, want 2", got)
	}

	if again := runConn(t, code); again.Created != 0 || again.Updated != 0 || again.Unchanged != 3 {
		t.Fatalf("повторный прогон = %+v, want 3 Unchanged", again)
	}

	m.Do(func(s *powerstoremock.Server) { s.Fail = true })
	if _, err := pipeline.RunConnector(context.Background(), code, pipeline.Options{UserID: 1}); err == nil {
		t.Fatal("сбой источника должен вернуть ошибку")
	}
}
