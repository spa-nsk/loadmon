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
		BaseURL: "https://example.local/DEV", Login: "itiluser", Domain: "itildomail",
		Password: "x", AuthMode: AuthNTLM, HTTPClient: http.DefaultClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := c.authUser(); got != `itildomail\itiluser` {
		t.Fatalf("authUser: %q", got)
	}
	if c.cfg.AuthMode != AuthNTLM {
		t.Fatalf("AuthMode: %s", c.cfg.AuthMode)
	}
}

func TestGetCUQueryAndParse(t *testing.T) {
	var gotPath, gotQuery, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodGet {
			t.Fatalf("method %s", r.Method)
		}
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nextVer":"2","data":[{"code":"IT-1","hostname":"srv01","ip":"10.0.0.1","sn":"SN1","status":"в работе","extraField":123}]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient(Config{
		Name: "t", BaseURL: srv.URL + "/DEV", Login: "itiluser", Domain: "itildomail",
		Password: "secret", AuthMode: AuthBasic, HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	q := GetCUQuery{
		NextVer: 1, CUClass: 145, Limit: 50,
		Fields: DefaultServerCUFields,
	}
	wantURL := c.API().GetCUURL(q)
	if !strings.Contains(wantURL, "/DEV/hs/api/getCU?") || !strings.Contains(wantURL, "cuClass=145") {
		t.Fatalf("URL: %s", wantURL)
	}

	resp, raw, err := c.API().GetCU(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/DEV/hs/api/getCU" {
		t.Fatalf("path: %s", gotPath)
	}
	for _, part := range []string{"nextVer=1", "cuClass=145", "limit=50", "fields=code"} {
		if !strings.Contains(gotQuery, part) {
			t.Fatalf("query без %s: %s", part, gotQuery)
		}
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("Authorization: %q", gotAuth)
	}
	units := resp.Units()
	if len(units) != 1 || units[0].Hostname.String() != "srv01" || units[0].Code.String() != "IT-1" {
		t.Fatalf("%+v", units)
	}
	if string(units[0].Extra["extraField"]) != "123" {
		t.Fatalf("Extra: %v", units[0].Extra)
	}
	if !json.Valid(raw) || resp.NextVer.String() != "2" {
		t.Fatalf("raw/nextVer: %s %q", raw, resp.NextVer)
	}
}

func TestGetCUArrayResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"code":"A","hostname":"h"}]`))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(Config{
		BaseURL: srv.URL, Login: "u", Password: "p", AuthMode: AuthBasic, HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, _, err := c.API().GetCU(context.Background(), GetCUQuery{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Units()) != 1 || resp.Units()[0].Code.String() != "A" {
		t.Fatalf("%+v", resp)
	}
}
