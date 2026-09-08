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

// AdminImportStorageSystems отображает форму загрузки xlsx-выгрузки Итиллиум
// по СХД (GET) и принимает загруженный файл (POST): сохраняет его в каталог
// importDir на сервере (тот же каталог, что и для импорта rack-серверов/
// блейд-корзин - см. importDir/sanitizeImportFileName в
// adminimportrackservers.go) и запускает itilliumimport.
// StartImportItiliumStorage(name) с именем сохранённого файла. Полный аналог
// AdminImportBladeSystems (adminimportblades.go), только для
// models.StorageSystems. Доступ только Priv==3.
//
// 2026-09-08, по запросу пользователя "добавь в меню Импорт пункт «Импорт
// СХД из Итилиум» и реализуй импорт по аналогии с блэйд корзинами" - импорт
// запускается АСИНХРОННО (в отдельной горутине через
// StartImportItiliumStorage) - страница после загрузки файла остаётся на
// /admin/importstorages с параметром ?job=<имя файла> и опрашивает прогресс
// через AJAX-эндпоинт AdminImportStorageSystemsStatus, см.
// server_web_new/views/admin/importstorages.tpl.
//
// Маршрут: /admin/importstorages
func (this *MainController) AdminImportStorageSystems() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/importstorages")
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

		itilliumimport.StartImportItiliumStorage(savedName)

		flash.Notice("Файл %s загружен, импорт СХД запущен - прогресс и итог см. ниже.", savedName)
		flash.Store(&this.Controller)
		this.Redirect("/admin/importstorages?job="+url.QueryEscape(savedName), 302)
		return
	}

	// jobId - имя файла последнего запущенного через эту страницу импорта
	// (см. редирект выше) - если параметр передан, шаблон отрисовывает
	// градуированный индикатор прогресса и опрашивает AdminImportStorageSystemsStatus.
	this.Data["JobId"] = this.GetString("job")
}

// AdminImportStorageSystemsStatus отдаёт JSON-снимок прогресса ранее
// запущенного (через AdminImportStorageSystems) импорта СХД по
// идентификатору задания (параметр "job" - имя сохранённого файла) - см.
// itilliumimport.GetImportJob/ImportProgress.Snapshot. Опрашивается со
// страницы /admin/importstorages через AJAX каждую секунду, пока задание не
// завершится (см. server_web_new/views/admin/importstorages.tpl).
//
// Маршрут: /admin/importstorages/status
func (this *MainController) AdminImportStorageSystemsStatus() {
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
