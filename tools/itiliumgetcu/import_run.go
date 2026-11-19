package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"loadmon/config"
	"loadmon/itilium"
	"loadmon/itilliumimport"
	"loadmon/utilities"

	"github.com/beego/beego/v2/client/orm"
)

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

func runImportCU(rows []itilium.CURow, cuClass int) itilliumimport.ImportCUStats {
	className := itilium.CUClassName(cuClass)
	if className == "" {
		className = "?"
	}
	log.Printf("импорт getCU cuClass=%d (%s) → БД: rows=%d", cuClass, className, len(rows))
	var st itilliumimport.ImportCUStats
	switch cuClass {
	case itilium.CUClassServers:
		st = itilliumimport.ImportRackServersFromCU(rows)
	case itilium.CUClassBlades:
		st = itilliumimport.ImportBladeServersFromCU(rows)
	case itilium.CUClassBladeEnclosures:
		st = itilliumimport.ImportBladeEnclosuresFromCU(rows)
	case itilium.CUClassStorage:
		st = itilliumimport.ImportStorageFromCU(rows)
	case itilium.CUClassTapeLibraries:
		st = itilliumimport.ImportTapeLibrariesFromCU(rows)
	case itilium.CUClassNetwork:
		st = itilliumimport.ImportNetworkEquipmentFromCU(rows)
	default:
		log.Printf("импорт getCU: неподдерживаемый cuClass=%d — пропуск", cuClass)
		st.Total = len(rows)
		st.Skipped = len(rows)
	}
	log.Printf("импорт завершён: inserted=%d updated=%d skipped=%d inactive=%d noLoc=%d unknownStatus=%d errors=%d",
		st.Inserted, st.Updated, st.Skipped, st.Inactive, st.NoLoc, st.UnknownSt, st.Errors)
	return st
}
