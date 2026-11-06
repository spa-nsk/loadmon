package controllers

import (
	"fmt"
	"loadmon/models"
	"net/url"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// equipPlacementHit - найденное по номеру оборудование и его размещение.
type equipPlacementHit struct {
	Kind            string // server | chassis | storage | switch | tape
	KindLabel       string
	Id              int64
	Name            string
	MatchField      string
	MatchValue      string
	SiteId          int64
	SiteName        string
	RackId          int64
	RackName        string
	ItiliumStatus   int
	StatusName      string
	NoVisualization bool   // списан / в пути - только карточка справочника
	ViewURL         string // план площадки (если есть размещение и можно визуализировать)
	CardURL         string // карточка оборудования в справочнике
}

// SprEquipmentPlacement - поиск размещения оборудования по строке
// (технический номер ИТ, инвентарный, бухгалтерский инвентарный или серийный номер)
// среди серверов, корзин, СХД, сетевого оборудования и ленточных библиотек.
// Один результат - переход на просмотр площадки с выделением шкафа и устройства;
// несколько - список для выбора.
//
// Маршрут: /sprsrv/equipmentplacement  (GET)
func (this *MainController) SprEquipmentPlacement() {
	RegisterRequest(this, LogTypeViewReference)
	this.activeContent("sprsrv/equipmentplacement")

	q := strings.TrimSpace(this.GetString("q"))
	this.Data["Query"] = q
	this.Data["Results"] = []equipPlacementHit{}
	this.Data["ResultCount"] = 0

	flash := beego.NewFlash()
	defer func() {
		if len(flash.Data) > 0 {
			flash.Store(&this.Controller)
		}
	}()

	if q == "" {
		return
	}
	if len(q) < 2 {
		flash.Error("Введите не менее 2 символов для поиска")
		return
	}

	o := orm.NewOrm()
	hits := searchEquipmentPlacement(o, q)
	this.Data["Results"] = hits
	this.Data["ResultCount"] = len(hits)

	if len(hits) == 0 {
		flash.Notice("По запросу «" + q + "» оборудование не найдено")
		return
	}
	if len(hits) == 1 && this.GetString("list") != "1" {
		h := hits[0]
		if h.NoVisualization {
			if h.CardURL == "" {
				flash.Error("Найдено оборудование со статусом «" + h.StatusName + "» без карточки")
				return
			}
			this.Redirect(h.CardURL, 302)
			return
		}
		if h.ViewURL == "" {
			if h.CardURL != "" {
				this.Redirect(h.CardURL, 302)
				return
			}
			flash.Error("Найдено оборудование без размещения в шкафу/площадке")
			return
		}
		this.Redirect(h.ViewURL, 302)
		return
	}
}

func searchEquipmentPlacement(o orm.Ormer, q string) []equipPlacementHit {
	hits := make([]equipPlacementHit, 0, 16)

	// --- серверы (ServerHardware) ---
	var hws []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).SetCond(equipNumCond(q)).Limit(50).
		All(&hws, "Id", "Vendor", "Model", "ShortModel", "SerialNum", "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT",
			"RackCabinetId", "FormFactor", "BladeSystemId", "ItiliumStatus"); err == nil {
		for _, hw := range hws {
			field, val, ok := matchEquipNumbers(q, hw.SerialNum, hw.InventoryNum, hw.InventoryNumBuh, hw.TechnicalNumberIT)
			if !ok {
				continue
			}
			name := firstNonEmpty(strings.TrimSpace(hw.Vendor+" "+hw.Model), hw.ShortModel, hw.SerialNum, fmt.Sprintf("server-%d", hw.Id))
			siteId, siteName, rackId, rackName := resolveHWPlacement(o, &hw)
			hits = append(hits, makeEquipHit("server", "Сервер", hw.Id, name, field, val, siteId, siteName, rackId, rackName, hw.ItiliumStatus))
		}
	}

	// --- блейд-корзины ---
	var chassis []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).SetCond(equipNumCond(q)).Limit(50).
		All(&chassis, "Id", "Name", "SerialNum", "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "RackCabinetId", "ItiliumStatus"); err == nil {
		for _, ch := range chassis {
			field, val, ok := matchEquipNumbers(q, ch.SerialNum, ch.InventoryNum, ch.InventoryNumBuh, ch.TechnicalNumberIT)
			if !ok {
				continue
			}
			name := firstNonEmpty(ch.Name, ch.SerialNum, fmt.Sprintf("chassis-%d", ch.Id))
			siteId, siteName, rackId, rackName := resolveRackPlacement(o, ch.RackCabinetId)
			hits = append(hits, makeEquipHit("chassis", "Блейд-корзина", ch.Id, name, field, val, siteId, siteName, rackId, rackName, ch.ItiliumStatus))
		}
	}

	// --- СХД ---
	var storages []models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).SetCond(equipNumCond(q)).Limit(50).
		All(&storages, "Id", "ShortName", "SerialNum", "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "ItiliumStatus"); err == nil {
		for _, st := range storages {
			field, val, ok := matchEquipNumbers(q, st.SerialNum, st.InventoryNum, st.InventoryNumBuh, st.TechnicalNumberIT)
			if !ok {
				continue
			}
			name := firstNonEmpty(st.ShortName, st.SerialNum, fmt.Sprintf("storage-%d", st.Id))
			siteId, siteName, rackId, rackName := resolveStoragePlacement(o, st.Id)
			hits = append(hits, makeEquipHit("storage", "СХД", st.Id, name, field, val, siteId, siteName, rackId, rackName, st.ItiliumStatus))
		}
	}

	// --- сетевое оборудование ---
	var nets []models.SwitchSprRecords
	if _, err := o.QueryTable(new(models.SwitchSprRecords)).SetCond(equipNumCond(q)).Limit(50).
		All(&nets, "Id", "Name", "SerialNum", "InventoryNum", "InventoryNumBuh", "TechnicalNumberIT", "RackCabinetId", "ItiliumStatus"); err == nil {
		for _, sw := range nets {
			field, val, ok := matchEquipNumbers(q, sw.SerialNum, sw.InventoryNum, sw.InventoryNumBuh, sw.TechnicalNumberIT)
			if !ok {
				continue
			}
			name := firstNonEmpty(sw.Name, sw.SerialNum, fmt.Sprintf("switch-%d", sw.Id))
			siteId, siteName, rackId, rackName := resolveRackPlacement(o, sw.RackCabinetId)
			hits = append(hits, makeEquipHit("switch", "Сетевое оборудование", sw.Id, name, field, val, siteId, siteName, rackId, rackName, sw.ItiliumStatus))
		}
	}

	// --- ленточные библиотеки ---
	var tapes []models.TapeLibrary
	if _, err := o.QueryTable(new(models.TapeLibrary)).SetCond(equipNumCond(q)).Limit(50).
		All(&tapes, "Id", "Name", "TechnicalNumberIT", "SerialNum", "InventoryNum", "InventoryNumBuh", "RackCabinetId", "ItiliumStatus"); err == nil {
		for _, tl := range tapes {
			field, val, ok := matchEquipNumbers(q, tl.SerialNum, tl.InventoryNum, tl.InventoryNumBuh, tl.TechnicalNumberIT)
			if !ok {
				continue
			}
			name := firstNonEmpty(tl.Name, tl.TechnicalNumberIT, tl.SerialNum, fmt.Sprintf("tape-%d", tl.Id))
			siteId, siteName, rackId, rackName := resolveRackPlacement(o, tl.RackCabinetId)
			hits = append(hits, makeEquipHit("tape", "Ленточная библиотека", tl.Id, name, field, val, siteId, siteName, rackId, rackName, tl.ItiliumStatus))
		}
	}

	return hits
}

func equipNumCond(q string) *orm.Condition {
	return orm.NewCondition().
		Or("TechnicalNumberIT__iexact", q).
		Or("InventoryNum__iexact", q).
		Or("InventoryNumBuh__iexact", q).
		Or("SerialNum__iexact", q)
}

func matchEquipNumbers(q, serial, inv, invBuh, tech string) (field, value string, ok bool) {
	q = strings.TrimSpace(q)
	eq := func(a string) bool { return strings.EqualFold(strings.TrimSpace(a), q) }
	switch {
	case eq(tech):
		return "Технический номер ИТ", strings.TrimSpace(tech), true
	case eq(inv):
		return "Инвентарный номер", strings.TrimSpace(inv), true
	case eq(invBuh):
		return "Бухгалтерский инвентарный номер", strings.TrimSpace(invBuh), true
	case eq(serial):
		return "Серийный номер", strings.TrimSpace(serial), true
	}
	return "", "", false
}

func resolveHWPlacement(o orm.Ormer, hw *models.ServerHardware) (siteId int64, siteName string, rackId int64, rackName string) {
	rackId = hw.RackCabinetId
	if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
		var ch models.BladeSystems
		if err := o.QueryTable(new(models.BladeSystems)).Filter("Id", hw.BladeSystemId).
			One(&ch, "Id", "RackCabinetId"); err == nil && ch.RackCabinetId > 0 {
			rackId = ch.RackCabinetId
		}
	}
	return resolveRackPlacement(o, rackId)
}

func resolveStoragePlacement(o orm.Ormer, storageId int64) (siteId int64, siteName string, rackId int64, rackName string) {
	var pl models.StorageSystemPlacement
	if err := o.QueryTable(new(models.StorageSystemPlacement)).
		Filter("StorageSystemsId", storageId).Filter("RackCabinetId__gt", 0).
		OrderBy("Id").One(&pl, "RackCabinetId"); err != nil {
		return 0, "", 0, ""
	}
	return resolveRackPlacement(o, pl.RackCabinetId)
}

func resolveRackPlacement(o orm.Ormer, rackId int64) (siteId int64, siteName string, outRackId int64, rackName string) {
	if rackId <= 0 {
		return 0, "", 0, ""
	}
	var rc models.RackCabinet
	if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", rackId).
		One(&rc, "Id", "Name", "DatacenterSiteId"); err != nil {
		return 0, "", 0, ""
	}
	siteName = ""
	if rc.DatacenterSiteId > 0 {
		var site models.DatacenterSite
		if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", rc.DatacenterSiteId).
			One(&site, "Id", "Name"); err == nil {
			siteName = site.Name
		}
	}
	return rc.DatacenterSiteId, siteName, rc.Id, rc.Name
}

func equipCardURL(kind string, id int64) string {
	switch kind {
	case "server":
		return fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", id)
	case "chassis":
		return fmt.Sprintf("/sprsrv/bladesystemsedit/%d", id)
	case "storage":
		return fmt.Sprintf("/sprsrv/storagesystemsedit-tab/%d", id)
	case "switch":
		return fmt.Sprintf("/sprsrv/networkequipmentedit/%d", id)
	case "tape":
		return fmt.Sprintf("/sprsrv/tapelibrariesedit/%d", id)
	}
	return ""
}

func makeEquipHit(kind, kindLabel string, id int64, name, field, val string, siteId int64, siteName string, rackId int64, rackName string, itiliumStatus int) equipPlacementHit {
	h := equipPlacementHit{
		Kind: kind, KindLabel: kindLabel, Id: id, Name: name,
		MatchField: field, MatchValue: val,
		SiteId: siteId, SiteName: siteName, RackId: rackId, RackName: rackName,
		ItiliumStatus: itiliumStatus, StatusName: models.ItiliumStatusName(itiliumStatus),
		CardURL: equipCardURL(kind, id),
	}
	h.NoVisualization = models.ItiliumStatusHiddenFromPlans(itiliumStatus)
	if !h.NoVisualization && siteId > 0 && rackId > 0 {
		h.ViewURL = fmt.Sprintf("/sprsrv/datacenterequipview/%d?rack=%d&kind=%s&eq=%d",
			siteId, rackId, url.QueryEscape(kind), id)
	}
	return h
}

// openEquipPlacementById - вспомогательный редирект по kind+id (для ссылок из списка).
// Маршрут: /sprsrv/equipmentplacement/go/:kind/:id
func (this *MainController) SprEquipmentPlacementGo() {
	RegisterRequest(this, LogTypeViewReference)
	flash := beego.NewFlash()
	kind := strings.TrimSpace(this.Ctx.Input.Param(":kind"))
	idStr := this.Ctx.Input.Param(":id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор оборудования")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/equipmentplacement", 302)
		return
	}
	o := orm.NewOrm()
	var siteId, rackId int64
	var itiliumStatus int
	switch kind {
	case "server":
		var hw models.ServerHardware
		if err := o.QueryTable(new(models.ServerHardware)).Filter("Id", id).
			One(&hw, "Id", "RackCabinetId", "FormFactor", "BladeSystemId", "ItiliumStatus"); err != nil {
			flash.Error("Сервер не найден")
			flash.Store(&this.Controller)
			this.Redirect("/sprsrv/equipmentplacement", 302)
			return
		}
		itiliumStatus = hw.ItiliumStatus
		siteId, _, rackId, _ = resolveHWPlacement(o, &hw)
	case "chassis":
		var ch models.BladeSystems
		if err := o.QueryTable(new(models.BladeSystems)).Filter("Id", id).
			One(&ch, "Id", "RackCabinetId", "ItiliumStatus"); err != nil {
			flash.Error("Корзина не найдена")
			flash.Store(&this.Controller)
			this.Redirect("/sprsrv/equipmentplacement", 302)
			return
		}
		itiliumStatus = ch.ItiliumStatus
		siteId, _, rackId, _ = resolveRackPlacement(o, ch.RackCabinetId)
	case "storage":
		var st models.StorageSystems
		if err := o.QueryTable(new(models.StorageSystems)).Filter("Id", id).
			One(&st, "Id", "ItiliumStatus"); err != nil {
			flash.Error("СХД не найдена")
			flash.Store(&this.Controller)
			this.Redirect("/sprsrv/equipmentplacement", 302)
			return
		}
		itiliumStatus = st.ItiliumStatus
		siteId, _, rackId, _ = resolveStoragePlacement(o, id)
	case "switch":
		var sw models.SwitchSprRecords
		if err := o.QueryTable(new(models.SwitchSprRecords)).Filter("Id", id).
			One(&sw, "Id", "RackCabinetId", "ItiliumStatus"); err != nil {
			flash.Error("Сетевое оборудование не найдено")
			flash.Store(&this.Controller)
			this.Redirect("/sprsrv/equipmentplacement", 302)
			return
		}
		itiliumStatus = sw.ItiliumStatus
		siteId, _, rackId, _ = resolveRackPlacement(o, sw.RackCabinetId)
	case "tape":
		var tl models.TapeLibrary
		if err := o.QueryTable(new(models.TapeLibrary)).Filter("Id", id).
			One(&tl, "Id", "RackCabinetId", "ItiliumStatus"); err != nil {
			flash.Error("Ленточная библиотека не найдена")
			flash.Store(&this.Controller)
			this.Redirect("/sprsrv/equipmentplacement", 302)
			return
		}
		itiliumStatus = tl.ItiliumStatus
		siteId, _, rackId, _ = resolveRackPlacement(o, tl.RackCabinetId)
	default:
		flash.Error("Неизвестный тип оборудования: " + kind)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/equipmentplacement", 302)
		return
	}
	cardURL := equipCardURL(kind, id)
	if models.ItiliumStatusHiddenFromPlans(itiliumStatus) {
		if cardURL == "" {
			flash.Error("Оборудование со статусом «" + models.ItiliumStatusName(itiliumStatus) + "» без карточки")
			flash.Store(&this.Controller)
			this.Redirect("/sprsrv/equipmentplacement", 302)
			return
		}
		this.Redirect(cardURL, 302)
		return
	}
	if siteId <= 0 || rackId <= 0 {
		if cardURL != "" {
			this.Redirect(cardURL, 302)
			return
		}
		flash.Error("У оборудования не указано размещение в шкафу/площадке")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/equipmentplacement?list=1&q=", 302)
		return
	}
	this.Redirect(fmt.Sprintf("/sprsrv/datacenterequipview/%d?rack=%d&kind=%s&eq=%d", siteId, rackId, url.QueryEscape(kind), id), 302)
}
