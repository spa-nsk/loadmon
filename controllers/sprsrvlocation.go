package controllers

import (
	"fmt"
	"loadmon/models"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// serverLocationURI - результат разбора физического размещения ServerSprRecords
// для перехода на просмотр оборудования площадки с выделением шкафа и сервера.
type serverLocationURI struct {
	SiteId int64
	RackId int64
	HwId   int64 // ServerHardware.Id для подсветки в rackcabinetview
}

// SprSrvLocationURI находит местоположение сервера по ServerSprRecords.Id и
// открывает /sprsrv/datacentersiteview/:siteId с выделением шкафа и сервера
// справа (iframe rackcabinetview). Для аппаратных серверов и гипервизоров
// (Type 0/2) - по ServerHardware.RackCabinetId (блейд - по корзине); для ВМ
// (Type 1) - размещение гипервизора, на котором ВМ выполняется в момент
// запроса (тот же алгоритм, что resolveVMPlacementText / vmplacementtext).
//
// Маршрут: /sprsrv/location-uri/:id
func (this *MainController) SprSrvLocationURI() {
	RegisterRequest(this, LogTypeViewReference)
	flash := beego.NewFlash()
	idStr := this.Ctx.Input.Param(":id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор сервера: %s", idStr)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	o := orm.NewOrm()
	loc, errMsg := resolveServerLocationURI(o, id)
	if errMsg != "" {
		flash.Error(errMsg)
		flash.Store(&this.Controller)
		this.Redirect(fmt.Sprintf("/sprsrv/updatebase-tab/%d", id), 302)
		return
	}
	this.Redirect(fmt.Sprintf("/sprsrv/datacentersiteview/%d?rack=%d&hw=%d", loc.SiteId, loc.RackId, loc.HwId), 302)
}

// resolveServerLocationURI возвращает площадку/шкаф/ServerHardware для
// выделения на плане; errMsg - человекочитаемая причина, если размещение
// определить нельзя.
func resolveServerLocationURI(o orm.Ormer, sprId int64) (serverLocationURI, string) {
	var spr models.ServerSprRecords
	if err := models.ServerSprQS(o).Filter("Id", sprId).One(&spr, "Id", "Type", "Name", "HWSprId", "VMSprId"); err != nil {
		return serverLocationURI{}, fmt.Sprintf("Сервер Id=%d не найден", sprId)
	}

	hwId := spr.HWSprId
	if spr.Type == 1 {
		// ВМ: живой хост ESXi → ServerHardware гипервизора (как vmplacementtext)
		if spr.VMSprId <= 0 {
			return serverLocationURI{}, "У виртуальной машины не указана ссылка на справочник ВМ"
		}
		_, hostHW := resolveVMPlacementText(o, spr.VMSprId)
		if hostHW <= 0 {
			return serverLocationURI{}, "Не удалось определить гипервизор или его размещение для ВМ \"" + spr.Name + "\""
		}
		hwId = hostHW
	}
	if hwId <= 0 {
		// запасной путь: ServerHardware.ServerId
		var hwBySrv models.ServerHardware
		if err := o.QueryTable(new(models.ServerHardware)).Filter("ServerId", spr.Id).One(&hwBySrv, "Id"); err == nil && hwBySrv.Id > 0 {
			hwId = hwBySrv.Id
		}
	}
	if hwId <= 0 {
		return serverLocationURI{}, "У сервера \"" + spr.Name + "\" нет связанной записи оборудования (ServerHardware)"
	}
	return locationURIFromHardware(o, hwId)
}

// locationURIFromHardware - площадка и шкаф по ServerHardware.Id (блейд → шкаф корзины).
func locationURIFromHardware(o orm.Ormer, hwId int64) (serverLocationURI, string) {
	var hw models.ServerHardware
	if err := o.QueryTable(new(models.ServerHardware)).Filter("Id", hwId).
		One(&hw, "Id", "RackCabinetId", "FormFactor", "BladeSystemId", "SerialNum"); err != nil {
		return serverLocationURI{}, fmt.Sprintf("Оборудование Id=%d не найдено", hwId)
	}

	rackId := hw.RackCabinetId
	if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
		var ch models.BladeSystems
		if err := o.QueryTable(new(models.BladeSystems)).Filter("Id", hw.BladeSystemId).
			One(&ch, "Id", "RackCabinetId"); err == nil && ch.RackCabinetId > 0 {
			rackId = ch.RackCabinetId
		}
	}
	if rackId <= 0 {
		label := strings.TrimSpace(hw.SerialNum)
		if label == "" {
			label = fmt.Sprintf("Id=%d", hw.Id)
		}
		return serverLocationURI{}, "У оборудования " + label + " не указан шкаф"
	}

	var rc models.RackCabinet
	if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", rackId).
		One(&rc, "Id", "DatacenterSiteId", "Name"); err != nil {
		return serverLocationURI{}, fmt.Sprintf("Шкаф Id=%d не найден", rackId)
	}
	if rc.DatacenterSiteId <= 0 {
		return serverLocationURI{}, "У шкафа \"" + rc.Name + "\" не указана площадка/ЦОД"
	}
	return serverLocationURI{SiteId: rc.DatacenterSiteId, RackId: rc.Id, HwId: hw.Id}, ""
}
