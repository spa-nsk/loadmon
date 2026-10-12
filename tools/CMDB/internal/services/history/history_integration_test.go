// E3.7: сравнение версий и глобальный журнал изменений — реальный PostgreSQL, миграции и seed
// nc2026 (internal/testutil). Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/history/...
package history

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

func TestCompare_AttributeChangeBetweenChangeSets(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	created, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "фикстура сравнения " + suffix, UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	csCreate := created.ChangeSetID

	desc := "описание " + suffix
	upd, err := ci.Update(ctx, ci.UpdateInput{CIID: created.CIID, ExpectedVersion: 1, Reason: "правка описания " + suffix,
		UserID: 1, Description: &desc})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	csUpdate := upd.ChangeSetID

	fields, err := Compare(ctx, created.CIID, csCreate, csUpdate)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	byName := map[string]CompareField{}
	for _, f := range fields {
		byName[f.Name] = f
	}
	d, ok := byName["description"]
	if !ok {
		t.Fatal("в сравнении нет колонки description")
	}
	if !d.Changed || d.B != desc {
		t.Errorf("description = %+v, want Changed=true B=%q", d, desc)
	}
	if n, ok := byName["name"]; !ok || n.Changed {
		t.Errorf("name не должно измениться между двумя change set: %+v", n)
	}
	for _, noisy := range []string{"uid", "sys_period", "row_version", "change_set_id", "created_at", "created_by"} {
		if _, present := byName[noisy]; present {
			t.Errorf("служебная колонка %q не должна попадать в сравнение", noisy)
		}
	}

	// Сравнение версии самой с собой — различий нет.
	same, err := Compare(ctx, created.CIID, csUpdate, csUpdate)
	if err != nil {
		t.Fatalf("Compare (та же версия): %v", err)
	}
	for _, f := range same {
		if f.Changed {
			t.Errorf("сравнение версии с собой: поле %q помечено изменённым: %+v", f.Name, f)
		}
	}
}

func TestGlobalLog_FiltersAndForChangeSet(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	reason := "глобальный журнал " + suffix

	created, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: reason, UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	res, err := GlobalLog(ctx, GlobalFilter{Q: suffix, PerPage: 10})
	if err != nil {
		t.Fatalf("GlobalLog: %v", err)
	}
	if res.Total != 1 {
		t.Fatalf("GlobalLog(Q=%q) Total = %d, want 1", suffix, res.Total)
	}
	row := res.Rows[0]
	wantID := strconv.FormatInt(created.ChangeSetID, 10)
	if row.ID != wantID {
		t.Errorf("GlobalLog: id = %q, want %q", row.ID, wantID)
	}
	if row.Reason != reason {
		t.Errorf("GlobalLog: reason = %q, want %q", row.Reason, reason)
	}
	if row.Source != "manual" {
		t.Errorf("GlobalLog: source = %q, want manual", row.Source)
	}
	if !strings.Contains(row.Entities, "ci") {
		t.Errorf("GlobalLog: entities = %q, want содержит ci", row.Entities)
	}

	// Фильтр по source_kind, не совпадающему с manual, не находит запись.
	res2, err := GlobalLog(ctx, GlobalFilter{Q: suffix, Source: "import", PerPage: 10})
	if err != nil {
		t.Fatalf("GlobalLog (source=import): %v", err)
	}
	if res2.Total != 0 {
		t.Errorf("GlobalLog(Q=%q, source=import) Total = %d, want 0", suffix, res2.Total)
	}

	entities, err := DistinctEntities(ctx)
	if err != nil {
		t.Fatalf("DistinctEntities: %v", err)
	}
	found := false
	for _, e := range entities {
		if e == "ci" {
			found = true
		}
	}
	if !found {
		t.Errorf("DistinctEntities не содержит 'ci': %v", entities)
	}

	cs, err := ForChangeSet(ctx, created.ChangeSetID)
	if err != nil {
		t.Fatalf("ForChangeSet: %v", err)
	}
	if cs.Reason != reason {
		t.Errorf("ForChangeSet: reason = %q, want %q", cs.Reason, reason)
	}
	if len(cs.Entries) == 0 {
		t.Error("ForChangeSet: нет записей change_log")
	}
}
