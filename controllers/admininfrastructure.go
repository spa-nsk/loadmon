package controllers

import (
	"encoding/json"
	"log"
	models "loadmon/models"

	beego "github.com/beego/beego/v2/server/web"
)

// AdminInfrastructure отображает интерактивную canvas-диаграмму инфраструктуры
// (WASM-модуль infracanvas + go-canvas). Доступ только Priv==3.
//
// Маршрут: /admin/infrastructure
func (this *MainController) AdminInfrastructure() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/infrastructure")
	flash := beego.NewFlash()
	RegisterRequest(this, LogTypeAdminAction)

	data, err := buildInfrastructureCanvasData()
	if err != nil {
		log.Printf("AdminInfrastructure: build data: %v", err)
		flash.Error("Не удалось загрузить данные инфраструктуры: " + err.Error())
		this.Data["InfraJSON"] = "{}"
	} else {
		raw, err := json.Marshal(data)
		if err != nil {
			log.Printf("AdminInfrastructure: marshal: %v", err)
			flash.Error("Ошибка сериализации данных инфраструктуры")
			this.Data["InfraJSON"] = "{}"
		} else {
			this.Data["InfraJSON"] = string(raw)
		}
	}

	flash.Store(&this.Controller)
	this.TplName = "admin/infrastructure.tpl"
}
