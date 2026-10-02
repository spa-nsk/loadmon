// Command scomdump - диагностическая утилита для Microsoft SCOM REST API (по
// образцу tools/storagemetrics -huaweidumpraw): аутентифицируется по записи
// ConfigSCOM из app2.json и сохраняет СЫРЫЕ ответы сервера - чтобы сверить
// реальные схемы ответов и имена свойств/счётчиков с тем, что ожидает пакет
// scommetrics (см. kontext/13-scom-metrics.md). В БД ничего не пишет.
//
// Файл конфигурации ищется так же, как у основного приложения
// (LOADMON_CONFIG или conf/app2.json рядом с бинарником/в текущем каталоге).
// Запись берётся по -name; Enabled=false для утилиты не помеха.
//
// Примеры:
//
//	go run ./tools/scomdump -objects                       # компьютеры класса ClassName
//	go run ./tools/scomdump -info <objectId>               # свойства + связанные объекты
//	go run ./tools/scomdump -counters <objectId>           # список счётчиков компьютера
//	go run ./tools/scomdump -perf <objectId>               # значения нужных счётчиков (/data/performance)
//	go run ./tools/scomdump -method POST -path /data/scomGroups -body '["DisplayName LIKE ''%Windows%''"]'
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/scommetrics"
)

func main() {
	name := flag.String("name", "", "ScomEndpoint.Name из ConfigSCOM (пусто - первая запись)")
	class := flag.String("class", "", "класс для -objects (пусто - ClassName записи или Microsoft.Windows.Computer)")
	objects := flag.Bool("objects", false, "POST /data/scomObjectsByClass")
	info := flag.String("info", "", "GET /data/objectInformation/<objectId>")
	counters := flag.String("counters", "", "GET /data/performanceCounters/<objectId>")
	perf := flag.String("perf", "", "POST /data/performance по нужным scommetrics счётчикам объекта <objectId>")
	duration := flag.Int("duration", 0, "duration для -perf (0 - PerfDuration записи или 60)")
	method := flag.String("method", "GET", "метод для -path")
	path := flag.String("path", "", "произвольный путь относительно Endpoint, например /data/scomGroups")
	body := flag.String("body", "", "тело запроса для -path (JSON)")
	out := flag.String("out", "", "файл для сырого ответа (пусто - stdout)")
	flag.Parse()

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("чтение app2.json: %v", err)
	}
	if len(config.ConfigData.ConfigSCOM) == 0 {
		log.Fatal("в app2.json нет записей ConfigSCOM")
	}
	ep := config.ConfigData.ConfigSCOM[0]
	if *name != "" {
		found := false
		for _, e := range config.ConfigData.ConfigSCOM {
			if e.Name == *name {
				ep, found = e, true
				break
			}
		}
		if !found {
			log.Fatalf("запись ConfigSCOM с Name=%q не найдена", *name)
		}
	}
	c, err := scommetrics.NewClient(ep)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var data []byte
	switch {
	case *objects:
		cl := *class
		if cl == "" {
			cl = ep.ClassName
		}
		if cl == "" {
			cl = "Microsoft.Windows.Computer"
		}
		b, _ := json.Marshal([]string{cl})
		data, err = c.Raw(ctx, http.MethodPost, "/data/scomObjectsByClass", b)
	case *info != "":
		data, err = c.Raw(ctx, http.MethodGet, "/data/objectInformation/"+url.PathEscape(*info), nil)
	case *counters != "":
		data, err = c.Raw(ctx, http.MethodGet, "/data/performanceCounters/"+url.PathEscape(*counters), nil)
	case *perf != "":
		data, err = dumpPerf(ctx, c, *perf, pickDuration(*duration, ep.PerfDuration))
	case *path != "":
		var b []byte
		if *body != "" {
			b = []byte(*body)
		}
		p := *path
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		data, err = c.Raw(ctx, strings.ToUpper(*method), p, b)
	default:
		flag.Usage()
		os.Exit(2)
	}
	if f := c.AuthFormat(); f != "" {
		log.Printf("аутентификация принята в формате %s", f)
	}
	if err != nil {
		log.Printf("ошибка: %v", err)
	}
	if len(data) == 0 {
		if err != nil {
			os.Exit(1)
		}
		return
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, data, "", "  ") == nil {
		data = pretty.Bytes()
	}
	if *out == "" {
		fmt.Println(string(data))
		return
	}
	if werr := os.WriteFile(*out, data, 0644); werr != nil {
		log.Fatal(werr)
	}
	log.Printf("ответ сохранён в %s (%d байт)", *out, len(data))
}

func pickDuration(flagV, cfgV int) int {
	if flagV > 0 {
		return flagV
	}
	if cfgV > 0 {
		return cfgV
	}
	return 60
}

// dumpPerf получает счётчики объекта, оставляет нужные scommetrics и
// возвращает СЫРОЙ ответ /data/performance.
func dumpPerf(ctx context.Context, c *scommetrics.Client, objectID string, duration int) ([]byte, error) {
	all, err := c.PerformanceCounters(ctx, objectID)
	if err != nil {
		return nil, err
	}
	need := scommetrics.NeededCounters(all)
	log.Printf("счётчиков у объекта: %d, нужных для server_metric: %d", len(all), len(need))
	if len(need) == 0 {
		return nil, fmt.Errorf("нет счётчиков, распознаваемых scommetrics/match.go - посмотрите полный список через -counters")
	}
	raws := make([]map[string]interface{}, 0, len(need))
	for _, ct := range need {
		raws = append(raws, ct.Raw)
	}
	b, _ := json.Marshal(map[string]interface{}{
		"id":                  "loadmon-scomdump",
		"duration":            duration,
		"performanceCounters": raws,
	})
	return c.Raw(ctx, http.MethodPost, "/data/performance", b)
}
