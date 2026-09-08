package controllers

import (
	"fmt"
	"loadmon/itilliumimport"
	"loadmon/models"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

// importDir - каталог на сервере, куда сохраняются загруженные через
// /admin/importrackservers xlsx-файлы перед вызовом
// itilliumimport.ImportItiliumServer. Аналог "./reports/" (см.
// controllers/sprhwconfig.go и др.), только для входящих файлов, а не
// исходящих отчётов.
const importDir = "./import"

// importFileNameRe - разрешённые символы в имени сохраняемого файла (после
// санитайзинга оригинального имени загруженного файла) - защита от
// path traversal (например, "../../etc/passwd") и прочих спецсимволов в
// имени файла, которое пришло от клиента и не должно использоваться в пути
// на диске без проверки.
var importFileNameRe = regexp.MustCompile(`[^A-Za-z0-9а-яА-ЯёЁ._-]+`)

// sanitizeImportFileName убирает путь (оставляет только базовое имя файла)
// и заменяет все символы вне разрешённого набора на "_", чтобы имя,
// полученное от браузера (FileHeader.Filename), нельзя было использовать
// для выхода за пределы importDir или для инъекции спецсимволов в путь.
func sanitizeImportFileName(name string) string {
	name = filepath.Base(name)
	name = importFileNameRe.ReplaceAllString(name, "_")
	if name == "" || name == "_" {
		name = "import.xlsx"
	}
	return name
}

// AdminImportRackServers отображает форму загрузки xlsx-выгрузки Итиллиум
// по rack-серверам (GET) и принимает загруженный файл (POST): сохраняет
// его в каталог importDir на сервере и запускает
// itilliumimport.StartImportItiliumServer(name) с именем сохранённого файла.
// Доступ только Priv==3, приём идентичен другим страницам /admin/* (см.
// controllers/admininfrastructure.go).
//
// ИЗМЕНЕНО (2026-09-08, по запросу пользователя "сделай аналогичный progress
// bar для импорта rack серверов и blade серверов" - то же решение, что уже
// сделано для блейд-корзин, см. controllers/adminimportblades.go): импорт
// запускается АСИНХРОННО, страница остаётся на /admin/importrackservers с
// параметром ?job=<имя файла> и опрашивает прогресс через AJAX-эндпоинт
// AdminImportRackServersStatus.
//
// Маршрут: /admin/importrackservers
func (this *MainController) AdminImportRackServers() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/importrackservers")
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

		// Имя на диске: временная метка + санитайзированное оригинальное
		// имя - защищает от перезаписи одноимённых файлов при повторной
		// загрузке и от path traversal/спецсимволов в имени клиента.
		savedName := fmt.Sprintf("%s_%s", time.Now().Local().Format("20060102_150405"), sanitizeImportFileName(header.Filename))
		destination := filepath.Join(importDir, savedName)

		if err := this.SaveToFile("ImportFile", destination); err != nil {
			flash.Error("Не удалось сохранить файл %s: %v", destination, err)
			flash.Store(&this.Controller)
			return
		}

		itilliumimport.StartImportItiliumServer(savedName)

		flash.Notice("Файл %s загружен, импорт rack-серверов запущен - прогресс и итог см. ниже.", savedName)
		flash.Store(&this.Controller)
		this.Redirect("/admin/importrackservers?job="+url.QueryEscape(savedName), 302)
		return
	}

	// jobId - имя файла последнего запущенного через эту страницу импорта
	// (см. редирект выше) - если параметр передан, шаблон отрисовывает
	// градуированный индикатор прогресса и опрашивает AdminImportRackServersStatus.
	this.Data["JobId"] = this.GetString("job")
}

// AdminImportRackServersStatus отдаёт JSON-снимок прогресса ранее
// запущенного (через AdminImportRackServers) импорта rack-серверов по
// идентификатору задания (параметр "job" - имя сохранённого файла) - см.
// itilliumimport.GetImportJob/ImportProgress.Snapshot. Полный аналог
// AdminImportBladeSystemsStatus (controllers/adminimportblades.go).
//
// Маршрут: /admin/importrackservers/status
func (this *MainController) AdminImportRackServersStatus() {
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
