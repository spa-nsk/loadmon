// Package baseline — именованные срезы (plan.md E6.2, intent §7.5 п.2). Baseline хранит ссылки на
// версии строк (сущность, id, row_version) и SHA-256 содержимого каждой версии, а не копию данных;
// состав определяется «машиной времени» (sys_period @> as_of_ts) и фильтром области. Жизненный цикл:
// draft -> frozen (вычисляется checksum, дальше неизменяем — гарантируют триггеры БД) -> approved.
package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/auditlog"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/svcerr"
)

// Scope — фильтр области среза; пусто — все КЕ на момент среза. Область определяется по состоянию КЕ
// на as_of_ts; в срез входят сами КЕ, их таблицы-расширения, компоненты и связи (связь — если хотя бы
// один её конец в области).
type Scope struct {
	Classes         []string `json:"classes,omitempty"`
	LocationCode    string   `json:"location_code,omitempty"`
	CompanyCode     string   `json:"company_code,omitempty"`
	EnvironmentCode string   `json:"environment_code,omitempty"`
	NameLike        string   `json:"name_like,omitempty"`
}

// Baseline — строка среза.
type Baseline struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AsOf        string `json:"as_of"`
	Status      string `json:"status"`
	Checksum    string `json:"checksum"`
	ScopeJSON   string `json:"scope_json"`
	ItemCount   string `json:"item_count"`
	CreatedAt   string `json:"created_at"`
	CreatedBy   string `json:"created_by"`
	FrozenAt    string `json:"frozen_at"`
	FrozenBy    string `json:"frozen_by"`
	ApprovedAt  string `json:"approved_at"`
	ApprovedBy  string `json:"approved_by"`
}

// Entities — сущности, входящие в срез, в стабильном порядке: ci, component, ci_rel и таблицы-расширения.
func Entities() []string {
	ext := make([]string, 0, len(ci.ExtensionTables))
	for t := range ci.ExtensionTables {
		ext = append(ext, t)
	}
	sort.Strings(ext)
	return append([]string{"ci", "component", "ci_rel"}, ext...)
}

// entityInfo — как читать версии сущности: имя первичного ключа и выражение КЕ-владельца.
func entityInfo(entity string) (pk, ciExpr, scopeCond string, ok bool) {
	switch {
	case entity == "ci":
		return "id", "id", "id IN (SELECT id FROM %s)", true
	case entity == "component":
		return "id", "ci_id", "ci_id IN (SELECT id FROM %s)", true
	case entity == "ci_rel":
		return "id", "src_ci_id", "(src_ci_id IN (SELECT id FROM %[1]s) OR dst_ci_id IN (SELECT id FROM %[1]s))", true
	case ci.ExtensionTables[entity]:
		return "ci_id", "ci_id", "ci_id IN (SELECT id FROM %s)", true
	}
	return "", "", "", false
}

const histCols = ` - 'sys_period' - 'hist_id' - 'hist_op' - 'hist_recorded_at'`

// hashExpr — SHA-256 содержимого версии строки: JSON без sys_period (граница периода меняется, когда
// строка уходит в hist, а содержимое версии — нет) и без служебных колонок hist.
const hashExpr = `encode(sha256(convert_to(j::text, 'UTF8')), 'hex')`

// VersionsSQL возвращает запрос версий сущности, действовавших в момент ts (плейсхолдер — один, момент
// времени), ограниченных областью — временной таблицей scopeTable (id КЕ). Колонки: entity_id,
// row_version, ci_id, row_hash. scopeTable подставляется в SQL: только внутренние имена (_bl_scope…).
func VersionsSQL(entity, scopeTable string) (string, error) {
	pk, ciExpr, cond, ok := entityInfo(entity)
	if !ok {
		return "", fmt.Errorf("baseline: неизвестная сущность %q", entity)
	}
	cond = fmt.Sprintf(cond, scopeTable)
	base, hist := "cmdb."+entity, "hist."+entity
	return fmt.Sprintf(`
		SELECT pk_ AS entity_id, row_version, ci_ AS ci_id, %[3]s AS row_hash FROM (
			SELECT %[1]s AS pk_, row_version, %[2]s AS ci_, to_jsonb(t) - 'sys_period' AS j FROM %[4]s t
			WHERE sys_period @> CAST(CAST(? AS text) AS timestamptz) AND %[6]s
			UNION ALL
			SELECT %[1]s AS pk_, row_version, %[2]s AS ci_, to_jsonb(h)%[7]s AS j FROM %[5]s h
			WHERE sys_period @> CAST(CAST(? AS text) AS timestamptz) AND %[6]s
		) x`, pk, ciExpr, hashExpr, base, hist, cond, histCols), nil
}

// BuildScope создаёт временную таблицу name с id КЕ области на момент ts (в переданной транзакции).
func BuildScope(ctx context.Context, tx orm.TxOrmer, name string, ts time.Time, sc Scope) error {
	conds, args := []string{"TRUE"}, []any{repo.TimeArg(ts)}
	if len(sc.Classes) > 0 {
		ph := make([]string, len(sc.Classes))
		for i, c := range sc.Classes {
			ph[i] = "?"
			args = append(args, c)
		}
		conds = append(conds, "c.ci_class IN ("+strings.Join(ph, ", ")+")")
	}
	add := func(col, v string) {
		if v != "" {
			conds = append(conds, "c."+col+" = ?")
			args = append(args, v)
		}
	}
	add("location_code", sc.LocationCode)
	add("company_code", sc.CompanyCode)
	add("environment_code", sc.EnvironmentCode)
	if sc.NameLike != "" {
		conds = append(conds, "c.name ILIKE ?")
		args = append(args, "%"+sc.NameLike+"%")
	}
	_, err := tx.RawWithCtx(ctx, `CREATE TEMP TABLE `+name+` ON COMMIT DROP AS
		SELECT c.id FROM cmdb.ci_asof(CAST(CAST(? AS text) AS timestamptz)) c WHERE `+strings.Join(conds, " AND "), args...).Exec()
	if err != nil {
		return fmt.Errorf("baseline: область среза: %w", err)
	}
	return nil
}

// Create создаёт срез-черновик на момент asOf по фильтру области и заполняет его позициями.
func Create(ctx context.Context, name, description string, asOf time.Time, sc Scope, userID int64) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, fmt.Errorf("%w: не задано имя среза", svcerr.ErrInvalidInput)
	}
	if asOf.After(time.Now()) {
		return 0, fmt.Errorf("%w: момент среза в будущем", svcerr.ErrInvalidInput)
	}
	scopeJSON, _ := json.Marshal(sc)

	var id int64
	err := orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		if err := tx.RawWithCtx(ctx, `
			INSERT INTO rpt.baseline (name, description, as_of_ts, scope_filter, created_by)
			VALUES (?, nullif(?, ''), CAST(CAST(? AS text) AS timestamptz), CAST(? AS jsonb), ?) RETURNING id
		`, name, description, repo.TimeArg(asOf), string(scopeJSON), userID).QueryRow(&id); err != nil {
			if svcerr.ViolatesConstraint(err, "baseline_name_uniq") {
				return fmt.Errorf("%w: срез с именем %q", svcerr.ErrDuplicate, name)
			}
			return fmt.Errorf("baseline: создание: %w", err)
		}
		if err := BuildScope(ctx, tx, "_bl_scope", asOf, sc); err != nil {
			return err
		}
		total := 0
		for _, entity := range Entities() {
			q, err := VersionsSQL(entity, "_bl_scope")
			if err != nil {
				return err
			}
			res, err := tx.RawWithCtx(ctx, `
				INSERT INTO rpt.baseline_item (baseline_id, entity, entity_id, row_version, ci_id, row_hash)
				SELECT ?, CAST(? AS text), entity_id, row_version, ci_id, row_hash FROM (`+q+`) v`,
				id, entity, repo.TimeArg(asOf), repo.TimeArg(asOf)).Exec()
			if err != nil {
				return fmt.Errorf("baseline: позиции %s: %w", entity, err)
			}
			n, _ := res.RowsAffected()
			total += int(n)
		}
		_, err := tx.RawWithCtx(ctx, `UPDATE rpt.baseline SET item_count = ? WHERE id = ?`, total, id).Exec()
		return err
	})
	if err != nil {
		return 0, err
	}
	_ = auditlog.Log(ctx, userID, "", "baseline_create", map[string]any{"baseline_id": id, "name": name, "as_of": asOf.UTC().Format(time.RFC3339)})
	return id, nil
}

// checksumSQL — SHA-256 по отсортированным позициям среза (сущность, id, версия, хеш версии).
const checksumSQL = `SELECT encode(sha256(convert_to(COALESCE(string_agg(entity || ':' || entity_id || ':' || row_version || ':' || row_hash, E'\n'
	ORDER BY entity, entity_id), ''), 'UTF8')), 'hex') AS sum FROM rpt.baseline_item WHERE baseline_id = ?`

// Freeze замораживает черновик: вычисляет контрольную сумму; дальше срез неизменяем (триггеры БД).
func Freeze(ctx context.Context, id, userID int64) error {
	err := orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		var status string
		if err := tx.RawWithCtx(ctx, `SELECT status FROM rpt.baseline WHERE id = ? FOR UPDATE`, id).QueryRow(&status); err != nil {
			if err == orm.ErrNoRows {
				return svcerr.NotFound("baseline", id)
			}
			return err
		}
		if status != "draft" {
			return fmt.Errorf("%w: срез уже %s", svcerr.ErrInvalidState, status)
		}
		var sum string
		if err := tx.RawWithCtx(ctx, checksumSQL, id).QueryRow(&sum); err != nil {
			return fmt.Errorf("baseline: контрольная сумма: %w", err)
		}
		_, err := tx.RawWithCtx(ctx, `UPDATE rpt.baseline SET status = 'frozen', checksum = ?, frozen_at = now(), frozen_by = ? WHERE id = ?`, sum, userID, id).Exec()
		return err
	})
	if err != nil {
		return err
	}
	return auditlog.Log(ctx, userID, "", "baseline_freeze", map[string]any{"baseline_id": id})
}

// Approve утверждает замороженный срез.
func Approve(ctx context.Context, id, userID int64) error {
	res, err := orm.NewOrm().RawWithCtx(ctx, `UPDATE rpt.baseline SET status = 'approved', approved_at = now(), approved_by = ? WHERE id = ? AND status = 'frozen'`, userID, id).Exec()
	if err != nil {
		return fmt.Errorf("baseline: утверждение: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: утвердить можно только замороженный срез", svcerr.ErrInvalidState)
	}
	return auditlog.Log(ctx, userID, "", "baseline_approve", map[string]any{"baseline_id": id})
}

// Delete удаляет черновик; замороженные и утверждённые не удаляются.
func Delete(ctx context.Context, id, userID int64) error {
	res, err := orm.NewOrm().RawWithCtx(ctx, `DELETE FROM rpt.baseline WHERE id = ? AND status = 'draft'`, id).Exec()
	if err != nil {
		return fmt.Errorf("baseline: удаление: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: удалить можно только черновик", svcerr.ErrInvalidState)
	}
	return auditlog.Log(ctx, userID, "", "baseline_delete", map[string]any{"baseline_id": id})
}

const selectCols = `id::text AS id, name, COALESCE(description, '') AS description,
	to_char(as_of_ts AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS as_of, scope_filter::text AS scope, status, item_count::text AS item_count,
	COALESCE(checksum, '') AS checksum, to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at, COALESCE(created_by::text, '') AS created_by,
	COALESCE(to_char(frozen_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS frozen_at, COALESCE(frozen_by::text, '') AS frozen_by,
	COALESCE(to_char(approved_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS approved_at, COALESCE(approved_by::text, '') AS approved_by`

func fromRow(r repo.Row) Baseline {
	return Baseline{ID: r["id"], Name: r["name"], Description: r["description"], AsOf: r["as_of"], ScopeJSON: r["scope"], Status: r["status"],
		ItemCount: r["item_count"], Checksum: r["checksum"], CreatedAt: r["created_at"], CreatedBy: r["created_by"], FrozenAt: r["frozen_at"],
		FrozenBy: r["frozen_by"], ApprovedAt: r["approved_at"], ApprovedBy: r["approved_by"]}
}

// List возвращает срезы от новых к старым.
func List(ctx context.Context) ([]Baseline, error) {
	rows, err := repo.QueryRows(ctx, `SELECT `+selectCols+` FROM rpt.baseline ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("baseline: список: %w", err)
	}
	out := make([]Baseline, len(rows))
	for i, r := range rows {
		out[i] = fromRow(r)
	}
	return out, nil
}

// Get возвращает срез или svcerr.ErrNotFound.
func Get(ctx context.Context, id int64) (Baseline, error) {
	row, err := repo.QueryRow(ctx, `SELECT `+selectCols+` FROM rpt.baseline WHERE id = ?`, id)
	if err != nil {
		return Baseline{}, fmt.Errorf("baseline: чтение: %w", err)
	}
	if row == nil {
		return Baseline{}, svcerr.NotFound("baseline", id)
	}
	return fromRow(row), nil
}

// AsTime возвращает момент среза.
func (b Baseline) AsTime() (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", b.AsOf, time.UTC)
}

// Scope разбирает фильтр области среза.
func (b Baseline) Scope() Scope {
	var s Scope
	_ = json.Unmarshal([]byte(b.ScopeJSON), &s)
	return s
}

// Count — число позиций среза по сущностям.
func Count(ctx context.Context, id int64) (map[string]int, error) {
	rows, err := repo.QueryRows(ctx, `SELECT entity, count(*)::text AS n FROM rpt.baseline_item WHERE baseline_id = ? GROUP BY entity`, id)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		n, _ := strconv.Atoi(r["n"])
		out[r["entity"]] = n
	}
	return out, nil
}

// VerifyResult — итог проверки целостности замороженного среза.
type VerifyResult struct {
	OK       bool
	Problems []string
}

// Verify перепроверяет срез: контрольная сумма позиций совпадает с сохранённой, а каждая версия строки
// (в cmdb.* или hist.*) существует и даёт тот же хеш содержимого — обнаруживает правку истории мимо приложения.
func Verify(ctx context.Context, id int64) (VerifyResult, error) {
	b, err := Get(ctx, id)
	if err != nil {
		return VerifyResult{}, err
	}
	var res VerifyResult
	if b.Status == "draft" {
		return res, fmt.Errorf("%w: черновик не имеет контрольной суммы", svcerr.ErrInvalidState)
	}
	row, err := repo.QueryRow(ctx, checksumSQL, id)
	if err != nil {
		return res, err
	}
	if sum := row["sum"]; sum != b.Checksum {
		res.Problems = append(res.Problems, fmt.Sprintf("контрольная сумма позиций %s не совпадает с сохранённой %s", sum, b.Checksum))
	}
	for _, entity := range Entities() {
		pk, _, _, _ := entityInfo(entity)
		q := fmt.Sprintf(`
			SELECT count(*) FILTER (WHERE x.h IS NULL)::text AS missing, count(*) FILTER (WHERE x.h IS NOT NULL AND x.h <> i.row_hash)::text AS mismatch
			FROM rpt.baseline_item i
			LEFT JOIN LATERAL (
				SELECT encode(sha256(convert_to(j::text, 'UTF8')), 'hex') AS h FROM (
					SELECT to_jsonb(t) - 'sys_period' AS j FROM cmdb.%[1]s t WHERE t.%[2]s = i.entity_id AND t.row_version = i.row_version
					UNION ALL
					SELECT to_jsonb(h)%[3]s AS j FROM hist.%[1]s h WHERE h.%[2]s = i.entity_id AND h.row_version = i.row_version
				) v LIMIT 1) x ON true
			WHERE i.baseline_id = ? AND i.entity = ?`, entity, pk, histCols)
		r, err := repo.QueryRow(ctx, q, id, entity)
		if err != nil {
			return res, fmt.Errorf("baseline: проверка %s: %w", entity, err)
		}
		if r["missing"] != "0" || r["mismatch"] != "0" {
			res.Problems = append(res.Problems, fmt.Sprintf("%s: версий не найдено %s, содержимое изменено %s", entity, r["missing"], r["mismatch"]))
		}
	}
	res.OK = len(res.Problems) == 0
	return res, nil
}

// VersionData возвращает содержимое версии строки (JSON без sys_period) из cmdb.* или hist.*;
// nil — версия не найдена.
func VersionData(ctx context.Context, entity string, entityID int64, version int) (map[string]any, error) {
	pk, _, _, ok := entityInfo(entity)
	if !ok {
		return nil, fmt.Errorf("baseline: неизвестная сущность %q", entity)
	}
	row, err := repo.QueryRow(ctx, fmt.Sprintf(`
		SELECT j::text AS j FROM (
			SELECT to_jsonb(t) - 'sys_period' AS j FROM cmdb.%[1]s t WHERE t.%[2]s = ? AND t.row_version = ?
			UNION ALL
			SELECT to_jsonb(h)%[3]s FROM hist.%[1]s h WHERE h.%[2]s = ? AND h.row_version = ?
		) v LIMIT 1`, entity, pk, histCols), entityID, version, entityID, version)
	if err != nil || row == nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(row["j"]), &m); err != nil {
		return nil, err
	}
	return m, nil
}
