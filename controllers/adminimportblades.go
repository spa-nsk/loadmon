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

// AdminImportBladeSystems отображает форму загрузки xlsx-выгрузки Итиллиум
// по блейд-корзинам (GET) и принимает загруженный файл (POST): сохраняет
// его в каталог importDir на сервере (тот же каталог, что и для импорта
// rack-серверов - см. importDir/sanitizeImportFileName в
// adminimportrackservers.go) и запускает itilliumimport.StartImportItiliumBlade(name)
// с именем сохранённого файла. Полный аналог AdminImportRackServers, только
// для models.BladeSystems. Доступ только Priv==3.
//
// ИЗМЕНЕНО (2026-09-08, по запросу пользователя "выводить статистику по
// итогу" + "отображать процесс импорта в виде градуированного линейного
// индикатора с процентами"): импорт запускается АСИНХРОННО (в отдельной
// горутине через StartImportItiliumBlade, а не синхронно ImportItiliumBlade,
// как раньше) - страница после загрузки файла остаётся на /admin/importblades
// с параметром ?job=<имя файла> и опрашивает прогресс через AJAX-эндпоинт
// AdminImportBladeSystemsStatus, см. server_web_new/views/admin/importblades.tpl.
//
// Маршрут: /admin/importblades
func (this *MainController) AdminImportBladeSystems() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/importblades")
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

		itilliumimport.StartImportItiliumBlade(savedName)

		flash.Notice("Файл %s загружен, импорт блейд-корзин запущен - прогресс и итог см. ниже.", savedName)
		flash.Store(&this.Controller)
		this.Redirect("/admin/importblades?job="+url.QueryEscape(savedName), 302)
		return
	}

	// jobId - имя файла последнего запущенного через эту страницу импорта
	// (см. редирект выше) - если параметр передан, шаблон отрисовывает
	// градуированный индикатор прогресса и опрашивает AdminImportBladeSystemsStatus.
	this.Data["JobId"] = this.GetString("job")
}

// AdminImportBladeSystemsStatus отдаёт JSON-снимок прогресса ранее
// запущенного (через AdminImportBladeSystems) импорта блейд-корзин по
// идентификатору задания (параметр "job" - имя сохранённого файла) - см.
// itilliumimport.GetImportJob/ImportProgress.Snapshot. Опрашивается со
// страницы /admin/importblades через AJAX каждую секунду, пока задание не
// завершится (см. server_web_new/views/admin/importblades.tpl).
//
// ИЗМЕНЕНО (2026-09-08): добавлено вместе с асинхронным запуском импорта в
// AdminImportBladeSystems, по запросу пользователя "отображать процесс
// импорта в виде градуированного линейного индикатора с процентами".
//
// Маршрут: /admin/importblades/status
func (this *MainController) AdminImportBladeSystemsStatus() {
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
