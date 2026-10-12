package ontap

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/connector"
	"cmdb/internal/importer/connectors/ontap/ontapmock"
	"cmdb/internal/importer/connectors/storage"
)

func newClient(t *testing.T, m *ontapmock.Server, password string) connector.Connector {
	t.Helper()
	srv := httptest.NewServer(m)
	t.Cleanup(srv.Close)
	t.Setenv("ONTAP_TEST_PASSWORD", password)
	c, err := New(connector.Config{Code: "ontap-test", Kind: Kind, Settings: map[string]any{
		"base_url": srv.URL, "username": m.User, "password_env": "ONTAP_TEST_PASSWORD"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func byExt(items []canon.Item) map[string]canon.Record {
	m := map[string]canon.Record{}
	for _, it := range items {
		m[it.Record.ExternalID] = it.Record
	}
	return m
}

func TestExtract_MapsClusterVolumesAndShares(t *testing.T) {
	m := ontapmock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	// массив + 2 агрегата + SVM + 2 тома + LUN + qtree + 2 ресурса (NFS-экспорт и CIFS; c$ пропущен)
	if len(items) != 10 {
		t.Fatalf("записей = %d, want 10", len(items))
	}
	recs := byExt(items)

	a := recs["array:ontap-test"]
	if a.Class != "storage_array" || a.SourceName != m.Name || a.Attrs["controllers"] != 2 || a.Attrs["raw_tb"] != 9.5 ||
		a.Attrs["microcode"] != m.Version || a.Attrs["array_family"] != "NetApp AFF-A400" || a.Attrs["mgmt_ip"] != "10.20.0.1" {
		t.Errorf("массив = %+v", a)
	}
	if len(a.Components) != 6 { // 2 узла + полка + 3 диска
		t.Errorf("компонентов массива %d, want 6", len(a.Components))
	}

	vol := recs["vol:"+m.Volumes[0].UUID]
	if vol.Class != "volume" || vol.SourceName != m.Name+"/svm1_0123456789:vol_data_0123456789" || vol.Attrs["thin"] != true || vol.Attrs["protocol"] != "nas" || vol.Attrs["size_gb"] != 500.0 {
		t.Errorf("FlexVol = %+v", vol)
	}
	types := map[string]int{}
	for _, r := range vol.Relations {
		types[r.Type]++
	}
	if types["runs_on"] != 1 || types["contains"] != 2 { // агрегат; qtree и NFS-экспорт
		t.Errorf("связи тома = %+v, want runs_on агрегата и contains qtree/экспорта", vol.Relations)
	}

	lun := recs["vol:lun:"+m.LUNs[0].UUID]
	if !strings.HasPrefix(lun.Identifiers["wwn"], "600a0980") || lun.Attrs["protocol"] != "block" {
		t.Errorf("LUN = %+v", lun)
	}
	if _, ok := recs["share:cifs:"+m.SVMs[0].UUID+"/c$"]; ok {
		t.Error("административный ресурс c$ не должен импортироваться")
	}
	if recs["share:nfs:"+m.Volumes[0].UUID].Attrs["share_type"] != "nfs" {
		t.Error("NFS-экспорт тома с nas.path")
	}
	svm := recs["svm:"+m.SVMs[0].UUID]
	if svm.Attrs["mgmt_ip"] != "10.20.0.5" {
		t.Errorf("SVM = %+v", svm)
	}
}

func TestExtract_AuthAndFailure(t *testing.T) {
	m := ontapmock.Default("0123456789")
	if err := newClient(t, m, m.Password).Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if err := newClient(t, m, "wrong").Test(context.Background()); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("Test с неверным паролем: %v", err)
	}
	m.Do(func(s *ontapmock.Server) { s.Fail = true })
	if items, err := newClient(t, m, m.Password).Extract(context.Background()); err == nil || items != nil {
		t.Fatalf("Extract при сбое: items=%d err=%v, want ошибка без частичного снимка", len(items), err)
	}
}

func TestNaaFromSerial(t *testing.T) {
	if got := naaFromSerial("ab"); got != "600a09806162" {
		t.Errorf("naaFromSerial = %q", got)
	}
	if naaFromSerial("") != "" {
		t.Error("пустой серийный номер -> пустой WWN")
	}
}

func wwpns(r canon.Record) []string {
	w, _ := r.Attrs["mapped_wwpns"].([]string)
	return w
}

// E5.5: WWPN инициаторов igroup, которым презентован LUN (iSCSI IQN отбрасывается).
func TestExtract_LUNMappingToInitiators(t *testing.T) {
	m := ontapmock.Default("0123456789")
	items, err := newClient(t, m, m.Password).Extract(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	recs := byExt(items)
	got := wwpns(recs["vol:lun:"+m.LUNs[0].UUID])
	if len(got) != 2 || got[0] != storage.SampleWWPN("0123456789", 1) || got[1] != storage.SampleWWPN("0123456789", 2) {
		t.Errorf("mapped_wwpns LUN = %v, want 2 FC-инициатора igroup", got)
	}
	if w, ok := recs["vol:"+m.Volumes[0].UUID].Attrs["mapped_wwpns"].([]string); !ok || len(w) != 0 {
		t.Errorf("том без презентации: %#v, want пустой список", recs["vol:"+m.Volumes[0].UUID].Attrs["mapped_wwpns"])
	}
}
