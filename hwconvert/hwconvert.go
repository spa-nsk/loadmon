// Package hwconvert переводит доменную модель oneview2.Server в gRPC-сообщение
// pb.ServerHardware и обратно (полная инвентаризация oneview2).
package hwconvert

import (
	pb "loadmon/protocol"

	oneview "github.com/spa-nsk/oneview2"
)

// resolveProcessorSpeedMHz подставляет частоту CPU, когда OneView API 7400 (и ряд
// других appliance-версий) отдаёт OperatingSpeedMHz/MaxSpeedMHz нулём в подресурсе
// processors, хотя сводное поле processorSpeedMhz на server-hardware заполнено.
func resolveProcessorSpeedMHz(operating, max, fallbackMHz int) (int, int) {
	op, mx := operating, max
	if op == 0 {
		if mx > 0 {
			op = mx
		} else if fallbackMHz > 0 {
			op = fallbackMHz
		}
	}
	if mx == 0 && fallbackMHz > 0 {
		mx = fallbackMHz
	}
	return op, mx
}

func processorSpeedFallbackMHz(processors []oneview.ProcessorInfo) int {
	for _, p := range processors {
		if p.OperatingSpeedMHz > 0 {
			return p.OperatingSpeedMHz
		}
		if p.MaxSpeedMHz > 0 {
			return p.MaxSpeedMHz
		}
	}
	return 0
}

// ServerHardwareToMessage заполняет pb.ServerHardware из oneview2.Server.
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
	speedFallback := processorSpeedFallbackMHz(in.Processors)
	if len(in.Processors) > 0 {
		p := in.Processors[0]
		out.Base.ProcessorType = p.Model
		out.Base.ProcessorCoreCount = int32(p.TotalCores)
		speed, _ := resolveProcessorSpeedMHz(p.OperatingSpeedMHz, p.MaxSpeedMHz, speedFallback)
		out.Base.ProcessorSpeedMhz = int32(speed)
	}
	out.Base.SerialNumber = in.Identity.SerialNumber
	out.Base.VirtualSerialNumber = in.Identity.VirtualSerialNumber
	out.Base.MaintenanceMode = in.Identity.MaintenanceMode
	out.Base.FormFactor = in.Identity.FormFactor
	out.Base.ServerName = in.Identity.ServerName
	out.Base.ShortModel = in.Identity.ShortModel
	out.Base.Status = in.Identity.Status
	out.Base.UUID = in.Identity.UUID
	out.Base.OperatingSystem = in.Identity.OperatingSystem
	out.Base.State = in.Identity.State
	out.Base.AssetTag = in.Identity.AssetTag
	out.Base.URI = in.Identity.URI
	out.Base.RomVersion = in.Identity.RomVersion
	out.Base.MpModel = in.Identity.MpModel
	out.Base.MpFirmwareVersion = in.Identity.MpFirmwareVersion
	out.Base.MpIpAddress = in.Identity.MpIPAddress
	out.Base.MpHostName = in.Identity.MpHostName
	out.Base.LocationURI = in.Identity.LocationURI
	out.Base.ServerProfileURI = in.Identity.ServerProfileURI
	out.Base.ServerHardwareTypeURI = in.Identity.ServerHardwareTypeURI
	out.Base.ServerHardwareTypeName = in.Identity.ServerHardwareTypeName
	out.Base.ApplianceLocation = in.Identity.ApplianceLocation
	out.Base.ApplianceName = in.Identity.ApplianceName
	out.Base.HostOsTypeName = in.Identity.HostOsType
	out.Base.Platform = in.Identity.Platform

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
		mm.SerialNumber = m.SerialNumber
		mm.MemoryType = m.MemoryType
		mm.ErrorCorrection = m.ErrorCorrection
		mm.Socket = int32(m.Socket)
		mm.Channel = int32(m.Channel)
		mm.Slot = int32(m.Slot)
		mm.Attributes = append([]string(nil), m.Attributes...)
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
	out.Rack.TopUSlot = int32(in.Rack.TopUSlot)
	out.Rack.MountUHeight = int32(in.Rack.MountUHeight)
	out.Rack.MountLocation = in.Rack.MountLocation

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

	for _, p := range in.Processors {
		op, mx := resolveProcessorSpeedMHz(p.OperatingSpeedMHz, p.MaxSpeedMHz, speedFallback)
		out.Processors = append(out.Processors, &pb.OVProcessorInfo{
			ID:                p.ID,
			Socket:            p.Socket,
			Model:             p.Model,
			Manufacturer:      p.Manufacturer,
			TotalCores:        int32(p.TotalCores),
			TotalThreads:      int32(p.TotalThreads),
			MaxSpeedMHz:       int32(mx),
			OperatingSpeedMHz: int32(op),
			Health:            p.Health,
			State:             p.State,
		})
	}
	for _, np := range in.NetworkPorts {
		out.NetworkPorts = append(out.NetworkPorts, &pb.OVNetworkPort{
			DeviceName:      np.DeviceName,
			DeviceSlot:      np.DeviceSlot,
			PortNumber:      int32(np.PortNumber),
			Type:            np.Type,
			MAC:             np.MAC,
			WWN:             np.WWN,
			VirtualMACs:     append([]string(nil), np.VirtualMACs...),
			InterconnectURI: np.Interconnect,
		})
	}
	for _, d := range in.Devices {
		out.Devices = append(out.Devices, &pb.OVPCIDevice{
			ID:                d.ID,
			Name:              d.Name,
			DeviceType:        d.DeviceType,
			Location:          d.Location,
			Manufacturer:      d.Manufacturer,
			PartNumber:        d.PartNumber,
			ProductPartNumber: d.ProductPartNumber,
			SerialNumber:      d.SerialNumber,
			FirmwareVersion:   d.FirmwareVersion,
			Health:            d.Health,
			State:             d.State,
		})
	}
	for _, f := range in.Firmware {
		out.Firmware = append(out.Firmware, &pb.OVFirmwareComponent{
			Name:     f.Name,
			Version:  f.Version,
			Location: f.Location,
			State:    f.State,
			Status:   f.Status,
		})
	}
	out.Metadata = &pb.OVServerMetadata{
		APIVersion: int32(in.APIVersion),
		Product:    in.Product,
		Endpoint:   in.Endpoint,
		Warnings:   append([]string(nil), in.Warnings...),
	}
	for _, src := range in.Sources {
		out.Metadata.Sources = append(out.Metadata.Sources, &pb.OVServerSource{
			Endpoint:   src.Endpoint,
			APIVersion: int32(src.APIVersion),
			Product:    src.Product,
		})
	}
}

// MessageToServerHardware заполняет oneview2.Server из pb.ServerHardware.
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
		out.Identity.VirtualSerialNumber = in.Base.VirtualSerialNumber
		out.Identity.MaintenanceMode = in.Base.MaintenanceMode
		out.Identity.FormFactor = in.Base.FormFactor
		out.Identity.ServerName = in.Base.ServerName
		out.Identity.ShortModel = in.Base.ShortModel
		out.Identity.Status = in.Base.Status
		out.Identity.UUID = in.Base.UUID
		out.Identity.OperatingSystem = in.Base.OperatingSystem
		out.Identity.State = in.Base.State
		out.Identity.AssetTag = in.Base.AssetTag
		out.Identity.URI = in.Base.URI
		out.Identity.RomVersion = in.Base.RomVersion
		out.Identity.MpModel = in.Base.MpModel
		out.Identity.MpFirmwareVersion = in.Base.MpFirmwareVersion
		out.Identity.MpIPAddress = in.Base.MpIpAddress
		out.Identity.MpHostName = in.Base.MpHostName
		out.Identity.LocationURI = in.Base.LocationURI
		out.Identity.ServerProfileURI = in.Base.ServerProfileURI
		out.Identity.ServerHardwareTypeURI = in.Base.ServerHardwareTypeURI
		out.Identity.ServerHardwareTypeName = in.Base.ServerHardwareTypeName
		out.Identity.ApplianceLocation = in.Base.ApplianceLocation
		out.Identity.ApplianceName = in.Base.ApplianceName
		out.Identity.HostOsType = in.Base.HostOsTypeName
		out.Identity.Platform = in.Base.Platform
	}

	baseSpeedFallback := 0
	if in.Base != nil {
		baseSpeedFallback = int(in.Base.ProcessorSpeedMhz)
	}
	if len(in.Processors) > 0 {
		for _, p := range in.Processors {
			op, mx := resolveProcessorSpeedMHz(int(p.OperatingSpeedMHz), int(p.MaxSpeedMHz), baseSpeedFallback)
			out.Processors = append(out.Processors, oneview.ProcessorInfo{
				ID:                p.ID,
				Socket:            p.Socket,
				Model:             p.Model,
				Manufacturer:      p.Manufacturer,
				TotalCores:        int(p.TotalCores),
				TotalThreads:      int(p.TotalThreads),
				MaxSpeedMHz:       mx,
				OperatingSpeedMHz: op,
				Health:            p.Health,
				State:             p.State,
			})
		}
	} else if in.Base != nil && (in.Base.ProcessorType != "" || in.Base.ProcessorCoreCount > 0 || in.Base.ProcessorSpeedMhz > 0) {
		out.Processors = append(out.Processors, oneview.ProcessorInfo{
			Model:             in.Base.ProcessorType,
			TotalCores:        int(in.Base.ProcessorCoreCount),
			OperatingSpeedMHz: int(in.Base.ProcessorSpeedMhz),
		})
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
			mm.SerialNumber = m.SerialNumber
			mm.MemoryType = m.MemoryType
			mm.ErrorCorrection = m.ErrorCorrection
			mm.Socket = int(m.Socket)
			mm.Channel = int(m.Channel)
			mm.Slot = int(m.Slot)
			mm.Attributes = append([]string(nil), m.Attributes...)
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
		out.Rack.TopUSlot = int(in.Rack.TopUSlot)
		out.Rack.MountUHeight = int(in.Rack.MountUHeight)
		out.Rack.MountLocation = in.Rack.MountLocation
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

	for _, np := range in.NetworkPorts {
		out.NetworkPorts = append(out.NetworkPorts, oneview.NetworkPortExport{
			DeviceName:   np.DeviceName,
			DeviceSlot:   np.DeviceSlot,
			PortNumber:   int(np.PortNumber),
			Type:         np.Type,
			MAC:          np.MAC,
			WWN:          np.WWN,
			VirtualMACs:  append([]string(nil), np.VirtualMACs...),
			Interconnect: np.InterconnectURI,
		})
	}
	for _, d := range in.Devices {
		out.Devices = append(out.Devices, oneview.PCIDevice{
			ID:                d.ID,
			Name:              d.Name,
			DeviceType:        d.DeviceType,
			Location:          d.Location,
			Manufacturer:      d.Manufacturer,
			PartNumber:        d.PartNumber,
			ProductPartNumber: d.ProductPartNumber,
			SerialNumber:      d.SerialNumber,
			FirmwareVersion:   d.FirmwareVersion,
			Health:            d.Health,
			State:             d.State,
		})
	}
	for _, f := range in.Firmware {
		out.Firmware = append(out.Firmware, oneview.FirmwareComponent{
			Name:     f.Name,
			Version:  f.Version,
			Location: f.Location,
			State:    f.State,
			Status:   f.Status,
		})
	}
	if in.Metadata != nil {
		out.APIVersion = int(in.Metadata.APIVersion)
		out.Product = in.Metadata.Product
		out.Endpoint = in.Metadata.Endpoint
		out.Warnings = append([]string(nil), in.Metadata.Warnings...)
		for _, src := range in.Metadata.Sources {
			out.Sources = append(out.Sources, oneview.ServerSource{
				Endpoint:   src.Endpoint,
				APIVersion: int(src.APIVersion),
				Product:    src.Product,
			})
		}
	}
}
