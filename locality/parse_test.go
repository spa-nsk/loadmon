package locality

import "testing"

func TestParseOktmo(t *testing.T) {
	list, err := Parse("../itilliumimport/parsing/oktmo_data_20261001.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 100000 {
		t.Fatalf("мало записей: %d", len(list))
	}
	seen := map[string]bool{}
	for _, l := range list {
		if len(l.Oktmo) != 11 || seen[l.Oktmo] {
			t.Fatalf("плохой/дублирующийся код %q", l.Oktmo)
		}
		seen[l.Oktmo] = true
	}
	for _, l := range list {
		if l.Name == "Новосибирск" && l.TypeShort == "г" {
			t.Logf("%+v", l)
			return
		}
	}
	t.Fatal("г Новосибирск не найден")
}
