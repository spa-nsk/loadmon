// Дашборд качества данных (plan.md E7.4) — реальный PostgreSQL, миграции и seed nc2026
// (internal/testutil). cmdb.ci — общая таблица для всего тестового набора, поэтому проверки на
// абсолютные числа здесь избегаются (см. TestGlobalLog_* в internal/services/history для того же
// приёма): где можно — проверяются структурные инварианты (сумма долей, разбиение источника
// подтверждения), где нельзя — берётся разница «до/после» вплотную к операции.
// Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/quality/...
package quality

import (
	"context"
	"testing"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

func TestPct_Helpers(t *testing.T) {
	if got := (ClassCompleteness{Total: 0}).Pct(); got != 100 {
		t.Errorf("ClassCompleteness{Total:0}.Pct() = %v, want 100 (нечему не хватать)", got)
	}
	if got := (ClassCompleteness{Total: 4, Complete: 3}).Pct(); got != 75 {
		t.Errorf("Pct() = %v, want 75", got)
	}
	if _, ok := OverallPct(nil); ok {
		t.Error("OverallPct(nil) ok=true, want false (нет данных)")
	}
	if pct, ok := OverallPct([]ClassCompleteness{{Total: 2, Complete: 1}, {Total: 2, Complete: 2}}); !ok || pct != 75 {
		t.Errorf("OverallPct = %v,%v, want 75,true", pct, ok)
	}
	if got := (NamingBreakdown{Count: 1}).Pct(0); got != 0 {
		t.Errorf("Pct(0) = %v, want 0", got)
	}
	if _, ok := (Staleness{SourceLinked: 0}).Confirmed7dPct(); ok {
		t.Error("Confirmed7dPct() при SourceLinked=0 ok=true, want false")
	}
	if pct, ok := (Staleness{SourceLinked: 10, Fresh7d: 9}).Confirmed7dPct(); !ok || pct != 90 {
		t.Errorf("Confirmed7dPct = %v,%v, want 90,true", pct, ok)
	}
	if n := DuplicateCICount([]DuplicateGroup{{CIIDs: []int64{1, 2}}, {CIIDs: []int64{2, 3}}}); n != 3 {
		t.Errorf("DuplicateCICount = %d, want 3 (id=2 не дважды)", n)
	}
}

func newServer(t *testing.T, ctx context.Context, reason string) int64 {
	t.Helper()
	res, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: reason, UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer (%s): %v", reason, err)
	}
	return res.CIID
}

func findPair(rows []ClassCompleteness, class, stage string) (ClassCompleteness, bool) {
	for _, r := range rows {
		if r.Class == class && r.Stage == stage {
			return r, true
		}
	}
	return ClassCompleteness{}, false
}

// TestCompleteness_ServerCommissioning — server/Commissioning требует cpu_count, ram_gb, os_family
// (database/seed/nc2026/006_attr_schema.sql). КЕ переводится в Commissioning прямым UPDATE, минуя
// ci.ChangeLifecycle: сам переход отклонил бы КЕ без этих атрибутов, а дашборду нужно уметь найти
// именно такую (кто-то завёл вручную в БД или прогнал импорт до того, как проверка появилась).
func TestCompleteness_ServerCommissioning(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	id := newServer(t, ctx, "фикстура полноты "+suffix)
	// прямой UPDATE минуя ci.ChangeLifecycle (см. комментарий к тесту), но не минуя changectx —
	// триггер БД (ADR-003) требует cmdb.change_set_id на любой записи в cmdb.*.
	if _, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура полноты " + suffix},
		func(ctx context.Context, tx orm.TxOrmer) error {
			_, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET lifecycle_stage = 'Commissioning' WHERE id = ?`, id).Exec()
			return err
		}); err != nil {
		t.Fatalf("перевод в Commissioning: %v", err)
	}

	before, err := Completeness(ctx)
	if err != nil {
		t.Fatalf("Completeness (до заполнения): %v", err)
	}
	pairBefore, ok := findPair(before, "server", "Commissioning")
	if !ok {
		t.Fatal("Completeness: нет пары server/Commissioning — фикстура только что создана")
	}
	if pairBefore.Complete >= pairBefore.Total {
		t.Fatalf("до заполнения атрибутов Complete=%d >= Total=%d — фикстура не должна считаться полной", pairBefore.Complete, pairBefore.Total)
	}

	if _, err := ci.UpsertExtension(ctx, ci.ExtensionInput{
		CIID: id, Table: "ci_server", Reason: "заполнение обязательных атрибутов " + suffix, UserID: 1,
		Attrs: map[string]any{"cpu_count": 8, "ram_gb": 32, "os_family": "Linux"},
	}); err != nil {
		t.Fatalf("UpsertExtension: %v", err)
	}

	after, err := Completeness(ctx)
	if err != nil {
		t.Fatalf("Completeness (после заполнения): %v", err)
	}
	pairAfter, ok := findPair(after, "server", "Commissioning")
	if !ok {
		t.Fatal("Completeness: пара server/Commissioning пропала после заполнения")
	}
	// Не строгое равенство: под `go test ./...` пакеты идут параллельно, и другой пакет
	// (например, TestChangeLifecycle_MatrixAndRetireClosesRelations в internal/services/ci) вполне
	// может провести свой server через Commissioning в этом же окне — важно лишь, что НАША КЕ
	// корректно перешла из «не хватает» в «полна», остальное — за рамками этого теста.
	if pairAfter.Total < pairBefore.Total {
		t.Fatalf("Total уменьшился: %d -> %d", pairBefore.Total, pairAfter.Total)
	}
	if pairAfter.Complete < pairBefore.Complete+1 {
		t.Errorf("Complete = %d, want >= %d (+1 за нашу КЕ)", pairAfter.Complete, pairBefore.Complete+1)
	}
}

// TestNaming_AllStatusesPresentAndSumMatchesTotal — структурный инвариант, не зависящий от того,
// что в общей тестовой базе накопили другие пакеты: все 4 статуса присутствуют (даже с нулём),
// сумма Count по статусам равна total.
func TestNaming_AllStatusesPresentAndSumMatchesTotal(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	rows, total, err := Naming(ctx)
	if err != nil {
		t.Fatalf("Naming: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("Naming: %d статусов, want 4: %+v", len(rows), rows)
	}
	sum := 0
	seen := map[string]bool{}
	for _, r := range rows {
		seen[r.Status] = true
		sum += r.Count
	}
	for _, s := range []string{"compliant", "legacy", "noncompliant", "exception"} {
		if !seen[s] {
			t.Errorf("Naming: нет статуса %q в ответе", s)
		}
	}
	if sum != total {
		t.Errorf("сумма Count = %d, total = %d", sum, total)
	}
}

// TestNaming_CompliantIncreasesByOne — КЕ, созданная через профиль именования, всегда compliant.
// Не строгое равенство: под `go test ./...` пакеты идут параллельно, другие тесты тоже создают
// КЕ через профиль именования в это же окно — важно лишь, что счётчик не остался на месте.
func TestNaming_CompliantIncreasesByOne(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	before, _, err := Naming(ctx)
	if err != nil {
		t.Fatalf("Naming (до): %v", err)
	}
	newServer(t, ctx, "фикстура именования "+testutil.Suffix())
	after, _, err := Naming(ctx)
	if err != nil {
		t.Fatalf("Naming (после): %v", err)
	}
	compliantBefore, compliantAfter := 0, 0
	for _, r := range before {
		if r.Status == "compliant" {
			compliantBefore = r.Count
		}
	}
	for _, r := range after {
		if r.Status == "compliant" {
			compliantAfter = r.Count
		}
	}
	if compliantAfter < compliantBefore+1 {
		t.Errorf("compliant = %d, want >= %d (+1 за нашу КЕ)", compliantAfter, compliantBefore+1)
	}
}

// TestStaleness_Invariants — Fresh7d/Stale30d/StaleOlder разбивают SourceLinked без остатка и без
// пересечения, SourceLinked+NoSource=TotalActive — это верно при любом состоянии общей тестовой
// базы (не зависит от того, что накопили другие пакеты), поэтому проверка ловит саму ошибку в SQL
// (например, неверную границу FILTER), а не гадает о контенте базы.
func TestStaleness_Invariants(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	s, err := ComputeStaleness(ctx)
	if err != nil {
		t.Fatalf("ComputeStaleness: %v", err)
	}
	if s.SourceLinked+s.NoSource() != s.TotalActive {
		t.Errorf("SourceLinked(%d) + NoSource(%d) != TotalActive(%d)", s.SourceLinked, s.NoSource(), s.TotalActive)
	}
	if s.Fresh7d+s.Stale30d+s.StaleOlder != s.SourceLinked {
		t.Errorf("Fresh7d(%d) + Stale30d(%d) + StaleOlder(%d) != SourceLinked(%d)", s.Fresh7d, s.Stale30d, s.StaleOlder, s.SourceLinked)
	}
}

// TestDuplicates_FindsSameSerialAndAssetTag — две КЕ без модели (model_id=NULL) с одинаковым
// серийным номером не запрещены уникальным индексом cmdb.ci (model_id, serial_number) — NULL
// делает пары различными для PostgreSQL; это и есть та дыра, что должен ловить дашборд, а не БД.
func TestDuplicates_FindsSameSerialAndAssetTag(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	serial := "SN-" + suffix
	tag := "TAG-" + suffix

	id1 := newServer(t, ctx, "фикстура дубля 1 "+suffix)
	id2 := newServer(t, ctx, "фикстура дубля 2 "+suffix)
	for _, id := range []int64{id1, id2} {
		if _, err := ci.Update(ctx, ci.UpdateInput{CIID: id, ExpectedVersion: 1, Reason: "серийный номер " + suffix, UserID: 1,
			SerialNumber: &serial, AssetTag: &tag}); err != nil {
			t.Fatalf("Update id=%d: %v", id, err)
		}
	}

	groups, err := Duplicates(ctx)
	if err != nil {
		t.Fatalf("Duplicates: %v", err)
	}
	var bySerial, byTag *DuplicateGroup
	for i := range groups {
		if groups[i].Value == serial {
			bySerial = &groups[i]
		}
		if groups[i].Value == tag {
			byTag = &groups[i]
		}
	}
	if bySerial == nil {
		t.Fatal("Duplicates: нет группы по серийному номеру " + serial)
	}
	if byTag == nil {
		t.Fatal("Duplicates: нет группы по инвентарному номеру " + tag)
	}
	for _, g := range []*DuplicateGroup{bySerial, byTag} {
		if len(g.CIIDs) != 2 || !containsBoth(g.CIIDs, id1, id2) {
			t.Errorf("группа %q.CIIDs = %v, want [%d %d]", g.Value, g.CIIDs, id1, id2)
		}
	}
	if n := DuplicateCICount(groups); n < 2 {
		t.Errorf("DuplicateCICount = %d, want >= 2 (наши id1/id2 хотя бы)", n)
	}
}

func containsBoth(ids []int64, a, b int64) bool {
	seen := map[int64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	return seen[a] && seen[b]
}

// TestOpenConflicts_Invariant — сумма по kind равна общему числу открытых конфликтов, старше 14
// дней не больше общего числа; верно независимо от того, что накопили другие пакеты/прогоны.
func TestOpenConflicts_Invariant(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	cs, err := OpenConflicts(ctx)
	if err != nil {
		t.Fatalf("OpenConflicts: %v", err)
	}
	sum := 0
	for _, n := range cs.ByKind {
		sum += n
	}
	if sum != cs.Open {
		t.Errorf("сумма ByKind = %d, Open = %d", sum, cs.Open)
	}
	if cs.OpenOver14d > cs.Open {
		t.Errorf("OpenOver14d(%d) > Open(%d)", cs.OpenOver14d, cs.Open)
	}
}

// TestComputeSaveHistory_RoundTrip — Compute → Save → History видит записанный снимок, доли
// именования в сумме дают ~100 % (с округлением numeric(5,2) до 2 знаков — округление могло
// сместить сумму не больше чем на несколько сотых на статус, три статуса, поэтому epsilon 0.1
// с запасом, не подгонка под конкретное число).
func TestComputeSaveHistory_RoundTrip(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()

	snap, err := Compute(ctx)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if err := Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	pts, err := History(ctx, 1)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(pts) != 1 {
		t.Fatalf("History(1) вернул %d строк, want 1", len(pts))
	}
	p := pts[0]
	if p.Date != snap.Date {
		t.Errorf("History[0].Date = %q, want %q", p.Date, snap.Date)
	}
	sumNaming := p.NamingCompliantPct + p.NamingLegacyPct + p.NamingNoncompliantPct + p.NamingExceptionPct
	if snap.NamingTotal > 0 && (sumNaming < 99.9 || sumNaming > 100.1) {
		t.Errorf("сумма долей именования = %.2f, want ~100", sumNaming)
	}
	if p.TotalActiveCI != snap.Staleness.TotalActive {
		t.Errorf("History.TotalActiveCI = %d, want %d", p.TotalActiveCI, snap.Staleness.TotalActive)
	}

	// повторное сохранение того же дня не размножает строки (снимок дня, не журнал)
	if err := Save(ctx, snap); err != nil {
		t.Fatalf("Save (повтор): %v", err)
	}
	pts2, err := History(ctx, 0)
	if err != nil {
		t.Fatalf("History(0): %v", err)
	}
	count := 0
	for _, p := range pts2 {
		if p.Date == snap.Date {
			count++
		}
	}
	if count != 1 {
		t.Errorf("после повторного Save на дату %s приходится %d строк, want 1", snap.Date, count)
	}
}
