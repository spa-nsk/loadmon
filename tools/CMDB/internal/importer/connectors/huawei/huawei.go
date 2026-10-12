// Package huawei — коннектор Huawei OceanStor Dorado V6 / OceanStor DeviceManager REST (harness/03,
// «Huawei OceanStor Dorado»): система, контроллеры, полки, диски, пулы, LUN.
//
// Учтены расхождения прошивки с документацией, ПОДТВЕРЖДЁННЫЕ на железе loadmon:
//   - сессия: POST …/sessions -> iBaseToken (заголовок) + cookie; ОБЯЗАТЕЛЬНЫЙ логаут на любой ветке
//     выхода — лимит ~32 сессий, иначе «reach the max login sessions»;
//   - ошибки приходят с HTTP 200 в error.code — разбираются;
//   - Enclosure.TYPE приходит числом (разбор нетипизированный), полку управления определять по «CTE»
//     в имени, association по полке не использовать (код типа не установлен);
//   - GET /disk списком — облегчённые записи: полные атрибуты только GET /disk/{id};
//   - ёмкость диска = SECTORS × SECTORSIZE, SECTORSIZE не 512 (на железе 520);
//   - LUN.PARENTID — id пула.
//
// Ёмкость пулов и LUN — в секторах по 512 байт (документация Huawei) — ⚠️ проверить на массиве.
package huawei

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"sync"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
	"cmdb/internal/importer/connectors/storage"
)

// Kind — значение imp.connector.kind.
const Kind = "huawei"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного массива.
type Client struct {
	code     string
	http     *httpx.Client
	user     string
	password string
	scope    string
	family   string
	deviceID string
	pageSize int
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://array:8088), username,
// password_env, scope (0 — локальный пользователь, по умолчанию), array_family (метка модели, по
// умолчанию "Huawei OceanStor Dorado V6"), page_size (размер блока range, 100), timeout_sec (на запрос, 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("huawei %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("huawei %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	scope := cfg.String("scope")
	if scope == "" {
		scope = "0"
	}
	family := cfg.String("array_family")
	if family == "" {
		family = "Huawei OceanStor Dorado V6"
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	jar, _ := cookiejar.New(nil)
	h.HTTP.Jar = jar
	return &Client{code: cfg.Code, http: h, user: user, password: pass, scope: scope, family: family, pageSize: max(1, cfg.Int("page_size", 100))}, nil
}

type node = map[string]any

// call выполняет запрос и разбирает конверт {data, error}; error.code != 0 — ошибка (HTTP 200).
func (c *Client) call(ctx context.Context, method, path string, body any) (any, error) {
	var env struct {
		Data  any `json:"data"`
		Error struct {
			Code        json.Number `json:"code"`
			Description string      `json:"description"`
		} `json:"error"`
	}
	if err := c.http.JSON(ctx, method, path, body, &env); err != nil {
		return nil, err
	}
	if code := env.Error.Code.String(); code != "0" && code != "" {
		return nil, fmt.Errorf("%s %s: код ошибки %s: %s", method, path, code, env.Error.Description)
	}
	return env.Data, nil
}

// login открывает сессию; возвращает функцию обязательного выхода.
func (c *Client) login(ctx context.Context) (func(), error) {
	data, err := c.call(ctx, http.MethodPost, "/deviceManager/rest/xxxxx/sessions",
		map[string]any{"username": c.user, "password": c.password, "scope": c.scope})
	if err != nil {
		return nil, err
	}
	d, _ := data.(map[string]any)
	c.deviceID, _ = d["deviceid"].(string)
	token, _ := d["iBaseToken"].(string)
	if c.deviceID == "" || token == "" {
		return nil, storage.Errorf(Kind, c.code, "вход не вернул deviceid/iBaseToken")
	}
	c.http.Header.Set("iBaseToken", token)
	return func() {
		lctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = c.call(lctx, http.MethodDelete, c.base()+"/sessions", nil)
		c.http.Header.Del("iBaseToken")
	}, nil
}

func (c *Client) base() string { return "/deviceManager/rest/" + c.deviceID }

// Test открывает и закрывает сессию.
func (c *Client) Test(ctx context.Context) error {
	logout, err := c.login(ctx)
	if err != nil {
		return err
	}
	logout()
	return nil
}

// list читает коллекцию блоками по pageSize (range=[from-to)) до неполного блока.
func (c *Client) list(ctx context.Context, typ string) ([]node, error) {
	var all []node
	for from := 0; from < 1000000; from += c.pageSize {
		data, err := c.call(ctx, http.MethodGet, fmt.Sprintf("%s/%s?range=[%d-%d]", c.base(), typ, from, from+c.pageSize), nil)
		if err != nil {
			return nil, err
		}
		rows, _ := data.([]any)
		for _, r := range rows {
			if n, ok := r.(map[string]any); ok {
				all = append(all, n)
			}
		}
		if len(rows) < c.pageSize {
			break
		}
	}
	return all, nil
}

// listAssoc читает коллекцию typ, ассоциированную с объектом assocType/assocID (код типа РОДИТЕЛЯ);
// objType — TYPE запрашиваемых объектов (для lun/associate).
func (c *Client) listAssoc(ctx context.Context, typ, objType, assocType, assocID string) ([]node, error) {
	var all []node
	for from := 0; from < 1000000; from += c.pageSize {
		q := fmt.Sprintf("%s/%s?ASSOCIATEOBJTYPE=%s&ASSOCIATEOBJID=%s&range=[%d-%d]", c.base(), typ, assocType, assocID, from, from+c.pageSize)
		if objType != "" {
			q += "&TYPE=" + objType
		}
		data, err := c.call(ctx, http.MethodGet, q, nil)
		if err != nil {
			return nil, err
		}
		rows, _ := data.([]any)
		for _, r := range rows {
			if n, ok := r.(map[string]any); ok {
				all = append(all, n)
			}
		}
		if len(rows) < c.pageSize {
			break
		}
	}
	return all, nil
}

func (c *Client) one(ctx context.Context, path string) (node, error) {
	data, err := c.call(ctx, http.MethodGet, c.base()+path, nil)
	if err != nil {
		return nil, err
	}
	n, _ := data.(map[string]any)
	return n, nil
}

// Extract строит полный снимок массива; обязательный логаут — на любой ветке выхода.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	logout, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	defer logout()

	sys, err := c.one(ctx, "/system/")
	if err != nil {
		return nil, err
	}
	snap := storage.Snapshot{Array: storage.Array{Name: firstNonEmpty(str(sys, "NAME"), c.code), Serial: c.deviceID, Family: c.family,
		Microcode: strings.TrimSpace(str(sys, "PRODUCTVERSION") + " " + str(sys, "pointRelease")),
		Vendor:    map[string]any{"vendor": "Huawei", "product_mode": str(sys, "PRODUCTMODE"), "location": str(sys, "LOCATION")}}}

	ctrls, err := c.list(ctx, "controller")
	if err != nil {
		return nil, err
	}
	for _, x := range ctrls {
		snap.Controllers = append(snap.Controllers, storage.Component{ID: str(x, "ID"), Name: firstNonEmpty(str(x, "LOCATION"), str(x, "NAME"), str(x, "ID")),
			Serial: str(x, "SN"), Firmware: str(x, "SOFTVER")})
	}

	encl, err := c.list(ctx, "enclosure")
	if err != nil {
		return nil, err
	}
	for _, e := range encl {
		snap.Enclosures = append(snap.Enclosures, storage.Component{ID: str(e, "ID"), Name: firstNonEmpty(str(e, "NAME"), str(e, "LOCATION")), Serial: str(e, "SERIALNUM"),
			Model: str(e, "MODEL")}) // TYPE приходит числом — как классификатор не используем; CTE — по имени/расположению
	}

	pools, err := c.list(ctx, "storagepool")
	if err != nil {
		return nil, err
	}
	for _, p := range pools {
		snap.Pools = append(snap.Pools, storage.Pool{ID: str(p, "ID"), Name: str(p, "NAME"), CapacityTB: num(p, "USERTOTALCAPACITY") * 512 / 1e12})
	}

	luns, err := c.list(ctx, "lun")
	if err != nil {
		return nil, err
	}
	for _, l := range luns {
		ss := num(l, "SECTORSIZE")
		if ss == 0 {
			ss = 512
		}
		v := storage.Volume{ID: str(l, "ID"), Name: str(l, "NAME"), PoolID: str(l, "PARENTID"), WWN: storage.NormalizeWWN(str(l, "WWN")),
			SizeGB: num(l, "CAPACITY") * ss / 1e9, Protocol: "block"}
		if id, err := strconv.Atoi(str(l, "ID")); err == nil {
			v.LunID = &id
		}
		if at := str(l, "ALLOCTYPE"); at != "" {
			thin := at == "1"
			v.Thin = &thin
		}
		snap.Volumes = append(snap.Volumes, v)
	}

	// Презентация: для каждого хоста — его FC-инициаторы (ID инициатора = WWPN) и LUN, ассоциированные с
	// хостом. Код типа хоста 21 ✅ подтверждён (harness/03); код типа полки — нет, поэтому по полкам
	// association не строится. Хост-запросы — по одному на хост.
	hosts, err := c.list(ctx, "host")
	if err != nil {
		return nil, err
	}
	for _, h := range hosts {
		hid := str(h, "ID")
		inits, err := c.listAssoc(ctx, "fc_initiator", "", "21", hid)
		if err != nil {
			return nil, err
		}
		var ws []string
		for _, i := range inits {
			ws = append(ws, str(i, "ID"))
		}
		mapped, err := c.listAssoc(ctx, "lun/associate", "11", "21", hid)
		if err != nil {
			return nil, err
		}
		for _, l := range mapped {
			snap.Mappings = append(snap.Mappings, storage.Mapping{VolumeID: str(l, "ID"), WWPNs: ws})
		}
	}
	snap.MappingsKnown = true

	// GET /disk списком отдаёт облегчённые записи (пустые CAPACITY/SLOTNUM/ENCLOSUREID): полные
	// атрибуты — только GET /disk/{id}. Сбой любого диска — ошибка снимка: частичный список дисков
	// снял бы живые компоненты.
	light, err := c.list(ctx, "disk")
	if err != nil {
		return nil, err
	}
	disks := make([]storage.Disk, len(light))
	errs := make([]error, len(light))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, d := range light {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			id := str(d, "ID")
			full, err := c.one(ctx, "/disk/"+id)
			if err != nil {
				errs[i] = err
				return
			}
			disks[i] = storage.Disk{ID: id, Serial: str(full, "SERIALNUMBER"), Model: str(full, "MODEL"), Manufacturer: str(full, "MANUFACTURER"),
				Firmware: str(full, "FIRMWAREVER"), Location: firstNonEmpty(str(full, "LOCATION"), id), EnclosureID: str(full, "ENCLOSUREID"),
				Type: diskType(str(full, "DISKTYPE")), CapacityGB: num(full, "SECTORS") * num(full, "SECTORSIZE") / 1e9}
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	snap.Disks = disks

	return storage.Items(storage.Build(snap, c.code)), nil
}

// diskType — код DISKTYPE (MIB HUAWEI-STORAGE-HARDWARE-MIB, общий для REST/SNMP): NVMe SSD 14/16/21,
// прочие SSD 3, 5, 6, 10, 12, 13, 17, 18; остальное — HDD. Недокументированные коды (19/20/22) — как disk,
// без домысливания (harness/03).
func diskType(code string) string {
	switch code {
	case "14", "16", "21":
		return "nvme"
	case "3", "5", "6", "10", "12", "13", "17", "18":
		return "ssd"
	}
	return "disk"
}

func str(n node, k string) string {
	switch v := n[k].(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

func num(n node, k string) float64 {
	switch v := n[k].(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	}
	return 0
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
