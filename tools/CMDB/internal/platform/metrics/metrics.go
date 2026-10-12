// Package metrics публикует метрики Prometheus на /metrics.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTPRequestsTotal считает HTTP-запросы по методу, маршруту и коду ответа.
var HTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "cmdb",
		Name:      "http_requests_total",
		Help:      "Количество HTTP-запросов cmdb-web.",
	},
	[]string{"method", "route", "status"},
)

// HTTPRequestDuration измеряет длительность обработки HTTP-запросов.
var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "cmdb",
		Name:      "http_request_duration_seconds",
		Help:      "Длительность обработки HTTP-запросов cmdb-web.",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "route"},
)

// ImportRunsTotal считает запуски коннекторов импорта по источнику и статусу.
var ImportRunsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "cmdb",
		Name:      "import_runs_total",
		Help:      "Количество запусков импорта cmdb-worker.",
	},
	[]string{"source_system", "status"},
)

// ImportConsecutiveFailures — длина серии подряд неудачных прогонов на источник (plan.md E8.6,
// «алерты — падение импорта»), обновляется internal/services/alerting.Check.
var ImportConsecutiveFailures = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "cmdb",
		Name:      "import_consecutive_failures",
		Help:      "Число подряд неудачных прогонов источника импорта (0 — последний прогон не failed).",
	},
	[]string{"connector"},
)

// OpenConflictsTotal — открытые конфликты импорта (plan.md E8.6, «алерты — рост конфликтов»),
// обновляется internal/services/alerting.Check.
var OpenConflictsTotal = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "cmdb",
		Name:      "import_open_conflicts",
		Help:      "Количество открытых конфликтов импорта (imp.conflict, status='open').",
	},
)

func init() {
	prometheus.MustRegister(HTTPRequestsTotal, HTTPRequestDuration, ImportRunsTotal, ImportConsecutiveFailures, OpenConflictsTotal)
}

// ObserveHTTP записывает метрики одного HTTP-запроса.
func ObserveHTTP(method, route, status string, start time.Time) {
	HTTPRequestsTotal.WithLabelValues(method, route, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
}
