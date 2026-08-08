package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type columnInfo struct {
	CID          int    `json:"cid"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	NotNull      bool   `json:"notNull"`
	DefaultValue any    `json:"defaultValue"`
	PrimaryKey   bool   `json:"primaryKey"`
}

type tableSchema struct {
	Name       string       `json:"name"`
	CreateSQL  string       `json:"createSql"`
	Columns    []columnInfo `json:"columns"`
	PrimaryKey []string     `json:"primaryKey"`
	Indexes    []string     `json:"indexes"`
}

type queryResult struct {
	Columns []string   `json:"columns"`
	Rows    [][]any    `json:"rows"`
	RowsAffected int64 `json:"rowsAffected,omitempty"`
	Message string     `json:"message,omitempty"`
}

func openSQLite(path string) (*sql.DB, error) {
	dsn := path + "?_journal_mode=WAL&_sync=NORMAL&cache=shared"
	return sql.Open("sqlite3", dsn)
}

func listTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func tableSchemaInfo(db *sql.DB, table string) (*tableSchema, error) {
	if !isSafeIdent(table) {
		return nil, fmt.Errorf("недопустимое имя таблицы")
	}

	var createSQL string
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&createSQL)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", quoteIdent(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	schema := &tableSchema{Name: table, CreateSQL: createSQL}
	for rows.Next() {
		var col columnInfo
		var notNull int
		var pk int
		var def sql.NullString
		if err := rows.Scan(&col.CID, &col.Name, &col.Type, &notNull, &def, &pk); err != nil {
			return nil, err
		}
		col.NotNull = notNull != 0
		col.PrimaryKey = pk != 0
		if def.Valid {
			col.DefaultValue = def.String
		}
		if col.PrimaryKey {
			schema.PrimaryKey = append(schema.PrimaryKey, col.Name)
		}
		schema.Columns = append(schema.Columns, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	idxRows, err := db.Query(`SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql IS NOT NULL`, table)
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()
	for idxRows.Next() {
		var idxSQL string
		if err := idxRows.Scan(&idxSQL); err != nil {
			return nil, err
		}
		schema.Indexes = append(schema.Indexes, idxSQL)
	}
	return schema, idxRows.Err()
}

func tableRows(db *sql.DB, table string, limit, offset int) (*queryResult, error) {
	if !isSafeIdent(table) {
		return nil, fmt.Errorf("недопустимое имя таблицы")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	q := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", quoteIdent(table))
	return runSelect(db, q, limit, offset)
}

func updateRow(db *sql.DB, table string, pk map[string]any, values map[string]any) (int64, error) {
	if !isSafeIdent(table) {
		return 0, fmt.Errorf("недопустимое имя таблицы")
	}
	if len(pk) == 0 {
		return 0, fmt.Errorf("не указан первичный ключ")
	}
	if len(values) == 0 {
		return 0, fmt.Errorf("нет полей для обновления")
	}

	setParts := make([]string, 0, len(values))
	args := make([]any, 0, len(values)+len(pk))
	for col, val := range values {
		if !isSafeIdent(col) {
			return 0, fmt.Errorf("недопустимое имя колонки: %s", col)
		}
		setParts = append(setParts, fmt.Sprintf("%s=?", quoteIdent(col)))
		args = append(args, val)
	}

	whereParts := make([]string, 0, len(pk))
	for col, val := range pk {
		if !isSafeIdent(col) {
			return 0, fmt.Errorf("недопустимое имя колонки: %s", col)
		}
		whereParts = append(whereParts, fmt.Sprintf("%s=?", quoteIdent(col)))
		args = append(args, val)
	}

	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		quoteIdent(table),
		strings.Join(setParts, ", "),
		strings.Join(whereParts, " AND "),
	)
	res, err := db.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func runSQL(db *sql.DB, sqlText string) (*queryResult, error) {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil, fmt.Errorf("пустой SQL запрос")
	}

	upper := strings.ToUpper(strings.TrimSpace(sqlText))
	if strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "PRAGMA") || strings.HasPrefix(upper, "WITH") || strings.HasPrefix(upper, "EXPLAIN") {
		return runSelect(db, sqlText)
	}

	res, err := db.Exec(sqlText)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	lastID, _ := res.LastInsertId()
	msg := fmt.Sprintf("OK, затронуто строк: %d", affected)
	if lastID > 0 {
		msg += fmt.Sprintf(", last_insert_rowid: %d", lastID)
	}
	return &queryResult{Message: msg, RowsAffected: affected}, nil
}

func runSelect(db *sql.DB, query string, args ...any) (*queryResult, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &queryResult{Columns: cols}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]any, len(cols))
		for i, v := range vals {
			switch x := v.(type) {
			case []byte:
				row[i] = string(x)
			default:
				row[i] = x
			}
		}
		result.Rows = append(result.Rows, row)
	}
	return result, rows.Err()
}

func isSafeIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if r != '_' && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
				return false
			}
			continue
		}
		if r != '_' && (r < '0' || r > '9') && (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
