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

// StorageSystemsFilter - фильтр списка /sprsrv/storagesystemslist. Числовые
// поля-перечисления (TypeWork/Status) хранятся как строки ("all"/"0"/"1"/...),
// чтобы отличать "фильтр не применён" от осмысленного значения 0 ("не
// указано" в самой таблице models.StorageSystems) - приём идентичен
// HWConfigFilter в controllers/sprhwconfig.go (образец стиля для этой формы).
type StorageSystemsFilter struct {
	TypeWork       string `form:"TypeWork"`
	Status         string `form:"Status"`
	Vendor         string `form:"Vendor"`
	Model          string `form:"Model"`
	ShortName      string `form:"ShortName"`
	SerialNum      string `form:"SerialNum"`
	InventoryNum   string `form:"InventoryNum"`
	DatacenterRoom string `form:"DatacenterRoom"`
	RackNum        string `form:"RackNum"`
}

func defaultStorageSystemsFilter() *StorageSystemsFilter {
	return &StorageSystemsFilter{TypeWork: "all", Status: "all"}
}

// SprStorageSystems отображает список записей справочника СХД
// (models.StorageSystems, таблица storage_systems) с фильтром по
// TypeWork/Status/Vendor/Model/ShortName/SerialNum/InventoryNum/
// DatacenterRoom/RackNum и постраничной навигацией, включая экспорт в CSV и
// XLSX - оформление, структура фильтра/пагинации и экспорта полностью
// повторяют /sprsrv/hwconflist (controllers/sprhwconfig.go), в отличие от
// которого здесь не нужен join со справочником серверов - models.StorageSystems
// самодостаточна, поэтому строки списка - это сами записи StorageSystems без
// обёртки.
//
// Маршрут: /sprsrv/storagesystemslist/:id (:id зарезервирован под будущее
// использование, как и в /sprsrv/hwconflist/:id - сейчас не используется).
func (this *MainController) SprStorageSystems() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_storagesystemslist_"+Id, 302)
		return
	}
	RegisterRequest(this, LogTypeViewReference) // просмотр справочника СХД

	this.activeContent("sprsrv/storagesystemslist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "StorageSystemsFilter"
	filter, ok := m[keyFilter].(*StorageSystemsFilter)
	if !ok {
		filter = defaultStorageSystemsFilter()
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
	qs := o.QueryTable("storage_systems")

	if filter.TypeWork != "" && filter.TypeWork != "all" {
		if v, err := strconv.Atoi(filter.TypeWork); err == nil {
			qs = qs.Filter("TypeWork", v)
		}
	}
	if filter.Status != "" && filter.Status != "all" {
		if v, err := strconv.Atoi(filter.Status); err == nil {
			qs = qs.Filter("Status", v)
		}
	}
	if filter.Vendor != "" {
		qs = qs.Filter("Vendor__icontains", filter.Vendor)
	}
	if filter.Model != "" {
		qs = qs.Filter("Model__icontains", filter.Model)
	}
	if filter.ShortName != "" {
		qs = qs.Filter("ShortName__icontains", filter.ShortName)
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
	// /sprsrv/hwconflist)
	records := make([]models.StorageSystems, 0, pageSize)
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
		this.exportStorageSystemsXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportStorageSystemsCSV(records, flash)
		return
	}

	this.renderStorageSystemsPage(records, totalRecords, page, pageSize)
}

// exportStorageSystemsXLSX отдаёт найденный по фильтру список записей
// models.StorageSystems файлом XLSX на скачивание - формирование листа
// аналогично экспорту /sprsrv/hwconflist (exportHWConfigXLSX).
func (this *MainController) exportStorageSystemsXLSX(rows []models.StorageSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("storagesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Справочник СХД"
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

// exportStorageSystemsCSV отдаёт найденный по фильтру список записей
// models.StorageSystems файлом CSV (utf8, разделитель "|") на скачивание -
// аналогично /sprsrv/hwconflist (exportHWConfigCSV).
func (this *MainController) exportStorageSystemsCSV(rows []models.StorageSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("storagesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
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

// renderStorageSystemsPage кладёт в Data готовую страницу списка (записи +
// параметры пагинации) - аналогично renderHWConfigPage.
func (this *MainController) renderStorageSystemsPage(rows []models.StorageSystems, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["StorageSystemsList"] = rows
	this.Data["StorageSystemsCount"] = totalRecords
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

// storageSystemsDateLayout - формат для полей <input type="date"> в форме
// редактирования (совпадает с форматом, который отдаёт и принимает браузер) -
// аналогично hwConfigDateLayout в controllers/sprhwconfig.go.
const storageSystemsDateLayout = "2006-01-02"

func formatStorageSystemsDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(storageSystemsDateLayout)
}

func parseStorageSystemsDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(storageSystemsDateLayout, s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SprStorageSystemsEdit отображает форму просмотра/редактирования одной
// записи models.StorageSystems (таблица storage_systems) по Id и сохраняет
// изменения по отправке формы (POST) - поля читаются вручную по имени (а не
// через this.ParseForm), чтобы не задевать служебные поля (Id_User,
// Time_Mfy), которые эта форма не показывает - приём идентичен
// SprHwConfigEdit в controllers/sprhwconfig.go (образец стиля для этой формы).
//
// Id=0 - режим добавления новой записи (ссылка "Добавить" со страницы
// /sprsrv/storagesystemslist ведёт на /sprsrv/storagesystemsedit/0): форма
// показывается пустой, по отправке создаёт новую запись (o.Insert) и
// переходит на страницу редактирования уже созданной записи по её Id.
//
// Маршрут: /sprsrv/storagesystemsedit/:id
func (this *MainController) SprStorageSystemsEdit() {

	idStr := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_storagesystemsedit_"+idStr, 302)
		return
	}
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки СХД - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("sprsrv/storagesystemsedit")
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

	var rec models.StorageSystems
	if !isNew {
		if err := o.QueryTable("storage_systems").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["ST"] = rec
			return
		}
		rec.UID = this.GetString("UID")
		rec.ShortName = this.GetString("ShortName")
		rec.TagName = this.GetString("TagName")
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.PartNum = this.GetString("PartNum")
		rec.SerialNum = this.GetString("SerialNum")
		rec.Firmware = this.GetString("Firmware")
		rec.TypeWork, _ = strconv.Atoi(this.GetString("TypeWork"))
		rec.Status, _ = strconv.Atoi(this.GetString("Status"))
		rec.Power, _ = strconv.Atoi(this.GetString("Power"))
		rec.ApiType, _ = strconv.Atoi(this.GetString("ApiType"))

		rec.Size, _ = strconv.Atoi(this.GetString("Size"))
		rec.Used, _ = strconv.Atoi(this.GetString("Used"))
		rec.IpAddress = this.GetString("IpAddress")
		rec.IpAddressNext = this.GetString("IpAddressNext")
		rec.TotalSizeRAW, _ = strconv.ParseFloat(this.GetString("TotalSizeRAW"), 64)
		rec.UsedPercent, _ = strconv.ParseFloat(this.GetString("UsedPercent"), 64)
		rec.WWNMask = this.GetString("WWNMask")

		rec.Ad = this.GetString("Ad") == "on"
		rec.Ldaps = this.GetString("Ldaps") == "on"
		rec.AdCert = this.GetString("AdCert") == "on"
		rec.AdCertDate = parseStorageSystemsDate(this.GetString("AdCertDate"))

		rec.City = this.GetString("City")
		rec.DatacenterRoom = this.GetString("DatacenterRoom")
		rec.RackNum = this.GetString("RackNum")
		rec.StartUnit = this.GetString("StartUnit")
		rec.SizeUnit = this.GetString("SizeUnit")

		rec.Owner = this.GetString("Owner")
		rec.Allocation = this.GetString("Allocation")
		rec.Anotation = this.GetString("Anotation")

		rec.InventoryNum = this.GetString("InventoryNum")
		rec.InventoryNumBuh = this.GetString("InventoryNumBuh")
		rec.ContractDoc = this.GetString("ContractDoc")
		rec.DateContractDoc = parseStorageSystemsDate(this.GetString("DateContractDoc"))
		rec.Invoice = this.GetString("Invoice")
		rec.DateInvoice = parseStorageSystemsDate(this.GetString("DateInvoice"))
		rec.Counterparty = this.GetString("Counterparty")
		rec.ServiceContract = this.GetString("ServiceContract")
		rec.DateService = parseStorageSystemsDate(this.GetString("DateService"))

		rec.Time_Mfy = time.Now().Local()

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["ST"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference) // создание записи справочника СХД
			this.Redirect(fmt.Sprintf("/sprsrv/storagesystemsedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["ST"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference) // изменение записи справочника СХД

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	this.Data["ST"] = rec
	this.Data["AdCertDateStr"] = formatStorageSystemsDate(rec.AdCertDate)
	this.Data["DateContractDocStr"] = formatStorageSystemsDate(rec.DateContractDoc)
	this.Data["DateInvoiceStr"] = formatStorageSystemsDate(rec.DateInvoice)
	this.Data["DateServiceStr"] = formatStorageSystemsDate(rec.DateService)
}
