package pipeline

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/canon"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
)

// identRule — один уровень цепочки идентификации класса (E4.6). Идентификатор берётся из
// canon.Record.Identifiers[key]; ExtCol — колонка таблицы-расширения класса, CoreCol — колонка cmdb.ci.
type identRule struct {
	Key       string
	ExtCol    string
	CoreCol   string
	IfaceRole string // MAC интерфейса с attrs.role = IfaceRole (BMC), таблица cmdb.interface
}

// identRules — правила по классам в порядке приоритета (intent §7.6). Для классов без записи
// (и после всех правил) действует последний уровень — имя (+площадка, если источник её сообщил).
// Правила зашиты в Go, а не в БД-таблице identification_rule — сознательное решение MVP.
var identRules = map[string][]identRule{
	"server":          {{Key: "bios_uuid", ExtCol: "bios_uuid"}, {Key: "serial_number", CoreCol: "serial_number"}, {Key: "bmc_mac", IfaceRole: "bmc"}},
	"vm":              {{Key: "instance_uuid", ExtCol: "instance_uuid"}, {Key: "bios_uuid", ExtCol: "bios_uuid"}},
	"vdi":             {{Key: "instance_uuid", ExtCol: "instance_uuid"}, {Key: "bios_uuid", ExtCol: "bios_uuid"}},
	"hypervisor_host": {{Key: "serial_number", CoreCol: "serial_number"}},
	"storage_array":   {{Key: "serial_number", CoreCol: "serial_number"}},
	"volume":          {{Key: "wwn", ExtCol: "wwn"}},
}

const notRetired = `c.lifecycle_stage NOT IN ('Retired', 'Disposed')`

// identify ищет существующую КЕ для rec: сначала по устойчивой связке источника (imp.match по
// ExternalID), затем по правилам класса и, наконец, по имени. Выведенные из эксплуатации КЕ
// (Retired/Disposed) исключаются — повторное обнаружение списанного актива не должно молча его
// "оживлять". Неоднозначность на любом уровне (>1 совпадение) — конфликт, без перехода к
// следующему уровню приоритета (без автослияния, как того явно требует E4.6).
func identify(ctx context.Context, q querier, env *runEnv, extTable string, rec canon.Record) (ciID int64, method string, ambiguous bool, err error) {
	try := func(method, query string, args ...any) (found bool, id int64, amb bool, err error) {
		id, amb, err = findOne(ctx, q, query, args...)
		return id != 0 || amb, id, amb, err
	}

	classCond, classArgs := "", []any{}
	if rec.Class != "" {
		classCond, classArgs = " AND c.ci_class = ?", []any{rec.Class}
	}

	if rec.ExternalID != "" {
		found, id, amb, err := try("external_id", `
			SELECT m.ci_id AS id FROM imp.match m JOIN cmdb.ci c ON c.id = m.ci_id
			WHERE m.connector_id = ? AND m.external_key = ? AND `+notRetired+classCond,
			append([]any{env.connectorID, "ext:" + rec.ExternalID}, classArgs...)...)
		if err != nil {
			return 0, "", false, err
		}
		if found {
			return id, "external_id", amb, nil
		}
	}

	for _, rule := range identRules[rec.Class] {
		v := rec.Identifiers[rule.Key]
		if v == "" {
			continue
		}
		var found bool
		var id int64
		var amb bool
		switch {
		case rule.ExtCol != "" && extTable != "":
			found, id, amb, err = try(rule.Key, `
				SELECT c.id FROM cmdb.ci c JOIN cmdb.`+extTable+` e ON e.ci_id = c.id
				WHERE c.ci_class = ? AND `+notRetired+` AND e.`+rule.ExtCol+`::text = lower(?)`, rec.Class, v)
		case rule.CoreCol != "":
			found, id, amb, err = try(rule.Key, `
				SELECT c.id FROM cmdb.ci c WHERE c.ci_class = ? AND `+notRetired+` AND lower(c.`+rule.CoreCol+`) = lower(?)`, rec.Class, v)
		case rule.IfaceRole != "":
			mac, perr := net.ParseMAC(v)
			if perr != nil {
				continue // некорректный MAC от источника не должен ронять идентификацию
			}
			found, id, amb, err = try(rule.Key, `
				SELECT DISTINCT c.id FROM cmdb.ci c JOIN cmdb.interface i ON i.ci_id = c.id
				WHERE c.ci_class = ? AND `+notRetired+` AND i.mac = CAST(? AS macaddr) AND i.attrs ->> 'role' = ?`,
				rec.Class, mac.String(), rule.IfaceRole)
		default:
			continue
		}
		if err != nil {
			return 0, "", false, err
		}
		if found {
			if !amb {
				if amb, err = ownedByOtherObject(ctx, q, env, id, rec); err != nil {
					return 0, "", false, err
				}
			}
			return id, rule.Key, amb, nil
		}
	}

	if rec.SourceName != "" {
		// Имя источника плюс альтернативы, которые коннектор знает про тот же объект: короткое имя
		// (первая метка FQDN — «srv01.corp.local» должен найти КЕ «srv01») и видимое имя.
		names := []string{rec.SourceName}
		for _, alt := range []string{rec.Identifiers["short_name"], rec.Identifiers["alias"]} {
			if alt != "" && !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, alt) }) {
				names = append(names, alt)
			}
		}
		conds := make([]string, len(names))
		args := append([]any{}, classArgs...)
		for i, n := range names {
			conds[i] = "lower(c.name) = lower(?)"
			args = append(args, n)
		}
		cond := "(" + strings.Join(conds, " OR ") + ")"
		query := `SELECT c.id FROM cmdb.ci c WHERE ` + notRetired + classCond + ` AND ` + cond
		// Имя — самый слабый признак: КЕ, у которой уже заполнен ДРУГОЙ сильный идентификатор
		// (иной instance_uuid, иной серийный номер), — это другой объект с тем же именем, а не та же КЕ.
		for _, rule := range identRules[rec.Class] {
			v := rec.Identifiers[rule.Key]
			switch {
			case v == "":
			case rule.ExtCol != "" && extTable != "":
				query += ` AND (NOT EXISTS (SELECT 1 FROM cmdb.` + extTable + ` e WHERE e.ci_id = c.id AND e.` + rule.ExtCol + ` IS NOT NULL)` +
					` OR EXISTS (SELECT 1 FROM cmdb.` + extTable + ` e WHERE e.ci_id = c.id AND e.` + rule.ExtCol + `::text = lower(?)))`
				args = append(args, v)
			case rule.CoreCol != "":
				query += ` AND (COALESCE(c.` + rule.CoreCol + `, '') = '' OR lower(c.` + rule.CoreCol + `) = lower(?))`
				args = append(args, v)
			}
		}
		if rec.LocationCode != "" {
			query += ` AND c.location_code = ?`
			args = append(args, rec.LocationCode)
		}
		found, id, amb, err := try("name_location", query, args...)
		if err != nil {
			return 0, "", false, err
		}
		if found {
			if !amb {
				if amb, err = ownedByOtherObject(ctx, q, env, id, rec); err != nil {
					return 0, "", false, err
				}
			}
			return id, "name_location", amb, nil
		}
	}
	return 0, "new", false, nil
}

// ownedByOtherObject — КЕ уже принадлежит ДРУГОМУ объекту этого источника: либо в этом же прогоне на
// неё указала другая запись, либо она связана (imp.match) с другим внешним ID. Два объекта источника
// с общим идентификатором (одинаковый серийный номер у разных хостов, клоны) — конфликт, а не одна КЕ:
// автослияния нет (E4.6). У записей без ExternalID (CSV) проверки нет.
func ownedByOtherObject(ctx context.Context, q querier, env *runEnv, ciID int64, rec canon.Record) (bool, error) {
	if rec.ExternalID == "" {
		return false, nil
	}
	if other, ok := env.claimed[ciID]; ok && other != rec.ExternalID {
		return true, nil
	}
	var rows []orm.Params
	if _, err := q.RawWithCtx(ctx, `SELECT external_key FROM imp.match WHERE connector_id = ? AND ci_id = ? AND external_key <> ?`,
		env.connectorID, ciID, "ext:"+rec.ExternalID).Values(&rows); err != nil {
		return false, fmt.Errorf("pipeline: проверка принадлежности КЕ id=%d: %w", ciID, err)
	}
	for _, r := range rows {
		// Прежняя связка учитывается, только если тот объект источника есть и в этом снимке; иначе это
		// переезд того же объекта (сменился адрес BMC) — связка перепривязывается, а не конфликт.
		if env.snapshotIDs[strings.TrimPrefix(fmt.Sprint(r["external_key"]), "ext:")] {
			return true, nil
		}
	}
	return false, nil
}

// findOne выполняет запрос, возвращающий колонку id, и сообщает: не найдено (0, false),
// единственное совпадение (id, false) или неоднозначность (0, true).
func findOne(ctx context.Context, q querier, query string, args ...any) (int64, bool, error) {
	var rows []orm.Params
	if _, err := q.RawWithCtx(ctx, query, args...).Values(&rows); err != nil {
		return 0, false, fmt.Errorf("pipeline: идентификация: %w", err)
	}
	switch len(rows) {
	case 0:
		return 0, false, nil
	case 1:
		var id int64
		fmt.Sscan(fmt.Sprint(rows[0]["id"]), &id)
		return id, false, nil
	default:
		return 0, true, nil
	}
}

// createCI вставляет новую КЕ и строку таблицы-расширения из record (E4.8). В отличие от ручного
// создания (internal/services/ci.Create) имя НЕ строится по seqalloc — источник даёт его как есть,
// импорт никогда не переименовывает (E4.5/intent §5.7); designation_code/sequence_number
// заполняются только когда имя уже само по себе соответствует активному профилю (naming.ParseHostName).
func createCI(ctx context.Context, q querier, env *runEnv, extTable string, rec canon.Record, namingRes naming.Result) (int64, error) {
	var namingProfileCode, designationCode, sequenceNumber any
	if namingRes.Status == naming.StatusCompliant {
		namingProfileCode = "CMDB-NC-2026"
		if p, err := naming.ParseHostName(rec.SourceName); err == nil {
			designationCode, sequenceNumber = p.Cls, p.Seq
		}
	}

	var ciID int64
	err := q.RawWithCtx(ctx, `
		INSERT INTO cmdb.ci
			(ci_class, name, source_name, naming_status, naming_profile_code, designation_code, sequence_number,
			 location_code, company_code, environment_code, serial_number, description, criticality,
			 first_seen_at, last_seen_at, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now(), now(), ?, ?)
		RETURNING id
	`, rec.Class, rec.SourceName, rec.SourceName, string(namingRes.Status), namingProfileCode, designationCode, sequenceNumber,
		nullableStr(rec.LocationCode), nullableStr(rec.CompanyCode), nullableStr(rec.EnvironmentCode),
		nullableStr(rec.SerialNumber), nullableStr(rec.Description), nullableStr(rec.Criticality), env.userID, env.userID,
	).QueryRow(&ciID)
	if err != nil {
		return 0, fmt.Errorf("pipeline: создание КЕ %q: %w", rec.SourceName, err)
	}

	extAttrs := map[string]any{}
	for k, v := range rec.Attrs {
		if v != nil && v != "" {
			extAttrs[k] = v
		}
	}
	if extTable != "" {
		if err := insertExtRow(ctx, q, extTable, ciID, extAttrs, env.userID); err != nil {
			return 0, err
		}
	}

	coreAttrs := map[string]any{}
	for _, name := range coreFields {
		if v := coreFieldValue(rec, name); v != "" {
			coreAttrs[name] = v
		}
	}
	if err := recordProvenance(ctx, q, ciID, extTable, coreAttrs, extAttrs, env.sourceSystemCode); err != nil {
		return 0, err
	}
	if _, err := applyVendor(ctx, q, env, ciID, rec.Vendor); err != nil {
		return 0, err
	}
	return ciID, nil
}

// insertExtRow создаёт строку таблицы-расширения новой КЕ (1:1, PK = ci_id). Если атрибутов нет,
// вставляется голая строка (тот же приём, что internal/services/ci/create_server.go применяет для
// ручного создания без атрибутов) — само 1:1-соответствие обязано существовать всегда. Если
// атрибуты есть, запись идёт через ci.UpsertExtensionTx — CAST-по-каталогу-БД логика, общая с
// ручным вводом, вместо повторной реализации приведения типов.
func insertExtRow(ctx context.Context, q querier, extTable string, ciID int64, attrs map[string]any, userID int64) error {
	if len(attrs) == 0 {
		if _, err := q.RawWithCtx(ctx, `INSERT INTO cmdb.`+extTable+` (ci_id, created_by, updated_by) VALUES (?, ?, ?)`,
			ciID, userID, userID).Exec(); err != nil {
			return fmt.Errorf("pipeline: создание строки расширения cmdb.%s: %w", extTable, err)
		}
		return nil
	}
	tx, ok := q.(orm.TxOrmer)
	if !ok {
		return fmt.Errorf("pipeline: запись атрибутов cmdb.%s требует транзакцию", extTable)
	}
	var res ci.ExtensionResult
	if err := ci.UpsertExtensionTx(ctx, tx, ci.ExtensionInput{CIID: ciID, Table: extTable, Attrs: attrs, UserID: userID}, &res); err != nil {
		return fmt.Errorf("pipeline: атрибуты cmdb.%s id=%d: %w", extTable, ciID, err)
	}
	return nil
}

// nullableStr — "" -> NULL: источник не сообщил значение, значит поле не заполнено, а не "пусто".
func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
