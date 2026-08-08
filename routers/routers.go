package routers

import (
	"loadmon/controllers"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	beego "github.com/beego/beego/v2/server/web"
)

func init() {
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
	beego.Router("/sprsrv/updateselectedbase/:selected", &controllers.MainController{}, "get,post:SprSrvUpdateSelectedBase")
	//beego.Router("/sprsrv/updatecfo/:backmode/:mode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateCFO") // метод SprSrvUpdateCFO не реализован в controllers
	//beego.Router("/sprsrv/viewserverhw/:server", &controllers.MainController{}, "get,post:ViewServerHW") // метод ViewServerHW не реализован в controllers
	//beego.Router("/sprsrv/updatehw/:backmode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateHW") // метод SprSrvUpdateHW не реализован в controllers
	//beego.Router("/sprsrv/updatevm/:backmode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateVM") // метод SprSrvUpdateVM не реализован в controllers
	//beego.Router("/sprsrv/updatevmcluster/:backmode/:id([0-9]+)", &controllers.MainController{}, "get,post:SprSrvUpdateVMCluster") // метод SprSrvUpdateVMCluster не реализован в controllers
	//beego.Router("/sprsrv/hwconflist/:id([0-9]+)", &controllers.MainController{}, "get,post:SprHwConfig") // метод SprHwConfig не реализован в controllers
	//beego.Router("/sprsrv/hwconfedit/:id([0-9]+)", &controllers.MainController{}, "get,post:SprHwConfigEdit") // метод SprHwConfigEdit не реализован в controllers
	//beego.Router("/sprsrv/hwconfadd", &controllers.MainController{}, "get,post:SprAddHWServerConfig") // метод SprAddHWServerConfig не реализован в controllers
	//beego.Router("/sprsrv/srvhwviewbysn/:SN", &controllers.MainController{}, "get,post:SrvHwViewBySn") // метод SrvHwViewBySn не реализован в controllers
	//beego.Router("/sprsrv/hwviewlist", &controllers.MainController{}, "get,post:SprHwViewList") // метод SprHwViewList не реализован в controllers
	beego.Router("/sprsrv/vm/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVM")
	//beego.Router("/sprsrv/hs/:Vcenter/:Datacenter/:Cluster/:HS", &controllers.MainController{}, "get,post:SprHost") // метод SprHost не реализован в controllers
	//beego.Router("/sprsrv/cluster/:Vcenter/:Datacenter/:Cluster", &controllers.MainController{}, "get,post:SprCluster") // метод SprCluster не реализован в controllers
	//beego.Router("/sprsrv/clusterperiod/:Vcenter/:Datacenter/:Cluster", &controllers.MainController{}, "get,post:SprClusterPeriod") // метод SprClusterPeriod не реализован в controllers
	//beego.Router("/sprsrv/vmperf/:Id", &controllers.MainController{}, "get,post:SprSrvVMPerf") // метод SprSrvVMPerf не реализован в controllers
	//beego.Router("/sprsrv/vmperfkeytest/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVMPerfKeyTest") //тестовой вывод метрик по виртуальной машине // метод SprSrvVMPerfKeyTest не реализован в controllers
	//beego.Router("/sprsrv/vmdelperfkey/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVMDelPerfKey") //удаление метрик по VM в определенном диалозоне времени // метод SprSrvVMDelPerfKey не реализован в controllers
	//beego.Router("/sprsrv/vmperfperiod/:Vcenter/:Datacenter/:Cluster/:VM", &controllers.MainController{}, "get,post:SprSrvVMPerfPeriod") // метод SprSrvVMPerfPeriod не реализован в controllers
	//beego.Router("/sprsrv/vmperfperiodbysrvid/:ServerId", &controllers.MainController{}, "get,post:SprSrvVMPerfPeriodSererId") // метод SprSrvVMPerfPeriodSererId не реализован в controllers
	//beego.Router("/sprsrv/hostperfbyid/:Id", &controllers.MainController{}, "get:HostPerfById") // метод HostPerfById не реализован в controllers
	//beego.Router("/sprsrv/hostperfbyserverid/:Mode/:ServerId", &controllers.MainController{}, "get,post:HostPerfByServerId") // метод HostPerfByServerId не реализован в controllers
	//beego.Router("/sprsrv/clusterperfbyid/:Id", &controllers.MainController{}, "get:ClusterPerfById") // метод ClusterPerfById не реализован в controllers
	//beego.Router("/sprsrv/clusterallocation/:Name", &controllers.MainController{}, "get:ClusterAllocationByName") // метод ClusterAllocationByName не реализован в controllers
	//beego.Router("/sprsrv/zabbixperfperiod/:Id", &controllers.MainController{}, "get,post:SprSrvZabbixPerfPeriod") // метод SprSrvZabbixPerfPeriod не реализован в controllers
	//beego.Router("/sprsrv/bladesystems", &controllers.MainController{}, "get:SprBladeSystems") // метод SprBladeSystems не реализован в controllers
	//beego.Router("/sprsrv/bladesystemconfig/:Id", &controllers.MainController{}, "get,post:SprBladeSystemConfig") // метод SprBladeSystemConfig не реализован в controllers
	//beego.Router("/sprsrv/storagesystems", &controllers.MainController{}, "get:SprStorageSystems") // метод SprStorageSystems не реализован в controllers
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
	//beego.Router("/sprsrv/listclusterview", &controllers.MainController{}, "get,post:ListClusterView") // метод ListClusterView не реализован в controllers
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
