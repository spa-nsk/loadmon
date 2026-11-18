package itilliumimport

import "strings"

// StatusMapped — кортеж enums.status из update/rack_server.yaml:
// Spr.Stat | Spr.PowerState | Spr.Active | Hw.PowerState | Hw.Active | Hw.ItiliumStatus
type StatusMapped struct {
	Stat, Power, Active, HwPower, HwActive, ItiliumStatus int
}

// Словарь как в update/rack_server.yaml (+ вариант написания getCU без «и»).
var cuStatusMap = map[string]StatusMapped{
	"В работе (эксплуатации)": {1, 1, 1, 1, 1, 1},
	"В работе (эксплуатация)": {1, 1, 1, 1, 1, 1},
	"Выведен":         {1, 2, 2, 2, 2, 2},
	"Списан":          {2, 2, 2, 2, 2, 3},
	"Свободно":        {1, 2, 2, 2, 2, 4},
	"На складе":       {1, 2, 2, 2, 2, 5},
	"Зарезервировано": {1, 2, 2, 2, 2, 6},
	"В ремонте":       {1, 2, 2, 2, 2, 7},
	"В пути":          {1, 2, 2, 2, 2, 8},
	"Черновик":        {1, 2, 2, 2, 2, 9},
}

// MapCUStatusLabel возвращает кортеж статусов импорта по label из getCU.
func MapCUStatusLabel(label string) (StatusMapped, bool) {
	v, ok := cuStatusMap[strings.TrimSpace(label)]
	return v, ok
}
