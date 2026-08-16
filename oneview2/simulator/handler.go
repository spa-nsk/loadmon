package simulator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Simulator) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path != "/" && !strings.HasPrefix(path, "/rest/") {
		http.NotFound(w, r)
		return
	}

	switch {
	case path == "/rest/version" && r.Method == http.MethodGet:
		s.handleVersion(w)
	case strings.HasPrefix(path, "/rest/login-sessions"):
		s.handleSession(w, r)
	case path == "/rest/server-hardware" && r.Method == http.MethodGet:
		s.handleServerHardwareList(w, r)
	case path == "/rest/racks" && r.Method == http.MethodGet:
		s.handleRackList(w, r)
	case path == "/rest/datacenters" && r.Method == http.MethodGet:
		s.handleDatacenterList(w, r)
	case path == "/rest/server-firmware" && r.Method == http.MethodGet:
		s.handleServerFirmwareList(w, r)
	case strings.HasPrefix(path, "/rest/enclosures/"):
		s.handleEnclosure(w, r, path)
	case strings.HasPrefix(path, "/rest/server-hardware/"):
		s.handleServerHardwareResource(w, r, path)
	default:
		http.NotFound(w, r)
	}
}

func (s *Simulator) handleVersion(w http.ResponseWriter) {
	if s.inventory.GlobalDash {
		writeJSON(w, http.StatusOK, map[string]int{
			"minimumVersion": 2,
			"currentVersion": 300,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"minimumVersion": 3800,
		"currentVersion": s.inventory.APIVersion,
	})
}

func (s *Simulator) handleSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var cred struct {
			UserName string `json:"userName"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&cred)
		if cred.UserName != s.cfg.Username || cred.Password != s.cfg.Password {
			writeJSON(w, http.StatusForbidden, map[string]string{"message": "Invalid credentials"})
			return
		}
		if s.inventory.GlobalDash {
			writeJSON(w, http.StatusOK, map[string]any{
				"token": "sim-gd-token",
				"user":  map[string]string{"userName": cred.UserName, "domain": "local"},
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"sessionID": "sim-appliance-session"})
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func (s *Simulator) handleServerHardwareList(w http.ResponseWriter, r *http.Request) {
	start, count := listParams(r)
	members := make([]map[string]any, 0)
	for _, uri := range s.inventory.ServerOrder {
		srv := s.inventory.Servers[uri]
		members = append(members, s.serverListMember(srv))
	}
	page, total, next := paginateMaps(members, start, count)
	writeJSON(w, http.StatusOK, collection[map[string]any]{
		Type:        "ServerHardwareListV7",
		Category:    "server-hardware",
		URI:         "/rest/server-hardware",
		Count:       len(page),
		Total:       total,
		Members:     page,
		NextPageURI: next,
	})
}

func (s *Simulator) serverListMember(srv *ServerRecord) map[string]any {
	return map[string]any{
		"name":               srv.Name,
		"uri":                srv.URI,
		"originalUri":        srv.OriginalURI,
		"uuid":               srv.UUID,
		"serialNumber":       srv.SerialNumber,
		"model":              srv.Model,
		"shortModel":         srv.ShortModel,
		"powerState":         srv.PowerState,
		"processorCount":     srv.ProcessorCount,
		"processorCoreCount": srv.ProcessorCoreCount,
		"memoryMb":           srv.MemoryMb,
	}
}

func (s *Simulator) handleServerHardwareResource(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	base := serverBasePath(path)
	if base == "" {
		http.NotFound(w, r)
		return
	}
	suffix := strings.TrimPrefix(path, base)
	suffix = strings.TrimPrefix(suffix, "/")

	srv := s.inventory.Servers[base]
	if srv == nil {
		http.NotFound(w, r)
		return
	}

	switch suffix {
	case "":
		writeJSON(w, http.StatusOK, s.serverDetail(srv))
	case "memory":
		writeJSON(w, http.StatusOK, map[string]any{
			"type": "ServerHardwareMemoryList", "members": srv.Memory, "data": srv.Memory,
		})
	case "memoryList":
		writeJSON(w, http.StatusOK, map[string]any{"members": []any{}})
	case "processors":
		writeJSON(w, http.StatusOK, map[string]any{
			"type": "ProcessorList", "members": srv.Processors, "data": srv.Processors,
		})
	case "devices":
		writeJSON(w, http.StatusOK, map[string]any{
			"type": "DeviceList", "members": srv.Devices, "data": srv.Devices,
		})
	case "localStorage", "localStorageV2":
		writeJSON(w, http.StatusOK, map[string]any{
			"type": "LocalStorageListV3", "members": srv.Controllers, "data": srv.Controllers,
			"controllers": srv.Controllers,
		})
	case "firmware":
		writeJSON(w, http.StatusOK, map[string]any{"members": srv.Firmware})
	default:
		http.NotFound(w, r)
	}
}

func (s *Simulator) serverDetail(srv *ServerRecord) map[string]any {
	m := map[string]any{
		"name":                   srv.Name,
		"serverName":             srv.ServerName,
		"uri":                    srv.URI,
		"originalUri":            srv.OriginalURI,
		"uuid":                   srv.UUID,
		"serialNumber":           srv.SerialNumber,
		"virtualSerialNumber":    srv.VirtualSerialNumber,
		"model":                  srv.Model,
		"shortModel":             srv.ShortModel,
		"formFactor":             srv.FormFactor,
		"powerState":             srv.PowerState,
		"status":                 srv.Status,
		"state":                  srv.State,
		"processorCount":         srv.ProcessorCount,
		"processorCoreCount":     srv.ProcessorCoreCount,
		"processorSpeedMhz":      srv.ProcessorSpeedMhz,
		"processorType":          srv.ProcessorType,
		"memoryMb":               srv.MemoryMb,
		"maintenanceMode":        false,
		"mpHostInfo":             map[string]string{"mpHostName": srv.ServerName + ".lab.local"},
		"portMap":                srv.PortMap,
		"romVersion":             "U32 v2.76 (10/15/2024)",
		"platform":               "Gen10",
		"hostOsType":             "Unknown",
		"serverHardwareTypeName": srv.ShortModel,
	}
	if srv.LocationURI != "" {
		m["locationUri"] = srv.LocationURI
		m["position"] = srv.Position
	}
	if !s.inventory.GlobalDash && len(srv.Processors) > 0 {
		m["subResources"] = map[string]any{
			"Processors":   map[string]any{"data": srv.Processors},
			"Memory":       map[string]any{"data": srv.Memory},
			"LocalStorage": map[string]any{"data": srv.Controllers},
		}
	}
	return m
}

func serverBasePath(path string) string {
	if !strings.HasPrefix(path, "/rest/server-hardware/") {
		return ""
	}
	rest := strings.TrimPrefix(path, "/rest/server-hardware/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	return "/rest/server-hardware/" + parts[0]
}

func (s *Simulator) handleEnclosure(w http.ResponseWriter, r *http.Request, path string) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	enc := s.inventory.Enclosures[path]
	if enc == nil {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":             enc.Name,
		"uri":              enc.URI,
		"originalUri":      enc.OriginalURI,
		"uuid":             enc.UUID,
		"serialNumber":     enc.SerialNumber,
		"enclosureModel":   enc.EnclosureModel,
		"enclosureType":    enc.EnclosureType,
		"rackName":         enc.RackName,
		"deviceBayCount":   enc.DeviceBayCount,
		"status":           "OK",
		"state":            "Configured",
		"activeOaPreferredIP": fmtIP(enc.ID),
		"deviceBays":       enc.DeviceBays,
	})
}

func (s *Simulator) handleRackList(w http.ResponseWriter, r *http.Request) {
	start, count := listParams(r)
	members := make([]map[string]any, 0, len(s.inventory.Racks))
	for _, rack := range s.inventory.Racks {
		members = append(members, map[string]any{
			"name":         rack.Name,
			"uri":          rack.URI,
			"originalUri":  rack.OriginalURI,
			"uuid":         rack.UUID,
			"model":        rack.Model,
			"serialNumber": rack.SerialNumber,
			"uHeight":      rack.UHeight,
			"rackMounts":   rack.RackMounts,
		})
	}
	page, total, next := paginateMaps(members, start, count)
	writeJSON(w, http.StatusOK, collection[map[string]any]{
		Count: len(page), Total: total, Members: page, NextPageURI: next,
	})
}

func (s *Simulator) handleDatacenterList(w http.ResponseWriter, r *http.Request) {
	start, count := listParams(r)
	members := make([]map[string]any, 0, len(s.inventory.Datacenters))
	for _, dc := range s.inventory.Datacenters {
		m := map[string]any{
			"name":            dc.Name,
			"uri":             dc.URI,
			"originalUri":     dc.OriginalURI,
			"uuid":            dc.UUID,
			"status":          dc.Status,
			"state":           dc.State,
			"coolingCapacity": dc.CoolingCapacity,
			"width":           dc.Width,
			"height":          dc.Height,
			"currency":        dc.Currency,
		}
		if s.inventory.GlobalDash {
			m["rackInventory"] = dc.RackInventory
		} else {
			m["contents"] = dc.Contents
		}
		members = append(members, m)
	}
	page, total, next := paginateMaps(members, start, count)
	writeJSON(w, http.StatusOK, collection[map[string]any]{
		Count: len(page), Total: total, Members: page, NextPageURI: next,
	})
}

func (s *Simulator) handleServerFirmwareList(w http.ResponseWriter, r *http.Request) {
	start, count := listParams(r)
	filterURI := firmwareFilter(r)
	members := make([]FirmwareRecord, 0)
	for _, fw := range s.inventory.Firmware {
		if filterURI != "" && fw.ServerHardwareOrig != filterURI && fw.ServerHardwareURI != filterURI {
			continue
		}
		members = append(members, fw)
	}
	page, total, next := paginateSlice(members, start, count)
	writeJSON(w, http.StatusOK, collection[FirmwareRecord]{
		Count: len(page), Total: total, Members: page, NextPageURI: next,
	})
}

func firmwareFilter(r *http.Request) string {
	for _, f := range r.URL.Query()["filter"] {
		if strings.Contains(f, "serverHardwareUri=") {
			return strings.Trim(strings.SplitN(f, "=", 2)[1], "'\"")
		}
	}
	if q := r.URL.Query().Get("query"); strings.Contains(q, "serverHardwareUri EQ") {
		parts := strings.Split(q, "'")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func listParams(r *http.Request) (start, count int) {
	start, _ = strconv.Atoi(r.URL.Query().Get("start"))
	count, _ = strconv.Atoi(r.URL.Query().Get("count"))
	if count == 0 {
		count = -1
	}
	return start, count
}

func paginateMaps[T any](items []T, start, count int) (page []T, total int, next string) {
	return paginateSlice(items, start, count)
}

func paginateSlice[T any](items []T, start, count int) (page []T, total int, next string) {
	total = len(items)
	if start >= total {
		return nil, total, ""
	}
	end := total
	if count > 0 && start+count < total {
		end = start + count
	}
	page = items[start:end]
	if end < total && count > 0 {
		next = fmt.Sprintf("?start=%d&count=%d", end, count)
	}
	return page, total, next
}

func fmtIP(id string) string {
	var n int
	for _, c := range id {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return fmt.Sprintf("172.16.%d.%d", (n>>8)&0xff, n&0xff)
}
