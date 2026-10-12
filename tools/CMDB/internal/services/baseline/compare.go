package baseline

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/svcerr"
)

// State — одна из сторон сравнения (plan.md E6.3): именованный срез, момент в прошлом («дата») или
// текущее состояние. Задаётся ровно одним способом.
type State struct {
	BaselineID int64
	AsOf       *time.Time
	Current    bool
}

// ParseState разбирает «b:12» (срез), «t:2026-09-01T10:00:00Z» / «t:2026-09-01» (момент), «now» (текущее).
func ParseState(s string) (State, error) {
	s = strings.TrimSpace(s)
	switch {
	case s == "now":
		return State{Current: true}, nil
	case strings.HasPrefix(s, "b:"):
		id, err := strconv.ParseInt(s[2:], 10, 64)
		if err != nil || id <= 0 {
			return State{}, fmt.Errorf("%w: срез %q", svcerr.ErrInvalidInput, s)
		}
		return State{BaselineID: id}, nil
	case strings.HasPrefix(s, "t:"):
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04", "2006-01-02"} {
			if t, err := time.ParseInLocation(layout, s[2:], time.UTC); err == nil {
				return State{AsOf: &t}, nil
			}
		}
	}
	return State{}, fmt.Errorf("%w: сторона сравнения %q (ожидается b:<id>, t:<момент> или now)", svcerr.ErrInvalidInput, s)
}

// FieldDiff — значение поля на сторонах A и B.
type FieldDiff struct{ Name, A, B string }

// Change — одно различие: добавлено (есть только в B), удалено (только в A) или изменено (версия
// строки на сторонах различается по содержимому).
type Change struct {
	Entity   string
	EntityID int64
	CIID     int64
	CIName   string
	Kind     string // added | removed | changed
	AVersion int
	BVersion int
	Fields   []FieldDiff // только у changed
}

// SummaryRow — счётчики различий по сущности.
type SummaryRow struct {
	Entity                  string
	Added, Removed, Changed int
}

// Result — итог сравнения.
type Result struct {
	LabelA, LabelB          string
	Summary                 []SummaryRow
	Added, Removed, Changed int
	Changes                 []Change
	Truncated               bool // подробности показаны не для всех различий (см. MaxDetails)
}

// MaxDetails — предел подробных различий в ответе (счётчики всегда полные): атрибутный дифф каждой
// изменённой строки — отдельное чтение версий.
const MaxDetails = 5000

// noiseFields — служебные колонки, всегда отличающиеся между версиями: в атрибутном диффе шумят.
var noiseFields = map[string]bool{"uid": true, "row_version": true, "change_set_id": true, "created_at": true, "created_by": true,
	"updated_at": true, "updated_by": true, "sys_period": true}

// Label возвращает подпись стороны для отчёта.
func (s State) Label(ctx context.Context) string {
	switch {
	case s.Current:
		return "Текущее состояние"
	case s.AsOf != nil:
		return "Состояние на " + s.AsOf.UTC().Format("2006-01-02 15:04:05") + " UTC"
	}
	if b, err := Get(ctx, s.BaselineID); err == nil {
		return fmt.Sprintf("Срез №%s «%s» (на %s UTC)", b.ID, b.Name, b.AsOf)
	}
	return fmt.Sprintf("Срез №%d", s.BaselineID)
}

// Compare сравнивает две стороны: срез↔срез, срез↔текущее (или дата), дата↔дата (E6.3). Область для
// сторон-моментов: если другая сторона — срез, берётся область этого среза (иначе состав нельзя было бы
// сопоставить), при сравнении дата↔дата — scope (пусто — все КЕ).
func Compare(ctx context.Context, a, b State, scope Scope) (Result, error) {
	var res Result
	res.LabelA, res.LabelB = a.Label(ctx), b.Label(ctx)

	// область для сторон-моментов
	sideScope := func(other State) (Scope, error) {
		if other.BaselineID != 0 {
			bl, err := Get(ctx, other.BaselineID)
			if err != nil {
				return Scope{}, err
			}
			return bl.Scope(), nil
		}
		return scope, nil
	}
	scopeA, err := sideScope(b)
	if err != nil {
		return res, err
	}
	scopeB, err := sideScope(a)
	if err != nil {
		return res, err
	}

	var rows []repo.Row
	err = orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		if err := loadSide(ctx, tx, "_cmp_a", a, scopeA); err != nil {
			return err
		}
		if err := loadSide(ctx, tx, "_cmp_b", b, scopeB); err != nil {
			return err
		}
		var raw []orm.Params
		if _, err := tx.RawWithCtx(ctx, `
			SELECT COALESCE(a.entity, b.entity) AS entity, COALESCE(a.entity_id, b.entity_id) AS entity_id,
			       COALESCE(b.ci_id, a.ci_id) AS ci_id,
			       CASE WHEN a.entity IS NULL THEN 'added' WHEN b.entity IS NULL THEN 'removed' ELSE 'changed' END AS kind,
			       COALESCE(a.row_version, 0) AS av, COALESCE(b.row_version, 0) AS bv
			FROM _cmp_a a FULL JOIN _cmp_b b ON a.entity = b.entity AND a.entity_id = b.entity_id
			WHERE a.entity IS NULL OR b.entity IS NULL OR a.row_hash <> b.row_hash
			ORDER BY COALESCE(a.entity, b.entity), COALESCE(a.entity_id, b.entity_id)`).Values(&raw); err != nil {
			return fmt.Errorf("baseline: сравнение: %w", err)
		}
		for _, p := range raw {
			r := repo.Row{}
			for k, v := range p {
				if v != nil {
					r[k] = fmt.Sprint(v)
				}
			}
			rows = append(rows, r)
		}
		return nil
	})
	if err != nil {
		return res, err
	}

	byEntity := map[string]*SummaryRow{}
	ciIDs := map[int64]bool{}
	for _, r := range rows {
		sr := byEntity[r["entity"]]
		if sr == nil {
			sr = &SummaryRow{Entity: r["entity"]}
			byEntity[r["entity"]] = sr
		}
		switch r["kind"] {
		case "added":
			sr.Added++
			res.Added++
		case "removed":
			sr.Removed++
			res.Removed++
		default:
			sr.Changed++
			res.Changed++
		}
	}
	for _, e := range Entities() {
		if sr := byEntity[e]; sr != nil {
			res.Summary = append(res.Summary, *sr)
		}
	}

	limit := len(rows)
	if limit > MaxDetails {
		limit, res.Truncated = MaxDetails, true
	}
	for _, r := range rows[:limit] {
		ch := Change{Entity: r["entity"], Kind: r["kind"]}
		ch.EntityID, _ = strconv.ParseInt(r["entity_id"], 10, 64)
		ch.CIID, _ = strconv.ParseInt(r["ci_id"], 10, 64)
		ch.AVersion, _ = strconv.Atoi(r["av"])
		ch.BVersion, _ = strconv.Atoi(r["bv"])
		if ch.CIID != 0 {
			ciIDs[ch.CIID] = true
		}
		if ch.Kind == "changed" {
			da, err := VersionData(ctx, ch.Entity, ch.EntityID, ch.AVersion)
			if err != nil {
				return res, err
			}
			db, err := VersionData(ctx, ch.Entity, ch.EntityID, ch.BVersion)
			if err != nil {
				return res, err
			}
			ch.Fields = diffFields(da, db)
		}
		res.Changes = append(res.Changes, ch)
	}

	names, err := ciNames(ctx, ciIDs)
	if err != nil {
		return res, err
	}
	for i := range res.Changes {
		res.Changes[i].CIName = names[res.Changes[i].CIID]
	}
	return res, nil
}

// loadSide заполняет временную таблицу name позициями стороны (entity, entity_id, row_version, ci_id, row_hash).
func loadSide(ctx context.Context, tx orm.TxOrmer, name string, s State, sc Scope) error {
	if _, err := tx.RawWithCtx(ctx, `CREATE TEMP TABLE `+name+` (entity text, entity_id bigint, row_version int, ci_id bigint, row_hash text,
		PRIMARY KEY (entity, entity_id)) ON COMMIT DROP`).Exec(); err != nil {
		return fmt.Errorf("baseline: %s: %w", name, err)
	}
	if s.BaselineID != 0 {
		res, err := tx.RawWithCtx(ctx, `INSERT INTO `+name+` SELECT entity, entity_id, row_version, ci_id, row_hash FROM rpt.baseline_item WHERE baseline_id = ?`, s.BaselineID).Exec()
		if err != nil {
			return fmt.Errorf("baseline: чтение среза %d: %w", s.BaselineID, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			if _, err := Get(ctx, s.BaselineID); err != nil {
				return err
			}
		}
		return nil
	}
	ts := time.Now()
	if s.AsOf != nil {
		ts = *s.AsOf
	}
	scopeTable := name + "_scope"
	if err := BuildScope(ctx, tx, scopeTable, ts, sc); err != nil {
		return err
	}
	for _, entity := range Entities() {
		q, err := VersionsSQL(entity, scopeTable)
		if err != nil {
			return err
		}
		if _, err := tx.RawWithCtx(ctx, `INSERT INTO `+name+` SELECT CAST(? AS text), entity_id, row_version, ci_id, row_hash FROM (`+q+`) v`,
			entity, repo.TimeArg(ts), repo.TimeArg(ts)).Exec(); err != nil {
			return fmt.Errorf("baseline: состояние %s: %w", entity, err)
		}
	}
	return nil
}

func diffFields(a, b map[string]any) []FieldDiff {
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	names := make([]string, 0, len(keys))
	for k := range keys {
		if !noiseFields[k] {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	var out []FieldDiff
	for _, k := range names {
		av, bv := fieldText(a[k]), fieldText(b[k])
		if av != bv {
			out = append(out, FieldDiff{Name: k, A: av, B: bv})
		}
	}
	return out
}

func fieldText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}

func ciNames(ctx context.Context, ids map[int64]bool) (map[int64]string, error) {
	out := map[int64]string{}
	list := make([]int64, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	for start := 0; start < len(list); start += 500 {
		chunk := list[start:min(start+500, len(list))]
		ph, args := make([]string, len(chunk)), make([]any, len(chunk))
		for i, id := range chunk {
			ph[i], args[i] = "?", id
		}
		rows, err := repo.QueryRows(ctx, `SELECT id::text AS id, name FROM cmdb.ci WHERE id IN (`+strings.Join(ph, ",")+`)`, args...)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			id, _ := strconv.ParseInt(r["id"], 10, 64)
			out[id] = r["name"]
		}
	}
	return out, nil
}
