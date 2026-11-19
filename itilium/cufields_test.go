package itilium

import (
	"testing"
)

func TestDefaultFieldCatalogServers(t *testing.T) {
	cat := DefaultFieldCatalog()
	if cat.ClassName(CUClassServers) != "Серверы" {
		t.Fatalf("name=%q", cat.ClassName(CUClassServers))
	}
	fs := cat.Fields(CUClassServers)
	if len(fs) < 40 {
		t.Fatalf("fields=%d", len(fs))
	}
	codes := map[string]bool{}
	for _, f := range fs {
		codes[f.Code] = true
	}
	for _, need := range []string{"hostname", "ip", "sn", "modelServ", "cpu", "ram_1", "iloip", "os", "role", "buhCode"} {
		if !codes[need] {
			t.Fatalf("нет поля %s в каталоге 145", need)
		}
	}
	req := DefaultFieldsForClass(CUClassServers)
	seen := map[string]bool{}
	for _, c := range req {
		seen[c] = true
	}
	for _, need := range []string{"code", "status", "linksUnit", "modelVendS", "units", "cpu", "os"} {
		if !seen[need] {
			t.Fatalf("DefaultFieldsForClass без %s: %v", need, req)
		}
	}
}

func TestLoadFieldCatalogJSONRoundtrip(t *testing.T) {
	cat, err := LoadFieldCatalogJSON(cuFieldsCatalogJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Classes()) < 30 {
		t.Fatalf("classes=%d", len(cat.Classes()))
	}
}
