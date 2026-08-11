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
	vsphere "loadmon/vspheremetrics"
)

func LoadVMListToClusterStorage(ins []*vsphere.Cluster) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return
	}
	defer conn.Close()
	defer cancel()

	for _, in := range ins {
		vms := make([]*vsphere.VMPoint, 0)
		for i, ds := range in.Storage {
			for _, vm := range ds.VM {
				vmp, err := c.GetVMByKey(ctx, &pb.VMReqKey{Key: vm.KeyName})
				if err != nil {
					continue
				}
				vmp2 := GetVMPointRPC(vmp)
				vms = append(vms, &vmp2)
			}
			in.Storage[i].VM = vms
		}
	}
}

// GetDSData возвращает данные хранилища (datastore) по его ключу и
// признак успеха. Возвращает (nil, false) при ошибке подключения/RPC.
func GetDSData(key string) (*vsphere.Datastore, bool) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, false
	}
	defer conn.Close()
	defer cancel()
	s, err := c.GetDSData(ctx, &pb.KeyString{Key: key})
	if err == nil {
		ds := vsphere.Datastore{ID: s.Id,
			UID:         s.UID,
			StorageName: s.StorageName,
			CapacityGb:  s.CapacityGb,
			FreeSpaceGb: s.FreeSpaceGb,
			PercentUsed: s.PercentUsed}
		for _, vmk := range s.VmKeys.KeyRecs {
			vm := vsphere.VMPoint{Name: vmk.VMName, KeyName: vmk.Key}
			ds.VMCount++
			ds.VM = append(ds.VM, &vm)
		}
		return &ds, true
	}
	return nil, false
}

// GetAllDSData возвращает список всех хранилищ, связанных с ключом key
// (например, все datastore кластера). При ошибке подключения возвращает
// пустой (не nil) срез; повреждённые записи потока пропускаются.
func GetAllDSData(key string) []*vsphere.Datastore {
	allds := make([]*vsphere.Datastore, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return allds
	}
	defer conn.Close()
	defer cancel()
	s, err := c.GetAllDSData(ctx, &pb.KeyString{Key: key})

	if err == nil {
		for {
			pbds, err := s.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			ds := vsphere.Datastore{ID: pbds.Id,
				UID:         pbds.UID,
				StorageName: pbds.StorageName,
				CapacityGb:  pbds.CapacityGb,
				FreeSpaceGb: pbds.FreeSpaceGb,
				PercentUsed: pbds.PercentUsed}

			allds = append(allds, &ds)
		}
	}
	return allds
}

// Очистить метрики в указанном диапазоне для одиночного cluster или всех cluster="all" или cluster=""
