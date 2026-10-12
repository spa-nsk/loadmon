// Package changectx — единственная точка записи рабочих данных cmdb.* (ADR-003). Run открывает
// транзакцию, создаёт audit.change_set, устанавливает SET LOCAL cmdb.change_set_id (который
// требует триггер hist.versioning()), выполняет переданную функцию и фиксирует committed_at.
package changectx

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/svcerr"
)

// SourceKind — источник изменения (audit.change_set.source_kind).
type SourceKind string

const (
	SourceManual    SourceKind = "manual"
	SourceImport    SourceKind = "import"
	SourceAPI       SourceKind = "api"
	SourceSystem    SourceKind = "system"
	SourceMigration SourceKind = "migration"
)

// Meta — контекст изменения: кто/откуда/почему (intent §7.4).
type Meta struct {
	Source           SourceKind
	SourceSystemCode string // код источника для импорта, опционально
	ImportRunID      *int64
	UserID           int64
	ClientIP         string
	Reason           string // обязательна для SourceManual
	ChangeRequestRef string
	EffectiveAt      *time.Time // дата учёта; nil = сейчас
	IsBackdated      bool       // выставляется и автоматически, когда EffectiveAt заметно раньше момента записи
	// AllowClosedPeriod — правка с датой учёта в ЗАКРЫТОМ периоде разрешена (роль «Администратор
	// учёта», E6.5/E7.2); требует непустой Reason, change_set помечается is_backdated.
	AllowClosedPeriod bool
}

// backdatedAfter — насколько effective_at должен отставать от момента записи, чтобы правка считалась
// заданной задним числом (запас на расхождение часов и время выполнения транзакции).
const backdatedAfter = time.Minute

// OrDefault возвращает k, а для пустого значения — SourceManual: большинство вызовов сервисов
// приходит из UI (ручной ввод) и не должно на каждом месте писать источник явно; REST API (E7.3)
// и импорт передают свой SourceKind сами.
func (k SourceKind) OrDefault() SourceKind {
	if k == "" {
		return SourceManual
	}
	return k
}

func (m Meta) validate() error {
	if m.Source == "" {
		return fmt.Errorf("changectx: Source обязателен")
	}
	if m.Source == SourceManual && m.Reason == "" {
		return fmt.Errorf("changectx: Reason обязателен для ручного изменения (source=manual)")
	}
	return nil
}

// Run выполняет fn в транзакции с установленным audit.change_set. Возвращает id созданного
// change set (даже при ошибке fn, если change_set успел создаться — для диагностики).
func Run(ctx context.Context, m Meta, fn func(ctx context.Context, tx orm.TxOrmer) error) (int64, error) {
	if err := m.validate(); err != nil {
		return 0, err
	}

	var changeSetID int64
	o := orm.NewOrm()
	err := o.DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		effectiveAt := time.Now()
		if m.EffectiveAt != nil {
			effectiveAt = *m.EffectiveAt
		}
		backdated := m.IsBackdated || effectiveAt.Before(time.Now().Add(-backdatedAfter))

		// Закрытые учётные периоды (E6.5): дата учёта в закрытом месяце — только с явным разрешением и причиной.
		var closed int
		if err := tx.RawWithCtx(ctx, `
			SELECT count(*) FROM rpt.accounting_period
			WHERE status = 'closed' AND period = date_trunc('month', CAST(CAST(? AS text) AS timestamptz) AT TIME ZONE 'UTC')::date
		`, repo.TimeArg(effectiveAt)).QueryRow(&closed); err != nil {
			return fmt.Errorf("changectx: проверка учётного периода: %w", err)
		}
		if closed > 0 {
			switch {
			case !m.AllowClosedPeriod:
				return fmt.Errorf("%w: дата учёта %s попадает в закрытый период — изменение отклонено",
					svcerr.ErrPeriodClosed, effectiveAt.UTC().Format("2006-01-02"))
			case m.Reason == "":
				return fmt.Errorf("%w: правка закрытого периода требует причины", svcerr.ErrPeriodClosed)
			}
			backdated = true
		}

		err := tx.RawWithCtx(ctx, `
			INSERT INTO audit.change_set
				(source_kind, source_system_code, import_run_id, user_id, client_ip, reason, change_request_ref, effective_at, is_backdated)
			VALUES (?, ?, ?, ?, nullif(?, '')::inet, ?, ?, CAST(CAST(? AS text) AS timestamptz), ?)
			RETURNING id
		`, string(m.Source), nullableString(m.SourceSystemCode), repo.NullPtr(m.ImportRunID), m.UserID, m.ClientIP,
			nullableString(m.Reason), nullableString(m.ChangeRequestRef), repo.TimeArg(effectiveAt), backdated,
		).QueryRow(&changeSetID)
		if err != nil {
			return fmt.Errorf("changectx: создание change_set: %w", err)
		}

		if _, err := tx.RawWithCtx(ctx, `SELECT set_config('cmdb.change_set_id', ?, true)`, strconv.FormatInt(changeSetID, 10)).Exec(); err != nil {
			return fmt.Errorf("changectx: set_config cmdb.change_set_id: %w", err)
		}

		if err := fn(ctx, tx); err != nil {
			return err
		}

		if _, err := tx.RawWithCtx(ctx, `UPDATE audit.change_set SET committed_at = now() WHERE id = ?`, changeSetID).Exec(); err != nil {
			return fmt.Errorf("changectx: фиксация committed_at: %w", err)
		}
		return nil
	})
	return changeSetID, err
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
