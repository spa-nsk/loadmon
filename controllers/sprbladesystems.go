package controllers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/csvtools"
	"loadmon/models"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// BladeSystemsFilter - фильтр списка /sprsrv/bladesystemslist. Текстовые
// поля фильтруются по подстроке, приём (фильтр в сессии, постраничная
// навигация, экспорт) идентичен StorageSystemsFilter в
// controllers/sprstoragesystems.go (образец стиля для этой формы).
// Active - числовое поле-перечисление (см. models.BladeSystems.Active),
// хранится строкой ("all"/"0"/"1"/"2") по тому же принципу, что и
// PowerState/Active/FormFactor в HWConfigFilter (controllers/sprhwconfig.go).
// ИЗМЕНЕНО (2026-09-08, по запросу пользователя): InventoryNum заменён на
// TechnicalNumberIT (технический номер ИТ, формат "IT-0000000000").
type BladeSystemsFilter struct {
	Vendor            string `form:"Vendor"`
	Model             string `form:"Model"`
	Type              string `form:"Type"`
	Name              string `form:"Name"`
	SerialNum         string `form:"SerialNum"`
	TechnicalNumberIT string `form:"TechnicalNumberIT"`
	Active            string `form:"Active"`
	ItiliumStatus     string `form:"ItiliumStatus"`
}

// defaultBladeSystemsFilter - фильтр по умолчанию для свежей сессии
// (см. keyFilter в SprBladeSystems). Active по умолчанию "1" (только
// активные корзины) - по запросу пользователя, в отличие от остальных
// текстовых полей (пустая строка - "без фильтра"). ItiliumStatus - "all"
// (поле новое, у старых записей часто 0).
func defaultBladeSystemsFilter() *BladeSystemsFilter {
	return &BladeSystemsFilter{Active: "1", ItiliumStatus: "all"}
}

// SprBladeSystems отображает список записей справочника блейд-корзин
// (models.BladeSystems, таблица blade_systems) с фильтром по
// Vendor/Model/Type/Name/SerialNum/TechnicalNumberIT/Active и
// постраничной навигацией, включая экспорт в CSV и XLSX - структура
// фильтра/пагинации/экспорта и оформление полностью повторяют
// /sprsrv/storagesystemslist (controllers/sprstoragesystems.go).
//
// Маршрут: /sprsrv/bladesystemslist/:id (:id зарезервирован под будущее
// использование, как и в /sprsrv/hwconflist/:id - сейчас не используется).
func (this *MainController) SprBladeSystems() {

	Id := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference) // просмотр справочника блейд-корзин

	this.activeContent("sprsrv/bladesystemslist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "BladeSystemsFilter"
	filter, ok := m[keyFilter].(*BladeSystemsFilter)
	if !ok {
		filter = defaultBladeSystemsFilter()
		m[keyFilter] = filter
	}

	page := 1
	pageSize := 50
	exportCSV := false
	exportXLSX := false

	if this.Ctx.Input.Method() == "POST" {
		if err := this.ParseForm(filter); err != nil {
			flash.Error("Ошибка распознавания формы фильтра: %v", err)
			flash.Store(&this.Controller)
			this.Data[keyFilter] = filter
			return
		}
		delete(m, keyFilter)
		m[keyFilter] = filter

		if p, err := strconv.Atoi(this.GetString("page")); err == nil && p > 0 {
			page = p
		}
		if s, err := strconv.Atoi(this.GetString("pagesize")); err == nil {
			switch s {
			case 25, 50, 100, 200:
				pageSize = s
			}
		}

		exportCSV = this.GetString("ExportBTN") == "Экспорт в CSV"
		exportXLSX = this.GetString("ExportBTN2") == "Экспорт в XLSX"
	}

	this.Data[keyFilter] = filter
	this.Data["ItiliumStatuses"] = models.ItiliumStatusList

	o := orm.NewOrm()
	qs := o.QueryTable("blade_systems")

	if filter.Vendor != "" {
		qs = qs.Filter("Vendor__icontains", filter.Vendor)
	}
	if filter.Model != "" {
		qs = qs.Filter("Model__icontains", filter.Model)
	}
	if filter.Type != "" {
		qs = qs.Filter("Type__icontains", filter.Type)
	}
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.SerialNum != "" {
		qs = qs.Filter("SerialNum__icontains", filter.SerialNum)
	}
	if filter.TechnicalNumberIT != "" {
		qs = qs.Filter("TechnicalNumberIT__icontains", filter.TechnicalNumberIT)
	}
	if filter.Active != "" && filter.Active != "all" {
		if v, err := strconv.Atoi(filter.Active); err == nil {
			qs = qs.Filter("Active", v)
		}
	}
	if filter.ItiliumStatus != "" && filter.ItiliumStatus != "all" {
		if v, err := strconv.Atoi(filter.ItiliumStatus); err == nil {
			qs = qs.Filter("ItiliumStatus", v)
		}
	}

	totalRecords, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	// экспорт отдаёт весь найденный по фильтру список, а не только текущую
	// страницу - поэтому Limit/Offset здесь не применяются (аналогично
	// /sprsrv/storagesystemslist)
	records := make([]models.BladeSystems, 0, pageSize)
	if exportCSV || exportXLSX {
		if _, err := qs.OrderBy("-Id").All(&records); err != nil {
			flash.Error("Ошибка выборки данных: %v", err)
			flash.Store(&this.Controller)
			return
		}
	} else {
		totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
		offset := (page - 1) * pageSize
		if _, err := qs.OrderBy("-Id").Limit(pageSize, offset).All(&records); err != nil {
			flash.Error("Ошибка выборки данных: %v", err)
			flash.Store(&this.Controller)
			return
		}
	}

	if exportXLSX {
		this.exportBladeSystemsXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportBladeSystemsCSV(records, flash)
		return
	}

	this.renderBladeSystemsPage(o, records, totalRecords, page, pageSize)
}

// bladeSystemsPlacementByCabinetId строит по списку записей models.BladeSystems
// (только по их RackCabinetId) карту "Id шкафа -> "ЦОД / шкаф"" - готовую
// строку для колонки "Размещение" списка /sprsrv/bladesystemslist. Сделано
// (2026-09-08, по запросу пользователя) взамен убранных ранее текстовых
// BladeSystems.DatacenterRoom/RackNum (см. CHANGELOG): единственный источник
// расположения корзины теперь FK RackCabinetId, поэтому имя ЦОД и шкафа
// приходится подтягивать отдельным запросом - аналогичный lookup уже
// используется в controllers/infrastructure_canvas.go для той же цели.
func bladeSystemsPlacementByCabinetId(o orm.Ormer, rows []models.BladeSystems) map[int64]string {
	result := map[int64]string{}
	cabinetIds := make([]int64, 0, len(rows))
	seen := map[int64]bool{}
	for _, r := range rows {
		if r.RackCabinetId > 0 && !seen[r.RackCabinetId] {
			seen[r.RackCabinetId] = true
			cabinetIds = append(cabinetIds, r.RackCabinetId)
		}
	}
	if len(cabinetIds) == 0 {
		return result
	}
	var cabinets []models.RackCabinet
	if _, err := o.QueryTable(new(models.RackCabinet)).Filter("Id__in", cabinetIds).All(&cabinets); err != nil {
		return result
	}
	siteIds := make([]int64, 0, len(cabinets))
	for _, c := range cabinets {
		if c.DatacenterSiteId > 0 {
			siteIds = append(siteIds, c.DatacenterSiteId)
		}
	}
	siteNameById := map[int64]string{}
	if len(siteIds) > 0 {
		var sites []models.DatacenterSite
		if _, err := o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", siteIds).All(&sites); err == nil {
			for _, s := range sites {
				siteNameById[s.Id] = s.Name
			}
		}
	}
	for _, c := range cabinets {
		site := siteNameById[c.DatacenterSiteId]
		switch {
		case site != "" && c.Name != "":
			result[c.Id] = site + " / " + c.Name
		case c.Name != "":
			result[c.Id] = c.Name
		default:
			result[c.Id] = site
		}
	}
	return result
}

// exportBladeSystemsXLSX отдаёт найденный по фильтру список записей
// models.BladeSystems файлом XLSX на скачивание - формирование листа
// аналогично экспорту /sprsrv/storagesystemslist (exportStorageSystemsXLSX).
func (this *MainController) exportBladeSystemsXLSX(rows []models.BladeSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("bladesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Блейд-корзины"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &header); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j := range rows {
		row, _ := csvtools.GetCSVData(&rows[j])
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		return
	}

	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	this.Ctx.Output.Header("Content-Disposition", "attachment; filename=\""+shortName+"\"")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")
	this.Ctx.Output.Header("Content-Length", fmt.Sprint(fileInfo.Size()))

	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}

// exportBladeSystemsCSV отдаёт найденный по фильтру список записей
// models.BladeSystems файлом CSV (utf8, разделитель "|") на скачивание -
// аналогично /sprsrv/storagesystemslist (exportStorageSystemsCSV).
func (this *MainController) exportBladeSystemsCSV(rows []models.BladeSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("bladesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	csvFile, err := os.Create(destination)
	if err != nil {
		flash.Error("Ошибка создания CSV файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		return
	}

	csvwriter := csv.NewWriter(csvFile)
	csvwriter.Comma = '|'
	header, _ := csvtools.GetCSVHeader(&rows[0])
	_ = csvwriter.Write(header)
	for j := range rows {
		data, _ := csvtools.GetCSVData(&rows[j])
		_ = csvwriter.Write(data)
	}
	csvwriter.Flush()
	csvFile.Close()

	this.Ctx.Output.Header("Content-Disposition", "attachment; filename="+shortName)
	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Type", "application/csv")
	this.Ctx.Output.Header("Content-Transfer-Encoding", "text")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")

	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}

// renderBladeSystemsPage кладёт в Data готовую страницу списка (записи +
// параметры пагинации) - аналогично renderStorageSystemsPage. Дополнительно
// строит bladeSystemsPlacementByCabinetId для колонки "Размещение" (см.
// комментарий у этой функции).
func (this *MainController) renderBladeSystemsPage(o orm.Ormer, rows []models.BladeSystems, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["BladeSystemsList"] = rows
	this.Data["BladeSystemsPlacement"] = bladeSystemsPlacementByCabinetId(o, rows)
	this.Data["BladeSystemsCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = int64(0)
	if totalRecords > 0 {
		this.Data["PageFrom"] = int64(offset) + 1
	}
	pageTo := int64(offset + pageSize)
	if pageTo > totalRecords {
		pageTo = totalRecords
	}
	this.Data["PageTo"] = pageTo
}

// bladeSystemsDateLayout - формат для полей <input type="date"> в форме
// редактирования - аналогично storageSystemsDateLayout в
// controllers/sprstoragesystems.go.
const bladeSystemsDateLayout = "2006-01-02"

func formatBladeSystemsDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(bladeSystemsDateLayout)
}

func parseBladeSystemsDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(bladeSystemsDateLayout, s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// bladeSystemsComputeSizeUnit вычисляет BladeSystems.SizeUnit (количество
// занимаемых юнитов) по StartUnitPos/EndUnitPos - по запросу пользователя
// (2026-09-08) SizeUnit больше не вводится в форме редактирования напрямую
// (введены только начальная и конечная позиция), а всегда пересчитывается
// перед сохранением - и здесь (форма редактирования), и при импорте из
// Итиллиум (см. itilliumimport/importblades.go). При некорректных или не
// заполненных значениях (EndUnitPos < StartUnitPos, StartUnitPos <= 0)
// возвращает 0 ("не указано") вместо того, чтобы писать отрицательное или
// заведомо неверное число.
func bladeSystemsComputeSizeUnit(startUnitPos, endUnitPos int) int {
	if startUnitPos <= 0 || endUnitPos < startUnitPos {
		return 0
	}
	return endUnitPos - startUnitPos + 1
}

// SprBladeSystemsEdit отображает форму просмотра/редактирования одной записи
// models.BladeSystems (таблица blade_systems) по Id и сохраняет изменения по
// отправке формы (POST) - поля читаются вручную по имени (а не через
// this.ParseForm), чтобы не задевать служебные поля (Id_User, Time_Mfy) -
// приём идентичен SprStorageSystemsEdit в controllers/sprstoragesystems.go
// (образец стиля для этой формы).
//
// Id=0 - режим добавления новой записи (ссылка "Добавить" со страницы
// /sprsrv/bladesystemslist ведёт на /sprsrv/bladesystemsedit-tab/0): форма
// показывается пустой, по отправке создаёт новую запись (o.Insert) и
// переходит на страницу редактирования уже созданной записи по её Id.
//
// Маршрут: /sprsrv/bladesystemsedit/:id
func (this *MainController) SprBladeSystemsEdit() {
	sprBladeSystemsEditPage(this, "sprsrv/bladesystemsedit", this.Ctx.Input.Param(":id"))
}

// SprBladeSystemsEditTab - та же карточка блейд-корзины, что
// SprBladeSystemsEdit, но с формой на закладках (шаблон bladesystemsedit_tab.tpl).
//
// Маршрут: /sprsrv/bladesystemsedit-tab/:id([0-9]+)
func (this *MainController) SprBladeSystemsEditTab() {
	sprBladeSystemsEditPage(this, "sprsrv/bladesystemsedit_tab", this.Ctx.Input.Param(":id"))
}

// bladeComponentTypes - типы модулей блейд-корзины для ручного добавления
// (значения = cmdb.ServerHWComponent*, см. syncbladesystemcomponents.go).
func bladeComponentTypes() []hwComponentTypeOption {
	return []hwComponentTypeOption{
		{6, "Interconnect"},
		{7, "Management / OA"},
		{12, "Блок питания"},
		{13, "Вентилятор"},
	}
}

// bladeComponentTabDefs - закладки модулей корзины (порядок на форме).
func bladeComponentTabDefs() []hwComponentTypeOption {
	return []hwComponentTypeOption{
		{6, "Interconnect"},
		{7, "Management / OA"},
		{12, "Блоки питания"},
		{13, "Вентиляторы"},
	}
}

// loadBladeComponents читает модули корзины (BladeSystemId).
func loadBladeComponents(o orm.Ormer, bladeSystemId int64) []models.ServerHardwareComponents {
	var comps []models.ServerHardwareComponents
	if bladeSystemId <= 0 {
		return comps
	}
	_, _ = o.QueryTable(new(models.ServerHardwareComponents)).
		Filter("BladeSystemId", bladeSystemId).
		OrderBy("Type", "Slot", "Id").
		All(&comps)
	return comps
}

// buildBladeComponentTabs раскладывает модули корзины по закладкам типов.
func buildBladeComponentTabs(comps []models.ServerHardwareComponents) []hwComponentTab {
	byType := map[int][]models.ServerHardwareComponents{}
	for _, c := range comps {
		byType[c.Type] = append(byType[c.Type], c)
	}
	defs := bladeComponentTabDefs()
	known := make(map[int]bool, len(defs))
	tabs := make([]hwComponentTab, 0, len(defs)+1)
	for _, d := range defs {
		known[d.Value] = true
		items := byType[d.Value]
		if items == nil {
			items = []models.ServerHardwareComponents{}
		}
		tabs = append(tabs, hwComponentTab{
			Type: d.Value, Name: d.Name, Items: items,
			ShowCapacity: false, ShowSubType: false,
		})
	}
	var other []models.ServerHardwareComponents
	for t, items := range byType {
		if !known[t] {
			other = append(other, items...)
		}
	}
	if len(other) > 0 {
		tabs = append(tabs, hwComponentTab{
			Type: 0, Name: "Прочие", Items: other,
			ShowCapacity: true, ShowSubType: true,
		})
	}
	return tabs
}

// setBladeComponentFormData кладёт закладки модулей в Data для bladesystemsedit_tab.tpl.
func setBladeComponentFormData(this *MainController, comps []models.ServerHardwareComponents) {
	this.Data["BsComponentTypes"] = bladeComponentTypes()
	this.Data["HwComponentDiskSubTypes"] = hwComponentDiskSubTypes() // для закладки «Прочие»
	this.Data["BsComponentTabs"] = buildBladeComponentTabs(comps)
}

// saveBladeComponents сохраняет набор ServerHardwareComponents для корзины
// (BladeSystemId) по полям CompType/CompSlot/... формы. Схема «удалить все
// по BladeSystemId, вставить заново» — как у hwconf и sync OneView.
// Вызывается только из tab-формы.
func (this *MainController) saveBladeComponents(o orm.Ormer, bs *models.BladeSystems) error {
	if bs == nil || bs.Id <= 0 {
		return nil
	}
	types := this.GetStrings("CompType")
	subTypes := this.GetStrings("CompSubType")
	vendors := this.GetStrings("CompVendor")
	compModels := this.GetStrings("CompModel")
	serials := this.GetStrings("CompSerialNum")
	caps := this.GetStrings("CompCapacityGb")
	slots := this.GetStrings("CompSlot")

	var currentUserId int64 = 9999
	if uid, ok := this.Data["UserId"].(int64); ok && uid > 0 {
		currentUserId = uid
	}
	now := time.Now().Local()

	trn, err := o.Begin()
	if err != nil {
		return err
	}
	if _, err := trn.QueryTable(new(models.ServerHardwareComponents)).Filter("BladeSystemId", bs.Id).Delete(); err != nil {
		_ = trn.Rollback()
		return err
	}

	for i, typeStr := range types {
		t, _ := strconv.Atoi(strings.TrimSpace(typeStr))
		if t <= 0 {
			continue
		}
		var sub int
		if i < len(subTypes) {
			sub, _ = strconv.Atoi(strings.TrimSpace(subTypes[i]))
		}
		vendor, model, serial, slot := "", "", "", ""
		if i < len(vendors) {
			vendor = strings.TrimSpace(vendors[i])
		}
		if i < len(compModels) {
			model = strings.TrimSpace(compModels[i])
		}
		if i < len(serials) {
			serial = strings.TrimSpace(serials[i])
		}
		if i < len(slots) {
			slot = strings.TrimSpace(slots[i])
		}
		var capGb float64
		if i < len(caps) {
			capGb, _ = strconv.ParseFloat(strings.TrimSpace(caps[i]), 64)
		}
		c := models.ServerHardwareComponents{
			BladeSystemId:   bs.Id,
			Type:            t,
			SubType:         sub,
			Vendor:          vendor,
			Model:           model,
			SerialNum:       serial,
			CapacityGb:      capGb,
			Slot:            slot,
			ContractDoc:     bs.ContractDoc,
			DateContractDoc: bs.DateContractDoc,
			Invoice:         bs.Invoice,
			DateInvoice:     bs.DateInvoice,
			ServiceContract: bs.ServiceContract,
			DateService:     bs.DateService,
			Id_User:         currentUserId,
			Time_Mfy:        now,
		}
		if _, err := trn.Insert(&c); err != nil {
			_ = trn.Rollback()
			return err
		}
	}
	return trn.Commit()
}

// sprBladeSystemsEditPage - общая логика GET/POST формы BladeSystems.
// tpl - sprsrv/bladesystemsedit или sprsrv/bladesystemsedit_tab.
func sprBladeSystemsEditPage(this *MainController, tpl, idStr string) {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки блейд-корзины - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent(tpl)
	this.Data["Id"] = idStr
	this.Data["BladeEditBasePath"] = "/sprsrv/bladesystemsedit"
	isTab := strings.Contains(tpl, "bladesystemsedit_tab")
	if isTab {
		this.Data["BladeEditBasePath"] = "/sprsrv/bladesystemsedit-tab"
	}

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}
	isNew := id == 0
	this.Data["IsNew"] = isNew

	o := orm.NewOrm()

	var rec models.BladeSystems
	if !isNew {
		if err := o.QueryTable("blade_systems").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["BS"] = rec
			if isTab {
				setBladeComponentFormData(this, loadBladeComponents(o, rec.Id))
			}
			return
		}
		rec.Name = this.GetString("Name")
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.Type = this.GetString("Type")
		rec.PartNum = this.GetString("PartNum")
		rec.SerialNum = this.GetString("SerialNum")
		rec.Size, _ = strconv.Atoi(this.GetString("Size"))
		rec.Used, _ = strconv.Atoi(this.GetString("Used"))

		// ИЗМЕНЕНО (2026-09-08, по запросу пользователя): в форме указываются
		// StartUnitPos и EndUnitPos, SizeUnit в форме больше не отображается -
		// вычисляется ниже перед сохранением (см.
		// bladeSystemsComputeSizeUnit).
		rec.StartUnitPos, _ = strconv.Atoi(this.GetString("StartUnitPos"))
		rec.EndUnitPos, _ = strconv.Atoi(this.GetString("EndUnitPos"))
		rec.SizeUnit = bladeSystemsComputeSizeUnit(rec.StartUnitPos, rec.EndUnitPos)
		rec.RackCabinetId, _ = strconv.ParseInt(this.GetString("RackCabinetId"), 10, 64)

		rec.Owner = this.GetString("Owner")
		rec.Anotation = this.GetString("Anotation")

		rec.PowerState, _ = strconv.Atoi(this.GetString("PowerState"))
		rec.Active, _ = strconv.Atoi(this.GetString("Active"))
		rec.ItiliumStatus, _ = strconv.Atoi(this.GetString("ItiliumStatus"))
		rec.TechnicalNumberIT = this.GetString("TechnicalNumberIT")

		rec.InventoryNum = this.GetString("InventoryNum")
		rec.InventoryNumBuh = this.GetString("InventoryNumBuh")
		rec.ContractDoc = this.GetString("ContractDoc")
		rec.DateContractDoc = parseBladeSystemsDate(this.GetString("DateContractDoc"))
		rec.Invoice = this.GetString("Invoice")
		rec.DateInvoice = parseBladeSystemsDate(this.GetString("DateInvoice"))
		rec.Counterparty = this.GetString("Counterparty")
		rec.ServiceContract = this.GetString("ServiceContract")
		rec.DateService = parseBladeSystemsDate(this.GetString("DateService"))

		rec.Time_Mfy = time.Now().Local()

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["BS"] = rec
				if isTab {
					setBladeComponentFormData(this, nil)
				}
				return
			}
			if isTab {
				if err := this.saveBladeComponents(o, &rec); err != nil {
					flash.Error("Запись создана (Id=%d), но не удалось сохранить модули: %v", rec.Id, err)
					flash.Store(&this.Controller)
				}
			}
			RegisterRequest(this, LogTypeEditReference) // создание записи справочника блейд-корзин
			basePath, _ := this.Data["BladeEditBasePath"].(string)
			if basePath == "" {
				basePath = "/sprsrv/bladesystemsedit"
			}
			this.Redirect(fmt.Sprintf("%s/%d", basePath, rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["BS"] = rec
			if isTab {
				setBladeComponentFormData(this, loadBladeComponents(o, rec.Id))
			}
			return
		}
		RegisterRequest(this, LogTypeEditReference) // изменение записи справочника блейд-корзин

		if isTab {
			if err := this.saveBladeComponents(o, &rec); err != nil {
				flash.Error("Запись сохранена, но не удалось сохранить модули: %v", err)
				flash.Store(&this.Controller)
			} else {
				flash.Notice("Запись сохранена.")
				flash.Store(&this.Controller)
			}
		} else {
			flash.Notice("Запись сохранена.")
			flash.Store(&this.Controller)
		}
	}

	// ДОБАВЛЕНО (2026-09-08, по запросу пользователя): выбор ЦОД, по которому
	// в форме отфильтровывается список шкафов этого ЦОД (каскадный select,
	// см. bladesystemsedit.tpl) - сохраняется по-прежнему только
	// BladeSystems.RackCabinetId, ЦОД выводится для удобства выбора и на
	// сервере не читается (this.GetString("DatacenterSiteId") нигде не
	// используется).
	var sites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&sites)
	this.Data["DatacenterSites"] = sites

	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks

	this.Data["BS"] = rec
	this.Data["ItiliumStatuses"] = models.ItiliumStatusList
	this.Data["DateContractDocStr"] = formatBladeSystemsDate(rec.DateContractDoc)
	this.Data["DateInvoiceStr"] = formatBladeSystemsDate(rec.DateInvoice)
	this.Data["DateServiceStr"] = formatBladeSystemsDate(rec.DateService)
	if isTab {
		setBladeComponentFormData(this, loadBladeComponents(o, rec.Id))
	}
}

// BladeSystemServerRow - одна строка списка /sprsrv/bladesystemserverslist
// (models.ServerHardware, привязанный к корзине через BladeSystemId),
// дополненная именем сервера из справочника серверов (models.ServerSprRecords,
// ServerHardware.ServerId) - расшифровка нужна, потому что сама
// ServerHardware имени не хранит (только конфигурацию).
type BladeSystemServerRow struct {
	models.ServerHardware
	ServerName string // ServerSprRecords.Name (или NameFQDN, если короткое имя пусто) по ServerId, "" если запись оборудования ни с одним сервером не связана
}

// SprBladeSystemServersList отображает read-only список серверов
// (models.ServerHardware), физически установленных в ОДНОЙ блейд-корзине
// (models.BladeSystems, BladeSystemId), отсортированный по номеру слота
// (BladeNum) - связка "блейд-сервер <-> блейд-корзина с фиксацией позиции"
// уже существует в схеме (ServerHardware.BladeSystemId/BladeNum,
// проставляется при синхронизации с HP OneView, см.
// cmdb/initrec.go:ImportHWSprRecord) - этот список делает её видимой в
// веб-интерфейсе.
//
// Открывается по правому клику на строке /sprsrv/bladesystemslist
// ("Серверы в корзине" в контекстном меню).
//
// Маршрут: /sprsrv/bladesystemserverslist/:id (:id - Id записи models.BladeSystems)
func (this *MainController) SprBladeSystemServersList() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр серверов в блейд-корзине

	this.activeContent("sprsrv/bladesystemserverslist")
	this.Data["Id"] = idStr

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()

	var bls models.BladeSystems
	if err := o.QueryTable("blade_systems").Filter("Id", id).One(&bls); err != nil {
		flash.Error("Блейд-корзина с Id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["BS"] = bls

	var hwList []models.ServerHardware
	if _, err := o.QueryTable("server_hardware").Filter("BladeSystemId", id).OrderBy("BladeNum").All(&hwList); err != nil {
		flash.Error("Ошибка чтения списка серверов: %v", err)
		flash.Store(&this.Controller)
		return
	}

	serverIds := make([]int64, 0, len(hwList))
	for _, hw := range hwList {
		if hw.ServerId != 0 {
			serverIds = append(serverIds, hw.ServerId)
		}
	}
	nameById := make(map[int64]string, len(serverIds))
	if len(serverIds) > 0 {
		var srvsprs []models.ServerSprRecords
		if _, err := models.ServerSprQS(o).Filter("Id__in", serverIds).All(&srvsprs); err == nil {
			for _, s := range srvsprs {
				name := s.Name
				if name == "" {
					name = s.NameFQDN
				}
				nameById[s.Id] = name
			}
		}
	}

	rows := make([]BladeSystemServerRow, 0, len(hwList))
	for _, hw := range hwList {
		rows = append(rows, BladeSystemServerRow{ServerHardware: hw, ServerName: nameById[hw.ServerId]})
	}
	this.Data["Servers"] = rows
	this.Data["ServersCount"] = len(rows)
}

// bladeChassisComp - компонент корзины (models.ServerHardwareComponents с
// BladeSystemId) для 3D-визуализации /sprsrv/bladesystemserverslistvisual.
// Kind: psu (Type=12), fan (13), interconnect (6), manager (7: OA / Frame Link).
type bladeChassisComp struct {
	Id    int64          `json:"id"`
	Kind  string         `json:"kind"`
	Bay   int            `json:"bay"`   // номер отсека (1..N), 0 если неизвестен
	Index int            `json:"index"` // порядковый индекс среди компонентов этого вида (0-based)
	Name  string         `json:"name"`
	Model string         `json:"model"`
	Power int            `json:"power"` // 0 неизвестно, 1 OK/Present, 2 Absent/Failed/выкл
	Rows  []rackViewKV   `json:"rows"`
	Links []rackViewLink `json:"links"`
}

// bladeChassisVisualData - JSON для 3D-визуализации одной блейд-корзины
// (/sprsrv/bladesystemserverslistvisual): слоты/блейды как rackViewItem
// kind=chassis в /sprsrv/rackcabinetview, плюс компоненты корзины.
type bladeChassisVisualData struct {
	Chassis    rackViewItem       `json:"chassis"`
	Components []bladeChassisComp `json:"components"`
}

// bladeChassisCompKinds - типы компонентов корзины, выводимые в 3D-виде.
var bladeChassisCompKinds = []struct {
	Type  int
	Kind  string
	Title string // заголовок в имени компонента
}{
	{12, "psu", "Блок питания"},
	{13, "fan", "Вентилятор"},
	{6, "interconnect", "Interconnect-модуль"},
	{7, "manager", "Модуль управления"},
}

// parseBladePSUBayAndStatus разбирает Slot компоненты БП вида
// "Bay 3, OK" / "Bay 1, Present" (см. cmdb/syncbladesystemcomponents.go).
func parseBladePSUBayAndStatus(slot string) (bay int, power int, status string) {
	slot = strings.TrimSpace(slot)
	power = 0
	if slot == "" {
		return 0, 0, ""
	}
	status = slot
	parts := strings.SplitN(slot, ",", 2)
	head := strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		status = strings.TrimSpace(parts[1])
	}
	lowHead := strings.ToLower(head)
	if strings.HasPrefix(lowHead, "bay") {
		num := strings.TrimSpace(head[3:])
		if n, err := strconv.Atoi(num); err == nil && n > 0 {
			bay = n
		}
	}
	st := strings.ToLower(status)
	switch {
	case strings.Contains(st, "absent"), strings.Contains(st, "fail"),
		strings.Contains(st, "error"), strings.Contains(st, "critical"),
		strings.Contains(st, "off"), strings.Contains(st, "выкл"):
		power = 2
	case strings.Contains(st, "ok"), strings.Contains(st, "present"), strings.Contains(st, "warn"),
		strings.Contains(st, "normal"), strings.Contains(st, "good"),
		strings.Contains(st, "on"), strings.HasPrefix(st, "ip "):
		power = 1
	}
	return bay, power, status
}

// SprBladeSystemServersListVisual - визуализация одной блейд-корзины с блейдами
// в слотах и блоками питания (отрисовка как корзины в /sprsrv/rackcabinetview)
// и панелью параметров выбранного блейда или БП.
//
// Маршрут: /sprsrv/bladesystemserverslistvisual/:id([0-9]+)
func (this *MainController) SprBladeSystemServersListVisual() {
	idStr := this.Ctx.Input.Param(":id")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/bladesystemserverslistvisual")
	this.Data["Id"] = idStr
	this.Data["ChassisJSON"] = template.JS("null")

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()
	var bls models.BladeSystems
	if err := o.QueryTable("blade_systems").Filter("Id", id).One(&bls); err != nil {
		flash.Error("Блейд-корзина с Id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["BS"] = bls

	var bladeRows []models.ServerHardware
	if _, err := o.QueryTable("server_hardware").Filter("BladeSystemId", id).OrderBy("BladeNum").All(&bladeRows); err != nil {
		flash.Error("Ошибка чтения списка серверов: %v", err)
		flash.Store(&this.Controller)
		return
	}

	hwIds := make([]int64, 0, len(bladeRows))
	for _, hw := range bladeRows {
		hwIds = append(hwIds, hw.Id)
	}
	sprByHW := map[int64]models.ServerSprRecords{}
	if len(hwIds) > 0 {
		var sprRows []models.ServerSprRecords
		if _, err := models.ServerSprQS(o).Filter("Stat", 1).Filter("HWSprId__in", hwIds).All(&sprRows); err == nil {
			for _, s := range sprRows {
				if _, dup := sprByHW[s.HWSprId]; !dup {
					sprByHW[s.HWSprId] = s
				}
			}
		}
	}

	bladeDetail := func(hw models.ServerHardware) (rows []rackViewKV, links []rackViewLink, name string) {
		spr, hasSpr := sprByHW[hw.Id]
		if hasSpr && strings.TrimSpace(spr.Name) != "" {
			name = spr.Name
		}
		if name == "" {
			name = firstNonEmpty(hw.SerialNum, hw.Model, fmt.Sprintf("srv-%d", hw.Id))
		}
		if hasSpr {
			rows = rackViewKVAdd(rows, "FQDN", spr.NameFQDN)
			rows = rackViewKVAdd(rows, "IP сервера", spr.IPAddress)
			if spr.Type == 2 {
				rows = rackViewKVAdd(rows, "Роль", "Гипервизор VMware")
			}
		}
		rows = rackViewKVAdd(rows, "Слот", fmt.Sprintf("%d", hw.BladeNum))
		rows = rackViewKVAdd(rows, "Серийный номер", hw.SerialNum)
		rows = rackViewKVAdd(rows, "Номер продукта", hw.PartNum)
		rows = rackViewKVAdd(rows, "Инвентарный номер", hw.InventoryNum)
		rows = rackViewKVAdd(rows, "IP iLO/MP", hw.MpIPAddress)
		rows = rackViewKVAdd(rows, "Hostname iLO/MP", hw.MpHostName)
		rows = rackViewKVAdd(rows, "Модель iLO/MP", hw.MpModel)
		if hw.CPUCount > 0 {
			rows = rackViewKVAdd(rows, "Процессоры", fmt.Sprintf("%d × %s (%d ядер @ %d MHz)", hw.CPUCount, strings.TrimSpace(hw.CPUModel), hw.CorePerCPU, hw.CoreMHz))
		}
		if hw.MemoryMb > 0 {
			rows = rackViewKVAdd(rows, "Память", fmt.Sprintf("%d ГБ (%d МБ)", hw.MemoryMb/1024, hw.MemoryMb))
		}
		rows = rackViewPowerRows(rows, hw.PowerAvg24hW, hw.PowerPeak24hW, hw.PowerRatedMaxW)
		rows = rackViewKVAdd(rows, "Состояние (health)", hw.Status)

		serverSprId := hw.ServerId
		if hasSpr && spr.Id > 0 {
			serverSprId = spr.Id
		}
		if serverSprId > 0 {
			links = append(links, rackViewLink{Title: "Общие данные по серверу", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", serverSprId)})
			links = append(links,
				rackViewLink{Title: "Мощность за 3 суток", URL: fmt.Sprintf("/sprsrv/powerrecent/server/%d", serverSprId)},
				rackViewLink{Title: "Мощность за период", URL: fmt.Sprintf("/sprsrv/powerview/server/%d", serverSprId)})
		}
		links = append(links, rackViewLink{Title: "Карточка оборудования", URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", hw.Id)})
		if sn := strings.TrimSpace(hw.SerialNum); sn != "" {
			links = append(links, rackViewLink{Title: "Данные OneView", URL: "/sprsrv/srvhwviewbysn/" + sn})
		}
		if hasSpr && spr.ZabbixHostId > 0 {
			links = append(links, rackViewLink{Title: "Метрики производительности (Zabbix)", URL: fmt.Sprintf("/sprsrv/zabbixperfperiod/%d", spr.ZabbixHostId)})
		}
		return rows, links, name
	}

	slots := bls.Size
	if slots <= 0 {
		slots = 16
	}
	sizeU := bls.SizeUnit
	if sizeU <= 0 && bls.EndUnitPos >= bls.StartUnitPos && bls.StartUnitPos > 0 {
		sizeU = bls.EndUnitPos - bls.StartUnitPos + 1
	}
	if sizeU <= 0 {
		sizeU = 10
	}

	var chRows []rackViewKV
	chRows = rackViewKVAdd(chRows, "Тип", bls.Type)
	chRows = rackViewKVAdd(chRows, "Серийный номер", bls.SerialNum)
	chRows = rackViewKVAdd(chRows, "Номер продукта", bls.PartNum)
	chRows = rackViewKVAdd(chRows, "Инвентарный номер", bls.InventoryNum)
	chRows = rackViewKVAdd(chRows, "Слотов (занято)", fmt.Sprintf("%d (%d)", slots, len(bladeRows)))
	chRows = rackViewPowerRows(chRows, bls.PowerAvg24hW, bls.PowerPeak24hW, bls.PowerRatedMaxW)
	chLinks := []rackViewLink{
		{Title: "Серверы в корзине (список)", URL: fmt.Sprintf("/sprsrv/bladesystemserverslist/%d", bls.Id)},
		{Title: "Мощность корзины за период", URL: fmt.Sprintf("/sprsrv/powerview/blade/%d", bls.Id)},
		{Title: "Редактировать корзину", URL: fmt.Sprintf("/sprsrv/bladesystemsedit-tab/%d", bls.Id)},
	}

	blades := make([]rackViewBlade, 0, len(bladeRows))
	for _, b := range bladeRows {
		brows, blinks, bname := bladeDetail(b)
		blades = append(blades, rackViewBlade{
			Id: b.Id, Slot: b.BladeNum, Name: bname, Model: firstNonEmpty(b.Model, b.ShortModel),
			Power: rackViewPower(b.PowerState), Rows: brows, Links: blinks,
		})
	}

	item := rackViewItem{
		Kind: "chassis", Id: bls.Id,
		Name:   firstNonEmpty(bls.Name, bls.SerialNum, fmt.Sprintf("chassis-%d", bls.Id)),
		Vendor: bls.Vendor, Model: firstNonEmpty(bls.Model, bls.Type),
		StartU: bls.StartUnitPos, SizeU: sizeU,
		Power: rackViewPower(bls.PowerState), Slots: slots,
		Rows: chRows, Links: chLinks, Blades: blades,
	}

	// Компоненты корзины (БП, вентиляторы, interconnect, OA/FLM) - объекты 3D-вида.
	components := make([]bladeChassisComp, 0, 40)
	countByKind := map[string]int{}
	for _, kd := range bladeChassisCompKinds {
		var comps []models.ServerHardwareComponents
		_, _ = o.QueryTable(new(models.ServerHardwareComponents)).
			Filter("BladeSystemId", id).Filter("Type", kd.Type).
			OrderBy("Id").All(&comps)
		for i, c := range comps {
			bay, pwr, status := parseBladePSUBayAndStatus(c.Slot)
			name := firstNonEmpty(c.Model, fmt.Sprintf("%s %d", kd.Title, i+1))
			if bay > 0 {
				name = fmt.Sprintf("%s, отсек %d", kd.Title, bay)
			}
			var rows []rackViewKV
			if bay > 0 {
				rows = rackViewKVAdd(rows, "Отсек (bay)", fmt.Sprintf("%d", bay))
			}
			rows = rackViewKVAdd(rows, "Производитель", c.Vendor)
			rows = rackViewKVAdd(rows, "Модель", c.Model)
			rows = rackViewKVAdd(rows, "Серийный номер", c.SerialNum)
			rows = rackViewKVAdd(rows, "Расположение / статус", c.Slot)
			rows = rackViewKVAdd(rows, "Статус", status)
			if c.CapacityGb > 0 && kd.Kind == "psu" {
				rows = rackViewKVAdd(rows, "Мощность (номинал)", fmt.Sprintf("%.0f Вт", c.CapacityGb))
			}
			components = append(components, bladeChassisComp{
				Id: c.Id, Kind: kd.Kind, Bay: bay, Index: i, Name: name,
				Model: firstNonEmpty(c.Model, c.Vendor), Power: pwr,
				Rows: rows,
				Links: []rackViewLink{
					{Title: "Редактировать корзину", URL: fmt.Sprintf("/sprsrv/bladesystemsedit-tab/%d", bls.Id)},
				},
			})
			countByKind[kd.Kind]++
		}
	}
	chRows = rackViewKVAdd(chRows, "Блоков питания", fmt.Sprintf("%d", countByKind["psu"]))
	if n := countByKind["fan"]; n > 0 {
		chRows = rackViewKVAdd(chRows, "Вентиляторов", fmt.Sprintf("%d", n))
	}
	if n := countByKind["interconnect"]; n > 0 {
		chRows = rackViewKVAdd(chRows, "Interconnect-модулей", fmt.Sprintf("%d", n))
	}
	if n := countByKind["manager"]; n > 0 {
		chRows = rackViewKVAdd(chRows, "Модулей управления (OA/FLM)", fmt.Sprintf("%d", n))
	}
	item.Rows = chRows

	payload, jerr := json.Marshal(bladeChassisVisualData{Chassis: item, Components: components})
	if jerr != nil {
		flash.Error("Ошибка сериализации данных визуализации: %v", jerr)
		flash.Store(&this.Controller)
		return
	}
	this.Data["ChassisJSON"] = template.JS(payload)
	this.Data["ServersCount"] = len(bladeRows)
}
