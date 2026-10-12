// Package zabbixmock — mock-сервер Zabbix JSON-RPC 7.0 для тестов и стенда (plan.md E5.6).
// Ошибки отдаются как в Zabbix: HTTP 200 и объект error в теле. Форма ответов из документации,
// не проверена на живом сервере — harness/11-source-zabbix.md.
package zabbixmock

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
)

// Server — состояние и обработчик mock-сервера.
type Server struct {
	mu       sync.Mutex
	Version  string
	Token    string // API-токен, принимаемый как Bearer
	User     string
	Password string
	Fail     bool // host.get отвечает ошибкой JSON-RPC
	Hosts    []Host
}

type Host struct {
	ID, Host, Name string
	Enabled        bool
	IPs            []string
	Groups         []string
	Templates      []string
	Serial         string
}

// Default — стенд из трёх хостов; имена уникализируются tag: FQDN-хост «a», хост «b» и хост без КЕ.
func Default(tag string) *Server {
	return &Server{
		Version: "7.0.5", Token: "zbx-token", User: "Admin", Password: "zabbix",
		Hosts: []Host{
			{ID: "10001", Host: "zbx-a-" + tag + ".corp.local", Name: "Сервер A " + tag, Enabled: true, IPs: []string{"10.1.0.1"},
				Groups: []string{"Linux servers"}, Templates: []string{"Linux by Zabbix agent"}, Serial: "SN-ZBX-A-" + tag},
			{ID: "10002", Host: "zbx-b-" + tag, Name: "zbx-b-" + tag, Enabled: true, IPs: []string{"10.1.0.2", "10.1.0.22"},
				Groups: []string{"Linux servers", "DB"}, Templates: []string{"Linux by Zabbix agent", "PostgreSQL by Zabbix agent"}, Serial: "SN-ZBX-B-" + tag},
			{ID: "10003", Host: "unknown-" + tag, Name: "unknown-" + tag, Enabled: true, IPs: []string{"10.1.0.3"}},
		},
	}
}

// Do выполняет f под блокировкой состояния.
func (s *Server) Do(f func(*Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

type request struct {
	Method string `json:"method"`
	Params any    `json:"params"` // объект, либо [] у методов без параметров
	ID     any    `json:"id"`
}

func reply(w http.ResponseWriter, id any, result any, errMsg, errData string) {
	out := map[string]any{"jsonrpc": "2.0", "id": id}
	if errMsg != "" {
		out["error"] = map[string]any{"code": -32500, "message": errMsg, "data": errData}
	} else {
		out["result"] = result
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.URL.Path != "/api_jsonrpc.php" {
		http.NotFound(w, r)
		return
	}
	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reply(w, nil, nil, "Parse error", err.Error())
		return
	}

	switch req.Method {
	case "apiinfo.version": // без авторизации
		reply(w, req.ID, s.Version, "", "")
		return
	case "user.login":
		params, _ := req.Params.(map[string]any)
		if params["username"] != s.User || params["password"] != s.Password {
			reply(w, req.ID, nil, "Incorrect user name or password or account is temporarily blocked.", "Login name or password is incorrect.")
			return
		}
		reply(w, req.ID, "session-token", "", "")
		return
	}

	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if bearer != s.Token && bearer != "session-token" {
		reply(w, req.ID, nil, "Not authorized.", "Session terminated, re-login, please.")
		return
	}
	if req.Method != "host.get" {
		reply(w, req.ID, nil, "Incorrect method.", "Method "+req.Method+" is not supported.")
		return
	}
	if s.Fail {
		reply(w, req.ID, nil, "Application error.", "SQL statement execution has failed.")
		return
	}

	list := []any{}
	for _, h := range s.Hosts {
		status := "1"
		if h.Enabled {
			status = "0"
		}
		ifaces := []any{}
		for i, ip := range h.IPs {
			main := "0"
			if i == 0 {
				main = "1"
			}
			ifaces = append(ifaces, map[string]any{"ip": ip, "main": main})
		}
		names := func(in []string) []any {
			out := []any{}
			for _, n := range in {
				out = append(out, map[string]any{"name": n})
			}
			return out
		}
		m := map[string]any{"hostid": h.ID, "host": h.Host, "name": h.Name, "status": status, "interfaces": ifaces,
			"hostgroups": names(h.Groups), "parentTemplates": names(h.Templates), "inventory": []any{}}
		if h.Serial != "" {
			m["inventory"] = map[string]any{"serial_no_a": h.Serial}
		}
		list = append(list, m)
	}
	reply(w, req.ID, list, "", "")
}
