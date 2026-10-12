package maintenance

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"cmdb/internal/testutil"
)

// newTestPartitioned — секционированная таблица только для этого теста (не hist.ci/audit.change_log
// — они общие для всех тестов в БД, нельзя полагаться на точные счётчики секций). Одна секция за
// пределами обслуживаемого диапазона — «старая», для проверки drop_old_partitions.
func newTestPartitioned(t *testing.T, db *sql.DB) (schema, table string) {
	t.Helper()
	table = "maint_test_" + testutil.Suffix()
	schema = "public"
	if _, err := db.Exec(`CREATE TABLE ` + schema + `.` + table + ` (
		id bigint GENERATED ALWAYS AS IDENTITY, ts timestamptz NOT NULL, PRIMARY KEY (id, ts)
	) PARTITION BY RANGE (ts)`); err != nil {
		t.Fatalf("создание тестовой секционированной таблицы: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + schema + `.` + table + `_default PARTITION OF ` + schema + `.` + table + ` DEFAULT`); err != nil {
		t.Fatalf("default-секция: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE ` + schema + `.` + table + `_old PARTITION OF ` + schema + `.` + table +
		` FOR VALUES FROM ('2000-01-01') TO ('2000-02-01')`); err != nil {
		t.Fatalf("старая секция: %v", err)
	}
	t.Cleanup(func() { db.Exec(`DROP TABLE IF EXISTS ` + schema + `.` + table + ` CASCADE`) })
	return schema, table
}

func partitionNames(t *testing.T, db *sql.DB, schema, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT c.relname FROM pg_inherits i JOIN pg_class c ON c.oid = i.inhrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE i.inhparent = ($1 || '.' || $2)::regclass AND n.nspname = $1 ORDER BY c.relname`, schema, table)
	if err != nil {
		t.Fatalf("список секций: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, name)
	}
	return out
}

func TestEnsurePartitions_CreatesMonthlyAndIdempotent(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	schema, table := newTestPartitioned(t, db)

	n, err := ensurePartitions(ctx, schema, table, "ts", "month", 1)
	if err != nil {
		t.Fatalf("ensurePartitions: %v", err)
	}
	if n != 2 { // текущий месяц + 1 вперёд
		t.Errorf("создано секций = %d, want 2", n)
	}
	names := partitionNames(t, db, schema, table)
	if len(names) != 4 { // default + old + 2 новых
		t.Fatalf("секций после первого вызова = %v, want 4", names)
	}

	// повторный вызов — идемпотентен, новых секций не создаёт
	n2, err := ensurePartitions(ctx, schema, table, "ts", "month", 1)
	if err != nil {
		t.Fatalf("ensurePartitions (повтор): %v", err)
	}
	if n2 != 0 {
		t.Errorf("повторный вызов создал %d секций, want 0", n2)
	}
}

func TestEnsurePartitions_MovesConflictingDefaultRows(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	schema, table := newTestPartitioned(t, db)

	// строка текущего месяца оказывается в DEFAULT (секций текущего месяца ещё нет)
	if _, err := db.Exec(`INSERT INTO ` + schema + `.` + table + ` (ts) VALUES (now())`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	var inDefaultBefore int
	db.QueryRow(`SELECT count(*) FROM ` + schema + `.` + table + `_default`).Scan(&inDefaultBefore)
	if inDefaultBefore != 1 {
		t.Fatalf("строка должна была попасть в default: %d", inDefaultBefore)
	}

	if _, err := ensurePartitions(ctx, schema, table, "ts", "month", 0); err != nil {
		t.Fatalf("ensurePartitions: %v", err)
	}

	var inDefaultAfter, total int
	db.QueryRow(`SELECT count(*) FROM ` + schema + `.` + table + `_default`).Scan(&inDefaultAfter)
	db.QueryRow(`SELECT count(*) FROM ` + schema + `.` + table).Scan(&total)
	if inDefaultAfter != 0 {
		t.Errorf("строка должна была переехать из default в новую секцию: осталось %d", inDefaultAfter)
	}
	if total != 1 {
		t.Errorf("строка потерялась при переносе: всего %d, want 1", total)
	}
}

func TestDropOldPartitions_RemovesOnlyOldNamedPartitions(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	schema, table := newTestPartitioned(t, db)

	if _, err := ensurePartitions(ctx, schema, table, "ts", "day", 1); err != nil {
		t.Fatalf("ensurePartitions: %v", err)
	}
	// секция «3 дня назад» — создана вручную (ensurePartitions никогда не создаёт секции в прошлом),
	// чтобы было что реально удалить ретеншном.
	stale := table + "_" + time.Now().AddDate(0, 0, -3).Format("20060102")
	from := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	to := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	if _, err := db.Exec(`CREATE TABLE ` + schema + `.` + stale + ` PARTITION OF ` + schema + `.` + table +
		` FOR VALUES FROM ('` + from + `') TO ('` + to + `')`); err != nil {
		t.Fatalf("создание устаревшей секции: %v", err)
	}

	dropped, err := dropOldPartitions(ctx, schema, table, "day", 365*24*time.Hour) // старше года — ничего не тронет
	if err != nil {
		t.Fatalf("dropOldPartitions: %v", err)
	}
	if dropped != 0 {
		t.Errorf("с горизонтом в год не должно быть удалений: удалено %d", dropped)
	}

	// ретеншн в 1 день — устаревшая (3 дня назад) секция должна уйти, сегодняшняя/завтрашняя и
	// нестандартно названная _old (не day-маска, не должна парситься и падать) — остаться.
	dropped2, err := dropOldPartitions(ctx, schema, table, "day", 24*time.Hour)
	if err != nil {
		t.Fatalf("dropOldPartitions (короткий ретеншн): %v", err)
	}
	if dropped2 != 1 {
		t.Errorf("удалено секций = %d, want 1 (только устаревшая)", dropped2)
	}
	after := partitionNames(t, db, schema, table)
	hasOld, hasStale := false, false
	for _, n := range after {
		if n == table+"_old" {
			hasOld = true
		}
		if n == stale {
			hasStale = true
		}
	}
	if !hasOld {
		t.Error("секция _old (нестандартное имя, не day-маска) не должна удаляться drop_old_partitions")
	}
	if hasStale {
		t.Error("устаревшая day-секция должна была удалиться")
	}
}

func TestRun_AllConfiguredTargetsSucceed(t *testing.T) {
	testutil.DB(t)
	res, err := Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// hist.ci/audit.change_log/imp.raw_record уже секционированы миграцией 0027 с запасом на
	// несколько периодов вперёд — повторный вызов в норме ничего нового не создаёт и ничего не
	// удаляет (ретеншн по умолчанию — 60 дней, свежие секции моложе).
	if res.PartitionsCreated < 0 || res.PartitionsDropped < 0 {
		t.Errorf("результат = %+v", res)
	}
}

func TestRawRecordRetentionDays_EnvOverride(t *testing.T) {
	if got := RawRecordRetentionDays(); got != 60 {
		t.Errorf("по умолчанию = %d, want 60", got)
	}
	t.Setenv("CMDB_RAW_RECORD_RETENTION_DAYS", "30")
	if got := RawRecordRetentionDays(); got != 30 {
		t.Errorf("с переопределением = %d, want 30", got)
	}
	t.Setenv("CMDB_RAW_RECORD_RETENTION_DAYS", "not-a-number")
	if got := RawRecordRetentionDays(); got != 60 {
		t.Errorf("с мусором в переменной должен остаться дефолт: %d", got)
	}
}
