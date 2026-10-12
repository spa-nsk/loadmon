// Package openmanage — коннектор Dell OpenManage Enterprise REST API (plan.md E8.2): массовая
// инвентаризация серверов (и шасси PowerEdge MX, если есть) через один API, без обхода каждого
// iDRAC по отдельности — то же назначение, что у internal/importer/connectors/oneview для HPE.
//
// Формат ответов взят из общедоступной документации Dell OME REST API и НЕ подтверждён на живом
// устройстве (тот же статус, что у остальных коннекторов E5/E8 без доступа к стенду —
// harness/10-source-*.md): в первую очередь могут потребовать поправки числовые коды DeviceType
// (сервер/шасси) и точные имена полей InventoryDetails — при несовпадении сервер просто не
// распознаётся как шасси/сервер соответствующего типа, импорт не падает, компоненты которых нет в
// ответе просто не создаются.
package openmanage

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
)

// Kind — значение imp.connector.kind.
const Kind = "openmanage"

func init() { connector.Register(Kind, New) }

// Коды DeviceService.Devices[].Type — по документации OME REST API, не подтверждены.
const (
	deviceTypeServer  = 1000
	deviceTypeChassis = 2000
)

// Client — коннектор одного OpenManage Enterprise.
type Client struct {
	code     string
	http     *httpx.Client
	user     string
	password string
	pageSize int
}

// New создаёт коннектор. Настройки imp.connector.config: base_url, username, password_env, page_size
// (по умолчанию 50), timeout_sec (по умолчанию 60), insecure_skip_verify.
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("openmanage %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("openmanage %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	h := httpx.New(base, time.Duration(cfg.Int("timeout_sec", 60))*time.Second, cfg.Bool("insecure_skip_verify"))
	return &Client{code: cfg.Code, http: h, user: user, password: pass, pageSize: cfg.Int("page_size", 50)}, nil
}

// login открывает сессию (X-Auth-Token из заголовка ответа) и возвращает функцию завершения
// (ошибка выхода не критична — сессия и так истечёт сама).
func (c *Client) login(ctx context.Context) (logout func(), err error) {
	body := map[string]any{"UserName": c.user, "Password": c.password, "SessionType": "API"}
	var resp struct {
		ID int64 `json:"Id"`
	}
	headers, err := c.http.JSONHeaders(ctx, http.MethodPost, "/api/SessionService/Sessions", body, &resp)
	if err != nil {
		return nil, fmt.Errorf("openmanage %s: вход: %w", c.code, err)
	}
	token := headers.Get("X-Auth-Token")
	if token == "" {
		return nil, fmt.Errorf("openmanage %s: сервер не вернул X-Auth-Token", c.code)
	}
	c.http.Header.Set("X-Auth-Token", token)
	return func() {
		lctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = c.http.JSON(lctx, http.MethodDelete, fmt.Sprintf("/api/SessionService/Sessions(%d)", resp.ID), nil, nil)
	}, nil
}

// Test проверяет учётные данные, ничего не загружая.
func (c *Client) Test(ctx context.Context) error {
	logout, err := c.login(ctx)
	if err != nil {
		return err
	}
	logout()
	return nil
}

// page — форма ответа OData-коллекций OME.
type page struct {
	Count    int              `json:"@odata.count"`
	Value    []map[string]any `json:"value"`
	NextLink string           `json:"@odata.nextLink"`
}

// collect читает коллекцию целиком, следуя @odata.nextLink (path в нём — уже без базового URL).
func (c *Client) collect(ctx context.Context, path string) ([]map[string]any, error) {
	next := path
	var all []map[string]any
	for i := 0; next != "" && i < 100000; i++ {
		var p page
		if err := c.http.JSON(ctx, http.MethodGet, next, nil, &p); err != nil {
			return nil, err
		}
		all = append(all, p.Value...)
		next = p.NextLink
	}
	return all, nil
}

// inventoryDetails читает InventoryDetails устройства и раскладывает по InventoryType —
// одним запросом, не по одному на каждый тип компонента.
func (c *Client) inventoryDetails(ctx context.Context, deviceID int64) (map[string][]map[string]any, error) {
	var resp struct {
		Value []struct {
			InventoryType string           `json:"InventoryType"`
			InventoryInfo []map[string]any `json:"InventoryInfo"`
		} `json:"value"`
	}
	path := fmt.Sprintf("/api/DeviceService/Devices(%d)/InventoryDetails", deviceID)
	if err := c.http.JSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("openmanage %s: инвентарь устройства %d: %w", c.code, deviceID, err)
	}
	out := make(map[string][]map[string]any, len(resp.Value))
	for _, v := range resp.Value {
		out[v.InventoryType] = v.InventoryInfo
	}
	return out, nil
}

// Extract строит полный снимок: шасси (если есть) и серверы со связью «содержит», компоненты и
// BMC каждого сервера — из InventoryDetails.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	logout, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	defer logout()

	devices, err := c.collect(ctx, fmt.Sprintf("/api/DeviceService/Devices?$top=%d", c.pageSize))
	if err != nil {
		return nil, fmt.Errorf("openmanage %s: список устройств: %w", c.code, err)
	}

	var recs []canon.Record
	chassisIdx := map[string]int{} // service tag шасси -> индекс в recs

	for _, d := range devices {
		if num(d, "Type") != deviceTypeChassis {
			continue
		}
		tag := str(d, "DeviceServiceTag")
		if tag == "" {
			continue
		}
		rec := canon.Record{
			Class: "blade_chassis", SourceName: str(d, "DeviceName"), ExternalID: "chassis:" + tag, SerialNumber: tag,
			Identifiers: map[string]string{"serial_number": tag}, Attrs: map[string]any{},
			Vendor: map[string]any{"model": str(d, "Model")},
		}
		chassisIdx[tag] = len(recs)
		recs = append(recs, rec)
	}

	for _, d := range devices {
		if num(d, "Type") != deviceTypeServer {
			continue
		}
		id := int64(num(d, "Id"))
		tag := str(d, "DeviceServiceTag")
		chassisTag := str(d, "ChassisServiceTag")
		_, isBlade := chassisIdx[chassisTag]

		inv, err := c.inventoryDetails(ctx, id)
		if err != nil {
			return nil, err
		}

		attrs := map[string]any{"form_factor": "rack"}
		if isBlade {
			attrs["form_factor"] = "blade"
			if slot := int(num(d, "SlotNumber")); slot > 0 {
				attrs["bay"] = strconv.Itoa(slot)
			}
		}
		var comps []canon.Component

		cpuCount, coreTotal := 0, 0
		for _, p := range inv["serverProcessors"] {
			cpuCount++
			cores := int(num(p, "NumberOfCores"))
			coreTotal += cores
			slot := firstNonEmpty(str(p, "SlotNumber"), strconv.Itoa(cpuCount))
			comps = append(comps, canon.Component{Key: "cpu:" + slot, Type: "cpu", Slot: slot,
				Manufacturer: str(p, "Manufacturer"), PartNumber: str(p, "ModelName"), Speed: str(p, "MaxSpeed")})
		}
		if cpuCount > 0 {
			attrs["cpu_count"] = cpuCount
			attrs["core_count"] = coreTotal
		}

		ramGB := 0
		for _, m := range inv["serverMemoryDevices"] {
			slot := firstNonEmpty(str(m, "BankName"), str(m, "Name"))
			mb := int(num(m, "Size"))
			if mb <= 0 {
				continue
			}
			gib := mb / 1024
			ramGB += gib
			serial := str(m, "SerialNumber")
			comps = append(comps, canon.Component{Key: "dimm:" + firstNonEmpty(serial, slot), Type: "dimm", Slot: slot,
				Manufacturer: str(m, "Manufacturer"), PartNumber: str(m, "PartNumber"), SerialNumber: serial,
				CapacityValue: fptr(float64(gib)), CapacityUnit: "GiB", Speed: str(m, "Speed")})
		}
		if ramGB > 0 {
			attrs["ram_gb"] = ramGB
		}

		for _, disk := range inv["serverArrayDisks"] {
			typ := "disk"
			if strings.EqualFold(str(disk, "MediaType"), "SSD") {
				typ = "ssd"
			}
			slot := str(disk, "Slot")
			serial := str(disk, "SerialNo")
			comps = append(comps, canon.Component{Key: "drive:" + firstNonEmpty(serial, slot), Type: typ, Slot: slot,
				Manufacturer: str(disk, "Manufacturer"), PartNumber: str(disk, "ProductId"), SerialNumber: serial,
				Firmware: str(disk, "Revision"), CapacityValue: fptr(num(disk, "Capacity") / 1e9), CapacityUnit: "GB"})
		}

		for _, psu := range inv["serverPowerSupplies"] {
			slot := str(psu, "Name")
			serial := str(psu, "SerialNumber")
			comps = append(comps, canon.Component{Key: "psu:" + firstNonEmpty(serial, slot), Type: "psu", Slot: slot,
				Manufacturer: str(psu, "Manufacturer"), PartNumber: str(psu, "PartNumber"), SerialNumber: serial})
		}

		if bios := firstOrNil(inv["serverBIOS"]); bios != nil {
			if v := str(bios, "Version"); v != "" {
				attrs["firmware"] = v
			}
		}
		if os := firstOrNil(inv["osInformation"]); os != nil {
			if name := str(os, "OsName"); name != "" {
				attrs["os_distro"] = name
				lower := strings.ToLower(name)
				switch {
				case strings.Contains(lower, "esxi") || strings.Contains(lower, "vmware"):
					attrs["os_family"] = "Linux"
					attrs["hypervisor_role"] = "vmware_esxi"
				case strings.Contains(lower, "hyper-v") || strings.Contains(lower, "windows"):
					attrs["os_family"] = "Windows"
				case strings.Contains(lower, "linux") || strings.Contains(lower, "rhel") || strings.Contains(lower, "ubuntu"):
					attrs["os_family"] = "Linux"
				}
			}
		}

		bmcIP := ""
		for _, m := range list(d, "DeviceManagement") {
			if ip := str(m, "NetworkAddress"); ip != "" {
				bmcIP = ip
				break
			}
		}
		if bmcIP != "" {
			attrs["bmc_ip"] = bmcIP
		}

		rec := canon.Record{
			Class: "server", SourceName: str(d, "DeviceName"), ExternalID: "srv:" + tag, SerialNumber: tag,
			Identifiers: map[string]string{}, Attrs: attrs, Components: comps,
			Vendor: map[string]any{"model": str(d, "Model"), "id": id},
		}
		if tag != "" {
			rec.Identifiers["serial_number"] = tag
		}
		idx := len(recs)
		recs = append(recs, rec)
		if isBlade {
			ci := chassisIdx[chassisTag]
			recs[ci].Relations = append(recs[ci].Relations, canon.Relation{Type: "contains", TargetExternalID: recs[idx].ExternalID})
		}
	}

	items := make([]canon.Item, len(recs))
	for i := range recs {
		items[i] = canon.Item{Seq: i + 1, Record: recs[i]}
	}
	return items, nil
}

// --- разбор нетипизированных ответов (тот же приём, что в internal/importer/connectors/oneview) ---

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

func num(m map[string]any, k string) float64 {
	switch v := m[k].(type) {
	case float64:
		return v
	case string:
		n, _ := strconv.ParseFloat(v, 64)
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

func firstOrNil(items []map[string]any) map[string]any {
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func fptr(f float64) *float64 { return &f }
