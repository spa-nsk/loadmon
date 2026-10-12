// Package report — типовые выгрузки (plan.md E7.5): инвентарь по локации/стойке, ёмкость СХД, ВМ
// по кластерам, КЕ без владельца, отклонения именования. Изменения за период — существующий
// internal/services/history.GlobalLog с диапазоном дат, здесь не дублируется. Все запросы читают
// cmdb.* напрямую (агрегаты/списки, не запись) — обычные repo.Row, как в internal/services/ci.List,
// без отдельных Go-типов там, где колонки один в один идут в таблицу отчёта.
package report

import (
	"context"
	"fmt"

	"cmdb/internal/repo"
)

// activeFilter — «действующая КЕ» = не выведенная из эксплуатации (E2.6), как и везде в приложении.
const activeFilter = `c.lifecycle_stage NOT IN ('Retired', 'Disposed')`

// InventorySummary — число действующих КЕ по локации, стойке (rack_position; КЕ без значения —
// отдельной строкой "(не указано)": оно не обязательно и по факту заполняется не всегда, см.
// комментарий к InventoryRows) и классу — сводка для страницы отчёта.
func InventorySummary(ctx context.Context, locationCode string) ([]repo.Row, error) {
	where, args := activeFilter, []any{}
	if locationCode != "" {
		where += " AND c.location_code = ?"
		args = append(args, locationCode)
	}
	return repo.QueryRows(ctx, `
		SELECT c.location_code, COALESCE(NULLIF(c.rack_position, ''), '(не указано)') AS rack_position,
		       k.name AS class_name, count(*)::text AS n
		FROM cmdb.ci c JOIN ref.ci_class k ON k.class_code = c.ci_class
		WHERE `+where+`
		GROUP BY c.location_code, rack_position, k.name
		ORDER BY c.location_code, rack_position, k.name`, args...)
}

// InventoryRows — построчный инвентарь для выгрузки (E7.5: «инвентарь по локации/стойке»).
// rack_position — свободный текст (cmdb.ci.rack_position); привязка к дереву размещения
// (ref.site_place, cmdb.ci.site_place_id) в модели данных предусмотрена, но её сегодня не
// заполняет ни один коннектор и ни одна форма (E1/E5) — строить отчёт по ней означало бы отчёт по
// всегда пустой колонке. Отчёт честно группирует по тому, что реально заполняется.
func InventoryRows(ctx context.Context, locationCode string) ([]repo.Row, error) {
	where, args := activeFilter, []any{}
	if locationCode != "" {
		where += " AND c.location_code = ?"
		args = append(args, locationCode)
	}
	return repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, k.name AS class_name, COALESCE(c.location_code, '') AS location_code,
		       COALESCE(c.rack_position, '') AS rack_position, c.lifecycle_stage,
		       COALESCE(c.criticality, '') AS criticality, COALESCE(c.serial_number, '') AS serial_number,
		       COALESCE(c.asset_tag, '') AS asset_tag
		FROM cmdb.ci c JOIN ref.ci_class k ON k.class_code = c.ci_class
		WHERE `+where+`
		ORDER BY c.location_code, c.rack_position, c.name`, args...)
}

// StorageArrays — ёмкость массивов СХД (E7.5): заявленная (raw/usable, из паспорта массива) и
// фактически размещённая (сумма cmdb.ci_volume.size_gb томов, выделенных из массива) — оба числа
// приходят из разных источников (raw/usable — из самого массива, тома — от каждой СХД отдельно),
// расхождение — тоже полезный сигнал (например, забыли привязать том к массиву при импорте).
func StorageArrays(ctx context.Context) ([]repo.Row, error) {
	return repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, COALESCE(sa.array_family, '') AS array_family,
		       COALESCE(sa.raw_tb::text, '') AS raw_tb, COALESCE(sa.usable_tb::text, '') AS usable_tb,
		       to_char(COALESCE((SELECT sum(v.size_gb) FROM cmdb.ci_volume v WHERE v.array_ci_id = c.id), 0) / 1024.0, 'FM999999990.00') AS allocated_tb
		FROM cmdb.ci c JOIN cmdb.ci_storage_array sa ON sa.ci_id = c.id
		WHERE `+activeFilter+`
		ORDER BY c.name`)
}

// StoragePools — то же самое на уровне пулов (E7.5), с именем массива-владельца для связки с
// StorageArrays на странице отчёта.
func StoragePools(ctx context.Context) ([]repo.Row, error) {
	return repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, COALESCE(arr.name, '') AS array_name, COALESCE(sp.tier, '') AS tier,
		       COALESCE(sp.capacity_tb::text, '') AS capacity_tb,
		       to_char(COALESCE((SELECT sum(v.size_gb) FROM cmdb.ci_volume v WHERE v.pool_ci_id = c.id), 0) / 1024.0, 'FM999999990.00') AS allocated_tb
		FROM cmdb.ci c JOIN cmdb.ci_storage_pool sp ON sp.ci_id = c.id
		LEFT JOIN cmdb.ci arr ON arr.id = sp.array_ci_id
		WHERE `+activeFilter+`
		ORDER BY array_name, c.name`)
}

// VMsByCluster — ВМ, сгруппированные по кластеру (E7.5): путь vm --runs_on--> host --member_of-->
// cluster (internal/importer/connectors/vcenter, E5.1) через LEFT JOIN — у ВМ без хоста или у
// хоста вне кластера просто пустое имя кластера, а не потерянная строка; clusterID>0 сужает до
// одного кластера (страница отчёта с фильтром).
func VMsByCluster(ctx context.Context, clusterID int64) ([]repo.Row, error) {
	where, args := `v.lifecycle_stage NOT IN ('Retired', 'Disposed')`, []any{}
	if clusterID > 0 {
		where += " AND cl.id = ?"
		args = append(args, clusterID)
	}
	return repo.QueryRows(ctx, `
		SELECT v.id::text AS id, v.name, COALESCE(cl.name, '') AS cluster_name, COALESCE(h.name, '') AS host_name,
		       COALESCE(ext.vcpu::text, '') AS vcpu, COALESCE(ext.ram_gb::text, '') AS ram_gb,
		       COALESCE(ext.guest_os, '') AS guest_os, v.lifecycle_stage
		FROM cmdb.ci v
		LEFT JOIN cmdb.ci_rel r1 ON r1.src_ci_id = v.id AND r1.rel_type = 'runs_on' AND upper_inf(r1.valid_period)
		LEFT JOIN cmdb.ci h ON h.id = r1.dst_ci_id
		LEFT JOIN cmdb.ci_rel r2 ON r2.src_ci_id = h.id AND r2.rel_type = 'member_of' AND upper_inf(r2.valid_period)
		LEFT JOIN cmdb.ci cl ON cl.id = r2.dst_ci_id
		LEFT JOIN cmdb.ci_vm ext ON ext.ci_id = v.id
		WHERE v.ci_class = 'vm' AND `+where+`
		ORDER BY COALESCE(cl.name, '￿'), v.name`, args...)
}

// WithoutOwner — действующие КЕ без владельца (cmdb.ci.owner_person_id IS NULL, E7.5). Компания-
// владелец (company_code) и персональный владелец — разные поля модели (intent §7.3): отчёт — про
// персональную ответственность, поэтому company_code не заменяет владельца, а показан для контекста.
func WithoutOwner(ctx context.Context) ([]repo.Row, error) {
	return repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, k.name AS class_name, COALESCE(c.location_code, '') AS location_code,
		       COALESCE(c.company_code, '') AS company_code, COALESCE(c.criticality, '') AS criticality, c.lifecycle_stage
		FROM cmdb.ci c JOIN ref.ci_class k ON k.class_code = c.ci_class
		WHERE `+activeFilter+` AND c.owner_person_id IS NULL
		ORDER BY c.location_code, k.name, c.name`)
}

// NamingDeviations — действующие КЕ с naming_status != compliant (E7.5); дашборд E7.4 показывает
// только доли, этот отчёт — сами КЕ, которые нужно поправить.
func NamingDeviations(ctx context.Context) ([]repo.Row, error) {
	return repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, k.name AS class_name, c.naming_status,
		       COALESCE(c.location_code, '') AS location_code, COALESCE(c.source_name, '') AS source_name
		FROM cmdb.ci c JOIN ref.ci_class k ON k.class_code = c.ci_class
		WHERE `+activeFilter+` AND c.naming_status <> 'compliant'
		ORDER BY c.naming_status, c.name`)
}

// SumTB суммирует колонку *_tb (текстом, как её возвращают QueryRows) — для итоговой строки
// таблицы ёмкости на странице отчёта; нечисловое/пустое значение пропускается, не валит отчёт.
func SumTB(rows []repo.Row, col string) float64 {
	var sum float64
	for _, r := range rows {
		var v float64
		if _, err := fmt.Sscan(r[col], &v); err == nil {
			sum += v
		}
	}
	return sum
}
