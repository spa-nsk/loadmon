package ci

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/platform/asof"
	"cmdb/internal/repo"
	"cmdb/internal/services/svcerr"
)

// Значения ListFilter.Stage помимо конкретной стадии.
const (
	StageActiveOnly = ""  // по умолчанию: без выведенных из эксплуатации (Retired, Disposed)
	StageAny        = "*" // все стадии
)

// ListFilter — фильтры, сортировка и страница списка КЕ.
type ListFilter struct {
	Class        string
	Location     string
	Env          string
	Stage        string // StageActiveOnly, StageAny или конкретная стадия
	NamingStatus string
	Q            string // подстрока имени / серийного номера / инвентарного номера или точный IP
	Sort         string // одна из колонок sortColumns
	Desc         bool
	Page         int // с 1
	PerPage      int
}

// ListResult — страница списка КЕ.
type ListResult struct {
	Rows    []repo.Row
	Total   int
	Page    int
	PerPage int
	Pages   int
}

var sortColumns = map[string]string{
	"name": "c.name", "class": "c.ci_class", "location": "c.location_code", "env": "c.environment_code",
	"stage": "c.lifecycle_stage", "naming": "c.naming_status", "updated": "c.updated_at",
}

// PerPageOptions — допустимые размеры страницы.
var PerPageOptions = []int{25, 50, 100, 200}

// escapeLike экранирует служебные символы шаблона LIKE в пользовательской подстроке.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// List возвращает страницу КЕ по фильтру. Итог считается отдельным запросом по тем же условиям.
// whereClause строит условие WHERE и аргументы по фильтру — общее для List (постранично) и
// ExportRows (весь отфильтрованный результат, без LIMIT/OFFSET).
func whereClause(f ListFilter) (string, []any) {
	var conds []string
	var args []any
	add := func(cond string, a ...any) {
		conds = append(conds, cond)
		args = append(args, a...)
	}
	if f.Class != "" {
		add("c.ci_class = ?", f.Class)
	}
	if f.Location != "" {
		add("c.location_code = ?", f.Location)
	}
	if f.Env != "" {
		add("c.environment_code = ?", f.Env)
	}
	if f.NamingStatus != "" {
		add("c.naming_status = ?", f.NamingStatus)
	}
	switch f.Stage {
	case StageAny:
	case StageActiveOnly:
		add("c.lifecycle_stage NOT IN ('Retired', 'Disposed')")
	default:
		add("c.lifecycle_stage = ?", f.Stage)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		like := "%" + escapeLike(q) + "%"
		add(`(c.name ILIKE ? OR c.serial_number ILIKE ? OR c.asset_tag ILIKE ?
			OR EXISTS (SELECT 1 FROM cmdb.ip_address ip WHERE ip.ci_id = c.id AND host(ip.address) = ?))`, like, like, like, q)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	return where, args
}

func orderClause(f ListFilter) string {
	col, ok := sortColumns[f.Sort]
	if !ok {
		col = "c.name"
	}
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	return col + " " + dir
}

// ciSource возвращает источник строк КЕ: cmdb.ci, а если контекст запроса несёт ?asof= (E3.2/E6.1) —
// cmdb.ci_asof(ts), «как было на дату» (E2.10); args — аргументы источника (идут в SQL первыми).
func ciSource(ctx context.Context) (from string, args []any) {
	if t, ok := asof.FromContext(ctx); ok {
		return "cmdb.ci_asof(CAST(CAST(? AS text) AS timestamptz)) c", []any{repo.TimeArg(t)}
	}
	return "cmdb.ci c", nil
}

// listSelect — колонки списка КЕ, общие для List (JSON/HTML) и ExportRows (CSV/XLSX).
func listSelect(from string) string {
	return `
	SELECT c.id::text AS id, c.name, c.ci_class, COALESCE(k.name, c.ci_class) AS class_name,
	       COALESCE(c.location_code, '') AS location_code, COALESCE(c.environment_code, '') AS environment_code,
	       c.lifecycle_stage, COALESCE(c.lifecycle_status, '') AS lifecycle_status, c.naming_status,
	       COALESCE(c.serial_number, '') AS serial_number,
	       to_char(c.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
	FROM ` + from + ` LEFT JOIN ref.ci_class k ON k.class_code = c.ci_class `
}

// ExportRowsLimit — предохранитель для выгрузки без пагинации (E3.3): достаточно с запасом для
// пилотных объёмов (intent §9: ориентир ~100 тыс. КЕ в промышленной эксплуатации, но выгрузка —
// как правило, по фильтру на конкретный срез, а не весь каталог целиком).
const ExportRowsLimit = 50000

// ExportRows возвращает все КЕ по фильтру (без пагинации, до ExportRowsLimit) для выгрузки в
// CSV/XLSX — те же колонки и порядок, что в List.
func ExportRows(ctx context.Context, f ListFilter) ([]repo.Row, error) {
	from, fromArgs := ciSource(ctx)
	where, args := whereClause(f)
	args = append(append(append([]any{}, fromArgs...), args...), ExportRowsLimit)
	return repo.QueryRows(ctx, listSelect(from)+where+" ORDER BY "+orderClause(f)+", c.id LIMIT ?", args...)
}

func List(ctx context.Context, f ListFilter) (ListResult, error) {
	from, fromArgs := ciSource(ctx)
	where, args := whereClause(f)
	if f.PerPage <= 0 {
		f.PerPage = PerPageOptions[0]
	}
	if f.Page < 1 {
		f.Page = 1
	}

	res := ListResult{Page: f.Page, PerPage: f.PerPage}
	total, err := repo.QueryRow(ctx, `SELECT count(*)::text AS n FROM `+from+` `+where, append(append([]any{}, fromArgs...), args...)...)
	if err != nil {
		return res, err
	}
	fmt.Sscan(total["n"], &res.Total)
	res.Pages = (res.Total + f.PerPage - 1) / f.PerPage
	if res.Pages == 0 {
		res.Pages = 1
	}

	pageArgs := append(append(append([]any{}, fromArgs...), args...), f.PerPage, (f.Page-1)*f.PerPage)
	res.Rows, err = repo.QueryRows(ctx, listSelect(from)+where+" ORDER BY "+orderClause(f)+", c.id LIMIT ? OFFSET ?", pageArgs...)
	return res, err
}

// Detail — КЕ целиком для карточки: колонки cmdb.ci, атрибуты класса и описание формы.
type Detail struct {
	CI        repo.Row
	Ext       repo.Row // пусто, если у класса нет формы/строки расширения
	Spec      *ClassSpec
	ClassName string
}

// Get читает КЕ по id; для несуществующей возвращает svcerr.ErrNotFound. Если контекст запроса
// несёт ?asof= (internal/filters.AsOf, E3.2), колонки cmdb.ci читаются через cmdb.ci_asof(ts)
// (E2.10) — "машина времени" на уровне основных полей КЕ. Атрибуты таблицы-расширения (d.Ext)
// пока читаются только текущими: отдельных *_asof() функций для каждой из ~20 таблиц-расширений
// нет (E2.10 их не создавал) — генерировать по одной на класс вне рамок этой задачи.
func Get(ctx context.Context, id int64) (*Detail, error) {
	from := "cmdb.ci c"
	args := []any{id}
	if t, ok := asof.FromContext(ctx); ok {
		from = "cmdb.ci_asof(CAST(CAST(? AS text) AS timestamptz)) c"
		args = []any{repo.TimeArg(t), id}
	}
	row, err := repo.QueryRow(ctx, `
		SELECT c.*, COALESCE(k.name, c.ci_class) AS class_name,
		       to_char(c.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS created_fmt,
		       to_char(c.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS updated_fmt
		FROM `+from+` LEFT JOIN ref.ci_class k ON k.class_code = c.ci_class WHERE c.id = ?`, args...)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, svcerr.NotFound("ci", id)
	}
	d := &Detail{CI: row, ClassName: row["class_name"], Spec: SpecFor(row["ci_class"])}
	if d.Spec != nil {
		if t, ok := asof.FromContext(ctx); ok {
			d.Ext, err = extAsOf(ctx, d.Spec.ExtTable, id, t)
		} else {
			d.Ext, err = repo.QueryRow(ctx, `SELECT * FROM cmdb.`+d.Spec.ExtTable+` WHERE ci_id = ?`, id)
		}
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

// extAsOf читает строку таблицы-расширения КЕ на момент t: текущую версию или закрытую из hist.<таблица>.
// Отдельных *_asof() функций для расширений нет (их ~20), поэтому запрос строится по каталогу колонок
// БД: hist.<t> шире cmdb.<t> на служебные колонки hist, явный список колонок обязателен (как в cmdb.ci_asof).
func extAsOf(ctx context.Context, table string, ciID int64, t time.Time) (repo.Row, error) {
	if !ExtensionTables[table] {
		return nil, fmt.Errorf("%w: %q не таблица-расширение КЕ", svcerr.ErrInvalidState, table)
	}
	colMap, err := ExtensionColumns(ctx, orm.NewOrm(), table)
	if err != nil {
		return nil, err
	}
	cols := make([]string, 0, len(colMap))
	for c := range colMap {
		cols = append(cols, c)
	}
	sort.Strings(cols)
	list := strings.Join(cols, ", ")
	ts := repo.TimeArg(t)
	return repo.QueryRow(ctx, `
		SELECT `+list+` FROM cmdb.`+table+` WHERE ci_id = ? AND sys_period @> CAST(CAST(? AS text) AS timestamptz)
		UNION ALL
		SELECT `+list+` FROM hist.`+table+` WHERE ci_id = ? AND sys_period @> CAST(CAST(? AS text) AS timestamptz)
		LIMIT 1`, ciID, ts, ciID, ts)
}

// FindByName находит действующую (не выведенную) КЕ по имени без учёта регистра.
func FindByName(ctx context.Context, name string) (int64, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT id::text AS id FROM cmdb.ci
		WHERE lower(name) = lower(?) AND lifecycle_stage NOT IN ('Retired', 'Disposed')
		ORDER BY id LIMIT 1`, strings.TrimSpace(name))
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, fmt.Errorf("%w: КЕ с именем %q", svcerr.ErrNotFound, name)
	}
	var id int64
	fmt.Sscan(row["id"], &id)
	return id, nil
}
