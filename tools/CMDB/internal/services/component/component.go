// Package component — операции с компонентами КЕ (диски, DIMM, платы, БП…): добавление, замена
// и перемещение между КЕ. Компоненты не удаляются: снятый компонент остаётся в cmdb.component со
// статусом 'replaced'/'removed', а прошлые состояния доступны через cmdb.component_asof.
package component

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/platform/asof"
	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// Статусы компонента (cmdb.component.status).
const (
	StatusInstalled = "installed"
	StatusReplaced  = "replaced" // заменён другим компонентом (attrs.replaced_by = uid нового)
	StatusRemoved   = "removed"
)

// Spec — описание физической детали; общее для добавления и замены.
type Spec struct {
	ModelID       *int64
	Manufacturer  string
	PartNumber    string
	SerialNumber  string
	Firmware      string
	CapacityValue *float64
	CapacityUnit  string
	Speed         string
}

// AddInput — установка нового компонента в КЕ.
type AddInput struct {
	CIID              int64
	ParentComponentID *int64 // вложенность: например, DIMM в плате, диск в контроллере
	ComponentType     string // ref.component_type.type_code
	Slot              string
	Spec              Spec
	Reason            string
	UserID            int64
	Source            changectx.SourceKind // "" — SourceManual (UI); REST API (E7.3) передаёт SourceAPI
}

// Add создаёт компонент со статусом installed и возвращает его id.
func Add(ctx context.Context, in AddInput) (int64, int64, error) {
	var id int64
	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	csID, err := changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		if in.ParentComponentID != nil {
			var parentCI int64
			err := tx.RawWithCtx(ctx, `SELECT ci_id FROM cmdb.component WHERE id = ?`, *in.ParentComponentID).QueryRow(&parentCI)
			if err == orm.ErrNoRows {
				return svcerr.NotFound("component", *in.ParentComponentID)
			}
			if err != nil {
				return fmt.Errorf("component: чтение родителя: %w", err)
			}
			if parentCI != in.CIID {
				return fmt.Errorf("%w: родительский компонент принадлежит другой КЕ", svcerr.ErrInvalidState)
			}
		}
		var err error
		id, _, err = insert(ctx, tx, in.CIID, in.ParentComponentID, in.ComponentType, in.Slot, in.Spec, in.UserID)
		return err
	})
	return id, csID, err
}

// ReplaceInput — замена компонента новой деталью в том же слоте той же КЕ.
type ReplaceInput struct {
	ComponentID     int64
	ExpectedVersion int // row_version заменяемого компонента
	New             Spec
	Reason          string
	UserID          int64
}

// ReplaceResult — идентификаторы после замены.
type ReplaceResult struct {
	NewComponentID int64
	ChangeSetID    int64
}

// Replace создаёт новый компонент (тот же тип, слот, родитель, КЕ), помечает старый статусом
// 'replaced' со ссылкой attrs.replaced_by и переносит вложенные компоненты на новый. Всё в одном
// change set — в истории замена видна как единое событие.
func Replace(ctx context.Context, in ReplaceInput) (ReplaceResult, error) {
	var res ReplaceResult
	meta := changectx.Meta{Source: changectx.SourceManual, UserID: in.UserID, Reason: in.Reason}
	csID, err := changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		cur, err := lockComponent(ctx, tx, in.ComponentID, in.ExpectedVersion)
		if err != nil {
			return err
		}
		if cur.status == StatusReplaced || cur.status == StatusRemoved {
			return fmt.Errorf("%w: компонент уже снят (status=%s)", svcerr.ErrInvalidState, cur.status)
		}

		newID, newUID, err := insert(ctx, tx, cur.ciID, cur.parentID, cur.componentType, cur.slot, in.New, in.UserID)
		if err != nil {
			return err
		}

		// Дочерние компоненты переезжают на новую деталь (пока старая ещё не помечена снятой).
		if _, err := tx.RawWithCtx(ctx, `
			UPDATE cmdb.component SET parent_component_id = ?, updated_at = now(), updated_by = ?
			WHERE parent_component_id = ? AND COALESCE(status, '') NOT IN ('replaced', 'removed')
		`, newID, in.UserID, in.ComponentID).Exec(); err != nil {
			return fmt.Errorf("component: перенос вложенных компонентов: %w", err)
		}

		if _, err := tx.RawWithCtx(ctx, `
			UPDATE cmdb.component
			SET status = ?, attrs = attrs || jsonb_build_object('replaced_by', ?::text),
			    updated_at = now(), updated_by = ?
			WHERE id = ?
		`, StatusReplaced, newUID, in.UserID, in.ComponentID).Exec(); err != nil {
			return fmt.Errorf("component: пометка заменённого компонента: %w", err)
		}
		res.NewComponentID = newID
		return nil
	})
	res.ChangeSetID = csID
	return res, err
}

// MoveInput — перенос компонента (вместе с вложенными) в другую КЕ.
type MoveInput struct {
	ComponentID     int64
	ExpectedVersion int
	ToCIID          int64
	ToSlot          string
	Reason          string
	UserID          int64
}

// Move переносит компонент в другую КЕ (или в другой слот той же): у корня переноса сбрасывается
// родитель и задаётся слот, все вложенные компоненты получают новый ci_id.
func Move(ctx context.Context, in MoveInput) (int64, error) {
	meta := changectx.Meta{Source: changectx.SourceManual, UserID: in.UserID, Reason: in.Reason}
	return changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		cur, err := lockComponent(ctx, tx, in.ComponentID, in.ExpectedVersion)
		if err != nil {
			return err
		}
		if cur.status == StatusReplaced || cur.status == StatusRemoved {
			return fmt.Errorf("%w: снятый компонент нельзя переносить (status=%s)", svcerr.ErrInvalidState, cur.status)
		}

		// Вложенные — отдельным запросом без корня: иначе корень обновился бы дважды в одной
		// транзакции и оставил в hist пустую версию с sys_period [t, t).
		if _, err := tx.RawWithCtx(ctx, `
			WITH RECURSIVE tree AS (
				SELECT id FROM cmdb.component WHERE parent_component_id = ?
				UNION ALL
				SELECT c.id FROM cmdb.component c JOIN tree t ON c.parent_component_id = t.id
			)
			UPDATE cmdb.component SET ci_id = ?, updated_at = now(), updated_by = ?
			WHERE id IN (SELECT id FROM tree)
		`, in.ComponentID, in.ToCIID, in.UserID).Exec(); err != nil {
			return fmt.Errorf("component: перенос вложенных компонентов: %w", err)
		}

		if _, err := tx.RawWithCtx(ctx, `
			UPDATE cmdb.component
			SET ci_id = ?, parent_component_id = NULL, slot = ?, updated_at = now(), updated_by = ?
			WHERE id = ?
		`, in.ToCIID, nullableString(in.ToSlot), in.UserID, in.ComponentID).Exec(); err != nil {
			return fmt.Errorf("component: перенос компонента: %w", err)
		}
		return nil
	})
}

type current struct {
	ciID          int64
	parentID      *int64
	componentType string
	slot          string
	status        string
}

// lockComponent блокирует строку и проверяет row_version (оптимистическая блокировка).
func lockComponent(ctx context.Context, tx orm.TxOrmer, id int64, expected int) (current, error) {
	var c current
	var version int
	var parent int64 // 0 = нет родителя (COALESCE: QueryRow beego не сканирует NULL)
	err := tx.RawWithCtx(ctx, `
		SELECT ci_id, COALESCE(parent_component_id, 0), component_type, COALESCE(slot, ''), COALESCE(status, ''), row_version
		FROM cmdb.component WHERE id = ? FOR UPDATE
	`, id).QueryRow(&c.ciID, &parent, &c.componentType, &c.slot, &c.status, &version)
	if err == orm.ErrNoRows {
		return c, svcerr.NotFound("component", id)
	}
	if err != nil {
		return c, fmt.Errorf("component: чтение компонента: %w", err)
	}
	if version != expected {
		return c, svcerr.Conflict("component", id, expected, version)
	}
	if parent != 0 {
		c.parentID = &parent
	}
	return c, nil
}

func insert(ctx context.Context, tx orm.TxOrmer, ciID int64, parentID *int64, componentType, slot string, s Spec, userID int64) (int64, string, error) {
	var id int64
	var uid string
	err := tx.RawWithCtx(ctx, `
		INSERT INTO cmdb.component
			(ci_id, parent_component_id, component_type, slot, model_id, manufacturer, part_number, serial_number,
			 firmware, capacity_value, capacity_unit, speed, status, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, uid::text
	`, ciID, repo.NullPtr(parentID), componentType, nullableString(slot), repo.NullPtr(s.ModelID), nullableString(s.Manufacturer),
		nullableString(s.PartNumber), nullableString(s.SerialNumber), nullableString(s.Firmware),
		repo.NullPtr(s.CapacityValue), nullableString(s.CapacityUnit), nullableString(s.Speed), StatusInstalled, userID, userID,
	).QueryRow(&id, &uid)
	if err != nil {
		return 0, "", fmt.Errorf("component: вставка: %w", err)
	}
	return id, uid, nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Item — компонент в дереве КЕ: строка и глубина вложенности.
type Item struct {
	repo.Row
	Depth int
}

// ListForCI возвращает компоненты КЕ деревом (родитель перед потомками, порядок по id внутри
// уровня). Снятые компоненты (replaced/removed) остаются в списке со своим статусом.
func ListForCI(ctx context.Context, ciID int64) ([]Item, error) {
	from, fromArgs := "cmdb.component c", []any(nil)
	if t, ok := asof.FromContext(ctx); ok { // «показать на дату» (E6.1): cmdb.component_asof(ts)
		from, fromArgs = "cmdb.component_asof(CAST(CAST(? AS text) AS timestamptz)) c", []any{repo.TimeArg(t)}
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT c.id::text AS id, COALESCE(c.parent_component_id::text, '') AS parent_id, c.row_version::text AS row_version,
		       c.component_type, COALESCE(t.name, c.component_type) AS type_name, COALESCE(c.slot, '') AS slot,
		       COALESCE(c.manufacturer, '') AS manufacturer, COALESCE(c.part_number, '') AS part_number,
		       COALESCE(c.serial_number, '') AS serial_number, COALESCE(c.firmware, '') AS firmware,
		       COALESCE(c.capacity_value::text || ' ' || COALESCE(c.capacity_unit, ''), '') AS capacity,
		       COALESCE(c.status, '') AS status
		FROM `+from+` LEFT JOIN ref.component_type t ON t.type_code = c.component_type
		WHERE c.ci_id = ? ORDER BY c.id`, append(fromArgs, ciID)...)
	if err != nil {
		return nil, err
	}
	children := map[string][]repo.Row{}
	ids := map[string]bool{}
	for _, r := range rows {
		ids[r["id"]] = true
	}
	var roots []repo.Row
	for _, r := range rows {
		if p := r["parent_id"]; p == "" || !ids[p] {
			roots = append(roots, r)
		} else {
			children[p] = append(children[p], r)
		}
	}
	var out []Item
	var walk func(rs []repo.Row, depth int)
	walk = func(rs []repo.Row, depth int) {
		for _, r := range rs {
			out = append(out, Item{Row: r, Depth: depth})
			walk(children[r["id"]], depth+1)
		}
	}
	walk(roots, 0)
	return out, nil
}
