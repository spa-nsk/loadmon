// E6.1: «Показать на дату» через настоящий роутер: карточка/список/вкладки читают прошлое состояние,
// запись запрещена сервером, cookie глобального переключателя работает, ?asof=now её перекрывает.
package ui_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/testutil"
)

func sendWithCookie(method, path string, form url.Values, hx bool, cookie string) *httptest.ResponseRecorder {
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
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	w := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)
	return w
}

func TestUI_AsOfShowsPastStateAndIsReadOnly(t *testing.T) {
	db := setup(t)
	suffix := testutil.Suffix()

	w := send("POST", "/cis", url.Values{"class": {"server"}, "location_code": {testutil.TestLocation}, "reason": {"ввод"},
		"serial_number": {"ASOF-" + suffix}, "cpu_count": {"2"}}, true)
	redirect := w.Header().Get("HX-Redirect")
	id := firstMatch(t, `^/cis/(\d+)`, redirect)
	name, _ := url.QueryUnescape(firstMatch(t, `created=([^&]+)`, redirect))

	time.Sleep(60 * time.Millisecond)
	past := time.Now().UTC()
	time.Sleep(60 * time.Millisecond)

	wantSuccess(t, "правка после момента", send("POST", "/cis/"+id+"/update", url.Values{"row_version": {"1"}, "kii_object_present": {"1"},
		"description": {"описание-после-момента"}, "reason": {"правка"}}, true))
	// атрибут класса тоже правим: cpu_count 2 -> 8
	wantSuccess(t, "правка атрибутов класса", send("POST", "/cis/"+id+"/ext", url.Values{"row_version": {"1"}, "cpu_count": {"8"}, "reason": {"правка класса"}}, true))

	asOf := past.Format(time.RFC3339Nano) // с долями секунды: КЕ создана меньше секунды назад
	now := send("GET", "/cis/"+id, nil, false).Body.String()
	mustContain(t, "текущая карточка", now, "описание-после-момента")
	if strings.Contains(now, "asof-banner") {
		t.Error("без asof баннера режима «только чтение» быть не должно")
	}

	// карточка на момент до правки: старое описание отсутствует, баннер есть
	then := send("GET", "/cis/"+id+"?asof="+url.QueryEscape(asOf), nil, false).Body.String()
	mustContain(t, "карточка на дату", then, name, "asof-banner", "только чтения")
	if strings.Contains(then, "описание-после-момента") {
		t.Error("карточка на дату показывает правку, сделанную ПОЗЖЕ этого момента")
	}
	// вкладка «Атрибуты класса»: на дату cpu_count = 2, сейчас 8
	cls := send("GET", "/cis/"+id+"/tab/class?asof="+url.QueryEscape(asOf), nil, true).Body.String()
	mustContain(t, "класс на дату", cls, `value="2"`)
	if strings.Contains(cls, `value="8"`) {
		t.Error("атрибуты класса на дату показывают значение из будущего")
	}
	if cur := send("GET", "/cis/"+id+"/tab/class", nil, true).Body.String(); !strings.Contains(cur, `value="8"`) {
		t.Error("текущие атрибуты класса должны содержать 8")
	}
	// история на дату: правок ещё не было — только создание
	hist := send("GET", "/cis/"+id+"/tab/history?asof="+url.QueryEscape(asOf), nil, true).Body.String()
	if strings.Contains(hist, "правка класса") || strings.Contains(hist, "описание-после-момента") {
		t.Error("история на дату содержит наборы изменений позже момента")
	}
	// вкладки без версий — понятное пояснение, а не тихо текущие данные
	mustContain(t, "сети на дату", send("GET", "/cis/"+id+"/tab/network?asof="+url.QueryEscape(asOf), nil, true).Body.String(), "не восстанавливаются")
	mustContain(t, "источники на дату", send("GET", "/cis/"+id+"/tab/sources?asof="+url.QueryEscape(asOf), nil, true).Body.String(), "не версионируется")

	// список на дату: КЕ есть, а текущий updated_at позже — строка берётся из cmdb.ci_asof
	list := send("GET", "/cis/data?q="+url.QueryEscape(name)+"&asof="+url.QueryEscape(asOf), nil, true).Body.String()
	mustContain(t, "список на дату", list, name)
	// запись на дату запрещена сервером (не только скрытыми кнопками)
	w = sendWithCookie("POST", "/cis/"+id+"/update", url.Values{"row_version": {"2"}, "description": {"из прошлого"}, "reason": {"попытка"}}, true, "asof="+asOf)
	if w.Code != http.StatusConflict || !strings.Contains(trigger(w), `"warning"`) || w.Header().Get("HX-Reswap") != "none" {
		t.Errorf("запись в режиме на дату: status=%d trigger=%q reswap=%q, want 409 + тост warning + reswap none", w.Code, trigger(w), w.Header().Get("HX-Reswap"))
	}
	var desc string
	db.QueryRow(`SELECT COALESCE(description, '') FROM cmdb.ci WHERE id = $1`, id).Scan(&desc)
	if desc != "описание-после-момента" {
		t.Errorf("описание после отклонённой записи = %q — запись не должна была пройти", desc)
	}

	// cookie глобального переключателя действует на любой запрос; ?asof=now перекрывает её
	viaCookie := sendWithCookie("GET", "/cis/"+id, nil, false, "asof="+asOf).Body.String()
	mustContain(t, "карточка по cookie", viaCookie, "asof-banner")
	if strings.Contains(viaCookie, "описание-после-момента") {
		t.Error("cookie asof не применилась к карточке")
	}
	overridden := sendWithCookie("GET", "/cis/"+id+"?asof=now", nil, false, "asof="+asOf).Body.String()
	if strings.Contains(overridden, "asof-banner") || !strings.Contains(overridden, "описание-после-момента") {
		t.Error("?asof=now должен показать текущее состояние даже при заданной cookie")
	}
	// мусор в cookie не ломает страницу
	if w := sendWithCookie("GET", "/cis/"+id, nil, false, "asof=не-дата"); w.Code != 200 || strings.Contains(w.Body.String(), "asof-banner") {
		t.Errorf("некорректная cookie asof: status=%d, want обычная страница", w.Code)
	}
	// переключатель есть в layout
	mustContain(t, "переключатель", now, `id="asof-input"`, "На дату")
}
