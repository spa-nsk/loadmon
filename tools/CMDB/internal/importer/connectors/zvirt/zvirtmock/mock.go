// Package zvirtmock — mock-сервер oVirt REST API для тестов и стенда (plan.md E5.6): отдаёт
// ответы в форме документации oVirt v4 из изменяемого в тесте состояния. Не проверялся на
// живом zVirt — см. harness/09-source-zvirt-ovirt.md.
package zvirtmock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Server — состояние и обработчик mock-менеджера.
type Server struct {
	mu       sync.Mutex
	User     string
	Password string
	Version  string
	Fail     bool // все запросы данных (кроме корня) отвечают HTTP 500 — сбой источника посреди снимка
	Clusters []Cluster
	Hosts    []Host
	VMs      []VM
	Domains  []Domain
}

type Cluster struct{ ID, Name string }
type Host struct{ ID, Name, ClusterID, Serial, Status, UUID string }
type VM struct {
	ID, Name, ClusterID, HostID string
	MemoryBytes                 int64
	Cores                       int
}
type Domain struct {
	ID, Name, Type, Storage string
	Available, Used         int64
}

// Default — небольшой стенд: 1 кластер, 2 хоста, 3 ВМ (одна выключена), 1 data-домен и 1 iso-домен.
// tag (10 hex-символов) делает идентификаторы и имена уникальными: тесты делят одну БД, а КЕ
// прошлых прогонов остаются в ней.
func Default(tag string) *Server {
	const g = int64(1) << 30
	id := func(kind byte, n int) string { return fmt.Sprintf("%c1000000-0000-0000-0000-%s%02d", kind, tag, n) }
	return &Server{
		User: "admin@internal", Password: "secret", Version: "4.5.5.1",
		Clusters: []Cluster{{id('c', 1), "Cluster-A-" + tag}},
		Hosts: []Host{
			{id('a', 1), "zv-host-01-" + tag, id('c', 1), "SN-H1-" + tag, "up", "3" + tag[1:8] + "-1111-4000-8000-" + tag + "01"},
			{id('a', 2), "zv-host-02-" + tag, id('c', 1), "SN-H2-" + tag, "up", "3" + tag[1:8] + "-1111-4000-8000-" + tag + "02"},
		},
		VMs: []VM{
			{id('b', 1), "app-01-" + tag, id('c', 1), id('a', 1), 8 * g, 4},
			{id('b', 2), "db-01-" + tag, id('c', 1), id('a', 2), 32 * g, 8},
			{id('b', 3), "test-off-" + tag, id('c', 1), "", 2 * g, 2},
		},
		Domains: []Domain{
			{id('d', 1), "data-nfs-" + tag, "data", "nfs", 400 * g, 100 * g},
			{id('d', 2), "iso-domain-" + tag, "iso", "nfs", 10 * g, 1 * g},
		},
	}
}

// Do выполняет f под блокировкой состояния (изменения из теста во время работы сервера).
func (s *Server) Do(f func(*Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

func str(n int64) string { return strconv.FormatInt(n, 10) }

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, p, ok := r.BasicAuth(); !ok || u != s.User || p != s.Password {
		http.Error(w, `{"detail":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/ovirt-engine/api")
	if s.Fail && path != "" {
		http.Error(w, `{"detail":"internal error"}`, http.StatusInternalServerError)
		return
	}

	var out any
	switch path {
	case "":
		out = map[string]any{"product_info": map[string]any{"name": "oVirt Engine", "version": map[string]any{"full_version": s.Version}}}
	case "/clusters":
		list := []any{}
		for _, c := range s.Clusters {
			list = append(list, map[string]any{"id": c.ID, "name": c.Name, "description": "", "ha_reservation": false})
		}
		out = map[string]any{"cluster": list}
	case "/hosts":
		list := []any{}
		for _, h := range s.Hosts {
			list = append(list, map[string]any{"id": h.ID, "name": h.Name, "status": h.Status,
				"cluster": map[string]any{"id": h.ClusterID}, "version": map[string]any{"full_version": "vdsm-4.50"},
				"hardware_information": map[string]any{"serial_number": h.Serial, "uuid": strings.ToUpper(h.UUID)}})
		}
		out = map[string]any{"host": list}
	case "/vms":
		list := []any{}
		for _, v := range s.VMs {
			m := map[string]any{"id": v.ID, "name": v.Name, "cluster": map[string]any{"id": v.ClusterID},
				"memory": str(v.MemoryBytes), // oVirt отдаёт int64 строкой
				"cpu":    map[string]any{"topology": map[string]any{"cores": str(int64(v.Cores)), "sockets": "1", "threads": "1"}}}
			if v.HostID != "" {
				m["host"] = map[string]any{"id": v.HostID}
			}
			list = append(list, m)
		}
		out = map[string]any{"vm": list}
	case "/storagedomains":
		list := []any{}
		for _, d := range s.Domains {
			list = append(list, map[string]any{"id": d.ID, "name": d.Name, "type": d.Type, "available": str(d.Available),
				"used": str(d.Used), "storage": map[string]any{"type": d.Storage}})
		}
		out = map[string]any{"storage_domain": list}
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
