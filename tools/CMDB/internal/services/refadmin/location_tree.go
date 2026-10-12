package refadmin

import (
	"context"

	"cmdb/internal/repo"
)

// PlaceNode — один узел дерева размещения площадки (здание → этаж → комната → ряд → стойка →
// юнит, intent §5.2) в глубину — тот же приём, что component.ListForCI использует для дерева
// компонентов КЕ (плоский список + Depth, а не вложенные структуры: проще отдать в шаблон через
// общую indent-функцию, см. internal/controllers/ui/funcs.go).
type PlaceNode struct {
	Row   repo.Row
	Depth int
}

// LocationTree строит дерево ref.site_place для площадки locationCode (plan.md E3.8: "дерево
// локаций до стойки (с юнитами)").
func LocationTree(ctx context.Context, locationCode string) ([]PlaceNode, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT sp.id::text AS id, COALESCE(sp.parent_id::text, '') AS parent_id, sp.kind, sp.code,
		       COALESCE(sp.name, '') AS name, COALESCE(sp.rack_units::text, '') AS rack_units
		FROM ref.site_place sp JOIN ref.location l ON l.id = sp.location_id
		WHERE l.location_code = ?
		ORDER BY sp.kind, sp.code`, locationCode)
	if err != nil {
		return nil, err
	}

	ids := make(map[string]bool, len(rows))
	for _, r := range rows {
		ids[r["id"]] = true
	}
	children := map[string][]repo.Row{}
	var roots []repo.Row
	for _, r := range rows {
		if p := r["parent_id"]; p == "" || !ids[p] {
			roots = append(roots, r)
		} else {
			children[p] = append(children[p], r)
		}
	}

	var out []PlaceNode
	var walk func(rs []repo.Row, depth int)
	walk = func(rs []repo.Row, depth int) {
		for _, r := range rs {
			out = append(out, PlaceNode{Row: r, Depth: depth})
			walk(children[r["id"]], depth+1)
		}
	}
	walk(roots, 0)
	return out, nil
}
