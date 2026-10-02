package scommetrics

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"loadmon/config"
	"loadmon/models"
)

const (
	defaultClassName         = "Microsoft.Windows.Computer"
	defaultInterval          = 5 * time.Minute
	defaultInventoryInterval = time.Hour
	defaultPerfDuration      = 60
)

// RunAll запускает сбор по всем записям config.ConfigData.ConfigSCOM с
// Enabled=true (по горутине на запись) и ждёт их завершения (отмена ctx).
// Ошибка одной записи не останавливает остальные.
func RunAll(ctx context.Context) {
	var wg sync.WaitGroup
	started := 0
	for _, ep := range config.ConfigData.ConfigSCOM {
		if !ep.Enabled {
			continue
		}
		started++
		wg.Add(1)
		go func(ep config.ScomEndpoint) {
			defer wg.Done()
			if err := Run(ctx, ep); err != nil && err != context.Canceled {
				log.Printf("scommetrics[%s]: остановлен: %v", ep.Name, err)
			}
		}(ep)
	}
	if started == 0 {
		log.Println("scommetrics: в ConfigSCOM нет записей с Enabled=true - сбор из SCOM не запускается")
		return
	}
	wg.Wait()
}

// Run - цикл сбора по одной записи ConfigSCOM: инвентаризация сразу и
// далее каждые InventoryIntervalSec, метрики каждые IntervalSec.
func Run(ctx context.Context, ep config.ScomEndpoint) error {
	c, err := NewClient(ep)
	if err != nil {
		return err
	}
	className := strings.TrimSpace(ep.ClassName)
	if className == "" {
		className = defaultClassName
	}
	interval := time.Duration(ep.IntervalSec) * time.Second
	if interval <= 0 {
		interval = defaultInterval
	}
	invInterval := time.Duration(ep.InventoryIntervalSec) * time.Second
	if invInterval <= 0 {
		invInterval = defaultInventoryInterval
	}
	duration := ep.PerfDuration
	if duration <= 0 {
		duration = defaultPerfDuration
	}
	log.Printf("scommetrics[%s]: сбор из %s, класс %s, метрики каждые %s, инвентаризация каждые %s",
		ep.Name, ep.Endpoint, className, interval, invInterval)

	var (
		monitored []monitoredComputer
		lastInv   time.Time
	)
	cycle := func() {
		if lastInv.IsZero() || time.Since(lastInv) >= invInterval {
			m, err := inventory(ctx, c, className)
			if err != nil {
				log.Printf("scommetrics[%s]: инвентаризация: %v", ep.Name, err)
				if lastInv.IsZero() {
					return // ни одного успешного списка - собирать метрики не по чему
				}
			} else {
				monitored = m
				lastInv = time.Now()
				if f := c.AuthFormat(); f != "" {
					log.Printf("scommetrics[%s]: аутентификация принята в формате %s", ep.Name, f)
				}
			}
		}
		if err := collect(ctx, c, ep.Name, monitored, interval, duration); err != nil {
			log.Printf("scommetrics[%s]: сбор метрик: %v", ep.Name, err)
		}
	}

	cycle()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			cycle()
		}
	}
}

// collect снимает значения счётчиков по каждому компьютеру и пишет строки
// server_metric (Source = SCOM). CollectedAt - время опроса, усечённое до
// интервала, чтобы повтор в том же интервале заменял, а не дублировал строку.
func collect(ctx context.Context, c *Client, name string, monitored []monitoredComputer, interval time.Duration, duration int) error {
	if len(monitored) == 0 {
		return nil
	}
	at := time.Now().Truncate(interval)
	rows := make([]*models.ServerMetric, 0, len(monitored))
	failed := 0
	for i, m := range monitored {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		reqID := fmt.Sprintf("loadmon-%s-%d", m.objectID, i)
		values, err := c.Performance(ctx, reqID, duration, m.counters)
		if err != nil {
			failed++
			log.Printf("scommetrics[%s]: %s: %v", name, m.name, err)
			continue
		}
		if row := buildRow(at, m.hostID, m.name, values); row != nil {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return fmt.Errorf("нет значений ни по одному из %d компьютеров (ошибок запроса: %d)", len(monitored), failed)
	}
	if err := models.ReplaceMetrics(ctx, rows); err != nil {
		return fmt.Errorf("запись server_metric: %w", err)
	}
	log.Printf("scommetrics[%s]: записано %d снимков на %s (ошибок запроса: %d)", name, len(rows), at.Format(time.RFC3339), failed)
	return nil
}
