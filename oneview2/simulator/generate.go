package simulator

import (
	"fmt"
	"strings"
)

type hwSpec struct {
	shortModel   string
	model        string
	formFactor   string
	uHeight      int
	isBlade      bool
	cpuCount     int
	coresPerCPU  int
	cpuModel     string
	speedMHz     int
	memVariants  []int
	diskVariants []int
}

var hpModels = []hwSpec{
	{"ProLiant DL380 Gen10", "HPE ProLiant DL380 Gen10", "RackMount", 2, false, 2, 16, "Intel Xeon Gold 6248R", 3000, []int{8, 12, 16, 24}, []int{4, 6, 8, 10, 12}},
	{"ProLiant DL360 Gen10", "HPE ProLiant DL360 Gen10", "RackMount", 1, false, 2, 12, "Intel Xeon Silver 4214R", 2400, []int{4, 8, 12, 16}, []int{2, 4, 6, 8}},
	{"ProLiant DL385 Gen10 Plus", "HPE ProLiant DL385 Gen10 Plus", "RackMount", 2, false, 2, 32, "AMD EPYC 7543", 2800, []int{8, 16, 24, 32}, []int{4, 8, 10, 12}},
	{"ProLiant ML350 Gen10", "HPE ProLiant ML350 Gen10", "Tower", 2, false, 2, 10, "Intel Xeon Silver 4210", 2200, []int{4, 8, 12}, []int{2, 4, 6, 8}},
	{"ProLiant BL460c Gen10", "HPE ProLiant BL460c Gen10", "Blade", 0, true, 2, 16, "Intel Xeon Gold 6230", 2600, []int{8, 12, 16}, []int{2, 4, 6}},
	{"ProLiant BL660c Gen10", "HPE ProLiant BL660c Gen10", "Blade", 0, true, 4, 16, "Intel Xeon Gold 6248", 2500, []int{16, 24, 32}, []int{2, 4, 6, 8}},
	{"Synergy 480 Gen10", "HPE Synergy 480 Gen10 Compute Module", "Blade", 0, true, 2, 20, "Intel Xeon Gold 6252N", 2800, []int{12, 16, 24}, []int{2, 4, 6}},
}

var dcNames = []string{"DC-Moscow", "DC-SaintPetersburg", "DC-Novosibirsk", "DC-Kazan", "DC-Yekaterinburg"}

// GenerateInventory builds datacenter topology and serverCount servers.
func GenerateInventory(cfg InstanceConfig) *Inventory {
	prefix := fmt.Sprintf("sim-%d", cfg.APIVersion)
	global := cfg.APIVersion <= 300
	count := cfg.ServerCount
	if count <= 0 {
		count = 300
	}

	const (
		roomsPerDC   = 4
		racksPerRoom = 3
	)
	serversPerRack := count / (len(dcNames) * roomsPerDC * racksPerRoom)
	if serversPerRack <= 0 {
		serversPerRack = 1
	}

	inv := &Inventory{
		APIVersion:  cfg.APIVersion,
		GlobalDash:  global,
		Servers:     make(map[string]*ServerRecord),
		ServerOrder: make([]string, 0, count),
		Enclosures:  make(map[string]*EnclosureRecord),
		Racks:       make(map[string]*RackRecord),
		Datacenters: make([]*DatacenterRecord, 0, len(dcNames)),
		Firmware:    make([]FirmwareRecord, 0, count*3),
	}

	rackByKey := make(map[string]*RackRecord)
	dcByName := make(map[string]*DatacenterRecord)

	for dcIdx, dcName := range dcNames {
		dcID := fmt.Sprintf("dc-%d-%d", cfg.APIVersion, dcIdx)
		dcURI := uri(prefix, "datacenters", dcID)
		dc := &DatacenterRecord{
			URI:             dcURI,
			OriginalURI:     dcURI,
			UUID:            uuidFor(prefix+"-dc", dcIdx),
			Name:            dcName,
			Status:          "OK",
			State:           "Configured",
			CoolingCapacity: 500 + dcIdx*50,
			Width:           20,
			Height:          30,
			Currency:        "USD",
			Contents:        nil,
			RackInventory:   nil,
		}
		inv.Datacenters = append(inv.Datacenters, dc)
		dcByName[dcName] = dc

		for room := 1; room <= roomsPerDC; room++ {
			roomName := fmt.Sprintf("Room-%d%02d", dcIdx+1, room)
			for rack := 1; rack <= racksPerRoom; rack++ {
				rackKey := fmt.Sprintf("%s/%s/rack-%d", dcName, roomName, rack)
				rackID := fmt.Sprintf("rk-%d-%d-%d-%d", cfg.APIVersion, dcIdx, room, rack)
				rackURI := uri(prefix, "racks", rackID)
				r := &RackRecord{
					ID:           rackID,
					URI:          rackURI,
					OriginalURI:  rackURI,
					UUID:         uuidFor(prefix+"-rk", dcIdx*100+room*10+rack),
					Name:         fmt.Sprintf("%s-%s-R%02d", dcName, roomName, rack),
					Model:        "HPE 1075 G2 Rack",
					SerialNumber: fmt.Sprintf("RCK%s%04d", strings.ToUpper(prefix[:3]), dcIdx*1000+room*100+rack),
					UHeight:      42,
					DCURI:        dcURI,
					Room:         roomName,
					RackMounts:   nil,
				}
				inv.Racks[rackURI] = r
				rackByKey[rackKey] = r
				dc.Contents = append(dc.Contents, dcContent{ResourceURI: rackURI, X: rack, Y: room})
				dc.RackInventory = append(dc.RackInventory, rackInventoryEntry{Name: r.Name, OriginalURI: rackURI})
			}
		}
	}

	encCounter := 0
	nextUSlot := make(map[string]int)
	bladeInEnc := make(map[string]int)

	for i := 0; i < count; i++ {
		spec := hpModels[i%len(hpModels)]
		dcIdx := i / (roomsPerDC * racksPerRoom * serversPerRack)
		if dcIdx >= len(dcNames) {
			dcIdx = len(dcNames) - 1
		}
		rem := i % (roomsPerDC * racksPerRoom * serversPerRack)
		room := rem/(racksPerRoom*serversPerRack) + 1
		rem2 := rem % (racksPerRoom * serversPerRack)
		rack := rem2/serversPerRack + 1
		slotInRack := rem2 % serversPerRack

		dcName := dcNames[dcIdx]
		roomName := fmt.Sprintf("Room-%d%02d", dcIdx+1, room)
		rackKey := fmt.Sprintf("%s/%s/rack-%d", dcName, roomName, rack)
		rackRec := rackByKey[rackKey]

		srvID := fmt.Sprintf("sh-%d-%d", cfg.APIVersion, i+1)
		srvURI := uri(prefix, "server-hardware", srvID)
		memCount := spec.memVariants[i%len(spec.memVariants)]
		diskCount := spec.diskVariants[i%len(spec.diskVariants)]
		totalMemMiB := memCount * 32768
		power := "On"
		if i%17 == 0 {
			power = "Off"
		}

		srv := &ServerRecord{
			ID:                  srvID,
			URI:                 srvURI,
			OriginalURI:         srvURI,
			UUID:                uuidFor(prefix+"-srv", i),
			SerialNumber:        fmt.Sprintf("SGH%s%06d", serialPrefix(cfg.APIVersion), i+1),
			VirtualSerialNumber: fmt.Sprintf("VSN%s%06d", serialPrefix(cfg.APIVersion), i+1),
			Model:               spec.model,
			ShortModel:          spec.shortModel,
			FormFactor:          spec.formFactor,
			PowerState:          power,
			Status:              "OK",
			State:               "Monitored",
			RackURI:             rackRec.URI,
			ProcessorCount:      spec.cpuCount,
			ProcessorCoreCount:    spec.coresPerCPU,
			ProcessorSpeedMhz:   spec.speedMHz,
			ProcessorType:       spec.cpuModel,
			MemoryMb:            totalMemMiB,
			Processors:          buildProcessors(i, spec),
			Memory:              buildMemory(i, memCount),
			Controllers:         buildControllers(i, diskCount),
			Devices:             buildDevices(i, spec),
			PortMap:             buildPorts(i),
			Firmware:            buildServerFirmware(i, srvURI),
		}

		if spec.isBlade {
			srv.IsBlade = true
			encKey := rackRec.URI + fmt.Sprintf("-enc-%d", slotInRack/8)
			enc := inv.Enclosures[encKey]
			if enc == nil {
				encCounter++
				encID := fmt.Sprintf("enc-%d-%d", cfg.APIVersion, encCounter)
				encURI := uri(prefix, "enclosures", encID)
				encType := "C7000"
				encModel := "HPE BladeSystem c7000 Enclosure G3"
				if strings.Contains(spec.shortModel, "Synergy") {
					encType = "Synergy"
					encModel = "HPE Synergy 12000 Frame"
				}
				enc = &EnclosureRecord{
					ID:             encID,
					URI:            encURI,
					OriginalURI:    encURI,
					UUID:           uuidFor(prefix+"-enc", encCounter),
					Name:           fmt.Sprintf("%s-ENC%02d", rackRec.Name, encCounter%100),
					SerialNumber:   fmt.Sprintf("ENC%s%05d", serialPrefix(cfg.APIVersion), encCounter),
					EnclosureModel: encModel,
					EnclosureType:  encType,
					RackName:       rackRec.Name,
					RackURI:        rackRec.URI,
					DCURI:          rackRec.DCURI,
					DeviceBayCount: 16,
					DeviceBays:     make([]deviceBay, 0, 16),
				}
				inv.Enclosures[encKey] = enc
				inv.Enclosures[encURI] = enc
				u := nextUSlot[rackRec.URI]
				if u == 0 {
					u = 38
				}
				nextUSlot[rackRec.URI] = u - 10
				rackRec.RackMounts = append(rackRec.RackMounts, rackMount{
					MountURI: encURI,
					TopUSlot: u,
					UHeight:  10,
					Location: "CenterFront",
				})
				bladeInEnc[encKey] = 0
			}
			bay := bladeInEnc[encKey] + 1
			bladeInEnc[encKey] = bay
			srv.LocationURI = enc.URI
			srv.Position = bay
			srv.Name = fmt.Sprintf("%s, bay %d", enc.Name, bay)
			srv.ServerName = fmt.Sprintf("srv-blade-%04d", i+1)
			enc.DeviceBays = append(enc.DeviceBays, deviceBay{
				Type:           "DeviceBayV200",
				BayNumber:      bay,
				DevicePresence: "Present",
				DeviceURI:      srvURI,
				DeviceBayType:  "C7000DeviceBay",
				BayPowerState:  power,
			})
		} else {
			srv.Name = fmt.Sprintf("srv-%03d", i+1)
			srv.ServerName = srv.Name
			u := nextUSlot[rackRec.URI]
			if u == 0 {
				u = 40
			}
			srv.TopUSlot = u
			srv.UHeight = spec.uHeight
			if srv.UHeight <= 0 {
				srv.UHeight = 2
			}
			nextUSlot[rackRec.URI] = u - srv.UHeight - 1
			rackRec.RackMounts = append(rackRec.RackMounts, rackMount{
				MountURI: srvURI,
				TopUSlot: srv.TopUSlot,
				UHeight:  srv.UHeight,
				Location: "CenterFront",
			})
		}

		inv.Servers[srvURI] = srv
		inv.ServerOrder = append(inv.ServerOrder, srvURI)
		for _, fw := range srv.Firmware {
			row := fw
			row.ServerHardwareOrig = srvURI
			if global {
				row.ServerHardwareURI = srv.OriginalURI
			}
			inv.Firmware = append(inv.Firmware, row)
		}
	}

	return inv
}

func serialPrefix(api int) string {
	if api <= 300 {
		return "GD3"
	}
	return fmt.Sprintf("%04d", api)[:4]
}

func uri(prefix, category, id string) string {
	return fmt.Sprintf("/rest/%s/%s", category, id)
}

func buildProcessors(seed int, spec hwSpec) []ProcessorRecord {
	out := make([]ProcessorRecord, 0, spec.cpuCount)
	for c := 0; c < spec.cpuCount; c++ {
		out = append(out, ProcessorRecord{
			ID:                fmt.Sprintf("%d", c+1),
			Socket:            fmt.Sprintf("CPU %d", c+1),
			Model:             spec.cpuModel,
			Manufacturer:      manufacturer(spec.cpuModel),
			TotalCores:        spec.coresPerCPU,
			TotalThreads:      spec.coresPerCPU * 2,
			MaxSpeedMHz:       spec.speedMHz,
			OperatingSpeedMHz: spec.speedMHz - 200,
			Status:            status{Health: "OK", State: "Enabled"},
		})
	}
	return out
}

func manufacturer(cpu string) string {
	if strings.Contains(cpu, "AMD") {
		return "AMD"
	}
	return "Intel"
}

func buildMemory(seed, count int) []MemoryRecord {
	out := make([]MemoryRecord, 0, count)
	for m := 0; m < count; m++ {
		socket := m/6 + 1
		channel := m % 3
		slot := m % 6
		out = append(out, MemoryRecord{
			Name:              fmt.Sprintf("DIMM %d", m+1),
			DeviceLocator:     fmt.Sprintf("PROC %d DIMM %d", socket, slot+1),
			Manufacturer:      "HPE",
			PartNumber:        "P03053-091",
			SerialNumber:      fmt.Sprintf("MEM%06d%02d", seed+1, m+1),
			CapacityMiB:       32768,
			OperatingSpeedMhz: 2933,
			BaseModuleType:    "RDIMM",
			MemoryDeviceType:  "DDR4",
			MemoryType:        "DDR4",
			RankCount:         2,
			ErrorCorrection:   "MultiBitECC",
			MemoryLocation: memoryLocation{
				Socket:           socket,
				Channel:          channel,
				Slot:             slot,
				MemoryController: socket - 1,
			},
			Oem: memoryOem{Hpe: struct {
				DIMMStatus string   `json:"DIMMStatus"`
				Attributes []string `json:"Attributes"`
			}{DIMMStatus: "Good", Attributes: []string{"HPE SmartMemory"}}},
			Status: status{Health: "OK", State: "Enabled"},
		})
	}
	return out
}

func buildControllers(seed, diskCount int) []ControllerRecord {
	drives := make([]DriveRecord, 0, diskCount)
	for d := 0; d < diskCount; d++ {
		media := "SSD"
		capMiB := 457862
		model := "VK000480GWSRT"
		if d%3 == 2 {
			media = "HDD"
			capMiB = 1907729
			model = "EG001800JWJNN"
		}
		use := "Data"
		if diskCount >= 4 && d == diskCount-1 {
			use = "Spare" // последний диск - горячий резерв (для проверки отображения)
		}
		drives = append(drives, DriveRecord{
			Location:        fmt.Sprintf("1:1:%d", d+1),
			Model:           model,
			SerialNumber:    fmt.Sprintf("DRV%06d%02d", seed+1, d+1),
			MediaType:       media,
			InterfaceType:   "SAS",
			CapacityMiB:     capMiB,
			DiskDriveUse:    use,
			FirmwareVersion: nestedFirmware{Current: struct{ VersionString string `json:"VersionString"` }{VersionString: "HPD9"}},
			EncryptedDrive:  d%5 == 0,
			Status:          status{Health: "OK", State: "Enabled"},
		})
	}
	volCap := 0
	for _, d := range drives {
		volCap += d.CapacityMiB
	}
	if diskCount >= 2 {
		volCap = volCap / 2
	}
	ctrl := ControllerRecord{
		Name:               "Smart Array P408i-a SR Gen10",
		Model:              "Smart Array P408i-a SR Gen10",
		SerialNumber:       fmt.Sprintf("CTL%06d", seed+1),
		AdapterType:        "SmartArray",
		Location:           "Slot 0",
		CacheMemorySizeMiB: 2048,
		FirmwareVersion:    nestedFirmware{Current: struct{ VersionString string `json:"VersionString"` }{VersionString: "5.61"}},
		Status:             status{Health: "OK", State: "Enabled"},
		PhysicalDrives:     drives,
		LogicalDrives: []VolumeRecord{{
			LogicalDriveName:   "Logical Drive 1",
			LogicalDriveNumber: 1,
			Raid:               raidLevel(diskCount),
			LogicalDriveType:   "Parity",
			CapacityMiB:        volCap,
			Status:             status{Health: "OK", State: "Enabled"},
		}},
	}
	return []ControllerRecord{ctrl}
}

func raidLevel(disks int) string {
	switch {
	case disks >= 6:
		return "Raid6"
	case disks >= 4:
		return "Raid5"
	case disks >= 2:
		return "Raid1"
	default:
		return "Raid0"
	}
}

func buildDevices(seed int, spec hwSpec) []DeviceRecord {
	return []DeviceRecord{
		{
			ID:              "1",
			Name:            "HPE Ethernet 10Gb 2-port 562SFP+ Adapter",
			DeviceType:      "Ethernet",
			Location:        "Mezzanine",
			Manufacturer:    "HPE",
			PartNumber:      "727055-B21",
			SerialNumber:    fmt.Sprintf("NIC%06dA", seed+1),
			FirmwareVersion: "20.28.41",
			Status:          status{Health: "OK", State: "Enabled"},
		},
		{
			ID:              "2",
			Name:            "HPE Smart Array P408i-a SR Gen10",
			DeviceType:      "StorageController",
			Location:        "Slot 0",
			Manufacturer:    "HPE",
			PartNumber:      "804331-B21",
			SerialNumber:    fmt.Sprintf("NIC%06dB", seed+1),
			FirmwareVersion: "5.61",
			Status:          status{Health: "OK", State: "Enabled"},
		},
	}
}

func buildPorts(seed int) []PortRecord {
	return []PortRecord{
		{PhysicalPortNumber: 1, Type: "Ethernet", MAC: fmt.Sprintf("94:18:82:%02x:%02x:01", (seed>>8)&0xff, seed&0xff)},
		{PhysicalPortNumber: 2, Type: "Ethernet", MAC: fmt.Sprintf("94:18:82:%02x:%02x:02", (seed>>8)&0xff, seed&0xff)},
	}
}

func buildServerFirmware(seed int, srvURI string) []FirmwareRecord {
	return []FirmwareRecord{
		{Name: "System ROM", Version: "U32 v2.76 (10/15/2024)", Location: "System Board", State: "Current", Status: "OK"},
		{Name: "Intelligent Provisioning", Version: "3.20.184", Location: "System Board", State: "Current", Status: "OK"},
		{Name: "Smart Array P408i-a SR Gen10", Version: "5.61", Location: "Slot 0", State: "Current", Status: "OK"},
	}
}
