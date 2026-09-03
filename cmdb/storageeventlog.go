package cmdb

import (
	"fmt"
	models "loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// StorageEventLogRetention - срок хранения журнала событий СХД
// (models.StorageEventLog: доступность опроса и обнаружение неизвестных
// LUN) - 3 суток, по прямому указанию пользователя (2026-09-03). См.
// ClearOldStorageEventLog ниже.
const StorageEventLogRetention = time.Hour * 24 * 3

// logStorageAvailability записывает в журнал storage_event_log результат
// ОЧЕРЕДНОЙ попытки опроса конкретной СХД (успех/ошибка) - в отличие от
// StorageSystems.LastScanTime/LastScanOK/LastScanError (см.
// recordStorageSystemScanResult в cmdb/volumestorage.go), которые хранят
// только последнее значение, здесь копится история за
// StorageEventLogRetention. Вызывается из того же места, что и
// recordStorageSystemScanResult - см. UpdateVolumeStorage.
func logStorageAvailability(o orm.Ormer, shd *models.StorageSystems, pollErr error) {
	entry := &models.StorageEventLog{
		Time:              time.Now().Local(),
		EventType:         models.StorageEventAvailability,
		StorageSystemsId:  shd.Id,
		Key:               shd.ShortName,
		OK:                pollErr == nil,
	}
	if pollErr != nil {
		entry.Message = pollErr.Error()
	}
	if _, err := o.Insert(entry); err != nil {
		fmt.Println("Ошибка записи журнала доступности СХД:", shd.ShortName, err)
	}
}

// logUnknownLun записывает в журнал storage_event_log факт обнаружения
// LUN (тома), который не удалось привязать ни к одной известной СХД (см.
// вызов в UpdateVolumeStorage - место, где GetShortNameStoreSys/
// GetWWNMaskKey не смогли определить StorageSystemsId для тома). Ключ
// дедупликации - UID/StorageName/VirtualName тома: если для этого же LUN
// уже есть запись StorageEventUnknownLun в пределах срока хранения,
// повторная запись не создаётся (чтобы не заспамить журнал одним и тем же
// неопознанным LUN на каждом суточном цикле опроса).
func logUnknownLun(o orm.Ormer, vol *models.StorageVolumes) {
	key := vol.UID
	if key == "" {
		key = vol.StorageName
	}
	if key == "" {
		key = vol.VirtualName
	}
	if key == "" {
		return // нечем идентифицировать том - логировать бессмысленно
	}

	exist := models.StorageEventLog{}
	err := o.QueryTable("storage_event_log").
		Filter("EventType", models.StorageEventUnknownLun).
		Filter("Key", key).
		One(&exist)
	if err == nil {
		return // уже залогировано в пределах срока хранения
	}

	entry := &models.StorageEventLog{
		Time:      time.Now().Local(),
		EventType: models.StorageEventUnknownLun,
		Key:       key,
		Message: fmt.Sprintf(
			"LUN не привязан ни к одной известной СХД: UID=%q StorageName=%q VirtualName=%q Vcenter=%q Cluster=%q",
			vol.UID, vol.StorageName, vol.VirtualName, vol.Vcenter, vol.Cluster),
	}
	if _, err := o.Insert(entry); err != nil {
		fmt.Println("Ошибка записи журнала неизвестных LUN:", key, err)
	}
}

// ClearOldStorageEventLog удаляет из таблицы storage_event_log записи
// старше StorageEventLogRetention (3 суток) - тот же паттерн порционного
// удаления, что и cmdb.ClearOldMetric (см.
// cmdb/savemetric_orchestration.go).
func ClearOldStorageEventLog() {
	cutoff := time.Now().Local().Add(-StorageEventLogRetention)
	o := orm.NewOrm()
	count := int64(0)
	for {
		res, err := o.Raw("DELETE FROM storage_event_log WHERE id IN (SELECT id FROM storage_event_log WHERE time <=? LIMIT 1000)", cutoff).Exec()
		if err != nil {
			fmt.Println("Ошибка удаления старых записей из таблицы storage_event_log", err)
			break
		}
		num, _ := res.RowsAffected()
		count += num
		if num == 0 {
			break
		}
	}
	fmt.Println("Удалено :", count, "записей старее", StorageEventLogRetention.String(), "из таблицы storage_event_log")
}
