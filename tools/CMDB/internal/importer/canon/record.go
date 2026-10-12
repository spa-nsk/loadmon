// Package canon описывает каноническую форму записи об одной КЕ из источника импорта (plan.md
// E4.3), после нормализации (E4.4) и до проверки имени/идентификации. Коннекторы (CSV сейчас,
// API-коннекторы позже, E5) преобразуют исходные данные в Record — единый формат, на котором
// работает остальной конвейер (internal/importer/pipeline), не зная про формат источника.
package canon

// Record — каноническая запись об одной КЕ.
type Record struct {
	Class           string // ref.ci_class.class_code
	SourceName      string // имя как в источнике — сохраняется всегда, даже если КЕ переименовывать нельзя (intent §5.7)
	LocationCode    string
	EnvironmentCode string
	CompanyCode     string
	Criticality     string
	SerialNumber    string
	Description     string
	// Identifiers — значения, по которым КЕ ищется среди уже существующих (E4.6): bios_uuid,
	// serial_number и т.п. Могут дублировать одноимённые поля выше (SerialNumber — одновременно
	// колонка cmdb.ci и идентификатор) — это разные роли одного значения, не ошибка.
	Identifiers map[string]string
	// Attrs — атрибуты таблицы-расширения класса (те же имена, что ci.ClassSpec.Fields), уже
	// нормализованные (единицы, регистр) и готовые к ci.ExtensionInput.Attrs.
	Attrs map[string]any
	// ExternalID — устойчивый идентификатор записи в источнике (id объекта API), с префиксом типа
	// ("vm:<id>"), чтобы не пересекаться между классами. По нему КЕ находится на повторных прогонах
	// через imp.match раньше правил идентификации по атрибутам. Пуст у CSV.
	ExternalID string
	// Relations — исходящие связи этой КЕ. Приёмник указывается ExternalID записи того же прогона
	// и разрешается после применения всех записей (порядок записей в снимке не важен).
	Relations []Relation
	// Vendor — вендорские сведения источника без своей колонки; сливаются в cmdb.ci.attrs под ключом
	// источника ({"zabbix": {...}}), только при изменении. Летучие значения (состояние, метрики)
	// сюда не класть — каждое изменение даёт версию в hist.
	Vendor map[string]any
	// Components — физические компоненты КЕ (CPU, DIMM, диски, БП…). nil — источник компоненты НЕ
	// сообщил (сбой подзапроса): существующие не трогаются; непустой или пустой (не nil) срез —
	// полный список: недостающие у источника компоненты помечаются removed, новые добавляются.
	Components []Component
	// Interfaces — сетевые интерфейсы КЕ; семантика nil/полный список та же, что у Components.
	Interfaces []Interface
	// Software — обнаруженное на КЕ ПО (plan.md E7.7). В отличие от Components/Interfaces —
	// только добавление/обновление, без «недостающее помечается снятым»: неполный охват одного
	// скана не должен молча объявлять ПО удалённым (internal/importer/pipeline.syncSoftware). nil —
	// источник ПО не сообщает (сегодня так у всех коннекторов 1-й волны, E5).
	Software []SoftwareInstall
}

// Component — физический компонент. Key уникален в пределах КЕ и стабилен между прогонами
// (серийный номер либо слот); по нему компоненты сопоставляются между прогонами — diff, а не
// «удалить и вставить» (harness/05, п. 10: иначе ложные изменения в hist).
type Component struct {
	Key           string
	ParentKey     string // Key родителя (DIMM в плате, диск в контроллере); "" — корень
	Type          string // ref.component_type.type_code
	Slot          string
	Manufacturer  string
	PartNumber    string
	SerialNumber  string
	Firmware      string
	CapacityValue *float64
	CapacityUnit  string
	Speed         string
}

// Interface — сетевой интерфейс (cmdb.interface); Name уникально в пределах КЕ.
type Interface struct {
	Name      string
	IfType    string // по умолчанию ethernet
	MAC       string
	SpeedMbps int64
	Enabled   bool
	WWPN      string // FC-порт (HBA): aa:bb:cc:dd:ee:ff:00:11 в нижнем регистре; по нему том сопоставляется с хостом (E5.5)
	Role      string // "bmc" — интерфейс BMC (по его MAC ищется КЕ, E4.6)
}

// Relation — связь Record --Type--> запись с ExternalID == TargetExternalID (ref.rel_type.rel_code).
type Relation struct {
	Type             string
	TargetExternalID string
}

// SoftwareInstall — обнаруженная на КЕ установка ПО (ref.software ищется/создаётся по
// Vendor+Product+Version+Edition, cmdb.ci_software_install — по InstallPath в её пределах).
type SoftwareInstall struct {
	Vendor      string
	Product     string
	Version     string
	Edition     string
	InstallPath string
}

// Item — одна запись снимка источника вместе с исходной строкой и ошибкой разбора: коннектор
// отдаёт ошибочные записи не прерывая снимок (одна плохая запись не должна ронять прогон).
type Item struct {
	Seq    int
	Raw    map[string]string
	Record Record
	Err    error
	// Retain — объект в источнике есть, но прочитать его сейчас не удалось (BMC недоступен): это
	// ошибка записи (Err), а не отсутствие объекта — КЕ не считается пропавшей и счётчик пропусков
	// не растёт (harness/04, урок 3). Нужен Record.ExternalID.
	Retain bool
}
