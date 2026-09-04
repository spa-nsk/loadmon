package cmdb

import (
	"fmt"
	"net/smtp"
	"strings"

	"loadmon/config"

	models "loadmon/models"
)

// SendNotificationEmail отправляет email-уведомление через SMTP по
// параметрам из config.ConfigData (файл app2.json - см.
// config/config.go:JsonConfig, поля Smtp*/NotifyRecipients, добавлены
// 2026-09-04 по прямому запросу "реализовать возможность отправки
// уведомлений по электронной почте через smtp... параметры для отправки
// хранить в файле конфигурации").
//
// Используется net/smtp напрямую (без внешних зависимостей) - в проекте уже
// вендорится SMTP-клиент через beego (github.com/beego/beego/v2/core/
// utils.Email), но он не даёт ничего сверх net/smtp для простого текстового
// письма без вложений, поэтому решено не тащить лишнюю зависимость.
//
// Если SmtpHost или список получателей пуст - тихо ничего не делает и
// возвращает nil (уведомления считаются просто выключенными, это не
// ошибка). Вызывающий код (см. notifyHardwareEvent ниже) не должен падать/
// прерывать синхронизацию конфигурации из-за проблем с отправкой почты -
// поэтому все вызовы SendNotificationEmail в этом проекте оборачиваются в
// best-effort (ошибка только логируется).
func SendNotificationEmail(subject, body string) error {
	cfg := config.ConfigData
	if strings.TrimSpace(cfg.SmtpHost) == "" {
		return nil
	}
	recipients := make([]string, 0, len(cfg.NotifyRecipients))
	for _, r := range cfg.NotifyRecipients {
		if strings.TrimSpace(r) != "" {
			recipients = append(recipients, strings.TrimSpace(r))
		}
	}
	if len(recipients) == 0 {
		return nil
	}

	from := cfg.SmtpFrom
	if from == "" {
		from = "loadmon@localhost"
	}

	addr := fmt.Sprintf("%s:%d", cfg.SmtpHost, cfg.SmtpPortOrDefault())

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, strings.Join(recipients, ", "), subject, body)

	var auth smtp.Auth
	if cfg.SmtpUseAuth {
		auth = smtp.PlainAuth("", cfg.SmtpUsername, cfg.SmtpPassword, cfg.SmtpHost)
	}

	// SmtpUseTLS (STARTTLS) для простоты не реализован через ручной TLS-
	// хендшейк - net/smtp.SendMail поддерживает только неявный/без TLS
	// сценарий "AUTH после успешного соединения". Для внутренних SMTP-
	// релеев (типичный случай в закрытом контуре CMDB) STARTTLS обычно не
	// требуется; если потребуется - нужно заменить SendMail на ручной
	// smtp.Dial+StartTLS(см. также комментарий у поля SmtpUseTLS в
	// config/config.go).
	if err := smtp.SendMail(addr, auth, from, recipients, []byte(msg)); err != nil {
		return fmt.Errorf("отправка email через %s: %w", addr, err)
	}
	return nil
}

// notifyHardwareEvent - best-effort отправка email-уведомления об одной
// записи журнала изменений аппаратной конфигурации (models.HardwareEventLog,
// см. cmdb/hardwareeventlog.go). Вызывается из logHardwareEvent сразу после
// записи строки в БД. Ошибки отправки только логируются в консоль -
// недоступность/неверные настройки SMTP не должны прерывать синхронизацию
// конфигурации из OneView.
func notifyHardwareEvent(entry *models.HardwareEventLog) {
	if entry == nil || !config.ConfigData.NotifyOnHardwareChange {
		return
	}
	subject := fmt.Sprintf("loadmon: изменение конфигурации оборудования (%s)", entry.EntityKind)
	if err := SendNotificationEmail(subject, entry.Message); err != nil {
		fmt.Println("notifyHardwareEvent: ошибка отправки email-уведомления:", err)
	}
}
