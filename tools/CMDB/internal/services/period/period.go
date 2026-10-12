// Package period — учётные периоды (plan.md E6.5, intent §7.5 п.3): закрытие и переоткрытие
// календарных месяцев (UTC) и отчёт о правках задним числом. Сама защита закрытых периодов — в
// internal/services/changectx.Run (проверка effective_at каждого change set).
package period

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/auditlog"
	"cmdb/internal/services/svcerr"
)

// MonthStart — первое число месяца (UTC) для момента t.
func MonthStart(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// ParseMonth разбирает «ГГГГ-ММ» (или полную дату — берётся её месяц).
func ParseMonth(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{"2006-01", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return MonthStart(t), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: период %q (ожидается ГГГГ-ММ)", svcerr.ErrInvalidInput, s)
}

// Period — учётный период в списке.
type Period struct {
	Month        string // ГГГГ-ММ
	Status       string // closed | open
	ClosedAt     string
	ClosedBy     string
	CloseReason  string
	ReopenedAt   string
	ReopenedBy   string
	ReopenReason string
}

// List возвращает периоды, когда-либо закрывавшиеся, от новых к старым.
func List(ctx context.Context) ([]Period, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT to_char(period, 'YYYY-MM') AS month, status,
		       to_char(closed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS closed_at, COALESCE(closed_by::text, '') AS closed_by,
		       COALESCE(close_reason, '') AS close_reason,
		       COALESCE(to_char(reopened_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'), '') AS reopened_at,
		       COALESCE(reopened_by::text, '') AS reopened_by, COALESCE(reopen_reason, '') AS reopen_reason
		FROM rpt.accounting_period ORDER BY period DESC`)
	if err != nil {
		return nil, fmt.Errorf("period: чтение периодов: %w", err)
	}
	out := make([]Period, len(rows))
	for i, r := range rows {
		out[i] = Period{Month: r["month"], Status: r["status"], ClosedAt: r["closed_at"], ClosedBy: r["closed_by"], CloseReason: r["close_reason"],
			ReopenedAt: r["reopened_at"], ReopenedBy: r["reopened_by"], ReopenReason: r["reopen_reason"]}
	}
	return out, nil
}

// IsClosed сообщает, закрыт ли период, в который попадает момент t.
func IsClosed(ctx context.Context, t time.Time) (bool, error) {
	row, err := repo.QueryRow(ctx, `SELECT count(*)::text AS n FROM rpt.accounting_period WHERE status = 'closed' AND period = CAST(? AS date)`,
		MonthStart(t).Format("2006-01-02"))
	if err != nil {
		return false, err
	}
	return row["n"] == "1", nil
}

// Close закрывает период. Закрыть можно только уже закончившийся месяц: закрытие текущего заблокировало
// бы запись любых изменений «сейчас», включая импорт. Повторное закрытие закрытого — ErrInvalidState.
// Действие пишется в audit.event (E7.6).
func Close(ctx context.Context, month time.Time, userID int64, reason string) error {
	month = MonthStart(month)
	if month.AddDate(0, 1, 0).After(time.Now().UTC()) {
		return fmt.Errorf("%w: период %s ещё не закончился", svcerr.ErrInvalidState, month.Format("2006-01"))
	}
	res, err := execRows(ctx, `
		INSERT INTO rpt.accounting_period (period, status, closed_at, closed_by, close_reason)
		VALUES (CAST(? AS date), 'closed', now(), ?, nullif(?, ''))
		ON CONFLICT (period) DO UPDATE
		SET status = 'closed', closed_at = now(), closed_by = EXCLUDED.closed_by, close_reason = EXCLUDED.close_reason
		WHERE rpt.accounting_period.status = 'open'
	`, month.Format("2006-01-02"), userID, reason)
	if err != nil {
		return fmt.Errorf("period: закрытие: %w", err)
	}
	if res == 0 {
		return fmt.Errorf("%w: период %s уже закрыт", svcerr.ErrInvalidState, month.Format("2006-01"))
	}
	return auditlog.Log(ctx, userID, "", "period_close", map[string]any{"period": month.Format("2006-01"), "reason": reason})
}

// Reopen переоткрывает закрытый период; причина обязательна.
func Reopen(ctx context.Context, month time.Time, userID int64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("%w: переоткрытие периода требует причины", svcerr.ErrInvalidInput)
	}
	month = MonthStart(month)
	res, err := execRows(ctx, `
		UPDATE rpt.accounting_period SET status = 'open', reopened_at = now(), reopened_by = ?, reopen_reason = ?
		WHERE period = CAST(? AS date) AND status = 'closed'
	`, userID, reason, month.Format("2006-01-02"))
	if err != nil {
		return fmt.Errorf("period: переоткрытие: %w", err)
	}
	if res == 0 {
		return fmt.Errorf("%w: период %s не закрыт", svcerr.ErrInvalidState, month.Format("2006-01"))
	}
	return auditlog.Log(ctx, userID, "", "period_reopen", map[string]any{"period": month.Format("2006-01"), "reason": reason})
}

// Backdated — правка задним числом: change set с датой учёта заметно раньше момента записи.
type Backdated struct {
	ChangeSetID  string
	EffectiveAt  string
	StartedAt    string
	UserID       string
	Source       string
	Reason       string
	Entries      string
	ClosedPeriod bool // правка внутри закрытого периода (разрешённая администратором учёта)
}

// BackdatedReport — отчёт о правках задним числом за интервал [from, to) по дате записи (E6.5).
func BackdatedReport(ctx context.Context, from, to time.Time) ([]Backdated, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT cs.id::text AS id,
		       to_char(cs.effective_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS effective_at,
		       to_char(cs.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS started_at,
		       COALESCE(cs.user_id::text, '') AS user_id, cs.source_kind, COALESCE(cs.reason, '') AS reason,
		       (SELECT count(*) FROM audit.change_log l WHERE l.change_set_id = cs.id)::text AS n,
		       EXISTS (SELECT 1 FROM rpt.accounting_period p WHERE p.status = 'closed'
		               AND p.period = date_trunc('month', cs.effective_at AT TIME ZONE 'UTC')::date) AS closed_period
		FROM audit.change_set cs
		WHERE cs.is_backdated AND cs.started_at >= CAST(CAST(? AS text) AS timestamptz) AND cs.started_at < CAST(CAST(? AS text) AS timestamptz)
		ORDER BY cs.id DESC`, repo.TimeArg(from), repo.TimeArg(to))
	if err != nil {
		return nil, fmt.Errorf("period: отчёт о правках задним числом: %w", err)
	}
	out := make([]Backdated, len(rows))
	for i, r := range rows {
		out[i] = Backdated{ChangeSetID: r["id"], EffectiveAt: r["effective_at"], StartedAt: r["started_at"], UserID: r["user_id"],
			Source: r["source_kind"], Reason: r["reason"], Entries: r["n"], ClosedPeriod: r["closed_period"] == "true"}
	}
	return out, nil
}

func execRows(ctx context.Context, query string, args ...any) (int64, error) {
	res, err := orm.NewOrm().RawWithCtx(ctx, query, args...).Exec()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
