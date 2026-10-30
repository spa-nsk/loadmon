package itilliumimport

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"loadmon/models"
)

// ResultBlade - результат конвертации одной строки выгрузки Итиллиум по
// блейд-корзинам (шасси) в models.BladeSystems. Аналог Result (см.
// convert.go) для серверов - отдельный тип, а не расширение Result, чтобы
// не размывать существующую (уже проверенную) логику импорта серверов:
// setTarget/applyComposite там жёстко завязаны на ServerSprRecords/
// ServerHardware, а здесь нужна ровно одна целевая структура
// (models.BladeSystems). Общие вспомогательные функции (cleanString,
// applyTransform, parseFlexibleInt/Float, sourceValue, setReflectField,
// unitRangeRe) переиспользуются из convert.go как есть - они не завязаны на
// конкретный тип структуры.
type ResultBlade struct {
	Bs models.BladeSystems

	// Extra - поля без прямого соответствия в схеме BladeSystems + сырые
	// значения составных преобразований (location_zone_code/room_code) -
	// см. Result.Extra в convert.go, назначение то же самое.
	Extra map[string]string

	// Warnings - см. Result.Warnings в convert.go.
	Warnings []string
}

func (r *ResultBlade) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// ConvertRowBlade - аналог ConvertRow (convert.go) для блейд-корзин: одна
// строка выгрузки Итиллиум (map "заголовок колонки" -> значение ячейки) по
// правилам cfg (см. LoadMapping/conf/yaml/rack_blade.yaml) превращается в
// ResultBlade. Целевые поля в mapping.yaml должны быть указаны как
// "BladeSystems.<Поле>" или "extra.<ключ>" (см. setTargetBlade).
func ConvertRowBlade(cfg *Config, row map[string]string) *ResultBlade {
	res := &ResultBlade{Extra: map[string]string{}}

	for _, f := range cfg.Fields {
		raw, ok := row[f.Header]
		if !ok {
			continue // колонка отсутствует в этой выгрузке - не считается ошибкой
		}
		val := cleanString(raw)
		out, warn := applyTransform(f.Transform, val, f.Params)
		if warn != "" {
			res.warn("%s: %s", f.Header, warn)
		}
		if out == "" {
			continue
		}
		if err := setTargetBlade(res, f.Target, out); err != nil {
			res.warn("%s -> %s: %v", f.Header, f.Target, err)
		}
	}

	for _, c := range cfg.Composites {
		applyCompositeBlade(cfg, res, row, c)
	}

	return res
}

// setTargetBlade - аналог setTarget (convert.go), но единственная
// поддерживаемая целевая структура - "BladeSystems" (плюс "extra.*").
func setTargetBlade(res *ResultBlade, target, out string) error {
	if strings.HasPrefix(target, "extra.") {
		key := strings.TrimPrefix(target, "extra.")
		if existing, ok := res.Extra[key]; ok && existing != "" {
			res.Extra[key] = existing + "; " + out
		} else {
			res.Extra[key] = out
		}
		return nil
	}

	parts := strings.SplitN(target, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("некорректный target %q (ожидается BladeSystems.Поле или extra.ключ)", target)
	}
	if parts[0] != "BladeSystems" {
		return fmt.Errorf("неизвестная целевая структура %q (для импорта блейд-корзин допустима только BladeSystems)", parts[0])
	}

	v := reflect.ValueOf(&res.Bs).Elem()
	return setReflectField(v, parts[1], out)
}

// applyCompositeBlade - аналог applyComposite (convert.go): местоположение,
// размещение в стойке и enum_multi для ItiliumStatus (см. blade_systems.yaml).
func applyCompositeBlade(cfg *Config, res *ResultBlade, row map[string]string, c CompositeMapping) {
	switch c.Name {
	case "location_split":
		compositeLocationSplitBlade(res, row, c)
	case "rack_unit_split":
		compositeRackUnitSplitBlade(res, row, c)
	case "concat_extra":
		compositeConcatExtraBlade(res, row, c)
	case "enum_multi":
		compositeEnumMultiBlade(cfg, res, row, c)
	default:
		res.warn("composite %q: неизвестное или неподдерживаемое для импорта блейд-корзин имя преобразования, пропущено", c.Name)
	}
}

// compositeEnumMultiBlade - как compositeEnumMultiTape: "Статус" -> BladeSystems.ItiliumStatus.
func compositeEnumMultiBlade(cfg *Config, res *ResultBlade, row map[string]string, c CompositeMapping) {
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
			if err := setTargetBlade(res, target, v); err != nil {
				res.warn("enum_multi -> %s: %v", target, err)
			}
		}
	}
}

// compositeLocationSplitBlade - аналог compositeLocationSplit (convert.go)
// для ResultBlade/setTargetBlade. Логика разбора идентична - см. подробное
// описание composite location_split в mapping.yaml (серверный образец).
func compositeLocationSplitBlade(res *ResultBlade, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	sep := c.Params["separator"]
	if sep == "" {
		sep = "|"
	}
	parts := strings.Split(val, sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(c.Targets) > 0 && len(parts) > 0 && parts[0] != "" {
		if err := setTargetBlade(res, c.Targets[0], parts[0]); err != nil {
			res.warn("location_split -> %s: %v", c.Targets[0], err)
		}
	}

	prefix := c.Params["extra_prefix"]
	if prefix == "" {
		prefix = "location"
	}
	res.Extra[prefix+"_raw"] = val // сырое "Местоположение" - разбирается locality.ParseLocation при резолве площадки
	if len(parts) > 1 && parts[1] != "" {
		res.Extra[prefix+"_zone_code"] = parts[1]
	}
	if len(parts) > 2 && parts[2] != "" {
		res.Extra[prefix+"_room_code"] = parts[2]
	}
}

// compositeRackUnitSplitBlade - аналог compositeRackUnitSplit (convert.go)
// для ResultBlade/setTargetBlade.
func compositeRackUnitSplitBlade(res *ResultBlade, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	sep := c.Params["separator"]
	if sep == "" {
		sep = "|"
	}
	parts := strings.SplitN(val, sep, 2)
	rackNum := strings.TrimSpace(parts[0])

	if len(c.Targets) > 0 && rackNum != "" {
		if err := setTargetBlade(res, c.Targets[0], rackNum); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[0], err)
		}
	}
	if len(parts) < 2 {
		return
	}
	unitRange := strings.TrimSpace(parts[1])
	m := unitRangeRe.FindStringSubmatch(unitRange)
	if m == nil {
		res.warn("rack_unit_split: не удалось разобрать диапазон юнитов %q (ожидается \"начало-конец\")", unitRange)
		return
	}
	start, _ := strconv.Atoi(m[1])
	end, _ := strconv.Atoi(m[2])
	// ИЗМЕНЕНО (2026-09-08, по запросу пользователя): раньше третьим target'ом
	// сюда писался вычисленный SizeUnit (end-start+1). Теперь пишется сам
	// EndUnitPos (конечная позиция) - SizeUnit больше не показывается ни в
	// форме редактирования, ни при импорте, и вычисляется по StartUnitPos/
	// EndUnitPos перед сохранением (см. itilliumimport/importblades.go и
	// controllers/sprbladesystems.go:bladeSystemsComputeSizeUnit).
	if len(c.Targets) > 1 {
		if err := setTargetBlade(res, c.Targets[1], strconv.Itoa(start)); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[1], err)
		}
	}
	if len(c.Targets) > 2 {
		if err := setTargetBlade(res, c.Targets[2], strconv.Itoa(end)); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[2], err)
		}
	}
}

// compositeConcatExtraBlade - аналог compositeConcatExtra (convert.go) для
// ResultBlade/setTargetBlade (например, для склейки нескольких текстовых
// колонок в BladeSystems.Description).
func compositeConcatExtraBlade(res *ResultBlade, row map[string]string, c CompositeMapping) {
	sep := c.Params["separator"]
	if sep == "" {
		sep = "; "
	}
	var parts []string
	for i := range c.Sources {
		v, ok := sourceValue(row, c.Sources, i)
		if ok && v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) == 0 || len(c.Targets) == 0 {
		return
	}
	if err := setTargetBlade(res, c.Targets[0], strings.Join(parts, sep)); err != nil {
		res.warn("concat_extra -> %s: %v", c.Targets[0], err)
	}
}
