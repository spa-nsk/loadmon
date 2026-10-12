package api

import (
	rootapi "cmdb/api"
)

// OpenAPIController отдаёт спецификацию REST API (plan.md E7.3). Публичный маршрут (без
// аутентификации, internal/filters.isPublic) — это документация, а не данные.
type OpenAPIController struct {
	BaseController
}

// Spec — GET /api/v1/openapi.yaml.
func (c *OpenAPIController) Spec() {
	c.Ctx.Output.Header("Content-Type", "application/yaml; charset=utf-8")
	_ = c.Ctx.Output.Body([]byte(rootapi.OpenAPIYAML))
	c.StopRun()
}
