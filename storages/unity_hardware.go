package storages

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"loadmon/emcstorage"
	datamap "loadmon/utilities"
)

const unityLogin = "spaview"

// openUnitySession открывает REST-сессию Unity (HTTPS, при ошибке — HTTP).
func openUnitySession(address string) (*emcstorage.Session, error) {
	pass := datamap.GetKeyData(unityLogin)
	session, err := emcstorage.NewSession(address, true, unityLogin, pass)
	if err != nil {
		session, err = emcstorage.NewSession(address, false, unityLogin, pass)
	}
	return session, err
}

// GetUnityDPE возвращает блоки DPE (полки управления) Unity.
func GetUnityDPE(address string) ([]emcstorage.DPEContent, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()
	resp, err := session.GetDPE()
	if err != nil {
		return nil, err
	}
	out := make([]emcstorage.DPEContent, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		out = append(out, e.Content)
	}
	return out, nil
}

// GetUnityDAE возвращает полки DAE (данные) Unity.
func GetUnityDAE(address string) ([]emcstorage.DAEContent, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()
	resp, err := session.GetDAE()
	if err != nil {
		return nil, err
	}
	out := make([]emcstorage.DAEContent, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		out = append(out, e.Content)
	}
	return out, nil
}

// GetUnityDisks возвращает физические диски Unity.
func GetUnityDisks(address string) ([]emcstorage.DiskContent, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()
	resp, err := session.GetDisks()
	if err != nil {
		return nil, err
	}
	out := make([]emcstorage.DiskContent, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		out = append(out, e.Content)
	}
	return out, nil
}

// GetUnityPools возвращает пулы хранения Unity (2026-09-16, добавлено по
// запросу пользователя loadmon "для VNX/Unity/Lenovo тоже нужно" (обработка
// пулов) - session.GetPool() уже существовал в emcstorage, но использовался
// только в закомментированном экспериментальном коде (storages/unity.go:
// InitData); теперь заведён отдельным обёрточным методом по образцу
// GetUnityDPE/GetUnityDAE/GetUnityDisks для использования в
// cmdb.syncUnityHardware).
func GetUnityPools(address string) ([]emcstorage.PoolContent, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()
	resp, err := session.GetPool()
	if err != nil {
		return nil, err
	}
	out := make([]emcstorage.PoolContent, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		out = append(out, e.Content)
	}
	return out, nil
}

// unityMetricCatalogFields - явный список полей для GET
// /api/types/metric/instances. ⚠️ ОБЯЗАТЕЛЕН (2026-09-23, найдено по
// реальному дампу пользователя `unity-metric-catalog1.json`): emcstorage.Request
// всегда добавляет `compact=true`, а Unity REST API в compact-режиме БЕЗ
// явного `fields` отдаёт для каждой записи коллекции ТОЛЬКО атрибут `id` -
// весь дамп 2026-09-22 оказался бесполезен для поиска реальных путей
// метрик именно по этой причине (раньше это не было известно - структура
// ответа не была ещё проверена сырым дампом). См.
// kontext/12-unity-rest-api.md.
const unityMetricCatalogFields = "id,name,path,type,description,isHistoricalAvailable,isRealtimeAvailable,unitDisplayString"

// unityCollectionEnvelope - минимальный разбор обёртки коллекции Unity REST
// API, нужный только для обхода пагинации (`links`/`entries`) - содержимое
// `entries` не типизируется (остаётся `json.RawMessage`), чтобы не терять
// поля при последующей сериализации в файл.
type unityCollectionEnvelope struct {
	Base    string `json:"@base"`
	Updated string `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []json.RawMessage `json:"entries"`
}

// DumpUnityMetricCatalog возвращает "сырой" (как есть, но объединённый по
// всем страницам) JSON-ответ Unity на запрос каталога доступных метрик
// (тип metric, GET /api/types/metric/instances) - по аналогии с тем, как
// для Huawei Dorado ранее снимался сырой дамп GET /disk/{id} (см.
// kontext/10-huawei-dorado-rest-api.md, CHANGELOG (174)) для проверки
// реальной схемы ответа перед реализацией разбора. Добавлено 2026-09-22 в
// рамках починки Unity KPI (см. cmdb/collectstoragemetrics.go, комментарий
// у case 2 в collectStorageMetricForSystem) - чтобы для конкретного
// массива (модель/версия Unity OE) можно было получить ДОСТОВЕРНЫЙ список
// путей метрик (path/isHistoricalAvailable/isRealtimeAvailable) вместо
// предположений по сторонним источникам, прежде чем реализовывать сбор
// IOPS/throughput/latency/CPU.
//
// ⚠️ ИСПРАВЛЕНО 2026-09-23 (было скрытым багом с момента появления этой
// функции): раньше `fields`/`filter` отдавались пустыми - в реальном дампе
// пользователя (`unity-metric-catalog1.json`) это дало 2000 записей вида
// `{"content": {"id": 1}}` БЕЗ единого поля `path`/`name` - т.е. каталог
// был снят, но оказался бесполезен для поиска реальных путей метрик.
// Причина - Unity REST API в compact-режиме (который `emcstorage.Request`
// включает всегда) без явного `fields` возвращает только `id`. Теперь
// запрашиваются явные поля (`unityMetricCatalogFields`). Также обнаружено,
// что коллекция превышает один "page" (в том же дампе - `"links":
// [{"rel": "next", "href": "&page=2"}]`) - раньше вторая и последующие
// страницы вообще не запрашивались, теперь функция обходит все страницы
// через `session.RequestPage` и объединяет `entries` в одну коллекцию.
func DumpUnityMetricCatalog(address string) (json.RawMessage, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()

	var allEntries []json.RawMessage
	page := 1
	for {
		var env unityCollectionEnvelope
		if err := session.RequestPage("GET", "/api/types/metric/instances", unityMetricCatalogFields, "", true, page, nil, &env); err != nil {
			return nil, fmt.Errorf("страница %d: %w", page, err)
		}
		allEntries = append(allEntries, env.Entries...)

		hasNext := false
		for _, l := range env.Links {
			if l.Rel == "next" {
				hasNext = true
				break
			}
		}
		if !hasNext {
			break
		}
		page++
		if page > 50 {
			// Защита от зацикливания, если сервер вернёт "next" бесконечно -
			// реальных каталогов метрик такого размера не бывает (см. дамп -
			// первая страница уже 2000 записей).
			log.Printf("storages.DumpUnityMetricCatalog(%s): остановлено на странице %d (защита от зацикливания)", address, page)
			break
		}
	}

	merged := struct {
		Entries []json.RawMessage `json:"entries"`
	}{Entries: allEntries}
	return json.Marshal(merged)
}

// UnityHardwareBundle - результат GetUnityHardwareBundle: полки/диски/пулы
// одной СХД Unity, полученные в рамках ОДНОЙ REST-сессии.
type UnityHardwareBundle struct {
	DPE      []emcstorage.DPEContent
	DAE      []emcstorage.DAEContent
	Disks    []emcstorage.DiskContent
	Pools    []emcstorage.PoolContent
	PoolsErr error // ошибка запроса пулов не прерывает получение остального (см. cmdb.syncUnityHardware)
}

// GetUnityHardwareBundle получает полки (DPE/DAE), диски и пулы Unity в
// рамках ОДНОЙ REST-сессии, вместо четырёх отдельных
// login/logout-циклов (GetUnityDPE+GetUnityDAE+GetUnityDisks+GetUnityPools
// по отдельности). Добавлено 2026-09-17 по жалобе пользователя loadmon
// "диски/полки/пулы не появились" - при разборе присланного
// storage_update.log обнаружилось, что большинство СХД Unity в это время
// массово возвращали HTTP 422 на не связанные с этой задачей запросы
// (GetHostsRest/GetLunsRest в cmdb/volumestorage.go), то есть проблема с
// самими REST-сессиями Unity в этом окружении уже существовала независимо
// от добавления пулов - но синхронизация hardware для одной СХД до этой
// правки открывала ЧЕТЫРЕ отдельные сессии подряд (DPE, DAE, Disks, Pools -
// последняя добавлена в этой же серии правок), что увеличивало нагрузку на
// и без того нестабильно отвечающий REST-интерфейс массива. Сведение к
// одной сессии не устраняет первопричину HTTP 422 (она вне этого кода - см.
// CHANGELOG), но снижает число login/logout-обращений к каждому Unity в
// 4 раза за один цикл синхронизации hardware.
//
// Диски/полки (DPE/DAE/Disks) сохраняют прежнее поведение "ошибка одного
// шага прерывает всю функцию" (как было у отдельных GetUnityDPE/DAE/Disks -
// syncUnityHardware исторически считал их фатальными). Пулы, как и раньше
// (см. syncUnityHardware), НЕ фатальны - ошибка кладётся в PoolsErr, и
// вызывающий код сам решает, логировать её как "не прерывает синхронизацию"
// и пропустить деактивацию пулов.
func GetUnityHardwareBundle(address string) (*UnityHardwareBundle, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()

	var b UnityHardwareBundle

	dpeResp, err := session.GetDPE()
	if err != nil {
		return nil, err
	}
	for _, e := range dpeResp.Entries {
		b.DPE = append(b.DPE, e.Content)
	}

	daeResp, err := session.GetDAE()
	if err != nil {
		return nil, err
	}
	for _, e := range daeResp.Entries {
		b.DAE = append(b.DAE, e.Content)
	}

	disksResp, err := session.GetDisks()
	if err != nil {
		return nil, err
	}
	for _, e := range disksResp.Entries {
		b.Disks = append(b.Disks, e.Content)
	}

	poolsResp, poolsErr := session.GetPool()
	if poolsErr != nil {
		b.PoolsErr = poolsErr
	} else {
		for _, e := range poolsResp.Entries {
			b.Pools = append(b.Pools, e.Content)
		}
	}

	return &b, nil
}

// unityKpiLatestPerEntry возвращает последнее по времени значение KPI для каждого
// экземпляра path (например, отдельный SP в kpi.sp.*.utilization).
//
// ⚠️ НЕ ИСПОЛЬЗУЕТСЯ в cmdb с 2026-09-22 (см. cmdb/collectstoragemetrics.go,
// комментарий у case 2) - опрос через emcstorage.Session.GetkpiValue бьёт в
// REST-эндпоинт /api/types/kpiValue/instances, которого в Unity/Unisphere
// REST API не существует (подтверждённые Dell типы - metricValue для
// истории, metricRealTimeQuery/metricQueryResult для реального времени) -
// поэтому эта функция ВСЕГДА возвращала ok=false на реальном оборудовании
// (100% отказ по всем метрикам на всех Unity СХД). Раньше настоящая
// причина (err от openUnitySession/GetkpiValue) нигде не логировалась -
// теперь хотя бы логируется, чтобы при повторном включении сразу было
// видно точный HTTP-код/тело ответа. Прежде чем снова звать эту функцию из
// cmdb - нужно либо переписать её на настоящий metricValue/
// metricRealTimeQuery, либо снять реальный каталог метрик массива
// (/api/types/metric/instances, см. tools/storagemetrics -dumpmetricsonly)
// и свериться с ним - см. kontext/11-storage-metrics.md.
func unityKpiLatestPerEntry(address, path string) ([]float64, bool) {
	session, err := openUnitySession(address)
	if err != nil {
		log.Printf("storages.unityKpiLatestPerEntry(%s, %s): не удалось открыть сессию: %v", address, path, err)
		return nil, false
	}
	defer session.CloseSession()
	resp, err := session.GetkpiValue(path)
	if err != nil || resp == nil {
		log.Printf("storages.unityKpiLatestPerEntry(%s, %s): GetkpiValue: %v", address, path, err)
		return nil, false
	}
	out := make([]float64, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		var latest float64
		var latestTime time.Time
		var have bool
		for t, v := range e.Content.Values {
			if !have || t.After(latestTime) {
				latestTime = t
				latest = v
				have = true
			}
		}
		if have {
			out = append(out, latest)
		}
	}
	return out, len(out) > 0
}

// GetUnityKpiAverage — среднее последних значений KPI по всем экземплярам path.
func GetUnityKpiAverage(address, path string) (float64, bool) {
	vals, ok := unityKpiLatestPerEntry(address, path)
	if !ok {
		return 0, false
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals)), true
}

// GetUnityKpiSum — сумма последних значений KPI по всем экземплярам path (IOPS, MB/s).
func GetUnityKpiSum(address, path string) (float64, bool) {
	vals, ok := unityKpiLatestPerEntry(address, path)
	if !ok {
		return 0, false
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum, true
}

// DumpUnityRealTimeMetrics выполняет ПОЛНЫЙ подтверждённый документацией цикл
// опроса живых метрик Unity через РЕАЛЬНО существующий REST-механизм
// (в отличие от несуществующего /api/types/kpiValue/instances, см. комментарий
// у unityKpiLatestPerEntry выше):
//
//  1. POST /api/types/metricRealTimeQuery/instances {paths, interval} -
//     создаёт real-time-запрос (session.NewMetricRealTimeQuery)
//  2. ожидание wait (должно быть >= interval, иначе первый отсчёт может быть
//     ещё не готов)
//  3. GET /api/types/metricQueryResult/instances?filter=queryId eq <id> -
//     забирает отсчёт (session.GetMetricRealTimeQueryResult)
//  4. DELETE /api/instances/metricRealTimeQuery/<id> - удаляет запрос
//     (best-effort, ошибка только логируется - см. kontext, "session will be
//     deleted automatically" через 10 периодов, если не удалить вручную)
//
// Добавлено 2026-09-23 по запросу пользователя loadmon "добавил свежие
// описания по работе с СХД Huawei и Dell EMC... добавь в harness контекст и
// базу знаний. Используй для доработок SDK" - точная схема (включая пример
// "sp.*.cpu.summary.busyTicks"/"idleTicks" и структуру
// metricRealTimeQuery/metricQueryResult) подтверждена документом
// doc-API-hardware/unity-restapi-refg_en-us_reference_guide11-ru.md (см.
// kontext/12-unity-rest-api.md). Методы session.NewMetricRealTimeQuery/
// GetMetricRealTimeQueryResult/DeleteMetricRealTimeQuery (emcstorage/metrics.go)
// существовали в SDK и раньше, но не вызывались НИОТКУДА - вместо них
// ошибочно использовался несуществующий kpiValue.
//
// ⚠️ Это ДИАГНОСТИЧЕСКАЯ функция (по аналогии с DumpUnityMetricCatalog выше и
// cmdb.DumpHuaweiRawRequest для Huawei) - она НЕ подключена к
// cmdb.CollectStorageMetrics и ничего не пишет в storage_metric. Конкретные
// пути метрик для агрегированных IOPS/throughput/latency/utilization на
// уровне SP/пула (то, что раньше пытались получить через несуществующий
// kpi.sp.*.*) в документации текстом НЕ перечислены - только категории
// верхнего уровня (sp.*.storage, sp.*.physical, sp.*.cpu, ...) и два
// подтверждённых примера конкретных листовых путей: "sp.*.cpu.summary.busyTicks"/
// "idleTicks" (пример из документации) и "sp.*.physical.disk.*.responseTime"
// (пример фильтра для metricValue). Полный список актуальных для конкретной
// модели/прошивки Unity путей нужно снять с реального массива через
// -dumpmetricsonly (GET /api/types/metric/instances, уже реализовано) и
// свериться, прежде чем подставлять их сюда - "не гадать", тот же принцип,
// что уже приводил к находке несуществующего kpiValue. См.
// tools/storagemetrics -unitydumprealtime для проверки конкретных path на
// реальном оборудовании перед тем, как подключать к постоянному сбору.
func DumpUnityRealTimeMetrics(address string, paths []string, interval uint32, wait time.Duration) (*emcstorage.MetricRealTimeQueryResult, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()

	created, err := session.NewMetricRealTimeQuery(paths, interval)
	if err != nil {
		return nil, fmt.Errorf("создание real-time запроса: %w", err)
	}
	if created == nil || created.Content.ID == 0 {
		return nil, fmt.Errorf("СХД не вернула id real-time запроса")
	}
	queryID := created.Content.ID
	defer func() {
		if derr := session.DeleteMetricRealTimeQuery(queryID); derr != nil {
			log.Printf("storages.DumpUnityRealTimeMetrics(%s): не удалось удалить real-time запрос id=%d: %v", address, queryID, derr)
		}
	}()

	time.Sleep(wait)

	result, err := session.GetMetricRealTimeQueryResult(queryID)
	if err != nil {
		return nil, fmt.Errorf("получение результата real-time запроса (queryId=%d): %w", queryID, err)
	}
	return result, nil
}

// Пути живых KPI Unity (пространство "kpi." с wildcard "+", в отличие от
// низкоуровневого "sp."/"lun." и т.п. с wildcard "*" - это ДВА разных
// пространства путей одного REST API, оба встречаются в одном каталоге
// метрик) - ПОДТВЕРЖДЕНЫ реальным каталогом метрик пользователя
// (`unity-metric-catalog.json`, 2862 записи, снято 2026-09-23 с массива
// 10.2.19.106 после починки `-dumpmetricsonly`, см.
// kontext/12-unity-rest-api.md):
//
//	id 1022-1028: kpi.lun.+.sp.+.responseTime/readBandwidth/writeBandwidth/
//	              totalBandwidth/readIOPS/writeIOPS/totalIOPS (isRealtimeAvailable=true)
//	id 10693/10695: sp.*.cpu.summary.busyTicks/idleTicks (isRealtimeAvailable=true)
//
// "+" в kpi.lun.+.sp.+.* означает "по всем LUN, по всем SP" - в ответе
// приходит одна запись на path с агрегированной (по всем экземплярам)
// картой значений (см. sumFloatOrValueMap).
const (
	unityKpiLunReadIOPS      = "kpi.lun.+.sp.+.readIOPS"
	unityKpiLunWriteIOPS     = "kpi.lun.+.sp.+.writeIOPS"
	unityKpiLunTotalIOPS     = "kpi.lun.+.sp.+.totalIOPS"
	unityKpiLunReadBandwidth  = "kpi.lun.+.sp.+.readBandwidth"  // KB/s, подтверждено каталогом
	unityKpiLunWriteBandwidth = "kpi.lun.+.sp.+.writeBandwidth" // KB/s
	unityKpiLunTotalBandwidth = "kpi.lun.+.sp.+.totalBandwidth" // KB/s
	// unityKpiLunResponseTime - единый показатель задержки (не раздельно
	// read/write, в отличие от Huawei 384/385 - см.
	// kontext/10-huawei-dorado-rest-api.md). Единица измерения в
	// real-time-варианте пути каталогом НЕ указана (unitDisplayString
	// пусто) - у исторического варианта ТОГО ЖЕ пути/ID (isHistoricalAvailable
	// вариант, id 10) документирована как "microseconds" - GetUnityLiveKpi
	// использует то же предположение по аналогии, честно помечая это в
	// комментарии, не как подтверждённый факт.
	unityKpiLunResponseTime = "kpi.lun.+.sp.+.responseTime"

	unityRawCPUBusyTicks = "sp.*.cpu.summary.busyTicks"
	unityRawCPUIdleTicks = "sp.*.cpu.summary.idleTicks"
)

// unityLiveKpiInterval/unityLiveKpiWait - период сэмплирования (секунды
// metricRealTimeQuery) и время ожидания перед опросом результата для
// GetUnityLiveKpi. CPU% считается по ДЕЛЬТЕ между первым и последним
// отсчётом busyTicks/idleTicks (это накопительные счётчики тиков с
// момента запуска SP, а не готовый процент) - поэтому нужно МИНИМУМ 2
// отсчёта, отсюда wait существенно больше interval.
const (
	unityLiveKpiInterval = 5
	unityLiveKpiWait     = 11 * time.Second
)

// UnityLiveKpi - агрегированные (по всем LUN/SP массива) живые KPI Unity,
// см. GetUnityLiveKpi.
type UnityLiveKpi struct {
	ReadIOPS, WriteIOPS, TotalIOPS float64
	ReadMBps, WriteMBps, TotalMBps float64
	ResponseTimeUs                 float64
	ResponseTimeOK                 bool
	CPUUtilPercent                 float64
	CPUUtilOK                      bool
}

// sumFloatOrValueMap рекурсивно суммирует все числовые листья значения
// Unity floatOrValueMap (см. doc-API-hardware/unity-restapi-refg...md) -
// Values в ответе metricQueryResult может быть ПЛОСКОЙ map[string]interface{}
// (например {"spa": "3675787", "spb": "1520850"} - подтверждено реальным
// дампом, unity-realtime-dump2.json) или ВЛОЖЕННОЙ (документация
// допускает иерархию, например LUN -> SP -> значение, для путей с
// несколькими "+" в маске вроде kpi.lun.+.sp.+.*). Функция НЕ предполагает
// конкретную структуру ("не гадать", тот же принцип, что и везде в этом
// проекте) - просто складывает ВСЕ числа, найденные на любой глубине,
// возвращая сумму и количество учтённых листьев (n=0 - ничего не нашлось,
// путь пуст/не пришёл).
func sumFloatOrValueMap(v interface{}) (sum float64, n int) {
	switch t := v.(type) {
	case nil:
		return 0, 0
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil {
			return 0, 0
		}
		return f, 1
	case float64:
		return t, 1
	case map[string]interface{}:
		for _, vv := range t {
			s, c := sumFloatOrValueMap(vv)
			sum += s
			n += c
		}
		return sum, n
	default:
		return 0, 0
	}
}

// GetUnityLiveKpi запрашивает и агрегирует живые KPI Unity ОДНИМ real-time
// запросом (создание -> ожидание unityLiveKpiWait -> один опрос результата
// -> удаление, тот же механизм, что и у DumpUnityRealTimeMetrics выше):
//   - IOPS/пропускная способность/задержка - сумма (для IOPS/bandwidth) по
//     ВСЕМ LUN и SP массива (kpi.lun.+.sp.+.*, подтверждено каталогом
//     метрик пользователя 2026-09-23, см. константы выше)
//   - CPU % - через ДЕЛЬТУ первого и последнего отсчёта sp.*.cpu.summary.
//     busyTicks/idleTicks; если за unityLiveKpiWait успел прийти только
//     ОДИН отсчёт каждого - CPUUtilOK=false (недостаточно данных, не
//     ошибка).
//
// Добавлено 2026-09-23 на основе РЕАЛЬНОГО каталога метрик пользователя
// (2862 подтверждённых пути, массив 10.2.19.106) - см.
// kontext/12-unity-rest-api.md. Взамен прежнего отсутствия живых
// IOPS/throughput/latency/CPU для Unity (см. историю у case 2 в
// cmdb/collectstoragemetrics.go).
//
// ⚠️ Пути метрик подтверждены каталогом устройства, но сама агрегация
// (сумма по всем LUN, дельта тиков CPU) не сверена с независимым
// источником (например, значениями в Unisphere UI) - см. предупреждения у
// вызывающего кода (cmdb.collectUnityKpi).
func GetUnityLiveKpi(address string) (*UnityLiveKpi, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()

	out := &UnityLiveKpi{}
	var errs []string

	// ⚠️ 2026-09-23: раньше ОДИН real-time запрос сразу мешал в одном
	// paths[] пути из ДВУХ разных пространств Unity REST API - "kpi.*"
	// (курированные метрики) и низкоуровневые "sp.*" (сырые счётчики) - по
	// жалобе пользователя "по Unity записывается только UsedPercent,
	// ошибка HTTP status codes: 500" (real-time запрос падал целиком, обе
	// группы метрик терялись разом). Совмещать оба пространства в одном
	// запросе нигде документацией не подтверждено - "не гадать" (тот же
	// принцип, что и везде в проекте): вместо предположения "почему именно
	// 500" запросы РАЗДЕЛЕНЫ на два независимых real-time-цикла (каждый
	// свой create/wait/get/delete) - так ошибка одной группы (если причина
	// действительно в смешении пространств, или в чём-то ещё специфичном
	// для одной группы путей) не блокирует другую, и в PollError видно,
	// какая именно группа не удалась.
	lunPaths := []string{
		unityKpiLunReadIOPS, unityKpiLunWriteIOPS, unityKpiLunTotalIOPS,
		unityKpiLunReadBandwidth, unityKpiLunWriteBandwidth, unityKpiLunTotalBandwidth,
		unityKpiLunResponseTime,
	}
	if lunResult, err := runUnityRealtimeQuery(session, address, lunPaths, unityLiveKpiInterval, unityLiveKpiWait); err != nil {
		errs = append(errs, "IOPS/bandwidth/responseTime (kpi.lun.+.sp.+.*): "+err.Error())
	} else {
		latestByPath := map[string]float64{}
		latestTimeByPath := map[string]time.Time{}
		for _, e := range lunResult.Entries {
			c := e.Content
			sum, n := sumFloatOrValueMap(c.Values)
			if n == 0 {
				continue
			}
			if prevT, ok := latestTimeByPath[c.Path]; !ok || c.Timestamp.After(prevT) {
				latestByPath[c.Path] = sum
				latestTimeByPath[c.Path] = c.Timestamp
			}
		}
		out.ReadIOPS = latestByPath[unityKpiLunReadIOPS]
		out.WriteIOPS = latestByPath[unityKpiLunWriteIOPS]
		out.TotalIOPS = latestByPath[unityKpiLunTotalIOPS]
		out.ReadMBps = latestByPath[unityKpiLunReadBandwidth] / 1024
		out.WriteMBps = latestByPath[unityKpiLunWriteBandwidth] / 1024
		out.TotalMBps = latestByPath[unityKpiLunTotalBandwidth] / 1024
		if _, ok := latestTimeByPath[unityKpiLunResponseTime]; ok {
			out.ResponseTimeUs = latestByPath[unityKpiLunResponseTime]
			out.ResponseTimeOK = true
		} else {
			errs = append(errs, "responseTime: не получено ни одного отсчёта")
		}
	}

	cpuPaths := []string{unityRawCPUBusyTicks, unityRawCPUIdleTicks}
	if cpuResult, err := runUnityRealtimeQuery(session, address, cpuPaths, unityLiveKpiInterval, unityLiveKpiWait); err != nil {
		errs = append(errs, "CPU (sp.*.cpu.summary.*Ticks): "+err.Error())
	} else {
		type tick struct {
			t time.Time
			v float64
		}
		var busy, idle []tick
		for _, e := range cpuResult.Entries {
			c := e.Content
			sum, n := sumFloatOrValueMap(c.Values)
			if n == 0 {
				continue
			}
			switch c.Path {
			case unityRawCPUBusyTicks:
				busy = append(busy, tick{c.Timestamp, sum})
			case unityRawCPUIdleTicks:
				idle = append(idle, tick{c.Timestamp, sum})
			}
		}
		if len(busy) >= 2 && len(idle) >= 2 {
			sort.Slice(busy, func(i, j int) bool { return busy[i].t.Before(busy[j].t) })
			sort.Slice(idle, func(i, j int) bool { return idle[i].t.Before(idle[j].t) })
			dBusy := busy[len(busy)-1].v - busy[0].v
			dIdle := idle[len(idle)-1].v - idle[0].v
			if dBusy+dIdle > 0 {
				out.CPUUtilPercent = dBusy / (dBusy + dIdle) * 100
				out.CPUUtilOK = true
			} else {
				errs = append(errs, "CPU: сумма дельт busy/idle <= 0")
			}
		} else {
			errs = append(errs, "CPU: недостаточно отсчётов busyTicks/idleTicks для дельты")
		}
	}

	if len(errs) > 0 {
		return out, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}

// runUnityRealtimeQuery - общий цикл ОДНОГО real-time запроса (create ->
// wait -> get -> delete), вынесенный из GetUnityLiveKpi при разделении на
// независимые запросы по группам путей (см. комментарий выше). Тот же
// механизм, что и у DumpUnityRealTimeMetrics, но без открытия отдельной
// сессии (использует уже открытую session вызывающего кода - несколько
// real-time запросов подряд в рамках одного цикла опроса Unity СХД не
// должны открывать сессию заново на каждый).
func runUnityRealtimeQuery(session *emcstorage.Session, address string, paths []string, interval uint32, wait time.Duration) (*emcstorage.MetricRealTimeQueryResult, error) {
	created, err := session.NewMetricRealTimeQuery(paths, interval)
	if err != nil {
		return nil, fmt.Errorf("создание real-time запроса: %w", err)
	}
	if created == nil || created.Content.ID == 0 {
		return nil, fmt.Errorf("СХД не вернула id real-time запроса")
	}
	queryID := created.Content.ID
	defer func() {
		if derr := session.DeleteMetricRealTimeQuery(queryID); derr != nil {
			log.Printf("storages.runUnityRealtimeQuery(%s): не удалось удалить real-time запрос id=%d: %v", address, queryID, derr)
		}
	}()

	time.Sleep(wait)

	result, err := session.GetMetricRealTimeQueryResult(queryID)
	if err != nil {
		return nil, fmt.Errorf("получение результата real-time запроса (queryId=%d): %w", queryID, err)
	}
	if result == nil || len(result.Entries) == 0 {
		return nil, fmt.Errorf("СХД не вернула ни одного отсчёта метрик")
	}
	return result, nil
}
