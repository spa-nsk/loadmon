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
	"net/url"
	"sort"
	"strconv"
	"time"

	oneview "github.com/spa-nsk/oneview2"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func hardwareGrpcTimeout() time.Duration {
	return config.GetGrpcHardwareTimeout()
}

// hwHasPlacement сообщает, содержит ли ответ данные о размещении сервера
// (датацентр или стойка). Используется при опросе нескольких зарегистрированных
// в Switch воркеров: один и тот же физический сервер может быть виден сразу
// нескольким воркерам (например, если он присутствует и в Global Dashboard, и
// на конкретной appliance) — и не у всех такой ответ будет одинаково полным
// (GD, в частности, не отдаёт /rest/racks и /rest/datacenters, поэтому
// размещение остаётся пустым при опросе через GD). Раньше функции
// GetServerHardwareBySN/ByName/ByVirtualSN возвращали ответ ПЕРВОГО
// откликнувшегося воркера, даже если он был неполным - из-за этого
// размещение на /sprsrv/srvhwviewbysn/ иногда пропадало, хотя на
// /sprsrv/hwviewlist (который агрегирует данные со всех воркеров через
// потоковый GetServerHardwareList) оно отображалось.
func hwHasPlacement(hw *pb.ServerHardware) bool {
	if hw == nil {
		return false
	}
	if hw.Datacenter != nil && hw.Datacenter.Name != "" {
		return true
	}
	if hw.Rack != nil && hw.Rack.Name != "" {
		return true
	}
	return false
}

func (p *server) GetServerHardwareBySN(_ context.Context, in *pb.SerialNumber) (*pb.ServerHardware, error) {
	if config.Switch.Endpoint.Count() > 0 {
		var best *pb.ServerHardware
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				continue
			}
			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), hardwareGrpcTimeout())
			hw, err := c.GetServerHardwareBySN(ctx, in)
			cancel()
			conn.Close()
			if err != nil {
				continue
			}
			if best == nil {
				best = hw
			}
			if hwHasPlacement(hw) { //нашли воркера с полными данными о размещении - дальше не ищем
				best = hw
				break
			}
		}
		if best != nil {
			return best, nil
		}
		return &pb.ServerHardware{}, fmt.Errorf(errSrvNotFound, in.SerialNumber) //ничего не найдено
	}
	ov := oneview.GetGlobalOVInfrastructure()
	// Примечание: раньше здесь читали ov.ServerBySN[in.SerialNumber] напрямую,
	// без блокировки - в момент, когда фоновое обновление данных
	// (Infrastructure.LoadServerHardwareList, тикер раз в 6 часов и при
	// старте) перестраивает и переприсваивает эту карту под
	// LockGetData/UnLockGetData, конкурентное чтение могло попасть на
	// частично обновлённое состояние. FindServerHardwareSN дожидается снятия
	// блокировки перед чтением (как и было задумано в oneview2/global.go).
	hw, err := ov.FindServerHardwareSN(in.SerialNumber)
	if err == nil {
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
		var best *pb.ServerHardware
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				continue
			}
			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), hardwareGrpcTimeout())
			hw, err := c.GetServerHardwareByName(ctx, in)
			cancel()
			conn.Close()
			if err != nil {
				continue
			}
			if best == nil {
				best = hw
			}
			if hwHasPlacement(hw) {
				best = hw
				break
			}
		}
		if best != nil {
			return best, nil
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
		var best *pb.ServerHardware
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				continue
			}
			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), hardwareGrpcTimeout())
			hw, err := c.GetServerHardwareByVirtualSN(ctx, in)
			cancel()
			conn.Close()
			if err != nil {
				continue
			}
			if best == nil {
				best = hw
			}
			if hwHasPlacement(hw) {
				best = hw
				break
			}
		}
		if best != nil {
			return best, nil
		}
		return &pb.ServerHardware{}, fmt.Errorf(errSrvVirtualNotFound, in.SerialNumber) //ничего не найдено
	}
	ov := oneview.GetGlobalOVInfrastructure()
	ov.WaitUnLockGetData() // см. комментарий в GetServerHardwareBySN про гонку с LoadServerHardwareList
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
			ctx, cancel := context.WithTimeout(context.Background(), hardwareGrpcTimeout())
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
				stream.SetTrailer(streamhw.Trailer()) // состояние сбора OneView на воркере (G)
			}
		}
		return nil
	}
	ov := oneview.GetGlobalOVInfrastructure()
	ov.WaitUnLockGetData() // см. комментарий в GetServerHardwareBySN про гонку с LoadServerHardwareList

	// G (2026-10-05): состояние сбора OneView передаётся клиенту в trailer-
	// метаданных потока (без изменения .proto) - см. OVStateTrailer.
	stream.SetTrailer(OVStateTrailer(ov.State()))

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
			if in.MaintenanceMode != "all" {
				want := in.MaintenanceMode == "true"
				if hw.Identity.MaintenanceMode != want {
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

// enclosureSummaryToMessage конвертирует oneview2.EnclosureSummary (агрегат
// по блейд-корзине, собранный со стороны серверов - см. oneview2/bladename.go)
// в gRPC-сообщение pb.BladeEnclosure.
func enclosureSummaryToMessage(es *oneview.EnclosureSummary) *pb.BladeEnclosure {
	if es == nil {
		return nil
	}
	out := &pb.BladeEnclosure{
		Name:          es.Name,
		SerialNumber:  es.SerialNumber,
		Model:         es.Model,
		EnclosureType: es.EnclosureType,
	}
	bays := make([]int, 0, len(es.ByBay))
	for bay := range es.ByBay {
		bays = append(bays, bay)
	}
	sort.Ints(bays)
	for _, bay := range bays {
		srv := es.ByBay[bay]
		ref := &pb.BladeBayRef{BayNumber: int32(bay)}
		if srv != nil {
			ref.ServerName = srv.Identity.Name
			ref.ServerSerialNumber = srv.Identity.SerialNumber
		}
		out.Bays = append(out.Bays, ref)
	}
	return out
}

// GetBladeEnclosureByName возвращает сводную информацию по блейд-корзине
// (см. pb.BladeEnclosure) по её нормализованному имени ("c7000-09"), собранную
// со стороны серверов корзины - oneview2.Infrastructure.EnclosureByName (см.
// oneview2/global.go, oneview2/bladename.go). Задел под будущие доработки.
func (p *server) GetBladeEnclosureByName(_ context.Context, in *pb.ServerName) (*pb.BladeEnclosure, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				continue
			}
			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), hardwareGrpcTimeout())
			enc, err := c.GetBladeEnclosureByName(ctx, in)
			cancel()
			conn.Close()
			if err == nil && enc != nil {
				return enc, nil
			}
		}
		return &pb.BladeEnclosure{}, fmt.Errorf(errBladeNotFoundName, in.Name)
	}
	ov := oneview.GetGlobalOVInfrastructure()
	ov.WaitUnLockGetData() // см. комментарий в GetServerHardwareBySN про гонку с LoadServerHardwareList
	if ov.EnclosureByName != nil {
		if es, ok := ov.EnclosureByName[in.Name]; ok {
			return enclosureSummaryToMessage(es), nil
		}
	}
	return nil, fmt.Errorf(errBladeNotFoundName, in.Name)
}

// GetBladeEnclosureBySN — то же самое, что GetBladeEnclosureByName, но поиск
// по серийному номеру корзины (oneview2.Infrastructure.EnclosureBySN) -
// известен, только если для хотя бы одного сервера корзины отработал
// GetEnclosure (см. комментарий у Infrastructure.EnclosureBySN).
func (p *server) GetBladeEnclosureBySN(_ context.Context, in *pb.SerialNumber) (*pb.BladeEnclosure, error) {
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				continue
			}
			c := pb.NewLoadmonClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), hardwareGrpcTimeout())
			enc, err := c.GetBladeEnclosureBySN(ctx, in)
			cancel()
			conn.Close()
			if err == nil && enc != nil {
				return enc, nil
			}
		}
		return &pb.BladeEnclosure{}, fmt.Errorf(errBladeNotFoundSN, in.SerialNumber)
	}
	ov := oneview.GetGlobalOVInfrastructure()
	ov.WaitUnLockGetData() // см. комментарий в GetServerHardwareBySN про гонку с LoadServerHardwareList
	if ov.EnclosureBySN != nil {
		if es, ok := ov.EnclosureBySN[in.SerialNumber]; ok {
			return enclosureSummaryToMessage(es), nil
		}
	}
	return nil, fmt.Errorf(errBladeNotFoundSN, in.SerialNumber)
}

// GetHWDatacenterNames отдаёт клиенту список всех различных значений
// Datacenter.Name, накопленных oneview2.Infrastructure по всем собранным
// серверам (см. oneview2/global.go, DatacenterNameList) - в отличие от
// GetServerHardwareList, не зависит от текущего фильтра. В режиме Switch
// (несколько зарегистрированных воркеров) списки со всех воркеров
// объединяются, чтобы не потерять значения, известные только части
// воркеров (например, разным appliance/Global Dashboard).
func (s *server) GetHWDatacenterNames(ctx context.Context, in *pb.Empty) (*pb.NameList, error) {
	if config.Switch.Endpoint.Count() > 0 {
		return mergedHWNameList(ctx, func(c pb.LoadmonClient, ctx context.Context) (*pb.NameList, error) {
			return c.GetHWDatacenterNames(ctx, in)
		})
	}
	ov := oneview.GetGlobalOVInfrastructure()
	return &pb.NameList{Names: ov.DatacenterNameList()}, nil
}

// GetHWRackNames отдаёт клиенту список всех различных значений Rack.Name,
// накопленных oneview2.Infrastructure по всем собранным серверам - см.
// GetHWDatacenterNames.
func (s *server) GetHWRackNames(ctx context.Context, in *pb.Empty) (*pb.NameList, error) {
	if config.Switch.Endpoint.Count() > 0 {
		return mergedHWNameList(ctx, func(c pb.LoadmonClient, ctx context.Context) (*pb.NameList, error) {
			return c.GetHWRackNames(ctx, in)
		})
	}
	ov := oneview.GetGlobalOVInfrastructure()
	return &pb.NameList{Names: ov.RackNameList()}, nil
}

// mergedHWNameList опрашивает всех зарегистрированных в Switch воркеров вызовом
// call и объединяет полученные списки имён в один отсортированный список без
// повторов - используется GetHWDatacenterNames/GetHWRackNames в режиме
// нескольких воркеров.
func mergedHWNameList(_ context.Context, call func(c pb.LoadmonClient, ctx context.Context) (*pb.NameList, error)) (*pb.NameList, error) {
	seen := make(map[string]struct{})
	for _, key := range config.Switch.Endpoint.Keys() {
		endpoint, ok := config.Switch.Endpoint.Get(key)
		if !ok {
			continue
		}
		conn, err := grpc.NewClient(endpoint.GetSrvApiAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("did not connect: %v\n", err)
			continue
		}
		c := pb.NewLoadmonClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), hardwareGrpcTimeout())
		resp, err := call(c, ctx)
		cancel()
		conn.Close()
		if err != nil {
			continue
		}
		for _, name := range resp.Names {
			if name != "" {
				seen[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return &pb.NameList{Names: out}, nil
}

//------------------- DB interface -------------------

// GetClusterSprRecords (реализация LoadmonDBServer) отдаёт клиенту потоком записи справочника кластеров из БД
// по фильтру ключа. Используется отдельным сервером БД (см. StartDBServerAddr).

// Ключи trailer-метаданных GetServerHardwareList (состояние сбора OneView).
const (
	TrailerOVRefreshing  = "x-ov-refreshing"   // "1" - сейчас идёт сбор
	TrailerOVLastRefresh = "x-ov-last-refresh"  // unix-время последнего успешного снимка, "0" - ещё не было
	TrailerOVLastError   = "x-ov-last-error"    // текст ошибки последнего сбора (может быть пустым)
	TrailerOVCount       = "x-ov-server-count"  // число серверов в снимке
)

// OVStateTrailer формирует trailer-метаданные потока по состоянию сбора.
// Текст ошибки приводится к ASCII-безопасному виду через url.QueryEscape,
// т.к. значения gRPC-метаданных не должны содержать непечатаемых символов.
func OVStateTrailer(st oneview.OVState) metadata.MD {
	refreshing := "0"
	if st.Refreshing {
		refreshing = "1"
	}
	last := "0"
	if !st.LastRefresh.IsZero() {
		last = strconv.FormatInt(st.LastRefresh.Unix(), 10)
	}
	return metadata.Pairs(
		TrailerOVRefreshing, refreshing,
		TrailerOVLastRefresh, last,
		TrailerOVLastError, url.QueryEscape(st.LastError),
		TrailerOVCount, strconv.Itoa(st.ServerCount),
	)
}
