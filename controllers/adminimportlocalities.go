package controllers

import (
	"fmt"
	"loadmon/locality"
	"loadmon/models"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

// AdminImportLocalities отображает форму загрузки файла справочника
// населённых пунктов (GET) и принимает загруженный файл (POST): сохраняет его
// в importDir (как и прочие режимы импорта, см. adminimportstorages.go) и
// запускает locality.Start(...) - асинхронную полную перезагрузку таблицы
// cmdb_locality. Источник - открытые данные ОКТМО Росстата (csv, разделитель
// ";", UTF-8; см. itilliumimport/parsing/oktmo_data_*.csv). Доступ только
// Priv==3.
//
// Маршрут: /admin/importlocalities
func (this *MainController) AdminImportLocalities() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/importlocalities")
	flash := beego.NewFlash()
	RegisterRequest(this, LogTypeAdminAction)

	if this.Ctx.Input.Method() == "POST" {
		_, header, err := this.GetFile("ImportFile")
		if err != nil {
			flash.Error("Не удалось получить загруженный файл: %v", err)
			flash.Store(&this.Controller)
			return
		}
		if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
			flash.Error("Ожидается файл в формате .csv (открытые данные ОКТМО), загружен: %s", header.Filename)
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

		locality.Start(savedName, destination)

		flash.Notice("Файл %s загружен, импорт справочника населённых пунктов запущен - прогресс и итог см. ниже.", savedName)
		flash.Store(&this.Controller)
		this.Redirect("/admin/importlocalities?job="+url.QueryEscape(savedName), 302)
		return
	}

	this.Data["JobId"] = this.GetString("job")
}

// AdminImportLocalitiesStatus отдаёт JSON-снимок прогресса загрузки
// справочника населённых пунктов по параметру "job" (имя сохранённого файла).
//
// Маршрут: /admin/importlocalities/status
func (this *MainController) AdminImportLocalitiesStatus() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Data["json"] = map[string]string{"error": "доступ запрещён"}
		this.ServeJSON()
		return
	}

	job, ok := locality.GetJob(this.GetString("job"))
	if !ok {
		this.Data["json"] = map[string]string{"error": "задание не найдено"}
		this.ServeJSON()
		return
	}
	this.Data["json"] = job.Snapshot()
	this.ServeJSON()
}
