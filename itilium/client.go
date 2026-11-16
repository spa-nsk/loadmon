package itilium

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/go-ntlmssp"
)

const (
	// DefaultExternalPrefix — корневой URL HTTP-сервиса внешнего API обращений
	// относительно публикации 1С (см. doc.go).
	DefaultExternalPrefix = "/hs/externalapi"
	// DefaultPortalPrefix — корневой URL HTTP-сервиса веб-портала.
	DefaultPortalPrefix = "/hs/web-portal-api"
	// DefaultAPIPrefix — кастомный HTTP-сервис `/hs/api` (getCU и родственные
	// методы выгрузки КЕ; в BCS — публикация DEV на itsd.global.bcs).
	DefaultAPIPrefix = "/hs/api"

	// AuthBasic — HTTP Basic (логин как есть, либо DOMAIN\user при Domain).
	AuthBasic = "basic"
	// AuthNTLM — NTLM/Negotiate через github.com/Azure/go-ntlmssp (доменная
	// авторизация к IIS/1С). Логин передаётся как DOMAIN\user.
	AuthNTLM = "ntlm"

	defaultTimeout = 60 * time.Second
)

// Config — параметры подключения к опубликованной информационной базе 1С:ITILIUM.
//
// BaseURL — корень публикации без завершающего слэша, например
// `https://itsd.global.bcs/DEV` или `http://192.168.1.50/ITILIUM_API`.
// ExternalPrefix / PortalPrefix / APIPrefix по умолчанию
// DefaultExternalPrefix / DefaultPortalPrefix / DefaultAPIPrefix.
//
// Доменная авторизация: задайте Domain (например `itildomail`) и AuthMode
// AuthNTLM (по умолчанию, если Domain не пуст) либо AuthBasic с тем же
// Domain — тогда Authorization уходит как `DOMAIN\Login`.
type Config struct {
	Name               string // для логов/ошибок
	BaseURL            string
	Login              string
	Password           string
	Domain             string // AD-домен; пусто - локальный пользователь 1С
	AuthMode           string // AuthBasic | AuthNTLM; пусто - ntlm при Domain, иначе basic
	ExternalPrefix     string
	PortalPrefix       string
	APIPrefix          string
	Timeout            time.Duration
	InsecureSkipVerify bool // только для тестовых стендов с самоподписанным TLS
	HTTPClient         *http.Client
}

// Client — общая HTTP-сессия к публикации ITILIUM.
type Client struct {
	cfg    Config
	base   *url.URL
	http   *http.Client
	extPre string
	porPre string
	apiPre string
}

// NewClient создаёт клиента без сетевых запросов. Пустые Login/BaseURL —
// ошибка конфигурации (пароль может быть пустым, если так настроена ИБ).
func NewClient(cfg Config) (*Client, error) {
	raw := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if raw == "" {
		return nil, ErrEmptyBaseURL
	}
	if strings.TrimSpace(cfg.Login) == "" {
		return nil, ErrEmptyLogin
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("itilium[%s]: BaseURL %q: %w", cfg.Name, raw, err)
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	authMode := normalizeAuthMode(cfg.AuthMode, cfg.Domain)
	cfg.AuthMode = authMode

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		if cfg.InsecureSkipVerify {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
		}
		var rt http.RoundTripper = tr
		if authMode == AuthNTLM {
			rt = ntlmssp.Negotiator{RoundTripper: tr}
		}
		httpClient = &http.Client{Timeout: timeout, Transport: rt}
	}
	ext := normalizePrefix(cfg.ExternalPrefix, DefaultExternalPrefix)
	por := normalizePrefix(cfg.PortalPrefix, DefaultPortalPrefix)
	api := normalizePrefix(cfg.APIPrefix, DefaultAPIPrefix)
	return &Client{cfg: cfg, base: u, http: httpClient, extPre: ext, porPre: por, apiPre: api}, nil
}

func normalizeAuthMode(mode, domain string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AuthBasic, "http-basic":
		return AuthBasic
	case AuthNTLM, "negotiate", "domain", "windows":
		return AuthNTLM
	case "":
		if strings.TrimSpace(domain) != "" {
			return AuthNTLM
		}
		return AuthBasic
	default:
		return AuthBasic
	}
}

func normalizePrefix(v, def string) string {
	v = strings.TrimRight(strings.TrimSpace(v), "/")
	if v == "" {
		v = def
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	return v
}

// External возвращает клиент внешнего API обращений (/hs/externalapi).
func (c *Client) External() *ExternalClient { return &ExternalClient{c: c} }

// Portal возвращает клиент HTTP-сервиса веб-портала (/hs/web-portal-api).
func (c *Client) Portal() *PortalClient { return &PortalClient{c: c} }

// API возвращает клиент кастомного HTTP-сервиса `/hs/api` (getCU и др.).
func (c *Client) API() *APIClient { return &APIClient{c: c} }

// authUser возвращает имя для Authorization: DOMAIN\user при заданном Domain,
// иначе Login как есть (уже может содержать `\`).
func (c *Client) authUser() string {
	login := strings.TrimSpace(c.cfg.Login)
	domain := strings.TrimSpace(c.cfg.Domain)
	if domain == "" || strings.Contains(login, `\`) || strings.Contains(login, "@") {
		return login
	}
	return domain + `\` + login
}

func (c *Client) resolve(prefix, methodPath string) string {
	methodPath = "/" + strings.TrimLeft(methodPath, "/")
	return c.base.String() + prefix + methodPath
}

func (c *Client) resolveQuery(prefix, methodPath string, q url.Values) string {
	u := c.resolve(prefix, methodPath)
	if len(q) == 0 {
		return u
	}
	return u + "?" + q.Encode()
}

func (c *Client) doJSON(ctx context.Context, method, fullURL string, reqBody any, out any) error {
	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("itilium[%s]: marshal: %w", c.cfg.Name, err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.authUser(), c.cfg.Password)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("itilium[%s]: %s %s: %w", c.cfg.Name, method, fullURL, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("itilium[%s]: чтение тела: %w", c.cfg.Name, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Method: method, URL: fullURL, Status: resp.StatusCode, Body: string(raw)}
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("itilium[%s]: decode %s: %w; body=%s", c.cfg.Name, fullURL, err, truncate(string(raw), 256))
	}
	return nil
}

// DoRaw выполняет запрос и возвращает сырое тело (для диагностики схемы ответа).
func (c *Client) DoRaw(ctx context.Context, method, fullURL string, reqBody any) ([]byte, int, error) {
	var body io.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, 0, err
	}
	req.SetBasicAuth(c.authUser(), c.cfg.Password)
	req.Header.Set("Accept", "application/json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return raw, resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
