package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/grpcclient"
	"loadmon/models"
	"loadmon/oneview"
	"log"
	"net/http"
	"os"
	"time"

	beego "github.com/beego/beego/v2/server/web"
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
	rooms := make(map[string]string)
	racks := make(map[string]string)
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

	records, err := grpcclient.GetServerHardwareList(hwFilter.DatacenterName, hwFilter.RackName, hwFilter.Status, hwFilter.MaintenanceMode, hwFilter.PowerState)
	num := len(records)

	if err != nil || num == 0 {
		flash.Error("Нет данных для отображения. Измените фильтр для выборки")
		flash.Store(&this.Controller)
		this.Data[keyHW] = hwFilter
		this.Data["OnView"] = 0
		return
	}

	for _, rec := range records {
		rooms[rec.Datacenter.Name] = rec.Datacenter.Name
		racks[rec.Rack.Name] = rec.Rack.Name
	}

	this.Data["Datacenters"] = rooms
	this.Data["Racks"] = racks
	this.Data["OnView"] = 1

	this.Data["ListId"] = Id
	this.Data["HWList"] = records
	this.Data["HWListCount"] = len(records)

	log.Println("Количество записей", len(records))

	this.Data[keyHW] = hwFilter

	if this.Ctx.Input.Method() == "POST" && len(records) > 0 {
		if str := this.GetString("ExportBTN"); str == "Экспорт в CSV" {
			ExportViewCSV(this, flash, records)
			return
		}
	}
}

func ExportViewCSV(this *MainController, flash *beego.FlashData, recordsView []*oneview.ServerHardware) {

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
