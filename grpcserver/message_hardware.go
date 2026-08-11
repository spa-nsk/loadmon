package grpcserver

import (
	"fmt"
	"loadmon/oneview"
	"strconv"

	pb "loadmon/protocol"

	"github.com/HewlettPackard/oneview-golang/utils"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func ServerHardwareToMessage(in *oneview.ServerHardware, out *pb.ServerHardware) {
	if in == nil || out == nil {
		return
	}
	out.Base = &pb.OVServerHardware{}
	out.Base.Hostname = in.Base.Hostname
	out.Base.HostOsType = int32(in.Base.HostOsType)
	out.Base.MaintenanceMode = in.Base.MaintenanceMode
	out.Base.MemoryMb = int32(in.Base.MemoryMb)
	out.Base.Model = in.Base.Model
	out.Base.Name = in.Base.Name
	out.Base.PartNumber = in.Base.PartNumber
	out.Base.Position = int32(in.Base.Position)
	out.Base.PowerState = in.Base.PowerState
	out.Base.ProcessorCoreCount = int32(in.Base.ProcessorCoreCount)
	out.Base.ProcessorCount = int32(in.Base.ProcessorCount)
	out.Base.ProcessorSpeedMhz = int32(in.Base.ProcessorSpeedMhz)
	out.Base.ProcessorType = in.Base.ProcessorType
	out.Base.SerialNumber = in.Base.SerialNumber.String()
	out.Base.VirtualSerialNumber = in.Base.VirtualSerialNumber.String()
	out.Base.FormFactor = in.Base.FormFactor
	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = in.Base.UUID.String()
	out.Base.OperatingSystem = in.OperatingSystem

	out.Memory = &pb.ServerHardwareMemory{}
	out.Memory.URI = in.Memory.URI
	out.Memory.CollectionState = in.Memory.CollectionState
	out.Memory.Modified = timestamppb.New(in.Memory.Modified)
	out.Memory.Etag = in.Memory.Etag
	out.Memory.Count = int32(in.Memory.Count)
	out.Memory.Name = in.Memory.Name
	for _, m := range in.Memory.Data {
		mm := pb.MemoryModule{}
		mm.BaseModuleType = m.BaseModuleType
		mm.CapacityMiB = int32(m.CapacityMiB)
		mm.DeviceLocator = m.DeviceLocator
		mm.Manufacturer = m.Manufacturer
		mm.MemoryDeviceType = m.MemoryDeviceType
		mm.MemoryLocation = &pb.MemoryLocation{Slot: int32(m.MemoryLocation.Slot), Socket: int32(m.MemoryLocation.Socket)}
		mm.Name = m.Name
		mm.Oem = &pb.OemMemory{
			Hpe: &pb.Hpe{
				DIMMStatus:             m.Oem.Hpe.DIMMStatus,
				MinimumVoltageVoltsX10: int32(m.Oem.Hpe.MinimumVoltageVoltsX10),
			},
		}
		mm.OperatingSpeedMhz = int32(m.OperatingSpeedMhz)
		mm.PartNumber = m.PartNumber
		mm.RankCount = int32(m.RankCount)
		mm.Status = &pb.MemoryStatus{Healt: m.Status.Health, State: m.Status.State}
		out.Memory.Data = append(out.Memory.Data, &mm)
	}

	out.Storage = &pb.ServerHardwareLocalStorage{}
	out.Storage.URI = in.Storage.URI
	out.Storage.CollectionState = in.Storage.CollectionState
	out.Storage.Modified = timestamppb.New(in.Storage.Modified)
	out.Storage.Etag = in.Storage.Etag
	out.Storage.Count = int32(in.Storage.Count)
	out.Storage.Name = in.Storage.Name
	for _, st := range in.Storage.Data {
		ls := pb.LocalStorage{}
		ls.AdapterType = st.AdapterType
		ls.CacheMemorySizeMiB = int32(st.CacheMemorySizeMiB)
		ls.CacheModuleSerialNumber = st.CacheModuleSerialNumber
		ls.CacheModuleStatus = &pb.CacheModuleStatus{Health: st.CacheModuleStatus.Health}
		ls.ControllerBoard = &pb.ControllerBoard{Status: &pb.ControllerSatus{Health: st.ControllerBoard.Status.Health, State: st.ControllerBoard.Status.State}}
		ls.EncryptionCspTestPassed = st.EncryptionCspTestPassed
		ls.EncryptionEnabled = st.EncryptionEnabled
		ls.EncryptionSelfTestPassed = st.EncryptionSelfTestPassed
		ls.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: st.FirmwareVersion.Current.VersionString}}
		ls.Location = st.Location
		ls.Model = st.Model
		ls.Name = st.Name
		ls.SerialNumber = st.SerialNumber
		ls.Status = &pb.Status{Health: st.Status.Health, State: st.Status.State}
		for _, ld := range st.LogicalDrives {
			logical := pb.LocalLogicalDrive{}
			logical.CapacityMiB = ld.CapacityMiB
			logical.LogicalDriveEncryption = ld.LogicalDriveEncryption
			logical.LogicalDriveNumber = int32(ld.LogicalDriveNumber)
			logical.LogicalDriveType = ld.LogicalDriveType
			logical.Raid = ld.Raid
			logical.Status = &pb.Status{Health: ld.Status.Health, State: ld.Status.State}
			for _, dd := range ld.DataDrives {
				drive := pb.LogicalDataDrive{}
				drive.CapacityMiB = dd.CapacityMiB
				drive.DiskDriveStatusReasons = append(drive.DiskDriveStatusReasons, dd.DiskDriveStatusReasons...)
				drive.DiskDriveUse = dd.DiskDriveUse
				drive.EncryptedDrive = dd.EncryptedDrive
				drive.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: dd.FirmwareVersion.Current.VersionString}}
				drive.Location = dd.Location
				drive.LocationFormat = dd.LocationFormat
				drive.MediaType = dd.MediaType
				drive.Model = dd.Model
				drive.SerialNumber = dd.SerialNumber
				drive.Status = &pb.Status{Health: dd.Status.Health, State: dd.Status.State}
				logical.DataDrives = append(logical.DataDrives, &drive)
			}
			ls.LogicalDrives = append(ls.LogicalDrives, &logical)
		}
		for _, pd := range st.PhysicalDrives {
			physical := pb.LocalPhysicalDrive{}
			physical.CapacityMiB = pd.CapacityMiB
			physical.DiskDriveStatusReasons = append(physical.DiskDriveStatusReasons, pd.DiskDriveStatusReasons...)
			physical.DiskDriveUse = pd.DiskDriveUse
			physical.EncryptedDrive = pd.EncryptedDrive
			physical.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
			physical.Location = pd.Location
			physical.LocationFormat = pd.LocationFormat
			physical.MediaType = pd.MediaType
			physical.Model = pd.Model
			physical.SerialNumber = pd.SerialNumber
			physical.Status = &pb.Status{Health: pd.Status.Health, State: pd.Status.State}
			ls.PhysicalDrives = append(ls.PhysicalDrives, &physical)
		}
		for _, enc := range st.StorageEnclosures {
			enclosure := pb.LocalStorageEnclosure{}
			enclosure.DriveBayCount = int32(enc.DriveBayCount)
			enclosure.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: enc.FirmwareVersion.Current.VersionString}}
			enclosure.Location = enc.Location
			enclosure.LocationFormat = enc.LocationFormat
			enclosure.Status = &pb.Status{Health: enc.Status.Health, State: enc.Status.State}
			ls.StorageEnclosures = append(ls.StorageEnclosures, &enclosure)
		}
		out.Storage.Data = append(out.Storage.Data, &ls)
	}

	out.Datacenter = &pb.DatacenterInfo{}
	out.Datacenter.Width = int32(in.Datacenter.Width)
	out.Datacenter.Depth = int32(in.Datacenter.Depth)
	out.Datacenter.Currency = in.Datacenter.Currency
	out.Datacenter.DeratingType = in.Datacenter.DeratingType
	out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
	out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
	for _, c := range in.Datacenter.Contents {
		content := pb.Contents{
			X:        c.X,
			Y:        c.Y,
			Rotation: c.Rotation,
		}
		out.Datacenter.Contents = append(out.Datacenter.Contents, &content)
	}
	out.Datacenter.Category = in.Datacenter.Category
	out.Datacenter.Created = timestamppb.New(in.Datacenter.Created)
	out.Datacenter.Modified = timestamppb.New(in.Datacenter.Modified)
	out.Datacenter.ID = in.Datacenter.ID
	out.Datacenter.UUID = in.Datacenter.UUID
	out.Datacenter.DefaultPowerLineVoltage = int32(in.Datacenter.DefaultPowerLineVoltage)
	out.Datacenter.Name = in.Datacenter.Name
	out.Datacenter.State = in.Datacenter.State
	out.Datacenter.Status = in.Datacenter.Status

	out.Rack = &pb.Racks{}
	out.Rack.UHeight = int32(in.Rack.UHeight)
	out.Rack.Width = int32(in.Rack.Width)
	out.Rack.Height = int32(in.Rack.Height)
	out.Rack.Depth = int32(in.Rack.Depth)
	out.Rack.SerialNumber = in.Rack.SerialNumber
	out.Rack.Created = timestamppb.New(in.Rack.Created)
	out.Rack.Modified = timestamppb.New(in.Rack.Modified)
	out.Rack.ID = in.Rack.ID
	out.Rack.UUID = in.Rack.UUID
	out.Rack.Name = in.Rack.Name
	out.Rack.State = in.Rack.State
	out.Rack.Status = in.Rack.Status

	out.EnclosureInfo = &pb.Enclosure{}
	out.EnclosureInfo.Type = in.EnclosureInfo.Type
	out.EnclosureInfo.Category = in.EnclosureInfo.Category
	out.EnclosureInfo.Created = timestamppb.New(in.EnclosureInfo.Created)
	out.EnclosureInfo.Modified = timestamppb.New(in.EnclosureInfo.Modified)
	out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState
	out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason
	out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType
	out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel
	out.EnclosureInfo.UUID = in.EnclosureInfo.UUID
	out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber
	out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber
	out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState
	out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent
	out.EnclosureInfo.DeviceBayCount = int32(in.EnclosureInfo.DeviceBayCount)
	for _, bay := range in.EnclosureInfo.DeviceBays {
		db := pb.DeviceBayInfo{}
		db.Type = bay.Type
		db.BayNumber = int32(bay.BayNumber)
		db.DevicePresence = bay.DevicePresence
		db.CoveredByDevice = bay.CoveredByDevice
		db.URI = bay.URI
		db.Category = bay.Category
		db.DeviceBayType = bay.DeviceBayType
		db.DeviceFormFactor = bay.DeviceFormFactor
		db.BayPowerState = bay.BayPowerState
		db.ChangeState = bay.ChangeState
		out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, &db)
	}
	out.EnclosureInfo.InterconnectBayCount = int32(in.EnclosureInfo.InterconnectBayCount)
	for _, bay := range in.EnclosureInfo.InterconnectBays {
		ib := pb.InterconnectBayInfo{}
		ib.BayNumber = int32(bay.BayNumber)
		ib.InterconnectModel = bay.InterconnectModel
		ib.SerialNumber = bay.SerialNumber
		ib.InterconnectBayType = bay.InterconnectBayType
		ib.ChangeState = bay.ChangeState
		ib.BayPowerState = bay.BayPowerState
		out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, &ib)
	}
	out.EnclosureInfo.FanBayCount = int32(in.EnclosureInfo.FanBayCount)
	for _, bay := range in.EnclosureInfo.FanBays {
		fb := pb.FanBayInfo{}
		fb.BayNumber = int32(bay.BayNumber)
		fb.DevicePresence = bay.DevicePresence
		fb.DeviceRequired = bay.DeviceRequired
		fb.Status = bay.Status
		fb.Model = bay.Model
		fb.PartNumber = bay.PartNumber
		fb.SparePartNumber = bay.SparePartNumber
		fb.FanBayType = bay.FanBayType
		fb.ChangeState = bay.ChangeState
		fb.State = bay.State
		out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, &fb)
	}
	out.EnclosureInfo.PowerSupplyBayCount = int32(in.EnclosureInfo.PowerSupplyBayCount)
	for _, bay := range in.EnclosureInfo.PowerSupplyBays {
		ps := pb.PowerSupplyBayInfo{}
		ps.BayNumber = int32(bay.BayNumber)
		ps.DevicePresence = bay.DevicePresence
		ps.Status = bay.Status
		ps.Model = bay.Model
		ps.SerialNumber = bay.SerialNumber
		ps.PartNumber = bay.PartNumber
		ps.SparePartNumber = bay.SparePartNumber
		ps.PowerSupplyBayType = bay.PowerSupplyBayType
		ps.ChangeState = bay.ChangeState
		out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, &ps)
	}
	out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged
	out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware
	for _, bay := range in.EnclosureInfo.ManagerBays {
		mb := pb.ManagerBayInfo{}
		mb.BayNumber = int32(bay.BayNumber)
		mb.ManagerType = bay.ManagerType
		mb.BayPowerState = bay.BayPowerState
		mb.FwVersion = bay.FwVersion
		mb.DevicePresence = bay.DevicePresence
		mb.Role = bay.Role
		mb.IPAddress = bay.IPAddress
		mb.ChangeState = bay.ChangeState
		mb.FwBuildDate = bay.FwBuildDate
		mb.DhcpIpv6Enable = bay.DhcpIpv6Enable
		mb.FqdnHostName = bay.FqdnHostName
		mb.DhcpEnable = bay.DhcpEnable
		out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, &mb)
	}
	out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState
	out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState
	out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType
	out.EnclosureInfo.Status = in.EnclosureInfo.Status
	out.EnclosureInfo.Name = in.EnclosureInfo.Name
	out.EnclosureInfo.State = in.EnclosureInfo.State
	out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP
	out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP
	out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag
	out.EnclosureInfo.RackName = in.EnclosureInfo.RackName
	out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID
	out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName
	out.EnclosureInfo.VcmMode = fmt.Sprint(in.EnclosureInfo.VcmMode)
	out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL
	out.EnclosureInfo.OaBays = int32(in.EnclosureInfo.OaBays)
	out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState

	out.BayInfo = &pb.DeviceBayInfo{}
	out.BayInfo.Type = in.BayInfo.Type
	out.BayInfo.BayNumber = int32(in.BayInfo.BayNumber)
	out.BayInfo.DevicePresence = in.BayInfo.DevicePresence
	out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice
	out.BayInfo.URI = in.BayInfo.URI
	out.BayInfo.Category = in.BayInfo.Category
	out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType
	out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor
	out.BayInfo.BayPowerState = in.BayInfo.BayPowerState
	out.BayInfo.ChangeState = in.BayInfo.ChangeState
	out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile
	out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile
	out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile
	out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile
}

// MessageToServerHardware выполняет обратную конвертацию gRPC-сообщения pb.ServerHardware в доменную модель oneview.ServerHardware.
// Принимает in и out; при in == nil или out == nil ничего не делает.
// В отличие от ServerHardwareToMessage не выделяет вложенные структуры out.Base/out.ProcessorInfo и т.п. заново — вызывающий код должен
// гарантировать, что out и его вложенные поля уже инициализированы, иначе возможен nil pointer dereference.
// Примечание: в grpcserver эта функция сейчас не используется (симметричный конвертер есть в grpcclient), оставлена для возможного будущего использования на стороне сервера.
func MessageToServerHardware(in *pb.ServerHardware, out *oneview.ServerHardware) {
	if in == nil || out == nil {
		return
	}
	out.Base.Hostname = in.Base.Hostname
	out.Base.HostOsType = int(in.Base.HostOsType)
	out.Base.MaintenanceMode = in.Base.MaintenanceMode
	out.Base.MemoryMb = int(in.Base.MemoryMb)
	out.Base.Model = in.Base.Model
	out.Base.Name = in.Base.Name
	out.Base.PartNumber = in.Base.PartNumber
	out.Base.Position = int(in.Base.Position)
	out.Base.PowerState = in.Base.PowerState
	out.Base.ProcessorCoreCount = int(in.Base.ProcessorCoreCount)
	out.Base.ProcessorCount = int(in.Base.ProcessorCount)
	out.Base.ProcessorSpeedMhz = int(in.Base.ProcessorSpeedMhz)
	out.Base.ProcessorType = in.Base.ProcessorType
	out.Base.SerialNumber = utils.Nstring(in.Base.SerialNumber)
	out.Base.VirtualSerialNumber = utils.Nstring(in.Base.VirtualSerialNumber)
	out.Base.FormFactor = in.Base.FormFactor
	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = utils.Nstring(in.Base.UUID)
	out.OperatingSystem = in.Base.OperatingSystem

	if in.Memory != nil {
		out.Memory.URI = in.Memory.URI
		out.Memory.CollectionState = in.Memory.CollectionState
		out.Memory.Modified = in.Memory.Modified.AsTime()
		out.Memory.Etag = in.Memory.Etag
		out.Memory.Count = int(in.Memory.Count)
		out.Memory.Name = in.Memory.Name
		for _, m := range in.Memory.Data {
			mm := oneview.MemoryModule{}
			mm.BaseModuleType = m.BaseModuleType
			mm.CapacityMiB = int(m.CapacityMiB)
			mm.DeviceLocator = m.DeviceLocator
			mm.Manufacturer = m.Manufacturer
			mm.MemoryDeviceType = m.MemoryDeviceType
			if m.MemoryLocation != nil {
				mm.MemoryLocation = oneview.MemoryLocation{Slot: int(m.MemoryLocation.Slot), Socket: int(m.MemoryLocation.Socket)}
			}
			mm.Name = m.Name
			if m.Oem != nil && m.Oem.Hpe != nil {
				mm.Oem.Hpe.DIMMStatus = m.Oem.Hpe.DIMMStatus
				mm.Oem.Hpe.MinimumVoltageVoltsX10 = int(m.Oem.Hpe.MinimumVoltageVoltsX10)
			}
			mm.OperatingSpeedMhz = int(m.OperatingSpeedMhz)
			mm.PartNumber = m.PartNumber
			mm.RankCount = int(m.RankCount)
			if m.Status != nil {
				mm.Status = oneview.MemoryStatus{Health: m.Status.Healt, State: m.Status.State}
			}
			out.Memory.Data = append(out.Memory.Data, mm)
		}
	}

	if in.Storage != nil {
		out.Storage.URI = in.Storage.URI
		out.Storage.CollectionState = in.Storage.CollectionState
		out.Storage.Modified = in.Storage.Modified.AsTime()
		out.Storage.Etag = in.Storage.Etag
		out.Storage.Count = int(in.Storage.Count)
		out.Storage.Name = in.Storage.Name
		for _, st := range in.Storage.Data {
			ls := oneview.LocalStorage{}
			ls.AdapterType = st.AdapterType
			ls.CacheMemorySizeMiB = int(st.CacheMemorySizeMiB)
			ls.CacheModuleSerialNumber = st.CacheModuleSerialNumber
			if st.CacheModuleStatus != nil {
				ls.CacheModuleStatus = oneview.CacheModuleStatus{Health: st.CacheModuleStatus.Health}
			}
			if st.ControllerBoard != nil && st.ControllerBoard.Status != nil {
				ls.ControllerBoard = oneview.ControllerBoard{Status: oneview.ControllerSatus{Health: st.ControllerBoard.Status.Health, State: st.ControllerBoard.Status.State}}
			}
			ls.EncryptionCspTestPassed = st.EncryptionCspTestPassed
			ls.EncryptionEnabled = st.EncryptionEnabled
			ls.EncryptionSelfTestPassed = st.EncryptionSelfTestPassed
			if st.FirmwareVersion != nil && st.FirmwareVersion.Current != nil {
				ls.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: st.FirmwareVersion.Current.VersionString}}
			}
			ls.Location = st.Location
			ls.Model = st.Model
			ls.Name = st.Name
			ls.SerialNumber = st.SerialNumber
			if st.Status != nil {
				ls.Status = oneview.Status{Health: st.Status.Health, State: st.Status.State}
			}
			for _, ld := range st.LogicalDrives {
				logical := oneview.LocalLogicalDrive{}
				logical.CapacityMiB = ld.CapacityMiB
				logical.LogicalDriveEncryption = ld.LogicalDriveEncryption
				logical.LogicalDriveNumber = int(ld.LogicalDriveNumber)
				logical.LogicalDriveType = ld.LogicalDriveType
				logical.Raid = ld.Raid
				logical.Status = oneview.Status{}
				if ld.Status != nil {
					logical.Status = oneview.Status{Health: ld.Status.Health, State: ld.Status.State}
				}
				for _, dd := range ld.DataDrives {
					drive := oneview.LogicalDataDrive{}
					drive.CapacityMiB = dd.CapacityMiB
					drive.DiskDriveStatusReasons = append(drive.DiskDriveStatusReasons, dd.DiskDriveStatusReasons...)
					drive.DiskDriveUse = dd.DiskDriveUse
					drive.EncryptedDrive = dd.EncryptedDrive
					if dd.FirmwareVersion != nil && dd.FirmwareVersion.Current != nil {
						drive.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: dd.FirmwareVersion.Current.VersionString}}
					}
					drive.Location = dd.Location
					drive.LocationFormat = dd.LocationFormat
					drive.MediaType = dd.MediaType
					drive.Model = dd.Model
					drive.SerialNumber = dd.SerialNumber
					if dd.Status != nil {
						drive.Status = oneview.Status{Health: dd.Status.Health, State: dd.Status.State}
					}
					logical.DataDrives = append(logical.DataDrives, drive)
				}
				ls.LogicalDrives = append(ls.LogicalDrives, logical)
			}
			for _, pd := range st.PhysicalDrives {
				physical := oneview.LocalPhysicalDrive{}
				physical.CapacityMiB = pd.CapacityMiB
				physical.DiskDriveStatusReasons = append(physical.DiskDriveStatusReasons, pd.DiskDriveStatusReasons...)
				physical.DiskDriveUse = pd.DiskDriveUse
				physical.EncryptedDrive = pd.EncryptedDrive
				if pd.FirmwareVersion != nil && pd.FirmwareVersion.Current != nil {
					physical.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
				}
				physical.Location = pd.Location
				physical.LocationFormat = pd.LocationFormat
				physical.MediaType = pd.MediaType
				physical.Model = pd.Model
				physical.SerialNumber = pd.SerialNumber
				if pd.Status != nil {
					physical.Status = oneview.Status{Health: pd.Status.Health, State: pd.Status.State}
				}
				ls.PhysicalDrives = append(ls.PhysicalDrives, physical)
			}
			for _, enc := range st.StorageEnclosures {
				enclosure := oneview.LocalStorageEnclosure{}
				enclosure.DriveBayCount = int(enc.DriveBayCount)
				if enc.FirmwareVersion != nil && enc.FirmwareVersion.Current != nil {
					enclosure.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: enc.FirmwareVersion.Current.VersionString}}
				}
				enclosure.Location = enc.Location
				enclosure.LocationFormat = enc.LocationFormat
				if enc.Status != nil {
					enclosure.Status = oneview.Status{Health: enc.Status.Health, State: enc.Status.State}
				}
				ls.StorageEnclosures = append(ls.StorageEnclosures, enclosure)
			}
			out.Storage.Data = append(out.Storage.Data, ls)
		}
	}

	if in.Datacenter != nil {
		out.Datacenter.Width = int(in.Datacenter.Width)
		out.Datacenter.Depth = int(in.Datacenter.Depth)
		out.Datacenter.Currency = in.Datacenter.Currency
		out.Datacenter.DeratingType = in.Datacenter.DeratingType
		out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
		out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
		for _, c := range in.Datacenter.Contents {
			content := oneview.Content{
				X:        c.X,
				Y:        c.Y,
				Rotation: c.Rotation,
			}
			out.Datacenter.Contents = append(out.Datacenter.Contents, content)
		}
		out.Datacenter.Category = in.Datacenter.Category
		if in.Datacenter.Created != nil {
			out.Datacenter.Created = in.Datacenter.Created.AsTime()
		}
		if in.Datacenter.Modified != nil {
			out.Datacenter.Modified = in.Datacenter.Modified.AsTime()
		}
		out.Datacenter.ID = in.Datacenter.ID
		out.Datacenter.UUID = in.Datacenter.UUID
		out.Datacenter.DefaultPowerLineVoltage = int(in.Datacenter.DefaultPowerLineVoltage)
		out.Datacenter.Name = in.Datacenter.Name
		out.Datacenter.State = in.Datacenter.State
		out.Datacenter.Status = in.Datacenter.Status
	}

	if in.Rack != nil {
		out.Rack.UHeight = int(in.Rack.UHeight)
		out.Rack.Width = int(in.Rack.Width)
		out.Rack.Height = int(in.Rack.Height)
		out.Rack.Depth = int(in.Rack.Depth)
		out.Rack.SerialNumber = in.Rack.SerialNumber
		if in.Rack.Created != nil {
			out.Rack.Created = in.Rack.Created.AsTime()
		}
		if in.Rack.Modified != nil {
			out.Rack.Modified = in.Rack.Modified.AsTime()
		}
		out.Rack.ID = in.Rack.ID
		out.Rack.UUID = in.Rack.UUID
		out.Rack.Name = in.Rack.Name
		out.Rack.State = in.Rack.State
		out.Rack.Status = in.Rack.Status
	}

	if in.EnclosureInfo != nil {
		out.EnclosureInfo.Type = in.EnclosureInfo.Type
		out.EnclosureInfo.Category = in.EnclosureInfo.Category
		if in.EnclosureInfo.Created != nil {
			out.EnclosureInfo.Created = in.EnclosureInfo.Created.AsTime()
		}
		if in.EnclosureInfo.Modified != nil {
			out.EnclosureInfo.Modified = in.EnclosureInfo.Modified.AsTime()
		}
		out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState
		out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason
		out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType
		out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel
		out.EnclosureInfo.UUID = in.EnclosureInfo.UUID
		out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber
		out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber
		out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState
		out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent
		out.EnclosureInfo.DeviceBayCount = int(in.EnclosureInfo.DeviceBayCount)
		for _, bay := range in.EnclosureInfo.DeviceBays {
			db := oneview.DeviceBayInfo{}
			db.Type = bay.Type
			db.BayNumber = int(bay.BayNumber)
			db.DevicePresence = bay.DevicePresence
			db.CoveredByDevice = bay.CoveredByDevice
			db.URI = bay.URI
			db.Category = bay.Category
			db.DeviceBayType = bay.DeviceBayType
			db.DeviceFormFactor = bay.DeviceFormFactor
			db.BayPowerState = bay.BayPowerState
			db.ChangeState = bay.ChangeState
			out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, db)
		}
		out.EnclosureInfo.InterconnectBayCount = int(in.EnclosureInfo.InterconnectBayCount)
		for _, bay := range in.EnclosureInfo.InterconnectBays {
			ib := oneview.InterconnectBayInfo{}
			ib.BayNumber = int(bay.BayNumber)
			ib.InterconnectModel = bay.InterconnectModel
			ib.SerialNumber = bay.SerialNumber
			ib.InterconnectBayType = bay.InterconnectBayType
			ib.ChangeState = bay.ChangeState
			ib.BayPowerState = bay.BayPowerState
			out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, ib)
		}
		out.EnclosureInfo.FanBayCount = int(in.EnclosureInfo.FanBayCount)
		for _, bay := range in.EnclosureInfo.FanBays {
			fb := oneview.FanBayInfo{}
			fb.BayNumber = int(bay.BayNumber)
			fb.DevicePresence = bay.DevicePresence
			fb.DeviceRequired = bay.DeviceRequired
			fb.Status = bay.Status
			fb.Model = bay.Model
			fb.PartNumber = bay.PartNumber
			fb.SparePartNumber = bay.SparePartNumber
			fb.FanBayType = bay.FanBayType
			fb.ChangeState = bay.ChangeState
			fb.State = bay.State
			out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, fb)
		}
		out.EnclosureInfo.PowerSupplyBayCount = int(in.EnclosureInfo.PowerSupplyBayCount)
		for _, bay := range in.EnclosureInfo.PowerSupplyBays {
			ps := oneview.PowerSupplyBayInfo{}
			ps.BayNumber = int(bay.BayNumber)
			ps.DevicePresence = bay.DevicePresence
			ps.Status = bay.Status
			ps.Model = bay.Model
			ps.SerialNumber = bay.SerialNumber
			ps.PartNumber = bay.PartNumber
			ps.SparePartNumber = bay.SparePartNumber
			ps.PowerSupplyBayType = bay.PowerSupplyBayType
			ps.ChangeState = bay.ChangeState
			out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, ps)
		}
		out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged
		out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware
		for _, bay := range in.EnclosureInfo.ManagerBays {
			mb := oneview.ManagerBayInfo{}
			mb.BayNumber = int(bay.BayNumber)
			mb.ManagerType = bay.ManagerType
			mb.BayPowerState = bay.BayPowerState
			mb.FwVersion = bay.FwVersion
			mb.DevicePresence = bay.DevicePresence
			mb.Role = bay.Role
			mb.IPAddress = bay.IPAddress
			mb.ChangeState = bay.ChangeState
			mb.FwBuildDate = bay.FwBuildDate
			mb.DhcpIpv6Enable = bay.DhcpIpv6Enable
			mb.FqdnHostName = bay.FqdnHostName
			mb.DhcpEnable = bay.DhcpEnable
			out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, mb)
		}
		out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState
		out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState
		out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType
		out.EnclosureInfo.Status = in.EnclosureInfo.Status
		out.EnclosureInfo.Name = in.EnclosureInfo.Name
		out.EnclosureInfo.State = in.EnclosureInfo.State
		out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP
		out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP
		out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag
		out.EnclosureInfo.RackName = in.EnclosureInfo.RackName
		out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID
		out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName
		if b, err := strconv.ParseBool(in.EnclosureInfo.VcmMode); err == nil {
			out.EnclosureInfo.VcmMode = b
		}
		out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL
		out.EnclosureInfo.OaBays = int(in.EnclosureInfo.OaBays)
		out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState
	}

	if in.BayInfo != nil {
		out.BayInfo.Type = in.BayInfo.Type
		out.BayInfo.BayNumber = int(in.BayInfo.BayNumber)
		out.BayInfo.DevicePresence = in.BayInfo.DevicePresence
		out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice
		out.BayInfo.URI = in.BayInfo.URI
		out.BayInfo.Category = in.BayInfo.Category
		out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType
		out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor
		out.BayInfo.BayPowerState = in.BayInfo.BayPowerState
		out.BayInfo.ChangeState = in.BayInfo.ChangeState
		out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile
		out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile
		out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile
		out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile
	}
}

// ClusterSprRecordsToMessage конвертирует models.ClusterSprRecords в gRPC-сообщение pb.ClusterSprRecord.
// Принимает in и out; при in == nil или out == nil ничего не делает.
