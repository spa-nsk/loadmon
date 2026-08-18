package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// vmPerfPeriodChart - данные одного графика "Загрузка за период": три ряда
// агрегатов (минимум/среднее/максимум) из VMRecord за каждую сохранённую
// точку периода, плюс необязательный четвёртый ряд "текущее значение" из
// VMPerfRecord (подробные данные, если они сохранены для соответствующих
// записей VMRecord).
type vmPerfPeriodChart struct {
	Title string              `json:"title"`
	Unit  string              `json:"unit"`
	Min   []vmPerfSeriesPoint `json:"min"`
	Avg   []vmPerfSeriesPoint `json:"avg"`
	Max   []vmPerfSeriesPoint `json:"max"`
	Raw   []vmPerfSeriesPoint `json:"raw"`
}

// SprSrvVMPerfPeriod отображает динамические графики загрузки виртуальной
// машины за произвольный период на основе данных из базы (а не текущего
// состояния гипервизора): models.VMSprRecord (справочник ВМ) -> по Id ищутся
// связанные models.VMRecord (агрегаты min/avg/max за период, Date в
// диапазоне фильтра) -> если для конкретной записи VMRecord есть подробные
// данные models.VMPerfRecord (VMPerfRecord.VMId = VMRecord.Id), они
// добавляются на график отдельным рядом "текущее значение".
//
// Диапазон дат задаётся в верхней части страницы (GET-параметры from/to,
// формат YYYY-MM-DD); при первом открытии (без параметров) - последняя
// неделя.
//
// Маршрут: /sprsrv/vmperfperiod/:Vcenter/:Datacenter/:Cluster/:VM
func (this *MainController) SprSrvVMPerfPeriod() {

	vcenter := this.Ctx.Input.Param(":Vcenter")
	datacenter := this.Ctx.Input.Param(":Datacenter")
	cluster := this.Ctx.Input.Param(":Cluster")
	vmname := this.Ctx.Input.Param(":VM")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_vmperfperiod_"+vcenter+"_"+datacenter+"_"+cluster+"_"+vmname, 302)
		return
	}
	RegisterRequest(this, LogTypeViewLoadTable) // график загрузки ВМ по данным БД за период

	this.activeContent("sprsrv/vmperfperiod")

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster
	this.Data["CurVM"] = vmname

	// значения по умолчанию - шаблон обращается к ним (в т.ч. в {{if ne .VMHostName ""}})
	// до того, как поиск ВМ в справочнике мог их переопределить; go-шаблоны
	// падают на сравнении nil-интерфейса со строкой, поэтому важно всегда
	// класть в Data именно string, а не оставлять ключ незаполненным
	this.Data["VMName"] = vmname
	this.Data["VMHostName"] = ""
	this.Data["RecordCount"] = 0
	this.Data["ChartsJSON"] = template.JS("[]")

	flash := beego.NewFlash()

	// период - по умолчанию последняя неделя (сегодня и 6 дней назад)
	to := CorrectTimeFilter(time.Now())
	from := to.AddDate(0, 0, -6)

	if s := this.GetString("from"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			from = t
		}
	}
	if s := this.GetString("to"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			to = t
		}
	}
	if to.Before(from) {
		from, to = to, from
	}

	this.Data["FromDate"] = from.Format("2006-01-02")
	this.Data["ToDate"] = to.Format("2006-01-02")

	o := orm.NewOrm()

	// 1) поиск ВМ в справочнике по vcenter/datacenter/cluster/имя
	var sprList []models.VMSprRecord
	_, err := o.QueryTable("vm_spr_record").
		Filter("Vcenter", vcenter).
		Filter("Datacenter", datacenter).
		Filter("Cluster", cluster).
		Filter("Vm_Name", vmname).
		Limit(1).
		All(&sprList)

	if err != nil || len(sprList) == 0 {
		flash.Error("Виртуальная машина \"%s\" не найдена в справочнике (vm_spr_record).", vmname)
		flash.Store(&this.Controller)
		this.Data["ChartsJSON"] = template.JS("[]")
		this.Data["RecordCount"] = 0
		return
	}
	sprRecord := sprList[0]
	this.Data["VMName"] = sprRecord.Vm_Name
	this.Data["VMHostName"] = sprRecord.Vm_HostName

	// 2) агрегаты min/avg/max за период (VMRecord.VMsprId = sprRecord.Id,
	// Date в диапазоне [from, to] включительно)
	var vmRecs []models.VMRecord
	_, err = o.QueryTable("vm_record").
		Filter("VMsprId", sprRecord.Id).
		Filter("Date__gte", from.UTC()).
		Filter("Date__lt", to.AddDate(0, 0, 1).UTC()).
		OrderBy("Date").
		All(&vmRecs)
	if err != nil {
		flash.Error("Ошибка загрузки данных за период: %v", err)
		flash.Store(&this.Controller)
		this.Data["ChartsJSON"] = template.JS("[]")
		this.Data["RecordCount"] = 0
		return
	}

	this.Data["RecordCount"] = len(vmRecs)

	// 3) подробные данные (если есть) - VMPerfRecord.VMId = VMRecord.Id для
	// каждой из найденных записей периода
	vmRecIds := make([]int64, 0, len(vmRecs))
	for _, r := range vmRecs {
		vmRecIds = append(vmRecIds, r.Id)
	}

	var perfRecs []models.VMPerfRecord
	if len(vmRecIds) > 0 {
		_, err = o.QueryTable("vm_perf").
			Filter("VMId__in", vmRecIds).
			OrderBy("Start").
			All(&perfRecs)
		if err != nil {
			// подробные данные не обязательны для отображения графиков -
			// логируем через flash.Notice и продолжаем без них
			flash.Notice("Подробные данные (vm_perf) не загружены: %v", err)
		}
	}

	// группировка по графикам: одна метрика = один график с рядами
	// min/avg/max (из VMRecord) и, при наличии, raw (из VMPerfRecord)
	charts := []*vmPerfPeriodChart{
		{Title: "CPU, использование", Unit: "%"},
		{Title: "Память, использование", Unit: "%"},
		{Title: "Диск, использование", Unit: "МБ/с"},
		{Title: "Диск, чтение", Unit: "МБ/с"},
		{Title: "Диск, запись", Unit: "МБ/с"},
		{Title: "Диск, задержка чтения", Unit: "мс"},
		{Title: "Диск, задержка записи", Unit: "мс"},
		{Title: "Сеть, использование", Unit: "МБ/с"},
	}
	for _, c := range charts {
		c.Min = make([]vmPerfSeriesPoint, 0, len(vmRecs))
		c.Avg = make([]vmPerfSeriesPoint, 0, len(vmRecs))
		c.Max = make([]vmPerfSeriesPoint, 0, len(vmRecs))
		c.Raw = make([]vmPerfSeriesPoint, 0, len(perfRecs))
	}

	for _, r := range vmRecs {
		t := r.Date.UnixMilli()

		charts[0].Min = append(charts[0].Min, vmPerfSeriesPoint{T: t, V: r.Cpu_usage_minimum})
		charts[0].Avg = append(charts[0].Avg, vmPerfSeriesPoint{T: t, V: r.Cpu_usage_average})
		charts[0].Max = append(charts[0].Max, vmPerfSeriesPoint{T: t, V: r.Cpu_usage_maximum})

		charts[1].Min = append(charts[1].Min, vmPerfSeriesPoint{T: t, V: r.Mem_usage_minimum})
		charts[1].Avg = append(charts[1].Avg, vmPerfSeriesPoint{T: t, V: r.Mem_usage_average})
		charts[1].Max = append(charts[1].Max, vmPerfSeriesPoint{T: t, V: r.Mem_usage_maximum})

		charts[2].Min = append(charts[2].Min, vmPerfSeriesPoint{T: t, V: r.Disk_usage_minimum})
		charts[2].Avg = append(charts[2].Avg, vmPerfSeriesPoint{T: t, V: r.Disk_usage_average})
		charts[2].Max = append(charts[2].Max, vmPerfSeriesPoint{T: t, V: r.Disk_usage_maximum})

		charts[3].Min = append(charts[3].Min, vmPerfSeriesPoint{T: t, V: r.Disk_read_minimum})
		charts[3].Avg = append(charts[3].Avg, vmPerfSeriesPoint{T: t, V: r.Disk_read_average})
		charts[3].Max = append(charts[3].Max, vmPerfSeriesPoint{T: t, V: r.Disk_read_maximum})

		charts[4].Min = append(charts[4].Min, vmPerfSeriesPoint{T: t, V: r.Disk_write_minimum})
		charts[4].Avg = append(charts[4].Avg, vmPerfSeriesPoint{T: t, V: r.Disk_write_average})
		charts[4].Max = append(charts[4].Max, vmPerfSeriesPoint{T: t, V: r.Disk_write_maximum})

		charts[5].Min = append(charts[5].Min, vmPerfSeriesPoint{T: t, V: r.Disk_totalReadLatency_minimum})
		charts[5].Avg = append(charts[5].Avg, vmPerfSeriesPoint{T: t, V: r.Disk_totalReadLatency_average})
		charts[5].Max = append(charts[5].Max, vmPerfSeriesPoint{T: t, V: r.Disk_totalReadLatency_maximum})

		charts[6].Min = append(charts[6].Min, vmPerfSeriesPoint{T: t, V: r.Disk_totalWriteLatency_minimum})
		charts[6].Avg = append(charts[6].Avg, vmPerfSeriesPoint{T: t, V: r.Disk_totalWriteLatency_average})
		charts[6].Max = append(charts[6].Max, vmPerfSeriesPoint{T: t, V: r.Disk_totalWriteLatency_maximum})

		charts[7].Min = append(charts[7].Min, vmPerfSeriesPoint{T: t, V: r.Net_usage_minimum})
		charts[7].Avg = append(charts[7].Avg, vmPerfSeriesPoint{T: t, V: r.Net_usage_average})
		charts[7].Max = append(charts[7].Max, vmPerfSeriesPoint{T: t, V: r.Net_usage_maximum})
	}

	for _, p := range perfRecs {
		t := p.Start.UnixMilli()
		charts[0].Raw = append(charts[0].Raw, vmPerfSeriesPoint{T: t, V: p.Cpu_usage})
		charts[1].Raw = append(charts[1].Raw, vmPerfSeriesPoint{T: t, V: p.Mem_usage})
		charts[2].Raw = append(charts[2].Raw, vmPerfSeriesPoint{T: t, V: p.Disk_usage})
		charts[3].Raw = append(charts[3].Raw, vmPerfSeriesPoint{T: t, V: p.Disk_read})
		charts[4].Raw = append(charts[4].Raw, vmPerfSeriesPoint{T: t, V: p.Disk_write})
		charts[5].Raw = append(charts[5].Raw, vmPerfSeriesPoint{T: t, V: p.Disk_totalReadLatency})
		charts[6].Raw = append(charts[6].Raw, vmPerfSeriesPoint{T: t, V: p.Disk_totalWriteLatency})
		charts[7].Raw = append(charts[7].Raw, vmPerfSeriesPoint{T: t, V: p.Net_usage})
	}

	b, jerr := json.Marshal(charts)
	if jerr != nil {
		flash.Error("Ошибка подготовки данных графиков: %v", jerr)
		flash.Store(&this.Controller)
		this.Data["ChartsJSON"] = template.JS("[]")
		return
	}

	this.Data["ChartsJSON"] = template.JS(b)
}
