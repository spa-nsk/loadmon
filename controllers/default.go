package controllers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	//"github.com/astaxie/beego"
	//"github.com/astaxie/beego/orm"
	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"

	chart "github.com/wcharczuk/go-chart"

	//	"github.com/astaxie/beego/validation"
	"loadmon/events"
	//	exportres "loadmon/export"
	models "loadmon/models"
)

type MainController struct {
	beego.Controller
}

func (this *MainController) Favicon_ico() {
	this.Ctx.ResponseWriter.Write([]byte{})
}

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
		this.Data["First"] = m["first"]           //отобразить логин
		this.Data["Last"] = m["last"]             //отобразить логин
		this.Data["UserId"] = m["user_id"].(int64) //Id пользователя
		this.Data["Username"] = m["username"]     //отобразить логин
		this.Data["Privilage"] = m["privilage"]   // 0 - обычный пользователь 1 - разрешено редактировать справочник 3 - админ
		this.Data["Podrazdelenie"] = m["podrazdelenie"] // 0 - не указано 1 - ДАТИ 2 - СТО
		this.Data["SearchString"] = m["SprSearchString"] //тип строки для фильтра справочника
		this.Data["SearchField"] = m["SprSearchField"] //стока фильтра справочника
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

func CorrectTimeFilter(t1 time.Time) time.Time {
	return time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.Local)
}

func IsNullTime(t time.Time) bool {
	//	if t.Year() == 1 && t.Month() == time.January && t.Day() == 1 { return true }
	if t.Year() == 1 || t.Year() == 1901 {
		return true
	}
	return false
}

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

func (this *MainController) Get() {
	this.activeContent("index")
	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_view2_all_0", 302)
		return
	}
	this.Redirect("/sprsrv/view2/all/0", 302)
}

func (this *MainController) Notice() {
	this.activeContent("notice")

	flash := beego.ReadFromRequest(&this.Controller)
	if n, ok := flash.Data["notice"]; ok {
		this.Data["notice"] = n
	}
}

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

func PatchCoreCPURecords(records []*models.ReportServerRecord) {
	o := orm.NewOrm()

	for ind, rec := range records {
		serverName := strings.Split(strings.ToLower(rec.ServerName), "(")
		name := strings.Split(strings.ToLower(strings.TrimSpace(serverName[0])), ".")
		srv := models.ServerSprRecords{}
		//o.Using("default")
		qs := o.QueryTable(srv)
		qs = qs.OrderBy("name").Filter("name__icontains", name[0])
		if !rec.Virtual {
			if rec.CPUCount <= 4 && rec.CoreCount > 36 {
				records[ind].CoreCount = rec.CoreCount / 2
			}
		}
		err := qs.One(&srv)
		if err == orm.ErrMultiRows {
			fmt.Println("Найдено несколько записей, ожидалась одна: ", rec.ServerName)
			continue
		}
		if err == orm.ErrNoRows {
			fmt.Println("Обнаружен новый хост:", rec.ServerName)
			continue
		}
		if err != nil {
			fmt.Println("Ошибка чтения данных справочника:", rec.ServerName, err)
			continue
		}
		if !rec.Virtual {
			srvHW := models.ServerHardware{ServerId: srv.Id}
			err = o.Read(&srvHW, "ServerId")
			if err == nil {
				if srvHW.CPUCount != 0 {
					records[ind].CPUCount = float64(srvHW.CPUCount)
					if srvHW.CorePerCPU != 0 {
						records[ind].CoreCount = float64(srvHW.CPUCount * srvHW.CorePerCPU)
					}
				}
			}
		} else {
			VM := models.VMSprRecord{Id: srv.VMSprId}

			//					o.Using("vm")
			//o.Using("default")
			err = o.Read(&VM, "Id")
			if err == nil {
				records[ind].ServerName = VM.Vm_HostName
				if int(rec.CPUCount) != int(VM.Vm_NumCPU) || int(rec.CoreCount) != int(VM.Vm_NumCores) {
					fmt.Println(VM.Vm_HostName, int(rec.CPUCount), "=", int(VM.Vm_NumCPU), "|", int(rec.CoreCount), "=", int(VM.Vm_NumCores))
				}
				records[ind].CPUCount = float64(VM.Vm_NumCPU)
				records[ind].CoreCount = float64(VM.Vm_NumCores)
			}
		}
	}
}

func CalcInventarisation(rec *models.ReportServerRecord) {

	if rec.UseWorkDisk {

		if rec.CPULoadMax >= 80 || rec.MemUseMax >= 80 {
			rec.Inventory = "недостаточно ресурсов по ряду метрик"
		} else if rec.CPULoadMax < 50 && rec.MemUseMax < 50 && rec.SysDiskFreePercentMin > 50 && rec.WorkDiskFreePercentMin > 50 {
			rec.Inventory = "возможно, требуется перераспределить ресурсы"
		} else {
			rec.Inventory = "нет рекомендаций по оптимизации"
		}
	} else {
		if rec.CPULoadMax >= 80 || rec.MemUseMax >= 80 {
			rec.Inventory = "недостаточно ресурсов по ряду метрик"
		} else if rec.CPULoadMax < 50 && rec.MemUseMax < 50 && rec.SysDiskFreePercentMin > 50 {
			rec.Inventory = "возможно, требуется перераспределить ресурсы"
		} else {
			rec.Inventory = "нет рекомендаций по оптимизации"
		}
	}

}

type lessFunc func(p1, p2 *models.ReportServerRecord) bool

type MaxResource struct {
	resource []*models.ReportServerRecord
	less     []lessFunc
}

func (mr *MaxResource) Sort(resource []*models.ReportServerRecord) {
	mr.resource = resource
	sort.Sort(mr)
}

func OrderedBy(less ...lessFunc) *MaxResource {
	return &MaxResource{
		less: less,
	}
}

func (mr *MaxResource) Len() int {
	return len(mr.resource)
}

func (mr *MaxResource) Swap(i, j int) {
	mr.resource[i], mr.resource[j] = mr.resource[j], mr.resource[i]
}

func (mr *MaxResource) Less(i, j int) bool {
	p, q := mr.resource[i], mr.resource[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(mr.less)-1; k++ {
		less := mr.less[k]
		switch {
		case less(p, q):
			// p < q, so we have a decision.
			return true
		case less(q, p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	return mr.less[k](p, q)
}

func CalcRecords(records []*models.ReportServerRecord) []*models.ReportServerRecord {
	var out []*models.ReportServerRecord
	var counts []float64
	out = make([]*models.ReportServerRecord, 0)

	for _, rec := range records {
		if rec.CoreCount < 1.0 && rec.CPUCount > 0.0 {
			rec.CoreCount = rec.CPUCount
		}
		found := false
		for j, s := range out {
			if s.ServerName == rec.ServerName {
				//					s.CPUCount = s.CPUCount + rec.CPUCount
				//					s.CoreCount = s.CoreCount + rec.CoreCount
				//					s.MemSize = s.MemSize + rec.MemSize
				//					s.SysDiskSize = s.SysDiskSize + rec.SysDiskSize
				//					s.WorkDiskSize = s.WorkDiskSize + rec.WorkDiskSize
				s.CPULoadAvg = s.CPULoadAvg + rec.CPULoadAvg

				if s.CPULoadMin > rec.CPULoadMin {
					s.CPULoadMin = rec.CPULoadMin
				}
				if s.CPULoadMax < rec.CPULoadMax {
					s.CPULoadMax = rec.CPULoadMax
				}

				s.MemUseAvg = s.MemUseAvg + rec.MemUseAvg

				if s.MemUseMin > rec.MemUseMin {
					s.MemUseMin = rec.MemUseMin
				}
				if s.MemUseMax < rec.MemUseMax {
					s.MemUseMax = rec.MemUseMax
				}

				s.SysDiskFreePercentAvg = s.SysDiskFreePercentAvg + rec.SysDiskFreePercentAvg

				if s.SysDiskFreePercentMin > rec.SysDiskFreePercentMin {
					s.SysDiskFreePercentMin = rec.SysDiskFreePercentMin
				}
				if s.SysDiskFreePercentMax < rec.SysDiskFreePercentMax {
					s.SysDiskFreePercentMax = rec.SysDiskFreePercentMax
				}

				s.SysDiskIdleTimeAvg = s.SysDiskIdleTimeAvg + rec.SysDiskIdleTimeAvg

				if s.SysDiskIdleTimeMin > rec.SysDiskIdleTimeMin {
					s.SysDiskIdleTimeMin = rec.SysDiskIdleTimeMin
				}
				if s.SysDiskIdleTimeMax < rec.SysDiskIdleTimeMax {
					s.SysDiskIdleTimeMax = rec.SysDiskIdleTimeMax
				}

				s.SysDiskQueueAvg = s.SysDiskQueueAvg + rec.SysDiskQueueAvg

				if s.SysDiskQueueMin > rec.SysDiskQueueMin {
					s.SysDiskQueueMin = rec.SysDiskQueueMin
				}
				if s.SysDiskQueueMax < rec.SysDiskQueueMax {
					s.SysDiskQueueMax = rec.SysDiskQueueMax
				}

				s.WorkDiskFreePercentAvg = s.WorkDiskFreePercentAvg + rec.WorkDiskFreePercentAvg

				if s.WorkDiskFreePercentMin > rec.WorkDiskFreePercentMin {
					s.WorkDiskFreePercentMin = rec.WorkDiskFreePercentMin
				}
				if s.WorkDiskFreePercentMax < rec.WorkDiskFreePercentMax {
					s.WorkDiskFreePercentMax = rec.WorkDiskFreePercentMax
				}

				s.WorkDiskIdleTimeAvg = s.WorkDiskIdleTimeAvg + rec.WorkDiskIdleTimeAvg

				if s.WorkDiskIdleTimeMin > rec.WorkDiskIdleTimeMin {
					s.WorkDiskIdleTimeMin = rec.WorkDiskIdleTimeMin
				}
				if s.WorkDiskIdleTimeMax < rec.WorkDiskIdleTimeMax {
					s.WorkDiskIdleTimeMax = rec.WorkDiskIdleTimeMax
				}

				s.WorkDiskQueueAvg = s.WorkDiskQueueAvg + rec.WorkDiskQueueAvg
				if s.WorkDiskQueueMin > rec.WorkDiskQueueMin {
					s.WorkDiskQueueMin = rec.WorkDiskQueueMin
				}
				if s.WorkDiskQueueMax < rec.WorkDiskQueueMax {
					s.WorkDiskQueueMax = rec.WorkDiskQueueMax
				}

				s.SysDiskIOAvg = s.SysDiskIOAvg + rec.SysDiskIOAvg
				if s.SysDiskIOMin > rec.SysDiskIOMin {
					s.SysDiskIOMin = rec.SysDiskIOMin
				}
				if s.SysDiskIOMax < rec.SysDiskIOMax {
					s.SysDiskIOMax = rec.SysDiskIOMax
				}

				s.SysDiskFreeMbAvg = s.SysDiskFreeMbAvg + rec.SysDiskFreeMbAvg
				if s.SysDiskFreeMbMin > rec.SysDiskFreeMbMin {
					s.SysDiskFreeMbMin = rec.SysDiskFreeMbMin
				}
				if s.SysDiskFreeMbMax < rec.SysDiskFreeMbMax {
					s.SysDiskFreeMbMax = rec.SysDiskFreeMbMax
				}

				s.WorkDiskIOAvg = s.WorkDiskIOAvg + rec.WorkDiskIOAvg
				if s.WorkDiskIOMin > rec.WorkDiskIOMin {
					s.WorkDiskIOMin = rec.WorkDiskIOMin
				}
				if s.WorkDiskIOMax < rec.WorkDiskIOMax {
					s.WorkDiskIOMax = rec.WorkDiskIOMax
				}

				s.WorkDiskFreeMbAvg = s.WorkDiskFreeMbAvg + rec.WorkDiskFreeMbAvg
				if s.WorkDiskFreeMbMin > rec.WorkDiskFreeMbMin {
					s.WorkDiskFreeMbMin = rec.WorkDiskFreeMbMin
				}
				if s.WorkDiskFreeMbMax < rec.WorkDiskFreeMbMax {
					s.WorkDiskFreeMbMax = rec.WorkDiskFreeMbMax
				}

				counts[j] = counts[j] + 1.0
				found = true
			}
		}

		if !found {
			r := *rec
			out = append(out, &r)
			counts = append(counts, 1.0)
		}
	}
	for k, s := range out {
		//		out[k].CPUCount = s.CPUCount / counts[k]
		//		out[k].CoreCount = s.CoreCount / counts[k]
		//		out[k].MemSize = s.MemSize / counts[k]
		//		out[k].SysDiskSize = s.SysDiskSize / counts[k]
		//		out[k].WorkDiskSize = s.WorkDiskSize / counts[k]
		out[k].CPULoadAvg = s.CPULoadAvg / counts[k]
		out[k].MemUseAvg = s.MemUseAvg / counts[k]
		out[k].SysDiskFreePercentAvg = s.SysDiskFreePercentAvg / counts[k]
		out[k].SysDiskIdleTimeAvg = s.SysDiskIdleTimeAvg / counts[k]
		out[k].SysDiskQueueAvg = s.SysDiskQueueAvg / counts[k]
		out[k].WorkDiskFreePercentAvg = s.WorkDiskFreePercentAvg / counts[k]
		out[k].WorkDiskIdleTimeAvg = s.WorkDiskIdleTimeAvg / counts[k]
		out[k].WorkDiskQueueAvg = s.WorkDiskQueueAvg / counts[k]
		out[k].SysDiskIOAvg = s.SysDiskIOAvg / counts[k]
		out[k].SysDiskFreeMbAvg = s.SysDiskFreeMbAvg / counts[k]
		out[k].WorkDiskIOAvg = s.WorkDiskIOAvg / counts[k]
		out[k].WorkDiskFreeMbAvg = s.WorkDiskFreeMbAvg / counts[k]
		CalcInventarisation(out[k]) //пересчет инвентаризации
	}

	//	sort.Sort(MaxResource(out))

	CoreCount := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.CoreCount > c2.CoreCount
	}
	MemSize := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.MemSize > c2.MemSize
	}

	SysDiskSize := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.SysDiskSize > c2.SysDiskSize
	}

	WorkDiskSize := func(c1, c2 *models.ReportServerRecord) bool {
		return c1.WorkDiskSize > c2.WorkDiskSize
	}

	OrderedBy(MemSize, CoreCount, SysDiskSize, WorkDiskSize).Sort(out)
	return out
}

func releases() []chart.GridLine {
	return []chart.GridLine{
		{Value: float64(time.Date(2019, 12, 2, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 3, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 4, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 5, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 6, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 7, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 8, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 9, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 10, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 11, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 12, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 13, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 14, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 15, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 16, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 17, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 18, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 19, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 20, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 21, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 22, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 23, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 24, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 25, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 26, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 27, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 28, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 29, 0, 0, 0, 0, time.Local).UnixNano())},
		{Value: float64(time.Date(2019, 12, 30, 0, 0, 0, 0, time.Local).UnixNano())},
	}
}

func MakeImageReportCPU1(records []*models.ReportServerRecord, destination string) error {

	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64
	num := len(records)
	count_series := 3
	series = make([]chart.Series, count_series) // +1 если добавить закомментированный кусок

	xValues := make([]time.Time, num)
	yCPUMax := make([]float64, num)
	yCPUAvg := make([]float64, num)
	yCPUMin := make([]float64, num)
	ServerType := "Сервер: "
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		yCPUMax[j] = record.CPULoadMax
		yCPUAvg[j] = record.CPULoadAvg
		yCPUMin[j] = record.CPULoadMin
	}

	series[0] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(15),
		},
		Name:    "Процессор макс.",
		XValues: xValues,
		YValues: yCPUMax,
	}

	series[1] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(15),
		},
		Name:    "Процессор ср.",
		XValues: xValues,
		YValues: yCPUAvg,
	}

	series[2] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(15),
		},
		Name:    "Процессор мин.",
		XValues: xValues,
		YValues: yCPUMin,
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

func MakeImageReportMEM1(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 3
	series = make([]chart.Series, count_series) // +1 если добавить закомментированный кусок

	xValues := make([]time.Time, num)
	yMEMMax := make([]float64, num)
	yMEMAvg := make([]float64, num)
	yMEMMin := make([]float64, num)
	ServerType := "Сервер: "
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		yMEMMax[j] = record.MemUseMax
		yMEMAvg[j] = record.MemUseAvg
		yMEMMin[j] = record.MemUseMin
	}

	series[0] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(15),
		},
		Name:    "Память макс.",
		XValues: xValues,
		YValues: yMEMMax,
	}

	series[1] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(15),
		},
		Name:    "Память ср.",
		XValues: xValues,
		YValues: yMEMAvg,
	}

	series[2] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(15),
		},
		Name:    "Память мин.",
		XValues: xValues,
		YValues: yMEMMin,
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

func MakeImageReportDiskLoad(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 0
	xValues := make([]time.Time, num)
	ySYS_QUEUE := make([]float64, num)
	yWORK_QUEUE := make([]float64, num)
	ySYS_DISK_LOADMax := make([]float64, num)
	ySYS_DISK_LOADAvg := make([]float64, num)
	ySYS_DISK_LOADMin := make([]float64, num)
	yWORK_DISK_LOADMax := make([]float64, num)
	yWORK_DISK_LOADAvg := make([]float64, num)
	yWORK_DISK_LOADMin := make([]float64, num)
	ServerType := "Сервер: "
	flag_UseWorkDisk := false
	flag_FromSCOM := false
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
			if record.UseWorkDisk {
				count_series = count_series + 4
				flag_UseWorkDisk = true
			} //добавляем данные по второму диску
			if !(record.SysDiskQueueMax == 0.0 && record.SysDiskIdleTimeMin == 0.0) {
				count_series = count_series + 4
				flag_FromSCOM = true
			}
			series = make([]chart.Series, count_series) // +1 если добавить законментированный кусок
			log.Println(" количество series:", count_series)
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		ySYS_QUEUE[j] = record.SysDiskQueueMax
		yWORK_QUEUE[j] = record.WorkDiskQueueMax
		ySYS_DISK_LOADMax[j] = 100.0 - record.SysDiskIdleTimeMin
		ySYS_DISK_LOADAvg[j] = 100.0 - record.SysDiskIdleTimeAvg
		ySYS_DISK_LOADMin[j] = 100.0 - record.SysDiskIdleTimeMax
		yWORK_DISK_LOADMax[j] = 100.0 - record.WorkDiskIdleTimeMin
		yWORK_DISK_LOADAvg[j] = 100.0 - record.WorkDiskIdleTimeAvg
		yWORK_DISK_LOADMin[j] = 100.0 - record.WorkDiskIdleTimeMax
	}
	k := 0
	if count_series == 0 { // нет данных по дискам
		return errors.New("Нет данных для отчета")
	}
	if flag_FromSCOM {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorBlack,
				FillColor:   chart.ColorBlack.WithAlpha(5),
			},
			Name:    "Очередь системного диска",
			XValues: xValues,
			YValues: ySYS_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorRed,
				FillColor:   chart.ColorRed.WithAlpha(15),
			},
			Name:    "Загрузка системного диска макс.",
			XValues: xValues,
			YValues: ySYS_DISK_LOADMax,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorGreen,
				FillColor:   chart.ColorGreen.WithAlpha(15),
			},
			Name:    "Загрузка системного диска ср.",
			XValues: xValues,
			YValues: ySYS_DISK_LOADAvg,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorBlue,
				FillColor:   chart.ColorBlue.WithAlpha(15),
			},
			Name:    "Загрузка системного диска мин.",
			XValues: xValues,
			YValues: ySYS_DISK_LOADMin,
		}
		k = k + 1
	}
	if flag_UseWorkDisk {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorCyan,
				FillColor:   chart.ColorCyan.WithAlpha(15),
			},
			Name:    "Очередь рабочего диска",
			XValues: xValues,
			YValues: yWORK_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorOrange,
				FillColor:   chart.ColorOrange.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска макс.",
			XValues: xValues,
			YValues: yWORK_DISK_LOADMax,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorYellow,
				FillColor:   chart.ColorYellow.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска ср.",
			XValues: xValues,
			YValues: yWORK_DISK_LOADAvg,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorAlternateBlue,
				FillColor:   chart.ColorAlternateBlue.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска мин.",
			XValues: xValues,
			YValues: yWORK_DISK_LOADMin,
		}
		k = k + 1
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

func MakeImageReportDiskUse(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 3
	xValues := make([]time.Time, num)
	ySYS_DISK_USEMax := make([]float64, num)
	ySYS_DISK_USEAvg := make([]float64, num)
	ySYS_DISK_USEMin := make([]float64, num)
	yWORK_DISK_USEMax := make([]float64, num)
	yWORK_DISK_USEAvg := make([]float64, num)
	yWORK_DISK_USEMin := make([]float64, num)
	ServerType := "Сервер: "
	flag_UseWorkDisk := false
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
			if record.UseWorkDisk {
				count_series = count_series + 3
				flag_UseWorkDisk = true
			} //добавляем данные по второму диску
			series = make([]chart.Series, count_series) // +1 если добавить законментированный кусок
			log.Println(" количество series:", count_series)
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		ySYS_DISK_USEMax[j] = 100.0 - record.SysDiskFreePercentMin
		ySYS_DISK_USEAvg[j] = 100.0 - record.SysDiskFreePercentAvg
		ySYS_DISK_USEMin[j] = 100.0 - record.SysDiskFreePercentMax
		yWORK_DISK_USEMax[j] = 100.0 - record.WorkDiskFreePercentMin
		yWORK_DISK_USEAvg[j] = 100.0 - record.WorkDiskFreePercentAvg
		yWORK_DISK_USEMin[j] = 100.0 - record.WorkDiskFreePercentMax
	}
	k := 0
	if count_series == 0 { // нет данных по дискам
		return errors.New("Нет данных для отчета")
	}

	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(5),
		},
		Name:    "Занято на системном диске макс.",
		XValues: xValues,
		YValues: ySYS_DISK_USEMax,
	}
	k = k + 1
	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(5),
		},
		Name:    "Занято на системном диске ср.",
		XValues: xValues,
		YValues: ySYS_DISK_USEAvg,
	}
	k = k + 1
	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(5),
		},
		Name:    "Занято на системном диске мин.",
		XValues: xValues,
		YValues: ySYS_DISK_USEMin,
	}
	k = k + 1

	if flag_UseWorkDisk {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorOrange,
				FillColor:   chart.ColorOrange.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске макс.",
			XValues: xValues,
			YValues: yWORK_DISK_USEMax,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorYellow,
				FillColor:   chart.ColorYellow.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске ср.",
			XValues: xValues,
			YValues: yWORK_DISK_USEAvg,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorAlternateBlue,
				FillColor:   chart.ColorAlternateBlue.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске мин.",
			XValues: xValues,
			YValues: yWORK_DISK_USEMin,
		}
		k = k + 1
	}

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

func MakeImageReportMIX1(records []*models.ReportServerRecord, destination string) error {
	var series []chart.Series
	var ServerName string
	var MemSize float64
	var CoreCount float64
	var SysDiskSize float64

	num := len(records)
	count_series := 3
	xValues := make([]time.Time, num)
	yCPU := make([]float64, num)
	yRAM := make([]float64, num)
	ySYS_QUEUE := make([]float64, num)
	yWORK_QUEUE := make([]float64, num)
	ySYS_DISK_LOAD := make([]float64, num)
	yWORK_DISK_LOAD := make([]float64, num)
	ySYS_DISK_USE := make([]float64, num)
	yWORK_DISK_USE := make([]float64, num)
	ServerType := "Сервер: "
	flag_UseWorkDisk := false
	flag_FromSCOM := false
	for j, record := range records {
		if j == 0 {
			ServerName = record.ServerName
			MemSize = record.MemSize
			CoreCount = record.CoreCount
			if CoreCount < 1.0 {
				CoreCount = record.CPUCount
			}
			SysDiskSize = record.SysDiskSize
			if record.Virtual {
				ServerType = "Виртуальный сервер: "
			}
			if record.UseWorkDisk {
				count_series = count_series + 3
				flag_UseWorkDisk = true
			} //добавляем данные по второму диску
			if !(record.SysDiskQueueMax == 0.0 && record.SysDiskIdleTimeMin == 0.0) {
				count_series = count_series + 2
				flag_FromSCOM = true
			}
			series = make([]chart.Series, count_series) // +1 если добавить законментированный кусок
			log.Println(" Количество series:", count_series)
		}
		xValues[j] = CorrectTimeFilter(record.Time)
		yCPU[j] = record.CPULoadMax
		yRAM[j] = record.MemUseMax
		ySYS_QUEUE[j] = record.SysDiskQueueMax
		yWORK_QUEUE[j] = record.WorkDiskQueueMax
		ySYS_DISK_LOAD[j] = 100.0 - record.SysDiskIdleTimeMin
		yWORK_DISK_LOAD[j] = 100.0 - record.WorkDiskIdleTimeMin
		ySYS_DISK_USE[j] = 100.0 - record.SysDiskFreePercentMin
		yWORK_DISK_USE[j] = 100.0 - record.WorkDiskFreePercentMin

	}
	k := 0
	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorRed,
			FillColor:   chart.ColorRed.WithAlpha(15),
		},
		Name:    "Процессор",
		XValues: xValues,
		YValues: yCPU,
	}
	k = k + 1

	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorGreen,
			FillColor:   chart.ColorGreen.WithAlpha(15),
		},
		Name:    "Память",
		XValues: xValues,
		YValues: yRAM,
	}
	k = k + 1

	series[k] = chart.TimeSeries{
		Style: chart.Style{
			StrokeColor: chart.ColorBlack,
			FillColor:   chart.ColorBlack.WithAlpha(5),
		},
		Name:    "Занято на системном диске",
		XValues: xValues,
		YValues: ySYS_DISK_USE,
	}
	k = k + 1

	if flag_FromSCOM {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorBlue,
				FillColor:   chart.ColorBlue.WithAlpha(15),
			},
			Name:    "Очередь системного диска",
			XValues: xValues,
			YValues: ySYS_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorCyan,
				FillColor:   chart.ColorCyan.WithAlpha(15),
			},
			Name:    "Загрузка системного диска",
			XValues: xValues,
			YValues: ySYS_DISK_LOAD,
		}
		k = k + 1
	}
	if flag_UseWorkDisk {
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorAlternateBlue,
				FillColor:   chart.ColorAlternateBlue.WithAlpha(5),
			},
			Name:    "Занято на рабочем диске",
			XValues: xValues,
			YValues: yWORK_DISK_USE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorOrange,
				FillColor:   chart.ColorOrange.WithAlpha(15),
			},
			Name:    "Очередь рабочего диска",
			XValues: xValues,
			YValues: yWORK_QUEUE,
		}
		k = k + 1
		series[k] = chart.TimeSeries{
			Style: chart.Style{
				StrokeColor: chart.ColorYellow,
				FillColor:   chart.ColorYellow.WithAlpha(15),
			},
			Name:    "Загрузка рабочего диска",
			XValues: xValues,
			YValues: yWORK_DISK_LOAD,
		}
		k = k + 1
	}

	//
	//	series[3] = chart.AnnotationSeries{
	//		Annotations: []chart.Value{
	//			{XValue: chart.TimeToFloat64(xValues[0]), YValue: yCPU[0], Label: "Процессор", Style: chart.Style{StrokeColor: chart.ColorRed, FillColor: chart.ColorRed.WithAlpha(20)}},
	//			{XValue: chart.TimeToFloat64(xValues[0]), YValue: yRAM[0], Label: "Память", Style: chart.Style{StrokeColor: chart.ColorGreen, FillColor: chart.ColorGreen.WithAlpha(20)}},
	//			{XValue: chart.TimeToFloat64(xValues[0]), YValue: yQUEUE[0], Label: "Очередь диска", Style: chart.Style{StrokeColor: chart.ColorBlue, FillColor: chart.ColorBlue.WithAlpha(20)}},
	//		},
	//	}
	//

	graph := chart.Chart{
		Title:  ServerType + ServerName + fmt.Sprintf("(Ядер:%.0f/Память:%.0fГб/Диск:%.0fГб)", CoreCount, MemSize, SysDiskSize),
		Width:  1850,
		Height: 900,
		DPI:    75,
		XAxis: chart.XAxis{
			Name: "Дата",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
			ValueFormatter: chart.TimeValueFormatterWithFormat("02.01.06"),
			GridMajorStyle: chart.Style{
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			//			GridLines: releases(),
		},
		YAxis: chart.YAxis{
			Name: "Проценты / значение",
			TickStyle: chart.Style{
				TextRotationDegrees: 45.0,
			},
		},
		Series: series,
	}
	graph.Elements = []chart.Renderable{chart.LegendLeft(&graph)}

	f, _ := os.Create(destination)
	graph.Render(chart.PNG, f)
	f.Close()
	return nil
}

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

type ErrorController struct {
	beego.Controller
}

func (this *ErrorController) activeContent(view string) {
	this.Layout = "basic-layout.tpl"
	this.LayoutSections = make(map[string]string)
	this.LayoutSections["Header"] = "header.tpl"
	//	this.LayoutSections["Sidebar"] = "sidebar.tpl"
	this.LayoutSections["Footer"] = "footer.tpl"
	this.TplName = view + ".tpl"
	this.Data["domainname"] = "https://spa.usvc.global.bcs"
	//this.Data["domainname"] = "yourdomainname"

	sess := this.GetSession("loadmon")
	if sess != nil {
		this.Data["InSession"] = 1 // for login bar in header.tpl
		this.Data["Filter"] = sess.(models.LoadFilter)
		this.Data["Filter_save"] = sess.(models.LoadFilter)
		p := this.Data["Filter"]
		filter := p.(models.LoadFilter)
		m := filter.Map
		this.Data["First"] = m["first"]           //отобразить логин
		this.Data["Last"] = m["last"]             //отобразить логин
		this.Data["Username"] = m["username"]     //отобразить логин
		this.Data["Privilage"] = m["privilage"]   // 0 - обычный пользователь 1 - разрешено редактировать справочник 3 - админ
		this.Data["Podrazdelenie"] = m["podrazdelenie"] // 0 - не указано 1 - ДАТИ 2 - СТО
	}
}

func (this *ErrorController) Error404() {
	//
	this.activeContent("404")
	this.Data["content"] = "Страница не найдена."
	destination := "./static/img/gopher_china.png"
	data, err := ioutil.ReadFile(destination)
	if err != nil {
		log.Println("Ошибка чтения файла: ", destination, " - ", err)
	} else {
		this.Data["Image"] = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

func (this *ErrorController) Error500() {
	this.activeContent("500")
	this.Data["content"] = "Внутренняя ошибка сервера."
	destination := "./static/img/admin.png"
	data, err := ioutil.ReadFile(destination)
	if err != nil {
		log.Println("Ошибка чтения файла: ", destination, " - ", err)
	} else {
		this.Data["Image"] = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

func (this *ErrorController) ErrorDb() {
	this.activeContent("dberror")
	this.Data["content"] = "Ошибка с базой данных."
}
