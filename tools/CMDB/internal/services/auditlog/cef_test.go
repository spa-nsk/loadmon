package auditlog

import (
	"strings"
	"testing"
)

func TestFormatCEF_BasicFields(t *testing.T) {
	r := Row{ID: "1", At: "2026-09-27 05:41:50", EventType: "login_failed", UserID: "0", Username: "",
		ClientIP: "203.0.113.5", Details: `{"filter": "a=b"}`}
	s := FormatCEF(r)

	if !strings.Contains(s, "CEF:0|CMDB|cmdb-web|1.0|login_failed|login_failed|6|") {
		t.Errorf("нет заголовка CEF с серьёзностью 6 (login_failed): %s", s)
	}
	if !strings.Contains(s, "rt=Sep 27 05:41:50 UTC 2026") {
		t.Errorf("нет rt= с ожидаемой меткой времени: %s", s)
	}
	if !strings.Contains(s, "suser=system") {
		t.Errorf("suser должен быть system при UserID=0 и пустом Username: %s", s)
	}
	if !strings.Contains(s, "src=203.0.113.5") {
		t.Errorf("нет src=203.0.113.5: %s", s)
	}
	if !strings.Contains(s, `cs1={"filter": "a\=b"}`) {
		t.Errorf("cs1 не экранировал '=' внутри details: %s", s)
	}
}

func TestFormatCEF_UsernamePreferredOverUserID(t *testing.T) {
	r := Row{At: "2026-01-01 00:00:00", EventType: "login", UserID: "42", Username: "ivanov", ClientIP: "", Details: "{}"}
	s := FormatCEF(r)
	if !strings.Contains(s, "suser=ivanov") {
		t.Errorf("suser должен быть username, а не user_id: %s", s)
	}
	if !strings.Contains(s, "src=0.0.0.0") {
		t.Errorf("пустой ClientIP должен стать 0.0.0.0: %s", s)
	}
}

func TestFormatCEF_EscapesPipeInEventType(t *testing.T) {
	r := Row{At: "2026-01-01 00:00:00", EventType: "weird|type", Details: "{}"}
	s := FormatCEF(r)
	if !strings.Contains(s, `weird\|type`) {
		t.Errorf("'|' в event_type должен быть экранирован: %s", s)
	}
}

func TestCefSeverity_KnownPrefixes(t *testing.T) {
	cases := map[string]int{"login": 1, "login_failed": 6, "login_locked": 8, "role_grant": 5, "ref_delete": 5, "unknown_event": 3}
	for eventType, want := range cases {
		if got := cefSeverity(eventType); got != want {
			t.Errorf("cefSeverity(%q) = %d, want %d", eventType, got, want)
		}
	}
}
