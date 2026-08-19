package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// BladeSystemsFilter - фильтр списка /sprsrv/bladesystemslist. models.BladeSystems
// в отличие от models.StorageSystems не содержит числовых полей-перечислений
// (TypeWork/Status и т.п.) - все поля фильтра текстовые, приём (фильтр в
// сессии, постраничная навигация, экспорт) идентичен StorageSystemsFilter в
// controllers/sprstoragesystems.go (образец стиля для этой формы).
type BladeSystemsFilter struct {
	Vendor         string `form:"Vendor"`
	Model          string `form:"Model"`
	Type           string `form:"Type"`
	Name           string `form:"Name"`
	SerialNum      string `form:"SerialNum"`
	InventoryNum   string `form:"InventoryNum"`
	DatacenterRoom string `form:"DatacenterRoom"`
	RackNum        string `form:"RackNum"`
}

func defaultBladeSystemsFilter() *BladeSystemsFilter {
	return &BladeSystemsFilter{}
}

// SprBladeSystems отображает список записей справочника блейд-корзин
// (models.BladeSystems, таблица blade_systems) с фильтром по
// Vendor/Model/Type/Name/SerialNum/InventoryNum/DatacenterRoom/RackNum и
// постраничной навигацией, включая экспорт в CSV и XLSX - структура
// фильтра/пагинации/экспорта и оформление полностью повторяют
// /sprsrv/storagesystemslist (controllers/sprstoragesystems.go).
//
// Маршрут: /sprsrv/bladesystemslist/:id (:id зарезервирован под будущее
// использование, как и в /sprsrv/hwconflist/:id - сейчас не используется).
func (this *MainController) SprBladeSystems() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_bladesystemslist_"+Id, 302)
		return
	}
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
	if filter.InventoryNum != "" {
		qs = qs.Filter("InventoryNum__icontains", filter.InventoryNum)
	}
	if filter.DatacenterRoom != "" {
		qs = qs.Filter("DatacenterRoom__icontains", filter.DatacenterRoom)
	}
	if filter.RackNum != "" {
		qs = qs.Filter("RackNum__icontains", filter.RackNum)
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

	this.renderBladeSystemsPage(records, totalRecords, page, pageSize)
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
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Блейд-корзины"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &header); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j := range rows {
		row, _ := csvtools.GetCSVData(&rows[j])
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" ", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" ", err)
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
		flash.Error("Ошибка создания CSV файла: "+destination+" ", err)
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
// параметры пагинации) - аналогично renderStorageSystemsPage.
func (this *MainController) renderBladeSystemsPage(rows []models.BladeSystems, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["BladeSystemsList"] = rows
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

// SprBladeSystemsEdit отображает форму просмотра/редактирования одной записи
// models.BladeSystems (таблица blade_systems) по Id и сохраняет изменения по
// отправке формы (POST) - поля читаются вручную по имени (а не через
// this.ParseForm), чтобы не задевать служебные поля (Id_User, Time_Mfy) -
// приём идентичен SprStorageSystemsEdit в controllers/sprstoragesystems.go
// (образец стиля для этой формы).
//
// Id=0 - режим добавления новой записи (ссылка "Добавить" со страницы
// /sprsrv/bladesystemslist ведёт на /sprsrv/bladesystemsedit/0): форма
// показывается пустой, по отправке создаёт новую запись (o.Insert) и
// переходит на страницу редактирования уже созданной записи по её Id.
//
// Маршрут: /sprsrv/bladesystemsedit/:id
func (this *MainController) SprBladeSystemsEdit() {

	idStr := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_bladesystemsedit_"+idStr, 302)
		return
	}
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки блейд-корзины - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("sprsrv/bladesystemsedit")
	this.Data["Id"] = idStr

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
		rec.Name = this.GetString("Name")
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.Type = this.GetString("Type")
		rec.PartNum = this.GetString("PartNum")
		rec.SerialNum = this.GetString("SerialNum")
		rec.Size, _ = strconv.Atoi(this.GetString("Size"))
		rec.Used, _ = strconv.Atoi(this.GetString("Used"))

		rec.DatacenterRoom = this.GetString("DatacenterRoom")
		rec.RackNum = this.GetString("RackNum")
		rec.StartUnit = this.GetString("StartUnit")
		rec.StartUnitPos, _ = strconv.Atoi(this.GetString("StartUnitPos"))
		rec.SizeUnit, _ = strconv.Atoi(this.GetString("SizeUnit"))

		rec.Owner = this.GetString("Owner")
		rec.Allocation = this.GetString("Allocation")
		rec.Anotation = this.GetString("Anotation")
		rec.Description = this.GetString("Description")

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
				return
			}
			RegisterRequest(this, LogTypeEditReference) // создание записи справочника блейд-корзин
			this.Redirect(fmt.Sprintf("/sprsrv/bladesystemsedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["BS"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference) // изменение записи справочника блейд-корзин

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	this.Data["BS"] = rec
	this.Data["DateContractDocStr"] = formatBladeSystemsDate(rec.DateContractDoc)
	this.Data["DateInvoiceStr"] = formatBladeSystemsDate(rec.DateInvoice)
	this.Data["DateServiceStr"] = formatBladeSystemsDate(rec.DateService)
}
