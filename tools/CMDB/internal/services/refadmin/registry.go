package refadmin

import (
	"context"

	"cmdb/internal/repo"
)

var specsBySlug = bySlug()

// Spec возвращает описание справочника по сегменту URL.
func Spec(slug string) (TableSpec, bool) {
	s, ok := specsBySlug[slug]
	return s, ok
}

// FKOption — вариант выпадающего списка для поля с внешним ключом.
type FKOption struct {
	Value string
	Label string
}

// FKOptions выполняет FK.Query поля и возвращает варианты для <select>.
func FKOptions(ctx context.Context, f Field) ([]FKOption, error) {
	if f.FK == nil {
		return nil, nil
	}
	rows, err := repo.QueryRows(ctx, f.FK.Query)
	if err != nil {
		return nil, err
	}
	out := make([]FKOption, len(rows))
	for i, r := range rows {
		out[i] = FKOption{Value: r["value"], Label: r["label"]}
	}
	return out, nil
}
