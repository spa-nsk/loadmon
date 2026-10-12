// Package oneviewmock — mock-сервер HPE OneView REST API 300 для тестов и стенда (plan.md E5.6):
// сессии (login-sessions/Auth), проверка X-API-Version, постраничные коллекции с nextPageUri.
// Форма ответов из документации, не проверена на живом устройстве — harness/10-source-oneview.md.
package oneviewmock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Server — состояние и обработчик mock-appliance.
type Server struct {
	mu         sync.Mutex
	User       string
	Password   string
	APIVersion string
	PageCap    int  // сервер отдаёт не больше PageCap записей на страницу (проверка nextPageUri)
	Fail       bool // /rest/server-hardware отвечает HTTP 500
	Logins     int
	Logouts    int

	Enclosures []Enclosure
	Servers    []Hardware
	Profiles   []Profile
	session    string
}

type Enclosure struct {
	ID, Name, Serial, Model string
	Bays                    int
	BaysUsed                int
	OAIP                    string
}

type Hardware struct {
	ID, Name, Serial, UUID, Model string
	EnclosureID                   string // "" — стоечный сервер
	Position                      int
	MemoryMB, CPUs, CoresPerCPU   int
	ROM, ILOIP, ILOName           string
}

type Profile struct{ Name, HardwareID string }

// Default — стенд: 1 корзина на 16 отсеков с 2 блейдами и 1 стоечный сервер; блейд 1 с профилем.
// tag (10 hex) делает серийные номера, UUID и имена уникальными в общей тестовой БД.
func Default(tag string) *Server {
	uuid := func(n int) string { return fmt.Sprintf("%08d-0000-4000-8000-%s%02d", 30373737+n, tag, n) }
	return &Server{
		User: "administrator", Password: "secret", APIVersion: "300", PageCap: 2,
		Enclosures: []Enclosure{{ID: "enc1", Name: "Enc-" + tag, Serial: "SN-ENC-" + tag, Model: "BladeSystem c7000", Bays: 16, BaysUsed: 2, OAIP: "10.0.0.5"}},
		Servers: []Hardware{
			{ID: "sh1", Name: "Enc-" + tag + ", bay 1", Serial: "SN-SRV1-" + tag, UUID: uuid(1), Model: "ProLiant BL460c Gen9", EnclosureID: "enc1", Position: 1,
				MemoryMB: 131072, CPUs: 2, CoresPerCPU: 12, ROM: "I36 v2.60", ILOIP: "10.0.1.1", ILOName: "ilo-bl1-" + tag},
			{ID: "sh2", Name: "Enc-" + tag + ", bay 2", Serial: "SN-SRV2-" + tag, UUID: uuid(2), Model: "ProLiant BL460c Gen9", EnclosureID: "enc1", Position: 2,
				MemoryMB: 65536, CPUs: 2, CoresPerCPU: 8, ROM: "I36 v2.60", ILOIP: "10.0.1.2", ILOName: "ilo-bl2-" + tag},
			{ID: "sh3", Name: "rack-dl380-" + tag, Serial: "SN-SRV3-" + tag, UUID: uuid(3), Model: "ProLiant DL380 Gen9",
				MemoryMB: 262144, CPUs: 2, CoresPerCPU: 10, ROM: "P89 v2.72", ILOIP: "10.0.1.3", ILOName: "ilo-dl3-" + tag},
		},
		Profiles: []Profile{{Name: "esx-blade-01-" + tag, HardwareID: "sh1"}},
	}
}

// Do выполняет f под блокировкой состояния.
func (s *Server) Do(f func(*Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

func uri(kind, id string) string { return "/rest/" + kind + "/" + id }

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if v := r.Header.Get("X-API-Version"); v != s.APIVersion {
		writeJSON(w, http.StatusNotAcceptable, map[string]any{"errorCode": "UNSUPPORTED_VERSION", "message": "X-API-Version " + v})
		return
	}

	if r.URL.Path == "/rest/login-sessions" {
		switch r.Method {
		case http.MethodPost:
			var in struct{ UserName, Password string }
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in.UserName != s.User || in.Password != s.Password {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"errorCode": "AUTHN_AUTH_FAIL"})
				return
			}
			s.Logins++
			s.session = fmt.Sprintf("session-%d", s.Logins)
			writeJSON(w, http.StatusOK, map[string]any{"sessionID": s.session})
		case http.MethodDelete:
			s.Logouts++
			s.session = ""
			w.WriteHeader(http.StatusNoContent)
		}
		return
	}

	if s.session == "" || r.Header.Get("Auth") != s.session {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"errorCode": "AUTHN_AUTH_FAIL"})
		return
	}
	if s.Fail && r.URL.Path == "/rest/server-hardware" {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"errorCode": "INTERNAL"})
		return
	}

	var members []any
	switch r.URL.Path {
	case "/rest/enclosures":
		for _, e := range s.Enclosures {
			bays := []any{}
			for i := 1; i <= e.Bays; i++ {
				p := "Absent"
				if i <= e.BaysUsed {
					p = "Present"
				}
				bays = append(bays, map[string]any{"bayNumber": i, "devicePresence": p})
			}
			members = append(members, map[string]any{"uri": uri("enclosures", e.ID), "name": e.Name, "serialNumber": e.Serial,
				"enclosureModel": e.Model, "deviceBayCount": e.Bays, "deviceBays": bays,
				"managerBays": []any{map[string]any{"bayNumber": 1, "ipAddress": e.OAIP}}})
		}
	case "/rest/server-hardware":
		for _, h := range s.Servers {
			m := map[string]any{"uri": uri("server-hardware", h.ID), "name": h.Name, "serialNumber": h.Serial, "uuid": strings.ToUpper(h.UUID),
				"model": h.Model, "shortModel": "BL460c", "memoryMb": h.MemoryMB, "processorCount": h.CPUs, "processorCoreCount": h.CoresPerCPU,
				"processorSpeedMhz": 4000, "romVersion": h.ROM, "mpIpAddress": h.ILOIP, "mpHostInfo": map[string]any{"mpHostName": h.ILOName},
				"powerState": "On", "status": "OK"}
			if h.EnclosureID != "" {
				m["locationUri"] = uri("enclosures", h.EnclosureID)
				m["position"] = h.Position
			}
			members = append(members, m)
		}
	case "/rest/server-profiles":
		for _, p := range s.Profiles {
			members = append(members, map[string]any{"name": p.Name, "serverHardwareUri": uri("server-hardware", p.HardwareID)})
		}
	default:
		http.NotFound(w, r)
		return
	}

	start, _ := strconv.Atoi(r.URL.Query().Get("start"))
	count, _ := strconv.Atoi(r.URL.Query().Get("count"))
	if count <= 0 || count > s.PageCap {
		count = s.PageCap
	}
	end := min(start+count, len(members))
	if start > len(members) {
		start = len(members)
	}
	out := map[string]any{"total": len(members), "count": end - start, "members": members[start:end]}
	if end < len(members) {
		out["nextPageUri"] = fmt.Sprintf("%s?start=%d&count=%d", r.URL.Path, end, count)
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
