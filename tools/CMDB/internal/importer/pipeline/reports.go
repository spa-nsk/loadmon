// reports.go — чтение прогонов и конфликтов импорта для UI (E4.9/E4.10): контроллер
// internal/controllers/ui/import.go только вызывает эти функции, вся SQL-логика здесь —
// тот же приём, что internal/services/history для журнала изменений.
package pipeline

import (
	"context"
	"fmt"
	"strings"

	"cmdb/internal/repo"
)

// ConnectorRow — настроенный источник импорта (imp.connector) для страницы импорта.
type ConnectorRow struct {
	Code        string
	Name        string
	Kind        string
	CIClass     string // только у CSV-коннектора
	Enabled     bool
	ScheduleSec string // "" — без расписания
	LastJob     string // статус последнего задания ("" — заданий не было)
	LastJobAt   string
}

// ListConnectors возвращает включённые коннекторы с состоянием последнего задания.
func ListConnectors(ctx context.Context) ([]ConnectorRow, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT c.connector_code, c.name, c.kind, COALESCE(c.ci_class, '') AS ci_class, c.enabled,
		       COALESCE(c.schedule_interval_sec::text, '') AS sched,
		       COALESCE(lj.status, '') AS last_job,
		       COALESCE(to_char(COALESCE(lj.finished_at, lj.created_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), '') AS last_job_at
		FROM imp.connector c
		LEFT JOIN LATERAL (SELECT status, finished_at, created_at FROM imp.job j WHERE j.connector_id = c.id ORDER BY j.id DESC LIMIT 1) lj ON true
		WHERE c.enabled ORDER BY c.name, c.connector_code`)
	if err != nil {
		return nil, fmt.Errorf("pipeline: чтение коннекторов: %w", err)
	}
	out := make([]ConnectorRow, len(rows))
	for i, r := range rows {
		out[i] = ConnectorRow{Code: r["connector_code"], Name: r["name"], Kind: r["kind"], CIClass: r["ci_class"],
			Enabled: r["enabled"] == "true", ScheduleSec: r["sched"], LastJob: r["last_job"], LastJobAt: r["last_job_at"]}
	}
	return out, nil
}

// JobRow — задание очереди для списка на странице импорта.
type JobRow struct {
	ID         string
	Connector  string
	Trigger    string
	DryRun     bool
	Status     string
	Attempts   string
	Max        string
	RunAfter   string
	Error      string
	RunID      string
	Cancelable bool
}

// ListJobs возвращает последние задания очереди, от новых к старым.
func ListJobs(ctx context.Context, limit int) ([]JobRow, error) {
	if limit <= 0 {
		limit = 15
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT j.id::text AS id, c.name AS connector, j.trigger, j.dry_run, j.status, j.attempts::text AS attempts,
		       j.max_attempts::text AS max_attempts,
		       to_char(j.run_after AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS run_after,
		       COALESCE(j.error, '') AS error,
		       COALESCE((SELECT r.id::text FROM imp.run r WHERE r.job_id = j.id ORDER BY r.id DESC LIMIT 1), '') AS run_id
		FROM imp.job j JOIN imp.connector c ON c.id = j.connector_id ORDER BY j.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("pipeline: чтение заданий: %w", err)
	}
	out := make([]JobRow, len(rows))
	for i, r := range rows {
		out[i] = JobRow{ID: r["id"], Connector: r["connector"], Trigger: r["trigger"], DryRun: r["dry_run"] == "true",
			Status: r["status"], Attempts: r["attempts"], Max: r["max_attempts"], RunAfter: r["run_after"],
			Error: r["error"], RunID: r["run_id"], Cancelable: r["status"] == "queued" || r["status"] == "running"}
	}
	return out, nil
}

const runCols = `r.id::text AS id, c.connector_code, c.name AS connector_name, r.status, r.dry_run,
	to_char(r.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS started_at,
	COALESCE(to_char(r.finished_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), '') AS finished_at,
	r.total_records, r.created_count, r.updated_count, r.unchanged_count, r.conflict_count,
	r.error_count, r.skipped_count, r.not_discovered_count, r.rel_created_count, r.rel_closed_count, COALESCE(r.change_set_id::text, '') AS change_set_id,
	COALESCE(r.error_message, '') AS error_message`

// RunRow — одна строка списка/карточки прогона импорта.
type RunRow struct {
	ID            string
	ConnectorCode string
	ConnectorName string
	Status        string
	DryRun        bool
	StartedAt     string
	FinishedAt    string
	Total         string
	Created       string
	Updated       string
	Unchanged     string
	Conflicts     string
	Errors        string
	Skipped       string
	NotDiscovered string
	RelCreated    string
	RelClosed     string
	ChangeSetID   string
	ErrorMessage  string
}

func rowToRun(r repo.Row) RunRow {
	return RunRow{
		ID: r["id"], ConnectorCode: r["connector_code"], ConnectorName: r["connector_name"],
		Status: r["status"], DryRun: r["dry_run"] == "true", StartedAt: r["started_at"], FinishedAt: r["finished_at"],
		Total: r["total_records"], Created: r["created_count"], Updated: r["updated_count"],
		Unchanged: r["unchanged_count"], Conflicts: r["conflict_count"], Errors: r["error_count"], Skipped: r["skipped_count"],
		NotDiscovered: r["not_discovered_count"], RelCreated: r["rel_created_count"], RelClosed: r["rel_closed_count"], ChangeSetID: r["change_set_id"], ErrorMessage: r["error_message"],
	}
}

// ListRuns возвращает последние прогоны всех коннекторов, от новых к старым (история на странице импорта).
func ListRuns(ctx context.Context, limit int) ([]RunRow, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT `+runCols+`
		FROM imp.run r JOIN imp.connector c ON c.id = r.connector_id
		ORDER BY r.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("pipeline: чтение прогонов: %w", err)
	}
	out := make([]RunRow, len(rows))
	for i, r := range rows {
		out[i] = rowToRun(r)
	}
	return out, nil
}

// GetRun возвращает один прогон (карточка результата после загрузки файла).
func GetRun(ctx context.Context, runID int64) (RunRow, bool, error) {
	row, err := repo.QueryRow(ctx, `
		SELECT `+runCols+`
		FROM imp.run r JOIN imp.connector c ON c.id = r.connector_id WHERE r.id = ?`, runID)
	if err != nil {
		return RunRow{}, false, fmt.Errorf("pipeline: чтение прогона %d: %w", runID, err)
	}
	if row == nil {
		return RunRow{}, false, nil
	}
	return rowToRun(row), true, nil
}

// ConflictFilter — фильтр очереди конфликтов (E4.10). Status по умолчанию "open" — закрытые
// конфликты не мешают разбирать текущие; "" — все статусы.
type ConflictFilter struct {
	Status  string
	Kind    string
	RunID   int64
	Page    int
	PerPage int
}

// ConflictRow — одна строка очереди конфликтов, с именем КЕ, если конфликт с ней связан.
type ConflictRow struct {
	ID        string
	RunID     string
	CIID      string
	CIName    string
	Kind      string
	Details   string
	Status    string
	CreatedAt string
}

// ConflictResult — страница очереди конфликтов.
type ConflictResult struct {
	Rows    []ConflictRow
	Total   int
	Page    int
	PerPage int
	Pages   int
}

// ListConflicts возвращает страницу конфликтов по фильтру, от новых к старым. Разрешение
// конфликтов (привязать/создать/игнорировать) в этот MVP не входит — только просмотр очереди.
func ListConflicts(ctx context.Context, f ConflictFilter) (ConflictResult, error) {
	var conds []string
	var args []any
	add := func(cond string, a ...any) {
		conds = append(conds, cond)
		args = append(args, a...)
	}
	if f.Status != "" {
		add("k.status = ?", f.Status)
	}
	if f.Kind != "" {
		add("k.kind = ?", f.Kind)
	}
	if f.RunID != 0 {
		add("k.run_id = ?", f.RunID)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	if f.PerPage <= 0 {
		f.PerPage = 25
	}
	if f.Page < 1 {
		f.Page = 1
	}

	res := ConflictResult{Page: f.Page, PerPage: f.PerPage}
	total, err := repo.QueryRow(ctx, `SELECT count(*)::text AS n FROM imp.conflict k `+where, args...)
	if err != nil {
		return res, err
	}
	fmt.Sscan(total["n"], &res.Total)
	res.Pages = (res.Total + f.PerPage - 1) / f.PerPage
	if res.Pages == 0 {
		res.Pages = 1
	}

	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	rows, err := repo.QueryRows(ctx, `
		SELECT k.id::text AS id, k.run_id::text AS run_id, COALESCE(k.ci_id::text, '') AS ci_id,
		       COALESCE(c.name, '') AS ci_name, k.kind, k.details::text AS details, k.status,
		       to_char(k.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS created_at
		FROM imp.conflict k LEFT JOIN cmdb.ci c ON c.id = k.ci_id `+where+`
		ORDER BY k.id DESC LIMIT ? OFFSET ?`, pageArgs...)
	if err != nil {
		return res, err
	}
	res.Rows = make([]ConflictRow, len(rows))
	for i, r := range rows {
		res.Rows[i] = ConflictRow{ID: r["id"], RunID: r["run_id"], CIID: r["ci_id"], CIName: r["ci_name"],
			Kind: r["kind"], Details: r["details"], Status: r["status"], CreatedAt: r["created_at"]}
	}
	return res, nil
}

// OpenConflictCount — счётчик открытых конфликтов для бейджа на странице импорта.
func OpenConflictCount(ctx context.Context) (int, error) {
	row, err := repo.QueryRow(ctx, `SELECT count(*)::text AS n FROM imp.conflict WHERE status = 'open'`)
	if err != nil {
		return 0, fmt.Errorf("pipeline: подсчёт открытых конфликтов: %w", err)
	}
	var n int
	fmt.Sscan(row["n"], &n)
	return n, nil
}
