// Package worker — очередь заданий и исполнитель импорта (plan.md E4.2): планировщик ставит
// задания API-коннекторам по расписанию, исполнитель забирает их через FOR UPDATE SKIP LOCKED,
// выполняет с тайм-аутом, повторяет с нарастающей паузой и не запускает два задания одного
// источника одновременно. Работает и в cmdb-worker, и в тестах (Tick).
package worker

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/svcerr"
)

// Job — задание, взятое исполнителем в работу.
type Job struct {
	ID            int64
	ConnectorID   int64
	ConnectorCode string
	DryRun        bool
	UserID        int64
	Attempts      int // включая текущую
	MaxAttempts   int
	TimeoutSec    int
}

// Enqueue ставит задание «запустить сейчас». Если у источника уже есть ожидающее задание, новое
// не создаётся и возвращается существующее (created=false): повторное нажатие кнопки не плодит очередь.
// CSV-коннектору задание не нужно — ему нужен файл, он запускается из UI загрузки.
func Enqueue(ctx context.Context, connectorCode, trigger string, dryRun bool, userID int64) (jobID int64, created bool, err error) {
	c, err := repo.QueryRow(ctx, `SELECT id::text AS id, kind FROM imp.connector WHERE connector_code = ? AND enabled`, connectorCode)
	if err != nil {
		return 0, false, err
	}
	if c == nil {
		return 0, false, fmt.Errorf("%w: коннектор %q не найден или отключён", svcerr.ErrNotFound, connectorCode)
	}
	if c["kind"] == "csv" {
		return 0, false, fmt.Errorf("%w: коннектор %q принимает только файл — используйте загрузку", svcerr.ErrInvalidState, connectorCode)
	}

	o := orm.NewOrm()
	var id int64
	err = o.RawWithCtx(ctx, `
		INSERT INTO imp.job (connector_id, trigger, dry_run, user_id, max_attempts)
		SELECT id, ?, ?, nullif(?, 0), max_attempts FROM imp.connector WHERE connector_code = ?
		ON CONFLICT (connector_id) WHERE status = 'queued' DO NOTHING
		RETURNING id
	`, trigger, dryRun, userID, connectorCode).QueryRow(&id)
	if err == nil {
		return id, true, nil
	}
	if err != orm.ErrNoRows {
		return 0, false, fmt.Errorf("worker: постановка задания: %w", err)
	}
	if err := o.RawWithCtx(ctx, `SELECT j.id FROM imp.job j JOIN imp.connector c ON c.id = j.connector_id
		WHERE c.connector_code = ? AND j.status = 'queued'`, connectorCode).QueryRow(&id); err != nil {
		return 0, false, fmt.Errorf("worker: чтение ожидающего задания: %w", err)
	}
	return id, false, nil
}

// Cancel отменяет задание: ожидающее — сразу, выполняющееся — запрашивает остановку (исполнитель
// увидит флаг на ближайшем heartbeat и отменит контекст прогона; транзакция откатится, данные не изменятся).
func Cancel(ctx context.Context, jobID int64) error {
	o := orm.NewOrm()
	res, err := o.RawWithCtx(ctx, `UPDATE imp.job SET status = 'cancelled', finished_at = now(), error = 'отменено пользователем'
		WHERE id = ? AND status = 'queued'`, jobID).Exec()
	if err != nil {
		return fmt.Errorf("worker: отмена задания: %w", err)
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	res, err = o.RawWithCtx(ctx, `UPDATE imp.job SET cancel_requested = true WHERE id = ? AND status = 'running'`, jobID).Exec()
	if err != nil {
		return fmt.Errorf("worker: запрос отмены задания: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: задание %d уже завершено или не существует", svcerr.ErrInvalidState, jobID)
	}
	return nil
}

// schedule ставит задания источникам, у которых истёк интервал расписания и нет ни ожидающего,
// ни выполняющегося задания. Возвращает число поставленных.
func schedule(ctx context.Context) (int, error) {
	res, err := orm.NewOrm().RawWithCtx(ctx, `
		INSERT INTO imp.job (connector_id, trigger, max_attempts)
		SELECT c.id, 'schedule', c.max_attempts FROM imp.connector c
		WHERE c.enabled AND c.kind <> 'csv' AND c.schedule_interval_sec IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM imp.job j WHERE j.connector_id = c.id AND j.status IN ('queued', 'running'))
		  AND COALESCE((SELECT max(COALESCE(j.finished_at, j.created_at)) FROM imp.job j WHERE j.connector_id = c.id), '-infinity'::timestamptz)
		      + make_interval(secs => c.schedule_interval_sec) <= now()
		ON CONFLICT (connector_id) WHERE status = 'queued' DO NOTHING
	`).Exec()
	if err != nil {
		return 0, fmt.Errorf("worker: планировщик: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// reapStale возвращает в очередь (или помечает failed, если попытки исчерпаны) задания, чей
// исполнитель перестал отправлять heartbeat — процесс упал посреди прогона. Незавершённые
// imp.run этих заданий помечаются failed.
func reapStale(ctx context.Context, staleAfterSec int) (int, error) {
	var rows []orm.Params
	_, err := orm.NewOrm().RawWithCtx(ctx, `
		WITH reaped AS (
			UPDATE imp.job j SET
				status = CASE WHEN j.attempts < j.max_attempts
				               AND NOT EXISTS (SELECT 1 FROM imp.job q WHERE q.connector_id = j.connector_id AND q.status = 'queued')
				              THEN 'queued' ELSE 'failed' END,
				error = 'исполнитель перестал отвечать', run_after = now(), locked_by = NULL,
				finished_at = CASE WHEN j.attempts < j.max_attempts THEN NULL ELSE now() END
			WHERE j.status = 'running' AND j.heartbeat_at < now() - make_interval(secs => ?)
			RETURNING j.id
		)
		UPDATE imp.run r SET status = 'failed', finished_at = now(), error_message = 'прогон прерван: исполнитель перестал отвечать'
		FROM reaped WHERE r.job_id = reaped.id AND r.status = 'running'
		RETURNING r.id
	`, staleAfterSec).Values(&rows)
	if err != nil {
		return 0, fmt.Errorf("worker: возврат зависших заданий: %w", err)
	}
	return len(rows), nil
}

// claim берёт следующее готовое задание (FOR UPDATE SKIP LOCKED: параллельные исполнители не
// берут одно и то же), помечает его running. Источник с выполняющимся заданием пропускается.
// nil — работы нет.
func claim(ctx context.Context, workerID string) (*Job, error) {
	var job *Job
	err := orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		var rows []orm.Params
		if _, err := tx.RawWithCtx(ctx, `
			SELECT j.id, j.connector_id, c.connector_code, j.dry_run, COALESCE(j.user_id, 0) AS user_id,
			       j.attempts, j.max_attempts, c.timeout_sec
			FROM imp.job j JOIN imp.connector c ON c.id = j.connector_id
			WHERE j.status = 'queued' AND j.run_after <= now() AND c.enabled
			  AND NOT EXISTS (SELECT 1 FROM imp.job r WHERE r.connector_id = j.connector_id AND r.status = 'running')
			ORDER BY j.run_after, j.id LIMIT 1 FOR UPDATE OF j SKIP LOCKED
		`).Values(&rows); err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		r := rows[0]
		j := &Job{ConnectorCode: fmt.Sprint(r["connector_code"]), DryRun: fmt.Sprint(r["dry_run"]) == "true"}
		fmt.Sscan(fmt.Sprint(r["id"]), &j.ID)
		fmt.Sscan(fmt.Sprint(r["connector_id"]), &j.ConnectorID)
		fmt.Sscan(fmt.Sprint(r["user_id"]), &j.UserID)
		fmt.Sscan(fmt.Sprint(r["attempts"]), &j.Attempts)
		fmt.Sscan(fmt.Sprint(r["max_attempts"]), &j.MaxAttempts)
		fmt.Sscan(fmt.Sprint(r["timeout_sec"]), &j.TimeoutSec)

		if _, err := tx.RawWithCtx(ctx, `
			UPDATE imp.job SET status = 'running', attempts = attempts + 1, started_at = now(), heartbeat_at = now(),
			       locked_by = ?, error = NULL WHERE id = ?
		`, workerID, j.ID).Exec(); err != nil {
			return err
		}
		j.Attempts++
		job = j
		return nil
	})
	if err != nil {
		if svcerr.ViolatesConstraint(err, "job_one_running") {
			return nil, nil // другой исполнитель успел взять задание этого источника
		}
		return nil, fmt.Errorf("worker: выбор задания: %w", err)
	}
	return job, nil
}
