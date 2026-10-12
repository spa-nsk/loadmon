// Package openmanagemock — mock-сервер Dell OpenManage Enterprise REST API для тестов (plan.md
// E8.2): сессии (X-Auth-Token), устройства (DeviceService/Devices, постранично через
// @odata.nextLink) и InventoryDetails. Форма ответов — из общедоступной документации, не
// проверена на живом устройстве (см. комментарий в ../openmanage.go).
package openmanagemock

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
)

const (
	typeServer  = 1000
	typeChassis = 2000
)

// CPU/Memory/Disk/PSU — компоненты устройства (InventoryDetails).
type CPU struct {
	Manufacturer, Model, Speed string
	Cores                      int
}
type Memory struct {
	Bank, Manufacturer, PartNumber, Serial, Speed string
	SizeMB                                        int
}
type Disk struct {
	Slot, Manufacturer, ProductID, Serial, Revision, MediaType string
	CapacityBytes                                              float64
}
type PSU struct{ Name, Manufacturer, PartNumber, Serial string }

// Device — одно устройство (сервер или шасси).
type Device struct {
	ID                int64
	ServiceTag        string
	Name              string
	Model             string
	Type              int // typeServer | typeChassis
	ChassisServiceTag string
	SlotNumber        int
	BMCIP             string
	OSName            string
	BIOSVersion       string
	CPUs              []CPU
	Memory            []Memory
	Disks             []Disk
	PSUs              []PSU
}

// Server — состояние и обработчик mock-appliance.
type Server struct {
	mu       sync.Mutex
	User     string
	Password string
	PageCap  int
	Fail     bool
	Logins   int
	Logouts  int

	Devices []Device

	token     string
	sessionID int64
}

// Default — стенд: 1 шасси с 1 блейд-сервером и 1 стоечный сервер (tag делает серийные номера и
// имена уникальными в общей тестовой БД).
func Default(tag string) *Server {
	return &Server{
		User: "admin", Password: "secret", PageCap: 2,
		Devices: []Device{
			{ID: 1, ServiceTag: "CH-" + tag, Name: "MX7000-" + tag, Model: "PowerEdge MX7000", Type: typeChassis},
			{ID: 2, ServiceTag: "BL-" + tag, Name: "blade-" + tag, Model: "PowerEdge MX840c", Type: typeServer,
				ChassisServiceTag: "CH-" + tag, SlotNumber: 1, BMCIP: "10.0.2.1", OSName: "VMware ESXi 8.0", BIOSVersion: "2.10.2",
				CPUs:   []CPU{{Manufacturer: "Intel", Model: "Xeon Gold 6338", Cores: 32, Speed: "2.0 GHz"}, {Manufacturer: "Intel", Model: "Xeon Gold 6338", Cores: 32, Speed: "2.0 GHz"}},
				Memory: []Memory{{Bank: "A1", Manufacturer: "Samsung", PartNumber: "M393A4K40", Serial: "MEM-" + tag + "-1", SizeMB: 32768, Speed: "3200 MT/s"}},
				Disks:  []Disk{{Slot: "0", Manufacturer: "Dell", ProductID: "SSDSC2KG96", Serial: "DSK-" + tag + "-1", Revision: "GXT5", MediaType: "SSD", CapacityBytes: 960e9}},
				PSUs:   []PSU{{Name: "PSU.Slot.1", Manufacturer: "Dell", PartNumber: "0YHDN", Serial: "PSU-" + tag + "-1"}},
			},
			{ID: 3, ServiceTag: "RK-" + tag, Name: "rack-" + tag, Model: "PowerEdge R750", Type: typeServer,
				BMCIP: "10.0.2.2", OSName: "Red Hat Enterprise Linux 9", BIOSVersion: "1.8.2",
				CPUs:   []CPU{{Manufacturer: "Intel", Model: "Xeon Silver 4314", Cores: 16, Speed: "2.4 GHz"}},
				Memory: []Memory{{Bank: "A1", Manufacturer: "SK Hynix", PartNumber: "HMA84GR7", Serial: "MEM-" + tag + "-2", SizeMB: 16384, Speed: "3200 MT/s"}},
				Disks:  []Disk{{Slot: "0", Manufacturer: "Dell", ProductID: "ST2000NX0463", Serial: "DSK-" + tag + "-2", Revision: "SN04", MediaType: "HDD", CapacityBytes: 2e12}},
				PSUs:   []PSU{{Name: "PSU.Slot.1", Manufacturer: "Dell", PartNumber: "0YHDN", Serial: "PSU-" + tag + "-2"}},
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

var invDetailsRe = regexp.MustCompile(`^/api/DeviceService/Devices\((\d+)\)/InventoryDetails$`)
var sessionDeleteRe = regexp.MustCompile(`^/api/SessionService/Sessions\(\d+\)$`)

// ServeHTTP реализует http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r.URL.Path == "/api/SessionService/Sessions" || sessionDeleteRe.MatchString(r.URL.Path) {
		switch r.Method {
		case http.MethodPost:
			var in struct{ UserName, Password string }
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in.UserName != s.User || in.Password != s.Password {
				writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "CGEN1006"}})
				return
			}
			s.Logins++
			s.sessionID++
			s.token = fmt.Sprintf("token-%d", s.sessionID)
			w.Header().Set("X-Auth-Token", s.token)
			writeJSON(w, http.StatusCreated, map[string]any{"Id": s.sessionID})
		case http.MethodDelete:
			s.Logouts++
			s.token = ""
			w.WriteHeader(http.StatusNoContent)
		}
		return
	}

	if s.token == "" || r.Header.Get("X-Auth-Token") != s.token {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "CGEN1006"}})
		return
	}

	if s.Fail && r.URL.Path == "/api/DeviceService/Devices" {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": map[string]any{"message": "CGEN1001"}})
		return
	}

	if m := invDetailsRe.FindStringSubmatch(r.URL.Path); m != nil {
		id, _ := strconv.ParseInt(m[1], 10, 64)
		for _, d := range s.Devices {
			if d.ID == id {
				writeJSON(w, http.StatusOK, map[string]any{"value": inventoryValue(d)})
				return
			}
		}
		http.NotFound(w, r)
		return
	}

	if r.URL.Path == "/api/DeviceService/Devices" {
		var members []any
		for _, d := range s.Devices {
			members = append(members, deviceJSON(d))
		}
		top, _ := strconv.Atoi(r.URL.Query().Get("$top"))
		skip, _ := strconv.Atoi(r.URL.Query().Get("$skip"))
		if top <= 0 || top > s.PageCap {
			top = s.PageCap
		}
		end := min(skip+top, len(members))
		if skip > len(members) {
			skip = len(members)
		}
		out := map[string]any{"@odata.count": len(members), "value": members[skip:end]}
		if end < len(members) {
			out["@odata.nextLink"] = fmt.Sprintf("/api/DeviceService/Devices?%s", url.Values{
				"$top": {strconv.Itoa(top)}, "$skip": {strconv.Itoa(end)},
			}.Encode())
		}
		writeJSON(w, http.StatusOK, out)
		return
	}

	http.NotFound(w, r)
}

func deviceJSON(d Device) map[string]any {
	m := map[string]any{"Id": d.ID, "DeviceServiceTag": d.ServiceTag, "DeviceName": d.Name, "Model": d.Model, "Type": d.Type,
		"DeviceManagement": []any{map[string]any{"NetworkAddress": d.BMCIP}}}
	if d.ChassisServiceTag != "" {
		m["ChassisServiceTag"] = d.ChassisServiceTag
		m["SlotNumber"] = d.SlotNumber
	}
	return m
}

func inventoryValue(d Device) []any {
	var out []any
	if len(d.CPUs) > 0 {
		var info []any
		for i, c := range d.CPUs {
			info = append(info, map[string]any{"SlotNumber": strconv.Itoa(i + 1), "Manufacturer": c.Manufacturer, "ModelName": c.Model,
				"NumberOfCores": c.Cores, "MaxSpeed": c.Speed})
		}
		out = append(out, map[string]any{"InventoryType": "serverProcessors", "InventoryInfo": info})
	}
	if len(d.Memory) > 0 {
		var info []any
		for _, m := range d.Memory {
			info = append(info, map[string]any{"BankName": m.Bank, "Manufacturer": m.Manufacturer, "PartNumber": m.PartNumber,
				"SerialNumber": m.Serial, "Size": m.SizeMB, "Speed": m.Speed})
		}
		out = append(out, map[string]any{"InventoryType": "serverMemoryDevices", "InventoryInfo": info})
	}
	if len(d.Disks) > 0 {
		var info []any
		for _, disk := range d.Disks {
			info = append(info, map[string]any{"Slot": disk.Slot, "Manufacturer": disk.Manufacturer, "ProductId": disk.ProductID,
				"SerialNo": disk.Serial, "Revision": disk.Revision, "MediaType": disk.MediaType, "Capacity": disk.CapacityBytes})
		}
		out = append(out, map[string]any{"InventoryType": "serverArrayDisks", "InventoryInfo": info})
	}
	if len(d.PSUs) > 0 {
		var info []any
		for _, p := range d.PSUs {
			info = append(info, map[string]any{"Name": p.Name, "Manufacturer": p.Manufacturer, "PartNumber": p.PartNumber, "SerialNumber": p.Serial})
		}
		out = append(out, map[string]any{"InventoryType": "serverPowerSupplies", "InventoryInfo": info})
	}
	if d.BIOSVersion != "" {
		out = append(out, map[string]any{"InventoryType": "serverBIOS", "InventoryInfo": []any{map[string]any{"Version": d.BIOSVersion}}})
	}
	if d.OSName != "" {
		out = append(out, map[string]any{"InventoryType": "osInformation", "InventoryInfo": []any{map[string]any{"OsName": d.OSName}}})
	}
	return out
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
