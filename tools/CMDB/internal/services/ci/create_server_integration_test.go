// Интеграционные тесты требуют PostgreSQL с примененными миграциями и seed nc2026:
// CMDB_TEST_DB_DSN=... go test ./internal/services/ci/... -v
// Без CMDB_TEST_DB_DSN тесты пропускаются (см. internal/testutil).
package ci

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/platform/asof"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

// newServer создаёт КЕ-фикстуру (класс server) и возвращает результат создания.
func newServer(t *testing.T, reason string) CreateServerResult {
	t.Helper()
	res, err := CreateServer(context.Background(), naming.DefaultRegistry(), CreateServerInput{
		Site: testutil.TestSite, LocationCode: testutil.TestLocation, Reason: reason, UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer (фикстура %q): %v", reason, err)
	}
	return res
}

func strp(s string) *string { return &s }

// AC-1: создание сервера через ChangeContext присваивает имя по профилю CMDB-NC-2026,
// создаёт change_set source=manual и по одной строке audit.change_log на атрибут (op='I').
func TestCreateServer_AssignsNameAndAudit(t *testing.T) {
	db := testutil.DB(t)

	res := newServer(t, "интеграционный тест AC-1")
	if !strings.HasPrefix(res.Name, testutil.TestSite+"-S-") {
		t.Errorf("Name = %q, ожидался префикс %s-S-", res.Name, testutil.TestSite)
	}

	var sourceKind string
	if err := db.QueryRow(`SELECT source_kind FROM audit.change_set WHERE id = $1`, res.ChangeSetID).Scan(&sourceKind); err != nil {
		t.Fatalf("select change_set: %v", err)
	}
	if sourceKind != "manual" {
		t.Errorf("source_kind = %q, want manual", sourceKind)
	}

	var logRows int
	if err := db.QueryRow(`SELECT count(*) FROM audit.change_log WHERE change_set_id = $1 AND entity = 'ci' AND op = 'I'`, res.ChangeSetID).Scan(&logRows); err != nil {
		t.Fatalf("select change_log: %v", err)
	}
	if logRows != 1 {
		t.Errorf("audit.change_log INSERT-строк для ci = %d, want 1", logRows)
	}
}

// AC-10: прямой UPDATE cmdb.ci мимо ChangeContext (без SET LOCAL cmdb.change_set_id) должен
// быть отклонён триггером hist.versioning().
func TestDirectUpdateWithoutChangeContext_Rejected(t *testing.T) {
	testutil.DB(t)
	res := newServer(t, "фикстура AC-10")

	o := orm.NewOrm()
	_, err := o.RawWithCtx(context.Background(), `UPDATE cmdb.ci SET description = 'обход' WHERE id = ?`, res.CIID).Exec()
	if err == nil {
		t.Fatal("прямой UPDATE мимо ChangeContext должен быть отклонён, но прошёл успешно")
	}
	if !strings.Contains(err.Error(), "change_set_id") {
		t.Errorf("ошибка %q не упоминает change_set_id — убедитесь, что сработал именно триггер hist.versioning()", err.Error())
	}
}

// E2.8/E2.11: Update меняет поля, увеличивает row_version, пишет закрытую версию в hist.ci и
// diff в audit.change_log; устаревшая версия отклоняется ErrVersionConflict.
func TestUpdate_OptimisticLockAndHistory(t *testing.T) {
	db := testutil.DB(t)
	res := newServer(t, "фикстура Update")

	upd, err := Update(context.Background(), UpdateInput{
		CIID: res.CIID, ExpectedVersion: 1, Reason: "уточнение описания", UserID: 1,
		Description: strp("новое описание"), Criticality: strp("high"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if upd.RowVersion != 2 {
		t.Errorf("RowVersion = %d, want 2", upd.RowVersion)
	}

	var histRows int
	if err := db.QueryRow(`SELECT count(*) FROM hist.ci WHERE id = $1 AND row_version = 1 AND hist_op = 'U'`, res.CIID).Scan(&histRows); err != nil {
		t.Fatalf("select hist.ci: %v", err)
	}
	if histRows != 1 {
		t.Errorf("hist.ci: версия 1 найдена %d раз, want 1", histRows)
	}

	var diffAttrs int
	if err := db.QueryRow(`SELECT count(*) FROM audit.change_log WHERE change_set_id = $1 AND entity = 'ci' AND op = 'U'
		AND attr_name IN ('description', 'criticality')`, upd.ChangeSetID).Scan(&diffAttrs); err != nil {
		t.Fatalf("select change_log: %v", err)
	}
	if diffAttrs != 2 {
		t.Errorf("change_log: diff-строк по description/criticality = %d, want 2", diffAttrs)
	}

	// Повтор с устаревшей версией.
	_, err = Update(context.Background(), UpdateInput{
		CIID: res.CIID, ExpectedVersion: 1, Reason: "устаревшая правка", UserID: 1, Description: strp("конфликт"),
	})
	if !errors.Is(err, svcerr.ErrVersionConflict) {
		t.Errorf("устаревшая версия: err = %v, want ErrVersionConflict", err)
	}

	_, err = Update(context.Background(), UpdateInput{
		CIID: -1, ExpectedVersion: 1, Reason: "нет такой КЕ", UserID: 1, Description: strp("x"),
	})
	if !errors.Is(err, svcerr.ErrNotFound) {
		t.Errorf("несуществующая КЕ: err = %v, want ErrNotFound", err)
	}

	_, err = Update(context.Background(), UpdateInput{CIID: res.CIID, ExpectedVersion: 2, Reason: "пусто", UserID: 1})
	if !errors.Is(err, svcerr.ErrNoChanges) {
		t.Errorf("пустое изменение: err = %v, want ErrNoChanges", err)
	}
}

// E3.2: internal/filters.AsOf кладёт ?asof= в контекст запроса — Get читает cmdb.ci через
// cmdb.ci_asof(ts) (E2.10), а не текущую строку, когда контекст его несёт.
func TestGet_RespectsAsOfContext(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	res := newServer(t, "фикстура AS OF")

	before, err := Get(ctx, res.CIID)
	if err != nil {
		t.Fatalf("Get (до правки): %v", err)
	}
	beforeDesc := before.CI["description"]
	asOfTS := time.Now()
	time.Sleep(20 * time.Millisecond) // sys_period нижняя граница правки должна быть строго позже asOfTS

	if _, err := Update(ctx, UpdateInput{CIID: res.CIID, ExpectedVersion: 1, Reason: "правка для AS OF", UserID: 1,
		Description: strp("описание после правки")}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	pastCtx := asof.WithTime(ctx, asOfTS)
	past, err := Get(pastCtx, res.CIID)
	if err != nil {
		t.Fatalf("Get (asof в прошлом): %v", err)
	}
	if past.CI["description"] != beforeDesc {
		t.Errorf("Get с asof в прошлом: description = %q, want %q (состояние до правки)", past.CI["description"], beforeDesc)
	}
	if past.CI["row_version"] != "1" {
		t.Errorf("Get с asof в прошлом: row_version = %q, want 1", past.CI["row_version"])
	}

	current, err := Get(ctx, res.CIID)
	if err != nil {
		t.Fatalf("Get (текущее состояние): %v", err)
	}
	if current.CI["description"] != "описание после правки" {
		t.Errorf("Get без asof: description = %q, want текущее значение", current.CI["description"])
	}
}

// E2.8: смена стадии ЖЦ по матрице; вывод из эксплуатации закрывает связи.
func TestChangeLifecycle_MatrixAndRetireClosesRelations(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	a := newServer(t, "фикстура ЖЦ A")
	b := newServer(t, "фикстура ЖЦ B")

	if _, err := ChangeLifecycle(ctx, ChangeLifecycleInput{CIID: a.CIID, ExpectedVersion: 1, ToStage: StageOperational, Reason: "минуя ввод", UserID: 1}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Planned → Operational: err = %v, want ErrInvalidTransition", err)
	}

	// E2.9: Commissioning/Operational требуют заполненных атрибутов (ref.ci_class.attr_schema,
	// database/seed/nc2026/006_attr_schema.sql) — фикстура не про атрибуты, поэтому заполняем их
	// одним upsert перед переходами, не проверяя это отдельно (см. TestChangeLifecycle_RequiresAttrs).
	if _, err := UpsertExtension(ctx, ExtensionInput{CIID: a.CIID, Table: "ci_server", Reason: "фикстура атрибутов", UserID: 1,
		Attrs: map[string]any{"cpu_count": 2, "ram_gb": 16, "os_family": "Linux", "bios_uuid": "11111111-1111-1111-1111-111111111111"},
	}); err != nil {
		t.Fatalf("UpsertExtension (фикстура атрибутов): %v", err)
	}

	version := 1
	for _, stage := range []string{StageCommissioning, StageOperational} {
		if _, err := ChangeLifecycle(ctx, ChangeLifecycleInput{CIID: a.CIID, ExpectedVersion: version, ToStage: stage, Reason: "ввод в эксплуатацию", UserID: 1}); err != nil {
			t.Fatalf("→ %s: %v", stage, err)
		}
		version++
	}

	if _, err := Retire(ctx, a.CIID, version, "напрямую из Operational", 1); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Operational → Retired: err = %v, want ErrInvalidTransition", err)
	}

	// Связь a → b, действующая на момент вывода.
	if _, err := db.Exec(`INSERT INTO ref.rel_type (rel_code, name_fwd, name_rev) VALUES ('depends_on', 'зависит от', 'используется') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("rel_type: %v", err)
	}
	_, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура связи"},
		func(ctx context.Context, tx orm.TxOrmer) error {
			_, err := tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci_rel (rel_type, src_ci_id, dst_ci_id) VALUES ('depends_on', ?, ?)`, a.CIID, b.CIID).Exec()
			return err
		})
	if err != nil {
		t.Fatalf("создание связи: %v", err)
	}
	time.Sleep(20 * time.Millisecond) // valid_period нижняя граница < now() вывода

	for _, stage := range []string{StageDecommissioning, StageRetired} {
		if _, err := ChangeLifecycle(ctx, ChangeLifecycleInput{CIID: a.CIID, ExpectedVersion: version, ToStage: stage, Reason: "вывод", UserID: 1}); err != nil {
			t.Fatalf("→ %s: %v", stage, err)
		}
		version++
	}

	var open int
	if err := db.QueryRow(`SELECT count(*) FROM cmdb.ci_rel WHERE src_ci_id = $1 AND upper_inf(valid_period)`, a.CIID).Scan(&open); err != nil {
		t.Fatalf("select ci_rel: %v", err)
	}
	if open != 0 {
		t.Errorf("после вывода осталось %d действующих связей, want 0", open)
	}
}

// E2.9: ref.ci_class.attr_schema (database/seed/nc2026/006_attr_schema.sql) требует cpu_count/
// ram_gb/os_family у сервера начиная со стадии Commissioning, bios_uuid — с Operational.
// Переход блокируется, пока эти атрибуты не заполнены, и проходит сразу после.
func TestChangeLifecycle_RequiresAttrs(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	s := newServer(t, "фикстура обязательных атрибутов")

	_, err := ChangeLifecycle(ctx, ChangeLifecycleInput{CIID: s.CIID, ExpectedVersion: 1, ToStage: StageCommissioning, Reason: "без атрибутов", UserID: 1})
	if !errors.Is(err, svcerr.ErrInvalidState) {
		t.Fatalf("Commissioning без атрибутов: err = %v, want ErrInvalidState", err)
	}
	for _, want := range []string{"cpu_count", "ram_gb", "os_family"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("текст ошибки не упоминает %q: %v", want, err)
		}
	}

	if _, err := UpsertExtension(ctx, ExtensionInput{CIID: s.CIID, Table: "ci_server", Reason: "заполнение атрибутов", UserID: 1,
		Attrs: map[string]any{"cpu_count": 2, "ram_gb": 16, "os_family": "Linux"},
	}); err != nil {
		t.Fatalf("UpsertExtension: %v", err)
	}
	if _, err := ChangeLifecycle(ctx, ChangeLifecycleInput{CIID: s.CIID, ExpectedVersion: 1, ToStage: StageCommissioning, Reason: "атрибуты заполнены", UserID: 1}); err != nil {
		t.Fatalf("Commissioning после заполнения атрибутов: %v", err)
	}

	// Operational дополнительно требует bios_uuid — ещё не заполнен.
	_, err = ChangeLifecycle(ctx, ChangeLifecycleInput{CIID: s.CIID, ExpectedVersion: 2, ToStage: StageOperational, Reason: "без bios_uuid", UserID: 1})
	if !errors.Is(err, svcerr.ErrInvalidState) || !strings.Contains(err.Error(), "bios_uuid") {
		t.Fatalf("Operational без bios_uuid: err = %v, want ErrInvalidState упоминающий bios_uuid", err)
	}
}

// E2.10/E2.11: cmdb.ci_asof(ts) возвращает точное прошлое состояние.
func TestCIAsOf_ReturnsPastState(t *testing.T) {
	db := testutil.DB(t)
	res := newServer(t, "фикстура AS OF")

	var beforeUpdate time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&beforeUpdate); err != nil {
		t.Fatalf("clock_timestamp: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	if _, err := Update(context.Background(), UpdateInput{
		CIID: res.CIID, ExpectedVersion: 1, Reason: "AS OF", UserID: 1, Description: strp("после правки"),
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var now time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&now); err != nil {
		t.Fatalf("clock_timestamp: %v", err)
	}

	desc := func(ts time.Time) sql.NullString {
		var d sql.NullString
		if err := db.QueryRow(`SELECT description FROM cmdb.ci_asof($1) WHERE id = $2`, ts, res.CIID).Scan(&d); err != nil {
			t.Fatalf("ci_asof(%s): %v", ts, err)
		}
		return d
	}
	if d := desc(beforeUpdate); d.Valid {
		t.Errorf("AS OF до правки: description = %q, want NULL", d.String)
	}
	if d := desc(now); d.String != "после правки" {
		t.Errorf("AS OF сейчас: description = %q, want %q", d.String, "после правки")
	}

	// До создания КЕ на эту дату не существовало.
	var cnt int
	if err := db.QueryRow(`SELECT count(*) FROM cmdb.ci_asof($1) WHERE id = $2`, beforeUpdate.Add(-24*time.Hour), res.CIID).Scan(&cnt); err != nil {
		t.Fatalf("ci_asof(до создания): %v", err)
	}
	if cnt != 0 {
		t.Errorf("AS OF за сутки до создания вернул %d строк, want 0", cnt)
	}
}
