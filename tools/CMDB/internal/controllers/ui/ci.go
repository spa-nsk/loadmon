package ui

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"cmdb/internal/platform/asof"
	"cmdb/internal/repo"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/component"
	"cmdb/internal/services/history"
	"cmdb/internal/services/network"
	"cmdb/internal/services/provenance"
	"cmdb/internal/services/refdata"
	"cmdb/internal/services/relation"
	"cmdb/internal/services/software"
	"cmdb/internal/services/svcerr"
)

// exportColumns/exportColumnKeys — заголовки и ключи repo.Row для выгрузки списка КЕ в CSV/XLSX
// (E3.3), в порядке колонок ci/_results.tpl.
var (
	exportColumns    = []string{"Имя", "Класс", "Площадка", "Окружение", "Стадия", "Статус имени", "Серийный №", "Изменена (UTC)"}
	exportColumnKeys = []string{"name", "class_name", "location_code", "environment_code", "lifecycle_stage", "naming_status", "serial_number", "updated_at"}
)

// CIController — список КЕ, создание и карточка с вкладками.
type CIController struct {
	BaseController
}

var (
	criticalities  = []string{"low", "medium", "high", "critical"}
	ownerships     = []string{"own", "leased", "retained", "cloud"}
	namingStatuses = []string{"compliant", "legacy", "noncompliant", "exception"}
	ifTypes        = []string{"ethernet", "fc", "ib", "bmc", "virtual", "lag", "loopback"}
	ipKinds        = []string{"primary", "secondary", "vip", "mgmt", "bmc", "service"}
)

func (c *CIController) ctx() context.Context { return c.Ctx.Request.Context() }

// must прерывает запрос с сообщением об ошибке, если err != nil.
func (c *CIController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

// Home перенаправляет на список КЕ.
func (c *CIController) Home() { c.redirect("/cis") }

// ---------- список ----------

func (c *CIController) listFilter() ci.ListFilter {
	f := ci.ListFilter{
		Class: c.GetString("class"), Location: c.GetString("location"), Env: c.GetString("env"),
		Stage: c.GetString("stage"), NamingStatus: c.GetString("naming"), Q: c.GetString("q"),
		Sort: c.GetString("sort", "name"), Desc: c.GetString("dir") == "desc",
	}
	f.Page, _ = c.GetInt("page", 1)
	f.PerPage, _ = c.GetInt("per", ci.PerPageOptions[0])
	valid := false
	for _, p := range ci.PerPageOptions {
		valid = valid || p == f.PerPage
	}
	if !valid {
		f.PerPage = ci.PerPageOptions[0]
	}
	return f
}

// queryValues — текущие фильтры как параметры URL (для ссылок сортировки и пагинации).
func queryValues(f ci.ListFilter) url.Values {
	v := url.Values{}
	set := func(k, val string) {
		if val != "" {
			v.Set(k, val)
		}
	}
	set("class", f.Class)
	set("location", f.Location)
	set("env", f.Env)
	set("stage", f.Stage)
	set("naming", f.NamingStatus)
	set("q", f.Q)
	set("sort", f.Sort)
	if f.Desc {
		v.Set("dir", "desc")
	}
	if f.PerPage != ci.PerPageOptions[0] {
		v.Set("per", strconv.Itoa(f.PerPage))
	}
	return v
}

// List отдаёт страницу списка: фильтр-бар (сразу заполнен из query-параметров — открытая по
// ссылке страница с фильтром выглядит как отфильтрованная) и пустой грид Tabulator, который сам
// запрашивает данные у Data() при загрузке и при каждом изменении фильтра (E3.1).
func (c *CIController) List() {
	f := c.listFilter()
	var err error
	c.Data["Filter"] = f
	c.Data["Nav"] = "cis"
	c.Data["Classes"], err = refdata.Classes(c.ctx())
	c.must(err)
	c.Data["Locations"], err = refdata.Locations(c.ctx())
	c.must(err)
	c.Data["Envs"], err = refdata.Environments(c.ctx())
	c.must(err)
	c.Data["Companies"], err = refdata.Companies(c.ctx())
	c.must(err)
	c.Data["Stages"] = ci.LifecycleStages()
	c.Data["NamingStatuses"] = namingStatuses
	c.Data["BulkApplied"] = c.GetString("bulk_applied")
	c.Data["BulkSkipped"] = c.GetString("bulk_skipped")
	c.page("ci/list.tpl")
}

// GridData отдаёт страницу списка КЕ в JSON — источник данных Tabulator (remote-пагинация/
// сортировка, параметры запроса — те же поля фильтра, что и у полной страницы, см. listFilter).
// Названо не Data — это имя уже занято полем web.Controller.Data (данные для шаблона). Формат
// ответа — {data, last_page} — соответствует контракту ajaxRequestFunc в ci/list.tpl.
func (c *CIController) GridData() {
	f := c.listFilter()
	res, err := ci.List(c.ctx(), f)
	c.must(err)
	c.Data["json"] = map[string]any{"data": res.Rows, "last_page": res.Pages, "total": res.Total}
	_ = c.ServeJSON()
}

// exportFilename — имя файла выгрузки списка с отметкой времени (UTC), чтобы не путать выгрузки.
func exportFilename(ext string) string {
	return "cis-" + time.Now().UTC().Format("20060102-1504") + "." + ext
}

// ExportCSV выгружает список КЕ по текущему фильтру (без пагинации, E3.3) в CSV.
func (c *CIController) ExportCSV() {
	f := c.listFilter()
	rows, err := ci.ExportRows(c.ctx(), f)
	c.must(err)
	c.logEvent("ci_export", map[string]any{"format": "csv", "rows": len(rows), "filter": f})
	c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", `attachment; filename="`+exportFilename("csv")+`"`)
	w := csv.NewWriter(c.Ctx.ResponseWriter)
	_ = w.Write(exportColumns)
	for _, r := range rows {
		rec := make([]string, len(exportColumns))
		for i, col := range exportColumnKeys {
			rec[i] = r[col]
		}
		_ = w.Write(rec)
	}
	w.Flush()
	c.StopRun()
}

// ExportXLSX — то же самое в формате XLSX (xuri/excelize).
func (c *CIController) ExportXLSX() {
	filter := c.listFilter()
	rows, err := ci.ExportRows(c.ctx(), filter)
	c.must(err)
	c.logEvent("ci_export", map[string]any{"format": "xlsx", "rows": len(rows), "filter": filter})

	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Sheet1"
	for i, h := range exportColumns {
		cellRef, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellStr(sheet, cellRef, h)
	}
	for r, row := range rows {
		for i, col := range exportColumnKeys {
			cellRef, _ := excelize.CoordinatesToCellName(i+1, r+2)
			_ = f.SetCellStr(sheet, cellRef, row[col])
		}
	}

	c.Ctx.Output.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Ctx.Output.Header("Content-Disposition", `attachment; filename="`+exportFilename("xlsx")+`"`)
	c.must(f.Write(c.Ctx.ResponseWriter))
	c.StopRun()
}

// ---------- массовые операции (E3.6) ----------

// bulkParams читает из формы поле/значение/список id для массовой операции.
func (c *CIController) bulkParams() (field ci.BulkField, value string, ids []int64) {
	_ = c.Ctx.Request.ParseForm()
	field = ci.BulkField(c.GetString("bulk_field"))
	switch field {
	case ci.BulkFieldStage:
		value = c.GetString("bulk_stage")
	case ci.BulkFieldCompany:
		value = c.GetString("bulk_company")
	case ci.BulkFieldLocation:
		value = c.GetString("bulk_location")
	}
	for _, raw := range c.Ctx.Request.PostForm["ids"] {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return field, value, ids
}

// BulkPreview показывает, что изменится при массовой операции, без записи в БД.
func (c *CIController) BulkPreview() {
	field, value, ids := c.bulkParams()
	if len(ids) == 0 {
		c.failText("Выберите хотя бы одну КЕ (отметьте строки в списке).")
	}
	items, err := ci.BulkPreview(c.ctx(), field, ids, value)
	c.must(err)

	c.Data["Field"] = field
	c.Data["Value"] = value
	c.Data["Items"] = items
	c.fragment("ci/_bulk_preview.tpl")
}

// BulkApply применяет массовую операцию (одна причина -> один change set) и возвращает к списку.
// Итог передаётся через query-параметры (как /cis/:id?created=… у карточки, а не тостом перед
// HX-Redirect — тост не успевает показаться до навигации).
func (c *CIController) BulkApply() {
	field, value, ids := c.bulkParams()
	c.requireBulkWrite(ids)
	reason := c.reason()
	res, err := ci.BulkApply(c.ctx(), field, ids, value, reason, c.userID())
	c.must(err)
	c.redirect(fmt.Sprintf("/cis?bulk_applied=%d&bulk_skipped=%d", res.Applied, res.Skipped))
}

// ---------- создание ----------

// New — форма создания КЕ (выбор класса, площадки, атрибутов класса).
func (c *CIController) New() {
	spec := c.specFromQuery()
	var err error
	c.Data["Nav"] = "new"
	c.Data["Specs"] = ci.ClassSpecs
	c.Data["Spec"] = spec
	c.Data["Values"] = map[string]string{}
	c.Data["Locations"], err = refdata.Locations(c.ctx())
	c.must(err)
	c.Data["Envs"], err = refdata.Environments(c.ctx())
	c.must(err)
	c.Data["Companies"], err = refdata.Companies(c.ctx())
	c.must(err)
	c.Data["Criticalities"] = criticalities
	c.page("ci/new.tpl")
}

func (c *CIController) specFromQuery() *ci.ClassSpec {
	if spec := ci.SpecFor(c.GetString("class")); spec != nil {
		return spec
	}
	return &ci.ClassSpecs[0]
}

// NewFields — поля выбранного класса (подгружаются при смене класса в форме создания).
func (c *CIController) NewFields() {
	c.Data["Spec"] = c.specFromQuery()
	c.Data["Values"] = map[string]string{}
	c.fragment("ci/_class_fields.tpl")
}

// NamePreview — подсказка имени по профилю до сохранения (номер выдаётся при создании).
func (c *CIController) NamePreview() {
	class, loc := c.GetString("class"), c.GetString("location_code")
	text := "выберите площадку"
	if des, ok := registry.ByClass(class); ok && loc != "" {
		row, err := repo.QueryRow(c.ctx(), `SELECT location_code_c FROM ref.location WHERE location_code = ?`, loc)
		if err == nil && row != nil {
			text = fmt.Sprintf("%s-%s-%s", row["location_code_c"], des.Code, strings.Repeat("N", des.SeqLength))
		}
	}
	c.Ctx.WriteString(text)
}

// Create создаёт КЕ и переходит на её карточку.
func (c *CIController) Create() {
	spec := ci.SpecFor(c.GetString("class"))
	if spec == nil {
		c.failText("Выберите класс КЕ.")
	}
	c.requireWriteClass(spec.Class)
	values := map[string]string{}
	for _, f := range spec.Fields {
		values[f.Name] = c.GetString(f.Name)
	}
	attrs, fieldErrs := spec.ParseAttrs(values)
	if len(fieldErrs) > 0 {
		c.failText(joinErrors(fieldErrs), fieldErrs)
	}
	if c.GetString("location_code") == "" {
		c.failText("Выберите площадку.", map[string]string{"location_code": "Обязательное поле"})
	}
	reason := c.reason()

	res, err := ci.Create(c.ctx(), registry, ci.CreateInput{
		Class: spec.Class, LocationCode: c.GetString("location_code"), CompanyCode: c.GetString("company_code"),
		EnvironmentCode: c.GetString("environment_code"), SerialNumber: strings.TrimSpace(c.GetString("serial_number")),
		Description: strings.TrimSpace(c.GetString("description")), Criticality: c.GetString("criticality"),
		Attrs: attrs, Reason: reason, UserID: c.userID(),
	})
	c.must(err)
	c.redirect(fmt.Sprintf("/cis/%d?created=%s", res.CIID, url.QueryEscape(res.Name)))
}

func joinErrors(m map[string]string) string {
	var parts []string
	for _, v := range m {
		parts = append(parts, v)
	}
	return strings.Join(parts, "; ")
}

// ---------- карточка ----------

func (c *CIController) id() int64 {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	return id
}

// Card — карточка КЕ (шапка + вкладки; открывается вкладка «Общие»).
func (c *CIController) Card() {
	d, err := ci.Get(c.ctx(), c.id())
	if errors.Is(err, svcerr.ErrNotFound) {
		c.notFoundPage("КЕ не найдена")
		return
	}
	c.must(err)
	c.logEvent("ci_view", map[string]any{"ci_id": c.id(), "name": d.CI["name"]})
	c.Data["Nav"] = "cis"
	c.Data["Created"] = c.GetString("created")
	c.loadTab(d, "general")
	c.page("ci/card.tpl")
}

// Tab — фрагмент вкладок с выбранной вкладкой.
func (c *CIController) Tab() { c.renderTabs(c.id(), c.Ctx.Input.Param(":tab")) }

// Compare — сравнение двух версий КЕ по id наборов изменений (E3.7).
func (c *CIController) Compare() {
	id := c.id()
	a, errA := strconv.ParseInt(c.GetString("a"), 10, 64)
	b, errB := strconv.ParseInt(c.GetString("b"), 10, 64)
	if errA != nil || errB != nil {
		c.failText("Выберите обе версии для сравнения.")
	}
	fields, err := history.Compare(c.ctx(), id, a, b)
	c.must(err)
	c.Data["Fields"] = fields
	c.fragment("ci/_compare.tpl")
}

// renderTabs загружает данные вкладки и рендерит блок вкладок (ответ на GET и на успешные POST).
func (c *CIController) renderTabs(id int64, tab string) {
	d, err := ci.Get(c.ctx(), id)
	c.must(err)
	c.loadTab(d, tab)
	c.fragment("ci/_tabs.tpl")
}

func (c *CIController) loadTab(d *ci.Detail, tab string) {
	ctx := c.ctx()
	id, _ := strconv.ParseInt(d.CI["id"], 10, 64)
	if tab == "class" && d.Spec == nil {
		tab = "general"
	}
	c.Data["D"] = d
	c.Data["CIID"] = id
	c.Data["Tab"] = tab
	c.Data["HasClassTab"] = d.Spec != nil

	var err error
	switch tab {
	case "class":
		c.Data["Spec"] = d.Spec
		c.Data["Values"] = d.Ext
	case "components":
		c.Data["Components"], err = component.ListForCI(ctx, id)
		c.must(err)
		c.Data["ComponentTypes"], err = refdata.ComponentTypes(ctx)
	case "software":
		c.Data["Installs"], err = software.ListForCI(ctx, id)
		c.must(err)
		c.Data["SoftwareOptions"], err = refdata.Software(ctx)
	case "network":
		c.Data["Interfaces"], err = network.ListInterfaces(ctx, id)
		c.must(err)
		c.Data["IPs"], err = network.ListIPs(ctx, id)
		c.must(err)
		c.Data["IfTypes"], c.Data["IPKinds"] = ifTypes, ipKinds
	case "relations":
		var rels []relation.Relation
		var at *time.Time
		if t, ok := asof.FromContext(ctx); ok {
			at = &t // «показать на дату» (E6.1): связи и граф — как было на момент
		}
		rels, err = relation.ListForCI(ctx, id, at)
		c.must(err)
		c.Data["Relations"] = rels
		c.Data["RelTypes"], err = refdata.RelTypes(ctx)
		c.Data["RelGraph"] = relGraphJSON(d.CI["id"], d.CI["name"], rels)
	case "sources":
		extTable := ""
		if d.Spec != nil {
			extTable = d.Spec.ExtTable
		}
		c.Data["Sources"], err = provenance.ListForCI(ctx, id, extTable)
	case "history":
		c.Data["Changes"], err = history.ForCI(ctx, id, 50)
	default:
		c.Data["Tab"] = "general"
		c.Data["Envs"], err = refdata.Environments(ctx)
		c.must(err)
		c.Data["Companies"], err = refdata.Companies(ctx)
		c.Data["Criticalities"], c.Data["Ownerships"] = criticalities, ownerships
		c.Data["Transitions"] = ci.AllowedTransitions(d.CI["lifecycle_stage"])
	}
	c.must(err)
}

// ---------- правка общих полей и атрибутов класса ----------

func (c *CIController) Update() {
	_ = c.Ctx.Request.ParseForm()
	id := c.id()
	c.requireCIWrite(id)
	d, err := ci.Get(c.ctx(), id)
	c.must(err)
	reason := c.reason()
	version, _ := c.GetInt("row_version")

	in := ci.UpdateInput{CIID: id, ExpectedVersion: version, Reason: reason, UserID: c.userID()}
	changed := false
	// sp возвращает указатель на новое значение, только если оно отличается от текущего.
	sp := func(field string) *string {
		if _, sent := c.Ctx.Request.PostForm[field]; !sent {
			return nil
		}
		v := strings.TrimSpace(c.GetString(field))
		if v == d.CI[field] {
			return nil
		}
		changed = true
		return &v
	}
	in.Description = sp("description")
	in.Criticality = sp("criticality")
	in.EnvironmentCode = sp("environment_code")
	in.SerialNumber = sp("serial_number")
	in.AssetTag = sp("asset_tag")
	in.RackPosition = sp("rack_position")
	in.Ownership = sp("ownership")
	in.ServiceProvider = sp("service_provider")
	in.KIICategory = sp("kii_category")
	in.LifecycleStatus = sp("lifecycle_status")
	if c.GetString("kii_object_present") == "1" {
		v := c.GetString("kii_object") == "1"
		if v != (d.CI["kii_object"] == "true") {
			in.KIIObject, changed = &v, true
		}
	}

	if !changed {
		c.trigger(map[string]any{"toast": toastPayload{Kind: "info", Text: "Изменений нет."}})
		c.Ctx.Output.Header("HX-Reswap", "none")
		c.Ctx.WriteString("")
		c.StopRun()
	}
	res, err := ci.Update(c.ctx(), in)
	c.must(err)
	c.ok(fmt.Sprintf("Сохранено (версия %d).", res.RowVersion))
	c.renderTabs(id, "general")
}

func (c *CIController) UpdateExt() {
	id := c.id()
	c.requireCIWrite(id)
	d, err := ci.Get(c.ctx(), id)
	c.must(err)
	if d.Spec == nil {
		c.failText("Для этого класса нет атрибутов.")
	}
	values := map[string]string{}
	for _, f := range d.Spec.Fields {
		values[f.Name] = c.GetString(f.Name)
	}
	attrs, fieldErrs := d.Spec.ParseAttrs(values)
	if len(fieldErrs) > 0 {
		c.failText(joinErrors(fieldErrs), fieldErrs)
	}
	reason := c.reason()
	in := ci.ExtensionInput{CIID: id, Table: d.Spec.ExtTable, Attrs: attrs, Reason: reason, UserID: c.userID()}
	if v, err := strconv.Atoi(c.GetString("row_version")); err == nil {
		in.ExpectedVersion = &v
	}
	res, err := ci.UpsertExtension(c.ctx(), in)
	c.must(err)
	if res.Changed {
		c.ok(fmt.Sprintf("Атрибуты сохранены (версия %d).", res.RowVersion))
	} else {
		c.trigger(map[string]any{"toast": toastPayload{Kind: "info", Text: "Изменений нет."}})
	}
	c.renderTabs(id, "class")
}

// Lifecycle — смена стадии жизненного цикла (шапка карточки меняется — полная перезагрузка).
func (c *CIController) Lifecycle() {
	id := c.id()
	c.requireCIWrite(id)
	reason := c.reason()
	version, _ := c.GetInt("row_version")
	_, err := ci.ChangeLifecycle(c.ctx(), ci.ChangeLifecycleInput{
		CIID: id, ExpectedVersion: version, ToStage: c.GetString("to_stage"),
		Status: strings.TrimSpace(c.GetString("status")), Reason: reason, UserID: c.userID(),
	})
	c.must(err)
	c.redirect(fmt.Sprintf("/cis/%d", id))
}

// ---------- компоненты ----------

func specFromForm(c *CIController) (component.Spec, map[string]string) {
	s := component.Spec{
		Manufacturer: strings.TrimSpace(c.GetString("manufacturer")), PartNumber: strings.TrimSpace(c.GetString("part_number")),
		SerialNumber: strings.TrimSpace(c.GetString("serial_number")), Firmware: strings.TrimSpace(c.GetString("firmware")),
		CapacityUnit: strings.TrimSpace(c.GetString("capacity_unit")), Speed: strings.TrimSpace(c.GetString("speed")),
	}
	errs := map[string]string{}
	if raw := strings.TrimSpace(c.GetString("capacity_value")); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			errs["capacity_value"] = "Ожидается число"
		}
		s.CapacityValue = &v
	}
	return s, errs
}

func (c *CIController) AddComponent() {
	id := c.id()
	c.requireCIWrite(id)
	spec, errs := specFromForm(c)
	if len(errs) > 0 {
		c.failText(joinErrors(errs), errs)
	}
	in := component.AddInput{CIID: id, ComponentType: c.GetString("component_type"), Slot: strings.TrimSpace(c.GetString("slot")),
		Spec: spec, Reason: c.reason(), UserID: c.userID()}
	if in.ComponentType == "" {
		c.failText("Выберите тип компонента.", map[string]string{"component_type": "Обязательное поле"})
	}
	if p, err := strconv.ParseInt(c.GetString("parent_id"), 10, 64); err == nil {
		in.ParentComponentID = &p
	}
	_, _, err := component.Add(c.ctx(), in)
	c.must(err)
	c.ok("Компонент добавлен.")
	c.renderTabs(id, "components")
}

func (c *CIController) ReplaceComponent() {
	cid, _ := strconv.ParseInt(c.Ctx.Input.Param(":cid"), 10, 64)
	c.requireComponentWrite(cid)
	ciID, _ := strconv.ParseInt(c.GetString("ci"), 10, 64)
	spec, errs := specFromForm(c)
	if len(errs) > 0 {
		c.failText(joinErrors(errs), errs)
	}
	version, _ := c.GetInt("row_version")
	_, err := component.Replace(c.ctx(), component.ReplaceInput{ComponentID: cid, ExpectedVersion: version, New: spec,
		Reason: c.reason(), UserID: c.userID()})
	c.must(err)
	c.ok("Компонент заменён.")
	c.renderTabs(ciID, "components")
}

// ---------- ПО ----------

// AddSoftware регистрирует установку ПО на КЕ (plan.md E7.7, вкладка «ПО» карточки).
func (c *CIController) AddSoftware() {
	id := c.id()
	c.requireCIWrite(id)
	swID, err := strconv.ParseInt(c.GetString("software_id"), 10, 64)
	if err != nil || swID == 0 {
		c.failText("Выберите ПО из справочника.", map[string]string{"software_id": "Обязательное поле"})
	}
	_, _, err = software.Add(c.ctx(), software.AddInput{
		CIID: id, SoftwareID: swID, Version: strings.TrimSpace(c.GetString("version")),
		InstallPath: strings.TrimSpace(c.GetString("install_path")), Reason: c.reason(), UserID: c.userID(),
	})
	c.must(err)
	c.ok("Установка ПО зарегистрирована.")
	c.renderTabs(id, "software")
}

// RemoveSoftware снимает запись об установке ПО.
func (c *CIController) RemoveSoftware() {
	instID, _ := strconv.ParseInt(c.Ctx.Input.Param(":sid"), 10, 64)
	ciID, _ := strconv.ParseInt(c.GetString("ci"), 10, 64)
	c.requireCIWrite(ciID)
	version, _ := c.GetInt("row_version")
	_, err := software.Remove(c.ctx(), software.RemoveInput{ID: instID, ExpectedVersion: version, Reason: c.reason(), UserID: c.userID()})
	c.must(err)
	c.ok("Установка ПО снята.")
	c.renderTabs(ciID, "software")
}

// ---------- сеть ----------

func (c *CIController) AddInterface() {
	id := c.id()
	c.requireCIWrite(id)
	in := network.InterfaceInput{CIID: id, Name: strings.TrimSpace(c.GetString("name")), IfType: c.GetString("if_type"),
		MAC: strings.TrimSpace(c.GetString("mac")), WWPN: strings.TrimSpace(c.GetString("wwpn")),
		Meta: network.Meta{Reason: c.reason(), UserID: c.userID()}}
	if in.Name == "" {
		c.failText("Укажите имя интерфейса.", map[string]string{"name": "Обязательное поле"})
	}
	if raw := strings.TrimSpace(c.GetString("speed_mbps")); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v <= 0 {
			c.failText("Скорость: ожидается положительное целое (Мбит/с).", map[string]string{"speed_mbps": "Ожидается целое число"})
		}
		in.SpeedMbps = &v
	}
	_, _, err := network.AddInterface(c.ctx(), in)
	c.must(err)
	c.ok("Интерфейс добавлен.")
	c.renderTabs(id, "network")
}

func (c *CIController) AssignIP() {
	id := c.id()
	c.requireCIWrite(id)
	in := network.AssignIPInput{CIID: id, Address: c.GetString("address"), VRF: strings.TrimSpace(c.GetString("vrf")),
		Kind: c.GetString("kind"), DNSName: strings.TrimSpace(c.GetString("dns_name")),
		Meta: network.Meta{Reason: c.reason(), UserID: c.userID()}}
	if v, err := strconv.ParseInt(c.GetString("interface_id"), 10, 64); err == nil {
		in.InterfaceID = &v
	}
	res, err := network.AssignIP(c.ctx(), in)
	c.must(err)
	if res.SubnetID == 0 {
		c.ok("Адрес назначен (подсеть не найдена).")
	} else {
		c.ok("Адрес назначен.")
	}
	c.renderTabs(id, "network")
}

func (c *CIController) ReleaseIP() {
	ipID, _ := strconv.ParseInt(c.Ctx.Input.Param(":ipid"), 10, 64)
	c.requireIPWrite(ipID)
	ciID, _ := strconv.ParseInt(c.GetString("ci"), 10, 64)
	version, _ := c.GetInt("row_version")
	_, err := network.ReleaseIP(c.ctx(), ipID, version, network.Meta{Reason: c.reason(), UserID: c.userID()})
	c.must(err)
	c.ok("Адрес освобождён.")
	c.renderTabs(ciID, "network")
}

// ---------- связи ----------

func (c *CIController) AddRelation() {
	id := c.id()
	c.requireCIWrite(id)
	reason := c.reason()
	peerName := strings.TrimSpace(c.GetString("peer_name"))
	if peerName == "" {
		c.failText("Укажите имя связанной КЕ.", map[string]string{"peer_name": "Обязательное поле"})
	}
	peerID, err := ci.FindByName(c.ctx(), peerName)
	if errors.Is(err, svcerr.ErrNotFound) {
		c.failText(fmt.Sprintf("КЕ %q не найдена среди действующих.", peerName), map[string]string{"peer_name": "Не найдена"})
	}
	c.must(err)

	in := relation.CreateInput{RelType: c.GetString("rel_type"), SrcCIID: id, DstCIID: peerID, Reason: reason, UserID: c.userID()}
	if c.GetString("direction") == "in" {
		in.SrcCIID, in.DstCIID = peerID, id
	}
	_, _, err = relation.Create(c.ctx(), in)
	c.must(err)
	c.ok("Связь создана.")
	c.renderTabs(id, "relations")
}

func (c *CIController) CloseRelation() {
	rid, _ := strconv.ParseInt(c.Ctx.Input.Param(":rid"), 10, 64)
	c.requireRelationWrite(rid)
	ciID, _ := strconv.ParseInt(c.GetString("ci"), 10, 64)
	version, _ := c.GetInt("row_version")
	_, err := relation.Close(c.ctx(), relation.CloseInput{RelID: rid, ExpectedVersion: version, Reason: c.reason(), UserID: c.userID()})
	c.must(err)
	c.ok("Связь закрыта.")
	c.renderTabs(ciID, "relations")
}

// relGraphNode — один элемент Cytoscape (узел или ребро): та же форма {data:{...}} для обоих,
// у ребра дополнительно заполнены Source/Target (E3.4).
type relGraphNode struct {
	Data map[string]any `json:"data"`
}

// relGraphJSON строит элементы графа связей КЕ для вкладки «Связи»: только 1 шаг от текущей КЕ
// (её прямые связи, уже загруженные для таблицы на той же вкладке) — обход графа на несколько
// шагов потребовал бы отдельного backend-запроса и не входит в этот срез задачи. template.JS —
// чтобы html/template не экранировал кавычки JSON при вставке в <script>.
func relGraphJSON(selfID, selfName string, rels []relation.Relation) template.JS {
	elements := []relGraphNode{{Data: map[string]any{"id": "ci-" + selfID, "label": selfName, "kind": "self", "url": "/cis/" + selfID}}}
	seenPeer := map[int64]bool{}
	for _, r := range rels {
		peerID := strconv.FormatInt(r.PeerID, 10)
		peerKey := "ci-" + peerID
		if !seenPeer[r.PeerID] {
			seenPeer[r.PeerID] = true
			elements = append(elements, relGraphNode{Data: map[string]any{"id": peerKey, "label": r.PeerName, "kind": "peer", "url": "/cis/" + peerID}})
		}
		src, dst := "ci-"+selfID, peerKey
		if !r.Outgoing {
			src, dst = peerKey, "ci-"+selfID
		}
		elements = append(elements, relGraphNode{Data: map[string]any{
			"id": fmt.Sprintf("rel-%d", r.ID), "source": src, "target": dst, "label": r.Label,
		}})
	}
	b, err := json.Marshal(elements)
	if err != nil {
		return template.JS("[]")
	}
	return template.JS(b)
}
