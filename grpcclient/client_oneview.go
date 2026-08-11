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
	"loadmon/oneview"
	pb "loadmon/protocol"
	"log"
)

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
