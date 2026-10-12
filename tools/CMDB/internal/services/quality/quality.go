// Package quality — дашборд качества данных (plan.md E7.4; целевые показатели — intent.md §9):
// полнота обязательных атрибутов по классам, соответствие профилю именования, устаревание (когда
// КЕ последний раз подтвердила хотя бы один источник), дубликаты, открытые конфликты импорта и
// дневной снимок этих чисел для отображения динамики — без снимков график по дням не из чего
// строить, а пересчитывать всю историю на каждый показ дашборда дорого при объёме из intent §9
// (сотни тысяч КЕ).
package quality

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/attrschema"
	"cmdb/internal/services/ci"
)

// activeFilter — везде в этом пакете «действующая КЕ» значит «не выведенная из эксплуатации»,
// как и во всём остальном приложении (E2.6): вывод из эксплуатации — смена стадии, а не удаление,
// и КЕ на складе истории не портит текущие показатели качества.
const activeFilter = `lifecycle_stage NOT IN ('Retired', 'Disposed')`

// ClassCompleteness — заполненность атрибутов, обязательных для стадии ЖЦ, среди действующих КЕ
// одного класса. Проверяется то же самое, что internal/services/ci.ChangeLifecycle проверяет при
// переходе в эту стадию (ref.ci_class.attr_schema, RequiredByStage) — здесь агрегированно по всем
// КЕ, а не по одной. Пары класс/стадия без обязательных атрибутов на этой стадии в отчёт не входят.
type ClassCompleteness struct {
	Class     string
	ClassName string
	Stage     string
	Total     int
	Complete  int
}

// Pct — доля КЕ с заполненными обязательными атрибутами; пустой класс считается полным (нечему не хватать).
func (c ClassCompleteness) Pct() float64 {
	if c.Total == 0 {
		return 100
	}
	return float64(c.Complete) / float64(c.Total) * 100
}

// Completeness — полнота по всем парам (класс, стадия), встретившимся среди действующих КЕ.
func Completeness(ctx context.Context) ([]ClassCompleteness, error) {
	pairs, err := repo.QueryRows(ctx, `
		SELECT DISTINCT c.ci_class, c.lifecycle_stage, k.name AS class_name,
		       COALESCE(k.ext_table, '') AS ext_table, COALESCE(k.attr_schema::text, '{}') AS attr_schema
		FROM cmdb.ci c JOIN ref.ci_class k ON k.class_code = c.ci_class
		WHERE `+activeFilter+` AND k.ext_table IS NOT NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("quality: пары класс/стадия: %w", err)
	}

	colsCache := map[string]map[string]string{}
	var out []ClassCompleteness
	for _, p := range pairs {
		extTable := p["ext_table"]
		if extTable == "" {
			continue
		}
		schema, err := attrschema.Parse(p["attr_schema"])
		if err != nil {
			return nil, fmt.Errorf("quality: %s.attr_schema: %w", p["ci_class"], err)
		}
		required := schema.Required(p["lifecycle_stage"])
		if len(required) == 0 {
			continue
		}
		cols, ok := colsCache[extTable]
		if !ok {
			if cols, err = ci.ExtensionColumns(ctx, orm.NewOrm(), extTable); err != nil {
				return nil, err
			}
			colsCache[extTable] = cols
		}
		var conds []string
		for _, f := range required {
			if _, ok := cols[f]; ok { // имена колонок берутся из attr_schema (JSON, правится через /admin/ref) — сверяем со списком реальных колонок, прежде чем подставлять в SQL
				conds = append(conds, `"`+f+`" IS NOT NULL`)
			}
		}
		if len(conds) == 0 {
			continue
		}
		row, err := repo.QueryRow(ctx, `
			SELECT count(*)::text AS total, count(*) FILTER (WHERE `+strings.Join(conds, " AND ")+`)::text AS complete
			FROM cmdb.`+extTable+` e JOIN cmdb.ci c ON c.id = e.ci_id
			WHERE c.ci_class = ? AND c.lifecycle_stage = ?
		`, p["ci_class"], p["lifecycle_stage"])
		if err != nil {
			return nil, fmt.Errorf("quality: полнота %s/%s: %w", p["ci_class"], p["lifecycle_stage"], err)
		}
		cc := ClassCompleteness{Class: p["ci_class"], ClassName: p["class_name"], Stage: p["lifecycle_stage"]}
		fmt.Sscan(row["total"], &cc.Total)
		fmt.Sscan(row["complete"], &cc.Complete)
		out = append(out, cc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pct() != out[j].Pct() {
			return out[i].Pct() < out[j].Pct() // худшие классы — первыми, их и нужно чинить
		}
		return out[i].Class+out[i].Stage < out[j].Class+out[j].Stage
	})
	return out, nil
}

// OverallPct — взвешенная доля по всем классам (не среднее по классам — иначе маленький класс
// с одной незаполненной КЕ веcил бы столько же, сколько тысяча серверов); ok=false — ни у одного
// класса в текущих данных нет обязательных атрибутов на его стадии, процент показывать нечем.
func OverallPct(rows []ClassCompleteness) (pct float64, ok bool) {
	var total, complete int
	for _, r := range rows {
		total += r.Total
		complete += r.Complete
	}
	if total == 0 {
		return 0, false
	}
	return float64(complete) / float64(total) * 100, true
}

// NamingBreakdown — число действующих КЕ с данным статусом соответствия профилю именования.
type NamingBreakdown struct {
	Status string
	Count  int
}

// namingStatuses — порядок вывода (internal/controllers/ui.namingStatuses уже перечисляет их для форм).
var namingStatuses = []string{"compliant", "legacy", "noncompliant", "exception"}

// Naming — разбивка действующих КЕ по naming_status (E7.4: «доля legacy/noncompliant»).
func Naming(ctx context.Context) ([]NamingBreakdown, int, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT naming_status, count(*)::text AS n FROM cmdb.ci WHERE `+activeFilter+` GROUP BY naming_status`)
	if err != nil {
		return nil, 0, fmt.Errorf("quality: соответствие именованию: %w", err)
	}
	counts := map[string]int{}
	total := 0
	for _, r := range rows {
		var n int
		fmt.Sscan(r["n"], &n)
		counts[r["naming_status"]] = n
		total += n
	}
	out := make([]NamingBreakdown, len(namingStatuses))
	for i, s := range namingStatuses {
		out[i] = NamingBreakdown{Status: s, Count: counts[s]}
	}
	return out, total, nil
}

// Pct — доля данного статуса среди total действующих КЕ; total=0 — 0 (пустая база, нечего делить).
func (b NamingBreakdown) Pct(total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(b.Count) / float64(total) * 100
}

// Staleness — когда действующие КЕ последний раз подтверждал источник (cmdb.ci.last_seen_at,
// обновляется импортом при каждом повторном обнаружении, E4.8/E5). У КЕ, введённых только вручную,
// last_seen_at всегда NULL — это не «устарела», а «источника никогда не было»; посчитана отдельно
// (NoSource), не как «устаревшая», иначе KPI «подтверждено источником за 7 дней ≥ 90 %»
// (intent.md §9) сравнивал бы источники с ручным вводом, для которого сам KPI не имеет смысла.
type Staleness struct {
	TotalActive  int
	SourceLinked int
	Fresh7d      int // last_seen_at не старше 7 дней
	Stale30d     int // 7–30 дней
	StaleOlder   int // старше 30 дней
}

func (s Staleness) NoSource() int { return s.TotalActive - s.SourceLinked }

// Confirmed7dPct — доля SourceLinked, подтверждённых за 7 дней; ok=false — SourceLinked=0 (ни одна
// действующая КЕ не привязана к источнику, процент показывать нечем).
func (s Staleness) Confirmed7dPct() (pct float64, ok bool) {
	if s.SourceLinked == 0 {
		return 0, false
	}
	return float64(s.Fresh7d) / float64(s.SourceLinked) * 100, true
}

// ComputeStaleness — см. Staleness.
func ComputeStaleness(ctx context.Context) (Staleness, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT count(*)::text AS total,
		       count(*) FILTER (WHERE last_seen_at IS NOT NULL)::text AS linked,
		       count(*) FILTER (WHERE last_seen_at >= now() - interval '7 days')::text AS fresh7,
		       count(*) FILTER (WHERE last_seen_at < now() - interval '7 days' AND last_seen_at >= now() - interval '30 days')::text AS stale30,
		       count(*) FILTER (WHERE last_seen_at < now() - interval '30 days')::text AS older
		FROM cmdb.ci WHERE `+activeFilter)
	if err != nil {
		return Staleness{}, fmt.Errorf("quality: устаревание: %w", err)
	}
	var s Staleness
	fmt.Sscan(row["total"], &s.TotalActive)
	fmt.Sscan(row["linked"], &s.SourceLinked)
	fmt.Sscan(row["fresh7"], &s.Fresh7d)
	fmt.Sscan(row["stale30"], &s.Stale30d)
	fmt.Sscan(row["older"], &s.StaleOlder)
	return s, nil
}

// DuplicateGroup — несколько действующих КЕ с одинаковым непустым значением поля, которое должно
// быть уникальным для физического актива (серийный номер, инвентарный номер). Эвристика, не точный
// дедуп: cmdb.ci и так не даёт создать дубль по имени внутри класса или по (модель, серийный номер)
// — она находит то, что эти ограничения не ловят: тот же серийный номер под другой моделью
// (опечатка при вводе) или на КЕ другого класса (одна и та же физическая машина заведена дважды
// разными источниками и не связалась автоматически, E5.5).
type DuplicateGroup struct {
	Field string // "серийный номер" | "инвентарный номер"
	Value string
	CIIDs []int64
	Names []string
}

func duplicatesByColumn(ctx context.Context, column, label string) ([]DuplicateGroup, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT `+column+` AS value, string_agg(id::text, ',' ORDER BY id) AS ids, string_agg(name, ', ' ORDER BY id) AS names
		FROM cmdb.ci WHERE `+activeFilter+` AND `+column+` IS NOT NULL AND `+column+` <> ''
		GROUP BY `+column+` HAVING count(*) > 1`)
	if err != nil {
		return nil, fmt.Errorf("quality: дубликаты по %s: %w", label, err)
	}
	out := make([]DuplicateGroup, len(rows))
	for i, r := range rows {
		g := DuplicateGroup{Field: label, Value: r["value"], Names: strings.Split(r["names"], ", ")}
		for _, s := range strings.Split(r["ids"], ",") {
			if id, err := strconv.ParseInt(s, 10, 64); err == nil {
				g.CIIDs = append(g.CIIDs, id)
			}
		}
		out[i] = g
	}
	return out, nil
}

// Duplicates ищет совпадения по серийному номеру и по инвентарному номеру отдельно (общая КЕ,
// совпавшая по обоим сразу, попадёт в оба списка — это по-прежнему две разные причины проверить
// её вручную, а не один и тот же сигнал).
func Duplicates(ctx context.Context) ([]DuplicateGroup, error) {
	serials, err := duplicatesByColumn(ctx, "serial_number", "серийный номер")
	if err != nil {
		return nil, err
	}
	tags, err := duplicatesByColumn(ctx, "asset_tag", "инвентарный номер")
	if err != nil {
		return nil, err
	}
	return append(serials, tags...), nil
}

// DuplicateCICount — сколько разных КЕ задето хотя бы одной группой совпадений (для заголовка
// дашборда и снимка); одна КЕ, встретившаяся в нескольких группах, считается один раз.
func DuplicateCICount(groups []DuplicateGroup) int {
	seen := map[int64]bool{}
	for _, g := range groups {
		for _, id := range g.CIIDs {
			seen[id] = true
		}
	}
	return len(seen)
}

// ConflictSummary — открытые конфликты импорта (imp.conflict, E4.6/E4.7). KPI (intent §9):
// открытых конфликтов старше 14 дней должно быть 0 — они означают, что очередь никто не разбирает.
type ConflictSummary struct {
	Open        int
	OpenOver14d int
	ByKind      map[string]int
}

// OpenConflicts — см. ConflictSummary.
func OpenConflicts(ctx context.Context) (ConflictSummary, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT kind, count(*)::text AS n, count(*) FILTER (WHERE created_at < now() - interval '14 days')::text AS old
		FROM imp.conflict WHERE status = 'open' GROUP BY kind`)
	if err != nil {
		return ConflictSummary{}, fmt.Errorf("quality: открытые конфликты: %w", err)
	}
	cs := ConflictSummary{ByKind: map[string]int{}}
	for _, r := range rows {
		var n, old int
		fmt.Sscan(r["n"], &n)
		fmt.Sscan(r["old"], &old)
		cs.ByKind[r["kind"]] = n
		cs.Open += n
		cs.OpenOver14d += old
	}
	return cs, nil
}

// Snapshot — все показатели на момент вычисления (для страницы дашборда и для записи в
// rpt.quality_snapshot).
type Snapshot struct {
	Date         string // YYYY-MM-DD (UTC)
	ComputedAt   time.Time
	Completeness []ClassCompleteness
	Naming       []NamingBreakdown
	NamingTotal  int
	Staleness    Staleness
	Duplicates   []DuplicateGroup
	Conflicts    ConflictSummary
}

// AttrsCompletePct — как OverallPct(s.Completeness).
func (s Snapshot) AttrsCompletePct() (float64, bool) { return OverallPct(s.Completeness) }

// DuplicatesCI — как DuplicateCICount(s.Duplicates).
func (s Snapshot) DuplicatesCI() int { return DuplicateCICount(s.Duplicates) }

// Compute считает все показатели одним проходом (несколько запросов, но каждый — агрегат по
// действующим КЕ, не построчная выборка: на объёме из intent §9 это доли секунды).
func Compute(ctx context.Context) (Snapshot, error) {
	now := time.Now().UTC()
	s := Snapshot{Date: now.Format("2006-01-02"), ComputedAt: now}
	var err error
	if s.Completeness, err = Completeness(ctx); err != nil {
		return s, err
	}
	if s.Naming, s.NamingTotal, err = Naming(ctx); err != nil {
		return s, err
	}
	if s.Staleness, err = ComputeStaleness(ctx); err != nil {
		return s, err
	}
	if s.Duplicates, err = Duplicates(ctx); err != nil {
		return s, err
	}
	if s.Conflicts, err = OpenConflicts(ctx); err != nil {
		return s, err
	}
	return s, nil
}

func namingPct(naming []NamingBreakdown, total int, status string) float64 {
	for _, b := range naming {
		if b.Status == status {
			return b.Pct(total)
		}
	}
	return 0
}

// Save сохраняет снимок за его Date (UTC), перезаписывая прежний снимок того же дня (снимок дня,
// не журнал — повторный пересчёт в течение дня не должен копиться строками).
func Save(ctx context.Context, s Snapshot) error {
	attrsPct, _ := s.AttrsCompletePct()
	var attrsPctArg any
	if _, ok := s.AttrsCompletePct(); ok {
		attrsPctArg = attrsPct
	}
	confirmedPct, ok := s.Staleness.Confirmed7dPct()
	var confirmedArg any
	if ok {
		confirmedArg = confirmedPct
	}
	_, err := orm.NewOrm().RawWithCtx(ctx, `
		INSERT INTO rpt.quality_snapshot
			(snapshot_date, total_active_ci, attrs_complete_pct,
			 naming_compliant_pct, naming_legacy_pct, naming_noncompliant_pct, naming_exception_pct,
			 source_linked_ci, confirmed_7d_pct, duplicates_ci, open_conflicts, open_conflicts_over_14d, computed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (snapshot_date) DO UPDATE SET
			total_active_ci = EXCLUDED.total_active_ci, attrs_complete_pct = EXCLUDED.attrs_complete_pct,
			naming_compliant_pct = EXCLUDED.naming_compliant_pct, naming_legacy_pct = EXCLUDED.naming_legacy_pct,
			naming_noncompliant_pct = EXCLUDED.naming_noncompliant_pct, naming_exception_pct = EXCLUDED.naming_exception_pct,
			source_linked_ci = EXCLUDED.source_linked_ci, confirmed_7d_pct = EXCLUDED.confirmed_7d_pct,
			duplicates_ci = EXCLUDED.duplicates_ci, open_conflicts = EXCLUDED.open_conflicts,
			open_conflicts_over_14d = EXCLUDED.open_conflicts_over_14d, computed_at = EXCLUDED.computed_at
	`, s.Date, s.Staleness.TotalActive, attrsPctArg,
		namingPct(s.Naming, s.NamingTotal, "compliant"), namingPct(s.Naming, s.NamingTotal, "legacy"),
		namingPct(s.Naming, s.NamingTotal, "noncompliant"), namingPct(s.Naming, s.NamingTotal, "exception"),
		s.Staleness.SourceLinked, confirmedArg, s.DuplicatesCI(), s.Conflicts.Open, s.Conflicts.OpenOver14d, s.ComputedAt).Exec()
	if err != nil {
		return fmt.Errorf("quality: запись снимка: %w", err)
	}
	return nil
}

// TrendPoint — одна строка снимка (rpt.quality_snapshot) для графика динамики.
type TrendPoint struct {
	Date                  string
	TotalActiveCI         int
	AttrsCompletePct      *float64
	NamingCompliantPct    float64
	NamingLegacyPct       float64
	NamingNoncompliantPct float64
	NamingExceptionPct    float64
	ConfirmedPct          *float64
	DuplicatesCI          int
	OpenConflicts         int
	OpenConflictsOver14d  int
}

// pctOrDash форматирует указатель на процент для шаблона (html/template не умеет разыменовывать
// указатель сам, а печать *float64 через fmt дала бы адрес, не число); nil — «н/д»: KPI не считался
// (например, ни одна КЕ ещё не была подтверждена источником).
func pctOrDash(p *float64) string {
	if p == nil {
		return "н/д"
	}
	return strconv.FormatFloat(*p, 'f', 1, 64) + "%"
}

// AttrsPctDisplay — AttrsCompletePct для шаблона (views/quality/index.tpl): число со знаком % либо «н/д».
func (p TrendPoint) AttrsPctDisplay() string { return pctOrDash(p.AttrsCompletePct) }

// ConfirmedPctDisplay — ConfirmedPct для шаблона, аналогично AttrsPctDisplay.
func (p TrendPoint) ConfirmedPctDisplay() string { return pctOrDash(p.ConfirmedPct) }

// History — последние days снимков, от новых к старым (для графика динамики E7.4); days<=0 — все.
func History(ctx context.Context, days int) ([]TrendPoint, error) {
	limit := ""
	if days > 0 {
		limit = fmt.Sprintf(" LIMIT %d", days)
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT to_char(snapshot_date, 'YYYY-MM-DD') AS date, total_active_ci::text AS total,
		       COALESCE(attrs_complete_pct::text, '') AS attrs_pct,
		       naming_compliant_pct::text AS nc, naming_legacy_pct::text AS nl, naming_noncompliant_pct::text AS nn, naming_exception_pct::text AS ne,
		       COALESCE(confirmed_7d_pct::text, '') AS confirmed_pct,
		       duplicates_ci::text AS dup, open_conflicts::text AS oc, open_conflicts_over_14d::text AS oc14
		FROM rpt.quality_snapshot ORDER BY snapshot_date DESC`+limit)
	if err != nil {
		return nil, fmt.Errorf("quality: история снимков: %w", err)
	}
	out := make([]TrendPoint, len(rows))
	for i, r := range rows {
		p := TrendPoint{Date: r["date"]}
		fmt.Sscan(r["total"], &p.TotalActiveCI)
		fmt.Sscan(r["nc"], &p.NamingCompliantPct)
		fmt.Sscan(r["nl"], &p.NamingLegacyPct)
		fmt.Sscan(r["nn"], &p.NamingNoncompliantPct)
		fmt.Sscan(r["ne"], &p.NamingExceptionPct)
		fmt.Sscan(r["dup"], &p.DuplicatesCI)
		fmt.Sscan(r["oc"], &p.OpenConflicts)
		fmt.Sscan(r["oc14"], &p.OpenConflictsOver14d)
		if r["attrs_pct"] != "" {
			var v float64
			fmt.Sscan(r["attrs_pct"], &v)
			p.AttrsCompletePct = &v
		}
		if r["confirmed_pct"] != "" {
			var v float64
			fmt.Sscan(r["confirmed_pct"], &v)
			p.ConfirmedPct = &v
		}
		out[i] = p
	}
	return out, nil
}
