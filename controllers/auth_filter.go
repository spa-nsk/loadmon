package controllers

import (
	"context"
	"net/url"
	"strings"

	"loadmon/config"

	beecontext "github.com/beego/beego/v2/server/web/context"
)

// publicPaths — маршруты, доступные без активной сессии ("loadmon"): страницы
// самой аутентификации (вход/регистрация/восстановление пароля), служебные
// технические эндпоинты (favicon, метрики Prometheus, pprof, SSO-хендшейк,
// shutdown) и корень "/", у которого есть собственная, отличающаяся от общей
// схемы, логика редиректа (см. MainController.Get()). Точное совпадение пути.
var publicExactPaths = map[string]bool{
	"/":              true,
	"/login":         true, // алиас входа; при domain сам редиректит на "/", без сессии
	"/user/logout":   true,
	"/user/register": true,
	"/user/forgot":   true,
	"/notice":        true,
	"/favicon.ico":   true,
	"/sso":           true,
	"/shutdown":      true,
	"/metrics":       true,
}

// publicPrefixPaths — маршруты с динамическими параметрами в пути (:back,
// :uuid и т.п.), которые нельзя перечислить точными строками, плюс
// осознанные исключения:
//   - "/api/" — этот маршрут и сегодня не защищён сессией (проверка в
//     controllers/api.go закомментирована) - сохраняем как есть, включение
//     авторизации для внешнего API - отдельное решение, не часть этой правки.
//   - "/sprsrv/itsystemcorrectpreview/" — AJAX-эндпоинт, отвечающий JSON
//     (в т.ч. `{"found":false,"error":"не авторизован"}` при отсутствии
//     сессии), а не HTTP-редиректом - общий фильтр с редиректом сломал бы
//     клиентский JS, ожидающий JSON; проверка сессии остаётся внутри
//     controllers/sprsrvupdatebase.go:SprITSystemCorrectPreview.
//   - "/sprsrv/vmplacementtext/" — ТОТ ЖЕ случай (AJAX-эндпоинт, отвечает
//     JSON), добавлен 2026-09-18 вместе с выносом "живого" gRPC-запроса
//     определения размещения VM из синхронного рендера /sprsrv/updatebase
//     в отдельный асинхронный вызов (см. controllers/sprsrvupdatebase.go:
//     SprSrvVMPlacementTextAsync) - проверка сессии аналогично осталась
//     внутри самого хендлера.
var publicPrefixPaths = []string{
	"/user/login/",
	"/user/verify/",
	"/user/reset/",
	"/debug/pprof",
	"/api/",
	"/sprsrv/itsystemcorrectpreview/",
	"/sprsrv/vmplacementtext/",
}

func isPublicPath(path string) bool {
	if publicExactPaths[path] {
		return true
	}
	for _, prefix := range publicPrefixPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// AuthFilter — централизованная проверка сессии пользователя ("loadmon"),
// регистрируется один раз как глобальный BeforeRouter-фильтр (см.
// routers/routers.go, beego.InsertFilter("*", beego.BeforeRouter,
// controllers.AuthFilter)) и заменяет собой ранее продублированный в ~40
// хендлерах паттерн:
//
//	sess := this.GetSession("loadmon")
//	if sess == nil {
//	    this.Redirect("/user/login/<page>", 302)
//	    return
//	}
//
// Статические файлы (/static/...) сюда не попадают — beego обслуживает их
// до выполнения BeforeRouter-фильтров (см. serverStaticRouter в
// beego/v2/server/web/router.go), поэтому в publicPaths их указывать не
// нужно.
//
// Редирект без сессии (цель в ?next= path[+?query], без кодировки через "_"):
//   - локальная авторизация → "/user/login/back?next=...";
//   - доменная авторизация → "/?next=..." (форма /login и /user/login
//     заблокированы и сами редиректят на "/?next=...", поэтому уводить на
//     login нельзя - получится цикл AuthFilter→login→/→AuthFilter…).
// Путь "/" публичный (publicExactPaths), фильтр на нём не срабатывает.
//
// Обработчики, которые раньше сами делали проверку "sess == nil" и
// использовали sess дальше по коду (например, для чтения privilage),
// оставляют строку `sess := this.GetSession("loadmon")` — на момент вызова
// хендлера фильтр уже гарантирует, что сессия существует, поэтому
// повторная проверка на nil не нужна и убрана.
func AuthFilter(ctx *beecontext.Context) {
	path := ctx.Request.URL.Path

	if isPublicPath(path) {
		return
	}

	if ctx.Input.CruSession == nil || ctx.Input.CruSession.Get(context.Background(), "loadmon") == nil {
		// path + query исходного URL — иначе «прямая ссылка» с ?rack=&kind= теряет
		// параметры, а кодировка через "_" ломает сегменты с подчёркиванием (404).
		next := sanitizeReturnURL(requestReturnURL(path, ctx.Request.URL.RawQuery))
		if config.ConfigData.IsDomainAuth() {
			ctx.Redirect(302, "/?next="+url.QueryEscape(next))
			return
		}
		ctx.Redirect(302, "/user/login/back?next="+url.QueryEscape(next))
	}
}
