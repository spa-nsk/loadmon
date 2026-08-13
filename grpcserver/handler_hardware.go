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
	"fmt"
	"io"
	"loadmon/config"
	pb "loadmon/protocol"
	"log"
	"time"

	oneview "github.com/spa-nsk/oneview2"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (p *server) GetServerHardwareBySN(_ context.Context, in *pb.SerialNumber) (*pb.ServerHardware, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ServerHardware{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			hw, err := c.GetServerHardwareBySN(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return hw, nil
			}
		}
		return &pb.ServerHardware{}, fmt.Errorf(errSrvNotFound, in.SerialNumber) //ничего не найдено
	}
	ov := oneview.GetGlobalOVInfrastructure()
	hw, ok := ov.ServerBySN[in.SerialNumber]
	if ok {
		pbhw := pb.ServerHardware{}
		ServerHardwareToMessage(hw, &pbhw)
		return &pbhw, nil
	}
	return nil, fmt.Errorf(errSrvNotFound, in.SerialNumber)
}

// GetServerHardwareByName возвращает данные об аппаратном сервере (HPE OneView) по его имени.
//
// Примечание про oneview2: индекс по имени (ServerByName) строится в
// oneview.Infrastructure.LoadServerHardwareList() из oneview2.GetHWserverName(),
// как и раньше.
func (p *server) GetServerHardwareByName(_ context.Context, in *pb.ServerName) (*pb.ServerHardware, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ServerHardware{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			hw, err := c.GetServerHardwareByName(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return hw, nil
			}
		}
		return &pb.ServerHardware{}, fmt.Errorf(errSrvNotFound, in.Name) //ничего не найдено
	}
	ov := oneview.GetGlobalOVInfrastructure()
	hw, ok := ov.ServerByName.Get(in.Name)
	if ok {
		pbhw := pb.ServerHardware{}
		ServerHardwareToMessage(hw, &pbhw)
		return &pbhw, nil
	}
	return nil, fmt.Errorf(errSrvNotFound, in.Name)
}

// GetServerHardwareByVirtualSN возвращает данные об аппаратном сервере (HPE OneView) по виртуальному серийному номеру.
//
// Примечание про oneview2: у oneview2.ServerIdentity нет поля виртуального
// серийного номера (в отличие от старого oneview.ServerHardware.Base.
// VirtualSerialNumber) - индекс ServerByVirtualSN в oneview2.Infrastructure
// сейчас всегда пуст, поэтому этот метод всегда возвращает "не найдено" в
// локальном режиме (без Switch.Endpoint). Если виртуальный серийный номер
// снова понадобится, поле нужно будет добавить в oneview2.ServerIdentity.
func (p *server) GetServerHardwareByVirtualSN(_ context.Context, in *pb.SerialNumber) (*pb.ServerHardware, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ServerHardware{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			hw, err := c.GetServerHardwareByVirtualSN(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return hw, nil
			}
		}
		return &pb.ServerHardware{}, fmt.Errorf(errSrvVirtualNotFound, in.SerialNumber) //ничего не найдено
	}
	ov := oneview.GetGlobalOVInfrastructure()
	hw, ok := ov.ServerByVirtualSN[in.SerialNumber]
	if ok {
		pbhw := pb.ServerHardware{}
		ServerHardwareToMessage(hw, &pbhw)
		return &pbhw, nil
	}
	return nil, fmt.Errorf(errSrvVirtualNotFound, in.SerialNumber)
}

// GetServerHardwareList отдаёт клиенту потоком список всех известных аппаратных серверов (HPE OneView).
func (s *server) GetServerHardwareList(in *pb.HWReq, stream grpc.ServerStreamingServer[pb.ServerHardware]) error {
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
			streamhw, err := c.GetServerHardwareList(ctx, in)
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
			}
		}
		return nil
	}
	ov := oneview.GetGlobalOVInfrastructure()

	for key := range ov.ServerBySN {
		hw, ok := ov.ServerBySN[key]
		if ok {
			if in.DatacenterName != "all" && in.DatacenterName != hw.Datacenter.Name {
				continue
			}
			if in.RackName != "all" && in.RackName != hw.Rack.Name {
				continue
			}
			if in.PowerState != "all" && in.PowerState != hw.Identity.PowerState {
				continue
			}
			if in.Status != "all" && in.Status != hw.Identity.Status {
				continue
			}
			// Примечание про oneview2: у oneview2.ServerIdentity нет признака
			// режима обслуживания (в отличие от старого oneview.ServerHardware.
			// Base.MaintenanceMode) - фильтр по MaintenanceMode сейчас
			// применить нельзя, он молча игнорируется (кроме значения "all").
			pbhw := pb.ServerHardware{}
			ServerHardwareToMessage(hw, &pbhw)
			if err := stream.Send(&pbhw); err != nil {
				return err
			}
		}
	}
	return nil
}

//------------------- DB interface -------------------

// GetClusterSprRecords (реализация LoadmonDBServer) отдаёт клиенту потоком записи справочника кластеров из БД
// по фильтру ключа. Используется отдельным сервером БД (см. StartDBServerAddr).
