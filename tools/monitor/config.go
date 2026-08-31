package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config — параметры мониторинга loadmon (metrics_grpc и server_web).
type Config struct {
	CheckIntervalSec int        `json:"check_interval_sec"`
	AlertCooldownSec int        `json:"alert_cooldown_sec"`
	GRPC             GRPCTarget `json:"grpc"`
	HTTP             HTTPTarget `json:"http"`
	Email            EmailConfig `json:"email"`
}

type GRPCTarget struct {
	Addr       string `json:"addr"`
	Port       int    `json:"port"`
	TimeoutSec int    `json:"timeout_sec"`
}

type HTTPTarget struct {
	URL           string `json:"url"`
	TimeoutSec    int    `json:"timeout_sec"`
	ExpectStatus  int    `json:"expect_status"`
}

type EmailConfig struct {
	SMTPHost string   `json:"smtp_host"`
	SMTPPort int      `json:"smtp_port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
	UseTLS   bool     `json:"use_tls"`
}

func (c *Config) applyDefaults() {
	if c.CheckIntervalSec <= 0 {
		c.CheckIntervalSec = 60
	}
	if c.AlertCooldownSec <= 0 {
		c.AlertCooldownSec = 300
	}
	if c.GRPC.Addr == "" {
		c.GRPC.Addr = "127.0.0.1"
	}
	if c.GRPC.Port == 0 {
		c.GRPC.Port = 50063
	}
	if c.GRPC.TimeoutSec <= 0 {
		c.GRPC.TimeoutSec = 10
	}
	if c.HTTP.URL == "" {
		c.HTTP.URL = "http://127.0.0.1:9001/metrics"
	}
	if c.HTTP.TimeoutSec <= 0 {
		c.HTTP.TimeoutSec = 10
	}
	if c.HTTP.ExpectStatus == 0 {
		c.HTTP.ExpectStatus = 200
	}
	if c.Email.SMTPPort == 0 {
		c.Email.SMTPPort = 587
	}
}

func (c *Config) validate() error {
	if c.Email.SMTPHost == "" {
		return fmt.Errorf("email.smtp_host обязателен")
	}
	if c.Email.From == "" {
		return fmt.Errorf("email.from обязателен")
	}
	if len(c.Email.To) == 0 {
		return fmt.Errorf("email.to должен содержать хотя бы один адрес")
	}
	return nil
}

func (g GRPCTarget) address() string {
	return fmt.Sprintf("%s:%d", g.Addr, g.Port)
}

func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("разбор JSON %s: %w", path, err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func resolveConfigPath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	candidates := []string{
		"conf/monitor.json",
		"tools/monitor/conf/monitor.json",
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append([]string{
			filepath.Join(dir, "conf", "monitor.json"),
			filepath.Join(dir, "monitor.json"),
		}, candidates...)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "conf/monitor.json"
}
