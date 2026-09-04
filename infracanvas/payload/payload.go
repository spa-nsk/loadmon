// Package payload описывает JSON-структуры для canvas-визуализации инфраструктуры
// (общие для Beego-контроллера и WASM-модуля infracanvas).
package payload

// CanvasData — корневой объект, передаваемый в браузер и WASM-рендерер.
type CanvasData struct {
	Pages          []PageMeta     `json:"pages"`
	Hardware       HardwarePage   `json:"hardware"`
	Virtualization VirtualPage    `json:"virtualization"`
	Monitoring     MonitoringPage `json:"monitoring"`
}

type PageMeta struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type HardwarePage struct {
	Racks   []Rack    `json:"racks"`
	Storage []Storage `json:"storage"`
	Stats   HWStats   `json:"stats"`
}

type HWStats struct {
	BladeChassis int `json:"bladeChassis"`
	BladeServers int `json:"bladeServers"`
	RackServers  int `json:"rackServers"`
	StorageUnits int `json:"storageUnits"`
}

type Rack struct {
	Room         string         `json:"room"`
	RackNum      string         `json:"rackNum"`
	Label        string         `json:"label"`
	BladeChassis []BladeChassis `json:"bladeChassis"`
	RackServers  []RackServer   `json:"rackServers"`
}

type BladeChassis struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Model     string      `json:"model"`
	SerialNum string      `json:"serialNum"`
	SlotCount int         `json:"slotCount"`
	StartUnit int         `json:"startUnit"`
	SizeUnit  int         `json:"sizeUnit"`
	Slots     []BladeSlot `json:"slots"`
}

type BladeSlot struct {
	Num        int    `json:"num"`
	Occupied   bool   `json:"occupied"`
	ServerName string `json:"serverName"`
	Model      string `json:"model"`
	PowerOn    bool   `json:"powerOn"`
}

type RackServer struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	SerialNum  string `json:"serialNum"`
	StartUnit  int    `json:"startUnit"`
	SizeUnit   int    `json:"sizeUnit"`
	PowerOn    bool   `json:"powerOn"`
}

type Storage struct {
	ShortName   string  `json:"shortName"`
	Vendor      string  `json:"vendor"`
	Model       string  `json:"model"`
	DiskCount   int     `json:"diskCount"`
	DiskMax     int     `json:"diskMax"`
	TotalSizeTB float64 `json:"totalSizeTb"`
	UsedPercent float64 `json:"usedPercent"`
	Room        string  `json:"room"`
	RackNum     string  `json:"rackNum"`
	PowerOn     bool    `json:"powerOn"`
}

type VirtualPage struct {
	VMCount         int      `json:"vmCount"`
	HypervisorCount int      `json:"hypervisorCount"`
	ClusterCount    int      `json:"clusterCount"`
	TopClusters     []Named  `json:"topClusters"`
}

type MonitoringPage struct {
	ZabbixHostCount int     `json:"zabbixHostCount"`
	LinkedServers   int     `json:"linkedServers"`
	StorageOnline   int     `json:"storageOnline"`
	StorageOffline  int     `json:"storageOffline"`
	AvgUsedPercent  float64 `json:"avgUsedPercent"`
}

type Named struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
