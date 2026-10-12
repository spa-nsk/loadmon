package refadmin

import (
	"context"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// FK-запросы переиспользуются между несколькими TableSpec — вынесены в константы, чтобы не
// разъезжаться при правке (те же колонки/сортировка, что и в internal/services/refdata).
const (
	fkCountry     = `SELECT country_code AS value, country_code || ' — ' || name_ru AS label FROM ref.country ORDER BY country_code`
	fkRegion      = `SELECT region_code AS value, region_code || ' — ' || name AS label FROM ref.region ORDER BY region_code`
	fkUnlocode    = `SELECT unlocode AS value, unlocode || ' — ' || name AS label FROM ref.unlocode ORDER BY unlocode`
	fkLocationByC = `SELECT location_code AS value, location_code || ' — ' || location_name AS label FROM ref.location ORDER BY location_code`
	fkLocationByI = `SELECT id::text AS value, location_code || ' — ' || location_name AS label FROM ref.location ORDER BY location_code`
	fkSitePlace   = `SELECT id::text AS value, kind || ': ' || code AS label FROM ref.site_place ORDER BY id`
	fkManufact    = `SELECT id::text AS value, name AS label FROM ref.manufacturer ORDER BY name`
	fkCIClass     = `SELECT class_code AS value, name AS label FROM ref.ci_class ORDER BY name`
	fkProfileCode = `SELECT profile_code AS value, profile_code || ' — ' || name AS label FROM ref.naming_profile_code ORDER BY profile_code`
	fkNamingRule  = `SELECT id::text AS value, profile_code || '/' || rule_code AS label FROM ref.naming_rule ORDER BY profile_code, rule_code`
	fkDesignation = `SELECT id::text AS value, profile_code || '/' || designation_code AS label FROM ref.designation ORDER BY profile_code, designation_code`
)

var (
	directoryKinds  = []string{"AD", "ALD_PRO", "FREEIPA", "SAMBA", "OTHER"}
	companyCats     = []string{"OWN", "JV", "TP"}
	siteKinds       = []string{"dc", "office", "cloud", "colocation"}
	sitePlaceKinds  = []string{"building", "floor", "room", "row", "rack", "u"}
	statusLifecycle = []string{"active", "deprecated", "retired"} // ref.designation.status
	casePolicies    = []string{"as_is", "lower", "upper"}
	seqRangePurpose = []string{"active", "legacy_reserved"}
	softwarePolicy  = []string{"allowed", "forbidden", "review"}
)

// All — справочники ref.* с CRUD-администрированием (plan.md E1.8), в порядке пункта меню.
var All = []TableSpec{
	specDirectory, specCountry, specUnlocode, specRegion, specCompany, specLocation, specSitePlace,
	specManufacturer, specModel, specSoftware, specCIClass, specComponentType,
	specNamingProfileCode, specNamingProfile, specNamingRule, specDesignation, specSeqRange,
	specRelType, specLifecycle, specEnvironment, specEmployeeType, specBaselineConfig, specSourceSystem,
}

// bySlug строится один раз из All (см. registry.go: Get).
func bySlug() map[string]TableSpec {
	m := make(map[string]TableSpec, len(All))
	for _, s := range All {
		m[s.Slug] = s
	}
	return m
}

var specDirectory = TableSpec{
	Slug: "directory", Schema: "ref", Table: "directory", Title: "Каталоги (LDAP/AD)",
	OrderBy: "directory_code", SearchCols: []string{"directory_code", "fqdn"},
	HasCreatedAt: true, HasUpdatedAt: true, HasCreatedBy: true, HasUpdatedBy: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "directory_code", Label: "Код", Type: TypeText, Required: true, ListShow: true},
		{Name: "kind", Label: "Тип", Type: TypeSelect, Options: directoryKinds, Required: true, ListShow: true},
		{Name: "fqdn", Label: "FQDN", Type: TypeText, Nullable: true, ListShow: true},
		{Name: "description", Label: "Описание", Type: TypeTextarea, Nullable: true},
	},
}

var specCountry = TableSpec{
	Slug: "country", Schema: "ref", Table: "country", Title: "Страны (ISO 3166-1)",
	OrderBy: "country_code", SearchCols: []string{"country_code", "name_ru", "name_en"},
	Fields: []Field{
		{Name: "country_code", Label: "Код (ISO)", Type: TypeText, PGType: "char(2)", PK: true, Required: true, ListShow: true},
		{Name: "name_ru", Label: "Название (RU)", Type: TypeText, Required: true, ListShow: true},
		{Name: "name_en", Label: "Название (EN)", Type: TypeText, Required: true, ListShow: true},
	},
}

var specUnlocode = TableSpec{
	Slug: "unlocode", Schema: "ref", Table: "unlocode", Title: "UN/LOCODE (обогащение площадок)",
	OrderBy: "unlocode", SearchCols: []string{"unlocode", "name"},
	Fields: []Field{
		{Name: "unlocode", Label: "UN/LOCODE", Type: TypeText, PGType: "char(5)", PK: true, Required: true, ListShow: true},
		{Name: "country_code", Label: "Страна", Type: TypeSelect, PGType: "char(2)", FK: &FK{Query: fkCountry}, Required: true, ListShow: true},
		{Name: "locality_code", Label: "Код населённого пункта", Type: TypeText, PGType: "char(3)", Required: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
	},
}

var specRegion = TableSpec{
	Slug: "region", Schema: "ref", Table: "region", Title: "Регионы",
	OrderBy: "region_code", SearchCols: []string{"region_code", "name"},
	Fields: []Field{
		{Name: "region_code", Label: "Код", Type: TypeText, PK: true, Required: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
	},
}

var specCompany = TableSpec{
	Slug: "company", Schema: "ref", Table: "company", Title: "Компании",
	OrderBy: "company_code", SearchCols: []string{"company_code", "company_name"},
	HasCreatedAt: true, HasUpdatedAt: true, HasCreatedBy: true, HasUpdatedBy: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "company_code", Label: "Код", Type: TypeText, Required: true, ListShow: true, Help: "2-8 латинских заглавных букв"},
		{Name: "company_name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
		{Name: "category", Label: "Категория", Type: TypeSelect, Options: companyCats, Required: true, ListShow: true},
		{Name: "company_type", Label: "Тип компании", Type: TypeText, Nullable: true},
		{Name: "cluster", Label: "Кластер", Type: TypeText, Nullable: true},
		{Name: "business_category", Label: "Бизнес-категория", Type: TypeText, Nullable: true},
		{Name: "creates_accounts", Label: "Заводит учётные записи", Type: TypeBool},
		{Name: "contacts_only", Label: "Только контакты", Type: TypeBool},
	},
}

var specLocation = TableSpec{
	Slug: "location", Schema: "ref", Table: "location", Title: "Площадки",
	OrderBy: "location_code", SearchCols: []string{"location_code", "location_name"},
	HasCreatedAt: true, HasUpdatedAt: true, HasCreatedBy: true, HasUpdatedBy: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "location_code", Label: "Код площадки", Type: TypeText, Required: true, ListShow: true, Help: "AAA или AAA-BBB, только A-Z/2-9"},
		{Name: "location_code_c", Label: "Код (сжатый)", Generated: true, ListShow: true},
		{Name: "location_name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
		{Name: "location_name_lat", Label: "Название (лат., ≤32)", Type: TypeText, Required: true},
		{Name: "country_code", Label: "Страна", Type: TypeSelect, PGType: "char(2)", FK: &FK{Query: fkCountry}, Required: true, ListShow: true},
		{Name: "region_code", Label: "Регион", Type: TypeSelect, FK: &FK{Query: fkRegion}, Nullable: true},
		{Name: "unlocode", Label: "UN/LOCODE", Type: TypeSelect, PGType: "char(5)", FK: &FK{Query: fkUnlocode}, Nullable: true},
		{Name: "category", Label: "Категория", Type: TypeText, Nullable: true},
		{Name: "site_kind", Label: "Вид площадки", Type: TypeSelect, Options: siteKinds, Required: true, ListShow: true},
		{Name: "address", Label: "Адрес", Type: TypeTextarea, Nullable: true},
	},
}

var specSitePlace = TableSpec{
	Slug: "site_place", Schema: "ref", Table: "site_place", Title: "Размещение (здание/этаж/стойка)",
	OrderBy: "location_id, parent_id, code", SearchCols: []string{"code", "name"},
	HasCreatedAt: true, HasUpdatedAt: true, HasCreatedBy: true, HasUpdatedBy: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "location_id", Label: "Площадка", Type: TypeSelect, PGType: "bigint", FK: &FK{Query: fkLocationByI}, Required: true, ListShow: true},
		{Name: "parent_id", Label: "Родитель", Type: TypeSelect, PGType: "bigint", FK: &FK{Query: fkSitePlace}, Nullable: true, ListShow: true},
		{Name: "kind", Label: "Уровень", Type: TypeSelect, Options: sitePlaceKinds, Required: true, ListShow: true},
		{Name: "code", Label: "Код", Type: TypeText, Required: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Nullable: true},
		{Name: "rack_units", Label: "Юнитов (для стойки)", Type: TypeInt, Nullable: true},
	},
}

var specManufacturer = TableSpec{
	Slug: "manufacturer", Schema: "ref", Table: "manufacturer", Title: "Производители",
	OrderBy: "name", SearchCols: []string{"name"},
	HasCreatedAt: true, HasUpdatedAt: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
		{Name: "aliases", Label: "Алиасы (через запятую)", Type: TypeTextArr, Nullable: true, ListShow: true},
	},
}

var specModel = TableSpec{
	Slug: "model", Schema: "ref", Table: "model", Title: "Модели оборудования",
	OrderBy: "manufacturer_id, model_name", SearchCols: []string{"model_name", "part_number"},
	HasCreatedAt: true, HasUpdatedAt: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "manufacturer_id", Label: "Производитель", Type: TypeSelect, PGType: "bigint", FK: &FK{Query: fkManufact}, Required: true, ListShow: true},
		{Name: "model_name", Label: "Модель", Type: TypeText, Required: true, ListShow: true},
		{Name: "part_number", Label: "Партномер", Type: TypeText, Nullable: true},
		{Name: "ci_class", Label: "Класс КЕ", Type: TypeSelect, FK: &FK{Query: fkCIClass}, Nullable: true, ListShow: true},
		{Name: "u_height", Label: "Высота, U", Type: TypeNumeric, PGType: "numeric(4,1)", Nullable: true},
		{Name: "eol_date", Label: "Конец продаж (EOL)", Type: TypeDate, Nullable: true},
		{Name: "eos_date", Label: "Конец поддержки (EOS)", Type: TypeDate, Nullable: true},
		{Name: "is_domestic", Label: "Отечественное", Type: TypeBool, ListShow: true},
		{Name: "registry_ref", Label: "Ссылка на реестр", Type: TypeText, Nullable: true},
	},
}

var specSoftware = TableSpec{
	Slug: "software", Schema: "ref", Table: "software", Title: "Программное обеспечение",
	OrderBy: "vendor, product", SearchCols: []string{"vendor", "product"},
	HasCreatedAt: true, HasUpdatedAt: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "vendor", Label: "Производитель", Type: TypeText, Required: true, ListShow: true},
		{Name: "product", Label: "Продукт", Type: TypeText, Required: true, ListShow: true},
		{Name: "version", Label: "Версия", Type: TypeText, Nullable: true, ListShow: true},
		{Name: "edition", Label: "Редакция", Type: TypeText, Nullable: true},
		{Name: "is_domestic", Label: "Отечественное", Type: TypeBool, ListShow: true},
		{Name: "registry_ref", Label: "Ссылка на реестр", Type: TypeText, Nullable: true},
		{Name: "policy", Label: "Политика", Type: TypeSelect, Options: softwarePolicy, Required: true, ListShow: true},
		{Name: "eol_date", Label: "Конец поддержки", Type: TypeDate, Nullable: true},
	},
}

var specCIClass = TableSpec{
	Slug: "ci_class", Schema: "ref", Table: "ci_class", Title: "Классы КЕ",
	OrderBy: "name", SearchCols: []string{"class_code", "name"},
	HasCreatedAt: true, HasUpdatedAt: true,
	Fields: []Field{
		{Name: "class_code", Label: "Код класса", Type: TypeText, PK: true, Required: true, ListShow: true},
		{Name: "parent_class", Label: "Родительский класс", Type: TypeSelect, FK: &FK{Query: fkCIClass}, Nullable: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
		{Name: "ext_table", Label: "Таблица-расширение", Type: TypeText, Nullable: true},
		{Name: "attr_schema", Label: "Схема атрибутов (JSON)", Type: TypeJSONB, Nullable: true},
		{Name: "owner_role", Label: "Роль-владелец", Type: TypeText, Nullable: true},
	},
}

var specComponentType = TableSpec{
	Slug: "component_type", Schema: "ref", Table: "component_type", Title: "Типы компонентов",
	OrderBy: "name", SearchCols: []string{"type_code", "name"},
	Fields: []Field{
		{Name: "type_code", Label: "Код", Type: TypeText, PK: true, Required: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
	},
}

var specNamingProfileCode = TableSpec{
	Slug: "naming_profile_code", Schema: "ref", Table: "naming_profile_code", Title: "Профили именования (линейки)",
	OrderBy: "profile_code", SearchCols: []string{"profile_code", "name"},
	Fields: []Field{
		{Name: "profile_code", Label: "Код профиля", Type: TypeText, PK: true, Required: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
	},
}

// specNamingProfile — версии профиля именования (E1.8: "версионирование профиля — новая версия
// не меняет статус ранее выданных имён"). ref.designation ссылается только на profile_code (не на
// конкретную version), поэтому создание новой версии само по себе не затрагивает ref.designation —
// единственная реальная бизнес-логика здесь в том, что is_default может быть true только у одной
// версии профиля одновременно (см. BeforeSave ниже).
var specNamingProfile = TableSpec{
	Slug: "naming_profile", Schema: "ref", Table: "naming_profile", Title: "Версии профиля именования",
	OrderBy: "profile_code, effective_from", SearchCols: []string{"version"},
	HasCreatedAt: true,
	BeforeSave: func(ctx context.Context, tx orm.TxOrmer, isCreate bool, values map[string]string) error {
		if values["is_default"] != "on" && values["is_default"] != "true" && values["is_default"] != "1" {
			return nil
		}
		profileCode := strings.TrimSpace(values["profile_code"])
		if profileCode == "" {
			return nil
		}
		_, err := tx.RawWithCtx(ctx, `UPDATE ref.naming_profile SET is_default = false WHERE profile_code = ?`, profileCode).Exec()
		return err
	},
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "profile_code", Label: "Профиль", Type: TypeSelect, FK: &FK{Query: fkProfileCode}, Required: true, ListShow: true},
		{Name: "version", Label: "Версия", Type: TypeText, Required: true, ListShow: true},
		{Name: "effective_from", Label: "Действует с", Type: TypeDate, Required: true, ListShow: true},
		{Name: "is_default", Label: "Текущая версия по умолчанию", Type: TypeBool, ListShow: true,
			Help: "Только одна версия профиля может быть текущей — при установке снимается у остальных версий этого профиля"},
	},
}

var specNamingRule = TableSpec{
	Slug: "naming_rule", Schema: "ref", Table: "naming_rule", Title: "Правила именования (шаблоны)",
	OrderBy: "profile_code, rule_code", SearchCols: []string{"rule_code", "template"},
	HasCreatedAt: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "rule_code", Label: "Код правила", Type: TypeText, Required: true, ListShow: true},
		{Name: "profile_code", Label: "Профиль", Type: TypeSelect, FK: &FK{Query: fkProfileCode}, Required: true, ListShow: true},
		{Name: "template", Label: "Шаблон", Type: TypeText, Required: true, ListShow: true, Help: "например {site}-{cls}-{seq}"},
		{Name: "regex", Label: "Regex (грубая проверка на уровне БД)", Type: TypeText, Required: true},
		{Name: "max_len", Label: "Макс. длина", Type: TypeInt, Nullable: true},
		{Name: "case_policy", Label: "Регистр", Type: TypeSelect, Options: casePolicies, Required: true},
		{Name: "applies_to_class", Label: "Применяется к классу", Type: TypeSelect, FK: &FK{Query: fkCIClass}, Nullable: true},
		{Name: "priority", Label: "Приоритет", Type: TypeInt, Required: true},
	},
}

var specDesignation = TableSpec{
	Slug: "designation", Schema: "ref", Table: "designation", Title: "Коды классов (designation)",
	OrderBy: "profile_code, designation_code", SearchCols: []string{"designation_code", "description"},
	HasCreatedAt: true, HasUpdatedAt: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "designation_code", Label: "Код", Type: TypeText, Required: true, ListShow: true},
		{Name: "profile_code", Label: "Профиль", Type: TypeSelect, FK: &FK{Query: fkProfileCode}, Required: true, ListShow: true},
		{Name: "object_class", Label: "Класс КЕ", Type: TypeSelect, FK: &FK{Query: fkCIClass}, Nullable: true, ListShow: true},
		{Name: "description", Label: "Описание", Type: TypeText, Required: true, ListShow: true},
		{Name: "seq_length", Label: "Длина номера", Type: TypeInt, Required: true, ListShow: true},
		{Name: "naming_rule_id", Label: "Правило именования", Type: TypeSelect, PGType: "bigint", FK: &FK{Query: fkNamingRule}, Nullable: true},
		{Name: "status", Label: "Статус", Type: TypeSelect, Options: statusLifecycle, Required: true, ListShow: true},
		{Name: "replaced_by", Label: "Заменён на", Type: TypeSelect, PGType: "bigint", FK: &FK{Query: fkDesignation}, Nullable: true, ListShow: true},
	},
}

var specSeqRange = TableSpec{
	Slug: "seq_range", Schema: "ref", Table: "seq_range", Title: "Диапазоны номеров",
	OrderBy: "designation_id, range_from", SearchCols: []string{"prefix"},
	HasCreatedAt: true,
	BeforeSave: func(ctx context.Context, tx orm.TxOrmer, isCreate bool, values map[string]string) error {
		if isCreate && strings.TrimSpace(values["next_value"]) == "" {
			values["next_value"] = values["range_from"]
		}
		return nil
	},
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "designation_id", Label: "Код класса", Type: TypeSelect, PGType: "bigint", FK: &FK{Query: fkDesignation}, Required: true, ListShow: true},
		{Name: "range_from", Label: "От", Type: TypeBigint, Required: true, ListShow: true},
		{Name: "range_to", Label: "До", Type: TypeBigint, Required: true, ListShow: true},
		{Name: "next_value", Label: "Текущий курсор (next_value)", Type: TypeBigint, Required: true, ListShow: true,
			Help: "Следующий номер, который выдаст аллокатор; при создании по умолчанию равен «От», вручную менять — только для исправления рассинхронизации"},
		{Name: "prefix", Label: "Пометка", Type: TypeText, Nullable: true},
		{Name: "purpose", Label: "Назначение", Type: TypeSelect, Options: seqRangePurpose, Required: true, ListShow: true},
		{Name: "location_code", Label: "Площадка (пусто — общий)", Type: TypeSelect, FK: &FK{Query: fkLocationByC}, Nullable: true, ListShow: true},
	},
}

var specRelType = TableSpec{
	Slug: "rel_type", Schema: "ref", Table: "rel_type", Title: "Типы связей",
	OrderBy: "rel_code", SearchCols: []string{"rel_code", "name_fwd"},
	Fields: []Field{
		{Name: "rel_code", Label: "Код", Type: TypeText, PK: true, Required: true, ListShow: true},
		{Name: "name_fwd", Label: "Название (прямое)", Type: TypeText, Required: true, ListShow: true},
		{Name: "name_rev", Label: "Название (обратное)", Type: TypeText, Required: true, ListShow: true},
	},
}

var specLifecycle = TableSpec{
	Slug: "lifecycle", Schema: "ref", Table: "lifecycle", Title: "Стадии/статусы жизненного цикла",
	OrderBy: "stage, status", SearchCols: []string{"stage", "status"},
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "stage", Label: "Стадия", Type: TypeText, Required: true, ListShow: true},
		{Name: "status", Label: "Статус", Type: TypeText, Required: true, ListShow: true},
	},
}

var specEnvironment = TableSpec{
	Slug: "environment", Schema: "ref", Table: "environment", Title: "Окружения",
	OrderBy: "environment_code", SearchCols: []string{"environment_code", "name"},
	Fields: []Field{
		{Name: "environment_code", Label: "Код", Type: TypeText, PK: true, Required: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
	},
}

var specEmployeeType = TableSpec{
	Slug: "employee_type", Schema: "ref", Table: "employee_type", Title: "Типы сотрудников",
	OrderBy: "employee_type_code", SearchCols: []string{"employee_type_code", "name"},
	Fields: []Field{
		{Name: "employee_type_code", Label: "Код", Type: TypeText, PGType: "char(1)", PK: true, Required: true, ListShow: true},
		{Name: "name", Label: "Название", Type: TypeText, Required: true, ListShow: true},
	},
}

var specBaselineConfig = TableSpec{
	Slug: "baseline_config", Schema: "ref", Table: "baseline_config", Title: "Эталонные конфигурации",
	OrderBy: "ci_class, role", SearchCols: []string{"role"},
	HasCreatedAt: true, HasUpdatedAt: true,
	Fields: []Field{
		{Name: "id", Label: "ID", PK: true, Generated: true, ListShow: true},
		{Name: "ci_class", Label: "Класс КЕ", Type: TypeSelect, FK: &FK{Query: fkCIClass}, Required: true, ListShow: true},
		{Name: "role", Label: "Роль", Type: TypeText, Nullable: true, ListShow: true},
		{Name: "required_attrs", Label: "Требуемые атрибуты (JSON)", Type: TypeJSONB, Nullable: true},
	},
}

var specSourceSystem = TableSpec{
	Slug: "source_system", Schema: "ref", Table: "source_system", Title: "Системы-источники импорта",
	OrderBy: "source_system_code", SearchCols: []string{"source_system_code", "kind"},
	Fields: []Field{
		{Name: "source_system_code", Label: "Код", Type: TypeText, PK: true, Required: true, ListShow: true},
		{Name: "kind", Label: "Вид", Type: TypeText, Required: true, ListShow: true},
		{Name: "trust_level", Label: "Уровень доверия", Type: TypeInt, Required: true, ListShow: true},
		{Name: "endpoint", Label: "Адрес API", Type: TypeText, Nullable: true},
		{Name: "secret_ref", Label: "Ссылка на секрет", Type: TypeText, Nullable: true, Help: "ссылка на хранилище секретов, не сам секрет (E0.3/A9)"},
	},
}
