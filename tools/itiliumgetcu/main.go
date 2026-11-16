// Command itiliumgetcu — тестовый вызов кастомного HTTP-сервиса 1С:ITILIUM
// `/hs/api/getCU` через пакет loadmon/itilium (доменная NTLM-авторизация).
//
// Пример (BCS DEV), как в запросе:
//
//	go run ./tools/itiliumgetcu ^
//	  -base https://itsd.global.bcs/DEV ^
//	  -domain itildomail -user itiluser -password "%ITILIUM_PASSWORD%" ^
//	  -cu-class 145 -next-ver 1 -limit 50 ^
//	  -fields code,status,modelServ,hostname,ip,linksUnit,sn,user,where,statusHistory ^
//	  -out getcu.json
//
// Пароль: флаг -password, иначе переменная окружения ITILIUM_PASSWORD,
// иначе utilities.GetKeyData(domain\user) / GetKeyData(user).
//
// В БД ничего не пишет. Нужен доступ к DNS/сети itsd.global.bcs.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"loadmon/itilium"
	"loadmon/utilities"
)

func main() {
	base := flag.String("base", "https://itsd.global.bcs/DEV", "корень публикации 1С (без /hs/...)")
	domain := flag.String("domain", "itildomail", "AD-домен")
	user := flag.String("user", "itiluser", "логин без домена")
	password := flag.String("password", "", "пароль (иначе ITILIUM_PASSWORD / GetKeyData)")
	auth := flag.String("auth", itilium.AuthNTLM, "режим: ntlm (доменный) или basic")
	apiPrefix := flag.String("api-prefix", itilium.DefaultAPIPrefix, "префикс HTTP-сервиса относительно base")
	cuClass := flag.Int("cu-class", 145, "cuClass")
	nextVer := flag.Int("next-ver", 1, "nextVer (0 - не передавать)")
	limit := flag.Int("limit", 50, "limit")
	fields := flag.String("fields", strings.Join(itilium.DefaultServerCUFields, ","), "список полей через запятую")
	insecure := flag.Bool("insecure", false, "пропуск проверки TLS-сертификата")
	timeout := flag.Duration("timeout", 2*time.Minute, "таймаут HTTP")
	out := flag.String("out", "", "файл для сырого JSON (пусто - stdout)")
	pretty := flag.Bool("pretty", true, "pretty-print JSON в stdout при отсутствии -out")
	flag.Parse()

	pass := strings.TrimSpace(*password)
	if pass == "" {
		pass = strings.TrimSpace(os.Getenv("ITILIUM_PASSWORD"))
	}
	if pass == "" {
		key := *user
		if d := strings.TrimSpace(*domain); d != "" {
			key = d + `\` + *user
		}
		pass = utilities.GetKeyData(key)
		if pass == "key" {
			pass = utilities.GetKeyData(*user)
		}
		if pass == "key" {
			log.Fatal("пароль не задан: -password, ITILIUM_PASSWORD или ключ в utilities.GetKeyData")
		}
	}

	c, err := itilium.NewClient(itilium.Config{
		Name:               "getcu",
		BaseURL:            *base,
		Login:              *user,
		Password:           pass,
		Domain:             *domain,
		AuthMode:           *auth,
		APIPrefix:          *apiPrefix,
		Timeout:            *timeout,
		InsecureSkipVerify: *insecure,
	})
	if err != nil {
		log.Fatal(err)
	}

	q := itilium.GetCUQuery{
		NextVer: *nextVer,
		CUClass: *cuClass,
		Limit:   *limit,
	}
	if f := strings.TrimSpace(*fields); f != "" {
		q.Fields = splitCSV(f)
	}

	api := c.API()
	log.Printf("GET %s (auth=%s user=%s\\%s)", api.GetCUURL(q), *auth, *domain, *user)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	resp, raw, err := api.GetCU(ctx, q)
	if err != nil {
		log.Fatalf("getCU: %v", err)
	}

	units := resp.Units()
	log.Printf("получено КЕ: %d, nextVer=%q, raw=%d байт", len(units), resp.NextVer.String(), len(raw))

	if *out != "" {
		if err := os.WriteFile(*out, raw, 0o644); err != nil {
			log.Fatal(err)
		}
		log.Printf("сырой ответ записан в %s", *out)
		if len(units) > 0 {
			printBrief(units)
		}
		return
	}

	if *pretty {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			fmt.Println(string(raw))
			return
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
		return
	}
	fmt.Println(string(raw))
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func printBrief(units []itilium.CU) {
	n := len(units)
	if n > 5 {
		n = 5
	}
	for i := 0; i < n; i++ {
		u := units[i]
		log.Printf("  [%d] code=%s hostname=%s ip=%s sn=%s", i, u.Code, u.Hostname, u.IP, u.SN)
	}
}
