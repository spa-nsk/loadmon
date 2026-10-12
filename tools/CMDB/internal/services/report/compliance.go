// Отчёты соответствия (plan.md E7.9, intent §7.6/8: приказ ФСТЭК № 117, доля отечественной
// продукции): свежесть инвентаризации, объекты КИИ и их состав, доля отечественного оборудования и
// ПО, legacy-имена с предложением нового.
package report

import (
	"context"
	"fmt"
	"strings"

	"cmdb/internal/repo"
	"cmdb/internal/services/naming"
)

// StaleInventory — действующие КЕ, не подтверждённые источником более 30 дней (intent §9 KPI) или
// никогда не подтверждённые (last_seen_at IS NULL — обычно КЕ только ручного ввода, без источника
// вовсе; тот же смысл, что «не подтверждена», просто с самого начала, а не с какого-то момента).
func StaleInventory(ctx context.Context) ([]repo.Row, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, k.name AS class_name, COALESCE(c.location_code, '') AS location_code,
		       COALESCE(to_char(c.last_seen_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS last_seen_at,
		       CASE WHEN c.last_seen_at IS NULL THEN '' ELSE (current_date - c.last_seen_at::date)::text END AS days_stale
		FROM cmdb.ci c JOIN ref.ci_class k ON k.class_code = c.ci_class
		WHERE `+activeFilter+` AND (c.last_seen_at IS NULL OR c.last_seen_at < now() - interval '30 days')
		ORDER BY c.last_seen_at NULLS FIRST, c.name`)
	if err != nil {
		return nil, fmt.Errorf("report: свежесть инвентаризации: %w", err)
	}
	return rows, nil
}

// KIIComposition — объекты КИИ (cmdb.ci.kii_object, интент §5.7/152-ФЗ) и их состав: одна строка
// на пару «объект КИИ — связанная КЕ» (LEFT JOIN — объект без связей всё равно попадает строкой с
// пустыми полями состава, не теряется).
func KIIComposition(ctx context.Context) ([]repo.Row, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT k.id::text AS kii_id, k.name AS kii_name, COALESCE(k.kii_category, '') AS kii_category,
		       COALESCE(kc.name, k.ci_class) AS kii_class_name,
		       COALESCE(p.id::text, '') AS member_id, COALESCE(p.name, '') AS member_name,
		       COALESCE(pc.name, COALESCE(p.ci_class, '')) AS member_class_name, COALESCE(r.rel_type, '') AS rel_type
		FROM cmdb.ci k
		LEFT JOIN ref.ci_class kc ON kc.class_code = k.ci_class
		LEFT JOIN cmdb.ci_rel r ON (r.src_ci_id = k.id OR r.dst_ci_id = k.id) AND upper_inf(r.valid_period)
		LEFT JOIN cmdb.ci p ON p.id = CASE WHEN r.src_ci_id = k.id THEN r.dst_ci_id ELSE r.src_ci_id END
		LEFT JOIN ref.ci_class pc ON pc.class_code = p.ci_class
		WHERE k.kii_object AND k.lifecycle_stage NOT IN ('Retired', 'Disposed')
		ORDER BY k.name, p.name`)
	if err != nil {
		return nil, fmt.Errorf("report: объекты КИИ: %w", err)
	}
	return rows, nil
}

// DomesticHardware — модели оборудования действующих КЕ с признаком отечественного происхождения
// (ref.model.is_domestic, ссылка на реестр) и числом КЕ на каждой модели.
func DomesticHardware(ctx context.Context) ([]repo.Row, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT m.id::text AS model_id, COALESCE(mf.name, '') AS manufacturer, m.model_name,
		       CASE WHEN m.is_domestic THEN 'да' ELSE 'нет' END AS is_domestic,
		       COALESCE(m.registry_ref, '') AS registry_ref, count(*)::text AS n
		FROM cmdb.ci c JOIN ref.model m ON m.id = c.model_id LEFT JOIN ref.manufacturer mf ON mf.id = m.manufacturer_id
		WHERE `+activeFilter+`
		GROUP BY m.id, mf.name, m.model_name, m.is_domestic, m.registry_ref
		ORDER BY m.is_domestic, count(*) DESC`)
	if err != nil {
		return nil, fmt.Errorf("report: отечественное оборудование: %w", err)
	}
	return rows, nil
}

// DomesticSummary — итог «сколько всего / сколько отечественных» для заголовка отчёта.
type DomesticSummary struct {
	Total    int
	Domestic int
}

// Pct — доля отечественных; Total=0 — 0 (считать нечего, не более 0 % и не «нет данных»: для
// заголовка отчёта пустое множество естественно показать как 0 %, а не как отдельный «н/д»-кейс,
// который тут же пришлось бы отдельно проверять в шаблоне без явной пользы).
func (s DomesticSummary) Pct() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Domestic) / float64(s.Total) * 100
}

// DomesticHardwareSummary — среди действующих КЕ с указанной моделью: сколько всего и сколько на
// отечественной (модели без is_domestic=true — весь остальной парк; КЕ без модели вовсе — вне
// подсчёта, доля отечественного тут просто неизвестна, не «нет»).
func DomesticHardwareSummary(ctx context.Context) (DomesticSummary, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT count(*)::text AS total, count(*) FILTER (WHERE m.is_domestic)::text AS domestic
		FROM cmdb.ci c JOIN ref.model m ON m.id = c.model_id
		WHERE `+activeFilter)
	if err != nil {
		return DomesticSummary{}, fmt.Errorf("report: сводка отечественного оборудования: %w", err)
	}
	var s DomesticSummary
	fmt.Sscan(row["total"], &s.Total)
	fmt.Sscan(row["domestic"], &s.Domestic)
	return s, nil
}

// DomesticSoftware — установленное ПО (на действующих КЕ) с признаком отечественного происхождения
// и числом установок на каждый продукт.
func DomesticSoftware(ctx context.Context) ([]repo.Row, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT sw.id::text AS software_id, sw.vendor, sw.product, COALESCE(sw.version, '') AS version,
		       CASE WHEN sw.is_domestic THEN 'да' ELSE 'нет' END AS is_domestic,
		       COALESCE(sw.registry_ref, '') AS registry_ref, count(*)::text AS n
		FROM cmdb.ci_software_install si
		JOIN ref.software sw ON sw.id = si.software_id
		JOIN cmdb.ci c ON c.id = si.ci_id
		WHERE `+activeFilter+`
		GROUP BY sw.id, sw.vendor, sw.product, sw.version, sw.is_domestic, sw.registry_ref
		ORDER BY sw.is_domestic, count(*) DESC`)
	if err != nil {
		return nil, fmt.Errorf("report: отечественное ПО: %w", err)
	}
	return rows, nil
}

// DomesticSoftwareSummary — среди установок ПО на действующих КЕ: сколько всего и сколько отечественных.
func DomesticSoftwareSummary(ctx context.Context) (DomesticSummary, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT count(*)::text AS total, count(*) FILTER (WHERE sw.is_domestic)::text AS domestic
		FROM cmdb.ci_software_install si JOIN ref.software sw ON sw.id = si.software_id JOIN cmdb.ci c ON c.id = si.ci_id
		WHERE `+activeFilter)
	if err != nil {
		return DomesticSummary{}, fmt.Errorf("report: сводка отечественного ПО: %w", err)
	}
	var s DomesticSummary
	fmt.Sscan(row["total"], &s.Total)
	fmt.Sscan(row["domestic"], &s.Domestic)
	return s, nil
}

// LegacyNames — действующие КЕ со статусом именования legacy (профиль GUIDE-2019) и предложенным
// именем по действующему профилю CMDB-NC-2026. Предложение — только шаблон («SITE-DES-NNNNN», N —
// плейсхолдер числа разрядов), не настоящее выделенное имя: реальный номер выдаёт только
// seqalloc при фактическом переименовании (ci.Create/ChangeLifecycle), отчёт не должен тратить
// диапазон номеров просто на предпросмотр (тот же принцип, что у CIController.NamePreview для формы
// создания КЕ). "" — у класса нет designation в профиле или у КЕ нет площадки, предложить нечего.
func LegacyNames(ctx context.Context, reg *naming.Registry) ([]repo.Row, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, c.ci_class, k.name AS class_name, COALESCE(c.location_code, '') AS location_code,
		       COALESCE(l.location_code_c, '') AS site, COALESCE(c.source_name, '') AS source_name
		FROM cmdb.ci c JOIN ref.ci_class k ON k.class_code = c.ci_class
		LEFT JOIN ref.location l ON l.location_code = c.location_code
		WHERE c.naming_status = 'legacy' AND `+activeFilter+`
		ORDER BY c.name`)
	if err != nil {
		return nil, fmt.Errorf("report: legacy-имена: %w", err)
	}
	for _, r := range rows {
		suggested := ""
		if des, ok := reg.ByClass(r["ci_class"]); ok && r["site"] != "" {
			suggested = fmt.Sprintf("%s-%s-%s", r["site"], des.Code, strings.Repeat("N", des.SeqLength))
		}
		r["suggested_name"] = suggested
	}
	return rows, nil
}
