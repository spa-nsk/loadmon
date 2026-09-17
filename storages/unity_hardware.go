package storages

import (
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
func unityKpiLatestPerEntry(address, path string) ([]float64, bool) {
	session, err := openUnitySession(address)
	if err != nil {
		return nil, false
	}
	defer session.CloseSession()
	resp, err := session.GetkpiValue(path)
	if err != nil || resp == nil {
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
