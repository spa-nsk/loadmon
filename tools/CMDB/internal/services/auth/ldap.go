package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// LDAPConfig — подключение к каталогу (AD, ALD Pro, FreeIPA). Пароль учётной записи поиска берётся
// только из окружения, в БД и в app.conf его нет.
type LDAPConfig struct {
	URL                string // ldaps://dc.corp.local:636 либо ldap://… (тогда StartTLS)
	StartTLS           bool
	InsecureSkipVerify bool // только для стендов
	BindDN             string
	BindPassword       string
	UserBase           string
	UserFilter         string // %s — имя пользователя (экранируется); по умолчанию по Kind
	Kind               string // ad | freeipa | aldpro
	GroupAttr          string // по умолчанию memberOf
	Timeout            time.Duration
}

// LDAPFromEnv собирает конфигурацию из CMDB_LDAP_*; nil — LDAP не настроен (входят только локальные учётные записи).
func LDAPFromEnv() *LDAPConfig {
	url := os.Getenv("CMDB_LDAP_URL")
	if url == "" {
		return nil
	}
	return &LDAPConfig{
		URL: url, StartTLS: os.Getenv("CMDB_LDAP_STARTTLS") == "1", InsecureSkipVerify: os.Getenv("CMDB_LDAP_INSECURE") == "1",
		BindDN: os.Getenv("CMDB_LDAP_BIND_DN"), BindPassword: os.Getenv("CMDB_LDAP_BIND_PASSWORD"),
		UserBase: os.Getenv("CMDB_LDAP_USER_BASE"), UserFilter: os.Getenv("CMDB_LDAP_USER_FILTER"), Kind: os.Getenv("CMDB_LDAP_KIND"),
		GroupAttr: os.Getenv("CMDB_LDAP_GROUP_ATTR"),
	}
}

// LDAPUser — найденный и аутентифицированный пользователь каталога.
type LDAPUser struct {
	DN, Username, DisplayName, Mail string
	Groups                          []string // DN групп (memberOf)
}

// Ошибки каталога: неверные данные отличаются от сбоя доступа (сбой не должен считаться неудачной попыткой входа).
var (
	ErrLDAPNotFound    = errors.New("пользователь не найден в каталоге")
	ErrLDAPBadCreds    = errors.New("неверный пароль в каталоге")
	ErrLDAPUnavailable = errors.New("каталог недоступен")
)

func (c *LDAPConfig) filter() string {
	if c.UserFilter != "" {
		return c.UserFilter
	}
	switch strings.ToLower(c.Kind) {
	case "freeipa", "aldpro", "ald":
		return "(&(objectClass=posixAccount)(uid=%s))"
	}
	return "(&(objectClass=user)(objectCategory=person)(|(sAMAccountName=%s)(userPrincipalName=%s)))"
}

func (c *LDAPConfig) dial(ctx context.Context) (*ldap.Conn, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	tlsCfg := &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify} //nolint:gosec // явная настройка стенда
	conn, err := ldap.DialURL(c.URL, ldap.DialWithDialer(&net.Dialer{Timeout: timeout}), ldap.DialWithTLSConfig(tlsCfg))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLDAPUnavailable, err)
	}
	conn.SetTimeout(timeout)
	if c.StartTLS && !strings.HasPrefix(strings.ToLower(c.URL), "ldaps") {
		if err := conn.StartTLS(tlsCfg); err != nil {
			conn.Close()
			return nil, fmt.Errorf("%w: StartTLS: %v", ErrLDAPUnavailable, err)
		}
	}
	return conn, nil
}

// Authenticate находит пользователя учётной записью поиска и проверяет пароль привязкой от его имени.
// Пустой пароль отклоняется до обращения к каталогу: простая привязка с пустым паролем — «анонимная»
// и у многих серверов завершается успехом.
func (c *LDAPConfig) Authenticate(ctx context.Context, username, password string) (LDAPUser, error) {
	var u LDAPUser
	if strings.TrimSpace(username) == "" || password == "" {
		return u, ErrLDAPBadCreds
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return u, err
	}
	defer conn.Close()

	if c.BindDN != "" {
		if err := conn.Bind(c.BindDN, c.BindPassword); err != nil {
			return u, fmt.Errorf("%w: учётная запись поиска: %v", ErrLDAPUnavailable, err)
		}
	}
	groupAttr := c.GroupAttr
	if groupAttr == "" {
		groupAttr = "memberOf"
	}
	esc := ldap.EscapeFilter(username)
	res, err := conn.Search(ldap.NewSearchRequest(c.UserBase, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 2, 0, false,
		strings.ReplaceAll(c.filter(), "%s", esc), // %s допускается несколько раз
		[]string{"dn", "displayName", "cn", "mail", "uid", "sAMAccountName", groupAttr}, nil))
	if err != nil {
		return u, fmt.Errorf("%w: поиск: %v", ErrLDAPUnavailable, err)
	}
	if len(res.Entries) != 1 {
		return u, ErrLDAPNotFound // 0 — нет такого; >1 — неоднозначно, тоже не входим
	}
	e := res.Entries[0]
	if err := conn.Bind(e.DN, password); err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return u, ErrLDAPBadCreds
		}
		return u, fmt.Errorf("%w: привязка: %v", ErrLDAPUnavailable, err)
	}
	u = LDAPUser{DN: e.DN, Username: firstNonEmpty(e.GetAttributeValue("sAMAccountName"), e.GetAttributeValue("uid"), username),
		DisplayName: firstNonEmpty(e.GetAttributeValue("displayName"), e.GetAttributeValue("cn")), Mail: e.GetAttributeValue("mail"),
		Groups: e.GetAttributeValues(groupAttr)}
	return u, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// groupKeys — ключи для сопоставления группы роли: полный DN и её CN (значение первого RDN), в нижнем регистре.
func groupKeys(dn string) []string {
	keys := []string{strings.ToLower(dn)}
	if parsed, err := ldap.ParseDN(dn); err == nil && len(parsed.RDNs) > 0 && len(parsed.RDNs[0].Attributes) > 0 {
		keys = append(keys, strings.ToLower(parsed.RDNs[0].Attributes[0].Value))
	}
	return keys
}
