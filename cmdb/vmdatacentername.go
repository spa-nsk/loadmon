// Package cmdb (файл vmdatacentername.go) содержит отдельную функцию
// UpdateVMDatacenterNames, заполняющую VMSprRecord.DatacenterName - код ЦОД
// (DatacenterSite.CodeJira), определяемый через физический ESXi хост
// виртуальной машины (VMSprRecord.Vm_EsxHost) и его размещение в стойке
// на момент записи метрики:
//
//	Vm_EsxHost -> ServerSprRecords (Type=2, сравнение по Name/NameFQDN,
//	регистронезависимо, по аналогии с controllers/sprsrvview2.go) ->
//	HWSprId -> ServerHardware.RackCabinetId -> RackCabinet.DatacenterSiteId
//	-> DatacenterSite.CodeJira
//
// По прямому указанию пользователя эта функция НЕ встроена в
// cmdb.SaveVMRecord (сохранение метрик ВМ), а вызывается ОТДЕЛЬНО и
// ЗАПУСКАЕТСЯ ПАРАЛЛЕЛЬНО с ним - см. вызов `go UpdateVMDatacenterNames(...)`
// рядом с SaveVMRecord(...) в cmdb.SaveClusterData
// (savemetric_orchestration.go). Обновление записи в БД выполняется через
// отдельный gRPC-метод LoadmonDB.UpdateVMSprDatacenterName (см.
// protocol/db.proto, grpcserver/handler_db_vmspr.go,
// grpcclient.UpdateVMSprDatacenterName) - по аналогии с тем, как
// AddClusterSprRecord/UpdateClusterSprRecord используют loopback-вызов
// gRPC внутри процесса metrics_grpc для записи справочных данных.
//
// ПЕРЕИМЕНОВАНО (2026-09-30) из UpdateVMDatastoreNames/DatastoreName (файл
// vmdatastorename.go) - опечатка, поле не имеет отношения к датасторам
// vSphere, только к ЦОД.
package cmdb

import (
	"loadmon/grpcclient"
	"loadmon/models"
	vsphere "loadmon/vspheremetrics"
	"log"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// UpdateVMDatacenterNames заполняет DatacenterName у всех записей справочника
// виртуальных машин (vm_spr_record) кластера cl, определяя код ЦОД через
// текущий физический ESXi хост каждой ВМ. Рассчитана на запуск как отдельная
// горутина параллельно с cmdb.SaveVMRecord для того же кластера (см.
// комментарий к пакету выше) - НЕ вызывается из SaveVMRecord напрямую.
func UpdateVMDatacenterNames(cl *vsphere.Cluster) {
	if cl == nil {
		return
	}
	o := orm.NewOrm()

	var vmSprList []*models.VMSprRecord
	num, err := o.QueryTable(new(models.VMSprRecord)).
		Filter("Vcenter", cl.Vcenter).
		Filter("ClusterId", cl.ClusterID).
		All(&vmSprList, "Id", "Vm_EsxHost", "DatacenterName")
	if err != nil || num == 0 {
		return //нет ВМ по этому кластеру в справочнике либо ошибка чтения - ничего не делаем
	}

	// карта имя_хоста(lower, trim) -> ServerHardware.Id (HWSprId) по
	// гипервизорам (Type=2) этого vcenter - сравнение регистронезависимо по
	// Name/NameFQDN, по аналогии с controllers/sprsrvview2.go (фильтр
	// размещения VM по хосту).
	var hosts []models.ServerSprRecords
	models.ServerSprQS(o).Filter("Type", 2).Filter("Vcenter", cl.Vcenter).
		All(&hosts, "Id", "Name", "NameFQDN", "HWSprId")
	if len(hosts) == 0 {
		return //нет данных о гипервизорах этого vcenter - резолвить не из чего
	}
	hostHWSprId := make(map[string]int64, len(hosts)*2)
	for _, h := range hosts {
		if h.HWSprId == 0 {
			continue
		}
		if n := strings.ToLower(strings.TrimSpace(h.Name)); n != "" {
			hostHWSprId[n] = h.HWSprId
		}
		if n := strings.ToLower(strings.TrimSpace(h.NameFQDN)); n != "" {
			hostHWSprId[n] = h.HWSprId
		}
	}
	if len(hostHWSprId) == 0 {
		return
	}

	codeCache := make(map[int64]string) // HWSprId -> DatacenterSite.CodeJira, чтобы не резолвить повторно ВМ на одном хосте
	for _, vmSpr := range vmSprList {
		if vmSpr == nil || vmSpr.Vm_EsxHost == "" {
			continue
		}
		hwSprId, ok := hostHWSprId[strings.ToLower(strings.TrimSpace(vmSpr.Vm_EsxHost))]
		if !ok {
			continue //хост ВМ не найден среди гипервизоров (Type=2) - резолвить нечем
		}
		code, ok := codeCache[hwSprId]
		if !ok {
			code = resolveDatacenterCodeByHWSprId(o, hwSprId)
			codeCache[hwSprId] = code
		}
		if code == "" || code == vmSpr.DatacenterName {
			continue //не удалось определить код ЦОД либо значение уже актуально
		}
		if status := grpcclient.UpdateVMSprDatacenterName(vmSpr.Id, code); status != 1 {
			log.Println("cmdb.UpdateVMDatacenterNames: ошибка обновления DatacenterName для VMSprId=", vmSpr.Id, "статус:", status)
		}
	}
}

// resolveDatacenterCodeByHWSprId резолвит код ЦОД (DatacenterSite.CodeJira)
// по Id записи ServerHardware (HWSprId гипервизора):
// ServerHardware.RackCabinetId -> RackCabinet.DatacenterSiteId ->
// DatacenterSite.CodeJira. Возвращает "", если цепочка не резолвится
// (шкаф/площадка не заполнены).
func resolveDatacenterCodeByHWSprId(o orm.Ormer, hwSprId int64) string {
	hw := models.ServerHardware{Id: hwSprId}
	if err := o.Read(&hw, "Id"); err != nil || hw.RackCabinetId == 0 {
		return ""
	}
	rack := models.RackCabinet{Id: hw.RackCabinetId}
	if err := o.Read(&rack, "Id"); err != nil || rack.DatacenterSiteId == 0 {
		return ""
	}
	site := models.DatacenterSite{Id: rack.DatacenterSiteId}
	if err := o.Read(&site, "Id"); err != nil {
		return ""
	}
	return site.CodeJira
}
