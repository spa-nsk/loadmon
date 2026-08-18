package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// SprSrvClusterPerfPeriod отображает динамические графики загрузки кластера
// за произвольный период на основе данных из базы: models.ClusterSprRecords
// (справочник кластеров) -> по Id ищутся связанные models.ClusterRecord
// (агрегаты min/avg/max за период, Date в диапазоне фильтра) -> если для
// конкретной записи ClusterRecord есть подробные данные models.ClusterPerf
// (ClusterPerf.ClusterId = ClusterRecord.Id), они добавляются на график
// отдельным рядом "текущее значение". Полностью аналогичен
// controllers/sprsrvvmperfperiod.go, но для кластера.
//
// Маршрут: /sprsrv/clusterperfperiod/:Vcenter/:Datacenter/:Cluster
func (this *MainController) SprSrvClusterPerfPeriod() {

	vcenter := this.Ctx.Input.Param(":Vcenter")
	datacenter := this.Ctx.Input.Param(":Datacenter")
	cluster := this.Ctx.Input.Param(":Cluster")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_clusterperfperiod_"+vcenter+"_"+datacenter+"_"+cluster, 302)
		return
	}
	RegisterRequest(this, LogTypeViewLoadTable) // график загрузки кластера по данным БД за период

	this.activeContent("sprsrv/clusterperfperiod")

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster

	this.Data["ClusterName"] = cluster
	this.Data["RecordCount"] = 0
	this.Data["ChartsJSON"] = template.JS("[]")

	flash := beego.NewFlash()

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

	// 1) поиск кластера в справочнике по vcenter/datacenter/имя
	var sprList []models.ClusterSprRecords
	_, err := o.QueryTable("cluster_spr_records").
		Filter("Vcenter", vcenter).
		Filter("Datacenter", datacenter).
		Filter("Name", cluster).
		Limit(1).
		All(&sprList)

	if err != nil || len(sprList) == 0 {
		flash.Error("Кластер \"%s\" не найден в справочнике (cluster_spr_records).", cluster)
		flash.Store(&this.Controller)
		this.Data["ChartsJSON"] = template.JS("[]")
		this.Data["RecordCount"] = 0
		return
	}
	sprRecord := sprList[0]
	this.Data["ClusterName"] = sprRecord.Name

	// 2) агрегаты min/avg/max за период (ClusterRecord.ClusterId = sprRecord.Id,
	// Date в диапазоне [from, to] включительно)
	var clRecs []models.ClusterRecord
	_, err = o.QueryTable("cluster_record").
		Filter("ClusterId", sprRecord.Id).
		Filter("Date__gte", from.UTC()).
		Filter("Date__lt", to.AddDate(0, 0, 1).UTC()).
		OrderBy("Date").
		All(&clRecs)
	if err != nil {
		flash.Error("Ошибка загрузки данных за период: %v", err)
		flash.Store(&this.Controller)
		this.Data["ChartsJSON"] = template.JS("[]")
		this.Data["RecordCount"] = 0
		return
	}

	this.Data["RecordCount"] = len(clRecs)

	// 3) подробные данные (если есть) - ClusterPerf.ClusterId = ClusterRecord.Id
	// для каждой из найденных записей периода
	clRecIds := make([]int64, 0, len(clRecs))
	for _, r := range clRecs {
		clRecIds = append(clRecIds, r.Id)
	}

	var perfRecs []models.ClusterPerf
	if len(clRecIds) > 0 {
		_, err = o.QueryTable("cluster_perf").
			Filter("ClusterId__in", clRecIds).
			OrderBy("Start").
			All(&perfRecs)
		if err != nil {
			flash.Notice("Подробные данные (cluster_perf) не загружены: %v", err)
		}
	}

	charts := []*vmPerfPeriodChart{
		{Title: "CPU, использование", Unit: "%"},
		{Title: "Память, использование", Unit: "%"},
		{Title: "Диск, использование", Unit: "МБ/с"},
		{Title: "Диск, запись", Unit: "МБ/с"},
		{Title: "Диск, чтение", Unit: "МБ/с"},
		{Title: "Диск, задержка чтения", Unit: "мс"},
		{Title: "Диск, задержка записи", Unit: "мс"},
		{Title: "Сеть, использование", Unit: "МБ/с"},
	}
	for _, c := range charts {
		c.Min = make([]vmPerfSeriesPoint, 0, len(clRecs))
		c.Avg = make([]vmPerfSeriesPoint, 0, len(clRecs))
		c.Max = make([]vmPerfSeriesPoint, 0, len(clRecs))
		c.Raw = make([]vmPerfSeriesPoint, 0, len(perfRecs))
	}

	for _, r := range clRecs {
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

		charts[3].Min = append(charts[3].Min, vmPerfSeriesPoint{T: t, V: r.Disk_write_minimum})
		charts[3].Avg = append(charts[3].Avg, vmPerfSeriesPoint{T: t, V: r.Disk_write_average})
		charts[3].Max = append(charts[3].Max, vmPerfSeriesPoint{T: t, V: r.Disk_write_maximum})

		charts[4].Min = append(charts[4].Min, vmPerfSeriesPoint{T: t, V: r.Disk_read_minimum})
		charts[4].Avg = append(charts[4].Avg, vmPerfSeriesPoint{T: t, V: r.Disk_read_average})
		charts[4].Max = append(charts[4].Max, vmPerfSeriesPoint{T: t, V: r.Disk_read_maximum})

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
		charts[3].Raw = append(charts[3].Raw, vmPerfSeriesPoint{T: t, V: p.Disk_write})
		charts[4].Raw = append(charts[4].Raw, vmPerfSeriesPoint{T: t, V: p.Disk_read})
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
