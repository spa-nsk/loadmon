// Package attrschema читает ref.ci_class.attr_schema (plan.md E2.9: "Валидация атрибутов по
// ci_class.attr_schema — JSON Schema для attrs, обязательные поля по стадии ЖЦ") и проверяет,
// заполнены ли атрибуты, обязательные для стадии жизненного цикла, в которую переводят КЕ.
//
// Схема — практическое подмножество JSON Schema, а не полная реализация draft 2020-12: типы и
// перечисления атрибутов класса уже проверяются на вводе в internal/services/ci.ParseAttrs
// (конкретные Go-типы формы полей), поэтому здесь не дублируется — нужная и отсутствующая до сих
// пор часть плана — именно обязательность атрибута начиная с определённой стадии.
package attrschema

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Property — описание одного атрибута (для документации/будущей проверки типов на уровне схемы;
// сейчас содержательно используется только RequiredByStage).
type Property struct {
	Type string   `json:"type,omitempty"`
	Enum []string `json:"enum,omitempty"`
}

// Schema — attr_schema класса КЕ.
type Schema struct {
	Properties map[string]Property `json:"properties,omitempty"`
	// RequiredByStage: стадия ЖЦ -> атрибуты, обязательные начиная с неё (и на всех последующих
	// стадиях согласно порядку жизненного цикла — однажды обязательный атрибут не становится
	// необязательным на более поздней стадии).
	RequiredByStage map[string][]string `json:"requiredByStage,omitempty"`
}

// Parse разбирает attr_schema. Пустая строка и "{}" — пустая схема (без ошибки): ci_class без
// заполненной схемы не накладывает никаких дополнительных требований.
func Parse(raw string) (*Schema, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return &Schema{}, nil
	}
	var s Schema
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		return nil, fmt.Errorf("attrschema: разбор attr_schema: %w", err)
	}
	return &s, nil
}

// Required возвращает атрибуты, обязательные для перехода именно в стадию target. Накапливать
// требования более ранних стадий не нужно: матрица переходов (internal/services/ci.allowedTransitions)
// и так не позволяет попасть, например, в Operational иначе как из Commissioning или Maintenance —
// то есть каждая стадия с непустым RequiredByStage на пути уже была проверена на своём собственном
// переходе. Обратное — накопление по позиции в общем порядке стадий — ошибочно: прямой переход
// Planned → Retired (например, отмена так и не введённого в эксплуатацию заказа) не должен требовать
// атрибутов, нужных только при вводе в эксплуатацию, просто потому что Retired стоит в списке стадий
// позже Commissioning.
func (s *Schema) Required(target string) []string {
	if s == nil {
		return nil
	}
	return s.RequiredByStage[target]
}

// Missing возвращает те из required полей, которых нет в attrs или значение которых пусто.
func Missing(required []string, attrs map[string]string) []string {
	var out []string
	for _, f := range required {
		if strings.TrimSpace(attrs[f]) == "" {
			out = append(out, f)
		}
	}
	return out
}
