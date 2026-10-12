// Package zvirt — коннектор zVirt / РЕД Виртуализация (oVirt REST API, plan.md E5.2a): собственный
// клиент на net/http. Забирает менеджер виртуализации, кластеры, хосты, ВМ и домены хранения
// (только type=data) и связи между ними; идентификаторы и связи описаны в canon.Record.
//
// Поля ответа API взяты из документации oVirt REST v4 и записанных ответов в
// test/fixtures/zvirt, а НЕ подтверждены на живом zVirt (harness/09-source-zvirt-ovirt.md).
package zvirt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
)

// Kind — значение imp.connector.kind.
const Kind = "zvirt"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного менеджера zVirt.
type Client struct {
	code        string
	baseURL     string
	managerName string
	user        string
	password    string
	http        *http.Client
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://engine.example.local),
// username (admin@internal), password_env (имя переменной окружения с паролем), manager_name
// (имя КЕ менеджера, по умолчанию код коннектора), timeout_sec (на один HTTP-запрос, по
// умолчанию 60), insecure_skip_verify (только для стендов с самоподписанным сертификатом).
func New(cfg connector.Config) (connector.Connector, error) {
	base := strings.TrimRight(cfg.String("base_url"), "/")
	if base == "" {
		return nil, fmt.Errorf("zvirt %s: не задан base_url", cfg.Code)
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("zvirt %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	name := cfg.String("manager_name")
	if name == "" {
		name = cfg.Code
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.Bool("insecure_skip_verify")}} //nolint:gosec // явная настройка стенда
	return &Client{
		code: cfg.Code, baseURL: base, managerName: name, user: user, password: pass,
		http: &http.Client{Transport: tr, Timeout: time.Duration(cfg.Int("timeout_sec", 60)) * time.Second},
	}, nil
}

// get выполняет GET к /ovirt-engine/api<path> и разбирает JSON в out.
func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ovirt-engine/api"+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.user, c.password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Version", "4")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
	if err != nil {
		return fmt.Errorf("GET %s: чтение ответа: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d: %s", path, resp.StatusCode, truncate(string(body), 200))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("GET %s: разбор JSON: %w", path, err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

// Test проверяет учётные данные запросом корня API.
func (c *Client) Test(ctx context.Context) error {
	var root apiRoot
	if err := c.get(ctx, "", &root); err != nil {
		return err
	}
	if root.ProductInfo.Name == "" {
		return fmt.Errorf("ответ не похож на oVirt API: нет product_info")
	}
	return nil
}

// Extract строит полный снимок менеджера. Любая ошибка запроса — ошибка всего снимка: частичный
// снимок привёл бы к ложному «не найдено» у КЕ, чьи объекты просто не успели прочитаться.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	var root apiRoot
	var clusters struct {
		Cluster []cluster `json:"cluster"`
	}
	var hosts struct {
		Host []host `json:"host"`
	}
	var vms struct {
		VM []vm `json:"vm"`
	}
	var domains struct {
		Domain []storageDomain `json:"storage_domain"`
	}
	for _, q := range []struct {
		path string
		out  any
	}{{"", &root}, {"/clusters", &clusters}, {"/hosts", &hosts}, {"/vms", &vms}, {"/storagedomains", &domains}} {
		if err := c.get(ctx, q.path, q.out); err != nil {
			return nil, err
		}
	}

	var recs []canon.Record

	managerID := "manager:" + c.code
	recs = append(recs, canon.Record{
		Class: "virt_manager", SourceName: c.managerName, ExternalID: managerID,
		Attrs: map[string]any{"product": "zVirt", "version": root.ProductInfo.Version.FullVersion, "api_url": c.baseURL},
	})

	known := map[string]bool{managerID: true}
	for _, cl := range clusters.Cluster {
		id := "cluster:" + cl.ID
		known[id] = true
		recs = append(recs, canon.Record{
			Class: "cluster", SourceName: cl.Name, ExternalID: id, Description: cl.Description,
			Attrs: map[string]any{"cluster_kind": "virt", "ha": cl.HAReservation, "platform": "zVirt"},
		})
		// связь «менеджер содержит кластер» исходит от записи менеджера (recs[0])
		recs[0].Relations = append(recs[0].Relations, canon.Relation{Type: "contains", TargetExternalID: id})
	}

	for _, h := range hosts.Host {
		id := "host:" + h.ID
		known[id] = true
		serial := strings.TrimSpace(h.Hardware.SerialNumber)
		rec := canon.Record{
			Class: "hypervisor_host", SourceName: h.Name, ExternalID: id, SerialNumber: serial,
			Identifiers: map[string]string{},
			Attrs: map[string]any{"hv_type": "kvm", "platform": "zVirt", "hv_version": h.Version.FullVersion,
				"maintenance_mode": h.Status == "maintenance", "bios_uuid": biosUUID(h.Hardware.UUID)},
		}
		if serial != "" {
			rec.Identifiers["serial_number"] = serial
		}
		if h.Cluster.ID != "" {
			rec.Relations = append(rec.Relations, canon.Relation{Type: "member_of", TargetExternalID: "cluster:" + h.Cluster.ID})
		}
		recs = append(recs, rec)
	}

	for _, v := range vms.VM {
		id := "vm:" + v.ID
		known[id] = true
		uuid := strings.ToLower(v.ID)
		rec := canon.Record{
			Class: "vm", SourceName: v.Name, ExternalID: id, Description: v.Description,
			Identifiers: map[string]string{"instance_uuid": uuid},
			Attrs: map[string]any{"instance_uuid": uuid, "template": false,
				"vcpu":   v.vcpus(),
				"ram_gb": gib(v.Memory.Int64())},
		}
		if d := v.Guest.Distribution; d != "" {
			rec.Attrs["guest_os"] = strings.TrimSpace(d + " " + v.Guest.Version.FullVersion)
		}
		if v.Host.ID != "" {
			rec.Relations = append(rec.Relations, canon.Relation{Type: "runs_on", TargetExternalID: "host:" + v.Host.ID})
		}
		if v.Cluster.ID != "" {
			rec.Relations = append(rec.Relations, canon.Relation{Type: "member_of", TargetExternalID: "cluster:" + v.Cluster.ID})
		}
		recs = append(recs, rec)
	}

	for _, d := range domains.Domain {
		if d.Type != "data" {
			continue // iso/export — не хранилище ВМ
		}
		id := "sd:" + d.ID
		known[id] = true
		recs = append(recs, canon.Record{
			Class: "datastore", SourceName: d.Name, ExternalID: id,
			Attrs:  map[string]any{"ds_type": dsType(d.Storage.Type), "capacity_gb": capacityGB(d.Available.Int64() + d.Used.Int64())},
			Vendor: map[string]any{"storage_type": d.Storage.Type},
		})
	}

	items := make([]canon.Item, len(recs))
	for i := range recs {
		// связь на объект, которого нет в снимке (например host отфильтрован), конвейер отметит
		// конфликтом apply_error; здесь ничего не отбрасываем молча
		items[i] = canon.Item{Seq: i + 1, Record: recs[i]}
	}
	return items, nil
}

// dsType сводит тип домена хранения oVirt к допустимым значениям ci_datastore.ds_type
// (vmfs/nfs/vsan): у oVirt осмысленно только nfs; fcp/iscsi/glusterfs/localfs остаются NULL, а
// исходное значение — в вендорских сведениях (иначе CHECK БД отклонил бы всю запись).
func dsType(t string) any {
	if t == "nfs" {
		return "nfs"
	}
	return nil
}

const giB = 1 << 30

// gib — байты в целые ГиБ (округление к ближайшему, но ненулевая память не сводится к 0).
func gib(b int64) int {
	if b <= 0 {
		return 0
	}
	return max(1, int(math.Round(float64(b)/giB)))
}

func capacityGB(b int64) float64 { return math.Round(float64(b)/giB*100) / 100 }

// flex — целое, которое oVirt отдаёт то числом, то строкой ("4294967296").
type flex int64

func (f *flex) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		fl, ferr := strconv.ParseFloat(s, 64)
		if ferr != nil {
			return fmt.Errorf("не число: %q", s)
		}
		n = int64(fl)
	}
	*f = flex(n)
	return nil
}

func (f flex) Int64() int64 { return int64(f) }
func (f flex) Int() int     { return int(f) }

type ref struct {
	ID string `json:"id"`
}

type apiRoot struct {
	ProductInfo struct {
		Name    string `json:"name"`
		Version struct {
			FullVersion string `json:"full_version"`
		} `json:"version"`
	} `json:"product_info"`
}

type cluster struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	HAReservation bool   `json:"ha_reservation"`
}

type host struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Cluster ref    `json:"cluster"`
	Version struct {
		FullVersion string `json:"full_version"`
	} `json:"version"`
	Hardware struct {
		SerialNumber string `json:"serial_number"`
		UUID         string `json:"uuid"`
	} `json:"hardware_information"`
}

type vm struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Cluster     ref    `json:"cluster"`
	Host        ref    `json:"host"`
	Memory      flex   `json:"memory"`
	CPU         struct {
		Topology struct {
			Cores   flex `json:"cores"`
			Sockets flex `json:"sockets"`
			Threads flex `json:"threads"`
		} `json:"topology"`
	} `json:"cpu"`
	Guest struct {
		Distribution string `json:"distribution"`
		Version      struct {
			FullVersion string `json:"full_version"`
		} `json:"version"`
	} `json:"guest_operating_system"`
}

// vcpus — число vCPU: ядра x сокеты x потоки; отсутствующие сокеты/потоки считаются единицей.
func (v vm) vcpus() int {
	t := v.CPU.Topology
	return t.Cores.Int() * max(1, t.Sockets.Int()) * max(1, t.Threads.Int())
}

type storageDomain struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Available flex   `json:"available"`
	Used      flex   `json:"used"`
	Storage   struct {
		Type string `json:"type"`
	} `json:"storage"`
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// biosUUID — SMBIOS UUID хоста (мост к серверу по UUID, E5.5); некорректное значение — nil, иначе
// CAST в колонку uuid отклонил бы запись целиком.
func biosUUID(s string) any {
	s = strings.TrimSpace(s)
	if !uuidRe.MatchString(s) {
		return nil
	}
	return strings.ToLower(s)
}
