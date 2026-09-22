package controllers

import (
	"fmt"
	"loadmon/grpcclient"
	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
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

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
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

	this.fillDBPlacement(sn)
}

// fillDBPlacement заполняет блок "Размещение" данными из справочника
// конфигураций серверов БД (models.ServerHardware, таблица server_hardware),
// найденного ПО СЕРИЙНОМУ НОМЕРУ (SerialNum) - по запросу пользователя "в
// форме /sprsrv/srvhwviewbysn блок размещение заполняй на основе данных из
// таблицы ServerHardware связав по серийному номеру". В отличие от block
// "Размещение" в /sprsrv/hwviewlist (где данные из БД лишь ДОПОЛНЯЮТ текст
// из "живых" данных OneView, см. CHANGELOG.md запись (102)), здесь именно
// ServerHardware - основной источник для полей ниже; поля из hw (oneview2)
// остаются в Data и используются шаблоном как fallback, только если в
// ServerHardware соответствующее значение не заполнено (RackCabinetId==0,
// SizeUnit==0 и т.п.) - так карточка не "пустеет" для серверов, ещё не
// привязанных к справочнику шкафов.
func (this *MainController) fillDBPlacement(sn string) {
	this.Data["DBDatacenterName"] = ""
	this.Data["DBRackName"] = ""
	this.Data["DBRackSlotRange"] = ""
	this.Data["DBEnclosureName"] = ""
	this.Data["DBBayNumber"] = 0

	if sn == "" {
		return
	}

	o := orm.NewOrm()
	var hwdb models.ServerHardware
	if err := o.QueryTable(new(models.ServerHardware)).Filter("SerialNum", sn).One(&hwdb); err != nil {
		return // сервера нет в справочнике конфигураций БД - весь блок останется на fallback из OneView (см. шаблон)
	}

	if hwdb.RackCabinetId > 0 {
		var rc models.RackCabinet
		if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", hwdb.RackCabinetId).One(&rc, "Id", "Name", "DatacenterSiteId"); err == nil {
			this.Data["DBRackName"] = rc.Name
			if rc.DatacenterSiteId > 0 {
				var site models.DatacenterSite
				if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", rc.DatacenterSiteId).One(&site, "Name"); err == nil {
					this.Data["DBDatacenterName"] = site.Name
				}
			}
		}
	}

	if hwdb.StartUnitPos > 0 {
		if hwdb.SizeUnit > 1 {
			this.Data["DBRackSlotRange"] = fmt.Sprintf("U%d–U%d", hwdb.StartUnitPos, hwdb.StartUnitPos+hwdb.SizeUnit-1)
		} else {
			this.Data["DBRackSlotRange"] = fmt.Sprintf("U%d", hwdb.StartUnitPos)
		}
	}

	if hwdb.BladeSystemId > 0 {
		var bs models.BladeSystems
		if err := o.QueryTable(new(models.BladeSystems)).Filter("Id", hwdb.BladeSystemId).One(&bs, "Name"); err == nil {
			this.Data["DBEnclosureName"] = bs.Name
		}
	}
	if hwdb.BladeNum > 0 {
		this.Data["DBBayNumber"] = hwdb.BladeNum
	}
}
