// Массовые операции над несколькими КЕ (plan.md E3.6): смена стадии ЖЦ, владельца (компании) или
// площадки для выбранного списка КЕ с предпросмотром (BulkPreview) и применением одной причиной
// в одном change_set (BulkApply) — а не по одному change_set на каждую КЕ.
package ci

import (
	"context"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// BulkField — какое поле массово меняется.
type BulkField string

const (
	BulkFieldStage    BulkField = "stage"
	BulkFieldCompany  BulkField = "company"
	BulkFieldLocation BulkField = "location"
)

// BulkItem — одна КЕ в предпросмотре/результате массового изменения. Skip — КЕ не будет
// затронута (уже в целевом состоянии, переход не разрешён матрицей, не хватает обязательных
// атрибутов и т.п.) с человекочитаемой причиной в SkipReason.
type BulkItem struct {
	CIID       int64
	Name       string
	Old        string
	New        string
	Skip       bool
	SkipReason string
}

// bulkPlan вычисляет для каждой запрошенной КЕ старое/новое значение и причину пропуска, не
// изменяя данные. Используется и для предпросмотра (BulkPreview, вне транзакции), и как первый
// шаг применения (BulkApply, внутри транзакции) — план пересчитывается заново на момент записи,
// чтобы не унести в апдейт КЕ, ставшую неприменимой между предпросмотром и подтверждением.
func bulkPlan(ctx context.Context, q rawQuerier, field BulkField, ids []int64, value string) ([]BulkItem, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: не выбрано ни одной КЕ", svcerr.ErrInvalidInput)
	}
	switch field {
	case BulkFieldStage:
		if !KnownStage(value) {
			return nil, fmt.Errorf("%w: неизвестная стадия %q", svcerr.ErrInvalidState, value)
		}
	case BulkFieldCompany, BulkFieldLocation:
		// пустое значение допустимо — сброс поля в NULL
	default:
		return nil, fmt.Errorf("%w: неизвестное поле массового изменения %q", svcerr.ErrInvalidInput, field)
	}

	items := make([]BulkItem, 0, len(ids))
	for _, id := range ids {
		var name, ciClass, stage, company, location string
		err := q.RawWithCtx(ctx, `
			SELECT name, ci_class, lifecycle_stage, COALESCE(company_code, ''), COALESCE(location_code, '')
			FROM cmdb.ci WHERE id = ?
		`, id).QueryRow(&name, &ciClass, &stage, &company, &location)
		if err == orm.ErrNoRows {
			items = append(items, BulkItem{CIID: id, Skip: true, SkipReason: "КЕ не найдена"})
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("ci: чтение КЕ id=%d: %w", id, err)
		}

		item := BulkItem{CIID: id, Name: name}
		switch field {
		case BulkFieldStage:
			item.Old, item.New = stage, value
			switch {
			case stage == value:
				item.Skip, item.SkipReason = true, "уже в этой стадии"
			case !CanTransition(stage, value):
				item.Skip, item.SkipReason = true, fmt.Sprintf("переход %s → %s не разрешён матрицей", stage, value)
			default:
				missing, err := requiredAttrsMissing(ctx, q, id, ciClass, value)
				if err != nil {
					return nil, err
				}
				if len(missing) > 0 {
					item.Skip, item.SkipReason = true, "не заполнены обязательные атрибуты: "+strings.Join(missing, ", ")
				}
			}
		case BulkFieldCompany:
			item.Old, item.New = company, value
			if company == value {
				item.Skip, item.SkipReason = true, "уже установлено"
			}
		case BulkFieldLocation:
			item.Old, item.New = location, value
			if location == value {
				item.Skip, item.SkipReason = true, "уже установлено"
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// BulkPreview показывает, что изменится при массовом изменении, не записывая ничего в БД.
func BulkPreview(ctx context.Context, field BulkField, ids []int64, value string) ([]BulkItem, error) {
	return bulkPlan(ctx, orm.NewOrm(), field, ids, value)
}

// BulkResult — итог применения массового изменения.
type BulkResult struct {
	ChangeSetID int64
	Applied     int
	Skipped     int
	Items       []BulkItem
}

// BulkApply применяет изменение ко всем не пропущенным (Skip == false) КЕ из bulkPlan одним
// change_set — все затронутые КЕ получают в audit.change_log один и тот же change_set_id с общей
// причиной, а не по одному change set на каждую (E3.6). Строки, ставшие неприменимыми к моменту
// записи, просто пропускаются (см. bulkPlan) — частичный успех не откатывает операцию целиком.
func BulkApply(ctx context.Context, field BulkField, ids []int64, value, reason string, userID int64) (BulkResult, error) {
	var res BulkResult
	meta := changectx.Meta{Source: changectx.SourceManual, UserID: userID, Reason: reason}
	csID, err := changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		items, err := bulkPlan(ctx, tx, field, ids, value)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.Skip {
				res.Skipped++
				continue
			}
			switch field {
			case BulkFieldStage:
				if _, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET lifecycle_stage = ?, updated_at = now(), updated_by = ? WHERE id = ?`,
					value, userID, item.CIID).Exec(); err != nil {
					return fmt.Errorf("ci: массовая смена стадии id=%d: %w", item.CIID, err)
				}
				if value == StageRetired || value == StageDisposed {
					if _, err := tx.RawWithCtx(ctx, `
						UPDATE cmdb.ci_rel
						SET valid_period = tstzrange(lower(valid_period), greatest(now(), lower(valid_period))), updated_at = now(), updated_by = ?
						WHERE (src_ci_id = ? OR dst_ci_id = ?) AND upper_inf(valid_period)
					`, userID, item.CIID, item.CIID).Exec(); err != nil {
						return fmt.Errorf("ci: закрытие связей id=%d: %w", item.CIID, err)
					}
				}
			case BulkFieldCompany:
				if _, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET company_code = ?, updated_at = now(), updated_by = ? WHERE id = ?`,
					nullableString(value), userID, item.CIID).Exec(); err != nil {
					return fmt.Errorf("ci: массовая смена владельца id=%d: %w", item.CIID, err)
				}
			case BulkFieldLocation:
				if _, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET location_code = ?, updated_at = now(), updated_by = ? WHERE id = ?`,
					nullableString(value), userID, item.CIID).Exec(); err != nil {
					return fmt.Errorf("ci: массовая смена площадки id=%d: %w", item.CIID, err)
				}
			}
			res.Applied++
		}
		res.Items = items
		return nil
	})
	res.ChangeSetID = csID
	return res, err
}
