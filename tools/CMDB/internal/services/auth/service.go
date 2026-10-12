package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"golang.org/x/crypto/bcrypt"

	"cmdb/internal/repo"
	"cmdb/internal/services/auditlog"
	"cmdb/internal/services/svcerr"
)

// Ошибки входа. Наружу (пользователю) все причины отказа показываются одним текстом — по коду видно
// причину только журналу: различие «нет такого пользователя / неверный пароль» помогало бы перебору имён.
var (
	ErrInvalidCredentials = errors.New("неверное имя пользователя или пароль")
	ErrLocked             = errors.New("вход временно заблокирован после неудачных попыток")
	ErrNoAccess           = errors.New("для учётной записи не назначено ни одной роли")
	ErrSessionExpired     = errors.New("сессия недействительна или истекла")
)

// Service — вход, сессии, пользователи.
type Service struct {
	LDAP        *LDAPConfig   // nil — только локальные учётные записи
	IdleTimeout time.Duration // тайм-аут простоя сессии (по умолчанию 30 мин)
	MaxAge      time.Duration // абсолютный срок сессии (по умолчанию 12 ч)
	MaxAttempts int           // неудачных попыток подряд до блокировки (по умолчанию 5)
	LockWindow  time.Duration // окно подсчёта и срок блокировки (по умолчанию 15 мин)
	BcryptCost  int           // по умолчанию bcrypt.DefaultCost
}

func (s *Service) defaults() Service {
	c := *s
	if c.IdleTimeout <= 0 {
		c.IdleTimeout = 30 * time.Minute
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 12 * time.Hour
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 5
	}
	if c.LockWindow <= 0 {
		c.LockWindow = 15 * time.Minute
	}
	if c.BcryptCost <= 0 {
		c.BcryptCost = bcrypt.DefaultCost
	}
	return c
}

// dummyHash — хеш заведомо неподходящего пароля: для несуществующего локального пользователя
// сравнение всё равно выполняется, чтобы время ответа не выдавало существование имени.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("cmdb-dummy-password"), bcrypt.DefaultCost)

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// locked — заблокировано ли имя: неудач с последнего успешного входа не меньше порога в пределах окна.
func (s Service) locked(ctx context.Context, username string) (bool, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT count(*)::text AS n FROM auth.login_event
		WHERE lower(username) = lower(?) AND NOT success AND reason <> 'locked'
		  AND at > now() - make_interval(secs => ?)
		  AND at > COALESCE((SELECT max(at) FROM auth.login_event WHERE lower(username) = lower(?) AND success), '-infinity')
	`, username, s.LockWindow.Seconds(), username)
	if err != nil {
		return false, fmt.Errorf("auth: проверка блокировки: %w", err)
	}
	var n int
	fmt.Sscan(row["n"], &n)
	return n >= s.MaxAttempts, nil
}

func recordAttempt(ctx context.Context, username string, success bool, ip, reason string) {
	_, _ = orm.NewOrm().RawWithCtx(ctx, `INSERT INTO auth.login_event (username, success, client_ip, reason) VALUES (?, ?, nullif(?, '')::inet, nullif(?, ''))`,
		username, success, ip, reason).Exec()
}

// Login проверяет учётные данные и открывает сессию. Порядок: блокировка по имени → локальная запись
// (bcrypt) → каталог. Сбой каталога не считается неудачной попыткой пользователя. Возвращает непрозрачный
// токен сессии (в БД хранится его SHA-256).
func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (string, Identity, error) {
	cfg := s.defaults()
	username = strings.TrimSpace(username)
	var id Identity

	if username == "" || password == "" {
		recordAttempt(ctx, username, false, ip, "empty")
		return "", id, ErrInvalidCredentials
	}
	if lk, err := cfg.locked(ctx, username); err != nil {
		return "", id, err
	} else if lk {
		recordAttempt(ctx, username, false, ip, "locked")
		_ = auditlog.Log(ctx, 0, ip, "login_locked", map[string]any{"username": username})
		return "", id, ErrLocked
	}

	user, found, err := loadUser(ctx, username)
	if err != nil {
		return "", id, err
	}
	fail := func(reason string) (string, Identity, error) {
		recordAttempt(ctx, username, false, ip, reason)
		_ = auditlog.Log(ctx, 0, ip, "login_failed", map[string]any{"username": username, "reason": reason})
		return "", Identity{}, ErrInvalidCredentials
	}

	switch {
	case found && user.Source == "local":
		hash := dummyHash
		if user.PasswordHash != "" {
			hash = []byte(user.PasswordHash)
		}
		if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
			return fail("bad_password")
		}
	case cfg.LDAP != nil && (!found || user.Source == "ldap"):
		lu, err := cfg.LDAP.Authenticate(ctx, username, password)
		switch {
		case errors.Is(err, ErrLDAPNotFound), errors.Is(err, ErrLDAPBadCreds):
			return fail("ldap_rejected")
		case err != nil:
			return "", id, err // каталог недоступен — не попытка пользователя
		}
		if user, err = syncLDAPUser(ctx, lu); err != nil {
			return "", id, err
		}
	default:
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password)) // выравнивание времени
		return fail("unknown_user")
	}

	if !user.Enabled {
		return fail("disabled")
	}
	ident, err := LoadIdentity(ctx, user.ID)
	if err != nil {
		return "", id, err
	}
	if len(ident.Grants) == 0 {
		recordAttempt(ctx, username, false, ip, "no_roles")
		_ = auditlog.Log(ctx, user.ID, ip, "login_denied_no_roles", map[string]any{"username": username})
		return "", id, ErrNoAccess
	}

	token, err := newToken()
	if err != nil {
		return "", id, err
	}
	if _, err := orm.NewOrm().RawWithCtx(ctx, `
		INSERT INTO auth.session (token_hash, user_id, expires_at, client_ip, user_agent)
		VALUES (?, ?, now() + make_interval(secs => ?), nullif(?, '')::inet, nullif(?, ''))`,
		hashToken(token), user.ID, cfg.MaxAge.Seconds(), ip, truncate(userAgent, 300)).Exec(); err != nil {
		return "", id, fmt.Errorf("auth: создание сессии: %w", err)
	}
	_, _ = orm.NewOrm().RawWithCtx(ctx, `UPDATE auth."user" SET last_login_at = now() WHERE id = ?`, user.ID).Exec()
	recordAttempt(ctx, username, true, ip, "")
	_ = auditlog.Log(ctx, user.ID, ip, "login", map[string]any{"username": user.Username, "source": user.Source})
	return token, ident, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Authenticate возвращает пользователя по токену сессии: проверяются абсолютный срок, тайм-аут простоя
// и то, что учётная запись включена; last_seen_at обновляется не чаще раза в минуту.
func (s *Service) Authenticate(ctx context.Context, token string) (Identity, error) {
	cfg := s.defaults()
	var id Identity
	if token == "" {
		return id, ErrSessionExpired
	}
	row, err := repo.QueryRow(ctx, `
		SELECT s.user_id::text AS user_id, (s.last_seen_at < now() - interval '1 minute') AS stale
		FROM auth.session s JOIN auth."user" u ON u.id = s.user_id
		WHERE s.token_hash = ? AND u.enabled AND s.expires_at > now()
		  AND s.last_seen_at > now() - make_interval(secs => ?)`, hashToken(token), cfg.IdleTimeout.Seconds())
	if err != nil {
		return id, fmt.Errorf("auth: чтение сессии: %w", err)
	}
	if row == nil {
		return id, ErrSessionExpired
	}
	var uid int64
	fmt.Sscan(row["user_id"], &uid)
	if row["stale"] == "true" {
		_, _ = orm.NewOrm().RawWithCtx(ctx, `UPDATE auth.session SET last_seen_at = now() WHERE token_hash = ?`, hashToken(token)).Exec()
	}
	return LoadIdentity(ctx, uid)
}

// Logout закрывает сессию.
func (s *Service) Logout(ctx context.Context, token string, userID int64, ip string) {
	_, _ = orm.NewOrm().RawWithCtx(ctx, `DELETE FROM auth.session WHERE token_hash = ?`, hashToken(token)).Exec()
	_ = auditlog.Log(ctx, userID, ip, "logout", nil)
}

// PurgeSessions удаляет истёкшие и простаивавшие сессии.
func (s *Service) PurgeSessions(ctx context.Context) (int64, error) {
	cfg := s.defaults()
	res, err := orm.NewOrm().RawWithCtx(ctx, `DELETE FROM auth.session WHERE expires_at <= now() OR last_seen_at <= now() - make_interval(secs => ?)`, cfg.IdleTimeout.Seconds()).Exec()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// dbUser — строка auth.user.
type dbUser struct {
	ID           int64
	Username     string
	DisplayName  string
	Source       string
	PasswordHash string
	Enabled      bool
}

func loadUser(ctx context.Context, username string) (dbUser, bool, error) {
	row, err := repo.QueryRow(ctx, `SELECT id::text AS id, username, display_name, source, COALESCE(password_hash, '') AS password_hash, enabled
		FROM auth."user" WHERE lower(username) = lower(?)`, username)
	if err != nil {
		return dbUser{}, false, fmt.Errorf("auth: чтение пользователя: %w", err)
	}
	if row == nil {
		return dbUser{}, false, nil
	}
	u := dbUser{Username: row["username"], DisplayName: row["display_name"], Source: row["source"], PasswordHash: row["password_hash"], Enabled: row["enabled"] == "true"}
	fmt.Sscan(row["id"], &u.ID)
	return u, true, nil
}

// LoadIdentity читает пользователя и его роли.
func LoadIdentity(ctx context.Context, userID int64) (Identity, error) {
	row, err := repo.QueryRow(ctx, `SELECT username, display_name, source FROM auth."user" WHERE id = ?`, userID)
	if err != nil {
		return Identity{}, err
	}
	if row == nil {
		return Identity{}, svcerr.NotFound("user", userID)
	}
	id := Identity{ID: userID, Username: row["username"], DisplayName: row["display_name"], Source: row["source"]}
	roles, err := repo.QueryRows(ctx, `SELECT role_code, COALESCE(ci_class, '') AS ci_class FROM auth.user_role WHERE user_id = ? ORDER BY role_code, ci_class`, userID)
	if err != nil {
		return id, err
	}
	for _, r := range roles {
		id.Grants = append(id.Grants, Grant{Role: r["role_code"], Class: r["ci_class"]})
	}
	return id, nil
}

// syncLDAPUser создаёт или обновляет учётную запись каталога и пересчитывает выведенные из групп роли
// (source='ldap'); назначенные вручную роли не трогаются.
func syncLDAPUser(ctx context.Context, lu LDAPUser) (dbUser, error) {
	var u dbUser
	err := orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		var id int64
		if err := tx.RawWithCtx(ctx, `
			INSERT INTO auth."user" (username, display_name, email, source, external_id)
			VALUES (?, ?, ?, 'ldap', ?)
			ON CONFLICT (lower(username)) DO UPDATE SET display_name = EXCLUDED.display_name, email = EXCLUDED.email, external_id = EXCLUDED.external_id
			WHERE auth."user".source = 'ldap'
			RETURNING id`, lu.Username, lu.DisplayName, lu.Mail, lu.DN).QueryRow(&id); err != nil {
			return fmt.Errorf("auth: учётная запись каталога: %w", err)
		}
		if _, err := tx.RawWithCtx(ctx, `DELETE FROM auth.user_role WHERE user_id = ? AND source = 'ldap'`, id).Exec(); err != nil {
			return err
		}
		keys := map[string]bool{}
		for _, g := range lu.Groups {
			for _, k := range groupKeys(g) {
				keys[k] = true
			}
		}
		var maps []orm.Params
		if _, err := tx.RawWithCtx(ctx, `SELECT lower(group_key) AS gk, role_code, ci_class FROM auth.group_role`).Values(&maps); err != nil {
			return err
		}
		for _, m := range maps {
			if !keys[fmt.Sprint(m["gk"])] {
				continue
			}
			var class any
			if m["ci_class"] != nil {
				class = fmt.Sprint(m["ci_class"])
			}
			if _, err := tx.RawWithCtx(ctx, `INSERT INTO auth.user_role (user_id, role_code, ci_class, source) VALUES (?, ?, ?, 'ldap') ON CONFLICT DO NOTHING`,
				id, fmt.Sprint(m["role_code"]), class).Exec(); err != nil {
				return err
			}
		}
		u.ID = id
		return nil
	})
	if err != nil {
		return u, err
	}
	loaded, _, err := loadUser(ctx, lu.Username)
	return loaded, err
}
