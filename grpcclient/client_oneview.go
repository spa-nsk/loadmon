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
	"log"
	"net/url"
	"strconv"
	"time"

	oneview "github.com/spa-nsk/oneview2"
	"google.golang.org/grpc/metadata"
)

// GetServerHardwareBySN возвращает данные сервера HPE OneView по серийному
// номеру и признак успеха. Возвращает (nil, false) при ошибке подключения
// или если сервер не найден.
func GetServerHardwareBySN(sn string) (*oneview.Server, bool) {
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	out := oneview.Server{}
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
func GetServerHardwareByName(name string) (*oneview.Server, bool) {
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	out := oneview.Server{}
	srv, err := c.GetServerHardwareByName(ctx, &pb.ServerName{Name: name})
	if err != nil {
		return nil, false
	}
	MessageToServerHardware(srv, &out)

	return &out, true
}

// GetServerHardwareByVirtualSN возвращает данные сервера HPE OneView по
// виртуальному серийному номеру и признак успеха.
func GetServerHardwareByVirtualSN(sn string) (*oneview.Server, bool) {
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	out := oneview.Server{}
	srv, err := c.GetServerHardwareByVirtualSN(ctx, &pb.SerialNumber{SerialNumber: sn})
	if err != nil {
		return nil, false
	}
	MessageToServerHardware(srv, &out)

	return &out, true
}

// OVRefreshState - состояние сбора данных HPE OneView на стороне сервера
// (передаётся в trailer-метаданных потока GetServerHardwareList, см.
// grpcserver.OVStateTrailer). Valid=false - сервер состояние не передал
// (старая версия сервера или ошибка до начала потока).
type OVRefreshState struct {
	Valid       bool
	Refreshing  bool      // сейчас идёт сбор из OneView
	LastRefresh time.Time // время последнего успешного снимка (zero - ещё не было)
	LastError   string    // ошибка последнего сбора
	ServerCount int       // число серверов в снимке
}

// Имена trailer-метаданных - дублируют grpcserver.TrailerOV* (пакет
// grpcclient не импортирует grpcserver).
const (
	trailerOVRefreshing  = "x-ov-refreshing"
	trailerOVLastRefresh = "x-ov-last-refresh"
	trailerOVLastError   = "x-ov-last-error"
	trailerOVCount       = "x-ov-server-count"
)

func ovStateFromTrailer(md metadata.MD) OVRefreshState {
	first := func(key string) (string, bool) {
		if v := md.Get(key); len(v) > 0 {
			return v[0], true
		}
		return "", false
	}
	st := OVRefreshState{}
	r, ok := first(trailerOVRefreshing)
	if !ok {
		return st
	}
	st.Valid = true
	st.Refreshing = r == "1"
	if v, ok := first(trailerOVLastRefresh); ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			st.LastRefresh = time.Unix(n, 0)
		}
	}
	if v, ok := first(trailerOVLastError); ok {
		if u, err := url.QueryUnescape(v); err == nil {
			st.LastError = u
		}
	}
	if v, ok := first(trailerOVCount); ok {
		st.ServerCount, _ = strconv.Atoi(v)
	}
	return st
}

// GetServerHardwareList возвращает список серверов HPE OneView,
// отфильтрованных по датацентру, стойке, статусу, режиму обслуживания и
// состоянию питания (пустая строка — фильтр не применяется).
func GetServerHardwareList(datacenterName, rackName, status, maintenanceMode, powerState string) ([]*oneview.Server, error) {
	out, _, err := GetServerHardwareListState(datacenterName, rackName, status, maintenanceMode, powerState)
	return out, err
}

// GetServerHardwareListState - то же, что GetServerHardwareList, плюс
// состояние сбора OneView на сервере (для индикатора "идёт обновление" на
// страницах, 2026-10-05).
func GetServerHardwareListState(datacenterName, rackName, status, maintenanceMode, powerState string) ([]*oneview.Server, OVRefreshState, error) {
	out := make([]*oneview.Server, 0)
	state := OVRefreshState{}
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return out, state, err
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
			s := oneview.Server{}
			MessageToServerHardware(srv, &s)
			out = append(out, &s)
		}
		state = ovStateFromTrailer(stream.Trailer()) // trailer доступен после завершения чтения потока
	}
	return out, state, nil
}

// messageToEnclosureSummary конвертирует ответ pb.BladeEnclosure в
// oneview2.EnclosureSummary. Данные о серверах в отсеках (ByBay) при этом
// неполные - с провода приходят только Identity.Name/Identity.SerialNumber
// (см. pb.BladeBayRef), остальные поля oneview2.Server остаются нулевыми -
// этого достаточно для задела под будущие доработки, полные данные сервера
// при необходимости нужно дозапрашивать отдельно через
// GetServerHardwareByName/GetServerHardwareBySN.
func messageToEnclosureSummary(in *pb.BladeEnclosure) *oneview.EnclosureSummary {
	if in == nil {
		return nil
	}
	out := &oneview.EnclosureSummary{
		Name:          in.Name,
		SerialNumber:  in.SerialNumber,
		Model:         in.Model,
		EnclosureType: in.EnclosureType,
		ByBay:         make(map[int]*oneview.Server, len(in.Bays)),
	}
	for _, ref := range in.Bays {
		if ref == nil {
			continue
		}
		out.ByBay[int(ref.BayNumber)] = &oneview.Server{
			Identity: oneview.ServerIdentity{
				Name:         ref.ServerName,
				SerialNumber: ref.ServerSerialNumber,
			},
		}
	}
	return out
}

// GetBladeEnclosureByName возвращает сводную информацию по блейд-корзине
// (см. oneview2.EnclosureSummary) по её нормализованному имени ("c7000-09")
// и признак успеха. Задел под будущие доработки - на момент добавления
// вызывающего кода в проекте ещё нет.
func GetBladeEnclosureByName(name string) (*oneview.EnclosureSummary, bool) {
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	enc, err := c.GetBladeEnclosureByName(ctx, &pb.ServerName{Name: name})
	if err != nil {
		return nil, false
	}
	return messageToEnclosureSummary(enc), true
}

// GetBladeEnclosureBySN возвращает сводную информацию по блейд-корзине
// (см. oneview2.EnclosureSummary) по её серийному номеру и признак успеха -
// серийный номер известен только если для хотя бы одного сервера корзины
// отработал GetEnclosure (см. комментарий у Infrastructure.EnclosureBySN).
func GetBladeEnclosureBySN(sn string) (*oneview.EnclosureSummary, bool) {
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	enc, err := c.GetBladeEnclosureBySN(ctx, &pb.SerialNumber{SerialNumber: sn})
	if err != nil {
		return nil, false
	}
	return messageToEnclosureSummary(enc), true
}

// GetHWDatacenterNames возвращает список всех различных значений
// Datacenter.Name, накопленных oneview2.Infrastructure по всем собранным
// серверам (см. oneview2/global.go) - в отличие от GetServerHardwareList,
// не зависит от текущих значений остальных полей фильтра. Используется для
// выпадающего списка "Датацентр" на /sprsrv/hwviewlist.
func GetHWDatacenterNames() ([]string, error) {
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	resp, err := c.GetHWDatacenterNames(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	return resp.Names, nil
}

// GetHWRackNames возвращает список всех различных значений Rack.Name,
// накопленных oneview2.Infrastructure по всем собранным серверам - см.
// GetHWDatacenterNames. Используется для выпадающего списка "Шкаф" на
// /sprsrv/hwviewlist.
func GetHWRackNames() ([]string, error) {
	c, conn, ctx, cancel, err := dialAPIHardware()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	resp, err := c.GetHWRackNames(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	return resp.Names, nil
}

//-------------------------------- DB interface ---------------------------------
