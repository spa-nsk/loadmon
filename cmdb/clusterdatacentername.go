// Package cmdb (файл clusterdatacentername.go) содержит отдельную функцию
// UpdateClusterDatacenterNames, заполняющую ClusterSprRecords.DatacenterName -
// объединённый через ";" (без дублей) список кодов ЦОД
// (DatacenterSite.CodeJira) всех ESXi гипервизоров кластера:
//
//	ServerSprRecords (Type=2, Vcenter+ClusterId == параметры кластера) ->
//	HWSprId -> ServerHardware.RackCabinetId -> RackCabinet.DatacenterSiteId
//	-> DatacenterSite.CodeJira
//
// По аналогии с VMSprRecord.DatacenterName (см. vmdatacentername.go) эта
// функция НЕ встроена в код сохранения метрик, а вызывается ОТДЕЛЬНО и
// ЗАПУСКАЕТСЯ ПАРАЛЛЕЛЬНО с ним - см. вызов `go UpdateClusterDatacenterNames
// (...)` рядом с SaveVMRecord/UpdateVMDatacenterNames в cmdb.SaveClusterData
// (savemetric_orchestration.go). Обновление записи в БД выполняется через
// отдельный gRPC-метод LoadmonDB.UpdateClusterSprDatacenterName (см.
// protocol/db.proto, grpcserver/handler_db_clusterspr_datacentername.go,
// grpcclient.UpdateClusterSprDatacenterName) - по тому же loopback-паттерну,
// что и UpdateVMSprDatacenterName/AddClusterSprRecord/UpdateClusterSprRecord.
//
// ПЕРЕИМЕНОВАНО (2026-09-30) из UpdateClusterDatastoreNames/DatastoreName
// (файл clusterdatastorename.go) - опечатка, поле не имеет отношения к
// датасторам vSphere, только к ЦОД.
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

// UpdateClusterDatacenterNames заполняет DatacenterName записи справочника
// кластеров (cluster_spr_records, Id=clusterSprId) объединённым списком
// кодов ЦОД всех ESXi гипервизоров кластера cl. Рассчитана на запуск как
// отдельная горутина параллельно с cmdb.SaveVMRecord/UpdateVMDatacenterNames
// для того же кластера (см. комментарий к пакету выше).
func UpdateClusterDatacenterNames(cl *vsphere.Cluster, clusterSprId int64) {
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
		code := resolveDatacenterCodeByHWSprId(o, h.HWSprId) // из vmdatacentername.go, тот же пакет cmdb
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
	datacenterName := strings.Join(codes, ";")

	clusterSpr := models.ClusterSprRecords{Id: clusterSprId}
	if err := o.Read(&clusterSpr, "Id"); err == nil && clusterSpr.DatacenterName == datacenterName {
		return //уже актуально
	}
	if status := grpcclient.UpdateClusterSprDatacenterName(clusterSprId, datacenterName); status != 1 {
		log.Println("cmdb.UpdateClusterDatacenterNames: ошибка обновления DatacenterName для ClusterSprId=", clusterSprId, "статус:", status)
	}
}
