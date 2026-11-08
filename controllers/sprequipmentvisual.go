package controllers

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// equipVisualHit - результат поиска для 3D-визуализации конкретного устройства.
// Implemented=true только для блейд-корзин и серверов (rack/blade);
// для СХД / сетевого / ленточных библиотек — сообщение «пока не реализовано».
type equipVisualHit struct {
	Kind        string
	KindLabel   string
	Id          int64
	Name        string
	MatchField  string
	MatchValue  string
	StatusName  string
	CardURL     string
	VisualURL   string // 3D-страница, если Implemented
	Implemented bool
}

// equipVisualURL возвращает URL 3D-вида для kind+id и флаг поддержки.
// Сейчас: server → /sprsrv/hwconfvisual/:id (rack и blade),
// chassis → /sprsrv/bladesystemserverslistvisual/:id.
func equipVisualURL(kind string, id int64) (url string, ok bool) {
	switch kind {
	case "server":
		return fmt.Sprintf("/sprsrv/hwconfvisual/%d", id), true
	case "chassis":
		return fmt.Sprintf("/sprsrv/bladesystemserverslistvisual/%d", id), true
	default:
		return "", false
	}
}

func toEquipVisualHits(src []equipPlacementHit) []equipVisualHit {
	out := make([]equipVisualHit, 0, len(src))
	for _, h := range src {
		v := equipVisualHit{
			Kind: h.Kind, KindLabel: h.KindLabel, Id: h.Id, Name: h.Name,
			MatchField: h.MatchField, MatchValue: h.MatchValue,
			StatusName: h.StatusName, CardURL: h.CardURL,
		}
		if u, ok := equipVisualURL(h.Kind, h.Id); ok {
			v.Implemented = true
			v.VisualURL = u
		}
		out = append(out, v)
	}
	return out
}

// SprEquipmentVisual - поиск оборудования по номеру (как «Поиск размещения»)
// с открытием 3D-вида конкретного устройства (корзина / rack / blade).
// Для СХД, сетевого оборудования и ленточных библиотек — «пока не реализовано».
//
// Маршрут: /sprsrv/equipmentvisual  (GET ?q=)
func (this *MainController) SprEquipmentVisual() {
	RegisterRequest(this, LogTypeViewReference)
	this.activeContent("sprsrv/equipmentvisual")

	q := strings.TrimSpace(this.GetString("q"))
	this.Data["Query"] = q
	this.Data["Results"] = []equipVisualHit{}
	this.Data["ResultCount"] = 0
	this.Data["NotImplemented"] = false
	this.Data["NotImplementedHit"] = equipVisualHit{}

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
	hits := toEquipVisualHits(searchEquipmentPlacement(o, q))
	this.Data["Results"] = hits
	this.Data["ResultCount"] = len(hits)

	if len(hits) == 0 {
		flash.Notice("По запросу «" + q + "» оборудование не найдено")
		return
	}
	if len(hits) == 1 && this.GetString("list") != "1" {
		h := hits[0]
		if h.Implemented && h.VisualURL != "" {
			this.Redirect(h.VisualURL, 302)
			return
		}
		this.Data["NotImplemented"] = true
		this.Data["NotImplementedHit"] = h
		flash.Notice(fmt.Sprintf("Визуализация для «%s» (%s) пока не реализована", h.KindLabel, h.Name))
		return
	}
}

// SprEquipmentVisualGo - переход из списка результатов на 3D-вид (или сообщение).
// Маршрут: /sprsrv/equipmentvisual/go/:kind/:id
func (this *MainController) SprEquipmentVisualGo() {
	RegisterRequest(this, LogTypeViewReference)
	flash := beego.NewFlash()
	kind := strings.TrimSpace(this.Ctx.Input.Param(":kind"))
	idStr := this.Ctx.Input.Param(":id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор оборудования")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/equipmentvisual", 302)
		return
	}
	if u, ok := equipVisualURL(kind, id); ok {
		this.Redirect(u, 302)
		return
	}
	label := kind
	switch kind {
	case "storage":
		label = "СХД"
	case "switch":
		label = "Сетевое оборудование"
	case "tape":
		label = "Ленточная библиотека"
	case "server":
		label = "Сервер"
	case "chassis":
		label = "Блейд-корзина"
	}
	flash.Notice(fmt.Sprintf("Визуализация для «%s» пока не реализована", label))
	flash.Store(&this.Controller)
	q := strings.TrimSpace(this.GetString("q"))
	if q != "" {
		this.Redirect("/sprsrv/equipmentvisual?list=1&q="+url.QueryEscape(q), 302)
		return
	}
	this.Redirect("/sprsrv/equipmentvisual", 302)
}
