package vspheremetrics

// defaultBillingIntervalSec — интервал опроса (в секундах), используемый для
// расчёта Billing-полей, если реальный интервал конкретного vCenter
// определить не удалось (сам vCenter ещё не зарегистрирован в
// VCenterMapList на момент вызова, либо у него не задан Wait). Совпадает со
// значением, исторически зашитым в формулы CPUBilling/MemBilling (см.
// ниже) - используется как безопасный fallback, а не как "правильный"
// интервал по умолчанию.
const defaultBillingIntervalSec = 180.0

// billingIntervalSeconds возвращает реальный интервал опроса (в секундах)
// vCenter с именем vcenterHostname - тот же интервал, с которым запущен его
// тикер опроса (VCenter.Wait, см. vcenter_timer.go:StartVcenterTimer).
// Используется при расчёте Billing-полей (DiskBilling/NetBilling, см.
// комментарий к ReCalcVMPerfomance/SetVMMetric/SetHostMetric) - величина
// метрики "за один снимок" (KB/с) переводится в объём "за интервал между
// снимками", а не за фиксированный условный промежуток.
//
// Если vCenter не найден в глобальной инфраструктуре или его Wait <= 0
// (не задан/некорректная конфигурация) - возвращает defaultBillingIntervalSec.
func billingIntervalSeconds(vcenterHostname string) float64 {
	if vcenterHostname == "" {
		return defaultBillingIntervalSec
	}
	infra := GetGlobalVMInfrastructure()
	if infra == nil {
		return defaultBillingIntervalSec
	}
	vc, err := infra.GetVCenter(vcenterHostname)
	if err != nil || vc.Wait <= 0 {
		return defaultBillingIntervalSec
	}
	return float64(vc.Wait)
}
