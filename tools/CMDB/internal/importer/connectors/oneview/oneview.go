// Package oneview — коннектор HPE OneView REST API, версия API 300 (plan.md E5/E8.2, harness/02):
// серверы (server-hardware) и блейд-корзины (enclosures) со связью «корзина содержит сервер».
// Имя сервера — имя профиля (server-profile), если он назначен, иначе имя оборудования: у нового
// сервера без ОС имя пустое/служебное, но КЕ создаётся по серийному номеру, а не пропускается
// (harness/02: «OneView»). Частоту процессора не берём: сводное поле недостоверно.
//
// Формат ответов взят из документации OneView REST API 300 и НЕ подтверждён на живом
// устройстве (harness/10-source-oneview.md).
package oneview

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
)

// Kind — значение imp.connector.kind.
const Kind = "oneview"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного OneView appliance.
type Client struct {
	code       string
	http       *httpx.Client
	user       string
	password   string
	domain     string
	apiVersion string
	pageSize   int
}

// New создаёт коннектор. Настройки imp.connector.config: base_url, username, password_env (имя
// переменной окружения), login_domain (необязательно), api_version (по умолчанию "300"),
// page_size (по умолчанию 200), timeout_sec (на запрос, 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("oneview %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("oneview %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	ver := cfg.String("api_version")
	if ver == "" {
		ver = "300"
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	h.Header.Set("X-API-Version", ver)
	return &Client{code: cfg.Code, http: h, user: user, password: pass, domain: cfg.String("login_domain"),
		apiVersion: ver, pageSize: cfg.Int("page_size", 200)}, nil
}

// login открывает сессию; возвращает функцию завершения сессии (ошибка выхода не критична).
func (c *Client) login(ctx context.Context) (logout func(), err error) {
	body := map[string]any{"userName": c.user, "password": c.password}
	if c.domain != "" {
		body["authLoginDomain"] = c.domain
	}
	var resp struct {
		SessionID string `json:"sessionID"`
	}
	if err := c.http.JSON(ctx, http.MethodPost, "/rest/login-sessions", body, &resp); err != nil {
		return nil, err
	}
	if resp.SessionID == "" {
		return nil, fmt.Errorf("OneView не вернул sessionID")
	}
	c.http.Header.Set("Auth", resp.SessionID)
	return func() {
		lctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = c.http.JSON(lctx, http.MethodDelete, "/rest/login-sessions", nil, nil)
		c.http.Header.Del("Auth")
	}, nil
}

// Test открывает и закрывает сессию — проверка адреса, учётных данных и версии API.
func (c *Client) Test(ctx context.Context) error {
	logout, err := c.login(ctx)
	if err != nil {
		return err
	}
	logout()
	return nil
}

type page struct {
	Members     []map[string]any `json:"members"`
	NextPageURI string           `json:"nextPageUri"`
}

// collect читает коллекцию целиком, следуя nextPageUri. Ошибка любой страницы — ошибка снимка.
func (c *Client) collect(ctx context.Context, path string) ([]map[string]any, error) {
	next := fmt.Sprintf("%s?start=0&count=%d", path, c.pageSize)
	var all []map[string]any
	for i := 0; next != "" && i < 100000; i++ {
		var p page
		if err := c.http.JSON(ctx, http.MethodGet, next, nil, &p); err != nil {
			return nil, err
		}
		all = append(all, p.Members...)
		next = p.NextPageURI
	}
	return all, nil
}

// Extract строит полный снимок: корзины, серверы, профили (для имён).
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	logout, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	defer logout()

	enclosures, err := c.collect(ctx, "/rest/enclosures")
	if err != nil {
		return nil, err
	}
	servers, err := c.collect(ctx, "/rest/server-hardware")
	if err != nil {
		return nil, err
	}
	profiles, err := c.collect(ctx, "/rest/server-profiles")
	if err != nil {
		return nil, err
	}

	profileByHW := map[string]string{}
	for _, p := range profiles {
		if hw := str(p, "serverHardwareUri"); hw != "" {
			profileByHW[hw] = str(p, "name")
		}
	}

	var recs []canon.Record
	encIdx := map[string]int{} // uri корзины -> индекс в recs

	for _, e := range enclosures {
		uri := str(e, "uri")
		used := 0
		for _, b := range list(e, "deviceBays") {
			if p := str(b, "devicePresence"); p != "" && p != "Absent" {
				used++
			}
		}
		attrs := map[string]any{"bays_total": nz(num(e, "deviceBayCount")), "bays_used": used}
		for _, m := range list(e, "managerBays") {
			if ip := str(m, "ipAddress"); ip != "" {
				attrs["oa_ip"] = ip
				break
			}
		}
		serial := strings.TrimSpace(str(e, "serialNumber"))
		rec := canon.Record{
			Class: "blade_chassis", SourceName: str(e, "name"), ExternalID: "enc:" + tail(uri), SerialNumber: serial,
			Identifiers: map[string]string{}, Attrs: attrs,
			Vendor: map[string]any{"uri": uri, "model": str(e, "enclosureModel"), "part_number": str(e, "partNumber")},
		}
		if serial != "" {
			rec.Identifiers["serial_number"] = serial
		}
		encIdx[uri] = len(recs)
		recs = append(recs, rec)
	}

	for _, h := range servers {
		uri := str(h, "uri")
		serial := strings.TrimSpace(str(h, "serialNumber"))
		name := profileByHW[uri]
		profileName := name
		if name == "" {
			name = str(h, "name")
		}
		uuid := strings.ToLower(str(h, "uuid"))
		encURI := str(h, "locationUri")
		_, isBlade := encIdx[encURI]

		attrs := map[string]any{
			"bios_uuid":   nilIfEmpty(uuid),
			"firmware":    nilIfEmpty(str(h, "romVersion")),
			"cpu_count":   nz(num(h, "processorCount")),
			"core_count":  nz(num(h, "processorCount") * num(h, "processorCoreCount")),
			"ram_gb":      nz(mibToGiB(num(h, "memoryMb"))),
			"bmc_ip":      nilIfEmpty(str(h, "mpIpAddress")),
			"bmc_name":    nilIfEmpty(nested(h, "mpHostInfo", "mpHostName")),
			"form_factor": "rack",
		}
		if isBlade {
			attrs["form_factor"] = "blade"
			attrs["bay"] = nil
			if pos := num(h, "position"); pos > 0 {
				attrs["bay"] = strconv.Itoa(pos)
			}
		}
		rec := canon.Record{
			Class: "server", SourceName: name, ExternalID: "sh:" + tail(uri), SerialNumber: serial,
			Description: "", Identifiers: map[string]string{}, Attrs: attrs,
			Vendor: map[string]any{"uri": uri, "model": str(h, "model"), "short_model": str(h, "shortModel"),
				"part_number": str(h, "partNumber"), "processor_type": str(h, "processorType"),
				"mp_model": str(h, "mpModel"), "mp_firmware": str(h, "mpFirmwareVersion"),
				"asset_tag": str(h, "assetTag"), "server_profile": profileName},
		}
		if serial != "" {
			rec.Identifiers["serial_number"] = serial
		}
		if uuid != "" {
			rec.Identifiers["bios_uuid"] = uuid
		}
		idx := len(recs)
		recs = append(recs, rec)
		if isBlade {
			e := encIdx[encURI]
			recs[e].Relations = append(recs[e].Relations, canon.Relation{Type: "contains", TargetExternalID: recs[idx].ExternalID})
		}
	}

	items := make([]canon.Item, len(recs))
	for i := range recs {
		items[i] = canon.Item{Seq: i + 1, Record: recs[i]}
	}
	return items, nil
}

// --- разбор нетипизированных ответов ---

func str(m map[string]any, k string) string {
	switch v := m[k].(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func num(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case string:
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

func list(m map[string]any, k string) []map[string]any {
	raw, _ := m[k].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		if mm, ok := r.(map[string]any); ok {
			out = append(out, mm)
		}
	}
	return out
}

func nested(m map[string]any, keys ...string) string {
	cur := m
	for i, k := range keys {
		if i == len(keys)-1 {
			return str(cur, k)
		}
		next, _ := cur[k].(map[string]any)
		if next == nil {
			return ""
		}
		cur = next
	}
	return ""
}

func tail(uri string) string {
	if i := strings.LastIndex(uri, "/"); i >= 0 {
		return uri[i+1:]
	}
	return uri
}

// nz — 0 -> nil: источник не сообщил значение (неукомплектованный/неопознанный сервер), а не «ноль».
func nz(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func mibToGiB(mib int) int {
	if mib <= 0 {
		return 0
	}
	return max(1, int(math.Round(float64(mib)/1024)))
}
