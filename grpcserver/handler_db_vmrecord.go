// Файл содержит обработчик GetDBVMRecord сервиса LoadmonDB - чтение записей
// виртуальной машины (таблица vm_record) за период.
package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"loadmon/config"
	"loadmon/models"
	pb "loadmon/protocol"
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const errDBVMRecordBadPeriod = "окончание периода раньше его начала"

// vmRecordPeriod переводит границы периода запроса в time.Time. Не заданная
// (nil) граница возвращается как нулевое время - "без ограничения". Если обе
// границы заданы и end < start - ошибка.
func vmRecordPeriod(in *pb.VMRecordPeriodReq) (start, end time.Time, err error) {
	if in.GetStart() != nil {
		start = in.GetStart().AsTime()
	}
	if in.GetEnd() != nil {
		end = in.GetEnd().AsTime()
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return time.Time{}, time.Time{}, errors.New(errDBVMRecordBadPeriod)
	}
	return start, end, nil
}

// loadDBVMRecords читает из БД записи vm_record ВМ с ключом key (формат
// Vcenter/Datacenter/Cluster/Vm_Name) с Date в периоде [start, end]
// (нулевая граница не ограничивает выборку), в порядке возрастания Date.
// Если ВМ в справочнике нет - ошибка; если записей нет - пустой срез.
func loadDBVMRecords(o orm.Ormer, key string, start, end time.Time) ([]*models.VMRecord, error) {
	sprIds, err := findVMSprIds(o, key)
	if err != nil {
		return nil, err
	}
	qs := o.QueryTable(new(models.VMRecord)).Filter("VMSprId__in", sprIds)
	if !start.IsZero() {
		qs = qs.Filter("Date__gte", start)
	}
	if !end.IsZero() {
		qs = qs.Filter("Date__lte", end)
	}
	recs := make([]*models.VMRecord, 0)
	if _, err := qs.OrderBy("Date", "Id").All(&recs); err != nil {
		return nil, fmt.Errorf("чтение vm_record: %w", err)
	}
	return recs, nil
}

// GetDBVMRecord возвращает записи ВМ (vm_record) за период [Start, End]
// (включительно, по VMRecord.Date) по ключу VMRecordPeriodReq.Key (формат
// Vcenter/Datacenter/Cluster/Vm_Name), в порядке возрастания Date. Не заданная
// граница периода не ограничивает выборку. Если сконфигурированы удалённые
// обработчики (config.Switch.Endpoint), запрос ретранслируется на первый из них,
// вернувший успех (сервис LoadmonDB, адрес GetSrvDBAddr), иначе данные читаются
// из локальной БД.
func (p *serverdb) GetDBVMRecord(_ context.Context, in *pb.VMRecordPeriodReq) (*pb.VMRecordSlice, error) {
	empty := func() *pb.VMRecordSlice { return &pb.VMRecordSlice{Records: make([]*pb.VMRecord, 0)} }
	if in.GetKey() == "" {
		return empty(), errors.New(errDBVMPerfEmptyKey)
	}
	start, end, err := vmRecordPeriod(in)
	if err != nil {
		return empty(), err
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
			recs, err := c.GetDBVMRecord(ctx, in)
			if err == nil { //как только получаем правильный ответ, пересылаем его
				return recs, nil
			}
		}
		return empty(), errors.New(errNotFound) //ничего не найдено
	}

	recs, err := loadDBVMRecords(orm.NewOrm(), in.GetKey(), start, end)
	if err != nil {
		log.Printf("GetDBVMRecord(%s): %v\n", in.GetKey(), err)
		return empty(), err
	}
	out := &pb.VMRecordSlice{Count: int64(len(recs)), Records: make([]*pb.VMRecord, 0, len(recs))}
	for _, r := range recs {
		m := &pb.VMRecord{}
		VMRecordToMessage(r, m)
		out.Records = append(out.Records, m)
	}
	return out, nil
}
