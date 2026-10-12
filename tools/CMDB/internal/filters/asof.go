package filters

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	beectx "github.com/beego/beego/v2/server/web/context"

	"cmdb/internal/platform/asof"
)

// AsOfCookie — имя cookie глобального переключателя «Показать на дату» (E6.1): выбранный момент
// действует на все страницы, пока пользователь не вернётся к текущему состоянию.
const AsOfCookie = "asof"

// AsOf переносит момент «как было на» в контекст запроса (plan.md E3.2/E6.1): параметр ?asof= (он
// сильнее и годится для ссылок) либо cookie глобального переключателя. ?asof=now — «текущее» для
// этого запроса, даже если cookie задана. Некорректное или отсутствующее значение — не ошибка:
// запрос идёт без asof (текущее состояние).
func AsOf(ctx *beectx.Context) {
	raw := ctx.Input.Query("asof")
	if raw == "" {
		raw = ctx.Input.Cookie(AsOfCookie)
	}
	if strings.EqualFold(raw, "now") {
		return
	}
	t, ok := asof.Parse(raw)
	if !ok {
		return
	}
	ctx.Request = ctx.Request.WithContext(asof.WithTime(ctx.Request.Context(), t))
}

// AsOfReadOnly запрещает изменяющие запросы, пока показано состояние на прошлую дату: прошлое
// нельзя править, а форма, отправленная «на дату», записала бы правку в настоящее по устаревшим данным
// (E6.1: визуальная пометка режима «только чтение» + защита на сервере, не только скрытые кнопки).
func AsOfReadOnly(ctx *beectx.Context) {
	if _, ok := asof.FromContext(ctx.Request.Context()); !ok {
		return
	}
	switch ctx.Request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return
	}
	if p := ctx.Request.URL.Path; p == "/login" || p == "/logout" {
		return // вход и выход не правят данные
	}
	const msg = "Показано состояние на прошлую дату — режим только чтения. Вернитесь к текущему состоянию, чтобы вносить изменения."
	ctx.Output.SetStatus(http.StatusConflict)
	if isAPIPath(ctx.Request.URL.Path) {
		ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(map[string]string{"error": msg})
		_ = ctx.Output.Body(b)
		return
	}
	if ctx.Request.Header.Get("HX-Request") == "true" {
		ctx.Output.Header("HX-Trigger", `{"toast":{"kind":"warning","text":`+strconv.QuoteToASCII(msg)+`}}`)
		ctx.Output.Header("HX-Reswap", "none")
	}
	_ = ctx.Output.Body([]byte(msg)) // WriteString писал бы прямо в ResponseWriter и потерял бы статус
}
