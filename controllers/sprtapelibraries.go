package controllers

import (
	"fmt"
	"loadmon/itilliumimport"
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// TapeLibrariesFilter - фильтр списка /sprsrv/tapelibrarieslist. Перечисления и
// Id хранятся строками ("all"/"0"/...), чтобы отличать "не применён" от 0.
type TapeLibrariesFilter struct {
	ItiliumStatus string `form:"ItiliumStatus"`
	Name          string `form:"Name"`
	Model         string `form:"Model"`
	SerialNum     string `form:"SerialNum"`
	TechNumberIT  string `form:"TechNumberIT"` // технический номер ИТ
	PcName        string `form:"PcName"`
	RackCabinetId string `form:"RackCabinetId"`
}

func defaultTapeLibrariesFilter() *TapeLibrariesFilter {
	return &TapeLibrariesFilter{ItiliumStatus: "1", RackCabinetId: "0"} // по умолчанию - "В работе"
}

// tapeLibraryRow - строка списка: запись + разрешённые размещение и сервер.
type tapeLibraryRow struct {
	models.TapeLibrary
	Placement  string
	ServerName string
}

// SprTapeLibraries - список справочника ленточных библиотек (models.TapeLibrary)
// с фильтром и постраничной навигацией. Маршрут: /sprsrv/tapelibrarieslist/:id
func (this *MainController) SprTapeLibraries() {
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/tapelibrarieslist")
	this.Data["ListId"] = this.Ctx.Input.Param(":id")

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "TapeLibrariesFilter"
	filter, ok := m[keyFilter].(*TapeLibrariesFilter)
	if !ok {
		filter = defaultTapeLibrariesFilter()
		m[keyFilter] = filter
	}

	page, pageSize := 1, 50
	exportCSV, exportXLSX := false, false
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

	var dcSites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&dcSites)
	this.Data["DatacenterSites"] = dcSites
	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks

	qs := o.QueryTable(new(models.TapeLibrary))
	if filter.ItiliumStatus != "" && filter.ItiliumStatus != "all" {
		if v, err := strconv.Atoi(filter.ItiliumStatus); err == nil {
			qs = qs.Filter("ItiliumStatus", v)
		}
	}
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.Model != "" {
		qs = qs.Filter("Model__icontains", filter.Model)
	}
	if filter.SerialNum != "" {
		qs = qs.Filter("SerialNum__icontains", filter.SerialNum)
	}
	if filter.TechNumberIT != "" {
		qs = qs.Filter("TechnicalNumberIT__icontains", filter.TechNumberIT)
	}
	if filter.PcName != "" {
		qs = qs.Filter("PcName__icontains", filter.PcName)
	}
	if v, err := strconv.ParseInt(filter.RackCabinetId, 10, 64); err == nil && v > 0 {
		qs = qs.Filter("RackCabinetId", v)
	}

	total, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	var records []models.TapeLibrary
	qsPage := qs.OrderBy("Name", "Id")
	if !exportCSV && !exportXLSX { // экспорт отдаёт весь найденный список, не страницу
		qsPage = qsPage.Limit(pageSize, offset)
	}
	if _, err := qsPage.All(&records); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	siteName := map[int64]string{}
	for _, s := range dcSites {
		siteName[s.Id] = s.Name
	}
	rackInfo := map[int64]models.RackCabinet{}
	for _, r := range racks {
		rackInfo[r.Id] = r
	}
	serverIds := make([]int64, 0, len(records))
	for _, r := range records {
		if r.ServerId > 0 {
			serverIds = append(serverIds, r.ServerId)
		}
	}
	serverName := tapeLibraryServerNames(o, serverIds)

	rows := make([]tapeLibraryRow, 0, len(records))
	for _, r := range records {
		row := tapeLibraryRow{TapeLibrary: r, ServerName: serverName[r.ServerId]}
		row.Placement = tapeLibraryPlacement(r, siteName, rackInfo)
		rows = append(rows, row)
	}

	if exportXLSX {
		this.exportTapeLibraries(rows, true, flash)
		return
	}
	if exportCSV {
		this.exportTapeLibraries(rows, false, flash)
		return
	}

	pageTo := int64(offset + pageSize)
	if pageTo > total {
		pageTo = total
	}
	pageFrom := int64(0)
	if total > 0 {
		pageFrom = int64(offset) + 1
	}
	this.Data["TapeLibrariesList"] = rows
	this.Data["TapeLibrariesCount"] = total
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = pageFrom
	this.Data["PageTo"] = pageTo
}

// tapeLibraryPlacement - "Площадка / Шкаф, U<начало>[-<конец>]" для списка.
func tapeLibraryPlacement(r models.TapeLibrary, siteName map[int64]string, racks map[int64]models.RackCabinet) string {
	rack, ok := racks[r.RackCabinetId]
	if !ok {
		return ""
	}
	s := siteName[rack.DatacenterSiteId]
	if s != "" {
		s += " / "
	}
	s += rack.Name
	if r.StartUnitPos > 0 {
		s += fmt.Sprintf(", U%d", r.StartUnitPos)
		if r.SizeUnit > 1 {
			s += fmt.Sprintf("-%d", r.StartUnitPos+r.SizeUnit-1)
		}
	}
	return s
}

// tapeLibraryServerNames возвращает имена серверов (ServerSprRecords) по Id.
func tapeLibraryServerNames(o orm.Ormer, ids []int64) map[int64]string {
	res := map[int64]string{}
	if len(ids) == 0 {
		return res
	}
	var list []models.ServerSprRecords
	if _, err := o.QueryTable(new(models.ServerSprRecords)).Filter("Id__in", ids).All(&list, "Id", "Name"); err != nil {
		return res
	}
	for _, s := range list {
		res[s.Id] = s.Name
	}
	return res
}

// SprTapeLibrariesEdit - просмотр/редактирование/добавление одной записи
// models.TapeLibrary (Id=0 - новая). Маршрут: /sprsrv/tapelibrariesedit/:id
func (this *MainController) SprTapeLibrariesEdit() {
	sprTapeLibrariesEditPage(this, "sprsrv/tapelibrariesedit", this.Ctx.Input.Param(":id"))
}

// SprTapeLibrariesEditTab - та же карточка ленточной библиотеки, что
// SprTapeLibrariesEdit, но с формой на закладках (tapelibrariesedit_tab.tpl).
//
// Маршрут: /sprsrv/tapelibrariesedit-tab/:id([0-9]+)
func (this *MainController) SprTapeLibrariesEditTab() {
	sprTapeLibrariesEditPage(this, "sprsrv/tapelibrariesedit_tab", this.Ctx.Input.Param(":id"))
}

// sprTapeLibrariesEditPage - общая логика GET/POST формы TapeLibrary.
// tpl - sprsrv/tapelibrariesedit или sprsrv/tapelibrariesedit_tab.
func sprTapeLibrariesEditPage(this *MainController, tpl, idStr string) {
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent(tpl)
	this.Data["Id"] = idStr
	this.Data["TapeEditBasePath"] = "/sprsrv/tapelibrariesedit"
	if strings.Contains(tpl, "tapelibrariesedit_tab") {
		this.Data["TapeEditBasePath"] = "/sprsrv/tapelibrariesedit-tab"
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
	var rec models.TapeLibrary
	if !isNew {
		if err := o.QueryTable(new(models.TapeLibrary)).Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.fillTapeLibraryEditData(o, rec)
			return
		}
		var parseErrors []string
		parseIntField := func(label, formName string) int {
			s := strings.TrimSpace(this.GetString(formName))
			if s == "" {
				return 0
			}
			v, err := strconv.Atoi(s)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("%s: %q - не целое число", label, s))
				return 0
			}
			return v
		}

		rec.Name = this.GetString("Name")
		rec.Model = this.GetString("Model")
		rec.SerialNum = this.GetString("SerialNum")
		rec.IPAddress = this.GetString("IPAddress")
		rec.ItiliumStatus = parseIntField("Статус Итилиум", "ItiliumStatus")
		rec.StartUnitPos = parseIntField("Начальный юнит", "StartUnitPos")
		rec.SizeUnit = parseIntField("Размер в юнитах", "SizeUnit")
		rec.RackCabinetId = int64(parseIntField("Шкаф", "RackCabinetId"))
		rec.ServerId = int64(parseIntField("Сервер", "ServerId"))
		rec.PcName = strings.TrimSpace(this.GetString("PcName"))
		rec.Owner = this.GetString("Owner")
		rec.Counterparty = this.GetString("Counterparty")
		rec.ContractDoc = this.GetString("ContractDoc")
		rec.DateContractDoc = parseStorageSystemsDate(this.GetString("DateContractDoc"))
		rec.InventoryNum = this.GetString("InventoryNum")
		rec.InventoryNumBuh = this.GetString("InventoryNumBuh")
		rec.TechnicalNumberIT = this.GetString("TechnicalNumberIT")
		rec.Anotation = this.GetString("Anotation")

		if len(parseErrors) > 0 {
			flash.Error("Не удалось сохранить - проверьте числовые поля: %s", strings.Join(parseErrors, "; "))
			flash.Store(&this.Controller)
			this.fillTapeLibraryEditData(o, rec)
			return
		}
		// Сервер не выбран явно - ищем по "Имя ПК" в справочнике серверов.
		if rec.ServerId == 0 && rec.PcName != "" {
			if sid, err := itilliumimport.FindServerByPcName(o, rec.PcName); err == nil {
				rec.ServerId = sid
			}
		}
		if uid, ok := this.Data["UserId"].(int64); ok {
			rec.Id_User = uid
		}
		rec.Time_Mfy = time.Now().Local()

		if isNew {
			rec.SourceUpdateType = models.SourceUpdateManual
			rec.TimeCreate = rec.Time_Mfy
			rec.CreatorUser = rec.Id_User
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.fillTapeLibraryEditData(o, rec)
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			basePath, _ := this.Data["TapeEditBasePath"].(string)
			if basePath == "" {
				basePath = "/sprsrv/tapelibrariesedit"
			}
			this.Redirect(fmt.Sprintf("%s/%d", basePath, rec.Id), 302)
			return
		}
		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.fillTapeLibraryEditData(o, rec)
			return
		}
		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
		RegisterRequest(this, LogTypeEditReference)
	}

	this.fillTapeLibraryEditData(o, rec)
}

func (this *MainController) fillTapeLibraryEditData(o orm.Ormer, rec models.TapeLibrary) {
	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks
	var dcSites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&dcSites)
	this.Data["DatacenterSites"] = dcSites

	this.Data["TL"] = rec
	this.Data["ItiliumStatuses"] = models.ItiliumStatusList
	this.Data["ServerName"] = tapeLibraryServerNames(o, []int64{rec.ServerId})[rec.ServerId]
	this.Data["DateContractDocStr"] = formatStorageSystemsDate(rec.DateContractDoc)
}

// tapeLibraryExportTable - заголовок и строки выгрузки списка (читаемые значения:
// статус текстом, размещение, имя сервера).
func tapeLibraryExportTable(rows []tapeLibraryRow) ([]string, [][]string) {
	header := []string{"Id", "Статус Итилиум", "Название", "Модель", "Серийный номер", "IP адрес",
		"Технический номер ИТ", "Инвентарный номер", "Инв. номер (бух.)", "Имя ПК", "Сервер (справочник)",
		"Размещение", "Начальный юнит", "Размер, юнитов", "Автор", "Поставщик", "Документы закупки",
		"Дата документа закупки", "Описание"}
	data := make([][]string, 0, len(rows))
	for _, r := range rows {
		data = append(data, []string{
			strconv.FormatInt(r.Id, 10), models.ItiliumStatusName(r.ItiliumStatus), r.Name, r.Model, r.SerialNum,
			r.IPAddress, r.TechnicalNumberIT, r.InventoryNum, r.InventoryNumBuh, r.PcName, r.ServerName,
			r.Placement, strconv.Itoa(r.StartUnitPos), strconv.Itoa(r.SizeUnit), r.Owner, r.Counterparty,
			r.ContractDoc, formatStorageSystemsDate(r.DateContractDoc), r.Anotation,
		})
	}
	return header, data
}

// exportTapeLibraries отдаёт список файлом XLSX (xlsx=true) или CSV на скачивание.
func (this *MainController) exportTapeLibraries(rows []tapeLibraryRow, xlsx bool, flash *beego.FlashData) {
	header, data := tapeLibraryExportTable(rows)
	this.exportTableFile(header, data, xlsx, "tapelibraries", "Ленточные библиотеки", flash)
}
