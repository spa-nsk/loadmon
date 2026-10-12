// Package unitymock — mock Dell EMC Unity REST (Unisphere) для тестов и стенда (plan.md E5.6):
// вход с CSRF-токеном и cookie, X-EMC-REST-CLIENT, постраничные коллекции (links rel=next), выдача
// только запрошенных fields=. Форма ответов из документации и наблюдений loadmon, не проверена на
// живом массиве — harness/14-source-storage.md.
package unitymock

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"cmdb/internal/importer/connectors/storage"
)

type obj = map[string]any

// Server — состояние и обработчик mock-массива. Данные хранятся в виде готовых content-объектов.
type Server struct {
	mu       sync.Mutex
	User     string
	Password string
	PerPage  int  // записей на страницу (меньше запрошенного per_page — проверка постраничности)
	Fail     bool // коллекции lun отвечают HTTP 500

	Logins  int
	Logouts int
	session string

	Data map[string][]obj // тип -> content-объекты
}

// Default — массив: 2 пула, 3 LUN, NAS-сервер, файловая система с SMB и NFS, 2 SP, DPE и DAE, 3 диска.
// tag (10 hex) делает серийные номера, имена и WWN уникальными в общей тестовой БД.
func Default(tag string) *Server {
	const tb = 1e12
	wwn := func(n int) string {
		return "60:06:01:60:" + tag[0:2] + ":" + tag[2:4] + ":" + tag[4:6] + ":" + tag[6:8] + ":00:00:00:0" + strconv.Itoa(n)
	}
	return &Server{
		User: "admin", Password: "secret", PerPage: 2,
		Data: map[string][]obj{
			"system":          {{"name": "unity-" + tag, "model": "Unity 480F", "serialNumber": "UNSN" + tag}},
			"basicSystemInfo": {{"softwareVersion": "5.3.0.0.5.120"}},
			"pool":            {{"id": "pool_1", "name": "Pool 1", "sizeTotal": 40 * tb}, {"id": "pool_2", "name": "Pool 2", "sizeTotal": 20 * tb}},
			"lun": {
				{"id": "sv_1", "name": "lun-a-" + tag, "sizeTotal": 100e9, "isThinEnabled": true, "wwn": wwn(1), "pool": obj{"id": "pool_1"}},
				{"id": "sv_2", "name": "lun-b-" + tag, "sizeTotal": 200e9, "isThinEnabled": false, "wwn": wwn(2), "pool": obj{"id": "pool_1"}},
				{"id": "sv_3", "name": "lun-c-" + tag, "sizeTotal": 300e9, "isThinEnabled": true, "wwn": wwn(3), "pool": obj{"id": "pool_2"}},
			},
			"hostInitiator": {
				{"id": "HostInitiator_1", "type": 1, "initiatorId": "20:00:00:00:00:00:00:01:" + storage.SampleWWPN(tag, 3), "parentHost": obj{"id": "Host_1"}},
				{"id": "HostInitiator_2", "type": 2, "initiatorId": "iqn.1998-01.com.vmware:esx-" + tag, "parentHost": obj{"id": "Host_1"}},
			},
			"hostLUN":    {{"id": "Host_1_sv_1_prod", "host": obj{"id": "Host_1"}, "lun": obj{"id": "sv_1"}}},
			"nasServer":  {{"id": "nas_1", "name": "nas-" + tag}},
			"fileSystem": {{"id": "fs_1", "name": "fs-" + tag, "sizeTotal": 500e9, "isThinEnabled": true, "nasServer": obj{"id": "nas_1"}, "pool": obj{"id": "pool_2"}}},
			"cifsShare":  {{"id": "SMBShare_1", "name": "share-" + tag, "path": "/", "filesystem": obj{"id": "fs_1"}}},
			"nfsShare":   {{"id": "NFSShare_1", "name": "export-" + tag, "path": "/", "filesystem": obj{"id": "fs_1"}}},
			"storageProcessor": {
				{"id": "spa", "name": "SP A", "model": "OBERON", "emcSerialNumber": "SPA" + tag},
				{"id": "spb", "name": "SP B", "model": "OBERON", "emcSerialNumber": "SPB" + tag},
			},
			"dpe": {{"id": "dpe", "name": "DPE", "model": "DPE 25 x 2.5", "emcSerialNumber": "DPE" + tag}},
			"dae": {{"id": "dae_1_1", "name": "DAE 1 1", "model": "DAE 25 x 2.5", "emcSerialNumber": "DAE" + tag}},
			"disk": {
				{"id": "dpe_disk_0", "name": "DPE Drive 0", "rawSize": 1900e9, "diskTechnology": "SAS_FLASH_2", "manufacturer": "SAMSUNG", "model": "PM1633", "emcSerialNumber": "UD0" + tag, "slotNumber": 0},
				{"id": "dpe_disk_1", "name": "DPE Drive 1", "rawSize": 1900e9, "diskTechnology": "SAS_FLASH_2", "manufacturer": "SAMSUNG", "model": "PM1633", "emcSerialNumber": "UD1" + tag, "slotNumber": 1},
				{"id": "dae_1_1_disk_0", "name": "DAE 1 1 Drive 0", "rawSize": 4000e9, "diskTechnology": "NL_SAS", "manufacturer": "SEAGATE", "model": "ST4000", "emcSerialNumber": "UD2" + tag, "slotNumber": 0},
			},
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

	if r.Header.Get("X-EMC-REST-CLIENT") != "true" {
		writeJSON(w, http.StatusForbidden, obj{"error": obj{"httpStatusCode": 403, "messages": []any{obj{"en-US": "missing X-EMC-REST-CLIENT"}}}})
		return
	}
	path := r.URL.Path

	if path == "/api/types/loginSessionInfo/action/logout" {
		if r.Header.Get("EMC-CSRF-TOKEN") != s.session || s.session == "" {
			writeJSON(w, http.StatusForbidden, obj{"error": obj{"httpStatusCode": 403}})
			return
		}
		s.Logouts++
		s.session = ""
		writeJSON(w, http.StatusOK, obj{})
		return
	}

	_, hasCookie := cookieValue(r, "mod_sessionid")
	if u, p, ok := r.BasicAuth(); !hasCookie || s.session == "" {
		if !ok || u != s.User || p != s.Password {
			writeJSON(w, http.StatusUnauthorized, obj{"error": obj{"httpStatusCode": 401}})
			return
		}
		if s.session == "" {
			s.Logins++
			s.session = "csrf-" + strconv.Itoa(s.Logins)
		}
		http.SetCookie(w, &http.Cookie{Name: "mod_sessionid", Value: s.session, Path: "/"})
		w.Header().Set("EMC-CSRF-TOKEN", s.session)
	}

	const prefix, suffix = "/api/types/", "/instances"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.NotFound(w, r)
		return
	}
	typ := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if typ == "loginSessionInfo" {
		writeJSON(w, http.StatusOK, obj{"entries": []any{obj{"content": obj{"id": "1"}}}})
		return
	}
	if s.Fail && typ == "lun" {
		writeJSON(w, http.StatusInternalServerError, obj{"error": obj{"httpStatusCode": 500}})
		return
	}
	rows, ok := s.Data[typ]
	if !ok {
		writeJSON(w, http.StatusNotFound, obj{"error": obj{"httpStatusCode": 404}})
		return
	}

	// без fields= Unity отдаёт только id (проверка, что коннектор перечисляет поля)
	fields := map[string]bool{"id": true}
	for _, f := range strings.Split(r.URL.Query().Get("fields"), ",") {
		fields[strings.TrimSpace(f)] = true
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	start := (page - 1) * s.PerPage
	end := min(start+s.PerPage, len(rows))
	if start > len(rows) {
		start = len(rows)
	}
	entries := []any{}
	for _, row := range rows[start:end] {
		content := obj{}
		for k, v := range row {
			if fields[k] {
				content[k] = v
			}
		}
		entries = append(entries, obj{"@base": "/api/instances/" + typ, "content": content})
	}
	links := []any{obj{"rel": "self", "href": "&page=" + strconv.Itoa(page)}}
	if end < len(rows) {
		links = append(links, obj{"rel": "next", "href": "&page=" + strconv.Itoa(page+1)})
	}
	writeJSON(w, http.StatusOK, obj{"@base": "/api/types/" + typ + "/instances", "links": links, "entries": entries})
}

func cookieValue(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
