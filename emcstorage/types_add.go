package emcstorage

import (
	"time"
)

// AlertContent - содержимое сведений об активном алерте (alert)
type AlertContent struct {
	ID          string `json:"id"`
	Severity    int    `json:"severity"`
	Description string `json:"description"`
}

// Type declarations
type Alert struct {
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
		Content AlertContent `json:"content"`
	} `json:"entries"`
}

//Information about the events reported by the storage system.
//The system monitors and reports on a variety of system events. It collects the events and writes them to the user log, which contains a record for each event.
//The health and alert providers promote some events to be alerts, which are usually events that require attention from the system administrator. For information about alerts, see the Help topic for the alert
//In the username attribute, is the value N/A or blank if a user did not cause the event or the account is unavailable?

// NodeEnum - идентификатор storage processor'а, сгенерировавшего событие (см. константы NodeSPA/NodeSPB/...)
type NodeEnum int

const (
	NodeSPA = iota
	NodeSPB
	NodeUnknown
	NodeUncnown989 = 989
)

// SeverityEnum - уровень серьёзности события (см. константы SeverityEMERGENCY/.../SeverityOK)
type SeverityEnum int

const (
	SeverityEMERGENCY = iota
	SeverityALERT
	SeverityCRITICAL
	SeverityERROR
	SeverityWARNING
	SeverityNOTICE
	SeverityINFO
	SeverityDEBUG
	SeverityOK
)

// EventCategoryEnum - категория события (см. константы EventUser/EventAudit/EventAuthentication)
type EventCategoryEnum int

const (
	EventUser = iota
	EventAudit
	EventAuthentication
)

// EventContent
type EventContent struct {
	ID           string            `json:"id"`           //Unique identifier of the event instance.
	Node         NodeEnum          `json:"node"`         //Storage Processor that generated the event. 0 - SPA 1 -SPB 2, 989 Unknown LUNs may return Unknown for current node when in an intermediate or fail
	CreationTime time.Time         `json:"creationTime"` //Date and time when the event was created.
	Severity     SeverityEnum      `json:"severity"`     //Severity of the event.
	MessageID    string            `json:"messageId"`    //Identifier of the message, without arguments or localization.
	Arguments    []string          `json:"arguments"`    //Arguments in the event message.
	Message      string            `json:"message"`      //Localized description of the event's cause or effects.
	Username     string            `json:"username"`     //If a user caused the event, and the user account still exists, the user associated with the event.
	Category     EventCategoryEnum `json:"category"`     //Event category.
	Source       string            `json:"source"`       //System component that caused the event. This information is intended for service personnel.
}

// Event
type Event struct {
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
		Content EventContent `json:"content"`
	} `json:"entries"`
}

const (
	TechSAS         = 1  //
	TechNL_SAS      = 2  //
	TechSAS_FLASH_2 = 6  //
	TechSAS_FLASH_3 = 7  //
	TechSAS_FLASH_4 = 8  //
	TechSAS_FLASH_5 = 9  //
	TechMixed       = 50 //
	TechVrtual      = 99 //
)

// DiskContent - содержимое сведений о физическом диске (disk)
type DiskContent struct {
	ID               string `json:"id"`
	NeedsReplacement bool   `json:"needsReplacement"`
	SlotNumber       uint32 `json:"slotNumber"`
	EstimatedEOL     string `json:"estimatedEOL"`
	BusId            uint32 `json:"busId"`
	Name             string `json:"name"`
	Manufacturer     string `json:"manufacturer"`
	Model            string `json:"model"`
	Version          string `json:"version"`
	EmcPartNumber    string `json:"emcPartNumber"`
	EmcSerialNumber  string `json:"emcSerialNumber"`
	TierType         uint16 `json:"tierType"`
	CurrentSpeed     uint64 `json:"currentSpeed"` //bytes/sec
	MaxSpeed         uint64 `json:"maxSpeed"`
	IsInUse          bool   `json:"isInUse"`
	IsFastCacheInUse bool   `json:"isFastCacheInUse"`
	Size             uint64 `json:"size"` //bytes
	RawSize          uint64 `json:"rawSize"`
	VendorSize       uint64 `json:"vendorSize"`
	WWN              string `json:"wwn"`
	DiskTechnology   int    `json:"diskTechnology"`
}

// Disk - ответ API со списком физических дисков (disk)
type Disk struct {
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
		Content DiskContent `json:"content"`
	} `json:"entries"`
}

// HealthEnum value
const (
	HealthUNKNOWN        = 0  //Unknown.
	HealthOK             = 5  //Ok.
	HealthOKBUT          = 7  //Ok But Minor Warning
	HealthDEGRADED       = 10 //Degraded.
	HealthMINOR          = 15 //Minor Issue.
	HealthMAJOR          = 20 //Major Issue.
	HealthCRITICAL       = 25 //Critical Issue.
	HealthNONRECOVERABLE = 30 //Non Recoverable Issue.
)

// HealthInfo - сведения о состоянии здоровья ресурса СХД (общая структура, используется в разных сущностях)
type HealthInfo struct {
	Value          int      `json:"value"`          //Possible values for health attribute.
	DescriptionIds []string `json:"descriptionIds"` //Description ids.
	Descriptions   []string `json:"descriptions"`   //Descriptions are localized messages describing details of the operating condition.
	ResolutionIds  []string `json:"resolutionIds"`  //Resolution ids.
	Resolutions    []string `json:"resolutions"`    //Resolutions are URLs to get more information about this health state.
}

// ResourceRef - ссылка на связанный ресурс (тип и идентификатор)
type ResourceRef struct {
	ID       string `json:"id"`       //The id of the referenced instance.
	Resource string `json:"resource"` //The class name of the referenced instance.
}

// SystemContent - содержимое сведений о системе СХД в целом (модель, серийный номер, платформа)
type SystemContent struct {
	ID                     string     `json:"id"`                     //Unique identifier of the storageProcessor instance.
	Health                 HealthInfo `json:"health"`                 //Health
	Name                   string     `json:"name"`                   //System name.
	Model                  string     `json:"model"`                  //System model name.
	SerialNumber           string     `json:"serialNumber"`           //System product serial number.
	UUIDBase               uint32     `json:"uuidBase"`               //Base value used to generate UUIDs in the host environment(e.g. OVMS host).
	InternalModel          string     `json:"internalModel"`          //Internal model name for the system.
	Platform               string     `json:"platform"`               //Hardware platform for the system.
	IsAllFlash             bool       `json:"isAllFlash"`             //Indicates whether sytem is all flash.
	MACAddress             string     `json:"macAddress"`             //MAC address of the management interface.
	IsEULAAccepted         bool       `json:"isEULAAccepted"`         //Indicates whether the End User License Agreement (EULA) was accepted for an upgrade.
	IsUpgradeComplete      bool       `json:"isUpgradeComplete"`      //Indicates whether an upgrade completed. true - Upgrade completed.
	IsAutoFailbackEnabled  bool       `json:"isAutoFailbackEnabled"`  //Indicates whether the automatic failback of NAS servers is enabled for the system. Values are: true - Automatic failback for NAS serv
	CurrentPower           uint32     `json:"currentPower"`           //Current amount of power used by the system.
	AvgPower               uint32     `json:"avgPower"`               //Average amount of power used by the system. The system uses a one hour window of 30-second samples to determine this value.
	SupportedUpgradeModels []int      `json:"supportedUpgradeModels"` //List of all supported models for hardware upgrade.
}

// DAEContent - содержимое сведений о дисковой полке DAE
type DAEContent struct {
	ID                 string        `json:"id"`                 //Unique identifier of the dae instance.
	Health             HealthInfo    `json:"health"`             //Health
	NeedsReplacement   bool          `json:"needReplacement"`    //Indicates whether the DAE needs replacement. Values are: true - DAE needs replacement. false - DAE does not need replacement.
	Parent             ResourceRef   `json:"parent"`             //Resource type and unique identifier of the DAE's parent enclosure.
	SlotNumber         uint32        `json:"slotNumber"`         //The enclosure number or position where the DAE is located on the bus.
	Name               string        `json:"name"`               //DAE name.
	Manufacturer       string        `json:"manufacturer"`       //Manufacturer of the DAE.
	Model              string        `json:"model"`              //Manufacturer's model number for the DAE.
	EmcPartNumber      string        `json:"emcPartNumber"`      //EMC part number for the DAE.
	EmcSerialNumber    string        `json:"emcSerialNumber"`    //EMC serial number for the DAE.
	VendorPartNumber   string        `json:"vendorPartNumber"`   //Vendor part number for the DAE.
	VendorSerialNumber string        `json:"vendorSerialNumber"` //Vendor serial number for the DAE.
	EnclosureType      int           `json:"enclosureType"`      //DAE enclosure type.
	BusID              uint32        `json:"busId"`              //Identifier of the bus used by the DAE.
	DriveTypes         []int         `json:"driveTypes"`         //Disk type of the disks in the DAE.
	CurrentPower       uint32        `json:"currentPower"`       //Current amount of power used by the DAE.
	AvgPower           uint32        `json:"avgPower"`           //Average amount of power used by the DAE. The system uses a one hour window of 30-second samples to determine this value.
	MaxPower           uint32        `json:"maxPower"`           //Maximum amount of power used by the DAE. The system uses a one hour window of 30-second samples to determine this value.
	CurrentTemperature uint32        `json:"currentTemperature"` //Current temperature of the DAE.
	AvgTemperature     uint32        `json:"avgTemperature"`     //Average temperature of the DAE. The system uses a one hour window of 30-second samples to determine this value.
	MaxTemperature     uint32        `json:"maxTemperature"`     //Maximum temperature of the DAE. The system uses a one hour window of 30-second samples to determine this value.
	CurrentSpeed       uint64        `json:"currentSpeed"`       //Current speed of the DAE SAS disk connection.
	MaxSpeed           uint64        `json:"maxSpeed"`           //Maximum speed of the DAE SAS disk connection. The system uses a one hour window of 30-second samples to determine this value.
	ParentSystem       SystemContent `json:"parentSystem"`       //Parent system of the Disk Array Enclosure (DAE).
}

// DPEContent - содержимое сведений о базовом блоке DPE
type DPEContent struct {
	ID                 string        `json:"id"`                 //Unique identifier of the storageProcessor instance.
	Health             HealthInfo    `json:"health"`             //Health
	NeedsReplacement   bool          `json:"needsReplacement"`   //Indicates whether the SP needs replacement. Values are: true - SP needs replacement. false - SP does not need replacement.
	Parent             ResourceRef   `json:"parent"`             //Resource type and unique identifier for the SP's parent enclosure.
	SlotNumber         uint32        `json:"slotNumber"`         //Slot where the SP is located in the parent enclosure.
	Name               string        `json:"name"`               //SP name.
	Manufacturer       string        `json:"manufacturer"`       //Manufacturer of the SP.
	Model              string        `json:"model"`              //Manufacturer's model number for the SP.
	EmcPartNumber      string        `json:"emcPartNumber"`      //EMC part number for the SP.
	EmcSerialNumber    string        `json:"emcSerialNumber"`    //EMC serial number for the SP.
	VendorPartNumber   string        `json:"vendorPartNumber"`   //Vendor part number for the SP.
	VendorSerialNumber string        `json:"vendorSerialNumber"` //Vendor serial number for the SP.
	EnclosureType      int           `json:"enclosureType"`      //DPE enclosure type.
	BusID              uint32        `json:"busId"`              //Identifier of the SAS bus used by the DPE.
	DriveTypes         []int         `json:"driveTypes"`         //Disk type of the disks in the DPE.
	CurrentPower       uint32        `json:"currentPower"`       //Current amount of power used by the DPE.
	AvgPower           uint32        `json:"avgPower"`           //Average amount of power used by the DPE. The system uses a one hour window of 30-second samples to determine this value.
	MaxPower           uint32        `json:"maxPower"`           //Maximum amout of power used by the DPE. The system uses a one hour window of 30-second samples to determine this value.
	CurrentTemperature uint32        `json:"currentTemperature"` //Current temperature of the DPE.
	AvgTemperature     uint32        `json:"avgTemperature"`     //Average temperature of the DPE. The system uses a one hour window of 30-second samples to determine this value.
	MaxTemperature     uint32        `json:"maxTemperature"`     //Maximum temperature of the DPE. The system uses a one hour window of 30-second samples to determine this value.
	CurrentSpeed       uint64        `json:"currentSpeed"`       //Current speed of the DPE SAS disk connection.
	MaxSpeed           uint64        `json:"maxSpeed"`           //Maximum speed of the DPE SAS disk connection. The system uses a one hour window of 30-second samples to determine this value.
	ParentSystem       SystemContent `json:"parentSystem"`       //Parent system of the Disk Processor Enclosure (DPE).
}

// StorageProcessorContent - содержимое сведений о storage processor'е СХД
type StorageProcessorContent struct {
	ID                   string      `json:"id"`                   //Unique identifier of the storageProcessor instance.
	Parent               ResourceRef `json:"parent"`               //Resource type and unique identifier for the SP's parent enclosure.
	Health               HealthInfo  `json:"health"`               //Health
	NeedsReplacement     bool        `json:"needsReplacement"`     //Indicates whether the SP needs replacement. Values are: true - SP needs replacement. false - SP does not need replacement.
	IsRescueMode         bool        `json:"isRescueMode"`         //Indicates whether the SP is in Service Mode. Values are: true - SP is in Service Mode. false - SP is not in Service Mode.
	Model                string      `json:"model"`                //Manufacturer's model number for the SP.
	SlotNumber           uint32      `json:"slotNumber"`           //Slot where the SP is located in the parent enclosure.
	Name                 string      `json:"name"`                 //SP name.
	EmcPartNumber        string      `json:"emcPartNumber"`        //EMC part number for the SP.
	EmcSerialNumber      string      `json:"emcSerialNumber"`      //EMC serial number for the SP.
	Manufacturer         string      `json:"manufacturer"`         //Manufacturer of the SP.
	VendorPartNumber     string      `json:"vendorPartNumber"`     //Vendor part number for the SP.
	VendorSerialNumber   string      `json:"vendorSerialNumber"`   //Vendor serial number for the SP.
	SasExpanderVersion   string      `json:"sasExpanderVersion"`   //Version number of the SAS Expander associated with the SP.
	BiosFirmwareRevision string      `json:"biosFirmwareRevision"` //Version number of the SP BIOS.
	PostFirmwareRevision string      `json:"postFirmwareRevision"` //Version number of the SP Power-On Self-Test software.
	MemorySize           uint32      `json:"memorySize"`           //SP RAM size.
	ParentDpe            DPEContent  `json:"parentDpe"`            //
	ModelName            int         `json:"modelName"`            //The physical model name of storageProcessor
}

// DiskGroupContent - содержимое сведений о группе дисков (tier-группе)
type DiskGroupContent struct {
	ID                             string `json:"id"`                             //Unique identifier of the diskGroup instance.
	Name                           string `json:"name"`                           //Disk group name.
	EmcPartNumber                  string `json:"emcPartNumber"`                  //EMC part number of the drives.
	TierType                       int    `json:"tierType"`                       //Drive type.
	DiskTechnology                 int    `json:"diskTechnology"`                 //Drive technology of the drives in the disk group. There are two hard drive technologies: SAS and NL-SAS, and flash drive
	IsFASTCacheAllowable           bool   `json:"isFASTCacheAllowable"`           //Indicates whether or not drives from this diskGroup are valid to use for FASTCache. Values are: true - Drives may be used
	DiskSize                       uint64 `json:"diskSize"`                       //The capacity of each drive.
	AdvertisedSize                 uint64 `json:"advertisedSize"`                 //Capacity written on the label of the drives in the disk group.
	RPM                            uint32 `json:"rpm"`                            //For non-Flash drives, the rotational speed in revolutions per minute (RPM).
	Speed                          uint64 `json:"speed"`                          //Interface speed of the drives in the disk group.
	TotalDisks                     uint32 `json:"totalDisks"`                     //Total number of drives in the disk group.
	MinHotSpareCandidates          uint32 `json:"minHotSpareCandidates"`          //Minimum number of hot spares required for traditional pools for the disk group. As a rule one Hot spare is required per 3
	HotSparePolicyStatus           int    `json:"hotSparePolicyStatus"`           //Status of the hot spare policy for the disk group.
	UnconfiguredDisks              uint32 `json:"unconfiguredDisks"`              //The number of unconfigured drives in disk group. Determined by subtracting the number of drives in use by pools, FAST Cac
	ConfiguredTraditionalPoolDisks uint32 `json:"configuredTraditionalPoolDisks"` //The number of configured drives used by traditional pools.
	ConfiguredFastCacheDisks       uint32 `json:"configuredFastCacheDisks"`       //The number of configured drives used by FAST Cache.
	DisksPastEOL                   uint32 `json:"disksPastEOL"`                   //The number of drives past EOL (end of life).
	DisksWithEOLLess30Days         uint32 `json:"disksWithEOLLess30Days"`         //The number of drives in disk group with EOL (end of life) less than 30 days.
	DisksWithEOLLess60Days         uint32 `json:"disksWithEOLLess60Days"`         //The number of drives in disk group with EOL (end of life) less than 60 days.
	DisksWithEOLLess90Days         uint32 `json:"disksWithEOLLess90Days"`         //The number of drives in disk group with EOL (end of life) less than 90 days.
	DisksWithEOLLess180Days        uint32 `json:"disksWithEOLLess180Days"`        //The number of drives in disk group with EOL (end of life) less than 180 days.
}

// HostId - ссылка на хост по идентификатору
type HostId struct {
	ID string `json:"id"`
}

// StorageResourceId - ссылка на ресурс хранения по идентификатору
type StorageResourceId struct {
	ID string `json:"id"`
}

// BlockHostAccess - права доступа хоста к блочному ресурсу (LUN)
type BlockHostAccess struct {
	Host             HostId `json:"host"`
	AccessMask       int    `json:"accessMask"`
	ProductionAccess int    `json:"productionAccess"`
	SnapshotAccess   int    `json:"snapshotAccess"`
	Hlu              int    `json:"hlu"`
}

// PoolId - ссылка на пул хранения по идентификатору
type PoolId struct {
	ID string `json:"id"`
}

// FamilyBaseLunId - ссылка на базовый LUN семейства (клоны/снапшоты) по идентификатору
type FamilyBaseLunId struct {
	ID string `json:"id"`
}

// LUNContent - содержимое подробных сведений о LUN
type LUNContent struct {
	ID                       string            `json:"id"`                       //Unique identifier of the LUN.
	Type                     int               `json:"type"`                     //Type of the LUN.
	DefaultNode              int               `json:"defaultNode"`              //
	CurrentNode              int               `json:"currentNode"`              //
	MigrationState           int               `json:"migrationState"`           //
	Health                   HealthInfo        `json:"health"`                   //Health information for the LUN, as defined by the health resource type.
	Name                     string            `json:"name"`                     //Name of the LUN
	Description              string            `json:"description"`              //Description of the LUN
	SizeTotal                int64             `json:"sizeTotal"`                //LUN size that the system presents to the host or end user
	SizeAllocated            int64             `json:"sizeAllocated"`            //
	SizePreallocated         int               `json:"sizePreallocated"`         //
	SizeAllocatedTotal       int64             `json:"sizeAllocatedTotal"`       //
	DataReductionSizeSaved   int               `json:"dataReductionSizeSaved"`   //
	DataReductionPercent     int               `json:"dataReductionPercent"`     //
	DataReductionRatio       float64           `json:"dataReductionRatio"`       //
	PerTierSizeUsed          []int64           `json:"perTierSizeUsed"`          //Sizes of space allocations by the LUN on the tiers of multi-tier storage pool. This list will have the same length as the tiers l
	IsThinEnabled            bool              `json:"isThinEnabled"`            //
	IsDataReductionEnabled   bool              `json:"isDataReductionEnabled"`   //
	IsAdvancedDedupEnabled   bool              `json:"isAdvancedDedupEnabled"`   //
	Wwn                      string            `json:"wwn"`                      //
	IsReplicationDestination bool              `json:"isReplicationDestination"` //
	MetadataSize             int64             `json:"metadataSize"`             //
	MetadataSizeAllocated    int64             `json:"metadataSizeAllocated"`    //
	SnapWwn                  string            `json:"snapWwn"`                  //
	SnapsSize                int               `json:"snapsSize"`                //
	SnapsSizeAllocated       int               `json:"snapsSizeAllocated"`       //
	//	HostAccess             BlockHostAccess     `json:"hostAccess"`//
	SnapCount                int               `json:"snapCount"`                //
	IsThinClone              bool              `json:"isThinClone"`              //
	FamilySnapCount          int               `json:"familySnapCount"`          //
	FamilyCloneCount         int               `json:"familyCloneCount"`         //
	NonBaseSizeAllocated     int               `json:"nonBaseSizeAllocated"`     //
	FamilySizeAllocatedTotal int64             `json:"familySizeAllocatedTotal"` //
	NonBaseSize              int               `json:"nonBaseSize"`              //
	StorageResource          StorageResourceId `json:"storageResource"`          //
	Pool                     PoolId            `json:"pool"`                     //
	FamilyBaseLun            FamilyBaseLunId   `json:"familyBaseLun"`            //
}

// LUNContentShort - краткое содержимое сведений о LUN (используется во вложенных ссылках, напр. HostLUNID)
type LUNContentShort struct {
	ID                     string     `json:"id"`                     //Unique identifier of the LUN.
	Health                 HealthInfo `json:"health"`                 //Health information for the LUN, as defined by the health resource type.
	Name                   string     `json:"name"`                   //Name of the LUN.
	Description            string     `json:"description"`            //Description of the LUN.
	Types                  int        `json:"type"`                   //Type of the LUN.
	SizeTotal              uint64     `json:"sizeTotal"`              //LUN size that the system presents to the host or end user.
	SizeUsed               uint64     `json:"sizeUsed"`               //Used size is not applicable to LUN and this value is not set.
	SizeAllocated          uint64     `json:"sizeAllocated"`          //Size of space actually allocated in the pool for the LUN: For thin-provisioned LUNs this as a rule is less than the sizeTotal attribu
	SizePreallocated       uint64     `json:"sizePreallocated"`       //Space reserved from the pool by the LUN, for future needs to make writes more efficient. The pool may be able to reclaim some of this
	SizeAllocatedTotal     uint64     `json:"sizeAllocatedTotal"`     //Total pool space used in the pool for the LUN.
	DataReductionSizeSaved uint64     `json:"dataReductionSizeSaved"` //Storage element saved space by data reduction, which includes savings from compression, deduplication and advanced deduplication.
	DataReductionPercent   uint64     `json:"dataReductionPercent"`   //Data reduction percentage is the percentage of the data that does not consume storage - the savings due to data reduction. For exampl
	DataReductionRatio     float64    `json:"dataReductionRatio"`     //Data reduction ratio. The data reduction ratio is the ratio between the size of the data and the amount of storage actually consumed.
	PerTierSizeUsed        []int64    `json:"perTierSizeUsed"`        //Sizes of space allocations by the LUN on the tiers of multi-tier storage pool. This list will have the same length as the tiers list
	IsThinEnabled          bool       `json:"isThinEnabled"`          //Indicates whether thin provisioning is enabled. true - The LUN is thin provisioned. false - The LUN is not thin provisioned.
	IsDataReductionEnabled bool       `json:"isDataReductionEnabled"` //True if data reduction is enabled.
	IsAdvancedDedupEnabled bool       `json:"isAdvancedDedupEnabled"` //Indicates whether to enable advanced deduplication for the LUN. It can only be enabled on the all flash high end platforms and only w
}

// HostContainerID (Applies to VCenter server and ESXi host configurations only.) Identifier of the parent host container, as defined by the hostContainer resource type.
type HostContainerID struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HostInitiatorID Information about host initiators in the storage system.
// After you create a host configuration for controlling host access to storage on the system, you need to create a host initiator for each host configuration that accesses iSCSI or Fibre Channel (FC) storage.
// Initiator-based registration allows you to register the initiator to a host without having to manage individual paths. By default, when an initiator is registered to a host, all of its paths, including thos
type HostInitiatorID struct {
	ID          string `json:"id"`
	InitiatorID string `json:"initiatorId"`
}

// HostIPPortID Information about the host's IP settings.
type HostIPPortID struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StorageResourceID Information about storage resources in the storage system.
type StorageResourceID struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HostLUNID Information about the LUNs and LUN snapshots to which a host has access.
type HostLUNID struct {
	ID  string          `json:"id"`
	LUN LUNContentShort `json:"lun"`
}

// DatastoreID Represents a VMware Datastore that is discovered via vCenter or ESXi.
type DatastoreID struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HostVVoldatastoreID Information about the VVoldatastore to which a host has access.
type HostVVoldatastoreID struct {
	ID string `json:"id"`
}

// VMID Represents a virtual machine.
type VMID struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	GuestHostName string `json:"guestHostName"`
}

// TenantID A tenant is a representation of a datacenter client, who uses an independent network structure and gets the independent and exclusive access to certain storage resources. On the storage array, it c
type TenantID struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HostContent - содержимое подробных сведений о хосте, подключённом к СХД
type HostContent struct {
	ID                  string                `json:"id"`                  //Unique identifier of the host instance.
	Health              HealthInfo            `json:"health"`              //Health information for the host, as defined by the health resource type.
	Name                string                `json:"name"`                //Host name: For an automatically-managed ESXi server through vCenter, this is the display name in the vCenter GUI. For a manually
	Description         string                `json:"description"`         //Host description.
	Type                int                   `json:"type"`                //Type of host configuration.
	OSType              string                `json:"osType"`              //Operating system running on the host.
	HostUUID            string                `json:"hostUUID"`            //(Applies to vCenter server and ESXi host configurations only.) Universal Unique Identifier (UUID) of the host.
	HostPushedUUID      string                `json:"hostPushedUUID"`      //(Applies to vCenter server and ESXi host configurations only.) Pushed UUID of the host.
	HostPolledUUID      string                `json:"hostPolledUUID"`      //(Applies to vCenter server and ESXi host configurations only.) Polled UUID of the host.
	LastPollTime        time.Time             `json:"lastPollTime"`        //(Applies to hosts on ESXi servers only.) Indicates the date and time when the storage array received the host configuration data
	AutoManageType      int                   `json:"autoManageType"`      //Indicates how the host is managed.
	RegistrationType    int                   `json:"registrationType"`    //Indicates how initiators are registered to the host.
	HostContainer       HostContainerID       `json:"hostContainer"`       //(Applies to vCenter server and ESXi host configurations only.) Identifier of the parent host container, as defined by the hostCo
	FcHostInitiators    []HostInitiatorID     `json:"fcHostInitiators"`    //Identifiers of the Fibre Channel initiators associated with the host, as defined by the hostInitiator resource type.
	IscsiHostInitiators []HostInitiatorID     `json:"iscsiHostInitiators"` //Identifiers of the iSCSI initiators associated with the host, as defined by the hostInitiator resource type.
	HostIPPorts         []HostIPPortID        `json:"hostIPPorts"`         //Identifiers of the network ports associated with the host, as defined by the hostIPPort resource type.
	StorageResources    []StorageResourceID   `json:"storageResources"`    //Identifiers of the storage resources used by the host, as defined by the storageResource resource type.
	HostLUNs            []HostLUNID           `json:"hostLUNs"`            //(Applies to iSCSI and Fibre Channel host configurations only.) Identifiers of the LUNs accessible by the host, as defined by the
	IsNfsv4Supported    bool                  `json:"isNfsv4Supported"`    //(Applies to vCenter server and ESXi host configurations only.) Flag indicating whether ESXi host supports NFSv4.
	NfsUsername         string                `json:"nfsUsername"`         //(Applies to vCenter server and ESXi host configurations only.) NFS user authentication information configured on the ESXi host.
	Tenant              TenantID              `json:"tenant"`              //Information about the tenant to which the host is assigned.
	Datastores          []DatastoreID         `json:"datastores"`          //The datastores associated with the host.
	HostVVoldatastore   []HostVVoldatastoreID `json:"hostVVoldatastore"`   //The hostVVolDatastores associated with the host.
	VMs                 []VMID                `json:"vms"`                 //The vms associated with the host.
}

// Host - ответ API со списком хостов, подключённых к СХД (host)
type Host struct {
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
		Content HostContent `json:"content"`
	} `json:"entries"`
}

// DiskGroup - ответ API со списком групп дисков (diskGroup)
type DiskGroup struct {
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
		Content DiskGroupContent `json:"content"`
	} `json:"entries"`
}

// StorageProcessor - ответ API со списком storage processor'ов (storageProcessor)
type StorageProcessor struct {
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
		Content StorageProcessorContent `json:"content"`
	} `json:"entries"`
}

// DAE - ответ API со списком дисковых полок (dae)
type DAE struct {
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
		Content DAEContent `json:"content"`
	} `json:"entries"`
}

// DPE - ответ API со списком базовых блоков (dpe)
type DPE struct {
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
		Content DPEContent `json:"content"`
	} `json:"entries"`
}

// Lun - ответ API со списком LUN (lun)
type Lun struct {
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
		Content LUNContent `json:"content"`
	} `json:"entries"`
}
