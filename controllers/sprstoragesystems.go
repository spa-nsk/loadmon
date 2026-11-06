package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/models"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// StorageSystemsFilter - фильтр списка /sprsrv/storagesystemslist. Числовые
// поля-перечисления (TypeWork/Status) хранятся как строки ("all"/"0"/"1"/...),
// чтобы отличать "фильтр не применён" от осмысленного значения 0 ("не
// указано" в самой таблице models.StorageSystems) - приём идентичен
// HWConfigFilter в controllers/sprhwconfig.go (образец стиля для этой формы).
// ИЗМЕНЕНО (2026-09-08, по запросу пользователя "добавь в models.StorageSystems
// поле RackCabinetId и сделай соответствующую обработку и отображение в
// справочнике СХД и форме редактирования СХД" - само поле RackCabinetId уже
// существовало в модели, см. models_add.go, доработка коснулась обработки в
// controllers/tpl): текстовые DatacenterRoom/RackNum заменены на
// RackCabinetId - выбор шкафа из справочника (с каскадным select ЦОД->Шкаф в
// шаблоне) - тот же приём, что уже сделан для HWConfigFilter в
// sprhwconfig.go.
type StorageSystemsFilter struct {
	TypeWork      string `form:"TypeWork"`
	Status        string `form:"Status"`
	ItiliumStatus string `form:"ItiliumStatus"`
	Vendor        string `form:"Vendor"`
	Model         string `form:"Model"`
	ShortName     string `form:"ShortName"`
	SerialNum     string `form:"SerialNum"`
	InventoryNum  string `form:"InventoryNum"`
	RackCabinetId string `form:"RackCabinetId"`
}

func defaultStorageSystemsFilter() *StorageSystemsFilter {
	// Status по умолчанию - "1" (Используется), а не "all" - по прямому
	// запросу пользователя ("в списке для справочника СХД по умолчанию в
	// фильтре проставь статус Используется", 2026-09-17): при первом
	// открытии /sprsrv/storagesystemslist (и после сброса фильтра) в списке
	// сразу показываются только реально эксплуатируемые СХД, а не весь
	// справочник вперемешку с демонтированными/тестовыми записями - тот же
	// принцип, что уже применён к умолчаниям TypeWork/RackCabinetId в этой
	// же структуре. ItiliumStatus - "all" (поле новое, у старых записей часто 0).
	return &StorageSystemsFilter{TypeWork: "all", Status: "1", ItiliumStatus: "all", RackCabinetId: "0"}
}

// SprStorageSystems отображает список записей справочника СХД
// (models.StorageSystems, таблица storage_systems) с фильтром по
// TypeWork/Status/Vendor/Model/ShortName/SerialNum/InventoryNum/
// DatacenterRoom/RackNum и постраничной навигацией, включая экспорт в CSV и
// XLSX - оформление, структура фильтра/пагинации и экспорта полностью
// повторяют /sprsrv/hwconflist (controllers/sprhwconfig.go), в отличие от
// которого здесь не нужен join со справочником серверов - models.StorageSystems
// самодостаточна, поэтому строки списка - это сами записи StorageSystems без
// обёртки.
//
// Маршрут: /sprsrv/storagesystemslist/:id (:id зарезервирован под будущее
// использование, как и в /sprsrv/hwconflist/:id - сейчас не используется).
func (this *MainController) SprStorageSystems() {

	Id := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference) // просмотр справочника СХД

	this.activeContent("sprsrv/storagesystemslist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "StorageSystemsFilter"
	filter, ok := m[keyFilter].(*StorageSystemsFilter)
	if !ok {
		filter = defaultStorageSystemsFilter()
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
	this.Data["ItiliumStatuses"] = models.ItiliumStatusList

	o := orm.NewOrm()

	// Справочники ЦОД/Шкаф для каскадного select-фильтра размещения (см.
	// комментарий у StorageSystemsFilter.RackCabinetId выше) - тот же приём,
	// что и в controllers/sprhwconfig.go:SprHwConfig.
	var dcSites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&dcSites)
	this.Data["DatacenterSites"] = dcSites

	var allRacks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&allRacks)
	this.Data["RackCabinets"] = allRacks

	qs := o.QueryTable("storage_systems")

	if filter.TypeWork != "" && filter.TypeWork != "all" {
		if v, err := strconv.Atoi(filter.TypeWork); err == nil {
			qs = qs.Filter("TypeWork", v)
		}
	}
	if filter.Status != "" && filter.Status != "all" {
		if v, err := strconv.Atoi(filter.Status); err == nil {
			qs = qs.Filter("Status", v)
		}
	}
	if filter.ItiliumStatus != "" && filter.ItiliumStatus != "all" {
		if v, err := strconv.Atoi(filter.ItiliumStatus); err == nil {
			qs = qs.Filter("ItiliumStatus", v)
		}
	}
	if filter.Vendor != "" {
		qs = qs.Filter("Vendor__icontains", filter.Vendor)
	}
	if filter.Model != "" {
		qs = qs.Filter("Model__icontains", filter.Model)
	}
	if filter.ShortName != "" {
		qs = qs.Filter("ShortName__icontains", filter.ShortName)
	}
	if filter.SerialNum != "" {
		qs = qs.Filter("SerialNum__icontains", filter.SerialNum)
	}
	if filter.InventoryNum != "" {
		qs = qs.Filter("InventoryNum__icontains", filter.InventoryNum)
	}
	if filter.RackCabinetId != "" && filter.RackCabinetId != "0" {
		if v, err := strconv.ParseInt(filter.RackCabinetId, 10, 64); err == nil && v > 0 {
			// ИЗМЕНЕНО (2026-09-09): RackCabinetId больше не хранится прямо
			// на StorageSystems (см. models.StorageSystemPlacement) -
			// фильтр по шкафу теперь ищет Id СХД, у которых есть хотя бы
			// одно размещение в этом шкафу, через отдельную таблицу связи -
			// тот же приём (список подходящих Id -> Filter("Id__in", ...)),
			// что уже применён в sprsrvview2.go:placementServerSprIds.
			ids, err := storageSystemsIdsByCabinetId(o, v)
			if err != nil {
				flash.Error("Ошибка выборки размещения СХД: %v", err)
				flash.Store(&this.Controller)
				return
			}
			if len(ids) == 0 {
				qs = qs.Filter("Id", int64(-1))
			} else {
				qs = qs.Filter("Id__in", ids)
			}
		}
	}

	totalRecords, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	// экспорт отдаёт весь найденный по фильтру список, а не только текущую
	// страницу - поэтому Limit/Offset здесь не применяются (аналогично
	// /sprsrv/hwconflist)
	records := make([]models.StorageSystems, 0, pageSize)
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
		this.exportStorageSystemsXLSX(records, flash)
		return
	}
	if exportCSV {
		this.exportStorageSystemsCSV(records, flash)
		return
	}

	// Размещение в колонке списка (2026-09-09) - "ЦОД / Шкаф" по ВСЕМ
	// записям StorageSystemPlacement данной СХД (может быть несколько),
	// ключ карты - Id самой СХД (не Id шкафа, как было раньше при одиночном
	// FK) - см. storageSystemsPlacementByStorageId.
	this.Data["StorageSystemsPlacement"] = storageSystemsPlacementByStorageId(o, records)

	this.renderStorageSystemsPage(records, totalRecords, page, pageSize)
}

// storageSystemsIdsByCabinetId возвращает список Id записей StorageSystems,
// у которых есть хотя бы одна запись StorageSystemPlacement в шкафу
// cabinetId - используется для фильтра списка по шкафу (RackCabinetId
// больше не хранится прямо на StorageSystems, см. models.
// StorageSystemPlacement и её комментарий).
func storageSystemsIdsByCabinetId(o orm.Ormer, cabinetId int64) ([]int64, error) {
	var placements []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("RackCabinetId", cabinetId).All(&placements, "StorageSystemsId"); err != nil {
		return nil, err
	}
	seen := map[int64]bool{}
	ids := make([]int64, 0, len(placements))
	for _, p := range placements {
		if !seen[p.StorageSystemsId] {
			seen[p.StorageSystemsId] = true
			ids = append(ids, p.StorageSystemsId)
		}
	}
	return ids, nil
}

// storageSystemsPlacementByStorageId строит по списку записей
// models.StorageSystems карту "Id СХД -> текст размещения" - для колонки
// "Размещение" в storagesystemslist.tpl. В ОТЛИЧИЕ от прежней версии этой
// функции (storageSystemsPlacementByCabinetId, ключ был Id шкафа - см.
// историю в CHANGELOG.md), ключ теперь Id самой СХД: одна СХД может занимать
// НЕСКОЛЬКО шкафов (models.StorageSystemPlacement), поэтому значение -
// список "ЦОД / Шкаф" всех её размещений, склеенный через "; " (основное,
// IsPrimary=true, - первым).
func storageSystemsPlacementByStorageId(o orm.Ormer, rows []models.StorageSystems) map[int64]string {
	result := map[int64]string{}
	if len(rows) == 0 {
		return result
	}
	storageIds := make([]int64, 0, len(rows))
	for _, r := range rows {
		storageIds = append(storageIds, r.Id)
	}

	var placements []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId__in", storageIds).OrderBy("-IsPrimary", "Id").All(&placements); err != nil || len(placements) == 0 {
		return result
	}

	cabinetIds := make([]int64, 0, len(placements))
	seenCabinet := map[int64]bool{}
	// siteOnlyIds - площадки БЕЗ конкретного шкафа (RackCabinetId==0,
	// DatacenterSiteId>0, см. models.StorageSystemPlacement.DatacenterSiteId
	// и upsertStorageSystemPlacementSiteOnly в itilliumimport/importstorages.go,
	// 2026-09-10) - собираются отдельно от cabinetIds, т.к. резолвятся
	// напрямую из DatacenterSite, а не через RackCabinet.
	siteOnlyIds := make([]int64, 0)
	seenSite := map[int64]bool{}
	for _, p := range placements {
		if p.RackCabinetId > 0 && !seenCabinet[p.RackCabinetId] {
			seenCabinet[p.RackCabinetId] = true
			cabinetIds = append(cabinetIds, p.RackCabinetId)
		}
		if p.RackCabinetId == 0 && p.DatacenterSiteId > 0 && !seenSite[p.DatacenterSiteId] {
			seenSite[p.DatacenterSiteId] = true
			siteOnlyIds = append(siteOnlyIds, p.DatacenterSiteId)
		}
	}
	cabinetText := map[int64]string{} // Id шкафа -> "ЦОД / Шкаф"
	if len(cabinetIds) > 0 {
		var cabinets []models.RackCabinet
		if _, err := o.QueryTable(new(models.RackCabinet)).Filter("Id__in", cabinetIds).All(&cabinets); err == nil {
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
					cabinetText[c.Id] = site + " / " + c.Name
				case c.Name != "":
					cabinetText[c.Id] = c.Name
				default:
					cabinetText[c.Id] = site
				}
			}
		}
	}

	// siteOnlyText - Id площадки -> "ЦОД (без шкафа)" - явная пометка "без
	// шкафа" в тексте, чтобы это не выглядело как опечатка/пропуск при
	// сравнении со строками "ЦОД / Шкаф" у остальных размещений.
	siteOnlyText := map[int64]string{}
	if len(siteOnlyIds) > 0 {
		var sites []models.DatacenterSite
		if _, err := o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", siteOnlyIds).All(&sites); err == nil {
			for _, s := range sites {
				if s.Name != "" {
					siteOnlyText[s.Id] = s.Name + " (без шкафа)"
				}
			}
		}
	}

	byStorage := map[int64][]string{}
	for _, p := range placements {
		var text string
		if p.RackCabinetId > 0 {
			text = cabinetText[p.RackCabinetId]
		} else if p.DatacenterSiteId > 0 {
			text = siteOnlyText[p.DatacenterSiteId]
		}
		if text == "" {
			continue
		}
		byStorage[p.StorageSystemsId] = append(byStorage[p.StorageSystemsId], text)
	}
	for id, texts := range byStorage {
		result[id] = strings.Join(texts, "; ")
	}
	return result
}

// exportStorageSystemsXLSX отдаёт найденный по фильтру список записей
// models.StorageSystems файлом XLSX на скачивание - формирование листа
// аналогично экспорту /sprsrv/hwconflist (exportHWConfigXLSX).
func (this *MainController) exportStorageSystemsXLSX(rows []models.StorageSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("storagesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	file := excelize.NewFile()
	if file == nil {
		flash.Error("Ошибка создания XLSX файла: %s", destination)
		flash.Store(&this.Controller)
		return
	}

	header, _ := csvtools.GetCSVHeader(&rows[0])
	sheet := "Справочник СХД"
	index, err := file.NewSheet(sheet)
	if err != nil {
		flash.Error("Ошибка создания листа XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	file.SetActiveSheet(index)

	cell, _ := excelize.CoordinatesToCellName(1, 1)
	if err := file.SetSheetRow(sheet, cell, &header); err != nil {
		flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}

	for j := range rows {
		row, _ := csvtools.GetCSVData(&rows[j])
		cell, _ := excelize.CoordinatesToCellName(1, j+2)
		if err := file.SetSheetRow(sheet, cell, &row); err != nil {
			flash.Error("Ошибка записи строки XLSX файла: "+destination+" %v", err)
			flash.Store(&this.Controller)
			file.Close()
			return
		}
	}

	if err := file.SaveAs(destination); err != nil {
		flash.Error("Ошибка сохранения XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		file.Close()
		return
	}
	if err := file.Close(); err != nil {
		flash.Error("Ошибка закрытия XLSX файла: "+destination+" %v", err)
		flash.Store(&this.Controller)
		return
	}

	fileInfo, err := os.Stat(destination)
	if err != nil {
		flash.Error("Ошибка получения данных XLSX файла: "+destination+" %v", err)
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

// exportStorageSystemsCSV отдаёт найденный по фильтру список записей
// models.StorageSystems файлом CSV (utf8, разделитель "|") на скачивание -
// аналогично /sprsrv/hwconflist (exportHWConfigCSV).
func (this *MainController) exportStorageSystemsCSV(rows []models.StorageSystems, flash *beego.FlashData) {
	if len(rows) == 0 {
		flash.Notice("Нет записей для экспорта.")
		flash.Store(&this.Controller)
		return
	}

	crt := time.Now().Local()
	shortName := fmt.Sprintf("storagesystems_%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
	destination := "./reports/" + shortName

	csvFile, err := os.Create(destination)
	if err != nil {
		flash.Error("Ошибка создания CSV файла: "+destination+" %v", err)
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

// renderStorageSystemsPage кладёт в Data готовую страницу списка (записи +
// параметры пагинации) - аналогично renderHWConfigPage.
func (this *MainController) renderStorageSystemsPage(rows []models.StorageSystems, totalRecords int64, page, pageSize int) {
	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	this.Data["StorageSystemsList"] = rows
	this.Data["StorageSystemsCount"] = totalRecords
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

// storageSystemsDateLayout - формат для полей <input type="date"> в форме
// редактирования (совпадает с форматом, который отдаёт и принимает браузер) -
// аналогично hwConfigDateLayout в controllers/sprhwconfig.go.
const storageSystemsDateLayout = "2006-01-02"

func formatStorageSystemsDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(storageSystemsDateLayout)
}

func parseStorageSystemsDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.ParseInLocation(storageSystemsDateLayout, s, time.Local)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SprStorageSystemsEdit отображает форму просмотра/редактирования одной
// записи models.StorageSystems (таблица storage_systems) по Id и сохраняет
// изменения по отправке формы (POST) - поля читаются вручную по имени (а не
// через this.ParseForm), чтобы не задевать служебные поля (Id_User,
// Time_Mfy), которые эта форма не показывает - приём идентичен
// SprHwConfigEdit в controllers/sprhwconfig.go (образец стиля для этой формы).
//
// Id=0 - режим добавления новой записи (ссылка "Добавить" со страницы
// /sprsrv/storagesystemslist ведёт на /sprsrv/storagesystemsedit/0): форма
// показывается пустой, по отправке создаёт новую запись (o.Insert) и
// переходит на страницу редактирования уже созданной записи по её Id.
//
// Маршрут: /sprsrv/storagesystemsedit/:id
func (this *MainController) SprStorageSystemsEdit() {
	sprStorageSystemsEditPage(this, "sprsrv/storagesystemsedit", this.Ctx.Input.Param(":id"))
}

// SprStorageSystemsEditTab - та же карточка СХД, что SprStorageSystemsEdit,
// но с формой на закладках (шаблон storagesystemsedit_tab.tpl).
//
// Маршрут: /sprsrv/storagesystemsedit-tab/:id([0-9]+)
func (this *MainController) SprStorageSystemsEditTab() {
	sprStorageSystemsEditPage(this, "sprsrv/storagesystemsedit_tab", this.Ctx.Input.Param(":id"))
}

// sprStorageSystemsEditPage - общая логика GET/POST формы StorageSystems.
// tpl - sprsrv/storagesystemsedit или sprsrv/storagesystemsedit_tab.
func sprStorageSystemsEditPage(this *MainController, tpl, idStr string) {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки СХД - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent(tpl)
	this.Data["Id"] = idStr
	this.Data["StorageEditBasePath"] = "/sprsrv/storagesystemsedit"
	if strings.Contains(tpl, "storagesystemsedit_tab") {
		this.Data["StorageEditBasePath"] = "/sprsrv/storagesystemsedit-tab"
	}

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

	var rec models.StorageSystems
	if !isNew {
		if err := o.QueryTable("storage_systems").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["ST"] = rec
			return
		}
		rec.UID = this.GetString("UID")
		rec.ShortName = this.GetString("ShortName")
		rec.TagName = this.GetString("TagName")
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.PartNum = this.GetString("PartNum")
		rec.SerialNum = this.GetString("SerialNum")
		rec.Firmware = this.GetString("Firmware")

		// Числовые поля формы - раньше ошибки strconv.Atoi/ParseFloat
		// молча игнорировались (значение по умолчанию 0), из-за чего
		// опечатка в поле формы тихо превращала его в 0 без единого
		// сообщения пользователю. Теперь каждая ошибка разбора собирается
		// в parseErrors и показывается одним flash.Error - запись НЕ
		// сохраняется, пока в форме есть хотя бы одно нечисловое значение
		// в числовом поле (см. блок ниже, после заполнения rec).
		var parseErrors []string
		parseIntField := func(label, formName string) int {
			s := strings.TrimSpace(this.GetString(formName))
			if s == "" {
				return 0
			}
			v, err := strconv.Atoi(s)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("%s: %q - не целое число", label, s))
				return 0
			}
			return v
		}
		parseFloatField := func(label, formName string) float64 {
			s := strings.TrimSpace(this.GetString(formName))
			if s == "" {
				return 0
			}
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("%s: %q - не число", label, s))
				return 0
			}
			return v
		}

		rec.TypeWork = parseIntField("Тип использования", "TypeWork")
		rec.Status = parseIntField("Состояние", "Status")
		rec.ItiliumStatus = parseIntField("Статус Итилиум", "ItiliumStatus")
		rec.Power = parseIntField("Питание", "Power")
		rec.ApiType = parseIntField("Тип API", "ApiType")

		rec.Size = parseIntField("Максимальное количество дисков", "Size")
		rec.Used = parseIntField("Установлено дисков", "Used")
		rec.IpAddress = this.GetString("IpAddress")
		rec.IpAddressNext = this.GetString("IpAddressNext")
		rec.TotalSizeRAW = parseFloatField("Размер дисков (сырой)", "TotalSizeRAW")
		rec.UsedPercent = parseFloatField("Использовано, %", "UsedPercent")
		rec.WWNMask = this.GetString("WWNMask")

		if len(parseErrors) > 0 {
			flash.Error("Не удалось сохранить - проверьте числовые поля: %s", strings.Join(parseErrors, "; "))
			flash.Store(&this.Controller)
			this.Data["ST"] = rec
			return
		}

		rec.Ad = this.GetString("Ad") == "on"
		rec.Ldaps = this.GetString("Ldaps") == "on"
		rec.AdCert = this.GetString("AdCert") == "on"
		rec.AdCertDate = parseStorageSystemsDate(this.GetString("AdCertDate"))

		rec.TechnicalNumberIT = this.GetString("TechnicalNumberIT")

		rec.Owner = this.GetString("Owner")
		rec.Anotation = this.GetString("Anotation")

		rec.InventoryNum = this.GetString("InventoryNum")
		rec.InventoryNumBuh = this.GetString("InventoryNumBuh")
		rec.ContractDoc = this.GetString("ContractDoc")
		rec.DateContractDoc = parseStorageSystemsDate(this.GetString("DateContractDoc"))
		rec.Invoice = this.GetString("Invoice")
		rec.DateInvoice = parseStorageSystemsDate(this.GetString("DateInvoice"))
		rec.Counterparty = this.GetString("Counterparty")
		rec.ServiceContract = this.GetString("ServiceContract")
		rec.DateService = parseStorageSystemsDate(this.GetString("DateService"))

		rec.Time_Mfy = time.Now().Local()

		if isNew {
			// SourceUpdateType проставляется Manual только при создании
			// новой записи (тот же принцип, что и для ServerSprRecords -
			// см. kontext/05-conventions-and-gotchas.md) - при ОБЫЧНОМ
			// редактировании существующей записи поле не трогаем.
			rec.SourceUpdateType = models.StorageSourceUpdateManual

			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["ST"] = rec
				return
			}
			if err := this.saveStorageSystemPlacements(o, rec.Id); err != nil {
				flash.Error("Запись создана (Id=%d), но не удалось сохранить размещение в шкафах: %v", rec.Id, err)
				flash.Store(&this.Controller)
			}
			RegisterRequest(this, LogTypeEditReference) // создание записи справочника СХД
			basePath, _ := this.Data["StorageEditBasePath"].(string)
			if basePath == "" {
				basePath = "/sprsrv/storagesystemsedit"
			}
			this.Redirect(fmt.Sprintf("%s/%d", basePath, rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["ST"] = rec
			return
		}
		if err := this.saveStorageSystemPlacements(o, rec.Id); err != nil {
			flash.Error("Запись сохранена, но не удалось сохранить размещение в шкафах: %v", err)
			flash.Store(&this.Controller)
		} else {
			flash.Notice("Запись сохранена.")
			flash.Store(&this.Controller)
		}
		RegisterRequest(this, LogTypeEditReference) // изменение записи справочника СХД
	}

	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks

	// ЦОД для каскадного select ЦОД->Шкаф в форме редактирования (2026-09-08,
	// тот же приём, что в hwconfedit.tpl).
	var dcSites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&dcSites)
	this.Data["DatacenterSites"] = dcSites

	// Размещения СХД (2026-09-09, вариант "A" - таблица связи "многие-ко-
	// многим", см. models.StorageSystemPlacement и обсуждение в
	// CHANGELOG.md) - грузятся ПОСЛЕ возможного сохранения выше, чтобы
	// форма после POST сразу отражала актуальный набор строк. Если у записи
	// пока нет ни одного размещения (новая запись либо СХД, у которой ещё
	// не указан ни один шкаф) - в шаблон передаётся ОДНА пустая строка,
	// чтобы было от чего склонировать следующую (см. JS в
	// storagesystemsedit.tpl) и чтобы форма не выглядела "сломанной".
	var placements []models.StorageSystemPlacement
	if rec.Id > 0 {
		o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId", rec.Id).OrderBy("-IsPrimary", "Id").All(&placements)
	}
	if len(placements) == 0 {
		placements = []models.StorageSystemPlacement{{}}
	}
	this.Data["Placements"] = placements

	this.Data["ST"] = rec
	this.Data["ItiliumStatuses"] = models.ItiliumStatusList
	this.Data["AdCertDateStr"] = formatStorageSystemsDate(rec.AdCertDate)
	this.Data["DateContractDocStr"] = formatStorageSystemsDate(rec.DateContractDoc)
	this.Data["DateInvoiceStr"] = formatStorageSystemsDate(rec.DateInvoice)
	this.Data["DateServiceStr"] = formatStorageSystemsDate(rec.DateService)
}

// saveStorageSystemPlacements сохраняет набор размещений (models.
// StorageSystemPlacement) записи СХД storageId по данным формы -
// параллельные массивы полей "PlacementRackCabinetId"/"PlacementStartUnitPos"/
// "PlacementEndUnitPos" (одно и то же имя поля повторяется в форме на каждую
// строку "шкаф", см. storagesystemsedit.tpl - GetStrings собирает все
// значения одноимённого поля по порядку их появления в форме, что и даёт
// параллельные срезы). Простая схема "удалить все старые размещения этой
// СХД, вставить заново из формы целиком" - количество строк размещения у
// одной СХД небольшое (единицы), поэтому производительность не важна, а
// код значительно проще diff'а по Id существующих записей.
//
// Строки с пустым/нулевым/некорректным RackCabinetId И пустым/нулевым
// DatacenterSiteId пропускаются молча (это позволяет отправлять форму с
// "пустыми" добавленными, но не заполненными строками без ошибки).
// IsPrimary=true проставляется ПЕРВОЙ успешно разобранной строке (порядок
// как в форме, обычно совпадает с порядком добавления пользователем) -
// используется там, где на экране можно показать только одно "основное"
// место размещения.
//
// 2026-09-09, по запросу пользователя "СХД может размещаться в нескольких
// шкафах" - вариант "A" (таблица связи), см. models.StorageSystemPlacement.
//
// ИЗМЕНЕНО (2026-09-10, по запросу пользователя про IT-0000041888 -
// "Местоположение указано, но данных по шкафу нет и информация по
// размещению в ЦОД пропала"): строка формы с выбранным ЦОД, но БЕЗ шкафа
// (RackCabinetId==0), теперь сохраняется как "site-only" размещение
// (models.StorageSystemPlacement.DatacenterSiteId заполнен,
// RackCabinetId==0) вместо того, чтобы молча пропадать - см. параллельный
// массив "PlacementDatacenterSiteId" (storagesystemsedit.tpl) и аналогичную
// логику на стороне импорта - itilliumimport/importstorages.go:
// upsertStorageSystemPlacementSiteOnly.
func (this *MainController) saveStorageSystemPlacements(o orm.Ormer, storageId int64) error {
	rackIds := this.GetStrings("PlacementRackCabinetId")
	startVals := this.GetStrings("PlacementStartUnitPos")
	endVals := this.GetStrings("PlacementEndUnitPos")
	dcIds := this.GetStrings("PlacementDatacenterSiteId")

	trn, err := o.Begin()
	if err != nil {
		return err
	}

	if _, err := trn.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId", storageId).Delete(); err != nil {
		trn.Rollback()
		return err
	}

	first := true
	for i, rackStr := range rackIds {
		rackId, _ := strconv.ParseInt(strings.TrimSpace(rackStr), 10, 64)
		var dcId int64
		if i < len(dcIds) {
			dcId, _ = strconv.ParseInt(strings.TrimSpace(dcIds[i]), 10, 64)
		}
		if rackId <= 0 && dcId <= 0 {
			continue // пустая/незаполненная строка формы - не ошибка, просто пропускается
		}
		p := models.StorageSystemPlacement{
			StorageSystemsId: storageId,
			IsPrimary:        first,
			Id_User:          9999,
			Time_Mfy:         time.Now().Local(),
		}
		if rackId > 0 {
			start := 0
			if i < len(startVals) {
				start, _ = strconv.Atoi(strings.TrimSpace(startVals[i]))
			}
			end := 0
			if i < len(endVals) {
				end, _ = strconv.Atoi(strings.TrimSpace(endVals[i]))
			}
			sizeUnit := 0
			if end >= start && start > 0 {
				sizeUnit = end - start + 1
			}
			p.RackCabinetId = rackId
			p.StartUnitPos = start
			p.EndUnitPos = end
			p.SizeUnit = sizeUnit
		} else {
			// Шкаф не выбран, но выбран ЦОД - site-only размещение.
			p.DatacenterSiteId = dcId
		}
		first = false
		if _, err := trn.Insert(&p); err != nil {
			trn.Rollback()
			return err
		}
	}

	return trn.Commit()
}

// StorageEnclosureWithDisks - одна полка/сторадж-процессор (models.StorageEnclosure)
// вместе со списком принадлежащих ей дисков (models.StorageDisk,
// StorageEnclosureId == Enclosure.Id) - для карточки просмотра конфигурации
// СХД (/sprsrv/storagesystemconfigview).
type StorageEnclosureWithDisks struct {
	Enclosure models.StorageEnclosure
	Disks     []models.StorageDisk
}

// PoolWithVolumes - один пул хранения (models.StoragePool) вместе со
// списком LUN, выделенных из него (models.StorageVolumes,
// PoolExternalKey == Pool.ExternalKey) - ДОБАВЛЕНО 2026-09-17 по запросу
// пользователя "нужно показать LUN на этих пулах", для карточки просмотра
// конфигурации СХД (/sprsrv/storagesystemconfigview).
type PoolWithVolumes struct {
	Pool    models.StoragePool
	Volumes []models.StorageVolumes
}

// SprStorageSystemConfigView отображает read-only карточку конфигурации
// ОДНОЙ СХД (models.StorageSystems) по Id: сама СХД, сторадж-процессоры
// (models.StorageEnclosure с EnclosureType == StorageEnclosureTypeManagement -
// полки управления/контроллеров), полки данных (EnclosureType ==
// StorageEnclosureTypeData) со своими дисками (models.StorageDisk,
// сгруппированы по StorageEnclosureId) и диски без привязки к известной
// полке отдельным списком (StorageEnclosureId == 0 или не найден среди
// прочитанных полок - бывает у СХД, где диски синхронизируются, а
// hardware-инвентарь полок для этого вендора не реализован, см.
// cmdb/syncstoragehardware.go).
//
// Все данные - результат автосинхронизации (cmdb.SyncStorageHardwareForSystem,
// см. kontext/05-conventions-and-gotchas.md), редактирования тут нет - по
// аналогии с /sprsrv/storagevolumeview у StorageVolumes.
//
// Открывается по правому клику на строке /sprsrv/storagesystemslist
// ("Конфигурация СХД" в контекстном меню).
//
// Маршрут: /sprsrv/storagesystemconfigview/:id
func (this *MainController) SprStorageSystemConfigView() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр конфигурации СХД

	this.activeContent("sprsrv/storagesystemconfigview")
	this.Data["Id"] = idStr

	flash := beego.NewFlash()

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
	this.Data["SHD"] = shd
	// ИСПРАВЛЕНО (2026-09-16, по замечанию пользователя про такую же ошибку
	// в storagevolumeview.tpl - здесь тот же баг): шаблон
	// storagesystemconfigview.tpl обращался к .SHD.DatacenterRoom/.SHD.RackNum/
	// .SHD.StartUnit - текстовым полям, удалённым из models.StorageSystems при
	// переходе на справочник RackCabinet/DatacenterSite и модель
	// StorageSystemPlacement (см. CHANGELOG (147)-(154)) - эта карточка
	// осталась не обновлена при том переходе.
	this.Data["SHDPlacementText"] = storageSystemsPlacementByStorageId(o, []models.StorageSystems{shd})[shd.Id]

	var enclosures []models.StorageEnclosure
	if _, err := o.QueryTable("storage_enclosure").Filter("StorageSystemsId", id).OrderBy("EnclosureType", "SlotNumber", "Name").All(&enclosures); err != nil {
		flash.Notice("Ошибка чтения списка полок: %v", err)
		flash.Store(&this.Controller)
	}

	var disks []models.StorageDisk
	if _, err := o.QueryTable("storage_disk").Filter("StorageSystemsId", id).OrderBy("StorageEnclosureId", "SlotNumber").All(&disks); err != nil {
		flash.Notice("Ошибка чтения списка дисков: %v", err)
		flash.Store(&this.Controller)
	}

	disksByEnclosure := make(map[int64][]models.StorageDisk, len(enclosures))
	var unassignedDisks []models.StorageDisk
	knownEnclosure := make(map[int64]struct{}, len(enclosures))
	for _, e := range enclosures {
		knownEnclosure[e.Id] = struct{}{}
	}
	for _, d := range disks {
		if _, ok := knownEnclosure[d.StorageEnclosureId]; ok && d.StorageEnclosureId != 0 {
			disksByEnclosure[d.StorageEnclosureId] = append(disksByEnclosure[d.StorageEnclosureId], d)
		} else {
			unassignedDisks = append(unassignedDisks, d)
		}
	}

	var processors []StorageEnclosureWithDisks // сторадж-процессоры (полки управления/контроллеров)
	var dataShelves []StorageEnclosureWithDisks // полки с данными
	for _, e := range enclosures {
		item := StorageEnclosureWithDisks{Enclosure: e, Disks: disksByEnclosure[e.Id]}
		if e.EnclosureType == models.StorageEnclosureTypeManagement {
			processors = append(processors, item)
		} else {
			dataShelves = append(dataShelves, item)
		}
	}

	this.Data["Processors"] = processors
	this.Data["DataShelves"] = dataShelves
	this.Data["UnassignedDisks"] = unassignedDisks
	this.Data["EnclosureCount"] = len(enclosures)
	this.Data["DiskCount"] = len(disks)

	// Пулы (models.StoragePool) - ДОБАВЛЕНО 2026-09-17 по прямому замечанию
	// пользователя "нигде не вижу отображения пулов, надо добавить в
	// карточку конфигурации СХД с указанием поля Description": пулы
	// синхронизируются для всех 4 вендоров (см. upsertStoragePool в
	// cmdb/syncstoragehardware.go, задача (164)-(165)) и уже хранятся в БД,
	// но эта карточка их не читала и не выводила вовсе - единственное
	// место, где Description пула вообще был виден, отсутствовало.
	var pools []models.StoragePool
	if _, err := o.QueryTable("storage_pool").Filter("StorageSystemsId", id).OrderBy("Name").All(&pools); err != nil {
		flash.Notice("Ошибка чтения списка пулов: %v", err)
		flash.Store(&this.Controller)
	}

	// LUN по пулам - ДОБАВЛЕНО 2026-09-17 по запросу пользователя "нужно
	// показать LUN на этих пулах": models.StorageVolumes.PoolExternalKey
	// заполняется синхронизацией томов для Huawei/Unity (см. комментарии у
	// этого поля и у мест его заполнения в cmdb/UpdateHuaweyVolume.go и
	// cmdb/volumestorage.go) - для VNX/Lenovo пока не реализовано
	// (ограничение текущего кода синхронизации томов этих вендоров, не
	// СХД/API), там StorageVolumesByPool будет всегда пуст для каждого
	// пула этих вендоров.
	var volumes []models.StorageVolumes
	if _, err := o.QueryTable("storage_volumes").Filter("StorageSystemsId", id).Filter("Status", 2).OrderBy("StorageName").All(&volumes); err != nil {
		flash.Notice("Ошибка чтения списка томов: %v", err)
		flash.Store(&this.Controller)
	}
	volumesByPool := make(map[string][]models.StorageVolumes, len(pools))
	for _, v := range volumes {
		if v.PoolExternalKey == "" {
			continue
		}
		volumesByPool[v.PoolExternalKey] = append(volumesByPool[v.PoolExternalKey], v)
	}
	poolsWithVolumes := make([]PoolWithVolumes, 0, len(pools))
	for _, p := range pools {
		poolsWithVolumes = append(poolsWithVolumes, PoolWithVolumes{Pool: p, Volumes: volumesByPool[p.ExternalKey]})
	}

	this.Data["Pools"] = poolsWithVolumes
	this.Data["PoolCount"] = len(pools)
}
