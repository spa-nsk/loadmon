// E7.3 через настоящий роутер: REST API /api/v1 — токен, RBAC, CRUD КЕ/связей/компонентов,
// source=api в журнале изменений, ?asof= только для чтения, спецификация OpenAPI.
package ui_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/services/auth"
	"cmdb/internal/testutil"
)

// sendJSON выполняет запрос с телом JSON (пусто — без тела) и заданными заголовками (обычно Authorization).
func sendJSON(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)
	return w
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// issueAPIToken создаёт учётную запись с ролью и выпускает ей токен API.
func issueAPIToken(t *testing.T, svc *auth.Service, role, class string) (username, token string) {
	t.Helper()
	username = mkUser(t, svc, role, class)
	db := testutil.DB(t)
	var uid int64
	if err := db.QueryRowContext(t.Context(), `SELECT id FROM auth."user" WHERE username = $1`, username).Scan(&uid); err != nil {
		t.Fatalf("поиск id пользователя %q: %v", username, err)
	}
	token, _, err := auth.IssueToken(t.Context(), 1, uid, "test-token", 0)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	return username, token
}

func jsonBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("ответ не JSON: %v\nbody: %s", err, w.Body.String())
	}
	return m
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("не число: %q: %v", s, err)
	}
	return n
}

// TestAPI_OpenAPISpecPublic — спецификация отдаётся без аутентификации.
func TestAPI_OpenAPISpecPublic(t *testing.T) {
	setup(t)
	w := send("GET", "/api/v1/openapi.yaml", nil, false)
	if w.Code != 200 {
		t.Fatalf("GET /api/v1/openapi.yaml: status=%d", w.Code)
	}
	mustContain(t, "спецификация OpenAPI", w.Body.String(), "openapi: 3.0.3", "/cis:")
}

// TestAPI_AuthAndRBAC — без токена/сессии 401 JSON; с токеном без права ci.write 403 JSON;
// с достаточным правом — 200.
func TestAPI_AuthAndRBAC(t *testing.T) {
	svc := withAuth(t, 0)
	_, viewerToken := issueAPIToken(t, svc, auth.RoleViewer, "")

	w := sendJSON("GET", "/api/v1/cis", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("без токена: status=%d, want 401", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("без токена: Content-Type=%q, want JSON", ct)
	}
	body := jsonBody(t, w)
	if s, _ := body["error"].(string); s == "" {
		t.Errorf("без токена: тело=%v, ожидалось непустое поле error", body)
	}

	// viewer: читать можно, писать нельзя (403 JSON, не HTML)
	if w = sendJSON("GET", "/api/v1/cis", "", bearer(viewerToken)); w.Code != 200 {
		t.Fatalf("viewer GET /api/v1/cis: status=%d body=%s", w.Code, w.Body.String())
	}
	w = sendJSON("POST", "/api/v1/cis", `{"class":"server","location_code":"`+testutil.TestLocation+`","reason":"тест"}`, bearer(viewerToken))
	if w.Code != http.StatusForbidden {
		t.Fatalf("viewer POST /api/v1/cis: status=%d body=%s, want 403", w.Code, w.Body.String())
	}
	mustContain(t, "403 JSON", w.Body.String(), "error")
}

// TestAPI_CreateUpdateLifecycle — создание, правка (оптимистическая блокировка), смена стадии ЖЦ;
// change_set у всех трёх — source=api (plan.md E7.3), не manual.
func TestAPI_CreateUpdateLifecycle(t *testing.T) {
	svc := withAuth(t, 0)
	_, token := issueAPIToken(t, svc, auth.RoleEditor, "")
	hdr := bearer(token)

	w := sendJSON("POST", "/api/v1/cis", `{"class":"server","location_code":"`+testutil.TestLocation+`","reason":"создание через API"}`, hdr)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/cis: status=%d body=%s", w.Code, w.Body.String())
	}
	created := jsonBody(t, w)
	id := int64(created["id"].(float64))
	if name, _ := created["name"].(string); name == "" {
		t.Errorf("создание: имя не выделено, ответ=%v", created)
	}

	// GET карточки
	path := fmt.Sprintf("/api/v1/cis/%d", id)
	if w = sendJSON("GET", path, "", hdr); w.Code != 200 {
		t.Fatalf("GET %s: status=%d", path, w.Code)
	}
	detail := jsonBody(t, w)
	ciRow, _ := detail["ci"].(map[string]any)
	version := mustAtoi(t, ciRow["row_version"].(string))

	// PATCH с верной версией — ок; с устаревшей — 409
	patchBody := fmt.Sprintf(`{"expected_version":%d,"reason":"правка","description":"через API"}`, version)
	if w = sendJSON("PATCH", path, patchBody, hdr); w.Code != 200 {
		t.Fatalf("PATCH %s: status=%d body=%s", path, w.Code, w.Body.String())
	}
	if w = sendJSON("PATCH", path, patchBody, hdr); w.Code != http.StatusConflict {
		t.Fatalf("PATCH с устаревшей версией: status=%d, want 409", w.Code)
	}

	// смена стадии ЖЦ
	w = sendJSON("GET", path, "", hdr)
	newVersion := mustAtoi(t, jsonBody(t, w)["ci"].(map[string]any)["row_version"].(string))
	lcBody := fmt.Sprintf(`{"expected_version":%d,"to_stage":"Ordered","reason":"переход"}`, newVersion)
	if w = sendJSON("POST", path+"/lifecycle", lcBody, hdr); w.Code != 200 {
		t.Fatalf("POST lifecycle: status=%d body=%s", w.Code, w.Body.String())
	}

	// журнал изменений: три change set, все source=api
	if w = sendJSON("GET", path+"/history", "", hdr); w.Code != 200 {
		t.Fatalf("GET history: status=%d", w.Code)
	}
	hist := jsonBody(t, w)
	sets, _ := hist["change_sets"].([]any)
	if len(sets) < 3 {
		t.Fatalf("history: %d change set-ов, want >= 3: %v", len(sets), sets)
	}
	for _, s := range sets {
		cs := s.(map[string]any)
		if cs["source"] != "api" {
			t.Errorf("change_set id=%v source=%v, want api", cs["id"], cs["source"])
		}
	}
}

// TestAPI_RelationsAndComponents — создание/список/закрытие связи, установка компонента.
func TestAPI_RelationsAndComponents(t *testing.T) {
	svc := withAuth(t, 0)
	_, token := issueAPIToken(t, svc, auth.RoleEditor, "")
	hdr := bearer(token)

	mk := func() int64 {
		w := sendJSON("POST", "/api/v1/cis", `{"class":"server","location_code":"`+testutil.TestLocation+`","reason":"фикстура"}`, hdr)
		if w.Code != http.StatusCreated {
			t.Fatalf("создание фикстуры: status=%d body=%s", w.Code, w.Body.String())
		}
		return int64(jsonBody(t, w)["id"].(float64))
	}
	src, dst := mk(), mk()

	relBody := fmt.Sprintf(`{"rel_type":"depends_on","dst_ci_id":%d,"reason":"связь"}`, dst)
	w := sendJSON("POST", fmt.Sprintf("/api/v1/cis/%d/relations", src), relBody, hdr)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST relations: status=%d body=%s", w.Code, w.Body.String())
	}
	relID := int64(jsonBody(t, w)["id"].(float64))

	if w = sendJSON("GET", fmt.Sprintf("/api/v1/cis/%d/relations", src), "", hdr); w.Code != 200 {
		t.Fatalf("GET relations: status=%d", w.Code)
	}
	rels, _ := jsonBody(t, w)["relations"].([]any)
	if len(rels) != 1 {
		t.Fatalf("GET relations: %d записей, want 1: %v", len(rels), rels)
	}

	w = sendJSON("POST", fmt.Sprintf("/api/v1/relations/%d/close", relID), `{"expected_version":1,"reason":"закрытие"}`, hdr)
	if w.Code != 200 {
		t.Fatalf("POST relations close: status=%d body=%s", w.Code, w.Body.String())
	}

	w = sendJSON("POST", fmt.Sprintf("/api/v1/cis/%d/components", src), `{"component_type":"disk","slot":"1","reason":"установка"}`, hdr)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST components: status=%d body=%s", w.Code, w.Body.String())
	}
	if w = sendJSON("GET", fmt.Sprintf("/api/v1/cis/%d/components", src), "", hdr); w.Code != 200 {
		t.Fatalf("GET components: status=%d", w.Code)
	}
	comps, _ := jsonBody(t, w)["components"].([]any)
	if len(comps) != 1 {
		t.Fatalf("GET components: %d записей, want 1: %v", len(comps), comps)
	}
}

// TestAPI_AsOfReadOnly — с ?asof= изменяющий запрос отклоняется 409 JSON (E6.1, тот же фильтр, что у UI).
func TestAPI_AsOfReadOnly(t *testing.T) {
	svc := withAuth(t, 0)
	_, token := issueAPIToken(t, svc, auth.RoleEditor, "")
	w := sendJSON("POST", "/api/v1/cis?asof=2020-01-01T00:00:00Z",
		`{"class":"server","location_code":"`+testutil.TestLocation+`","reason":"на дату"}`, bearer(token))
	if w.Code != http.StatusConflict {
		t.Fatalf("POST с ?asof=: status=%d body=%s, want 409", w.Code, w.Body.String())
	}
	mustContain(t, "409 JSON", w.Body.String(), "error")
}

// TestAPI_NotFound — несуществующая КЕ — 404 JSON.
func TestAPI_NotFound(t *testing.T) {
	svc := withAuth(t, 0)
	_, token := issueAPIToken(t, svc, auth.RoleViewer, "")
	w := sendJSON("GET", "/api/v1/cis/999999999", "", bearer(token))
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET несуществующей КЕ: status=%d, want 404", w.Code)
	}
	mustContain(t, "404 JSON", w.Body.String(), "error")
}
