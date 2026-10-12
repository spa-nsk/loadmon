// E5.5: сквозная цепочка «ВМ → хост → сервер → датастор → том → массив», собранная из ТРЁХ источников
// (OneView — сервер, «виртуализация» — хост/ВМ/датастор, Unity — том и его презентация); связи между
// источниками выводит сам конвейер по серийному номеру, NAA и WWPN.
package pipeline_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/oneview/oneviewmock"
	"cmdb/internal/importer/connectors/storage"
	"cmdb/internal/importer/connectors/unity/unitymock"
	"cmdb/internal/testutil"
)

// chainVirt — тестовый коннектор виртуализации: хост (серийный номер, FC-порт), ВМ на нём и датастор
// с NAA. Настоящий vcsim не умеет отдавать NAA extent'ов, а связка проверяется на уровне конвейера.
type chainVirt struct{ tag, serial, wwpn, naa, bios string }

var registerChainVirt sync.Once

type chainVirtConn struct{ chainVirt }

func (c chainVirtConn) Test(ctx context.Context) error { return nil }

func (c chainVirtConn) Extract(ctx context.Context) ([]canon.Item, error) {
	t := c.tag
	recs := []canon.Record{
		{Class: "hypervisor_host", SourceName: "esx-" + t, ExternalID: "host:1", SerialNumber: c.serial, Identifiers: serialIdent(c.serial),
			Attrs:      map[string]any{"hv_type": "esxi", "platform": "vSphere", "bios_uuid": nilIfEmpty(c.bios)},
			Interfaces: []canon.Interface{{Name: "vmhba2", IfType: "fc", WWPN: c.wwpn, Enabled: true}}},
		{Class: "datastore", SourceName: "ds-" + t, ExternalID: "ds:1", Attrs: map[string]any{"ds_type": "vmfs", "naa": []string{c.naa}}},
		{Class: "vm", SourceName: "vm-" + t, ExternalID: "vm:1", Attrs: map[string]any{"moref": "vm-1"},
			Relations: []canon.Relation{{Type: "runs_on", TargetExternalID: "host:1"}, {Type: "uses_storage", TargetExternalID: "ds:1"}}},
	}
	return storage.Items(recs), nil
}

func ensureChainKind() {
	registerChainVirt.Do(func() {
		connector.Register("chainvirt", func(cfg connector.Config) (connector.Connector, error) {
			return chainVirtConn{chainVirt{tag: cfg.String("tag"), serial: cfg.String("serial"), wwpn: cfg.String("wwpn"), naa: cfg.String("naa"), bios: cfg.String("bios")}}, nil
		})
	})
}

func serialIdent(serial string) map[string]string {
	if serial == "" {
		return map[string]string{}
	}
	return map[string]string{"serial_number": serial}
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func newChainConnector(t *testing.T, db *sql.DB, tag, serial, wwpn, naa string) string {
	t.Helper()
	return newChainConnectorBIOS(t, db, tag, serial, wwpn, naa, "")
}

func newChainConnectorBIOS(t *testing.T, db *sql.DB, tag, serial, wwpn, naa, bios string) string {
	t.Helper()
	ensureChainKind()
	code := "chain-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"tag": tag, "serial": serial, "wwpn": wwpn, "naa": naa, "bios": bios})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind, trust_level) VALUES ('chainvirt', 'api', 60) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config) VALUES ($1, 'chainvirt', 'Тест: виртуализация', 'chainvirt', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatal(err)
	}
	return code
}

// containerOf — КЕ, которая `contains` данную (источник любого происхождения).
func containerOf(t *testing.T, db *sql.DB, dst int64) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT src_ci_id FROM cmdb.ci_rel WHERE rel_type = 'contains' AND dst_ci_id = $1 AND upper_inf(valid_period)`, dst).Scan(&id); err != nil {
		t.Fatalf("контейнер КЕ %d: %v", dst, err)
	}
	return id
}

func one(t *testing.T, ids []int64, what string) int64 {
	t.Helper()
	if len(ids) != 1 {
		t.Fatalf("%s: связей %v, want ровно одна", what, ids)
	}
	return ids[0]
}

func TestChain_VMHostServerDatastoreVolumeArray(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)

	ov := oneviewmock.Default(tag)
	ovCode := newOneViewConnector(t, db, ov)

	un := unitymock.Default(tag)
	unSrv := httptest.NewServer(un)
	t.Cleanup(unSrv.Close)
	unCode := newStorageConnector(t, db, "unity", "unity", unSrv.URL, "CMDB_UNITY_TEST_PW", un.User, un.Password)

	lunWWN := storage.NormalizeWWN(un.Data["lun"][0]["wwn"].(string)) // sv_1
	serverSerial := "SN-SRV3-" + tag                                  // стоечный сервер OneView
	virtCode := newChainConnector(t, db, tag, serverSerial, storage.SampleWWPN(tag, 3), lunWWN)

	for _, code := range []string{ovCode, unCode, virtCode} {
		if sum := runConn(t, code); sum.Errors != 0 || sum.Conflicts != 0 {
			t.Fatalf("прогон %s = %+v", code, sum)
		}
	}

	vm, host := ciID(t, db, "vm-"+tag), ciID(t, db, "esx-"+tag)
	server, ds := ciID(t, db, "rack-dl380-"+tag), ciID(t, db, "ds-"+tag)
	volume, array := ciID(t, db, un.Data["system"][0]["name"].(string)+"/lun-a-"+tag), ciID(t, db, un.Data["system"][0]["name"].(string))

	// ВМ → хост → сервер
	if got := one(t, openRels(t, db, vm, "runs_on"), "ВМ runs_on хост"); got != host {
		t.Fatalf("ВМ работает на %d, want хост %d", got, host)
	}
	if got := one(t, openRels(t, db, host, "runs_on"), "хост runs_on сервер"); got != server {
		t.Fatalf("хост работает на %d, want сервер OneView %d (по серийному номеру)", got, server)
	}
	// ВМ → датастор → том → пул → массив
	if got := one(t, openRels(t, db, vm, "uses_storage"), "ВМ uses_storage датастор"); got != ds {
		t.Fatalf("ВМ использует %d, want датастор %d", got, ds)
	}
	if got := one(t, openRels(t, db, ds, "depends_on"), "датастор depends_on том"); got != volume {
		t.Fatalf("датастор зависит от %d, want том %d (по NAA = WWN)", got, volume)
	}
	var backing sql.NullInt64
	db.QueryRow(`SELECT backing_volume_ci_id FROM cmdb.ci_datastore WHERE ci_id = $1`, ds).Scan(&backing)
	if !backing.Valid || backing.Int64 != volume {
		t.Errorf("ci_datastore.backing_volume_ci_id = %v, want %d", backing, volume)
	}
	pool := containerOf(t, db, volume)
	if containerOf(t, db, pool) != array {
		t.Fatalf("пул %d не принадлежит массиву %d — цепочка до массива разорвана", pool, array)
	}
	// том → хост (презентация по WWPN)
	if got := one(t, openRels(t, db, volume, "mapped_to"), "том mapped_to хост"); got != host {
		t.Fatalf("том презентован %d, want хост %d (по WWPN инициатора)", got, host)
	}

	// повторные прогоны — связей не появляется и не исчезает
	for _, code := range []string{ovCode, unCode, virtCode} {
		if again := runConn(t, code); again.RelCreated != 0 || again.RelClosed != 0 || again.Updated != 0 {
			t.Fatalf("повторный прогон %s = %+v, want без изменений (в т.ч. выведенных связей)", code, again)
		}
	}

	// презентацию LUN хосту сняли на массиве: связь том → хост закрывается
	un.Do(func(s *unitymock.Server) { s.Data["hostLUN"] = nil })
	if sum := runConn(t, unCode); sum.RelClosed != 1 {
		t.Fatalf("прогон после снятия презентации = %+v, want закрыта 1 выведенная связь", sum)
	}
	if got := openRels(t, db, volume, "mapped_to"); len(got) != 0 {
		t.Fatalf("mapped_to = %v, want закрыта", got)
	}
	// сервер переустановлен под другим серийным номером в источнике хоста: связь хост → сервер закрывается
	if _, err := db.Exec(`UPDATE imp.connector SET config = jsonb_set(config, '{serial}', to_jsonb($2::text)) WHERE connector_code = $1`, virtCode, "SN-OTHER-"+tag); err != nil {
		t.Fatal(err)
	}
	runConn(t, virtCode)
	if got := openRels(t, db, host, "runs_on"); len(got) != 0 {
		t.Fatalf("runs_on хоста = %v, want связь с сервером закрыта после смены серийного номера", got)
	}
}

// swapUUIDBytes меняет порядок байт первых трёх групп UUID (SMBIOS до 2.6 против 2.6+).
func swapUUIDBytes(u string) string {
	rev := func(h string) string {
		out := ""
		for i := len(h); i > 0; i -= 2 {
			out += h[i-2 : i]
		}
		return out
	}
	return rev(u[0:8]) + "-" + rev(u[9:13]) + "-" + rev(u[14:18]) + u[18:]
}

// Хост без серийного номера связывается с сервером по SMBIOS UUID — и в записи как есть, и «зеркально»
// (vSphere и Redfish/OneView могут показывать один UUID с разным порядком байт первых групп).
func TestChain_HostToServerByBIOSUUID(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	ov := oneviewmock.Default(tag)
	ovCode := newOneViewConnector(t, db, ov)
	runConn(t, ovCode)
	server := ciID(t, db, "rack-dl380-"+tag)
	var uuid string
	if err := db.QueryRow(`SELECT bios_uuid::text FROM cmdb.ci_server WHERE ci_id = $1`, server).Scan(&uuid); err != nil {
		t.Fatal(err)
	}

	for name, bios := range map[string]string{"direct": uuid, "swapped": swapUUIDBytes(uuid)} {
		hostTag := randHex(10)
		code := newChainConnectorBIOS(t, db, hostTag, "", storage.SampleWWPN(hostTag, 1), "", bios)
		runConn(t, code)
		host := ciID(t, db, "esx-"+hostTag)
		if got := openRels(t, db, host, "runs_on"); len(got) != 1 || got[0] != server {
			t.Errorf("%s: runs_on хоста = %v, want [%d] (по SMBIOS UUID без серийного номера)", name, got, server)
		}
		// освободить сервер для следующего варианта: хост выводится из эксплуатации нельзя — просто закроем связь сменой UUID
		if _, err := db.Exec(`UPDATE imp.connector SET config = jsonb_set(config, '{bios}', '"00000000-0000-4000-8000-000000000000"') WHERE connector_code = $1`, code); err != nil {
			t.Fatal(err)
		}
		runConn(t, code)
		if got := openRels(t, db, host, "runs_on"); len(got) != 0 {
			t.Errorf("%s: связь должна закрыться, когда UUID перестал совпадать: %v", name, got)
		}
	}
}
