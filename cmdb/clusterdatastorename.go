// Package cmdb (файл clusterdatastorename.go) содержит отдельную функцию
// UpdateClusterDatastoreNames, заполняющую ClusterSprRecords.DatastoreName -
// объединённый через ";" (без дублей) список кодов ЦОД
// (DatacenterSite.CodeJira) всех ESXi гипервизоров кластера:
//
//	ServerSprRecords (Type=2, Vcenter+ClusterId == параметры кластера) ->
//	HWSprId -> ServerHardware.RackCabinetId -> RackCabinet.DatacenterSiteId
//	-> DatacenterSite.CodeJira
//
// По аналогии с VMSprRecord.DatastoreName (см. vmdatastorename.go) эта
// функция НЕ встроена в код сохранения метрик, а вызывается ОТДЕЛЬНО и
// ЗАПУСКАЕТСЯ ПАРАЛЛЕЛЬНО с ним - см. вызов `go UpdateClusterDatastoreNames
// (...)` рядом с SaveVMRecord/UpdateVMDatastoreNames в cmdb.SaveClusterData
// (savemetric_orchestration.go). Обновление записи в БД выполняется через
// отдельный gRPC-метод LoadmonDB.UpdateClusterSprDatastoreName (см.
// protocol/db.proto, grpcserver/handler_db_clusterspr_datastorename.go,
// grpcclient.UpdateClusterSprDatastoreName) - по тому же loopback-паттерну,
// что и UpdateVMSprDatastoreName/AddClusterSprRecord/UpdateClusterSprRecord.
package cmdb

import (
	"loadmon/grpcclient"
	"loadmon/models"
	vsphere "loadmon/vspheremetrics"
	"log"
	"sort"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// UpdateClusterDatastoreNames заполняет DatastoreName записи справочника
// кластеров (cluster_spr_records, Id=clusterSprId) объединённым списком
// кодов ЦОД всех ESXi гипервизоров кластера cl. Рассчитана на запуск как
// отдельная горутина параллельно с cmdb.SaveVMRecord/UpdateVMDatastoreNames
// для того же кластера (см. комментарий к пакету выше).
func UpdateClusterDatastoreNames(cl *vsphere.Cluster, clusterSprId int64) {
	if cl == nil || clusterSprId == 0 {
		return
	}
	o := orm.NewOrm()

	// гипервизоры (Type=2) этого кластера - сопоставление по Vcenter+ClusterId
	// (канонический MOID кластера), без обращения к именам ВМ.
	var hosts []models.ServerSprRecords
	o.QueryTable("server_spr_records").Filter("Type", 2).
		Filter("Vcenter", cl.Vcenter).Filter("ClusterId", cl.ClusterID).
		All(&hosts, "Id", "HWSprId")
	if len(hosts) == 0 {
		return //нет данных о гипервизорах этого кластера - резолвить не из чего
	}

	seen := make(map[string]bool, len(hosts))
	codes := make([]string, 0, len(hosts))
	for _, h := range hosts {
		if h.HWSprId == 0 {
			continue
		}
		code := resolveDatacenterCodeByHWSprId(o, h.HWSprId) // из vmdatastorename.go, тот же пакет cmdb
		if code == "" || seen[code] {
			continue //не удалось определить код ЦОД либо уже встречался (без дублей)
		}
		seen[code] = true
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		return
	}
	sort.Strings(codes) // стабильный порядок склейки - не порождать лишних обновлений при неизменном составе
	datastoreName := strings.Join(codes, ";")

	clusterSpr := models.ClusterSprRecords{Id: clusterSprId}
	if err := o.Read(&clusterSpr, "Id"); err == nil && clusterSpr.DatastoreName == datastoreName {
		return //уже актуально
	}
	if status := grpcclient.UpdateClusterSprDatastoreName(clusterSprId, datastoreName); status != 1 {
		log.Println("cmdb.UpdateClusterDatastoreNames: ошибка обновления DatastoreName для ClusterSprId=", clusterSprId, "статус:", status)
	}
}
