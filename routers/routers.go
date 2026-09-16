// Package routers связывает HTTP-маршруты веб-приложения loadmon с методами
// контроллера MainController: регистрирует пути аутентификации, справочников
// серверов/ВМ/кластеров, API-запросов и служебных эндпоинтов (метрики
// Prometheus, профилирование pprof). Маршруты регистрируются в функции init
// через API фреймворка beego.
package routers

import (
	"loadmon/controllers"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	// Централизованная проверка сессии ("авторизован ли пользователь") -
	// заменяет собой ~40 ранее продублированных в отдельных хендлерах
	// проверок `if this.GetSession("loadmon") == nil { this.Redirect(...) }`.
	// Список путей, доступных без сессии (страницы входа/регистрации,
	// технические эндпоинты и т.п.), и обоснование двух исключений (/api/,
	// /sprsrv/itsystemcorrectpreview/) - см. controllers/auth_filter.go.
	beego.InsertFilter("*", beego.BeforeRouter, controllers.AuthFilter)

	beego.Router("/", &controllers.MainController{})
	beego.Handler("/metrics", promhttp.Handler())
	beego.Router("/user/login/:back", &controllers.MainController{}, "get,post:Login")
	beego.Router("/user/logout", &controllers.MainController{}, "get:Logout")
	beego.Router("/shutdown", &controllers.MainController{}, "post:Shutdown")
	beego.Router("/user/register", &controllers.MainController{}, "get,post:Register")
	beego.Router("/user/verify/:uuid", &controllers.MainController{}, "get:Verify")
	beego.Router("/user/profile", &controllers.MainController{}, "get,post:Profile")
	beego.Router("/user/remove", &controllers.MainController{}, "get,post:Remove")
	beego.Router("/user/forgot", &controllers.MainController{}, "get,post:Forgot")
	beego.Router("/user/reset/:uuid", &controllers.MainController{}, "get,post:Reset")
	beego.Router("/favicon.ico", &controllers.MainController{}, "get:Favicon_ico")
	beego.Router("/sso", &controllers.MainController{}, "get,post:GetSSO")
	//beego.Router("/view/:Mode", &controllers.MainController{}, "get:View") // view.tpl отсутствует
	//beego.Router("/top/:Mode", &controllers.MainController{}, "get:ViewTop") // top.tpl отсутствует
	beego.Router("/filter", &controllers.MainController{}, "get,post:Filter")
	beego.Router("/details/:Mode/:ServerName", &controllers.MainController{}, "get,post:ServerDetails")
	//beego.Router("/detailsvm/:ServerName", &controllers.MainController{}, "get,post:ServerDetailsVM") // метод ServerDetailsVM не реализован в controllers
	//beego.Router("/detailskeyvm/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:ServerDetailsKeyVM") // метод ServerDetailsKeyVM не реализован в controllers
	//beego.Router("/detailsvmperiod/:Mode/:ServerName", &controllers.MainController{}, "get,post:ServerDetailsVMPeriod") // метод ServerDetailsVMPeriod не реализован в controllers
	//beego.Router("/detailsvmid/:Mode/:Id", &controllers.MainController{}, "get,post:ServerDetailsVMId") // метод ServerDetailsVMId не реализован в controllers
	beego.Router("/notice", &controllers.MainController{}, "get:Notice")
	beego.Router("/sprsrv/view2/:mode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvViewV2")
	beego.Router("/sprsrv/addserver/:backmode/:name", &controllers.MainController{}, "get,post:SprAddServer")
	beego.Router("/sprsrv/updatebase/:backmode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateBase")
	beego.Router("/sprsrv/vmeditbyname/:Name", &controllers.MainController{}, "get:SprSrvUpdateBaseByVMName") // находит запись models.ServerSprRecords по имени виртуальной машины (Type=1) и переходит на её /sprsrv/updatebase/all/:id
	beego.Router("/sprsrv/itsystemcorrectpreview/:key", &controllers.MainController{}, "get:SprITSystemCorrectPreview") // JSON-предпросмотр полей, которые пересчитает correctServerSprRecord для выбранной ИТ-системы - используется формой /sprsrv/updatebase для интерактивного обновления полей при выборе ИТ-системы, до сохранения
	beego.Router("/sprsrv/updateselectedbase/:selected", &controllers.MainController{}, "get,post:SprSrvUpdateSelectedBase")
	//beego.Router("/sprsrv/updatecfo/:backmode/:mode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateCFO") // метод SprSrvUpdateCFO не реализован в controllers
	//beego.Router("/sprsrv/viewserverhw/:server", &controllers.MainController{}, "get,post:ViewServerHW") // метод ViewServerHW не реализован в controllers
	//beego.Router("/sprsrv/updatehw/:backmode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateHW") // метод SprSrvUpdateHW не реализован в controllers
	//beego.Router("/sprsrv/updatevm/:backmode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateVM") // метод SprSrvUpdateVM не реализован в controllers
	//beego.Router("/sprsrv/updatevmcluster/:backmode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateVMCluster") // метод SprSrvUpdateVMCluster не реализован в controllers
	beego.Router("/sprsrv/hwconflist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprHwConfig")     // список записей models.ServerHardware (справочник конфигураций серверов) с фильтром и пагинацией
	beego.Router("/sprsrv/hwconfedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprHwConfigEdit") // просмотр/редактирование одной записи models.ServerHardware по Id
	beego.Router("/sprsrv/hwconfeditbysn/:SN", &controllers.MainController{}, "get:SprHwConfigEditBySN")      // находит запись models.ServerHardware по серийному номеру и переходит на её /sprsrv/hwconfedit/:id (используется пунктом меню "Конфигурация сервера" на /sprsrv/view2)
	beego.Router("/sprsrv/hwconfeditbyname/:Name", &controllers.MainController{}, "get:SprHwConfigEditByName") // находит запись models.ServerSprRecords по имени (физический сервер/гипервизор) и переходит на связанную models.ServerHardware через HWSprId (используется хостами на /sprsrv/cluster/... без известного серийного номера)
	beego.Router("/sprsrv/hwconfbulkedit", &controllers.MainController{}, "get,post:SprHwConfigBulkEdit")     // групповое редактирование статуса/размещения выбранных записей models.ServerHardware
	//beego.Router("/sprsrv/hwconfadd", &controllers.MainController{}, "get,post:SprAddHWServerConfig") // метод SprAddHWServerConfig не реализован в controllers
	beego.Router("/sprsrv/srvhwviewbysn/:SN", &controllers.MainController{}, "get,post:SrvHwViewBySn") // подробная карточка сервера HPE OneView по серийному номеру (ссылка из /sprsrv/hwviewlist)
	beego.Router("/sprsrv/hwviewlist", &controllers.MainController{}, "get,post:SprHwViewList")
	beego.Router("/sprsrv/vm/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVM")
	beego.Router("/sprsrv/vmperfday/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVMPerfDay") // динамические графики метрик ВМ (VMPerf) за текущий день
	//beego.Router("/sprsrv/hs/:Vcenter/:Datacenter/:Cluster/:HS", &controllers.MainController{}, "get,post:SprHost") // метод SprHost не реализован в controllers
	beego.Router("/sprsrv/cluster/:Vcenter/:Datacenter/:Cluster", &controllers.MainController{}, "get,post:SprCluster")
	beego.Router("/sprsrv/clusterperioddb", &controllers.MainController{}, "get,post:SprClusterPeriodDB")             // список кластеров/хостов с метриками за период, посчитанными по данным БД (cluster_record/host_record/vm_record), аналогично /sprsrv/cluster, но без обращения к live vSphere через gRPC
	beego.Router("/sprsrv/clustermetricsexport", &controllers.MainController{}, "get,post:SprClusterMetricsExport") // выгрузка метрик кластеров за период (cluster_record, дополненный в начале Vcenter/Datacenter/Cluster) в CSV/XLSX
	beego.Router("/sprsrv/vmmetricsexport", &controllers.MainController{}, "get,post:SprVMMetricsExport")           // выгрузка метрик виртуальных машин за период (vm_record, дополненный Vcenter/Datacenter/Cluster/Vm_Name из vm_spr_record) в CSV/XLSX
	//beego.Router("/sprsrv/clusterperiod/:Vcenter/:Datacenter/:Cluster", &controllers.MainController{}, "get,post:SprClusterPeriod") // метод SprClusterPeriod не реализован в controllers
	//beego.Router("/sprsrv/vmperf/:Id", &controllers.MainController{}, "get,post:SprSrvVMPerf") // метод SprSrvVMPerf не реализован в controllers
	//beego.Router("/sprsrv/vmperfkeytest/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVMPerfKeyTest") //тестовой вывод метрик по виртуальной машине // метод SprSrvVMPerfKeyTest не реализован в controllers
	//beego.Router("/sprsrv/vmdelperfkey/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVMDelPerfKey") //удаление метрик по VM в определенном диалозоне времени // метод SprSrvVMDelPerfKey не реализован в controllers
	beego.Router("/sprsrv/vmperfperiod/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVMPerfPeriod")       // динамические графики загрузки ВМ по данным БД (VMRecord/VMPerfRecord) за произвольный период
	beego.Router("/sprsrv/clusterperfday/:Vcenter/:Datacenter/:Cluster", &controllers.MainController{}, "get,post:SprSrvClusterPerfDay")       // динамические графики метрик кластера (CLPerf) за текущий день
	beego.Router("/sprsrv/clusterperfperiod/:Vcenter/:Datacenter/:Cluster", &controllers.MainController{}, "get,post:SprSrvClusterPerfPeriod") // динамические графики загрузки кластера по данным БД (ClusterRecord/ClusterPerf) за произвольный период
	//beego.Router("/sprsrv/vmperfperiodbysrvid/:ServerId", &controllers.MainController{}, "get,post:SprSrvVMPerfPeriodSererId") // метод SprSrvVMPerfPeriodSererId не реализован в controllers
	//beego.Router("/sprsrv/hostperfbyid/:Id", &controllers.MainController{}, "get:HostPerfById") // метод HostPerfById не реализован в controllers
	//beego.Router("/sprsrv/hostperfbyserverid/:Mode/:ServerId", &controllers.MainController{}, "get,post:HostPerfByServerId") // метод HostPerfByServerId не реализован в controllers
	//beego.Router("/sprsrv/clusterperfbyid/:Id", &controllers.MainController{}, "get:ClusterPerfById") // метод ClusterPerfById не реализован в controllers
	//beego.Router("/sprsrv/clusterallocation/:Name", &controllers.MainController{}, "get:ClusterAllocationByName") // метод ClusterAllocationByName не реализован в controllers
	beego.Router("/sprsrv/zabbixperfperiod/:Id", &controllers.MainController{}, "get,post:SprSrvZabbixPerfPeriod") // графики загрузки аппаратного сервера за период по данным Zabbix (zabbix_item/zabbix_key), см. controllers/sprsrvzabbixperfperiod.go
	beego.Router("/sprsrv/bladesystemslist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprBladeSystems")     // список записей models.BladeSystems (справочник блейд-корзин) с фильтром, пагинацией и экспортом в CSV/XLSX
	beego.Router("/sprsrv/bladesystemsedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprBladeSystemsEdit") // просмотр/редактирование/добавление одной записи models.BladeSystems по Id (Id=0 - новая запись)

	// Справочник ЦОД (models.DatacenterSite) - по аналогии с Datacenter из oneview2, с альтернативным наименованием.
	beego.Router("/sprsrv/datacentersitelist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprDatacenterSite")
	beego.Router("/sprsrv/datacentersiteedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprDatacenterSiteEdit")

	// Справочник шкафов (models.RackCabinet) - по аналогии с Rack из oneview2, связан с ЦОД (DatacenterSiteId)
	// и с BladeSystems/ServerHardware (RackCabinetId на этих моделях).
	beego.Router("/sprsrv/rackcabinetlist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprRackCabinet")
	beego.Router("/sprsrv/rackcabinetedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprRackCabinetEdit")

	// Справочник сетей (models.NetworkSprRecords) для серверов и других компонент инфраструктуры.
	beego.Router("/sprsrv/networklist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprNetwork")
	beego.Router("/sprsrv/networkedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprNetworkEdit")

	// Справочник IP-адресов (models.IPAddressSprRecords) - производная таблица,
	// автосинхронизация из справочника серверов (см. cmdb/syncnetworkip.go),
	// read-only, без формы редактирования (2026-09-16).
	beego.Router("/sprsrv/ipaddresslist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprIPAddress")

	// Справочник сетевых коммутаторов (models.SwitchSprRecords) со связями с сетями (models.SwitchNetworkLink).
	beego.Router("/sprsrv/switchlist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSwitch")
	beego.Router("/sprsrv/switchedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSwitchEdit")
	beego.Router("/sprsrv/bladesystemserverslist/:id([0-9]+)", &controllers.MainController{}, "get:SprBladeSystemServersList") // read-only список models.ServerHardware, установленных в ОДНОЙ блейд-корзине (BladeSystemId), с фиксацией позиции (BladeNum)
	beego.Router("/sprsrv/storagesystemslist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprStorageSystems")     // список записей models.StorageSystems (справочник СХД) с фильтром, пагинацией и экспортом в CSV/XLSX
	beego.Router("/sprsrv/storagesystemsedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprStorageSystemsEdit") // просмотр/редактирование/добавление одной записи models.StorageSystems по Id (Id=0 - новая запись)
	beego.Router("/sprsrv/storagesystemconfigview/:id([0-9]+)", &controllers.MainController{}, "get:SprStorageSystemConfigView") // read-only карточка конфигурации СХД: сама СХД + сторадж-процессоры/полки (models.StorageEnclosure) + диски (models.StorageDisk)
	beego.Router("/sprsrv/storagevolumeslist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprStorageVolumes") // список записей models.StorageVolumes (справочник томов) с фильтром, пагинацией и экспортом в CSV/XLSX, см. controllers/sprstoragevolumes.go
	beego.Router("/sprsrv/storagevolumeview/:id([0-9]+)", &controllers.MainController{}, "get:SprStorageVolumeView")    // read-only карточка детального просмотра одной записи models.StorageVolumes по Id (без формы сохранения - у StorageVolumes нет CRUD)
	beego.Router("/sprsrv/clustersprlist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprClusterList") // список записей models.ClusterSprRecords (справочник кластеров) с фильтром, пагинацией и экспортом в CSV/XLSX
	beego.Router("/sprsrv/clusterspredit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprClusterEdit") // просмотр/редактирование одной записи models.ClusterSprRecords по Id (добавление вручную не предусмотрено - справочник наполняется автоматически)
	//beego.Router("/sprsrv/storagesystemconfig/:Id", &controllers.MainController{}, "get,post:SprStorageSystemConfig") // метод SprStorageSystemConfig не реализован в controllers
	//beego.Router("/sprsrv/storagevolumes", &controllers.MainController{}, "get:SprStorageVolumes") // метод SprStorageVolumes не реализован в controllers
	//beego.Router("/sprsrv/storagevolumeconfig/:Id", &controllers.MainController{}, "get,post:SprStorageVolumeConfig") // метод SprStorageVolumeConfig не реализован в controllers
	//beego.Router("/sprsrv/viewservices", &controllers.MainController{}, "get:ViewServices") // метод ViewServices не реализован в controllers
	//beego.Router("/sprsrv/editservice/:Id", &controllers.MainController{}, "get,post:EditService") // метод EditService не реализован в controllers
	//beego.Router("/sprsrv/addservice", &controllers.MainController{}, "get,post:AddService") // метод AddService не реализован в controllers
	//beego.Router("/sprsrv/delservice/:Id", &controllers.MainController{}, "get,post:DelService") // метод DelService не реализован в controllers
	//beego.Router("/sprsrv/viewmembersfg", &controllers.MainController{}, "get:ViewMembersFG") // метод ViewMembersFG не реализован в controllers
	//beego.Router("/sprsrv/editmemberfg/:Id", &controllers.MainController{}, "get,post:EditMemberFG") // метод EditMemberFG не реализован в controllers
	//beego.Router("/sprsrv/addmemberfg", &controllers.MainController{}, "get,post:AddMemberFG") // метод AddMemberFG не реализован в controllers
	//beego.Router("/sprsrv/delmemberfg/:Id", &controllers.MainController{}, "get,post:DelMemberFG") // метод DelMemberFG не реализован в controllers
	//beego.Router("/sprsrv/viewul", &controllers.MainController{}, "get:ViewCounterparty") // метод ViewCounterparty не реализован в controllers
	//beego.Router("/sprsrv/editul/:Id", &controllers.MainController{}, "get,post:EditCounterparty") // метод EditCounterparty не реализован в controllers
	//beego.Router("/sprsrv/addul", &controllers.MainController{}, "get,post:AddCounterparty") // метод AddCounterparty не реализован в controllers
	//beego.Router("/sprsrv/delul/:Id", &controllers.MainController{}, "get,post:DelCounterparty") // метод DelCounterparty не реализован в controllers
	//beego.Router("/sprsrv/serverowner/:Id", &controllers.MainController{}, "get,post:ServerOwnership") // метод ServerOwnership не реализован в controllers
	//beego.Router("/sprsrv/editsrvdoc/:Id/:docid", &controllers.MainController{}, "get,post:EditServerDocument") // метод EditServerDocument не реализован в controllers
	//beego.Router("/sprsrv/addsrvdoc/:Id", &controllers.MainController{}, "get,post:AddServerDocument") // метод AddServerDocument не реализован в controllers
	//beego.Router("/sprsrv/csvdata", &controllers.MainController{}, "get:GetCSVListServer") // метод GetCSVListServer не реализован в controllers
	beego.Router("/sprsrv/listclusterview", &controllers.MainController{}, "get,post:SprSrvListClusterView") // список кластеров vSphere по аналогии со списком ВМ (/sprsrv/vm)
	beego.Router("/sprsrv/utilization", &controllers.MainController{}, "get,post:SprUtilization") // интерактивная сводная страница оценки утилизации виртуальной инфраструктуры (кластеры/хосты/VM/СХД/LUN), риски перегруза и рекомендации, см. controllers/sprutilization.go
	beego.Router("/admin/userlog", &controllers.MainController{}, "get:ViewUserLog") // журнал действий пользователей (userlog) - доступ только для models.AuthUser.Priv == 3 (проверяется внутри контроллера)
	beego.Router("/admin/storageeventlog", &controllers.MainController{}, "get:ViewStorageEventLog")   // журнал событий СХД (доступность опроса, неизвестные LUN) - хранится 3 суток, доступ только Priv==3, см. controllers/adminstorageeventlog.go
	beego.Router("/admin/hardwareeventlog", &controllers.MainController{}, "get:ViewHardwareEventLog")   // журнал изменений конфигурации серверов/блейд-корзин - хранится 30 суток, доступ только Priv==3, см. controllers/adminhardwareeventlog.go
	beego.Router("/admin/serverspreventlog", &controllers.MainController{}, "get:ViewServerSprEventLog") // журнал ручных изменений карточек серверов (с автором) - хранится 180 суток, доступ только Priv==3, см. controllers/adminserverspreventlog.go
	beego.Router("/admin/metricsexport", &controllers.MainController{}, "get,post:AdminMetricsUnifiedExport") // единообразная выгрузка метрик утилизации (CPU/память/диск/сеть) по ВМ и Zabbix-серверам, ресемплированная на минутную сетку - доступ только для models.AuthUser.Priv == 3 (проверяется внутри контроллера), см. controllers/adminmetricsexport.go
	beego.Router("/admin/deactivatedservers", &controllers.MainController{}, "get,post:AdminDeactivatedServersList") // список серверов, выведенных из учёта автосверкой (ServerSprRecords.Stat==0), с групповым выбором для возврата в работу - доступ только Priv==3, см. controllers/admindeactivatedservers.go
	beego.Router("/admin/deactivatedservers/restore/:selected", &controllers.MainController{}, "get,post:AdminDeactivatedServersRestore") // форма возврата в учёт выбранных серверов (PowerState/Active, каскад на VM/аппаратный сервер) - доступ только Priv==3, см. controllers/admindeactivatedservers.go
	beego.Router("/admin/viewexport/:name", &controllers.MainController{}, "get:AdminViewExport") // просмотр/CSV/XLSX-выгрузка результата SQL-файла conf/viewsql/:name.sql - доступ только Priv==3, доступ через подменю Администрирование→"Выгрузка по View" в header.tpl, см. controllers/adminviewexport.go
	beego.Router("/admin/infrastructure", &controllers.MainController{}, "get:AdminInfrastructure") // canvas-диаграмма инфраструктуры (WASM infracanvas + go-canvas) - доступ только Priv==3, см. controllers/admininfrastructure.go
	beego.Router("/admin/hwinfra", &controllers.MainController{}, "get:AdminHwInfra")     // аппаратная инфраструктура ЦОД: обзор ЦОД -> шкафы -> СХД/серверы/блейд-корзины (обычный JS+canvas), см. controllers/adminhwinfra.go
	beego.Router("/admin/virtinfra", &controllers.MainController{}, "get:AdminVirtInfra") // виртуальная инфраструктура: vCenter -> Datacenter -> Cluster -> Host, см. controllers/adminvirtinfra.go

	beego.Router("/admin/importrackservers", &controllers.MainController{}, "get,post:AdminImportRackServers")             // форма загрузки xlsx-выгрузки Итиллиум по rack-серверам в ./import и асинхронный запуск itilliumimport.StartImportItiliumServer(name) - доступ только Priv==3, см. controllers/adminimportrackservers.go
	beego.Router("/admin/importrackservers/status", &controllers.MainController{}, "get:AdminImportRackServersStatus")     // JSON-опрос прогресса импорта rack-серверов (параметр job=<имя файла>) - для градуированного индикатора на /admin/importrackservers, см. controllers/adminimportrackservers.go
	beego.Router("/admin/importblades", &controllers.MainController{}, "get,post:AdminImportBladeSystems")                 // форма загрузки xlsx-выгрузки Итиллиум по блейд-корзинам в ./import и асинхронный запуск itilliumimport.StartImportItiliumBlade(name) - доступ только Priv==3, см. controllers/adminimportblades.go
	beego.Router("/admin/importblades/status", &controllers.MainController{}, "get:AdminImportBladeSystemsStatus")         // JSON-опрос прогресса импорта блейд-корзин (параметр job=<имя файла>) - для градуированного индикатора на /admin/importblades, см. controllers/adminimportblades.go
	beego.Router("/admin/importbladeservers", &controllers.MainController{}, "get,post:AdminImportBladeServers")           // форма загрузки xlsx-выгрузки Итиллиум по блейд-серверам в ./import и асинхронный запуск itilliumimport.StartImportItiliumBladeServer(name) - доступ только Priv==3, см. controllers/adminimportbladeservers.go
	beego.Router("/admin/importbladeservers/status", &controllers.MainController{}, "get:AdminImportBladeServersStatus")   // JSON-опрос прогресса импорта блейд-серверов (параметр job=<имя файла>) - для градуированного индикатора на /admin/importbladeservers, см. controllers/adminimportbladeservers.go
	beego.Router("/admin/importstorages", &controllers.MainController{}, "get,post:AdminImportStorageSystems")            // форма загрузки xlsx-выгрузки Итиллиум по СХД в ./import и асинхронный запуск itilliumimport.StartImportItiliumStorage(name) - доступ только Priv==3, см. controllers/adminimportstorages.go
	beego.Router("/admin/importstorages/status", &controllers.MainController{}, "get:AdminImportStorageSystemsStatus")    // JSON-опрос прогресса импорта СХД (параметр job=<имя файла>) - для градуированного индикатора на /admin/importstorages, см. controllers/adminimportstorages.go
	beego.Router("/api/:json", &controllers.MainController{}, "get,post:RequestAPI")
	beego.Router("/debug/pprof/", &controllers.MainController{}, "get,post:Prof")
	beego.Router("/debug/pprof/cmdline", &controllers.MainController{}, "get,post:ProfCmdline")
	beego.Router("/debug/pprof/profile", &controllers.MainController{}, "get,post:ProfProfile")
	beego.Router("/debug/pprof/symbol", &controllers.MainController{}, "get,post:ProfSymbol")
	beego.Router("/debug/pprof/trace", &controllers.MainController{}, "get,post:ProfTrace")
	//beego.Router("/cmdb/listcpuspr", &controllers.MainController{}, "get,post:ListCPUSpr") // метод ListCPUSpr не реализован в controllers
	//beego.Router("/cmdb/listramspr", &controllers.MainController{}, "get,post:ListRAMSpr") // метод ListRAMSpr не реализован в controllers
	//beego.Router("/cmdb/listdiskspr", &controllers.MainController{}, "get,post:ListDiskSpr") // метод ListDiskSpr не реализован в controllers
	//beego.Router("/cmdb/listmodelvendorspr", &controllers.MainController{}, "get,post:ListModelVendorSpr") // метод ListModelVendorSpr не реализован в controllers
	//beego.Router("/cmdb/listmodelspr", &controllers.MainController{}, "get,post:ListModelSpr") // метод ListModelSpr не реализован в controllers
	//beego.Router("/cmdb/listconfunit", &controllers.MainController{}, "get,post:ListConfigurationUnits") // метод ListConfigurationUnits не реализован в controllers
	//beego.Router("/cmdb/editconfunitbase/:sn", &controllers.MainController{}, "get,post:EditConfigurationUnitBase") // метод EditConfigurationUnitBase не реализован в controllers
	//beego.Router("/role/list", &controllers.MainController{}, "get,post:ListPersonalRole") // метод ListPersonalRole не реализован в controllers
	//beego.Router("/role/card/:code", &controllers.MainController{}, "get,post:CardPersonalRole") // метод CardPersonalRole не реализован в controllers
	//beego.Router("/role/cardadd", &controllers.MainController{}, "get,post:NewCardPersonalRole") // метод NewCardPersonalRole не реализован в controllers
}
