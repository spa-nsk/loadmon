// Package storages объединяет получение данных с различных СХД EMC (Unity через REST API пакета emcstorage
// и VNX через утилиту naviseccli, см. vnxcli.go) и агрегирует их в единую структуру Storage для дальнейшей обработки.
package storages

import (
	"loadmon/emcstorage"
	datamap "loadmon/utilities"
	"log"
)

// Connect структура с данными для подключения к СХД
type Connect struct {
	session  emcstorage.Session
	Point    string
	Username string
	Password string
}

// GetPoint получить имя подключения
func (c *Connect) GetPoint() string {
	return c.Point
}

// Storage структура содержащая данные по всем СХД
type Storage struct {
	Connections      []Connect
	System           map[string]*emcstorage.SystemContent           //информация для подключения к СХД
	DPE              map[string]*emcstorage.DPEContent              //основная информация по СХД
	DAE              map[string]*emcstorage.DAEContent              //
	StorageProcessor map[string]*emcstorage.StorageProcessorContent //
	Disk             map[string]*emcstorage.Disk                    //
	DiskGroup        map[string]*emcstorage.DiskGroupContent        //
	Host             map[string]*emcstorage.HostContent             //

	Event map[string]*emcstorage.EventContent //
}

// Types
type pool struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SizeFree       uint64 `json:"sizeFree"`
	SizeTotal      uint64 `json:"sizeTotal"`
	SizeUsed       uint64 `json:"sizeUsed"`
	SizeSubscribed uint64 `json:"sizeSubscribed"`
}

type storageresource struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SizeAllocated uint64 `json:"sizeAllocated"`
	SizeTotal     uint64 `json:"sizeTotal"`
	SizeUsed      uint64 `json:"sizeUsed"`
	Type          int    `json:"type"`
}

type alert struct {
	ID          string `json:"id"`
	Severity    int    `json:"severity"`
	Description string `json:"description"`
}

type disk struct {
	ID               string `json:"id"`
	NeedsReplacement bool   `json:"needsReplacement"`
	SlotNumber       uint32 `json:"slotNumber"`
	EstimatedEOL     string `json:"estimatedEOL"`
	BusId            uint32 `json:"busId"`
	Name             string `json:"name"`
	Manufacturer     string `json:"manufacturer"`
	Model            string `json:"model"`
	Version          string `json:"version"`
	EmcPartNumber    string `json:"emcPartNumber"`
	EmcSerialNumber  string `json:"emcSerialNumber"`
	TierType         uint16 `json:"tierType"`
	CurrentSpeed     uint64 `json:"currentSpeed"` //bytes/sec
	MaxSpeed         uint64 `json:"maxSpeed"`
	IsInUse          bool   `json:"isInUse"`
	IsFastCacheInUse bool   `json:"isFastCacheInUse"`
	Size             uint64 `json:"size"` //bytes
	RawSize          uint64 `json:"rawSize"`
	VendorSize       uint64 `json:"vendorSize"`
	WWN              string `json:"wwn"`
	DiskTechnology   int    `json:"diskTechnology"`
}

var login = "spaview"

// GetLunsRest получает список LUN СХД Unity по адресу adress через REST API (emcstorage)
func GetLunsRest(adress string) ([]*emcstorage.LUNContent, error) {
	var lunList []*emcstorage.LUNContent

	pass := datamap.GetKeyData(login)

	session, err := emcstorage.NewSession(adress, true, login, pass)

	if err != nil {
		return nil, err
	}
	defer session.CloseSession()
	lun, err := session.GetLun()
	if err != nil {
		return nil, err
	} else {
		for _, s := range lun.Entries {
			var rec = new(emcstorage.LUNContent)
			*rec = s.Content
			lunList = append(lunList, rec)
		}
	}
	return lunList, nil
}

// GetHostsRest получает список хостов СХД Unity по адресу adress через REST API (emcstorage)
func GetHostsRest(adress string) ([]*emcstorage.HostContent, error) {
	var hostList []*emcstorage.HostContent

	pass := datamap.GetKeyData(login)

	session, err := emcstorage.NewSession(adress, true, login, pass)

	if err != nil {
		session, err = emcstorage.NewSession(adress, false, login, pass)
		if err != nil {
			return nil, err
		}
	}
	defer session.CloseSession()
	host, err := session.GetHost()
	if err != nil {
		return nil, err
	} else {
		for _, s := range host.Entries {
			var rec = new(emcstorage.HostContent)
			*rec = s.Content
			hostList = append(hostList, rec)
		}
	}
	return hostList, nil
}

// -unity 192.168.26.34 -user semenovpa -password Password123# -histkpipaths kpi.sp.*.utilization
// InitData
func InitData() {

//	var unityName string
//	var unityPools []pool
//	var unityStorageResource []storageresource
//	var alertMessaage []alert
//	var disks []disk
//	var diskGroup []types.DiskGroupContent
//	var sps []types.StorageProcessorContent
//	var daeList []types.DAEContent
//	var dpeList []types.DPEContent
//	var hostList []types.HostContent
//	var eventList []types.EventContent
	var lunList []emcstorage.LUNContent
	pass := datamap.GetKeyData(login)

	session, err := emcstorage.NewSession("10.2.19.111", true, login, pass)

	if err != nil {
		log.Fatal(err)
	}

	defer session.CloseSession()
	/*
		// Get system informations
		System, err := session.GetbasicSystemInfo()
		if err != nil {
			log.Fatal(err)
		} else {
			// Store the name of the Unity
			unityName = System.Entries[0].Content.Name
			base := System.Base
			links := System.Links
			cont := System.Entries[0].Content
			_ = cont
			_ = links
			_ = base
		}
		_ = unityName
		// Store pools informations
		Pools, err := session.GetPool()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, p := range Pools.Entries {
				unityPools = append(unityPools, p.Content)
			}
		}

		// Store storage resources informations
		StorageResources, err := session.GetStorageResource()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range StorageResources.Entries {
				unityStorageResource = append(unityStorageResource, s.Content)
			}
		}
		Alert, err := session.GetAlert()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range Alert.Entries {
				alertMessaage = append(alertMessaage, s.Content)
			}
		}

		event, err := session.GetEventIDs()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range event.Entries {
				eventList = append(eventList, s.Content)
			}
		}

		Disks, err := session.GetDisks()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range Disks.Entries {
				disks = append(disks, s.Content)
			}
		}

		diskgroup, err := session.GetDiskGroup()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range diskgroup.Entries {
				diskGroup = append(diskGroup, s.Content)
			}
		}

		sp, err := session.GetStorageProcessor()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range sp.Entries {
				sps = append(sps, s.Content)
			}
		}

		dae, err := session.GetDAE()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range dae.Entries {
				daeList = append(daeList, s.Content)
			}
		}

		dpe, err := session.GetDPE()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range dpe.Entries {
				dpeList = append(dpeList, s.Content)
			}
		}

		host, err := session.GetHost()
		if err != nil {
			log.Fatal(err)
		} else {
			for _, s := range host.Entries {
				hostList = append(hostList, s.Content)
			}
		}
	}
*/
	lun, err := session.GetLun()
	if err != nil {
		log.Fatal(err)
	} else {
		for _, s := range lun.Entries {
			lunList = append(lunList, s.Content)
		}
	}
}
