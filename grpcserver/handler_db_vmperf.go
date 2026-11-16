// Файл содержит обработчик GetDBVMPerf сервиса LoadmonDB - чтение детальных
// метрик загрузки виртуальной машины (таблица vm_perf) за последние 3 суток.
package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"loadmon/config"
	"loadmon/models"
	pb "loadmon/protocol"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// dbVMPerfPeriod - глубина выборки GetDBVMPerf. Совпадает со сроком хранения
// детальных метрик vm_perf (cmdb.vmPerfRetention, см. ClearOldMetric), поэтому
// более длинный период всё равно не вернул бы данных.
const dbVMPerfPeriod = 3 * 24 * time.Hour

// dbVMPerfInChunk - максимальное число идентификаторов в одном условии IN
// при чтении vm_perf (ограничение на число параметров запроса).
const dbVMPerfInChunk = 500

const (
	errDBVMPerfEmptyKey = "не указан ключ виртуальной машины"
	errDBVMPerfBadKey   = "некорректный ключ виртуальной машины, ожидается Vcenter/Datacenter/Cluster/Vm_Name"
	errDBVMPerfNotFound = "виртуальная машина не найдена в справочнике (vm_spr_record)"
)

// splitVMKey разбирает ключ ВМ формата Vcenter/Datacenter/Cluster/Vm_Name
// (VMSprRecord.KeyName). Имя ВМ - всё, что после третьего разделителя, поэтому
// допускает символ "/".
func splitVMKey(key string) (vcenter, datacenter, cluster, name string, err error) {
	parts := strings.SplitN(strings.TrimSpace(key), "/", 4)
	if len(parts) != 4 {
		return "", "", "", "", errors.New(errDBVMPerfBadKey)
	}
	for _, p := range parts {
		if p == "" {
			return "", "", "", "", errors.New(errDBVMPerfBadKey)
		}
	}
	return parts[0], parts[1], parts[2], parts[3], nil
}

// findVMSprIds возвращает Id всех записей справочника ВМ (vm_spr_record) с
// ключом key (формат Vcenter/Datacenter/Cluster/Vm_Name). Записей может быть
// несколько (ВМ удалена и создана заново). Если ключ некорректен или ВМ в
// справочнике нет - ошибка.
func findVMSprIds(o orm.Ormer, key string) ([]int64, error) {
	vcenter, datacenter, cluster, name, err := splitVMKey(key)
	if err != nil {
		return nil, err
	}
	var sprs []*models.VMSprRecord
	if _, err := o.QueryTable(new(models.VMSprRecord)).
		Filter("Vcenter", vcenter).
		Filter("Datacenter", datacenter).
		Filter("Cluster", cluster).
		Filter("Vm_Name", name).
		All(&sprs, "Id"); err != nil {
		return nil, fmt.Errorf("чтение vm_spr_record: %w", err)
	}
	if len(sprs) == 0 {
		return nil, errors.New(errDBVMPerfNotFound)
	}
	ids := make([]int64, 0, len(sprs))
	for _, s := range sprs {
		ids = append(ids, s.Id)
	}
	return ids, nil
}

// loadDBVMPerf читает из БД детальные метрики ВМ с ключом key (формат
// Vcenter/Datacenter/Cluster/Vm_Name), снятые не раньше since, в порядке
// возрастания Start. Цепочка связей: vm_spr_record (справочник ВМ) ->
// vm_record (запись ВМ по циклу сбора, VMSprId) -> vm_perf (VMId = vm_record.Id).
// Если ВМ в справочнике нет - ошибка; если метрик нет - пустой срез без ошибки.
func loadDBVMPerf(o orm.Ormer, key string, since time.Time) ([]*models.VMPerfRecord, error) {
	sprIds, err := findVMSprIds(o, key)
	if err != nil {
		return nil, err
	}

	// Метка Start точки замера не позже Date записи vm_record, в которую она
	// сохранена, поэтому записи старше since метрик за период не содержат.
	var recs []*models.VMRecord
	if _, err := o.QueryTable(new(models.VMRecord)).
		Filter("VMSprId__in", sprIds).
		Filter("Date__gte", since).
		All(&recs, "Id"); err != nil {
		return nil, fmt.Errorf("чтение vm_record: %w", err)
	}
	recIds := make([]int64, 0, len(recs))
	for _, r := range recs {
		recIds = append(recIds, r.Id)
	}

	res := make([]*models.VMPerfRecord, 0)
	for from := 0; from < len(recIds); from += dbVMPerfInChunk {
		to := from + dbVMPerfInChunk
		if to > len(recIds) {
			to = len(recIds)
		}
		var chunk []*models.VMPerfRecord
		if _, err := o.QueryTable(new(models.VMPerfRecord)).
			Filter("VMId__in", recIds[from:to]).
			Filter("Start__gte", since).
			All(&chunk); err != nil {
			return nil, fmt.Errorf("чтение vm_perf: %w", err)
		}
		res = append(res, chunk...)
	}
	sortVMPerfByStart(res)
	return res, nil
}

// sortVMPerfByStart сортирует метрики по Start по возрастанию (при равенстве -
// по Id), чтобы результат не зависел от разбиения выборки на части.
func sortVMPerfByStart(recs []*models.VMPerfRecord) {
	sort.SliceStable(recs, func(i, j int) bool { return vmPerfLess(recs[i], recs[j]) })
}

func vmPerfLess(a, b *models.VMPerfRecord) bool {
	if a.Start.Equal(b.Start) {
		return a.Id < b.Id
	}
	return a.Start.Before(b.Start)
}

// GetDBVMPerf возвращает детальные метрики загрузки ВМ (vm_perf) за последние
// 3 суток по ключу VMReqKey.Key (формат Vcenter/Datacenter/Cluster/Vm_Name).
// Если сконфигурированы удалённые обработчики (config.Switch.Endpoint), запрос
// ретранслируется на первый из них, вернувший успех (сервис LoadmonDB, адрес
// GetSrvDBAddr), иначе данные читаются из локальной БД.
func (p *serverdb) GetDBVMPerf(_ context.Context, in *pb.VMReqKey) (*pb.VMPerfRecordSlice, error) {
	empty := func() *pb.VMPerfRecordSlice { return &pb.VMPerfRecordSlice{Perf: make([]*pb.VMPerfRecord, 0)} }
	if in.GetKey() == "" {
		return empty(), errors.New(errDBVMPerfEmptyKey)
	}
	if config.Switch.Endpoint.Count() > 0 {
		for _, key := range config.Switch.Endpoint.Keys() {
			endpoint, ok := config.Switch.Endpoint.Get(key)
			if !ok {
				continue
			}
			conn, err := grpc.NewClient(endpoint.GetSrvDBAddr(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v\n", err)
				return empty(), err
			}
			defer conn.Close()

			c := pb.NewLoadmonDBClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			perf, err := c.GetDBVMPerf(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return perf, nil
			}
		}
		return empty(), errors.New(errNotFound) //ничего не найдено
	}

	recs, err := loadDBVMPerf(orm.NewOrm(), in.GetKey(), time.Now().Add(-dbVMPerfPeriod))
	if err != nil {
		log.Printf("GetDBVMPerf(%s): %v\n", in.GetKey(), err)
		return empty(), err
	}
	out := &pb.VMPerfRecordSlice{Count: int64(len(recs)), Perf: make([]*pb.VMPerfRecord, 0, len(recs))}
	for _, r := range recs {
		m := &pb.VMPerfRecord{}
		VMPerfRecordToMessage(r, m)
		out.Perf = append(out.Perf, m)
	}
	return out, nil
}
