// E4.2: очередь заданий и исполнитель на реальном PostgreSQL. Исполнение заданий подменяется
// (Worker.Run), кроме теста «настоящего» прогона zVirt через mock-сервер.
package worker_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "cmdb/internal/importer/connectors/all"
	"cmdb/internal/importer/connectors/zvirt/zvirtmock"
	"cmdb/internal/importer/worker"
	"cmdb/internal/testutil"
)

func newConnector(t *testing.T, db *sql.DB, cfg map[string]any) string {
	t.Helper()
	return newConnectorWith(t, db, cfg, "")
}

// newConnectorWith — коннектор с дополнительными колонками (", max_attempts", 2 и т.п.).
func newConnectorWith(t *testing.T, db *sql.DB, cfg map[string]any, extraCols string, extraArgs ...any) string {
	t.Helper()
	code := "wk-" + testutil.Suffix()
	b, _ := json.Marshal(cfg)
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('zvirt', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	q := `INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config` + extraCols + `) VALUES ($1, 'zvirt', 'Тест очереди', 'zvirt', $2::jsonb`
	args := []any{code, string(b)}
	for i, a := range extraArgs {
		q += ", $" + string(rune('3'+i))
		args = append(args, a)
	}
	if _, err := db.Exec(q+")", args...); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

type jobRow struct {
	Status   string
	Attempts int
	Error    string
	RunAfter time.Time
}

func job(t *testing.T, db *sql.DB, id int64) jobRow {
	t.Helper()
	var j jobRow
	var e sql.NullString
	if err := db.QueryRow(`SELECT status, attempts, error, run_after FROM imp.job WHERE id = $1`, id).Scan(&j.Status, &j.Attempts, &e, &j.RunAfter); err != nil {
		t.Fatalf("imp.job %d: %v", id, err)
	}
	j.Error = e.String
	return j
}

func fastWorker(run worker.RunFunc) *worker.Worker {
	return &worker.Worker{ID: "test-" + testutil.Suffix(), Run: run, Heartbeat: 30 * time.Millisecond, BackoffBase: 80 * time.Millisecond}
}

func TestEnqueue_DedupesAndRejectsCSV(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newConnector(t, db, map[string]any{})

	id1, created1, err := worker.Enqueue(ctx, code, "manual", false, 1)
	if err != nil || !created1 {
		t.Fatalf("первая постановка: id=%d created=%v err=%v", id1, created1, err)
	}
	id2, created2, err := worker.Enqueue(ctx, code, "manual", false, 1)
	if err != nil || created2 || id2 != id1 {
		t.Fatalf("повторная постановка: id=%d created=%v err=%v, want тот же id и created=false", id2, created2, err)
	}
	if _, _, err := worker.Enqueue(ctx, "csv-server", "manual", false, 1); err == nil {
		t.Error("постановка CSV-коннектора в очередь должна быть отклонена")
	}
}

func TestTick_RealZvirtRunLinksRunToJob(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	tag := fmt.Sprintf("%010x", rand.Int63n(1<<40)) // hex: идентификаторы ВМ — uuid
	m := zvirtmock.Default(tag)
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("CMDB_ZVIRT_TEST_PW", m.Password)
	code := newConnector(t, db, map[string]any{"base_url": srv.URL, "username": m.User, "password_env": "CMDB_ZVIRT_TEST_PW"})

	id, _, err := worker.Enqueue(ctx, code, "manual", false, 1)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	w := fastWorker(nil)
	if _, err := w.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if j := job(t, db, id); j.Status != "succeeded" || j.Attempts != 1 {
		t.Fatalf("задание = %+v, want succeeded с 1 попыткой", j)
	}
	var created int
	if err := db.QueryRow(`SELECT created_count FROM imp.run WHERE job_id = $1`, id).Scan(&created); err != nil || created != 8 {
		t.Fatalf("imp.run.created_count по job_id = %d (err=%v), want 8", created, err)
	}
}

func TestTick_RetryWithBackoffThenFail(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newConnectorWith(t, db, map[string]any{}, ", max_attempts", 2)
	id, _, _ := worker.Enqueue(ctx, code, "manual", false, 1)

	var calls atomic.Int32
	w := fastWorker(func(ctx context.Context, j worker.Job) error {
		if j.ID == id {
			calls.Add(1)
			return errors.New("источник недоступен")
		}
		return nil
	})

	if _, err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	j := job(t, db, id)
	if j.Status != "queued" || j.Attempts != 1 || !strings.Contains(j.Error, "недоступен") || !j.RunAfter.After(time.Now().Add(-time.Second)) {
		t.Fatalf("после 1-й неудачи = %+v, want queued с паузой и текстом ошибки", j)
	}
	if _, err := w.Tick(ctx); err != nil { // пауза ещё не прошла
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("задание выполнено %d раз до окончания паузы, want 1", calls.Load())
	}

	time.Sleep(150 * time.Millisecond)
	if _, err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if j := job(t, db, id); j.Status != "failed" || j.Attempts != 2 {
		t.Fatalf("после исчерпания попыток = %+v, want failed, attempts=2", j)
	}
	if calls.Load() != 2 {
		t.Fatalf("вызовов = %d, want 2", calls.Load())
	}
}

func TestTick_PanicBecomesFailure(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newConnectorWith(t, db, map[string]any{}, ", max_attempts", 1)
	id, _, _ := worker.Enqueue(ctx, code, "manual", false, 1)

	w := fastWorker(func(ctx context.Context, j worker.Job) error {
		if j.ID == id {
			panic("сломанный SDK")
		}
		return nil
	})
	if _, err := w.Tick(ctx); err != nil {
		t.Fatalf("Tick не должен падать из-за паники задания: %v", err)
	}
	if j := job(t, db, id); j.Status != "failed" || !strings.Contains(j.Error, "паника") {
		t.Fatalf("задание = %+v, want failed с текстом паники", j)
	}
}

// Два исполнителя на одну очередь: каждое задание выполняется ровно один раз (SKIP LOCKED).
func TestTick_ConcurrentWorkersRunEachJobOnce(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	ids := map[int64]bool{}
	for i := 0; i < 12; i++ {
		code := newConnector(t, db, map[string]any{})
		id, _, err := worker.Enqueue(ctx, code, "manual", false, 1)
		if err != nil {
			t.Fatal(err)
		}
		ids[id] = true
	}

	var mu sync.Mutex
	runs := map[int64]int{}
	run := func(ctx context.Context, j worker.Job) error {
		mu.Lock()
		runs[j.ID]++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		return nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := fastWorker(run).Tick(ctx); err != nil {
				t.Errorf("Tick: %v", err)
			}
		}()
	}
	wg.Wait()

	for id := range ids {
		if runs[id] != 1 {
			t.Errorf("задание %d выполнено %d раз, want 1", id, runs[id])
		}
	}
}

func TestTick_SchedulerEnqueuesByInterval(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newConnectorWith(t, db, map[string]any{}, ", schedule_interval_sec", 3600)

	var mu sync.Mutex
	var jobs []int64
	var trigger string
	w := fastWorker(func(ctx context.Context, j worker.Job) error {
		if j.ConnectorCode == code {
			mu.Lock()
			jobs = append(jobs, j.ID)
			mu.Unlock()
		}
		return nil
	})
	if _, err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("плановых заданий = %d, want 1", len(jobs))
	}
	db.QueryRow(`SELECT trigger FROM imp.job WHERE id = $1`, jobs[0]).Scan(&trigger)
	if trigger != "schedule" {
		t.Errorf("trigger = %q, want schedule", trigger)
	}
	if _, err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 {
		t.Fatalf("после 2-го прохода заданий = %d, want 1 (интервал 1 ч ещё не истёк)", len(jobs))
	}

	// интервал «истёк»: состарить последнее задание
	if _, err := db.Exec(`UPDATE imp.job SET created_at = now() - interval '2 hours', finished_at = now() - interval '2 hours' WHERE id = $1`, jobs[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Fatalf("после истечения интервала заданий = %d, want 2", len(jobs))
	}
}

func TestCancel_QueuedAndRunning(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()

	// ожидающее — отменяется сразу и не выполняется
	code1 := newConnector(t, db, map[string]any{})
	id1, _, _ := worker.Enqueue(ctx, code1, "manual", false, 1)
	if err := worker.Cancel(ctx, id1); err != nil {
		t.Fatalf("Cancel ожидающего: %v", err)
	}
	if j := job(t, db, id1); j.Status != "cancelled" {
		t.Fatalf("статус = %q, want cancelled", j.Status)
	}
	if err := worker.Cancel(ctx, id1); err == nil {
		t.Error("повторная отмена завершённого задания должна вернуть ошибку")
	}

	// выполняющееся — исполнитель получает отмену контекста на ближайшем heartbeat
	code2 := newConnector(t, db, map[string]any{})
	id2, _, _ := worker.Enqueue(ctx, code2, "manual", false, 1)
	started := make(chan struct{})
	w := fastWorker(func(ctx context.Context, j worker.Job) error {
		if j.ID != id2 {
			return nil
		}
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	done := make(chan struct{})
	go func() { defer close(done); w.Tick(ctx) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("задание не стартовало")
	}
	if err := worker.Cancel(ctx, id2); err != nil {
		t.Fatalf("Cancel выполняющегося: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("исполнитель не остановил задание после отмены")
	}
	if j := job(t, db, id2); j.Status != "cancelled" {
		t.Fatalf("статус = %q, want cancelled", j.Status)
	}
}

func TestTick_ReapsStaleRunningJob(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newConnector(t, db, map[string]any{})
	var connID int64
	db.QueryRow(`SELECT id FROM imp.connector WHERE connector_code = $1`, code).Scan(&connID)
	var id int64
	if err := db.QueryRow(`INSERT INTO imp.job (connector_id, status, trigger, attempts, started_at, heartbeat_at, locked_by)
		VALUES ($1, 'running', 'schedule', 1, now() - interval '1 hour', now() - interval '1 hour', 'dead-worker') RETURNING id`, connID).Scan(&id); err != nil {
		t.Fatal(err)
	}

	var ran atomic.Bool
	w := fastWorker(func(ctx context.Context, j worker.Job) error {
		if j.ID == id {
			ran.Store(true)
		}
		return nil
	})
	if _, err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if !ran.Load() {
		t.Fatal("зависшее задание должно быть возвращено в очередь и выполнено")
	}
	if j := job(t, db, id); j.Status != "succeeded" || j.Attempts != 2 {
		t.Fatalf("задание = %+v, want succeeded со 2-й попытки", j)
	}
}

func TestTick_SkipsSourceWithRunningJob(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newConnector(t, db, map[string]any{})
	var connID int64
	db.QueryRow(`SELECT id FROM imp.connector WHERE connector_code = $1`, code).Scan(&connID)
	// живое выполняющееся задание (свежий heartbeat) + ожидающее того же источника
	if _, err := db.Exec(`INSERT INTO imp.job (connector_id, status, trigger, attempts, started_at, heartbeat_at, locked_by)
		VALUES ($1, 'running', 'schedule', 1, now(), now(), 'alive-worker')`, connID); err != nil {
		t.Fatal(err)
	}
	queued, _, err := worker.Enqueue(ctx, code, "manual", false, 1)
	if err != nil {
		t.Fatal(err)
	}

	var ran atomic.Bool
	w := fastWorker(func(ctx context.Context, j worker.Job) error {
		if j.ID == queued {
			ran.Store(true)
		}
		return nil
	})
	if _, err := w.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	if ran.Load() {
		t.Fatal("ожидающее задание не должно стартовать, пока у источника есть выполняющееся")
	}
	if j := job(t, db, queued); j.Status != "queued" {
		t.Fatalf("статус = %q, want queued", j.Status)
	}
}
