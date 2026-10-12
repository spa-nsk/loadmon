// Типовые отчёты (plan.md E7.5) — реальный PostgreSQL, миграции и seed nc2026 (internal/testutil).
// cmdb.ci — общая таблица для всего тестового набора (см. internal/services/quality для того же
// приёма): каждая фикстура получает уникальное (testutil.Suffix()) имя/значение и ищется в
// результате по нему, а не по позиции или общему счётчику — так проверки не зависят от того, что
// накопили другие пакеты в общей cmdb_test.
// Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/report/...
package report_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/relation"
	"cmdb/internal/services/report"
	"cmdb/internal/testutil"
)

// insertCI создаёт КЕ произвольного класса напрямую (как internal/services/ci.insertCI в
// extension_integration_test.go) — отчётам нужны классы (storage_array, storage_pool, volume,
// cluster, hypervisor_host) без формы ручного создания (internal/services/ci.SpecFor), поэтому
// ci.Create/CreateServer им не подходят. locationCode="" — NULL.
func insertCI(t *testing.T, class, name, locationCode string) int64 {
	t.Helper()
	var loc any
	if locationCode != "" {
		loc = locationCode
	}
	var id int64
	_, err := changectx.Run(context.Background(), changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура отчёта"},
		func(ctx context.Context, tx orm.TxOrmer) error {
			return tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci (ci_class, name, location_code) VALUES (?, ?, ?) RETURNING id`,
				class, name, loc).QueryRow(&id)
		})
	if err != nil {
		t.Fatalf("insertCI(%s): %v", class, err)
	}
	return id
}

func mustExt(t *testing.T, ctx context.Context, id int64, table string, attrs map[string]any) {
	t.Helper()
	if _, err := ci.UpsertExtension(ctx, ci.ExtensionInput{CIID: id, Table: table, Attrs: attrs, UserID: 1, Reason: "фикстура отчёта"}); err != nil {
		t.Fatalf("UpsertExtension(%s): %v", table, err)
	}
}

func TestSumTB(t *testing.T) {
	rows := []repo.Row{{"tb": "1.5"}, {"tb": "2.5"}, {"tb": "не число"}, {"tb": ""}}
	if got := report.SumTB(rows, "tb"); got != 4 {
		t.Errorf("SumTB = %v, want 4 (нечисловые/пустые значения пропускаются)", got)
	}
	if got := report.SumTB(nil, "tb"); got != 0 {
		t.Errorf("SumTB(nil) = %v, want 0", got)
	}
}

func TestInventory_FindsFixtureByUniqueRack(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	name := "TEST-INV-" + suffix
	rack := "RACK-" + suffix

	id := insertCI(t, "server", name, testutil.TestLocation)
	if _, err := ci.Update(ctx, ci.UpdateInput{CIID: id, ExpectedVersion: 1, Reason: "фикстура инвентаря " + suffix, UserID: 1, RackPosition: &rack}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	rows, err := report.InventoryRows(ctx, testutil.TestLocation)
	if err != nil {
		t.Fatalf("InventoryRows: %v", err)
	}
	found := false
	for _, r := range rows {
		if r["name"] == name {
			found = true
			if r["rack_position"] != rack {
				t.Errorf("rack_position = %q, want %q", r["rack_position"], rack)
			}
		}
	}
	if !found {
		t.Fatalf("InventoryRows(%q): фикстура %q не найдена", testutil.TestLocation, name)
	}

	summary, err := report.InventorySummary(ctx, testutil.TestLocation)
	if err != nil {
		t.Fatalf("InventorySummary: %v", err)
	}
	foundSummary := false
	for _, r := range summary {
		if r["rack_position"] == rack {
			foundSummary = true
			if r["n"] != "1" {
				t.Errorf("InventorySummary: n = %q для уникальной стойки %q, want 1", r["n"], rack)
			}
		}
	}
	if !foundSummary {
		t.Fatalf("InventorySummary: нет строки для стойки %q", rack)
	}
}

// TestInventory_LocationFilterExcludesOthers — площадка в фильтре реально сужает выборку: КЕ на
// другой площадке с тем же уникальным маркером не попадает в результат.
func TestInventory_LocationFilterExcludesOthers(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	other := testutil.NewLocation(t, db)
	name := "TEST-INV-OTHER-" + suffix

	insertCI(t, "server", name, other)

	rows, err := report.InventoryRows(ctx, testutil.TestLocation)
	if err != nil {
		t.Fatalf("InventoryRows: %v", err)
	}
	for _, r := range rows {
		if r["name"] == name {
			t.Fatalf("InventoryRows(%q) содержит КЕ %q с другой площадки (%q)", testutil.TestLocation, name, other)
		}
	}
}

func toFloat(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("не число: %q: %v", s, err)
	}
	return v
}

func TestStorageCapacity_ArrayPoolVolume(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	arrayName, poolName, volName := "TEST-ARR-"+suffix, "TEST-POOL-"+suffix, "TEST-VOL-"+suffix

	arrID := insertCI(t, "storage_array", arrayName, "")
	poolID := insertCI(t, "storage_pool", poolName, "")
	volID := insertCI(t, "volume", volName, "")

	mustExt(t, ctx, arrID, "ci_storage_array", map[string]any{"raw_tb": 100, "usable_tb": 80})
	mustExt(t, ctx, poolID, "ci_storage_pool", map[string]any{"array_ci_id": arrID, "capacity_tb": 50})
	mustExt(t, ctx, volID, "ci_volume", map[string]any{"array_ci_id": arrID, "pool_ci_id": poolID, "size_gb": 2048}) // 2 ТБ

	arrays, err := report.StorageArrays(ctx)
	if err != nil {
		t.Fatalf("StorageArrays: %v", err)
	}
	var arrRow map[string]string
	for _, r := range arrays {
		if r["name"] == arrayName {
			arrRow = r
		}
	}
	if arrRow == nil {
		t.Fatalf("StorageArrays: массив %q не найден", arrayName)
	}
	if v := toFloat(t, arrRow["raw_tb"]); v != 100 {
		t.Errorf("raw_tb = %v, want 100", v)
	}
	if v := toFloat(t, arrRow["usable_tb"]); v != 80 {
		t.Errorf("usable_tb = %v, want 80", v)
	}
	if v := toFloat(t, arrRow["allocated_tb"]); v != 2 {
		t.Errorf("allocated_tb = %v, want 2 (2048 ГБ тома)", v)
	}

	pools, err := report.StoragePools(ctx)
	if err != nil {
		t.Fatalf("StoragePools: %v", err)
	}
	var poolRow map[string]string
	for _, r := range pools {
		if r["name"] == poolName {
			poolRow = r
		}
	}
	if poolRow == nil {
		t.Fatalf("StoragePools: пул %q не найден", poolName)
	}
	if poolRow["array_name"] != arrayName {
		t.Errorf("array_name = %q, want %q", poolRow["array_name"], arrayName)
	}
	if v := toFloat(t, poolRow["capacity_tb"]); v != 50 {
		t.Errorf("capacity_tb = %v, want 50", v)
	}
	if v := toFloat(t, poolRow["allocated_tb"]); v != 2 {
		t.Errorf("allocated_tb = %v, want 2", v)
	}
}

func TestVMsByCluster_JoinsHostAndCluster(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	clusterName, hostName, vmName := "TEST-CL-"+suffix, "TEST-HOST-"+suffix, "TEST-VM-"+suffix

	clID := insertCI(t, "cluster", clusterName, "")
	hostID := insertCI(t, "hypervisor_host", hostName, "")
	vmID := insertCI(t, "vm", vmName, "")

	if _, _, err := relation.Create(ctx, relation.CreateInput{RelType: "runs_on", SrcCIID: vmID, DstCIID: hostID, Reason: "фикстура кластера " + suffix, UserID: 1}); err != nil {
		t.Fatalf("relation.Create(runs_on): %v", err)
	}
	if _, _, err := relation.Create(ctx, relation.CreateInput{RelType: "member_of", SrcCIID: hostID, DstCIID: clID, Reason: "фикстура кластера " + suffix, UserID: 1}); err != nil {
		t.Fatalf("relation.Create(member_of): %v", err)
	}

	// clID уникален для этого прогона — фильтр по нему изолирует ровно нашу фикстуру, точное
	// сравнение здесь не флаки в отличие от глобальных агрегатов.
	rows, err := report.VMsByCluster(ctx, clID)
	if err != nil {
		t.Fatalf("VMsByCluster: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("VMsByCluster(%d) = %d строк, want 1: %+v", clID, len(rows), rows)
	}
	r := rows[0]
	if r["name"] != vmName || r["cluster_name"] != clusterName || r["host_name"] != hostName {
		t.Errorf("строка = %+v, want name=%q cluster_name=%q host_name=%q", r, vmName, clusterName, hostName)
	}
}

// TestVMsByCluster_NoMembershipShowsEmpty — ВМ на хосте вне кластера не теряется (LEFT JOIN), а
// получает пустое имя кластера.
func TestVMsByCluster_NoMembershipShowsEmpty(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	hostName, vmName := "TEST-HOST-NOCLUSTER-"+suffix, "TEST-VM-NOCLUSTER-"+suffix

	hostID := insertCI(t, "hypervisor_host", hostName, "")
	vmID := insertCI(t, "vm", vmName, "")
	if _, _, err := relation.Create(ctx, relation.CreateInput{RelType: "runs_on", SrcCIID: vmID, DstCIID: hostID, Reason: "фикстура " + suffix, UserID: 1}); err != nil {
		t.Fatalf("relation.Create: %v", err)
	}

	rows, err := report.VMsByCluster(ctx, 0)
	if err != nil {
		t.Fatalf("VMsByCluster: %v", err)
	}
	found := false
	for _, r := range rows {
		if r["name"] == vmName {
			found = true
			if r["host_name"] != hostName {
				t.Errorf("host_name = %q, want %q", r["host_name"], hostName)
			}
			if r["cluster_name"] != "" {
				t.Errorf("cluster_name = %q, want пусто (хост не в кластере)", r["cluster_name"])
			}
		}
	}
	if !found {
		t.Fatalf("VMsByCluster(0): ВМ %q не найдена", vmName)
	}
}

func TestWithoutOwner_FindsFixtureAndExcludesWithOwner(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	noOwnerName := "TEST-NOOWN-" + suffix
	withOwnerName := "TEST-HASOWN-" + suffix

	insertCI(t, "server", noOwnerName, "")
	withOwnerID := insertCI(t, "server", withOwnerName, "")
	if _, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура владельца " + suffix},
		func(ctx context.Context, tx orm.TxOrmer) error {
			_, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET owner_person_id = 1 WHERE id = ?`, withOwnerID).Exec()
			return err
		}); err != nil {
		t.Fatalf("проставление владельца: %v", err)
	}

	rows, err := report.WithoutOwner(ctx)
	if err != nil {
		t.Fatalf("WithoutOwner: %v", err)
	}
	var hasNoOwner, hasWithOwner bool
	for _, r := range rows {
		if r["name"] == noOwnerName {
			hasNoOwner = true
		}
		if r["name"] == withOwnerName {
			hasWithOwner = true
		}
	}
	if !hasNoOwner {
		t.Errorf("WithoutOwner: %q без владельца не найдена", noOwnerName)
	}
	if hasWithOwner {
		t.Errorf("WithoutOwner: %q с владельцем не должна попадать в отчёт", withOwnerName)
	}
}

func TestNamingDeviations_FindsFixture(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	name := "TEST-LEGACY-" + suffix

	insertCI(t, "server", name, "") // naming_status по умолчанию 'exception' (не создана через профиль)

	rows, err := report.NamingDeviations(ctx)
	if err != nil {
		t.Fatalf("NamingDeviations: %v", err)
	}
	found := false
	for _, r := range rows {
		if r["name"] == name {
			found = true
			if r["naming_status"] != "exception" {
				t.Errorf("naming_status = %q, want exception", r["naming_status"])
			}
		}
	}
	if !found {
		t.Fatalf("NamingDeviations: %q не найдена", name)
	}
}

// TestForbiddenSoftware_FindsForbiddenExcludesAllowed — установка с политикой forbidden найдена,
// с allowed — нет; серийные значения (продукт, КЕ) уникальны для прогона.
func TestForbiddenSoftware_FindsForbiddenExcludesAllowed(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()

	mkSoftware := func(policy string) int64 {
		var id int64
		if err := db.QueryRow(`INSERT INTO ref.software (vendor, product, version, policy) VALUES ('TestVendor', $1, '1.0', $2) RETURNING id`,
			"TestSW-"+policy+"-"+suffix, policy).Scan(&id); err != nil {
			t.Fatalf("ref.software(%s): %v", policy, err)
		}
		return id
	}
	forbiddenSW, allowedSW := mkSoftware("forbidden"), mkSoftware("allowed")

	// insertCI пишет только cmdb.ci — ci_software_install ссылается на неё напрямую, строка
	// таблицы-расширения (ci_server) для этого теста не нужна.
	forbiddenCI := insertCI(t, "server", "TEST-FORBIDDEN-CI-"+suffix, testutil.TestLocation)
	allowedCI := insertCI(t, "server", "TEST-ALLOWED-CI-"+suffix, testutil.TestLocation)

	for id, sw := range map[int64]int64{forbiddenCI: forbiddenSW, allowedCI: allowedSW} {
		if _, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура установки ПО"},
			func(ctx context.Context, tx orm.TxOrmer) error {
				_, err := tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci_software_install (ci_id, software_id, version) VALUES (?, ?, '1.0')`, id, sw).Exec()
				return err
			}); err != nil {
			t.Fatalf("ci_software_install: %v", err)
		}
	}

	rows, err := report.ForbiddenSoftware(ctx)
	if err != nil {
		t.Fatalf("ForbiddenSoftware: %v", err)
	}
	var foundForbidden, foundAllowed bool
	for _, r := range rows {
		if r["id"] == strconv.FormatInt(forbiddenCI, 10) {
			foundForbidden = true
			if r["policy"] != "forbidden" {
				t.Errorf("policy = %q, want forbidden", r["policy"])
			}
		}
		if r["id"] == strconv.FormatInt(allowedCI, 10) {
			foundAllowed = true
		}
	}
	if !foundForbidden {
		t.Errorf("ForbiddenSoftware не нашёл КЕ %d с forbidden-ПО", forbiddenCI)
	}
	if foundAllowed {
		t.Errorf("ForbiddenSoftware не должен находить КЕ %d с allowed-ПО", allowedCI)
	}
}

// TestNamingDeviations_CompliantCIExcluded — КЕ, созданная через профиль именования (compliant),
// не попадает в отчёт об отклонениях.
func TestNamingDeviations_CompliantCIExcluded(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	suffix := testutil.Suffix()
	res, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "фикстура именования " + suffix, UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	rows, err := report.NamingDeviations(ctx)
	if err != nil {
		t.Fatalf("NamingDeviations: %v", err)
	}
	for _, r := range rows {
		if r["name"] == res.Name {
			t.Errorf("NamingDeviations: compliant КЕ %q не должна попадать в отчёт", res.Name)
		}
	}
}
