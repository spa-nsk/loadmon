package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	"golang.org/x/crypto/bcrypt"

	"cmdb/internal/repo"
	"cmdb/internal/services/auditlog"
	"cmdb/internal/services/svcerr"
)

// MinPasswordLen — минимальная длина локального пароля.
const MinPasswordLen = 10

func (s *Service) hash(password string) (string, error) {
	cfg := s.defaults()
	if len(password) < MinPasswordLen {
		return "", fmt.Errorf("%w: пароль короче %d символов", svcerr.ErrInvalidInput, MinPasswordLen)
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), cfg.BcryptCost)
	return string(h), err
}

// CreateLocal создаёт локальную учётную запись (пароль — bcrypt) без ролей; actor — кто создаёт (журнал).
func (s *Service) CreateLocal(ctx context.Context, actor int64, username, displayName, email, password string) (int64, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, fmt.Errorf("%w: не задано имя пользователя", svcerr.ErrInvalidInput)
	}
	h, err := s.hash(password)
	if err != nil {
		return 0, err
	}
	var id int64
	err = orm.NewOrm().RawWithCtx(ctx, `INSERT INTO auth."user" (username, display_name, email, source, password_hash) VALUES (?, ?, ?, 'local', ?) RETURNING id`,
		username, displayName, email, h).QueryRow(&id)
	if svcerr.ViolatesConstraint(err, "user_username_uniq") {
		return 0, fmt.Errorf("%w: пользователь %q", svcerr.ErrDuplicate, username)
	}
	if err != nil {
		return 0, fmt.Errorf("auth: создание пользователя: %w", err)
	}
	_ = auditlog.Log(ctx, actor, "", "user_create", map[string]any{"user_id": id, "username": username})
	return id, nil
}

// SetPassword меняет пароль локальной учётной записи и закрывает все её сессии.
func (s *Service) SetPassword(ctx context.Context, actor, userID int64, password string) error {
	h, err := s.hash(password)
	if err != nil {
		return err
	}
	res, err := orm.NewOrm().RawWithCtx(ctx, `UPDATE auth."user" SET password_hash = ? WHERE id = ? AND source = 'local'`, h, userID).Exec()
	if err != nil {
		return fmt.Errorf("auth: смена пароля: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: пароль меняется только у локальной учётной записи", svcerr.ErrInvalidState)
	}
	_, _ = orm.NewOrm().RawWithCtx(ctx, `DELETE FROM auth.session WHERE user_id = ?`, userID).Exec()
	return auditlog.Log(ctx, actor, "", "user_password_change", map[string]any{"user_id": userID})
}

// SetEnabled включает/отключает учётную запись; отключение закрывает её сессии.
func SetEnabled(ctx context.Context, actor, userID int64, enabled bool) error {
	if _, err := orm.NewOrm().RawWithCtx(ctx, `UPDATE auth."user" SET enabled = ? WHERE id = ?`, enabled, userID).Exec(); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if !enabled {
		_, _ = orm.NewOrm().RawWithCtx(ctx, `DELETE FROM auth.session WHERE user_id = ?`, userID).Exec()
	}
	return auditlog.Log(ctx, actor, "", "user_enable", map[string]any{"user_id": userID, "enabled": enabled})
}

// GrantRole назначает роль вручную (class — только для class_owner). Изменение роли — событие журнала (E7.6).
func GrantRole(ctx context.Context, actor, userID int64, role, class string) error {
	if _, ok := rolePerms[role]; !ok {
		return fmt.Errorf("%w: неизвестная роль %q", svcerr.ErrInvalidInput, role)
	}
	if role == RoleClassOwner && class == "" {
		return fmt.Errorf("%w: для роли class_owner нужен класс КЕ", svcerr.ErrInvalidInput)
	}
	if role != RoleClassOwner {
		class = ""
	}
	var cl any
	if class != "" {
		cl = class
	}
	if _, err := orm.NewOrm().RawWithCtx(ctx, `INSERT INTO auth.user_role (user_id, role_code, ci_class, source) VALUES (?, ?, ?, 'manual') ON CONFLICT DO NOTHING`, userID, role, cl).Exec(); err != nil {
		return fmt.Errorf("auth: назначение роли: %w", err)
	}
	return auditlog.Log(ctx, actor, "", "role_grant", map[string]any{"user_id": userID, "role": role, "class": class})
}

// RevokeRole снимает назначенную вручную роль. Последнего администратора лишить роли нельзя — иначе
// системой никто не сможет управлять.
func RevokeRole(ctx context.Context, actor, userID int64, role, class string) error {
	if role == RoleAdmin {
		row, err := repo.QueryRow(ctx, `SELECT count(*)::text AS n FROM auth.user_role r JOIN auth."user" u ON u.id = r.user_id
			WHERE r.role_code = 'admin' AND u.enabled AND r.user_id <> ?`, userID)
		if err != nil {
			return err
		}
		if row["n"] == "0" {
			return fmt.Errorf("%w: нельзя снять роль admin у последнего администратора", svcerr.ErrInvalidState)
		}
	}
	if _, err := orm.NewOrm().RawWithCtx(ctx, `DELETE FROM auth.user_role WHERE user_id = ? AND role_code = ? AND COALESCE(ci_class, '') = ? AND source = 'manual'`,
		userID, role, class).Exec(); err != nil {
		return fmt.Errorf("auth: снятие роли: %w", err)
	}
	return auditlog.Log(ctx, actor, "", "role_revoke", map[string]any{"user_id": userID, "role": role, "class": class})
}

// UserRow — пользователь в списке администратора.
type UserRow struct {
	ID, Username, DisplayName, Email, Source, LastLogin string
	Enabled                                             bool
	Roles                                               string
}

// ListUsers возвращает пользователей с ролями.
func ListUsers(ctx context.Context) ([]UserRow, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT u.id::text AS id, u.username, u.display_name, u.email, u.source, u.enabled,
		       COALESCE(to_char(u.last_login_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS last_login,
		       COALESCE((SELECT string_agg(r.role_code || COALESCE(':' || r.ci_class, '') || CASE WHEN r.source = 'ldap' THEN ' (LDAP)' ELSE '' END, ', ' ORDER BY r.role_code)
		                 FROM auth.user_role r WHERE r.user_id = u.id), '') AS roles
		FROM auth."user" u ORDER BY lower(u.username)`)
	if err != nil {
		return nil, err
	}
	out := make([]UserRow, len(rows))
	for i, r := range rows {
		out[i] = UserRow{ID: r["id"], Username: r["username"], DisplayName: r["display_name"], Email: r["email"], Source: r["source"],
			Enabled: r["enabled"] == "true", LastLogin: r["last_login"], Roles: r["roles"]}
	}
	return out, nil
}

// SetGroupRole сопоставляет группу каталога роли; UnsetGroupRole снимает сопоставление.
func SetGroupRole(ctx context.Context, actor int64, groupKey, role, class string) error {
	groupKey = strings.TrimSpace(groupKey)
	if groupKey == "" {
		return fmt.Errorf("%w: не задана группа", svcerr.ErrInvalidInput)
	}
	if _, ok := rolePerms[role]; !ok {
		return fmt.Errorf("%w: неизвестная роль %q", svcerr.ErrInvalidInput, role)
	}
	if role == RoleClassOwner && class == "" {
		return fmt.Errorf("%w: для роли class_owner нужен класс КЕ", svcerr.ErrInvalidInput)
	}
	var cl any
	if role == RoleClassOwner {
		cl = class
	}
	if _, err := orm.NewOrm().RawWithCtx(ctx, `INSERT INTO auth.group_role (group_key, role_code, ci_class) VALUES (?, ?, ?) ON CONFLICT DO NOTHING`, groupKey, role, cl).Exec(); err != nil {
		return fmt.Errorf("auth: сопоставление группы: %w", err)
	}
	return auditlog.Log(ctx, actor, "", "group_role_set", map[string]any{"group": groupKey, "role": role, "class": class})
}

func UnsetGroupRole(ctx context.Context, actor int64, groupKey, role, class string) error {
	if _, err := orm.NewOrm().RawWithCtx(ctx, `DELETE FROM auth.group_role WHERE lower(group_key) = lower(?) AND role_code = ? AND COALESCE(ci_class, '') = ?`, groupKey, role, class).Exec(); err != nil {
		return err
	}
	return auditlog.Log(ctx, actor, "", "group_role_unset", map[string]any{"group": groupKey, "role": role, "class": class})
}

// GroupRoleRow — сопоставление группы роли.
type GroupRoleRow struct{ Group, Role, Class string }

// ListGroupRoles возвращает сопоставления групп ролям.
func ListGroupRoles(ctx context.Context) ([]GroupRoleRow, error) {
	rows, err := repo.QueryRows(ctx, `SELECT group_key, role_code, COALESCE(ci_class, '') AS ci_class FROM auth.group_role ORDER BY lower(group_key), role_code`)
	if err != nil {
		return nil, err
	}
	out := make([]GroupRoleRow, len(rows))
	for i, r := range rows {
		out[i] = GroupRoleRow{r["group_key"], r["role_code"], r["ci_class"]}
	}
	return out, nil
}
