// Файл содержит обработчик AddClusterRecord сервиса LoadmonDB - добавление
// ежедневной исторической записи кластера (cluster_record).
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

const errAddClusterRecord = "Ошибка вставки записи в таблицу кластеров (cluster_record)"

// AddClusterRecord добавляет новую запись ClusterRecord (снимок состояния и
// агрегированных метрик кластера за период) и возвращает присвоенный
// идентификатор: Status=1 при успехе, Status=2 при ошибке вставки. Id во
// входном сообщении игнорируется - запись всегда создаётся новая. Если
// сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос
// ретранслируется на первый из них, вернувший успех (сервис LoadmonDB, адрес
// GetSrvDBAddr).
func (p *serverdb) AddClusterRecord(_ context.Context, in *pb.ClusterRecord) (*pb.ClusterRecordId, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ClusterRecordId{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			id, err := c.AddClusterRecord(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterRecordId{Status: 2}, errors.New(errAddClusterRecord) //ничего не найдено
	}

	res := pb.ClusterRecordId{}
	rec := models.ClusterRecord{}
	MessageToClusterRecord(in, &rec)
	rec.Id = 0 //всегда новая запись
	o := orm.NewOrm()
	id, err := o.Insert(&rec)
	if err != nil {
		log.Printf("AddClusterRecord: %v\n", err)
		res.Status = 2
		return &res, errors.New(errAddClusterRecord)
	}
	res.Id = id
	res.Status = 1
	return &res, nil
}
