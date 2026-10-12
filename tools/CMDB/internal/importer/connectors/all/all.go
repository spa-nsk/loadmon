// Package all подключает все виды API-коннекторов (регистрация в init) — импортируется
// исполнителем cmdb-worker и веб-приложением (кнопка «проверить подключение»).
package all

import (
	_ "cmdb/internal/importer/connectors/brocade"
	_ "cmdb/internal/importer/connectors/huawei"
	_ "cmdb/internal/importer/connectors/kubernetes"
	_ "cmdb/internal/importer/connectors/lenovo"
	_ "cmdb/internal/importer/connectors/oneview"
	_ "cmdb/internal/importer/connectors/ontap"
	_ "cmdb/internal/importer/connectors/openmanage"
	_ "cmdb/internal/importer/connectors/powerstore"
	_ "cmdb/internal/importer/connectors/redfish"
	_ "cmdb/internal/importer/connectors/snmpnet"
	_ "cmdb/internal/importer/connectors/unity"
	_ "cmdb/internal/importer/connectors/vcenter"
	_ "cmdb/internal/importer/connectors/vnx"
	_ "cmdb/internal/importer/connectors/zabbix"
	_ "cmdb/internal/importer/connectors/zvirt"
)
