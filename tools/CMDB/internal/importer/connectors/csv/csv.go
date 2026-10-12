// Package csv — коннектор ручной загрузки CSV/XLSX (plan.md E4.9): единственный источник в этом
// MVP, но реализует тот же интерфейс "сырые строки -> canon.Record", которым позже будут пользоваться
// API-коннекторы (E5) — пайплайн (internal/importer/pipeline) от формата файла не зависит.
package csv

import (
	"bytes"
	csvstd "encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"

	"cmdb/internal/importer/canon"
	"cmdb/internal/importer/normalize"
	"cmdb/internal/services/ci"
)

// coreColumns — колонки cmdb.ci, не относящиеся к таблице-расширению класса; в этом порядке идут
// первыми в шаблоне (Columns/Template) и распознаются Parse отдельно от spec.Fields.
var coreColumns = []string{"name", "location_code", "serial_number", "environment_code", "company_code", "criticality", "description"}

// Columns — колонки шаблона для класса: базовые + атрибуты таблицы-расширения (её же поля видит
// форма ручного ввода, ci.ClassSpec.Fields — E3.5).
func Columns(spec ci.ClassSpec) []string {
	cols := append([]string{}, coreColumns...)
	for _, f := range spec.Fields {
		cols = append(cols, f.Name)
	}
	return cols
}

// Template строит пустой CSV-шаблон (заголовок) для класса — E4.9 "генерация шаблона".
func Template(spec ci.ClassSpec) []byte {
	var buf bytes.Buffer
	w := csvstd.NewWriter(&buf)
	_ = w.Write(Columns(spec))
	w.Flush()
	return buf.Bytes()
}

// Row — одна строка источника: исходные значения по колонкам (для imp.raw_record) и результат
// нормализации (для imp.norm_record) — либо Record, либо Err, если строка не разобралась.
type Row = canon.Item

// Parse читает файл (format: "csv" или "xlsx") и возвращает по одной Row на строку данных (без
// заголовка). Ошибка в отдельной строке не прерывает разбор остальных — она попадает в Row.Err,
// пайплайн покажет её как status='error' у конкретной записи (E4.11: одна плохая строка не должна
// ронять весь импорт).
func Parse(r io.Reader, format string, spec ci.ClassSpec) ([]Row, error) {
	var header []string
	var records [][]string
	switch format {
	case "csv":
		cr := csvstd.NewReader(r)
		cr.FieldsPerRecord = -1 // строки короче заголовка — обычное дело для необязательных колонок
		all, err := cr.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("csv: чтение файла: %w", err)
		}
		if len(all) == 0 {
			return nil, fmt.Errorf("csv: пустой файл")
		}
		header, records = all[0], all[1:]
	case "xlsx":
		f, err := excelize.OpenReader(r)
		if err != nil {
			return nil, fmt.Errorf("xlsx: чтение файла: %w", err)
		}
		defer f.Close()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("xlsx: нет листов")
		}
		all, err := f.GetRows(sheets[0])
		if err != nil {
			return nil, fmt.Errorf("xlsx: чтение листа %q: %w", sheets[0], err)
		}
		if len(all) == 0 {
			return nil, fmt.Errorf("xlsx: лист %q пуст", sheets[0])
		}
		header, records = all[0], all[1:]
	default:
		return nil, fmt.Errorf("csv: неизвестный формат %q (ожидается csv или xlsx)", format)
	}

	for i, h := range header {
		header[i] = strings.TrimSpace(h)
	}

	rows := make([]Row, 0, len(records))
	for i, rec := range records {
		raw := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(rec) {
				raw[h] = rec[i]
			}
		}
		row := Row{Seq: i + 1, Raw: raw}
		row.Record, row.Err = normalizeRow(raw, spec)
		rows = append(rows, row)
	}
	return rows, nil
}

// normalizeRow — E4.4: единицы (ram_gb), регистр/формат UUID (bios_uuid), обрезка пробелов;
// остальная типизация (int/bool/uuid-формат/select) — уже существующая ci.ClassSpec.ParseAttrs
// (та же, что валидирует форму ручного ввода, E3.5) — не дублируем её здесь.
func normalizeRow(raw map[string]string, spec ci.ClassSpec) (canon.Record, error) {
	rec := canon.Record{
		Class:           spec.Class,
		SourceName:      normalize.Trim(raw["name"]),
		LocationCode:    normalize.Trim(raw["location_code"]),
		SerialNumber:    normalize.Trim(raw["serial_number"]),
		EnvironmentCode: normalize.Trim(raw["environment_code"]),
		CompanyCode:     normalize.Trim(raw["company_code"]),
		Criticality:     normalize.Trim(raw["criticality"]),
		Description:     normalize.Trim(raw["description"]),
		Identifiers:     map[string]string{},
	}
	if rec.SourceName == "" {
		return rec, fmt.Errorf("не заполнена колонка name")
	}
	if rec.SerialNumber != "" {
		rec.Identifiers["serial_number"] = rec.SerialNumber
	}

	fieldValues := make(map[string]string, len(spec.Fields))
	for _, f := range spec.Fields {
		v := raw[f.Name]
		switch f.Name {
		case "ram_gb":
			if n, err := normalize.GB(v); err == nil {
				v = n
			} // ошибку не глотаем молча — ParseAttrs ниже даст понятное сообщение на исходном значении
		case "bios_uuid":
			if n, err := normalize.UUID(v); err == nil {
				v = n
			}
		}
		fieldValues[f.Name] = v
	}
	attrs, fieldErrs := spec.ParseAttrs(fieldValues)
	if len(fieldErrs) > 0 {
		msgs := make([]string, 0, len(fieldErrs))
		for _, m := range fieldErrs {
			msgs = append(msgs, m)
		}
		return rec, fmt.Errorf("%s", strings.Join(msgs, "; "))
	}
	rec.Attrs = attrs
	if v, _ := attrs["bios_uuid"].(string); v != "" {
		rec.Identifiers["bios_uuid"] = v
	}
	return rec, nil
}
