package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/grpcclient"
	"loadmon/models"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	beego "github.com/beego/beego/v2/server/web"
	oneview "github.com/spa-nsk/oneview2"
	"github.com/xuri/excelize/v2"
)

// HWViewFilter — фильтр страницы списка конфигураций серверов OneView.
type HWViewFilter struct {
	DatacenterName  string `form:"DatacenterName"`
	RackName        string `form:"RackName"`
	PowerState      string `form:"PowerState"`
	Status          string `form:"Status"`
	MaintenanceMode string `form:"MaintenanceMode"`
}

// SprHwViewList — список конфигураций серверов HPE OneView (восстановлено из скриншотов).
func (this *MainController) SprHwViewList() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_hwviewlist_"+Id, 302)
		return
	}

	m := sess.(models.LoadFilter).Map
	this.activeContent("sprsrv/hwviewlist")

	keyHW := "HWViewFilter"
	flash := beego.NewFlash()

	hwFilter, ok := m[keyHW].(*HWViewFilter)
	if !ok {
		hwFilter = &HWViewFilter{DatacenterName: "all", RackName: "all", PowerState: "all", Status: "all", MaintenanceMode: "all"}
		m[keyHW] = hwFilter
	}

	if this.Ctx.Input.Method() == "POST" {
		if err := this.ParseForm(hwFilter); err != nil {
			flash.Error("Ошибка распознавания формы: %v", err)
			flash.Store(&this.Controller)
			this.Data[keyHW] = hwFilter
			this.Data["OnView"] = 0
			return
		}
		delete(m, keyHW)
		m[keyHW] = hwFilter
	}

	// Списки значений для выпадающих списков фильтра "Датацентр"/"Шкаф"
	// запрашиваются отдельно от основной выборки (GetHWDatacenterNames/
	// GetHWRackNames, накопленные oneview2.Infrastructure по всем собранным
	// серверам) - раньше эти списки строились по текущему отфильтрованному
	// результату, из-за чего при выборе конкретного датацентра список шкафов
	// схлопывался только до шкафов этого датацентра, и наоборот. Список
	// нужен независимо от того, найдёт ли текущий фильтр хоть одну запись,
	// поэтому запрашивается до early-return по пустому результату.
	if dcNames, err := grpcclient.GetHWDatacenterNames(); err == nil {
		this.Data["Datacenters"] = dcNames
	} else {
		log.Println("GetHWDatacenterNames:", err)
	}
	if rackNames, err := grpcclient.GetHWRackNames(); err == nil {
		this.Data["Racks"] = rackNames
	} else {
		log.Println("GetHWRackNames:", err)
	}

	records, err := grpcclient.GetServerHardwareList(hwFilter.DatacenterName, hwFilter.RackName, hwFilter.Status, hwFilter.MaintenanceMode, hwFilter.PowerState)
	num := len(records)

	if err != nil || num == 0 {
		flash.Error("Нет данных для отображения. Измените фильтр для выборки")
		flash.Store(&this.Controller)
		this.Data[keyHW] = hwFilter
		this.Data["OnView"] = 0
		return
	}

	this.Data["OnView"] = 1

	this.Data["ListId"] = Id

	log.Println("Количество записей", len(records))

	this.Data[keyHW] = hwFilter

	// экспорт отдаёт весь найденный по фильтру список, а не только текущую
	// страницу
	if this.Ctx.Input.Method() == "POST" && len(records) > 0 {
		if str := this.GetString("ExportBTN"); str == "Экспорт в CSV" {
			ExportViewCSV(this, flash, records)
			return
		}
		if str := this.GetString("ExportBTN2"); str == "Экспорт в XLSX" {
			ExportViewXLSX(this, flash, records)
			return
		}
	}

	// grpcclient.GetServerHardwareList не поддерживает постраничную выборку
	// на стороне сервера (инвентарь HPE OneView отдаётся из оперативной
	// памяти, а не из БД, откуда можно было бы делать Limit/Offset как в
	// /sprsrv/hwconflist) - поэтому список сортируется по серийному номеру
	// (для стабильного порядка между запросами) и режется на страницу здесь.
	sort.Slice(records, func(i, j int) bool {
		return records[i].Identity.SerialNumber < records[j].Identity.SerialNumber
	})

	page := 1
	pageSize := 50
	if p, err := strconv.Atoi(this.GetString("page")); err == nil && p > 0 {
		page = p
	}
	if s, err := strconv.Atoi(this.GetString("pagesize")); err == nil {
		switch s {
		case 25, 50, 100, 200:
			pageSize = s
		}
	}

	totalRecords := len(records)
	totalPages := (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > totalRecords {
		end = totalRecords
	}
	pageRecords := records[offset:end]

	this.Data["HWList"] = pageRecords
	this.Data["HWListCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = int64(0)
	if totalRecords > 0 {
		this.Data["PageFrom"] = int64(offset) + 1
	}
	this.Data["PageTo"] = int64(end)
}

func ExportViewCSV(this *MainController, flash *beego.FlashData, recordsView []*oneview.Server) {

	crt := time.Now().Local()
	short_name := fmt.Sprintf("hw%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + "CSV_" + short_name

	csvFile, err := os.Create(destination)
	if err != nil {
		flash.Error("Ошибка создания CSV файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		return
	}
	csvwriter := csv.NewWriter(csvFile)
	csvwriter.Comma = '|'
	header, _ := csvtools.GetCSVHeader(&recordsView[0])
	_ = csvwriter.Write(header)

	for _, rec := range recordsView {
		data, _ := csvtools.GetCSVData(rec)
		_ = csvwriter.Write(data)
	}
	csvwriter.Flush()
	csvFile.Close()

	this.Ctx.Output.Header("Content-Disposition", "attachment; "+"filename="+short_name)
	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Type", "application/csv")
	this.Ctx.Output.Header("Content-Transfer-Encoding", "text")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")

	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}

// ExportViewXLSX отдаёт найденный по фильтру список серверов HPE OneView
// (весь, а не только текущую страницу) файлом XLSX на скачивание -
// формирование листа аналогично экспорту /sprsrv/hwconflist.
func ExportViewXLSX(this *MainController, flash *beego.FlashData, recordsView []*oneview.Server) {
	if len(recordsView) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("hw%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&recordsView[0])
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

	for j, rec := range recordsView {
		row, _ := csvtools.GetCSVData(rec)
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
