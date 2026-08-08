package emcstorage

import (
	"time"
)

// Type declarations

// DNSServer struct
type DNSServer struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []struct {
		Content struct {
			Origin    int      `json:"origin"`
			ID        string   `json:"id"`
			Addresses []string `json:"addresses"`
		} `json:"content"`
	} `json:"entries"`
}

// MetricRealTimeQueryResponse struct
type MetricRealTimeQueryResponse struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Content struct {
		Interval       int       `json:"interval"`
		ID             int       `json:"id"`
		Paths          []string  `json:"paths"`
		MaximumSamples int       `json:"maximumSamples"`
		Expiration     time.Time `json:"expiration"`
	} `json:"content"`
}

type MetricValueResponseContent struct {
	Interval  int         `json:"interval"`
	Path      string      `json:"path"`
	Timestamp time.Time   `json:"timestamp"`
	Values    interface{} `json:"values"`
}

// MetricValueResponse struct
type MetricValueResponse struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []struct {
		Base    string    `json:"@base"`
		Updated time.Time `json:"updated"`
		Links   []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
		Content MetricValueResponseContent `json:"content"`
	} `json:"entries"`
}

type KpiValueResponseContent struct {
	Path      string                `json:"path"`
	StartTime time.Time             `json:"startTime"`
	EndTime   time.Time             `json:"endTime"`
	Interval  int                   `json:"interval"`
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Values    map[time.Time]float64 `json:"values"`
}

// KpiValueResponse struct
type KpiValueResponse struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []struct {
		Content KpiValueResponseContent `json:"content"`
	} `json:"entries"`
}

type MetricRealTimeQueryResultContent struct {
	QueryID   int         `json:"queryId"`
	Path      string      `json:"path"`
	Timestamp time.Time   `json:"timestamp"`
	Values    interface{} `json:"values"`
}

// MetricRealTimeQueryResult struct
type MetricRealTimeQueryResult struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []struct {
		Base    string    `json:"@base"`
		Updated time.Time `json:"updated"`
		Links   []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
		Content MetricRealTimeQueryResultContent `json:"content"`
	} `json:"entries"`
}

type BasicSystemInfoContent struct {
	ID                 string `json:"id"`                 // Unique identifier of the basicSystemInfo instance.
	Model              string `json:"model"`              // Model name of this storage system. This value comes from the model attribute of the system resource.
	Name               string `json:"name"`               // Name of this storage system. This value comes from the name attribute of the system resource.
	SoftwareVersion    string `json:"softwareVersion"`    // Software version of this storage system. This value comes from the version attribute of the installedSoftwareVersion resource.
	APIVersion         string `json:"apiVersion"`         // Latest version of the REST API that this storage system supports.
	EarliestAPIVersion string `json:"earliestApiVersion"` // Earliest version of the REST API that this storage system supports.
}

// BasicSystemInfo struct
type BasicSystemInfo struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []struct {
		Base    string    `json:"@base"`
		Updated time.Time `json:"updated"`
		Links   []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
		Content BasicSystemInfoContent `json:"content"`
	} `json:"entries"`
}

type PoolContent struct {
	ID             string `json:"id"`             //Unique identifier of the pool instance.
	Name           string `json:"name"`           //Pool name, unique in the storage system.
	SizeFree       uint64 `json:"sizeFree"`       //Size of free space available in the pool.
	SizeTotal      uint64 `json:"sizeTotal"`      //The total size of space from the pool, which will be the sum of sizeFree, sizeUsed and sizePreallocated space.
	SizeUsed       uint64 `json:"sizeUsed"`       //Size allocated from the pool by storage resources, used for storing data. This will be the sum of the sizeAllocated values of each storage resource in t
	SizeSubscribed uint64 `json:"sizeSubscribed"` //Size of space requested by the storage resources allocated in the pool for possible future allocations. If this value is greater than the total size of t
}

// Pool struct
type Pool struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []struct {
		Base    string    `json:"@base"`
		Updated time.Time `json:"updated"`
		Links   []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
		Content PoolContent `json:"content"`
	} `json:"entries"`
}

type storageResourceTypeEnum int

const (
	filesystem storageResourceTypeEnum = iota + 1
	consistencyGroup
	vmwarefs
	vmwareiscsi
	lun
	vVolDatastoreFS
	vVolDatastoreISCSI
)

type StorageResourceContent struct {
	ID            string `json:"id"`            //Unique identifier of the storage resource.
	Name          string `json:"name"`          //Name of the storage resource.
	SizeAllocated uint64 `json:"sizeAllocated"` //Size of space actually allocated in the pool for the storage resource: For all thin-provisioned resources, this can be less than the sizeTotal attribute. F
	SizeTotal     uint64 `json:"sizeTotal"`     //Storage resource size that the system presents to the host or end user.
	SizeUsed      uint64 `json:"sizeUsed"`      //Size of the storage resource space consumed by the host. Applicable only for file system based storage resource. Indicates the size of file system space oc
	Type          int    `json:"type"`          //Storage resource type.
}

// StorageResource struct
type StorageResource struct {
	Base    string    `json:"@base"`
	Updated time.Time `json:"updated"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
	Entries []struct {
		Base    string    `json:"@base"`
		Updated time.Time `json:"updated"`
		Links   []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
		Content StorageResourceContent `json:"content"`
	} `json:"entries"`
}
