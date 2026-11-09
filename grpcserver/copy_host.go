// Файл содержит функции глубокого копирования gRPC-сообщений хостов,
// полученных от удалённых обработчиков (config.Switch.Endpoint), перед
// пересылкой клиенту.
//
// Реализованы через proto.Clone: копируются ВСЕ поля (включая ClusterId,
// MacAddress, IpAddress, ProductFullName, Interfaces, Perf с задержками
// диска и поля, добавленные в protocol/vm.proto позже), которые ранее
// терялись при ручном копировании по списку полей.
package grpcserver

import (
	pb "loadmon/protocol"

	"google.golang.org/protobuf/proto"
)

// CopyHost создаёт независимую копию сообщения pb.Host. При nil возвращает пустое сообщение.
func CopyHost(hs *pb.Host) *pb.Host {
	if hs == nil {
		return &pb.Host{}
	}
	return proto.Clone(hs).(*pb.Host)
}

// CopyPbHostFull создаёт независимую копию сообщения pb.HostFull (вместе с Perf и Interfaces). При nil возвращает пустое сообщение.
func CopyPbHostFull(in *pb.HostFull) *pb.HostFull {
	if in == nil {
		return &pb.HostFull{}
	}
	return proto.Clone(in).(*pb.HostFull)
}
