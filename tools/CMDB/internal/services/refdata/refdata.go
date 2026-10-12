// Package refdata отдаёт справочники в виде пар «значение — подпись» для выпадающих списков форм и фильтров.
package refdata

import (
	"context"

	"cmdb/internal/repo"
)

// Option — элемент выпадающего списка.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func options(ctx context.Context, query string) ([]Option, error) {
	rows, err := repo.QueryRows(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]Option, len(rows))
	for i, r := range rows {
		out[i] = Option{Value: r["value"], Label: r["label"]}
	}
	return out, nil
}

// Locations — площадки: код и название.
func Locations(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT location_code AS value, location_code || ' — ' || location_name AS label
		FROM ref.location ORDER BY location_code`)
}

// Environments — окружения (PROD, TEST, …).
func Environments(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT environment_code AS value, environment_code || ' — ' || name AS label
		FROM ref.environment ORDER BY environment_code`)
}

// Companies — компании.
func Companies(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT company_code AS value, company_code || ' — ' || company_name AS label
		FROM ref.company ORDER BY company_code`)
}

// ComponentTypes — типы компонентов.
func ComponentTypes(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT type_code AS value, name || ' (' || type_code || ')' AS label
		FROM ref.component_type ORDER BY name`)
}

// RelTypes — типы связей (подпись — прямое название).
func RelTypes(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT rel_code AS value, name_fwd AS label FROM ref.rel_type ORDER BY rel_code`)
}

// Classes — классы КЕ (для фильтра списка).
func Classes(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT class_code AS value, name AS label FROM ref.ci_class ORDER BY name`)
}

// Clusters — действующие кластеры (для фильтра отчёта «ВМ по кластерам», plan.md E7.5).
func Clusters(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT id::text AS value, name AS label FROM cmdb.ci
		WHERE ci_class = 'cluster' AND lifecycle_stage NOT IN ('Retired', 'Disposed') ORDER BY name`)
}

// Software — справочник ПО (для выбора установки на вкладке «ПО» карточки КЕ, plan.md E7.7);
// политика видна прямо в подписи — админ сразу видит, что выбирает запрещённое.
func Software(ctx context.Context) ([]Option, error) {
	return options(ctx, `SELECT id::text AS value,
		vendor || ' ' || product || COALESCE(' ' || version, '') || COALESCE(' (' || edition || ')', '') || ' — ' || policy AS label
		FROM ref.software ORDER BY vendor, product, version`)
}
