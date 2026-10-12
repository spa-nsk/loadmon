// E7.7 через настоящий роутер и шаблоны: вкладка «ПО» карточки КЕ — регистрация и снятие установки.
package ui_test

import (
	"net/url"
	"strconv"
	"strings"
	"testing"

	"cmdb/internal/testutil"
)

func TestSoftwareTab_AddAndRemove(t *testing.T) {
	db := setup(t)
	suffix := testutil.Suffix()

	w := send("POST", "/cis", url.Values{
		"class": {"server"}, "location_code": {testutil.TestLocation}, "reason": {"фикстура ПО " + suffix},
	}, true)
	redirect := w.Header().Get("HX-Redirect")
	id := firstMatch(t, `^/cis/(\d+)`, redirect)

	product := "UITestSW-" + suffix
	var swID int64
	if err := db.QueryRow(`INSERT INTO ref.software (vendor, product, version, policy) VALUES ('UITestVendor', $1, '1.0', 'forbidden') RETURNING id`, product).Scan(&swID); err != nil {
		t.Fatalf("ref.software: %v", err)
	}
	swIDStr := strconv.FormatInt(swID, 10)

	// регистрация без причины — отклоняется
	wantFailure(t, "регистрация без причины", send("POST", "/cis/"+id+"/software", url.Values{"software_id": {swIDStr}}, true))

	w = send("POST", "/cis/"+id+"/software", url.Values{
		"software_id": {swIDStr}, "version": {"1.0"}, "install_path": {"/opt/uitest"}, "reason": {"установка"},
	}, true)
	wantSuccess(t, "регистрация установки", w)
	mustContain(t, "вкладка ПО после регистрации", w.Body.String(), "UITestVendor", product, "forbidden", "/opt/uitest")

	instID := firstMatch(t, `/software-installs/(\d+)/remove`, w.Body.String())
	rowVersion := firstMatch(t, `name="row_version" value="(\d+)"`, w.Body.String())

	// снятие без причины — отклоняется
	wantFailure(t, "снятие без причины", send("POST", "/software-installs/"+instID+"/remove",
		url.Values{"ci": {id}, "row_version": {rowVersion}}, true))

	w = send("POST", "/software-installs/"+instID+"/remove", url.Values{"ci": {id}, "row_version": {rowVersion}, "reason": {"снятие"}}, true)
	wantSuccess(t, "снятие установки", w)
	// продукт остаётся в справочнике (выпадающий список формы регистрации) — проверяем пустоту
	// именно таблицы установленного ПО, а не отсутствие строки "product" на всей странице.
	mustContain(t, "вкладка ПО после снятия", w.Body.String(), "ПО не зарегистрировано")
	if strings.Contains(w.Body.String(), "/opt/uitest") {
		t.Errorf("после снятия путь установки всё ещё виден: %s", w.Body.String())
	}
}
