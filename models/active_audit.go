package models

import "time"

// SetActiveWithAudit централизованно проставляет поле Active (1 - активен,
// 2 - выведен из эксплуатации) вместе с датой/автором соответствующего
// перехода, СТРОГО не перезаписывая дату и автора, если статус фактически
// не меняется - по запросу пользователя (2026-09-22): "нужно заполнять дату
// активации/деактивации и пользователя. Если деактивация была выполнена то
// при попытке повторной деактивации не перезаписывать дату и пользователя,
// аналогично и при активации".
//
// Используется как для models.ServerHardware, так и для
// models.ServerSprRecords - у обеих моделей поля Active/TimeActiveState/
// ActivatorUser/TimeDeActiveState/DeActivatorUser называются и типизированы
// одинаково (int/time.Time/int64), поэтому достаточно одной функции,
// принимающей указатели на эти поля напрямую (без завязки на конкретный
// тип структуры).
//
// newActive должно быть 1 (активен) или 2 (выведен из эксплуатации); любое
// другое значение (в т.ч. 0 - "не указано") присваивается полю Active как
// есть, но дата/автор при этом не трогаются - потому что это не
// "активация" и не "деактивация" в смысле этой функции.
//
// userId - псевдо-ID пользователя/системы, вносящей изменение: реальный
// Id пользователя при ручном изменении через форму, либо один из кодов
// UserSourceManual/UserSourceVMware/UserSourceOneView/UserSourceZabbix при
// автоматическом обновлении - вызывающий код сам решает, какой код
// использовать, в зависимости от того, какая система инициировала запись
// (см. правки cmdb/updatehwhostinfo.go, cmdb/updatepower.go,
// cmdb/initrec.go и т.п., CHANGELOG.md).
func SetActiveWithAudit(active *int, timeActiveState *time.Time, activatorUser *int64, timeDeActiveState *time.Time, deActivatorUser *int64, newActive int, userId int64) {
	if active == nil {
		return
	}
	if *active == newActive {
		// Статус фактически не меняется (в т.ч. повторная попытка
		// активации уже активного или деактивации уже деактивированного) -
		// дату и автора НЕ трогаем, только это и требовалось по запросу.
		return
	}

	now := time.Now().Local()
	switch newActive {
	case 1: // активация
		if timeActiveState != nil {
			*timeActiveState = now
		}
		if activatorUser != nil {
			*activatorUser = userId
		}
	case 2: // деактивация (выведен из эксплуатации)
		if timeDeActiveState != nil {
			*timeDeActiveState = now
		}
		if deActivatorUser != nil {
			*deActivatorUser = userId
		}
	}

	*active = newActive
}
