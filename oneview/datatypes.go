package oneview

import (
	"time"
	"github.com/HewlettPackard/oneview-golang/ov"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type ovEndpoint struct {
	Domain   string
	Login    string
	Password string
	Endpoint string
	Version  int
}

// ServerHardware - структура описывающая сервер в OneView
type ServerHardware struct {
	Base            ov.ServerHardware          `csv:"Base"`
	OperatingSystem string                     `csv:"OperatingSystem"` // тип ОС сервера (ov.ServerHardware такого поля не содержит)
	Memory          ServerHardwareMemory       `csv:"Memory"`
	Storage         ServerHardwareLocalStorage `csv:"Storage"`
	Datacenter      DatacenterInfo             `csv:"Datacenter"`
	Rack            Racks                      `csv:"Rack"`
	EnclosureInfo   Enclosure                  `csv:"Enclosure"`
	BayInfo         DeviceBayInfo              `csv:"BayInfo"`
}

// OVInfrastructure - структура описывающая обьекты OneView
//
//proteus:generate
type OVInfrastructure struct {
	LockFlag               int32 // 0 - не заблокировано 1 - заблокировано
	Endpoints                []*ovEndpoint
	Servers                  []*ServerHardware
	ServersCount             int
	AllDatacenters           DatacentersList
	DataCenters              map[string]*DatacenterInfo
	AllRacks                 map[string]*Racks
	ALLRacksDeviceTopology   map[string]*RacksDeviceTopology
	AllDeviceInfo            map[string]*DeviceInfo
	AllEnclosures            map[string]*Enclosure
	AllEnclosureBySN         map[string]*ov.Enclosure
	AllServerHardware        map[string]*ServerHardware
	AllDeviceBay             map[string]*DeviceBayInfo
	ServerBySN               map[string]*ServerHardware
	ServerByVirtualSN        map[string]*ServerHardware
	ServerByName             cmap.ConcurrentMap[string, *ServerHardware]
}

// MemoryLocation - расположение можуля памяти Socket - номер CPU, Slot - номер слота для данного CPU
type MemoryLocation struct {
	Slot   int `json:"Slot"`
	Socket int `json:"Socket"`
}

// HPE - структура показывающая состояние модуля памяти DIMMStatus в нормальном состоянии в значении "GoodInUse" MinimumVoltageVoltsX10 напряжение питания в вольтах
type HPE struct {
	DIMMStatus              string `json:"DIMMStatus"`
	MinimumVoltageVoltsX10 int    `json:"MinimumVoltageVoltsX10"`
}

// OemMemory - данные по производителям, в данном случае HPE
type OemMemory struct {
	Hpe HPE `json:"Hpe"`
}

// MemoryStatus - состояние модуля памяти Health в значении "OK", State в значении "Enabled"
type MemoryStatus struct {
	Health string `json:"Health"`
	State  string `json:"State"`
}

// MemoryModule - описание модуля памяти
type MemoryModule struct {
	BaseModuleType   string       `json:"BaseModuleType"`   //"RDIMM","UDIMM","SO_DIMM","LRDIMM","Mini_RDIMM","Mini_UDIMM","SO_RDIMM_72b","SO_UDIMM_72b","SO_DIMM_16b","SO_DIMM_32b"
	CapacityMiB      int          `json:"CapacityMiB"`      //емкость модуля в мегабайтах
	DeviceLocator    string       `json:"DeviceLocator"`    // "PROC1 DIMM 1" дублирует MemoryLocation в виде строки
	Manufacturer     string       `json:"Manufacturer"`     //производитель модуля памяти "HPE"
	MemoryDeviceType string       `json:"MemoryDeviceType"` //тип памяти "DDR","DDR2","DDR3","DDR4","DDR4_SDRAM","DDR4E_SDRAM","LPDDR4_SDRAM","DDR3_SDRAM","LPDDR3_SDRAM","DDR2_SDRAM","DDR2_SDRAM_FB_DIM
	MemoryLocation   MemoryLocation `json:"MemoryLocation"`   //расположение модуля памяти
	Name             string       `json:"Name"`              // "proc1dimm1" наименование модуля
	Oem              OemMemory    `json:"Oem"`                //дополнения от производителя
	OperatingSpeedMhz int         `json:"OperatingSpeedMhz"` //частота работы модуля памяти в мегагерцах
	PartNumber       string       `json:"PartNumber"`        //номер производителя
	RankCount        int          `json:"RankCount"`         //ранг модуля памяти
	Status           MemoryStatus `json:"Status"`             //состояние модуля памяти
}

// ServerHardwareMemory - структура возвращаемая при запросе, содержит количество и описание модулей памяти сервера.
type ServerHardwareMemory struct {
	URI            string         `json:"uri"`            //uri запроса
	CollectionState string        `json:"collectionState"` //состояние "Collected"
	Modified       time.Time      `json:"modified"`
	Data           []MemoryModule `json:"data"` //срез с данными модулей данные только по занятым слотам!!!
	Etag           string         `json:"etag"`
	Count          int            `json:"count"` //количество модулей
	Name           string         `json:"name"`  //"Memory"
}

// ServerHardwareMemory300 - структура ответа Redfish (endpoint.Version == 300) с данными о модулях памяти сервера
type ServerHardwareMemory300 struct {
	URI             string    `json:"uri"`
	CollectionState string    `json:"collectionState"`
	Modified        time.Time `json:"modified"`
	Data            []struct {
		OdataEtag        string `json:"@odata.etag"`
		OdataType        string `json:"@odata.type"`
		BaseModuleType   string `json:"BaseModuleType"`
		BusWidthBits     int    `json:"BusWidthBits"`
		CapacityMiB      int    `json:"CapacityMiB"`
		DataWidthBits    int    `json:"DataWidthBits"`
		DeviceLocator    string `json:"DeviceLocator"`
		ErrorCorrection  string `json:"ErrorCorrection"`
		MemoryDeviceType string `json:"MemoryDeviceType"`
		MemoryLocation   struct {
			Channel           int `json:"Channel"`
			MemoryController int `json:"MemoryController"`
			Slot              int `json:"Slot"`
			Socket            int `json:"Socket"`
		} `json:"MemoryLocation"`
		MemoryMedia []string `json:"MemoryMedia"`
		MemoryType  string   `json:"MemoryType"`
		Name        string   `json:"Name"`
		Oem         struct {
			Hpe struct {
				OdataType                   string   `json:"@odata.type"`
				Attributes                   []string `json:"Attributes"`
				DIMMStatus                   string   `json:"DIMMStatus"`
				MinimumVoltageVoltsX10       int      `json:"MinimumVoltageVoltsX10"`
				SPDBytes                      []string `json:"SPDBytes"`                      //[]interface{}
				RelatedPhysicalDIMMs          []string `json:"RelatedPhysicalDIMMs"`           //[]interface{}
				RelatedBackupStores           []string `json:"RelatedBackupStores"`            //[]interface{}
				RelatedPowerBackupUnitBays    []string `json:"RelatedPowerBackupUnitBays"`     //[]interface{}
			} `json:"Hpe"`
		} `json:"Oem"`
		OperatingMemoryModes []string `json:"OperatingMemoryModes"`
		OperatingSpeedMhz    int      `json:"OperatingSpeedMhz"`
		RankCount            int      `json:"RankCount"`
		Status               struct {
			Health string `json:"Health"`
			State  string `json:"State"`
		} `json:"Status"`
		VendorID string `json:"VendorID"` //был int
	} `json:"data"`
	Etag  string `json:"etag"`
	Count int    `json:"count"`
	Name  string `json:"name"`
}

// MemoryList300 - структура ответа Redfish (endpoint.Version == 300) со сводными данными о памяти сервера
type MemoryList300 struct {
	URI             string    `json:"uri"`
	CollectionState string    `json:"collectionState"`
	Modified        time.Time `json:"modified"`
	Data            []struct {
		BoardCPUNumber           int `json:"BoardCpuNumber"`
		BoardNumberOfSockets     int `json:"BoardNumberOfSockets"`
		BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
		BoardOperationalVoltage  int `json:"BoardOperationalVoltage"`
		BoardTotalMemorySize     int `json:"BoardTotalMemorySize"`
	} `json:"data"`
	Etag  string `json:"etag"`
	Count int    `json:"count"`
	Name  string `json:"name"`
}

// ServerHardwareListNew - структура ответа /rest/server-hardware, получаемого через собственный Redfish-парсер
type ServerHardwareListNew struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
	Members  []struct {
		Type            string    `json:"type"`
		Name            string    `json:"name"`
		ServerName      string    `json:"serverName"`
		State           string    `json:"state"`
		StateReason     string    `json:"stateReason"`
		AssetTag        string    `json:"assetTag"`
		Category        string    `json:"category"`
		Created         time.Time `json:"created"`
		Description     any       `json:"description"`
		ETag            string    `json:"eTag"`
		FormFactor      string    `json:"formFactor"`
		Generation      string    `json:"generation"`
		HostOsType      int       `json:"hostOsType"`
		LicensingIntent string    `json:"licensingIntent"`
		LocationURI     string    `json:"locationUri"`
		MaintenanceMode bool      `json:"maintenanceMode"`
		MemoryMb        int       `json:"memoryMb"`
		Model           string    `json:"model"`
		Modified        time.Time `json:"modified"`
		MpFirmwareVersion string  `json:"mpFirmwareVersion"`
		MpHostInfo      struct {
			MpHostName    string `json:"mpHostName"`
			MpIPAddresses []struct {
				Address string `json:"address"`
				Type    string `json:"type"`
			} `json:"mpIpAddresses"`
		} `json:"mpHostInfo"`
		MpLicenseType string `json:"mpLicenseType"`
		MpModel       string `json:"mpModel"`

		MpState             string `json:"mpState"`
		OperatingSystem     string `json:"operatingSystem"`
		PartNumber          string `json:"partNumber"`
		PhysicalServerHardwareURI any `json:"physicalServerHardwareUri"`
		Platform            string `json:"platform"`
		PortMap             struct {
			DeviceSlots []struct {
				PhysicalPorts []any  `json:"physicalPorts"`
				SlotNumber    int    `json:"slotNumber"`
				Location      string `json:"location"`
				DeviceName    string `json:"deviceName"`
				DeviceNumber  int    `json:"deviceNumber"`
			} `json:"deviceSlots"`
		} `json:"portMap"`
		Position               int    `json:"position"`
		PowerLock              bool   `json:"powerLock"`
		PowerState             string `json:"powerState"`
		ProcessorCoreCount     int    `json:"processorCoreCount"`
		ProcessorCount         int    `json:"processorCount"`
		ProcessorSpeedMhz      int    `json:"processorSpeedMhz"`
		ProcessorType          string `json:"processorType"`
		RefreshState           string `json:"refreshState"`
		RemoteSupportSettings  struct {
			RemoteSupportCurrentState string `json:"remoteSupportCurrentState"`
			Destination               string `json:"destination"`
		} `json:"remoteSupportSettings"`
		RomVersion            string `json:"romVersion"`
		SerialNumber          string `json:"serialNumber"`
		ServerGroupURI        string `json:"serverGroupUri"`
		ServerHardwareTypeURI string `json:"serverHardwareTypeUri"`
		ServerProfileURI      string `json:"serverProfileUri"`
		ShortModel            string `json:"shortModel"`
		Status                string `json:"status"`
		SubResources          struct {
			Memory struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					PartNumber                    string `json:"PartNumber"`
					PersistentRegionSizeLimitMiB int    `json:"PersistentRegionSizeLimitMiB"`
					OdataType                     string `json:"@odata.type"`
					LogicalSizeMiB                int    `json:"LogicalSizeMiB"`
					RankCount                     int    `json:"RankCount"`
					SecurityCapabilities           struct {
					} `json:"SecurityCapabilities"`
					BusWidthBits int    `json:"BusWidthBits"`
					CacheSizeMiB int    `json:"CacheSizeMiB"`
					Name         string `json:"Name"`

					NonVolatileSizeMiB   int      `json:"NonVolatileSizeMiB"`
					OdataID              string   `json:"@odata.id"`
					OperatingSpeedMhz    int      `json:"OperatingSpeedMhz,omitempty"`
					OperatingMemoryModes []string `json:"OperatingMemoryModes"`
					BaseModuleType       string   `json:"BaseModuleType,omitempty"`
					Manufacturer         string   `json:"Manufacturer,omitempty"`
					MemoryLocation       struct {
						Channel           int `json:"Channel"`
						Slot              int `json:"Slot"`
						MemoryController int `json:"MemoryController"`
						Socket            int `json:"Socket"`
					} `json:"MemoryLocation"`
					DeviceLocator string `json:"DeviceLocator"`
					MemoryType    string `json:"MemoryType"`
					Status        struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					VendorID         string `json:"VendorID"`
					OdataContext     string `json:"@odata.context"`
					DataWidthBits    int    `json:"DataWidthBits"`
					CapacityMiB      int    `json:"CapacityMiB"`
					VolatileSizeMiB  int    `json:"VolatileSizeMiB"`
					ErrorCorrection  string `json:"ErrorCorrection"`
					MemoryDeviceType string `json:"MemoryDeviceType,omitempty"`
					SerialNumber     string `json:"SerialNumber"`
					MemoryMedia      []string `json:"MemoryMedia"`
					Oem              struct {
						Hpe struct {
							VendorName          string   `json:"VendorName"`
							PartNumber          string   `json:"PartNumber"`
							DIMMStatus          string   `json:"DIMMStatus"`
							OdataType           string   `json:"@odata.type"`
							DIMMManufacturingDate string `json:"DIMMManufacturingDate"`
							MinimumVoltageVoltsX10 int   `json:"MinimumVoltageVoltsX10"`
							Attributes          []string `json:"Attributes"`
							BaseModuleType      string   `json:"BaseModuleType"`
							MaxOperatingSpeedMTs int     `json:"MaxOperatingSpeedMTs"`
							OdataContext        string   `json:"@odata.context"`
						} `json:"Hpe"`
					} `json:"Oem"`
					VolatileRegionSizeLimitMiB int    `json:"VolatileRegionSizeLimitMiB"`
					ID                          string `json:"Id"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"Memory"`
			SoftwareInventory struct {

				URI             string `json:"uri"`
				CollectionState string `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					OdataID     string `json:"@odata.id"`
					Description string `json:"Description"`
					Version     string `json:"Version"`
					OdataType   string `json:"@odata.type"`
					Oem         struct {
						Hpe struct {
							OdataType    string   `json:"@odata.type"`
							DeviceClass  string   `json:"DeviceClass"`
							Targets      []string `json:"Targets"`
							OdataContext string   `json:"@odata.context"`
						} `json:"Hpe"`
					} `json:"Oem"`
					Updateable   bool   `json:"Updateable"`
					OdataContext string `json:"@odata.context"`
					ID           string `json:"Id"`
					Name         string `json:"Name"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"SoftwareInventory"`
			MemoryList struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
					BoardCPUNumber            int `json:"BoardCpuNumber"`
					BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
					BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
					BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"MemoryList"`
			LocalStorageV2 struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					Status struct {
						HealthRollup string `json:"HealthRollup"`
						State        string `json:"State"`
					} `json:"Status"`

					Controllers []struct {
						SpeedGbps int `json:"SpeedGbps"`
						Status    struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						PCIeInterface struct {
							PCIeType    string `json:"PCIeType"`
							MaxPCIeType string `json:"MaxPCIeType"`
							MaxLanes    int    `json:"MaxLanes"`
							LanesInUse  int    `json:"LanesInUse"`
						} `json:"PCIeInterface"`
						PartNumber    string `json:"PartNumber"`
						OdataType     string `json:"@odata.type"`
						ControllerRates struct {
							RedfishWriteableProperties                    []string `json:"@Redfish.WriteableProperties"`
							TransformationRatePercentRedfishAllowableNumbers []string `json:"TransformationRatePercent@Redfish.AllowableNumbers"`
							ConsistencyCheckRatePercent                    int      `json:"ConsistencyCheckRatePercent"`
							RebuildRatePercent                              int      `json:"RebuildRatePercent"`
							ConsistencyCheckRatePercentRedfishAllowableNumbers []string `json:"ConsistencyCheckRatePercent@Redfish.AllowableNumbers"`
							RebuildRatePercentRedfishAllowableNumbers       []string `json:"RebuildRatePercent@Redfish.AllowableNumbers"`
							TransformationRatePercent                       int      `json:"TransformationRatePercent"`
						} `json:"ControllerRates"`
						OdataEtag string `json:"@odata.etag"`
						Ports     struct {
							OdataID string `json:"@odata.id"`
						} `json:"Ports"`
						CacheSummary struct {
							Status struct {
								Health string `json:"Health"`
								State  string `json:"State"`
							} `json:"Status"`
							TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
							PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
						} `json:"CacheSummary"`
						SupportedRAIDTypes         []string `json:"SupportedRAIDTypes"`
						FirmwareVersion            string   `json:"FirmwareVersion"`
						Name                       string   `json:"Name"`
						OdataID                    string   `json:"@odata.id"`
						SupportedControllerProtocols []string `json:"SupportedControllerProtocols"`
						SerialNumber               string   `json:"SerialNumber"`
						Manufacturer               string   `json:"Manufacturer"`
						Model                      string   `json:"Model"`
						Links                      struct {
							BatteriesOdataCount int `json:"Batteries@odata.count"`
							Batteries           []struct {
								OdataID string `json:"@odata.id"`
							} `json:"Batteries"`
						} `json:"Links"`

						SupportedDeviceProtocols []string `json:"SupportedDeviceProtocols"`
						ID                       string   `json:"Id"`
						Identifiers              []struct {
							DurableNameFormat string `json:"DurableNameFormat"`
							DurableName       string `json:"DurableName"`
						} `json:"Identifiers"`
						Sku      string `json:"SKU"`
						Location struct {
							PartLocation struct {
								ServiceLabel        string `json:"ServiceLabel"`
								LocationOrdinalValue int   `json:"LocationOrdinalValue"`
								LocationType        string `json:"LocationType"`
							} `json:"PartLocation"`
						} `json:"Location"`
					} `json:"Controllers"`
					Actions struct {
						StorageResetToDefaults struct {
							ResetTypeRedfishAllowableValues []string `json:"ResetType@Redfish.AllowableValues"`
							Target                           string   `json:"target"`
						} `json:"#Storage.ResetToDefaults"`
					} `json:"Actions"`
					OdataType string `json:"@odata.type"`
					Drives    []struct {
						RedfishWriteableProperties []string `json:"@Redfish.WriteableProperties"`
						HotspareType                string   `json:"HotspareType"`
						OdataType                   string   `json:"@odata.type"`
						CapacityBytes               int64    `json:"CapacityBytes"`
						Name                        string   `json:"Name"`
						StatusIndicator             string   `json:"StatusIndicator"`
						OdataID                     string   `json:"@odata.id"`
						EncryptionAbility           string   `json:"EncryptionAbility"`
						NegotiatedSpeedGbs          int      `json:"NegotiatedSpeedGbs"`
						MediaType                   string   `json:"MediaType"`
						LocationIndicatorActive     bool     `json:"LocationIndicatorActive"`
						Protocol                    string   `json:"Protocol"`
						Operations                  []any    `json:"Operations"`
						PhysicalLocation            struct {
							PartLocation struct {
								ServiceLabel        string `json:"ServiceLabel"`
								LocationOrdinalValue int   `json:"LocationOrdinalValue"`
								LocationType        string `json:"LocationType"`
							} `json:"PartLocation"`
						} `json:"PhysicalLocation"`
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						OdataEtag        string `json:"@odata.etag"`
						WriteCacheEnabled bool  `json:"WriteCacheEnabled"`

						BlockSizeBytes  int    `json:"BlockSizeBytes"`
						CapableSpeedGbs int    `json:"CapableSpeedGbs"`
						Multipath       bool   `json:"Multipath"`
						FailurePredicted bool  `json:"FailurePredicted"`
						Revision        string `json:"Revision"`
						SerialNumber    string `json:"SerialNumber"`
						Model           string `json:"Model"`
						RotationSpeedRPM int   `json:"RotationSpeedRPM"`
						Links           struct {
							Volumes []struct {
								OdataID string `json:"@odata.id"`
							} `json:"Volumes"`
							Storage struct {
								OdataID string `json:"@odata.id"`
							} `json:"Storage"`
							VolumesOdataCount int `json:"Volumes@odata.count"`
						} `json:"Links"`
						ID          string `json:"Id"`
						Identifiers []struct {
							DurableNameFormat string `json:"DurableNameFormat"`
							DurableName       string `json:"DurableName"`
						} `json:"Identifiers"`
					} `json:"Drives"`
					OdataEtag string `json:"@odata.etag"`
					Name      string `json:"Name"`
					OdataID   string `json:"@odata.id"`
					Volumes   []struct {
						RedfishWriteableProperties          []string `json:"@Redfish.WriteableProperties"`
						WriteCachePolicyRedfishAllowableValues []string `json:"WriteCachePolicy@Redfish.AllowableValues"`
						OdataType                            string   `json:"@odata.type"`
						Encrypted                            bool     `json:"Encrypted"`
						CapacityBytes                        int64    `json:"CapacityBytes"`
						Name                                  string   `json:"Name"`
						OptimumIOSizeBytes                   int      `json:"OptimumIOSizeBytes"`
						DisplayNameRedfishAllowablePattern    string   `json:"DisplayName@Redfish.AllowablePattern"`
						OdataID                               string   `json:"@odata.id"`
						DisplayName                           string   `json:"DisplayName"`
						StripSizeBytes                        int      `json:"StripSizeBytes"`
						Operations                            []any    `json:"Operations"`
						Status                                struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						ReadCachePolicy string `json:"ReadCachePolicy"`
						EncryptionTypes []any  `json:"EncryptionTypes"`
						OdataEtag       string `json:"@odata.etag"`
						RAIDType        string `json:"RAIDType"`
						WriteCachePolicy string `json:"WriteCachePolicy"`
						BlockSizeBytes  int    `json:"BlockSizeBytes"`

						LogicalUnitNumber                    int      `json:"LogicalUnitNumber"`
						ReadCachePolicyRedfishAllowableValues []string `json:"ReadCachePolicy@Redfish.AllowableValues"`
						Links                                 struct {
							RedfishWriteableProperties  []any `json:"@Redfish.WriteableProperties"`
							DedicatedSpareDrives         []any `json:"DedicatedSpareDrives"`
							DedicatedSpareDrivesOdataCount int `json:"DedicatedSpareDrives@odata.count"`
							DrivesOdataCount             int   `json:"Drives@odata.count"`
							Drives                        []struct {
								OdataID string `json:"@odata.id"`
							} `json:"Drives"`
						} `json:"Links"`
						ID          string `json:"Id"`
						Identifiers []struct {
							DurableNameFormat string `json:"DurableNameFormat"`
							DurableName       string `json:"DurableName"`
						} `json:"Identifiers"`
						WriteCacheState string `json:"WriteCacheState"`
						VolumeUsage     string `json:"VolumeUsage"`
					} `json:"Volumes"`
					Annotations struct {
						Volumes struct {
							WriteCachePolicyRedfishAllowableValues          []string `json:"WriteCachePolicy@Redfish.AllowableValues"`
							DisplayNameRedfishUpdatableAfterCreate          bool     `json:"DisplayName@Redfish.UpdatableAfterCreate"`
							OdataType                                       string   `json:"@odata.type"`
							Name                                            string   `json:"Name"`
							InitializeMethodRedfishOptionalOnCreate         bool     `json:"InitializeMethod@Redfish.OptionalOnCreate"`
							DisplayNameRedfishAllowablePattern              string   `json:"DisplayName@Redfish.AllowablePattern"`
							VolumeUsageRedfishAllowableValues               []string `json:"VolumeUsage@Redfish.AllowableValues"`
							OdataID                                         string   `json:"@odata.id"`
							NameRedfishOptionalOnCreate                     bool     `json:"Name@Redfish.OptionalOnCreate"`
							IOPerfModeEnabledRedfishOptionalOnCreate        bool     `json:"IOPerfModeEnabled@Redfish.OptionalOnCreate"`
							ReadCachePolicyRedfishUpdatableAfterCreate      bool     `json:"ReadCachePolicy@Redfish.UpdatableAfterCreate"`
							InitializeMethodRedfishAllowableValues          []string `json:"InitializeMethod@Redfish.AllowableValues"`
							MediaSpanCountRedfishOptionalOnCreate           bool     `json:"MediaSpanCount@Redfish.OptionalOnCreate"`
							MediaSpanCountRedfishAllowableNumbers           []any    `json:"MediaSpanCount@Redfish.AllowableNumbers"`
							StripSizeBytesRedfishOptionalOnCreate           bool     `json:"StripSizeBytes@Redfish.OptionalOnCreate"`
							WriteCachePolicyRedfishOptionalOnCreate         bool     `json:"WriteCachePolicy@Redfish.OptionalOnCreate"`
							VolumeUsageRedfishOptionalOnCreate              bool     `json:"VolumeUsage@Redfish.OptionalOnCreate"`
							OdataEtag                                       string   `json:"@odata.etag"`
							RAIDTypeRedfishRequiredOnCreate                 bool     `json:"RAIDType@Redfish.RequiredOnCreate"`
							ReadCachePolicyRedfishOptionalOnCreate          bool     `json:"ReadCachePolicy@Redfish.OptionalOnCreate"`
							CapacityBytesRedfishOptionalOnCreate            bool     `json:"CapacityBytes@Redfish.OptionalOnCreate"`
							ReadCachePolicyRedfishAllowableValues           []string `json:"ReadCachePolicy@Redfish.AllowableValues"`
							CapacityBytesRedfishAllowableNumbers            []any    `json:"CapacityBytes@Redfish.AllowableNumbers"`
							LinksRedfishRequiredOnCreate                    bool     `json:"Links@Redfish.RequiredOnCreate"`
							NameRedfishAllowablePattern                    string   `json:"Name@Redfish.AllowablePattern"`
							StripSizeBytesRedfishAllowableNumbers           []string `json:"StripSizeBytes@Redfish.AllowableNumbers"`
							WriteCachePolicyRedfishUpdatableAfterCreate     bool     `json:"WriteCachePolicy@Redfish.UpdatableAfterCreate"`
							Links                                           struct {

								DedicatedSpareDrivesRedfishOptionalOnCreate     bool `json:"DedicatedSpareDrives@Redfish.OptionalOnCreate"`
								DedicatedSpareDrivesRedfishUpdatableAfterCreate bool `json:"DedicatedSpareDrives@Redfish.UpdatableAfterCreate"`
								DrivesRedfishRequiredOnCreate                   bool `json:"Drives@Redfish.RequiredOnCreate"`
							} `json:"Links"`
							ID                                       string   `json:"Id"`
							RAIDTypeRedfishAllowableValues           []string `json:"RAIDType@Redfish.AllowableValues"`
							IOPerfModeEnabledRedfishUpdatableAfterCreate bool `json:"IOPerfModeEnabled@Redfish.UpdatableAfterCreate"`
							DisplayNameRedfishOptionalOnCreate       bool     `json:"DisplayName@Redfish.OptionalOnCreate"`
						} `json:"Volumes"`
					} `json:"Annotations"`
					StorageControllers []struct {
						SpeedGbps int `json:"SpeedGbps"`
						Status    struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						PCIeInterface struct {
							PCIeType    string `json:"PCIeType"`
							MaxPCIeType string `json:"MaxPCIeType"`
							MaxLanes    int    `json:"MaxLanes"`
							LanesInUse  int    `json:"LanesInUse"`
						} `json:"PCIeInterface"`
						PartNumber      string `json:"PartNumber"`
						OdataType       string `json:"@odata.type"`
						ControllerRates struct {
							RedfishWriteableProperties                        []string `json:"@Redfish.WriteableProperties"`
							TransformationRatePercentRedfishAllowableNumbers  []string `json:"TransformationRatePercent@Redfish.AllowableNumbers"`
							ConsistencyCheckRatePercent                       int      `json:"ConsistencyCheckRatePercent"`
							RebuildRatePercent                                int      `json:"RebuildRatePercent"`
							ConsistencyCheckRatePercentRedfishAllowableNumbers []string `json:"ConsistencyCheckRatePercent@Redfish.AllowableNumbers"`
							RebuildRatePercentRedfishAllowableNumbers          []string `json:"RebuildRatePercent@Redfish.AllowableNumbers"`
							TransformationRatePercent                          int      `json:"TransformationRatePercent"`
						} `json:"ControllerRates"`
						OdataEtag    string `json:"@odata.etag"`
						Ports        struct {
							OdataID string `json:"@odata.id"`
						} `json:"Ports"`
						CacheSummary struct {
							Status struct {
								Health string `json:"Health"`
								State  string `json:"State"`
							} `json:"Status"`
							TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
							PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
						} `json:"CacheSummary"`
						SupportedRAIDTypes []string `json:"SupportedRAIDTypes"`
						FirmwareVersion    string   `json:"FirmwareVersion"`
						Name               string   `json:"Name"`
						OdataID            string   `json:"@odata.id"`

						SupportedControllerProtocols []string `json:"SupportedControllerProtocols"`
						SerialNumber                string   `json:"SerialNumber"`
						Manufacturer                string   `json:"Manufacturer"`
						Model                        string   `json:"Model"`
						Links                        struct {
							BatteriesOdataCount int `json:"Batteries@odata.count"`
							Batteries           []struct {
								OdataID string `json:"@odata.id"`
							} `json:"Batteries"`
						} `json:"Links"`
						SupportedDeviceProtocols []string `json:"SupportedDeviceProtocols"`
						ID                       string   `json:"Id"`
						Identifiers              []struct {
							DurableNameFormat string `json:"DurableNameFormat"`
							DurableName       string `json:"DurableName"`
						} `json:"Identifiers"`
						Sku      string `json:"SKU"`
						Location struct {
							PartLocation struct {
								ServiceLabel        string `json:"ServiceLabel"`
								LocationOrdinalValue int   `json:"LocationOrdinalValue"`
								LocationType        string `json:"LocationType"`
							} `json:"PartLocation"`
						} `json:"Location"`
					} `json:"StorageControllers"`
					DrivesOdataCount int    `json:"Drives@odata.count"`
					ID               string `json:"Id"`
					AutoVolumeCreate string `json:"AutoVolumeCreate"`
					EncryptionMode   string `json:"EncryptionMode"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"LocalStorageV2"`
			Thermal struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					MemberID string `json:"MemberId"`
					OdataID  string `json:"@odata.id"`
					Reading  int    `json:"Reading"`
					Oem      struct {
						Hpe struct {
							OdataType string `json:"@odata.type"`

							HotPluggable bool   `json:"HotPluggable"`
							Redundant    bool   `json:"Redundant"`
							OdataContext string `json:"@odata.context"`
							Location     string `json:"Location"`
						} `json:"Hpe"`
					} `json:"Oem"`
					ReadingUnits    string `json:"ReadingUnits"`
					Name            string `json:"Name"`
					PhysicalContext string `json:"PhysicalContext"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"Thermal"`
			AdvancedMemoryProtection struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					AmpModeActive    string   `json:"AmpModeActive"`
					OdataType        string   `json:"@odata.type"`
					AmpModeSupported []string `json:"AmpModeSupported"`
					OdataContext     string   `json:"@odata.context"`
					AmpModeStatus    string   `json:"AmpModeStatus"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"AdvancedMemoryProtection"`
			Devices struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					Status struct {
						State string `json:"State"`
					} `json:"Status,omitempty"`
					PartNumber string `json:"PartNumber"`
					OdataType  string `json:"@odata.type"`
					PCISlot    struct {
						LinkLanes string `json:"LinkLanes"`
						Status    struct {
							OperationalStatus []struct {
								Status string `json:"Status"`
							} `json:"OperationalStatus"`
						} `json:"Status"`
						DeviceNumber int    `json:"DeviceNumber"`
						OdataType    string `json:"@odata.type"`
						OdataEtag    string `json:"@odata.etag"`

						OdataContext string `json:"@odata.context"`
						Name         string `json:"Name"`
						UEFIDevicePath string `json:"UEFIDevicePath"`
						BusNumber    int    `json:"BusNumber"`
						SupportsHotPlug bool `json:"SupportsHotPlug"`
						OdataID      string `json:"@odata.id"`
						FunctionNumber int  `json:"FunctionNumber"`
						Characteristics struct {
							SupportsPowerManagementEventSignal bool `json:"SupportsPowerManagementEventSignal"`
							Provides33Volts                    bool `json:"Provides3_3Volts"`
						} `json:"Characteristics"`
						Length     string `json:"Length"`
						Technology string `json:"Technology"`
						ID         string `json:"Id"`
					} `json:"PCISlot,omitempty"`
					OdataContext  string `json:"@odata.context"`
					ProductVersion string `json:"ProductVersion"`
					FirmwareVersion struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					Name             string `json:"Name"`
					DeviceType       string `json:"DeviceType"`
					OdataID          string `json:"@odata.id"`
					SerialNumber     string `json:"SerialNumber"`
					ProductPartNumber string `json:"ProductPartNumber"`
					Manufacturer     string `json:"Manufacturer"`
					Links            struct {
						PCISlot struct {
							OdataID string `json:"@odata.id"`
						} `json:"PCISlot"`
					} `json:"Links,omitempty"`
					ID                    string `json:"Id"`
					MCTPProtocolDisabled  bool   `json:"MCTPProtocolDisabled,omitempty"`
					Location              string `json:"Location"`
					Status0               struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status,omitempty"`
					DeviceInstances []struct {
						OdataID string `json:"@odata.id"`
					} `json:"DeviceInstances,omitempty"`
					Status1 struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status,omitempty"`
					Status2 struct {
						Health string `json:"Health"`

						State string `json:"State"`
					} `json:"Status,omitempty"`
					Status3 struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status,omitempty"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"Devices"`
			PowerSupplies struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []any     `json:"data"`
				Etag            string    `json:"etag"`
				Count           int       `json:"count"`
				Name            string    `json:"name"`
			} `json:"PowerSupplies"`
			LocalStorage struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					ControllerPartNumber          string `json:"ControllerPartNumber"`
					EncryptionMixedVolumesEnabled bool   `json:"EncryptionMixedVolumesEnabled"`
					CurrentOperatingMode          string `json:"CurrentOperatingMode"`
					Description                   string `json:"Description"`
					LocationFormat                string `json:"LocationFormat"`
					EncryptionStandaloneModeEnabled bool `json:"EncryptionStandaloneModeEnabled"`
					OdataType                     string `json:"@odata.type"`
					AdapterType                   string `json:"AdapterType"`
					PhysicalDrives                []struct {
						CarrierApplicationVersion string `json:"CarrierApplicationVersion"`
						Description                string `json:"Description"`
						LocationFormat              string `json:"LocationFormat"`
						CapacityLogicalBlocks       int    `json:"CapacityLogicalBlocks"`
						UncorrectedReadErrors       int    `json:"UncorrectedReadErrors"`
						OdataType                   string `json:"@odata.type"`
						PowerOnHours                any    `json:"PowerOnHours"`
						MaximumTemperatureCelsius   int    `json:"MaximumTemperatureCelsius"`
						Name                        string `json:"Name"`
						OdataID                     string `json:"@odata.id"`
						InterfaceType               string `json:"InterfaceType"`
						DiskDriveStatusReasons      []string `json:"DiskDriveStatusReasons"`
						CarrierAuthenticationStatus string `json:"CarrierAuthenticationStatus"`
						MediaType                   string `json:"MediaType"`
						CurrentTemperatureCelsius   int    `json:"CurrentTemperatureCelsius"`

						InterfaceSpeedMbps    int `json:"InterfaceSpeedMbps"`
						UncorrectedWriteErrors int `json:"UncorrectedWriteErrors"`
						Status                struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						CapacityGB                     int    `json:"CapacityGB"`
						OdataEtag                      string `json:"@odata.etag"`
						OdataContext                    string `json:"@odata.context"`
						BlockSizeBytes                  int    `json:"BlockSizeBytes"`
						SSDEnduranceUtilizationPercentage any  `json:"SSDEnduranceUtilizationPercentage"`
						CapacityMiB                      int    `json:"CapacityMiB"`
						FirmwareVersion                  struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"FirmwareVersion"`
						LegacyBootPriority string `json:"LegacyBootPriority"`
						SerialNumber       string `json:"SerialNumber"`
						Model              string `json:"Model"`
						RotationalSpeedRpm int    `json:"RotationalSpeedRpm"`
						ID                 string `json:"Id"`
						DiskDriveUse       string `json:"DiskDriveUse"`
						EncryptedDrive     bool   `json:"EncryptedDrive"`
						Location           string `json:"Location"`
					} `json:"PhysicalDrives"`
					LogicalDrives []struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						LogicalDriveEncryption bool   `json:"LogicalDriveEncryption"`
						Description             string `json:"Description"`
						StripeSizeBytes         int    `json:"StripeSizeBytes"`
						DataDrives             []struct {
							CarrierApplicationVersion string   `json:"CarrierApplicationVersion"`
							Description                string   `json:"Description"`
							LocationFormat             string   `json:"LocationFormat"`
							CapacityLogicalBlocks      int      `json:"CapacityLogicalBlocks"`
							UncorrectedReadErrors      int      `json:"UncorrectedReadErrors"`
							OdataType                  string   `json:"@odata.type"`
							PowerOnHours               any      `json:"PowerOnHours"`
							MaximumTemperatureCelsius  int      `json:"MaximumTemperatureCelsius"`
							Name                       string   `json:"Name"`
							OdataID                    string   `json:"@odata.id"`
							InterfaceType              string   `json:"InterfaceType"`
							DiskDriveStatusReasons     []string `json:"DiskDriveStatusReasons"`
							CarrierAuthenticationStatus string  `json:"CarrierAuthenticationStatus"`
							MediaType                  string   `json:"MediaType"`

							CurrentTemperatureCelsius int `json:"CurrentTemperatureCelsius"`
							InterfaceSpeedMbps        int `json:"InterfaceSpeedMbps"`
							UncorrectedWriteErrors    int `json:"UncorrectedWriteErrors"`
							Status                    struct {
								Health string `json:"Health"`
								State  string `json:"State"`
							} `json:"Status"`
							CapacityGB                        int    `json:"CapacityGB"`
							OdataContext                       string `json:"@odata.context"`
							BlockSizeBytes                     int    `json:"BlockSizeBytes"`
							SSDEnduranceUtilizationPercentage any    `json:"SSDEnduranceUtilizationPercentage"`
							CapacityMiB                        int    `json:"CapacityMiB"`
							FirmwareVersion                    struct {
								Current struct {
									VersionString string `json:"VersionString"`
								} `json:"Current"`
							} `json:"FirmwareVersion"`
							LegacyBootPriority string `json:"LegacyBootPriority"`
							SerialNumber       string `json:"SerialNumber"`
							Model              string `json:"Model"`
							RotationalSpeedRpm int    `json:"RotationalSpeedRpm"`
							ID                 string `json:"Id"`
							DiskDriveUse       string `json:"DiskDriveUse"`
							EncryptedDrive     bool   `json:"EncryptedDrive"`
							Location           string `json:"Location"`
						} `json:"DataDrives"`
						OdataType          string `json:"@odata.type"`
						OdataEtag          string `json:"@odata.etag"`
						OdataContext       string `json:"@odata.context"`
						Raid               string `json:"Raid"`
						CapacityMiB        int    `json:"CapacityMiB"`
						LogicalDriveType   string `json:"LogicalDriveType"`
						LogicalDriveNumber int    `json:"LogicalDriveNumber"`
						AccelerationMethod string `json:"AccelerationMethod"`
						Name               string `json:"Name"`
						OdataID            string `json:"@odata.id"`
						InterfaceType      string `json:"InterfaceType"`
						LegacyBootPriority string `json:"LegacyBootPriority"`
						Links              struct {
							DataDrives struct {
								OdataID string `json:"@odata.id"`
							} `json:"DataDrives"`
						} `json:"Links"`
						LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
						MediaType                 string   `json:"MediaType"`
						ID                        string   `json:"Id"`
						VolumeUniqueIdentifier    string   `json:"VolumeUniqueIdentifier"`
						LogicalDriveName          string   `json:"LogicalDriveName"`
					} `json:"LogicalDrives"`

					ReadCachePercent    int    `json:"ReadCachePercent"`
					InternalPortCount   int    `json:"InternalPortCount"`
					CacheMemorySizeMiB  int    `json:"CacheMemorySizeMiB"`
					EncryptionFwLocked  bool   `json:"EncryptionFwLocked"`
					Name                string `json:"Name"`
					OdataID             string `json:"@odata.id"`
					BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
					DriveWriteCache     string `json:"DriveWriteCache"`
					DualModeControllerSupported bool `json:"DualModeControllerSupported"`
					HardwareRevision    string `json:"HardwareRevision"`
					Status              struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					EncryptionSelfTestPassed bool  `json:"EncryptionSelfTestPassed"`
					ExternalPortCount        int   `json:"ExternalPortCount"`
					OdataContext             string `json:"@odata.context"`
					UnconfiguredDrives       []any `json:"UnconfiguredDrives"`
					ControllerBoard          struct {
						Status struct {
							Health string `json:"Health"`
						} `json:"Status"`
					} `json:"ControllerBoard"`
					EncryptionEnabled bool `json:"EncryptionEnabled"`
					EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
					FirmwareVersion struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
					EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
					SerialNumber                       string `json:"SerialNumber"`
					CacheModuleStatus                  struct {
						Health string `json:"Health"`
					} `json:"CacheModuleStatus"`
					WriteCacheWithoutBackupPowerEnabled bool `json:"WriteCacheWithoutBackupPowerEnabled"`
					StorageEnclosures                   []struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						Description    string `json:"Description"`
						LocationFormat string `json:"LocationFormat"`
						OdataType      string `json:"@odata.type"`
						DriveBayCount  int    `json:"DriveBayCount"`
						OdataEtag      string `json:"@odata.etag"`
						OdataContext   string `json:"@odata.context"`
						FirmwareVersion struct {

							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"FirmwareVersion"`
						Name         string `json:"Name"`
						OdataID      string `json:"@odata.id"`
						SerialNumber string `json:"SerialNumber"`
						Model        string `json:"Model"`
						ID           string `json:"Id"`
						Location     string `json:"Location"`
					} `json:"StorageEnclosures"`
					CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
					Model                   string `json:"Model"`
					Links                   struct {
						StorageEnclosures struct {
							OdataID string `json:"@odata.id"`
						} `json:"StorageEnclosures"`
						PhysicalDrives struct {
							OdataID string `json:"@odata.id"`
						} `json:"PhysicalDrives"`
						LogicalDrives struct {
							OdataID string `json:"@odata.id"`
						} `json:"LogicalDrives"`
						UnconfiguredDrives struct {
							OdataID string `json:"@odata.id"`
						} `json:"UnconfiguredDrives"`
					} `json:"Links"`
					ID                        string `json:"Id"`
					WriteCacheBypassThresholdKB int  `json:"WriteCacheBypassThresholdKB"`
					Location                  string `json:"Location"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"LocalStorage"`
			Chassis struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status,omitempty"`
					OdataType    string `json:"@odata.type"`
					OdataContext string `json:"@odata.context"`
					Name         string `json:"Name"`
					PCIeSlots    struct {
						OdataID string `json:"@odata.id"`

					} `json:"PCIeSlots,omitempty"`
					AssetTag    string `json:"AssetTag,omitempty"`
					OdataID     string `json:"@odata.id"`
					PCIeDevices struct {
						OdataID string `json:"@odata.id"`
					} `json:"PCIeDevices,omitempty"`
					SerialNumber string `json:"SerialNumber"`
					Oem          struct {
						Hpe struct {
							Actions struct {
								HpeServerChassisFactoryResetMCTP struct {
									Target string `json:"target"`
								} `json:"#HpeServerChassis.FactoryResetMCTP"`
								HpeServerChassisDisableMCTPOnServer struct {
									Target string `json:"target"`
								} `json:"#HpeServerChassis.DisableMCTPOnServer"`
							} `json:"Actions"`
							BayNumber              int    `json:"BayNumber"`
							OdataType              string `json:"@odata.type"`
							SystemMaintenanceSwitches struct {
								Sw5  string `json:"Sw5"`
								Sw4  string `json:"Sw4"`
								Sw12 string `json:"Sw12"`
								Sw7  string `json:"Sw7"`
								Sw11 string `json:"Sw11"`
								Sw6  string `json:"Sw6"`
								Sw10 string `json:"Sw10"`
								Sw9  string `json:"Sw9"`
								Sw8  string `json:"Sw8"`
								Sw1  string `json:"Sw1"`
								Sw3  string `json:"Sw3"`
								Sw2  string `json:"Sw2"`
							} `json:"SystemMaintenanceSwitches"`
							BaysConsumedHeight  int    `json:"BaysConsumedHeight"`
							MCTPEnabledOnServer bool   `json:"MCTPEnabledOnServer"`
							OdataContext        string `json:"@odata.context"`
							BaysConsumedWidth   int    `json:"BaysConsumedWidth"`
							FormFactor          string `json:"FormFactor"`
							SmartStorageBattery []struct {
								Status struct {
									Health string `json:"Health"`
									State  string `json:"State"`
								} `json:"Status"`
								MaximumCapWatts          int    `json:"MaximumCapWatts"`
								ProductName              string `json:"ProductName"`
								SerialNumber             string `json:"SerialNumber"`
								Model                    string `json:"Model"`
								RemainingChargeTimeSeconds int  `json:"RemainingChargeTimeSeconds"`
								Index                    int    `json:"Index"`

						ChargeLevelPercent int    `json:"ChargeLevelPercent"`
						SparePartNumber    string `json:"SparePartNumber"`
						FirmwareVersion    string `json:"FirmwareVersion"`
					} `json:"SmartStorageBattery"`
					Links struct {
						BaseFrus struct {
							OdataID string `json:"@odata.id"`
						} `json:"BaseFrus"`
						MezzFrus struct {
							OdataID string `json:"@odata.id"`
						} `json:"MezzFrus"`
						BladeEnclosure struct {
							OdataID string `json:"@odata.id"`
						} `json:"BladeEnclosure"`
						Devices struct {
							OdataID string `json:"@odata.id"`
						} `json:"Devices"`
					} `json:"Links"`
					Firmware struct {
						SystemProgrammableLogicDevice struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"SystemProgrammableLogicDevice"`
						PlatformDefinitionTable struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"PlatformDefinitionTable"`
						PowerManagementController struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"PowerManagementController"`
						SPSFirmwareVersionData struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"SPSFirmwareVersionData"`
						PowerManagementControllerBootloader struct {
							Current struct {
								VersionString string `json:"VersionString"`
								Family        string `json:"Family"`
							} `json:"Current"`
						} `json:"PowerManagementControllerBootloader"`
					} `json:"Firmware"`
					ElConfigOverride bool `json:"ElConfigOverride"`
					PowerAlertMode   struct {
						Activated bool `json:"Activated"`

						AlertPowerWatts int `json:"AlertPowerWatts"`
					} `json:"PowerAlertMode"`
				} `json:"Hpe"`
			} `json:"Oem"`
			Manufacturer string `json:"Manufacturer"`
			Model        string `json:"Model,omitempty"`
			Links        struct {
				ContainedBy struct {
					OdataID string `json:"@odata.id"`
				} `json:"ContainedBy"`
				ManagedBy []struct {
					OdataID string `json:"@odata.id"`
				} `json:"ManagedBy"`
				ComputerSystems []struct {
					OdataID string `json:"@odata.id"`
				} `json:"ComputerSystems"`
			} `json:"Links,omitempty"`
			NetworkAdapters struct {
				OdataID string `json:"@odata.id"`
			} `json:"NetworkAdapters,omitempty"`
			ID                      string `json:"Id"`
			ChassisType             string `json:"ChassisType"`
			LocationIndicatorActive bool   `json:"LocationIndicatorActive"`
			Sku                     string `json:"SKU,omitempty"`
			PowerState              string `json:"PowerState"`
			IndicatorLED            string `json:"IndicatorLED"`
			Power                   struct {
				OdataID string `json:"@odata.id"`
			} `json:"Power,omitempty"`
			Thermal struct {
				OdataID string `json:"@odata.id"`
			} `json:"Thermal,omitempty"`
			Status0 struct {
				Health string `json:"Health"`
			} `json:"Status,omitempty"`
			Drives struct {
				OdataID string `json:"@odata.id"`
			} `json:"Drives,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Chassis"`
	FirmwareInventory struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {

				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status,omitempty"`
			OdataID     string `json:"@odata.id"`
			Description string `json:"Description"`
			Version     string `json:"Version"`
			OdataType   string `json:"@odata.type"`
			Oem         struct {
				Hpe struct {
					DeviceContext string   `json:"DeviceContext"`
					OdataType     string   `json:"@odata.type"`
					DeviceClass   string   `json:"DeviceClass"`
					Targets       []string `json:"Targets"`
					OdataContext  string   `json:"@odata.context"`
				} `json:"Hpe"`
			} `json:"Oem"`
			Updateable   bool   `json:"Updateable"`
			OdataContext string `json:"@odata.context"`
			ID           string `json:"Id"`
			Name         string `json:"Name"`
			SoftwareID   string `json:"SoftwareId,omitempty"`
			RelatedItem  []struct {
				OdataID string `json:"@odata.id"`
			} `json:"RelatedItem,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"FirmwareInventory"`
	NetworkAdapters struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Controllers []struct {
				ControllerCapabilities struct {
					DataCenterBridging struct {
						Capable bool `json:"Capable"`
					} `json:"DataCenterBridging"`
					Npar struct {
						NparEnabled bool `json:"NparEnabled"`
						NparCapable bool `json:"NparCapable"`
					} `json:"NPAR"`
					VirtualizationOffload struct {
						VirtualFunction struct {
							DeviceMaxCount        int `json:"DeviceMaxCount"`
							NetworkPortMaxCount   int `json:"NetworkPortMaxCount"`
							MinAssignmentGroupSize int `json:"MinAssignmentGroupSize"`
						} `json:"VirtualFunction"`

						Sriov struct {
							SRIOVVEPACapable bool `json:"SRIOVVEPACapable"`
						} `json:"SRIOV"`
					} `json:"VirtualizationOffload"`
					Npiv struct {
						MaxDeviceLogins int `json:"MaxDeviceLogins"`
						MaxPortLogins   int `json:"MaxPortLogins"`
					} `json:"NPIV"`
					NetworkPortCount           int `json:"NetworkPortCount"`
					NetworkDeviceFunctionCount int `json:"NetworkDeviceFunctionCount"`
				} `json:"ControllerCapabilities"`
				FirmwarePackageVersion string `json:"FirmwarePackageVersion"`
				Links                  struct {
					NetworkDeviceFunctions []struct {
						OdataID string `json:"@odata.id"`
					} `json:"NetworkDeviceFunctions"`
					Ports []struct {
						OdataID string `json:"@odata.id"`
					} `json:"Ports"`
				} `json:"Links"`
				Location struct {
					PartLocation struct {
						ServiceLabel        string `json:"ServiceLabel"`
						LocationOrdinalValue int   `json:"LocationOrdinalValue"`
						LocationType        string `json:"LocationType"`
					} `json:"PartLocation"`
				} `json:"Location"`
			} `json:"Controllers"`
			Description string `json:"Description"`
			PartNumber  string `json:"PartNumber"`
			Actions     struct {
				NetworkAdapterResetSettingsToDefault struct {
					Title  string `json:"title"`
					Target string `json:"target"`
				} `json:"#NetworkAdapter.ResetSettingsToDefault"`
			} `json:"Actions"`
			OdataType string `json:"@odata.type"`
			Ports     []struct {
				LinkStatus          string `json:"LinkStatus"`
				Description         string `json:"Description"`
				OdataType           string `json:"@odata.type"`
				SignalDetected      bool   `json:"SignalDetected"`
				LinkNetworkTechnology string `json:"LinkNetworkTechnology"`
				FunctionMinBandwidth []struct {
					AllocationPercent  int `json:"AllocationPercent"`
					NetworkDeviceFunction struct {
						OdataID string `json:"@odata.id"`
					} `json:"NetworkDeviceFunction"`
				} `json:"FunctionMinBandwidth"`

				PortID        string `json:"PortId"`
				EEEEnabled    bool   `json:"EEEEnabled"`
				OdataContext  string `json:"@odata.context"`
				Name          string `json:"Name"`
				PortMaximumMTU int   `json:"PortMaximumMTU"`
				PortProtocol  string `json:"PortProtocol"`
				OdataID       string `json:"@odata.id"`
				LinkConfiguration []struct {
					AutoSpeedNegotiationCapable any   `json:"AutoSpeedNegotiationCapable"`
					CapableLinkSpeedGbps        []int `json:"CapableLinkSpeedGbps"`
				} `json:"LinkConfiguration"`
				CurrentSpeedGbps int `json:"CurrentSpeedGbps"`
				Oem              struct {
					Hpe struct {
						PortDiagnosticRemoteLoopbackCapable bool `json:"PortDiagnosticRemoteLoopbackCapable"`
						PortDiagnosticEnableRemoteLoopback  bool `json:"PortDiagnosticEnableRemoteLoopback"`
						ConfiguredNetworkLinks              []struct {
							ConfiguredLinkSpeedGbps int `json:"ConfiguredLinkSpeedGbps"`
						} `json:"ConfiguredNetworkLinks"`
						Actions struct {
							HpeNetworkPortDiagnosticsReset struct {
								Title  string `json:"title"`
								Target string `json:"target"`
							} `json:"#HpeNetworkPort.DiagnosticsReset"`
							NetworkPortDiagnosticsReset struct {
								Title  string `json:"title"`
								Target string `json:"target"`
							} `json:"#NetworkPort.DiagnosticsReset"`
						} `json:"Actions"`
						OdataType string `json:"@odata.type"`
						Lldp      struct {
							OdataID string `json:"odataId"`
						} `json:"Lldp"`
						VirtualFunctionAllocations []struct {
							VirtualFunctionsAllocated int `json:"VirtualFunctionsAllocated"`
							NetworkDeviceFunction      struct {
								OdataID string `json:"odataId"`
							} `json:"NetworkDeviceFunction"`
						} `json:"VirtualFunctionAllocations"`
						OdataContext           string `json:"@odata.context"`
						AutoNegotiationCapable bool   `json:"AutoNegotiationCapable"`
						Evb                    struct {
							OdataID string `json:"odataId"`
						} `json:"Evb"`
						PortDiagnosticEnableLocalLoopback bool `json:"PortDiagnosticEnableLocalLoopback"`
						EvbData                            struct {
							Receiving struct {
								CdcpActive bool `json:"CdcpActive"`
								ChannelLinkControlRemoteSequenceNumber int `json:"ChannelLinkControlRemoteSequenceNumber"`

								ChannelDescriptionLocalSequenceNumber  int    `json:"ChannelDescriptionLocalSequenceNumber"`
								ChannelLinkControlSubtype              int    `json:"ChannelLinkControlSubtype"`
								ChannelActive                          bool   `json:"ChannelActive"`
								ChannelDescriptionRemoteSequenceNumber int    `json:"ChannelDescriptionRemoteSequenceNumber"`
								ChannelLinkControlLocalSequenceNumber  int    `json:"ChannelLinkControlLocalSequenceNumber"`
								SChannelAssignments                    []struct {
									Svid int `json:"Svid"`
									Scid int `json:"Scid"`
								} `json:"SChannelAssignments"`
								Role                        string `json:"Role"`
								ChannelDescriptionSubtype   int    `json:"ChannelDescriptionSubtype"`
								SChannelMaxCount            int    `json:"SChannelMaxCount"`
								SChannelsSupported          bool   `json:"SChannelsSupported"`
								ChannelLinkControlMCEnable  bool   `json:"ChannelLinkControlMCEnable"`
								VirtualLinkStatusVector     string `json:"VirtualLinkStatusVector"`
							} `json:"Receiving"`
							Transmitting struct {
								Role             string `json:"Role"`
								SChannelMaxCount int    `json:"SChannelMaxCount"`
								ChannelLinkControlRemoteSequenceNumber int `json:"ChannelLinkControlRemoteSequenceNumber"`
								ChannelDescriptionData []struct {
									Descriptor1Scid                    int    `json:"Descriptor1Scid"`
									Descriptor0Scid                    int    `json:"Descriptor0Scid"`
									Descriptor0BandwidthPirMbps        int    `json:"Descriptor0BandwidthPirMbps"`
									Descriptor0ChannelTerminationType string `json:"Descriptor0ChannelTerminationType"`
									Descriptor0FCoESupport             bool   `json:"Descriptor0FCoESupport"`
									Descriptor1ConnectionID             any    `json:"Descriptor1ConnectionID"`
									Descriptor0ISCSISupport             bool   `json:"Descriptor0iSCSISupport"`
								} `json:"ChannelDescriptionData"`
								ChannelDescriptionLocalSequenceNumber  int    `json:"ChannelDescriptionLocalSequenceNumber"`
								SChannelsSupported                     bool   `json:"SChannelsSupported"`
								ChannelDescriptionRemoteSequenceNumber int    `json:"ChannelDescriptionRemoteSequenceNumber"`
								ChannelLinkControlMCEnable             bool   `json:"ChannelLinkControlMCEnable"`
								ChannelDescriptorCount                 int    `json:"ChannelDescriptorCount"`
								ChannelLinkControlLocalSequenceNumber  int    `json:"ChannelLinkControlLocalSequenceNumber"`
								VirtualLinkStatusVector                string `json:"VirtualLinkStatusVector"`
								SChannelAssignments                    []struct {
									Svid int `json:"Svid"`
									Scid int `json:"Scid"`
								} `json:"SChannelAssignments"`
							} `json:"Transmitting"`
						} `json:"EvbData"`
						PortDiagnosticLocalLoopbackCapable bool `json:"PortDiagnosticLocalLoopbackCapable"`
						PortDiagnosticsEnabled              bool `json:"PortDiagnosticsEnabled"`
						LldpData                             struct {
							Receiving struct {
								ManagementIFNumSubtype string `json:"ManagementIFNumSubtype"`
								ChassisID              string `json:"ChassisID"`
								Dcb                    struct {

								ApplicationPriorityTable []struct {
									ApplicationProtocol int    `json:"ApplicationProtocol"`
									ApplicationSelect   string `json:"ApplicationSelect"`
									ApplicationPriority int    `json:"ApplicationPriority"`
								} `json:"ApplicationPriorityTable"`
								Ets struct {
									PriorityAssignmentTable      []int    `json:"PriorityAssignmentTable"`
									MaximumTrafficClassCount     int      `json:"MaximumTrafficClassCount"`
									BandwidthAssignmentTable     []int    `json:"BandwidthAssignmentTable"`
									TlvReceived                  bool     `json:"TlvReceived"`
									CreditBasedShaper            bool     `json:"CreditBasedShaper"`
									TrafficSelectionAlgorithmTable []string `json:"TrafficSelectionAlgorithmTable"`
									Willing                       bool     `json:"Willing"`
								} `json:"Ets"`
								Version                   string `json:"Version"`
								LocalEqualRemote          bool   `json:"LocalEqualRemote"`
								ApplicationPriorityReceived bool `json:"ApplicationPriorityReceived"`
								Pfc                       struct {
									EnabledTrafficClasses     string `json:"EnabledTrafficClasses"`
									MaximumTrafficClassCount  int    `json:"MaximumTrafficClassCount"`
									TlvReceived               bool   `json:"TlvReceived"`
									MacSecurityBypassCapability bool `json:"MacSecurityBypassCapability"`
									Willing                   bool   `json:"Willing"`
								} `json:"Pfc"`
							} `json:"Dcb"`
							ManagementAddress       string `json:"ManagementAddress"`
							SystemDescription       string `json:"SystemDescription"`
							ManagementVlanReceived  bool   `json:"ManagementVlanReceived"`
							PortDescription         string `json:"PortDescription"`
							PortID                  string `json:"PortID"`
							ManagementAddressType   string `json:"ManagementAddressType"`
							PortIDSubtype           string `json:"PortIDSubtype"`
							ChassisIDSubtype        string `json:"ChassisIDSubtype"`
							NearestBridgeTTLExpired bool   `json:"NearestBridgeTtlExpired"`
							ManagementIFNumber      string `json:"ManagementIFNumber"`
							ManagementAddressReceived bool `json:"ManagementAddressReceived"`
						} `json:"Receiving"`
						Transmitting struct {
							ChassisIDSubtype  string `json:"ChassisIDSubtype"`
							ManagementAddresses []struct {
								ManagementIPAddressType string `json:"ManagementIPAddressType"`
								ManagementIPAddress     string `json:"ManagementIPAddress"`
							} `json:"ManagementAddresses"`
							ChassisID              string `json:"ChassisID"`
							SystemDescription      string `json:"SystemDescription"`
							PortDescription        string `json:"PortDescription"`
							PortID                 string `json:"PortID"`
							ManagementCvlanIDTransmitted bool `json:"ManagementCvlanIDTransmitted"`
							PortIDSubtype          string `json:"PortIDSubtype"`

						} `json:"Transmitting"`
					} `json:"LldpData"`
				} `json:"Hpe"`
			} `json:"Oem"`
			Ethernet struct {
				FlowControlStatus       any      `json:"FlowControlStatus"`
				AssociatedMACAddresses  []string `json:"AssociatedMACAddresses"`
				FlowControlConfiguration string  `json:"FlowControlConfiguration"`
				SupportedEthernetCapabilities []string `json:"SupportedEthernetCapabilities"`
			} `json:"Ethernet"`
			RedfishSettings struct {
				SettingsObject struct {
					OdataID string `json:"@odata.id"`
				} `json:"SettingsObject"`
			} `json:"@Redfish.Settings"`
			ID                 string `json:"Id"`
			FunctionMaxBandwidth []struct {
				AllocationPercent    int `json:"AllocationPercent"`
				NetworkDeviceFunction struct {
					OdataID string `json:"@odata.id"`
				} `json:"NetworkDeviceFunction"`
			} `json:"FunctionMaxBandwidth"`
			Settings struct {
				Href string `json:"href"`
			} `json:"Settings"`
		} `json:"Ports"`
		OdataContext string `json:"@odata.context"`
		Name         string `json:"Name"`
		OdataID      string `json:"@odata.id"`
		NetworkDeviceFunctions []struct {
			Description                string `json:"Description"`
			OdataType                  string `json:"@odata.type"`
			AssignablePhysicalNetworkPorts []struct {
				OdataID string `json:"@odata.id"`
			} `json:"AssignablePhysicalNetworkPorts"`
			BootMode      string `json:"BootMode"`
			DeviceEnabled bool   `json:"DeviceEnabled"`
			OdataContext  string `json:"@odata.context"`
			PhysicalPortAssignment struct {
				OdataID string `json:"@odata.id"`
			} `json:"PhysicalPortAssignment"`
			Name    string `json:"Name"`
			OdataID string `json:"@odata.id"`
			AssignablePhysicalPorts []struct {
				OdataID string `json:"@odata.id"`
			} `json:"AssignablePhysicalPorts"`
			PhysicalNetworkPortAssignment struct {
				OdataID string `json:"@odata.id"`
			} `json:"PhysicalNetworkPortAssignment"`

			VirtualFunctionsEnabled bool   `json:"VirtualFunctionsEnabled"`
			NetDevFuncType          string `json:"NetDevFuncType"`
			Oem                     struct {
				Hpe struct {
					ConnectionID        string   `json:"ConnectionID"`
					VirtualLinkStatus   string   `json:"VirtualLinkStatus"`
					NetDevFuncType      string   `json:"NetDevFuncType"`
					OdataType           string   `json:"@odata.type"`
					WakeOnLANEnabled    bool     `json:"WakeOnLANEnabled"`
					OdataContext        string   `json:"@odata.context"`
					BootTargetSource    string   `json:"BootTargetSource"`
					NetDevFuncCapabilities []string `json:"NetDevFuncCapabilities"`
				} `json:"Hpe"`
			} `json:"Oem"`
			Ethernet struct {
				MACAddress          string `json:"MACAddress"`
				PermanentMACAddress string `json:"PermanentMACAddress"`
			} `json:"Ethernet"`
			RedfishSettings struct {
				SettingsObject struct {
					OdataID string `json:"@odata.id"`
				} `json:"SettingsObject"`
			} `json:"@Redfish.Settings"`
			Links struct {
				PhysicalNetworkPortAssignment struct {
					OdataID string `json:"@odata.id"`
				} `json:"PhysicalNetworkPortAssignment"`
				PhysicalPortAssignment struct {
					OdataID string `json:"@odata.id"`
				} `json:"PhysicalPortAssignment"`
			} `json:"Links"`
			MaxVirtualFunctions   int      `json:"MaxVirtualFunctions"`
			ID                    string   `json:"Id"`
			NetDevFuncCapabilities []string `json:"NetDevFuncCapabilities"`
			Settings              struct {
				Href string `json:"href"`
			} `json:"Settings"`
			FibreChannel struct {
				Wwpn        string `json:"WWPN"`
				Wwnn        string `json:"WWNN"`
				BootTargets []struct {
					BootPriority any `json:"BootPriority"`
					Lunid        any `json:"LUNID"`
					Wwpn         any `json:"WWPN"`
				} `json:"BootTargets"`
			} `json:"FibreChannel,omitempty"`
			ISCSIBoot struct {
				PrimaryTargetIPAddress string `json:"PrimaryTargetIPAddress"`
				IPMaskDNSViaDHCP       bool   `json:"IPMaskDNSViaDHCP"`

				InitiatorName          string `json:"InitiatorName"`
				SecondaryTargetIPAddress string `json:"SecondaryTargetIPAddress"`
				InitiatorDefaultGateway string `json:"InitiatorDefaultGateway"`
				AuthenticationMethod   string `json:"AuthenticationMethod"`
				TargetInfoViaDHCP      bool   `json:"TargetInfoViaDHCP"`
				MutualCHAPUsername     string `json:"MutualCHAPUsername"`
				SecondaryDNS           string `json:"SecondaryDNS"`
				CHAPSecret             any    `json:"CHAPSecret"`
				MutualCHAPSecret       any    `json:"MutualCHAPSecret"`
				InitiatorNetmask       string `json:"InitiatorNetmask"`
				PrimaryDNS             string `json:"PrimaryDNS"`
				IPAddressType          string `json:"IPAddressType"`
				CHAPUsername           string `json:"CHAPUsername"`
				PrimaryLUN             int    `json:"PrimaryLUN"`
				SecondaryTargetTCPPort int    `json:"SecondaryTargetTCPPort"`
				InitiatorIPAddress     string `json:"InitiatorIPAddress"`
				PrimaryTargetTCPPort   int    `json:"PrimaryTargetTCPPort"`
				PrimaryTargetName      string `json:"PrimaryTargetName"`
			} `json:"iSCSIBoot,omitempty"`
		} `json:"NetworkDeviceFunctions"`
		SerialNumber string `json:"SerialNumber"`
		Oem          struct {
			Hpe struct {
				Controllers []struct {
					FunctionTypeLimits struct {
						FCoEResourcesConsumed       int    `json:"FCoEResourcesConsumed"`
						ISCSIResourcesConsumed      int    `json:"iSCSIResourcesConsumed"`
						ConstraintDescription       string `json:"ConstraintDescription"`
						TotalSharedResourcesAvailable int  `json:"TotalSharedResourcesAvailable"`
					} `json:"FunctionTypeLimits"`
					DeviceLimitationsBitmap int `json:"DeviceLimitationsBitmap"`
					ConfigurationStatus     struct {
						Summary int `json:"Summary"`
						Detail  []struct {
							Group    int `json:"Group"`
							Summary  int `json:"Summary"`
							SubGroup int `json:"SubGroup"`
						} `json:"Detail"`
					} `json:"ConfigurationStatus"`
					MostRecentConfigurationChangeSource string `json:"MostRecentConfigurationChangeSource"`
					EdgeVirtualBridging struct {
						ChannelDescriptionTLVCapable bool `json:"ChannelDescriptionTLVCapable"`
						ChannelLinkControlTLVCapable bool `json:"ChannelLinkControlTLVCapable"`
					} `json:"EdgeVirtualBridging"`
					RDMASupport         []string `json:"RDMASupport"`
					EmbeddedLLDPFunctions struct {
						Optional bool `json:"Optional"`
						Enabled  bool `json:"Enabled"`
					} `json:"EmbeddedLLDPFunctions"`

					FunctionTypes         []string `json:"FunctionTypes"`
					UnderlyingDataSource  string   `json:"UnderlyingDataSource"`
				} `json:"Controllers"`
				CLPVersion string `json:"CLPVersion"`
				Actions    struct {
					HpeNetworkAdapterFlushConfigurationToNVM struct {
						Title  string `json:"title"`
						Target string `json:"target"`
					} `json:"#HpeNetworkAdapter.FlushConfigurationToNVM"`
					NetworkAdapterFlushConfigurationToNVM struct {
						Title  string `json:"title"`
						Target string `json:"target"`
					} `json:"#NetworkAdapter.FlushConfigurationToNVM"`
				} `json:"Actions"`
				FactoryDefaultsActuationBehavior string `json:"FactoryDefaultsActuationBehavior"`
				OdataType                        string `json:"@odata.type"`
				RedfishConfiguration             string `json:"RedfishConfiguration"`
				OdataContext                     string `json:"@odata.context"`
				PCAVersion                       string `json:"PCAVersion"`
			} `json:"Hpe"`
		} `json:"Oem"`
		RedfishSettings struct {
			SettingsObject struct {
				OdataID string `json:"@odata.id"`
			} `json:"SettingsObject"`
		} `json:"@Redfish.Settings"`
		Manufacturer string `json:"Manufacturer"`
		Model        string `json:"Model"`
		ID           string `json:"Id"`
		Sku          string `json:"SKU"`
		Settings     struct {
			Href string `json:"href"`
		} `json:"Settings"`
	} `json:"data"`
	Etag  string `json:"etag"`
	Count int    `json:"count"`
	Name  string `json:"name"`
} `json:"NetworkAdapters"`
Processors struct {
	URI             string    `json:"uri"`
	CollectionState string    `json:"collectionState"`
	Modified        time.Time `json:"modified"`
	Data            []struct {
		Status struct {
			Health string `json:"Health"`
			State  string `json:"State"`
		} `json:"Status"`
		PartNumber   string `json:"PartNumber"`
		TotalThreads int    `json:"TotalThreads"`

		ProcessorArchitecture string `json:"ProcessorArchitecture"`
		ProcessorID           struct {
			IdentificationRegisters string `json:"IdentificationRegisters"`
			VendorID                string `json:"VendorId"`
			MicrocodeInfo           any    `json:"MicrocodeInfo"`
			Step                    string `json:"Step"`
			EffectiveModel          string `json:"EffectiveModel"`
			EffectiveFamily         string `json:"EffectiveFamily"`
		} `json:"ProcessorId"`
		ProcessorType string `json:"ProcessorType"`
		OdataType     string `json:"@odata.type"`
		TotalCores    int    `json:"TotalCores"`
		OdataContext  string `json:"@odata.context"`
		InstructionSet string `json:"InstructionSet"`
		Name          string `json:"Name"`
		OdataID       string `json:"@odata.id"`
		SerialNumber  string `json:"SerialNumber"`
		Oem           struct {
			Hpe struct {
				MicrocodePatches []struct {
					CPUID   string    `json:"CpuId"`
					PatchID string    `json:"PatchId"`
					Date    time.Time `json:"Date"`
				} `json:"MicrocodePatches"`
				PartNumber      string `json:"PartNumber"`
				OdataType       string `json:"@odata.type"`
				CoresEnabled    int    `json:"CoresEnabled"`
				OdataContext    string `json:"@odata.context"`
				RatedSpeedMHz   int    `json:"RatedSpeedMHz"`
				AssetTag        string `json:"AssetTag"`
				ExternalClockMHz int   `json:"ExternalClockMHz"`
				SerialNumber    string `json:"SerialNumber"`
				Characteristics []string `json:"Characteristics"`
				VoltageVoltsX10 int    `json:"VoltageVoltsX10"`
				ConfigStatus    struct {
					State     string `json:"State"`
					Populated bool   `json:"Populated"`
				} `json:"ConfigStatus"`
				Cache []struct {
					Policy           string   `json:"Policy"`
					InstalledSizeKB  int      `json:"InstalledSizeKB"`
					CurrentSRAMType  []string `json:"CurrentSRAMType"`
					SupportedSRAMType []string `json:"SupportedSRAMType"`
					EccType          string   `json:"EccType"`
					SystemCacheType  string   `json:"SystemCacheType"`
					CacheSpeedns     int      `json:"CacheSpeedns"`
					MaximumSizeKB    int      `json:"MaximumSizeKB"`
					Socketed         bool     `json:"Socketed"`
					Associativity    string   `json:"Associativity"`

						Location string `json:"Location"`
						Name     string `json:"Name"`
					} `json:"Cache"`
				} `json:"Hpe"`
			} `json:"Oem"`
			Manufacturer string `json:"Manufacturer"`
			MaxSpeedMHz  int    `json:"MaxSpeedMHz"`
			Model        string `json:"Model"`
			ID           string `json:"Id"`
			Socket       string `json:"Socket"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Processors"`
} `json:"subResources,omitempty"`
SupportDataCollectionState any    `json:"supportDataCollectionState"`
SupportDataCollectionType  any    `json:"supportDataCollectionType"`
SupportState               string `json:"supportState"`
UIDState                   string `json:"uidState"`
URI                        string `json:"uri"`
UUID                       string `json:"uuid"`
VirtualSerialNumber        string `json:"virtualSerialNumber"`
VirtualUUID                string `json:"virtualUuid"`
MpDNSName                  string `json:"mpDnsName"`
ID                         string `json:"id"`
OriginalURI                string `json:"originalUri"`
Appluri                    string `json:"appluri"`
ApplianceName              string `json:"applianceName"`
ApplianceLocation          string `json:"applianceLocation"`
MpFirmwareVersionNumber    int64  `json:"mpFirmwareVersionNumber"`
RomVersionNumber           float64 `json:"romVersionNumber"`
MpIPAddresses              string `json:"_mpIpAddresses"`
ServerHardwareTypeName     string `json:"serverHardwareTypeName"`
ServerProfileName          string `json:"serverProfileName"`
MpIPAddress                string `json:"mpIpAddress"`
OrderedStatus              int    `json:"_orderedStatus"`
SubResources0              struct {
	Devices struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				Health any `json:"Health"`
				State  any `json:"State"`
			} `json:"Status,omitempty"`
			PartNumber string `json:"PartNumber"`
			OdataType  string `json:"@odata.type"`

			OdataEtag       string `json:"@odata.etag"`
			ProductVersion  string `json:"ProductVersion"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			Name              string `json:"Name"`
			DeviceType        string `json:"DeviceType"`
			SerialNumber      string `json:"SerialNumber"`
			ProductPartNumber string `json:"ProductPartNumber"`
			Manufacturer      string `json:"Manufacturer"`
			ID                string `json:"Id"`
			Location          string `json:"Location"`
			Status0           struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
			MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Devices"`
	LocalStorage struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
			CurrentOperatingMode            string `json:"CurrentOperatingMode"`
			LocationFormat                  string `json:"LocationFormat"`
			EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
			AdapterType                     string `json:"AdapterType"`
			PhysicalDrives                  []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LocationFormat      string `json:"LocationFormat"`
				CapacityLogicalBlocks int  `json:"CapacityLogicalBlocks"`
				BlockSizeBytes      int    `json:"BlockSizeBytes"`
				CapacityMiB         int    `json:"CapacityMiB"`
				FirmwareVersion     struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				InterfaceType string `json:"InterfaceType"`
				LegacyBootPriority string `json:"LegacyBootPriority"`

				SerialNumber           string   `json:"SerialNumber"`
				DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
				Model                  string   `json:"Model"`
				MediaType              string   `json:"MediaType"`
				RotationalSpeedRpm     int      `json:"RotationalSpeedRpm"`
				DiskDriveUse           string   `json:"DiskDriveUse"`
				EncryptedDrive         bool     `json:"EncryptedDrive"`
				InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
				Location               string   `json:"Location"`
			} `json:"PhysicalDrives"`
			LogicalDrives []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LogicalDriveEncryption bool `json:"LogicalDriveEncryption"`
				DataDrives             []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat        string `json:"LocationFormat"`
					CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
					BlockSizeBytes        int    `json:"BlockSizeBytes"`
					CapacityMiB           int    `json:"CapacityMiB"`
					FirmwareVersion       struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceType         string   `json:"InterfaceType"`
					LegacyBootPriority    string   `json:"LegacyBootPriority"`
					SerialNumber          string   `json:"SerialNumber"`
					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					Model                 string   `json:"Model"`
					MediaType             string   `json:"MediaType"`
					RotationalSpeedRpm    int      `json:"RotationalSpeedRpm"`
					DiskDriveUse          string   `json:"DiskDriveUse"`
					EncryptedDrive        bool     `json:"EncryptedDrive"`
					InterfaceSpeedMbps    int      `json:"InterfaceSpeedMbps"`
					Location              string   `json:"Location"`
				} `json:"DataDrives"`
				StripeSizeBytes    int    `json:"StripeSizeBytes"`
				Raid               string `json:"Raid"`
				CapacityMiB        int    `json:"CapacityMiB"`
				LogicalDriveType   string `json:"LogicalDriveType"`
				LogicalDriveNumber int    `json:"LogicalDriveNumber"`
				AccelerationMethod string `json:"AccelerationMethod"`
				InterfaceType      string `json:"InterfaceType"`

				LegacyBootPriority     string   `json:"LegacyBootPriority"`
				LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
				MediaType              string   `json:"MediaType"`
				VolumeUniqueIdentifier string   `json:"VolumeUniqueIdentifier"`
				LogicalDriveName       string   `json:"LogicalDriveName"`
			} `json:"LogicalDrives"`
			InternalPortCount  int    `json:"InternalPortCount"`
			CacheMemorySizeMiB int    `json:"CacheMemorySizeMiB"`
			EncryptionFwLocked bool   `json:"EncryptionFwLocked"`
			Name               string `json:"Name"`
			BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
			DriveWriteCache    string `json:"DriveWriteCache"`
			Status             struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			EncryptionSelfTestPassed bool `json:"EncryptionSelfTestPassed"`
			ExternalPortCount        int  `json:"ExternalPortCount"`
			ControllerBoard          struct {
				Status struct {
					Health string `json:"Health"`
				} `json:"Status"`
			} `json:"ControllerBoard"`
			EncryptionEnabled bool `json:"EncryptionEnabled"`
			EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
			EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
			SerialNumber                       string `json:"SerialNumber"`
			CacheModuleStatus                  struct {
				Health string `json:"Health"`
			} `json:"CacheModuleStatus"`
			StorageEnclosures []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LocationFormat string `json:"LocationFormat"`
				DriveBayCount  int    `json:"DriveBayCount"`
				ID             string `json:"Id"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`

				Location string `json:"Location"`
			} `json:"StorageEnclosures"`
			CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
			Model                   string `json:"Model"`
			Location                string `json:"Location"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"LocalStorage"`
	Memory struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			PersistentRegionSizeLimitMiB int    `json:"PersistentRegionSizeLimitMiB"`
			OdataType                    string `json:"@odata.type"`
			LogicalSizeMiB               int    `json:"LogicalSizeMiB"`
			RankCount                    any    `json:"RankCount"`
			VendorID                     string `json:"VendorID"`
			OdataEtag                    string `json:"@odata.etag"`
			SecurityCapabilities         struct {
			} `json:"SecurityCapabilities"`
			BusWidthBits     int      `json:"BusWidthBits"`
			CacheSizeMiB     int      `json:"CacheSizeMiB"`
			DataWidthBits    int      `json:"DataWidthBits"`
			CapacityMiB      int      `json:"CapacityMiB"`
			VolatileSizeMiB  int      `json:"VolatileSizeMiB"`
			Name             string   `json:"Name"`
			NonVolatileSizeMiB int    `json:"NonVolatileSizeMiB"`
			ErrorCorrection  string   `json:"ErrorCorrection"`
			MemoryMedia      []string `json:"MemoryMedia"`
			Oem              struct {
				Hpe struct {
					DIMMStatus             string `json:"DIMMStatus"`
					OdataType              string `json:"@odata.type"`
					MinimumVoltageVoltsX10 int    `json:"MinimumVoltageVoltsX10"`
				} `json:"Hpe"`
			} `json:"Oem"`
			OperatingMemoryModes    []string `json:"OperatingMemoryModes"`
			VolatileRegionSizeLimitMiB int   `json:"VolatileRegionSizeLimitMiB"`
			ID                      string `json:"Id"`
			MemoryLocation          struct {
				Channel           int `json:"Channel"`
				Slot              int `json:"Slot"`

				MemoryController int `json:"MemoryController"`
				Socket           int `json:"Socket"`
			} `json:"MemoryLocation"`
			DeviceLocator     string `json:"DeviceLocator"`
			MemoryType        string `json:"MemoryType"`
			PartNumber        string `json:"PartNumber,omitempty"`
			OperatingSpeedMhz int    `json:"OperatingSpeedMhz,omitempty"`
			BaseModuleType    string `json:"BaseModuleType,omitempty"`
			Manufacturer      string `json:"Manufacturer,omitempty"`
			MemoryDeviceType  string `json:"MemoryDeviceType,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Memory"`
	MemoryList struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
			BoardCPUNumber            int `json:"BoardCpuNumber"`
			BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
			BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
			BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"MemoryList"`
	LocalStorageV2 struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []any     `json:"data"`
		Etag            string    `json:"etag"`
		Count           int       `json:"count"`
		Name            string    `json:"name"`
	} `json:"LocalStorageV2"`
	AdvancedMemoryProtection struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			AmpModeActive    string   `json:"AmpModeActive"`
			OdataType        string   `json:"@odata.type"`
			AmpModeSupported []string `json:"AmpModeSupported"`
			AmpModeStatus    string   `json:"AmpModeStatus"`
		} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"AdvancedMemoryProtection"`
		} `json:"subResources,omitempty"`
		SubResources1 struct {
		LocalStorage struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
				CurrentOperatingMode            string `json:"CurrentOperatingMode"`
				LocationFormat                  string `json:"LocationFormat"`
				EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
				AdapterType                     string `json:"AdapterType"`
				PhysicalDrives                  []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat            string `json:"LocationFormat"`
					CapacityLogicalBlocks     int    `json:"CapacityLogicalBlocks"`
					PowerOnHours              int    `json:"PowerOnHours"`
					BlockSizeBytes            int    `json:"BlockSizeBytes"`
					SSDEnduranceUtilizationPercentage int `json:"SSDEnduranceUtilizationPercentage"`
					CapacityMiB               int    `json:"CapacityMiB"`
					FirmwareVersion           struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceType      string   `json:"InterfaceType"`
					LegacyBootPriority string   `json:"LegacyBootPriority"`
					SerialNumber       string   `json:"SerialNumber"`
					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					Model              string   `json:"Model"`
					MediaType          string   `json:"MediaType"`
					DiskDriveUse       string   `json:"DiskDriveUse"`
					EncryptedDrive     bool     `json:"EncryptedDrive"`
					InterfaceSpeedMbps int      `json:"InterfaceSpeedMbps"`
					Location           string   `json:"Location"`
				} `json:"PhysicalDrives"`
				LogicalDrives []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LogicalDriveEncryption bool `json:"LogicalDriveEncryption"`

					DataDrives []struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						LocationFormat        string `json:"LocationFormat"`
						CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
						PowerOnHours          int    `json:"PowerOnHours"`
						BlockSizeBytes        int    `json:"BlockSizeBytes"`
						SSDEnduranceUtilizationPercentage int `json:"SSDEnduranceUtilizationPercentage"`
						CapacityMiB           int    `json:"CapacityMiB"`
						FirmwareVersion       struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"FirmwareVersion"`
						InterfaceType         string   `json:"InterfaceType"`
						LegacyBootPriority    string   `json:"LegacyBootPriority"`
						SerialNumber          string   `json:"SerialNumber"`
						DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
						Model                 string   `json:"Model"`
						MediaType             string   `json:"MediaType"`
						DiskDriveUse          string   `json:"DiskDriveUse"`
						EncryptedDrive        bool     `json:"EncryptedDrive"`
						InterfaceSpeedMbps    int      `json:"InterfaceSpeedMbps"`
						Location              string   `json:"Location"`
					} `json:"DataDrives"`
					StripeSizeBytes        int    `json:"StripeSizeBytes"`
					Raid                   string `json:"Raid"`
					CapacityMiB            int    `json:"CapacityMiB"`
					LogicalDriveType       string `json:"LogicalDriveType"`
					LogicalDriveNumber     int    `json:"LogicalDriveNumber"`
					AccelerationMethod     string `json:"AccelerationMethod"`
					InterfaceType          string `json:"InterfaceType"`
					LegacyBootPriority     string `json:"LegacyBootPriority"`
					LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
					MediaType              string `json:"MediaType"`
					VolumeUniqueIdentifier string `json:"VolumeUniqueIdentifier"`
					LogicalDriveName       string `json:"LogicalDriveName"`
				} `json:"LogicalDrives"`
				InternalPortCount       int    `json:"InternalPortCount"`
				CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
				EncryptionFwLocked      bool   `json:"EncryptionFwLocked"`
				Name                    string `json:"Name"`
				BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
				DriveWriteCache         string `json:"DriveWriteCache"`
				Status                  struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`

				EncryptionSelfTestPassed bool `json:"EncryptionSelfTestPassed"`
				ExternalPortCount        int  `json:"ExternalPortCount"`
				ControllerBoard          struct {
					Status struct {
						Health string `json:"Health"`
					} `json:"Status"`
				} `json:"ControllerBoard"`
				EncryptionEnabled bool `json:"EncryptionEnabled"`
				EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
				EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
				SerialNumber                       string `json:"SerialNumber"`
				StorageEnclosures                  []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat string `json:"LocationFormat"`
					DriveBayCount  int    `json:"DriveBayCount"`
					ID             string `json:"Id"`
					FirmwareVersion struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					Location string `json:"Location"`
				} `json:"StorageEnclosures"`
				CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
				Model                   string `json:"Model"`
				Location                string `json:"Location"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"LocalStorage"`
		Memory struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
			} `json:"data"`

				PersistentRegionSizeLimitMiB int    `json:"PersistentRegionSizeLimitMiB"`
				OdataType                    string `json:"@odata.type"`
				LogicalSizeMiB               int    `json:"LogicalSizeMiB"`
				RankCount                    any    `json:"RankCount"`
				VendorID                     string `json:"VendorID"`
				OdataEtag                    string `json:"@odata.etag"`
				SecurityCapabilities         struct {
				} `json:"SecurityCapabilities"`
				BusWidthBits       int      `json:"BusWidthBits"`
				CacheSizeMiB       int      `json:"CacheSizeMiB"`
				DataWidthBits      int      `json:"DataWidthBits"`
				CapacityMiB        int      `json:"CapacityMiB"`
				VolatileSizeMiB    int      `json:"VolatileSizeMiB"`
				Name               string   `json:"Name"`
				NonVolatileSizeMiB int      `json:"NonVolatileSizeMiB"`
				ErrorCorrection    string   `json:"ErrorCorrection"`
				MemoryMedia        []string `json:"MemoryMedia"`
				Oem                struct {
					Hpe struct {
						DIMMStatus             string `json:"DIMMStatus"`
						OdataType              string `json:"@odata.type"`
						MinimumVoltageVoltsX10 int    `json:"MinimumVoltageVoltsX10"`
					} `json:"Hpe"`
				} `json:"Oem"`
				OperatingMemoryModes      []string `json:"OperatingMemoryModes"`
				VolatileRegionSizeLimitMiB int     `json:"VolatileRegionSizeLimitMiB"`
				ID                        string   `json:"Id"`
				MemoryLocation            struct {
					Channel          int `json:"Channel"`
					Slot             int `json:"Slot"`
					MemoryController int `json:"MemoryController"`
					Socket           int `json:"Socket"`
				} `json:"MemoryLocation"`
				DeviceLocator     string `json:"DeviceLocator"`
				MemoryType        string `json:"MemoryType"`
				PartNumber        string `json:"PartNumber,omitempty"`
				OperatingSpeedMhz int    `json:"OperatingSpeedMhz,omitempty"`
				BaseModuleType    string `json:"BaseModuleType,omitempty"`
				Manufacturer      string `json:"Manufacturer,omitempty"`
				MemoryDeviceType  string `json:"MemoryDeviceType,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Memory"`
		MemoryList struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`

			Data []struct {
				BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
				BoardCPUNumber            int `json:"BoardCpuNumber"`
				BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
				BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
				BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"MemoryList"`
		AdvancedMemoryProtection struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				AmpModeActive    string   `json:"AmpModeActive"`
				OdataType        string   `json:"@odata.type"`
				AmpModeSupported []string `json:"AmpModeSupported"`
				AmpModeStatus    string   `json:"AmpModeStatus"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"AdvancedMemoryProtection"`
		LocalStorageV2 struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []any     `json:"data"`
			Etag            string    `json:"etag"`
			Count           int       `json:"count"`
			Name            string    `json:"name"`
		} `json:"LocalStorageV2"`
		Devices struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
				PartNumber      string `json:"PartNumber"`
				OdataType       string `json:"@odata.type"`
				OdataEtag       string `json:"@odata.etag"`
				ProductVersion  string `json:"ProductVersion"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`

				Name              string `json:"Name"`
				DeviceType        string `json:"DeviceType"`
				SerialNumber      string `json:"SerialNumber"`
				ProductPartNumber string `json:"ProductPartNumber"`
				Manufacturer      string `json:"Manufacturer"`
				ID                string `json:"Id"`
				Location          string `json:"Location"`
				Status0           struct {
					State string `json:"State"`
				} `json:"Status,omitempty"`
				MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
				Status1              struct {
					State string `json:"State"`
				} `json:"Status,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Devices"`
		SubResources2 struct {
		Devices struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
				PartNumber      string `json:"PartNumber"`
				OdataType       string `json:"@odata.type"`
				OdataEtag       string `json:"@odata.etag"`
				ProductVersion  string `json:"ProductVersion"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				Name              string `json:"Name"`
				DeviceType        string `json:"DeviceType"`
				SerialNumber      string `json:"SerialNumber"`
				ProductPartNumber string `json:"ProductPartNumber"`
				Manufacturer      string `json:"Manufacturer"`
				ID                string `json:"Id"`

				Location string `json:"Location"`
				Status0  struct {
					State string `json:"State"`
				} `json:"Status,omitempty"`
				MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
				Status1              struct {
					State string `json:"State"`
				} `json:"Status,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Devices"`
		Memory struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				PersistentRegionSizeLimitMiB int    `json:"PersistentRegionSizeLimitMiB"`
				OdataType                    string `json:"@odata.type"`
				LogicalSizeMiB               int    `json:"LogicalSizeMiB"`
				RankCount                    any    `json:"RankCount"`
				VendorID                     string `json:"VendorID"`
				OdataEtag                    string `json:"@odata.etag"`
				SecurityCapabilities         struct {
				} `json:"SecurityCapabilities"`
				BusWidthBits       int      `json:"BusWidthBits"`
				CacheSizeMiB       int      `json:"CacheSizeMiB"`
				DataWidthBits      int      `json:"DataWidthBits"`
				CapacityMiB        int      `json:"CapacityMiB"`
				VolatileSizeMiB    int      `json:"VolatileSizeMiB"`
				Name               string   `json:"Name"`
				NonVolatileSizeMiB int      `json:"NonVolatileSizeMiB"`
				ErrorCorrection    string   `json:"ErrorCorrection"`
				MemoryMedia        []string `json:"MemoryMedia"`
				Oem                struct {
					Hpe struct {
						DIMMStatus             string `json:"DIMMStatus"`
						OdataType              string `json:"@odata.type"`
						MinimumVoltageVoltsX10 int    `json:"MinimumVoltageVoltsX10"`
					} `json:"Hpe"`
				} `json:"Oem"`
				OperatingMemoryModes       []string `json:"OperatingMemoryModes"`
				VolatileRegionSizeLimitMiB int      `json:"VolatileRegionSizeLimitMiB"`
				ID                         string   `json:"Id"`

				MemoryLocation struct {
					Channel          int `json:"Channel"`
					Slot             int `json:"Slot"`
					MemoryController int `json:"MemoryController"`
					Socket           int `json:"Socket"`
				} `json:"MemoryLocation"`
				DeviceLocator     string `json:"DeviceLocator"`
				MemoryType        string `json:"MemoryType"`
				PartNumber        string `json:"PartNumber,omitempty"`
				OperatingSpeedMhz int    `json:"OperatingSpeedMhz,omitempty"`
				BaseModuleType    string `json:"BaseModuleType,omitempty"`
				Manufacturer      string `json:"Manufacturer,omitempty"`
				MemoryDeviceType  string `json:"MemoryDeviceType,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Memory"`
		LocalStorage struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
				CurrentOperatingMode            string `json:"CurrentOperatingMode"`
				LocationFormat                  string `json:"LocationFormat"`
				EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
				AdapterType                     string `json:"AdapterType"`
				PhysicalDrives                  []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat        string `json:"LocationFormat"`
					CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
					BlockSizeBytes        int    `json:"BlockSizeBytes"`
					CapacityMiB           int    `json:"CapacityMiB"`
					FirmwareVersion       struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceType         string `json:"InterfaceType"`
					LegacyBootPriority    string `json:"LegacyBootPriority"`
					SerialNumber          string `json:"SerialNumber"`
					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					Model                 string `json:"Model"`
					MediaType             string `json:"MediaType"`
					RotationalSpeedRpm    int    `json:"RotationalSpeedRpm"`

					DiskDriveUse       string `json:"DiskDriveUse"`
					EncryptedDrive     bool   `json:"EncryptedDrive"`
					InterfaceSpeedMbps int    `json:"InterfaceSpeedMbps"`
					Location           string `json:"Location"`
				} `json:"PhysicalDrives"`
				LogicalDrives []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LogicalDriveEncryption bool `json:"LogicalDriveEncryption"`
					DataDrives             []struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						LocationFormat        string `json:"LocationFormat"`
						CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
						BlockSizeBytes        int    `json:"BlockSizeBytes"`
						CapacityMiB           int    `json:"CapacityMiB"`
						FirmwareVersion       struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"FirmwareVersion"`
						InterfaceType          string   `json:"InterfaceType"`
						LegacyBootPriority     string   `json:"LegacyBootPriority"`
						SerialNumber           string   `json:"SerialNumber"`
						DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
						Model                  string   `json:"Model"`
						MediaType              string   `json:"MediaType"`
						RotationalSpeedRpm     int      `json:"RotationalSpeedRpm"`
						DiskDriveUse           string   `json:"DiskDriveUse"`
						EncryptedDrive         bool     `json:"EncryptedDrive"`
						InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
						Location               string   `json:"Location"`
					} `json:"DataDrives"`
					StripeSizeBytes        int    `json:"StripeSizeBytes"`
					Raid                   string `json:"Raid"`
					CapacityMiB            int    `json:"CapacityMiB"`
					LogicalDriveType       string `json:"LogicalDriveType"`
					LogicalDriveNumber     int    `json:"LogicalDriveNumber"`
					AccelerationMethod     string `json:"AccelerationMethod"`
					InterfaceType          string `json:"InterfaceType"`
					LegacyBootPriority     string `json:"LegacyBootPriority"`
					LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
					MediaType              string `json:"MediaType"`
					VolumeUniqueIdentifier string `json:"VolumeUniqueIdentifier"`
					LogicalDriveName       string `json:"LogicalDriveName"`
				} `json:"LogicalDrives"`

				InternalPortCount       int    `json:"InternalPortCount"`
				CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
				EncryptionFwLocked      bool   `json:"EncryptionFwLocked"`
				Name                    string `json:"Name"`
				BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
				DriveWriteCache         string `json:"DriveWriteCache"`
				Status                  struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				EncryptionSelfTestPassed bool `json:"EncryptionSelfTestPassed"`
				ExternalPortCount        int  `json:"ExternalPortCount"`
				ControllerBoard          struct {
					Status struct {
						Health string `json:"Health"`
					} `json:"Status"`
				} `json:"ControllerBoard"`
				EncryptionEnabled bool `json:"EncryptionEnabled"`
				EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
				EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
				SerialNumber                       string `json:"SerialNumber"`
				CacheModuleStatus                  struct {
					Health string `json:"Health"`
				} `json:"CacheModuleStatus"`
				StorageEnclosures []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat string `json:"LocationFormat"`
					DriveBayCount  int    `json:"DriveBayCount"`
					ID             string `json:"Id"`
					FirmwareVersion struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					Location string `json:"Location"`
				} `json:"StorageEnclosures"`
				CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
				Model                   string `json:"Model"`
				Location                string `json:"Location"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"LocalStorage"`

		MemoryList struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
				BoardCPUNumber            int `json:"BoardCpuNumber"`
				BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
				BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
				BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"MemoryList"`
		LocalStorageV2 struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []any     `json:"data"`
			Etag            string    `json:"etag"`
			Count           int       `json:"count"`
			Name            string    `json:"name"`
		} `json:"LocalStorageV2"`
		AdvancedMemoryProtection struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				AmpModeActive    string   `json:"AmpModeActive"`
				OdataType        string   `json:"@odata.type"`
				AmpModeSupported []string `json:"AmpModeSupported"`
				AmpModeStatus    string   `json:"AmpModeStatus"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"AdvancedMemoryProtection"`
		SubResources3 struct {
		Devices struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health any `json:"Health"`
					State  any `json:"State"`
				} `json:"Status,omitempty"`
				PartNumber string `json:"PartNumber"`
				OdataType  string `json:"@odata.type"`
				OdataEtag  string `json:"@odata.etag"`

				ProductVersion  string `json:"ProductVersion"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				Name              string `json:"Name"`
				DeviceType        string `json:"DeviceType"`
				SerialNumber      string `json:"SerialNumber"`
				ProductPartNumber string `json:"ProductPartNumber"`
				Manufacturer      string `json:"Manufacturer"`
				ID                string `json:"Id"`
				Location          string `json:"Location"`
				Status0           struct {
					State string `json:"State"`
				} `json:"Status,omitempty"`
				MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
				Status1              struct {
					State string `json:"State"`
				} `json:"Status,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Devices"`
		LocalStorage struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
				CurrentOperatingMode            string `json:"CurrentOperatingMode"`
				LocationFormat                  string `json:"LocationFormat"`
				EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
				AdapterType                     string `json:"AdapterType"`
				PhysicalDrives                  []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat         string   `json:"LocationFormat"`
					CapacityLogicalBlocks  int      `json:"CapacityLogicalBlocks"`
					BlockSizeBytes         int      `json:"BlockSizeBytes"`
					CapacityMiB            int      `json:"CapacityMiB"`
					FirmwareVersion        struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceType          string   `json:"InterfaceType"`
					LegacyBootPriority     string   `json:"LegacyBootPriority"`
					SerialNumber           string   `json:"SerialNumber"`
					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					Model                  string   `json:"Model"`
					MediaType              string   `json:"MediaType"`
					RotationalSpeedRpm     int      `json:"RotationalSpeedRpm"`
					DiskDriveUse           string   `json:"DiskDriveUse"`
					EncryptedDrive         bool     `json:"EncryptedDrive"`
					InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
					Location               string   `json:"Location"`
				} `json:"PhysicalDrives"`
				LogicalDrives []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LogicalDriveEncryption bool `json:"LogicalDriveEncryption"`
					DataDrives             []struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						LocationFormat        string `json:"LocationFormat"`
						CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
						BlockSizeBytes        int    `json:"BlockSizeBytes"`
						CapacityMiB           int    `json:"CapacityMiB"`
						FirmwareVersion       struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"FirmwareVersion"`
						InterfaceType          string   `json:"InterfaceType"`
						LegacyBootPriority     string   `json:"LegacyBootPriority"`
						SerialNumber           string   `json:"SerialNumber"`
						DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
						Model                  string   `json:"Model"`
						MediaType              string   `json:"MediaType"`
						RotationalSpeedRpm     int      `json:"RotationalSpeedRpm"`
						DiskDriveUse           string   `json:"DiskDriveUse"`
						EncryptedDrive         bool     `json:"EncryptedDrive"`

						InterfaceSpeedMbps int    `json:"InterfaceSpeedMbps"`
						Location           string `json:"Location"`
					} `json:"DataDrives"`
					StripeSizeBytes           int      `json:"StripeSizeBytes"`
					Raid                      string   `json:"Raid"`
					CapacityMiB               int      `json:"CapacityMiB"`
					LogicalDriveType          string   `json:"LogicalDriveType"`
					LogicalDriveNumber        int      `json:"LogicalDriveNumber"`
					AccelerationMethod        string   `json:"AccelerationMethod"`
					InterfaceType             string   `json:"InterfaceType"`
					LegacyBootPriority        string   `json:"LegacyBootPriority"`
					LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
					MediaType                 string   `json:"MediaType"`
					VolumeUniqueIdentifier    string   `json:"VolumeUniqueIdentifier"`
					LogicalDriveName          string   `json:"LogicalDriveName"`
				} `json:"LogicalDrives"`
				InternalPortCount       int    `json:"InternalPortCount"`
				CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
				EncryptionFwLocked      bool   `json:"EncryptionFwLocked"`
				Name                    string `json:"Name"`
				BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
				DriveWriteCache         string `json:"DriveWriteCache"`
				Status                  struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				EncryptionSelfTestPassed bool `json:"EncryptionSelfTestPassed"`
				ExternalPortCount        int  `json:"ExternalPortCount"`
				ControllerBoard          struct {
					Status struct {
						Health string `json:"Health"`
					} `json:"Status"`
				} `json:"ControllerBoard"`
				EncryptionEnabled bool `json:"EncryptionEnabled"`
				EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
				EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
				SerialNumber                       string `json:"SerialNumber"`
				CacheModuleStatus                  struct {
					Health string `json:"Health"`
				} `json:"CacheModuleStatus"`
				StorageEnclosures []struct {
					Status struct {
						Health string `json:"Health"`

						State string `json:"State"`
					} `json:"Status"`
					LocationFormat  string `json:"LocationFormat"`
					DriveBayCount   int    `json:"DriveBayCount"`
					ID              string `json:"Id"`
					FirmwareVersion struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					Location string `json:"Location"`
				} `json:"StorageEnclosures"`
				CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
				Model                   string `json:"Model"`
				Location                string `json:"Location"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"LocalStorage"`
		Memory struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				PartNumber                  string `json:"PartNumber"`
				PersistentRegionSizeLimitMiB int   `json:"PersistentRegionSizeLimitMiB"`
				OdataType                   string `json:"@odata.type"`
				LogicalSizeMiB              int    `json:"LogicalSizeMiB"`
				RankCount                   any    `json:"RankCount"`
				SecurityCapabilities        struct {
				} `json:"SecurityCapabilities"`
				BusWidthBits          int      `json:"BusWidthBits"`
				CacheSizeMiB          int      `json:"CacheSizeMiB"`
				Name                  string   `json:"Name"`
				NonVolatileSizeMiB    int      `json:"NonVolatileSizeMiB"`
				OperatingMemoryModes  []string `json:"OperatingMemoryModes"`
				MemoryLocation        struct {
					Channel          int `json:"Channel"`
					Slot             int `json:"Slot"`
					MemoryController int `json:"MemoryController"`
					Socket           int `json:"Socket"`
				} `json:"MemoryLocation"`
				DeviceLocator string `json:"DeviceLocator"`
				MemoryType    string `json:"MemoryType"`

				VendorID          string   `json:"VendorID"`
				OdataEtag         string   `json:"@odata.etag"`
				DataWidthBits     int      `json:"DataWidthBits"`
				CapacityMiB       int      `json:"CapacityMiB"`
				VolatileSizeMiB   int      `json:"VolatileSizeMiB"`
				ErrorCorrection   string   `json:"ErrorCorrection"`
				SerialNumber      string   `json:"SerialNumber"`
				MemoryMedia       []string `json:"MemoryMedia"`
				Oem               struct {
					Hpe struct {
						DIMMStatus                string `json:"DIMMStatus"`
						OdataType                 string `json:"@odata.type"`
						DIMMManufacturingDate     string `json:"DIMMManufacturingDate"`
						MinimumVoltageVoltsX10    int    `json:"MinimumVoltageVoltsX10"`
					} `json:"Hpe"`
				} `json:"Oem"`
				VolatileRegionSizeLimitMiB int    `json:"VolatileRegionSizeLimitMiB"`
				ID                         string `json:"Id"`
				OperatingSpeedMhz          int    `json:"OperatingSpeedMhz,omitempty"`
				BaseModuleType             string `json:"BaseModuleType,omitempty"`
				Manufacturer               string `json:"Manufacturer,omitempty"`
				MemoryDeviceType           string `json:"MemoryDeviceType,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Memory"`
		MemoryList struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
				BoardCPUNumber            int `json:"BoardCpuNumber"`
				BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
				BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
				BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"MemoryList"`
		LocalStorageV2 struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					HealthRollup string `json:"HealthRollup"`

					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				Controllers []struct {
					SpeedGbps int `json:"SpeedGbps"`
					Status    struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					PCIeInterface struct {
						PCIeType    string `json:"PCIeType"`
						MaxPCIeType string `json:"MaxPCIeType"`
						MaxLanes    int    `json:"MaxLanes"`
						LanesInUse  int    `json:"LanesInUse"`
					} `json:"PCIeInterface"`
					PartNumber      string `json:"PartNumber"`
					OdataType       string `json:"@odata.type"`
					ControllerRates struct {
						ConsistencyCheckRatePercent int `json:"ConsistencyCheckRatePercent"`
						RebuildRatePercent          int `json:"RebuildRatePercent"`
						TransformationRatePercent   int `json:"TransformationRatePercent"`
					} `json:"ControllerRates"`
					Ports []struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						OdataType            string `json:"@odata.type"`
						LinkNetworkTechnology any    `json:"LinkNetworkTechnology"`
						OdataEtag            string `json:"@odata.etag"`
						FunctionMinBandwidth any    `json:"FunctionMinBandwidth"`
						MaxSpeedGbps         int    `json:"MaxSpeedGbps"`
						PortID               string `json:"PortId"`
						Name                 string `json:"Name"`
						PortProtocol         string `json:"PortProtocol"`
						OdataID              string `json:"@odata.id"`
						LinkConfiguration    []any  `json:"LinkConfiguration"`
						CurrentSpeedGbps     int    `json:"CurrentSpeedGbps"`
						Ethernet             any    `json:"Ethernet"`
						ID                   string `json:"Id"`
						FunctionMaxBandwidth any    `json:"FunctionMaxBandwidth"`
						Width                int    `json:"Width"`
						Location             struct {
							PartLocation struct {
								ServiceLabel       string `json:"ServiceLabel"`
								LocationOrdinalValue int  `json:"LocationOrdinalValue"`
								LocationType       string `json:"LocationType"`
							} `json:"PartLocation"`
						} `json:"Location"`

						PortType string `json:"PortType"`
					} `json:"Ports"`
					CacheSummary struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
						PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
					} `json:"CacheSummary"`
					SupportedRAIDTypes         []string `json:"SupportedRAIDTypes"`
					FirmwareVersion            string   `json:"FirmwareVersion"`
					MemberID                   any      `json:"MemberId"`
					OdataID                    string   `json:"@odata.id"`
					SupportedControllerProtocols []string `json:"SupportedControllerProtocols"`
					SerialNumber               string   `json:"SerialNumber"`
					Manufacturer               string   `json:"Manufacturer"`
					Model                      string   `json:"Model"`
					SupportedDeviceProtocols   []string `json:"SupportedDeviceProtocols"`
					Identifiers                []struct {
						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
					Sku      string `json:"SKU"`
					Location struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"Location"`
				} `json:"Controllers"`
				OdataID string `json:"@odata.id"`
				Volumes []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					ReadCachePolicy  string `json:"ReadCachePolicy"`
					OdataType        string `json:"@odata.type"`
					Encrypted        bool   `json:"Encrypted"`
					EncryptionTypes  []any  `json:"EncryptionTypes"`
					CapacityBytes    int64  `json:"CapacityBytes"`
					OdataEtag        string `json:"@odata.etag"`
					RAIDType         string `json:"RAIDType"`
					BlockSizeBytes   int    `json:"BlockSizeBytes"`
					Name             string `json:"Name"`
					OptimumIOSizeBytes int  `json:"OptimumIOSizeBytes"`
					LogicalUnitNumber int   `json:"LogicalUnitNumber"`

					OdataID        string `json:"@odata.id"`
					VolumeType     any    `json:"VolumeType"`
					DisplayName    string `json:"DisplayName"`
					Links          struct {
						DedicatedSpareDrives []any `json:"DedicatedSpareDrives"`
						Drives               []struct {
							OdataID string `json:"@odata.id"`
						} `json:"Drives"`
					} `json:"Links"`
					ID             string `json:"Id"`
					StripSizeBytes int    `json:"StripSizeBytes"`
					Identifiers    []struct {
						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
					Operations  []any  `json:"Operations"`
					VolumeUsage string `json:"VolumeUsage"`
				} `json:"Volumes"`
				StorageControllers []struct {
					SpeedGbps int `json:"SpeedGbps"`
					Status    struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					PCIeInterface struct {
						PCIeType    string `json:"PCIeType"`
						MaxPCIeType string `json:"MaxPCIeType"`
						MaxLanes    int    `json:"MaxLanes"`
						LanesInUse  int    `json:"LanesInUse"`
					} `json:"PCIeInterface"`
					PartNumber      string `json:"PartNumber"`
					OdataType       string `json:"@odata.type"`
					ControllerRates struct {
						ConsistencyCheckRatePercent int `json:"ConsistencyCheckRatePercent"`
						RebuildRatePercent          int `json:"RebuildRatePercent"`
						TransformationRatePercent   int `json:"TransformationRatePercent"`
					} `json:"ControllerRates"`
					Ports []struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						OdataType             string `json:"@odata.type"`
						LinkNetworkTechnology any    `json:"LinkNetworkTechnology"`
						OdataEtag             string `json:"@odata.etag"`
						FunctionMinBandwidth  any    `json:"FunctionMinBandwidth"`
						MaxSpeedGbps          int    `json:"MaxSpeedGbps"`
						PortID                string `json:"PortId"`
						Name                  string `json:"Name"`

						PortProtocol      string `json:"PortProtocol"`
						OdataID           string `json:"@odata.id"`
						LinkConfiguration []any  `json:"LinkConfiguration"`
						CurrentSpeedGbps  int    `json:"CurrentSpeedGbps"`
						Ethernet          any    `json:"Ethernet"`
						ID                string `json:"Id"`
						FunctionMaxBandwidth any `json:"FunctionMaxBandwidth"`
						Width             int    `json:"Width"`
						Location          struct {
							PartLocation struct {
								ServiceLabel         string `json:"ServiceLabel"`
								LocationOrdinalValue int    `json:"LocationOrdinalValue"`
								LocationType         string `json:"LocationType"`
							} `json:"PartLocation"`
						} `json:"Location"`
						PortType string `json:"PortType"`
					} `json:"Ports"`
					CacheSummary struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
						PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
					} `json:"CacheSummary"`
					SupportedRAIDTypes           []string `json:"SupportedRAIDTypes"`
					FirmwareVersion               string   `json:"FirmwareVersion"`
					MemberID                      any      `json:"MemberId"`
					OdataID                       string   `json:"@odata.id"`
					SupportedControllerProtocols  []string `json:"SupportedControllerProtocols"`
					SerialNumber                  string   `json:"SerialNumber"`
					Manufacturer                  string   `json:"Manufacturer"`
					Model                         string   `json:"Model"`
					SupportedDeviceProtocols      []string `json:"SupportedDeviceProtocols"`
					Identifiers                   []struct {
						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
					Sku      string `json:"SKU"`
					Location struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"Location"`
				} `json:"StorageControllers"`
				OdataType string `json:"@odata.type"`
				OdataEtag string `json:"@odata.etag"`

				Drives []struct {
					HotspareType             string `json:"HotspareType"`
					OdataType                string `json:"@odata.type"`
					CapacityBytes            int64  `json:"CapacityBytes"`
					Name                     string `json:"Name"`
					StatusIndicator          string `json:"StatusIndicator"`
					OdataID                  string `json:"@odata.id"`
					NegotiatedSpeedGbs       int    `json:"NegotiatedSpeedGbs"`
					Manufacturer             string `json:"Manufacturer"`
					MediaType                string `json:"MediaType"`
					Protocol                 string `json:"Protocol"`
					Operations               []any  `json:"Operations"`
					IndicatorLED             any    `json:"IndicatorLED"`
					PhysicalLocation         struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"PhysicalLocation"`
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					OdataEtag                    string `json:"@odata.etag"`
					WriteCacheEnabled             bool   `json:"WriteCacheEnabled"`
					BlockSizeBytes                int    `json:"BlockSizeBytes"`
					PredictedMediaLifeLeftPercent any    `json:"PredictedMediaLifeLeftPercent"`
					CapableSpeedGbs               int    `json:"CapableSpeedGbs"`
					FailurePredicted              bool   `json:"FailurePredicted"`
					Revision                      string `json:"Revision"`
					SerialNumber                  string `json:"SerialNumber"`
					Model                         string `json:"Model"`
					Links                         struct {
						Chassis any    `json:"Chassis"`
						Volumes []struct {
							OdataID string `json:"@odata.id"`
						} `json:"Volumes"`
					} `json:"Links"`
					ID          string `json:"Id"`
					Identifiers []struct {
						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
					Location any `json:"Location"`
				} `json:"Drives"`
				ID   string `json:"Id"`
				Name string `json:"Name"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
			} `json:"LocalStorageV2"`
			AdvancedMemoryProtection struct {
				URI             string    `json:"uri"`
				CollectionState string    `json:"collectionState"`
				Modified        time.Time `json:"modified"`
				Data            []struct {
					AmpModeActive    string   `json:"AmpModeActive"`
					OdataType        string   `json:"@odata.type"`
					AmpModeSupported []string `json:"AmpModeSupported"`
					AmpModeStatus    string   `json:"AmpModeStatus"`
				} `json:"data"`
				Etag  string `json:"etag"`
				Count int    `json:"count"`
				Name  string `json:"name"`
			} `json:"AdvancedMemoryProtection"`
		} `json:"subResources,omitempty"`
		} `json:"SubResources2"`
	} `json:"members"`
	Start        int    `json:"start"`
	Total        int    `json:"total"`
	PrevPageURI  any    `json:"prevPageUri"`
	NextPageURI  string `json:"nextPageUri"`
	URI0         string `json:"uri"`
}

// ServerHardwareExt - расширенное описание сервера OneView (ответ /rest/server-hardware),
// включая вложенные под-ресурсы (память, память AMP и т.п.)
type ServerHardwareExt struct {
	ApplianceLocation string          `json:"applianceLocation"`
	ApplianceName     string          `json:"applianceName"`
	Appluri           string          `json:"appluri"`
	AssetTag          string          `json:"assetTag"`
	Category          string          `json:"category"`
	Created           time.Time       `json:"created"`
	Description       interface{}     `json:"description"`
	ETag              string          `json:"eTag"`
	FormFactor        string          `json:"formFactor"`
	Generation        string          `json:"generation"`
	Groups            []struct {
	} `json:"groups"`
	HostOsType        interface{} `json:"hostOsType"`
	ID                string      `json:"id"`
	LicensingIntent   string      `json:"licensingIntent"`
	LocationURI       string      `json:"locationUri"`
	MemoryMb          int         `json:"memoryMb"`
	Model             string      `json:"model"`
	Modified          time.Time   `json:"modified"`
	MpDNSName         string      `json:"mpDnsName"`
	MpFirmwareVersion string      `json:"mpFirmwareVersion"`
	MpHostInfo        struct {

		MpHostName    string        `json:"mpHostName"`
		MpIPAddresses []interface{} `json:"mpIpAddresses"`
	} `json:"mpHostInfo"`
	MpIPAddress               string      `json:"mpIpAddress"`
	MpModel                   string      `json:"mpModel"`
	MpState                   string      `json:"mpState"`
	Name                      string      `json:"name"`
	OriginalURI               string      `json:"originalUri"`
	PartNumber                string      `json:"partNumber"`
	PhysicalServerHardwareURI interface{} `json:"physicalServerHardwareUri"`
	Position                  int         `json:"position"`
	PowerLock                 bool        `json:"powerLock"`
	PowerState                string      `json:"powerState"`
	ProcessorCoreCount        int         `json:"processorCoreCount"`
	ProcessorCount            int         `json:"processorCount"`
	ProcessorSpeedMhz         int         `json:"processorSpeedMhz"`
	ProcessorType             string      `json:"processorType"`
	RefreshState              string      `json:"refreshState"`
	RemoteSupportSettings     struct {
		RemoteSupportCurrentState string `json:"remoteSupportCurrentState"`
	} `json:"remoteSupportSettings"`
	RomVersion              string `json:"romVersion"`
	SerialNumber            string `json:"serialNumber"`
	ServerGroupURI          string `json:"serverGroupUri"`
	ServerHardwareTypeURI   string `json:"serverHardwareTypeUri"`
	ServerName              string `json:"serverName"`
	ServerProfileURI        string `json:"serverProfileUri"`
	ShortModel              string `json:"shortModel"`
	State                   string `json:"state"`
	StateReason             string `json:"stateReason"`
	Status                  string `json:"status"`
	SupportState            string `json:"supportState"`
	SupportDataCollectionState string `json:"supportDataCollectionState"`
	SupportDataCollectionType  string `json:"supportDataCollectionType"`
	SubResources            struct {
		AdvancedMemoryProtection struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				OdataType        string        `json:"@odata.type"`
				AmpModeActive    string        `json:"AmpModeActive"`
				AmpModeStatus    string        `json:"AmpModeStatus"`
				AmpModeSupported []interface{} `json:"AmpModeSupported"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"AdvancedMemoryProtection"`

		Devices struct {
		} `json:"Devices"`
		LocalStorage struct {
		} `json:"LocalStorage"`
		Memory struct {
		} `json:"Memory"`
		MemoryList struct {
		} `json:"MemoryList"`
	} `json:"subResources"`
	Type                string `json:"type"`
	UIDState            string `json:"uidState"`
	URI                 string `json:"uri"`
	UUID                string `json:"uuid"`
	VirtualSerialNumber string `json:"virtualSerialNumber"`
	VirtualUUID         string `json:"virtualUuid"`
}

// EnvironmentalConfiguration - конфигурация питания/охлаждения и позиция сервера в стойке (энвайронментал-параметры OneView)
type EnvironmentalConfiguration struct {
	PowerHistorySupported       bool   `json:"powerHistorySupported"`
	ThermalHistorySupported     bool   `json:"thermalHistorySupported"`
	UtilizationHistorySupported bool   `json:"utilizationHistorySupported"`
	CapHistorySupported         bool   `json:"capHistorySupported"`
	HistorySampleIntervalSeconds int   `json:"historySampleIntervalSeconds"`
	HistoryBufferSize           int    `json:"historyBufferSize"`
	PowerCapType                string `json:"powerCapType"`
	IdleMaxPower                int    `json:"idleMaxPower"`
	CalibratedMaxPower          int    `json:"calibratedMaxPower"`
	PsuList                     []struct {
		Capacity     int    `json:"capacity"`
		PsuID        int    `json:"psuId"`
		InputVoltage int    `json:"inputVoltage"`
		Side         string `json:"side"`
		PowerType    string `json:"powerType"`
	} `json:"psuList"`
	Height             int         `json:"height"`
	RackID             interface{} `json:"rackId"`
	USlot              int         `json:"uSlot"`
	RackName           interface{} `json:"rackName"`
	RelativeOrder      int         `json:"relativeOrder"`
	RackModel          string      `json:"rackModel"`
	LicenseRequirement string      `json:"licenseRequirement"`
	RackUHeight        int         `json:"rackUHeight"`
}

// CacheModuleStatus  - Health состояние модуля кэша "OK" при наличие и нормально состоянии, если отсутствует то ""
type CacheModuleStatus struct {
	Health string `json:"Health"`
}

// ControllerSatus состояние контроллера Health "OK", State "Enabled"
type ControllerSatus struct {
	Health string `json:"Health"`
	State  string `json:"State"`
}

// ControllerBoard  -
type ControllerBoard struct {
	Status ControllerSatus `json:"Status"`
}

// Version  - версия в виде строки VersionString
type Version struct {
	VersionString string `json:"VersionString"`
}

// FirmwareVersion  - текущая версия Firmware
type FirmwareVersion struct {
	Current Version `json:"Current"`
}

// Status  - состояние Health "OK", State "Enabled"
type Status struct {
	Health string `json:"Health"`
	State  string `json:"State"`
}

// LogicalDataDrive  - описание диска в логическом томе
type LogicalDataDrive struct {
	CapacityMiB            float64         `json:"CapacityMiB"`            //емкость диска в мегабайтах
	DiskDriveStatusReasons []string        `json:"DiskDriveStatusReasons"` //статусы "None"
	DiskDriveUse           string          `json:"DiskDriveUse"`           //использование диска в томе "Data"
	EncryptedDrive         bool            `json:"EncryptedDrive"`         //включено ли шифрование true, false
	FirmwareVersion        FirmwareVersion `json:"FirmwareVersion"`        //текущая версия прошивки
	Location               string          `json:"Location"`               //расположение "1I:1:7"
	LocationFormat         string          `json:"LocationFormat"`         //формат строки расположения "ControllerPort:Box:Bay"
	MediaType              string          `json:"MediaType"`              //тип диска "HDD", ...
	Model                  string          `json:"Model"`                  //модель "MB4000JVYZQ"
	SerialNumber           string          `json:"SerialNumber"`           //серийный номер "ZC17LE2X"
	Status                 Status          `json:"Status"`                 //состояние
}

// LocalLogicalDrive  - описание тома
type LocalLogicalDrive struct {
	CapacityMiB            float64            `json:"CapacityMiB"`            //емкость в мегабайтах
	DataDrives             []LogicalDataDrive `json:"DataDrives"`             //диски
	LogicalDriveEncryption bool               `json:"LogicalDriveEncryption"` //шифрование
	LogicalDriveNumber     int                `json:"LogicalDriveNumber"`     //номер логического диска
	LogicalDriveType       string             `json:"LogicalDriveType"`       //тип логического диска

	Raid   string `json:"Raid"`   //уровень Raid "5", ...
	Status Status `json:"Status"` //состояние
}

// LocalPhysicalDrive  - описание диска
type LocalPhysicalDrive struct {
	CapacityMiB            float64         `json:"CapacityMiB"`            //емкость
	DiskDriveStatusReasons []string        `json:"DiskDriveStatusReasons"` //статусы "None"
	DiskDriveUse           string          `json:"DiskDriveUse"`           //использование диска в томе "Data"
	EncryptedDrive         bool            `json:"EncryptedDrive"`         //включено ли шифрование true, false
	FirmwareVersion        FirmwareVersion `json:"FirmwareVersion"`        //текущая версия прошивки
	Location               string          `json:"Location"`               //расположение "1I:1:7"
	LocationFormat         string          `json:"LocationFormat"`         //формат строки расположения "ControllerPort:Box:Bay"
	MediaType              string          `json:"MediaType"`              //тип диска "HDD", ...
	Model                  string          `json:"Model"`                  //модель "MB4000JVYZQ"
	SerialNumber           string          `json:"SerialNumber"`           //серийный номер "ZC17LE2X"
	Status                 Status          `json:"Status"`                 //состояние
}

// LocalStorageEnclosure  - описание корзины для дисков
type LocalStorageEnclosure struct {
	DriveBayCount   int             `json:"DriveBayCount"`   //количество мест
	FirmwareVersion FirmwareVersion `json:"FirmwareVersion"` //версия прошивки
	Location        string          `json:"Location"`        //расположение "1I:1"
	LocationFormat  string          `json:"LocationFormat"`  //формат расположения "ControllerPort:Box"
	Status          Status          `json:"Status"`          //состояние
}

// LocalStorage  - описание локального хранилища
type LocalStorage struct {
	AdapterType               string                   `json:"AdapterType"`               //тип адаптера "SmartArray"
	CacheMemorySizeMiB        int                      `json:"CacheMemorySizeMiB"`        //размер кэша контроллера в мегабайтах
	CacheModuleSerialNumber   string                   `json:"CacheModuleSerialNumber"`   //серийный номер контроллера
	CacheModuleStatus         CacheModuleStatus        `json:"CacheModuleStatus"`         //состояние кэша
	ControllerBoard           ControllerBoard          `json:"ControllerBoard"`
	EncryptionCspTestPassed   bool                     `json:"EncryptionCspTestPassed"`   //результат тестового прохода
	EncryptionEnabled         bool                     `json:"EncryptionEnabled"`         //шифрование включено?
	EncryptionSelfTestPassed  bool                     `json:"EncryptionSelfTestPassed"`  //результата теста шифрования
	FirmwareVersion           FirmwareVersion          `json:"FirmwareVersion"`           //прошивка
	Location                  string                   `json:"Location"`                  //расположение "Slot 3"
	LogicalDrives             []LocalLogicalDrive      `json:"LogicalDrives"`             //тома (логические диски)
	Model                     string                   `json:"Model"`                      //модель "Smart array p840 Controller"
	Name                      string                   `json:"Name"`                       //наименование "Controller in Slot 3"
	PhysicalDrives            []LocalPhysicalDrive     `json:"PhysicalDrives"`             //диски подключенные к хранилищу
	SerialNumber              string                   `json:"SerialNumber"`               //серийный номер
	Status                    Status                   `json:"Status"`                     //состояние
	StorageEnclosures         []LocalStorageEnclosure  `json:"StorageEnclosures"`          //корзины для дисков
}

// ServerHardwareLocalStorage  - структура возвращаемая по запросу о хранилищ
type ServerHardwareLocalStorage struct {
	URI             string         `json:"uri"`            //ссылка на запрос
	CollectionState string         `json:"collectionState"` //состояние "Collected"
	Modified        time.Time      `json:"modified"`        //время модификации
	Data            []LocalStorage `json:"data"`             //срез хранилищ
	Etag            string         `json:"etag"`             //тэг
	Count           int            `json:"count"`            //количество
	Name            string         `json:"name"`             ///"LocalStorage"
}

// DeviceBayInfo - описание отсека (bay) для сервера в корзине (enclosure) OneView
type DeviceBayInfo struct {
	Type                                string      `json:"type"`
	BayNumber                           int         `json:"bayNumber"`
	Model                               interface{} `json:"model"`
	DevicePresence                      string      `json:"devicePresence"`
	ProfileURI                          interface{} `json:"profileUri"`
	DeviceURI                           string      `json:"deviceUri"`
	CoveredByProfile                    interface{} `json:"coveredByProfile"`
	CoveredByDevice                     string      `json:"coveredByDevice"`
	Ipv4Setting                         interface{} `json:"ipv4Setting"`
	Ipv6Setting                         interface{} `json:"ipv6Setting"`
	URI                                 string      `json:"uri"`
	Category                            string      `json:"category"`
	ETag                                interface{} `json:"eTag"`
	Created                             interface{} `json:"created"`
	Modified                            interface{} `json:"modified"`
	AvailableForHalfHeightProfile       bool        `json:"availableForHalfHeightProfile"`
	AvailableForFullHeightProfile       bool        `json:"availableForFullHeightProfile"`
	DeviceBayType                       string      `json:"deviceBayType"`
	DeviceFormFactor                    string      `json:"deviceFormFactor"`
	BayPowerState                       string      `json:"bayPowerState"`
	ChangeState                         string      `json:"changeState"`
	AvailableForHalfHeightDoubleWideProfile bool    `json:"availableForHalfHeightDoubleWideProfile"`
	AvailableForFullHeightDoubleWideProfile bool    `json:"availableForFullHeightDoubleWideProfile"`
	UUID                                interface{} `json:"uuid"`
}

// InterconnectBayInfo - описание отсека коммутатора-интерконнекта в корзине (enclosure) OneView
type InterconnectBayInfo struct {
	BayNumber           int         `json:"bayNumber"`
	InterconnectURI     string      `json:"interconnectUri"`
	LogicalInterconnectURI interface{} `json:"logicalInterconnectUri"`
	InterconnectModel   string      `json:"interconnectModel"`
	Ipv4Setting         interface{} `json:"ipv4Setting"`
	Ipv6Setting         interface{} `json:"ipv6Setting"`
	SerialNumber        string      `json:"serialNumber"`
	InterconnectBayType string      `json:"interconnectBayType"`
	ChangeState         string      `json:"changeState"`
	BayPowerState       string      `json:"bayPowerState"`
}

// FanBayInfo - описание отсека вентилятора в корзине (enclosure) OneView
type FanBayInfo struct {
	BayNumber      int    `json:"bayNumber"`
	DevicePresence string `json:"devicePresence"`
	DeviceRequired bool   `json:"deviceRequired"`
	Status         string `json:"status"`
	Model          string `json:"model"`
	PartNumber     string `json:"partNumber"`
	SparePartNumber string `json:"sparePartNumber"`
	FanBayType     string `json:"fanBayType"`
	ChangeState    string `json:"changeState"`
	State          string `json:"state"`
}

// PowerSupplyBayInfo - описание отсека блока питания в корзине (enclosure) OneView
type PowerSupplyBayInfo struct {
	BayNumber           int    `json:"bayNumber"`
	DevicePresence      string `json:"devicePresence"`
	Status              string `json:"status"`
	Model               string `json:"model"`
	SerialNumber        string `json:"serialNumber"`
	PartNumber          string `json:"partNumber"`
	SparePartNumber     string `json:"sparePartNumber"`
	PowerSupplyBayType  string `json:"powerSupplyBayType"`
	ChangeState         string `json:"changeState"`
}

// ManagerBayInfo - описание отсека модуля управления (Onboard Administrator) в корзине (enclosure) OneView
type ManagerBayInfo struct {
	BayNumber      int         `json:"bayNumber"`
	ManagerType    string      `json:"managerType"`
	UIDState       interface{} `json:"uidState"`
	BayPowerState  string      `json:"bayPowerState"`
	FwVersion      string      `json:"fwVersion"`
	DevicePresence string      `json:"devicePresence"`
	Role           string      `json:"role"`
	IPAddress      string      `json:"ipAddress"`
	ChangeState    string      `json:"changeState"`
	FwBuildDate    string      `json:"fwBuildDate"`
	DhcpIpv6Enable bool        `json:"dhcpIpv6Enable"`
	Ipv6Addresses  []interface{} `json:"ipv6Addresses"`
	FqdnHostName   string      `json:"fqdnHostName"`
	DhcpEnable     bool        `json:"dhcpEnable"`
}

// Enclosure - описание корзины (enclosure) OneView со всеми её отсеками (серверы, интерконнекты, вентиляторы, БП, менеджеры)
type Enclosure struct {
	Type     string    `json:"type"`
	URI      string    `json:"uri"`
	Category string    `json:"category"`
	ETag     string    `json:"eTag"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`

	RefreshState            string                 `json:"refreshState"`
	StateReason             string                 `json:"stateReason"`
	EnclosureType           string                 `json:"enclosureType"`
	EnclosureTypeURI        string                 `json:"enclosureTypeUri"`
	EnclosureModel          string                 `json:"enclosureModel"`
	UUID                    string                 `json:"uuid"`
	SerialNumber            string                 `json:"serialNumber"`
	PartNumber              string                 `json:"partNumber"`
	ReconfigurationState    string                 `json:"reconfigurationState"`
	UIDState                interface{}            `json:"uidState"`
	LicensingIntent         string                 `json:"licensingIntent"`
	DeviceBayCount          int                    `json:"deviceBayCount"`
	DeviceBays              []DeviceBayInfo        `json:"deviceBays"`
	InterconnectBayCount    int                    `json:"interconnectBayCount"`
	InterconnectBays        []InterconnectBayInfo  `json:"interconnectBays"`
	FanBayCount             int                    `json:"fanBayCount"`
	FanBays                 []FanBayInfo           `json:"fanBays"`
	PowerSupplyBayCount     int                    `json:"powerSupplyBayCount"`
	PowerSupplyBays         []PowerSupplyBayInfo   `json:"powerSupplyBays"`
	EnclosureGroupURI       interface{}            `json:"enclosureGroupUri"`
	FwBaselineURI           interface{}            `json:"fwBaselineUri"`
	FwBaselineName          interface{}            `json:"fwBaselineName"`
	IsFwManaged             bool                   `json:"isFwManaged"`
	ForceInstallFirmware    bool                   `json:"forceInstallFirmware"`
	LogicalEnclosureURI     interface{}            `json:"logicalEnclosureUri"`
	ManagerBays             []ManagerBayInfo       `json:"managerBays"`
	SupportState            string                 `json:"supportState"`
	SupportDataCollectionState string              `json:"supportDataCollectionState"`
	SupportDataCollectionType  string              `json:"supportDataCollectionType"`
	SupportDataCollectionsURI  string              `json:"supportDataCollectionsUri"`
	RemoteSupportURI        string                 `json:"remoteSupportUri"`
	RemoteSupportSettings   struct {
		RemoteSupportCurrentState string `json:"remoteSupportCurrentState"`
		Destination               string `json:"destination"`
	} `json:"remoteSupportSettings"`
	CrossBars      []interface{} `json:"crossBars"`
	Partitions     []interface{} `json:"partitions"`
	ScopesURI      string        `json:"scopesUri"`
	Status         string        `json:"status"`
	Name           string        `json:"name"`
	State          string        `json:"state"`
	Description    interface{}   `json:"description"`
	StandbyOaPreferredIP string  `json:"standbyOaPreferredIP"`
	ActiveOaPreferredIP  string  `json:"activeOaPreferredIP"`
	AssetTag       string        `json:"assetTag"`
	RackName       string        `json:"rackName"`
	VcmDomainID    string        `json:"vcmDomainId"`
	VcmDomainName  string        `json:"vcmDomainName"`
	VcmMode        bool          `json:"vcmMode"`

	VcmURL         string `json:"vcmUrl"`
	OaBays         int    `json:"oaBays"`
	MigrationState string `json:"migrationState"`
}

// ServerSSOUrl - ссылка для single sign-on на iLO сервера
type ServerSSOUrl struct {
	IloSsoURL string `json:"iloSsoUrl"`
}

// ServerjavaRemoteConsoleUrl - ссылка для запуска Java-консоли удалённого управления сервером
type ServerjavaRemoteConsoleUrl struct {
	JavaRemoteConsoleUrl string `json:"javaRemoteConsoleUrl"`
}

// Content - элемент содержимого дата-центра (позиция объекта на плане: координаты и ссылка на ресурс)
type Content struct {
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Rotation    float64 `json:"rotation"`
	ResourceURI string  `json:"resourceUri"`
}

// DatacenterInfo - описание дата-центра OneView (геометрия, охлаждение, стоимость электроэнергии, содержимое)
type DatacenterInfo struct {
	Width                int         `json:"width"`
	Depth                int         `json:"depth"`
	CoolingCapacity      interface{} `json:"coolingCapacity"`
	CostPerKilowattHour  interface{} `json:"costPerKilowattHour"`
	Currency             string      `json:"currency"`
	DeratingType         string      `json:"deratingType"`
	DeratingPercentage   float64     `json:"deratingPercentage"`
	CoolingMultiplier    float64     `json:"coolingMultiplier"`
	Contents             []Content   `json:"contents"`
	RemoteSupportURI     string      `json:"remoteSupportUri"`
	Category             string      `json:"category"`
	ETag                 string      `json:"eTag"`
	Created              time.Time   `json:"created"`
	Modified             time.Time   `json:"modified"`
	ID                   string      `json:"id"`
	UUID                 string      `json:"uuid"`
	DefaultPowerLineVoltage int      `json:"defaultPowerLineVoltage"`
	Name                 string      `json:"name"`
	URI                  string      `json:"uri"`
	State                string      `json:"state"`
	Status               string      `json:"status"`
}

// DatacentersList - постраничный список дата-центров OneView
type DatacentersList struct {
	Type     string      `json:"type"`
	URI      string      `json:"uri"`
	Category string      `json:"category"`
	ETag     interface{} `json:"eTag"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Start    int       `json:"start"`
	Count    int       `json:"count"`
	Total    int       `json:"total"`
	PrevPageURI interface{} `json:"prevPageUri"`
	NextPageURI interface{} `json:"nextPageUri"`
	Members  []DatacenterInfo `json:"members"`
}

// Racks - описание стойки OneView (габариты, тепловой лимит, серийный номер, состояние)
type Racks struct {
	UHeight      int           `json:"uHeight"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Depth        int           `json:"depth"`
	ThermalLimit interface{}   `json:"thermalLimit"`
	PartNumber   interface{}   `json:"partNumber"`
	SerialNumber string        `json:"serialNumber"`
	RackMounts   []interface{} `json:"rackMounts"`
	Model        interface{}   `json:"model"`
	Category     string        `json:"category"`
	ETag         string        `json:"eTag"`
	Created      time.Time     `json:"created"`
	Modified     time.Time     `json:"modified"`
	ID           string        `json:"id"`
	UUID         string        `json:"uuid"`
	Name         string        `json:"name"`
	URI          string        `json:"uri"`
	State        string        `json:"state"`
	Status       string        `json:"status"`
}

// DeviceInfo - краткое описание устройства, установленного в стойке
type DeviceInfo struct {
	URI           string      `json:"uri"`
	Name          string      `json:"name"`
	Status        string      `json:"status"`
	State         string      `json:"state"`
	Category      string      `json:"category"`
	ID            interface{} `json:"id"`
	UUID          string      `json:"uuid"`
	Created       time.Time   `json:"created"`
	Modified      time.Time   `json:"modified"`
	ETag          string      `json:"eTag"`
	MaxPwrConsumed interface{} `json:"maxPwrConsumed"`
	DeviceType    string      `json:"deviceType"`
	Model         string      `json:"model"`
}

// DeviceRecord - подробная запись об устройстве в стойке (габариты, точки монтажа, состояние)
type DeviceRecord struct {

	UHeight      int         `json:"uHeight"`
	Width        int         `json:"width"`
	Height       int         `json:"height"`
	Depth        int         `json:"depth"`
	ThermalLimit interface{} `json:"thermalLimit"`
	PartNumber   interface{} `json:"partNumber"`
	SerialNumber string      `json:"serialNumber"`
	RackMounts   []struct {
		MountURI      string `json:"mountUri"`
		TopUSlot      int    `json:"topUSlot"`
		UHeight       int    `json:"uHeight"`
		RelativeOrder int    `json:"relativeOrder"`
		Location      string `json:"location"`
	} `json:"rackMounts"`
	Category string      `json:"category"`
	ETag     string      `json:"eTag"`
	Created  time.Time   `json:"created"`
	Modified time.Time   `json:"modified"`
	ID       string      `json:"id"`
	UUID     string      `json:"uuid"`
	Name     string      `json:"name"`
	URI      string      `json:"uri"`
	State    string      `json:"state"`
	Model    interface{} `json:"model"`
	Status   string      `json:"status"`
}

// RacksDeviceTopology - топология размещения устройств в стойке (температура, мощность, вложенная иерархия устройств)
type RacksDeviceTopology struct {
	Device            DeviceRecord    `json:"device"`
	Max               int             `json:"max"`
	PeakTemp          int             `json:"peakTemp"`
	PeakTempTime      int64           `json:"peakTempTime"`
	Capacity          int             `json:"capacity"`
	InferredPosition  bool            `json:"inferredPosition"`
	PowerDistributors []interface{}   `json:"powerDistributors"`
	Height            int             `json:"height"`
	Children          []struct {
		Device            DeviceInfo    `json:"device"`
		Max               int           `json:"max"`
		PeakTemp          int           `json:"peakTemp"`
		PeakTempTime      int64         `json:"peakTempTime"`
		Capacity          int           `json:"capacity"`
		InferredPosition  bool          `json:"inferredPosition"`
		PowerDistributors []interface{} `json:"powerDistributors"`
		Height            int           `json:"height"`
		Children          []interface{} `json:"children"`
		UnconnectedPowerSupplies struct {
			Num1 struct {
				ID   int    `json:"id"`

				Side string `json:"side"`
			} `json:"1"`
			Num2 struct {
				ID   int    `json:"id"`
				Side string `json:"side"`
			} `json:"2"`
			Num3 struct {
				ID   int    `json:"id"`
				Side string `json:"side"`
			} `json:"3"`
			Num4 struct {
				ID   int    `json:"id"`
				Side string `json:"side"`
			} `json:"4"`
			Num5 struct {
				ID   int    `json:"id"`
				Side string `json:"side"`
			} `json:"5"`
			Num6 struct {
				ID   int    `json:"id"`
				Side string `json:"side"`
			} `json:"6"`
		} `json:"unconnectedPowerSupplies"`
	} `json:"children"`
	UnconnectedPowerSupplies struct {
		Num1 struct {
			ID   int    `json:"id"`
			Side string `json:"side"`
		} `json:"1"`
		Num2 struct {
			ID   int    `json:"id"`
			Side string `json:"side"`
		} `json:"2"`
	} `json:"unconnectedPowerSupplies"`
}

// ServerProfile - список профилей серверов OneView (ответ /rest/server-profiles)
type ServerProfile struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
	Members  []struct {
		Type                    string `json:"type"`
		URI                     string `json:"uri"`
		Name                    string `json:"name"`
		Description             string `json:"description"`
		SerialNumber            string `json:"serialNumber"`
		UUID                    string `json:"uuid"`
		ServerProfileTemplateURI string `json:"serverProfileTemplateUri"`
		TemplateCompliance      string `json:"templateCompliance"`
		ServerHardwareURI       string `json:"serverHardwareUri"`

		ServerHardwareTypeURI string      `json:"serverHardwareTypeUri"`
		EnclosureGroupURI    string      `json:"enclosureGroupUri"`
		EnclosureURI         string      `json:"enclosureUri"`
		EnclosureBay         int         `json:"enclosureBay"`
		Affinity             string      `json:"affinity"`
		AssociatedServer     any         `json:"associatedServer"`
		Firmware             struct {
			FirmwareBaselineURI       string `json:"firmwareBaselineUri"`
			ManageFirmware            bool   `json:"manageFirmware"`
			ForceInstallFirmware      bool   `json:"forceInstallFirmware"`
			FirmwareInstallType       string `json:"firmwareInstallType"`
			FirmwareScheduleDateTime  string `json:"firmwareScheduleDateTime"`
			FirmwareActivationType    any    `json:"firmwareActivationType"`
			ReapplyState              string `json:"reapplyState"`
		} `json:"firmware"`
		MacType          string `json:"macType"`
		WwnType          string `json:"wwnType"`
		SerialNumberType string `json:"serialNumberType"`
		Category         string `json:"category"`
		Created          time.Time `json:"created"`
		Modified         time.Time `json:"modified"`
		Status           string `json:"status"`
		State            string `json:"state"`
		TaskURI          string `json:"taskUri"`
		ConnectionSettings struct {
			ReapplyState string `json:"reapplyState"`
			Connections  []struct {
				MacType          string `json:"macType"`
				InterconnectPort int    `json:"interconnectPort"`
				NetworkName      any    `json:"networkName"`
				LagName          string `json:"lagName"`
				PortID           string `json:"portId"`
				RequestedMbps    string `json:"requestedMbps"`
				Mac              string `json:"mac"`
				Ipv4             any    `json:"ipv4"`
				Managed          bool   `json:"managed"`
				Wwnn             any    `json:"wwnn"`
				ID               int    `json:"id"`
				FunctionType     string `json:"functionType"`
				State            string `json:"state"`
				Boot             struct {
					Priority string `json:"priority"`
				} `json:"boot"`
				NetworkURI          string `json:"networkUri"`
				InterconnectURI     string `json:"interconnectUri"`
				PrivateVlanPortType string `json:"privateVlanPortType"`
				Wwpn                any    `json:"wwpn"`
				RequestedVFs        string `json:"requestedVFs"`
				MaximumMbps         int    `json:"maximumMbps"`

				IsolatedTrunk bool   `json:"isolatedTrunk"`
				AllocatedVFs  int    `json:"allocatedVFs"`
				Name          string `json:"name"`
				AllocatedMbps int    `json:"allocatedMbps"`
				WwpnType      string `json:"wwpnType"`
				Status        string `json:"status"`
			} `json:"connections"`
		} `json:"connectionSettings"`
		Boot struct {
			ManageBoot bool     `json:"manageBoot"`
			Order      []string `json:"order"`
		} `json:"boot"`
		OsDeploymentSettings struct {
			OsDeploymentPlanURI  any    `json:"osDeploymentPlanUri"`
			OsVolumeURI          any    `json:"osVolumeUri"`
			ForceOsDeployment    bool   `json:"forceOsDeployment"`
			OsCustomAttributes   []any  `json:"osCustomAttributes"`
			ReapplyState         string `json:"reapplyState"`
			OsDeploymentPlanName any    `json:"osDeploymentPlanName"`
		} `json:"osDeploymentSettings"`
		ETag              string `json:"eTag"`
		ID                string `json:"id"`
		OrderedStatus     int    `json:"_orderedStatus"`
		OriginalURI       string `json:"originalUri"`
		Appluri           string `json:"appluri"`
		ApplianceName     string `json:"applianceName"`
		ApplianceLocation string `json:"applianceLocation"`
		ServerName        string `json:"serverName"`
		ServerHardwareTypeName string `json:"serverHardwareTypeName"`
		EnclosureGroupName string `json:"enclosureGroupName"`
	} `json:"members"`
	Start       int    `json:"start"`
	Total       int    `json:"total"`
	PrevPageURI any    `json:"prevPageUri"`
	NextPageURI string `json:"nextPageUri"`
	URI         string `json:"uri"`
}

// ServerHardware300 - структура ответа /rest/server-hardware через Redfish-парсер (endpoint.Version == 300)
type ServerHardware300 struct {
	Category    string      `json:"category"`
	Count       int         `json:"count"`
	Members     []Members300 `json:"members"`
	Start       int         `json:"start"`
	Total       int         `json:"total"`
	PrevPageURI any         `json:"prevPageUri"`
	NextPageURI string      `json:"nextPageUri"`
	URI         string      `json:"uri"`
}

// Members300 - элемент списка серверов в ответе Redfish-парсера (endpoint.Version == 300)
type Members300 struct {
	Type            string      `json:"type"`
	Name            string      `json:"name"`
	ServerName      string      `json:"serverName"`
	State           string      `json:"state"`
	StateReason     string      `json:"stateReason"`
	AssetTag        string      `json:"assetTag"`
	Category        string      `json:"category"`
	Created         time.Time   `json:"created"`
	Description     any         `json:"description"`
	ETag            string      `json:"eTag"`
	FormFactor      string      `json:"formFactor"`
	Generation      string      `json:"generation"`
	HostOsType      int         `json:"hostOsType"`
	LicensingIntent string      `json:"licensingIntent"`
	LocationURI     string      `json:"locationUri"`
	MaintenanceMode bool        `json:"maintenanceMode"`
	MemoryMb        int         `json:"memoryMb"`
	Model           string      `json:"model"`
	Modified        time.Time   `json:"modified"`
	MpFirmwareVersion string    `json:"mpFirmwareVersion"`
	MpHostInfo      struct {
		MpHostName    string `json:"mpHostName"`
		MpIPAddresses []struct {
			Address string `json:"address"`
			Type    string `json:"type"`
		} `json:"mpIpAddresses"`
	} `json:"mpHostInfo"`
	MpLicenseType   string `json:"mpLicenseType"`
	MpModel         string `json:"mpModel"`
	MpState         string `json:"mpState"`
	OperatingSystem string `json:"operatingSystem"`
	PartNumber      string `json:"partNumber"`
	PhysicalServerHardwareURI any `json:"physicalServerHardwareUri"`
	Platform        string `json:"platform"`
	PortMap         struct {
		DeviceSlots []struct {
			PhysicalPorts []any  `json:"physicalPorts"`
			SlotNumber    int    `json:"slotNumber"`
			Location      string `json:"location"`
			DeviceName    string `json:"deviceName"`
			DeviceNumber  int    `json:"deviceNumber"`
		} `json:"deviceSlots"`
	} `json:"portMap"`
	Position           int    `json:"position"`
	PowerLock          bool   `json:"powerLock"`
	PowerState         string `json:"powerState"`
	ProcessorCoreCount int    `json:"processorCoreCount"`
	ProcessorCount     int    `json:"processorCount"`

	ProcessorSpeedMhz     int    `json:"processorSpeedMhz"`
	ProcessorType         string `json:"processorType"`
	RefreshState          string `json:"refreshState"`
	RemoteSupportSettings struct {
		RemoteSupportCurrentState string `json:"remoteSupportCurrentState"`
		Destination               string `json:"destination"`
	} `json:"remoteSupportSettings"`
	RomVersion            string `json:"romVersion"`
	SerialNumber          string `json:"serialNumber"`
	ServerGroupURI        string `json:"serverGroupUri"`
	ServerHardwareTypeURI string `json:"serverHardwareTypeUri"`
	ServerProfileURI      string `json:"serverProfileUri"`
	ShortModel            string `json:"shortModel"`
	Status                string `json:"status"`
	SubResources          struct {
		Memory struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				PartNumber                    string `json:"PartNumber"`
				PersistentRegionSizeLimitMiB int     `json:"PersistentRegionSizeLimitMiB"`
				OdataType                    string `json:"@odata.type"`
				LogicalSizeMiB               int    `json:"LogicalSizeMiB"`
				RankCount                    int    `json:"RankCount"`
				SecurityCapabilities         struct {
				} `json:"SecurityCapabilities"`
				BusWidthBits         int      `json:"BusWidthBits"`
				CacheSizeMiB         int      `json:"CacheSizeMiB"`
				Name                 string   `json:"Name"`
				NonVolatileSizeMiB   int      `json:"NonVolatileSizeMiB"`
				OdataID              string   `json:"@odata.id"`
				OperatingSpeedMhz    int      `json:"OperatingSpeedMhz,omitempty"`
				OperatingMemoryModes []string `json:"OperatingMemoryModes"`
				BaseModuleType       string   `json:"BaseModuleType,omitempty"`
				Manufacturer         string   `json:"Manufacturer,omitempty"`
				MemoryLocation       struct {
					Channel          int `json:"Channel"`
					Slot             int `json:"Slot"`
					MemoryController int `json:"MemoryController"`
					Socket           int `json:"Socket"`
				} `json:"MemoryLocation"`
				DeviceLocator string `json:"DeviceLocator"`
				MemoryType    string `json:"MemoryType"`
				Status        struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				VendorID string `json:"VendorID"`

				OdataContext      string   `json:"@odata.context"`
				DataWidthBits     int      `json:"DataWidthBits"`
				CapacityMiB       int      `json:"CapacityMiB"`
				VolatileSizeMiB   int      `json:"VolatileSizeMiB"`
				ErrorCorrection   string   `json:"ErrorCorrection"`
				MemoryDeviceType  string   `json:"MemoryDeviceType,omitempty"`
				SerialNumber      string   `json:"SerialNumber"`
				MemoryMedia       []string `json:"MemoryMedia"`
				Oem               struct {
					Hpe struct {
						VendorName             string   `json:"VendorName"`
						PartNumber             string   `json:"PartNumber"`
						DIMMStatus             string   `json:"DIMMStatus"`
						OdataType              string   `json:"@odata.type"`
						DIMMManufacturingDate  string   `json:"DIMMManufacturingDate"`
						MinimumVoltageVoltsX10 int      `json:"MinimumVoltageVoltsX10"`
						Attributes             []string `json:"Attributes"`
						BaseModuleType         string   `json:"BaseModuleType"`
						MaxOperatingSpeedMTs   int      `json:"MaxOperatingSpeedMTs"`
						OdataContext           string   `json:"@odata.context"`
					} `json:"Hpe"`
				} `json:"Oem"`
				VolatileRegionSizeLimitMiB int    `json:"VolatileRegionSizeLimitMiB"`
				ID                         string `json:"Id"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Memory"`
		SoftwareInventory struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				OdataID     string `json:"@odata.id"`
				Description string `json:"Description"`
				Version     string `json:"Version"`
				OdataType   string `json:"@odata.type"`
				Oem         struct {
					Hpe struct {
						OdataType    string   `json:"@odata.type"`
						DeviceClass  string   `json:"DeviceClass"`
						Targets      []string `json:"Targets"`
						OdataContext string   `json:"@odata.context"`
					} `json:"Hpe"`
				} `json:"Oem"`
				Updateable   bool   `json:"Updateable"`
				OdataContext string `json:"@odata.context"`
				ID           string `json:"Id"`

				Name string `json:"Name"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"SoftwareInventory"`
		MemoryList struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
				BoardCPUNumber            int `json:"BoardCpuNumber"`
				BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
				BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
				BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"MemoryList"`
		LocalStorageV2 struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					HealthRollup string `json:"HealthRollup"`
					State        string `json:"State"`
				} `json:"Status"`
				Controllers []struct {
					SpeedGbps int `json:"SpeedGbps"`
					Status    struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					PCIeInterface struct {
						PCIeType    string `json:"PCIeType"`
						MaxPCIeType string `json:"MaxPCIeType"`
						MaxLanes    int    `json:"MaxLanes"`
						LanesInUse  int    `json:"LanesInUse"`
					} `json:"PCIeInterface"`
					PartNumber      string `json:"PartNumber"`
					OdataType       string `json:"@odata.type"`
					ControllerRates struct {
						RedfishWriteableProperties                       []string `json:"@Redfish.WriteableProperties"`
						TransformationRatePercentRedfishAllowableNumbers []string `json:"TransformationRatePercent@Redfish.AllowableNumbers"`
						ConsistencyCheckRatePercent                      int      `json:"ConsistencyCheckRatePercent"`
						RebuildRatePercent                               int      `json:"RebuildRatePercent"`

						ConsistencyCheckRatePercentRedfishAllowableNumbers []string `json:"ConsistencyCheckRatePercent@Redfish.AllowableNumbers"`
						RebuildRatePercentRedfishAllowableNumbers          []string `json:"RebuildRatePercent@Redfish.AllowableNumbers"`
						TransformationRatePercent                          int      `json:"TransformationRatePercent"`
					} `json:"ControllerRates"`
					OdataEtag string `json:"@odata.etag"`
					Ports     struct {
						OdataID string `json:"@odata.id"`
					} `json:"Ports"`
					CacheSummary struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
						PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
					} `json:"CacheSummary"`
					SupportedRAIDTypes           []string `json:"SupportedRAIDTypes"`
					FirmwareVersion               string   `json:"FirmwareVersion"`
					Name                          string   `json:"Name"`
					OdataID                       string   `json:"@odata.id"`
					SupportedControllerProtocols  []string `json:"SupportedControllerProtocols"`
					SerialNumber                  string   `json:"SerialNumber"`
					Manufacturer                  string   `json:"Manufacturer"`
					Model                         string   `json:"Model"`
					Links                         struct {
						BatteriesOdataCount int `json:"Batteries@odata.count"`
						Batteries           []struct {
							OdataID string `json:"@odata.id"`
						} `json:"Batteries"`
					} `json:"Links"`
					SupportedDeviceProtocols []string `json:"SupportedDeviceProtocols"`
					ID                       string   `json:"Id"`
					Identifiers              []struct {
						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
					Sku      string `json:"SKU"`
					Location struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"Location"`
				} `json:"Controllers"`
				Actions struct {
					StorageResetToDefaults struct {
						ResetTypeRedfishAllowableValues []string `json:"ResetType@Redfish.AllowableValues"`
						Target                           string   `json:"target"`

					} `json:"#Storage.ResetToDefaults"`
				} `json:"Actions"`
				OdataType string `json:"@odata.type"`
				Drives    []struct {
					RedfishWriteableProperties []string `json:"@Redfish.WriteableProperties"`
					HotspareType               string   `json:"HotspareType"`
					OdataType                  string   `json:"@odata.type"`
					CapacityBytes              int64    `json:"CapacityBytes"`
					Name                       string   `json:"Name"`
					StatusIndicator            string   `json:"StatusIndicator"`
					OdataID                    string   `json:"@odata.id"`
					EncryptionAbility          string   `json:"EncryptionAbility"`
					NegotiatedSpeedGbs         int      `json:"NegotiatedSpeedGbs"`
					MediaType                  string   `json:"MediaType"`
					LocationIndicatorActive    bool     `json:"LocationIndicatorActive"`
					Protocol                   string   `json:"Protocol"`
					Operations                 []any    `json:"Operations"`
					PhysicalLocation           struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"PhysicalLocation"`
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					OdataEtag         string `json:"@odata.etag"`
					WriteCacheEnabled bool   `json:"WriteCacheEnabled"`
					BlockSizeBytes    int    `json:"BlockSizeBytes"`
					CapableSpeedGbs   int    `json:"CapableSpeedGbs"`
					Multipath         bool   `json:"Multipath"`
					FailurePredicted  bool   `json:"FailurePredicted"`
					Revision          string `json:"Revision"`
					SerialNumber      string `json:"SerialNumber"`
					Model             string `json:"Model"`
					RotationSpeedRPM  int    `json:"RotationSpeedRPM"`
					Links             struct {
						Volumes []struct {
							OdataID string `json:"@odata.id"`
						} `json:"Volumes"`
						Storage struct {
							OdataID string `json:"@odata.id"`
						} `json:"Storage"`
						VolumesOdataCount int `json:"Volumes@odata.count"`
					} `json:"Links"`
					ID          string `json:"Id"`
					Identifiers []struct {

						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
				} `json:"Drives"`
				OdataEtag string `json:"@odata.etag"`
				Name      string `json:"Name"`
				OdataID   string `json:"@odata.id"`
				Volumes   []struct {
					RedfishWriteableProperties        []string `json:"@Redfish.WriteableProperties"`
					WriteCachePolicyRedfishAllowableValues []string `json:"WriteCachePolicy@Redfish.AllowableValues"`
					OdataType                          string   `json:"@odata.type"`
					Encrypted                          bool     `json:"Encrypted"`
					CapacityBytes                      int64    `json:"CapacityBytes"`
					Name                                string   `json:"Name"`
					OptimumIOSizeBytes                 int      `json:"OptimumIOSizeBytes"`
					DisplayNameRedfishAllowablePattern  string   `json:"DisplayName@Redfish.AllowablePattern"`
					OdataID                             string   `json:"@odata.id"`
					DisplayName                         string   `json:"DisplayName"`
					StripSizeBytes                      int      `json:"StripSizeBytes"`
					Operations                          []any    `json:"Operations"`
					Status                              struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					ReadCachePolicy   string `json:"ReadCachePolicy"`
					EncryptionTypes   []any  `json:"EncryptionTypes"`
					OdataEtag         string `json:"@odata.etag"`
					RAIDType          string `json:"RAIDType"`
					WriteCachePolicy  string `json:"WriteCachePolicy"`
					BlockSizeBytes    int    `json:"BlockSizeBytes"`
					LogicalUnitNumber int    `json:"LogicalUnitNumber"`
					ReadCachePolicyRedfishAllowableValues []string `json:"ReadCachePolicy@Redfish.AllowableValues"`
					Links             struct {
						RedfishWriteableProperties  []any `json:"@Redfish.WriteableProperties"`
						DedicatedSpareDrives         []any `json:"DedicatedSpareDrives"`
						DedicatedSpareDrivesOdataCount int `json:"DedicatedSpareDrives@odata.count"`
						DrivesOdataCount             int   `json:"Drives@odata.count"`
						Drives                       []struct {
							OdataID string `json:"@odata.id"`
						} `json:"Drives"`
					} `json:"Links"`
					ID          string `json:"Id"`
					Identifiers []struct {
						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
					WriteCacheState string `json:"WriteCacheState"`
					VolumeUsage     string `json:"VolumeUsage"`
				} `json:"Volumes"`

				Annotations struct {
					Volumes struct {
						WriteCachePolicyRedfishAllowableValues     []string `json:"WriteCachePolicy@Redfish.AllowableValues"`
						DisplayNameRedfishUpdatableAfterCreate     bool     `json:"DisplayName@Redfish.UpdatableAfterCreate"`
						OdataType                                  string   `json:"@odata.type"`
						Name                                       string   `json:"Name"`
						InitializeMethodRedfishOptionalOnCreate    bool     `json:"InitializeMethod@Redfish.OptionalOnCreate"`
						DisplayNameRedfishAllowablePattern         string   `json:"DisplayName@Redfish.AllowablePattern"`
						VolumeUsageRedfishAllowableValues          []string `json:"VolumeUsage@Redfish.AllowableValues"`
						OdataID                                    string   `json:"@odata.id"`
						NameRedfishOptionalOnCreate                bool     `json:"Name@Redfish.OptionalOnCreate"`
						IOPerfModeEnabledRedfishOptionalOnCreate   bool     `json:"IOPerfModeEnabled@Redfish.OptionalOnCreate"`
						ReadCachePolicyRedfishUpdatableAfterCreate bool     `json:"ReadCachePolicy@Redfish.UpdatableAfterCreate"`
						InitializeMethodRedfishAllowableValues     []string `json:"InitializeMethod@Redfish.AllowableValues"`
						MediaSpanCountRedfishOptionalOnCreate      bool     `json:"MediaSpanCount@Redfish.OptionalOnCreate"`
						MediaSpanCountRedfishAllowableNumbers      []any    `json:"MediaSpanCount@Redfish.AllowableNumbers"`
						StripSizeBytesRedfishOptionalOnCreate      bool     `json:"StripSizeBytes@Redfish.OptionalOnCreate"`
						WriteCachePolicyRedfishOptionalOnCreate    bool     `json:"WriteCachePolicy@Redfish.OptionalOnCreate"`
						VolumeUsageRedfishOptionalOnCreate         bool     `json:"VolumeUsage@Redfish.OptionalOnCreate"`
						OdataEtag                                  string   `json:"@odata.etag"`
						RAIDTypeRedfishRequiredOnCreate            bool     `json:"RAIDType@Redfish.RequiredOnCreate"`
						ReadCachePolicyRedfishOptionalOnCreate     bool     `json:"ReadCachePolicy@Redfish.OptionalOnCreate"`
						CapacityBytesRedfishOptionalOnCreate       bool     `json:"CapacityBytes@Redfish.OptionalOnCreate"`
						ReadCachePolicyRedfishAllowableValues      []string `json:"ReadCachePolicy@Redfish.AllowableValues"`
						CapacityBytesRedfishAllowableNumbers       []any    `json:"CapacityBytes@Redfish.AllowableNumbers"`
						LinksRedfishRequiredOnCreate               bool     `json:"Links@Redfish.RequiredOnCreate"`
						NameRedfishAllowablePattern                string   `json:"Name@Redfish.AllowablePattern"`
						StripSizeBytesRedfishAllowableNumbers      []string `json:"StripSizeBytes@Redfish.AllowableNumbers"`
						WriteCachePolicyRedfishUpdatableAfterCreate bool    `json:"WriteCachePolicy@Redfish.UpdatableAfterCreate"`
						Links                                      struct {
							DedicatedSpareDrivesRedfishOptionalOnCreate  bool `json:"DedicatedSpareDrives@Redfish.OptionalOnCreate"`
							DedicatedSpareDrivesRedfishUpdatableAfterCreate bool `json:"DedicatedSpareDrives@Redfish.UpdatableAfterCreate"`
							DrivesRedfishRequiredOnCreate                bool `json:"Drives@Redfish.RequiredOnCreate"`
						} `json:"Links"`
						ID                                        string   `json:"Id"`
						RAIDTypeRedfishAllowableValues             []string `json:"RAIDType@Redfish.AllowableValues"`
						IOPerfModeEnabledRedfishUpdatableAfterCreate bool   `json:"IOPerfModeEnabled@Redfish.UpdatableAfterCreate"`
						DisplayNameRedfishOptionalOnCreate         bool     `json:"DisplayName@Redfish.OptionalOnCreate"`
					} `json:"Volumes"`
				} `json:"Annotations"`
				StorageControllers []struct {
					SpeedGbps int `json:"SpeedGbps"`
					Status    struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					PCIeInterface struct {
						PCIeType    string `json:"PCIeType"`
						MaxPCIeType string `json:"MaxPCIeType"`

						MaxLanes   int `json:"MaxLanes"`
						LanesInUse int `json:"LanesInUse"`
					} `json:"PCIeInterface"`
					PartNumber      string `json:"PartNumber"`
					OdataType       string `json:"@odata.type"`
					ControllerRates struct {
						RedfishWriteableProperties                         []string `json:"@Redfish.WriteableProperties"`
						TransformationRatePercentRedfishAllowableNumbers   []string `json:"TransformationRatePercent@Redfish.AllowableNumbers"`
						ConsistencyCheckRatePercent                        int      `json:"ConsistencyCheckRatePercent"`
						RebuildRatePercent                                 int      `json:"RebuildRatePercent"`
						ConsistencyCheckRatePercentRedfishAllowableNumbers []string `json:"ConsistencyCheckRatePercent@Redfish.AllowableNumbers"`
						RebuildRatePercentRedfishAllowableNumbers          []string `json:"RebuildRatePercent@Redfish.AllowableNumbers"`
						TransformationRatePercent                          int      `json:"TransformationRatePercent"`
					} `json:"ControllerRates"`
					OdataEtag string `json:"@odata.etag"`
					Ports     struct {
						OdataID string `json:"@odata.id"`
					} `json:"Ports"`
					CacheSummary struct {
						Status struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
						PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
					} `json:"CacheSummary"`
					SupportedRAIDTypes           []string `json:"SupportedRAIDTypes"`
					FirmwareVersion               string   `json:"FirmwareVersion"`
					Name                          string   `json:"Name"`
					OdataID                       string   `json:"@odata.id"`
					SupportedControllerProtocols  []string `json:"SupportedControllerProtocols"`
					SerialNumber                  string   `json:"SerialNumber"`
					Manufacturer                  string   `json:"Manufacturer"`
					Model                         string   `json:"Model"`
					Links                         struct {
						BatteriesOdataCount int `json:"Batteries@odata.count"`
						Batteries           []struct {
							OdataID string `json:"@odata.id"`
						} `json:"Batteries"`
					} `json:"Links"`
					SupportedDeviceProtocols []string `json:"SupportedDeviceProtocols"`
					ID                       string   `json:"Id"`
					Identifiers              []struct {
						DurableNameFormat string `json:"DurableNameFormat"`
						DurableName       string `json:"DurableName"`
					} `json:"Identifiers"`
					Sku      string `json:"SKU"`
					Location struct {
						PartLocation struct {

							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"Location"`
				} `json:"StorageControllers"`
				DrivesOdataCount int    `json:"Drives@odata.count"`
				ID               string `json:"Id"`
				AutoVolumeCreate string `json:"AutoVolumeCreate"`
				EncryptionMode   string `json:"EncryptionMode"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"LocalStorageV2"`
		Thermal struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				MemberID string `json:"MemberId"`
				OdataID  string `json:"@odata.id"`
				Reading  int    `json:"Reading"`
				Oem      struct {
					Hpe struct {
						OdataType    string `json:"@odata.type"`
						HotPluggable bool   `json:"HotPluggable"`
						Redundant    bool   `json:"Redundant"`
						OdataContext string `json:"@odata.context"`
						Location     string `json:"Location"`
					} `json:"Hpe"`
				} `json:"Oem"`
				ReadingUnits    string `json:"ReadingUnits"`
				Name            string `json:"Name"`
				PhysicalContext string `json:"PhysicalContext"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Thermal"`
		AdvancedMemoryProtection struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {

				AmpModeActive    string   `json:"AmpModeActive"`
				OdataType        string   `json:"@odata.type"`
				AmpModeSupported []string `json:"AmpModeSupported"`
				OdataContext     string   `json:"@odata.context"`
				AmpModeStatus    string   `json:"AmpModeStatus"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"AdvancedMemoryProtection"`
		Devices struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					State string `json:"State"`
				} `json:"Status,omitempty"`
				PartNumber string `json:"PartNumber"`
				OdataType  string `json:"@odata.type"`
				PCISlot    struct {
					LinkLanes string `json:"LinkLanes"`
					Status    struct {
						OperationalStatus []struct {
							Status string `json:"Status"`
						} `json:"OperationalStatus"`
					} `json:"Status"`
					DeviceNumber   int    `json:"DeviceNumber"`
					OdataType      string `json:"@odata.type"`
					OdataEtag      string `json:"@odata.etag"`
					OdataContext   string `json:"@odata.context"`
					Name           string `json:"Name"`
					UEFIDevicePath string `json:"UEFIDevicePath"`
					BusNumber      int    `json:"BusNumber"`
					SupportsHotPlug bool  `json:"SupportsHotPlug"`
					OdataID        string `json:"@odata.id"`
					FunctionNumber int    `json:"FunctionNumber"`
					Characteristics struct {
						SupportsPowerManagementEventSignal bool `json:"SupportsPowerManagementEventSignal"`
						Provides33Volts                    bool `json:"Provides3_3Volts"`
					} `json:"Characteristics"`
					Length     string `json:"Length"`
					Technology string `json:"Technology"`
					ID         string `json:"Id"`
				} `json:"PCISlot,omitempty"`
				OdataContext    string `json:"@odata.context"`
				ProductVersion  string `json:"ProductVersion"`
				FirmwareVersion struct {
					Current struct {

						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				Name              string `json:"Name"`
				DeviceType        string `json:"DeviceType"`
				OdataID           string `json:"@odata.id"`
				SerialNumber      string `json:"SerialNumber"`
				ProductPartNumber string `json:"ProductPartNumber"`
				Manufacturer      string `json:"Manufacturer"`
				Links             struct {
					PCISlot struct {
						OdataID string `json:"@odata.id"`
					} `json:"PCISlot"`
				} `json:"Links,omitempty"`
				ID                   string `json:"Id"`
				MCTPProtocolDisabled bool   `json:"MCTPProtocolDisabled,omitempty"`
				Location             string `json:"Location"`
				Status0              struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
				DeviceInstances []struct {
					OdataID string `json:"@odata.id"`
				} `json:"DeviceInstances,omitempty"`
				Status1 struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
				Status2 struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
				Status3 struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Devices"`
		PowerSupplies struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []any     `json:"data"`
			Etag            string    `json:"etag"`
			Count           int       `json:"count"`
			Name            string    `json:"name"`

		} `json:"PowerSupplies"`
		LocalStorage struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				ControllerPartNumber            string `json:"ControllerPartNumber"`
				EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
				CurrentOperatingMode            string `json:"CurrentOperatingMode"`
				Description                     string `json:"Description"`
				LocationFormat                  string `json:"LocationFormat"`
				EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
				OdataType                       string `json:"@odata.type"`
				AdapterType                     string `json:"AdapterType"`
				PhysicalDrives                  []struct {
					CarrierApplicationVersion string `json:"CarrierApplicationVersion"`
					Description                string `json:"Description"`
					LocationFormat             string `json:"LocationFormat"`
					CapacityLogicalBlocks      int    `json:"CapacityLogicalBlocks"`
					UncorrectedReadErrors      int    `json:"UncorrectedReadErrors"`
					OdataType                  string `json:"@odata.type"`
					PowerOnHours               any    `json:"PowerOnHours"`
					MaximumTemperatureCelsius  int    `json:"MaximumTemperatureCelsius"`
					Name                       string `json:"Name"`
					OdataID                    string `json:"@odata.id"`
					InterfaceType              string `json:"InterfaceType"`
					DiskDriveStatusReasons     []string `json:"DiskDriveStatusReasons"`
					CarrierAuthenticationStatus string `json:"CarrierAuthenticationStatus"`
					MediaType                  string `json:"MediaType"`
					CurrentTemperatureCelsius  int    `json:"CurrentTemperatureCelsius"`
					InterfaceSpeedMbps         int    `json:"InterfaceSpeedMbps"`
					UncorrectedWriteErrors     int    `json:"UncorrectedWriteErrors"`
					Status                     struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					CapacityGB               int    `json:"CapacityGB"`
					OdataEtag                string `json:"@odata.etag"`
					OdataContext             string `json:"@odata.context"`
					BlockSizeBytes           int    `json:"BlockSizeBytes"`
					SSDEnduranceUtilizationPercentage any `json:"SSDEnduranceUtilizationPercentage"`
					CapacityMiB              int    `json:"CapacityMiB"`
					FirmwareVersion          struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					LegacyBootPriority string `json:"LegacyBootPriority"`
					SerialNumber       string `json:"SerialNumber"`

					Model              string `json:"Model"`
					RotationalSpeedRpm int    `json:"RotationalSpeedRpm"`
					ID                 string `json:"Id"`
					DiskDriveUse       string `json:"DiskDriveUse"`
					EncryptedDrive     bool   `json:"EncryptedDrive"`
					Location           string `json:"Location"`
				} `json:"PhysicalDrives"`
				LogicalDrives []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LogicalDriveEncryption bool   `json:"LogicalDriveEncryption"`
					Description             string `json:"Description"`
					StripeSizeBytes         int    `json:"StripeSizeBytes"`
					DataDrives             []struct {
						CarrierApplicationVersion string `json:"CarrierApplicationVersion"`
						Description                string `json:"Description"`
						LocationFormat             string `json:"LocationFormat"`
						CapacityLogicalBlocks      int    `json:"CapacityLogicalBlocks"`
						UncorrectedReadErrors      int    `json:"UncorrectedReadErrors"`
						OdataType                  string `json:"@odata.type"`
						PowerOnHours               any    `json:"PowerOnHours"`
						MaximumTemperatureCelsius  int    `json:"MaximumTemperatureCelsius"`
						Name                       string `json:"Name"`
						OdataID                    string `json:"@odata.id"`
						InterfaceType              string `json:"InterfaceType"`
						DiskDriveStatusReasons     []string `json:"DiskDriveStatusReasons"`
						CarrierAuthenticationStatus string `json:"CarrierAuthenticationStatus"`
						MediaType                  string `json:"MediaType"`
						CurrentTemperatureCelsius  int    `json:"CurrentTemperatureCelsius"`
						InterfaceSpeedMbps         int    `json:"InterfaceSpeedMbps"`
						UncorrectedWriteErrors     int    `json:"UncorrectedWriteErrors"`
						Status                     struct {
							Health string `json:"Health"`
							State  string `json:"State"`
						} `json:"Status"`
						CapacityGB   int    `json:"CapacityGB"`
						OdataContext string `json:"@odata.context"`
						BlockSizeBytes int  `json:"BlockSizeBytes"`
						SSDEnduranceUtilizationPercentage any `json:"SSDEnduranceUtilizationPercentage"`
						CapacityMiB  int    `json:"CapacityMiB"`
						FirmwareVersion struct {
							Current struct {
								VersionString string `json:"VersionString"`
							} `json:"Current"`
						} `json:"FirmwareVersion"`
						LegacyBootPriority string `json:"LegacyBootPriority"`
						SerialNumber       string `json:"SerialNumber"`

						Model              string `json:"Model"`
						RotationalSpeedRpm int    `json:"RotationalSpeedRpm"`
						ID                 string `json:"Id"`
						DiskDriveUse       string `json:"DiskDriveUse"`
						EncryptedDrive     bool   `json:"EncryptedDrive"`
						Location           string `json:"Location"`
					} `json:"DataDrives"`
					OdataType   string `json:"@odata.type"`
					OdataEtag   string `json:"@odata.etag"`
					OdataContext string `json:"@odata.context"`
					Raid        string `json:"Raid"`
					CapacityMiB int    `json:"CapacityMiB"`
					LogicalDriveType   string `json:"LogicalDriveType"`
					LogicalDriveNumber int    `json:"LogicalDriveNumber"`
					AccelerationMethod string `json:"AccelerationMethod"`
					Name        string `json:"Name"`
					OdataID     string `json:"@odata.id"`
					InterfaceType      string `json:"InterfaceType"`
					LegacyBootPriority string `json:"LegacyBootPriority"`
					Links       struct {
						DataDrives struct {
							OdataID string `json:"@odata.id"`
						} `json:"DataDrives"`
					} `json:"Links"`
					LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
					MediaType                 string   `json:"MediaType"`
					ID                        string   `json:"Id"`
					VolumeUniqueIdentifier    string   `json:"VolumeUniqueIdentifier"`
					LogicalDriveName          string   `json:"LogicalDriveName"`
				} `json:"LogicalDrives"`
				ReadCachePercent        int    `json:"ReadCachePercent"`
				InternalPortCount       int    `json:"InternalPortCount"`
				CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
				EncryptionFwLocked      bool   `json:"EncryptionFwLocked"`
				Name                    string `json:"Name"`
				OdataID                 string `json:"@odata.id"`
				BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
				DriveWriteCache         string `json:"DriveWriteCache"`
				DualModeControllerSupported bool `json:"DualModeControllerSupported"`
				HardwareRevision        string `json:"HardwareRevision"`
				Status                  struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				EncryptionSelfTestPassed bool   `json:"EncryptionSelfTestPassed"`
				ExternalPortCount        int    `json:"ExternalPortCount"`
				OdataContext             string `json:"@odata.context"`
				UnconfiguredDrives       []any  `json:"UnconfiguredDrives"`
				ControllerBoard          struct {

					Status struct {
						Health string `json:"Health"`
					} `json:"Status"`
				} `json:"ControllerBoard"`
				EncryptionEnabled bool `json:"EncryptionEnabled"`
				EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
				EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
				SerialNumber                       string `json:"SerialNumber"`
				CacheModuleStatus                  struct {
					Health string `json:"Health"`
				} `json:"CacheModuleStatus"`
				WriteCacheWithoutBackupPowerEnabled bool `json:"WriteCacheWithoutBackupPowerEnabled"`
				StorageEnclosures                   []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					Description    string `json:"Description"`
					LocationFormat string `json:"LocationFormat"`
					OdataType      string `json:"@odata.type"`
					DriveBayCount  int    `json:"DriveBayCount"`
					OdataEtag      string `json:"@odata.etag"`
					OdataContext   string `json:"@odata.context"`
					FirmwareVersion struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					Name         string `json:"Name"`
					OdataID      string `json:"@odata.id"`
					SerialNumber string `json:"SerialNumber"`
					Model        string `json:"Model"`
					ID           string `json:"Id"`
					Location     string `json:"Location"`
				} `json:"StorageEnclosures"`
				CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
				Model                   string `json:"Model"`
				Links                   struct {
					StorageEnclosures struct {
						OdataID string `json:"@odata.id"`
					} `json:"StorageEnclosures"`
					PhysicalDrives struct {
						OdataID string `json:"@odata.id"`

					} `json:"PhysicalDrives"`
					LogicalDrives struct {
						OdataID string `json:"@odata.id"`
					} `json:"LogicalDrives"`
					UnconfiguredDrives struct {
						OdataID string `json:"@odata.id"`
					} `json:"UnconfiguredDrives"`
				} `json:"Links"`
				ID                       string `json:"Id"`
				WriteCacheBypassThresholdKB int `json:"WriteCacheBypassThresholdKB"`
				Location                 string `json:"Location"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"LocalStorage"`
		Chassis struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
				OdataType    string `json:"@odata.type"`
				OdataContext string `json:"@odata.context"`
				Name         string `json:"Name"`
				PCIeSlots    struct {
					OdataID string `json:"@odata.id"`
				} `json:"PCIeSlots,omitempty"`
				AssetTag    string `json:"AssetTag,omitempty"`
				OdataID     string `json:"@odata.id"`
				PCIeDevices struct {
					OdataID string `json:"@odata.id"`
				} `json:"PCIeDevices,omitempty"`
				SerialNumber string `json:"SerialNumber"`
				Oem          struct {
					Hpe struct {
						Actions struct {
							HpeServerChassisFactoryResetMCTP struct {
								Target string `json:"target"`
							} `json:"#HpeServerChassis.FactoryResetMCTP"`
							HpeServerChassisDisableMCTPOnServer struct {
								Target string `json:"target"`
							} `json:"#HpeServerChassis.DisableMCTPOnServer"`
						} `json:"Actions"`
						BayNumber int    `json:"BayNumber"`
						OdataType string `json:"@odata.type"`

						SystemMaintenanceSwitches struct {
							Sw5  string `json:"Sw5"`
							Sw4  string `json:"Sw4"`
							Sw12 string `json:"Sw12"`
							Sw7  string `json:"Sw7"`
							Sw11 string `json:"Sw11"`
							Sw6  string `json:"Sw6"`
							Sw10 string `json:"Sw10"`
							Sw9  string `json:"Sw9"`
							Sw8  string `json:"Sw8"`
							Sw1  string `json:"Sw1"`
							Sw3  string `json:"Sw3"`
							Sw2  string `json:"Sw2"`
						} `json:"SystemMaintenanceSwitches"`
						BaysConsumedHeight int    `json:"BaysConsumedHeight"`
						MCTPEnabledOnServer bool  `json:"MCTPEnabledOnServer"`
						OdataContext       string `json:"@odata.context"`
						BaysConsumedWidth  int    `json:"BaysConsumedWidth"`
						FormFactor         string `json:"FormFactor"`
						SmartStorageBattery []struct {
							Status struct {
								Health string `json:"Health"`
								State  string `json:"State"`
							} `json:"Status"`
							MaximumCapWatts          int    `json:"MaximumCapWatts"`
							ProductName              string `json:"ProductName"`
							SerialNumber             string `json:"SerialNumber"`
							Model                    string `json:"Model"`
							RemainingChargeTimeSeconds int  `json:"RemainingChargeTimeSeconds"`
							Index                    int    `json:"Index"`
							ChargeLevelPercent       int    `json:"ChargeLevelPercent"`
							SparePartNumber          string `json:"SparePartNumber"`
							FirmwareVersion          string `json:"FirmwareVersion"`
						} `json:"SmartStorageBattery"`
						Links struct {
							BaseFrus struct {
								OdataID string `json:"@odata.id"`
							} `json:"BaseFrus"`
							MezzFrus struct {
								OdataID string `json:"@odata.id"`
							} `json:"MezzFrus"`
							BladeEnclosure struct {
								OdataID string `json:"@odata.id"`
							} `json:"BladeEnclosure"`
							Devices struct {
								OdataID string `json:"@odata.id"`
							} `json:"Devices"`
						} `json:"Links"`
						Firmware struct {

							SystemProgrammableLogicDevice struct {
								Current struct {
									VersionString string `json:"VersionString"`
								} `json:"Current"`
							} `json:"SystemProgrammableLogicDevice"`
							PlatformDefinitionTable struct {
								Current struct {
									VersionString string `json:"VersionString"`
								} `json:"Current"`
							} `json:"PlatformDefinitionTable"`
							PowerManagementController struct {
								Current struct {
									VersionString string `json:"VersionString"`
								} `json:"Current"`
							} `json:"PowerManagementController"`
							SPSFirmwareVersionData struct {
								Current struct {
									VersionString string `json:"VersionString"`
								} `json:"Current"`
							} `json:"SPSFirmwareVersionData"`
							PowerManagementControllerBootloader struct {
								Current struct {
									VersionString string `json:"VersionString"`
									Family        string `json:"Family"`
								} `json:"Current"`
							} `json:"PowerManagementControllerBootloader"`
						} `json:"Firmware"`
						ElConfigOverride bool `json:"ElConfigOverride"`
						PowerAlertMode   struct {
							Activated       bool `json:"Activated"`
							AlertPowerWatts int  `json:"AlertPowerWatts"`
						} `json:"PowerAlertMode"`
					} `json:"Hpe"`
				} `json:"Oem"`
				Manufacturer string `json:"Manufacturer"`
				Model        string `json:"Model,omitempty"`
				Links        struct {
					ContainedBy struct {
						OdataID string `json:"@odata.id"`
					} `json:"ContainedBy"`
					ManagedBy []struct {
						OdataID string `json:"@odata.id"`
					} `json:"ManagedBy"`
					ComputerSystems []struct {
						OdataID string `json:"@odata.id"`
					} `json:"ComputerSystems"`
				} `json:"Links,omitempty"`
				NetworkAdapters struct {
					OdataID string `json:"@odata.id"`

				} `json:"NetworkAdapters,omitempty"`
				ID                      string `json:"Id"`
				ChassisType             string `json:"ChassisType"`
				LocationIndicatorActive bool   `json:"LocationIndicatorActive"`
				Sku                     string `json:"SKU,omitempty"`
				PowerState              string `json:"PowerState"`
				IndicatorLED            string `json:"IndicatorLED"`
				Power                   struct {
					OdataID string `json:"@odata.id"`
				} `json:"Power,omitempty"`
				Thermal struct {
					OdataID string `json:"@odata.id"`
				} `json:"Thermal,omitempty"`
				Status0 struct {
					Health string `json:"Health"`
				} `json:"Status,omitempty"`
				Drives struct {
					OdataID string `json:"@odata.id"`
				} `json:"Drives,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"Chassis"`
		FirmwareInventory struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status,omitempty"`
				OdataID     string `json:"@odata.id"`
				Description string `json:"Description"`
				Version     string `json:"Version"`
				OdataType   string `json:"@odata.type"`
				Oem         struct {
					Hpe struct {
						DeviceContext string   `json:"DeviceContext"`
						OdataType     string   `json:"@odata.type"`
						DeviceClass   string   `json:"DeviceClass"`
						Targets       []string `json:"Targets"`
						OdataContext  string   `json:"@odata.context"`
					} `json:"Hpe"`
				} `json:"Oem"`
				Updateable   bool   `json:"Updateable"`
				OdataContext string `json:"@odata.context"`
				ID           string `json:"Id"`

				Name         string `json:"Name"`
				SoftwareID   string `json:"SoftwareId,omitempty"`
				RelatedItem  []struct {
					OdataID string `json:"@odata.id"`
				} `json:"RelatedItem,omitempty"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"FirmwareInventory"`
		NetworkAdapters struct {
			URI             string    `json:"uri"`
			CollectionState string    `json:"collectionState"`
			Modified        time.Time `json:"modified"`
			Data            []struct {
				Controllers []struct {
					ControllerCapabilities struct {
						DataCenterBridging struct {
							Capable bool `json:"Capable"`
						} `json:"DataCenterBridging"`
						Npar struct {
							NparEnabled bool `json:"NparEnabled"`
							NparCapable bool `json:"NparCapable"`
						} `json:"NPAR"`
						VirtualizationOffload struct {
							VirtualFunction struct {
								DeviceMaxCount        int `json:"DeviceMaxCount"`
								NetworkPortMaxCount   int `json:"NetworkPortMaxCount"`
								MinAssignmentGroupSize int `json:"MinAssignmentGroupSize"`
							} `json:"VirtualFunction"`
							Sriov struct {
								SRIOVVEPACapable bool `json:"SRIOVVEPACapable"`
							} `json:"SRIOV"`
						} `json:"VirtualizationOffload"`
						Npiv struct {
							MaxDeviceLogins int `json:"MaxDeviceLogins"`
							MaxPortLogins   int `json:"MaxPortLogins"`
						} `json:"NPIV"`
						NetworkPortCount           int `json:"NetworkPortCount"`
						NetworkDeviceFunctionCount int `json:"NetworkDeviceFunctionCount"`
					} `json:"ControllerCapabilities"`
					FirmwarePackageVersion string `json:"FirmwarePackageVersion"`
					Links                  struct {
						NetworkDeviceFunctions []struct {
							OdataID string `json:"@odata.id"`
						} `json:"NetworkDeviceFunctions"`
						Ports []struct {
							OdataID string `json:"@odata.id"`
						} `json:"Ports"`

					} `json:"Links"`
					Location struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"Location"`
				} `json:"Controllers"`
				Description string `json:"Description"`
				PartNumber  string `json:"PartNumber"`
				Actions     struct {
					NetworkAdapterResetSettingsToDefault struct {
						Title  string `json:"title"`
						Target string `json:"target"`
					} `json:"#NetworkAdapter.ResetSettingsToDefault"`
				} `json:"Actions"`
				OdataType string `json:"@odata.type"`
				Ports     []struct {
					LinkStatus            string `json:"LinkStatus"`
					Description           string `json:"Description"`
					OdataType             string `json:"@odata.type"`
					SignalDetected        bool   `json:"SignalDetected"`
					LinkNetworkTechnology string `json:"LinkNetworkTechnology"`
					FunctionMinBandwidth  []struct {
						AllocationPercent int `json:"AllocationPercent"`
						NetworkDeviceFunction struct {
							OdataID string `json:"@odata.id"`
						} `json:"NetworkDeviceFunction"`
					} `json:"FunctionMinBandwidth"`
					PortID          string `json:"PortId"`
					EEEEnabled      bool   `json:"EEEEnabled"`
					OdataContext    string `json:"@odata.context"`
					Name            string `json:"Name"`
					PortMaximumMTU  int    `json:"PortMaximumMTU"`
					PortProtocol    string `json:"PortProtocol"`
					OdataID         string `json:"@odata.id"`
					LinkConfiguration []struct {
						AutoSpeedNegotiationCapable any   `json:"AutoSpeedNegotiationCapable"`
						CapableLinkSpeedGbps         []int `json:"CapableLinkSpeedGbps"`
					} `json:"LinkConfiguration"`
					CurrentSpeedGbps int `json:"CurrentSpeedGbps"`
					Oem              struct {
						Hpe struct {
							PortDiagnosticRemoteLoopbackCapable bool `json:"PortDiagnosticRemoteLoopbackCapable"`
							PortDiagnosticEnableRemoteLoopback  bool `json:"PortDiagnosticEnableRemoteLoopback"`
							ConfiguredNetworkLinks              []struct {
								ConfiguredLinkSpeedGbps int `json:"ConfiguredLinkSpeedGbps"`
							} `json:"ConfiguredNetworkLinks"`
						Actions struct {
							HpeNetworkPortDiagnosticsReset struct {
								Title  string `json:"title"`
								Target string `json:"target"`
							} `json:"#HpeNetworkPort.DiagnosticsReset"`
							NetworkPortDiagnosticsReset struct {
								Title  string `json:"title"`
								Target string `json:"target"`
							} `json:"#NetworkPort.DiagnosticsReset"`
						} `json:"Actions"`
					} `json:"Hpe"`
				} `json:"Oem"`
				OdataType0 string `json:"@odata.type"`
				Lldp      struct {
					OdataID string `json:"odataId"`
				} `json:"Lldp"`
				VirtualFunctionAllocations []struct {
					VirtualFunctionsAllocated int `json:"VirtualFunctionsAllocated"`
					NetworkDeviceFunction     struct {
						OdataID string `json:"odataId"`
					} `json:"NetworkDeviceFunction"`
				} `json:"VirtualFunctionAllocations"`
				OdataContext0          string `json:"@odata.context"`
				AutoNegotiationCapable bool  `json:"AutoNegotiationCapable"`
				Evb                    struct {
					OdataID string `json:"odataId"`
				} `json:"Evb"`
				PortDiagnosticEnableLocalLoopback bool `json:"PortDiagnosticEnableLocalLoopback"`
				EvbData                            struct {
					Receiving struct {
						CdcpActive                              bool `json:"CdcpActive"`
						ChannelLinkControlRemoteSequenceNumber  int  `json:"ChannelLinkControlRemoteSequenceNumber"`
						ChannelDescriptionLocalSequenceNumber   int  `json:"ChannelDescriptionLocalSequenceNumber"`
						ChannelLinkControlSubtype               int  `json:"ChannelLinkControlSubtype"`
						ChannelActive                           bool `json:"ChannelActive"`
						ChannelDescriptionRemoteSequenceNumber  int  `json:"ChannelDescriptionRemoteSequenceNumber"`
						ChannelLinkControlLocalSequenceNumber   int  `json:"ChannelLinkControlLocalSequenceNumber"`
						SChannelAssignments                     []struct {
							Svid int `json:"Svid"`
							Scid int `json:"Scid"`
						} `json:"SChannelAssignments"`
						Role                     string `json:"Role"`
						ChannelDescriptionSubtype int   `json:"ChannelDescriptionSubtype"`
						SChannelMaxCount         int    `json:"SChannelMaxCount"`
						SChannelsSupported       bool   `json:"SChannelsSupported"`
						ChannelLinkControlMCEnable bool `json:"ChannelLinkControlMCEnable"`
						VirtualLinkStatusVector  string `json:"VirtualLinkStatusVector"`
					} `json:"Receiving"`
					Transmitting struct {
						Role             string `json:"Role"`
						SChannelMaxCount int    `json:"SChannelMaxCount"`

						ChannelLinkControlRemoteSequenceNumber int `json:"ChannelLinkControlRemoteSequenceNumber"`
						ChannelDescriptionData []struct {
							Descriptor1Scid                    int    `json:"Descriptor1Scid"`
							Descriptor0Scid                     int    `json:"Descriptor0Scid"`
							Descriptor0BandwidthPirMbps         int    `json:"Descriptor0BandwidthPirMbps"`
							Descriptor0ChannelTerminationType   string `json:"Descriptor0ChannelTerminationType"`
							Descriptor0FCoESupport              bool   `json:"Descriptor0FCoESupport"`
							Descriptor1ConnectionID             any    `json:"Descriptor1ConnectionID"`
							Descriptor0ISCSISupport             bool   `json:"Descriptor0iSCSISupport"`
						} `json:"ChannelDescriptionData"`
						ChannelDescriptionLocalSequenceNumber int    `json:"ChannelDescriptionLocalSequenceNumber"`
						SChannelsSupported                     bool   `json:"SChannelsSupported"`
						ChannelDescriptionRemoteSequenceNumber int    `json:"ChannelDescriptionRemoteSequenceNumber"`
						ChannelLinkControlMCEnable             bool   `json:"ChannelLinkControlMCEnable"`
						ChannelDescriptorCount                 int    `json:"ChannelDescriptorCount"`
						ChannelLinkControlLocalSequenceNumber  int    `json:"ChannelLinkControlLocalSequenceNumber"`
						VirtualLinkStatusVector                string `json:"VirtualLinkStatusVector"`
						SChannelAssignments                    []struct {
							Svid int `json:"Svid"`
							Scid int `json:"Scid"`
						} `json:"SChannelAssignments"`
					} `json:"Transmitting"`
				} `json:"EvbData"`
				PortDiagnosticLocalLoopbackCapable bool `json:"PortDiagnosticLocalLoopbackCapable"`
				PortDiagnosticsEnabled              bool `json:"PortDiagnosticsEnabled"`
				LldpData                             struct {
					Receiving struct {
						ManagementIFNumSubtype string `json:"ManagementIFNumSubtype"`
						ChassisID               string `json:"ChassisID"`
						Dcb                     struct {
							ApplicationPriorityTable []struct {
								ApplicationProtocol int    `json:"ApplicationProtocol"`
								ApplicationSelect   string `json:"ApplicationSelect"`
								ApplicationPriority int    `json:"ApplicationPriority"`
							} `json:"ApplicationPriorityTable"`
							Ets struct {
								PriorityAssignmentTable      []int    `json:"PriorityAssignmentTable"`
								MaximumTrafficClassCount     int      `json:"MaximumTrafficClassCount"`
								BandwidthAssignmentTable     []int    `json:"BandwidthAssignmentTable"`
								TlvReceived                  bool     `json:"TlvReceived"`
								CreditBasedShaper            bool     `json:"CreditBasedShaper"`
								TrafficSelectionAlgorithmTable []string `json:"TrafficSelectionAlgorithmTable"`
								Willing                       bool     `json:"Willing"`
							} `json:"Ets"`
							Version           string `json:"Version"`
							LocalEqualRemote  bool   `json:"LocalEqualRemote"`

							ApplicationPriorityReceived bool `json:"ApplicationPriorityReceived"`
							Pfc struct {
								EnabledTrafficClasses     string `json:"EnabledTrafficClasses"`
								MaximumTrafficClassCount  int    `json:"MaximumTrafficClassCount"`
								TlvReceived               bool   `json:"TlvReceived"`
								MacSecurityBypassCapability bool `json:"MacSecurityBypassCapability"`
								Willing                   bool   `json:"Willing"`
							} `json:"Pfc"`
						} `json:"Dcb"`
						ManagementAddress       string `json:"ManagementAddress"`
						SystemDescription       string `json:"SystemDescription"`
						ManagementVlanReceived  bool   `json:"ManagementVlanReceived"`
						PortDescription         string `json:"PortDescription"`
						PortID                  string `json:"PortID"`
						ManagementAddressType   string `json:"ManagementAddressType"`
						PortIDSubtype           string `json:"PortIDSubtype"`
						ChassisIDSubtype        string `json:"ChassisIDSubtype"`
						NearestBridgeTTLExpired bool   `json:"NearestBridgeTtlExpired"`
						ManagementIFNumber      string `json:"ManagementIFNumber"`
						ManagementAddressReceived bool `json:"ManagementAddressReceived"`
					} `json:"Receiving"`
					Transmitting struct {
						ChassisIDSubtype    string `json:"ChassisIDSubtype"`
						ManagementAddresses []struct {
							ManagementIPAddressType string `json:"ManagementIPAddressType"`
							ManagementIPAddress     string `json:"ManagementIPAddress"`
						} `json:"ManagementAddresses"`
						ChassisID                 string `json:"ChassisID"`
						SystemDescription         string `json:"SystemDescription"`
						PortDescription           string `json:"PortDescription"`
						PortID                    string `json:"PortID"`
						ManagementCvlanIDTransmitted bool `json:"ManagementCvlanIDTransmitted"`
						PortIDSubtype             string `json:"PortIDSubtype"`
					} `json:"Transmitting"`
				} `json:"LldpData"`
		Ethernet struct {
			FlowControlStatus         any      `json:"FlowControlStatus"`
			AssociatedMACAddresses    []string `json:"AssociatedMACAddresses"`
			FlowControlConfiguration  string   `json:"FlowControlConfiguration"`
			SupportedEthernetCapabilities []string `json:"SupportedEthernetCapabilities"`
		} `json:"Ethernet"`
		RedfishSettings struct {
			SettingsObject struct {
				OdataID string `json:"@odata.id"`
			} `json:"SettingsObject"`
		} `json:"@Redfish.Settings"`
		ID              string `json:"Id"`
		FunctionMaxBandwidth []struct {
			AllocationPercent     int `json:"AllocationPercent"`
			NetworkDeviceFunction struct {

				OdataID string `json:"@odata.id"`
			} `json:"NetworkDeviceFunction"`
		} `json:"FunctionMaxBandwidth"`
		Settings struct {
			Href string `json:"href"`
		} `json:"Settings"`
	} `json:"Ports"`
	OdataContext string `json:"@odata.context"`
	Name         string `json:"Name"`
	OdataID      string `json:"@odata.id"`
	NetworkDeviceFunctions []struct {
		Description                 string `json:"Description"`
		OdataType                   string `json:"@odata.type"`
		AssignablePhysicalNetworkPorts []struct {
			OdataID string `json:"@odata.id"`
		} `json:"AssignablePhysicalNetworkPorts"`
		BootMode      string `json:"BootMode"`
		DeviceEnabled bool   `json:"DeviceEnabled"`
		OdataContext  string `json:"@odata.context"`
		PhysicalPortAssignment struct {
			OdataID string `json:"@odata.id"`
		} `json:"PhysicalPortAssignment"`
		Name                  string `json:"Name"`
		OdataID               string `json:"@odata.id"`
		AssignablePhysicalPorts []struct {
			OdataID string `json:"@odata.id"`
		} `json:"AssignablePhysicalPorts"`
		PhysicalNetworkPortAssignment struct {
			OdataID string `json:"@odata.id"`
		} `json:"PhysicalNetworkPortAssignment"`
		VirtualFunctionsEnabled bool   `json:"VirtualFunctionsEnabled"`
		NetDevFuncType          string `json:"NetDevFuncType"`
		Oem                     struct {
			Hpe struct {
				ConnectionID          string   `json:"ConnectionID"`
				VirtualLinkStatus     string   `json:"VirtualLinkStatus"`
				NetDevFuncType        string   `json:"NetDevFuncType"`
				OdataType             string   `json:"@odata.type"`
				WakeOnLANEnabled      bool     `json:"WakeOnLANEnabled"`
				OdataContext          string   `json:"@odata.context"`
				BootTargetSource      string   `json:"BootTargetSource"`
				NetDevFuncCapabilities []string `json:"NetDevFuncCapabilities"`
			} `json:"Hpe"`
		} `json:"Oem"`
		Ethernet struct {
			MACAddress          string `json:"MACAddress"`
			PermanentMACAddress string `json:"PermanentMACAddress"`
		} `json:"Ethernet"`
		RedfishSettings struct {

			SettingsObject struct {
				OdataID string `json:"@odata.id"`
			} `json:"SettingsObject"`
		} `json:"@Redfish.Settings"`
		Links struct {
			PhysicalNetworkPortAssignment struct {
				OdataID string `json:"@odata.id"`
			} `json:"PhysicalNetworkPortAssignment"`
			PhysicalPortAssignment struct {
				OdataID string `json:"@odata.id"`
			} `json:"PhysicalPortAssignment"`
		} `json:"Links"`
		MaxVirtualFunctions   int      `json:"MaxVirtualFunctions"`
		ID                    string   `json:"Id"`
		NetDevFuncCapabilities []string `json:"NetDevFuncCapabilities"`
		Settings              struct {
			Href string `json:"href"`
		} `json:"Settings"`
		FibreChannel struct {
			Wwpn        string `json:"WWPN"`
			Wwnn        string `json:"WWNN"`
			BootTargets []struct {
				BootPriority any `json:"BootPriority"`
				Lunid        any `json:"LUNID"`
				Wwpn         any `json:"WWPN"`
			} `json:"BootTargets"`
		} `json:"FibreChannel,omitempty"`
		ISCSIBoot struct {
			PrimaryTargetIPAddress   string `json:"PrimaryTargetIPAddress"`
			IPMaskDNSViaDHCP         bool   `json:"IPMaskDNSViaDHCP"`
			InitiatorName            string `json:"InitiatorName"`
			SecondaryTargetIPAddress string `json:"SecondaryTargetIPAddress"`
			InitiatorDefaultGateway  string `json:"InitiatorDefaultGateway"`
			AuthenticationMethod     string `json:"AuthenticationMethod"`
			TargetInfoViaDHCP        bool   `json:"TargetInfoViaDHCP"`
			MutualCHAPUsername       string `json:"MutualCHAPUsername"`
			SecondaryDNS             string `json:"SecondaryDNS"`
			CHAPSecret               any    `json:"CHAPSecret"`
			MutualCHAPSecret         any    `json:"MutualCHAPSecret"`
			InitiatorNetmask         string `json:"InitiatorNetmask"`
			PrimaryDNS               string `json:"PrimaryDNS"`
			IPAddressType            string `json:"IPAddressType"`
			CHAPUsername             string `json:"CHAPUsername"`
			PrimaryLUN               int    `json:"PrimaryLUN"`
			SecondaryTargetTCPPort   int    `json:"SecondaryTargetTCPPort"`
			InitiatorIPAddress       string `json:"InitiatorIPAddress"`
			PrimaryTargetTCPPort     int    `json:"PrimaryTargetTCPPort"`
			PrimaryTargetName        string `json:"PrimaryTargetName"`
		} `json:"iSCSIBoot,omitempty"`

	} `json:"NetworkDeviceFunctions"`
		SerialNumber string `json:"SerialNumber"`
		Oem          struct {
		Hpe struct {
			Controllers []struct {
				FunctionTypeLimits []struct {
					FCoEResourcesConsumed       int    `json:"FCoEResourcesConsumed"`
					ISCSIResourcesConsumed      int    `json:"iSCSIResourcesConsumed"`
					ConstraintDescription       string `json:"ConstraintDescription"`
					TotalSharedResourcesAvailable int  `json:"TotalSharedResourcesAvailable"`
				} `json:"FunctionTypeLimits"`
				DeviceLimitationsBitmap int `json:"DeviceLimitationsBitmap"`
				ConfigurationStatus     struct {
					Summary int `json:"Summary"`
					Detail  []struct {
						Group    int `json:"Group"`
						Summary  int `json:"Summary"`
						SubGroup int `json:"SubGroup"`
					} `json:"Detail"`
				} `json:"ConfigurationStatus"`
				MostRecentConfigurationChangeSource string `json:"MostRecentConfigurationChangeSource"`
				EdgeVirtualBridging                 struct {
					ChannelDescriptionTLVCapable bool `json:"ChannelDescriptionTLVCapable"`
					ChannelLinkControlTLVCapable bool `json:"ChannelLinkControlTLVCapable"`
				} `json:"EdgeVirtualBridging"`
				RDMASupport         []string `json:"RDMASupport"`
				EmbeddedLLDPFunctions struct {
					Optional bool `json:"Optional"`
					Enabled  bool `json:"Enabled"`
				} `json:"EmbeddedLLDPFunctions"`
				FunctionTypes      []string `json:"FunctionTypes"`
				UnderlyingDataSource string `json:"UnderlyingDataSource"`
			} `json:"Controllers"`
			CLPVersion string `json:"CLPVersion"`
			Actions    struct {
				HpeNetworkAdapterFlushConfigurationToNVM struct {
					Title  string `json:"title"`
					Target string `json:"target"`
				} `json:"#HpeNetworkAdapter.FlushConfigurationToNVM"`
				NetworkAdapterFlushConfigurationToNVM struct {
					Title  string `json:"title"`
					Target string `json:"target"`
				} `json:"#NetworkAdapter.FlushConfigurationToNVM"`
			} `json:"Actions"`
			FactoryDefaultsActuationBehavior string `json:"FactoryDefaultsActuationBehavior"`
			OdataType                        string `json:"@odata.type"`
			RedfishConfiguration             string `json:"RedfishConfiguration"`
			OdataContext                     string `json:"@odata.context"`
			PCAVersion                       string `json:"PCAVersion"`

		} `json:"Hpe"`
	} `json:"Oem"`
		RedfishSettings struct {
		SettingsObject struct {
			OdataID string `json:"@odata.id"`
		} `json:"SettingsObject"`
				} `json:"@Redfish.Settings"`
				Manufacturer string `json:"Manufacturer"`
				Model        string `json:"Model"`
				ID           string `json:"Id"`
				Sku          string `json:"SKU"`
				Settings     struct {
					Href string `json:"href"`
				} `json:"Settings"`
			} `json:"data"`
			Etag  string `json:"etag"`
			Count int    `json:"count"`
			Name  string `json:"name"`
		} `json:"NetworkAdapters"`
		Processors struct {
	URI             string    `json:"uri"`
	CollectionState string    `json:"collectionState"`
	Modified        time.Time `json:"modified"`
	Data            []struct {
		Status struct {
			Health string `json:"Health"`
			State  string `json:"State"`
		} `json:"Status"`
		PartNumber            string `json:"PartNumber"`
		TotalThreads          int    `json:"TotalThreads"`
		ProcessorArchitecture string `json:"ProcessorArchitecture"`
		ProcessorID           struct {
			IdentificationRegisters string `json:"IdentificationRegisters"`
			VendorID                string `json:"VendorId"`
			MicrocodeInfo           any    `json:"MicrocodeInfo"`
			Step                    string `json:"Step"`
			EffectiveModel          string `json:"EffectiveModel"`
			EffectiveFamily         string `json:"EffectiveFamily"`
		} `json:"ProcessorId"`
		ProcessorType string `json:"ProcessorType"`
		OdataType     string `json:"@odata.type"`
		TotalCores    int    `json:"TotalCores"`
		OdataContext  string `json:"@odata.context"`
		InstructionSet string `json:"InstructionSet"`
		Name          string `json:"Name"`
		OdataID       string `json:"@odata.id"`
		SerialNumber  string `json:"SerialNumber"`
		Oem           struct {
			Hpe struct {

				MicrocodePatches []struct {
					CPUID   string    `json:"CpuId"`
					PatchID string    `json:"PatchId"`
					Date    time.Time `json:"Date"`
				} `json:"MicrocodePatches"`
				PartNumber       string `json:"PartNumber"`
				OdataType        string `json:"@odata.type"`
				CoresEnabled     int    `json:"CoresEnabled"`
				OdataContext     string `json:"@odata.context"`
				RatedSpeedMHz    int    `json:"RatedSpeedMHz"`
				AssetTag         string `json:"AssetTag"`
				ExternalClockMHz int    `json:"ExternalClockMHz"`
				SerialNumber     string `json:"SerialNumber"`
				Characteristics  []string `json:"Characteristics"`
				VoltageVoltsX10  int    `json:"VoltageVoltsX10"`
				ConfigStatus     struct {
					State     string `json:"State"`
					Populated bool   `json:"Populated"`
				} `json:"ConfigStatus"`
				Cache []struct {
					Policy            string   `json:"Policy"`
					InstalledSizeKB   int      `json:"InstalledSizeKB"`
					CurrentSRAMType   []string `json:"CurrentSRAMType"`
					SupportedSRAMType []string `json:"SupportedSRAMType"`
					EccType           string   `json:"EccType"`
					SystemCacheType   string   `json:"SystemCacheType"`
					CacheSpeedns      int      `json:"CacheSpeedns"`
					MaximumSizeKB     int      `json:"MaximumSizeKB"`
					Socketed          bool     `json:"Socketed"`
					Associativity     string   `json:"Associativity"`
					Location          string   `json:"Location"`
					Name              string   `json:"Name"`
				} `json:"Cache"`
			} `json:"Hpe"`
		} `json:"Oem"`
		Manufacturer string `json:"Manufacturer"`
		MaxSpeedMHz  int    `json:"MaxSpeedMHz"`
		Model        string `json:"Model"`
		ID           string `json:"Id"`
		Socket       string `json:"Socket"`
	} `json:"data"`
	Etag  string `json:"etag"`
	Count int    `json:"count"`
	Name  string `json:"name"`
} `json:"Processors"`
} `json:"subResources,omitempty"`
SupportDataCollectionState any    `json:"supportDataCollectionState"`
SupportDataCollectionType  any    `json:"supportDataCollectionType"`
SupportState               string `json:"supportState"`

UIDState               string    `json:"uidState"`
URI                    string    `json:"uri"`
UUID                   string    `json:"uuid"`
VirtualSerialNumber    string    `json:"virtualSerialNumber"`
VirtualUUID            string    `json:"virtualUuid"`
MpDNSName              string    `json:"mpDnsName"`
MpIPAddress            string    `json:"mpIpAddress"`
ID                     string    `json:"id"`
OrderedStatus          int       `json:"_orderedStatus"`
OriginalURI            string    `json:"originalUri"`
Appluri                string    `json:"appluri"`
ApplianceName          string    `json:"applianceName"`
ApplianceLocation      string    `json:"applianceLocation"`
MpFirmwareVersionNumber int64    `json:"mpFirmwareVersionNumber"`
RomVersionNumber       float64   `json:"romVersionNumber"`
MpIPAddresses          string    `json:"_mpIpAddresses"`
ServerHardwareTypeName string    `json:"serverHardwareTypeName"`
ServerProfileName      string    `json:"serverProfileName,omitempty"`
SubResources0          struct {
	LocalStorage struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			AdapterType             string `json:"AdapterType"`
			BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
			CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
			CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
			CacheModuleStatus       struct {
				Health string `json:"Health"`
			} `json:"CacheModuleStatus"`
			ControllerBoard struct {
				Status struct {
					Health string `json:"Health"`
				} `json:"Status"`
			} `json:"ControllerBoard"`
			CurrentOperatingMode              string `json:"CurrentOperatingMode"`
			DriveWriteCache                   string `json:"DriveWriteCache"`
			EncryptionCryptoOfficerPasswordSet bool  `json:"EncryptionCryptoOfficerPasswordSet"`
			EncryptionCspTestPassed            bool  `json:"EncryptionCspTestPassed"`
			EncryptionEnabled                  bool  `json:"EncryptionEnabled"`
			EncryptionFwLocked                 bool  `json:"EncryptionFwLocked"`
			EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
			EncryptionMixedVolumesEnabled      bool  `json:"EncryptionMixedVolumesEnabled"`
			EncryptionSelfTestPassed           bool  `json:"EncryptionSelfTestPassed"`
			EncryptionStandaloneModeEnabled    bool  `json:"EncryptionStandaloneModeEnabled"`
			ExternalPortCount                  int   `json:"ExternalPortCount"`
			FirmwareVersion                    struct {
				Current struct {

					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			InternalPortCount int    `json:"InternalPortCount"`
			Location          string `json:"Location"`
			LocationFormat    string `json:"LocationFormat"`
			LogicalDrives     []struct {
				AccelerationMethod string `json:"AccelerationMethod"`
				CapacityMiB        int    `json:"CapacityMiB"`
				DataDrives         []struct {
					BlockSizeBytes         int      `json:"BlockSizeBytes"`
					CapacityLogicalBlocks  int      `json:"CapacityLogicalBlocks"`
					CapacityMiB            int      `json:"CapacityMiB"`
					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					DiskDriveUse           string   `json:"DiskDriveUse"`
					EncryptedDrive         bool     `json:"EncryptedDrive"`
					FirmwareVersion        struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceSpeedMbps  int    `json:"InterfaceSpeedMbps"`
					InterfaceType       string `json:"InterfaceType"`
					LegacyBootPriority  string `json:"LegacyBootPriority"`
					Location            string `json:"Location"`
					LocationFormat      string `json:"LocationFormat"`
					MediaType           string `json:"MediaType"`
					Model               string `json:"Model"`
					RotationalSpeedRpm  int    `json:"RotationalSpeedRpm"`
					SerialNumber        string `json:"SerialNumber"`
					Status              struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
				} `json:"DataDrives"`
				InterfaceType             string   `json:"InterfaceType"`
				LegacyBootPriority        string   `json:"LegacyBootPriority"`
				LogicalDriveEncryption    bool     `json:"LogicalDriveEncryption"`
				LogicalDriveName          string   `json:"LogicalDriveName"`
				LogicalDriveNumber        int      `json:"LogicalDriveNumber"`
				LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
				MediaType                 string   `json:"MediaType"`
				Raid                      string   `json:"Raid"`
				Status                    struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				StripeSizeBytes        int    `json:"StripeSizeBytes"`
				VolumeUniqueIdentifier string `json:"VolumeUniqueIdentifier"`

			} `json:"LogicalDrives"`
			Model           string `json:"Model"`
			Name            string `json:"Name"`
			PhysicalDrives  []struct {
				BlockSizeBytes         int      `json:"BlockSizeBytes"`
				CapacityLogicalBlocks  int      `json:"CapacityLogicalBlocks"`
				CapacityMiB            int      `json:"CapacityMiB"`
				DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
				DiskDriveUse           string   `json:"DiskDriveUse"`
				EncryptedDrive         bool     `json:"EncryptedDrive"`
				FirmwareVersion        struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				InterfaceSpeedMbps int    `json:"InterfaceSpeedMbps"`
				InterfaceType      string `json:"InterfaceType"`
				LegacyBootPriority string `json:"LegacyBootPriority"`
				Location           string `json:"Location"`
				LocationFormat     string `json:"LocationFormat"`
				MediaType          string `json:"MediaType"`
				Model              string `json:"Model"`
				RotationalSpeedRpm int    `json:"RotationalSpeedRpm"`
				SerialNumber       string `json:"SerialNumber"`
				Status             struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
			} `json:"PhysicalDrives"`
			SerialNumber      string `json:"SerialNumber"`
			Status            struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			StorageEnclosures []struct {
				DriveBayCount   int    `json:"DriveBayCount"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				ID             string `json:"Id"`
				Location       string `json:"Location"`
				LocationFormat string `json:"LocationFormat"`
				Status         struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`

			} `json:"StorageEnclosures"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"LocalStorage"`
	Memory struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			OdataEtag         string `json:"@odata.etag"`
			OdataType         string `json:"@odata.type"`
			ID                string `json:"Id"`
			BusWidthBits      int    `json:"BusWidthBits"`
			CacheSizeMiB      int    `json:"CacheSizeMiB"`
			CapacityMiB       int    `json:"CapacityMiB"`
			DataWidthBits     int    `json:"DataWidthBits"`
			DeviceLocator     string `json:"DeviceLocator"`
			ErrorCorrection   string `json:"ErrorCorrection"`
			LogicalSizeMiB    int    `json:"LogicalSizeMiB"`
			MemoryLocation    struct {
				Channel          int `json:"Channel"`
				MemoryController int `json:"MemoryController"`
				Slot             int `json:"Slot"`
				Socket           int `json:"Socket"`
			} `json:"MemoryLocation"`
			MemoryMedia        []string `json:"MemoryMedia"`
			MemoryType         string   `json:"MemoryType"`
			Name               string   `json:"Name"`
			NonVolatileSizeMiB int      `json:"NonVolatileSizeMiB"`
			Oem                struct {
				Hpe struct {
					OdataType              string `json:"@odata.type"`
					DIMMStatus             string `json:"DIMMStatus"`
					MinimumVoltageVoltsX10 int    `json:"MinimumVoltageVoltsX10"`
				} `json:"Hpe"`
			} `json:"Oem"`
			OperatingMemoryModes        []string `json:"OperatingMemoryModes"`
			PersistentRegionSizeLimitMiB int     `json:"PersistentRegionSizeLimitMiB"`
			RankCount                   any      `json:"RankCount"`
			SecurityCapabilities        struct {
			} `json:"SecurityCapabilities"`
			Status struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			VendorID                   string `json:"VendorID"`
			VolatileRegionSizeLimitMiB int    `json:"VolatileRegionSizeLimitMiB"`

			VolatileSizeMiB  int    `json:"VolatileSizeMiB"`
			BaseModuleType   string `json:"BaseModuleType,omitempty"`
			Manufacturer     string `json:"Manufacturer,omitempty"`
			MemoryDeviceType string `json:"MemoryDeviceType,omitempty"`
			OperatingSpeedMhz int   `json:"OperatingSpeedMhz,omitempty"`
			PartNumber       string `json:"PartNumber,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Memory"`
	MemoryList struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			BoardCPUNumber            int `json:"BoardCpuNumber"`
			BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
			BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
			BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
			BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"MemoryList"`
	AdvancedMemoryProtection struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			OdataType        string   `json:"@odata.type"`
			AmpModeActive    string   `json:"AmpModeActive"`
			AmpModeStatus    string   `json:"AmpModeStatus"`
			AmpModeSupported []string `json:"AmpModeSupported"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"AdvancedMemoryProtection"`
	LocalStorageV2 struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []any     `json:"data"`
		Etag            string    `json:"etag"`
		Count           int       `json:"count"`
		Name            string    `json:"name"`
	} `json:"LocalStorageV2"`

	Devices struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			OdataEtag       string `json:"@odata.etag"`
			OdataType       string `json:"@odata.type"`
			ID              string `json:"Id"`
			DeviceType      string `json:"DeviceType"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			Location          string `json:"Location"`
			Manufacturer      string `json:"Manufacturer"`
			Name              string `json:"Name"`
			PartNumber        string `json:"PartNumber"`
			ProductPartNumber string `json:"ProductPartNumber"`
			ProductVersion    string `json:"ProductVersion"`
			SerialNumber      string `json:"SerialNumber"`
			Status            struct {
				Health any `json:"Health"`
				State  any `json:"State"`
			} `json:"Status,omitempty"`
			MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
			Status0              struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Devices"`
} `json:"subResources,omitempty"`
SubResources1 struct {
	LocalStorage struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
			CurrentOperatingMode            string `json:"CurrentOperatingMode"`
			LocationFormat                  string `json:"LocationFormat"`
			EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
			AdapterType                     string `json:"AdapterType"`
			PhysicalDrives                  []struct {
				Status struct {
					Health string `json:"Health"`

					State string `json:"State"`
				} `json:"Status"`
				LocationFormat        string `json:"LocationFormat"`
				CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
				PowerOnHours          int    `json:"PowerOnHours"`
				BlockSizeBytes        int    `json:"BlockSizeBytes"`
				SSDEnduranceUtilizationPercentage int `json:"SSDEnduranceUtilizationPercentage"`
				CapacityMiB           int    `json:"CapacityMiB"`
				FirmwareVersion       struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				InterfaceType          string   `json:"InterfaceType"`
				LegacyBootPriority     string   `json:"LegacyBootPriority"`
				SerialNumber           string   `json:"SerialNumber"`
				DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
				Model                  string   `json:"Model"`
				MediaType              string   `json:"MediaType"`
				DiskDriveUse           string   `json:"DiskDriveUse"`
				EncryptedDrive         bool     `json:"EncryptedDrive"`
				InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
				Location               string   `json:"Location"`
			} `json:"PhysicalDrives"`
			LogicalDrives []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LogicalDriveEncryption bool `json:"LogicalDriveEncryption"`
				DataDrives             []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat        string `json:"LocationFormat"`
					CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
					PowerOnHours          int    `json:"PowerOnHours"`
					BlockSizeBytes        int    `json:"BlockSizeBytes"`
					SSDEnduranceUtilizationPercentage int `json:"SSDEnduranceUtilizationPercentage"`
					CapacityMiB           int    `json:"CapacityMiB"`
					FirmwareVersion       struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceType      string `json:"InterfaceType"`
					LegacyBootPriority string `json:"LegacyBootPriority"`
					SerialNumber       string `json:"SerialNumber"`

					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					Model                  string   `json:"Model"`
					MediaType              string   `json:"MediaType"`
					DiskDriveUse           string   `json:"DiskDriveUse"`
					EncryptedDrive         bool     `json:"EncryptedDrive"`
					InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
					Location               string   `json:"Location"`
				} `json:"DataDrives"`
				StripeSizeBytes           int      `json:"StripeSizeBytes"`
				Raid                      string   `json:"Raid"`
				CapacityMiB               int      `json:"CapacityMiB"`
				LogicalDriveType          string   `json:"LogicalDriveType"`
				LogicalDriveNumber        int      `json:"LogicalDriveNumber"`
				AccelerationMethod        string   `json:"AccelerationMethod"`
				InterfaceType             string   `json:"InterfaceType"`
				LegacyBootPriority        string   `json:"LegacyBootPriority"`
				LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
				MediaType                 string   `json:"MediaType"`
				VolumeUniqueIdentifier    string   `json:"VolumeUniqueIdentifier"`
				LogicalDriveName          string   `json:"LogicalDriveName"`
			} `json:"LogicalDrives"`
			InternalPortCount       int    `json:"InternalPortCount"`
			CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
			EncryptionFwLocked      bool   `json:"EncryptionFwLocked"`
			Name                    string `json:"Name"`
			BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
			DriveWriteCache         string `json:"DriveWriteCache"`
			Status                  struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			EncryptionSelfTestPassed bool `json:"EncryptionSelfTestPassed"`
			ExternalPortCount        int  `json:"ExternalPortCount"`
			ControllerBoard          struct {
				Status struct {
					Health string `json:"Health"`
				} `json:"Status"`
			} `json:"ControllerBoard"`
			EncryptionEnabled bool `json:"EncryptionEnabled"`
			EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
			EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
			SerialNumber                       string `json:"SerialNumber"`
			StorageEnclosures                  []struct {

				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LocationFormat  string `json:"LocationFormat"`
				DriveBayCount   int    `json:"DriveBayCount"`
				ID              string `json:"Id"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				Location string `json:"Location"`
			} `json:"StorageEnclosures"`
			CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
			Model                   string `json:"Model"`
			Location                string `json:"Location"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"LocalStorage"`
	Memory struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			PersistentRegionSizeLimitMiB int    `json:"PersistentRegionSizeLimitMiB"`
			OdataType                    string `json:"@odata.type"`
			LogicalSizeMiB               int    `json:"LogicalSizeMiB"`
			RankCount                    any    `json:"RankCount"`
			VendorID                     string `json:"VendorID"`
			OdataEtag                    string `json:"@odata.etag"`
			SecurityCapabilities         struct {
			} `json:"SecurityCapabilities"`
			BusWidthBits       int    `json:"BusWidthBits"`
			CacheSizeMiB       int    `json:"CacheSizeMiB"`
			DataWidthBits      int    `json:"DataWidthBits"`
			CapacityMiB        int    `json:"CapacityMiB"`
			VolatileSizeMiB    int    `json:"VolatileSizeMiB"`
			Name               string `json:"Name"`
			NonVolatileSizeMiB int    `json:"NonVolatileSizeMiB"`
			ErrorCorrection    string `json:"ErrorCorrection"`
			MemoryMedia        []string `json:"MemoryMedia"`
			Oem                struct {

				Hpe struct {
					DIMMStatus             string `json:"DIMMStatus"`
					OdataType              string `json:"@odata.type"`
					MinimumVoltageVoltsX10 int    `json:"MinimumVoltageVoltsX10"`
				} `json:"Hpe"`
			} `json:"Oem"`
			OperatingMemoryModes       []string `json:"OperatingMemoryModes"`
			VolatileRegionSizeLimitMiB int     `json:"VolatileRegionSizeLimitMiB"`
			ID                         string  `json:"Id"`
			MemoryLocation             struct {
				Channel          int `json:"Channel"`
				Slot             int `json:"Slot"`
				MemoryController int `json:"MemoryController"`
				Socket           int `json:"Socket"`
			} `json:"MemoryLocation"`
			DeviceLocator     string `json:"DeviceLocator"`
			MemoryType        string `json:"MemoryType"`
			PartNumber        string `json:"PartNumber,omitempty"`
			OperatingSpeedMhz int    `json:"OperatingSpeedMhz,omitempty"`
			BaseModuleType    string `json:"BaseModuleType,omitempty"`
			Manufacturer      string `json:"Manufacturer,omitempty"`
			MemoryDeviceType  string `json:"MemoryDeviceType,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Memory"`
	MemoryList struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
			BoardCPUNumber            int `json:"BoardCpuNumber"`
			BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
			BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
			BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"MemoryList"`
	AdvancedMemoryProtection struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			AmpModeActive string `json:"AmpModeActive"`
			OdataType     string `json:"@odata.type"`

			AmpModeSupported []string `json:"AmpModeSupported"`
			AmpModeStatus    string   `json:"AmpModeStatus"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"AdvancedMemoryProtection"`
	LocalStorageV2 struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []any     `json:"data"`
		Etag            string    `json:"etag"`
		Count           int       `json:"count"`
		Name            string    `json:"name"`
	} `json:"LocalStorageV2"`
	Devices struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status,omitempty"`
			PartNumber      string `json:"PartNumber"`
			OdataType       string `json:"@odata.type"`
			OdataEtag       string `json:"@odata.etag"`
			ProductVersion  string `json:"ProductVersion"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			Name              string `json:"Name"`
			DeviceType        string `json:"DeviceType"`
			SerialNumber      string `json:"SerialNumber"`
			ProductPartNumber string `json:"ProductPartNumber"`
			Manufacturer      string `json:"Manufacturer"`
			ID                string `json:"Id"`
			Location          string `json:"Location"`
			Status0           struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
			MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
			Status1              struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
		} `json:"data"`

		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Devices"`
} `json:"subResources,omitempty"`
SubResources2 struct {
	LocalStorage struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
			CurrentOperatingMode            string `json:"CurrentOperatingMode"`
			LocationFormat                  string `json:"LocationFormat"`
			EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
			AdapterType                     string `json:"AdapterType"`
			PhysicalDrives                  []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LocationFormat        string `json:"LocationFormat"`
				CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
				BlockSizeBytes        int    `json:"BlockSizeBytes"`
				CapacityMiB           int    `json:"CapacityMiB"`
				FirmwareVersion       struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				InterfaceType          string `json:"InterfaceType"`
				LegacyBootPriority     string `json:"LegacyBootPriority"`
				SerialNumber           string `json:"SerialNumber"`
				DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
				Model                  string `json:"Model"`
				MediaType              string `json:"MediaType"`
				RotationalSpeedRpm     int    `json:"RotationalSpeedRpm"`
				DiskDriveUse           string `json:"DiskDriveUse"`
				EncryptedDrive         bool   `json:"EncryptedDrive"`
				InterfaceSpeedMbps     int    `json:"InterfaceSpeedMbps"`
				Location               string `json:"Location"`
			} `json:"PhysicalDrives"`
			LogicalDrives []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LogicalDriveEncryption bool `json:"LogicalDriveEncryption"`
				DataDrives             []struct {

					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat        string `json:"LocationFormat"`
					CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
					BlockSizeBytes        int    `json:"BlockSizeBytes"`
					CapacityMiB           int    `json:"CapacityMiB"`
					FirmwareVersion       struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceType          string   `json:"InterfaceType"`
					LegacyBootPriority     string   `json:"LegacyBootPriority"`
					SerialNumber           string   `json:"SerialNumber"`
					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					Model                  string   `json:"Model"`
					MediaType              string   `json:"MediaType"`
					RotationalSpeedRpm     int      `json:"RotationalSpeedRpm"`
					DiskDriveUse           string   `json:"DiskDriveUse"`
					EncryptedDrive         bool     `json:"EncryptedDrive"`
					InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
					Location               string   `json:"Location"`
				} `json:"DataDrives"`
				StripeSizeBytes           int      `json:"StripeSizeBytes"`
				Raid                      string   `json:"Raid"`
				CapacityMiB               int      `json:"CapacityMiB"`
				LogicalDriveType          string   `json:"LogicalDriveType"`
				LogicalDriveNumber        int      `json:"LogicalDriveNumber"`
				AccelerationMethod        string   `json:"AccelerationMethod"`
				InterfaceType             string   `json:"InterfaceType"`
				LegacyBootPriority        string   `json:"LegacyBootPriority"`
				LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
				MediaType                 string   `json:"MediaType"`
				VolumeUniqueIdentifier    string   `json:"VolumeUniqueIdentifier"`
				LogicalDriveName          string   `json:"LogicalDriveName"`
			} `json:"LogicalDrives"`
			InternalPortCount       int    `json:"InternalPortCount"`
			CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
			EncryptionFwLocked      bool   `json:"EncryptionFwLocked"`
			Name                    string `json:"Name"`
			BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
			DriveWriteCache         string `json:"DriveWriteCache"`
			Status                  struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			EncryptionSelfTestPassed bool `json:"EncryptionSelfTestPassed"`

			ExternalPortCount        int    `json:"ExternalPortCount"`
			ControllerBoard          struct {
				Status struct {
					Health string `json:"Health"`
				} `json:"Status"`
			} `json:"ControllerBoard"`
			EncryptionEnabled bool `json:"EncryptionEnabled"`
			EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
			EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
			SerialNumber                       string `json:"SerialNumber"`
			CacheModuleStatus                  struct {
				Health string `json:"Health"`
			} `json:"CacheModuleStatus"`
			StorageEnclosures []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LocationFormat  string `json:"LocationFormat"`
				DriveBayCount   int    `json:"DriveBayCount"`
				ID              string `json:"Id"`
				FirmwareVersion struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				Location string `json:"Location"`
			} `json:"StorageEnclosures"`
			CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
			Model                   string `json:"Model"`
			Location                string `json:"Location"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"LocalStorage"`
	Memory struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				Health string `json:"Health"`

				State string `json:"State"`
			} `json:"Status"`
			PersistentRegionSizeLimitMiB int    `json:"PersistentRegionSizeLimitMiB"`
			OdataType                    string `json:"@odata.type"`
			LogicalSizeMiB               int    `json:"LogicalSizeMiB"`
			RankCount                    any    `json:"RankCount"`
			VendorID                     string `json:"VendorID"`
			OdataEtag                    string `json:"@odata.etag"`
			SecurityCapabilities         struct {
			} `json:"SecurityCapabilities"`
			BusWidthBits       int    `json:"BusWidthBits"`
			CacheSizeMiB       int    `json:"CacheSizeMiB"`
			DataWidthBits      int    `json:"DataWidthBits"`
			CapacityMiB        int    `json:"CapacityMiB"`
			VolatileSizeMiB    int    `json:"VolatileSizeMiB"`
			Name               string `json:"Name"`
			NonVolatileSizeMiB int    `json:"NonVolatileSizeMiB"`
			ErrorCorrection    string `json:"ErrorCorrection"`
			MemoryMedia        []string `json:"MemoryMedia"`
			Oem                struct {
				Hpe struct {
					DIMMStatus             string `json:"DIMMStatus"`
					OdataType              string `json:"@odata.type"`
					MinimumVoltageVoltsX10 int    `json:"MinimumVoltageVoltsX10"`
				} `json:"Hpe"`
			} `json:"Oem"`
			OperatingMemoryModes       []string `json:"OperatingMemoryModes"`
			VolatileRegionSizeLimitMiB int     `json:"VolatileRegionSizeLimitMiB"`
			ID                         string  `json:"Id"`
			MemoryLocation             struct {
				Channel          int `json:"Channel"`
				Slot             int `json:"Slot"`
				MemoryController int `json:"MemoryController"`
				Socket           int `json:"Socket"`
			} `json:"MemoryLocation"`
			DeviceLocator     string `json:"DeviceLocator"`
			MemoryType        string `json:"MemoryType"`
			PartNumber        string `json:"PartNumber,omitempty"`
			OperatingSpeedMhz int    `json:"OperatingSpeedMhz,omitempty"`
			BaseModuleType    string `json:"BaseModuleType,omitempty"`
			Manufacturer      string `json:"Manufacturer,omitempty"`
			MemoryDeviceType  string `json:"MemoryDeviceType,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Memory"`
	MemoryList struct {
		URI             string    `json:"uri"`

		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
			BoardCPUNumber            int `json:"BoardCpuNumber"`
			BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
			BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
			BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"MemoryList"`
	AdvancedMemoryProtection struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			AmpModeActive    string   `json:"AmpModeActive"`
			OdataType        string   `json:"@odata.type"`
			AmpModeSupported []string `json:"AmpModeSupported"`
			AmpModeStatus    string   `json:"AmpModeStatus"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"AdvancedMemoryProtection"`
	LocalStorageV2 struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []any     `json:"data"`
		Etag            string    `json:"etag"`
		Count           int       `json:"count"`
		Name            string    `json:"name"`
	} `json:"LocalStorageV2"`
	Devices struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status,omitempty"`
			PartNumber     string `json:"PartNumber"`
			OdataType      string `json:"@odata.type"`
			OdataEtag      string `json:"@odata.etag"`
			ProductVersion string `json:"ProductVersion"`

			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			Name              string `json:"Name"`
			DeviceType        string `json:"DeviceType"`
			SerialNumber      string `json:"SerialNumber"`
			ProductPartNumber string `json:"ProductPartNumber"`
			Manufacturer      string `json:"Manufacturer"`
			ID                string `json:"Id"`
			Location          string `json:"Location"`
			Status0           struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
			MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
			Status1              struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Devices"`
} `json:"subResources,omitempty"`
SubResources3 struct {
	LocalStorage struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			EncryptionMixedVolumesEnabled   bool   `json:"EncryptionMixedVolumesEnabled"`
			CurrentOperatingMode            string `json:"CurrentOperatingMode"`
			LocationFormat                  string `json:"LocationFormat"`
			EncryptionStandaloneModeEnabled bool   `json:"EncryptionStandaloneModeEnabled"`
			AdapterType                     string `json:"AdapterType"`
			PhysicalDrives                  []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LocationFormat        string `json:"LocationFormat"`
				CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
				BlockSizeBytes        int    `json:"BlockSizeBytes"`
				CapacityMiB           int    `json:"CapacityMiB"`
				FirmwareVersion       struct {
					Current struct {
						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`

				InterfaceType          string   `json:"InterfaceType"`
				LegacyBootPriority     string   `json:"LegacyBootPriority"`
				SerialNumber           string   `json:"SerialNumber"`
				DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
				Model                  string   `json:"Model"`
				MediaType              string   `json:"MediaType"`
				RotationalSpeedRpm     int      `json:"RotationalSpeedRpm"`
				DiskDriveUse           string   `json:"DiskDriveUse"`
				EncryptedDrive         bool     `json:"EncryptedDrive"`
				InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
				Location               string   `json:"Location"`
			} `json:"PhysicalDrives"`
			LogicalDrives []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LogicalDriveEncryption bool `json:"LogicalDriveEncryption"`
				DataDrives             []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					LocationFormat        string `json:"LocationFormat"`
					CapacityLogicalBlocks int    `json:"CapacityLogicalBlocks"`
					BlockSizeBytes        int    `json:"BlockSizeBytes"`
					CapacityMiB           int    `json:"CapacityMiB"`
					FirmwareVersion       struct {
						Current struct {
							VersionString string `json:"VersionString"`
						} `json:"Current"`
					} `json:"FirmwareVersion"`
					InterfaceType          string   `json:"InterfaceType"`
					LegacyBootPriority     string   `json:"LegacyBootPriority"`
					SerialNumber           string   `json:"SerialNumber"`
					DiskDriveStatusReasons []string `json:"DiskDriveStatusReasons"`
					Model                  string   `json:"Model"`
					MediaType              string   `json:"MediaType"`
					RotationalSpeedRpm     int      `json:"RotationalSpeedRpm"`
					DiskDriveUse           string   `json:"DiskDriveUse"`
					EncryptedDrive         bool     `json:"EncryptedDrive"`
					InterfaceSpeedMbps     int      `json:"InterfaceSpeedMbps"`
					Location               string   `json:"Location"`
				} `json:"DataDrives"`
				StripeSizeBytes    int    `json:"StripeSizeBytes"`
				Raid               string `json:"Raid"`
				CapacityMiB        int    `json:"CapacityMiB"`
				LogicalDriveType   string `json:"LogicalDriveType"`

				LogicalDriveNumber        int      `json:"LogicalDriveNumber"`
				AccelerationMethod        string   `json:"AccelerationMethod"`
				InterfaceType             string   `json:"InterfaceType"`
				LegacyBootPriority        string   `json:"LegacyBootPriority"`
				LogicalDriveStatusReasons []string `json:"LogicalDriveStatusReasons"`
				MediaType                 string   `json:"MediaType"`
				VolumeUniqueIdentifier    string   `json:"VolumeUniqueIdentifier"`
				LogicalDriveName          string   `json:"LogicalDriveName"`
			} `json:"LogicalDrives"`
			InternalPortCount       int    `json:"InternalPortCount"`
			CacheMemorySizeMiB      int    `json:"CacheMemorySizeMiB"`
			EncryptionFwLocked      bool   `json:"EncryptionFwLocked"`
			Name                    string `json:"Name"`
			BackupPowerSourceStatus string `json:"BackupPowerSourceStatus"`
			DriveWriteCache         string `json:"DriveWriteCache"`
			Status                  struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			EncryptionSelfTestPassed bool `json:"EncryptionSelfTestPassed"`
			ExternalPortCount        int  `json:"ExternalPortCount"`
			ControllerBoard          struct {
				Status struct {
					Health string `json:"Health"`
				} `json:"Status"`
			} `json:"ControllerBoard"`
			EncryptionEnabled bool `json:"EncryptionEnabled"`
			EncryptionHasLockedVolumesMissingBootPassword bool `json:"EncryptionHasLockedVolumesMissingBootPassword"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			EncryptionCryptoOfficerPasswordSet bool   `json:"EncryptionCryptoOfficerPasswordSet"`
			EncryptionCspTestPassed            bool   `json:"EncryptionCspTestPassed"`
			SerialNumber                       string `json:"SerialNumber"`
			CacheModuleStatus                  struct {
				Health string `json:"Health"`
			} `json:"CacheModuleStatus"`
			StorageEnclosures []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				LocationFormat string `json:"LocationFormat"`
				DriveBayCount  int    `json:"DriveBayCount"`
				ID             string `json:"Id"`
				FirmwareVersion struct {
					Current struct {

						VersionString string `json:"VersionString"`
					} `json:"Current"`
				} `json:"FirmwareVersion"`
				Location string `json:"Location"`
			} `json:"StorageEnclosures"`
			CacheModuleSerialNumber string `json:"CacheModuleSerialNumber"`
			Model                   string `json:"Model"`
			Location                string `json:"Location"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"LocalStorage"`
	Memory struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			PartNumber                    string `json:"PartNumber"`
			PersistentRegionSizeLimitMiB int     `json:"PersistentRegionSizeLimitMiB"`
			OdataType                    string `json:"@odata.type"`
			LogicalSizeMiB               int    `json:"LogicalSizeMiB"`
			RankCount                    any    `json:"RankCount"`
			SecurityCapabilities         struct {
			} `json:"SecurityCapabilities"`
			BusWidthBits         int      `json:"BusWidthBits"`
			CacheSizeMiB         int      `json:"CacheSizeMiB"`
			Name                 string   `json:"Name"`
			NonVolatileSizeMiB   int      `json:"NonVolatileSizeMiB"`
			OperatingMemoryModes []string `json:"OperatingMemoryModes"`
			MemoryLocation       struct {
				Channel          int `json:"Channel"`
				Slot             int `json:"Slot"`
				MemoryController int `json:"MemoryController"`
				Socket           int `json:"Socket"`
			} `json:"MemoryLocation"`
			DeviceLocator string `json:"DeviceLocator"`
			MemoryType    string `json:"MemoryType"`
			Status        struct {
				Health string `json:"Health"`
				State  string `json:"State"`
			} `json:"Status"`
			VendorID        string `json:"VendorID"`
			OdataEtag       string `json:"@odata.etag"`
			DataWidthBits   int    `json:"DataWidthBits"`
			CapacityMiB     int    `json:"CapacityMiB"`
			VolatileSizeMiB int    `json:"VolatileSizeMiB"`
			ErrorCorrection string `json:"ErrorCorrection"`
			SerialNumber    string `json:"SerialNumber"`

			MemoryMedia []string `json:"MemoryMedia"`
			Oem         struct {
				Hpe struct {
					DIMMStatus                string `json:"DIMMStatus"`
					OdataType                 string `json:"@odata.type"`
					DIMMManufacturingDate     string `json:"DIMMManufacturingDate"`
					MinimumVoltageVoltsX10    int    `json:"MinimumVoltageVoltsX10"`
				} `json:"Hpe"`
			} `json:"Oem"`
			VolatileRegionSizeLimitMiB int    `json:"VolatileRegionSizeLimitMiB"`
			ID                         string `json:"Id"`
			OperatingSpeedMhz          int    `json:"OperatingSpeedMhz,omitempty"`
			BaseModuleType             string `json:"BaseModuleType,omitempty"`
			Manufacturer               string `json:"Manufacturer,omitempty"`
			MemoryDeviceType           string `json:"MemoryDeviceType,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Memory"`
	MemoryList struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			BoardNumberOfSockets      int `json:"BoardNumberOfSockets"`
			BoardCPUNumber            int `json:"BoardCpuNumber"`
			BoardOperationalVoltage   int `json:"BoardOperationalVoltage"`
			BoardTotalMemorySize      int `json:"BoardTotalMemorySize"`
			BoardOperationalFrequency int `json:"BoardOperationalFrequency"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"MemoryList"`
	AdvancedMemoryProtection struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			AmpModeActive    string   `json:"AmpModeActive"`
			OdataType        string   `json:"@odata.type"`
			AmpModeSupported []string `json:"AmpModeSupported"`
			AmpModeStatus    string   `json:"AmpModeStatus"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"AdvancedMemoryProtection"`

	LocalStorageV2 struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				HealthRollup string `json:"HealthRollup"`
				Health       string `json:"Health"`
				State        string `json:"State"`
			} `json:"Status"`
			Controllers []struct {
				SpeedGbps int `json:"SpeedGbps"`
				Status    struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				PCIeInterface struct {
					PCIeType    string `json:"PCIeType"`
					MaxPCIeType string `json:"MaxPCIeType"`
					MaxLanes    int    `json:"MaxLanes"`
					LanesInUse  int    `json:"LanesInUse"`
				} `json:"PCIeInterface"`
				PartNumber      string `json:"PartNumber"`
				OdataType       string `json:"@odata.type"`
				ControllerRates struct {
					ConsistencyCheckRatePercent int `json:"ConsistencyCheckRatePercent"`
					RebuildRatePercent          int `json:"RebuildRatePercent"`
					TransformationRatePercent   int `json:"TransformationRatePercent"`
				} `json:"ControllerRates"`
				Ports []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					OdataType             string `json:"@odata.type"`
					LinkNetworkTechnology any    `json:"LinkNetworkTechnology"`
					OdataEtag             string `json:"@odata.etag"`
					FunctionMinBandwidth  any    `json:"FunctionMinBandwidth"`
					MaxSpeedGbps          int    `json:"MaxSpeedGbps"`
					PortID                string `json:"PortId"`
					Name                  string `json:"Name"`
					PortProtocol          string `json:"PortProtocol"`
					OdataID               string `json:"@odata.id"`
					LinkConfiguration     []any  `json:"LinkConfiguration"`
					CurrentSpeedGbps      int    `json:"CurrentSpeedGbps"`
					Ethernet              any    `json:"Ethernet"`
					ID                    string `json:"Id"`
					FunctionMaxBandwidth  any    `json:"FunctionMaxBandwidth"`
					Width                 int    `json:"Width"`

					Location struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"Location"`
					PortType string `json:"PortType"`
				} `json:"Ports"`
				CacheSummary struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
					PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
				} `json:"CacheSummary"`
				SupportedRAIDTypes           []string `json:"SupportedRAIDTypes"`
				FirmwareVersion               string   `json:"FirmwareVersion"`
				MemberID                      any      `json:"MemberId"`
				OdataID                       string   `json:"@odata.id"`
				SupportedControllerProtocols  []string `json:"SupportedControllerProtocols"`
				SerialNumber                  string   `json:"SerialNumber"`
				Manufacturer                  string   `json:"Manufacturer"`
				Model                         string   `json:"Model"`
				SupportedDeviceProtocols      []string `json:"SupportedDeviceProtocols"`
				Identifiers                   []struct {
					DurableNameFormat string `json:"DurableNameFormat"`
					DurableName       string `json:"DurableName"`
				} `json:"Identifiers"`
				Sku      string `json:"SKU"`
				Location struct {
					PartLocation struct {
						ServiceLabel         string `json:"ServiceLabel"`
						LocationOrdinalValue int    `json:"LocationOrdinalValue"`
						LocationType         string `json:"LocationType"`
					} `json:"PartLocation"`
				} `json:"Location"`
			} `json:"Controllers"`
			OdataID string `json:"@odata.id"`
			Volumes []struct {
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				ReadCachePolicy string `json:"ReadCachePolicy"`
				OdataType       string `json:"@odata.type"`
				Encrypted       bool   `json:"Encrypted"`
				EncryptionTypes []any  `json:"EncryptionTypes"`

				CapacityBytes       int64  `json:"CapacityBytes"`
				OdataEtag           string `json:"@odata.etag"`
				RAIDType            string `json:"RAIDType"`
				BlockSizeBytes      int    `json:"BlockSizeBytes"`
				Name                string `json:"Name"`
				OptimumIOSizeBytes  int    `json:"OptimumIOSizeBytes"`
				LogicalUnitNumber   int    `json:"LogicalUnitNumber"`
				OdataID             string `json:"@odata.id"`
				VolumeType          any    `json:"VolumeType"`
				DisplayName         string `json:"DisplayName"`
				Links               struct {
					DedicatedSpareDrives []any `json:"DedicatedSpareDrives"`
					Drives               []struct {
						OdataID string `json:"@odata.id"`
					} `json:"Drives"`
				} `json:"Links"`
				ID             string `json:"Id"`
				StripSizeBytes int    `json:"StripSizeBytes"`
				Identifiers    []struct {
					DurableNameFormat string `json:"DurableNameFormat"`
					DurableName       string `json:"DurableName"`
				} `json:"Identifiers"`
				Operations  []any  `json:"Operations"`
				VolumeUsage string `json:"VolumeUsage"`
			} `json:"Volumes"`
			StorageControllers []struct {
				SpeedGbps int `json:"SpeedGbps"`
				Status    struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				PCIeInterface struct {
					PCIeType    string `json:"PCIeType"`
					MaxPCIeType string `json:"MaxPCIeType"`
					MaxLanes    int    `json:"MaxLanes"`
					LanesInUse  int    `json:"LanesInUse"`
				} `json:"PCIeInterface"`
				PartNumber      string `json:"PartNumber"`
				OdataType       string `json:"@odata.type"`
				ControllerRates struct {
					ConsistencyCheckRatePercent int `json:"ConsistencyCheckRatePercent"`
					RebuildRatePercent          int `json:"RebuildRatePercent"`
					TransformationRatePercent   int `json:"TransformationRatePercent"`
				} `json:"ControllerRates"`
				Ports []struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`

					OdataType             string `json:"@odata.type"`
					LinkNetworkTechnology any    `json:"LinkNetworkTechnology"`
					OdataEtag             string `json:"@odata.etag"`
					FunctionMinBandwidth  any    `json:"FunctionMinBandwidth"`
					MaxSpeedGbps          int    `json:"MaxSpeedGbps"`
					PortID                string `json:"PortId"`
					Name                  string `json:"Name"`
					PortProtocol          string `json:"PortProtocol"`
					OdataID               string `json:"@odata.id"`
					LinkConfiguration     []any  `json:"LinkConfiguration"`
					CurrentSpeedGbps      int    `json:"CurrentSpeedGbps"`
					Ethernet              any    `json:"Ethernet"`
					ID                    string `json:"Id"`
					FunctionMaxBandwidth  any    `json:"FunctionMaxBandwidth"`
					Width                 int    `json:"Width"`
					Location              struct {
						PartLocation struct {
							ServiceLabel         string `json:"ServiceLabel"`
							LocationOrdinalValue int    `json:"LocationOrdinalValue"`
							LocationType         string `json:"LocationType"`
						} `json:"PartLocation"`
					} `json:"Location"`
					PortType string `json:"PortType"`
				} `json:"Ports"`
				CacheSummary struct {
					Status struct {
						Health string `json:"Health"`
						State  string `json:"State"`
					} `json:"Status"`
					TotalCacheSizeMiB      int `json:"TotalCacheSizeMiB"`
					PersistentCacheSizeMiB int `json:"PersistentCacheSizeMiB"`
				} `json:"CacheSummary"`
				SupportedRAIDTypes           []string `json:"SupportedRAIDTypes"`
				FirmwareVersion               string   `json:"FirmwareVersion"`
				MemberID                      any      `json:"MemberId"`
				OdataID                       string   `json:"@odata.id"`
				SupportedControllerProtocols  []string `json:"SupportedControllerProtocols"`
				SerialNumber                  string   `json:"SerialNumber"`
				Manufacturer                  string   `json:"Manufacturer"`
				Model                         string   `json:"Model"`
				SupportedDeviceProtocols      []string `json:"SupportedDeviceProtocols"`
				Identifiers                   []struct {
					DurableNameFormat string `json:"DurableNameFormat"`
					DurableName       string `json:"DurableName"`
				} `json:"Identifiers"`
				Sku      string `json:"SKU"`
				Location struct {
					PartLocation struct {
						ServiceLabel string `json:"ServiceLabel"`

						LocationOrdinalValue int    `json:"LocationOrdinalValue"`
						LocationType         string `json:"LocationType"`
					} `json:"PartLocation"`
				} `json:"Location"`
			} `json:"StorageControllers"`
			OdataType string `json:"@odata.type"`
			OdataEtag string `json:"@odata.etag"`
			Drives    []struct {
				HotspareType       string `json:"HotspareType"`
				OdataType          string `json:"@odata.type"`
				CapacityBytes      int64  `json:"CapacityBytes"`
				Name               string `json:"Name"`
				StatusIndicator    string `json:"StatusIndicator"`
				OdataID            string `json:"@odata.id"`
				NegotiatedSpeedGbs int    `json:"NegotiatedSpeedGbs"`
				Manufacturer       string `json:"Manufacturer"`
				MediaType          string `json:"MediaType"`
				Protocol           string `json:"Protocol"`
				Operations         []any  `json:"Operations"`
				IndicatorLED       any    `json:"IndicatorLED"`
				PhysicalLocation   struct {
					PartLocation struct {
						ServiceLabel         string `json:"ServiceLabel"`
						LocationOrdinalValue int    `json:"LocationOrdinalValue"`
						LocationType         string `json:"LocationType"`
					} `json:"PartLocation"`
				} `json:"PhysicalLocation"`
				Status struct {
					Health string `json:"Health"`
					State  string `json:"State"`
				} `json:"Status"`
				OdataEtag                     string `json:"@odata.etag"`
				WriteCacheEnabled              bool   `json:"WriteCacheEnabled"`
				BlockSizeBytes                 int    `json:"BlockSizeBytes"`
				PredictedMediaLifeLeftPercent  any    `json:"PredictedMediaLifeLeftPercent"`
				CapableSpeedGbs                int    `json:"CapableSpeedGbs"`
				FailurePredicted               bool   `json:"FailurePredicted"`
				Revision                       string `json:"Revision"`
				SerialNumber                   string `json:"SerialNumber"`
				Model                          string `json:"Model"`
				Links                          struct {
					Chassis any    `json:"Chassis"`
					Volumes []struct {
						OdataID string `json:"@odata.id"`
					} `json:"Volumes"`
				} `json:"Links"`
				ID          string `json:"Id"`
				Identifiers []struct {
					DurableNameFormat string `json:"DurableNameFormat"`

					DurableName string `json:"DurableName"`
				} `json:"Identifiers"`
				Location any `json:"Location"`
			} `json:"Drives"`
			ID   string `json:"Id"`
			Name string `json:"Name"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"LocalStorageV2"`
	Devices struct {
		URI             string    `json:"uri"`
		CollectionState string    `json:"collectionState"`
		Modified        time.Time `json:"modified"`
		Data            []struct {
			Status struct {
				Health any `json:"Health"`
				State  any `json:"State"`
			} `json:"Status,omitempty"`
			PartNumber      string `json:"PartNumber"`
			OdataType       string `json:"@odata.type"`
			OdataEtag       string `json:"@odata.etag"`
			ProductVersion  string `json:"ProductVersion"`
			FirmwareVersion struct {
				Current struct {
					VersionString string `json:"VersionString"`
				} `json:"Current"`
			} `json:"FirmwareVersion"`
			Name              string `json:"Name"`
			DeviceType        string `json:"DeviceType"`
			SerialNumber      string `json:"SerialNumber"`
			ProductPartNumber string `json:"ProductPartNumber"`
			Manufacturer      string `json:"Manufacturer"`
			ID                string `json:"Id"`
			Location          string `json:"Location"`
			Status0           struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
			MCTPProtocolDisabled bool `json:"MCTPProtocolDisabled,omitempty"`
			Status1              struct {
				State string `json:"State"`
			} `json:"Status,omitempty"`
		} `json:"data"`
		Etag  string `json:"etag"`
		Count int    `json:"count"`
		Name  string `json:"name"`
	} `json:"Devices"`
} `json:"subResources,omitempty"`

}

