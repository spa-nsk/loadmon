package controllers

import (
	"loadmon/grpcclient"

	beego "github.com/beego/beego/v2/server/web"
)

// SrvHwViewBySn отображает подробную карточку сервера HPE OneView по его
// серийному номеру: основные характеристики (модель, процессоры, память,
// состояние), модули памяти, локальные RAID-контроллеры с логическими и
// физическими дисками, размещение (датацентр/стойка).
//
// Ссылка на этот маршрут стоит в списке /sprsrv/hwviewlist (каждая строка
// таблицы, столбец "Серийный №").
//
// Маршрут: /sprsrv/srvhwviewbysn/:SN
func (this *MainController) SrvHwViewBySn() {

	sn := this.Ctx.Input.Param(":SN")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_srvhwviewbysn_"+sn, 302)
		return
	}

	this.activeContent("sprsrv/srvhwviewbysn")

	this.Data["SN"] = sn

	flash := beego.NewFlash()

	hw, ok := grpcclient.GetServerHardwareBySN(sn)
	if !ok || hw == nil {
		flash.Error("Сервер с серийным номером \"%s\" не найден.", sn)
		flash.Store(&this.Controller)
		return
	}

	this.Data["HW"] = hw
}
