package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// SprStorageSystemMetricsView отображает динамические графики производительности
// СХД за произвольный период - полностью по аналогии с
// controllers/sprsrvvmperfperiod.go и controllers/sprsrvclusterperfperiod.go
// (тот же vmPerfPeriodChart/vmPerfSeriesPoint, тот же ApexCharts/PDF шаблон),
// только источник данных - models.StorageMetricRecord (суточные агрегаты
// min/avg/max, cmdb.SaveStorageMetricRecord) и models.StorageMetric (сырые
// поминутные снимки, хранятся 3 суток - cmdb.collectstoragemetrics.go,
// storageMetricRetention) вместо VMRecord/VMPerfRecord.
//
// СХД ищется по числовому Id (как storagesystemconfigview/storagesystemsedit),
// а не по составному пути vcenter/datacenter/... как у VM/кластера.
//
// Вызывается по правому клику на /sprsrv/storagesystemslist ("Метрики СХД" в
// контекстном меню).
//
// Маршрут: /sprsrv/storagesystemmetricsview/:id
func (this *MainController) SprStorageSystemMetricsView() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewLoadTable) // график производительности СХД по данным БД за период

	this.activeContent("sprsrv/storagesystemmetricsview")

	this.Data["Id"] = idStr
	this.Data["ShortName"] = ""
	this.Data["IpAddress"] = ""
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

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()

	var shd models.StorageSystems
	if err := o.QueryTable("storage_systems").Filter("Id", id).One(&shd); err != nil {
		flash.Error("СХД с Id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["ShortName"] = shd.ShortName
	this.Data["IpAddress"] = shd.IpAddress

	// 1) агрегаты min/avg/max за период (StorageMetricRecord.StorageSystemsId = id,
	// Date в диапазоне [from, to] включительно)
	var recs []models.StorageMetricRecord
	_, err = o.QueryTable("storage_metric_record").
		Filter("StorageSystemsId", id).
		Filter("Date__gte", from.UTC()).
		Filter("Date__lt", to.AddDate(0, 0, 1).UTC()).
		OrderBy("Date").
		All(&recs)
	if err != nil {
		flash.Error("Ошибка загрузки данных за период: %v", err)
		flash.Store(&this.Controller)
		return
	}

	this.Data["RecordCount"] = len(recs)

	// 2) сырые поминутные снимки (models.StorageMetric, хранятся только 3
	// суток - см. storageMetricRetention в cmdb/collectstoragemetrics.go),
	// без join через Id суточной записи - StorageMetric и так фильтруется
	// напрямую по StorageSystemsId/CollectedAt.
	var rawRecs []models.StorageMetric
	_, err = o.QueryTable("storage_metric").
		Filter("StorageSystemsId", id).
		Filter("CollectedAt__gte", from.UTC()).
		Filter("CollectedAt__lt", to.AddDate(0, 0, 1).UTC()).
		OrderBy("CollectedAt").
		All(&rawRecs)
	if err != nil {
		// подробные данные не обязательны для отображения графиков -
		// логируем через flash.Notice и продолжаем без них
		flash.Notice("Подробные данные (storage_metric) не загружены: %v", err)
	}

	charts := []*vmPerfPeriodChart{
		{Title: "CPU, использование", Unit: "%"},
		{Title: "Заполнение пулов", Unit: "%"},
		{Title: "IOPS, чтение", Unit: "IOPS"},
		{Title: "IOPS, запись", Unit: "IOPS"},
		{Title: "IOPS, всего", Unit: "IOPS"},
		{Title: "Пропускная способность, чтение", Unit: "МБ/с"},
		{Title: "Пропускная способность, запись", Unit: "МБ/с"},
		{Title: "Задержка чтения", Unit: "мс"},
		{Title: "Задержка записи", Unit: "мс"},
	}
	for _, c := range charts {
		c.Min = make([]vmPerfSeriesPoint, 0, len(recs))
		c.Avg = make([]vmPerfSeriesPoint, 0, len(recs))
		c.Max = make([]vmPerfSeriesPoint, 0, len(recs))
		c.Raw = make([]vmPerfSeriesPoint, 0, len(rawRecs))
	}

	for _, r := range recs {
		t := r.Date.UnixMilli()

		charts[0].Min = append(charts[0].Min, vmPerfSeriesPoint{T: t, V: r.CPUUtilMinimum})
		charts[0].Avg = append(charts[0].Avg, vmPerfSeriesPoint{T: t, V: r.CPUUtilAverage})
		charts[0].Max = append(charts[0].Max, vmPerfSeriesPoint{T: t, V: r.CPUUtilMaximum})

		charts[1].Min = append(charts[1].Min, vmPerfSeriesPoint{T: t, V: r.UsedPercentMinimum})
		charts[1].Avg = append(charts[1].Avg, vmPerfSeriesPoint{T: t, V: r.UsedPercentAverage})
		charts[1].Max = append(charts[1].Max, vmPerfSeriesPoint{T: t, V: r.UsedPercentMaximum})

		charts[2].Min = append(charts[2].Min, vmPerfSeriesPoint{T: t, V: r.ReadIOPSMinimum})
		charts[2].Avg = append(charts[2].Avg, vmPerfSeriesPoint{T: t, V: r.ReadIOPSAverage})
		charts[2].Max = append(charts[2].Max, vmPerfSeriesPoint{T: t, V: r.ReadIOPSMaximum})

		charts[3].Min = append(charts[3].Min, vmPerfSeriesPoint{T: t, V: r.WriteIOPSMinimum})
		charts[3].Avg = append(charts[3].Avg, vmPerfSeriesPoint{T: t, V: r.WriteIOPSAverage})
		charts[3].Max = append(charts[3].Max, vmPerfSeriesPoint{T: t, V: r.WriteIOPSMaximum})

		charts[4].Min = append(charts[4].Min, vmPerfSeriesPoint{T: t, V: r.TotalIOPSMinimum})
		charts[4].Avg = append(charts[4].Avg, vmPerfSeriesPoint{T: t, V: r.TotalIOPSAverage})
		charts[4].Max = append(charts[4].Max, vmPerfSeriesPoint{T: t, V: r.TotalIOPSMaximum})

		charts[5].Min = append(charts[5].Min, vmPerfSeriesPoint{T: t, V: r.ReadMBpsMinimum})
		charts[5].Avg = append(charts[5].Avg, vmPerfSeriesPoint{T: t, V: r.ReadMBpsAverage})
		charts[5].Max = append(charts[5].Max, vmPerfSeriesPoint{T: t, V: r.ReadMBpsMaximum})

		charts[6].Min = append(charts[6].Min, vmPerfSeriesPoint{T: t, V: r.WriteMBpsMinimum})
		charts[6].Avg = append(charts[6].Avg, vmPerfSeriesPoint{T: t, V: r.WriteMBpsAverage})
		charts[6].Max = append(charts[6].Max, vmPerfSeriesPoint{T: t, V: r.WriteMBpsMaximum})

		charts[7].Min = append(charts[7].Min, vmPerfSeriesPoint{T: t, V: r.ReadLatencyMsMinimum})
		charts[7].Avg = append(charts[7].Avg, vmPerfSeriesPoint{T: t, V: r.ReadLatencyMsAverage})
		charts[7].Max = append(charts[7].Max, vmPerfSeriesPoint{T: t, V: r.ReadLatencyMsMaximum})

		charts[8].Min = append(charts[8].Min, vmPerfSeriesPoint{T: t, V: r.WriteLatencyMsMinimum})
		charts[8].Avg = append(charts[8].Avg, vmPerfSeriesPoint{T: t, V: r.WriteLatencyMsAverage})
		charts[8].Max = append(charts[8].Max, vmPerfSeriesPoint{T: t, V: r.WriteLatencyMsMaximum})
	}

	for _, p := range rawRecs {
		t := p.CollectedAt.UnixMilli()
		charts[0].Raw = append(charts[0].Raw, vmPerfSeriesPoint{T: t, V: p.CPUUtilPercent})
		charts[1].Raw = append(charts[1].Raw, vmPerfSeriesPoint{T: t, V: p.UsedPercent})
		charts[2].Raw = append(charts[2].Raw, vmPerfSeriesPoint{T: t, V: p.ReadIOPS})
		charts[3].Raw = append(charts[3].Raw, vmPerfSeriesPoint{T: t, V: p.WriteIOPS})
		charts[4].Raw = append(charts[4].Raw, vmPerfSeriesPoint{T: t, V: p.TotalIOPS})
		charts[5].Raw = append(charts[5].Raw, vmPerfSeriesPoint{T: t, V: p.ReadMBps})
		charts[6].Raw = append(charts[6].Raw, vmPerfSeriesPoint{T: t, V: p.WriteMBps})
		charts[7].Raw = append(charts[7].Raw, vmPerfSeriesPoint{T: t, V: p.ReadLatencyMs})
		charts[8].Raw = append(charts[8].Raw, vmPerfSeriesPoint{T: t, V: p.WriteLatencyMs})
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
