// Тест-проба потребляемой мощности HPE OneView на ЖИВЫХ appliance (API 300 /
// 3800 / 8800 и др.). Без параметров командной строки: все настройки - в блоке
// "ПАРАМЕТРЫ ТЕСТА" ниже. Запускать из VS Code: "debug test" над
// TestOneViewPower (или F5 с конфигурацией "Go: Test Function").
//
// Что делает:
//  1. Загружает conf/app2.json (как приложение) и берёт все ConfigOV
//     (или только отфильтрованные onlyEndpoints).
//  2. Для каждого appliance: Login, GET /rest/version, постраничный список
//     server-hardware.
//  3. Для каждого сервера (в пределах maxServers) проверяет по справочникам
//     HPE (kontext/14-hpe-oneview-rest-api.md):
//     - GET {srv}/utilization (AveragePower/PeakPower/PowerCap/... за окно
//     windowHours) - измеренная мощность;
//     - GET {srv}/environmentalConfiguration - licenseRequirement,
//     powerHistorySupported, calibratedMaxPower (паспорт, не измерение);
//     - GET {srv}/powerSupplies - "живая" мощность блоков питания;
//     - GET {enclosure}/utilization для блейдов (locationUri) - есть ли
//     шасси на данной версии API (в справочнике 8800 разделов шасси нет).
//  4. Печатает построчный результат и итоговую сводку, в т.ч. разрез по
//     лицензиям (licensingIntent / mpLicenseType / licenseRequirement) -
//     у каких комбинаций реально приходят данные.
//
// Тест ничего не пишет в БД и не меняет OneView (только GET + Login/Logout).
// Он диагностический: отсутствие лицензии/данных НЕ считается ошибкой теста,
// ошибкой считается только невозможность войти/получить список серверов.
package oneviewpowertest

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"loadmon/config"
	"loadmon/utilities"

	oneview "github.com/spa-nsk/oneview2"
)

// ============================================================================
// ПАРАМЕТРЫ ТЕСТА - меняйте здесь, затем запускайте тест в отладчике.
// ============================================================================
var (
	// configPath - путь к app2.json. Пусто: LOADMON_CONFIG, иначе
	// <корень репозитория>/conf/app2.json.
	configPath = ""

	// onlyEndpoints - подстроки Endpoint из ConfigOV (без учёта регистра);
	// пусто - проходить ВСЕ ConfigOV. Пример: []string{"ov-8800", "10.1.2.3"}.
	onlyEndpoints = []string{}

	// onlyAPIVersions - ограничение по ConfigOV.Version (3800, 8800, 300...);
	// пусто - любые.
	onlyAPIVersions = []int{}

	// passwordOverride - пароль для ВСЕХ appliance (отладка). Пусто: значение
	// из ConfigOV; "password" заменяется паролем из хранилища ключей
	// (как в config.InitOVParams); переменная LOADMON_OV_PASSWORD - тоже
	// переопределяет.
	passwordOverride = ""

	// ---- какие серверы проверять ----

	maxServersPerEndpoint = 5  // 0 - все серверы appliance
	serverNameContains    = "" // подстрока имени server-hardware ("" - любые)
	onlyPoweredOn         = true
	onlyBlades            = false // true - только серверы с locationUri (в шасси)
	pageSize              = 100   // размер страницы списка server-hardware

	// ---- параметры запроса /utilization ----

	windowHours = 24 // окно: [now-windowHours; now] (UTC)
	// utilView: "native" (5 мин) | "hour" | "day" | "" (без параметра view).
	utilView = "hour"
	// utilFields - метрики через запятую; "" - без fields (все поддерживаемые).
	utilFields = "AveragePower,PeakPower,PowerCap,CpuUtilization,AmbientTemperature"
	// probeAllMetricsOnce - для первого проверяемого сервера каждого appliance
	// дополнительно запросить /utilization БЕЗ fields и показать реальный набор
	// metricName.
	probeAllMetricsOnce = true

	// ---- дополнительные проверки ----

	checkEnvConfig     = true // GET {srv}/environmentalConfiguration
	checkPowerSupplies = true // GET {srv}/powerSupplies
	checkEnclosure     = true // GET {enclosure}/utilization (+ environmentalConfiguration)

	// ---- вывод ----

	printRawJSON = false // печатать сырые ответы (обрезаются rawMaxChars)
	rawMaxChars  = 2000
	// reportCSV - файл CSV со всеми результатами (";"); пусто - не писать.
	reportCSV = ""

	// ---- таймауты ----

	requestTimeout = 90 * time.Second // таймаут одного HTTP-запроса (OperationTimeout)
	totalTimeout   = 60 * time.Minute // общий таймаут теста
	// pauseBetweenServers - пауза между серверами (щадим iLO/appliance).
	pauseBetweenServers = 0 * time.Millisecond
)

// ============================================================================

// serverResult - итог проверки одного сервера (строка отчёта).
type serverResult struct {
	Endpoint, API, Name, URI, Model, Generation, PowerState string
	LicensingIntent, MpLicenseType, MpModel, MpState        string
	LicenseRequirement                                      string // из environmentalConfiguration
	PowerHistorySupported                                   string
	CalibratedMaxW                                          float64
	UtilHTTP                                                int
	UtilClass                                               string // OK / ZEROS / NO_SAMPLES / NOT_FOUND / HTTP_ERR / ERR
	Metrics                                                 map[string]metricStat
	PSUHTTP                                                 int
	PSUCount                                                int
	PSUSumW                                                 float64
	EnclosureHTTP                                           int
	EnclosureClass                                          string
	Note                                                    string
}

// metricStat - сводка по одному metricName из UtilizationData.
type metricStat struct {
	Samples, NonNull int
	Last, Avg, Max   float64 // Last - самый свежий не-null (сэмплы идут от новых к старым)
	Capacity         string
}

func TestOneViewPower(t *testing.T) {
	// Корень репозитория определяем по расположению этого файла.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	cfgFile := configPath
	if cfgFile == "" {
		cfgFile = os.Getenv("LOADMON_CONFIG")
	}
	if cfgFile == "" {
		cfgFile = filepath.Join(repoRoot, "conf", "app2.json")
	}
	// config.configFilePath читает LOADMON_CONFIG - выставляем явно.
	if err := os.Setenv("LOADMON_CONFIG", cfgFile); err != nil {
		t.Fatal(err)
	}
	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		t.Fatalf("чтение конфигурации %s: %v", cfgFile, err)
	}
	endpoints := selectEndpoints(config.ConfigData.ConfigOV)
	if len(endpoints) == 0 {
		t.Fatalf("в %s нет подходящих ConfigOV (onlyEndpoints=%v onlyAPIVersions=%v)", cfgFile, onlyEndpoints, onlyAPIVersions)
	}
	say("Конфигурация: %s, appliance к проверке: %d", cfgFile, len(endpoints))

	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	var all []serverResult
	for _, ovi := range endpoints {
		res, err := probeEndpoint(ctx, ovi)
		all = append(all, res...)
		if err != nil {
			t.Errorf("%s (API %d): %v", ovi.Endpoint, ovi.Version, err)
		}
	}

	printSummary(all)
	if reportCSV != "" {
		if err := writeCSV(reportCSV, all); err != nil {
			t.Errorf("запись CSV %s: %v", reportCSV, err)
		} else {
			say("CSV записан: %s", reportCSV)
		}
	}
	if len(all) == 0 {
		t.Fatal("ни одного сервера не проверено (проверьте фильтры onlyPoweredOn/serverNameContains/maxServersPerEndpoint)")
	}
}

func selectEndpoints(in []config.OvEndpoint) []config.OvEndpoint {
	var out []config.OvEndpoint
	for _, e := range in {
		if len(onlyEndpoints) > 0 {
			ok := false
			for _, s := range onlyEndpoints {
				if s != "" && strings.Contains(strings.ToLower(e.Endpoint), strings.ToLower(s)) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if len(onlyAPIVersions) > 0 {
			ok := false
			for _, v := range onlyAPIVersions {
				if v == e.Version {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, e)
	}
	return out
}

// probeEndpoint проходит по одному appliance. Возвращает накопленные строки
// даже при ошибке середины прохода.
func probeEndpoint(ctx context.Context, ovi config.OvEndpoint) ([]serverResult, error) {
	say("")
	say("==== %s (ConfigOV.Version=%d, login=%s) ====", ovi.Endpoint, ovi.Version, ovi.Login)

	pwd := ovi.Password
	if pwd == "password" {
		pwd = utilities.GetKeyData("usr-nsk-dati-02@global.bcs")
	}
	if passwordOverride != "" {
		pwd = passwordOverride
	}
	if v := os.Getenv("LOADMON_OV_PASSWORD"); v != "" {
		pwd = v
	}
	cfg := oneview.NewEndpointConfig(ovi.Endpoint, ovi.Domain, ovi.Login, pwd, ovi.Version, requestTimeout)
	c, err := oneview.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("создание клиента: %w", err)
	}
	if err := c.Login(ctx); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	defer c.Logout(ctx)

	if v, err := c.GetVersion(ctx); err == nil {
		say("GET /rest/version: min=%d current=%d; используется X-API-Version=%d, product=%s",
			int(v.MinimumVersion), int(v.CurrentVersion), c.APIVersion(), c.Product())
	} else {
		say("GET /rest/version: %v (используется X-API-Version=%d)", err, c.APIVersion())
	}
	apiLabel := fmt.Sprintf("%d", c.APIVersion())

	servers, err := listServers(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("список server-hardware: %w", err)
	}
	say("server-hardware: получено %d, к проверке после фильтров: %d", len(servers.all), len(servers.selected))

	var results []serverResult
	encCache := map[string]string{} // locationUri -> класс результата шасси (чтобы не дёргать повторно)
	first := true
	for i, hw := range servers.selected {
		r := serverResult{
			Endpoint: ovi.Endpoint, API: apiLabel, Name: hw.Name, URI: hw.URI,
			Model: hw.Model, Generation: hw.Generation, PowerState: hw.PowerState,
			LicensingIntent: hw.LicensingIntent, MpLicenseType: hw.MpLicenseType,
			MpModel: hw.MpModel, MpState: hw.MpState,
		}
		say("")
		say("[%d/%d] %s  %s", i+1, len(servers.selected), hw.Name, hw.URI)
		say("    model=%q gen=%q power=%s licensingIntent=%q mpLicenseType=%q mpModel=%q mpState=%q",
			hw.Model, hw.Generation, hw.PowerState, hw.LicensingIntent, hw.MpLicenseType, hw.MpModel, hw.MpState)

		checkUtilization(ctx, c, &r, utilFields)
		if first && probeAllMetricsOnce {
			probeAllMetrics(ctx, c, hw.URI)
		}
		first = false
		if checkEnvConfig {
			checkEnvironmental(ctx, c, &r, hw.URI)
		}
		if checkPowerSupplies {
			checkPSU(ctx, c, &r, hw.URI)
		}
		if checkEnclosure && hw.LocationURI != "" {
			checkEnclosureOnce(ctx, c, &r, hw.LocationURI, encCache)
		}
		results = append(results, r)
		if pauseBetweenServers > 0 {
			time.Sleep(pauseBetweenServers)
		}
		if ctx.Err() != nil {
			return results, ctx.Err()
		}
	}
	return results, nil
}

type serverList struct {
	all      []oneview.ServerHardware
	selected []oneview.ServerHardware
}

func listServers(ctx context.Context, c *oneview.Client) (*serverList, error) {
	out := &serverList{}
	start := 0
	for {
		col, err := c.ListServerHardware(ctx, oneview.ListOptions{Start: start, Count: pageSize})
		if err != nil {
			return out, err
		}
		out.all = append(out.all, col.Members...)
		start += len(col.Members)
		if len(col.Members) == 0 || start >= col.Total {
			break
		}
	}
	for _, hw := range out.all {
		if serverNameContains != "" && !strings.Contains(strings.ToLower(hw.Name), strings.ToLower(serverNameContains)) {
			continue
		}
		if onlyPoweredOn && !strings.EqualFold(hw.PowerState, "On") {
			continue
		}
		if onlyBlades && hw.LocationURI == "" {
			continue
		}
		out.selected = append(out.selected, hw)
		if maxServersPerEndpoint > 0 && len(out.selected) >= maxServersPerEndpoint {
			break
		}
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// /utilization
// ---------------------------------------------------------------------------

// utilizationURL собирает путь по справочнику HPE: каждое условие filter -
// отдельным параметром, даты в компактном ISO 8601 (yyyyMMddTHHmmss.SSSZ).
func utilizationURL(base, fields string) string {
	const layout = "20060102T150405.000Z"
	end := time.Now().UTC()
	start := end.Add(-time.Duration(windowHours) * time.Hour)
	q := url.Values{}
	for _, f := range strings.Split(fields, ",") {
		if f = strings.TrimSpace(f); f != "" {
			q.Add("fields", f)
		}
	}
	if utilView != "" {
		q.Set("view", utilView)
	}
	q.Add("filter", "startDate="+start.Format(layout))
	q.Add("filter", "endDate="+end.Format(layout))
	return base + "/utilization?" + q.Encode()
}

func checkUtilization(ctx context.Context, c *oneview.Client, r *serverResult, fields string) {
	path := utilizationURL(r.URI, fields)
	status, body, err := doGET(ctx, c, path)
	r.UtilHTTP = status
	say("    GET %s", path)
	if status == 0 {
		r.UtilClass = "ERR"
		r.Note = errText(err)
		say("    -> сбой запроса: %v", err)
		return
	}
	raw(body)
	switch {
	case status == 404:
		r.UtilClass = "NOT_FOUND"
		say("    -> HTTP 404: /utilization не поддерживается этим appliance/API")
		return
	case status >= 400:
		r.UtilClass = "HTTP_ERR"
		r.Note = shorten(string(body), 200)
		say("    -> HTTP %d: %s", status, r.Note)
		return
	}
	var u struct {
		Resolution       json.Number `json:"resolution"`
		IsFresh          *bool       `json:"isFresh"`
		NewestSampleTime string      `json:"newestSampleTime"`
		OldestSampleTime string      `json:"oldestSampleTime"`
		RefreshTaskURI   *string     `json:"refreshTaskUri"`
		MetricList       []struct {
			MetricName     string            `json:"metricName"`
			MetricSamples  []json.RawMessage `json:"metricSamples"`
			MetricCapacity json.RawMessage   `json:"metricCapacity"`
		} `json:"metricList"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		r.UtilClass = "ERR"
		r.Note = "разбор JSON: " + err.Error()
		say("    -> %s", r.Note)
		return
	}
	r.Metrics = map[string]metricStat{}
	haveAny, nonZero := false, false
	for _, m := range u.MetricList {
		st := metricStat{Samples: len(m.MetricSamples), Capacity: strings.TrimSpace(string(m.MetricCapacity))}
		var sum float64
		haveLast := false
		for _, s := range m.MetricSamples {
			v, ok := sampleValue(s)
			if !ok {
				continue
			}
			st.NonNull++
			sum += v
			if !haveLast {
				st.Last, haveLast = v, true
			}
			if v > st.Max {
				st.Max = v
			}
		}
		if st.NonNull > 0 {
			st.Avg = round1(sum / float64(st.NonNull))
			haveAny = true
			if st.Max > 0 {
				nonZero = true
			}
		}
		r.Metrics[m.MetricName] = st
		say("    %-20s сэмплов=%-4d не-null=%-4d last=%-8.1f avg=%-8.1f max=%-8.1f capacity=%s",
			m.MetricName, st.Samples, st.NonNull, st.Last, st.Avg, st.Max, st.Capacity)
	}
	switch {
	case len(u.MetricList) == 0:
		r.UtilClass = "NO_SAMPLES"
	case !haveAny:
		r.UtilClass = "NO_SAMPLES"
	case !nonZero:
		r.UtilClass = "ZEROS"
	default:
		r.UtilClass = "OK"
	}
	say("    -> HTTP 200, класс=%s, resolution=%s мс, newest=%s, isFresh=%s",
		r.UtilClass, u.Resolution.String(), u.NewestSampleTime, boolPtr(u.IsFresh))
}

// probeAllMetrics - /utilization без fields: реальный набор метрик сервера.
func probeAllMetrics(ctx context.Context, c *oneview.Client, uri string) {
	path := utilizationURL(uri, "")
	status, body, err := doGET(ctx, c, path)
	say("    [все метрики] GET %s -> HTTP %d %s", path, status, errText(err))
	if status != 200 {
		return
	}
	var u struct {
		MetricList []struct {
			MetricName string `json:"metricName"`
		} `json:"metricList"`
	}
	if json.Unmarshal(body, &u) == nil {
		var names []string
		for _, m := range u.MetricList {
			names = append(names, m.MetricName)
		}
		say("    [все метрики] metricName: %s", strings.Join(names, ", "))
	}
}

// sampleValue разбирает один сэмпл "[ms, value]"; null/нечисло -> ok=false.
func sampleValue(raw json.RawMessage) (float64, bool) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) < 2 {
		return 0, false
	}
	var v float64
	if err := json.Unmarshal(arr[1], &v); err != nil {
		var s string
		if json.Unmarshal(arr[1], &s) != nil {
			return 0, false
		}
		if _, err := fmt.Sscanf(strings.TrimSpace(s), "%g", &v); err != nil {
			return 0, false
		}
	}
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// ---------------------------------------------------------------------------
// environmentalConfiguration, powerSupplies, шасси
// ---------------------------------------------------------------------------

func checkEnvironmental(ctx context.Context, c *oneview.Client, r *serverResult, uri string) {
	path := uri + "/environmentalConfiguration"
	status, body, err := doGET(ctx, c, path)
	say("    GET %s -> HTTP %d %s", path, status, errText(err))
	if status != 200 {
		return
	}
	raw(body)
	var e map[string]json.RawMessage
	if json.Unmarshal(body, &e) != nil {
		return
	}
	r.LicenseRequirement = unquote(e["licenseRequirement"])
	r.PowerHistorySupported = unquote(e["powerHistorySupported"])
	r.CalibratedMaxW, _ = rawFloat(e["calibratedMaxPower"])
	idle, _ := rawFloat(e["idleMaxPower"])
	say("    env: licenseRequirement=%q powerHistorySupported=%s utilizationHistorySupported=%s capHistorySupported=%s powerCapType=%s calibratedMaxPower=%.0f Вт idleMaxPower=%.0f Вт",
		r.LicenseRequirement, r.PowerHistorySupported, unquote(e["utilizationHistorySupported"]),
		unquote(e["capHistorySupported"]), unquote(e["powerCapType"]), r.CalibratedMaxW, idle)
}

func checkPSU(ctx context.Context, c *oneview.Client, r *serverResult, uri string) {
	path := uri + "/powerSupplies"
	status, body, err := doGET(ctx, c, path)
	r.PSUHTTP = status
	say("    GET %s -> HTTP %d %s", path, status, errText(err))
	if status != 200 {
		return
	}
	raw(body)
	var root any
	if json.Unmarshal(body, &root) != nil {
		return
	}
	// Структура "data" - Redfish-подобная; обходим дерево и суммируем
	// AveragePowerOutputWatts (при отсутствии - LastPowerOutputWatts) по БП.
	var psuAvg, psuLast []float64
	walk(root, func(m map[string]any) {
		if v, ok := toFloat(m["AveragePowerOutputWatts"]); ok {
			psuAvg = append(psuAvg, v)
		}
		if v, ok := toFloat(m["LastPowerOutputWatts"]); ok {
			psuLast = append(psuLast, v)
		}
	})
	vals := psuAvg
	label := "AveragePowerOutputWatts"
	if len(vals) == 0 {
		vals, label = psuLast, "LastPowerOutputWatts"
	}
	for _, v := range vals {
		r.PSUSumW += v
	}
	r.PSUCount = len(vals)
	say("    powerSupplies: БП с %s=%d, сумма=%.0f Вт", label, r.PSUCount, r.PSUSumW)
}

func checkEnclosureOnce(ctx context.Context, c *oneview.Client, r *serverResult, encURI string, cache map[string]string) {
	if cls, ok := cache[encURI]; ok {
		r.EnclosureClass = cls + " (кэш)"
		return
	}
	path := encURI + "/utilization?fields=AveragePower&fields=PeakPower&fields=RatedCapacity&view=hour"
	status, body, err := doGET(ctx, c, path)
	r.EnclosureHTTP = status
	say("    шасси: GET %s -> HTTP %d %s", path, status, errText(err))
	switch {
	case status == 404:
		r.EnclosureClass = "NOT_FOUND"
	case status >= 400 || status == 0:
		r.EnclosureClass = "HTTP_ERR"
	default:
		raw(body)
		var u struct {
			MetricList []struct {
				MetricName    string            `json:"metricName"`
				MetricSamples []json.RawMessage `json:"metricSamples"`
			} `json:"metricList"`
		}
		r.EnclosureClass = "NO_SAMPLES"
		if json.Unmarshal(body, &u) == nil {
			for _, m := range u.MetricList {
				for _, s := range m.MetricSamples {
					if v, ok := sampleValue(s); ok {
						if v > 0 {
							r.EnclosureClass = "OK"
						} else if r.EnclosureClass != "OK" {
							r.EnclosureClass = "ZEROS"
						}
						say("    шасси %s: последнее значение %.1f", m.MetricName, v)
						break
					}
				}
			}
		}
	}
	cache[encURI] = r.EnclosureClass
	say("    шасси: класс=%s", r.EnclosureClass)
}

// ---------------------------------------------------------------------------
// сводка и CSV
// ---------------------------------------------------------------------------

func printSummary(all []serverResult) {
	say("")
	say("================ СВОДКА ================")
	type key struct{ ep, api, cls string }
	byClass := map[key]int{}
	for _, r := range all {
		byClass[key{r.Endpoint, r.API, r.UtilClass}]++
	}
	var keys []key
	for k := range byClass {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].ep != keys[j].ep {
			return keys[i].ep < keys[j].ep
		}
		return keys[i].cls < keys[j].cls
	})
	say("/utilization по appliance (OK=есть ненулевые данные, ZEROS=нули, NO_SAMPLES=пусто, NOT_FOUND=404):")
	for _, k := range keys {
		say("  %-45s API %-5s %-11s %d", k.ep, k.api, k.cls, byClass[k])
	}

	type lk struct{ intent, mp, req string }
	type lv struct{ total, ok int }
	lic := map[lk]*lv{}
	for _, r := range all {
		k := lk{r.LicensingIntent, r.MpLicenseType, r.LicenseRequirement}
		v := lic[k]
		if v == nil {
			v = &lv{}
			lic[k] = v
		}
		v.total++
		if r.UtilClass == "OK" {
			v.ok++
		}
	}
	var lks []lk
	for k := range lic {
		lks = append(lks, k)
	}
	sort.Slice(lks, func(i, j int) bool { return fmt.Sprint(lks[i]) < fmt.Sprint(lks[j]) })
	say("")
	say("Лицензии -> есть ли данные /utilization:")
	say("  %-16s %-34s %-18s %s", "licensingIntent", "mpLicenseType(iLO)", "licenseRequirement", "серверов с данными / всего")
	for _, k := range lks {
		say("  %-16s %-34s %-18s %d / %d", q(k.intent), q(k.mp), q(k.req), lic[k].ok, lic[k].total)
	}

	say("")
	say("Сервер | /utilization | AvgPower(last/avg/max) | PeakPower(max) | PSU сумма Вт | calibratedMax | шасси")
	for _, r := range all {
		ap, pp := r.Metrics["AveragePower"], r.Metrics["PeakPower"]
		say("  %-28s | %-10s | %6.0f/%6.0f/%6.0f | %6.0f | %6.0f | %6.0f | %s",
			cut(r.Name, 28), r.UtilClass, ap.Last, ap.Avg, ap.Max, pp.Max, r.PSUSumW, r.CalibratedMaxW, r.EnclosureClass)
	}
}

func writeCSV(path string, all []serverResult) error {
	var b strings.Builder
	b.WriteString("endpoint;api;server;uri;model;generation;powerState;licensingIntent;mpLicenseType;mpModel;mpState;licenseRequirement;powerHistorySupported;calibratedMaxW;utilHTTP;utilClass;avgLast;avgAvg;avgMax;peakMax;powerCapLast;psuHTTP;psuCount;psuSumW;enclosureClass;note\n")
	for _, r := range all {
		ap, pp, pc := r.Metrics["AveragePower"], r.Metrics["PeakPower"], r.Metrics["PowerCap"]
		fmt.Fprintf(&b, "%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%s;%.0f;%d;%s;%.1f;%.1f;%.1f;%.1f;%.1f;%d;%d;%.0f;%s;%s\n",
			r.Endpoint, r.API, r.Name, r.URI, r.Model, r.Generation, r.PowerState, r.LicensingIntent, r.MpLicenseType,
			r.MpModel, r.MpState, r.LicenseRequirement, r.PowerHistorySupported, r.CalibratedMaxW, r.UtilHTTP, r.UtilClass,
			ap.Last, ap.Avg, ap.Max, pp.Max, pc.Last, r.PSUHTTP, r.PSUCount, r.PSUSumW, r.EnclosureClass,
			strings.ReplaceAll(r.Note, ";", ","))
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// ---------------------------------------------------------------------------
// мелкие помощники
// ---------------------------------------------------------------------------

func doGET(ctx context.Context, c *oneview.Client, path string) (int, []byte, error) {
	resp, err := c.Do(ctx, "GET", path, nil)
	if resp == nil {
		return 0, nil, err
	}
	return resp.StatusCode, resp.Body, err
}

// say печатает сразу в stdout (видно в Debug Console без -v).
func say(format string, a ...any) { fmt.Printf(format+"\n", a...) }

func raw(body []byte) {
	if printRawJSON {
		say("    RAW: %s", shorten(string(body), rawMaxChars))
	}
}

func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func q(s string) string { return fmt.Sprintf("%q", s) }

func errText(err error) string {
	if err == nil {
		return ""
	}
	return "(" + shorten(err.Error(), 160) + ")"
}

func boolPtr(b *bool) string {
	if b == nil {
		return "?"
	}
	return fmt.Sprint(*b)
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func unquote(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return ""
	}
	var str string
	if json.Unmarshal(raw, &str) == nil {
		return str
	}
	return s
}

func rawFloat(raw json.RawMessage) (float64, bool) {
	var v float64
	if json.Unmarshal(raw, &v) == nil {
		return v, true
	}
	return 0, false
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(x), "%g", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}

// walk обходит произвольное JSON-дерево, вызывая fn для каждого объекта.
func walk(n any, fn func(map[string]any)) {
	switch x := n.(type) {
	case map[string]any:
		fn(x)
		for _, v := range x {
			walk(v, fn)
		}
	case []any:
		for _, v := range x {
			walk(v, fn)
		}
	}
}
