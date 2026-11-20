// Command itiliumgetcu — ручной запуск синхронизации getCU
// (та же itilliumimport.SyncGetCU, что вызывается из metrics_grpc).
//
//	go run ./tools/itiliumgetcu
//
// Нужны app2.json (ConfigItilium.Enabled + SyncRackServers) и пароль
// utilities.GetKeyData(Login).
package main

import (
	"log"

	"loadmon/config"
	"loadmon/itilliumimport"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("app2.json: %v", err)
	}
	config.DBInit()
	itilliumimport.SyncGetCU()
}
