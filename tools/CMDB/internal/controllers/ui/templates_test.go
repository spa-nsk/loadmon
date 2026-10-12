package ui

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/importer/pipeline"
	"cmdb/internal/repo"
	"cmdb/internal/services/baseline"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/component"
	"cmdb/internal/services/history"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/period"
	"cmdb/internal/services/refadmin"
	"cmdb/internal/services/refdata"
	"cmdb/internal/services/relation"
)

// Тест не требует БД: рендерит каждый шаблон на фиктивных данных той же формы, что кладут
// контроллеры, и ловит ошибки шаблонов (неверные имена полей, функций, типов) до запуска сервера.

type tplData = map[interface{}]interface{}

func viewsDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "views"))
	if err != nil {
		t.Fatal(err)
	}
	if err := web.AddViewPath(dir); err != nil { // регистрирует путь и собирает шаблоны; повторный вызов безопасен
		t.Fatalf("AddViewPath: %v", err)
	}
	return dir
}

func render(t *testing.T, dir, name string, data tplData) string {
	t.Helper()
	var buf bytes.Buffer
	if err := web.ExecuteViewPathTemplate(&buf, name, dir, data); err != nil {
		t.Fatalf("шаблон %s: %v", name, err)
	}
	return buf.String()
}

func fakeDetail() *ci.Detail {
	return &ci.Detail{
		CI: repo.Row{
			"id": "7", "uid": "0190-uid", "name": "TSTQA1-S-01001", "ci_class": "server", "lifecycle_stage": "Operational",
			"lifecycle_status": "", "naming_status": "compliant", "environment_code": "PROD", "criticality": "high",
			"ownership": "own", "serial_number": "SN1", "kii_object": "true", "row_version": "3", "location_code": "TST-QA1",
			"created_fmt": "2026-09-25 10:00:00", "updated_fmt": "2026-09-25 11:00:00",
		},
		Ext:       repo.Row{"row_version": "2", "cpu_count": "2", "form_factor": "rack", "template": "false"},
		Spec:      ci.SpecFor("server"),
		ClassName: "Физический сервер",
	}
}

func tabsData(tab string) tplData {
	d := fakeDetail()
	return tplData{
		"D": d, "CIID": int64(7), "Tab": tab, "HasClassTab": true, "Xsrf": "token", "User": "dev", "Nav": "cis",
		"Envs":          []refdata.Option{{Value: "PROD", Label: "PROD — Промышленная"}},
		"Companies":     []refdata.Option{{Value: "ACME", Label: "ACME"}},
		"Criticalities": criticalities, "Ownerships": ownerships,
		"Transitions": ci.AllowedTransitions("Operational"),
		"Spec":        d.Spec, "Values": d.Ext,
		"Components": []component.Item{
			{Row: repo.Row{"id": "1", "row_version": "1", "type_name": "RAID-контроллер", "slot": "RAID1", "status": "installed", "capacity": ""}, Depth: 0},
			{Row: repo.Row{"id": "2", "row_version": "1", "type_name": "SSD", "slot": "0", "status": "replaced", "capacity": "480 GB"}, Depth: 1},
		},
		"ComponentTypes": []refdata.Option{{Value: "ssd", Label: "SSD (ssd)"}},
		"Interfaces":     []repo.Row{{"id": "3", "name": "eth0", "if_type": "ethernet", "mac": "00:1b:44:11:3a:b7", "wwpn": "", "speed_mbps": "10000", "vlan": "100", "description": ""}},
		"IPs":            []repo.Row{{"id": "4", "row_version": "1", "address": "10.0.0.5", "masklen": "24", "vrf": "default", "kind": "primary", "status": "active", "dns_name": "a.test", "iface": "eth0", "subnet": "10.0.0.0/24"}},
		"IfTypes":        ifTypes, "IPKinds": ipKinds,
		"Relations": []relation.Relation{{ID: 5, Version: 1, RelType: "depends_on", Label: "зависит от", PeerID: 8, PeerName: "TSTQA1-S-01002", PeerClass: "server", ValidFrom: "2026-09-25 10:00", Outgoing: true}},
		"RelTypes":  []refdata.Option{{Value: "depends_on", Label: "зависит от"}},
		"RelGraph":  template.JS(`[{"data":{"id":"ci-7","label":"TSTQA1-S-01001","kind":"self"}}]`),
		"Sources": []repo.Row{{"entity": "ci_server", "attr_name": "bios_uuid", "source_system_code": "vcenter1",
			"source_value": `"11111111-1111-1111-1111-111111111111"`, "observed_at": "2026-09-25 10:00:00", "is_locked": "false", "locked_by": ""}},
		"Changes": []history.ChangeSet{{ID: "9", At: "2026-09-25 10:00:00", Source: "manual", UserID: "1", Reason: "тест", Entries: []history.Entry{
			{Entity: "ci", Op: "I", New: `{"name":"X"}`},
			{Entity: "ci", Op: "U", Attr: "description", Old: `null`, New: `"новое"`},
			{Entity: "ip_address", Op: "D", Old: `{"address":"10.0.0.5/32"}`},
		}}},
	}
}

func TestTemplates_CardTabsRender(t *testing.T) {
	dir := viewsDir(t)
	want := map[string]string{
		"general":    "Жизненный цикл",
		"class":      "Физический сервер",
		"components": "RAID-контроллер",
		"network":    "10.0.0.5/24",
		"relations":  "TSTQA1-S-01002",
		"sources":    "bios_uuid",
		"history":    "Набор изменений №9",
	}
	for tab, marker := range want {
		out := render(t, dir, "ci/_tabs.tpl", tabsData(tab))
		if !strings.Contains(out, marker) {
			t.Errorf("вкладка %s: в HTML нет %q", tab, marker)
		}
		if !strings.Contains(out, `id="card-tabs"`) {
			t.Errorf("вкладка %s: нет контейнера #card-tabs", tab)
		}
		if tab == "relations" && !strings.Contains(out, `"id":"ci-7"`) {
			t.Errorf(`вкладка relations: граф Cytoscape не содержит неэкранированный JSON "id":"ci-7": %s`, out)
		}
	}
	// Пустые списки — ветки {{else}}.
	empty := tabsData("components")
	empty["Components"], empty["Changes"], empty["Relations"], empty["Interfaces"], empty["IPs"], empty["Sources"] =
		[]component.Item(nil), []history.ChangeSet(nil), []relation.Relation(nil), []repo.Row(nil), []repo.Row(nil), []repo.Row(nil)
	for _, tab := range []string{"components", "history", "relations", "network", "sources"} {
		empty["Tab"] = tab
		render(t, dir, "ci/_tabs.tpl", empty)
	}
}

func TestTemplates_CardListNewAndLayout(t *testing.T) {
	dir := viewsDir(t)

	card := tabsData("general")
	card["Created"] = "TSTQA1-S-01001"
	if out := render(t, dir, "ci/card.tpl", card); !strings.Contains(out, "КЕ создана") {
		t.Error("card.tpl: нет баннера создания")
	}

	f := ci.ListFilter{Sort: "name", PerPage: 25, Page: 1}
	list := tplData{
		"Filter":  f,
		"Classes": []refdata.Option{{Value: "server", Label: "Физический сервер"}}, "Locations": []refdata.Option{{Value: "TST-QA1", Label: "TST-QA1 — Тест"}},
		"Envs": []refdata.Option{{Value: "PROD", Label: "PROD"}}, "Companies": []refdata.Option{{Value: "ACME", Label: "ACME"}},
		"Stages": ci.LifecycleStages(), "NamingStatuses": namingStatuses,
		"BulkApplied": "2", "BulkSkipped": "1", "Perms": map[string]bool{"ci.write": true},
	}
	out := render(t, dir, "ci/list.tpl", list)
	for _, marker := range []string{"применено 2, пропущено 1", "ajaxURL", "/static/vendor/tabulator.min.js", "/cis/new"} {
		if !strings.Contains(out, marker) {
			t.Errorf("list.tpl: нет %q", marker)
		}
	}

	bulkPreview := tplData{
		"Field": ci.BulkFieldStage, "Value": "Commissioning",
		"Items": []ci.BulkItem{
			{CIID: 7, Name: "TSTQA1-S-01001", Old: "Planned", New: "Commissioning"},
			{CIID: 8, Name: "TSTQA1-S-01002", Old: "Operational", New: "Commissioning", Skip: true, SkipReason: "переход не разрешён матрицей"},
		},
	}
	out = render(t, dir, "ci/_bulk_preview.tpl", bulkPreview)
	for _, marker := range []string{"TSTQA1-S-01001", "будет применено", "переход не разрешён матрицей", `value="stage"`} {
		if !strings.Contains(out, marker) {
			t.Errorf("_bulk_preview.tpl: нет %q", marker)
		}
	}

	newData := tplData{
		"Specs": ci.ClassSpecs, "Spec": &ci.ClassSpecs[0], "Values": map[string]string{},
		"Locations": list["Locations"], "Envs": list["Envs"], "Companies": []refdata.Option{{Value: "ACME", Label: "ACME"}}, "Criticalities": criticalities,
	}
	render(t, dir, "ci/new.tpl", newData)
	newData["Spec"] = &ci.ClassSpecs[1] // ВМ: есть булево поле
	if out := render(t, dir, "ci/_class_fields.tpl", newData); !strings.Contains(out, `type="checkbox"`) {
		t.Error("_class_fields.tpl: для булева поля нет checkbox")
	}

	page := tplData{"Xsrf": "tok", "User": "dev", "Nav": "cis", "LayoutContent": template.HTML("<p>содержимое</p>"),
		"Perms": map[string]bool{"ci.read": true, "ci.write": true}, "LoggedIn": true, "UserRoles": "editor"}
	out = render(t, dir, "layout.tpl", page)
	// меню скрывает недоступное: у редактора нет ни пользователей, ни журнала, ни импорта
	mustHave := func(marker string, want bool) {
		if strings.Contains(out, marker) != want {
			t.Errorf("layout.tpl: %q присутствует=%v, want %v", marker, !want, want)
		}
	}
	mustHave("Создать КЕ", true)
	mustHave(">Пользователи<", false)
	mustHave(">Журнал изменений<", false)
	mustHave(">Импорт<", false)
	mustHave(`action="/logout"`, true)
	for _, marker := range []string{`"X-Xsrftoken": "tok"`, "<p>содержимое</p>", "/static/vendor/htmx.min.js"} {
		if !strings.Contains(out, marker) {
			t.Errorf("layout.tpl: нет %q", marker)
		}
	}
	render(t, dir, "error.tpl", tplData{"Message": "нет"})
}

func TestTemplates_Compare(t *testing.T) {
	dir := viewsDir(t)
	out := render(t, dir, "ci/_compare.tpl", tplData{"Fields": []history.CompareField{
		{Name: "description", A: "старое", B: "новое", Changed: true},
		{Name: "name", A: "TSTQA1-S-01001", B: "TSTQA1-S-01001", Changed: false},
	}})
	for _, marker := range []string{"старое", "новое", "table-warning"} {
		if !strings.Contains(out, marker) {
			t.Errorf("_compare.tpl: нет %q", marker)
		}
	}
	render(t, dir, "ci/_compare.tpl", tplData{"Fields": []history.CompareField(nil)})
}

func TestTemplates_AuditLog(t *testing.T) {
	dir := viewsDir(t)
	f := history.GlobalFilter{PerPage: 25, Page: 1}
	list := tplData{
		"Filter": f,
		"Result": history.GlobalResult{Rows: []history.GlobalRow{
			{ID: "42", At: "2026-09-25 10:00:00", Source: "manual", UserID: "1", Reason: "тестовое изменение", Count: "3", Entities: "ci, component"},
		}, Total: 1, Page: 1, PerPage: 25, Pages: 1},
		"PageURL":  func(p int) string { return "/changes?page=" + string(rune('0'+p)) },
		"PageNums": []int{1},
		"Sources":  sourceKinds, "Entities": []string{"ci", "component"},
	}
	out := render(t, dir, "audit/list.tpl", list)
	for _, marker := range []string{"тестовое изменение", "Найдено: 1", `hx-get="/changes/42"`} {
		if !strings.Contains(out, marker) {
			t.Errorf("audit/list.tpl: нет %q", marker)
		}
	}
	render(t, dir, "audit/_results.tpl", list)
	render(t, dir, "audit/_results.tpl", tplData{"Filter": f, "Result": history.GlobalResult{Page: 1, PerPage: 25, Pages: 1}, "PageURL": list["PageURL"], "PageNums": []int{1}})

	out = render(t, dir, "audit/_detail.tpl", tplData{"CS": history.ChangeSet{ID: "42", Entries: []history.Entry{
		{Entity: "ci", Op: "U", Attr: "description", Old: `"старое"`, New: `"новое"`},
	}}})
	if !strings.Contains(out, "description") {
		t.Error("audit/_detail.tpl: нет атрибута")
	}
	render(t, dir, "audit/_detail.tpl", tplData{"CS": history.ChangeSet{}})
}

func TestTemplates_NamingSandbox(t *testing.T) {
	dir := viewsDir(t)
	render(t, dir, "admin/naming_sandbox.tpl", tplData{"Name": ""})

	out := render(t, dir, "admin/_naming_result.tpl", tplData{"HasResult": false, "Name": ""})
	if !strings.Contains(out, "Введите имя") {
		t.Error("_naming_result.tpl: нет подсказки для пустого ввода")
	}

	out = render(t, dir, "admin/_naming_result.tpl", tplData{
		"HasResult": true, "Name": "AMSDC1-S-01000",
		"Result": naming.Result{Status: naming.StatusLegacy, Reason: "площадка встречается в примерах Guide 2019", Suggestion: ""},
	})
	for _, marker := range []string{"legacy", "Guide 2019"} {
		if !strings.Contains(out, marker) {
			t.Errorf("_naming_result.tpl (legacy): нет %q", marker)
		}
	}

	out = render(t, dir, "admin/_naming_result.tpl", tplData{
		"HasResult": true, "Name": "webserver01",
		"Result": naming.Result{Status: naming.StatusNoncompliant, Reason: "не соответствует шаблону", Suggestion: "OVBDC1-V-00001"},
	})
	if !strings.Contains(out, "OVBDC1-V-00001") {
		t.Error("_naming_result.tpl (noncompliant): нет предложенного имени")
	}
}

func TestTemplates_LocationTree(t *testing.T) {
	dir := viewsDir(t)
	data := tplData{
		"LocationCode": "TST-QA1", "Location": repo.Row{"location_name": "Тестовая площадка"},
		"Nodes": []refadmin.PlaceNode{
			{Row: repo.Row{"id": "1", "kind": "building", "code": "A", "name": "Корпус A", "rack_units": ""}, Depth: 0},
			{Row: repo.Row{"id": "2", "kind": "room", "code": "101", "name": "", "rack_units": ""}, Depth: 1},
			{Row: repo.Row{"id": "3", "kind": "rack", "code": "R01", "name": "", "rack_units": "42"}, Depth: 2},
		},
	}
	out := render(t, dir, "admin/location_tree.tpl", data)
	for _, marker := range []string{"Тестовая площадка", "Корпус A", "R01", "42"} {
		if !strings.Contains(out, marker) {
			t.Errorf("location_tree.tpl: нет %q", marker)
		}
	}
	render(t, dir, "admin/location_tree.tpl", tplData{"LocationCode": "X", "Location": repo.Row{}, "Nodes": []refadmin.PlaceNode(nil)})
}

func TestTemplates_Import(t *testing.T) {
	dir := viewsDir(t)

	index := tplData{
		"Connectors":    []pipeline.ConnectorRow{{Code: "csv-server", Name: "Загрузка CSV/XLSX — серверы", Kind: "csv", CIClass: "server", Enabled: true}},
		"APIConnectors": []pipeline.ConnectorRow{{Code: "zv-1", Name: "zVirt пилот", Kind: "zvirt", ScheduleSec: "3600", LastJob: "succeeded", LastJobAt: "2026-09-26 10:00:00"}},
		"Jobs": []pipeline.JobRow{
			{ID: "3", Connector: "zVirt пилот", Trigger: "manual", Status: "queued", Attempts: "0", Max: "3", RunAfter: "2026-09-26 10:01:00", Cancelable: true},
			{ID: "2", Connector: "zVirt пилот", Trigger: "schedule", Status: "failed", Attempts: "3", Max: "3", Error: "источник недоступен", RunID: "9"},
		},
		"Runs":          []pipeline.RunRow{{ID: "1", ConnectorName: "Загрузка CSV/XLSX — серверы", Status: "succeeded", DryRun: false, StartedAt: "2026-09-26 10:00:00", Total: "3", Created: "2", Updated: "1"}},
		"OpenConflicts": 2,
	}
	out := render(t, dir, "import/index.tpl", index)
	for _, marker := range []string{"Загрузка CSV/XLSX — серверы", "/import/template?class=server", "Подробнее",
		"/import/connectors/zv-1/run", "/import/jobs/3/cancel", "источник недоступен", "в очереди"} {
		if !strings.Contains(out, marker) {
			t.Errorf("import/index.tpl: нет %q", marker)
		}
	}
	render(t, dir, "import/index.tpl", tplData{"Connectors": []pipeline.ConnectorRow(nil), "APIConnectors": []pipeline.ConnectorRow(nil),
		"Jobs": []pipeline.JobRow(nil), "Runs": []pipeline.RunRow(nil), "OpenConflicts": 0})

	out = render(t, dir, "import/run.tpl", tplData{"Run": pipeline.RunRow{
		ID: "7", ConnectorName: "Загрузка CSV/XLSX — серверы", Status: "succeeded", DryRun: false,
		Total: "10", Created: "4", Updated: "6", Unchanged: "0", Conflicts: "0", Errors: "0", NotDiscovered: "0", ChangeSetID: "99",
	}})
	if !strings.Contains(out, "Прогон №7") {
		t.Error("import/run.tpl: нет номера прогона")
	}
	render(t, dir, "import/run.tpl", tplData{"Run": pipeline.RunRow{ID: "8", Status: "failed", ErrorMessage: "коннектор не найден"}})

	f := pipeline.ConflictFilter{Status: "open", PerPage: 25, Page: 1}
	conflicts := tplData{
		"Filter": f,
		"Result": pipeline.ConflictResult{Rows: []pipeline.ConflictRow{
			{ID: "5", RunID: "7", CIID: "3", CIName: "TST-QA1-S-00003", Kind: "locked_attr", Details: `{"attrs":["ci_server.ram_gb"]}`, Status: "open", CreatedAt: "2026-09-26 10:05:00"},
			{ID: "6", RunID: "7", Kind: "not_discovered", Details: `{}`, Status: "open", CreatedAt: "2026-09-26 10:05:01"},
		}, Total: 2, Page: 1, PerPage: 25, Pages: 1},
	}
	out = render(t, dir, "import/conflicts.tpl", conflicts)
	for _, marker := range []string{"TST-QA1-S-00003", "Заблокированный атрибут", "Не найдена источником"} {
		if !strings.Contains(out, marker) {
			t.Errorf("import/conflicts.tpl: нет %q", marker)
		}
	}
	render(t, dir, "import/_conflicts_results.tpl", conflicts)
	render(t, dir, "import/_conflicts_results.tpl", tplData{"Filter": f, "Result": pipeline.ConflictResult{Page: 1, PerPage: 25, Pages: 1}})
}

func TestTemplates_BaselinesAndPeriods(t *testing.T) {
	dir := viewsDir(t)
	b := baseline.Baseline{ID: "3", Name: "Срез на 01.10", AsOf: "2026-10-01 00:00:00", Status: "frozen", ItemCount: "42",
		Checksum: strings.Repeat("ab", 32), CreatedAt: "2026-10-02 10:00", CreatedBy: "1", FrozenAt: "2026-10-02 10:05", FrozenBy: "1"}
	out := render(t, dir, "baselines/list.tpl", tplData{"Baselines": []baseline.Baseline{b, {ID: "4", Name: "Черновик", Status: "draft"}}, "NowInput": "2026-10-02T10:00"})
	for _, m := range []string{"Срез на 01.10", "заморожен", "черновик", "abababababab…"} {
		if !strings.Contains(out, m) {
			t.Errorf("baselines/list.tpl: нет %q", m)
		}
	}
	render(t, dir, "baselines/list.tpl", tplData{"Baselines": []baseline.Baseline(nil), "NowInput": ""})

	show := tplData{"B": b, "Scope": baseline.Scope{Classes: []string{"server", "vm"}, LocationCode: "OVB-DC1"},
		"ByEntity": []struct {
			Entity string
			N      int
		}{{"ci", 40}, {"component", 2}}, "Others": []baseline.Baseline{b}, "Formats": baseline.Formats}
	out = render(t, dir, "baselines/show.tpl", show)
	for _, m := range []string{"Проверить целостность", "/baselines/3/export/xlsx", "Утвердить", "классы: server, vm", "OVB-DC1"} {
		if !strings.Contains(out, m) {
			t.Errorf("baselines/show.tpl: нет %q", m)
		}
	}
	b.Status = "draft"
	show["B"] = b
	out = render(t, dir, "baselines/show.tpl", show)
	if !strings.Contains(out, "Заморозить") || strings.Contains(out, "export/csv") {
		t.Error("baselines/show.tpl: у черновика должна быть кнопка заморозки и не должно быть выгрузки")
	}

	res := baseline.Result{LabelA: "Срез №3", LabelB: "Текущее состояние", Added: 1, Changed: 1,
		Summary: []baseline.SummaryRow{{Entity: "ci", Added: 1, Changed: 1}},
		Changes: []baseline.Change{
			{Entity: "ci", EntityID: 7, CIID: 7, CIName: "TSTQA1-S-00007", Kind: "changed", Fields: []baseline.FieldDiff{{Name: "description", A: "", B: "новое"}}},
			{Entity: "ci", EntityID: 8, CIID: 8, CIName: "TSTQA1-S-00008", Kind: "added", BVersion: 1}}, Truncated: true}
	out = render(t, dir, "baselines/compare.tpl", tplData{"Res": res, "A": "b:3", "B": "now", "Scope": baseline.Scope{}})
	for _, m := range []string{"TSTQA1-S-00007", "description", "новое", "добавлено: 1", "первые различия"} {
		if !strings.Contains(out, m) {
			t.Errorf("baselines/compare.tpl: нет %q", m)
		}
	}
	render(t, dir, "baselines/compare.tpl", tplData{"Res": baseline.Result{}, "A": "now", "B": "now", "Scope": baseline.Scope{}})

	out = render(t, dir, "periods/list.tpl", tplData{
		"Periods": []period.Period{{Month: "2026-09", Status: "closed", ClosedAt: "2026-10-01 09:00", CloseReason: "закрытие"}},
		"Report":  []period.Backdated{{ChangeSetID: "12", EffectiveAt: "2026-09-15 00:00", StartedAt: "2026-10-05 10:00", Reason: "корректировка", ClosedPeriod: true}},
		"From":    "2026-09-01", "To": "2026-10-05", "LastMonth": "2026-09"})
	for _, m := range []string{"2026-09", "закрыт", "корректировка", "закрытый период"} {
		if !strings.Contains(out, m) {
			t.Errorf("periods/list.tpl: нет %q", m)
		}
	}
	render(t, dir, "periods/list.tpl", tplData{"Periods": []period.Period(nil), "Report": []period.Backdated(nil), "From": "", "To": "", "LastMonth": ""})
}

func TestTriggerHeaderIsASCII(t *testing.T) {
	c := &BaseController{}
	_ = c // заголовок формируется в trigger(); проверяем экранирование отдельной функцией ниже
	got := asciiEscape(`{"toast":{"text":"Сохранено 😀"}}`)
	for _, r := range got {
		if r > 126 {
			t.Fatalf("в заголовке остался не-ASCII символ %q: %s", r, got)
		}
	}
	if !strings.Contains(got, "\\u0421") || !strings.Contains(got, "\\ud83d\\ude00") {
		t.Errorf("ожидались \\u-последовательности, получено %s", got)
	}
}
