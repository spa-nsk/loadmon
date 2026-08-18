package controllers

import (
	"fmt"
	"loadmon/grpcclient"

	beego "github.com/beego/beego/v2/server/web"
	oneview "github.com/spa-nsk/oneview2"
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
	RegisterRequest(this, LogTypeServerDetails) // карточка сервера HPE OneView по серийному номеру

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
	// oneview2.Server не хранит готовый итоговый объём памяти (в отличие от
	// старого Base.MemoryMb) - считаем его по установленным модулям памяти.
	this.Data["TotalMemoryMb"] = oneview.TotalMemoryMb(hw)

	// Размещение в шкафу задаётся верхним юнитом крепления (TopUSlot) и
	// высотой в юнитах (MountUHeight); если OneView отдаёт готовое текстовое
	// расположение (MountLocation), шаблон использует его напрямую и это
	// поле не нужно. Диапазон вычисляется здесь, а не в шаблоне, потому что
	// html/template по умолчанию не регистрирует арифметические функции.
	this.Data["RackSlotRange"] = ""
	if hw.Rack.TopUSlot > 0 {
		if hw.Rack.MountUHeight > 1 {
			this.Data["RackSlotRange"] = fmt.Sprintf("U%d–U%d", hw.Rack.TopUSlot, hw.Rack.TopUSlot+hw.Rack.MountUHeight-1)
		} else {
			this.Data["RackSlotRange"] = fmt.Sprintf("U%d", hw.Rack.TopUSlot)
		}
	}
}
