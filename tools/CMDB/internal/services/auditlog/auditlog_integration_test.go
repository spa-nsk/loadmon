// E3.2/E7.6: реальный PostgreSQL, миграции.
package auditlog

import (
	"context"
	"database/sql"
	"testing"

	"cmdb/internal/testutil"
)

func TestLog_InsertsEvent(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	// event_type с уникальным суффиксом — audit.event только добавляется (E2.6/E7.6), прошлые
	// прогоны за эту же cmdb_test не убираются, поэтому точный count по общему event_type не годится.
	eventType := "ci_view_" + testutil.Suffix()

	if err := Log(ctx, 1, "203.0.113.5", eventType, map[string]any{"ci_id": 42}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	var count int
	var ip sql.NullString
	var details string
	if err := db.QueryRow(`SELECT count(*) FROM audit.event WHERE event_type = $1 AND user_id = 1`, eventType).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("audit.event: найдено %d строк %s для user_id=1, want 1", count, eventType)
	}
	if err := db.QueryRow(`SELECT host(client_ip), details::text FROM audit.event WHERE event_type = $1 AND user_id = 1 ORDER BY id DESC LIMIT 1`, eventType).
		Scan(&ip, &details); err != nil {
		t.Fatal(err)
	}
	if !ip.Valid || ip.String != "203.0.113.5" {
		t.Errorf("client_ip = %+v, want 203.0.113.5", ip)
	}
	if details != `{"ci_id": 42}` && details != `{"ci_id":42}` {
		t.Errorf("details = %q, want содержит ci_id=42", details)
	}

	// Пустой IP -> NULL, не ошибка.
	if err := Log(ctx, 1, "", "ref_export", nil); err != nil {
		t.Fatalf("Log (без IP): %v", err)
	}
}
