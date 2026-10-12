// Package history — чтение истории изменений КЕ из audit.change_set / audit.change_log для вкладки «История».
package history

import (
	"context"
	"strings"

	"cmdb/internal/platform/asof"
	"cmdb/internal/repo"
)

// Entry — одно изменение внутри набора: создание/удаление записи или замена значения атрибута.
type Entry struct {
	Entity string `json:"entity"` // таблица: ci, ci_server, component, ip_address, …
	Op     string `json:"op"`     // I, U, D
	Attr   string `json:"attr"`   // имя атрибута для U; пусто для I/D
	Old    string `json:"old"`    // JSON-значение (для I/D — снимок строки)
	New    string `json:"new"`
}

// ChangeSet — атомарный набор изменений: кто, когда, откуда и почему.
type ChangeSet struct {
	ID      string  `json:"id"`
	At      string  `json:"at"`
	Source  string  `json:"source"`
	UserID  string  `json:"user_id"`
	Reason  string  `json:"reason"`
	RFC     string  `json:"change_request_ref"`
	Entries []Entry `json:"entries"`
}

// ForCI возвращает последние limit наборов изменений, затронувших КЕ (саму КЕ, её расширения,
// компоненты, интерфейсы — всё, где audit.change_log.ci_id = id), от новых к старым.
func ForCI(ctx context.Context, ciID int64, limit int) ([]ChangeSet, error) {
	if limit <= 0 {
		limit = 50
	}
	// «Показать на дату» (E6.1): в истории — только наборы изменений, зафиксированные к этому моменту.
	cutoff, cutoffArgs := "", []any(nil)
	if t, ok := asof.FromContext(ctx); ok {
		cutoff = " AND COALESCE(cs.committed_at, cs.started_at) <= CAST(CAST(? AS text) AS timestamptz)"
		cutoffArgs = []any{repo.TimeArg(t)}
	}
	subCutoff := strings.ReplaceAll(cutoff, "cs.", "s.")
	args := append([]any{ciID, ciID}, cutoffArgs...)
	args = append(args, limit)
	rows, err := repo.QueryRows(ctx, `
		SELECT cs.id::text AS cs_id,
		       to_char(COALESCE(cs.committed_at, cs.started_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS at,
		       cs.source_kind, COALESCE(cs.user_id::text, '') AS user_id, COALESCE(cs.reason, '') AS reason,
		       COALESCE(cs.change_request_ref, '') AS rfc,
		       l.entity, l.op, COALESCE(l.attr_name, '') AS attr,
		       COALESCE(l.old_value::text, '') AS old_value, COALESCE(l.new_value::text, '') AS new_value
		FROM audit.change_log l
		JOIN audit.change_set cs ON cs.id = l.change_set_id
		WHERE l.ci_id = ?
		  AND l.change_set_id IN (
		      SELECT l2.change_set_id FROM audit.change_log l2 JOIN audit.change_set s ON s.id = l2.change_set_id
		      WHERE l2.ci_id = ?`+subCutoff+`
		      GROUP BY l2.change_set_id ORDER BY l2.change_set_id DESC LIMIT ?)
		ORDER BY cs.id DESC, l.id`, args...)
	if err != nil {
		return nil, err
	}

	var out []ChangeSet
	for _, r := range rows {
		if n := len(out); n == 0 || out[n-1].ID != r["cs_id"] {
			out = append(out, ChangeSet{ID: r["cs_id"], At: r["at"], Source: r["source_kind"], UserID: r["user_id"],
				Reason: r["reason"], RFC: r["rfc"]})
		}
		cs := &out[len(out)-1]
		cs.Entries = append(cs.Entries, Entry{Entity: r["entity"], Op: r["op"], Attr: r["attr"], Old: r["old_value"], New: r["new_value"]})
	}
	return out, nil
}
