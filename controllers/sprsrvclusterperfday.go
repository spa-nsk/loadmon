package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/grpcclient"
	vsphere "loadmon/vspheremetrics"
	"sort"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

// SprSrvClusterPerfDay отображает динамические графики по всем метрикам
// производительности кластера (vsphere.Cluster.Perf, []*CLPerf) за текущий
// календарный день. Полностью аналогичен controllers/sprsrvvmperfday.go,
// но источником данных служит один кластер (grpcclient.GetClusterFull),
// а не одна виртуальная машина.
//
// Маршрут: /sprsrv/clusterperfday/:Vcenter/:Datacenter/:Cluster
func (this *MainController) SprSrvClusterPerfDay() {

	vcenter := this.Ctx.Input.Param(":Vcenter")
	datacenter := this.Ctx.Input.Param(":Datacenter")
	cluster := this.Ctx.Input.Param(":Cluster")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_clusterperfday_"+vcenter+"_"+datacenter+"_"+cluster, 302)
		return
	}
	RegisterRequest(this, LogTypeViewLoadTable) // график загрузки кластера за текущий день

	this.activeContent("sprsrv/clusterperfday")

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster

	// значения по умолчанию - шаблон читает их до того, как загрузка
	// кластера могла их переопределить (см. аналогичный приём в
	// sprsrvvmperfday.go)
	this.Data["ClusterName"] = cluster
	this.Data["ChartsJSON"] = template.JS("[]")
	this.Data["PerfCount"] = 0

	flash := beego.NewFlash()

	cl := grpcclient.GetClusterFull(vcenter, datacenter, cluster)
	if cl == nil {
		flash.Error("Кластер не найден: %s / %s / %s", vcenter, datacenter, cluster)
		flash.Store(&this.Controller)
		return
	}

	this.Data["ClusterName"] = cl.Name

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	todayPerf := make([]*vsphere.CLPerf, 0, len(cl.Perf))
	for _, p := range cl.Perf {
		if p == nil {
			continue
		}
		if p.Start.Before(dayStart) || !p.Start.Before(dayEnd) {
			continue
		}
		todayPerf = append(todayPerf, p)
	}
	sort.Slice(todayPerf, func(i, j int) bool {
		return todayPerf[i].Start.Before(todayPerf[j].Start)
	})

	this.Data["PerfCount"] = len(todayPerf)

	// группировка метрик по графикам - те же смысловые группы, что и для
	// ВМ (CPU, память, диск, сеть), плюс отдельный график загрузки CPU в %
	// от выделенных ресурсов (CPUUtilization), специфичный для кластера
	charts := []vmPerfChartData{
		{Title: "CPU, использование", Unit: "%", Series: map[string][]vmPerfSeriesPoint{
			"CPU usage": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
		}},
		{Title: "CPU, утилизация", Unit: "%", Series: map[string][]vmPerfSeriesPoint{
			"CPU utilization": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
		}},
		{Title: "Память, использование", Unit: "%", Series: map[string][]vmPerfSeriesPoint{
			"MEM usage": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
		}},
		{Title: "Диск, использование", Unit: "МБ/с", Series: map[string][]vmPerfSeriesPoint{
			"Disk usage": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
		}},
		{Title: "Диск, чтение/запись", Unit: "МБ/с", Series: map[string][]vmPerfSeriesPoint{
			"Disk read":  make([]vmPerfSeriesPoint, 0, len(todayPerf)),
			"Disk write": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
		}},
		{Title: "Сеть, использование", Unit: "МБ/с", Series: map[string][]vmPerfSeriesPoint{
			"Net usage": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
		}},
		{Title: "Сеть, приём/передача", Unit: "МБ/с", Series: map[string][]vmPerfSeriesPoint{
			"Net received":    make([]vmPerfSeriesPoint, 0, len(todayPerf)),
			"Net transmitted": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
		}},
	}

	for _, p := range todayPerf {
		t := p.Start.UnixMilli()
		charts[0].Series["CPU usage"] = append(charts[0].Series["CPU usage"], vmPerfSeriesPoint{T: t, V: p.CPUUsage})
		charts[1].Series["CPU utilization"] = append(charts[1].Series["CPU utilization"], vmPerfSeriesPoint{T: t, V: p.CPUUtilization})
		charts[2].Series["MEM usage"] = append(charts[2].Series["MEM usage"], vmPerfSeriesPoint{T: t, V: p.MemUsage})
		charts[3].Series["Disk usage"] = append(charts[3].Series["Disk usage"], vmPerfSeriesPoint{T: t, V: p.DiskUsage})
		charts[4].Series["Disk read"] = append(charts[4].Series["Disk read"], vmPerfSeriesPoint{T: t, V: p.DiskRead})
		charts[4].Series["Disk write"] = append(charts[4].Series["Disk write"], vmPerfSeriesPoint{T: t, V: p.DiskWrite})
		charts[5].Series["Net usage"] = append(charts[5].Series["Net usage"], vmPerfSeriesPoint{T: t, V: p.NetUsage})
		charts[6].Series["Net received"] = append(charts[6].Series["Net received"], vmPerfSeriesPoint{T: t, V: p.NetReceived})
		charts[6].Series["Net transmitted"] = append(charts[6].Series["Net transmitted"], vmPerfSeriesPoint{T: t, V: p.NetTransmitted})
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
