package itilium

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthUserAndNTLM(t *testing.T) {
	c, err := NewClient(Config{
		BaseURL: "https://example.local/ITILIUM", Login: "itiluser", Domain: "itildomail",
		Password: "x", AuthMode: AuthNTLM, HTTPClient: http.DefaultClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.authUser(); got != `itildomail\itiluser` {
		t.Fatalf("authUser: %q", got)
	}
}

func TestLabelValueUnmarshal(t *testing.T) {
	var lv LabelValue
	if err := json.Unmarshal([]byte(`{"label":"В работе","value":"1"}`), &lv); err != nil {
		t.Fatal(err)
	}
	if lv.Label != "В работе" || lv.ValueString() != "1" || lv.String() != "1" {
		t.Fatalf("%+v", lv)
	}
	if err := json.Unmarshal([]byte(`"srv01"`), &lv); err != nil {
		t.Fatal(err)
	}
	if lv.ValueString() != "srv01" || lv.String() != "srv01" {
		t.Fatalf("%+v", lv)
	}
	if err := json.Unmarshal([]byte(`{"label":"n","value":42}`), &lv); err != nil {
		t.Fatal(err)
	}
	if lv.ValueString() != "42" || lv.String() != "42" {
		t.Fatalf("числовое value: %+v", lv)
	}
}

func TestLinksUnitFieldUnmarshal(t *testing.T) {
	var f LinksUnitField
	if err := json.Unmarshal([]byte(`"blade-01"`), &f); err != nil {
		t.Fatal(err)
	}
	if f.String() != "blade-01" || len(f) != 1 {
		t.Fatalf("строка: %+v", f)
	}
	if err := json.Unmarshal([]byte(`{"label":"Unit","value":"U10"}`), &f); err != nil {
		t.Fatal(err)
	}
	if f.String() != "U10" || f[0].Label != "Unit" {
		t.Fatalf("объект: %+v", f)
	}
	if err := json.Unmarshal([]byte(`[{"label":"a","value":"1"},{"label":"b","value":"2"}]`), &f); err != nil {
		t.Fatal(err)
	}
	if f.String() != "1; 2" || len(f) != 2 {
		t.Fatalf("массив: %+v / %q", f, f.String())
	}
	if err := json.Unmarshal([]byte(`null`), &f); err != nil || f != nil {
		t.Fatalf("null: %+v %v", f, err)
	}
}

func TestCUClassCatalog(t *testing.T) {
	if CUClassName(CUClassServers) != "IT_Серверы" || CUClassName(CUClassStorage) != "IT_СХД" {
		t.Fatalf("%q %q", CUClassName(CUClassServers), CUClassName(CUClassStorage))
	}
	list := CUClasses()
	if len(list) != len(cuClassOrder) || list[0].Code != CUClassIPHeadsets {
		t.Fatalf("%d %+v", len(list), list[0])
	}
	if CUClassName(99999) != "" {
		t.Fatal("unknown")
	}
}

func TestGetCUParseItilliumCu(t *testing.T) {
	const sample = `{
  "rows": [{
    "rowKey": "rk1",
    "uid": "uid-1",
    "deleted": false,
    "status": {"label": "В работе", "value": "1"},
    "code": {"label": "IT-0001", "value": "IT-0001"},
    "sortField": "IT-0001",
    "name": {"label": "srv01", "value": "srv01"},
    "description": {"label": "", "value": ""},
    "author": {"label": "Иванов", "value": "user-1"},
    "cuClass": "145",
    "linksUnit": "blade-01",
    "hostname": {"label": "FQDN", "value": "srv01.example.local"},
    "ip": "10.0.0.1",
    "modelServ": {"label": "Модель", "value": "DL380"},
    "sn": {"label": "S/N", "value": "CZ1234"},
    "user": {"label": "Владелец", "value": "petrov"},
    "where": {"label": "Место", "value": "ЦОД-1"},
    "statusHistory": "2024:в работе"
  }],
  "totalCount": 100,
  "last": "50",
  "hasMore": true
}`
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sample))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{
		Name: "t", BaseURL: srv.URL + "/ITILIUM", Login: "itiluser", Domain: "itildomail",
		Password: "secret", AuthMode: AuthBasic, HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	q := GetCUQuery{
		CUClass: CUClassServers, CUStatus: CUStatusAll,
		Fields: DefaultServerCUFields,
	}
	u := c.API().GetCUURL(q)
	if !strings.Contains(u, "/ITILIUM/hs/api/getCU?") || !strings.Contains(u, "cuClass=145") || !strings.Contains(u, "cuStatus=all") {
		t.Fatalf("URL: %s", u)
	}

	resp, raw, err := c.API().GetCURaw(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/ITILIUM/hs/api/getCU" {
		t.Fatalf("path: %s", gotPath)
	}
	for _, part := range []string{"cuClass=145", "cuStatus=all", "fields=code"} {
		if !strings.Contains(gotQuery, part) {
			t.Fatalf("query без %s: %s", part, gotQuery)
		}
	}
	for _, need := range []string{"modelVendS", "linksUnit", "cpu", "os", "ram_1"} {
		if !strings.Contains(gotQuery, need) {
			t.Fatalf("query без %s: %s", need, gotQuery)
		}
	}
	if !json.Valid(raw) || len(resp.Rows) != 1 {
		t.Fatalf("%+v", resp)
	}
	row := resp.Rows[0]
	if row.UID != "uid-1" || row.Code.String() != "IT-0001" || row.Status.Label != "В работе" ||
		row.Hostname.String() != "srv01.example.local" || row.IP.String() != "10.0.0.1" ||
		row.SN.String() != "CZ1234" || row.CuClass != "145" || row.LinksUnit.String() != "blade-01" {
		t.Fatalf("%+v", row)
	}
	next, ok := resp.NextPageQuery(q)
	if !ok || next.NextVer != "50" {
		t.Fatalf("next: %+v ok=%v", next, ok)
	}
}

func TestGetCULinksUnitObjectAndArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"rows":[
			{"uid":"1","linksUnit":{"label":"Unit","value":"U10"}},
			{"uid":"2","linksUnit":[{"label":"a","value":"1"},{"value":"2"}]}
		]}`))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(Config{
		BaseURL: srv.URL, Login: "u", Password: "p", AuthMode: AuthBasic, HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.API().GetCU(context.Background(), GetCUQuery{CUClass: 145})
	if err != nil || len(resp.Rows) != 2 {
		t.Fatalf("%v %v", resp, err)
	}
	if resp.Rows[0].LinksUnit.String() != "U10" {
		t.Fatalf("object: %q", resp.Rows[0].LinksUnit)
	}
	if resp.Rows[1].LinksUnit.String() != "1; 2" {
		t.Fatalf("array: %q", resp.Rows[1].LinksUnit)
	}
}

func TestGetCUIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"isError":true,"rows":[]}`))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(Config{
		BaseURL: srv.URL, Login: "u", Password: "p", AuthMode: AuthBasic, HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.API().GetCU(context.Background(), GetCUQuery{CUClass: 145})
	if err == nil || resp == nil || !resp.IsError {
		t.Fatalf("%v %v", resp, err)
	}
}
