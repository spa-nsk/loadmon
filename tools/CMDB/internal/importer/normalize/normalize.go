// Package normalize приводит значения атрибутов из источников импорта к единому виду (plan.md
// E4.4: "единицы, производители, регистр и формат MAC/WWN/UUID"). Общий пакет, а не часть
// конкретного коннектора — те же правила понадобятся любому источнику (CSV сейчас, API-коннекторы
// позже, E5), а не только CSV.
package normalize

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// UUID проверяет формат и приводит к нижнему регистру (bios_uuid и т.п. хранятся как есть, но
// сравниваются регистронезависимо — источники отдают то в верхнем, то в нижнем регистре).
func UUID(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", nil
	}
	if !uuidRe.MatchString(v) {
		return "", fmt.Errorf("normalize: %q не похоже на UUID", raw)
	}
	return strings.ToLower(v), nil
}

// unitMultiplier — множитель для приведения к ГБ (считаем гибибайты гигабайтами, как везде в проекте).
var unitMultiplier = map[string]float64{
	"": 1, "b": 1.0 / (1 << 30), "kb": 1.0 / (1 << 20), "mb": 1.0 / 1024, "gb": 1, "tb": 1024,
}

var numUnitRe = regexp.MustCompile(`^([0-9]+(?:[.,][0-9]+)?)\s*([a-zA-Zа-яА-Я]*)$`)

// GB разбирает значение объёма памяти/ёмкости с необязательной единицей ("16", "16 GB",
// "16384MB", "16Гб") и возвращает целое число гигабайт как строку (готово для CAST(? AS int) в
// таблицах-расширениях, см. ci.ExtensionInput). Пустая строка -> пустая строка (поле не заполнено
// источником, не 0 — в базе останется NULL).
func GB(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", nil
	}
	m := numUnitRe.FindStringSubmatch(v)
	if m == nil {
		return "", fmt.Errorf("normalize: %q не похоже на объём памяти", raw)
	}
	num, err := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
	if err != nil {
		return "", fmt.Errorf("normalize: %q: %w", raw, err)
	}
	unit := strings.ToLower(m[2])
	switch unit {
	case "", "gb", "гб", "г":
		unit = "gb"
	case "mb", "мб", "м":
		unit = "mb"
	case "tb", "тб", "т":
		unit = "tb"
	case "b", "б":
		unit = "b"
	default:
		return "", fmt.Errorf("normalize: %q: неизвестная единица %q", raw, m[2])
	}
	gb := num * unitMultiplier[unit]
	return strconv.Itoa(int(gb + 0.5)), nil
}

// Trim — базовая нормализация текстовых полей без явного формата: обрезка пробелов по краям.
func Trim(raw string) string {
	return strings.TrimSpace(raw)
}
