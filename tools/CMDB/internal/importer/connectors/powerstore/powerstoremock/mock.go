// Package powerstoremock — mock Dell PowerStore REST API для тестов (plan.md E8.1): HTTP Basic на
// каждый запрос, коллекции — бо́льшие JSON-массивы верхнего уровня (без обёртки, в отличие от
// Unity/OME), фильтр field=op.value (например type=eq.Primary). Форма ответов — из общедоступной
// документации, не проверена на живом массиве (см. комментарий в ../powerstore.go).
package powerstoremock

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"cmdb/internal/importer/connectors/storage"
)

type obj = map[string]any

// Server — состояние и обработчик mock-кластера.
type Server struct {
	mu       sync.Mutex
	User     string
	Password string
	Fail     bool // коллекция volume отвечает HTTP 500

	Data map[string][]obj // ресурс -> строки
}

// Default — кластер: 1 appliance, 2 тома (+1 снимок, должен быть отфильтрован type=eq.Primary),
// 2 диска, презентация одного тома одному инициатору. tag делает серийные номера/имена/WWN
// уникальными в общей тестовой БД.
func Default(tag string) *Server {
	return &Server{
		User: "admin", Password: "secret",
		Data: map[string][]obj{
			"cluster":   {{"id": "0", "name": "ps-" + tag}},
			"appliance": {{"id": "A1", "name": "Appliance-1", "service_tag": "PSSN" + tag, "model": "PowerStore 1000T"}},
			"volume": {
				{"id": "vol1", "name": "vol-a-" + tag, "size": 100e9, "wwn": "naa.68ccf098" + tag + "1", "is_thin_enabled": true, "type": "Primary"},
				{"id": "vol2", "name": "vol-b-" + tag, "size": 200e9, "wwn": "naa.68ccf098" + tag + "2", "is_thin_enabled": true, "type": "Primary"},
				{"id": "vol1-snap", "name": "vol-a-" + tag + "-snap", "size": 100e9, "wwn": "naa.68ccf098" + tag + "9", "is_thin_enabled": true, "type": "Snapshot"},
			},
			"hardware": {
				{"id": "drive1", "name": "Drive_1", "slot": "0", "serial_number": "PD0" + tag, "manufacturer": "Samsung", "part_number": "PM1653", "drive_type": "SAS_FLASH", "size": 1.92e12, "type": "Drive"},
				{"id": "drive2", "name": "Drive_2", "slot": "1", "serial_number": "PD1" + tag, "manufacturer": "Samsung", "part_number": "PM1653", "drive_type": "NVMe", "size": 3.84e12, "type": "Drive"},
			},
			"host_initiator":      {{"id": "hi1", "host_id": "host1", "port_name": storage.SampleWWPN(tag, 1)}},
			"host_volume_mapping": {{"id": "hvm1", "host_id": "host1", "volume_id": "vol1"}},
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

	if u, p, ok := r.BasicAuth(); !ok || u != s.User || p != s.Password {
		w.Header().Set("WWW-Authenticate", `Basic realm="powerstore"`)
		writeJSON(w, http.StatusUnauthorized, obj{"message": "Unauthorized"})
		return
	}

	const prefix = "/api/rest/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	res := strings.TrimPrefix(r.URL.Path, prefix)

	if s.Fail && res == "volume" {
		writeJSON(w, http.StatusInternalServerError, obj{"message": "internal error"})
		return
	}

	rows, ok := s.Data[res]
	if !ok {
		http.NotFound(w, r)
		return
	}

	filters := map[string]string{}
	for k, v := range r.URL.Query() {
		if k == "select" || len(v) == 0 {
			continue
		}
		filters[k] = v[0]
	}

	fields := strings.Split(r.URL.Query().Get("select"), ",")

	var out []obj
	for _, row := range rows {
		if !matchesAll(row, filters) {
			continue
		}
		item := obj{}
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if v, ok := row[f]; ok {
				item[f] = v
			}
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

// matchesAll — фильтр PowerStore field=op.value; поддержан только eq (единственный, которым
// пользуется коннектор).
func matchesAll(row obj, filters map[string]string) bool {
	for k, v := range filters {
		want, ok := strings.CutPrefix(v, "eq.")
		if !ok {
			continue
		}
		got, _ := row[k].(string)
		if got != want {
			return false
		}
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
