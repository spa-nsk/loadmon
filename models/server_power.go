package models

import (
	"context"
	"fmt"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ServerPowerMetric - поминутный замер потребляемой мощности сервера
// ("Present Power Reading" из Zabbix, пакет zabbixpower). Привязан к
// ServerSprRecords.Id, а не к хосту Zabbix: серверов Zabbix несколько, у
// каждого свои hostid. Хранится 3 суток, затем остаётся только суточная
// агрегация ServerPowerRecord.
type ServerPowerMetric struct {
	Id          int64     `form:"Id" orm:"auto;pk;column(id)"`
	ServerId    int64     `form:"ServerId" orm:"column(server_id);index"`                      // ServerSprRecords.Id
	CollectedAt time.Time `form:"CollectedAt" orm:"column(collected_at);type(datetime);index"` // начало минуты сбора
	HostName    string    `form:"HostName" orm:"column(host_name);size(255)"`                  // имя хоста в Zabbix (ILO-...), по которому найден сервер
	Watts       float64   `form:"Watts" orm:"column(watts);digits(12);decimals(3)"`
}

func (m *ServerPowerMetric) TableName() string   { return "server_power_metric" }
func (m *ServerPowerMetric) TableEngine() string { return "INNODB" }
func (m *ServerPowerMetric) TableUnique() [][]string {
	return [][]string{{"ServerId", "CollectedAt"}}
}

// ServerPowerRecord - суточная агрегация мощности сервера (одна запись на
// ServerId и сутки), строится из server_power_metric в 00:00.
type ServerPowerRecord struct {
	Id           int64     `form:"Id" orm:"auto;pk;column(id)"`
	ServerId     int64     `form:"ServerId" orm:"column(server_id);index"`
	Date         time.Time `form:"Date" orm:"column(date);type(date);index"` // сутки (00:00 локального времени)
	WattsMax     float64   `form:"WattsMax" orm:"column(watts_max);digits(12);decimals(3)"`
	WattsMaxTime time.Time `form:"WattsMaxTime" orm:"column(watts_max_time);null"`
	WattsAvg     float64   `form:"WattsAvg" orm:"column(watts_avg);digits(12);decimals(3)"`
	WattsMin     float64   `form:"WattsMin" orm:"column(watts_min);digits(12);decimals(3)"`
	WattsMinTime time.Time `form:"WattsMinTime" orm:"column(watts_min_time);null"`
	EnergyKWh    float64   `form:"EnergyKWh" orm:"column(energy_kwh);digits(12);decimals(4)"` // сумма замеров * 1 мин
	PerfCount    int64     `form:"PerfCount" orm:"column(perf_count)"`                        // число минутных замеров в агрегате
	Time_Mfy     time.Time `form:"Time_Mfy" orm:"column(time_mfy)"`
}

func (m *ServerPowerRecord) TableName() string   { return "server_power_record" }
func (m *ServerPowerRecord) TableEngine() string { return "INNODB" }
func (m *ServerPowerRecord) TableUnique() [][]string {
	return [][]string{{"ServerId", "Date"}}
}

// ReplaceServerPower записывает минутные замеры, заменяя строки с той же
// парой (ServerId, CollectedAt).
func ReplaceServerPower(ctx context.Context, rows []*ServerPowerMetric) error {
	if len(rows) == 0 {
		return nil
	}
	return orm.NewOrm().DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		qs := tx.QueryTable(new(ServerPowerMetric))
		for _, r := range rows {
			if r.ServerId <= 0 {
				return fmt.Errorf("models: ServerPowerMetric.ServerId is required")
			}
			if _, err := qs.Filter("ServerId", r.ServerId).Filter("CollectedAt", r.CollectedAt).Delete(); err != nil {
				return err
			}
		}
		_, err := tx.InsertMultiWithCtx(ctx, 50, rows)
		return err
	})
}
