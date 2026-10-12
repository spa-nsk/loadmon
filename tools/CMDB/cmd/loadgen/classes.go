package main

import (
	"fmt"
	"math/rand"
)

// Построители таблиц-расширений — по одному на класс, что реально нужен для SLA-проверки
// (список/карточка/на дату), см. cmd/loadgen/main.go, classSpecs. Малые/второстепенные классы
// (share, qtree, db_instance, tape_library, backup_server, facility_*) — только голая cmdb.ci,
// без атрибутов класса (у них и в бою нет формы атрибутов без ClassSpec — не искажает нагрузку на
// список/карточку/поиск, только не даёт вкладки «Атрибуты класса», которой у них и так нет).

func extServer(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "form_factor", "cpu_count", "core_count", "ram_gb", "os_family"}
	rows := make([][]any, len(ids))
	forms := []string{"rack", "blade", "tower"}
	families := []string{"Linux", "Windows", "Unix", "Other"}
	for i, id := range ids {
		rows[i] = []any{id, forms[rand.Intn(len(forms))], 2, 32, 128, families[rand.Intn(len(families))]}
	}
	return "ci_server", cols, rows
}

func extHypervisor(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "hv_type", "platform", "hv_version"}
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, "esxi", "vSphere", "8.0.3 build 24022510"}
	}
	return "ci_hypervisor", cols, rows
}

func extVM(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "vcpu", "ram_gb", "guest_os", "template"}
	rows := make([][]any, len(ids))
	guests := []string{"Ubuntu Server 24.04", "РЕД ОС 8", "Astra Linux 1.8", "Windows Server 2022"}
	for i, id := range ids {
		rows[i] = []any{id, 1 + rand.Intn(16), 2 + rand.Intn(64), guests[rand.Intn(len(guests))], false}
	}
	return "ci_vm", cols, rows
}

func extStorageArray(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "array_family", "controllers", "raw_tb"}
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, "Huawei OceanStor Dorado", 2, 100 + rand.Float64()*400}
	}
	return "ci_storage_array", cols, rows
}

func extFCSwitch(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "fabric", "domain_id", "ports_total"}
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, "fabric-a", 1 + rand.Intn(200), 48}
	}
	return "ci_fc_switch", cols, rows
}

func extNetworkDevice(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "device_role", "os_version"}
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, "access", "v1.2.3"}
	}
	return "ci_network_device", cols, rows
}

func extCluster(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "cluster_kind", "ha", "drs", "platform"}
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, "virt", true, true, "vSphere"}
	}
	return "ci_cluster", cols, rows
}

func extDatastore(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "ds_type", "capacity_gb"}
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, "vmfs", 500 + rand.Float64()*9500}
	}
	return "ci_datastore", cols, rows
}

func extVolume(ids []int64) (string, []string, [][]any) {
	cols := []string{"ci_id", "size_gb", "thin", "protocol"}
	rows := make([][]any, len(ids))
	for i, id := range ids {
		rows[i] = []any{id, 10 + rand.Float64()*990, true, "block"}
	}
	return "ci_volume", cols, rows
}

// compsPerCI/ifacesPerCI — сколько компонентов/интерфейсов на одну КЕ класса; примерное
// соответствие intent.md §9 (сервер ≈ 40 компонентов, СХД ≈ 250, ВМ ≈ 3; порты коммутаторов —
// пример «1500 × 48» в интенте) для классов, участвующих в нагрузке; достигнутый суммарный объём
// — не точно 250к/120-150к, а тот же порядок величины, см. итоговый отчёт о прогоне.
func compsPerCI(class string) int {
	switch class {
	case "server", "hypervisor_host":
		return 40
	case "storage_array":
		return 250
	case "vm":
		return 3
	case "net_switch", "net_firewall", "net_loadbalancer", "net_wifi_ap", "net_proxy", "net_mgmt":
		return 5
	}
	return 0
}

func ifacesPerCI(class string) int {
	switch class {
	case "server", "hypervisor_host":
		return 2
	case "net_switch", "net_firewall", "net_loadbalancer", "net_wifi_ap", "net_proxy", "net_mgmt", "fc_switch":
		return 48
	}
	return 0
}

var componentTypesByClass = map[string][]string{
	"server":           {"cpu", "cpu", "dimm", "dimm", "disk", "psu"},
	"hypervisor_host":  {"cpu", "cpu", "dimm", "dimm", "disk", "psu"},
	"storage_array":    {"disk", "disk", "disk", "raid_ctrl", "psu"},
	"vm":               {"disk"},
	"net_switch":       {"psu", "fan", "module"},
	"net_firewall":     {"psu", "fan", "module"},
	"net_loadbalancer": {"psu", "fan", "module"},
	"net_wifi_ap":      {"psu", "fan", "module"},
	"net_proxy":        {"psu", "fan", "module"},
	"net_mgmt":         {"psu", "fan", "module"},
}

func componentTypeAt(class string, i, n int) string {
	types := componentTypesByClass[class]
	if len(types) == 0 {
		return "module"
	}
	return types[i%len(types)]
}

func serialFor(ciID int64, i int) string {
	return fmt.Sprintf("LG-%d-%d", ciID, i)
}
