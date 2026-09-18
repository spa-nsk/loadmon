package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/models"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// ClusterRecordExportRow - строка выгрузки метрик кластера из cluster_record
// за период, дополненная в начале колонками Vcenter/Datacenter/Cluster - по
// аналогии с VMRecordExportRow (controllers/sprsrvvmmetricsexport.go). В
// отличие от VMRecord, у ClusterRecord эти поля уже хранятся прямо в самой
// записи (cmdb.SaveClusterRecord заполняет их из cluster_spr_records на
// момент сохранения) - дополнительный join не нужен, значения берутся с той
// же исторической записи (то, что было актуально на дату записи, а не
// текущее состояние справочника).
type ClusterRecordExportRow struct {
	Vcenter    string `csv:"Vcenter"`
	Datacenter string `csv:"Datacenter"`
	Cluster    string `csv:"Cluster"`
	models.ClusterRecord
}

// SprClusterMetricsExport отображает форму фильтра (vCenter/датацентр/кластер
// + период from/to) и предпросмотр метрик кластеров из cluster_record за
// период; по ?export=csv/?export=xlsx выгружает файлом всю отфильтрованную
// выборку (не только текущую страницу предпросмотра, см.
// kontext/06-list-page-pattern.md, п.5). Аналог SprVMMetricsExport
// (controllers/sprsrvvmmetricsexport.go), но по таблице cluster_record.
//
// Маршрут: /sprsrv/clustermetricsexport, доступен любому авторизованному
// пользователю (как и /sprsrv/clusterperioddb - собственных ограничений
// прав, кроме общей проверки сессии в AuthFilter, здесь нет).
func (this *MainController) SprClusterMetricsExport() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	this.activeContent("sprsrv/clustermetricsexport")

	vcenter := this.GetString("vcenter")
	datacenter := this.GetString("datacenter")
	cluster := this.GetString("cluster")
	if vcenter == "" {
		vcenter = "all"
	}
	if datacenter == "" {
		datacenter = "all"
	}
	if cluster == "" {
		cluster = "all"
	}

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
	dateFrom := from.UTC()
	dateTo := to.AddDate(0, 0, 1).UTC() // включительно до конца дня "to"

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster
	this.Data["FromDate"] = from.Format("2006-01-02")
	this.Data["ToDate"] = to.Format("2006-01-02")

	flash := beego.NewFlash()
	o := orm.NewOrm()

	// справочник кластеров - источник списков значений фильтра и точка входа
	// для отбора подходящих под фильтр ClusterId (тот же приём, что и на
	// /sprsrv/clusterperioddb, controllers/sprsrvclusterperioddb.go).
	var sprAll []models.ClusterSprRecords
	if _, err := o.QueryTable("cluster_spr_records").All(&sprAll); err != nil {
		flash.Error("Ошибка чтения справочника кластеров: %v", err)
		flash.Store(&this.Controller)
		this.Data["Rows"] = []ClusterRecordExportRow{}
		return
	}

	vcSet := make(map[string]struct{})
	dcSet := make(map[string]struct{})
	clSet := make(map[string]struct{})
	matchedIds := make([]int64, 0, len(sprAll))
	for i := range sprAll {
		s := &sprAll[i]
		vcSet[s.Vcenter] = struct{}{}
		if vcenter == "all" || s.Vcenter == vcenter {
			dcSet[s.Datacenter] = struct{}{}
		}
		if (vcenter == "all" || s.Vcenter == vcenter) && (datacenter == "all" || s.Datacenter == datacenter) {
			clSet[s.Name] = struct{}{}
		}

		if (vcenter == "all" || s.Vcenter == vcenter) &&
			(datacenter == "all" || s.Datacenter == datacenter) &&
			(cluster == "all" || s.Name == cluster) {
			matchedIds = append(matchedIds, s.Id)
		}
	}
	this.Data["VcenterList"] = toSortedSlice(vcSet)
	this.Data["DatacenterList"] = toSortedSlice(dcSet)
	this.Data["ClusterList"] = toSortedSlice(clSet)

	rows := make([]ClusterRecordExportRow, 0, 1024)
	if len(matchedIds) > 0 {
		var recs []models.ClusterRecord
		if _, err := o.QueryTable("cluster_record").
			Filter("ClusterId__in", matchedIds).
			Filter("Date__gte", dateFrom).
			Filter("Date__lt", dateTo).
			OrderBy("ClusterId", "Date").
			All(&recs); err != nil {
			flash.Error("Ошибка чтения данных кластеров за период (cluster_record): %v", err)
			flash.Store(&this.Controller)
		}
		for i := range recs {
			rows = append(rows, ClusterRecordExportRow{
				Vcenter:       recs[i].Vcenter,
				Datacenter:    recs[i].Datacenter,
				Cluster:       recs[i].Name,
				ClusterRecord: recs[i],
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Cluster != rows[j].Cluster {
			return rows[i].Cluster < rows[j].Cluster
		}
		return rows[i].Date.Before(rows[j].Date)
	})

	switch this.GetString("export") {
	case "xlsx":
		RegisterRequest(this, LogTypeExport) // выгрузка метрик кластеров за период в XLSX
		exportClusterMetricsXLSX(this, flash, rows)
		return
	case "csv":
		RegisterRequest(this, LogTypeExport) // выгрузка метрик кластеров за период в CSV
		exportClusterMetricsCSV(this, flash, rows)
		return
	}

	RegisterRequest(this, LogTypeViewLoadTable) // просмотр таблицы метрик кластеров за период (по данным БД)

	// Постраничный просмотр на экране (kontext/06-list-page-pattern.md, п.4) -
	// пагинация только для отображения, сама выборка rows строится по всему
	// отфильтрованному набору (нужна целиком и для подсчёта количества
	// страниц, и для CSV/XLSX-выгрузки выше).
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
	totalRecords := len(rows)
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
	pageRows := make([]ClusterRecordExportRow, 0)
	if offset < totalRecords {
		pageRows = rows[offset:end]
	}

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

// exportClusterMetricsXLSX отдаёт выгрузку метрик кластеров за период файлом
// XLSX на скачивание - формирование листа аналогично exportVMMetricsXLSX
// (controllers/sprsrvvmmetricsexport.go).
func exportClusterMetricsXLSX(this *MainController, flash *beego.FlashData, rows []ClusterRecordExportRow) {
	if len(rows) == 0 {
		flash.Notice("Нет данных для выгрузки за выбранный период и фильтр.")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("cluster_metrics_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Метрики кластеров"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &header); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}

	for j := range rows {
		row, _ := csvtools.GetCSVData(&rows[j])
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			file.Close()
			this.Redirect("/sprsrv/clustermetricsexport", 302)
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/clustermetricsexport", 302)
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

// exportClusterMetricsCSV отдаёт выгрузку метрик кластеров за период файлом
// CSV (utf8, разделитель "|") на скачивание - формирование аналогично
// exportVMMetricsCSV (controllers/sprsrvvmmetricsexport.go).
func exportClusterMetricsCSV(this *MainController, flash *beego.FlashData, rows []ClusterRecordExportRow) {
	if len(rows) == 0 {
		flash.Notice("Нет данных для выгрузки за выбранный период и фильтр.")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("cluster_metrics_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	csvFile, err := os.Create(destination)
	if err != nil {
		flash.Error("Ошибка создания CSV файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/clustermetricsexport", 302)
		return
	}

	csvwriter := csv.NewWriter(csvFile)
	csvwriter.Comma = '|'
	header, _ := csvtools.GetCSVHeader(&rows[0])
	_ = csvwriter.Write(header)
	for j := range rows {
		data, _ := csvtools.GetCSVData(&rows[j])
		_ = csvwriter.Write(data)
	}
	csvwriter.Flush()
	csvFile.Close()

	this.Ctx.Output.Header("Content-Disposition", "attachment; filename="+shortName)
	this.Ctx.Output.Header("Content-Description", "File Transfer")
	this.Ctx.Output.Header("Content-Type", "application/csv")
	this.Ctx.Output.Header("Content-Transfer-Encoding", "text")
	this.Ctx.Output.Header("Expires", "0")
	this.Ctx.Output.Header("Cache-Control", "must-revalidate")
	this.Ctx.Output.Header("Pragma", "public")

	http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)
}
