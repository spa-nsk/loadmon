package auth

import "testing"

func TestIdentity_PermissionsMatrix(t *testing.T) {
	id := func(grants ...Grant) Identity { return Identity{Username: "x", Grants: grants} }
	cases := []struct {
		name string
		id   Identity
		perm string
		want bool
	}{
		{"viewer читает", id(Grant{Role: RoleViewer}), CIRead, true},
		{"viewer не пишет", id(Grant{Role: RoleViewer}), CIWrite, false},
		{"editor пишет", id(Grant{Role: RoleEditor}), CIWrite, true},
		{"editor не админ", id(Grant{Role: RoleEditor}), UserAdmin, false},
		{"auditor читает журнал", id(Grant{Role: RoleAuditor}), AuditRead, true},
		{"auditor не правит (независимый контроль)", id(Grant{Role: RoleAuditor}), CIWrite, false},
		{"auditor не запускает импорт", id(Grant{Role: RoleAuditor}), ImportRun, false},
		{"import_admin запускает импорт", id(Grant{Role: RoleImport}), ImportRun, true},
		{"import_admin не правит КЕ вручную", id(Grant{Role: RoleImport}), CIWrite, false},
		{"accounting_admin закрывает период", id(Grant{Role: RoleAccounting}), PeriodManage, true},
		{"editor не закрывает период", id(Grant{Role: RoleEditor}), PeriodManage, false},
		{"admin всё", id(Grant{Role: RoleAdmin}), NamingAdmin, true},
		{"без ролей ничего", id(), CIRead, false},
		{"неизвестная роль ничего", id(Grant{Role: "ghost"}), CIRead, false},
		{"права суммируются", id(Grant{Role: RoleViewer}, Grant{Role: RoleAccounting}), PeriodManage, true},
	}
	for _, c := range cases {
		if got := c.id.Can(c.perm); got != c.want {
			t.Errorf("%s: Can(%s) = %v, want %v", c.name, c.perm, got, c.want)
		}
	}
}

func TestIdentity_CanWriteClass(t *testing.T) {
	owner := Identity{Grants: []Grant{{Role: RoleClassOwner, Class: "server"}}}
	if !owner.CanWriteClass("server") || owner.CanWriteClass("vm") {
		t.Error("class_owner(server) должен править server и не vm")
	}
	if !(Identity{Grants: []Grant{{Role: RoleEditor}}}).CanWriteClass("anything") {
		t.Error("editor правит любой класс")
	}
	if (Identity{Grants: []Grant{{Role: RoleViewer}}}).CanWriteClass("server") {
		t.Error("viewer ничего не правит")
	}
	mixed := Identity{Grants: []Grant{{Role: RoleClassOwner, Class: "server"}, {Role: RoleClassOwner, Class: "vm"}}}
	if !mixed.CanWriteClass("vm") || mixed.CanWriteClass("datastore") {
		t.Error("несколько class_owner суммируются по классам")
	}
	if (Identity{Grants: []Grant{{Role: RoleAuditor}}}).CanWriteClass("server") {
		t.Error("auditor не правит")
	}
}

func TestGroupKeys(t *testing.T) {
	got := groupKeys("CN=CMDB-Admins,OU=Groups,DC=corp,DC=local")
	if len(got) != 2 || got[0] != "cn=cmdb-admins,ou=groups,dc=corp,dc=local" || got[1] != "cmdb-admins" {
		t.Errorf("groupKeys = %v", got)
	}
	if got := groupKeys("не dn"); len(got) != 1 {
		t.Errorf("groupKeys(не DN) = %v, want только исходная строка", got)
	}
}
