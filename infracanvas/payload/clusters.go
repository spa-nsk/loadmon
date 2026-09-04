package payload

// ClusterMapPage — карта кластеров vSphere (страница 4).
type ClusterMapPage struct {
	Vcenters []VcenterGroup `json:"vcenters"`
	Stats    ClusterStats   `json:"stats"`
}

type ClusterStats struct {
	VcenterCount    int `json:"vcenterCount"`
	DatacenterCount int `json:"datacenterCount"`
	ClusterCount    int `json:"clusterCount"`
	HostCount       int `json:"hostCount"`
	VMCount         int `json:"vmCount"`
}

type VcenterGroup struct {
	Name        string             `json:"name"`
	Datacenters []DatacenterGroup  `json:"datacenters"`
}

type DatacenterGroup struct {
	Name     string        `json:"name"`
	Clusters []ClusterNode `json:"clusters"`
}

type ClusterNode struct {
	Name      string `json:"name"`
	VMCount   int    `json:"vmCount"`
	HostCount int    `json:"hostCount"`
	TypeWork  int    `json:"typeWork"`
}
