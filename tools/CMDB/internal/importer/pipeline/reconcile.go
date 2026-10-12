package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/canon"
	"cmdb/internal/services/ci"
)

// coreFields — колонки cmdb.ci, которые импорт может обновлять при повторном согласовании (E4.7).
// name/source_name сюда не входят — импорт никогда не переименовывает уже существующую КЕ
// (E4.5/intent §5.7). Все перечисленные колонки текстовые (database/migrations/0007_cmdb_ci.sql) —
// CAST по каталогу БД, как для таблиц-расширений, тут не нужен.
var coreFields = []string{"location_code", "company_code", "environment_code", "serial_number", "description", "criticality"}

func coreFieldValue(rec canon.Record, name string) string {
	switch name {
	case "location_code":
		return rec.LocationCode
	case "company_code":
		return rec.CompanyCode
	case "environment_code":
		return rec.EnvironmentCode
	case "serial_number":
		return rec.SerialNumber
	case "description":
		return rec.Description
	case "criticality":
		return rec.Criticality
	}
	return ""
}

// applyToExisting согласует повторно найденную КЕ с record (E4.7): непустые поля источника
// переписывают текущие значения, кроме атрибутов, заблокированных вручную
// (cmdb.attr_source.is_locked) — те попадают в locked и становятся конфликтом, не прерывая прогон.
// dryRun — определить, изменится ли что-то, без записи (предпросмотр перед реальным прогоном).
func applyToExisting(ctx context.Context, q querier, env *runEnv, extTable string, ciID int64, rec canon.Record) (changed bool, locked []string, err error) {
	dryRun := env.dryRun
	if env.supplement {
		return applySupplement(ctx, q, env, ciID, rec)
	}
	lockedCI, lockedExt, outCI, outExt, err := lockedAttrs(ctx, q, env, extTable, ciID)
	if err != nil {
		return false, nil, err
	}

	var coreNames []string
	var coreValues []any
	coreAttrs := map[string]any{}
	for _, name := range coreFields {
		v := coreFieldValue(rec, name)
		if v == "" {
			continue
		}
		if outCI[name] {
			continue // значение подтверждено источником с более высоким доверием — не перебиваем
		}
		if lockedCI[name] {
			locked = append(locked, "ci."+name)
			continue
		}
		coreNames = append(coreNames, name)
		coreValues = append(coreValues, v)
		coreAttrs[name] = v
	}

	extAttrs := map[string]any{}
	for k, v := range rec.Attrs {
		if v == nil || v == "" {
			continue
		}
		if outExt[k] {
			continue
		}
		if lockedExt[k] {
			locked = append(locked, extTable+"."+k)
			continue
		}
		extAttrs[k] = v
	}
	sort.Strings(locked)

	coreChanged, err := applyCoreFields(ctx, q, ciID, coreNames, coreValues, dryRun)
	if err != nil {
		return false, locked, err
	}
	extChanged, err := applyExtFields(ctx, q, extTable, ciID, extAttrs, dryRun, env.userID)
	if err != nil {
		return false, locked, err
	}

	vendorChanged, err := applyVendor(ctx, q, env, ciID, rec.Vendor)
	if err != nil {
		return false, locked, err
	}

	if !dryRun {
		if err := recordProvenance(ctx, q, ciID, extTable, coreAttrs, extAttrs, env.sourceSystemCode); err != nil {
			return false, locked, err
		}
	}
	return coreChanged || extChanged || vendorChanged, locked, nil
}

// lastSeenThrottle — не чаще этого интервала last_seen_at обновляется у уже известной КЕ: каждое
// обновление cmdb.ci — версия в hist, а опрос идёт раз в часы (100 тыс. КЕ x 24 прогона в сутки
// раздули бы историю впустую). Суточная точность достаточна для отчёта «не подтверждалась > 30 дней».
const lastSeenThrottle = "24 hours"

// touchLastSeen подтверждает существование КЕ источником (E4.8) — не чаще раза в сутки.
func touchLastSeen(ctx context.Context, q querier, ciID int64) error {
	if _, err := q.RawWithCtx(ctx, `
		UPDATE cmdb.ci SET last_seen_at = now(), updated_at = now()
		WHERE id = ? AND (last_seen_at IS NULL OR last_seen_at < now() - CAST('`+lastSeenThrottle+`' AS interval))
	`, ciID).Exec(); err != nil {
		return fmt.Errorf("pipeline: last_seen_at id=%d: %w", ciID, err)
	}
	return nil
}

// applyVendor сливает вендорские сведения источника в cmdb.ci.attrs под ключом источника; пишет
// только при отличии от текущего значения (иначе лишняя версия в hist). dryRun — только проверка.
func applyVendor(ctx context.Context, q querier, env *runEnv, ciID int64, vendor map[string]any) (bool, error) {
	if len(vendor) == 0 {
		return false, nil
	}
	b, err := json.Marshal(vendor)
	if err != nil {
		return false, fmt.Errorf("pipeline: вендорские сведения: %w", err)
	}
	if env.dryRun {
		var changed bool
		if err := q.RawWithCtx(ctx, `SELECT (attrs -> CAST(? AS text)) IS DISTINCT FROM CAST(? AS jsonb) FROM cmdb.ci WHERE id = ?`,
			env.vendorKey, string(b), ciID).QueryRow(&changed); err != nil {
			return false, fmt.Errorf("pipeline: проверка вендорских сведений id=%d: %w", ciID, err)
		}
		return changed, nil
	}
	res, err := q.RawWithCtx(ctx, `
		UPDATE cmdb.ci SET attrs = attrs || jsonb_build_object(CAST(? AS text), CAST(? AS jsonb)), updated_at = now(), updated_by = ?
		WHERE id = ? AND (attrs -> CAST(? AS text)) IS DISTINCT FROM CAST(? AS jsonb)
	`, env.vendorKey, string(b), env.userID, ciID, env.vendorKey, string(b)).Exec()
	if err != nil {
		return false, fmt.Errorf("pipeline: вендорские сведения id=%d: %w", ciID, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// applySupplement — согласование для источника-дополнения (connector.ModeSupplement): пустые поля
// cmdb.ci заполняются значением источника, непустые НЕ перезаписываются (источник с низким
// приоритетом; harness/04, п. 2: режим supplement), плюс вендорские сведения. Атрибуты классов
// (таблицы-расширения) дополнение не пишет.
func applySupplement(ctx context.Context, q querier, env *runEnv, ciID int64, rec canon.Record) (bool, []string, error) {
	changed := false
	for _, name := range coreFields {
		v := coreFieldValue(rec, name)
		if v == "" {
			continue
		}
		if env.dryRun {
			var empty bool
			if err := q.RawWithCtx(ctx, `SELECT COALESCE(`+name+`, '') = '' FROM cmdb.ci WHERE id = ?`, ciID).QueryRow(&empty); err != nil {
				return false, nil, fmt.Errorf("pipeline: проверка cmdb.ci.%s id=%d: %w", name, ciID, err)
			}
			changed = changed || empty
			continue
		}
		res, err := q.RawWithCtx(ctx, `UPDATE cmdb.ci SET `+name+` = ?, updated_at = now(), updated_by = ?
			WHERE id = ? AND COALESCE(`+name+`, '') = ''`, v, env.userID, ciID).Exec()
		if err != nil {
			return false, nil, fmt.Errorf("pipeline: дополнение cmdb.ci.%s id=%d: %w", name, ciID, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			changed = true
			if err := recordProvenance(ctx, q, ciID, "", map[string]any{name: v}, nil, env.sourceSystemCode); err != nil {
				return false, nil, err
			}
		}
	}
	vendorChanged, err := applyVendor(ctx, q, env, ciID, rec.Vendor)
	if err != nil {
		return false, nil, err
	}
	return changed || vendorChanged, nil, nil
}

// lockedAttrs возвращает атрибуты КЕ, которые импорт не должен перезаписывать, двух видов
// (отдельно для колонок cmdb.ci — entity='ci' — и таблицы-расширения — entity=extTable):
//   - locked — заблокированные вручную (cmdb.attr_source.is_locked): о попытке перезаписи заводится конфликт;
//   - outranked — текущее значение подтверждено ДРУГИМ источником с более высоким trust_level
//     (ref.source_system): приоритет источников (E4.7) — без конфликта, молча: иначе два источника
//     одного сервера (OneView и Redfish) перезаписывали бы друг друга при каждом прогоне.
//
// Приоритет действует, пока провенанс не устарел — правило refresh_after пока не реализовано.
func lockedAttrs(ctx context.Context, q querier, env *runEnv, extTable string, ciID int64) (ciLocked, extLocked, ciOut, extOut map[string]bool, err error) {
	var rows []orm.Params
	if _, err := q.RawWithCtx(ctx, `
		SELECT a.entity, a.attr_name, a.is_locked
		FROM cmdb.attr_source a LEFT JOIN ref.source_system s ON s.source_system_code = a.source_system_code
		WHERE a.entity_id = ? AND a.entity IN ('ci', ?)
		  AND (a.is_locked OR (a.source_system_code IS NOT NULL AND a.source_system_code <> ? AND COALESCE(s.trust_level, 0) > ?))
	`, ciID, extTable, env.sourceSystemCode, env.trustLevel).Values(&rows); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("pipeline: чтение блокировок id=%d: %w", ciID, err)
	}
	ciLocked, extLocked, ciOut, extOut = map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		name := fmt.Sprint(r["attr_name"])
		manual := fmt.Sprint(r["is_locked"]) == "true"
		switch {
		case fmt.Sprint(r["entity"]) == "ci" && manual:
			ciLocked[name] = true
		case fmt.Sprint(r["entity"]) == "ci":
			ciOut[name] = true
		case manual:
			extLocked[name] = true
		default:
			extOut[name] = true
		}
	}
	return ciLocked, extLocked, ciOut, extOut, nil
}

// applyCoreFields обновляет непустые незаблокированные поля cmdb.ci (names/values уже
// отфильтрованы вызывающей стороной). dryRun — только определить факт изменения, без записи.
func applyCoreFields(ctx context.Context, q querier, ciID int64, names []string, values []any, dryRun bool) (bool, error) {
	if len(names) == 0 {
		return false, nil
	}
	phs := placeholders(len(names))
	rowExpr := strings.Join(names, ", ")
	rowPhs := strings.Join(phs, ", ")

	if dryRun {
		args := append(append([]any{}, values...), ciID)
		var changed bool
		q2 := fmt.Sprintf(`SELECT (ROW(%s) IS DISTINCT FROM ROW(%s)) FROM cmdb.ci WHERE id = ?`, rowExpr, rowPhs)
		if err := q.RawWithCtx(ctx, q2, args...).QueryRow(&changed); err != nil {
			return false, fmt.Errorf("pipeline: проверка изменений cmdb.ci id=%d: %w", ciID, err)
		}
		return changed, nil
	}

	sets := make([]string, len(names))
	for i, n := range names {
		sets[i] = n + " = " + phs[i]
	}
	// Без WHERE ... IS DISTINCT FROM триггер hist.versioning() записал бы версию и при неизменных данных.
	args := append(append([]any{}, values...), ciID)
	args = append(args, values...)
	q2 := fmt.Sprintf(`UPDATE cmdb.ci SET %s, updated_at = now() WHERE id = ? AND ROW(%s) IS DISTINCT FROM ROW(%s)`,
		strings.Join(sets, ", "), rowExpr, rowPhs)
	result, err := q.RawWithCtx(ctx, q2, args...).Exec()
	if err != nil {
		return false, fmt.Errorf("pipeline: обновление cmdb.ci id=%d: %w", ciID, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// applyExtFields согласует атрибуты класса в таблице-расширении. Запись переиспользует
// ci.UpsertExtensionTx (та же CAST-по-каталогу-БД логика, что у ручного ввода) — ей нужна
// транзакция, поэтому в dry-run (только чтение) вместо неё используется extProbeChanged.
func applyExtFields(ctx context.Context, q querier, extTable string, ciID int64, attrs map[string]any, dryRun bool, userID int64) (bool, error) {
	if len(attrs) == 0 || extTable == "" {
		return false, nil
	}
	if dryRun {
		return extProbeChanged(ctx, q, extTable, ciID, attrs)
	}
	tx, ok := q.(orm.TxOrmer)
	if !ok {
		return false, fmt.Errorf("pipeline: запись атрибутов cmdb.%s требует транзакцию", extTable)
	}
	var res ci.ExtensionResult
	if err := ci.UpsertExtensionTx(ctx, tx, ci.ExtensionInput{CIID: ciID, Table: extTable, Attrs: attrs, UserID: userID}, &res); err != nil {
		return false, fmt.Errorf("pipeline: атрибуты cmdb.%s id=%d: %w", extTable, ciID, err)
	}
	return res.Changed, nil
}

// extProbeChanged — read-only эквивалент ci.UpsertExtensionTx для dry-run: та же проверка
// ROW(...) IS DISTINCT FROM ROW(...) по каталогу колонок БД, но без записи.
func extProbeChanged(ctx context.Context, q querier, extTable string, ciID int64, attrs map[string]any) (bool, error) {
	cols, err := ci.ExtensionColumns(ctx, q, extTable)
	if err != nil {
		return false, err
	}
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		if _, ok := cols[name]; !ok {
			return false, fmt.Errorf("pipeline: колонка %q недоступна в cmdb.%s", name, extTable)
		}
		names = append(names, name)
	}
	sort.Strings(names)

	values := make([]any, len(names))
	casts := make([]string, len(names))
	for i, name := range names {
		v, err := ci.PGText(attrs[name])
		if err != nil {
			return false, fmt.Errorf("pipeline: значение %q: %w", name, err)
		}
		values[i] = v
		casts[i] = "CAST(CAST(? AS text) AS " + cols[name] + ")"
	}

	args := append(append([]any{}, values...), ciID)
	var changed bool
	q2 := fmt.Sprintf(`SELECT (ROW(%s) IS DISTINCT FROM ROW(%s)) FROM cmdb.%s WHERE ci_id = ?`,
		strings.Join(names, ", "), strings.Join(casts, ", "), extTable)
	if err := q.RawWithCtx(ctx, q2, args...).QueryRow(&changed); err != nil {
		if err == orm.ErrNoRows {
			return true, nil // строки расширения ещё нет — реальный прогон её создаст
		}
		return false, fmt.Errorf("pipeline: проверка изменений cmdb.%s id=%d: %w", extTable, ciID, err)
	}
	return changed, nil
}

func placeholders(n int) []string {
	p := make([]string, n)
	for i := range p {
		p[i] = "?"
	}
	return p
}

// recordProvenance фиксирует источник текущих значений атрибутов (cmdb.attr_source, E4.7/intent
// §7.3): пишется при каждом прогоне для каждого непустого незаблокированного атрибута независимо
// от того, отличалось ли значение от прежнего — источник подтвердил его именно сейчас, а это само
// по себе полезная информация (наблюдение остаётся свежим).
func recordProvenance(ctx context.Context, q querier, ciID int64, extTable string, coreAttrs, extAttrs map[string]any, sourceSystemCode string) error {
	write := func(entity, name string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("pipeline: провенанс %s.%s: %w", entity, name, err)
		}
		if _, err := q.RawWithCtx(ctx, `
			INSERT INTO cmdb.attr_source (entity, entity_id, attr_name, source_system_code, source_value, observed_at)
			VALUES (?, ?, ?, nullif(?, ''), CAST(? AS jsonb), now())
			ON CONFLICT (entity, entity_id, attr_name) DO UPDATE
			SET source_system_code = EXCLUDED.source_system_code, source_value = EXCLUDED.source_value, observed_at = now()
			WHERE NOT cmdb.attr_source.is_locked
		`, entity, ciID, name, sourceSystemCode, string(b)).Exec(); err != nil {
			return fmt.Errorf("pipeline: провенанс %s.%s id=%d: %w", entity, name, ciID, err)
		}
		return nil
	}
	for name, v := range coreAttrs {
		if err := write("ci", name, v); err != nil {
			return err
		}
	}
	if extTable == "" {
		return nil
	}
	for name, v := range extAttrs {
		if err := write(extTable, name, v); err != nil {
			return err
		}
	}
	return nil
}
