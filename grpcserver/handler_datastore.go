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
	"io"
	"loadmon/config"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (p *server) GetDSData(_ context.Context, in *pb.KeyString) (*pb.Datastore, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.Datastore{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cl, err := c.GetDSData(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return cl, nil
			}
		}
		return &pb.Datastore{}, errors.New(errNotFound) //ничего не найдено
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	ds, ok := infra.DataStoreMapList.Get(in.GetKey())
	if ok {
		s := pb.Datastore{}
		s.Id = ds.ID
		s.UID = ds.UID
		s.StorageName = ds.StorageName
		s.CapacityGb = ds.CapacityGb
		s.FreeSpaceGb = ds.FreeSpaceGb
		s.PercentUsed = ds.PercentUsed
		s.VmKeys = &pb.VMListKey{}
		for _, vm := range ds.VM {
			s.VmKeys.Count++
			key := pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName}
			s.VmKeys.KeyRecs = append(s.VmKeys.KeyRecs, &key)
		}
		return &s, nil
	}
	return nil, errors.New("Storage не найден")
}

// GetAllDSData отдаёт клиенту потоком все известные хранилища (datastore). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (p *server) GetAllDSData(in *pb.KeyString, stream grpc.ServerStreamingServer[pb.Datastore]) error {
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
			streamperf, err := c.GetAllDSData(ctx, in)
			if err == nil {
				for {
					ds, err := streamperf.Recv()
					if err == io.EOF {
						break
					}
					if err := stream.Send(ds); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	for _, key := range infra.DataStoreMapList.Keys() {
		ds, ok := infra.DataStoreMapList.Get(key)
		if !ok {
			continue
		}
		s := pb.Datastore{}
		s.Id = ds.ID
		s.UID = ds.UID
		s.StorageName = ds.StorageName
		s.CapacityGb = ds.CapacityGb
		s.FreeSpaceGb = ds.FreeSpaceGb
		s.PercentUsed = ds.PercentUsed
		s.VmKeys = &pb.VMListKey{}
		for _, vm := range ds.VM {
			s.VmKeys.Count++
			key := pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName}
			s.VmKeys.KeyRecs = append(s.VmKeys.KeyRecs, &key)
		}
		if err := stream.Send(&s); err != nil {
			_ = err
		}
	}
	return nil
}

// compact возвращает подсрез временных меток src, попадающих в полуоткрытый диапазон [start, end).
// Используется для обрезки истории метрик перед возвратом клиенту либо перед очисткой.
