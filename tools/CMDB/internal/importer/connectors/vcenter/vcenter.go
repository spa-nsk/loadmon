// Package vcenter — коннектор VMware vCenter / VCF (govmomi, SOAP; plan.md E5.1). Строит снимок
// через container view + property collector: менеджер, кластеры, хосты ESXi, ВМ, датасторы и связи
// contains / member_of / runs_on / uses_storage. Серийный номер хоста (ServiceTag/SerialNumberTag)
// — мост к физическому серверу из OneView/Redfish (E5.5, выводится конвейером). Одиночный ESXi без
// vCenter обслуживается тем же кодом — только другой base_url (E8.3). vSAN — как признак кластера
// (vsan_enabled, govmomi vsan.VsanClusterGetConfig) и обычный vsan-датастор со свободной ёмкостью
// (free_gb); отдельный внутренний SOAP-сервис vsanHealth с детальными health-проверками
// не используется — не документирован для сторонних клиентов (E8.3).
//
// Проверено на симуляторе govmomi (vcsim, vSphere 8), НЕ на живом vCenter/VCF 9
// (harness/12-source-vcenter.md). Дубликаты UUID у ВМ (клоны) — норма для VMware: такие UUID не
// используются как идентификаторы (harness/02).
package vcenter

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/vsan"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/storage"
)

// Kind — значение imp.connector.kind.
const Kind = "vcenter"

func init() { connector.Register(Kind, New) }

// Client — коннектор одного vCenter.
type Client struct {
	code        string
	url         *url.URL
	insecure    bool
	managerName string
}

// New создаёт коннектор. Настройки imp.connector.config: base_url (https://vcenter.example.local;
// путь /sdk добавляется), username, password_env (имя переменной окружения), manager_name (имя КЕ
// менеджера, по умолчанию код коннектора), insecure_skip_verify (только для стендов).
func New(cfg connector.Config) (connector.Connector, error) {
	base := cfg.String("base_url")
	if base == "" {
		return nil, fmt.Errorf("vcenter %s: не задан base_url", cfg.Code)
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("vcenter %s: некорректный base_url %q", cfg.Code, base)
	}
	if !strings.HasSuffix(u.Path, "/sdk") {
		u.Path = strings.TrimRight(u.Path, "/") + "/sdk"
	}
	user := cfg.String("username")
	if user == "" {
		return nil, fmt.Errorf("vcenter %s: не задан username", cfg.Code)
	}
	pass, err := cfg.Secret("password_env")
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(user, pass)
	name := cfg.String("manager_name")
	if name == "" {
		name = cfg.Code
	}
	return &Client{code: cfg.Code, url: u, insecure: cfg.Bool("insecure_skip_verify"), managerName: name}, nil
}

func (c *Client) login(ctx context.Context) (*govmomi.Client, func(), error) {
	gc, err := govmomi.NewClient(ctx, c.url, c.insecure)
	if err != nil {
		return nil, nil, fmt.Errorf("vcenter %s: вход: %w", c.code, err)
	}
	return gc, func() {
		// выход не должен зависеть от отменённого контекста задания
		_ = gc.Logout(context.WithoutCancel(ctx))
	}, nil
}

// Test открывает и закрывает сессию.
func (c *Client) Test(ctx context.Context) error {
	_, logout, err := c.login(ctx)
	if err != nil {
		return err
	}
	logout()
	return nil
}

// Extract строит полный снимок; ошибка любого запроса — ошибка всего снимка.
func (c *Client) Extract(ctx context.Context) ([]canon.Item, error) {
	gc, logout, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	defer logout()

	snap, err := fetch(ctx, gc)
	if err != nil {
		return nil, fmt.Errorf("vcenter %s: %w", c.code, err)
	}
	recs := buildRecords(snap, c.code, c.managerName)
	items := make([]canon.Item, len(recs))
	for i := range recs {
		items[i] = canon.Item{Seq: i + 1, Record: recs[i]}
	}
	return items, nil
}

// --- снимок: плоские структуры без govmomi-типов (buildRecords тестируется без vCenter) ---

type clusterInfo struct {
	ID, Name    string
	HA, DRS     bool
	VsanEnabled bool
}

type hbaInfo struct{ Device, WWPN string }

type hostInfo struct {
	HBAs                        []hbaInfo // FC-адаптеры: WWPN — мост к презентации томов СХД (E5.5)
	ID, Name, ClusterID         string    // ClusterID пуст у одиночного хоста
	Serial, UUID, Vendor, Model string
	Version, Build              string
	InMaintenance               bool
}

type vmInfo struct {
	ID, Name, UUID, InstanceUUID string
	Template                     bool
	CPU, MemoryMB                int
	GuestOS, ToolsStatus, HostID string
	DatastoreIDs                 []string
}

type dsInfo struct {
	ID, Name, Type           string
	CapacityBytes, FreeBytes int64
	NAA                      []string // устройства хранилища VMFS (extent): NAA -> WWN тома СХД (E5.5)
}

type snapshot struct {
	Product, Version, Build string
	Clusters                []clusterInfo
	Hosts                   []hostInfo
	VMs                     []vmInfo
	Datastores              []dsInfo
}

func fetch(ctx context.Context, gc *govmomi.Client) (snapshot, error) {
	var snap snapshot
	about := gc.ServiceContent.About
	snap.Product, snap.Version, snap.Build = about.Name, about.Version, about.Build

	vm := view.NewManager(gc.Client)
	root := gc.ServiceContent.RootFolder
	// Живой набор строится отдельным запросом, а не из кэша (harness/02).
	get := func(kind string, props []string, dst any) error {
		v, err := vm.CreateContainerView(ctx, root, []string{kind}, true)
		if err != nil {
			return fmt.Errorf("представление %s: %w", kind, err)
		}
		defer v.Destroy(context.WithoutCancel(ctx))
		if err := v.Retrieve(ctx, []string{kind}, props, dst); err != nil {
			return fmt.Errorf("чтение %s: %w", kind, err)
		}
		return nil
	}

	var clusters []mo.ClusterComputeResource
	if err := get("ClusterComputeResource", []string{"name", "configuration.dasConfig.enabled", "configuration.drsConfig.enabled"}, &clusters); err != nil {
		return snap, err
	}
	clusterIDs := map[string]bool{}
	for _, cl := range clusters {
		info := clusterInfo{ID: cl.Self.Value, Name: cl.Name}
		if cl.Configuration.DasConfig.Enabled != nil {
			info.HA = *cl.Configuration.DasConfig.Enabled
		}
		if cl.Configuration.DrsConfig.Enabled != nil {
			info.DRS = *cl.Configuration.DrsConfig.Enabled
		}
		clusterIDs[info.ID] = true
		snap.Clusters = append(snap.Clusters, info)
	}

	// vsan_enabled — единственный признак vSAN, который надёжно берётся стандартным способом
	// (VsanClusterGetConfig — часть публикуемого govmomi-пакета vsan, проверяется симулятором);
	// внутренний SOAP-сервис vsanHealth с детальными проверками здоровья кластера не документирован
	// для сторонних клиентов и не имеет опубликованного идентификатора управляемого объекта — не
	// используется (E8.3, по решению пользователя: надёжный минимум вместо угадывания схемы).
	// Сбой самого vSAN-сервиса (не включён/старая версия vCenter) не должен обрушивать весь снимок.
	if vc, err := vsan.NewClient(ctx, gc.Client); err == nil {
		for i := range snap.Clusters {
			ref := types.ManagedObjectReference{Type: "ClusterComputeResource", Value: snap.Clusters[i].ID}
			if cfg, err := vc.VsanClusterGetConfig(ctx, ref); err == nil && cfg.Enabled != nil {
				snap.Clusters[i].VsanEnabled = *cfg.Enabled
			}
		}
	}

	var hosts []mo.HostSystem
	if err := get("HostSystem", []string{"name", "parent", "hardware.systemInfo", "config.product", "config.storageDevice.hostBusAdapter", "runtime.inMaintenanceMode"}, &hosts); err != nil {
		return snap, err
	}
	for _, h := range hosts {
		info := hostInfo{ID: h.Self.Value, Name: h.Name, Serial: hostSerial(h.Hardware),
			InMaintenance: h.Runtime.InMaintenanceMode}
		if h.Parent != nil && clusterIDs[h.Parent.Value] {
			info.ClusterID = h.Parent.Value
		}
		if h.Hardware != nil {
			info.UUID, info.Vendor, info.Model = strings.ToLower(h.Hardware.SystemInfo.Uuid), h.Hardware.SystemInfo.Vendor, h.Hardware.SystemInfo.Model
		}
		if h.Config != nil {
			info.Version, info.Build = h.Config.Product.Version, h.Config.Product.Build
			if h.Config.StorageDevice != nil {
				for _, hba := range h.Config.StorageDevice.HostBusAdapter {
					if fc, ok := hba.(*types.HostFibreChannelHba); ok {
						info.HBAs = append(info.HBAs, hbaInfo{Device: fc.Device, WWPN: fmt.Sprintf("%016x", uint64(fc.PortWorldWideName))})
					}
				}
			}
		}
		snap.Hosts = append(snap.Hosts, info)
	}

	var vms []mo.VirtualMachine
	if err := get("VirtualMachine", []string{"name", "config.uuid", "config.instanceUuid", "config.template", "config.hardware.numCPU",
		"config.hardware.memoryMB", "config.guestFullName", "guest.toolsStatus", "runtime.host", "datastore"}, &vms); err != nil {
		return snap, err
	}
	for _, x := range vms {
		if x.Config == nil {
			continue // ВМ в процессе создания/удаления — конфигурации ещё/уже нет
		}
		info := vmInfo{ID: x.Self.Value, Name: x.Name, UUID: strings.ToLower(x.Config.Uuid), InstanceUUID: strings.ToLower(x.Config.InstanceUuid),
			Template: x.Config.Template, CPU: int(x.Config.Hardware.NumCPU),
			MemoryMB: int(x.Config.Hardware.MemoryMB), GuestOS: x.Config.GuestFullName}
		if x.Guest != nil {
			info.ToolsStatus = string(x.Guest.ToolsStatus)
		}
		if x.Runtime.Host != nil {
			info.HostID = x.Runtime.Host.Value
		}
		for _, d := range x.Datastore {
			info.DatastoreIDs = append(info.DatastoreIDs, d.Value)
		}
		snap.VMs = append(snap.VMs, info)
	}

	var dss []mo.Datastore
	if err := get("Datastore", []string{"name", "summary.type", "summary.capacity", "summary.freeSpace", "info"}, &dss); err != nil {
		return snap, err
	}
	for _, d := range dss {
		info := dsInfo{ID: d.Self.Value, Name: d.Name, Type: d.Summary.Type, CapacityBytes: d.Summary.Capacity, FreeBytes: d.Summary.FreeSpace}
		if v, ok := d.Info.(*types.VmfsDatastoreInfo); ok && v.Vmfs != nil {
			for _, e := range v.Vmfs.Extent {
				if naa := storage.NormalizeWWN(e.DiskName); naa != "" {
					info.NAA = append(info.NAA, naa)
				}
			}
		}
		snap.Datastores = append(snap.Datastores, info)
	}
	return snap, nil
}

// hostSerial — серийный номер сервера из идентификаторов оборудования (порядок предпочтения:
// SerialNumberTag, ServiceTag; EnclosureSerialNumberTag — серийный корзины, не сервера).
func hostSerial(hw *types.HostHardwareInfo) string {
	if hw == nil {
		return ""
	}
	found := map[string]string{}
	for _, o := range hw.SystemInfo.OtherIdentifyingInfo {
		if d := o.IdentifierType.GetElementDescription(); d != nil {
			found[d.Key] = strings.TrimSpace(o.IdentifierValue)
		}
	}
	for _, key := range []string{"SerialNumberTag", "ServiceTag"} {
		if v := found[key]; v != "" {
			return v
		}
	}
	return ""
}

// buildRecords превращает снимок в канонические записи (чистая функция).
func buildRecords(snap snapshot, code, managerName string) []canon.Record {
	var recs []canon.Record
	managerID := "manager:" + code
	// product — как отвечает сам endpoint (ServiceContent.About.Name), не жёстко "vCenter": та же
	// связка govmomi/vSphere API отвечает и на прямое подключение к отдельному хосту ESXi без
	// vCenter (plan.md E8.3 — «vmware» без centralized-менеджера), тогда About.Name = "VMware
	// ESXi", а не "VMware vCenter Server" — see TestExtract_AgainstStandaloneESXi.
	manager := canon.Record{
		Class: "virt_manager", SourceName: managerName, ExternalID: managerID,
		Attrs: map[string]any{"product": snap.Product, "version": strings.TrimSpace(snap.Version + " build " + snap.Build)},
	}

	clusterOf := map[string]string{} // id хоста -> id кластера
	for _, h := range snap.Hosts {
		if h.ClusterID != "" {
			clusterOf[h.ID] = h.ClusterID
		}
	}
	hostKnown := map[string]bool{}
	for _, h := range snap.Hosts {
		hostKnown[h.ID] = true
	}
	dsKnown := map[string]bool{}
	for _, d := range snap.Datastores {
		dsKnown[d.ID] = true
	}

	for _, cl := range snap.Clusters {
		id := "cluster:" + cl.ID
		manager.Relations = append(manager.Relations, canon.Relation{Type: "contains", TargetExternalID: id})
		recs = append(recs, canon.Record{
			Class: "cluster", SourceName: cl.Name, ExternalID: id,
			Attrs: map[string]any{"cluster_kind": "virt", "ha": cl.HA, "drs": cl.DRS, "platform": "vSphere", "vsan_enabled": cl.VsanEnabled},
		})
	}

	for _, h := range snap.Hosts {
		id := "host:" + h.ID
		rec := canon.Record{
			Class: "hypervisor_host", SourceName: h.Name, ExternalID: id, SerialNumber: h.Serial, Identifiers: map[string]string{},
			Attrs: map[string]any{"hv_type": "esxi", "platform": "vSphere", "hv_version": strings.TrimSpace(h.Version + " build " + h.Build), "maintenance_mode": h.InMaintenance,
				"bios_uuid": nilIfEmpty(h.UUID)},
			Vendor: map[string]any{"vendor": h.Vendor, "model": h.Model, "smbios_uuid": h.UUID},
		}
		if h.Serial != "" {
			rec.Identifiers["serial_number"] = h.Serial
		}
		rec.Interfaces = []canon.Interface{} // список FC-портов достоверен (возможно, пуст)
		for _, hba := range h.HBAs {
			if w := storage.NormalizeWWPN(hba.WWPN); w != "" {
				rec.Interfaces = append(rec.Interfaces, canon.Interface{Name: hba.Device, IfType: "fc", WWPN: w, Enabled: true})
			}
		}
		if h.ClusterID != "" {
			rec.Relations = append(rec.Relations, canon.Relation{Type: "member_of", TargetExternalID: "cluster:" + h.ClusterID})
		} else {
			manager.Relations = append(manager.Relations, canon.Relation{Type: "contains", TargetExternalID: id})
		}
		recs = append(recs, rec)
	}

	// UUID ВМ в VMware не уникальны (клоны, шаблоны): дубликат внутри снимка не может служить
	// идентификатором — иначе две ВМ схлопнулись бы в одну КЕ (harness/02).
	instCount, biosCount := map[string]int{}, map[string]int{}
	for _, v := range snap.VMs {
		if v.InstanceUUID != "" {
			instCount[v.InstanceUUID]++
		}
		if v.UUID != "" {
			biosCount[v.UUID]++
		}
	}
	for _, v := range snap.VMs {
		id := "vm:" + v.ID
		rec := canon.Record{
			Class: "vm", SourceName: v.Name, ExternalID: id, Identifiers: map[string]string{},
			Attrs: map[string]any{"moref": v.ID, "vcpu": nz(v.CPU), "ram_gb": nz(gib(v.MemoryMB)), "guest_os": nilIfEmpty(v.GuestOS),
				"tools_status": nilIfEmpty(v.ToolsStatus), "template": v.Template,
				"instance_uuid": nilIfEmpty(v.InstanceUUID), "bios_uuid": nilIfEmpty(v.UUID)},
		}
		dup := false
		if v.InstanceUUID != "" {
			if instCount[v.InstanceUUID] > 1 {
				dup = true
			} else {
				rec.Identifiers["instance_uuid"] = v.InstanceUUID
			}
		}
		if v.UUID != "" {
			if biosCount[v.UUID] > 1 {
				dup = true
			} else {
				rec.Identifiers["bios_uuid"] = v.UUID
			}
		}
		if dup {
			rec.Vendor = map[string]any{"duplicate_uuid": true}
		}
		if hostKnown[v.HostID] {
			rec.Relations = append(rec.Relations, canon.Relation{Type: "runs_on", TargetExternalID: "host:" + v.HostID})
			if cl := clusterOf[v.HostID]; cl != "" {
				rec.Relations = append(rec.Relations, canon.Relation{Type: "member_of", TargetExternalID: "cluster:" + cl})
			}
		}
		ds := append([]string(nil), v.DatastoreIDs...)
		sort.Strings(ds)
		for _, d := range ds {
			if dsKnown[d] {
				rec.Relations = append(rec.Relations, canon.Relation{Type: "uses_storage", TargetExternalID: "ds:" + d})
			}
		}
		recs = append(recs, rec)
	}

	for _, d := range snap.Datastores {
		// free_gb — для vSAN-датастора это и есть свободная ёмкость vSAN (ёмкость vSAN представлена
		// в vSphere как обычный датастор общего доступа для кластера, отдельного API не требуется —
		// E8.3, по решению пользователя: надёжный минимум вместо непроверенной схемы REST).
		recs = append(recs, canon.Record{
			Class: "datastore", SourceName: d.Name, ExternalID: "ds:" + d.ID,
			Attrs: map[string]any{"ds_type": dsType(d.Type), "capacity_gb": math.Round(float64(d.CapacityBytes)/(1<<30)*100) / 100,
				"free_gb": math.Round(float64(d.FreeBytes)/(1<<30)*100) / 100, "naa": nonNil(d.NAA)},
			Vendor: map[string]any{"storage_type": d.Type},
		})
	}
	return append([]canon.Record{manager}, recs...)
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func gib(mib int) int {
	if mib <= 0 {
		return 0
	}
	return max(1, int(math.Round(float64(mib)/1024)))
}

// nz — 0 -> nil: источник не сообщил значение, а не «ноль».
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

// dsType сводит тип датастора vSphere к допустимым значениям ci_datastore.ds_type (vmfs/nfs/vsan);
// прочие типы (vVol, CIFS, OTHER) остаются NULL, а исходное значение — в вендорских сведениях.
func dsType(t string) any {
	switch strings.ToLower(t) {
	case "vmfs":
		return "vmfs"
	case "nfs", "nfs41":
		return "nfs"
	case "vsan":
		return "vsan"
	}
	return nil
}
