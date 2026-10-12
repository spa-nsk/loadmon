package ci

import (
	"context"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/attrschema"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// Стадии жизненного цикла КЕ (intent §7.2, ref.lifecycle.stage).
const (
	StagePlanned         = "Planned"
	StageOrdered         = "Ordered"
	StageInStock         = "In stock"
	StageCommissioning   = "Commissioning"
	StageOperational     = "Operational"
	StageMaintenance     = "Maintenance"
	StageDecommissioning = "Decommissioning"
	StageRetired         = "Retired"
	StageDisposed        = "Disposed"
)

// ErrInvalidTransition — переход между стадиями не разрешён матрицей.
var ErrInvalidTransition = fmt.Errorf("%w: переход стадии ЖЦ не разрешён", svcerr.ErrInvalidState)

// allowedTransitions — матрица переходов. Допущение (подлежит подтверждению заказчиком вместе с
// профилем E1.0): виртуальные КЕ проходят стадии без «Ordered/In stock» — для них разрешены
// прямые переходы Planned → Commissioning. Из Retired возможен возврат на склад (повторное
// использование оборудования); Disposed — конечная стадия.
var allowedTransitions = map[string][]string{
	StagePlanned:         {StageOrdered, StageInStock, StageCommissioning, StageRetired},
	StageOrdered:         {StageInStock, StageCommissioning, StageRetired},
	StageInStock:         {StageCommissioning, StageRetired},
	StageCommissioning:   {StageOperational, StageInStock, StageRetired},
	StageOperational:     {StageMaintenance, StageDecommissioning},
	StageMaintenance:     {StageOperational, StageDecommissioning},
	StageDecommissioning: {StageRetired, StageOperational},
	StageRetired:         {StageDisposed, StageInStock},
	StageDisposed:        {},
}

// KnownStage сообщает, известна ли стадия.
func KnownStage(stage string) bool {
	_, ok := allowedTransitions[stage]
	return ok
}

// CanTransition проверяет переход from → to. Совпадающая стадия допустима (смена только статуса).
func CanTransition(from, to string) bool {
	if !KnownStage(from) || !KnownStage(to) {
		return false
	}
	if from == to {
		return true
	}
	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// ChangeLifecycleInput — смена стадии/статуса ЖЦ.
type ChangeLifecycleInput struct {
	CIID            int64
	ExpectedVersion int    // row_version, который видел пользователь
	ToStage         string // целевая стадия
	Status          string // lifecycle_status (уточнение внутри стадии); "" — сбросить в NULL
	Reason          string
	UserID          int64
	Source          changectx.SourceKind // "" — SourceManual (UI); REST API (E7.3) передаёт SourceAPI
}

// ChangeLifecycle переводит КЕ на другую стадию по матрице переходов. При переходе в Retired
// или Disposed закрывает действующие связи КЕ (valid_period получает верхнюю границу) —
// выведенная КЕ не должна фигурировать в актуальной топологии. Возвращает id change set.
func ChangeLifecycle(ctx context.Context, in ChangeLifecycleInput) (int64, error) {
	if !KnownStage(in.ToStage) {
		return 0, fmt.Errorf("%w: неизвестная стадия %q", svcerr.ErrInvalidState, in.ToStage)
	}

	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	return changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		var stage, ciClass string
		var version int
		err := tx.RawWithCtx(ctx, `SELECT lifecycle_stage, row_version, ci_class FROM cmdb.ci WHERE id = ? FOR UPDATE`, in.CIID).
			QueryRow(&stage, &version, &ciClass)
		if err == orm.ErrNoRows {
			return svcerr.NotFound("ci", in.CIID)
		}
		if err != nil {
			return fmt.Errorf("ci: чтение КЕ: %w", err)
		}
		if version != in.ExpectedVersion {
			return svcerr.Conflict("ci", in.CIID, in.ExpectedVersion, version)
		}
		if !CanTransition(stage, in.ToStage) {
			return fmt.Errorf("%w: %s → %s", ErrInvalidTransition, stage, in.ToStage)
		}
		if err := checkRequiredAttrs(ctx, tx, in.CIID, ciClass, in.ToStage); err != nil {
			return err
		}

		if _, err := tx.RawWithCtx(ctx, `
			UPDATE cmdb.ci
			SET lifecycle_stage = ?, lifecycle_status = ?, updated_at = now(), updated_by = ?
			WHERE id = ?
		`, in.ToStage, nullableString(in.Status), in.UserID, in.CIID).Exec(); err != nil {
			return fmt.Errorf("ci: смена стадии: %w", err)
		}

		if in.ToStage == StageRetired || in.ToStage == StageDisposed {
			if _, err := tx.RawWithCtx(ctx, `
				UPDATE cmdb.ci_rel
				SET valid_period = tstzrange(lower(valid_period), greatest(now(), lower(valid_period))),
				    updated_at = now(), updated_by = ?
				WHERE (src_ci_id = ? OR dst_ci_id = ?) AND upper_inf(valid_period)
			`, in.UserID, in.CIID, in.CIID).Exec(); err != nil {
				return fmt.Errorf("ci: закрытие связей выведенной КЕ: %w", err)
			}
		}
		return nil
	})
}

// rawQuerier — общая часть orm.Ormer и orm.TxOrmer, нужная для чтения. Позволяет использовать
// requiredAttrsMissing и в транзакции (ChangeLifecycle, BulkApply), и без неё, только на чтение
// (BulkPreview) — одной и той же логикой, без дублирования под два разных типа.
type rawQuerier interface {
	RawWithCtx(ctx context.Context, query string, args ...any) orm.RawSeter
}

// requiredAttrsMissing проверяет атрибуты класса-расширения (E2.9): ref.ci_class.attr_schema
// задаёт requiredByStage — атрибуты, обязательные для перехода именно в целевую стадию (см.
// attrschema.Schema.Required, включая обоснование, почему требования не накапливаются по более
// ранним стадиям). У класса без таблицы-расширения или без схемы проверка не выполняется —
// пустая схема ничего не требует (attrschema.Parse("{}")).
func requiredAttrsMissing(ctx context.Context, q rawQuerier, ciID int64, ciClass, toStage string) ([]string, error) {
	var extTable, attrSchemaRaw string
	err := q.RawWithCtx(ctx, `SELECT COALESCE(ext_table, ''), COALESCE(attr_schema::text, '{}') FROM ref.ci_class WHERE class_code = ?`, ciClass).
		QueryRow(&extTable, &attrSchemaRaw)
	if err == orm.ErrNoRows || extTable == "" {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ci: чтение ref.ci_class %q: %w", ciClass, err)
	}
	schema, err := attrschema.Parse(attrSchemaRaw)
	if err != nil {
		return nil, fmt.Errorf("ci: %s.attr_schema: %w", ciClass, err)
	}
	required := schema.Required(toStage)
	if len(required) == 0 {
		return nil, nil
	}

	var rows []orm.Params
	if _, err := q.RawWithCtx(ctx, `SELECT * FROM cmdb.`+extTable+` WHERE ci_id = ?`, ciID).Values(&rows); err != nil {
		return nil, fmt.Errorf("ci: чтение атрибутов cmdb.%s: %w", extTable, err)
	}
	attrs := map[string]string{}
	if len(rows) > 0 {
		for k, v := range rows[0] {
			if v != nil {
				attrs[k] = fmt.Sprint(v)
			}
		}
	}
	return attrschema.Missing(required, attrs), nil
}

// checkRequiredAttrs — requiredAttrsMissing, обёрнутый в понятную ошибку для одиночного перехода.
func checkRequiredAttrs(ctx context.Context, q rawQuerier, ciID int64, ciClass, toStage string) error {
	missing, err := requiredAttrsMissing(ctx, q, ciID, ciClass, toStage)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: для перехода в стадию %q не заполнены обязательные атрибуты: %s",
			svcerr.ErrInvalidState, toStage, strings.Join(missing, ", "))
	}
	return nil
}

// Retire — вывод из эксплуатации вместо удаления (intent §7.4): переводит КЕ в Retired.
// Из Operational/Maintenance напрямую нельзя — сначала Decommissioning, как требует матрица.
func Retire(ctx context.Context, ciID int64, expectedVersion int, reason string, userID int64) (int64, error) {
	return ChangeLifecycle(ctx, ChangeLifecycleInput{
		CIID: ciID, ExpectedVersion: expectedVersion, ToStage: StageRetired, Reason: reason, UserID: userID,
	})
}
