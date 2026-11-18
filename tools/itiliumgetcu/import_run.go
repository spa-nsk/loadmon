package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/itilium"
	"loadmon/itilliumimport"
	"loadmon/utilities"

	"github.com/beego/beego/v2/client/orm"
)

func applyItiliumConfig(cfg *config.ItiliumConfig, base, domain, user, password, auth, apiPrefix *string, insecure *bool, timeout *time.Duration, cuStatus *string) {
	if cfg == nil || !cfg.Enabled {
		return
	}
	if cfg.BaseURL != "" && (*base == "" || *base == "https://itsd.global.bcs/ITILIUM") {
		*base = cfg.BaseURL
	}
	if cfg.Domain != "" && (*domain == "" || *domain == "itildomail") {
		*domain = cfg.Domain
	}
	if cfg.Login != "" && (*user == "" || *user == "itiluser") {
		*user = cfg.Login
	}
	if strings.TrimSpace(*password) == "" && cfg.Password != "" && cfg.Password != "password" {
		*password = cfg.Password
	}
	if cfg.AuthMode != "" {
		*auth = cfg.AuthMode
	}
	if cfg.APIPrefix != "" {
		*apiPrefix = cfg.APIPrefix
	}
	if cfg.InsecureSkipVerify {
		*insecure = true
	}
	if cfg.TimeoutSec > 0 {
		*timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}
	if cfg.CUStatus != "" {
		*cuStatus = cfg.CUStatus
	}
}

func resolveItiliumPassword(pass, domain, user string, cfg config.ItiliumConfig) string {
	pass = strings.TrimSpace(pass)
	if pass != "" && pass != "password" {
		return pass
	}
	if p := strings.TrimSpace(os.Getenv("ITILIUM_PASSWORD")); p != "" {
		return p
	}
	if cfg.Password != "" && cfg.Password != "password" {
		return cfg.Password
	}
	key := user
	if d := strings.TrimSpace(domain); d != "" {
		key = d + `\` + user
	}
	p := utilities.GetKeyData(key)
	if p == "key" {
		p = utilities.GetKeyData(user)
	}
	if p == "key" {
		return ""
	}
	return p
}

func initDBForImport() error {
	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		return fmt.Errorf("app2.json: %w", err)
	}
	config.DBInit()
	if _, err := orm.GetDB("default"); err != nil {
		return fmt.Errorf("нет подключения к БД (%w); проверьте app2.json / LOADMON_CONFIG", err)
	}
	return nil
}

func runImportCU(rows []itilium.CURow) itilliumimport.ImportCURackStats {
	log.Printf("импорт rack getCU → БД: rows=%d (справочник locality из cmdb_locality)", len(rows))
	st := itilliumimport.ImportRackServersFromCU(rows)
	log.Printf("импорт завершён: inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}
