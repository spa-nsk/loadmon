package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/models"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// ClusterSprFilter - фильтр списка /sprsrv/clustersprlist. Числовые
// поля-перечисления (TypeWork/Stat) хранятся как строки ("all"/"0"/"1"/...),
// чтобы отличать "фильтр не применён" от осмысленного значения 0 ("не
// указано" в самой таблице models.ClusterSprRecords) - приём идентичен
// StorageSystemsFilter в controllers/sprstoragesystems.go (образец стиля для
// этой формы).
type ClusterSprFilter struct {
	TypeWork   string `form:"TypeWork"`
	Stat       string `form:"Stat"`
	Name       string `form:"Name"`
	Vcenter    string `form:"Vcenter"`
	Datacenter string `form:"Datacenter"`
	ClusterId  string `form:"ClusterId"`
}

func defaultClusterSprFilter() *ClusterSprFilter {
	return &ClusterSprFilter{TypeWork: "all", Stat: "all"}
}

// SprClusterList отображает список записей справочника кластеров
// (models.ClusterSprRecords, таблица cluster_spr_records) с фильтром по
// TypeWork/Stat/Name/Vcenter/Datacenter/ClusterId и постраничной навигацией,
// включая экспорт в CSV и XLSX - оформление, структура фильтра/пагинации и
// экспорта полностью повторяют /sprsrv/storagesystemslist
// (controllers/sprstoragesystems.go). Справочник наполняется автоматически
// (см. cmdb) - добавление новых записей вручную с этой страницы не
// предусмотрено, доступна только правка существующих записей.
//
// Маршрут: /sprsrv/clustersprlist/:id (:id зарезервирован под будущее
// использование, как и в /sprsrv/storagesystemslist/:id - сейчас не
// используется).
func (this *MainController) SprClusterList() {

	Id := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference) // просмотр справочника кластеров

	this.activeContent("sprsrv/clustersprlist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "ClusterSprFilter"
	filter, ok := m[keyFilter].(*ClusterSprFilter)
	if !ok {
		filter = defaultClusterSprFilter()
		m[keyFilter] = filter
	}

	page := 1
	pageSize := 50
	exportCSV := false
	exportXLSX := false

	if this.Ctx.Input.Method() == "POST" {
		if err := this.ParseForm(filter); err != nil {
			flash.Error("Ошибка распознавания формы фильтра: %v", err)
			flash.Store(&this.Controller)
			this.Data[keyFilter] = filter
			return
		}
		delete(m, keyFilter)
		m[keyFilter] = filter

		if p, err := strconv.Atoi(this.GetString("page")); err == nil && p > 0 {
			page = p
		}
		if s, err := strconv.Atoi(this.GetString("pagesize")); err == nil {
			switch s {
			case 25, 50, 100, 200:
				pageSize = s
			}
		}

		exportCSV = this.GetString("ExportBTN") == "Экспорт в CSV"
		exportXLSX = this.GetString("ExportBTN2") == "Экспорт в XLSX"
	}

	this.Data[keyFilter] = filter

	o := orm.NewOrm()
	qs := o.QueryTable("cluster_spr_records")

	if filter.TypeWork != "" && filter.TypeWork != "all" {
		if v, err := strconv.Atoi(filter.TypeWork); err == nil {
			qs = qs.Filter("TypeWork", v)
		}
	}
	if filter.Stat != "" && filter.Stat != "all" {
		if v, err := strconv.Atoi(filter.Stat); err == nil {
			qs = qs.Filter("Stat", v)
		}
	}
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.Vcenter != "" {
		qs = qs.Filter("Vcenter__icontains", filter.Vcenter)
	}
	if filter.Datacenter != "" {
		qs = qs.Filter("Datacenter__icontains", filter.Datacenter)
	}
	if filter.ClusterId != "" {
		qs = qs.Filter("ClusterId__icontains", filter.ClusterId)
	}

	totalRecords, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	// экспорт отдаёт весь найденный по фильтру список, а не только текущую
	// страницу - поэтому Limit/Offset здесь не применяются (аналогично
	// /sprsrv/storagesystemslist)
	records := make([]models.ClusterSprRecords, 0, pageSize)
	if exportCSV || exportXLSX {
		if _, err := qs.OrderBy("Name").All(&records); err != nil {
			flash.Error("Ошибка выборки данных: %v", err)
			flash.Store(&this.Controller)
			return
		}
	} else {
		totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
		if totalPages < 1 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}
		offset := (page - 1) * pageSize
		if _, err := qs.OrderBy("Name").Limit(pageSize, offset).All(&records); err != nil {
			flash.Error("Ошибка выборки данных: %v", err)
			flash.Store(&this.Controller)
			return
		}
	}

	if exportXLSX {
		this.exportClusterSprXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportClusterSprCSV(records, flash)
		return
	}

	this.renderClusterSprPage(records, totalRecords, page, pageSize)
}

// exportClusterSprXLSX отдаёт найденный по фильтру список записей
// models.ClusterSprRecords файлом XLSX на скачивание - формирование листа
// аналогично экспорту /sprsrv/storagesystemslist (exportStorageSystemsXLSX).
func (this *MainController) exportClusterSprXLSX(rows []models.ClusterSprRecords, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("clustersprrecords_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Справочник кластеров"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &header); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j := range rows {
		row, _ := csvtools.GetCSVData(&rows[j])
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" ", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" ", err)
		flash.Store(&this.Controller)
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

// exportClusterSprCSV отдаёт найденный по фильтру список записей
// models.ClusterSprRecords файлом CSV (utf8, разделитель "|") на скачивание -
// аналогично /sprsrv/storagesystemslist (exportStorageSystemsCSV).
func (this *MainController) exportClusterSprCSV(rows []models.ClusterSprRecords, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("clustersprrecords_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	csvFile, err := os.Create(destination)
	if err != nil {
		flash.Error("Ошибка создания CSV файла: "+destination+" ", err)
		flash.Store(&this.Controller)
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

// renderClusterSprPage кладёт в Data готовую страницу списка (записи +
// параметры пагинации) - аналогично renderStorageSystemsPage.
func (this *MainController) renderClusterSprPage(rows []models.ClusterSprRecords, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["ClusterSprList"] = rows
	this.Data["ClusterSprCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = int64(0)
	if totalRecords > 0 {
		this.Data["PageFrom"] = int64(offset) + 1
	}
	pageTo := int64(offset + pageSize)
	if pageTo > totalRecords {
		pageTo = totalRecords
	}
	this.Data["PageTo"] = pageTo
}

// SprClusterEdit отображает форму просмотра/редактирования одной записи
// models.ClusterSprRecords (таблица cluster_spr_records) по Id и сохраняет
// изменения по отправке формы (POST) - поля читаются вручную по имени (а не
// через this.ParseForm), чтобы не задевать служебные поля (Id_User,
// Time_Mfy) - приём идентичен SprStorageSystemsEdit в
// controllers/sprstoragesystems.go (образец стиля для этой формы).
//
// В отличие от SprStorageSystemsEdit добавление новой записи здесь не
// предусмотрено (Id=0 => ошибка) - справочник кластеров наполняется
// автоматически (см. cmdb), с этой страницы доступна только правка уже
// существующих записей.
//
// Маршрут: /sprsrv/clusterspredit/:id
func (this *MainController) SprClusterEdit() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки кластера - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("sprsrv/clusterspredit")
	this.Data["Id"] = idStr

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s. Добавление новых записей справочника кластеров вручную не предусмотрено - справочник наполняется автоматически.", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()

	var rec models.ClusterSprRecords
	if err := o.QueryTable("cluster_spr_records").Filter("Id", id).One(&rec); err != nil {
		flash.Error("Запись с Id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		return
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["CL"] = rec
			return
		}

		rec.Vcenter = this.GetString("Vcenter")
		rec.Datacenter = this.GetString("Datacenter")
		rec.Name = this.GetString("Name")
		rec.ClusterId = this.GetString("ClusterId")
		rec.Annotation = this.GetString("Annotation")
		rec.TypeWork, _ = strconv.Atoi(this.GetString("TypeWork"))
		rec.Stat, _ = strconv.Atoi(this.GetString("Stat"))

		var currentUserId int64
		if uid, ok := this.Data["UserId"].(int64); ok {
			currentUserId = uid
		}
		rec.Id_User = currentUserId
		rec.Time_Mfy = time.Now().Local()

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["CL"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference) // изменение записи справочника кластеров

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	this.Data["CL"] = rec
}
