package storage

import (
	"testing"

	"cmdb/internal/importer/canon"
)

func lun(n int) *int { return &n }

func byExt(recs []canon.Record) map[string]canon.Record {
	m := map[string]canon.Record{}
	for _, r := range recs {
		m[r.ExternalID] = r
	}
	return m
}

func hasRel(r canon.Record, typ, target string) bool {
	for _, rel := range r.Relations {
		if rel.Type == typ && rel.TargetExternalID == target {
			return true
		}
	}
	return false
}

func TestBuild_BlockArray(t *testing.T) {
	thin := true
	recs := Build(Snapshot{
		Array:       Array{Name: "ARR1", Serial: "SN1", Family: "Vendor X1", Microcode: "1.2"},
		Pools:       []Pool{{ID: "p1", Name: "pool1", RAID: "raid6", CapacityTB: 10}},
		Volumes:     []Volume{{ID: "v1", Name: "lun1", PoolID: "p1", LunID: lun(5), WWN: "600a09800000abcd", SizeGB: 100, Thin: &thin, Protocol: "fc"}},
		Controllers: []Component{{ID: "A", Name: "SPA", Serial: "C1"}},
		Enclosures:  []Component{{ID: "e0", Name: "DAE0"}},
		Disks: []Disk{
			{ID: "d1", Serial: "D1", EnclosureID: "e0", Type: "ssd", CapacityGB: 1000},
			{ID: "d2", Serial: "D2", EnclosureID: "missing", CapacityGB: 1000},
		},
	}, "arr1")
	m := byExt(recs)

	a := m["array:arr1"]
	if a.Class != "storage_array" || a.SerialNumber != "SN1" || a.Attrs["raw_tb"] != 2.0 || a.Attrs["controllers"] != 1 {
		t.Errorf("массив = %+v", a)
	}
	if !hasRel(a, "contains", "pool:p1") {
		t.Error("массив должен содержать пул")
	}
	if len(a.Components) != 4 {
		t.Fatalf("компонентов %d, want 4 (контроллер, полка, 2 диска)", len(a.Components))
	}
	for _, c := range a.Components {
		if c.Key == "disk:D1" && c.ParentKey != "enc:e0" {
			t.Errorf("диск в известной полке должен быть вложен: %+v", c)
		}
		if c.Key == "disk:D2" && c.ParentKey != "" {
			t.Errorf("диск с неизвестной полкой — корневой: %+v", c)
		}
	}
	p := m["pool:p1"]
	if p.SourceName != "ARR1/pool1" || !hasRel(p, "contains", "vol:v1") {
		t.Errorf("пул = %+v", p)
	}
	v := m["vol:v1"]
	if v.SourceName != "ARR1/lun1" || v.Identifiers["wwn"] != "600a09800000abcd" || v.Attrs["lun_id"] != 5 || v.Attrs["thin"] != true {
		t.Errorf("том = %+v", v)
	}
}

func TestBuild_NASArray(t *testing.T) {
	recs := Build(Snapshot{
		Array:   Array{Name: "C1"},
		Pools:   []Pool{{ID: "aggr1", Name: "aggr1"}},
		SVMs:    []SVM{{ID: "s1", Name: "svm1"}},
		Volumes: []Volume{{ID: "v1", Name: "vol1", SVMID: "s1", PoolID: "aggr1", Protocol: "nfs"}},
		Qtrees:  []Qtree{{ID: "q1", Name: "vol1/q1", VolumeID: "v1", Path: "/vol1/q1", QuotaGB: 10}},
		Shares:  []Share{{ID: "sh1", Name: "svm1/data", VolumeID: "v1", Type: "SMB", Path: "/vol1"}, {ID: "sh2", Name: "x", VolumeID: "v1", Type: "weird"}},
	}, "c1")
	m := byExt(recs)
	if !hasRel(m["svm:s1"], "contains", "vol:v1") || !hasRel(m["vol:v1"], "runs_on", "pool:aggr1") {
		t.Errorf("SVM содержит том, том размещён на агрегате: svm=%+v vol=%+v", m["svm:s1"].Relations, m["vol:v1"].Relations)
	}
	if !hasRel(m["vol:v1"], "contains", "qtree:q1") || !hasRel(m["vol:v1"], "contains", "share:sh1") {
		t.Errorf("том содержит qtree и ресурс: %+v", m["vol:v1"].Relations)
	}
	if m["share:sh1"].Attrs["share_type"] != "smb" || m["share:sh2"].Attrs["share_type"] != nil {
		t.Error("тип ресурса: SMB -> smb, неизвестный -> nil (CHECK допускает только smb/nfs)")
	}
	if m["array:c1"].Identifiers["serial_number"] != "" {
		t.Error("без серийного номера идентификатора serial_number быть не должно")
	}
}

func TestNormalizeWWN(t *testing.T) {
	for in, want := range map[string]string{"NAA.600A0980:0000-ABcd": "600a09800000abcd", " 60:06:01 ": "600601"} {
		if got := NormalizeWWN(in); got != want {
			t.Errorf("NormalizeWWN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeWWPN(t *testing.T) {
	cases := map[string]string{
		"2100000E1E1A2B3C":                                "21:00:00:0e:1e:1a:2b:3c",
		"21:00:00:0e:1e:1a:2b:3c":                         "21:00:00:0e:1e:1a:2b:3c",
		"0x21000e1e1a2b3c4d":                              "21:00:0e:1e:1a:2b:3c:4d",
		"20:00:00:00:c9:aa:bb:cc:10:00:00:00:c9:aa:bb:cc": "10:00:00:00:c9:aa:bb:cc", // Unity: WWNN:WWPN
		"iqn.1998-01.com.vmware:esx1":                     "",
		"xyz":                                             "",
		"":                                                "",
	}
	for in, want := range cases {
		if got := NormalizeWWPN(in); got != want {
			t.Errorf("NormalizeWWPN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuild_MappedWWPNs(t *testing.T) {
	recs := Build(Snapshot{
		Array:         Array{Name: "A"},
		Volumes:       []Volume{{ID: "v1", Name: "l1"}, {ID: "v2", Name: "l2"}},
		Mappings:      []Mapping{{VolumeID: "v1", WWPNs: []string{"2100000E1E1A2B3C", "21:00:00:0e:1e:1a:2b:3c", "iqn.x", "21:00:00:0e:1e:1a:2b:00"}}},
		MappingsKnown: true,
	}, "a")
	m := byExt(recs)
	got, _ := m["vol:v1"].Attrs["mapped_wwpns"].([]string)
	if len(got) != 2 || got[0] != "21:00:00:0e:1e:1a:2b:00" || got[1] != "21:00:00:0e:1e:1a:2b:3c" {
		t.Errorf("mapped_wwpns v1 = %v, want 2 уникальных WWPN по порядку (IQN отброшен)", got)
	}
	if empty, ok := m["vol:v2"].Attrs["mapped_wwpns"].([]string); !ok || len(empty) != 0 {
		t.Errorf("v2 без презентации при MappingsKnown должен иметь пустой список, got %#v", m["vol:v2"].Attrs["mapped_wwpns"])
	}
	unknown := Build(Snapshot{Array: Array{Name: "A"}, Volumes: []Volume{{ID: "v1", Name: "l1"}}}, "a")
	if _, has := byExt(unknown)["vol:v1"].Attrs["mapped_wwpns"]; has {
		t.Error("без MappingsKnown атрибут mapped_wwpns не должен передаваться")
	}
}
