package models

// Коды статуса Итилиум (единый перечень для ServerHardware, BladeSystems,
// StorageSystems, TapeLibrary, SwitchSprRecords и импорта из Итилиум).
const (
	ItiliumStatusUnspecified    = 0 // не указано
	ItiliumStatusInService      = 1 // в работе
	ItiliumStatusDecommissioned = 2 // выведен
	ItiliumStatusWrittenOff     = 3 // списан
	ItiliumStatusFree           = 4 // свободно
	ItiliumStatusInStock        = 5 // на складе
	ItiliumStatusReserved       = 6 // зарезервирован
	ItiliumStatusInRepair       = 7 // в ремонте
	ItiliumStatusInTransit      = 8 // в пути
	ItiliumStatusDraft          = 9 // черновик
)

// ItiliumStatusItem - пара код/название для select в формах и фильтров.
type ItiliumStatusItem struct {
	Value int
	Name  string
}

// ItiliumStatusList - упорядоченный список статусов Итилиум (для UI).
var ItiliumStatusList = []ItiliumStatusItem{
	{ItiliumStatusUnspecified, "Не указано"},
	{ItiliumStatusInService, "В работе"},
	{ItiliumStatusDecommissioned, "Выведен"},
	{ItiliumStatusWrittenOff, "Списан"},
	{ItiliumStatusFree, "Свободно"},
	{ItiliumStatusInStock, "На складе"},
	{ItiliumStatusReserved, "Зарезервировано"},
	{ItiliumStatusInRepair, "В ремонте"},
	{ItiliumStatusInTransit, "В пути"},
	{ItiliumStatusDraft, "Черновик"},
}

// ItiliumStatusNames - названия статусов по коду (экспорт, бейджи).
var ItiliumStatusNames = map[int]string{
	ItiliumStatusUnspecified:    "Не указано",
	ItiliumStatusInService:      "В работе",
	ItiliumStatusDecommissioned: "Выведен",
	ItiliumStatusWrittenOff:     "Списан",
	ItiliumStatusFree:           "Свободно",
	ItiliumStatusInStock:        "На складе",
	ItiliumStatusReserved:       "Зарезервировано",
	ItiliumStatusInRepair:       "В ремонте",
	ItiliumStatusInTransit:      "В пути",
	ItiliumStatusDraft:          "Черновик",
}

// ItiliumStatusName возвращает название статуса; неизвестный код - "Не указано".
func ItiliumStatusName(st int) string {
	if n, ok := ItiliumStatusNames[st]; ok {
		return n
	}
	return ItiliumStatusNames[ItiliumStatusUnspecified]
}

// ItiliumStatusHiddenFromPlans - списанное и «в пути» не показывают на планах ЦОД/шкафа.
func ItiliumStatusHiddenFromPlans(st int) bool {
	return st == ItiliumStatusWrittenOff || st == ItiliumStatusInTransit
}
