// Выгрузка журнала действий в формате, который принимают SIEM без адаптера (plan.md E7.6:
// "экспорт в SIEM (syslog/CEF)"): ArcSight CEF внутри строки syslog (RFC 3164) — этот формат
// разбирает подавляющее большинство SIEM, включая распространённые в РФ (MaxPatrol SIEM, KUMA).
// Реального push-форвардинга (открытое соединение к приёмнику) здесь нет — только выгрузка файла
// по требованию тем же способом, что CSV-выгрузки остальных отчётов (E7.5): SIEM в реальных
// внедрениях почти всегда сам забирает файл по расписанию или ждёт его на shared-диске/SFTP, а не
// принимает соединения от источника.
package auditlog

import (
	"fmt"
	"strconv"
	"strings"
)

// cefSeverity — грубая категоризация серьёзности события для сортировки/триажа в SIEM (0–10,
// как того требует спецификация CEF): неудачный/заблокированный вход заметнее обычного логина,
// изменения ролей и правил — заметнее экспорта.
var cefSeverityRules = []struct {
	prefix string
	sev    int
}{
	{"login_locked", 8}, {"login_failed", 6}, {"login_denied", 6},
	{"role_", 5}, {"group_role", 5}, {"user_enable", 5}, {"user_create", 4}, {"user_password", 5},
	{"api_token_revoke", 5}, {"api_token_issue", 4}, {"service_account_create", 4},
	{"period_", 5}, {"ref_delete", 5}, {"ref_update", 4}, {"ref_create", 3}, {"ref_import", 4},
	{"baseline_delete", 5}, {"baseline_", 3},
	{"login", 1}, {"logout", 1},
}

func cefSeverity(eventType string) int {
	for _, r := range cefSeverityRules {
		if strings.HasPrefix(eventType, r.prefix) {
			return r.sev
		}
	}
	return 3
}

// cefEscapeHeader экранирует '\' и '|' в полях заголовка CEF (спецификация CEF: эти два символа
// обязательны к экранированию, иначе парсер SIEM рассинхронизируется на границах полей).
func cefEscapeHeader(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `|`, `\|`)
}

// cefEscapeExt экранирует '\', '=' и переводы строк в значениях extension-полей CEF.
func cefEscapeExt(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `=`, `\=`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return strings.ReplaceAll(s, "\r", "")
}

// FormatCEF форматирует одну запись журнала как строку syslog с полезной нагрузкой CEF:0.
// suser — username, если учётная запись ещё существует, иначе числовой user_id, иначе "system"
// (user_id=0 — действия без вошедшего пользователя, например неудачные попытки входа неизвестным
// именем). cs1/cs1Label — details целиком (в БД и так jsonb, детальный разбор SIEM не нужен: набор
// полей в details разный для каждого event_type, см. вызовы auditlog.Log по коду).
func FormatCEF(r Row) string {
	user := r.Username
	if user == "" {
		user = r.UserID
	}
	if user == "" || user == "0" {
		user = "system"
	}
	src := r.ClientIP
	if src == "" {
		src = "0.0.0.0"
	}
	ts := cefTimestamp(r.At)
	ext := fmt.Sprintf("rt=%s suser=%s src=%s cs1Label=details cs1=%s",
		ts, cefEscapeExt(user), src, cefEscapeExt(r.Details))
	// <134> = facility 16 (local0) * 8 + severity 6 (informational) — фиксированный уровень
	// транспорта syslog; действительная серьёзность события — в самом CEF (следующее поле).
	return fmt.Sprintf("<134>%s cmdb-web CEF:0|CMDB|cmdb-web|1.0|%s|%s|%d|%s",
		ts, cefEscapeHeader(r.EventType), cefEscapeHeader(r.EventType), cefSeverity(r.EventType), ext)
}

// cefTimestamp переводит "YYYY-MM-DD HH:MI:SS" (Row.At, уже в UTC) в "Mon Jan _2 HH:MM:SS UTC
// YYYY" — читается и как метка CEF rt=, и достаточно для syslog-заголовка; при ошибке разбора
// (не должно случаться — At приходит из to_char в query.go) возвращает исходную строку.
func cefTimestamp(at string) string {
	// "2026-09-27 05:41:50" -> без стороннего парсера: раскладываем руками, чтобы не тащить time
	// с часовым поясом для строки, которая и так уже в UTC текстом.
	if len(at) < 19 {
		return at
	}
	months := map[string]string{"01": "Jan", "02": "Feb", "03": "Mar", "04": "Apr", "05": "May", "06": "Jun",
		"07": "Jul", "08": "Aug", "09": "Sep", "10": "Oct", "11": "Nov", "12": "Dec"}
	year, mon, day, clock := at[0:4], at[5:7], at[8:10], at[11:19]
	m, ok := months[mon]
	if !ok {
		return at
	}
	d, err := strconv.Atoi(day)
	if err != nil {
		return at
	}
	return fmt.Sprintf("%s %2d %s UTC %s", m, d, clock, year)
}
