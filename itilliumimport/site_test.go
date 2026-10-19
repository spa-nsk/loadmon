package itilliumimport

import (
	"testing"

	"loadmon/locality"
)

func TestCabinetFromPlacement(t *testing.T) {
	loc := locality.ParseLocation("Саяногорск, БЦ (на Советской) | 6 | 6.2 (623) | Шкаф 6.2-B-01 | Unit 32",
		func(s string) bool { return s == "Саяногорск" })
	if loc.Floor != "6" || loc.Room != "6.2 (623)" || loc.Street != "БЦ (на Советской)" {
		t.Fatalf("%+v", loc)
	}
	if got := buildSiteName(loc, "Саяногорск"); got != "Саяногорск, БЦ (на Советской), этаж 6, 6.2 (623)" {
		t.Fatalf("name %q", got)
	}
	// размещение задано - альтернативное наименование отделяется от чистого кода
	c, alt, pos := cabinetFromPlacement("ЦОД Домофон / 3.11.0", loc)
	if c != "3.11.0" || alt != "ЦОД Домофон" || pos != nil {
		t.Fatalf("%q %q %v", c, alt, pos)
	}
	// размещение пустое - шкаф и позиция из "Местоположения"
	c, alt, pos = cabinetFromPlacement("", loc)
	if c != "6.2-B-01" || alt != "" || pos == nil || pos.Start != 32 || pos.End != 32 {
		t.Fatalf("%q %q %v", c, alt, pos)
	}
}
