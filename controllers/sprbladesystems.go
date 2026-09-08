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

// BladeSystemsFilter - фильтр списка /sprsrv/bladesystemslist. models.BladeSystems
// в отличие от models.StorageSystems не содержит числовых полей-перечислений
// (TypeWork/Status и т.п.) - все поля фильтра текстовые, приём (фильтр в
// сессии, постраничная навигация, экспорт) идентичен StorageSystemsFilter в
// controllers/sprstoragesystems.go (образец стиля для этой формы).
type BladeSystemsFilter struct {
	Vendor       string `form:"Vendor"`
	Model        string `form:"Model"`
	Type         string `form:"Type"`
	Name         string `form:"Name"`
	SerialNum    string `form:"SerialNum"`
	InventoryNum string `form:"InventoryNum"`
}

func defaultBladeSystemsFilter() *BladeSystemsFilter {
	return &BladeSystemsFilter{}
}

// SprBladeSystems отображает список записей справочника блейд-корзин
// (models.BladeSystems, таблица blade_systems) с фильтром по
// Vendor/Model/Type/Name/SerialNum/InventoryNum и
// постраничной навигацией, включая экспорт в CSV и XLSX - структура
// фильтра/пагинации/экспорта и оформление полностью повторяют
// /sprsrv/storagesystemslist (controllers/sprstoragesystems.go).
//
// Маршрут: /sprsrv/bladesystemslist/:id (:id зарезервирован под будущее
// использование, как и в /sprsrv/hwconflist/:id - сейчас не используется).
func (this *MainController) SprBladeSystems() {

	Id := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference) // просмотр справочника блейд-корзин

	this.activeContent("sprsrv/bladesystemslist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "BladeSystemsFilter"
	filter, ok := m[keyFilter].(*BladeSystemsFilter)
	if !ok {
		filter = defaultBladeSystemsFilter()
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
	qs := o.QueryTable("blade_systems")

	if filter.Vendor != "" {
		qs = qs.Filter("Vendor__icontains", filter.Vendor)
	}
	if filter.Model != "" {
		qs = qs.Filter("Model__icontains", filter.Model)
	}
	if filter.Type != "" {
		qs = qs.Filter("Type__icontains", filter.Type)
	}
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.SerialNum != "" {
		qs = qs.Filter("SerialNum__icontains", filter.SerialNum)
	}
	if filter.InventoryNum != "" {
		qs = qs.Filter("InventoryNum__icontains", filter.InventoryNum)
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
	records := make([]models.BladeSystems, 0, pageSize)
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
		this.exportBladeSystemsXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportBladeSystemsCSV(records, flash)
		return
	}

	this.renderBladeSystemsPage(o, records, totalRecords, page, pageSize)
}

// bladeSystemsPlacementByCabinetId строит по списку записей models.BladeSystems
// (только по их RackCabinetId) карту "Id шкафа -> "ЦОД / шкаф"" - готовую
// строку для колонки "Размещение" списка /sprsrv/bladesystemslist. Сделано
// (2026-09-08, по запросу пользователя) взамен убранных ранее текстовых
// BladeSystems.DatacenterRoom/RackNum (см. CHANGELOG): единственный источник
// расположения корзины теперь FK RackCabinetId, поэтому имя ЦОД и шкафа
// приходится подтягивать отдельным запросом - аналогичный lookup уже
// используется в controllers/infrastructure_canvas.go для той же цели.
func bladeSystemsPlacementByCabinetId(o orm.Ormer, rows []models.BladeSystems) map[int64]string {
	result := map[int64]string{}
	cabinetIds := make([]int64, 0, len(rows))
	seen := map[int64]bool{}
	for _, r := range rows {
		if r.RackCabinetId > 0 && !seen[r.RackCabinetId] {
			seen[r.RackCabinetId] = true
			cabinetIds = append(cabinetIds, r.RackCabinetId)
		}
	}
	if len(cabinetIds) == 0 {
		return result
	}
	var cabinets []models.RackCabinet
	if _, err := o.QueryTable(new(models.RackCabinet)).Filter("Id__in", cabinetIds).All(&cabinets); err != nil {
		return result
	}
	siteIds := make([]int64, 0, len(cabinets))
	for _, c := range cabinets {
		if c.DatacenterSiteId > 0 {
			siteIds = append(siteIds, c.DatacenterSiteId)
		}
	}
	siteNameById := map[int64]string{}
	if len(siteIds) > 0 {
		var sites []models.DatacenterSite
		if _, err := o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", siteIds).All(&sites); err == nil {
			for _, s := range sites {
				siteNameById[s.Id] = s.Name
			}
		}
	}
	for _, c := range cabinets {
		site := siteNameById[c.DatacenterSiteId]
		switch {
		case site != "" && c.Name != "":
			result[c.Id] = site + " / " + c.Name
		case c.Name != "":
			result[c.Id] = c.Name
		default:
			result[c.Id] = site
		}
	}
	return result
}

// exportBladeSystemsXLSX отдаёт найденный по фильтру список записей
// models.BladeSystems файлом XLSX на скачивание - формирование листа
// аналогично экспорту /sprsrv/storagesystemslist (exportStorageSystemsXLSX).
func (this *MainController) exportBladeSystemsXLSX(rows []models.BladeSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("bladesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: " + destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Блейд-корзины"
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

// exportBladeSystemsCSV отдаёт найденный по фильтру список записей
// models.BladeSystems файлом CSV (utf8, разделитель "|") на скачивание -
// аналогично /sprsrv/storagesystemslist (exportStorageSystemsCSV).
func (this *MainController) exportBladeSystemsCSV(rows []models.BladeSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("bladesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
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

// renderBladeSystemsPage кладёт в Data готовую страницу списка (записи +
// параметры пагинации) - аналогично renderStorageSystemsPage. Дополнительно
// строит bladeSystemsPlacementByCabinetId для колонки "Размещение" (см.
// комментарий у этой функции).
func (this *MainController) renderBladeSystemsPage(o orm.Ormer, rows []models.BladeSystems, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["BladeSystemsList"] = rows
	this.Data["BladeSystemsPlacement"] = bladeSystemsPlacementByCabinetId(o, rows)
	this.Data["BladeSystemsCount"] = totalRecords
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

// bladeSystemsDateLayout - формат для полей <input type="date"> в форме
// редактирования - аналогично storageSystemsDateLayout в
// controllers/sprstoragesystems.go.
const bladeSystemsDateLayout = "2006-01-02"

func formatBladeSystemsDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(bladeSystemsDateLayout)
}

func parseBladeSystemsDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(bladeSystemsDateLayout, s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// bladeSystemsComputeSizeUnit вычисляет BladeSystems.SizeUnit (количество
// занимаемых юнитов) по StartUnitPos/EndUnitPos - по запросу пользователя
// (2026-09-08) SizeUnit больше не вводится в форме редактирования напрямую
// (введены только начальная и конечная позиция), а всегда пересчитывается
// перед сохранением - и здесь (форма редактирования), и при импорте из
// Итиллиум (см. itilliumimport/importblades.go). При некорректных или не
// заполненных значениях (EndUnitPos < StartUnitPos, StartUnitPos <= 0)
// возвращает 0 ("не указано") вместо того, чтобы писать отрицательное или
// заведомо неверное число.
func bladeSystemsComputeSizeUnit(startUnitPos, endUnitPos int) int {
	if startUnitPos <= 0 || endUnitPos < startUnitPos {
		return 0
	}
	return endUnitPos - startUnitPos + 1
}

// SprBladeSystemsEdit отображает форму просмотра/редактирования одной записи
// models.BladeSystems (таблица blade_systems) по Id и сохраняет изменения по
// отправке формы (POST) - поля читаются вручную по имени (а не через
// this.ParseForm), чтобы не задевать служебные поля (Id_User, Time_Mfy) -
// приём идентичен SprStorageSystemsEdit в controllers/sprstoragesystems.go
// (образец стиля для этой формы).
//
// Id=0 - режим добавления новой записи (ссылка "Добавить" со страницы
// /sprsrv/bladesystemslist ведёт на /sprsrv/bladesystemsedit/0): форма
// показывается пустой, по отправке создаёт новую запись (o.Insert) и
// переходит на страницу редактирования уже созданной записи по её Id.
//
// Маршрут: /sprsrv/bladesystemsedit/:id
func (this *MainController) SprBladeSystemsEdit() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки блейд-корзины - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("sprsrv/bladesystemsedit")
	this.Data["Id"] = idStr

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}
	isNew := id == 0
	this.Data["IsNew"] = isNew

	o := orm.NewOrm()

	var rec models.BladeSystems
	if !isNew {
		if err := o.QueryTable("blade_systems").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["BS"] = rec
			return
		}
		rec.Name = this.GetString("Name")
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.Type = this.GetString("Type")
		rec.PartNum = this.GetString("PartNum")
		rec.SerialNum = this.GetString("SerialNum")
		rec.Size, _ = strconv.Atoi(this.GetString("Size"))
		rec.Used, _ = strconv.Atoi(this.GetString("Used"))

		// ИЗМЕНЕНО (2026-09-08, по запросу пользователя): в форме указываются
		// StartUnitPos и EndUnitPos, SizeUnit в форме больше не отображается -
		// вычисляется ниже перед сохранением (см.
		// bladeSystemsComputeSizeUnit).
		rec.StartUnitPos, _ = strconv.Atoi(this.GetString("StartUnitPos"))
		rec.EndUnitPos, _ = strconv.Atoi(this.GetString("EndUnitPos"))
		rec.SizeUnit = bladeSystemsComputeSizeUnit(rec.StartUnitPos, rec.EndUnitPos)
		rec.RackCabinetId, _ = strconv.ParseInt(this.GetString("RackCabinetId"), 10, 64)

		rec.Owner = this.GetString("Owner")
		rec.Anotation = this.GetString("Anotation")

		rec.PowerState, _ = strconv.Atoi(this.GetString("PowerState"))
		rec.Active, _ = strconv.Atoi(this.GetString("Active"))
		rec.TechnicalNumberIT = this.GetString("TechnicalNumberIT")

		rec.InventoryNum = this.GetString("InventoryNum")
		rec.InventoryNumBuh = this.GetString("InventoryNumBuh")
		rec.ContractDoc = this.GetString("ContractDoc")
		rec.DateContractDoc = parseBladeSystemsDate(this.GetString("DateContractDoc"))
		rec.Invoice = this.GetString("Invoice")
		rec.DateInvoice = parseBladeSystemsDate(this.GetString("DateInvoice"))
		rec.Counterparty = this.GetString("Counterparty")
		rec.ServiceContract = this.GetString("ServiceContract")
		rec.DateService = parseBladeSystemsDate(this.GetString("DateService"))

		rec.Time_Mfy = time.Now().Local()

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["BS"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference) // создание записи справочника блейд-корзин
			this.Redirect(fmt.Sprintf("/sprsrv/bladesystemsedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["BS"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference) // изменение записи справочника блейд-корзин

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	// ДОБАВЛЕНО (2026-09-08, по запросу пользователя): выбор ЦОД, по которому
	// в форме отфильтровывается список шкафов этого ЦОД (каскадный select,
	// см. bladesystemsedit.tpl) - сохраняется по-прежнему только
	// BladeSystems.RackCabinetId, ЦОД выводится для удобства выбора и на
	// сервере не читается (this.GetString("DatacenterSiteId") нигде не
	// используется).
	var sites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&sites)
	this.Data["DatacenterSites"] = sites

	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks

	this.Data["BS"] = rec
	this.Data["DateContractDocStr"] = formatBladeSystemsDate(rec.DateContractDoc)
	this.Data["DateInvoiceStr"] = formatBladeSystemsDate(rec.DateInvoice)
	this.Data["DateServiceStr"] = formatBladeSystemsDate(rec.DateService)
}

// BladeSystemServerRow - одна строка списка /sprsrv/bladesystemserverslist
// (models.ServerHardware, привязанный к корзине через BladeSystemId),
// дополненная именем сервера из справочника серверов (models.ServerSprRecords,
// ServerHardware.ServerId) - расшифровка нужна, потому что сама
// ServerHardware имени не хранит (только конфигурацию).
type BladeSystemServerRow struct {
	models.ServerHardware
	ServerName string // ServerSprRecords.Name (или NameFQDN, если короткое имя пусто) по ServerId, "" если запись оборудования ни с одним сервером не связана
}

// SprBladeSystemServersList отображает read-only список серверов
// (models.ServerHardware), физически установленных в ОДНОЙ блейд-корзине
// (models.BladeSystems, BladeSystemId), отсортированный по номеру слота
// (BladeNum) - связка "блейд-сервер <-> блейд-корзина с фиксацией позиции"
// уже существует в схеме (ServerHardware.BladeSystemId/BladeNum,
// проставляется при синхронизации с HP OneView, см.
// cmdb/initrec.go:ImportHWSprRecord) - этот список делает её видимой в
// веб-интерфейсе.
//
// Открывается по правому клику на строке /sprsrv/bladesystemslist
// ("Серверы в корзине" в контекстном меню).
//
// Маршрут: /sprsrv/bladesystemserverslist/:id (:id - Id записи models.BladeSystems)
func (this *MainController) SprBladeSystemServersList() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр серверов в блейд-корзине

	this.activeContent("sprsrv/bladesystemserverslist")
	this.Data["Id"] = idStr

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()

	var bls models.BladeSystems
	if err := o.QueryTable("blade_systems").Filter("Id", id).One(&bls); err != nil {
		flash.Error("Блейд-корзина с Id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["BS"] = bls

	var hwList []models.ServerHardware
	if _, err := o.QueryTable("server_hardware").Filter("BladeSystemId", id).OrderBy("BladeNum").All(&hwList); err != nil {
		flash.Error("Ошибка чтения списка серверов: %v", err)
		flash.Store(&this.Controller)
		return
	}

	serverIds := make([]int64, 0, len(hwList))
	for _, hw := range hwList {
		if hw.ServerId != 0 {
			serverIds = append(serverIds, hw.ServerId)
		}
	}
	nameById := make(map[int64]string, len(serverIds))
	if len(serverIds) > 0 {
		var srvsprs []models.ServerSprRecords
		if _, err := o.QueryTable("server_spr_records").Filter("Id__in", serverIds).All(&srvsprs); err == nil {
			for _, s := range srvsprs {
				name := s.Name
				if name == "" {
					name = s.NameFQDN
				}
				nameById[s.Id] = name
			}
		}
	}

	rows := make([]BladeSystemServerRow, 0, len(hwList))
	for _, hw := range hwList {
		rows = append(rows, BladeSystemServerRow{ServerHardware: hw, ServerName: nameById[hw.ServerId]})
	}
	this.Data["Servers"] = rows
	this.Data["ServersCount"] = len(rows)
}
