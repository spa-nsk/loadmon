package zabbix

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/zabbix/zabbixmock"
)

func newClient(t *testing.T, m *zabbixmock.Server, settings map[string]any) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("ZABBIX_TEST_TOKEN", m.Token)
	t.Setenv("ZABBIX_TEST_PASSWORD", m.Password)
	cfg := map[string]any{"base_url": srv.URL}
	for k, v := range settings {
		cfg[k] = v
	}
	c, err := New(connector.Config{Code: "zbx-test", Kind: Kind, Settings: cfg})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func TestExtract_SupplementRecords(t *testing.T) {
	m := zabbixmock.Default("0123456789")
	c := newClient(t, m, map[string]any{"token_env": "ZABBIX_TEST_TOKEN"})
	if connector.ModeOf(c) != connector.ModeSupplement {
		t.Fatal("Zabbix должен работать в режиме supplement")
	}
	items, err := c.Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("записей = %d, want 3", len(items))
	}
	a := items[0].Record
	if a.Class != "" || a.SourceName != "zbx-a-0123456789.corp.local" || a.Identifiers["short_name"] != "zbx-a-0123456789" ||
		a.Identifiers["alias"] != "Сервер A 0123456789" || a.SerialNumber != "SN-ZBX-A-0123456789" || a.ExternalID != "zbx:10001" {
		t.Errorf("хост A = %+v", a)
	}
	b := items[1].Record
	if _, has := b.Identifiers["alias"]; has {
		t.Error("видимое имя, совпадающее с host, не должно быть alias")
	}
	ips, _ := b.Vendor["ips"].([]string)
	if len(ips) != 2 || b.Vendor["monitored"] != true {
		t.Errorf("вендорские сведения B = %+v", b.Vendor)
	}
	if groups, _ := b.Vendor["groups"].([]string); len(groups) != 2 || groups[0] != "DB" {
		t.Errorf("группы B = %v, want отсортированные [DB Linux servers]", groups)
	}
	if items[2].Record.SerialNumber != "" {
		t.Error("хост без inventory не должен давать серийный номер")
	}
}

func TestAuth_TokenAndLogin(t *testing.T) {
	m := zabbixmock.Default("0123456789")
	if err := newClient(t, m, map[string]any{"token_env": "ZABBIX_TEST_TOKEN"}).Test(context.Background()); err != nil {
		t.Fatalf("Test с токеном: %v", err)
	}
	if err := newClient(t, m, map[string]any{"username": "Admin", "password_env": "ZABBIX_TEST_PASSWORD"}).Test(context.Background()); err != nil {
		t.Fatalf("Test с логином: %v", err)
	}
	m.Do(func(s *zabbixmock.Server) { s.Password = "changed" })
	if err := newClient(t, m, map[string]any{"username": "Admin", "password_env": "ZABBIX_TEST_TOKEN"}).Test(context.Background()); err == nil || !strings.Contains(err.Error(), "Incorrect user name") {
		t.Fatalf("Test с неверным паролем: %v, want ошибка JSON-RPC", err)
	}
}

func TestTest_RejectsUnsupportedVersion(t *testing.T) {
	m := zabbixmock.Default("0123456789")
	m.Do(func(s *zabbixmock.Server) { s.Version = "5.0.1" })
	if err := newClient(t, m, map[string]any{"token_env": "ZABBIX_TEST_TOKEN"}).Test(context.Background()); err == nil || !strings.Contains(err.Error(), "5.0.1") {
		t.Fatalf("Test для Zabbix 5.0: %v, want ошибка версии", err)
	}
}

func TestExtract_RPCErrorIsWholeSnapshotError(t *testing.T) {
	m := zabbixmock.Default("0123456789")
	m.Fail = true
	items, err := newClient(t, m, map[string]any{"token_env": "ZABBIX_TEST_TOKEN"}).Extract(context.Background())
	if err == nil || items != nil || !strings.Contains(err.Error(), "SQL statement") {
		t.Fatalf("Extract при ошибке JSON-RPC: items=%d err=%v", len(items), err)
	}
}

func TestNew_RequiresAuth(t *testing.T) {
	if _, err := New(connector.Config{Code: "x", Settings: map[string]any{"base_url": "http://h"}}); err == nil {
		t.Error("New без token_env и username должен вернуть ошибку")
	}
}
