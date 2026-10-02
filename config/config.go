// Package config отвечает за загрузку и хранение конфигурации приложения
// loadmon (файл app2.json), инициализацию подключений к базам данных ORM
// beego (см. DBInit) и запуск фонового опроса метрик vCenter/OneView по
// параметрам, заданным в конфигурации.
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

	"loadmon/vspheremetrics"

	oneview "github.com/spa-nsk/oneview2"
)

// OvEndpoint описывает подключение к одному эндпоинту HPE OneView (домен, учётные данные, версия API).
type OvEndpoint struct {
	Domain              string
	Login               string
	Password            string
	Endpoint            string
	Version             int
	OperationTimeoutSec int // таймаут одной HTTP-операции к appliance; 0 — OneViewOperationTimeoutSec
}

// ScomEndpoint описывает подключение к одной management group Microsoft
// System Center Operations Manager (SCOM) через официальный REST API
// (SCOM 2019 UR1+, базовый путь /OperationsManager, см.
// kontext/13-scom-metrics.md).
//
// Используется сборщиком scommetrics (metrics_grpc запускает по горутине
// на каждую запись с Enabled=true). Пустой список ConfigSCOM или
// Enabled=false у всех записей - штатное состояние, сбор не идёт.
type ScomEndpoint struct {
	// Name - короткая метка источника (имя management group), попадает в
	// логи сборщика; должна быть уникальной в пределах ConfigSCOM.
	Name string
	// Enabled - разрешает опрос этой записи. По умолчанию false, чтобы
	// появление записи в app2.json само по себе не запускало сбор.
	Enabled bool
	// Endpoint - базовый URL REST API management-сервера, БЕЗ завершающего
	// "/", например "https://scom-ms01.example.local/OperationsManager".
	Endpoint string
	// AuthMode - режим аутентификации POST /OperationsManager/authenticate
	// (тело - base64 строки "<AuthMode>:<Domain\Login>:<Password>", см.
	// PowerShell-пример в официальном Operations Manager REST API
	// Reference): "Network" - явные учётные данные (Domain/Login/Password
	// ниже); "Windows" - учётные данные процесса (Negotiate/Kerberos), для
	// Go потребует отдельного SPNEGO-клиента - до проверки на реальном
	// сервере использовать "Network".
	AuthMode string
	Domain   string // домен AD учётной записи (часть "DOMAIN\login")
	Login    string // сервисная учётная запись с правами Operator/Read-only Operator в SCOM
	// Password - пароль; значение "password" - как и у ConfigOV/vCenter,
	// признак "подставить из хранилища ключей" (utilities.GetKeyData(Login)).
	Password string
	// InsecureSkipVerify - отключить проверку TLS-сертификата (как
	// zabbixmetrics.InsecureSkipVerify); только для тестовых стендов.
	InsecureSkipVerify bool
	// ClassName - класс объектов SCOM, которые опрашиваются (тело запроса
	// POST /data/scomObjectsByClass); пусто -> "Microsoft.Windows.Computer".
	// Метрики собираются только для тех из них, кто сопоставлен с
	// ServerSprRecords аппаратного сервера (Type=0) без ZabbixHostId.
	ClassName string
	// IntervalSec - период опроса, сек; 0 -> 300. Правила сбора
	// счётчиков Windows Server MP по умолчанию пишут значения раз в 5
	// минут, поэтому поминутный опрос (как у Zabbix) смысла не имеет.
	IntervalSec int
	// InventoryIntervalSec - период обновления списка компьютеров, их
	// свойств (ScomHost, дозаполнение ServerSprRecords) и списков счётчиков,
	// сек; 0 -> 3600.
	InventoryIntervalSec int
	// PerfDuration - значение поля "duration" запроса POST
	// /OperationsManager/data/performance (окно, за которое SCOM вернёт
	// average/min/max/last). Единица измерения в документации НЕ указана,
	// поэтому сборщик берёт "lastvalue" (последний замер), а окно нужно
	// лишь достаточно широким, чтобы в него попал хотя бы один замер;
	// 0 -> 60.
	PerfDuration int
	// OperationTimeoutSec - таймаут одного HTTP-запроса к SCOM, сек; 0 -> 120.
	OperationTimeoutSec int
}

// AuthenticationGroup описывает соответствие одной группы MS Active
// Directory уровню привилегий loadmon (см. JsonConfig.Auth) - используется
// при доменной (Kerberos/SPNEGO) аутентификации: список групп пользователя,
// полученный из PAC Kerberos-билета (ADCredentials.GroupMembershipSIDs, см.
// server_web_new/main_web.go:appHandler), сверяется с ObjectSID записей
// этого списка.
type AuthenticationGroup struct {
	Privilege  int    // уровень привилегий loadmon: 0 - просмотр, 1 - редактирование справочников, 3 - администрирование (те же значения, что и models.AuthUser.Priv)
	Name       string // имя группы в MS Active Directory (справочно, в сверке не участвует - PAC-билет имён групп не содержит)
	ObjectGUID string // ObjectGUID группы в MS Active Directory (справочно, зарезервировано на случай перехода на LDAP-сверку)
	ObjectSID  string // ObjectSID группы в MS Active Directory - именно по этому полю происходит сверка с группами пользователя из Kerberos-билета
}

// JsonConfig — структура конфигурации приложения, загружаемая из файла app2.json:
// параметры подключения к БД, режимы работы, адреса gRPC-серверов и клиентов,
// список опрашиваемых vCenter и OneView.
type JsonConfig struct {
	AliasName  string
	DriverName string
	UserName   string
	Password   string
	DBName     string
	DBFile     string // путь к файлу sqlite3 (DBFile в app2.json); относительный — от каталога conf
	DBFileAdd  string // путь ко второй sqlite3 (default-add); относительный — от conf
	HostName   string
	Port       int
	SSLmode    string
	ORMdebug   bool
	WorkMode   string
	AuthMode   string // режим авторизации пользователей веб-интерфейса: "domain" - Kerberos/SPNEGO (Active Directory), "local" - логин/пароль по локальной таблице models.AuthUser. Пусто или неизвестное значение трактуется как "domain" (поведение по умолчанию, сохраняет совместимость)
	// Auth - соответствие групп MS Active Directory уровням привилегий
	// loadmon, используется при доменной (Kerberos/SPNEGO) аутентификации
	// (см. AuthenticationGroup и ResolveDomainPrivilege ниже). Если ни одна
	// из групп пользователя не найдена в этом списке (в т.ч. если сам
	// список пуст) - доступ отклоняется, пользователю показывается
	// страница с предложением подать заявку на доступ вместо создания
	// сессии (см. controllers/user2.go:Login).
	Auth []AuthenticationGroup
	// AccessRequestURL - адрес формы/системы подачи заявки ("служебной
	// записки") на предоставление доступа, показывается доменному
	// пользователю, чьи группы AD не найдены в Auth. Пусто - ссылка на
	// странице отказа в доступе не показывается, только текстовая
	// инструкция обратиться к администратору.
	AccessRequestURL string
	HttpKey          string
	HttpsKey         string
	GRPCSrvDBAddr    string
	GRPCSrvDBPort    int64
	GRPCCliDBAddr    string
	GRPCCliDBPort    int64
	GRPCSrvApiAddr   string
	GRPCSrvApiPort   int64
	GRPCCliApiAddr   string
	GRPCCliApiPort   int64
	GRPCSwitchAddr   string
	GRPCSwitchPort   int64
	ConfigVCenters   []vspheremetrics.VCenter
	ConfigOV         []OvEndpoint
	// OneViewOperationTimeoutSec — таймаут одной HTTP-операции к HPE OneView (сек).
	OneViewOperationTimeoutSec int
	// OneViewCollectTimeoutSec — общий таймаут на групповой сбор инвентаризации OneView (сек).
	OneViewCollectTimeoutSec int
	// GrpcHardwareTimeoutSec — таймаут gRPC-запросов к инвентарю оборудования OneView (сек).
	GrpcHardwareTimeoutSec int

	// ConfigSCOM - подключения к Microsoft SCOM (REST API) для сбора
	// метрик ОС серверов и сведений для ServerSprRecords (пакет
	// scommetrics), см. ScomEndpoint и kontext/13-scom-metrics.md.
	ConfigSCOM []ScomEndpoint

	// --- Уведомления по электронной почте (SMTP) - добавлено 2026-09-04
	// по запросу "реализовать возможность отправки уведомлений по
	// электронной почте через smtp, параметры хранить в файле
	// конфигурации" (см. cmdb/notify.go). Параметры хранятся в том же
	// app2.json, что и остальная конфигурация приложения (см. пример
	// значений в conf/app2.json.example) - отдельный файл конфигурации
	// заводить не стали, чтобы не плодить второй механизм конфигурации
	// (в проекте уже есть beego conf/app.conf для фреймворка и app2.json
	// для бизнес-логики - см. kontext/05-conventions-and-gotchas.md).
	SmtpHost string // адрес SMTP-сервера, например "smtp.company.local"
	SmtpPort int    // порт SMTP-сервера (обычно 25/465/587); 0 -> defaultSmtpPort (25)
	// SmtpUseAuth - использовать ли SMTP AUTH (PLAIN) с SmtpUsername/
	// SmtpPassword; false - анонимная отправка без авторизации (типично
	// для внутренних релеев без аутентификации).
	SmtpUseAuth  bool
	SmtpUsername string
	SmtpPassword string
	// SmtpUseTLS - обернуть соединение в STARTTLS перед AUTH/DATA (для
	// серверов, требующих шифрование, обычно порт 587).
	SmtpUseTLS bool
	SmtpFrom   string // адрес отправителя в заголовке From
	// NotifyRecipients - список адресов, на которые отправляются
	// уведомления об изменении аппаратной конфигурации серверов/корзин
	// (см. NotifyOnHardwareChange ниже и cmdb/notify.go). Пусто -
	// уведомления не отправляются, даже если NotifyOnHardwareChange=true.
	NotifyRecipients []string
	// NotifyOnHardwareChange - включает отправку email при каждой новой
	// записи в hardware_event_log (см. cmdb/hardwareeventlog.go). По
	// умолчанию false - чтобы не начать рассылку писем на существующих
	// инсталляциях без явного включения в app2.json.
	NotifyOnHardwareChange bool
}

// defaultSmtpPort - порт SMTP по умолчанию, если SmtpPort не задан в
// app2.json (0).
const defaultSmtpPort = 25

// SmtpPortOrDefault возвращает SmtpPort из конфигурации, подставляя
// defaultSmtpPort, если он не задан (0) - тот же приём, что и
// OneViewOperationTimeout/OneViewCollectTimeout ниже для таймаутов.
func (s *JsonConfig) SmtpPortOrDefault() int {
	if s.SmtpPort <= 0 {
		return defaultSmtpPort
	}
	return s.SmtpPort
}

// EndpointRecord содержит адреса gRPC API/DB-сервера и карты соответствия
// vCenter/OneView для одного узла в топологии переключения (Switch).
type EndpointRecord struct {
	GRPCSrvApiAddr string
	GRPCSrvApiPort int64
	GRPCSrvDBAddr  string
	GRPCSrvDBPort  int64
	VcenterMap     cmap.ConcurrentMap[string, string]
	OneViewMap     cmap.ConcurrentMap[string, string]
}

// ConfigSwitch хранит потокобезопасную карту эндпоинтов (EndpointRecord) для
// маршрутизации запросов между несколькими инстансами gRPC-серверов.
type ConfigSwitch struct {
	Endpoint cmap.ConcurrentMap[string, *EndpointRecord]
}

var (
	// ConfigData — глобальный экземпляр конфигурации приложения, заполняемый LoadConfigData.
	ConfigData JsonConfig
	// Switch — глобальная таблица переключения между эндпоинтами gRPC-серверов.
	Switch ConfigSwitch
)

func init() {
	Switch.Endpoint = cmap.New[*EndpointRecord]()
}

// GetSrvApiAddr возвращает адрес вида "host:port" для gRPC API-сервера данной записи.
func (r *EndpointRecord) GetSrvApiAddr() string {
	s := fmt.Sprintf("%s:%d", r.GRPCSrvApiAddr, r.GRPCSrvApiPort)
	return s
}

// GetSrvDBAddr возвращает адрес вида "host:port" для gRPC DB-сервера данной записи.
func (r *EndpointRecord) GetSrvDBAddr() string {
	s := fmt.Sprintf("%s:%d", r.GRPCSrvDBAddr, r.GRPCSrvDBPort)
	return s
}

// confDir возвращает каталог, в котором расположен файл конфигурации app2.json.
func confDir() string {
	return filepath.Dir(configFilePath())
}

// configFilePath определяет путь к файлу конфигурации app2.json: переменная
// окружения LOADMON_CONFIG имеет приоритет, иначе перебираются стандартные
// расположения относительно исполняемого файла и текущего каталога.
func configFilePath() string {
	if p := os.Getenv("LOADMON_CONFIG"); p != "" {
		return p
	}
	candidates := []string{
		filepath.Join(filepath.Dir(os.Args[0]), "conf", "app2.json"),
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "conf", "app2.json"),
			filepath.Join(cwd, "metrics_grpc", "conf", "app2.json"),
			filepath.Join(cwd, "server_web_new", "conf", "app2.json"),
		)
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return candidates[0]
}

// LoadConfigData читает и разбирает файл конфигурации app2.json в структуру JsonConfig.
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
	log.Printf("config: загружен %s (WorkMode=%s, vCenters=%d Oneview=%d SCOM=%d)", fullPath, s.WorkMode, len(s.ConfigVCenters), len(s.ConfigOV), len(s.ConfigSCOM))
	return *s, nil
}

// SaveConfigData сериализует конфигурацию в JSON и сохраняет её в файл app2.json.
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

// InitParams применяет загруженную конфигурацию: обновляет список опрашиваемых
// vCenter и OneView-эндпоинтов в глобальной инфраструктуре и, если WorkMode == "api",
// запускает фоновый поток опроса метрик (StartGetMetricThread).
func (s *JsonConfig) InitParams() error {

	if s == nil {
		return fmt.Errorf("Нет данных для инициализации")
	}

	password := utilities.GetKeyData("usr-nsk-dati-02@global.bcs")

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
	ov.SetCollectTimeout(s.OneViewCollectTimeout())
	for _, ovi := range s.ConfigOV {
		switch ovi.Password {
		case "password":
			ov.AddEndpoint(ovi.Endpoint, ovi.Domain, ovi.Login, password, ovi.Version, s.OneViewOperationTimeoutFor(&ovi))
		default:
			ov.AddEndpoint(ovi.Endpoint, ovi.Domain, ovi.Login, ovi.Password, ovi.Version, s.OneViewOperationTimeoutFor(&ovi))
		}
	}

	if infra.VCenterMapList.Count() > 0 {
		if s.WorkMode == "api" {
			log.Printf("InitParams: запуск опроса метрик для %d vCenter", infra.VCenterMapList.Count())
			go StartGetMetricThread() //запускаем опрос метрик
		} else {
			log.Printf("InitParams: WorkMode=%q — фоновый опрос метрик не запускается (нужен \"api\")", s.WorkMode)
		}
	} else {
		log.Println("InitParams: ConfigVCenters пуст — опрос метрик не запущен")
	}
	return nil
}

// InitOVParams регистрирует эндпоинты OneView из конфигурации в глобальной инфраструктуре OneView.
func (s *JsonConfig) InitOVParams() {
	password := utilities.GetKeyData("usr-nsk-dati-02@global.bcs")

	ov := oneview.GlobalInitOVInfrastructure()
	ov.SetCollectTimeout(s.OneViewCollectTimeout())
	for _, ovi := range s.ConfigOV {
		switch ovi.Password {
		case "password":
			ov.AddEndpoint(ovi.Endpoint, ovi.Domain, ovi.Login, password, ovi.Version, s.OneViewOperationTimeoutFor(&ovi))
		}
	}
}

// GetCliApiAddr возвращает адрес "host:port" клиента gRPC API из глобальной конфигурации.
func GetCliApiAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCCliApiAddr, ConfigData.GRPCCliApiPort)
	return s
}

// GetCliDBAddr возвращает адрес "host:port" клиента gRPC DB из глобальной конфигурации.
func GetCliDBAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCCliDBAddr, ConfigData.GRPCCliDBPort)
	return s
}

// GetSrvApiAddr возвращает адрес "host:port" сервера gRPC API из глобальной конфигурации.
func GetSrvApiAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCSrvApiAddr, ConfigData.GRPCSrvApiPort)
	return s
}

// GetSrvDBAddr возвращает адрес "host:port" сервера gRPC DB из глобальной конфигурации.
func GetSrvDBAddr() string {
	s := fmt.Sprintf("%s:%d", ConfigData.GRPCSrvDBAddr, ConfigData.GRPCSrvDBPort)
	return s
}

// GetSwitchAddr возвращает адрес "host:port" сервиса переключения (Switch) из конфигурации.
func (s *JsonConfig) GetSwitchAddr() string {
	str := fmt.Sprintf("%s:%d", s.GRPCSwitchAddr, s.GRPCSwitchPort)
	return str
}

const (
	defaultOneViewOperationTimeoutSec = 90
	defaultOneViewCollectTimeoutSec   = 300
	defaultGrpcHardwareTimeoutSec     = 120
)

// OneViewOperationTimeout возвращает таймаут одной HTTP-операции к OneView.
func (s *JsonConfig) OneViewOperationTimeout() time.Duration {
	sec := s.OneViewOperationTimeoutSec
	if sec <= 0 {
		sec = defaultOneViewOperationTimeoutSec
	}
	return time.Duration(sec) * time.Second
}

// OneViewOperationTimeoutFor возвращает таймаут операции для конкретного endpoint (с учётом переопределения).
func (s *JsonConfig) OneViewOperationTimeoutFor(ovi *OvEndpoint) time.Duration {
	if ovi != nil && ovi.OperationTimeoutSec > 0 {
		return time.Duration(ovi.OperationTimeoutSec) * time.Second
	}
	return s.OneViewOperationTimeout()
}

// OneViewCollectTimeout возвращает общий таймаут на групповой сбор инвентаризации OneView.
func (s *JsonConfig) OneViewCollectTimeout() time.Duration {
	sec := s.OneViewCollectTimeoutSec
	if sec <= 0 {
		sec = defaultOneViewCollectTimeoutSec
	}
	return time.Duration(sec) * time.Second
}

// GetGrpcHardwareTimeout возвращает таймаут gRPC-запросов к инвентарю оборудования.
func GetGrpcHardwareTimeout() time.Duration {
	sec := ConfigData.GrpcHardwareTimeoutSec
	if sec <= 0 {
		sec = defaultGrpcHardwareTimeoutSec
	}
	return time.Duration(sec) * time.Second
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

// ResolveDomainPrivilege сопоставляет список SID групп Active Directory
// пользователя, прошедшего Kerberos/SPNEGO-аутентификацию (PAC-билет,
// ADCredentials.GroupMembershipSIDs), с настроенными в Auth соответствиями
// "группа -> привилегия". Если пользователь состоит сразу в нескольких
// группах, перечисленных в Auth, с разными Privilege - возвращается
// максимальный из найденных уровней. found == false означает, что ни одна
// группа пользователя не найдена в Auth (в т.ч. когда сам список Auth ещё
// не заполнен) - в этом случае доступ должен быть отклонён, а не выдан с
// каким-то значением по умолчанию.
func (s *JsonConfig) ResolveDomainPrivilege(groupSIDs []string) (privilege int, found bool) {
	if len(s.Auth) == 0 || len(groupSIDs) == 0 {
		return 0, false
	}
	sids := make(map[string]bool, len(groupSIDs))
	for _, sid := range groupSIDs {
		sids[sid] = true
	}
	for _, g := range s.Auth {
		if g.ObjectSID == "" || !sids[g.ObjectSID] {
			continue
		}
		if !found || g.Privilege > privilege {
			privilege = g.Privilege
		}
		found = true
	}
	return privilege, found
}

func pollVCenter(vc *vspheremetrics.VCenter) {
	infra := vspheremetrics.GetGlobalVMInfrastructure()
	if infra.IsLockGetMetric() {
		log.Printf("опрос метрик пропущен (блокировка): %s", vc.Hostname)
		return
	}
	log.Printf("опрос метрик: %s", vc.Hostname)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := infra.UpdateListCluster(ctx, *vc); err != nil {
		log.Printf("UpdateListCluster %s: %v", vc.Hostname, err)
	}
	if err := infra.LoadHostMetricN(ctx, *vc); err != nil {
		log.Printf("LoadHostMetricN %s: %v", vc.Hostname, err)
	}
	if err := infra.LoadVmMetricV4(ctx, *vc); err != nil {
		log.Printf("LoadVmMetricV4 %s: %v", vc.Hostname, err)
	}
	// Сверка пула в памяти с фактическим состоянием vCenter и удаление
	// объектов (ВМ/хостов/кластеров), которых там больше нет - см.
	// комментарий к vspheremetrics.PruneStaleObjects.
	if err := infra.PruneStaleObjects(ctx, *vc); err != nil {
		log.Printf("PruneStaleObjects %s: %v", vc.Hostname, err)
	}
}

// StartGetMetricThread запускает по отдельной горутине на каждый настроенный
// vCenter: первый опрос выполняется немедленно, далее — по таймеру с интервалом,
// заданным для конкретного vCenter (vc.Wait), до сигнала остановки (DoneFl).
func StartGetMetricThread() {
	infra := vspheremetrics.GetGlobalVMInfrastructure()
	infra.StartVcenterTimer()
	count := int32(infra.VCenterMapList.Count())
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		vcLocal := vc
		go func() {
			infra2 := vspheremetrics.GetGlobalVMInfrastructure()
			atomic.AddInt64(&infra2.CountThread, int64(1))
			defer atomic.AddInt64(&infra2.CountThread, int64(-1))
			log.Println("Запуск обработки запросов по: ", vcLocal.Hostname, " с интервалом:", vcLocal.Wait, " сек.")

			atomic.StoreInt64(&vcLocal.CountThread, int64(1)) //заглушка

			// первый опрос сразу, без ожидания Wait секунд
			pollVCenter(vcLocal)

			ticker := time.NewTicker(time.Second * time.Duration(1))
			defer ticker.Stop()

			for {
				select {
				case <-vcLocal.Ticker.C:
					pollVCenter(vcLocal)
				case <-ticker.C:
					if atomic.CompareAndSwapInt32(&vcLocal.DoneFl, 1, 0) {
						atomic.StoreInt64(&vcLocal.CountThread, int64(0)) //заглушка
						atomic.AddInt32(&count, -1)
						log.Println("Остановка запросов по: ", vcLocal.Hostname, "осталось", atomic.LoadInt32(&count), "активных обработчиков")
						return
					}
				}
			}
		}()
	}
}
