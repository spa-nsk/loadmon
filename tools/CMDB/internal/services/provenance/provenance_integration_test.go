// E3.4 «Источники»: реальный PostgreSQL, миграции и seed nc2026. cmdb.attr_source не под
// hist.versioning() (intent §7.3 — провенанс, не история значения), поэтому пишется обычным
// db.Exec, без ChangeContext.
package provenance

import (
	"context"
	"testing"

	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

func TestListForCI_OwnAndExtensionEntities(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	res, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "фикстура provenance", UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO cmdb.attr_source (entity, entity_id, attr_name, source_system_code, source_value, is_locked)
		VALUES ('ci', $1, 'description', NULL, '"вручную"'::jsonb, false)
	`, res.CIID); err != nil {
		t.Fatalf("insert attr_source (ci): %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO cmdb.attr_source (entity, entity_id, attr_name, source_system_code, source_value, is_locked)
		VALUES ('ci_server', $1, 'bios_uuid', NULL, '"11111111-1111-1111-1111-111111111111"'::jsonb, true)
	`, res.CIID); err != nil {
		t.Fatalf("insert attr_source (ci_server): %v", err)
	}

	rows, err := ListForCI(ctx, res.CIID, "ci_server")
	if err != nil {
		t.Fatalf("ListForCI: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("ListForCI = %d строк, want 2: %+v", len(rows), rows)
	}
	byAttr := map[string]map[string]string{}
	for _, r := range rows {
		byAttr[r["attr_name"]] = r
	}
	if byAttr["description"]["entity"] != "ci" || byAttr["description"]["is_locked"] != "false" {
		t.Errorf("description = %+v", byAttr["description"])
	}
	if byAttr["bios_uuid"]["entity"] != "ci_server" || byAttr["bios_uuid"]["is_locked"] != "true" {
		t.Errorf("bios_uuid = %+v", byAttr["bios_uuid"])
	}

	// Без extTable — только записи по самой cmdb.ci.
	rows, err = ListForCI(ctx, res.CIID, "")
	if err != nil {
		t.Fatalf("ListForCI (без extTable): %v", err)
	}
	if len(rows) != 1 || rows[0]["attr_name"] != "description" {
		t.Errorf("ListForCI без extTable = %+v, want только description", rows)
	}
}
