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
	"io"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
)

func LoadVMListKeys(vcenter, datacenter, cluster, vmname string) []*pb.VMKeyRec {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})
	if err != nil || vml == nil {
		return nil
	}
	return vml.KeyRecs
}

// LoadVMByKeyRecs догружает полные данные (VMPoint) только для переданного
// подмножества ключей keyRecs (например, только для текущей страницы
// таблицы). Используется вместе с LoadVMListKeys, чтобы не запрашивать
// весь отфильтрованный список целиком, если реально отображается лишь
// PageSize записей.
func LoadVMByKeyRecs(keyRecs []*pb.VMKeyRec) []*vsphere.VMPoint {
	vmlist := make([]*vsphere.VMPoint, 0, len(keyRecs))
	if len(keyRecs) == 0 {
		return vmlist
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vmlist
	}
	defer conn.Close()
	defer cancel()

	stream, err2 := c.GetVMStreamByKeyList(ctx, &pb.VMListKey{KeyRecs: keyRecs, Count: int64(len(keyRecs))})
	if err2 == nil {
		for {
			vmp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetVMStreamByKeyList: ошибка чтения потока: %v", err)
				break
			}
			vm2 := GetVMPointRPC(vmp)
			vmlist = append(vmlist, &vm2)
		}
	}
	return vmlist
}

// LoadVMList запрашивает у сервера полный список виртуальных машин по
// фильтрам vcenter/datacenter/cluster/vmname и возвращает их доменные
// представления. При ошибке подключения или вызова возвращает пустой срез.
func LoadVMList(vcenter, datacenter, cluster, vmname string) []*vsphere.VMPoint {
	vmlist := make([]*vsphere.VMPoint, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vmlist
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})

	stream, err2 := c.GetVMStreamByKeyList(ctx, vml)

	if err2 == nil {
		for {
			vmp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetVMStreamByKeyList: ошибка чтения потока: %v", err)
				break
			}
			vm2 := GetVMPointRPC(vmp)
			vmlist = append(vmlist, &vm2)
		}
	}
	return vmlist
}

// LoadVMListFull работает как LoadVMList, но дополнительно вычисляет
// показатели загрузки (CPUUsageMaximum/Average/Minimum, MemUsage*,
// DiskUsage*, NetUsage* и т.п.) по точкам метрик, присланным сервером в
// каждом vmp.Perf - через CopyAndReCalcVMPerformance, как это уже делает
// GetVMByKeyWithPerf для одиночной ВМ. LoadVMList этого не делает
// намеренно (см. комментарий там) - используется там, где отображаются
// только идентификационные поля ВМ (например, /sprsrv/vm), без загрузки
// лишних данных метрик. LoadVMListFull нужен там, где показатели загрузки
// действительно отображаются (например, "Виртуальные машины кластера" на
// /sprsrv/cluster/... - раньше эти столбцы всегда показывали 0.00, так как
// использовался LoadVMList без пересчёта).
func LoadVMListFull(vcenter, datacenter, cluster, vmname string) []*vsphere.VMPoint {
	vmlist := make([]*vsphere.VMPoint, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vmlist
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})

	stream, err2 := c.GetVMStreamByKeyList(ctx, vml)

	if err2 == nil {
		for {
			vmp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetVMStreamByKeyList: ошибка чтения потока: %v", err)
				break
			}
			vm2 := GetVMPointRPC(vmp)
			CopyAndReCalcVMPerformance(&vm2, vmp)
			vmlist = append(vmlist, &vm2)
		}
	}
	return vmlist
}

// CopyAndReCalcVMPerformance копирует точки метрик производительности VM
// (in.Perf) в доменную модель out, попутно вычисляя минимум/максимум/среднее
// по CPU, памяти, диску и сети. При out == nil или in == nil ничего не
// делает. Перед копированием очищает ранее накопленные метрики out
// (out.VMClearPerfomance()).
func GetVMByKey(vcenter, datacenter, cluster, vmname string) (vsphere.VMPoint, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})

	stream, err2 := c.GetVMStreamByKeyList(ctx, vml)

	vmlist := make([]vsphere.VMPoint, 0)

	if err2 == nil {
		for {
			vmp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("GetVMStreamByKeyList: ошибка чтения потока: %v", err)
				break
			}
			vmlist = append(vmlist, GetVMPointRPC(vmp))
		}
	}
	if len(vmlist) > 0 {
		return vmlist[0], nil
	}
	return vsphere.VMPoint{}, errors.New("Данные по ключу не найдены.")
}

// GetVMByKeyWithPerf аналогичен GetVMByKey (возвращает первую VM, подходящую
// под vcenter/datacenter/cluster/vmname), но дополнительно заполняет
// vm.Perf/vm.PerfCount - временной ряд метрик производительности. GetVMByKey
// этого не делает: он строит доменный объект через GetVMPointRPC, которая
// сознательно не трогает Perf (иначе он заполнялся бы дважды при массовой
// загрузке списка ВМ, где Perf обычно не нужен - см. LoadVMList/LoadVMByKeyRecs).
// Использовать эту функцию там, где нужен именно временной ряд метрик одной
// конкретной ВМ (например, графики загрузки).
func GetVMByKeyWithPerf(vcenter, datacenter, cluster, vmname string) (vsphere.VMPoint, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})
	if err != nil {
		return vsphere.VMPoint{}, err
	}

	stream, err2 := c.GetVMStreamByKeyList(ctx, vml)
	if err2 != nil {
		return vsphere.VMPoint{}, err2
	}

	for {
		vmp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("GetVMStreamByKeyList: ошибка чтения потока: %v", err)
			break
		}
		vm := GetVMPointRPC(vmp)
		CopyAndReCalcVMPerformance(&vm, vmp)
		return vm, nil
	}
	return vsphere.VMPoint{}, errors.New("Данные по ключу не найдены.")
}

// GetVM возвращает VM по её уникальному ключу key. Возвращает ошибку, если
// подключиться к серверу не удалось или данные по ключу не найдены.
func GetVM(key string) (vsphere.VMPoint, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vm, err := c.GetVMByKey(ctx, &pb.VMReqKey{Key: key})
	if err == nil {
		return GetVMPointRPC(vm), nil
	}
	return vsphere.VMPoint{}, errors.New("Данные по ключу не найдены.")
}

// GetVMListKey возвращает список ключей VM, подходящих под заданные
// фильтры. Возвращает ошибку, если подключение не удалось или ни одной
// записи не найдено (vml.Count == 0).
func GetVMListKey(vcenter, datacenter, cluster, vmname string) (*pb.VMListKey, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	vml, err := c.GetVMListKey(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})
	if vml != nil && err == nil {
		if vml.Count > 0 {
			return vml, nil
		}
	}
	return vml, errors.New("Данные по ключу не найдены.")
}

// LoadVMListToClusterStorage дозагружает полные данные VM (по ключам,
// уже сохранённым в in.Storage[*].VM) для каждого кластера из ins и
// подставляет их обратно в in.Storage[i].VM. Ошибки по отдельным VM
// (например, VM удалена) пропускаются без прерывания всей операции.
// При ошибке подключения к серверу функция тихо завершается, не изменяя ins.
