package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/platform/asof"
	"cmdb/internal/platform/logging"
	"cmdb/internal/services/auditlog"
	"cmdb/internal/services/auth"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/svcerr"
)

var registry = naming.DefaultRegistry()

// BaseController — общее поведение страниц: данные шаблона layout, ответы HTMX (тосты, ошибки
// полей), разбор ошибок сервисов в понятные пользователю сообщения.
type BaseController struct {
	web.Controller
}

// Prepare задаёт данные layout и выдаёт XSRF-токен (куки + значение для заголовка HTMX).
func (c *BaseController) Prepare() {
	c.Data["Perms"] = map[string]bool{} // layout читает права через index; у неавторизованного их нет
	// Пользователь запроса (фильтр internal/filters.Auth): имя в шапке и права для скрытия недоступных пунктов меню.
	if id, ok := auth.IdentityFromContext(c.Ctx.Request.Context()); ok {
		name := id.DisplayName
		if name == "" {
			name = id.Username
		}
		c.Data["User"] = name
		c.Data["UserRoles"] = strings.Join(id.RoleNames(), ", ")
		c.Data["LoggedIn"] = true
		perms := map[string]bool{}
		for _, p := range []string{auth.CIRead, auth.CIWrite, auth.RefWrite, auth.ImportRead, auth.ImportRun, auth.AuditRead,
			auth.BaselineRead, auth.BaselineWrite, auth.PeriodManage, auth.UserAdmin} {
			perms[p] = id.Can(p)
		}
		c.Data["Perms"] = perms
	}
	c.Data["Xsrf"] = c.XSRFToken()
	c.Data["Nav"] = ""
	// «Показать на дату» (E6.1): режим «только чтение» для всех страниц.
	if t, ok := asof.FromContext(c.Ctx.Request.Context()); ok {
		c.Data["AsOf"] = t.UTC().Format("2006-01-02 15:04")
		c.Data["AsOfInput"] = t.UTC().Format("2006-01-02T15:04")
	}
}

// userID — пользователь запроса; 0 у неаутентифицированного (такие запросы до контроллеров не доходят).
func (c *BaseController) userID() int64 {
	if id, ok := auth.IdentityFromContext(c.Ctx.Request.Context()); ok {
		return id.ID
	}
	return 0
}

// requireWriteClass прерывает запрос, если у пользователя нет права править КЕ этого класса
// (роль class_owner ограничена классами; фильтр RBAC проверяет только наличие права ci.write).
func (c *BaseController) requireWriteClass(class string) {
	if id, ok := auth.IdentityFromContext(c.Ctx.Request.Context()); ok && !id.CanWriteClass(class) {
		c.Ctx.Output.SetStatus(403)
		c.failText("Недостаточно прав: правка КЕ класса «" + class + "» вам недоступна.")
	}
}

// logEvent пишет audit.event (E3.2/E7.6: просмотры, экспорт) от имени текущего пользователя.
// Ошибка не прерывает запрос и не долетает до пользователя тостом — сам просмотр/экспорт уже
// состоялся и откатывать его из-за сбоя аудита не нужно; ошибка только логируется в slog.
func (c *BaseController) logEvent(eventType string, details map[string]any) {
	if err := auditlog.Log(c.Ctx.Request.Context(), c.userID(), c.Ctx.Input.IP(), eventType, details); err != nil {
		logging.FromContext(c.Ctx.Request.Context()).Warn("audit_event_failed", "event_type", eventType, "error", err)
	}
}

// isHTMX — запрос сделан HTMX (нужен фрагмент страницы, а не полная страница).
func (c *BaseController) isHTMX() bool {
	return c.Ctx.Request.Header.Get("HX-Request") == "true"
}

// page рендерит полную страницу в layout.
func (c *BaseController) page(tpl string) {
	c.Layout = "layout.tpl"
	c.TplName = tpl
}

// fragment рендерит фрагмент без layout.
func (c *BaseController) fragment(tpl string) {
	c.Layout = ""
	c.TplName = tpl
}

// notFoundPage показывает страницу error.tpl со статусом 404 — устаревшая ссылка (id/код в адресе
// не существует), а не опечатка в форме, поэтому страница, не тост (используют CIController.Card,
// RefAdminController.Edit, BaselineConfigController.Edit, BaselineController.notFound,
// ImportController.RunDetail и т.п. — одна и та же ситуация у разных сущностей). Названа не
// notFound: у BaselineController уже есть свой notFound(err) bool с другой сигнатурой (решает,
// прерывать ли вызывающий метод), который теперь вызывает этот. Рендерит тело сам, тем же приёмом,
// что и AuthController.renderLogin: если
// статус выставлен заранее через Output.SetStatus, beego пропускает автоrender (условие в его
// роутере — «рендерить, только если Output.Status ещё 0»), поэтому HTML нужно получить
// RenderString и записать явно через Output.Body — тот же вызов заодно помечает ответ начатым
// (ResponseWriter.Started), так что последующий автоrender не перезапишет его повторно; отдельного
// StopRun() не нужно, если это последняя строка обработчика (как и в вызывающих методах). Ошибка
// самого рендера — не повод остаться совсем без ответа, отдаём как есть текстом.
func (c *BaseController) notFoundPage(msg string) {
	c.Data["Message"] = msg
	c.page("error.tpl")
	body, err := c.RenderString()
	if err != nil {
		logging.FromContext(c.Ctx.Request.Context()).Error("notfound_render_failed", "error", err)
		body = msg
	}
	c.Ctx.Output.SetStatus(404)
	_ = c.Ctx.Output.Body([]byte(body))
}

// toastPayload — событие «toast» для static/js/app.js.
type toastPayload struct {
	Kind string `json:"kind"` // success | danger | warning | info
	Text string `json:"text"`
}

// trigger добавляет заголовок HX-Trigger с событиями. Значение ASCII-экранировано: заголовки
// HTTP с не-ASCII символами браузер читает как Latin-1 и получается мусор.
func (c *BaseController) trigger(events map[string]any) {
	b, err := json.Marshal(events)
	if err != nil {
		return
	}
	c.Ctx.Output.Header("HX-Trigger", asciiEscape(string(b)))
}

// ok показывает тост об успехе (страница/фрагмент рендерится как обычно).
func (c *BaseController) ok(text string) {
	c.trigger(map[string]any{"toast": toastPayload{Kind: "success", Text: text}})
}

// fail сообщает об ошибке тостом и не меняет страницу (HX-Reswap: none) — введённые в форму
// значения остаются на месте. fieldErrors подсвечивает поля формы.
func (c *BaseController) fail(err error, fieldErrors ...map[string]string) {
	events := map[string]any{"toast": toastPayload{Kind: "danger", Text: friendly(err)}}
	if len(fieldErrors) > 0 && len(fieldErrors[0]) > 0 {
		events["fieldErrors"] = fieldErrors[0]
	}
	c.trigger(events)
	c.Ctx.Output.Header("HX-Reswap", "none")
	if c.Ctx.Output.Status != 0 {
		_ = c.Ctx.Output.Body([]byte{}) // WriteString потерял бы заранее выставленный статус (403 и т.п.)
	} else {
		c.Ctx.WriteString("")
	}
	c.StopRun()
}

// failText — fail с готовым текстом.
func (c *BaseController) failText(text string, fieldErrors ...map[string]string) {
	c.fail(errors.New(text), fieldErrors...)
}

// redirect перенаправляет: для HTMX — заголовком HX-Redirect, иначе обычным редиректом.
func (c *BaseController) redirect(url string) {
	if c.isHTMX() {
		c.Ctx.Output.Header("HX-Redirect", url)
		c.Ctx.WriteString("")
		c.StopRun()
		return
	}
	c.Redirect(url, 302)
	c.StopRun()
}

// friendly превращает ошибку сервиса в сообщение для пользователя.
func friendly(err error) string {
	switch {
	case errors.Is(err, svcerr.ErrVersionConflict):
		return "Запись изменена другим пользователем. Обновите вкладку и повторите правку."
	case errors.Is(err, svcerr.ErrNotFound):
		return "Не найдено: " + trimSentinel(err, svcerr.ErrNotFound)
	case errors.Is(err, svcerr.ErrDuplicate):
		return "Дубликат: " + trimSentinel(err, svcerr.ErrDuplicate)
	case errors.Is(err, ci.ErrInvalidTransition):
		return "Такой переход стадии жизненного цикла не разрешён."
	case errors.Is(err, svcerr.ErrInvalidState), errors.Is(err, svcerr.ErrInvalidInput), errors.Is(err, svcerr.ErrNoChanges):
		return err.Error()
	}
	return err.Error()
}

// trimSentinel убирает из текста ошибки префикс самой sentinel-ошибки («запись не найдена: ci id=5» → «ci id=5»).
func trimSentinel(err, sentinel error) string {
	return strings.TrimPrefix(strings.TrimPrefix(err.Error(), sentinel.Error()), ": ")
}

// reason возвращает обязательную причину изменения из формы или прерывает запрос с ошибкой.
func (c *BaseController) reason() string {
	r := strings.TrimSpace(c.GetString("reason"))
	if r == "" {
		c.failText("Укажите причину изменения.", map[string]string{"reason": "Обязательное поле"})
	}
	return r
}

// asciiEscape заменяет все не-ASCII символы JSON-строки на \uXXXX (с суррогатными парами).
func asciiEscape(s string) string {
	var sb strings.Builder
	for _, r := range s {
		switch {
		case r <= 126:
			sb.WriteRune(r)
		case r > 0xFFFF:
			r1, r2 := (r-0x10000)>>10+0xD800, (r-0x10000)&0x3FF+0xDC00
			fmt.Fprintf(&sb, `\u%04x\u%04x`, r1, r2)
		default:
			fmt.Fprintf(&sb, `\u%04x`, r)
		}
	}
	return sb.String()
}
