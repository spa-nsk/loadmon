package itilium

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientValidation(t *testing.T) {
	if _, err := NewClient(Config{}); !errors.Is(err, ErrEmptyBaseURL) {
		t.Fatalf("ожидался ErrEmptyBaseURL, получено %v", err)
	}
	if _, err := NewClient(Config{BaseURL: "http://x/ITILIUM"}); !errors.Is(err, ErrEmptyLogin) {
		t.Fatalf("ожидался ErrEmptyLogin, получено %v", err)
	}
}

func TestFlexTypes(t *testing.T) {
	var b FlexBool
	for _, raw := range []string{`true`, `"true"`, `"TRUE"`, `"1"`} {
		if err := json.Unmarshal([]byte(raw), &b); err != nil || !b.Bool() {
			t.Fatalf("%s → %v %v", raw, b, err)
		}
	}
	for _, raw := range []string{`false`, `"false"`, `""`, `null`} {
		if err := json.Unmarshal([]byte(raw), &b); err != nil || b.Bool() {
			t.Fatalf("%s → %v %v", raw, b, err)
		}
	}
	var s FlexString
	if err := json.Unmarshal([]byte(`1`), &s); err != nil || s.String() != "1" {
		t.Fatalf("число: %q %v", s, err)
	}
	if err := json.Unmarshal([]byte(`"000000001"`), &s); err != nil || s.String() != "000000001" {
		t.Fatalf("строка: %q %v", s, err)
	}
}

func startTestServer(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := NewClient(Config{
		Name: "test", BaseURL: srv.URL + "/ITILIUM", Login: "user", Password: "pass",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return c, srv
}

func requireBasic(t *testing.T, r *http.Request) {
	t.Helper()
	u, p, ok := r.BasicAuth()
	if !ok || u != "user" || p != "pass" {
		t.Fatalf("BasicAuth: ok=%v user=%q pass=%q", ok, u, p)
	}
}

func TestExternalGetIncidentsList(t *testing.T) {
	c, _ := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireBasic(t, r)
		if r.Method != http.MethodPost || r.URL.Path != "/ITILIUM/hs/externalapi/getIncidentsList" {
			t.Fatalf("неверный запрос: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var f IncidentListFilter
		if err := json.Unmarshal(body, &f); err != nil {
			t.Fatal(err)
		}
		if f.StartFrom.String() != "0" || f.Count.String() != "2" || f.StateUUID == "" {
			t.Fatalf("фильтр: %+v", f)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Incidents":[{"linkUuid":"a","date":"27.09.2022 13:43:52","number":"0000000593","state":"Зарегистрировано","stateUuid":"s1","topic":"","description":"","availableStates":[{"state":"В работе","stateUuid":"s2"}]}]}`))
	})
	list, err := c.External().GetIncidentsList(context.Background(), IncidentListFilter{
		StartFrom: "0", Count: "2", StateUUID: "28643652-c3e4-11e4-bc1f-c631c2db3ae1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Number != "0000000593" || len(list[0].AvailableStates) != 1 {
		t.Fatalf("%+v", list)
	}
}

func TestExternalDetailAndFileAndChange(t *testing.T) {
	var seenChange bool
	c, _ := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireBasic(t, r)
		switch {
		case strings.HasSuffix(r.URL.Path, "/extGetDetailIncidentInfo"):
			w.Write([]byte(`{"uuid":"u1","number":"0000000605","mass":"false","ThereAreFiles":true,"grade":"0","AvailaibleStates":[{"Name":"В работе","UID":"s2"}],"FilesDefinitions":[{"Name":"a.xml","UID":"f1 "}],"ConfigurationItems":[{"Name":"Интернет","UID":"ci1","Cod":"058","Classifier":{"Name":"ПАК","UID":"cl1"}}]}`))
		case strings.HasSuffix(r.URL.Path, "/extGetFileData"):
			payload := base64.StdEncoding.EncodeToString([]byte("hello"))
			w.Write([]byte(`{"Name":"a.xml","Data":"` + payload + `"}`))
		case strings.HasSuffix(r.URL.Path, "/changeIncidentState"):
			seenChange = true
			var req ChangeIncidentStateRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.DocUUID != "u1" || req.NewStateUUID != "s2" {
				t.Fatalf("%+v", req)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("путь: %s", r.URL.Path)
		}
	})
	ext := c.External()
	d, err := ext.GetDetailIncidentInfo(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if d.Number != "0000000605" || d.Mass.Bool() || !d.ThereAreFiles.Bool() || d.Grade.String() != "0" {
		t.Fatalf("%+v", d)
	}
	if len(d.AvailableStates) != 1 || d.FilesDefinitions[0].FileID() != "f1" {
		t.Fatalf("states/files: %+v", d)
	}
	fd, err := ext.GetFileData(context.Background(), "f1")
	if err != nil || fd.Name != "a.xml" {
		t.Fatalf("%v %v", fd, err)
	}
	raw, _ := base64.StdEncoding.DecodeString(fd.Data)
	if string(raw) != "hello" {
		t.Fatalf("data: %q", raw)
	}
	if err := ext.ChangeIncidentState(context.Background(), "u1", "s2"); err != nil || !seenChange {
		t.Fatalf("change: %v seen=%v", err, seenChange)
	}
}

func TestExternalCatalogs(t *testing.T) {
	c, _ := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireBasic(t, r)
		switch {
		case strings.HasSuffix(r.URL.Path, "/getWorkgroups"):
			w.Write([]byte(`{"Workgroups":[{"Workgroup":"Провайдеры","WorkgroupUuid":"w1","WorkgroupCode":"0000000007"}]}`))
		case strings.HasSuffix(r.URL.Path, "/getPriorities"):
			w.Write([]byte(`{"Priorities":[{"Priority":"1_Высокий","PriorityUuid":"p1","PriorityCode":1}]}`))
		case strings.HasSuffix(r.URL.Path, "/getEmployers"):
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "empWorkgroupUuid") {
				t.Fatalf("ожидался empWorkgroupUuid: %s", body)
			}
			w.Write([]byte(`{"Employers":[{"Employer":"Админ","EmployerUuid":"e1","EmployerCode":"3"}]}`))
		default:
			t.Fatalf("путь: %s", r.URL.Path)
		}
	})
	ext := c.External()
	wg, err := ext.GetWorkgroups(context.Background())
	if err != nil || len(wg) != 1 || wg[0].WorkgroupUUID != "w1" {
		t.Fatalf("%v %v", wg, err)
	}
	pr, err := ext.GetPriorities(context.Background())
	if err != nil || pr[0].PriorityCode.String() != "1" {
		t.Fatalf("%v %v", pr, err)
	}
	em, err := ext.GetEmployers(context.Background(), "wg1")
	if err != nil || em[0].Employer != "Админ" {
		t.Fatalf("%v %v", em, err)
	}
}

func TestExternalHTTPError(t *testing.T) {
	c, _ := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "нет публикации", http.StatusNotFound)
	})
	_, err := c.External().GetClients(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 404 {
		t.Fatalf("%v", err)
	}
	if !errors.Is(err, ErrBadStatus) {
		t.Fatalf("Is ErrBadStatus: %v", err)
	}
}

func TestPortalAPI(t *testing.T) {
	c, _ := startTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requireBasic(t, r)
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/authenticate") && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(path, "/getCI"):
			w.Write([]byte(`[{"UID":"ci1","Name":"ПК-1","Cod":"1","Classifier":{"UID":"c","Name":"ПК"},"Services":[{"UID":"s","Name":"ИТ"}]}]`))
		case strings.HasSuffix(path, "/getListIncidentsOpen"):
			w.Write([]byte(`[{"UID":"i1","Number":"1","Topic":"не работает","Status":{"UID":"s","Name":"Открыто","IsClosed":false}}]`))
		case strings.HasSuffix(path, "/addNewIncident") && r.Method == http.MethodPost:
			var in PortalNewIncident
			json.NewDecoder(r.Body).Decode(&in)
			if in.Topic == "" || in.MembershipServices == nil {
				t.Fatalf("%+v", in)
			}
			w.WriteHeader(http.StatusOK)
		case strings.Contains(path, "/getDetailInfoIncindent/"):
			w.Write([]byte(`{"UID":"i1","Number":"1","Topic":"t","TheHistoryOfCommunication":[{"Text":"hi","Data":"1","Autor":"u"}]}`))
		default:
			t.Fatalf("%s %s", r.Method, path)
		}
	})
	p := c.Portal()
	if err := p.Authenticate(context.Background()); err != nil {
		t.Fatal(err)
	}
	ci, err := p.GetCI(context.Background())
	if err != nil || len(ci) != 1 || ci[0].Name != "ПК-1" {
		t.Fatalf("%v %v", ci, err)
	}
	open, err := p.GetListIncidentsOpen(context.Background())
	if err != nil || open[0].UID != "i1" || open[0].Status.IsClosed.Bool() {
		t.Fatalf("%v %v", open, err)
	}
	if err := p.AddNewIncident(context.Background(), PortalNewIncident{
		Topic: "x", Data: "01.01.2026 12:00:00", Description: "d",
		MembershipServices: &PortalMembershipServices{UID: "ms1"},
	}); err != nil {
		t.Fatal(err)
	}
	d, err := p.GetDetailInfoIncident(context.Background(), "i1")
	if err != nil || len(d.TheHistoryOfCommunication) != 1 {
		t.Fatalf("%v %v", d, err)
	}
}

func TestCustomPrefixes(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.Write([]byte(`{"States":[]}`))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(Config{
		BaseURL: srv.URL + "/pub", Login: "u", Password: "p",
		ExternalPrefix: "/hs/my-external", HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.External().GetStates(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got != "/pub/hs/my-external/getStates" {
		t.Fatalf("путь: %s", got)
	}
}
