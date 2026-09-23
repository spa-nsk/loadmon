package storages

import (
	"encoding/json"
	"fmt"
	"log"
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

// DumpUnityMetricCatalog возвращает "сырой" (как есть) JSON-ответ Unity на
// запрос каталога доступных метрик (тип metric, GET
// /api/types/metric/instances) - по аналогии с тем, как для Huawei Dorado
// ранее снимался сырой дамп GET /disk/{id} (см. kontext/10-huawei-dorado-rest-api.md,
// CHANGELOG (174)) для проверки реальной схемы ответа перед реализацией
// разбора. Добавлено 2026-09-22 в рамках починки Unity KPI (см.
// cmdb/collectstoragemetrics.go, комментарий у case 2 в
// collectStorageMetricForSystem) - чтобы для конкретного массива (модель/
// версия Unity OE) можно было получить ДОСТОВЕРНЫЙ список путей метрик
// (path/isHistoricalAvailable/isRealtimeAvailable) вместо предположений по
// сторонним источникам, прежде чем реализовывать сбор IOPS/throughput/
// latency/CPU. Поля fields/filter отдаются пустыми - получаем весь каталог
// целиком, как он есть на этой конкретной СХД.
func DumpUnityMetricCatalog(address string) (json.RawMessage, error) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, err
	}
	defer session.CloseSession()

	var raw json.RawMessage
	if err := session.Request("GET", "/api/types/metric/instances", "", "", true, nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
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
