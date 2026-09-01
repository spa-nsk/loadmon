package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
	vsphere "loadmon/vspheremetrics"

	//"loadmon/itsystem"
	models "loadmon/models"
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

var (
	//nilPointerError string = "Ожидался указатель на переменную, отличный от nil"
	timeOutError string = "Превышено время ожидания"
)

// vmPerfRetention - срок хранения детальных ("сырых") метрик загрузки ВМ
// (таблица vm_perf, models.VMPerfRecord) - по требованию 3 суток; за более
// давние периоды доступны только суточные агрегированные записи vm_record
// (models.VMRecord, хранятся без отдельного ограничения времени, как и
// раньше). См. также controllers/adminmetricsexport.go - выбор источника
// данных (детальный vm_perf/zabbix_item или суточный vm_record) для
// единообразной выгрузки метрик опирается на эту же границу в 3 суток.
const vmPerfRetention = time.Hour * 24 * 3

// ClearOldMetric удаляет из таблиц vm_perf (детальные метрики ВМ, срок
// хранения - vmPerfRetention, см. выше), cluster_perf и insight_vm_perf
// записи метрик производительности старше срока хранения, порциями по 1000
// строк.
func ClearOldMetric() {

	currentTime := time.Now().Local() //запоминаем текущее время
	currentTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.Local)
	o := orm.NewOrm()
	count := int64(0)
	for {
		res, err := o.Raw("DELETE FROM vm_perf WHERE id IN (SELECT id FROM vm_perf WHERE start <=? LIMIT 1000)", currentTime.Add(-vmPerfRetention)).Exec()
		if err != nil {
			fmt.Println("Ошибка удаления старых записей из таблицы vm_perf", err)
			break
		} else {
			num, _ := res.RowsAffected()
			count += num
			if num == 0 {
				break
			}
		}
	}
	fmt.Println("Удалено :", count, "записей старее", vmPerfRetention.String(), "из таблицы vm_perf")
	count = int64(0)
	for {
		res, err := o.Raw("DELETE FROM cluster_perf WHERE id IN (SELECT id FROM cluster_perf WHERE start <=? LIMIT 1000)", currentTime.Add(-time.Hour*24)).Exec()
		if err != nil {
			fmt.Println("Ошибка удаления старых записей из таблицы cluster_perf", err)
			break
		} else {
			num, _ := res.RowsAffected()
			count += num
			if num == 0 {
				break
			}
		}
	}
	fmt.Println("Удалено :", count, "записей старее 1 дня из таблицы cluster_perf")
	count = int64(0)
	for {
		res, err := o.Raw("DELETE FROM insight_vm_perf WHERE id IN (SELECT id FROM insight_vm_perf WHERE time <=? LIMIT 1000)", currentTime.Add(-time.Hour*24)).Exec()
		if err != nil {
			fmt.Println("Ошибка удаления старых записей из таблицы insight_vm_perf", err)
			break
		} else {
			num, _ := res.RowsAffected()
			count += num
			if num == 0 {
				break
			}
		}
	}
	fmt.Println("Удалено :", count, "записей старее 1 дня из таблицы insight_vm_perf")
}

// SaveClusterInfo() функция для сохранения данных по кластерам, хостам и виртуальным машинам
func SaveMetricData() {
	fmtdate := "02.01.2006 15:04:05.000"
	currentTime := time.Now().Local() //запоминаем текущее время
	fmt.Println(currentTime.Format(fmtdate), "Запись данных кластеров, хостов и виртуальных машин в базу")

	clusterSpr, err := grpcclient.LoadClusterSpr("all", "all", "all")
	if err != nil || clusterSpr == nil {
		return
	}

	clList := grpcclient.LoadClusterList("all", "all")
	count := len(clList)

	for i, clShort := range clList {
		//cl := grpcclient.GetClusterMedium(clShort.Vcenter, clShort.Datacenter, clShort.Name)
		cl := grpcclient.GetClusterMedium(clShort.Vcenter, clShort.Datacenter, clShort.Name)
		if cl == nil {
			fmt.Println("Ошибка получения данных кластера", clShort.Vcenter, clShort.Datacenter, clShort.Name)
			continue
		}
		notData := "нет данных"

		if cl.Datacenter == notData || cl.Name == notData || cl.HostFlag { //если не заполнены данные по датацентру, кластеру или это одиночный хост
			log.Println("Пропускаем одиночный хост или если данные не заполнены: ", i+1, "/", count, cl.Name)
			ClearClusterMetricData(cl.Name)
			continue
		}

		log.Println("запись кластера: ", i+1, "/", count, cl.Name)
		SaveClusterData(cl, clusterSpr, currentTime)
	}

	// После сохранения данных по всем кластерам (в т.ч. виртуальным машинам,
	// см. SaveClusterData -> SaveVMRecord) приводим в порядок статусы
	// справочника серверов - см. комментарий к функции.
	CheckServerSprRecordsExistence()

	//  for !atomic.CompareAndSwapInt32(&flDelOldVMMetrics, 1, 0) {
	//      runtime.Gosched()
	//  }

	fmt.Println(time.Now().Local().Format(fmtdate), "Запись данных завершена.")
}

// ClearMetricData определяет период (текущие либо предыдущие сутки) и запускает очистку
// метрик производительности за этот период через InitPerfMetricPeriodAndReloadConfig.
func ClearMetricData() {
	var (
		now, firstTime, endTime time.Time
	)
	currentTime := time.Now().Local() //запоминаем текущее время

	now = currentTime

	mode := 0

	// now.Hour() < 4 - недавно наступила полночь (планово вызывается около
	// 02:59, через ~3 часа после ночного cmdb.SaveMetricData(), см.
	// metrics_grpc/metrics_grpc.go) - к этому моменту данные за ПРОШЕДШИЕ
	// сутки уже сохранены в БД (vm_record/host_record/cluster_record),
	// поэтому можно безопасно очистить их из in-memory Perf. Если функция
	// будет вызвана в другое время суток (например, вручную/из другого
	// цикла) - очищаем текущие, ещё не сохранённые сутки.
	//
	// ИСПРАВЛЕНО 2026-09-01: было `now.Hour() > 23`, что для Go time.Time
	// НИКОГДА не выполняется (Hour() всегда возвращает 0-23) - ветка mode=1
	// была недостижимым мёртвым кодом, и функция всегда работала в режиме
	// mode=2 ("вчера"). При фиксированном расписании (~02:59) это давало
	// правильный результат случайно, но при изменении времени запуска в
	// будущем привело бы к очистке не того периода без каких-либо ошибок в
	// логах. См. kontext/05-conventions-and-gotchas.md.
	if now.Hour() < 4 {
		mode = 2
		now = now.AddDate(0, 0, -1)
		firstTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		endTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 59, time.Local)
	} else {
		mode = 1
		firstTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		endTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 59, time.Local)
	}

	InitPerfMetricPeriodAndReloadConfig(mode, firstTime, endTime) //очистка метрик
}

// ClearClusterMetricData определяет период (текущие либо предыдущие сутки) и очищает
// метрики производительности указанного кластера через grpcclient.ClearMetric.
func ClearClusterMetricData(cluster string) {
	var (
		now, firstTime, endTime time.Time
	)
	currentTime := time.Now().Local() //запоминаем текущее время

	now = currentTime

	mode := 0

	if now.Hour() > 23 {
		mode = 1
		firstTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		endTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 59, time.Local)
	} else {
		mode = 2
		now = now.AddDate(0, 0, -1)
		firstTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		endTime = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 59, time.Local)
	}

	//InitPerfMetricPeriodAndReloadConfig(mode, firstTime, endTime) //очистка метрик

	fmtdate := "02.01.2006 15:04:05.000"
	fmt.Print(mode, ") Очистка метрик с ", firstTime.Format(fmtdate), " по ", endTime.Format(fmtdate), "\n")
	s, err := grpcclient.ClearMetric(cluster, firstTime, endTime)
	if err != nil || s.State != 0 {
		fmt.Print("Очистка метрик завершилась с ошибкой:", err, s.Details, "\n")
	}
}

// инициализация и перезагрузка метрик
func InitPerfMetricAndReloadConfig() {
	//приостановить сбор метрик
	//перезагрузить конфигурацию
	//возобновить сбор метрик
	//удалить метрики за прошлые сутки
	//произвести необходимые обновления данных
	fmtdate := "02.01.2006 15:04:05.000"
	now := time.Now().Local()

	if now.Hour() > 23 {
		firstTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 59, time.Local)
		fmt.Print("(1) Очистка метрик с ", firstTime.Format(fmtdate), " по ", endTime.Format(fmtdate), "\n")
		grpcclient.ClearMetric("all", firstTime, endTime)
	} else {
		now = now.AddDate(0, 0, -1)
		firstTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 59, time.Local)
		fmt.Print("(2) Очистка метрик с ", firstTime.Format(fmtdate), " по ", endTime.Format(fmtdate), "\n")
		grpcclient.ClearMetric("all", firstTime, endTime)
	}

	/*
		firstTime := time.Date(now.Year(), now.Month(), now.Day(), 9, 7, 0, 0, time.Local)
		endTime := time.Date(now.Year(), now.Month(), now.Day(), 10, 11, 0, 0, time.Local)
		grpcclient.ClearMetric(firstTime, endTime)
	*/
}

// InitPerfMetricPeriodAndReloadConfig очищает метрики производительности по всем кластерам
// за указанный период времени через grpcclient.ClearMetric.
func InitPerfMetricPeriodAndReloadConfig(mode int, firstTime, endTime time.Time) error {
	//приостановить сбор метрик
	//перезагрузить конфигурацию
	//возобновить сбор метрик
	//удалить метрики за прошлые сутки
	//произвести необходимые обновления данных
	fmtdate := "02.01.2006 15:04:05.000"
	fmt.Print(mode, " Очистка метрик с ", firstTime.Format(fmtdate), " по ", endTime.Format(fmtdate), "\n")
	s, err := grpcclient.ClearMetric("all", firstTime, endTime)
	if err != nil || s.State != 0 {
		fmt.Print("Очистка метрик завершилась с ошибкой:", err, s.Details, "\n")
		return err
	}
	return nil
}

// InitPerfMetricPeriodAndReloadConfigByKey очищает метрики производительности указанного
// кластера (по ключу vsphere.ClusterShort) за заданный период времени.
func InitPerfMetricPeriodAndReloadConfigByKey(mode int, key vsphere.ClusterShort, firstTime, endTime time.Time) {
	//приостановить сбор метрик
	//перезагрузить конфигурацию
	//возобновить сбор метрик
	//удалить метрики за прошлые сутки
	//произвести необходимые обновления данных
	fmtdate := "02.01.2006 15:04:05.000"
	fmt.Print(mode, " Очистка метрик с ", firstTime.Format(fmtdate), " по ", endTime.Format(fmtdate), "\n")
	grpcclient.ClearAllMetricByKey(key.Vcenter, key.Datacenter, key.Name, "all", firstTime, endTime)
}

// сохраняем данные кластера
func SaveClusterData(cl *vsphere.Cluster, clusterSpr map[string]*models.ClusterSprRecords, currentTime time.Time) {
	// поиск информации о кластере в справочнике кластеров
	clId := int64(0)
	//produtcion := int(0)
	cls, ok := clusterSpr[fmt.Sprintf("%s/%s/%s", cl.Vcenter, cl.Datacenter, cl.Name)]
	if ok { // кластер найден
		clId = cls.Id // запоминаем id код из справочника
		//production := cls.Production
		cls.ClusterId = cl.ClusterID
		cls.TypeWork = cl.TypeWork
		cls.Time_Mfy = currentTime
		cls.Stat = 1
		_, status := grpcclient.UpdateClusterSprRecord(cls)
		if status != 1 { //ошибка вставки записи кластера в справочник
			return //пропускаем этот кластер
		}

	} else {
		//rec := models.ClusterSprRecords{Vcenter: cl.Vcenter, Datacenter: cl.Datacenter, Name: cl.Name, Time_Mfy: time.Now().Local()}
		rec := models.ClusterSprRecords{Vcenter: cl.Vcenter, Datacenter: cl.Datacenter, Name: cl.Name, TypeWork: cl.TypeWork, Time_Mfy: currentTime, ClusterId: cl.ClusterID}
		status := 0
		rec.Stat = 1
		clId, status = grpcclient.AddClusterSprRecord(&rec)
		if status != 1 { //ошибка вставки записи кластера в справочник
			return //пропускаем этот кластер
		}
	}
	clusterId, err := SaveClusterRecord(cl, clId, currentTime)
	if err != nil {
		return
	}
	SaveStorageVolumes(cl)
	SaveHostRecord(cl, clusterId, currentTime)
	SaveVMRecord(cl, clusterId, currentTime)
}

// возврат mapStorSys, mapStorSysWWN
// LoadStorageSpr загружает из справочника storage_systems активные системы хранения и
// строит две карты: по краткому имени СХД и по маске WWN, для последующего сопоставления
// томов с системами хранения.
