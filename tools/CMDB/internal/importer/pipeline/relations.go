package pipeline

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
)

// relKey — связь типа Type к КЕ Dst.
type relKey struct {
	Type string
	Dst  int64
}

// syncRelations приводит связи КЕ снимка в соответствие с record.Relations (E5.1/E5.2a): недостающие
// создаются, а действующие связи ЭТОГО источника (ci_rel.source_system_code), которых в снимке
// больше нет, закрываются (ВМ переехала на другой хост — старая связь runs_on закрывается, а не
// остаётся второй; связь не удаляется, у неё лишь появляется верхняя граница valid_period).
// Чужие связи (введённые вручную или другим источником) не затрагиваются. Связь на КЕ,
// которой нет в снимке, пропускается и фиксируется конфликтом apply_error — молча не теряется
// (harness/04, п. 10). Порядок записей в снимке не важен: приёмники разрешаются после применения всех.
func syncRelations(ctx context.Context, q querier, env *runEnv, applied []appliedRecord) (created, closed int, err error) {
	byExt := make(map[string]int64, len(applied))
	for _, a := range applied {
		byExt[a.rec.ExternalID] = a.ciID
	}

	for _, a := range applied {
		desired := map[relKey]bool{}
		for _, rel := range a.rec.Relations {
			dst, ok := byExt[rel.TargetExternalID]
			if !ok || dst == a.ciID {
				if err := insertConflict(ctx, q, env.runID, nil, &a.ciID, "apply_error", map[string]any{
					"reason": "приёмник связи не найден в снимке источника", "rel_type": rel.Type, "target_external_id": rel.TargetExternalID}); err != nil {
					return created, closed, err
				}
				continue
			}
			desired[relKey{rel.Type, dst}] = true
		}

		var rows []orm.Params
		if _, err := q.RawWithCtx(ctx, `
			SELECT id, rel_type, dst_ci_id FROM cmdb.ci_rel
			WHERE src_ci_id = ? AND source_system_code IS NOT DISTINCT FROM nullif(?, '') AND upper_inf(valid_period)
		`, a.ciID, env.sourceSystemCode).Values(&rows); err != nil {
			return created, closed, fmt.Errorf("pipeline: чтение связей id=%d: %w", a.ciID, err)
		}
		existing := map[relKey]bool{}
		for _, r := range rows {
			var dst, id int64
			fmt.Sscan(fmt.Sprint(r["dst_ci_id"]), &dst)
			fmt.Sscan(fmt.Sprint(r["id"]), &id)
			k := relKey{fmt.Sprint(r["rel_type"]), dst}
			existing[k] = true
			if !desired[k] {
				if _, err := q.RawWithCtx(ctx, `
					UPDATE cmdb.ci_rel
					SET valid_period = tstzrange(lower(valid_period), greatest(now(), lower(valid_period))), updated_at = now(), updated_by = ?
					WHERE id = ?
				`, env.userID, id).Exec(); err != nil {
					return created, closed, fmt.Errorf("pipeline: закрытие связи %d: %w", id, err)
				}
				closed++
			}
		}

		for k := range desired {
			if existing[k] {
				continue
			}
			// Связь того же типа между теми же КЕ, действующая от другого источника или введённая
			// вручную, повторно не создаётся (ограничение WITHOUT OVERLAPS всё равно не позволило бы).
			res, err := q.RawWithCtx(ctx, `
				INSERT INTO cmdb.ci_rel (rel_type, src_ci_id, dst_ci_id, valid_period, source_system_code, created_by, updated_by)
				SELECT ?, ?, ?, tstzrange(now(), NULL), nullif(?, ''), ?, ?
				WHERE NOT EXISTS (SELECT 1 FROM cmdb.ci_rel WHERE rel_type = ? AND src_ci_id = ? AND dst_ci_id = ? AND upper_inf(valid_period))
			`, k.Type, a.ciID, k.Dst, env.sourceSystemCode, env.userID, env.userID, k.Type, a.ciID, k.Dst).Exec()
			if err != nil {
				return created, closed, fmt.Errorf("pipeline: создание связи %s %d->%d: %w", k.Type, a.ciID, k.Dst, err)
			}
			if n, _ := res.RowsAffected(); n > 0 {
				created++
			}
		}
	}
	return created, closed, nil
}
