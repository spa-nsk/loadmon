// Package maintenance — фоновое обслуживание секций (plan.md E8.6): заблаговременное создание
// будущих секций `hist.ci`/`audit.change_log` (по месяцу) и `imp.raw_record` (по дню), и ретеншн
// `imp.raw_record` (DROP PARTITION вместо DELETE — секция целиком, без построчного удаления).
// Сами операции — SQL-функции `ops.ensure_partitions`/`ops.drop_old_partitions` (миграция
// 0027_partitioning.sql); этот пакет — тонкая Go-обёртка, вызываемая периодически из cmdb-worker
// (см. internal/importer/worker, поле Worker.PartitionMaintenanceEvery).
package maintenance

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// target — одна секционированная таблица под обслуживание.
type target struct {
	schema, table, column, unit string
	ahead                       int
}

// targets — те же три таблицы, что перевела на секции миграция 0027_partitioning.sql. hist.ci —
// самая объёмная история (intent.md §9), остальные таблицы-расширения секций не имеют (см.
// комментарий миграции) и в обслуживании не нуждаются.
var targets = []target{
	{schema: "hist", table: "ci", column: "hist_recorded_at", unit: "month", ahead: 2},
	{schema: "audit", table: "change_log", column: "at", unit: "month", ahead: 2},
	{schema: "imp", table: "raw_record", column: "created_at", unit: "day", ahead: 7},
}

// RawRecordRetentionDays — сколько дней хранить imp.raw_record (intent.md §9: «хранить 30–90
// дней»), по умолчанию середина диапазона; переопределяется CMDB_RAW_RECORD_RETENTION_DAYS.
func RawRecordRetentionDays() int {
	if v := os.Getenv("CMDB_RAW_RECORD_RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 60
}

// Result — что сделал один проход обслуживания.
type Result struct {
	PartitionsCreated int
	PartitionsDropped int
}

// Run создаёт отсутствующие будущие секции целевых таблиц и удаляет секции imp.raw_record старше
// RawRecordRetentionDays(). Ошибка одной таблицы не прерывает обслуживание остальных — частичный
// прогон лучше полного отказа: секции создаются идемпотентно, пропущенная в этот раз будет
// создана при следующем вызове.
func Run(ctx context.Context) (Result, error) {
	var res Result
	var firstErr error
	note := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for _, t := range targets {
		n, err := ensurePartitions(ctx, t.schema, t.table, t.column, t.unit, t.ahead)
		if err != nil {
			note(fmt.Errorf("ensure_partitions %s.%s: %w", t.schema, t.table, err))
			continue
		}
		res.PartitionsCreated += n
	}

	dropped, err := dropOldPartitions(ctx, "imp", "raw_record", "day", time.Duration(RawRecordRetentionDays())*24*time.Hour)
	if err != nil {
		note(fmt.Errorf("drop_old_partitions imp.raw_record: %w", err))
	} else {
		res.PartitionsDropped = dropped
	}

	return res, firstErr
}

func ensurePartitions(ctx context.Context, schema, table, column, unit string, ahead int) (int, error) {
	var n int
	err := orm.NewOrm().RawWithCtx(ctx, `SELECT ops.ensure_partitions(?, ?, ?, ?, ?)`, schema, table, column, unit, ahead).QueryRow(&n)
	return n, err
}

func dropOldPartitions(ctx context.Context, schema, table, unit string, olderThan time.Duration) (int, error) {
	var n int
	err := orm.NewOrm().RawWithCtx(ctx, `SELECT ops.drop_old_partitions(?, ?, ?, ?)`, schema, table, unit, fmt.Sprintf("%d seconds", int(olderThan.Seconds()))).QueryRow(&n)
	return n, err
}
