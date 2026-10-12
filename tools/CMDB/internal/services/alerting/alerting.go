// Package alerting — обнаружение и уведомление об операционных проблемах импорта (plan.md E8.6:
// «алерты — падение импорта, рост конфликтов»). Источник фактов — imp.run/imp.conflict (те же
// таблицы, что показывает /import), находка становится и метрикой Prometheus (для внешнего
// Alertmanager/Zabbix — org-специфичный канал, которого у этого проекта нет), и, если настроен
// SMTP, письмом получателям CMDB_ALERT_EMAILS. Оба канала не исключают друг друга: метрика — для
// организаций с уже развёрнутым мониторингом, письмо — минимально достаточный канал «из коробки»
// (тот же mailpit из docker-compose.yml, что и для писем в проде — вместо угадывания стека
// оповещений заказчика, единственная разумная зависимость — SMTP, есть почти везде).
package alerting

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"cmdb/internal/platform/logging"
	"cmdb/internal/platform/metrics"
	"cmdb/internal/repo"
)

// Thresholds — пороги срабатывания; переопределяются переменными окружения.
type Thresholds struct {
	ConsecutiveFailures int           // подряд неудачных прогонов одного источника, по умолчанию 3
	NewConflicts        int           // новых открытых конфликтов за Window, по умолчанию 20
	Window              time.Duration // окно подсчёта новых конфликтов, по умолчанию 24 ч
}

// DefaultThresholds — CMDB_ALERT_CONSECUTIVE_FAILURES, CMDB_ALERT_NEW_CONFLICTS, CMDB_ALERT_WINDOW_HOURS.
func DefaultThresholds() Thresholds {
	return Thresholds{
		ConsecutiveFailures: envInt("CMDB_ALERT_CONSECUTIVE_FAILURES", 3),
		NewConflicts:        envInt("CMDB_ALERT_NEW_CONFLICTS", 20),
		Window:              time.Duration(envInt("CMDB_ALERT_WINDOW_HOURS", 24)) * time.Hour,
	}
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// Finding — одна обнаруженная проблема.
type Finding struct {
	Kind    string // "import_failing" | "conflicts_growing"
	Subject string // код источника (import_failing) либо "" (conflicts_growing — по всей системе)
	Count   int
	Message string
}

// Check ищет проблемы по текущему состоянию imp.run/imp.conflict и обновляет метрики Prometheus
// (последнее — независимо от того, настроено ли уведомление письмом: метрики нужны и когда SMTP
// не настроен вовсе).
func Check(ctx context.Context, th Thresholds) ([]Finding, error) {
	var findings []Finding

	streaks, err := consecutiveFailures(ctx)
	if err != nil {
		return nil, fmt.Errorf("alerting: подряд неудачные прогоны: %w", err)
	}
	metrics.ImportConsecutiveFailures.Reset()
	for code, n := range streaks {
		metrics.ImportConsecutiveFailures.WithLabelValues(code).Set(float64(n))
		if n >= th.ConsecutiveFailures {
			findings = append(findings, Finding{Kind: "import_failing", Subject: code, Count: n,
				Message: fmt.Sprintf("источник %q: %d прогонов подряд завершились ошибкой", code, n)})
		}
	}

	openTotal, err := openConflictsTotal(ctx)
	if err != nil {
		return nil, fmt.Errorf("alerting: открытые конфликты: %w", err)
	}
	metrics.OpenConflictsTotal.Set(float64(openTotal))

	newCount, err := newOpenConflicts(ctx, th.Window)
	if err != nil {
		return nil, fmt.Errorf("alerting: новые конфликты: %w", err)
	}
	if newCount >= th.NewConflicts {
		findings = append(findings, Finding{Kind: "conflicts_growing", Count: newCount,
			Message: fmt.Sprintf("новых открытых конфликтов за %s: %d (всего открытых: %d)", th.Window, newCount, openTotal)})
	}

	return findings, nil
}

// consecutiveFailures — для каждого включённого источника длина серии подряд идущих 'failed' в
// последних прогонах (dry-run не в счёт — это не реальный опрос источника), считая от самого
// свежего прогона назад до первого не-'failed'.
func consecutiveFailures(ctx context.Context) (map[string]int, error) {
	rows, err := repo.QueryRows(ctx, `
		SELECT connector_code, status, rn FROM (
			SELECT c.connector_code, r.status,
			       row_number() OVER (PARTITION BY c.id ORDER BY r.started_at DESC) AS rn
			FROM imp.connector c JOIN imp.run r ON r.connector_id = c.id AND r.dry_run = false
			WHERE c.enabled
		) t
		WHERE rn <= 50
		ORDER BY connector_code, rn`)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	stopped := map[string]bool{}
	for _, r := range rows {
		code := r["connector_code"]
		if stopped[code] {
			continue
		}
		if r["status"] == "failed" {
			out[code]++
		} else {
			stopped[code] = true
		}
	}
	return out, nil
}

func openConflictsTotal(ctx context.Context) (int, error) {
	row, err := repo.QueryRow(ctx, `SELECT count(*) AS n FROM imp.conflict WHERE status = 'open'`)
	if err != nil || row == nil {
		return 0, err
	}
	n, _ := strconv.Atoi(row["n"])
	return n, nil
}

func newOpenConflicts(ctx context.Context, window time.Duration) (int, error) {
	row, err := repo.QueryRow(ctx, `SELECT count(*) AS n FROM imp.conflict WHERE status = 'open' AND created_at >= now() - ?::interval`,
		fmt.Sprintf("%d seconds", int(window.Seconds())))
	if err != nil || row == nil {
		return 0, err
	}
	n, _ := strconv.Atoi(row["n"])
	return n, nil
}

// Notify отправляет находки одним письмом получателям CMDB_ALERT_EMAILS через SMTP
// (CMDB_SMTP_HOST/…). Пустой CMDB_SMTP_HOST или CMDB_ALERT_EMAILS — уведомление по почте
// выключено (не ошибка: метрики Prometheus продолжают обновляться независимо, см. Check).
func Notify(ctx context.Context, findings []Finding) error {
	if len(findings) == 0 {
		return nil
	}
	host := os.Getenv("CMDB_SMTP_HOST")
	to := strings.FieldsFunc(os.Getenv("CMDB_ALERT_EMAILS"), func(r rune) bool { return r == ',' || r == ';' || r == ' ' })
	if host == "" || len(to) == 0 {
		logging.FromContext(ctx).Info("alerting_email_disabled", "findings", len(findings))
		return nil
	}
	port := os.Getenv("CMDB_SMTP_PORT")
	if port == "" {
		port = "25"
	}
	from := os.Getenv("CMDB_SMTP_FROM")
	if from == "" {
		from = "cmdb@localhost"
	}

	var body strings.Builder
	fmt.Fprintf(&body, "Subject: CMDB: %d операционных находок\r\n", len(findings))
	fmt.Fprintf(&body, "From: %s\r\nTo: %s\r\n\r\n", from, strings.Join(to, ", "))
	for _, f := range findings {
		fmt.Fprintf(&body, "- [%s] %s\n", f.Kind, f.Message)
	}

	addr := host + ":" + port
	var auth smtp.Auth
	if user := os.Getenv("CMDB_SMTP_USER"); user != "" {
		auth = smtp.PlainAuth("", user, os.Getenv("CMDB_SMTP_PASSWORD"), host)
	}
	if os.Getenv("CMDB_SMTP_STARTTLS") == "1" {
		return sendStartTLS(addr, host, auth, from, to, []byte(body.String()), os.Getenv("CMDB_SMTP_INSECURE") == "1")
	}
	return smtp.SendMail(addr, auth, from, to, []byte(body.String()))
}

// sendStartTLS — путь для настоящих SMTP-релеев прод-окружения (mailpit из docker-compose.yml, в
// отличие от них, принимает письма без TLS и авторизации — CMDB_SMTP_STARTTLS не задан в dev).
func sendStartTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte, insecure bool) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close()
	if err := c.StartTLS(&tls.Config{ServerName: host, InsecureSkipVerify: insecure}); err != nil { //nolint:gosec // явная настройка стенда
		return err
	}
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
