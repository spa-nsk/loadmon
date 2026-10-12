package pipeline

import (
	"context"
	"fmt"
)

// Выведенные связи (plan.md E5.5): объекты приходят из РАЗНЫХ источников и сами друг про друга не
// знают — сквозную цепочку «ВМ → хост → сервер → датастор → том → массив» строит сама CMDB по
// совпадению технических идентификаторов. Связи ведутся от имени источника 'derived' и
// синхронизируются после каждого прогона: устаревшие закрываются (совпадение пропало), ручные и
// связи других источников не трогаются. Неоднозначность (идентификатор совпал с несколькими
// объектами) связь не создаёт — угадывания и автослияния нет.

// SMBIOS до версии 2.6 хранит первые три группы UUID в другом порядке байт: vSphere и Redfish могут
// показывать один и тот же UUID «зеркально». Сопоставляем обе формы записи.
const swapUUID = `substr(%[1]s, 7, 2) || substr(%[1]s, 5, 2) || substr(%[1]s, 3, 2) || substr(%[1]s, 1, 2) || '-' ||
	substr(%[1]s, 12, 2) || substr(%[1]s, 10, 2) || '-' || substr(%[1]s, 17, 2) || substr(%[1]s, 15, 2) || substr(%[1]s, 19)`

// Каждый запрос возвращает (src_id, dst_id, n): n = 1 — совпадение однозначно.
var (
	// хост гипервизора runs_on физический сервер: серийный номер либо SMBIOS UUID
	pairsHostServer = `
		SELECT src_id, dst_id, count(*) OVER (PARTITION BY src_id) AS n FROM (
			SELECT h.id AS src_id, s.id AS dst_id
			FROM cmdb.ci h JOIN cmdb.ci s ON s.ci_class = 'server' AND s.lifecycle_stage NOT IN ('Retired', 'Disposed')
			     AND lower(s.serial_number) = lower(h.serial_number)
			WHERE h.ci_class = 'hypervisor_host' AND h.lifecycle_stage NOT IN ('Retired', 'Disposed') AND COALESCE(h.serial_number, '') <> ''
			UNION
			SELECT h.id, s.id
			FROM cmdb.ci h JOIN cmdb.ci_hypervisor hv ON hv.ci_id = h.id
			     JOIN cmdb.ci_server ss ON ss.bios_uuid IS NOT NULL AND (ss.bios_uuid = hv.bios_uuid OR ss.bios_uuid::text = ` + fmt.Sprintf(swapUUID, "hv.bios_uuid::text") + `)
			     JOIN cmdb.ci s ON s.id = ss.ci_id AND s.ci_class = 'server' AND s.lifecycle_stage NOT IN ('Retired', 'Disposed')
			WHERE h.ci_class = 'hypervisor_host' AND h.lifecycle_stage NOT IN ('Retired', 'Disposed') AND hv.bios_uuid IS NOT NULL
		) c`

	// датастор depends_on том СХД: NAA устройства хранилища = WWN тома; n считается на пару (датастор, NAA)
	pairsDatastoreVolume = `
		SELECT d.id AS src_id, v.id AS dst_id, count(*) OVER (PARTITION BY d.id, n.naa) AS n
		FROM cmdb.ci d JOIN cmdb.ci_datastore dd ON dd.ci_id = d.id, unnest(dd.naa) AS n(naa)
		     JOIN cmdb.ci_volume cv ON cv.wwn = n.naa
		     JOIN cmdb.ci v ON v.id = cv.ci_id AND v.ci_class = 'volume' AND v.lifecycle_stage NOT IN ('Retired', 'Disposed')
		WHERE d.ci_class = 'datastore' AND d.lifecycle_stage NOT IN ('Retired', 'Disposed')`

	// том mapped_to хост: WWPN инициатора, которому том презентован, принадлежит порту ровно одного хоста
	pairsVolumeHost = `
		WITH m AS (
			SELECT v.id AS vol_id, i.ci_id AS host_id, w.wwpn
			FROM cmdb.ci v JOIN cmdb.ci_volume cv ON cv.ci_id = v.id, unnest(cv.mapped_wwpns) AS w(wwpn)
			     JOIN cmdb.interface i ON i.wwpn = w.wwpn AND i.enabled
			     JOIN cmdb.ci h ON h.id = i.ci_id AND h.ci_class IN ('hypervisor_host', 'server') AND h.lifecycle_stage NOT IN ('Retired', 'Disposed')
			WHERE v.ci_class = 'volume' AND v.lifecycle_stage NOT IN ('Retired', 'Disposed')
		), u AS (SELECT wwpn, count(DISTINCT host_id) AS n FROM m GROUP BY wwpn)
		SELECT DISTINCT m.vol_id AS src_id, m.host_id AS dst_id, 1 AS n FROM m JOIN u ON u.wwpn = m.wwpn WHERE u.n = 1`
)

// syncDerived приводит связи 'derived' типа relType в соответствие с pairsSQL.
func syncDerived(ctx context.Context, q querier, env *runEnv, relType, pairsSQL string) (created, closed int, err error) {
	res, err := q.RawWithCtx(ctx, `
		UPDATE cmdb.ci_rel r
		SET valid_period = tstzrange(lower(valid_period), greatest(now(), lower(valid_period))), updated_at = now(), updated_by = ?
		WHERE r.rel_type = ? AND r.source_system_code = 'derived' AND upper_inf(r.valid_period)
		  AND NOT EXISTS (SELECT 1 FROM (`+pairsSQL+`) p WHERE p.n = 1 AND p.src_id = r.src_ci_id AND p.dst_id = r.dst_ci_id)
	`, env.userID, relType).Exec()
	if err != nil {
		return 0, 0, fmt.Errorf("pipeline: закрытие выведенных связей %s: %w", relType, err)
	}
	n, _ := res.RowsAffected()
	closed = int(n)

	res, err = q.RawWithCtx(ctx, `
		INSERT INTO cmdb.ci_rel (rel_type, src_ci_id, dst_ci_id, valid_period, source_system_code, created_by, updated_by)
		SELECT DISTINCT CAST(? AS text), p.src_id, p.dst_id, tstzrange(now(), NULL), 'derived', CAST(? AS bigint), CAST(? AS bigint)
		FROM (`+pairsSQL+`) p
		WHERE p.n = 1 AND NOT EXISTS (SELECT 1 FROM cmdb.ci_rel r WHERE r.rel_type = ? AND r.src_ci_id = p.src_id
		                              AND r.dst_ci_id = p.dst_id AND upper_inf(r.valid_period))
	`, relType, env.userID, env.userID, relType).Exec()
	if err != nil {
		return 0, closed, fmt.Errorf("pipeline: создание выведенных связей %s: %w", relType, err)
	}
	n, _ = res.RowsAffected()
	return int(n), closed, nil
}

// syncBackingVolumes заполняет ci_datastore.backing_volume_ci_id по выведенной связи, если колонка
// ещё пуста (введённое вручную значение не перезаписывается).
func syncBackingVolumes(ctx context.Context, q querier, env *runEnv) error {
	_, err := q.RawWithCtx(ctx, `
		UPDATE cmdb.ci_datastore dd SET backing_volume_ci_id = p.vol_id, updated_at = now(), updated_by = ?
		FROM (SELECT src_id AS ds_id, min(dst_id) AS vol_id FROM (`+pairsDatastoreVolume+`) x WHERE n = 1 GROUP BY src_id) p
		WHERE dd.ci_id = p.ds_id AND dd.backing_volume_ci_id IS NULL
	`, env.userID).Exec()
	if err != nil {
		return fmt.Errorf("pipeline: backing_volume_ci_id: %w", err)
	}
	return nil
}

// linkDerived строит все выведенные связи E5.5; возвращает число созданных и закрытых.
func linkDerived(ctx context.Context, q querier, env *runEnv) (created, closed int, err error) {
	for _, l := range []struct{ rel, sql string }{
		{"runs_on", pairsHostServer}, {"depends_on", pairsDatastoreVolume}, {"mapped_to", pairsVolumeHost},
	} {
		c, cl, err := syncDerived(ctx, q, env, l.rel, l.sql)
		if err != nil {
			return created, closed, err
		}
		created, closed = created+c, closed+cl
	}
	return created, closed, syncBackingVolumes(ctx, q, env)
}
