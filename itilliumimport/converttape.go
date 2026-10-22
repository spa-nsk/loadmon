package itilliumimport

import (
	"fmt"
	"reflect"
	"strings"

	"loadmon/models"
)

// ResultTape - результат конвертации одной строки выгрузки Итиллиум по
// ленточным библиотекам в models.TapeLibrary (аналог ResultStorage).
type ResultTape struct {
	Tl models.TapeLibrary

	// Extra - поля без прямого соответствия в схеме (extra.*) и сырое
	// "Местоположение" (location_raw) для разбора адреса при импорте.
	Extra map[string]string

	// Warnings - что не удалось разобрать штатно (строка не отбрасывается).
	Warnings []string
}

func (r *ResultTape) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// ConvertRowTape конвертирует одну строку выгрузки по правилам cfg
// (update/tape_library.yaml). Целевые поля: "TapeLibrary.<Поле>" или "extra.<ключ>".
func ConvertRowTape(cfg *Config, row map[string]string) *ResultTape {
	res := &ResultTape{Extra: map[string]string{}}

	for _, f := range cfg.Fields {
		raw, ok := row[f.Header]
		if !ok {
			continue
		}
		out, warn := applyTransform(f.Transform, cleanString(raw), f.Params)
		if warn != "" {
			res.warn("%s: %s", f.Header, warn)
		}
		if out == "" {
			continue
		}
		if err := setTargetTape(res, f.Target, out); err != nil {
			res.warn("%s -> %s: %v", f.Header, f.Target, err)
		}
	}

	for _, c := range cfg.Composites {
		switch c.Name {
		case "location_split":
			compositeLocationTape(res, row, c)
		case "enum_multi":
			compositeEnumMultiTape(cfg, res, row, c)
		default:
			res.warn("composite %q: неизвестное или неподдерживаемое для импорта ленточных библиотек имя преобразования, пропущено", c.Name)
		}
	}
	return res
}

func setTargetTape(res *ResultTape, target, out string) error {
	if strings.HasPrefix(target, "extra.") {
		key := strings.TrimPrefix(target, "extra.")
		if existing := res.Extra[key]; existing != "" {
			res.Extra[key] = existing + "; " + out
		} else {
			res.Extra[key] = out
		}
		return nil
	}
	parts := strings.SplitN(target, ".", 2)
	if len(parts) != 2 || parts[0] != "TapeLibrary" {
		return fmt.Errorf("некорректный target %q (ожидается TapeLibrary.Поле или extra.ключ)", target)
	}
	return setReflectField(reflect.ValueOf(&res.Tl).Elem(), parts[1], out)
}

// compositeLocationTape сохраняет сырое "Местоположение" в extra[prefix+"_raw"];
// площадку, шкаф и юнит из него разбирает импорт (locality.ParseLocation).
func compositeLocationTape(res *ResultTape, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	prefix := c.Params["extra_prefix"]
	if prefix == "" {
		prefix = "location"
	}
	res.Extra[prefix+"_raw"] = val
}

// compositeEnumMultiTape - как compositeEnumMulti (convert.go) для ResultTape.
func compositeEnumMultiTape(cfg *Config, res *ResultTape, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	enumName := c.Params["enum"]
	table := cfg.Enums[enumName]
	if table == nil {
		res.warn("enum_multi: словарь %q не найден (enums)", enumName)
		return
	}
	mapped, ok := table[val]
	if !ok {
		res.warn("enum_multi: значение %q отсутствует в словаре %q", val, enumName)
		res.Extra["status_raw"] = val
		return
	}
	values := strings.Split(mapped, "|")
	for i, target := range c.Targets {
		if i >= len(values) {
			break
		}
		v := strings.TrimSpace(values[i])
		if v == "" {
			continue
		}
		if err := setTargetTape(res, target, v); err != nil {
			res.warn("enum_multi -> %s: %v", target, err)
		}
	}
}

// ConvertFileTape - аналог ConvertFileStorage для ленточных библиотек.
func ConvertFileTape(cfg *Config, path string) ([]*ResultTape, error) {
	rows, err := ReadRows(cfg, path)
	if err != nil {
		return nil, err
	}
	results := make([]*ResultTape, 0, len(rows))
	for _, row := range rows {
		results = append(results, ConvertRowTape(cfg, row))
	}
	return results, nil
}
