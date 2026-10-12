// Сквозной тест UI: настоящий роутер Beego + шаблоны + PostgreSQL (миграции до 0013, seed nc2026).
// CMDB_TEST_DB_DSN=... go test ./internal/controllers/ui/... -v
package ui_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/filters"
	"cmdb/internal/routers"
	"cmdb/internal/services/auth"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

var setupOnce sync.Once

func setup(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.DB(t)
	setupOnce.Do(func() {
		dir, err := filepath.Abs(filepath.Join("..", "..", "..", "views"))
		if err != nil {
			t.Fatal(err)
		}
		web.BConfig.WebConfig.ViewsPath = dir
		web.BConfig.WebConfig.EnableXSRF = false // токен проверяется beego; здесь важна логика страниц
		web.BConfig.CopyRequestBody = true
		if err := web.AddViewPath(dir); err != nil {
			t.Fatal(err)
		}
		filters.SetAuth(&auth.Service{}, true) // сквозные тесты страниц идут без входа (от dev-администратора); вход — auth_integration_test.go
		routers.Init()
	})
	return db
}

// send выполняет запрос как HTMX (hx=true) или обычный браузер.
func send(method, path string, form url.Values, hx bool) *httptest.ResponseRecorder {
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if hx {
		req.Header.Set("HX-Request", "true")
	}
	w := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)
	return w
}

func mustContain(t *testing.T, what, body string, markers ...string) {
	t.Helper()
	for _, m := range markers {
		if !strings.Contains(body, m) {
			t.Errorf("%s: в ответе нет %q\n--- ответ (начало) ---\n%.600s", what, m, body)
		}
	}
}

func trigger(w *httptest.ResponseRecorder) string { return w.Header().Get("HX-Trigger") }

// wantFailure проверяет ответ-ошибку: тост danger, страница не меняется, заголовок — чистый ASCII.
func wantFailure(t *testing.T, what string, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Header().Get("HX-Reswap") != "none" || !strings.Contains(trigger(w), `"danger"`) {
		t.Errorf("%s: ожидалась ошибка (HX-Reswap=none, тост danger), получено status=%d reswap=%q trigger=%q",
			what, w.Code, w.Header().Get("HX-Reswap"), trigger(w))
	}
	for _, r := range trigger(w) {
		if r > 126 {
			t.Errorf("%s: в HX-Trigger не-ASCII символ %q", what, r)
			break
		}
	}
}

func wantSuccess(t *testing.T, what string, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != 200 || !strings.Contains(trigger(w), `"success"`) {
		t.Errorf("%s: ожидался успех, получено status=%d trigger=%q\n%.500s", what, w.Code, trigger(w), w.Body.String())
	}
}

func firstMatch(t *testing.T, re, body string) string {
	t.Helper()
	m := regexp.MustCompile(re).FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("не найдено %q в ответе:\n%.800s", re, body)
	}
	return m[1]
}

func TestUI_FullFlow(t *testing.T) {
	db := setup(t)
	suffix := testutil.Suffix()

	// --- список и форма создания ---
	w := send("GET", "/cis", nil, false)
	mustContain(t, "GET /cis", w.Body.String(), "Конфигурационные единицы", `id="ci-grid"`)
	w = send("GET", "/cis/new", nil, false)
	mustContain(t, "GET /cis/new", w.Body.String(), "Новая конфигурационная единица", "Физический сервер", "Виртуальная машина")
	w = send("GET", "/cis/new/fields?class=vm", nil, true)
	mustContain(t, "поля ВМ", w.Body.String(), "vCPU", `type="checkbox"`)
	w = send("GET", "/cis/name-preview?class=server&location_code="+testutil.TestLocation, nil, true)
	mustContain(t, "превью имени", w.Body.String(), testutil.TestSite+"-S-NNNNN")

	// --- создание: ошибки ---
	wantFailure(t, "создание без причины", send("POST", "/cis", url.Values{
		"class": {"server"}, "location_code": {testutil.TestLocation}}, true))
	wantFailure(t, "создание без площадки", send("POST", "/cis", url.Values{
		"class": {"server"}, "reason": {"тест"}}, true))
	w = send("POST", "/cis", url.Values{"class": {"server"}, "location_code": {testutil.TestLocation}, "reason": {"тест"}, "cpu_count": {"abc"}}, true)
	wantFailure(t, "создание с нечисловым cpu_count", w)
	mustContain(t, "подсветка поля", trigger(w), "fieldErrors", "cpu_count")

	// --- создание: успех ---
	w = send("POST", "/cis", url.Values{
		"class": {"server"}, "location_code": {testutil.TestLocation}, "reason": {"ввод сервера"},
		"serial_number": {"UI-" + suffix}, "cpu_count": {"2"}, "os_family": {"Linux"}, "template": {"1"},
	}, true)
	redirect := w.Header().Get("HX-Redirect")
	if !strings.HasPrefix(redirect, "/cis/") {
		t.Fatalf("создание: HX-Redirect = %q, status=%d, trigger=%q", redirect, w.Code, trigger(w))
	}
	id := firstMatch(t, `^/cis/(\d+)`, redirect)
	name := firstMatch(t, `created=([^&]+)`, redirect)
	name, _ = url.QueryUnescape(name)
	if !strings.HasPrefix(name, testutil.TestSite+"-S-") {
		t.Errorf("имя = %q, ожидался префикс %s-S-", name, testutil.TestSite)
	}

	// --- карточка и все вкладки ---
	w = send("GET", redirect, nil, false)
	mustContain(t, "карточка", w.Body.String(), name, "КЕ создана", "Жизненный цикл", `id="card-tabs"`)
	for tab, marker := range map[string]string{
		"general": "Причина изменения", "class": "Физический сервер", "components": "Добавить компонент",
		"software": "Зарегистрировать установку", "network": "Добавить интерфейс", "relations": "Добавить связь", "history": "Набор изменений",
	} {
		w = send("GET", "/cis/"+id+"/tab/"+tab, nil, true)
		if w.Code != 200 {
			t.Errorf("вкладка %s: status %d", tab, w.Code)
		}
		mustContain(t, "вкладка "+tab, w.Body.String(), marker)
	}
	if w = send("GET", "/cis/999999999", nil, false); w.Code != 404 {
		t.Errorf("несуществующая КЕ: status %d, want 404", w.Code)
	}

	// --- правка общих полей: успех, отсутствие изменений, конфликт версии ---
	w = send("POST", "/cis/"+id+"/update", url.Values{"row_version": {"1"}, "kii_object_present": {"1"},
		"description": {"проверка правки"}, "criticality": {"high"}, "reason": {"уточнение"}}, true)
	wantSuccess(t, "правка", w)
	mustContain(t, "правка", w.Body.String(), "проверка правки", `value="2"`)

	w = send("POST", "/cis/"+id+"/update", url.Values{"row_version": {"2"}, "kii_object_present": {"1"},
		"description": {"проверка правки"}, "criticality": {"high"}, "reason": {"повтор"}}, true)
	if !strings.Contains(trigger(w), `"info"`) || w.Header().Get("HX-Reswap") != "none" {
		t.Errorf("правка без изменений: trigger=%q reswap=%q, want info + none", trigger(w), w.Header().Get("HX-Reswap"))
	}
	wantFailure(t, "правка устаревшей версии", send("POST", "/cis/"+id+"/update", url.Values{
		"row_version": {"1"}, "description": {"конфликт"}, "reason": {"устарело"}}, true))
	wantFailure(t, "правка без причины", send("POST", "/cis/"+id+"/update", url.Values{
		"row_version": {"2"}, "description": {"x"}}, true))

	// --- атрибуты класса ---
	w = send("POST", "/cis/"+id+"/ext", url.Values{"row_version": {"1"}, "cpu_count": {"4"}, "ram_gb": {"16"}, "os_family": {"Linux"},
		"template": {}, "reason": {"замена процессоров"}}, true)
	wantSuccess(t, "атрибуты класса", w)
	mustContain(t, "атрибуты класса", w.Body.String(), `value="4"`)
	wantFailure(t, "атрибуты: неверный IP", send("POST", "/cis/"+id+"/ext", url.Values{
		"row_version": {"2"}, "bmc_ip": {"999.1.1.1"}, "reason": {"ошибка"}}, true))

	// --- компоненты ---
	wantFailure(t, "компонент без типа", send("POST", "/cis/"+id+"/components", url.Values{"reason": {"x"}}, true))
	w = send("POST", "/cis/"+id+"/components", url.Values{"component_type": {"ssd"}, "slot": {"0"},
		"serial_number": {"SSD-" + suffix}, "capacity_value": {"480"}, "capacity_unit": {"GB"}, "reason": {"установка диска"}}, true)
	wantSuccess(t, "добавление компонента", w)
	mustContain(t, "добавление компонента", w.Body.String(), "SSD-"+suffix, "480")
	compID := firstMatch(t, `/components/(\d+)/replace`, w.Body.String())
	w = send("POST", "/components/"+compID+"/replace", url.Values{"ci": {id}, "row_version": {"1"},
		"serial_number": {"SSD2-" + suffix}, "reason": {"отказ диска"}}, true)
	wantSuccess(t, "замена компонента", w)
	mustContain(t, "замена компонента", w.Body.String(), "SSD2-"+suffix, "replaced")

	// --- сеть ---
	wantFailure(t, "интерфейс без имени", send("POST", "/cis/"+id+"/interfaces", url.Values{"reason": {"x"}}, true))
	w = send("POST", "/cis/"+id+"/interfaces", url.Values{"name": {"eth0"}, "if_type": {"ethernet"}, "speed_mbps": {"10000"},
		"reason": {"добавлен интерфейс"}}, true)
	wantSuccess(t, "интерфейс", w)
	ifaceID := firstMatch(t, `<option value="(\d+)">eth0</option>`, w.Body.String())
	wantFailure(t, "дубликат интерфейса", send("POST", "/cis/"+id+"/interfaces", url.Values{"name": {"eth0"}, "reason": {"дубль"}}, true))

	vrf := "ui" + suffix
	w = send("POST", "/cis/"+id+"/ips", url.Values{"address": {"10.99.1.5/24"}, "interface_id": {ifaceID}, "vrf": {vrf},
		"dns_name": {"ui-" + suffix + ".test"}, "reason": {"назначение адреса"}}, true)
	wantSuccess(t, "IP", w)
	mustContain(t, "IP", w.Body.String(), "10.99.1.5/24")
	ipID := firstMatch(t, `/ips/(\d+)/release`, w.Body.String())
	wantFailure(t, "дубликат IP", send("POST", "/cis/"+id+"/ips", url.Values{"address": {"10.99.1.5"}, "vrf": {vrf}, "reason": {"дубль"}}, true))
	wantFailure(t, "некорректный IP", send("POST", "/cis/"+id+"/ips", url.Values{"address": {"10.99.1.999"}, "reason": {"ошибка"}}, true))
	w = send("POST", "/ips/"+ipID+"/release", url.Values{"ci": {id}, "row_version": {"1"}, "reason": {"адрес больше не нужен"}}, true)
	wantSuccess(t, "освобождение IP", w)
	if strings.Contains(w.Body.String(), "10.99.1.5/24") {
		t.Error("освобождённый адрес остался в таблице")
	}

	// --- связи ---
	peer, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "партнёр для связи", UserID: 1})
	if err != nil {
		t.Fatalf("CreateServer (партнёр): %v", err)
	}
	wantFailure(t, "связь с неизвестной КЕ", send("POST", "/cis/"+id+"/relations", url.Values{
		"rel_type": {"depends_on"}, "peer_name": {"NO-SUCH-CI"}, "reason": {"x"}}, true))
	w = send("POST", "/cis/"+id+"/relations", url.Values{"rel_type": {"depends_on"}, "direction": {"out"},
		"peer_name": {strings.ToLower(peer.Name)}, "reason": {"зависимость"}}, true) // имя без учёта регистра
	wantSuccess(t, "связь", w)
	mustContain(t, "связь", w.Body.String(), peer.Name)
	wantFailure(t, "дубликат связи", send("POST", "/cis/"+id+"/relations", url.Values{
		"rel_type": {"depends_on"}, "peer_name": {peer.Name}, "reason": {"дубль"}}, true))
	relID := firstMatch(t, `/relations/(\d+)/close`, w.Body.String())
	w = send("POST", "/relations/"+relID+"/close", url.Values{"ci": {id}, "row_version": {"1"}, "reason": {"связь разорвана"}}, true)
	wantSuccess(t, "закрытие связи", w)
	mustContain(t, "закрытие связи", w.Body.String(), "Действующих связей нет")

	// --- жизненный цикл ---
	var version string
	if err := db.QueryRow(`SELECT row_version::text FROM cmdb.ci WHERE id = $1`, id).Scan(&version); err != nil {
		t.Fatal(err)
	}
	wantFailure(t, "недопустимый переход", send("POST", "/cis/"+id+"/lifecycle", url.Values{
		"row_version": {version}, "to_stage": {"Operational"}, "reason": {"минуя ввод"}}, true))
	w = send("POST", "/cis/"+id+"/lifecycle", url.Values{"row_version": {version}, "to_stage": {"Commissioning"}, "reason": {"начало ввода"}}, true)
	if w.Header().Get("HX-Redirect") != "/cis/"+id {
		t.Errorf("смена стадии: HX-Redirect = %q, trigger=%q", w.Header().Get("HX-Redirect"), trigger(w))
	}

	// --- история: все шаги видны ---
	w = send("GET", "/cis/"+id+"/tab/history", nil, true)
	mustContain(t, "история", w.Body.String(), "ввод сервера", "уточнение", "замена процессоров", "установка диска",
		"отказ диска", "добавлен интерфейс", "назначение адреса", "адрес больше не нужен", "зависимость", "связь разорвана", "начало ввода",
		"description", "lifecycle_stage")

	// --- список: страница и JSON-источник данных грида Tabulator (E3.1) ---
	w = send("GET", "/cis", nil, false)
	mustContain(t, "страница списка", w.Body.String(), "Конфигурационные единицы", `id="ci-grid"`)

	w = send("GET", "/cis/data?q="+url.QueryEscape(name)+"&stage=*", nil, true)
	mustContain(t, "поиск по имени", w.Body.String(), name, `"total":1`)
	w = send("GET", "/cis/data?q=UI-"+suffix, nil, true)
	mustContain(t, "поиск по серийному номеру", w.Body.String(), name)
	w = send("GET", "/cis/data?q=UI-"+suffix+"&class=vm", nil, true)
	mustContain(t, "фильтр по классу", w.Body.String(), `"total":0`)
	w = send("GET", "/cis/data?sort=updated&dir=desc&per=50&page=1&location="+testutil.TestLocation, nil, true)
	if w.Code != 200 {
		t.Errorf("список с сортировкой: status %d", w.Code)
	}

	// --- экспорт списка (E3.3) ---
	w = send("GET", "/cis/export.csv?q="+url.QueryEscape(name), nil, false)
	if w.Code != 200 || !strings.Contains(w.Body.String(), name) {
		t.Errorf("экспорт CSV: status %d, содержит имя: %v", w.Code, strings.Contains(w.Body.String(), name))
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("экспорт CSV: Content-Type = %q", ct)
	}
	w = send("GET", "/cis/export.xlsx?q="+url.QueryEscape(name), nil, false)
	if w.Code != 200 || w.Body.Len() == 0 {
		t.Errorf("экспорт XLSX: status %d, длина тела %d", w.Code, w.Body.Len())
	}
}

// Создание ВМ через ту же универсальную форму (класс с другой таблицей-расширением и кодом V).
func TestUI_CreateVM(t *testing.T) {
	setup(t)
	w := send("POST", "/cis", url.Values{
		"class": {"vm"}, "location_code": {testutil.TestLocation}, "reason": {"создание ВМ"},
		"vcpu": {"4"}, "guest_os": {"Astra Linux"}, "template": {"1"},
	}, true)
	redirect := w.Header().Get("HX-Redirect")
	if !strings.HasPrefix(redirect, "/cis/") {
		t.Fatalf("создание ВМ: HX-Redirect = %q, trigger=%q", redirect, trigger(w))
	}
	name, _ := url.QueryUnescape(firstMatch(t, `created=([^&]+)`, redirect))
	if !strings.HasPrefix(name, testutil.TestSite+"-V-") {
		t.Errorf("имя ВМ = %q, ожидался префикс %s-V-", name, testutil.TestSite)
	}
	id := firstMatch(t, `^/cis/(\d+)`, redirect)
	w = send("GET", "/cis/"+id+"/tab/class", nil, true)
	mustContain(t, "атрибуты ВМ", w.Body.String(), "Виртуальная машина", "Astra Linux", `value="4"`, "checked")
}
