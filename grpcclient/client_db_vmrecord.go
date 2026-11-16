package grpcclient

import (
	"errors"
	"loadmon/models"
	pb "loadmon/protocol"
	"log"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// newVMRecordPeriodReq собирает запрос GetDBVMRecord. Нулевое время
// (time.Time{}) означает "граница не задана" и передаётся как nil.
func newVMRecordPeriodReq(key string, start, end time.Time) *pb.VMRecordPeriodReq {
	req := &pb.VMRecordPeriodReq{Key: key}
	if !start.IsZero() {
		req.Start = timestamppb.New(start)
	}
	if !end.IsZero() {
		req.End = timestamppb.New(end)
	}
	return req
}

// GetDBVMRecord возвращает записи виртуальной машины (vm_record) за период
// [start, end] включительно (по VMRecord.Date) через сервис LoadmonDB по ключу
// ВМ key (формат Vcenter/Datacenter/Cluster/Vm_Name), в порядке возрастания
// Date - подробности в grpcserver.(*serverdb).GetDBVMRecord. Нулевое время
// (time.Time{}) в start/end - граница не задана. Отсутствие записей не ошибка
// (пустой срез). Возвращает ошибку при пустом ключе, end раньше start, ошибке
// подключения/RPC или если ВМ не найдена в справочнике.
func GetDBVMRecord(key string, start, end time.Time) ([]*models.VMRecord, error) {
	if key == "" {
		return nil, errors.New("не указан ключ виртуальной машины")
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return nil, errors.New("окончание периода раньше его начала")
	}
	c, conn, ctx, cancel, err := dialDB()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer cancel()
	res, err := c.GetDBVMRecord(ctx, newVMRecordPeriodReq(key, start, end))
	if err != nil {
		log.Printf("GetDBVMRecord(%s): %v", key, err)
		return nil, err
	}
	return messageToVMRecords(res), nil
}

// messageToVMRecords преобразует ответ GetDBVMRecord в срез моделей
// (nil-элементы пропускаются, результат никогда не nil).
func messageToVMRecords(in *pb.VMRecordSlice) []*models.VMRecord {
	out := make([]*models.VMRecord, 0, len(in.GetRecords()))
	for _, m := range in.GetRecords() {
		if m == nil {
			continue
		}
		rec := &models.VMRecord{}
		MessageToVMRecord(m, rec)
		out = append(out, rec)
	}
	return out
}
