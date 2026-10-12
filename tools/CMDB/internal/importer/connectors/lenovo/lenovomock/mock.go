// Package lenovomock — mock встроенного REST Lenovo DE / NetApp SANtricity для тестов и стенда
// (plan.md E5.6): GET /devmgr/v2/storage-systems/1 и /graph. Ёмкости строками, как в ответах
// реального массива (harness/03: типы расходятся с документацией). Имена полей graph — по
// документации, не проверены на живом массиве — harness/14-source-storage.md.
package lenovomock

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
	mu       sync.Mutex
	User     string
	Password string
	Fail     bool

	Label, Serial string
	Graph         obj
	Hosts         []any
	Mappings      []any
}

// Default — массив с 2 контроллерами, 1 полкой, 3 дисками, группой томов, дисковым пулом и 3 томами.
// tag (10 hex) делает серийные номера, WWN и имена уникальными в общей тестовой БД.
func Default(tag string) *Server {
	wwn := func(n int) string { return "60080E50002" + tag + strings.Repeat("0", 10) + strconv.Itoa(n) } // 32 hex
	s := &Server{User: "admin", Password: "secret", Label: "de-" + tag, Serial: "DESN" + tag}
	s.Graph = obj{
		"sa":         obj{"saData": obj{"storageArrayLabel": s.Label}, "chassisSerialNumber": s.Serial, "fwVersion": "08.74.00.00"},
		"controller": []any{obj{"id": "070000000000000000000001", "modelName": "DE4000H", "serialNumber": "CA" + tag, "appVersion": "08.74.00.00", "physicalLocation": obj{"label": "Controller A"}}, obj{"id": "070000000000000000000002", "modelName": "DE4000H", "serialNumber": "CB" + tag, "appVersion": "08.74.00.00", "physicalLocation": obj{"label": "Controller B"}}},
		"tray":       []any{obj{"id": "0E00000000000000000000A1", "trayId": 0, "serialNumber": "TR" + tag, "partNumber": "DE240S"}},
		"drive": []any{
			obj{"id": "0100000050000001", "serialNumber": "DK1" + tag, "productID": "AL15SEB", "manufacturer": "TOSHIBA", "rawCapacity": "1200243695616", "driveMediaType": "hdd", "firmwareVersion": "E1", "physicalLocation": obj{"trayRef": "0E00000000000000000000A1", "slot": 1}},
			obj{"id": "0100000050000002", "serialNumber": "DK2" + tag, "productID": "PM1643", "manufacturer": "SAMSUNG", "rawCapacity": "1920383410176", "driveMediaType": "ssd", "firmwareVersion": "EDA5", "physicalLocation": obj{"trayRef": "0E00000000000000000000A1", "slot": 2}},
			obj{"id": "0100000050000003", "serialNumber": "DK3" + tag, "productID": "PM1735", "manufacturer": "SAMSUNG", "rawCapacity": "3200631791616", "driveMediaType": "ssd", "interfaceType": obj{"driveType": "nvme"}, "physicalLocation": obj{"trayRef": "0E00000000000000000000A1", "slot": 3}},
		},
		"volumeGroup": []any{obj{"id": "04000000600A098000", "label": "VG1", "raidLevel": "raid6", "totalRaidedSpace": "3600000000000"}},
		"diskPool":    []any{obj{"id": "04000000600A098001", "label": "DP1", "raidLevel": "raidDiskPool", "totalRaidedSpace": "9600000000000"}},
		"volume": []any{
			obj{"id": "02000000600A0980001", "label": "vol-a-" + tag, "capacity": "107374182400", "worldWideName": wwn(1), "volumeGroupRef": "04000000600A098000", "thinProvisioned": false, "volumeHandle": 0},
			obj{"id": "02000000600A0980002", "label": "vol-b-" + tag, "capacity": "214748364800", "worldWideName": wwn(2), "volumeGroupRef": "04000000600A098001", "thinProvisioned": true, "volumeHandle": 1},
		},
		"thinVolume": []any{obj{"id": "02000000600A0980003", "label": "thin-c-" + tag, "capacity": "53687091200", "worldWideName": wwn(3), "diskPoolRef": "04000000600A098001", "thinProvisioned": true}},
	}
	s.Hosts = []any{obj{"hostRef": "84000000600A0980HOST1", "label": "esx-" + tag, "clusterRef": "0000000000000000000000000000000000000000",
		"initiators": []any{obj{"nodeName": obj{"ioInterfaceType": "fc", "remoteNodeWWN": strings.ReplaceAll(storage.SampleWWPN(tag, 5), ":", "")}}}}}
	s.Mappings = []any{obj{"lunMappingRef": "88000000A1", "volumeRef": "02000000600A0980001", "mapRef": "84000000600A0980HOST1", "lun": 0, "type": "host"}}
	return s
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
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(obj{"errorMessage": "Unauthorized", "retcode": "denied"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/devmgr/v2/storage-systems/1":
		_ = json.NewEncoder(w).Encode(obj{"id": "1", "name": s.Label, "status": "optimal"})
	case "/devmgr/v2/storage-systems/1/hosts":
		_ = json.NewEncoder(w).Encode(s.Hosts)
	case "/devmgr/v2/storage-systems/1/volume-mappings":
		_ = json.NewEncoder(w).Encode(s.Mappings)
	case "/devmgr/v2/storage-systems/1/graph":
		if s.Fail {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(s.Graph)
	default:
		http.NotFound(w, r)
	}
}
