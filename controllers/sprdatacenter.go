package controllers

import (
	"fmt"
	"loadmon/models"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// DatacenterSiteFilter - фильтр списка /sprsrv/datacentersitelist. Простой
// текстовый фильтр по Name/AltName, без экспорта в CSV/XLSX (справочник
// заведомо небольшой) - структура/приём пагинации идентичны
// BladeSystemsFilter в controllers/sprbladesystems.go.
type DatacenterSiteFilter struct {
	Name    string `form:"Name"`
	AltName string `form:"AltName"`
}

func defaultDatacenterSiteFilter() *DatacenterSiteFilter {
	return &DatacenterSiteFilter{}
}

// SprDatacenterSite отображает список записей справочника ЦОД
// (models.DatacenterSite, таблица cmdb_datacenter_site) - новый справочник,
// заведённый по запросу пользователя "структуры описывающие ЦОД по аналогии
// с Datacenter из oneview2" (см. models.DatacenterSite).
//
// Маршрут: /sprsrv/datacentersitelist/:id (:id зарезервирован под будущее
// использование, как и в аналогичных справочниках).
func (this *MainController) SprDatacenterSite() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/datacentersitelist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "DatacenterSiteFilter"
	filter, ok := m[keyFilter].(*DatacenterSiteFilter)
	if !ok {
		filter = defaultDatacenterSiteFilter()
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
	qs := o.QueryTable("cmdb_datacenter_site")
	if filter.Name != "" {
		qs = qs.Filter("Name__icontains", filter.Name)
	}
	if filter.AltName != "" {
		qs = qs.Filter("AltName__icontains", filter.AltName)
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

	records := make([]models.DatacenterSite, 0, pageSize)
	if _, err := qs.OrderBy("Name").Limit(pageSize, offset).All(&records); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	this.Data["DatacenterSiteList"] = records
	this.Data["DatacenterSiteCount"] = totalRecords
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

// SprDatacenterSiteEdit отображает форму просмотра/редактирования одной
// записи models.DatacenterSite по Id и сохраняет изменения по отправке формы
// (POST). Id=0 - режим добавления новой записи.
//
// Маршрут: /sprsrv/datacentersiteedit/:id
func (this *MainController) SprDatacenterSiteEdit() {

	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/datacentersiteedit")
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

	var rec models.DatacenterSite
	if !isNew {
		if err := o.QueryTable("cmdb_datacenter_site").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["DC"] = rec
			return
		}
		rec.Name = this.GetString("Name")
		rec.AltName = this.GetString("AltName")
		rec.Width, _ = strconv.ParseInt(this.GetString("Width"), 10, 64)
		rec.Height, _ = strconv.ParseInt(this.GetString("Height"), 10, 64)
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
				this.Data["DC"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.Redirect(fmt.Sprintf("/sprsrv/datacentersiteedit/%d", rec.Id), 302)
			return
		}

		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["DC"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference)

		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	this.Data["DC"] = rec
}
