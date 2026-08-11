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
