// Чтение audit.event для страницы «Журнал действий» (plan.md E7.6: входы, экспорт, изменения
// ролей/правил/профиля именования, закрытие периода — всё, что уже пишет auditlog.Log). В отличие
// от internal/services/history.GlobalLog (audit.change_set/change_log — что изменилось в ДАННЫХ),
// здесь — что сделал пользователь с самой системой.
package auditlog

import (
	"context"
	"fmt"
	"strings"

	"cmdb/internal/repo"
)

// Filter — фильтры страницы журнала действий.
type Filter struct {
	EventType string
	UserID    int64  // 0 — любой
	Q         string // подстрока в event_type или details (ILIKE)
	From      string // дата ГГГГ-ММ-ДД, включительно
	To        string // дата ГГГГ-ММ-ДД, включительно
	Page      int
	PerPage   int
}

// Row — одна запись журнала для отображения/выгрузки.
type Row struct {
	ID        string
	At        string // YYYY-MM-DD HH:MI:SS UTC
	EventType string
	UserID    string // "" — система/аноним (user_id=0 у попыток входа с неизвестным именем)
	Username  string // "" — учётная запись не найдена (тоже относится к user_id=0)
	ClientIP  string
	Details   string // JSON текстом, как лежит в БД
}

// Result — страница журнала.
type Result struct {
	Rows    []Row
	Total   int
	Page    int
	PerPage int
	Pages   int
}

// List возвращает страницу audit.event по фильтру, от новых к старым.
func List(ctx context.Context, f Filter) (Result, error) {
	var conds []string
	var args []any
	add := func(cond string, a ...any) {
		conds = append(conds, cond)
		args = append(args, a...)
	}
	if f.EventType != "" {
		add("e.event_type = ?", f.EventType)
	}
	if f.UserID > 0 {
		add("e.user_id = ?", f.UserID)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		add("(e.event_type ILIKE ? OR e.details::text ILIKE ?)", "%"+q+"%", "%"+q+"%")
	}
	if f.From != "" {
		add("e.at >= ?::date", f.From)
	}
	if f.To != "" {
		add("e.at < (?::date + 1)", f.To)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	if f.PerPage <= 0 {
		f.PerPage = 50
	}
	if f.Page < 1 {
		f.Page = 1
	}

	res := Result{Page: f.Page, PerPage: f.PerPage}
	total, err := repo.QueryRow(ctx, `SELECT count(*)::text AS n FROM audit.event e `+where, args...)
	if err != nil {
		return res, fmt.Errorf("auditlog: подсчёт: %w", err)
	}
	fmt.Sscan(total["n"], &res.Total)
	res.Pages = (res.Total + f.PerPage - 1) / f.PerPage
	if res.Pages == 0 {
		res.Pages = 1
	}

	pageArgs := append(append([]any{}, args...), f.PerPage, (f.Page-1)*f.PerPage)
	rows, err := repo.QueryRows(ctx, `
		SELECT e.id::text AS id, to_char(e.at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') AS at, e.event_type,
		       COALESCE(e.user_id::text, '') AS user_id, COALESCE(u.username, '') AS username,
		       COALESCE(host(e.client_ip), '') AS client_ip, e.details::text AS details
		FROM audit.event e LEFT JOIN auth."user" u ON u.id = e.user_id
		`+where+` ORDER BY e.id DESC LIMIT ? OFFSET ?`, pageArgs...)
	if err != nil {
		return res, fmt.Errorf("auditlog: чтение: %w", err)
	}
	res.Rows = make([]Row, len(rows))
	for i, r := range rows {
		res.Rows[i] = Row{ID: r["id"], At: r["at"], EventType: r["event_type"], UserID: r["user_id"],
			Username: r["username"], ClientIP: r["client_ip"], Details: r["details"]}
	}
	return res, nil
}

// DistinctEventTypes — встречающиеся типы событий, для фильтра страницы.
func DistinctEventTypes(ctx context.Context) ([]string, error) {
	rows, err := repo.QueryRows(ctx, `SELECT DISTINCT event_type FROM audit.event ORDER BY event_type`)
	if err != nil {
		return nil, fmt.Errorf("auditlog: типы событий: %w", err)
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r["event_type"]
	}
	return out, nil
}
