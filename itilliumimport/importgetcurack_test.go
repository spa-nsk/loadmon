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
