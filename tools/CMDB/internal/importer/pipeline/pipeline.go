// Package pipeline прогоняет снимок источника через весь конвейер импорта (plan.md E4):
// проверка имени (E4.5) -> идентификация (E4.6) -> согласование с учётом блокировок (E4.7) ->
// применение через ChangeContext (E4.8). Источник — файл CSV/XLSX (Run) или настроенный
// API-коннектор (RunConnector, internal/importer/connector); конвейер от вида источника не зависит.
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime/debug"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	importcsv "cmdb/internal/importer/connectors/csv"
	"cmdb/internal/platform/logging"
	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/svcerr"
)

// notDiscoveredAfterMisses — сколько прогонов подряд источник должен не находить КЕ, прежде чем
// она помечается lifecycle_status='not_discovered' (E4.8). Не конфигурируется через UI —
// фиксированное консервативное значение (не сработает от одной случайно пропущенной записи).
const notDiscoveredAfterMisses = 3

// Summary — итог одного прогона импорта.
type Summary struct {
	RunID         int64
	DryRun        bool
	Total         int
	Created       int
	Updated       int
	Unchanged     int
	Conflicts     int
	Errors        int
	Skipped       int
	NotDiscovered int
	RelCreated    int
	RelClosed     int
	ChangeSetID   int64
}

// Options — параметры прогона.
type Options struct {
	DryRun bool
	UserID int64
	JobID  int64 // imp.job, из которого запущен прогон; 0 — ручной запуск вне очереди
}

// connectorRow — строка imp.connector.
type connectorRow struct {
	ID               int64
	Code             string
	Kind             string
	CIClass          string
	SourceSystemCode string
	TrustLevel       int // ref.source_system.trust_level источника (приоритет при согласовании)
	Settings         map[string]any
}

func loadConnector(ctx context.Context, code string) (connectorRow, error) {
	var cn connectorRow
	row, err := repo.QueryRow(ctx, `SELECT c.id::text AS id, c.kind, COALESCE(c.ci_class, '') AS ci_class,
			COALESCE(c.source_system_code, '') AS ssc, COALESCE(s.trust_level, 50)::text AS trust, c.config::text AS config
		FROM imp.connector c LEFT JOIN ref.source_system s ON s.source_system_code = c.source_system_code
		WHERE c.connector_code = ? AND c.enabled`, code)
	if err != nil {
		return cn, fmt.Errorf("pipeline: чтение коннектора: %w", err)
	}
	if row == nil {
		return cn, fmt.Errorf("%w: коннектор %q не найден или отключён", svcerr.ErrNotFound, code)
	}
	fmt.Sscan(row["id"], &cn.ID)
	cn.Code, cn.Kind, cn.CIClass, cn.SourceSystemCode = code, row["kind"], row["ci_class"], row["ssc"]
	fmt.Sscan(row["trust"], &cn.TrustLevel)
	if err := json.Unmarshal([]byte(row["config"]), &cn.Settings); err != nil {
		return cn, fmt.Errorf("pipeline: config коннектора %q: %w", code, err)
	}
	return cn, nil
}

func (cn connectorRow) config() connector.Config {
	return connector.Config{Code: cn.Code, Kind: cn.Kind, Settings: cn.Settings}
}

// Run загружает файл CSV/XLSX коннектора kind=csv и прогоняет его через конвейер.
func Run(ctx context.Context, connectorCode string, file io.Reader, format string, dryRun bool, userID int64) (Summary, error) {
	cn, err := loadConnector(ctx, connectorCode)
	if err != nil {
		return Summary{DryRun: dryRun}, err
	}
	if cn.Kind != "csv" {
		return Summary{DryRun: dryRun}, fmt.Errorf("%w: коннектор %q (%s) не принимает файлы", svcerr.ErrInvalidState, connectorCode, cn.Kind)
	}
	spec := ci.SpecFor(cn.CIClass)
	if spec == nil {
		return Summary{DryRun: dryRun}, fmt.Errorf("%w: для класса %q нет формы атрибутов — импорт не поддержан", svcerr.ErrInvalidState, cn.CIClass)
	}
	items, err := importcsv.Parse(file, format, *spec)
	if err != nil {
		return Summary{DryRun: dryRun}, err
	}
	return execute(ctx, cn, items, Options{DryRun: dryRun, UserID: userID}, connector.ModeFull)
}

// RunConnector строит снимок API-коннектора и прогоняет его через конвейер. Если снимок построить
// не удалось, прогон фиксируется как failed и НИЧЕГО не меняет — в том числе не увеличивает
// счётчики пропусков (harness/04, урок 3: сбой источника не должен «списывать» его КЕ).
// Паника коннектора превращается в ошибку прогона, а не роняет процесс (harness/05, п. 8).
func RunConnector(ctx context.Context, connectorCode string, opts Options) (sum Summary, err error) {
	cn, err := loadConnector(ctx, connectorCode)
	if err != nil {
		return Summary{DryRun: opts.DryRun}, err
	}
	conn, err := connector.New(cn.config())
	if err != nil {
		return Summary{DryRun: opts.DryRun}, err
	}

	items, extractErr := safeExtract(ctx, conn)
	if extractErr != nil {
		sum.DryRun = opts.DryRun
		runID, insErr := insertRun(ctx, cn.ID, opts, 0)
		if insErr == nil {
			sum.RunID = runID
			_ = finishRun(ctx, runID, "failed", sum, 0, extractErr.Error())
		}
		return sum, fmt.Errorf("pipeline: снимок коннектора %s: %w", connectorCode, extractErr)
	}
	return execute(ctx, cn, items, opts, connector.ModeOf(conn))
}

func safeExtract(ctx context.Context, conn connector.Connector) (items []canon.Item, err error) {
	defer func() {
		if r := recover(); r != nil {
			logging.FromContext(ctx).Error("connector_panic", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			err = fmt.Errorf("паника коннектора: %v", r)
		}
	}()
	return conn.Extract(ctx)
}

// TestConnector проверяет подключение API-коннектора (без загрузки данных).
func TestConnector(ctx context.Context, connectorCode string) error {
	cn, err := loadConnector(ctx, connectorCode)
	if err != nil {
		return err
	}
	conn, err := connector.New(cn.config())
	if err != nil {
		return err
	}
	return conn.Test(ctx)
}

func execute(ctx context.Context, cn connectorRow, items []canon.Item, opts Options, mode connector.Mode) (Summary, error) {
	sum := Summary{DryRun: opts.DryRun, Total: len(items)}

	runID, err := insertRun(ctx, cn.ID, opts, sum.Total)
	if err != nil {
		return sum, err
	}
	sum.RunID = runID

	for _, it := range items {
		var raw []byte
		if it.Raw != nil {
			raw, _ = json.Marshal(it.Raw)
		} else {
			raw, _ = json.Marshal(it.Record)
		}
		if _, err := repo.QueryRows(ctx, `INSERT INTO imp.raw_record (run_id, seq, raw) VALUES (?, ?, CAST(? AS jsonb))`,
			runID, it.Seq, string(raw)); err != nil {
			return sum, fmt.Errorf("pipeline: запись raw_record #%d: %w", it.Seq, err)
		}
	}

	env := &runEnv{connectorID: cn.ID, runID: runID, reg: naming.DefaultRegistry(), dryRun: opts.DryRun,
		userID: opts.UserID, sourceSystemCode: cn.SourceSystemCode, supplement: mode == connector.ModeSupplement, trustLevel: cn.TrustLevel,
		vendorKey: cmpOr(cn.SourceSystemCode, cn.Code), extTables: map[string]string{}, claimed: map[int64]string{}, snapshotIDs: map[string]bool{}}
	for _, it := range items {
		if it.Record.ExternalID != "" {
			env.snapshotIDs[it.Record.ExternalID] = true
		}
	}
	meta := changectx.Meta{Source: changectx.SourceImport, SourceSystemCode: cn.SourceSystemCode, ImportRunID: &runID,
		UserID: opts.UserID, Reason: fmt.Sprintf("импорт: коннектор %s, прогон %d", cn.Code, runID)}

	apply := func(ctx context.Context, q querier) error {
		var applied []appliedRecord
		for _, it := range items {
			out, ciID, procErr := processRowIsolated(ctx, q, env, it)
			if procErr != nil {
				return procErr
			}
			switch out {
			case outcomeCreated:
				sum.Created++
			case outcomeUpdated:
				sum.Updated++
			case outcomeUnchanged:
				sum.Unchanged++
			case outcomeConflict:
				sum.Conflicts++
			case outcomeError:
				sum.Errors++
			case outcomeSkipped:
				sum.Skipped++
			}
			if ciID != 0 && it.Record.ExternalID != "" {
				applied = append(applied, appliedRecord{ciID: ciID, rec: it.Record})
			}
		}
		if !opts.DryRun {
			created, closed, err := syncRelations(ctx, q, env, applied)
			if err != nil {
				return err
			}
			sum.RelCreated, sum.RelClosed = created, closed

			if !env.supplement {
				// Источник-дополнение не владеет КЕ: его отсутствие в снимке ничего не значит.
				linked, unlinked, err := linkDerived(ctx, q, env)
				if err != nil {
					return err
				}
				sum.RelCreated, sum.RelClosed = sum.RelCreated+linked, sum.RelClosed+unlinked

				n, err := markNotDiscovered(ctx, q, cn.ID, runID)
				if err != nil {
					return err
				}
				sum.NotDiscovered = n
			}
		}
		return nil
	}

	var changeSetID int64
	if opts.DryRun {
		// Только чтение — ChangeContext не нужен (нет записи в cmdb.*): orm.NewOrm() (не транзакция)
		// удовлетворяет тому же минимальному интерфейсу querier, что и orm.TxOrmer внутри changectx.Run.
		if err := apply(ctx, orm.NewOrm()); err != nil {
			_ = finishRun(ctx, runID, "failed", sum, 0, err.Error())
			return sum, err
		}
	} else {
		changeSetID, err = changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error { return apply(ctx, tx) })
		if err != nil {
			_ = finishRun(ctx, runID, "failed", sum, 0, err.Error())
			return sum, err
		}
	}
	sum.ChangeSetID = changeSetID

	if err := finishRun(ctx, runID, "succeeded", sum, changeSetID, ""); err != nil {
		return sum, err
	}
	return sum, nil
}

func insertRun(ctx context.Context, connectorID int64, opts Options, total int) (int64, error) {
	var runID int64
	err := orm.NewOrm().RawWithCtx(ctx, `
		INSERT INTO imp.run (connector_id, dry_run, user_id, total_records, job_id) VALUES (?, ?, ?, ?, nullif(?, 0)) RETURNING id
	`, connectorID, opts.DryRun, opts.UserID, total, opts.JobID).QueryRow(&runID)
	if err != nil {
		return 0, fmt.Errorf("pipeline: создание imp.run: %w", err)
	}
	return runID, nil
}

func finishRun(ctx context.Context, runID int64, status string, sum Summary, changeSetID int64, errMsg string) error {
	_, err := orm.NewOrm().RawWithCtx(ctx, `
		UPDATE imp.run SET status = ?, finished_at = now(), created_count = ?, updated_count = ?,
		       unchanged_count = ?, conflict_count = ?, error_count = ?, skipped_count = ?, not_discovered_count = ?,
		       rel_created_count = ?, rel_closed_count = ?,
		       change_set_id = nullif(?, 0), error_message = nullif(?, '')
		WHERE id = ?
	`, status, sum.Created, sum.Updated, sum.Unchanged, sum.Conflicts, sum.Errors, sum.Skipped, sum.NotDiscovered,
		sum.RelCreated, sum.RelClosed, changeSetID, errMsg, runID).Exec()
	if err != nil {
		return fmt.Errorf("pipeline: завершение imp.run: %w", err)
	}
	return nil
}

// markNotDiscovered увеличивает счётчик пропусков у всех связок этого коннектора, не подтверждённых
// в текущем прогоне, и переводит КЕ, перешагнувшие порог, в lifecycle_status='not_discovered'
// (E4.8) — КЕ не удаляется, только помечается. Вызывается только для успешно построенного снимка.
// Возвращает число КЕ, помеченных в этом прогоне.
func markNotDiscovered(ctx context.Context, tx querier, connectorID, runID int64) (int, error) {
	if _, err := tx.RawWithCtx(ctx, `
		UPDATE imp.match SET missed_runs = missed_runs + 1 WHERE connector_id = ? AND last_seen_run_id <> ?
	`, connectorID, runID).Exec(); err != nil {
		return 0, fmt.Errorf("pipeline: учёт пропусков: %w", err)
	}

	var rows []orm.Params
	if _, err := tx.RawWithCtx(ctx, `
		SELECT m.ci_id, c.row_version FROM imp.match m JOIN cmdb.ci c ON c.id = m.ci_id
		WHERE m.connector_id = ? AND m.missed_runs >= ? AND m.last_seen_run_id <> ?
		  AND c.lifecycle_stage NOT IN ('Retired', 'Disposed') AND COALESCE(c.lifecycle_status, '') <> 'not_discovered'
	`, connectorID, notDiscoveredAfterMisses, runID).Values(&rows); err != nil {
		return 0, fmt.Errorf("pipeline: поиск пропавших КЕ: %w", err)
	}

	for _, r := range rows {
		ciID := fmt.Sprint(r["ci_id"])
		if _, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET lifecycle_status = 'not_discovered', updated_at = now() WHERE id = ?`, ciID).Exec(); err != nil {
			return 0, fmt.Errorf("pipeline: пометка not_discovered id=%s: %w", ciID, err)
		}
		if _, err := tx.RawWithCtx(ctx, `
			INSERT INTO imp.conflict (run_id, ci_id, kind, details) VALUES (?, ?, 'not_discovered', CAST(? AS jsonb))
		`, runID, ciID, fmt.Sprintf(`{"missed_runs": %d}`, notDiscoveredAfterMisses)).Exec(); err != nil {
			return 0, fmt.Errorf("pipeline: конфликт not_discovered id=%s: %w", ciID, err)
		}
	}
	return len(rows), nil
}

// TemplateCSV — CSV-шаблон для класса (E4.9), используется контроллером для скачивания.
func TemplateCSV(ciClass string) ([]byte, error) {
	spec := ci.SpecFor(ciClass)
	if spec == nil {
		return nil, fmt.Errorf("%w: нет формы для класса %q", svcerr.ErrInvalidState, ciClass)
	}
	return importcsv.Template(*spec), nil
}

// DetectFormat определяет формат файла по расширению имени.
func DetectFormat(filename string) string {
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		return "xlsx"
	}
	return "csv"
}

// querier — общая часть orm.Ormer и orm.TxOrmer (тот же приём, что internal/services/ci.rawQuerier):
// позволяет processRow/markNotDiscovered работать и в транзакции применения, и в режиме
// предпросмотра (dry-run, только чтение, без ChangeContext) одним и тем же кодом.
type querier interface {
	RawWithCtx(ctx context.Context, query string, args ...any) orm.RawSeter
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
