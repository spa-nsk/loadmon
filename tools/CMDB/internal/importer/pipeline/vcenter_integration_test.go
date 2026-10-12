// E5.1: коннектор vCenter через весь конвейер на PostgreSQL. Источник — симулятор govmomi (vcsim),
// состояние которого тест меняет настоящими вызовами vSphere API.
package pipeline_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/beego/beego/v2/client/orm"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25/types"

	"cmdb/internal/services/changectx"
	"cmdb/internal/testutil"
)

type sim struct {
	url  *url.URL // с учётными данными
	base string
}

// startVCSim поднимает симулятор. Серийный номер (ServiceTag) у всех хостов vcsim одинаковый —
// у реальных серверов он уникален, поэтому делаем его уникальным по имени хоста и tag.
func startVCSim(t *testing.T, tag string) sim {
	t.Helper()
	model := simulator.VPX()
	if err := model.Create(); err != nil {
		t.Fatalf("vcsim: %v", err)
	}
	t.Cleanup(model.Remove)
	for _, ref := range simulator.Map.AllReference("HostSystem") {
		h := ref.(*simulator.HostSystem)
		// у vcsim структура оборудования общая для всех хостов — копируем, иначе правка зацепит все
		hw := *h.Hardware
		info := append([]types.HostSystemIdentificationInfo(nil), hw.SystemInfo.OtherIdentifyingInfo...)
		for i, o := range info {
			if o.IdentifierType.GetElementDescription().Key == "ServiceTag" {
				info[i].IdentifierValue = "VCSN-" + h.Name + "-" + tag
			}
		}
		hw.SystemInfo.OtherIdentifyingInfo = info
		h.Hardware = &hw
	}
	model.Service.Listen = &url.URL{User: url.UserPassword("vcuser", "vcpass")}
	srv := model.Service.NewServer()
	t.Cleanup(srv.Close)
	return sim{url: srv.URL, base: (&url.URL{Scheme: srv.URL.Scheme, Host: srv.URL.Host}).String()}
}

// retireSimObjects выводит из эксплуатации КЕ прошлых прогонов с фиксированными именами vcsim
// (DC0_*, LocalDS_0): БД тестов общая и постоянная, а имя активной КЕ класса уникально.
func retireSimObjects(t *testing.T) {
	t.Helper()
	_, err := changectx.Run(context.Background(), changectx.Meta{Source: changectx.SourceSystem, UserID: 1},
		func(ctx context.Context, tx orm.TxOrmer) error {
			_, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET lifecycle_stage = 'Retired', updated_at = now()
				WHERE lifecycle_stage NOT IN ('Retired', 'Disposed') AND (name LIKE 'DC0\_%' OR name = 'LocalDS_0')`).Exec()
			return err
		})
	if err != nil {
		t.Fatalf("очистка КЕ vcsim: %v", err)
	}
}

func newVCenterConnector(t *testing.T, db *sql.DB, s sim) string {
	t.Helper()
	t.Setenv("CMDB_VCENTER_TEST_PW", "vcpass")
	code := "vc-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"base_url": s.base, "username": "vcuser", "password_env": "CMDB_VCENTER_TEST_PW",
		"manager_name": "mgr-" + code, "insecure_skip_verify": true})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind) VALUES ('vcenter', 'api') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'vcenter', 'Тестовый vCenter', 'vcenter', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func TestRunConnector_VCenterSnapshotMigrationAndRemoval(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	retireSimObjects(t)
	s := startVCSim(t, randHex(10))
	code := newVCenterConnector(t, db, s)

	first := runConn(t, code)
	// менеджер + кластер + 4 хоста + 4 ВМ + датастор
	if first.Created != 11 || first.Errors != 0 || first.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=11", first)
	}
	// manager contains (кластер + одиночный хост)=2, host member_of=3, vm runs_on=4, vm member_of=2, vm uses_storage=4
	if first.RelCreated != 15 {
		t.Fatalf("первый прогон: связей создано %d, want 15", first.RelCreated)
	}

	var ramGB, vcpu int
	var bios string
	vm := ciID(t, db, "DC0_C0_RP0_VM0")
	if err := db.QueryRow(`SELECT COALESCE(ram_gb, 0), vcpu, bios_uuid::text FROM cmdb.ci_vm WHERE ci_id = $1`, vm).Scan(&ramGB, &vcpu, &bios); err != nil {
		t.Fatalf("ci_vm: %v", err)
	}
	if ramGB != 1 || vcpu != 1 || bios == "" {
		t.Errorf("ci_vm ram_gb=%d vcpu=%d bios=%q, want 1/1/<uuid> (32 МиБ округляются до 1 ГиБ)", ramGB, vcpu, bios)
	}
	var serial string
	if err := db.QueryRow(`SELECT serial_number FROM cmdb.ci WHERE name = 'DC0_C0_H0'`).Scan(&serial); err != nil || !strings.HasPrefix(serial, "VCSN-DC0_C0_H0-") {
		t.Errorf("серийный номер хоста (ServiceTag) = %q, err=%v", serial, err)
	}

	if second := runConn(t, code); second.Created != 0 || second.Updated != 0 || second.Unchanged != 11 || second.RelCreated != 0 || second.RelClosed != 0 {
		t.Fatalf("повторный прогон = %+v, want 11 Unchanged без связей (идемпотентность)", second)
	}

	// ВМ переехала на другой хост кластера — связь runs_on переезжает
	gc, err := govmomi.NewClient(ctx, s.url, true)
	if err != nil {
		t.Fatalf("govmomi: %v", err)
	}
	f := find.NewFinder(gc.Client, true)
	dc, err := f.Datacenter(ctx, "DC0")
	if err != nil {
		t.Fatalf("Datacenter: %v", err)
	}
	f.SetDatacenter(dc)
	simVM, err := f.VirtualMachine(ctx, "DC0_C0_RP0_VM0")
	if err != nil {
		t.Fatalf("VirtualMachine: %v", err)
	}
	cur := openRels(t, db, vm, "runs_on")
	if len(cur) != 1 {
		t.Fatalf("исходный runs_on = %v, want ровно одна связь", cur)
	}
	oldHost := cur[0]
	var oldName string
	db.QueryRow(`SELECT name FROM cmdb.ci WHERE id = $1`, oldHost).Scan(&oldName)
	targetName := "DC0_C0_H0" // другой хост того же кластера
	if oldName == targetName {
		targetName = "DC0_C0_H1"
	}
	newHostObj, err := f.HostSystem(ctx, targetName)
	if err != nil {
		t.Fatalf("HostSystem: %v", err)
	}
	ref := newHostObj.Reference()
	task, err := simVM.Relocate(ctx, types.VirtualMachineRelocateSpec{Host: &ref}, types.VirtualMachineMovePriorityDefaultPriority)
	if err != nil {
		t.Fatalf("Relocate: %v", err)
	}
	if err := task.Wait(ctx); err != nil {
		t.Fatalf("Relocate wait: %v", err)
	}
	moved := runConn(t, code)
	if moved.RelClosed != 1 || moved.RelCreated != 1 || moved.Created != 0 {
		t.Fatalf("прогон после переезда = %+v, want закрыта 1 и создана 1 связь", moved)
	}
	if got := openRels(t, db, vm, "runs_on"); len(got) != 1 || got[0] != ciID(t, db, targetName) {
		t.Fatalf("runs_on после переезда = %v, want [%s]", got, targetName)
	}

	// ВМ удалена в источнике: КЕ не удаляется, через 3 прогона — not_discovered
	victim, err := f.VirtualMachine(ctx, "DC0_H0_VM1")
	if err != nil {
		t.Fatalf("VirtualMachine: %v", err)
	}
	if t2, err := victim.PowerOff(ctx); err == nil {
		_ = t2.Wait(ctx)
	}
	dtask, err := object.NewVirtualMachine(gc.Client, victim.Reference()).Destroy(ctx)
	if err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if err := dtask.Wait(ctx); err != nil {
		t.Fatalf("Destroy wait: %v", err)
	}
	for i := 0; i < 3; i++ {
		runConn(t, code)
	}
	var status sql.NullString
	if err := db.QueryRow(`SELECT lifecycle_status FROM cmdb.ci WHERE name = 'DC0_H0_VM1' AND lifecycle_stage <> 'Retired'`).Scan(&status); err != nil {
		t.Fatalf("КЕ удалённой ВМ: %v", err)
	}
	if status.String != "not_discovered" {
		t.Fatalf("lifecycle_status = %q, want not_discovered", status.String)
	}
}
