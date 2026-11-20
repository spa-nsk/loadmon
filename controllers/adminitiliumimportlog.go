package controllers

import (
	"fmt"
	"log"
	models "loadmon/models"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"loadmon/itilium"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// itiliumImportResultLabel — расшифровка models.ItiliumImportLog.Result.
func itiliumImportResultLabel(r int) string {
	switch r {
	case models.ItiliumImportResultInserted:
		return "Добавлен"
	case models.ItiliumImportResultUpdated:
		return "Обновлён"
	case models.ItiliumImportResultSkipped:
		return "Пропуск"
	case models.ItiliumImportResultError:
		return "Ошибка"
	default:
		return fmt.Sprintf("Result %d", r)
	}
}

// itiliumImportClassLabel — имя класса КЕ или «системная» при 0.
func itiliumImportClassLabel(cuClass int) string {
	if cuClass == 0 {
		return "системная"
	}
	if name := itilium.CUClassName(cuClass); name != "" {
		return fmt.Sprintf("%d — %s", cuClass, name)
	}
	return strconv.Itoa(cuClass)
}

// ItiliumImportLogRow — строка журнала для шаблона/экспорта.
type ItiliumImportLogRow struct {
	models.ItiliumImportLog
	ClassLabel  string
	ResultLabel string
}

// syncCUClassFilterOptions — классы, которые пишет SyncGetCU.
var syncCUClassFilterOptions = []int{
	itilium.CUClassBladeEnclosures,
	itilium.CUClassBlades,
	itilium.CUClassServers,
	itilium.CUClassStorage,
	itilium.CUClassTapeLibraries,
	itilium.CUClassNetwork,
}

// ViewItiliumImportLog — журнал последнего SyncGetCU (таблица
// itilium_import_log, перезаписывается при каждом импорте). Фильтр по
// классу КЕ / результату / ключу, пагинация, экспорт XLSX. Только Priv==3.
//
// Маршрут: /admin/itiliumimportlog
func (this *MainController) ViewItiliumImportLog() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/itiliumimportlog")
	flash := beego.NewFlash()

	classFilter := this.GetString("cuclass")
	if classFilter == "" {
		classFilter = "all"
	}
	this.Data["CurClass"] = classFilter
	classOptions := make([]UserLogTypeOption, 0, len(syncCUClassFilterOptions)+1)
	classOptions = append(classOptions, UserLogTypeOption{Value: 0, Label: "системная (0)"})
	for _, c := range syncCUClassFilterOptions {
		classOptions = append(classOptions, UserLogTypeOption{Value: c, Label: itiliumImportClassLabel(c)})
	}
	this.Data["ClassList"] = classOptions

	resultFilter := this.GetString("result")
	if resultFilter == "" {
		resultFilter = "all"
	}
	this.Data["CurResult"] = resultFilter
	resultCodes := []int{
		models.ItiliumImportResultInserted,
		models.ItiliumImportResultUpdated,
		models.ItiliumImportResultSkipped,
		models.ItiliumImportResultError,
	}
	resultOptions := make([]UserLogTypeOption, 0, len(resultCodes))
	for _, r := range resultCodes {
		resultOptions = append(resultOptions, UserLogTypeOption{Value: r, Label: itiliumImportResultLabel(r)})
	}
	this.Data["ResultList"] = resultOptions

	keyFilter := strings.TrimSpace(this.GetString("key"))
	this.Data["KeyFilter"] = keyFilter

	onlyErr := this.GetString("onlyerr") == "1"
	this.Data["OnlyErr"] = onlyErr

	o := orm.NewOrm()

	buildQuery := func() orm.QuerySeter {
		qs := o.QueryTable("itilium_import_log")
		if classFilter != "all" {
			if t, err := strconv.Atoi(classFilter); err == nil {
				qs = qs.Filter("CUClass", t)
			}
		}
		if onlyErr {
			qs = qs.Filter("Result", models.ItiliumImportResultError)
		} else if resultFilter != "all" {
			if t, err := strconv.Atoi(resultFilter); err == nil {
				qs = qs.Filter("Result", t)
			}
		}
		if keyFilter != "" {
			qs = qs.Filter("Key__icontains", keyFilter)
		}
		return qs
	}
	buildOrderedQuery := func() orm.QuerySeter {
		return buildQuery().OrderBy("CUClass", "Id")
	}

	if this.GetString("export") == "xlsx" {
		RegisterRequest(this, LogTypeExport)
		var records []models.ItiliumImportLog
		if _, err := buildOrderedQuery().All(&records); err != nil {
			flash.Error("Ошибка загрузки журнала для экспорта: %v", err)
			flash.Store(&this.Controller)
			return
		}
		rows := buildItiliumImportLogRows(records)
		ExportItiliumImportLogXLSX(this, flash, rows)
		return
	}

	RegisterRequest(this, LogTypeAdminAction)

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
		log.Println("ViewItiliumImportLog: Count:", err)
		flash.Error("Ошибка чтения журнала импорта Итилиум: %v", err)
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

	var records []models.ItiliumImportLog
	if _, err := buildOrderedQuery().Limit(pageSize, offset).All(&records); err != nil {
		log.Println("ViewItiliumImportLog: All:", err)
		flash.Error("Ошибка чтения журнала импорта Итилиум: %v", err)
		flash.Store(&this.Controller)
	}

	var lastRun time.Time
	var one models.ItiliumImportLog
	if err := o.QueryTable("itilium_import_log").OrderBy("-Time").One(&one); err == nil {
		lastRun = one.Time
	}
	this.Data["LastRun"] = lastRun
	this.Data["HasLastRun"] = !lastRun.IsZero()

	rows := buildItiliumImportLogRows(records)
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

func buildItiliumImportLogRows(records []models.ItiliumImportLog) []ItiliumImportLogRow {
	rows := make([]ItiliumImportLogRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, ItiliumImportLogRow{
			ItiliumImportLog: r,
			ClassLabel:       itiliumImportClassLabel(r.CUClass),
			ResultLabel:      itiliumImportResultLabel(r.Result),
		})
	}
	return rows
}

var itiliumImportLogXLSXHeader = []string{"Id", "Время", "Класс КЕ", "Результат", "Ключ", "Сообщение"}

func itiliumImportLogXLSXRow(r ItiliumImportLogRow) []string {
	return []string{
		strconv.FormatInt(r.Id, 10),
		r.Time.Format("2006-01-02 15:04:05"),
		r.ClassLabel,
		r.ResultLabel,
		r.Key,
		r.Message,
	}
}

// ExportItiliumImportLogXLSX — выгрузка отфильтрованного журнала в XLSX.
func ExportItiliumImportLogXLSX(this *MainController, flash *beego.FlashData, rows []ItiliumImportLogRow) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("itiliumimportlog%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx",
		crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	sheet := "Импорт Итилиум"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX: %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &itiliumImportLogXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX: %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j, r := range rows {
		row := itiliumImportLogXLSXRow(r)
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX: %v", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX: %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX: %v", err)
		flash.Store(&this.Controller)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX: %v", err)
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
