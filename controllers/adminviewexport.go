package controllers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
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

// viewExportSQLDir - каталог с файлами *.sql для режима "Выгрузка по View"
// (см. AdminViewExportList/AdminViewExport ниже). Путь берётся из app.conf
// (ключ viewsqldir), а если ключ не задан - используется значение по
// умолчанию "./conf/viewsql" (тот же относительный стиль, что и "./reports"
// у AdminMetricsUnifiedExport/прочих XLSX-выгрузок - оба пути относительны
// рабочей директории процесса веб-приложения server_web_new).
func viewExportSQLDir() string {
	if dir, err := beego.AppConfig.String("viewsqldir"); err == nil && strings.TrimSpace(dir) != "" {
		return dir
	}
	return "./conf/viewsql"
}

// viewExportMode - один пункт белого списка режимов "Выгрузка по View" (имя
// файла запроса без ".sql" + заголовок для меню/страницы).
type viewExportMode struct {
	Name  string // имя режима == имя файла запроса без ".sql" == значение :name в URL
	Title string // заголовок в меню/на странице
}

// viewExportModesDefault - белый список режимов на случай, если ключ
// viewexportmodes в app.conf не задан или пуст (чтобы страница не осталась
// полностью пустой при "голом" конфиге, например сразу после разворачивания
// проекта). Используется только как запасной вариант - основной источник
// списка - app.conf, см. viewExportModes() ниже.
var viewExportModesDefault = []viewExportMode{
	{"server-hw", "Аппаратные серверы"},
	{"server-vm", "Виртуальные серверы"},
	{"cluster", "Кластеры"},
	{"lun", "LUN"},
	{"storage-systems", "Системы хранения"},
	{"storage-vm", "СХД - виртуальные тома"},
}

// viewExportModes возвращает разрешённый список режимов (белый список) для
// маршрута /admin/viewexport/:name и одноимённых файлов запросов в
// viewExportSQLDir() - читает ключ viewexportmodes из app.conf, формат:
// "имя1:Заголовок1,имя2:Заголовок2,...". Список сознательно берётся из
// белого списка, а не из перечисления файлов *.sql в каталоге - параметр
// :name идёт прямо в путь к файлу на диске, поэтому подстановка вида
// "../../etc/passwd" должна отсекаться ДО открытия файла, и явный список
// разрешённых имён в конфиге - самый прямой способ это гарантировать.
// Требование добавить новый режим - это правка ключа viewexportmodes в
// app.conf плюс добавление одноимённого файла
// server_web_new/conf/viewsql/<name>.sql (или пути, заданного viewsqldir),
// а не изменение кода контроллера/шаблонов.
//
// Конфиг читается при каждом обращении (список короткий, а app.conf может
// быть перечитан живым процессом через beego.LoadAppConfig - см.
// AdminReloadConfig, если такой маршрут появится) - постоянного кэша в
// памяти намеренно нет.
func viewExportModes() []viewExportMode {
	raw, err := beego.AppConfig.String("viewexportmodes")
	if err != nil || strings.TrimSpace(raw) == "" {
		return viewExportModesDefault
	}

	modes := make([]viewExportMode, 0, 8)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		nameTitle := strings.SplitN(part, ":", 2)
		name := strings.TrimSpace(nameTitle[0])
		if name == "" {
			continue
		}
		title := name
		if len(nameTitle) == 2 {
			if t := strings.TrimSpace(nameTitle[1]); t != "" {
				title = t
			}
		}
		modes = append(modes, viewExportMode{Name: name, Title: title})
	}
	if len(modes) == 0 {
		return viewExportModesDefault
	}
	return modes
}

// viewExportTitle возвращает заголовок режима по имени, либо само имя, если
// оно не найдено в viewExportModes() (на случай, если файл .sql уже
// добавлен в каталог, а ключ viewexportmodes в app.conf ещё не обновлён -
// страница всё равно откроется, просто без "красивого" заголовка).
func viewExportTitle(name string) string {
	for _, m := range viewExportModes() {
		if m.Name == name {
			return m.Title
		}
	}
	return name
}

// viewExportModeAllowed проверяет имя режима по белому списку
// viewExportModes() (см. комментарий там же про причину белого списка).
func viewExportModeAllowed(name string) bool {
	for _, m := range viewExportModes() {
		if m.Name == name {
			return true
		}
	}
	return false
}

// viewExportResult - результат выполнения SQL-запроса режима: список
// колонок в порядке, как их вернула СУБД (а не отсортированные по имени -
// важно, если автор .sql запроса продумал порядок колонок), и сами строки в
// виде []string того же порядка. NULL-значения представлены пустой строкой -
// для табличной выгрузки/просмотра разница между NULL и пустой строкой не
// принципиальна, в отличие от отчётов, требующих различать эти случаи.
type viewExportResult struct {
	Columns []string
	Rows    [][]string
}

// runViewExportQuery читает файл <viewExportSQLDir()>/<name>.sql и выполняет
// его содержимое как единственный SELECT-запрос напрямую через
// database/sql (orm.GetDB() - тот же пул соединений, что использует beego
// ORM для алиаса "default", см. github.com/beego/beego/v2/client/orm -
// db_alias.go:GetDB), а не через типизированные ORM-структуры: список
// колонок и их порядок в этом режиме сознательно берутся из самого
// результата запроса (rows.Columns()), а не из Go-структуры - файл .sql
// может быть и простым "SELECT * FROM ..." по таблице, соответствующей
// одной из моделей ORM (тогда набор колонок совпадёт со структурой модели),
// и сложным JOIN/агрегатом с произвольным списком колонок - оба случая
// обрабатываются одинаково.
func runViewExportQuery(name string) (*viewExportResult, error) {
	if !viewExportModeAllowed(name) {
		return nil, fmt.Errorf("неизвестный режим выгрузки: %q", name)
	}

	path := viewExportSQLDir() + "/" + name + ".sql"
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл запроса %s: %w", path, err)
	}
	sqlText := strings.TrimSpace(string(raw))
	if sqlText == "" {
		return nil, fmt.Errorf("файл запроса %s пуст", path)
	}

	db, err := orm.GetDB()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить соединение с БД: %w", err)
	}

	rows, err := db.Query(sqlText)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса %s: %w", path, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список колонок результата %s: %w", path, err)
	}

	result := &viewExportResult{Columns: cols, Rows: make([][]string, 0, 256)}
	for rows.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("ошибка чтения строки результата %s: %w", path, err)
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			if v.Valid {
				row[i] = v.String
			}
		}
		result.Rows = append(result.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка чтения результата %s: %w", path, err)
	}

	return result, nil
}

// AdminViewExportMenu отображает список доступных режимов "Выгрузка по View"
// (пункт меню Администрирование → Выгрузка по View) - просто список ссылок
// на /admin/viewexport/:name, отдельной логики выборки здесь нет.
//
// Доступ - только models.AuthUser.Priv == 3.
//
// Маршрут: /admin/viewexport.
func (this *MainController) AdminViewExportMenu() {

	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/viewexportmenu")
	this.Data["Modes"] = viewExportModes()

	RegisterRequest(this, LogTypeAdminAction) // просмотр списка режимов выгрузки по View (администрирование)
}

// AdminViewExport отображает постранично результат SQL-запроса режима :name
// (файл <viewExportSQLDir()>/<name>.sql, см. runViewExportQuery) либо, при
// export=csv/xlsx, отдаёт результат целиком (без пагинации) файлом на
// скачивание - тот же приём, что и в AdminMetricsUnifiedExport
// (controllers/adminmetricsexport.go): постранично на экране, но выгрузка -
// по всей выборке. Список/порядок колонок в обоих случаях - из самого
// результата запроса, режим только просматривается (правка данных здесь не
// предусмотрена).
//
// Доступ - только models.AuthUser.Priv == 3.
//
// Маршрут: /admin/viewexport/:name.
func (this *MainController) AdminViewExport() {

	name := this.Ctx.Input.Param(":name")

	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/viewexportlist")
	this.Data["ModeName"] = name
	this.Data["ModeTitle"] = viewExportTitle(name)

	flash := beego.NewFlash()

	if !viewExportModeAllowed(name) {
		flash.Error("Неизвестный режим выгрузки: %q", name)
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport", 302)
		return
	}

	result, err := runViewExportQuery(name)
	if err != nil {
		flash.Error("Ошибка выполнения запроса режима %q: %v", name, err)
		flash.Store(&this.Controller)
		this.Data["Columns"] = []string{}
		this.Data["Rows"] = [][]string{}
		this.Data["RowCount"] = 0
		this.Data["Page"] = 1
		this.Data["PageSize"] = 50
		this.Data["TotalPages"] = 1
		this.Data["PageFrom"] = 0
		this.Data["PageTo"] = 0
		return
	}

	export := this.GetString("export")
	if export == "csv" || export == "xlsx" {
		RegisterRequest(this, LogTypeExport) // выгрузка режима "Выгрузка по View" (администрирование)
		if export == "csv" {
			exportViewCSV(this, flash, name, result)
		} else {
			exportViewXLSX(this, flash, name, result)
		}
		return
	}

	RegisterRequest(this, LogTypeAdminAction) // просмотр режима "Выгрузка по View" (администрирование)

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
	totalRecords := len(result.Rows)
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
	pageRows := make([][]string, 0)
	if offset < totalRecords {
		pageRows = result.Rows[offset:end]
	}

	this.Data["Columns"] = result.Columns
	this.Data["Rows"] = pageRows
	this.Data["RowCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	if totalRecords == 0 {
		this.Data["PageFrom"] = 0
		this.Data["PageTo"] = 0
	} else {
		this.Data["PageFrom"] = offset + 1
		this.Data["PageTo"] = end
	}
}

// exportViewCSV отдаёт результат режима :name файлом CSV на скачивание -
// стандартная библиотека encoding/csv, т.к. пакет csvtools (Get CSVHeader/
// GetCSVData) рассчитан на типизированные Go-структуры со тегами csv:"...",
// а здесь набор колонок заранее не известен (произвольный SQL).
func exportViewCSV(this *MainController, flash *beego.FlashData, name string, result *viewExportResult) {
	if len(result.Rows) == 0 {
		flash.Notice("Нет данных для выгрузки")
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("%s_%02d_%02d_%04d_%02d_%02d_%02d.csv", name, crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second())
	destination := "./reports/" + shortName

	f, err := os.Create(destination)
	if err != nil {
		flash.Error("Ошибка создания CSV файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	// БОМ для корректного открытия кириллицы в Excel на Windows - тот же
	// приём, что и в остальных CSV-выгрузках проекта.
	f.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write(result.Columns); err != nil {
		flash.Error("Ошибка записи заголовка CSV файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		f.Close()
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}
	for _, row := range result.Rows {
		if err := w.Write(row); err != nil {
			flash.Error("Ошибка записи строки CSV файла: "+destination+" ", err)
			flash.Store(&this.Controller)
			f.Close()
			this.Redirect("/admin/viewexport/"+name, 302)
			return
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		flash.Error("Ошибка формирования CSV файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		f.Close()
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}
	if err := f.Close(); err != nil {
		flash.Error("Ошибка закрытия CSV файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных CSV файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	this.Ctx.Output.Header("Content-Disposition", "attachment; filename=\""+shortName+"\"")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")
	this.Ctx.Output.Header("Content-Length", fmt.Sprint(fileInfo.Size()))

	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}

// exportViewXLSXHeaderStyle - не используется напрямую, оставлено место для
// единообразия с другими выгрузками XLSX проекта (жирный заголовок и т.п.
// сознательно не добавлены - формат этой выгрузки максимально простой,
// "как есть" из результата запроса).

// exportViewXLSX отдаёт результат режима :name файлом XLSX на скачивание -
// формирование листа аналогично exportUnifiedMetricsXLSX
// (controllers/adminmetricsexport.go), но с динамическим (не заранее
// известным) списком колонок.
func exportViewXLSX(this *MainController, flash *beego.FlashData, name string, result *viewExportResult) {
	if len(result.Rows) == 0 {
		flash.Notice("Нет данных для выгрузки")
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("%s_%02d_%02d_%04d_%02d_%02d_%02d.xlsx", name, crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	sheet := "Выгрузка"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &result.Columns); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	for j, row := range result.Rows {
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		rowCopy := row
		if err := file.SetSheetRow(sheet, cell, &rowCopy); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" ", err)
			flash.Store(&this.Controller)
			file.Close()
			this.Redirect("/admin/viewexport/"+name, 302)
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		this.Redirect("/admin/viewexport/"+name, 302)
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
