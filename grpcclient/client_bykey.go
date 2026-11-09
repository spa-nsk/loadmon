// Клиентские обёртки RPC сервиса Loadmon, работающих по составному ключу
// объекта ("vCenter/Датацентр/Кластер/Имя", см. VMKeyRec/HSKeyRec/CLKeyRec),
// и RPC получения списков ключей по одному критерию. Общий стиль - как в
// client_vm.go: короткоживущее соединение dialAPI(), таймаут 30 с, при ошибке
// причина пишется в лог (log.Printf) и возвращается вызывающему.
package grpcclient

import (
	"context"
	"errors"
	pb "loadmon/protocol"
	vsphere "loadmon/vspheremetrics"
	"log"
)

const (
	errEmptyKey = "не указан ключ объекта"
	errNoData   = "Данные по ключу не найдены."
)

// GetVMShortByKey возвращает укороченное описание ВМ по составному ключу key
// (gRPC GetVMShortByKey): идентификация, ресурсы и готовые агрегаты метрик, без
// временного ряда Perf, файлов, дисков и использования хранилищ. Дешевле GetVM,
// когда нужны только строки списков/таблиц. Возвращает ошибку при пустом ключе,
// ошибке подключения/RPC или если ВМ не найдена.
func GetVMShortByKey(key string) (vsphere.VMPoint, error) {
	if key == "" {
		return vsphere.VMPoint{}, errors.New(errEmptyKey)
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.VMPoint{}, err
	}
	defer conn.Close()
	defer cancel()
	vmp, err := c.GetVMShortByKey(ctx, &pb.VMReqKey{Key: key})
	if err != nil {
		log.Printf("GetVMShortByKey(%s): %v", key, err)
		return vsphere.VMPoint{}, err
	}
	return GetVMPointShortRPC(vmp), nil
}

// GetVMsSliceByKeyList возвращает полные данные ВМ (как GetVM, включая Perf и
// пересчитанные агрегаты) для всех ключей из keys одним unary-ответом
// (gRPC GetVMsSliceByKeyList). Ответ ограничен размером gRPC-сообщения - для
// больших списков используйте потоковый вариант (LoadVMByKeyRecs/
// GetVMStreamByKeyList). Пустой список ключей даёт пустой результат без
// обращения к серверу.
func GetVMsSliceByKeyList(keys *pb.VMListKey) ([]vsphere.VMPoint, error) {
	out := make([]vsphere.VMPoint, 0)
	if keys == nil || len(keys.KeyRecs) == 0 {
		return out, nil
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out, err
	}
	defer conn.Close()
	defer cancel()
	slice, err := c.GetVMsSliceByKeyList(ctx, keys)
	if err != nil {
		log.Printf("GetVMsSliceByKeyList: %v", err)
		return out, err
	}
	for _, vmp := range slice.GetVMPoints() {
		vm := GetVMPointRPC(vmp)
		CopyAndReCalcVMPerformance(&vm, vmp)
		out = append(out, vm)
	}
	return out, nil
}

// GetVMFiles возвращает файлы ВМ (конфигурация, диски, снапшоты и т.п.) по
// составному ключу key (gRPC GetVMFiles). Возвращает ошибку при пустом ключе,
// ошибке подключения/RPC или если ВМ не найдена; ВМ без файлов - пустой срез.
func GetVMFiles(key string) ([]vsphere.VMFile, error) {
	if key == "" {
		return nil, errors.New(errEmptyKey)
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetVMFiles(ctx, &pb.VMReqKey{Key: key})
	if err != nil {
		log.Printf("GetVMFiles(%s): %v", key, err)
		return nil, err
	}
	return VMFilesRPC(res.GetVMFiles()), nil
}

// GetVMGuestDiskInfo возвращает диски гостевой ОС ВМ (путь, объём, свободное
// место) по составному ключу key (gRPC GetVMGuestDiskInfo). Возвращает ошибку
// при пустом ключе, ошибке подключения/RPC или если ВМ не найдена.
func GetVMGuestDiskInfo(key string) ([]vsphere.GuestDiskInfo, error) {
	if key == "" {
		return nil, errors.New(errEmptyKey)
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetVMGuestDiskInfo(ctx, &pb.VMReqKey{Key: key})
	if err != nil {
		log.Printf("GetVMGuestDiskInfo(%s): %v", key, err)
		return nil, err
	}
	return VMDisksRPC(res.GetGuestDisks()), nil
}

// GetVMUsageOnDatastore возвращает использование хранилищ ВМ (выделено/не
// выделено/не разделено) по составному ключу key (gRPC GetVMUsageOnDatastore).
// Возвращает ошибку при пустом ключе, ошибке подключения/RPC или если ВМ не
// найдена.
func GetVMUsageOnDatastore(key string) ([]vsphere.VMUsageOnDatastore, error) {
	if key == "" {
		return nil, errors.New(errEmptyKey)
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetVMUsageOnDatastore(ctx, &pb.VMReqKey{Key: key})
	if err != nil {
		log.Printf("GetVMUsageOnDatastore(%s): %v", key, err)
		return nil, err
	}
	return VMUsageOnDatastoreRPC(res.GetUsageOnDatastores()), nil
}

// getVMListKeyBy - общая часть GetVMListKeyBy*: выполняет вызов и, как
// GetVMListKey, считает пустой результат (Count == 0) ошибкой "данные не найдены".
func getVMListKeyBy(name string, call func(c pb.LoadmonClient, ctx context.Context) (*pb.VMListKey, error)) (*pb.VMListKey, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	vml, err := call(c, ctx)
	if err != nil {
		log.Printf("%s: %v", name, err)
		return nil, err
	}
	if vml == nil || vml.Count == 0 {
		return vml, errors.New(errNoData)
	}
	return vml, nil
}

// GetVMListKeyByVMName возвращает ключи всех ВМ с именем vmname во всех
// vCenter/датацентрах/кластерах (gRPC GetVMListKeyByVMName). Пустой результат -
// ошибка "данные не найдены" (как у GetVMListKey).
func GetVMListKeyByVMName(vmname string) (*pb.VMListKey, error) {
	return getVMListKeyBy("GetVMListKeyByVMName", func(c pb.LoadmonClient, ctx context.Context) (*pb.VMListKey, error) {
		return c.GetVMListKeyByVMName(ctx, &pb.VMReqVMName{VMName: vmname})
	})
}

// GetVMListKeyByCluster возвращает ключи всех ВМ кластера с именем cluster
// (gRPC GetVMListKeyByCluster). Пустой результат - ошибка "данные не найдены".
func GetVMListKeyByCluster(cluster string) (*pb.VMListKey, error) {
	return getVMListKeyBy("GetVMListKeyByCluster", func(c pb.LoadmonClient, ctx context.Context) (*pb.VMListKey, error) {
		return c.GetVMListKeyByCluster(ctx, &pb.VMReqCluster{Cluster: cluster})
	})
}

// GetVMListKeyByVcenter возвращает ключи всех ВМ указанного vCenter (gRPC
// GetVMListKeyByVcenter). Пустой результат - ошибка "данные не найдены".
func GetVMListKeyByVcenter(vcenter string) (*pb.VMListKey, error) {
	return getVMListKeyBy("GetVMListKeyByVcenter", func(c pb.LoadmonClient, ctx context.Context) (*pb.VMListKey, error) {
		return c.GetVMListKeyByVcenter(ctx, &pb.VMReqVcenter{Vcenter: vcenter})
	})
}

// GetVMListKeyByDatacenter возвращает ключи всех ВМ указанного датацентра
// (gRPC GetVMListKeyByDatacenter). Пустой результат - ошибка "данные не найдены".
func GetVMListKeyByDatacenter(datacenter string) (*pb.VMListKey, error) {
	return getVMListKeyBy("GetVMListKeyByDatacenter", func(c pb.LoadmonClient, ctx context.Context) (*pb.VMListKey, error) {
		return c.GetVMListKeyByDatacenter(ctx, &pb.VMReqDatacenter{Datacenter: datacenter})
	})
}

// GetClusterListKey возвращает список ключей кластеров, подходящих под
// фильтры vcenter/datacenter/cluster (gRPC GetClusterListKey; значение "all"
// отключает фильтр по соответствующему полю, как и в остальных *ListKey).
// Пустой результат - ошибка "данные не найдены".
func GetClusterListKey(vcenter, datacenter, cluster string) (*pb.CLListKey, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	cll, err := c.GetClusterListKey(ctx, &pb.CLReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster})
	if err != nil {
		log.Printf("GetClusterListKey: %v", err)
		return nil, err
	}
	if cll == nil || cll.Count == 0 {
		return cll, errors.New(errNoData)
	}
	return cll, nil
}

// GetClusterByKey возвращает кластер с агрегированными метриками (без списка
// хостов, ВМ и хранилищ) по составному ключу key (gRPC GetClusterByKey).
// Возвращает ошибку при пустом ключе, ошибке подключения/RPC или если кластер
// не найден.
func GetClusterByKey(key string) (vsphere.Cluster, error) {
	if key == "" {
		return vsphere.Cluster{}, errors.New(errEmptyKey)
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.Cluster{}, err
	}
	defer conn.Close()
	defer cancel()
	pbcl, err := c.GetClusterByKey(ctx, &pb.CLReqKey{Key: key})
	if err != nil {
		log.Printf("GetClusterByKey(%s): %v", key, err)
		return vsphere.Cluster{}, err
	}
	cl := vsphere.Cluster{}
	MessageToClusterMedium(pbcl, &cl)
	return cl, nil
}

// GetClusterPerfByKey возвращает временной ряд метрик кластера по составному
// ключу key единым срезом (gRPC GetClusterPerfByKey). Возвращает ошибку при
// пустом ключе или ошибке подключения/RPC; кластер без метрик - пустой срез.
func GetClusterPerfByKey(key string) ([]*vsphere.CLPerf, error) {
	if key == "" {
		return nil, errors.New(errEmptyKey)
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetClusterPerfByKey(ctx, &pb.CLReqKey{Key: key})
	if err != nil {
		log.Printf("GetClusterPerfByKey(%s): %v", key, err)
		return nil, err
	}
	return CLPerfRPC(res.GetPerf()), nil
}

// GetHostPerfByKey возвращает временной ряд метрик хоста ESXi по составному
// ключу key единым срезом (gRPC GetHostPerfByKey). Возвращает ошибку при пустом
// ключе, ошибке подключения/RPC или если хост не найден.
func GetHostPerfByKey(key string) ([]*vsphere.HSPerf, error) {
	if key == "" {
		return nil, errors.New(errEmptyKey)
	}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetHostPerfByKey(ctx, &pb.HSReqKey{Key: key})
	if err != nil {
		log.Printf("GetHostPerfByKey(%s): %v", key, err)
		return nil, err
	}
	return HSPerfRPC(res.GetPerf()), nil
}
