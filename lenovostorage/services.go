// Package lenovostorage реализует клиент для сбора информации со СХД Lenovo (Storage Manager REST API,
// /devmgr/v2/storage-systems/...): аутентификация (session.go), а также получение томов, пулов хранения,
// хостов, групп хостов и статистики через методы Session.
package lenovostorage

// https://10.177.180.101/devmgr/v2/storage-systems/1/access-volume
type Volume struct {
	Offline           bool   `json:"offline"`
	ExtremeProtection bool   `json:"extremeProtection"`
	VolumeHandle      int    `json:"volumeHandle"`
	RaidLevel         string `json:"raidLevel"`
	SectorOffset      string `json:"sectorOffset"`
	WorldWideName     string `json:"worldWideName"`
	Label             string `json:"label"`
	BlkSize           int    `json:"blkSize"`
	Capacity          string `json:"capacity"`
	ReconPriority     int    `json:"reconPriority"`
	SegmentSize       int    `json:"segmentSize"`
	Action            string `json:"action"`
	Cache             struct {
		Cwob                bool   `json:"cwob"`
		EnterpriseCacheDump bool   `json:"enterpriseCacheDump"`
		MirrorActive        bool   `json:"mirrorActive"`
		MirrorEnable        bool   `json:"mirrorEnable"`
		ReadCacheActive     bool   `json:"readCacheActive"`
		ReadCacheEnable     bool   `json:"readCacheEnable"`
		WriteCacheActive    bool   `json:"writeCacheActive"`
		WriteCacheEnable    bool   `json:"writeCacheEnable"`
		CacheFlushModifier  string `json:"cacheFlushModifier"`
		ReadAheadMultiplier int    `json:"readAheadMultiplier"`
	} `json:"cache"`
	MediaScan struct {
		Enable                 bool `json:"enable"`
		ParityValidationEnable bool `json:"parityValidationEnable"`
	} `json:"mediaScan"`
	VolumeRef        string `json:"volumeRef"`
	Status           string `json:"status"`
	VolumeGroupRef   string `json:"volumeGroupRef"`
	CurrentManager   string `json:"currentManager"`
	PreferredManager string `json:"preferredManager"`
	Perms            struct {
		MapToLUN                      bool `json:"mapToLUN"`
		SnapShot                      bool `json:"snapShot"`
		Format                        bool `json:"format"`
		Reconfigure                   bool `json:"reconfigure"`
		MirrorPrimary                 bool `json:"mirrorPrimary"`
		MirrorSecondary               bool `json:"mirrorSecondary"`
		CopySource                    bool `json:"copySource"`
		CopyTarget                    bool `json:"copyTarget"`
		Readable                      bool `json:"readable"`
		Writable                      bool `json:"writable"`
		Rollback                      bool `json:"rollback"`
		MirrorSync                    bool `json:"mirrorSync"`
		NewImage                      bool `json:"newImage"`
		AllowDVE                      bool `json:"allowDVE"`
		AllowDSS                      bool `json:"allowDSS"`
		ConcatVolumeMember            bool `json:"concatVolumeMember"`
		FlashReadCache                bool `json:"flashReadCache"`
		AsyncMirrorPrimary            bool `json:"asyncMirrorPrimary"`
		AsyncMirrorSecondary          bool `json:"asyncMirrorSecondary"`
		PitGroup                      bool `json:"pitGroup"`
		CacheParametersChangeable     bool `json:"cacheParametersChangeable"`
		AllowThinManualExpansion      bool `json:"allowThinManualExpansion"`
		AllowThinGrowthParametersChange bool `json:"allowThinGrowthParametersChange"`
		AllowImportTarget             bool `json:"allowImportTarget"`
	} `json:"perms"`
	MgmtClientAttribute                         int    `json:"mgmtClientAttribute"`
	DssPreallocEnabled                          bool   `json:"dssPreallocEnabled"`
	DssMaxSegmentSize                           int    `json:"dssMaxSegmentSize"`
	PreReadRedundancyCheckEnabled               bool   `json:"preReadRedundancyCheckEnabled"`
	ProtectionInformationCapable                bool   `json:"protectionInformationCapable"`
	ProtectionType                              string `json:"protectionType"`
	ApplicationTagOwned                         bool   `json:"applicationTagOwned"`
	RepairedBlockCount                          int    `json:"repairedBlockCount"`
	ExtendedUniqueIdentifier                    string `json:"extendedUniqueIdentifier"`
	CacheMirroringValidateProtectionInformation bool   `json:"cacheMirroringValidateProtectionInformation"`
	ExpectedProtectionInformationAppTag         int    `json:"expectedProtectionInformationAppTag"`
	HostUnmapEnabled                            bool   `json:"hostUnmapEnabled"`
	CachePoolID                                 int    `json:"cachePoolID"`
	BlkSizePhysical                             int    `json:"blkSizePhysical"`
	VolumeUse                                   string `json:"volumeUse"`
	VolumeFull                                  bool   `json:"volumeFull"`
	VolumeCopyTarget                            bool   `json:"volumeCopyTarget"`
	VolumeCopySource                            bool   `json:"volumeCopySource"`
	PitBaseVolume                               bool   `json:"pitBaseVolume"`
	AsyncMirrorTarget                           bool   `json:"asyncMirrorTarget"`
	AsyncMirrorSource                           bool   `json:"asyncMirrorSource"`
	RemoteMirrorSource                          bool   `json:"remoteMirrorSource"`
	RemoteMirrorTarget                          bool   `json:"remoteMirrorTarget"`
	DiskPool                                    bool   `json:"diskPool"`
	FlashCached                                 bool   `json:"flashCached"`
	IncreasingBy                                string `json:"increasingBy"`
	Metadata                                    []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"metadata"`
	DataAssurance         bool   `json:"dataAssurance"`
	ObjectType            string `json:"objectType"`
	Wwn                   string `json:"wwn"`
	PreferredControllerID string `json:"preferredControllerId"`
	TotalSizeInBytes      string `json:"totalSizeInBytes"`
	OnlineVolumeCopy      bool   `json:"onlineVolumeCopy"`
	ListOfMappings        []struct {
		LunMappingRef string `json:"lunMappingRef"`
		Lun           int    `json:"lun"`
		Ssid          int    `json:"ssid"`
		Perms         int    `json:"perms"`
		VolumeRef     string `json:"volumeRef"`
		Type          string `json:"type"`
		MapRef        string `json:"mapRef"`
		ID            string `json:"id"`
	} `json:"listOfMappings"`
	Mapped                bool   `json:"mapped"`
	CurrentControllerID   string `json:"currentControllerId"`
	CacheSettings         struct {
		Cwob                bool   `json:"cwob"`
		EnterpriseCacheDump bool   `json:"enterpriseCacheDump"`
		MirrorActive        bool   `json:"mirrorActive"`
		MirrorEnable        bool   `json:"mirrorEnable"`
		ReadCacheActive     bool   `json:"readCacheActive"`
		ReadCacheEnable     bool   `json:"readCacheEnable"`
		WriteCacheActive    bool   `json:"writeCacheActive"`
		WriteCacheEnable    bool   `json:"writeCacheEnable"`
		CacheFlushModifier  string `json:"cacheFlushModifier"`
		ReadAheadMultiplier int    `json:"readAheadMultiplier"`
	} `json:"cacheSettings"`
	ThinProvisioned bool   `json:"thinProvisioned"`
	Name            string `json:"name"`
	ID              string `json:"id"`
}
// https://10.177.100.101/devmgr/v2/storage-systems/1/access-volume
type AccessVolume struct {
	Enabled               bool   `json:"enabled"`
	VolumeHandle          int    `json:"volumeHandle"`
	Capacity              string `json:"capacity"`
	AccessVolumeRef       string `json:"accessVolumeRef"`
	Reserved1             string `json:"reserved1"`
	ObjectType            string `json:"objectType"`
	Wwn                   string `json:"wwn"`
	PreferredControllerID string `json:"preferredControllerId"`
	TotalSizeInBytes      string `json:"totalSizeInBytes"`
	ListOfMappings        []struct {
		LunMappingRef string `json:"lunMappingRef"`
		Lun           int    `json:"lun"`
		Ssid          int    `json:"ssid"`
		Perms         int    `json:"perms"`
		VolumeRef     string `json:"volumeRef"`
		Type          string `json:"type"`
		MapRef        string `json:"mapRef"`
		ID            string `json:"id"`
	} `json:"listOfMappings"`
	Mapped              bool   `json:"mapped"`
	CurrentControllerId string `json:"currentControllerId"`
	Name                string `json:"name"`
	ID                  string `json:"id"`
}

// Host - описание хоста, подключённого к СХД Lenovo (инициаторы, порты, отображения томов)
type Host struct {
	HostRef                                   string `json:"hostRef"`
	ClusterRef                                string `json:"clusterRef"`
	Label                                     string `json:"label"`
	IsSAControlled                            bool   `json:"isSAControlled"`
	ConfirmLUNMappingCreation                 bool   `json:"confirmLUNMappingCreation"`
	HostTypeIndex                             int    `json:"hostTypeIndex"`
	ProtectionInformationCapableAccessMethod bool   `json:"protectionInformationCapableAccessMethod"`
	IsLargeBlockFormatHost                    bool   `json:"isLargeBlockFormatHost"`
	IsLun0Restricted                         bool   `json:"isLun0Restricted"`
	Ports                                     []any  `json:"ports"`
	Initiators                                []struct {
		InitiatorRef string `json:"initiatorRef"`
		NodeName     struct {
			IoInterfaceType string `json:"ioInterfaceType"`
			IscsiNodeName   string `json:"iscsiNodeName"`
			RemoteNodeWWN   any    `json:"remoteNodeWWN"`
			NvmeNodeName    any    `json:"nvmeNodeName"`
		} `json:"nodeName"`
		Alias struct {
			IoInterfaceType string `json:"ioInterfaceType"`
			IscsiAlias      string `json:"iscsiAlias"`
		} `json:"alias"`
		Label                 string `json:"label"`
		ConfiguredAuthMethods struct {
			AuthMethodData []struct {
				AuthMethod string `json:"authMethod"`
				ChapSecret any    `json:"chapSecret"`
			} `json:"authMethodData"`
		} `json:"configuredAuthMethods"`
		HostRef           string `json:"hostRef"`
		InitiatorInactive bool   `json:"initiatorInactive"`
		InitiatorNodeName struct {
			NodeName struct {
				IoInterfaceType string `json:"ioInterfaceType"`
				IscsiNodeName   string `json:"iscsiNodeName"`
				RemoteNodeWWN   any    `json:"remoteNodeWWN"`
				NvmeNodeName    any    `json:"nvmeNodeName"`
			} `json:"nodeName"`
			InterfaceType string `json:"interfaceType"`
		} `json:"initiatorNodeName"`
		ID string `json:"id"`
	} `json:"initiators"`
	HostSidePorts []struct {
		ID                 string `json:"id"`
		Type               string `json:"type"`
		Address            string `json:"address"`
		Label              string `json:"label"`
		MtpIoInterfaceType string `json:"mtpIoInterfaceType"`
		Name               string `json:"name"`
	} `json:"hostSidePorts"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StoragePool - описание пула хранения (дисковой группы) СХД Lenovo
type StoragePool struct {
	SequenceNum                  int    `json:"sequenceNum"`
	Offline                      bool   `json:"offline"`
	RaidLevel                    string `json:"raidLevel"`
	WorldwideName                string `json:"worldwideName"`
	VolumeGroupRef               string `json:"volumeGroupRef"`
	Reserved1                    string `json:"reserved1"`
	Reserved2                    string `json:"reserved2"`
	TrayLossProtection           bool   `json:"trayLossProtection"`
	Label                        string `json:"label"`
	State                        string `json:"state"`
	SpindleSpeedMatch            bool   `json:"spindleSpeedMatch"`
	SpindleSpeed                 int    `json:"spindleSpeed"`
	IsInaccessible               bool   `json:"isInaccessible"`
	SecurityType                 string `json:"securityType"`
	DrawerLossProtection         bool   `json:"drawerLossProtection"`
	ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
	ProtectionInformationCapabilities struct {
		ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
		ProtectionType               string `json:"protectionType"`
	} `json:"protectionInformationCapabilities"`
	VolumeGroupData struct {
		Type         string `json:"type"`
		DiskPoolData struct {
			ReconstructionReservedDriveCount        int    `json:"reconstructionReservedDriveCount"`
			ReconstructionReservedAmt               string `json:"reconstructionReservedAmt"`
			ReconstructionReservedDriveCountCurrent int    `json:"reconstructionReservedDriveCountCurrent"`
			PoolUtilizationWarningThreshold         int    `json:"poolUtilizationWarningThreshold"`
			PoolUtilizationCriticalThreshold        int    `json:"poolUtilizationCriticalThreshold"`
			PoolUtilizationState                    string `json:"poolUtilizationState"`
			UnusableCapacity                        string `json:"unusableCapacity"`
			DegradedReconstructPriority             string `json:"degradedReconstructPriority"`
			CriticalReconstructPriority             string `json:"criticalReconstructPriority"`
			BackgroundOperationPriority             string `json:"backgroundOperationPriority"`
			AllocGranularity                        string `json:"allocGranularity"`
			MinimumDriveCount                       int    `json:"minimumDriveCount"`
		} `json:"diskPoolData"`
	} `json:"volumeGroupData"`
	Usage                    string `json:"usage"`
	DriveBlockFormat         string `json:"driveBlockFormat"`
	ReservedSpaceAllocated   bool   `json:"reservedSpaceAllocated"`
	SecurityLevel            string `json:"securityLevel"`
	DulbeEnabled             bool   `json:"dulbeEnabled"`
	BlkSizeSupported         []int  `json:"blkSizeSupported"`
	BlkSizeRecommended       int    `json:"blkSizeRecommended"`
	UsedSpace                string `json:"usedSpace"`
	TotalRaidedSpace         string `json:"totalRaidedSpace"`
	Extents                  []struct {
		SectorOffset    string `json:"sectorOffset"`
		RawCapacity     string `json:"rawCapacity"`
		RaidLevel       string `json:"raidLevel"`
		VolumeGroupRef  string `json:"volumeGroupRef"`
		FreeExtentRef   string `json:"freeExtentRef"`
		Reserved1       string `json:"reserved1"`
		Reserved2       string `json:"reserved2"`
		DdpRAIDCapacities []struct {
			DdpVolRAIDLevel string `json:"ddpVolRAIDLevel"`
			UsableCapacity  string `json:"usableCapacity"`
		} `json:"ddpRAIDCapacities"`
	} `json:"extents"`
	LargestFreeExtentSize  string `json:"largestFreeExtentSize"`
	RaidStatus             string `json:"raidStatus"`
	FreeSpace              string `json:"freeSpace"`
	DrivePhysicalType      string `json:"drivePhysicalType"`
	DriveMediaType         string `json:"driveMediaType"`
	NormalizedSpindleSpeed string `json:"normalizedSpindleSpeed"`
	DiskPool               bool   `json:"diskPool"`
	ID                     string `json:"id"`
	Name                   string `json:"name"`
}

// /devmgr/v2/storage-systems/1/host-groups
type HostGroup struct {
	ClusterRef                                 string `json:"clusterRef"`
	Label                                      string `json:"label"`
	IsSAControlled                             bool   `json:"isSAControlled"`
	ConfirmLUNMappingCreation                  bool   `json:"confirmLUNMappingCreation"`
	ProtectionInformationCapableAccessMethod bool   `json:"protectionInformationCapableAccessMethod"`
	IsLun0Restricted                           bool   `json:"isLun0Restricted"`
	ID                                         string `json:"id"`
	Name                                       string `json:"name"`
}

// /devmgr/v2/storage-systems/1/about
type About struct {
	Name               string `json:"name"`
	Wwn                string `json:"wwn"`
	SerialNumber       string `json:"serialNumber"`
	ControllerPosition int    `json:"controllerPosition"`
}
// /devmgr/v2/storage-systems/1/graph
type Graph struct {
	ConfigGeneration string `json:"configGeneration"`
	Sa               struct {
		SaData struct {
			NeedsAttention bool `json:"needsAttention"`
			Fixing         bool `json:"fixing"`
			SaID           struct {
				WorldWideName string `json:"worldWideName"`
			} `json:"saId"`
			ManagementClassName           string   `json:"managementClassName"`
			StorageArrayLabel             string   `json:"storageArrayLabel"`
			BootTime                      string   `json:"bootTime"`
			FwVersion                     string   `json:"fwVersion"`
			AppVersion                    string   `json:"appVersion"`
			BootVersion                   string   `json:"bootVersion"`
			NvsramVersion                 string   `json:"nvsramVersion"`
			FwPrefix                      string   `json:"fwPrefix"`
			ChassisSerialNumber           string   `json:"chassisSerialNumber"`
			EventConfigurationDataVersion string   `json:"eventConfigurationDataVersion"`
			ArrayAttributes               []int    `json:"arrayAttributes"`
			Res4                          []any    `json:"res4"`
			Res5                          []any    `json:"res5"`
			Res6                          []any    `json:"res6"`
			Res7                          string   `json:"res7"`
			ExtendedSAData                struct {
				Reserved1    string `json:"reserved1"`
				CodeVersions []struct {
					CodeModule    string `json:"codeModule"`
					VersionString string `json:"versionString"`
				} `json:"codeVersions"`
			} `json:"extendedSAData"`
		} `json:"saData"`
		SafeID struct {
			WorldWideName string `json:"worldWideName"`
		} `json:"safeId"`
		Cache struct {
			CacheBlkSize                        int    `json:"cacheBlkSize"`
			DemandFlushAmount                   int    `json:"demandFlushAmount"`
			DemandFlushThreshold                int    `json:"demandFlushThreshold"`
			Reserved1                           string `json:"reserved1"`
			Reserved2                           string `json:"reserved2"`
			CacheMirroringProtectionInformation bool   `json:"cacheMirroringProtectionInformation"`
		} `json:"cache"`
		AccessVolume struct {
			Enabled               bool   `json:"enabled"`
			VolumeHandle          int    `json:"volumeHandle"`
			Capacity              string `json:"capacity"`
			AccessVolumeRef       string `json:"accessVolumeRef"`
			Reserved1             string `json:"reserved1"`
			ObjectType            string `json:"objectType"`
			Wwn                   string `json:"wwn"`
			PreferredControllerID string `json:"preferredControllerId"`
			TotalSizeInBytes      string `json:"totalSizeInBytes"`
			ListOfMappings        []struct {
				LunMappingRef string `json:"lunMappingRef"`
				Lun           int    `json:"lun"`
				Ssid          int    `json:"ssid"`
				Perms         int    `json:"perms"`
				VolumeRef     string `json:"volumeRef"`
				Type          string `json:"type"`
				MapRef        string `json:"mapRef"`
				ID            string `json:"id"`
			} `json:"listOfMappings"`
			Mapped              bool   `json:"mapped"`
			CurrentControllerID string `json:"currentControllerId"`
			Name                string `json:"name"`
			ID                  string `json:"id"`
		} `json:"accessVolume"`
		FeatureParameters struct {
			MaxHotSpares                              int      `json:"maxHotSpares"`
			CacheBlockSizes                           []int    `json:"cacheBlockSizes"`
			MaxDCEDrives                              int      `json:"maxDCEDrives"`
			MaxReadAheadMultiplier                    int      `json:"maxReadAheadMultiplier"`
			MinMediaScanPeriod                        int      `json:"minMediaScanPeriod"`
			MaxMediaScanPeriod                        int      `json:"maxMediaScanPeriod"`
			SupportedSegSizes                         []int    `json:"supportedSegSizes"`
			NumStandardSegSizes                       int      `json:"numStandardSegSizes"`
			SegTransitions                            []string `json:"segTransitions"`
			MappableLUNCount                          int      `json:"mappableLUNCount"`
			MaxPartitionCount                         int      `json:"maxPartitionCount"`
			MaxMappingsPerVolume                      int      `json:"maxMappingsPerVolume"`
			MaxVolumes                                int      `json:"maxVolumes"`
			MaxSnapshots                              int      `json:"maxSnapshots"`
			MaxSnapshotsPerBase                       int      `json:"maxSnapshotsPerBase"`
			MinRepositorySize                         int      `json:"minRepositorySize"`
			MaxMirrors                                int      `json:"maxMirrors"`
			MaxVolCopys                               int      `json:"maxVolCopys"`
			MaxConcurrentBackgroundCopys              int      `json:"maxConcurrentBackgroundCopys"`
			MaxUtmLunValue                            int      `json:"maxUtmLunValue"`
			Reserved1                                 string   `json:"reserved1"`
			Reserved2                                 string   `json:"reserved2"`
			BatteryNotInConfiguration                 bool     `json:"battery_not_in_configuration"`
			SupportedDriveTypes                       []string `json:"supportedDriveTypes"`
			CurrentMaxMirrors                         int      `json:"currentMaxMirrors"`
			MaxVolumesPerGroup                        int      `json:"maxVolumesPerGroup"`
			MaxDriveDownloadFiles                     int      `json:"maxDriveDownloadFiles"`
			MaxMgmtClientRecords                      int      `json:"maxMgmtClientRecords"`
			MaxMgmtClientRecordSize                   int      `json:"maxMgmtClientRecordSize"`
			RedundancyMode                            string   `json:"redundancyMode"`
			MaxExpansionDriveTrays                    int      `json:"maxExpansionDriveTrays"`
			SupportedAuthMethods                      []string `json:"supportedAuthMethods"`
			MaxIscsiIpv6LocalAddressesPerPort         int      `json:"maxIscsiIpv6LocalAddressesPerPort"`
			MaxIscsiIpv6RoutableAddressesPerPort      int      `json:"maxIscsiIpv6RoutableAddressesPerPort"`
			AbsoluteMaxDssSegmentSize                 int      `json:"absoluteMaxDssSegmentSize"`
			SmartBatteryTechnology                    bool     `json:"smartBatteryTechnology"`
			WriteCachingEnabledDuringLearnCycle       bool     `json:"writeCachingEnabledDuringLearnCycle"`
			MaxSSDsSupported                          int      `json:"maxSSDsSupported"`
			SsdInterfaceTypeMixingSupported           bool     `json:"ssdInterfaceTypeMixingSupported"`
			MaxDriveSlots                             int      `json:"maxDriveSlots"`
			SupportedProtectionTypes                  []string `json:"supportedProtectionTypes"`
			SupportedEventPriorities                 []string `json:"supportedEventPriorities"`
			MaxFlashCacheSize                         string   `json:"maxFlashCacheSize"`
			DefaultMediaScanPeriod                    int      `json:"defaultMediaScanPeriod"`
			MaxNvmeofRoceV2Ipv6LocalAddressesPerPort    int    `json:"maxNvmeofRoceV2Ipv6LocalAddressesPerPort"`
			MaxNvmeofRoceV2Ipv6RoutableAddressesPerPort int    `json:"maxNvmeofRoceV2Ipv6RoutableAddressesPerPort"`
			MaxCachePoolID                            int      `json:"maxCachePoolID"`
			MaxExternalKeyServers                     int      `json:"maxExternalKeyServers"`
		} `json:"featureParameters"`
		MediaScanPeriod      int      `json:"mediaScanPeriod"`
		Capabilities         []string `json:"capabilities"`
		PremiumFeatures      []any    `json:"premiumFeatures"`
		TrayPositionList     struct {
			TrayPosition []any `json:"trayPosition"`
		} `json:"trayPositionList"`
		HostPortType     []string `json:"hostPortType"`
		HostSpecificVals []struct {
			WaitForDownload                            bool   `json:"waitForDownload"`
			NotReadySenseForQuiescenceInProgress       bool   `json:"notReadySenseForQuiescenceInProgress"`
			NotReadySenseForQuiescenceInProgress2      bool   `json:"notReadySenseForQuiescenceInProgress2"`
			CheckConditionForVendorUniqueUAConditions bool   `json:"checkConditionForVendorUniqueUAConditions"`
			ReportDeferredErrors                       bool   `json:"reportDeferredErrors"`
			IgnoreForceUnitAccess                      bool   `json:"ignoreForceUnitAccess"`
			IgnoreWriteCacheDisable                    bool   `json:"ignoreWriteCacheDisable"`
			AllowReservationOnUnownedLUNs              bool   `json:"allowReservationOnUnownedLUNs"`
			WorldWideNameInStandardInquiry             bool   `json:"worldWideNameInStandardInquiry"`
			PropagateHostBusResets                     bool   `json:"propagateHostBusResets"`
			ReportSCCDeviceEnable                      bool   `json:"reportSCCDeviceEnable"`
			ConditionalFCLogout                        bool   `json:"conditionalFCLogout"`
			ForceSCSI2                                 bool   `json:"forceSCSI2"`
			ReportUnconfiguredLUNs                     bool   `json:"reportUnconfiguredLUNs"`
			IgnoreSyncCache                            bool   `json:"ignoreSyncCache"`
			IgnoreUTMLunOwnership                      bool   `json:"ignoreUTMLunOwnership"`
			ReportLUNPreferredPathInStandardInquiry    bool   `json:"reportLUNPreferredPathInStandardInquiry"`
			AutoLUNTransfer                            bool   `json:"autoLUNTransfer"`
			InquiryUnconfiguredDeviceType              int    `json:"inquiryUnconfiguredDeviceType"`
			InquiryUnownedDeviceType                   int    `json:"inquiryUnownedDeviceType"`
			UnownedBlock0Read                          int    `json:"unownedBlock0Read"`
			RetainLogins                               bool   `json:"retainLogins"`
			HostPortType                               string `json:"hostPortType"`
			MaxRestrictedLUNs                          int    `json:"maxRestrictedLUNs"`
			Reserved1                                  string `json:"reserved1"`
			AvtExclusionExtentsEnabled                 bool   `json:"avtExclusionExtentsEnabled"`
			SuppressVolOffPreferredPathAlert           bool   `json:"suppressVolOffPreferredPathAlert"`
			HostType                                   string `json:"hostType"`
			TpgsSupported                              bool   `json:"tpgsSupported"`
			VolumeFormatOptions                        bool   `json:"volumeFormatOptions"`
			RestrictedLun0                             bool   `json:"restrictedLun0"`
			UsePage2CQuiescenceTimeout                 bool   `json:"usePage2CQuiescenceTimeout"`
			AsymmetricLUAccess                         bool   `json:"asymmetricLUAccess"`
			IgnoreISCSISessionIDInITN                  bool   `json:"ignoreISCSISessionIDInITN"`
			EnableLUCheckConditionParallelResponses    bool   `json:"enableLUCheckConditionParallelResponses"`
			DisableExplicitTPGS                        bool   `json:"disableExplicitTPGS"`
			UnownedCheckConditionDuringFWActivation    bool   `json:"unownedCheckConditionDuringFWActivation"`
			AllowPreferredOwnerReportingSuppression    bool   `json:"allowPreferredOwnerReportingSuppression"`
			EnableTargetFailback                       bool   `json:"enableTargetFailback"`
			EnableAutoLoadBalance                      bool   `json:"enableAutoLoadBalance"`
			QuiescenceInProgressSenseKey              int    `json:"quiescenceInProgressSenseKey"`
			QuiescenceInProgressASC                   int    `json:"quiescenceInProgressASC"`
			QuiescenceInProgressASCQ                  int    `json:"quiescenceInProgressASCQ"`
			PersistentID                               int    `json:"persistentID"`
			NvmeVUFailoverSupported                    bool   `json:"nvmeVUFailoverSupported"`
			NvmeStandardFailoverSupported              bool   `json:"nvmeStandardFailoverSupported"`
			HostUnmapEnabled                           bool   `json:"hostUnmapEnabled"`
			HostTypeCategory                           string `json:"hostTypeCategory"`
			HostTypePriority                           int    `json:"hostTypePriority"`
			HostConnectivityReportingSupported         bool   `json:"hostConnectivityReportingSupported"`
			Index                                      int    `json:"index"`
			Default                                    bool   `json:"default"`
		} `json:"hostSpecificVals"`
		Compatibility []struct {
			FwVersion  string `json:"fwVersion"`
			Compatible bool   `json:"compatible"`
		} `json:"compatibility"`
		UsageHints []struct {
			UsageHint   string `json:"usageHint"`
			SegmentSize int    `json:"segmentSize"`
			ReadAhead   int    `json:"readAhead"`
		} `json:"usageHints"`
		DefinedPartitionCount    int    `json:"definedPartitionCount"`
		DefaultHostPortTypeIndex int    `json:"defaultHostPortTypeIndex"`
		RemoteAccessID           string `json:"remoteAccessID"`
		RemoteMirroringActive    bool   `json:"remoteMirroringActive"`
		VolXferAlertDelayPeriod  int    `json:"volXferAlertDelayPeriod"`
		Reserved1                string `json:"reserved1"`
		Reserved2                string `json:"reserved2"`
		StagedFirmware           struct {
			StagedFwValid       bool     `json:"stagedFwValid"`
			FwVersion           string   `json:"fwVersion"`
			AppVersion          string   `json:"appVersion"`
			BootVersion         string   `json:"bootVersion"`
			NvsramVersion       string   `json:"nvsramVersion"`
			TimeStamp           string   `json:"timeStamp"`
			NvsramVersionString string   `json:"nvsramVersionString"`
			CodeVersions        []any    `json:"codeVersions"`
		} `json:"stagedFirmware"`
		UnreadableSectors struct {
			Count    int `json:"count"`
			MaxLimit int `json:"maxLimit"`
		} `json:"unreadableSectors"`
		DriveChannels []struct {
			Channel             int    `json:"channel"`
			State               string `json:"state"`
			Reason              string `json:"reason"`
			DegradingController string `json:"degradingController"`
			StatsControllerA    struct {
				StartTimeStamp    string `json:"startTimeStamp"`
				EndTimeStamp      string `json:"endTimeStamp"`
				TotalIoCount      int    `json:"totalIoCount"`
				TotalIoErrorCount int    `json:"totalIoErrorCount"`
				LinkStatus        string `json:"linkStatus"`
				ErrorCounts       struct {
					NonChannelErrors         int `json:"nonChannelErrors"`
					TimeoutErrors            int `json:"timeoutErrors"`
					ControllerDetectedErrors int `json:"controllerDetectedErrors"`
					DriveDetectedErrors      int `json:"driveDetectedErrors"`
					LinkDownErrors           int `json:"linkDownErrors"`
				} `json:"errorCounts"`
			} `json:"statsControllerA"`
			StatsControllerB struct {
				StartTimeStamp    string `json:"startTimeStamp"`
				EndTimeStamp      string `json:"endTimeStamp"`
				TotalIoCount      int    `json:"totalIoCount"`
				TotalIoErrorCount int    `json:"totalIoErrorCount"`
				LinkStatus        string `json:"linkStatus"`
				ErrorCounts       struct {
					NonChannelErrors         int `json:"nonChannelErrors"`
					TimeoutErrors            int `json:"timeoutErrors"`
					ControllerDetectedErrors int `json:"controllerDetectedErrors"`
					DriveDetectedErrors      int `json:"driveDetectedErrors"`
					LinkDownErrors           int `json:"linkDownErrors"`
				} `json:"errorCounts"`
			} `json:"statsControllerB"`
		} `json:"driveChannels"`
		PortLocations []any `json:"portLocations"`
		DdcTag        int   `json:"ddcTag"`
		FeatureBundle struct {
			Name            string `json:"name"`
			BundledFeatures []struct {
				FeatureID        string `json:"featureId"`
				FeatureAttribute []struct {
					FeatureAttrID      string `json:"featureAttrId"`
					KeyEnabled         bool   `json:"keyEnabled"`
					KeyLimit           int    `json:"keyLimit"`
					ShippedEnabled     bool   `json:"shippedEnabled"`
					ShippedLimit       int    `json:"shippedLimit"`
					SubjectToGoldKey   bool   `json:"subjectToGoldKey"`
					SupportedValues    any    `json:"supportedValues"`
					EvaluationDuration int    `json:"evaluationDuration"`
				} `json:"featureAttribute"`
				BundleSubmodelID string `json:"bundleSubmodelId"`
			} `json:"bundledFeatures"`
			FeatureState []struct {
				Capability               string `json:"capability"`
				IsEnabled                bool   `json:"isEnabled"`
				IsCompliant              bool   `json:"isCompliant"`
				IsWithinLimits           bool   `json:"isWithinLimits"`
				FeatureID                string `json:"featureId"`
				FeatureRef               string `json:"featureRef"`
				Limit                    int    `json:"limit"`
				Duration                 int    `json:"duration"`
				EnabledTime              string `json:"enabledTime"`
				SupportedFeatureBundleID int    `json:"supportedFeatureBundleId"`
				PermanentLicenseApplied  bool   `json:"permanentLicenseApplied"`
			} `json:"featureState"`
		} `json:"featureBundle"`
		ChannelPortLocations []struct {
			PortRef    string `json:"portRef"`
			PortParent struct {
				Type          string `json:"type"`
				ControllerRef string `json:"controllerRef"`
				EsmRef        any    `json:"esmRef"`
				MinihubRef    any    `json:"minihubRef"`
				HicRef        any    `json:"hicRef"`
			} `json:"portParent"`
			PortNumber       int    `json:"portNumber"`
			Channel          int    `json:"channel"`
			ChannelType      string `json:"channelType"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  string `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference any    `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			IsTrunkActive bool   `json:"isTrunkActive"`
			ChannelList   []int  `json:"channelList"`
			ID            string `json:"id"`
		} `json:"channelPortLocations"`
		IscsiEntityData struct {
			IcmpPingResponseEnabled         bool   `json:"icmpPingResponseEnabled"`
			UnnamedDiscoverySessionsEnabled bool   `json:"unnamedDiscoverySessionsEnabled"`
			IsnsServerTcpListenPort         int    `json:"isnsServerTcpListenPort"`
			Ipv4IsnsServerAddressConfigMethod string `json:"ipv4IsnsServerAddressConfigMethod"`
			Ipv4IsnsServerAddress           string `json:"ipv4IsnsServerAddress"`
			Ipv6IsnsServerAddressConfigMethod string `json:"ipv6IsnsServerAddressConfigMethod"`
			Ipv6IsnsServerAddress           string `json:"ipv6IsnsServerAddress"`
			IsnsRegistrationState           string `json:"isnsRegistrationState"`
			IsnsServerRegistrationEnabled   bool   `json:"isnsServerRegistrationEnabled"`
			HostPortsConfiguredDHCP         bool   `json:"hostPortsConfiguredDHCP"`
		} `json:"iscsiEntityData"`
		HostType                 []string `json:"hostType"`
		DefaultHostTypeIndex     int      `json:"defaultHostTypeIndex"`
		RvmControllerHostChannel int      `json:"rvmControllerHostChannel"`
		SmartBatteryData         struct {
			LastBatteryLearnCycle     string `json:"lastBatteryLearnCycle"`
			NextBatteryLearnCycle     string `json:"nextBatteryLearnCycle"`
			BatteryLearnCycleInterval int    `json:"batteryLearnCycleInterval"`
		} `json:"smartBatteryData"`
		InvalidSystemConfig bool   `json:"invalidSystemConfig"`
		SecurityLockKey     bool   `json:"securityLockKey"`
		LockKeyIDRef        string `json:"lockKeyIDRef"`
		SyslogConfiguration struct {
			SyslogEnabled          bool `json:"syslogEnabled"`
			SyslogServerPortNumber int  `json:"syslogServerPortNumber"`
			SyslogServerIPAddress  struct {
				AddressType string `json:"addressType"`
				Ipv4Address any    `json:"ipv4Address"`
				Ipv6Address string `json:"ipv6Address"`
			} `json:"syslogServerIPAddress"`
		} `json:"syslogConfiguration"`
		SecurityExternalKmsEnable    bool   `json:"securityExternalKmsEnable"`
		NewKeySequence               int    `json:"newKeySequence"`
		SecurityExternalKmsKeyStatus string `json:"securityExternalKmsKeyStatus"`
		LockKeyNeededList            []any  `json:"lockKeyNeededList"`
		DplCoreDumpData              struct {
			DplCoreDumpTag            int    `json:"dplCoreDumpTag"`
			DplCoreDumpNeedsRetrieved bool   `json:"dplCoreDumpNeedsRetrieved"`
			DplCoreDumpEnabled        bool   `json:"dplCoreDumpEnabled"`
			TimeStamp                 string `json:"timeStamp"`
		} `json:"dplCoreDumpData"`
		FcRVMState             string `json:"fcRVMState"`
		AsupEnabled            bool   `json:"asupEnabled"`
		AsupOnDemandEnabled    bool   `json:"asupOnDemandEnabled"`
		AsupRemoteDiagsEnabled bool   `json:"asupRemoteDiagsEnabled"`
		IocDumpData            struct {
			IocDumpEnabled        bool   `json:"iocDumpEnabled"`
			IocDumpNeedsRetrieved bool   `json:"iocDumpNeedsRetrieved"`
			IocDumpTag            int    `json:"iocDumpTag"`
			TimeStamp             string `json:"timeStamp"`
		} `json:"iocDumpData"`
		DriveChannelPortDisabled        bool   `json:"driveChannelPortDisabled"`
		DirectExpansionPortInUse        bool   `json:"directExpansionPortInUse"`
		HighResolutionStatisticsEnabled bool   `json:"highResolutionStatisticsEnabled"`
		AutoLoadBalancingEnabled        bool   `json:"autoLoadBalancingEnabled"`
		Reserved3                       string `json:"reserved3"`
		EmbeddedExternalKMSInformation struct {
			InstalledKMSCertificateFile []any `json:"installedKMSCertificateFile"`
			KmsEndpoint                 struct {
				KmsAddress struct {
					AddrType    string `json:"addrType"`
					DomainName  any    `json:"domainName"`
					IpvxAddress any    `json:"ipvxAddress"`
				} `json:"kmsAddress"`
				KmsPort int `json:"kmsPort"`
			} `json:"kmsEndpoint"`
			KmsEndpointList []any `json:"kmsEndpointList"`
		} `json:"embeddedExternalKMSInformation"`
		IsLun0RestrictedForDefaultPartition bool `json:"isLun0RestrictedForDefaultPartition"`
		SecuritySettings                    struct {
			CertificateRevocationSettings struct {
				CertificateRevocationCheckEnabled bool   `json:"certificateRevocationCheckEnabled"`
				OcspResponderAddress              string `json:"ocspResponderAddress"`
				IsOCSPResponderAddressSet         bool   `json:"isOCSPResponderAddressSet"`
			} `json:"certificateRevocationSettings"`
		} `json:"securitySettings"`
		ParityCheckOnMediaType struct {
			EnabledOnHDD bool `json:"enabledOnHDD"`
			EnabledOnSSD bool `json:"enabledOnSSD"`
		} `json:"parityCheckOnMediaType"`
		HostConnectivityReportingEnabled  bool `json:"hostConnectivityReportingEnabled"`
		ResourceProvisionedVolumesEnabled bool `json:"resourceProvisionedVolumesEnabled"`
		Vaaienabled                       bool `json:"vaaienabled"`
		Odxenabled                        bool `json:"odxenabled"`
	} `json:"sa"`
	Tray []struct {
		Type               string `json:"type"`
		Orientation        string `json:"orientation"`
		NumControllerSlots int    `json:"numControllerSlots"`
		NumDriveSlots      int    `json:"numDriveSlots"`
		TrayId             int    `json:"trayId"`
		TrayRef            string `json:"trayRef"`
		NonRedundantAccess bool   `json:"nonRedundantAccess"`
		PartNumber         string `json:"partNumber"`
		SerialNumber       string `json:"serialNumber"`
		VendorName         string `json:"vendorName"`
		ManufacturerDate   string `json:"manufacturerDate"`
		FruType            string `json:"fruType"`
		TrayIDMismatch     bool   `json:"trayIDMismatch"`
		TrayIDConflict     bool   `json:"trayIDConflict"`
		EsmVersionMismatch bool   `json:"esmVersionMismatch"`
		EsmMiswire         bool   `json:"esmMiswire"`
		DrvMHSpeedMismatch bool   `json:"drvMHSpeedMismatch"`
		UnsupportedTray    bool   `json:"unsupportedTray"`
		WorkingChannel     int    `json:"workingChannel"`
		MaxSpeed           string `json:"maxSpeed"`
		TrayTechnologyType string `json:"trayTechnologyType"`
		EsmGroupError      bool   `json:"esmGroupError"`
		UncertifiedTray    bool   `json:"uncertifiedTray"`
		LocateTray         bool   `json:"locateTray"`
		EsmHardwareMismatch bool  `json:"esmHardwareMismatch"`
		HasConfigurableTrayID bool `json:"hasConfigurableTrayID"`
		FrontEndInterfaceTechnology string   `json:"frontEndInterfaceTechnology"`
		DriveTechnologies           []string `json:"driveTechnologies"`
		NumDriveCompartments        int      `json:"numDriveCompartments"`
		NumDriveSlotsPerCompartment int      `json:"numDriveSlotsPerCompartment"`
		TrayAttributes              []struct {
			AttributeID    string `json:"attributeId"`
			AttributeValue string `json:"attributeValue"`
		} `json:"trayAttributes"`
		IsMisconfigured            bool `json:"isMisconfigured"`
		EsmFactoryDefaultsMismatch bool `json:"esmFactoryDefaultsMismatch"`
		PhysicalLocation           struct {
			TrayRef        string `json:"trayRef"`
			Slot           int    `json:"slot"`
			LocationParent struct {
				RefType        string `json:"refType"`
				ControllerRef  any    `json:"controllerRef"`
				SymbolRef      any    `json:"symbolRef"`
				TypedReference struct {
					ComponentType string `json:"componentType"`
					SymbolRef     string `json:"symbolRef"`
				} `json:"typedReference"`
			} `json:"locationParent"`
			LocationPosition int    `json:"locationPosition"`
			Label            string `json:"label"`
		} `json:"physicalLocation"`
		NumDrawers  int `json:"numDrawers"`
		DriveLayout struct {
			DriveOrientation   string `json:"driveOrientation"`
			NumRows            int    `json:"numRows"`
			NumColumns         int    `json:"numColumns"`
			PrimaryTraversal   string `json:"primaryTraversal"`
			SecondaryTraversal string `json:"secondaryTraversal"`
		} `json:"driveLayout"`
		FactoryDefaultsData struct {
			FactoryDefaultsVersion string `json:"factoryDefaultsVersion"`
			IsSupported            bool   `json:"isSupported"`
		} `json:"factoryDefaultsData"`
		LocateInProgress         bool   `json:"locateInProgress"`
		HasTrayIdentityIndicator bool   `json:"hasTrayIdentityIndicator"`
		OemPartNumber            string `json:"oemPartNumber"`
		TrayPositionIndex        int    `json:"trayPositionIndex"`
		Id                       string `json:"id"`
	} `json:"tray"`
	ComponentBundle struct {
		Fan []struct {
			FanRef           string `json:"fanRef"`
			Status           string `json:"status"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  any    `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference struct {
						ComponentType string `json:"componentType"`
						SymbolRef     string `json:"symbolRef"`
					} `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			Reserved1     string `json:"reserved1"`
			Reserved2     string `json:"reserved2"`
			RtrAttributes struct {
				CruType   string `json:"cruType"`
				ParentCru struct {
					Type                  string `json:"type"`
					ParentController      any    `json:"parentController"`
					ParentEsm             any    `json:"parentEsm"`
					ParentFan             any    `json:"parentFan"`
					ParentInterconnectCru any    `json:"parentInterconnectCru"`
					ParentSupportCru      string `json:"parentSupportCru"`
					ParentDrawer          any    `json:"parentDrawer"`
				} `json:"parentCru"`
				RtrAttributeData any `json:"rtrAttributeData"`
			} `json:"rtrAttributes"`
			RepairPolicy struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes any    `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			ID string `json:"id"`
		} `json:"fan"`
		Battery []struct {
			BatteryRef       string `json:"batteryRef"`
			Status           string `json:"status"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  string `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference any    `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			BatteryAge           int `json:"batteryAge"`
			BatteryLifeRemaining int `json:"batteryLifeRemaining"`
			BatteryTypeData      struct {
				BatteryType         string `json:"batteryType"`
				CruParentController any    `json:"cruParentController"`
				ParentController    string `json:"parentController"`
			} `json:"batteryTypeData"`
			Reserved1        string `json:"reserved1"`
			Reserved2        string `json:"reserved2"`
			ManufacturerDate string `json:"manufacturerDate"`
			VendorName       string `json:"vendorName"`
			VendorPN         string `json:"vendorPN"`
			VendorSN         string `json:"vendorSN"`
			FruType          string `json:"fruType"`
			RtrAttributes    struct {
				CruType          string `json:"cruType"`
				ParentCru        any    `json:"parentCru"`
				RtrAttributeData any    `json:"rtrAttributeData"`
			} `json:"rtrAttributes"`
			RepairPolicy struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes any    `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			BatteryCanExpire   bool `json:"batteryCanExpire"`
			AutomaticAgeReset bool `json:"automaticAgeReset"`
			LearnCycleData    struct {
				LastBatteryLearnCycle     string `json:"lastBatteryLearnCycle"`
				NextBatteryLearnCycle     string `json:"nextBatteryLearnCycle"`
				BatteryLearnCycleInterval int    `json:"batteryLearnCycleInterval"`
			} `json:"learnCycleData"`
			SmartBatteryData struct {
				LastBatteryLearnCycle     string `json:"lastBatteryLearnCycle"`
				NextBatteryLearnCycle     string `json:"nextBatteryLearnCycle"`
				BatteryLearnCycleInterval int    `json:"batteryLearnCycleInterval"`
			} `json:"smartBatteryData"`
			ID string `json:"id"`
		} `json:"battery"`
		PowerSupply []struct {
			PowerSupplyRef   string `json:"powerSupplyRef"`
			Status           string `json:"status"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  any    `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference struct {
						ComponentType string `json:"componentType"`
						SymbolRef     string `json:"symbolRef"`
					} `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			PartNumber       string `json:"partNumber"`
			SerialNumber     string `json:"serialNumber"`
			VendorName       string `json:"vendorName"`
			ManufacturerDate string `json:"manufacturerDate"`
			FruType          string `json:"fruType"`
			Reserved1        string `json:"reserved1"`
			Reserved2        string `json:"reserved2"`
			RtrAttributes    struct {
				CruType   string `json:"cruType"`
				ParentCru struct {
					Type                  string `json:"type"`
					ParentController      any    `json:"parentController"`
					ParentEsm             any    `json:"parentEsm"`
					ParentFan             any    `json:"parentFan"`
					ParentInterconnectCru any    `json:"parentInterconnectCru"`
					ParentSupportCru      string `json:"parentSupportCru"`
					ParentDrawer          any    `json:"parentDrawer"`
				} `json:"parentCru"`
				RtrAttributeData any `json:"rtrAttributeData"`
			} `json:"rtrAttributes"`
			RepairPolicy struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes any    `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			FirmwareRevision        string `json:"firmwareRevision"`
			ManufacturingDeviceCode int    `json:"manufacturingDeviceCode"`
			ID                      string `json:"id"`
		} `json:"powerSupply"`
		ThermalSensor []struct {
			ThermalSensorRef string `json:"thermalSensorRef"`
			Status           string `json:"status"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  string `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference any    `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			Reserved1     string `json:"reserved1"`
			Reserved2     string `json:"reserved2"`
			RtrAttributes struct {
				CruType   string `json:"cruType"`
				ParentCru struct {
					Type                  string `json:"type"`
					ParentController      string `json:"parentController"`
					ParentEsm             any    `json:"parentEsm"`
					ParentFan             any    `json:"parentFan"`
					ParentInterconnectCru any    `json:"parentInterconnectCru"`
					ParentSupportCru      string `json:"parentSupportCru"`
					ParentDrawer          any    `json:"parentDrawer"`
				} `json:"parentCru"`
				RtrAttributeData any `json:"rtrAttributeData"`
			} `json:"rtrAttributes"`
			RepairPolicy struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes any    `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			ID string `json:"id"`
		} `json:"thermalSensor"`
		Esm []struct {
			EsmRef           string `json:"esmRef"`
			Status           string `json:"status"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  any    `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference struct {
						ComponentType string `json:"componentType"`
						SymbolRef     string `json:"symbolRef"`
					} `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			NonRedundantAccess bool   `json:"nonRedundantAccess"`
			PartNumber         string `json:"partNumber"`
			SerialNumber       string `json:"serialNumber"`
			ManufacturerDate   string `json:"manufacturerDate"`
			Manufacturer       string `json:"manufacturer"`
			FruType            string `json:"fruType"`
			SoftwareVersion    string `json:"softwareVersion"`
			EsmInterfaceData   struct {
				IoInterfaceType string `json:"ioInterfaceType"`
				PortList        any    `json:"portList"`
			} `json:"esmInterfaceData"`
			ProductID      string `json:"productID"`
			WorkingChannel int    `json:"workingChannel"`
			CurrentSpeed   string `json:"currentSpeed"`
			MaxSpeed       string `json:"maxSpeed"`
			Reserved1      string `json:"reserved1"`
			Reserved2      string `json:"reserved2"`
			FibreEsm       struct {
				EsmPortRef string `json:"esmPortRef"`
				LoopID     int    `json:"loopID"`
				PortRef    string `json:"portRef"`
			} `json:"fibreEsm"`
			RtrAttributes struct {
				CruType          string `json:"cruType"`
				ParentCru        any    `json:"parentCru"`
				RtrAttributeData struct {
					HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
					ReadyToRemove             bool `json:"readyToRemove"`
				} `json:"rtrAttributeData"`
			} `json:"rtrAttributes"`
			EsmInterfaceAttributes struct {
				IoInterfaceType    string `json:"ioInterfaceType"`
				FibreEsmAttributes any    `json:"fibreEsmAttributes"`
				SasEsmAttributes   struct {
					SasExpander []string `json:"sasExpander"`
				} `json:"sasEsmAttributes"`
			} `json:"esmInterfaceAttributes"`
			BoardID             string `json:"boardID"`
			FactoryDefaultsData struct {
				IsSupported            bool   `json:"isSupported"`
				FactoryDefaultsVersion string `json:"factoryDefaultsVersion"`
			} `json:"factoryDefaultsData"`
			RepairPolicy struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes struct {
						HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
						ReadyToRemove             bool `json:"readyToRemove"`
					} `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			IsTrunkCapable           bool   `json:"isTrunkCapable"`
			TrunkMiswire             bool   `json:"trunkMiswire"`
			LocateInProgress         bool   `json:"locateInProgress"`
			HasTrayIdentityIndicator bool   `json:"hasTrayIdentityIndicator"`
			EsmType                  string `json:"esmType"`
			ID                       string `json:"id"`
		} `json:"esm"`
		Ups     []any `json:"ups"`
		Minihub []any `json:"minihub"`
		Gbic    []any `json:"gbic"`
		Sfp     []struct {
			SfpRef           string `json:"sfpRef"`
			Status           string `json:"status"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  string `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference any    `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			SfpType struct {
				SfpIdentType         string   `json:"sfpIdentType"`
				SfpConnectType       string   `json:"sfpConnectType"`
				SfpLinkLType         string   `json:"sfpLinkLType"`
				SfpTransmitterType   string   `json:"sfpTransmitterType"`
				SfpSpeed             []string `json:"sfpSpeed"`
				SfpTransmissionMedia []string `json:"sfpTransmissionMedia"`
				ManufacturerDate     string   `json:"manufacturerDate"`
				VendorName           string   `json:"vendorName"`
				VendorPN             string   `json:"vendorPN"`
				VendorRev            string   `json:"vendorRev"`
				VendorSN             string   `json:"vendorSN"`
				VendorOUI            string   `json:"vendorOUI"`
				Reserved1            string   `json:"reserved1"`
				Reserved2            string   `json:"reserved2"`
			} `json:"sfpType"`
			SfpPort    string `json:"sfpPort"`
			ParentData struct {
				SfpParentType string `json:"sfpParentType"`
				ControllerSFP struct {
					ControllerSFPType string `json:"controllerSFPType"`
					ParentController  string `json:"parentController"`
					Channel           int    `json:"channel"`
					Reserved1         string `json:"reserved1"`
					Reserved2         string `json:"reserved2"`
				} `json:"controllerSFP"`
				ParentEsm     any `json:"parentEsm"`
				ParentMinihub any `json:"parentMinihub"`
			} `json:"parentData"`
			Reserved1    string `json:"reserved1"`
			Reserved2    string `json:"reserved2"`
			SfpPortRef   string `json:"sfpPortRef"`
			RepairPolicy struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes struct {
						HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
						ReadyToRemove             bool `json:"readyToRemove"`
					} `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			ID string `json:"id"`
		} `json:"sfp"`
		InterconnectCRU []any `json:"interconnectCRU"`
		SupportCRU      []struct {
			SupportCRURef    string `json:"supportCRURef"`
			Status           string `json:"status"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  any    `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference struct {
						ComponentType string `json:"componentType"`
						SymbolRef     string `json:"symbolRef"`
					} `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			ManufacturerDate string `json:"manufacturerDate"`
			VendorName       string `json:"vendorName"`
			VendorPN         string `json:"vendorPN"`
			VendorSN         string `json:"vendorSN"`
			FruType          string `json:"fruType"`
			ReadyToRemove    bool   `json:"readyToRemove"`
			RtrAttributes    struct {
				CruType          string `json:"cruType"`
				ParentCru        any    `json:"parentCru"`
				RtrAttributeData struct {
					HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
					ReadyToRemove             bool `json:"readyToRemove"`
				} `json:"rtrAttributeData"`
			} `json:"rtrAttributes"`
			ConfiguredComponents []string `json:"configuredComponents"`
			Type                 string   `json:"type"`
			RepairPolicy         struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes struct {
						HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
						ReadyToRemove             bool `json:"readyToRemove"`
					} `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			ID string `json:"id"`
		} `json:"supportCRU"`
		Alarm     []any `json:"alarm"`
		HostBoard []struct {
			ParentController string `json:"parentController"`
			PartNumber       string `json:"partNumber"`
			SerialNumber     string `json:"serialNumber"`
			VendorName       string `json:"vendorName"`
			ManufacturerDate string `json:"manufacturerDate"`
			FruType          string `json:"fruType"`
			HostBoardID      string `json:"hostBoardId"`
			Status           string `json:"status"`
			Type             string `json:"type"`
			HostBoardRef     string `json:"hostBoardRef"`
			NumberOfPorts    int    `json:"numberOfPorts"`
			HbTypeData       struct {
				Type                     string `json:"type"`
				DualPortIbTypeData       any    `json:"dualPortIbTypeData"`
				DualPortDenaliIbTypeData any    `json:"dualPortDenaliIbTypeData"`
			} `json:"hbTypeData"`
			OemPartNumber           string `json:"oemPartNumber"`
			HostBoardControllerSlot int    `json:"hostBoardControllerSlot"`
			RepairPolicy            struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes any    `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  string `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference any    `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			ID string `json:"id"`
		} `json:"hostBoard"`
		SasExpander []struct {
			ExpanderRef string `json:"expanderRef"`
			Parent      struct {
				ExpanderParentType string `json:"expanderParentType"`
				ParentController   string `json:"parentController"`
				ParentEsm          any    `json:"parentEsm"`
				ParentDrawer       any    `json:"parentDrawer"`
			} `json:"parent"`
			Channel    int    `json:"channel"`
			DeviceName string `json:"deviceName"`
			VendorID   string `json:"vendorId"`
			ProductID  string `json:"productId"`
			FwVersion  string `json:"fwVersion"`
			ExpanderPorts []struct {
				Parent struct {
					Type         string `json:"type"`
					Controller   any    `json:"controller"`
					Drive        any    `json:"drive"`
					Expander     string `json:"expander"`
					HostBoardRef any    `json:"hostBoardRef"`
				} `json:"parent"`
				AttachedDevice struct {
					Channel             int    `json:"channel"`
					ChannelType         string `json:"channelType"`
					SasAttachedDeviceData struct {
						Type                  string `json:"type"`
						AlternateController   any    `json:"alternateController"`
						Drive                 string `json:"drive"`
						Expander              any    `json:"expander"`
						RemoteHostPortAddress any    `json:"remoteHostPortAddress"`
						LocalController       any    `json:"localController"`
						PhysicalLocation      any    `json:"physicalLocation"`
					} `json:"sasAttachedDeviceData"`
				} `json:"attachedDevice"`
				State          string `json:"state"`
				MiswireType    string `json:"miswireType"`
				ChannelPortRef string `json:"channelPortRef"`
				SasPhys        []struct {
					PhyIdentifier int  `json:"phyIdentifier"`
					IsOperational bool `json:"isOperational"`
				} `json:"sasPhys"`
				PortTypeData struct {
					PortType       string `json:"portType"`
					PortIdentifier any    `json:"portIdentifier"`
					RoutingType    string `json:"routingType"`
				} `json:"portTypeData"`
				PortMode                string   `json:"portMode"`
				DomainNumber            int      `json:"domainNumber"`
				AttachedChannelPortRef  string   `json:"attachedChannelPortRef"`
				DiscoveryStatus         int      `json:"discoveryStatus"`
				NonOptimalCablingType   string   `json:"nonOptimalCablingType"`
				ChannelPortRefs         []string `json:"channelPortRefs"`
				AttachedChannelPortRefs []string `json:"attachedChannelPortRefs"`
			} `json:"expanderPorts"`
			DomainNumber int    `json:"domainNumber"`
			ID           string `json:"id"`
		} `json:"sasExpander"`
		CacheBackupDevice []struct {
			BackupDeviceRef            string `json:"backupDeviceRef"`
			BackupDeviceControllerSlot int    `json:"backupDeviceControllerSlot"`
			ParentController           string `json:"parentController"`
			BackupDeviceType           string `json:"backupDeviceType"`
			BackupDeviceStatus         string `json:"backupDeviceStatus"`
			BackupDeviceVpd            struct {
				Manufacturer    string `json:"manufacturer"`
				ManufactureDate string `json:"manufactureDate"`
				ProductID       string `json:"productId"`
				ProductRevLevel string `json:"productRevLevel"`
				PartNumber      string `json:"partNumber"`
				SerialNumber    string `json:"serialNumber"`
			} `json:"backupDeviceVpd"`
			BackupDeviceCapacity int `json:"backupDeviceCapacity"`
			RepairPolicy         struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes any    `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  string `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference any    `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			CacheBackupDeviceWearLife struct {
				CacheBackupDeviceWearLifeReportingLevel string `json:"cacheBackupDeviceWearLifeReportingLevel"`
				PercentEnduranceUsed                      int    `json:"percentEnduranceUsed"`
				CacheBackupDeviceWearLifeStats            struct {
					PercentEnduranceUsed      int `json:"percentEnduranceUsed"`
					PercentEnduranceRemaining int `json:"percentEnduranceRemaining"`
					AverageEraseCount         int `json:"averageEraseCount"`
				} `json:"cacheBackupDeviceWearLifeStats"`
			} `json:"cacheBackupDeviceWearLife"`
			ID string `json:"id"`
		} `json:"cacheBackupDevice"`
		CacheMemoryDimm []struct {
			CacheMemoryDimmRef     string `json:"cacheMemoryDimmRef"`
			Status                 string `json:"status"`
			CapacityInMegabytes    int    `json:"capacityInMegabytes"`
			SerialNumber           string `json:"serialNumber"`
			PartNumber             string `json:"partNumber"`
			OemPartNumber          string `json:"oemPartNumber"`
			ManufacturerPartNumber string `json:"manufacturerPartNumber"`
			Manufacturer           string `json:"manufacturer"`
			ManufactureDate        string `json:"manufactureDate"`
			RepairPolicy           struct {
				RemovalData struct {
					RemovalMethod string `json:"removalMethod"`
					RtrAttributes any    `json:"rtrAttributes"`
				} `json:"removalData"`
				ReplacementMethod string `json:"replacementMethod"`
			} `json:"repairPolicy"`
			PhysicalLocation struct {
				TrayRef        string `json:"trayRef"`
				Slot           int    `json:"slot"`
				LocationParent struct {
					RefType        string `json:"refType"`
					ControllerRef  string `json:"controllerRef"`
					SymbolRef      any    `json:"symbolRef"`
					TypedReference any    `json:"typedReference"`
				} `json:"locationParent"`
				LocationPosition int    `json:"locationPosition"`
				Label            string `json:"label"`
			} `json:"physicalLocation"`
			Shared       bool   `json:"shared"`
			RevisionCode string `json:"revisionCode"`
		} `json:"cacheMemoryDimm"`
		ProcessorMemoryDimm []any `json:"processorMemoryDimm"`
		Drawer              []any `json:"drawer"`
	} `json:"componentBundle"`
	Controller []struct {
		Active           bool   `json:"active"`
		Quiesced         bool   `json:"quiesced"`
		Status           string `json:"status"`
		ControllerRef    string `json:"controllerRef"`
		PhysicalLocation struct {
			TrayRef        string `json:"trayRef"`
			Slot           int    `json:"slot"`
			LocationParent struct {
				RefType        string `json:"refType"`
				ControllerRef  any    `json:"controllerRef"`
				SymbolRef      string `json:"symbolRef"`
				TypedReference any    `json:"typedReference"`
			} `json:"locationParent"`
			LocationPosition int    `json:"locationPosition"`
			Label            string `json:"label"`
		} `json:"physicalLocation"`
		Manufacturer        string `json:"manufacturer"`
		ManufacturerDate    string `json:"manufacturerDate"`
		AppVersion          string `json:"appVersion"`
		BootVersion         string `json:"bootVersion"`
		ProductID           string `json:"productID"`
		ProductRevLevel     string `json:"productRevLevel"`
		SerialNumber        string `json:"serialNumber"`
		BoardID             string `json:"boardID"`
		CacheMemorySize     int    `json:"cacheMemorySize"`
		ProcessorMemorySize int    `json:"processorMemorySize"`
		HostInterfaces      []struct {
			InterfaceType string `json:"interfaceType"`
			Fibre         any    `json:"fibre"`
			Ib            any    `json:"ib"`
			Iscsi         struct {
				Channel        int    `json:"channel"`
				ChannelPortRef string `json:"channelPortRef"`
				TcpListenPort  int    `json:"tcpListenPort"`
				Ipv4Enabled    bool   `json:"ipv4Enabled"`
				Ipv4Data       struct {
					Ipv4Address             string `json:"ipv4Address"`
					Ipv4AddressConfigMethod string `json:"ipv4AddressConfigMethod"`
					Ipv4OutboundPacketPriority struct {
						IsEnabled bool `json:"isEnabled"`
						Value     int  `json:"value"`
					} `json:"ipv4OutboundPacketPriority"`
					Ipv4VlanId struct {
						IsEnabled bool `json:"isEnabled"`
						Value     int  `json:"value"`
					} `json:"ipv4VlanId"`
					Ipv4AddressData struct {
						ConfigState        string `json:"configState"`
						Ipv4Address        string `json:"ipv4Address"`
						Ipv4SubnetMask     string `json:"ipv4SubnetMask"`
						Ipv4GatewayAddress string `json:"ipv4GatewayAddress"`
					} `json:"ipv4AddressData"`
				} `json:"ipv4Data"`
				InterfaceData struct {
					Type         string `json:"type"`
					EthernetData struct {
						PartData struct {
							VendorName     string `json:"vendorName"`
							PartNumber     string `json:"partNumber"`
							RevisionNumber string `json:"revisionNumber"`
							SerialNumber   string `json:"serialNumber"`
						} `json:"partData"`
						MacAddress                    string   `json:"macAddress"`
						FullDuplex                    bool     `json:"fullDuplex"`
						MaximumFramePayloadSize       int      `json:"maximumFramePayloadSize"`
						CurrentInterfaceSpeed         string   `json:"currentInterfaceSpeed"`
						MaximumInterfaceSpeed         string   `json:"maximumInterfaceSpeed"`
						LinkStatus                    string   `json:"linkStatus"`
						SupportedInterfaceSpeeds      []string `json:"supportedInterfaceSpeeds"`
						AutoconfigSupport             bool     `json:"autoconfigSupport"`
						CopperCableDiagnosticsSupport bool     `json:"copperCableDiagnosticsSupport"`
						OperationalInterfaceSpeed     string   `json:"operationalInterfaceSpeed"`
						SupportedFECModes             []string `json:"supportedFECModes"`
						ConfiguredFECMode             string   `json:"configuredFECMode"`
						CurrentFECMode                string   `json:"currentFECMode"`
					} `json:"ethernetData"`
					InfinibandData any `json:"infinibandData"`
				} `json:"interfaceData"`
				InterfaceRef string `json:"interfaceRef"`
				Ipv6Enabled  bool   `json:"ipv6Enabled"`
				Ipv6Data     struct {
					Ipv6LocalAddresses []struct {
						Address      string `json:"address"`
						AddressState struct {
							AddressType           string `json:"addressType"`
							InterfaceAddressState string `json:"interfaceAddressState"`
							RouterAddressState    string `json:"routerAddressState"`
						} `json:"addressState"`
					} `json:"ipv6LocalAddresses"`
					Ipv6RoutableAddresses []struct {
						Address      string `json:"address"`
						AddressState struct {
							AddressType           string `json:"addressType"`
							InterfaceAddressState string `json:"interfaceAddressState"`
							RouterAddressState    string `json:"routerAddressState"`
						} `json:"addressState"`
					} `json:"ipv6RoutableAddresses"`
					Ipv6PortRouterAddress struct {
						Address      string `json:"address"`
						AddressState struct {
							AddressType           string `json:"addressType"`
							InterfaceAddressState string `json:"interfaceAddressState"`
							RouterAddressState    string `json:"routerAddressState"`
						} `json:"addressState"`
					} `json:"ipv6PortRouterAddress"`
					Ipv6AddressConfigMethod string `json:"ipv6AddressConfigMethod"`
					Ipv6OutboundPacketPriority struct {
						IsEnabled bool `json:"isEnabled"`
						Value     int  `json:"value"`
					} `json:"ipv6OutboundPacketPriority"`
					Ipv6VlanId struct {
						IsEnabled bool `json:"isEnabled"`
						Value     int  `json:"value"`
					} `json:"ipv6VlanId"`
					Ipv6HopLimit                            int `json:"ipv6HopLimit"`
					Ipv6NdReachableTime                     int `json:"ipv6NdReachableTime"`
					Ipv6NdRetransmitTime                    int `json:"ipv6NdRetransmitTime"`
					Ipv6NdStaleTimeout                      int `json:"ipv6NdStaleTimeout"`
					Ipv6DuplicateAddressDetectionAttempts int `json:"ipv6DuplicateAddressDetectionAttempts"`
				} `json:"ipv6Data"`
				PhysicalLocation struct {
					TrayRef        string `json:"trayRef"`
					Slot           int    `json:"slot"`
					LocationParent struct {
						RefType        string `json:"refType"`
						ControllerRef  any    `json:"controllerRef"`
						SymbolRef      string `json:"symbolRef"`
						TypedReference any    `json:"typedReference"`
					} `json:"locationParent"`
					LocationPosition int    `json:"locationPosition"`
					Label            string `json:"label"`
				} `json:"physicalLocation"`
				ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
				IsIPv6Capable                bool   `json:"isIPv6Capable"`
				OneWayMaxRate                string `json:"oneWayMaxRate"`
				BidirectionalMaxRate         string `json:"bidirectionalMaxRate"`
				Iqn                          string `json:"iqn"`
				ControllerID                 string `json:"controllerId"`
				InterfaceID                  string `json:"interfaceId"`
				AddressID                    string `json:"addressId"`
				NiceAddressID                string `json:"niceAddressId"`
				ID                           string `json:"id"`
			} `json:"iscsi"`
			Sas                any `json:"sas"`
			Sata               any `json:"sata"`
			Scsi               any `json:"scsi"`
			CouplingDriverNvme any `json:"couplingDriverNvme"`
			Ethernet           any `json:"ethernet"`
			Pcie               any `json:"pcie"`
		} `json:"hostInterfaces"`
		DriveInterfaces []struct {
			InterfaceType string `json:"interfaceType"`
			Fibre         any    `json:"fibre"`
			Ib            any    `json:"ib"`
			Iscsi         any    `json:"iscsi"`
			Sas           struct {
				Channel               int    `json:"channel"`
				CurrentInterfaceSpeed string `json:"currentInterfaceSpeed"`
				MaximumInterfaceSpeed string `json:"maximumInterfaceSpeed"`
				Part                  string `json:"part"`
				Revision              int    `json:"revision"`
				IsDegraded            bool   `json:"isDegraded"`
				IocPort               struct {
					Parent struct {
						Type         string `json:"type"`
						Controller   string `json:"controller"`
						Drive        any    `json:"drive"`
						Expander     any    `json:"expander"`
						HostBoardRef any    `json:"hostBoardRef"`
					} `json:"parent"`
					AttachedDevice struct {
						Channel               int    `json:"channel"`
						ChannelType           string `json:"channelType"`
						SasAttachedDeviceData struct {
							Type                  string `json:"type"`
							AlternateController   any    `json:"alternateController"`
							Drive                 any    `json:"drive"`
							Expander              string `json:"expander"`
							RemoteHostPortAddress any    `json:"remoteHostPortAddress"`
							LocalController       any    `json:"localController"`
							PhysicalLocation      any    `json:"physicalLocation"`
						} `json:"sasAttachedDeviceData"`
					} `json:"attachedDevice"`
					State          string `json:"state"`
					MiswireType    string `json:"miswireType"`
					ChannelPortRef string `json:"channelPortRef"`
					SasPhys        []struct {
						PhyIdentifier int  `json:"phyIdentifier"`
						IsOperational bool `json:"isOperational"`
					} `json:"sasPhys"`
					PortTypeData struct {
						PortType       string `json:"portType"`
						PortIdentifier string `json:"portIdentifier"`
						RoutingType    string `json:"routingType"`
					} `json:"portTypeData"`
					PortMode                string   `json:"portMode"`
					DomainNumber            int      `json:"domainNumber"`
					AttachedChannelPortRef  string   `json:"attachedChannelPortRef"`
					DiscoveryStatus         int      `json:"discoveryStatus"`
					NonOptimalCablingType   string   `json:"nonOptimalCablingType"`
					ChannelPortRefs         []string `json:"channelPortRefs"`
					AttachedChannelPortRefs []string `json:"attachedChannelPortRefs"`
				} `json:"iocPort"`
				InterfaceRef string `json:"interfaceRef"`
				PhysicalLocation struct {
					TrayRef        string `json:"trayRef"`
					Slot           int    `json:"slot"`
					LocationParent struct {
						RefType        string `json:"refType"`
						ControllerRef  any    `json:"controllerRef"`
						SymbolRef      string `json:"symbolRef"`
						TypedReference any    `json:"typedReference"`
					} `json:"locationParent"`
					LocationPosition int    `json:"locationPosition"`
					Label            string `json:"label"`
				} `json:"physicalLocation"`
				ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
				OneWayMaxRate                string `json:"oneWayMaxRate"`
				BidirectionalMaxRate         string `json:"bidirectionalMaxRate"`
				ControllerID                 any    `json:"controllerId"`
				InterfaceID                  string `json:"interfaceId"`
				AddressID                    string `json:"addressId"`
				NiceAddressID                string `json:"niceAddressId"`
				BasePortAddress              string `json:"basePortAddress"`
				ID                           string `json:"id"`
			} `json:"sas"`
			Sata               any `json:"sata"`
			Scsi               any `json:"scsi"`
			CouplingDriverNvme any `json:"couplingDriverNvme"`
			Ethernet           any `json:"ethernet"`
			Pcie               any `json:"pcie"`
		} `json:"driveInterfaces"`
		NetInterfaces []struct {
			InterfaceType string `json:"interfaceType"`
			Ethernet      struct {
				InterfaceName            string `json:"interfaceName"`
				Channel                  int    `json:"channel"`
				Speed                    int    `json:"speed"`
				IP                       int    `json:"ip"`
				Alias                    string `json:"alias"`
				MacAddr                  string `json:"macAddr"`
				// GatewayIP - ИСПРАВЛЕНО 2026-09-17 по реальной ошибке при
				// синхронизации hardware Lenovo/SANtricity через
				// tools/storagesync - "cannot unmarshal number into Go
				// struct field .controller.netInterfaces.ethernet.gatewayIp
				// of type string": реальное устройство отдаёт gatewayIp
				// числом (как и соседнее поле IP чуть выше), а не строкой -
				// изначально было заведено как string по аналогии с
				// остальными адресными полями (Ipv4Address и т.п.), которые
				// у этого же устройства оказались строками корректно.
				GatewayIP                int    `json:"gatewayIp"`
				SubnetMask               int    `json:"subnetMask"`
				BootUsed                 bool   `json:"bootUsed"`
				RloginEnabled            bool   `json:"rloginEnabled"`
				Reserved1                string `json:"reserved1"`
				SetupError               bool   `json:"setupError"`
				Reserved2                string `json:"reserved2"`
				InterfaceRef             string `json:"interfaceRef"`
				LinkStatus               string `json:"linkStatus"`
				Ipv4Enabled              bool   `json:"ipv4Enabled"`
				Ipv4Address              string `json:"ipv4Address"`
				Ipv4SubnetMask           string `json:"ipv4SubnetMask"`
				Ipv4AddressConfigMethod  string `json:"ipv4AddressConfigMethod"`
				Ipv6Enabled              bool   `json:"ipv6Enabled"`
				Ipv6LocalAddress         struct {
					Address      string `json:"address"`
					AddressState struct {
						AddressType           string `json:"addressType"`
						InterfaceAddressState string `json:"interfaceAddressState"`
						RouterAddressState    string `json:"routerAddressState"`
					} `json:"addressState"`
				} `json:"ipv6LocalAddress"`
				Ipv6PortStaticRoutableAddress struct {
					Address      string `json:"address"`
					AddressState struct {
						AddressType           string `json:"addressType"`
						InterfaceAddressState string `json:"interfaceAddressState"`
						RouterAddressState    string `json:"routerAddressState"`
					} `json:"addressState"`
				} `json:"ipv6PortStaticRoutableAddress"`
				Ipv6PortRoutableAddresses []any  `json:"ipv6PortRoutableAddresses"`
				Ipv6AddressConfigMethod   string `json:"ipv6AddressConfigMethod"`
				FullDuplex                bool   `json:"fullDuplex"`
				SupportedSpeedSettings    []string `json:"supportedSpeedSettings"`
				ConfiguredSpeedSetting    string   `json:"configuredSpeedSetting"`
				CurrentSpeed              string   `json:"currentSpeed"`
				PhysicalLocation          struct {
					TrayRef        string `json:"trayRef"`
					Slot           int    `json:"slot"`
					LocationParent struct {
						RefType        string `json:"refType"`
						ControllerRef  string `json:"controllerRef"`
						SymbolRef      any    `json:"symbolRef"`
						TypedReference any    `json:"typedReference"`
					} `json:"locationParent"`
					LocationPosition int    `json:"locationPosition"`
					Label            string `json:"label"`
				} `json:"physicalLocation"`
				Ipv4GatewayAddress string `json:"ipv4GatewayAddress"`
				ControllerRef      string `json:"controllerRef"`
				ControllerSlot     int    `json:"controllerSlot"`
				DNSProperties      struct {
					AcquisitionProperties struct {
						DNSAcquisitionType string `json:"dnsAcquisitionType"`
						DNSServers         []struct {
							AddressType string `json:"addressType"`
							Ipv4Address string `json:"ipv4Address"`
							Ipv6Address any    `json:"ipv6Address"`
						} `json:"dnsServers"`
					} `json:"acquisitionProperties"`
					DhcpAcquiredDNSServers []any `json:"dhcpAcquiredDnsServers"`
				} `json:"dnsProperties"`
				NtpProperties struct {
					AcquisitionProperties struct {
						NtpAcquisitionType string `json:"ntpAcquisitionType"`
						NtpServers         []struct {
							AddrType     string `json:"addrType"`
							DomainName   string `json:"domainName"`
							IpvxAddress  any    `json:"ipvxAddress"`
						} `json:"ntpServers"`
					} `json:"acquisitionProperties"`
					DhcpAcquiredNtpServers []any `json:"dhcpAcquiredNtpServers"`
				} `json:"ntpProperties"`
				ID string `json:"id"`
			} `json:"ethernet"`
		} `json:"netInterfaces"`
		Inventory                 []any  `json:"inventory"`
		Reserved1                 string `json:"reserved1"`
		Reserved2                 string `json:"reserved2"`
		HostBoardID               string `json:"hostBoardID"`
		PhysicalCacheMemorySize   int    `json:"physicalCacheMemorySize"`
		ReadyToRemove             bool   `json:"readyToRemove"`
		BoardSubmodelID           string `json:"boardSubmodelID"`
		SubmodelSupported         bool   `json:"submodelSupported"`
		OemPartNumber             string `json:"oemPartNumber"`
		PartNumber                string `json:"partNumber"`
		RtrAttributes             struct {
			CruType          string `json:"cruType"`
			ParentCru        any    `json:"parentCru"`
			RtrAttributeData struct {
				HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
				ReadyToRemove             bool `json:"readyToRemove"`
			} `json:"rtrAttributeData"`
		} `json:"rtrAttributes"`
		BootTime        string `json:"bootTime"`
		ModelName       string `json:"modelName"`
		NetworkSettings struct {
			Ipv4DefaultRouterAddress string `json:"ipv4DefaultRouterAddress"`
			Ipv6DefaultRouterAddress struct {
				Address      string `json:"address"`
				AddressState struct {
					AddressType           string `json:"addressType"`
					InterfaceAddressState string `json:"interfaceAddressState"`
					RouterAddressState    string `json:"routerAddressState"`
				} `json:"addressState"`
			} `json:"ipv6DefaultRouterAddress"`
			Ipv6CandidateDefaultRouterAddresses []any `json:"ipv6CandidateDefaultRouterAddresses"`
			RemoteAccessEnabled                 bool  `json:"remoteAccessEnabled"`
			DNSProperties                       struct {
				AcquisitionProperties struct {
					DNSAcquisitionType string `json:"dnsAcquisitionType"`
					DNSServers         []struct {
						AddressType string `json:"addressType"`
						Ipv4Address string `json:"ipv4Address"`
						Ipv6Address any    `json:"ipv6Address"`
					} `json:"dnsServers"`
				} `json:"acquisitionProperties"`
				DhcpAcquiredDnsServers []any `json:"dhcpAcquiredDnsServers"`
			} `json:"dnsProperties"`
			NtpProperties struct {
				AcquisitionProperties struct {
					NtpAcquisitionType string `json:"ntpAcquisitionType"`
					NtpServers         []struct {
						AddrType    string `json:"addrType"`
						DomainName  string `json:"domainName"`
						IpvxAddress any    `json:"ipvxAddress"`
					} `json:"ntpServers"`
				} `json:"acquisitionProperties"`
				DhcpAcquiredNtpServers []any `json:"dhcpAcquiredNtpServers"`
			} `json:"ntpProperties"`
		} `json:"networkSettings"`
		RepairPolicy struct {
			RemovalData struct {
				RemovalMethod string `json:"removalMethod"`
				RtrAttributes any    `json:"rtrAttributes"`
			} `json:"removalData"`
			ReplacementMethod string `json:"replacementMethod"`
		} `json:"repairPolicy"`
		FlashCacheMemorySize int `json:"flashCacheMemorySize"`
		CtrlIocDumpData      struct {
			IocDumpNeedsRetrieved bool   `json:"iocDumpNeedsRetrieved"`
			IocDumpTag            int    `json:"iocDumpTag"`
			TimeStamp             string `json:"timeStamp"`
		} `json:"ctrlIocDumpData"`
		LocateInProgress         bool `json:"locateInProgress"`
		HasTrayIdentityIndicator bool `json:"hasTrayIdentityIndicator"`
		ControllerErrorMode      string `json:"controllerErrorMode"`
		CodeVersions             []struct {
			CodeModule    string `json:"codeModule"`
			VersionString string `json:"versionString"`
		} `json:"codeVersions"`
		ContainerServicesConfiguration struct {
			ContainerServicesCapable bool `json:"containerServicesCapable"`
			ContainerServicesEnabled bool `json:"containerServicesEnabled"`
		} `json:"containerServicesConfiguration"`
		ID string `json:"id"`
	} `json:"controller"`
	Drive []struct {
		Offline          bool   `json:"offline"`
		HotSpare         bool   `json:"hotSpare"`
		InvalidDriveData bool   `json:"invalidDriveData"`
		Available        bool   `json:"available"`
		Pfa              bool   `json:"pfa"`
		DriveRef         string `json:"driveRef"`
		Status           string `json:"status"`
		Cause            string `json:"cause"`
		InterfaceType    struct {
			DriveType string `json:"driveType"`
			Fibre     any    `json:"fibre"`
			Sas       struct {
				DeviceName string `json:"deviceName"`
				DrivePortAddresses []struct {
					Channel        int    `json:"channel"`
					PortIdentifier string `json:"portIdentifier"`
				} `json:"drivePortAddresses"`
			} `json:"sas"`
			Scsi any `json:"scsi"`
			Nvme any `json:"nvme"`
		} `json:"interfaceType"`
		PhysicalLocation struct {
			TrayRef        string `json:"trayRef"`
			Slot           int    `json:"slot"`
			LocationParent struct {
				RefType        string `json:"refType"`
				ControllerRef  any    `json:"controllerRef"`
				SymbolRef      any    `json:"symbolRef"`
				TypedReference struct {
					ComponentType string `json:"componentType"`
					SymbolRef     string `json:"symbolRef"`
				} `json:"typedReference"`
			} `json:"locationParent"`
			LocationPosition int    `json:"locationPosition"`
			Label            string `json:"label"`
		} `json:"physicalLocation"`
		Manufacturer            string `json:"manufacturer"`
		ManufacturerDate        string `json:"manufacturerDate"`
		ProductID               string `json:"productID"`
		SerialNumber            string `json:"serialNumber"`
		SoftwareVersion         string `json:"softwareVersion"`
		BlkSize                 int    `json:"blkSize"`
		UsableCapacity          string `json:"usableCapacity"`
		RawCapacity             string `json:"rawCapacity"`
		WorldWideName           string `json:"worldWideName"`
		CurrentVolumeGroupRef   string `json:"currentVolumeGroupRef"`
		SparedForDriveRef       string `json:"sparedForDriveRef"`
		MirrorDrive             string `json:"mirrorDrive"`
		NonRedundantAccess      bool   `json:"nonRedundantAccess"`
		WorkingChannel          int    `json:"workingChannel"`
		VolumeGroupIndex        int    `json:"volumeGroupIndex"`
		CurrentSpeed            string `json:"currentSpeed"`
		MaxSpeed                string `json:"maxSpeed"`
		Uncertified             bool   `json:"uncertified"`
		HasDegradedChannel      bool   `json:"hasDegradedChannel"`
		DegradedChannels        []any  `json:"degradedChannels"`
		PhyDriveType            string `json:"phyDriveType"`
		SpindleSpeed            int    `json:"spindleSpeed"`
		RtrAttributes           struct {
			CruType          string `json:"cruType"`
			ParentCru        any    `json:"parentCru"`
			RtrAttributeData struct {
				HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
				ReadyToRemove             bool `json:"readyToRemove"`
			} `json:"rtrAttributeData"`
		} `json:"rtrAttributes"`
		Reserved         string `json:"reserved"`
		PhyDriveTypeData struct {
			PhyDriveType          string `json:"phyDriveType"`
			SataDriveAttributes   any    `json:"sataDriveAttributes"`
		} `json:"phyDriveTypeData"`
		PfaReason    string `json:"pfaReason"`
		BypassSource []any  `json:"bypassSource"`
		RepairPolicy struct {
			RemovalData struct {
				RemovalMethod string `json:"removalMethod"`
				RtrAttributes struct {
					HasReadyToRemoveIndicator bool `json:"hasReadyToRemoveIndicator"`
					ReadyToRemove             bool `json:"readyToRemove"`
				} `json:"rtrAttributes"`
			} `json:"removalData"`
			ReplacementMethod string `json:"replacementMethod"`
		} `json:"repairPolicy"`
		FdeCapable  bool   `json:"fdeCapable"`
		FdeEnabled  bool   `json:"fdeEnabled"`
		FdeLocked   bool   `json:"fdeLocked"`
		LockKeyID   string `json:"lockKeyID"`
		SsdWearLife struct {
			AverageEraseCountPercent      int  `json:"averageEraseCountPercent"`
			SpareBlocksRemainingPercent   int  `json:"spareBlocksRemainingPercent"`
			IsWearLifeMonitoringSupported bool `json:"isWearLifeMonitoringSupported"`
			PercentEnduranceUsed          int  `json:"percentEnduranceUsed"`
		} `json:"ssdWearLife"`
		DriveMediaType string `json:"driveMediaType"`
		FpgaVersion    string `json:"fpgaVersion"`
		ProtectionInformationCapabilities struct {
			ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
			ProtectionType               string `json:"protectionType"`
		} `json:"protectionInformationCapabilities"`
		ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
		ProtectionType               string `json:"protectionType"`
		InterposerPresent            bool   `json:"interposerPresent"`
		InterposerRef                string `json:"interposerRef"`
		CurrentCommandAgingTimeout   int    `json:"currentCommandAgingTimeout"`
		DefaultCommandAgingTimeout   int    `json:"defaultCommandAgingTimeout"`
		DriveTemperature             struct {
			CurrentTemp int `json:"currentTemp"`
			RefTemp     int `json:"refTemp"`
		} `json:"driveTemperature"`
		BlkSizePhysical  int    `json:"blkSizePhysical"`
		LowestAlignedLBA string `json:"lowestAlignedLBA"`
		Removed          bool   `json:"removed"`
		LocateInProgress bool   `json:"locateInProgress"`
		FipsCapable      bool   `json:"fipsCapable"`
		DulbeCapable     bool   `json:"dulbeCapable"`
		SanitizeCapable  bool   `json:"sanitizeCapable"`
		FirmwareVersion  string `json:"firmwareVersion"`
		LockKeyIDValue   any    `json:"lockKeyIDValue"`
		ID               string `json:"id"`
	} `json:"drive"`
	VolumeGroup []struct {
		SequenceNum      int    `json:"sequenceNum"`
		Offline          bool   `json:"offline"`
		RaidLevel        string `json:"raidLevel"`
		WorldWideName    string `json:"worldWideName"`
		VolumeGroupRef   string `json:"volumeGroupRef"`
		Reserved1        string `json:"reserved1"`
		Reserved2        string `json:"reserved2"`
		TrayLossProtection bool `json:"trayLossProtection"`
		Label            string `json:"label"`
		State            string `json:"state"`
		SpindleSpeedMatch string `json:"spindleSpeedMatch"`
		SpindleSpeed     int    `json:"spindleSpeed"`
		IsInaccessible   bool   `json:"isInaccessible"`
		SecurityType     string `json:"securityType"`
		DrawerLossProtection bool `json:"drawerLossProtection"`
		ProtectionInformationCapable bool `json:"protectionInformationCapable"`
		ProtectionInformationCapabilities struct {
			ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
			ProtectionType               string `json:"protectionType"`
		} `json:"protectionInformationCapabilities"`
		VolumeGroupData struct {
			Type string `json:"type"`
			DiskPoolData struct {
				ReconstructionReservedDriveCount        int    `json:"reconstructionReservedDriveCount"`
				ReconstructionReservedAmt             string `json:"reconstructionReservedAmt"`
				ReconstructionReservedDriveCountCurrent int    `json:"reconstructionReservedDriveCountCurrent"`
				PoolUtilizationWarningThreshold         int    `json:"poolUtilizationWarningThreshold"`
				PoolUtilizationCriticalThreshold        int    `json:"poolUtilizationCriticalThreshold"`
				PoolUtilizationState                    string `json:"poolUtilizationState"`
				UnusableCapacity                        string `json:"unusableCapacity"`
				DegradedReconstructPriority             string `json:"degradedReconstructPriority"`
				CriticalReconstructPriority             string `json:"criticalReconstructPriority"`
				BackgroundOperationPriority             string `json:"backgroundOperationPriority"`
				AllocGranularity                        string `json:"allocGranularity"`
				MinimumDriveCount                       int    `json:"minimumDriveCount"`
			} `json:"diskPoolData"`
		} `json:"volumeGroupData"`
		Usage                    string `json:"usage"`
		DriveBlockFormat         string `json:"driveBlockFormat"`
		ReservedSpaceAllocated   bool   `json:"reservedSpaceAllocated"`
		SecurityLevel            string `json:"securityLevel"`
		DulbeEnabled             bool   `json:"dulbeEnabled"`
		BlkSizeSupported         []int  `json:"blkSizeSupported"`
		BlkSizeRecommended       int    `json:"blkSizeRecommended"`
		UsedSpace                string `json:"usedSpace"`
		TotalRaidedSpace         string `json:"totalRaidedSpace"`
		Extents                  []struct {
			SectorOffset    string `json:"sectorOffset"`
			RawCapacity     string `json:"rawCapacity"`
			RaidLevel       string `json:"raidLevel"`
			VolumeGroupRef  string `json:"volumeGroupRef"`
			FreeExtentRef   string `json:"freeExtentRef"`
			Reserved1       string `json:"reserved1"`
			Reserved2       string `json:"reserved2"`
			DdpRAIDCapacities []struct {
				DdpVolRAIDLevel string `json:"ddpVolRAIDLevel"`
				UsableCapacity  string `json:"usableCapacity"`
			} `json:"ddpRAIDCapacities"`
		} `json:"extents"`
		LargestFreeExtentSize  string `json:"largestFreeExtentSize"`
		RaidStatus             string `json:"raidStatus"`
		FreeSpace              string `json:"freeSpace"`
		DrivePhysicalType      string `json:"drivePhysicalType"`
		DriveMediaType         string `json:"driveMediaType"`
		NormalizedSpindleSpeed string `json:"normalizedSpindleSpeed"`
		DiskPool               bool   `json:"diskPool"`
		ID                     string `json:"id"`
		Name                   string `json:"name"`
	} `json:"volumeGroup"`
	Volume []struct {
		Offline           bool   `json:"offline"`
		ExtremeProtection bool   `json:"extremeProtection"`
		VolumeHandle      int    `json:"volumeHandle"`
		RaidLevel         string `json:"raidLevel"`
		SectorOffset      string `json:"sectorOffset"`
		WorldWideName     string `json:"worldWideName"`
		Label             string `json:"label"`
		BlkSize           int    `json:"blkSize"`
		Capacity          string `json:"capacity"`
		ReconPriority     int    `json:"reconPriority"`
		SegmentSize       int    `json:"segmentSize"`
		Action            string `json:"action"`
		Cache             struct {
			Cwob                bool   `json:"cwob"`
			EnterpriseCacheDump bool   `json:"enterpriseCacheDump"`
			MirrorActive        bool   `json:"mirrorActive"`
			MirrorEnable        bool   `json:"mirrorEnable"`
			ReadCacheActive     bool   `json:"readCacheActive"`
			ReadCacheEnable     bool   `json:"readCacheEnable"`
			WriteCacheActive    bool   `json:"writeCacheActive"`
			WriteCacheEnable    bool   `json:"writeCacheEnable"`
			CacheFlushModifier  string `json:"cacheFlushModifier"`
			ReadAheadMultiplier int    `json:"readAheadMultiplier"`
		} `json:"cache"`
		MediaScan struct {
			Enable                 bool `json:"enable"`
			ParityValidationEnable bool `json:"parityValidationEnable"`
		} `json:"mediaScan"`
		VolumeRef        string `json:"volumeRef"`
		Status           string `json:"status"`
		VolumeGroupRef   string `json:"volumeGroupRef"`
		CurrentManager   string `json:"currentManager"`
		PreferredManager string `json:"preferredManager"`
		Perms            struct {
			MapToLUN                        bool `json:"mapToLUN"`
			SnapShot                        bool `json:"snapShot"`
			Format                          bool `json:"format"`
			Reconfigure                     bool `json:"reconfigure"`
			MirrorPrimary                   bool `json:"mirrorPrimary"`
			MirrorSecondary                 bool `json:"mirrorSecondary"`
			CopySource                      bool `json:"copySource"`
			CopyTarget                      bool `json:"copyTarget"`
			Readable                        bool `json:"readable"`
			Writable                        bool `json:"writable"`
			Rollback                        bool `json:"rollback"`
			MirrorSync                      bool `json:"mirrorSync"`
			NewImage                        bool `json:"newImage"`
			AllowDVE                        bool `json:"allowDVE"`
			AllowDSS                        bool `json:"allowDSS"`
			ConcatVolumeMember              bool `json:"concatVolumeMember"`
			FlashReadCache                  bool `json:"flashReadCache"`
			AsyncMirrorPrimary              bool `json:"asyncMirrorPrimary"`
			AsyncMirrorSecondary            bool `json:"asyncMirrorSecondary"`
			PitGroup                        bool `json:"pitGroup"`
			CacheParametersChangeable       bool `json:"cacheParametersChangeable"`
			AllowThinManualExpansion        bool `json:"allowThinManualExpansion"`
			AllowThinGrowthParametersChange bool `json:"allowThinGrowthParametersChange"`
			AllowImportTarget               bool `json:"allowImportTarget"`
		} `json:"perms"`
		MgmtClientAttribute                         int    `json:"mgmtClientAttribute"`
		DssPreallocEnabled                          bool   `json:"dssPreallocEnabled"`
		DssMaxSegmentSize                           int    `json:"dssMaxSegmentSize"`
		PreReadRedundancyCheckEnabled               bool   `json:"preReadRedundancyCheckEnabled"`
		ProtectionInformationCapable                bool   `json:"protectionInformationCapable"`
		ProtectionType                              string `json:"protectionType"`
		ApplicationTagOwned                         bool   `json:"applicationTagOwned"`
		RepairedBlockCount                          int    `json:"repairedBlockCount"`
		ExtendedUniqueIdentifier                    string `json:"extendedUniqueIdentifier"`
		CacheMirroringValidateProtectionInformation bool   `json:"cacheMirroringValidateProtectionInformation"`
		ExpectedProtectionInformationAppTag         string `json:"expectedProtectionInformationAppTag"`
		HostUnmapEnabled                            bool   `json:"hostUnmapEnabled"`
		CachePoolID                                 int    `json:"cachePoolID"`
		BlkSizePhysical                             int    `json:"blkSizePhysical"`
		VolumeUse                                   string `json:"volumeUse"`
		VolumeFull                                  bool   `json:"volumeFull"`
		VolumeCopyTarget                            bool   `json:"volumeCopyTarget"`
		VolumeCopySource                            bool   `json:"volumeCopySource"`
		PitBaseVolume                               bool   `json:"pitBaseVolume"`
		AsyncMirrorTarget                           bool   `json:"asyncMirrorTarget"`
		AsyncMirrorSource                           bool   `json:"asyncMirrorSource"`
		RemoteMirrorSource                          bool   `json:"remoteMirrorSource"`
		RemoteMirrorTarget                          bool   `json:"remoteMirrorTarget"`
		DiskPool                                    bool   `json:"diskPool"`
		FlashCached                                 bool   `json:"flashCached"`
		IncreasingBy                                string `json:"increasingBy"`
		Metadata                                    []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"metadata"`
		DataAssurance         bool   `json:"dataAssurance"`
		ObjectType            string `json:"objectType"`
		Wwn                   string `json:"wwn"`
		PreferredControllerID string `json:"preferredControllerId"`
		TotalSizeInBytes      string `json:"totalSizeInBytes"`
		OnlineVolumeCopy      bool   `json:"onlineVolumeCopy"`
		ListOfMappings        []struct {
			LunMappingRef string `json:"lunMappingRef"`
			Lun           int    `json:"lun"`
			Ssid          int    `json:"ssid"`
			Perms         int    `json:"perms"`
			VolumeRef     string `json:"volumeRef"`
			Type          string `json:"type"`
			MapRef        string `json:"mapRef"`
			ID            string `json:"id"`
		} `json:"listOfMappings"`
		Mapped              bool   `json:"mapped"`
		CurrentControllerID string `json:"currentControllerId"`
		CacheSettings       struct {
			Cwob                bool   `json:"cwob"`
			EnterpriseCacheDump bool   `json:"enterpriseCacheDump"`
			MirrorActive        bool   `json:"mirrorActive"`
			MirrorEnable        bool   `json:"mirrorEnable"`
			ReadCacheActive     bool   `json:"readCacheActive"`
			ReadCacheEnable     bool   `json:"readCacheEnable"`
			WriteCacheActive    bool   `json:"writeCacheActive"`
			WriteCacheEnable    bool   `json:"writeCacheEnable"`
			CacheFlushModifier  string `json:"cacheFlushModifier"`
			ReadAheadMultiplier int    `json:"readAheadMultiplier"`
		} `json:"cacheSettings"`
		ThinProvisioned bool   `json:"thinProvisioned"`
		Name            string `json:"name"`
		ID              string `json:"id"`
	} `json:"volume"`
	FreeExtent []struct {
		SectorOffset   string `json:"sectorOffset"`
		RawCapacity    string `json:"rawCapacity"`
		RaidLevel      string `json:"raidLevel"`
		VolumeGroupRef string `json:"volumeGroupRef"`
		FreeExtentRef  string `json:"freeExtentRef"`
		Reserved1      string `json:"reserved1"`
		Reserved2      string `json:"reserved2"`
		DdpRAIDCapacities []struct {
			DdpVolRAIDLevel string `json:"ddpVolRAIDLevel"`
			UsableCapacity  string `json:"usableCapacity"`
		} `json:"ddpRAIDCapacities"`
	} `json:"freeExtent"`
	DveVol []any `json:"dveVol"`
	StoragePoolBundle struct {
		SaPortGroup []any `json:"saPortGroup"`
		SaPort      []any `json:"saPort"`
		Cluster     []struct {
			ClusterRef                                 string `json:"clusterRef"`
			Label                                      string `json:"label"`
			IsSAControlled                             bool   `json:"isSAControlled"`
			ConfirmLUNMappingCreation                  bool   `json:"confirmLUNMappingCreation"`
			ProtectionInformationCapableAccessMethod bool   `json:"protectionInformationCapableAccessMethod"`
			IsLun0Restricted                           bool   `json:"isLun0Restricted"`
			ID                                         string `json:"id"`
			Name                                       string `json:"name"`
		} `json:"cluster"`
		Host []struct {
			HostRef                                  string `json:"hostRef"`
			ClusterRef                               string `json:"clusterRef"`
			Label                                    string `json:"label"`
			IsSAControlled                           bool   `json:"isSAControlled"`
			ConfirmLUNMappingCreation                bool   `json:"confirmLUNMappingCreation"`
			HostTypeIndex                            int    `json:"hostTypeIndex"`
			ProtectionInformationCapableAccessMethod bool   `json:"protectionInformationCapableAccessMethod"`
			IsLargeBlockFormatHost                   bool   `json:"isLargeBlockFormatHost"`
			IsLun0Restricted                         bool   `json:"isLun0Restricted"`
			Ports                                    []any  `json:"ports"`
			Initiators                               []struct {
				InitiatorRef string `json:"initiatorRef"`
				NodeName     struct {
					IoInterfaceType string `json:"ioInterfaceType"`
					IscsiNodeName   string `json:"iscsiNodeName"`
					RemoteNodeWWN   any    `json:"remoteNodeWWN"`
					NvmeNodeName    any    `json:"nvmeNodeName"`
				} `json:"nodeName"`
				Alias struct {
					IoInterfaceType string `json:"ioInterfaceType"`
					IscsiAlias      string `json:"iscsiAlias"`
				} `json:"alias"`
				Label                 string `json:"label"`
				ConfiguredAuthMethods struct {
					AuthMethodData []struct {
						AuthMethod string `json:"authMethod"`
						ChapSecret any    `json:"chapSecret"`
					} `json:"authMethodData"`
				} `json:"configuredAuthMethods"`
				HostRef           string `json:"hostRef"`
				InitiatorInactive bool   `json:"initiatorInactive"`
				InitiatorNodeName struct {
					NodeName struct {
						IoInterfaceType string `json:"ioInterfaceType"`
						IscsiNodeName   string `json:"iscsiNodeName"`
						RemoteNodeWWN   any    `json:"remoteNodeWWN"`
						NvmeNodeName    any    `json:"nvmeNodeName"`
					} `json:"nodeName"`
					InterfaceType string `json:"interfaceType"`
				} `json:"initiatorNodeName"`
				ID string `json:"id"`
			} `json:"initiators"`
			HostSidePorts []struct {
				ID                 string `json:"id"`
				Type               string `json:"type"`
				Address            string `json:"address"`
				Label              string `json:"label"`
				MtpIoInterfaceType string `json:"mtpIoInterfaceType"`
				Name               string `json:"name"`
			} `json:"hostSidePorts"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"host"`
		HostPort   []any `json:"hostPort"`
		LunMapping []struct {
			LunMappingRef string `json:"lunMappingRef"`
			Lun           int    `json:"lun"`
			Ssid          int    `json:"ssid"`
			Perms         int    `json:"perms"`
			VolumeRef     string `json:"volumeRef"`
			Type          string `json:"type"`
			MapRef        string `json:"mapRef"`
			ID            string `json:"id"`
		} `json:"lunMapping"`
		Initiator []struct {
			InitiatorRef string `json:"initiatorRef"`
			NodeName     struct {
				IoInterfaceType string `json:"ioInterfaceType"`
				IscsiNodeName   string `json:"iscsiNodeName"`
				RemoteNodeWWN   any    `json:"remoteNodeWWN"`
				NvmeNodeName    any    `json:"nvmeNodeName"`
			} `json:"nodeName"`
			Alias struct {
				IoInterfaceType string `json:"ioInterfaceType"`
				IscsiAlias      string `json:"iscsiAlias"`
			} `json:"alias"`
			Label                 string `json:"label"`
			ConfiguredAuthMethods struct {
				AuthMethodData []struct {
					AuthMethod string `json:"authMethod"`
					ChapSecret any    `json:"chapSecret"`
				} `json:"authMethodData"`
			} `json:"configuredAuthMethods"`
			HostRef           string `json:"hostRef"`
			InitiatorInactive bool   `json:"initiatorInactive"`
			InitiatorNodeName struct {
				IoInterfaceType string `json:"ioInterfaceType"`
				IscsiNodeName   string `json:"iscsiNodeName"`
				RemoteNodeWWN   any    `json:"remoteNodeWWN"`
				NvmeNodeName    any    `json:"nvmeNodeName"`
			} `json:"initiatorNodeName"`
			InterfaceType string `json:"interfaceType"`
			ID            string `json:"id"`
		} `json:"initiator"`
		Target []struct {
			TargetRef string `json:"targetRef"`
			NodeName  struct {
				IoInterfaceType string `json:"ioInterfaceType"`
				IscsiNodeName   string `json:"iscsiNodeName"`
				RemoteNodeWWN   any    `json:"remoteNodeWWN"`
				NvmeNodeName    any    `json:"nvmeNodeName"`
			} `json:"nodeName"`
			Alias struct {
				IoInterfaceType string `json:"ioInterfaceType"`
				IscsiAlias      string `json:"iscsiAlias"`
			} `json:"alias"`
			ConfiguredAuthMethods struct {
				AuthMethodData []struct {
					AuthMethod string `json:"authMethod"`
					ChapSecret any    `json:"chapSecret"`
				} `json:"authMethodData"`
			} `json:"configuredAuthMethods"`
			Portals []struct {
				GroupTag  int `json:"groupTag"`
				IPAddress struct {
					AddressType string `json:"addressType"`
					Ipv4Address string `json:"ipv4Address"`
					Ipv6Address any    `json:"ipv6Address"`
				} `json:"ipAddress"`
				TCPListenPort int `json:"tcpListenPort"`
			} `json:"portals"`
		} `json:"target"`
		RemoteTarget []any `json:"remoteTarget"`
	} `json:"storagePoolBundle"`
	HighLevelVolBundle struct {
		SnapshotVol                      []any `json:"snapshotVol"`
		MirrorVol                        []any `json:"mirrorVol"`
		MetadataVol                      []any `json:"metadataVol"`
		RemoteVol                        []any `json:"remoteVol"`
		VolumeCopy                       []any `json:"volumeCopy"`
		PitGroup                         []any `json:"pitGroup"`
		Pit                              []any `json:"pit"`
		PitView                          []any `json:"pitView"`
		PitConsistencyGroup              []any `json:"pitConsistencyGroup"`
		ConcatVolume                     []any `json:"concatVolume"`
		ConcatVolMember                  []any `json:"concatVolMember"`
		PitConsistencyGroupView          []any `json:"pitConsistencyGroupView"`
		ThinVolume                       []any `json:"thinVolume"`
		AsyncMirrorGroup                 []any `json:"asyncMirrorGroup"`
		AsyncMirrorGroupMember           []any `json:"asyncMirrorGroupMember"`
		AsyncMirrorGroupIncompleteMember []any `json:"asyncMirrorGroupIncompleteMember"`
		RemoteTargetGroups               []any `json:"remoteTargetGroups"`
		RemoteTargetVolumes              []any `json:"remoteTargetVolumes"`
	} `json:"highLevelVolBundle"`
	GhostVolBundle struct {
		GhostVol []any `json:"ghostVol"`
	} `json:"ghostVolBundle"`
	ConsistencyGroupBundle struct {
		ConsistencyGroup []struct {
			GroupRef string `json:"groupRef"`
		} `json:"consistencyGroup"`
	} `json:"consistencyGroupBundle"`
	IoInterface []struct {
		InterfaceRef string `json:"interfaceRef"`
		ChannelType  string `json:"channelType"`
		IoInterfaceTypeData struct {
			InterfaceType string `json:"interfaceType"`
			Fibre         any    `json:"fibre"`
			Ib            any    `json:"ib"`
			Iscsi         any    `json:"iscsi"`
			Sas           struct {
				Channel               int    `json:"channel"`
				CurrentInterfaceSpeed string `json:"currentInterfaceSpeed"`
				MaximumInterfaceSpeed string `json:"maximumInterfaceSpeed"`
				Part                  string `json:"part"`
				Revision              int    `json:"revision"`
				IsDegraded            bool   `json:"isDegraded"`
				IocPort               struct {
					Parent struct {
						Type         string `json:"type"`
						Controller   string `json:"controller"`
						Drive        any    `json:"drive"`
						Expander     any    `json:"expander"`
						HostBoardRef any    `json:"hostBoardRef"`
					} `json:"parent"`
					AttachedDevice struct {
						Channel     int    `json:"channel"`
						ChannelType string `json:"channelType"`
						SasAttachedDeviceData struct {
							Type                  string `json:"type"`
							AlternateController   any    `json:"alternateController"`
							Drive                 any    `json:"drive"`
							Expander              string `json:"expander"`
							RemoteHostPortAddress any    `json:"remoteHostPortAddress"`
							LocalController       any    `json:"localController"`
							PhysicalLocation      any    `json:"physicalLocation"`
						} `json:"sasAttachedDeviceData"`
					} `json:"attachedDevice"`
					State                   string `json:"state"`
					MiswireType             string `json:"miswireType"`
					ChannelPortRef          string `json:"channelPortRef"`
					SasPhys                 []struct {
						PhyIdentifier int  `json:"phyIdentifier"`
						IsOperational bool `json:"isOperational"`
					} `json:"sasPhys"`
					PortTypeData struct {
						PortType       string `json:"portType"`
						PortIdentifier string `json:"portIdentifier"`
						RoutingType    string `json:"routingType"`
					} `json:"portTypeData"`
					PortMode                string   `json:"portMode"`
					DomainNumber            int      `json:"domainNumber"`
					AttachedChannelPortRef  string   `json:"attachedChannelPortRef"`
					DiscoveryStatus         int      `json:"discoveryStatus"`
					NonOptimalCablingType   string   `json:"nonOptimalCablingType"`
					ChannelPortRefs         []string `json:"channelPortRefs"`
					AttachedChannelPortRefs []string `json:"attachedChannelPortRefs"`
				} `json:"iocPort"`
				InterfaceRef string `json:"interfaceRef"`
				PhysicalLocation struct {
					TrayRef        string `json:"trayRef"`
					Slot           int    `json:"slot"`
					LocationParent struct {
						RefType        string `json:"refType"`
						ControllerRef  any    `json:"controllerRef"`
						SymbolRef      string `json:"symbolRef"`
						TypedReference any    `json:"typedReference"`
					} `json:"locationParent"`
					LocationPosition int    `json:"locationPosition"`
					Label            string `json:"label"`
				} `json:"physicalLocation"`
				ProtectionInformationCapable bool   `json:"protectionInformationCapable"`
				OneWayMaxRate                string `json:"oneWayMaxRate"`
				BidirectionalMaxRate         string `json:"bidirectionalMaxRate"`
				ControllerID                 any    `json:"controllerId"`
				InterfaceID                  string `json:"interfaceId"`
				AddressID                    string `json:"addressId"`
				NiceAddressID                string `json:"niceAddressId"`
				BasePortAddress              string `json:"basePortAddress"`
				ID                           string `json:"id"`
			} `json:"sas"`
			Sata               any `json:"sata"`
			Scsi               any `json:"scsi"`
			CouplingDriverNvme any `json:"couplingDriverNvme"`
			Ethernet           any `json:"ethernet"`
			Pcie               any `json:"pcie"`
		} `json:"ioInterfaceTypeData"`
		ControllerRef string `json:"controllerRef"`
		CommandProtocolPropertiesList struct {
			CommandProtocolProperties []any `json:"commandProtocolProperties"`
		} `json:"commandProtocolPropertiesList"`
		ChannelPortRef  string   `json:"channelPortRef"`
		ChannelPortRefs []string `json:"channelPortRefs"`
	} `json:"ioInterface"`
	NetworkInterface []any `json:"networkInterface"`
	LockKeyIDs       []any `json:"lockKeyIDs"`
	Interposer       []any `json:"interposer"`
	IoInterfaceHicMap []struct {
		InterfaceRef string `json:"interfaceRef"`
		HostBoardRef string `json:"hostBoardRef"`
	} `json:"ioInterfaceHicMap"`
	Schedule    []any `json:"schedule"`
	RemoteArray []any `json:"remoteArray"`
	FlashCacheBundle struct {
		FlashCacheMap []any `json:"flashCacheMap"`
		FlashCache    []any `json:"flashCache"`
	} `json:"flashCacheBundle"`
	SnmpBundle struct {
		Community       []any `json:"community"`
		TrapDestination []any `json:"trapDestination"`
		SystemVariables struct {
			SysName     string `json:"sysName"`
			SysContact  string `json:"sysContact"`
			SysLocation string `json:"sysLocation"`
		} `json:"systemVariables"`
		UsmUser []any `json:"usmUser"`
	} `json:"snmpBundle"`
	ApplicationAwarenessBundle struct {
		KeyValueTag []struct {
			KeyValueTagRef string `json:"keyValueTagRef"`
			Key            string `json:"key"`
			Value          string `json:"value"`
		} `json:"keyValueTag"`
		KeyValueTagMapping []struct {
			KeyValueTagRef string `json:"keyValueTagRef"`
			AssociatedObject struct {
				ObjectType string `json:"objectType"`
				Volume     any    `json:"volume"`
				Workload   string `json:"workload"`
			} `json:"associatedObject"`
			MappingRef string `json:"mappingRef"`
		} `json:"keyValueTagMapping"`
		Workload []struct {
			WorkloadRef string `json:"workloadRef"`
			Description string `json:"description"`
		} `json:"workload"`
		WorkloadVolumeMapping []any `json:"workloadVolumeMapping"`
	} `json:"applicationAwarenessBundle"`
}

// https://10.177.100.101/devmgr/v2/storage-systems/1/system-config
type SystemConfig struct {
	VolumeGroups []any `json:"volumeGroups"`
	Pools        []struct {
		Drives []struct {
			Tray   int `json:"tray"`
			Drawer int `json:"drawer"`
			Slot   int `json:"slot"`
		} `json:"drives"`
		Label                            string `json:"label"`
		SecurityType                     string `json:"securityType"`
		ReservedSpaceAllocated           bool   `json:"reservedSpaceAllocated"`
		UtilizationWarningThreshold      int    `json:"utilizationWarningThreshold"`
		UtilizationCriticalThreshold     int    `json:"utilizationCriticalThreshold"`
		BackgroundOperationPriority      string `json:"backgroundOperationPriority"`
		CriticalReconstructionPriority   string `json:"criticalReconstructionPriority"`
		DegradedReconstructionPriority   string `json:"degradedReconstructionPriority"`
		ReservedDriveCount               int    `json:"reservedDriveCount"`
	} `json:"pools"`
	Volumes []struct {
		Label              string `json:"label"`
		VolumeGroupLabel   string `json:"volumeGroupLabel"`
		Capacity           string `json:"capacity"`
		SegmentSize        int    `json:"segmentSize"`
		OwningController   string `json:"owningController"`
		T10ProtectionType  string `json:"t10ProtectionType"`
		DssPreallocateEnabled bool `json:"dssPreallocateEnabled"`
		RaidLevel          string `json:"raidLevel"`
		CacheSettings      struct {
			CacheFlushModifier              string `json:"cacheFlushModifier"`
			CachingWithoutBatteriesEnabled  bool   `json:"cachingWithoutBatteriesEnabled"`
			MirrorEnabled                   bool   `json:"mirrorEnabled"`
			ReadAheadEnabled                bool   `json:"readAheadEnabled"`
			ReadCacheEnabled                bool   `json:"readCacheEnabled"`
			WriteCacheEnabled               bool   `json:"writeCacheEnabled"`
		} `json:"cacheSettings"`
		PreReadRedundancyCheckEnabled bool `json:"preReadRedundancyCheckEnabled"`
		MediaScanSettings             struct {
			Enabled                 bool `json:"enabled"`
			ParityValidationEnabled bool `json:"parityValidationEnabled"`
		} `json:"mediaScanSettings"`
		ReconstructionPriority int `json:"reconstructionPriority"`
		BlockSize              int `json:"blockSize"`
	} `json:"volumes"`
	HotSpareDrives []any `json:"hotSpareDrives"`
	SystemSettings struct {
		CacheSettings struct {
			BlockSize                               int  `json:"blockSize"`
			FlushStart                              int  `json:"flushStart"`
			FlushStop                               int  `json:"flushStop"`
			MirroringProtectionInformationEnabled   bool `json:"mirroringProtectionInformationEnabled"`
		} `json:"cacheSettings"`
		MediaScanPeriod          int  `json:"mediaScanPeriod"`
		AutoLoadBalancingEnabled bool `json:"autoLoadBalancingEnabled"`
	} `json:"systemSettings"`
}

//https://10.177.100.101/devmgr/v2/storage-systems/1/mappable-objects
type MappableObject struct {
	Offline           bool   `json:"offline,omitempty"`
	ExtremeProtection bool   `json:"extremeProtection,omitempty"`
	VolumeHandle      int    `json:"volumeHandle"`
	RaidLevel         string `json:"raidLevel,omitempty"`
	SectorOffset      string `json:"sectorOffset,omitempty"`
	WorldWideName     string `json:"worldWideName,omitempty"`
	Label             string `json:"label,omitempty"`
	BlkSize           int    `json:"blkSize,omitempty"`
	Capacity          string `json:"capacity"`
	ReconPriority     int    `json:"reconPriority,omitempty"`
	SegmentSize       int    `json:"segmentSize,omitempty"`
	Action            string `json:"action,omitempty"`
	Cache             struct {
		Cwob                bool   `json:"cwob"`
		EnterpriseCacheDump bool   `json:"enterpriseCacheDump"`
		MirrorActive        bool   `json:"mirrorActive"`
		MirrorEnable        bool   `json:"mirrorEnable"`
		ReadCacheActive     bool   `json:"readCacheActive"`
		ReadCacheEnable     bool   `json:"readCacheEnable"`
		WriteCacheActive    bool   `json:"writeCacheActive"`
		WriteCacheEnable    bool   `json:"writeCacheEnable"`
		CacheFlushModifier  string `json:"cacheFlushModifier"`
		ReadAheadMultiplier int    `json:"readAheadMultiplier"`
	} `json:"cache,omitempty"`
	MediaScan struct {
		Enable                 bool `json:"enable"`
		ParityValidationEnable bool `json:"parityValidationEnable"`
	} `json:"mediaScan,omitempty"`
	VolumeRef        string `json:"volumeRef,omitempty"`
	Status           string `json:"status,omitempty"`
	VolumeGroupRef   string `json:"volumeGroupRef,omitempty"`
	CurrentManager   string `json:"currentManager,omitempty"`
	PreferredManager string `json:"preferredManager,omitempty"`
	Perms            struct {
		MapToLUN                        bool `json:"mapToLUN"`
		SnapShot                        bool `json:"snapShot"`
		Format                          bool `json:"format"`
		Reconfigure                     bool `json:"reconfigure"`
		MirrorPrimary                   bool `json:"mirrorPrimary"`
		MirrorSecondary                 bool `json:"mirrorSecondary"`
		CopySource                      bool `json:"copySource"`
		CopyTarget                      bool `json:"copyTarget"`
		Readable                        bool `json:"readable"`
		Writable                        bool `json:"writable"`
		Rollback                        bool `json:"rollback"`
		MirrorSync                      bool `json:"mirrorSync"`
		NewImage                        bool `json:"newImage"`
		AllowDVE                        bool `json:"allowDVE"`
		AllowDSS                        bool `json:"allowDSS"`
		ConcatVolumeMember              bool `json:"concatVolumeMember"`
		FlashReadCache                  bool `json:"flashReadCache"`
		AsyncMirrorPrimary              bool `json:"asyncMirrorPrimary"`
		AsyncMirrorSecondary            bool `json:"asyncMirrorSecondary"`
		PitGroup                        bool `json:"pitGroup"`
		CacheParametersChangeable       bool `json:"cacheParametersChangeable"`
		AllowThinManualExpansion        bool `json:"allowThinManualExpansion"`
		AllowThinGrowthParametersChange bool `json:"allowThinGrowthParametersChange"`
		AllowImportTarget               bool `json:"allowImportTarget"`
	} `json:"perms,omitempty"`
	MgmtClientAttribute                         int    `json:"mgmtClientAttribute,omitempty"`
	DssPreallocEnabled                          bool   `json:"dssPreallocEnabled,omitempty"`
	DssMaxSegmentSize                           int    `json:"dssMaxSegmentSize,omitempty"`
	PreReadRedundancyCheckEnabled               bool   `json:"preReadRedundancyCheckEnabled,omitempty"`
	ProtectionInformationCapable                bool   `json:"protectionInformationCapable,omitempty"`
	ProtectionType                              string `json:"protectionType,omitempty"`
	ApplicationTagOwned                         bool   `json:"applicationTagOwned,omitempty"`
	RepairedBlockCount                          int    `json:"repairedBlockCount,omitempty"`
	ExtendedUniqueIdentifier                    string `json:"extendedUniqueIdentifier,omitempty"`
	CacheMirroringValidateProtectionInformation bool   `json:"cacheMirroringValidateProtectionInformation,omitempty"`
	ExpectedProtectionInformationAppTag         int    `json:"expectedProtectionInformationAppTag,omitempty"`
	HostUnmapEnabled                            bool   `json:"hostUnmapEnabled,omitempty"`
	CachePoolID                                 int    `json:"cachePoolID,omitempty"`
	BlkSizePhysical                             int    `json:"blkSizePhysical,omitempty"`
	VolumeUse                                   string `json:"volumeUse,omitempty"`
	VolumeFull                                  bool   `json:"volumeFull,omitempty"`
	VolumeCopyTarget                            bool   `json:"volumeCopyTarget,omitempty"`
	VolumeCopySource                            bool   `json:"volumeCopySource,omitempty"`
	PitBaseVolume                               bool   `json:"pitBaseVolume,omitempty"`
	AsyncMirrorTarget                           bool   `json:"asyncMirrorTarget,omitempty"`
	AsyncMirrorSource                           bool   `json:"asyncMirrorSource,omitempty"`
	RemoteMirrorSource                          bool   `json:"remoteMirrorSource,omitempty"`
	RemoteMirrorTarget                          bool   `json:"remoteMirrorTarget,omitempty"`
	DiskPool                                    bool   `json:"diskPool,omitempty"`
	FlashCached                                 bool   `json:"flashCached,omitempty"`
	IncreasingBy                                string `json:"increasingBy,omitempty"`
	Metadata                                    []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"metadata,omitempty"`
	DataAssurance         bool   `json:"dataAssurance,omitempty"`
	ObjectType            string `json:"objectType"`
	Wwn                   string `json:"wwn"`
	PreferredControllerID string `json:"preferredControllerId"`
	TotalSizeInBytes      string `json:"totalSizeInBytes"`
	OnlineVolumeCopy      bool   `json:"onlineVolumeCopy,omitempty"`
	ListOfMappings        []struct {
		LunMappingRef string `json:"lunMappingRef"`
		Lun           int    `json:"lun"`
		Ssid          int    `json:"ssid"`
		Perms         int    `json:"perms"`
		VolumeRef     string `json:"volumeRef"`
		Type          string `json:"type"`
		MapRef        string `json:"mapRef"`
		ID            string `json:"id"`
	} `json:"listOfMappings"`
	Mapped              bool   `json:"mapped"`
	CurrentControllerID string `json:"currentControllerId"`
	CacheSettings       struct {
		Cwob                bool   `json:"cwob"`
		EnterpriseCacheDump bool   `json:"enterpriseCacheDump"`
		MirrorActive        bool   `json:"mirrorActive"`
		MirrorEnable        bool   `json:"mirrorEnable"`
		ReadCacheActive     bool   `json:"readCacheActive"`
		ReadCacheEnable     bool   `json:"readCacheEnable"`
		WriteCacheActive    bool   `json:"writeCacheActive"`
		WriteCacheEnable    bool   `json:"writeCacheEnable"`
		CacheFlushModifier  string `json:"cacheFlushModifier"`
		ReadAheadMultiplier int    `json:"readAheadMultiplier"`
	} `json:"cacheSettings,omitempty"`
	ThinProvisioned bool   `json:"thinProvisioned,omitempty"`
	Name            string `json:"name"`
	ID              string `json:"id"`
	Enabled         bool   `json:"enabled,omitempty"`
	AccessVolumeRef string `json:"accessVolumeRef,omitempty"`
	Reserved1       string `json:"reserved1,omitempty"`
}

// GetVolumes возвращает список томов СХД (/devmgr/v2/storage-systems/1/volumes)
func (session *Session) GetVolumes() ([]Volume, error) {
	resp := make([]Volume, 0)
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/volumes", "", "", false, nil, &resp)
	return resp, err
}

// GetAccessVolume возвращает служебный том доступа (/devmgr/v2/storage-systems/1/access-volume)
func (session *Session) GetAccessVolume() (AccessVolume, error) {
	resp := AccessVolume{}
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/access-volume", "", "", false, nil, &resp)
	return resp, err
}

// GetHosts возвращает список хостов, подключённых к СХД (/devmgr/v2/storage-systems/1/hosts)
func (session *Session) GetHosts() ([]Host, error) {
	resp := make([]Host, 0)
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/hosts", "", "", false, nil, &resp)
	return resp, err
}

// GetStoragePools возвращает список пулов хранения (/devmgr/v2/storage-systems/1/storage-pools)
func (session *Session) GetStoragePools() ([]StoragePool, error) {
	resp := make([]StoragePool, 0)
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/storage-pools", "", "", false, nil, &resp)
	return resp, err
}

// GetHostGroup возвращает список групп хостов (/devmgr/v2/storage-systems/1/host-groups)
func (session *Session) GetHostGroup() ([]HostGroup, error) {
	resp := make([]HostGroup, 0)
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/host-groups", "", "", false, nil, &resp)
	return resp, err
}

// GetAbout возвращает общую информацию о СХД (/devmgr/v2/storage-systems/1/about)
func (session *Session) GetAbout() (About, error) {
	resp := About{}
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/about", "", "", false, nil, &resp)
	return resp, err
}

// GetGraph возвращает граф топологии объектов СХД (/devmgr/v2/storage-systems/1/graph)
func (session *Session) GetGraph() (Graph, error) {
	resp := Graph{}
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/graph", "", "", false, nil, &resp)
	return resp, err
}

// GetSystemConfig возвращает конфигурацию системы (/devmgr/v2/storage-systems/1/system-config)
func (session *Session) GetSystemConfig() (SystemConfig, error) {
	resp := SystemConfig{}
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/system-config", "", "", false, nil, &resp)
	return resp, err
}

// GetMappableObject возвращает список объектов, доступных для отображения на хосты (/devmgr/v2/storage-systems/1/mappable-objects)
func (session *Session) GetMappableObject() ([]MappableObject, error) {
	resp := make([]MappableObject, 0)
	err := session.Request("GET", "/devmgr/v2/storage-systems/1/mappable-objects", "", "", false, nil, &resp)
	return resp, err
}
