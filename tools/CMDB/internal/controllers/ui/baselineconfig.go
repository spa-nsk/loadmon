package ui

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"cmdb/internal/services/baselineconfig"
	"cmdb/internal/services/refdata"
	"cmdb/internal/services/svcerr"
)

// BaselineConfigController — администрирование эталонных конфигураций (plan.md E7.8, право
// ref.write — та же настройка контроля, что остальной ref.*, не отдельное право).
type BaselineConfigController struct {
	BaseController
}

func (c *BaselineConfigController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *BaselineConfigController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

func (c *BaselineConfigController) id() int64 {
	id, _ := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	return id
}

func (c *BaselineConfigController) refLists() {
	var err error
	c.Data["Classes"], err = refdata.Classes(c.ctx())
	c.must(err)
	c.Data["Environments"], err = refdata.Environments(c.ctx())
	c.must(err)
	c.Data["Companies"], err = refdata.Companies(c.ctx())
	c.must(err)
	c.Data["Locations"], err = refdata.Locations(c.ctx())
	c.must(err)
	c.Data["SoftwareOptions"], err = refdata.Software(c.ctx())
	c.must(err)
}

// List — страница администрирования: список конфигураций и форма создания.
func (c *BaselineConfigController) List() {
	configs, err := baselineconfig.List(c.ctx(), false)
	c.must(err)
	c.Data["Nav"] = "admin"
	c.Data["Configs"] = configs
	c.refLists()
	c.page("admin/baseline_configs.tpl")
}

// New — страница создания (тот же приём, что RefAdminController.Edit/admin/ref_form.tpl — отдельная
// страница, не модальное окно: форма достаточно большая — класс, три измерения области, JSON
// атрибутов, список требуемого ПО).
func (c *BaselineConfigController) New() {
	c.Data["Nav"] = "admin"
	c.Data["IsCreate"] = true
	c.Data["Config"] = baselineconfig.Config{Enabled: true}
	c.Data["AttrsJSON"] = "{}"
	c.Data["SelectedSoftware"] = map[string]bool{}
	c.refLists()
	c.page("admin/baseline_config_form.tpl")
}

// Edit — страница правки существующей конфигурации; устаревшая ссылка — 404 (c.notFound), не тост.
func (c *BaselineConfigController) Edit() {
	cfg, err := baselineconfig.Get(c.ctx(), c.id())
	if errors.Is(err, svcerr.ErrNotFound) {
		c.notFoundPage("Конфигурация не найдена")
		return
	}
	c.must(err)
	c.Data["Nav"] = "admin"
	c.Data["IsCreate"] = false
	c.Data["Config"] = cfg
	attrsJSON, err := json.MarshalIndent(cfg.Attrs, "", "  ")
	c.must(err)
	c.Data["AttrsJSON"] = string(attrsJSON)
	selected := map[string]bool{}
	for _, sw := range cfg.RequiredSoftware {
		selected[strconv.FormatInt(sw.SoftwareID, 10)] = true
	}
	c.Data["SelectedSoftware"] = selected
	c.refLists()
	c.page("admin/baseline_config_form.tpl")
}

// input читает форму в baselineconfig.Input; ошибка разбора JSON атрибутов — понятная, не «invalid character».
func (c *BaselineConfigController) input() baselineconfig.Input {
	in := baselineconfig.Input{
		Name: strings.TrimSpace(c.GetString("name")), Description: strings.TrimSpace(c.GetString("description")),
		CIClass: c.GetString("ci_class"), Role: strings.TrimSpace(c.GetString("role")),
		EnvironmentCode: c.GetString("environment_code"), CompanyCode: c.GetString("company_code"), LocationCode: c.GetString("location_code"),
		Enabled: c.GetString("enabled") == "1", UserID: c.userID(),
	}
	raw := strings.TrimSpace(c.GetString("attrs_json"))
	if raw == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), &in.Attrs); err != nil {
		c.failText("Атрибуты: некорректный JSON — " + err.Error())
	}
	_ = c.Ctx.Request.ParseForm()
	for _, s := range c.Ctx.Request.PostForm["software_ids"] {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			in.RequiredSoftware = append(in.RequiredSoftware, id)
		}
	}
	return in
}

// Create создаёт конфигурацию.
func (c *BaselineConfigController) Create() {
	_, err := baselineconfig.Create(c.ctx(), c.input())
	c.must(err)
	c.logEvent("baseline_config_create", map[string]any{"name": c.GetString("name")})
	c.ok("Конфигурация создана.")
	c.redirect("/admin/baseline-configs")
}

// Update сохраняет правку.
func (c *BaselineConfigController) Update() {
	id := c.id()
	c.must(baselineconfig.Update(c.ctx(), id, c.input()))
	c.logEvent("baseline_config_update", map[string]any{"id": id})
	c.ok("Конфигурация сохранена.")
	c.redirect("/admin/baseline-configs")
}

// Delete удаляет конфигурацию.
func (c *BaselineConfigController) Delete() {
	id := c.id()
	c.must(baselineconfig.Delete(c.ctx(), id))
	c.logEvent("baseline_config_delete", map[string]any{"id": id})
	c.ok("Конфигурация удалена.")
	c.redirect("/admin/baseline-configs")
}
