// E3.6: массовые операции — реальный PostgreSQL, миграции и seed nc2026 (internal/testutil).
package ci

import (
	"context"
	"database/sql"
	"testing"

	"cmdb/internal/testutil"
)

func insertCompany(t *testing.T, db *sql.DB, code string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO ref.company (company_code, company_name, category) VALUES ($1, $1, 'OWN') ON CONFLICT (company_code) DO NOTHING`, code); err != nil {
		t.Fatalf("insertCompany(%q): %v", code, err)
	}
}

// TestBulkApply_Stage — E3.6: предпросмотр и применение массовой смены стадии одним change_set;
// КЕ без обязательных атрибутов (E2.9) пропускается, а не блокирует остальные.
func TestBulkApply_Stage(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	ready := newServer(t, "фикстура bulk: готов")
	notReady := newServer(t, "фикстура bulk: не готов")

	if _, err := UpsertExtension(ctx, ExtensionInput{CIID: ready.CIID, Table: "ci_server", Reason: "атрибуты", UserID: 1,
		Attrs: map[string]any{"cpu_count": 2, "ram_gb": 16, "os_family": "Linux"},
	}); err != nil {
		t.Fatalf("UpsertExtension: %v", err)
	}

	ids := []int64{ready.CIID, notReady.CIID}
	preview, err := BulkPreview(ctx, BulkFieldStage, ids, StageCommissioning)
	if err != nil {
		t.Fatalf("BulkPreview: %v", err)
	}
	byID := map[int64]BulkItem{}
	for _, it := range preview {
		byID[it.CIID] = it
	}
	if byID[ready.CIID].Skip {
		t.Errorf("готовая КЕ помечена Skip: %+v", byID[ready.CIID])
	}
	if !byID[notReady.CIID].Skip {
		t.Errorf("КЕ без атрибутов должна быть Skip: %+v", byID[notReady.CIID])
	}

	res, err := BulkApply(ctx, BulkFieldStage, ids, StageCommissioning, "массовый ввод в эксплуатацию", 1)
	if err != nil {
		t.Fatalf("BulkApply: %v", err)
	}
	if res.Applied != 1 || res.Skipped != 1 {
		t.Fatalf("BulkApply = %+v, want Applied=1 Skipped=1", res)
	}
	if res.ChangeSetID == 0 {
		t.Fatal("ChangeSetID не проставлен")
	}

	var stage string
	if err := db.QueryRow(`SELECT lifecycle_stage FROM cmdb.ci WHERE id = $1`, ready.CIID).Scan(&stage); err != nil {
		t.Fatal(err)
	}
	if stage != StageCommissioning {
		t.Errorf("готовая КЕ: стадия = %q, want %q", stage, StageCommissioning)
	}
	if err := db.QueryRow(`SELECT lifecycle_stage FROM cmdb.ci WHERE id = $1`, notReady.CIID).Scan(&stage); err != nil {
		t.Fatal(err)
	}
	if stage != StagePlanned {
		t.Errorf("пропущенная КЕ: стадия изменилась на %q, want %q (без изменений)", stage, StagePlanned)
	}

	// Повторный BulkApply той же парой (готовая теперь уже в целевой стадии) — Applied=0, Skipped=2.
	res2, err := BulkApply(ctx, BulkFieldStage, ids, StageCommissioning, "повтор", 1)
	if err != nil {
		t.Fatalf("BulkApply повтор: %v", err)
	}
	if res2.Applied != 0 || res2.Skipped != 2 {
		t.Errorf("повторный BulkApply = %+v, want Applied=0 Skipped=2", res2)
	}
}

// TestBulkApply_Company — одна причина -> один change_set на несколько КЕ (проверяется по
// audit.change_log.change_set_id).
func TestBulkApply_Company(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	insertCompany(t, db, "BULKCO")

	a := newServer(t, "фикстура bulk company A")
	b := newServer(t, "фикстура bulk company B")
	ids := []int64{a.CIID, b.CIID}

	res, err := BulkApply(ctx, BulkFieldCompany, ids, "BULKCO", "единая причина", 1)
	if err != nil {
		t.Fatalf("BulkApply: %v", err)
	}
	if res.Applied != 2 || res.Skipped != 0 {
		t.Fatalf("BulkApply = %+v, want Applied=2 Skipped=0", res)
	}

	var n int
	if err := db.QueryRow(`
		SELECT count(DISTINCT change_set_id) FROM audit.change_log
		WHERE entity = 'ci' AND attr_name = 'company_code' AND entity_id IN ($1, $2)
	`, a.CIID, b.CIID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("на 2 КЕ пришлось %d разных change_set_id, want 1 (единый change set)", n)
	}

	for _, id := range ids {
		var company string
		if err := db.QueryRow(`SELECT company_code FROM cmdb.ci WHERE id = $1`, id).Scan(&company); err != nil {
			t.Fatal(err)
		}
		if company != "BULKCO" {
			t.Errorf("id=%d: company_code = %q, want BULKCO", id, company)
		}
	}
}

// TestBulkPreview_EmptyIDs — пустой список отклоняется понятной ошибкой, а не паникой/пустым
// результатом без объяснения.
func TestBulkPreview_EmptyIDs(t *testing.T) {
	testutil.DB(t)
	if _, err := BulkPreview(context.Background(), BulkFieldStage, nil, StageCommissioning); err == nil {
		t.Fatal("expected error for empty ids")
	}
}
