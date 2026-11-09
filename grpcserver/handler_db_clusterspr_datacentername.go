// Package grpcserver реализует gRPC-серверную часть системы мониторинга
// loadmon: два gRPC-сервиса, Loadmon (сервер server, инвентарь и метрики
// VMware vSphere, оборудования HPE OneView и т.п. — контракт см. в
// protocol/*.proto) и LoadmonDB (сервер serverdb, доступ к данным,
// сохранённым в БД проекта).
//
// Настоящий файл содержит обработчик UpdateClusterSprDatacenterName - точечное
// обновление поля ClusterSprRecord.DatacenterName (объединённый через ";" без
// дублей список кодов ЦОД DatacenterSite.CodeJira всех ESXi гипервизоров
// кластера). Вызывается отдельной функцией cmdb.UpdateClusterDatacenterNames
// (файл cmdb/clusterdatacentername.go), запускаемой параллельно с сохранением
// метрик кластера (cmdb.SaveClusterData) - по аналогии с
// UpdateVMSprDatacenterName/cmdb.UpdateVMDatacenterNames (см.
// handler_db_vmspr.go).
//
// ПЕРЕИМЕНОВАНО (2026-09-30) из UpdateClusterSprDatastoreName/DatastoreName
// (файл handler_db_clusterspr_datastorename.go) - опечатка, поле не имеет
// отношения к датасторам vSphere, только к ЦОД.
package grpcserver

import (
	"context"
	"errors"
	"loadmon/config"
	"loadmon/models"
	pb "loadmon/protocol"
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const errUpdateClusterSprDatacenterName = "Ошибка обновления DatacenterName в справочнике кластеров"

// UpdateClusterSprDatacenterName обновляет поле DatacenterName записи
// справочника кластеров (cluster_spr_records) по Id. Возвращает Status=1 при
// успехе, Status=2 при ошибке чтения/обновления записи.
func (p *serverdb) UpdateClusterSprDatacenterName(_ context.Context, in *pb.ClusterSprDatacenterNameRequest) (*pb.ClusterSprId, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ClusterSprId{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			id, err := c.UpdateClusterSprDatacenterName(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterSprId{}, errors.New(errUpdateClusterSprDatacenterName) //ничего не найдено
	}

	res := pb.ClusterSprId{Id: in.Id}
	o := orm.NewOrm()
	clusterSpr := models.ClusterSprRecords{Id: in.Id}
	if err := o.Read(&clusterSpr, "Id"); err != nil {
		res.Status = 2
		return &res, errors.New(errUpdateClusterSprDatacenterName)
	}
	if clusterSpr.DatacenterName == in.DatacenterName { //уже актуально, обновление не требуется
		res.Status = 1
		return &res, nil
	}
	clusterSpr.DatacenterName = in.DatacenterName
	if _, err := o.Update(&clusterSpr, "DatacenterName"); err != nil {
		res.Status = 2
		return &res, errors.New(errUpdateClusterSprDatacenterName)
	}
	res.Status = 1
	return &res, nil
}
