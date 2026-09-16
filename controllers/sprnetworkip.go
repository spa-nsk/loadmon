package controllers

import (
	"loadmon/models"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// IPAddressFilter - фильтр списка /sprsrv/ipaddresslist. Приём идентичен
// NetworkFilter (controllers/sprnetwork.go).
type IPAddressFilter struct {
	IPAddress  string `form:"IPAddress"`
	MacAddress string `form:"MacAddress"`
	ServerName string `form:"ServerName"`
	NetworkId  string `form:"NetworkId"`
}

func defaultIPAddressFilter() *IPAddressFilter {
	return &IPAddressFilter{}
}

// SprIPAddress отображает read-only список записей справочника IP-адресов
// (models.IPAddressSprRecords, таблица ip_address_spr_records) - заведён по
// запросу пользователя "на основе данных из справочника серверов по полю
// Network построй справочник сетей и отдельно создай справочник IP адресов
// и свяжи с сетями" (2026-09-16). В ОТЛИЧИЕ от /sprsrv/networklist эта
// таблица целиком производная (пересобирается автосинхронизацией,
// см. cmdb/syncnetworkip.go), поэтому у неё нет формы добавления/
// редактирования - только просмотр со ссылкой на исходную карточку сервера.
//
// Маршрут: /sprsrv/ipaddresslist/:id
func (this *MainController) SprIPAddress() {

	Id := this.Ctx.Input.Param(":id")

	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/ipaddresslist")
	this.Data["ListId"] = Id

	m := sess.(models.LoadFilter).Map
	flash := beego.NewFlash()

	const keyFilter = "IPAddressFilter"
	filter, ok := m[keyFilter].(*IPAddressFilter)
	if !ok {
		filter = defaultIPAddressFilter()
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

	var networks []models.NetworkSprRecords
	o.QueryTable(new(models.NetworkSprRecords)).OrderBy("Name").All(&networks, "Id", "Name")
	this.Data["NetworkList"] = networks

	qs := o.QueryTable(new(models.IPAddressSprRecords))
	if filter.IPAddress != "" {
		qs = qs.Filter("IPAddress__icontains", filter.IPAddress)
	}
	if filter.MacAddress != "" {
		qs = qs.Filter("MacAddress__icontains", filter.MacAddress)
	}
	if filter.ServerName != "" {
		qs = qs.Filter("ServerName__icontains", filter.ServerName)
	}
	if filter.NetworkId != "" && filter.NetworkId != "0" {
		if v, err := strconv.ParseInt(filter.NetworkId, 10, 64); err == nil && v > 0 {
			qs = qs.Filter("NetworkId", v)
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

	records := make([]models.IPAddressSprRecords, 0, pageSize)
	if _, err := qs.OrderBy("IPAddress").Limit(pageSize, offset).All(&records); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	// Имя сети для отображения - резолвится здесь одним запросом по всем Id
	// текущей страницы, чтобы не плодить JOIN'ы/N+1 запросы в шаблоне.
	netIds := make([]int64, 0, len(records))
	seenNet := map[int64]bool{}
	for _, r := range records {
		if r.NetworkId > 0 && !seenNet[r.NetworkId] {
			seenNet[r.NetworkId] = true
			netIds = append(netIds, r.NetworkId)
		}
	}
	netNameById := map[int64]string{}
	if len(netIds) > 0 {
		var nets []models.NetworkSprRecords
		o.QueryTable(new(models.NetworkSprRecords)).Filter("Id__in", netIds).All(&nets, "Id", "Name")
		for _, n := range nets {
			netNameById[n.Id] = n.Name
		}
	}
	this.Data["NetworkNameById"] = netNameById

	this.Data["IPList"] = records
	this.Data["IPCount"] = totalRecords
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
