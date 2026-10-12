package ci

import (
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// FieldType — тип поля формы; определяет виджет и предварительную проверку значения.
type FieldType string

const (
	FieldText     FieldType = "text"
	FieldTextarea FieldType = "textarea"
	FieldInt      FieldType = "int"
	FieldNumber   FieldType = "number"
	FieldBool     FieldType = "bool"
	FieldSelect   FieldType = "select"
	FieldInet     FieldType = "inet"
	FieldUUID     FieldType = "uuid"
)

// Field — описание одного атрибута класса КЕ для универсальной формы.
type Field struct {
	Name    string // колонка таблицы-расширения
	Label   string
	Type    FieldType
	Options []string // для FieldSelect
	Help    string
}

// ClassSpec — форма класса КЕ: какая таблица-расширение и какие поля показывать. Для классов
// без ClassSpec доступны только общие поля cmdb.ci.
type ClassSpec struct {
	Class    string
	Title    string
	ExtTable string
	Fields   []Field
}

// ClassSpecs — классы, для которых в UI есть форма создания и правки атрибутов. Новый класс
// добавляется записью сюда (таблица-расширение и seed класса уже есть в миграциях).
var ClassSpecs = []ClassSpec{
	{
		Class: "server", Title: "Физический сервер", ExtTable: "ci_server",
		Fields: []Field{
			{Name: "form_factor", Label: "Форм-фактор", Type: FieldSelect, Options: []string{"rack", "blade", "tower"}},
			{Name: "bay", Label: "Отсек (blade)", Type: FieldText},
			{Name: "bios_uuid", Label: "BIOS UUID", Type: FieldUUID},
			{Name: "bmc_ip", Label: "IP BMC", Type: FieldInet},
			{Name: "bmc_name", Label: "Имя BMC", Type: FieldText, Help: "по правилу <host>-bmc"},
			{Name: "firmware", Label: "Прошивка", Type: FieldText},
			{Name: "cpu_count", Label: "Процессоров", Type: FieldInt},
			{Name: "core_count", Label: "Ядер", Type: FieldInt},
			{Name: "ram_gb", Label: "ОЗУ, ГБ", Type: FieldInt},
			{Name: "os_family", Label: "Семейство ОС", Type: FieldSelect, Options: []string{"Linux", "Windows", "Unix", "Other"}},
			{Name: "os_distro", Label: "Дистрибутив ОС", Type: FieldText, Help: "Astra, РЕД ОС, ALT, RHEL, Windows Server …"},
			{Name: "os_version", Label: "Версия ОС", Type: FieldText},
			{Name: "hypervisor_role", Label: "Роль гипервизора", Type: FieldText},
		},
	},
	{
		Class: "vm", Title: "Виртуальная машина", ExtTable: "ci_vm",
		Fields: []Field{
			{Name: "instance_uuid", Label: "Instance UUID", Type: FieldUUID},
			{Name: "bios_uuid", Label: "BIOS UUID", Type: FieldUUID},
			{Name: "moref", Label: "MoRef / ID в платформе", Type: FieldText},
			{Name: "vcpu", Label: "vCPU", Type: FieldInt},
			{Name: "ram_gb", Label: "ОЗУ, ГБ", Type: FieldInt},
			{Name: "guest_os", Label: "Гостевая ОС", Type: FieldText},
			{Name: "tools_status", Label: "Статус гостевых утилит", Type: FieldText},
			{Name: "template", Label: "Шаблон ВМ", Type: FieldBool},
		},
	},
}

// SpecFor возвращает форму класса или nil.
func SpecFor(class string) *ClassSpec {
	for i := range ClassSpecs {
		if ClassSpecs[i].Class == class {
			return &ClassSpecs[i]
		}
	}
	return nil
}

// ParseAttrs превращает значения формы (имя поля → строка) в Attrs для UpsertExtension.
// Пустая строка = NULL; числа проверяются заранее, чтобы пользователь получал понятное сообщение
// по конкретному полю, а не ошибку БД. Возвращает карту значений и карту ошибок по полям.
func (s *ClassSpec) ParseAttrs(values map[string]string) (map[string]any, map[string]string) {
	attrs := make(map[string]any, len(s.Fields))
	errs := map[string]string{}
	for _, f := range s.Fields {
		raw := strings.TrimSpace(values[f.Name])
		if f.Type == FieldBool {
			attrs[f.Name] = raw == "1" || raw == "true" || raw == "on"
			continue
		}
		if raw == "" {
			attrs[f.Name] = nil
			continue
		}
		switch f.Type {
		case FieldInt:
			if _, err := strconv.ParseInt(raw, 10, 32); err != nil {
				errs[f.Name] = fmt.Sprintf("%s: ожидается целое число", f.Label)
				continue
			}
		case FieldNumber:
			if _, err := strconv.ParseFloat(raw, 64); err != nil {
				errs[f.Name] = fmt.Sprintf("%s: ожидается число", f.Label)
				continue
			}
		case FieldInet:
			if !validInet(raw) {
				errs[f.Name] = fmt.Sprintf("%s: ожидается IP-адрес", f.Label)
				continue
			}
		case FieldUUID:
			if !uuidRe.MatchString(raw) {
				errs[f.Name] = fmt.Sprintf("%s: ожидается UUID", f.Label)
				continue
			}
		case FieldSelect:
			if !contains(f.Options, raw) {
				errs[f.Name] = fmt.Sprintf("%s: недопустимое значение %q", f.Label, raw)
				continue
			}
		}
		attrs[f.Name] = raw
	}
	return attrs, errs
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// AllowedTransitions возвращает стадии, в которые можно перейти из текущей (по матрице ЖЦ),
// в порядке жизненного цикла.
func AllowedTransitions(from string) []string {
	var out []string
	for _, s := range lifecycleOrder {
		if s != from && CanTransition(from, s) {
			out = append(out, s)
		}
	}
	return out
}

var lifecycleOrder = []string{StagePlanned, StageOrdered, StageInStock, StageCommissioning, StageOperational,
	StageMaintenance, StageDecommissioning, StageRetired, StageDisposed}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func validInet(s string) bool {
	if strings.Contains(s, "/") {
		_, err := netip.ParsePrefix(s)
		return err == nil
	}
	_, err := netip.ParseAddr(s)
	return err == nil
}

// LifecycleStages возвращает все стадии в порядке жизненного цикла.
func LifecycleStages() []string {
	return append([]string(nil), lifecycleOrder...)
}
