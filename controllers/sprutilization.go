package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"sort"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Пороговые значения (эвристики) для оценки утилизации и рисков перегруза/
// недоиспользования. Значения не подтверждены заказчиком - выбраны как
// разумные умолчания по общепринятой практике мониторинга виртуальной
// инфраструктуры (vSphere/СХД). Если потребуется - вынести в конфигурацию.
const (
	utilOverloadCpu      = 80.0 // % - кластер/хост считается перегруженным по CPU
	utilOverloadMem      = 85.0 // % - кластер/хост считается перегруженным по памяти
	utilUnderloadCpu     = 15.0 // % - кластер/хост считается недоиспользуемым по CPU
	utilUnderloadMem     = 20.0 // % - кластер/хост считается недоиспользуемым по памяти
	utilVMOverloadCpu    = 85.0 // % - VM считается перегруженной по CPU (средняя загрузка)
	utilVMIdleCpuMax     = 5.0  // % - VM считается "простаивающей" если максимум CPU за сутки ниже этого значения
	utilStorageOverload  = 90.0 // % - СХД/LUN считается перегруженной по занятому месту
	utilStorageUnderload = 10.0 // % - СХД/LUN считается недоиспользуемой по занятому месту
	utilStorageMinGbSize = 100.0 // Gb - минимальный объём, чтобы недоиспользование СХД/LUN считалось значимым
	utilLookbackDays     = 7    // сколько дней в прошлое просматривать в поиске "последней" суточной записи по сущности
)

// utilKPI - блок обобщённых показателей утилизации (карточки в верхней части страницы).
type utilKPI struct {
	ClusterCount        int
	HostCount           int
	VMCount             int
	StorageSystemCount  int
	LUNCount            int
	AvgClusterCpu       float64
	AvgClusterMem       float64
	AvgHostCpu          float64
	AvgHostMem          float64
	AvgVMCpu            float64
	AvgVMMem            float64
	AvgStorageUsedPct   float64
	OverloadedCount     int
	UnderusedCount      int
	TotalCpuMhz         float64
	TotalMemGB          float64
	TotalStorageRawTB   float64
}

// utilEntity - строка таблицы/точка диаграммы по одной сущности (кластер/хост/VM/СХД/LUN).
type utilEntity struct {
	Name    string  `json:"name"`
	Parent  string  `json:"parent"` // для VM/хоста - имя кластера, для LUN - имя СХД
	CpuPct  float64 `json:"cpuPct"`
	MemPct  float64 `json:"memPct"`
	DiskPct float64 `json:"diskPct"`
	SizeGb  float64 `json:"sizeGb"`
	Status  string  `json:"status"` // "overload" | "underused" | "normal"
}

// utilRisk - строка списка рисков/рекомендаций.
type utilRisk struct {
	Category       string  `json:"category"` // "Кластер"/"Хост"/"VM"/"СХД"/"LUN"
	Object         string  `json:"object"`
	Metric         string  `json:"metric"`
	Value          float64 `json:"value"`
	Severity       string  `json:"severity"` // "danger" (перегруз) | "warning" (недоиспользование)
	Recommendation string  `json:"recommendation"`
}

// SprUtilization строит интерактивную стр��ницу оценки утилизации виртуальной
// инфраструктуры: сводные KPI, структуру связанного оборудования (vCenter/
// кластер/хост/VM и СХД/LUN), диаграммы загрузки и таблицу рисков и
// рекомендаций по оптимизации.
//
// Источники данных - персистентные суточные агрегаты, которые уже строит
// cmdb.SaveMetricData() (см. cmdb/savemetric_orchestration.go): models.ClusterRecord,
// models.HostRecord, models.VMRecord (по одной строке на сущность за сутки,
// см. kontext/03-data-model.md) - для "текущего состояния" берётся самая
// свежая строка за последние utilLookbackDays суток по каждой сущности
// (выборка ORM по окну времени + группировка в Go, без raw SQL DISTINCT ON,
// т.к. драйвер БД - PostgreSQL, но код должен оставаться портируемым, см.
// config/dbinit.go). models.StorageSystems/models.StorageVolumes (СХД/LUN) -
// таблицы текущего состояния без суточной истории, читаются целиком.
//
// Маршрут: /sprsrv/utilization.
func (this *MainController) SprUtilization() {

	RegisterRequest(this, LogTypeViewLoadTable) // просмотр сводной аналитики утилизации виртуальной инфраструктуры

	this.activeContent("sprsrv/utilization")

	this.Data["ChartsJSON"] = template.JS("[]")
	this.Data["EntitiesJSON"] = template.JS("[]")
	this.Data["RisksJSON"] = template.JS("[]")
	this.Data["DataGapsJSON"] = template.JS("[]")

	o := orm.NewOrm()

	since := time.Now().AddDate(0, 0, -utilLookbackDays)

	// --- Кластеры: активные записи справочника + последняя суточная запись ClusterRecord ---
	var clusterSpr []models.ClusterSprRecords
	o.QueryTable(new(models.ClusterSprRecords)).Filter("Stat", 1).All(&clusterSpr)

	var clusterRecs []models.ClusterRecord
	o.QueryTable(new(models.ClusterRecord)).Filter("Date__gte", since).OrderBy("ClusterId", "-Date").All(&clusterRecs)
	latestClusterByKey := latestByGroup(clusterRecs, func(r models.ClusterRecord) int64 { return r.ClusterId }, func(r models.ClusterRecord) time.Time { return r.Date })
	clusterNameByClusterId := make(map[int64]string)
	for id, r := range latestClusterByKey {
		clusterNameByClusterId[id] = r.Name
	}

	// --- Хосты: последняя суточная запись HostRecord ---
	var hostRecs []models.HostRecord
	o.QueryTable(new(models.HostRecord)).Filter("Date__gte", since).OrderBy("ServerId", "-Date").All(&hostRecs)
	latestHostByServer := latestByGroup(hostRecs, func(r models.HostRecord) int64 { return r.ServerId }, func(r models.HostRecord) time.Time { return r.Date })

	// --- ВМ: последняя суточная запись VMRecord ---
	var vmRecs []models.VMRecord
	o.QueryTable(new(models.VMRecord)).Filter("Date__gte", since).OrderBy("VMSprId", "-Date").All(&vmRecs)
	latestVMBySpr := latestByGroup(vmRecs, func(r models.VMRecord) int64 { return r.VMSprId }, func(r models.VMRecord) time.Time { return r.Date })

	// Имена ВМ (VMRecord не хранит имя) - подтягиваем из справочника vm_spr_record.
	vmNameBySpr := make(map[int64]string)
	if len(latestVMBySpr) > 0 {
		var vmSprIds []int64
		for id := range latestVMBySpr {
			vmSprIds = append(vmSprIds, id)
		}
		var vmSprs []models.VMSprRecord
		o.QueryTable(new(models.VMSprRecord)).Filter("Id__in", vmSprIds).All(&vmSprs)
		for _, v := range vmSprs {
			vmNameBySpr[v.Id] = v.Vm_Name
		}
	}

	// --- СХД и LUN - текущее состояние, суточной истории не ведётся ---
	// Из общего количества строк в справочниках исключаем заведомо неактуальные
	// записи, иначе счётчики "СХД"/"LUN" на странице выглядят завышенными за
	// счёт демонтированного оборудования и удалённых томов:
	//  - StorageSystems.Status: 0-не указано 1-используется 2-выключен
	//    3-не используется 4-тестирование 5-приемка 6-демонтирован
	//    (исключаем только 6 - демонтированное оборудование физически убрано)
	//  - StorageVolumes.Status: 0-том выведен из работы или удалён
	//    1-том в виртуализации 2-все остальные случаи (исключаем только 0)
	var storagesAll []models.StorageSystems
	o.QueryTable(new(models.StorageSystems)).All(&storagesAll)
	var storages []models.StorageSystems
	for _, s := range storagesAll {
		if s.Status == 6 {
			continue
		}
		storages = append(storages, s)
	}
	var volumesAll []models.StorageVolumes
	o.QueryTable(new(models.StorageVolumes)).All(&volumesAll)
	var volumes []models.StorageVolumes
	for _, v := range volumesAll {
		if v.Status == 0 {
			continue
		}
		volumes = append(volumes, v)
	}

	kpi := utilKPI{
		ClusterCount:       len(latestClusterByKey),
		HostCount:          len(latestHostByServer),
		VMCount:            len(latestVMBySpr),
		StorageSystemCount: len(storages),
		LUNCount:           len(volumes),
	}

	var clusterEntities, hostEntities, vmEntities, storageEntities, lunEntities []utilEntity
	var risks []utilRisk

	// Кластеры
	{
		var cpuSum, memSum float64
		for _, r := range latestClusterByKey {
			status := classify(r.Cpu_usage_average, r.Mem_usage_average, utilOverloadCpu, utilOverloadMem, utilUnderloadCpu, utilUnderloadMem)
			clusterEntities = append(clusterEntities, utilEntity{
				Name: r.Name, Parent: r.Vcenter,
				CpuPct: r.Cpu_usage_average, MemPct: r.Mem_usage_average, DiskPct: r.AllPercentUsed,
				SizeGb: r.AllCapacityGb, Status: status,
			})
			cpuSum += r.Cpu_usage_average
			memSum += r.Mem_usage_average
			kpi.TotalCpuMhz += float64(r.CpuMhzAvg)
			kpi.TotalMemGB += r.AllMemorySizeGB
			addRisk(&risks, "Кластер", r.Name, r.Cpu_usage_average, r.Mem_usage_average, status,
				"Перераспределить VM на менее загруженные кластеры или добавить вычислительные ресурсы (CPU/RAM)",
				"Кластер недоиспользуется - рассмотреть консолидацию нагрузки с других кластеров или сокращение выделенных ресурсов")
		}
		if kpi.ClusterCount > 0 {
			kpi.AvgClusterCpu = cpuSum / float64(kpi.ClusterCount)
			kpi.AvgClusterMem = memSum / float64(kpi.ClusterCount)
		}
	}

	// Хосты
	{
		var cpuSum, memSum float64
		for _, r := range latestHostByServer {
			status := classify(r.Cpu_usage_average, r.Mem_usage_average, utilOverloadCpu, utilOverloadMem, utilUnderloadCpu, utilUnderloadMem)
			hostEntities = append(hostEntities, utilEntity{
				Name: r.Name, Parent: r.Cluster,
				CpuPct: r.Cpu_usage_average, MemPct: r.Mem_usage_average, DiskPct: r.Disk_usage_average,
				SizeGb: r.MemorySizeGB, Status: status,
			})
			cpuSum += r.Cpu_usage_average
			memSum += r.Mem_usage_average
			addRisk(&risks, "Хост", r.Name, r.Cpu_usage_average, r.Mem_usage_average, status,
				"Перенести часть VM на другой хост (vMotion) или добавить ресурсы хосту",
				"Хост недоиспользуется - кандидат на консолидацию VM с других хостов или вывод в резерв")
		}
		if kpi.HostCount > 0 {
			kpi.AvgHostCpu = cpuSum / float64(kpi.HostCount)
			kpi.AvgHostMem = memSum / float64(kpi.HostCount)
		}
	}

	// Виртуальные машины
	{
		var cpuSum, memSum float64
		for sprId, r := range latestVMBySpr {
			name := vmNameBySpr[sprId]
			if name == "" {
				name = r.Vm_IpAddress
			}
			status := ""
			if r.Cpu_usage_average >= utilVMOverloadCpu {
				status = "overload"
			} else if r.Cpu_usage_maximum < utilVMIdleCpuMax && (r.Vm_NumCores >= 2 || r.Vm_MemorySizeGB >= 4) {
				status = "underused"
			}
			vmEntities = append(vmEntities, utilEntity{
				Name: name, Parent: clusterNameByClusterId[r.ClusterId],
				CpuPct: r.Cpu_usage_average, MemPct: r.Mem_usage_average, DiskPct: r.Vm_PercentUsed,
				SizeGb: r.Vm_CapacityGb, Status: status,
			})
			cpuSum += r.Cpu_usage_average
			memSum += r.Mem_usage_average
			if status == "overload" {
				risks = append(risks, utilRisk{Category: "VM", Object: name, Metric: "CPU avg %", Value: r.Cpu_usage_average, Severity: "danger",
					Recommendation: "Увеличить vCPU/ресурсы VM либо оптимизировать нагружающее приложение"})
			} else if status == "underused" {
				risks = append(risks, utilRisk{Category: "VM", Object: name, Metric: "CPU max %", Value: r.Cpu_usage_maximum, Severity: "warning",
					Recommendation: "VM практически простаивает (CPU max < 5% при " + itoaCores(r.Vm_NumCores) + " vCPU / " + ftoa(r.Vm_MemorySizeGB) + " ГБ RAM) - кандидат на уменьшение ресурсов (right-sizing) или отключение"})
			}
		}
		if kpi.VMCount > 0 {
			kpi.AvgVMCpu = cpuSum / float64(kpi.VMCount)
			kpi.AvgVMMem = memSum / float64(kpi.VMCount)
		}
	}

	// СХД
	{
		var usedSum float64
		for _, s := range storages {
			status := ""
			if s.UsedPercent >= utilStorageOverload {
				status = "overload"
			} else if s.UsedPercent <= utilStorageUnderload && s.TotalSizeRAW >= utilStorageMinGbSize {
				status = "underused"
			}
			storageEntities = append(storageEntities, utilEntity{
				Name: s.ShortName, Parent: s.Vendor,
				DiskPct: s.UsedPercent, SizeGb: s.TotalSizeRAW, Status: status,
			})
			usedSum += s.UsedPercent
			kpi.TotalStorageRawTB += s.TotalSizeRAW / 1024.0
			if status == "overload" {
				risks = append(risks, utilRisk{Category: "СХД", Object: s.ShortName, Metric: "Занято %", Value: s.UsedPercent, Severity: "danger",
					Recommendation: "СХД близка к заполнению - высвободить место (удалить неиспользуемые LUN/снапшоты) или расширить объём"})
			} else if status == "underused" {
				risks = append(risks, utilRisk{Category: "СХД", Object: s.ShortName, Metric: "Занято %", Value: s.UsedPercent, Severity: "warning",
					Recommendation: "СХД используется менее чем на 10% - рассмотреть консолидацию LUN на другую систему хранения"})
			}
		}
		if len(storages) > 0 {
			kpi.AvgStorageUsedPct = usedSum / float64(len(storages))
		}
	}

	// LUN
	for _, v := range volumes {
		status := ""
		if v.PercentUsed >= utilStorageOverload {
			status = "overload"
		} else if v.PercentUsed <= utilStorageUnderload && v.CapacityGb >= utilStorageMinGbSize {
			status = "underused"
		}
		lunEntities = append(lunEntities, utilEntity{
			Name: v.ShortName, Parent: v.StorageName,
			DiskPct: v.PercentUsed, SizeGb: v.CapacityGb, Status: status,
		})
		if status == "overload" {
			risks = append(risks, utilRisk{Category: "LUN", Object: v.ShortName, Metric: "Занято %", Value: v.PercentUsed, Severity: "danger",
				Recommendation: "LUN близок к заполнению - расширить том или очистить неиспользуемые данные"})
		} else if status == "underused" {
			risks = append(risks, utilRisk{Category: "LUN", Object: v.ShortName, Metric: "Занято %", Value: v.PercentUsed, Severity: "warning",
				Recommendation: "LUN используется менее чем на 10% при объёме " + ftoa(v.CapacityGb) + " ГБ - кандидат на сокращение или переиспользование"})
		}
	}

	for _, r := range risks {
		if r.Severity == "danger" {
			kpi.OverloadedCount++
		} else {
			kpi.UnderusedCount++
		}
	}

	sort.Slice(risks, func(i, j int) bool {
		if risks[i].Severity != risks[j].Severity {
			return risks[i].Severity == "danger" // сначала перегрузы, потом недоиспользование
		}
		return risks[i].Value > risks[j].Value
	})

	// Предложения по дополнительным методам сбора данных - выводятся на странице,
	// если данных недостаточно для полноценной оценки (например, справочники/
	// суточные агрегаты пустые). Список фиксированный - отражает известные
	// пробелы модели данных (см. kontext/03-data-model.md).
	var dataGaps []string
	if kpi.ClusterCount == 0 && kpi.HostCount == 0 && kpi.VMCount == 0 {
		dataGaps = append(dataGaps, "Не найдено ни одной суточной записи ClusterRecord/HostRecord/VMRecord за последние "+itoaCores(int32(utilLookbackDays))+" дней - проверьте, что cmdb.SaveMetricData() выполняется по расписанию")
	} else if kpi.VMCount == 0 {
		// Кластеры/хосты есть, а VM - нет: см. cmdb/savevmrecord.go:SaveVMRecord -
		// строка with "if rec.Cpu_usage_maximum == 0.0 && rec.Mem_usage_maximum == 0.0 { continue }"
		// молча пропускает запись vm_record, если импортированные из vspheremetrics
		// метрики CPU/MEM у VM нулевые (например, пуст vm.Perf) - в отличие от
		// SaveHostRecord/SaveClusterRecord, которые пишут запись безусловно.
		dataGaps = append(dataGaps, "Кластеры и хосты собираются, а виртуальные машины - нет (VMRecord за последние "+itoaCores(int32(utilLookbackDays))+" дней не найдены): вероятная причина - в cmdb/savevmrecord.go:SaveVMRecord строка vm_record не пишется в БД, если у VM за сутки нулевые Cpu_usage_maximum и Mem_usage_maximum (обычно это означает, что vm.Perf пуст на момент сборки - проблема на стороне сбора performance-метрик по VM, не самой страницы утилизации)")
	}
	dataGaps = append(dataGaps,
		"Нет истории по LUN/СХД (StorageSystems/StorageVolumes) - хранится только текущее состояние без суточных снимков, поэтому невозможно построить тренд роста занятого места на СХД аналогично vm_record/host_record",
		"Нет данных о привязке LUN к конкретным датastore/host на уровне точного маппинга (StorageVolumes.Server хранит одно имя сервера, при подключении LUN к нескольким хостам кластера информация теряется)",
		"Нет данных о тонком/толстом провижининге и коэффициенте переподписки (overcommit) по CPU/памяти на хостах и кластерах",
		"Нет метрик балансировки DRS/vMotion и affinity-правил, которые объясняли бы неравномерность загрузки хостов внутри кластера",
		"Нет истории простоя (длительности выключенного состояния) VM - модели.VMSprRecord.Vm_PowerState хранит только текущий снимок",
		"Нет разбивки сетевого трафика по отдельным vNIC/физическим адаптерам - Net_usage у VM/хоста/кластера агрегирован суммарно",
	)

	chartsPayload := map[string]interface{}{
		"clusters": clusterEntities,
		"hosts":    hostEntities,
		"vms":      vmEntities,
		"storages": storageEntities,
		"luns":     lunEntities,
	}
	if b, err := json.Marshal(chartsPayload); err == nil {
		this.Data["ChartsJSON"] = template.JS(b)
	}
	if b, err := json.Marshal(risks); err == nil {
		this.Data["RisksJSON"] = template.JS(b)
	}
	if b, err := json.Marshal(dataGaps); err == nil {
		this.Data["DataGapsJSON"] = template.JS(b)
	}

	this.Data["KPI"] = kpi
}

// classify возвращает "overload"/"underused"/"normal" по паре cpu/mem % и порогам.
func classify(cpu, mem, overloadCpu, overloadMem, underloadCpu, underloadMem float64) string {
	if cpu >= overloadCpu || mem >= overloadMem {
		return "overload"
	}
	if cpu <= underloadCpu && mem <= underloadMem {
		return "underused"
	}
	return "normal"
}

func addRisk(risks *[]utilRisk, category, object string, cpu, mem float64, status string, overloadRec, underRec string) {
	if status == "overload" {
		metric := "CPU avg %"
		val := cpu
		if mem >= cpu {
			metric, val = "MEM avg %", mem
		}
		*risks = append(*risks, utilRisk{Category: category, Object: object, Metric: metric, Value: val, Severity: "danger", Recommendation: overloadRec})
	} else if status == "underused" {
		*risks = append(*risks, utilRisk{Category: category, Object: object, Metric: "CPU avg %", Value: cpu, Severity: "warning", Recommendation: underRec})
	}
}

// latestByGroup из списка суточных записей (отсортированного по (group, -Date))
// выбирает по одной, самой свежей, записи на группу.
func latestByGroup[T any, K comparable](rows []T, keyFn func(T) K, dateFn func(T) time.Time) map[K]T {
	res := make(map[K]T)
	best := make(map[K]time.Time)
	for _, r := range rows {
		k := keyFn(r)
		d := dateFn(r)
		if cur, ok := best[k]; !ok || d.After(cur) {
			best[k] = d
			res[k] = r
		}
	}
	return res
}

func itoaCores(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}

func ftoa(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}
