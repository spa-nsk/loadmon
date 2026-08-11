package grpcclient

import (
	"fmt"
	"loadmon/oneview"
	pb "loadmon/protocol"
	"strconv"

	"github.com/HewlettPackard/oneview-golang/utils"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	vsn := in.Base.VirtualSerialNumber.String()

	if vsn == "null" {
		out.Base.VirtualSerialNumber = ""
	} else {
		out.Base.VirtualSerialNumber = vsn
	}

	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = in.Base.UUID.String()
	out.Base.FormFactor = in.Base.FormFactor
	out.Base.OperatingSystem = in.OperatingSystem

	out.Memory = &pb.ServerHardwareMemory{}

	out.Memory.Modified = timestamppb.New(in.Memory.Modified)
	out.Memory.Count = int32(in.Memory.Count)
	out.Memory.Name = in.Memory.Name

	for _, p := range in.Memory.Data {
		m := pb.MemoryModule{}
		m.BaseModuleType = p.BaseModuleType
		m.CapacityMiB = int32(p.CapacityMiB)
		m.DeviceLocator = p.DeviceLocator
		m.Manufacturer = p.Manufacturer
		m.MemoryDeviceType = p.MemoryDeviceType
		m.Name = p.Name
		m.OperatingSpeedMhz = int32(p.OperatingSpeedMhz)
		m.PartNumber = p.PartNumber
		m.RankCount = int32(p.RankCount)
		m.Status = &pb.MemoryStatus{Healt: p.Status.Health, State: p.Status.State}
		out.Memory.Data = append(out.Memory.Data, &m)
	}

	out.Storage = &pb.ServerHardwareLocalStorage{}

	out.Storage.Modified = timestamppb.New(in.Storage.Modified)
	out.Storage.Count = int32(in.Storage.Count)
	out.Storage.Name = in.Storage.Name
	for _, p := range in.Storage.Data {
		s := pb.LocalStorage{}

		s.AdapterType = p.AdapterType                                                                                                                        //тип адаптера "SmartArray"
		s.CacheMemorySizeMiB = int32(p.CacheMemorySizeMiB)                                                                                                   //размер кэша контроллера в мегабайтах
		s.CacheModuleSerialNumber = p.CacheModuleSerialNumber                                                                                                //серийный номер контроллера
		s.EncryptionCspTestPassed = p.EncryptionCspTestPassed                                                                                                //результат тестового прохода
		s.EncryptionEnabled = p.EncryptionEnabled                                                                                                            //шифрование включено?
		s.EncryptionSelfTestPassed = p.EncryptionSelfTestPassed                                                                                              //результата теста шифрования
		s.Location = p.Location                                                                                                                              //расположение "Slot 3"
		s.Model = p.Model                                                                                                                                    //модель "Smart array p840 Controller"
		s.Name = p.Name                                                                                                                                      //наименование "Controller in Slot 3"
		s.SerialNumber = p.SerialNumber                                                                                                                      //серийный номер
		s.Status = &pb.Status{Health: p.Status.Health, State: p.Status.State}                                                                                //состояние
		s.CacheModuleStatus = &pb.CacheModuleStatus{Health: p.CacheModuleStatus.Health}                                                                      //состояние кэша
		s.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: p.FirmwareVersion.Current.VersionString}}                                //прошивка
		s.ControllerBoard = &pb.ControllerBoard{Status: &pb.ControllerSatus{Health: p.ControllerBoard.Status.Health, State: p.ControllerBoard.Status.State}} // p.ControllerBoard      *ControllerBo
		for _, l := range p.LogicalDrives {                                                                                                                  //томы (логические диски)
			nl := pb.LocalLogicalDrive{}
			nl.CapacityMiB = l.CapacityMiB
			nl.LogicalDriveEncryption = l.LogicalDriveEncryption
			nl.LogicalDriveNumber = int32(l.LogicalDriveNumber)
			nl.LogicalDriveType = l.LogicalDriveType
			nl.Raid = l.Raid
			nl.Status = &pb.Status{Health: l.Status.Health, State: l.Status.State}
			for _, d := range l.DataDrives {
				ld := pb.LogicalDataDrive{}
				ld.CapacityMiB = d.CapacityMiB                //float64               //емкость диска в мегабайтах
				for _, ps := range d.DiskDriveStatusReasons { //[]string              //статусы "None"
					ld.DiskDriveStatusReasons = append(ld.DiskDriveStatusReasons, ps)
				}
				ld.DiskDriveUse = d.DiskDriveUse                                                                                       //string           //использование диска в томе "Data"
				ld.EncryptedDrive = d.EncryptedDrive                                                                                   //bool             //включено ли шифрование true, false
				ld.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: d.FirmwareVersion.Current.VersionString}} //*FirmwareVersion //текущая версия прошивки
				ld.Location = d.Location                                                                                               //string           //расположение "1I:1:7"
				ld.LocationFormat = d.LocationFormat                                                                                   //string           //формат строки расположения "Control
				ld.MediaType = d.MediaType                                                                                             //string           //тип диска "HDD", ...
				ld.Model = d.Model                                                                                                     //string           //модель "MB4000JVYZQ"
				ld.SerialNumber = d.SerialNumber                                                                                       //string           //серийный номер "ZC17LE2X"
				ld.Status = &pb.Status{}                                                                                               //d.Status         //*Status            //состояние

				nl.DataDrives = append(nl.DataDrives, &ld)
			}

			s.LogicalDrives = append(s.LogicalDrives, &nl)
		}
		for _, pd := range p.PhysicalDrives {
			d := pb.LocalPhysicalDrive{}
			d.CapacityMiB = pd.CapacityMiB
			for _, ds := range pd.DiskDriveStatusReasons {
				d.DiskDriveStatusReasons = append(d.DiskDriveStatusReasons, ds)
			}
			d.DiskDriveUse = pd.DiskDriveUse
			d.EncryptedDrive = pd.EncryptedDrive
			d.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
			d.Location = pd.Location
			d.LocationFormat = pd.LocationFormat
			d.MediaType = pd.MediaType
			d.Model = pd.Model
			d.SerialNumber = pd.SerialNumber
			d.Status = &pb.Status{Health: pd.Status.Health, State: pd.Status.State}
			s.PhysicalDrives = append(s.PhysicalDrives, &d)
		}
		for _, se := range p.StorageEnclosures {
			enc := pb.LocalStorageEnclosure{}
			enc.DriveBayCount = int32(se.DriveBayCount)
			enc.FirmwareVersion = &pb.FirmwareVersion{Current: &pb.Version{VersionString: se.FirmwareVersion.Current.VersionString}}
			enc.Location = se.Location
			enc.LocationFormat = se.LocationFormat
			enc.Status = &pb.Status{Health: se.Status.Health, State: se.Status.State}
			s.StorageEnclosures = append(s.StorageEnclosures, &enc)
		}
		out.Storage.Data = append(out.Storage.Data, &s)
	}
	out.Datacenter = &pb.DatacenterInfo{}
	out.Datacenter.Width = int32(in.Datacenter.Width)
	out.Datacenter.Depth = int32(in.Datacenter.Depth)
	out.Datacenter.Currency = in.Datacenter.Currency
	out.Datacenter.DeratingType = in.Datacenter.DeratingType
	out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
	out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
	for _, ct := range in.Datacenter.Contents {
		out.Datacenter.Contents = append(out.Datacenter.Contents, &pb.Contents{X: ct.X, Y: ct.Y, Rotation: ct.Rotation})
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
	out.EnclosureInfo.Type = in.EnclosureInfo.Type                                 //string
	out.EnclosureInfo.Category = in.EnclosureInfo.Category                         //string
	out.EnclosureInfo.Created = timestamppb.New(in.EnclosureInfo.Created)          //*timestamppb.Timestamp
	out.EnclosureInfo.Modified = timestamppb.New(in.EnclosureInfo.Modified)        //*timestamppb.Timestamp
	out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState                 //string
	out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason                   //string
	out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType               //string
	out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel             //string
	out.EnclosureInfo.UUID = in.EnclosureInfo.UUID                                 //string
	out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber                 //string
	out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber                     //string
	out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState //string
	out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent           //string
	out.EnclosureInfo.DeviceBayCount = int32(in.EnclosureInfo.DeviceBayCount)      //int32

	//out.EnclosureInfo.DeviceBays = in.EnclosureInfo.DeviceBays             //[]*DeviceBayInfo
	for _, dev := range in.EnclosureInfo.DeviceBays {
		nd := pb.DeviceBayInfo{}
		nd.Type = dev.Type                                                                       //string
		nd.BayNumber = int32(dev.BayNumber)                                                      //int32
		nd.DevicePresence = dev.DevicePresence                                                   //string
		nd.CoveredByDevice = dev.CoveredByDevice                                                 //string
		nd.URI = dev.URI                                                                         //string
		nd.Category = dev.Category                                                               //string
		nd.AvailableForHalfHeightProfile = dev.AvailableForHalfHeightProfile                     //bool
		nd.AvailableForFullHeightProfile = dev.AvailableForFullHeightProfile                     //bool
		nd.DeviceBayType = dev.DeviceBayType                                                     //string
		nd.DeviceFormFactor = dev.DeviceFormFactor                                               //string
		nd.BayPowerState = dev.BayPowerState                                                     //string
		nd.ChangeState = dev.ChangeState                                                         //string
		nd.AvailableForHalfHeightDoubleWideProfile = dev.AvailableForHalfHeightDoubleWideProfile //bool
		nd.AvailableForFullHeightDoubleWideProfile = dev.AvailableForFullHeightDoubleWideProfile //bool

		out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, &nd)
	}

	out.EnclosureInfo.InterconnectBayCount = int32(in.EnclosureInfo.InterconnectBayCount) //int32

	//out.EnclosureInfo.InterconnectBays = in.EnclosureInfo.InterconnectBays //[]*InterconnectBayInfo
	for _, con := range in.EnclosureInfo.InterconnectBays {
		ncon := pb.InterconnectBayInfo{}

		ncon.BayNumber = int32(con.BayNumber)              //int32
		ncon.InterconnectModel = con.InterconnectModel     ////string
		ncon.SerialNumber = con.SerialNumber               //string
		ncon.InterconnectBayType = con.InterconnectBayType //string
		ncon.ChangeState = con.ChangeState                 //string
		ncon.BayPowerState = con.BayPowerState             //string

		out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, &ncon)
	}

	out.EnclosureInfo.FanBayCount = int32(in.EnclosureInfo.FanBayCount) //int32

	//out.EnclosureInfo.FanBays = in.EnclosureInfo.FanBays //[]*FanBayInfo
	for _, fan := range in.EnclosureInfo.FanBays {
		nfan := pb.FanBayInfo{}
		nfan.BayNumber = int32(fan.BayNumber)      //int32
		nfan.DevicePresence = fan.DevicePresence   //string
		nfan.DeviceRequired = fan.DeviceRequired   //bool
		nfan.Status = fan.Status                   //string
		nfan.Model = fan.Model                     //string
		nfan.PartNumber = fan.PartNumber           //string
		nfan.SparePartNumber = fan.SparePartNumber //string
		nfan.FanBayType = fan.FanBayType           //string
		nfan.ChangeState = fan.ChangeState         //string
		nfan.State = fan.State                     //string
		out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, &nfan)
	}

	out.EnclosureInfo.PowerSupplyBayCount = int32(in.EnclosureInfo.PowerSupplyBayCount) //int32

	//out.EnclosureInfo.PowerSupplyBays = in.EnclosureInfo.PowerSupplyBays //[]*PowerSupplyBayInfo
	for _, pwr := range in.EnclosureInfo.PowerSupplyBays {
		npwr := pb.PowerSupplyBayInfo{}
		npwr.BayNumber = int32(pwr.BayNumber)            //int32
		npwr.DevicePresence = pwr.DevicePresence         //string
		npwr.Status = pwr.Status                         //string
		npwr.Model = pwr.Model                           //string
		npwr.SerialNumber = pwr.SerialNumber             //string
		npwr.PartNumber = pwr.PartNumber                 //string
		npwr.SparePartNumber = pwr.SparePartNumber       //string
		npwr.PowerSupplyBayType = pwr.PowerSupplyBayType //string
		npwr.ChangeState = pwr.ChangeState               //string
		out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, &npwr)
	}

	out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged                   //bool
	out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware //bool

	//out.EnclosureInfo.ManagerBays = in.EnclosureInfo.ManagerBays //[]*ManagerBayInfo
	for _, man := range in.EnclosureInfo.ManagerBays {
		nman := pb.ManagerBayInfo{}

		nman.BayNumber = int32(man.BayNumber)    //int32
		nman.ManagerType = man.ManagerType       //string
		nman.BayPowerState = man.BayPowerState   //string
		nman.FwVersion = man.FwVersion           //string
		nman.DevicePresence = man.DevicePresence //string
		nman.Role = man.Role                     //string
		nman.IPAddress = man.IPAddress           //string
		nman.ChangeState = man.ChangeState       //string
		nman.FwBuildDate = man.FwBuildDate       //string
		nman.DhcpIpv6Enable = man.DhcpIpv6Enable //bool
		nman.FqdnHostName = man.FqdnHostName     //string
		nman.DhcpEnable = man.DhcpEnable         //bool

		out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, &nman)
	}

	out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState                             //string
	out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState //string
	out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType   //string
	out.EnclosureInfo.Status = in.EnclosureInfo.Status                                         //string
	out.EnclosureInfo.Name = in.EnclosureInfo.Name                                             //string
	out.EnclosureInfo.State = in.EnclosureInfo.State                                           //string
	out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP             //string
	out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP               //string
	out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag                                     //string
	out.EnclosureInfo.RackName = in.EnclosureInfo.RackName                                     //string
	out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID                               //string
	out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName                           //string
	out.EnclosureInfo.VcmMode = fmt.Sprint(in.EnclosureInfo.VcmMode)                           //bool
	out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL                                         //string
	out.EnclosureInfo.OaBays = int32(in.EnclosureInfo.OaBays)                                  //int32
	out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState                         //string

	out.BayInfo = &pb.DeviceBayInfo{}

	out.BayInfo.Type = in.BayInfo.Type                                                                       //string
	out.BayInfo.BayNumber = int32(in.BayInfo.BayNumber)                                                      //int32
	out.BayInfo.DevicePresence = in.BayInfo.DevicePresence                                                   //string
	out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice                                                 //string
	out.BayInfo.URI = in.BayInfo.URI                                                                         //string
	out.BayInfo.Category = in.BayInfo.Category                                                               //string
	out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile                     //bool
	out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile                     //bool
	out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType                                                     //string
	out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor                                               //string
	out.BayInfo.BayPowerState = in.BayInfo.BayPowerState                                                     //string
	out.BayInfo.ChangeState = in.BayInfo.ChangeState                                                         //string
	out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile //bool
	out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile //bool
}

// MessageToServerHardware конвертирует gRPC-сообщение pb.ServerHardware обратно в доменную модель oneview.ServerHardware. При in == nil или out == nil ничего не делает.
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

	vsn := in.Base.VirtualSerialNumber

	if vsn == "null" {
		out.Base.VirtualSerialNumber = utils.Nstring("")
	} else {
		out.Base.VirtualSerialNumber = utils.Nstring(vsn)
	}

	out.Base.FormFactor = in.Base.FormFactor

	out.Base.ServerName = in.Base.ServerName
	out.Base.ShortModel = in.Base.ShortModel
	out.Base.Status = in.Base.Status
	out.Base.UUID = utils.Nstring(in.Base.UUID)

	out.OperatingSystem = in.Base.OperatingSystem

	out.Memory.Modified = in.Memory.Modified.AsTime()
	out.Memory.Count = int(in.Memory.Count)
	out.Memory.Name = in.Memory.Name

	for _, p := range in.Memory.Data {
		m := oneview.MemoryModule{}
		m.BaseModuleType = p.BaseModuleType
		m.CapacityMiB = int(p.CapacityMiB)
		m.DeviceLocator = p.DeviceLocator
		m.Manufacturer = p.Manufacturer
		m.MemoryDeviceType = p.MemoryDeviceType
		m.Name = p.Name
		m.OperatingSpeedMhz = int(p.OperatingSpeedMhz)
		m.PartNumber = p.PartNumber
		m.RankCount = int(p.RankCount)
		m.Status = oneview.MemoryStatus{Health: p.Status.Healt, State: p.Status.State}
		out.Memory.Data = append(out.Memory.Data, m)
	}

	out.Storage.Modified = in.Storage.Modified.AsTime()
	out.Storage.Count = int(in.Storage.Count)
	out.Storage.Name = in.Storage.Name
	for _, p := range in.Storage.Data {
		s := oneview.LocalStorage{}

		s.AdapterType = p.AdapterType                                                                                                                                //тип адаптера "SmartArray"
		s.CacheMemorySizeMiB = int(p.CacheMemorySizeMiB)                                                                                                             //размер кэша контроллера в мегабайтах
		s.CacheModuleSerialNumber = p.CacheModuleSerialNumber                                                                                                        //серийный номер контроллера
		s.EncryptionCspTestPassed = p.EncryptionCspTestPassed                                                                                                        //тезультат тестового прохода
		s.EncryptionEnabled = p.EncryptionEnabled                                                                                                                    //шифрование включено?
		s.EncryptionSelfTestPassed = p.EncryptionSelfTestPassed                                                                                                      //результата теста шифрования
		s.Location = p.Location                                                                                                                                      //расположение "Slot 3"
		s.Model = p.Model                                                                                                                                            //модель "Smart array p840 Controlle
		s.Name = p.Name                                                                                                                                              //наименование "Controller in Slot 3
		s.SerialNumber = p.SerialNumber                                                                                                                              //серийный номер
		s.Status = oneview.Status{Health: p.Status.Health, State: p.Status.State}                                                                                    //состояние
		s.CacheModuleStatus = oneview.CacheModuleStatus{Health: p.CacheModuleStatus.Health}                                                                          //состояние кэша
		s.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: p.FirmwareVersion.Current.VersionString}}                                //прошивка
		s.ControllerBoard = oneview.ControllerBoard{Status: oneview.ControllerSatus{Health: p.ControllerBoard.Status.Health, State: p.ControllerBoard.Status.State}} //  p.ControllerBoard           *Cont
		for _, l := range p.LogicalDrives {                                                                                                                          //томы (логические диски)
			nl := oneview.LocalLogicalDrive{}
			nl.CapacityMiB = l.CapacityMiB
			nl.LogicalDriveEncryption = l.LogicalDriveEncryption
			nl.LogicalDriveNumber = int(l.LogicalDriveNumber)
			nl.LogicalDriveType = l.LogicalDriveType
			nl.Raid = l.Raid
			nl.Status = oneview.Status{Health: l.Status.Health, State: l.Status.State} //
			for _, d := range l.DataDrives {
				ld := oneview.LogicalDataDrive{}
				ld.CapacityMiB = d.CapacityMiB                //float64		//емкость диска в мегабайтах
				for _, ps := range d.DiskDriveStatusReasons { //[]string		//стаатусы "None"
					ld.DiskDriveStatusReasons = append(ld.DiskDriveStatusReasons, ps)
				}
				ld.DiskDriveUse = d.DiskDriveUse                                                                                               //string		//использование диска в томе "
				ld.EncryptedDrive = d.EncryptedDrive                                                                                           //bool		//включено ли шифрование true,
				ld.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: d.FirmwareVersion.Current.VersionString}} //*FirmwareVersion //текущая версия прошивки
				ld.Location = d.Location                                                                                                       //string		//расположение "1I:1:7"
				ld.LocationFormat = d.LocationFormat                                                                                           //string		//формат строки расположения
				ld.MediaType = d.MediaType                                                                                                     //string		//тип диска "HDD", ...
				ld.Model = d.Model                                                                                                             //string		//модель "MB4000JVYZQ"
				ld.SerialNumber = d.SerialNumber                                                                                               //string		//серийный номер "ZC17LE2X"
				ld.Status = oneview.Status{}                                                                                                   //d.Status		//*Status		//состояние

				nl.DataDrives = append(nl.DataDrives, ld)
			}

			s.LogicalDrives = append(s.LogicalDrives, nl)
		}
		for _, pd := range p.PhysicalDrives {
			d := oneview.LocalPhysicalDrive{}
			d.CapacityMiB = pd.CapacityMiB
			for _, ds := range pd.DiskDriveStatusReasons {
				d.DiskDriveStatusReasons = append(d.DiskDriveStatusReasons, ds)
			}
			d.DiskDriveUse = pd.DiskDriveUse
			d.EncryptedDrive = pd.EncryptedDrive
			d.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: pd.FirmwareVersion.Current.VersionString}}
			d.Location = pd.Location
			d.LocationFormat = pd.LocationFormat
			d.MediaType = pd.MediaType
			d.Model = pd.Model
			d.SerialNumber = pd.SerialNumber
			d.Status = oneview.Status{Health: pd.Status.Health, State: pd.Status.State}
			s.PhysicalDrives = append(s.PhysicalDrives, d)
		}
		for _, se := range p.StorageEnclosures {
			enc := oneview.LocalStorageEnclosure{}
			enc.DriveBayCount = int(se.DriveBayCount)
			enc.FirmwareVersion = oneview.FirmwareVersion{Current: oneview.Version{VersionString: se.FirmwareVersion.Current.VersionString}}
			enc.Location = se.Location
			enc.LocationFormat = se.LocationFormat
			enc.Status = oneview.Status{Health: se.Status.Health, State: se.Status.State}
			s.StorageEnclosures = append(s.StorageEnclosures, enc)
		}
		out.Storage.Data = append(out.Storage.Data, s)
	}
	out.Datacenter.Width = int(in.Datacenter.Width)
	out.Datacenter.Depth = int(in.Datacenter.Depth)
	out.Datacenter.Currency = in.Datacenter.Currency
	out.Datacenter.DeratingType = in.Datacenter.DeratingType
	out.Datacenter.DeratingPercentage = in.Datacenter.DeratingPercentage
	out.Datacenter.CoolingMultiplier = in.Datacenter.CoolingMultiplier
	for _, ct := range in.Datacenter.Contents {
		out.Datacenter.Contents = append(out.Datacenter.Contents, oneview.Content{X: ct.X, Y: ct.Y, Rotation: ct.Rotation})
	}

	out.Datacenter.Category = in.Datacenter.Category
	out.Datacenter.Created = in.Datacenter.Created.AsTime()
	out.Datacenter.Modified = in.Datacenter.Modified.AsTime()
	out.Datacenter.ID = in.Datacenter.ID
	out.Datacenter.UUID = in.Datacenter.UUID
	out.Datacenter.DefaultPowerLineVoltage = int(in.Datacenter.DefaultPowerLineVoltage)
	out.Datacenter.Name = in.Datacenter.Name
	out.Datacenter.State = in.Datacenter.State
	out.Datacenter.Status = in.Datacenter.Status

	out.Rack.UHeight = int(in.Rack.UHeight)
	out.Rack.Width = int(in.Rack.Width)
	out.Rack.Height = int(in.Rack.Height)
	out.Rack.Depth = int(in.Rack.Depth)
	out.Rack.SerialNumber = in.Rack.SerialNumber
	out.Rack.Created = in.Rack.Created.AsTime()
	out.Rack.Modified = in.Rack.Modified.AsTime()
	out.Rack.ID = in.Rack.ID
	out.Rack.UUID = in.Rack.UUID
	out.Rack.Name = in.Rack.Name
	out.Rack.State = in.Rack.State
	out.Rack.Status = in.Rack.Status

	out.EnclosureInfo.Type = in.EnclosureInfo.Type                                 //string
	out.EnclosureInfo.Category = in.EnclosureInfo.Category                         //string
	out.EnclosureInfo.Created = in.EnclosureInfo.Created.AsTime()                  //*timestamppb.Timestamp
	out.EnclosureInfo.Modified = in.EnclosureInfo.Modified.AsTime()                //*timestamppb.Timestamp
	out.EnclosureInfo.RefreshState = in.EnclosureInfo.RefreshState                 //string
	out.EnclosureInfo.StateReason = in.EnclosureInfo.StateReason                   //string
	out.EnclosureInfo.EnclosureType = in.EnclosureInfo.EnclosureType               //string
	out.EnclosureInfo.EnclosureModel = in.EnclosureInfo.EnclosureModel             //string
	out.EnclosureInfo.UUID = in.EnclosureInfo.UUID                                 //string
	out.EnclosureInfo.SerialNumber = in.EnclosureInfo.SerialNumber                 //string
	out.EnclosureInfo.PartNumber = in.EnclosureInfo.PartNumber                     //string
	out.EnclosureInfo.ReconfigurationState = in.EnclosureInfo.ReconfigurationState //string
	out.EnclosureInfo.LicensingIntent = in.EnclosureInfo.LicensingIntent           //string
	out.EnclosureInfo.DeviceBayCount = int(in.EnclosureInfo.DeviceBayCount)        //int32

	//out.EnclosureInfo.DeviceBays = in.EnclosureInfo.DeviceBays //[]*DeviceBayInfo
	for _, dev := range in.EnclosureInfo.DeviceBays {
		nd := oneview.DeviceBayInfo{}
		nd.Type = dev.Type                                                                       //string
		nd.BayNumber = int(dev.BayNumber)                                                        //int32
		nd.DevicePresence = dev.DevicePresence                                                   //string
		nd.CoveredByDevice = dev.CoveredByDevice                                                 //string
		nd.URI = dev.URI                                                                         //string
		nd.Category = dev.Category                                                               //string
		nd.AvailableForHalfHeightProfile = dev.AvailableForHalfHeightProfile                     //bool
		nd.AvailableForFullHeightProfile = dev.AvailableForFullHeightProfile                     //bool
		nd.DeviceBayType = dev.DeviceBayType                                                     //string
		nd.DeviceFormFactor = dev.DeviceFormFactor                                               //string
		nd.BayPowerState = dev.BayPowerState                                                     //string
		nd.ChangeState = dev.ChangeState                                                         //string
		nd.AvailableForHalfHeightDoubleWideProfile = dev.AvailableForHalfHeightDoubleWideProfile //bool
		nd.AvailableForFullHeightDoubleWideProfile = dev.AvailableForFullHeightDoubleWideProfile //bool

		out.EnclosureInfo.DeviceBays = append(out.EnclosureInfo.DeviceBays, nd)
	}

	out.EnclosureInfo.InterconnectBayCount = int(in.EnclosureInfo.InterconnectBayCount) //int32

	//out.EnclosureInfo.InterconnectBays = in.EnclosureInfo.InterconnectBays //[]*InterconnectBayInfo
	for _, con := range in.EnclosureInfo.InterconnectBays {
		ncon := oneview.InterconnectBayInfo{}

		ncon.BayNumber = int(con.BayNumber)                //int32
		ncon.InterconnectModel = con.InterconnectModel     //string
		ncon.SerialNumber = con.SerialNumber               //string
		ncon.InterconnectBayType = con.InterconnectBayType //string
		ncon.ChangeState = con.ChangeState                 //string
		ncon.BayPowerState = con.BayPowerState             //string

		out.EnclosureInfo.InterconnectBays = append(out.EnclosureInfo.InterconnectBays, ncon)
	}

	out.EnclosureInfo.FanBayCount = int(in.EnclosureInfo.FanBayCount) //int32

	//out.EnclosureInfo.FanBays = in.EnclosureInfo.FanBays //[]*FanBayInfo
	for _, fan := range in.EnclosureInfo.FanBays {
		nfan := oneview.FanBayInfo{}
		nfan.BayNumber = int(fan.BayNumber)        //int32
		nfan.DevicePresence = fan.DevicePresence   //string
		nfan.DeviceRequired = fan.DeviceRequired   //bool
		nfan.Status = fan.Status                   //string
		nfan.Model = fan.Model                     //string
		nfan.PartNumber = fan.PartNumber           //string
		nfan.SparePartNumber = fan.SparePartNumber //string
		nfan.FanBayType = fan.FanBayType           //string
		nfan.ChangeState = fan.ChangeState         //string
		nfan.State = fan.State                     //string
		out.EnclosureInfo.FanBays = append(out.EnclosureInfo.FanBays, nfan)
	}

	out.EnclosureInfo.PowerSupplyBayCount = int(in.EnclosureInfo.PowerSupplyBayCount) //int32

	//out.EnclosureInfo.PowerSupplyBays = in.EnclosureInfo.PowerSupplyBays //[]*PowerSupplyBayInfo
	for _, pwr := range in.EnclosureInfo.PowerSupplyBays {
		npwr := oneview.PowerSupplyBayInfo{}
		npwr.BayNumber = int(pwr.BayNumber)              //int32
		npwr.DevicePresence = pwr.DevicePresence         //string
		npwr.Status = pwr.Status                         //string
		npwr.Model = pwr.Model                           //string
		npwr.SerialNumber = pwr.SerialNumber             //string
		npwr.PartNumber = pwr.PartNumber                 //string
		npwr.SparePartNumber = pwr.SparePartNumber       //string
		npwr.PowerSupplyBayType = pwr.PowerSupplyBayType //string
		npwr.ChangeState = pwr.ChangeState               //string
		out.EnclosureInfo.PowerSupplyBays = append(out.EnclosureInfo.PowerSupplyBays, npwr)
	}

	out.EnclosureInfo.IsFwManaged = in.EnclosureInfo.IsFwManaged                   //bool
	out.EnclosureInfo.ForceInstallFirmware = in.EnclosureInfo.ForceInstallFirmware //bool

	//out.EnclosureInfo.ManagerBays = in.EnclosureInfo.ManagerBays //[]*ManagerBayInfo
	for _, man := range in.EnclosureInfo.ManagerBays {
		nman := oneview.ManagerBayInfo{}

		nman.BayNumber = int(man.BayNumber)      //int32
		nman.ManagerType = man.ManagerType       //string
		nman.BayPowerState = man.BayPowerState   //string
		nman.FwVersion = man.FwVersion           //string
		nman.DevicePresence = man.DevicePresence //string
		nman.Role = man.Role                     //string
		nman.IPAddress = man.IPAddress           //string
		nman.ChangeState = man.ChangeState       //string
		nman.FwBuildDate = man.FwBuildDate       //string
		nman.DhcpIpv6Enable = man.DhcpIpv6Enable //bool
		nman.FqdnHostName = man.FqdnHostName     //string
		nman.DhcpEnable = man.DhcpEnable         //bool

		out.EnclosureInfo.ManagerBays = append(out.EnclosureInfo.ManagerBays, nman)
	}

	out.EnclosureInfo.SupportState = in.EnclosureInfo.SupportState                             //string
	out.EnclosureInfo.SupportDataCollectionState = in.EnclosureInfo.SupportDataCollectionState //string
	out.EnclosureInfo.SupportDataCollectionType = in.EnclosureInfo.SupportDataCollectionType   //string
	out.EnclosureInfo.Status = in.EnclosureInfo.Status                                         //string
	out.EnclosureInfo.Name = in.EnclosureInfo.Name                                             //string
	out.EnclosureInfo.State = in.EnclosureInfo.State                                           //string
	out.EnclosureInfo.StandbyOaPreferredIP = in.EnclosureInfo.StandbyOaPreferredIP             //string
	out.EnclosureInfo.ActiveOaPreferredIP = in.EnclosureInfo.ActiveOaPreferredIP               //string
	out.EnclosureInfo.AssetTag = in.EnclosureInfo.AssetTag                                     //string
	out.EnclosureInfo.RackName = in.EnclosureInfo.RackName                                     //string
	out.EnclosureInfo.VcmDomainID = in.EnclosureInfo.VcmDomainID                               //string
	out.EnclosureInfo.VcmDomainName = in.EnclosureInfo.VcmDomainName                           //string
	if b, err := strconv.ParseBool(in.EnclosureInfo.VcmMode); err == nil {
		out.EnclosureInfo.VcmMode = b //bool
	}
	out.EnclosureInfo.VcmURL = in.EnclosureInfo.VcmURL                 //string
	out.EnclosureInfo.OaBays = int(in.EnclosureInfo.OaBays)            //int32
	out.EnclosureInfo.MigrationState = in.EnclosureInfo.MigrationState //string

	out.BayInfo.Type = in.BayInfo.Type                                                                       //string
	out.BayInfo.BayNumber = int(in.BayInfo.BayNumber)                                                        //int32
	out.BayInfo.DevicePresence = in.BayInfo.DevicePresence                                                   //string
	out.BayInfo.CoveredByDevice = in.BayInfo.CoveredByDevice                                                 //string
	out.BayInfo.URI = in.BayInfo.URI                                                                         //string
	out.BayInfo.Category = in.BayInfo.Category                                                               //string
	out.BayInfo.AvailableForHalfHeightProfile = in.BayInfo.AvailableForHalfHeightProfile                     //bool
	out.BayInfo.AvailableForFullHeightProfile = in.BayInfo.AvailableForFullHeightProfile                     //bool
	out.BayInfo.DeviceBayType = in.BayInfo.DeviceBayType                                                     //string
	out.BayInfo.DeviceFormFactor = in.BayInfo.DeviceFormFactor                                               //string
	out.BayInfo.BayPowerState = in.BayInfo.BayPowerState                                                     //string
	out.BayInfo.ChangeState = in.BayInfo.ChangeState                                                         //string
	out.BayInfo.AvailableForHalfHeightDoubleWideProfile = in.BayInfo.AvailableForHalfHeightDoubleWideProfile //bool
	out.BayInfo.AvailableForFullHeightDoubleWideProfile = in.BayInfo.AvailableForFullHeightDoubleWideProfile //bool
}

// MessageToClusterSprRecords конвертирует gRPC-сообщение pb.ClusterSprRecord (запись справочника кластеров) в доменную модель models.ClusterSprRecords. При in == nil или out == nil ничего не делает.
