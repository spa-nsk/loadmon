// Package provenance читает cmdb.attr_source — откуда взято текущее значение атрибута и не
// заблокировано ли оно ручной правкой (intent §7.3, plan.md вкладка «Источники», E3.4). Таблица
// пока не заполняется: значения появятся с импортом (E4.7 reconcile) — до тех пор вкладка
// показывает пустой список, что честно отражает состояние "источник ещё не подтверждал значение".
package provenance

import (
	"context"

	"cmdb/internal/repo"
)

// ListForCI возвращает провенанс атрибутов КЕ: собственных колонок cmdb.ci и, если у класса есть
// таблица-расширение, её колонок (attr_source.entity_id — общий id, ci.id == <ext_table>.ci_id).
func ListForCI(ctx context.Context, ciID int64, extTable string) ([]repo.Row, error) {
	const cols = `entity, attr_name, COALESCE(source_system_code, '') AS source_system_code,
		       COALESCE(source_value::text, '') AS source_value,
		       to_char(observed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS observed_at,
		       is_locked, COALESCE(locked_by::text, '') AS locked_by`
	if extTable == "" {
		return repo.QueryRows(ctx, `SELECT `+cols+` FROM cmdb.attr_source WHERE entity_id = ? AND entity = 'ci' ORDER BY attr_name`, ciID)
	}
	return repo.QueryRows(ctx, `SELECT `+cols+` FROM cmdb.attr_source WHERE entity_id = ? AND entity IN ('ci', ?) ORDER BY entity, attr_name`, ciID, extTable)
}
