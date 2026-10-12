// Package auth — аутентификация и права (plan.md E7.1/E7.2): локальные учётные записи (bcrypt),
// LDAP (AD, ALD Pro, FreeIPA), сессии с тайм-аутами, блокировка после серии неудачных попыток и
// ролевая модель. Права проверяются фильтром (маршрут → право, internal/filters.RBAC) и сервисами
// там, где нужна область (класс КЕ у роли class_owner).
package auth

import "sort"

// Права. Строки — часть контракта (audit.event, OpenAPI E7.3), не переименовывать.
const (
	CIRead        = "ci.read"        // просмотр КЕ, компонентов, связей, истории, «на дату»
	CIWrite       = "ci.write"       // ввод и правка КЕ и связанных данных (для class_owner — только свои классы)
	RefWrite      = "ref.write"      // справочники ref.*
	NamingAdmin   = "naming.admin"   // правила и профили именования
	ImportRead    = "import.read"    // прогоны, очередь конфликтов
	ImportRun     = "import.run"     // загрузка файлов, запуск/отмена заданий коннекторов
	AuditRead     = "audit.read"     // журнал изменений и журнал действий
	BaselineRead  = "baseline.read"  // срезы, сравнение, выгрузка
	BaselineWrite = "baseline.write" // создание, заморозка, утверждение срезов
	PeriodManage  = "period.manage"  // закрытие/переоткрытие учётных периодов, правки закрытого периода
	UserAdmin     = "user.admin"     // пользователи, роли, соответствие групп каталога ролям
)

// Роли (auth.role).
const (
	RoleViewer     = "viewer"
	RoleEditor     = "editor"
	RoleClassOwner = "class_owner"
	RoleImport     = "import_admin"
	RoleAuditor    = "auditor"
	RoleAccounting = "accounting_admin"
	RoleAdmin      = "admin"
)

var allPerms = []string{CIRead, CIWrite, RefWrite, NamingAdmin, ImportRead, ImportRun, AuditRead, BaselineRead, BaselineWrite, PeriodManage, UserAdmin}

var rolePerms = map[string][]string{
	RoleViewer:     {CIRead, BaselineRead},
	RoleEditor:     {CIRead, BaselineRead, CIWrite},
	RoleClassOwner: {CIRead, BaselineRead, CIWrite},
	RoleImport:     {CIRead, BaselineRead, ImportRead, ImportRun},
	// Аудитор видит всё, но не правит ничего (E7.2/ФСТЭК № 117: независимый контроль).
	RoleAuditor:    {CIRead, BaselineRead, ImportRead, AuditRead},
	RoleAccounting: {CIRead, BaselineRead, BaselineWrite, PeriodManage},
	RoleAdmin:      allPerms,
}

// PermsOf возвращает права роли (копия, отсортирована).
func PermsOf(role string) []string {
	out := append([]string(nil), rolePerms[role]...)
	sort.Strings(out)
	return out
}

// Grant — роль пользователя; Class непуст у class_owner (роль действует только для КЕ этого класса).
type Grant struct {
	Role  string
	Class string
}

// Identity — вошедший пользователь.
type Identity struct {
	ID          int64
	Username    string
	DisplayName string
	Source      string
	Grants      []Grant
}

// Can сообщает, есть ли у пользователя право perm хотя бы по одной роли. Для CIWrite область класса
// проверяет CanWriteClass.
func (i Identity) Can(perm string) bool {
	for _, g := range i.Grants {
		for _, p := range rolePerms[g.Role] {
			if p == perm {
				return true
			}
		}
	}
	return false
}

// CanWriteClass — можно ли править КЕ класса class: роль с правом ci.write без ограничения по классу
// (редактор, администратор) либо class_owner именно для этого класса.
func (i Identity) CanWriteClass(class string) bool {
	for _, g := range i.Grants {
		writes := false
		for _, p := range rolePerms[g.Role] {
			if p == CIWrite {
				writes = true
			}
		}
		if writes && (g.Class == "" || g.Class == class) {
			return true
		}
	}
	return false
}

// RoleNames — коды ролей без повторов (для отображения).
func (i Identity) RoleNames() []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range i.Grants {
		name := g.Role
		if g.Class != "" {
			name += ":" + g.Class
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
