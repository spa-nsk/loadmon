// Package oneview содержит клиент для сбора информации об оборудовании HPE OneView:
// хранение точек подключения (endpoint'ов), потокобезопасный доступ к глобальной
// структуре данных (OVInfrastructure) и функции загрузки серверов, корзин, стоек
// и дата-центров через SDK oneview-golang либо через собственный Redfish-парсер
// (для точек подключения с endpoint.Version == 300, см. LoadServerHardwareList).
package oneview

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/HewlettPackard/oneview-golang/ov"
	"github.com/HewlettPackard/oneview-golang/utils"
	cmap "github.com/orcaman/concurrent-map/v2"
)

// LockGetData - блокировка доступа к глобальной структуре
func (info *OVInfrastructure) LockGetData() {
	info.WaitUnLockGetData()
	atomic.SwapInt32(&info.LockFlag, 1)
}

// UnLockGetData - снятие блокировки доступа к глобальной структуре
func (info *OVInfrastructure) UnLockGetData() {
	atomic.SwapInt32(&info.LockFlag, 0)
}

// WaitUnLockGetData - ожидание снятия блокировки
func (info *OVInfrastructure) WaitUnLockGetData() {
	for !atomic.CompareAndSwapInt32(&info.LockFlag, 0, 0) {
		runtime.Gosched()
	}
}

// IsLockGetData - проверка установлена ли блокировка
func (info *OVInfrastructure) IsLockGetData() bool {
	return atomic.CompareAndSwapInt32(&info.LockFlag, 1, 1)
}

// CheckLockGetData - получение текущего значения блокировки
func (info *OVInfrastructure) CheckLockGetData() int32 {
	return atomic.LoadInt32(&info.LockFlag)
}

// AddEndpoint - функция добавления точки подключения к списку подключений
func (infra *OVInfrastructure) AddEndpoint(endpoint string, domain string, login string, password string, ver int) {
	infra.Endpoints = append(infra.Endpoints, &ovEndpoint{Endpoint: endpoint, Domain: domain, Login: login, Password: password, Version: ver})
}

var infrastructureGlobal *OVInfrastructure

// Init - функция инициализации глобальной структуры
func (infra *OVInfrastructure) Init() {
	if infra == nil {
		infrastructureGlobal = &OVInfrastructure{}
		infra = infrastructureGlobal //проверить корректность обработки
	}
	infra.LockGetData()
	defer infra.UnLockGetData()

	infra.Servers = make([]*ServerHardware, 0)
	infra.ServersCount = 0
	infra.Endpoints = make([]*ovEndpoint, 0)
	infra.DataCenters = make(map[string]*DatacenterInfo)
	infra.AllRacks = make(map[string]*Racks)
	infra.ALLRacksDeviceTopology = make(map[string]*RacksDeviceTopology)
	infra.AllDeviceInfo = make(map[string]*DeviceInfo)
	infra.AllEnclosures = make(map[string]*Enclosure)
	infra.AllEnclosureBySN = make(map[string]*ov.Enclosure)
	infra.AllServerHardware = make(map[string]*ServerHardware)
	infra.AllDeviceBay = make(map[string]*DeviceBayInfo)
	infra.ServerBySN = make(map[string]*ServerHardware)
	infra.ServerByVirtualSN = make(map[string]*ServerHardware)
	infra.ServerByName = cmap.New[*ServerHardware]()
}

// Destroy - функция разрушения глобальной структуры
func (infra *OVInfrastructure) Destroy() {
	if infra == nil {
		infrastructureGlobal = &OVInfrastructure{}
		infra = infrastructureGlobal //проверить корректность обработки
	}
	infra.LockGetData()
	defer infra.UnLockGetData()
	infra.Servers = make([]*ServerHardware, 0)
	infra.ServersCount = 0
	infra.Endpoints = make([]*ovEndpoint, 0)
	infra.ServerByName.Clear()
}

// GlobalInitOVInfrastructure - инициализация пакета
func GlobalInitOVInfrastructure() *OVInfrastructure {
	infrastructureGlobal = &OVInfrastructure{}
	infrastructureGlobal.Init()

	return infrastructureGlobal
}

// GetGlobalOVInfrastructure - получение ссылки глобальную структуру
func GetGlobalOVInfrastructure() *OVInfrastructure {
	if infrastructureGlobal == nil {
		infrastructureGlobal = &OVInfrastructure{}
		infrastructureGlobal.Init()
	}
	return infrastructureGlobal
}

// DestroyGlobalOVInfrastructureObj - уничтожение глобальной структуры
func DestroyGlobalOVInfrastructureObj() {
	if infrastructureGlobal != nil {
		infrastructureGlobal.Destroy()
	}
}

// GetDeviceBayInfo - получение информации о слоте устройства по URI
func (infra *OVInfrastructure) GetDeviceBayInfo(uri string) (*DeviceBayInfo, bool) {
	if infra == nil {
		return nil, false
	}
	infra.WaitUnLockGetData()

	if dev, ok := infra.AllDeviceBay[uri]; ok {
		return dev, true
	}
	return nil, false
}

// GetEnclosure - получение корзины по слоту устройства
func (infra *OVInfrastructure) GetEnclosure(devbay *DeviceBayInfo) (*Enclosure, bool) {
	if infra == nil {
		return nil, false
	}
	if devbay == nil {
		return nil, false
	}
	infra.WaitUnLockGetData()

	tmp := strings.Split(devbay.URI, "/device-bays")
	if len(tmp) > 1 {
		if enc, ok := infra.AllEnclosures[tmp[0]]; ok {
			return enc, true
		}
	}
	return nil, false
}

// GetEnclosureByURI - получение корзины по URI
func (infra *OVInfrastructure) GetEnclosureByURI(uri string) (*Enclosure, bool) {
	if infra == nil {
		return nil, false
	}
	infra.WaitUnLockGetData()

	if enc, ok := infra.AllEnclosures[uri]; ok {
		return enc, true
	}
	return nil, false
}

// newOVClient создаёт клиента SDK oneview-golang (ov.OVClient) для указанной точки
// подключения. Вынесено в отдельную функцию, так как раньше идентичный код
// повторялся в LoadEnclosureList, LoadServerHardwareList и LoadServerProfileList.
func newOVClient(endpoint *ovEndpoint) *ov.OVClient {
	var c *ov.OVClient
	return c.NewOVClient(
		endpoint.Login,
		endpoint.Password,
		endpoint.Domain,
		endpoint.Endpoint,
		false,
		endpoint.Version,
		"*")
}

// LoadEnclosureList - загрузка информации по всем корзинам со всех точек подключения.
// Результат собирается в единую карту по серийному номеру корзины.
func (infra *OVInfrastructure) LoadEnclosureList() (map[string]*ov.Enclosure, error) {
	if infra == nil {
		return nil, errors.New("Указатель на infra = nil")
	}
	infra.WaitUnLockGetData()

	enclosureMap := make(map[string]*ov.Enclosure)
	for _, endpoint := range infra.Endpoints {
		ovc := newOVClient(endpoint)

		enclosures, err := GetServerEnclosures(ovc, "", "", "", "-1", "", "")
		if err != nil {
			return enclosureMap, err
		}
		for _, enc := range enclosures.Members {
			if _, ok := enclosureMap[enc.SerialNumber]; !ok {
				enclosureMap[enc.SerialNumber] = &enc
			}
		}
	}
	return enclosureMap, nil
}

// addServerHardwareRecordSDK сохраняет запись сервера, полученную через стандартный SDK
// oneview-golang (ov.ServerHardware), дозагружает по ней данные о памяти и локальном
// хранилище и регистрирует сервер во всех индексах инфраструктуры (по URI, серийному
// номеру, имени и виртуальному серийному номеру). Используется для точек подключения,
// не относящихся к специальному случаю endpoint.Version == 300.
func (infra *OVInfrastructure) addServerHardwareRecordSDK(ovc *ov.OVClient, rec ov.ServerHardware) {
	s := ServerHardware{}
	s.Base = rec
	s.Memory, _ = GetServerHardwareMemory(ovc, rec.UUID)        // запрос по памяти в сервере
	s.Storage, _ = GetServerHardwareLocalStorage(ovc, rec.UUID) // запрос по локальным хранилищам

	infra.Servers = append(infra.Servers, &s)
	infra.AllServerHardware[rec.URI.String()] = &s
	infra.ServerBySN[rec.SerialNumber.String()] = &s

	if serverName := GetHWserverName(&s); serverName != "" {
		infra.ServerByName.Set(serverName, &s)
	}

	if vsn := rec.VirtualSerialNumber.String(); vsn != "" && vsn != "null" {
		infra.ServerByVirtualSN[vsn] = &s
	} else {
		s.Base.VirtualSerialNumber = ""
	}

	infra.ServersCount++
}

// LoadServerHardwareList - загрузка информации о серверах со всех точек подключения.
// Для endpoint.Version == 300 используется собственный Redfish-парсер
// (GetServerHardwareListNew), так как этот appliance отдаёт данные в нестандартной
// для SDK oneview-golang схеме. Для остальных версий (в т.ч. 3800 и 7400) применяется
// штатный клиент SDK (ovc.GetServerHardwareList). Дополнительно по каждой точке
// подключения загружаются дата-центры, стойки и топология размещения оборудования.
func (infra *OVInfrastructure) LoadServerHardwareList() ([]*ServerHardware, error) {

	if infra == nil {
		return nil, errors.New("Указатель на infra = nil")
	}
	infra.WaitUnLockGetData()

	for _, endpoint := range infra.Endpoints {
		ovc := newOVClient(endpoint)

		if endpoint.Version == 300 {
			enclosures, err := GetServerEnclosures(ovc, "", "", "", "-1", "", "")
			if err == nil {
				for i, enc := range enclosures.Members {
					if _, ok := infra.AllEnclosureBySN[enc.SerialNumber]; !ok {
						infra.AllEnclosureBySN[enc.SerialNumber] = &enclosures.Members[i]
					}
				}
			}

			serverList := ServerHardwareListNew{}
			userQuery := ""
			query := ""
			groupUris := ""
			sort := ""
			start := ""
			total := ""

			// постранично собираем полный список серверов, пока не наберём serverList.Total записей
			for {
				var srvList ServerHardwareListNew
				srvList, err = GetServerHardwareListNew(ovc, userQuery, query, groupUris, sort, start, total)
				if err != nil {
					break
				}
				serverList.Members = append(serverList.Members, srvList.Members...)

				if serverList.Count == 0 {
					serverList.Total = srvList.Total
				}

				serverList.Count += srvList.Count
				if serverList.Count >= serverList.Total {
					break
				}
				start = strconv.Itoa(serverList.Count)
				total = strconv.Itoa(serverList.Total - serverList.Count)
			}

			if err == nil {
				for _, rec := range serverList.Members {
					s := ServerHardware{}

					s.Base.AssetTag = rec.AssetTag
					s.Base.Category = rec.Category
					s.Base.FormFactor = rec.FormFactor
					s.Base.HostOsType = rec.HostOsType

					if rec.OperatingSystem == "Unknown" {
						s.OperatingSystem = ""
					} else {
						s.OperatingSystem = rec.OperatingSystem
					}

					s.Base.Hostname = rec.ServerName
					s.Base.LocationURI = utils.Nstring(rec.LocationURI)
					s.Base.MemoryMb = rec.MemoryMb
					s.Base.Model = rec.Model
					s.Base.Name = rec.Name
					s.Base.PartNumber = rec.PartNumber
					s.Base.Position = rec.Position
					s.Base.PowerState = rec.PowerState
					s.Base.ProcessorCoreCount = rec.ProcessorCoreCount
					s.Base.ProcessorCount = rec.ProcessorCount
					s.Base.ProcessorSpeedMhz = rec.ProcessorSpeedMhz
					s.Base.ProcessorType = rec.ProcessorType
					s.Base.SerialNumber = utils.Nstring(rec.SerialNumber)
					s.Base.ServerHardwareTypeURI = utils.Nstring(rec.ServerHardwareTypeURI)
					s.Base.ServerGroupURI = utils.Nstring(rec.ServerGroupURI)
					s.Base.ServerName = rec.ServerName
					s.Base.ServerProfileURI = utils.Nstring(rec.ServerProfileURI)
					s.Base.ShortModel = rec.ShortModel
					s.Base.State = rec.State
					s.Base.Status = rec.Status
					s.Base.URI = utils.Nstring(rec.URI)
					s.Base.UUID = utils.Nstring(rec.UUID)

					if vn := utils.Nstring(rec.VirtualSerialNumber); vn != "" && vn != "null" {
						s.Base.VirtualSerialNumber = vn
					} else {
						s.Base.VirtualSerialNumber = ""
					}

					s.Base.VirtualUUID = rec.VirtualUUID

					s.Memory.Count = rec.SubResources.Memory.Count
					s.Memory.URI = rec.SubResources.Memory.URI
					s.Memory.Name = rec.SubResources.Memory.Name

					for _, mem := range rec.SubResources.Memory.Data {
						if mem.CapacityMiB == 0 {
							continue
						}
						module := MemoryModule{}
						module.BaseModuleType = mem.BaseModuleType
						module.CapacityMiB = mem.CapacityMiB
						module.DeviceLocator = mem.DeviceLocator
						module.Manufacturer = mem.VendorID
						module.MemoryDeviceType = mem.MemoryDeviceType
						module.MemoryLocation.Slot = mem.MemoryLocation.Slot
						module.MemoryLocation.Socket = mem.MemoryLocation.Socket
						module.Name = mem.Name
						module.Oem.Hpe.DIMMStatus = mem.Oem.Hpe.DIMMStatus
						module.OperatingSpeedMhz = mem.OperatingSpeedMhz
						module.RankCount = mem.RankCount
						module.Status = mem.Status
						s.Memory.Data = append(s.Memory.Data, module)
					}

					s.Storage.Count = rec.SubResources.LocalStorage.Count
					s.Storage.Name = rec.SubResources.LocalStorage.Name
					s.Storage.URI = rec.SubResources.LocalStorage.URI

					for _, st := range rec.SubResources.LocalStorage.Data {
						ls := LocalStorage{}
						ls.AdapterType = st.AdapterType
						ls.CacheMemorySizeMiB = st.CacheMemorySizeMiB
						ls.CacheModuleSerialNumber = st.SerialNumber
						ls.FirmwareVersion.Current = st.FirmwareVersion.Current
						ls.FirmwareVersion.Current.VersionString = st.FirmwareVersion.Current.VersionString
						ls.Location = st.Location
						ls.Model = st.Model
						ls.Name = st.Name
						s.Storage.Data = append(s.Storage.Data, ls)
					}

					infra.Servers = append(infra.Servers, &s)
					infra.AllServerHardware[rec.URI] = &s
					infra.ServerBySN[rec.SerialNumber] = &s
					if serverName := GetHWserverName(&s); serverName != "" {
						infra.ServerByName.Set(serverName, &s)
					}
					if vsn := rec.VirtualSerialNumber; vsn != "" && vsn != "null" {
						infra.ServerByVirtualSN[vsn] = &s
					} else {
						s.Base.VirtualSerialNumber = ""
					}
					infra.ServersCount++
				}
			}
			continue
		}

		filters := []string{""}
		sort := ""
		start := ""
		count := ""
		expand := "all"

		ServerList, err := ovc.GetServerHardwareList(filters, sort, start, count, expand)
		if err == nil {
			for _, rec := range ServerList.Members {
				infra.addServerHardwareRecordSDK(ovc, rec)
			}
		} else {
			fmt.Println(endpoint.Endpoint, "Failed to fetch server List : ", err)
		}

		// SDK возвращает данные постранично (ServerList.Count за раз) - дозагружаем остаток
		for i := ServerList.Count; i < ServerList.Total; i = i + ServerList.Count {
			if ServerList.Total-i > ServerList.Count {
				count = strconv.Itoa(ServerList.Count)
			} else {
				count = strconv.Itoa(ServerList.Total - i)
			}
			start = strconv.Itoa(i)
			ServerList, err = ovc.GetServerHardwareList(filters, sort, start, count, expand)
			if err == nil {
				for _, rec := range ServerList.Members {
					infra.addServerHardwareRecordSDK(ovc, rec)
				}
			} else {
				fmt.Println("Failed to fetch server List : ", err)
			}
		}

		// загрузка дата-центров, стоек и топологии размещения оборудования (стойка -> корзина/сервер -> слот)
		infra.AllDatacenters, _ = LoadDatcenterList(ovc, []string{""}, "", "", "")
		for j, dc := range infra.AllDatacenters.Members {
			infra.DataCenters[dc.URI] = &infra.AllDatacenters.Members[j]

			for _, rack := range dc.Contents {
				rackInfo, _ := GetRacksByUri(ovc, rack.ResourceURI)
				infra.AllRacks[rack.ResourceURI] = &rackInfo

				deviceTopology, _ := GetRacksDeviceTopology(ovc, rack.ResourceURI)
				infra.ALLRacksDeviceTopology[rack.ResourceURI] = &deviceTopology

				for k, device := range deviceTopology.Children {
					infra.AllDeviceInfo[device.Device.URI] = &deviceTopology.Children[k].Device

					switch device.Device.Category {
					case "enclosures":
						encHardware, _ := GetServerEnclosure(ovc, device.Device.URI)
						infra.AllEnclosures[device.Device.URI] = &encHardware

						for i, bay := range encHardware.DeviceBays {
							if bay.DevicePresence != "Present" || bay.DeviceURI == "" {
								continue
							}
							if _, ok := infra.GetDeviceBayInfo(bay.DeviceURI); !ok {
								infra.AllDeviceBay[bay.DeviceURI] = &encHardware.DeviceBays[i]
							}
							bay2, _ := infra.GetDeviceBayInfo(bay.DeviceURI)

							srvHardware, _ := ovc.GetServerHardwareByUri(utils.NewNstring(bay.DeviceURI))
							if srvLink, ok := infra.AllServerHardware[srvHardware.URI.String()]; ok {
								srvLink.Datacenter = dc
								srvLink.Rack = rackInfo
								srvLink.EnclosureInfo = encHardware
								if bay2 != nil {
									srvLink.BayInfo = *bay2
								}
							}
						}
					case "server-hardware":
						srvHardware, _ := ovc.GetServerHardwareByUri(utils.NewNstring(device.Device.URI))
						if srvLink, ok := infra.AllServerHardware[srvHardware.URI.String()]; ok {
							srvLink.Datacenter = dc
							srvLink.Rack = rackInfo
						}
						// отбрасываем служебный префикс "ILO"/"iLO"/"ilo" (с дефисом или без) из имени устройства
						for _, prefix := range []string{"ILO-", "ILO", "ilo-", "ilo", "iLO-", "iLO"} {
							if !strings.Contains(device.Device.Name, prefix) {
								continue
							}
							if parts := strings.Split(device.Device.Name, prefix); len(parts) == 2 {
								device.Device.Name = parts[1]
							}
							break
						}
					}
				}
			}
		}
	}
	return infra.Servers, nil
}

// LoadServerProfileList - загрузка списка серверных профилей (server-profiles) со всех
// точек подключения через собственный Redfish-парсер GetServerProfileListNew.
func (infra *OVInfrastructure) LoadServerProfileList() (interface{}, error) {
	if infra == nil {
		return nil, errors.New("Указатель на infra = nil")
	}
	data := make([]interface{}, 0)
	for _, endpoint := range infra.Endpoints {
		ovc := newOVClient(endpoint)
		d, err := GetServerProfileListNew(ovc, "", "", "", "", "", "")
		data = append(data, d)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}

// FindServerHardwareSN - поиск сервера со всех точек подключения по серийному номеру
func (infra *OVInfrastructure) FindServerHardwareSN(sn string) (*ServerHardware, error) {
	if infra == nil {
		return nil, errors.New("Указатель на infra = nil")
	}
	infra.WaitUnLockGetData()

	srvHW, ok := infra.ServerBySN[sn]
	if ok {
		return srvHW, nil
	}
	return nil, errors.New("Server by Serial Number: " + sn + " not found")
}

// GetHWserverName возвращает "человекочитаемое" имя сервера. Если поле Name уже
// совпадает с серийным номером, используется ServerName. Иначе из Name вырезается
// служебный префикс вида "ILO-", "ILO", "ilo-", "ilo", "iLO-" или "iLO"; если ни один
// из префиксов не встретился (или встретился более одного раза), возвращается ServerName.
func GetHWserverName(srv *ServerHardware) string {
	if strings.Contains(srv.Base.Name, srv.Base.SerialNumber.String()) {
		return srv.Base.ServerName
	}
	for _, prefix := range []string{"ILO-", "ILO", "ilo-", "ilo", "iLO-", "iLO"} {
		if !strings.Contains(srv.Base.Name, prefix) {
			continue
		}
		if parts := strings.Split(srv.Base.Name, prefix); len(parts) == 2 {
			return parts[1]
		}
		break
	}
	return srv.Base.ServerName
}

// GetHWserverDomain возвращает доменную часть, вырезанную из Name после исключения
// из неё серийного номера (либо, если серийный номер не найден, - имени сервера).
func GetHWserverDomain(srv *ServerHardware) string {
	if srv == nil {
		return ""
	}
	s := strings.Split(srv.Base.Name, srv.Base.SerialNumber.String())
	if len(s) > 1 {
		return s[len(s)-1]
	}
	s = strings.Split(srv.Base.Name, srv.Base.ServerName)
	if len(s) > 1 {
		return s[len(s)-1]
	}
	return ""
}

// GetHWserverNameFQDN возвращает полное доменное имя сервера: Hostname, если он
// заполнен; иначе ServerName + домен (см. GetHWserverDomain); а если и ServerName
// не заполнен - результат GetHWserverName.
func GetHWserverNameFQDN(srv *ServerHardware) string {
	if srv.Base.Hostname != "" {
		return srv.Base.Hostname
	}
	if srv.Base.ServerName != "" {
		return fmt.Sprintf("%s%s", srv.Base.ServerName, GetHWserverDomain(srv))
	}
	return GetHWserverName(srv)
}
