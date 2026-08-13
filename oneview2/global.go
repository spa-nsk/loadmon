package oneview

import (
	"context"
	"errors"
	"runtime"
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
// version соответствует X-API-Version (0 - использовать текущую версию
// appliance, как определит клиент при логине).
func (infra *Infrastructure) AddEndpoint(endpoint string, domain string, login string, password string, version int) {
	infra.Endpoints = append(infra.Endpoints, Config{
		Endpoint:    endpoint,
		Username:    login,
		Password:    password,
		Domain:      domain,
		APIVersion:  version,
		InsecureTLS: true,
	})
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
// серийному номеру, URI и имени. Ошибки отдельных endpoint'ов не прерывают
// сбор с остальных (см. CollectServers) - они складываются в LastError, а
// уже собранные до сбоя серверы остаются в индексе.
//
// Таймаут на весь опрос - 2 минуты; при большом количестве точек подключения
// или медленной appliance его может понадобиться увеличить.
func (infra *Infrastructure) LoadServerHardwareList() ([]*Server, error) {
	infra.WaitUnLockGetData()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	servers, err := CollectServers(ctx, infra.Endpoints)

	infra.LockGetData()
	defer infra.UnLockGetData()

	list := make([]*Server, 0, len(servers))
	bySN := make(map[string]*Server, len(servers))
	byVSN := make(map[string]*Server, len(servers))
	byURI := make(map[string]*Server, len(servers))
	byName := cmap.New[*Server]()
	var warnings []string

	for i := range servers {
		s := &servers[i]
		list = append(list, s)
		if sn := strings.TrimSpace(s.Identity.SerialNumber); sn != "" {
			bySN[sn] = s
		}
		if uri := strings.TrimSpace(s.Identity.URI); uri != "" {
			byURI[uri] = s
		}
		if name := GetHWserverName(s); name != "" {
			byName.Set(name, s)
		}
		warnings = append(warnings, s.Warnings...)
	}

	infra.Servers = list
	infra.ServersCount = len(list)
	infra.ServerBySN = bySN
	infra.ServerByVirtualSN = byVSN
	infra.ServerByURI = byURI
	infra.ServerByName = byName
	infra.LastError = err
	infra.LastRefresh = time.Now()
	infra.LastWarnings = warnings

	return infra.Servers, err
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
	if srv.Identity.MpHostName != "" {
		return srv.Identity.MpHostName
	}
	if srv.Identity.ServerName != "" {
		return srv.Identity.ServerName + GetHWserverDomain(srv)
	}
	return GetHWserverName(srv)
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
