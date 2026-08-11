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
	"loadmon/models"
	pb "loadmon/protocol"
	"log"
)

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
