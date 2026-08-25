// Package controllers реализует веб-контроллеры интерфейса пользователя (beego)
// системы мониторинга loadmon: отображение и фильтрацию списка серверов и
// виртуальных машин, карточки серверов с показателями нагрузки (CPU, память,
// диски) и графиками производительности, работу со справочником серверов
// (добавление, обновление данных из систем-источников), аутентификацию и
// профиль пользователя, журналирование действий, а также вспомогательный API
// для внешних систем.
package controllers

import (
	"encoding/base64"
	"io/ioutil"
	"log"

	//"github.com/astaxie/beego"
	//"github.com/astaxie/beego/orm"

	beego "github.com/beego/beego/v2/server/web"

	//	"github.com/astaxie/beego/validation"

	//	exportres "loadmon/export"
	models "loadmon/models"
)

type ErrorController struct {
	beego.Controller
}

func (this *ErrorController) activeContent(view string) {
	this.Layout = "basic-layout.tpl"
	this.LayoutSections = make(map[string]string)
	this.LayoutSections["Header"] = "header.tpl"
	//	this.LayoutSections["Sidebar"] = "sidebar.tpl"
	this.LayoutSections["Footer"] = "footer.tpl"
	this.TplName = view + ".tpl"
	this.Data["domainname"] = "https://spa.usvc.global.bcs"
	this.Data["StaticVer"] = staticAssetVersion // cache-busting для ссылок на /static/... (см. комментарий у staticAssetVersion в main_controller.go) - нужен на 404.tpl/500.tpl/dberror.tpl, которые ссылаются на /static/js/reload.min.js
	//this.Data["domainname"] = "yourdomainname"

	sess := this.GetSession("loadmon")
	if sess != nil {
		this.Data["InSession"] = 1 // for login bar in header.tpl
		this.Data["Filter"] = sess.(models.LoadFilter)
		this.Data["Filter_save"] = sess.(models.LoadFilter)
		p := this.Data["Filter"]
		filter := p.(models.LoadFilter)
		m := filter.Map
		this.Data["First"] = m["first"]                 //отобразить логин
		this.Data["Last"] = m["last"]                   //отобразить логин
		this.Data["Username"] = m["username"]           //отобразить логин
		this.Data["Privilage"] = m["privilage"]         // 0 - обычный пользователь 1 - разрешено редактировать справочник 3 - админ
		this.Data["Podrazdelenie"] = m["podrazdelenie"] // 0 - не указано 1 - ДАТИ 2 - СТО
	}
}

// Error404 отображает страницу "страница не найдена" (HTTP 404).
func (this *ErrorController) Error404() {
	//
	this.activeContent("404")
	this.Data["content"] = "Страница не найдена."
	destination := "./static/img/gopher_china.png"
	data, err := ioutil.ReadFile(destination)
	if err != nil {
		log.Println("Ошибка чтения файла: ", destination, " - ", err)
	} else {
		this.Data["Image"] = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

// Error500 отображает страницу внутренней ошибки сервера (HTTP 500).
func (this *ErrorController) Error500() {
	this.activeContent("500")
	this.Data["content"] = "Внутренняя ошибка сервера."
	destination := "./static/img/admin.png"
	data, err := ioutil.ReadFile(destination)
	if err != nil {
		log.Println("Ошибка чтения файла: ", destination, " - ", err)
	} else {
		this.Data["Image"] = base64.StdEncoding.EncodeToString([]byte(data))
	}
}

// ErrorDb отображает страницу ошибки при работе с базой данных.
func (this *ErrorController) ErrorDb() {
	this.activeContent("dberror")
	this.Data["content"] = "Ошибка с базой данных."
}
