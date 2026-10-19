// Команда sitemigrate обновляет ранее загруженные из Итиллиум площадки/ЦОД и
// шкафы под новый разбор "Местоположения" (см. itilliumimport.MigrateSites):
// населённый пункт -> код ОКТМО, описание -> альтернативное наименование,
// улица+дом -> адрес, этаж, комната, "альт / шкаф" -> AltName + чистый Name.
// Справочник населённых пунктов (cmdb_locality) должен быть уже загружен.
//
// По умолчанию - пробный прогон (БД не меняется); для применения -apply:
//
//	go run ./tools/sitemigrate            # посмотреть, что изменится
//	go run ./tools/sitemigrate -apply     # применить
package main

import (
	"flag"
	"log"

	"loadmon/config"
	"loadmon/itilliumimport"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	apply := flag.Bool("apply", false, "применить изменения (по умолчанию - пробный прогон)")
	flag.Parse()

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("sitemigrate: конфигурация: %v", err)
	}
	config.DBInit()
	if _, err := orm.GetDB("default"); err != nil {
		log.Fatalf("sitemigrate: нет подключения к БД (%v) - проверьте пользователя/пароль в conf/app2.json", err)
	}

	rep, err := itilliumimport.MigrateSites(!*apply)
	if err != nil {
		log.Fatalf("sitemigrate: %v", err)
	}
	for _, l := range rep.Lines {
		log.Println(l)
	}
	mode := "ПРОБНЫЙ ПРОГОН (ничего не изменено, для применения добавьте -apply)"
	if *apply {
		mode = "ПРИМЕНЕНО"
	}
	log.Printf("sitemigrate: %s. Площадок: всего %d, обновлено %d, пропущено %d, без найденного населённого пункта %d; шкафов обновлено %d; дублей площадок %d",
		mode, rep.SitesTotal, rep.SitesUpdated, rep.SitesSkipped, rep.SitesNoLocality, rep.CabinetsUpdated, rep.Duplicates)
}
