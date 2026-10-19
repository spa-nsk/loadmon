// Команда localityload загружает справочник населённых пунктов (ОКТМО Росстата)
// в таблицу cmdb_locality. Конфигурация и БД - те же, что у остальных утилит
// (conf/app2.json, переменная LOADMON_CONFIG).
//
// Запускать из корня репозитория (или указать LOADMON_CONFIG), иначе подхватится
// чужой conf/app2.json из текущего каталога:
//
//	go run ./tools/localityload -file itilliumimport/parsing/oktmo_data_20261001.csv
//	set LOADMON_CONFIG=C:\path\to\conf\app2.json
package main

import (
	"flag"
	"log"

	"loadmon/config"
	"loadmon/locality"

	"github.com/beego/beego/v2/client/orm"

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

	// DBInit не останавливается, если БД отвергла подключение (например,
	// неверный пароль) - алиас "default" тогда просто не регистрируется, а
	// NewOrm() падает с panic "unknown db alias name 'default'".
	if _, err := orm.GetDB("default"); err != nil {
		log.Fatalf("localityload: нет подключения к БД (%v). Проверьте пользователя/пароль в загруженном app2.json (см. строку config: загружен ...) либо задайте LOADMON_CONFIG на рабочий conf/app2.json и запускайте из корня репозитория", err)
	}

	n, err := locality.Load(*file)
	if err != nil {
		log.Fatalf("localityload: %v", err)
	}
	log.Printf("localityload: загружено населённых пунктов: %d", n)
}
