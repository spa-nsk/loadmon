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

// storageRecentChart - данные одного графика "за последние 3 суток": один
// ряд поминутных точек из models.StorageMetric (без суточной агрегации -
// в отличие от controllers/sprstoragesystemmetricsview.go, здесь график не
// зависит от models.StorageMetricRecord).
type storageRecentChart struct {
	Title  string              `json:"title"`
	Unit   string              `json:"unit"`
	Points []vmPerfSeriesPoint `json:"points"`
}

// storageMetricRecentWindow - глубина графика "за последние 3 суток" -
// намеренно равна storageMetricRetention (cmdb/collectstoragemetrics.go):
// более старых поминутных данных в storage_metric физически нет (retention
// чистит их отдельным циклом, см. cmdb.ClearOldStorageMetric), поэтому
// более широкое окно здесь бессмысленно.
const storageMetricRecentWindow = time.Hour * 24 * 3

// SprStorageSystemMetricsRecent отображает динамический график
// производительности СХД за последние 3 суток НАПРЯМУЮ по поминутным
// данным models.StorageMetric - в отличие от
// controllers/sprstoragesystemmetricsview.go (который строится на суточных
// агрегатах models.StorageMetricRecord и показывает произвольный период),
// эта страница не зависит от суточной агрегации (cmdb.SaveStorageMetricRecord,
// запускается раз в сутки) и всегда показывает то, что реально уже собрано
// опросом (cmdb.CollectStorageMetrics, раз в минуту) - удобно как самая
// свежая картина без ожидания ночной агрегации.
//
// Вызывается по правому клику на /sprsrv/storagesystemslist ("Метрики СХД за
// 3 суток" в контекстном меню).
//
// Маршрут: /sprsrv/storagesystemmetricsrecent/:id
func (this *MainController) SprStorageSystemMetricsRecent() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewLoadTable) // график производительности СХД за последние 3 суток по storage_metric

	this.activeContent("sprsrv/storagesystemmetricsrecent")

	this.Data["Id"] = idStr
	this.Data["ShortName"] = ""
	this.Data["IpAddress"] = ""
	this.Data["RecordCount"] = 0
	this.Data["ChartsJSON"] = template.JS("[]")

	flash := beego.NewFlash()

	to := time.Now().Local()
	from := to.Add(-storageMetricRecentWindow)

	this.Data["FromLabel"] = from.Format("02.01.2006 15:04")
	this.Data["ToLabel"] = to.Format("02.01.2006 15:04")

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

	var samples []models.StorageMetric
	_, err = o.QueryTable("storage_metric").
		Filter("StorageSystemsId", id).
		Filter("CollectedAt__gte", from.UTC()).
		Filter("CollectedAt__lt", to.UTC()).
		OrderBy("CollectedAt").
		All(&samples)
	if err != nil {
		flash.Error("Ошибка загрузки данных storage_metric: %v", err)
		flash.Store(&this.Controller)
		return
	}

	this.Data["RecordCount"] = len(samples)

	charts := []*storageRecentChart{
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
		c.Points = make([]vmPerfSeriesPoint, 0, len(samples))
	}

	for _, s := range samples {
		t := s.CollectedAt.UnixMilli()
		charts[0].Points = append(charts[0].Points, vmPerfSeriesPoint{T: t, V: s.CPUUtilPercent})
		charts[1].Points = append(charts[1].Points, vmPerfSeriesPoint{T: t, V: s.UsedPercent})
		charts[2].Points = append(charts[2].Points, vmPerfSeriesPoint{T: t, V: s.ReadIOPS})
		charts[3].Points = append(charts[3].Points, vmPerfSeriesPoint{T: t, V: s.WriteIOPS})
		charts[4].Points = append(charts[4].Points, vmPerfSeriesPoint{T: t, V: s.TotalIOPS})
		charts[5].Points = append(charts[5].Points, vmPerfSeriesPoint{T: t, V: s.ReadMBps})
		charts[6].Points = append(charts[6].Points, vmPerfSeriesPoint{T: t, V: s.WriteMBps})
		charts[7].Points = append(charts[7].Points, vmPerfSeriesPoint{T: t, V: s.ReadLatencyMs})
		charts[8].Points = append(charts[8].Points, vmPerfSeriesPoint{T: t, V: s.WriteLatencyMs})
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
