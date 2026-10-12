// Интеграционные тесты таблиц-расширений КЕ: нужны миграции до 0013 и seed nc2026 (включая 005).
package ci

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

// insertCI создаёт КЕ заданного класса напрямую (имя без проверки профиля — тесту важна только
// таблица-расширение) и возвращает id.
func insertCI(t *testing.T, class string) int64 {
	t.Helper()
	var id int64
	_, err := changectx.Run(context.Background(), changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "фикстура расширения"},
		func(ctx context.Context, tx orm.TxOrmer) error {
			return tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci (ci_class, name) VALUES (?, ?) RETURNING id`,
				class, "TEST-EXT-"+class+"-"+testutil.Suffix()).QueryRow(&id)
		})
	if err != nil {
		t.Fatalf("insertCI(%s): %v", class, err)
	}
	return id
}

func count(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

// Для каждой таблицы-расширения: создание → повтор тех же значений (без новой версии) → изменение
// (закрытая версия в hist + diff в audit.change_log). Проверяет и сами таблицы миграций
// 0007/0013, и универсальный UpsertExtension.
func TestUpsertExtension_AllTables(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()

	cases := []struct {
		table, class   string
		first, changed map[string]any
		changedAttr    string
	}{
		{"ci_server", "server", map[string]any{"cpu_count": 2, "os_family": "Linux", "bmc_ip": "10.1.1.1"}, map[string]any{"cpu_count": 4}, "cpu_count"},
		{"ci_blade_chassis", "blade_chassis", map[string]any{"bays_total": 16, "oa_ip": "10.1.1.2"}, map[string]any{"bays_used": 3}, "bays_used"},
		{"ci_hypervisor", "hypervisor_host", map[string]any{"hv_type": "esxi", "platform": "vSphere"}, map[string]any{"maintenance_mode": true}, "maintenance_mode"},
		{"ci_vm", "vm", map[string]any{"vcpu": 4, "guest_os": "Astra Linux"}, map[string]any{"vcpu": 8}, "vcpu"},
		{"ci_cluster", "cluster", map[string]any{"cluster_kind": "virt", "ha": true}, map[string]any{"drs": true}, "drs"},
		{"ci_k8s", "cluster", map[string]any{"distribution": "Deckhouse", "node_count": 5}, map[string]any{"node_count": 6}, "node_count"},
		{"ci_virt_manager", "virt_manager", map[string]any{"product": "vCenter", "version": "8.0"}, map[string]any{"version": "8.0.2"}, "version"},
		{"ci_datastore", "datastore", map[string]any{"ds_type": "nfs", "capacity_gb": 1024.5}, map[string]any{"capacity_gb": 2048}, "capacity_gb"},
		{"ci_storage_array", "storage_array", map[string]any{"array_family": "FAS", "controllers": 2, "mgmt_ip": "10.1.2.1"}, map[string]any{"usable_tb": 120.75}, "usable_tb"},
		{"ci_storage_pool", "storage_pool", map[string]any{"raid_type": "RAID-DP", "capacity_tb": 50}, map[string]any{"tier": "t2"}, "tier"},
		{"ci_svm", "svm", map[string]any{"mgmt_ip": "10.1.2.2"}, map[string]any{"mgmt_ip": "10.1.2.3"}, "mgmt_ip"},
		{"ci_volume", "volume", map[string]any{"lun_id": 7, "size_gb": 500, "thin": true}, map[string]any{"size_gb": 750}, "size_gb"},
		{"ci_qtree", "qtree", map[string]any{"path": "/vol/a/q1"}, map[string]any{"quota_gb": 100}, "quota_gb"},
		{"ci_share", "share", map[string]any{"share_type": "smb", "path": `\\fs01\share`}, map[string]any{"path": `\\fs01\share2`}, "path"},
		{"ci_object_store", "object_store", map[string]any{"product": "MinIO", "capacity_tb": 200}, map[string]any{"endpoint_url": "https://s3.test"}, "endpoint_url"},
		{"ci_fc_switch", "fc_switch", map[string]any{"fabric": "A", "domain_id": 5}, map[string]any{"ports_total": 48}, "ports_total"},
		{"ci_san_zone", "san_zone", map[string]any{"fabric": "A", "members": []string{"50:01:43:80:00:00:00:01", "50:01:43:80:00:00:00:02"}}, map[string]any{"zoneset": "prod"}, "zoneset"},
		{"ci_tape_library", "tape_library", map[string]any{"drives": 4, "slots": 100}, map[string]any{"is_virtual": true}, "is_virtual"},
		{"ci_backup", "backup_server", map[string]any{"product": "RuBackup"}, map[string]any{"version": "2.1"}, "version"},
		{"ci_network_device", "net_switch", map[string]any{"device_role": "leaf", "mgmt_ip": "10.1.3.1"}, map[string]any{"ha_group": "vpc1"}, "ha_group"},
		{"ci_facility", "facility_ups", map[string]any{"kva": 40, "phase": 3}, map[string]any{"erm_count": 2}, "erm_count"},
		{"ci_db_instance", "db_instance", map[string]any{"db_tech": "P", "port": 5432}, map[string]any{"version": "17"}, "version"},
		{"ci_cluster_resource", "cluster_resource", map[string]any{"resource_code": "Q", "instance_no": 1}, map[string]any{"instance_no": 2}, "instance_no"},
	}

	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			ciID := insertCI(t, tc.class)
			first := map[string]any{}
			for k, v := range tc.first {
				first[k] = v
			}
			if tc.table == "ci_cluster_resource" {
				first["cluster_ci_id"] = insertCI(t, "cluster") // обязательная ссылка на кластер
			}
			in := ExtensionInput{CIID: ciID, Table: tc.table, Attrs: first, Reason: "тест расширения", UserID: 1}

			created, err := UpsertExtension(ctx, in)
			if err != nil {
				t.Fatalf("создание: %v", err)
			}
			if !created.Created || !created.Changed || created.RowVersion != 1 {
				t.Fatalf("создание: %+v, want Created/Changed, версия 1", created)
			}

			same, err := UpsertExtension(ctx, in)
			if err != nil {
				t.Fatalf("повтор: %v", err)
			}
			if same.Created || same.Changed || same.RowVersion != 1 {
				t.Errorf("повтор тех же значений: %+v, want без изменений, версия 1", same)
			}
			if n := count(t, db, `SELECT count(*) FROM hist.`+tc.table+` WHERE ci_id = $1`, ciID); n != 0 {
				t.Errorf("после повтора в hist.%s %d строк, want 0", tc.table, n)
			}

			in.Attrs = tc.changed
			upd, err := UpsertExtension(ctx, in)
			if err != nil {
				t.Fatalf("изменение: %v", err)
			}
			if upd.Created || !upd.Changed || upd.RowVersion != 2 {
				t.Errorf("изменение: %+v, want Changed, версия 2", upd)
			}
			if n := count(t, db, `SELECT count(*) FROM hist.`+tc.table+` WHERE ci_id = $1 AND row_version = 1 AND hist_op = 'U'`, ciID); n != 1 {
				t.Errorf("hist.%s: закрытых версий %d, want 1", tc.table, n)
			}
			// Аудит привязан к КЕ (ci_id) и содержит diff по изменённому атрибуту.
			if n := count(t, db, `SELECT count(*) FROM audit.change_log WHERE change_set_id = $1 AND entity = $2 AND ci_id = $3 AND op = 'U' AND attr_name = $4`,
				upd.ChangeSetID, tc.table, ciID, tc.changedAttr); n != 1 {
				t.Errorf("audit.change_log: diff по %s = %d строк, want 1", tc.changedAttr, n)
			}
		})
	}
}

func TestUpsertExtension_Validation(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	ciID := insertCI(t, "server")
	base := ExtensionInput{CIID: ciID, Table: "ci_server", Attrs: map[string]any{"cpu_count": 2}, Reason: "тест", UserID: 1}

	bad := base
	bad.Table = "ci" // не расширение
	if _, err := UpsertExtension(ctx, bad); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("таблица не из белого списка: err = %v, want ErrInvalidState", err)
	}
	bad = base
	bad.Attrs = map[string]any{"no_such_column": 1}
	if _, err := UpsertExtension(ctx, bad); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("несуществующая колонка: err = %v, want ErrInvalidState", err)
	}
	bad = base
	bad.Attrs = map[string]any{"row_version": 99}
	if _, err := UpsertExtension(ctx, bad); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("служебная колонка: err = %v, want ErrInvalidState", err)
	}
	bad = base
	bad.Attrs = nil
	if _, err := UpsertExtension(ctx, bad); !errors.Is(err, svcerr.ErrNoChanges) {
		t.Errorf("пустые атрибуты: err = %v, want ErrNoChanges", err)
	}
	bad = base
	bad.CIID = -1
	if _, err := UpsertExtension(ctx, bad); !errors.Is(err, svcerr.ErrNotFound) {
		t.Errorf("несуществующая КЕ: err = %v, want ErrNotFound", err)
	}
	bad = base
	bad.Attrs = map[string]any{"cpu_count": "много"}
	if _, err := UpsertExtension(ctx, bad); err == nil {
		t.Error("нечисловое значение для int-колонки должно быть отклонено БД")
	}

	if _, err := UpsertExtension(ctx, base); err != nil {
		t.Fatalf("создание: %v", err)
	}
	stale := 7
	conflict := base
	conflict.Attrs = map[string]any{"cpu_count": 8}
	conflict.ExpectedVersion = &stale
	if _, err := UpsertExtension(ctx, conflict); !errors.Is(err, svcerr.ErrVersionConflict) {
		t.Errorf("устаревшая версия: err = %v, want ErrVersionConflict", err)
	}

	// NULL сбрасывает значение.
	reset := base
	reset.Attrs = map[string]any{"cpu_count": nil}
	res, err := UpsertExtension(ctx, reset)
	if err != nil || !res.Changed {
		t.Errorf("сброс в NULL: %+v, err = %v; want Changed", res, err)
	}
}

// ci_software_install — не расширение 1:1, а список установок; проверяем версионирование и
// уникальность установки.
func TestSoftwareInstall_VersioningAndUniqueness(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	ciID := insertCI(t, "server")

	product := "TestProduct-" + testutil.Suffix()
	if _, err := db.Exec(`INSERT INTO ref.software (vendor, product, version) VALUES ('TestVendor', $1, '1.0')`, product); err != nil {
		t.Fatalf("ref.software: %v", err)
	}
	var swID int64
	if err := db.QueryRow(`SELECT id FROM ref.software WHERE product = $1`, product).Scan(&swID); err != nil {
		t.Fatalf("select ref.software: %v", err)
	}

	var instID int64
	run := func(reason string, fn func(ctx context.Context, tx orm.TxOrmer) error) error {
		_, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: reason}, fn)
		return err
	}
	if err := run("установка ПО", func(ctx context.Context, tx orm.TxOrmer) error {
		return tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci_software_install (ci_id, software_id, version, install_path)
			VALUES (?, ?, '1.0', '/opt/test') RETURNING id`, ciID, swID).QueryRow(&instID)
	}); err != nil {
		t.Fatalf("установка: %v", err)
	}

	dup := run("дубль установки", func(ctx context.Context, tx orm.TxOrmer) error {
		_, err := tx.RawWithCtx(ctx, `INSERT INTO cmdb.ci_software_install (ci_id, software_id, version, install_path)
			VALUES (?, ?, '1.0', '/opt/test')`, ciID, swID).Exec()
		return err
	})
	if !svcerr.ViolatesConstraint(dup, "ci_software_install_uniq") {
		t.Errorf("дубль установки: err = %v, want нарушение ci_software_install_uniq", dup)
	}

	if err := run("обновление ПО", func(ctx context.Context, tx orm.TxOrmer) error {
		_, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci_software_install SET version = '1.1', updated_at = now() WHERE id = ?`, instID).Exec()
		return err
	}); err != nil {
		t.Fatalf("обновление: %v", err)
	}
	if n := count(t, db, `SELECT count(*) FROM hist.ci_software_install WHERE id = $1 AND version = '1.0'`, instID); n != 1 {
		t.Errorf("hist.ci_software_install: версий 1.0 = %d, want 1", n)
	}
}
