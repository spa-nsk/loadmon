// Command oneviewpowerprobe — диагностика потребляемой мощности HPE OneView.
//
// Два источника:
//
//  1. /utilization — задокументированные метрики OneView (AveragePower/
//     PeakPower/PowerCap). Работают ТОЛЬКО с лицензией OneView Advanced;
//     при Standard отвечают 404 (старые API, напр. 300) или 200 с нулями.
//  2. iLO через SSO (режим -ilo) — OneView Standard умеет выдавать SSO-токен
//     на iLO каждого сервера (как командлет Get-OVIloSso из HPEOneView.660):
//     GET /rest/server-hardware/{id}/iloSsoUrl -> iloSsoUrl вида
//     https://ilo:443/Proxy/SSO?TKN=...&HA=SHA1&KEY=ssononce ...&XE=<appliance>&UN=<user>&URL=...
//     (это НЕ готовый X-Auth-Token, а URL SSO-входа; токен сессии извлекается
//     из ответа iLO - в разработке, probe показывает сырой SSO-шаг).
//
// Подтверждённые на реальных appliance ответы:
//   - /utilization: API 300 -> 404; API 3800/7400 -> 200 с нулями (нет лицензии Advanced);
//   - iloSsoUrl: API 3800/7400 -> 200 (формат /Proxy/SSO выше);
//     Global Dashboard (API 300) -> 404 [GET_ILO_SSO_FAIL] - SSO брать с
//     ЛОКАЛЬНОГО appliance, где сервер состоит, а не с GD.
//
// В БД не пишет. Конфиг — как у приложения (LOADMON_CONFIG или conf/app2.json).
//
// Примеры:
//
//	go run ./tools/oneviewpowerprobe -endpoint https://oneview.example -list-enclosures
//	go run ./tools/oneviewpowerprobe -server srv-01
//	go run ./tools/oneviewpowerprobe -server srv-01 -variants
//	go run ./tools/oneviewpowerprobe -server srv-01 -ilo
//	go run ./tools/oneviewpowerprobe -server srv-01 -ilo -raw
//	go run ./tools/oneviewpowerprobe -all-servers
//	go run ./tools/oneviewpowerprobe -enclosure "c7000-01"
//	go run ./tools/oneviewpowerprobe -list-ilo                      (все ConfigOV, в т.ч. Global Dashboard)
//	go run ./tools/oneviewpowerprobe -list-ilo -csv -out ilo_ips.csv
//	go run ./tools/oneviewpowerprobe -list-ilo -list-ilo-detail -endpoint https://gd.example
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
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
	encFlag := flag.String("enclosure", "", "имя или URI/id корзины: GET /rest/enclosures/{id} + utilization")
	serverURI := flag.String("server-uri", "", "URI server-hardware для /utilization или -ilo")
	serverName := flag.String("server", "", "имя server-hardware (поиск в списке) для /utilization или -ilo")
	allServers := flag.Bool("all-servers", false, "пробежать /utilization по всем server-hardware и свести таблицей")
	variants := flag.Bool("variants", false, "server-режим: обойти варианты URL /utilization (поля/view) и показать каждый")
	iloFlag := flag.Bool("ilo", false, "server-режим: мощность через iLO SSO-токен (OneView Standard) вместо /utilization")
	fieldsFlag := flag.String("fields", "AveragePower,PeakPower,PowerCap", "список fields для /utilization (через запятую; пусто — без параметра fields)")
	viewFlag := flag.String("view", "day", "view для /utilization (day/hour/...; пусто — без параметра)")
	envFlag := flag.Bool("env", false, "дополнительно показать environmentalConfiguration (старый эвристический источник)")
	rawOnly := flag.Bool("raw", false, "только сырой JSON без summary")
	diagBlade := flag.String("diag-blade", "", "имя корзины: сравнение enclosure power vs sum(блейдов) (нужна БД loadmon)")
	out := flag.String("out", "", "файл для вывода (пусто — stdout)")
	listIlo := flag.Bool("list-ilo", false, "выгрузить IP iLO всех серверов (все ConfigOV, либо один по -endpoint) для заявки на сетевой доступ")
	iloDetail := flag.Bool("list-ilo-detail", false, "для -list-ilo: если в списке нет адреса, добрать GET'ом по каждому серверу (медленнее)")
	iloCSV := flag.Bool("csv", false, "для -list-ilo: вывод в CSV (разделитель ;)")
	flag.Parse()

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("чтение конфигурации: %v", err)
	}
	if len(config.ConfigData.ConfigOV) == 0 {
		log.Fatal("в конфиге нет ConfigOV")
	}
	if *listIlo {
		text := strings.Join(runListIlo(*endpointFlag, *iloDetail, *iloCSV), "\n")
		if *out != "" {
			if err := os.WriteFile(*out, []byte(text+"\n"), 0o644); err != nil {
				log.Fatal(err)
			}
			return
		}
		fmt.Println(text)
		return
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

	fields := strings.ReplaceAll(strings.TrimSpace(*fieldsFlag), " ", "")
	view := strings.TrimSpace(*viewFlag)

	var lines []string
	// 2026-10-08: у Global Dashboard набор ресурсов урезан (нет стоек/ЦОД и,
	// возможно, подресурсов server-hardware) - показываем, с чем говорим.
	lines = append(lines, fmt.Sprintf("endpoint=%s product=%s apiVersion=%d globalDashboard=%v",
		c.BaseURL(), c.Product().String(), c.APIVersion(), c.IsGlobalDashboard()))
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
			lines = append(lines, fmt.Sprintf("parsed enclosure power (поля ресурса): avg=%.0f peak=%.0f rated=%.0f source=%s",
				p.Avg24hW, p.Peak24hW, p.RatedMaxW, p.Source))
		}
		// новый источник: utilization самой корзины (AveragePower/PeakPower по статье HPE)
		lines = append(lines, "--- utilization корзины "+enc.URI+" ---")
		lines = append(lines, analyzeUtilization(ctx, c, enc.URI, fields, view, *rawOnly)...)
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
	case *allServers:
		col, err := c.ListServerHardware(ctx, oneview.ListOptions{Count: -1})
		if err != nil {
			log.Fatal(err)
		}
		lines = append(lines, fmt.Sprintf("server-hardware: %d шт., fields=%q view=%q", len(col.Members), fields, view))
		var withData, zero, errs int
		for i := range col.Members {
			hw := col.Members[i]
			line := summarizeServerUtilizationLine(ctx, c, hw.Name, hw.URI, hw.PowerState, fields, view)
			lines = append(lines, line)
			switch {
			case strings.Contains(line, "err="):
				errs++
			case strings.Contains(line, "ZERO"):
				zero++
			default:
				withData++
			}
		}
		lines = append(lines, fmt.Sprintf("итого: %d всего, %d с данными (avg>0), %d с нулями при power=On, %d с ошибкой",
			len(col.Members), withData, zero, errs))
	case *serverURI != "" || *serverName != "":
		uri := *serverURI
		if uri == "" {
			uri = findServerURI(ctx, c, *serverName)
		}
		switch {
		case *iloFlag:
			lines = append(lines, probeIloPower(ctx, c, uri, *rawOnly)...)
		case *variants:
			lines = append(lines, "--- обход вариантов URL /utilization (сервер "+uri+") ---")
			for _, v := range utilizationVariants(fields, view) {
				lines = append(lines, "вариант fields="+v.fields+" view="+v.view+":")
				for _, l := range analyzeUtilization(ctx, c, uri, v.fields, v.view, *rawOnly) {
					lines = append(lines, "  "+l)
				}
			}
		default:
			lines = append(lines, "--- utilization сервера "+uri+" ---")
			lines = append(lines, analyzeUtilization(ctx, c, uri, fields, view, *rawOnly)...)
		}
		if *envFlag {
			lines = append(lines, probeServerEnvironment(ctx, c, uri, *rawOnly)...)
		}
	default:
		flag.Usage()
		log.Fatal("укажите -list-enclosures, -enclosure, -server-uri, -server, -all-servers или -diag-blade")
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

// utilizationResponse - верхний уровень ответа GET .../utilization.
// Числа/булевы читаем через RawMessage/Number - HPE варьирует типы между
// версиями API ("true" vs true, число vs строка), не гадаем.
type utilizationResponse struct {
	IsFresh               json.RawMessage     `json:"isFresh"`
	RefreshState          string              `json:"refreshState"`
	SampleInterval        json.Number         `json:"sampleInterval"`
	NumberOfSamples       json.Number         `json:"numberOfSamples"`
	SampleStartTime       string              `json:"sampleStartTime"`
	SampleEndTime         string              `json:"sampleEndTime"`
	OldestSampleTimestamp string              `json:"oldestSampleTimestamp"`
	MetricList            []utilizationMetric `json:"metricList"`
}

type utilizationMetric struct {
	MetricName  string                       `json:"metricName"`
	MetricUnits string                       `json:"metricUnits"`
	DataList    []map[string]json.RawMessage `json:"dataList"`
}

// metricSummary - сводка по одному показателю из dataList.
type metricSummary struct {
	Name     string
	Units    string
	Samples  int // сэмплов с разобранными значениями
	DataLen  int // всего элементов в dataList
	Skipped  int // элементы, у которых avg/max/min не разобрались
	Avg      float64
	Max      float64
	Min      float64
	RawFirst []string // первые сырые сэмплы (только когда Max == 0)
	Ok       bool
}

func buildUtilizationURL(uri, fields, view string) string {
	u := uri + "/utilization"
	sep := "?"
	if fields != "" {
		u += sep + "fields=" + fields
		sep = "&"
	}
	if view != "" {
		u += sep + "view=" + view
	}
	return u
}

type urlVariant struct{ fields, view string }

// utilizationVariants - комбинации параметров для -variants: где-то данные
// могут прятаться (другой view, другой набор полей, без параметров вообще).
func utilizationVariants(fields, view string) []urlVariant {
	out := []urlVariant{
		{fields, view},
		{fields, ""}, // без view
		{fields, "hour"},
		{"", ""}, // вообще без параметров - все метрики, дефолтный view
	}
	if fields != "AveragePower" {
		out = append(out, urlVariant{"AveragePower", view})
	}
	return out
}

// analyzeUtilization запрашивает /utilization, печатает заголовок ответа
// (свежесть/окно/шаг) и сводку по каждой запрошенной метрике; при raw=true
// дополнительно отдаёт сырой JSON. Когда все значения метрики нулевые,
// показывает первые сырые сэмплы - чтобы отличить "реальные нули" от
// "значения не распарсились/пришли пустыми".
func analyzeUtilization(ctx context.Context, c *oneview.Client, uri, fields, view string, raw bool) []string {
	url := buildUtilizationURL(uri, fields, view)
	status, body, err := doGET(ctx, c, url)
	if err != nil || status != 200 {
		return []string{fmt.Sprintf("GET %s -> HTTP %d, ошибка: %v\n%s", url, status, err, truncate(body))}
	}
	if raw {
		var pretty json.RawMessage = body
		b, mErr := json.MarshalIndent(pretty, "", "  ")
		if mErr != nil {
			return []string{string(body)}
		}
		return []string{string(b)}
	}
	resp, pErr := parseUtilization(body)
	if pErr != nil {
		return []string{fmt.Sprintf("GET %s -> HTTP 200, но разбор не удался: %v\n%s", url, pErr, truncate(body))}
	}
	lines := []string{
		fmt.Sprintf("GET %s -> HTTP 200", url),
		fmt.Sprintf("fresh=%s refreshState=%s samples=%s interval=%ss window=%s..%s oldest=%s",
			rawOrDash(resp.IsFresh), resp.RefreshState, resp.NumberOfSamples.String(), resp.SampleInterval.String(),
			resp.SampleStartTime, resp.SampleEndTime, resp.OldestSampleTimestamp),
	}
	sums := summarizeMetrics(resp)
	if len(sums) == 0 {
		lines = append(lines, "(metricList пуст - метрик по запрошенным fields нет; попробуйте -variants или -raw)")
		return lines
	}
	for _, s := range sums {
		lines = append(lines, fmt.Sprintf("  %-20s samples=%d/%d (skipped=%d) avg=%10.1f  max=%10.1f  min=%10.1f  [%s]",
			s.Name, s.Samples, s.DataLen, s.Skipped, s.Avg, s.Max, s.Min, s.Units))
		if s.Max == 0 && len(s.RawFirst) > 0 {
			lines = append(lines, "    все значения 0, сырые первые сэмплы:")
			for _, r := range s.RawFirst {
				lines = append(lines, "      "+r)
			}
		}
	}
	return lines
}

// summarizeServerUtilizationLine - одна строка на сервер для -all-servers:
// имя, питание, avg(AveragePower), max(PeakPower), max(PowerCap). Серверам
// с power=On и avg=0 помечает ZERO - так видно масштаб проблемы.
func summarizeServerUtilizationLine(ctx context.Context, c *oneview.Client, name, uri, powerState, fields, view string) string {
	url := buildUtilizationURL(uri, fields, view)
	status, body, err := doGET(ctx, c, url)
	if err != nil || status != 200 {
		return fmt.Sprintf("%-40s power=%-8s HTTP %d err=%v", name, powerState, status, err)
	}
	resp, pErr := parseUtilization(body)
	if pErr != nil {
		return fmt.Sprintf("%-40s power=%-8s HTTP 200, разбор: %v", name, powerState, pErr)
	}
	sums := summarizeMetrics(resp)
	byName := make(map[string]metricSummary, len(sums))
	for _, s := range sums {
		byName[s.Name] = s
	}
	avg := byName["AveragePower"]
	peak := byName["PeakPower"]
	cap_ := byName["PowerCap"]
	mark := ""
	if strings.EqualFold(strings.TrimSpace(powerState), "on") && avg.Avg == 0 {
		mark = " ZERO"
	}
	return fmt.Sprintf("%-40s power=%-8s avg=%7.1f peak=%7.1f cap=%7.1f samples=%d/%s fresh=%s%s",
		name, powerState, avg.Avg, peak.Max, cap_.Max, avg.Samples, resp.NumberOfSamples.String(), rawOrDash(resp.IsFresh), mark)
}

// probeIloPower - мощность сервера напрямую с iLO по SSO-токену OneView
// (путь командлета Get-OVIloSso): iloSsoUrl содержит token, к iLO ходим
// с заголовком X-Auth-Token. Все сырые ответы печатаются - схемы Redfish/OEM
// у iLO 4/5 и разных лицензий отличаются, не гадаем.
func probeIloPower(ctx context.Context, c *oneview.Client, uri string, raw bool) []string {
	var lines []string
	var hw oneview.ServerHardware
	if err := c.GetJSON(ctx, uri, &hw); err != nil {
		return []string{fmt.Sprintf("iLO: GET %s (server-hardware) не удался: %v", uri, err)}
	}
	lines = append(lines, fmt.Sprintf("--- iLO мощность через SSO (сервер %s) ---", hw.Name))
	lines = append(lines, fmt.Sprintf("mpIpAddress=%s mpHostInfo=%+v mpModel=%s mpState=%s mpLicenseType=%s",
		hw.MpIpAddress, hw.MpHostInfo, hw.MpModel, hw.MpState, hw.MpLicenseType))

	// адрес iLO: поле mpIpAddress у части серверов (напр. на API 7400) пустое -
	// тогда адрес лежит в mpHostInfo.mpIpAddresses, запасной вариант - mpDnsName.
	host := iloHost(&hw)
	if host == "" {
		return append(lines, "адрес iLO неизвестен (mpIpAddress/mpHostInfo/mpDnsName пусты), SSO-токен не применить")
	}

	// SSO-токен: штатный метод SDK - GET /rest/server-hardware/{id}/iloSsoUrl
	// (эндпоинт из swagger 300; GET /iloSso и /sso на API 3800/7400 не существуют -
	// проверено probe'ом, отвечают GENERIC_HTTP_404).
	sso, err := c.GetServerHardwareILOSSO(ctx, uri)
	if err != nil {
		return append(lines, fmt.Sprintf("GetServerHardwareILOSSO (%s/iloSsoUrl): %v", uri, err))
	}
	if raw {
		if b, mErr := json.MarshalIndent(sso, "", "  "); mErr == nil {
			lines = append(lines, "сырой ответ iloSsoUrl:", string(b))
		}
	}
	ssoURL := strings.TrimSpace(sso.IloSsoUrl)
	if ssoURL == "" {
		ssoURL = strings.TrimSpace(sso.SSOUrl)
	}
	if ssoURL == "" {
		return append(lines, "iloSsoUrl/ssoUrl пусты в ответе")
	}
	u, perr := url.Parse(ssoURL)
	if perr != nil {
		return append(lines, fmt.Sprintf("iloSsoUrl не парсится: %v", perr))
	}
	var keys []string
	for k := range u.Query() {
		keys = append(keys, k)
	}
	lines = append(lines, fmt.Sprintf("iloSsoUrl: scheme=%s host=%s path=%s params=[%s]",
		u.Scheme, u.Host, u.Path, strings.Join(keys, ",")))

	// iLO отдаёт самоподписанные сертификаты - TLS не проверяем (только диагностика).
	ilo := &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	// базовый адрес iLO: host из iloSsoUrl (авторитетно), запасной - инвентарный.
	baseHost := u.Host
	if baseHost == "" {
		baseHost = host
	}
	base := "https://" + baseHost

	// 1) связность/TLS до iLO без авторизации - отдельно от SSO, чтобы
	//    отличить "сеть/файрвол" от "авторизация".
	lines = append(lines, "--- связность с iLO (GET "+base+"/, без авторизации) ---")
	lines = append(lines, iloPing(ctx, ilo, base))

	// 2) SSO-вход: прямой GET iloSsoUrl БЕЗ следования редиректам - токен
	//    сессии почти наверняка приезжает в Location/Set-Cookie/теле ответа.
	lines = append(lines, "--- SSO-вход: GET /Proxy/SSO?... (redirect не следуем) ---")
	lines = append(lines, iloSSOStep(ctx, ilo, ssoURL)...)
	lines = append(lines, "(по результатам этого шага определим, откуда брать X-Auth-Token)")
	return lines
}

// iloPing проверяет базовую связность с iLO: TCP+TLS+ответ на GET /.
func iloPing(ctx context.Context, client *http.Client, base string) string {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/", nil)
	if err != nil {
		return fmt.Sprintf("GET %s/ -> ошибка запроса: %v", base, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("GET %s/ -> ошибка (сеть/TLS?): %v", base, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Sprintf("GET %s/ -> HTTP %d, %d байт тела: %s", base, resp.StatusCode, len(body), truncate(body))
}

// iloSSOStep выполняет GET на iloSsoUrl без следования редиректам и печатает
// статус, заголовки Location/Set-Cookie и начало тела - чтобы фактически
// увидеть механизм выдачи токена на данной версии iLO.
func iloSSOStep(ctx context.Context, client *http.Client, ssoURL string) []string {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cl := *client
	cl.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ssoURL, nil)
	if err != nil {
		return []string{fmt.Sprintf("GET iloSsoUrl -> ошибка запроса: %v", err)}
	}
	req.Header.Set("Accept", "text/html,application/json,*/*")
	resp, err := cl.Do(req)
	if err != nil {
		return []string{fmt.Sprintf("GET iloSsoUrl -> ошибка: %v", err)}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return []string{
		fmt.Sprintf("HTTP %d %s", resp.StatusCode, resp.Status),
		"Location: " + resp.Header.Get("Location"),
		"Set-Cookie: " + truncateValue(resp.Header.Get("Set-Cookie")),
		"Content-Type: " + resp.Header.Get("Content-Type"),
		"тело (усеч.):\n" + truncate(body),
	}
}

// truncateValue обрезает длинный заголовок (Set-Cookie/Location) для вывода.
func truncateValue(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

// iloHost определяет адрес iLO сервера: mpIpAddress, затем список из
// mpHostInfo.mpIpAddresses, затем mpDnsName.
func iloHost(hw *oneview.ServerHardware) string {
	if hw == nil {
		return ""
	}
	if v := strings.TrimSpace(hw.MpIpAddress); v != "" {
		return v
	}
	for _, a := range hw.MpHostInfo.MpIpAddresses {
		if v := strings.TrimSpace(a.Address); v != "" {
			return v
		}
	}
	return strings.TrimSpace(hw.MpDnsName)
}

// iloGET выполняет GET к iLO с SSO-токеном и возвращает статус + сводку
// (для /Power/ - PowerControl) либо усечённое сырое тело; второе значение -
// признак HTTP 200 (используется при выборе рабочего варианта токена).
func iloGET(ctx context.Context, client *http.Client, base, path, token string, raw bool) (string, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return fmt.Sprintf("GET %s -> ошибка запроса: %v", path, err), false
	}
	req.Header.Set("X-Auth-Token", token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("GET %s -> ошибка: %v", path, err), false
	}
	defer resp.Body.Close()
	body, rerr := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if rerr != nil {
		return fmt.Sprintf("GET %s -> HTTP %d, чтение тела: %v", path, resp.StatusCode, rerr), false
	}
	if resp.StatusCode != http.StatusOK || raw {
		return fmt.Sprintf("GET %s -> HTTP %d\n%s", path, resp.StatusCode, truncate(body)), resp.StatusCode == http.StatusOK
	}
	if strings.HasSuffix(path, "/Power/") {
		var redfish struct {
			PowerControl []struct {
				PowerConsumedWatts float64 `json:"PowerConsumedWatts"`
				PowerMetrics       struct {
					AverageConsumedWatts float64 `json:"AverageConsumedWatts"`
					MaxConsumedWatts     float64 `json:"MaxConsumedWatts"`
					MinConsumedWatts     float64 `json:"MinConsumedWatts"`
				} `json:"PowerMetrics"`
			} `json:"PowerControl"`
		}
		if err := json.Unmarshal(body, &redfish); err == nil && len(redfish.PowerControl) > 0 {
			pc := redfish.PowerControl[0]
			return fmt.Sprintf("GET %s -> HTTP 200: PowerConsumedWatts=%.0f avg=%.0f max=%.0f min=%.0f",
				path, pc.PowerConsumedWatts, pc.PowerMetrics.AverageConsumedWatts, pc.PowerMetrics.MaxConsumedWatts, pc.PowerMetrics.MinConsumedWatts), true
		}
	}
	if strings.HasSuffix(path, "/Power") {
		var oem struct {
			PowerControl []struct {
				PowerConsumedWatts float64 `json:"PowerConsumedWatts"`
			} `json:"PowerControl"`
		}
		if err := json.Unmarshal(body, &oem); err == nil && len(oem.PowerControl) > 0 {
			return fmt.Sprintf("GET %s -> HTTP 200: PowerConsumedWatts=%.0f", path, oem.PowerControl[0].PowerConsumedWatts), true
		}
	}
	return fmt.Sprintf("GET %s -> HTTP 200\n%s", path, truncate(body)), true
}

// probeServerEnvironment - старый путь (export + environmentalConfiguration),
// оставлен под флагом -env для сравнения с /utilization.
func probeServerEnvironment(ctx context.Context, c *oneview.Client, uri string, rawOnly bool) []string {
	var lines []string
	exp, err := c.ExportServer(ctx, uri, oneview.ExportOptions{
		SkipFirmware:     true,
		SkipLocalStorage: true,
		SkipBIOS:         true,
		SkipProfile:      true,
		SkipHardwareType: true,
		SkipEnclosure:    true,
	})
	if err != nil {
		return []string{fmt.Sprintf("environmentalConfiguration: export не удался: %v", err)}
	}
	p := oneview.DeriveServerPower(exp)
	lines = append(lines, "--- environmentalConfiguration (legacy) ---")
	lines = append(lines, fmt.Sprintf("server %s powerState=%s parsed: avg=%.0f peak=%.0f rated=%.0f source=%s",
		exp.Identity.Name, exp.Identity.PowerState, p.Avg24hW, p.Peak24hW, p.RatedMaxW, p.Source))
	if len(exp.Environment) > 0 {
		if rawOnly {
			lines = append(lines, string(exp.Environment))
		} else {
			var pretty json.RawMessage = exp.Environment
			b, _ := json.MarshalIndent(pretty, "", "  ")
			lines = append(lines, string(b))
		}
	} else {
		lines = append(lines, "(environmentalConfiguration пуст)")
	}
	if len(exp.Warnings) > 0 {
		lines = append(lines, "предупреждения экспорта:")
		for _, w := range exp.Warnings {
			lines = append(lines, "  - "+w)
		}
	}
	return lines
}

func parseUtilization(body []byte) (*utilizationResponse, error) {
	var resp utilizationResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// summarizeMetrics считает avg/max/min по dataList каждой метрики.
// Значения в dataList у HPE бывают и числами, и строками ("698,"), а при
// отсутствии данных - "NaN"/"" (такие сэмплы считаются skipped и при
// нулевом max показываются сырыми).
func summarizeMetrics(resp *utilizationResponse) []metricSummary {
	if resp == nil {
		return nil
	}
	out := make([]metricSummary, 0, len(resp.MetricList))
	for _, m := range resp.MetricList {
		s := metricSummary{Name: m.MetricName, Units: m.MetricUnits, DataLen: len(m.DataList)}
		var avgSum float64
		var avgN int
		first := true
		zeroMax := false
		for i, sample := range m.DataList {
			avg, hasAvg := flexNum(sample["avg"])
			mx, hasMax := flexNum(sample["max"])
			mn, hasMin := flexNum(sample["min"])
			if !hasAvg && !hasMax && !hasMin {
				s.Skipped++
				if len(s.RawFirst) < 3 {
					s.RawFirst = append(s.RawFirst, rawSample(sample))
				}
				continue
			}
			if !hasMax {
				mx = avg
			}
			if !hasMin {
				mn = avg
			}
			if hasAvg {
				avgSum += avg
				avgN++
			}
			if first || mx > s.Max {
				s.Max = mx
			}
			if first || mn < s.Min {
				s.Min = mn
			}
			if i < 3 && mx == 0 {
				zeroMax = true
			}
			first = false
			s.Samples++
		}
		if s.Samples == 0 {
			out = append(out, s)
			continue
		}
		if avgN > 0 {
			s.Avg = avgSum / float64(avgN)
		}
		s.Ok = true
		if s.Max == 0 && (zeroMax || s.Skipped > 0) && len(s.RawFirst) == 0 {
			for i := 0; i < len(m.DataList) && len(s.RawFirst) < 3; i++ {
				s.RawFirst = append(s.RawFirst, rawSample(m.DataList[i]))
			}
		}
		out = append(out, s)
	}
	return out
}

func rawSample(sample map[string]json.RawMessage) string {
	b, err := json.Marshal(sample)
	if err != nil {
		return fmt.Sprintf("%v", sample)
	}
	if len(b) > 300 {
		b = append(b[:300], []byte("...")...)
	}
	return string(b)
}

// flexNum разбирает число из JSON-числа или строки ("105.0", "698,").
// "NaN"/"Inf" НЕ считаются числом - это признак "данных нет", а не нуля.
func flexNum(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err == nil {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return v, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, false
	}
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), ","))
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

func rawOrDash(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "-"
	}
	return strings.Trim(string(raw), `"`)
}

func truncate(body []byte) string {
	const max = 4000
	if len(body) > max {
		return string(body[:max]) + "...(усечено)"
	}
	return string(body)
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

// doGET выполняет GET path и возвращает статус и тело ответа.
func doGET(ctx context.Context, c *oneview.Client, path string) (int, []byte, error) {
	resp, err := c.Do(ctx, "GET", path, nil)
	if resp == nil {
		return 0, nil, err
	}
	return resp.StatusCode, resp.Body, err
}
