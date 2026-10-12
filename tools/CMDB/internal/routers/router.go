// Package routers регистрирует маршруты и сквозные фильтры cmdb-web.
package routers

import (
	web "github.com/beego/beego/v2/server/web"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	apictl "cmdb/internal/controllers/api"
	"cmdb/internal/controllers/ui"
	"cmdb/internal/filters"
)

// Init подключает фильтры и маршруты. Вызывается один раз при старте cmdb-web.
func Init() {
	web.InsertFilter("/*", web.BeforeRouter, filters.RequestID)
	web.InsertFilter("/*", web.BeforeRouter, filters.Auth)
	web.InsertFilter("/*", web.BeforeRouter, filters.RBAC)
	web.InsertFilter("/*", web.BeforeRouter, filters.AsOf)
	web.InsertFilter("/*", web.BeforeRouter, filters.AsOfReadOnly)
	web.InsertFilter("/*", web.FinishRouter, filters.AccessLog)

	web.Router("/healthz", &ui.HealthController{})
	web.Router("/login", &ui.AuthController{}, "get:LoginForm;post:Login")
	web.Router("/logout", &ui.AuthController{}, "post:Logout")
	web.Router("/admin/users", &ui.UserAdminController{}, "get:Users;post:CreateUser")
	web.Router("/admin/users/:id([0-9]+)/role", &ui.UserAdminController{}, "post:GrantRole")
	web.Router("/admin/users/:id([0-9]+)/role/revoke", &ui.UserAdminController{}, "post:RevokeRole")
	web.Router("/admin/users/:id([0-9]+)/enabled", &ui.UserAdminController{}, "post:SetEnabled")
	web.Router("/admin/users/:id([0-9]+)/password", &ui.UserAdminController{}, "post:SetPassword")
	web.Router("/admin/groups", &ui.UserAdminController{}, "get:Groups;post:SetGroupRole")
	web.Router("/admin/groups/delete", &ui.UserAdminController{}, "post:UnsetGroupRole")

	// Журнал действий привилегированных пользователей (plan.md E7.6).
	web.Router("/admin/events", &ui.EventsController{}, "get:List")
	web.Router("/admin/events/export.csv", &ui.EventsController{}, "get:ExportCSV")
	web.Router("/admin/events/export.cef", &ui.EventsController{}, "get:ExportCEF")
	web.Handler("/metrics", promhttp.Handler())

	web.Router("/", &ui.CIController{}, "get:Home")
	web.Router("/cis", &ui.CIController{}, "get:List;post:Create")
	web.Router("/cis/data", &ui.CIController{}, "get:GridData")
	web.Router("/cis/export.csv", &ui.CIController{}, "get:ExportCSV")
	web.Router("/cis/export.xlsx", &ui.CIController{}, "get:ExportXLSX")
	web.Router("/cis/new", &ui.CIController{}, "get:New")
	web.Router("/cis/new/fields", &ui.CIController{}, "get:NewFields")
	web.Router("/cis/name-preview", &ui.CIController{}, "get:NamePreview")

	web.Router("/cis/:id([0-9]+)", &ui.CIController{}, "get:Card")
	web.Router("/cis/:id([0-9]+)/tab/:tab", &ui.CIController{}, "get:Tab")
	web.Router("/cis/:id([0-9]+)/compare", &ui.CIController{}, "get:Compare")
	web.Router("/cis/:id([0-9]+)/update", &ui.CIController{}, "post:Update")
	web.Router("/cis/:id([0-9]+)/ext", &ui.CIController{}, "post:UpdateExt")
	web.Router("/cis/:id([0-9]+)/lifecycle", &ui.CIController{}, "post:Lifecycle")
	web.Router("/cis/:id([0-9]+)/components", &ui.CIController{}, "post:AddComponent")
	web.Router("/cis/:id([0-9]+)/software", &ui.CIController{}, "post:AddSoftware")
	web.Router("/cis/:id([0-9]+)/interfaces", &ui.CIController{}, "post:AddInterface")
	web.Router("/cis/:id([0-9]+)/ips", &ui.CIController{}, "post:AssignIP")
	web.Router("/cis/:id([0-9]+)/relations", &ui.CIController{}, "post:AddRelation")
	web.Router("/cis/bulk/preview", &ui.CIController{}, "post:BulkPreview")
	web.Router("/cis/bulk/apply", &ui.CIController{}, "post:BulkApply")

	// Глобальный журнал изменений (plan.md E3.7).
	web.Router("/changes", &ui.AuditController{}, "get:List")
	web.Router("/changes/export.csv", &ui.AuditController{}, "get:ExportCSV")
	web.Router("/changes/:id([0-9]+)", &ui.AuditController{}, "get:Detail")

	// Импорт CSV/XLSX и очередь конфликтов (plan.md E4).
	web.Router("/import", &ui.ImportController{}, "get:Index")
	web.Router("/import/template", &ui.ImportController{}, "get:Template")
	web.Router("/import/run", &ui.ImportController{}, "post:Run")
	web.Router("/import/runs/:id([0-9]+)", &ui.ImportController{}, "get:RunDetail")
	web.Router("/import/conflicts", &ui.ImportController{}, "get:Conflicts")
	web.Router("/import/connectors/:code/run", &ui.ImportController{}, "post:ConnectorRun")
	web.Router("/import/connectors/:code/test", &ui.ImportController{}, "post:ConnectorTest")
	web.Router("/import/jobs/:id([0-9]+)/cancel", &ui.ImportController{}, "post:JobCancel")

	// Срезы, сравнение и выгрузка (plan.md E6.2–E6.4); учётные периоды (E6.5).
	web.Router("/baselines", &ui.BaselineController{}, "get:List;post:Create")
	web.Router("/baselines/compare", &ui.BaselineController{}, "get:Compare")
	web.Router("/baselines/:id([0-9]+)", &ui.BaselineController{}, "get:Show")
	web.Router("/baselines/:id([0-9]+)/freeze", &ui.BaselineController{}, "post:Freeze")
	web.Router("/baselines/:id([0-9]+)/approve", &ui.BaselineController{}, "post:Approve")
	web.Router("/baselines/:id([0-9]+)/delete", &ui.BaselineController{}, "post:Delete")
	web.Router("/baselines/:id([0-9]+)/verify", &ui.BaselineController{}, "post:Verify")
	web.Router("/baselines/:id([0-9]+)/export/:fmt", &ui.BaselineController{}, "get:Export")
	web.Router("/periods", &ui.PeriodController{}, "get:List")
	web.Router("/periods/close", &ui.PeriodController{}, "post:Close")
	web.Router("/periods/reopen", &ui.PeriodController{}, "post:Reopen")

	// Эталонные конфигурации (plan.md E7.8).
	web.Router("/admin/baseline-configs", &ui.BaselineConfigController{}, "get:List;post:Create")
	web.Router("/admin/baseline-configs/new", &ui.BaselineConfigController{}, "get:New")
	web.Router("/admin/baseline-configs/:id([0-9]+)/edit", &ui.BaselineConfigController{}, "get:Edit")
	web.Router("/admin/baseline-configs/:id([0-9]+)", &ui.BaselineConfigController{}, "post:Update")
	web.Router("/admin/baseline-configs/:id([0-9]+)/delete", &ui.BaselineConfigController{}, "post:Delete")

	// Дашборд качества данных (plan.md E7.4).
	web.Router("/quality", &ui.QualityController{}, "get:Index")

	// Типовые отчёты (plan.md E7.5).
	web.Router("/reports", &ui.ReportController{}, "get:Index")
	web.Router("/reports/inventory", &ui.ReportController{}, "get:Inventory")
	web.Router("/reports/inventory/export.csv", &ui.ReportController{}, "get:InventoryCSV")
	web.Router("/reports/storage", &ui.ReportController{}, "get:Storage")
	web.Router("/reports/storage/export.csv", &ui.ReportController{}, "get:StorageCSV")
	web.Router("/reports/vm-clusters", &ui.ReportController{}, "get:VMClusters")
	web.Router("/reports/vm-clusters/export.csv", &ui.ReportController{}, "get:VMClustersCSV")
	web.Router("/reports/no-owner", &ui.ReportController{}, "get:NoOwner")
	web.Router("/reports/no-owner/export.csv", &ui.ReportController{}, "get:NoOwnerCSV")
	web.Router("/reports/naming", &ui.ReportController{}, "get:Naming")
	web.Router("/reports/naming/export.csv", &ui.ReportController{}, "get:NamingCSV")
	web.Router("/reports/software", &ui.ReportController{}, "get:Software")
	web.Router("/reports/software/export.csv", &ui.ReportController{}, "get:SoftwareCSV")
	web.Router("/reports/baseline-config", &ui.ReportController{}, "get:BaselineConfig")
	web.Router("/reports/baseline-config/export.csv", &ui.ReportController{}, "get:BaselineConfigCSV")
	web.Router("/reports/stale-inventory", &ui.ReportController{}, "get:StaleInventory")
	web.Router("/reports/stale-inventory/export.csv", &ui.ReportController{}, "get:StaleInventoryCSV")
	web.Router("/reports/kii", &ui.ReportController{}, "get:KII")
	web.Router("/reports/kii/export.csv", &ui.ReportController{}, "get:KIICSV")
	web.Router("/reports/domestic", &ui.ReportController{}, "get:Domestic")
	web.Router("/reports/domestic/hardware.csv", &ui.ReportController{}, "get:DomesticHardwareCSV")
	web.Router("/reports/domestic/software.csv", &ui.ReportController{}, "get:DomesticSoftwareCSV")
	web.Router("/reports/legacy-names", &ui.ReportController{}, "get:LegacyNames")
	web.Router("/reports/legacy-names/export.csv", &ui.ReportController{}, "get:LegacyNamesCSV")

	// Песочница правил именования (plan.md E3.8).
	web.Router("/admin/naming", &ui.NamingController{}, "get:Sandbox")
	web.Router("/admin/naming/check", &ui.NamingController{}, "get:Check")

	web.Router("/components/:cid([0-9]+)/replace", &ui.CIController{}, "post:ReplaceComponent")
	web.Router("/software-installs/:sid([0-9]+)/remove", &ui.CIController{}, "post:RemoveSoftware")
	web.Router("/ips/:ipid([0-9]+)/release", &ui.CIController{}, "post:ReleaseIP")
	web.Router("/relations/:rid([0-9]+)/close", &ui.CIController{}, "post:CloseRelation")

	// Администрирование справочников ref.* (plan.md E1.8) — один контроллер на все таблицы.
	web.Router("/admin/ref", &ui.RefAdminController{}, "get:Index")
	web.Router("/admin/ref/:table", &ui.RefAdminController{}, "get:List;post:Create")
	web.Router("/admin/ref/:table/new", &ui.RefAdminController{}, "get:New")
	web.Router("/admin/ref/:table/export", &ui.RefAdminController{}, "get:Export")
	web.Router("/admin/ref/:table/import", &ui.RefAdminController{}, "post:Import")
	web.Router("/admin/ref/:table/:pk/edit", &ui.RefAdminController{}, "get:Edit;post:Update")
	web.Router("/admin/ref/:table/:pk/delete", &ui.RefAdminController{}, "post:Delete")
	web.Router("/admin/ref/location/:code/tree", &ui.RefAdminController{}, "get:LocationTree")

	// Сервисные учётные записи и токены API (plan.md E7.3).
	web.Router("/admin/api-tokens", &ui.TokenAdminController{}, "get:Index")
	web.Router("/admin/api-tokens/accounts", &ui.TokenAdminController{}, "post:CreateServiceAccount")
	web.Router("/admin/api-tokens/issue", &ui.TokenAdminController{}, "post:IssueToken")
	web.Router("/admin/api-tokens/:id([0-9]+)/revoke", &ui.TokenAdminController{}, "post:RevokeToken")

	// REST API /api/v1 (plan.md E7.3): КЕ, компоненты, связи, справочники, история, срезы.
	// Аутентификация — токен API (Authorization: Bearer) или cookie сессии, права — тот же RBAC,
	// «показать на дату» — тот же ?asof= (internal/filters.AsOf/AsOfReadOnly, общие фильтры "/*").
	web.Router("/api/v1/openapi.yaml", &apictl.OpenAPIController{}, "get:Spec")
	web.Router("/api/v1/cis", &apictl.CIController{}, "get:List;post:Create")
	web.Router("/api/v1/cis/:id([0-9]+)", &apictl.CIController{}, "get:Get;patch:Update")
	web.Router("/api/v1/cis/:id([0-9]+)/lifecycle", &apictl.CIController{}, "post:Lifecycle")
	web.Router("/api/v1/cis/:id([0-9]+)/history", &apictl.CIController{}, "get:History")
	web.Router("/api/v1/cis/:id([0-9]+)/components", &apictl.CIController{}, "get:Components;post:AddComponent")
	web.Router("/api/v1/cis/:id([0-9]+)/relations", &apictl.CIController{}, "get:Relations;post:AddRelation")
	web.Router("/api/v1/relations/:id([0-9]+)/close", &apictl.RelationsController{}, "post:Close")
	web.Router("/api/v1/ref/:name", &apictl.RefController{}, "get:List")
	web.Router("/api/v1/baselines", &apictl.BaselinesController{}, "get:List")
	web.Router("/api/v1/baselines/:id([0-9]+)", &apictl.BaselinesController{}, "get:Get")
}
