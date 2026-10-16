package zabbixpower

import (
	"log"

	"github.com/beego/beego/v2/client/orm"

	"loadmon/models"
)

// initHardwarePower: при первом замере (PowerAvg24hW и PowerPeak24hW ещё 0)
// записывает текущую мощность в оба поля ServerHardware. Дальше поля
// обновляет суточная агрегация (setHardwarePower).
func initHardwarePower(rows []*models.ServerPowerMetric) {
	if len(rows) == 0 {
		return
	}
	watts := make(map[int64]float64, len(rows))
	for _, r := range rows {
		watts[r.ServerId] = r.Watts
	}
	o := orm.NewOrm()
	var empty []*models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).
		Filter("ServerId__gt", 0).Filter("PowerAvg24hW", 0).Filter("PowerPeak24hW", 0).
		Exclude("SourceUpdateType", models.SourceUpdateManual).
		All(&empty, "Id", "ServerId"); err != nil {
		log.Printf("zabbixpower: read server_hardware: %v", err)
		return
	}
	for _, h := range empty {
		w, ok := watts[h.ServerId]
		if !ok || w <= 0 {
			continue
		}
		if _, err := o.QueryTable(new(models.ServerHardware)).Filter("Id", h.Id).
			Update(orm.Params{"PowerAvg24hW": w, "PowerPeak24hW": w}); err != nil {
			log.Printf("zabbixpower: init power hardware=%d: %v", h.Id, err)
		}
	}
}

// setHardwarePower записывает суточные среднее и максимум в ServerHardware.
func setHardwarePower(o orm.Ormer, rec *models.ServerPowerRecord) {
	if _, err := o.QueryTable(new(models.ServerHardware)).
		Filter("ServerId", rec.ServerId).
		Exclude("SourceUpdateType", models.SourceUpdateManual).
		Update(orm.Params{"PowerAvg24hW": rec.WattsAvg, "PowerPeak24hW": rec.WattsMax}); err != nil {
		log.Printf("zabbixpower: set power server=%d: %v", rec.ServerId, err)
	}
}
