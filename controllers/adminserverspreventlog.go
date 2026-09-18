package controllers

import (
	"fmt"
	"log"
	models "loadmon/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// serverSprEventTypeLabel переводит код models.ServerSprEventLog.EventType в
// текст для отображения в журнале (см. models/models_add.go).
func serverSprEventTypeLabel(t int) string {
	switch t {
	case models.ServerSprEventCreated:
		return "Карточка создана"
	case models.ServerSprEventFieldChanged:
		return "Изменилось поле"
	default:
		return fmt.Sprintf("Тип %d", t)
	}
}

// ViewServerSprEventLog отображает журнал ручных изменений карточек
// серверов (таблица server_spr_event_log: создание записи и изменение
// отслеживаемых полей формы /sprsrv/updatebase, с автором и датой - см.
// cmdb/serversprchangelog.go) с фильтром по типу события, конкретному
// серверу (?server=Id, используется ссылкой "История изменений карточки" из
// самой карточки - server_web_new/views/sprsrv/updatebase.tpl) и диапазону
// дат, постраничной навигацией и экспортом в XLSX. Как и /admin/
// hardwareeventlog, доступен только пользователям с models.AuthUser.Priv
// == 3 (администратор). В отличие от HardwareEventLog email-уведомления НЕ
// отправляются (прямое указание пользователя) и хранится дольше -
// auditlog.ServerSprEventLogRetention (180 суток), поскольку это аудит именно
// РУЧНЫХ правок с автором, а не каждый цикл автосинхронизации.
//
// Маршрут: /admin/serverspreventlog
func (this *MainController) ViewServerSprEventLog() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")

	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/serverspreventlog")
	flash := beego.NewFlash()

	typeFilter := this.GetString("type")
	if typeFilter == "" {
		typeFilter = "all"
	}
	this.Data["CurType"] = typeFilter
	typeCodes := []int{models.ServerSprEventCreated, models.ServerSprEventFieldChanged}
	typeOptions := make([]UserLogTypeOption, 0, len(typeCodes))
	for _, t := range typeCodes {
		typeOptions = append(typeOptions, UserLogTypeOption{Value: t, Label: serverSprEventTypeLabel(t)})
	}
	this.Data["TypeList"] = typeOptions

	// Фильтр по конкретному серверу - используется ссылкой "История
	// изменений карточки" прямо со страницы /sprsrv/updatebase.
	serverIdFilter, _ := strconv.ParseInt(this.GetString("server"), 10, 64)
	this.Data["CurServerId"] = serverIdFilter

	fromDateStr := this.GetString("from")
	toDateStr := this.GetString("to")
	this.Data["FromDate"] = fromDateStr
	this.Data["ToDate"] = toDateStr

	var fromDate, toDate time.Time
	var hasFrom, hasTo bool
	if fromDateStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", fromDateStr, time.Local); err == nil {
			fromDate = t
			hasFrom = true
		} else {
			this.Data["FromDate"] = ""
		}
	}
	if toDateStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", toDateStr, time.Local); err == nil {
			toDate = t
			hasTo = true
		} else {
			this.Data["ToDate"] = ""
		}
	}
	if hasFrom && hasTo && toDate.Before(fromDate) {
		fromDate, toDate = toDate, fromDate
		this.Data["FromDate"], this.Data["ToDate"] = this.Data["ToDate"], this.Data["FromDate"]
	}

	o := orm.NewOrm()

	buildQuery := func() orm.QuerySeter {
		qs := o.QueryTable("server_spr_event_log")
		if typeFilter != "all" {
			if t, err := strconv.Atoi(typeFilter); err == nil {
				qs = qs.Filter("EventType", t)
			}
		}
		if serverIdFilter > 0 {
			qs = qs.Filter("ServerId", serverIdFilter)
		}
		if hasFrom {
			qs = qs.Filter("Time__gte", fromDate)
		}
		if hasTo {
			qs = qs.Filter("Time__lt", toDate.AddDate(0, 0, 1))
		}
		return qs
	}
	buildOrderedQuery := func() orm.QuerySeter {
		return buildQuery().OrderBy("-Time")
	}

	if this.GetString("export") == "xlsx" {
		RegisterRequest(this, LogTypeExport)
		var records []models.ServerSprEventLog
		if _, err := buildOrderedQuery().All(&records); err != nil {
			flash.Error("Ошибка загрузки журнала для экспорта: %v", err)
			flash.Store(&this.Controller)
			return
		}
		ExportServerSprEventLogXLSX(this, flash, records)
		return
	}

	RegisterRequest(this, LogTypeAdminAction) // просмотр журнала изменений карточек серверов

	pageSize, _ := strconv.Atoi(this.GetString("pagesize"))
	switch pageSize {
	case 25, 50, 100, 200:
	default:
		pageSize = 50
	}
	page, _ := strconv.Atoi(this.GetString("page"))
	if page < 1 {
		page = 1
	}

	totalRecords, err := buildQuery().Count()
	if err != nil {
		log.Println("ViewServerSprEventLog: ошибка подсчёта записей журнала (server_spr_event_log):", err)
		flash.Error("Ошибка чтения журнала изменений карточек серверов: %v", err)
		flash.Store(&this.Controller)
	}
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	var records []models.ServerSprEventLog
	if _, err := buildOrderedQuery().Limit(pageSize, offset).All(&records); err != nil {
		log.Println("ViewServerSprEventLog: ошибка выборки записей журнала (server_spr_event_log):", err)
		flash.Error("Ошибка чтения журнала изменений карточек серверов: %v", err)
		flash.Store(&this.Controller)
	}

	rows := make([]struct {
		models.ServerSprEventLog
		TypeLabel string
	}, 0, len(records))
	for _, r := range records {
		rows = append(rows, struct {
			models.ServerSprEventLog
			TypeLabel string
		}{ServerSprEventLog: r, TypeLabel: serverSprEventTypeLabel(r.EventType)})
	}

	this.Data["Rows"] = rows
	this.Data["TotalRecords"] = totalRecords
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

// serverSprEventLogXLSXHeader - заголовок листа XLSX-экспорта /admin/serverspreventlog.
var serverSprEventLogXLSXHeader = []string{"Id", "Время", "Тип события", "ServerId", "Сервер", "Поле", "Было", "Стало", "Автор", "Сообщение"}

// ExportServerSprEventLogXLSX отдаёт найденный по фильтру журнал изменений
// карточек серверов (весь, а не только текущую страницу) файлом XLSX на
// скачивание - формирование листа аналогично /admin/hardwareeventlog.
func ExportServerSprEventLogXLSX(this *MainController, flash *beego.FlashData, records []models.ServerSprEventLog) {
	if len(records) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("serverspreventlog%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	sheet := "Журнал карточек серверов"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &serverSprEventLogXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j, r := range records {
		row := []string{
			strconv.FormatInt(r.Id, 10),
			r.Time.Format("2006-01-02 15:04:05"),
			serverSprEventTypeLabel(r.EventType),
			strconv.FormatInt(r.ServerId, 10),
			r.Key,
			r.FieldName,
			r.OldValue,
			r.NewValue,
			r.UserName,
			r.Message,
		}
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
