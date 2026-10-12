// E7.8 через настоящий роутер и шаблоны: администрирование эталонных конфигураций и отчёт отклонений.
package ui_test

import (
	"net/url"
	"strings"
	"testing"

	"cmdb/internal/testutil"
)

func TestBaselineConfig_AdminPagesRender(t *testing.T) {
	setup(t)
	for _, tc := range []struct {
		path    string
		markers []string
	}{
		{"/admin/baseline-configs", []string{"Эталонные конфигурации", "Новая конфигурация"}},
		{"/admin/baseline-configs/new", []string{"Класс КЕ", "Требуемые атрибуты", "Обязательное ПО"}},
		{"/admin/ref", []string{"Эталонные конфигурации"}},
		{"/reports/baseline-config", []string{"Отклонения от эталонных конфигураций"}},
		{"/reports", []string{"Отклонения от эталонных конфигураций"}},
	} {
		w := send("GET", tc.path, nil, false)
		if w.Code != 200 {
			t.Fatalf("GET %s: status=%d body=%.500s", tc.path, w.Code, w.Body.String())
		}
		mustContain(t, "GET "+tc.path, w.Body.String(), tc.markers...)
	}
}

func TestBaselineConfig_CreateEditDeleteFlow(t *testing.T) {
	setup(t)
	suffix := testutil.Suffix()
	name := "UI-BCFG-" + suffix

	w := send("POST", "/admin/baseline-configs", url.Values{
		"name": {name}, "ci_class": {"server"}, "role": {"веб"}, "attrs_json": {`{"os_family":"Linux"}`}, "enabled": {"1"},
	}, true)
	wantSuccess(t, "создание конфигурации", w)

	w = send("GET", "/admin/baseline-configs", nil, false)
	mustContain(t, "список после создания", w.Body.String(), name)
	// список копит строки от прежних прогонов (своя фикстура, не чистится, как и везде в этом
	// наборе тестов) — ищем id именно в строке нашей конфигурации, не первый попавшийся в таблице.
	id := firstMatch(t, name+`[\s\S]*?/admin/baseline-configs/(\d+)/edit`, w.Body.String())

	w = send("GET", "/admin/baseline-configs/"+id+"/edit", nil, false)
	mustContain(t, "форма правки", w.Body.String(), name, "Linux")

	w = send("POST", "/admin/baseline-configs/"+id, url.Values{
		"name": {name}, "ci_class": {"server"}, "attrs_json": {`{"os_family":"Windows"}`}, "enabled": {"1"},
	}, true)
	wantSuccess(t, "правка конфигурации", w)
	w = send("GET", "/admin/baseline-configs/"+id+"/edit", nil, false)
	mustContain(t, "форма после правки", w.Body.String(), "Windows")

	// некорректный JSON атрибутов отклоняется понятной ошибкой, не падением
	wantFailure(t, "некорректный JSON", send("POST", "/admin/baseline-configs/"+id,
		url.Values{"name": {name}, "ci_class": {"server"}, "attrs_json": {"не json"}}, true))

	w = send("POST", "/admin/baseline-configs/"+id+"/delete", nil, true)
	wantSuccess(t, "удаление конфигурации", w)
	w = send("GET", "/admin/baseline-configs", nil, false)
	if strings.Contains(w.Body.String(), name) {
		t.Errorf("конфигурация %q всё ещё в списке после удаления", name)
	}
	if w = send("GET", "/admin/baseline-configs/"+id+"/edit", nil, false); w.Code != 404 {
		t.Errorf("GET удалённой конфигурации: status=%d, want 404", w.Code)
	}
}

func TestBaselineConfig_ReportCSVExport(t *testing.T) {
	setup(t)
	w := send("GET", "/reports/baseline-config/export.csv", nil, false)
	if w.Code != 200 {
		t.Fatalf("GET /reports/baseline-config/export.csv: status=%d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Errorf("Content-Type=%q, want text/csv", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("Content-Disposition=%q, want attachment", cd)
	}
}
