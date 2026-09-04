package models

import (
	"context"
	"fmt"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// StorageMetric — поминутный снимок метрик СХД (детальное хранение 3 суток).
type StorageMetric struct {
	Id               int64     `form:"Id" orm:"auto;pk"`
	StorageSystemsId int64     `form:"StorageSystemsId" orm:"index"`
	CollectedAt      time.Time `form:"CollectedAt" orm:"type(datetime);index"`
	ShortName        string    `form:"ShortName" orm:"size(64);index"`

	CPUUtilPercent float64 `form:"CPUUtilPercent" orm:"digits(10);decimals(4)"`
	UsedPercent    float64 `form:"UsedPercent" orm:"digits(10);decimals(4)"`
	ReadIOPS       float64 `form:"ReadIOPS" orm:"digits(16);decimals(4)"`
	WriteIOPS      float64 `form:"WriteIOPS" orm:"digits(16);decimals(4)"`
	TotalIOPS      float64 `form:"TotalIOPS" orm:"digits(16);decimals(4)"`
	ReadMBps       float64 `form:"ReadMBps" orm:"digits(16);decimals(6)"`
	WriteMBps      float64 `form:"WriteMBps" orm:"digits(16);decimals(6)"`
	ReadLatencyMs  float64 `form:"ReadLatencyMs" orm:"digits(16);decimals(6)"`
	WriteLatencyMs float64 `form:"WriteLatencyMs" orm:"digits(16);decimals(6)"`

	// PollOK/PollError (2026-09-03) - отличают "реальный ноль" от "опрос
	// метрики не удался": раньше при сбое отдельного KPI-запроса
	// (cmdb.collectStorageMetricForSystem) поле просто оставалось нулевым
	// без единой пометки - min/avg в суточной агрегации
	// (cmdb.SaveStorageMetricRecord) искажались фантомными нулями от
	// сбойных опросов, неотличимыми от настоящего нуля IOPS/латентности.
	// PollOK=false - хотя бы один запрос метрики к СХД в этом цикле не
	// удался (см. PollError - через ";" перечислены названия несобранных
	// метрик/причина). Пустой PollError при PollOK=true для источников без
	// живого API метрик (VNX CLI) означает "опрос не выполнялся, доступна
	// только статическая UsedPercent из справочника СХД" - это НЕ ошибка.
	PollOK    bool   `form:"PollOK"`
	PollError string `form:"PollError" orm:"size(512)"`
}

func (m *StorageMetric) TableName() string { return "storage_metric" }

func (m *StorageMetric) TableUnique() [][]string {
	return [][]string{{"StorageSystemsId", "CollectedAt"}}
}

func (m *StorageMetric) TableIndex() [][]string {
	return [][]string{
		{"CollectedAt"},
		{"StorageSystemsId"},
		{"ShortName"},
	}
}

// ReplaceStorageMetrics записывает снимки за минуту, заменяя существующие.
func ReplaceStorageMetrics(ctx context.Context, rows []*StorageMetric) error {
	if len(rows) == 0 {
		return nil
	}
	o := orm.NewOrm()
	return o.DoTxWithCtx(ctx, func(ctx context.Context, tx orm.TxOrmer) error {
		qs := tx.QueryTable(new(StorageMetric))
		for _, row := range rows {
			if row.StorageSystemsId <= 0 {
				return fmt.Errorf("models: StorageMetric.StorageSystemsId is required")
			}
			if _, err := qs.Filter("StorageSystemsId", row.StorageSystemsId).
				Filter("CollectedAt", row.CollectedAt).
				Delete(); err != nil {
				return err
			}
		}
		_, err := tx.InsertMultiWithCtx(ctx, 50, rows)
		return err
	})
}
