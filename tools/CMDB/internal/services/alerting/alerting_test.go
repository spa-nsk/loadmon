package alerting

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"cmdb/internal/testutil"
)

func newFailingConnector(t *testing.T, db *sql.DB, streak int) string {
	t.Helper()
	code := "alert-" + testutil.Suffix()
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('alerttest', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	var connID int64
	if err := db.QueryRow(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config, enabled)
		VALUES ($1, 'alerttest', 'Тестовый источник', 'alerttest', '{}'::jsonb, true) RETURNING id`, code).Scan(&connID); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	for i := 0; i < streak; i++ {
		started := time.Now().Add(-time.Duration(streak-i) * time.Minute)
		if _, err := db.Exec(`INSERT INTO imp.run (connector_id, status, dry_run, started_at, finished_at)
			VALUES ($1, 'failed', false, $2, $2)`, connID, started); err != nil {
			t.Fatalf("imp.run: %v", err)
		}
	}
	return code
}

// seedOpenConflicts создаёт СВОЙ отдельный источник и прогон для конфликтов — переиспользование
// коннектора newFailingConnector сломало бы серию неудач: новый успешный прогон, нужный для
// imp.conflict.run_id, стал бы самым свежим и оборвал серию 'failed' на нём же (реальный баг
// теста, пойманный именно так — первая версия молча проверяла не тот коннектор).
func seedOpenConflicts(t *testing.T, db *sql.DB, n int) {
	t.Helper()
	code := "alertconf-" + testutil.Suffix()
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('alerttest', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	var connID, runID int64
	if err := db.QueryRow(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config, enabled)
		VALUES ($1, 'alerttest', 'Тестовый источник конфликтов', 'alerttest', '{}'::jsonb, true) RETURNING id`, code).Scan(&connID); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO imp.run (connector_id, status, dry_run) VALUES ($1, 'succeeded', false) RETURNING id`, connID).Scan(&runID); err != nil {
		t.Fatalf("imp.run: %v", err)
	}
	for i := 0; i < n; i++ {
		if _, err := db.Exec(`INSERT INTO imp.conflict (run_id, kind, status, created_at) VALUES ($1, 'not_discovered', 'open', now())`, runID); err != nil {
			t.Fatalf("imp.conflict: %v", err)
		}
	}
}

func TestCheck_DetectsConsecutiveFailuresAndNewConflicts(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newFailingConnector(t, db, 4)
	seedOpenConflicts(t, db, 5)

	findings, err := Check(ctx, Thresholds{ConsecutiveFailures: 3, NewConflicts: 5, Window: 24 * time.Hour})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	var gotFailing, gotConflicts bool
	for _, f := range findings {
		if f.Kind == "import_failing" && f.Subject == code {
			gotFailing = true
			if f.Count < 4 {
				t.Errorf("import_failing count = %d, want >= 4", f.Count)
			}
		}
		if f.Kind == "conflicts_growing" {
			gotConflicts = true
		}
	}
	if !gotFailing {
		t.Errorf("не найдена находка import_failing для %s: %+v", code, findings)
	}
	if !gotConflicts {
		t.Errorf("не найдена находка conflicts_growing (порог 5, создано 5 за окно): %+v", findings)
	}
}

func TestCheck_BelowThresholdNoFindings(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	code := newFailingConnector(t, db, 1) // 1 неудача — ниже порога

	findings, err := Check(ctx, Thresholds{ConsecutiveFailures: 3, NewConflicts: 1000, Window: 24 * time.Hour})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	for _, f := range findings {
		if f.Kind == "import_failing" && f.Subject == code {
			t.Errorf("не ожидалась находка ниже порога: %+v", f)
		}
	}
}

type mailpitMessage struct {
	Subject string
	To      []struct{ Address string }
}
type mailpitList struct {
	Messages []mailpitMessage
}

func TestNotify_SendsRealEmailViaMailpit(t *testing.T) {
	testutil.DB(t) // тот же пропуск без CMDB_TEST_DB_DSN, что у остальных интеграционных тестов
	mailpitURL := "http://localhost:8025"
	resp, err := http.Get(mailpitURL + "/api/v1/messages")
	if err != nil {
		t.Skip("mailpit недоступен (docker compose up -d mailpit) — пропуск сквозной проверки письма")
	}
	resp.Body.Close()

	req, _ := http.NewRequest(http.MethodDelete, mailpitURL+"/api/v1/messages", nil)
	http.DefaultClient.Do(req) // очистка перед проверкой — best effort

	t.Setenv("CMDB_SMTP_HOST", "localhost")
	t.Setenv("CMDB_SMTP_PORT", "1025")
	t.Setenv("CMDB_SMTP_FROM", "cmdb-test@localhost")
	to := "ops-" + testutil.Suffix() + "@example.test"
	t.Setenv("CMDB_ALERT_EMAILS", to)

	findings := []Finding{{Kind: "import_failing", Subject: "acme-dc1", Count: 5, Message: "источник acme-dc1: 5 прогонов подряд завершились ошибкой"}}
	if err := Notify(context.Background(), findings); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	var list mailpitList
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		r, err := http.Get(mailpitURL + "/api/v1/messages")
		if err == nil {
			b, _ := io.ReadAll(r.Body)
			r.Body.Close()
			json.Unmarshal(b, &list)
			if len(list.Messages) > 0 {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if len(list.Messages) == 0 {
		t.Fatal("mailpit не получил ни одного письма")
	}
	found := false
	for _, m := range list.Messages {
		for _, rcpt := range m.To {
			if rcpt.Address == to {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("письмо получателю %s не найдено среди принятых mailpit: %+v", to, list.Messages)
	}
}

func TestNotify_DisabledWithoutSMTPHost(t *testing.T) {
	t.Setenv("CMDB_SMTP_HOST", "")
	t.Setenv("CMDB_ALERT_EMAILS", "someone@example.test")
	if err := Notify(context.Background(), []Finding{{Kind: "import_failing", Message: "x"}}); err != nil {
		t.Errorf("без CMDB_SMTP_HOST Notify должен тихо ничего не делать, не ошибаться: %v", err)
	}
}

func TestDefaultThresholds_EnvOverride(t *testing.T) {
	th := DefaultThresholds()
	if th.ConsecutiveFailures != 3 || th.NewConflicts != 20 || th.Window != 24*time.Hour {
		t.Errorf("значения по умолчанию = %+v", th)
	}
	t.Setenv("CMDB_ALERT_CONSECUTIVE_FAILURES", "7")
	th2 := DefaultThresholds()
	if th2.ConsecutiveFailures != 7 {
		t.Errorf("переопределение = %d, want 7", th2.ConsecutiveFailures)
	}
}
