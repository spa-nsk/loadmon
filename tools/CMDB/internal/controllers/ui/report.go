package ui

import (
	"context"
	"encoding/csv"
	"strconv"
	"time"

	"cmdb/internal/repo"
	"cmdb/internal/services/baseline"
	"cmdb/internal/services/baselineconfig"
	"cmdb/internal/services/refdata"
	"cmdb/internal/services/report"
)

// ReportController — типовые отчёты (plan.md E7.5): инвентарь по локации/стойке, ёмкость СХД, ВМ
// по кластерам, КЕ без владельца, отклонения именования. Шестой отчёт плана — «изменения за
// период» — не отдельная страница: это существующий /changes (internal/controllers/ui.AuditController)
// с диапазоном дат, на него ведёт ссылка с индекса.
type ReportController struct {
	BaseController
}

func (c *ReportController) ctx() context.Context { return c.Ctx.Request.Context() }

func (c *ReportController) must(err error) {
	if err != nil {
		c.fail(err)
	}
}

// htmlRowLimit — сколько строк показывает страница отчёта; на объёме из intent §9 (сотни тысяч
// КЕ) рендерить всё в HTML незачем — полные данные всегда доступны через CSV.
const htmlRowLimit = 500

func capRows(rows []repo.Row) (shown []repo.Row, truncated bool) {
	if len(rows) > htmlRowLimit {
		return rows[:htmlRowLimit], true
	}
	return rows, false
}

// Index — список отчётов со сводкой (для быстрой ориентации, без перехода в каждый).
func (c *ReportController) Index() {
	c.Data["Nav"] = "reports"
	c.page("report/index.tpl")
}

// exportCSV — общий экспортёр: headers/keys задают порядок и состав колонок CSV.
func (c *ReportController) exportCSV(filename string, headers, keys []string, rows []repo.Row) {
	c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", `attachment; filename="`+filename+"-"+time.Now().UTC().Format("20060102-1504")+`.csv"`)
	w := csv.NewWriter(c.Ctx.ResponseWriter)
	_ = w.Write(headers)
	rec := make([]string, len(keys))
	for _, r := range rows {
		for i, k := range keys {
			rec[i] = r[k]
		}
		_ = w.Write(rec)
	}
	w.Flush()
	c.StopRun()
}

// ---------- инвентарь по локации/стойке ----------

var (
	inventoryColumns = []string{"Имя", "Класс", "Локация", "Стойка", "Стадия ЖЦ", "Критичность", "Серийный №", "Инв. номер"}
	inventoryKeys    = []string{"name", "class_name", "location_code", "rack_position", "lifecycle_stage", "criticality", "serial_number", "asset_tag"}
)

func (c *ReportController) Inventory() {
	loc := c.GetString("location")
	summary, err := report.InventorySummary(c.ctx(), loc)
	c.must(err)
	locations, err := refdata.Locations(c.ctx())
	c.must(err)
	c.Data["Nav"] = "reports"
	c.Data["Location"] = loc
	c.Data["Locations"] = locations
	c.Data["Summary"] = summary
	c.page("report/inventory.tpl")
}

func (c *ReportController) InventoryCSV() {
	rows, err := report.InventoryRows(c.ctx(), c.GetString("location"))
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "inventory", "rows": len(rows)})
	c.exportCSV("inventory", inventoryColumns, inventoryKeys, rows)
}

// ---------- ёмкость СХД ----------

func (c *ReportController) Storage() {
	arrays, err := report.StorageArrays(c.ctx())
	c.must(err)
	pools, err := report.StoragePools(c.ctx())
	c.must(err)
	c.Data["Nav"] = "reports"
	c.Data["Arrays"] = arrays
	c.Data["Pools"] = pools
	c.Data["RawTotal"] = report.SumTB(arrays, "raw_tb")
	c.Data["UsableTotal"] = report.SumTB(arrays, "usable_tb")
	c.Data["AllocatedTotal"] = report.SumTB(arrays, "allocated_tb")
	c.page("report/storage.tpl")
}

var (
	storageArrayColumns = []string{"Массив", "Семейство", "Raw, ТБ", "Usable, ТБ", "Размещено, ТБ"}
	storageArrayKeys    = []string{"name", "array_family", "raw_tb", "usable_tb", "allocated_tb"}
)

func (c *ReportController) StorageCSV() {
	rows, err := report.StorageArrays(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "storage", "rows": len(rows)})
	c.exportCSV("storage", storageArrayColumns, storageArrayKeys, rows)
}

// ---------- ВМ по кластерам ----------

func (c *ReportController) VMClusters() {
	clusterID, _ := strconv.ParseInt(c.GetString("cluster"), 10, 64)
	rows, err := report.VMsByCluster(c.ctx(), clusterID)
	c.must(err)
	clusters, err := refdata.Clusters(c.ctx())
	c.must(err)
	shown, truncated := capRows(rows)
	c.Data["Nav"] = "reports"
	c.Data["ClusterID"] = clusterID
	c.Data["Clusters"] = clusters
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.page("report/vm_clusters.tpl")
}

var (
	vmClusterColumns = []string{"ВМ", "Кластер", "Хост", "vCPU", "ОЗУ, ГБ", "Гостевая ОС", "Стадия ЖЦ"}
	vmClusterKeys    = []string{"name", "cluster_name", "host_name", "vcpu", "ram_gb", "guest_os", "lifecycle_stage"}
)

func (c *ReportController) VMClustersCSV() {
	clusterID, _ := strconv.ParseInt(c.GetString("cluster"), 10, 64)
	rows, err := report.VMsByCluster(c.ctx(), clusterID)
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "vm_clusters", "rows": len(rows)})
	c.exportCSV("vm-clusters", vmClusterColumns, vmClusterKeys, rows)
}

// ---------- КЕ без владельца ----------

var (
	noOwnerColumns = []string{"Имя", "Класс", "Локация", "Компания", "Критичность", "Стадия ЖЦ"}
	noOwnerKeys    = []string{"name", "class_name", "location_code", "company_code", "criticality", "lifecycle_stage"}
)

func (c *ReportController) NoOwner() {
	rows, err := report.WithoutOwner(c.ctx())
	c.must(err)
	shown, truncated := capRows(rows)
	c.Data["Nav"] = "reports"
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.page("report/no_owner.tpl")
}

func (c *ReportController) NoOwnerCSV() {
	rows, err := report.WithoutOwner(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "no_owner", "rows": len(rows)})
	c.exportCSV("no-owner", noOwnerColumns, noOwnerKeys, rows)
}

// ---------- отклонения именования ----------

var (
	namingColumns = []string{"Имя", "Класс", "Статус", "Локация", "Исходное имя источника"}
	namingKeys    = []string{"name", "class_name", "naming_status", "location_code", "source_name"}
)

func (c *ReportController) Naming() {
	rows, err := report.NamingDeviations(c.ctx())
	c.must(err)
	shown, truncated := capRows(rows)
	c.Data["Nav"] = "reports"
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.page("report/naming.tpl")
}

func (c *ReportController) NamingCSV() {
	rows, err := report.NamingDeviations(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "naming", "rows": len(rows)})
	c.exportCSV("naming-deviations", namingColumns, namingKeys, rows)
}

// ---------- запрещённое/неразрешённое ПО (E7.7) ----------

var (
	softwareColumns = []string{"КЕ", "Локация", "Производитель", "Продукт", "Версия", "Политика", "Путь установки"}
	softwareKeys    = []string{"name", "location_code", "vendor", "product", "version", "policy", "install_path"}
)

func (c *ReportController) Software() {
	rows, err := report.ForbiddenSoftware(c.ctx())
	c.must(err)
	shown, truncated := capRows(rows)
	c.Data["Nav"] = "reports"
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.page("report/software.tpl")
}

func (c *ReportController) SoftwareCSV() {
	rows, err := report.ForbiddenSoftware(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "software", "rows": len(rows)})
	c.exportCSV("forbidden-software", softwareColumns, softwareKeys, rows)
}

// ---------- отклонения от эталонных конфигураций (E7.8) ----------

// BaselineConfig — отчёт отклонений: по действующим КЕ (?asof= из общего фильтра, как везде) либо
// по составу конкретного замороженного/утверждённого среза (?baseline=<id> — «связь с
// baseline-срезами» из плана).
func (c *ReportController) BaselineConfig() {
	baselineID, _ := strconv.ParseInt(c.GetString("baseline"), 10, 64)
	rows, err := baselineconfig.Deviations(c.ctx(), baselineID)
	c.must(err)
	shown := rows
	truncated := false
	if len(rows) > htmlRowLimit {
		shown, truncated = rows[:htmlRowLimit], true
	}
	baselines, err := baseline.List(c.ctx())
	c.must(err)
	var frozen []baseline.Baseline
	for _, b := range baselines {
		if b.Status != "draft" {
			frozen = append(frozen, b)
		}
	}
	c.Data["Nav"] = "reports"
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.Data["BaselineID"] = baselineID
	c.Data["Baselines"] = frozen
	c.page("report/baseline_config.tpl")
}

var (
	baselineConfigColumns = []string{"Конфигурация", "КЕ", "Тип", "Поле", "Требуется", "Фактически"}
)

func (c *ReportController) BaselineConfigCSV() {
	baselineID, _ := strconv.ParseInt(c.GetString("baseline"), 10, 64)
	rows, err := baselineconfig.Deviations(c.ctx(), baselineID)
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "baseline_config", "rows": len(rows), "baseline_id": baselineID})

	c.Ctx.Output.Header("Content-Type", "text/csv; charset=utf-8")
	c.Ctx.Output.Header("Content-Disposition", `attachment; filename="baseline-config-deviations-`+time.Now().UTC().Format("20060102-1504")+`.csv"`)
	w := csv.NewWriter(c.Ctx.ResponseWriter)
	_ = w.Write(baselineConfigColumns)
	for _, r := range rows {
		_ = w.Write([]string{r.ConfigName, r.CIName, r.Kind, r.Field, r.Expected, r.Actual})
	}
	w.Flush()
	c.StopRun()
}

// ---------- отчёты соответствия (E7.9) ----------

var (
	staleInventoryColumns = []string{"Имя", "Класс", "Локация", "Последнее подтверждение (UTC)", "Дней без подтверждения"}
	staleInventoryKeys    = []string{"name", "class_name", "location_code", "last_seen_at", "days_stale"}
)

// StaleInventory — КЕ без подтверждения источником > 30 дней (intent §9 KPI).
func (c *ReportController) StaleInventory() {
	rows, err := report.StaleInventory(c.ctx())
	c.must(err)
	shown, truncated := capRows(rows)
	c.Data["Nav"] = "reports"
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.page("report/stale_inventory.tpl")
}

func (c *ReportController) StaleInventoryCSV() {
	rows, err := report.StaleInventory(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "stale_inventory", "rows": len(rows)})
	c.exportCSV("stale-inventory", staleInventoryColumns, staleInventoryKeys, rows)
}

var (
	kiiColumns = []string{"Объект КИИ", "Категория", "Класс объекта", "Состав: КЕ", "Состав: класс", "Тип связи"}
	kiiKeys    = []string{"kii_name", "kii_category", "kii_class_name", "member_name", "member_class_name", "rel_type"}
)

// KII — объекты КИИ и их состав (152-ФЗ).
func (c *ReportController) KII() {
	rows, err := report.KIIComposition(c.ctx())
	c.must(err)
	shown, truncated := capRows(rows)
	c.Data["Nav"] = "reports"
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.page("report/kii.tpl")
}

func (c *ReportController) KIICSV() {
	rows, err := report.KIIComposition(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "kii", "rows": len(rows)})
	c.exportCSV("kii-objects", kiiColumns, kiiKeys, rows)
}

var (
	domesticHWColumns = []string{"Производитель", "Модель", "Отечественная", "Реестр", "КЕ"}
	domesticHWKeys    = []string{"manufacturer", "model_name", "is_domestic", "registry_ref", "n"}
	domesticSWColumns = []string{"Вендор", "Продукт", "Версия", "Отечественное", "Реестр", "Установок"}
	domesticSWKeys    = []string{"vendor", "product", "version", "is_domestic", "registry_ref", "n"}
)

// Domestic — доля отечественного оборудования и ПО (intent §9/§7.6).
func (c *ReportController) Domestic() {
	hw, err := report.DomesticHardware(c.ctx())
	c.must(err)
	sw, err := report.DomesticSoftware(c.ctx())
	c.must(err)
	hwSummary, err := report.DomesticHardwareSummary(c.ctx())
	c.must(err)
	swSummary, err := report.DomesticSoftwareSummary(c.ctx())
	c.must(err)
	c.Data["Nav"] = "reports"
	c.Data["Hardware"] = hw
	c.Data["Software"] = sw
	c.Data["HWSummary"] = hwSummary
	c.Data["SWSummary"] = swSummary
	c.page("report/domestic.tpl")
}

func (c *ReportController) DomesticHardwareCSV() {
	rows, err := report.DomesticHardware(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "domestic_hardware", "rows": len(rows)})
	c.exportCSV("domestic-hardware", domesticHWColumns, domesticHWKeys, rows)
}

func (c *ReportController) DomesticSoftwareCSV() {
	rows, err := report.DomesticSoftware(c.ctx())
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "domestic_software", "rows": len(rows)})
	c.exportCSV("domestic-software", domesticSWColumns, domesticSWKeys, rows)
}

var (
	legacyNamesColumns = []string{"Текущее имя", "Класс", "Локация", "Предложенное имя", "Исходное имя источника"}
	legacyNamesKeys    = []string{"name", "class_name", "location_code", "suggested_name", "source_name"}
)

// LegacyNames — legacy-имена (профиль GUIDE-2019) с предложением нового имени по CMDB-NC-2026.
func (c *ReportController) LegacyNames() {
	rows, err := report.LegacyNames(c.ctx(), registry)
	c.must(err)
	shown, truncated := capRows(rows)
	c.Data["Nav"] = "reports"
	c.Data["Rows"] = shown
	c.Data["Total"] = len(rows)
	c.Data["Truncated"] = truncated
	c.page("report/legacy_names.tpl")
}

func (c *ReportController) LegacyNamesCSV() {
	rows, err := report.LegacyNames(c.ctx(), registry)
	c.must(err)
	c.logEvent("report_export", map[string]any{"report": "legacy_names", "rows": len(rows)})
	c.exportCSV("legacy-names", legacyNamesColumns, legacyNamesKeys, rows)
}
