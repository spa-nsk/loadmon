package itilliumimport

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"loadmon/models"
)

// ResultStorage - результат конвертации одной строки выгрузки Итиллиум по
// СХД в models.StorageSystems. Полный аналог ResultBlade (convertblade.go)
// для блейд-корзин - отдельный тип, а не расширение Result/ResultBlade, по
// той же причине (см. комментарий у ResultBlade): setTarget/applyComposite
// там жёстко завязаны на другие целевые структуры, а здесь нужна ровно одна
// (models.StorageSystems). Общие вспомогательные функции (cleanString,
// applyTransform, parseFlexibleInt/Float, sourceValue, setReflectField,
// unitRangeRe) переиспользуются из convert.go как есть.
//
// 2026-09-08, по запросу пользователя "добавь в меню Импорт пункт «Импорт
// СХД из Итилиум» и реализуй импорт по аналогии с блэйд корзинами" - см.
// rack_storage.yaml (маппинг колонок) и importstorages.go (сохранение в БД).
type ResultStorage struct {
	St models.StorageSystems

	// Extra - поля без прямого соответствия в схеме StorageSystems + сырые
	// значения составных преобразований (location_zone_code/room_code,
	// статус текстом, "сырое" размещение в шкафу для Anotation) - см.
	// Result.Extra в convert.go, назначение то же самое.
	Extra map[string]string

	// Warnings - см. Result.Warnings в convert.go.
	Warnings []string
}

func (r *ResultStorage) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// ConvertRowStorage - аналог ConvertRow/ConvertRowBlade для СХД: одна строка
// выгрузки Итиллиум (map "заголовок колонки" -> значение ячейки) по правилам
// cfg (см. LoadMapping/server_web_new/conf/yaml/rack_storage.yaml)
// превращается в ResultStorage. Целевые поля в yaml должны быть указаны как
// "StorageSystems.<Поле>" или "extra.<ключ>" (см. setTargetStorage).
func ConvertRowStorage(cfg *Config, row map[string]string) *ResultStorage {
	res := &ResultStorage{Extra: map[string]string{}}

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
		if err := setTargetStorage(res, f.Target, out); err != nil {
			res.warn("%s -> %s: %v", f.Header, f.Target, err)
		}
	}

	for _, c := range cfg.Composites {
		applyCompositeStorage(res, row, c)
	}

	return res
}

// setTargetStorage - аналог setTargetBlade (convertblade.go), но
// единственная поддерживаемая целевая структура - "StorageSystems" (плюс
// "extra.*").
func setTargetStorage(res *ResultStorage, target, out string) error {
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
		return fmt.Errorf("некорректный target %q (ожидается StorageSystems.Поле или extra.ключ)", target)
	}
	if parts[0] != "StorageSystems" {
		return fmt.Errorf("неизвестная целевая структура %q (для импорта СХД допустима только StorageSystems)", parts[0])
	}

	v := reflect.ValueOf(&res.St).Elem()
	return setReflectField(v, parts[1], out)
}

// applyCompositeStorage - аналог applyCompositeBlade (convertblade.go),
// набор составных преобразований, нужных для StorageSystems (местоположение,
// размещение в стойке, "сырое" размещение целиком для Anotation).
func applyCompositeStorage(res *ResultStorage, row map[string]string, c CompositeMapping) {
	switch c.Name {
	case "location_split":
		compositeLocationSplitStorage(res, row, c)
	case "rack_unit_split":
		compositeRackUnitSplitStorage(res, row, c)
	case "concat_extra":
		compositeConcatExtraStorage(res, row, c)
	default:
		res.warn("composite %q: неизвестное или неподдерживаемое для импорта СХД имя преобразования, пропущено", c.Name)
	}
}

// compositeLocationSplitStorage - аналог compositeLocationSplitBlade
// (convertblade.go) для ResultStorage/setTargetStorage. Логика разбора
// идентична - см. описание composite location_split в mapping.yaml
// (серверный образец).
func compositeLocationSplitStorage(res *ResultStorage, row map[string]string, c CompositeMapping) {
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
		if err := setTargetStorage(res, c.Targets[0], parts[0]); err != nil {
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

	// ИЗМЕНЕНО (2026-09-10, по запросу пользователя про IT-0000042272 -
	// "Местоположение" вида "Новосибирск, ЦОД РТК, Менделеева 1 |
	// Colocation | Зал 205 | Шкаф 2.4.4 | Unit 15", т.е. шкаф/юнит указаны
	// ПРЯМО в "Местоположение" вместо отдельной колонки "Размещение в
	// шкафу/стойке"): части после индекса 2 (если есть) сохраняются целиком
	// в extra[prefix+"_tail"] - разбор конкретно на "Шкаф"/"Unit" делает
	// itilliumimport/storageplacement.go:parseLocationEmbeddedPlacement,
	// используется importstorages.go как запасной источник размещения,
	// когда отдельная колонка "Размещение в шкафу/стойке" пуста.
	//
	// ⚠️ ИСТОРИЯ ОШИБКИ: этот же код изначально был добавлен только в
	// compositeLocationSplit (convert.go) - универсальную версию для
	// серверов/блейд-корзин. Импорт СХД идёт через ОТДЕЛЬНУЮ реализацию
	// (applyCompositeStorage -> compositeLocationSplitStorage, этот файл,
	// работает с ResultStorage/setTargetStorage, а не с общим Result) -
	// поэтому для СХД правка не действовала, пока не была продублирована
	// сюда.
	if len(parts) > 3 {
		tail := strings.Join(parts[3:], sep)
		if strings.TrimSpace(tail) != "" {
			res.Extra[prefix+"_tail"] = tail
		}
	}
}

// compositeRackUnitSplitStorage - аналог compositeRackUnitSplitBlade
// (convertblade.go) для ResultStorage/setTargetStorage. Третий/четвёртый
// target - StartUnitPos/EndUnitPos (числовые поля StorageSystems) - SizeUnit
// вычисляется отдельно в importstorages.go по этим двум значениям (тот же
// приём, что для BladeSystems в importblades.go).
func compositeRackUnitSplitStorage(res *ResultStorage, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	sep := c.Params["separator"]
	if sep == "" {
		sep = "|"
	}
	// В ячейке может быть несколько шкафов через запятую ("шкаф1 | 9-22,
	// шкаф2 | 24-40"): все сегменты разбирает parseStoragePlacementSegments
	// (storageplacement.go), здесь - только первый, чтобы остаток после "|"
	// не принимался за диапазон юнитов и не сыпал предупреждениями.
	val = strings.SplitN(val, ",", 2)[0]
	parts := strings.SplitN(val, sep, 2)
	rackNum := strings.TrimSpace(parts[0])

	if len(c.Targets) > 0 && rackNum != "" {
		if err := setTargetStorage(res, c.Targets[0], rackNum); err != nil {
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
	if len(c.Targets) > 1 {
		if err := setTargetStorage(res, c.Targets[1], strconv.Itoa(start)); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[1], err)
		}
	}
	if len(c.Targets) > 2 {
		if err := setTargetStorage(res, c.Targets[2], strconv.Itoa(end)); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[2], err)
		}
	}
}

// compositeConcatExtraStorage - аналог compositeConcatExtraBlade
// (convertblade.go) для ResultStorage/setTargetStorage - используется в
// rack_storage.yaml, чтобы сохранить СЫРОЕ (нетронутое) значение колонки
// "Размещение в шкафу/стойке" целиком в extra.rack_placement_raw
// (importstorages.go подставляет его в StorageSystems.Anotation после
// текстового статуса, см. комментарий в rack_storage.yaml).
func compositeConcatExtraStorage(res *ResultStorage, row map[string]string, c CompositeMapping) {
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
	if err := setTargetStorage(res, c.Targets[0], strings.Join(parts, sep)); err != nil {
		res.warn("concat_extra -> %s: %v", c.Targets[0], err)
	}
}
