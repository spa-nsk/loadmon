// E7.7: установленное ПО на КЕ — реальный PostgreSQL, миграции и seed nc2026 (internal/testutil).
// Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/software/...
package software_test

import (
	"context"
	"errors"
	"testing"

	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/software"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

func newServer(t *testing.T) int64 {
	t.Helper()
	res, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "фикстура ПО " + testutil.Suffix(), UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	return res.CIID
}

func TestAddListRemove_RoundTrip(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	ciID := newServer(t)

	product := "TestProduct-" + suffix
	var swID int64
	if err := db.QueryRow(`INSERT INTO ref.software (vendor, product, version, policy) VALUES ('TestVendor', $1, '1.0', 'forbidden') RETURNING id`, product).Scan(&swID); err != nil {
		t.Fatalf("ref.software: %v", err)
	}

	id, csID, err := software.Add(ctx, software.AddInput{CIID: ciID, SoftwareID: swID, Version: "1.0", InstallPath: "/opt/test", Reason: "установка", UserID: 1})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if id == 0 || csID == 0 {
		t.Fatalf("Add вернул id=%d changeSetID=%d, want оба ненулевые", id, csID)
	}

	installs, err := software.ListForCI(ctx, ciID)
	if err != nil {
		t.Fatalf("ListForCI: %v", err)
	}
	if len(installs) != 1 {
		t.Fatalf("ListForCI = %d записей, want 1: %+v", len(installs), installs)
	}
	got := installs[0]
	if got.Vendor != "TestVendor" || got.Product != product || got.Policy != "forbidden" || got.Version != "1.0" {
		t.Errorf("установка = %+v, want vendor=TestVendor product=%s policy=forbidden version=1.0", got, product)
	}

	// дубль (та же КЕ+ПО+версия+путь) отклоняется
	if _, _, err := software.Add(ctx, software.AddInput{CIID: ciID, SoftwareID: swID, Version: "1.0", InstallPath: "/opt/test", Reason: "дубль", UserID: 1}); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("Add(дубль) = %v, want ErrDuplicate", err)
	}

	// снятие с устаревшей версией — конфликт
	if _, err := software.Remove(ctx, software.RemoveInput{ID: got.ID, ExpectedVersion: 999, Reason: "снятие", UserID: 1}); !errors.Is(err, svcerr.ErrVersionConflict) {
		t.Errorf("Remove(устаревшая версия) = %v, want ErrVersionConflict", err)
	}

	// корректное снятие
	if _, err := software.Remove(ctx, software.RemoveInput{ID: got.ID, ExpectedVersion: got.RowVersion, Reason: "снятие", UserID: 1}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	after, err := software.ListForCI(ctx, ciID)
	if err != nil {
		t.Fatalf("ListForCI (после снятия): %v", err)
	}
	if len(after) != 0 {
		t.Errorf("ListForCI после Remove = %d записей, want 0: %+v", len(after), after)
	}

	// снятие уже снятой — не найдена
	if _, err := software.Remove(ctx, software.RemoveInput{ID: got.ID, ExpectedVersion: got.RowVersion, Reason: "повтор", UserID: 1}); !errors.Is(err, svcerr.ErrNotFound) {
		t.Errorf("Remove(повторно) = %v, want ErrNotFound", err)
	}
}

func TestAdd_UnknownSoftwareOrCI(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	ciID := newServer(t)

	if _, _, err := software.Add(ctx, software.AddInput{CIID: ciID, SoftwareID: 999999999, Reason: "нет такого ПО", UserID: 1}); !errors.Is(err, svcerr.ErrNotFound) {
		t.Errorf("Add(несуществующее ПО) = %v, want ErrNotFound", err)
	}
}
