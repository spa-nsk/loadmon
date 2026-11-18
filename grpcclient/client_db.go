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

// AddClusterRecord добавляет новую ежедневную историческую запись кластера
// (cluster_record) через сервис LoadmonDB. Возвращает (id, статус): статус 1 -
// успех, 2 - ошибка вставки на сервере; при ошибке подключения возвращает
// (0, 1) - как и остальные Add*/Update* функции этого файла, - при ошибке RPC
// (0, 3).
func AddClusterRecord(in *models.ClusterRecord) (int64, int) {
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return 0, 1
	}
	defer conn.Close()
	defer cancel()
	req := pb.ClusterRecord{}
	ClusterRecordToMessage(in, &req)
	res, err := c.AddClusterRecord(ctx, &req)
	if err == nil {
		return res.Id, int(res.Status)
	}
	log.Printf("AddClusterRecord: %v", err)
	return 0, 3 //ошибка операции
}

// GetServerSpr возвращает активную запись справочника серверов
// (server_spr_records, Stat = 1) через сервис LoadmonDB по FQDN сервера fqdn
// (сравнение с NameFQDN без учёта регистра) - подробности в
// grpcserver.(*serverdb).GetServer. Возвращает ошибку при пустом FQDN, ошибке
// подключения/RPC или если запись не найдена.
func GetServerSpr(fqdn string) (*models.ServerSprRecords, error) {
	if fqdn == "" {
		return nil, errors.New("не указан FQDN сервера")
	}
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetServer(ctx, &pb.DBKeyString{Key: fqdn})
	if err != nil {
		log.Printf("GetServer(%s): %v", fqdn, err)
		return nil, err
	}
	out := models.ServerSprRecords{}
	MessageToServerSprRecod(res, &out)
	return &out, nil
}

// GetDBVMPerf возвращает детальные метрики загрузки виртуальной машины
// (vm_perf) за последние 3 суток через сервис LoadmonDB по ключу ВМ key (формат
// Vcenter/Datacenter/Cluster/Vm_Name), в порядке возрастания Start - подробности
// в grpcserver.(*serverdb).GetDBVMPerf. Отсутствие метрик не ошибка (пустой
// срез). Возвращает ошибку при пустом ключе, ошибке подключения/RPC или если ВМ
// не найдена в справочнике.
func GetDBVMPerf(key string) ([]*models.VMPerfRecord, error) {
	if key == "" {
		return nil, errors.New("не указан ключ виртуальной машины")
	}
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetDBVMPerf(ctx, &pb.VMReqKey{Key: key})
	if err != nil {
		log.Printf("GetDBVMPerf(%s): %v", key, err)
		return nil, err
	}
	return messageToVMPerfRecords(res), nil
}

// messageToVMPerfRecords преобразует ответ GetDBVMPerf в срез моделей
// (nil-элементы пропускаются, результат никогда не nil).
func messageToVMPerfRecords(in *pb.VMPerfRecordSlice) []*models.VMPerfRecord {
	out := make([]*models.VMPerfRecord, 0, len(in.GetPerf()))
	for _, m := range in.GetPerf() {
		if m == nil {
			continue
		}
		rec := &models.VMPerfRecord{}
		MessageToVMPerfRecord(m, rec)
		out = append(out, rec)
	}
	return out
}

// UpdateVMSprDatacenterName обновляет поле VMSprRecord.DatacenterName (код ЦОД
// DatacenterSite.CodeJira, определённый через физический ESXi хост ВМ) через
// сервис LoadmonDB, по Id справочной записи ВМ (vm_spr_record.Id). Вызывается
// из cmdb.UpdateVMDatacenterNames (cmdb/vmdatacentername.go) - отдельной
// функции, запускаемой параллельно с сохранением метрик ВМ, а не встроенной
// в cmdb.SaveVMRecord. Возвращает статус; при ошибке подключения возвращает
// 1, при ошибке RPC - 3.
// ПЕРЕИМЕНОВАНО (2026-09-30) из UpdateVMSprDatastoreName - опечатка.
func UpdateVMSprDatacenterName(id int64, datacenterName string) int {
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return 1
	}
	defer conn.Close()
	defer cancel()
	req := pb.VMSprDatacenterNameRequest{Id: id, DatacenterName: datacenterName}
	res, err := c.UpdateVMSprDatacenterName(ctx, &req)
	if err == nil {
		return int(res.Status)
	}
	return 3 //ошибка операции
}

// UpdateClusterSprDatacenterName обновляет поле ClusterSprRecord.DatacenterName
// (объединённый через "||" без дублей список кодов ЦОД DatacenterSite.CodeJira
// всех ESXi гипервизоров кластера) через сервис LoadmonDB, по Id справочной
// записи кластера (cluster_spr_records.Id). Вызывается из
// cmdb.UpdateClusterDatacenterNames (cmdb/clusterdatacentername.go) - отдельной
// функции, запускаемой параллельно с сохранением метрик кластера, по
// аналогии с UpdateVMSprDatacenterName. Возвращает статус; при ошибке
// подключения возвращает 1, при ошибке RPC - 3.
// ПЕРЕИМЕНОВАНО (2026-09-30) из UpdateClusterSprDatastoreName - опечатка.
func UpdateClusterSprDatacenterName(id int64, datacenterName string) int {
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return 1
	}
	defer conn.Close()
	defer cancel()
	req := pb.ClusterSprDatacenterNameRequest{Id: id, DatacenterName: datacenterName}
	res, err := c.UpdateClusterSprDatacenterName(ctx, &req)
	if err == nil {
		return int(res.Status)
	}
	return 3 //ошибка операции
}
