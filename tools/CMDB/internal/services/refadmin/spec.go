// Package refadmin — универсальный CRUD над справочниками схемы ref.* (plan.md E1.8): список,
// создание, правка, удаление, импорт/экспорт CSV. Около 20 таблиц ref.* — это плоские, не
// историзируемые (без hist.versioning, см. ADR-003) записи с почти одинаковой формой (id/код,
// набор атрибутов, редко — внешние ключи на другие справочники), поэтому вместо Go-структуры,
// контроллера и шаблона на каждую таблицу здесь одна декларативная TableSpec на таблицу
// (specs.go) и один движок CRUD поверх неё (service.go), управляемый через internal/repo (Row —
// map[string]string, тот же примитив, что уже используют CI/component/network).
package refadmin

import (
	"context"

	"github.com/beego/beego/v2/client/orm"
)

// FieldType — тип поля: как рендерить в форме/списке и приводить значение из формы к SQL.
type FieldType string

const (
	TypeText     FieldType = "text"
	TypeTextarea FieldType = "textarea"
	TypeInt      FieldType = "int"
	TypeBigint   FieldType = "bigint"
	TypeNumeric  FieldType = "numeric"
	TypeBool     FieldType = "bool"
	TypeDate     FieldType = "date"
	TypeJSONB    FieldType = "jsonb"
	TypeTextArr  FieldType = "text[]" // CSV-подобное поле в форме (значения через запятую)
	TypeSelect   FieldType = "select" // статический список Field.Options
)

// FK — поле ссылается на другой справочник: в форме — <select>, источник вариантов — SQL-запрос
// (а не обязательно другая TableSpec — например ref.ci_class.class_code или ref.country.country_code).
type FK struct {
	// Query возвращает "value"/"label" колонки, например:
	// `SELECT location_code AS value, location_code || ' — ' || location_name AS label FROM ref.location ORDER BY location_code`.
	Query string
}

// Field — одна колонка таблицы в форме/списке/CSV.
type Field struct {
	Name      string // имя колонки в БД
	Label     string
	Type      FieldType
	PK        bool // первичный ключ (естественный или суррогатный)
	Generated bool // значение выставляет БД (identity, GENERATED ALWAYS AS, uid/created_at и т.п.) — нет в INSERT/UPDATE
	Required  bool
	Nullable  bool     // пустое значение формы -> SQL NULL (иначе -> '' для text/'{}' для jsonb/text[])
	PGType    string   // явный тип для CAST(? AS <PGType>), напр. "char(2)", "numeric(4,1)"; по умолчанию — из Type
	Options   []string // варианты для TypeSelect без FK
	FK        *FK
	ListShow  bool // показывать колонку в таблице списка (не только в форме карточки)
	Help      string
}

// TableSpec — декларативное описание одной справочной таблицы ref.*.
type TableSpec struct {
	Slug   string // сегмент URL: /admin/ref/{Slug}
	Schema string // "ref"
	Table  string
	Title  string
	Fields []Field
	// OrderBy — ORDER BY в списке; по умолчанию — PK.
	OrderBy string
	// SearchCols — text-колонки, по которым ищет строка поиска в списке (ILIKE).
	SearchCols []string

	// Системные колонки, которых нет среди Fields (не редактируются пользователем напрямую),
	// но генерik-CRUD проставляет их сам при наличии в таблице.
	HasCreatedAt bool
	HasUpdatedAt bool
	HasCreatedBy bool
	HasUpdatedBy bool

	// BeforeSave — хук для точечной бизнес-логики поверх обычного CRUD (например, версия профиля
	// именования: новая версия с is_default=true должна снять этот флаг у старых версий того же
	// profile_code, см. specs.go). Выполняется в той же транзакции, до основного INSERT/UPDATE.
	BeforeSave func(ctx context.Context, tx orm.TxOrmer, isCreate bool, values map[string]string) error
}

func (s TableSpec) qualified() string { return s.Schema + "." + s.Table }

// PKField возвращает поле первичного ключа (ровно одно — составных PK в ref.* нет).
func (s TableSpec) PKField() Field { return s.pk() }

// pk возвращает поле первичного ключа (ровно одно — составных PK в ref.* нет).
func (s TableSpec) pk() Field {
	for _, f := range s.Fields {
		if f.PK {
			return f
		}
	}
	panic("refadmin: у таблицы " + s.Table + " не задано поле PK")
}

func pgType(f Field) string {
	if f.PGType != "" {
		return f.PGType
	}
	switch f.Type {
	case TypeInt:
		return "int"
	case TypeBigint:
		return "bigint"
	case TypeNumeric:
		return "numeric"
	case TypeBool:
		return "boolean"
	case TypeDate:
		return "date"
	case TypeJSONB:
		return "jsonb"
	case TypeTextArr:
		return "text[]"
	default:
		return "text"
	}
}
