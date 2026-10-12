package ui

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/filters"
	"cmdb/internal/platform/logging"
	"cmdb/internal/services/auth"
)

// AuthController — вход и выход (plan.md E7.1).
type AuthController struct {
	BaseController
}

// safeNext — адрес возврата после входа: только путь на этом же сайте (защита от открытого редиректа).
func safeNext(next string) string {
	if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") && !strings.HasPrefix(next, "/\\") && !strings.HasPrefix(next, "/login") {
		return next
	}
	return "/"
}

func (c *AuthController) renderLogin(status int, msg string) {
	c.Layout = ""
	c.TplName = "auth/login.tpl"
	c.Data["Error"] = msg
	c.Data["Next"] = safeNext(c.GetString("next"))
	c.Data["LDAP"] = filters.AuthService() != nil
	if status == http.StatusOK {
		return // beego сам отрендерит шаблон
	}
	// при уже выставленном статусе beego автоматический рендер пропускает — рендерим вручную
	body, err := c.RenderString()
	if err != nil {
		logging.FromContext(c.Ctx.Request.Context()).Error("login_render_failed", "error", err)
		body = "Ошибка отображения страницы входа."
	}
	c.Ctx.Output.SetStatus(status)
	_ = c.Ctx.Output.Body([]byte(body))
}

// LoginForm — страница входа.
func (c *AuthController) LoginForm() {
	if _, ok := auth.IdentityFromContext(c.Ctx.Request.Context()); ok && !filters.AuthDisabled() {
		c.Redirect(safeNext(c.GetString("next")), http.StatusFound)
		return
	}
	c.renderLogin(http.StatusOK, "")
}

// Login проверяет учётные данные и выставляет cookie сессии. Пользователю показывается один текст для
// любой причины отказа (кроме блокировки и недоступности каталога — они не раскрывают, существует ли имя).
func (c *AuthController) Login() {
	svc := filters.AuthService()
	if svc == nil {
		c.renderLogin(http.StatusServiceUnavailable, "Вход не настроен.")
		return
	}
	ip := c.Ctx.Input.IP()
	token, ident, err := svc.Login(c.Ctx.Request.Context(), c.GetString("username"), c.GetString("password"), ip, c.Ctx.Request.UserAgent())
	switch {
	case err == nil:
	case errors.Is(err, auth.ErrLocked):
		c.renderLogin(http.StatusTooManyRequests, "Вход временно заблокирован после неудачных попыток. Повторите позже.")
		return
	case errors.Is(err, auth.ErrNoAccess):
		c.renderLogin(http.StatusForbidden, "Для вашей учётной записи не назначено ни одной роли. Обратитесь к администратору.")
		return
	case errors.Is(err, auth.ErrInvalidCredentials):
		c.renderLogin(http.StatusUnauthorized, "Неверное имя пользователя или пароль.")
		return
	default:
		logging.FromContext(c.Ctx.Request.Context()).Error("login_failed", "error", err)
		c.renderLogin(http.StatusServiceUnavailable, "Служба входа временно недоступна (каталог пользователей не отвечает). Повторите позже.")
		return
	}
	http.SetCookie(c.Ctx.ResponseWriter, &http.Cookie{Name: filters.SessionCookie, Value: token, Path: "/", HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: filters.CookieSecure(c.Ctx.Request)})
	logging.FromContext(c.Ctx.Request.Context()).Info("login", "user", ident.Username)
	c.Redirect(safeNext(c.GetString("next")), http.StatusFound)
}

// Logout закрывает сессию и снимает cookie.
func (c *AuthController) Logout() {
	if svc := filters.AuthService(); svc != nil {
		if ck, err := c.Ctx.Request.Cookie(filters.SessionCookie); err == nil {
			svc.Logout(c.Ctx.Request.Context(), ck.Value, c.userID(), c.Ctx.Input.IP())
		}
	}
	http.SetCookie(c.Ctx.ResponseWriter, &http.Cookie{Name: filters.SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Expires: time.Unix(0, 0)})
	c.Redirect("/login", http.StatusFound)
}

// UserAdminController — пользователи, роли и соответствие групп каталога ролям (право user.admin).
type UserAdminController struct {
	BaseController
}

func (c *UserAdminController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

func (c *UserAdminController) uid() int64 {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	return id
}

var roleOptions = []string{auth.RoleViewer, auth.RoleEditor, auth.RoleClassOwner, auth.RoleImport, auth.RoleAuditor, auth.RoleAccounting, auth.RoleAdmin}

// Users — список пользователей и формы управления.
func (c *UserAdminController) Users() {
	users, err := auth.ListUsers(c.Ctx.Request.Context())
	c.must(err)
	c.Data["Nav"] = "users"
	c.Data["Users"] = users
	c.Data["Roles"] = roleOptions
	c.Data["MinPass"] = auth.MinPasswordLen
	c.page("admin/users.tpl")
}

// CreateUser создаёт локальную учётную запись, при указанной роли — сразу назначает её.
func (c *UserAdminController) CreateUser() {
	svc := filters.AuthService()
	if svc == nil {
		c.failText("Аутентификация не настроена.")
	}
	id, err := svc.CreateLocal(c.Ctx.Request.Context(), c.userID(), c.GetString("username"), c.GetString("display_name"), c.GetString("email"), c.GetString("password"))
	c.must(err)
	if role := c.GetString("role"); role != "" {
		c.must(auth.GrantRole(c.Ctx.Request.Context(), c.userID(), id, role, strings.TrimSpace(c.GetString("ci_class"))))
	}
	c.ok("Пользователь создан.")
	c.redirect("/admin/users")
}

func (c *UserAdminController) GrantRole() {
	c.must(auth.GrantRole(c.Ctx.Request.Context(), c.userID(), c.uid(), c.GetString("role"), strings.TrimSpace(c.GetString("ci_class"))))
	c.ok("Роль назначена.")
	c.redirect("/admin/users")
}

func (c *UserAdminController) RevokeRole() {
	c.must(auth.RevokeRole(c.Ctx.Request.Context(), c.userID(), c.uid(), c.GetString("role"), strings.TrimSpace(c.GetString("ci_class"))))
	c.ok("Роль снята.")
	c.redirect("/admin/users")
}

func (c *UserAdminController) SetEnabled() {
	enabled := c.GetString("enabled") == "1"
	if !enabled && c.uid() == c.userID() {
		c.failText("Нельзя отключить собственную учётную запись.")
	}
	c.must(auth.SetEnabled(c.Ctx.Request.Context(), c.userID(), c.uid(), enabled))
	c.ok("Готово.")
	c.redirect("/admin/users")
}

func (c *UserAdminController) SetPassword() {
	svc := filters.AuthService()
	if svc == nil {
		c.failText("Аутентификация не настроена.")
	}
	c.must(svc.SetPassword(c.Ctx.Request.Context(), c.userID(), c.uid(), c.GetString("password")))
	c.ok("Пароль изменён, сессии пользователя закрыты.")
	c.redirect("/admin/users")
}

// Groups — соответствие групп каталога ролям.
func (c *UserAdminController) Groups() {
	list, err := auth.ListGroupRoles(c.Ctx.Request.Context())
	c.must(err)
	c.Data["Nav"] = "users"
	c.Data["Mappings"] = list
	c.Data["Roles"] = roleOptions
	c.page("admin/groups.tpl")
}

func (c *UserAdminController) SetGroupRole() {
	c.must(auth.SetGroupRole(c.Ctx.Request.Context(), c.userID(), c.GetString("group"), c.GetString("role"), strings.TrimSpace(c.GetString("ci_class"))))
	c.ok("Сопоставление сохранено. Роли пересчитаются при следующем входе пользователей.")
	c.redirect("/admin/groups")
}

func (c *UserAdminController) UnsetGroupRole() {
	c.must(auth.UnsetGroupRole(c.Ctx.Request.Context(), c.userID(), c.GetString("group"), c.GetString("role"), strings.TrimSpace(c.GetString("ci_class"))))
	c.ok("Сопоставление удалено.")
	c.redirect("/admin/groups")
}
