package controllers

import (
	"loadmon/models"
	"log"
	"sort"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// PeriodMetric - агрегированный по периоду показатель (максимум/среднее/минимум).
// Максимум и минимум берутся как максимум максимумов и минимум минимумов
// суточных записей (ClusterRecord/HostRecord/VMRecord из таблиц БД
// cluster_record/host_record/vm_record - по одной записи в сутки), среднее -
// как простое среднее суточных средних (используется приближённо, без учёта
// разного числа замеров внутри разных суток). Счётчик суток n хранится внутри
// самой метрики, поэтому вызывающему коду не нужно вручную вести отдельные
// счётчики для каждого из полей CPU/Mem/Disk/Net.
type PeriodMetric struct {
	Max, Avg, Min float64
	n             int
}

// addDay добавляет в агрегат данные ещё одной суточной записи.
func (m *PeriodMetric) addDay(max, avg, min float64) {
	if m.n == 0 || max > m.Max {
		m.Max = max
	}
	if m.n == 0 || min < m.Min {
		m.Min = min
	}
	m.Avg += avg
	m.n++
}

// finish превращает накопленную сумму средних в собственно среднее -
// вызывается один раз после того, как все суточные записи добавлены.
func (m *PeriodMetric) finish() {
	if m.n > 0 {
		m.Avg /= float64(m.n)
	}
}

// ClusterPeriodRow - агрегированная за период строка кластера для отображения
// на /sprsrv/clusterperioddb, аналогично vsphere.Cluster на /sprsrv/cluster,
// но построенная из данных БД (ClusterRecord/HostRecord/VMRecord), а не из
// текущего состояния vSphere в памяти (oneview2/vsphere.Infrastructure).
type ClusterPeriodRow struct {
	Vcenter, Datacenter, Name              string
	AllNumCpuPkgs, AllNumCpuCores          int16
	CpuMhzAvg                              int32
	AllMemorySizeGB                        float64
	AllCapacityGb, AllFreeSpaceGb          float64
	AllPercentUsed                         float64
	CPU, Mem, Disk, Net                    PeriodMetric
	HostCount, VMCount                     int64
	RecordCount                            int
	Hosts                                  []*HostPeriodRow
	VM                                     []*VMPeriodRow
	// VMDataMissing = true, когда по текущей конфигурации кластера ВМ есть
	// (VMCount > 0), но за выбранный период в vm_record не нашлось ни одной
	// сохранённой записи по ВМ этого кластера - отличает "данных ещё нет"
	// от "ВМ в кластере нет" в шаблоне (views/sprsrv/clusterperioddb.tpl).
	VMDataMissing bool
}

// HostPeriodRow - агрегированная за период строка хоста кластера (по данным HostRecord).
type HostPeriodRow struct {
	Name                           string
	SerialNumber                   string
	NumCpuPkgs, NumCpuCores        int16
	CpuMhz                         int32
	MemorySizeGB                   float64
	CPU, Mem, Disk, Net            PeriodMetric
}

// VMPeriodRow - агрегированная за период строка виртуальной машины кластера (по данным VMRecord).
type VMPeriodRow struct {
	Name                     string
	NumCPU, NumCores         int32
	MemorySizeGB             float64
	CPU, Mem, Disk, Net      PeriodMetric
}

// SprClusterPeriodDB отображает список кластеров и одиночных хостов с метриками,
// подсчитанными по данным БД (таблицы cluster_record/host_record/vm_record,
// заполняются ежесуточно в cmdb.SaveMetricData) за указанный период - в
// отличие от /sprsrv/cluster (controllers/sprclusterview.go), которая берёт
// ТЕКУЩЕЕ состояние из vsphere.Infrastructure в памяти через gRPC. Разметка и
// состав колонок повторяют /sprsrv/cluster (views/sprsrv/cluster.tpl).
//
// Маршрут: /sprsrv/clusterperioddb (фильтр vcenter/datacenter/cluster и период
// from/to передаются GET-параметрами, как на /sprsrv/listclusterview).
func (this *MainController) SprClusterPeriodDB() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewLoadTable) // просмотр списка кластеров/хостов с загрузкой за период (по данным БД)

	this.activeContent("sprsrv/clusterperioddb")

	vcenter := this.GetString("vcenter")
	datacenter := this.GetString("datacenter")
	cluster := this.GetString("cluster")
	if vcenter == "" {
		vcenter = "all"
	}
	if datacenter == "" {
		datacenter = "all"
	}
	if cluster == "" {
		cluster = "all"
	}

	to := CorrectTimeFilter(time.Now())
	from := to.AddDate(0, 0, -6)
	if s := this.GetString("from"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			from = t
		}
	}
	if s := this.GetString("to"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			to = t
		}
	}
	if to.Before(from) {
		from, to = to, from
	}
	dateFrom := from.UTC()
	dateTo := to.AddDate(0, 0, 1).UTC() // включительно до конца дня "to"

	this.Data["CurVcenter"] = vcenter
	this.Data["CurDatacenter"] = datacenter
	this.Data["CurCluster"] = cluster
	this.Data["FromDate"] = from.Format("2006-01-02")
	this.Data["ToDate"] = to.Format("2006-01-02")

	flash := beego.NewFlash()
	o := orm.NewOrm()

	// справочник кластеров - источник списков для фильтра (vcenter/датацентр/
	// кластер) и точка входа для поиска суточных записей за период
	var sprAll []models.ClusterSprRecords
	if _, err := o.QueryTable("cluster_spr_records").All(&sprAll); err != nil {
		this.Data["ClusterList"] = []*ClusterPeriodRow{}
		return
	}

	vcSet := make(map[string]struct{})
	dcSet := make(map[string]struct{})
	clSet := make(map[string]struct{})
	sprBySpr := make(map[int64]*models.ClusterSprRecords, len(sprAll))
	matchedSprIds := make([]int64, 0, len(sprAll))
	for i := range sprAll {
		s := &sprAll[i]
		vcSet[s.Vcenter] = struct{}{}
		if vcenter == "all" || s.Vcenter == vcenter {
			dcSet[s.Datacenter] = struct{}{}
		}
		if (vcenter == "all" || s.Vcenter == vcenter) && (datacenter == "all" || s.Datacenter == datacenter) {
			clSet[s.Name] = struct{}{}
		}
		sprBySpr[s.Id] = s

		if (vcenter == "all" || s.Vcenter == vcenter) &&
			(datacenter == "all" || s.Datacenter == datacenter) &&
			(cluster == "all" || s.Name == cluster) {
			matchedSprIds = append(matchedSprIds, s.Id)
		}
	}
	this.Data["VcenterList"] = toSortedSlice(vcSet)
	this.Data["DatacenterList"] = toSortedSlice(dcSet)
	this.Data["ClusterList2"] = toSortedSlice(clSet)

	rowsBySpr := make(map[int64]*ClusterPeriodRow)
	clRecIdToSpr := make(map[int64]int64)
	allClRecIds := make([]int64, 0)

	if len(matchedSprIds) > 0 {
		var clRecs []models.ClusterRecord
		if _, err := o.QueryTable("cluster_record").
			Filter("ClusterId__in", matchedSprIds).
			Filter("Date__gte", dateFrom).
			Filter("Date__lt", dateTo).
			OrderBy("Date").
			All(&clRecs); err != nil {
			log.Println("SprClusterPeriodDB: ошибка выборки cluster_record:", err)
			flash.Error("Ошибка чтения данных кластеров за период (cluster_record): %v", err)
			flash.Store(&this.Controller)
		}

		for i := range clRecs {
			r := &clRecs[i]
			sprId := r.ClusterId
			spr := sprBySpr[sprId]
			if spr == nil {
				continue
			}
			row, ok := rowsBySpr[sprId]
			if !ok {
				row = &ClusterPeriodRow{Vcenter: spr.Vcenter, Datacenter: spr.Datacenter, Name: spr.Name}
				rowsBySpr[sprId] = row
			}
			row.CPU.addDay(r.Cpu_usage_maximum, r.Cpu_usage_average, r.Cpu_usage_minimum)
			row.Mem.addDay(r.Mem_usage_maximum, r.Mem_usage_average, r.Mem_usage_minimum)
			row.Disk.addDay(r.Disk_usage_maximum, r.Disk_usage_average, r.Disk_usage_minimum)
			row.Net.addDay(r.Net_usage_maximum, r.Net_usage_average, r.Net_usage_minimum)

			// статические характеристики (число CPU/память/объём хранилища) и
			// счётчики хостов/ВМ берём из самой свежей записи периода - это
			// текущая конфигурация, а не показатель нагрузки, усреднять его
			// по суткам не имеет смысла
			row.AllNumCpuPkgs = r.AllNumCpuPkgs
			row.AllNumCpuCores = r.AllNumCpuCores
			row.CpuMhzAvg = r.CpuMhzAvg
			row.AllMemorySizeGB = r.AllMemorySizeGB
			row.AllCapacityGb = r.AllCapacityGb
			row.AllFreeSpaceGb = r.AllFreeSpaceGb
			row.AllPercentUsed = r.AllPercentUsed
			row.HostCount = r.HostCount
			row.VMCount = r.VMCount
			row.RecordCount++

			clRecIdToSpr[r.Id] = sprId
			allClRecIds = append(allClRecIds, r.Id)
		}
		for _, row := range rowsBySpr {
			row.CPU.finish()
			row.Mem.finish()
			row.Disk.finish()
			row.Net.finish()
		}
	}

	// хосты кластеров за период (HostRecord.ClusterId ссылается на конкретную
	// суточную запись ClusterRecord.Id, а не на справочник cluster_spr_records -
	// см. cmdb.SaveHostRecord/cmdb.SaveClusterRecord)
	hostsBySpr := make(map[int64]map[string]*HostPeriodRow)
	serverIdSet := make(map[int64]struct{})
	hostServerIdByKey := make(map[string]int64)

	if len(allClRecIds) > 0 {
		var hostRecs []models.HostRecord
		if _, err := o.QueryTable("host_record").
			Filter("ClusterId__in", allClRecIds).
			OrderBy("Date").
			All(&hostRecs); err != nil {
			log.Println("SprClusterPeriodDB: ошибка выборки host_record:", err)
			flash.Error("Ошибка чтения данных хостов за период (host_record): %v", err)
			flash.Store(&this.Controller)
		}

		for i := range hostRecs {
			h := &hostRecs[i]
			sprId, ok := clRecIdToSpr[h.ClusterId]
			if !ok {
				continue
			}
			key := h.Name
			if hostsBySpr[sprId] == nil {
				hostsBySpr[sprId] = make(map[string]*HostPeriodRow)
			}
			row, ok := hostsBySpr[sprId][key]
			if !ok {
				row = &HostPeriodRow{Name: h.Name}
				hostsBySpr[sprId][key] = row
			}
			row.CPU.addDay(h.Cpu_usage_maximum, h.Cpu_usage_average, h.Cpu_usage_minimum)
			row.Mem.addDay(h.Mem_usage_maximum, h.Mem_usage_average, h.Mem_usage_minimum)
			row.Disk.addDay(h.Disk_usage_maximum, h.Disk_usage_average, h.Disk_usage_minimum)
			row.Net.addDay(h.Net_usage_maximum, h.Net_usage_average, h.Net_usage_minimum)

			row.NumCpuPkgs = h.NumCpuPkgs
			row.NumCpuCores = h.NumCpuCores
			row.CpuMhz = h.CpuMhz
			row.MemorySizeGB = h.MemorySizeGB

			if h.ServerId > 0 {
				serverIdSet[h.ServerId] = struct{}{}
				hostServerIdByKey[key] = h.ServerId
			}
		}
		for _, hosts := range hostsBySpr {
			for _, row := range hosts {
				row.CPU.finish()
				row.Mem.finish()
				row.Disk.finish()
				row.Net.finish()
			}
		}
	}

	// серийные номера хостов для ссылки на конфигурацию сервера (аналогично
	// /sprsrv/cluster) - одним запросом по всем встретившимся ServerId
	serialByServerId := make(map[int64]string)
	if len(serverIdSet) > 0 {
		serverIds := make([]int64, 0, len(serverIdSet))
		for id := range serverIdSet {
			serverIds = append(serverIds, id)
		}
		var srvList []models.ServerSprRecords
		_, _ = models.ServerSprQS(o).Filter("Id__in", serverIds).All(&srvList, "Id", "SerialNumber")
		for _, s := range srvList {
			serialByServerId[s.Id] = s.SerialNumber
		}
	}

	// виртуальные машины кластеров за период (VMRecord.ClusterId - та же связь
	// с суточной записью ClusterRecord.Id, что и у HostRecord)
	vmsBySpr := make(map[int64]map[int64]*VMPeriodRow)
	vmSprIdSet := make(map[int64]struct{})

	if len(allClRecIds) > 0 {
		var vmRecs []models.VMRecord
		if _, err := o.QueryTable("vm_record").
			Filter("ClusterId__in", allClRecIds).
			OrderBy("Date").
			All(&vmRecs); err != nil {
			log.Println("SprClusterPeriodDB: ошибка выборки vm_record:", err)
			flash.Error("Ошибка чтения данных виртуальных машин за период (vm_record): %v", err)
			flash.Store(&this.Controller)
		}

		for i := range vmRecs {
			v := &vmRecs[i]
			sprId, ok := clRecIdToSpr[v.ClusterId]
			if !ok || v.VMSprId == 0 {
				continue
			}
			if vmsBySpr[sprId] == nil {
				vmsBySpr[sprId] = make(map[int64]*VMPeriodRow)
			}
			row, ok := vmsBySpr[sprId][v.VMSprId]
			if !ok {
				row = &VMPeriodRow{}
				vmsBySpr[sprId][v.VMSprId] = row
			}
			row.CPU.addDay(v.Cpu_usage_maximum, v.Cpu_usage_average, v.Cpu_usage_minimum)
			row.Mem.addDay(v.Mem_usage_maximum, v.Mem_usage_average, v.Mem_usage_minimum)
			row.Disk.addDay(v.Disk_usage_maximum, v.Disk_usage_average, v.Disk_usage_minimum)
			row.Net.addDay(v.Net_usage_maximum, v.Net_usage_average, v.Net_usage_minimum)

			row.NumCPU = v.Vm_NumCPU
			row.NumCores = v.Vm_NumCores
			row.MemorySizeGB = v.Vm_MemorySizeGB

			vmSprIdSet[v.VMSprId] = struct{}{}
		}
		for _, vms := range vmsBySpr {
			for _, row := range vms {
				row.CPU.finish()
				row.Mem.finish()
				row.Disk.finish()
				row.Net.finish()
			}
		}
	}

	// имена ВМ (VMRecord их не хранит - только ссылку VMSprId на vm_spr_record)
	vmNameById := make(map[int64]string)
	if len(vmSprIdSet) > 0 {
		vmSprIds := make([]int64, 0, len(vmSprIdSet))
		for id := range vmSprIdSet {
			vmSprIds = append(vmSprIds, id)
		}
		var vmSprList []models.VMSprRecord
		_, _ = o.QueryTable("vm_spr_record").Filter("Id__in", vmSprIds).All(&vmSprList, "Id", "Vm_Name")
		for _, v := range vmSprList {
			vmNameById[v.Id] = v.Vm_Name
		}
	}

	// сборка итогового списка
	clusterList := make([]*ClusterPeriodRow, 0, len(rowsBySpr))
	for sprId, row := range rowsBySpr {
		if hosts, ok := hostsBySpr[sprId]; ok {
			for key, h := range hosts {
				h.SerialNumber = serialByServerId[hostServerIdByKey[key]]
				row.Hosts = append(row.Hosts, h)
			}
			sort.Slice(row.Hosts, func(i, j int) bool { return row.Hosts[i].Name < row.Hosts[j].Name })
		}
		if vms, ok := vmsBySpr[sprId]; ok {
			for vmSprId, v := range vms {
				v.Name = vmNameById[vmSprId]
				row.VM = append(row.VM, v)
			}
			sort.Slice(row.VM, func(i, j int) bool { return row.VM[i].Name < row.VM[j].Name })
		}
		if row.VMCount > 0 && len(row.VM) == 0 {
			row.VMDataMissing = true
		}
		clusterList = append(clusterList, row)
	}
	sort.Slice(clusterList, func(i, j int) bool {
		if clusterList[i].Vcenter != clusterList[j].Vcenter {
			return clusterList[i].Vcenter < clusterList[j].Vcenter
		}
		if clusterList[i].Datacenter != clusterList[j].Datacenter {
			return clusterList[i].Datacenter < clusterList[j].Datacenter
		}
		return clusterList[i].Name < clusterList[j].Name
	})

	this.Data["ClusterList"] = clusterList
}

// toSortedSlice возвращает отсортированный срез ключей множества (используется
// для построения списков значений фильтра vcenter/датацентр/кластер).
func toSortedSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
