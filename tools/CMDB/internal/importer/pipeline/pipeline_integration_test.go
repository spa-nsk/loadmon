// E4.11: реальный PostgreSQL, миграции и seed nc2026 (коннектор csv-server из
// database/seed/nc2026/007_import_connectors.sql). Чёрный ящик — только экспортируемые
// pipeline.Run/TemplateCSV и прямые проверки через SQL, как читает их будущий UI.
package pipeline_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	importcsv "cmdb/internal/importer/connectors/csv"
	"cmdb/internal/importer/pipeline"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

const connectorCode = "csv-server"

func header(t *testing.T) []string {
	t.Helper()
	spec := ci.SpecFor("server")
	if spec == nil {
		t.Fatal("нет ClassSpec для server")
	}
	return importcsv.Columns(*spec)
}

// buildCSV строит CSV-файл по полному заголовку класса: cols — только заполняемые колонки,
// остальные остаются пустыми (как обычное дело для необязательных атрибутов, Parse это поддерживает).
func buildCSV(t *testing.T, rows []map[string]string) string {
	t.Helper()
	h := header(t)
	var sb strings.Builder
	sb.WriteString(strings.Join(h, ",") + "\n")
	for _, row := range rows {
		vals := make([]string, len(h))
		for i, col := range h {
			vals[i] = row[col]
		}
		sb.WriteString(strings.Join(vals, ",") + "\n")
	}
	return sb.String()
}

func randHex(n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[rand.Intn(len(hex))]
	}
	return string(b)
}

func randUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s", randHex(8), randHex(4), randHex(4), randHex(4), randHex(12))
}

func run(t *testing.T, ctx context.Context, csvContent string, dryRun bool) pipeline.Summary {
	t.Helper()
	sum, err := pipeline.Run(ctx, connectorCode, strings.NewReader(csvContent), "csv", dryRun, 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return sum
}

func changeSetIDs(t *testing.T, db *sql.DB, ciID int64) []int64 {
	t.Helper()
	rows, err := db.Query(`SELECT DISTINCT change_set_id FROM audit.change_log WHERE ci_id = $1 AND change_set_id IS NOT NULL`, ciID)
	if err != nil {
		t.Fatalf("change_set_id для КЕ %d: %v", ciID, err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, id)
	}
	return out
}

// TestRun_IdempotentReimport — E4.11: повторный прогон с тем же файлом ничего не меняет (0 Created,
// 0 Updated, всё Unchanged) — импорт не должен «дребезжать» при опросе одного и того же источника.
func TestRun_IdempotentReimport(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	name := "IMP-IDEMP-" + testutil.Suffix()
	serial := "SN-" + testutil.Suffix()
	bios := randUUID()

	row := map[string]string{
		"name": name, "location_code": testutil.TestLocation, "serial_number": serial,
		"bios_uuid": bios, "cpu_count": "2", "ram_gb": "16384MB",
	}
	csvContent := buildCSV(t, []map[string]string{row})

	first := run(t, ctx, csvContent, false)
	if first.Created != 1 || first.Updated != 0 || first.Errors != 0 || first.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=1", first)
	}

	var ciID int64
	if err := db.QueryRow(`SELECT id FROM cmdb.ci WHERE name = $1`, name).Scan(&ciID); err != nil {
		t.Fatalf("поиск созданной КЕ: %v", err)
	}
	var ramGB sql.NullString
	if err := db.QueryRow(`SELECT ram_gb::text FROM cmdb.ci_server WHERE ci_id = $1`, ciID).Scan(&ramGB); err != nil {
		t.Fatalf("чтение ci_server: %v", err)
	}
	if ramGB.String != "16" {
		t.Errorf("ram_gb = %q, want 16 (16384MB нормализуется в ГБ)", ramGB.String)
	}

	second := run(t, ctx, csvContent, false)
	if second.Created != 0 || second.Updated != 0 || second.Unchanged != 1 || second.Errors != 0 {
		t.Fatalf("повторный прогон = %+v, want Unchanged=1 без создания/обновления", second)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE name = $1`, name).Scan(&count); err != nil {
		t.Fatalf("подсчёт КЕ: %v", err)
	}
	if count != 1 {
		t.Fatalf("количество КЕ с именем %q = %d, want 1 (повторный прогон не должен плодить дубликаты)", name, count)
	}
}

// TestRun_AmbiguousMatchIsConflict — E4.6: две КЕ с одинаковым serial_number и без bios_uuid
// в файле — неоднозначность, конфликт без автослияния.
func TestRun_AmbiguousMatchIsConflict(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	serial := "SN-AMBIG-" + testutil.Suffix()

	for _, n := range []string{"IMP-AMBIG-A-" + testutil.Suffix(), "IMP-AMBIG-B-" + testutil.Suffix()} {
		res, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{
			LocationCode: testutil.TestLocation, SerialNumber: serial, Reason: "фикстура: дубликат serial_number", UserID: 1,
		})
		if err != nil {
			t.Fatalf("CreateServer(%s): %v", n, err)
		}
		_ = res
	}

	row := map[string]string{"name": "IMP-AMBIG-NEW-" + testutil.Suffix(), "location_code": testutil.TestLocation, "serial_number": serial}
	sum := run(t, ctx, buildCSV(t, []map[string]string{row}), false)
	if sum.Conflicts != 1 || sum.Created != 0 || sum.Updated != 0 {
		t.Fatalf("прогон с неоднозначным serial_number = %+v, want Conflicts=1", sum)
	}

	var kind string
	if err := db.QueryRow(`SELECT kind FROM imp.conflict WHERE run_id = $1`, sum.RunID).Scan(&kind); err != nil {
		t.Fatalf("чтение imp.conflict: %v", err)
	}
	if kind != "ambiguous_match" {
		t.Errorf("kind = %q, want ambiguous_match", kind)
	}
}

// TestRun_LockedAttributeNotOverwritten — E4.7: атрибут, заблокированный вручную
// (cmdb.attr_source.is_locked), импорт не перезаписывает и заводит конфликт locked_attr,
// не прерывая применение остальных полей строки.
func TestRun_LockedAttributeNotOverwritten(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	name := "IMP-LOCK-" + testutil.Suffix()
	serial := "SN-" + testutil.Suffix()

	first := run(t, ctx, buildCSV(t, []map[string]string{{
		"name": name, "location_code": testutil.TestLocation, "serial_number": serial, "description": "исходное описание",
	}}), false)
	if first.Created != 1 {
		t.Fatalf("создание фикстуры = %+v, want Created=1", first)
	}
	var ciID int64
	if err := db.QueryRow(`SELECT id FROM cmdb.ci WHERE name = $1`, name).Scan(&ciID); err != nil {
		t.Fatalf("поиск КЕ: %v", err)
	}
	// createCI уже завёл строку attr_source для description (E4.7 пишет провенанс и при создании) —
	// апсертим, а не вставляем: строка обычно уже есть.
	if _, err := db.Exec(`
		INSERT INTO cmdb.attr_source (entity, entity_id, attr_name, is_locked) VALUES ('ci', $1, 'description', true)
		ON CONFLICT (entity, entity_id, attr_name) DO UPDATE SET is_locked = true
	`, ciID); err != nil {
		t.Fatalf("блокировка description: %v", err)
	}

	second := run(t, ctx, buildCSV(t, []map[string]string{{
		"name": name, "location_code": testutil.TestLocation, "serial_number": serial,
		"description": "попытка перезаписи", "criticality": "high",
	}}), false)
	if second.Updated != 1 {
		t.Fatalf("прогон с заблокированным description = %+v, want Updated=1 (criticality всё равно применяется)", second)
	}

	var desc, crit string
	if err := db.QueryRow(`SELECT COALESCE(description, ''), COALESCE(criticality, '') FROM cmdb.ci WHERE id = $1`, ciID).Scan(&desc, &crit); err != nil {
		t.Fatalf("чтение КЕ: %v", err)
	}
	if desc != "исходное описание" {
		t.Errorf("description = %q, заблокированное поле не должно было перезаписаться", desc)
	}
	if crit != "high" {
		t.Errorf("criticality = %q, want high (не заблокировано)", crit)
	}

	var lockedCount int
	if err := db.QueryRow(`SELECT count(*) FROM imp.conflict WHERE run_id = $1 AND kind = 'locked_attr'`, second.RunID).Scan(&lockedCount); err != nil {
		t.Fatalf("чтение imp.conflict: %v", err)
	}
	if lockedCount != 1 {
		t.Errorf("конфликтов locked_attr = %d, want 1", lockedCount)
	}
}

// TestRun_OneChangeSetPerRun — E4.8 DoD: несколько изменённых строк одного прогона попадают
// в ОДИН change_set, а не по одному на строку.
func TestRun_OneChangeSetPerRun(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()

	var rows []map[string]string
	var names []string
	for i := 0; i < 3; i++ {
		n := fmt.Sprintf("IMP-BATCH-%d-%s", i, testutil.Suffix())
		names = append(names, n)
		rows = append(rows, map[string]string{"name": n, "location_code": testutil.TestLocation, "serial_number": "SN-" + testutil.Suffix()})
	}

	sum := run(t, ctx, buildCSV(t, rows), false)
	if sum.Created != 3 || sum.ChangeSetID == 0 {
		t.Fatalf("прогон = %+v, want Created=3 и непустой ChangeSetID", sum)
	}

	seen := map[int64]bool{}
	for _, n := range names {
		var ciID int64
		if err := db.QueryRow(`SELECT id FROM cmdb.ci WHERE name = $1`, n).Scan(&ciID); err != nil {
			t.Fatalf("поиск КЕ %q: %v", n, err)
		}
		for _, csID := range changeSetIDs(t, db, ciID) {
			seen[csID] = true
		}
	}
	if len(seen) != 1 {
		t.Fatalf("change_set_id, затронутые прогоном = %v, want ровно один (DoD: N изменённых строк -> один change set)", seen)
	}
	for csID := range seen {
		if csID != sum.ChangeSetID {
			t.Errorf("change_set_id в audit.change_log = %d, want совпадение с Summary.ChangeSetID = %d", csID, sum.ChangeSetID)
		}
	}

	var runChangeSetID sql.NullInt64
	if err := db.QueryRow(`SELECT change_set_id FROM imp.run WHERE id = $1`, sum.RunID).Scan(&runChangeSetID); err != nil {
		t.Fatalf("чтение imp.run: %v", err)
	}
	if !runChangeSetID.Valid || runChangeSetID.Int64 != sum.ChangeSetID {
		t.Errorf("imp.run.change_set_id = %v, want %d", runChangeSetID, sum.ChangeSetID)
	}
}

// TestRun_NotDiscoveredAfterMisses — E4.8: КЕ, не встретившаяся в файле 3 прогона подряд после
// последнего обнаружения, помечается lifecycle_status='not_discovered', а не удаляется.
func TestRun_NotDiscoveredAfterMisses(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	name := "IMP-MISS-" + testutil.Suffix()

	created := run(t, ctx, buildCSV(t, []map[string]string{{
		"name": name, "location_code": testutil.TestLocation, "serial_number": "SN-" + testutil.Suffix(),
	}}), false)
	if created.Created != 1 {
		t.Fatalf("создание фикстуры = %+v, want Created=1", created)
	}
	var ciID int64
	if err := db.QueryRow(`SELECT id FROM cmdb.ci WHERE name = $1`, name).Scan(&ciID); err != nil {
		t.Fatalf("поиск КЕ: %v", err)
	}

	emptyCSV := strings.Join(header(t), ",") + "\n"
	var last pipeline.Summary
	for i := 0; i < 3; i++ {
		last = run(t, ctx, emptyCSV, false)
	}

	var status sql.NullString
	if err := db.QueryRow(`SELECT lifecycle_status FROM cmdb.ci WHERE id = $1`, ciID).Scan(&status); err != nil {
		t.Fatalf("чтение lifecycle_status: %v", err)
	}
	if status.String != "not_discovered" {
		t.Errorf("lifecycle_status = %q, want not_discovered после 3 пропусков подряд", status.String)
	}
	if last.NotDiscovered != 1 {
		t.Errorf("последний прогон NotDiscovered = %d, want 1", last.NotDiscovered)
	}

	var missedRuns int
	if err := db.QueryRow(`SELECT missed_runs FROM imp.match WHERE ci_id = $1`, ciID).Scan(&missedRuns); err != nil {
		t.Fatalf("чтение imp.match: %v", err)
	}
	if missedRuns != 3 {
		t.Errorf("missed_runs = %d, want 3", missedRuns)
	}
}

// TestRun_DryRunWritesNothing — предпросмотр (dry-run) только считает, ничего не пишет в cmdb.*.
func TestRun_DryRunWritesNothing(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	name := "IMP-DRYRUN-" + testutil.Suffix()

	sum := run(t, ctx, buildCSV(t, []map[string]string{{
		"name": name, "location_code": testutil.TestLocation, "serial_number": "SN-" + testutil.Suffix(),
	}}), true)
	if !sum.DryRun || sum.Created != 1 {
		t.Fatalf("dry-run = %+v, want DryRun=true, Created=1 (посчитано, не записано)", sum)
	}
	if sum.ChangeSetID != 0 {
		t.Errorf("ChangeSetID = %d, want 0 — dry-run не пишет change set", sum.ChangeSetID)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM cmdb.ci WHERE name = $1`, name).Scan(&count); err != nil {
		t.Fatalf("подсчёт КЕ: %v", err)
	}
	if count != 0 {
		t.Fatalf("КЕ с именем %q = %d, want 0 — dry-run не должен ничего создавать", name, count)
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM imp.run WHERE id = $1`, sum.RunID).Scan(&status); err != nil {
		t.Fatalf("чтение imp.run: %v", err)
	}
	if status != "succeeded" {
		t.Errorf("status прогона = %q, want succeeded", status)
	}
}
