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
	"loadmon/config"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
