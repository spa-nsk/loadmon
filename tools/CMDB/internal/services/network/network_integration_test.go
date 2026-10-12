// Интеграционные тесты сетевой части: PostgreSQL с миграциями до 0012 и seed nc2026.
// CMDB_TEST_DB_DSN=... go test ./internal/services/network/... -v
package network

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

var testMeta = Meta{Reason: "интеграционный тест сети", UserID: 1}

func newServer(t *testing.T) int64 {
	t.Helper()
	res, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		Site: testutil.TestSite, LocationCode: testutil.TestLocation, Reason: "фикстура сети", UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	return res.CIID
}

func clock(t *testing.T, db *sql.DB) time.Time {
	t.Helper()
	var ts time.Time
	if err := db.QueryRow(`SELECT clock_timestamp()`).Scan(&ts); err != nil {
		t.Fatalf("clock_timestamp: %v", err)
	}
	return ts
}

func TestVLANAndSubnet_UniquenessAndValidation(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	loc := testutil.NewLocation(t, db)
	vrf := "t" + testutil.Suffix()

	vlanID, _, err := CreateVLAN(ctx, VLANInput{Number: 100, Name: "mgmt", LocationCode: loc, Meta: testMeta})
	if err != nil {
		t.Fatalf("CreateVLAN: %v", err)
	}
	if _, _, err := CreateVLAN(ctx, VLANInput{Number: 100, Name: "дубль", LocationCode: loc, Meta: testMeta}); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("повторный VLAN на площадке: err = %v, want ErrDuplicate", err)
	}
	if _, _, err := CreateVLAN(ctx, VLANInput{Number: 100, Name: "другая площадка", LocationCode: testutil.NewLocation(t, db), Meta: testMeta}); err != nil {
		t.Errorf("тот же номер на другой площадке должен быть допустим: %v", err)
	}

	if _, _, err := CreateSubnet(ctx, SubnetInput{VRF: vrf, CIDR: "10.20.0.0/24", VLANID: &vlanID, LocationCode: loc, Gateway: "10.20.0.1", Meta: testMeta}); err != nil {
		t.Fatalf("CreateSubnet: %v", err)
	}
	if _, _, err := CreateSubnet(ctx, SubnetInput{VRF: vrf, CIDR: "10.20.0.0/24", Meta: testMeta}); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("повторная подсеть: err = %v, want ErrDuplicate", err)
	}
	if _, _, err := CreateSubnet(ctx, SubnetInput{VRF: vrf, CIDR: "10.20.0.5/24", Meta: testMeta}); err == nil {
		t.Error("подсеть с битами хоста (10.20.0.5/24) должна быть отклонена")
	}
}

func TestInterface_NormalizesWWNAndRejectsDuplicates(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	srvA, srvB := newServer(t), newServer(t)
	wwpn := testutil.RandomWWN()

	id, _, err := AddInterface(ctx, InterfaceInput{CIID: srvA, Name: "fc0", IfType: "fc", WWPN: upperWWN(wwpn), Meta: testMeta})
	if err != nil {
		t.Fatalf("AddInterface: %v", err)
	}
	var stored string
	if err := db.QueryRow(`SELECT wwpn FROM cmdb.interface WHERE id = $1`, id).Scan(&stored); err != nil {
		t.Fatalf("select interface: %v", err)
	}
	if stored != wwpn {
		t.Errorf("wwpn в БД = %q, want нормализованный %q", stored, wwpn)
	}

	if _, _, err := AddInterface(ctx, InterfaceInput{CIID: srvA, Name: "fc0", IfType: "fc", Meta: testMeta}); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("повторное имя интерфейса: err = %v, want ErrDuplicate", err)
	}
	if _, _, err := AddInterface(ctx, InterfaceInput{CIID: srvB, Name: "fc0", IfType: "fc", WWPN: wwpn, Meta: testMeta}); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("повторный WWPN: err = %v, want ErrDuplicate", err)
	}
	if _, _, err := AddInterface(ctx, InterfaceInput{CIID: srvA, Name: "bad", WWPN: "zz", Meta: testMeta}); !errors.Is(err, svcerr.ErrInvalidInput) {
		t.Errorf("некорректный WWPN: err = %v, want ErrInvalidInput", err)
	}
}

func upperWWN(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'f' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func TestAssignIP_SubnetLookupDuplicatesAndRelease(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	vrf := "t" + testutil.Suffix()
	srvA, srvB := newServer(t), newServer(t)

	wide, _, err := CreateSubnet(ctx, SubnetInput{VRF: vrf, CIDR: "10.77.0.0/16", Meta: testMeta})
	if err != nil {
		t.Fatalf("CreateSubnet /16: %v", err)
	}
	narrow, _, err := CreateSubnet(ctx, SubnetInput{VRF: vrf, CIDR: "10.77.1.0/24", Meta: testMeta})
	if err != nil {
		t.Fatalf("CreateSubnet /24: %v", err)
	}
	ifaceA, _, err := AddInterface(ctx, InterfaceInput{CIID: srvA, Name: "eth0", Meta: testMeta})
	if err != nil {
		t.Fatalf("AddInterface: %v", err)
	}

	// Подсеть выбирается самая узкая из содержащих адрес.
	res, err := AssignIP(ctx, AssignIPInput{CIID: srvA, InterfaceID: &ifaceA, Address: "10.77.1.5", VRF: vrf, DNSName: "a.test", Meta: testMeta})
	if err != nil {
		t.Fatalf("AssignIP: %v", err)
	}
	if res.SubnetID != narrow {
		t.Errorf("SubnetID = %d, want узкая подсеть %d (широкая %d)", res.SubnetID, narrow, wide)
	}
	if res, err := AssignIP(ctx, AssignIPInput{CIID: srvA, Address: "10.77.9.9", VRF: vrf, Meta: testMeta}); err != nil || res.SubnetID != wide {
		t.Errorf("адрес вне /24: SubnetID = %d, err = %v; want широкая подсеть %d", res.SubnetID, err, wide)
	}
	if res, err := AssignIP(ctx, AssignIPInput{CIID: srvA, Address: "192.168.200.1", VRF: vrf, Meta: testMeta}); err != nil || res.SubnetID != 0 {
		t.Errorf("адрес без подсети: SubnetID = %d, err = %v; want 0", res.SubnetID, err)
	}

	// Тот же адрес с другой маской — всё равно дубликат в пределах VRF.
	if _, err := AssignIP(ctx, AssignIPInput{CIID: srvB, Address: "10.77.1.5/24", VRF: vrf, Meta: testMeta}); !errors.Is(err, svcerr.ErrDuplicate) {
		t.Errorf("повторный адрес: err = %v, want ErrDuplicate", err)
	}
	// Интерфейс чужой КЕ.
	if _, err := AssignIP(ctx, AssignIPInput{CIID: srvB, InterfaceID: &ifaceA, Address: "10.77.1.6", VRF: vrf, Meta: testMeta}); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("интерфейс другой КЕ: err = %v, want ErrInvalidState", err)
	}
	if _, err := AssignIP(ctx, AssignIPInput{CIID: srvA, Address: "10.77.1.999", VRF: vrf, Meta: testMeta}); !errors.Is(err, svcerr.ErrInvalidInput) {
		t.Errorf("некорректный адрес: err = %v, want ErrInvalidInput", err)
	}

	// Освобождение: адрес уходит из текущей таблицы, но AS OF до освобождения его помнит.
	beforeRelease := clock(t, db)
	time.Sleep(50 * time.Millisecond)
	if _, err := ReleaseIP(ctx, res.ID, 99, testMeta); !errors.Is(err, svcerr.ErrVersionConflict) {
		t.Errorf("ReleaseIP с неверной версией: err = %v, want ErrVersionConflict", err)
	}
	csID, err := ReleaseIP(ctx, res.ID, 1, testMeta)
	if err != nil {
		t.Fatalf("ReleaseIP: %v", err)
	}

	var current, past, deletedInHist, deletedInAudit int
	q := func(dst *int, query string, args ...any) {
		t.Helper()
		if err := db.QueryRow(query, args...).Scan(dst); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	q(&current, `SELECT count(*) FROM cmdb.ip_address WHERE id = $1`, res.ID)
	q(&past, `SELECT count(*) FROM cmdb.ip_address_asof($1) WHERE id = $2 AND ci_id = $3`, beforeRelease, res.ID, srvA)
	q(&deletedInHist, `SELECT count(*) FROM hist.ip_address WHERE id = $1 AND hist_op = 'D'`, res.ID)
	q(&deletedInAudit, `SELECT count(*) FROM audit.change_log WHERE change_set_id = $1 AND entity = 'ip_address' AND op = 'D'`, csID)
	if current != 0 || past != 1 || deletedInHist != 1 || deletedInAudit != 1 {
		t.Errorf("после освобождения: в таблице=%d (want 0), AS OF до=%d (want 1), hist D=%d (want 1), audit D=%d (want 1)",
			current, past, deletedInHist, deletedInAudit)
	}

	// Освобождённый адрес можно назначить другой КЕ.
	if _, err := AssignIP(ctx, AssignIPInput{CIID: srvB, Address: "10.77.1.5", VRF: vrf, Meta: testMeta}); err != nil {
		t.Errorf("повторное назначение освобождённого адреса: %v", err)
	}
}

func TestInterfaceAsOf_ReturnsPastState(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	srv := newServer(t)

	id, _, err := AddInterface(ctx, InterfaceInput{CIID: srv, Name: "eth0", Description: "было", Meta: testMeta})
	if err != nil {
		t.Fatalf("AddInterface: %v", err)
	}
	before := clock(t, db)
	time.Sleep(50 * time.Millisecond)

	_, err = changectx.Run(ctx, changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "правка интерфейса"},
		func(ctx context.Context, tx orm.TxOrmer) error {
			_, e := tx.RawWithCtx(ctx, `UPDATE cmdb.interface SET description = 'стало', updated_at = now() WHERE id = ?`, id).Exec()
			return e
		})
	if err != nil {
		t.Fatalf("правка интерфейса: %v", err)
	}

	var old, cur string
	if err := db.QueryRow(`SELECT description FROM cmdb.interface_asof($1) WHERE id = $2`, before, id).Scan(&old); err != nil {
		t.Fatalf("interface_asof(до): %v", err)
	}
	if err := db.QueryRow(`SELECT description FROM cmdb.interface_asof($1) WHERE id = $2`, clock(t, db), id).Scan(&cur); err != nil {
		t.Fatalf("interface_asof(сейчас): %v", err)
	}
	if old != "было" || cur != "стало" {
		t.Errorf("AS OF: до = %q (want было), сейчас = %q (want стало)", old, cur)
	}
}
