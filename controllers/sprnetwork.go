package controllers

import (
	"fmt"
	"loadmon/models"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// NetworkFilter - фильтр списка /sprsrv/networklist. Приём идентичен
// DatacenterSiteFilter (controllers/sprdatacenter.go).
type NetworkFilter struct {
	Name   string `form:"Name"`
	Vlan   string `form:"Vlan"`
	Subnet string `form:"Subnet"`
}

func defaultNetworkFilter() *NetworkFilter {
	return &NetworkFilter{}
}

// SprNetwork отображает список записей справочника сетей
// (models.NetworkSprRecords, таблица network_spr_records) - новый справочник
// сетей для серверов и других компонент инфраструктуры, заведённый по запросу
// пользователя.
//
// Маршрут: /sprsrv/networklist/:id
func (this *MainController) SprNetwork() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/networklist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "NetworkFilter"
	filter, ok := m[keyFilter].(*NetworkFilter)
	if !ok {
		filter = defaultNetworkFilter()
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
	qs := o.QueryTable("network_spr_records")
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.Subnet != "" {
		qs = qs.Filter("Subnet__icontains", filter.Subnet)
	}
	if filter.Vlan != "" {
		if v, err := strconv.ParseInt(filter.Vlan, 10, 64); err == nil {
			qs = qs.Filter("Vlan", v)
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

	records := make([]models.NetworkSprRecords, 0, pageSize)
	if _, err := qs.OrderBy("Name").Limit(pageSize, offset).All(&records); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	this.Data["NetworkList"] = records
	this.Data["NetworkCount"] = totalRecords
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

// SprNetworkEdit отображает форму просмотра/редактирования одной записи
// models.NetworkSprRecords по Id и сохраняет изменения по отправке формы
// (POST). Id=0 - режим добавления новой записи.
//
// Маршрут: /sprsrv/networkedit/:id
func (this *MainController) SprNetworkEdit() {

	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/networkedit")
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

	var rec models.NetworkSprRecords
	if !isNew {
		if err := o.QueryTable("network_spr_records").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["NW"] = rec
			return
		}
		rec.Name = this.GetString("Name")
		rec.Vlan, _ = strconv.ParseInt(this.GetString("Vlan"), 10, 64)
		rec.Subnet = this.GetString("Subnet")
		rec.Gateway = this.GetString("Gateway")
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
				this.Data["NW"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.Redirect(fmt.Sprintf("/sprsrv/networkedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["NW"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference)

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	this.Data["NW"] = rec
}
