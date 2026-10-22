package controllers

import (
	"fmt"
	"loadmon/itilliumimport"
	"loadmon/models"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

// AdminImportNetworkEquipment - форма загрузки xlsx-выгрузки Итиллиум по ленточным
// библиотекам (GET) и приём файла (POST) с асинхронным запуском
// itilliumimport.StartImportItiliumNetworkEquipment. Аналог AdminImportStorageSystems.
// Доступ только Priv==3. Маршрут: /admin/importnetworkequipment
func (this *MainController) AdminImportNetworkEquipment() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/importnetworkequipment")
	flash := beego.NewFlash()
	RegisterRequest(this, LogTypeAdminAction)

	if this.Ctx.Input.Method() == "POST" {
		_, header, err := this.GetFile("ImportFile")
		if err != nil {
			flash.Error("Не удалось получить загруженный файл: %v", err)
			flash.Store(&this.Controller)
			return
		}
		if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
			flash.Error("Ожидается файл в формате .xlsx, загружен: %s", header.Filename)
			flash.Store(&this.Controller)
			return
		}
		if err := os.MkdirAll(importDir, 0755); err != nil {
			flash.Error("Не удалось создать каталог %s: %v", importDir, err)
			flash.Store(&this.Controller)
			return
		}

		savedName := fmt.Sprintf("%s_%s", time.Now().Local().Format("20060102_150405"), sanitizeImportFileName(header.Filename))
		destination := filepath.Join(importDir, savedName)
		if err := this.SaveToFile("ImportFile", destination); err != nil {
			flash.Error("Не удалось сохранить файл %s: %v", destination, err)
			flash.Store(&this.Controller)
			return
		}

		itilliumimport.StartImportItiliumNetworkEquipment(savedName)

		flash.Notice("Файл %s загружен, импорт сетевого оборудования запущен - прогресс и итог см. ниже.", savedName)
		flash.Store(&this.Controller)
		this.Redirect("/admin/importnetworkequipment?job="+url.QueryEscape(savedName), 302)
		return
	}

	this.Data["JobId"] = this.GetString("job")
}

// AdminImportNetworkEquipmentStatus отдаёт JSON-снимок прогресса импорта
// (параметр job - имя сохранённого файла). Маршрут: /admin/importnetworkequipment/status
func (this *MainController) AdminImportNetworkEquipmentStatus() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Data["json"] = map[string]string{"error": "доступ запрещён"}
		this.ServeJSON()
		return
	}

	job, ok := itilliumimport.GetImportJob(this.GetString("job"))
	if !ok {
		this.Data["json"] = map[string]string{"error": "задание не найдено"}
		this.ServeJSON()
		return
	}
	this.Data["json"] = job.Snapshot()
	this.ServeJSON()
}
