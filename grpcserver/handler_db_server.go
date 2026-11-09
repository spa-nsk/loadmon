// Файл содержит обработчик GetServer сервиса LoadmonDB - получение записи
// справочника серверов (server_spr_records) по ключу.
package grpcserver

import (
	"context"
	"errors"
	"loadmon/config"
	"loadmon/models"
	pb "loadmon/protocol"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const errServerSprNotFound = "Запись справочника серверов не найдена"

// GetServer возвращает активную (Stat = models.ServerSprStatUsed) запись
// справочника серверов по FQDN: DBKeyString.Key сравнивается с полем
// NameFQDN (без учёта регистра, как принято для DNS-имён). Если подходящих
// записей несколько, возвращается самая новая (с наибольшим Id).
// Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос
// ретранслируется на каждый из них (сервис LoadmonDB, адрес GetSrvDBAddr), ответ
// первого успешного возвращается как есть. Пустой ключ и отсутствие записи -
// ошибка.
func (p *serverdb) GetServer(_ context.Context, in *pb.DBKeyString) (*pb.ServerSprRecords, error) {
	key := strings.TrimSpace(in.GetKey())
	if key == "" {
		return &pb.ServerSprRecords{}, errors.New("не указан FQDN сервера")
	}
	if config.Switch.Endpoint.Count() > 0 {
		for _, ekey := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(ekey)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ServerSprRecords{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			srv, err := c.GetServer(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return srv, nil
			}
		}
		return &pb.ServerSprRecords{}, errors.New(errServerSprNotFound) //ничего не найдено
	}

	o := orm.NewOrm()
	cond := orm.NewCondition().And("Stat", models.ServerSprStatUsed).And("NameFQDN__iexact", key)
	qs := o.QueryTable("server_spr_records").SetCond(cond)
	records := make([]*models.ServerSprRecords, 0)
	if _, err := qs.OrderBy("-Id").Limit(1).All(&records); err != nil || len(records) == 0 {
		return &pb.ServerSprRecords{}, errors.New(errServerSprNotFound)
	}
	out := pb.ServerSprRecords{}
	ServerSprRecodToMessage(records[0], &out)
	return &out, nil
}
