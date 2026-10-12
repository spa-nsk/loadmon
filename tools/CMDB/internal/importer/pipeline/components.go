package pipeline

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/canon"
)

// syncExtras синхронизирует компоненты, интерфейсы и установленное ПО (E7.7) КЕ с записью
// источника (E5.2). Компоненты/интерфейсы — только для записей, где источник их сообщил (не nil),
// и только «свои» строки — с attrs.source = ключ этого источника: введённые вручную или другим
// источником не трогаются. ПО синхронизируется иначе — см. syncSoftware. Dry-run ничего не считает.
func syncExtras(ctx context.Context, q querier, env *runEnv, ciID int64, rec canon.Record) (bool, error) {
	if env.dryRun || env.supplement {
		return false, nil
	}
	c1, err := syncComponents(ctx, q, env, ciID, rec.Components)
	if err != nil {
		return false, err
	}
	c2, err := syncInterfaces(ctx, q, env, ciID, rec.Interfaces)
	if err != nil {
		return false, err
	}
	c3, err := syncSoftware(ctx, q, ciID, rec.Software)
	if err != nil {
		return false, err
	}
	return c1 || c2 || c3, nil
}

type existingComponent struct {
	id                                                            int64
	parentID                                                      int64
	compType, slot, manufacturer, part, serial, firmware, capUnit string
	capValue, speed                                               string
}

func numText(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}

// syncComponents — diff по стабильному ключу компонента (Component.Key): совпавшие обновляются
// точечно и только при отличии, новые добавляются, пропавшие у источника получают status='removed'
// (ничего не удаляется, история остаётся). Родители обрабатываются раньше потомков.
func syncComponents(ctx context.Context, q querier, env *runEnv, ciID int64, comps []canon.Component) (bool, error) {
	if comps == nil {
		return false, nil
	}
	var rows []orm.Params
	if _, err := q.RawWithCtx(ctx, `
		SELECT id, COALESCE(parent_component_id, 0) AS parent_id, component_type, COALESCE(slot, '') AS slot,
		       COALESCE(manufacturer, '') AS manufacturer, COALESCE(part_number, '') AS part, COALESCE(serial_number, '') AS serial,
		       COALESCE(firmware, '') AS firmware, COALESCE(capacity_value::text, '') AS cap_value,
		       COALESCE(capacity_unit, '') AS cap_unit, COALESCE(speed, '') AS speed, attrs ->> 'key' AS key
		FROM cmdb.component
		WHERE ci_id = ? AND attrs ->> 'source' = ? AND COALESCE(status, '') = 'installed'
	`, ciID, env.vendorKey).Values(&rows); err != nil {
		return false, fmt.Errorf("pipeline: чтение компонентов id=%d: %w", ciID, err)
	}
	existing := map[string]*existingComponent{}
	for _, r := range rows {
		e := &existingComponent{compType: fmt.Sprint(r["component_type"]), slot: fmt.Sprint(r["slot"]),
			manufacturer: fmt.Sprint(r["manufacturer"]), part: fmt.Sprint(r["part"]), serial: fmt.Sprint(r["serial"]),
			firmware: fmt.Sprint(r["firmware"]), capValue: fmt.Sprint(r["cap_value"]), capUnit: fmt.Sprint(r["cap_unit"]), speed: fmt.Sprint(r["speed"])}
		fmt.Sscan(fmt.Sprint(r["id"]), &e.id)
		fmt.Sscan(fmt.Sprint(r["parent_id"]), &e.parentID)
		existing[fmt.Sprint(r["key"])] = e
	}

	changed := false
	ids := map[string]int64{} // ключ -> id компонента в БД (для родителей)
	seen := map[string]bool{}
	pending := comps
	for len(pending) > 0 {
		var next []canon.Component
		progress := false
		for _, c := range pending {
			if c.ParentKey != "" && ids[c.ParentKey] == 0 {
				// родитель ещё не обработан; если его нет в списке вовсе — считаем корневым
				if hasKey(comps, c.ParentKey) && !seen[c.ParentKey] {
					next = append(next, c)
					continue
				}
			}
			progress = true
			seen[c.Key] = true
			parent := ids[c.ParentKey]
			if e := existing[c.Key]; e != nil {
				ids[c.Key] = e.id
				upd, err := updateComponent(ctx, q, env, e, c, parent)
				if err != nil {
					return false, err
				}
				changed = changed || upd
				continue
			}
			id, err := insertComponent(ctx, q, env, ciID, c, parent)
			if err != nil {
				return false, err
			}
			ids[c.Key] = id
			changed = true
		}
		if !progress { // цикл ссылок на родителей — разрываем, вставляя оставшиеся корневыми
			for _, c := range next {
				c.ParentKey = ""
				if _, err := insertComponent(ctx, q, env, ciID, c, 0); err != nil {
					return false, err
				}
				changed = true
			}
			break
		}
		pending = next
	}

	for key, e := range existing {
		if seen[key] {
			continue
		}
		if _, err := q.RawWithCtx(ctx, `UPDATE cmdb.component SET status = 'removed', updated_at = now(), updated_by = ? WHERE id = ?`,
			env.userID, e.id).Exec(); err != nil {
			return false, fmt.Errorf("pipeline: снятие компонента %d: %w", e.id, err)
		}
		changed = true
	}
	return changed, nil
}

func hasKey(comps []canon.Component, key string) bool {
	for _, c := range comps {
		if c.Key == key {
			return true
		}
	}
	return false
}

func insertComponent(ctx context.Context, q querier, env *runEnv, ciID int64, c canon.Component, parentID int64) (int64, error) {
	var id int64
	err := q.RawWithCtx(ctx, `
		INSERT INTO cmdb.component
			(ci_id, parent_component_id, component_type, slot, manufacturer, part_number, serial_number, firmware,
			 capacity_value, capacity_unit, speed, status, attrs, first_seen_at, last_seen_at, created_by, updated_by)
		VALUES (?, nullif(?, 0), ?, nullif(?, ''), nullif(?, ''), nullif(?, ''), nullif(?, ''), nullif(?, ''),
		        CAST(nullif(?, '') AS numeric), nullif(?, ''), nullif(?, ''), 'installed',
		        jsonb_build_object('source', CAST(? AS text), 'key', CAST(? AS text)), now(), now(), ?, ?)
		RETURNING id
	`, ciID, parentID, c.Type, c.Slot, c.Manufacturer, c.PartNumber, c.SerialNumber, c.Firmware,
		numText(c.CapacityValue), c.CapacityUnit, c.Speed, env.vendorKey, c.Key, env.userID, env.userID).QueryRow(&id)
	if err != nil {
		return 0, fmt.Errorf("pipeline: вставка компонента %s: %w", c.Key, err)
	}
	return id, nil
}

func updateComponent(ctx context.Context, q querier, env *runEnv, e *existingComponent, c canon.Component, parentID int64) (bool, error) {
	if e.slot == c.Slot && e.manufacturer == c.Manufacturer && e.part == c.PartNumber && e.serial == c.SerialNumber &&
		e.firmware == c.Firmware && e.capValue == numText(c.CapacityValue) && e.capUnit == c.CapacityUnit && e.speed == c.Speed &&
		e.parentID == parentID {
		return false, nil
	}
	if _, err := q.RawWithCtx(ctx, `
		UPDATE cmdb.component SET parent_component_id = nullif(?, 0), slot = nullif(?, ''), manufacturer = nullif(?, ''),
		       part_number = nullif(?, ''), serial_number = nullif(?, ''), firmware = nullif(?, ''),
		       capacity_value = CAST(nullif(?, '') AS numeric), capacity_unit = nullif(?, ''), speed = nullif(?, ''),
		       updated_at = now(), updated_by = ?
		WHERE id = ?
	`, parentID, c.Slot, c.Manufacturer, c.PartNumber, c.SerialNumber, c.Firmware, numText(c.CapacityValue),
		c.CapacityUnit, c.Speed, env.userID, e.id).Exec(); err != nil {
		return false, fmt.Errorf("pipeline: обновление компонента %d: %w", e.id, err)
	}
	return true, nil
}

// syncInterfaces приводит интерфейсы КЕ, заведённые этим источником (attrs.source), в соответствие
// с записью: новые добавляются, отличающиеся обновляются, пропавшие отключаются (enabled=false) —
// не удаляются. Интерфейс с тем же именем, заведённый вручную, не перезаписывается.
func syncInterfaces(ctx context.Context, q querier, env *runEnv, ciID int64, ifs []canon.Interface) (bool, error) {
	if ifs == nil {
		return false, nil
	}
	var rows []orm.Params
	if _, err := q.RawWithCtx(ctx, `
		SELECT name, COALESCE(mac::text, '') AS mac, COALESCE(wwpn, '') AS wwpn, COALESCE(speed_mbps, 0) AS speed, enabled, COALESCE(attrs ->> 'role', '') AS role
		FROM cmdb.interface WHERE ci_id = ? AND attrs ->> 'source' = ?
	`, ciID, env.vendorKey).Values(&rows); err != nil {
		return false, fmt.Errorf("pipeline: чтение интерфейсов id=%d: %w", ciID, err)
	}
	type cur struct {
		mac, wwpn, role string
		speed           int64
		enabled         bool
	}
	existing := map[string]cur{}
	for _, r := range rows {
		var c cur
		fmt.Sscan(fmt.Sprint(r["speed"]), &c.speed)
		c.mac, c.wwpn, c.role, c.enabled = fmt.Sprint(r["mac"]), fmt.Sprint(r["wwpn"]), fmt.Sprint(r["role"]), fmt.Sprint(r["enabled"]) == "true"
		existing[fmt.Sprint(r["name"])] = c
	}

	changed := false
	seen := map[string]bool{}
	for _, i := range ifs {
		seen[i.Name] = true
		mac := ""
		if hw, err := net.ParseMAC(i.MAC); err == nil {
			mac = hw.String()
		}
		ifType := i.IfType
		if ifType == "" {
			ifType = "ethernet"
			if i.WWPN != "" {
				ifType = "fc"
			}
		}
		if c, ok := existing[i.Name]; ok {
			if c.mac == mac && c.wwpn == i.WWPN && c.speed == i.SpeedMbps && c.enabled == i.Enabled && c.role == i.Role {
				continue
			}
			if _, err := q.RawWithCtx(ctx, `
				UPDATE cmdb.interface SET mac = CAST(nullif(?, '') AS macaddr), wwpn = nullif(?, ''), speed_mbps = nullif(?, 0), enabled = ?,
				       attrs = jsonb_build_object('source', CAST(? AS text), 'role', CAST(? AS text)), updated_at = now(), updated_by = ?
				WHERE ci_id = ? AND name = ?
			`, mac, i.WWPN, i.SpeedMbps, i.Enabled, env.vendorKey, i.Role, env.userID, ciID, i.Name).Exec(); err != nil {
				return false, fmt.Errorf("pipeline: обновление интерфейса %s: %w", i.Name, err)
			}
			changed = true
			continue
		}
		res, err := q.RawWithCtx(ctx, `
			INSERT INTO cmdb.interface (ci_id, name, if_type, mac, wwpn, speed_mbps, enabled, attrs, created_by, updated_by)
			VALUES (?, ?, ?, CAST(nullif(?, '') AS macaddr), nullif(?, ''), nullif(?, 0), ?,
			        jsonb_build_object('source', CAST(? AS text), 'role', CAST(? AS text)), ?, ?)
			ON CONFLICT (ci_id, name) DO NOTHING
		`, ciID, i.Name, ifType, mac, i.WWPN, i.SpeedMbps, i.Enabled, env.vendorKey, i.Role, env.userID, env.userID).Exec()
		if err != nil {
			return false, fmt.Errorf("pipeline: вставка интерфейса %s: %w", i.Name, err)
		}
		if n, _ := res.RowsAffected(); n > 0 {
			changed = true
		}
	}

	for name, c := range existing {
		if seen[name] || !c.enabled {
			continue
		}
		if _, err := q.RawWithCtx(ctx, `UPDATE cmdb.interface SET enabled = false, updated_at = now(), updated_by = ? WHERE ci_id = ? AND name = ?`,
			env.userID, ciID, name).Exec(); err != nil {
			return false, fmt.Errorf("pipeline: отключение интерфейса %s: %w", name, err)
		}
		changed = true
	}
	return changed, nil
}
