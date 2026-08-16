package simulator

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Simulator serves REST endpoints consumed by github.com/spa-nsk/oneview2.
type Simulator struct {
	cfg       InstanceConfig
	inventory *Inventory
}

// Inventory holds generated datacenter topology and server payloads.
type Inventory struct {
	APIVersion int
	GlobalDash bool

	Servers     map[string]*ServerRecord
	ServerOrder []string
	Enclosures  map[string]*EnclosureRecord
	Racks       map[string]*RackRecord
	Datacenters []*DatacenterRecord
	Firmware    []FirmwareRecord
}

type ServerRecord struct {
	ID                  string
	URI                 string
	OriginalURI         string
	UUID                string
	SerialNumber        string
	VirtualSerialNumber string
	Name                string
	ServerName          string
	Model               string
	ShortModel          string
	FormFactor          string
	PowerState          string
	Status              string
	State               string
	Position            int
	LocationURI         string
	RackURI             string
	TopUSlot            int
	UHeight             int
	IsBlade             bool

	ProcessorCount     int
	ProcessorCoreCount int
	ProcessorSpeedMhz  int
	ProcessorType      string
	MemoryMb           int

	Processors  []ProcessorRecord
	Memory      []MemoryRecord
	Controllers []ControllerRecord
	Devices     []DeviceRecord
	PortMap     []PortRecord
	Firmware    []FirmwareRecord
}

type ProcessorRecord struct {
	ID                string `json:"Id"`
	Socket            string `json:"Socket"`
	Model             string `json:"Model"`
	Manufacturer      string `json:"Manufacturer"`
	TotalCores        int    `json:"TotalCores"`
	TotalThreads      int    `json:"TotalThreads"`
	MaxSpeedMHz       int    `json:"MaxSpeedMHz"`
	OperatingSpeedMHz int    `json:"OperatingSpeedMHz"`
	Status            status `json:"Status"`
}

type MemoryRecord struct {
	Name              string         `json:"Name"`
	DeviceLocator     string         `json:"DeviceLocator"`
	Manufacturer      string         `json:"Manufacturer"`
	PartNumber        string         `json:"PartNumber"`
	SerialNumber      string         `json:"SerialNumber"`
	CapacityMiB       int            `json:"CapacityMiB"`
	OperatingSpeedMhz int            `json:"OperatingSpeedMhz"`
	BaseModuleType      string         `json:"BaseModuleType"`
	MemoryDeviceType  string         `json:"MemoryDeviceType"`
	MemoryType        string         `json:"MemoryType"`
	RankCount         int            `json:"RankCount"`
	ErrorCorrection   string         `json:"ErrorCorrection"`
	MemoryLocation    memoryLocation `json:"MemoryLocation"`
	Oem               memoryOem      `json:"Oem"`
	Status            status         `json:"Status"`
}

type memoryLocation struct {
	Channel          int `json:"Channel"`
	MemoryController int `json:"MemoryController"`
	Slot             int `json:"Slot"`
	Socket           int `json:"Socket"`
}

type memoryOem struct {
	Hpe struct {
		DIMMStatus string   `json:"DIMMStatus"`
		Attributes []string `json:"Attributes"`
	} `json:"Hpe"`
}

type ControllerRecord struct {
	Name               string         `json:"Name"`
	Model              string         `json:"Model"`
	SerialNumber       string         `json:"SerialNumber"`
	AdapterType        string         `json:"AdapterType"`
	Location           string         `json:"Location"`
	FirmwareVersion    nestedFirmware `json:"FirmwareVersion"`
	CacheMemorySizeMiB int            `json:"CacheMemorySizeMiB"`
	Status             status         `json:"Status"`
	PhysicalDrives     []DriveRecord  `json:"PhysicalDrives"`
	LogicalDrives      []VolumeRecord `json:"LogicalDrives"`
}

type DriveRecord struct {
	Location        string         `json:"Location"`
	Model           string         `json:"Model"`
	SerialNumber    string         `json:"SerialNumber"`
	MediaType       string         `json:"MediaType"`
	InterfaceType   string         `json:"InterfaceType"`
	CapacityMiB     int            `json:"CapacityMiB"`
	DiskDriveUse    string         `json:"DiskDriveUse"`
	FirmwareVersion nestedFirmware `json:"FirmwareVersion"`
	EncryptedDrive  bool           `json:"EncryptedDrive"`
	Status          status         `json:"Status"`
}

type VolumeRecord struct {
	LogicalDriveName   string `json:"LogicalDriveName"`
	LogicalDriveNumber int    `json:"LogicalDriveNumber"`
	Raid               string `json:"Raid"`
	LogicalDriveType   string `json:"LogicalDriveType"`
	CapacityMiB        int    `json:"CapacityMiB"`
	Status             status `json:"Status"`
}

type nestedFirmware struct {
	Current struct {
		VersionString string `json:"VersionString"`
	} `json:"Current"`
}

type DeviceRecord struct {
	ID                string `json:"Id"`
	Name              string `json:"Name"`
	DeviceType        string `json:"DeviceType"`
	Location          string `json:"Location"`
	Manufacturer      string `json:"Manufacturer"`
	PartNumber        string `json:"PartNumber"`
	SerialNumber      string `json:"SerialNumber"`
	FirmwareVersion   string `json:"FirmwareVersion"`
	ProductPartNumber string `json:"ProductPartNumber"`
	Status            status `json:"Status"`
}

type PortRecord struct {
	PhysicalPortNumber int      `json:"physicalPortNumber"`
	Type               string   `json:"type"`
	MAC                string   `json:"macAddress"`
	WWPN               string   `json:"wwn,omitempty"`
	VirtualMACs        []string `json:"virtualMacAddresses,omitempty"`
}

type FirmwareRecord struct {
	Name               string `json:"name,omitempty"`
	Version            string `json:"version,omitempty"`
	Location           string `json:"location,omitempty"`
	State              string `json:"state,omitempty"`
	Status             string `json:"status,omitempty"`
	ServerHardwareURI  string `json:"serverHardwareUri,omitempty"`
	ServerHardwareOrig string `json:"-"`
}

type EnclosureRecord struct {
	ID             string
	URI            string
	OriginalURI    string
	UUID           string
	Name           string
	SerialNumber   string
	EnclosureModel string
	EnclosureType  string
	RackName       string
	RackURI        string
	DCURI          string
	DeviceBayCount int
	DeviceBays     []deviceBay
}

type deviceBay struct {
	Type           string `json:"type"`
	BayNumber      int    `json:"bayNumber"`
	DevicePresence string `json:"devicePresence"`
	DeviceURI      string `json:"deviceUri,omitempty"`
	DeviceBayType  string `json:"deviceBayType"`
	BayPowerState  string `json:"bayPowerState"`
}

type RackRecord struct {
	ID           string
	URI          string
	OriginalURI  string
	UUID         string
	Name         string
	Model        string
	SerialNumber string
	UHeight      int
	DCURI        string
	Room         string
	RackMounts   []rackMount
}

type rackMount struct {
	MountURI string `json:"mountUri"`
	TopUSlot int    `json:"topUSlot"`
	UHeight  int    `json:"uHeight"`
	Location string `json:"location"`
}

type DatacenterRecord struct {
	URI             string
	OriginalURI     string
	UUID            string
	Name            string
	Room            string
	Status          string
	State           string
	CoolingCapacity int
	Width           int
	Height          int
	Currency        string
	Contents        []dcContent
	RackInventory   []rackInventoryEntry
}

type dcContent struct {
	ResourceURI string `json:"resourceUri"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

type rackInventoryEntry struct {
	Name        string `json:"name"`
	OriginalURI string `json:"originalUri"`
}

type status struct {
	Health string `json:"Health"`
	State  string `json:"State"`
}

type collection[T any] struct {
	Type        string `json:"type,omitempty"`
	URI         string `json:"uri,omitempty"`
	Category    string `json:"category,omitempty"`
	Count       int    `json:"count"`
	Total       int    `json:"total"`
	Members     []T    `json:"members"`
	NextPageURI string `json:"nextPageUri,omitempty"`
}

// New builds a simulator for one configured instance.
func New(cfg InstanceConfig) *Simulator {
	inv := GenerateInventory(cfg)
	return &Simulator{cfg: cfg, inventory: inv}
}

// ListenAndServe starts the HTTP listener (blocks).
func (s *Simulator) ListenAndServe() error {
	addr := s.cfg.Listen
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	srv := &http.Server{Addr: addr, Handler: mux}
	return srv.ListenAndServe()
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func uuidFor(prefix string, n int) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		hash32(prefix, n),
		(uint16(n)&0xffff)|0x4000,
		0x8000|uint16(n>>16)&0x0fff,
		uint16(n>>8)&0xffff,
		uint64(n)*0x100000000+uint64(hash32(prefix, n^0x5f3759df)))
}

func hash32(prefix string, n int) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(prefix); i++ {
		h ^= uint32(prefix[i])
		h *= 16777619
	}
	h ^= uint32(n)
	h *= 16777619
	return h
}
