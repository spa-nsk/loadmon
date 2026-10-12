package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/canon"
	"cmdb/internal/services/naming"
)

// outcome — как строка файла повлияла (или не повлияла) на данные — используется только для
// счётчиков imp.run, не хранится как отдельный тип в БД (там status у imp.norm_record).
type outcome int

const (
	outcomeCreated outcome = iota
	outcomeUpdated
	outcomeUnchanged
	outcomeConflict
	outcomeError
	outcomeSkipped
)

// processRowIsolated выполняет processRow в SAVEPOINT: нарушение ограничения БД данными одной записи
// (CHECK, уникальность, формат значения — SQLSTATE 22/23) откатывает только эту запись и
// фиксируется как ошибка записи + конфликт apply_error, а не роняет весь прогон (E4.11). Прочие
// ошибки (обрыв соединения и т.п.) по-прежнему прерывают прогон. Dry-run только читает — без savepoint.
func processRowIsolated(ctx context.Context, q querier, env *runEnv, r canon.Item) (outcome, int64, error) {
	if env.dryRun {
		return processRow(ctx, q, env, r)
	}
	if _, err := q.RawWithCtx(ctx, "SAVEPOINT import_record").Exec(); err != nil {
		return outcomeError, 0, fmt.Errorf("pipeline: savepoint: %w", err)
	}
	out, ciID, err := processRow(ctx, q, env, r)
	if err == nil {
		if _, rerr := q.RawWithCtx(ctx, "RELEASE SAVEPOINT import_record").Exec(); rerr != nil {
			return outcomeError, 0, fmt.Errorf("pipeline: release savepoint: %w", rerr)
		}
		return out, ciID, nil
	}
	if !isDataError(err) {
		return outcomeError, 0, err
	}
	if _, rerr := q.RawWithCtx(ctx, "ROLLBACK TO SAVEPOINT import_record").Exec(); rerr != nil {
		return outcomeError, 0, fmt.Errorf("pipeline: откат записи: %w (исходная ошибка: %v)", rerr, err)
	}
	normID, ierr := insertNormRecord(ctx, q, env.runID, r.Seq, r.Record, naming.Result{}, 0, "", "error", err.Error())
	if ierr != nil {
		return outcomeError, 0, ierr
	}
	if cerr := insertConflict(ctx, q, env.runID, &normID, nil, "apply_error",
		map[string]any{"error": err.Error(), "source_name": r.Record.SourceName, "external_id": r.Record.ExternalID}); cerr != nil {
		return outcomeError, 0, cerr
	}
	return outcomeError, 0, nil
}

// isDataError — ошибка вызвана данными записи (класс SQLSTATE 22 «data exception» или 23
// «integrity constraint violation»), а не отказом инфраструктуры.
func isDataError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "(SQLSTATE 22") || strings.Contains(msg, "(SQLSTATE 23")
}

// runEnv — параметры одного прогона, общие для всех записей снимка.
type runEnv struct {
	connectorID      int64
	runID            int64
	reg              *naming.Registry
	dryRun           bool
	userID           int64
	sourceSystemCode string
	trustLevel       int               // trust_level источника: чужие значения источников с большим доверием не перезаписываются
	supplement       bool              // коннектор только дополняет существующие КЕ (connector.ModeSupplement)
	vendorKey        string            // ключ вендорских сведений в cmdb.ci.attrs
	extTables        map[string]string // класс -> таблица-расширение ("" — у класса её нет)
	snapshotIDs      map[string]bool   // ExternalID всех записей снимка
	claimed          map[int64]string  // id КЕ -> ExternalID записи этого прогона, которая на неё указала
}

// appliedRecord — запись снимка, для которой в БД есть КЕ (нужна для разрешения связей).
type appliedRecord struct {
	ciID int64
	rec  canon.Record
}

// extTable возвращает таблицу-расширение класса из ref.ci_class (пусто — у класса нет расширения).
// Неизвестный класс — ошибка записи, а не прогона: другие записи снимка это не затрагивает.
func (e *runEnv) extTable(ctx context.Context, q querier, class string) (string, error) {
	if class == "" {
		return "", nil // источник-дополнение не знает класс КЕ
	}
	if t, ok := e.extTables[class]; ok {
		return t, nil
	}
	var rows []orm.Params
	if _, err := q.RawWithCtx(ctx, `SELECT COALESCE(ext_table, '') AS t FROM ref.ci_class WHERE class_code = ?`, class).Values(&rows); err != nil {
		return "", fmt.Errorf("pipeline: чтение класса %q: %w", class, err)
	}
	if len(rows) == 0 {
		return "", errUnknownClass
	}
	t := fmt.Sprint(rows[0]["t"])
	e.extTables[class] = t
	return t, nil
}

var errUnknownClass = errors.New("неизвестный класс КЕ")

// processRow — E4.5 (проверка имени) -> E4.6 (идентификация) -> E4.7 (согласование с учётом
// блокировок) -> E4.8 (применение, если не dry-run) для одной записи источника. Ошибка самой
// записи (Item.Err от коннектора, неизвестный класс, неоднозначная идентификация) не возвращается
// как error — это не ошибка ПРОГОНА, а зафиксированный исход записи (E4.11: одна плохая запись не
// должна прерывать импорт остальных). error возвращается только на сбой самого пайплайна (БД
// недоступна и т.п.), из-за которого продолжать прогон бессмысленно. Второе значение — id КЕ
// записи (0, если её нет: ошибка, конфликт или новая КЕ в dry-run).
func processRow(ctx context.Context, q querier, env *runEnv, r canon.Item) (outcome, int64, error) {
	if r.Err != nil {
		if r.Retain && r.Record.ExternalID != "" && !env.dryRun {
			if err := retainMatch(ctx, q, env, r.Record.ExternalID); err != nil {
				return outcomeError, 0, err
			}
		}
		if _, err := insertNormRecord(ctx, q, env.runID, r.Seq, canon.Record{Class: r.Record.Class, SourceName: r.Record.SourceName},
			naming.Result{}, 0, "", "error", r.Err.Error()); err != nil {
			return outcomeError, 0, err
		}
		return outcomeError, 0, nil
	}
	rec := r.Record
	namingRes := naming.Validate(env.reg, rec.SourceName)

	extTable, err := env.extTable(ctx, q, rec.Class)
	if errors.Is(err, errUnknownClass) {
		if _, err := insertNormRecord(ctx, q, env.runID, r.Seq, rec, namingRes, 0, "", "error",
			fmt.Sprintf("неизвестный класс КЕ %q", rec.Class)); err != nil {
			return outcomeError, 0, err
		}
		return outcomeError, 0, nil
	}
	if err != nil {
		return outcomeError, 0, err
	}

	ciID, method, ambiguous, err := identify(ctx, q, env, extTable, rec)
	if err != nil {
		return outcomeError, 0, err
	}
	if ambiguous {
		normID, err := insertNormRecord(ctx, q, env.runID, r.Seq, rec, namingRes, 0, "ambiguous", "conflict", "")
		if err != nil {
			return outcomeError, 0, err
		}
		if err := insertConflict(ctx, q, env.runID, &normID, nil, "ambiguous_match",
			map[string]any{"identifiers": rec.Identifiers, "external_id": rec.ExternalID, "source_name": rec.SourceName}); err != nil {
			return outcomeError, 0, err
		}
		return outcomeConflict, 0, nil
	}

	if ciID != 0 && rec.ExternalID != "" {
		env.claimed[ciID] = rec.ExternalID
	}

	if ciID == 0 && env.supplement {
		if _, err := insertNormRecord(ctx, q, env.runID, r.Seq, rec, namingRes, 0, "", "skipped",
			"в CMDB нет КЕ с таким именем — источник только дополняет существующие"); err != nil {
			return outcomeError, 0, err
		}
		return outcomeSkipped, 0, nil
	}

	if ciID == 0 {
		if env.dryRun {
			if _, err := insertNormRecord(ctx, q, env.runID, r.Seq, rec, namingRes, 0, "new", "pending", ""); err != nil {
				return outcomeError, 0, err
			}
			return outcomeCreated, 0, nil
		}
		newID, err := createCI(ctx, q, env, extTable, rec, namingRes)
		if err != nil {
			return outcomeError, 0, err
		}
		if _, err := syncExtras(ctx, q, env, newID, rec); err != nil {
			return outcomeError, 0, err
		}
		if err := touchMatch(ctx, q, env.connectorID, newID, primaryIdentifier(rec), "new", env.runID); err != nil {
			return outcomeError, 0, err
		}
		if _, err := insertNormRecord(ctx, q, env.runID, r.Seq, rec, namingRes, newID, "new", "applied", ""); err != nil {
			return outcomeError, 0, err
		}
		return outcomeCreated, newID, nil
	}

	changed, locked, err := applyToExisting(ctx, q, env, extTable, ciID, rec)
	if err != nil {
		return outcomeError, 0, err
	}
	extrasChanged, err := syncExtras(ctx, q, env, ciID, rec)
	if err != nil {
		return outcomeError, 0, err
	}
	changed = changed || extrasChanged
	if !env.dryRun {
		if err := touchMatch(ctx, q, env.connectorID, ciID, primaryIdentifier(rec), method, env.runID); err != nil {
			return outcomeError, 0, err
		}
		if err := touchLastSeen(ctx, q, ciID); err != nil {
			return outcomeError, 0, err
		}
	}
	status := "applied"
	if env.dryRun {
		status = "pending"
	}
	normID, err := insertNormRecord(ctx, q, env.runID, r.Seq, rec, namingRes, ciID, method, status, "")
	if err != nil {
		return outcomeError, 0, err
	}
	if len(locked) > 0 {
		if err := insertConflict(ctx, q, env.runID, &normID, &ciID, "locked_attr", map[string]any{"attrs": locked}); err != nil {
			return outcomeError, 0, err
		}
	}
	if !changed {
		return outcomeUnchanged, ciID, nil
	}
	return outcomeUpdated, ciID, nil
}

// primaryIdentifier — какой идентификатор используется как ключ imp.match: самый надёжный из
// присутствующих (тот же порядок приоритета, что identify).
func primaryIdentifier(rec canon.Record) string {
	if rec.ExternalID != "" {
		return "ext:" + rec.ExternalID
	}
	if v := rec.Identifiers["bios_uuid"]; v != "" {
		return "bios_uuid:" + v
	}
	if v := rec.Identifiers["serial_number"]; v != "" {
		return "serial_number:" + v
	}
	return "name_location:" + rec.SourceName + "|" + rec.LocationCode
}

func touchMatch(ctx context.Context, q querier, connectorID, ciID int64, externalKey, method string, runID int64) error {
	_, err := q.RawWithCtx(ctx, `
		INSERT INTO imp.match (connector_id, ci_id, external_key, match_method, first_seen_run_id, last_seen_run_id, last_seen_at, missed_runs)
		VALUES (?, ?, ?, ?, ?, ?, now(), 0)
		ON CONFLICT (connector_id, ci_id) DO UPDATE
		SET external_key = EXCLUDED.external_key, match_method = EXCLUDED.match_method,
		    last_seen_run_id = EXCLUDED.last_seen_run_id, last_seen_at = now(), missed_runs = 0
	`, connectorID, ciID, externalKey, method, runID, runID).Exec()
	if err != nil {
		return fmt.Errorf("pipeline: imp.match id=%d: %w", ciID, err)
	}
	return nil
}

func insertNormRecord(ctx context.Context, q querier, runID int64, seq int, rec canon.Record, namingRes naming.Result,
	matchCIID int64, matchMethod, status, errMsg string) (int64, error) {
	identJSON, _ := json.Marshal(rec.Identifiers)
	attrsJSON, _ := json.Marshal(rec.Attrs)
	var id int64
	err := q.RawWithCtx(ctx, `
		INSERT INTO imp.norm_record (run_id, seq, ci_class, source_name, identifiers, attrs, naming_status,
		       naming_suggestion, match_ci_id, match_method, status, error_message)
		VALUES (?, ?, ?, ?, CAST(? AS jsonb), CAST(? AS jsonb), nullif(?, ''), nullif(?, ''), nullif(?, 0), nullif(?, ''), ?, nullif(?, ''))
		RETURNING id
	`, runID, seq, rec.Class, rec.SourceName, string(identJSON), string(attrsJSON), string(namingRes.Status),
		namingRes.Suggestion, matchCIID, matchMethod, status, errMsg).QueryRow(&id)
	if err != nil {
		return 0, fmt.Errorf("pipeline: imp.norm_record #%d: %w", seq, err)
	}
	return id, nil
}

func insertConflict(ctx context.Context, q querier, runID int64, normRecordID, ciID *int64, kind string, details map[string]any) error {
	b, _ := json.Marshal(details)
	_, err := q.RawWithCtx(ctx, `
		INSERT INTO imp.conflict (run_id, norm_record_id, ci_id, kind, details) VALUES (?, ?, ?, ?, CAST(? AS jsonb))
	`, runID, nullInt64(normRecordID), nullInt64(ciID), kind, string(b)).Exec()
	if err != nil {
		return fmt.Errorf("pipeline: imp.conflict: %w", err)
	}
	return nil
}

func nullInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

// retainMatch подтверждает связку объекта, который прочитать не удалось: last_seen_run_id
// переставляется на текущий прогон, а missed_runs не растёт — недоступность источника не делает КЕ
// «пропавшей» (harness/04, урок 3). Счётчик и метка времени пропуска остаются прежними.
func retainMatch(ctx context.Context, q querier, env *runEnv, externalID string) error {
	if _, err := q.RawWithCtx(ctx, `UPDATE imp.match SET last_seen_run_id = ? WHERE connector_id = ? AND external_key = ?`,
		env.runID, env.connectorID, "ext:"+externalID).Exec(); err != nil {
		return fmt.Errorf("pipeline: удержание связки %s: %w", externalID, err)
	}
	return nil
}
