package oneview

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexPowerW_stringWithComma(t *testing.T) {
	var f FlexPowerW
	if err := json.Unmarshal([]byte(`"698,"`), &f); err != nil {
		t.Fatal(err)
	}
	if f.Float64() != 698 {
		t.Fatalf("got %v", f.Float64())
	}
}

func TestEnclosurePowerWatts(t *testing.T) {
	raw := `{
		"powerAvg24Hrs": "1200,",
		"powerPeak24Hrs": 2400,
		"powerCapacity24Hrs": "4854,",
		"powerLastUpdate": "` + time.Now().UTC().Add(-time.Hour).Format("2006-01-02T15:04:05.000Z") + `"
	}`
	var enc Enclosure
	if err := json.Unmarshal([]byte(raw), &enc); err != nil {
		t.Fatal(err)
	}
	p := EnclosurePowerWatts(&enc)
	if p.Avg24hW != 1200 || p.Peak24hW != 2400 || p.RatedMaxW != 4854 {
		t.Fatalf("%+v", p)
	}
	if p.Source != "enclosure" {
		t.Fatalf("source=%q", p.Source)
	}
}

func TestParseEnvironmentalPower_nested(t *testing.T) {
	raw := json.RawMessage(`{
		"powerCapWatts": 500,
		"powerMetrics": {
			"averagePowerConsumed": 234,
			"maximumPowerConsumed": 310
		}
	}`)
	p := ParseEnvironmentalPower(raw)
	if p.Avg24hW != 234 || p.Peak24hW != 310 || p.RatedMaxW != 500 {
		t.Fatalf("%+v", p)
	}
}

func TestDeriveServerPower_offZerosAvgPeak(t *testing.T) {
	exp := &ServerExport{
		Identity:    ServerIdentity{PowerState: "Off"},
		Environment: json.RawMessage(`{"powerMetrics":{"averagePowerConsumed":100,"maximumPowerConsumed":200},"powerCapWatts":500}`),
	}
	p := DeriveServerPower(exp)
	if p.Avg24hW != 0 || p.Peak24hW != 0 || p.RatedMaxW != 500 {
		t.Fatalf("%+v", p)
	}
}
