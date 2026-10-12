// Package relation — типизированные связи между КЕ с периодом действия (cmdb.ci_rel). Связь не
// удаляется: она закрывается (верхняя граница valid_period), а прошлые состояния топологии
// остаются доступны по valid_period и через cmdb.ci_rel_asof.
package relation

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// Имя ограничения WITHOUT OVERLAPS из миграции 0008 — по нему распознаём дубликат.
const overlapConstraint = "ci_rel_no_overlap"

// CreateInput — создание связи src --rel_type--> dst.
type CreateInput struct {
	RelType          string // ref.rel_type.rel_code
	SrcCIID          int64
	DstCIID          int64
	SrcComponentID   *int64
	DstComponentID   *int64
	Attrs            map[string]any
	ValidFrom        *time.Time // nil = сейчас
	SourceSystemCode string
	Reason           string
	UserID           int64
	Source           changectx.SourceKind // "" — SourceManual (UI); REST API (E7.3) передаёт SourceAPI
}

// Create создаёт связь. Проверяет, что обе КЕ существуют и не выведены из эксплуатации, а
// такая же связь (тип + пара КЕ) не действует одновременно (ErrDuplicate).
func Create(ctx context.Context, in CreateInput) (id, changeSetID int64, err error) {
	if in.SrcCIID == in.DstCIID {
		return 0, 0, fmt.Errorf("%w: связь КЕ с самой собой", svcerr.ErrInvalidState)
	}
	attrs := []byte("{}")
	if len(in.Attrs) > 0 {
		if attrs, err = json.Marshal(in.Attrs); err != nil {
			return 0, 0, fmt.Errorf("relation: attrs: %w", err)
		}
	}

	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	changeSetID, err = changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		for _, ciID := range []int64{in.SrcCIID, in.DstCIID} {
			var stage string
			e := tx.RawWithCtx(ctx, `SELECT lifecycle_stage FROM cmdb.ci WHERE id = ?`, ciID).QueryRow(&stage)
			if e == orm.ErrNoRows {
				return svcerr.NotFound("ci", ciID)
			}
			if e != nil {
				return fmt.Errorf("relation: чтение КЕ: %w", e)
			}
			if stage == "Retired" || stage == "Disposed" {
				return fmt.Errorf("%w: КЕ id=%d выведена из эксплуатации (%s)", svcerr.ErrInvalidState, ciID, stage)
			}
		}

		e := tx.RawWithCtx(ctx, `
			INSERT INTO cmdb.ci_rel
				(rel_type, src_ci_id, dst_ci_id, src_component_id, dst_component_id, attrs, valid_period,
				 source_system_code, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?, CAST(? AS jsonb), tstzrange(COALESCE(CAST(CAST(? AS text) AS timestamptz), now()), NULL), ?, ?, ?)
			RETURNING id
		`, in.RelType, in.SrcCIID, in.DstCIID, repo.NullPtr(in.SrcComponentID), repo.NullPtr(in.DstComponentID),
			string(attrs), repo.NullTime(in.ValidFrom), nullableString(in.SourceSystemCode), in.UserID, in.UserID,
		).QueryRow(&id)
		switch {
		case svcerr.ViolatesConstraint(e, overlapConstraint):
			return fmt.Errorf("%w: связь %s между КЕ %d и %d уже действует в этот период", svcerr.ErrDuplicate, in.RelType, in.SrcCIID, in.DstCIID)
		case e != nil:
			return fmt.Errorf("relation: вставка: %w", e)
		}
		return nil
	})
	return id, changeSetID, err
}

// CloseInput — завершение действия связи.
type CloseInput struct {
	RelID           int64
	ExpectedVersion int
	EndAt           *time.Time // nil = сейчас
	Reason          string
	UserID          int64
	Source          changectx.SourceKind // "" — SourceManual (UI); REST API (E7.3) передаёт SourceAPI
}

// Close закрывает действующую связь: valid_period получает верхнюю границу. Уже закрытую связь
// закрыть повторно нельзя (ErrInvalidState).
func Close(ctx context.Context, in CloseInput) (int64, error) {
	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	return changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		var version int
		var open int
		err := tx.RawWithCtx(ctx, `SELECT row_version, CASE WHEN upper_inf(valid_period) THEN 1 ELSE 0 END FROM cmdb.ci_rel WHERE id = ? FOR UPDATE`, in.RelID).
			QueryRow(&version, &open)
		if err == orm.ErrNoRows {
			return svcerr.NotFound("ci_rel", in.RelID)
		}
		if err != nil {
			return fmt.Errorf("relation: чтение связи: %w", err)
		}
		if version != in.ExpectedVersion {
			return svcerr.Conflict("ci_rel", in.RelID, in.ExpectedVersion, version)
		}
		if open == 0 {
			return fmt.Errorf("%w: связь уже закрыта", svcerr.ErrInvalidState)
		}

		// greatest(...) не даёт верхней границе стать меньше нижней (закрытие «задним числом»
		// раньше начала — вырождается в пустой период).
		_, err = tx.RawWithCtx(ctx, `
			UPDATE cmdb.ci_rel
			SET valid_period = tstzrange(lower(valid_period), greatest(COALESCE(CAST(CAST(? AS text) AS timestamptz), now()), lower(valid_period))),
			    updated_at = now(), updated_by = ?
			WHERE id = ?
		`, repo.NullTime(in.EndAt), in.UserID, in.RelID).Exec()
		if err != nil {
			return fmt.Errorf("relation: закрытие: %w", err)
		}
		return nil
	})
}

// Relation — связь в виде, удобном для отображения соседей КЕ.
type Relation struct {
	ID        int64  `json:"id"`
	Version   int    `json:"version"`
	RelType   string `json:"rel_type"`
	Label     string `json:"label"` // название связи с точки зрения запрошенной КЕ («работает на» / «размещает»)
	SrcCIID   int64  `json:"src_ci_id"`
	DstCIID   int64  `json:"dst_ci_id"`
	PeerID    int64  `json:"peer_id"`
	PeerName  string `json:"peer_name"`
	PeerClass string `json:"peer_class"`
	Outgoing  bool   `json:"outgoing"` // true — КЕ является источником связи, false — приёмником
	ValidFrom string `json:"valid_from"`
	ValidTo   string `json:"valid_to"` // "" — действует
}

// ListForCI возвращает связи КЕ, действовавшие в момент asOf (nil = сейчас), в обе стороны.
func ListForCI(ctx context.Context, ciID int64, asOf *time.Time) ([]Relation, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT r.id::text AS id, r.row_version::text AS version, r.rel_type,
		       CASE WHEN r.src_ci_id = ? THEN COALESCE(t.name_fwd, r.rel_type) ELSE COALESCE(t.name_rev, r.rel_type) END AS label,
		       r.src_ci_id::text AS src, r.dst_ci_id::text AS dst, p.id::text AS peer_id, p.name AS peer_name, p.ci_class AS peer_class,
		       to_char(lower(r.valid_period) AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS valid_from,
		       COALESCE(to_char(upper(r.valid_period) AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS valid_to
		FROM cmdb.ci_rel r
		JOIN cmdb.ci p ON p.id = CASE WHEN r.src_ci_id = ? THEN r.dst_ci_id ELSE r.src_ci_id END
		LEFT JOIN ref.rel_type t ON t.rel_code = r.rel_type
		WHERE (r.src_ci_id = ? OR r.dst_ci_id = ?)
		  AND r.valid_period @> COALESCE(CAST(CAST(? AS text) AS timestamptz), now())
		ORDER BY r.rel_type, r.id`, ciID, ciID, ciID, ciID, repo.NullTime(asOf))
	if err != nil {
		return nil, fmt.Errorf("relation: список связей: %w", err)
	}

	out := make([]Relation, 0, len(rows))
	for _, r := range rows {
		rel := Relation{RelType: r["rel_type"], Label: r["label"], PeerName: r["peer_name"], PeerClass: r["peer_class"],
			ValidFrom: r["valid_from"], ValidTo: r["valid_to"]}
		rel.ID, _ = strconv.ParseInt(r["id"], 10, 64)
		rel.Version, _ = strconv.Atoi(r["version"])
		rel.SrcCIID, _ = strconv.ParseInt(r["src"], 10, 64)
		rel.DstCIID, _ = strconv.ParseInt(r["dst"], 10, 64)
		rel.PeerID, _ = strconv.ParseInt(r["peer_id"], 10, 64)
		rel.Outgoing = rel.SrcCIID == ciID
		out = append(out, rel)
	}
	return out, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
