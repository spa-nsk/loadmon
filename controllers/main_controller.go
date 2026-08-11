// Package controllers реализует веб-контроллеры интерфейса пользователя (beego)
// системы мониторинга loadmon: отображение и фильтрацию списка серверов и
// виртуальных машин, карточки серверов с показателями нагрузки (CPU, память,
// диски) и графиками производительности, работу со справочником серверов
// (добавление, обновление данных из систем-источников), аутентификацию и
// профиль пользователя, журналирование действий, а также вспомогательный API
// для внешних систем.
package controllers

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	//"github.com/astaxie/beego"
	//"github.com/astaxie/beego/orm"
	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"

	//	"github.com/astaxie/beego/validation"
	"loadmon/events"
	//	exportres "loadmon/export"
	models "loadmon/models"
)

// MainController — базовый контроллер веб-приложения loadmon.
// Реализует общую логику отображения страниц (фильтры, сессия пользователя,
// отчёты по серверам/ВМ, графики нагрузки) для всех действий раздела мониторинга.
type MainController struct {
	beego.Controller
}

// Favicon_ico отдаёт пустой ответ на запрос /favicon.ico.
func (this *MainController) Favicon_ico() {
	this.Ctx.ResponseWriter.Write([]byte{})
}

// Shutdown корректно останавливает HTTP-сервер приложения.
func (this *MainController) Shutdown() {
	events.ShutdownServerHTTP()
	this.Ctx.ResponseWriter.Write([]byte{})
}

func (this *MainController) activeContent(view string) {
	this.Layout = "basic-layout.tpl"
	this.LayoutSections = make(map[string]string)
	this.LayoutSections["Header"] = "header.tpl"
	//	this.LayoutSections["Sidebar"] = "sidebar.tpl"
	if view != "view" {
		this.LayoutSections["Footer"] = "footer.tpl"
	}
	this.TplName = view + ".tpl"
	this.Data["domainname"] = "https://spa.usvc.global.bcs"
	//this.Data["domainname"] = "yourdomainname"
	//	_ = this.Render()

	sess := this.GetSession("loadmon")
	if sess != nil {
		this.Data["InSession"] = 1 // for login bar in header.tpl
		this.Data["Filter"] = sess.(models.LoadFilter)
		this.Data["Filter_save"] = sess.(models.LoadFilter)
		p := this.Data["Filter"]
		filter := p.(models.LoadFilter)
		m := filter.Map
		this.Data["First"] = m["first"]                  //отобразить логин
		this.Data["Last"] = m["last"]                    //отобразить логин
		this.Data["UserId"] = m["user_id"].(int64)       //Id пользователя
		this.Data["Username"] = m["username"]            //отобразить логин
		this.Data["Privilage"] = m["privilage"]          // 0 - обычный пользователь 1 - разрешено редактировать справочник 3 - админ
		this.Data["Podrazdelenie"] = m["podrazdelenie"]  // 0 - не указано 1 - ДАТИ 2 - СТО
		this.Data["SearchString"] = m["SprSearchString"] //тип строки для фильтра справочника
		this.Data["SearchField"] = m["SprSearchField"]   //стока фильтра справочника
	}
}

func (this *MainController) SaveContent(name string, data interface{}) {
	sess := this.GetSession("loadmon")
	if sess != nil {
		p := sess.(models.LoadFilter)
		m := p.Map
		m[name] = data
	}
}

func (this *MainController) GetContent(name string) interface{} {
	sess := this.GetSession("loadmon")
	if sess != nil {
		p := sess.(models.LoadFilter)
		m := p.Map
		return m[name]
	}
	return nil
}

func (this *MainController) RemoveContent(name string) {
	sess := this.GetSession("loadmon")
	if sess != nil {
		p := sess.(models.LoadFilter)
		m := p.Map
		delete(m, name)
	}
}

// CorrectTimeFilter обнуляет время суток у переданной даты,
// приводя её к началу дня по локальной временной зоне.
func CorrectTimeFilter(t1 time.Time) time.Time {
	return time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.Local)
}

// IsNullTime проверяет, является ли время "пустым" (незаполненным) значением даты,
// используемым как признак отсутствия фильтра по дате.
func IsNullTime(t time.Time) bool {
	//	if t.Year() == 1 && t.Month() == time.January && t.Day() == 1 { return true }
	if t.Year() == 1 || t.Year() == 1901 {
		return true
	}
	return false
}

// GetTime разбирает значение поля формы с именем name как дату
// в одном из форматов (dd-mm-yyyy, dd/mm/yyyy, dd.mm.yyyy) и возвращает time.Time.
func (this *MainController) GetTime(name string) time.Time {

	var day, month, year int

	cell := this.GetString(name)
	cell = strings.TrimSpace(cell)

	d_str := strings.Split(cell, "-")

	if len(d_str) != 3 {
		d_str = strings.Split(cell, "/")
	}
	if len(d_str) != 3 {
		d_str = strings.Split(cell, ".")
	}
	if len(d_str) == 3 {
		day, _ = strconv.Atoi(d_str[2])
		month, _ = strconv.Atoi(d_str[1])
		year, _ = strconv.Atoi(d_str[0])
		if day > 1901 {
			d := year
			year = day
			day = d
		}
	} else {
		log.Println("GetTime ошибка в формате даты:", cell)
		year = 1901
		month = 1
		day = 1
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)

}

// TraceFilter выводит в лог значения всех полей фильтра filter
// с префиксом hdr — используется для отладки.
func TraceFilter(hdr string, filter models.LoadFilter) {

	log.Println(hdr+"FromTime:", filter.FromTime)
	log.Println(hdr+"ToTime:", filter.ToTime)
	log.Println(hdr+"ServerName:", filter.ServerName)
	log.Println(hdr+"Inventory:", filter.Inventory)
	log.Println(hdr+"Owner:", filter.Owner)
	log.Println(hdr+"Allocation:", filter.Allocation)
	log.Println(hdr+"Anotation:", filter.Anotation)
	log.Println(hdr+"Datacenter:", filter.Datacenter)
	log.Println(hdr+"Claster:", filter.Claster)
	log.Println(hdr+"Service:", filter.Service)
	log.Println(hdr+"Busines:", filter.Busines)
	log.Println(hdr+"Businesln:", filter.Businesln)
	log.Println(hdr+"CenterMonney:", filter.CenterMonney)
}

// Get обрабатывает главную страницу приложения: проверяет наличие
// сессии пользователя и перенаправляет на страницу входа или на список серверов.
func (this *MainController) Get() {
	this.activeContent("index")
	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_view2_all_0", 302)
		return
	}
	this.Redirect("/sprsrv/view2/all/0", 302)
}

// Notice отображает страницу с уведомлением (flash-сообщением).
func (this *MainController) Notice() {
	this.activeContent("notice")

	flash := beego.ReadFromRequest(&this.Controller)
	if n, ok := flash.Data["notice"]; ok {
		this.Data["notice"] = n
	}
}

// Filter отображает и обрабатывает форму фильтра отчётов:
// GET — показывает текущие значения, POST — сохраняет новые значения фильтра в сессии.
func (this *MainController) Filter() {
	this.activeContent("filter")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/filter", 302)
		return
	}

	p := this.Data["Filter"]
	filter := p.(models.LoadFilter)

	switch {
	case this.Ctx.Input.Method() == "GET":

	case this.Ctx.Input.Method() == "POST":
		save_filter := filter

		//добавить проверку на ошибки
		filter.FromTime = this.GetTime("FromTime")
		filter.ToTime = this.GetTime("ToTime")
		filter.ServerName = this.GetString("ServerName")
		filter.Inventory = this.GetString("Inventory")
		filter.Owner = this.GetString("Owner")
		filter.Allocation = this.GetString("Allocation")
		filter.Anotation = this.GetString("Anotation")
		filter.Datacenter = this.GetString("Datacenter")
		filter.Claster = this.GetString("Claster")
		filter.Service = this.GetString("Service")
		filter.Busines = this.GetString("Busines")
		filter.Businesln = this.GetString("Businesln")
		filter.CenterMonney = this.GetString("CenterMonney")

		if IsNullTime(filter.FromTime) {
			filter.FromTime = save_filter.FromTime
		}
		if IsNullTime(filter.ToTime) {
			filter.ToTime = save_filter.ToTime
		}
		filter.FromTime = CorrectTimeFilter(filter.FromTime)
		filter.ToTime = CorrectTimeFilter(filter.ToTime)
		filter.ServerName = strings.TrimSpace(filter.ServerName)
		filter.Inventory = strings.TrimSpace(filter.Inventory)
		filter.Owner = strings.TrimSpace(filter.Owner)
		filter.Allocation = strings.TrimSpace(filter.Allocation)
		filter.Anotation = strings.TrimSpace(filter.Anotation)
		filter.Datacenter = strings.TrimSpace(filter.Datacenter)
		filter.Claster = strings.TrimSpace(filter.Claster)
		filter.Service = strings.TrimSpace(filter.Service)
		filter.Busines = strings.TrimSpace(filter.Busines)
		filter.Businesln = strings.TrimSpace(filter.Businesln)
		filter.CenterMonney = strings.TrimSpace(filter.CenterMonney)

		log.Println("metod POST (Значения из формы) From: ", filter.FromTime, "To", filter.ToTime)
		this.DelSession("loadmon")
		this.SetSession("loadmon", filter)
		this.Data["Filter"] = filter
		this.Data["Filter_save"] = filter
		//		this.Redirect("/view/filter", 302)
		this.Redirect("/", 302)
	}
}

// View отображает отчёт по серверам за период, заданный фильтром
// (или предустановленный режим "месяц"/"весь период" через параметр :Mode).
func (this *MainController) View() {
	this.Redirect("/", 302)
	//this.activeContent("view") // view.tpl отсутствует

	var records []*models.ReportServerRecord

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/view_filter", 302)
		return
	}

	p := this.Data["Filter"]
	filter := p.(models.LoadFilter)

	o := orm.NewOrm()
	//o.Using("default")
	qs := o.QueryTable("report")
	qs = qs.OrderBy("time")

	mode := this.Ctx.Input.Param(":Mode")
	if mode == "" || mode == "filter" {
		qs = qs.Filter("time__gte", filter.FromTime.UTC()).Filter("time__lt", filter.ToTime.UTC().AddDate(0, 0, 1))
	} else if mode == "month" {
		to := time.Now().Local()
		to = CorrectTimeFilter(to)
		from := to.AddDate(0, -1, 0)
		from = CorrectTimeFilter(from)
		qs = qs.Filter("time__gte", from.UTC()).Filter("time__lt", to.UTC().AddDate(0, 0, 1))
		filter.FromTime = from
		filter.ToTime = to
		this.Data["Filter"] = filter
	} else {
		to := time.Now().Local()
		to = CorrectTimeFilter(to)
		from := time.Date(2019, time.Month(10), 1, 0, 0, 0, 0, time.Local)
		filter.FromTime = from
		filter.ToTime = to
		this.Data["Filter"] = filter
	}

	//	qs = qs.Filter("time__gte", filter.FromTime).Filter("time__lte", filter.ToTime)
	log.Println("View (Значения для отчета) From: ", filter.FromTime, "To", filter.ToTime)
	//	qs = qs.Filter("time__gt", time.Date(2019, time.Month(11), 01, 0, 0, 0, 0, time.Loc)).Filter("time__lt", time.Date(2019, time.Month(12), 01, 0, 0, 0, 0, time.Loc))
	//	qs.Filter("time__gt", "2019-11-01 00:00:00")
	//	qs = qs.Filter("server_name", "s-rms-sql-01.global.bcs")
	if filter.Inventory != "" {
		qs = qs.Filter("inventory__contains", filter.Inventory)
	}
	if filter.ServerName != "" {
		qs = qs.Filter("server_name__contains", filter.ServerName)
	}
	if filter.Allocation != "" {
		qs = qs.Filter("allocation__contains", filter.Allocation)
	}
	if filter.Owner != "" {
		qs = qs.Filter("owner__contains", filter.Owner)
	}
	if filter.Service != "" {
		qs = qs.Filter("service__contains", filter.Service)
	}

	num, err := qs.All(&records)
	log.Println("Количество записей", len(records))

	if err != orm.ErrNoRows && num > 0 {
		for i := range records {
			records[i].Time = records[i].Time.Local()
		}
		this.Data["records"] = records
	}
	RegisterRequest(this, 7) // просмотр данных

}

// ViewTop отображает агрегированный отчёт (топ) по серверам за период,
// с досчётом ядер CPU и вычислением максимальных показателей нагрузки.
func (this *MainController) ViewTop() {
	//this.activeContent("top") // top.tpl отсутствует

	var records []*models.ReportServerRecord
	var recs []*models.ReportServerRecord

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/top_filter", 302)
		return
	}

	p := this.Data["Filter"]
	filter := p.(models.LoadFilter)

	o := orm.NewOrm()
	//o.Using("default")
	qs := o.QueryTable("report")
	qs = qs.OrderBy("time")

	mode := this.Ctx.Input.Param(":Mode")
	if mode == "" || mode == "filter" {
		qs = qs.Filter("time__gte", filter.FromTime.UTC()).Filter("time__lt", filter.ToTime.UTC().AddDate(0, 0, 1))
	} else if mode == "month" {
		to := time.Now().Local()
		to = CorrectTimeFilter(to)
		from := to.AddDate(0, -1, 0)
		from = CorrectTimeFilter(from)
		qs = qs.Filter("time__gte", from.UTC()).Filter("time__lt", to.UTC().AddDate(0, 0, 1))
		filter.FromTime = from
		filter.ToTime = to
		this.Data["Filter"] = filter
	} else {
		to := time.Now().Local()
		to = CorrectTimeFilter(to)
		from := time.Date(2019, time.Month(10), 1, 0, 0, 0, 0, time.Local)
		filter.FromTime = from
		filter.ToTime = to
		this.Data["Filter"] = filter
	}

	//	qs = qs.Filter("time__gte", filter.FromTime).Filter("time__lte", filter.ToTime)
	log.Println("View (Значения для отчета) From: ", filter.FromTime, "To", filter.ToTime)
	//	qs = qs.Filter("time__gt", time.Date(2019, time.Month(11), 01, 0, 0, 0, 0, time.Loc)).Filter("time__lt", time.Date(2019, time.Month(12), 01, 0, 0, 0, 0, time.Loc))
	//	qs.Filter("time__gt", "2019-11-01 00:00:00")
	//	qs = qs.Filter("server_name", "s-rms-sql-01.global.bcs")
	if filter.Inventory != "" {
		qs = qs.Filter("inventory__contains", filter.Inventory)
	}
	if filter.ServerName != "" {
		qs = qs.Filter("server_name__contains", strings.Split(filter.ServerName, " ")[0])
	}
	if filter.Allocation != "" {
		qs = qs.Filter("allocation__contains", filter.Allocation)
	}
	if filter.Owner != "" {
		qs = qs.Filter("owner__contains", filter.Owner)
	}
	if filter.Service != "" {
		qs = qs.Filter("service__contains", filter.Service)
	}

	num, err := qs.All(&records)
	PatchCoreCPURecords(records)
	recs = CalcRecords(records)
	log.Println("Количество записей", len(recs))

	if err != orm.ErrNoRows && num > 0 {
		for i := range records {
			records[i].Time = records[i].Time.Local()
		}
		this.Data["records"] = recs
	}
	RegisterRequest(this, 7) // просмотр данных

}

// PatchCoreCPURecords дополняет и корректирует записи records данными
// о количестве CPU и ядер из справочника серверов/ВМ и данных оборудования.
func (this *MainController) ServerDetails() {
	this.activeContent("details")

	var records []*models.ReportServerRecord

	sess := this.GetSession("loadmon")
	if sess == nil {
		redirect := "/user/login/details_" + this.Ctx.Input.Param(":Mode") + "_" + this.Ctx.Input.Param(":ServerName")
		this.Redirect(redirect, 302)
		//this.Redirect("/user/login/view_filter", 302)
		return
	}

	p := this.Data["Filter"]
	login := strings.TrimSpace(this.Data["First"].(string))
	filter := p.(models.LoadFilter)

	filter.ServerName, _ = url.QueryUnescape(this.Ctx.Input.Param(":ServerName"))
	srv := strings.Split(filter.ServerName, ".")
	filter.ServerName = srv[0]

	mode := this.Ctx.Input.Param(":Mode")

	o := orm.NewOrm()
	//o.Using("default")
	qs := o.QueryTable("report")

	qs = qs.OrderBy("time")

	if mode == "" || mode == "filter" {
		qs = qs.Filter("time__gte", filter.FromTime.UTC()).Filter("time__lt", filter.ToTime.UTC().AddDate(0, 0, 1))
	} else if mode == "month" {
		to := time.Now().Local()
		to = CorrectTimeFilter(to)
		from := to.AddDate(0, -1, 0)
		from = CorrectTimeFilter(from)
		qs = qs.Filter("time__gte", from.UTC()).Filter("time__lt", to.UTC().AddDate(0, 0, 1))
		filter.FromTime = from
		filter.ToTime = to
		this.Data["Filter"] = filter
	} else {
		to := time.Now().Local()
		to = CorrectTimeFilter(to)
		from := time.Date(2019, time.Month(10), 1, 0, 0, 0, 0, time.Local)
		filter.FromTime = from
		filter.ToTime = to
		this.Data["Filter"] = filter
	}

	//	qs = qs.Filter("server_name", filter.ServerName)

	qs = qs.Filter("server_name__icontains", strings.Split(filter.ServerName, " ")[0])
	num, err := qs.All(&records)

	for i := range records {
		records[i].Time = records[i].Time.Local()
	}

	PatchCoreCPURecords(records) // корректировка значений количества cpu и core

	log.Println("Количество записей", len(records))

	if err != orm.ErrNoRows && num > 0 {
		short_name := fmt.Sprintf("%s_%02d_%02d_%04d__%02d_%02d_%04d.bin", login, filter.FromTime.Day(), filter.FromTime.Month(), filter.FromTime.Year(), filter.ToTime.Day(), filter.ToTime.Month(), filter.ToTime.Year())
		destination := "./reports/" + "CPU_" + short_name

		MakeImageReportCPU1(records, destination)

		data, err := ioutil.ReadFile(destination)
		if err != nil {
			log.Println("Ошибка чтения файла: ", destination, " - ", err)
		} else {
			this.Data["DetailsCPU"] = base64.StdEncoding.EncodeToString([]byte(data))
		}

		destination = "./reports/" + "MEM_" + short_name
		MakeImageReportMEM1(records, destination)

		data, err = ioutil.ReadFile(destination)
		if err != nil {
			log.Println("Ошибка чтения файла: ", destination, " - ", err)
		} else {
			this.Data["DetailsMEM"] = base64.StdEncoding.EncodeToString([]byte(data))
		}

		destination = "./reports/" + "DSKL_" + short_name
		if MakeImageReportDiskLoad(records, destination) == nil {
			data, err = ioutil.ReadFile(destination)
			if err != nil {
				log.Println("Ошибка чтения файла: ", destination, " - ", err)
			} else {
				this.Data["DetailsDiskLoad"] = base64.StdEncoding.EncodeToString([]byte(data))
			}
		}

		destination = "./reports/" + "DSKU_" + short_name
		if MakeImageReportDiskUse(records, destination) == nil {
			data, err = ioutil.ReadFile(destination)
			if err != nil {
				log.Println("Ошибка чтения файла: ", destination, " - ", err)
			} else {
				this.Data["DetailsDiskUse"] = base64.StdEncoding.EncodeToString([]byte(data))
			}
		}

		destination = "./reports/" + "MIX_" + short_name
		MakeImageReportMIX1(records, destination)

		data, err = ioutil.ReadFile(destination)
		if err != nil {
			log.Println("Ошибка чтения файла: ", destination, " - ", err)
		} else {
			this.Data["DetailsMIX"] = base64.StdEncoding.EncodeToString([]byte(data))
		}
	}

	this.Data["Filter"] = filter
	this.Data["records"] = records

	RegisterRequest(this, 8)
}

// ErrorController — контроллер для отображения страниц ошибок
// (404, 500, ошибка базы данных) в едином оформлении приложения.
