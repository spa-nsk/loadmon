// E7.6: чтение audit.event для страницы «Журнал действий» — реальный PostgreSQL.
// Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/auditlog/...
package auditlog

import (
	"context"
	"strings"
	"testing"

	"cmdb/internal/testutil"
)

func TestList_FiltersByTypeUserAndQ(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	eventType := "test_event_" + suffix

	if err := Log(ctx, 1, "198.51.100.7", eventType, map[string]any{"marker": suffix}); err != nil {
		t.Fatalf("Log: %v", err)
	}

	// по типу события — находит ровно нашу запись (тип уникален для этого прогона)
	res, err := List(ctx, Filter{EventType: eventType})
	if err != nil {
		t.Fatalf("List(event_type): %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("List(event_type=%q).Total = %d, want 1", eventType, res.Total)
	}
	row := res.Rows[0]
	if row.UserID != "1" {
		t.Errorf("UserID = %q, want 1", row.UserID)
	}
	if row.ClientIP != "198.51.100.7" {
		t.Errorf("ClientIP = %q, want 198.51.100.7", row.ClientIP)
	}
	if !strings.Contains(row.Details, suffix) {
		t.Errorf("Details = %q, want содержит %q", row.Details, suffix)
	}

	// по пользователю без указания типа — тоже находит (среди прочего)
	byUser, err := List(ctx, Filter{UserID: 1, Q: suffix})
	if err != nil {
		t.Fatalf("List(user_id, q): %v", err)
	}
	if byUser.Total != 1 {
		t.Fatalf("List(user_id=1, q=%q).Total = %d, want 1", suffix, byUser.Total)
	}

	// несовпадающий тип не находит
	none, err := List(ctx, Filter{EventType: "no-such-event-type-" + suffix})
	if err != nil {
		t.Fatalf("List(нет такого типа): %v", err)
	}
	if none.Total != 0 {
		t.Errorf("List(нет такого типа).Total = %d, want 0", none.Total)
	}
}

func TestDistinctEventTypes_ContainsLoggedType(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	eventType := "test_distinct_" + testutil.Suffix()
	if err := Log(ctx, 1, "", eventType, nil); err != nil {
		t.Fatalf("Log: %v", err)
	}

	types, err := DistinctEventTypes(ctx)
	if err != nil {
		t.Fatalf("DistinctEventTypes: %v", err)
	}
	found := false
	for _, ty := range types {
		if ty == eventType {
			found = true
		}
	}
	if !found {
		t.Errorf("DistinctEventTypes не содержит %q", eventType)
	}
}
