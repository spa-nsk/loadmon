// E6.2–E6.5 через настоящий роутер: создание, заморозка, проверка, выгрузка и сравнение срезов;
// закрытие и переоткрытие периодов.
package ui_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"cmdb/internal/testutil"
)

func TestUI_BaselineLifecycleExportAndCompare(t *testing.T) {
	db := setup(t)
	suffix := testutil.Suffix()
	loc := testutil.NewLocation(t, db)

	// КЕ в отдельной площадке — срез с фильтром области содержит только её
	w := send("POST", "/cis", url.Values{"class": {"server"}, "location_code": {loc}, "reason": {"ввод"}}, true)
	id := firstMatch(t, `^/cis/(\d+)`, w.Header().Get("HX-Redirect"))
	time.Sleep(60 * time.Millisecond)

	mustContain(t, "список срезов", send("GET", "/baselines", nil, false).Body.String(), "Срезы (baseline)", "Создать срез", `id="bl-name"`)

	name := "ui-срез-" + suffix
	wantFailure(t, "срез без имени", send("POST", "/baselines", url.Values{"location_code": {loc}}, true))
	wantFailure(t, "срез с мусорной датой", send("POST", "/baselines", url.Values{"name": {"x" + suffix}, "as_of": {"вчера"}}, true))
	w = send("POST", "/baselines", url.Values{"name": {name}, "location_code": {loc}}, true)
	redirect := w.Header().Get("HX-Redirect")
	if !strings.HasPrefix(redirect, "/baselines/") {
		t.Fatalf("создание среза: HX-Redirect=%q status=%d trigger=%q", redirect, w.Code, trigger(w))
	}
	bid := firstMatch(t, `^/baselines/(\d+)`, redirect)
	wantFailure(t, "повтор имени", send("POST", "/baselines", url.Values{"name": {name}}, true))

	show := send("GET", redirect, nil, false).Body.String()
	mustContain(t, "карточка среза", show, name, "черновик", "Заморозить", "площадка: "+loc)
	if strings.Contains(show, "export/csv") {
		t.Error("у черновика не должно быть выгрузки")
	}
	wantFailure(t, "выгрузка черновика", send("GET", "/baselines/"+bid+"/export/csv", nil, false))

	// после среза правим КЕ — срез и текущее различаются
	wantSuccess(t, "правка после среза", send("POST", "/cis/"+id+"/update", url.Values{"row_version": {"1"}, "kii_object_present": {"1"},
		"description": {"правка-после-среза"}, "reason": {"правка"}}, true))

	wantSuccess(t, "заморозка", send("POST", "/baselines/"+bid+"/freeze", nil, true))
	show = send("GET", "/baselines/"+bid, nil, false).Body.String()
	mustContain(t, "замороженный срез", show, "заморожен", "SHA-256", "Проверить целостность", "export/csv", "export/xlsx", "export/json", "Утвердить")
	if strings.Contains(show, "hx-post=\"/baselines/"+bid+"/freeze\"") {
		t.Error("у замороженного среза не должно быть кнопки «Заморозить»")
	}
	wantFailure(t, "повторная заморозка", send("POST", "/baselines/"+bid+"/freeze", nil, true))
	wantFailure(t, "удаление замороженного", send("POST", "/baselines/"+bid+"/delete", nil, true))

	w = send("POST", "/baselines/"+bid+"/verify", nil, true)
	if !strings.Contains(trigger(w), `"success"`) {
		t.Errorf("проверка целостности: trigger=%q, want success", trigger(w))
	}

	// выгрузка: заголовки вложения, манифест с контрольной суммой
	w = send("GET", "/baselines/"+bid+"/export/json", nil, false)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Disposition"), "baseline-"+bid+".json") {
		t.Fatalf("export.json: status=%d disposition=%q", w.Code, w.Header().Get("Content-Disposition"))
	}
	var doc struct {
		Manifest struct {
			Checksum string `json:"checksum_sha256"`
			Name     string `json:"name"`
		} `json:"manifest"`
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil || len(doc.Manifest.Checksum) != 64 || doc.Manifest.Name != name || len(doc.Items) == 0 {
		t.Fatalf("export.json: %v, манифест=%+v, позиций=%d", err, doc.Manifest, len(doc.Items))
	}
	csv := send("GET", "/baselines/"+bid+"/export/csv", nil, false)
	mustContain(t, "export.csv", csv.Body.String(), "# checksum_sha256: "+doc.Manifest.Checksum, "entity,entity_id")
	if x := send("GET", "/baselines/"+bid+"/export/xlsx", nil, false); x.Code != 200 || x.Body.Len() < 1000 {
		t.Errorf("export.xlsx: status=%d размер=%d", x.Code, x.Body.Len())
	}
	wantFailure(t, "неизвестный формат", send("GET", "/baselines/"+bid+"/export/pdf", nil, false))

	// сравнение срез ↔ текущее: видна правка описания
	cmp := send("GET", "/baselines/compare?a=b:"+bid+"&b=now", nil, false).Body.String()
	mustContain(t, "сравнение", cmp, "Сравнение", "изменено: 1", "description", "правка-после-среза")
	csvCmp := send("GET", "/baselines/compare?a=b:"+bid+"&b=now&format=csv", nil, false)
	mustContain(t, "CSV сравнения", csvCmp.Body.String(), "# добавлено / удалено / изменено", "description")
	wantFailure(t, "сравнение с мусором", send("GET", "/baselines/compare?a=мусор&b=now", nil, true))
	if w := send("GET", "/baselines/compare?a=b:999999999&b=now", nil, false); w.Code != 404 {
		t.Errorf("сравнение с несуществующим срезом: status=%d, want 404", w.Code)
	}
	if w := send("GET", "/baselines/999999999", nil, false); w.Code != 404 {
		t.Errorf("несуществующий срез: status=%d, want 404", w.Code)
	} else {
		mustContain(t, "страница 404 несуществующего среза", w.Body.String(), "Срез не найден")
	}

	wantSuccess(t, "утверждение", send("POST", "/baselines/"+bid+"/approve", nil, true))
	mustContain(t, "утверждённый срез", send("GET", "/baselines/"+bid, nil, false).Body.String(), "утверждён")
}

func TestUI_PeriodsCloseReopenAndReport(t *testing.T) {
	db := setup(t)
	_ = db
	month := time.Date(1960+int(time.Now().UnixNano()/7%40), time.Month(1+int(time.Now().UnixNano()/13%12)), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")

	mustContain(t, "страница периодов", send("GET", "/periods", nil, false).Body.String(), "Учётные периоды", "Правки задним числом", `id="pc-month"`)
	wantFailure(t, "закрытие мусорного периода", send("POST", "/periods/close", url.Values{"month": {"март"}}, true))
	wantFailure(t, "закрытие текущего месяца", send("POST", "/periods/close", url.Values{"month": {time.Now().UTC().Format("2006-01")}}, true))
	wantSuccess(t, "закрытие", send("POST", "/periods/close", url.Values{"month": {month}, "reason": {"ui-тест"}}, true))
	t.Cleanup(func() { db.Exec(`DELETE FROM rpt.accounting_period WHERE period = $1`, month+"-01") })
	mustContain(t, "период в списке", send("GET", "/periods", nil, false).Body.String(), month, "закрыт", "ui-тест")
	wantFailure(t, "повторное закрытие", send("POST", "/periods/close", url.Values{"month": {month}}, true))
	wantFailure(t, "переоткрытие без причины", send("POST", "/periods/reopen", url.Values{"month": {month}}, true))
	wantSuccess(t, "переоткрытие", send("POST", "/periods/reopen", url.Values{"month": {month}, "reason": {"ошибка закрытия"}}, true))
	mustContain(t, "после переоткрытия", send("GET", "/periods", nil, false).Body.String(), "ошибка закрытия")
}
