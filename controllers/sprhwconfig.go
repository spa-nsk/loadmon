package controllers

import (
	"encoding/csv"
	"fmt"
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

// HWConfigFilter - фильтр списка /sprsrv/hwconflist. Числовые поля-перечисления
// (PowerState/Active/FormFactor) хранятся как строки ("all"/"0"/"1"/"2"/"3"),
// чтобы отличать "фильтр не применён" от осмысленного значения 0 ("не
// указано" в самой таблице models.ServerHardware). ServerName - поиск по
// связанному справочнику серверов (models.ServerSprRecords.Name/NameFQDN,
// связь через ServerHardware.ServerId).
//
// ИЗМЕНЕНО (2026-09-08, по запросу пользователя "в справочнике серверов в
// фильтре используй схему с выбором ЦОД и Шкафа вместо ЦОД: адрес и площадка
// по старым данным"): текстовые поля DatacenterRoom/RackNum (свободный текст
// по старым данным Итиллиум/OneView) заменены на RackCabinetId - выбор
// конкретного шкафа из справочника models.RackCabinet (с каскадным select
// ЦОД -> Шкаф в шаблоне, тот же приём, что уже сделан в форме
// bladesystemsedit.tpl для models.BladeSystems, см. sprbladesystems.go).
// "0"/"" означает "фильтр по шкафу не применён" (см. RackCabinetId ниже),
// по аналогии с "все" для select-фильтров выше.
type HWConfigFilter struct {
	PowerState        string `form:"PowerState"`
	Active            string `form:"Active"`
	Vendor            string `form:"Vendor"`
	Model             string `form:"Model"`
	PartNum           string `form:"PartNum"`
	FormFactor        string `form:"FormFactor"`
	CPUModel          string `form:"CPUModel"`
	SerialNum         string `form:"SerialNum"`
	RackCabinetId     string `form:"RackCabinetId"`
	ServerName        string `form:"ServerName"`
	TechnicalNumberIT string `form:"TechnicalNumberIT"`
}

func defaultHWConfigFilter() *HWConfigFilter {
	return &HWConfigFilter{PowerState: "all", Active: "all", FormFactor: "all", RackCabinetId: "0"}
}

// HWConfigRow - одна строка списка /sprsrv/hwconflist: запись
// models.ServerHardware, дополненная именем сервера из связанного
// справочника models.ServerSprRecords (ServerHardware.ServerId ->
// ServerSprRecords.Id). ServerName/ServerNameFQDN остаются пустыми, если
// ServerId не заполнен или ссылается на несуществующую запись.
type HWConfigRow struct {
	models.ServerHardware
	ServerName     string
	ServerNameFQDN string
	// Placement - человекочитаемое "ЦОД / Шкаф" по RackCabinetId (тот же
	// текст, что показывается в колонке "Размещение" hwconflist.tpl через
	// Data["HWConfigPlacement"]) - добавлено в CSV/XLSX-экспорт (2026-09-09,
	// по запросу пользователя "реализуй аналогичную обработку... и других
	// где отображается ServerHardware"), чтобы выгрузка не отдавала только
	// сырой числовой RackCabinetId без резолва в имя.
	Placement string `csv:"Размещение (ЦОД / Шкаф)"`
}

// SprHwConfig отображает список записей справочника аппаратной конфигурации
// серверов (models.ServerHardware, таблица server_hardware) с фильтром по
// PowerState/Active/Vendor/Model/PartNum/FormFactor/CPUModel/RackCabinetId
// (шкаф из справочника, см. HWConfigFilter)/имени сервера и постраничной
// навигацией - оформление аналогично /sprsrv/view2 (фильтр POST-формой,
// классическая таблица, номер страницы передаётся скрытым полем формы).
//
// Маршрут: /sprsrv/hwconflist/:id
func (this *MainController) SprHwConfig() {

	Id := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference) // просмотр справочника конфигураций серверов

	this.activeContent("sprsrv/hwconflist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "HWConfigFilter"
	filter, ok := m[keyFilter].(*HWConfigFilter)
	if !ok {
		filter = defaultHWConfigFilter()
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

		// beego.ParseForm (beego/v2/server/web/context/form.go: formFirstValue)
		// для строковых полей молча пропускает присвоение, если значение поля
		// формы - пустая строка. Из-за этого при очистке текстового поля
		// фильтра и нажатии "Просмотр" ParseForm никак не меняет
		// соответствующее поле filter - а filter это не новый объект на
		// каждый запрос, а указатель, сохранённый в сессии (m[keyFilter]
		// выше), поэтому в нём остаётся прежнее (непустое) значение с
		// прошлого поиска, и очищенное поле фильтра "само" заполняется
		// обратно. Читаем эти строковые поля вручную через this.GetString -
		// он корректно возвращает "" для пустого/отсутствующего поля - и
		// присваиваем их filter напрямую, перекрывая то, что (не) сделал
		// ParseForm. Тот же паттерн уже применён в controllers/sprsrvview2.go
		// (см. kontext/05-conventions-and-gotchas.md). Select-поля
		// (PowerState/Active/FormFactor) этой проблемы не имеют - они всегда
		// отправляют непустое значение ("all"/"1"/"2"/...), даже когда выбран
		// вариант "Все".
		filter.Vendor = this.GetString("Vendor")
		filter.Model = this.GetString("Model")
		filter.PartNum = this.GetString("PartNum")
		filter.CPUModel = this.GetString("CPUModel")
		filter.SerialNum = this.GetString("SerialNum")
		filter.ServerName = this.GetString("ServerName")
		filter.TechnicalNumberIT = this.GetString("TechnicalNumberIT")

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

	// Справочники ЦОД/Шкаф для каскадного select-фильтра размещения (см.
	// комментарий у HWConfigFilter.RackCabinetId выше и JS в hwconflist.tpl) -
	// тот же приём, что уже используется в форме bladesystemsedit.tpl
	// (sprbladesystems.go:SprBladeSystemsEdit).
	var sites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&sites)
	this.Data["DatacenterSites"] = sites

	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks

	qs := o.QueryTable("server_hardware")

	if filter.PowerState != "" && filter.PowerState != "all" {
		if v, err := strconv.Atoi(filter.PowerState); err == nil {
			qs = qs.Filter("PowerState", v)
		}
	}
	if filter.Active != "" && filter.Active != "all" {
		if v, err := strconv.Atoi(filter.Active); err == nil {
			qs = qs.Filter("Active", v)
		}
	}
	if filter.FormFactor != "" && filter.FormFactor != "all" {
		if v, err := strconv.Atoi(filter.FormFactor); err == nil {
			qs = qs.Filter("FormFactor", v)
		}
	}
	if filter.Vendor != "" {
		qs = qs.Filter("Vendor__icontains", filter.Vendor)
	}
	if filter.Model != "" {
		qs = qs.Filter("Model__icontains", filter.Model)
	}
	if filter.PartNum != "" {
		qs = qs.Filter("PartNum__icontains", filter.PartNum)
	}
	if filter.CPUModel != "" {
		qs = qs.Filter("CPUModel__icontains", filter.CPUModel)
	}
	if filter.SerialNum != "" {
		qs = qs.Filter("SerialNum__icontains", filter.SerialNum)
	}
	if filter.RackCabinetId != "" && filter.RackCabinetId != "0" {
		if v, err := strconv.ParseInt(filter.RackCabinetId, 10, 64); err == nil && v > 0 {
			qs = qs.Filter("RackCabinetId", v)
		}
	}
	if filter.TechnicalNumberIT != "" {
		qs = qs.Filter("TechnicalNumberIT__icontains", filter.TechnicalNumberIT)
	}

	// поиск по имени сервера ищет в связанном справочнике
	// (server_spr_records.Name/NameFQDN), т.к. в самой server_hardware имени
	// нет - только ссылка ServerId. Если по имени ничего не найдено, дальше
	// основной запрос выполнять не нужно - результат заведомо пуст.
	if filter.ServerName != "" {
		cond := orm.NewCondition().
			Or("Name__icontains", filter.ServerName).
			Or("NameFQDN__icontains", filter.ServerName)
		var sprMatches []models.ServerSprRecords
		if _, err := o.QueryTable("server_spr_records").SetCond(cond).All(&sprMatches, "Id"); err != nil {
			flash.Error("Ошибка поиска по имени сервера: %v", err)
			flash.Store(&this.Controller)
			return
		}
		if len(sprMatches) == 0 {
			this.renderHWConfigPage(nil, 0, page, pageSize)
			return
		}
		sprIds := make([]int64, 0, len(sprMatches))
		for _, s := range sprMatches {
			sprIds = append(sprIds, s.Id)
		}
		qs = qs.Filter("ServerId__in", sprIds)
	}

	totalRecords, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	// экспорт отдаёт весь найденный по фильтру список, а не только текущую
	// страницу - поэтому Limit/Offset здесь не применяются
	records := make([]models.ServerHardware, 0, pageSize)
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

	// имена серверов из связанного справочника - одним запросом по всем
	// ServerId выбранного набора записей, а не по одному запросу на строку
	rows := make([]HWConfigRow, len(records))
	serverIds := make([]int64, 0, len(records))
	for i, rec := range records {
		rows[i] = HWConfigRow{ServerHardware: rec}
		if rec.ServerId != 0 {
			serverIds = append(serverIds, rec.ServerId)
		}
	}
	if len(serverIds) > 0 {
		var sprRecs []models.ServerSprRecords
		_, _ = o.QueryTable("server_spr_records").Filter("Id__in", serverIds).All(&sprRecs, "Id", "Name", "NameFQDN")
		sprById := make(map[int64]models.ServerSprRecords, len(sprRecs))
		for _, s := range sprRecs {
			sprById[s.Id] = s
		}
		for i := range rows {
			if s, ok := sprById[rows[i].ServerId]; ok {
				rows[i].ServerName = s.Name
				rows[i].ServerNameFQDN = s.NameFQDN
			}
		}
	}

	// Размещение "ЦОД / Шкаф" по RackCabinetId (2026-09-08, по запросу
	// пользователя "исправь размещение в списке конфигураций серверов
	// аналогично блэйд корзинам") - тот же приём (и тот же lookup-паттерн),
	// что уже применяется в bladesystemslist.tpl, см.
	// sprbladesystems.go:bladeSystemsPlacementByCabinetId. Вычисляется ДО
	// ветвления на экспорт (было только после - экспорт получал сырой
	// RackCabinetId без резолва, см. HWConfigRow.Placement выше, добавлено
	// по запросу "реализуй аналогичную обработку... и других где
	// отображается ServerHardware") - используется и для HTML-колонки, и
	// для CSV/XLSX.
	placementByCabinetId := hwConfigPlacementByCabinetId(o, records)
	for i := range rows {
		if rows[i].RackCabinetId > 0 {
			rows[i].Placement = placementByCabinetId[rows[i].RackCabinetId]
		}
	}

	if exportXLSX {
		this.exportHWConfigXLSX(rows, flash)
		return
	}
	if exportCSV {
		this.exportHWConfigCSV(rows, flash)
		return
	}

	this.Data["HWConfigPlacement"] = placementByCabinetId

	this.renderHWConfigPage(rows, totalRecords, page, pageSize)
}

// hwConfigPlacementByCabinetId строит по списку записей models.ServerHardware
// (только по их RackCabinetId) карту "Id шкафа -> "ЦОД / шкаф"" - для колонки
// "Размещение" в hwconflist.tpl. Полный аналог
// sprbladesystems.go:bladeSystemsPlacementByCabinetId (для models.BladeSystems) -
// продублирован здесь как отдельная функция, а не переиспользован напрямую,
// т.к. тот принимает []models.BladeSystems, а не []models.ServerHardware, и
// добавлять ради этого обобщённый (generics/интерфейс) вариант ради одной
// строчки не оправдано.
func hwConfigPlacementByCabinetId(o orm.Ormer, rows []models.ServerHardware) map[int64]string {
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

// exportHWConfigXLSX отдаёт найденный по фильтру список записей
// models.ServerHardware (в виде HWConfigRow - с именем сервера) файлом XLSX
// на скачивание - формирование листа аналогично экспорту /sprsrv/view2.
func (this *MainController) exportHWConfigXLSX(rows []HWConfigRow, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("hwconf_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Конфигурация серверов"
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

// exportHWConfigCSV отдаёт найденный по фильтру список записей
// models.ServerHardware (в виде HWConfigRow - с именем сервера) файлом CSV
// (utf8, разделитель ";") на скачивание - формирование аналогично экспорту
// /sprsrv/view2.
func (this *MainController) exportHWConfigCSV(rows []HWConfigRow, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("hwconf_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
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

// renderHWConfigPage кладёт в Data готовую страницу списка (строки +
// параметры пагинации) - общий хвост как для обычной выборки, так и для
// случая "по имени сервера ничего не найдено" (rows == nil, totalRecords == 0).
func (this *MainController) renderHWConfigPage(rows []HWConfigRow, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["HWConfigList"] = rows
	this.Data["HWConfigCount"] = totalRecords
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

// hwConfigDateLayout - формат для полей <input type="date"> в форме
// редактирования (совпадает с форматом, который отдаёт и принимает браузер).
const hwConfigDateLayout = "2006-01-02"

// formatHWConfigDate возвращает t в формате hwConfigDateLayout, либо пустую
// строку для нулевого времени (чтобы <input type="date"> не показывал
// "0001-01-01").
func formatHWConfigDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(hwConfigDateLayout)
}

// parseHWConfigDate разбирает строку из <input type="date">; при пустой или
// некорректной строке возвращает нулевое время (поле в БД - orm:"null").
func parseHWConfigDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(hwConfigDateLayout, s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SprHwConfigEdit отображает форму редактирования одной записи
// models.ServerHardware (таблица server_hardware) по Id и сохраняет
// изменения по отправке формы (POST). Поля, доступные для правки, читаются
// вручную по имени (а не через this.ParseForm) - так проще выборочно
// сохранять только редактируемые поля и не задевать служебные (Id_User,
// Time_Mfy, ссылочные Id на другие справочники), которые эта форма не
// показывает.
//
// Id=0 - режим добавления новой записи (ссылка "Добавить" со страницы
// /sprsrv/hwconflist ведёт на /sprsrv/hwconfedit/0): вместо загрузки
// существующей записи форма показывается пустой, а по отправке форма не
// обновляет, а создаёт новую запись (o.Insert) и переходит на страницу
// редактирования уже созданной записи по её настоящему Id.
//
// SprHwConfigEditBySN находит запись models.ServerHardware по серийному
// номеру (SerialNum) и переходит на её карточку /sprsrv/hwconfedit/:id,
// найденному по внутреннему Id. Используется пунктом меню "Конфигурация
// сервера" на /sprsrv/view2 (там для строк аппаратного сервера/гипервизора
// известен только серийный номер оборудования, а не внутренний Id записи
// server_hardware - в отличие от /sprsrv/hwconflist, где ссылка строится по
// уже загруженной записи и её Id известен сразу).
//
// Если запись с таким серийным номером не найдена, пользователь
// перенаправляется на форму создания новой записи (/sprsrv/hwconfedit/0) с
// предупреждением в flash - серийный номер при этом не подставляется
// автоматически, так как форма добавления не принимает предзаполнение через
// параметры запроса.
//
// Маршрут: /sprsrv/hwconfeditbysn/:SN
func (this *MainController) SprHwConfigEditBySN() {
	sn := this.Ctx.Input.Param(":SN")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeServerDetails) // переход к конфигурации сервера по серийному номеру

	flash := beego.NewFlash()

	o := orm.NewOrm()
	var rec models.ServerHardware
	if err := o.QueryTable("server_hardware").Filter("SerialNum", sn).One(&rec); err != nil {
		flash.Notice("Запись конфигурации сервера с серийным номером \"%s\" не найдена. Можно создать новую.", sn)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/hwconfedit/0", 302)
		return
	}

	this.Redirect(fmt.Sprintf("/sprsrv/hwconfedit/%d", rec.Id), 302)
}

// SprHwConfigEditByName находит запись справочника серверов
// (models.ServerSprRecords) по имени (Name) - физический сервер или
// гипервизор VMware (Type 0/2) - и переходит на карточку связанной с ней
// записи конфигурации оборудования (models.ServerHardware) по её Id
// (ServerSprRecords.HWSprId - прямая ссылка на server_hardware, см.
// cmdb/savedata.go и др.). В отличие от SprHwConfigEditBySN (поиск по
// серийному номеру оборудования), здесь ключ поиска - имя записи
// справочника серверов, как оно хранится в БД (совпадает с именем
// хоста/сервера, а не с серийным номером) - нужен для ссылок оттуда, где
// известно только имя (например, хосты кластера на /sprsrv/cluster/...,
// у которых серийный номер оборудования не всегда собирается через
// vSphere).
//
// Если запись справочника серверов с таким именем не найдена, или найдена,
// но ссылки на конфигурацию оборудования у неё нет (HWSprId == 0),
// пользователь перенаправляется на форму создания новой записи
// конфигурации (/sprsrv/hwconfedit/0) с уведомлением в flash.
//
// Маршрут: /sprsrv/hwconfeditbyname/:Name
func (this *MainController) SprHwConfigEditByName() {
	name := this.Ctx.Input.Param(":Name")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeServerDetails) // переход к конфигурации сервера по имени

	flash := beego.NewFlash()

	o := orm.NewOrm()
	var rec models.ServerSprRecords
	err := o.QueryTable("server_spr_records").
		Filter("Name", name).
		Filter("Type__in", 0, 2).
		One(&rec)
	if err != nil || rec.HWSprId <= 0 {
		flash.Notice("Запись конфигурации сервера/гипервизора с именем \"%s\" не найдена. Можно создать новую.", name)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/hwconfedit/0", 302)
		return
	}

	this.Redirect(fmt.Sprintf("/sprsrv/hwconfedit/%d", rec.HWSprId), 302)
}

// Маршрут: /sprsrv/hwconfedit/:id
func (this *MainController) SprHwConfigEdit() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки конфигурации сервера - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("sprsrv/hwconfedit")
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

	var rec models.ServerHardware
	if !isNew {
		if err := o.QueryTable("server_hardware").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["HW"] = rec
			this.Data["DateContractDocStr"] = formatHWConfigDate(rec.DateContractDoc)
			this.Data["DateInvoiceStr"] = formatHWConfigDate(rec.DateInvoice)
			this.Data["DateServiceStr"] = formatHWConfigDate(rec.DateService)
			return
		}
		rec.PowerState, _ = strconv.Atoi(this.GetString("PowerState"))
		rec.Active, _ = strconv.Atoi(this.GetString("Active"))
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.PartNum = this.GetString("PartNum")
		rec.FormFactor, _ = strconv.Atoi(this.GetString("FormFactor"))
		rec.SerialNum = this.GetString("SerialNum")
		rec.VirtualSerialNum = this.GetString("VirtualSerialNum")

		rec.CPUModel = this.GetString("CPUModel")
		rec.CPUCount, _ = strconv.Atoi(this.GetString("CPUCount"))
		rec.CorePerCPU, _ = strconv.Atoi(this.GetString("CorePerCPU"))
		rec.CoreMHz, _ = strconv.Atoi(this.GetString("CoreMHz"))
		rec.MemoryMb, _ = strconv.Atoi(this.GetString("MemoryMb"))

		rec.DatacenterRoom = this.GetString("DatacenterRoom")
		rec.RackNum = this.GetString("RackNum")
		rec.DatacenterName = this.GetString("DatacenterName")
		rec.RackName = this.GetString("RackName")
		rec.StartUnitPos, _ = strconv.Atoi(this.GetString("StartUnitPos"))
		rec.SizeUnit, _ = strconv.Atoi(this.GetString("SizeUnit"))
		rec.RackCabinetId, _ = strconv.ParseInt(this.GetString("RackCabinetId"), 10, 64)

		rec.InventoryNum = this.GetString("InventoryNum")
		rec.InventoryNumBuh = this.GetString("InventoryNumBuh")
		rec.TechnicalNumberIT = this.GetString("TechnicalNumberIT")
		rec.ContractDoc = this.GetString("ContractDoc")
		rec.DateContractDoc = parseHWConfigDate(this.GetString("DateContractDoc"))
		rec.Invoice = this.GetString("Invoice")
		rec.DateInvoice = parseHWConfigDate(this.GetString("DateInvoice"))
		rec.Counterparty = this.GetString("Counterparty")
		rec.ServiceContract = this.GetString("ServiceContract")
		rec.DateService = parseHWConfigDate(this.GetString("DateService"))
		rec.SourceUpdate = this.GetString("SourceUpdate")
		if id == 0 {
			// запись с Id=0 - "шаблон" для новых серверов, в неё SourceUpdate
			// не пишем, чтобы не затирать шаблон при ручном редактировании
			rec.SourceUpdateType = models.SourceUpdateManual
		}
		rec.Time_Mfy = time.Now().Local()

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["HW"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference) // создание записи конфигурации сервера
			this.Redirect(fmt.Sprintf("/sprsrv/hwconfedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["HW"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference) // изменение записи конфигурации сервера

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	// ЦОД/Шкаф для каскадного select размещения (2026-09-08, по запросу
	// пользователя "исправь ... в самой карточке редактирования") - тот же
	// приём, что и в bladesystemsedit.tpl (sprbladesystems.go:SprBladeSystemsEdit)
	// и в фильтре /sprsrv/hwconflist (см. SprHwConfig выше).
	var sites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&sites)
	this.Data["DatacenterSites"] = sites

	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks

	this.Data["HW"] = rec
	this.Data["DateContractDocStr"] = formatHWConfigDate(rec.DateContractDoc)
	this.Data["DateInvoiceStr"] = formatHWConfigDate(rec.DateInvoice)
	this.Data["DateServiceStr"] = formatHWConfigDate(rec.DateService)
}

// parseHWConfigIds разбирает список идентификаторов вида "1,2,3" (значение
// скрытого поля "ids" формы группового редактирования) в срез int64,
// отбрасывая пустые и некорректные фрагменты.
func parseHWConfigIds(s string) []int64 {
	parts := strings.Split(s, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, err := strconv.ParseInt(p, 10, 64); err == nil && v > 0 {
			out = append(out, v)
		}
	}
	return out
}

// SprHwConfigBulkEdit реализует групповое редактирование выделенных на
// /sprsrv/hwconflist записей models.ServerHardware. В отличие от одиночного
// редактирования (SprHwConfigEdit), здесь правится только статус (Active) и
// размещение (DatacenterRoom/RackNum/DatacenterName/RackName/StartUnitPos/
// SizeUnit) - остальные поля у выделенных серверов, как правило, разные и
// группового смысла не имеют.
//
// Каждое поле формы сопровождается отдельным чекбоксом "apply_<Поле>":
// значение поля применяется к выделенным записям только если чекбокс
// отмечен - иначе, если просто применять все поля формы как есть, пустое
// текстовое поле (никогда не заполнявшееся) затирало бы уже сохранённые
// данные у всех выделенных серверов сразу.
//
// Запрос без отметки "ApplyBulk" (первый переход со страницы списка, поле
// "ids" содержит выбранные Id) только показывает форму подтверждения с
// текущими значениями первой из выделенных записей. Запрос с "ApplyBulk"
// выполняет обновление.
//
// Маршрут: /sprsrv/hwconfbulkedit
func (this *MainController) SprHwConfigBulkEdit() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	if this.GetString("ApplyBulk") == "1" {
		RegisterRequest(this, LogTypeEditReference) // групповое изменение записей конфигурации серверов
	} else {
		RegisterRequest(this, LogTypeViewReference) // просмотр формы группового редактирования
	}

	this.activeContent("sprsrv/hwconfbulkedit")

	flash := beego.NewFlash()

	ids := parseHWConfigIds(this.GetString("ids"))
	this.Data["IdsCSV"] = this.GetString("ids")
	this.Data["IdsCount"] = len(ids)

	if len(ids) == 0 {
		flash.Error("Не выбрано ни одной записи для группового редактирования.")
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()

	if this.GetString("ApplyBulk") == "1" && !this.RequireEditReference() {
		// нет прав на сохранение - flash с ошибкой уже положен в RequireEditReference,
		// применять изменения не будем, но форма ниже всё равно отрисуется со
		// значениями первой выделенной записи (см. this.Data["HW"] в конце функции)
	} else if this.GetString("ApplyBulk") == "1" {
		values := orm.Params{}

		if this.GetString("apply_Active") == "on" {
			if v, err := strconv.Atoi(this.GetString("Active")); err == nil {
				values["Active"] = v
			}
		}
		if this.GetString("apply_DatacenterRoom") == "on" {
			values["DatacenterRoom"] = this.GetString("DatacenterRoom")
		}
		if this.GetString("apply_RackNum") == "on" {
			values["RackNum"] = this.GetString("RackNum")
		}
		if this.GetString("apply_DatacenterName") == "on" {
			values["DatacenterName"] = this.GetString("DatacenterName")
		}
		if this.GetString("apply_RackName") == "on" {
			values["RackName"] = this.GetString("RackName")
		}
		if this.GetString("apply_StartUnitPos") == "on" {
			if v, err := strconv.Atoi(this.GetString("StartUnitPos")); err == nil {
				values["StartUnitPos"] = v
			}
		}
		if this.GetString("apply_SizeUnit") == "on" {
			if v, err := strconv.Atoi(this.GetString("SizeUnit")); err == nil {
				values["SizeUnit"] = v
			}
		}

		if len(values) == 0 {
			flash.Error("Не отмечено ни одно поле для изменения - отметьте чекбокс рядом с полем, значение которого нужно применить.")
			flash.Store(&this.Controller)
		} else {
			values["Time_Mfy"] = time.Now().Local()
			// SourceUpdateType намеренно НЕ трогаем здесь - см.
			// kontext/05-conventions-and-gotchas.md (раздел SourceUpdateType,
			// уточнение "массовые формы не проставляют"). Поле помечает
			// записи, внесённые вручную и отсутствующие в
			// VMware/OneView/Zabbix, чтобы код автообновления из этих систем
			// их игнорировал и не портил (например, не проставлял Stat=0/
			// PowerState=2/Active=2 записи, которой в источнике нет).
			// Массовое редактирование через эту форму задевает лишь узкий
			// набор полей у произвольного набора записей - простановка
			// Manual здесь могла бы ошибочно "рассекретить" от
			// автообновления запись, которая на самом деле числится в одной
			// из автоматических систем.
			affected, err := o.QueryTable("server_hardware").Filter("Id__in", ids).Update(values)
			if err != nil {
				flash.Error("Ошибка группового сохранения: %v", err)
				flash.Store(&this.Controller)
			} else {
				flash.Notice("Изменено записей: %d.", affected)
				flash.Store(&this.Controller)
			}
		}
	}

	// значения формы (по умолчанию) берём из первой выделенной записи -
	// исключительно для удобства ввода, к остальным записям применяются
	// только явно отмеченные ("apply_...") поля
	var rec models.ServerHardware
	if err := o.QueryTable("server_hardware").Filter("Id", ids[0]).One(&rec); err == nil {
		this.Data["HW"] = rec
	} else {
		this.Data["HW"] = models.ServerHardware{}
	}
}
