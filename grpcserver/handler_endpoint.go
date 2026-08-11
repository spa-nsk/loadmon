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
	"loadmon/config"
	pb "loadmon/protocol"

	cmap "github.com/orcaman/concurrent-map/v2"

	grpc "google.golang.org/grpc"
)

func (p *server) RegisterWorkProcess(_ context.Context, in *pb.RegisterEndpoint) (*pb.StatusWork, error) {
	count := config.Switch.Endpoint.Count()
	key := fmt.Sprintf("%d", count)
	endpoint := config.EndpointRecord{
		GRPCSrvApiAddr: in.GRPCSrvApiAddr,
		GRPCSrvApiPort: in.GRPCSrvApiPort,
		GRPCSrvDBAddr:  in.GRPCSrvDBAddr,
		GRPCSrvDBPort:  in.GRPCSrvDBPort,
	}
	fmt.Println("Регистрация обработчика: ", key, "GRPCSrvApiAddr: ", in.GRPCSrvApiAddr, "GRPCSrvApiPort: ", in.GRPCSrvApiPort, "GRPCSrvDBAddr: ", in.GRPCSrvDBAddr, "GRPCSrvDBPort: ", in.GRPCSrvDBPort)
	endpoint.VcenterMap = cmap.New[string]()
	fmt.Println("Обработка Vcenter:")
	for _, rec := range in.VCenter {
		endpoint.VcenterMap.Set(rec.Hostname, rec.Hostname)
		fmt.Println("\t\t", rec.Hostname)
	}
	endpoint.OneViewMap = cmap.New[string]()
	fmt.Println("Обработка OneView:")
	for _, rec := range in.OneView {
		endpoint.OneViewMap.Set(rec.Endpoint, rec.Endpoint)
		fmt.Println("\t\t", rec.Endpoint)
	}
	config.Switch.Endpoint.Set(key, &endpoint)
	st := pb.StatusWork{State: 0, Details: "ok"}
	return &st, nil
}

// GetClusterSprRecords (реализация LoadmonServer) отдаёт клиенту потоком список зарегистрированных удалённых эндпоинтов (config.Switch.Endpoint). Несмотря на название, к справочнику кластеров БД отношения не имеет — совпадение с методом serverdb.GetClusterSprRecords вызвано, по-видимому, копипастой при добавлении метода.
func (s *server) GetClusterSprRecords(in *pb.EndpointReq, stream grpc.ServerStreamingServer[pb.RegisterEndpoint]) error {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			out := pb.RegisterEndpoint{}
			EndpointToMessage(&out, endpoint)
			if err := stream.Send(&out); err != nil {
				return err
			}
		}
	}
	return nil
}

// EndpointToMessage конвертирует внутреннюю запись config.EndpointRecord в gRPC-сообщение pb.RegisterEndpoint.
// Принимает out и in; при in == nil или out == nil ничего не делает.
func EndpointToMessage(out *pb.RegisterEndpoint, in *config.EndpointRecord) {
	out.GRPCSrvApiAddr = in.GRPCSrvApiAddr
	out.GRPCSrvApiPort = in.GRPCSrvApiPort
	out.GRPCSrvDBAddr = in.GRPCSrvDBAddr
	out.GRPCSrvDBPort = in.GRPCSrvDBPort
	for _, key := range in.VcenterMap.Keys() {
		endpoint, ok := in.VcenterMap.Get(key)
		if !ok {
			continue
		}
		out.VCenter = append(out.VCenter, &pb.VCenter{Hostname: endpoint})
	}

	for _, key := range in.OneViewMap.Keys() {
		endpoint, ok := in.OneViewMap.Get(key)
		if !ok {
			continue
		}
		out.OneView = append(out.OneView, &pb.OV{Endpoint: endpoint})
	}
}
