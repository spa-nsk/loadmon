package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/models"
	"loadmon/rackpower"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// RackCabinetFilter - фильтр списка /sprsrv/rackcabinetlist. Простой
// текстовый фильтр, приём идентичен DatacenterSiteFilter
// (controllers/sprdatacenter.go). Экспорт в CSV/XLSX добавлен 2026-09-21 по
// запросу пользователя ("добавь выгрузку как во всех справочниках"), по
// образцу controllers/sprbladesystems.go.
type RackCabinetFilter struct {
	// LocalityCode - код ОКТМО населённого пункта (подсказка как в
	// /sprsrv/datacentersitelist); сужает список через площадки с этим кодом.
	LocalityCode     string `form:"LocalityCode"`
	DatacenterSiteId string `form:"DatacenterSiteId"` // площадка/ЦОД (Id)
	Name             string `form:"Name"`
	AltName          string `form:"AltName"`
	// ShowUnnamed == "1" - показывать и шкафы, у которых не заполнены оба
	// наименования (Name и AltName); по умолчанию такие записи скрыты.
	ShowUnnamed string `form:"ShowUnnamed"`
}

func defaultRackCabinetFilter() *RackCabinetFilter {
	return &RackCabinetFilter{}
}

// RackCabinetRow - строка списка шкафов, дополненная наименованием ЦОД для
// отображения (DatacenterSite.Name по DatacenterSiteId).
type RackCabinetRow struct {
	models.RackCabinet
	DatacenterSiteName string
	PowerAvg24hW       float64
	PowerPeak24hW      float64
	PowerRatedMaxW     float64
}

// SprRackCabinet отображает список записей справочника шкафов
// (models.RackCabinet, таблица cmdb_rack_cabinet) - новый справочник,
// заведённый по запросу пользователя "Шкаф в ЦОД аналогично Rack из oneview".
//
// Маршрут: /sprsrv/rackcabinetlist/:id
func (this *MainController) SprRackCabinet() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/rackcabinetlist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "RackCabinetFilter"
	filter, ok := m[keyFilter].(*RackCabinetFilter)
	if !ok {
		filter = defaultRackCabinetFilter()
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
		// чекбокс при снятой галочке в форме не передаётся вовсе, а ParseForm
		// не сбрасывает поля, отсутствующие в запросе - обнуляем явно
		filter.ShowUnnamed = this.GetString("ShowUnnamed")
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

	// Справочник площадок для select «Площадка/ЦОД» (каскад по LocalityCode на клиенте).
	var allSites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&allSites, "Id", "Name", "LocalityCode")
	this.Data["DatacenterSites"] = allSites

	qs := o.QueryTable("cmdb_rack_cabinet")
	if filter.ShowUnnamed != "1" {
		// Пробелы заполнением не считаются - TRIM средствами ORM не
		// выразить, поэтому id "пустых" шкафов берутся сырым запросом.
		var emptyIds []int64
		if _, err := o.Raw("SELECT id FROM cmdb_rack_cabinet WHERE (name IS NULL OR TRIM(name) = '') AND (alt_name IS NULL OR TRIM(alt_name) = '')").QueryRows(&emptyIds); err != nil {
			flash.Notice("Не удалось определить шкафы без наименований: %v", err)
		} else if len(emptyIds) > 0 {
			qs = qs.Exclude("Id__in", emptyIds)
		}
	}
	filter.LocalityCode = strings.TrimSpace(filter.LocalityCode)
	if filter.LocalityCode != "" {
		this.Data["FilterLocalityName"] = localityNamesByCode(o, []models.DatacenterSite{{LocalityCode: filter.LocalityCode}})[filter.LocalityCode]
	}
	// Площадка/ЦОД: точный Id; иначе при выбранном населённом пункте - все
	// шкафы площадок с этим LocalityCode.
	if filter.DatacenterSiteId != "" && filter.DatacenterSiteId != "0" {
		if dcid, err := strconv.ParseInt(filter.DatacenterSiteId, 10, 64); err == nil && dcid > 0 {
			qs = qs.Filter("DatacenterSiteId", dcid)
		}
	} else if filter.LocalityCode != "" {
		siteIds := make([]int64, 0)
		for _, s := range allSites {
			if s.LocalityCode == filter.LocalityCode {
				siteIds = append(siteIds, s.Id)
			}
		}
		if len(siteIds) == 0 {
			qs = qs.Filter("Id", int64(-1)) // пустой результат
		} else {
			qs = qs.Filter("DatacenterSiteId__in", siteIds)
		}
	}
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.AltName != "" {
		qs = qs.Filter("AltName__icontains", filter.AltName)
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
	// controllers/sprbladesystems.go:SprBladeSystems. Экспортируются
	// "плоские" models.RackCabinet (с csv-тегами), а не обёрнутые
	// RackCabinetRow ниже - DatacenterSiteName в CSV/XLSX не нужен, т.к.
	// там уже есть RackCabinet.DatacenterSiteId.
	records := make([]models.RackCabinet, 0, pageSize)
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
		this.exportRackCabinetXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportRackCabinetCSV(records, flash)
		return
	}

	dcIds := make([]int64, 0, len(records))
	for _, r := range records {
		if r.DatacenterSiteId != 0 {
			dcIds = append(dcIds, r.DatacenterSiteId)
		}
	}
	dcNameById := make(map[int64]string, len(dcIds))
	if len(dcIds) > 0 {
		var dcs []models.DatacenterSite
		if _, err := o.QueryTable("cmdb_datacenter_site").Filter("Id__in", dcIds).All(&dcs); err == nil {
			for _, dc := range dcs {
				dcNameById[dc.Id] = dc.Name
			}
		}
	}

	rackIds := make([]int64, 0, len(records))
	for _, r := range records {
		if r.Id > 0 {
			rackIds = append(rackIds, r.Id)
		}
	}
	powerByRack := rackpower.RackCabinetPowerTotalsBatch(o, rackIds)

	rows := make([]RackCabinetRow, 0, len(records))
	for _, r := range records {
		p := powerByRack[r.Id]
		rows = append(rows, RackCabinetRow{
			RackCabinet:        r,
			DatacenterSiteName: dcNameById[r.DatacenterSiteId],
			PowerAvg24hW:       p.Avg24hW,
			PowerPeak24hW:      p.Peak24hW,
			PowerRatedMaxW:     p.RatedMaxW,
		})
	}

	this.Data["RackCabinetList"] = rows
	this.Data["RackCabinetCount"] = totalRecords
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

// SprRackCabinetEdit отображает форму просмотра/редактирования одной записи
// models.RackCabinet по Id и сохраняет изменения по отправке формы (POST).
// Id=0 - режим добавления новой записи.
//
// Маршрут: /sprsrv/rackcabinetedit/:id
func (this *MainController) SprRackCabinetEdit() {

	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/rackcabinetedit")
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

	var rec models.RackCabinet
	if !isNew {
		if err := o.QueryTable("cmdb_rack_cabinet").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["RC"] = rec
			return
		}
		rec.Name = this.GetString("Name")
		rec.AltName = this.GetString("AltName")
		rec.DatacenterSiteId, _ = strconv.ParseInt(this.GetString("DatacenterSiteId"), 10, 64)
		rec.Model = this.GetString("Model")
		rec.SerialNumber = this.GetString("SerialNumber")
		rec.Width, _ = strconv.ParseInt(this.GetString("Width"), 10, 64)
		rec.Depth, _ = strconv.ParseInt(this.GetString("Depth"), 10, 64)
		rec.Height, _ = strconv.ParseInt(this.GetString("Height"), 10, 64)
		rec.UHeight, _ = strconv.ParseInt(this.GetString("UHeight"), 10, 64)
		rec.Annotation = this.GetString("Annotation")
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
				this.Data["RC"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.Redirect(fmt.Sprintf("/sprsrv/rackcabinetedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["RC"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference)

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	var sites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&sites)
	this.Data["DatacenterSites"] = sites

	this.Data["RC"] = rec
}

// exportRackCabinetXLSX формирует и отдаёт клиенту XLSX-файл с записями
// справочника шкафов, отобранными текущим фильтром - по образцу
// controllers/sprbladesystems.go:exportBladeSystemsXLSX.
func (this *MainController) exportRackCabinetXLSX(rows []models.RackCabinet, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("rackcabinet_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Шкафы"
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

// exportRackCabinetCSV формирует и отдаёт клиенту CSV-файл (разделитель "|")
// с записями справочника шкафов, отобранными текущим фильтром - по образцу
// controllers/sprbladesystems.go:exportBladeSystemsCSV.
func (this *MainController) exportRackCabinetCSV(rows []models.RackCabinet, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("rackcabinet_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
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
