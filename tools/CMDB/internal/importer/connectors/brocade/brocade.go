// Package brocade — коннектор Brocade Fabric OS REST API (plan.md E8.4): FC-коммутатор (класс
// fc_switch), его порты как WWPN-интерфейсы (мост к презентации томов СХД, E5.5) и активные SAN-зоны
// (класс san_zone) из действующей (effective), а не только заданной (defined) конфигурации зонирования
// — CMDB описывает фактически применённое состояние, а не любую сохранённую, но не активную
// конфигурацию (тот же принцип, что harness/02 для остальных источников).
//
// Аутентификация — сессия: POST /rest/login с HTTP Basic → заголовок Authorization в ответе
// используется как есть на последующих запросах (FOS REST не документирует его как bearer-токен,
// это непрозрачная строка сессии); DELETE /rest/logout завершает сессию.
//
// Формат ответов взят из публичной документации Brocade Fabric OS REST API и НЕ подтверждён на
// живом коммутаторе (тот же статус, что у остальных коннекторов E5/E8 без доступа к стенду) — в
// первую очередь могут потребовать поправки вложенность и точные имена полей ответа зонирования
// (brocade-zone/effective-configuration).
package brocade

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
	"cmdb/internal/importer/connectors/storage"
)

// Kind — значение imp.connector.kind.
const Kind = "brocade"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного коммутатора (свитча) Brocade.
type Client struct {
	code     string
	http     *httpx.Client
	user     string
	password string
	fabric   string
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://switch.example.local),
// username, password_env (имя переменной окружения), fabric_name (имя фабрики для КЕ зон, по
// умолчанию код коннектора), timeout_sec (на запрос, 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("brocade %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("brocade %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	h.Header.Set("Accept", "application/yang-data+json")
	h.Header.Set("Content-Type", "application/yang-data+json")
	fabric := cfg.String("fabric_name")
	if fabric == "" {
		fabric = cfg.Code
	}
	return &Client{code: cfg.Code, http: h, user: user, password: pass, fabric: fabric}, nil
}

// login открывает сессию; возвращает функцию завершения сессии (ошибка выхода не критична).
func (c *Client) login(ctx context.Context) (logout func(), err error) {
	c.http.User, c.http.Password = c.user, c.password
	hdr, err := c.http.JSONHeaders(ctx, http.MethodPost, "/rest/login", nil, nil)
	c.http.User, c.http.Password = "", ""
	if err != nil {
		return nil, err
	}
	token := hdr.Get("Authorization")
	if token == "" {
		return nil, fmt.Errorf("brocade %s: /rest/login не вернул Authorization", c.code)
	}
	c.http.Header.Set("Authorization", token)
	return func() {
		lctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = c.http.JSON(lctx, http.MethodDelete, "/rest/logout", nil, nil)
		c.http.Header.Del("Authorization")
	}, nil
}

// Test открывает и закрывает сессию.
func (c *Client) Test(ctx context.Context) error {
	logout, err := c.login(ctx)
	if err != nil {
		return err
	}
	logout()
	return nil
}

type node = map[string]any

// get разбирает конверт FOS REST {"Response": {<key>: <значение>}}; значение — объект или список.
func (c *Client) get(ctx context.Context, path, key string) (any, error) {
	var resp struct {
		Response map[string]any `json:"Response"`
	}
	if err := c.http.JSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Response[key], nil
}

func asList(v any) []node {
	switch x := v.(type) {
	case []any:
		out := make([]node, 0, len(x))
		for _, e := range x {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		return []node{x} // одиночный объект (не список) — FOS отдаёт так при единственной записи
	}
	return nil
}

// Extract строит полный снимок: коммутатор, его FC-порты, активные зоны.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	logout, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	defer logout()

	sw, err := c.get(ctx, "/rest/running/brocade-fibrechannel-switch/fibrechannel-switch", "fibrechannel-switch")
	if err != nil {
		return nil, err
	}
	switches := asList(sw)
	if len(switches) == 0 {
		return nil, storage.Errorf(Kind, c.code, "коммутатор не вернул сведений о себе")
	}
	s := switches[0]

	ports, err := c.get(ctx, "/rest/running/brocade-interface/fibrechannel", "fibrechannel")
	if err != nil {
		return nil, err
	}

	eff, err := c.get(ctx, "/rest/running/brocade-zone/effective-configuration", "effective-configuration")
	if err != nil {
		return nil, err
	}

	recs := buildRecords(s, asList(ports), effNode(eff), c.code, c.fabric)
	return storage.Items(recs), nil
}

func effNode(v any) node {
	switch x := v.(type) {
	case map[string]any:
		return x
	}
	return nil
}

// buildRecords — сведение snapshot -> canon без обращения к API (тестируется без коммутатора).
func buildRecords(sw node, ports []node, eff node, code, fabric string) []canon.Record {
	switchID := "switch:" + code
	rec := canon.Record{
		Class: "fc_switch", SourceName: str(sw, "name"), ExternalID: switchID,
		Identifiers: map[string]string{}, Attrs: map[string]any{"fabric": fabric, "domain_id": nilIntOr(str(sw, "domain-id")), "ports_total": len(ports)},
		Vendor: map[string]any{"vendor": "Brocade", "model": str(sw, "model"), "firmware_version": str(sw, "firmware-version")},
	}
	if wwn := str(sw, "wwn"); wwn != "" {
		rec.SerialNumber = wwn
		rec.Identifiers["serial_number"] = wwn
	}
	rec.Interfaces = []canon.Interface{}
	for _, p := range ports {
		w := storage.NormalizeWWPN(str(p, "wwn"))
		if w == "" {
			continue // порт без WWPN (пустой слот/не FC) — недостоверно, пропущен
		}
		rec.Interfaces = append(rec.Interfaces, canon.Interface{Name: str(p, "name"), IfType: "fc", WWPN: w,
			Enabled: strings.EqualFold(str(p, "physical-state"), "online")})
	}

	recs := []canon.Record{rec}
	if eff == nil {
		return recs
	}
	cfgName := str(eff, "cfg-name")
	for _, z := range asList(eff["zone"]) {
		var members []string
		for _, w := range strList(z["member-entry"], "entry-name") {
			if nw := storage.NormalizeWWPN(w); nw != "" {
				members = append(members, nw)
			}
		}
		recs = append(recs, canon.Record{
			Class: "san_zone", SourceName: fabric + "/" + str(z, "zone-name"), ExternalID: "zone:" + code + ":" + str(z, "zone-name"),
			Attrs: map[string]any{"fabric": fabric, "zoneset": nilStrOr(cfgName, ""), "members": members},
		})
	}
	return recs
}

// strList — вложенный список строк {"member-entry": {"entry-name": ["wwpn1", "wwpn2"]}}.
func strList(v any, key string) []string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	arr, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func str(n node, k string) string {
	if n == nil {
		return ""
	}
	s, _ := n[k].(string)
	return strings.TrimSpace(s)
}

func nilStrOr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func nilIntOr(s string) any {
	if s == "" {
		return nil
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return nil
	}
	return n
}
