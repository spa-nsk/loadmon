// Package grpcserver реализует gRPC-серверную часть системы мониторинга
// loadmon: два gRPC-сервиса, Loadmon (сервер server, инвентарь и метрики
// VMware vSphere, оборудования HPE OneView и т.п. — контракт см. в
// protocol/*.proto) и LoadmonDB (сервер serverdb, доступ к данным,
// сохранённым в БД проекта).
//
// Настоящий файл содержит обработчик UpdateVMSprDatastoreName - точечное
// обновление поля VMSprRecord.DatastoreName (код ЦОД DatacenterSite.CodeJira,
// определяемый через физический ESXi хост ВМ). Вызывается отдельной функцией
// cmdb.UpdateVMDatastoreNames (файл cmdb/vmdatastorename.go), запускаемой
// параллельно с сохранением метрик ВМ (cmdb.SaveVMRecord) - см. комментарий
// у rpc UpdateVMSprDatastoreName в protocol/db.proto.
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

const errUpdateVMSprDatastoreName = "Ошибка обновления DatastoreName в справочнике виртуальных машин"

// UpdateVMSprDatastoreName обновляет поле DatastoreName записи справочника
// виртуальных машин (vm_spr_record) по Id. Возвращает Status=1 при успехе,
// Status=2 при ошибке чтения/обновления записи.
func (p *serverdb) UpdateVMSprDatastoreName(_ context.Context, in *pb.VMSprDatastoreNameRequest) (*pb.ClusterSprId, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ClusterSprId{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			id, err := c.UpdateVMSprDatastoreName(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterSprId{}, errors.New(errUpdateVMSprDatastoreName) //ничего не найдено
	}

	res := pb.ClusterSprId{Id: in.Id}
	o := orm.NewOrm()
	vmSpr := models.VMSprRecord{Id: in.Id}
	if err := o.Read(&vmSpr, "Id"); err != nil {
		res.Status = 2
		return &res, errors.New(errUpdateVMSprDatastoreName)
	}
	if vmSpr.DatastoreName == in.DatastoreName { //уже актуально, обновление не требуется
		res.Status = 1
		return &res, nil
	}
	vmSpr.DatastoreName = in.DatastoreName
	if _, err := o.Update(&vmSpr, "DatastoreName"); err != nil {
		res.Status = 2
		return &res, errors.New(errUpdateVMSprDatastoreName)
	}
	res.Status = 1
	return &res, nil
}
