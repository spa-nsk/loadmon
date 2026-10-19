// Команда localityload загружает справочник населённых пунктов (ОКТМО Росстата)
// в таблицу cmdb_locality. Конфигурация и БД - те же, что у остальных утилит
// (conf/app2.json, переменная LOADMON_CONFIG).
//
//	go run ./tools/localityload -file itilliumimport/parsing/oktmo_data_20261001.csv
package main

import (
	"flag"
	"log"

	"loadmon/config"
	"loadmon/locality"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	file := flag.String("file", "itilliumimport/parsing/oktmo_data_20261001.csv", "файл data-*.csv с открытыми данными ОКТМО")
	flag.Parse()

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("localityload: конфигурация: %v", err)
	}
	config.DBInit()

	n, err := locality.Load(*file)
	if err != nil {
		log.Fatalf("localityload: %v", err)
	}
	log.Printf("localityload: загружено населённых пунктов: %d", n)
}
