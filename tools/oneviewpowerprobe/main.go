// Command oneviewpowerprobe — диагностика мощности HPE OneView (вариант C):
// сырой JSON enclosure / environmentalConfiguration и разбор oneview2.PowerWatts.
// В БД не пишет. Конфиг — как у приложения (LOADMON_CONFIG или conf/app2.json).
//
// Примеры:
//
//	go run ./tools/oneviewpowerprobe -endpoint https://oneview.example -list-enclosures
//	go run ./tools/oneviewpowerprobe -enclosure "c7000-01"
//	go run ./tools/oneviewpowerprobe -server-uri /rest/server-hardware/abc123
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"loadmon/cmdb"
	"loadmon/config"
	models "loadmon/models"
	"loadmon/utilities"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

func main() {
	endpointFlag := flag.String("endpoint", "", "OvEndpoint.Endpoint из ConfigOV (пусто — первая запись)")
	listEnc := flag.Bool("list-enclosures", false, "список корзин и parsed enclosure power")
	encFlag := flag.String("enclosure", "", "имя или URI/id корзины для GET /rest/enclosures/{id}")
	serverURI := flag.String("server-uri", "", "URI server-hardware для export + environmentalConfiguration")
	serverName := flag.String("server", "", "имя server-hardware (поиск в списке)")
	rawOnly := flag.Bool("raw", false, "только сырой JSON без summary")
	diagBlade := flag.String("diag-blade", "", "имя корзины: сравнение enclosure power vs sum(bлейдов) (нужна БД loadmon)")
	out := flag.String("out", "", "файл для вывода (пусто — stdout)")
	flag.Parse()

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("чтение конфигурации: %v", err)
	}
	if len(config.ConfigData.ConfigOV) == 0 {
		log.Fatal("в конфиге нет ConfigOV")
	}
	ovi := config.ConfigData.ConfigOV[0]
	if *endpointFlag != "" {
		found := false
		for _, e := range config.ConfigData.ConfigOV {
			if strings.EqualFold(strings.TrimSpace(e.Endpoint), strings.TrimSpace(*endpointFlag)) {
				ovi, found = e, true
				break
			}
		}
		if !found {
			log.Fatalf("ConfigOV Endpoint=%q не найден", *endpointFlag)
		}
	}

	// Пароль - тем же способом, что в config.InitParams: фиктивное значение
	// "password" в ConfigOV заменяется паролем из хранилища ключей
	// (utilities.GetKeyData); остальные значения используются как есть.
	// LOADMON_OV_PASSWORD - необязательное переопределение для отладки.
	pwd := ovi.Password
	if pwd == "password" {
		pwd = utilities.GetKeyData("usr-nsk-dati-02@global.bcs")
	}
	if v := os.Getenv("LOADMON_OV_PASSWORD"); v != "" {
		pwd = v
	}
	cfg := oneview.NewEndpointConfig(ovi.Endpoint, ovi.Domain, ovi.Login, pwd, ovi.Version, config.ConfigData.OneViewOperationTimeoutFor(&ovi))
	c, err := oneview.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := c.Login(ctx); err != nil {
		log.Fatal(err)
	}
	defer c.Logout(ctx)

	var lines []string
	switch {
	case *listEnc:
		col, err := c.ListEnclosures(ctx, oneview.ListOptions{Count: -1})
		if err != nil {
			log.Fatal(err)
		}
		for i := range col.Members {
			enc := col.Members[i]
			p := oneview.EnclosurePowerWatts(&enc)
			lines = append(lines, fmt.Sprintf("%s  model=%s  avg=%.0f peak=%.0f rated=%.0f  uri=%s",
				enc.Name, enc.EnclosureModel, p.Avg24hW, p.Peak24hW, p.RatedMaxW, enc.URI))
			if *rawOnly {
				b, _ := json.MarshalIndent(enc, "", "  ")
				lines = append(lines, string(b))
			}
		}
	case *encFlag != "":
		id := resolveEnclosureID(ctx, c, *encFlag)
		enc, err := c.GetEnclosure(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		if *rawOnly {
			b, _ := json.MarshalIndent(enc, "", "  ")
			lines = append(lines, string(b))
		} else {
			p := oneview.EnclosurePowerWatts(enc)
			lines = append(lines, fmt.Sprintf("parsed enclosure power: avg=%.0f peak=%.0f rated=%.0f source=%s",
				p.Avg24hW, p.Peak24hW, p.RatedMaxW, p.Source))
			b, _ := json.MarshalIndent(enc, "", "  ")
			lines = append(lines, string(b))
		}
	case *diagBlade != "":
		config.DBInit()
		o := orm.NewOrm()
		name := bladeSystemNameFromProbe(*diagBlade)
		bls := models.BladeSystems{Name: name}
		if err := o.Read(&bls, "Name"); err != nil {
			log.Fatalf("корзина %q не найдена в БД: %v", name, err)
		}
		id := resolveEnclosureID(ctx, c, *diagBlade)
		enc, err := c.GetEnclosure(ctx, id)
		if err != nil {
			log.Fatal(err)
		}
		lines = append(lines, cmdb.BladeSystemPowerDiagnostics(o, enc, bls.Id))
		lines = append(lines, fmt.Sprintf("DB BladeSystems Id=%d avg=%.0f peak=%.0f rated=%.0f",
			bls.Id, bls.PowerAvg24hW, bls.PowerPeak24hW, bls.PowerRatedMaxW))
	case *serverURI != "" || *serverName != "":
		uri := *serverURI
		if uri == "" {
			uri = findServerURI(ctx, c, *serverName)
		}
		exp, err := c.ExportServer(ctx, uri, oneview.ExportOptions{
			SkipFirmware:     true,
			SkipLocalStorage: true,
			SkipBIOS:         true,
			SkipProfile:      true,
			SkipHardwareType: true,
			SkipEnclosure:    true,
		})
		if err != nil {
			log.Fatal(err)
		}
		p := oneview.DeriveServerPower(exp)
		if !*rawOnly {
			lines = append(lines, fmt.Sprintf("server %s powerState=%s parsed: avg=%.0f peak=%.0f rated=%.0f source=%s",
				exp.Identity.Name, exp.Identity.PowerState, p.Avg24hW, p.Peak24hW, p.RatedMaxW, p.Source))
		}
		if len(exp.Environment) > 0 {
			if *rawOnly {
				lines = append(lines, string(exp.Environment))
			} else {
				lines = append(lines, "environmentalConfiguration:")
				var pretty json.RawMessage = exp.Environment
				b, _ := json.MarshalIndent(pretty, "", "  ")
				lines = append(lines, string(b))
			}
		} else {
			lines = append(lines, "(environmentalConfiguration пуст или не запрошен)")
		}
	default:
		flag.Usage()
		log.Fatal("укажите -list-enclosures, -enclosure, -server-uri или -server")
	}

	text := strings.Join(lines, "\n")
	if *out != "" {
		if err := os.WriteFile(*out, []byte(text+"\n"), 0o644); err != nil {
			log.Fatal(err)
		}
		return
	}
	fmt.Print(text)
	if !strings.HasSuffix(text, "\n") {
		fmt.Println()
	}
}

func resolveEnclosureID(ctx context.Context, c *oneview.Client, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		log.Fatal("пустой -enclosure")
	}
	if strings.HasPrefix(ref, "/rest/enclosures/") {
		return ref
	}
	col, err := c.ListEnclosures(ctx, oneview.ListOptions{Count: -1})
	if err != nil {
		log.Fatal(err)
	}
	lower := strings.ToLower(ref)
	for i := range col.Members {
		enc := col.Members[i]
		if strings.EqualFold(enc.Name, ref) || strings.EqualFold(enc.URI, ref) || strings.EqualFold(oneview.IDFromURI(enc.URI), ref) {
			return enc.URI
		}
		if strings.ToLower(enc.Name) == lower {
			return enc.URI
		}
	}
	log.Fatalf("корзина %q не найдена", ref)
	return ""
}

func bladeSystemNameFromProbe(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "-")
	if len(parts) == 2 {
		if num, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			return fmt.Sprintf("%s-%02d", strings.TrimSpace(parts[0]), num)
		}
	}
	return strings.TrimSpace(name)
}

func findServerURI(ctx context.Context, c *oneview.Client, name string) string {
	col, err := c.ListServerHardware(ctx, oneview.ListOptions{Count: -1})
	if err != nil {
		log.Fatal(err)
	}
	for i := range col.Members {
		hw := col.Members[i]
		if strings.EqualFold(hw.Name, name) || strings.EqualFold(hw.ServerName, name) {
			return hw.URI
		}
	}
	log.Fatalf("server-hardware %q не найден", name)
	return ""
}
