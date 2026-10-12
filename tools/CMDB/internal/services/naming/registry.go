// Package naming реализует профиль именования CMDB-NC-2026 (intent.md §5): построение и
// разбор имён КЕ, проверку статуса кода класса (active/deprecated/retired) и лимитов длины.
//
// Registry сейчас заполняется литералами Go, синхронизированными вручную с
// database/seed/nc2026/002_naming_rules_and_designations.sql. Перевод на загрузку из
// ref.designation/ref.naming_rule через internal/repo — задача E1.8 (версионирование профиля
// в БД); текущий вид достаточен для полного покрытия правил профиля 1.0 тестами без БД.
package naming

import "strings"

// Designation — код класса объекта (CLS) в шаблоне <SITE>-<CLS>-<SEQ>.
type Designation struct {
	Code        string // например "S", "CL", "SA"
	ObjectClass string
	SeqLength   int    // ширина номера в цифрах (3 или 5 по intent §5.3)
	Active      bool   // true — выдаётся новым КЕ; false — только распознаётся (deprecated/retired)
	ReplacedBy  string // код-замена, если Active == false
}

// Registry — набор кодов классов, активных в профиле (плюс распознаваемые устаревшие для теста AC-11).
type Registry struct {
	byCode map[string]Designation
}

// NewRegistry строит реестр из списка Designation (коды должны быть в верхнем регистре).
func NewRegistry(items []Designation) *Registry {
	r := &Registry{byCode: make(map[string]Designation, len(items))}
	for _, d := range items {
		r.byCode[strings.ToUpper(d.Code)] = d
	}
	return r
}

// Lookup возвращает описание кода класса (регистронезависимо).
func (r *Registry) Lookup(code string) (Designation, bool) {
	d, ok := r.byCode[strings.ToUpper(code)]
	return d, ok
}

// DefaultRegistry — активные коды классов CMDB-NC-2026 (intent §5.3) плюс иллюстративный
// устаревший код "FC" (intent §5.6: Firewall Cluster → заменён CL) для проверки AC-11.
// Полный список устаревших кодов Guide переносится сюда/в БД в E1.4.
func DefaultRegistry() *Registry {
	return NewRegistry([]Designation{
		{Code: "S", ObjectClass: "server", SeqLength: 5, Active: true},
		{Code: "H", ObjectClass: "hypervisor_host", SeqLength: 5, Active: true},
		{Code: "V", ObjectClass: "vm", SeqLength: 5, Active: true},
		{Code: "VD", ObjectClass: "vdi", SeqLength: 5, Active: true},
		{Code: "BC", ObjectClass: "blade_chassis", SeqLength: 3, Active: true},
		{Code: "CL", ObjectClass: "cluster", SeqLength: 3, Active: true},
		{Code: "K", ObjectClass: "k8s_node", SeqLength: 5, Active: true},
		{Code: "D", ObjectClass: "desktop", SeqLength: 5, Active: true},
		{Code: "L", ObjectClass: "laptop", SeqLength: 5, Active: true},
		{Code: "T", ObjectClass: "thin_client", SeqLength: 5, Active: true},
		{Code: "MB", ObjectClass: "mobile_device", SeqLength: 5, Active: true},
		{Code: "SA", ObjectClass: "storage_array", SeqLength: 3, Active: true},
		{Code: "NS", ObjectClass: "nas", SeqLength: 3, Active: true},
		{Code: "SO", ObjectClass: "object_store", SeqLength: 3, Active: true},
		{Code: "FS", ObjectClass: "fc_switch", SeqLength: 3, Active: true},
		{Code: "TL", ObjectClass: "tape_library", SeqLength: 3, Active: true},
		{Code: "BK", ObjectClass: "backup_server", SeqLength: 3, Active: true},
		{Code: "ES", ObjectClass: "net_switch", SeqLength: 3, Active: true},
		{Code: "FW", ObjectClass: "net_firewall", SeqLength: 3, Active: true},
		{Code: "LB", ObjectClass: "net_loadbalancer", SeqLength: 3, Active: true},
		{Code: "AP", ObjectClass: "net_wifi_ap", SeqLength: 3, Active: true},
		{Code: "PX", ObjectClass: "net_proxy", SeqLength: 3, Active: true},
		{Code: "NM", ObjectClass: "net_mgmt", SeqLength: 3, Active: true},
		{Code: "UP", ObjectClass: "facility_ups", SeqLength: 3, Active: true},
		{Code: "PD", ObjectClass: "facility_pdu", SeqLength: 3, Active: true},
		{Code: "CU", ObjectClass: "facility_cooling", SeqLength: 3, Active: true},
		{Code: "TS", ObjectClass: "facility_console", SeqLength: 3, Active: true},
		{Code: "NT", ObjectClass: "facility_ntp", SeqLength: 3, Active: true},
		{Code: "HSM", ObjectClass: "facility_hsm", SeqLength: 3, Active: true},
		// Устаревшие коды Guide 2019 (intent §5.6, E1.4) — распознаются (StatusLegacy), но не
		// выдаются новым КЕ (BuildHostName отказывает, см. AC-11). Сопоставление — по §5.6.
		{Code: "C", ObjectClass: "net_proxy", SeqLength: 3, Active: false, ReplacedBy: "PX"},  // ISA/NetCache Proxy
		{Code: "CC", ObjectClass: "net_proxy", SeqLength: 3, Active: false, ReplacedBy: "PX"}, // ISA Proxy Cluster
		{Code: "CM", ObjectClass: "net_proxy", SeqLength: 3, Active: false, ReplacedBy: "PX"}, // ISA Cache Mgmt Server
		{Code: "F", ObjectClass: "net_firewall", SeqLength: 3, Active: false, ReplacedBy: "FW"},
		{Code: "FC", ObjectClass: "net_firewall", SeqLength: 3, Active: false, ReplacedBy: "CL"},
		{Code: "FM", ObjectClass: "net_firewall", SeqLength: 3, Active: false, ReplacedBy: "FW"},  // Firewall Mgmt / Provider-1 MDS/MLM
		{Code: "E", ObjectClass: "net_firewall", SeqLength: 5, Active: false, ReplacedBy: "FW"},   // VPN End Point
		{Code: "EC", ObjectClass: "net_firewall", SeqLength: 5, Active: false, ReplacedBy: "CL"},  // VPN End Point Cluster
		{Code: "RA", ObjectClass: "net_firewall", SeqLength: 5, Active: false, ReplacedBy: "FW"},  // SSL VPN (hardened Linux)
		{Code: "VCM", ObjectClass: "net_firewall", SeqLength: 5, Active: false, ReplacedBy: "FW"}, // Provider-1 CMA
		{Code: "VCL", ObjectClass: "net_firewall", SeqLength: 5, Active: false, ReplacedBy: "FW"}, // Provider-1 CLM
		{Code: "WF", ObjectClass: "net_proxy", SeqLength: 3, Active: false, ReplacedBy: "PX"},     // Web Filter
		{Code: "WCG", ObjectClass: "net_proxy", SeqLength: 3, Active: false, ReplacedBy: "PX"},    // Content Gateway
		{Code: "N", ObjectClass: "server", SeqLength: 5, Active: false, ReplacedBy: "S"},          // Linux (ОС — атрибут в nc2026)
		{Code: "U", ObjectClass: "server", SeqLength: 5, Active: false, ReplacedBy: "S"},          // Unix
		{Code: "NA", ObjectClass: "nas", SeqLength: 3, Active: false, ReplacedBy: "NS"},           // Legacy GI-S NetApp
		{Code: "VC", ObjectClass: "vm", SeqLength: 5, Active: false, ReplacedBy: "V"},             // GI-NG Virtual Machine (Vista)
		{Code: "XD", ObjectClass: "vdi", SeqLength: 5, Active: false, ReplacedBy: "VD"},           // XenDesktop
		{Code: "M", ObjectClass: "asset", SeqLength: 4, Active: false},                            // Maintainable Item — замена: атрибут asset_tag
		{Code: "B", ObjectClass: "asset", SeqLength: 4, Active: false},                            // Billable Item — замена: атрибут asset_tag
		{Code: "AG", ObjectClass: "net_gateway", SeqLength: 3, Active: false},                     // Access Gateway
	})
}

// ByClass возвращает активный код класса объекта для класса КЕ (например, "server" → "S").
func (r *Registry) ByClass(objectClass string) (Designation, bool) {
	for _, d := range r.byCode {
		if d.Active && d.ObjectClass == objectClass {
			return d, true
		}
	}
	return Designation{}, false
}
