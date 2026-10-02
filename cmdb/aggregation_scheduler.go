package cmdb

import (
	"context"
	"log"
	"time"
)

// RunMetricAggregation - независимый планировщик суточной агрегации
// поминутных метрик (2026-10-02, по жалобе "агрегация метрик по СХД так и
// не работает, приходится запускать storagemetrics -aggregate <дата>"):
//
//   - storage_metric -> storage_metric_record (SaveStorageMetricRecord);
//   - server_metric  -> server_metric_record  (SaveServerMetricRecord,
//     Zabbix и SCOM).
//
// Раньше обе агрегации вызывались только внутри SaveMetricData - один раз
// в сутки в 23:59:59 и ПОСЛЕ сохранения кластеров/хостов/ВМ vSphere. Если
// vSphere-часть шла часами, зависала или процесс metrics_grpc
// перезапускался до её окончания, агрегация за день не выполнялась вовсе,
// и догнать её было нечем (поминутные данные СХД хранятся всего 3 дня).
//
// Теперь (горутина из metrics_grpc):
//   - при старте - догоняющая агрегация за последние aggregationCatchUpDays
//     дней (идемпотентно: обе функции делают upsert по сущность+дата);
//   - каждый час - агрегация за сегодня по текущий момент (суточные записи
//     для отчётов появляются в течение дня, а не только ночью);
//   - при первом запуске после смены суток - окончательная агрегация за
//     весь прошедший день.
func RunMetricAggregation(ctx context.Context) {
	now := time.Now().Local()
	for d := aggregationCatchUpDays; d >= 1; d-- {
		aggregateMetricsForDay(endOfDay(now.AddDate(0, 0, -d)))
	}
	aggregateMetricsForDay(now)
	lastDay := dayKey(now)

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().Local()
			if day := dayKey(now); day != lastDay {
				// сутки сменились - закрываем прошедший день целиком
				aggregateMetricsForDay(endOfDay(now.AddDate(0, 0, -1)))
				lastDay = day
			}
			aggregateMetricsForDay(now)
		}
	}
}

// aggregationCatchUpDays - за сколько прошедших дней агрегировать при старте:
// поминутные метрики СХД хранятся 3 дня (storageMetricRetention).
const aggregationCatchUpDays = 3

// aggregateMetricsForDay - агрегация за сутки, содержащие t, по момент t.
// Ошибки/паника одной агрегации не мешают другой.
func aggregateMetricsForDay(t time.Time) {
	safeAggregation("SaveStorageMetricRecord", func() { SaveStorageMetricRecord(t) })
	safeAggregation("SaveServerMetricRecord", func() { SaveServerMetricRecord(t) })
}

func safeAggregation(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("cmdb.RunMetricAggregation: ПАНИКА в %s: %v", name, r)
		}
	}()
	fn()
}

// endOfDay - последний момент суток, содержащих t (обе агрегации берут
// строки с CollectedAt < t, а CollectedAt усечён до минуты).
func endOfDay(t time.Time) time.Time {
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return start.Add(24*time.Hour - time.Second)
}

func dayKey(t time.Time) string { return t.Format("2006-01-02") }
