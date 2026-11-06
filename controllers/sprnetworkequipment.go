package controllers

import (
	"fmt"
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// NetworkEquipmentFilter - фильтр списка /sprsrv/networkequipmentlist. Id и
// перечисления хранятся строками ("all"/"0"/...), чтобы отличать "не применён" от 0.
type NetworkEquipmentFilter struct {
	Name             string `form:"Name"`
	Vendor           string `form:"Vendor"`
	SerialNum        string `form:"SerialNum"`
	TypeId           string `form:"TypeId"`
	ItiliumStatus    string `form:"ItiliumStatus"`
	DatacenterSiteId string `form:"DatacenterSiteId"`
}

func defaultNetworkEquipmentFilter() *NetworkEquipmentFilter {
	return &NetworkEquipmentFilter{TypeId: "all", ItiliumStatus: "1", DatacenterSiteId: "0"} // по умолчанию - "В работе"
}

// NetworkEquipmentRow - строка списка сетевого оборудования, дополненная
// наименованием типа, ЦОД/шкафа и списком имён связанных сетей.
type NetworkEquipmentRow struct {
	models.SwitchSprRecords
	TypeName           string
	DatacenterSiteName string
	RackCabinetName    string
	Networks           string // имена связанных сетей через ", "
	UnitText           string // "15" или "15-16" - занимаемые юниты
}


// SprNetworkEquipment отображает список справочника сетевого оборудования
// (models.SwitchSprRecords, таблица switch_spr_records - бывший справочник
// коммутаторов, расширенный типом оборудования, статусом Итилиум, позицией в
// шкафу и учётными данными) со связями с сетями. Поддерживает экспорт в CSV/XLSX.
//
// Маршрут: /sprsrv/networkequipmentlist/:id
func (this *MainController) SprNetworkEquipment() {
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/networkequipmentlist")
	this.Data["ListId"] = this.Ctx.Input.Param(":id")

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "NetworkEquipmentFilter"
	filter, ok := m[keyFilter].(*NetworkEquipmentFilter)
	if !ok {
		filter = defaultNetworkEquipmentFilter()
		m[keyFilter] = filter
	}

	page, pageSize := 1, 50
	exportCSV, exportXLSX := false, false
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

	var types []models.NetworkEquipmentType
	o.QueryTable(new(models.NetworkEquipmentType)).OrderBy("Name").All(&types)
	this.Data["EquipmentTypes"] = types
	var sites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&sites)
	this.Data["DatacenterSites"] = sites

	qs := o.QueryTable("switch_spr_records")
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.Vendor != "" {
		qs = qs.Filter("Vendor__icontains", filter.Vendor)
	}
	if filter.SerialNum != "" {
		qs = qs.Filter("SerialNum__icontains", filter.SerialNum)
	}
	if filter.TypeId != "" && filter.TypeId != "all" {
		if v, err := strconv.ParseInt(filter.TypeId, 10, 64); err == nil {
			qs = qs.Filter("TypeId", v)
		}
	}
	if filter.ItiliumStatus != "" && filter.ItiliumStatus != "all" {
		if v, err := strconv.Atoi(filter.ItiliumStatus); err == nil {
			qs = qs.Filter("ItiliumStatus", v)
		}
	}
	if v, err := strconv.ParseInt(filter.DatacenterSiteId, 10, 64); err == nil && v > 0 {
		qs = qs.Filter("DatacenterSiteId", v)
	}

	total, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	records := make([]models.SwitchSprRecords, 0, pageSize)
	qsPage := qs.OrderBy("Name", "Id")
	if !exportCSV && !exportXLSX { // экспорт отдаёт весь найденный список, не страницу
		qsPage = qsPage.Limit(pageSize, offset)
	}
	if _, err := qsPage.All(&records); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	typeName := make(map[int64]string, len(types))
	for _, t := range types {
		typeName[t.Id] = t.Name
	}
	dcName := make(map[int64]string, len(sites))
	for _, s := range sites {
		dcName[s.Id] = s.Name
	}
	rcIds := make([]int64, 0, len(records))
	swIds := make([]int64, 0, len(records))
	for _, r := range records {
		if r.RackCabinetId != 0 {
			rcIds = append(rcIds, r.RackCabinetId)
		}
		swIds = append(swIds, r.Id)
	}
	rcName := make(map[int64]string, len(rcIds))
	if len(rcIds) > 0 {
		var rcs []models.RackCabinet
		if _, err := o.QueryTable("cmdb_rack_cabinet").Filter("Id__in", rcIds).All(&rcs); err == nil {
			for _, rc := range rcs {
				rcName[rc.Id] = rc.Name
			}
		}
	}
	networksBySwitch := loadNetworkNamesBySwitch(o, swIds)

	rows := make([]NetworkEquipmentRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, NetworkEquipmentRow{
			SwitchSprRecords:   r,
			TypeName:           typeName[r.TypeId],
			DatacenterSiteName: dcName[r.DatacenterSiteId],
			RackCabinetName:    rcName[r.RackCabinetId],
			Networks:           networksBySwitch[r.Id],
			UnitText:           networkEquipmentUnitText(r.StartUnitPos, r.SizeUnit),
		})
	}

	if exportCSV || exportXLSX {
		header, data := networkEquipmentExportTable(rows)
		this.exportTableFile(header, data, exportXLSX, "networkequipment", "Сетевое оборудование", flash)
		return
	}

	pageTo := int64(offset + pageSize)
	if pageTo > total {
		pageTo = total
	}
	pageFrom := int64(0)
	if total > 0 {
		pageFrom = int64(offset) + 1
	}
	this.Data["NetworkEquipmentList"] = rows
	this.Data["NetworkEquipmentCount"] = total
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = pageFrom
	this.Data["PageTo"] = pageTo
}

// networkEquipmentUnitText - занимаемые юниты: "15" или "15-16"; пусто, если позиция неизвестна.
func networkEquipmentUnitText(start, size int) string {
	if start <= 0 {
		return ""
	}
	if size > 1 {
		return fmt.Sprintf("%d-%d", start, start+size-1)
	}
	return strconv.Itoa(start)
}

// networkEquipmentPlacementText - "ЦОД / Шкаф, U<начало>[-<конец>]" для списка и выгрузки.
func networkEquipmentPlacementText(r NetworkEquipmentRow) string {
	s := r.DatacenterSiteName
	if r.RackCabinetName != "" {
		if s != "" {
			s += " / "
		}
		s += r.RackCabinetName
	}
	if r.UnitText != "" {
		s += ", U" + r.UnitText
	}
	return s
}

func networkEquipmentExportTable(rows []NetworkEquipmentRow) ([]string, [][]string) {
	header := []string{"Id", "Тип оборудования", "Статус Итилиум", "Наименование", "Производитель", "Модель",
		"Серийный номер", "IP управления", "MAC адрес", "Технический номер ИТ", "Инвентарный номер",
		"Инв. номер (бух.)", "Размещение", "Начальный юнит", "Размер, юнитов", "Принадлежность к прибору",
		"Модель замены", "Стоимость", "Стоимость со скидкой", "Автор", "Поставщик", "Документы закупки",
		"Дата документа закупки", "Сети", "Описание"}
	data := make([][]string, 0, len(rows))
	for _, r := range rows {
		data = append(data, []string{
			strconv.FormatInt(r.Id, 10), r.TypeName, models.ItiliumStatusName(r.ItiliumStatus), r.Name,
			r.Vendor, r.Model, r.SerialNum, r.ManagementIp, r.MacAddress, r.TechnicalNumberIT, r.InventoryNum,
			r.InventoryNumBuh, networkEquipmentPlacementText(r), strconv.Itoa(r.StartUnitPos), strconv.Itoa(r.SizeUnit),
			r.NetworkRole, r.ReplacementModel, strconv.FormatFloat(r.Cost, 'f', 2, 64),
			strconv.FormatFloat(r.CostDiscount, 'f', 2, 64), r.Owner, r.Counterparty, r.ContractDoc,
			formatStorageSystemsDate(r.DateContractDoc), r.Networks, r.Annotation,
		})
	}
	return header, data
}

// loadNetworkNamesBySwitch собирает для каждого устройства из switchIds
// строку с именами связанных сетей (через SwitchNetworkLink), разделённых
// ", " - используется и списком, и карточкой редактирования.
func loadNetworkNamesBySwitch(o orm.QueryExecutor, switchIds []int64) map[int64]string {
	result := make(map[int64]string, len(switchIds))
	if len(switchIds) == 0 {
		return result
	}
	var links []models.SwitchNetworkLink
	if _, err := o.QueryTable("switch_network_link").Filter("SwitchId__in", switchIds).All(&links); err != nil {
		return result
	}
	netIds := make([]int64, 0, len(links))
	netIdsBySwitch := make(map[int64][]int64, len(switchIds))
	for _, l := range links {
		netIds = append(netIds, l.NetworkId)
		netIdsBySwitch[l.SwitchId] = append(netIdsBySwitch[l.SwitchId], l.NetworkId)
	}
	if len(netIds) == 0 {
		return result
	}
	netNameById := make(map[int64]string, len(netIds))
	var nets []models.NetworkSprRecords
	if _, err := o.QueryTable("network_spr_records").Filter("Id__in", netIds).All(&nets); err == nil {
		for _, n := range nets {
			netNameById[n.Id] = n.Name
		}
	}
	for swId, ids := range netIdsBySwitch {
		names := make([]string, 0, len(ids))
		for _, id := range ids {
			if name, ok := netNameById[id]; ok && name != "" {
				names = append(names, name)
			}
		}
		result[swId] = strings.Join(names, ", ")
	}
	return result
}

// SprNetworkEquipmentEdit отображает форму просмотра/редактирования одной
// записи справочника сетевого оборудования вместе со списком связанных сетей
// (SwitchNetworkLink, чекбоксы; связи пересобираются по принципу mark-and-sweep)
// и сохраняет изменения по POST. Id=0 - добавление новой записи.
//
// Маршрут: /sprsrv/networkequipmentedit/:id
func (this *MainController) SprNetworkEquipmentEdit() {
	sprNetworkEquipmentEditPage(this, "sprsrv/networkequipmentedit", this.Ctx.Input.Param(":id"))
}

// SprNetworkEquipmentEditTab - та же карточка сетевого оборудования, что
// SprNetworkEquipmentEdit, но с формой на закладках
// (шаблон networkequipmentedit_tab.tpl).
//
// Маршрут: /sprsrv/networkequipmentedit-tab/:id([0-9]+)
func (this *MainController) SprNetworkEquipmentEditTab() {
	sprNetworkEquipmentEditPage(this, "sprsrv/networkequipmentedit_tab", this.Ctx.Input.Param(":id"))
}

// sprNetworkEquipmentEditPage - общая логика GET/POST формы SwitchSprRecords.
// tpl - sprsrv/networkequipmentedit или sprsrv/networkequipmentedit_tab.
func sprNetworkEquipmentEditPage(this *MainController, tpl, idStr string) {
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent(tpl)
	this.Data["Id"] = idStr
	this.Data["NetEqEditBasePath"] = "/sprsrv/networkequipmentedit"
	if strings.Contains(tpl, "networkequipmentedit_tab") {
		this.Data["NetEqEditBasePath"] = "/sprsrv/networkequipmentedit-tab"
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
	var rec models.SwitchSprRecords
	if !isNew {
		if err := o.QueryTable("switch_spr_records").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.fillNetworkEquipmentEditData(o, rec, isNew)
			return
		}
		var parseErrors []string
		parseInt := func(label, formName string) int64 {
			s := strings.TrimSpace(this.GetString(formName))
			if s == "" {
				return 0
			}
			v, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Sprintf("%s: %q - не целое число", label, s))
				return 0
			}
			return v
		}
		parseFloat := func(label, formName string) float64 {
			s := strings.TrimSpace(strings.ReplaceAll(this.GetString(formName), ",", "."))
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

		rec.Name = this.GetString("Name")
		rec.TypeId = parseInt("Тип оборудования", "TypeId")
		rec.ItiliumStatus = int(parseInt("Статус Итилиум", "ItiliumStatus"))
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.SerialNum = this.GetString("SerialNum")
		rec.ManagementIp = this.GetString("ManagementIp")
		rec.MacAddress = this.GetString("MacAddress")
		rec.DatacenterSiteId = parseInt("ЦОД", "DatacenterSiteId")
		rec.RackCabinetId = parseInt("Шкаф", "RackCabinetId")
		rec.StartUnitPos = int(parseInt("Начальный юнит", "StartUnitPos"))
		rec.SizeUnit = int(parseInt("Размер, юнитов", "SizeUnit"))
		rec.TechnicalNumberIT = this.GetString("TechnicalNumberIT")
		rec.InventoryNum = this.GetString("InventoryNum")
		rec.InventoryNumBuh = this.GetString("InventoryNumBuh")
		rec.Owner = this.GetString("Owner")
		rec.Counterparty = this.GetString("Counterparty")
		rec.ContractDoc = this.GetString("ContractDoc")
		rec.DateContractDoc = parseStorageSystemsDate(this.GetString("DateContractDoc"))
		rec.NetworkRole = this.GetString("NetworkRole")
		rec.ReplacementModel = this.GetString("ReplacementModel")
		rec.Cost = parseFloat("Стоимость", "Cost")
		rec.CostDiscount = parseFloat("Стоимость со скидкой", "CostDiscount")
		rec.Annotation = this.GetString("Annotation")

		if len(parseErrors) > 0 {
			flash.Error("Не удалось сохранить - проверьте числовые поля: %s", strings.Join(parseErrors, "; "))
			flash.Store(&this.Controller)
			this.fillNetworkEquipmentEditData(o, rec, isNew)
			return
		}
		var currentUserId int64
		if uid, ok := this.Data["UserId"].(int64); ok {
			currentUserId = uid
		}
		rec.Id_User = currentUserId
		rec.Time_Mfy = time.Now().Local()

		if isNew {
			rec.SourceUpdateType = models.SourceUpdateManual
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.fillNetworkEquipmentEditData(o, rec, isNew)
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.saveSwitchNetworkLinks(o, rec.Id)
			basePath, _ := this.Data["NetEqEditBasePath"].(string)
			if basePath == "" {
				basePath = "/sprsrv/networkequipmentedit"
			}
			this.Redirect(fmt.Sprintf("%s/%d", basePath, rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.fillNetworkEquipmentEditData(o, rec, isNew)
			return
		}
		this.saveSwitchNetworkLinks(o, rec.Id)
		RegisterRequest(this, LogTypeEditReference)
		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	this.fillNetworkEquipmentEditData(o, rec, isNew)
}

func (this *MainController) fillNetworkEquipmentEditData(o orm.Ormer, rec models.SwitchSprRecords, isNew bool) {
	var types []models.NetworkEquipmentType
	o.QueryTable(new(models.NetworkEquipmentType)).OrderBy("Name").All(&types)
	this.Data["EquipmentTypes"] = types

	var sites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&sites)
	this.Data["DatacenterSites"] = sites

	var racks []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&racks)
	this.Data["RackCabinets"] = racks

	var networks []models.NetworkSprRecords
	o.QueryTable("network_spr_records").OrderBy("Name").All(&networks)
	this.Data["Networks"] = networks

	linkedIds := map[int64]bool{}
	if !isNew {
		var links []models.SwitchNetworkLink
		if _, err := o.QueryTable("switch_network_link").Filter("SwitchId", rec.Id).All(&links); err == nil {
			for _, l := range links {
				linkedIds[l.NetworkId] = true
			}
		}
	}
	this.Data["LinkedNetworkIds"] = linkedIds
	this.Data["SW"] = rec
	this.Data["ItiliumStatuses"] = models.ItiliumStatusList
	this.Data["DateContractDocStr"] = formatStorageSystemsDate(rec.DateContractDoc)
	this.Data["TimeActiveStateStr"] = formatStorageSystemsDate(rec.TimeActiveState)
}

// saveSwitchNetworkLinks пересобирает связи оборудование-сеть (mark-and-sweep):
// удаляет все текущие SwitchNetworkLink для данного switchId и вставляет
// заново по значениям чекбоксов "NetworkIds" из формы редактирования.
func (this *MainController) saveSwitchNetworkLinks(o orm.Ormer, switchId int64) {
	if _, err := o.QueryTable("switch_network_link").Filter("SwitchId", switchId).Delete(); err != nil {
		return
	}
	for _, s := range this.GetStrings("NetworkIds") {
		netId, err := strconv.ParseInt(s, 10, 64)
		if err != nil || netId <= 0 {
			continue
		}
		link := models.SwitchNetworkLink{SwitchId: switchId, NetworkId: netId}
		o.Insert(&link)
	}
}

// redirectSwitchList / redirectSwitchEdit - старые адреса справочника
// коммутаторов (/sprsrv/switchlist, /sprsrv/switchedit) перенаправляются на
// справочник сетевого оборудования, чтобы не ломались закладки.
func (this *MainController) RedirectSwitchList() {
	this.Redirect("/sprsrv/networkequipmentlist/0", 301)
}

func (this *MainController) RedirectSwitchEdit() {
	this.Redirect("/sprsrv/networkequipmentedit-tab/"+this.Ctx.Input.Param(":id"), 301)
}
