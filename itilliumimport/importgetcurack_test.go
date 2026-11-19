package itilliumimport

import (
	"testing"

	"loadmon/itilium"
	"loadmon/models"
)

func TestLocationUndefinedForCU(t *testing.T) {
	if !locationUndefinedForCU("x", models.ItiliumStatusWrittenOff) {
		t.Fatal("списан")
	}
	if !locationUndefinedForCU("x", models.ItiliumStatusInTransit) {
		t.Fatal("в пути")
	}
	if !locationUndefinedForCU("! Покинуло ФГ что-то", models.ItiliumStatusInService) {
		t.Fatal("покинуло ФГ")
	}
	if locationUndefinedForCU("Новосибирск | 1 | зал", models.ItiliumStatusInService) {
		t.Fatal("обычный where")
	}
}

func TestCUModelFallback(t *testing.T) {
	row := itilium.CURow{
		ModelServ:  itilium.LabelValue{Value: "Norm"},
		ModelVendS: itilium.LabelValue{Value: "Vend"},
	}
	if cuModel(row) != "Vend" {
		t.Fatal(cuModel(row))
	}
	row.ModelVendS = itilium.LabelValue{}
	if cuModel(row) != "Norm" {
		t.Fatal(cuModel(row))
	}
	row.ModelServ = itilium.LabelValue{}
	row.Description = itilium.LabelValue{Value: "HP ProLiant DL380 Gen10"}
	if cuModel(row) != "HP ProLiant DL380 Gen10" {
		t.Fatal(cuModel(row))
	}
	row.Description = itilium.LabelValue{Value: "dell poweredge r740"}
	if cuModel(row) != "dell poweredge r740" {
		t.Fatal(cuModel(row))
	}
	for _, desc := range []string{"Chieftec 4U", "Inflex box", "DEPO Storm 3450", "Cisco UCS C240"} {
		row.Description = itilium.LabelValue{Value: desc}
		if cuModel(row) != desc {
			t.Fatalf("desc=%q got=%q", desc, cuModel(row))
		}
	}
	row.Description = itilium.LabelValue{Value: "просто текст без вендора"}
	if cuModel(row) != "" {
		t.Fatal(cuModel(row))
	}
	// modelServ приоритетнее description
	row.ModelServ = itilium.LabelValue{Value: "Norm"}
	row.Description = itilium.LabelValue{Value: "HP anything"}
	if cuModel(row) != "Norm" {
		t.Fatal(cuModel(row))
	}
}

func TestMapCUStatusLabel(t *testing.T) {
	st, ok := MapCUStatusLabel("Черновик")
	if !ok || st.ItiliumStatus != 9 {
		t.Fatalf("%+v %v", st, ok)
	}
	st, ok = MapCUStatusLabel("В работе (эксплуатация)")
	if !ok || st.Active != 1 {
		t.Fatalf("%+v %v", st, ok)
	}
}

func TestCUMemoryAndDate(t *testing.T) {
	row := itilium.CURow{
		RAM1:  itilium.LabelValue{Value: "DDR4 32Gb"},
		CRAM1: itilium.LabelValue{Value: "4"},
		RAM2:  itilium.LabelValue{Value: "16 GB"},
		CRAM2: itilium.LabelValue{Value: "2"},
	}
	mb := cuMemoryMb(row)
	// 32*4 + 16*2 = 160 GB → 163840 MB
	if mb != 160*1024 {
		t.Fatalf("memory=%d", mb)
	}
	tm, ok := cuParseDateRU("23.10.2025 0:00:00")
	if !ok || tm.Day() != 23 || tm.Month() != 10 || tm.Year() != 2025 {
		t.Fatalf("%v %v", tm, ok)
	}
}

func TestCUStatusHistoryTime(t *testing.T) {
	row := itilium.CURow{
		StatusHistory: itilium.LabelValue{Value: "10.02.2023"},
	}
	tm, ok := cuStatusHistoryTime(row)
	if !ok || tm.Day() != 10 || tm.Month() != 2 || tm.Year() != 2023 {
		t.Fatalf("statusHistory: %v %v", tm, ok)
	}
	row = itilium.CURow{StatusHistoryYY: "2023-02-10"}
	tm, ok = cuStatusHistoryTime(row)
	if !ok || tm.Day() != 10 || tm.Month() != 2 || tm.Year() != 2023 {
		t.Fatalf("statusHistoryYY: %v %v", tm, ok)
	}
	row = itilium.CURow{StatusHistory: itilium.LabelValue{Value: "2024:в работе"}}
	if _, ok := cuStatusHistoryTime(row); ok {
		t.Fatal("ожидали отказ на «2024:в работе» без YY")
	}
}

func TestCUDetailsAndMpIP(t *testing.T) {
	row := itilium.CURow{
		Description: itilium.LabelValue{Value: "desc"},
		Comments:    itilium.LabelValue{Value: "note"},
		ILOIP:       itilium.LabelValue{Value: "10.0.0.1"},
		IPMI:        itilium.LabelValue{Value: "10.0.0.2"},
	}
	if cuDetails(row) != "desc; note" {
		t.Fatal(cuDetails(row))
	}
	if cuMpIP(row) != "10.0.0.1" {
		t.Fatal(cuMpIP(row))
	}
}
