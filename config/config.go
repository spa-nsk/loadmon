package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"

	"loadmon/utilities"

	"loadmon/oneview"
	"loadmon/vspheremetrics"
)

type OvEndpoint struct {
	Domain   string
	Login    string
	Password string
	Endpoint string
	Version  int
}

type JsonConfig struct {
	AliasName      string
	DriverName     string
	UserName       string
	Password       string
	DBName         string
	DBFile         string // путь к файлу sqlite3 (DBFile в app2.json); относительный — от каталога conf
	DBFileAdd      string // путь ко второй sqlite3 (default-add); относительный — от conf
	HostName       string
	Port           int
	SSLmode        string
	ORMdebug       bool
	WorkMode       string
	AuthMode       string // режим авторизации пользователей веб-интерфейса: "domain" - Kerberos/SPNEGO (Active Directory), "local" - логин/пароль по локальной таблице models.AuthUser. Пусто или неизвестное значение трактуется как "domain" (поведение по умолчанию, сохраняет совместимость)
	HttpKey        string
	HttpsKey       string
	GRPCSrvDBAddr  string
	GRPCSrvDBPort  int64
	GRPCCliDBAddr  string
	GRPCCliDBPort  int64
	GRPCSrvApiAddr string
	GRPCSrvApiPort int64
	GRPCCliApiAddr string
	GRPCCliApiPort int64
	GRPCSwitchAddr string
	GRPCSwitchPort int64
	ConfigVCenters []vspheremetrics.VCenter
	ConfigOV       []OvEndpoint
}

type EndpointRecord struct {
	GRPCSrvApiAddr string
	GRPCSrvApiPort int64
	GRPCSrvDBAddr  string
	GRPCSrvDBPort  int64
	VcenterMap     cmap.ConcurrentMap[string, string]
	OneViewMap     cmap.ConcurrentMap[string, string]
}

type ConfigSwitch struct {
	Endpoint cmap.ConcurrentMap[string, *EndpointRecord]
}

var (
	ConfigData JsonConfig
	Switch     ConfigSwitch
)

func init() {
	Switch.Endpoint = cmap.New[*EndpointRecord]()
}

func (r *EndpointRecord) GetSrvApiAddr() string {
	s := fmt.Sprintf("%s:%d", r.GRPCSrvApiAddr, r.GRPCSrvApiPort)
	return s
}

func (r *EndpointRecord) GetSrvDBAddr() string {
	s := fmt.Sprintf("%s:%d", r.GRPCSrvDBAddr, r.GRPCSrvDBPort)
	return s
}

func confDir() string {
	return filepath.Dir(configFilePath())
}

func configFilePath() string {
	candidates := []string{
		filepath.Join(filepath.Dir(os.Args[0]), "conf", "app2.json"),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "conf", "app2.json"))
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return candidates[0]
}

func (s *JsonConfig) LoadConfigData() (JsonConfig, error) {
	if s == nil {
		s = &JsonConfig{}
	}
	fullPath := configFilePath()
	buff, err := os.ReadFile(fullPath)
	if err != nil {
		return *s, err
	}
	err = json.Unmarshal(buff, s)
	if err != nil {
		return *s, err
	}
	return *s, nil
}

func (s *JsonConfig) SaveConfigData() error {
	if s == nil {
		return fmt.Errorf("Нет данных для сохранения")
	}
	fullPath := configFilePath()
	buff, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = os.WriteFile(fullPath, buff, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (s *JsonConfig) InitParams() error {

	if s == nil {
		return fmt.Errorf("Нет данных для инициализации")
	}
	//username3 := "usr-nsk-dati-03"
	//password3 := "3Z7rn8Z1qxvaop2t@(XzmBEMSF)NRx"

	password := utilities.GetKeyData("usr-nsk-dati-02@global.bcs")

	//  password2 := utilities.GetKeyData("svc-vmw-vc-ro-01@am-uralsib.ru")

	if s.Password == "password" {
		s.Password = utilities.GetKeyData("work")
	}

	infra := vspheremetrics.GetGlobalVMInfrastructure() //получаем указатель на текущую инфраструктуру, если первый раз то инициализация

	infra.LockGetMetric() //блокируем запросы на обработку метрик

	defer infra.UnLockGetMetric() //после окончания изменения конфигурации разрешаем сбор метрик

	if infra.VCenterMapList.Count() > 0 {
		infra.SetDoneVcenter() //останавливаем опрос метрик
	}

	if infra.VCenterMapList.Count() > 0 { //если уже есть данные по vcenter
		infra.VCenterMapList.Clear() //обнуляем хэш таблицу
	}

	for _, vc := range s.ConfigVCenters {
		//		switch vc.Password {
		//		case "password":
		//			infra.AddVCenter(vc.Hostname, vc.Username, password, vc.Wait) //добавляем запись с указанием времени опроса и заменой пароля 1
		//		case "password2":
		//			infra.AddVCenter(vc.Hostname, vc.Username, password2, vc.Wait) //добавляем запись с указанием времени опроса и заменой пароля
		//		}
		infra.AddVCenter(vc.Hostname, vc.Username, utilities.GetKeyData(vc.Username), vc.Wait) //добавляем запись с указанием времени опроса и заменой пароля
	}
	ov := oneview.GlobalInitOVInfrastructure() //пересоздаем структуру по оборудованию
	for _, ovi := range s.ConfigOV {
		switch ovi.Password {
		case "password":
			ov.AddEndpoint(ovi.Endpoint, ovi.Domain, ovi.Login, password, ovi.Version)
		}
	}

	if infra.VCenterMapList.Count() > 0 {
		if s.WorkMode == "api" {
			go StartGetMetricThread() //запускаем опрос метрик
		}
	}
	return nil
}

func (s *JsonConfig) InitOVParams() {
	password := utilities.GetKeyData("usr-nsk-dati-02@global.bcs")

	ov := oneview.GlobalInitOVInfrastructure()
	for _, ovi := range s.ConfigOV {
		switch ovi.Password {
		case "password":
			ov.AddEndpoint(ovi.Endpoint, ovi.Domain, ovi.Login, password, ovi.Version)
		}
	}
}

func GetCliApiAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCCliApiAddr, ConfigData.GRPCCliApiPort)
	return s
}

func GetCliDBAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCCliDBAddr, ConfigData.GRPCCliDBPort)
	return s
}

func GetSrvApiAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCSrvApiAddr, ConfigData.GRPCSrvApiPort)
	return s
}

func GetSrvDBAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCSrvDBAddr, ConfigData.GRPCSrvDBPort)
	return s
}

func (s *JsonConfig) GetSwitchAddr() string {
	str := fmt.Sprintf("%s:%d", s.GRPCSwitchAddr, s.GRPCSwitchPort)
	return str
}

// IsLocalAuth возвращает true, если в конфигурации явно указан режим локальной авторизации
// (логин/пароль по локальной таблице models.AuthUser).
func (s *JsonConfig) IsLocalAuth() bool {
	return s.AuthMode == "local"
}

// IsDomainAuth возвращает true, если требуется доменная авторизация через Kerberos/SPNEGO.
// Пустое или неизвестное значение AuthMode трактуется как "domain" для обратной совместимости
// с конфигурациями, где параметр ещё не указан.
func (s *JsonConfig) IsDomainAuth() bool {
	return !s.IsLocalAuth()
}

func StartGetMetricThread() {
	infra := vspheremetrics.GetGlobalVMInfrastructure()
	infra.StartVcenterTimer()
	count := int32(infra.VCenterMapList.Count())
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		go func() {
			infra2 := vspheremetrics.GetGlobalVMInfrastructure()
			atomic.AddInt64(&infra2.CountThread, int64(1))
			defer atomic.AddInt64(&infra2.CountThread, int64(-1))
			log.Println("Запуск обработки запросов по: ", vc.Hostname, " с интервалом:", vc.Wait, " сек.")

			atomic.StoreInt64(&vc.CountThread, int64(1)) //заглушка

			ticker := time.NewTicker(time.Second * time.Duration(1))
			defer ticker.Stop()

			for {
				select {
				case <-vc.Ticker.C:
					infra2 = vspheremetrics.GetGlobalVMInfrastructure()
					if !infra2.IsLockGetMetric() {
						ctx, cancel := context.WithCancel(context.Background())
						defer cancel()
						infra2.UpdateListCluster(ctx, *vc)
						infra2.LoadHostMetricN(ctx, *vc)
						infra2.LoadVmMetricV4(ctx, *vc)
					}
				case <-ticker.C:
					//default:
					if atomic.CompareAndSwapInt32(&vc.DoneFl, 1, 0) {
						atomic.StoreInt64(&vc.CountThread, int64(0)) //заглушка
						atomic.AddInt32(&count, -1)
						log.Println("Остановка запросов по: ", vc.Hostname, "осталось", atomic.LoadInt32(&count), "активных обработчиков")
						return
					}
					//runtime.Gosched()
				}
			}
		}()
	}
}
