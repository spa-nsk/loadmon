package filters

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	beectx "github.com/beego/beego/v2/server/web/context"

	"cmdb/internal/platform/logging"
	"cmdb/internal/services/auth"
)

// SessionCookie — имя cookie сессии. Значение — непрозрачный токен; в БД хранится только его SHA-256.
const SessionCookie = "cmdb_session"

var (
	authSvc      *auth.Service
	authDisabled bool
)

// DevIdentity — «пользователь» режима без аутентификации (только локальная разработка и тесты).
var DevIdentity = auth.Identity{ID: 1, Username: "dev", DisplayName: "dev", Source: "local", Grants: []auth.Grant{{Role: auth.RoleAdmin}}}

// SetAuth подключает сервис аутентификации. disabled=true — режим разработки без входа: все запросы
// выполняются от DevIdentity с ролью admin. В prod-профиле такой режим недопустим — main отказывается
// стартовать (см. cmd/cmdb-web), здесь проверки нет намеренно: фильтр не знает о профиле.
func SetAuth(s *auth.Service, disabled bool) {
	authSvc, authDisabled = s, disabled
}

// AuthService возвращает подключённый сервис (для контроллера входа).
func AuthService() *auth.Service { return authSvc }

// AuthDisabled сообщает, отключена ли аутентификация.
func AuthDisabled() bool { return authDisabled }

func isPublic(path string) bool {
	switch {
	case path == "/login", path == "/healthz", path == "/metrics", path == "/api/v1/openapi.yaml", strings.HasPrefix(path, "/static/"):
		return true
	}
	return false
}

// isAPIPath — запрос к REST API /api/v1 (plan.md E7.3): ответы там всегда JSON, не HTML/HTMX-тост
// и не редирект на страницу входа.
func isAPIPath(path string) bool { return strings.HasPrefix(path, "/api/") }

func isHTMX(ctx *beectx.Context) bool { return ctx.Request.Header.Get("HX-Request") == "true" }

// bearerToken достаёт токен API из заголовка Authorization: Bearer <token>; "" — заголовка нет
// или он не в этом формате.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if len(h) > len(p) && strings.EqualFold(h[:len(p)], p) {
		return strings.TrimSpace(h[len(p):])
	}
	return ""
}

// Auth определяет пользователя запроса и кладёт его в контекст: токен API (Authorization: Bearer,
// REST /api/v1, E7.3) либо cookie сессии (UI) — в этом порядке, так что токен работает и на
// страницах UI (скрипту незачем заводить сессию). Без действующих учётных данных: обычный переход
// GET — на страницу входа с возвратом, HTMX — HX-Redirect, /api/v1 и прочее — 401.
func Auth(ctx *beectx.Context) {
	path := ctx.Request.URL.Path
	if authDisabled {
		ctx.Request = ctx.Request.WithContext(auth.WithIdentity(ctx.Request.Context(), DevIdentity))
		return
	}
	if authSvc == nil {
		deny(ctx, http.StatusServiceUnavailable, "Аутентификация не настроена.")
		return
	}
	if token := bearerToken(ctx.Request); token != "" {
		if id, err := auth.AuthenticateToken(ctx.Request.Context(), token); err == nil {
			ctx.Request = ctx.Request.WithContext(auth.WithIdentity(ctx.Request.Context(), id))
			return
		}
	} else if c, err := ctx.Request.Cookie(SessionCookie); err == nil {
		if id, err := authSvc.Authenticate(ctx.Request.Context(), c.Value); err == nil {
			ctx.Request = ctx.Request.WithContext(auth.WithIdentity(ctx.Request.Context(), id))
			return
		}
	}
	if isPublic(path) {
		return
	}
	switch {
	case isAPIPath(path):
		deny(ctx, http.StatusUnauthorized, "Требуется токен API (Authorization: Bearer …) или вход.")
	case isHTMX(ctx):
		ctx.Output.Header("HX-Redirect", "/login")
		ctx.Output.SetStatus(http.StatusUnauthorized)
		_ = ctx.Output.Body(nil)
	case ctx.Request.Method == http.MethodGet:
		next := ctx.Request.URL.RequestURI()
		ctx.Redirect(http.StatusFound, "/login?next="+url.QueryEscape(next))
	default:
		deny(ctx, http.StatusUnauthorized, "Требуется вход.")
	}
}

// deny отвечает ошибкой без действия сервиса (аутентификация, RBAC): /api/v1 получает
// {"error": "..."}, страницы UI — текст (см. internal/controllers/ui.BaseController.fail для
// ответов уже внутри контроллера).
func deny(ctx *beectx.Context, status int, msg string) {
	ctx.Output.SetStatus(status)
	if isAPIPath(ctx.Request.URL.Path) {
		ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
		b, _ := json.Marshal(map[string]string{"error": msg})
		_ = ctx.Output.Body(b)
		return
	}
	_ = ctx.Output.Body([]byte(msg))
}

// RequiredPerm возвращает право, нужное для запроса method path; "" — достаточно любого вошедшего
// пользователя; ok=false — маршрут не описан (отклоняется: запрет по умолчанию для изменяющих запросов).
func RequiredPerm(method, path string) (perm string, ok bool) {
	write := method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
	pick := func(read, wr string) (string, bool) {
		if write {
			return wr, true
		}
		return read, true
	}
	switch {
	case path == "/logout":
		return "", true
	case strings.HasPrefix(path, "/admin/users"), strings.HasPrefix(path, "/admin/groups"), strings.HasPrefix(path, "/admin/api-tokens"):
		return auth.UserAdmin, true
	case strings.HasPrefix(path, "/admin/events"):
		return auth.AuditRead, true
	case strings.HasPrefix(path, "/api/v1/baselines"):
		return auth.BaselineRead, true
	case strings.HasPrefix(path, "/api/v1/ref"):
		return auth.CIRead, true
	case strings.HasPrefix(path, "/api/v1/cis"), strings.HasPrefix(path, "/api/v1/relations"):
		return pick(auth.CIRead, auth.CIWrite)
	case strings.HasPrefix(path, "/admin/naming"):
		return pick(auth.CIRead, auth.NamingAdmin) // песочница имён — чтение
	case strings.HasPrefix(path, "/admin/ref"), strings.HasPrefix(path, "/admin/baseline-configs"):
		return pick(auth.CIRead, auth.RefWrite)
	case strings.HasPrefix(path, "/import"):
		return pick(auth.ImportRead, auth.ImportRun)
	case strings.HasPrefix(path, "/changes"):
		return auth.AuditRead, true
	case strings.HasPrefix(path, "/quality"):
		return auth.CIRead, true // дашборд только читает агрегаты — как список КЕ
	case strings.HasPrefix(path, "/reports"):
		return auth.CIRead, true // отчёты только читают и выгружают — как список КЕ
	case strings.HasPrefix(path, "/baselines"):
		if strings.HasSuffix(path, "/verify") { // проверка целостности — чтение, хоть и POST
			return auth.BaselineRead, true
		}
		return pick(auth.BaselineRead, auth.BaselineWrite)
	case strings.HasPrefix(path, "/periods"):
		return pick(auth.BaselineRead, auth.PeriodManage)
	case path == "/cis/bulk/preview": // предпросмотр массовой операции ничего не пишет
		return auth.CIRead, true
	case path == "/", strings.HasPrefix(path, "/cis"), strings.HasPrefix(path, "/components"), strings.HasPrefix(path, "/ips"),
		strings.HasPrefix(path, "/relations"), strings.HasPrefix(path, "/software-installs"):
		return pick(auth.CIRead, auth.CIWrite)
	}
	if !write {
		return auth.CIRead, true
	}
	return "", false
}

// RBAC проверяет право на маршрут. Область класса КЕ (class_owner) проверяет контроллер: фильтр не знает,
// к какому классу относится КЕ по идентификатору в адресе.
func RBAC(ctx *beectx.Context) {
	path := ctx.Request.URL.Path
	if isPublic(path) {
		return
	}
	id, ok := auth.IdentityFromContext(ctx.Request.Context())
	if !ok {
		return // Auth уже ответил
	}
	perm, known := RequiredPerm(ctx.Request.Method, path)
	if !known {
		forbid(ctx, id, "маршрут без описания прав")
		return
	}
	if perm != "" && !id.Can(perm) {
		forbid(ctx, id, "нужно право "+perm)
	}
}

func forbid(ctx *beectx.Context, id auth.Identity, why string) {
	logging.FromContext(ctx.Request.Context()).Warn("access_denied", "user", id.Username, "method", ctx.Request.Method, "path", ctx.Request.URL.Path, "why", why)
	const msg = "Недостаточно прав для этого действия."
	if isHTMX(ctx) {
		ctx.Output.Header("HX-Trigger", `{"toast":{"kind":"danger","text":"Недостаточно прав для этого действия."}}`)
		ctx.Output.Header("HX-Reswap", "none")
	}
	deny(ctx, http.StatusForbidden, msg)
}

// CookieSecure — ставить ли Secure на cookie сессии: соединение по TLS либо доверенный обратный прокси
// сообщил https (X-Forwarded-Proto).
func CookieSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
