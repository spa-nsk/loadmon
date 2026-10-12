package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cmdb/internal/repo"
	"cmdb/internal/services/refadmin"
	"cmdb/internal/services/svcerr"
)

// RefAdminController — универсальный CRUD над справочниками ref.* (plan.md E1.8): один
// контроллер на все ~20 таблиц, поведение которых описывает refadmin.TableSpec.
type RefAdminController struct {
	BaseController
}

func (c *RefAdminController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *RefAdminController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

// spec достаёт TableSpec по сегменту :table или прерывает запрос 404.
func (c *RefAdminController) spec() refadmin.TableSpec {
	s, ok := refadmin.Spec(c.Ctx.Input.Param(":table"))
	if !ok {
		c.Ctx.Output.SetStatus(404)
		c.Data["Message"] = "Неизвестный справочник"
		c.page("error.tpl")
		c.StopRun()
	}
	return s
}

// fkOptions собирает варианты выпадающих списков для всех полей с внешним ключом — карта
// "имя поля" -> варианты, шаблон читает через {{index .FKOptions "имя_поля"}}.
func (c *RefAdminController) fkOptions(spec refadmin.TableSpec) map[string][]refadmin.FKOption {
	out := map[string][]refadmin.FKOption{}
	for _, f := range spec.Fields {
		if f.FK == nil {
			continue
		}
		opts, err := refadmin.FKOptions(c.ctx(), f)
		c.must(err)
		out[f.Name] = opts
	}
	return out
}

// formValues читает из формы значения всех неавтогенерируемых полей справочника.
func (c *RefAdminController) formValues(spec refadmin.TableSpec, includePK bool) map[string]string {
	_ = c.Ctx.Request.ParseForm()
	values := map[string]string{}
	for _, f := range spec.Fields {
		if f.Generated || (f.PK && !includePK) {
			continue
		}
		values[f.Name] = c.GetString(f.Name)
	}
	return values
}

// List — таблица строк справочника с простым поиском.
func (c *RefAdminController) List() {
	spec := c.spec()
	q := c.GetString("q")
	rows, err := refadmin.List(c.ctx(), spec, q)
	c.must(err)

	c.Data["Nav"] = "admin"
	c.Data["Spec"] = spec
	c.Data["Rows"] = rows
	c.Data["Q"] = q
	c.Data["Specs"] = refadmin.All
	if c.isHTMX() {
		c.fragment("admin/_ref_results.tpl")
		return
	}
	c.page("admin/ref_list.tpl")
}

// Index — список справочников (меню администрирования).
func (c *RefAdminController) Index() {
	c.Data["Nav"] = "admin"
	c.Data["Specs"] = refadmin.All
	c.page("admin/ref_index.tpl")
}

// New — форма создания записи.
func (c *RefAdminController) New() {
	spec := c.spec()
	c.Data["Nav"] = "admin"
	c.Data["Spec"] = spec
	c.Data["Values"] = map[string]string{}
	c.Data["FKOptions"] = c.fkOptions(spec)
	c.Data["IsCreate"] = true
	c.page("admin/ref_form.tpl")
}

// Create сохраняет новую запись. Ошибка показывается тостом (как везде в приложении, см.
// ci.Create) — форма не перерисовывается, введённые значения остаются в браузере как есть.
func (c *RefAdminController) Create() {
	spec := c.spec()
	values := c.formValues(spec, true)
	c.must(refadmin.Create(c.ctx(), spec, values, c.userID()))
	c.logEvent("ref_create", map[string]any{"table": spec.Table, "pk": values[spec.PKField().Name]})
	c.redirect("/admin/ref/" + spec.Slug)
}

// Edit — форма правки существующей записи; устаревшая ссылка — 404 (c.notFound), не тост.
func (c *RefAdminController) Edit() {
	spec := c.spec()
	pk := c.Ctx.Input.Param(":pk")
	row, err := refadmin.Get(c.ctx(), spec, pk)
	if errors.Is(err, svcerr.ErrNotFound) {
		c.notFoundPage("Запись не найдена")
		return
	}
	c.must(err)

	c.Data["Nav"] = "admin"
	c.Data["Spec"] = spec
	c.Data["Values"] = row
	c.Data["PK"] = pk
	c.Data["FKOptions"] = c.fkOptions(spec)
	c.Data["IsCreate"] = false
	c.page("admin/ref_form.tpl")
}

// Update сохраняет правку записи.
func (c *RefAdminController) Update() {
	spec := c.spec()
	pk := c.Ctx.Input.Param(":pk")
	values := c.formValues(spec, false)
	c.must(refadmin.Update(c.ctx(), spec, pk, values, c.userID()))
	c.logEvent("ref_update", map[string]any{"table": spec.Table, "pk": pk})
	c.redirect("/admin/ref/" + spec.Slug)
}

// Delete удаляет запись.
func (c *RefAdminController) Delete() {
	spec := c.spec()
	pk := c.Ctx.Input.Param(":pk")
	err := refadmin.Delete(c.ctx(), spec, pk)
	c.must(err)
	c.logEvent("ref_delete", map[string]any{"table": spec.Table, "pk": pk})
	c.redirect("/admin/ref/" + spec.Slug)
}

// Export — выгрузка справочника в CSV.
func (c *RefAdminController) Export() {
	spec := c.spec()
	c.logEvent("ref_export", map[string]any{"table": spec.Table})
	c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, spec.Table))
	err := refadmin.ExportCSV(c.ctx(), spec, c.Ctx.ResponseWriter)
	c.must(err)
	c.StopRun()
}

// Import загружает CSV (та же колоночная форма, что у Export) и делает upsert построчно.
func (c *RefAdminController) Import() {
	spec := c.spec()
	file, _, err := c.Ctx.Request.FormFile("file")
	if err != nil {
		c.failText("Выберите CSV-файл.")
	}
	defer file.Close()

	res, err := refadmin.ImportCSV(c.ctx(), spec, file, c.userID())
	c.must(err)
	c.logEvent("ref_import", map[string]any{"table": spec.Table, "inserted": res.Inserted, "updated": res.Updated, "errors": len(res.Errors)})

	if len(res.Errors) == 0 {
		c.ok(fmt.Sprintf("Импортировано: добавлено %d, обновлено %d.", res.Inserted, res.Updated))
	} else {
		c.trigger(map[string]any{"toast": map[string]any{"kind": "warning",
			"text": fmt.Sprintf("Добавлено %d, обновлено %d, ошибок %d: %s", res.Inserted, res.Updated, len(res.Errors), strings.Join(res.Errors, "; "))}})
	}
	c.redirect("/admin/ref/" + spec.Slug)
}

// LocationTree — дерево размещения площадки до стойки/юнита (plan.md E3.8). Площадка ищется по
// location_code, а не через generic refadmin.Get (у ref.location PK — суррогатный id, а не код).
func (c *RefAdminController) LocationTree() {
	code := c.Ctx.Input.Param(":code")
	nodes, err := refadmin.LocationTree(c.ctx(), code)
	c.must(err)
	loc, err := repo.QueryRow(c.ctx(), `SELECT * FROM ref.location WHERE location_code = ?`, code)
	c.must(err)
	if loc == nil {
		c.Ctx.Output.SetStatus(404)
		c.Data["Message"] = "Площадка не найдена"
		c.page("error.tpl")
		c.StopRun()
	}

	c.Data["Nav"] = "admin"
	c.Data["LocationCode"] = code
	c.Data["Location"] = loc
	c.Data["Nodes"] = nodes
	c.page("admin/location_tree.tpl")
}
