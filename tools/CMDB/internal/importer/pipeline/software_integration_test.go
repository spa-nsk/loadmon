// E7.7: internal/importer/pipeline.syncSoftware напрямую (не через полный Run — ни один коннектор
// 1-й волны (E5) ещё не сообщает ПО, полного чёрного ящика для этого нет). Реальный PostgreSQL.
// Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/importer/pipeline/...
package pipeline

import (
	"context"
	"testing"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/importer/canon"
	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

func newServerForSoftwareTest(t *testing.T) int64 {
	t.Helper()
	res, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: testutil.TestLocation, Reason: "фикстура ПО " + testutil.Suffix(), UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	return res.CIID
}

func withChangeCtx(t *testing.T, fn func(ctx context.Context, tx orm.TxOrmer) error) {
	t.Helper()
	if _, err := changectx.Run(context.Background(), changectx.Meta{Source: changectx.SourceImport, Reason: "", UserID: 1}, fn); err != nil {
		t.Fatalf("changectx.Run: %v", err)
	}
}

// TestSyncSoftware_CreatesRefAndInstall — новое ПО создаёт запись в ref.software (find-or-create)
// и установку в cmdb.ci_software_install.
func TestSyncSoftware_CreatesRefAndInstall(t *testing.T) {
	db := testutil.DB(t)
	suffix := testutil.Suffix()
	ciID := newServerForSoftwareTest(t)
	product := "SyncTest-" + suffix

	withChangeCtx(t, func(ctx context.Context, tx orm.TxOrmer) error {
		changed, err := syncSoftware(ctx, tx, ciID, []canon.SoftwareInstall{
			{Vendor: "SyncVendor", Product: product, Version: "2.0", InstallPath: "/usr/local"},
		})
		if err != nil {
			return err
		}
		if !changed {
			t.Error("syncSoftware: changed=false, want true (новая установка)")
		}
		return nil
	})

	var n int
	if err := db.QueryRow(`
		SELECT count(*) FROM cmdb.ci_software_install si JOIN ref.software sw ON sw.id = si.software_id
		WHERE si.ci_id = $1 AND sw.vendor = 'SyncVendor' AND sw.product = $2 AND si.version = '2.0' AND si.install_path = '/usr/local'
	`, ciID, product).Scan(&n); err != nil {
		t.Fatalf("проверка установки: %v", err)
	}
	if n != 1 {
		t.Fatalf("установок = %d, want 1", n)
	}
}

// TestSyncSoftware_FindsExistingSoftwareWithNilVersion — find-or-create не плодит дубли в
// ref.software при NULL version/edition: UNIQUE-ограничение там объявлено без NULLS NOT DISTINCT
// (0002_ref_foundation.sql), поэтому findOrCreateSoftware ищет сам через IS NOT DISTINCT FROM,
// а не полагается на ON CONFLICT — этот тест ловит именно регресс в эту сторону.
func TestSyncSoftware_FindsExistingSoftwareWithNilVersion(t *testing.T) {
	db := testutil.DB(t)
	suffix := testutil.Suffix()
	ciID1, ciID2 := newServerForSoftwareTest(t), newServerForSoftwareTest(t)
	product := "NilVerTest-" + suffix

	install := []canon.SoftwareInstall{{Vendor: "NilVerVendor", Product: product}} // version="" -> NULL
	withChangeCtx(t, func(ctx context.Context, tx orm.TxOrmer) error {
		_, err := syncSoftware(ctx, tx, ciID1, install)
		return err
	})
	withChangeCtx(t, func(ctx context.Context, tx orm.TxOrmer) error {
		_, err := syncSoftware(ctx, tx, ciID2, install)
		return err
	})

	var swCount int
	if err := db.QueryRow(`SELECT count(*) FROM ref.software WHERE vendor = 'NilVerVendor' AND product = $1`, product).Scan(&swCount); err != nil {
		t.Fatalf("count ref.software: %v", err)
	}
	if swCount != 1 {
		t.Errorf("ref.software: %d записей для одного ПО без версии, want 1 (не должно дублироваться)", swCount)
	}
}

// TestSyncSoftware_RepeatDoesNotDuplicate — повторная синхронизация того же снимка не плодит строк
// в cmdb.ci_software_install (ON CONFLICT ... DO UPDATE, не INSERT), но и changed=true каждый раз
// (detected_at обновляется — это ожидаемое поведение "источник снова подтвердил").
func TestSyncSoftware_RepeatDoesNotDuplicate(t *testing.T) {
	db := testutil.DB(t)
	suffix := testutil.Suffix()
	ciID := newServerForSoftwareTest(t)
	install := []canon.SoftwareInstall{{Vendor: "RepeatVendor", Product: "RepeatTest-" + suffix, Version: "1.0"}}

	for i := 0; i < 2; i++ {
		withChangeCtx(t, func(ctx context.Context, tx orm.TxOrmer) error {
			_, err := syncSoftware(ctx, tx, ciID, install)
			return err
		})
	}

	var n int
	if err := db.QueryRow(`
		SELECT count(*) FROM cmdb.ci_software_install si JOIN ref.software sw ON sw.id = si.software_id
		WHERE si.ci_id = $1 AND sw.vendor = 'RepeatVendor'
	`, ciID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("после двух прогонов установок = %d, want 1 (не дублируется)", n)
	}
}
