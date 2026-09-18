package controllers

import (
	"encoding/base64"
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

	// InfraJSON передаётся в шаблон в base64 (см. infrastructure.tpl -
	// значение кладётся в data-атрибут, а не в текст <script>) и
	// раскодируется уже в JS/WASM (infracanvas/main.go:loadData) через
	// atob()+JSON.parse. Раньше сырой JSON вставлялся прямо в текст
	// <script type="application/json">{{.InfraJSON}}</script> - Beego
	// рендерит шаблоны через html/template, который экранирует кавычки в
	// HTML-сущности (&#34; и т.п.); `<script>` в HTML5 - RAWTEXT-элемент, то
	// есть браузер НЕ декодирует HTML-сущности в его содержимом, поэтому
	// WASM получал в textContent заведомо битый JSON (буквально "&#34;"
	// вместо кавычек) и json.Unmarshal ничего не парсил (ошибка к тому же
	// была молча проигнорирована - `_ = json.Unmarshal(...)`, из-за чего
	// страница выглядела "рабочей", но без единой карточки данных). Base64
	// в атрибуте полностью обходит эту проблему - алфавит base64 не требует
	// HTML-экранирования.
	data, err := buildInfrastructureCanvasData()
	if err != nil {
		log.Printf("AdminInfrastructure: build data: %v", err)
		flash.Error("Не удалось загрузить данные инфраструктуры: %v", err)
		this.Data["InfraJSON"] = base64.StdEncoding.EncodeToString([]byte("{}"))
	} else {
		raw, err := json.Marshal(data)
		if err != nil {
			log.Printf("AdminInfrastructure: marshal: %v", err)
			flash.Error("Ошибка сериализации данных инфраструктуры")
			this.Data["InfraJSON"] = base64.StdEncoding.EncodeToString([]byte("{}"))
		} else {
			this.Data["InfraJSON"] = base64.StdEncoding.EncodeToString(raw)
		}
	}

	flash.Store(&this.Controller)
	this.TplName = "admin/infrastructure.tpl"
}
