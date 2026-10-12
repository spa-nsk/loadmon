// E7.6 через настоящий роутер: журнал действий — страница, фильтры, выгрузка CSV/CEF, и что
// admin/ref CRUD теперь тоже пишет в audit.event (было пробелом — только Export логировался).
package ui_test

import (
	"net/url"
	"strings"
	"testing"

	"cmdb/internal/testutil"
)

func TestEvents_PageRendersAndFilters(t *testing.T) {
	setup(t)
	w := send("GET", "/admin/events", nil, false)
	if w.Code != 200 {
		t.Fatalf("GET /admin/events: status=%d body=%.500s", w.Code, w.Body.String())
	}
	mustContain(t, "журнал действий", w.Body.String(), "Журнал действий", "Выгрузить CSV", "Выгрузить CEF")
}

func TestEvents_CSVAndCEFExports(t *testing.T) {
	setup(t)
	for _, tc := range []struct {
		path   string
		wantCT string
	}{
		{"/admin/events/export.csv", "text/csv"},
		{"/admin/events/export.cef", "text/plain"},
	} {
		w := send("GET", tc.path, nil, false)
		if w.Code != 200 {
			t.Fatalf("GET %s: status=%d body=%.300s", tc.path, w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, tc.wantCT) {
			t.Errorf("GET %s: Content-Type=%q, want содержит %q", tc.path, ct, tc.wantCT)
		}
		if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
			t.Errorf("GET %s: Content-Disposition=%q, want attachment", tc.path, cd)
		}
	}
}

// TestEvents_RefAdminActionsAreLogged — создание/правка/удаление в /admin/ref пишет в audit.event
// (был пробел: логировался только Export). Использует справочник environment — без внешних
// зависимостей (FK на другие таблицы) и без риска задеть данные, нужные другим тестам.
func TestEvents_RefAdminActionsAreLogged(t *testing.T) {
	setup(t)
	code := "TST" + testutil.Suffix()

	send("POST", "/admin/ref/environment", url.Values{"environment_code": {code}, "name": {"Тестовое " + code}}, false)
	send("POST", "/admin/ref/environment/"+code+"/edit", url.Values{"name": {"Тестовое изменённое " + code}}, false)
	send("POST", "/admin/ref/environment/"+code+"/delete", nil, false)

	w := send("GET", "/admin/events?q="+code, nil, false)
	if w.Code != 200 {
		t.Fatalf("GET /admin/events?q=%s: status=%d", code, w.Code)
	}
	body := w.Body.String()
	mustContain(t, "журнал действий ref.*", body, "ref_create", "ref_update", "ref_delete")
}

// TestRefAdmin_EditMissingRowReturns404 — устаревшая ссылка на запись справочника отвечает 404 со
// страницей, а не 200 (StopRun() после SetStatus(404)+page() терял статус — найдено и исправлено
// в E7.8 заодно с той же ошибкой в internal/controllers/ui/baselineconfig.go).
func TestRefAdmin_EditMissingRowReturns404(t *testing.T) {
	setup(t)
	w := send("GET", "/admin/ref/environment/NO-SUCH-CODE-"+testutil.Suffix()+"/edit", nil, false)
	if w.Code != 404 {
		t.Errorf("GET несуществующей записи справочника: status=%d, want 404", w.Code)
	}
	mustContain(t, "страница 404", w.Body.String(), "Запись не найдена")
}

// TestImport_RunDetailMissingReturns404 — тот же класс ошибки (StopRun терял тело ответа), уже
// у прогона импорта.
func TestImport_RunDetailMissingReturns404(t *testing.T) {
	setup(t)
	w := send("GET", "/import/runs/999999999", nil, false)
	if w.Code != 404 {
		t.Errorf("GET несуществующего прогона импорта: status=%d, want 404", w.Code)
	}
	mustContain(t, "страница 404 прогона импорта", w.Body.String(), "Прогон импорта не найден")
}
