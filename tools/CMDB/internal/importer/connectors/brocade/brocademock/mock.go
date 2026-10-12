// Package brocademock — mock Brocade Fabric OS REST API для тестов (plan.md E8.4): сессия
// (POST /rest/login с Basic, заголовок Authorization), конверт {"Response": {...}}. Форма ответов
// — из общедоступной документации, не проверена на живом коммутаторе (см. комментарий в
// ../brocade.go).
package brocademock

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
)

type obj = map[string]any

// Port — один FC-порт коммутатора.
type Port struct {
	Name, WWPN, PhysicalState string
}

// Zone — одна зона активной (effective) конфигурации.
type Zone struct {
	Name    string
	Members []string // WWPN
}

// Server — состояние и обработчик mock-коммутатора.
type Server struct {
	mu       sync.Mutex
	User     string
	Password string
	Fail     bool

	Name, WWN, Model, Firmware string
	DomainID                   int
	Ports                      []Port
	CfgName                    string
	Zones                      []Zone

	Logins, Logouts int
	session         string
}

// Default — коммутатор с 2 портами (1 со сброшенным WWPN — недостоверный, пропускается) и 1 активной
// зоной с 2 участниками. tag делает WWN/имена уникальными в общей тестовой БД.
func Default(tag string) *Server {
	return &Server{
		User: "admin", Password: "password", Name: "sw-" + tag, WWN: "10:00:00:05:1e:" + tag[:2] + ":" + tag[2:4] + ":" + tag[4:6],
		Model: "Brocade G720", Firmware: "9.2.0", DomainID: 5,
		Ports: []Port{
			{Name: "0/1", WWPN: "20:01:00:05:1e:" + tag[:2] + ":" + tag[2:4] + ":01", PhysicalState: "online"},
			{Name: "0/2", WWPN: "", PhysicalState: "offline"},
		},
		CfgName: "prod_cfg",
		Zones: []Zone{
			{Name: "zone_" + tag, Members: []string{"20:01:00:05:1e:" + tag[:2] + ":" + tag[2:4] + ":01", "50:0a:09:81:88:" + tag[:2] + ":" + tag[2:4] + ":01"}},
		},
	}
}

// Do выполняет f под блокировкой состояния.
func (s *Server) Do(f func(*Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.URL.Path {
	case "/rest/login":
		u, p, ok := r.BasicAuth()
		if !ok || u != s.User || p != s.Password {
			writeJSON(w, http.StatusUnauthorized, obj{"errors": obj{"error": []any{obj{"error-message": "Invalid credentials"}}}})
			return
		}
		s.Logins++
		s.session = "AUTH_" + s.Name + "_" + http.MethodPost
		w.Header().Set("Authorization", s.session)
		writeJSON(w, http.StatusOK, obj{})
		return
	case "/rest/logout":
		s.Logouts++
		s.session = ""
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if s.session == "" || r.Header.Get("Authorization") != s.session {
		writeJSON(w, http.StatusUnauthorized, obj{"errors": obj{"error": []any{obj{"error-message": "Not authenticated"}}}})
		return
	}

	switch r.URL.Path {
	case "/rest/running/brocade-fibrechannel-switch/fibrechannel-switch":
		writeJSON(w, http.StatusOK, obj{"Response": obj{"fibrechannel-switch": []any{obj{
			"name": s.Name, "wwn": s.WWN, "model": s.Model, "firmware-version": s.Firmware, "domain-id": strconv.Itoa(s.DomainID),
		}}}})
	case "/rest/running/brocade-interface/fibrechannel":
		if s.Fail {
			writeJSON(w, http.StatusInternalServerError, obj{"errors": obj{"error": []any{obj{"error-message": "internal error"}}}})
			return
		}
		var ports []any
		for _, p := range s.Ports {
			ports = append(ports, obj{"name": p.Name, "wwn": p.WWPN, "physical-state": p.PhysicalState})
		}
		writeJSON(w, http.StatusOK, obj{"Response": obj{"fibrechannel": ports}})
	case "/rest/running/brocade-zone/effective-configuration":
		var zones []any
		for _, z := range s.Zones {
			var members []any
			for _, m := range z.Members {
				members = append(members, m)
			}
			zones = append(zones, obj{"zone-name": z.Name, "member-entry": obj{"entry-name": members}})
		}
		writeJSON(w, http.StatusOK, obj{"Response": obj{"effective-configuration": obj{"cfg-name": s.CfgName, "zone": zones}}})
	default:
		http.NotFound(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/yang-data+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
