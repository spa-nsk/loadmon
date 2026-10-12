// Package powerstore — коннектор Dell PowerStore REST API (plan.md E8.1): кластер/appliance как
// массив, тома, физические диски, презентация томов инициаторам (LUN masking).
//
// Отличие от Dell Unity (internal/importer/connectors/unity, тот же вендор, другая линейка): у
// PowerStore нет пользовательских пулов — весь кластер использует общее тонкое пространство
// (документированная архитектура, не «не проверено»), поэтому storage.Snapshot.Pools не
// заполняется — тома привязываются напрямую к массиву (storage.Build уже это умеет, см. default
// case). Фильтр запроса PowerStore — вида field=op.value (например type=eq.Primary), в отличие от
// OData ($top/$skip) у OME/OneView или compact/fields у Unity — используется, чтобы не тянуть
// снимки/клоны томов (volume.type) в тела CMDB.
//
// Аутентификация — HTTP Basic на каждый запрос (официально поддерживаемый простой режим PowerStore
// REST API; сессия с CSRF, как у Unity, не нужна — коннектор делает только GET).
//
// Формат ответов взят из публичной документации PowerStore REST API и НЕ подтверждён на живом
// массиве (тот же статус, что у всех коннекторов E5/E8 без доступа к стенду) — в первую очередь
// могут потребовать поправки точные имена полей hardware (диски).
package powerstore

import (
	"context"
	"fmt"
	"net/http"
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
const Kind = "powerstore"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного кластера PowerStore.
type Client struct {
	code string
	http *httpx.Client
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://powerstore.example.local),
// username, password_env (имя переменной окружения), timeout_sec (на запрос, 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("powerstore %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("powerstore %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	h.User, h.Password = user, pass
	return &Client{code: cfg.Code, http: h}, nil
}

type node = map[string]any

// Test проверяет адрес и учётные данные пробным запросом кластера.
func (c *Client) Test(ctx context.Context) error {
	_, err := c.list(ctx, "cluster", "id,name")
	return err
}

// list читает коллекцию ресурса целиком (PowerStore не отдаёт коллекции постранично по умолчанию —
// ограничение через limit/offset нужно только для очень больших списков, здесь не требуется).
func (c *Client) list(ctx context.Context, res, fields string, filters ...string) ([]node, error) {
	q := url.Values{"select": {fields}}
	for _, f := range filters {
		kv := strings.SplitN(f, "=", 2)
		if len(kv) == 2 {
			q.Set(kv[0], kv[1])
		}
	}
	var out []node
	if err := c.http.JSON(ctx, http.MethodGet, "/api/rest/"+res+"?"+q.Encode(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Extract строит полный снимок кластера.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	clusters, err := c.list(ctx, "cluster", "id,name")
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, storage.Errorf(Kind, c.code, "кластер не вернул сведений о себе")
	}
	appliances, err := c.list(ctx, "appliance", "id,name,service_tag,model")
	if err != nil {
		return nil, err
	}

	snap := storage.Snapshot{Array: storage.Array{Name: str(clusters[0], "name"), Vendor: map[string]any{"vendor": "Dell EMC"}}}
	if snap.Array.Name == "" {
		snap.Array.Name = c.code
	}
	if len(appliances) > 0 {
		snap.Array.Serial = str(appliances[0], "service_tag")
		snap.Array.Family = strings.TrimSpace("Dell PowerStore " + str(appliances[0], "model"))
	}
	if u, err := url.Parse(c.http.BaseURL); err == nil {
		snap.Array.MgmtIP = u.Hostname()
	}

	// type=eq.Primary — исключает снимки и клоны томов (volume.type), которые PowerStore отдаёт
	// в той же коллекции.
	vols, err := c.list(ctx, "volume", "id,name,size,wwn,is_thin_enabled", "type=eq.Primary")
	if err != nil {
		return nil, err
	}
	for _, v := range vols {
		thin, _ := v["is_thin_enabled"].(bool)
		snap.Volumes = append(snap.Volumes, storage.Volume{ID: str(v, "id"), Name: str(v, "name"),
			WWN: storage.NormalizeWWN(str(v, "wwn")), SizeGB: num(v, "size") / 1e9, Thin: &thin, Protocol: "block"})
	}

	drives, err := c.list(ctx, "hardware", "id,name,slot,serial_number,manufacturer,part_number,drive_type,size", "type=eq.Drive")
	if err != nil {
		return nil, err
	}
	for _, d := range drives {
		snap.Disks = append(snap.Disks, storage.Disk{ID: str(d, "id"), Serial: str(d, "serial_number"), Model: str(d, "part_number"),
			Manufacturer: str(d, "manufacturer"), Location: firstNonEmpty(str(d, "slot"), str(d, "name"), str(d, "id")),
			Type: driveType(str(d, "drive_type")), CapacityGB: num(d, "size") / 1e9})
	}

	// презентация: инициаторы хостов (FC: port_name — WWPN; iSCSI IQN отбрасываются нормализацией)
	// и host_volume_mapping (хост <-> том).
	inits, err := c.list(ctx, "host_initiator", "id,host_id,port_name")
	if err != nil {
		return nil, err
	}
	hostWWPNs := map[string][]string{}
	for _, i := range inits {
		h := str(i, "host_id")
		hostWWPNs[h] = append(hostWWPNs[h], str(i, "port_name"))
	}
	maps, err := c.list(ctx, "host_volume_mapping", "id,host_id,volume_id")
	if err != nil {
		return nil, err
	}
	for _, m := range maps {
		snap.Mappings = append(snap.Mappings, storage.Mapping{VolumeID: str(m, "volume_id"), WWPNs: hostWWPNs[str(m, "host_id")]})
	}
	snap.MappingsKnown = true

	return storage.Items(storage.Build(snap, c.code)), nil
}

// driveType — drive_type PowerStore (SAS, SAS_FLASH, NVMe…) в тип компонента.
func driveType(t string) string {
	t = strings.ToUpper(t)
	switch {
	case strings.Contains(t, "NVME"):
		return "nvme"
	case strings.Contains(t, "FLASH"), strings.Contains(t, "SSD"):
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
