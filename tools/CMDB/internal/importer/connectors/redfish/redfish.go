// Package redfish — коннектор Redfish (DMTF) для BMC серверов (plan.md E5.2): iLO, iDRAC, XClarity,
// Supermicro, YADRO, Aquarius и др. Каждый BMC — отдельная цель (targets) с общими или своими
// учётными данными. Из каждого BMC читается сервер (Systems), его компоненты (процессоры, память,
// диски, контроллеры, блоки питания, вентиляторы, BMC), сетевые интерфейсы и MAC BMC.
//
// Недоступный BMC — ошибка ЕГО записи (Item.Retain), а не пропажа сервера: КЕ не помечается
// not_discovered (harness/04, урок 3). Если недоступны все BMC — ошибка снимка целиком.
//
// Реализовано на net/http и разборе ресурсов по @odata.id, а не на gofish: у разных вендоров
// схемы расходятся, а нетипизированный обход даёт деградацию до поддерживаемого, а не отказ на
// неизвестном поле. Формат из спецификации DMTF, НЕ проверен на живых BMC (harness/13-source-redfish.md).
package redfish

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/httpx"
)

// Kind — значение imp.connector.kind.
const Kind = "redfish"

func init() { connector.Register(Kind, New) }

type target struct {
	base     string // https://host[:port]
	host     string // host[:port] — ключ внешнего идентификатора
	user     string
	password string
}

// Client — коннектор набора BMC.
type Client struct {
	code        string
	targets     []target
	concurrency int
	timeout     time.Duration
	insecure    bool
}

// New создаёт коннектор. Настройки imp.connector.config: targets — список BMC (строка "10.0.0.5" /
// "https://bmc.local" либо объект {"address" | "url", "username", "password_env"}); username +
// password_env — общие учётные данные (переопределяются в объекте цели); concurrency (по умолчанию
// 8), timeout_sec (на запрос, 30), insecure_skip_verify (BMC почти всегда с самоподписанным сертификатом).
func New(cfg connector.Config) (connector.Connector, error) {
	raw, _ := cfg.Settings["targets"].([]any)
	if len(raw) == 0 {
		return nil, fmt.Errorf("redfish %s: не задан список targets", cfg.Code)
	}
	sharedUser := cfg.String("username")
	sharedPass, sharedErr := cfg.Secret("password_env")

	c := &Client{code: cfg.Code, concurrency: max(1, cfg.Int("concurrency", 8)),
		timeout: time.Duration(cfg.Int("timeout_sec", 30)) * time.Second, insecure: cfg.Bool("insecure_skip_verify")}
	seen := map[string]bool{}
	for i, r := range raw {
		var addr, user, passEnv string
		switch v := r.(type) {
		case string:
			addr = v
		case map[string]any:
			addr, _ = v["url"].(string)
			if addr == "" {
				addr, _ = v["address"].(string)
			}
			user, _ = v["username"].(string)
			passEnv, _ = v["password_env"].(string)
		}
		if strings.TrimSpace(addr) == "" {
			return nil, fmt.Errorf("redfish %s: цель #%d без адреса", cfg.Code, i+1)
		}
		t, err := parseTarget(addr)
		if err != nil {
			return nil, fmt.Errorf("redfish %s: цель %q: %w", cfg.Code, addr, err)
		}
		if seen[t.host] {
			return nil, fmt.Errorf("redfish %s: цель %s указана дважды", cfg.Code, t.host)
		}
		seen[t.host] = true
		t.user, t.password = sharedUser, sharedPass
		if user != "" {
			t.user = user
		}
		if passEnv != "" {
			pw, err := connector.Config{Code: cfg.Code, Settings: map[string]any{"p": passEnv}}.Secret("p")
			if err != nil {
				return nil, err
			}
			t.password = pw
		} else if sharedErr != nil {
			return nil, sharedErr
		}
		if t.user == "" {
			return nil, fmt.Errorf("redfish %s: для %s не задан username", cfg.Code, t.host)
		}
		c.targets = append(c.targets, t)
	}
	return c, nil
}

func parseTarget(addr string) (target, error) {
	if !strings.Contains(addr, "://") {
		addr = "https://" + addr
	}
	u, err := url.Parse(addr)
	if err != nil || u.Host == "" {
		return target{}, fmt.Errorf("некорректный адрес")
	}
	return target{base: u.Scheme + "://" + u.Host, host: u.Host}, nil
}

func (c *Client) client(t target) *httpx.Client {
	h := httpx.New(t.base, c.timeout, c.insecure)
	h.User, h.Password = t.user, t.password
	h.Header.Set("OData-Version", "4.0")
	return h
}

// Test проверяет все цели: корень сервиса и учётные данные.
func (c *Client) Test(ctx context.Context) error {
	var bad []string
	for _, t := range c.targets {
		var root map[string]any
		if err := c.client(t).JSON(ctx, http.MethodGet, "/redfish/v1", nil, &root); err != nil {
			bad = append(bad, fmt.Sprintf("%s: %v", t.host, err))
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("redfish %s: недоступны %d из %d BMC: %s", c.code, len(bad), len(c.targets), strings.Join(bad, "; "))
	}
	return nil
}

// Extract опрашивает BMC параллельно. Ошибка отдельного BMC — Item{Err, Retain}; если не ответил
// ни один — ошибка снимка.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	items := make([]canon.Item, len(c.targets))
	sem := make(chan struct{}, c.concurrency)
	var wg sync.WaitGroup
	for i, t := range c.targets {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			rec, err := c.collect(ctx, t)
			items[i] = canon.Item{Seq: i + 1, Record: rec}
			if err != nil {
				items[i].Err, items[i].Retain = err, true
				items[i].Record = canon.Record{Class: "server", SourceName: t.host, ExternalID: "bmc:" + t.host}
			}
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	failed := 0
	for _, it := range items {
		if it.Err != nil {
			failed++
		}
	}
	if failed == len(items) {
		return nil, fmt.Errorf("redfish %s: не ответил ни один BMC из %d (первая ошибка: %v)", c.code, len(items), items[0].Err)
	}
	return items, nil
}

type node = map[string]any

type walker struct {
	ctx     context.Context
	http    *httpx.Client
	partial bool // какая-то категория компонентов не прочитана — список компонентов недостоверен
}

func (w *walker) get(path string) (node, error) {
	var n node
	if err := w.http.JSON(w.ctx, http.MethodGet, path, nil, &n); err != nil {
		return nil, err
	}
	return n, nil
}

// optional читает ресурс, допуская его отсутствие (404 — вендор не реализует): (nil, nil).
// Любая другая ошибка помечает компоненты недостоверными.
func (w *walker) optional(path string) node {
	if path == "" {
		return nil
	}
	n, err := w.get(path)
	if err != nil {
		if !strings.Contains(err.Error(), "HTTP 404") {
			w.partial = true
		}
		return nil
	}
	return n
}

func odata(n node, key string) string {
	m, _ := n[key].(map[string]any)
	s, _ := m["@odata.id"].(string)
	return s
}

// members возвращает ресурсы-члены коллекции; ошибка члена помечает компоненты недостоверными.
func (w *walker) members(coll string) []node {
	c := w.optional(coll)
	if c == nil {
		return nil
	}
	var out []node
	list, _ := c["Members"].([]any)
	for _, m := range list {
		mm, _ := m.(map[string]any)
		id, _ := mm["@odata.id"].(string)
		if id == "" {
			continue
		}
		if n := w.optional(id); n != nil {
			out = append(out, n)
		}
	}
	return out
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

func absent(n node) bool {
	st, _ := n["Status"].(map[string]any)
	s, _ := st["State"].(string)
	return s == "Absent"
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// collect читает один BMC и строит запись сервера.
func (c *Client) collect(ctx context.Context, t target) (canon.Record, error) {
	w := &walker{ctx: ctx, http: c.client(t)}

	root, err := w.get("/redfish/v1")
	if err != nil {
		return canon.Record{}, err
	}
	// Systems обязательны: без них записи нет — это ошибка BMC, а не «пустой сервер».
	sysColl, err := w.get(odata(root, "Systems"))
	if err != nil {
		return canon.Record{}, err
	}
	sysLinks, _ := sysColl["Members"].([]any)
	if len(sysLinks) == 0 {
		return canon.Record{}, fmt.Errorf("BMC не сообщил ни одной системы")
	}
	first, _ := sysLinks[0].(map[string]any)
	sysPath, _ := first["@odata.id"].(string)
	sys, err := w.get(sysPath)
	if err != nil {
		return canon.Record{}, err
	}

	serial := str(sys, "SerialNumber")
	manufacturer := str(sys, "Manufacturer")
	rec := canon.Record{
		Class: "server", ExternalID: "bmc:" + t.host, SerialNumber: serial, Identifiers: map[string]string{},
		Attrs: map[string]any{}, Description: "",
	}
	rec.SourceName = str(sys, "HostName")
	if rec.SourceName == "" { // сервер без ОС: имя временное по серийному номеру (noncompliant), КЕ не пропускается
		rec.SourceName = strings.NewReplacer(" ", "-").Replace(strings.Trim(manufacturer+"-"+serial, "-"))
		if rec.SourceName == "" {
			rec.SourceName = t.host
		}
	}
	if serial != "" {
		rec.Identifiers["serial_number"] = serial
	}
	if u := str(sys, "UUID"); uuidRe.MatchString(u) {
		u = strings.ToLower(u)
		rec.Attrs["bios_uuid"] = u
		rec.Identifiers["bios_uuid"] = u
	}
	if v := str(sys, "BiosVersion"); v != "" {
		rec.Attrs["firmware"] = v
	}

	var comps []canon.Component
	var ifaces []canon.Interface

	// --- процессоры
	cpuCount, cores := 0, 0
	for _, p := range w.members(odata(sys, "Processors")) {
		if absent(p) {
			continue
		}
		if pt := str(p, "ProcessorType"); pt != "" && pt != "CPU" {
			continue // GPU/FPGA как процессоры не считаем
		}
		cpuCount++
		cores += int(num(p, "TotalCores"))
		slot := firstNonEmpty(str(p, "Socket"), str(p, "Id"))
		comps = append(comps, canon.Component{Key: "cpu:" + slot, Type: "cpu", Slot: slot, Manufacturer: str(p, "Manufacturer"),
			PartNumber: str(p, "Model"), Speed: mhz(num(p, "MaxSpeedMHz"))})
	}
	if cpuCount == 0 {
		cpuCount = int(num(nestedNode(sys, "ProcessorSummary"), "Count"))
	}

	// --- память
	dimmGiB := 0.0
	for _, d := range w.members(odata(sys, "Memory")) {
		if absent(d) {
			continue
		}
		loc := firstNonEmpty(str(d, "DeviceLocator"), str(d, "Id"))
		serialD := str(d, "SerialNumber")
		gib := num(d, "CapacityMiB") / 1024
		dimmGiB += gib
		comps = append(comps, canon.Component{Key: "dimm:" + firstNonEmpty(serialD, loc), Type: "dimm", Slot: loc,
			Manufacturer: str(d, "Manufacturer"), PartNumber: str(d, "PartNumber"), SerialNumber: serialD,
			CapacityValue: fptr(gib), CapacityUnit: "GiB", Speed: mhz(num(d, "OperatingSpeedMhz"))})
	}

	// --- хранилище: контроллеры и диски (диск — потомок контроллера)
	for _, st := range w.members(odata(sys, "Storage")) {
		ctrlKey := ""
		if ctrls, _ := st["StorageControllers"].([]any); len(ctrls) > 0 {
			cc, _ := ctrls[0].(map[string]any)
			ctrlKey = "raid:" + str(st, "Id")
			comps = append(comps, canon.Component{Key: ctrlKey, Type: "raid_ctrl", Slot: str(st, "Id"), Manufacturer: str(cc, "Manufacturer"),
				PartNumber: str(cc, "Model"), SerialNumber: str(cc, "SerialNumber"), Firmware: str(cc, "FirmwareVersion")})
		}
		drives, _ := st["Drives"].([]any)
		for _, dl := range drives {
			dm, _ := dl.(map[string]any)
			id, _ := dm["@odata.id"].(string)
			d := w.optional(id)
			if d == nil || absent(d) {
				continue
			}
			typ := "disk"
			switch {
			case strings.EqualFold(str(d, "Protocol"), "NVMe"):
				typ = "nvme"
			case strings.EqualFold(str(d, "MediaType"), "SSD"):
				typ = "ssd"
			}
			serialD := str(d, "SerialNumber")
			comps = append(comps, canon.Component{Key: "drive:" + firstNonEmpty(serialD, str(st, "Id")+"/"+str(d, "Id")), ParentKey: ctrlKey,
				Type: typ, Slot: str(d, "Id"), Manufacturer: str(d, "Manufacturer"), PartNumber: str(d, "Model"), SerialNumber: serialD,
				Firmware: str(d, "Revision"), CapacityValue: fptr(math.Round(num(d, "CapacityBytes")/1e9*10) / 10), CapacityUnit: "GB"})
		}
	}

	// --- сетевые интерфейсы сервера
	for _, e := range w.members(odata(sys, "EthernetInterfaces")) {
		st, _ := e["Status"].(map[string]any)
		ifaces = append(ifaces, canon.Interface{Name: firstNonEmpty(str(e, "Id"), str(e, "Name")), MAC: firstNonEmpty(str(e, "MACAddress"), str(e, "PermanentMACAddress")),
			SpeedMbps: int64(num(e, "SpeedMbps")), Enabled: fmt.Sprint(st["State"]) != "Disabled"})
	}

	// --- шасси: форм-фактор, БП, вентиляторы
	links, _ := sys["Links"].(map[string]any)
	if chList, _ := links["Chassis"].([]any); len(chList) > 0 {
		cm, _ := chList[0].(map[string]any)
		chPath, _ := cm["@odata.id"].(string)
		if ch := w.optional(chPath); ch != nil {
			switch str(ch, "ChassisType") {
			case "Blade":
				rec.Attrs["form_factor"] = "blade"
			case "RackMount":
				rec.Attrs["form_factor"] = "rack"
			case "Tower":
				rec.Attrs["form_factor"] = "tower"
			}
			if pw := w.optional(odata(ch, "Power")); pw != nil {
				psus, _ := pw["PowerSupplies"].([]any)
				for _, p := range psus {
					pn, _ := p.(map[string]any)
					if absent(pn) {
						continue
					}
					name := firstNonEmpty(str(pn, "Name"), str(pn, "MemberId"))
					serialP := str(pn, "SerialNumber")
					comps = append(comps, canon.Component{Key: "psu:" + firstNonEmpty(serialP, name), Type: "psu", Slot: name,
						Manufacturer: str(pn, "Manufacturer"), PartNumber: str(pn, "Model"), SerialNumber: serialP, Firmware: str(pn, "FirmwareVersion"),
						CapacityValue: fptr(num(pn, "PowerCapacityWatts")), CapacityUnit: "W"})
				}
			}
			if th := w.optional(odata(ch, "Thermal")); th != nil {
				fans, _ := th["Fans"].([]any)
				for _, f := range fans {
					fn, _ := f.(map[string]any)
					if absent(fn) {
						continue
					}
					name := firstNonEmpty(str(fn, "Name"), str(fn, "MemberId"))
					comps = append(comps, canon.Component{Key: "fan:" + name, Type: "fan", Slot: name})
				}
			}
		}
	}

	// --- BMC (менеджер): компонент, MAC, IP, имя
	bmcIP := ""
	if net.ParseIP(strings.Split(t.host, ":")[0]) != nil {
		bmcIP = strings.Split(t.host, ":")[0]
	}
	if mgrList, _ := links["ManagedBy"].([]any); len(mgrList) > 0 {
		mm, _ := mgrList[0].(map[string]any)
		mPath, _ := mm["@odata.id"].(string)
		if m := w.optional(mPath); m != nil {
			comps = append(comps, canon.Component{Key: "bmc:" + str(m, "Id"), Type: "bmc", Slot: str(m, "Id"),
				Manufacturer: str(m, "Manufacturer"), PartNumber: str(m, "Model"), Firmware: str(m, "FirmwareVersion")})
			for _, e := range w.members(odata(m, "EthernetInterfaces")) {
				mac := firstNonEmpty(str(e, "MACAddress"), str(e, "PermanentMACAddress"))
				if mac == "" {
					continue
				}
				ifaces = append(ifaces, canon.Interface{Name: "bmc", MAC: mac, Enabled: true, Role: "bmc"})
				if hw, err := net.ParseMAC(mac); err == nil {
					rec.Identifiers["bmc_mac"] = hw.String()
				}
				if bmcIP == "" {
					if a, _ := e["IPv4Addresses"].([]any); len(a) > 0 {
						an, _ := a[0].(map[string]any)
						bmcIP = str(an, "Address")
					}
				}
				break // основной интерфейс
			}
			if np := w.optional(odata(m, "NetworkProtocol")); np != nil {
				if h := str(np, "HostName"); h != "" {
					rec.Attrs["bmc_name"] = h
				}
			}
		}
	}
	if bmcIP != "" {
		rec.Attrs["bmc_ip"] = bmcIP
	}

	// --- сводные атрибуты ci_server
	if cpuCount > 0 {
		rec.Attrs["cpu_count"] = cpuCount
	}
	if cores > 0 {
		rec.Attrs["core_count"] = cores
	}
	ramGiB := num(nestedNode(sys, "MemorySummary"), "TotalSystemMemoryGiB")
	if ramGiB == 0 {
		ramGiB = dimmGiB
	}
	if ramGiB > 0 {
		rec.Attrs["ram_gb"] = int(math.Round(ramGiB))
	}

	rec.Vendor = map[string]any{"manufacturer": manufacturer, "model": str(sys, "Model"), "sku": str(sys, "SKU"),
		"part_number": str(sys, "PartNumber"), "redfish_version": str(root, "RedfishVersion")}

	// Компоненты и интерфейсы сообщаются только когда прочитаны целиком; иначе nil — конвейер
	// оставляет прежние (частичный список пометил бы живые компоненты снятыми).
	if !w.partial {
		sort.Slice(comps, func(i, j int) bool { return comps[i].Key < comps[j].Key })
		if comps == nil {
			comps = []canon.Component{}
		}
		rec.Components = comps
		if ifaces == nil {
			ifaces = []canon.Interface{}
		}
		rec.Interfaces = ifaces
	}
	return rec, nil
}

func nestedNode(n node, k string) node {
	m, _ := n[k].(map[string]any)
	return m
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func fptr(f float64) *float64 {
	if f <= 0 {
		return nil
	}
	return &f
}

func mhz(f float64) string {
	if f <= 0 {
		return ""
	}
	return strconv.FormatFloat(f, 'f', -1, 64) + " MHz"
}
