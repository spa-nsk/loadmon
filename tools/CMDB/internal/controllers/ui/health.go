// Package ui содержит веб-контроллеры серверного интерфейса CMDB.
package ui

import (
	"context"
	"time"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/repo"
)

// HealthController отвечает на /healthz: проверяет доступность процесса и соединения с БД.
type HealthController struct {
	web.Controller
}

// Get возвращает 200 и статус БД, либо 503, если PostgreSQL недоступен.
func (c *HealthController) Get() {
	ctx, cancel := context.WithTimeout(c.Ctx.Request.Context(), 2*time.Second)
	defer cancel()

	dbOK := repo.Ping(ctx) == nil

	status := "ok"
	if !dbOK {
		status = "degraded"
		c.Ctx.Output.SetStatus(503)
	}

	c.Data["json"] = map[string]any{
		"status": status,
		"db":     dbOK,
	}
	_ = c.ServeJSON()
}
