package models

import "time"

// StorageMetricRecord — суточная агрегация поминутных метрик СХД
// (models.StorageMetric) с max/min/avg и временем достижения extrema.
type StorageMetricRecord struct {
	Id               int64     `form:"Id" orm:"auto;pk"`
	StorageSystemsId int64     `form:"StorageSystemsId" orm:"index"`
	ShortName        string    `form:"ShortName" orm:"size(64);index"`
	Date             time.Time `form:"Date" orm:"type(date);index"`

	CPUUtilMaximum     float64   `form:"CPUUtilMaximum" orm:"digits(10);decimals(4)"`
	CPUUtilMaximumTime time.Time `form:"CPUUtilMaximumTime" orm:"null"`
	CPUUtilAverage     float64   `form:"CPUUtilAverage" orm:"digits(10);decimals(4)"`
	CPUUtilMinimum     float64   `form:"CPUUtilMinimum" orm:"digits(10);decimals(4)"`
	CPUUtilMinimumTime time.Time `form:"CPUUtilMinimumTime" orm:"null"`

	UsedPercentMaximum     float64   `form:"UsedPercentMaximum" orm:"digits(10);decimals(4)"`
	UsedPercentMaximumTime time.Time `form:"UsedPercentMaximumTime" orm:"null"`
	UsedPercentAverage     float64   `form:"UsedPercentAverage" orm:"digits(10);decimals(4)"`
	UsedPercentMinimum     float64   `form:"UsedPercentMinimum" orm:"digits(10);decimals(4)"`
	UsedPercentMinimumTime time.Time `form:"UsedPercentMinimumTime" orm:"null"`

	ReadIOPSMaximum     float64   `form:"ReadIOPSMaximum" orm:"digits(16);decimals(4)"`
	ReadIOPSMaximumTime time.Time `form:"ReadIOPSMaximumTime" orm:"null"`
	ReadIOPSAverage     float64   `form:"ReadIOPSAverage" orm:"digits(16);decimals(4)"`
	ReadIOPSMinimum     float64   `form:"ReadIOPSMinimum" orm:"digits(16);decimals(4)"`
	ReadIOPSMinimumTime time.Time `form:"ReadIOPSMinimumTime" orm:"null"`

	WriteIOPSMaximum     float64   `form:"WriteIOPSMaximum" orm:"digits(16);decimals(4)"`
	WriteIOPSMaximumTime time.Time `form:"WriteIOPSMaximumTime" orm:"null"`
	WriteIOPSAverage     float64   `form:"WriteIOPSAverage" orm:"digits(16);decimals(4)"`
	WriteIOPSMinimum     float64   `form:"WriteIOPSMinimum" orm:"digits(16);decimals(4)"`
	WriteIOPSMinimumTime time.Time `form:"WriteIOPSMinimumTime" orm:"null"`

	TotalIOPSMaximum     float64   `form:"TotalIOPSMaximum" orm:"digits(16);decimals(4)"`
	TotalIOPSMaximumTime time.Time `form:"TotalIOPSMaximumTime" orm:"null"`
	TotalIOPSAverage     float64   `form:"TotalIOPSAverage" orm:"digits(16);decimals(4)"`
	TotalIOPSMinimum     float64   `form:"TotalIOPSMinimum" orm:"digits(16);decimals(4)"`
	TotalIOPSMinimumTime time.Time `form:"TotalIOPSMinimumTime" orm:"null"`

	ReadMBpsMaximum     float64   `form:"ReadMBpsMaximum" orm:"digits(16);decimals(6)"`
	ReadMBpsMaximumTime time.Time `form:"ReadMBpsMaximumTime" orm:"null"`
	ReadMBpsAverage     float64   `form:"ReadMBpsAverage" orm:"digits(16);decimals(6)"`
	ReadMBpsMinimum     float64   `form:"ReadMBpsMinimum" orm:"digits(16);decimals(6)"`
	ReadMBpsMinimumTime time.Time `form:"ReadMBpsMinimumTime" orm:"null"`

	WriteMBpsMaximum     float64   `form:"WriteMBpsMaximum" orm:"digits(16);decimals(6)"`
	WriteMBpsMaximumTime time.Time `form:"WriteMBpsMaximumTime" orm:"null"`
	WriteMBpsAverage     float64   `form:"WriteMBpsAverage" orm:"digits(16);decimals(6)"`
	WriteMBpsMinimum     float64   `form:"WriteMBpsMinimum" orm:"digits(16);decimals(6)"`
	WriteMBpsMinimumTime time.Time `form:"WriteMBpsMinimumTime" orm:"null"`

	ReadLatencyMsMaximum     float64   `form:"ReadLatencyMsMaximum" orm:"digits(16);decimals(6)"`
	ReadLatencyMsMaximumTime time.Time `form:"ReadLatencyMsMaximumTime" orm:"null"`
	ReadLatencyMsAverage     float64   `form:"ReadLatencyMsAverage" orm:"digits(16);decimals(6)"`
	ReadLatencyMsMinimum     float64   `form:"ReadLatencyMsMinimum" orm:"digits(16);decimals(6)"`
	ReadLatencyMsMinimumTime time.Time `form:"ReadLatencyMsMinimumTime" orm:"null"`

	WriteLatencyMsMaximum     float64   `form:"WriteLatencyMsMaximum" orm:"digits(16);decimals(6)"`
	WriteLatencyMsMaximumTime time.Time `form:"WriteLatencyMsMaximumTime" orm:"null"`
	WriteLatencyMsAverage     float64   `form:"WriteLatencyMsAverage" orm:"digits(16);decimals(6)"`
	WriteLatencyMsMinimum     float64   `form:"WriteLatencyMsMinimum" orm:"digits(16);decimals(6)"`
	WriteLatencyMsMinimumTime time.Time `form:"WriteLatencyMsMinimumTime" orm:"null"`

	PerfCount int64     `form:"PerfCount" orm:"column(perf_count)"`
	Time_Mfy  time.Time `form:"Time_Mfy" orm:"column(time_mfy)"`
}

func (m *StorageMetricRecord) TableName() string { return "storage_metric_record" }

func (m *StorageMetricRecord) TableUnique() [][]string {
	return [][]string{{"StorageSystemsId", "Date"}}
}

func (m *StorageMetricRecord) TableIndex() [][]string {
	return [][]string{
		{"Date"},
		{"StorageSystemsId"},
		{"ShortName"},
	}
}
