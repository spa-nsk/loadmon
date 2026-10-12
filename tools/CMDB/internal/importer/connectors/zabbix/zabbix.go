// Package zabbix — коннектор Zabbix JSON-RPC (7.0 LTS, plan.md E5.4): источник-ДОПОЛНЕНИЕ
// (connector.ModeSupplement). Zabbix не владеет КЕ: он подтверждает существование (last_seen_at),
// заполняет пустые поля (серийный номер из inventory) и приносит вендорские сведения — группы,
// шаблоны (признак роли), IP интерфейсов, признак «на мониторинге». Хост без КЕ в CMDB
// пропускается с причиной, а не создаёт КЕ; отсутствие хоста в Zabbix КЕ не «списывает».
// Летучие значения (доступность агента, метрики) не берутся — каждое изменение стало бы версией в hist.
//
// Формат ответов взят из документации Zabbix API 7.0 и НЕ подтверждён на живом сервере
// (harness/11-source-zabbix.md). Ограничения, проверенные в loadmon: host.get не отдаёт MAC.
package zabbix

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
)

// Kind — значение imp.connector.kind.
const Kind = "zabbix"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного сервера Zabbix.
type Client struct {
	code     string
	http     *httpx.Client
	token    string // API-токен (предпочтительно); пусто — вход по логину/паролю
	user     string
	password string
	nextID   int
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://zabbix.example.local),
// аутентификация — либо token_env (имя переменной окружения с API-токеном), либо username +
// password_env; timeout_sec (на запрос, 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("zabbix %s: не задан base_url", cfg.Code)
	}
	c := &Client{code: cfg.Code}
	if cfg.String("token_env") != "" {
		tok, err := cfg.Secret("token_env")
		if err != nil {
			return nil, err
		}
		c.token = tok
	} else {
		if c.user = cfg.String("username"); c.user == "" {
			return nil, fmt.Errorf("zabbix %s: задайте token_env либо username и password_env", cfg.Code)
		}
		pass, err := cfg.Secret("password_env")
		if err != nil {
			return nil, err
		}
		c.password = pass
	}
	c.http = httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	return c, nil
}

// Mode — Zabbix только дополняет существующие КЕ.
func (c *Client) Mode() connector.Mode { return connector.ModeSupplement }

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// call вызывает метод JSON-RPC. Zabbix отвечает HTTP 200 и на ошибки — error разбирается из тела.
func (c *Client) call(ctx context.Context, method string, params, result any) error {
	c.nextID++
	req := map[string]any{"jsonrpc": "2.0", "method": method, "params": params, "id": c.nextID}
	var resp struct {
		Result any       `json:"result"`
		Error  *rpcError `json:"error"`
	}
	resp.Result = result
	if err := c.http.JSON(ctx, http.MethodPost, "/api_jsonrpc.php", req, &resp); err != nil {
		return fmt.Errorf("zabbix %s: %w", method, err)
	}
	if resp.Error != nil {
		return fmt.Errorf("zabbix %s: %s: %s", method, resp.Error.Message, resp.Error.Data)
	}
	return nil
}

func (c *Client) auth(ctx context.Context) error {
	c.http.Header.Set("Content-Type", "application/json-rpc")
	if c.token != "" {
		c.http.Header.Set("Authorization", "Bearer "+c.token)
		return nil
	}
	var tok string
	if err := c.call(ctx, "user.login", map[string]any{"username": c.user, "password": c.password}, &tok); err != nil {
		return err
	}
	c.http.Header.Set("Authorization", "Bearer "+tok)
	return nil
}

// Test проверяет версию API и учётные данные (пробный host.get).
func (c *Client) Test(ctx context.Context) error {
	var ver string
	if err := c.call(ctx, "apiinfo.version", []any{}, &ver); err != nil {
		return err
	}
	if !strings.HasPrefix(ver, "7.") {
		return fmt.Errorf("zabbix %s: версия API %q, коннектор рассчитан на 7.0 LTS (8.0 — проверить)", c.code, ver)
	}
	if err := c.auth(ctx); err != nil {
		return err
	}
	var probe []map[string]any
	return c.call(ctx, "host.get", map[string]any{"output": []string{"hostid"}, "limit": 1}, &probe)
}

type zbxHost struct {
	HostID     string `json:"hostid"`
	Host       string `json:"host"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Interfaces []struct {
		IP   string `json:"ip"`
		Main string `json:"main"`
	} `json:"interfaces"`
	Groups    []struct{ Name string } `json:"hostgroups"`
	Templates []struct{ Name string } `json:"parentTemplates"`
	Inventory any                     `json:"inventory"` // объект или [] при выключенном inventory
}

// Extract строит снимок хостов. Любая ошибка API — ошибка всего снимка.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	if err := c.auth(ctx); err != nil {
		return nil, err
	}
	var hosts []zbxHost
	if err := c.call(ctx, "host.get", map[string]any{
		"output":                []string{"hostid", "host", "name", "status"},
		"selectInterfaces":      []string{"ip", "main"},
		"selectHostGroups":      []string{"name"},
		"selectParentTemplates": []string{"name"},
		"selectInventory":       []string{"serial_no_a"},
		"sortfield":             "hostid",
	}, &hosts); err != nil {
		return nil, err
	}

	items := make([]canon.Item, len(hosts))
	for i, h := range hosts {
		rec := canon.Record{
			SourceName: h.Host, ExternalID: "zbx:" + h.HostID, Identifiers: map[string]string{},
		}
		if dot := strings.Index(h.Host, "."); dot > 0 {
			rec.Identifiers["short_name"] = h.Host[:dot]
		}
		if h.Name != "" && !strings.EqualFold(h.Name, h.Host) {
			rec.Identifiers["alias"] = h.Name
		}
		if inv, ok := h.Inventory.(map[string]any); ok {
			if sn, _ := inv["serial_no_a"].(string); strings.TrimSpace(sn) != "" {
				rec.SerialNumber = strings.TrimSpace(sn)
			}
		}

		var ips []string
		for _, ifc := range h.Interfaces {
			if ifc.IP != "" && ifc.IP != "127.0.0.1" {
				ips = append(ips, ifc.IP)
			}
		}
		rec.Vendor = map[string]any{
			"hostid":    h.HostID,
			"name":      h.Name,
			"groups":    sortedNames(h.Groups),
			"templates": sortedNames(h.Templates),
			"ips":       sortedUnique(ips),
			"monitored": h.Status == "0",
		}
		items[i] = canon.Item{Seq: i + 1, Record: rec}
	}
	return items, nil
}

func sortedNames(in []struct{ Name string }) []string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, x.Name)
	}
	sort.Strings(out)
	return out
}

func sortedUnique(in []string) []string {
	sort.Strings(in)
	out := make([]string, 0, len(in))
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}
