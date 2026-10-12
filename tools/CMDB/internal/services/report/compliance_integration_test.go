// Отчёты соответствия (plan.md E7.9) — реальный PostgreSQL, миграции и seed nc2026. Общие таблицы
// для всего тестового набора (как и в остальных пакетах отчётов) — фикстуры ищутся по уникальному
// маркеру, не по позиции/счётчику.
// Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/report/...
package report_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/relation"
	"cmdb/internal/services/report"
	"cmdb/internal/services/software"
	"cmdb/internal/testutil"
)

func setLastSeen(t *testing.T, ciID int64, at time.Time) {
	t.Helper()
	if _, err := changectx.Run(context.Background(), changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура свежести"},
		func(ctx context.Context, tx orm.TxOrmer) error {
			_, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET last_seen_at = ? WHERE id = ?`, at, ciID).Exec()
			return err
		}); err != nil {
		t.Fatalf("setLastSeen: %v", err)
	}
}

func TestStaleInventory_FindsOldAndNeverConfirmedExcludesFresh(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	old := insertCI(t, "server", "TEST-STALE-OLD-"+suffix, testutil.TestLocation)
	setLastSeen(t, old, time.Now().Add(-40*24*time.Hour))
	never := insertCI(t, "server", "TEST-STALE-NEVER-"+suffix, testutil.TestLocation) // last_seen_at остаётся NULL
	fresh := insertCI(t, "server", "TEST-STALE-FRESH-"+suffix, testutil.TestLocation)
	setLastSeen(t, fresh, time.Now())

	rows, err := report.StaleInventory(ctx)
	if err != nil {
		t.Fatalf("StaleInventory: %v", err)
	}
	seen := map[string]string{}
	for _, r := range rows {
		seen[r["id"]] = r["days_stale"]
	}
	oldID, neverID, freshID := idText(old), idText(never), idText(fresh)
	if days, ok := seen[oldID]; !ok {
		t.Error("КЕ с last_seen_at 40 дней назад не найдена в отчёте")
	} else if days == "" {
		t.Error("для устаревшей КЕ days_stale пусто")
	}
	if days, ok := seen[neverID]; !ok || days != "" {
		t.Errorf("никогда не подтверждённая КЕ: найдена=%v days_stale=%q, want найдена и пусто", ok, days)
	}
	if _, ok := seen[freshID]; ok {
		t.Error("свежеподтверждённая КЕ не должна попадать в отчёт устаревания")
	}
}

func TestKIIComposition_FindsObjectAndMember(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	kiiName, memberName := "TEST-KII-"+suffix, "TEST-KII-MEMBER-"+suffix
	kiiID := insertCI(t, "server", kiiName, testutil.TestLocation)
	memberID := insertCI(t, "server", memberName, testutil.TestLocation)

	category := "Значимый объект II категории"
	kiiObject := true
	if _, err := ci.Update(ctx, ci.UpdateInput{CIID: kiiID, ExpectedVersion: 1, Reason: "фикстура КИИ", UserID: 1,
		KIICategory: &category, KIIObject: &kiiObject}); err != nil {
		t.Fatalf("ci.Update (kii_object): %v", err)
	}
	if _, _, err := relation.Create(ctx, relation.CreateInput{RelType: "depends_on", SrcCIID: kiiID, DstCIID: memberID, Reason: "фикстура КИИ", UserID: 1}); err != nil {
		t.Fatalf("relation.Create: %v", err)
	}

	rows, err := report.KIIComposition(ctx)
	if err != nil {
		t.Fatalf("KIIComposition: %v", err)
	}
	found := false
	for _, r := range rows {
		if r["kii_name"] == kiiName && r["member_name"] == memberName {
			found = true
			if r["kii_category"] != category {
				t.Errorf("kii_category = %q, want %q", r["kii_category"], category)
			}
			if r["rel_type"] != "depends_on" {
				t.Errorf("rel_type = %q, want depends_on", r["rel_type"])
			}
		}
	}
	if !found {
		t.Fatalf("KIIComposition не нашёл пару (%q, %q)", kiiName, memberName)
	}
}

func TestDomesticHardware_FindsModelWithRegistryRef(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	var mfID, modelID int64
	if err := db.QueryRow(`INSERT INTO ref.manufacturer (name) VALUES ($1) RETURNING id`, "TestMF-"+suffix).Scan(&mfID); err != nil {
		t.Fatalf("ref.manufacturer: %v", err)
	}
	modelName := "TestModel-" + suffix
	if err := db.QueryRow(`INSERT INTO ref.model (manufacturer_id, model_name, is_domestic, registry_ref) VALUES ($1, $2, true, $3) RETURNING id`,
		mfID, modelName, "reestr.minprom.gov.ru/12345").Scan(&modelID); err != nil {
		t.Fatalf("ref.model: %v", err)
	}

	if _, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "фикстура отечественного оборудования " + suffix, UserID: 1, ModelID: &modelID,
	}); err != nil {
		t.Fatalf("CreateServer: %v", err)
	}

	rows, err := report.DomesticHardware(ctx)
	if err != nil {
		t.Fatalf("DomesticHardware: %v", err)
	}
	var found *string
	for _, r := range rows {
		if r["model_name"] == modelName {
			v := r["is_domestic"]
			found = &v
			if r["registry_ref"] != "reestr.minprom.gov.ru/12345" {
				t.Errorf("registry_ref = %q, want reestr.minprom.gov.ru/12345", r["registry_ref"])
			}
			if r["n"] != "1" {
				t.Errorf("n = %q, want 1", r["n"])
			}
		}
	}
	if found == nil {
		t.Fatalf("DomesticHardware не нашёл модель %q", modelName)
	}
	if *found != "да" {
		t.Errorf("is_domestic = %q, want да", *found)
	}
}

func TestDomesticSoftware_FindsInstallWithDomesticFlag(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	var swID int64
	product := "TestDomesticSW-" + suffix
	if err := db.QueryRow(`INSERT INTO ref.software (vendor, product, version, is_domestic, registry_ref) VALUES ('TestVendor', $1, '1.0', true, $2) RETURNING id`,
		product, "reestr.minsvyaz.ru/999").Scan(&swID); err != nil {
		t.Fatalf("ref.software: %v", err)
	}
	ciID := insertCI(t, "server", "TEST-DOMSW-CI-"+suffix, testutil.TestLocation)
	if _, _, err := software.Add(ctx, software.AddInput{CIID: ciID, SoftwareID: swID, Reason: "фикстура отечественного ПО " + suffix, UserID: 1}); err != nil {
		t.Fatalf("software.Add: %v", err)
	}

	rows, err := report.DomesticSoftware(ctx)
	if err != nil {
		t.Fatalf("DomesticSoftware: %v", err)
	}
	found := false
	for _, r := range rows {
		if r["product"] == product {
			found = true
			if r["is_domestic"] != "да" {
				t.Errorf("is_domestic = %q, want да", r["is_domestic"])
			}
			if r["registry_ref"] != "reestr.minsvyaz.ru/999" {
				t.Errorf("registry_ref = %q, want reestr.minsvyaz.ru/999", r["registry_ref"])
			}
		}
	}
	if !found {
		t.Fatalf("DomesticSoftware не нашёл продукт %q", product)
	}
}

func TestLegacyNames_SuggestsNamePattern(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	name := "legacy-host-" + suffix

	var id int64
	if _, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура legacy-имени"},
		func(ctx context.Context, tx orm.TxOrmer) error {
			return tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci (ci_class, name, location_code, naming_status) VALUES ('server', ?, ?, 'legacy') RETURNING id`,
				name, testutil.TestLocation).QueryRow(&id)
		}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := report.LegacyNames(ctx, naming.DefaultRegistry())
	if err != nil {
		t.Fatalf("LegacyNames: %v", err)
	}
	var suggested string
	found := false
	for _, r := range rows {
		if r["name"] == name {
			found = true
			suggested = r["suggested_name"]
		}
	}
	if !found {
		t.Fatalf("LegacyNames не нашёл %q", name)
	}
	if suggested == "" || !strings.Contains(suggested, "-S-") || !strings.Contains(suggested, "N") {
		t.Errorf("suggested_name = %q, want шаблон вида SITE-S-NNNNN", suggested)
	}
}

func idText(id int64) string {
	return strconv.FormatInt(id, 10)
}
