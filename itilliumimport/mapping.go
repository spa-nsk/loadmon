// Package itilliumimport реализует конвертацию строк выгрузки rack-серверов
// из Итиллиум (см. пример "Серверы_templ.xlsx", лист "export") в модели
// loadmon models.ServerSprRecords / models.ServerHardware.
//
// Связка "колонка Итиллиум -> поле модели" вынесена в отдельный файл
// mapping.yaml (в этом же каталоге), а не зашита в Go-код - см. подробные
// комментарии в самом mapping.yaml. Смысл: если Итиллиум поменяет состав
// колонок в новой выгрузке (переименует столбец, добавит новый), либо
// потребуется другое сопоставление/словарь статусов - правится только YAML,
// без пересборки логики парсинга.
//
// Пакет НЕ читает .xlsx самостоятельно - на вход ConvertRow подаётся уже
// разобранная строка в виде map[string]string (ключ - точный заголовок
// колонки, как в mapping.yaml/fields[].header). Разбор самого .xlsx (через
// excelize, который уже используется в проекте, см. controllers/*.go) -
// отдельный, более простой шаг, вызывающая сторона делает это сама (открыть
// файл, взять cfg.SourceSheet, построить map по заголовкам первой строки).
package itilliumimport

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FieldMapping - одно простое правило "одна колонка -> одно поле".
type FieldMapping struct {
	Header    string            `yaml:"header"`
	Target    string            `yaml:"target"`
	Transform string            `yaml:"transform"`
	Params    map[string]string `yaml:"params,omitempty"`
}

// CompositeMapping - правило "несколько колонок -> одно или несколько
// полей" (Sources и Targets читаются/пишутся позиционно, где это применимо
// для конкретной функции - см. applyComposite в convert.go).
type CompositeMapping struct {
	Name    string            `yaml:"name"`
	Sources []string          `yaml:"sources"`
	Targets []string          `yaml:"targets"`
	Params  map[string]string `yaml:"params,omitempty"`
}

// Config - корневая структура mapping.yaml.
type Config struct {
	SourceSheet string                       `yaml:"source_sheet"`
	Fields      []FieldMapping               `yaml:"fields"`
	Composites  []CompositeMapping           `yaml:"composites"`
	Enums       map[string]map[string]string `yaml:"enums"`
}

// LoadMapping читает и разбирает mapping.yaml по указанному пути.
func LoadMapping(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("itilliumimport: чтение %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("itilliumimport: разбор %s: %w", path, err)
	}
	if cfg.SourceSheet == "" {
		cfg.SourceSheet = "export"
	}
	return &cfg, nil
}
