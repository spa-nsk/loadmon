// Файл содержит функции глубокого копирования gRPC-сообщений кластеров и
// хранилищ, полученных от удалённых обработчиков (config.Switch.Endpoint),
// перед пересылкой клиенту.
//
// Все функции реализованы через proto.Clone: копируются ВСЕ поля сообщения
// (включая вложенные Perf, Storages, VmKeys и поля, добавленные в
// protocol/vm.proto позже), и нет риска забыть новое поле или получить
// nil-разыменование вложенных сообщений (ранее ручное копирование теряло
// ClusterId/TypeWork и падало на Storages[].VmKeys == nil).
package grpcserver

import (
	pb "loadmon/protocol"

	"google.golang.org/protobuf/proto"
)

// CopyClusterFull создаёт независимую копию сообщения pb.ClusterFull (с метриками Perf и Storages). При nil возвращает пустое сообщение.
func CopyClusterFull(pbcl *pb.ClusterFull) *pb.ClusterFull {
	if pbcl == nil {
		return &pb.ClusterFull{}
	}
	return proto.Clone(pbcl).(*pb.ClusterFull)
}

// CopyClusterStorages создаёт независимую копию сообщения pb.Datastore, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту. При nil возвращает пустое сообщение.
func CopyClusterStorages(st *pb.Datastore) *pb.Datastore {
	return CopyDatastore(st)
}

// CopyCLPerf создаёт независимую копию точки метрик кластера pb.CLPerf. При nil возвращает пустое сообщение.
func CopyCLPerf(in *pb.CLPerf) *pb.CLPerf {
	if in == nil {
		return &pb.CLPerf{}
	}
	return proto.Clone(in).(*pb.CLPerf)
}

// CopyClusterMedium создаёт независимую копию сообщения pb.Cluster (агрегированные метрики без хостов/хранилищ). При nil возвращает пустое сообщение.
func CopyClusterMedium(pbcl *pb.Cluster) *pb.Cluster {
	if pbcl == nil {
		return &pb.Cluster{}
	}
	return proto.Clone(pbcl).(*pb.Cluster)
}

// CopyDatastore создаёт независимую копию сообщения pb.Datastore вместе со списком ключей связанных ВМ (VmKeys). При nil возвращает пустое сообщение.
func CopyDatastore(in *pb.Datastore) *pb.Datastore {
	if in == nil {
		return &pb.Datastore{}
	}
	return proto.Clone(in).(*pb.Datastore)
}
