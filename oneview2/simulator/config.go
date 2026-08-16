package simulator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Config describes one or more independent OneView simulator instances.
type Config struct {
	Instances []InstanceConfig `json:"instances"`
}

// InstanceConfig is a single HTTP listener emulating OneView or Global Dashboard.
type InstanceConfig struct {
	Name        string `json:"name"`
	Listen      string `json:"listen"`
	APIVersion  int    `json:"apiVersion"`
	ServerCount int    `json:"serverCount"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

// LoadConfig reads simulator configuration from a JSON file.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if len(cfg.Instances) == 0 {
		return cfg, fmt.Errorf("simulator: no instances in %s", path)
	}
	for i := range cfg.Instances {
		inst := &cfg.Instances[i]
		if inst.Listen == "" {
			return cfg, fmt.Errorf("simulator: instance[%d] listen is required", i)
		}
		if inst.APIVersion == 0 {
			if inst.Name == "" || strings.Contains(strings.ToLower(inst.Name), "300") {
				inst.APIVersion = 300
			} else {
				inst.APIVersion = 7400
			}
		}
		if inst.ServerCount <= 0 {
			inst.ServerCount = 300
		}
		if inst.Username == "" {
			inst.Username = "Administrator"
		}
		if inst.Password == "" {
			inst.Password = "admin"
		}
		if inst.Name == "" {
			inst.Name = fmt.Sprintf("oneview-%d", inst.APIVersion)
		}
	}
	return cfg, nil
}
