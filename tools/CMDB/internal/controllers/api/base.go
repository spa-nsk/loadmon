// Package api — REST API /api/v1 (plan.md E7.3): КЕ, компоненты, связи, справочники, история,
// срезы. Аутентификация — сессия (cookie) или токен API (заголовок Authorization: Bearer …,
// internal/services/auth.AuthenticateToken), права — тот же RBAC-фильтр, что у UI
// (internal/filters.RBAC), «показать на дату» — тот же ?asof=. Ответы всегда JSON, ошибки сервисов
// превращаются в HTTP-статус по тому же соответствию, что и у UI (internal/controllers/ui.friendly),
// без HTML/HTMX-специфики. Все записи идут через changectx с source=api (Input.Source), не manual —
// чтобы в журнале изменений (E3.7) и в audit.change_set было видно, что правку внесла интеграция,
// а не человек за формой.
package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/services/auth"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/svcerr"
)

// BaseController — общее поведение контроллеров /api/v1.
type BaseController struct {
	web.Controller
}

// Prepare отключает проверку XSRF-токена beego (иначе на любой POST/PATCH без поля _xsrf в форме
// beego отвечал бы 422 "'_xsrf' argument missing" — токен рассчитан на HTML-формы, не на клиентов
// REST API). Тело запросов API — всегда JSON (Content-Type: application/json), а не
// application/x-www-form-urlencoded, поэтому это не «простой» межсайтовый запрос: браузер шлёт
// его только после успешного CORS preflight, а CORS здесь не настроен — межсайтовая форма или
// fetch с чужого источника до контроллера просто не дойдёт, даже если у пользователя открыта
// сессия UI. Токен API (Authorization: Bearer) браузер тем более не приложит сам ни к одному
// чужому запросу.
func (c *BaseController) Prepare() {
	c.EnableXSRF = false
}

func (c *BaseController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *BaseController) identity() auth.Identity {
	id, _ := auth.IdentityFromContext(c.ctx())
	return id
}

func (c *BaseController) userID() int64 { return c.identity().ID }

// id — числовой параметр маршрута ":id" (0, если не задан/не число — сервис ответит «не найдено»).
func (c *BaseController) id() int64 {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	return id
}

// writeJSON отдаёт v как JSON с заданным статусом (200 — как есть, autorender у beego выключается
// сам, раз статус/тело уже отправлены — см. internal/controllers/ui.CIController.GridData).
func (c *BaseController) writeJSON(status int, v any) {
	if status != http.StatusOK {
		c.Ctx.Output.SetStatus(status)
	}
	c.Data["json"] = v
	_ = c.ServeJSON()
}

type errorBody struct {
	Error string `json:"error"`
}

// fail превращает ошибку сервиса в JSON {"error": "..."} с HTTP-статусом по её типу — тем же
// соответствием, что internal/controllers/ui.friendly использует для тоста, но без текста для
// человека и без HTML: 404/409/422/500. Прерывает обработчик (как ui.BaseController.fail).
func (c *BaseController) fail(err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, svcerr.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, svcerr.ErrVersionConflict), errors.Is(err, svcerr.ErrDuplicate):
		status = http.StatusConflict
	case errors.Is(err, svcerr.ErrInvalidInput), errors.Is(err, svcerr.ErrInvalidState),
		errors.Is(err, svcerr.ErrNoChanges), errors.Is(err, ci.ErrInvalidTransition), errors.Is(err, svcerr.ErrPeriodClosed):
		status = http.StatusUnprocessableEntity
	}
	c.writeJSON(status, errorBody{Error: err.Error()})
	c.StopRun()
}

// must — fail(err), если err != nil.
func (c *BaseController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

// badRequest отдаёт 400 с готовым текстом (некорректный запрос, ещё не дошедший до сервиса: не
// разбирается JSON, не хватает обязательного поля и т.п.).
func (c *BaseController) badRequest(text string) {
	c.writeJSON(http.StatusBadRequest, errorBody{Error: text})
	c.StopRun()
}

// requireCIWrite — как ui.CIController.requireCIWrite: право ci.write есть по RBAC-фильтру
// (иначе запрос не дошёл бы до контроллера), но у роли class_owner оно ограничено классами —
// класс берётся из БД, не из тела запроса.
func (c *BaseController) requireCIWrite(class string) {
	if class == "" {
		return // КЕ не найдена — сервис сам ответит 404
	}
	if id := c.identity(); !id.CanWriteClass(class) {
		c.writeJSON(http.StatusForbidden, errorBody{Error: "недостаточно прав: правка КЕ класса «" + class + "» недоступна"})
		c.StopRun()
	}
}

// decodeJSON разбирает тело запроса в v; при ошибке отвечает 400 и прерывает обработчик.
func (c *BaseController) decodeJSON(v any) {
	if err := c.BindJSON(v); err != nil {
		c.badRequest("некорректное тело запроса: " + err.Error())
	}
}
