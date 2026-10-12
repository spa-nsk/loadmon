// E3.8: дерево размещения площадки — реальный PostgreSQL, миграции и seed nc2026.
package refadmin

import (
	"context"
	"testing"

	"cmdb/internal/repo"
	"cmdb/internal/testutil"
)

func TestLocationTree_BuildsHierarchyByDepth(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	spec, ok := Spec("site_place")
	if !ok {
		t.Fatal("спецификация site_place не найдена")
	}
	suffix := testutil.Suffix()
	locationID := locationIDFor(t, ctx, testutil.TestLocation)

	if err := Create(ctx, spec, map[string]string{"location_id": locationID, "kind": "building", "code": "A" + suffix}, 1); err != nil {
		t.Fatalf("Create building: %v", err)
	}
	buildingID := lastPlaceID(t, ctx, testutil.TestLocation, "A"+suffix)
	t.Cleanup(func() { Delete(ctx, spec, buildingID) })

	if err := Create(ctx, spec, map[string]string{
		"location_id": locationID, "parent_id": buildingID, "kind": "room", "code": "101" + suffix,
	}, 1); err != nil {
		t.Fatalf("Create room: %v", err)
	}
	roomID := lastPlaceID(t, ctx, testutil.TestLocation, "101"+suffix)
	t.Cleanup(func() { Delete(ctx, spec, roomID) })

	if err := Create(ctx, spec, map[string]string{
		"location_id": locationID, "parent_id": roomID, "kind": "rack", "code": "R01" + suffix, "rack_units": "42",
	}, 1); err != nil {
		t.Fatalf("Create rack: %v", err)
	}
	rackID := lastPlaceID(t, ctx, testutil.TestLocation, "R01"+suffix)
	t.Cleanup(func() { Delete(ctx, spec, rackID) })

	nodes, err := LocationTree(ctx, testutil.TestLocation)
	if err != nil {
		t.Fatalf("LocationTree: %v", err)
	}
	depthByCode := map[string]int{}
	for _, n := range nodes {
		depthByCode[n.Row["code"]] = n.Depth
	}
	if depthByCode["A"+suffix] != 0 {
		t.Errorf("building depth = %d, want 0", depthByCode["A"+suffix])
	}
	if depthByCode["101"+suffix] != 1 {
		t.Errorf("room depth = %d, want 1", depthByCode["101"+suffix])
	}
	if depthByCode["R01"+suffix] != 2 {
		t.Errorf("rack depth = %d, want 2", depthByCode["R01"+suffix])
	}
	for _, n := range nodes {
		if n.Row["code"] == "R01"+suffix && n.Row["rack_units"] != "42" {
			t.Errorf("rack rack_units = %q, want 42", n.Row["rack_units"])
		}
	}
}

func locationIDFor(t *testing.T, ctx context.Context, locationCode string) string {
	t.Helper()
	row, err := repo.QueryRow(ctx, `SELECT id::text AS id FROM ref.location WHERE location_code = ?`, locationCode)
	if err != nil || row == nil {
		t.Fatalf("locationIDFor(%q): не найдено: %v", locationCode, err)
	}
	return row["id"]
}

// lastPlaceID находит id только что созданного site_place по location_code+code (Create не
// возвращает id — generic-CRUD не имеет типизированного результата, см. service.go).
func lastPlaceID(t *testing.T, ctx context.Context, locationCode, code string) string {
	t.Helper()
	row, err := repo.QueryRow(ctx, `
		SELECT sp.id::text AS id FROM ref.site_place sp JOIN ref.location l ON l.id = sp.location_id
		WHERE l.location_code = ? AND sp.code = ?`, locationCode, code)
	if err != nil || row == nil {
		t.Fatalf("lastPlaceID(%q, %q): не найдено: %v", locationCode, code, err)
	}
	return row["id"]
}
