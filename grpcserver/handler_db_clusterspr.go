// Package grpcserver реализует gRPC-серверную часть системы мониторинга
// loadmon: два gRPC-сервиса, Loadmon (сервер server, инвентарь и метрики
// VMware vSphere, оборудования HPE OneView и т.п. — контракт см. в
// protocol/*.proto) и LoadmonDB (сервер serverdb, доступ к данным,
// сохранённым в БД проекта). Сервисы принимают запросы, полученные клиентом
// пакета grpcclient, выполняют выборку/агрегацию данных из внутренних
// пакетов проекта (vspheremetrics, oneview, models и др.) и упаковывают
// результат в сгенерированные gRPC-сообщения пакета protocol. Файл
// serverfunc.go содержит запуск серверов и вспомогательные функции, файл
// message_srv.go — функции-конвертеры доменных моделей в gRPC-сообщения.
package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"loadmon/config"
	"loadmon/models"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (s *serverdb) GetClusterSprRecords(in *pb.ClusterSprKey1, stream grpc.ServerStreamingServer[pb.ClusterSprRecord]) error {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			streamhw, err := c.GetClusterSprRecords(ctx, in)
			if err == nil {
				for {
					hw, err := streamhw.Recv()
					if err == io.EOF {
						break
					}
					if err := stream.Send(hw); err != nil {
						return err
					}
				}
				return nil //Как только выполняется запрос прекращаем отправку остальным
			}
		}
		return nil
	}

	o := orm.NewOrm()

	qs := o.QueryTable("cluster_spr_records")
	qs = qs.OrderBy("Vcenter", "Datacenter", "Name")
	cond := orm.NewCondition()

	if in.Vcenter != "all" {
		cond = cond.And("Vcenter", in.Vcenter)
	}
	if in.Datacenter != "all" {
		cond = cond.And("Datacenter", in.Datacenter)
	}

	if in.Cluster != "all" {
		cond = cond.And("name", in.GetCluster())
	}

	qs = qs.SetCond(cond)
	clusters := make([]*models.ClusterSprRecords, 0)

	num, err := qs.All(&clusters)
	infra := vsphere.GetGlobalVMInfrastructure()
	if num > 0 && err == nil {
		for i := range clusters {
			clspr := models.ClusterSprRecords{Id: clusters[i].Id}
			err := o.Read(&clspr, "Id")
			if clspr.Stat != 2 {
				clspr.Stat = 0
			}
			key := fmt.Sprintf("%s/%s/%s", clusters[i].Vcenter, clusters[i].Datacenter, clusters[i].Name)
			cl, ok := infra.ClusterMapList.Get(key)
			if ok {
				clusters[i].TypeWork = cl.TypeWork
				clusters[i].Stat = 1
				clspr.TypeWork = cl.TypeWork
				clspr.Stat = 1
			}
			if err == nil {
				o.Update(&clspr)
			}
			out := pb.ClusterSprRecord{}
			ClusterSprRecordsToMessage(clusters[i], &out)
			if err := stream.Send(&out); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddClusterSprRecord добавляет новую запись в справочник кластеров БД и возвращает присвоенный идентификатор.
// Возвращает ошибку errAddClusterSpr, если вставка записи не удалась.
func (p *serverdb) AddClusterSprRecord(_ context.Context, in *pb.ClusterSprRecord) (*pb.ClusterSprId, error) {
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
			id, err := c.AddClusterSprRecord(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterSprId{}, errors.New(errAddClusterSpr) //ничего не найдено
	}
	var err error
	clusterSprId := pb.ClusterSprId{}
	clusterSpr := models.ClusterSprRecords{}
	o := orm.NewOrm()
	MessageToClusterSprRecords(in, &clusterSpr)
	if clusterSprId.Id, err = o.Insert(&clusterSpr); err == nil {
		clusterSprId.Status = 1
		return &clusterSprId, nil
	}
	clusterSprId.Status = 2
	return &clusterSprId, errors.New(errAddClusterSpr)
}

// UpdateClusterSprRecord обновляет существующую запись в справочнике кластеров БД и возвращает её идентификатор.
func (p *serverdb) UpdateClusterSprRecord(_ context.Context, in *pb.ClusterSprRecord) (*pb.ClusterSprId, error) {
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
			id, err := c.UpdateClusterSprRecord(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterSprId{}, errors.New(errAddClusterSpr) //ничего не найдено
	}
	var err error
	clusterSprId := pb.ClusterSprId{}
	clusterSpr := models.ClusterSprRecords{}
	o := orm.NewOrm()
	clusterSpr.Vcenter = in.Vcenter
	clusterSpr.Datacenter = in.Datacenter
	clusterSpr.Name = in.Name
	err = o.Read(&clusterSpr, "Vcenter", "Datacenter", "Name")
	clusterSprId = pb.ClusterSprId{}
	if err == nil {
		clusterSprId.Id = clusterSpr.Id
		MessageToClusterSprRecords(in, &clusterSpr)
		if _, err = o.Update(&clusterSpr); err == nil {
			clusterSprId.Status = 1
			return &clusterSprId, nil
		}
	}
	clusterSprId.Status = 2
	return &clusterSprId, errors.New(errAddClusterSpr)
}

// для всех
