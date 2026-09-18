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

// VMRecordExportRow - строка выгрузки метрик виртуальной машины из vm_record
// за период, дополненная в начале колонками Vcenter/Datacenter/Cluster/
// Vm_Name, прочитанными из связанной справочной записи vm_spr_record
// (VMRecord.VMSprId -> VMSprRecord.Id), и следом - ITSystemKey/TypeWork/
// TagName/TagDescription из справочника серверов server_spr_records
// (VMRecord.ServerId -> ServerSprRecords.Id) - по запросу пользователя.
// Остальные колонки - как в models.VMRecord (набор и подписи колонок
// определяются её собственными тегами csv, см. models/models_add.go).
type VMRecordExportRow struct {
	Vcenter        string `csv:"Vcenter"`
	Datacenter     string `csv:"Datacenter"`
	Cluster        string `csv:"Cluster"`
	Vm_Name        string `csv:"Vm_Name"`
	ITSystemKey    string `csv:"ITSystemKey"`
	TypeWork       int    `csv:"TypeWork"`
	TagName        string `csv:"TagName"`
	TagDescription string `csv:"TagDescription"`
	models.VMRecord
}

// SprVMMetricsExport отображает форму фильтра (vCenter/датацентр/кластер +
// период from/to) и предпросмотр метрик виртуальных машин из vm_record за
// период; по ?export=csv/?export=xlsx выгружает файлом всю отфильтрованную
// выборку (не только текущую страницу предпросмотра, см.
// kontext/06-list-page-pattern.md, п.5).
//
// Маршрут: /sprsrv/vmmetricsexport, доступен любому авторизованному
// пользователю (как и /sprsrv/clusterperioddb - собственных ограничений
// прав, кроме общей проверки сессии в AuthFilter, здесь нет).
func (this *MainController) SprVMMetricsExport() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	this.activeContent("sprsrv/vmmetricsexport")

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

	// справочник ВМ - источник списков значений фильтра и точка входа для
	// отбора подходящих под фильтр VMSprId (аналогично cluster_spr_records
	// на /sprsrv/clusterperioddb).
	var sprAll []models.VMSprRecord
	if _, err := o.QueryTable("vm_spr_record").All(&sprAll); err != nil {
		flash.Error("Ошибка чтения справочника виртуальных машин: %v", err)
		flash.Store(&this.Controller)
		this.Data["Rows"] = []VMRecordExportRow{}
		return
	}

	vcSet := make(map[string]struct{})
	dcSet := make(map[string]struct{})
	clSet := make(map[string]struct{})
	sprById := make(map[int64]*models.VMSprRecord, len(sprAll))
	matchedIds := make([]int64, 0, len(sprAll))
	for i := range sprAll {
		s := &sprAll[i]
		vcSet[s.Vcenter] = struct{}{}
		if vcenter == "all" || s.Vcenter == vcenter {
			dcSet[s.Datacenter] = struct{}{}
		}
		if (vcenter == "all" || s.Vcenter == vcenter) && (datacenter == "all" || s.Datacenter == datacenter) {
			clSet[s.Cluster] = struct{}{}
		}
		sprById[s.Id] = s

		if (vcenter == "all" || s.Vcenter == vcenter) &&
			(datacenter == "all" || s.Datacenter == datacenter) &&
			(cluster == "all" || s.Cluster == cluster) {
			matchedIds = append(matchedIds, s.Id)
		}
	}
	this.Data["VcenterList"] = toSortedSlice(vcSet)
	this.Data["DatacenterList"] = toSortedSlice(dcSet)
	this.Data["ClusterList"] = toSortedSlice(clSet)

	rows := make([]VMRecordExportRow, 0, 1024)
	if len(matchedIds) > 0 {
		var recs []models.VMRecord
		if _, err := o.QueryTable("vm_record").
			Filter("VMSprId__in", matchedIds).
			Filter("Date__gte", dateFrom).
			Filter("Date__lt", dateTo).
			OrderBy("VMSprId", "Date").
			All(&recs); err != nil {
			flash.Error("Ошибка чтения данных виртуальных машин за период (vm_record): %v", err)
			flash.Store(&this.Controller)
		}

		// справочник серверов - только для колонок ITSystemKey/TypeWork/
		// TagName/TagDescription, по связи VMRecord.ServerId ->
		// ServerSprRecords.Id (по запросу пользователя); один запрос по
		// всем встретившимся ServerId, а не по одной записи на строку.
		serverIdSet := make(map[int64]struct{}, len(recs))
		for i := range recs {
			if recs[i].ServerId > 0 {
				serverIdSet[recs[i].ServerId] = struct{}{}
			}
		}
		srvSprById := make(map[int64]*models.ServerSprRecords, len(serverIdSet))
		if len(serverIdSet) > 0 {
			serverIds := make([]int64, 0, len(serverIdSet))
			for id := range serverIdSet {
				serverIds = append(serverIds, id)
			}
			var srvSprList []*models.ServerSprRecords
			if _, err := o.QueryTable("server_spr_records").
				Filter("Id__in", serverIds).
				All(&srvSprList, "Id", "ITSystemKey", "TypeWork", "TagName", "TagDescription"); err != nil {
				flash.Error("Ошибка чтения справочника серверов: %v", err)
				flash.Store(&this.Controller)
			}
			for _, s := range srvSprList {
				srvSprById[s.Id] = s
			}
		}

		for i := range recs {
			spr := sprById[recs[i].VMSprId]
			if spr == nil {
				continue
			}
			row := VMRecordExportRow{
				Vcenter:    spr.Vcenter,
				Datacenter: spr.Datacenter,
				Cluster:    spr.Cluster,
				Vm_Name:    spr.Vm_Name,
				VMRecord:   recs[i],
			}
			if srvSpr := srvSprById[recs[i].ServerId]; srvSpr != nil {
				row.ITSystemKey = srvSpr.ITSystemKey
				row.TypeWork = srvSpr.TypeWork
				row.TagName = srvSpr.TagName
				row.TagDescription = srvSpr.TagDescription
			}
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Vm_Name != rows[j].Vm_Name {
			return rows[i].Vm_Name < rows[j].Vm_Name
		}
		return rows[i].Date.Before(rows[j].Date)
	})

	switch this.GetString("export") {
	case "xlsx":
		RegisterRequest(this, LogTypeExport) // выгрузка метрик виртуальных машин за период в XLSX
		exportVMMetricsXLSX(this, flash, rows)
		return
	case "csv":
		RegisterRequest(this, LogTypeExport) // выгрузка метрик виртуальных машин за период в CSV
		exportVMMetricsCSV(this, flash, rows)
		return
	}

	RegisterRequest(this, LogTypeViewLoadTable) // просмотр таблицы метрик виртуальных машин за период (по данным БД)

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
	pageRows := make([]VMRecordExportRow, 0)
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

// exportVMMetricsXLSX отдаёт выгрузку метрик виртуальных машин за период
// файлом XLSX на скачивание - формирование листа аналогично
// exportHWConfigXLSX (controllers/sprhwconfig.go).
func exportVMMetricsXLSX(this *MainController, flash *beego.FlashData, rows []VMRecordExportRow) {
	if len(rows) == 0 {
		flash.Notice("Нет данных для выгрузки за выбранный период и фильтр.")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/vmmetricsexport", 302)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("vm_metrics_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/vmmetricsexport", 302)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Метрики ВМ"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/sprsrv/vmmetricsexport", 302)
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &header); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/sprsrv/vmmetricsexport", 302)
		return
	}

	for j := range rows {
		row, _ := csvtools.GetCSVData(&rows[j])
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			file.Close()
			this.Redirect("/sprsrv/vmmetricsexport", 302)
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		this.Redirect("/sprsrv/vmmetricsexport", 302)
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/vmmetricsexport", 302)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/vmmetricsexport", 302)
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

// exportVMMetricsCSV отдаёт выгрузку метрик виртуальных машин за период
// файлом CSV (utf8, разделитель "|") на скачивание - формирование аналогично
// exportHWConfigCSV (controllers/sprhwconfig.go).
func exportVMMetricsCSV(this *MainController, flash *beego.FlashData, rows []VMRecordExportRow) {
	if len(rows) == 0 {
		flash.Notice("Нет данных для выгрузки за выбранный период и фильтр.")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/vmmetricsexport", 302)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("vm_metrics_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	csvFile, err := os.Create(destination)
	if err != nil {
		flash.Error("Ошибка создания CSV файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/vmmetricsexport", 302)
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
