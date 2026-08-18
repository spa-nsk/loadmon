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

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

/*
type UserLog struct {
	Id      int64     `form:"id:"`                // id записи
	UserId  int       `orm:"null" form:"userid"`  // id пользователя
	Login   time.Time `orm:"null" form:"login"`   // время регистрации
	Time    time.Time `orm:"null" form:"last"`    // время операции
	Type    int       `form:"type"`               // тип операции - см. список кодов ниже
	Request string    `form:"request"`            // текст запроса
}
*/

// Коды типа операции (RegisterRequest(this, type_request)) - используются для
// журналирования (таблица user_log) факта обращения к любому маршруту
// приложения, кроме служебных/отладочных (pprof, /metrics, favicon,
// статические файлы) - для них журналирование не имеет смысла.
//
//	 1 - авторизация (вход)
//	 2 - выход
//	 3 - сброс/восстановление пароля (forgot/reset)
//	 4 - обновление профиля пользователя
//	 5 - чтение справочника / просмотр списка (view2, hwconflist, hwviewlist, vm, listclusterview и т.п.)
//	 6 - изменение данных справочника (создание/редактирование/групповое редактирование записи)
//	 7 - чтение таблицы загрузки (динамические графики метрик - cluster/host/vm perf day/period)
//	 8 - получение деталей по серверу/хосту/ВМ (карточка объекта, переход по серийному номеру/имени)
//	 9 - выгрузка данных (экспорт в CSV/XLSX)
//	10 - регистрация нового пользователя
//	11 - административная операция (остановка сервиса, удаление учётной записи)
//	12 - обращение к API (/api/:json)
//	13 - служебная страница (SSO, произвольные уведомления/фильтры без справочника)
const (
	LogTypeLogin         = 1
	LogTypeLogout        = 2
	LogTypePasswordReset = 3
	LogTypeProfileUpdate = 4
	LogTypeViewReference = 5
	LogTypeEditReference = 6
	LogTypeViewLoadTable = 7
	LogTypeServerDetails = 8
	LogTypeExport        = 9
	LogTypeRegister      = 10
	LogTypeAdminAction   = 11
	LogTypeAPI           = 12
	LogTypeServicePage   = 13
)

// RegisterRequest записывает в журнал (таблица user_log) факт обращения
// пользователя к приложению с указанным типом операции type_request
// (авторизация, выход, чтение/изменение справочника и т.д.).
func RegisterRequest(this *MainController, type_request int) error {

	sess := this.GetSession("loadmon")
	if sess == nil {
		return fmt.Errorf("Нет активного сеанса. Требуется авторизация.")
	}

	rec := models.UserLog{}
	filter := sess.(models.LoadFilter)
	m := filter.Map

	rec.UserId = m["user_id"].(int64)      //Id пользователя
	rec.Login = m["timestamp"].(time.Time) //время регистрации
	rec.Time = time.Now()                  //время запроса
	rec.Type = type_request                //тип запроса
	rec.Request = this.Ctx.Input.Method() + "-" + this.Ctx.Input.URL()

	o := orm.NewOrm()
	//o.Using("default")

	_, err := o.Insert(&rec)
	if err != nil {
		return err
	}
	this.Data["First"] = m["first"]       //отобразить логин
	this.Data["Last"] = m["last"]         //отобразить логин
	this.Data["Username"] = m["username"] //отобразить логин

	//log.Println(this.Data["First"].(string), " ", this.Data["Last"].(string), " ", this.Data["Username"].(string), " | ", rec.Request)
	return nil
}

// UserLogRow - одна строка журнала действий (models.UserLog) дополненная
// именем пользователя для отображения в таблице (в самой таблице userlog
// хранится только UserId, без имени).
type UserLogRow struct {
	models.UserLog
	UserName  string
	TypeLabel string
}

// UserLogTypeOption - одна опция выпадающего списка фильтра по типу операции
// (код + расшифровка для отображения).
type UserLogTypeOption struct {
	Value int
	Label string
}

// userLogTypeLabel переводит код типа операции (см. константы LogType* выше)
// в текст для отображения в журнале.
func userLogTypeLabel(t int) string {
	switch t {
	case LogTypeLogin:
		return "Авторизация"
	case LogTypeLogout:
		return "Выход"
	case LogTypePasswordReset:
		return "Сброс/восстановление пароля"
	case LogTypeProfileUpdate:
		return "Обновление профиля"
	case LogTypeViewReference:
		return "Чтение справочника"
	case LogTypeEditReference:
		return "Изменение данных справочника"
	case LogTypeViewLoadTable:
		return "Чтение таблицы загрузки"
	case LogTypeServerDetails:
		return "Детали по серверу"
	case LogTypeExport:
		return "Выгрузка данных"
	case LogTypeRegister:
		return "Регистрация пользователя"
	case LogTypeAdminAction:
		return "Административная операция"
	case LogTypeAPI:
		return "Обращение к API"
	case LogTypeServicePage:
		return "Служебная страница"
	default:
		return fmt.Sprintf("Тип %d", t)
	}
}

// ViewUserLog отображает журнал действий пользователей (таблица userlog,
// заполняется через RegisterRequest) с фильтром по типу операции и
// постраничной навигацией. Раздел "Администрирование" в меню (header.tpl) и
// сам маршрут доступны только пользователям с models.AuthUser.Priv == 3
// (администратор) - при недостаточных правах запрос перенаправляется на
// справочник серверов без ошибки.
//
// Маршрут: /admin/userlog
func (this *MainController) ViewUserLog() {

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/admin_userlog", 302)
		return
	}

	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/userlog")
	flash := beego.NewFlash()

	typeFilter := this.GetString("type")
	if typeFilter == "" {
		typeFilter = "all"
	}
	this.Data["CurType"] = typeFilter
	typeCodes := []int{
		LogTypeLogin, LogTypeLogout, LogTypePasswordReset, LogTypeProfileUpdate,
		LogTypeViewReference, LogTypeEditReference, LogTypeViewLoadTable,
		LogTypeServerDetails, LogTypeExport, LogTypeRegister, LogTypeAdminAction,
		LogTypeAPI, LogTypeServicePage,
	}
	typeOptions := make([]UserLogTypeOption, 0, len(typeCodes))
	for _, t := range typeCodes {
		typeOptions = append(typeOptions, UserLogTypeOption{Value: t, Label: userLogTypeLabel(t)})
	}
	this.Data["TypeList"] = typeOptions

	// Фильтр по диапазону дат (по полю Time - время самой операции, а не
	// входа в систему). Обе границы необязательны и независимы: можно
	// задать только "с" или только "по"; если ни одна не задана - фильтр по
	// дате не применяется и отдаётся журнал целиком (в пределах остальных
	// фильтров). Верхняя граница включительна по дате - для этого к ToDate
	// прибавляются сутки и используется Time__lt.
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

	// buildQuery возвращает queryset ТОЛЬКО с фильтром (без сортировки) -
	// именно он должен использоваться для Count(): beego ORM встраивает
	// ORDER BY прямо в SQL с агрегатом COUNT(*), а PostgreSQL не допускает
	// ORDER BY по колонке, не входящей в GROUP BY/агрегат, в таком запросе
	// ("column t0.time must appear in the GROUP BY clause..."). Сортировка
	// (OrderBy) добавляется отдельно, только для выборки самих записей -
	// см. buildOrderedQuery().
	buildQuery := func() orm.QuerySeter {
		qs := o.QueryTable("userlog")
		if typeFilter != "all" {
			if t, err := strconv.Atoi(typeFilter); err == nil {
				qs = qs.Filter("Type", t)
			}
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
	// нарезки) - тот же приём, что и в /sprsrv/listclusterview и /sprsrv/vm
	if this.GetString("export") == "xlsx" {
		RegisterRequest(this, LogTypeExport) // выгрузка журнала действий в XLSX
		var records []models.UserLog
		if _, err := buildOrderedQuery().All(&records); err != nil {
			flash.Error("Ошибка загрузки журнала для экспорта: %v", err)
			flash.Store(&this.Controller)
			return
		}
		rows := this.buildUserLogRows(o, records)
		ExportUserLogXLSX(this, flash, rows)
		return
	}

	RegisterRequest(this, LogTypeAdminAction) // просмотр журнала действий пользователей

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
		log.Println("ViewUserLog: ошибка подсчёта записей журнала (userlog):", err)
		flash.Error("Ошибка чтения журнала (userlog): %v", err)
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

	var records []models.UserLog
	if _, err := buildOrderedQuery().Limit(pageSize, offset).All(&records); err != nil {
		log.Println("ViewUserLog: ошибка выборки записей журнала (userlog):", err)
		flash.Error("Ошибка чтения журнала (userlog): %v", err)
		flash.Store(&this.Controller)
	}

	rows := this.buildUserLogRows(o, records)

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

// buildUserLogRows дополняет записи журнала (models.UserLog) именем
// пользователя (запрос по таблице users одним IN-запросом сразу по всем
// встретившимся UserId, а не по одному на запись) и текстовой расшифровкой
// типа операции.
func (this *MainController) buildUserLogRows(o orm.Ormer, records []models.UserLog) []UserLogRow {
	userIdSet := make(map[int64]struct{}, len(records))
	for _, r := range records {
		userIdSet[r.UserId] = struct{}{}
	}
	userIds := make([]int64, 0, len(userIdSet))
	for id := range userIdSet {
		userIds = append(userIds, id)
	}
	userById := make(map[int64]models.AuthUser, len(userIds))
	if len(userIds) > 0 {
		var users []models.AuthUser
		if _, err := o.QueryTable("users").Filter("Id__in", userIds).All(&users); err == nil {
			for _, u := range users {
				userById[int64(u.Id)] = u
			}
		}
	}

	rows := make([]UserLogRow, 0, len(records))
	for _, r := range records {
		row := UserLogRow{UserLog: r}
		if u, ok := userById[r.UserId]; ok {
			row.UserName = strings.TrimSpace(u.First + " " + u.Last)
			if row.UserName == "" {
				row.UserName = u.Email
			}
		}
		if row.UserName == "" {
			row.UserName = fmt.Sprintf("Id=%d", r.UserId)
		}
		row.TypeLabel = userLogTypeLabel(r.Type)
		rows = append(rows, row)
	}
	return rows
}

// userLogXLSXHeader - заголовок листа XLSX-экспорта /admin/userlog.
var userLogXLSXHeader = []string{"Id", "Пользователь", "Время входа", "Время операции", "Тип операции", "Запрос"}

// userLogXLSXRow формирует одну строку XLSX-экспорта из UserLogRow.
func userLogXLSXRow(r UserLogRow) []string {
	return []string{
		strconv.FormatInt(r.Id, 10),
		r.UserName,
		r.Login.Format("2006-01-02 15:04:05"),
		r.Time.Format("2006-01-02 15:04:05"),
		r.TypeLabel,
		r.Request,
	}
}

// ExportUserLogXLSX отдаёт найденный по фильтру журнал действий (весь, а не
// только текущую страницу) файлом XLSX на скачивание - формирование листа
// аналогично экспорту /sprsrv/listclusterview (ExportClusterListXLSX).
func ExportUserLogXLSX(this *MainController, flash *beego.FlashData, rows []UserLogRow) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("userlog%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	sheet := "Журнал"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &userLogXLSXHeader); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j, r := range rows {
		row := userLogXLSXRow(r)
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
