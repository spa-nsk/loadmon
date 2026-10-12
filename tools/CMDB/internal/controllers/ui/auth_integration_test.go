// E7.1/E7.2 через настоящий роутер: вход, cookie сессии, права по маршрутам и области класса КЕ,
// выход, блокировка, управление пользователями.
package ui_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/filters"
	"cmdb/internal/services/auth"
	"cmdb/internal/testutil"
)

const testPass = "test-password-123"

// withAuth включает аутентификацию на время теста и возвращает прежний режим (остальные сквозные
// тесты идут в режиме без входа).
func withAuth(t *testing.T, maxAttempts int) *auth.Service {
	t.Helper()
	setup(t)
	svc := &auth.Service{MaxAttempts: maxAttempts, BcryptCost: 4}
	filters.SetAuth(svc, false)
	t.Cleanup(func() { filters.SetAuth(&auth.Service{}, true) })
	return svc
}

func mkUser(t *testing.T, svc *auth.Service, role, class string) string {
	t.Helper()
	name := "u-" + role + "-" + testutil.Suffix()
	id, err := svc.CreateLocal(t.Context(), 1, name, "Тест "+role, "", testPass)
	if err != nil {
		t.Fatalf("CreateLocal: %v", err)
	}
	if err := auth.GrantRole(t.Context(), 1, id, role, class); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	return name
}

func post(path string, form url.Values, hx bool, cookie string) *httptest.ResponseRecorder {
	return sendWithCookie("POST", path, form, hx, cookie)
}

// login выполняет вход через форму и возвращает cookie сессии ("" — вход не удался).
func login(t *testing.T, name, pass string) (cookie string, w *httptest.ResponseRecorder) {
	t.Helper()
	w = post("/login", url.Values{"username": {name}, "password": {pass}}, false, "")
	for _, c := range w.Result().Cookies() {
		if c.Name == filters.SessionCookie && c.Value != "" {
			return c.Name + "=" + c.Value, w
		}
	}
	return "", w
}

func TestAuth_LoginRedirectsAndCookieFlags(t *testing.T) {
	svc := withAuth(t, 0)
	name := mkUser(t, svc, auth.RoleViewer, "")

	// без сессии: страница — на вход с возвратом; HTMX — 401 + HX-Redirect; публичное — без входа
	w := send("GET", "/cis?q=abc", nil, false)
	if w.Code != http.StatusFound || !strings.HasPrefix(w.Header().Get("Location"), "/login?next=") {
		t.Fatalf("GET /cis без сессии: status=%d Location=%q, want 302 на /login?next=…", w.Code, w.Header().Get("Location"))
	}
	if w = send("POST", "/cis", url.Values{"class": {"server"}}, true); w.Code != http.StatusUnauthorized || w.Header().Get("HX-Redirect") != "/login" {
		t.Errorf("HTMX POST без сессии: status=%d HX-Redirect=%q", w.Code, w.Header().Get("HX-Redirect"))
	}
	if w = send("POST", "/cis", url.Values{"class": {"server"}}, false); w.Code != http.StatusUnauthorized {
		t.Errorf("POST без сессии: status=%d, want 401", w.Code)
	}
	if w = send("GET", "/healthz", nil, false); w.Code != 200 {
		t.Errorf("/healthz должен быть публичным, status=%d", w.Code)
	}

	form := send("GET", "/login?next=/cis", nil, false)
	mustContain(t, "форма входа", form.Body.String(), `name="username"`, `name="password"`, `name="_xsrf"`, `value="/cis"`)

	// неверные данные: 401, одно сообщение для любой причины, cookie нет
	for _, c := range [][2]string{{name, "wrong-password-1"}, {"no-such-" + name, testPass}} {
		ck, w := login(t, c[0], c[1])
		if ck != "" || w.Code != http.StatusUnauthorized {
			t.Errorf("вход %q/неверный: cookie=%q status=%d, want 401 без cookie", c[0], ck, w.Code)
		}
		mustContain(t, "сообщение об ошибке", w.Body.String(), "Неверное имя пользователя или пароль")
	}

	// успех: 302 на next, cookie HttpOnly + SameSite=Lax, без Secure по http
	req := httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{"username": {name}, "password": {testPass}, "next": {"/cis?q=abc"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/cis?q=abc" {
		t.Fatalf("вход: status=%d Location=%q", w.Code, w.Header().Get("Location"))
	}
	var sess *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == filters.SessionCookie {
			sess = c
		}
	}
	if sess == nil || !sess.HttpOnly || sess.SameSite != http.SameSiteLaxMode || sess.Secure || len(sess.Value) < 40 {
		t.Fatalf("cookie сессии = %+v, want HttpOnly, SameSite=Lax, длинный токен", sess)
	}
	// за TLS-прокси (X-Forwarded-Proto: https) cookie получает Secure
	req = httptest.NewRequest("POST", "/login", strings.NewReader(url.Values{"username": {name}, "password": {testPass}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-Proto", "https")
	w = httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)
	secure := false
	for _, c := range w.Result().Cookies() {
		if c.Name == filters.SessionCookie {
			secure = c.Secure
		}
	}
	if !secure {
		t.Error("за https-прокси cookie сессии должна быть Secure")
	}

	// открытый редирект: next на чужой сайт игнорируется
	for _, next := range []string{"//evil.example", "https://evil.example", "/\\evil"} {
		w = post("/login", url.Values{"username": {name}, "password": {testPass}, "next": {next}}, false, "")
		if loc := w.Header().Get("Location"); loc != "/" {
			t.Errorf("next=%q: Location=%q, want «/» (открытый редирект)", next, loc)
		}
	}

	// с cookie страница открывается, имя и выход видны, недоступное меню скрыто
	ck, _ := login(t, name, testPass)
	page := sendWithCookie("GET", "/cis", nil, false, ck)
	if page.Code != 200 {
		t.Fatalf("GET /cis с сессией: status=%d", page.Code)
	}
	mustContain(t, "шапка", page.Body.String(), "Тест viewer", `action="/logout"`)
	if strings.Contains(page.Body.String(), "Создать КЕ") || strings.Contains(page.Body.String(), ">Пользователи<") {
		t.Error("наблюдатель не должен видеть пункты «Создать КЕ» и «Пользователи»")
	}
	// подделанная cookie
	if w := sendWithCookie("GET", "/cis", nil, false, filters.SessionCookie+"=forged-token"); w.Code != http.StatusFound {
		t.Errorf("подделанная cookie: status=%d, want 302 на вход", w.Code)
	}

	// выход закрывает сессию
	w = post("/logout", nil, false, ck)
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/login" {
		t.Errorf("выход: status=%d Location=%q", w.Code, w.Header().Get("Location"))
	}
	if w := sendWithCookie("GET", "/cis", nil, false, ck); w.Code != http.StatusFound {
		t.Errorf("после выхода старая cookie должна не работать, status=%d", w.Code)
	}
}

func TestAuth_LockoutAnd429(t *testing.T) {
	svc := withAuth(t, 3)
	name := mkUser(t, svc, auth.RoleViewer, "")
	for i := 0; i < 3; i++ {
		login(t, name, "wrong-password-1")
	}
	ck, w := login(t, name, testPass)
	if ck != "" || w.Code != http.StatusTooManyRequests {
		t.Errorf("вход при блокировке: cookie=%q status=%d, want 429 без cookie", ck, w.Code)
	}
	mustContain(t, "сообщение блокировки", w.Body.String(), "заблокирован")
}

func TestAuth_NoRolesIsForbiddenNotSession(t *testing.T) {
	svc := withAuth(t, 0)
	name := "norole-" + testutil.Suffix()
	if _, err := svc.CreateLocal(t.Context(), 1, name, "", "", testPass); err != nil {
		t.Fatal(err)
	}
	ck, w := login(t, name, testPass)
	if ck != "" || w.Code != http.StatusForbidden {
		t.Errorf("вход без ролей: cookie=%q status=%d, want 403 без cookie", ck, w.Code)
	}
}

// Матрица прав по маршрутам: у каждой роли — что разрешено и что отклонено (403, а не тихо ничего).
func TestRBAC_RoutePermissionsAndClassScope(t *testing.T) {
	svc := withAuth(t, 0)
	db := testutil.DB(t)
	sess := map[string]string{}
	for _, r := range []struct{ key, role, class string }{
		{"viewer", auth.RoleViewer, ""}, {"editor", auth.RoleEditor, ""}, {"auditor", auth.RoleAuditor, ""},
		{"import", auth.RoleImport, ""}, {"acc", auth.RoleAccounting, ""}, {"admin", auth.RoleAdmin, ""}, {"owner", auth.RoleClassOwner, "server"},
	} {
		ck, _ := login(t, mkUser(t, svc, r.role, r.class), testPass)
		if ck == "" {
			t.Fatalf("вход %s не удался", r.key)
		}
		sess[r.key] = ck
	}
	forbidden := func(w *httptest.ResponseRecorder) bool { return w.Code == http.StatusForbidden }

	// чтение
	for key, want := range map[string]int{"viewer": 200, "editor": 200, "auditor": 200, "import": 200, "acc": 200, "admin": 200} {
		if w := sendWithCookie("GET", "/cis", nil, false, sess[key]); w.Code != want {
			t.Errorf("%s: GET /cis = %d, want %d", key, w.Code, want)
		}
	}
	// журнал изменений — только audit.read (аудитор, админ)
	for key, allowed := range map[string]bool{"auditor": true, "admin": true, "viewer": false, "editor": false, "import": false, "acc": false} {
		if w := sendWithCookie("GET", "/changes", nil, false, sess[key]); forbidden(w) == allowed {
			t.Errorf("%s: GET /changes = %d, доступ должен быть %v", key, w.Code, allowed)
		}
	}
	// импорт: читать — import_admin/аудитор/админ; запускать — только import_admin/админ
	for key, allowed := range map[string]bool{"import": true, "auditor": true, "admin": true, "viewer": false, "editor": false} {
		if w := sendWithCookie("GET", "/import", nil, false, sess[key]); forbidden(w) == allowed {
			t.Errorf("%s: GET /import = %d, доступ должен быть %v", key, w.Code, allowed)
		}
	}
	for key, allowed := range map[string]bool{"import": true, "admin": true, "auditor": false, "editor": false} {
		if w := post("/import/connectors/x/test", nil, true, sess[key]); forbidden(w) == allowed {
			t.Errorf("%s: POST /import/connectors/…/test = %d, доступ должен быть %v", key, w.Code, allowed)
		}
	}
	// учётные периоды и срезы: писать — accounting_admin/админ; аудитор и редактор — нет
	for key, allowed := range map[string]bool{"acc": true, "admin": true, "editor": false, "auditor": false, "viewer": false} {
		if w := post("/periods/close", url.Values{"month": {"март"}}, true, sess[key]); forbidden(w) == allowed {
			t.Errorf("%s: POST /periods/close = %d, доступ должен быть %v (при доступе — ошибка проверки, не 403)", key, w.Code, allowed)
		}
		if w := post("/baselines", url.Values{"name": {""}}, true, sess[key]); forbidden(w) == allowed {
			t.Errorf("%s: POST /baselines = %d, доступ должен быть %v", key, w.Code, allowed)
		}
	}
	if w := sendWithCookie("GET", "/baselines", nil, false, sess["viewer"]); w.Code != 200 {
		t.Errorf("наблюдатель читает срезы: %d", w.Code)
	}
	// справочники: читать все, писать — только админ; пользователи и группы — только админ
	if w := sendWithCookie("GET", "/admin/ref", nil, false, sess["viewer"]); w.Code != 200 {
		t.Errorf("наблюдатель читает справочники: %d", w.Code)
	}
	for key, allowed := range map[string]bool{"admin": true, "editor": false, "import": false} {
		if w := post("/admin/ref/company", url.Values{}, true, sess[key]); forbidden(w) == allowed {
			t.Errorf("%s: POST /admin/ref/company = %d, доступ должен быть %v", key, w.Code, allowed)
		}
		if w := sendWithCookie("GET", "/admin/users", nil, false, sess[key]); forbidden(w) == allowed {
			t.Errorf("%s: GET /admin/users = %d, доступ должен быть %v", key, w.Code, allowed)
		}
	}
	// маршрут без описания прав: изменяющий запрос отклоняется по умолчанию даже у администратора
	if w := post("/unknown/route", nil, true, sess["admin"]); !forbidden(w) {
		t.Errorf("POST на неописанный маршрут: %d, want 403 (запрет по умолчанию)", w.Code)
	}

	// ввод КЕ: viewer/auditor — 403; editor — создаёт; class_owner(server) — только server
	create := func(key, class string) *httptest.ResponseRecorder {
		return post("/cis", url.Values{"class": {class}, "location_code": {testutil.TestLocation}, "reason": {"права"}}, true, sess[key])
	}
	for _, key := range []string{"viewer", "auditor", "import", "acc"} {
		if w := create(key, "server"); !forbidden(w) {
			t.Errorf("%s: создание КЕ = %d, want 403", key, w.Code)
		}
	}
	w := create("editor", "server")
	if !strings.HasPrefix(w.Header().Get("HX-Redirect"), "/cis/") {
		t.Fatalf("редактор не смог создать КЕ: status=%d trigger=%q", w.Code, trigger(w))
	}
	serverID := firstMatch(t, `^/cis/(\d+)`, w.Header().Get("HX-Redirect"))
	if w := create("owner", "server"); !strings.HasPrefix(w.Header().Get("HX-Redirect"), "/cis/") {
		t.Errorf("class_owner(server) должен создавать server: status=%d trigger=%q", w.Code, trigger(w))
	}
	if w := create("owner", "vm"); !forbidden(w) {
		t.Errorf("class_owner(server) не должен создавать vm: status=%d", w.Code)
	}
	// правка чужого класса: КЕ vm создаёт редактор, владелец server править её не может (класс берётся из БД, не из формы)
	w = create("editor", "vm")
	vmID := firstMatch(t, `^/cis/(\d+)`, w.Header().Get("HX-Redirect"))
	edit := url.Values{"row_version": {"1"}, "kii_object_present": {"1"}, "description": {"чужая правка"}, "reason": {"попытка"}}
	if w := post("/cis/"+vmID+"/update", edit, true, sess["owner"]); !forbidden(w) {
		t.Errorf("class_owner(server) правит vm: status=%d, want 403", w.Code)
	}
	if w := post("/cis/"+serverID+"/update", edit, true, sess["owner"]); forbidden(w) || w.Code >= 400 {
		t.Errorf("class_owner(server) правит server: status=%d trigger=%q, want успех", w.Code, trigger(w))
	}
	var desc string
	db.QueryRow(`SELECT COALESCE(description, '') FROM cmdb.ci WHERE id = $1`, vmID).Scan(&desc)
	if desc == "чужая правка" {
		t.Error("отклонённая правка всё равно записана")
	}
	// в журнале изменений автор — вошедший пользователь, а не dev
	var uid int64
	db.QueryRow(`SELECT user_id FROM audit.change_set WHERE reason = 'права' ORDER BY id DESC LIMIT 1`).Scan(&uid)
	if uid <= 1 {
		t.Errorf("user_id набора изменений = %d: запись должна идти от вошедшего пользователя", uid)
	}
}

func TestUserAdmin_CreateUserGrantAndProtections(t *testing.T) {
	svc := withAuth(t, 0)
	admin := mkUser(t, svc, auth.RoleAdmin, "")
	ck, _ := login(t, admin, testPass)

	page := sendWithCookie("GET", "/admin/users", nil, false, ck)
	mustContain(t, "страница пользователей", page.Body.String(), "Пользователи", "Создать локального пользователя", admin)

	nu := "new-" + testutil.Suffix()
	wantFailure(t, "короткий пароль", post("/admin/users", url.Values{"username": {nu}, "password": {"short"}}, true, ck))
	w := post("/admin/users", url.Values{"username": {nu}, "display_name": {"Новый"}, "password": {testPass}, "role": {auth.RoleEditor}}, true, ck)
	wantSuccess(t, "создание пользователя", w)
	wantFailure(t, "дубликат имени", post("/admin/users", url.Values{"username": {nu}, "password": {testPass}}, true, ck))

	// созданный пользователь входит и получает свою роль
	nck, lw := login(t, nu, testPass)
	if nck == "" {
		t.Fatalf("созданный пользователь не вошёл: %d", lw.Code)
	}
	if w := sendWithCookie("GET", "/admin/users", nil, false, nck); w.Code != http.StatusForbidden {
		t.Errorf("редактор открыл /admin/users: %d", w.Code)
	}

	// отключение закрывает сессию; себя отключить нельзя
	var uid string
	for _, u := range mustUsers(t) {
		if u.Username == nu {
			uid = u.ID
		}
	}
	wantSuccess(t, "отключение", post("/admin/users/"+uid+"/enabled", url.Values{"enabled": {"0"}}, true, ck))
	if w := sendWithCookie("GET", "/cis", nil, false, nck); w.Code != http.StatusFound {
		t.Errorf("сессия отключённого пользователя должна быть закрыта, status=%d", w.Code)
	}
	var adminID string
	for _, u := range mustUsers(t) {
		if u.Username == admin {
			adminID = u.ID
		}
	}
	wantFailure(t, "самоотключение", post("/admin/users/"+adminID+"/enabled", url.Values{"enabled": {"0"}}, true, ck))
	// назначение роли и сопоставление группы
	wantSuccess(t, "назначение роли", post("/admin/users/"+uid+"/role", url.Values{"role": {auth.RoleViewer}}, true, ck))
	wantFailure(t, "class_owner без класса", post("/admin/users/"+uid+"/role", url.Values{"role": {auth.RoleClassOwner}}, true, ck))
	g := "grp-" + testutil.Suffix()
	wantSuccess(t, "группа → роль", post("/admin/groups", url.Values{"group": {g}, "role": {auth.RoleAuditor}}, true, ck))
	mustContain(t, "список групп", sendWithCookie("GET", "/admin/groups", nil, false, ck).Body.String(), g, auth.RoleAuditor)
	wantSuccess(t, "удаление сопоставления", post("/admin/groups/delete", url.Values{"group": {g}, "role": {auth.RoleAuditor}}, true, ck))
}

func mustUsers(t *testing.T) []auth.UserRow {
	t.Helper()
	u, err := auth.ListUsers(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return u
}
