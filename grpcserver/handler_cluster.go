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
	"loadmon/protocol"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func (s *server) GetCluster(in *pb.CLReq, stream grpc.ServerStreamingServer[pb.ClusterShort]) error {
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
			streamcl, err := c.GetCluster(ctx, in)
			if err != nil {
				return err
			}
			for {
				pbcl, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				clsh := pb.ClusterShort{
					Vcenter:    pbcl.Vcenter,
					Datacenter: pbcl.Datacenter,
					Name:       pbcl.Name,
				}
				if err := stream.Send(&clsh); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if in.Vcenter != "all" && cl.Vcenter != in.Vcenter {
			continue
		}
		if in.Datacenter != "all" && cl.Datacenter != in.Datacenter {
			continue
		}
		out := pb.ClusterShort{Vcenter: cl.Vcenter, Datacenter: cl.Datacenter, Name: cl.Name}
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

// GetClusterFull отдаёт клиенту потоком полную информацию о кластерах (включая хосты и хранилища), отфильтрованную по vCenter/датацентру/имени кластера. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetClusterFull(in *pb.CLReq, stream grpc.ServerStreamingServer[pb.ClusterFull]) error {
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
			streamcl, err := c.GetClusterFull(ctx, in)
			if err != nil {
				return err
			}
			for {
				pbcl, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(CopyClusterFull(pbcl)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if in.Vcenter != "all" && cl.Vcenter != in.Vcenter {
			continue
		}
		if in.Datacenter != "all" && cl.Datacenter != in.Datacenter {
			continue
		}
		if in.Cluster != "all" && cl.Name != in.Cluster {
			continue
		}
		out := pb.ClusterFull{}
		ClusterToMessageFull(*cl, &out)
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

// GetClusterStorages отдаёт клиенту потоком хранилища (datastore), принадлежащие выбранным кластерам. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetClusterStorages(in *pb.CLReq, stream grpc.ServerStreamingServer[pb.Datastore]) error {
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
			streamcl, err := c.GetClusterStorages(ctx, in)
			if err != nil {
				return err
			}
			for {
				pbcl, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(CopyClusterStorages(pbcl)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if in.Vcenter != "all" && cl.Vcenter != in.Vcenter {
			continue
		}
		if in.Datacenter != "all" && cl.Datacenter != in.Datacenter {
			continue
		}
		if in.Cluster != "all" && cl.Name != in.Cluster {
			continue
		}
		out := pb.Datastore{}
		for _, st := range cl.Storage {
			ClusterStorageToMessage(st, &out)
			if err := stream.Send(&out); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetClusterHost отдаёт клиенту потоком хосты, принадлежащие выбранным кластерам, с пересчитанными агрегированными метриками. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetClusterHost(in *pb.CLReq, stream grpc.ServerStreamingServer[pb.Host]) error {
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
			streamcl, err := c.GetClusterHost(ctx, in)
			if err != nil {
				return err
			}
			for {
				pbhs, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(CopyHost(pbhs)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if in.Vcenter != "all" && cl.Vcenter != in.Vcenter {
			continue
		}
		if in.Datacenter != "all" && cl.Datacenter != in.Datacenter {
			continue
		}
		if in.Cluster != "all" && cl.Name != in.Cluster {
			continue
		}
		out := pb.Host{}
		for _, hs := range cl.Hosts {
			RecalcHostToMessageShort(hs, &out)
			if err := stream.Send(&out); err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyHost создаёт независимую копию сообщения pb.Host, чтобы избежать совместного использования указателей при пересылке данных от удалённого gRPC-эндпоинта клиенту. При hs == nil возвращает пустую структуру.
func (s *server) GetClusterMetrics(in *pb.CLReq, stream grpc.ServerStreamingServer[pb.CLPerf]) error {
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
			streamcl, err := c.GetClusterMetrics(ctx, in)
			if err != nil {
				return err
			}
			for {
				pbcl, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(CopyCLPerf(pbcl)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := fmt.Sprintf("%s/%s/%s", in.Vcenter, in.Datacenter, in.Cluster)
	cl, ok := infra.ClusterMapList.Get(key)
	if !ok {
		return errors.New("Кластер не найден")
	}
	cl.CalcPerfomance()
	for _, perf := range cl.Perf {
		out := pb.CLPerf{}
		ClusterMetricToMessage(perf, &out)
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) GetClusterMedium(in *pb.CLReq, stream grpc.ServerStreamingServer[pb.Cluster]) error {
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
			streamcl, err := c.GetClusterMedium(ctx, in)
			if err != nil {
				return err
			}
			for {
				pbcl, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(CopyClusterMedium(pbcl)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := fmt.Sprintf("%s/%s/%s", in.Vcenter, in.Datacenter, in.Cluster)
	cl, ok := infra.ClusterMapList.Get(key)
	if !ok {
		return errors.New("Кластер не найден")
	}
	out := pb.Cluster{}
	ClusterToMessageMedium(*cl, &out)
	if err := stream.Send(&out); err != nil {
		return err
	}

	return nil
}

// CopyClusterMedium создаёт независимую копию сообщения pb.Cluster, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту.
func (s *server) GetClusterListKey(_ context.Context, in *pb.CLReqListKey) (*pb.CLListKey, error) {
	if config.Switch.Endpoint.Count() > 0 {
		clall := pb.CLListKey{KeyRecs: make([]*pb.CLKeyRec, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &clall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetClusterListKey(ctx, in)
			if err == nil {
				for _, rec := range vms.KeyRecs {
					clall.KeyRecs = append(clall.KeyRecs, rec)
					clall.Count++
				}
			}
		}
		return &clall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	vcenter := in.GetVcenter()
	datacenter := in.GetDatacenter()
	cluster := in.GetCluster()
	out := pb.CLListKey{}

	if infra == nil {
		return &out, errors.New(errNotInitialized)
	}

	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}

		if vcenter != "all" && vcenter != "" && cl.Vcenter != vcenter {
			continue
		}
		if datacenter != "all" && datacenter != "" && cl.Datacenter != datacenter {
			continue
		}
		if cluster != "all" && cluster != "" && cl.Name != cluster {
			continue
		}

		out.KeyRecs = append(out.KeyRecs, &pb.CLKeyRec{CLName: cl.Name, Key: key})
		out.Count++
	}
	return &out, nil
}

// GetHostPerf отдаёт клиенту потоком историю метрик производительности хоста в указанном диапазоне времени. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetClusterPerfByKey(_ context.Context, in *pb.CLReqKey) (*pb.CLPerfSlice, error) {
	if config.Switch.Endpoint.Count() > 0 {
		clperfs := pb.CLPerfSlice{Perf: make([]*pb.CLPerf, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &clperfs, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			perfSlice, err := c.GetClusterPerfByKey(ctx, in)
			if err == nil {
				for _, perf := range perfSlice.Perf {
					clperfs.Perf = append(clperfs.Perf, perf)
					clperfs.Count++
				}
			}
		}
		return &clperfs, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	cl, ok := infra.ClusterMapList.Get(key)
	out := pb.CLPerfSlice{}
	if !ok {
		return &out, errors.New("Host не найден")
	}
	cl.CalcPerfomance()
	for _, p := range cl.Perf {
		perf := protocol.CLPerf{}
		perf.Id = p.ID
		perf.Start = timestamppb.New(p.Start)
		perf.CpuUsage = p.CPUUsage
		perf.CpuUtilization = p.CPUUtilization
		perf.MemUsage = p.MemUsage
		perf.DiskUsage = p.DiskUsage
		perf.DiskRead = p.DiskRead
		perf.DiskWrite = p.DiskWrite
		perf.NetUsage = p.NetUsage
		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		out.Perf = append(out.Perf, &perf)
	}
	return &out, nil
}

// GetHostByKey возвращает краткую информацию о хосте по его ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetClusterByKey(_ context.Context, in *pb.CLReqKey) (*pb.Cluster, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.Cluster{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cl, err := c.GetClusterByKey(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return cl, nil
			}
		}
		return &pb.Cluster{}, errors.New(errNotFound) //ничего не найдено
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	cl, ok := infra.ClusterMapList.Get(key)
	if !ok {
		return &pb.Cluster{}, errors.New("Cluster не найден")
	}
	out := pb.Cluster{}
	ClusterToMessage(*cl, &out)
	return &out, nil
}

// GetDSData возвращает информацию о хранилище (datastore) по его ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
