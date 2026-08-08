package vspheremetrics

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func ReCalcVMPerfomance(vmP *VMPoint, perf VMPerf) {
	if vmP == nil {
		return
	}

	vmP.CPUBilling = vmP.CPUBilling + float64(perf.CPUUsage)*float64(vmP.CPUMhz)*float64(vmP.NumCores)/180.0 //утилизация процессора частота в мегагерцах умноженная процент утилизации и промежуток времени
	vmP.MemBilling = vmP.MemBilling + float64(perf.MemUsage)*vmP.MemorySizeGB*1024.0/180.0                   // утилизация памяти мегабайт часах

	if vmP.CPUUsageMaximum < perf.CPUUsage {
		vmP.CPUUsageMaximum = perf.CPUUsage
		vmP.CPUUsageMaximumTime = perf.Start
	}
	if vmP.CPUUsageMinimum > perf.CPUUsage {
		vmP.CPUUsageMinimum = perf.CPUUsage
		vmP.CPUUsageMinimumTime = perf.Start
	}
	vmP.CPUUsageAverage = vmP.CPUUsageAverage + perf.CPUUsage

	if vmP.MemUsageMaximum < perf.MemUsage {
		vmP.MemUsageMaximum = perf.MemUsage
		vmP.MemUsageMaximumTime = perf.Start
	}
	if vmP.MemUsageMinimum > perf.MemUsage {
		vmP.MemUsageMinimum = perf.MemUsage
		vmP.MemUsageMinimumTime = perf.Start
	}
	vmP.MemUsageAverage = vmP.MemUsageAverage + perf.MemUsage

	if vmP.NetUsageMaximum < perf.NetUsage {
		vmP.NetUsageMaximum = perf.NetUsage
		vmP.NetUsageMaximumTime = perf.Start
	}
	if vmP.NetUsageMinimum > perf.NetUsage {
		vmP.NetUsageMinimum = perf.NetUsage
		vmP.NetUsageMinimumTime = perf.Start
	}
	vmP.NetUsageAverage = vmP.NetUsageAverage + perf.NetUsage

	if vmP.DiskUsageMaximum < perf.DiskUsage {
		vmP.DiskUsageMaximum = perf.DiskUsage
		vmP.DiskUsageMaximumTime = perf.Start
	}
	if vmP.DiskUsageMinimum > perf.DiskUsage {
		vmP.DiskUsageMinimum = perf.DiskUsage
		vmP.DiskUsageMinimumTime = perf.Start
	}
	vmP.DiskUsageAverage = vmP.DiskUsageAverage + perf.DiskUsage

	if vmP.DiskReadMaximum < perf.DiskRead {
		vmP.DiskReadMaximum = perf.DiskRead
		vmP.DiskReadMaximumTime = perf.Start
	}
	if vmP.DiskReadMinimum > perf.DiskRead {
		vmP.DiskReadMinimum = perf.DiskRead
		vmP.DiskReadMinimumTime = perf.Start
	}
	vmP.DiskReadAverage = vmP.DiskReadAverage + perf.DiskRead

	if vmP.DiskWriteMaximum < perf.DiskWrite {
		vmP.DiskWriteMaximum = perf.DiskWrite
		vmP.DiskWriteMaximumTime = perf.Start
	}
	if vmP.DiskWriteMinimum > perf.DiskWrite {
		vmP.DiskWriteMinimum = perf.DiskWrite
		vmP.DiskWriteMinimumTime = perf.Start
	}
	vmP.DiskWriteAverage = vmP.DiskWriteAverage + perf.DiskWrite

	if vmP.DiskTotalReadLatencyMaximum < perf.DiskTotalReadLatency {
		vmP.DiskTotalReadLatencyMaximum = perf.DiskTotalReadLatency
		vmP.DiskTotalReadLatencyMaximumTime = perf.Start
	}
	if vmP.DiskTotalReadLatencyMinimum > perf.DiskTotalReadLatency {
		vmP.DiskTotalReadLatencyMinimum = perf.DiskTotalReadLatency
		vmP.DiskTotalReadLatencyMinimumTime = perf.Start
	}
	vmP.DiskTotalReadLatencyAverage = vmP.DiskTotalReadLatencyAverage + perf.DiskTotalReadLatency

	if vmP.DiskTotalWriteLatencyMaximum < perf.DiskTotalWriteLatency {
		vmP.DiskTotalWriteLatencyMaximum = perf.DiskTotalWriteLatency
		vmP.DiskTotalWriteLatencyMaximumTime = perf.Start
	}
	if vmP.DiskTotalWriteLatencyMinimum > perf.DiskTotalWriteLatency {
		vmP.DiskTotalWriteLatencyMinimum = perf.DiskTotalWriteLatency
		vmP.DiskTotalWriteLatencyMinimumTime = perf.Start
	}
	vmP.DiskTotalWriteLatencyAverage = vmP.DiskTotalWriteLatencyAverage + perf.DiskTotalWriteLatency
}

func CalcVMAveragePerfomance(vmP *VMPoint) {
	if vmP == nil {
		return
	}

	count := len(vmP.Perf)
	if count == 0 {
		return
	}

	vmP.CPUUsageAverage = vmP.CPUUsageAverage / float64(count)

	vmP.MemUsageAverage = vmP.MemUsageAverage / float64(count)

	vmP.NetUsageAverage = vmP.NetUsageAverage / float64(count)

	vmP.DiskUsageAverage = vmP.DiskUsageAverage / float64(count)

	vmP.DiskReadAverage = vmP.DiskReadAverage / float64(count)

	vmP.DiskWriteAverage = vmP.DiskWriteAverage / float64(count)

	vmP.DiskTotalReadLatencyAverage = vmP.DiskTotalReadLatencyAverage / float64(count)

	vmP.DiskTotalWriteLatencyAverage = vmP.DiskTotalWriteLatencyAverage / float64(count)
}

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

func (vm *VMPoint) VMClearPerfomance() {
	if vm == nil {
		return
	}

	vm.CPUUsageMaximum = 0.0
	vm.CPUUsageMaximumTime = time.Time{}
	vm.CPUUsageAverage = 0.0
	vm.CPUUsageMinimum = math.MaxFloat64
	vm.CPUUsageMinimumTime = time.Time{}
	vm.MemUsageMaximum = 0.0
	vm.MemUsageMaximumTime = time.Time{}
	vm.MemUsageAverage = 0.0
	vm.MemUsageMinimum = math.MaxFloat64
	vm.MemUsageMinimumTime = time.Time{}
	vm.DiskUsageMaximum = 0.0
	vm.DiskUsageMaximumTime = time.Time{}
	vm.DiskUsageAverage = 0.0
	vm.DiskUsageMinimum = math.MaxFloat64
	vm.DiskUsageMinimumTime = time.Time{}
	vm.DiskTotalReadLatencyMaximum = 0.0
	vm.DiskTotalReadLatencyMaximumTime = time.Time{}
	vm.DiskTotalReadLatencyAverage = 0.0
	vm.DiskTotalReadLatencyMinimum = math.MaxFloat64
	vm.DiskTotalReadLatencyMinimumTime = time.Time{}
	vm.DiskTotalWriteLatencyMaximum = 0.0
	vm.DiskTotalWriteLatencyMaximumTime = time.Time{}
	vm.DiskTotalWriteLatencyAverage = 0.0
	vm.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	vm.DiskTotalWriteLatencyMinimumTime = time.Time{}
	vm.DiskWriteMaximum = 0.0
	vm.DiskWriteMaximumTime = time.Time{}
	vm.DiskWriteAverage = 0.0
	vm.DiskWriteMinimum = math.MaxFloat64
	vm.DiskWriteMinimumTime = time.Time{}
	vm.DiskReadMaximum = 0.0
	vm.DiskReadMaximumTime = time.Time{}
	vm.DiskReadAverage = 0.0
	vm.DiskReadMinimum = math.MaxFloat64
	vm.DiskReadMinimumTime = time.Time{}
	vm.NetUsageMaximum = 0.0
	vm.NetUsageMaximumTime = time.Time{}
	vm.NetUsageAverage = 0.0
	vm.NetUsageMinimum = math.MaxFloat64
	vm.NetUsageMinimumTime = time.Time{}
	vm.PerfCount = 0
	vm.CPUBilling = 0
	vm.MemBilling = 0
	vm.DiskBilling = 0
	vm.NetBilling = 0
}

func (cl *Cluster) CLClearPerfomance() {
	if cl == nil {
		return
	}

	cl.CPUMhzAvg = 0
	cl.AllNumCPUPkgs = 0
	cl.AllNumCPUCores = 0
	cl.AllNumCPUThreads = 0
	cl.AllMemorySizeGB = 0.0
	cl.CPUUsageMaximum = 0.0
	cl.CPUUsageMaximumTime = time.Time{}
	cl.CPUUsageAverage = 0.0
	cl.CPUUsageMinimum = math.MaxFloat64
	cl.CPUUsageMinimumTime = time.Time{}
	cl.MemUsageMaximum = 0.0
	cl.MemUsageMaximumTime = time.Time{}
	cl.MemUsageAverage = 0.0
	cl.MemUsageMinimum = math.MaxFloat64
	cl.MemUsageMinimumTime = time.Time{}
	cl.DiskUsageMaximum = 0.0
	cl.DiskUsageMaximumTime = time.Time{}
	cl.DiskUsageAverage = 0.0
	cl.DiskUsageMinimum = math.MaxFloat64
	cl.DiskUsageMinimumTime = time.Time{}
	cl.DiskTotalReadLatencyMaximum = 0.0
	cl.DiskTotalReadLatencyMaximumTime = time.Time{}
	cl.DiskTotalReadLatencyAverage = 0.0
	cl.DiskTotalReadLatencyMinimum = math.MaxFloat64
	cl.DiskTotalReadLatencyMinimumTime = time.Time{}
	cl.DiskTotalWriteLatencyMaximum = 0.0
	cl.DiskTotalWriteLatencyMaximumTime = time.Time{}
	cl.DiskTotalWriteLatencyAverage = 0.0
	cl.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	cl.DiskTotalWriteLatencyMinimumTime = time.Time{}
	cl.DiskWriteMaximum = 0.0
	cl.DiskWriteMaximumTime = time.Time{}
	cl.DiskWriteAverage = 0.0
	cl.DiskWriteMinimum = math.MaxFloat64
	cl.DiskWriteMinimumTime = time.Time{}
	cl.DiskReadMaximum = 0.0
	cl.DiskReadMaximumTime = time.Time{}
	cl.DiskReadAverage = 0.0
	cl.DiskReadMinimum = math.MaxFloat64
	cl.DiskReadMinimumTime = time.Time{}
	cl.NetUsageMaximum = 0.0
	cl.NetUsageMaximumTime = time.Time{}
	cl.NetUsageAverage = 0.0
	cl.NetUsageMinimum = math.MaxFloat64
	cl.NetUsageMinimumTime = time.Time{}
	cl.CPUBilling = 0.0
	cl.DiskBilling = 0.0
	cl.MemBilling = 0.0
	cl.NetBilling = 0.0
}

func (hs *HostInfo) HSClearPerfomance() {
	if hs == nil {
		return
	}

	hs.CPUUsageMaximum = 0.0
	hs.CPUUsageMaximumTime = time.Time{}
	hs.CPUUsageAverage = 0.0
	hs.CPUUsageMinimum = math.MaxFloat64
	hs.CPUUsageMinimumTime = time.Time{}
	hs.MemUsageMaximum = 0.0
	hs.MemUsageMaximumTime = time.Time{}
	hs.MemUsageAverage = 0.0
	hs.MemUsageMinimum = math.MaxFloat64
	hs.MemUsageMinimumTime = time.Time{}
	hs.DiskUsageMaximum = 0.0
	hs.DiskUsageMaximumTime = time.Time{}
	hs.DiskUsageAverage = 0.0
	hs.DiskUsageMinimum = math.MaxFloat64
	hs.DiskUsageMinimumTime = time.Time{}
	hs.DiskTotalReadLatencyMaximum = 0.0
	hs.DiskTotalReadLatencyMaximumTime = time.Time{}
	hs.DiskTotalReadLatencyAverage = 0.0
	hs.DiskTotalReadLatencyMinimum = math.MaxFloat64
	hs.DiskTotalReadLatencyMinimumTime = time.Time{}
	hs.DiskTotalWriteLatencyMaximum = 0.0
	hs.DiskTotalWriteLatencyMaximumTime = time.Time{}
	hs.DiskTotalWriteLatencyAverage = 0.0
	hs.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	hs.DiskTotalWriteLatencyMinimumTime = time.Time{}
	hs.DiskWriteMaximum = 0.0
	hs.DiskWriteMaximumTime = time.Time{}
	hs.DiskWriteAverage = 0.0
	hs.DiskWriteMinimum = math.MaxFloat64
	hs.DiskWriteMinimumTime = time.Time{}
	hs.DiskReadMaximum = 0.0
	hs.DiskReadMaximumTime = time.Time{}
	hs.DiskReadAverage = 0.0
	hs.DiskReadMinimum = math.MaxFloat64
	hs.DiskReadMinimumTime = time.Time{}
	hs.NetUsageMaximum = 0.0
	hs.NetUsageMaximumTime = time.Time{}
	hs.NetUsageAverage = 0.0
	hs.NetUsageMinimum = math.MaxFloat64
	hs.NetUsageMinimumTime = time.Time{}
	hs.Perf = nil
	hs.PerfCount = 0
}

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

func (cl *Cluster) AddHostPerfomance(hi HostInfo) {
	if cl == nil {
		return
	}
	cl.LockObject()
	defer cl.UnLockObject()

	cl.CPUMhzAvg = cl.CPUMhzAvg + hi.CPUMhz
	cl.AllNumCPUPkgs = cl.AllNumCPUPkgs + hi.NumCPUPkgs
	cl.AllNumCPUCores = cl.AllNumCPUCores + hi.NumCPUCores
	cl.AllNumCPUThreads = cl.AllNumCPUThreads + hi.NumCPUThreads
	cl.AllMemorySizeGB = cl.AllMemorySizeGB + hi.MemorySizeGB
	cl.CPUUsageMaximum = cl.CPUUsageMaximum + hi.CPUUsageMaximum
	cl.CPUUsageAverage = cl.CPUUsageAverage + hi.CPUUsageAverage
	cl.CPUUsageMinimum = cl.CPUUsageMinimum + hi.CPUUsageMinimum
	cl.MemUsageMaximum = cl.MemUsageMaximum + hi.MemUsageMaximum
	cl.MemUsageAverage = cl.MemUsageAverage + hi.MemUsageAverage
	cl.MemUsageMinimum = cl.MemUsageMinimum + hi.MemUsageMinimum
	cl.DiskUsageMaximum = cl.DiskUsageMaximum + hi.DiskUsageMaximum
	cl.DiskUsageAverage = cl.DiskUsageAverage + hi.DiskUsageAverage
	cl.DiskUsageMinimum = cl.DiskUsageMinimum + hi.DiskUsageMinimum
	cl.DiskTotalReadLatencyMaximum = cl.DiskTotalReadLatencyMaximum + hi.DiskTotalReadLatencyMaximum
	cl.DiskTotalReadLatencyAverage = cl.DiskTotalReadLatencyAverage + hi.DiskTotalReadLatencyAverage
	cl.DiskTotalReadLatencyMinimum = cl.DiskTotalReadLatencyMinimum + hi.DiskTotalReadLatencyMinimum
	cl.DiskTotalWriteLatencyMaximum = cl.DiskTotalWriteLatencyMaximum + hi.DiskTotalWriteLatencyMaximum
	cl.DiskTotalWriteLatencyAverage = cl.DiskTotalWriteLatencyAverage + hi.DiskTotalWriteLatencyAverage
	cl.DiskTotalWriteLatencyMinimum = cl.DiskTotalWriteLatencyMinimum + hi.DiskTotalWriteLatencyMinimum
	cl.DiskWriteMaximum = cl.DiskWriteMaximum + hi.DiskWriteMaximum
	cl.DiskWriteAverage = cl.DiskWriteAverage + hi.DiskWriteAverage
	cl.DiskWriteMinimum = cl.DiskWriteMinimum + hi.DiskWriteMinimum
	cl.DiskReadMaximum = cl.DiskReadMaximum + hi.DiskReadMaximum
	cl.DiskReadAverage = cl.DiskReadAverage + hi.DiskReadAverage
	cl.DiskReadMinimum = cl.DiskReadMinimum + hi.DiskReadMinimum
	cl.NetUsageMaximum = cl.NetUsageMaximum + hi.NetUsageMaximum
	cl.NetUsageAverage = cl.NetUsageAverage + hi.NetUsageAverage
	cl.NetUsageMinimum = cl.NetUsageMinimum + hi.NetUsageMinimum
	cl.CPUBilling = cl.CPUBilling + hi.CPUBilling
	cl.DiskBilling = cl.DiskBilling + hi.DiskBilling
	cl.MemBilling = cl.DiskBilling + hi.MemBilling
	cl.NetBilling = cl.NetBilling + hi.NetBilling
	cl.CoreTransferRate = float64(cl.AllocatedCoreCount) / float64(cl.AllNumCPUCores)
	cl.MemoryTransferRate = float64(cl.AllocatedMemorySizeGb) / float64(cl.AllMemorySizeGB)
}

func (cl *Cluster) CalcHostPerfomance() {
	if cl == nil {
		return
	}

	cl.LockObject()
	defer cl.UnLockObject()

	count := float64(len(cl.Hosts))
	if len(cl.Hosts) == 0 {
		return
	}

	cl.CPUMhzAvg = int32(float64(cl.CPUMhzAvg) / count)
	cl.CPUUsageMaximum = cl.CPUUsageMaximum / count
	cl.CPUUsageAverage = cl.CPUUsageAverage / count
	cl.CPUUsageMinimum = cl.CPUUsageMinimum / count
	cl.MemUsageMaximum = cl.MemUsageMaximum / count
	cl.MemUsageAverage = cl.MemUsageAverage / count
	cl.MemUsageMinimum = cl.MemUsageMinimum / count
}

func (cl *Cluster) CalcPerfomance() {
	if cl == nil {
		return
	}

	//cl.LockObject()
	//defer cl.UnLockObject()

	if len(cl.Hosts) == 0 {
		return
	}
	maxCount := int64(0)
	cl.CPUUsageMinimum = math.MaxFloat64
	cl.MemUsageMinimum = math.MaxFloat64
	cl.DiskUsageMinimum = math.MaxFloat64
	cl.DiskTotalReadLatencyMinimum = math.MaxFloat64
	cl.DiskTotalWriteLatencyMinimum = math.MaxFloat64
	cl.DiskWriteMinimum = math.MaxFloat64
	cl.DiskReadMinimum = math.MaxFloat64
	cl.NetUsageMinimum = math.MaxFloat64

	for _, hs := range cl.Hosts {
		if hs.PerfCount > maxCount {
			maxCount = hs.PerfCount
		}

		cl.CPUMhzAvg += (hs.CPUMhz * int32(hs.NumCPUCores))
		cl.AllNumCPUPkgs += hs.NumCPUPkgs
		cl.AllNumCPUCores += hs.NumCPUCores
		cl.AllNumCPUThreads += hs.NumCPUThreads
		cl.AllMemorySizeGB += hs.MemorySizeGB

		if len(hs.Perf) == 0 {
			continue
		}

		cl.CPUBilling += hs.CPUBilling
		cl.DiskBilling += hs.DiskBilling
		cl.MemBilling += hs.MemBilling
		cl.NetBilling += hs.NetBilling
	}

	if cl.AllNumCPUCores > 0 {
		cl.CPUMhzAvg = cl.CPUMhzAvg / int32(cl.AllNumCPUCores)
	}

	cl.Perf = make([]*CLPerf, maxCount)

	for i := 0; i < int(maxCount); i++ {
		cl.Perf[i] = &CLPerf{}
		cl.Perf[i].Init()
		cl.Perf[i].ID = int64(i)
		cl.Perf[i].CPUUsage = 0.0
		cl.Perf[i].MemUsage = 0.0
		cl.Perf[i].DiskUsage = 0.0
		cl.Perf[i].DiskTotalReadLatency = 0.0
		cl.Perf[i].DiskTotalWriteLatency = 0.0
		cl.Perf[i].DiskWrite = 0.0
		cl.Perf[i].DiskRead = 0.0
		cl.Perf[i].NetUsage = 0.0
		cl.Perf[i].NetReceived = 0.0
		cl.Perf[i].NetTransmitted = 0.0

		for _, hs := range cl.Hosts {
			if i < len(hs.Perf) {
				if cl.Perf[i].Start.Equal(time.Time{}) {
					cl.Perf[i].Start = hs.Perf[i].Start // проверить
				}
				cl.Perf[i].CPUUsage += hs.Perf[i].CPUUsage
				cl.Perf[i].MemUsage += hs.Perf[i].MemUsage
				cl.Perf[i].DiskUsage += hs.Perf[i].DiskUsage
				cl.Perf[i].DiskTotalReadLatency += hs.Perf[i].DiskTotalReadLatency
				cl.Perf[i].DiskTotalWriteLatency += hs.Perf[i].DiskTotalWriteLatency
				cl.Perf[i].DiskWrite += hs.Perf[i].DiskWrite
				cl.Perf[i].DiskRead += hs.Perf[i].DiskRead
				cl.Perf[i].NetUsage += hs.Perf[i].NetUsage
				cl.Perf[i].NetReceived += hs.Perf[i].NetReceived
				cl.Perf[i].NetTransmitted += hs.Perf[i].NetTransmitted
			}
		}
		if len(cl.Hosts) > 0 {
			cl.Perf[i].CPUUsage /= float64(len(cl.Hosts))
			cl.Perf[i].MemUsage /= float64(len(cl.Hosts))
		}

		if cl.CPUUsageMaximum < cl.Perf[i].CPUUsage {
			cl.CPUUsageMaximum = cl.Perf[i].CPUUsage
			cl.CPUUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.CPUUsageMinimum > cl.Perf[i].CPUUsage {
			cl.CPUUsageMinimum = cl.Perf[i].CPUUsage
			cl.CPUUsageMinimumTime = cl.Perf[i].Start
		}
		cl.CPUUsageAverage += cl.Perf[i].CPUUsage

		if cl.MemUsageMaximum < cl.Perf[i].MemUsage {
			cl.MemUsageMaximum = cl.Perf[i].MemUsage
			cl.MemUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.MemUsageMinimum > cl.Perf[i].MemUsage {
			cl.MemUsageMinimum = cl.Perf[i].MemUsage
			cl.MemUsageMinimumTime = cl.Perf[i].Start
		}
		cl.MemUsageAverage += cl.Perf[i].MemUsage

		if cl.DiskUsageMaximum < cl.Perf[i].DiskUsage {
			cl.DiskUsageMaximum = cl.Perf[i].DiskUsage
			cl.DiskUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskUsageMinimum > cl.Perf[i].DiskUsage {
			cl.DiskUsageMinimum = cl.Perf[i].DiskUsage
			cl.DiskUsageMinimumTime = cl.Perf[i].Start
		}
		cl.DiskUsageAverage += cl.Perf[i].DiskUsage

		if cl.DiskWriteMaximum < cl.Perf[i].DiskWrite {
			cl.DiskWriteMaximum = cl.Perf[i].DiskWrite
			cl.DiskWriteMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskWriteMinimum > cl.Perf[i].DiskWrite {
			cl.DiskWriteMinimum = cl.Perf[i].DiskWrite
			cl.DiskWriteMinimumTime = cl.Perf[i].Start
		}
		cl.DiskWriteAverage += cl.Perf[i].DiskWrite

		if cl.DiskReadMaximum < cl.Perf[i].DiskRead {
			cl.DiskReadMaximum = cl.Perf[i].DiskRead
			cl.DiskReadMaximumTime = cl.Perf[i].Start
		}

		if cl.DiskReadMinimum > cl.Perf[i].DiskRead {
			cl.DiskReadMinimum = cl.Perf[i].DiskRead
			cl.DiskReadMinimumTime = cl.Perf[i].Start
		}
		cl.DiskReadAverage += cl.Perf[i].DiskRead

		if cl.DiskTotalReadLatencyMaximum < cl.Perf[i].DiskTotalReadLatency {
			cl.DiskTotalReadLatencyMaximum = cl.Perf[i].DiskTotalReadLatency
			cl.DiskTotalReadLatencyMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskTotalReadLatencyMinimum > cl.Perf[i].DiskTotalReadLatency {
			cl.DiskTotalReadLatencyMinimum = cl.Perf[i].DiskTotalReadLatency
			cl.DiskTotalReadLatencyMinimumTime = cl.Perf[i].Start
		}
		cl.DiskTotalReadLatencyAverage += cl.Perf[i].DiskTotalReadLatency

		if cl.DiskTotalWriteLatencyMaximum < cl.Perf[i].DiskTotalWriteLatency {
			cl.DiskTotalWriteLatencyMaximum = cl.Perf[i].DiskTotalWriteLatency
			cl.DiskTotalWriteLatencyMaximumTime = cl.Perf[i].Start
		}
		if cl.DiskTotalWriteLatencyMinimum > cl.Perf[i].DiskTotalWriteLatency {
			cl.DiskTotalWriteLatencyMinimum = cl.Perf[i].DiskTotalWriteLatency
			cl.DiskTotalWriteLatencyMinimumTime = cl.Perf[i].Start
		}
		cl.DiskTotalWriteLatencyAverage += cl.Perf[i].DiskTotalWriteLatency

		if cl.NetUsageMaximum < cl.Perf[i].NetUsage {
			cl.NetUsageMaximum = cl.Perf[i].NetUsage
			cl.NetUsageMaximumTime = cl.Perf[i].Start
		}
		if cl.NetUsageMinimum > cl.Perf[i].NetUsage {
			cl.NetUsageMinimum = cl.Perf[i].NetUsage
			cl.NetUsageMinimumTime = cl.Perf[i].Start
		}
		cl.NetUsageAverage += cl.Perf[i].NetUsage
	}
	switch {
	case maxCount > 0:
		cl.CPUUsageAverage /= float64(maxCount)
		cl.MemUsageAverage /= float64(maxCount)
		cl.DiskUsageAverage /= float64(maxCount)
		cl.DiskWriteAverage /= float64(maxCount)
		cl.DiskReadAverage /= float64(maxCount)
		cl.DiskTotalReadLatencyAverage /= float64(maxCount)
		cl.DiskTotalWriteLatencyAverage /= float64(maxCount)
		cl.NetUsageAverage /= float64(maxCount)
	default:
		cl.CPUUsageMaximum = 0.0
		cl.CPUUsageAverage = 0.0
		cl.CPUUsageMinimum = 0.0
		cl.MemUsageMaximum = 0.0
		cl.MemUsageAverage = 0.0
		cl.MemUsageMinimum = 0.0
		cl.DiskUsageMaximum = 0.0
		cl.DiskUsageAverage = 0.0
		cl.DiskUsageMinimum = 0.0
		cl.DiskTotalReadLatencyMaximum = 0.0
		cl.DiskTotalReadLatencyAverage = 0.0
		cl.DiskTotalReadLatencyMinimum = 0.0
		cl.DiskTotalWriteLatencyMaximum = 0.0
		cl.DiskTotalWriteLatencyAverage = 0.0
		cl.DiskTotalWriteLatencyMinimum = 0.0
		cl.DiskWriteMaximum = 0.0
		cl.DiskWriteAverage = 0.0
		cl.DiskWriteMinimum = 0.0
		cl.DiskReadMaximum = 0.0
		cl.DiskReadAverage = 0.0
		cl.DiskReadMinimum = 0.0
		cl.NetUsageMaximum = 0.0
		cl.NetUsageAverage = 0.0
		cl.NetUsageMinimum = 0.0
	}
	cl.PerfCount = (int64)(len(cl.Perf))
}

func CheckAndCorrectHSPerf(perf *HSPerf) {
	if perf == nil {
		return
	}

	if perf.CPUUsage > 100 {
		perf.CPUUsage = 100.0
	}
	if perf.MemUsage > 100 {
		perf.MemUsage = 100.0

	}
	if math.IsNaN(perf.CPUUsage) {
		perf.CPUUsage = 0.0
	}
	if math.IsNaN(perf.MemUsage) {
		perf.MemUsage = 0.0
	}
	if math.IsNaN(perf.DiskUsage) {
		perf.DiskUsage = 0.0
	}
	if math.IsNaN(perf.NetUsage) {
		perf.NetUsage = 0.0
	}
	if math.IsNaN(perf.DiskTotalReadLatency) {
		perf.DiskTotalReadLatency = 0.0
	}
	if math.IsNaN(perf.DiskTotalWriteLatency) {
		perf.DiskTotalWriteLatency = 0.0
	}
	if math.IsNaN(perf.DiskWrite) {
		perf.DiskWrite = 0.0
	}
	if math.IsNaN(perf.DiskRead) {
		perf.DiskRead = 0.0
	}
}

func SetHostPerfMetrics(perf *HSPerf, metric performance.EntityMetric) {
	if perf == nil {
		return
	}

	for _, v := range metric.Value {
		if len(v.Value) > 0 {
			//fmt.Printf("1 - %s\n", v.Name)
			switch v.Name {
			case "net.transmitted.average":
				if v.Instance == "" {
					perf.NetTransmitted = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				}
				//perf.NetTransmittedMap = append(perf.NetTransmittedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
			case "net.received.average":
				if v.Instance == "" {
					perf.NetReceived = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				}
				//perf.NetReceivedMap = append(perf.NetReceivedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
			case "cpu.usage.average":
				if v.Instance == "" {
					perf.CPUUsage = float64(maxValue(v.Value)) / 100.0
				} else {
					//perf.CPUUsageMap = append(perf.CPUUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 100.0})
				}
			case "cpu.utilization.average":
				if v.Instance == "" {
					perf.CPUUtilization = float64(maxValue(v.Value)) / 100.0
				}
			case "mem.usage.average":
				if v.Instance == "" {
					perf.MemUsage = float64(maxValue(v.Value)) / 100.0
				}
			case "disk.usage.average":
				if v.Instance == "" {
					perf.DiskUsage = float64(maxValue(v.Value)) / 1024.0
				}
			case "net.usage.average": //может быть несколько сетевых карт с именем в Instance
				if v.Instance == "4000" {
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
				if v.Instance == "" {
					perf.NetUsage = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "disk.deviceReadLatency.average":
				if v.Instance == "" {
					perf.DiskTotalReadLatency = float64(maxValue(v.Value)) //ms
				} else {
					//perf.DiskTotalReadLatencyMap = append(perf.DiskTotalReadLatencyMap, Metric{v.Instance, float64(maxValue(v.Value))}) //ms
				}
			case "disk.deviceWriteLatency.average":
				if v.Instance == "" {
					perf.DiskTotalWriteLatency = float64(maxValue(v.Value)) //ms
				} else {
					//perf.DiskTotalWriteLatencyMap = append(perf.DiskTotalWriteLatencyMap, Metric{v.Instance, float64(maxValue(v.Value))}) //ms
				}
			case "disk.write.average":
				if v.Instance == "" {
					perf.DiskWrite = float64(maxValue(v.Value)) / 1024.0
				} else {
					//perf.DiskWriteMap = append(perf.DiskWriteMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0})
				}
			case "disk.read.average":
				if v.Instance == "" {
					perf.DiskRead = float64(maxValue(v.Value)) / 1024.0
				} else {
					//perf.DiskReadMap = append(perf.DiskReadMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0})
				}
			}
		}
	}
}

func SetHostMetric(hi *HostInfo, perf HSPerf) {
	if hi == nil {
		return
	}

	hi.LockObject()
	defer hi.UnLockObject()

	hi.CPUBilling = hi.CPUBilling + float64(perf.CPUUsage)*float64(hi.CPUMhz)*float64(hi.NumCPUCores)/180.0 //утилизация процессора частота в мегагерцах умноженная процент утилизации и промежуток времени
	hi.MemBilling = hi.MemBilling + float64(perf.MemUsage)*hi.MemorySizeGB*1024.0/180.0                     // утилизация памяти мегабайт часах

	if hi.CPUUsageMaximum < perf.CPUUsage {
		hi.CPUUsageMaximum = perf.CPUUsage
		hi.CPUUsageMaximumTime = perf.Start
	}
	if hi.CPUUsageMinimum > perf.CPUUsage {
		hi.CPUUsageMinimum = perf.CPUUsage
		hi.CPUUsageMinimumTime = perf.Start
	}
	hi.CPUUsageAverage = hi.CPUUsageAverage + perf.CPUUsage

	if hi.MemUsageMaximum < perf.MemUsage {
		hi.MemUsageMaximum = perf.MemUsage
		hi.MemUsageMaximumTime = perf.Start
	}
	if hi.MemUsageMinimum > perf.MemUsage {
		hi.MemUsageMinimum = perf.MemUsage
		hi.MemUsageMinimumTime = perf.Start
	}
	hi.MemUsageAverage = hi.MemUsageAverage + perf.MemUsage

	if hi.NetUsageMaximum < perf.NetUsage {
		hi.NetUsageMaximum = perf.NetUsage
		hi.NetUsageMaximumTime = perf.Start
	}
	if hi.NetUsageMinimum > perf.NetUsage {
		hi.NetUsageMinimum = perf.NetUsage
		hi.NetUsageMinimumTime = perf.Start
	}
	hi.NetUsageAverage = hi.NetUsageAverage + perf.NetUsage

	if hi.DiskUsageMaximum < perf.DiskUsage {
		hi.DiskUsageMaximum = perf.DiskUsage
		hi.DiskUsageMaximumTime = perf.Start
	}
	if hi.DiskUsageMinimum > perf.DiskUsage {
		hi.DiskUsageMinimum = perf.DiskUsage
		hi.DiskUsageMinimumTime = perf.Start
	}
	hi.DiskUsageAverage = hi.DiskUsageAverage + perf.DiskUsage

	if hi.DiskReadMaximum < perf.DiskRead {
		hi.DiskReadMaximum = perf.DiskRead
		hi.DiskReadMaximumTime = perf.Start
	}
	if hi.DiskReadMinimum > perf.DiskRead {
		hi.DiskReadMinimum = perf.DiskRead
		hi.DiskReadMinimumTime = perf.Start
	}
	hi.DiskReadAverage = hi.DiskReadAverage + perf.DiskRead

	if hi.DiskWriteMaximum < perf.DiskWrite {
		hi.DiskWriteMaximum = perf.DiskWrite
		hi.DiskWriteMaximumTime = perf.Start
	}
	if hi.DiskWriteMinimum > perf.DiskWrite {
		hi.DiskWriteMinimum = perf.DiskWrite
		hi.DiskWriteMinimumTime = perf.Start
	}
	hi.DiskWriteAverage = hi.DiskWriteAverage + perf.DiskWrite

	if hi.DiskTotalReadLatencyMaximum < perf.DiskTotalReadLatency {
		hi.DiskTotalReadLatencyMaximum = perf.DiskTotalReadLatency
		hi.DiskTotalReadLatencyMaximumTime = perf.Start
	}
	if hi.DiskTotalReadLatencyMinimum > perf.DiskTotalReadLatency {
		hi.DiskTotalReadLatencyMinimum = perf.DiskTotalReadLatency
		hi.DiskTotalReadLatencyMinimumTime = perf.Start
	}
	hi.DiskTotalReadLatencyAverage = hi.DiskTotalReadLatencyAverage + perf.DiskTotalReadLatency

	if hi.DiskTotalWriteLatencyMaximum < perf.DiskTotalWriteLatency {
		hi.DiskTotalWriteLatencyMaximum = perf.DiskTotalWriteLatency
		hi.DiskTotalWriteLatencyMaximumTime = perf.Start
	}
	if hi.DiskTotalWriteLatencyMinimum > perf.DiskTotalWriteLatency {
		hi.DiskTotalWriteLatencyMinimum = perf.DiskTotalWriteLatency
		hi.DiskTotalWriteLatencyMinimumTime = perf.Start
	}
	hi.DiskTotalWriteLatencyAverage = hi.DiskTotalWriteLatencyAverage + perf.DiskTotalWriteLatency
}

func SetVMPerfMetrics(perf *VMPerf, metric performance.EntityMetric) {
	if perf == nil {
		return
	}

	for _, v := range metric.Value {
		if len(v.Value) > 0 {
			//fmt.Printf("1 - %s\n", v.Name)
			switch v.Name {
			case "net.transmitted.average":
				if v.Instance == "" {
					perf.NetTransmitted = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				}
				if v.Instance == "4000" {
					//perf.NetTransmittedMap = append(perf.NetTransmittedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "net.received.average":
				perf.NetReceived = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				if v.Instance == "4000" {
					//perf.NetReceivedMap = append(perf.NetReceivedMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "cpu.usage.average":
				if v.Instance == "" {
					perf.CPUUsage = float64(maxValue(v.Value)) / 100.0 //unit:"percent"
				} else {
					//perf.CPUUsageMap = append(perf.CPUUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 100.0}) //unit:"percent"
				}
			case "mem.usage.average":
				if v.Instance == "" {
					perf.MemUsage = float64(maxValue(v.Value)) / 100.0 //unit:"percent"
				}
			case "disk.usage.average":
				if v.Instance == "" {
					perf.DiskUsage = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
				} else {
					//perf.DiskUsageMap = append(perf.DiskUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "net.usage.average": //может быть несколько сетевых карт с именем в Instance
				if v.Instance == "4000" {
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
				if v.Instance == "" {
					perf.NetUsage = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
					//perf.NetUsageMap = append(perf.NetUsageMap, Metric{v.Instance, float64(maxValue(v.Value)) / 1024.0}) //unit:"kiloBytesPerSecond"
				}
			case "virtualDisk.totalReadLatency.average":
				perf.DiskTotalReadLatency = float64(maxValue(v.Value)) //unit:"millisecond"
			case "virtualDisk.totalWriteLatency.average":
				perf.DiskTotalWriteLatency = float64(maxValue(v.Value)) //unit:"millisecond"
			case "virtualDisk.write.average":
				perf.DiskWrite = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
			case "virtualDisk.read.average":
				perf.DiskRead = float64(maxValue(v.Value)) / 1024.0 //unit:"kiloBytesPerSecond"
			}
		}
	}
}

func CheckAndCorrectVMPerf(perf *VMPerf) {
	if perf == nil {
		return
	}

	//	perf.LockGetMetric()
	//	defer perf.UnLockGetMetric()

	if perf.CPUUsage > 100.0 {
		perf.CPUUsage = 100.0
	}
	if perf.MemUsage > 100.0 {
		perf.MemUsage = 100.0
	}
	if math.IsNaN(perf.CPUUsage) {
		perf.CPUUsage = 0.0
	}
	if math.IsNaN(perf.MemUsage) {
		perf.MemUsage = 0.0
	}
	if math.IsNaN(perf.DiskUsage) {
		perf.DiskUsage = 0.0
	}
	if math.IsNaN(perf.NetUsage) {
		perf.NetUsage = 0.0
	}
	if math.IsNaN(perf.DiskTotalReadLatency) {
		perf.DiskTotalReadLatency = 0.0
	}
	if math.IsNaN(perf.DiskTotalWriteLatency) {
		perf.DiskTotalWriteLatency = 0.0
	}
	if math.IsNaN(perf.DiskWrite) {
		perf.DiskWrite = 0.0
	}
	if math.IsNaN(perf.DiskRead) {
		perf.DiskRead = 0.0
	}
}

func SetVMMetric(vmP *VMPoint, perf VMPerf, cpuMhz float64) {
	if vmP == nil {
		return
	}
	vmP.LockObject()
	defer vmP.UnLockObject()
	vmP.CPUMhz = cpuMhz
	vmP.CPUBilling = vmP.CPUBilling + float64(perf.CPUUsage)*float64(cpuMhz)*float64(vmP.NumCores)/180.0 //утилизация процессора частота в мегагерцах умноженная процент утилизации и промежуток времени
	vmP.MemBilling = vmP.MemBilling + float64(perf.MemUsage)*vmP.MemorySizeGB*1024.0/180.0               // утилизация памяти мегабайт часах

	if vmP.CPUUsageMaximum < perf.CPUUsage {
		vmP.CPUUsageMaximum = perf.CPUUsage
		vmP.CPUUsageMaximumTime = perf.Start
	}
	if vmP.CPUUsageMinimum > perf.CPUUsage {
		vmP.CPUUsageMinimum = perf.CPUUsage
		vmP.CPUUsageMinimumTime = perf.Start
	}
	vmP.CPUUsageAverage = vmP.CPUUsageAverage + perf.CPUUsage

	if vmP.MemUsageMaximum < perf.MemUsage {
		vmP.MemUsageMaximum = perf.MemUsage
		vmP.MemUsageMaximumTime = perf.Start
	}
	if vmP.MemUsageMinimum > perf.MemUsage {
		vmP.MemUsageMinimum = perf.MemUsage
		vmP.MemUsageMinimumTime = perf.Start
	}
	vmP.MemUsageAverage = vmP.MemUsageAverage + perf.MemUsage

	if vmP.NetUsageMaximum < perf.NetUsage {
		vmP.NetUsageMaximum = perf.NetUsage
		vmP.NetUsageMaximumTime = perf.Start
	}
	if vmP.NetUsageMinimum > perf.NetUsage {
		vmP.NetUsageMinimum = perf.NetUsage
		vmP.NetUsageMinimumTime = perf.Start
	}
	vmP.NetUsageAverage = vmP.NetUsageAverage + perf.NetUsage

	if vmP.DiskUsageMaximum < perf.DiskUsage {
		vmP.DiskUsageMaximum = perf.DiskUsage
		vmP.DiskUsageMaximumTime = perf.Start
	}
	if vmP.DiskUsageMinimum > perf.DiskUsage {
		vmP.DiskUsageMinimum = perf.DiskUsage
		vmP.DiskUsageMinimumTime = perf.Start
	}
	vmP.DiskUsageAverage = vmP.DiskUsageAverage + perf.DiskUsage

	if vmP.DiskReadMaximum < perf.DiskRead {
		vmP.DiskReadMaximum = perf.DiskRead
		vmP.DiskReadMaximumTime = perf.Start
	}
	if vmP.DiskReadMinimum > perf.DiskRead {
		vmP.DiskReadMinimum = perf.DiskRead
		vmP.DiskReadMinimumTime = perf.Start
	}
	vmP.DiskReadAverage = vmP.DiskReadAverage + perf.DiskRead

	if vmP.DiskWriteMaximum < perf.DiskWrite {
		vmP.DiskWriteMaximum = perf.DiskWrite
		vmP.DiskWriteMaximumTime = perf.Start
	}
	if vmP.DiskWriteMinimum > perf.DiskWrite {
		vmP.DiskWriteMinimum = perf.DiskWrite
		vmP.DiskWriteMinimumTime = perf.Start
	}
	vmP.DiskWriteAverage = vmP.DiskWriteAverage + perf.DiskWrite

	if vmP.DiskTotalReadLatencyMaximum < perf.DiskTotalReadLatency {
		vmP.DiskTotalReadLatencyMaximum = perf.DiskTotalReadLatency
		vmP.DiskTotalReadLatencyMaximumTime = perf.Start
	}
	if vmP.DiskTotalReadLatencyMinimum > perf.DiskTotalReadLatency {
		vmP.DiskTotalReadLatencyMinimum = perf.DiskTotalReadLatency
		vmP.DiskTotalReadLatencyMinimumTime = perf.Start
	}
	vmP.DiskTotalReadLatencyAverage = vmP.DiskTotalReadLatencyAverage + perf.DiskTotalReadLatency

	if vmP.DiskTotalWriteLatencyMaximum < perf.DiskTotalWriteLatency {
		vmP.DiskTotalWriteLatencyMaximum = perf.DiskTotalWriteLatency
		vmP.DiskTotalWriteLatencyMaximumTime = perf.Start
	}
	if vmP.DiskTotalWriteLatencyMinimum > perf.DiskTotalWriteLatency {
		vmP.DiskTotalWriteLatencyMinimum = perf.DiskTotalWriteLatency
		vmP.DiskTotalWriteLatencyMinimumTime = perf.Start
	}
	vmP.DiskTotalWriteLatencyAverage = vmP.DiskTotalWriteLatencyAverage + perf.DiskTotalWriteLatency
}

func VmfsStrLunToUID(str string) string {
	size := len(str)
	if size < 32 {
		return ""
	}
	return strings.ToUpper(str[0:2] + ":" + str[2:4] + ":" + str[4:6] + ":" + str[6:8] + ":" + str[8:10] + ":" + str[10:12] + ":" + str[12:14] + ":" + str[14:16] + ":" +
		str[16:18] + ":" + str[18:20] + ":" + str[20:22] + ":" + str[22:24] + ":" + str[24:26] + ":" + str[26:28] + ":" + str[28:30] + ":" + str[30:32])
}
