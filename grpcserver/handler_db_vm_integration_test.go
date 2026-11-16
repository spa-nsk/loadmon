package grpcserver

// Интеграционные тесты GetDBVMPerf и GetDBVMRecord: реальные запросы beego ORM
// к sqlite (in-memory, shared cache), сквозной вызов через gRPC (клиент
// grpcclient -> сервер serverdb) и ретрансляция на удалённые обработчики.

import (
	"context"
	"errors"
	"loadmon/config"
	"loadmon/grpcclient"
	"loadmon/models"
	pb "loadmon/protocol"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
	grpc "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var vmDBOnce sync.Once

// setupVMDB подключает in-memory sqlite под алиасом default, регистрирует
// модели vm_spr_record/vm_record/vm_perf и пересоздаёт таблицы (чистая БД на
// каждый тест).
func setupVMDB(t *testing.T) orm.Ormer {
	t.Helper()
	vmDBOnce.Do(func() {
		orm.RegisterModel(new(models.VMSprRecord), new(models.VMRecord), new(models.VMPerfRecord))
		if err := orm.RegisterDataBase("default", "sqlite3", "file:vmdbtest?mode=memory&cache=shared"); err != nil {
			t.Fatalf("RegisterDataBase: %v", err)
		}
	})
	if err := orm.RunSyncdb("default", true, false); err != nil {
		t.Fatalf("RunSyncdb: %v", err)
	}
	return orm.NewOrm()
}

func mustInsert(t *testing.T, o orm.Ormer, m interface{}) int64 {
	t.Helper()
	id, err := o.Insert(m)
	if err != nil {
		t.Fatalf("Insert %T: %v", m, err)
	}
	return id
}

// vmFixture - тестовые данные: идентификаторы записей и время "сейчас".
type vmFixture struct {
	now time.Time
	// записи vm_record ВМ vc1/dc1/cl1/vm1 (Cpu_usage_maximum = маркер)
	rec120h, rec48h, rec30h, rec24h, rec1h int64
}

const (
	keyVM1     = "vc1/dc1/cl1/vm1"
	keyVM2     = "vc1/dc1/cl1/vm2"
	keyVMEmpty = "vc1/dc1/cl1/vm-empty"
	keyVMSlash = "vc1/dc1/cl1/dir/vm"
)

func seedVMs(t *testing.T, o orm.Ormer) vmFixture {
	t.Helper()
	f := vmFixture{now: time.Now().Truncate(time.Second)}
	ago := func(d time.Duration) time.Time { return f.now.Add(-d) }

	spr := func(name string) int64 {
		return mustInsert(t, o, &models.VMSprRecord{Vcenter: "vc1", Datacenter: "dc1", Cluster: "cl1", Vm_Name: name})
	}
	spr1 := spr("vm1")
	spr1dup := spr("vm1") // ВМ удалили и создали заново - вторая запись справочника с тем же ключом
	spr2 := spr("vm2")
	spr("vm-empty")
	sprSlash := spr("dir/vm")

	rec := func(sprId int64, d time.Duration, marker float64) int64 {
		return mustInsert(t, o, &models.VMRecord{VMSprId: sprId, Date: ago(d), Cpu_usage_maximum: marker})
	}
	f.rec120h = rec(spr1, 120*time.Hour, 1)
	f.rec48h = rec(spr1, 48*time.Hour, 2)
	f.rec30h = rec(spr1dup, 30*time.Hour, 5)
	f.rec24h = rec(spr1, 24*time.Hour, 3)
	f.rec1h = rec(spr1, time.Hour, 4)
	rec2 := rec(spr2, time.Hour, 60)
	recSlash := rec(sprSlash, time.Hour, 70)

	perf := func(vmRecId int64, d time.Duration, marker float64) {
		mustInsert(t, o, &models.VMPerfRecord{VMId: vmRecId, Start: ago(d), Cpu_usage: marker, Disk_totalReadLatency: marker * 2})
	}
	// вставка намеренно не по порядку Start
	perf(f.rec1h, 30*time.Minute, 104)
	perf(f.rec1h, 90*time.Minute, 103)
	perf(f.rec24h, 25*time.Hour, 102)
	perf(f.rec30h, 29*time.Hour, 101)
	perf(f.rec48h, 47*time.Hour, 100)
	perf(f.rec48h, 73*time.Hour, 99)   // старше 3 суток - не должна попасть, хотя запись vm_record в окне
	perf(f.rec120h, 100*time.Hour, 98) // запись vm_record вне окна
	perf(rec2, 30*time.Minute, 500)    // чужая ВМ
	perf(recSlash, 10*time.Minute, 700)
	return f
}

func TestGetDBVMPerfDB(t *testing.T) {
	o := setupVMDB(t)
	seedVMs(t, o)
	s := &serverdb{}

	t.Run("только последние 3 суток, по возрастанию Start, обе записи справочника", func(t *testing.T) {
		res, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: keyVM1})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		want := []float64{100, 101, 102, 103, 104}
		if res.Count != int64(len(want)) || len(res.Perf) != len(want) {
			t.Fatalf("Count=%d len=%d, ожидалось %d", res.Count, len(res.Perf), len(want))
		}
		for i, p := range res.Perf {
			if p.CpuUsage != want[i] {
				t.Errorf("позиция %d: CpuUsage=%v, ожидалось %v", i, p.CpuUsage, want[i])
			}
			if p.DiskTotalReadLatency != want[i]*2 {
				t.Errorf("позиция %d: DiskTotalReadLatency=%v, ожидалось %v", i, p.DiskTotalReadLatency, want[i]*2)
			}
			if p.Start == nil || p.VMID == 0 || p.Id == 0 {
				t.Errorf("позиция %d: не заполнены Id/VMID/Start: %v", i, p)
			}
			if i > 0 && p.Start.AsTime().Before(res.Perf[i-1].Start.AsTime()) {
				t.Errorf("позиция %d: нарушен порядок Start", i)
			}
		}
	})

	t.Run("чужие метрики не попадают", func(t *testing.T) {
		res, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: keyVM2})
		if err != nil || len(res.Perf) != 1 || res.Perf[0].CpuUsage != 500 {
			t.Fatalf("vm2: ожидалась одна метрика 500, получено %v, %v", res, err)
		}
	})

	t.Run("имя ВМ со слэшем", func(t *testing.T) {
		res, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: keyVMSlash})
		if err != nil || len(res.Perf) != 1 || res.Perf[0].CpuUsage != 700 {
			t.Fatalf("ожидалась одна метрика 700, получено %v, %v", res, err)
		}
	})

	t.Run("ВМ без метрик - пустой массив без ошибки", func(t *testing.T) {
		res, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: keyVMEmpty})
		if err != nil || res == nil || res.Count != 0 || res.Perf == nil || len(res.Perf) != 0 {
			t.Fatalf("ожидался пустой не nil массив без ошибки, получено %v, %v", res, err)
		}
	})

	t.Run("ошибки", func(t *testing.T) {
		for name, key := range map[string]string{
			"пустой ключ":     "",
			"неполный ключ":   "vc1/dc1/cl1",
			"неизвестная ВМ":  "vc1/dc1/cl1/nope",
			"неизвестный vc":  "vcX/dc1/cl1/vm1",
			"регистр значим?": "VC1/dc1/cl1/vm1",
		} {
			res, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: key})
			if name == "регистр значим?" {
				// фиксируем текущее поведение: сравнение точное (как в controllers)
				if err == nil {
					t.Errorf("%s: ключ в другом регистре неожиданно найден", name)
				}
				continue
			}
			if err == nil {
				t.Errorf("%s (%q): ожидалась ошибка", name, key)
			}
			if res == nil || res.Perf == nil {
				t.Errorf("%s: при ошибке ответ должен быть не nil с пустым массивом", name)
			}
		}
	})
}

func TestGetDBVMRecordDB(t *testing.T) {
	o := setupVMDB(t)
	f := seedVMs(t, o)
	s := &serverdb{}
	ago := func(d time.Duration) *timestamppb.Timestamp { return timestamppb.New(f.now.Add(-d)) }

	markers := func(res *pb.VMRecordSlice) []float64 {
		out := make([]float64, 0, len(res.Records))
		for _, r := range res.Records {
			out = append(out, r.CpuUsageMaximum)
		}
		return out
	}
	equal := func(a, b []float64) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	cases := []struct {
		name       string
		key        string
		start, end *timestamppb.Timestamp
		want       []float64
	}{
		{"без границ - вся история по возрастанию Date", keyVM1, nil, nil, []float64{1, 2, 5, 3, 4}},
		{"только Start", keyVM1, ago(48 * time.Hour), nil, []float64{2, 5, 3, 4}},
		// Верхняя граница проверяется со сдвигом +1с от Date записи: sqlite хранит
		// время как "2006-01-02 15:04:05+00:00", а beego передаёт параметр без
		// суффикса "+00:00", поэтому при ТОЧНОМ совпадении строк `__lte`/`__lt`
		// на sqlite не срабатывают (на PostgreSQL сравниваются timestamp, и
		// граница включительная). Поэтому на sqlite нельзя проверить включительность границ при точном\n		// совпадении (для Start `__gt` и `__gte` неразличимы, для End - наоборот);\n		// тесты проверяют фильтрацию по окну, а включительность - за PostgreSQL.
		{"только End", keyVM1, nil, ago(24*time.Hour - time.Second), []float64{1, 2, 5, 3}},
		{"End раньше последней записи окна", keyVM1, nil, ago(25 * time.Hour), []float64{1, 2, 5}},
		{"Start и End", keyVM1, ago(48 * time.Hour), ago(24*time.Hour - time.Second), []float64{2, 5, 3}},
		{"узкое окно без записей", keyVM1, ago(20 * time.Hour), ago(10 * time.Hour), []float64{}},
		{"окно в будущем", keyVM1, timestamppb.New(f.now.Add(time.Hour)), timestamppb.New(f.now.Add(2 * time.Hour)), []float64{}},
		{"узкое окно вокруг одной записи", keyVM1, ago(24*time.Hour + time.Second), ago(24*time.Hour - time.Second), []float64{3}},
		{"другая ВМ", keyVM2, nil, nil, []float64{60}},
		{"имя со слэшем", keyVMSlash, nil, nil, []float64{70}},
		{"ВМ без записей", keyVMEmpty, nil, nil, []float64{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := s.GetDBVMRecord(context.Background(), &pb.VMRecordPeriodReq{Key: c.key, Start: c.start, End: c.end})
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if res.Records == nil || res.Count != int64(len(res.Records)) {
				t.Fatalf("Count=%d len=%d (Records nil=%v)", res.Count, len(res.Records), res.Records == nil)
			}
			if got := markers(res); !equal(got, c.want) {
				t.Fatalf("получены маркеры %v, ожидались %v", got, c.want)
			}
		})
	}

	t.Run("поля записи заполнены", func(t *testing.T) {
		res, err := s.GetDBVMRecord(context.Background(), &pb.VMRecordPeriodReq{Key: keyVM1, Start: ago(2 * time.Hour)})
		if err != nil || len(res.Records) != 1 {
			t.Fatalf("ожидалась одна запись, получено %v, %v", res, err)
		}
		r := res.Records[0]
		if r.Id != f.rec1h || r.VMSprId == 0 || r.Date == nil || !r.Date.AsTime().Equal(f.now.Add(-time.Hour)) {
			t.Fatalf("неверные Id/VMSprId/Date: %v (ожидался Id=%d)", r, f.rec1h)
		}
	})

	t.Run("ошибки", func(t *testing.T) {
		bad := map[string]*pb.VMRecordPeriodReq{
			"пустой ключ":    {},
			"неполный ключ":  {Key: "vc1/dc1"},
			"неизвестная ВМ": {Key: "vc1/dc1/cl1/nope"},
			"End < Start":    {Key: keyVM1, Start: ago(time.Hour), End: ago(2 * time.Hour)},
		}
		for name, req := range bad {
			res, err := s.GetDBVMRecord(context.Background(), req)
			if err == nil {
				t.Errorf("%s: ожидалась ошибка", name)
			}
			if res == nil || res.Records == nil {
				t.Errorf("%s: при ошибке ответ должен быть не nil с пустым массивом", name)
			}
		}
	})
}

// startDBServer поднимает gRPC-сервер LoadmonDB на свободном порту loopback и
// возвращает порт.
func startDBServer(t *testing.T, srv pb.LoadmonDBServer) int64 {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	gs := grpc.NewServer()
	pb.RegisterLoadmonDBServer(gs, srv)
	go gs.Serve(lis)
	t.Cleanup(gs.Stop)
	return int64(lis.Addr().(*net.TCPAddr).Port)
}

// TestGetDBVMEndToEnd - вызов через сеть: клиентские обёртки grpcclient ->
// gRPC -> serverdb -> БД.
func TestGetDBVMEndToEnd(t *testing.T) {
	o := setupVMDB(t)
	f := seedVMs(t, o)
	port := startDBServer(t, &serverdb{})

	oldAddr, oldPort := config.ConfigData.GRPCCliDBAddr, config.ConfigData.GRPCCliDBPort
	config.ConfigData.GRPCCliDBAddr, config.ConfigData.GRPCCliDBPort = "127.0.0.1", port
	t.Cleanup(func() { config.ConfigData.GRPCCliDBAddr, config.ConfigData.GRPCCliDBPort = oldAddr, oldPort })

	t.Run("GetDBVMPerf", func(t *testing.T) {
		recs, err := grpcclient.GetDBVMPerf(keyVM1)
		if err != nil {
			t.Fatalf("ошибка: %v", err)
		}
		if len(recs) != 5 {
			t.Fatalf("ожидалось 5 метрик, получено %d", len(recs))
		}
		for i, r := range recs {
			if want := float64(100 + i); r.Cpu_usage != want || r.Disk_totalReadLatency != want*2 {
				t.Errorf("позиция %d: Cpu_usage=%v Disk_totalReadLatency=%v, ожидалось %v/%v", i, r.Cpu_usage, r.Disk_totalReadLatency, want, want*2)
			}
			if r.Start.IsZero() || r.VMId == 0 {
				t.Errorf("позиция %d: не заполнены Start/VMId: %+v", i, r)
			}
		}
		if _, err := grpcclient.GetDBVMPerf("vc1/dc1/cl1/nope"); err == nil {
			t.Error("неизвестная ВМ: ожидалась ошибка")
		}
	})

	t.Run("GetDBVMRecord", func(t *testing.T) {
		recs, err := grpcclient.GetDBVMRecord(keyVM1, f.now.Add(-48*time.Hour), f.now.Add(-24*time.Hour+time.Second))
		if err != nil {
			t.Fatalf("ошибка: %v", err)
		}
		want := []float64{2, 5, 3}
		if len(recs) != len(want) {
			t.Fatalf("ожидалось %d записей, получено %d", len(want), len(recs))
		}
		for i, r := range recs {
			if r.Cpu_usage_maximum != want[i] || r.Date.IsZero() || r.VMSprId == 0 {
				t.Errorf("позиция %d: %+v", i, r)
			}
		}
		all, err := grpcclient.GetDBVMRecord(keyVM1, time.Time{}, time.Time{})
		if err != nil || len(all) != 5 {
			t.Fatalf("без границ ожидалось 5 записей, получено %d, %v", len(all), err)
		}
		if _, err := grpcclient.GetDBVMRecord("vc1/dc1/cl1/nope", time.Time{}, time.Time{}); err == nil {
			t.Error("неизвестная ВМ: ожидалась ошибка")
		}
	})
}

// fakeDB - удалённый обработчик LoadmonDB для проверки ретрансляции.
type fakeDB struct {
	pb.UnimplementedLoadmonDBServer
	mu      sync.Mutex
	perfReq *pb.VMReqKey
	recReq  *pb.VMRecordPeriodReq
	fail    bool
}

func (f *fakeDB) GetDBVMPerf(_ context.Context, in *pb.VMReqKey) (*pb.VMPerfRecordSlice, error) {
	f.mu.Lock()
	f.perfReq = in
	f.mu.Unlock()
	if f.fail {
		return nil, errors.New("fake: failure")
	}
	return &pb.VMPerfRecordSlice{Count: 1, Perf: []*pb.VMPerfRecord{{Id: 42, CpuUsage: 7}}}, nil
}

func (f *fakeDB) GetDBVMRecord(_ context.Context, in *pb.VMRecordPeriodReq) (*pb.VMRecordSlice, error) {
	f.mu.Lock()
	f.recReq = in
	f.mu.Unlock()
	if f.fail {
		return nil, errors.New("fake: failure")
	}
	return &pb.VMRecordSlice{Count: 1, Records: []*pb.VMRecord{{Id: 43, CpuUsageMaximum: 8}}}, nil
}

func setEndpoints(t *testing.T, ports ...int64) {
	t.Helper()
	config.Switch.Endpoint.Clear()
	for i, p := range ports {
		config.Switch.Endpoint.Set(string(rune('a'+i)), &config.EndpointRecord{GRPCSrvDBAddr: "127.0.0.1", GRPCSrvDBPort: p})
	}
	t.Cleanup(config.Switch.Endpoint.Clear)
}

func TestGetDBVMRelay(t *testing.T) {
	s := &serverdb{}
	req := &pb.VMRecordPeriodReq{
		Key:   keyVM1,
		Start: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		End:   timestamppb.New(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)),
	}

	t.Run("ответ удалённого обработчика пересылается, запрос передаётся без изменений", func(t *testing.T) {
		fake := &fakeDB{}
		setEndpoints(t, startDBServer(t, fake))

		perf, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: keyVM1})
		if err != nil || perf.Count != 1 || perf.Perf[0].Id != 42 || perf.Perf[0].CpuUsage != 7 {
			t.Fatalf("GetDBVMPerf: %v, %v", perf, err)
		}
		if fake.perfReq.GetKey() != keyVM1 {
			t.Errorf("удалённый обработчик получил ключ %q", fake.perfReq.GetKey())
		}

		rec, err := s.GetDBVMRecord(context.Background(), req)
		if err != nil || rec.Count != 1 || rec.Records[0].Id != 43 {
			t.Fatalf("GetDBVMRecord: %v, %v", rec, err)
		}
		if fake.recReq.GetKey() != keyVM1 ||
			!fake.recReq.GetStart().AsTime().Equal(req.Start.AsTime()) ||
			!fake.recReq.GetEnd().AsTime().Equal(req.End.AsTime()) {
			t.Errorf("удалённый обработчик получил искажённый запрос: %v", fake.recReq)
		}
	})

	t.Run("отказавший обработчик пропускается, берётся следующий", func(t *testing.T) {
		setEndpoints(t, startDBServer(t, &fakeDB{fail: true}), startDBServer(t, &fakeDB{}))
		if perf, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: keyVM1}); err != nil || perf.Count != 1 {
			t.Fatalf("GetDBVMPerf: %v, %v", perf, err)
		}
		if rec, err := s.GetDBVMRecord(context.Background(), req); err != nil || rec.Count != 1 {
			t.Fatalf("GetDBVMRecord: %v, %v", rec, err)
		}
	})

	t.Run("все обработчики отказали - ошибка и пустой ответ", func(t *testing.T) {
		setEndpoints(t, startDBServer(t, &fakeDB{fail: true}))
		perf, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{Key: keyVM1})
		if err == nil || perf == nil || perf.Perf == nil || len(perf.Perf) != 0 {
			t.Errorf("GetDBVMPerf: ожидалась ошибка и пустой ответ, получено %v, %v", perf, err)
		}
		rec, err := s.GetDBVMRecord(context.Background(), req)
		if err == nil || rec == nil || rec.Records == nil || len(rec.Records) != 0 {
			t.Errorf("GetDBVMRecord: ожидалась ошибка и пустой ответ, получено %v, %v", rec, err)
		}
	})

	t.Run("некорректный запрос не ретранслируется", func(t *testing.T) {
		fake := &fakeDB{}
		setEndpoints(t, startDBServer(t, fake))
		if _, err := s.GetDBVMRecord(context.Background(), &pb.VMRecordPeriodReq{Key: keyVM1, Start: req.End, End: req.Start}); err == nil {
			t.Error("End < Start: ожидалась ошибка")
		}
		if _, err := s.GetDBVMPerf(context.Background(), &pb.VMReqKey{}); err == nil {
			t.Error("пустой ключ: ожидалась ошибка")
		}
		if fake.recReq != nil || fake.perfReq != nil {
			t.Error("некорректные запросы не должны доходить до удалённого обработчика")
		}
	})
}
