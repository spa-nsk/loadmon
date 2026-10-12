package repo

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
)

// Row — строка результата в текстовом виде (NULL → пустая строка). Так удобно отдавать данные в
// шаблоны: тип колонки не важен, форматирование — на стороне представления.
type Row map[string]string

// QueryRows выполняет чтение и возвращает строки как Row. beego Values() читает все колонки через
// sql.NullString, поэтому работает для любых типов PostgreSQL (inet, jsonb, массивы, диапазоны).
func QueryRows(ctx context.Context, query string, args ...any) ([]Row, error) {
	var raw []orm.Params
	if _, err := orm.NewOrm().RawWithCtx(ctx, query, args...).Values(&raw); err != nil {
		return nil, fmt.Errorf("repo: запрос: %w", err)
	}
	rows := make([]Row, 0, len(raw))
	for _, p := range raw {
		r := make(Row, len(p))
		for k, v := range p {
			if v == nil {
				r[k] = ""
			} else {
				r[k] = fmt.Sprint(v)
			}
		}
		rows = append(rows, r)
	}
	return rows, nil
}

// QueryRow возвращает первую строку или nil, если результат пуст.
func QueryRow(ctx context.Context, query string, args ...any) (Row, error) {
	rows, err := QueryRows(ctx, query, args...)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}
