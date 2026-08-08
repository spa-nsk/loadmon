package emcstorage

func (session *Session) GetAlert() (resp *Alert, err error) {
	fields := "id,severity,description"
	err = session.Request("GET", "/api/types/alert/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetEvent() (resp *Event, err error) {
	fields := "id,node,creationTime,severity,messageId,arguments,message,username,category,source"
	err = session.Request("GET", "/api/types/event/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetEventIDs() (resp *Event, err error) {
	fields := "id,node,creationTime,severity,messageId,category"
	err = session.Request("GET", "/api/types/event/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetDisks() (resp *Disk, err error) {
	fields := "id,needsReplacement,slotNumber,estimatedEOL,busId,name,manufacturer,model,version,emcPartNumber,emcSerialNumber,tierType,currentSpeed,maxSpeed,isInUse,isFastCacheInUse,size,rawSize,vendorSize,wwn,diskTechnology"
	err = session.Request("GET", "/api/types/disk/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetStorageProcessor() (resp *StorageProcessor, err error) {
	//	fields := "id,parent.id,parent.resource,health.value,helth.descriptionIds,helth.descriptions,helth.resolutionIds,helth.resolutions,isRescueMode,model,
	//slotNumber,name,emcPartNumber,emcSerialNumber,manufacturer,vendorPartNumber,vendorSerialNumber,sasExpanderVersion,biosFirmwareRevision,postFirmwareRevision,memorySize,modelName"
	fields := "id,parent.id,parent.resource,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,isRescueMode,model,slotNumber,name,emcPartNumber," +
		"emcSerialNumber,manufacturer,vendorPartNumber,vendorSerialNumber,sasExpanderVersion,biosFirmwareRevision,postFirmwareRevision,memorySize,modelName," +
		"parentDpe.id,parentDpe.health.value,parentDpe.health.descriptionIds,parentDpe.health.descriptions,parentDpe.health.resolutionIds,parentDpe.health.resolutions,parentDpe.needsReplacement," +
		"parentDpe.parent.id,parentDpe.parent.resource,parentDpe.slotNumber,parentDpe.name,parentDpe.manufacturer,parentDpe.model,parentDpe.emcPartNumber," +
		"parentDpe.vendorSerialNumber,parentDpe.enclosureType,parentDpe.busId,parentDpe.driveTypes,parentDpe.currentPower,parentDpe.avgPower,parentDpe.maxPower," +
		"parentDpe.currentTemperature,parentDpe.avgTemperature," +
		"parentDpe.maxTemperature,parentDpe.currentSpeed,parentDpe.maxSpeed," +
		"parentDpe.parentSystem.id,parentDpe.parentSystem.health.value,parentDpe.parentSystem.health.descriptionIds,parentDpe.parentSystem.health.descriptions," +
		"parentDpe.parentSystem.health.resolutionIds,parentDpe.parentSystem.health.resolutions," +
		"parentDpe.parentSystem.name,parentDpe.parentSystem.model,parentDpe.parentSystem.serialNumber,parentDpe.parentSystem.uuidBase,parentDpe.parentSystem.internalModel," +
		"parentDpe.parentSystem.platform,parentDpe.parentSystem.isAllFlash," +
		"parentDpe.parentSystem.macAddress,parentDpe.parentSystem.isEULAAccepted,parentDpe.parentSystem.isUpgradeComplete,parentDpe.parentSystem.isAutoFailbackEnabled," +
		"parentDpe.parentSystem.currentPower,parentDpe.parentSystem.avgPower,parentDpe.parentSystem.supportedUpgradeModels"

	err = session.Request("GET", "/api/types/storageProcessor/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetDAE() (resp *DAE, err error) {

	fields := "id,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,needsReplacement,parent.id,parent.resource,slotNumber," +
		"name,manufacturer,model,emcPartNumber,emcSerialNumber,vendorPartNumber,vendorSerialNumber,enclosureType,busId,driveTypes,currentPower,avgPower,maxPower,currentTemperature," +
		"avgTemperature,maxTemperature,currentSpeed,maxSpeed," +
		"parentSystem.id,parentSystem.health.value,parentSystem.health.descriptionIds,parentSystem.health.descriptions,parentSystem.health.resolutionIds,parentSystem.health.resolutions," +
		"parentSystem.name,parentSystem.model,parentSystem.serialNumber,parentSystem.uuidBase,parentSystem.internalModel,parentSystem.platform,parentSystem.isAllFlash," +
		"parentSystem.macAddress,parentSystem.isEULAAccepted,parentSystem.isUpgradeComplete,parentSystem.isAutoFailbackEnabled,parentSystem.currentPower," +
		"parentSystem.avgPower,parentSystem.supportedUpgradeModels"

	err = session.Request("GET", "/api/types/dae/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetDPE() (resp *DPE, err error) {

	fields := "id,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,needsReplacement,parent.id,parent.resource,slotNumber," +
		"name,manufacturer,model,emcPartNumber,emcSerialNumber,vendorPartNumber,vendorSerialNumber,enclosureType,busId,driveTypes,currentPower,avgPower,maxPower,currentTemperature," +
		"avgTemperature,maxTemperature,currentSpeed,maxSpeed," +
		"parentSystem.id,parentSystem.health.value,parentSystem.health.descriptionIds,parentSystem.health.descriptions,parentSystem.health.resolutionIds,parentSystem.health.resolutions," +
		"parentSystem.name,parentSystem.model,parentSystem.serialNumber,parentSystem.uuidBase,parentSystem.internalModel,parentSystem.platform,parentSystem.isAllFlash," +
		"parentSystem.macAddress,parentSystem.isEULAAccepted,parentSystem.isUpgradeComplete,parentSystem.isAutoFailbackEnabled,parentSystem.currentPower," +
		"parentSystem.avgPower,parentSystem.supportedUpgradeModels"

	err = session.Request("GET", "/api/types/dpe/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetDiskGroup() (resp *DiskGroup, err error) {

	fields := "id,name,emcPartNumber,tierType,diskTechnology,isFASTCacheAllowable,diskSize,advertisedSize,rpm,speed,totalDisks,minHotSpareCandidates,hotSparePolicyStatus,unconfiguredDisks," +
		"configuredTraditionalPoolDisks,configuredFastCacheDisks,disksPastEOL,disksWithEOLLess30Days,disksWithEOLLess60Days,disksWithEOLLess90Days,disksWithEOLLess180Days"

	err = session.Request("GET", "/api/types/diskGroup/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetHost() (resp *Host, err error) {

	fields := "id,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,name,description,type,osType,hostUUID,hostPushedUUID,hostPolledUUID," +
		"lastPollTime,autoManageType,registrationType,hostContainer.id,hostContainer.name,fcHostInitiators.id,fcHostInitiators.initiatorId,iscsiHostInitiators.id,iscsiHostInitiators.initiatorId," +
		"hostIPPorts.id,hostIPPorts.name,storageResources.id,storageResources.name,storageResources.description," +
		"hostLUNs.id,hostLUNs.lun.id,hostLUNs.lun.health.value,hostLUNs.lun.health.descriptionIds,hostLUNs.lun.health.descriptions,hostLUNs.lun.health.resolutionIds,hostLUNs.lun.health.resolutions," +
		"hostLUNs.lun.name,hostLUNs.lun.description,hostLUNs.lun.type,hostLUNs.lun.sizeTotal,hostLUNs.lun.sizeUsed,hostLUNs.lun.sizeAllocated," +
		"hostLUNs.lun.sizePreallocated,hostLUNs.lun.sizeAllocatedTotal," +
		"hostLUNs.lun.dataReductionSizeSaved,hostLUNs.lun.dataReductionPercent,hostLUNs.lun.dataReductionRatio,hostLUNs.lun.perTierSizeUsed," +
		"hostLUNs.lun.isThinEnabled,hostLUNs.lun.isDataReductionEnabled,hostLUNs.lun.isAdvancedDedupEnabled," +
		"isNfsv4Supported,nfsUsername,tenant.id,tenant.name,datastores.id,datastores.name,hostVVoldatastore.id,vms.id,vms.name,vms.guestHostName"

	err = session.Request("GET", "/api/types/host/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetLun() (resp *Lun, err error) {
	/*
		fields := "id,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,name,description,type,sizeTotal,sizeUsed,sizeAllocated," +
			"sizePreallocated,sizeAllocatedTotal,dataReductionSizeSaved,dataReductionPercent,dataReductionRatio,perTierSizeUsed,isThinEnabled,isDataReductionEnabled," +
			"isAdvancedDedupEnabled,storageResource.id,pool.id,wwn,tieringPolicy,defaultNode,isReplicationDestination,currentNode,metadataSize,metadataSizeAllocated," +
			"snapWwn,snapsSize,snapsSizeAllocated,hostAccess.host.id,hostAccess.host.name,hostAccess.accessMask,hostAccess.productionAccess,hostAccess.hlu,hostAccess.snapshotAccess,snapCount," +
			"isThinClone,familyBaseLun.id,parentSnap.id,originalParentLun.id,familySnapCount,familyCloneCount,nonBaseSizeAllocated,familySizeAllocatedTotal," +
			"nonBaseSize,migrationState"
	*/
	fields := "id,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,name,description,type,sizeTotal,sizeUsed,sizeAllocated," +
		"sizePreallocated,sizeAllocatedTotal,dataReductionSizeSaved,dataReductionPercent,dataReductionRatio,perTierSizeUsed,isThinEnabled,isDataReductionEnabled," +
		"isAdvancedDedupEnabled,storageResource.id,pool.id,wwn,tieringPolicy,defaultNode,isReplicationDestination,currentNode,metadataSize,metadataSizeAllocated," +
		"snapWwn,snapsSize,snapsSizeAllocated,snapCount," +
		"isThinClone,familyBaseLun.id,parentSnap.id,originalParentLun.id,familySnapCount,familyCloneCount,nonBaseSizeAllocated,familySizeAllocatedTotal," +
		"nonBaseSize,migrationState"

	err = session.Request("GET", "/api/types/lun/instances", fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetLunById(id string) (resp *Lun, err error) {
	fields := "id,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,name,description,type,sizeTotal,sizeUsed,sizeAllocated," +
		"sizePreallocated,sizeAllocatedTotal,dataReductionSizeSaved,dataReductionPercent,dataReductionRatio,perTierSizeUsed,isThinEnabled,isDataReductionEnabled," +
		"isAdvancedDedupEnabled,storageResource.id,pool.id,wwn,tieringPolicy,defaultNode,isReplicationDestination,currentNode,metadataSize,metadataSizeAllocated," +
		"snapWwn,snapsSize,snapsSizeAllocated,snapCount," +
		"isThinClone,familyBaseLun.id,parentSnap.id,originalParentLun.id,familySnapCount,familyCloneCount,nonBaseSizeAllocated,familySizeAllocatedTotal," +
		"nonBaseSize,migrationState"
	err = session.Request("GET", "/api/instances/lun/"+id, fields, "", false, nil, &resp)
	return resp, err
}

func (session *Session) GetLunByName(name string) (resp *Lun, err error) {
	fields := "id,health.value,health.descriptionIds,health.descriptions,health.resolutionIds,health.resolutions,name,description,type,sizeTotal,sizeUsed,sizeAllocated," +
		"sizePreallocated,sizeAllocatedTotal,dataReductionSizeSaved,dataReductionPercent,dataReductionRatio,perTierSizeUsed,isThinEnabled,isDataReductionEnabled," +
		"isAdvancedDedupEnabled,storageResource.id,pool.id,wwn,tieringPolicy,defaultNode,isReplicationDestination,currentNode,metadataSize,metadataSizeAllocated," +
		"snapWwn,snapsSize,snapsSizeAllocated,snapCount," +
		"isThinClone,familyBaseLun.id,parentSnap.id,originalParentLun.id,familySnapCount,familyCloneCount,nonBaseSizeAllocated,familySizeAllocatedTotal," +
		"nonBaseSize,migrationState"
	err = session.Request("GET", "/api/instances/lun/name:"+name, fields, "", false, nil, &resp)
	return resp, err
}
