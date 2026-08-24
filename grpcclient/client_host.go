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

func LoadHostListFull(vcenter, datacenter, cluster, hostname string) []*vsphere.HostInfo {
	out := make([]*vsphere.HostInfo, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetHostFull(ctx, &pb.HSReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, HSName: hostname})
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			hs := vsphere.HostInfo{}
			MessageToHostInfoFull(pbcl, &hs)
			out = append(out, &hs)
		}
	}
	return out
}

// LoadHostListKey запрашивает список ключей (KeyName) хостов, подходящих
// под заданные фильтры. При ошибке подключения или RPC возвращает пустой
// (не nil) срез.
func LoadHostListKey(vcenter, datacenter, cluster, hostname string) []string {
	out := make([]string, 0)
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return out
	}
	defer conn.Close()
	defer cancel()
	listKey, err := c.GetHostListKey(ctx, &pb.HSReqListKey{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, HSName: hostname})
	if err == nil {
		for i := int64(0); i < listKey.Count; i++ {
			out = append(out, listKey.KeyRecs[i].Key)
		}
	}
	return out
}

// GetHostByKey возвращает информацию о хосте по его уникальному ключу.
// Возвращает nil при ошибке подключения или если хост не найден.
func GetHostByKey(key string) *vsphere.HostInfo {
	out := vsphere.HostInfo{}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	hs, err := c.GetHostByKey(ctx, &pb.HSReqKey{Key: key})
	if err == nil {
		MessageToHostInfo(hs, &out)
		return &out
	}
	return nil
}

// GetHostByKeyDirect - аналог GetHostByKey выше, но данные запрашиваются
// НАПРЯМУЮ в VMware через отдельное govmomi-соединение (см.
// vspheremetrics.GetHostDirect, вызывается на сервере обработчиком
// GetHostByKeyDirect, grpcserver/handler_host.go), минуя закэшированный в
// памяти пул данных (VMInfrastructure.HostMapList), который читает
// GetHostByKey. Использовать там, где важно самое текущее состояние хоста
// в vSphere прямо сейчас - см. комментарий к grpcclient.GetVMByKeyDirect
// (тот же принцип, для хостов).
func GetHostByKeyDirect(vcenter, datacenter, cluster, hostname string) (vsphere.HostInfo, error) {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return vsphere.HostInfo{}, err
	}
	defer conn.Close()
	defer cancel()
	hs, err := c.GetHostByKeyDirect(ctx, &pb.HSReq{Vcenter: vcenter, Datacenter: datacenter, Cluster: cluster, HSName: hostname})
	if err != nil {
		return vsphere.HostInfo{}, err
	}
	out := vsphere.HostInfo{}
	MessageToHostInfo(hs, &out)
	return out, nil
}

// GetHostBySN возвращает информацию о хосте по серийному номеру. Возвращает
// nil при ошибке подключения или если хост не найден.
func GetHostBySN(sn string) *vsphere.HostInfo {
	out := vsphere.HostInfo{}
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	hs, err := c.GetHostBySN(ctx, &pb.SerialNumber{SerialNumber: sn})
	if err == nil {
		MessageToHostInfo(hs, &out)
		return &out
	}
	return nil
}

// LoadHostMetrics возвращает временной ряд метрик производительности хоста
// по его ключу. При ошибке подключения возвращает nil; повреждённые записи
// потока пропускаются.
func LoadHostMetrics(key string) []*vsphere.HSPerf {
	c, conn, ctx, cancel, err := dialAPI()
	if err != nil {
		return nil
	}
	defer conn.Close()
	defer cancel()
	stream, err := c.GetHostPerf(ctx, &pb.HSReqKey{Key: key})
	perf := make([]*vsphere.HSPerf, 0)
	if err == nil {
		for {
			pbcl, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			clp := vsphere.HSPerf{}
			MessageToHostMetric(pbcl, &clp)
			perf = append(perf, &clp)
		}
	}
	return perf
}

// LinkVMToCluster связывает список VM со списком кластеров по составному
// ключу vcenter/datacenter/cluster, заполняя cl[i].VM и cl[i].VMCount.
// Чисто локальная операция, gRPC не использует.
//
// cl[i].VMCount на входе уже может быть заполнен сервером (см.
// grpcserver.ClusterToMessageFull/MessageToClusterFull - VMCount берётся из
// инфраструктуры независимо от среза VM). Здесь счётчик пересчитывается с
// нуля по фактически связанным записям VM, а не увеличивается поверх уже
// установленного значения - иначе он задваивался бы.
func LinkVMToCluster(vm []*vsphere.VMPoint, cl []*vsphere.Cluster) {
	mapcl := make(map[string]*vsphere.Cluster)
	for _, clp := range cl {
		clp.VM = clp.VM[:0]
		clp.VMCount = 0
		mapcl[clp.Vcenter+"/"+clp.Datacenter+"/"+clp.Name] = clp
	}
	for _, vmp := range vm {
		clp, ok := mapcl[vmp.Vcenter+"/"+vmp.Datacenter+"/"+vmp.Cluster]
		if ok {
			clp.VM = append(clp.VM, vmp)
			clp.VMCount++
		}
	}
}

// LinkHostToCluster связывает список хостов со списком кластеров по
// составному ключу vcenter/datacenter/cluster, заполняя cl[i].Hosts и
// cl[i].HostCount. Чисто локальная операция, gRPC не использует.
//
// Аналогично LinkVMToCluster: cl[i].HostCount на входе уже может быть
// заполнен сервером, поэтому счётчик и срез Hosts здесь пересчитываются с
// нуля, а не дополняются поверх существующего значения.
func LinkHostToCluster(hs []*vsphere.HostInfo, cl []*vsphere.Cluster) {
	mapcl := make(map[string]*vsphere.Cluster)
	for _, clp := range cl {
		clp.Hosts = clp.Hosts[:0]
		clp.HostCount = 0
		mapcl[clp.Vcenter+"/"+clp.Datacenter+"/"+clp.Name] = clp
	}
	for _, hsp := range hs {
		clp, ok := mapcl[hsp.Vcenter+"/"+hsp.Datacenter+"/"+hsp.Cluster]
		if ok {
			clp.Hosts = append(clp.Hosts, hsp)
			clp.HostCount++
		}
	}
}
