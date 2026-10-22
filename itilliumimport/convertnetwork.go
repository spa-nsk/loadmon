package itilliumimport

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"loadmon/models"
)

// ResultNet - результат конвертации одной строки выгрузки Итиллиум по сетевому
// оборудованию в models.SwitchSprRecords (аналог ResultTape/ResultStorage).
type ResultNet struct {
	Eq models.SwitchSprRecords

	// Extra - extra.* из yaml: type_name (тип оборудования), location_raw
	// (сырое "Местоположение"), rack_num (шкаф из "Размещение в шкафу/стойке").
	Extra map[string]string

	Warnings []string
}

func (r *ResultNet) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// ConvertRowNet конвертирует одну строку по правилам cfg
// (update/network_equipment.yaml). Цели: "SwitchSprRecords.<Поле>" или "extra.<ключ>".
func ConvertRowNet(cfg *Config, row map[string]string) *ResultNet {
	res := &ResultNet{Extra: map[string]string{}}

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
		if err := setTargetNet(res, f.Target, out); err != nil {
			res.warn("%s -> %s: %v", f.Header, f.Target, err)
		}
	}

	for _, c := range cfg.Composites {
		switch c.Name {
		case "location_split":
			if val, ok := sourceValue(row, c.Sources, 0); ok && val != "" {
				prefix := c.Params["extra_prefix"]
				if prefix == "" {
					prefix = "location"
				}
				res.Extra[prefix+"_raw"] = val
			}
		case "rack_unit_split":
			compositeRackUnitNet(res, row, c)
		case "enum_multi":
			compositeEnumMultiNet(cfg, res, row, c)
		default:
			res.warn("composite %q: неизвестное или неподдерживаемое для импорта сетевого оборудования имя преобразования, пропущено", c.Name)
		}
	}
	return res
}

func setTargetNet(res *ResultNet, target, out string) error {
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
	if len(parts) != 2 || parts[0] != "SwitchSprRecords" {
		return fmt.Errorf("некорректный target %q (ожидается SwitchSprRecords.Поле или extra.ключ)", target)
	}
	return setReflectField(reflect.ValueOf(&res.Eq).Elem(), parts[1], out)
}

// compositeRackUnitNet разбирает "Размещение в шкафу/стойке" вида
// "[альт. наименование /] шкаф | старт-конец". Несколько шкафов через запятую
// ("A | 8-8, B | 8-8") - берётся первый. Шкаф -> targets[0], старт -> targets[1],
// размер в юнитах (конец-старт+1) -> targets[2].
func compositeRackUnitNet(res *ResultNet, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	sep := c.Params["separator"]
	if sep == "" {
		sep = "|"
	}
	val = strings.SplitN(val, ",", 2)[0]
	parts := strings.SplitN(val, sep, 2)
	if rack := strings.TrimSpace(parts[0]); rack != "" && len(c.Targets) > 0 {
		if err := setTargetNet(res, c.Targets[0], rack); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[0], err)
		}
	}
	if len(parts) < 2 {
		return
	}
	m := unitRangeRe.FindStringSubmatch(strings.TrimSpace(parts[1]))
	if m == nil {
		res.warn("rack_unit_split: не удалось разобрать диапазон юнитов %q (ожидается \"начало-конец\")", parts[1])
		return
	}
	start, _ := strconv.Atoi(m[1])
	end, _ := strconv.Atoi(m[2])
	size := end - start + 1
	if size < 1 {
		size = 1
	}
	if len(c.Targets) > 1 {
		_ = setTargetNet(res, c.Targets[1], strconv.Itoa(start))
	}
	if len(c.Targets) > 2 {
		_ = setTargetNet(res, c.Targets[2], strconv.Itoa(size))
	}
}

func compositeEnumMultiNet(cfg *Config, res *ResultNet, row map[string]string, c CompositeMapping) {
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
		if v := strings.TrimSpace(values[i]); v != "" {
			if err := setTargetNet(res, target, v); err != nil {
				res.warn("enum_multi -> %s: %v", target, err)
			}
		}
	}
}

// ConvertFileNet читает xlsx и прогоняет строки через ConvertRowNet.
func ConvertFileNet(cfg *Config, path string) ([]*ResultNet, error) {
	rows, err := ReadRows(cfg, path)
	if err != nil {
		return nil, err
	}
	results := make([]*ResultNet, 0, len(rows))
	for _, row := range rows {
		results = append(results, ConvertRowNet(cfg, row))
	}
	return results, nil
}
