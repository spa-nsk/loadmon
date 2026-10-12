package repo

import "time"

// NullPtr превращает необязательное значение в аргумент raw-SQL: nil → NULL, иначе само значение.
// Передавать в orm.Raw типизированный nil-указатель (например, (*int64)(nil)) нельзя — beego
// в getFlatParams разыменовывает указатель без проверки и паникует.
func NullPtr[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// TimeArg форматирует момент времени для параметра raw-SQL. Передавать time.Time в orm.Raw нельзя:
// beego форматирует его как "2006-01-02 15:04:05" в собственном часовом поясе — теряются доли
// секунды и зона. В SQL параметр нужно приводить явно: CAST(CAST(? AS text) AS timestamptz).
func TimeArg(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// NullTime — TimeArg для необязательного значения: nil → NULL.
func NullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return TimeArg(*t)
}
