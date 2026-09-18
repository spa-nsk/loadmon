package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"sort"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Пороговые значения индикаторов утилизации по Площадкам ЦОД - ОТДЕЛЬНЫЕ от
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

// utilDcCluster - утилизация одного кластера ВНУТРИ конкретной Площадки ЦОД
// (кластер целиком относится к той площадке, где физически размещено
// большинство/все его хосты за этот период - см. комментарий у
// SprUtilizationDatacenter про то, как определяется площадка хоста).
// HostCount/VMCount - ИСПРАВЛЕНО 2026-09-18 по замечанию пользователя "в
// кластерах не указано количество хостов и виртуальных машин по кластеру":
// считаются НЕПОСРЕДСТВЕННО из тех же хостов, что вошли в агрегаты этого
// кластера на этой странице (не через отдельный, потенциально
// рассинхронизированный подсчёт) - см. цикл ниже.
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
}

// utilDcSummary - агрегат по одной Площадке ЦОД (models.DatacenterSite):
// сумма CPU/RAM по всем её хостам-гипервизорам (сгруппированным ниже по
// кластерам ТОЛЬКО для показа разбивки - сама площадка НЕ равна кластеру и
// НЕ равна логическому vCenter Datacenter, см. комментарий у
// SprUtilizationDatacenter) + сумма СХД, физически размещённых на этой же
// площадке (models.StorageSystemPlacement), независимо от того, каким
// именно кластерам/VM они отдают LUN - "утилизация Площадки ЦОД" в целом, а
// не только той её части, что видна конкретному кластеру.
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

	StorageSystemCount int     `json:"storageSystemCount"`
	StorageTotalGb     float64 `json:"storageTotalGb"`
	StorageUsedGb      float64 `json:"storageUsedGb"`
	StoragePct         float64 `json:"storagePct"`

	Clusters []utilDcCluster `json:"clusters"`
}

// SprUtilizationDatacenter строит отчёт по утилизации виртуальной
// инфраструктуры В РАЗРЕЗЕ ФИЗИЧЕСКИХ ПЛОЩАДОК ЦОД - по прямому запросу
// пользователя (2026-09-17, "Нужно было делать по Площадкам ЦОД и уже на
// каждой странице указывать количество ресурсов и утилизацию ресурсов на
// Площадке ЦОД" - первая версия этой страницы ошибочно группировала по
// ЛОГИЧЕСКОМУ объекту vCenter "Datacenter", что пользователь явно отверг).
//
// "Площадка ЦОД" здесь - models.DatacenterSite (тот же справочник, что
// показывается на /sprsrv/datacentersitelist, с признаком IsDatacenter) -
// физическое место установки оборудования, а НЕ логический контейнер
// vCenter. Определение площадки ХОСТА-гипервизора - по цепочке физического
// размещения, ТОЙ ЖЕ, что использует остальной проект (см.
// controllers/adminhwinfra.go, controllers/sprstoragesystems.go):
//
//	HostRecord.ServerId -> ServerSprRecords.HWSprId -> ServerHardware.Id
//	  -> ServerHardware.RackCabinetId -> RackCabinet.DatacenterSiteId
//	  -> DatacenterSite.Name
//
// Хост, для которого эта цепочка не резолвится целиком (нет связанной
// ServerHardware, шкаф не указан и т.п.) - НЕ отбрасывается молча, а
// попадает в отдельный псевдо-ЦОД "Без определённой площадки" (см. ниже) -
// тот же принцип прозрачности пробелов данных, что и на /sprsrv/utilization
// (раздел "Данных недостаточно?").
//
// СХД (StorageSystems) физически привязана к площадке через
// StorageSystemPlacement (RackCabinetId ИЛИ DatacenterSiteId напрямую, см.
// комментарий у этой модели) - суммируется по каждой площадке НЕЗАВИСИМО от
// того, каким кластерам/VM она фактически отдаёт LUN (СХД физически стоит в
// одном месте и потребляет место в стойках/электричество именно там).
//
// Маршрут: /sprsrv/utildatacenter.
func (this *MainController) SprUtilizationDatacenter() {

	RegisterRequest(this, LogTypeViewLoadTable) // просмотр отчёта утилизации виртуальной инфраструктуры по Площадкам ЦОД

	this.activeContent("sprsrv/utildatacenter")

	this.Data["DcDataJSON"] = template.JS("{}")
	this.Data["DcNamesJSON"] = template.JS("[]")
	this.Data["DcCompareJSON"] = template.JS("[]")

	o := orm.NewOrm()

	// --- Диапазон дат отчёта - по умолчанию последние utilLookbackDays
	// суток, выбирается пользователем через GET ?from=&to=
	// (controllers/utildaterange.go). Метрики CPU/RAM УСРЕДНЯЮТСЯ по всем
	// записям каждого хоста в пределах этого диапазона (см. hostAvg ниже) -
	// ИЗМЕНЕНО 2026-09-18 по прямому запросу пользователя "нужно усреднение
	// за период" (первая версия с диапазоном дат ещё брала только
	// последнюю запись в диапазоне - пользователь уточнил, что нужно именно
	// усреднение). ---
	from, _, toExclusive := utilDateRangeFilter(this)

	// --- Хосты: ВСЕ суточные записи HostRecord за выбранный диапазон дат -
	// источник метрик CPU/RAM (УСРЕДНЯЕМЫХ по хосту за период, см. hostAvg
	// ниже - ИЗМЕНЕНО 2026-09-18 по прямому запросу пользователя "нужно
	// усреднение за период", раньше бралась только последняя запись) И
	// основа для определения площадки (через ServerId). ---
	var hostRecs []models.HostRecord
	o.QueryTable(new(models.HostRecord)).Filter("Date__gte", from).Filter("Date__lt", toExclusive).OrderBy("ServerId", "-Date").All(&hostRecs)

	// hostAvg - усреднённые за период метрики ОДНОГО хоста + "статичный"
	// контекст (кластер/vcenter - в норме не меняется день ото дня, поэтому
	// берётся из самой свежей записи периода, а не усредняется, усреднять
	// строковые/ID поля бессмысленно). CpuTotalMhz/RamTotalGb - тоже
	// усреднённая (а не однократно взятая) ёмкость хоста - на случай если
	// конфигурация хоста менялась внутри периода (добавили/убрали ядра или
	// память), а не просто чтобы "было симметрично" - используется для
	// расчёта итогового CpuPct/RamPct площадки/кластера как
	// avgUsed/avgTotal, что эквивалентно взвешенному по дням среднему.
	type hostAvg struct {
		ClusterId int64
		Cluster   string
		Vcenter   string

		CpuTotalMhz float64
		CpuUsedMhz  float64
		RamTotalGb  float64
		RamUsedGb   float64
	}
	type hostAccum struct {
		n                                                 int
		cpuTotalSum, cpuUsedSum, ramTotalSum, ramUsedSum float64
		capSet                                            bool
		clusterId                                         int64
		cluster, vcenter                                  string
	}
	hostAccumByServer := map[int64]*hostAccum{}
	for _, r := range hostRecs {
		a, ok := hostAccumByServer[r.ServerId]
		if !ok {
			a = &hostAccum{}
			hostAccumByServer[r.ServerId] = a
		}
		cpuTotal := float64(r.CpuMhz) * float64(r.NumCpuCores)
		cpuUsed := cpuTotal * r.Cpu_usage_average / 100
		ramTotal := r.MemorySizeGB
		ramUsed := ramTotal * r.Mem_usage_average / 100
		a.n++
		a.cpuTotalSum += cpuTotal
		a.cpuUsedSum += cpuUsed
		a.ramTotalSum += ramTotal
		a.ramUsedSum += ramUsed
		if !a.capSet { // первая запись группы - самая свежая (сортировка -Date), берём кластер/vcenter отсюда
			a.clusterId = r.ClusterId
			a.cluster = r.Cluster
			a.vcenter = r.Vcenter
			a.capSet = true
		}
	}
	avgHostByServer := map[int64]hostAvg{}
	for serverId, a := range hostAccumByServer {
		if a.n == 0 {
			continue
		}
		n := float64(a.n)
		avgHostByServer[serverId] = hostAvg{
			ClusterId:   a.clusterId,
			Cluster:     a.cluster,
			Vcenter:     a.vcenter,
			CpuTotalMhz: a.cpuTotalSum / n,
			CpuUsedMhz:  a.cpuUsedSum / n,
			RamTotalGb:  a.ramTotalSum / n,
			RamUsedGb:   a.ramUsedSum / n,
		}
	}

	// --- Кластеры: только для имени/vCenter в разбивке по кластерам внутри
	// площадки (сами метрики CPU/RAM кластера здесь НЕ используются - берутся
	// напрямую из хостов, см. комментарий у SprUtilizationDatacenter). ---
	var clusterRecs []models.ClusterRecord
	o.QueryTable(new(models.ClusterRecord)).Filter("Date__gte", from).Filter("Date__lt", toExclusive).OrderBy("ClusterId", "-Date").All(&clusterRecs)
	latestClusterByKey := latestByGroup(clusterRecs, func(r models.ClusterRecord) int64 { return r.ClusterId }, func(r models.ClusterRecord) time.Time { return r.Date })

	// --- ВМ: последняя суточная запись VMRecord - для подсчёта VMCount по
	// кластеру/площадке (см. vmCountByCluster ниже). ---
	var vmRecs []models.VMRecord
	o.QueryTable(new(models.VMRecord)).Filter("Date__gte", from).Filter("Date__lt", toExclusive).OrderBy("VMSprId", "-Date").All(&vmRecs)
	latestVMBySpr := latestByGroup(vmRecs, func(r models.VMRecord) int64 { return r.VMSprId }, func(r models.VMRecord) time.Time { return r.Date })
	vmCountByCluster := map[int64]int{}
	for _, v := range latestVMBySpr {
		vmCountByCluster[v.ClusterId]++
	}

	// --- Цепочка физического размещения хоста -> площадка (см. комментарий
	// у функции выше). Строится тремя короткими join-выборками по
	// собранным на предыдущем шаге Id, а не по одной хосту за раз - иначе
	// на инфраструктуре с сотнями хостов это было бы N+3 запроса вместо 4. ---
	serverIds := make([]int64, 0, len(avgHostByServer))
	for id := range avgHostByServer {
		serverIds = append(serverIds, id)
	}
	hwSprIdByServerId := map[int64]int64{}
	if len(serverIds) > 0 {
		var sprs []models.ServerSprRecords
		o.QueryTable(new(models.ServerSprRecords)).Filter("Id__in", serverIds).All(&sprs, "Id", "HWSprId")
		for _, s := range sprs {
			if s.HWSprId > 0 {
				hwSprIdByServerId[s.Id] = s.HWSprId
			}
		}
	}

	hwIds := make([]int64, 0, len(hwSprIdByServerId))
	for _, hwId := range hwSprIdByServerId {
		hwIds = append(hwIds, hwId)
	}
	cabinetIdByHWId := map[int64]int64{}
	if len(hwIds) > 0 {
		var hws []models.ServerHardware
		o.QueryTable(new(models.ServerHardware)).Filter("Id__in", hwIds).All(&hws, "Id", "RackCabinetId")
		for _, h := range hws {
			if h.RackCabinetId > 0 {
				cabinetIdByHWId[h.Id] = h.RackCabinetId
			}
		}
	}

	cabIds := make([]int64, 0, len(cabinetIdByHWId))
	for _, cabId := range cabinetIdByHWId {
		cabIds = append(cabIds, cabId)
	}
	siteIdByCabinetId := map[int64]int64{}
	if len(cabIds) > 0 {
		var cabs []models.RackCabinet
		o.QueryTable(new(models.RackCabinet)).Filter("Id__in", cabIds).All(&cabs, "Id", "DatacenterSiteId")
		for _, c := range cabs {
			if c.DatacenterSiteId > 0 {
				siteIdByCabinetId[c.Id] = c.DatacenterSiteId
			}
		}
	}

	siteIdSet := map[int64]bool{}
	for _, siteId := range siteIdByCabinetId {
		siteIdSet[siteId] = true
	}
	siteIds := make([]int64, 0, len(siteIdSet))
	for id := range siteIdSet {
		siteIds = append(siteIds, id)
	}
	siteNameById := map[int64]string{}
	if len(siteIds) > 0 {
		var sites []models.DatacenterSite
		o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", siteIds).All(&sites, "Id", "Name")
		for _, s := range sites {
			siteNameById[s.Id] = s.Name
		}
	}

	const noSiteLabel = "Без определённой площадки"

	// siteNameByServerId - итог всей цепочки: имя площадки (или noSiteLabel,
	// если цепочка не резолвится целиком) по ServerId хоста.
	siteNameByServerId := map[int64]string{}
	for serverId := range avgHostByServer {
		name := noSiteLabel
		if hwId, ok := hwSprIdByServerId[serverId]; ok {
			if cabId, ok := cabinetIdByHWId[hwId]; ok {
				if siteId, ok := siteIdByCabinetId[cabId]; ok {
					if n, ok := siteNameById[siteId]; ok && n != "" {
						name = n
					}
				}
			}
		}
		siteNameByServerId[serverId] = name
	}

	// --- ЭТАП 1 (по прямому требованию пользователя - "первым этапом
	// определи список таких ЦОД"): список Площадок ЦОД, где есть хотя бы
	// один хост-гипервизор с данными за период. ---
	dcByName := map[string]*utilDcSummary{}
	var dcNames []string
	// clusterAgg - промежуточная агрегация "площадка -> ClusterId -> сумма
	// по хостам этого кластера на этой площадке" - кластер попадает в
	// разбивку ОТДЕЛЬНО на каждой площадке, где реально стоят его хосты
	// (в норме - одна площадка, но код не предполагает этого жёстко).
	type clusterAggKey struct {
		site      string
		clusterId int64
	}
	clusterAgg := map[clusterAggKey]*utilDcCluster{}

	for serverId, h := range avgHostByServer {
		site := siteNameByServerId[serverId]

		sum, ok := dcByName[site]
		if !ok {
			sum = &utilDcSummary{Name: site}
			dcByName[site] = sum
			dcNames = append(dcNames, site)
		}

		cpuTotal := h.CpuTotalMhz
		cpuUsed := h.CpuUsedMhz
		ramTotal := h.RamTotalGb
		ramUsed := h.RamUsedGb

		sum.HostCount++
		sum.CpuTotalMhz += cpuTotal
		sum.CpuUsedMhz += cpuUsed
		sum.RamTotalGb += ramTotal
		sum.RamUsedGb += ramUsed

		key := clusterAggKey{site: site, clusterId: h.ClusterId}
		cl, ok := clusterAgg[key]
		if !ok {
			name := h.Cluster
			vcenter := h.Vcenter
			if cr, ok := latestClusterByKey[h.ClusterId]; ok {
				if cr.Name != "" {
					name = cr.Name
				}
				if cr.Vcenter != "" {
					vcenter = cr.Vcenter
				}
			}
			if name == "" {
				name = "(без кластера)"
			}
			cl = &utilDcCluster{Name: name, Vcenter: vcenter, VMCount: vmCountByCluster[h.ClusterId]}
			clusterAgg[key] = cl
		}
		cl.HostCount++
		cl.CpuTotalMhz += cpuTotal
		cl.CpuUsedMhz += cpuUsed
		cl.RamTotalGb += ramTotal
		cl.RamUsedGb += ramUsed
	}

	for key, cl := range clusterAgg {
		cl.CpuPct = pctOf(cl.CpuUsedMhz, cl.CpuTotalMhz)
		cl.RamPct = pctOf(cl.RamUsedGb, cl.RamTotalGb)
		sum := dcByName[key.site]
		sum.Clusters = append(sum.Clusters, *cl)
		sum.ClusterCount++
		sum.VMCount += cl.VMCount
	}

	// --- СХД по площадке (StorageSystemPlacement -> площадка, см.
	// комментарий у SprUtilizationDatacenter) - считается для ВСЕХ площадок,
	// у которых уже есть запись в dcByName (то есть где есть гипервизоры) -
	// СХД на площадках без гипервизоров сюда сознательно не попадает, этот
	// отчёт именно про "утилизацию ВИРТУАЛЬНОЙ инфраструктуры". ---
	if len(dcByName) > 0 {
		var placements []models.StorageSystemPlacement
		o.QueryTable(new(models.StorageSystemPlacement)).All(&placements)

		// storageIdsBySite - дедупликация: одна СХД может иметь несколько
		// записей размещения (несколько шкафов) на одной площадке - считать
		// её ёмкость нужно ровно один раз.
		storageIdsBySite := map[string]map[int64]bool{}
		for _, p := range placements {
			siteId := p.DatacenterSiteId
			if p.RackCabinetId > 0 {
				if resolved, ok := siteIdByCabinetId[p.RackCabinetId]; ok {
					siteId = resolved
				} else {
					// Шкаф указан, но сам не привязан к площадке (или не
					// попал в siteIdByCabinetId, т.к. на нём не оказалось ни
					// одного хоста-гипервизора из выборки выше) - подгружаем
					// точечно, это не частый случай.
					var cab models.RackCabinet
					if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", p.RackCabinetId).One(&cab, "DatacenterSiteId"); err == nil {
						siteId = cab.DatacenterSiteId
					}
				}
			}
			if siteId <= 0 {
				continue
			}
			name, ok := siteNameById[siteId]
			if !ok || name == "" {
				var site models.DatacenterSite
				if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", siteId).One(&site, "Name"); err == nil {
					name = site.Name
					siteNameById[siteId] = name
				}
			}
			if name == "" {
				continue
			}
			if _, ok := dcByName[name]; !ok {
				continue // площадка без гипервизоров - СХД сюда не включаем, см. комментарий выше
			}
			set, ok := storageIdsBySite[name]
			if !ok {
				set = map[int64]bool{}
				storageIdsBySite[name] = set
			}
			set[p.StorageSystemsId] = true
		}

		// Собираем все затронутые StorageSystemsId одним запросом.
		allStorageIds := map[int64]bool{}
		for _, set := range storageIdsBySite {
			for id := range set {
				allStorageIds[id] = true
			}
		}
		if len(allStorageIds) > 0 {
			ids := make([]int64, 0, len(allStorageIds))
			for id := range allStorageIds {
				ids = append(ids, id)
			}
			var storages []models.StorageSystems
			o.QueryTable(new(models.StorageSystems)).Filter("Id__in", ids).All(&storages, "Id", "TotalSizeRAW", "UsedPercent", "Status")
			storageById := map[int64]models.StorageSystems{}
			for _, s := range storages {
				storageById[s.Id] = s
			}
			for name, set := range storageIdsBySite {
				sum := dcByName[name]
				for id := range set {
					s, ok := storageById[id]
					if !ok || s.Status == 6 { // демонтированное оборудование - не учитываем, тот же фильтр, что на /sprsrv/utilization
						continue
					}
					used := s.TotalSizeRAW * s.UsedPercent / 100
					sum.StorageSystemCount++
					sum.StorageTotalGb += s.TotalSizeRAW
					sum.StorageUsedGb += used
				}
			}
		}
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

	// --- Сводная вкладка сравнения: те же итоги по площадкам одним плоским
	// списком в порядке dcNames - шаблон строит сравнительные диаграммы и
	// таблицу разниц на клиенте. ---
	compare := make([]*utilDcSummary, 0, len(dcNames))
	for _, name := range dcNames {
		compare = append(compare, dcByName[name])
	}
	if b, err := json.Marshal(compare); err == nil {
		this.Data["DcCompareJSON"] = template.JS(b)
	}

	this.Data["DcCount"] = len(dcNames)
}

// pctOf - used/total*100 с защитой от деления на 0 (когда у площадки/
// кластера почему-то нулевая суммарная ёмкость ресурса).
func pctOf(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return used / total * 100
}
