package controllers

import (
	"fmt"
	"loadmon/models"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// SwitchFilter - фильтр списка /sprsrv/switchlist. Приём идентичен
// RackCabinetFilter (controllers/sprrackcabinet.go).
type SwitchFilter struct {
	Name             string `form:"Name"`
	Vendor           string `form:"Vendor"`
	DatacenterSiteId string `form:"DatacenterSiteId"`
}

func defaultSwitchFilter() *SwitchFilter {
	return &SwitchFilter{}
}

// SwitchRow - строка списка коммутаторов, дополненная наименованием ЦОД/шкафа
// и списком имён связанных сетей (через SwitchNetworkLink) для отображения.
type SwitchRow struct {
	models.SwitchSprRecords
	DatacenterSiteName string
	RackCabinetName    string
	Networks           string // имена связанных сетей через ", "
}

// SprSwitch отображает список записей справочника коммутаторов
// (models.SwitchSprRecords, таблица switch_spr_records) со связями с сетями
// (models.SwitchNetworkLink/NetworkSprRecords) - новый справочник, заведённый
// по запросу пользователя "Создай справочник сетевых коммутаторов со связями
// с сетями".
//
// Маршрут: /sprsrv/switchlist/:id
func (this *MainController) SprSwitch() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/switchlist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "SwitchFilter"
	filter, ok := m[keyFilter].(*SwitchFilter)
	if !ok {
		filter = defaultSwitchFilter()
		m[keyFilter] = filter
	}

	page := 1
	pageSize := 50

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
	}

	this.Data[keyFilter] = filter

	o := orm.NewOrm()
	qs := o.QueryTable("switch_spr_records")
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.Vendor != "" {
		qs = qs.Filter("Vendor__icontains", filter.Vendor)
	}
	if filter.DatacenterSiteId != "" {
		if dcid, err := strconv.ParseInt(filter.DatacenterSiteId, 10, 64); err == nil {
			qs = qs.Filter("DatacenterSiteId", dcid)
		}
	}

	totalRecords, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	totalPages := int((totalRecords + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	records := make([]models.SwitchSprRecords, 0, pageSize)
	if _, err := qs.OrderBy("Name").Limit(pageSize, offset).All(&records); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	dcIds := make([]int64, 0, len(records))
	rcIds := make([]int64, 0, len(records))
	swIds := make([]int64, 0, len(records))
	for _, r := range records {
		if r.DatacenterSiteId != 0 {
			dcIds = append(dcIds, r.DatacenterSiteId)
		}
		if r.RackCabinetId != 0 {
			rcIds = append(rcIds, r.RackCabinetId)
		}
		swIds = append(swIds, r.Id)
	}
	dcNameById := make(map[int64]string, len(dcIds))
	if len(dcIds) > 0 {
		var dcs []models.DatacenterSite
		if _, err := o.QueryTable("cmdb_datacenter_site").Filter("Id__in", dcIds).All(&dcs); err == nil {
			for _, dc := range dcs {
				dcNameById[dc.Id] = dc.Name
			}
		}
	}
	rcNameById := make(map[int64]string, len(rcIds))
	if len(rcIds) > 0 {
		var rcs []models.RackCabinet
		if _, err := o.QueryTable("cmdb_rack_cabinet").Filter("Id__in", rcIds).All(&rcs); err == nil {
			for _, rc := range rcs {
				rcNameById[rc.Id] = rc.Name
			}
		}
	}
	networksBySwitch := loadNetworkNamesBySwitch(o, swIds)

	rows := make([]SwitchRow, 0, len(records))
	for _, r := range records {
		rows = append(rows, SwitchRow{
			SwitchSprRecords:   r,
			DatacenterSiteName: dcNameById[r.DatacenterSiteId],
			RackCabinetName:    rcNameById[r.RackCabinetId],
			Networks:           networksBySwitch[r.Id],
		})
	}

	this.Data["SwitchList"] = rows
	this.Data["SwitchCount"] = totalRecords
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

// loadNetworkNamesBySwitch собирает для каждого коммутатора из switchIds
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
		s := ""
		for i, n := range names {
			if i > 0 {
				s += ", "
			}
			s += n
		}
		result[swId] = s
	}
	return result
}

// SprSwitchEdit отображает форму просмотра/редактирования одной записи
// models.SwitchSprRecords по Id вместе со списком связанных сетей (через
// SwitchNetworkLink, отмечаемых чекбоксами) и сохраняет изменения по
// отправке формы (POST), включая пересборку связей "коммутатор-сеть" по
// принципу mark-and-sweep (удалить все текущие связи и вставить заново по
// отмеченным чекбоксам) - тот же приём, что и для синхронизации компонент
// оборудования (см. cmdb/syncserverhardwarecomponents.go). Id=0 - режим
// добавления новой записи.
//
// Маршрут: /sprsrv/switchedit/:id
func (this *MainController) SprSwitchEdit() {

	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/switchedit")
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
			this.Data["SW"] = rec
			return
		}
		rec.Name = this.GetString("Name")
		rec.Vendor = this.GetString("Vendor")
		rec.Model = this.GetString("Model")
		rec.SerialNum = this.GetString("SerialNum")
		rec.ManagementIp = this.GetString("ManagementIp")
		rec.DatacenterSiteId, _ = strconv.ParseInt(this.GetString("DatacenterSiteId"), 10, 64)
		rec.RackCabinetId, _ = strconv.ParseInt(this.GetString("RackCabinetId"), 10, 64)
		rec.Annotation = this.GetString("Annotation")
		var currentUserId int64
		if uid, ok := this.Data["UserId"].(int64); ok {
			currentUserId = uid
		}
		rec.Id_User = currentUserId
		rec.Time_Mfy = time.Now().Local()

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["SW"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.saveSwitchNetworkLinks(o, rec.Id)
			this.Redirect(fmt.Sprintf("/sprsrv/switchedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["SW"] = rec
			return
		}
		this.saveSwitchNetworkLinks(o, rec.Id)
		RegisterRequest(this, LogTypeEditReference)

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

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
}

// saveSwitchNetworkLinks пересобирает связи коммутатор-сеть (mark-and-sweep):
// удаляет все текущие SwitchNetworkLink для данного switchId и вставляет
// заново по значениям чекбоксов "NetworkIds" из формы редактирования.
func (this *MainController) saveSwitchNetworkLinks(o orm.Ormer, switchId int64) {
	if _, err := o.QueryTable("switch_network_link").Filter("SwitchId", switchId).Delete(); err != nil {
		return
	}
	ids := this.GetStrings("NetworkIds")
	for _, s := range ids {
		netId, err := strconv.ParseInt(s, 10, 64)
		if err != nil || netId <= 0 {
			continue
		}
		link := models.SwitchNetworkLink{SwitchId: switchId, NetworkId: netId}
		o.Insert(&link)
	}
}
