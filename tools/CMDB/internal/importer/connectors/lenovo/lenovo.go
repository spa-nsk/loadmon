// Package lenovo — коннектор Lenovo ThinkSystem DE (NetApp SANtricity, встроенный REST контроллера):
// массив, контроллеры, полки, диски, группы томов и дисковые пулы, тома (harness/03, «Lenovo DE»).
//
// ✅ (loadmon, 54 СХД): база /devmgr/v2/storage-systems/1/…, ВСЯ инвентаризация полок и дисков —
// одним GET …/graph (Graph.Tray, Graph.Drive); отдельных drives/trays нет. ✅ Типы полей в ответах
// расходятся с документацией (GatewayIP, SpindleSpeedMatch, ExpectedProtectionInformationAppTag) —
// поэтому разбор нетипизированный и терпимый к числам-строкам. Имена полей graph сверить на пилоте
// (harness/14-source-storage.md).
package lenovo

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
	"cmdb/internal/importer/connectors/storage"
)

// Kind — значение imp.connector.kind.
const Kind = "lenovo"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного массива.
type Client struct {
	code     string
	http     *httpx.Client
	systemID string
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://controller:8443),
// username, password_env, system_id ("1" для встроенного API), timeout_sec (60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("lenovo %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("lenovo %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	sid := cfg.String("system_id")
	if sid == "" {
		sid = "1"
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	h.User, h.Password = user, pass
	return &Client{code: cfg.Code, http: h, systemID: sid}, nil
}

type node = map[string]any

func (c *Client) path(suffix string) string {
	return "/devmgr/v2/storage-systems/" + c.systemID + suffix
}

// Test проверяет учётные данные запросом состояния системы.
func (c *Client) Test(ctx context.Context) error {
	var out node
	return c.http.JSON(ctx, http.MethodGet, "/devmgr/v2/storage-systems/"+c.systemID, nil, &out)
}

// Extract строит снимок одним запросом graph.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	var g node
	if err := c.http.JSON(ctx, http.MethodGet, c.path("/graph"), nil, &g); err != nil {
		return nil, err
	}
	sa := sub(g, "sa")
	saData := sub(sa, "saData")
	ctrls := list(g, "controller")

	snap := storage.Snapshot{Array: storage.Array{
		Name:      firstNonEmpty(str(saData, "storageArrayLabel"), str(sa, "storageArrayLabel"), str(g, "storageArrayLabel"), c.code),
		Serial:    firstNonEmpty(str(sa, "chassisSerialNumber"), str(saData, "chassisSerialNumber")),
		Microcode: firstNonEmpty(str(sa, "fwVersion"), str(sa, "bundleVersion")),
		Vendor:    map[string]any{"vendor": "Lenovo", "platform": "DE (SANtricity)"},
	}}
	if len(ctrls) > 0 {
		model := firstNonEmpty(str(ctrls[0], "modelName"), str(sa, "modelName"))
		snap.Array.Family = strings.TrimSpace("Lenovo ThinkSystem " + model)
		if snap.Array.Microcode == "" {
			snap.Array.Microcode = str(ctrls[0], "appVersion")
		}
	}

	for _, x := range ctrls {
		snap.Controllers = append(snap.Controllers, storage.Component{ID: refOf(x), Name: firstNonEmpty(str(sub(x, "physicalLocation"), "label"), refOf(x)),
			Model: str(x, "modelName"), Serial: str(x, "serialNumber"), Firmware: str(x, "appVersion")})
	}
	trayKnown := map[string]bool{}
	for _, tr := range list(g, "tray") {
		id := refOf(tr)
		trayKnown[id] = true
		snap.Enclosures = append(snap.Enclosures, storage.Component{ID: id, Name: "Tray " + str(tr, "trayId"), Model: str(tr, "partNumber"), Serial: str(tr, "serialNumber")})
	}
	for _, d := range list(g, "drive") {
		loc := sub(d, "physicalLocation")
		tray := firstNonEmpty(str(loc, "trayRef"), str(d, "trayRef"))
		typ := "disk"
		switch strings.ToLower(str(d, "driveMediaType")) {
		case "ssd":
			typ = "ssd"
		}
		if strings.EqualFold(str(sub(d, "interfaceType"), "driveType"), "nvme") {
			typ = "nvme"
		}
		snap.Disks = append(snap.Disks, storage.Disk{ID: refOf(d), Serial: str(d, "serialNumber"), Model: str(d, "productID"), Manufacturer: firstNonEmpty(str(d, "manufacturer"), str(d, "vendorId")),
			Firmware: firstNonEmpty(str(d, "firmwareVersion"), str(d, "softwareVersion")), Location: "slot " + str(loc, "slot"), EnclosureID: tray, Type: typ,
			CapacityGB: num(d, "rawCapacity") / 1e9})
	}

	for _, key := range []string{"volumeGroup", "diskPool"} {
		for _, p := range list(g, key) {
			snap.Pools = append(snap.Pools, storage.Pool{ID: refOf(p), Name: firstNonEmpty(str(p, "label"), str(p, "name")), RAID: str(p, "raidLevel"),
				CapacityTB: num(p, "totalRaidedSpace") / 1e12})
		}
	}
	vols := list(g, "volume")
	vols = append(vols, list(g, "thinVolume")...)
	for _, v := range vols {
		thin := boolOf(v, "thinProvisioned") || v["thinVolume"] == true
		vol := storage.Volume{ID: refOf(v), Name: firstNonEmpty(str(v, "label"), str(v, "name")), PoolID: firstNonEmpty(str(v, "volumeGroupRef"), str(v, "diskPoolRef")),
			WWN: storage.NormalizeWWN(str(v, "worldWideName")), SizeGB: num(v, "capacity") / 1e9, Thin: &thin, Protocol: "block"}
		if lun, err := strconv.Atoi(str(v, "volumeHandle")); err == nil {
			vol.LunID = &lun
		}
		snap.Volumes = append(snap.Volumes, vol)
	}

	// Презентация: volume-mappings (том <-> host либо группа хостов) и hosts (инициаторы FC).
	var mappings, hosts []node
	if err := c.http.JSON(ctx, http.MethodGet, c.path("/volume-mappings"), nil, &mappings); err != nil {
		return nil, err
	}
	if err := c.http.JSON(ctx, http.MethodGet, c.path("/hosts"), nil, &hosts); err != nil {
		return nil, err
	}
	wwpns := map[string][]string{} // hostRef и clusterRef (группа) -> WWPN
	for _, h := range hosts {
		var ws []string
		inits, _ := h["initiators"].([]any)
		for _, i := range inits {
			m, _ := i.(map[string]any)
			if nn := sub(m, "nodeName"); strings.EqualFold(str(nn, "ioInterfaceType"), "fc") {
				ws = append(ws, str(nn, "remoteNodeWWN"))
			}
		}
		wwpns[str(h, "hostRef")] = append(wwpns[str(h, "hostRef")], ws...)
		if cl := str(h, "clusterRef"); cl != "" {
			wwpns[cl] = append(wwpns[cl], ws...)
		}
	}
	for _, m := range mappings {
		snap.Mappings = append(snap.Mappings, storage.Mapping{VolumeID: str(m, "volumeRef"), WWPNs: wwpns[str(m, "mapRef")]})
	}
	snap.MappingsKnown = true

	return storage.Items(storage.Build(snap, c.code)), nil
}

// refOf — идентификатор объекта SANtricity: id, иначе *Ref-поле.
func refOf(n node) string {
	if v := str(n, "id"); v != "" {
		return v
	}
	for _, k := range []string{"controllerRef", "driveRef", "trayRef", "volumeRef", "volumeGroupRef", "diskPoolRef"} {
		if v := str(n, k); v != "" {
			return v
		}
	}
	return ""
}

func list(n node, key string) []node {
	raw, _ := n[key].([]any)
	out := make([]node, 0, len(raw))
	for _, r := range raw {
		if m, ok := r.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func sub(n node, key string) node {
	m, _ := n[key].(map[string]any)
	return m
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

// num — число из числа или строки (SANtricity отдаёт ёмкость строкой байт).
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

func boolOf(n node, k string) bool {
	switch v := n[k].(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
