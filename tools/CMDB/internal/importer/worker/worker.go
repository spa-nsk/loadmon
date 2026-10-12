package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/pipeline"
	"cmdb/internal/platform/logging"
	"cmdb/internal/services/alerting"
	"cmdb/internal/services/maintenance"
)

// RunFunc выполняет одно задание. По умолчанию — pipeline.RunConnector; тесты подменяют.
type RunFunc func(ctx context.Context, job Job) error

// Worker — планировщик и исполнитель заданий импорта.
type Worker struct {
	ID                        string
	Concurrency               int           // одновременных заданий (разных источников), по умолчанию 4
	PollEvery                 time.Duration // период опроса очереди, по умолчанию 5 с
	Heartbeat                 time.Duration // период heartbeat выполняющегося задания, по умолчанию 10 с
	StaleAfter                time.Duration // через сколько без heartbeat задание считается зависшим, по умолчанию 2 мин
	BackoffBase               time.Duration // пауза перед 2-й попыткой; далее удваивается (потолок 15 мин), по умолчанию 30 с
	PartitionMaintenanceEvery time.Duration // период создания будущих секций и ретеншна imp.raw_record (E8.6), по умолчанию 24 ч
	AlertCheckEvery           time.Duration // период проверки алертов — падение импорта, рост конфликтов (E8.6), по умолчанию 15 мин
	Run                       RunFunc

	lastPartitionMaintenance time.Time
	lastAlertCheck           time.Time
}

func (w *Worker) defaults() {
	if w.ID == "" {
		host, _ := os.Hostname()
		w.ID = fmt.Sprintf("%s/%d", host, os.Getpid())
	}
	if w.Concurrency <= 0 {
		w.Concurrency = 4
	}
	if w.PollEvery <= 0 {
		w.PollEvery = 5 * time.Second
	}
	if w.Heartbeat <= 0 {
		w.Heartbeat = 10 * time.Second
	}
	if w.StaleAfter <= 0 {
		w.StaleAfter = 2 * time.Minute
	}
	if w.BackoffBase <= 0 {
		w.BackoffBase = 30 * time.Second
	}
	if w.PartitionMaintenanceEvery <= 0 {
		w.PartitionMaintenanceEvery = 24 * time.Hour
	}
	if w.AlertCheckEvery <= 0 {
		w.AlertCheckEvery = 15 * time.Minute
	}
	if w.Run == nil {
		w.Run = func(ctx context.Context, job Job) error {
			_, err := pipeline.RunConnector(ctx, job.ConnectorCode, pipeline.Options{DryRun: job.DryRun, UserID: job.UserID, JobID: job.ID})
			return err
		}
	}
}

// Loop работает до отмены ctx: раз в PollEvery ставит плановые задания, возвращает зависшие и
// раздаёт готовые исполнителям. При остановке ждёт завершения выполняющихся заданий (их контекст
// отменяется, задание возвращается в очередь без расхода попытки).
func (w *Worker) Loop(ctx context.Context) {
	w.defaults()
	log := logging.FromContext(ctx)
	sem := make(chan struct{}, w.Concurrency)
	var wg sync.WaitGroup

	ticker := time.NewTicker(w.PollEvery)
	defer ticker.Stop()
	for {
		w.housekeeping(ctx)
	fill:
		for {
			select {
			case sem <- struct{}{}:
			default:
				break fill // все слоты заняты
			}
			job, err := claim(ctx, w.ID)
			if err != nil || job == nil {
				<-sem
				if err != nil && ctx.Err() == nil {
					log.Error("worker_claim_failed", "error", err)
				}
				break fill
			}
			wg.Add(1)
			go func() {
				defer func() { <-sem; wg.Done() }()
				w.execute(ctx, *job)
			}()
		}

		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
		}
	}
}

// Tick — один синхронный проход для тестов: планировщик, возврат зависших, выполнение всех
// готовых заданий. Возвращает число выполненных.
func (w *Worker) Tick(ctx context.Context) (int, error) {
	w.defaults()
	w.housekeeping(ctx)
	n := 0
	for {
		job, err := claim(ctx, w.ID)
		if err != nil {
			return n, err
		}
		if job == nil {
			return n, nil
		}
		w.execute(ctx, *job)
		n++
	}
}

func (w *Worker) housekeeping(ctx context.Context) {
	log := logging.FromContext(ctx)
	if n, err := reapStale(ctx, int(w.StaleAfter.Seconds())); err != nil {
		log.Error("worker_reap_failed", "error", err)
	} else if n > 0 {
		log.Warn("worker_reaped_stale_runs", "count", n)
	}
	if n, err := schedule(ctx); err != nil {
		log.Error("worker_schedule_failed", "error", err)
	} else if n > 0 {
		log.Info("worker_scheduled", "jobs", n)
	}
	if time.Since(w.lastPartitionMaintenance) >= w.PartitionMaintenanceEvery {
		w.lastPartitionMaintenance = time.Now()
		if res, err := maintenance.Run(ctx); err != nil {
			log.Error("worker_partition_maintenance_failed", "error", err)
		} else if res.PartitionsCreated > 0 || res.PartitionsDropped > 0 {
			log.Info("worker_partition_maintenance", "created", res.PartitionsCreated, "dropped", res.PartitionsDropped)
		}
	}
	if time.Since(w.lastAlertCheck) >= w.AlertCheckEvery {
		w.lastAlertCheck = time.Now()
		findings, err := alerting.Check(ctx, alerting.DefaultThresholds())
		if err != nil {
			log.Error("worker_alert_check_failed", "error", err)
		} else if len(findings) > 0 {
			log.Warn("worker_alert_findings", "count", len(findings))
			if err := alerting.Notify(ctx, findings); err != nil {
				log.Error("worker_alert_notify_failed", "error", err)
			}
		}
	}
}

func (w *Worker) execute(parent context.Context, job Job) {
	log := logging.FromContext(parent).With("job_id", job.ID, "connector", job.ConnectorCode, "attempt", job.Attempts)
	log.Info("job_started")

	timeout := time.Duration(job.TimeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	// heartbeat + опрос флага отмены пользователем
	stopHB := make(chan struct{})
	var hbWG sync.WaitGroup
	hbWG.Add(1)
	go func() {
		defer hbWG.Done()
		t := time.NewTicker(w.Heartbeat)
		defer t.Stop()
		for {
			select {
			case <-stopHB:
				return
			case <-t.C:
				var requested bool
				err := orm.NewOrm().RawWithCtx(context.Background(),
					`UPDATE imp.job SET heartbeat_at = now() WHERE id = ? RETURNING cancel_requested`, job.ID).QueryRow(&requested)
				if err == nil && requested {
					cancel()
					return
				}
			}
		}
	}()

	err := w.safeRun(ctx, job)
	close(stopHB)
	hbWG.Wait()

	// Итог фиксируется независимо от отмены контекста задания.
	fin := context.Background()
	var cancelRequested bool
	_ = orm.NewOrm().RawWithCtx(fin, `SELECT cancel_requested FROM imp.job WHERE id = ?`, job.ID).QueryRow(&cancelRequested)

	switch {
	case err == nil:
		w.finish(fin, job, "succeeded", "")
		log.Info("job_succeeded")
	case cancelRequested:
		w.finish(fin, job, "cancelled", "отменено пользователем")
		log.Info("job_cancelled")
	case parent.Err() != nil:
		// остановка процесса: вернуть в очередь, не расходуя попытку
		w.requeue(fin, job, 0, true, "остановка исполнителя")
		log.Warn("job_interrupted_by_shutdown")
	case job.Attempts < job.MaxAttempts:
		delay := w.backoff(job.Attempts)
		w.requeue(fin, job, delay, false, err.Error())
		log.Warn("job_failed_will_retry", "error", err, "retry_in", delay.String())
	default:
		w.finish(fin, job, "failed", err.Error())
		log.Error("job_failed", "error", err)
	}
}

func (w *Worker) safeRun(ctx context.Context, job Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.FromContext(ctx).Error("job_panic", "job_id", job.ID, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			err = fmt.Errorf("паника при выполнении задания: %v", r)
		}
	}()
	err = w.Run(ctx, job)
	if err == nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = ctx.Err()
	}
	return err
}

// backoff — пауза перед следующей попыткой после attempts-й неудачной: base, 2*base, 4*base … до 15 мин.
func (w *Worker) backoff(attempts int) time.Duration {
	d := w.BackoffBase
	for i := 1; i < attempts && d < 15*time.Minute; i++ {
		d *= 2
	}
	return min(d, 15*time.Minute)
}

func (w *Worker) finish(ctx context.Context, job Job, status, errMsg string) {
	_, err := orm.NewOrm().RawWithCtx(ctx, `UPDATE imp.job SET status = ?, finished_at = now(), locked_by = NULL, error = nullif(?, '') WHERE id = ?`,
		status, errMsg, job.ID).Exec()
	if err != nil {
		logging.FromContext(ctx).Error("job_finish_failed", "job_id", job.ID, "error", err)
	}
}

// requeue возвращает задание в очередь через delay. Если у источника уже стоит другое ожидающее
// задание (unique job_one_queued), текущее завершается failed — очередь не дублируется.
func (w *Worker) requeue(ctx context.Context, job Job, delay time.Duration, refund bool, errMsg string) {
	_, err := orm.NewOrm().RawWithCtx(ctx, `
		UPDATE imp.job j SET
			status = CASE WHEN NOT EXISTS (SELECT 1 FROM imp.job q WHERE q.connector_id = j.connector_id AND q.status = 'queued')
			              THEN 'queued' ELSE 'failed' END,
			run_after = now() + make_interval(secs => ?), locked_by = NULL, error = nullif(?, ''),
			attempts = CASE WHEN ? THEN GREATEST(attempts - 1, 0) ELSE attempts END,
			finished_at = CASE WHEN NOT EXISTS (SELECT 1 FROM imp.job q WHERE q.connector_id = j.connector_id AND q.status = 'queued')
			                   THEN NULL ELSE now() END
		WHERE j.id = ?
	`, delay.Seconds(), errMsg, refund, job.ID).Exec()
	if err != nil {
		logging.FromContext(ctx).Error("job_requeue_failed", "job_id", job.ID, "error", err)
	}
}
