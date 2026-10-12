package ui

import (
	"strconv"
	"strings"
	"time"

	"cmdb/internal/services/auth"
)

// TokenAdminController — сервисные учётные записи и токены API (plan.md E7.3, право user.admin):
// интеграциям (скриптам, другим системам) нужен доступ к REST API без входа человека — токен
// выпускается на сервисную учётную запись (source=service) либо на обычную (личный токен под
// своими правами), права и область те же, что у сессии (auth.Identity по auth.user_role).
type TokenAdminController struct {
	BaseController
}

func (c *TokenAdminController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

func (c *TokenAdminController) tid() int64 {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	return id
}

// Index — список сервисных и обычных учётных записей (для выбора получателя токена) и выпущенных токенов.
func (c *TokenAdminController) Index() {
	users, err := auth.ListUsers(c.Ctx.Request.Context())
	c.must(err)
	tokens, err := auth.ListTokens(c.Ctx.Request.Context(), 0)
	c.must(err)
	c.Data["Nav"] = "users"
	c.Data["Users"] = users
	c.Data["Tokens"] = tokens
	c.page("admin/tokens.tpl")
}

// CreateServiceAccount создаёт сервисную учётную запись (без пароля, только для токенов API).
func (c *TokenAdminController) CreateServiceAccount() {
	_, err := auth.CreateServiceAccount(c.Ctx.Request.Context(), c.userID(), c.GetString("username"), c.GetString("display_name"))
	c.must(err)
	c.ok("Сервисная учётная запись создана. Назначьте ей роль на странице «Пользователи» и выпустите токен.")
	c.redirect("/admin/api-tokens")
}

// IssueToken выпускает токен и показывает его один раз фрагментом — второй раз он не восстановим,
// в БД хранится только его SHA-256 (как у сессий, internal/services/auth.hashToken).
func (c *TokenAdminController) IssueToken() {
	userID, _ := strconv.ParseInt(c.GetString("user_id"), 10, 64)
	if userID == 0 {
		c.failText("Выберите учётную запись.")
	}
	name := strings.TrimSpace(c.GetString("name"))
	var ttl time.Duration
	if days, err := strconv.Atoi(strings.TrimSpace(c.GetString("ttl_days"))); err == nil && days > 0 {
		ttl = time.Duration(days) * 24 * time.Hour
	}
	token, tokenID, err := auth.IssueToken(c.Ctx.Request.Context(), c.userID(), userID, name, ttl)
	c.must(err)
	c.Data["IssuedToken"] = token
	c.Data["IssuedTokenID"] = tokenID
	c.fragment("admin/_token_issued.tpl")
}

// RevokeToken отзывает токен немедленно.
func (c *TokenAdminController) RevokeToken() {
	c.must(auth.RevokeToken(c.Ctx.Request.Context(), c.userID(), c.tid()))
	c.ok("Токен отозван.")
	c.redirect("/admin/api-tokens")
}
