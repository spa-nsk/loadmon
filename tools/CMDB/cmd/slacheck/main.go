// cmd/slacheck — проверка нефункциональных требований intent.md §10 (plan.md E8.5) РЕАЛЬНЫМ кодом
// сервисов (internal/services/ci), не переписанными заново SQL-запросами — иначе проверялась бы
// не та производительность, с которой реально работает приложение. Запускать против БД, уже
// наполненной cmd/loadgen (CMDB_DB_DSN — та же переменная, что у остальных cmd/*).
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"cmdb/internal/platform/asof"
	"cmdb/internal/platform/config"
	"cmdb/internal/repo"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/svcerr"
)

const iterations = 20

func main() {
	config.ApplyEnvOverrides()
	if err := repo.Init(config.LoadDBConfig()); err != nil {
		log.Fatalf("repo.Init: %v", err)
	}
	ctx := context.Background()
	if err := repo.Ping(ctx); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}

	ids := sampleIDs(ctx, "vm", iterations)
	if len(ids) == 0 {
		log.Fatal("нет КЕ класса vm — сначала запустите cmd/loadgen")
	}

	fmt.Println("== cmdb/slacheck: проверка SLA (intent.md §10) на данных cmd/loadgen ==")
	check("Список 100 тыс. КЕ с фильтром (p95)", 1*time.Second, iterations, func(i int) error {
		_, err := ci.List(ctx, ci.ListFilter{Class: "vm", PerPage: 200, Page: 1, Sort: "name"})
		return err
	})
	check("Список без фильтра, весь каталог (p95)", 1*time.Second, iterations, func(i int) error {
		_, err := ci.List(ctx, ci.ListFilter{PerPage: 200, Page: 1, Sort: "name"})
		return err
	})
	check("Карточка КЕ (p95)", 500*time.Millisecond, iterations, func(i int) error {
		_, err := ci.Get(ctx, ids[i%len(ids)])
		return err
	})
	asofCtx := asof.WithTime(ctx, time.Now().AddDate(0, -6, 0))
	check("Список «на дату» по классу, 6 мес. назад (p95)", 3*time.Second, iterations, func(i int) error {
		_, err := ci.List(asofCtx, ci.ListFilter{Class: "vm", PerPage: 200, Page: 1, Sort: "name"})
		return err
	})
	// Синтетическая история cmd/loadgen не восстанавливает настоящий sys_period за прошлые даты
	// (см. cmd/loadgen/history.go) — ci_asof() на давнюю дату для сгенерированной сегодня КЕ
	// корректно вернёт «не найдена», это не сбой запроса, а верный ответ на некорректные для этой
	// даты данные; время выполнения по-прежнему валидно как замер SLA.
	check("Карточка «на дату», 6 мес. назад (p95)", 3*time.Second, iterations, func(i int) error {
		_, err := ci.Get(asofCtx, ids[i%len(ids)])
		if errors.Is(err, svcerr.ErrNotFound) {
			return nil
		}
		return err
	})
}

func sampleIDs(ctx context.Context, class string, n int) []int64 {
	rows, err := repo.QueryRows(ctx, `SELECT id FROM cmdb.ci WHERE ci_class = ? ORDER BY random() LIMIT ?`, class, n)
	if err != nil {
		log.Fatalf("sampleIDs: %v", err)
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		fmt.Sscan(r["id"], &ids[i])
	}
	return ids
}

func check(name string, target time.Duration, n int, fn func(i int) error) {
	durs := make([]time.Duration, n)
	for i := 0; i < n; i++ {
		t0 := time.Now()
		if err := fn(i); err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		durs[i] = time.Since(t0)
	}
	sort.Slice(durs, func(i, j int) bool { return durs[i] < durs[j] })
	p95 := durs[int(float64(n)*0.95)]
	status := "OK"
	if p95 > target {
		status = "ПРЕВЫШЕНО"
	}
	fmt.Printf("%-55s p95=%-10s (min=%s max=%s) цель <%s  [%s]\n", name, p95.Round(time.Millisecond),
		durs[0].Round(time.Millisecond), durs[n-1].Round(time.Millisecond), target, status)
}
