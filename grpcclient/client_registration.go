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
	"errors"
	"fmt"
	"io"
	"loadmon/config"
	pb "loadmon/protocol"
	"log"
)

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
