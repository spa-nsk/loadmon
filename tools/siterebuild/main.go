// Команда siterebuild пересоздаёт справочник площадок/шкафов с нуля из свежего
// импорта Итиллиум (см. itilliumimport/rebuildsites.go).
//
// Порядок работы:
//
//  1. Очистка (по умолчанию - пробный прогон, показывает сколько записей затронет):
//     go run ./tools/siterebuild -clear
//     go run ./tools/siterebuild -clear -apply     # резервная копия + очистка
//     Очистка: обнуляет привязки серверов, блейд-корзин и коммутаторов к шкафам,
//     удаляет размещения СХД, шкафы, площадки, историю мощности шкафов и площадок.
//     Перед очисткой ВСЕГДА пишется резервная копия (файл -backup).
//  2. Импорт четырёх выгрузок Итиллиум через меню Администрирование -> Импорт
//     (rack-серверы, blade-корзины, blade-серверы, СХД).
//  3. Возврат ручных атрибутов (признак ЦОД, код Jira, размеры, данные шкафов,
//     привязки коммутаторов) из резервной копии:
//     go run ./tools/siterebuild -restore siterebuild_backup_....json -apply
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"loadmon/config"
	"loadmon/itilliumimport"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	clearFlag := flag.Bool("clear", false, "очистить привязки, шкафы, площадки и историю мощности")
	restore := flag.String("restore", "", "файл резервной копии: вернуть ручные атрибуты после повторного импорта")
	backup := flag.String("backup", fmt.Sprintf("siterebuild_backup_%s.json", time.Now().Format("20060102_150405")), "файл резервной копии")
	apply := flag.Bool("apply", false, "применить (по умолчанию - пробный прогон)")
	flag.Parse()

	if !*clearFlag && *restore == "" {
		flag.Usage()
		os.Exit(2)
	}

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("siterebuild: конфигурация: %v", err)
	}
	if err := config.ConfigData.InitParams(); err != nil {
		log.Printf("InitParams: %v", err)
	}
	httpKey := config.ConfigData.HttpKey
	httpsKey := config.ConfigData.HttpsKey
	httpEnv := os.Getenv(httpKey)
	httpsEnv := os.Getenv(httpsKey)
	defer os.Setenv(httpKey, httpEnv)
	defer os.Setenv(httpsKey, httpsEnv)
	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	config.DBInit()
	if _, err := orm.GetDB("default"); err != nil {
		log.Fatalf("siterebuild: нет подключения к БД (%v) - проверьте пользователя/пароль в conf/app2.json", err)
	}

	if *clearFlag {
		if *apply {
			b, err := itilliumimport.BackupSites(*backup)
			if err != nil {
				log.Fatalf("siterebuild: резервная копия не создана, очистка отменена: %v", err)
			}
			log.Printf("siterebuild: резервная копия %s (площадок %d, шкафов %d, коммутаторов с привязкой %d)",
				*backup, len(b.Sites), len(b.Cabinets), len(b.Switches))
		}
		rep, err := itilliumimport.ClearSites(!*apply)
		if err != nil {
			log.Fatalf("siterebuild: %v", err)
		}
		mode := "ПРОБНЫЙ ПРОГОН (ничего не изменено, для применения добавьте -apply)"
		if *apply {
			mode = "ОЧИЩЕНО"
		}
		log.Printf("siterebuild: %s. Серверов отвязано %d, блейд-корзин %d, коммутаторов (шкаф) %d; удалено: размещений СХД %d, шкафов %d, площадок %d, истории мощности шкафов %d и площадок %d",
			mode, rep.ServersUnlinked, rep.BladesUnlinked, rep.SwitchesUnlinked, rep.StoragePlacements, rep.Cabinets, rep.Sites, rep.RackPower, rep.SitePower)
		if *apply {
			log.Printf("siterebuild: теперь загрузите 4 выгрузки Итиллиум, затем: go run ./tools/siterebuild -restore %s -apply", *backup)
		}
	}

	if *restore != "" {
		rep, err := itilliumimport.RestoreManual(*restore, !*apply)
		if err != nil {
			log.Fatalf("siterebuild: %v", err)
		}
		for _, l := range rep.Lines {
			log.Println(l)
		}
		mode := "ПРОБНЫЙ ПРОГОН"
		if *apply {
			mode = "ПРИМЕНЕНО"
		}
		log.Printf("siterebuild: %s. Площадок восстановлено %d (не найдено %d), шкафов %d (не найдено %d), коммутаторов перепривязано %d",
			mode, rep.SitesRestored, rep.SitesNotFound, rep.CabinetsRestored, rep.CabinetsNotFound, rep.SwitchesRelinked)
	}
}
