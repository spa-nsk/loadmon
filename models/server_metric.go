package models

import (
	"context"
	"fmt"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ServerMetric — поминутный снимок метрик одного сервера.
type ServerMetric struct {
	Id                 int64     `form:"Id" orm:"auto;pk;column(id)"`
	HostId             int64     `form:"HostId" orm:"column(host_id);index"`                                            //ссылка на ZabbixHost.Id
	CollectedAt        time.Time `form:"CollectedAt" orm:"column(collected_at);type(datetime);index"`                   // CollectedAt — время сбора метрики (начало минуты).
	ServerName         string    `form:"ServerName" orm:"column(server_name);size(255);index"`                          // ServerName — имя сервера (видимое имя хоста в Zabbix).
	CPUUtilPercent     float64   `form:"CPUUtilPercent" orm:"column(cpu_util_percent);digits(10);decimals(4)"`          // CPUUtilPercent — общая утилизация CPU, %.
	RAMUtilPercent     float64   `form:"RAMUtilPercent" orm:"column(ram_util_percent);digits(10);decimals(4)"`          // RAMUtilPercent — утилизация RAM, %.
	DiskReadMBps       float64   `form:"DiskReadMBps" orm:"column(disk_read_mbps);digits(16);decimals(6)"`              // DiskReadMBps — чтение с дисков, МБ/с.
	DiskWriteMBps      float64   `form:"DiskWriteMBps" orm:"column(disk_write_mbps);digits(16);decimals(6)"`            // DiskWriteMBps — запись на диски, МБ/с.
	DiskReadLatencyMs  float64   `form:"DiskReadLatencyMs" orm:"column(disk_read_latency_ms);digits(16);decimals(6)"`   // DiskReadLatencyMs — задержка операций чтения с диска, мс.
	DiskWriteLatencyMs float64   `form:"DiskWriteLatencyMs" orm:"column(disk_write_latency_ms);digits(16);decimals(6)"` // DiskWriteLatencyMs — задержка операций записи на диск, мс.
	DiskReadQueue      float64   `form:"DiskReadQueue" orm:"column(disk_read_queue);digits(16);decimals(6)"`            // DiskReadQueue — длина очереди чтения с диска.
	DiskWriteQueue     float64   `form:"DiskWriteQueue" orm:"column(disk_write_queue);digits(16);decimals(6)"`          // DiskWriteQueue — длина очереди записи на диск.
	NetRecvMBps        float64   `form:"NetRecvMBps" orm:"column(net_recv_mbps);digits(16);decimals(6)"`                // NetRecvMBps — приём из сети, МБ/с.
	NetSentMBps        float64   `form:"NetSentMBps" orm:"column(net_sent_mbps);digits(16);decimals(6)"`                // NetSentMBps — передача в сеть, МБ/с.
}

func (m *ServerMetric) TableName() string {
	return "server_metric"
}

func (m *ServerMetric) TableEngine() string {
	return "INNODB"
}

func (m *ServerMetric) TableUnique() [][]string {
	return [][]string{
		{"HostId", "CollectedAt"},
	}
}

// TableIndex() намеренно не объявлен: у HostId/CollectedAt/ServerName уже
// стоит тег `orm:"...;index"` на самом поле (см. выше) - дублирование тех
// же одиночных колонок здесь приводило к двум одинаковым индексам с одним
// именем и падению RunSyncdb с "index ... already exists" на чистой базе
// (rtOnError=true прерывает синхронизацию остальных моделей). См.
// аналогичный фикс и комментарий в models/storage_metric.go, CHANGELOG.

// ReplaceMetrics записывает снимки за минуту, заменяя уже существующие строки
// с той же парой (HostId, CollectedAt).
func ReplaceMetrics(ctx context.Context, rows []*ServerMetric) error {
	if len(rows) == 0 {
		return nil
	}
	o := orm.NewOrm()
	return o.DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		qs := tx.QueryTable(new(ServerMetric))
		for _, row := range rows {
			if row.HostId <= 0 {
				return fmt.Errorf("models: ServerMetric.HostId is required")
			}
			if _, err := qs.Filter("HostId", row.HostId).
				Filter("CollectedAt", row.CollectedAt).
				Delete(); err != nil {
				return err
			}
		}
		_, err := tx.InsertMultiWithCtx(ctx, 50, rows)
		return err
	})
}
