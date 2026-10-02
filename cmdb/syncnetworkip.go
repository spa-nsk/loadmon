// Package cmdb (файл syncnetworkip.go) - автосинхронизация справочников сетей
// (models.NetworkSprRecords) и IP-адресов (models.IPAddressSprRecords) из
// текстовых полей models.ServerSprRecords.Network/IPAddress/MacAddress.
//
// Заведено 2026-09-16 по запросу пользователя "на основе данных из
// справочника серверов по полю Network построй справочник сетей и отдельно
// создай справочник IP адресов и свяжи с сетями. В справочнике IP адресов
// помимо IP храни MAC адрес. делай автообновление 1 раз в сутки после
// сохранения метрик" - "после сохранения метрик" в итоге реализовано как
// отдельный независимый суточный таймер (см. metrics_grpc/metrics_grpc.go,
// tickerNetworkIP), а не как хвост конкретного цикла сбора метрик - так он
// не зависит от порядка/успешности отработки разных источников (VMware/HPE
// OneView/Zabbix), пишущих в ServerSprRecords по своим независимым таймерам.
package cmdb

import (
	"log"
	"loadmon/models"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// SyncNetworkAndIPDirectories - точка входа автосинхронизации, вызывается
// суточным тикером (см. metrics_grpc.go). Делает две вещи:
//
//  1. Справочник СЕТЕЙ (models.NetworkSprRecords, тот же, что редактируется
//     вручную на /sprsrv/networklist) ТОЛЬКО ДОПОЛНЯЕТСЯ - для каждого
//     уникального непустого значения из ServerSprRecords.Network (список
//     через ";", см. комментарий поля), которого ещё нет в справочнике по
//     Name, заводится новая запись с пустыми Vlan/Subnet/Gateway/Annotation
//     (дозаполняются администратором вручную при необходимости) - уже
//     существующие записи и их вручную заполненные поля НЕ трогаются и не
//     удаляются, даже если соответствующее значение больше не встречается
//     ни у одного сервера (сеть могла временно опустеть, но остаться
//     значимой).
//
//  2. Справочник IP-АДРЕСОВ (models.IPAddressSprRecords) - ЦЕЛИКОМ производная
//     таблица, у нее нет формы редактирования - пересобирается С НУЛЯ
//     (полное удаление + вставка) при каждом запуске: для каждой активной
//     записи ServerSprRecords (кроме помеченных к уничтожению, Stat=666) разбираются списки IPAddress/
//     MacAddress/Network (все - через ";") и сопоставляются ПО ПОЗИЦИИ В
//     СПИСКЕ, только если длины списков совпадают (гарантированного порядка/
//     соответствия элементов друг другу в исходных текстовых полях нет -
//     это лучшее доступное приближение без более точного источника данных).
//     Если длины не совпадают - IP всё равно заводится в справочник, но
//     MacAddress/NetworkId у него остаются пустыми/0.
func SyncNetworkAndIPDirectories() {
	start := time.Now().Local()
	log.Println(start.Format("2006-01-02 15:04:05"), "SyncNetworkAndIPDirectories: запуск автосинхронизации справочников сетей и IP-адресов")

	o := orm.NewOrm()

	var records []models.ServerSprRecords
	if _, err := models.ServerSprQS(o).
		All(&records, "Id", "Name", "Network", "IPAddress", "MacAddress"); err != nil {
		log.Println("SyncNetworkAndIPDirectories: ошибка чтения server_spr_records:", err)
		return
	}

	networkIdByName, err := ensureNetworkDirectoryEntries(o, records)
	if err != nil {
		log.Println("SyncNetworkAndIPDirectories: ошибка дополнения справочника сетей:", err)
		return
	}

	if err := rebuildIPAddressDirectory(o, records, networkIdByName); err != nil {
		log.Println("SyncNetworkAndIPDirectories: ошибка пересборки справочника IP-адресов:", err)
		return
	}

	log.Println(time.Now().Local().Format("2006-01-02 15:04:05"), "SyncNetworkAndIPDirectories: автосинхронизация завершена за", time.Since(start).String())
}

// splitListField разбирает текстовое поле-список ("vlan144;vlan200" и т.п.)
// на непустые обрезанные пробелами элементы - общий приём для Network/
// IPAddress/MacAddress на ServerSprRecords (см. комментарии этих полей -
// "список через разделитель ;").
func splitListField(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ";")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// ensureNetworkDirectoryEntries собирает множество уникальных имён сетей из
// ServerSprRecords.Network по всем переданным записям, заводит в
// models.NetworkSprRecords недостающие (по имени, регистронезависимо) и
// возвращает карту "имя сети (как оно встретилось у сервера) -> Id записи
// справочника" для дальнейшей привязки IP-адресов.
func ensureNetworkDirectoryEntries(o orm.Ormer, records []models.ServerSprRecords) (map[string]int64, error) {
	var existing []models.NetworkSprRecords
	if _, err := o.QueryTable(new(models.NetworkSprRecords)).All(&existing, "Id", "Name"); err != nil {
		return nil, err
	}

	idByLowerName := make(map[string]int64, len(existing))
	for _, n := range existing {
		idByLowerName[strings.ToLower(strings.TrimSpace(n.Name))] = n.Id
	}

	now := time.Now().Local()
	newCount := 0
	for _, rec := range records {
		for _, name := range splitListField(rec.Network) {
			key := strings.ToLower(name)
			if _, ok := idByLowerName[key]; ok {
				continue
			}
			nw := models.NetworkSprRecords{
				Name: name,
				// Id_User=9999 - тот же приём, что и в остальных местах
				// проекта для записей, заведённых автоматикой, а не
				// вручную конкретным пользователем (см. например
				// resolveDatacenterSiteByName в itilliumimport).
				Id_User:  9999,
				Time_Mfy: now,
				Annotation: "Заведено автоматически синхронизацией со справочником серверов " +
					"(SyncNetworkAndIPDirectories) по значению поля \"Сеть\" (Network)",
			}
			if _, err := o.Insert(&nw); err != nil {
				return nil, err
			}
			idByLowerName[key] = nw.Id
			newCount++
		}
	}
	if newCount > 0 {
		log.Printf("SyncNetworkAndIPDirectories: в справочник сетей добавлено новых записей: %d", newCount)
	}

	// Карта в исходном (не приведённом к нижнему регистру) написании имени -
	// строится из idByLowerName, но ключи нужны в том виде, в каком имя
	// встретится у КОНКРЕТНОГО сервера при сборке IP-адресов ниже, поэтому
	// возвращаем карту "lower(name) -> Id" - вызывающая сторона сама
	// приводит имя к нижнему регистру перед поиском (см.
	// rebuildIPAddressDirectory).
	return idByLowerName, nil
}

// rebuildIPAddressDirectory полностью пересобирает models.IPAddressSprRecords
// (производная таблица без формы редактирования - см. комментарий
// SyncNetworkAndIPDirectories) по актуальному состоянию переданных записей
// ServerSprRecords.
func rebuildIPAddressDirectory(o orm.Ormer, records []models.ServerSprRecords, networkIdByLowerName map[string]int64) error {
	trn, err := o.Begin()
	if err != nil {
		return err
	}

	if _, err := trn.QueryTable(new(models.IPAddressSprRecords)).Filter("Id__gt", 0).Delete(); err != nil {
		trn.Rollback()
		return err
	}

	now := time.Now().Local()
	inserted := 0
	for _, rec := range records {
		ips := splitListField(rec.IPAddress)
		if len(ips) == 0 {
			continue
		}
		macs := splitListField(rec.MacAddress)
		nets := splitListField(rec.Network)

		macByIndex := func(i int) string {
			if len(macs) == len(ips) {
				return macs[i]
			}
			return ""
		}
		networkIdByIndex := func(i int) int64 {
			if len(nets) != len(ips) {
				return 0
			}
			return networkIdByLowerName[strings.ToLower(nets[i])]
		}

		for i, ip := range ips {
			row := models.IPAddressSprRecords{
				IPAddress:   ip,
				MacAddress:  macByIndex(i),
				NetworkId:   networkIdByIndex(i),
				ServerSprId: rec.Id,
				ServerName:  rec.Name,
				Time_Mfy:    now,
			}
			if _, err := trn.Insert(&row); err != nil {
				trn.Rollback()
				return err
			}
			inserted++
		}
	}

	if err := trn.Commit(); err != nil {
		return err
	}
	log.Printf("SyncNetworkAndIPDirectories: справочник IP-адресов пересобран, записей: %d", inserted)
	return nil
}
