// Интеграционные тесты компонентов: PostgreSQL с миграциями и seed nc2026 (ref.component_type).
// CMDB_TEST_DB_DSN=... go test ./internal/services/component/... -v
package component

import (
	"context"
	"errors"
	"testing"
	"time"

	"cmdb/internal/platform/asof"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

func newServer(t *testing.T) int64 {
	t.Helper()
	res, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		Site: testutil.TestSite, LocationCode: testutil.TestLocation, Reason: "фикстура компонентов", UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	return res.CIID
}

// Замена: старый компонент остаётся со статусом replaced и ссылкой на новый, вложенный компонент
// переезжает на новый, прошлое состояние доступно через component_asof.
func TestReplace_KeepsHistoryAndMovesChildren(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	ciID := newServer(t)

	ctrlID, _, err := Add(ctx, AddInput{CIID: ciID, ComponentType: "raid_ctrl", Slot: "RAID1",
		Spec: Spec{Manufacturer: "TestVendor", SerialNumber: "OLD-CTRL"}, Reason: "установка", UserID: 1})
	if err != nil {
		t.Fatalf("Add контроллер: %v", err)
	}
	diskID, _, err := Add(ctx, AddInput{CIID: ciID, ParentComponentID: &ctrlID, ComponentType: "ssd", Slot: "0",
		Spec: Spec{SerialNumber: "DISK-1"}, Reason: "установка", UserID: 1})
	if err != nil {
		t.Fatalf("Add диск: %v", err)
	}

	var beforeReplace time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&beforeReplace); err != nil {
		t.Fatalf("clock_timestamp: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	res, err := Replace(ctx, ReplaceInput{ComponentID: ctrlID, ExpectedVersion: 1,
		New: Spec{Manufacturer: "TestVendor", SerialNumber: "NEW-CTRL"}, Reason: "отказ контроллера", UserID: 1})
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}

	var oldStatus, replacedBy, newUID string
	if err := db.QueryRow(`SELECT status, attrs->>'replaced_by' FROM cmdb.component WHERE id = $1`, ctrlID).Scan(&oldStatus, &replacedBy); err != nil {
		t.Fatalf("select старый: %v", err)
	}
	if err := db.QueryRow(`SELECT uid::text FROM cmdb.component WHERE id = $1`, res.NewComponentID).Scan(&newUID); err != nil {
		t.Fatalf("select новый: %v", err)
	}
	if oldStatus != StatusReplaced || replacedBy != newUID {
		t.Errorf("старый: status=%q replaced_by=%q; want %q / %q", oldStatus, replacedBy, StatusReplaced, newUID)
	}

	var diskParent int64
	if err := db.QueryRow(`SELECT parent_component_id FROM cmdb.component WHERE id = $1`, diskID).Scan(&diskParent); err != nil {
		t.Fatalf("select диск: %v", err)
	}
	if diskParent != res.NewComponentID {
		t.Errorf("родитель диска = %d, want новый контроллер %d", diskParent, res.NewComponentID)
	}

	// Состояние до замены: контроллер ещё installed, а нового не было.
	var status string
	if err := db.QueryRow(`SELECT status FROM cmdb.component_asof($1) WHERE id = $2`, beforeReplace, ctrlID).Scan(&status); err != nil {
		t.Fatalf("component_asof: %v", err)
	}
	if status != StatusInstalled {
		t.Errorf("AS OF до замены: status = %q, want %q", status, StatusInstalled)
	}
	var cnt int
	if err := db.QueryRow(`SELECT count(*) FROM cmdb.component_asof($1) WHERE id = $2`, beforeReplace, res.NewComponentID).Scan(&cnt); err != nil {
		t.Fatalf("component_asof (новый): %v", err)
	}
	if cnt != 0 {
		t.Errorf("AS OF до замены видит новый компонент (%d строк), want 0", cnt)
	}

	// Повторная замена уже снятого компонента и устаревшая версия отклоняются.
	if _, err := Replace(ctx, ReplaceInput{ComponentID: ctrlID, ExpectedVersion: 2, New: Spec{}, Reason: "повтор", UserID: 1}); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("повторная замена: err = %v, want ErrInvalidState", err)
	}
	if _, err := Replace(ctx, ReplaceInput{ComponentID: res.NewComponentID, ExpectedVersion: 99, New: Spec{}, Reason: "версия", UserID: 1}); !errors.Is(err, svcerr.ErrVersionConflict) {
		t.Errorf("устаревшая версия: err = %v, want ErrVersionConflict", err)
	}
}

// Перемещение: компонент и его вложенные уходят в другую КЕ, корень теряет родителя.
func TestMove_CarriesChildrenToAnotherCI(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	src, dst := newServer(t), newServer(t)

	ctrlID, _, err := Add(ctx, AddInput{CIID: src, ComponentType: "raid_ctrl", Slot: "RAID1", Reason: "установка", UserID: 1})
	if err != nil {
		t.Fatalf("Add контроллер: %v", err)
	}
	diskID, _, err := Add(ctx, AddInput{CIID: src, ParentComponentID: &ctrlID, ComponentType: "ssd", Reason: "установка", UserID: 1})
	if err != nil {
		t.Fatalf("Add диск: %v", err)
	}

	if _, err := Move(ctx, MoveInput{ComponentID: ctrlID, ExpectedVersion: 1, ToCIID: dst, ToSlot: "RAID2", Reason: "перенос в другой сервер", UserID: 1}); err != nil {
		t.Fatalf("Move: %v", err)
	}

	for _, id := range []int64{ctrlID, diskID} {
		var ciID int64
		if err := db.QueryRow(`SELECT ci_id FROM cmdb.component WHERE id = $1`, id).Scan(&ciID); err != nil {
			t.Fatalf("select component %d: %v", id, err)
		}
		if ciID != dst {
			t.Errorf("component %d: ci_id = %d, want %d", id, ciID, dst)
		}
	}

	// Родитель из другой КЕ недопустим при добавлении.
	if _, _, err := Add(ctx, AddInput{CIID: src, ParentComponentID: &ctrlID, ComponentType: "ssd", Reason: "чужой родитель", UserID: 1}); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("родитель из другой КЕ: err = %v, want ErrInvalidState", err)
	}
}

// E6.1: ListForCI при ?asof= в контексте показывает список компонентов «как было» (component_asof).
func TestListForCI_RespectsAsOfContext(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	ciID := newServer(t)

	id, _, err := Add(ctx, AddInput{CIID: ciID, ComponentType: "ssd", Slot: "0", Spec: Spec{SerialNumber: "OLD-DISK"}, Reason: "установка", UserID: 1})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	var before time.Time
	db.QueryRow(`SELECT clock_timestamp()`).Scan(&before)
	time.Sleep(50 * time.Millisecond)
	if _, err := Replace(ctx, ReplaceInput{ComponentID: id, ExpectedVersion: 1, New: Spec{SerialNumber: "NEW-DISK"}, Reason: "замена", UserID: 1}); err != nil {
		t.Fatal(err)
	}

	now, err := ListForCI(ctx, ciID)
	if err != nil || len(now) != 2 {
		t.Fatalf("текущий список = %d записей (err=%v), want 2 (старый replaced + новый)", len(now), err)
	}
	past, err := ListForCI(asof.WithTime(ctx, before), ciID)
	if err != nil || len(past) != 1 || past[0].Row["serial_number"] != "OLD-DISK" || past[0].Row["status"] != "installed" {
		t.Fatalf("список на момент до замены = %+v (err=%v), want один установленный OLD-DISK", past, err)
	}
}
