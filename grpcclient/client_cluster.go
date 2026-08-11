// Package grpcclient — gRPC-клиент пакета loadmon. Обращается к серверу
// пакета grpcserver, который отдаёт метрики VMware vSphere (сервис Loadmon,
// адрес config.GetCliApiAddr()) и данные из БД (сервис LoadmonDB, адрес
// config.GetCliDBAddr()), а также к серверу HPE OneView за инвентарём
// оборудования.
//
// Общий стиль функций пакета:
//   - на каждый вызов открывается короткоживущее соединение через dialAPI()/
//     dialDB()/dialAPIAddr() с таймаутом 30 секунд на весь запрос (включая
//     чтение потоковых ответов), соединение закрывается через defer сразу
//     после установления;
//   - при ошибке подключения или вызова RPC в лог пишется причина
//     (log.Printf), а вызывающему возвращается "пустой" результат (nil/nil-
//     срез/false) и, где это предусмотрено сигнатурой, сама ошибка;
//   - при потоковом чтении (stream.Recv()) признаком конца потока служит
//     io.EOF, любая другая ошибка прерывает или пропускает элемент в
//     зависимости от функции (см. комментарии на местах).
package grpcclient

import (
	"io"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
)

func LoadClusterListMedium(vcenter, datacenter, cluster string) []*vsphere.Cluster {
	out := make([]*vsphere.Cluster, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetClusterMedium(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			cl := vsphere.Cluster{}
			MessageToClusterMedium(pbcl, &cl)
			out = append(out, &cl)
		}
	}
	return out
}

// GetClusterMedium возвращает один кластер (первый найденный по ключу
// vcenter/datacenter/cluster) со средним уровнем детализации, либо nil, если
// кластер не найден или произошла ошибка подключения/RPC.
func GetClusterMedium(vcenter, datacenter, cluster string) *vsphere.Cluster {
	out := make([]*vsphere.Cluster, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetClusterMedium(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			cl := vsphere.Cluster{}
			MessageToClusterMedium(pbcl, &cl)
			out = append(out, &cl)
		}
	}
	if len(out) > 0 {
		return out[0]
	}
	return nil
}

// GetClusterFull возвращает один кластер (первый найденный по ключу
// vcenter/datacenter/cluster) с полным набором данных (метрики, хранилища),
// либо nil, если кластер не найден или произошла ошибка подключения/RPC.
func GetClusterFull(vcenter, datacenter, cluster string) *vsphere.Cluster {
	out := make([]*vsphere.Cluster, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetClusterFull(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			cl := vsphere.Cluster{}
			MessageToClusterFull(pbcl, &cl)
			out = append(out, &cl)
		}
	}
	if len(out) > 0 {
		return out[0]
	}
	return nil
}

// GetClusterStorages возвращает список хранилищ (datastore) указанного
// кластера. При ошибке подключения возвращает nil; повреждённые записи
// потока пропускаются.
func GetClusterStorages(vcenter, datacenter, cluster string) []*vsphere.Datastore {
	out := make([]*vsphere.Datastore, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetClusterStorages(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err == nil {
		for {
			pbds, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			ds := vsphere.Datastore{}
			MessageToClusterStorages(pbds, &ds)
			out = append(out, &ds)
		}
	}
	return out
}

// GetClusterHost возвращает список хостов указанного кластера. При ошибке
// подключения возвращает nil; повреждённые записи потока пропускаются.
func GetClusterHost(vcenter, datacenter, cluster string) []*vsphere.HostInfo {
	out := make([]*vsphere.HostInfo, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetClusterHost(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err == nil {
		for {
			pbhs, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			hs := vsphere.HostInfo{}
			MessageToHostInfo(pbhs, &hs)
			out = append(out, &hs)
		}
	}
	return out
}

// LoadClusterMetrics возвращает временной ряд метрик производительности
// указанного кластера. При ошибке подключения возвращает пустой (не nil)
// срез; при ошибке чтения потока (кроме io.EOF) чтение прерывается.
func LoadClusterMetrics(vcenter, datacenter, cluster string) []*vsphere.CLPerf {
	perf := make([]*vsphere.CLPerf, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return perf
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetClusterMetrics(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			clp := vsphere.CLPerf{}
			MessageToClusterMetric(pbcl, &clp)
			perf = append(perf, &clp)
		}
	}
	return perf
}

// LoadClusterListFull запрашивает список кластеров с полным набором данных
// (аналогично GetClusterFull, но возвращает все найденные записи, а не
// только первую). При ошибке подключения возвращает пустой (не nil) срез.
func LoadClusterListFull(vcenter, datacenter, cluster string) []*vsphere.Cluster {
	out := make([]*vsphere.Cluster, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetClusterFull(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			cl := vsphere.Cluster{}
			MessageToClusterFull(pbcl, &cl)
			out = append(out, &cl)
		}
	}
	return out
}

// LoadHostListFull запрашивает список хостов с полным набором данных для
// заданных vcenter/datacenter/cluster/hostname (фильтры могут быть пустыми).
// При ошибке подключения возвращает пустой (не nil) срез.
