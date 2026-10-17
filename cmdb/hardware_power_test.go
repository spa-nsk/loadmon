package cmdb

import (
	"testing"

	models "loadmon/models"

	oneview "github.com/spa-nsk/oneview2"
)

func cand(endpoint string, gd bool, name, sn string, p oneview.PowerWatts) encCandidate {
	enc := oneview.Enclosure{SerialNumber: sn}
	enc.Name = name
	return encCandidate{Endpoint: endpoint, GD: gd, Enc: enc, Power: p}
}

// c7000: локальный /utilization приоритетнее устаревших (уже отброшенных) и
// даже свежих полей GD, независимо от порядка опроса endpoint; описание - с локального.
func TestPickEnclosureSources_localMeasurementWins(t *testing.T) {
	local := cand("local", false, "C7000-12", "CZ2731064V", oneview.PowerWatts{
		Avg24hW: 2722, Peak24hW: 2908, RatedMaxW: 5212, RatedSource: oneview.RatedSourceCalibrated, Source: oneview.PowerSourceUtilization})
	gdFresh := cand("gd", true, "C7000-12", "CZ2731064V", oneview.PowerWatts{
		Avg24hW: 542, Peak24hW: 544, RatedMaxW: 1130, RatedSource: oneview.RatedSourceCapacity, Source: oneview.PowerSourceObject})

	for _, order := range [][]encCandidate{{local, gdFresh}, {gdFresh, local}} {
		inv, p := pickEnclosureSources(order)
		if inv.GD || inv.Endpoint != "local" {
			t.Fatalf("описание должно быть с локального appliance: %+v", inv)
		}
		if p.Source != oneview.PowerSourceUtilization || p.Avg24hW != 2722 || p.RatedMaxW != 5212 {
			t.Fatalf("%+v", p)
		}
	}
}

// Устаревшие поля GD (пустой PowerWatts) + локальное измерение отсутствует:
// результат пустой -> в БД будет оценка по блейдам, а не старые числа.
func TestPickEnclosureSources_staleGDGivesEmpty(t *testing.T) {
	_, p := pickEnclosureSources([]encCandidate{cand("gd", true, "C7000-7", "CZ30495498", oneview.PowerWatts{})})
	if p.HasMeasured() || p.Source != "" || p.RatedMaxW != 0 {
		t.Fatalf("%+v", p)
	}
}

// Synergy-фрейм виден только в GD: свежие поля объекта остаются единственным источником.
func TestPickEnclosureSources_synergyOnlyGD(t *testing.T) {
	gd := cand("gd", true, "ENC-18", "CZJ12305XV", oneview.PowerWatts{
		Avg24hW: 3552, Peak24hW: 4086, RatedMaxW: 6877, RatedSource: oneview.RatedSourceCapacity, Source: oneview.PowerSourceObject})
	inv, p := pickEnclosureSources([]encCandidate{gd})
	if !inv.GD || p.Source != oneview.PowerSourceObject || p.Avg24hW != 3552 || p.RatedMaxW != 6877 {
		t.Fatalf("%+v %+v", inv, p)
	}
}

func TestEncKey_serialNumberWinsOverName(t *testing.T) {
	a := oneview.Enclosure{SerialNumber: " cz2731064v "}
	a.Name = "C7000-12"
	b := oneview.Enclosure{SerialNumber: "CZ2731064V"}
	b.Name = "c7000-12-renamed"
	if encKey(&a) != encKey(&b) {
		t.Fatalf("%q != %q", encKey(&a), encKey(&b))
	}
	c := oneview.Enclosure{}
	c.Name = "c7000-1"
	d := oneview.Enclosure{}
	d.Name = "C7000-01"
	if encKey(&c) != encKey(&d) {
		t.Fatalf("без серийника ключ по нормализованному имени: %q != %q", encKey(&c), encKey(&d))
	}
}

// Измерение записывается как есть.
func TestApplyBladeSystemPower_measured(t *testing.T) {
	bls := models.BladeSystems{PowerAvg24hW: 4577, PowerPeak24hW: 4756, PowerRatedMaxW: 1130}
	ApplyBladeSystemPowerVariantC(nil, &bls, oneview.PowerWatts{
		Avg24hW: 2722, Peak24hW: 2908, RatedMaxW: 5212, Source: oneview.PowerSourceUtilization})
	if bls.PowerAvg24hW != 2722 || bls.PowerPeak24hW != 2908 || bls.PowerRatedMaxW != 5212 {
		t.Fatalf("%+v", bls)
	}
}

// Нет ни измерения, ни оценки (нет блейдов): устаревшие значения очищаются,
// номинал из calibratedMaxPower/powerCapacity24Hrs сохраняется.
func TestApplyBladeSystemPower_noMeasurementClearsStaleKeepsRated(t *testing.T) {
	bls := models.BladeSystems{PowerAvg24hW: 4577, PowerPeak24hW: 4756, PowerRatedMaxW: 7134}
	ApplyBladeSystemPowerVariantC(nil, &bls, oneview.PowerWatts{RatedMaxW: 5212, RatedSource: oneview.RatedSourceCalibrated})
	if bls.PowerAvg24hW != 0 || bls.PowerPeak24hW != 0 {
		t.Fatalf("устаревшие avg/peak должны быть очищены: %+v", bls)
	}
	if bls.PowerRatedMaxW != 5212 {
		t.Fatalf("номинал: %+v", bls)
	}
}
