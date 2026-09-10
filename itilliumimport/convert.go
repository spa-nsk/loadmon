package itilliumimport

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"loadmon/models"
)

// Result - результат конвертации одной строки выгрузки Итиллиум.
type Result struct {
	Spr models.ServerSprRecords
	Hw  models.ServerHardware

	// Extra - поля, для которых нет прямого соответствия в схеме (см.
	// комментарии в mapping.yaml у target: extra.*), плюс диагностические
	// "сырые" значения составных преобразований (например,
	// "location_zone_code"/"location_room_code" из location_split) - на
	// ручной разбор/донастройку словарей после первого прогона импорта.
	Extra map[string]string

	// Warnings - непустые, если что-то не удалось разобрать штатно
	// (неизвестный статус, кривая дата, поле назначения не найдено и т.п.)
	// - строка не проваливает импорт целиком, а помечается для ручной
	// проверки.
	Warnings []string
}

func (r *Result) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// naValues - значения, которые нужно трактовать как "пусто". "nan"/"NaN"
// характерны для экспорта через pandas/markitdown (см. пример файла),
// "-"/"n/a" встречаются в самих данных Итиллиум как признак "не указано".
var naValues = map[string]bool{
	"": true, "nan": true, "NaN": true, "-": true, "n/a": true, "-n/a-": true, "- n/a -": true,
}

func cleanString(s string) string {
	s = strings.TrimSpace(s)
	if naValues[s] {
		return ""
	}
	return s
}

// ConvertRow конвертирует одну строку выгрузки Итиллиум (map "заголовок
// колонки" -> "значение ячейки как текст") в Result, по правилам из cfg
// (см. LoadMapping/mapping.yaml).
func ConvertRow(cfg *Config, row map[string]string) *Result {
	res := &Result{Extra: map[string]string{}}

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
		if err := setTarget(res, f.Target, out); err != nil {
			res.warn("%s -> %s: %v", f.Header, f.Target, err)
		}
	}

	for _, c := range cfg.Composites {
		applyComposite(cfg, res, row, c)
	}

	return res
}

// applyTransform выполняет простое (одна колонка -> одно значение)
// преобразование. Возвращает итоговое строковое представление (setTarget
// сам приводит его к нужному типу поля через reflect) и, если применимо,
// предупреждение.
func applyTransform(name, val string, params map[string]string) (string, string) {
	switch name {
	case "", "trim":
		return val, ""
	case "numeric_to_string":
		// "111111.0" -> "111111" (pandas/markitdown отдают числовые
		// колонки с плавающей точкой, даже если по смыслу это целое число
		// или вовсе текстовый код).
		if val == "" {
			return "", ""
		}
		if f, err := strconv.ParseFloat(val, 64); err == nil && f == float64(int64(f)) {
			return strconv.FormatInt(int64(f), 10), ""
		}
		return val, ""
	case "date_ru":
		if val == "" {
			return "", ""
		}
		t, err := time.ParseInLocation("02.01.2006", val, time.Local)
		if err != nil {
			return "", fmt.Sprintf("не удалось разобрать дату %q (ожидается ДД.ММ.ГГГГ): %v", val, err)
		}
		return t.Format(time.RFC3339), ""
	default:
		return val, fmt.Sprintf("неизвестная трансформация %q, значение скопировано как есть", name)
	}
}

// setTarget пишет строковое значение out в поле, указанное строкой target
// ("ServerSprRecords.Name" / "ServerHardware.CPUModel" / "extra.<ключ>").
// Для extra.* значение просто кладётся в карту. Для ServerSprRecords.*/
// ServerHardware.* поле находится через reflect по имени и приводится к
// его типу (string/int/int64/float64/time.Time - этого достаточно для
// обеих моделей; остальные типы полей этими справочниками не используются).
func setTarget(res *Result, target, out string) error {
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
		return fmt.Errorf("некорректный target %q (ожидается Структура.Поле или extra.ключ)", target)
	}

	var v reflect.Value
	switch parts[0] {
	case "ServerSprRecords":
		v = reflect.ValueOf(&res.Spr).Elem()
	case "ServerHardware":
		v = reflect.ValueOf(&res.Hw).Elem()
	default:
		return fmt.Errorf("неизвестная целевая структура %q", parts[0])
	}
	return setReflectField(v, parts[1], out)
}

// setReflectField приводит строку out к типу поля fieldName структуры v и
// устанавливает его.
func setReflectField(v reflect.Value, fieldName, out string) error {
	fv := v.FieldByName(fieldName)
	if !fv.IsValid() {
		return fmt.Errorf("поле %q не найдено в %s", fieldName, v.Type().Name())
	}
	if !fv.CanSet() {
		return fmt.Errorf("поле %q недоступно для записи", fieldName)
	}

	switch {
	case fv.Type() == reflect.TypeOf(time.Time{}):
		t, err := time.Parse(time.RFC3339, out)
		if err != nil {
			// setTarget тоже может быть вызван из composite-функций напрямую
			// со "сырой" датой в другом формате - пробуем ещё и ДД.ММ.ГГГГ.
			t, err = time.ParseInLocation("02.01.2006", out, time.Local)
			if err != nil {
				return fmt.Errorf("не удалось привести %q к дате: %w", out, err)
			}
		}
		fv.Set(reflect.ValueOf(t))
		return nil
	case fv.Kind() == reflect.String:
		fv.SetString(out)
		return nil
	case fv.Kind() == reflect.Int || fv.Kind() == reflect.Int64 || fv.Kind() == reflect.Int32:
		n, err := parseFlexibleInt(out)
		if err != nil {
			return fmt.Errorf("не удалось привести %q к целому числу: %w", out, err)
		}
		fv.SetInt(n)
		return nil
	case fv.Kind() == reflect.Float64 || fv.Kind() == reflect.Float32:
		n, err := parseFlexibleFloat(out)
		if err != nil {
			return fmt.Errorf("не удалось привести %q к числу с плавающей точкой: %w", out, err)
		}
		fv.SetFloat(n)
		return nil
	default:
		return fmt.Errorf("неподдерживаемый тип поля %s (%s)", fieldName, fv.Kind())
	}
}

// parseFlexibleFloat разбирает число, допуская и точку, и запятую как
// десятичный разделитель (в Итиллиум встречаются оба варианта, см. "HDD"
// колонку в примере файла: "1,00").
func parseFlexibleFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(s, 64)
}

func parseFlexibleInt(s string) (int64, error) {
	f, err := parseFlexibleFloat(s)
	if err != nil {
		return 0, err
	}
	return int64(f), nil
}

// applyComposite выполняет одно составное преобразование c над строкой row,
// записывая результат(ы) в res. Список источников/целей и параметры
// (разделители, имя словаря enums и т.п.) берутся из конфигурации
// (mapping.yaml) - сама логика разбора у каждой именованной функции здесь,
// в Go (см. комментарий в начале mapping.yaml про то, что можно, а что
// нельзя настроить без правки кода).
func applyComposite(cfg *Config, res *Result, row map[string]string, c CompositeMapping) {
	switch c.Name {
	case "location_split":
		compositeLocationSplit(res, row, c)
	case "rack_unit_split":
		compositeRackUnitSplit(res, row, c)
	case "memory_total":
		compositeMemoryTotal(res, row, c)
	case "enum_multi":
		compositeEnumMulti(cfg, res, row, c)
	case "concat_extra":
		compositeConcatExtra(res, row, c)
	case "blade_ref_split":
		compositeBladeRefSplit(res, row, c)
	default:
		res.warn("composite %q: неизвестное имя преобразования, пропущено", c.Name)
	}
}

// bladeRefRe разбирает ссылку на блейд-корзину вида "(IT-0000003911)
// C7000-15" - тех.номер шасси в скобках (для справки/диагностики) + имя
// шасси (см. rack_blade_server.yaml, колонка "Корзина"). Имя - та же
// мнемоника, что и BladeSystems.Name (см. rack_blade.yaml) - "C7000-15" и
// т.п.
var bladeRefRe = regexp.MustCompile(`^\(([^)]*)\)\s*(.*)$`)

// compositeBladeRefSplit кладёт в extra[prefix+"_name"] имя блейд-корзины и
// в extra[prefix+"_tech_number"] тех.номер в скобках (если был) из колонки
// "Корзина" выгрузки Итиллиум по блейд-серверам. Само сопоставление имени с
// models.BladeSystems.Id (поиск в БД) - НЕ здесь (этот пакет намеренно не
// пишет в БД, см. README.md), а в importbladeservers.go:ImportItiliumBladeServer,
// который читает res.Extra[prefix+"_name"].
func compositeBladeRefSplit(res *Result, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	prefix := c.Params["extra_prefix"]
	if prefix == "" {
		prefix = "blade_ref"
	}
	m := bladeRefRe.FindStringSubmatch(val)
	if m == nil {
		// Формат не совпал (нет скобок) - трактуем всё значение как имя
		// шасси, тех.номер не заполняем.
		res.Extra[prefix+"_name"] = val
		return
	}
	if techNum := strings.TrimSpace(m[1]); techNum != "" {
		res.Extra[prefix+"_tech_number"] = techNum
	}
	if name := strings.TrimSpace(m[2]); name != "" {
		res.Extra[prefix+"_name"] = name
	}
}

func sourceValue(row map[string]string, sources []string, i int) (string, bool) {
	if i >= len(sources) {
		return "", false
	}
	raw, ok := row[sources[i]]
	if !ok {
		return "", false
	}
	return cleanString(raw), true
}

// compositeLocationSplit разбирает "Город1, ЦОД1 | 9 | 9.2 (991)" -
// см. описание в mapping.yaml у composite location_split.
func compositeLocationSplit(res *Result, row map[string]string, c CompositeMapping) {
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
		if err := setTarget(res, c.Targets[0], parts[0]); err != nil {
			res.warn("location_split -> %s: %v", c.Targets[0], err)
		}
	}

	prefix := c.Params["extra_prefix"]
	if prefix == "" {
		prefix = "location"
	}
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
	if len(parts) > 3 {
		tail := strings.Join(parts[3:], sep)
		if strings.TrimSpace(tail) != "" {
			res.Extra[prefix+"_tail"] = tail
		}
	}
}

// unitRangeRe разбирает диапазон юнитов вида "23-23" или "23-25".
var unitRangeRe = regexp.MustCompile(`^\s*(\d+)\s*-\s*(\d+)\s*$`)

// compositeRackUnitSplit разбирает "9.2-A-04 | 23-23" - см. описание в
// mapping.yaml у composite rack_unit_split.
func compositeRackUnitSplit(res *Result, row map[string]string, c CompositeMapping) {
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
		if err := setTarget(res, c.Targets[0], rackNum); err != nil {
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
	size := end - start + 1
	if size < 1 {
		size = 1
	}
	if len(c.Targets) > 1 {
		if err := setTarget(res, c.Targets[1], strconv.Itoa(start)); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[1], err)
		}
	}
	if len(c.Targets) > 2 {
		if err := setTarget(res, c.Targets[2], strconv.Itoa(size)); err != nil {
			res.warn("rack_unit_split -> %s: %v", c.Targets[2], err)
		}
	}
}

// memoryModuleGbRe вытаскивает объём ОДНОГО модуля памяти в Гб из
// произвольного текстового описания вида "RAM 2x32Gb_2666" (тут не
// применяется - см. ниже, объём/количество в примере разнесены по разным
// колонкам) или "DDR3 4 GB 800/1066/...". Первое число перед Gb/GB/ГБ.
var memoryModuleGbRe = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*(?:gb|гб)\b`)

// parseMemoryModuleGb извлекает объём одного модуля памяти в Гб из
// текстового описания (колонки "Память (1)"/"Память (2)").
func parseMemoryModuleGb(spec string) (float64, bool) {
	m := memoryModuleGbRe.FindStringSubmatch(spec)
	if m == nil {
		return 0, false
	}
	v, err := parseFlexibleFloat(m[1])
	if err != nil {
		return 0, false
	}
	return v, true
}

// compositeMemoryTotal суммирует "Память (1)"+"Память (1), кол." и
// "Память (2)"+"Память (2), кол." в суммарный объём ОЗУ в Мб
// (ServerHardware.MemoryMb) - см. описание в mapping.yaml у composite
// memory_total. Порядок source: [spec1, count1, spec2, count2].
func compositeMemoryTotal(res *Result, row map[string]string, c CompositeMapping) {
	if len(c.Targets) == 0 {
		return
	}
	var totalGb float64
	found := false

	pairs := [][2]int{{0, 1}, {2, 3}}
	for _, p := range pairs {
		spec, okSpec := sourceValue(row, c.Sources, p[0])
		countStr, okCount := sourceValue(row, c.Sources, p[1])
		if !okSpec || spec == "" {
			continue
		}
		gb, ok := parseMemoryModuleGb(spec)
		if !ok {
			res.warn("memory_total: не удалось извлечь объём модуля из %q", spec)
			continue
		}
		count := 1.0
		if okCount && countStr != "" {
			if c, err := parseFlexibleFloat(countStr); err == nil && c > 0 {
				count = c
			}
		}
		totalGb += gb * count
		found = true
	}
	if !found {
		return
	}
	memoryMb := int64(totalGb * 1024)
	if err := setTarget(res, c.Targets[0], strconv.FormatInt(memoryMb, 10)); err != nil {
		res.warn("memory_total -> %s: %v", c.Targets[0], err)
	}
}

// compositeEnumMulti сопоставляет значение колонки-источника (по точному
// совпадению текста) со словарём cfg.Enums[params["enum"]] и раскладывает
// "|"-разделённое значение словаря по targets позиционно - см. описание в
// mapping.yaml у composite enum_multi (сейчас применяется к "Статус").
func compositeEnumMulti(cfg *Config, res *Result, row map[string]string, c CompositeMapping) {
	val, ok := sourceValue(row, c.Sources, 0)
	if !ok || val == "" {
		return
	}
	enumName := c.Params["enum"]
	table := cfg.Enums[enumName]
	if table == nil {
		res.warn("enum_multi: словарь %q не найден в mapping.yaml (enums)", enumName)
		return
	}
	mapped, ok := table[val]
	if !ok {
		res.warn("enum_multi: значение %q отсутствует в словаре %q - добавьте его в mapping.yaml (enums.%s)", val, enumName, enumName)
		if len(c.Sources) > 0 {
			res.Extra["status_raw"] = val
		}
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
		if err := setTarget(res, target, v); err != nil {
			res.warn("enum_multi -> %s: %v", target, err)
		}
	}
}

// compositeConcatExtra склеивает несколько текстовых колонок через
// params["separator"] (по умолчанию "; "), пропуская пустые - см. описание
// в mapping.yaml у composite concat_extra (сейчас применяется для
// ServerSprRecords.Details).
func compositeConcatExtra(res *Result, row map[string]string, c CompositeMapping) {
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
	if err := setTarget(res, c.Targets[0], strings.Join(parts, sep)); err != nil {
		res.warn("concat_extra -> %s: %v", c.Targets[0], err)
	}
}
