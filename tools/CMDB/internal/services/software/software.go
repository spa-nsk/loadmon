// Package software — установленное ПО на КЕ (plan.md E7.7: cmdb.ci_software_install, схема — из
// E2/0013_cmdb_ci_extensions.sql). Не через internal/services/ci.UpsertExtension: та обслуживает
// расширения 1:1 (PK = ci_id), а на одной КЕ может быть много установок ПО, у cmdb.ci_software_install
// свой суррогатный id (см. ci.ExtensionTables — этой таблицы там нет, ci не путает 1:1 и 1:многие).
package software

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/platform/asof"
	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
)

// Install — одна установка ПО на КЕ со справочными полями ref.software (политика, признак
// отечественного происхождения) — вкладке «ПО» карточки КЕ и отчёту «запрещённое/неразрешённое
// ПО» (E7.7) нужны оба сразу, отдельным запросом в ref.software не пользуются.
type Install struct {
	ID          int64
	Version     string // версия конкретной установки; "" — берётся версия из справочника (RefVersion)
	InstallPath string
	DetectedAt  string // "" — установлено вручную, не источником (cmdb.ci_software_install.detected_at)
	RowVersion  int

	SoftwareID int64
	Vendor     string
	Product    string
	RefVersion string
	Edition    string
	Policy     string // allowed | forbidden | review
	IsDomestic bool
}

// ListForCI — установленное ПО на КЕ; с ?asof= — состояние на дату (E6.1). У cmdb.ci_software_install
// нет своей *_asof()-функции (E2.10 их строил только для internal/services/ci.ExtensionTables,
// расширений 1:1) — тот же приём UNION cmdb.* и hist.*, что internal/services/ci.extAsOf использует
// для расширений, только по нескольким строкам, а не одной.
func ListForCI(ctx context.Context, ciID int64) ([]Install, error) {
	from := `cmdb.ci_software_install si`
	fromArgs := []any{}
	if t, ok := asof.FromContext(ctx); ok {
		ts := repo.TimeArg(t)
		from = `(
			SELECT id, ci_id, software_id, version, install_path, detected_at, row_version
			FROM cmdb.ci_software_install WHERE ci_id = ? AND sys_period @> CAST(CAST(? AS text) AS timestamptz)
			UNION ALL
			SELECT id, ci_id, software_id, version, install_path, detected_at, row_version
			FROM hist.ci_software_install WHERE ci_id = ? AND sys_period @> CAST(CAST(? AS text) AS timestamptz)
		) si`
		fromArgs = []any{ciID, ts, ciID, ts}
	}
	rows, err := repo.QueryRows(ctx, `
		SELECT si.id::text AS id, si.software_id::text AS software_id, si.row_version::text AS row_version,
		       COALESCE(si.version, '') AS version, COALESCE(si.install_path, '') AS install_path,
		       COALESCE(to_char(si.detected_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS detected_at,
		       sw.vendor, sw.product, COALESCE(sw.version, '') AS ref_version, COALESCE(sw.edition, '') AS edition,
		       sw.policy, sw.is_domestic
		FROM `+from+` JOIN ref.software sw ON sw.id = si.software_id
		WHERE si.ci_id = ?
		ORDER BY sw.vendor, sw.product`, append(fromArgs, ciID)...)
	if err != nil {
		return nil, fmt.Errorf("software: список ПО КЕ id=%d: %w", ciID, err)
	}
	out := make([]Install, len(rows))
	for i, r := range rows {
		out[i] = Install{Version: r["version"], InstallPath: r["install_path"], DetectedAt: r["detected_at"],
			Vendor: r["vendor"], Product: r["product"], RefVersion: r["ref_version"], Edition: r["edition"],
			Policy: r["policy"], IsDomestic: r["is_domestic"] == "true"}
		fmt.Sscan(r["id"], &out[i].ID)
		fmt.Sscan(r["software_id"], &out[i].SoftwareID)
		fmt.Sscan(r["row_version"], &out[i].RowVersion)
	}
	return out, nil
}

// AddInput — регистрация установки ПО на КЕ вручную (карточка КЕ, вкладка «ПО»); импорт из
// источников делает то же самое пакетно через пайплайн (internal/importer/pipeline), не этой функцией.
type AddInput struct {
	CIID        int64
	SoftwareID  int64
	Version     string
	InstallPath string
	Reason      string
	UserID      int64
	Source      changectx.SourceKind // "" — SourceManual
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Add регистрирует установку. Дубль (та же КЕ+ПО+версия+путь, включая совпадающие NULL — таблица
// объявлена с NULLS NOT DISTINCT) отклоняется svcerr.ErrDuplicate, а не тихо игнорируется: то, что
// две установки выглядят как один и тот же дубль, надо заметить и разобраться, а не потерять из виду.
func Add(ctx context.Context, in AddInput) (id, changeSetID int64, err error) {
	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	changeSetID, err = changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		var exists int
		e := tx.RawWithCtx(ctx, `SELECT 1 FROM ref.software WHERE id = ?`, in.SoftwareID).QueryRow(&exists)
		if e == orm.ErrNoRows {
			return svcerr.NotFound("software", in.SoftwareID)
		}
		if e != nil {
			return fmt.Errorf("software: чтение справочника: %w", e)
		}
		e = tx.RawWithCtx(ctx, `SELECT 1 FROM cmdb.ci WHERE id = ?`, in.CIID).QueryRow(&exists)
		if e == orm.ErrNoRows {
			return svcerr.NotFound("ci", in.CIID)
		}
		if e != nil {
			return fmt.Errorf("software: чтение КЕ: %w", e)
		}
		e = tx.RawWithCtx(ctx, `
			INSERT INTO cmdb.ci_software_install (ci_id, software_id, version, install_path, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?, ?) RETURNING id
		`, in.CIID, in.SoftwareID, nullableString(in.Version), nullableString(in.InstallPath), in.UserID, in.UserID).QueryRow(&id)
		if svcerr.ViolatesConstraint(e, "ci_software_install_uniq") {
			return fmt.Errorf("%w: такая установка уже зарегистрирована на этой КЕ", svcerr.ErrDuplicate)
		}
		if e != nil {
			return fmt.Errorf("software: установка: %w", e)
		}
		return nil
	})
	return id, changeSetID, err
}

// RemoveInput — снятие записи об установке (ПО удалили с КЕ или запись была ошибочной). Жёсткое
// DELETE, не смена статуса — у cmdb.ci_software_install, в отличие от cmdb.component, нет понятия
// «снят, но виден в истории как объект»: сама история и так не теряется — hist.versioning()
// архивирует удаляемую строку в hist.ci_software_install с закрытым sys_period (проверено
// TestSoftwareInstall_VersioningAndUniqueness, internal/services/ci).
type RemoveInput struct {
	ID              int64
	ExpectedVersion int
	Reason          string
	UserID          int64
	Source          changectx.SourceKind
}

// Remove снимает запись об установке с оптимистической блокировкой.
func Remove(ctx context.Context, in RemoveInput) (int64, error) {
	meta := changectx.Meta{Source: in.Source.OrDefault(), UserID: in.UserID, Reason: in.Reason}
	return changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
		var version int
		e := tx.RawWithCtx(ctx, `SELECT row_version FROM cmdb.ci_software_install WHERE id = ? FOR UPDATE`, in.ID).QueryRow(&version)
		if e == orm.ErrNoRows {
			return svcerr.NotFound("ci_software_install", in.ID)
		}
		if e != nil {
			return fmt.Errorf("software: чтение установки: %w", e)
		}
		if version != in.ExpectedVersion {
			return svcerr.Conflict("ci_software_install", in.ID, in.ExpectedVersion, version)
		}
		if _, e := tx.RawWithCtx(ctx, `DELETE FROM cmdb.ci_software_install WHERE id = ?`, in.ID).Exec(); e != nil {
			return fmt.Errorf("software: удаление: %w", e)
		}
		return nil
	})
}
