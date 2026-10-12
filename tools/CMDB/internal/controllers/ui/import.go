package ui

import (
	"context"
	"fmt"
	"strconv"

	"cmdb/internal/importer/pipeline"
	"cmdb/internal/importer/worker"
)

// ImportController — загрузка CSV/XLSX и очередь конфликтов (plan.md E4.9/E4.10). MVP: один
// коннектор (csv-server), без разрешения конфликтов из UI — только просмотр очереди.
type ImportController struct {
	BaseController
}

func (c *ImportController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *ImportController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

// Index — страница загрузки: выбор коннектора, файл, переключатель dry-run, история прогонов.
func (c *ImportController) Index() {
	connectors, err := pipeline.ListConnectors(c.ctx())
	c.must(err)
	runs, err := pipeline.ListRuns(c.ctx(), 20)
	c.must(err)
	openConflicts, err := pipeline.OpenConflictCount(c.ctx())
	c.must(err)

	jobs, err := pipeline.ListJobs(c.ctx(), 15)
	c.must(err)

	var csvConns, apiConns []pipeline.ConnectorRow
	for _, cn := range connectors {
		if cn.Kind == "csv" {
			csvConns = append(csvConns, cn)
		} else {
			apiConns = append(apiConns, cn)
		}
	}

	c.Data["Nav"] = "import"
	c.Data["Connectors"] = csvConns
	c.Data["APIConnectors"] = apiConns
	c.Data["Jobs"] = jobs
	c.Data["Runs"] = runs
	c.Data["OpenConflicts"] = openConflicts
	c.page("import/index.tpl")
}

// Template отдаёт CSV-шаблон колонок для класса КЕ коннектора (E4.9).
func (c *ImportController) Template() {
	class := c.GetString("class")
	b, err := pipeline.TemplateCSV(class)
	c.must(err)
	c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="import-%s-template.csv"`, class))
	c.Ctx.Output.Body(b)
	c.StopRun()
}

// Run загружает файл и прогоняет его через пайплайн (dry-run или реальное применение), затем
// перенаправляет на карточку результата — тяжёлые файлы этот MVP не рассчитан обрабатывать
// асинхронно (imp.job/cmdb-worker — вне текущего объёма, см. E5).
func (c *ImportController) Run() {
	connectorCode := c.GetString("connector")
	if connectorCode == "" {
		c.failText("Выберите источник.")
	}
	file, header, err := c.Ctx.Request.FormFile("file")
	if err != nil {
		c.failText("Выберите файл CSV или XLSX.")
	}
	defer file.Close()
	dryRun, _ := c.GetBool("dry_run", false)

	format := pipeline.DetectFormat(header.Filename)
	sum, err := pipeline.Run(c.ctx(), connectorCode, file, format, dryRun, c.userID())
	c.must(err)

	c.logEvent("import_run", map[string]any{"connector": connectorCode, "run_id": sum.RunID, "dry_run": dryRun})
	c.redirect("/import/runs/" + strconv.FormatInt(sum.RunID, 10))
}

// RunDetail — карточка результата одного прогона.
func (c *ImportController) RunDetail() {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	run, ok, err := pipeline.GetRun(c.ctx(), id)
	c.must(err)
	if !ok {
		c.notFoundPage("Прогон импорта не найден")
		return
	}
	c.Data["Nav"] = "import"
	c.Data["Run"] = run
	c.page("import/run.tpl")
}

// Conflicts — очередь конфликтов (E4.10): неоднозначная идентификация, заблокированные атрибуты,
// не найденные источником КЕ. Только просмотр — разбор (привязать/создать/игнорировать) в UI не входит в этот MVP.
func (c *ImportController) Conflicts() {
	f := pipeline.ConflictFilter{Status: c.GetString("status", "open"), Kind: c.GetString("kind")}
	f.RunID, _ = c.GetInt64("run_id", 0)
	f.Page, _ = c.GetInt("page", 1)
	f.PerPage, _ = c.GetInt("per", 25)

	res, err := pipeline.ListConflicts(c.ctx(), f)
	c.must(err)

	c.Data["Filter"] = f
	c.Data["Result"] = res
	if c.isHTMX() {
		c.fragment("import/_conflicts_results.tpl")
		return
	}
	c.Data["Nav"] = "import"
	c.page("import/conflicts.tpl")
}

// ConnectorRun ставит API-коннектору задание «запустить сейчас» (обычное или dry-run); выполняет
// его cmdb-worker. Повторное нажатие при уже ожидающем задании новых не создаёт.
func (c *ImportController) ConnectorRun() {
	code := c.Ctx.Input.Param(":code")
	dryRun, _ := c.GetBool("dry_run", false)
	id, created, err := worker.Enqueue(c.ctx(), code, "manual", dryRun, c.userID())
	c.must(err)
	c.logEvent("import_enqueue", map[string]any{"connector": code, "job_id": id, "dry_run": dryRun})
	if created {
		c.ok("Задание поставлено в очередь. Выполняет cmdb-worker.")
	} else {
		c.trigger(map[string]any{"toast": toastPayload{Kind: "info", Text: "Задание для этого источника уже ожидает выполнения."}})
	}
	c.redirect("/import")
}

// ConnectorTest проверяет подключение к источнику (без загрузки данных).
func (c *ImportController) ConnectorTest() {
	code := c.Ctx.Input.Param(":code")
	c.must(pipeline.TestConnector(c.ctx(), code))
	c.ok("Подключение к источнику успешно.")
	c.Ctx.WriteString("")
	c.StopRun()
}

// JobCancel отменяет ожидающее или выполняющееся задание.
func (c *ImportController) JobCancel() {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	c.must(worker.Cancel(c.ctx(), id))
	c.ok("Отмена запрошена.")
	c.redirect("/import")
}
