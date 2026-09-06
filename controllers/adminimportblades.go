package controllers

import (
	"fmt"
	"loadmon/itilliumimport"
	"loadmon/models"
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
// adminimportrackservers.go) и вызывает itilliumimport.ImportItiliumBlade(name)
// с именем сохранённого файла. Полный аналог AdminImportRackServers, только
// для models.BladeSystems. Доступ только Priv==3.
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

		itilliumimport.ImportItiliumBlade(savedName)

		flash.Notice("Файл %s загружен и передан на импорт блейд-корзин (см. итог в логе сервера).", savedName)
		flash.Store(&this.Controller)
		this.Redirect("/admin/importblades", 302)
		return
	}
}
