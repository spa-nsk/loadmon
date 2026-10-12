// E8.4: запись сети-устройства (класс net_switch, ci_network_device) через весь конвейер на
// PostgreSQL. SNMP — бинарный UDP-протокол с USM-аутентификацией/шифрованием: в отличие от
// остальных коннекторов E5/E8 его нельзя поднять как httptest-мок (проверено REST/SOAP), поэтому
// здесь не настоящий коннектор snmpnet (его сведение PDU -> canon покрыто unit-тестом с
// заглушкой fetcher, internal/importer/connectors/snmpnet/snmpnet_test.go), а тестовый Connector,
// отдающий ТУ ЖЕ каноническую форму (device_role/os_version/mgmt_ip, компонент, интерфейс), что и
// buildRecord у snmpnet — проверяется запись в ci_network_device/component/interface конвейером,
// которую до этого коннектора не проверял ни один тест.
package pipeline_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/testutil"
)

type snmpFakeConn struct{ tag string }

func (c snmpFakeConn) Test(ctx context.Context) error { return nil }

func (c snmpFakeConn) Extract(ctx context.Context) ([]canon.Item, error) {
	t := c.tag
	rec := canon.Record{
		Class: "net_switch", SourceName: "sw-" + t, ExternalID: "device:" + t, SerialNumber: "SN" + t,
		Identifiers: map[string]string{"serial_number": "SN" + t},
		Attrs:       map[string]any{"device_role": "access", "os_version": "Eltex MES3324 v3.4.2", "mgmt_ip": "10.20.0.1"},
		Vendor:      map[string]any{"model": "MES3324"},
		Components:  []canon.Component{{Key: "psu:PSU" + t, Type: "psu", Slot: "PSU 1", SerialNumber: "PSU" + t}},
		Interfaces:  []canon.Interface{{Name: "Gi0/1", IfType: "ethernet", MAC: "aa:bb:cc:dd:ee:ff", SpeedMbps: 1000, Enabled: true}},
	}
	return []canon.Item{{Seq: 1, Record: rec}}, nil
}

var registerSNMPFake sync.Once

func ensureSNMPFakeKind() {
	registerSNMPFake.Do(func() {
		connector.Register("snmpfaketest", func(cfg connector.Config) (connector.Connector, error) {
			return snmpFakeConn{tag: cfg.String("tag")}, nil
		})
	})
}

func newSNMPFakeConnector(t *testing.T, db *sql.DB, tag string) string {
	t.Helper()
	ensureSNMPFakeKind()
	code := "snmp-" + testutil.Suffix()
	cfg, _ := json.Marshal(map[string]any{"tag": tag})
	if _, err := db.Exec(`INSERT INTO ref.source_system (source_system_code, kind, trust_level) VALUES ('snmp', 'api', 60) ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("source_system: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO imp.connector (connector_code, kind, name, source_system_code, config)
		VALUES ($1, 'snmpfaketest', 'Тестовый SNMP-коммутатор', 'snmp', $2::jsonb)`, code, string(cfg)); err != nil {
		t.Fatalf("imp.connector: %v", err)
	}
	return code
}

func TestRunConnector_SNMPNetworkDeviceComponentsAndInterfaces(t *testing.T) {
	db := testutil.DB(t)
	tag := randHex(10)
	code := newSNMPFakeConnector(t, db, tag)

	sum := runConn(t, code)
	if sum.Created != 1 || sum.Errors != 0 || sum.Conflicts != 0 {
		t.Fatalf("первый прогон = %+v, want Created=1", sum)
	}

	dev := ciID(t, db, "sw-"+tag)
	var role, osVersion, mgmtIP string
	if err := db.QueryRow(`SELECT device_role, os_version, mgmt_ip::text FROM cmdb.ci_network_device WHERE ci_id = $1`, dev).Scan(&role, &osVersion, &mgmtIP); err != nil {
		t.Fatalf("ci_network_device: %v", err)
	}
	if role != "access" || osVersion != "Eltex MES3324 v3.4.2" || mgmtIP != "10.20.0.1/32" {
		t.Errorf("ci_network_device = role=%q os_version=%q mgmt_ip=%q", role, osVersion, mgmtIP)
	}

	if n := compCount(t, db, dev, "installed"); n != 1 {
		t.Errorf("компонентов = %d, want 1 (БП)", n)
	}
	var ifCount int
	db.QueryRow(`SELECT count(*) FROM cmdb.interface WHERE ci_id = $1`, dev).Scan(&ifCount)
	if ifCount != 1 {
		t.Errorf("интерфейсов = %d, want 1", ifCount)
	}

	if second := runConn(t, code); second.Created != 0 || second.Updated != 0 || second.Unchanged != 1 {
		t.Errorf("повторный прогон = %+v, want 1 Unchanged", second)
	}
}
