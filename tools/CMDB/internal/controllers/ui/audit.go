package ui

import (
	"context"
	"encoding/csv"
	"net/url"
	"strconv"
	"time"

	"cmdb/internal/services/history"
)

// AuditController — глобальный журнал изменений с фильтрами (plan.md E3.7): в отличие от вкладки
// «История» карточки КЕ (internal/controllers/ui/ci.go), не привязан к одной КЕ.
type AuditController struct {
	BaseController
}

func (c *AuditController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *AuditController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

var sourceKinds = []string{"manual", "import", "api", "system", "migration"}

func (c *AuditController) filter() history.GlobalFilter {
	f := history.GlobalFilter{
		Source: c.GetString("source"), Entity: c.GetString("entity"), Q: c.GetString("q"),
		From: c.GetString("from"), To: c.GetString("to"),
	}
	f.Page, _ = c.GetInt("page", 1)
	f.PerPage, _ = c.GetInt("per", 25)
	return f
}

func auditQueryValues(f history.GlobalFilter) url.Values {
	v := url.Values{}
	set := func(k, val string) {
		if val != "" {
			v.Set(k, val)
		}
	}
	set("source", f.Source)
	set("entity", f.Entity)
	set("q", f.Q)
	set("from", f.From)
	set("to", f.To)
	return v
}

// List — страница глобального журнала; на запрос HTMX — только таблица с пагинацией.
func (c *AuditController) List() {
	f := c.filter()
	res, err := history.GlobalLog(c.ctx(), f)
	c.must(err)

	c.Data["Filter"] = f
	c.Data["Result"] = res
	c.Data["PageURL"] = func(p int) string {
		v := auditQueryValues(f)
		v.Set("page", strconv.Itoa(p))
		return "/changes?" + v.Encode()
	}
	var nums []int
	for p := max(1, res.Page-3); p <= min(res.Pages, res.Page+3); p++ {
		nums = append(nums, p)
	}
	c.Data["PageNums"] = nums

	if c.isHTMX() {
		c.fragment("audit/_results.tpl")
		return
	}
	c.Data["Nav"] = "changes"
	entities, err := history.DistinctEntities(c.ctx())
	c.must(err)
	c.Data["Entities"] = entities
	c.Data["Sources"] = sourceKinds
	c.page("audit/list.tpl")
}

// changesExportColumns — колонки выгрузки, в порядке отображения.
var changesExportColumns = []string{"Момент (UTC)", "Источник", "Пользователь", "Причина", "Заявка", "Записей", "Сущности"}

// ExportCSV — выгрузка журнала изменений за период по текущему фильтру (E7.5: «изменения за
// период»), без пагинации — весь диапазон одним файлом.
func (c *AuditController) ExportCSV() {
	f := c.filter()
	f.Page, f.PerPage = 1, 1_000_000
	res, err := history.GlobalLog(c.ctx(), f)
	c.must(err)
	c.logEvent("changes_export", map[string]any{"rows": len(res.Rows), "from": f.From, "to": f.To})

	c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", `attachment; filename="changes-`+time.Now().UTC().Format("20060102-1504")+`.csv"`)
	w := csv.NewWriter(c.Ctx.ResponseWriter)
	_ = w.Write(changesExportColumns)
	for _, row := range res.Rows {
		rec := []string{row.At, row.Source, row.UserID, row.Reason, row.RFC, row.Count, row.Entities}
		_ = w.Write(rec)
	}
	w.Flush()
	c.StopRun()
}

// Detail — фрагмент с полным содержимым одного набора изменений (раскрытие строки журнала).
func (c *AuditController) Detail() {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	cs, err := history.ForChangeSet(c.ctx(), id)
	c.must(err)
	c.Data["CS"] = cs
	c.fragment("audit/_detail.tpl")
}
