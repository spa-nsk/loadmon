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

// AdminImportBladeServers отображает форму загрузки xlsx-выгрузки Итиллиум
// по блейд-серверам (GET) и принимает загруженный файл (POST): сохраняет
// его в каталог importDir на сервере (тот же каталог, что и для импорта
// rack-серверов/блейд-корзин - см. importDir/sanitizeImportFileName в
// adminimportrackservers.go) и запускает itilliumimport.StartImportItiliumBladeServer(name)
// с именем сохранённого файла. Полный аналог AdminImportRackServers, только
// для блейд-серверов (см. itilliumimport/importbladeservers.go про
// дополнительную привязку к models.BladeSystems). Доступ только Priv==3.
//
// ИЗМЕНЕНО (2026-09-08, по запросу пользователя "сделай аналогичный progress
// bar для импорта rack серверов и blade серверов"): импорт запускается
// АСИНХРОННО, страница остаётся на /admin/importbladeservers с параметром
// ?job=<имя файла> и опрашивает прогресс через AJAX-эндпоинт
// AdminImportBladeServersStatus - см. controllers/adminimportblades.go для
// того же решения на блейд-корзинах.
//
// Маршрут: /admin/importbladeservers
func (this *MainController) AdminImportBladeServers() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/importbladeservers")
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

		itilliumimport.StartImportItiliumBladeServer(savedName)

		flash.Notice("Файл %s загружен, импорт блейд-серверов запущен - прогресс и итог см. ниже.", savedName)
		flash.Store(&this.Controller)
		this.Redirect("/admin/importbladeservers?job="+url.QueryEscape(savedName), 302)
		return
	}

	// jobId - имя файла последнего запущенного через эту страницу импорта
	// (см. редирект выше) - если параметр передан, шаблон отрисовывает
	// градуированный индикатор прогресса и опрашивает AdminImportBladeServersStatus.
	this.Data["JobId"] = this.GetString("job")
}

// AdminImportBladeServersStatus отдаёт JSON-снимок прогресса ранее
// запущенного (через AdminImportBladeServers) импорта блейд-серверов по
// идентификатору задания (параметр "job" - имя сохранённого файла) - см.
// itilliumimport.GetImportJob/ImportProgress.Snapshot. Полный аналог
// AdminImportBladeSystemsStatus (controllers/adminimportblades.go).
//
// Маршрут: /admin/importbladeservers/status
func (this *MainController) AdminImportBladeServersStatus() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Data["json"] = map[string]string{"error": "доступ запрещён"}
		this.ServeJSON()
		return
	}

	jobId := this.GetString("job")
	job, ok := itilliumimport.GetImportJob(jobId)
	if !ok {
		this.Data["json"] = map[string]string{"error": "задание не найдено"}
		this.ServeJSON()
		return
	}

	this.Data["json"] = job.Snapshot()
	this.ServeJSON()
}
