package cmdb

import (
	"fmt"
	"math"

	models "loadmon/models"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

func oneViewMayUpdateHardwarePower(h *models.ServerHardware) bool {
	if h == nil {
		return false
	}
	return h.SourceUpdateType != models.SourceUpdateManual
}

// ApplyServerHardwarePowerFromOneView проставляет Power* на ServerHardware (вариант C: environment).
func ApplyServerHardwarePowerFromOneView(h *models.ServerHardware, p oneview.PowerWatts) {
	if !oneViewMayUpdateHardwarePower(h) {
		return
	}
	if h.PowerState == 2 {
		h.PowerAvg24hW = 0
		h.PowerPeak24hW = 0
	} else {
		if p.Avg24hW >= 0 {
			h.PowerAvg24hW = p.Avg24hW
		}
		if p.Peak24hW >= 0 {
			h.PowerPeak24hW = p.Peak24hW
		}
	}
	if p.RatedMaxW > 0 {
		h.PowerRatedMaxW = p.RatedMaxW
	}
}

// ApplyBladeSystemPowerFromOneView — primary: enclosure power; fallback: sum(blades)+overhead.
func ApplyBladeSystemPowerFromOneView(o orm.QueryExecutor, bls *models.BladeSystems, enc *oneview.Enclosure) {
	if bls == nil {
		return
	}
	measured := oneview.EnclosurePowerWatts(enc)
	var applied oneview.PowerWatts
	if measured.HasMeasured() || measured.RatedMaxW > 0 {
		applied = measured
	} else {
		applied = estimateBladeSystemPowerFromDB(o, bls.Id)
	}
	bls.PowerAvg24hW = applied.Avg24hW
	bls.PowerPeak24hW = applied.Peak24hW
	if applied.RatedMaxW > 0 {
		bls.PowerRatedMaxW = applied.RatedMaxW
	}
}

func estimateBladeSystemPowerFromDB(o orm.QueryExecutor, bladeSystemId int64) oneview.PowerWatts {
	if o == nil || bladeSystemId <= 0 {
		return oneview.PowerWatts{}
	}
	var blades []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).
		Filter("BladeSystemId", bladeSystemId).
		Filter("PowerState", 1).
		All(&blades); err != nil {
		return oneview.PowerWatts{}
	}
	powers := make([]oneview.PowerWatts, 0, len(blades))
	for _, b := range blades {
		powers = append(powers, oneview.PowerWatts{
			Avg24hW:   b.PowerAvg24hW,
			Peak24hW:  b.PowerPeak24hW,
			RatedMaxW: b.PowerRatedMaxW,
		})
	}
	return oneview.EstimateEnclosurePowerFromBlades(powers, oneview.DefaultBladeChassisOverheadW)
}

// BladeSystemPowerDiagnostics сравнивает measured enclosure с суммой блейдов (только лог/утилита).
func BladeSystemPowerDiagnostics(o orm.QueryExecutor, enc *oneview.Enclosure, bladeSystemId int64) string {
	measured := oneview.EnclosurePowerWatts(enc)
	sum := estimateBladeSystemPowerFromDB(o, bladeSystemId)
	if !measured.HasMeasured() {
		return fmt.Sprintf("enclosure: no measured power; blade-sum estimate avg=%.0f peak=%.0f rated=%.0f",
			sum.Avg24hW, sum.Peak24hW, sum.RatedMaxW)
	}
	dAvg := math.Abs(measured.Avg24hW - sum.Avg24hW)
	dPeak := math.Abs(measured.Peak24hW - sum.Peak24hW)
	return fmt.Sprintf("enclosure avg=%.0f peak=%.0f rated=%.0f | blade-sum avg=%.0f peak=%.0f | delta avg=%.0f peak=%.0f",
		measured.Avg24hW, measured.Peak24hW, measured.RatedMaxW, sum.Avg24hW, sum.Peak24hW, dAvg, dPeak)
}
