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
	"context"
	"errors"
	"fmt"
	"io"
	"loadmon/config"
	"loadmon/models"
	"loadmon/oneview"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// dialTimeout — таймаут на установление соединения и выполнение одного
// gRPC-запроса (включая чтение потокового ответа целиком).
const dialTimeout = 30 * time.Second

// StatusWork описывает результат серверной операции (например, очистки
// метрик): State — код состояния (0 — успех, иное — ошибка/предупреждение),
// Details — человекочитаемое описание результата.
type StatusWork struct {
	State   int64
	Details string
}

// dialAPIAddr устанавливает соединение с gRPC-сервисом Loadmon (API метрик
// vSphere/OneView) по произвольному адресу addr и возвращает готового
// клиента, соединение, контекст с таймаутом dialTimeout и функцию его
// отмены. Вызывающая сторона обязана закрыть соединение (defer conn.Close())
// и отменить контекст (defer cancel()) сразу после получения результата.
// При ошибке подключения причина логируется и возвращается в err; conn,
// ctx и cancel в этом случае равны нулевым значениям и использовать их нельзя.
func dialAPIAddr(addr string) (pb.LoadmonClient, *grpc.ClientConn, context.Context, context.CancelFunc, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	return pb.NewLoadmonClient(conn), conn, ctx, cancel, nil
}

// dialAPI — сокращённая форма dialAPIAddr для адреса основного API-сервера
// loadmon, взятого из config.GetCliApiAddr().
func dialAPI() (pb.LoadmonClient, *grpc.ClientConn, context.Context, context.CancelFunc, error) {
	return dialAPIAddr(config.GetCliApiAddr())
}

// dialDB устанавливает соединение с gRPC-сервисом LoadmonDB (работа со
// справочниками/БД) по адресу config.GetCliDBAddr() и возвращает готового
// клиента, соединение, контекст с таймаутом dialTimeout и функцию его
// отмены. Правила использования и обработки ошибки такие же, как у
// dialAPIAddr.
func dialDB() (pb.LoadmonDBClient, *grpc.ClientConn, context.Context, context.CancelFunc, error) {
	conn, err := grpc.NewClient(config.GetCliDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("did not connect: %v", err)
		return nil, nil, nil, nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	return pb.NewLoadmonDBClient(conn), conn, ctx, cancel, nil
}

// LoadVCenterList запрашивает у API-сервера loadmon список подключённых
// vCenter'ов, имена которых содержат filter (пустая строка — все). Использует
// адрес по умолчанию (config.GetCliApiAddr()). При ошибке подключения или RPC
// возвращает пустой (не nil) срез — ошибка логируется внутри.
func LoadVCenterList(filter string) []*vsphere.VCenter {
	return LoadVCenterListEndpoint(nil, filter)
}

// LoadVCenterListEndpoint аналогична LoadVCenterList, но позволяет явно
// указать endpoint (адрес конкретного воркер-процесса) вместо адреса по
// умолчанию; если endpoint == nil, используется config.GetCliApiAddr().
// При ошибке подключения или RPC возвращает пустой (не nil) срез.
func LoadVCenterListEndpoint(endpoint *config.EndpointRecord, filter string) []*vsphere.VCenter {
	out := make([]*vsphere.VCenter, 0)
	connect := config.GetCliApiAddr()
	if endpoint != nil {
		connect = endpoint.GetSrvApiAddr()
	}

	c, conn, ctx, cancel, err := dialAPIAddr(connect)
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetVCenters(ctx, &pb.VCReq{Filter: filter})
	if err == nil {
		for {
			pbvc, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				// Ошибка чтения потока (например, обрыв соединения) — прерываем
				// цикл, чтобы не уйти в бесконечный busy-loop на постоянно
				// повторяющейся ошибке и не разыменовать nil pbvc ниже.
				log.Printf("GetVCenters: ошибка чтения потока: %v", err)
				break
			}
			out = append(out, &vsphere.VCenter{Hostname: pbvc.Hostname, Connected: pbvc.Connected})
		}
	}
	return out
}

// LoadDatacenterList запрашивает список датацентров указанного vCenter.
// При ошибке подключения или RPC возвращает пустой (не nil) срез, ошибка
// логируется внутри функции.
func LoadDatacenterList(vcenter string) []*vsphere.Datacenter {
	out := make([]*vsphere.Datacenter, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetDatacenter(ctx, &pb.DCReq{Vcenter: vcenter})
	if err == nil {
		for {
			pbdc, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetDatacenter: ошибка чтения потока: %v", err)
				break
			}
			out = append(out, &vsphere.Datacenter{Vcenter: pbdc.VCenter, Name: pbdc.Name})
		}
	}
	return out
}

// LoadClusterList запрашивает краткий список кластеров (vcenter/datacenter)
// в указанном датацентре. При ошибке подключения или RPC возвращает пустой
// (не nil) срез, ошибка логируется внутри функции.
func LoadClusterList(vcenter, datacenter string) []*vsphere.ClusterShort {
	out := make([]*vsphere.ClusterShort, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()

	stream, err := c.GetCluster(ctx, &pb.CLReq{Vcenter: vcenter, Datacenter: datacenter})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetCluster: ошибка чтения потока: %v", err)
				break
			}
			out = append(out, &vsphere.ClusterShort{Vcenter: pbcl.Vcenter, Datacenter: pbcl.Datacenter, Name: pbcl.Name})
		}
	}
	return out
}

// LoadClusterListMedium запрашивает список кластеров со средним уровнем
// детализации (без полного списка метрик и хранилищ). При ошибке подключения
// возвращает пустой (не nil) срез; отдельные повреждённые записи потока
// пропускаются (continue), не прерывая чтение остальных.
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
func LoadHostListFull(vcenter, datacenter, cluster, hostname string) []*vsphere.HostInfo {
	out := make([]*vsphere.HostInfo, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetHostFull(ctx, &pb.HSReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, HSName: hostname})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			hs := vsphere.HostInfo{}
			MessageToHostInfoFull(pbcl, &hs)
			out = append(out, &hs)
		}
	}
	return out
}

// LoadHostListKey запрашивает список ключей (KeyName) хостов, подходящих
// под заданные фильтры. При ошибке подключения или RPC возвращает пустой
// (не nil) срез.
func LoadHostListKey(vcenter, datacenter, cluster, hostname string) []string {
	out := make([]string, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	listKey, err := c.GetHostListKey(ctx, &pb.HSReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, HSName: hostname})
	if err == nil {
		for i := int64(0); i < listKey.Count; i++ {
			out = append(out, listKey.KeyRecs[i].Key)
		}
	}
	return out
}

// GetHostByKey возвращает информацию о хосте по его уникальному ключу.
// Возвращает nil при ошибке подключения или если хост не найден.
func GetHostByKey(key string) *vsphere.HostInfo {
	out := vsphere.HostInfo{}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	hs, err := c.GetHostByKey(ctx, &pb.HSReqKey{Key: key})
	if err == nil {
		MessageToHostInfo(hs, &out)
		return &out
	}
	return nil
}

// GetHostBySN возвращает информацию о хосте по серийному номеру. Возвращает
// nil при ошибке подключения или если хост не найден.
func GetHostBySN(sn string) *vsphere.HostInfo {
	out := vsphere.HostInfo{}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	hs, err := c.GetHostBySN(ctx, &pb.SerialNumber{SerialNumber: sn})
	if err == nil {
		MessageToHostInfo(hs, &out)
		return &out
	}
	return nil
}

// LoadHostMetrics возвращает временной ряд метрик производительности хоста
// по его ключу. При ошибке подключения возвращает nil; повреждённые записи
// потока пропускаются.
func LoadHostMetrics(key string) []*vsphere.HSPerf {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetHostPerf(ctx, &pb.HSReqKey{Key: key})
	perf := make([]*vsphere.HSPerf, 0)
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			clp := vsphere.HSPerf{}
			MessageToHostMetric(pbcl, &clp)
			perf = append(perf, &clp)
		}
	}
	return perf
}

// LinkVMToCluster связывает список VM со списком кластеров по составному
// ключу vcenter/datacenter/cluster, заполняя cl[i].VM и cl[i].VMCount.
// Чисто локальная операция, gRPC не использует.
func LinkVMToCluster(vm []*vsphere.VMPoint, cl []*vsphere.Cluster) {
	mapcl := make(map[string]*vsphere.Cluster)
	for _, clp := range cl {
		mapcl[clp.Vcenter+"/"+clp.Datacenter+"/"+clp.Name] = clp
	}
	for _, vmp := range vm {
		clp, ok := mapcl[vmp.Vcenter+"/"+vmp.Datacenter+"/"+vmp.Cluster]
		if ok {
			clp.VM = append(clp.VM, vmp)
			clp.VMCount++
		}
	}
}

// LinkHostToCluster связывает список хостов со списком кластеров по
// составному ключу vcenter/datacenter/cluster, заполняя cl[i].Hosts и
// cl[i].HostCount. Чисто локальная операция, gRPC не использует.
func LinkHostToCluster(hs []*vsphere.HostInfo, cl []*vsphere.Cluster) {
	mapcl := make(map[string]*vsphere.Cluster)
	for _, clp := range cl {
		mapcl[clp.Vcenter+"/"+clp.Datacenter+"/"+clp.Name] = clp
	}
	for _, hsp := range hs {
		clp, ok := mapcl[hsp.Vcenter+"/"+hsp.Datacenter+"/"+hsp.Cluster]
		if ok {
			clp.Hosts = append(clp.Hosts, hsp)
			clp.HostCount++
		}
	}
}

// LoadVMList возвращает список VM, подходящих под заданные фильтры
// (vcenter/datacenter/cluster/vmname могут быть пустыми). Сначала
// запрашивает список ключей (GetVMListKey), затем читает по ним поток VM
// (GetVMStreamByKeyList). При ошибке подключения или запроса списка ключей
// возвращает пустой (не nil) срез.
func LoadVMList(vcenter, datacenter, cluster, vmname string) []*vsphere.VMPoint {
	vmlist := make([]*vsphere.VMPoint, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vmlist
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})

	stream, err2 := c.GetVMStreamByKeyList(ctx, vml)

	if err2 == nil {
		for {
			vmp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetVMStreamByKeyList: ошибка чтения потока: %v", err)
				break
			}
			vm2 := GetVMPointRPC(vmp)
			vmlist = append(vmlist, &vm2)
		}
	}
	return vmlist
}

// CopyAndReCalcVMPerformance копирует точки метрик производительности VM
// (in.Perf) в доменную модель out, попутно вычисляя минимум/максимум/среднее
// по CPU, памяти, диску и сети. При out == nil или in == nil ничего не
// делает. Перед копированием очищает ранее накопленные метрики out
// (out.VMClearPerfomance()).
func CopyAndReCalcVMPerformance(out *vsphere.VMPoint, in *pb.VMPoint) {
	if out == nil || in == nil {
		return
	}
	out.VMClearPerfomance()
	for _, p := range in.Perf {
		perf := vsphere.VMPerf{}
		perf.ID = p.Id
		perf.Start = p.Start.AsTime().Local()
		perf.CPUUsage = p.CpuUsage
		if out.CPUUsageMaximum < perf.CPUUsage {
			out.CPUUsageMaximum = perf.CPUUsage
			out.CPUUsageMaximumTime = perf.Start
		}
		if out.CPUUsageMinimum > perf.CPUUsage {
			out.CPUUsageMinimum = perf.CPUUsage
			out.CPUUsageMinimumTime = perf.Start
		}
		out.CPUUsageAverage += perf.CPUUsage

		perf.MemUsage = p.MemUsage
		if out.MemUsageMaximum < perf.MemUsage {
			out.MemUsageMaximum = perf.MemUsage
			out.MemUsageMaximumTime = perf.Start
		}
		if out.MemUsageMinimum > perf.MemUsage {
			out.MemUsageMinimum = perf.MemUsage
			out.MemUsageMinimumTime = perf.Start
		}
		out.MemUsageAverage += perf.MemUsage

		perf.DiskUsage = p.DiskUsage
		if out.DiskUsageMaximum < perf.DiskUsage {
			out.DiskUsageMaximum = perf.DiskUsage
			out.DiskUsageMaximumTime = perf.Start
		}
		if out.DiskUsageMinimum > perf.DiskUsage {
			out.DiskUsageMinimum = perf.DiskUsage
			out.DiskUsageMinimumTime = perf.Start
		}
		out.DiskUsageAverage += perf.DiskUsage

		perf.DiskRead = p.DiskRead
		if out.DiskReadMaximum < perf.DiskRead {
			out.DiskReadMaximum = perf.DiskRead
			out.DiskReadMaximumTime = perf.Start
		}
		if out.DiskReadMinimum > perf.DiskRead {
			out.DiskReadMinimum = perf.DiskRead
			out.DiskReadMinimumTime = perf.Start
		}
		out.DiskReadAverage += perf.DiskRead

		perf.DiskWrite = p.DiskWrite
		if out.DiskWriteMaximum < perf.DiskWrite {
			out.DiskWriteMaximum = perf.DiskWrite
			out.DiskWriteMaximumTime = perf.Start
		}
		if out.DiskWriteMinimum > perf.DiskWrite {
			out.DiskWriteMinimum = perf.DiskWrite
			out.DiskWriteMinimumTime = perf.Start
		}
		out.DiskWriteAverage += perf.DiskWrite

		perf.NetUsage = p.NetUsage
		if out.NetUsageMaximum < perf.NetUsage {
			out.NetUsageMaximum = perf.NetUsage
			out.NetUsageMaximumTime = perf.Start
		}
		if out.NetUsageMinimum > perf.NetUsage {
			out.NetUsageMinimum = perf.NetUsage
			out.NetUsageMinimumTime = perf.Start
		}
		out.NetUsageAverage += perf.NetUsage

		perf.NetReceived = p.NetReceived
		perf.NetTransmitted = p.NetTransmitted
		perf.DiskTotalReadLatency = p.DiskTotalReadLatency
		perf.DiskTotalWriteLatency = p.DiskTotalWriteLatency
		out.Perf = append(out.Perf, &perf)
		out.PerfCount++
	}

	if out.PerfCount > 0 {
		out.CPUUsageAverage /= float64(out.PerfCount)
		out.MemUsageAverage /= float64(out.PerfCount)
		out.DiskUsageAverage /= float64(out.PerfCount)
		out.DiskReadAverage /= float64(out.PerfCount)
		out.DiskWriteAverage /= float64(out.PerfCount)
		out.NetUsageAverage /= float64(out.PerfCount)
	}
}

// GetVMPointRPC конвертирует gRPC-сообщение pb.VMPoint в доменную модель
// vsphere.VMPoint (основные поля, метрики, файлы VM, диски, использование
// хранилищ). При in == nil возвращает нулевое значение vsphere.VMPoint.
func GetVMPointRPC(in *pb.VMPoint) vsphere.VMPoint {
	var vm vsphere.VMPoint
	if in == nil {
		return vm
	}

	vm.WorkStation = in.Workstation
	vm.TypeWork = int(in.TypeWork)
	vm.Template = in.Template
	vm.TagName = in.TagName
	vm.TagDescription = in.TagDescription
	vm.ID = in.Id

	vm.Name = in.Name
	vm.DomainName = in.DomainName
	vm.GuestDetailedData = in.GuestDetailedData
	vm.ToolsStatus = in.ToolsStatus

	vm.GuestFullName = in.GuestFullName
	vm.HostName = in.HostName
	vm.PathName = in.PathName
	vm.PowerState = in.PowerState
	vm.CPUMhz = in.CpuMhz
	vm.NumCPU = in.NumCPU
	vm.NumCores = in.NumCores
	vm.MemorySizeGB = in.MemorySizeGB
	vm.Network = in.Network
	vm.IPAddress = in.IpAddress
	vm.MacAddress = in.MacAddress
	vm.CapacityGb = in.CapacityGb
	vm.FreeSpaceGb = in.FreeSpaceGb
	vm.PercentUsed = in.PercentUsed

	// Метрики производительности (CPUUsage*, MemUsage*, Disk*, NetUsage*)
	// заполняются внутри CopyAndReCalcVMPerfomance выше — на основе временного
	// ряда in.Perf, а не готовых агрегатов из сообщения.
	vm.Vcenter = in.Vcenter
	vm.Datacenter = in.Datacenter
	vm.Cluster = in.Cluster
	vm.ClusterID = in.ClusterId

	vm.KeyName = in.KeyName
	vm.UUID = in.Uuid
	vm.GuestID = in.GuestId
	vm.Annotation = in.Annotation
	vm.NumEthernetCards = in.NumEthernetCards
	vm.NumVirtualDisks = in.NumVirtualDisks
	vm.DiskCommittedGB = in.DiskCommittedGB
	vm.DiskUncommittedGB = in.DiskUncommittedGB
	vm.CPUBilling = in.CpuBilling
	vm.MemBilling = in.MemBilling
	vm.DiskBilling = in.DiskBilling
	vm.NetBilling = in.NetBilling

	// vm.Perf и vm.PerfCount заполняются внутри CopyAndReCalcVMPerfomance выше.

	for _, f := range in.VMFiles {
		fn := vsphere.VMFile{}
		fn.Type = f.Type
		fn.Name = f.Name
		fn.Size = f.Size
		fn.UniqueSize = f.UniqueSize
		vm.Files = append(vm.Files, fn)
	}
	for _, d := range in.VMDisk {
		dn := vsphere.GuestDiskInfo{}
		dn.CapacityGb = d.CapacityGb
		dn.DiskPath = d.DiskPath
		dn.FreeSpaceGb = d.FreeSpaceGb
		dn.PercentUsed = d.PercentUsed
		vm.Disk = append(vm.Disk, dn)
	}
	for _, d := range in.UsageDatastore {
		dn := vsphere.VMUsageOnDatastore{}
		dn.ID = d.Id
		dn.UID = d.UID
		dn.Vcenter = d.Vcenter
		dn.Datacenter = d.Datacenter
		dn.Cluster = d.Cluster
		dn.CommittedGb = d.CommittedGb
		dn.UncommittedGb = d.UncommittedGb
		dn.UnsharedGb = d.UnsharedGb
		dn.PercentAllocateStorage = d.PercentAllocateStorage
		dn.StorageName = d.StorageName
		dn.Datastore = d.Datastore
		vm.UsageDatastore = append(vm.UsageDatastore, dn)
	}
	return vm
}

// CopyAndReCalcVMPerfomance — устаревшее имя-обёртка над
// CopyAndReCalcVMPerformance (оставлено для обратной совместимости из-за
// опечатки в исходном названии).
func CopyAndReCalcVMPerfomance(out *vsphere.VMPoint, in *pb.VMPoint) {
	CopyAndReCalcVMPerformance(out, in)
}

// GetVMByKey возвращает одну VM (первую найденную) по заданным
// vcenter/datacenter/cluster/vmname. Возвращает ошибку, если подключиться
// к серверу не удалось или данные по ключу не найдены.
func GetVMByKey(vcenter, datacenter, cluster, vmname string) (vsphere.VMPoint, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})

	stream, err2 := c.GetVMStreamByKeyList(ctx, vml)

	vmlist := make([]vsphere.VMPoint, 0)

	if err2 == nil {
		for {
			vmp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetVMStreamByKeyList: ошибка чтения потока: %v", err)
				break
			}
			vmlist = append(vmlist, GetVMPointRPC(vmp))
		}
	}
	if len(vmlist) > 0 {
		return vmlist[0], nil
	}
	return vsphere.VMPoint{}, errors.New("Данные по ключу не найдены.")
}

// GetVM возвращает VM по её уникальному ключу key. Возвращает ошибку, если
// подключиться к серверу не удалось или данные по ключу не найдены.
func GetVM(key string) (vsphere.VMPoint, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vm, err := c.GetVMByKey(ctx, &pb.VMReqKey{Key: key})
	if err == nil {
		return GetVMPointRPC(vm), nil
	}
	return vsphere.VMPoint{}, errors.New("Данные по ключу не найдены.")
}

// GetVMListKey возвращает список ключей VM, подходящих под заданные
// фильтры. Возвращает ошибку, если подключение не удалось или ни одной
// записи не найдено (vml.Count == 0).
func GetVMListKey(vcenter, datacenter, cluster, vmname string) (*pb.VMListKey, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})
	if vml != nil && err == nil {
		if vml.Count > 0 {
			return vml, nil
		}
	}
	return vml, errors.New("Данные по ключу не найдены.")
}

// LoadVMListToClusterStorage дозагружает полные данные VM (по ключам,
// уже сохранённым в in.Storage[*].VM) для каждого кластера из ins и
// подставляет их обратно в in.Storage[i].VM. Ошибки по отдельным VM
// (например, VM удалена) пропускаются без прерывания всей операции.
// При ошибке подключения к серверу функция тихо завершается, не изменяя ins.
func LoadVMListToClusterStorage(ins []*vsphere.Cluster) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return
	}
	defer conn.Close()
	defer cancel()

	for _, in := range ins {
		vms := make([]*vsphere.VMPoint, 0)
		for i, ds := range in.Storage {
			for _, vm := range ds.VM {
				vmp, err := c.GetVMByKey(ctx, &pb.VMReqKey{Key: vm.KeyName})
				if err != nil {
					continue
				}
				vmp2 := GetVMPointRPC(vmp)
				vms = append(vms, &vmp2)
			}
			in.Storage[i].VM = vms
		}
	}
}

// GetDSData возвращает данные хранилища (datastore) по его ключу и
// признак успеха. Возвращает (nil, false) при ошибке подключения/RPC.
func GetDSData(key string) (*vsphere.Datastore, bool) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	s, err := c.GetDSData(ctx, &pb.KeyString{Key: key})
	if err == nil {
		ds := vsphere.Datastore{ID: s.Id,
			UID:         s.UID,
			StorageName: s.StorageName,
			CapacityGb:  s.CapacityGb,
			FreeSpaceGb: s.FreeSpaceGb,
			PercentUsed: s.PercentUsed}
		for _, vmk := range s.VmKeys.KeyRecs {
			vm := vsphere.VMPoint{Name: vmk.VMName, KeyName: vmk.Key}
			ds.VMCount++
			ds.VM = append(ds.VM, &vm)
		}
		return &ds, true
	}
	return nil, false
}

// GetAllDSData возвращает список всех хранилищ, связанных с ключом key
// (например, все datastore кластера). При ошибке подключения возвращает
// пустой (не nil) срез; повреждённые записи потока пропускаются.
func GetAllDSData(key string) []*vsphere.Datastore {
	allds := make([]*vsphere.Datastore, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return allds
	}
	defer conn.Close()
	defer cancel()
	s, err := c.GetAllDSData(ctx, &pb.KeyString{Key: key})

	if err == nil {
		for {
			pbds, err := s.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			ds := vsphere.Datastore{ID: pbds.Id,
				UID:         pbds.UID,
				StorageName: pbds.StorageName,
				CapacityGb:  pbds.CapacityGb,
				FreeSpaceGb: pbds.FreeSpaceGb,
				PercentUsed: pbds.PercentUsed}

			allds = append(allds, &ds)
		}
	}
	return allds
}

// Очистить метрики в указанном диапазоне для одиночного cluster или всех cluster="all" или cluster=""
func ClearMetric(cluster string, start, end time.Time) (*StatusWork, error) {
	s2 := StatusWork{State: -2, Details: "Нет данных"}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		s2.Details = "did not connect: " + err.Error()
		return &s2, nil
	}
	defer conn.Close()
	defer cancel()
	s, err := c.ClearMetric(ctx, &pb.Duration{Cluster: cluster, Start: timestamppb.New(start), End: timestamppb.New(end)})
	if err != nil {
		return &s2, err
	}
	if s != nil {
		s2.State = s.State
		s2.Details = s.Details
	}
	return &s2, nil
}

// Очистить метрики в указанном диапазоне
func ClearMetricByKey(vcenter, datacenter, cluster, vmname string, start, end time.Time) error {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return err
	}
	defer conn.Close()
	defer cancel()
	s, err := c.ClearMetricByKey(ctx, &pb.ClearMetricKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname, Start: timestamppb.New(start), End: timestamppb.New(end)})
	if err == nil {
		if s.State != 0 {
			return errors.New(s.Details)
		}
		return nil
	}
	return err
}

// Очистить метрики в указанном диапазоне
func ClearAllMetricByKey(vcenter, datacenter, cluster, vmname string, start, end time.Time) error {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return err
	}
	defer conn.Close()
	defer cancel()
	s, err := c.ClearAllMetricByKey(ctx, &pb.ClearMetricKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname, Start: timestamppb.New(start), End: timestamppb.New(end)})
	if err == nil {
		if s.State != 0 {
			return errors.New(s.Details)
		}
		return nil
	}
	return err
}

// написать функцию аналог vsphere.DestroyData()
func DestroyData() {

}

// написать функцию аналог vsphere
func Load_DC_HS_CL_Info() {

}

// написать функцию аналог vsphere
func LoadVMInfo() {

}

// GetServerHardwareBySN возвращает данные сервера HPE OneView по его
// серийному номеру и признак успеха. Возвращает (nil, false) при ошибке
// подключения или если сервер не найден.
func GetServerHardwareBySN(sn string) (*oneview.ServerHardware, bool) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	out := oneview.ServerHardware{}
	srv, err := c.GetServerHardwareBySN(ctx, &pb.SerialNumber{SerialNumber: sn})
	if err != nil {
		return nil, false
	}
	MessageToServerHardware(srv, &out)

	return &out, true
}

// GetServerHardwareByName возвращает данные сервера HPE OneView по его
// имени и признак успеха. Возвращает (nil, false) при ошибке подключения
// или если сервер не найден.
func GetServerHardwareByName(name string) (*oneview.ServerHardware, bool) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	out := oneview.ServerHardware{}
	srv, err := c.GetServerHardwareByName(ctx, &pb.ServerName{Name: name})
	if err != nil {
		return nil, false
	}
	MessageToServerHardware(srv, &out)

	return &out, true
}

// GetServerHardwareByVirtualSN возвращает данные сервера HPE OneView по
// виртуальному серийному номеру и признак успеха. Возвращает (nil, false)
// при ошибке подключения или если сервер не найден.
func GetServerHardwareByVirtualSN(sn string) (*oneview.ServerHardware, bool) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	out := oneview.ServerHardware{}
	srv, err := c.GetServerHardwareByVirtualSN(ctx, &pb.SerialNumber{SerialNumber: sn})
	if err != nil {
		return nil, false
	}
	MessageToServerHardware(srv, &out)

	return &out, true
}

// GetServerHardwareList возвращает список серверов HPE OneView,
// отфильтрованных по датацентру, стойке, статусу, режиму обслуживания и
// состоянию питания (пустая строка — фильтр не применяется). При ошибке
// подключения возвращает (пустой срез, ошибка); повреждённые записи потока
// прерывают дальнейшее чтение.
func GetServerHardwareList(datacenterName, rackName, status, maintenanceMode, powerState string) ([]*oneview.ServerHardware, error) {
	out := make([]*oneview.ServerHardware, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out, err
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetServerHardwareList(ctx, &pb.HWReq{DatacenterName: datacenterName,
		RackName:        rackName,
		Status:          status,
		MaintenanceMode: maintenanceMode,
		PowerState:      powerState})
	if err == nil {
		for {
			srv, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetServerHardwareList: ошибка чтения потока: %v", err)
				break
			}
			s := oneview.ServerHardware{}
			MessageToServerHardware(srv, &s)
			out = append(out, &s)
		}
	}
	return out, nil
}

//-------------------------------- DB interface ---------------------------------

// LoadClusterSpr запрашивает у сервиса LoadmonDB справочные записи кластеров
// (ClusterSprRecords) по заданным vcenter/datacenter/cluster и возвращает их
// в виде карты по ключу "vcenter/datacenter/name". Возвращает ошибку при
// неудачном подключении или если не найдено ни одной записи.
func LoadClusterSpr(vcenter, datacenter, cluster string) (map[string]*models.ClusterSprRecords, error) {
	clmap := make(map[string]*models.ClusterSprRecords, 0)
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return clmap, err
	}
	defer conn.Close()
	defer cancel()

	stream, err := c.GetClusterSprRecords(ctx, &pb.ClusterSprKey1{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})

	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetClusterSprRecords: ошибка чтения потока: %v", err)
				break
			}
			cl := models.ClusterSprRecords{}
			MessageToClusterSprRecords(pbcl, &cl)
			clmap[fmt.Sprintf("%s/%s/%s", cl.Vcenter, cl.Datacenter, cl.Name)] = &cl
		}
	}
	if len(clmap) > 0 {
		return clmap, nil
	}
	return nil, errors.New("Данные по ключу не найдены.")
}

// AddClusterSprRecord добавляет новую справочную запись кластера через
// сервис LoadmonDB. Возвращает (id, статус); при ошибке подключения
// возвращает (0, 1), при ошибке RPC — (0, 3).
func AddClusterSprRecord(in *models.ClusterSprRecords) (int64, int) {
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return 0, 1
	}
	defer conn.Close()
	defer cancel()
	req := pb.ClusterSprRecord{}
	ClusterSprRecordsToMessage(in, &req)
	clusterId, err := c.AddClusterSprRecord(ctx, &req)
	if err == nil {
		return clusterId.Id, int(clusterId.Status)
	}
	return 0, 3 //ошибка операции
}

// UpdateClusterSprRecord обновляет существующую справочную запись кластера
// через сервис LoadmonDB. Возвращает (id, статус); при ошибке подключения
// возвращает (0, 1), при ошибке RPC — (0, 3).
func UpdateClusterSprRecord(in *models.ClusterSprRecords) (int64, int) {
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return 0, 1
	}
	defer conn.Close()
	defer cancel()
	req := pb.ClusterSprRecord{}
	ClusterSprRecordsToMessage(in, &req)
	clusterId, err := c.UpdateClusterSprRecord(ctx, &req)
	if err == nil {
		return clusterId.Id, int(clusterId.Status)
	}
	return 0, 3 //ошибка операции
}

// общие функции
func RegisterWorkProcess(conf *config.JsonConfig) error {
	c, conn, ctx, cancel, err := dialAPIAddr(conf.GetSwitchAddr())
	if err != nil {
		return err
	}
	defer conn.Close()
	defer cancel()
	req := pb.RegisterEndpoint{}
	req.GRPCSrvApiPort = conf.GRPCSrvApiPort
	req.GRPCSrvApiAddr = conf.GRPCSrvApiAddr
	req.GRPCSrvDBAddr = conf.GRPCCliDBAddr
	req.GRPCSrvDBPort = conf.GRPCCliDBPort
	for _, rec := range conf.ConfigVCenters {
		req.VCenter = append(req.VCenter, &pb.VCenter{Hostname: rec.Hostname})
	}
	for _, rec := range conf.ConfigOV {
		req.OneView = append(req.OneView, &pb.OV{Endpoint: rec.Endpoint})
	}

	s, err := c.RegisterWorkProcess(ctx, &req)
	if err == nil {
		if s.State != 0 {
			return errors.New(s.Details)
		}
		return nil
	}
	return err
}

// GetWorkEndpoints запрашивает у switch-сервера актуальный список
// зарегистрированных воркеров (endpoint'ов) для заданных vcenter/oneview
// и заменяет ими содержимое реестра config.Switch.Endpoint. Возвращает
// ошибку при неудачном подключении или ошибке RPC-вызова.
func GetWorkEndpoints(vcenter, oneview string) error {
	config.Switch.Endpoint.Clear()
	count := config.Switch.Endpoint.Count()
	// Примечание: здесь намеренно используется LoadmonClient (не LoadmonDBClient),
	// т.к. "switch"-процесс отдаёт список зарегистрированных воркеров через
	// сервис Loadmon, слушающий на адресе, сконфигурированном как DB-адрес.
	c, conn, ctx, cancel, err := dialAPIAddr(config.GetCliDBAddr())
	if err != nil {
		return err
	}
	defer conn.Close()
	defer cancel()

	stream, err := c.GetClusterSprRecords(ctx, &pb.EndpointReq{Vcenter: vcenter, OneView: oneview})

	if err != nil {
		return err
	}
	for {
		endpoint, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("GetClusterSprRecords (switch): ошибка чтения потока: %v", err)
			break
		}
		key := fmt.Sprintf("%d", count)
		out := config.EndpointRecord{}
		MessageToEndpoint(&out, endpoint)
		config.Switch.Endpoint.Set(key, &out)
	}
	return nil
}

// MessageToEndpoint конвертирует gRPC-сообщение pb.RegisterEndpoint в
// доменную модель config.EndpointRecord, заполняя карты обслуживаемых
// vCenter и OneView.
func MessageToEndpoint(out *config.EndpointRecord, in *pb.RegisterEndpoint) {
	out.GRPCSrvApiAddr = in.GRPCSrvApiAddr
	out.GRPCSrvApiPort = in.GRPCSrvApiPort
	out.GRPCSrvDBAddr = in.GRPCSrvDBAddr
	out.GRPCSrvDBPort = in.GRPCSrvDBPort
	for _, rec := range in.VCenter {
		out.VcenterMap.Set(rec.Hostname, rec.Hostname)
	}
	for _, rec := range in.OneView {
		out.OneViewMap.Set(rec.Endpoint, rec.Endpoint)
	}
}
