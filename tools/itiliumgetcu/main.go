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
// Анализ / проверка импорта (update/rack_server.yaml + locality):
//
//	go run ./tools/itiliumgetcu -analyze -all -out getcu.json
//	go run ./tools/itiliumgetcu -analyze -in getcu.json
//
// При -analyze пишутся:
//
//	<stem>-import-check.json      — разбор всей выгрузки
//	<stem>-import-questions.json  — только неоднозначности и вопросы (если есть)
//
// stem берётся из -in / -out или «getcu». Пути можно задать явно:
//
//	-report PATH  -questions PATH
//
// Импорт в БД (app2.json → DB + ConfigItilium):
//
//	go run ./tools/itiliumgetcu -import -all
//	go run ./tools/itiliumgetcu -import -in getcu.json
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
	"path/filepath"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/itilium"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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
	fields := flag.String("fields", "", "fields через запятую (пусто - DefaultFieldsForClass(-cu-class))")
	all := flag.Bool("all", false, "листать страницы по last/hasMore")
	listClasses := flag.Bool("list-classes", false, "напечатать справочник cuClass и выйти")
	listFields := flag.Bool("list-fields", false, "поля КЕ для -cu-class (каталог + supplemental)")
	analyze := flag.Bool("analyze", false, "проверка выгрузки по правилам импорта + отчёты")
	doImport := flag.Bool("import", false, "записать rows в БД из app2.json (Import*FromCU по -cu-class)")
	inFile := flag.String("in", "", "прочитать GetCUResponse JSON из файла (без HTTP)")
	reportPath := flag.String("report", "", "файл полного разбора (по умолчанию <stem>-import-check.json)")
	questionsPath := flag.String("questions", "", "файл вопросов (по умолчанию <stem>-import-questions.json)")
	insecure := flag.Bool("insecure", false, "пропуск проверки TLS")
	timeout := flag.Duration("timeout", 2*time.Minute, "таймаут одного HTTP-запроса")
	out := flag.String("out", "", "файл для сырого/полного JSON ответа getCU")
	pretty := flag.Bool("pretty", true, "pretty-print при выводе в stdout")
	flag.Parse()

	if *listClasses {
		for _, c := range itilium.CUClasses() {
			fmt.Printf("%3d  %s\n", c.Code, c.Name)
		}
		return
	}
	if *listFields {
		cat := itilium.DefaultFieldCatalog()
		fmt.Printf("cuClass=%d %s\n", *cuClass, firstNonEmpty(cat.ClassName(*cuClass), itilium.CUClassName(*cuClass), "?"))
		fmt.Printf("--- каталог (Список полей КЕ):\n")
		for _, f := range cat.Fields(*cuClass) {
			fmt.Printf("  %-12s  %s\n", f.Code, f.Label)
		}
		fmt.Printf("--- fields= для запроса (%d):\n", len(itilium.DefaultFieldsForClass(*cuClass)))
		fmt.Println(strings.Join(itilium.DefaultFieldsForClass(*cuClass), ","))
		return
	}

	// Подтянуть ConfigItilium из app2.json (для -import и при Enabled).
	var itilCfg config.ItiliumConfig
	if _, err := config.ConfigData.LoadConfigData(); err == nil {
		itilCfg = config.ConfigData.ConfigItilium
		applyItiliumConfig(&itilCfg, base, domain, user, password, auth, apiPrefix, insecure, timeout, cuStatus)
	} else if *doImport {
		log.Fatalf("app2.json: %v (нужен для -import / LOADMON_CONFIG)", err)
	}

	if *doImport {
		if err := initDBForImport(); err != nil {
			log.Fatal(err)
		}
		if !*all && *inFile == "" {
			log.Printf("совет: для полной выгрузки укажите -all (или -in getcu.json)")
		}
	}

	var (
		allRows []itilium.CURow
		last    string
		total   int
		pages   int
		rawLast []byte
	)

	if *inFile != "" {
		b, err := os.ReadFile(*inFile)
		if err != nil {
			log.Fatal(err)
		}
		var resp itilium.GetCUResponse
		if err := json.Unmarshal(b, &resp); err != nil {
			log.Fatalf("-in: decode: %v", err)
		}
		allRows = resp.Rows
		total = resp.TotalCount
		last = resp.Last
		rawLast = b
		pages = 1
		log.Printf("прочитано %s: rows=%d totalCount=%d", *inFile, len(allRows), total)
	} else {
		pass := resolveItiliumPassword(*password, *domain, *user, itilCfg)
		if pass == "" {
			log.Fatal("пароль не задан: -password, ITILIUM_PASSWORD, ConfigItilium.Password или GetKeyData")
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
		} else {
			q.Fields = itilium.DefaultFieldsForClass(*cuClass)
		}

		className := itilium.CUClassName(*cuClass)
		if className == "" {
			className = "?"
		}
		api := c.API()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

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
	}

	log.Printf("итого: страниц=%d rows=%d totalCount=%d last=%q", pages, len(allRows), total, last)

	if *doImport {
		runImportCU(allRows, *cuClass)
	}

	if *analyze {
		printRackAnalyze(allRows)
		src := *inFile
		if src == "" {
			src = "getCU"
			if *out != "" {
				src = *out
			}
		}
		rep, qrep := validateDump(allRows, src)
		stem := reportStem(*inFile, *out)
		rPath := strings.TrimSpace(*reportPath)
		if rPath == "" {
			rPath = stem + "-import-check.json"
		}
		if err := writeJSONFile(rPath, rep); err != nil {
			log.Fatal(err)
		}
		log.Printf("проверка импорта: ok=%d issues=%d → %s", rep.OKCount, rep.IssueCount, rPath)
		if len(rep.ByCode) > 0 {
			log.Printf("коды замечаний: %v", rep.ByCode)
		}
		if qrep.ProblemRows > 0 {
			qPath := strings.TrimSpace(*questionsPath)
			if qPath == "" {
				qPath = stem + "-import-questions.json"
			}
			if err := writeJSONFile(qPath, qrep); err != nil {
				log.Fatal(err)
			}
			log.Printf("вопросы по неоднозначностям: rows=%d → %s", qrep.ProblemRows, qPath)
		} else {
			log.Printf("неоднозначностей нет — файл вопросов не создан")
		}
	}

	payload := any(rawLast)
	if *all || *inFile != "" {
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
	if *analyze {
		return
	}
	b, err := marshalOut(payload, *pretty)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}

func reportStem(inPath, outPath string) string {
	for _, p := range []string{inPath, outPath} {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		base := filepath.Base(p)
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}
	return "getcu"
}

func writeJSONFile(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
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

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if t := strings.TrimSpace(s); t != "" {
			return t
		}
	}
	return ""
}

func printBrief(rows []itilium.CURow) {
	n := len(rows)
	if n > 5 {
		n = 5
	}
	for i := 0; i < n; i++ {
		u := rows[i]
		log.Printf("  [%d] code=%s name=%s hostname=%s ip=%s sn=%s links=%s model=%s status=%s",
			i, u.Code.String(), u.Name.String(), u.Hostname.String(), u.IP.String(), u.SN.String(),
			u.LinksUnit.String(), u.ModelVendS.String(), u.Status.Label)
	}
}
