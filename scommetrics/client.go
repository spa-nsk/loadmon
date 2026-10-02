// Package scommetrics собирает метрики ОС серверов и сведения о серверах из
// Microsoft System Center Operations Manager (SCOM) через официальный REST
// API (SCOM 2019 UR1+, базовый путь /OperationsManager) - по образцу пакета
// zabbixmetrics. Метрики пишутся в ту же таблицу server_metric с
// Source = models.MetricSourceSCOM; сведения о компьютерах - в справочник
// models.ScomHost и (только пустые поля) в models.ServerSprRecords.
//
// Параметры подключения - app2.json, блок ConfigSCOM (config.ScomEndpoint).
// Подробности, что подтверждено документацией Microsoft и что требует
// проверки на реальном сервере, - kontext/13-scom-metrics.md.
package scommetrics

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"loadmon/config"
	"loadmon/utilities"
)

// csrfCookieName - cookie с CSRF-токеном, которую SCOM 2019 UR1+ ставит при
// аутентификации; её значение (URL-decoded) передаётся заголовком с тем же
// именем во всех последующих запросах сессии (официальный REST API
// Reference, раздел "Initialize the CSRF token").
const csrfCookieName = "SCOM-CSRF-TOKEN"

// Client - HTTP-сессия к одному management-серверу SCOM.
type Client struct {
	ep      config.ScomEndpoint
	base    *url.URL
	http    *http.Client
	mu      sync.Mutex
	csrf    string
	authed  bool
	authFmt string // какой формат тела /authenticate сработал (для лога)
}

// NewClient создаёт клиента по записи ConfigSCOM (без сетевых запросов).
func NewClient(ep config.ScomEndpoint) (*Client, error) {
	raw := strings.TrimRight(strings.TrimSpace(ep.Endpoint), "/")
	if raw == "" {
		return nil, fmt.Errorf("scommetrics[%s]: Endpoint пуст", ep.Name)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("scommetrics[%s]: Endpoint %q: %w", ep.Name, raw, err)
	}
	jar, _ := cookiejar.New(nil)
	timeout := time.Duration(ep.OperationTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if ep.InsecureSkipVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // только для тестовых стендов, явно включается в app2.json
	}
	return &Client{
		ep:   ep,
		base: u,
		http: &http.Client{Jar: jar, Timeout: timeout, Transport: tr},
	}, nil
}

// password возвращает пароль с учётом соглашения проекта: значение
// "password" означает "взять из хранилища ключей" по логину (как у ConfigOV).
func (c *Client) password() string {
	if c.ep.Password == "password" {
		return utilities.GetKeyData(c.ep.Login)
	}
	return c.ep.Password
}

func (c *Client) userName() string {
	if c.ep.Domain != "" && !strings.Contains(c.ep.Login, `\`) {
		return c.ep.Domain + `\` + c.ep.Login
	}
	return c.ep.Login
}

// authenticate выполняет POST <Endpoint>/authenticate.
//
// Тело - JSON-строка с base64 от "<режим>:<DOMAIN\login>:<пароль>". В
// официальном PowerShell-примере Microsoft режим записан в скобках -
// "(Network):domain\user:password", в ряде других примеров - без скобок.
// Какой вариант принимает конкретный сервер, документацией однозначно не
// определено, поэтому пробуются оба (сначала официальный), сработавший
// пишется в лог. Режим "Windows" (учётные данные процесса,
// Negotiate/Kerberos) стандартный net/http не поддерживает.
func (c *Client) authenticate(ctx context.Context) error {
	mode := strings.TrimSpace(c.ep.AuthMode)
	if mode == "" {
		mode = "Network"
	}
	if !strings.EqualFold(mode, "Network") {
		return fmt.Errorf("scommetrics[%s]: AuthMode %q не поддерживается (нужен \"Network\" с Domain/Login/Password)", c.ep.Name, mode)
	}
	variants := []struct{ name, raw string }{
		{"(Network):user:pass", "(Network):" + c.userName() + ":" + c.password()},
		{"Network:user:pass", "Network:" + c.userName() + ":" + c.password()},
	}
	var lastErr error
	for _, v := range variants {
		body, _ := json.Marshal(base64.StdEncoding.EncodeToString([]byte(v.raw)))
		status, resp, err := c.send(ctx, http.MethodPost, "/authenticate", body, false)
		if err != nil {
			lastErr = err
			continue
		}
		if status >= 200 && status < 300 {
			c.csrf = c.csrfFromJar()
			c.authed = true
			c.authFmt = v.name
			return nil
		}
		lastErr = fmt.Errorf("authenticate (%s): HTTP %d: %s", v.name, status, truncate(resp, 300))
	}
	return fmt.Errorf("scommetrics[%s]: %w", c.ep.Name, lastErr)
}

func (c *Client) csrfFromJar() string {
	for _, ck := range c.http.Jar.Cookies(c.base) {
		if ck.Name == csrfCookieName {
			if v, err := url.QueryUnescape(ck.Value); err == nil {
				return v
			}
			return ck.Value
		}
	}
	return ""
}

// send выполняет один HTTP-запрос по пути относительно Endpoint.
func (c *Client) send(ctx context.Context, method, path string, body []byte, withCSRF bool) (int, []byte, error) {
	full := c.base.String() + path
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, full, rd)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	if withCSRF && c.csrf != "" {
		req.Header.Set(csrfCookieName, c.csrf)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

// Raw выполняет запрос с автоматической аутентификацией (и одной
// повторной попыткой после 401/403 - сессия могла истечь) и возвращает
// тело ответа как есть. body == nil - запрос без тела.
func (c *Client) Raw(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for attempt := 0; attempt < 2; attempt++ {
		if !c.authed {
			if err := c.authenticate(ctx); err != nil {
				return nil, err
			}
		}
		status, data, err := c.send(ctx, method, path, body, true)
		if err != nil {
			return nil, fmt.Errorf("scommetrics[%s]: %s %s: %w", c.ep.Name, method, path, err)
		}
		if (status == http.StatusUnauthorized || status == http.StatusForbidden) && attempt == 0 {
			c.authed = false
			continue
		}
		if status < 200 || status >= 300 {
			return data, fmt.Errorf("scommetrics[%s]: %s %s: HTTP %d: %s", c.ep.Name, method, path, status, truncate(data, 300))
		}
		return data, nil
	}
	return nil, fmt.Errorf("scommetrics[%s]: %s %s: не удалось аутентифицироваться", c.ep.Name, method, path)
}

// call - Raw + JSON-кодирование тела и разбор ответа в dest.
func (c *Client) call(ctx context.Context, method, path string, payload interface{}, dest interface{}) error {
	var body []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = b
	}
	data, err := c.Raw(ctx, method, path, body)
	if err != nil {
		return err
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("scommetrics[%s]: %s %s: разбор ответа: %w (%s)", c.ep.Name, method, path, err, truncate(data, 300))
	}
	return nil
}

// AuthFormat - какой вариант тела /authenticate принял сервер (после первого запроса).
func (c *Client) AuthFormat() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.authFmt
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
