// Package grpcserver реализует gRPC-серверную часть системы мониторинга
// loadmon: два gRPC-сервиса, Loadmon (сервер server, инвентарь и метрики
// VMware vSphere, оборудования HPE OneView и т.п. — контракт см. в
// protocol/*.proto) и LoadmonDB (сервер serverdb, доступ к данным,
// сохранённым в БД проекта).
//
// Настоящий файл содержит обработчик UpdateVMSprDatacenterName - точечное
// обновление поля VMSprRecord.DatacenterName (код ЦОД DatacenterSite.CodeJira,
// определяемый через физический ESXi хост ВМ). Вызывается отдельной функцией
// cmdb.UpdateVMDatacenterNames (файл cmdb/vmdatacentername.go), запускаемой
// параллельно с сохранением метрик ВМ (cmdb.SaveVMRecord) - см. комментарий
// у rpc UpdateVMSprDatacenterName в protocol/db.proto.
//
// ПЕРЕИМЕНОВАНО (2026-09-30) из UpdateVMSprDatastoreName/DatastoreName -
// опечатка, поле не имеет отношения к датасторам vSphere, только к ЦОД.
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

const errUpdateVMSprDatacenterName = "Ошибка обновления DatacenterName в справочнике виртуальных машин"

// UpdateVMSprDatacenterName обновляет поле DatacenterName записи справочника
// виртуальных машин (vm_spr_record) по Id. Возвращает Status=1 при успехе,
// Status=2 при ошибке чтения/обновления записи.
func (p *serverdb) UpdateVMSprDatacenterName(_ context.Context, in *pb.VMSprDatacenterNameRequest) (*pb.ClusterSprId, error) {
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
			id, err := c.UpdateVMSprDatacenterName(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterSprId{}, errors.New(errUpdateVMSprDatacenterName) //ничего не найдено
	}

	res := pb.ClusterSprId{Id: in.Id}
	o := orm.NewOrm()
	vmSpr := models.VMSprRecord{Id: in.Id}
	if err := o.Read(&vmSpr, "Id"); err != nil {
		res.Status = 2
		return &res, errors.New(errUpdateVMSprDatacenterName)
	}
	if vmSpr.DatacenterName == in.DatacenterName { //уже актуально, обновление не требуется
		res.Status = 1
		return &res, nil
	}
	vmSpr.DatacenterName = in.DatacenterName
	if _, err := o.Update(&vmSpr, "DatacenterName"); err != nil {
		res.Status = 2
		return &res, errors.New(errUpdateVMSprDatacenterName)
	}
	res.Status = 1
	return &res, nil
}
