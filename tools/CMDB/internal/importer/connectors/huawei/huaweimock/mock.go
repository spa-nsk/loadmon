// Package huaweimock — mock Huawei OceanStor DeviceManager REST для тестов и стенда (plan.md E5.6).
// Воспроизводит подтверждённые на железе особенности (harness/03): ошибки в error.code при HTTP 200,
// счётчик сессий и лимит (reach the max login sessions), Enclosure.TYPE числом, облегчённый список
// дисков и полные GET /disk/{id} с SECTORS × SECTORSIZE (520). Остальное — по документации, не
// проверено на живом массиве.
package huaweimock

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"cmdb/internal/importer/connectors/storage"
)

type obj = map[string]any

// Server — состояние и обработчик mock-массива.
type Server struct {
	mu         sync.Mutex
	User       string
	Password   string
	DeviceID   string
	MaxSession int  // лимит одновременных сессий (у железа ~32)
	Fail       bool // GET /lun -> ошибка в error.code
	PageCap    int  // записей в ответе списка (проверка range-постраничности)

	Sessions int // открытых сейчас
	Logins   int
	Logouts  int

	Name  string
	Pools []obj
	LUNs  []obj
	Ctrls []obj
	Encl  []obj
	Disks []obj // полные записи; в списке отдаются облегчённо

	Hosts  []obj
	Inits  map[string][]string // id хоста -> WWPN (ID FC-инициатора, hex без разделителей)
	Mapped map[string][]string // id хоста -> id презентованных LUN
}

// Default — массив с 2 контроллерами, полками CTE и DAE, 3 дисками (SAS HDD, NVMe, SSD), пулом и 3 LUN.
// tag (10 hex) делает серийные номера, WWN и имена уникальными в общей тестовой БД.
func Default(tag string) *Server {
	disk := func(id, typ, serial string, sectors int64) obj {
		return obj{"ID": id, "LOCATION": "CTE0.0." + id, "MODEL": "SSDM" + typ, "SERIALNUMBER": serial + tag, "FIRMWAREVER": "1.0", "DISKTYPE": typ,
			"SECTORS": strconv.FormatInt(sectors, 10), "SECTORSIZE": "520", "MANUFACTURER": "HUAWEI", "ENCLOSUREID": "0"}
	}
	wwn := func(n int) string { return "6" + tag + strings.Repeat("0", 20) + strconv.Itoa(n) } // 32 hex-символа, как у Huawei
	return &Server{
		User: "admin", Password: "secret", DeviceID: "2102350" + tag, MaxSession: 4, PageCap: 2, Name: "dorado-" + tag,
		Ctrls: []obj{{"ID": "0A", "LOCATION": "CTE0.A", "SOFTVER": "6.1.6"}, {"ID": "0B", "LOCATION": "CTE0.B", "SOFTVER": "6.1.6"}},
		Encl:  []obj{{"ID": "0", "NAME": "CTE0", "TYPE": 206, "SERIALNUM": "ENC0" + tag}, {"ID": "1", "NAME": "DAE010", "TYPE": 207, "SERIALNUM": "ENC1" + tag}},
		Pools: []obj{{"ID": "0", "NAME": "StoragePool001", "USERTOTALCAPACITY": "39062500000"}}, // секторов по 512 Б = 20 ТБ
		LUNs: []obj{
			{"ID": "0", "NAME": "lun-a-" + tag, "WWN": wwn(1), "CAPACITY": "209715200", "SECTORSIZE": "512", "PARENTID": "0", "ALLOCTYPE": "1"},
			{"ID": "1", "NAME": "lun-b-" + tag, "WWN": wwn(2), "CAPACITY": "419430400", "SECTORSIZE": "512", "PARENTID": "0", "ALLOCTYPE": "0"},
			{"ID": "2", "NAME": "lun-c-" + tag, "WWN": wwn(3), "CAPACITY": "104857600", "SECTORSIZE": "512", "PARENTID": "0", "ALLOCTYPE": "1"},
		},
		Hosts:  []obj{{"ID": "0", "NAME": "esx-" + tag}},
		Inits:  map[string][]string{"0": {strings.ReplaceAll(storage.SampleWWPN(tag, 4), ":", "")}},
		Mapped: map[string][]string{"0": {"0"}},
		Disks:  []obj{disk("0", "1", "HD", 1_153_846_153), disk("1", "14", "NV", 3_653_846_153), disk("2", "3", "SS", 1_500_000_000)},
	}
}

// Do выполняет f под блокировкой состояния.
func (s *Server) Do(f func(*Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

func reply(w http.ResponseWriter, data any, code int, desc string) {
	out := obj{"error": obj{"code": code, "description": desc}}
	if data != nil {
		out["data"] = data
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "/deviceManager/rest/xxxxx/sessions" && r.Method == http.MethodPost {
		var in struct{ Username, Password, Scope string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		if in.Username != s.User || in.Password != s.Password {
			reply(w, nil, 1077949061, "the user name or password is incorrect")
			return
		}
		if s.Sessions >= s.MaxSession {
			reply(w, nil, 1077949069, "reach the max login sessions")
			return
		}
		s.Sessions++
		s.Logins++
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "s", Path: "/"})
		reply(w, obj{"deviceid": s.DeviceID, "iBaseToken": "token-" + strconv.Itoa(s.Logins)}, 0, "")
		return
	}

	prefix := "/deviceManager/rest/" + s.DeviceID
	if !strings.HasPrefix(path, prefix) || r.Header.Get("iBaseToken") == "" {
		reply(w, nil, -401, "unauthorized")
		return
	}
	rest := strings.TrimPrefix(path, prefix)
	switch {
	case rest == "/sessions" && r.Method == http.MethodDelete:
		s.Sessions--
		s.Logouts++
		reply(w, obj{}, 0, "")
	case rest == "/system":
		reply(w, obj{"NAME": s.Name, "LOCATION": "lab", "PRODUCTMODE": "811", "PRODUCTVERSION": "6.1.6", "pointRelease": "SPH100", "SECTORSIZE": "512"}, 0, "")
	case rest == "/controller":
		reply(w, page(s.Ctrls, r, s.PageCap), 0, "")
	case rest == "/enclosure":
		reply(w, page(s.Encl, r, s.PageCap), 0, "")
	case rest == "/storagepool":
		reply(w, page(s.Pools, r, s.PageCap), 0, "")
	case rest == "/lun":
		if s.Fail {
			reply(w, nil, 50331651, "internal error")
			return
		}
		reply(w, page(s.LUNs, r, s.PageCap), 0, "")
	case rest == "/host":
		reply(w, page(s.Hosts, r, s.PageCap), 0, "")
	case rest == "/fc_initiator":
		rows := []obj{}
		for _, id := range s.Inits[r.URL.Query().Get("ASSOCIATEOBJID")] {
			rows = append(rows, obj{"ID": id, "TYPE": 223})
		}
		reply(w, page(rows, r, s.PageCap), 0, "")
	case rest == "/lun/associate":
		rows := []obj{}
		for _, id := range s.Mapped[r.URL.Query().Get("ASSOCIATEOBJID")] {
			rows = append(rows, obj{"ID": id, "TYPE": 11})
		}
		reply(w, page(rows, r, s.PageCap), 0, "")
	case rest == "/disk":
		light := []obj{}
		for _, d := range s.Disks { // облегчённые записи: без ёмкости, слота, полки
			light = append(light, obj{"ID": d["ID"], "PARENTID": "0", "CAPACITY": "", "SLOTNUM": "", "ENCLOSUREID": ""})
		}
		reply(w, page(light, r, s.PageCap), 0, "")
	case strings.HasPrefix(rest, "/disk/"):
		id := strings.TrimPrefix(rest, "/disk/")
		for _, d := range s.Disks {
			if d["ID"] == id {
				reply(w, d, 0, "")
				return
			}
		}
		reply(w, nil, 1077936859, "object does not exist")
	default:
		http.NotFound(w, r)
	}
}

// page отдаёт срез по range=[from-to], но не больше pageCap записей (сервер может отдавать меньше запрошенного).
func page(all []obj, r *http.Request, pageCap int) any {
	q := strings.Trim(r.URL.Query().Get("range"), "[]")
	from := 0
	if parts := strings.SplitN(q, "-", 2); len(parts) == 2 {
		from, _ = strconv.Atoi(parts[0])
	}
	if from > len(all) {
		from = len(all)
	}
	end := min(from+pageCap, len(all))
	if end == from {
		return nil // Huawei: пустая выборка — data отсутствует
	}
	return all[from:end]
}
