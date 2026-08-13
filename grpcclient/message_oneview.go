package grpcclient

import (
	pb "loadmon/protocol"

	oneview "github.com/spa-nsk/oneview2"
)

// ServerHardwareToMessage конвертирует доменную модель oneview2.Server
// (пакет github.com/spa-nsk/oneview2, см. loadmon/oneview2/collect.go) в
// существующее gRPC-сообщение pb.ServerHardware. Сообщение (protocol/*.proto)
// не менялось при переходе со старого пакета loadmon/oneview на oneview2 -
// это осознанное решение, чтобы не требовать пересборки protoc на стороне
// пользователя; поэтому заполняются только те поля pb.ServerHardware, для
// которых в oneview2.Server есть прямой источник данных. Идентична
// grpcserver.ServerHardwareToMessage - дублируется здесь, так как клиент и
// сервер исторически не делят пакет с конвертерами.
func ServerHardwareToMessage(in *oneview.Server, out *pb.ServerHardware) {
	if in == nil || out == nil {
		return
	}

	out.Base = &pb.OVServerHardware{}
	out.Base.Hostname = in.Identity.MpHostName
	out.Base.MemoryMb = int32(oneview.TotalMemoryMb(in))
	out.Base.Model = in.Identity.Model
	out.Base.Name = in.Identity.Name
	out.Base.PartNumber = in.Identity.PartNumber
	out.Base.Position = int32(in.Identity.Position)
	out.Base.PowerState = in.Identity.PowerState
	out.Base.ProcessorCount = int32(len(in.Processors))
	if len(in.Processors) > 0 {
		p := in.Processors[0]
		out.Base.ProcessorType = p.Model
		out.Base.ProcessorCoreCount = int32(p.TotalCores)
		speed := p.OperatingSpeedMHz
		if speed == 0 {
			speed = p.MaxSpeedMHz
		}
		out.Base.ProcessorSpeedMhz = int32(speed)
	}
	out.Base.SerialNumber = in.Identity.SerialNumber
	// oneview2.ServerIdentity не хранит виртуальный серийный номер (в отличие
	// от старого oneview.ServerHardware.Base.VirtualSerialNumber).
	out.Base.VirtualSerialNumber = ""
	out.Base.FormFactor = in.Identity.FormFactor
	out.Base.ServerName = in.Identity.ServerName
	out.Base.ShortModel = in.Identity.ShortModel
	out.Base.Status = in.Identity.Status
	out.Base.UUID = in.Identity.UUID
	out.Base.OperatingSystem = in.Identity.OperatingSystem
	// oneview2.ServerIdentity не хранит признак режима обслуживания (в
	// отличие от старого Base.MaintenanceMode) и тип хостовой ОС
	// (HostOsType) - оба остаются нулевыми.

	out.Memory = &pb.ServerHardwareMemory{}
	out.Memory.Count = int32(len(in.Memory))
	for _, m := range in.Memory {
		mm := pb.MemoryModule{}
		mm.BaseModuleType = m.BaseModuleType
		mm.CapacityMiB = int32(m.CapacityMiB)
		mm.DeviceLocator = m.DeviceLocator
		mm.Manufacturer = m.Manufacturer
		mm.MemoryDeviceType = m.MemoryDeviceType
		mm.Name = m.Name
		mm.Oem = &pb.OemMemory{Hpe: &pb.Hpe{DIMMStatus: m.DIMMStatus}}
		mm.OperatingSpeedMhz = int32(m.OperatingSpeedMHz)
		mm.PartNumber = m.PartNumber
		mm.RankCount = int32(m.RankCount)
		mm.Status = &pb.MemoryStatus{Healt: m.Health, State: m.State}
		out.Memory.Data = append(out.Memory.Data, &mm)
	}

	out.Storage = &pb.ServerHardwareLocalStorage{}
	out.Storage.Count = int32(len(in.Controllers))
	for _, c := range in.Controllers {
		ls := pb.LocalStorage{}
		ls.AdapterType = c.AdapterType
		ls.CacheMemorySizeMiB = int32(c.CacheMemorySizeMiB)
		ls.Location = c.Location
		ls.Model = c.Model
		ls.Name = c.Name
		ls.SerialNumber = c.SerialNumber
		ls.Status = &pb.Status{Health: c.Health, State: c.State}
		ls.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: c.FirmwareVersion}}
		for _, v := range c.Volumes {
			logical := pb.LocalLogicalDrive{}
			logical.CapacityMiB = float64(v.CapacityMiB)
			logical.LogicalDriveNumber = int32(v.Number)
			logical.LogicalDriveType = v.MediaType
			logical.Raid = v.RAID
			logical.Status = &pb.Status{Health: v.Health, State: v.State}
			ls.LogicalDrives = append(ls.LogicalDrives, &logical)
		}
		for _, d := range c.Drives {
			physical := pb.LocalPhysicalDrive{}
			physical.CapacityMiB = float64(d.CapacityMiB)
			physical.DiskDriveUse = d.DiskDriveUse
			physical.EncryptedDrive = d.Encrypted
			physical.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: d.FirmwareVersion}}
			physical.Location = d.Location
			physical.MediaType = d.MediaType
			physical.Model = d.Model
			physical.SerialNumber = d.SerialNumber
			physical.Status = &pb.Status{Health: d.Health, State: d.State}
			ls.PhysicalDrives = append(ls.PhysicalDrives, &physical)
		}
		out.Storage.Data = append(out.Storage.Data, &ls)
	}

	out.Datacenter = &pb.DatacenterInfo{}
	out.Datacenter.Name = in.Datacenter.Name
	out.Datacenter.UUID = in.Datacenter.UUID
	out.Datacenter.ID = in.Datacenter.URI
	out.Datacenter.Status = in.Datacenter.Status
	out.Datacenter.State = in.Datacenter.State
	out.Datacenter.Width = int32(in.Datacenter.Width)
	out.Datacenter.Currency = in.Datacenter.Currency

	out.Rack = &pb.Racks{}
	out.Rack.Name = in.Rack.Name
	out.Rack.UUID = in.Rack.UUID
	out.Rack.ID = in.Rack.URI
	out.Rack.SerialNumber = in.Rack.SerialNumber
	out.Rack.UHeight = int32(in.Rack.UHeight)
	out.Rack.Status = in.Rack.Status
	out.Rack.State = in.Rack.State

	out.EnclosureInfo = &pb.Enclosure{}
	out.EnclosureInfo.Name = in.EnclosureInfo.Name
	out.EnclosureInfo.UUID = in.EnclosureInfo.UUID
	out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber
	out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.Model
	out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType
	out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber
	out.EnclosureInfo.RackName = in.EnclosureInfo.RackName
	out.EnclosureInfo.DeviceBayCount = int32(in.EnclosureInfo.DeviceBayCount)
	out.EnclosureInfo.Status = in.EnclosureInfo.Status
	out.EnclosureInfo.State = in.EnclosureInfo.State
	out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOAPreferredIP

	out.BayInfo = &pb.DeviceBayInfo{}
	out.BayInfo.BayNumber = int32(in.BayInfo.BayNumber)
	out.BayInfo.DevicePresence = in.BayInfo.DevicePresence
	out.BayInfo.URI = in.BayInfo.DeviceURI
	out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType
	out.BayInfo.BayPowerState = in.BayInfo.PowerState
}

// MessageToServerHardware конвертирует gRPC-сообщение pb.ServerHardware обратно в доменную модель oneview2.Server. При in == nil или out == nil ничего не делает.
func MessageToServerHardware(in *pb.ServerHardware, out *oneview.Server) {
	if in == nil || out == nil {
		return
	}

	if in.Base != nil {
		out.Identity.MpHostName = in.Base.Hostname
		out.Identity.Model = in.Base.Model
		out.Identity.Name = in.Base.Name
		out.Identity.PartNumber = in.Base.PartNumber
		out.Identity.Position = int(in.Base.Position)
		out.Identity.PowerState = in.Base.PowerState
		out.Identity.SerialNumber = in.Base.SerialNumber
		out.Identity.FormFactor = in.Base.FormFactor
		out.Identity.ServerName = in.Base.ServerName
		out.Identity.ShortModel = in.Base.ShortModel
		out.Identity.Status = in.Base.Status
		out.Identity.UUID = in.Base.UUID
		out.Identity.OperatingSystem = in.Base.OperatingSystem
		if in.Base.ProcessorType != "" || in.Base.ProcessorCoreCount > 0 || in.Base.ProcessorSpeedMhz > 0 {
			out.Processors = append(out.Processors, oneview.ProcessorInfo{
				Model:             in.Base.ProcessorType,
				TotalCores:        int(in.Base.ProcessorCoreCount),
				OperatingSpeedMHz: int(in.Base.ProcessorSpeedMhz),
			})
		}
	}

	if in.Memory != nil {
		for _, m := range in.Memory.Data {
			mm := oneview.MemoryModule{}
			mm.BaseModuleType = m.BaseModuleType
			mm.CapacityMiB = int(m.CapacityMiB)
			mm.DeviceLocator = m.DeviceLocator
			mm.Manufacturer = m.Manufacturer
			mm.MemoryDeviceType = m.MemoryDeviceType
			mm.Name = m.Name
			if m.Oem != nil && m.Oem.Hpe != nil {
				mm.DIMMStatus = m.Oem.Hpe.DIMMStatus
			}
			mm.OperatingSpeedMHz = int(m.OperatingSpeedMhz)
			mm.PartNumber = m.PartNumber
			mm.RankCount = int(m.RankCount)
			if m.Status != nil {
				mm.Health = m.Status.Healt
				mm.State = m.Status.State
			}
			out.Memory = append(out.Memory, mm)
		}
	}

	if in.Storage != nil {
		for _, st := range in.Storage.Data {
			c := oneview.StorageController{}
			c.AdapterType = st.AdapterType
			c.CacheMemorySizeMiB = int(st.CacheMemorySizeMiB)
			c.Location = st.Location
			c.Model = st.Model
			c.Name = st.Name
			c.SerialNumber = st.SerialNumber
			if st.Status != nil {
				c.Health = st.Status.Health
				c.State = st.Status.State
			}
			if st.FirmwareVersion != nil && st.FirmwareVersion.Current != nil {
				c.FirmwareVersion = st.FirmwareVersion.Current.VersionString
			}
			for _, ld := range st.LogicalDrives {
				v := oneview.LogicalDrive{
					Number:      int(ld.LogicalDriveNumber),
					RAID:        ld.Raid,
					MediaType:   ld.LogicalDriveType,
					CapacityMiB: int(ld.CapacityMiB),
				}
				if ld.Status != nil {
					v.Health = ld.Status.Health
					v.State = ld.Status.State
				}
				c.Volumes = append(c.Volumes, v)
			}
			for _, pd := range st.PhysicalDrives {
				d := oneview.PhysicalDrive{
					Location:     pd.Location,
					Model:        pd.Model,
					SerialNumber: pd.SerialNumber,
					MediaType:    pd.MediaType,
					CapacityMiB:  int(pd.CapacityMiB),
					DiskDriveUse: pd.DiskDriveUse,
					Encrypted:    pd.EncryptedDrive,
				}
				if pd.FirmwareVersion != nil && pd.FirmwareVersion.Current != nil {
					d.FirmwareVersion = pd.FirmwareVersion.Current.VersionString
				}
				if pd.Status != nil {
					d.Health = pd.Status.Health
					d.State = pd.Status.State
				}
				c.Drives = append(c.Drives, d)
			}
			out.Controllers = append(out.Controllers, c)
		}
	}

	if in.Datacenter != nil {
		out.Datacenter.Name = in.Datacenter.Name
		out.Datacenter.UUID = in.Datacenter.UUID
		out.Datacenter.URI = in.Datacenter.ID
		out.Datacenter.Status = in.Datacenter.Status
		out.Datacenter.State = in.Datacenter.State
		out.Datacenter.Width = int(in.Datacenter.Width)
		out.Datacenter.Currency = in.Datacenter.Currency
	}

	if in.Rack != nil {
		out.Rack.Name = in.Rack.Name
		out.Rack.UUID = in.Rack.UUID
		out.Rack.URI = in.Rack.ID
		out.Rack.SerialNumber = in.Rack.SerialNumber
		out.Rack.UHeight = int(in.Rack.UHeight)
		out.Rack.Status = in.Rack.Status
		out.Rack.State = in.Rack.State
	}

	if in.EnclosureInfo != nil {
		out.EnclosureInfo.Name = in.EnclosureInfo.Name
		out.EnclosureInfo.UUID = in.EnclosureInfo.UUID
		out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber
		out.EnclosureInfo.Model = in.EnclosureInfo.EnclosureModel
		out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType
		out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber
		out.EnclosureInfo.RackName = in.EnclosureInfo.RackName
		out.EnclosureInfo.DeviceBayCount = int(in.EnclosureInfo.DeviceBayCount)
		out.EnclosureInfo.Status = in.EnclosureInfo.Status
		out.EnclosureInfo.State = in.EnclosureInfo.State
		out.EnclosureInfo.ActiveOAPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP
	}

	if in.BayInfo != nil {
		out.BayInfo.BayNumber = int(in.BayInfo.BayNumber)
		out.BayInfo.DevicePresence = in.BayInfo.DevicePresence
		out.BayInfo.DeviceURI = in.BayInfo.URI
		out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType
		out.BayInfo.PowerState = in.BayInfo.BayPowerState
	}
}


// MessageToClusterSprRecords конвертирует gRPC-сообщение pb.ClusterSprRecord (запись справочника кластеров) в доменную модель models.ClusterSprRecords. При in == nil или out == nil ничего не делает.
