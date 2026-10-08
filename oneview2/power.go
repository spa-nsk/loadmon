package oneview

import (
	"encoding/json"
	"math"
	"strings"
)

// PowerWatts — снимок потребления/лимита (ватты) для сервера или корзины.
type PowerWatts struct {
	Avg24hW   float64 `json:"avg24hW,omitempty"`
	Peak24hW  float64 `json:"peak24hW,omitempty"`
	RatedMaxW float64 `json:"ratedMaxW,omitempty"`
	// Source — откуда взяты значения: "environment", "enclosure", "estimate", "".
	Source string `json:"source,omitempty"`
}

func (p PowerWatts) HasMeasured() bool {
	return p.Avg24hW > 0 || p.Peak24hW > 0
}

// FlexPowerW unmarshals JSON numbers or numeric strings (в т.ч. "698,") в float64 ватт.
type FlexPowerW float64

func (f FlexPowerW) Float64() float64 { return float64(f) }

func (f *FlexPowerW) UnmarshalJSON(b []byte) error {
	b = bytesTrim(b)
	if len(b) == 0 || string(b) == "null" {
		*f = 0
		return nil
	}
	var v float64
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		s = strings.TrimSuffix(s, ",")
		if s == "" {
			*f = 0
			return nil
		}
		var err error
		v, err = parsePowerString(s)
		if err != nil {
			*f = 0
			return nil
		}
		*f = FlexPowerW(v)
		return nil
	}
	if err := json.Unmarshal(b, &v); err != nil {
		*f = 0
		return nil
	}
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		*f = 0
		return nil
	}
	*f = FlexPowerW(v)
	return nil
}

func parsePowerString(s string) (float64, error) {
	s = strings.ReplaceAll(s, ",", "")
	return json.Number(s).Float64()
}

// EnclosurePowerWatts читает powerAvg24Hrs / powerPeak24Hrs / powerCapacity24Hrs с корзины.
func EnclosurePowerWatts(enc *Enclosure) PowerWatts {
	if enc == nil {
		return PowerWatts{}
	}
	p := PowerWatts{
		Avg24hW:   enc.PowerAvg24Hrs.Float64(),
		Peak24hW:  enc.PowerPeak24Hrs.Float64(),
		RatedMaxW: enc.PowerCapacity24Hrs.Float64(),
	}
	if p.HasMeasured() || p.RatedMaxW > 0 {
		p.Source = "enclosure"
	}
	return p
}

// DeriveServerPower собирает мощность сервера из environmentalConfiguration и контекста export.
func DeriveServerPower(exp *ServerExport) PowerWatts {
	if exp == nil {
		return PowerWatts{}
	}
	p := ParseEnvironmentalPower(exp.Environment)
	if p.Source == "" && (p.Avg24hW > 0 || p.Peak24hW > 0 || p.RatedMaxW > 0) {
		p.Source = "environment"
	}
	if serverPowerOff(exp.Identity.PowerState) {
		p.Avg24hW = 0
		p.Peak24hW = 0
	}
	return p
}

func serverPowerOff(state string) bool {
	s := strings.ToLower(strings.TrimSpace(state))
	return s != "" && s != "on"
}

// ParseEnvironmentalPower извлекает ватты из GET .../environmentalConfiguration (схема у HPE плавает).
func ParseEnvironmentalPower(raw json.RawMessage) PowerWatts {
	if len(raw) == 0 {
		return PowerWatts{}
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return PowerWatts{}
	}
	avgKeys := []string{
		"averagePower", "averagePowerConsumed", "powerAverage", "avgPower",
		"averagePowerWatts", "powerAvg24Hrs", "averagePowerWattsConsumed",
	}
	peakKeys := []string{
		"peakPower", "maximumPowerConsumed", "maxPower", "peakPowerWatts",
		"powerPeak24Hrs", "maximumPowerWatts", "peakPowerWattsConsumed",
	}
	ratedKeys := []string{
		"powerCapWatts", "calibratedMaxPower", "powerCapacityWatts", "ratedPower",
		"powerCapacity24Hrs", "maxPower", "powerSupplyWatts",
	}
	var p PowerWatts
	p.Avg24hW = firstPowerInTree(root, avgKeys...)
	p.Peak24hW = firstPowerInTree(root, peakKeys...)
	p.RatedMaxW = firstPowerInTree(root, ratedKeys...)
	if p.Avg24hW > 0 || p.Peak24hW > 0 || p.RatedMaxW > 0 {
		p.Source = "environment"
	}
	return p
}

func firstPowerInTree(node any, keys ...string) float64 {
	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keySet[strings.ToLower(k)] = struct{}{}
	}
	var best float64
	var walk func(any)
	walk = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			for k, val := range v {
				if _, ok := keySet[strings.ToLower(k)]; ok {
					if w := anyToWatts(val); w > best {
						best = w
					}
				}
				walk(val)
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(node)
	return best
}

func anyToWatts(v any) float64 {
	switch x := v.(type) {
	case float64:
		if x > 0 && !math.IsNaN(x) {
			return x
		}
	case json.Number:
		f, err := x.Float64()
		if err == nil && f > 0 {
			return f
		}
	case string:
		x = strings.TrimSpace(x)
		x = strings.TrimSuffix(x, ",")
		if x == "" {
			return 0
		}
		f, err := parsePowerString(x)
		if err == nil && f > 0 {
			return f
		}
	}
	return 0
}

// EstimateEnclosurePowerFromBlades суммирует мощность включённых блейдов + overhead (диагностика / fallback).
func EstimateEnclosurePowerFromBlades(bladePowers []PowerWatts, overheadW float64) PowerWatts {
	if overheadW < 0 {
		overheadW = 0
	}
	var avg, peak, rated float64
	for _, b := range bladePowers {
		avg += b.Avg24hW
		peak += b.Peak24hW
		rated += b.RatedMaxW
	}
	if avg == 0 && peak == 0 && rated == 0 && overheadW == 0 {
		return PowerWatts{}
	}
	return PowerWatts{
		Avg24hW:   avg + overheadW,
		Peak24hW:  peak + overheadW,
		RatedMaxW: rated + overheadW,
		Source:    "estimate",
	}
}

// DefaultBladeChassisOverheadW — базовый overhead шасси (interconnect/OA/PS/fan), если OneView не отдал enclosure power.
const DefaultBladeChassisOverheadW = 450.0

func mergePowerWatts(dst *PowerWatts, src PowerWatts) {
	if src.Source == "" && !src.HasMeasured() && src.RatedMaxW <= 0 {
		return
	}
	preferSrc := src.HasMeasured() || (src.RatedMaxW > 0 && dst.RatedMaxW == 0)
	if preferSrc {
		if src.Avg24hW > 0 || dst.Avg24hW == 0 {
			dst.Avg24hW = src.Avg24hW
		}
		if src.Peak24hW > 0 || dst.Peak24hW == 0 {
			dst.Peak24hW = src.Peak24hW
		}
		if src.RatedMaxW > 0 || dst.RatedMaxW == 0 {
			dst.RatedMaxW = src.RatedMaxW
		}
		if src.Source != "" {
			dst.Source = src.Source
		}
	}
}
