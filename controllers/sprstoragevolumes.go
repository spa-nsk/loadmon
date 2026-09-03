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

// StorageVolumesFilter - фильтр списка /sprsrv/storagevolumeslist. Числовое
// поле-перечисление (Status) хранится как строка ("all"/"0"/"1"/"2"), чтобы
// отличать "фильтр не применён" от осмысленного значения 0 - тот же приём,
// что и StorageSystemsFilter (controllers/sprstoragesystems.go).
type StorageVolumesFilter struct {
	Status      string `form:"Status"`
	ShortName   string `form:"ShortName"`   // мнемоническое имя СХД (StorageVolumes.ShortName)
	StorageName string `form:"StorageName"` // имя тома на СХД
	VirtualName string `form:"VirtualName"` // имя хранилища в vmware
	Vcenter     string `form:"Vcenter"`
	Cluster     string `form:"Cluster"`
	DiskType    string `form:"DiskType"`
}

func defaultStorageVolumesFilter() *StorageVolumesFilter {
	return &StorageVolumesFilter{Status: "all"}
}

// StorageVolumeRow - одна строка списка /sprsrv/storagevolumeslist
// (models.StorageVolumes), дополненная расшифровкой СХД, к которой привязан
// том (в самой таблице storage_volumes хранится только StorageSystemsId и
// мнемоническое ShortName, без производителя/модели) - "с расшифровкой
// наименования СХД" по запросу пользователя.
type StorageVolumeRow struct {
	models.StorageVolumes
	StorageSystemLabel string // "ShortName (Vendor Model)" по StorageSystemsId, либо ShortName/"—" если СХД не определена
}

// storageSystemLabel формирует расшифровку СХД для отображения в списке
// томов и на карточке детального просмотра.
func storageSystemLabel(shd *models.StorageSystems) string {
	if shd == nil {
		return ""
	}
	label := shd.ShortName
	extra := ""
	if shd.Vendor != "" || shd.Model != "" {
		extra = shd.Vendor
		if shd.Model != "" {
			if extra != "" {
				extra += " "
			}
			extra += shd.Model
		}
	}
	if extra != "" {
		if label != "" {
			label = fmt.Sprintf("%s (%s)", label, extra)
		} else {
			label = extra
		}
	}
	return label
}

// SprStorageVolumes отображает список записей справочника томов СХД
// (models.StorageVolumes, таблица storage_volumes) с фильтром по
// Status/ShortName/StorageName/VirtualName/Vcenter/Cluster/DiskType и
// постраничной навигацией, включая экспорт в CSV и XLSX - оформление,
// структура фильтра/пагинации и экспорта повторяют
// /sprsrv/storagesystemslist (controllers/sprstoragesystems.go). Строки
// дополняются расшифровкой СХД (Vendor/Model) по StorageSystemsId - одним
// IN-запросом сразу по всем встретившимся Id, а не по одному на запись
// (тот же приём, что buildStorageEventLogRows в
// controllers/adminstorageeventlog.go).
//
// По правому клику на строке - переход на карточку детального просмотра
// /sprsrv/storagevolumeview/:Id (без возможности редактирования - у
// StorageVolumes нет CRUD-формы, запись полностью заполняется
// автосинхронизацией, см. kontext/05-conventions-and-gotchas.md).
//
// Маршрут: /sprsrv/storagevolumeslist/:id (:id зарезервирован под будущее
// использование, как и в /sprsrv/storagesystemslist - сейчас не
// используется).
func (this *MainController) SprStorageVolumes() {

	Id := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference) // просмотр справочника томов СХД

	this.activeContent("sprsrv/storagevolumeslist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "StorageVolumesFilter"
	filter, ok := m[keyFilter].(*StorageVolumesFilter)
	if !ok {
		filter = defaultStorageVolumesFilter()
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
	qs := o.QueryTable("storage_volumes")

	if filter.Status != "" && filter.Status != "all" {
		if v, err := strconv.Atoi(filter.Status); err == nil {
			qs = qs.Filter("Status", v)
		}
	}
	if filter.ShortName != "" {
		qs = qs.Filter("ShortName__icontains", filter.ShortName)
	}
	if filter.StorageName != "" {
		qs = qs.Filter("StorageName__icontains", filter.StorageName)
	}
	if filter.VirtualName != "" {
		qs = qs.Filter("VirtualName__icontains", filter.VirtualName)
	}
	if filter.Vcenter != "" {
		qs = qs.Filter("Vcenter__icontains", filter.Vcenter)
	}
	if filter.Cluster != "" {
		qs = qs.Filter("Cluster__icontains", filter.Cluster)
	}
	if filter.DiskType != "" {
		qs = qs.Filter("DiskType__icontains", filter.DiskType)
	}

	totalRecords, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	// экспорт отдаёт весь найденный по фильтру список, а не только текущую
	// страницу - аналогично /sprsrv/storagesystemslist
	records := make([]models.StorageVolumes, 0, pageSize)
	if exportCSV || exportXLSX {
		if _, err := qs.OrderBy("-Id").All(&records); err != nil {
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
		if _, err := qs.OrderBy("-Id").Limit(pageSize, offset).All(&records); err != nil {
			flash.Error("Ошибка выборки данных: %v", err)
			flash.Store(&this.Controller)
			return
		}
	}

	if exportXLSX {
		this.exportStorageVolumesXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportStorageVolumesCSV(records, flash)
		return
	}

	rows := this.buildStorageVolumeRows(o, records)
	this.renderStorageVolumesPage(rows, totalRecords, page, pageSize)
}

// buildStorageVolumeRows дополняет записи томов (models.StorageVolumes)
// расшифровкой СХД (Vendor/Model по StorageSystemsId) одним IN-запросом
// сразу по всем встретившимся Id.
func (this *MainController) buildStorageVolumeRows(o orm.Ormer, records []models.StorageVolumes) []StorageVolumeRow {
	idSet := make(map[int64]struct{}, len(records))
	for _, r := range records {
		if r.StorageSystemsId != 0 {
			idSet[r.StorageSystemsId] = struct{}{}
		}
	}
	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	shdById := make(map[int64]models.StorageSystems, len(ids))
	if len(ids) > 0 {
		var systems []models.StorageSystems
		if _, err := o.QueryTable("storage_systems").Filter("Id__in", ids).All(&systems); err == nil {
			for _, s := range systems {
				shdById[s.Id] = s
			}
		}
	}

	rows := make([]StorageVolumeRow, 0, len(records))
	for _, r := range records {
		row := StorageVolumeRow{StorageVolumes: r}
		if shd, ok := shdById[r.StorageSystemsId]; ok {
			row.StorageSystemLabel = storageSystemLabel(&shd)
		} else if r.ShortName != "" {
			row.StorageSystemLabel = r.ShortName
		}
		rows = append(rows, row)
	}
	return rows
}

// renderStorageVolumesPage кладёт в Data готовую страницу списка (записи +
// параметры пагинации) - аналогично renderStorageSystemsPage.
func (this *MainController) renderStorageVolumesPage(rows []StorageVolumeRow, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["StorageVolumesList"] = rows
	this.Data["StorageVolumesCount"] = totalRecords
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

// exportStorageVolumesXLSX отдаёт найденный по фильтру список записей
// models.StorageVolumes файлом XLSX на скачивание - формирование листа
// аналогично /sprsrv/storagesystemslist (exportStorageSystemsXLSX).
func (this *MainController) exportStorageVolumesXLSX(rows []models.StorageVolumes, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("storagevolumes_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Справочник томов"
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

// exportStorageVolumesCSV отдаёт найденный по фильтру список записей
// models.StorageVolumes файлом CSV (utf8, разделитель "|") на скачивание -
// аналогично /sprsrv/storagesystemslist (exportStorageSystemsCSV).
func (this *MainController) exportStorageVolumesCSV(rows []models.StorageVolumes, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("storagevolumes_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
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

// SprStorageVolumeView отображает карточку детального просмотра ОДНОЙ
// записи models.StorageVolumes по Id - строго read-only, без формы
// сохранения: у StorageVolumes нет CRUD (запись целиком заполняется
// автосинхронизацией - vSphere-скан либо опрос API СХД, см.
// kontext/05-conventions-and-gotchas.md), поэтому в отличие от
// /sprsrv/storagesystemsedit здесь нет ни POST-обработки, ни кнопки
// "Сохранить". Открывается по правому клику на строке
// /sprsrv/storagevolumeslist ("Просмотр" в контекстном меню).
//
// Маршрут: /sprsrv/storagevolumeview/:id
func (this *MainController) SprStorageVolumeView() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки тома СХД

	this.activeContent("sprsrv/storagevolumeview")
	this.Data["Id"] = idStr

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()

	var rec models.StorageVolumes
	if err := o.QueryTable("storage_volumes").Filter("Id", id).One(&rec); err != nil {
		flash.Error("Запись с Id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["ST"] = rec

	if rec.StorageSystemsId != 0 {
		var shd models.StorageSystems
		if err := o.QueryTable("storage_systems").Filter("Id", rec.StorageSystemsId).One(&shd); err == nil {
			this.Data["SHD"] = shd
			this.Data["StorageSystemLabel"] = storageSystemLabel(&shd)
		}
	}
}
