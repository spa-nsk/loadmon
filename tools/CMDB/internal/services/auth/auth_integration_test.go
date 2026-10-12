// E7.1/E7.2: реальный PostgreSQL и встроенный LDAP-сервер (ldaptest).
package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cmdb/internal/services/auth"
	"cmdb/internal/services/auth/ldaptest"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

func fastSvc(ldapCfg *auth.LDAPConfig) *auth.Service {
	return &auth.Service{LDAP: ldapCfg, IdleTimeout: 2 * time.Second, MaxAge: 5 * time.Second, MaxAttempts: 3, LockWindow: 600 * time.Millisecond, BcryptCost: 4}
}

const goodPass = "correct-horse-battery"

func TestLocal_LoginSessionLockoutAndRoles(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	svc := fastSvc(nil)
	name := "loc-" + testutil.Suffix()

	id, err := svc.CreateLocal(ctx, 1, name, "Локальный", name+"@corp.local", goodPass)
	if err != nil {
		t.Fatalf("CreateLocal: %v", err)
	}
	if _, err := svc.CreateLocal(ctx, 1, name, "", "", goodPass); !errors.Is(err, errDuplicate()) {
		t.Errorf("повторное имя: err = %v, want ErrDuplicate", err)
	}
	if _, err := svc.CreateLocal(ctx, 1, name+"x", "", "", "short"); err == nil {
		t.Error("слишком короткий пароль должен быть отклонён")
	}

	// без ролей войти нельзя
	if _, _, err := svc.Login(ctx, name, goodPass, "10.0.0.1", "test"); !errors.Is(err, auth.ErrNoAccess) {
		t.Fatalf("вход без ролей: err = %v, want ErrNoAccess", err)
	}
	if err := auth.GrantRole(ctx, 1, id, auth.RoleClassOwner, ""); err == nil {
		t.Error("class_owner без класса должен быть отклонён")
	}
	if err := auth.GrantRole(ctx, 1, id, "ghost", ""); err == nil {
		t.Error("неизвестная роль должна быть отклонена")
	}
	if err := auth.GrantRole(ctx, 1, id, auth.RoleClassOwner, "server"); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if err := auth.GrantRole(ctx, 1, id, auth.RoleViewer, ""); err != nil {
		t.Fatal(err)
	}

	token, ident, err := svc.Login(ctx, name, goodPass, "10.0.0.1", "test-agent")
	if err != nil || token == "" {
		t.Fatalf("Login: token=%q err=%v", token, err)
	}
	if !ident.Can(auth.CIWrite) || !ident.CanWriteClass("server") || ident.CanWriteClass("vm") || ident.Can(auth.UserAdmin) {
		t.Errorf("права после входа: %+v", ident.RoleNames())
	}
	// в БД — только хеш токена
	db := testutil.DB(t)
	var n int
	db.QueryRow(`SELECT count(*) FROM auth.session WHERE token_hash = $1`, token).Scan(&n)
	if n != 0 {
		t.Error("токен сессии хранится в БД открытым текстом")
	}
	got, err := svc.Authenticate(ctx, token)
	if err != nil || got.ID != id {
		t.Fatalf("Authenticate: %+v err=%v", got, err)
	}
	if _, err := svc.Authenticate(ctx, token+"x"); !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("подделанный токен: err = %v, want ErrSessionExpired", err)
	}

	// выход закрывает сессию
	svc.Logout(ctx, token, id, "10.0.0.1")
	if _, err := svc.Authenticate(ctx, token); !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("после выхода: err = %v, want ErrSessionExpired", err)
	}

	// неверный пароль: текст одинаков для «нет пользователя» и «неверный пароль»
	if _, _, err := svc.Login(ctx, name, "wrong-password-1", "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("неверный пароль: err = %v", err)
	}
	if _, _, err := svc.Login(ctx, "нет-такого-"+name, goodPass, "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("неизвестный пользователь: err = %v, want тот же ErrInvalidCredentials", err)
	}
}

func TestLocal_LockoutCountsConsecutiveFailuresAndExpires(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	svc := fastSvc(nil)
	name := "lock-" + testutil.Suffix()
	id, _ := svc.CreateLocal(ctx, 1, name, "", "", goodPass)
	auth.GrantRole(ctx, 1, id, auth.RoleViewer, "")

	bad := func() error { _, _, err := svc.Login(ctx, name, "wrong-password-1", "", ""); return err }
	// 2 неудачи + успех + 2 неудачи: не подряд 3 — не блокируется
	bad()
	bad()
	if _, _, err := svc.Login(ctx, name, goodPass, "", ""); err != nil {
		t.Fatalf("вход после 2 неудач: %v", err)
	}
	bad()
	bad()
	if _, _, err := svc.Login(ctx, name, goodPass, "", ""); err != nil {
		t.Fatalf("успех сбрасывает счётчик, вход после 2 неудач должен пройти: %v", err)
	}

	// 3 неудачи подряд — блокировка: даже верный пароль отклоняется
	bad()
	bad()
	bad()
	if _, _, err := svc.Login(ctx, name, goodPass, "", ""); !errors.Is(err, auth.ErrLocked) {
		t.Fatalf("вход при блокировке с верным паролем: err = %v, want ErrLocked", err)
	}
	// блокировка временная
	time.Sleep(700 * time.Millisecond)
	if _, _, err := svc.Login(ctx, name, goodPass, "", ""); err != nil {
		t.Fatalf("после окна блокировки вход должен пройти: %v", err)
	}

	// перебор неизвестных имён тоже ограничен
	ghost := "ghost-" + testutil.Suffix()
	for i := 0; i < 3; i++ {
		svc.Login(ctx, ghost, "x-wrong-password", "", "")
	}
	if _, _, err := svc.Login(ctx, ghost, goodPass, "", ""); !errors.Is(err, auth.ErrLocked) {
		t.Errorf("неизвестное имя после 3 попыток: err = %v, want ErrLocked (перебор имён ограничен)", err)
	}
}

func TestSession_IdleAbsoluteDisabledAndPasswordChange(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	svc := &auth.Service{IdleTimeout: 400 * time.Millisecond, MaxAge: 5 * time.Second, BcryptCost: 4}
	name := "sess-" + testutil.Suffix()
	id, _ := svc.CreateLocal(ctx, 1, name, "", "", goodPass)
	auth.GrantRole(ctx, 1, id, auth.RoleViewer, "")

	// простой
	tok, _, err := svc.Login(ctx, name, goodPass, "", "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	if _, err := svc.Authenticate(ctx, tok); !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("сессия после тайм-аута простоя: err = %v, want ErrSessionExpired", err)
	}
	if n, _ := svc.PurgeSessions(ctx); n < 1 {
		t.Errorf("PurgeSessions удалила %d, want ≥1 простаивавшую", n)
	}

	// абсолютный срок
	short := &auth.Service{IdleTimeout: time.Minute, MaxAge: 300 * time.Millisecond, BcryptCost: 4}
	tok, _, _ = short.Login(ctx, name, goodPass, "", "")
	time.Sleep(400 * time.Millisecond)
	if _, err := short.Authenticate(ctx, tok); !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("сессия после абсолютного срока: err = %v", err)
	}

	// отключение закрывает сессии и не пускает
	tok, _, _ = svc.Login(ctx, name, goodPass, "", "")
	if err := auth.SetEnabled(ctx, 1, id, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, tok); !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("сессия отключённого пользователя: err = %v", err)
	}
	if _, _, err := svc.Login(ctx, name, goodPass, "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("вход отключённого: err = %v, want ErrInvalidCredentials (причина — только в журнале)", err)
	}
	auth.SetEnabled(ctx, 1, id, true)

	// смена пароля закрывает все сессии
	tok, _, _ = svc.Login(ctx, name, goodPass, "", "")
	if err := svc.SetPassword(ctx, 1, id, "another-strong-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, tok); !errors.Is(err, auth.ErrSessionExpired) {
		t.Errorf("сессия после смены пароля: err = %v, want закрыта", err)
	}
	if _, _, err := svc.Login(ctx, name, "another-strong-pass", "", ""); err != nil {
		t.Errorf("вход с новым паролем: %v", err)
	}

	// действия попали в журнал действий (E7.6)
	var events int
	db.QueryRow(`SELECT count(*) FROM audit.event WHERE event_type IN ('login', 'user_create', 'user_enable', 'user_password_change') AND (details ->> 'username' = $1 OR (details ->> 'user_id')::bigint = $2)`, name, id).Scan(&events)
	if events < 4 {
		t.Errorf("событий журнала действий = %d, want ≥4 (вход, создание, отключение, смена пароля)", events)
	}
}

func TestRevokeRole_KeepsLastAdmin(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	svc := fastSvc(nil)
	// делаем тест независимым от чужих администраторов: временно отключаем прочих
	rows, _ := db.Query(`SELECT r.user_id FROM auth.user_role r JOIN auth."user" u ON u.id = r.user_id WHERE r.role_code = 'admin' AND u.enabled`)
	var others []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		others = append(others, id)
	}
	rows.Close()
	for _, o := range others {
		auth.SetEnabled(ctx, 1, o, false)
	}
	t.Cleanup(func() {
		for _, o := range others {
			auth.SetEnabled(ctx, 1, o, true)
		}
	})

	a, _ := svc.CreateLocal(ctx, 1, "adm-"+testutil.Suffix(), "", "", goodPass)
	auth.GrantRole(ctx, 1, a, auth.RoleAdmin, "")
	if err := auth.RevokeRole(ctx, 1, a, auth.RoleAdmin, ""); err == nil {
		t.Fatal("последнего администратора лишить роли нельзя")
	}
	b, _ := svc.CreateLocal(ctx, 1, "adm2-"+testutil.Suffix(), "", "", goodPass)
	auth.GrantRole(ctx, 1, b, auth.RoleAdmin, "")
	if err := auth.RevokeRole(ctx, 1, a, auth.RoleAdmin, ""); err != nil {
		t.Errorf("при втором администраторе роль снимается: %v", err)
	}
}

func TestLDAP_LoginGroupsToRolesAndFailures(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	sfx := testutil.Suffix()
	editors := "cn=cmdb-editors-" + sfx + ",ou=groups," + ldaptest.BaseDN
	admins := "CN=CMDB-Admins-" + sfx + ",OU=Groups," + ldaptest.BaseDN
	owners := "cn=cmdb-owners-" + sfx + ",ou=groups," + ldaptest.BaseDN

	jdoe, alice, bob, carol := "jdoe"+sfx, "alice"+sfx, "bob"+sfx, "carol"+sfx
	srv := ldaptest.Start(t, []ldaptest.User{
		{UID: jdoe, Display: "Иван Иванов", Mail: jdoe + "@corp.local", Password: "ldap-pass-1", Groups: []string{editors}},
		{UID: alice, Display: "Алиса", Password: "ldap-pass-2", Groups: []string{admins}}, // DN другого регистра, чем в сопоставлении
		{UID: bob, Display: "Боб", Password: "ldap-pass-3", Groups: []string{"cn=other," + ldaptest.BaseDN}},
		{UID: carol, Display: "Кэрол", Password: "ldap-pass-4", Groups: []string{owners}},
	})
	for _, m := range [][3]string{{"cmdb-editors-" + sfx, auth.RoleEditor, ""}, {"cn=cmdb-admins-" + sfx + ",ou=groups," + ldaptest.BaseDN, auth.RoleAdmin, ""}, {"cmdb-owners-" + sfx, auth.RoleClassOwner, "vm"}} {
		if err := auth.SetGroupRole(ctx, 1, m[0], m[1], m[2]); err != nil {
			t.Fatalf("SetGroupRole: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, k := range []string{"cmdb-editors-" + sfx, "cn=cmdb-admins-" + sfx + ",ou=groups," + ldaptest.BaseDN, "cmdb-owners-" + sfx} {
			auth.UnsetGroupRole(ctx, 1, k, auth.RoleEditor, "")
			auth.UnsetGroupRole(ctx, 1, k, auth.RoleAdmin, "")
			auth.UnsetGroupRole(ctx, 1, k, auth.RoleClassOwner, "vm")
		}
	})

	cfg := &auth.LDAPConfig{URL: srv.URL, BindDN: ldaptest.ServiceDN, BindPassword: ldaptest.ServicePass, UserBase: ldaptest.BaseDN, Kind: "ad", Timeout: 2 * time.Second}
	svc := fastSvc(cfg)

	// CN группы и полный DN (без учёта регистра) сопоставляются ролям; учётная запись создаётся при первом входе
	tok, id, err := svc.Login(ctx, jdoe, "ldap-pass-1", "", "")
	if err != nil || tok == "" {
		t.Fatalf("вход jdoe: %v", err)
	}
	if id.Source != "ldap" || id.DisplayName != "Иван Иванов" || !id.Can(auth.CIWrite) || id.Can(auth.UserAdmin) {
		t.Errorf("jdoe = %+v роли %v", id, id.RoleNames())
	}
	if _, aid, err := svc.Login(ctx, alice, "ldap-pass-2", "", ""); err != nil || !aid.Can(auth.UserAdmin) {
		t.Errorf("alice (DN другого регистра) должна получить admin: %v %v", err, aid.RoleNames())
	}
	if _, cid, err := svc.Login(ctx, carol, "ldap-pass-4", "", ""); err != nil || !cid.CanWriteClass("vm") || cid.CanWriteClass("server") {
		t.Errorf("carol: class_owner только для vm: %v %v", err, cid.RoleNames())
	}
	// вход по UPN тоже работает (фильтр AD)
	if _, _, err := svc.Login(ctx, jdoe+"@corp.local", "ldap-pass-1", "", ""); err != nil {
		t.Errorf("вход по userPrincipalName: %v", err)
	}
	// пользователь без сопоставленных групп — нет доступа
	if _, _, err := svc.Login(ctx, bob, "ldap-pass-3", "", ""); !errors.Is(err, auth.ErrNoAccess) {
		t.Errorf("bob без ролей: err = %v, want ErrNoAccess", err)
	}
	// неверный пароль и неизвестный пользователь — одна и та же ошибка
	if _, _, err := svc.Login(ctx, jdoe, "wrong-ldap-pass", "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("неверный пароль каталога: err = %v", err)
	}
	if _, _, err := svc.Login(ctx, "nobody"+sfx, "whatever-pass", "", ""); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("неизвестный в каталоге: err = %v", err)
	}
	// пустой пароль отклоняется без обращения к каталогу (простая привязка с пустым паролем — «анонимная»)
	before := srv.Binds
	if _, _, err := svc.Login(ctx, jdoe, "", "", ""); !errors.Is(err, auth.ErrInvalidCredentials) || srv.Binds != before {
		t.Errorf("пустой пароль: err = %v, привязок каталога +%d, want отказ без привязки", err, srv.Binds-before)
	}

	// группы изменились в каталоге: при следующем входе роли пересчитываются, ручные — сохраняются
	if err := auth.GrantRole(ctx, 1, id.ID, auth.RoleViewer, ""); err != nil {
		t.Fatal(err)
	}
	srv.SetUsers([]ldaptest.User{{UID: jdoe, Display: "Иван Иванов", Password: "ldap-pass-1", Groups: []string{admins}}})
	_, id2, err := svc.Login(ctx, jdoe, "ldap-pass-1", "", "")
	if err != nil || !id2.Can(auth.UserAdmin) {
		t.Fatalf("после смены групп jdoe должен стать admin: %v %v", err, id2.RoleNames())
	}
	names := id2.RoleNames()
	hasEditor, hasViewer := false, false
	for _, n := range names {
		hasEditor = hasEditor || n == auth.RoleEditor
		hasViewer = hasViewer || n == auth.RoleViewer
	}
	if hasEditor || !hasViewer {
		t.Errorf("роли после пересчёта = %v: editor (из старой группы) должен исчезнуть, назначенная вручную viewer — остаться", names)
	}
}

func TestLDAP_DirectoryFailureIsNotAFailedAttempt(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	sfx := testutil.Suffix()
	uid := "dir" + sfx
	srv := ldaptest.Start(t, []ldaptest.User{{UID: uid, Password: "ldap-pass-1"}})
	cfg := &auth.LDAPConfig{URL: srv.URL, BindDN: ldaptest.ServiceDN, BindPassword: "wrong-service-pass", UserBase: ldaptest.BaseDN, Timeout: time.Second}
	svc := fastSvc(cfg)

	for i := 0; i < 5; i++ { // больше порога блокировки
		_, _, err := svc.Login(ctx, uid, "ldap-pass-1", "", "")
		if err == nil || errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrLocked) {
			t.Fatalf("сбой каталога (учётная запись поиска не принята): err = %v, want ошибка недоступности", err)
		}
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM auth.login_event WHERE username = $1`, uid).Scan(&n)
	if n != 0 {
		t.Errorf("сбои каталога записаны как попытки входа пользователя (%d) — так недоступный каталог блокировал бы всех", n)
	}
	// адрес недоступен вовсе
	dead := fastSvc(&auth.LDAPConfig{URL: "ldap://127.0.0.1:1", UserBase: ldaptest.BaseDN, Timeout: 300 * time.Millisecond})
	if _, _, err := dead.Login(ctx, uid, "ldap-pass-1", "", ""); err == nil || errors.Is(err, auth.ErrInvalidCredentials) {
		t.Errorf("недоступный адрес каталога: err = %v", err)
	}
}

func errDuplicate() error { return svcerr.ErrDuplicate }
