package itilliumimport

import "testing"

func TestParseStoragePlacementSegmentsMultiCabinet(t *testing.T) {
	segs := parseStoragePlacementSegments("6.2-B-06 | 1-6, 6.2-B-07 | 1-42, 6.2-B-08 | 1-40")
	if len(segs) != 3 || segs[1].CabinetCode != "6.2-B-07" || segs[1].EndUnitPos != 42 || segs[2].CabinetCode != "6.2-B-08" {
		t.Fatalf("неверный разбор: %+v", segs)
	}
	segs = parseStoragePlacementSegments("5.1-A-04 | 9-22, 5.1-A-04 | 24-40")
	if len(segs) != 1 || segs[0].StartUnitPos != 9 || segs[0].EndUnitPos != 40 {
		t.Fatalf("неверный разбор одного шкафа: %+v", segs)
	}
	segs = parseStoragePlacementSegments("ЦОД Мегафон / 3.13.0 | 22-24, ЦОД Мегафон / 3.13.0 | 25-26")
	if len(segs) != 1 || segs[0].AltName != "ЦОД Мегафон" || segs[0].EndUnitPos != 26 {
		t.Fatalf("неверный разбор colocation: %+v", segs)
	}
}

func TestParseStoragePlacementSegmentsMixedDatacenters(t *testing.T) {
	segs := parseStoragePlacementSegments("3.02-B-01 | 10-12, ЦОД М1 / R15-5 | 1-2, ЦОД Мегафон / 3.13.0 | 25-26, 3.02-B-02 | 4-6")
	want := []struct {
		dc, cab string
		s, e    int
	}{{"", "3.02-B-01", 10, 12}, {"ЦОД М1", "R15-5", 1, 2}, {"ЦОД Мегафон", "3.13.0", 25, 26}, {"", "3.02-B-02", 4, 6}}
	if len(segs) != len(want) {
		t.Fatalf("сегментов %d, ожидалось %d: %+v", len(segs), len(want), segs)
	}
	for i, w := range want {
		g := segs[i]
		if g.AltName != w.dc || g.CabinetCode != w.cab || g.StartUnitPos != w.s || g.EndUnitPos != w.e {
			t.Errorf("сегмент %d: %+v, ожидалось %+v", i, g, w)
		}
	}
}
