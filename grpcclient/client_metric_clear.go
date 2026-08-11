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
	pb "loadmon/protocol"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

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
