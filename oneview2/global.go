package oneview

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
)

// Infrastructure is the in-memory index of all HPE OneView servers collected
// from every configured endpoint (via CollectServers). It plays the same role
// the old package `loadmon/oneview`'s OVInfrastructure played, but the
// records it holds are oneview2.Server values instead of the old
// oneview.ServerHardware values - callers reading individual server fields
// must use the new Server/ServerIdentity/DatacenterInfo/RackInfo/
// EnclosureInfo/BayInfo shape (see collect.go, location.go, inventory.go).
//
// Concurrency: as in the old package, a simple spin-lock (LockFlag) guards
// rebuilds of the index so readers never see a half-populated map; this
// mirrors OVInfrastructure.LockGetData/UnLockGetData/WaitUnLockGetData
// on purpose, to keep behavior identical for existing callers.
type Infrastructure struct {
	LockFlag int32

	Endpoints []Config

	// CollectTimeout — общий таймаут на групповой сбор инвентаризации (LoadServerHardwareList).
	CollectTimeout time.Duration

	Servers      []*Server
	ServersCount int

	// ServerBySN/ServerByVirtualSN/ServerByURI - обычные map, доступ только
	// под LockGetData/WaitUnLockGetData (как и в старом пакете).
	ServerBySN        map[string]*Server
	ServerByVirtualSN map[string]*Server
	ServerByURI       map[string]*Server
	// ServerByName - потокобезопасная карта (как и в старом пакете), можно
	// читать без явной блокировки.
	ServerByName cmap.ConcurrentMap[string, *Server]

	// DatacenterNames/RackNames - множество различных значений
	// Server.Datacenter.Name/Server.Rack.Name, накопленное по ВСЕМ
	// собранным серверам (а не только по подмножеству, вернувшемуся из
	// какого-то одного фильтрованного запроса). Нужно для того, чтобы
	// выпадающие списки фильтра "Датацентр"/"Шкаф" на /sprsrv/hwviewlist
	// показывали полный перечень значений независимо от того, что сейчас
	// выбрано в остальных полях фильтра. Доступ - только под
	// LockGetData/WaitUnLockGetData, как и остальные обычные map этой
	// структуры.
	DatacenterNames map[string]struct{}
	RackNames       map[string]struct{}

	// EnclosureByName/EnclosureBySN - агрегированные по блейд-корзинам
	// данные, построенные из имени ServerHardware ("c7000-9, bay 15", см.
	// ParseBladeNameBay) - БЕЗ дополнительного REST-вызова GetEnclosure на
	// каждый сервер. EnclosureInfo (заполняется именно через GetEnclosure)
	// для части серверов не заполняется вовсе (см. комментарий у
	// EnclosureSummary) - эти карты дают надёжный путь к линковке
	// сервер->корзина независимо от того, отработал ли GetEnclosure.
	//
	// EnclosureByName - ключ: нормализованное имя корзины (см.
	// NormalizeEnclosureName, формат "PREFIX-NN"). EnclosureBySN - ключ:
	// серийный номер корзины, ЕСЛИ он известен хотя бы по одному серверу
	// этой корзины (из Server.EnclosureInfo.SerialNumber, когда
	// GetEnclosure для НЕГО всё же отработал) - может быть неполной, если
	// ни один сервер корзины не получил EnclosureInfo. Доступ - только под
	// LockGetData/WaitUnLockGetData.
	EnclosureByName map[string]*EnclosureSummary
	EnclosureBySN   map[string]*EnclosureSummary

	LastError    error
	LastRefresh  time.Time
	LastWarnings []string
}

func (infra *Infrastructure) LockGetData() {
	infra.WaitUnLockGetData()
	atomic.SwapInt32(&infra.LockFlag, 1)
}

func (infra *Infrastructure) UnLockGetData() {
	atomic.SwapInt32(&infra.LockFlag, 0)
}

func (infra *Infrastructure) WaitUnLockGetData() {
	for !atomic.CompareAndSwapInt32(&infra.LockFlag, 0, 0) {
		runtime.Gosched()
	}
}

func (infra *Infrastructure) IsLockGetData() bool {
	return atomic.CompareAndSwapInt32(&infra.LockFlag, 1, 1)
}

func (infra *Infrastructure) CheckLockGetData() int32 {
	return atomic.LoadInt32(&infra.LockFlag)
}

// AddEndpoint регистрирует точку подключения HPE OneView/Global Dashboard.
// operationTimeout — таймаут одной HTTP-операции к appliance (0 — 60 с по умолчанию).
func (infra *Infrastructure) AddEndpoint(endpoint string, domain string, login string, password string, version int, operationTimeout time.Duration) {
	if operationTimeout <= 0 {
		operationTimeout = 60 * time.Second
	}
	infra.Endpoints = append(infra.Endpoints, Config{
		Endpoint:    endpoint,
		Username:    login,
		Password:    password,
		Domain:      domain,
		APIVersion:  version,
		InsecureTLS: true,
		Timeout:     operationTimeout,
	})
}

// SetCollectTimeout задаёт общий таймаут на групповой сбор инвентаризации.
func (infra *Infrastructure) SetCollectTimeout(d time.Duration) {
	infra.CollectTimeout = d
}

func (infra *Infrastructure) Init() {
	infra.LockGetData()
	defer infra.UnLockGetData()

	infra.Endpoints = make([]Config, 0)
	infra.Servers = make([]*Server, 0)
	infra.ServersCount = 0
	infra.ServerBySN = make(map[string]*Server)
	infra.ServerByVirtualSN = make(map[string]*Server)
	infra.ServerByURI = make(map[string]*Server)
	infra.ServerByName = cmap.New[*Server]()
	infra.DatacenterNames = make(map[string]struct{})
	infra.RackNames = make(map[string]struct{})
	infra.EnclosureByName = make(map[string]*EnclosureSummary)
	infra.EnclosureBySN = make(map[string]*EnclosureSummary)
}

func (infra *Infrastructure) Destroy() {
	infra.LockGetData()
	defer infra.UnLockGetData()
	infra.Endpoints = make([]Config, 0)
	infra.Servers = make([]*Server, 0)
	infra.ServersCount = 0
	infra.ServerBySN = make(map[string]*Server)
	infra.ServerByVirtualSN = make(map[string]*Server)
	infra.ServerByURI = make(map[string]*Server)
	infra.ServerByName.Clear()
	infra.DatacenterNames = make(map[string]struct{})
	infra.RackNames = make(map[string]struct{})
	infra.EnclosureByName = make(map[string]*EnclosureSummary)
	infra.EnclosureBySN = make(map[string]*EnclosureSummary)
}

var infrastructureGlobal *Infrastructure

// GlobalInitOVInfrastructure пересоздаёт глобальную структуру (используется
// при перезагрузке конфигурации - config.JsonConfig.InitParams).
func GlobalInitOVInfrastructure() *Infrastructure {
	infrastructureGlobal = &Infrastructure{}
	infrastructureGlobal.Init()
	return infrastructureGlobal
}

// GetGlobalOVInfrastructure возвращает ссылку на глобальную структуру,
// создавая её при первом обращении.
func GetGlobalOVInfrastructure() *Infrastructure {
	if infrastructureGlobal == nil {
		infrastructureGlobal = &Infrastructure{}
		infrastructureGlobal.Init()
	}
	return infrastructureGlobal
}

// DestroyGlobalOVInfrastructureObj уничтожает глобальную структуру.
func DestroyGlobalOVInfrastructureObj() {
	if infrastructureGlobal != nil {
		infrastructureGlobal.Destroy()
	}
}

// LoadServerHardwareList опрашивает все зарегистрированные точки подключения
// (CollectServers) и перестраивает индексы по серийному номеру, виртуальному
// серийному номеру, URI и имени.
func (infra *Infrastructure) LoadServerHardwareList() ([]*Server, error) {
	infra.WaitUnLockGetData()

	collectTimeout := infra.CollectTimeout
	if collectTimeout <= 0 {
		collectTimeout = 120 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), collectTimeout)
	defer cancel()
	log.Println("количество конфигураций: ", len(infra.Endpoints), infra.Endpoints)
	servers, err := CollectServers(ctx, infra.Endpoints)

	infra.LockGetData()
	defer infra.UnLockGetData()

	list := make([]*Server, 0, len(servers))
	bySN := make(map[string]*Server, len(servers))
	byVSN := make(map[string]*Server, len(servers))
	byURI := make(map[string]*Server, len(servers))
	byName := cmap.New[*Server]()
	var warnings []string

	// DatacenterNames/RackNames накапливаются, а не перестраиваются заново -
	// если очередной цикл сбора инвентаризации (LoadServerHardwareList
	// запускается по тикеру раз в 6 часов) по какой-то причине не увидел
	// сервер из ранее известного датацентра/шкафа (временная недоступность
	// appliance, фильтр в конфигурации endpoint'ов и т.п.), значение не
	// пропадает из списка фильтра - оно всё ещё актуально для CMDB.
	if infra.DatacenterNames == nil {
		infra.DatacenterNames = make(map[string]struct{})
	}
	if infra.RackNames == nil {
		infra.RackNames = make(map[string]struct{})
	}

	// enclosuresByName - см. Infrastructure.EnclosureByName/EnclosureBySN и
	// ParseBladeNameBay: строится из Server.Identity.Name блейд-серверов
	// ("c7000-9, bay 15"), без дополнительных запросов к appliance.
	enclosuresByName := make(map[string]*EnclosureSummary)

	for i := range servers {
		s := &servers[i]
		list = append(list, s)
		if sn := strings.TrimSpace(s.Identity.SerialNumber); sn != "" {
			bySN[sn] = s
		}
		if vsn := strings.TrimSpace(s.Identity.VirtualSerialNumber); vsn != "" {
			byVSN[vsn] = s
		}
		if uri := strings.TrimSpace(s.Identity.URI); uri != "" {
			byURI[uri] = s
		}
		if name := GetHWserverName(s); name != "" {
			byName.Set(name, s)
		}
		if dc := strings.TrimSpace(s.Datacenter.Name); dc != "" {
			infra.DatacenterNames[dc] = struct{}{}
		}
		if rk := strings.TrimSpace(s.Rack.Name); rk != "" {
			infra.RackNames[rk] = struct{}{}
		}
		if chassis, bay, ok := ParseBladeNameBay(s.Identity.Name); ok {
			es, exists := enclosuresByName[chassis]
			if !exists {
				es = &EnclosureSummary{Name: chassis, ByBay: make(map[int]*Server)}
				enclosuresByName[chassis] = es
			}
			es.ByBay[bay] = s
			// Дозаполняем лучшими известными данными из EnclosureInfo -
			// оно доступно только если GetEnclosure отработал для ЭТОГО
			// конкретного сервера (см. комментарий у EnclosureSummary) -
			// первого непустого значения среди серверов корзины достаточно.
			if es.SerialNumber == "" && s.EnclosureInfo.SerialNumber != "" {
				es.SerialNumber = s.EnclosureInfo.SerialNumber
			}
			if es.Model == "" && s.EnclosureInfo.Model != "" {
				es.Model = s.EnclosureInfo.Model
			}
			if es.EnclosureType == "" && s.EnclosureInfo.EnclosureType != "" {
				es.EnclosureType = s.EnclosureInfo.EnclosureType
			}
		}
		warnings = append(warnings, s.Warnings...)
	}

	enclosuresBySN := make(map[string]*EnclosureSummary, len(enclosuresByName))
	for _, es := range enclosuresByName {
		if es.SerialNumber != "" {
			enclosuresBySN[es.SerialNumber] = es
		}
	}

	infra.Servers = list
	infra.ServersCount = len(list)
	infra.ServerBySN = bySN
	infra.ServerByVirtualSN = byVSN
	infra.ServerByURI = byURI
	infra.ServerByName = byName
	infra.EnclosureByName = enclosuresByName
	infra.EnclosureBySN = enclosuresBySN
	infra.LastError = err
	infra.LastRefresh = time.Now()
	infra.LastWarnings = warnings

	return infra.Servers, err
}

// DatacenterNameList возвращает отсортированный список всех различных
// значений Datacenter.Name, накопленных по всем циклам сбора инвентаризации
// (см. комментарий в LoadServerHardwareList) - в отличие от перебора
// текущего отфильтрованного среза серверов, этот список не зависит от
// значений остальных полей фильтра.
func (infra *Infrastructure) DatacenterNameList() []string {
	infra.WaitUnLockGetData()
	out := make([]string, 0, len(infra.DatacenterNames))
	for name := range infra.DatacenterNames {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// RackNameList возвращает отсортированный список всех различных значений
// Rack.Name, накопленных по всем циклам сбора инвентаризации - см.
// DatacenterNameList.
func (infra *Infrastructure) RackNameList() []string {
	infra.WaitUnLockGetData()
	out := make([]string, 0, len(infra.RackNames))
	for name := range infra.RackNames {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// FindServerHardwareSN ищет сервер по серийному номеру. Сигнатура (значение,
// error) сохранена как в старом пакете loadmon/oneview, чтобы не менять
// вызывающий код по всему cmdb (там повсеместно используется идиома
// `hw, err := ...; if err == nil { ... }`).
func (infra *Infrastructure) FindServerHardwareSN(sn string) (*Server, error) {
	infra.WaitUnLockGetData()
	s, ok := infra.ServerBySN[sn]
	if !ok {
		return nil, errors.New("Server by Serial Number: " + sn + " not found")
	}
	return s, nil
}

// GetHWserverName возвращает "человекочитаемое" имя сервера: если поле Name
// совпадает с серийным номером, используется ServerName; иначе из Name
// вырезается служебный префикс "ILO"/"iLO" (с дефисом или без); если ни один
// из вариантов не подошёл, возвращается ServerName как есть.
func GetHWserverName(srv *Server) string {
	if srv == nil {
		return ""
	}
	if srv.Identity.SerialNumber != "" && strings.Contains(srv.Identity.Name, srv.Identity.SerialNumber) {
		return srv.Identity.ServerName
	}
	for _, prefix := range []string{"ILO-", "ILO", "ilo-", "ilo", "iLO-", "iLO"} {
		if !strings.Contains(srv.Identity.Name, prefix) {
			continue
		}
		if parts := strings.Split(srv.Identity.Name, prefix); len(parts) == 2 {
			return parts[1]
		}
		break
	}
	return srv.Identity.ServerName
}

// GetHWserverDomain возвращает доменную часть, вырезанную из Name после
// исключения из неё серийного номера (либо, если серийный номер не найден в
// Name, - имени сервера).
func GetHWserverDomain(srv *Server) string {
	if srv == nil {
		return ""
	}
	if srv.Identity.SerialNumber != "" {
		if parts := strings.Split(srv.Identity.Name, srv.Identity.SerialNumber); len(parts) > 1 {
			return parts[len(parts)-1]
		}
	}
	if srv.Identity.ServerName != "" {
		if parts := strings.Split(srv.Identity.Name, srv.Identity.ServerName); len(parts) > 1 {
			return parts[len(parts)-1]
		}
	}
	return ""
}

// GetHWserverNameFQDN возвращает полное доменное имя сервера: MpHostName,
// если он заполнен (аналог старого Base.Hostname - имя контроллера
// управления, iLO/OA); иначе ServerName + домен (см. GetHWserverDomain); а
// если и ServerName не заполнен - результат GetHWserverName.
func GetHWserverNameFQDN(srv *Server) string {
	if srv == nil {
		return ""
	}
	if srv.Identity.SerialNumber != "" && strings.Contains(srv.Identity.MpHostName, srv.Identity.SerialNumber) {
		return srv.Identity.ServerName
	}
	for _, prefix := range []string{"ILO-", "ILO", "ilo-", "ilo", "iLO-", "iLO"} {
		if !strings.Contains(srv.Identity.MpHostName, prefix) {
			continue
		}
		if parts := strings.Split(srv.Identity.MpHostName, prefix); len(parts) == 2 {
			return parts[1]
		}
		break
	}
	return srv.Identity.ServerName

	/*
		if srv.Identity.MpHostName != "" {
			return srv.Identity.MpHostName
		}
		if srv.Identity.ServerName != "" {
			return srv.Identity.ServerName + GetHWserverDomain(srv)
		}
		return GetHWserverName(srv)
	*/
}

// TotalMemoryMb суммирует ёмкость всех установленных модулей памяти сервера
// в мегабайтах (в oneview2.Server, в отличие от старого Base.MemoryMb, нет
// готового поля с итоговым объёмом - его нужно считать по Memory[]).
func TotalMemoryMb(srv *Server) int {
	if srv == nil {
		return 0
	}
	total := 0
	for _, m := range srv.Memory {
		total += m.CapacityMiB
	}
	return total
}
