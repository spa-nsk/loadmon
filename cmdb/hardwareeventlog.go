package cmdb

import (
	"fmt"
	models "loadmon/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// HardwareEventLogRetention - срок хранения журнала изменений аппаратной
// конфигурации (models.HardwareEventLog) - 30 суток. В отличие от
// storage_event_log (StorageEventLogRetention = 3 суток, там журналируется
// КАЖДАЯ попытка опроса), сюда пишется только сам факт РЕАЛЬНОГО изменения
// конфигурации - события заведомо более редкие и ценные для последующего
// разбора ("когда и что поменялось в сервере/корзине"), поэтому срок
// хранения намного больше. См. ClearOldHardwareEventLog ниже.
const HardwareEventLogRetention = time.Hour * 24 * 30

// componentTypeName - человекочитаемое имя типа компонента
// models.ServerHardwareComponents.Type для журнала и уведомлений.
func componentTypeName(t int) string {
	switch t {
	case ServerHWComponentRAM:
		return "Память (RAM)"
	case ServerHWComponentDisk:
		return "Диск"
	case ServerHWComponentNetwork:
		return "Сетевой порт"
	case ServerHWComponentFC:
		return "FC-порт"
	case ServerHWComponentRAID:
		return "RAID/storage-контроллер"
	case ServerHWComponentInterconnect:
		return "Interconnect-модуль корзины"
	case ServerHWComponentManagement:
		return "Management-модуль корзины (OA)"
	case ServerHWComponentPCIDevice:
		return "PCI-устройство"
	case ServerHWComponentFirmware:
		return "Прошивка"
	case ServerHWComponentProcessor:
		return "Процессор"
	case ServerHWComponentPowerSupply:
		return "Блок питания корзины"
	case ServerHWComponentFan:
		return "Вентилятор корзины"
	default:
		return fmt.Sprintf("Тип %d", t)
	}
}

// componentKey - ключ идентификации компонента между циклами синхронизации
// для diff'а (сравнения предыдущего и нового набора строк одного типа по
// одному серверу/корзине). Серийный номер - самый надёжный признак, если
// его нет (компонент без серийника) - используем слот+модель, чтобы не
// считать "новым" компонент, который просто не поменялся.
func componentKey(c models.ServerHardwareComponents) string {
	if c.SerialNum != "" {
		return c.SerialNum
	}
	return c.Slot + "|" + c.Model
}

// componentSummary - строка-снимок изменяемых характеристик компонента,
// используется, чтобы понять, поменялось ли у уже существующего компонента
// (тот же ключ) что-то существенное (модель/объём/слот), не заводя
// отдельную колонку под каждое поле.
func componentSummary(c models.ServerHardwareComponents) string {
	return fmt.Sprintf("Vendor=%s Model=%s CapacityGb=%.2f Slot=%s SubType=%d", c.Vendor, c.Model, c.CapacityGb, c.Slot, c.SubType)
}

// diffAndLogComponents сравнивает предыдущий набор строк
// models.ServerHardwareComponents данного типа (oldRows, прочитанные ДО
// удаления) с новым набором (newRows, только что построенным из свежих
// данных OneView) и пишет в hardware_event_log добавления/удаления/
// изменения. entityKind - "Server" или "BladeSystem" (для отображения в
// журнале), serverId/bladeSystemId - один из них >0, другой 0.
// entitySerial - серийный номер родителя (для читаемого Message).
func diffAndLogComponents(o orm.Ormer, entityKind string, serverId, bladeSystemId int64, entitySerial string, t int, oldRows, newRows []models.ServerHardwareComponents) {
	if len(oldRows) == 0 && len(newRows) == 0 {
		return
	}
	oldByKey := make(map[string]models.ServerHardwareComponents, len(oldRows))
	for _, r := range oldRows {
		oldByKey[componentKey(r)] = r
	}
	newByKey := make(map[string]models.ServerHardwareComponents, len(newRows))
	for _, r := range newRows {
		newByKey[componentKey(r)] = r
	}

	typeName := componentTypeName(t)
	kind := entityKind + "/" + typeName
	now := time.Now().Local()

	for key, nr := range newByKey {
		or, existed := oldByKey[key]
		if !existed {
			logHardwareEvent(o, &models.HardwareEventLog{
				Time: now, EventType: models.HardwareEventComponentAdded,
				EntityKind: kind, ServerId: serverId, BladeSystemId: bladeSystemId, Key: key,
				Message: fmt.Sprintf("%s: добавлен новый компонент \"%s\" (%s) у %s", typeName, nr.Model, key, entitySerial),
			})
			continue
		}
		if componentSummary(or) != componentSummary(nr) {
			logHardwareEvent(o, &models.HardwareEventLog{
				Time: now, EventType: models.HardwareEventComponentChanged,
				EntityKind: kind, ServerId: serverId, BladeSystemId: bladeSystemId, Key: key,
				OldValue: componentSummary(or), NewValue: componentSummary(nr),
				Message: fmt.Sprintf("%s: изменились характеристики компонента \"%s\" у %s: было [%s], стало [%s]", typeName, key, entitySerial, componentSummary(or), componentSummary(nr)),
			})
		}
	}
	for key, or := range oldByKey {
		if _, stillThere := newByKey[key]; !stillThere {
			logHardwareEvent(o, &models.HardwareEventLog{
				Time: now, EventType: models.HardwareEventComponentRemoved,
				EntityKind: kind, ServerId: serverId, BladeSystemId: bladeSystemId, Key: key,
				OldValue: componentSummary(or),
				Message:  fmt.Sprintf("%s: компонент \"%s\" (%s) у %s больше не обнаруживается", typeName, or.Model, key, entitySerial),
			})
		}
	}
}

// logHardwareFieldChange пишет в журнал единичное изменение скалярного
// поля ServerHardware/BladeSystems (например, RomVersion, Status, Model) -
// вызывается точечно там, где известны старое и новое значение конкретного
// поля (см. cmdb/insertHWSpr.go, cmdb/updatehwsprhp.go,
// cmdb/syncbladesystemcomponents.go). Пустой diff (oldValue==newValue) не
// логируется - вызывающий код не обязан сам это проверять.
func logHardwareFieldChange(o orm.Ormer, entityKind string, serverId, bladeSystemId int64, key, fieldName, oldValue, newValue string) {
	if oldValue == newValue {
		return
	}
	logHardwareEvent(o, &models.HardwareEventLog{
		Time: time.Now().Local(), EventType: models.HardwareEventFieldChanged,
		EntityKind: entityKind, ServerId: serverId, BladeSystemId: bladeSystemId, Key: key,
		FieldName: fieldName, OldValue: oldValue, NewValue: newValue,
		Message: fmt.Sprintf("%s %s: поле %s изменилось с %q на %q", entityKind, key, fieldName, oldValue, newValue),
	})
}

// logHardwareEvent - низкоуровневая запись строки в hardware_event_log +
// (best-effort, не блокирует синхронизацию при ошибке SMTP) отправка
// email-уведомления, если это включено в конфигурации
// (config.ConfigData.NotifyOnHardwareChange, см. cmdb/notify.go).
func logHardwareEvent(o orm.Ormer, entry *models.HardwareEventLog) {
	if _, err := o.Insert(entry); err != nil {
		fmt.Println("Ошибка записи журнала изменений аппаратной конфигурации:", entry.Message, err)
	}
	notifyHardwareEvent(entry)
}

// ClearOldHardwareEventLog удаляет из таблицы hardware_event_log записи
// старше HardwareEventLogRetention (30 суток) - тот же приём порционного
// удаления, что и ClearOldStorageEventLog/cmdb.ClearOldMetric.
func ClearOldHardwareEventLog() {
	cutoff := time.Now().Local().Add(-HardwareEventLogRetention)
	o := orm.NewOrm()
	count := int64(0)
	for {
		res, err := o.Raw("DELETE FROM hardware_event_log WHERE id IN (SELECT id FROM hardware_event_log WHERE time <=? LIMIT 1000)", cutoff).Exec()
		if err != nil {
			fmt.Println("Ошибка удаления старых записей из таблицы hardware_event_log", err)
			break
		}
		num, _ := res.RowsAffected()
		count += num
		if num == 0 {
			break
		}
	}
	fmt.Println("Удалено :", count, "записей старее", HardwareEventLogRetention.String(), "из таблицы hardware_event_log")
}
