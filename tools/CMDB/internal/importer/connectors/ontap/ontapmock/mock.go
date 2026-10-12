// Package ontapmock — mock NetApp ONTAP REST для тестов и стенда (plan.md E5.6): кластер, узлы,
// агрегаты, SVM, тома, LUN, qtree, CIFS-ресурсы, полки, диски; постраничная выдача через
// _links.next. Форма ответов из документации, не проверена на живом кластере — harness/14.
package ontapmock

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"cmdb/internal/importer/connectors/storage"
)

// Server — состояние и обработчик mock-кластера.
type Server struct {
	mu       sync.Mutex
	User     string
	Password string
	PageCap  int  // записей на страницу
	Fail     bool // /api/storage/* отвечает HTTP 500

	Name, UUID, Version string
	Nodes               []Node
	Aggregates          []Aggregate
	SVMs                []SVM
	Volumes             []Volume
	LUNs                []LUN
	Qtrees              []Qtree
	Shares              []Share
	Shelves             []Shelf
	Disks               []Disk
	IGroups             []IGroup
	LUNMaps             []LUNMap
}

type Node struct{ Name, UUID, Serial, Model string }
type Aggregate struct {
	Name, UUID, RAID string
	SizeBytes        int64
}
type SVM struct{ Name, UUID, IP string }
type Volume struct {
	Name, UUID, SVM, Aggr string
	SizeBytes             int64
	Thin                  bool
	NASPath               string
}
type LUN struct {
	Name, UUID, Serial, SVM string
	SizeBytes               int64
}
type Qtree struct{ Name, ID, Volume string }
type Share struct{ Name, Path, SVM string }
type IGroup struct {
	Name, UUID string
	Initiators []string
}
type LUNMap struct{ LUN, IGroup string } // uuid LUN и igroup
type Shelf struct{ UID, Name, Model, Serial string }
type Disk struct {
	Name, Serial, Model, Type, Shelf string
	UsableBytes                      int64
}

// Default — кластер из 2 узлов: 2 агрегата, 1 SVM, 2 тома (один с NFS-путём), 1 LUN, 1 qtree, 1 CIFS-ресурс,
// 1 полка и 3 диска. tag (10 hex) делает UUID и имена уникальными в общей тестовой БД.
func Default(tag string) *Server {
	u := func(kind string, n int) string { return kind + "-" + tag + "-" + strconv.Itoa(n) }
	const tb = int64(1e12)
	return &Server{
		User: "admin", Password: "secret", PageCap: 2, Name: "ontap-" + tag, UUID: u("cluster", 0), Version: "NetApp Release 9.12.1",
		Nodes:      []Node{{"node-" + tag + "-01", u("node", 1), "SNODE" + tag + "1", "AFF-A400"}, {"node-" + tag + "-02", u("node", 2), "SNODE" + tag + "2", "AFF-A400"}},
		Aggregates: []Aggregate{{"aggr1_" + tag, u("aggr", 1), "raid_dp", 50 * tb}, {"aggr2_" + tag, u("aggr", 2), "raid_tec", 30 * tb}},
		SVMs:       []SVM{{"svm1_" + tag, u("svm", 1), "10.20.0.5"}},
		Volumes: []Volume{
			{"vol_data_" + tag, u("vol", 1), u("svm", 1), u("aggr", 1), 500 * int64(1e9), true, "/vol_data"},
			{"vol_lun_" + tag, u("vol", 2), u("svm", 1), u("aggr", 2), 200 * int64(1e9), false, ""},
		},
		LUNs:    []LUN{{"/vol/vol_lun_" + tag + "/lun1", u("lun", 1), "wXyZ" + tag[:8], u("svm", 1), 100 * int64(1e9)}},
		Qtrees:  []Qtree{{"", "0", u("vol", 1)}, {"team_a", "1", u("vol", 1)}},
		Shares:  []Share{{"data", "/vol_data", u("svm", 1)}, {"c$", "/", u("svm", 1)}},
		IGroups: []IGroup{{"esx-" + tag, u("ig", 1), []string{storage.SampleWWPN(tag, 1), storage.SampleWWPN(tag, 2), "iqn.1998-01.com.vmware:esx-" + tag}}},
		LUNMaps: []LUNMap{{u("lun", 1), u("ig", 1)}},
		Shelves: []Shelf{{"shelf-" + tag, "1.0", "DS224C", "SHELFSN" + tag}},
		Disks: []Disk{
			{"1.0.0", "DSN1" + tag, "X371", "ssd", "shelf-" + tag, 3800 * int64(1e9)},
			{"1.0.1", "DSN2" + tag, "X371", "ssd", "shelf-" + tag, 3800 * int64(1e9)},
			{"1.0.2", "DSN3" + tag, "X4001", "nvme_ssd", "shelf-" + tag, 1900 * int64(1e9)},
		},
	}
}

// Do выполняет f под блокировкой состояния.
func (s *Server) Do(f func(*Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

type obj = map[string]any

func ref(name, uuid string) obj { return obj{"name": name, "uuid": uuid} }

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, p, ok := r.BasicAuth(); !ok || u != s.User || p != s.Password {
		writeJSON(w, http.StatusUnauthorized, obj{"error": obj{"message": "Authentication failed", "code": "6691623"}})
		return
	}
	path := r.URL.Path
	if s.Fail && strings.HasPrefix(path, "/api/storage/") {
		writeJSON(w, http.StatusInternalServerError, obj{"error": obj{"message": "internal"}})
		return
	}

	var recs []any
	switch path {
	case "/api/cluster":
		writeJSON(w, http.StatusOK, obj{"name": s.Name, "uuid": s.UUID, "version": obj{"full": s.Version},
			"management_interfaces": []any{obj{"name": "cluster_mgmt", "ip": obj{"address": "10.20.0.1"}}}})
		return
	case "/api/cluster/nodes":
		for _, n := range s.Nodes {
			recs = append(recs, obj{"name": n.Name, "uuid": n.UUID, "serial_number": n.Serial, "model": n.Model, "version": obj{"full": s.Version}})
		}
	case "/api/storage/aggregates":
		for _, a := range s.Aggregates {
			recs = append(recs, obj{"name": a.Name, "uuid": a.UUID, "space": obj{"block_storage": obj{"size": a.SizeBytes}},
				"block_storage": obj{"primary": obj{"raid_type": a.RAID}, "storage_type": "ssd"}})
		}
	case "/api/svm/svms":
		for _, v := range s.SVMs {
			recs = append(recs, obj{"name": v.Name, "uuid": v.UUID, "ip_interfaces": []any{obj{"ip": obj{"address": v.IP}}}})
		}
	case "/api/storage/volumes":
		svmName := map[string]string{}
		for _, v := range s.SVMs {
			svmName[v.UUID] = v.Name
		}
		for _, v := range s.Volumes {
			m := obj{"name": v.Name, "uuid": v.UUID, "size": v.SizeBytes, "svm": ref(svmName[v.SVM], v.SVM), "aggregates": []any{obj{"uuid": v.Aggr}},
				"style": "flexvol", "guarantee": obj{"type": map[bool]string{true: "none", false: "volume"}[v.Thin]}}
			if v.NASPath != "" {
				m["nas"] = obj{"path": v.NASPath}
			}
			recs = append(recs, m)
		}
	case "/api/storage/luns":
		svmName := map[string]string{}
		for _, v := range s.SVMs {
			svmName[v.UUID] = v.Name
		}
		for _, l := range s.LUNs {
			recs = append(recs, obj{"name": l.Name, "uuid": l.UUID, "serial_number": l.Serial, "space": obj{"size": l.SizeBytes}, "svm": ref(svmName[l.SVM], l.SVM)})
		}
	case "/api/storage/qtrees":
		volSVM := map[string]string{}
		for _, v := range s.Volumes {
			volSVM[v.UUID] = v.SVM
		}
		for _, q := range s.Qtrees {
			recs = append(recs, obj{"name": q.Name, "id": json.Number(q.ID), "volume": obj{"uuid": q.Volume}, "svm": ref("svm", volSVM[q.Volume])})
		}
	case "/api/protocols/cifs/shares":
		svmName := map[string]string{}
		for _, v := range s.SVMs {
			svmName[v.UUID] = v.Name
		}
		for _, sh := range s.Shares {
			recs = append(recs, obj{"name": sh.Name, "path": sh.Path, "svm": ref(svmName[sh.SVM], sh.SVM)})
		}
	case "/api/protocols/san/igroups":
		for _, g := range s.IGroups {
			inits := []any{}
			for _, n := range g.Initiators {
				inits = append(inits, obj{"name": n})
			}
			recs = append(recs, obj{"name": g.Name, "uuid": g.UUID, "initiators": inits})
		}
	case "/api/protocols/san/lun-maps":
		for _, m := range s.LUNMaps {
			recs = append(recs, obj{"lun": obj{"uuid": m.LUN}, "igroup": obj{"uuid": m.IGroup}})
		}
	case "/api/storage/shelves":
		for _, sh := range s.Shelves {
			recs = append(recs, obj{"uid": sh.UID, "name": sh.Name, "model": sh.Model, "serial_number": sh.Serial})
		}
	case "/api/storage/disks":
		for _, d := range s.Disks {
			recs = append(recs, obj{"name": d.Name, "serial_number": d.Serial, "model": d.Model, "type": d.Type, "usable_size": d.UsableBytes,
				"firmware_version": "NA01", "vendor": "NETAPP", "shelf": obj{"uid": d.Shelf}})
		}
	default:
		http.NotFound(w, r)
		return
	}

	start, _ := strconv.Atoi(r.URL.Query().Get("_start"))
	end := min(start+s.PageCap, len(recs))
	out := obj{"records": recs[start:end], "num_records": end - start}
	if end < len(recs) {
		q := r.URL.Query()
		q.Set("_start", strconv.Itoa(end))
		out["_links"] = obj{"next": obj{"href": path + "?" + q.Encode()}}
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
