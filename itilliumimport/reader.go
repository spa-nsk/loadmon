package itilliumimport

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ReadRows открывает xlsx-файл выгрузки Итиллиум (path) и построчно
// читает лист cfg.SourceSheet (по умолчанию "export", см. LoadMapping),
// возвращая срез строк как map[заголовок колонки]значение_ячейки_текстом.
// Первая строка листа считается строкой заголовков.
//
// excelize уже используется в проекте (см. controllers/sprbladesystems.go
// и другие экспорт-в-XLSX функции) - здесь взят тот же пакет, чтобы не
// тянуть лишнюю зависимость.
func ReadRows(cfg *Config, path string) ([]map[string]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("itilliumimport: открытие %s: %w", path, err)
	}
	defer f.Close()

	sheet := cfg.SourceSheet
	if sheet == "" {
		sheet = "export"
	}

	all, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("itilliumimport: чтение листа %q: %w", sheet, err)
	}
	if len(all) == 0 {
		return nil, nil
	}

	header := all[0]
	rows := make([]map[string]string, 0, len(all)-1)
	for _, line := range all[1:] {
		row := make(map[string]string, len(header))
		for i, colName := range header {
			if i >= len(line) {
				row[colName] = ""
				continue
			}
			row[colName] = line[i]
		}
		// Пропускаем полностью пустые строки (частый "хвост" в выгрузках
		// Итиллиум - пустые строки форматирования после последней записи).
		empty := true
		for _, v := range row {
			if cleanString(v) != "" {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// ConvertFile - удобный "весь файл целиком" вход: читает path через
// ReadRows и прогоняет каждую строку через ConvertRow.
func ConvertFile(cfg *Config, path string) ([]*Result, error) {
	rows, err := ReadRows(cfg, path)
	if err != nil {
		return nil, err
	}
	results := make([]*Result, 0, len(rows))
	for _, row := range rows {
		results = append(results, ConvertRow(cfg, row))
	}
	return results, nil
}

// ConvertFileBlade - аналог ConvertFile для импорта блейд-корзин
// (models.BladeSystems, см. convertblade.go/importblades.go): читает path
// через ReadRows и прогоняет каждую строку через ConvertRowBlade.
func ConvertFileBlade(cfg *Config, path string) ([]*ResultBlade, error) {
	rows, err := ReadRows(cfg, path)
	if err != nil {
		return nil, err
	}
	results := make([]*ResultBlade, 0, len(rows))
	for _, row := range rows {
		results = append(results, ConvertRowBlade(cfg, row))
	}
	return results, nil
}
