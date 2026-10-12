package ui

import (
	"cmdb/internal/platform/logging"
	"cmdb/internal/services/quality"
)

// QualityController — дашборд качества данных (plan.md E7.4; целевые показатели — intent.md §9):
// полнота обязательных атрибутов, соответствие профилю именования, устаревание, дубликаты,
// открытые конфликты импорта и динамика по дням.
type QualityController struct {
	BaseController
}

func (c *QualityController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

// Index считает показатели заново на каждый заход (агрегаты по действующим КЕ — быстро даже на
// объёме из intent §9) и попутно обновляет сегодняшний снимок (rpt.quality_snapshot), чтобы график
// динамики не ждал ночного пересчёта (cmd/cmdb-web) — ошибка записи снимка не портит страницу,
// показанные числа и так свежие.
func (c *QualityController) Index() {
	ctx := c.Ctx.Request.Context()
	snap, err := quality.Compute(ctx)
	c.must(err)
	if err := quality.Save(ctx, snap); err != nil {
		logging.FromContext(ctx).Warn("quality_snapshot_save_failed", "error", err)
	}
	history, err := quality.History(ctx, 30)
	c.must(err)

	c.Data["Nav"] = "quality"
	c.Data["Snap"] = snap
	c.Data["History"] = history
	if pct, ok := snap.AttrsCompletePct(); ok {
		c.Data["AttrsPct"] = pct
		c.Data["AttrsPctOK"] = true
	}
	if pct, ok := snap.Staleness.Confirmed7dPct(); ok {
		c.Data["ConfirmedPct"] = pct
		c.Data["ConfirmedPctOK"] = true
	}
	c.Data["DuplicatesCI"] = snap.DuplicatesCI()
	c.page("quality/index.tpl")
}
