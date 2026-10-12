// Интеграционные тесты связей: PostgreSQL с миграциями и seed nc2026 (ref.rel_type).
// CMDB_TEST_DB_DSN=... go test ./internal/services/relation/... -v
package relation

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

func newServer(t *testing.T) ci.CreateServerResult {
	t.Helper()
	res, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		Site: testutil.TestSite, LocationCode: testutil.TestLocation, Reason: "фикстура связей", UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	return res
}

func clock(t *testing.T, db *sql.DB) time.Time {
	t.Helper()
	var ts time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&ts); err != nil {
		t.Fatalf("clock_timestamp: %v", err)
	}
	return ts
}

func TestCreateValidation(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	a, b := newServer(t), newServer(t)
	base := CreateInput{RelType: "depends_on", SrcCIID: a.CIID, DstCIID: b.CIID, Reason: "тест", UserID: 1}

	self := base
	self.DstCIID = a.CIID
	if _, _, err := Create(ctx, self); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("связь с самой собой: err = %v, want ErrInvalidState", err)
	}

	missing := base
	missing.DstCIID = -1
	if _, _, err := Create(ctx, missing); !errors.Is(err, svcerr.ErrNotFound) {
		t.Errorf("несуществующая КЕ: err = %v, want ErrNotFound", err)
	}

	unknownType := base
	unknownType.RelType = "no_such_type"
	if _, _, err := Create(ctx, unknownType); err == nil {
		t.Error("неизвестный rel_type должен быть отклонён внешним ключом")
	}

	// Выведенная из эксплуатации КЕ не участвует в новых связях.
	if _, err := ci.Retire(ctx, b.CIID, 1, "снята до ввода", 1); err != nil {
		t.Fatalf("Retire: %v", err)
	}
	if _, _, err := Create(ctx, base); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("связь с выведенной КЕ: err = %v, want ErrInvalidState", err)
	}
}

func TestLifecycleOfRelation_DuplicateCloseAndAsOf(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	a, b := newServer(t), newServer(t)
	in := CreateInput{RelType: "depends_on", SrcCIID: a.CIID, DstCIID: b.CIID,
		Attrs: map[string]any{"port": 5432}, Reason: "тест", UserID: 1}

	relID, _, err := Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := Create(ctx, in); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("параллельная такая же связь: err = %v, want ErrDuplicate", err)
	}

	// Связь видна с обеих сторон, направление определяется относительно запрошенной КЕ.
	fromA, err := ListForCI(ctx, a.CIID, nil)
	if err != nil || len(fromA) != 1 || !fromA[0].Outgoing || fromA[0].ID != relID || fromA[0].ValidTo != "" {
		t.Fatalf("ListForCI(a) = %+v, err = %v; want одна исходящая действующая связь %d", fromA, err, relID)
	}
	fromB, err := ListForCI(ctx, b.CIID, nil)
	if err != nil || len(fromB) != 1 || fromB[0].Outgoing {
		t.Fatalf("ListForCI(b) = %+v, err = %v; want одна входящая связь", fromB, err)
	}

	time.Sleep(50 * time.Millisecond)
	whileActive := clock(t, db)
	time.Sleep(50 * time.Millisecond)

	if _, err := Close(ctx, CloseInput{RelID: relID, ExpectedVersion: 99, Reason: "тест", UserID: 1}); !errors.Is(err, svcerr.ErrVersionConflict) {
		t.Errorf("Close с неверной версией: err = %v, want ErrVersionConflict", err)
	}
	if _, err := Close(ctx, CloseInput{RelID: relID, ExpectedVersion: 1, Reason: "связь разорвана", UserID: 1}); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := Close(ctx, CloseInput{RelID: relID, ExpectedVersion: 2, Reason: "повтор", UserID: 1}); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("повторное закрытие: err = %v, want ErrInvalidState", err)
	}

	now, _ := ListForCI(ctx, a.CIID, nil)
	past, _ := ListForCI(ctx, a.CIID, &whileActive)
	if len(now) != 0 || len(past) != 1 {
		t.Errorf("после закрытия: сейчас %d связей (want 0), на момент до закрытия %d (want 1)", len(now), len(past))
	}

	// Закрытая связь не мешает создать такую же заново (периоды не пересекаются).
	if _, _, err := Create(ctx, in); err != nil {
		t.Errorf("повторное создание после закрытия: %v", err)
	}

	// Атрибуты связи сохранены как jsonb.
	var port int
	if err := db.QueryRow(`SELECT (attrs->>'port')::int FROM cmdb.ci_rel WHERE id = $1`, relID).Scan(&port); err != nil || port != 5432 {
		t.Errorf("attrs.port = %d, err = %v; want 5432", port, err)
	}
}
