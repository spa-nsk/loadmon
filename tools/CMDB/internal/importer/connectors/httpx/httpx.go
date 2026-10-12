// Package httpx — тонкий JSON-клиент поверх net/http для REST-коннекторов: общие тайм-аут,
// TLS-настройка стенда, лимит размера ответа и единый текст ошибки с путём и кодом HTTP.
package httpx

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client — клиент одного источника.
type Client struct {
	BaseURL  string
	HTTP     *http.Client
	Header   http.Header // постоянные заголовки (версия API, токен сессии)
	User     string      // если задан — Basic-авторизация
	Password string
}

// maxBody — верхняя граница размера ответа (защита от неограниченного чтения).
const maxBody = 256 << 20

// New создаёт клиента. insecure отключает проверку сертификата — только для стендов с
// самоподписанным сертификатом (явная настройка коннектора).
func New(baseURL string, timeout time.Duration, insecure bool) *Client {
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}} //nolint:gosec // явная настройка стенда
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Transport: tr, Timeout: timeout},
		Header:  http.Header{"Accept": {"application/json"}},
	}
}

// JSON выполняет запрос method BaseURL+path с телом body (nil — без тела; иначе кодируется в
// JSON) и разбирает ответ в out (nil — тело игнорируется). Статус, отличный от 2xx, — ошибка.
func (c *Client) JSON(ctx context.Context, method, path string, body, out any) error {
	_, err := c.JSONHeaders(ctx, method, path, body, out)
	return err
}

// JSONHeaders — как JSON, но возвращает и заголовки ответа (токены сессий: Unity EMC-CSRF-TOKEN,
// Huawei iBaseToken приходит в теле — для него не нужен).
func (c *Client) JSONHeaders(ctx context.Context, method, path string, body, out any) (http.Header, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%s %s: тело запроса: %w", method, path, err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rd)
	if err != nil {
		return nil, err
	}
	for k, v := range c.Header {
		req.Header[k] = v
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.User != "" {
		req.SetBasicAuth(c.User, c.Password)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("%s %s: чтение ответа: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, Truncate(string(data), 200))
	}
	if out == nil {
		return resp.Header, nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return nil, fmt.Errorf("%s %s: разбор JSON: %w", method, path, err)
	}
	return resp.Header, nil
}

// Truncate обрезает строку до n байт для сообщений об ошибках.
func Truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
