package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// InventoryConfig задаёт размеры эмулируемого инвентаря vCenter (число
// датацентров, кластеров, хостов, ВМ и т.д.).
type InventoryConfig struct {
	Datacenter  int `json:"datacenter"`
	Cluster     int `json:"cluster"`
	ClusterHost int `json:"clusterHost"`
	Pool        int `json:"pool"`
	Host        int `json:"host"`
	Machine     int `json:"machine"`
	Datastore   int `json:"datastore"`
	Portgroup   int `json:"portgroup"`
	App         int `json:"app"`
}

// PerfConfig задаёт параметры проверки метрик производительности эмулятора.
type PerfConfig struct {
	VerifyOnStart bool `json:"verifyOnStart"`
}

// LoadmonVCenter описывает параметры подключения loadmon к эмулируемому vCenter
// (используется для генерации фрагмента конфигурации app2.json).
type LoadmonVCenter struct {
	Hostname string `json:"hostname"`
	Username string `json:"username"`
	Password string `json:"password"`
	Wait     int    `json:"wait"`
}

// LoadmonConfig содержит список подключений vCenter для фрагмента конфигурации loadmon.
type LoadmonConfig struct {
	ConfigVCenters []LoadmonVCenter `json:"configVCenters"`
}

// VcsimConfig описывает конфигурацию эмулятора vcsim: адрес прослушивания,
// учётные данные, TLS, инвентарь и параметры проверки.
type VcsimConfig struct {
	ListenAddr string          `json:"listenAddr"`
	Username   string          `json:"username"`
	Password   string          `json:"password"`
	TLS        bool            `json:"tls"`
	Autostart  bool            `json:"autostart"`
	Inventory  InventoryConfig `json:"inventory"`
	Perf       PerfConfig      `json:"perf"`
	Loadmon    LoadmonConfig   `json:"loadmon"`
}

func defaultConfig() VcsimConfig {
	return VcsimConfig{
		ListenAddr: "127.0.0.1:8989",
		Username:   "vcsim@local",
		Password:   "pass",
		TLS:        true,
		Autostart:  true,
		Inventory: InventoryConfig{
			Datacenter:  2,
			Cluster:     5,
			ClusterHost: 8,
			Pool:        0,
			Host:        0,
			Machine:     1000,
			Datastore:   3,
			Portgroup:   1,
			App:         0,
		},
		Perf: PerfConfig{
			VerifyOnStart: true,
		},
		Loadmon: LoadmonConfig{
			ConfigVCenters: []LoadmonVCenter{
				{
					Hostname: "127.0.0.1:8989",
					Username: "vcsim@local",
					Password: "pass",
					Wait:     60,
				},
			},
		},
	}
}

func loadConfig(path string) (VcsimConfig, error) {
	cfg := defaultConfig()
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	cfg.normalize()
	return cfg, nil
}

func (c *VcsimConfig) normalize() {
	if c.ListenAddr == "" {
		c.ListenAddr = "127.0.0.1:8989"
	}
	if c.Username == "" {
		c.Username = "vcsim@local"
	}
	if c.Password == "" {
		c.Password = "pass"
	}
	if c.Inventory.Datacenter <= 0 {
		c.Inventory.Datacenter = 1
	}
	if c.Inventory.Cluster <= 0 {
		c.Inventory.Cluster = 1
	}
	if c.Inventory.ClusterHost <= 0 {
		c.Inventory.ClusterHost = 4
	}
	if c.Inventory.Datastore <= 0 {
		c.Inventory.Datastore = 1
	}
	if len(c.Loadmon.ConfigVCenters) == 0 {
		c.Loadmon.ConfigVCenters = []LoadmonVCenter{
			{
				Hostname: c.ListenAddr,
				Username: c.Username,
				Password: c.Password,
				Wait:     60,
			},
		}
	}
}

func (c *VcsimConfig) applyVMTarget(target int) {
	if target <= 0 {
		return
	}
	inv := &c.Inventory
	if inv.Pool != 0 || inv.App != 0 || inv.Host != 0 {
		return
	}
	den := inv.Datacenter * inv.Cluster
	if den <= 0 {
		return
	}
	inv.Machine = target / den
	if inv.Machine <= 0 {
		inv.Machine = 1
	}
}

func (inv InventoryConfig) expectedVMs() int {
	total := inv.Datacenter * inv.Host * inv.Machine
	total += inv.Datacenter * inv.Cluster * inv.Machine
	if inv.Pool > 0 {
		total += inv.Datacenter * inv.Cluster * inv.Pool * inv.Machine
	}
	if inv.App > 0 {
		total += inv.Datacenter * inv.Cluster * inv.App * inv.Machine
	}
	return total
}

func resolveConfigPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if cwd, err := os.Getwd(); err == nil {
		alt := filepath.Join(cwd, path)
		if _, err := os.Stat(alt); err == nil {
			return alt
		}
	}
	return path
}

func printLoadmonSnippet(cfg VcsimConfig) {
	fragment := struct {
		ConfigVCenters []LoadmonVCenter `json:"ConfigVCenters"`
	}{
		ConfigVCenters: cfg.Loadmon.ConfigVCenters,
	}
	b, err := json.MarshalIndent(fragment, "", "\t")
	if err != nil {
		return
	}
	fmt.Println("Фрагмент для app2.json (ConfigVCenters):")
	fmt.Println(string(b))
	fmt.Println("Добавьте в utilities/keyfunc.go: keymap[\"" + cfg.Username + "\"] = \"" + cfg.Password + "\"")
}
