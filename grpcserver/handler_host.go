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

func (s *server) GetHostFull(in *pb.HSReq, stream grpc.ServerStreamingServer[pb.HostFull]) error {
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
			streamcl, err := c.GetHostFull(ctx, in)
			if err != nil {
				return err
			}
			for {
				pbcl, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(CopyPbHostFull(pbcl)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		if in.Vcenter != "all" && hs.Vcenter != in.Vcenter {
			continue
		}
		if in.Datacenter != "all" && hs.Datacenter != in.Datacenter {
			continue
		}
		if in.Cluster != "all" && hs.Cluster != in.Cluster {
			continue
		}
		if in.HSName != "all" && hs.Name != in.HSName {
			continue
		}

		out := pb.HostFull{}
		RecalcHostToMessageFull(hs, &out)
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

// CopyPbHostFull создаёт независимую копию сообщения pb.HostFull, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту.
func (s *server) GetHostListKey(_ context.Context, in *pb.HSReqListKey) (*pb.HSListKey, error) {
	if config.Switch.Endpoint.Count() > 0 {
		hsall := pb.HSListKey{KeyRecs: make([]*pb.HSKeyRec, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}

			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &hsall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetHostListKey(ctx, in)
			if err == nil {
				for _, rec := range vms.KeyRecs {
					hsall.KeyRecs = append(hsall.KeyRecs, rec)
					hsall.Count++
				}
			}
		}
		return &hsall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	vcenter := in.GetVcenter()
	datacenter := in.GetDatacenter()
	cluster := in.GetCluster()
	out := pb.HSListKey{}

	if infra == nil {
		return &out, errors.New(errNotInitialized)
	}

	for _, key := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}

		if vcenter != "all" && vcenter != "" && hs.Vcenter != vcenter {
			continue
		}
		if datacenter != "all" && datacenter != "" && hs.Datacenter != datacenter {
			continue
		}
		if cluster != "all" && cluster != "" && hs.Cluster != cluster {
			continue
		}
		out.KeyRecs = append(out.KeyRecs, &pb.HSKeyRec{HSName: hs.Name, Key: key})
		out.Count++
	}
	return &out, nil
}

// GetClusterListKey возвращает список ключей кластеров, соответствующих фильтру запроса. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetHostPerf(in *pb.HSReqKey, stream grpc.ServerStreamingServer[pb.HSPerf]) error {
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
			streamperf, err := c.GetHostPerf(ctx, in)
			if err == nil {
				for {
					perf, err := streamperf.Recv()
					if err == io.EOF {
						break
					}
					if err := stream.Send(perf); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	hs, ok := infra.HostMapList.Get(key)
	if !ok {
		return errors.New("Host не найден")
	}
	for _, p := range hs.Perf {
		if p == nil {
			continue
		}
		if err := stream.Send(HSPerfToMessage(p)); err != nil {
			_ = err
		}
	}
	return nil
}

// GetHostPerfByKey возвращает историю метрик производительности хоста по его ключу единым срезом. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetHostPerfByKey(_ context.Context, in *pb.HSReqKey) (*pb.HSPerfSlice, error) {
	if config.Switch.Endpoint.Count() > 0 {
		hsperfs := pb.HSPerfSlice{Perf: make([]*pb.HSPerf, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &hsperfs, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			perfSlice, err := c.GetHostPerfByKey(ctx, in)
			if err == nil {
				for _, perf := range perfSlice.Perf {
					hsperfs.Perf = append(hsperfs.Perf, perf)
					hsperfs.Count++
				}
			}
		}
		return &hsperfs, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	hs, ok := infra.HostMapList.Get(key)
	out := pb.HSPerfSlice{}
	if !ok {
		return &out, errors.New("Host не найден")
	}
	for _, p := range hs.Perf {
		if p == nil {
			continue
		}
		out.Perf = append(out.Perf, HSPerfToMessage(p))
	}
	out.Count = int64(len(out.Perf))
	return &out, nil
}

// GetHostByKey возвращает краткую информацию о хосте по его ключу.
func (s *server) GetHostByKey(_ context.Context, in *pb.HSReqKey) (*pb.Host, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.Host{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			hs, err := c.GetHostByKey(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return hs, nil
			}
		}
		return &pb.Host{}, errors.New(errNotFound) //ничего не найдено
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	hs, ok := infra.HostMapList.Get(key)
	if !ok {
		return &pb.Host{}, errors.New("Host не найден")
	}
	out := pb.Host{}

	RecalcHostToMessageShort(hs, &out)
	return &out, nil
}

// GetHostByKeyDirect возвращает информацию о хосте, полученную НАПРЯМУЮ из
// VMware (vsphere.GetHostDirect - отдельное govmomi-соединение, без чтения
// закэшированного в памяти состояния инфраструктуры VMInfrastructure), в
// отличие от GetHostByKey выше, которая читает уже собранный ранее снимок
// (infra.HostMapList). Запрос ретранслируется на удалённые обработчики
// (config.Switch.Endpoint) по тому же принципу, что и остальные методы
// этого файла.
func (s *server) GetHostByKeyDirect(_ context.Context, in *pb.HSReq) (*pb.Host, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.Host{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			hs, err := c.GetHostByKeyDirect(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return hs, nil
			}
		}
		return &pb.Host{}, errors.New(errNotFound) //ничего не найдено
	}

	hs, err := vsphere.GetHostDirect(in.GetVcenter(), in.GetDatacenter(), in.GetCluster(), in.GetHSName())
	if err != nil {
		return &pb.Host{}, err
	}
	out := pb.Host{}
	RecalcHostToMessageShort(&hs, &out)
	return &out, nil
}

// GetHostBySN возвращает краткую информацию о хосте по серийному номеру сервера. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetHostBySN(_ context.Context, in *pb.SerialNumber) (*pb.Host, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.Host{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			hs, err := c.GetHostBySN(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return hs, nil
			}
		}
		return &pb.Host{}, errors.New(errNotFound) //ничего не найдено
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	sn := in.GetSerialNumber()
	hs, ok := infra.HostBySNMapList.Get(sn)
	if !ok {
		return &pb.Host{}, errors.New("Host не найден")
	}
	out := pb.Host{}

	RecalcHostToMessageShort(hs, &out)
	return &out, nil
}
