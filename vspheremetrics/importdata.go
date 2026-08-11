package vspheremetrics

import (
	"fmt"
	"strings"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func (vmp *VMPoint) ImportData(vm mo.VirtualMachine, infra *VMInfrastructure, vcenter, datacenter, cluster string) {
	if infra == nil {
		return
	}

	vmp.LockObject()
	defer vmp.UnLockObject()

	vmp.WorkStation = false

	osUp := strings.ToUpper(vm.Summary.Config.GuestFullName)
	if strings.Contains(osUp, "WINDOWS") {
		if !strings.Contains(osUp, "SERVER") {
			vmp.WorkStation = true // это рабочая станция windows
		}
	}

	vmp.Template = vm.Config.Template
	if strings.Contains(vm.Name, "vCLS.") || strings.Contains(vm.Name, "tmpl-") {
		vmp.Template = true
	}

	vmp.Name = vm.Name
	vmp.GuestFullName = vm.Summary.Config.GuestFullName
	vmp.PathName = vm.Summary.Config.VmPathName

	if vm.Summary.Runtime.PowerState == "poweredOff" {
		vmp.PowerState = false
	} else {
		vmp.PowerState = true
	}
	key := fmt.Sprintf("%s/%s/%s/%s", vcenter, datacenter, cluster, vm.Name)
	switch key {
	case "s-mf-vc-01.global.bcs/MF-DC-01/CL-MF-DITRB/s-mf-portal-ws-03":
		vmp.UUID = "420f697e-e446-e1ae-3f18-442e1403fb6f-s-mf-portal-ws-03"
	case "s-mf-vc-01.global.bcs/MF-DC-01/CL-MF-DITRB/s-mf-portal-ws-07":
		vmp.UUID = "420f697e-e446-e1ae-3f18-442e1403fb6f-s-mf-portal-ws-07"
	case "s-vc-02.global.bcs/BCS-Production/CL-PROD-CMN-03/s-rds-sh-45":
		vmp.UUID = "420f1855-aae8-5f6e-98e6-d9ea9adef0ca-s-rds-sh-45"
	case "s-vc-02.global.bcs/BCS-Production/CL-PROD-CMN-03/S-NSK01-SH00045":
		vmp.UUID = "420f1855-aae8-5f6e-98e6-d9ea9adef0ca-S-NSK01-SH00045"
	default:
		vmp.UUID = vm.Summary.Config.Uuid
	}
	//if

	vmp.GuestID = vm.Summary.Config.GuestId
	vmp.Annotation = vm.Summary.Config.Annotation

	if vm.Config != nil {

		vmp.NumCores = vm.Config.Hardware.NumCPU
		if vm.Config.Hardware.NumCoresPerSocket != nil && *vm.Config.Hardware.NumCoresPerSocket > 0 {
			vmp.NumCPU = vm.Config.Hardware.NumCPU / *vm.Config.Hardware.NumCoresPerSocket
		} else {
			vmp.NumCPU = vm.Config.Hardware.NumCPU
		}
		vmp.MemorySizeGB = float64(vm.Config.Hardware.MemoryMB) / 1024.0
	} else {
		vmp.NumCores = vm.Summary.Config.NumCpu
		vmp.NumCPU = vm.Summary.Config.NumCpu
		vmp.MemorySizeGB = float64(vm.Summary.Config.MemorySizeMB) / 1024.0
	}

	vmp.CapacityGb = 0.0
	vmp.FreeSpaceGb = 0.0

	vmp.UsageDatastore = make([]VMUsageOnDatastore, 0)
	stID := int64(0)
	if vm.Storage != nil {
		if len(vm.Storage.PerDatastoreUsage) > 0 {
			for _, st := range vm.Storage.PerDatastoreUsage {
				usageDS := VMUsageOnDatastore{
					Vcenter:                vcenter,
					Datacenter:             datacenter,
					Cluster:                cluster,
					Datastore:              vcenter + "/" + st.Datastore.Value,
					CommittedGb:            float64(st.Committed) / (1024.0 * 1024.0 * 1024.0),
					UncommittedGb:          float64(st.Uncommitted) / (1024.0 * 1024.0 * 1024.0),
					UnsharedGb:             float64(st.Unshared) / (1024.0 * 1024.0 * 1024.0),
					PercentAllocateStorage: 0.0}
				if !strings.Contains(st.Datastore.Value, "datastore") {
					fmt.Println("local storage", st.Datastore.Value)
				}
				if ds, ok := infra.DataStoreMapList.Get(vcenter + "/" + st.Datastore.Value); ds != nil && ok {
					usageDS.PercentAllocateStorage = 100.0 * usageDS.CommittedGb / ds.CapacityGb
					usageDS.StorageName = ds.StorageName
					usageDS.UID = ds.UID
				} else {
					//fmt.Println("Ошибка поиска хранилища", st.Datastore)
				}
				//проверить ошибка по памяти
				usageDS.ID = stID
				stID++
				vmp.UsageDatastore = append(vmp.UsageDatastore, usageDS)
			}
		}
	}

	vmp.Disk = nil
	if vm.Guest != nil {
		vmp.ToolsStatus = (string)(vm.Guest.ToolsStatus)
		vmp.GuestDetailedData = vm.Guest.GuestDetailedData
		vmp.HostName = vm.Guest.HostName
		vmp.NumVirtualDisks = int32(len(vm.Guest.Disk))
		vmp.IPAddress = ""
		vmp.MacAddress = ""
		vmp.DomainName = ""
		vmp.Network = ""
		for _, net := range vm.Guest.Net {
			if net.MacAddress != "" {
				vmp.MacAddress = vmp.MacAddress + net.MacAddress + "; "
			}

			for _, ip := range net.IpAddress {
				if ip != "" {
					vmp.IPAddress = vmp.IPAddress + ip + "; "
				}
			}
			if net.Network != "" {
				vmp.Network = vmp.Network + net.Network + "; "
			}

		}
		if vmp.IPAddress == "" {
			vmp.IPAddress = vm.Guest.IpAddress
		}

		//if len(vm.Guest.IpStack) > 1 {
		//	fmt.Println("len(IpStack) > 1")
		//}
		for _, ipSt := range vm.Guest.IpStack {
			if ipSt.DnsConfig != nil {
				if vm.Guest.HostName != ipSt.DnsConfig.HostName {
					if vmp.HostName == "" {
						vmp.HostName = ipSt.DnsConfig.HostName
					}
				}
				//адрес локальный и контроллеров домена
				//				for _, ip := range ipSt.DnsConfig.IpAddress {
				//					if ip != "" {
				//						vmp.IPAddress = vmp.IPAddress + ip + "; "
				//					}
				//				}
				if len(vm.Guest.IpStack) == 1 {
					vmp.DomainName = ipSt.DnsConfig.DomainName
				} else {
					if ipSt.DnsConfig.DomainName != "" {
						vmp.DomainName = vmp.DomainName + ipSt.DnsConfig.DomainName + "; " //имя домена global.bcs
					}
				}
				//for _, ipAddress := range ipSt.DnsConfig.IpAddress {
				//	_ = ipAddress //IP адреса
				//}
			}

		}

		for _, disk := range vm.Guest.Disk {
			d := GuestDiskInfo{DiskPath: disk.DiskPath, FileSystemType: disk.FilesystemType}
			d.CapacityGb = float64(disk.Capacity) / (1024.0 * 1024.0 * 1024)
			d.FreeSpaceGb = float64(disk.FreeSpace) / (1024.0 * 1024.0 * 1024)
			d.PercentUsed = 100.0 * (d.CapacityGb - d.FreeSpaceGb) / d.CapacityGb
			vmp.CapacityGb = vmp.CapacityGb + d.CapacityGb
			vmp.FreeSpaceGb = vmp.FreeSpaceGb + d.FreeSpaceGb
			vmp.Disk = append(vmp.Disk, d)
		}

		if vmp.CapacityGb != 0.0 {
			vmp.PercentUsed = 100.0 * (vmp.CapacityGb - vmp.FreeSpaceGb) / vmp.CapacityGb
		}

	} else {
		//fmt.Println(vmp.Name, "vm.Guest == nil")
	}

	vmp.NumEthernetCards = vm.Summary.Config.NumEthernetCards
	if vmp.NumVirtualDisks == 0 {
		vmp.NumVirtualDisks = vm.Summary.Config.NumVirtualDisks
	}
	if vm.Summary.Storage != nil {
		vmp.DiskCommittedGB = float64(vm.Summary.Storage.Committed) / (1024.0 * 1024.0 * 1024)
		vmp.DiskUncommittedGB = float64(vm.Summary.Storage.Uncommitted) / (1024.0 * 1024.0 * 1024.0)
	}
	if vm.Config == nil {
		return
	}
	for _, dev := range vm.Config.Hardware.Device {
		backing, ok := dev.GetVirtualDevice().Backing.(*types.VirtualDiskRawDiskMappingVer1BackingInfo) //types.VirtualDiskRawDiskMappingVer1BackingInfo
		if !ok || backing == nil {
			continue
		}
		label := dev.GetVirtualDevice().DeviceInfo.GetDescription().Label
		scsiLun, ok := infra.LUNMapListUUID.Get(vcenter + "/" + backing.LunUuid)
		if scsiLun != nil && ok {
			usageDS := VMUsageOnDatastore{
				Vcenter:                vcenter,
				Datacenter:             datacenter,
				Cluster:                cluster,
				UID:                    scsiLun.UID,
				StorageName:            vm.Name + "/" + label,
				Datastore:              vcenter + "/" + scsiLun.CanonicalName,
				CommittedGb:            float64(dev.(*types.VirtualDisk).CapacityInKB) / (1024.0 * 1024.0),
				UncommittedGb:          0.0,
				UnsharedGb:             0.0,
				PercentAllocateStorage: 0.0}
			//vmp.LockGetMetric()
			usageDS.ID = stID
			stID++
			vmp.UsageDatastore = append(vmp.UsageDatastore, usageDS)
			//vmp.UnLockGetMetric()
		}
	}
	if vm.LayoutEx != nil {
		vmp.Files = nil
		for _, fileRec := range vm.LayoutEx.File {
			if fileRec.Type != "snapshotData" {
				continue
			} //пока только снапшоты
			file := VMFile{}
			file.Name = fileRec.Name
			file.Type = fileRec.Type
			file.Size = float64(fileRec.Size) / 1024.0             //kb
			file.UniqueSize = float64(fileRec.UniqueSize) / 1024.0 //kb
			vmp.Files = append(vmp.Files, file)
		}
	}
}

// VMClearPerfomance сбрасывает агрегированные показатели производительности (min/max/average) виртуальной машины.
func (hs *HostInfo) ImportData(hi mo.HostSystem) {
	if hs == nil {
		return
	}
	hs.LockObject()
	defer hs.UnLockObject()

	s := hi.Summary
	h := s.Hardware
	r := hi.Runtime
	if h == nil {
		return
	}
	hs.MemorySizeGB = float64(h.MemorySize) / (1024.0 * 1024.0 * 1024.0)
	hs.HostMaxVirtualDiskCapacity = float64(r.HostMaxVirtualDiskCapacity) / (1024.0 * 1024.0)
	hs.Name = s.Config.Name

	hs.Vendor = h.Vendor
	hs.Model = h.Model
	hs.UUID = h.Uuid
	hs.CPUModel = h.CpuModel
	hs.CPUMhz = h.CpuMhz
	hs.NumCPUPkgs = h.NumCpuPkgs
	hs.NumCPUCores = h.NumCpuCores
	hs.NumCPUThreads = h.NumCpuThreads
	hs.NumNics = h.NumNics
	hs.NumHBAs = h.NumHBAs

	if r.PowerState == "poweredOn" {
		hs.PowerState = true
	} else {
		hs.PowerState = false
	}
	hs.InMaintenanceMode = r.InMaintenanceMode

	for _, info := range hi.Hardware.SystemInfo.OtherIdentifyingInfo {
		keyTag := info.IdentifierType.GetElementDescription().Key
		switch keyTag {
		case "SerialNumberTag":
			hs.SerialNumber = info.IdentifierValue
		case "EnclosureSerialNumberTag":
			hs.EnclosureSerialNumber = info.IdentifierValue
		}
	}

	if strings.TrimSpace(hs.SerialNumber) == strings.TrimSpace(hs.EnclosureSerialNumber) {
		hs.EnclosureSerialNumber = ""
	}

	if hi.Hardware.SystemInfo.SerialNumber != "" {
		hs.SerialNumber = hi.Hardware.SystemInfo.SerialNumber
	}
	if hi.Summary.Config.Product != nil {
		hs.ProductFullName = hi.Summary.Config.Product.FullName //версия esxi
		hs.Product = *hi.Summary.Config.Product
	}
	if hi.Config != nil {
		if hi.Config.Network != nil {
			hs.IPAddress = ""
			hs.MacAddress = ""
			for _, vnic := range hi.Config.Network.Vnic {
				if vnic.Spec.Ip.IpAddress != "" {
					hs.IPAddress = hs.IPAddress + vnic.Spec.Ip.IpAddress + "; "
					hs.MacAddress = hs.MacAddress + vnic.Spec.Mac + ";"
				}
			}
		}
	}
}

// AddHostPerfomance добавляет вклад одного хоста в суммарные ресурсы кластера (CPU, память, диски).
