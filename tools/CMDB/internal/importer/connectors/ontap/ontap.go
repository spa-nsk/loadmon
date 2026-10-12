// Package ontap — коннектор NetApp ONTAP REST (9.6+; ZAPI не используется), plan.md E5.3: кластер,
// агрегаты, SVM, тома, LUN, qtree, ресурсы CIFS и экспорты NFS, узлы (контроллеры), полки и диски.
// Разбор ответов в общий storage.Snapshot; записи строит storage.Build.
//
// Формат ответов взят из документации ONTAP REST и НЕ подтверждён на живом кластере
// (harness/14-source-storage.md).
package ontap

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
	"cmdb/internal/importer/connectors/storage"
)

// Kind — значение imp.connector.kind.
const Kind = "ontap"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного кластера ONTAP.
type Client struct {
	code string
	http *httpx.Client
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://cluster-mgmt), username,
// password_env (имя переменной окружения), timeout_sec (на запрос, 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("ontap %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("ontap %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	h.User, h.Password = user, pass
	return &Client{code: cfg.Code, http: h}, nil
}

type rec = map[string]any

// Test проверяет учётные данные запросом кластера.
func (c *Client) Test(ctx context.Context) error {
	var out rec
	return c.http.JSON(ctx, http.MethodGet, "/api/cluster?fields=name", nil, &out)
}

// records читает коллекцию целиком, следуя _links.next.href; ошибка любой страницы — ошибка снимка.
func (c *Client) records(ctx context.Context, path string, fields string, extra url.Values) ([]rec, error) {
	q := url.Values{"fields": {fields}, "max_records": {"500"}}
	for k, v := range extra {
		q[k] = v
	}
	next := path + "?" + q.Encode()
	var all []rec
	for i := 0; next != "" && i < 100000; i++ {
		var page struct {
			Records []rec `json:"records"`
			Links   rec   `json:"_links"`
		}
		if err := c.http.JSON(ctx, http.MethodGet, next, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Records...)
		next = ""
		if nx, _ := page.Links["next"].(map[string]any); nx != nil {
			next, _ = nx["href"].(string)
		}
	}
	return all, nil
}

// Extract строит полный снимок кластера.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	var cluster rec
	if err := c.http.JSON(ctx, http.MethodGet, "/api/cluster?fields=name,uuid,version,management_interfaces", nil, &cluster); err != nil {
		return nil, err
	}
	snap := storage.Snapshot{Array: storage.Array{Name: str(cluster, "name"), Microcode: nested(cluster, "version", "full"),
		Vendor: map[string]any{"vendor": "NetApp", "cluster_uuid": str(cluster, "uuid")}}}
	if snap.Array.Name == "" {
		return nil, storage.Errorf(Kind, c.code, "кластер не вернул имя")
	}
	if mi, _ := cluster["management_interfaces"].([]any); len(mi) > 0 {
		m, _ := mi[0].(map[string]any)
		snap.Array.MgmtIP = nested(m, "ip", "address")
	}

	nodes, err := c.records(ctx, "/api/cluster/nodes", "name,uuid,serial_number,model,version", nil)
	if err != nil {
		return nil, err
	}
	for _, n := range nodes {
		snap.Controllers = append(snap.Controllers, storage.Component{ID: str(n, "uuid"), Name: str(n, "name"), Serial: str(n, "serial_number"),
			Model: str(n, "model"), Firmware: nested(n, "version", "full")})
		if snap.Array.Family == "" {
			snap.Array.Family = strings.TrimSpace("NetApp " + str(n, "model"))
		}
	}

	aggrs, err := c.records(ctx, "/api/storage/aggregates", "name,uuid,space,block_storage", nil)
	if err != nil {
		return nil, err
	}
	for _, a := range aggrs {
		snap.Pools = append(snap.Pools, storage.Pool{ID: str(a, "uuid"), Name: str(a, "name"),
			RAID:       nested(a, "block_storage", "primary", "raid_type"),
			Tier:       nested(a, "block_storage", "storage_type"),
			CapacityTB: bytesTo(num(sub(sub(a, "space"), "block_storage"), "size"), 1e12)})
	}

	svms, err := c.records(ctx, "/api/svm/svms", "name,uuid,ip_interfaces", nil)
	if err != nil {
		return nil, err
	}
	svmName := map[string]string{}
	for _, s := range svms {
		v := storage.SVM{ID: str(s, "uuid"), Name: str(s, "name")}
		if ifs, _ := s["ip_interfaces"].([]any); len(ifs) > 0 {
			m, _ := ifs[0].(map[string]any)
			v.MgmtIP = nested(m, "ip", "address")
		}
		svmName[v.ID] = v.Name
		snap.SVMs = append(snap.SVMs, v)
	}

	vols, err := c.records(ctx, "/api/storage/volumes", "name,uuid,size,svm,aggregates,style,guarantee,nas", url.Values{"is_svm_root": {"false"}})
	if err != nil {
		return nil, err
	}
	volName := map[string]string{}
	for _, v := range vols {
		id := str(v, "uuid")
		vol := storage.Volume{ID: id, Name: str(v, "name"), SVMID: nested(v, "svm", "uuid"), SizeGB: bytesTo(num(v, "size"), 1e9)}
		vol.Name = str(nestedRec(v, "svm"), "name") + ":" + str(v, "name")
		if aggs, _ := v["aggregates"].([]any); len(aggs) > 0 {
			a, _ := aggs[0].(map[string]any)
			vol.PoolID = str(a, "uuid")
		}
		thin := nested(v, "guarantee", "type") == "none"
		vol.Thin = &thin
		volName[id] = str(v, "name")
		if p := nested(v, "nas", "path"); p != "" {
			vol.Protocol = "nas"
			snap.Shares = append(snap.Shares, storage.Share{ID: "nfs:" + id, Name: str(nestedRec(v, "svm"), "name") + ":" + p,
				SVMID: vol.SVMID, VolumeID: id, Type: "nfs", Path: p})
		}
		snap.Volumes = append(snap.Volumes, vol)
	}

	luns, err := c.records(ctx, "/api/storage/luns", "name,uuid,serial_number,space,svm,location,os_type", nil)
	if err != nil {
		return nil, err
	}
	for _, l := range luns {
		vol := storage.Volume{ID: "lun:" + str(l, "uuid"), Name: str(nestedRec(l, "svm"), "name") + ":" + str(l, "name"),
			SVMID: nested(l, "svm", "uuid"), SizeGB: bytesTo(num(nestedRec(l, "space"), "size"), 1e9), Protocol: "block", WWN: naaFromSerial(str(l, "serial_number"))}
		snap.Volumes = append(snap.Volumes, vol)
	}

	qtrees, err := c.records(ctx, "/api/storage/qtrees", "name,id,volume,svm", nil)
	if err != nil {
		return nil, err
	}
	for _, q := range qtrees {
		if str(q, "name") == "" {
			continue // корневой (безымянный) qtree тома — не отдельный объект
		}
		volID := nested(q, "volume", "uuid")
		snap.Qtrees = append(snap.Qtrees, storage.Qtree{ID: volID + "/" + str(q, "id"), Name: str(nestedRec(q, "svm"), "name") + ":" + volName[volID] + "/" + str(q, "name"),
			SVMID: nested(q, "svm", "uuid"), VolumeID: volID, Path: "/vol/" + volName[volID] + "/" + str(q, "name")})
	}

	cifs, err := c.records(ctx, "/api/protocols/cifs/shares", "name,path,svm", nil)
	if err != nil {
		return nil, err
	}
	for _, s := range cifs {
		if str(s, "name") == "" || strings.HasSuffix(str(s, "name"), "$") {
			continue // административные ресурсы (c$, admin$, ipc$)
		}
		snap.Shares = append(snap.Shares, storage.Share{ID: "cifs:" + nested(s, "svm", "uuid") + "/" + str(s, "name"),
			Name: str(nestedRec(s, "svm"), "name") + ":" + str(s, "name"), SVMID: nested(s, "svm", "uuid"), Type: "smb", Path: str(s, "path")})
	}

	// презентация LUN: igroup (инициаторы) и lun-maps (LUN <-> igroup); WWPN — FC-инициаторы, IQN iSCSI отбрасываются
	igroups, err := c.records(ctx, "/api/protocols/san/igroups", "uuid,name,initiators", nil)
	if err != nil {
		return nil, err
	}
	wwpns := map[string][]string{}
	for _, g := range igroups {
		inits, _ := g["initiators"].([]any)
		for _, i := range inits {
			m, _ := i.(map[string]any)
			wwpns[str(g, "uuid")] = append(wwpns[str(g, "uuid")], str(m, "name"))
		}
	}
	lunMaps, err := c.records(ctx, "/api/protocols/san/lun-maps", "lun,igroup", nil)
	if err != nil {
		return nil, err
	}
	for _, m := range lunMaps {
		snap.Mappings = append(snap.Mappings, storage.Mapping{VolumeID: "lun:" + nested(m, "lun", "uuid"), WWPNs: wwpns[nested(m, "igroup", "uuid")]})
	}
	snap.MappingsKnown = true

	shelves, err := c.records(ctx, "/api/storage/shelves", "uid,name,model,serial_number", nil)
	if err != nil {
		return nil, err
	}
	for _, s := range shelves {
		snap.Enclosures = append(snap.Enclosures, storage.Component{ID: str(s, "uid"), Name: str(s, "name"), Model: str(s, "model"), Serial: str(s, "serial_number")})
	}
	disks, err := c.records(ctx, "/api/storage/disks", "name,serial_number,model,type,usable_size,firmware_version,vendor,bay,shelf", nil)
	if err != nil {
		return nil, err
	}
	for _, d := range disks {
		snap.Disks = append(snap.Disks, storage.Disk{ID: str(d, "name"), Serial: str(d, "serial_number"), Model: str(d, "model"), Manufacturer: str(d, "vendor"),
			Firmware: str(d, "firmware_version"), Location: str(d, "name"), EnclosureID: nested(d, "shelf", "uid"), Type: diskType(str(d, "type")),
			CapacityGB: bytesTo(num(d, "usable_size"), 1e9)})
	}

	return storage.Items(storage.Build(snap, c.code)), nil
}

func diskType(t string) string {
	t = strings.ToLower(t)
	switch {
	case strings.Contains(t, "nvm"):
		return "nvme"
	case strings.Contains(t, "ssd"):
		return "ssd"
	}
	return "disk"
}

// naaFromSerial — NAA-идентификатор LUN ONTAP: 600a0980 + серийный номер LUN в hex (документированный
// формат NetApp). "" — серийного номера нет.
func naaFromSerial(serial string) string {
	if serial == "" {
		return ""
	}
	return "600a0980" + hex.EncodeToString([]byte(serial))
}

func bytesTo(b float64, unit float64) float64 { return b / unit }

func str(m rec, k string) string {
	switch v := m[k].(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		return fmt.Sprint(int64(v))
	}
	return ""
}

func num(m rec, k string) float64 {
	f, _ := m[k].(float64)
	return f
}

func nestedRec(m rec, k string) rec {
	r, _ := m[k].(map[string]any)
	return r
}

func sub(m rec, k string) rec { return nestedRec(m, k) }

func nested(m rec, keys ...string) string {
	cur := m
	for i, k := range keys {
		if i == len(keys)-1 {
			return str(cur, k)
		}
		if cur = nestedRec(cur, k); cur == nil {
			return ""
		}
	}
	return ""
}
