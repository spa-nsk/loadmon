package models

import (
	"time"
)

// ServerMetricRecord — суточная агрегированная запись по метрикам сервера
// (CPU/RAM/диск/сеть), построенная из поминутных "сырых" снимков
// models.ServerMetric (таблица server_metric, см. пакет zabbixmetrics) -
// по аналогии с тем, как models.VMRecord (таблица vm_record) является
// суточной агрегацией "сырых" замеров models.VMPerfRecord (vm_perf).
//
// Один HostId (см. models.ServerMetric.HostId - ссылка на
// models.ZabbixHost.Id) и одни сутки (Date, время обнулено до начала дня,
// как и у VMRecord.Date) - одна запись. Формируется в
// cmdb.SaveServerMetricRecord, вызываемой из cmdb.SaveMetricData (см.
// metrics_grpc/metrics_grpc.go).
type ServerMetricRecord struct {
	Id         int64     `form:"Id" orm:"auto;pk;column(id)"`
	HostId     int64     `form:"HostId" orm:"column(host_id);index"`       // ссылка на models.ZabbixHost.Id (как и у ServerMetric.HostId)
	ServerId   int64     `form:"ServerId" orm:"column(server_id);index"`   // ссылка на models.ServerSprRecords.Id (через ZabbixHost.ServerId, если он проставлен)
	ServerName string    `form:"ServerName" orm:"column(server_name);size(255);index"` // видимое имя хоста в Zabbix (денормализовано, как и у ServerMetric)
	Date       time.Time `form:"Date" orm:"column(date);type(date);index"` // сутки, за которые агрегированы данные (00:00 локального времени)

	CPUUtilMaximum     float64   `form:"CPUUtilMaximum" orm:"column(cpu_util_maximum);digits(10);decimals(4)"`
	CPUUtilMaximumTime time.Time `form:"CPUUtilMaximumTime" orm:"column(cpu_util_maximum_time);null"`
	CPUUtilAverage     float64   `form:"CPUUtilAverage" orm:"column(cpu_util_average);digits(10);decimals(4)"`
	CPUUtilMinimum     float64   `form:"CPUUtilMinimum" orm:"column(cpu_util_minimum);digits(10);decimals(4)"`
	CPUUtilMinimumTime time.Time `form:"CPUUtilMinimumTime" orm:"column(cpu_util_minimum_time);null"`

	RAMUtilMaximum     float64   `form:"RAMUtilMaximum" orm:"column(ram_util_maximum);digits(10);decimals(4)"`
	RAMUtilMaximumTime time.Time `form:"RAMUtilMaximumTime" orm:"column(ram_util_maximum_time);null"`
	RAMUtilAverage     float64   `form:"RAMUtilAverage" orm:"column(ram_util_average);digits(10);decimals(4)"`
	RAMUtilMinimum     float64   `form:"RAMUtilMinimum" orm:"column(ram_util_minimum);digits(10);decimals(4)"`
	RAMUtilMinimumTime time.Time `form:"RAMUtilMinimumTime" orm:"column(ram_util_minimum_time);null"`

	DiskReadMBpsMaximum     float64   `form:"DiskReadMBpsMaximum" orm:"column(disk_read_mbps_maximum);digits(16);decimals(6)"`
	DiskReadMBpsMaximumTime time.Time `form:"DiskReadMBpsMaximumTime" orm:"column(disk_read_mbps_maximum_time);null"`
	DiskReadMBpsAverage     float64   `form:"DiskReadMBpsAverage" orm:"column(disk_read_mbps_average);digits(16);decimals(6)"`
	DiskReadMBpsMinimum     float64   `form:"DiskReadMBpsMinimum" orm:"column(disk_read_mbps_minimum);digits(16);decimals(6)"`
	DiskReadMBpsMinimumTime time.Time `form:"DiskReadMBpsMinimumTime" orm:"column(disk_read_mbps_minimum_time);null"`

	DiskWriteMBpsMaximum     float64   `form:"DiskWriteMBpsMaximum" orm:"column(disk_write_mbps_maximum);digits(16);decimals(6)"`
	DiskWriteMBpsMaximumTime time.Time `form:"DiskWriteMBpsMaximumTime" orm:"column(disk_write_mbps_maximum_time);null"`
	DiskWriteMBpsAverage     float64   `form:"DiskWriteMBpsAverage" orm:"column(disk_write_mbps_average);digits(16);decimals(6)"`
	DiskWriteMBpsMinimum     float64   `form:"DiskWriteMBpsMinimum" orm:"column(disk_write_mbps_minimum);digits(16);decimals(6)"`
	DiskWriteMBpsMinimumTime time.Time `form:"DiskWriteMBpsMinimumTime" orm:"column(disk_write_mbps_minimum_time);null"`

	DiskReadLatencyMsMaximum     float64   `form:"DiskReadLatencyMsMaximum" orm:"column(disk_read_latency_ms_maximum);digits(16);decimals(6)"`
	DiskReadLatencyMsMaximumTime time.Time `form:"DiskReadLatencyMsMaximumTime" orm:"column(disk_read_latency_ms_maximum_time);null"`
	DiskReadLatencyMsAverage     float64   `form:"DiskReadLatencyMsAverage" orm:"column(disk_read_latency_ms_average);digits(16);decimals(6)"`
	DiskReadLatencyMsMinimum     float64   `form:"DiskReadLatencyMsMinimum" orm:"column(disk_read_latency_ms_minimum);digits(16);decimals(6)"`
	DiskReadLatencyMsMinimumTime time.Time `form:"DiskReadLatencyMsMinimumTime" orm:"column(disk_read_latency_ms_minimum_time);null"`

	DiskWriteLatencyMsMaximum     float64   `form:"DiskWriteLatencyMsMaximum" orm:"column(disk_write_latency_ms_maximum);digits(16);decimals(6)"`
	DiskWriteLatencyMsMaximumTime time.Time `form:"DiskWriteLatencyMsMaximumTime" orm:"column(disk_write_latency_ms_maximum_time);null"`
	DiskWriteLatencyMsAverage     float64   `form:"DiskWriteLatencyMsAverage" orm:"column(disk_write_latency_ms_average);digits(16);decimals(6)"`
	DiskWriteLatencyMsMinimum     float64   `form:"DiskWriteLatencyMsMinimum" orm:"column(disk_write_latency_ms_minimum);digits(16);decimals(6)"`
	DiskWriteLatencyMsMinimumTime time.Time `form:"DiskWriteLatencyMsMinimumTime" orm:"column(disk_write_latency_ms_minimum_time);null"`

	DiskReadQueueMaximum     float64   `form:"DiskReadQueueMaximum" orm:"column(disk_read_queue_maximum);digits(16);decimals(6)"`
	DiskReadQueueMaximumTime time.Time `form:"DiskReadQueueMaximumTime" orm:"column(disk_read_queue_maximum_time);null"`
	DiskReadQueueAverage     float64   `form:"DiskReadQueueAverage" orm:"column(disk_read_queue_average);digits(16);decimals(6)"`
	DiskReadQueueMinimum     float64   `form:"DiskReadQueueMinimum" orm:"column(disk_read_queue_minimum);digits(16);decimals(6)"`
	DiskReadQueueMinimumTime time.Time `form:"DiskReadQueueMinimumTime" orm:"column(disk_read_queue_minimum_time);null"`

	DiskWriteQueueMaximum     float64   `form:"DiskWriteQueueMaximum" orm:"column(disk_write_queue_maximum);digits(16);decimals(6)"`
	DiskWriteQueueMaximumTime time.Time `form:"DiskWriteQueueMaximumTime" orm:"column(disk_write_queue_maximum_time);null"`
	DiskWriteQueueAverage     float64   `form:"DiskWriteQueueAverage" orm:"column(disk_write_queue_average);digits(16);decimals(6)"`
	DiskWriteQueueMinimum     float64   `form:"DiskWriteQueueMinimum" orm:"column(disk_write_queue_minimum);digits(16);decimals(6)"`
	DiskWriteQueueMinimumTime time.Time `form:"DiskWriteQueueMinimumTime" orm:"column(disk_write_queue_minimum_time);null"`

	NetRecvMBpsMaximum     float64   `form:"NetRecvMBpsMaximum" orm:"column(net_recv_mbps_maximum);digits(16);decimals(6)"`
	NetRecvMBpsMaximumTime time.Time `form:"NetRecvMBpsMaximumTime" orm:"column(net_recv_mbps_maximum_time);null"`
	NetRecvMBpsAverage     float64   `form:"NetRecvMBpsAverage" orm:"column(net_recv_mbps_average);digits(16);decimals(6)"`
	NetRecvMBpsMinimum     float64   `form:"NetRecvMBpsMinimum" orm:"column(net_recv_mbps_minimum);digits(16);decimals(6)"`
	NetRecvMBpsMinimumTime time.Time `form:"NetRecvMBpsMinimumTime" orm:"column(net_recv_mbps_minimum_time);null"`

	NetSentMBpsMaximum     float64   `form:"NetSentMBpsMaximum" orm:"column(net_sent_mbps_maximum);digits(16);decimals(6)"`
	NetSentMBpsMaximumTime time.Time `form:"NetSentMBpsMaximumTime" orm:"column(net_sent_mbps_maximum_time);null"`
	NetSentMBpsAverage     float64   `form:"NetSentMBpsAverage" orm:"column(net_sent_mbps_average);digits(16);decimals(6)"`
	NetSentMBpsMinimum     float64   `form:"NetSentMBpsMinimum" orm:"column(net_sent_mbps_minimum);digits(16);decimals(6)"`
	NetSentMBpsMinimumTime time.Time `form:"NetSentMBpsMinimumTime" orm:"column(net_sent_mbps_minimum_time);null"`

	PerfCount int64     `form:"PerfCount" orm:"column(perf_count)"` // количество поминутных снимков server_metric, вошедших в агрегат за сутки
	Time_Mfy  time.Time `form:"Time_Mfy" orm:"column(time_mfy)"`    // время формирования/обновления этой суточной записи
}

func (m *ServerMetricRecord) TableName() string {
	return "server_metric_record"
}

func (m *ServerMetricRecord) TableEngine() string {
	return "INNODB"
}

func (m *ServerMetricRecord) TableUnique() [][]string {
	return [][]string{
		{"HostId", "Date"},
	}
}

// TableIndex() намеренно не объявлен - HostId/ServerId/ServerName/Date уже
// проиндексированы тегом `orm:"...;index"` на самих полях (см. выше);
// повтор тех же колонок здесь давал дублирующиеся индексы и падение
// RunSyncdb на чистой базе (см. models/storage_metric.go и CHANGELOG).
