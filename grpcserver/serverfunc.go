package grpcserver

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"loadmon/config"
	"loadmon/models"
	"loadmon/oneview"
	"loadmon/protocol"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/orm"
	cmap "github.com/orcaman/concurrent-map/v2"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errNotFound         = "Объект не найден"
	errNotInitialized   = "Инфраструктура vsphere не инициализирована"
	errSrvNotFound      = "Сервер с SN:%s не найден"
	errSrvVirtualNotFound = "Сервер с Virtual SN:%s не найден"
	errAddClusterSpr    = "Ошибка вставки записи в справочник Cluster"
)

type server struct {
	pb.UnimplementedLoadmonServer
}

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

// GetVCenters отдаёт клиенту потоком список известных vCenter'ов. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVCenters(in *pb.VCReq, stream grpc.ServerStreamingServer[pb.VCenter]) error {
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
			streamcl, err := c.GetVCenters(ctx, &pb.VCReq{Filter: in.Filter})
			if err != nil {
				return err
			}
			for {
				pbvc, err := streamcl.Recv()
				if err == io.EOF {
					break
				}
				if err := stream.Send(pbvc); err != nil {
					return err
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.VCenterMapList.Keys() {
		vc, ok := infra.VCenterMapList.Get(key)
		if !ok {
			continue
		}
		out := pb.VCenter{Hostname: vc.Hostname, Connected: vc.Connected}
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

// GetDatacenter отдаёт клиенту потоком список датацентров (уникальных по имени в рамках выбранного vCenter). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetDatacenter(in *pb.DCReq, stream grpc.ServerStreamingServer[pb.Datacenter]) error {
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
			streamcl, err := c.GetDatacenter(ctx, &pb.DCReq{Vcenter: in.Vcenter})
			if err == nil {
				for {
					pbdc, err := streamcl.Recv()
					if err == io.EOF {
						break
					}
					if err := stream.Send(pbdc); err != nil {
						return err
					}
				}
			} else {
				return err
			}
		}
		return nil
	}
	dcm := make(map[string]*string)
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if in.Vcenter != "all" && cl.Vcenter != in.Vcenter {
			continue
		}
		if _, ok := dcm[cl.Datacenter]; ok {
			continue
		}
		dcm[cl.Datacenter] = nil
		out := pb.Datacenter{VCenter: cl.Vcenter, Name: cl.Datacenter}
		if err := stream.Send(&out); err != nil {
			return err
		}
	}
	return nil
}

// GetCluster отдаёт клиенту потоком краткую информацию о кластерах, отфильтрованную по vCenter/датацентру. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
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
func CopyHost(hs *pb.Host) *pb.Host {
	out := pb.Host{}
	if hs == nil {
		return &out
	}
	out.TagName = hs.TagName
	out.TagDescription = hs.TagDescription
	out.Id = hs.Id
	out.Name = hs.Name

	out.Vcenter = hs.Vcenter
	out.Datacenter = hs.Datacenter
	out.Cluster = hs.Cluster

	out.KeyName = hs.KeyName
	out.Vendor = hs.Vendor
	out.Model = hs.Model

	out.ProductFullName = hs.ProductFullName
	out.SerialNumber = hs.SerialNumber

	out.VirtualSerialNumber = hs.VirtualSerialNumber
	out.EnclosureSerialNumber = hs.EnclosureSerialNumber
	out.Uuid = hs.Uuid
	out.PowerState = hs.PowerState
	out.InMaintenanceMode = hs.InMaintenanceMode
	out.CpuModel = hs.CpuModel
	out.CpuMhz = hs.CpuMhz
	out.NumCpuPkgs = hs.NumCpuPkgs
	out.NumCpuCores = hs.NumCpuCores
	out.NumCpuThreads = hs.NumCpuThreads
	out.NumHBAs = hs.NumHBAs
	out.NumNics = hs.NumNics
	out.IpAddress = hs.IpAddress
	out.MemorySizeGB = hs.MemorySizeGB
	out.HostMaxVirtualDiskCapacity = hs.HostMaxVirtualDiskCapacity
	out.PerfCount = hs.PerfCount
	out.CpuUsageMaximum = hs.CpuUsageMaximum
	out.CpuUsageMaximumTime = hs.CpuUsageMaximumTime
	out.CpuUsageMinimum = hs.CpuUsageMinimum
	out.CpuUsageMinimumTime = hs.CpuUsageMinimumTime
	out.MemUsageMaximum = hs.MemUsageMaximum
	out.MemUsageMaximumTime = hs.MemUsageMaximumTime
	out.MemUsageMinimum = hs.MemUsageMinimum
	out.MemUsageMinimumTime = hs.MemUsageMinimumTime
	out.DiskUsageMaximum = hs.DiskUsageMaximum
	out.DiskUsageMaximumTime = hs.DiskUsageMaximumTime
	out.DiskUsageMinimum = hs.DiskUsageMinimum
	out.DiskUsageMinimumTime = hs.DiskUsageMinimumTime
	out.DiskWriteMaximum = hs.DiskWriteMaximum
	out.DiskWriteMaximumTime = hs.DiskWriteMaximumTime
	out.DiskWriteMinimum = hs.DiskWriteMinimum
	out.DiskWriteMinimumTime = hs.DiskWriteMinimumTime
	out.DiskReadMaximum = hs.DiskReadMaximum
	out.DiskReadMaximumTime = hs.DiskReadMaximumTime
	out.DiskReadMinimum = hs.DiskReadMinimum
	out.DiskReadMinimumTime = hs.DiskReadMinimumTime
	out.NetUsageMaximum = hs.NetUsageMaximum
	out.NetUsageMaximumTime = hs.NetUsageMaximumTime
	out.NetUsageMinimum = hs.NetUsageMinimum
	out.NetUsageMinimumTime = hs.NetUsageMinimumTime
	out.CpuUsageAverage = hs.CpuUsageAverage
	out.MemUsageAverage = hs.MemUsageAverage
	out.DiskUsageAverage = hs.DiskUsageAverage
	out.DiskWriteAverage = hs.DiskWriteAverage
	out.DiskReadAverage = hs.DiskReadAverage
	out.NetUsageAverage = hs.NetUsageAverage
	return &out
}

// CopyClusterFull создаёт независимую копию сообщения pb.ClusterFull, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту.
func CopyClusterFull(pbcl *pb.ClusterFull) *pb.ClusterFull {
	pbclf := pb.ClusterFull{
		Id:                    pbcl.Id,
		Date:                  pbcl.Date,
		Vcenter:               pbcl.Vcenter,
		Datacenter:            pbcl.Datacenter,
		Name:                  pbcl.Name,
		CpuMhzAvg:             pbcl.CpuMhzAvg,
		AllNumCpuPkgs:         pbcl.AllNumCpuPkgs,
		AllNumCpuCores:        pbcl.AllNumCpuCores,
		AllNumCpuThreads:      pbcl.AllNumCpuThreads,
		AllMemorySizeGB:       pbcl.AllMemorySizeGB,
		AllCapacityGb:         pbcl.AllCapacityGb,
		AllFreeSpaceGb:        pbcl.AllFreeSpaceGb,
		AllPercentUsed:        pbcl.AllPercentUsed,
		AllocatedCoreCount:    pbcl.AllocatedCoreCount,
		AllocatedMemorySizeGb: pbcl.AllocatedMemorySizeGb,
		CoreTransferRate:      pbcl.CoreTransferRate,
		MemoryTransferRate:    pbcl.MemoryTransferRate,
		CpuUsageMaximum:       pbcl.CpuUsageMaximum,
		CpuUsageMaximumTime:   pbcl.CpuUsageMaximumTime,
		CpuUsageAverage:       pbcl.CpuUsageAverage,
		CpuUsageMinimum:       pbcl.CpuUsageMinimum,
		CpuUsageMinimumTime:   pbcl.CpuUsageMinimumTime,
		MemUsageMaximum:       pbcl.MemUsageMaximum,
		MemUsageMaximumTime:   pbcl.MemUsageMaximumTime,
		MemUsageAverage:       pbcl.MemUsageAverage,
		MemUsageMinimum:       pbcl.MemUsageMinimum,
		MemUsageMinimumTime:   pbcl.MemUsageMinimumTime,
		DiskUsageMaximum:      pbcl.DiskUsageMaximum,
		DiskUsageMaximumTime:  pbcl.DiskUsageMaximumTime,
		DiskUsageAverage:      pbcl.DiskUsageAverage,
		DiskUsageMinimum:      pbcl.DiskUsageMinimum,
		DiskUsageMinimumTime:  pbcl.DiskUsageMinimumTime,
		DiskWriteMaximum:      pbcl.DiskWriteMaximum,
		DiskWriteMaximumTime:  pbcl.DiskWriteMaximumTime,
		DiskWriteAverage:      pbcl.DiskWriteAverage,
		DiskWriteMinimum:      pbcl.DiskWriteMinimum,
		DiskWriteMinimumTime:  pbcl.DiskWriteMinimumTime,
		DiskReadMaximum:       pbcl.DiskReadMaximum,
		DiskReadMaximumTime:   pbcl.DiskReadMaximumTime,
		DiskReadAverage:       pbcl.DiskReadAverage,
		DiskReadMinimum:       pbcl.DiskReadMinimum,
		DiskReadMinimumTime:   pbcl.DiskReadMinimumTime,
		NetUsageMaximum:       pbcl.NetUsageMaximum,
		NetUsageMaximumTime:   pbcl.NetUsageMaximumTime,
		NetUsageAverage:       pbcl.NetUsageAverage,
		NetUsageMinimum:       pbcl.NetUsageMinimum,
		NetUsageMinimumTime:   pbcl.NetUsageMinimumTime,
		PerfCount:             pbcl.PerfCount,
		HostFlag:              pbcl.HostFlag,
		HostCount:             pbcl.HostCount,
		VMCount:               pbcl.VMCount,
		Perf:                  make([]*pb.CLPerf, len(pbcl.Perf)),
		Storages:              make([]*pb.Datastore, len(pbcl.Storages))}

	for i, perfcl := range pbcl.Perf {
		perf := pb.CLPerf{
			Id:             perfcl.Id,
			Start:          perfcl.Start,
			CpuUsage:       perfcl.CpuUsage,
			CpuUtilization: perfcl.CpuUtilization,
			MemUsage:       perfcl.MemUsage,
			DiskUsage:      perfcl.DiskUsage,
			DiskWrite:      perfcl.DiskWrite,
			DiskRead:       perfcl.DiskRead,
			NetUsage:       perfcl.NetUsage,
			NetReceived:    perfcl.NetReceived,
			NetTransmitted: perfcl.NetTransmitted}
		pbclf.Perf[i] = &perf
	}
	for i, st := range pbcl.Storages {
		stn := pb.Datastore{
			Id:          st.Id,
			UID:         st.UID,
			StorageName: st.StorageName,
			CapacityGb:  st.CapacityGb,
			FreeSpaceGb: st.FreeSpaceGb,
			PercentUsed: st.PercentUsed}
		stn.VmKeys.Count = st.VmKeys.Count
		for _, key := range st.VmKeys.KeyRecs {
			stn.VmKeys.KeyRecs = append(stn.VmKeys.KeyRecs, key)
		}
		pbclf.Storages[i] = &stn
	}
	return &pbclf
}

// CopyClusterStorages создаёт независимую копию сообщения pb.Datastore, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту.
func CopyClusterStorages(st *pb.Datastore) *pb.Datastore {
	stn := pb.Datastore{
		Id:          st.Id,
		UID:         st.UID,
		StorageName: st.StorageName,
		CapacityGb:  st.CapacityGb,
		FreeSpaceGb: st.FreeSpaceGb,
		PercentUsed: st.PercentUsed}
	stn.VmKeys.Count = st.VmKeys.Count
	for _, key := range st.VmKeys.KeyRecs {
		stn.VmKeys.KeyRecs = append(stn.VmKeys.KeyRecs, key)
	}
	return &stn
}

// GetClusterMetrics отдаёт клиенту потоком историю метрик производительности конкретного кластера (предварительно пересчитывая агрегаты через cl.CalcPerfomance()). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
// Возвращает ошибку, если кластер с указанными vCenter/датацентром/именем не найден.
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

// CopyCLPerf создаёт независимую копию сообщения pb.CLPerf, полученного от удалённого gRPC-эндпоинта, перед пересылкой клиенту.
func CopyCLPerf(in *pb.CLPerf) *pb.CLPerf {
	clperf := pb.CLPerf{
		Id:             in.Id,
		Start:          in.Start,
		CpuUsage:       in.CpuUsage,
		CpuUtilization: in.CpuUtilization,
		MemUsage:       in.MemUsage,
		DiskUsage:      in.DiskUsage,
		DiskWrite:      in.DiskWrite,
		DiskRead:       in.DiskRead,
		NetUsage:       in.NetUsage,
		NetReceived:    in.NetReceived,
		NetTransmitted: in.NetTransmitted}
	return &clperf
}

// GetClusterMedium отдаёт клиенту потоком информацию о кластерах с агрегированными метриками производительности (без списка хостов/хранилищ). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
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
func CopyClusterMedium(pbcl *pb.Cluster) *pb.Cluster {
	pbclf := pb.Cluster{
		Id:                    pbcl.Id,
		Date:                  pbcl.Date,
		Vcenter:               pbcl.Vcenter,
		Datacenter:            pbcl.Datacenter,
		Name:                  pbcl.Name,
		CpuMhzAvg:             pbcl.CpuMhzAvg,
		AllNumCpuPkgs:         pbcl.AllNumCpuPkgs,
		AllNumCpuCores:        pbcl.AllNumCpuCores,
		AllNumCpuThreads:      pbcl.AllNumCpuThreads,
		AllMemorySizeGB:       pbcl.AllMemorySizeGB,
		AllCapacityGb:         pbcl.AllCapacityGb,
		AllFreeSpaceGb:        pbcl.AllFreeSpaceGb,
		AllPercentUsed:        pbcl.AllPercentUsed,
		AllocatedCoreCount:    pbcl.AllocatedCoreCount,
		AllocatedMemorySizeGb: pbcl.AllocatedMemorySizeGb,
		CoreTransferRate:      pbcl.CoreTransferRate,
		MemoryTransferRate:    pbcl.MemoryTransferRate,
		CpuUsageMaximum:       pbcl.CpuUsageMaximum,
		CpuUsageMaximumTime:   pbcl.CpuUsageMaximumTime,
		CpuUsageAverage:       pbcl.CpuUsageAverage,
		CpuUsageMinimum:       pbcl.CpuUsageMinimum,
		CpuUsageMinimumTime:   pbcl.CpuUsageMinimumTime,
		MemUsageMaximum:       pbcl.MemUsageMaximum,
		MemUsageMaximumTime:   pbcl.MemUsageMaximumTime,
		MemUsageAverage:       pbcl.MemUsageAverage,
		MemUsageMinimum:       pbcl.MemUsageMinimum,
		MemUsageMinimumTime:   pbcl.MemUsageMinimumTime,
		DiskUsageMaximum:      pbcl.DiskUsageMaximum,
		DiskUsageMaximumTime:  pbcl.DiskUsageMaximumTime,
		DiskUsageAverage:      pbcl.DiskUsageAverage,
		DiskUsageMinimum:      pbcl.DiskUsageMinimum,
		DiskUsageMinimumTime:  pbcl.DiskUsageMinimumTime,
		DiskWriteMaximum:      pbcl.DiskWriteMaximum,
		DiskWriteMaximumTime:  pbcl.DiskWriteMaximumTime,
		DiskWriteAverage:      pbcl.DiskWriteAverage,
		DiskWriteMinimum:      pbcl.DiskWriteMinimum,
		DiskWriteMinimumTime:  pbcl.DiskWriteMinimumTime,
		DiskReadMaximum:       pbcl.DiskReadMaximum,
		DiskReadMaximumTime:   pbcl.DiskReadMaximumTime,
		DiskReadAverage:       pbcl.DiskReadAverage,
		DiskReadMinimum:       pbcl.DiskReadMinimum,
		DiskReadMinimumTime:   pbcl.DiskReadMinimumTime,
		NetUsageMaximum:       pbcl.NetUsageMaximum,
		NetUsageMaximumTime:   pbcl.NetUsageMaximumTime,
		NetUsageAverage:       pbcl.NetUsageAverage,
		NetUsageMinimum:       pbcl.NetUsageMinimum,
		NetUsageMinimumTime:   pbcl.NetUsageMinimumTime,
		PerfCount:             pbcl.PerfCount,
		HostFlag:              pbcl.HostFlag,
		HostCount:             pbcl.HostCount,
		VMCount:               pbcl.VMCount}

	return &pbclf
}

// CopyDatastore создаёт независимую копию сообщения pb.Datastore вместе со списком ключей связанных ВМ (VmKeys).
func CopyDatastore(in *pb.Datastore) *pb.Datastore {
	out := pb.Datastore{
		Id:          in.Id,
		UID:         in.UID,
		StorageName: in.StorageName,
		CapacityGb:  in.CapacityGb,
		FreeSpaceGb: in.FreeSpaceGb,
		PercentUsed: in.PercentUsed,
	}

	out.VmKeys.Count = in.VmKeys.Count
	for _, key := range in.VmKeys.KeyRecs {
		out.VmKeys.KeyRecs = append(out.VmKeys.KeyRecs, key)
	}
	return &out
}

// GetHostFull отдаёт клиенту потоком полную информацию о хостах, отфильтрованную по vCenter/датацентру/кластеру/имени, с пересчитанными агрегированными метриками. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
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
func CopyPbHostFull(in *pb.HostFull) *pb.HostFull {
	out := pb.HostFull{
		Id:                         in.Id,
		Vcenter:                    in.Vcenter,
		Datacenter:                 in.Datacenter,
		Cluster:                    in.Cluster,
		Name:                       in.Name,
		KeyName:                    in.KeyName,
		Vendor:                     in.Vendor,
		Model:                      in.Model,
		SerialNumber:               in.SerialNumber,
		VirtualSerialNumber:        in.VirtualSerialNumber,
		EnclosureSerialNumber:      in.EnclosureSerialNumber,
		Uuid:                       in.Uuid,
		PowerState:                 in.PowerState,
		InMaintenanceMode:          in.InMaintenanceMode,
		CpuModel:                   in.CpuModel,
		CpuMhz:                     in.CpuMhz,
		NumCpuPkgs:                 in.NumCpuPkgs,
		NumCpuCores:                in.NumCpuCores,
		NumCpuThreads:              in.NumCpuThreads,
		NumNics:                    in.NumNics,
		NumHBAs:                    in.NumHBAs,
		MemorySizeGB:               in.MemorySizeGB,
		HostMaxVirtualDiskCapacity: in.HostMaxVirtualDiskCapacity,
		CpuUsageMaximum:            in.CpuUsageMaximum,
		CpuUsageMaximumTime:        in.CpuUsageMaximumTime,
		CpuUsageAverage:            in.CpuUsageAverage,
		CpuUsageMinimum:            in.CpuUsageMinimum,
		CpuUsageMinimumTime:        in.CpuUsageMinimumTime,
		MemUsageMaximum:            in.MemUsageMaximum,
		MemUsageMaximumTime:        in.MemUsageMaximumTime,
		MemUsageAverage:            in.MemUsageAverage,
		MemUsageMinimum:            in.MemUsageMinimum,
		MemUsageMinimumTime:        in.MemUsageMinimumTime,
		DiskUsageMaximum:           in.DiskUsageMaximum,
		DiskUsageMaximumTime:       in.DiskUsageMaximumTime,
		DiskUsageAverage:           in.DiskUsageAverage,
		DiskUsageMinimum:           in.DiskUsageMinimum,
		DiskUsageMinimumTime:       in.DiskUsageMinimumTime,
		DiskWriteMaximum:           in.DiskWriteMaximum,
		DiskWriteMaximumTime:       in.DiskWriteMaximumTime,
		DiskWriteAverage:           in.DiskWriteAverage,
		DiskWriteMinimum:           in.DiskWriteMinimum,
		DiskWriteMinimumTime:       in.DiskWriteMinimumTime,
		DiskReadMaximum:            in.DiskReadMaximum,
		DiskReadMaximumTime:        in.DiskReadMaximumTime,
		DiskReadAverage:            in.DiskReadAverage,
		DiskReadMinimum:            in.DiskReadMinimum,
		DiskReadMinimumTime:        in.DiskReadMinimumTime,
		NetUsageMaximum:            in.NetUsageMaximum,
		NetUsageMaximumTime:        in.NetUsageMaximumTime,
		NetUsageAverage:            in.NetUsageAverage,
		NetUsageMinimum:            in.NetUsageMinimum,
		NetUsageMinimumTime:        in.NetUsageMinimumTime,
		PerfCount:                  in.PerfCount,
		TagName:                    in.TagName,
		TagDescription:             in.TagDescription,
	}
	for _, perf := range in.Perf {
		hsperf := pb.HSPerf{
			Id:             perf.Id,
			Start:          perf.Start,
			CpuUsage:       perf.CpuUsage,
			CpuUtilization: perf.CpuUtilization,
			MemUsage:       perf.MemUsage,
			DiskUsage:      perf.DiskUsage,
			DiskWrite:      perf.DiskWrite,
			DiskRead:       perf.DiskRead,
			NetUsage:       perf.NetUsage,
			NetReceived:    perf.NetReceived,
			NetTransmitted: perf.NetTransmitted}
		out.Perf = append(out.Perf, &hsperf)
	}
	return &out
}

// GetVMFiles возвращает список файлов виртуальной машины по её ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
// При обращении к удалённым эндпоинтам ошибки подключения к отдельным эндпоинтам не прерывают обход остальных (накопленный результат всё равно возвращается вызывающему).
func (s *server) GetVMFiles(_ context.Context, in *pb.VMReqKey) (*pb.VMFileSlice, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmfiles := pb.VMFileSlice{VMFiles: make([]*pb.VMFile, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmfiles, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vmfile, err := c.GetVMFiles(ctx, in)
			if err == nil {
				for _, vmf := range vmfile.VMFiles {
					vmfiles.VMFiles = append(vmfiles.VMFiles, CopyPbVMFile(vmf))
					vmfiles.Count++
				}
			}
		}
		return &vmfiles, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	vm, err := infra.GetVMKey(key)
	if err != nil {
		return &pb.VMFileSlice{}, err
	}
	out := pb.VMFileSlice{}
	out.Count = int64(len(vm.Files))
	for _, f := range vm.Files {
		fn := pb.VMFile{}
		fn.Type = f.Type
		fn.Name = f.Name
		fn.Size = f.Size
		fn.UniqueSize = f.UniqueSize
		out.VMFiles = append(out.VMFiles, &fn)
	}
	return &out, nil
}

// CopyPbVMFile создаёт независимую копию сообщения pb.VMFile.
func CopyPbVMFile(in *pb.VMFile) *pb.VMFile {
	out := pb.VMFile{
		Type:       in.Type,
		Name:       in.Name,
		Size:       in.Size,
		UniqueSize: in.UniqueSize,
	}
	return &out
}

// GetVMGuestDiskInfo возвращает информацию о дисках гостевой ОС виртуальной машины по её ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMGuestDiskInfo(_ context.Context, in *pb.VMReqKey) (*pb.GuestDiskSlice, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmdisks := pb.GuestDiskSlice{GuestDisks: make([]*pb.GuestDiskInfo, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmdisks, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vmdisk, err := c.GetVMGuestDiskInfo(ctx, in)
			if err == nil {
				for _, dsk := range vmdisk.GuestDisks {
					vmdisks.GuestDisks = append(vmdisks.GuestDisks, dsk)
					vmdisks.Count++
				}
			}
		}
		return &vmdisks, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	vm, err := infra.GetVMKey(key)
	if err != nil {
		return &pb.GuestDiskSlice{}, err
	}
	out := pb.GuestDiskSlice{}
	out.Count = int64(len(vm.Disk))
	for _, d := range vm.Disk {
		dn := pb.GuestDiskInfo{}
		dn.CapacityGb = d.CapacityGb
		dn.DiskPath = d.DiskPath
		dn.FreeSpaceGb = d.FreeSpaceGb
		dn.PercentUsed = d.PercentUsed
		out.GuestDisks = append(out.GuestDisks, &dn)
	}
	return &out, nil
}

// GetVMUsageOnDatastore возвращает данные об использовании хранилищ виртуальной машиной по её ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMUsageOnDatastore(_ context.Context, in *pb.VMReqKey) (*pb.VMUsageOnDatastoreSlice, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmdisks := pb.VMUsageOnDatastoreSlice{UsageOnDatastores: make([]*pb.VMUsageOnDatastore, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmdisks, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vmdisk, err := c.GetVMUsageOnDatastore(ctx, in)
			if err == nil {
				for _, dsk := range vmdisk.UsageOnDatastores {
					vmdisks.UsageOnDatastores = append(vmdisks.UsageOnDatastores, dsk)
					vmdisks.Count++
				}
			}
		}
		return &vmdisks, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	vm, err := infra.GetVMKey(key)
	if err != nil {
		return &pb.VMUsageOnDatastoreSlice{}, err
	}
	out := pb.VMUsageOnDatastoreSlice{}
	out.Count = int64(len(vm.UsageDatastore))
	for _, d := range vm.UsageDatastore {
		dn := pb.VMUsageOnDatastore{}
		dn.Id = d.ID
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
		out.UsageOnDatastores = append(out.UsageOnDatastores, &dn)
	}
	return &out, nil
}

// GetVMByKey возвращает полную информацию о виртуальной машине по её ключу. Если сконфигурированы удалённые
// обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные
// берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
// Возвращает ошибку errNotFound, если ни на одном удалённом эндпоинте ВМ не найдена, либо ошибку infra.GetVMKey
// при локальном поиске.
func (s *server) GetVMByKey(_ context.Context, in *pb.VMReqKey) (*pb.VMPoint, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.VMPoint{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vm, err := c.GetVMByKey(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return vm, nil
			}
		}
		return &pb.VMPoint{}, errors.New(errNotFound) //ничего не найдено
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	vm, err := infra.GetVMKey(key)
	if err != nil {
		return &pb.VMPoint{}, err
	}
	out := pb.VMPoint{}

	VMToMessageFull(vm, &out)

	return &out, nil
}

// GetVMsSliceByKeyList возвращает срез с полной информацией о нескольких виртуальных машинах по списку ключей. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMsSliceByKeyList(_ context.Context, in *pb.VMListKey) (*pb.VMPointSlice, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmsall := pb.VMPointSlice{VMPoints: make([]*pb.VMPoint, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmsall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetVMsSliceByKeyList(ctx, in)
			if err == nil {
				for _, vm := range vms.VMPoints {
					vmsall.VMPoints = append(vmsall.VMPoints, vm)
					vmsall.Count++
				}
			}
		}
		return &vmsall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	outSlice := pb.VMPointSlice{}
	for _, key := range in.KeyRecs {
		vm, err := infra.GetVMKey(key.Key)
		if err != nil {
			return &outSlice, err
		}
		out := pb.VMPoint{}
		VMToMessageFull(vm, &out)
		outSlice.VMPoints = append(outSlice.VMPoints, &out)
		outSlice.Count++
	}
	return &outSlice, nil
}

// GetVMStreamByKeyList отдаёт клиенту потоком полную информацию о виртуальных машинах по списку ключей. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMStreamByKeyList(in *pb.VMListKey, stream grpc.ServerStreamingServer[pb.VMPoint]) error {
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
			streamcl, err := c.GetVMStreamByKeyList(ctx, in)
			if err == nil {
				for {
					vm, err := streamcl.Recv()
					if err == io.EOF {
						break
					}
					if err := stream.Send(vm); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	for _, key := range in.KeyRecs {
		vm, err := infra.GetVMKey(key.Key)
		if err != nil {
			return err
		}
		out := pb.VMPoint{}
		VMToMessageFull(vm, &out)
		if err := stream.Send(&out); err != nil {
			_ = err
		}
	}
	return nil
}

// GetVMShortByKey возвращает укороченную информацию о виртуальной машине по её ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMShortByKey(_ context.Context, in *pb.VMReqKey) (*pb.VMPointShort, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.VMPointShort{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vm, err := c.GetVMShortByKey(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return vm, nil
			}
		}
		return &pb.VMPointShort{}, errors.New(errNotFound) //ничего не найдено
	}
	infra := vsphere.GetGlobalVMInfrastructure()
	key := in.GetKey()
	vm, err := infra.GetVMKey(key)
	if err != nil {
		return &pb.VMPointShort{}, err
	}
	out := pb.VMPointShort{}
	VMToMessageShort(vm, &out)
	return &out, nil
}

// GetVMListKey возвращает список ключей всех виртуальных машин, соответствующих фильтру запроса. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMListKey(_ context.Context, in *pb.VMReqListKey) (*pb.VMListKey, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmsall := pb.VMListKey{KeyRecs: make([]*pb.VMKeyRec, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmsall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetVMListKey(ctx, in)
			if err == nil {
				for _, rec := range vms.KeyRecs {
					vmsall.KeyRecs = append(vmsall.KeyRecs, rec)
					vmsall.Count++
				}
			}
		}
		return &vmsall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	vcenter := in.GetVcenter()
	datacenter := in.GetDatacenter()
	cluster := in.GetCluster()
	vmname := in.GetVMName()

	out := pb.VMListKey{}

	if infra == nil {
		return &out, errors.New(errNotInitialized)
	}

	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if vcenter != "all" && vcenter != "" && vm.Vcenter != vcenter {
			continue
		}
		if datacenter != "all" && datacenter != "" && vm.Datacenter != datacenter {
			continue
		}
		if cluster != "all" && cluster != "" && vm.Cluster != cluster {
			continue
		}
		if vmname != "all" && vmname != "" && vm.Name != vmname {
			continue
		}

		out.KeyRecs = append(out.KeyRecs, &pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName})
		out.Count++
	}
	return &out, nil
}

// GetVMListKeyByVMName возвращает список ключей виртуальных машин, соответствующих имени ВМ (с фильтрацией по маске/точному совпадению). Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMListKeyByVMName(_ context.Context, in *pb.VMReqVMName) (*pb.VMListKey, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmsall := pb.VMListKey{KeyRecs: make([]*pb.VMKeyRec, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmsall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetVMListKeyByVMName(ctx, in)
			if err == nil {
				for _, rec := range vms.KeyRecs {
					vmsall.KeyRecs = append(vmsall.KeyRecs, rec)
					vmsall.Count++
				}
			}
		}
		return &vmsall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	vmname := in.GetVMName()
	out := pb.VMListKey{}

	if infra == nil {
		return &out, errors.New(errNotInitialized)
	}

	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if vmname != "all" && vmname != "" && vm.Name != vmname {
			continue
		}
		out.KeyRecs = append(out.KeyRecs, &pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName})
		out.Count++
	}
	return &out, nil
}

// GetVMListKeyByCluster возвращает список ключей виртуальных машин, принадлежащих указанному кластеру. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMListKeyByCluster(_ context.Context, in *pb.VMReqCluster) (*pb.VMListKey, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmsall := pb.VMListKey{KeyRecs: make([]*pb.VMKeyRec, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmsall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetVMListKeyByCluster(ctx, in)
			if err == nil {
				for _, rec := range vms.KeyRecs {
					vmsall.KeyRecs = append(vmsall.KeyRecs, rec)
					vmsall.Count++
				}
			}
		}
		return &vmsall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	cluster := in.GetCluster()
	out := pb.VMListKey{}

	if infra == nil {
		return &out, errors.New(errNotInitialized)
	}

	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if cluster != "all" && cluster != "" && vm.Cluster != cluster {
			continue
		}

		out.KeyRecs = append(out.KeyRecs, &pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName})
		out.Count++
	}
	return &out, nil
}

// GetVMListKeyByVcenter возвращает список ключей виртуальных машин, принадлежащих указанному vCenter. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMListKeyByVcenter(_ context.Context, in *pb.VMReqVcenter) (*pb.VMListKey, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmsall := pb.VMListKey{KeyRecs: make([]*pb.VMKeyRec, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmsall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetVMListKeyByVcenter(ctx, in)
			if err == nil {
				for _, rec := range vms.KeyRecs {
					vmsall.KeyRecs = append(vmsall.KeyRecs, rec)
					vmsall.Count++
				}
			}
		}
		return &vmsall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	vcenter := in.GetVcenter()
	out := pb.VMListKey{}

	if infra == nil {
		return &out, errors.New(errNotInitialized)
	}

	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if vcenter != "all" && vcenter != "" && vm.Vcenter != vcenter {
			continue
		}

		out.KeyRecs = append(out.KeyRecs, &pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName})
		out.Count++
	}
	return &out, nil
}

// GetVMListKeyByDatacenter возвращает список ключей виртуальных машин, принадлежащих указанному датацентру. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
func (s *server) GetVMListKeyByDatacenter(_ context.Context, in *pb.VMReqDatacenter) (*pb.VMListKey, error) {
	if config.Switch.Endpoint.Count() > 0 {
		vmsall := pb.VMListKey{KeyRecs: make([]*pb.VMKeyRec, 0)}

		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &vmsall, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			vms, err := c.GetVMListKeyByDatacenter(ctx, in)
			if err == nil {
				for _, rec := range vms.KeyRecs {
					vmsall.KeyRecs = append(vmsall.KeyRecs, rec)
					vmsall.Count++
				}
			}
		}
		return &vmsall, nil
	}
	infra := vsphere.GetGlobalVMInfrastructure()

	datacenter := in.GetDatacenter()
	out := pb.VMListKey{}

	if infra == nil {
		return &out, errors.New(errNotInitialized)
	}

	for _, key := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		if datacenter != "all" && datacenter != "" && vm.Datacenter != datacenter {
			continue
		}

		out.KeyRecs = append(out.KeyRecs, &pb.VMKeyRec{VMName: vm.Name, Key: vm.KeyName})
		out.Count++
	}
	return &out, nil
}
// GetHostListKey возвращает список ключей хостов, соответствующих фильтру запроса. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
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
		perf := protocol.HSPerf{}
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
		if err := stream.Send(&perf); err != nil {
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
		perf := protocol.HSPerf{}
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

// GetClusterPerfByKey возвращает историю метрик производительности кластера по его ключу единым срезом. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
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

// GetClusterByKey возвращает информацию о кластере с агрегированными метриками по его ключу. Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос ретранслируется на каждый из них через gRPC-клиент, иначе данные берутся из локальной инфраструктуры vsphere.GetGlobalVMInfrastructure().
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
func compact(src []*time.Time, start, end time.Time) []*time.Time {

	s := -1
	e := -1
	dst := make([]*time.Time, 0)

	for i := range src {
		if src[i].Unix() >= start.Unix() && s == -1 {
			s = i

		} else if s == -1 {
			t := *src[i]
			dst = append(dst, &t)
		}
		if src[i].Unix() >= end.Unix() && e == -1 {
			e = i
		}
		if s >= 0 && e >= 0 {
			t := *src[i]
			dst = append(dst, &t)
		}
	}
	return dst
}

// clearCLPerf возвращает подсрез записей метрик кластера src, чьи временные метки Start попадают в диапазон [start, end).
func clearCLPerf(src []*vsphere.CLPerf, start, end time.Time) []*vsphere.CLPerf {
	s := -1
	e := -1
	dst := make([]*vsphere.CLPerf, 0)

	for i := range src {
		if src[i].Start.Unix() >= start.Unix() && s == -1 {
			s = i
		} else if s == -1 {
			t := *src[i]
			dst = append(dst, &t)
		}
		if src[i].Start.Unix() >= end.Unix() && e == -1 {
			e = i
		}
		if s >= 0 && e >= 0 {
			t := *src[i]
			dst = append(dst, &t)
		}
	}
	return dst
}

// clearHSPerf возвращает подсрез записей метрик хоста src, чьи временные метки Start попадают в диапазон [start, end).
func clearHSPerf(src []*vsphere.HSPerf, start, end time.Time) []*vsphere.HSPerf {
	s := -1
	e := -1
	dst := make([]*vsphere.HSPerf, 0)

	for i := range src {
		if src[i].Start.Unix() >= start.Unix() && s == -1 {
			s = i
		} else if s == -1 {
			t := *src[i]
			dst = append(dst, &t)
		}
		if src[i].Start.Unix() >= end.Unix() && e == -1 {
			e = i
		}
		if s >= 0 && e >= 0 {
			t := *src[i]
			dst = append(dst, &t)
		}
	}
	return dst
}

// clearVMPerf возвращает подсрез записей метрик виртуальной машины src, чьи временные метки Start попадают в диапазон [start, end). Параметр vm в текущей реализации не используется.
func clearVMPerf(vm *vsphere.VMPoint, src []*vsphere.VMPerf, start, end time.Time) []*vsphere.VMPerf {
	s := -1
	e := -1
	dst := make([]*vsphere.VMPerf, 0)

	for i := range src {
		if src[i].Start.Unix() >= start.Unix() && s == -1 {
			s = i
		} else if s == -1 {
			t := *src[i]
			vsphere.ReCalcVMPerfomance(vm, t)
			dst = append(dst, &t)
		}
		if src[i].Start.Unix() >= end.Unix() && e == -1 {
			e = i
		}
		if s >= 0 && e >= 0 {
			t := *src[i]
			vsphere.ReCalcVMPerfomance(vm, t)
			dst = append(dst, &t)
		}
	}
	return dst
}

// !!!! добавить пересчет метрик в записях кластера, хоста и виртуальной машины? после удаления части метрик
// пересчет максимального и минимального значения, накопление среднего.
func (p *server) ClearMetric(_ context.Context, in *pb.Duration) (*pb.StatusWork, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.StatusWork{State: -1, Details: "did not connect" + err.Error()}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			st, err := c.ClearMetric(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return st, nil
			}
		}
		return &pb.StatusWork{State: -2, Details: "not found"}, errors.New(errNotFound) //ничего не найдено
	}
	start := in.Start.AsTime()
	end := in.End.AsTime()

	infra := vsphere.GetGlobalVMInfrastructure()

	infra.LockGetMetric()
	defer infra.UnLockGetMetric()

	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if in.Cluster != "all" && in.Cluster != "" {
			if cl.Name != in.Cluster {
				continue
			}
		}
		cl.Perf = clearCLPerf(cl.Perf, start, end)
		cl.PerfCount = int64(len(cl.Perf))
	}

	for _, keyHS := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(keyHS)
		if !ok {
			continue
		}

		if in.Cluster != "all" && in.Cluster != "" {
			if hs.Cluster != in.Cluster {
				continue
			}
		}

		hs.Perf = clearHSPerf(hs.Perf, start, end)
		hs.PerfCount = int64(len(hs.Perf))
	}

	for _, keyVM := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(keyVM)
		if !ok {
			continue
		}

		if in.Cluster != "all" && in.Cluster != "" {
			if vm.Cluster != in.Cluster {
				continue
			}
		}
		vm.VMClearPerfomance()
		vm.Perf = clearVMPerf(vm, vm.Perf, start, end)
		vm.PerfCount = int64(len(vm.Perf))
	}

	st := pb.StatusWork{State: 0, Details: "ok"}
	return &st, nil
}

// !!!! добавить пересчет метрик в записях кластера, хоста и виртуальной машины? после удаления части метрик
//пересчет максимального и минимального значения, накопление среднего.

// ClearAllMetricByKey полностью очищает историю метрик (кластера, хоста или ВМ, в зависимости от типа ключа) для объекта с указанным ключом. Возвращает статус выполнения операции.
func (p *server) ClearAllMetricByKey(_ context.Context, in *pb.ClearMetricKey) (*pb.StatusWork, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}

			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.StatusWork{State: -1, Details: "did not connect" + err.Error()}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			st, err := c.ClearAllMetricByKey(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return st, nil
			}
		}
		return &pb.StatusWork{State: -2, Details: "not found"}, errors.New(errNotFound) //ничего не найдено
	}
	start := in.Start.AsTime()
	end := in.End.AsTime()

	infra := vsphere.GetGlobalVMInfrastructure()

	infra.LockGetMetric()
	defer infra.UnLockGetMetric()

	for _, key := range infra.ClusterMapList.Keys() {
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok {
			continue
		}
		if cl.Vcenter != in.Vcenter && cl.Vcenter != "all" {
			continue
		}
		if cl.Datacenter != in.Datacenter && cl.Datacenter != "all" {
			continue
		}
		if cl.Name != in.Cluster && cl.Name != "all" {
			continue
		}

		cl.Perf = clearCLPerf(cl.Perf, start, end)
		cl.PerfCount = int64(len(cl.Perf))
	}

	for _, keyHS := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(keyHS)
		if !ok {
			continue
		}
		if hs.Vcenter != in.Vcenter && hs.Vcenter != "all" {
			continue
		}
		if hs.Datacenter != in.Datacenter && hs.Datacenter != "all" {
			continue
		}
		if hs.Cluster != in.Cluster && hs.Cluster != "all" {
			continue
		}

		hs.Perf = clearHSPerf(hs.Perf, start, end)
		hs.PerfCount = int64(len(hs.Perf))
	}

	for _, keyVM := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(keyVM)
		if !ok {
			continue
		}
		if vm.Vcenter != in.Vcenter && vm.Vcenter != "all" {
			continue
		}
		if vm.Datacenter != in.Datacenter && vm.Datacenter != "all" {
			continue
		}
		if vm.Cluster != in.Cluster && vm.Cluster != "all" {
			continue
		}

		vm.VMClearPerfomance()
		vm.Perf = clearVMPerf(vm, vm.Perf, start, end)
		vm.PerfCount = int64(len(vm.Perf))
	}

	st := pb.StatusWork{State: 0, Details: "ok"}
	return &st, nil
}

// ClearMetricByKey очищает историю метрик объекта (кластера, хоста или ВМ) с указанным ключом, оставляя только записи внутри переданного интервала времени. Возвращает статус выполнения операции.
func (p *server) ClearMetricByKey(_ context.Context, in *pb.ClearMetricKey) (*pb.StatusWork, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}

			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.StatusWork{State: -1, Details: "did not connect" + err.Error()}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			st, err := c.ClearMetricByKey(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return st, nil
			}
		}
		return &pb.StatusWork{State: -2, Details: "not found"}, errors.New(errNotFound) //ничего не найдено
	}
	start := in.Start.AsTime()
	end := in.End.AsTime()

	infra := vsphere.GetGlobalVMInfrastructure()

	infra.LockGetMetric()
	defer infra.UnLockGetMetric()

	for _, keyVM := range infra.VMMapList.Keys() {
		vm, ok := infra.VMMapList.Get(keyVM)
		if !ok {
			continue
		}
		if vm.Vcenter != in.Vcenter && in.Vcenter != "all" {
			continue
		}
		if vm.Datacenter != in.Datacenter && in.Datacenter != "all" {
			continue
		}
		if vm.Cluster != in.Cluster && in.Cluster != "all" {
			continue
		}

		vm.VMClearPerfomance()
		vm.Perf = clearVMPerf(vm, vm.Perf, start, end)
		vm.PerfCount = int64(len(vm.Perf))
	}

	st := pb.StatusWork{State: 0, Details: "ok"}
	return &st, nil
}

//HP OneView
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
			if in.PowerState != "all" && in.PowerState != hw.Base.PowerState {
				continue
			}
			if in.Status != "all" && in.Status != hw.Base.Status {
				continue
			}
			if in.MaintenanceMode != "all" {
				b, err := strconv.ParseBool(in.MaintenanceMode)
				if err == nil && b != hw.Base.MaintenanceMode {
					continue
				}
			}
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
func (s *serverdb) GetClusterSprRecords(in *pb.ClusterSprKey1, stream grpc.ServerStreamingServer[pb.ClusterSprRecord]) error {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			streamhw, err := c.GetClusterSprRecords(ctx, in)
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
				return nil //Как только выполняется запрос прекращаем отправку остальным
			}
		}
		return nil
	}

	o := orm.NewOrm()

	qs := o.QueryTable("cluster_spr_records")
	qs = qs.OrderBy("Vcenter", "Datacenter", "Name")
	cond := orm.NewCondition()

	if in.Vcenter != "all" {
		cond = cond.And("v_center", in.Vcenter)
	}
	if in.Datacenter != "all" {
		cond = cond.And("v_datacenter", in.Datacenter)
	}

	if in.Cluster != "all" {
		cond = cond.And("name", in.GetCluster())
	}

	qs = qs.SetCond(cond)
	clusters := make([]*models.ClusterSprRecords, 0)

	num, err := qs.All(&clusters)
	infra := vsphere.GetGlobalVMInfrastructure()
	if num > 0 && err == nil {
		for i := range clusters {
			clspr := models.ClusterSprRecords{Id: clusters[i].Id}
			err := o.Read(&clspr, "Id")
			if clspr.Stat != 2 {
				clspr.Stat = 0
			}
			key := fmt.Sprintf("%s/%s/%s", clusters[i].Vcenter, clusters[i].Datacenter, clusters[i].Name)
			cl, ok := infra.ClusterMapList.Get(key)
			if ok {
				clusters[i].TypeWork = cl.TypeWork
				clusters[i].Stat = 1
				clspr.TypeWork = cl.TypeWork
				clspr.Stat = 1
			}
			if err == nil {
				o.Update(&clspr)
			}
			out := pb.ClusterSprRecord{}
			ClusterSprRecordsToMessage(clusters[i], &out)
			if err := stream.Send(&out); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddClusterSprRecord добавляет новую запись в справочник кластеров БД и возвращает присвоенный идентификатор.
// Возвращает ошибку errAddClusterSpr, если вставка записи не удалась.
func (p *serverdb) AddClusterSprRecord(_ context.Context, in *pb.ClusterSprRecord) (*pb.ClusterSprId, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ClusterSprId{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			id, err := c.AddClusterSprRecord(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterSprId{}, errors.New(errAddClusterSpr) //ничего не найдено
	}
	var err error
	clusterSprId := pb.ClusterSprId{}
	clusterSpr := models.ClusterSprRecords{}
	o := orm.NewOrm()
	MessageToClusterSprRecords(in, &clusterSpr)
	if clusterSprId.Id, err = o.Insert(&clusterSpr); err == nil {
		clusterSprId.Status = 1
		return &clusterSprId, nil
	}
	clusterSprId.Status = 2
	return &clusterSprId, errors.New(errAddClusterSpr)
}

// UpdateClusterSprRecord обновляет существующую запись в справочнике кластеров БД и возвращает её идентификатор.
func (p *serverdb) UpdateClusterSprRecord(_ context.Context, in *pb.ClusterSprRecord) (*pb.ClusterSprId, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return &pb.ClusterSprId{}, err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			id, err := c.UpdateClusterSprRecord(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return id, nil
			}
		}
		return &pb.ClusterSprId{}, errors.New(errAddClusterSpr) //ничего не найдено
	}
	var err error
	clusterSprId := pb.ClusterSprId{}
	clusterSpr := models.ClusterSprRecords{}
	o := orm.NewOrm()
	clusterSpr.Vcenter = in.Vcenter
	clusterSpr.Datacenter = in.Datacenter
	clusterSpr.Name = in.Name
	err = o.Read(&clusterSpr, "Vcenter", "Datacenter", "Name")
	clusterSprId = pb.ClusterSprId{}
	if err == nil {
		clusterSprId.Id = clusterSpr.Id
		MessageToClusterSprRecords(in, &clusterSpr)
		if _, err = o.Update(&clusterSpr); err == nil {
			clusterSprId.Status = 1
			return &clusterSprId, nil
		}
	}
	clusterSprId.Status = 2
	return &clusterSprId, errors.New(errAddClusterSpr)
}

// для всех
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
