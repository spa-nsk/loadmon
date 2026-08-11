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
	"loadmon/config"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
)

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
