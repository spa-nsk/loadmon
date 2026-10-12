package refadmin

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"github.com/jackc/pgx/v5/pgconn"

	"cmdb/internal/repo"
	"cmdb/internal/services/svcerr"
)

// List возвращает все строки таблицы (столбцы — как в БД; список показывает только
// Field.ListShow), отфильтрованные строкой поиска по TableSpec.SearchCols (ILIKE), в порядке
// spec.OrderBy (по умолчанию — по PK). Справочники малы (десятки-сотни строк) — постраничность
// не нужна (в отличие от cmdb.ci, см. ci.List).
func List(ctx context.Context, spec TableSpec, q string) ([]repo.Row, error) {
	query := fmt.Sprintf("SELECT * FROM %s", spec.qualified())
	var args []any
	if q = strings.TrimSpace(q); q != "" && len(spec.SearchCols) > 0 {
		conds := make([]string, len(spec.SearchCols))
		for i, col := range spec.SearchCols {
			conds[i] = col + "::text ILIKE ?"
			args = append(args, "%"+q+"%")
		}
		query += " WHERE " + strings.Join(conds, " OR ")
	}
	order := spec.OrderBy
	if order == "" {
		order = spec.pk().Name
	}
	query += " ORDER BY " + order
	rows, err := repo.QueryRows(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("refadmin: список %s: %w", spec.Table, err)
	}
	return rows, nil
}

// Get возвращает одну строку по значению первичного ключа.
func Get(ctx context.Context, spec TableSpec, pkValue string) (repo.Row, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", spec.qualified(), spec.pk().Name)
	row, err := repo.QueryRow(ctx, query, pkValue)
	if err != nil {
		return nil, fmt.Errorf("refadmin: чтение %s: %w", spec.Table, err)
	}
	if row == nil {
		return nil, fmt.Errorf("%w: %s %s=%s", svcerr.ErrNotFound, spec.Table, spec.pk().Name, pkValue)
	}
	return row, nil
}

// requiredMissing проверяет обязательные поля (кроме генерируемых/PK суррогатных ключей).
func requiredMissing(spec TableSpec, values map[string]string, isCreate bool) error {
	for _, f := range spec.Fields {
		if f.Generated || (f.PK && !isCreate) {
			continue
		}
		if f.Required && strings.TrimSpace(values[f.Name]) == "" {
			return fmt.Errorf("%w: поле %q обязательно", svcerr.ErrInvalidInput, f.Label)
		}
	}
	return nil
}

// argFor приводит текстовое значение формы к аргументу SQL-запроса для поля f.
func argFor(f Field, raw string) (any, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		switch {
		case f.Nullable:
			return nil, nil
		case f.Type == TypeJSONB:
			return "{}", nil
		case f.Type == TypeTextArr:
			return "{}", nil
		case f.Type == TypeBool:
			return "false", nil
		default:
			return v, nil
		}
	}
	switch f.Type {
	case TypeBool:
		switch v {
		case "on", "true", "1", "yes":
			return "true", nil
		default:
			return "false", nil
		}
	case TypeJSONB:
		if !json.Valid([]byte(v)) {
			return nil, fmt.Errorf("%w: поле %q — некорректный JSON", svcerr.ErrInvalidInput, f.Label)
		}
		return v, nil
	case TypeTextArr:
		parts := strings.Split(v, ",")
		for i, p := range parts {
			p = strings.TrimSpace(p)
			p = strings.ReplaceAll(p, `\`, `\\`)
			p = strings.ReplaceAll(p, `"`, `\"`)
			parts[i] = `"` + p + `"`
		}
		return "{" + strings.Join(parts, ",") + "}", nil
	default:
		return v, nil
	}
}

func castPlaceholder(f Field) string {
	if f.Type == TypeText || f.Type == TypeTextarea || f.Type == TypeSelect {
		return "?"
	}
	return "CAST(? AS " + pgType(f) + ")"
}

// Create вставляет новую строку. userID проставляется в created_by/updated_by, если у таблицы
// есть такие колонки (TableSpec.HasCreatedBy); значения генерируемых колонок (PK identity,
// created_at и т.п.) не передаются — их выставляет БД.
func Create(ctx context.Context, spec TableSpec, values map[string]string, userID int64) error {
	if err := requiredMissing(spec, values, true); err != nil {
		return err
	}
	return withTx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		if spec.BeforeSave != nil {
			if err := spec.BeforeSave(ctx, tx, true, values); err != nil {
				return err
			}
		}
		var cols, placeholders []string
		var args []any
		for _, f := range spec.Fields {
			if f.Generated || (f.PK && f.Generated) {
				continue
			}
			arg, err := argFor(f, values[f.Name])
			if err != nil {
				return err
			}
			cols = append(cols, f.Name)
			placeholders = append(placeholders, castPlaceholder(f))
			args = append(args, arg)
		}
		if spec.HasCreatedAt {
			cols, placeholders = append(cols, "created_at"), append(placeholders, "now()")
		}
		if spec.HasUpdatedAt {
			cols, placeholders = append(cols, "updated_at"), append(placeholders, "now()")
		}
		if spec.HasCreatedBy {
			cols, placeholders, args = append(cols, "created_by"), append(placeholders, "?"), append(args, userID)
		}
		if spec.HasUpdatedBy {
			cols, placeholders, args = append(cols, "updated_by"), append(placeholders, "?"), append(args, userID)
		}
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", spec.qualified(), strings.Join(cols, ", "), strings.Join(placeholders, ", "))
		if _, err := tx.RawWithCtx(ctx, query, args...).Exec(); err != nil {
			return translateErr(err)
		}
		return nil
	})
}

// Update правит строку по первичному ключу (не редактируется — исключается из SET).
func Update(ctx context.Context, spec TableSpec, pkValue string, values map[string]string, userID int64) error {
	if err := requiredMissing(spec, values, false); err != nil {
		return err
	}
	return withTx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		if spec.BeforeSave != nil {
			if err := spec.BeforeSave(ctx, tx, false, values); err != nil {
				return err
			}
		}
		var sets []string
		var args []any
		for _, f := range spec.Fields {
			if f.PK || f.Generated {
				continue
			}
			arg, err := argFor(f, values[f.Name])
			if err != nil {
				return err
			}
			sets = append(sets, f.Name+" = "+castPlaceholder(f))
			args = append(args, arg)
		}
		if spec.HasUpdatedAt {
			sets = append(sets, "updated_at = now()")
		}
		if spec.HasUpdatedBy {
			sets, args = append(sets, "updated_by = ?"), append(args, userID)
		}
		if len(sets) == 0 {
			return fmt.Errorf("%w", svcerr.ErrNoChanges)
		}
		args = append(args, pkValue)
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", spec.qualified(), strings.Join(sets, ", "), spec.pk().Name)
		res, err := tx.RawWithCtx(ctx, query, args...).Exec()
		if err != nil {
			return translateErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("%w: %s %s=%s", svcerr.ErrNotFound, spec.Table, spec.pk().Name, pkValue)
		}
		return nil
	})
}

// Delete удаляет строку по первичному ключу. Ссылки из других справочников/КЕ (FOREIGN KEY без
// ON DELETE CASCADE — стандарт для ref.*, см. миграции 0002-0003) превращаются в понятную ошибку,
// а не в сырой текст PostgreSQL.
func Delete(ctx context.Context, spec TableSpec, pkValue string) error {
	return withTx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", spec.qualified(), spec.pk().Name)
		res, err := tx.RawWithCtx(ctx, query, pkValue).Exec()
		if err != nil {
			return translateErr(err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("%w: %s %s=%s", svcerr.ErrNotFound, spec.Table, spec.pk().Name, pkValue)
		}
		return nil
	})
}

func withTx(ctx context.Context, fn func(ctx context.Context, tx orm.TxOrmer) error) error {
	return orm.NewOrm().DoTxWithCtx(ctx, fn)
}

// translateErr сообщает о нарушении уникальности/внешнего ключа понятным текстом, не разбирая
// имена конкретных constraint'ов (у ~20 таблиц их слишком много, чтобы перечислять поимённо).
func translateErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w (%s)", svcerr.ErrDuplicate, pgErr.ConstraintName)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: значение используется в другой таблице (%s) или ссылается на несуществующую запись", svcerr.ErrInvalidState, pgErr.ConstraintName)
		}
	}
	return fmt.Errorf("refadmin: %w", err)
}

// ---------- CSV ----------

// ExportCSV пишет все строки таблицы в CSV: заголовок — Field.Name в порядке spec.Fields.
func ExportCSV(ctx context.Context, spec TableSpec, w io.Writer) error {
	rows, err := List(ctx, spec, "")
	if err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	header := make([]string, len(spec.Fields))
	for i, f := range spec.Fields {
		header[i] = f.Name
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rows {
		rec := make([]string, len(spec.Fields))
		for i, f := range spec.Fields {
			rec[i] = r[f.Name]
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ImportResult — итог импорта CSV.
type ImportResult struct {
	Inserted int
	Updated  int
	Errors   []string // "строка N: причина" — импорт не прерывается на первой ошибке
}

// ImportCSV построчно загружает CSV (та же колоночная форма, что и ExportCSV — обязателен
// заголовок с именами колонок, порядок не важен) и делает upsert по первичному ключу: если
// значение PK уже существует — Update, иначе — Create. Каждая строка — отдельная попытка,
// ошибка одной не прерывает импорт остальных (E1.8: "импорт/экспорт CSV").
func ImportCSV(ctx context.Context, spec TableSpec, r io.Reader, userID int64) (ImportResult, error) {
	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		if err == io.EOF {
			return ImportResult{}, fmt.Errorf("%w: пустой файл", svcerr.ErrInvalidInput)
		}
		return ImportResult{}, fmt.Errorf("%w: не удалось прочитать заголовок CSV: %v", svcerr.ErrInvalidInput, err)
	}
	pkName := spec.pk().Name
	var res ImportResult
	rowNum := 1
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		rowNum++
		if err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", rowNum, err))
			continue
		}
		values := make(map[string]string, len(header))
		for i, col := range header {
			if i < len(rec) {
				values[col] = rec[i]
			}
		}
		pkVal := values[pkName]
		if pkVal == "" {
			if err := Create(ctx, spec, values, userID); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", rowNum, err))
				continue
			}
			res.Inserted++
			continue
		}
		existing, err := Get(ctx, spec, pkVal)
		switch {
		case errors.Is(err, svcerr.ErrNotFound):
			if err := Create(ctx, spec, values, userID); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", rowNum, err))
				continue
			}
			res.Inserted++
		case err != nil:
			res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", rowNum, err))
		default:
			_ = existing
			if err := Update(ctx, spec, pkVal, values, userID); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("строка %d: %v", rowNum, err))
				continue
			}
			res.Updated++
		}
	}
	return res, nil
}
