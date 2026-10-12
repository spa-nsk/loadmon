// Сравнение двух версий КЕ и глобальный журнал изменений (plan.md E3.7).
package history

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cmdb/internal/repo"
)

// noiseColumns — служебные колонки cmdb.ci, не несущие содержательной информации в сравнении
// версий (суррогатный/версионный учёт, не бизнес-атрибут): всегда отличаются между версиями и
// только шумят в диффе.
var noiseColumns = map[string]bool{"uid": true, "sys_period": true, "row_version": true, "change_set_id": true, "created_at": true, "created_by": true}

// CompareField — значение одной колонки cmdb.ci на два выбранных момента (по change set).
type CompareField struct {
	Name    string
	A       string
	B       string
	Changed bool
}

// Compare сравнивает состояние КЕ сразу после применения change set csA и сразу после csB
// (cmdb.ci_asof на момент committed_at/started_at каждого набора, E2.10) — "сравнение двух любых
// версий КЕ". Возвращает только неслужебные колонки, сначала изменившиеся.
func Compare(ctx context.Context, ciID, csA, csB int64) ([]CompareField, error) {
	rowA, err := asOfChangeSet(ctx, ciID, csA)
	if err != nil {
		return nil, fmt.Errorf("history: версия A (change_set %d): %w", csA, err)
	}
	rowB, err := asOfChangeSet(ctx, ciID, csB)
	if err != nil {
		return nil, fmt.Errorf("history: версия B (change_set %d): %w", csB, err)
	}

	names := make([]string, 0, len(rowA))
	for k := range rowA {
		if !noiseColumns[k] {
			names = append(names, k)
		}
	}
	sort.Strings(names)

	var changed, unchanged []CompareField
	for _, name := range names {
		f := CompareField{Name: name, A: rowA[name], B: rowB[name]}
		f.Changed = f.A != f.B
		if f.Changed {
			changed = append(changed, f)
		} else {
			unchanged = append(unchanged, f)
		}
	}
	return append(changed, unchanged...), nil
}

func asOfChangeSet(ctx context.Context, ciID, csID int64) (repo.Row, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT a.* FROM audit.change_set cs
		CROSS JOIN LATERAL cmdb.ci_asof(COALESCE(cs.committed_at, cs.started_at)) a
		WHERE cs.id = ? AND a.id = ?
	`, csID, ciID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("нет версии КЕ id=%d на момент change_set %d", ciID, csID)
	}
	return row, nil
}

// GlobalFilter — фильтры глобального журнала изменений (E3.7).
type GlobalFilter struct {
	Source  string // audit.change_set.source_kind, "" = любой
	Entity  string // audit.change_log.entity, "" = любая
	Q       string // подстрока в reason
	From    string // дата ГГГГ-ММ-ДД, включительно
	To      string // дата ГГГГ-ММ-ДД, включительно
	Page    int
	PerPage int
}

// GlobalRow — одна строка глобального журнала: набор изменений + сводка (сколько записей и какие
// сущности затронуты), без разворачивания атрибутов (это — GlobalEntries по запросу).
type GlobalRow struct {
	ID       string
	At       string
	Source   string
	UserID   string
	Reason   string
	RFC      string
	Count    string
	Entities string
}

// GlobalResult — страница глобального журнала.
type GlobalResult struct {
	Rows    []GlobalRow
	Total   int
	Page    int
	PerPage int
	Pages   int
}

// GlobalLog возвращает страницу change set'ов по фильтру, от новых к старым.
func GlobalLog(ctx context.Context, f GlobalFilter) (GlobalResult, error) {
	var conds []string
	var args []any
	add := func(cond string, a ...any) {
		conds = append(conds, cond)
		args = append(args, a...)
	}
	if f.Source != "" {
		add("cs.source_kind = ?", f.Source)
	}
	if f.Entity != "" {
		add("EXISTS (SELECT 1 FROM audit.change_log l WHERE l.change_set_id = cs.id AND l.entity = ?)", f.Entity)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		add("cs.reason ILIKE ?", "%"+q+"%")
	}
	if f.From != "" {
		add("cs.started_at >= ?::date", f.From)
	}
	if f.To != "" {
		add("cs.started_at < (?::date + 1)", f.To)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	if f.PerPage <= 0 {
		f.PerPage = 25
	}
	if f.Page < 1 {
		f.Page = 1
	}

	res := GlobalResult{Page: f.Page, PerPage: f.PerPage}
	total, err := repo.QueryRow(ctx, `SELECT count(*)::text AS n FROM audit.change_set cs `+where, args...)
	if err != nil {
		return res, err
	}
	fmt.Sscan(total["n"], &res.Total)
	res.Pages = (res.Total + f.PerPage - 1) / f.PerPage
	if res.Pages == 0 {
		res.Pages = 1
	}

	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	rows, err := repo.QueryRows(ctx, `
		SELECT cs.id::text AS id,
		       to_char(COALESCE(cs.committed_at, cs.started_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS at,
		       cs.source_kind, COALESCE(cs.user_id::text, '') AS user_id, COALESCE(cs.reason, '') AS reason,
		       COALESCE(cs.change_request_ref, '') AS rfc,
		       (SELECT count(*) FROM audit.change_log l WHERE l.change_set_id = cs.id)::text AS n,
		       COALESCE((SELECT string_agg(DISTINCT entity, ', ' ORDER BY entity) FROM audit.change_log l WHERE l.change_set_id = cs.id), '') AS entities
		FROM audit.change_set cs `+where+`
		ORDER BY cs.id DESC LIMIT ? OFFSET ?`, pageArgs...)
	if err != nil {
		return res, err
	}
	res.Rows = make([]GlobalRow, len(rows))
	for i, r := range rows {
		res.Rows[i] = GlobalRow{ID: r["id"], At: r["at"], Source: r["source_kind"], UserID: r["user_id"],
			Reason: r["reason"], RFC: r["rfc"], Count: r["n"], Entities: r["entities"]}
	}
	return res, nil
}

// DistinctEntities возвращает встречающиеся значения audit.change_log.entity для фильтра журнала.
func DistinctEntities(ctx context.Context) ([]string, error) {
	rows, err := repo.QueryRows(ctx, `SELECT DISTINCT entity AS v FROM audit.change_log ORDER BY entity`)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r["v"]
	}
	return out, nil
}

// ForChangeSet возвращает один набор изменений целиком (все записи, не только относящиеся к
// какой-то КЕ) — раскрытие строки глобального журнала.
func ForChangeSet(ctx context.Context, csID int64) (ChangeSet, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT id::text AS id, to_char(COALESCE(committed_at, started_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS at,
		       source_kind, COALESCE(user_id::text, '') AS user_id, COALESCE(reason, '') AS reason, COALESCE(change_request_ref, '') AS rfc
		FROM audit.change_set WHERE id = ?`, csID)
	if err != nil {
		return ChangeSet{}, err
	}
	if row == nil {
		return ChangeSet{}, fmt.Errorf("набор изменений %d не найден", csID)
	}
	cs := ChangeSet{ID: row["id"], At: row["at"], Source: row["source_kind"], UserID: row["user_id"], Reason: row["reason"], RFC: row["rfc"]}

	rows, err := repo.QueryRows(ctx, `
		SELECT entity, op, COALESCE(attr_name, '') AS attr, COALESCE(old_value::text, '') AS old_value, COALESCE(new_value::text, '') AS new_value
		FROM audit.change_log WHERE change_set_id = ? ORDER BY id`, csID)
	if err != nil {
		return ChangeSet{}, err
	}
	cs.Entries = make([]Entry, len(rows))
	for i, r := range rows {
		cs.Entries[i] = Entry{Entity: r["entity"], Op: r["op"], Attr: r["attr"], Old: r["old_value"], New: r["new_value"]}
	}
	return cs, nil
}
