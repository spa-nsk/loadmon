// Package auditlog пишет audit.event — журнал действий, не данных (просмотры, экспорт; в
// перспективе входы, изменения ролей и т.п., E7.6 — "требование приказа ФСТЭК №117"). В отличие
// от audit.change_set/change_log (internal/services/changectx, кто/когда/почему изменил ДАННЫЕ),
// это отдельная, только-добавляемая таблица о действиях над системой (внутри и вне ChangeContext).
package auditlog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
)

// Log добавляет запись в audit.event. Ошибка логирования не должна мешать основному действию
// пользователя (открыть карточку, скачать экспорт) — вызывающая сторона решает сама, что делать
// с ошибкой (обычно — записать в slog и продолжить), Log её только возвращает.
func Log(ctx context.Context, userID int64, clientIP, eventType string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	b, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("auditlog: сериализация details: %w", err)
	}
	_, err = orm.NewOrm().RawWithCtx(ctx, `
		INSERT INTO audit.event (user_id, client_ip, event_type, details)
		VALUES (?, nullif(?, '')::inet, ?, CAST(? AS jsonb))
	`, userID, clientIP, eventType, string(b)).Exec()
	if err != nil {
		return fmt.Errorf("auditlog: запись audit.event: %w", err)
	}
	return nil
}
