package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/rackpower"
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

// DatacenterSiteFilter - фильтр списка /sprsrv/datacentersitelist. Простой
// текстовый фильтр по Name/AltName. Экспорт в CSV/XLSX добавлен 2026-09-21
// по запросу пользователя ("добавь выгрузку как во всех справочниках"), по
// образцу controllers/sprbladesystems.go - структура/приём пагинации
// идентичны BladeSystemsFilter там же.
type DatacenterSiteFilter struct {
	Name     string `form:"Name"`
	AltName  string `form:"AltName"`
	CodeJira string `form:"CodeJira"` // 2026-10-02: фильтр по коду Jira (вхождение, без учёта регистра)
}

func defaultDatacenterSiteFilter() *DatacenterSiteFilter {
	return &DatacenterSiteFilter{}
}

// DatacenterSiteRow — строка списка ЦОД с агрегированной мощностью (вариант C по шкафам).
type DatacenterSiteRow struct {
	models.DatacenterSite
	PowerAvg24hW   float64
	PowerPeak24hW  float64
	PowerRatedMaxW float64
}

// SprDatacenterSite отображает список записей справочника ЦОД
// (models.DatacenterSite, таблица cmdb_datacenter_site) - новый справочник,
// заведённый по запросу пользователя "структуры описывающие ЦОД по аналогии
// с Datacenter из oneview2" (см. models.DatacenterSite).
//
// Маршрут: /sprsrv/datacentersitelist/:id (:id зарезервирован под будущее
// использование, как и в аналогичных справочниках).
func (this *MainController) SprDatacenterSite() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/datacentersitelist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "DatacenterSiteFilter"
	filter, ok := m[keyFilter].(*DatacenterSiteFilter)
	if !ok {
		filter = defaultDatacenterSiteFilter()
		m[keyFilter] = filter
	}

	page := 1
	pageSize := 50
	exportCSV := false
	exportXLSX := false
	bulkEdit := false

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
		bulkEdit = this.GetString("SetBaseBTN") == "Редактировать выделенные"
	}

	// Групповое редактирование (2026-10-02) - по образцу /sprsrv/view2:
	// отмеченные чекбоксами chSelection<Id> строки передаются в
	// /sprsrv/datacentersiteupdateselected/<id-id-id>.
	if bulkEdit {
		ids := make([]string, 0)
		for _, s := range this.GetStrings("chSelection") {
			if id, err := strconv.ParseInt(s, 10, 64); err == nil && id > 0 {
				ids = append(ids, strconv.FormatInt(id, 10))
			}
		}
		if len(ids) > 0 {
			this.Redirect("/sprsrv/datacentersiteupdateselected/"+strings.Join(ids, "-"), 302)
			return
		}
		flash.Error("Выберите хотя бы одну площадку")
		flash.Store(&this.Controller)
	}

	this.Data[keyFilter] = filter

	o := orm.NewOrm()
	qs := o.QueryTable("cmdb_datacenter_site")
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.AltName != "" {
		qs = qs.Filter("AltName__icontains", filter.AltName)
	}
	if filter.CodeJira != "" {
		qs = qs.Filter("CodeJira__icontains", filter.CodeJira)
	}

	totalRecords, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	// Для экспорта выгружается ВЕСЬ набор, отобранный фильтром (без
	// постраничного Limit/Offset) - тот же приём, что в
	// controllers/sprbladesystems.go:SprBladeSystems.
	records := make([]models.DatacenterSite, 0, pageSize)
	if exportCSV || exportXLSX {
		if _, err := qs.OrderBy("Name").All(&records); err != nil {
			flash.Error("Ошибка выборки данных: %v", err)
			flash.Store(&this.Controller)
			return
		}
	} else {
		if _, err := qs.OrderBy("Name").Limit(pageSize, offset).All(&records); err != nil {
			flash.Error("Ошибка выборки данных: %v", err)
			flash.Store(&this.Controller)
			return
		}
	}

	if exportXLSX {
		this.exportDatacenterSiteXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportDatacenterSiteCSV(records, flash)
		return
	}

	siteIds := make([]int64, 0, len(records))
	for _, r := range records {
		if r.Id > 0 {
			siteIds = append(siteIds, r.Id)
		}
	}
	powerBySite := rackpower.DatacenterSitePowerTotalsBatch(o, siteIds)
	rows := make([]DatacenterSiteRow, 0, len(records))
	for _, r := range records {
		p := powerBySite[r.Id]
		rows = append(rows, DatacenterSiteRow{
			DatacenterSite: r,
			PowerAvg24hW:   p.Avg24hW,
			PowerPeak24hW:  p.Peak24hW,
			PowerRatedMaxW: p.RatedMaxW,
		})
	}

	this.Data["DatacenterSiteList"] = rows
	this.Data["CanEdit"] = this.CanEditReference()
	this.Data["DatacenterSiteCount"] = totalRecords
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

// SprDatacenterSiteEdit отображает форму просмотра/редактирования одной
// записи models.DatacenterSite по Id и сохраняет изменения по отправке формы
// (POST). Id=0 - режим добавления новой записи.
//
// Маршрут: /sprsrv/datacentersiteedit/:id
func (this *MainController) SprDatacenterSiteEdit() {

	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/datacentersiteedit")
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

	var rec models.DatacenterSite
	if !isNew {
		if err := o.QueryTable("cmdb_datacenter_site").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	} else {
		// По умолчанию новая площадка считается ЦОД (типичный случай) -
		// чекбокс в форме создания сразу отмечен, поле IsDatacenter
		// добавлено в модель 2026-09-10.
		rec.IsDatacenter = true
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["DC"] = rec
			return
		}
		rec.Name = this.GetString("Name")
		rec.AltName = this.GetString("AltName")
		// IsDatacenter - чекбокс в форме; несданный чекбокс не приходит в
		// POST вовсе (стандартное поведение HTML-форм), поэтому его
		// отсутствие в этом запросе однозначно означает "снят" - тот же
		// приём, что и для прочих bool-полей проекта, отправляемых через
		// <input type="checkbox">.
		rec.IsDatacenter = this.GetString("IsDatacenter") != ""
		rec.Address = strings.TrimSpace(this.GetString("Address"))
		rec.Floor = strings.TrimSpace(this.GetString("Floor"))
		rec.Width, _ = strconv.ParseInt(this.GetString("Width"), 10, 64)
		rec.Height, _ = strconv.ParseInt(this.GetString("Height"), 10, 64)
		rec.Annotation = this.GetString("Annotation")
		rec.CodeJira = this.GetString("CodeJira")
		var currentUserId int64
		if uid, ok := this.Data["UserId"].(int64); ok {
			currentUserId = uid
		}
		rec.Id_User = currentUserId
		rec.Time_Mfy = time.Now().Local()

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["DC"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.Redirect(fmt.Sprintf("/sprsrv/datacentersiteedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["DC"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference)

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	this.Data["DC"] = rec
}

// exportDatacenterSiteXLSX формирует и отдаёт клиенту XLSX-файл с записями
// справочника ЦОД, отобранными текущим фильтром - по образцу
// controllers/sprbladesystems.go:exportBladeSystemsXLSX.
func (this *MainController) exportDatacenterSiteXLSX(rows []models.DatacenterSite, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("datacentersite_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Площадки-ЦОД"
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
		flash.Error("Ошибка чтения файла для отдачи: "+destination+" %v", err)
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

// exportDatacenterSiteCSV формирует и отдаёт клиенту CSV-файл (разделитель
// "|") с записями справочника ЦОД, отобранными текущим фильтром - по
// образцу controllers/sprbladesystems.go:exportBladeSystemsCSV.
func (this *MainController) exportDatacenterSiteCSV(rows []models.DatacenterSite, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("datacentersite_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
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
