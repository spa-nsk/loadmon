package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// ExtensionTables — таблицы-расширения КЕ 1:1 (PK = ci_id, intent §7.3). Белый список: имя таблицы
// подставляется в SQL, поэтому принимаются только эти значения.
var ExtensionTables = map[string]bool{
	"ci_server": true, "ci_blade_chassis": true, "ci_hypervisor": true, "ci_vm": true, "ci_cluster": true,
	"ci_k8s": true, "ci_virt_manager": true, "ci_datastore": true, "ci_storage_array": true,
	"ci_storage_pool": true, "ci_volume": true, "ci_svm": true, "ci_qtree": true, "ci_share": true,
	"ci_object_store": true, "ci_fc_switch": true, "ci_san_zone": true, "ci_tape_library": true,
	"ci_backup": true, "ci_network_device": true, "ci_facility": true, "ci_db_instance": true,
	"ci_cluster_resource": true,
}

// Служебные колонки расширений задаёт система, менять их через Attrs нельзя.
var reservedColumns = map[string]bool{
	"ci_id": true, "created_at": true, "created_by": true, "updated_at": true, "updated_by": true,
	"change_set_id": true, "row_version": true, "sys_period": true,
}

var safeTypeName = regexp.MustCompile(`^[a-z0-9_ ()\[\],]+$`)

// ExtensionInput — запись/правка атрибутов класса КЕ в таблице-расширении.
type ExtensionInput struct {
	CIID  int64
	Table string         // одна из ExtensionTables
	Attrs map[string]any // имя колонки → значение (nil = NULL); допустимы string, bool, числа, []string, time.Time
	// ExpectedVersion — row_version, который видел пользователь; nil для первичного создания
	// или системных загрузок без оптимистической блокировки.
	ExpectedVersion *int
	Reason          string
	UserID          int64
	// Source — источник для changectx.Meta в UpsertExtension (собственная транзакция); при вызове
	// UpsertExtensionTx внутри чужой транзакции (создание КЕ, импорт) не используется. "" — SourceManual.
	Source changectx.SourceKind
}

// ExtensionResult — что произошло с записью расширения.
type ExtensionResult struct {
	Created     bool // строки не было — создана
	Changed     bool // значения отличались — записана новая версия (при Created всегда true)
	RowVersion  int
	ChangeSetID int64
}

// UpsertExtension создаёт или обновляет строку расширения КЕ. Значения передаются как текст и
// приводятся к типу колонки на стороне БД (CAST), поэтому одна функция обслуживает все 23
// таблицы; список допустимых колонок берётся из каталога PostgreSQL. Повторная запись тех же
// значений не создаёт новой версии в hist.
func UpsertExtension(ctx context.Context, in ExtensionInput) (ExtensionResult, error) {
	var res ExtensionResult
	if !ExtensionTables[in.Table] {
		return res, fmt.Errorf("%w: %q не таблица-расширение КЕ", svcerr.ErrInvalidState, in.Table)
	}
	if len(in.Attrs) == 0 {
		return res, svcerr.ErrNoChanges
	}

	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	csID, err := changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		return UpsertExtensionTx(ctx, tx, in, &res)
	})
	res.ChangeSetID = csID
	return res, err
}

// UpsertExtensionTx — тело UpsertExtension внутри уже открытой транзакции ChangeContext. Кроме
// создания КЕ (in.Attrs записывается в тот же change set) им пользуется пайплайн импорта
// (internal/importer/pipeline, E4.8) — та же CAST-по-каталогу-БД логика годится для любого
// источника записи атрибутов класса, не только формы ручного ввода. in.Attrs пуст не должен быть
// (см. UpsertExtension) — при создании КЕ без атрибутов вызывающая сторона вставляет голую строку
// расширения сама (см. internal/services/ci/create_server.go).
func UpsertExtensionTx(ctx context.Context, tx orm.TxOrmer, in ExtensionInput, res *ExtensionResult) error {
	cols, err := ExtensionColumns(ctx, tx, in.Table)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(in.Attrs))
	for name := range in.Attrs {
		t, ok := cols[name]
		if !ok || reservedColumns[name] {
			return fmt.Errorf("%w: колонка %q недоступна в cmdb.%s", svcerr.ErrInvalidState, name, in.Table)
		}
		if !safeTypeName.MatchString(t) {
			return fmt.Errorf("ci: неожиданный тип колонки %s.%s: %q", in.Table, name, t)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	values := make([]any, len(names))
	casts := make([]string, len(names))
	for i, name := range names {
		v, err := PGText(in.Attrs[name])
		if err != nil {
			return fmt.Errorf("ci: значение колонки %q: %w", name, err)
		}
		values[i] = v
		casts[i] = "CAST(CAST(? AS text) AS " + cols[name] + ")"
	}

	var exists int
	err = tx.RawWithCtx(ctx, `SELECT 1 FROM cmdb.ci WHERE id = ?`, in.CIID).QueryRow(&exists)
	if err == orm.ErrNoRows {
		return svcerr.NotFound("ci", in.CIID)
	}
	if err != nil {
		return fmt.Errorf("ci: чтение КЕ: %w", err)
	}

	var version int
	err = tx.RawWithCtx(ctx, `SELECT row_version FROM cmdb.`+in.Table+` WHERE ci_id = ? FOR UPDATE`, in.CIID).QueryRow(&version)
	switch {
	case err == orm.ErrNoRows:
		return insertExtension(ctx, tx, in, names, casts, values, res)
	case err != nil:
		return fmt.Errorf("ci: чтение cmdb.%s: %w", in.Table, err)
	}

	if in.ExpectedVersion != nil && *in.ExpectedVersion != version {
		return svcerr.Conflict(in.Table, in.CIID, *in.ExpectedVersion, version)
	}
	return updateExtension(ctx, tx, in, names, casts, values, version, res)
}

func insertExtension(ctx context.Context, tx orm.TxOrmer, in ExtensionInput, names, casts []string, values []any, res *ExtensionResult) error {
	args := append([]any{in.CIID}, values...)
	args = append(args, in.UserID, in.UserID)
	q := fmt.Sprintf(`INSERT INTO cmdb.%s (ci_id, %s, created_by, updated_by) VALUES (?, %s, ?, ?) RETURNING row_version`,
		in.Table, strings.Join(names, ", "), strings.Join(casts, ", "))
	if err := tx.RawWithCtx(ctx, q, args...).QueryRow(&res.RowVersion); err != nil {
		return fmt.Errorf("ci: вставка в cmdb.%s: %w", in.Table, err)
	}
	res.Created, res.Changed = true, true
	return nil
}

func updateExtension(ctx context.Context, tx orm.TxOrmer, in ExtensionInput, names, casts []string, values []any, version int, res *ExtensionResult) error {
	sets := make([]string, len(names))
	for i, name := range names {
		sets[i] = name + " = " + casts[i]
	}
	// Без WHERE ... IS DISTINCT FROM триггер записал бы версию в hist и при неизменных данных.
	args := append([]any{}, values...)
	args = append(args, in.UserID, in.CIID)
	args = append(args, values...)
	q := fmt.Sprintf(`UPDATE cmdb.%s SET %s, updated_at = now(), updated_by = ?
		WHERE ci_id = ? AND ROW(%s) IS DISTINCT FROM ROW(%s) RETURNING row_version`,
		in.Table, strings.Join(sets, ", "), strings.Join(names, ", "), strings.Join(casts, ", "))
	err := tx.RawWithCtx(ctx, q, args...).QueryRow(&res.RowVersion)
	if err == orm.ErrNoRows {
		res.RowVersion = version // значения совпали с текущими
		return nil
	}
	if err != nil {
		return fmt.Errorf("ci: обновление cmdb.%s: %w", in.Table, err)
	}
	res.Changed = true
	return nil
}

// ExtensionColumns возвращает колонки таблицы cmdb.<table> и их типы (format_type). Параметр —
// только для чтения (rawQuerier), поэтому годится и в транзакции, и вне её (dry-run пайплайна
// импорта, internal/importer/pipeline).
func ExtensionColumns(ctx context.Context, q rawQuerier, table string) (map[string]string, error) {
	var rows []orm.Params
	_, err := q.RawWithCtx(ctx, `
		SELECT a.attname AS name, format_type(a.atttypid, a.atttypmod) AS type
		FROM pg_attribute a
		WHERE a.attrelid = to_regclass('cmdb.' || ?) AND a.attnum > 0 AND NOT a.attisdropped
	`, table).Values(&rows)
	if err != nil {
		return nil, fmt.Errorf("ci: чтение колонок cmdb.%s: %w", table, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("ci: таблица cmdb.%s не найдена — примените миграции", table)
	}
	cols := make(map[string]string, len(rows))
	for _, r := range rows {
		cols[fmt.Sprint(r["name"])] = fmt.Sprint(r["type"])
	}
	return cols, nil
}

// PGText приводит значение к текстовому виду, понятному PostgreSQL при CAST(text AS <тип>).
func PGText(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case string:
		return x, nil
	case bool:
		return strconv.FormatBool(x), nil
	case int:
		return strconv.Itoa(x), nil
	case int32:
		return strconv.FormatInt(int64(x), 10), nil
	case int64:
		return strconv.FormatInt(x, 10), nil
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64), nil
	case json.Number:
		return x.String(), nil
	case time.Time:
		return x.Format(time.RFC3339Nano), nil
	case []string:
		parts := make([]string, len(x))
		for i, s := range x {
			s = strings.ReplaceAll(s, `\`, `\\`)
			parts[i] = `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
		}
		return "{" + strings.Join(parts, ",") + "}", nil
	default:
		return nil, fmt.Errorf("неподдерживаемый тип %T", v)
	}
}
