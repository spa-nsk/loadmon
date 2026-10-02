package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/itsystem"
	"loadmon/models"
	"sort"

	"github.com/beego/beego/v2/client/orm"
)

// utilItSysServer - индивидуальная строка списка серверов внутри блока
// (виртуальная инфраструктура ИЛИ аппаратные серверы) одной ИТ системы -
// по прямому требованию пользователя ("в каждом блоке укажи список серверов
// с их индивидуальной загрузкой"), а не только агрегат по блоку.
type utilItSysServer struct {
	Name   string `json:"name"`
	CpuPct float64 `json:"cpuPct"`
	RamPct float64 `json:"ramPct"`
	// Detail - произвольная короткая строка с дополнительным контекстом
	// (для VM: vCPU/RAM размер + число усреднённых записей за период; для
	// аппаратных серверов: число усреднённых замеров ServerMetric за
	// период) - показывается под именем сервера.
	Detail string `json:"detail"`
}

// utilItSysBlock - один из двух блоков на странице ИТ системы (виртуальная
// инфраструктура ИЛИ аппаратные серверы). Servers[i].CpuPct/RamPct - УЖЕ
// усреднённые ЗА ВЫБРАННЫЙ ПЕРИОД значения для конкретного сервера/VM (см.
// vmAccum/metricAccum в SprUtilizationItSystem - "нужно усреднение за
// период", 2026-09-18). CpuPct/RamPct самого блока - ПРОСТОЕ среднее ПО
// ЭТИМ уже усреднённым построчным значениям (НЕ средневзвешенное по
// реальной ёмкости узла/MHz-ёмкости, как на /sprsrv/utildatacenter) - у
// VMRecord и models.ServerMetric нет полей суммарной физической ёмкости
// ресурса (только проценты утилизации), поэтому "взвешивать" здесь нечем
// без домысливания данных, которых в схеме нет (см. kontext/05-conventions-
// and-gotchas.md, принцип "не гадать") - среднее по списку серверов блока
// является честным и объяснимым показателем при таких исходных данных.
type utilItSysBlock struct {
	Count   int               `json:"count"`
	CpuPct  float64           `json:"cpuPct"`
	RamPct  float64           `json:"ramPct"`
	Servers []utilItSysServer `json:"servers"`
}

// utilItSysSummary - агрегат по одной ИТ системе (models.ItSystemRecord,
// сопоставленной по ServerSprRecords.ITSystemKey - см. комментарий у
// SprUtilizationItSystem).
type utilItSysSummary struct {
	Key  string `json:"key"`
	Name string `json:"name"`

	VM utilItSysBlock `json:"vm"`
	HW utilItSysBlock `json:"hw"`
}

// SprUtilizationItSystem строит отчёт по утилизации инфраструктуры В
// РАЗРЕЗЕ ИТ СИСТЕМ - по прямому запросу пользователя (2026-09-18, "сделай
// аналогичный отчет но в другом разрезе по ИТ Системам. Теперь каждая
// страница по отдельной ИТ системе, без учёта размещения. В ИТ системах
// может быть часть виртуальной инфраструктуры и аппаратные серверы. Сделай
// на каждой странице 2 блока 1 по виртуальной инфраструктуре, второй по
// аппаратным серверам. Утилизацию по аппаратным серверам выбирай из
// models.ServerMetric. В каждом блоке укажи список серверов с их
// индивидуальной загрузкой.").
//
// "ИТ система" здесь - НЕ отдельная ORM-таблица-справочник в этом проекте
// (см. controllers/sprsrvview2.go, itsystem/jira.go): это плоский список
// models.ItSystemRecord, импортируемый из Jira CMDB и кэшируемый в памяти
// пакета itsystem (itsystem.ImportITSystemSpr() - дешёвая операция, читает
// уже загруженный кэш под RLock, а не ходит в Jira на каждый запрос
// страницы). Привязка КОНКРЕТНОГО сервера к ИТ системе - строкой
// ServerSprRecords.ITSystemKey (сверяется с ItSystemRecord.Key) - это
// ЕДИНСТВЕННОЕ поле привязки, общее для физических серверов (Type=0) и
// виртуальных машин (Type=1) в одной и той же таблице ServerSprRecords -
// в отличие от /sprsrv/utildatacenter здесь НЕТ цепочки физического
// размещения (шкаф/площадка) - группировка по ИТ системе полностью
// "без учёта размещения", как прямо указал пользователь.
//
// Два блока на страницу, ПО ИСТОЧНИКУ ДАННЫХ (не просто "виртуальные и
// физические машины", а разные способы измерения загрузки):
//   - "Виртуальная инфраструктура" - ServerSprRecords.Type == 1 (VM),
//     метрики CPU/RAM из models.VMRecord.Cpu_usage_average/Mem_usage_average
//     (замеры на уровне гипервизора/vCenter, тот же источник, что и на
//     /sprsrv/utildatacenter).
//   - "Аппаратные серверы" - ServerSprRecords.Type == 0 (физический
//     bare-metal сервер), метрики CPU/RAM из models.ServerMetric (замеры
//     на уровне ОС через Zabbix-агента, см. комментарий у самой модели) -
//     ПРЯМО указано пользователем: "Утилизацию по аппаратным серверам
//     выбирай из models.ServerMetric". Связь ServerMetric (Source, HostId) ->
//     ServerSprRecords - через ServerSprRecords.ZabbixHostId (Zabbix, в
//     приоритете) или ScomHostId (SCOM, только для серверов без Zabbix),
//     правило - models.ServerMetricHostFor.
//
// ServerSprRecords.Type == 2 (host vmware, сам гипервизор) в этот отчёт
// СОЗНАТЕЛЬНО не включается - гипервизор является инфраструктурным
// компонентом площадки (см. /sprsrv/utildatacenter), а не частью
// прикладной ИТ системы, которой он не принадлежит "как сервис".
//
// Серверы без заполненного ITSystemKey (пустая строка) исключены из
// отчёта - в отличие от /sprsrv/utildatacenter (где НЕ резолвящийся хост
// всё равно показывается в псевдо-площадке "Без определённой площадки"),
// здесь такой псевдо-группы сознательно нет: страница именно "по ИТ
// системам", а не общий реестр серверов, и количество серверов без
// привязки к ИТ системе в этой инфраструктуре может быть очень большим
// (тестовые/служебные/ещё не размеченные в Jira) - реестр таких серверов
// уже есть на /sprsrv/view2 с фильтром по пустой ИТ системе.
//
// Маршрут: /sprsrv/utilitsystem.
func (this *MainController) SprUtilizationItSystem() {

	RegisterRequest(this, LogTypeViewLoadTable) // просмотр отчёта утилизации инфраструктуры по ИТ системам

	this.activeContent("sprsrv/utilitsystem")

	this.Data["ItSysDataJSON"] = template.JS("{}")
	this.Data["ItSysNamesJSON"] = template.JS("[]")

	o := orm.NewOrm()

	// --- Диапазон дат отчёта - настраиваемый пользователем через GET
	// ?from=&to= (см. controllers/utildaterange.go) - по прямому запросу
	// пользователя (2026-09-18, "за какой период формируются оба отчета?
	// Добавь диапазон дат для формирования за период в оба отчета"). Как и
	// на /sprsrv/utildatacenter, это НЕ усреднение за период - по каждой
	// VM/серверу берётся последняя запись в пределах выбранного окна. ---
	from, _, toExclusive := utilDateRangeFilter(this)

	// --- Справочник ИТ систем из Jira (кэш в памяти, см. комментарий выше) -
	// имя/название для отображения по ключу, только для КРАСИВОГО заголовка
	// вкладки/карточки (если система с таким ключом не найдена в кэше -
	// используется сам ключ, транспарентно, как и везде в проекте). ---
	itSysNameByKey := map[string]string{}
	for _, rec := range itsystem.ImportITSystemSpr() {
		if rec.Key == "" {
			continue
		}
		name := rec.Name
		if name == "" {
			name = rec.Label
		}
		if name == "" {
			name = rec.FullName
		}
		if name != "" {
			itSysNameByKey[rec.Key] = name
		}
	}

	// --- Справочник серверов с непустым ITSystemKey - основа группировки,
	// одним запросом на оба блока (Type IN (0,1)), см. комментарий у
	// функции выше про то, почему Type=2 (гипервизор) сюда не входит. ---
	var sprs []models.ServerSprRecords
	models.ServerSprQS(o).
		Filter("Type__in", 0, 1).
		Exclude("ITSystemKey", "").
		Filter("Active", 1).
		All(&sprs, "Id", "Type", "Name", "NameFQDN", "ITSystemKey", "ZabbixHostId", "ScomHostId")

	vmSprById := map[int64]models.ServerSprRecords{}
	hwSprById := map[int64]models.ServerSprRecords{}
	// Источник метрик аппаратного сервера - по единому правилу
	// models.ServerMetricHostFor: Zabbix в приоритете, SCOM - только для
	// серверов без ZabbixHostId. HostId уникален лишь в пределах источника,
	// поэтому обратный индекс - по источнику: hwSprIdByHost[source][hostId].
	hwSprIdByHost := map[int]map[int64]int64{}
	for _, s := range sprs {
		switch s.Type {
		case 1:
			vmSprById[s.Id] = s
		case 0:
			hwSprById[s.Id] = s
			if source, hostId, ok := models.ServerMetricHostFor(&s); ok {
				if hwSprIdByHost[source] == nil {
					hwSprIdByHost[source] = map[int64]int64{}
				}
				hwSprIdByHost[source][hostId] = s.Id
			}
		}
	}

	itSysByKey := map[string]*utilItSysSummary{}
	var itSysKeys []string
	getSys := func(key string) *utilItSysSummary {
		sum, ok := itSysByKey[key]
		if !ok {
			name := itSysNameByKey[key]
			if name == "" {
				name = key
			}
			sum = &utilItSysSummary{Key: key, Name: name}
			itSysByKey[key] = sum
			itSysKeys = append(itSysKeys, key)
		}
		return sum
	}

	// --- Блок "Виртуальная инфраструктура" - СРЕДНЕЕ ЗА ПЕРИОД по всем
	// суточным записям VMRecord в выбранном диапазоне дат (не последняя
	// запись - ИЗМЕНЕНО 2026-09-18 по прямому запросу пользователя "нужно
	// усреднение за период"), по ServerId (тому же справочнику
	// ServerSprRecords, что и на /sprsrv/utildatacenter, НЕ по VMSprId - у
	// одной VM может не быть VMSprId, но ServerId всегда проставляется при
	// импорте). Ёмкость VM (vCPU/ГБ) для подписи берётся из САМОЙ СВЕЖЕЙ
	// записи периода (она статична, "усреднять" её нет смысла) - т.к. в
	// выборке уже отсортировано по -Date, это первая встреченная запись
	// каждой VM, см. vmAccum.capSet ниже. ---
	if len(vmSprById) > 0 {
		vmServerIds := make([]int64, 0, len(vmSprById))
		for id := range vmSprById {
			vmServerIds = append(vmServerIds, id)
		}
		var vmRecs []models.VMRecord
		o.QueryTable(new(models.VMRecord)).
			Filter("ServerId__in", vmServerIds).
			Filter("Date__gte", from).
			Filter("Date__lt", toExclusive).
			OrderBy("ServerId", "-Date").
			All(&vmRecs)

		type vmAccum struct {
			n                int
			cpuSum, ramSum   float64
			capSet           bool
			vCPU             int32
			memGb            float64
		}
		accByServer := map[int64]*vmAccum{}
		for _, r := range vmRecs {
			a, ok := accByServer[r.ServerId]
			if !ok {
				a = &vmAccum{}
				accByServer[r.ServerId] = a
			}
			a.n++
			a.cpuSum += r.Cpu_usage_average
			a.ramSum += r.Mem_usage_average
			if !a.capSet { // первая запись группы - самая свежая (сортировка -Date), берём ёмкость VM отсюда
				a.vCPU = r.Vm_NumCPU
				a.memGb = r.Vm_MemorySizeGB
				a.capSet = true
			}
		}

		for serverId, a := range accByServer {
			if a.n == 0 {
				continue
			}
			spr, ok := vmSprById[serverId]
			if !ok {
				continue
			}
			sum := getSys(spr.ITSystemKey)
			name := spr.Name
			if name == "" {
				name = spr.NameFQDN
			}
			avgCpu := a.cpuSum / float64(a.n)
			avgRam := a.ramSum / float64(a.n)
			detail := fmt.Sprintf("%s · среднее по %d записям за период", fmtVCpuMem(a.vCPU, a.memGb), a.n)
			sum.VM.Servers = append(sum.VM.Servers, utilItSysServer{Name: name, CpuPct: avgCpu, RamPct: avgRam, Detail: detail})
		}
	}

	// --- Блок "Аппаратные серверы" - СРЕДНЕЕ ЗА ПЕРИОД по всем записям
	// models.ServerMetric (поминутный "сырой" снимок из Zabbix, ПРЯМО указан
	// пользователем как источник для этого блока) в выбранном диапазоне дат
	// (ИЗМЕНЕНО 2026-09-18, та же причина, что у блока VM выше - "нужно
	// усреднение за период", а не последний замер). При большом диапазоне
	// дат это может быть много строк на хост (запись каждую минуту) -
	// усредняется всё целиком в Go, без учёта их числа для отчёта это
	// приемлемо (аналогично тому, как cmdb/saveservermetricrecord.go уже
	// делает суточное усреднение ServerMetric -> ServerMetricRecord, только
	// здесь усреднение "по требованию" за произвольный период, а не за
	// сутки). Связь (Source, HostId) -> ServerSprRecords - через
	// ServerSprRecords.ZabbixHostId/ScomHostId (обратный индекс
	// hwSprIdByHost, собран выше; с 2026-10-01 серверы без Zabbix берутся
	// из SCOM). ---
	type metricAccum struct {
		n              int
		cpuSum, ramSum float64
	}
	for source, byHostId := range hwSprIdByHost {
		if len(byHostId) == 0 {
			continue
		}
		hostIds := make([]int64, 0, len(byHostId))
		for hostId := range byHostId {
			hostIds = append(hostIds, hostId)
		}
		var metricRecs []models.ServerMetric
		o.QueryTable(new(models.ServerMetric)).
			Filter("Source", source).
			Filter("HostId__in", hostIds).
			Filter("CollectedAt__gte", from).
			Filter("CollectedAt__lt", toExclusive).
			OrderBy("HostId", "-CollectedAt").
			All(&metricRecs)

		accByHost := map[int64]*metricAccum{}
		for _, r := range metricRecs {
			a, ok := accByHost[r.HostId]
			if !ok {
				a = &metricAccum{}
				accByHost[r.HostId] = a
			}
			a.n++
			a.cpuSum += r.CPUUtilPercent
			a.ramSum += r.RAMUtilPercent
		}

		sourceLabel := "Zabbix"
		if source == models.MetricSourceSCOM {
			sourceLabel = "SCOM"
		}
		for hostId, a := range accByHost {
			if a.n == 0 {
				continue
			}
			serverId, ok := byHostId[hostId]
			if !ok {
				continue
			}
			spr, ok := hwSprById[serverId]
			if !ok {
				continue
			}
			sum := getSys(spr.ITSystemKey)
			name := spr.Name
			if name == "" {
				name = spr.NameFQDN
			}
			avgCpu := a.cpuSum / float64(a.n)
			avgRam := a.ramSum / float64(a.n)
			detail := fmt.Sprintf("%s · среднее по %d замерам за период", sourceLabel, a.n)
			sum.HW.Servers = append(sum.HW.Servers, utilItSysServer{Name: name, CpuPct: avgCpu, RamPct: avgRam, Detail: detail})
		}
	}

	sort.Strings(itSysKeys)

	for _, sum := range itSysByKey {
		sum.VM.Count = len(sum.VM.Servers)
		sum.VM.CpuPct = avgPct(sum.VM.Servers, func(s utilItSysServer) float64 { return s.CpuPct })
		sum.VM.RamPct = avgPct(sum.VM.Servers, func(s utilItSysServer) float64 { return s.RamPct })
		sort.Slice(sum.VM.Servers, func(i, j int) bool { return sum.VM.Servers[i].Name < sum.VM.Servers[j].Name })

		sum.HW.Count = len(sum.HW.Servers)
		sum.HW.CpuPct = avgPct(sum.HW.Servers, func(s utilItSysServer) float64 { return s.CpuPct })
		sum.HW.RamPct = avgPct(sum.HW.Servers, func(s utilItSysServer) float64 { return s.RamPct })
		sort.Slice(sum.HW.Servers, func(i, j int) bool { return sum.HW.Servers[i].Name < sum.HW.Servers[j].Name })
	}

	if b, err := json.Marshal(itSysByKey); err == nil {
		this.Data["ItSysDataJSON"] = template.JS(b)
	}
	if b, err := json.Marshal(itSysKeys); err == nil {
		this.Data["ItSysNamesJSON"] = template.JS(b)
	}
	this.Data["ItSysCount"] = len(itSysKeys)
}

// avgPct - простое среднее по срезу серверов блока (0, если срез пуст) -
// см. комментарий у utilItSysBlock про то, почему не взвешенное.
func avgPct[T any](items []T, fn func(T) float64) float64 {
	if len(items) == 0 {
		return 0
	}
	sum := 0.0
	for _, it := range items {
		sum += fn(it)
	}
	return sum / float64(len(items))
}

// fmtVCpuMem - короткая строка "N vCPU, M ГБ" для подписи VM в списке.
func fmtVCpuMem(numCPU int32, memGb float64) string {
	return fmt.Sprintf("%d vCPU, %.0f ГБ", numCPU, memGb)
}
