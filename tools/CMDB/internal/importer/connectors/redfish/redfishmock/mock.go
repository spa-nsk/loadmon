// Package redfishmock — mock одного BMC с Redfish API для тестов и стенда (plan.md E5.6): дерево
// ресурсов ServiceRoot → Systems/Chassis/Managers в форме спецификации DMTF Redfish. Не проверялся
// на живых BMC — harness/13-source-redfish.md.
package redfishmock

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Server — состояние одного BMC.
type Server struct {
	mu       sync.Mutex
	User     string
	Password string
	Fail     bool   // ресурсы Systems отвечают HTTP 500 (BMC «завис»)
	FailPath string // ресурсы, путь которых содержит эту подстроку, отвечают HTTP 500 (сбой отдельного подресурса)

	Manufacturer, Model, SKU, Serial, PartNumber, UUID, HostName, BiosVersion string
	ChassisType                                                               string // Blade | RackMount | Tower
	MemoryGiB                                                                 int

	CPUs   []CPU
	DIMMs  []DIMM
	Drives []Drive
	PSUs   []PSU
	Fans   []string
	NICs   []NIC

	BMCID, BMCModel, BMCFirmware, BMCMAC, BMCIP, BMCHostName string
}

type CPU struct {
	ID, Socket, Model, Manufacturer string
	Cores                           int
	MaxMHz                          int
	Absent                          bool
}

type DIMM struct {
	ID, Locator, Manufacturer, Part, Serial string
	CapacityMiB, SpeedMHz                   int
	Absent                                  bool
}

type Drive struct {
	ID, Model, Manufacturer, Serial, Revision string
	CapacityBytes                             int64
	Media                                     string // HDD | SSD
	Protocol                                  string // SAS | SATA | NVMe
}

type PSU struct {
	Name, Model, Serial, Firmware string
	Watts                         int
}

type NIC struct {
	ID, MAC string
	Mbps    int
	Enabled bool
}

// New — стоечный двухпроцессорный сервер: 2 CPU, 2 DIMM, 2 диска (HDD SAS + NVMe), 2 БП, 2 вентилятора, 2 порта.
// tag (10 hex) делает серийные номера, UUID и MAC уникальными в общей тестовой БД, n (1..99) — номер сервера.
func New(tag string, n int) *Server {
	suf := tag + pad(n)
	mac := func(k int) string {
		return "aa:bb:" + tag[0:2] + ":" + tag[2:4] + ":" + tag[4:6] + ":" + pad(n*10 + k)[0:2]
	}
	return &Server{
		User: "root", Password: "secret",
		Manufacturer: "HPE", Model: "ProLiant DL380 Gen10", SKU: "868703-B21", Serial: "SN-RF-" + suf, PartNumber: "868703-B21",
		UUID: "3" + tag[1:8] + "-0000-4000-8000-" + tag[0:10] + pad(n), HostName: "rf-host-" + suf, BiosVersion: "U30 v2.72", ChassisType: "RackMount", MemoryGiB: 64,
		CPUs: []CPU{
			{ID: "1", Socket: "Proc 1", Model: "Intel Xeon Gold 6230", Manufacturer: "Intel", Cores: 20, MaxMHz: 3900},
			{ID: "2", Socket: "Proc 2", Model: "Intel Xeon Gold 6230", Manufacturer: "Intel", Cores: 20, MaxMHz: 3900},
		},
		DIMMs: []DIMM{
			{ID: "1", Locator: "PROC 1 DIMM 1", Manufacturer: "Samsung", Part: "M393A4K40CB2", Serial: "DIMM-A-" + suf, CapacityMiB: 32768, SpeedMHz: 2933},
			{ID: "2", Locator: "PROC 2 DIMM 1", Manufacturer: "Samsung", Part: "M393A4K40CB2", Serial: "DIMM-B-" + suf, CapacityMiB: 32768, SpeedMHz: 2933},
		},
		Drives: []Drive{
			{ID: "1", Model: "EG1200JEHMC", Manufacturer: "HPE", Serial: "DRV-1-" + suf, Revision: "HPD9", CapacityBytes: 1200000000000, Media: "HDD", Protocol: "SAS"},
			{ID: "2", Model: "MZ1LB960HAJQ", Manufacturer: "Samsung", Serial: "DRV-2-" + suf, Revision: "EDA5", CapacityBytes: 960000000000, Media: "SSD", Protocol: "NVMe"},
		},
		PSUs:  []PSU{{"Power Supply 1", "865408-B21", "PSU-1-" + suf, "1.00", 800}, {"Power Supply 2", "865408-B21", "PSU-2-" + suf, "1.00", 800}},
		Fans:  []string{"Fan 1", "Fan 2"},
		NICs:  []NIC{{"NIC1", mac(1), 10000, true}, {"NIC2", mac(2), 10000, true}},
		BMCID: "1", BMCModel: "iLO 5", BMCFirmware: "iLO 5 v2.72", BMCMAC: mac(9), BMCIP: "10.0.9." + strconv.Itoa(n), BMCHostName: "rf-host-" + suf + "-bmc",
	}
}

func hex2(n int) string {
	const h = "0123456789abcdef"
	return string([]byte{h[n>>4], h[n&15]})
}

func pad(n int) string {
	s := strconv.Itoa(n)
	if len(s) < 2 {
		s = "0" + s
	}
	return s
}

// Do выполняет f под блокировкой состояния.
func (s *Server) Do(f func(*Server)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f(s)
}

type obj = map[string]any

func ref(path string) obj { return obj{"@odata.id": path} }

func collection(base string, ids []string) obj {
	members := []any{}
	for _, id := range ids {
		members = append(members, ref(base+"/"+id))
	}
	return obj{"Members": members, "Members@odata.count": len(members)}
}

func state(absent bool) obj {
	if absent {
		return obj{"State": "Absent"}
	}
	return obj{"State": "Enabled", "Health": "OK"}
}

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if u, p, ok := r.BasicAuth(); !ok || u != s.User || p != s.Password {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(obj{"error": obj{"code": "Base.1.0.NoValidSession"}})
		return
	}
	path := strings.TrimRight(r.URL.Path, "/")
	if s.Fail && strings.HasPrefix(path, "/redfish/v1/Systems") {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if s.FailPath != "" && strings.Contains(path, s.FailPath) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body, ok := s.resource(path)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(obj{"error": obj{"code": "Base.1.0.ResourceMissingAtURI"}})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func (s *Server) resource(path string) (obj, bool) {
	sys := "/redfish/v1/Systems/1"
	cha := "/redfish/v1/Chassis/1"
	mgr := "/redfish/v1/Managers/" + s.BMCID
	switch path {
	case "/redfish/v1":
		return obj{"RedfishVersion": "1.11.0", "Systems": ref("/redfish/v1/Systems"), "Chassis": ref("/redfish/v1/Chassis"), "Managers": ref("/redfish/v1/Managers")}, true
	case "/redfish/v1/Systems":
		return collection("/redfish/v1/Systems", []string{"1"}), true
	case sys:
		return obj{"Id": "1", "Manufacturer": s.Manufacturer, "Model": s.Model, "SKU": s.SKU, "SerialNumber": s.Serial, "PartNumber": s.PartNumber,
			"UUID": strings.ToUpper(s.UUID), "HostName": s.HostName, "BiosVersion": s.BiosVersion, "PowerState": "On",
			"ProcessorSummary": obj{"Count": len(s.CPUs), "Model": "Intel Xeon"}, "MemorySummary": obj{"TotalSystemMemoryGiB": s.MemoryGiB},
			"Processors": ref(sys + "/Processors"), "Memory": ref(sys + "/Memory"), "Storage": ref(sys + "/Storage"), "EthernetInterfaces": ref(sys + "/EthernetInterfaces"),
			"Links": obj{"Chassis": []any{ref(cha)}, "ManagedBy": []any{ref(mgr)}}}, true
	case sys + "/Processors":
		return collection(sys+"/Processors", ids(len(s.CPUs), func(i int) string { return s.CPUs[i].ID })), true
	case sys + "/Memory":
		return collection(sys+"/Memory", ids(len(s.DIMMs), func(i int) string { return s.DIMMs[i].ID })), true
	case sys + "/Storage":
		return collection(sys+"/Storage", []string{"1"}), true
	case sys + "/Storage/1":
		drives := []any{}
		for _, d := range s.Drives {
			drives = append(drives, ref(sys+"/Storage/1/Drives/"+d.ID))
		}
		return obj{"Id": "1", "StorageControllers": []any{obj{"Manufacturer": "HPE", "Model": "Smart Array P408i-a", "SerialNumber": "RAID-" + s.Serial, "FirmwareVersion": "2.65"}}, "Drives": drives}, true
	case sys + "/EthernetInterfaces":
		return collection(sys+"/EthernetInterfaces", ids(len(s.NICs), func(i int) string { return s.NICs[i].ID })), true
	case "/redfish/v1/Chassis":
		return collection("/redfish/v1/Chassis", []string{"1"}), true
	case cha:
		return obj{"Id": "1", "ChassisType": s.ChassisType, "Power": ref(cha + "/Power"), "Thermal": ref(cha + "/Thermal")}, true
	case cha + "/Power":
		psus := []any{}
		for _, p := range s.PSUs {
			psus = append(psus, obj{"Name": p.Name, "Model": p.Model, "SerialNumber": p.Serial, "FirmwareVersion": p.Firmware, "Manufacturer": "HPE", "PowerCapacityWatts": p.Watts, "Status": state(false)})
		}
		return obj{"PowerSupplies": psus}, true
	case cha + "/Thermal":
		fans := []any{}
		for i, f := range s.Fans {
			fans = append(fans, obj{"Name": f, "MemberId": strconv.Itoa(i), "Status": state(false)})
		}
		return obj{"Fans": fans}, true
	case "/redfish/v1/Managers":
		return collection("/redfish/v1/Managers", []string{s.BMCID}), true
	case mgr:
		return obj{"Id": s.BMCID, "Model": s.BMCModel, "FirmwareVersion": s.BMCFirmware, "Manufacturer": s.Manufacturer,
			"EthernetInterfaces": ref(mgr + "/EthernetInterfaces"), "NetworkProtocol": ref(mgr + "/NetworkProtocol")}, true
	case mgr + "/EthernetInterfaces":
		return collection(mgr+"/EthernetInterfaces", []string{"1"}), true
	case mgr + "/EthernetInterfaces/1":
		return obj{"Id": "1", "MACAddress": strings.ToUpper(s.BMCMAC), "IPv4Addresses": []any{obj{"Address": s.BMCIP}}, "Status": state(false)}, true
	case mgr + "/NetworkProtocol":
		return obj{"HostName": s.BMCHostName}, true
	}
	for i, c := range s.CPUs {
		if path == sys+"/Processors/"+c.ID {
			return obj{"Id": c.ID, "Socket": c.Socket, "Model": c.Model, "Manufacturer": c.Manufacturer, "TotalCores": c.Cores, "TotalThreads": c.Cores * 2,
				"MaxSpeedMHz": c.MaxMHz, "ProcessorType": "CPU", "Status": state(s.CPUs[i].Absent)}, true
		}
	}
	for i, d := range s.DIMMs {
		if path == sys+"/Memory/"+d.ID {
			return obj{"Id": d.ID, "DeviceLocator": d.Locator, "Manufacturer": d.Manufacturer, "PartNumber": d.Part, "SerialNumber": d.Serial,
				"CapacityMiB": d.CapacityMiB, "OperatingSpeedMhz": d.SpeedMHz, "Status": state(s.DIMMs[i].Absent)}, true
		}
	}
	for _, d := range s.Drives {
		if path == sys+"/Storage/1/Drives/"+d.ID {
			return obj{"Id": d.ID, "Model": d.Model, "Manufacturer": d.Manufacturer, "SerialNumber": d.Serial, "Revision": d.Revision,
				"CapacityBytes": d.CapacityBytes, "MediaType": d.Media, "Protocol": d.Protocol, "Status": state(false)}, true
		}
	}
	for _, n := range s.NICs {
		if path == sys+"/EthernetInterfaces/"+n.ID {
			st := state(false)
			if !n.Enabled {
				st = obj{"State": "Disabled"}
			}
			return obj{"Id": n.ID, "MACAddress": strings.ToUpper(n.MAC), "SpeedMbps": n.Mbps, "Status": st}, true
		}
	}
	return nil, false
}

func ids(n int, f func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = f(i)
	}
	return out
}
