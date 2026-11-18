// Command itiliumgetcu — тестовый вызов `/hs/api/getCU` через loadmon/itilium
// (доменная NTLM-авторизация).
//
// Пример (BCS, серверы):
//
//	go run ./tools/itiliumgetcu ^
//	  -base https://itsd.global.bcs/ITILIUM ^
//	  -domain itildomail -user itiluser -password "%ITILIUM_PASSWORD%" ^
//	  -cu-class 145 -cu-status all ^
//	  -fields code,status,modelServ,hostname,ip,linksUnit,sn,user,where,statusHistory ^
//	  -out getcu.json
//
//	go run ./tools/itiliumgetcu -list-classes
//
// Ответ: { rows:[{ rowKey,uid,deleted,status{label,value},code{…},… }] , … }.
// При -all листает страницы по last/hasMore (если API их отдаёт).
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
	httpKey := "http_proxy"
	httpsKey := "https_proxy"
	httpEnv := os.Getenv(httpKey)
	defer os.Setenv(httpKey, httpEnv)

	httpsEnv := os.Getenv(httpsKey)
	defer os.Setenv(httpsKey, httpsEnv)

	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	s1 := os.Getenv(httpKey)
	s2 := os.Getenv(httpKey)

	_ = s1
	_ = s2

	base := flag.String("base", "https://itsd.global.bcs/ITILIUM", "корень публикации 1С")
	domain := flag.String("domain", "itildomail", "AD-домен")
	user := flag.String("user", "itiluser", "логин без домена")
	password := flag.String("password", "", "пароль (иначе ITILIUM_PASSWORD / GetKeyData)")
	auth := flag.String("auth", itilium.AuthNTLM, "ntlm или basic")
	apiPrefix := flag.String("api-prefix", itilium.DefaultAPIPrefix, "префикс HTTP-сервиса")
	cuClass := flag.Int("cu-class", itilium.CUClassServers, "cuClass (145=серверы)")
	cuStatus := flag.String("cu-status", itilium.CUStatusAll, "cuStatus (all - все)")
	nextVer := flag.String("next-ver", "", "nextVer (пусто - не передавать)")
	limit := flag.Int("limit", 0, "limit (0 - не передавать)")
	fields := flag.String("fields", strings.Join(itilium.DefaultServerCUFields, ","), "fields через запятую")
	all := flag.Bool("all", false, "листать страницы по last/hasMore")
	listClasses := flag.Bool("list-classes", false, "напечатать справочник cuClass и выйти")
	insecure := flag.Bool("insecure", false, "пропуск проверки TLS")
	timeout := flag.Duration("timeout", 2*time.Minute, "таймаут одного HTTP-запроса")
	out := flag.String("out", "", "файл для JSON")
	pretty := flag.Bool("pretty", true, "pretty-print при выводе в stdout")
	flag.Parse()

	if *listClasses {
		for _, c := range itilium.CUClasses() {
			fmt.Printf("%3d  %s\n", c.Code, c.Name)
		}
		return
	}

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
		Name: "getcu", BaseURL: *base, Login: *user, Password: pass,
		Domain: *domain, AuthMode: *auth, APIPrefix: *apiPrefix,
		Timeout: *timeout, InsecureSkipVerify: *insecure,
	})
	if err != nil {
		log.Fatal(err)
	}

	q := itilium.GetCUQuery{
		NextVer: *nextVer, CUClass: *cuClass, CUStatus: *cuStatus, Limit: *limit,
	}
	if f := strings.TrimSpace(*fields); f != "" {
		q.Fields = splitCSV(f)
	}

	className := itilium.CUClassName(*cuClass)
	if className == "" {
		className = "?"
	}
	api := c.API()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var (
		allRows []itilium.CURow
		last    string
		total   int
		pages   int
		rawLast []byte
	)

	for {
		pages++
		log.Printf("GET %s (class=%d %s, auth=%s %s\\%s)", api.GetCUURL(q), *cuClass, className, *auth, *domain, *user)
		resp, raw, err := api.GetCURaw(ctx, q)
		if err != nil {
			log.Fatalf("getCU: %v", err)
		}
		rawLast = raw
		total = resp.TotalCount
		last = resp.Last
		allRows = append(allRows, resp.Rows...)
		log.Printf("страница %d: rows=%d totalCount=%d last=%q hasMore=%v", pages, len(resp.Rows), resp.TotalCount, resp.Last, resp.HasMore)
		printBrief(resp.Rows)

		if !*all {
			break
		}
		next, ok := resp.NextPageQuery(q)
		if !ok {
			break
		}
		q = next
	}

	log.Printf("итого: страниц=%d rows=%d totalCount=%d last=%q", pages, len(allRows), total, last)

	payload := any(rawLast)
	if *all {
		payload = itilium.GetCUResponse{Rows: allRows, TotalCount: total, Last: last, HasMore: false}
	}

	if *out != "" {
		b, err := marshalOut(payload, true)
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			log.Fatal(err)
		}
		log.Printf("записано в %s (%d байт)", *out, len(b))
		return
	}
	b, err := marshalOut(payload, *pretty)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

func marshalOut(v any, pretty bool) ([]byte, error) {
	if b, ok := v.([]byte); ok {
		if !pretty {
			return b, nil
		}
		var anyV any
		if err := json.Unmarshal(b, &anyV); err != nil {
			return b, nil
		}
		return json.MarshalIndent(anyV, "", "  ")
	}
	if pretty {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
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

func printBrief(rows []itilium.CURow) {
	n := len(rows)
	if n > 5 {
		n = 5
	}
	for i := 0; i < n; i++ {
		u := rows[i]
		log.Printf("  [%d] code=%s name=%s hostname=%s ip=%s sn=%s links=%s status=%s",
			i, u.Code.String(), u.Name.String(), u.Hostname.String(), u.IP.String(), u.SN.String(), u.LinksUnit.String(), u.Status.Label)
	}
}
