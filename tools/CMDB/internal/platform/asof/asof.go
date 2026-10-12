// Package asof переносит параметр запроса ?asof= (plan.md E3.2: "asof-filter — параметр ?asof= в
// контексте запроса") через context.Context к сервисам, которые умеют отвечать на прошлый момент
// через *_asof() функции БД (E2.10). Сам глобальный переключатель «показать на дату» во всём
// UI/списках/графе — задача E6 (срезы и учётные периоды); здесь — только инфраструктура и один
// реальный потребитель уже сейчас: internal/services/ci.Get.
package asof

import (
	"context"
	"time"
)

type ctxKey struct{}

// WithTime возвращает контекст с привязанным моментом времени "как было на".
func WithTime(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

// FromContext достаёт момент времени из контекста, если ?asof= был передан и разобран.
func FromContext(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(ctxKey{}).(time.Time)
	return t, ok
}

// Parse разбирает значение параметра ?asof=: полный RFC3339 (2026-01-15T10:00:00Z) или дата
// (2026-01-15, трактуется как полночь UTC). Пустая строка и нераспознанный формат — ok=false
// (фильтр молча пропускает запрос без asof, а не отклоняет его — некорректный параметр не должен
// ронять обычный просмотр).
func Parse(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		return t, true
	}
	return time.Time{}, false
}
