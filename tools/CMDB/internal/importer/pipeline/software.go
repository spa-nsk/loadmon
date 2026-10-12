package pipeline

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/canon"
)

// syncSoftware регистрирует установки ПО, сообщённые источником (plan.md E7.7). В отличие от
// syncComponents/syncInterfaces — только добавление и обновление детали (версия/путь/момент
// обнаружения detected_at), без пометки пропавших «снятыми»: у cmdb.ci_software_install нет
// колонки статуса (internal/services/software.Remove снимает установку жёстким DELETE только по
// решению человека), а неполный охват одного скана ПО (не все агенты одинаково полны) не должен
// молча «удалить» установку, которую видел прошлый прогон — ложное удаление опаснее устаревшей
// записи. nil/пустой список — источник ПО не сообщает (сегодня так у всех коннекторов 1-й волны,
// E5) — ничего не делает.
func syncSoftware(ctx context.Context, q querier, ciID int64, installs []canon.SoftwareInstall) (bool, error) {
	changed := false
	for _, in := range installs {
		swID, err := findOrCreateSoftware(ctx, q, in.Vendor, in.Product, in.Version, in.Edition)
		if err != nil {
			return changed, err
		}
		res, err := q.RawWithCtx(ctx, `
			INSERT INTO cmdb.ci_software_install (ci_id, software_id, version, install_path, detected_at)
			VALUES (?, ?, nullif(?, ''), nullif(?, ''), now())
			ON CONFLICT (ci_id, software_id, version, install_path) DO UPDATE SET detected_at = now(), updated_at = now()
		`, ciID, swID, in.Version, in.InstallPath).Exec()
		if err != nil {
			return changed, fmt.Errorf("pipeline: установка ПО %s/%s на ci_id=%d: %w", in.Vendor, in.Product, ciID, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			changed = true
		}
	}
	return changed, nil
}

// findOrCreateSoftware — ref.software.vendor/product/version/edition уникальны, но constraint
// объявлен без NULLS NOT DISTINCT (0002_ref_foundation.sql) — ON CONFLICT по нему не совпал бы при
// NULL version/edition (для PostgreSQL каждый NULL в UNIQUE различен). Поэтому ищем сами через
// IS NOT DISTINCT FROM, а не полагаемся на конфликт.
func findOrCreateSoftware(ctx context.Context, q querier, vendor, product, version, edition string) (int64, error) {
	var id int64
	err := q.RawWithCtx(ctx, `
		SELECT id FROM ref.software
		WHERE vendor = ? AND product = ? AND version IS NOT DISTINCT FROM nullif(?, '') AND edition IS NOT DISTINCT FROM nullif(?, '')
	`, vendor, product, version, edition).QueryRow(&id)
	if err == nil {
		return id, nil
	}
	if err != orm.ErrNoRows {
		return 0, fmt.Errorf("pipeline: чтение справочника ПО %s/%s: %w", vendor, product, err)
	}
	if err := q.RawWithCtx(ctx, `
		INSERT INTO ref.software (vendor, product, version, edition) VALUES (?, ?, nullif(?, ''), nullif(?, ''))
		RETURNING id
	`, vendor, product, version, edition).QueryRow(&id); err != nil {
		return 0, fmt.Errorf("pipeline: создание записи справочника ПО %s/%s: %w", vendor, product, err)
	}
	return id, nil
}
