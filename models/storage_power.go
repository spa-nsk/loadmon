package models

import (
	"context"
	"fmt"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// StoragePowerMetric - поминутный замер потребляемой мощности СХД
// (Unity: сумма currentPower DPE+DAE; Huawei: get_realtime_power_consumption).
// Хранится 3 суток, далее - суточная агрегация StoragePowerRecord.
type StoragePowerMetric struct {
	Id               int64     `form:"Id" orm:"auto;pk;column(id)"`
	StorageSystemsId int64     `form:"StorageSystemsId" orm:"column(storage_systems_id);index"`
	CollectedAt      time.Time `form:"CollectedAt" orm:"column(collected_at);type(datetime);index"`
	Watts            float64   `form:"Watts" orm:"column(watts);digits(12);decimals(3)"`
}

func (m *StoragePowerMetric) TableName() string   { return "storage_power_metric" }
func (m *StoragePowerMetric) TableEngine() string { return "INNODB" }
func (m *StoragePowerMetric) TableUnique() [][]string {
	return [][]string{{"StorageSystemsId", "CollectedAt"}}
}

// StoragePowerRecord - суточная агрегация мощности СХД (как ServerPowerRecord).
type StoragePowerRecord struct {
	Id               int64     `form:"Id" orm:"auto;pk;column(id)"`
	StorageSystemsId int64     `form:"StorageSystemsId" orm:"column(storage_systems_id);index"`
	Date             time.Time `form:"Date" orm:"column(date);type(date);index"`
	WattsMax         float64   `form:"WattsMax" orm:"column(watts_max);digits(12);decimals(3)"`
	WattsMaxTime     time.Time `form:"WattsMaxTime" orm:"column(watts_max_time);null"`
	WattsAvg         float64   `form:"WattsAvg" orm:"column(watts_avg);digits(12);decimals(3)"`
	WattsMin         float64   `form:"WattsMin" orm:"column(watts_min);digits(12);decimals(3)"`
	WattsMinTime     time.Time `form:"WattsMinTime" orm:"column(watts_min_time);null"`
	EnergyKWh        float64   `form:"EnergyKWh" orm:"column(energy_kwh);digits(12);decimals(4)"`
	PerfCount        int64     `form:"PerfCount" orm:"column(perf_count)"`
	Time_Mfy         time.Time `form:"Time_Mfy" orm:"column(time_mfy)"`
}

func (m *StoragePowerRecord) TableName() string   { return "storage_power_record" }
func (m *StoragePowerRecord) TableEngine() string { return "INNODB" }
func (m *StoragePowerRecord) TableUnique() [][]string {
	return [][]string{{"StorageSystemsId", "Date"}}
}

// RackPowerRecord - суточная мощность шкафа: серверы + блейд-корзины +
// СХД (пакет rackpower, вариант C), на момент формирования записи.
type RackPowerRecord struct {
	Id            int64     `form:"Id" orm:"auto;pk;column(id)"`
	RackCabinetId int64     `form:"RackCabinetId" orm:"column(rack_cabinet_id);index"`
	Date          time.Time `form:"Date" orm:"column(date);type(date);index"`
	AvgW          float64   `form:"AvgW" orm:"column(avg_w);digits(12);decimals(3)"`
	PeakW         float64   `form:"PeakW" orm:"column(peak_w);digits(12);decimals(3)"` // сумма суточных пиков устройств (верхняя оценка)
	RatedMaxW     float64   `form:"RatedMaxW" orm:"column(rated_max_w);digits(12);decimals(3)"`
	Time_Mfy      time.Time `form:"Time_Mfy" orm:"column(time_mfy)"`
}

func (m *RackPowerRecord) TableName() string   { return "rack_power_record" }
func (m *RackPowerRecord) TableEngine() string { return "INNODB" }
func (m *RackPowerRecord) TableUnique() [][]string {
	return [][]string{{"RackCabinetId", "Date"}}
}

// SitePowerRecord - суточная мощность ЦОД/площадки (сумма по шкафам и СХД
// без шкафа).
type SitePowerRecord struct {
	Id               int64     `form:"Id" orm:"auto;pk;column(id)"`
	DatacenterSiteId int64     `form:"DatacenterSiteId" orm:"column(datacenter_site_id);index"`
	Date             time.Time `form:"Date" orm:"column(date);type(date);index"`
	AvgW             float64   `form:"AvgW" orm:"column(avg_w);digits(12);decimals(3)"`
	PeakW            float64   `form:"PeakW" orm:"column(peak_w);digits(12);decimals(3)"`
	RatedMaxW        float64   `form:"RatedMaxW" orm:"column(rated_max_w);digits(12);decimals(3)"`
	Time_Mfy         time.Time `form:"Time_Mfy" orm:"column(time_mfy)"`
}

func (m *SitePowerRecord) TableName() string   { return "site_power_record" }
func (m *SitePowerRecord) TableEngine() string { return "INNODB" }
func (m *SitePowerRecord) TableUnique() [][]string {
	return [][]string{{"DatacenterSiteId", "Date"}}
}

// ReplaceStoragePower записывает минутные замеры СХД, заменяя строки с той же
// парой (StorageSystemsId, CollectedAt).
func ReplaceStoragePower(ctx context.Context, rows []*StoragePowerMetric) error {
	if len(rows) == 0 {
		return nil
	}
	return orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		qs := tx.QueryTable(new(StoragePowerMetric))
		for _, r := range rows {
			if r.StorageSystemsId <= 0 {
				return fmt.Errorf("models: StoragePowerMetric.StorageSystemsId is required")
			}
			if _, err := qs.Filter("StorageSystemsId", r.StorageSystemsId).Filter("CollectedAt", r.CollectedAt).Delete(); err != nil {
				return err
			}
		}
		_, err := tx.InsertMultiWithCtx(ctx, 50, rows)
		return err
	})
}
