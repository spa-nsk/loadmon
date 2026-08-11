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
	"io"
	"loadmon/config"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (s *server) GetVCenters(in *pb.VCReq, stream grpc.ServerStreamingServer[pb.VCenter]) error {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			streamcl, err := c.GetVCenters(ctx, &pb.VCReq{Filter: in.Filter})
			if err != nil {
				return err
			}
			for {
				pbvc, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(pbvc); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		out := pb.VCenter{Hostname: vc.Hostname, Connected: vc.Connected}
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

// GetDatacenter отдаёт клиенту потоком список датацентров (уникальных по имени в рамках выбранного vCenter). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetDatacenter(in *pb.DCReq, stream grpc.ServerStreamingServer[pb.Datacenter]) error {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			streamcl, err := c.GetDatacenter(ctx, &pb.DCReq{Vcenter: in.Vcenter})
			if err == nil {
				for {
					pbdc, err := streamcl.Recv()
					if err == io.EOF {
						break
					}
					if err := stream.Send(pbdc); err != nil {
						return err
					}
				}
			} else {
				return err
			}
		}
		return nil
	}
	dcm := make(map[string]*string)
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if in.Vcenter != "all" && cl.Vcenter != in.Vcenter {
			continue
		}
		if _, ok := dcm[cl.Datacenter]; ok {
			continue
		}
		dcm[cl.Datacenter] = nil
		out := pb.Datacenter{VCenter: cl.Vcenter, Name: cl.Datacenter}
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

// GetCluster отдаёт клиенту потоком краткую информацию о кластерах, отфильтрованную по vCenter/датацентру. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
