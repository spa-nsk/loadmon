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

// storageEventTypeLabel переводит код models.StorageEventLog.EventType в
// текст для отображения в журнале (см. models/models_add.go).
func storageEventTypeLabel(t int) string {
	switch t {
	case models.StorageEventAvailability:
		return "Доступность СХД"
	case models.StorageEventUnknownLun:
		return "Неизвестный LUN"
	default:
		return fmt.Sprintf("Тип %d", t)
	}
}

// StorageEventLogRow - одна строка журнала событий СХД (models.StorageEventLog),
// дополненная кратким наименованием СХД для отображения (в самой таблице
// storage_event_log хранится только StorageSystemsId).
type StorageEventLogRow struct {
	models.StorageEventLog
	StorageSystemShortName string
	TypeLabel              string
}

// ViewStorageEventLog отображает журнал событий СХД (таблица
// storage_event_log: доступность каждого опроса СХД и обнаружение LUN, не
// привязанных ни к одной известной СХД - см. cmdb/storageeventlog.go) с
// фильтром по типу события и диапазону дат, постраничной навигацией и
// экспортом в XLSX. Как и /admin/userlog, доступен только пользователям с
// models.AuthUser.Priv == 3 (администратор). Хранится ограниченное время
// (cmdb.StorageEventLogRetention, 3 суток) - за более давние периоды
// записей в журнале уже не будет.
//
// Маршрут: /admin/storageeventlog
func (this *MainController) ViewStorageEventLog() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")

	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/storageeventlog")
	flash := beego.NewFlash()

	typeFilter := this.GetString("type")
	if typeFilter == "" {
		typeFilter = "all"
	}
	this.Data["CurType"] = typeFilter
	typeCodes := []int{models.StorageEventAvailability, models.StorageEventUnknownLun}
	typeOptions := make([]UserLogTypeOption, 0, len(typeCodes))
	for _, t := range typeCodes {
		typeOptions = append(typeOptions, UserLogTypeOption{Value: t, Label: storageEventTypeLabel(t)})
	}
	this.Data["TypeList"] = typeOptions

	// Фильтр "только ошибки/недоступность" - имеет смысл только для событий
	// StorageEventAvailability (OK=false); для StorageEventUnknownLun поле OK
	// не используется, фильтр на такие записи не влияет.
	onlyFail := this.GetString("onlyfail") == "1"
	this.Data["OnlyFail"] = onlyFail

	// Фильтр по диапазону дат (по полю Time) - тот же приём, что и
	// /admin/userlog (controllers/userlog.go:ViewUserLog): обе границы
	// необязательны, верхняя граница включительна по дате (Time__lt на
	// ToDate+1 сутки).
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

	// buildQuery - см. комментарий в ViewUserLog (controllers/userlog.go) про
	// то, почему ORDER BY нельзя примешивать к запросу, используемому для
	// Count().
	buildQuery := func() orm.QuerySeter {
		qs := o.QueryTable("storage_event_log")
		if typeFilter != "all" {
			if t, err := strconv.Atoi(typeFilter); err == nil {
				qs = qs.Filter("EventType", t)
			}
		}
		if onlyFail {
			qs = qs.Filter("OK", false)
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

	// экспорт в XLSX отдаёт весь найденный по фильтру журнал (без постраничной
	// нарезки) - тот же приём, что и /admin/userlog
	if this.GetString("export") == "xlsx" {
		RegisterRequest(this, LogTypeExport)
		var records []models.StorageEventLog
		if _, err := buildOrderedQuery().All(&records); err != nil {
			flash.Error("Ошибка загрузки журнала для экспорта: %v", err)
			flash.Store(&this.Controller)
			return
		}
		rows := this.buildStorageEventLogRows(o, records)
		ExportStorageEventLogXLSX(this, flash, rows)
		return
	}

	RegisterRequest(this, LogTypeAdminAction) // просмотр журнала событий СХД

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
		log.Println("ViewStorageEventLog: ошибка подсчёта записей журнала (storage_event_log):", err)
		flash.Error("Ошибка чтения журнала событий СХД: %v", err)
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

	var records []models.StorageEventLog
	if _, err := buildOrderedQuery().Limit(pageSize, offset).All(&records); err != nil {
		log.Println("ViewStorageEventLog: ошибка выборки записей журнала (storage_event_log):", err)
		flash.Error("Ошибка чтения журнала событий СХД: %v", err)
		flash.Store(&this.Controller)
	}

	rows := this.buildStorageEventLogRows(o, records)

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

// buildStorageEventLogRows дополняет записи журнала (models.StorageEventLog)
// кратким наименованием СХД (одним IN-запросом сразу по всем встретившимся
// StorageSystemsId, а не по одному на запись) и текстовой расшифровкой типа
// события.
func (this *MainController) buildStorageEventLogRows(o orm.Ormer, records []models.StorageEventLog) []StorageEventLogRow {
	idSet := make(map[int64]struct{}, len(records))
	for _, r := range records {
		if r.StorageSystemsId != 0 {
			idSet[r.StorageSystemsId] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	shortNameById := make(map[int64]string, len(ids))
	if len(ids) > 0 {
		var systems []models.StorageSystems
		if _, err := o.QueryTable("storage_systems").Filter("Id__in", ids).All(&systems); err == nil {
			for _, s := range systems {
				shortNameById[s.Id] = s.ShortName
			}
		}
	}

	rows := make([]StorageEventLogRow, 0, len(records))
	for _, r := range records {
		row := StorageEventLogRow{StorageEventLog: r}
		if name, ok := shortNameById[r.StorageSystemsId]; ok && name != "" {
			row.StorageSystemShortName = name
		} else if r.Key != "" {
			row.StorageSystemShortName = r.Key
		}
		row.TypeLabel = storageEventTypeLabel(r.EventType)
		rows = append(rows, row)
	}
	return rows
}

// storageEventLogXLSXHeader - заголовок листа XLSX-экспорта /admin/storageeventlog.
var storageEventLogXLSXHeader = []string{"Id", "Время", "Тип события", "СХД / ключ", "Успех", "Сообщение"}

// storageEventLogXLSXRow формирует одну строку XLSX-экспорта из StorageEventLogRow.
func storageEventLogXLSXRow(r StorageEventLogRow) []string {
	okLabel := ""
	if r.EventType == models.StorageEventAvailability {
		if r.OK {
			okLabel = "Да"
		} else {
			okLabel = "Нет"
		}
	}
	return []string{
		strconv.FormatInt(r.Id, 10),
		r.Time.Format("2006-01-02 15:04:05"),
		r.TypeLabel,
		r.StorageSystemShortName,
		okLabel,
		r.Message,
	}
}

// ExportStorageEventLogXLSX отдаёт найденный по фильтру журнал событий СХД
// (весь, а не только текущую страницу) файлом XLSX на скачивание -
// формирование листа аналогично /admin/userlog (ExportUserLogXLSX).
func ExportStorageEventLogXLSX(this *MainController, flash *beego.FlashData, rows []StorageEventLogRow) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("storageeventlog%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	sheet := "Журнал СХД"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &storageEventLogXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j, r := range rows {
		row := storageEventLogXLSXRow(r)
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
