// E7.5/E7.7/E7.8/E7.9 через настоящий роутер и шаблоны: страницы /reports/* рендерятся, CSV
// отдаётся с правильным Content-Type, а журнал изменений (/changes/export.csv) — тоже.
package ui_test

import (
	"strings"
	"testing"
)

func TestReports_PagesRender(t *testing.T) {
	setup(t)
	for _, tc := range []struct {
		path    string
		markers []string
	}{
		{"/reports", []string{"Отчёты", "Инвентарь по локации/стойке", "Ёмкость СХД", "ВМ по кластерам", "КЕ без владельца", "Отклонения именования"}},
		{"/reports/inventory", []string{"Инвентарь по локации/стойке"}},
		{"/reports/storage", []string{"Ёмкость СХД", "Массивы", "Пулы"}},
		{"/reports/vm-clusters", []string{"ВМ по кластерам"}},
		{"/reports/no-owner", []string{"КЕ без владельца"}},
		{"/reports/naming", []string{"Отклонения именования"}},
		{"/reports/software", []string{"Запрещённое/неразрешённое ПО"}},
		{"/reports/baseline-config", []string{"Отклонения от эталонных конфигураций"}},
		{"/reports/stale-inventory", []string{"Свежесть инвентаризации"}},
		{"/reports/kii", []string{"Объекты КИИ и их состав"}},
		{"/reports/domestic", []string{"Доля отечественного оборудования и ПО"}},
		{"/reports/legacy-names", []string{"Legacy-имена"}},
	} {
		w := send("GET", tc.path, nil, false)
		if w.Code != 200 {
			t.Fatalf("GET %s: status=%d body=%.500s", tc.path, w.Code, w.Body.String())
		}
		mustContain(t, "GET "+tc.path, w.Body.String(), tc.markers...)
	}
}

func TestReports_CSVExports(t *testing.T) {
	setup(t)
	for _, path := range []string{
		"/reports/inventory/export.csv",
		"/reports/storage/export.csv",
		"/reports/vm-clusters/export.csv",
		"/reports/no-owner/export.csv",
		"/reports/naming/export.csv",
		"/reports/software/export.csv",
		"/reports/baseline-config/export.csv",
		"/reports/stale-inventory/export.csv",
		"/reports/kii/export.csv",
		"/reports/domestic/hardware.csv",
		"/reports/domestic/software.csv",
		"/reports/legacy-names/export.csv",
		"/changes/export.csv",
	} {
		w := send("GET", path, nil, false)
		if w.Code != 200 {
			t.Fatalf("GET %s: status=%d body=%.300s", path, w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
			t.Errorf("GET %s: Content-Type=%q, want text/csv", path, ct)
		}
		if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
			t.Errorf("GET %s: Content-Disposition=%q, want attachment", path, cd)
		}
	}
}
