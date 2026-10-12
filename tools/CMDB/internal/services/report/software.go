package report

import (
	"context"
	"fmt"

	"cmdb/internal/platform/asof"
	"cmdb/internal/repo"
)

// ForbiddenSoftware — установки ПО с политикой forbidden/review на действующих КЕ (plan.md E7.7:
// «отчёт запрещённое/неразрешённое ПО на дату»). С ?asof= — состояние на дату: и КЕ (cmdb.ci_asof),
// и сами установки (UNION cmdb.*/hist.*, как internal/services/software.ListForCI).
func ForbiddenSoftware(ctx context.Context) ([]repo.Row, error) {
	ciFrom, instFrom := "cmdb.ci c", "cmdb.ci_software_install si"
	args := []any{}
	if t, ok := asof.FromContext(ctx); ok {
		ts := repo.TimeArg(t)
		ciFrom = "cmdb.ci_asof(CAST(CAST(? AS text) AS timestamptz)) c"
		instFrom = `(
			SELECT ci_id, software_id, version, install_path FROM cmdb.ci_software_install WHERE sys_period @> CAST(CAST(? AS text) AS timestamptz)
			UNION ALL
			SELECT ci_id, software_id, version, install_path FROM hist.ci_software_install WHERE sys_period @> CAST(CAST(? AS text) AS timestamptz)
		) si`
		args = []any{ts, ts, ts}
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT c.id::text AS id, c.name, COALESCE(c.location_code, '') AS location_code,
		       sw.vendor, sw.product, COALESCE(si.version, sw.version, '') AS version, sw.policy,
		       COALESCE(si.install_path, '') AS install_path
		FROM `+instFrom+`
		JOIN `+ciFrom+` ON c.id = si.ci_id
		JOIN ref.software sw ON sw.id = si.software_id
		WHERE sw.policy IN ('forbidden', 'review') AND c.lifecycle_stage NOT IN ('Retired', 'Disposed')
		ORDER BY sw.policy, c.name`, args...)
	if err != nil {
		return nil, fmt.Errorf("report: запрещённое/неразрешённое ПО: %w", err)
	}
	return rows, nil
}
