// Package grpcserver реализует gRPC-серверную часть системы мониторинга
// loadmon: два gRPC-сервиса, Loadmon (сервер server, инвентарь и метрики
// VMware vSphere, оборудования HPE OneView и т.п. — контракт см. в
// protocol/*.proto) и LoadmonDB (сервер serverdb, доступ к данным,
// сохранённым в БД проекта). Сервисы принимают запросы, полученные клиентом
// пакета grpcclient, выполняют выборку/агрегацию данных из внутренних
// пакетов проекта (vspheremetrics, oneview, models и др.) и упаковывают
// результат в сгенерированные gRPC-сообщения пакета protocol. Исходники
// разбиты по доменам: server.go, handler_*.go, copy_*.go, message_*.go.
package grpcserver

import (
	"flag"
	"fmt"
	"loadmon/config"
	pb "loadmon/protocol"
	"log"
	"net"

	grpc "google.golang.org/grpc"
)

// Тексты типовых ошибок, возвращаемых обработчиками gRPC-сервиса клиенту.
var (
	errNotFound           = "Объект не найден"
	errNotInitialized     = "Инфраструктура vsphere не инициализирована"
	errSrvNotFound        = "Сервер с SN:%s не найден"
	errSrvVirtualNotFound = "Сервер с Virtual SN:%s не найден"
	errAddClusterSpr      = "Ошибка вставки записи в справочник Cluster"
)

// server реализует сгенерированный интерфейс pb.LoadmonServer — основной
// gRPC-сервис, отдающий инвентарь и метрики vSphere/оборудования.
type server struct {
	pb.UnimplementedLoadmonServer
}

// serverdb реализует сгенерированный интерфейс pb.LoadmonDBServer — gRPC-
// сервис доступа к данным, сохранённым в БД проекта.
type serverdb struct {
	pb.UnimplementedLoadmonDBServer
}

// StartServer запускает основной gRPC-сервер Loadmon на адресе из config.GetSrvApiAddr() и блокируется в Serve().
// Возвращает ошибку, если не удалось начать приём соединений или сервер завершился с ошибкой во время работы.
func StartServer() error {
	flag.Parse()
	lis, err := net.Listen("tcp", config.GetSrvApiAddr())
	if err != nil {
		log.Fatalf("Ошибка Listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterLoadmonServer(s, &server{})
	log.Printf("Сервер ждет соединения на адресе %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Printf("Ошибка запуска сервера: %v\n", err)
		return err
	}
	return nil
}

// StartServerAddr запускает основной gRPC-сервер Loadmon на заданном ipaddr:portnew и блокируется в Serve().
// Возвращает ошибку при сбое прослушивания порта или во время работы сервера.
func StartServerAddr(ipaddr string, portnew int64) error {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ipaddr, portnew))
	if err != nil {
		log.Printf("Ошибка listen: %v\n", err)
		return err
	}
	s := grpc.NewServer()
	pb.RegisterLoadmonServer(s, &server{})
	log.Printf("Сервер ждет соединения на адресе %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Printf("Ошибка запуска сервера: %v\n", err)
		return err
	}
	return nil
}

// StartDBServerAddr запускает gRPC-сервер БД (LoadmonDBServer) на заданном ipaddr:portnew и блокируется в Serve().
// Возвращает ошибку при сбое прослушивания порта или во время работы сервера.
func StartDBServerAddr(ipaddr string, portnew int64) error {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ipaddr, portnew))
	if err != nil {
		log.Printf("Ошибка listen: %v\n", err)
		return err
	}
	s := grpc.NewServer()
	pb.RegisterLoadmonDBServer(s, &serverdb{})
	log.Printf("Сервер ждет соединения на адресе %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Printf("Ошибка запуска сервера: %v\n", err)
		return err
	}
	return nil
}
