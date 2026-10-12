// Package filters содержит сквозные beego-фильтры: request-id, метрики, (позже) auth/rbac/csrf/audit/asof.
package filters

import (
	"strconv"
	"time"

	beectx "github.com/beego/beego/v2/server/web/context"
	"github.com/google/uuid"

	"cmdb/internal/platform/logging"
	"cmdb/internal/platform/metrics"
)

const headerRequestID = "X-Request-Id"

// RequestID проставляет (или переиспользует входящий) идентификатор запроса в контекст и заголовок ответа.
func RequestID(ctx *beectx.Context) {
	id := ctx.Input.Header(headerRequestID)
	if id == "" {
		id = uuid.NewString()
	}
	ctx.Request = ctx.Request.WithContext(logging.WithRequestID(ctx.Request.Context(), id))
	ctx.Output.Header(headerRequestID, id)
	ctx.Input.SetData("start_time", time.Now())
}

// AccessLog пишет структурную запись о завершённом запросе и обновляет метрики Prometheus.
func AccessLog(ctx *beectx.Context) {
	start, _ := ctx.Input.GetData("start_time").(time.Time)
	if start.IsZero() {
		start = time.Now()
	}
	// Метка маршрута — шаблон (например /cis/:id), а не URL: иначе id в пути раздувают кардинальность метрик.
	route, _ := ctx.Input.GetData("RouterPattern").(string)
	if route == "" {
		route = "unmatched"
	}
	status := ctx.Output.Status
	logging.FromContext(ctx.Request.Context()).Info("http_request",
		"method", ctx.Input.Method(),
		"path", ctx.Input.URL(),
		"route", route,
		"status", status,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	metrics.ObserveHTTP(ctx.Input.Method(), route, statusLabel(status), start)
}

func statusLabel(status int) string {
	if status == 0 {
		return "200"
	}
	return strconv.Itoa(status)
}
