package models

import (
	"context"
	"fmt"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Источники метрик server_metric/server_metric_record - значения поля
// Source (добавлено 2026-10-01 вместе со сборщиком scommetrics). HostId
// трактуется В ПАРЕ с Source: у каждого источника свой справочник хостов
// со своей нумерацией, идентификаторы двух справочников пересекаются.
const (
	MetricSourceZabbix = 0 // HostId -> models.ZabbixHost.Id (пакет zabbixmetrics); 0 - значение по умолчанию для всех строк, записанных до появления поля
	MetricSourceSCOM   = 1 // HostId -> models.ScomHost.Id (пакет scommetrics)
)

// ServerMetric — снимок метрик одного сервера на момент CollectedAt
// (Zabbix - поминутно, SCOM - с периодом опроса, по умолчанию 5 минут).
type ServerMetric struct {
	Id                 int64     `form:"Id" orm:"auto;pk;column(id)"`
	Source             int       `form:"Source" orm:"column(source);default(0)"`                                         // источник: MetricSourceZabbix/MetricSourceSCOM - определяет, на какой справочник ссылается HostId
	HostId             int64     `form:"HostId" orm:"column(host_id);index"`                                            // ссылка на ZabbixHost.Id (Source=0) или ScomHost.Id (Source=1)
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
		{"Source", "HostId", "CollectedAt"},
	}
}

// Уникальность (Source, HostId, CollectedAt) - на уже существующих базах
// старое ограничение (HostId, CollectedAt) RunSyncdb сам не меняет, его
// переносит config.migrateServerMetricSourceUnique (config/dbinit_servermetric.go).

// TableIndex() намеренно не объявлен: у HostId/CollectedAt/ServerName уже
// стоит тег `orm:"...;index"` на самом поле (см. выше) - дублирование тех
// же одиночных колонок здесь приводило к двум одинаковым индексам с одним
// именем и падению RunSyncdb с "index ... already exists" на чистой базе
// (rtOnError=true прерывает синхронизацию остальных моделей). См.
// аналогичный фикс и комментарий в models/storage_metric.go, CHANGELOG.

// ReplaceMetrics записывает снимки, заменяя уже существующие строки с той
// же тройкой (Source, HostId, CollectedAt) - Source берётся из каждой строки,
// поэтому снимки Zabbix и SCOM с совпадающими HostId друг друга не затирают.
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
			if _, err := qs.Filter("Source", row.Source).
				Filter("HostId", row.HostId).
				Filter("CollectedAt", row.CollectedAt).
				Delete(); err != nil {
				return err
			}
		}
		_, err := tx.InsertMultiWithCtx(ctx, 50, rows)
		return err
	})
}

// ServerMetricHostFor возвращает, из какого источника брать метрики
// server_metric/server_metric_record для карточки сервера - единое правило
// приоритета для всех потребителей (отчёты, выгрузки):
//   - есть ZabbixHostId -> Zabbix (данные поминутные), даже если сервер
//     одновременно есть и в SCOM;
//   - иначе есть ScomHostId -> SCOM;
//   - иначе метрик нет (ok == false).
func ServerMetricHostFor(srv *ServerSprRecords) (source int, hostId int64, ok bool) {
	if srv == nil {
		return 0, 0, false
	}
	if srv.ZabbixHostId > 0 {
		return MetricSourceZabbix, srv.ZabbixHostId, true
	}
	if srv.ScomHostId > 0 {
		return MetricSourceSCOM, srv.ScomHostId, true
	}
	return 0, 0, false
}
