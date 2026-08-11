package vspheremetrics

import (
	"log"

	"time"

	"errors"
	"sync/atomic"
)

func (infra *VMInfrastructure) StartVcenterTimer() {
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		vc.Ticker = time.NewTicker(time.Second * time.Duration(vc.Wait))
	}
	infra.Ticker = time.NewTicker(time.Hour * time.Duration(1))
	infra.Done = make(chan bool)
}

// функция пока не используется
func (infra *VMInfrastructure) CalcMetricPerHour() {
	for {
		select {
		case <-infra.Done:
			return
		case <-infra.Ticker.C:
			atomic.StoreUint64(&infra.MetricsPerHour, atomic.LoadUint64(&infra.CountMetrics))
			atomic.StoreUint64(&infra.CountMetrics, 0)
		}
	}
}

// StopVcenterTimer останавливает тикеры опроса для vCenter и инфраструктуры, у которых ещё есть активные потоки (CountThread > 0).
func (infra *VMInfrastructure) StopVcenterTimer() {
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		if atomic.LoadInt64(&vc.CountThread) > 0 {
			vc.Ticker.Stop()
		}
	}
	if atomic.LoadInt64(&infra.CountThread) > 0 {
		infra.Ticker.Stop()
	}
}

// SetDoneVcenter останавливает таймеры опроса и подаёт сигнал остановки (DoneFl)
// всем активным обработчикам опроса vCenter.
func (infra *VMInfrastructure) SetDoneVcenter() {
	log.Println("Остановка таймеров обработчиков метрик")
	infra.StopVcenterTimer()
	log.Println("Таймеры обработчиков метрик остановлены")
	count := infra.VCenterMapList.Count()
	for i, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			log.Println("Ошибка поиска vcenter:", key)
			continue
		}
		if atomic.LoadInt64(&vc.CountThread) > 0 {
			log.Println("Передача сигнала остановки обработчика: ", vc.Hostname, i, "из", count)
			atomic.StoreInt32(&vc.DoneFl, 1)
			log.Println("Сигнал остановки обработчика:", vc.Hostname, "передан")
		}
	}
}

// GetVCenter возвращает запись VCenter по имени хоста.
func (infra *VMInfrastructure) GetVCenter(hostname string) (VCenter, error) {

	if infra == nil {
		return VCenter{}, errors.New(vminfraNotInitialised)
	}
	vc, ok := infra.VCenterMapList.Get(hostname)
	if ok {
		return *vc, nil
	}
	return VCenter{}, errors.New(hostname + " не найден.")
}

// возвращает максимальное значение из среза
// сейчас сделана корректировка если длинна среза равна 2, то возвращает 1 значение
func maxValue(value []int64) int64 {
	start := true
	max := int64(0)
	for i := range value {
		if start {
			start = false
			max = value[i]
			continue
		}
		if max < value[i] {
			max = value[i]
		}
	}
	return max
}
