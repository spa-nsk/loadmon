package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"sort"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Пороговые значения индикаторов утилизации по ЦОД - ОТДЕЛЬНЫЕ от
// utilOverloadCpu/utilUnderloadCpu и т.п. в controllers/sprutilization.go
// (та страница про риски "перегруз/недоиспользование/норма" - три состояния
// с другими границами), заданы дословно по прямому запросу пользователя
// (2026-09-17, "сделай отчет по утилизации виртуальной инфраструктуры по
// ЦОД"): "красными при превышении 95% оранжевыми при 75-94% зеленым от 5-74%
// всё меньше голубым". Четыре уровня, а не три - у этого отчёта.
const (
	utilDcCritPct = 95.0 // % и выше - красный (критический перегруз)
	utilDcWarnPct = 75.0 // % и выше (но ниже utilDcCritPct) - оранжевый
	utilDcLowPct  = 5.0  // % и выше (но ниже utilDcWarnPct) - зелёный; ниже - голубой ("простаивает")
)

// utilDcTier возвращает код цветового уровня индикатора по проценту
// утилизации - см. константы выше. Общий для CPU/RAM/СХД на этой странице.
func utilDcTier(pct float64) string {
	switch {
	case pct >= utilDcCritPct:
		return "crit"
	case pct >= utilDcWarnPct:
		return "warn"
	case pct >= utilDcLowPct:
		return "ok"
	default:
		return "low"
	}
}

// utilDcCluster - утилизация одного кластера (строка внутри вкладки ЦОД).
// Объём/использование CPU - в МГц (произведение среднего МГц на ядро на
// число ядер кластера, см. models.ClusterRecord.CpuMhzAvg/AllNumCpuCores -
// тот же источник данных, что и на /sprsrv/utilization), RAM/СХД - в Гб.
type utilDcCluster struct {
	Name      string `json:"name"`
	Vcenter   string `json:"vcenter"`
	HostCount int    `json:"hostCount"`
	VMCount   int    `json:"vmCount"`

	CpuTotalMhz float64 `json:"cpuTotalMhz"`
	CpuUsedMhz  float64 `json:"cpuUsedMhz"`
	CpuPct      float64 `json:"cpuPct"`

	RamTotalGb float64 `json:"ramTotalGb"`
	RamUsedGb  float64 `json:"ramUsedGb"`
	RamPct     float64 `json:"ramPct"`

	StorageTotalGb float64 `json:"storageTotalGb"`
	StorageUsedGb  float64 `json:"storageUsedGb"`
	StoragePct     float64 `json:"storagePct"`
}

// utilDcSummary - агрегат по одному ЦОД (сумма по всем его кластерам, %
// пересчитан от суммарных total/used, а не как среднее процентов - иначе
// маленький кластер исказил бы общий показатель ЦОД наравне с большим).
type utilDcSummary struct {
	Name         string `json:"name"`
	ClusterCount int    `json:"clusterCount"`
	HostCount    int    `json:"hostCount"`
	VMCount      int    `json:"vmCount"`

	CpuTotalMhz float64 `json:"cpuTotalMhz"`
	CpuUsedMhz  float64 `json:"cpuUsedMhz"`
	CpuPct      float64 `json:"cpuPct"`

	RamTotalGb float64 `json:"ramTotalGb"`
	RamUsedGb  float64 `json:"ramUsedGb"`
	RamPct     float64 `json:"ramPct"`

	StorageTotalGb float64 `json:"storageTotalGb"`
	StorageUsedGb  float64 `json:"storageUsedGb"`
	StoragePct     float64 `json:"storagePct"`

	Clusters []utilDcCluster `json:"clusters"`
}

// SprUtilizationDatacenter строит отчёт по утилизации виртуальной
// инфраструктуры В РАЗРЕЗЕ ЦОД (физически/логически разделённых площадок
// vCenter Datacenter) - по прямому запросу пользователя (2026-09-17):
// отдельная вкладка на каждый ЦОД, где есть гипервизоры, с canvas-
// индикаторами загрузки CPU/RAM/СХД по каждому кластеру этого ЦОД
// (с указанием общего объёма ресурса и фактической утилизации), плюс
// сводная вкладка сравнения ЦОД между собой.
//
// "ЦОД" здесь - НЕ физическая площадка/стойка (models.DatacenterSite,
// справочник шкафов/размещения оборудования), а логический объект
// "Datacenter" из инвентаря vCenter (vCenter → Datacenter → Cluster → Host),
// который уже собирается и хранится в models.ClusterRecord.Datacenter/
// models.HostRecord.Datacenter (см. cmdb/savedata.go,
// cmdb/savemetric_orchestration.go - запись ClusterRecord вообще не
// создаётся, если Datacenter не определён, значение "нет данных" туда не
// попадает). Этот выбор объекта группировки - осознанный: "ЦОД, где есть
// гипервизоры" однозначно указывает на логическую группировку хостов
// ESXi/гипервизоров в инвентаре vCenter, а не на физическое размещение в
// стойках (которое к тому же не для всех хостов заполнено и относится к
// другому смыслу "ЦОД" в этом проекте, см. models.DatacenterSite).
//
// СХД (StorageSystems/StorageVolumes) не привязана к этому логическому ЦОД
// напрямую пофизически - вместо отдельного join сюда идут уже готовые
// агрегаты "объём датастора, видимого кластеру" из ClusterRecord.
// AllCapacityGb/AllFreeSpaceGb/AllPercentUsed (тот же источник, что даёт
// "Cluster DISK (Gb)" на /sprsrv/utilization) - это ёмкость СХД, реально
// потребляемая ВИРТУАЛЬНОЙ инфраструктурой этого кластера, что точнее
// отвечает формулировке запроса "утилизация ВИРТУАЛЬНОЙ инфраструктуры",
// чем сумма по всем LUN конкретной физической СХД (которая может отдавать
// LUN сразу в несколько ЦОД/кластеров).
//
// Маршрут: /sprsrv/utildatacenter.
func (this *MainController) SprUtilizationDatacenter() {

	RegisterRequest(this, LogTypeViewLoadTable) // просмотр отчёта утилизации виртуальной инфраструктуры по ЦОД

	this.activeContent("sprsrv/utildatacenter")

	this.Data["DcDataJSON"] = template.JS("{}")
	this.Data["DcNamesJSON"] = template.JS("[]")
	this.Data["DcCompareJSON"] = template.JS("[]")

	o := orm.NewOrm()

	since := time.Now().AddDate(0, 0, -utilLookbackDays)

	// --- Кластеры: последняя суточная запись ClusterRecord за окно (тот же
	// приём, что в SprUtilization) ---
	var clusterRecs []models.ClusterRecord
	o.QueryTable(new(models.ClusterRecord)).Filter("Date__gte", since).OrderBy("ClusterId", "-Date").All(&clusterRecs)
	latestClusterByKey := latestByGroup(clusterRecs, func(r models.ClusterRecord) int64 { return r.ClusterId }, func(r models.ClusterRecord) time.Time { return r.Date })

	// --- Хосты/ВМ - нужны только для счётчиков "хостов"/"VM" по кластеру,
	// сами метрики загрузки берутся из ClusterRecord (уже агрегированы) ---
	var hostRecs []models.HostRecord
	o.QueryTable(new(models.HostRecord)).Filter("Date__gte", since).OrderBy("ServerId", "-Date").All(&hostRecs)
	latestHostByServer := latestByGroup(hostRecs, func(r models.HostRecord) int64 { return r.ServerId }, func(r models.HostRecord) time.Time { return r.Date })

	var vmRecs []models.VMRecord
	o.QueryTable(new(models.VMRecord)).Filter("Date__gte", since).OrderBy("VMSprId", "-Date").All(&vmRecs)
	latestVMBySpr := latestByGroup(vmRecs, func(r models.VMRecord) int64 { return r.VMSprId }, func(r models.VMRecord) time.Time { return r.Date })

	hostCountByCluster := map[int64]int{}
	for _, h := range latestHostByServer {
		hostCountByCluster[h.ClusterId]++
	}
	vmCountByCluster := map[int64]int{}
	for _, v := range latestVMBySpr {
		vmCountByCluster[v.ClusterId]++
	}

	// --- ЭТАП 1 (по прямому требованию пользователя - "первым этапом
	// определи список таких ЦОД"): список ЦОД, где есть хотя бы один
	// кластер/гипервизор с данными за период, отсортирован по имени. ---
	dcByName := map[string]*utilDcSummary{}
	var dcNames []string
	for _, r := range latestClusterByKey {
		dc := r.Datacenter
		if dc == "" {
			continue
		}
		sum, ok := dcByName[dc]
		if !ok {
			sum = &utilDcSummary{Name: dc}
			dcByName[dc] = sum
			dcNames = append(dcNames, dc)
		}

		cpuTotal := float64(r.CpuMhzAvg) * float64(r.AllNumCpuCores)
		cpuUsed := cpuTotal * r.Cpu_usage_average / 100
		ramTotal := r.AllMemorySizeGB
		ramUsed := ramTotal * r.Mem_usage_average / 100
		storageTotal := r.AllCapacityGb
		storageUsed := storageTotal - r.AllFreeSpaceGb
		if storageUsed < 0 {
			storageUsed = 0
		}

		cl := utilDcCluster{
			Name:      r.Name,
			Vcenter:   r.Vcenter,
			HostCount: hostCountByCluster[r.ClusterId],
			VMCount:   vmCountByCluster[r.ClusterId],

			CpuTotalMhz: cpuTotal,
			CpuUsedMhz:  cpuUsed,
			CpuPct:      r.Cpu_usage_average,

			RamTotalGb: ramTotal,
			RamUsedGb:  ramUsed,
			RamPct:     r.Mem_usage_average,

			StorageTotalGb: storageTotal,
			StorageUsedGb:  storageUsed,
			StoragePct:     r.AllPercentUsed,
		}
		sum.Clusters = append(sum.Clusters, cl)
		sum.ClusterCount++
		sum.HostCount += cl.HostCount
		sum.VMCount += cl.VMCount
		sum.CpuTotalMhz += cpuTotal
		sum.CpuUsedMhz += cpuUsed
		sum.RamTotalGb += ramTotal
		sum.RamUsedGb += ramUsed
		sum.StorageTotalGb += storageTotal
		sum.StorageUsedGb += storageUsed
	}

	sort.Strings(dcNames)

	for _, sum := range dcByName {
		sum.CpuPct = pctOf(sum.CpuUsedMhz, sum.CpuTotalMhz)
		sum.RamPct = pctOf(sum.RamUsedGb, sum.RamTotalGb)
		sum.StoragePct = pctOf(sum.StorageUsedGb, sum.StorageTotalGb)
		sort.Slice(sum.Clusters, func(i, j int) bool { return sum.Clusters[i].CpuPct > sum.Clusters[j].CpuPct })
	}

	if b, err := json.Marshal(dcByName); err == nil {
		this.Data["DcDataJSON"] = template.JS(b)
	}
	if b, err := json.Marshal(dcNames); err == nil {
		this.Data["DcNamesJSON"] = template.JS(b)
	}

	// --- Сводная вкладка сравнения: те же итоги по ЦОД одним плоским
	// списком в порядке dcNames - шаблон строит сравнительные диаграммы и
	// таблицу разниц (максимум-минимум по каждому ресурсу) на клиенте. ---
	compare := make([]*utilDcSummary, 0, len(dcNames))
	for _, name := range dcNames {
		compare = append(compare, dcByName[name])
	}
	if b, err := json.Marshal(compare); err == nil {
		this.Data["DcCompareJSON"] = template.JS(b)
	}

	this.Data["DcCount"] = len(dcNames)
}

// pctOf - used/total*100 с защитой от деления на 0 (когда у ЦОД/кластера
// почему-то нулевая суммарная ёмкость ресурса - например, СХД не отдала
// AllCapacityGb).
func pctOf(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return used / total * 100
}
