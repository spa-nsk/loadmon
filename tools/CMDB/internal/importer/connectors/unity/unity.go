// Package unity — коннектор Dell EMC Unity / Unisphere REST (harness/03, «Dell EMC Unity»): массив,
// пулы, LUN, файловые системы, NAS-серверы, SMB/NFS-ресурсы, процессоры хранения, полки и диски.
//
// Факты loadmon, учтённые здесь (✅ подтверждены на железе): сессия с CSRF-токеном
// (EMC-CSRF-TOKEN из ответа на вход, cookie), заголовок X-EMC-REST-CLIENT; без явного fields=
// ответ содержит только id — поля перечисляются всегда; коллекции постраничные (links rel=next).
// Конкретные имена полей ответов сверить на пилоте (harness/14-source-storage.md).
package unity

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
	"cmdb/internal/importer/connectors/storage"
)

// Kind — значение imp.connector.kind.
const Kind = "unity"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного массива Unity.
type Client struct {
	code     string
	http     *httpx.Client
	user     string
	password string
	csrf     string
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://unity.example.local),
// username, password_env (имя переменной окружения), timeout_sec (на запрос, 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("unity %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("unity %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	h.Header.Set("X-EMC-REST-CLIENT", "true")
	jar, _ := cookiejar.New(nil)
	h.HTTP.Jar = jar // сессия держится cookie; иначе каждый запрос открывал бы новую сессию
	return &Client{code: cfg.Code, http: h, user: user, password: pass}, nil
}

type node = map[string]any

// login открывает сессию и запоминает CSRF-токен; возвращает функцию выхода.
func (c *Client) login(ctx context.Context) (func(), error) {
	c.http.User, c.http.Password = c.user, c.password
	hdr, err := c.http.JSONHeaders(ctx, http.MethodGet, "/api/types/loginSessionInfo/instances?compact=true&fields=id", nil, nil)
	if err != nil {
		return nil, err
	}
	c.csrf = hdr.Get("EMC-CSRF-TOKEN")
	return func() {
		lctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if c.csrf != "" {
			c.http.Header.Set("EMC-CSRF-TOKEN", c.csrf)
			_ = c.http.JSON(lctx, http.MethodPost, "/api/types/loginSessionInfo/action/logout", map[string]any{"localCleanupOnly": "true"}, nil)
			c.http.Header.Del("EMC-CSRF-TOKEN")
		}
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

// entries читает все страницы коллекции типа; поля перечисляются явно (без fields= приходит только id).
func (c *Client) entries(ctx context.Context, typ, fields string) ([]node, error) {
	var all []node
	for page := 1; page < 100000; page++ {
		q := url.Values{"compact": {"true"}, "fields": {fields}, "per_page": {"500"}, "page": {strconv.Itoa(page)}}
		var resp struct {
			Links   []node `json:"links"`
			Entries []node `json:"entries"`
		}
		if err := c.http.JSON(ctx, http.MethodGet, "/api/types/"+typ+"/instances?"+q.Encode(), nil, &resp); err != nil {
			return nil, err
		}
		for _, e := range resp.Entries {
			if content, ok := e["content"].(map[string]any); ok {
				all = append(all, content)
			}
		}
		hasNext := false
		for _, l := range resp.Links {
			if l["rel"] == "next" {
				hasNext = true
			}
		}
		if !hasNext {
			break
		}
	}
	return all, nil
}

// Extract строит полный снимок массива.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	logout, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	defer logout()

	sys, err := c.entries(ctx, "system", "name,model,serialNumber")
	if err != nil {
		return nil, err
	}
	if len(sys) == 0 {
		return nil, storage.Errorf(Kind, c.code, "массив не вернул сведений о системе")
	}
	info, err := c.entries(ctx, "basicSystemInfo", "softwareVersion")
	if err != nil {
		return nil, err
	}
	snap := storage.Snapshot{Array: storage.Array{Name: str(sys[0], "name"), Serial: str(sys[0], "serialNumber"),
		Family: strings.TrimSpace("Dell Unity " + str(sys[0], "model")), Vendor: map[string]any{"vendor": "Dell EMC"}}}
	if len(info) > 0 {
		snap.Array.Microcode = str(info[0], "softwareVersion")
	}
	if snap.Array.Name == "" {
		snap.Array.Name = c.code
	}
	if u, err := url.Parse(c.http.BaseURL); err == nil {
		snap.Array.MgmtIP = u.Hostname() // фильтр IP — в storage.Build
	}

	pools, err := c.entries(ctx, "pool", "id,name,sizeTotal")
	if err != nil {
		return nil, err
	}
	for _, p := range pools {
		snap.Pools = append(snap.Pools, storage.Pool{ID: str(p, "id"), Name: str(p, "name"), CapacityTB: num(p, "sizeTotal") / 1e12})
	}

	luns, err := c.entries(ctx, "lun", "id,name,sizeTotal,isThinEnabled,wwn,pool")
	if err != nil {
		return nil, err
	}
	for _, l := range luns {
		thin, _ := l["isThinEnabled"].(bool)
		snap.Volumes = append(snap.Volumes, storage.Volume{ID: "lun:" + str(l, "id"), Name: str(l, "name"), PoolID: refID(l, "pool"),
			WWN: storage.NormalizeWWN(str(l, "wwn")), SizeGB: num(l, "sizeTotal") / 1e9, Thin: &thin, Protocol: "block"})
	}

	nas, err := c.entries(ctx, "nasServer", "id,name")
	if err != nil {
		return nil, err
	}
	for _, n := range nas {
		snap.SVMs = append(snap.SVMs, storage.SVM{ID: str(n, "id"), Name: str(n, "name")})
	}
	fss, err := c.entries(ctx, "fileSystem", "id,name,sizeTotal,isThinEnabled,nasServer,pool")
	if err != nil {
		return nil, err
	}
	for _, f := range fss {
		thin, _ := f["isThinEnabled"].(bool)
		snap.Volumes = append(snap.Volumes, storage.Volume{ID: "fs:" + str(f, "id"), Name: str(f, "name"), PoolID: refID(f, "pool"), SVMID: refID(f, "nasServer"),
			SizeGB: num(f, "sizeTotal") / 1e9, Thin: &thin, Protocol: "nas"})
	}
	for _, spec := range []struct{ typ, kind, fsKey string }{{"cifsShare", "smb", "filesystem"}, {"nfsShare", "nfs", "filesystem"}} {
		shares, err := c.entries(ctx, spec.typ, "id,name,path,filesystem")
		if err != nil {
			return nil, err
		}
		for _, s := range shares {
			snap.Shares = append(snap.Shares, storage.Share{ID: spec.kind + ":" + str(s, "id"), Name: str(s, "name"), Type: spec.kind, Path: str(s, "path"),
				VolumeID: "fs:" + refID(s, spec.fsKey)})
		}
	}

	// презентация: инициаторы хостов (FC: "WWNN:WWPN"; iSCSI IQN отбрасываются нормализацией) и hostLUN (хост <-> LUN)
	inits, err := c.entries(ctx, "hostInitiator", "id,initiatorId,parentHost")
	if err != nil {
		return nil, err
	}
	hostWWPNs := map[string][]string{}
	for _, i := range inits {
		h := refID(i, "parentHost")
		hostWWPNs[h] = append(hostWWPNs[h], str(i, "initiatorId"))
	}
	hostLUNs, err := c.entries(ctx, "hostLUN", "id,host,lun")
	if err != nil {
		return nil, err
	}
	for _, hl := range hostLUNs {
		snap.Mappings = append(snap.Mappings, storage.Mapping{VolumeID: "lun:" + refID(hl, "lun"), WWPNs: hostWWPNs[refID(hl, "host")]})
	}
	snap.MappingsKnown = true

	sps, err := c.entries(ctx, "storageProcessor", "id,name,model,emcSerialNumber")
	if err != nil {
		return nil, err
	}
	for _, sp := range sps {
		snap.Controllers = append(snap.Controllers, storage.Component{ID: str(sp, "id"), Name: str(sp, "name"), Model: str(sp, "model"), Serial: str(sp, "emcSerialNumber")})
	}
	for _, typ := range []string{"dpe", "dae"} {
		encl, err := c.entries(ctx, typ, "id,name,model,emcSerialNumber")
		if err != nil {
			return nil, err
		}
		for _, e := range encl {
			snap.Enclosures = append(snap.Enclosures, storage.Component{ID: str(e, "id"), Name: str(e, "name"), Model: str(e, "model"), Serial: str(e, "emcSerialNumber")})
		}
	}
	disks, err := c.entries(ctx, "disk", "id,name,rawSize,diskTechnology,manufacturer,model,emcSerialNumber,slotNumber")
	if err != nil {
		return nil, err
	}
	for _, d := range disks {
		snap.Disks = append(snap.Disks, storage.Disk{ID: str(d, "id"), Serial: str(d, "emcSerialNumber"), Model: str(d, "model"), Manufacturer: str(d, "manufacturer"),
			Location: firstNonEmpty(str(d, "name"), str(d, "id")), Type: diskType(str(d, "diskTechnology")), CapacityGB: num(d, "rawSize") / 1e9})
	}

	return storage.Items(storage.Build(snap, c.code)), nil
}

// diskType — технология диска Unity (unityDiskTechnologyLabel: SAS, NL_SAS, SAS_FLASH_2…) в тип компонента.
func diskType(t string) string {
	t = strings.ToUpper(t)
	switch {
	case strings.Contains(t, "NVME"):
		return "nvme"
	case strings.Contains(t, "FLASH"), strings.Contains(t, "SSD"):
		return "ssd"
	}
	return "disk"
}

// refID — идентификатор вложенной ссылки: {"pool": {"id": "pool_1"}} -> "pool_1"; при compact=true
// ссылка может прийти и строкой.
func refID(n node, key string) string {
	switch v := n[key].(type) {
	case map[string]any:
		return str(v, "id")
	case string:
		return v
	}
	return ""
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
		f, _ := strconv.ParseFloat(v, 64)
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
