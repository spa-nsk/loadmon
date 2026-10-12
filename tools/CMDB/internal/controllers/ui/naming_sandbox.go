package ui

import (
	"strings"

	"cmdb/internal/services/naming"
)

// NamingController — песочница правил именования (plan.md E3.8): ввести имя → разбор по профилю
// CMDB-NC-2026 и, если не подходит, по legacy-правилам Guide 2019 (internal/services/naming).
// Ничего не пишет в БД — чистая обёртка над уже существующим naming.Validate.
type NamingController struct {
	BaseController
}

// Sandbox — страница с полем ввода имени.
func (c *NamingController) Sandbox() {
	c.Data["Nav"] = "naming"
	c.Data["Name"] = ""
	c.page("admin/naming_sandbox.tpl")
}

// Check разбирает введённое имя и возвращает фрагмент с результатом (статус/причина/предложение).
func (c *NamingController) Check() {
	name := strings.TrimSpace(c.GetString("name"))
	c.Data["Name"] = name
	c.Data["HasResult"] = name != ""
	if name != "" {
		c.Data["Result"] = naming.Validate(registry, name)
	}
	c.fragment("admin/_naming_result.tpl")
}
