package ci

import (
	"context"
	"fmt"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// UpdateInput — правка изменяемых атрибутов КЕ. nil-поле не меняется; для строковых полей ""
// сбрасывает значение в NULL. Имя, класс и стадия ЖЦ здесь не меняются: имя — через переименование
// по правилам naming, стадия — через ChangeLifecycle.
type UpdateInput struct {
	CIID            int64
	ExpectedVersion int // row_version, который видел пользователь (оптимистическая блокировка)
	Reason          string
	UserID          int64
	Source          changectx.SourceKind // "" — SourceManual (UI); REST API (E7.3) передаёт SourceAPI

	Description     *string
	Criticality     *string
	EnvironmentCode *string
	SerialNumber    *string
	AssetTag        *string
	RackPosition    *string
	Ownership       *string
	ServiceProvider *string
	KIICategory     *string
	LifecycleStatus *string
	KIIObject       *bool
}

// UpdateResult — новая версия строки и id change set.
type UpdateResult struct {
	RowVersion  int
	ChangeSetID int64
}

// Update применяет изменения одним UPDATE с условием row_version. Если версия в БД ушла вперёд,
// возвращает svcerr.ErrVersionConflict, если КЕ нет — svcerr.ErrNotFound.
func Update(ctx context.Context, in UpdateInput) (UpdateResult, error) {
	var sets []string
	var args []any
	addStr := func(col string, v *string) {
		if v != nil {
			sets = append(sets, col+" = ?")
			args = append(args, nullableString(*v))
		}
	}
	addStr("description", in.Description)
	addStr("criticality", in.Criticality)
	addStr("environment_code", in.EnvironmentCode)
	addStr("serial_number", in.SerialNumber)
	addStr("asset_tag", in.AssetTag)
	addStr("rack_position", in.RackPosition)
	addStr("ownership", in.Ownership)
	addStr("service_provider", in.ServiceProvider)
	addStr("kii_category", in.KIICategory)
	addStr("lifecycle_status", in.LifecycleStatus)
	if in.KIIObject != nil {
		sets = append(sets, "kii_object = ?")
		args = append(args, *in.KIIObject)
	}
	if len(sets) == 0 {
		return UpdateResult{}, svcerr.ErrNoChanges
	}
	sets = append(sets, "updated_at = now()", "updated_by = ?")
	args = append(args, in.UserID, in.CIID, in.ExpectedVersion)

	var res UpdateResult
	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	csID, err := changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		err := tx.RawWithCtx(ctx,
			`UPDATE cmdb.ci SET `+strings.Join(sets, ", ")+` WHERE id = ? AND row_version = ? RETURNING row_version`, args...,
		).QueryRow(&res.RowVersion)
		if err == nil {
			return nil
		}
		if err != orm.ErrNoRows {
			return fmt.Errorf("ci: обновление: %w", err)
		}
		// Ни одна строка не обновлена: КЕ нет или версия устарела.
		var actual int
		err = tx.RawWithCtx(ctx, `SELECT row_version FROM cmdb.ci WHERE id = ?`, in.CIID).QueryRow(&actual)
		if err == orm.ErrNoRows {
			return svcerr.NotFound("ci", in.CIID)
		}
		if err != nil {
			return fmt.Errorf("ci: чтение версии: %w", err)
		}
		return svcerr.Conflict("ci", in.CIID, in.ExpectedVersion, actual)
	})
	res.ChangeSetID = csID
	return res, err
}
