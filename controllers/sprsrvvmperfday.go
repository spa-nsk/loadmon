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

// vmPerfSeriesPoint - одна точка временного ряда для графика Chart.js:
// ось X - время в миллисекундах от эпохи (Unix ms, JS Date совместим),
// ось Y - значение метрики.
type vmPerfSeriesPoint struct {
	T int64   `json:"t"`
	V float64 `json:"v"`
}

// vmPerfChartData - данные одного графика (один или несколько рядов на
// общей временной шкале, например "чтение"/"запись" диска на одном графике).
type vmPerfChartData struct {
	Title  string                         `json:"title"`
	Unit   string                         `json:"unit"`
	Series map[string][]vmPerfSeriesPoint `json:"series"`
}

// SprSrvVMPerfDay отображает динамические графики по всем метрикам
// производительности (vsphere.VMPoint.Perf, []*VMPerf) выбранной виртуальной
// машины за текущий календарный день (по времени сервера). Данные
// запрашиваются одним точечным вызовом (grpcclient.GetVMByKey) - без
// потоковой загрузки всего списка ВМ, так как нужна только одна запись.
//
// Метрики группируются по смыслу в отдельные графики: CPU, память, диск
// (использование/задержки/чтение-запись отдельно), сеть (использование/
// приём/передача) - в сумме 6 графиков на базе 10 полей VMPerf.
//
// Маршрут: /sprsrv/vmperfday/:Vcenter/:Datacenter/:Cluster/:VM
func (this *MainController) SprSrvVMPerfDay() {

	vcenter := this.Ctx.Input.Param(":Vcenter")
	datacenter := this.Ctx.Input.Param(":Datacenter")
	cluster := this.Ctx.Input.Param(":Cluster")
	vmname := this.Ctx.Input.Param(":VM")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_vmperfday_"+vcenter+"_"+datacenter+"_"+cluster+"_"+vmname, 302)
		return
	}

	this.activeContent("sprsrv/vmperfday")

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster
	this.Data["CurVM"] = vmname

	// значения по умолчанию - шаблон читает их (в т.ч. {{if ne .VMHostName ""}})
	// до того, как загрузка ВМ могла их переопределить; go-шаблоны падают на
	// сравнении nil-интерфейса со строкой, если ключ не заполнен вовсе
	this.Data["VMName"] = vmname
	this.Data["VMHostName"] = ""
	this.Data["ChartsJSON"] = template.JS("[]")
	this.Data["PerfCount"] = 0

	flash := beego.NewFlash()

	vm, err := grpcclient.GetVMByKeyWithPerf(vcenter, datacenter, cluster, vmname)
	if err != nil {
		flash.Error("Виртуальная машина не найдена: %v", err)
		flash.Store(&this.Controller)
		this.Data["ChartsJSON"] = template.JS("[]")
		this.Data["PerfCount"] = 0
		return
	}

	this.Data["VMName"] = vm.Name
	this.Data["VMHostName"] = vm.HostName

	// оставляем только точки за текущий календарный день (по локальному
	// времени сервера) и сортируем по времени - на случай, если Perf
	// пришёл не строго в хронологическом порядке
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)

	todayPerf := make([]*vsphere.VMPerf, 0, len(vm.Perf))
	for _, p := range vm.Perf {
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

	// группировка метрик по графикам: заголовок, единица измерения и
	// один или несколько именованных рядов на общей временной шкале
	charts := []vmPerfChartData{
		{Title: "CPU, использование", Unit: "%", Series: map[string][]vmPerfSeriesPoint{
			"CPU usage": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
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
		{Title: "Диск, задержки", Unit: "мс", Series: map[string][]vmPerfSeriesPoint{
			"Read latency":  make([]vmPerfSeriesPoint, 0, len(todayPerf)),
			"Write latency": make([]vmPerfSeriesPoint, 0, len(todayPerf)),
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
		charts[1].Series["MEM usage"] = append(charts[1].Series["MEM usage"], vmPerfSeriesPoint{T: t, V: p.MemUsage})
		charts[2].Series["Disk usage"] = append(charts[2].Series["Disk usage"], vmPerfSeriesPoint{T: t, V: p.DiskUsage})
		charts[3].Series["Disk read"] = append(charts[3].Series["Disk read"], vmPerfSeriesPoint{T: t, V: p.DiskRead})
		charts[3].Series["Disk write"] = append(charts[3].Series["Disk write"], vmPerfSeriesPoint{T: t, V: p.DiskWrite})
		charts[4].Series["Read latency"] = append(charts[4].Series["Read latency"], vmPerfSeriesPoint{T: t, V: p.DiskTotalReadLatency})
		charts[4].Series["Write latency"] = append(charts[4].Series["Write latency"], vmPerfSeriesPoint{T: t, V: p.DiskTotalWriteLatency})
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

	// template.JS - данные передаются в шаблон как есть, без экранирования
	// html/template (это валидный JSON, который становится JS-объектом),
	// иначе автоэкранирование html/template испортит содержимое внутри
	// <script>
	this.Data["ChartsJSON"] = template.JS(b)
}
