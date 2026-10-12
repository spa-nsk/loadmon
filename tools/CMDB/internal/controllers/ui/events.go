package ui

import (
	"context"
	"encoding/csv"
	"net/url"
	"strconv"
	"time"

	"cmdb/internal/services/auditlog"
	"cmdb/internal/services/auth"
)

// EventsController — журнал действий привилегированных пользователей (plan.md E7.6, audit.event):
// входы, экспорт, изменения ролей/справочников (правил, профиля именования), закрытие периода.
// В отличие от AuditController (audit.change_set/change_log — что изменилось в данных), здесь —
// что сделал пользователь с самой системой. Просмотр секретов в список не входит: секреты
// коннекторов (internal/importer/connector.Config.Secret) нигде не показываются пользователю —
// хранится и читается только имя переменной окружения, самого значения в приложении не видно
// никому, значит, и «просмотр» как действие невозможен в принципе.
type EventsController struct {
	BaseController
}

func (c *EventsController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *EventsController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

func (c *EventsController) filter() auditlog.Filter {
	f := auditlog.Filter{EventType: c.GetString("event_type"), Q: c.GetString("q"), From: c.GetString("from"), To: c.GetString("to")}
	f.UserID, _ = c.GetInt64("user_id")
	f.Page, _ = c.GetInt("page", 1)
	f.PerPage, _ = c.GetInt("per", 50)
	return f
}

func eventsQueryValues(f auditlog.Filter) url.Values {
	v := url.Values{}
	set := func(k, val string) {
		if val != "" {
			v.Set(k, val)
		}
	}
	set("event_type", f.EventType)
	set("q", f.Q)
	set("from", f.From)
	set("to", f.To)
	if f.UserID > 0 {
		v.Set("user_id", strconv.FormatInt(f.UserID, 10))
	}
	return v
}

// List — страница журнала действий; на запрос HTMX — только таблица с пагинацией.
func (c *EventsController) List() {
	f := c.filter()
	res, err := auditlog.List(c.ctx(), f)
	c.must(err)

	c.Data["Filter"] = f
	c.Data["Result"] = res
	c.Data["PageURL"] = func(p int) string {
		v := eventsQueryValues(f)
		v.Set("page", strconv.Itoa(p))
		return "/admin/events?" + v.Encode()
	}
	var nums []int
	for p := max(1, res.Page-3); p <= min(res.Pages, res.Page+3); p++ {
		nums = append(nums, p)
	}
	c.Data["PageNums"] = nums

	if c.isHTMX() {
		c.fragment("admin/_events_results.tpl")
		return
	}
	c.Data["Nav"] = "events"
	types, err := auditlog.DistinctEventTypes(c.ctx())
	c.must(err)
	c.Data["EventTypes"] = types
	users, err := auth.ListUsers(c.ctx())
	c.must(err)
	c.Data["Users"] = users
	c.page("admin/events.tpl")
}

// allRows — все строки по текущему фильтру, без пагинации (для выгрузок).
func (c *EventsController) allRows() []auditlog.Row {
	f := c.filter()
	f.Page, f.PerPage = 1, 1_000_000
	res, err := auditlog.List(c.ctx(), f)
	c.must(err)
	return res.Rows
}

// eventsExportColumns — колонки CSV-выгрузки, в порядке отображения.
var eventsExportColumns = []string{"Момент (UTC)", "Тип события", "Пользователь", "IP", "Детали"}

// ExportCSV — выгрузка журнала действий по текущему фильтру в CSV (для анализа в таблицах).
func (c *EventsController) ExportCSV() {
	rows := c.allRows()
	c.logEvent("audit_event_export", map[string]any{"format": "csv", "rows": len(rows)})

	c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", `attachment; filename="audit-events-`+time.Now().UTC().Format("20060102-1504")+`.csv"`)
	w := csv.NewWriter(c.Ctx.ResponseWriter)
	_ = w.Write(eventsExportColumns)
	for _, r := range rows {
		user := r.Username
		if user == "" {
			user = r.UserID
		}
		_ = w.Write([]string{r.At, r.EventType, user, r.ClientIP, r.Details})
	}
	w.Flush()
	c.StopRun()
}

// ExportCEF — выгрузка в формате CEF/syslog для приёма SIEM (plan.md E7.6).
func (c *EventsController) ExportCEF() {
	rows := c.allRows()
	c.logEvent("audit_event_export", map[string]any{"format": "cef", "rows": len(rows)})

	c.Ctx.Output.Header("Content-Type", "text/plain; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", `attachment; filename="audit-events-`+time.Now().UTC().Format("20060102-1504")+`.cef.log"`)
	for _, r := range rows {
		_, _ = c.Ctx.ResponseWriter.Write([]byte(auditlog.FormatCEF(r) + "\n"))
	}
	c.StopRun()
}
