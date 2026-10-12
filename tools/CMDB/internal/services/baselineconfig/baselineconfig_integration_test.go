// E7.8: эталонные конфигурации — реальный PostgreSQL, миграции и seed nc2026 (internal/testutil).
// ref.baseline_config — общая таблица для всего тестового набора (как cmdb.ci и т.п. в других
// пакетах): проверки ищут свои фикстуры по уникальному (testutil.Suffix()) маркеру, а деградацию
// покрытия других конфигураций (созданных другими тестами/прогонами) не проверяют.
// Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/baselineconfig/...
package baselineconfig_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/baseline"
	"cmdb/internal/services/baselineconfig"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

// insertCI — как в internal/services/report (общий приём для фикстур классов без формы ручного
// создания, здесь server её имеет, но со своей площадкой и именем, минуя seqalloc/naming).
func insertCI(t *testing.T, class, name, locationCode string) int64 {
	t.Helper()
	var id int64
	_, err := changectx.Run(context.Background(), changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура эталонной конфигурации"},
		func(ctx context.Context, tx orm.TxOrmer) error {
			return tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci (ci_class, name, location_code) VALUES (?, ?, ?) RETURNING id`,
				class, name, locationCode).QueryRow(&id)
		})
	if err != nil {
		t.Fatalf("insertCI(%s): %v", class, err)
	}
	return id
}

func mustExt(t *testing.T, ctx context.Context, id int64, table string, attrs map[string]any) {
	t.Helper()
	if _, err := ci.UpsertExtension(ctx, ci.ExtensionInput{CIID: id, Table: table, Attrs: attrs, UserID: 1, Reason: "фикстура"}); err != nil {
		t.Fatalf("UpsertExtension(%s): %v", table, err)
	}
}

func TestCRUD_RoundTrip(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	name := "TEST-BCFG-CRUD-" + suffix

	id, err := baselineconfig.Create(ctx, baselineconfig.Input{
		Name: name, Description: "тестовая", CIClass: "server", Role: "веб-сервер",
		Attrs: map[string]any{"os_family": "Linux"}, Enabled: true, UserID: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := baselineconfig.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != name || got.CIClass != "server" || got.Role != "веб-сервер" || got.Attrs["os_family"] != "Linux" {
		t.Errorf("Get = %+v, не совпадает с созданным", got)
	}

	if err := baselineconfig.Update(ctx, id, baselineconfig.Input{
		Name: name, CIClass: "server", Attrs: map[string]any{"os_family": "Windows"}, Enabled: false, UserID: 1,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = baselineconfig.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get (после правки): %v", err)
	}
	if got.Attrs["os_family"] != "Windows" || got.Enabled {
		t.Errorf("после Update = %+v, want os_family=Windows enabled=false", got)
	}

	if err := baselineconfig.Delete(ctx, id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := baselineconfig.Get(ctx, id); !errors.Is(err, svcerr.ErrNotFound) {
		t.Errorf("Get после Delete = %v, want ErrNotFound", err)
	}
}

func TestCreate_DuplicateNameRejected(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	name := "TEST-BCFG-DUP-" + testutil.Suffix()
	if _, err := baselineconfig.Create(ctx, baselineconfig.Input{Name: name, CIClass: "server", Enabled: true, UserID: 1}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := baselineconfig.Create(ctx, baselineconfig.Input{Name: name, CIClass: "server", Enabled: true, UserID: 1}); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("Create (дубль имени) = %v, want ErrDuplicate", err)
	}
}

func TestCreate_UnknownAttrColumnRejected(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	_, err := baselineconfig.Create(ctx, baselineconfig.Input{
		Name: "TEST-BCFG-BADATTR-" + testutil.Suffix(), CIClass: "server",
		Attrs: map[string]any{"no_such_column_xyz": "1"}, Enabled: true, UserID: 1,
	})
	if !errors.Is(err, svcerr.ErrInvalidInput) {
		t.Errorf("Create (неизвестная колонка) = %v, want ErrInvalidInput", err)
	}
}

func TestCreate_WithRequiredSoftware(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	var swID int64
	if err := db.QueryRow(`INSERT INTO ref.software (vendor, product, version) VALUES ('TestVendor', $1, '1.0') RETURNING id`,
		"TEST-BCFG-SW-"+suffix).Scan(&swID); err != nil {
		t.Fatalf("ref.software: %v", err)
	}

	id, err := baselineconfig.Create(ctx, baselineconfig.Input{
		Name: "TEST-BCFG-REQSW-" + suffix, CIClass: "server", RequiredSoftware: []int64{swID}, Enabled: true, UserID: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := baselineconfig.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.RequiredSoftware) != 1 || got.RequiredSoftware[0].SoftwareID != swID {
		t.Errorf("RequiredSoftware = %+v, want [%d]", got.RequiredSoftware, swID)
	}

	// Update с пустым списком ПО — полностью заменяет (не добавляет к старому)
	if err := baselineconfig.Update(ctx, id, baselineconfig.Input{Name: got.Name, CIClass: "server", Enabled: true, UserID: 1}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err = baselineconfig.Get(ctx, id)
	if err != nil {
		t.Fatalf("Get (после Update): %v", err)
	}
	if len(got.RequiredSoftware) != 0 {
		t.Errorf("после Update без software_ids RequiredSoftware = %+v, want пусто", got.RequiredSoftware)
	}
}

func TestDeviationsLive_FindsAttrMismatchAndMissingSoftware(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	loc := testutil.NewLocation(t, db) // своя площадка — Deviations не будет пересчитывать чужие КЕ на TestLocation

	ciID := insertCI(t, "server", "TEST-BCFG-DEV-"+suffix, loc)
	mustExt(t, ctx, ciID, "ci_server", map[string]any{"os_family": "Windows"})

	var swID int64
	if err := db.QueryRow(`INSERT INTO ref.software (vendor, product, version) VALUES ('TestVendor', $1, '1.0') RETURNING id`,
		"TEST-BCFG-REQ-"+suffix).Scan(&swID); err != nil {
		t.Fatalf("ref.software: %v", err)
	}

	cfgID, err := baselineconfig.Create(ctx, baselineconfig.Input{
		Name: "TEST-BCFG-CFG-" + suffix, CIClass: "server", LocationCode: loc,
		Attrs: map[string]any{"os_family": "Linux"}, RequiredSoftware: []int64{swID}, Enabled: true, UserID: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	devs, err := baselineconfig.Deviations(ctx, 0)
	if err != nil {
		t.Fatalf("Deviations: %v", err)
	}
	var attrDev, swDev *baselineconfig.Deviation
	for i := range devs {
		if devs[i].ConfigID != cfgID || devs[i].CIID != ciID {
			continue
		}
		switch devs[i].Kind {
		case "attr":
			attrDev = &devs[i]
		case "software":
			swDev = &devs[i]
		}
	}
	if attrDev == nil {
		t.Fatal("нет отклонения по атрибуту os_family")
	}
	if attrDev.Field != "os_family" || attrDev.Expected != "Linux" || attrDev.Actual != "Windows" {
		t.Errorf("отклонение по атрибуту = %+v, want field=os_family expected=Linux actual=Windows", attrDev)
	}
	if swDev == nil {
		t.Fatal("нет отклонения по обязательному ПО")
	}
}

func TestDeviationsLive_NoDeviationWhenMatching(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	loc := testutil.NewLocation(t, db)

	ciID := insertCI(t, "server", "TEST-BCFG-OK-"+suffix, loc)
	mustExt(t, ctx, ciID, "ci_server", map[string]any{"os_family": "Linux"})

	cfgID, err := baselineconfig.Create(ctx, baselineconfig.Input{
		Name: "TEST-BCFG-CFGOK-" + suffix, CIClass: "server", LocationCode: loc,
		Attrs: map[string]any{"os_family": "Linux"}, Enabled: true, UserID: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	devs, err := baselineconfig.Deviations(ctx, 0)
	if err != nil {
		t.Fatalf("Deviations: %v", err)
	}
	for _, d := range devs {
		if d.ConfigID == cfgID && d.CIID == ciID {
			t.Errorf("КЕ соответствует эталону, а отклонение найдено: %+v", d)
		}
	}
}

func TestDeviationsSnapshot_UsesFrozenStateNotLive(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	loc := testutil.NewLocation(t, db)

	ciID := insertCI(t, "server", "TEST-BCFG-SNAP-"+suffix, loc)
	mustExt(t, ctx, ciID, "ci_server", map[string]any{"os_family": "Linux"})

	bID, err := baseline.Create(ctx, "TEST-BL-BCFG-"+suffix, "", time.Now(), baseline.Scope{LocationCode: loc}, 1)
	if err != nil {
		t.Fatalf("baseline.Create: %v", err)
	}
	if err := baseline.Freeze(ctx, bID, 1); err != nil {
		t.Fatalf("baseline.Freeze: %v", err)
	}

	// после заморозки состояние меняется — срез должен остаться «как было»
	mustExt(t, ctx, ciID, "ci_server", map[string]any{"os_family": "Windows"})

	cfgID, err := baselineconfig.Create(ctx, baselineconfig.Input{
		Name: "TEST-BCFG-CFGSNAP-" + suffix, CIClass: "server", LocationCode: loc,
		Attrs: map[string]any{"os_family": "Linux"}, Enabled: true, UserID: 1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// против среза — соответствует (на момент заморозки было Linux)
	devSnap, err := baselineconfig.Deviations(ctx, bID)
	if err != nil {
		t.Fatalf("Deviations(снимок): %v", err)
	}
	for _, d := range devSnap {
		if d.ConfigID == cfgID && d.CIID == ciID {
			t.Errorf("против среза не должно быть отклонения (на момент заморозки было Linux): %+v", d)
		}
	}

	// против текущего состояния — отклонение есть (сейчас Windows)
	devLive, err := baselineconfig.Deviations(ctx, 0)
	if err != nil {
		t.Fatalf("Deviations(live): %v", err)
	}
	found := false
	for _, d := range devLive {
		if d.ConfigID == cfgID && d.CIID == ciID && d.Actual == "Windows" {
			found = true
		}
	}
	if !found {
		t.Error("против текущего состояния отклонение (os_family=Windows) не найдено")
	}
}

func TestDeviationsSnapshot_DraftBaselineRejected(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	loc := testutil.NewLocation(t, db)

	bID, err := baseline.Create(ctx, "TEST-BL-DRAFT-"+suffix, "", time.Now(), baseline.Scope{LocationCode: loc}, 1)
	if err != nil {
		t.Fatalf("baseline.Create: %v", err)
	}
	if _, err := baselineconfig.Deviations(ctx, bID); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("Deviations(черновик) = %v, want ErrInvalidState", err)
	}
}
