package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/services/baseline"
	"cmdb/internal/services/period"
	"cmdb/internal/services/svcerr"
)

// BaselineController — именованные срезы и сравнение (plan.md E6.2–E6.4).
type BaselineController struct {
	BaseController
}

func (c *BaselineController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *BaselineController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

func (c *BaselineController) id() int64 {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	return id
}

func (c *BaselineController) notFound(err error) bool {
	if err == nil {
		return false
	}
	if isNotFound(err) {
		c.notFoundPage("Срез не найден")
		return true
	}
	c.fail(err)
	return true
}

func isNotFound(err error) bool { return strings.Contains(err.Error(), svcerr.ErrNotFound.Error()) }

// parseUTC разбирает значение поля datetime-local / дату как UTC; "" — ok=false.
func parseUTC(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// List — список срезов и форма создания.
func (c *BaselineController) List() {
	list, err := baseline.List(c.ctx())
	c.must(err)
	c.Data["Nav"] = "baselines"
	c.Data["Baselines"] = list
	c.Data["NowInput"] = time.Now().UTC().Format("2006-01-02T15:04")
	c.page("baselines/list.tpl")
}

// Create создаёт срез-черновик на момент as_of (пусто — сейчас) по фильтру области.
func (c *BaselineController) Create() {
	asOf := time.Now().UTC()
	if v := strings.TrimSpace(c.GetString("as_of")); v != "" {
		t, ok := parseUTC(v)
		if !ok {
			c.failText("Момент среза: ожидается дата и время (UTC).", map[string]string{"as_of": "Некорректная дата"})
		}
		asOf = t
	}
	sc := baseline.Scope{LocationCode: strings.TrimSpace(c.GetString("location_code")), CompanyCode: strings.TrimSpace(c.GetString("company_code")),
		EnvironmentCode: strings.TrimSpace(c.GetString("environment_code")), NameLike: strings.TrimSpace(c.GetString("name_like"))}
	for _, cl := range strings.Split(c.GetString("classes"), ",") {
		if cl = strings.TrimSpace(cl); cl != "" {
			sc.Classes = append(sc.Classes, cl)
		}
	}
	id, err := baseline.Create(c.ctx(), c.GetString("name"), c.GetString("description"), asOf, sc, c.userID())
	c.must(err)
	c.ok("Срез создан (черновик).")
	c.redirect("/baselines/" + strconv.FormatInt(id, 10))
}

// Show — карточка среза: паспорт, состав по сущностям, действия, форма сравнения.
func (c *BaselineController) Show() {
	b, err := baseline.Get(c.ctx(), c.id())
	if c.notFound(err) {
		return
	}
	counts, err := baseline.Count(c.ctx(), c.id())
	c.must(err)
	others, err := baseline.List(c.ctx())
	c.must(err)
	type cnt struct {
		Entity string
		N      int
	}
	var byEntity []cnt
	for _, e := range baseline.Entities() {
		if n := counts[e]; n > 0 {
			byEntity = append(byEntity, cnt{e, n})
		}
	}
	c.Data["Nav"] = "baselines"
	c.Data["B"] = b
	c.Data["Scope"] = b.Scope()
	c.Data["ByEntity"] = byEntity
	c.Data["Others"] = others
	c.Data["Formats"] = baseline.Formats
	c.page("baselines/show.tpl")
}

// Freeze замораживает черновик: считает контрольную сумму, дальше срез неизменяем.
func (c *BaselineController) Freeze() {
	c.must(baseline.Freeze(c.ctx(), c.id(), c.userID()))
	c.ok("Срез заморожен.")
	c.redirect("/baselines/" + strconv.FormatInt(c.id(), 10))
}

// Approve утверждает замороженный срез.
func (c *BaselineController) Approve() {
	c.must(baseline.Approve(c.ctx(), c.id(), c.userID()))
	c.ok("Срез утверждён.")
	c.redirect("/baselines/" + strconv.FormatInt(c.id(), 10))
}

// Delete удаляет черновик.
func (c *BaselineController) Delete() {
	c.must(baseline.Delete(c.ctx(), c.id(), c.userID()))
	c.ok("Черновик удалён.")
	c.redirect("/baselines")
}

// Verify перепроверяет контрольную сумму и версии строк среза; результат — тостом.
func (c *BaselineController) Verify() {
	res, err := baseline.Verify(c.ctx(), c.id())
	c.must(err)
	c.logEvent("baseline_verify", map[string]any{"baseline_id": c.id(), "ok": res.OK})
	if res.OK {
		c.ok("Срез цел: контрольная сумма и содержимое всех версий совпадают.")
	} else {
		c.trigger(map[string]any{"toast": toastPayload{Kind: "danger", Text: "Целостность нарушена: " + strings.Join(res.Problems, "; ")}})
	}
	c.Ctx.WriteString("")
	c.StopRun()
}

// Export отдаёт срез с манифестом (csv | xlsx | json). Скачивание пишется в audit.event.
func (c *BaselineController) Export() {
	format := c.Ctx.Input.Param(":fmt")
	types := map[string]string{"csv": "text/csv; charset=utf-8", "json": "application/json; charset=utf-8",
		"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}
	ct, ok := types[format]
	if !ok {
		c.failText("Неизвестный формат выгрузки.")
	}
	// сначала проверки состояния (черновик, несуществующий срез) — до отправки заголовков вложения
	b, err := baseline.Get(c.ctx(), c.id())
	c.must(err)
	if b.Status == "draft" {
		c.failText("Черновик выгружать нельзя — заморозьте срез.")
	}
	c.Ctx.Output.Header("Content-Type", ct)
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="baseline-%s.%s"`, b.ID, format))
	m, err := baseline.Export(c.ctx(), c.id(), format, c.Ctx.ResponseWriter)
	if err != nil {
		c.Ctx.ResponseWriter.Header().Set("HX-Trigger", "") // заголовки уже могли уйти; ошибка — в лог
		c.StopRun()
	}
	c.logEvent("baseline_export", map[string]any{"baseline_id": c.id(), "format": format, "checksum": m.Checksum, "items": m.ItemCount})
	c.StopRun()
}

// Compare — сравнение двух сторон (срез b:<id>, момент t:<UTC>, текущее now); format=csv — отчёт файлом.
func (c *BaselineController) Compare() {
	a, err := baseline.ParseState(c.GetString("a"))
	c.must(err)
	b, err := baseline.ParseState(c.GetString("b"))
	c.must(err)
	sc := baseline.Scope{LocationCode: c.GetString("location_code"), CompanyCode: c.GetString("company_code"),
		EnvironmentCode: c.GetString("environment_code")}
	if cl := strings.TrimSpace(c.GetString("classes")); cl != "" {
		for _, x := range strings.Split(cl, ",") {
			sc.Classes = append(sc.Classes, strings.TrimSpace(x))
		}
	}
	res, err := baseline.Compare(c.ctx(), a, b, sc)
	if c.notFound(err) {
		return
	}
	c.logEvent("baseline_compare", map[string]any{"a": c.GetString("a"), "b": c.GetString("b")})

	if c.GetString("format") == "csv" {
		c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
		c.Ctx.Output.Header("Content-Disposition", `attachment; filename="compare.csv"`)
		_ = baseline.CompareCSV(res, c.Ctx.ResponseWriter)
		c.StopRun()
	}
	c.Data["Nav"] = "baselines"
	c.Data["Res"] = res
	c.Data["A"], c.Data["B"] = c.GetString("a"), c.GetString("b")
	c.Data["Scope"] = sc
	c.page("baselines/compare.tpl")
}

// PeriodController — учётные периоды и отчёт о правках задним числом (plan.md E6.5).
type PeriodController struct {
	BaseController
}

func (c *PeriodController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *PeriodController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

// List — периоды, форма закрытия/переоткрытия, отчёт о правках задним числом за последние 30 суток
// (или за from/to).
func (c *PeriodController) List() {
	list, err := period.List(c.ctx())
	c.must(err)
	to := time.Now().UTC().Add(time.Minute)
	from := to.AddDate(0, 0, -30)
	if t, ok := parseUTC(c.GetString("from")); ok {
		from = t
	}
	if t, ok := parseUTC(c.GetString("to")); ok {
		to = t.AddDate(0, 0, 1)
	}
	rep, err := period.BackdatedReport(c.ctx(), from, to)
	c.must(err)
	c.Data["Nav"] = "periods"
	c.Data["Periods"] = list
	c.Data["Report"] = rep
	c.Data["From"], c.Data["To"] = from.Format("2006-01-02"), to.AddDate(0, 0, -1).Format("2006-01-02")
	c.Data["LastMonth"] = period.MonthStart(time.Now().UTC().AddDate(0, -1, 0)).Format("2006-01")
	c.page("periods/list.tpl")
}

// Close закрывает завершившийся месяц. Роль «Администратор учёта» появится с E7 — пока действие
// доступно единственному пользователю, но пишется в audit.event.
func (c *PeriodController) Close() {
	m, err := period.ParseMonth(c.GetString("month"))
	c.must(err)
	c.must(period.Close(c.ctx(), m, c.userID(), strings.TrimSpace(c.GetString("reason"))))
	c.ok("Период " + m.Format("2006-01") + " закрыт.")
	c.redirect("/periods")
}

// Reopen переоткрывает период; причина обязательна.
func (c *PeriodController) Reopen() {
	m, err := period.ParseMonth(c.GetString("month"))
	c.must(err)
	c.must(period.Reopen(c.ctx(), m, c.userID(), c.GetString("reason")))
	c.ok("Период " + m.Format("2006-01") + " переоткрыт.")
	c.redirect("/periods")
}
