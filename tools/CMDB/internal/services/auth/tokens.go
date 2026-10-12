package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/auditlog"
	"cmdb/internal/services/svcerr"
)

// TokenPrefix — префикс токена API: по нему в тексте (логе, конфиге) сразу видно, что это токен
// CMDB, не пароль и не токен другой системы.
const TokenPrefix = "cmdb_pat_"

// CreateServiceAccount создаёт сервисную учётную запись (source='service'): вход только по
// токену API, пароля нет. Имя — как у обычного пользователя, должно быть уникальным.
func CreateServiceAccount(ctx context.Context, actor int64, username, displayName string) (int64, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, fmt.Errorf("%w: не задано имя учётной записи", svcerr.ErrInvalidInput)
	}
	var id int64
	err := orm.NewOrm().RawWithCtx(ctx, `
		INSERT INTO auth."user" (username, display_name, source, enabled) VALUES (?, ?, 'service', true) RETURNING id`,
		username, displayName).QueryRow(&id)
	if svcerr.ViolatesConstraint(err, "user_username_uniq") {
		return 0, fmt.Errorf("%w: учётная запись %q", svcerr.ErrDuplicate, username)
	}
	if err != nil {
		return 0, fmt.Errorf("auth: создание сервисной учётной записи: %w", err)
	}
	_ = auditlog.Log(ctx, actor, "", "service_account_create", map[string]any{"user_id": id, "username": username})
	return id, nil
}

// IssueToken выпускает новый токен API для учётной записи (локальной, LDAP или сервисной) —
// сервисные учётки в сессиях UI не участвуют, но токен может быть выпущен и для обычного
// пользователя (личный токен для скриптов под своими правами). ttl<=0 — без срока действия.
// Возвращает токен в открытом виде один раз — он не восстанавливается, только перевыпускается.
func IssueToken(ctx context.Context, actor, userID int64, name string, ttl time.Duration) (token string, tokenID int64, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", 0, fmt.Errorf("%w: не задано имя токена", svcerr.ErrInvalidInput)
	}
	raw, err := newToken()
	if err != nil {
		return "", 0, err
	}
	token = TokenPrefix + raw
	var expires any
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	err = orm.NewOrm().RawWithCtx(ctx, `
		INSERT INTO auth.api_token (user_id, name, token_hash, token_prefix, created_by, expires_at)
		VALUES (?, ?, ?, ?, ?, ?) RETURNING id`,
		userID, name, hashToken(token), token[:len(TokenPrefix)+8], actor, expires).QueryRow(&tokenID)
	if err != nil {
		return "", 0, fmt.Errorf("auth: выпуск токена: %w", err)
	}
	_ = auditlog.Log(ctx, actor, "", "api_token_issue", map[string]any{"user_id": userID, "token_id": tokenID, "name": name})
	return token, tokenID, nil
}

// RevokeToken отзывает токен немедленно (действующие запросы по нему получат 401).
func RevokeToken(ctx context.Context, actor, tokenID int64) error {
	res, err := orm.NewOrm().RawWithCtx(ctx, `UPDATE auth.api_token SET revoked_at = now() WHERE id = ? AND revoked_at IS NULL`, tokenID).Exec()
	if err != nil {
		return fmt.Errorf("auth: отзыв токена: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return svcerr.NotFound("api_token", tokenID)
	}
	return auditlog.Log(ctx, actor, "", "api_token_revoke", map[string]any{"token_id": tokenID})
}

// TokenInfo — токен в списке администратора (сам токен нигде, кроме момента выпуска, не хранится).
type TokenInfo struct {
	ID         int64
	UserID     int64
	Username   string
	Name       string
	Prefix     string
	CreatedAt  string
	ExpiresAt  string
	LastUsedAt string
	Revoked    bool
}

// ListTokens возвращает все выпущенные токены (для страницы администратора); userID>0 — только
// токены одной учётной записи.
func ListTokens(ctx context.Context, userID int64) ([]TokenInfo, error) {
	where, args := "", []any{}
	if userID > 0 {
		where, args = "WHERE t.user_id = ?", []any{userID}
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT t.id::text AS id, t.user_id::text AS user_id, u.username, t.name, t.token_prefix,
		       to_char(t.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
		       COALESCE(to_char(t.expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS expires_at,
		       COALESCE(to_char(t.last_used_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS last_used_at,
		       (t.revoked_at IS NOT NULL) AS revoked
		FROM auth.api_token t JOIN auth."user" u ON u.id = t.user_id
		`+where+` ORDER BY t.created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	out := make([]TokenInfo, len(rows))
	for i, r := range rows {
		out[i] = TokenInfo{Name: r["name"], Username: r["username"], Prefix: r["token_prefix"],
			CreatedAt: r["created_at"], ExpiresAt: r["expires_at"], LastUsedAt: r["last_used_at"], Revoked: r["revoked"] == "true"}
		fmt.Sscan(r["id"], &out[i].ID)
		fmt.Sscan(r["user_id"], &out[i].UserID)
	}
	return out, nil
}

// AuthenticateToken возвращает пользователя по токену API: тот же принцип, что и у сессий (сравнение
// по SHA-256), плюс проверка срока действия и отзыва. last_used_at обновляется не чаще раза в минуту.
func AuthenticateToken(ctx context.Context, token string) (Identity, error) {
	var id Identity
	if !strings.HasPrefix(token, TokenPrefix) {
		return id, ErrSessionExpired
	}
	row, err := repo.QueryRow(ctx, `
		SELECT t.id::text AS id, t.user_id::text AS user_id, (t.last_used_at IS NULL OR t.last_used_at < now() - interval '1 minute') AS stale
		FROM auth.api_token t JOIN auth."user" u ON u.id = t.user_id
		WHERE t.token_hash = ? AND t.revoked_at IS NULL AND (t.expires_at IS NULL OR t.expires_at > now()) AND u.enabled`,
		hashToken(token))
	if err != nil {
		return id, fmt.Errorf("auth: чтение токена: %w", err)
	}
	if row == nil {
		return id, ErrSessionExpired
	}
	var uid, tid int64
	fmt.Sscan(row["user_id"], &uid)
	fmt.Sscan(row["id"], &tid)
	if row["stale"] == "true" {
		_, _ = orm.NewOrm().RawWithCtx(ctx, `UPDATE auth.api_token SET last_used_at = now() WHERE id = ?`, tid).Exec()
	}
	return LoadIdentity(ctx, uid)
}
