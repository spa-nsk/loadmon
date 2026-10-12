package repo

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
)

// AllocateSeq выделяет следующий свободный номер для (profileCode, designationCode, locationCode),
// блокируя конкретную строку ref.seq_range (SELECT ... FOR UPDATE OF) и продвигая её курсор
// next_value в той же транзакции (plan.md E1.6). Диапазоны purpose='legacy_reserved' в выборку
// не попадают — из них новые номера не выдаются (intent §5.3).
//
// locationCode == "" — использовать только общий диапазон (location_code IS NULL); если передан
// код конкретной площадки, диапазон, привязанный к ней, имеет приоритет над общим.
func AllocateSeq(ctx context.Context, tx orm.TxOrmer, profileCode, designationCode, locationCode string) (int64, error) {
	var rangeID, nextValue, rangeTo int64
	err := tx.RawWithCtx(ctx, `
		SELECT sr.id, sr.next_value, sr.range_to
		FROM ref.seq_range sr
		JOIN ref.designation d ON d.id = sr.designation_id
		WHERE d.profile_code = ? AND d.designation_code = ? AND sr.purpose = 'active'
		  AND (sr.location_code = ? OR sr.location_code IS NULL)
		ORDER BY (sr.location_code IS NULL), sr.next_value
		FOR UPDATE OF sr
		LIMIT 1
	`, profileCode, designationCode, locationCode).QueryRow(&rangeID, &nextValue, &rangeTo)
	if err != nil {
		return 0, fmt.Errorf("repo: нет активного диапазона номеров для %s/%s: %w", profileCode, designationCode, err)
	}
	if nextValue > rangeTo {
		return 0, fmt.Errorf("repo: диапазон номеров для %s/%s исчерпан (next=%d, до=%d)", profileCode, designationCode, nextValue, rangeTo)
	}

	if _, err := tx.RawWithCtx(ctx, `UPDATE ref.seq_range SET next_value = next_value + 1 WHERE id = ?`, rangeID).Exec(); err != nil {
		return 0, fmt.Errorf("repo: не удалось продвинуть курсор диапазона номеров: %w", err)
	}
	return nextValue, nil
}
