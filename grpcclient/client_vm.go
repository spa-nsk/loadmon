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
//
// В отличие от LoadVMList/GetVMByKey, эта функция используется там, где
// показатели загрузки (CPUUsageMaximum/Average, MemUsage* и т.п.) реально
// отображаются в таблице (controllers/sprsrvvm.go -> views/sprsrv/vmlist.tpl,
// столбцы "Процессоры/Ядра"/"Память, ГБ" с "(NN%)") - поэтому после
// GetVMPointRPC (которая сознательно не трогает Perf/агрегаты, см. её
// комментарий) дополнительно вызывается CopyAndReCalcVMPerformance, как уже
// сделано в LoadVMListFull для аналогичного случая на /sprsrv/cluster. Без
// этого вызова колонки CPU/Память всегда показывали 0% (баг, воспроизводимый
// стабильно на /sprsrv/vm/all/all/all для любой ВМ, независимо от реальной
// загрузки — GetVMPointRPC оставляет CPUUsageMaximum/MemUsageMaximum в
// нулевом значении Go по умолчанию, хотя сервер уже присылает и готовые
// агрегаты, и сырой временной ряд vmp.Perf в том же ответе).
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
			CopyAndReCalcVMPerformance(&vm2, vmp)
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
// GetVMByKeyWithPerf для одиночной ВМ (и, начиная с 2026-08-26,
// LoadVMByKeyRecs - см. её комментарий, там был баг с постоянными 0% на
// /sprsrv/vm по той же причине). LoadVMList этого по-прежнему не делает
// намеренно - используется только устаревшим неподключённым к маршрутам
// controllers/sprsrvvmold.go (SprSrvVMOld, см. kontext/05-conventions-and-
// gotchas.md про мёртвый код). LoadVMListFull нужен там, где показатели
// загрузки действительно отображаются (например, "Виртуальные машины
// кластера" на
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

// GetVMByKeyDirect - аналог GetVMByKey выше, но данные запрашиваются
// НАПРЯМУЮ в VMware через отдельное govmomi-соединение (см.
// vspheremetrics.GetVMDirect, вызывается на сервере обработчиком
// GetVMByKeyDirect, grpcserver/handler_vm.go), минуя закэшированный в
// памяти пул данных (VMInfrastructure.VMMapList и связанные карты
// глобальной инфраструктуры сборщика), который читает GetVMByKey.
//
// Использовать там, где важно не устаревшее (до нескольких минут, до
// следующего опроса кластера сборщиком), а самое текущее состояние ВМ в
// vSphere прямо сейчас - например, перед тем как пометить запись
// справочника выведенной из эксплуатации из-за предполагаемого отсутствия
// ВМ (см. cmdb.CheckServerSprRecordsExistence). Для обычного чтения
// данных ВМ (списки, карточки, графики и т.п.) следует по-прежнему
// использовать GetVMByKey/GetVM/LoadVMList* - прямой запрос к vCenter
// заметно медленнее и не должен вызываться массово.
func GetVMByKeyDirect(vcenter, datacenter, cluster, vmname string) (vsphere.VMPoint, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vm, err := c.GetVMByKeyDirect(ctx, &pb.VMReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, VMName: vmname})
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	return GetVMPointRPC(vm), nil
}

// GetVMByKeyWithPerf аналогичен GetVMByKey (возвращает первую VM, подходящую
// под vcenter/datacenter/cluster/vmname), но дополнительно заполняет
// vm.Perf/vm.PerfCount - временной ряд метрик производительности. GetVMByKey
// этого не делает: он строит доменный объект через GetVMPointRPC, которая
// сознательно не трогает Perf/агрегаты сама по себе (иначе они заполнялись
// бы дважды в функциях, которые сами вызывают CopyAndReCalcVMPerformance
// после неё - см. LoadVMListFull, LoadVMByKeyRecs). GetVMByKey и голый
// LoadVMList (в отличие от LoadVMByKeyRecs) по-прежнему не пересчитывают
// агрегаты - используются там, где реально нужны только идентификационные
// поля ВМ, без показателей загрузки.
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
//
// Дополнительно вызывает CopyAndReCalcVMPerformance поверх GetVMPointRPC -
// как и в LoadVMByKeyRecs/GetVMByKeyWithPerf, сам по себе GetVMPointRPC не
// пересчитывает агрегаты CPUUsageMaximum/MemUsageMaximum и т.п. из vm.Perf
// (см. их комментарии). Без этого вызова агрегаты оставались нулевыми, и
// cmdb.SaveVMRecord (единственный вызывающий эту функцию) пропускал вставку
// суточной записи в vm_record из-за проверки "Cpu_usage_maximum == 0 &&
// Mem_usage_maximum == 0 => пропустить" - в результате на
// /sprsrv/clusterperioddb для всех ВМ кластера показывалось "нет
// сохранённых данных в БД (vm_record)" за весь период, хотя реальные данные
// по загрузке были. Исправлено 2026-08-27 (тот же класс бага, что и в
// LoadVMByKeyRecs, но в другой функции).
func GetVM(key string) (vsphere.VMPoint, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vmp, err := c.GetVMByKey(ctx, &pb.VMReqKey{Key: key})
	if err == nil {
		vm := GetVMPointRPC(vmp)
		CopyAndReCalcVMPerformance(&vm, vmp)
		return vm, nil
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
