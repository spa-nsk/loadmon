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

// hardwareEventTypeLabel переводит код models.HardwareEventLog.EventType в
// текст для отображения в журнале (см. models/models_add.go).
func hardwareEventTypeLabel(t int) string {
	switch t {
	case models.HardwareEventComponentAdded:
		return "Добавлен компонент"
	case models.HardwareEventComponentRemoved:
		return "Компонент пропал"
	case models.HardwareEventComponentChanged:
		return "Компонент изменился"
	case models.HardwareEventFieldChanged:
		return "Изменилось поле"
	default:
		return fmt.Sprintf("Тип %d", t)
	}
}

// HardwareEventLogRow - одна строка журнала изменений аппаратной
// конфигурации (models.HardwareEventLog), дополненная кратким
// наименованием родительского сервера/корзины для отображения (в самой
// таблице hardware_event_log хранится только ServerId/BladeSystemId).
type HardwareEventLogRow struct {
	models.HardwareEventLog
	ParentName string
	TypeLabel  string
}

// ViewHardwareEventLog отображает журнал изменений аппаратной конфигурации
// серверов и блейд-корзин (таблица hardware_event_log: добавление/удаление/
// изменение компонентов и изменение скалярных полей ServerHardware/
// BladeSystems - см. cmdb/hardwareeventlog.go) с фильтром по типу события,
// сущности (сервер/корзина) и диапазону дат, постраничной навигацией и
// экспортом в XLSX. Как и /admin/storageeventlog, доступен только
// пользователям с models.AuthUser.Priv == 3 (администратор). Хранится
// ограниченное время (cmdb.HardwareEventLogRetention, 30 суток).
//
// Маршрут: /admin/hardwareeventlog
func (this *MainController) ViewHardwareEventLog() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")

	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/hardwareeventlog")
	flash := beego.NewFlash()

	typeFilter := this.GetString("type")
	if typeFilter == "" {
		typeFilter = "all"
	}
	this.Data["CurType"] = typeFilter
	typeCodes := []int{models.HardwareEventComponentAdded, models.HardwareEventComponentRemoved, models.HardwareEventComponentChanged, models.HardwareEventFieldChanged}
	typeOptions := make([]UserLogTypeOption, 0, len(typeCodes))
	for _, t := range typeCodes {
		typeOptions = append(typeOptions, UserLogTypeOption{Value: t, Label: hardwareEventTypeLabel(t)})
	}
	this.Data["TypeList"] = typeOptions

	// Фильтр по сущности - "server"/"blade"/"all" (по тому, какое из полей
	// ServerId/BladeSystemId заполнено в записи журнала).
	entityFilter := this.GetString("entity")
	if entityFilter == "" {
		entityFilter = "all"
	}
	this.Data["CurEntity"] = entityFilter

	// Фильтр по диапазону дат (по полю Time) - тот же приём, что и
	// /admin/storageeventlog (controllers/adminstorageeventlog.go).
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

	// buildQuery - см. комментарий в ViewStorageEventLog про то, почему
	// ORDER BY нельзя примешивать к запросу, используемому для Count().
	buildQuery := func() orm.QuerySeter {
		qs := o.QueryTable("hardware_event_log")
		if typeFilter != "all" {
			if t, err := strconv.Atoi(typeFilter); err == nil {
				qs = qs.Filter("EventType", t)
			}
		}
		switch entityFilter {
		case "server":
			qs = qs.Filter("ServerId__gt", 0)
		case "blade":
			qs = qs.Filter("BladeSystemId__gt", 0)
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

	// экспорт в XLSX отдаёт весь найденный по фильтру журнал (без
	// постраничной нарезки) - тот же приём, что и /admin/storageeventlog.
	if this.GetString("export") == "xlsx" {
		RegisterRequest(this, LogTypeExport)
		var records []models.HardwareEventLog
		if _, err := buildOrderedQuery().All(&records); err != nil {
			flash.Error("Ошибка загрузки журнала для экспорта: %v", err)
			flash.Store(&this.Controller)
			return
		}
		rows := this.buildHardwareEventLogRows(o, records)
		ExportHardwareEventLogXLSX(this, flash, rows)
		return
	}

	RegisterRequest(this, LogTypeAdminAction) // просмотр журнала изменений аппаратной конфигурации

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
		log.Println("ViewHardwareEventLog: ошибка подсчёта записей журнала (hardware_event_log):", err)
		flash.Error("Ошибка чтения журнала изменений конфигурации: %v", err)
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

	var records []models.HardwareEventLog
	if _, err := buildOrderedQuery().Limit(pageSize, offset).All(&records); err != nil {
		log.Println("ViewHardwareEventLog: ошибка выборки записей журнала (hardware_event_log):", err)
		flash.Error("Ошибка чтения журнала изменений конфигурации: %v", err)
		flash.Store(&this.Controller)
	}

	rows := this.buildHardwareEventLogRows(o, records)

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

// buildHardwareEventLogRows дополняет записи журнала (models.HardwareEventLog)
// кратким наименованием родительского сервера/корзины (одним IN-запросом
// сразу по всем встретившимся Id, а не по одному на запись) и текстовой
// расшифровкой типа события.
func (this *MainController) buildHardwareEventLogRows(o orm.Ormer, records []models.HardwareEventLog) []HardwareEventLogRow {
	serverIdSet := make(map[int64]struct{})
	bladeIdSet := make(map[int64]struct{})
	for _, r := range records {
		if r.ServerId != 0 {
			serverIdSet[r.ServerId] = struct{}{}
		}
		if r.BladeSystemId != 0 {
			bladeIdSet[r.BladeSystemId] = struct{}{}
		}
	}
	serverNameById := make(map[int64]string, len(serverIdSet))
	if len(serverIdSet) > 0 {
		ids := make([]int64, 0, len(serverIdSet))
		for id := range serverIdSet {
			ids = append(ids, id)
		}
		var servers []models.ServerHardware
		if _, err := o.QueryTable("server_hardware").Filter("Id__in", ids).All(&servers, "Id", "SerialNum", "Model", "ServerId"); err == nil {
			srvIds := make([]int64, 0, len(servers))
			for _, s := range servers {
				if s.ServerId != 0 {
					srvIds = append(srvIds, s.ServerId)
				}
			}
			sprNameById := make(map[int64]string, len(srvIds))
			if len(srvIds) > 0 {
				var sprs []models.ServerSprRecords
				if _, err := models.ServerSprQS(o).Filter("Id__in", srvIds).All(&sprs, "Id", "Name"); err == nil {
					for _, s := range sprs {
						sprNameById[s.Id] = s.Name
					}
				}
			}
			for _, s := range servers {
				name := sprNameById[s.ServerId]
				if name == "" {
					name = s.Model
				}
				if name == "" {
					name = s.SerialNum
				}
				serverNameById[s.Id] = name
			}
		}
	}
	bladeNameById := make(map[int64]string, len(bladeIdSet))
	if len(bladeIdSet) > 0 {
		ids := make([]int64, 0, len(bladeIdSet))
		for id := range bladeIdSet {
			ids = append(ids, id)
		}
		var blades []models.BladeSystems
		if _, err := o.QueryTable("blade_systems").Filter("Id__in", ids).All(&blades, "Id", "Name"); err == nil {
			for _, b := range blades {
				bladeNameById[b.Id] = b.Name
			}
		}
	}

	rows := make([]HardwareEventLogRow, 0, len(records))
	for _, r := range records {
		row := HardwareEventLogRow{HardwareEventLog: r}
		if r.ServerId != 0 {
			row.ParentName = serverNameById[r.ServerId]
		} else if r.BladeSystemId != 0 {
			row.ParentName = bladeNameById[r.BladeSystemId]
		}
		if row.ParentName == "" {
			row.ParentName = r.Key
		}
		row.TypeLabel = hardwareEventTypeLabel(r.EventType)
		rows = append(rows, row)
	}
	return rows
}

// hardwareEventLogXLSXHeader - заголовок листа XLSX-экспорта /admin/hardwareeventlog.
var hardwareEventLogXLSXHeader = []string{"Id", "Время", "Тип события", "Сущность", "Сервер/Корзина", "Ключ", "Поле", "Было", "Стало", "Сообщение"}

// hardwareEventLogXLSXRow формирует одну строку XLSX-экспорта из HardwareEventLogRow.
func hardwareEventLogXLSXRow(r HardwareEventLogRow) []string {
	return []string{
		strconv.FormatInt(r.Id, 10),
		r.Time.Format("2006-01-02 15:04:05"),
		r.TypeLabel,
		r.EntityKind,
		r.ParentName,
		r.Key,
		r.FieldName,
		r.OldValue,
		r.NewValue,
		r.Message,
	}
}

// ExportHardwareEventLogXLSX отдаёт найденный по фильтру журнал изменений
// аппаратной конфигурации (весь, а не только текущую страницу) файлом XLSX
// на скачивание - формирование листа аналогично /admin/storageeventlog.
func ExportHardwareEventLogXLSX(this *MainController, flash *beego.FlashData, rows []HardwareEventLogRow) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("hardwareeventlog%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	sheet := "Журнал изменений"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &hardwareEventLogXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j, r := range rows {
		row := hardwareEventLogXLSXRow(r)
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
