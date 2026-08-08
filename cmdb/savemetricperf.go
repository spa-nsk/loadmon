package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
	vsphere "loadmon/vspheremetrics"
	"strings"

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

func ClearOldMetric() {

	currentTime := time.Now().Local() //запоминаем текущее время
	currentTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.Local)
	o := orm.NewOrm()
	count := int64(0)
	for {
		res, err := o.Raw("DELETE FROM vm_perf WHERE id IN (SELECT id FROM vm_perf WHERE start <=? LIMIT 1000)", currentTime.Add(-time.Hour*24)).Exec()
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
	fmt.Println("Удалено :", count, "записей старее 1 дня из таблицы vm_perf")
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

	//  for !atomic.CompareAndSwapInt32(&flDelOldVMMetrics, 1, 0) {
	//      runtime.Gosched()
	//  }

	fmt.Println(time.Now().Local().Format(fmtdate), "Запись данных завершена.")
}

func ClearMetricData() {
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

	InitPerfMetricPeriodAndReloadConfig(mode, firstTime, endTime) //очистка метрик
}

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
func LoadStorageSpr() (map[string]*models.StorageSystems, map[string]*models.StorageSystems) {
	o := orm.NewOrm()
	//o.Using("default")
	var storSys []*models.StorageSystems
	qs := o.QueryTable("storage_systems")
	qs = qs.OrderBy("Id")
	num, err := qs.All(&storSys)
	if err != nil || num == 0 {
		return nil, nil
	}
	mapStorSys := make(map[string]*models.StorageSystems)
	mapStorSysWWN := make(map[string]*models.StorageSystems)
	for i, shd := range storSys {
		if shd.Power != 1 {
			continue
		} //пропускаем выключенные СХД
		mapStorSys[shd.ShortName] = storSys[i] //запоминаем в карте по краткому имени СХД

		/*
			wwnMask := strings.Split(storSys[i].WWNMask, "\n")
			for _, mask := range wwnMask {
				wwn := GetWWNMaskKey(mask)
				mapStorSysWWN[wwn] = storSys[i]
			}
		*/
		for mask := range strings.SplitSeq(storSys[i].WWNMask, "\n") {
			wwn := GetWWNMaskKey(mask)
			mapStorSysWWN[wwn] = storSys[i]
		}

	}
	return mapStorSys, mapStorSysWWN
}

func SaveStorageVolumes(cl *vsphere.Cluster) {
	ds := grpcclient.GetClusterStorages(cl.Vcenter, cl.Datacenter, cl.Name)

	o := orm.NewOrm()

	for _, st := range ds {

		trn, err := o.Begin()
		//defer trn.RollbackUnlessCommit()
		if err != nil {
			fmt.Println("Ошибка начала транзакции ", err)
			return
		}

		volume := models.StorageVolumes{}
		volume.UID = strings.ToUpper(st.UID)
		foundVolume := false
		err = trn.Read(&volume, "UID")
		if err == nil {
			foundVolume = true
		}
		ImportStorageCL(cl, &volume)
		ImportStorage(st, &volume)
		if foundVolume {
			_, err = trn.Update(&volume)
			if err != nil {
				fmt.Println("Ошибка обновления записи StorageVolumes UID=", st.UID, err)
				trn.Rollback()
				continue
			}
		} else {
			_, err = trn.Insert(&volume)
			if err != nil {
				fmt.Println("Ошибка вставки записи StorageVolumes UID=", st.UID, err)
				trn.Rollback()
				continue
			}
		}
		err = trn.Commit()
		if err != nil {
			fmt.Println("Ошибка завершения транзакции", err)
			trn.Rollback()
		}
	}
}

func ImportStorageCL(cl *vsphere.Cluster, volume *models.StorageVolumes) {
	volume.Vcenter = cl.Vcenter
	volume.Datacenter = cl.Datacenter
	volume.Cluster = cl.Name

	//добавить получение списка хостов для кластера сейчас пустой список
	hosts := grpcclient.GetClusterHost(cl.Vcenter, cl.Datacenter, cl.Name)
	for i, host := range hosts {
		hn, _ := SplitNameFQDN(host.Name)
		if i == 0 {
			volume.Server = hn
		} else {
			volume.Server = fmt.Sprintf("%s;%s", volume.Server, hn)
		}
	}
}

func ImportStorage(from *vsphere.Datastore, to *models.StorageVolumes) {
	to.Status = 1
	to.UID = strings.ToUpper(from.UID)
	to.VirtualName = from.StorageName
	to.DiskType = from.Type
	to.CapacityGb = from.CapacityGb
	to.FreeSpaceGb = from.FreeSpaceGb
	to.PercentUsed = from.PercentUsed
	to.Time_Mfy = time.Now().Local()
}

func SaveClusterRecord(cl *vsphere.Cluster, clId int64, currentTime time.Time) (int64, error) {
	//flMetrics := int32(0)
	//perf := make([]*vsphere.CLPerf, 0)
	//go func() {
	perf := grpcclient.LoadClusterMetrics(cl.Vcenter, cl.Datacenter, cl.Name)
	//atomic.StoreInt32(&flMetrics, int32(1))
	//}()

	o := orm.NewOrm()

	// формируем запись по кластеру и вставляем в базу данных (по одной записи на кластер в сутки с итоговыми данными за сутки)
	//clusterRecord := models.ClusterRecord{Name: cl.Name, ClusterId: clId, Date: time.Now().Local()}
	clusterRecord := models.ClusterRecord{Name: cl.Name, ClusterId: clId, Date: currentTime}
	importClusterRecord(&clusterRecord, cl)
	clusterId, err := o.Insert(&clusterRecord) // в переменной clusterId запоминаем id записи о кластере, далее этот номер будет использоваться для связи с хостами, харнилищами и виртуальными
	if err != nil {
		//fmt.Println("Ошибка вставки записи по кластеру", cl.Name, err)
		return 0, err
	}
	//ждем окончания запроса по метрикам
	//for i := int32(0); !atomic.CompareAndSwapInt32(&flMetrics, 1, 0); i++ { //ожидаем 1 в переменной flEnd
	//	if i >= atomic.LoadInt32(&vsphere.TimeOut) {
	//		fmt.Println(timeOutError, cl.Name)
	//		break
	//	}
	//	time.Sleep(time.Millisecond)
	//runtime.Gosched() //передаём управление планировщику
	// }

	//записываем метрики по кластеру
	for _, clPerf := range perf {
		clusterPerf := models.ClusterPerf{ClusterId: clusterId}
		ImportClusterPerf(&clusterPerf, clPerf)
		_, err := o.Insert(&clusterPerf)
		if err != nil {
			//fmt.Println("Ошибка вставки записи метрик по кластеру", cl.Name, err)
			return clusterId, err
		}
	}
	return clusterId, nil
}

func CheckEnclosureSprRecord(host *vsphere.HostInfo) (int64, error) {
	if host == nil {
		return 0, nil
	}
	if host.EnclosureSerialNumber == "" {
		return 0, nil
	}

	o := orm.NewOrm()
	blhw := models.BladeSystems{SerialNum: host.EnclosureSerialNumber}
	bladeSystemID := int64(0)
	//ищем запись в справочнике
	if err := o.Read(&blhw, "SerialNum"); err == nil {
		bladeSystemID = blhw.Id
	} else { //создаем пустую при отсутствии
		blhw.Name = "blade system for host: " + host.Name
		blhw.InventoryNum = host.EnclosureSerialNumber
		blhw.SerialNum = host.EnclosureSerialNumber
		zeroDate := time.Date(1901, time.January, 1, 0, 0, 0, 0, time.Local)
		blhw.DateContractDoc = zeroDate
		blhw.DateInvoice = zeroDate
		blhw.DateService = zeroDate
		blhw.Id_User = 0
		blhw.Time_Mfy = time.Now().Local()
		bladeSystemID, err = o.Insert(&blhw)
		if err != nil {
			//fmt.Println("Ошибка вставки записи о корзине с SN ", host.EnclosureSerialNumber, err)
			return 0, err
		}
	}
	return bladeSystemID, nil
}

func SaveHostRecord(cl *vsphere.Cluster, clusterId int64, currentTime time.Time) {
	o := orm.NewOrm()

	hsKeys := grpcclient.LoadHostListKey(cl.Vcenter, cl.Datacenter, cl.Name, "all")
	for _, key := range hsKeys {
		//получить аппаратную конфигурацию хоста и серийный номер корзины, при наличие
		//log.Println("запись метрик хоста: ", cl.Name, host.Name)
		host := grpcclient.GetHostByKey(key)
		if host == nil {
			fmt.Println("Хост ", key, " не найден")
			return
		}
		if host.SerialNumber == "" {
			//если нет серийного номера то выключен или в странном состоянии ))) пропускаем
			continue
		}

		serverId, err := UpdateHostSprRecord(*host, cl) //поиск в справочнике, при отсутствии вставка записи
		if err != nil {                                 //если ошибка то пропускаем хост
			continue
		}

		serverSprRecord := models.ServerSprRecords{Id: serverId}

		//формируем запись для метрик хоста
		//hostRecord := models.HostRecord{ClusterId: clusterId, Date: time.Now().Local()} // запоминаем связку с кластером и текущее время записи
		hostRecord := models.HostRecord{ClusterId: clusterId, Date: currentTime} // запоминаем связку с кластером и текущее время записи
		importHostRecord(host, &hostRecord)
		hostRecord.ServerId = serverSprRecord.Id // запоминаем ссылку на запись в справочнике серверов

		// вставляем запись о хосте в базу данных
		hostId := int64(0)
		if hostId, err = o.Insert(&hostRecord); err != nil {
			fmt.Println("Ошибка вставки записи данных хоста: ", err)
		}
		_ = hostId
	}
}

func SaveHostRecordOld(cl *vsphere.Cluster, clusterId int64, currentTime time.Time) {
	o := orm.NewOrm()

	hsKeys := grpcclient.LoadHostListKey(cl.Vcenter, cl.Datacenter, cl.Name, "all")
	for _, key := range hsKeys {
		//получить аппаратную конфигурацию хоста и серийный номер корзины, при наличие
		//log.Println("запись метрик хоста: ", cl.Name, host.Name)
		host := grpcclient.GetHostByKey(key)

		hw, serialNum, snEnc := GetHostData(host.SerialNumber) //проверяем наличие сервера в hp one view
		if serialNum == "" {
			serialNum = host.SerialNumber
		}

		host.EnclosureSerialNumber = snEnc
		bladeSystemID, err := CheckEnclosureSprRecord(host)
		_ = err

		serverId, err := UpdateHostSprRecord(*host, cl) //поиск в справочнике, при отсутствии вставка записи
		if err != nil {                                 //если ошибка то пропускаем хост
			continue
		}

		//проверяем наличие записи в справочнике конфигураций по серийному номеру
		srvhw := models.ServerHardware{}
		if serialNum != "" {
			srvhw.SerialNum = serialNum
			if err := o.Read(&srvhw, "SerialNum"); err != nil {
				//конфигурация сервера не обнаружена, необходимо вставить новую запись
				if bladeSystemID != 0 {
					srvhw.BladeSystemId = bladeSystemID
				}
				srvhw.ServerId = serverId
				if hw != nil { //пока оставил так
					srvhw.Vendor = host.Vendor
					srvhw.Model = host.Model
					srvhw.InventoryNum = serialNum
					srvhw.SerialNum = serialNum
					srvhw.CPUModel = host.CPUModel
					srvhw.CPUCount = int(host.NumCPUPkgs)
					srvhw.CorePerCPU = int(host.NumCPUCores)
					srvhw.CoreMHz = int(host.CPUMhz)
				} else {
					srvhw.Vendor = host.Vendor
					srvhw.Model = host.Model
					srvhw.InventoryNum = serialNum
					srvhw.SerialNum = serialNum
					srvhw.CPUModel = host.CPUModel
					srvhw.CPUCount = int(host.NumCPUPkgs)
					srvhw.CorePerCPU = int(host.NumCPUCores)
					srvhw.CoreMHz = int(host.CPUMhz)
				}
				srvhw.Id_User = 0
				srvhw.Time_Mfy = time.Now().Local()
				if srvhw.SerialNum != "" {
					srvhw.Id, err = o.Insert(&srvhw) //вставляем запись и запоминаем значение Id для повторного чтения
					if err != nil {
						fmt.Println("Ошибка вставки записи о конфигурации сервера", host.Name, err) //дальнейшая обработка в следующей итерации
					}
				}
			}
		}

		serverSprRecord := models.ServerSprRecords{Id: serverId}

		//читаем запись в справочнике серверов (должна быть)
		if err := o.Read(&serverSprRecord, "Id"); err == nil {
			//добавлено 27.02.2023

			CopyHostToSprSrv(*host, &serverSprRecord)      //формируем данные для обновления записи в справочнике серверов (включая, vcenter, datacenter, cluster)
			serverSprRecord.SerialNumber = srvhw.SerialNum //запоминаем серийный номер
			serverSprRecord.HWSprId = srvhw.Id

			serverSprRecord.Time_Mfy = time.Now().Local()
			_, err = o.Update(&serverSprRecord) //обновляем данные в справочнике
			_ = err

			if serialNum != "" {

				if err := o.Read(&srvhw, "Id"); err == nil && srvhw.Id > 0 { //если запись реально есть
					switch {
					case serverSprRecord.Id == srvhw.ServerId: // все хорошо, ничего не делаем
						if bladeSystemID != 0 {
							srvhw.BladeSystemId = bladeSystemID
							_, err := o.Update(&srvhw)
							if err != nil {
								fmt.Println("Ошибка обновления записи о конфигурации сервера", host.Name, err)
							}
						}
					case srvhw.ServerId == 0: //нет записи о конфигурации, нужно привязать найденную
						srvhw.ServerId = serverSprRecord.Id
						if bladeSystemID != 0 {
							srvhw.BladeSystemId = bladeSystemID
						}
						_, err := o.Update(&srvhw)
						if err != nil {
							fmt.Println("Ошибка обновления записи о конфигурации сервера", host.Name, err)
						}
					default: //есть запись, но конфигурация с таким серийным номерам привязана к другому серверу
						chksrv := models.ServerSprRecords{Id: srvhw.ServerId}
						o.Read(&chksrv, "Id")
						// TODO: truncated in screenshot — remainder of message after "\nОборудование при" not visible
						fmt.Println("!!! На хосте", host.Name, "изменилось оборудование, новый инвентарный и серийный номера", srvhw.InventoryNum, host.SerialNumber, "\nОборудование при")
						srvhw.ServerId = serverId

						if bladeSystemID != 0 {
							srvhw.BladeSystemId = bladeSystemID
						}
						_, err := o.Update(&srvhw)
						if err != nil {
							fmt.Println("Ошибка обновления записи о конфигурации сервера", host.Name, err)
						}
					}
				}
			}
		}

		//формируем запись для метрик хоста
		//hostRecord := models.HostRecord{ClusterId: clusterId, Date: time.Now().Local()} // запоминаем связку с кластером и текущее время записи
		hostRecord := models.HostRecord{ClusterId: clusterId, Date: currentTime} // запоминаем связку с кластером и текущее время записи
		importHostRecord(host, &hostRecord)
		hostRecord.ServerId = serverSprRecord.Id // запоминаем ссылку на запись в справочнике серверов

		// вставляем запись о хосте в базу данных
		hostId := int64(0)
		if hostId, err = o.Insert(&hostRecord); err != nil {
			fmt.Println("Ошибка вставки записи данных хоста: ", err)
		}
		_ = hostId
	}
}

func SaveVMRecord(cl *vsphere.Cluster, clusterId int64, currentTime time.Time) {

	if cl == nil {
		return
	}
	vml, err := grpcclient.GetVMListKey(cl.Vcenter, cl.Datacenter, cl.Name, "all")
	if err != nil || vml == nil {
		fmt.Println("Ошибка получения ключей виртуальных машин кластера: ", cl.Name, err)
		return
	}
	o := orm.NewOrm()
	var storSys []*models.StorageSystems
	qs := o.QueryTable("storage_systems")
	qs = qs.OrderBy("Id")
	num, err := qs.All(&storSys)
	mapStorSys := make(map[string]*models.StorageSystems)
	mapStorSysWWN := make(map[string]*models.StorageSystems)

	if num > 0 && err == nil {
		for i, shd := range storSys {
			//if shd.Power != 1 {
			//    continue
			//} //пропускаем выключенные СХД
			mapStorSys[shd.ShortName] = storSys[i] //запоминаем в карте по краткому имени СХД
			for mask := range strings.SplitSeq(storSys[i].WWNMask, "\n") {
				wwn := GetWWNMaskKey(mask)
				mapStorSysWWN[wwn] = storSys[i]
			}
		}
	}

	// записываем данные по виртуальным машинам кластера
	for _, vmk := range vml.KeyRecs {
		if vmk == nil {
			fmt.Println("Ошибка получения данных виртуальной машины vmk == nil")
			continue
		}
		vm, err := grpcclient.GetVM(vmk.Key)
		if err != nil {
			fmt.Println("Ошибка получения данных виртуальной машины ", vmk.Key, err)
			continue
		}
		if vm.Cluster == "не в кластере" && vm.Datacenter == "нет данных" { //пропускаем vm без заполненных данных (возникает при длительной потере связи с vcenter)
			continue
		}

		if vm.Template { //пропускаем шаблоны
			continue
		}

		//блок для записи данных для Игоря Титова
		insightVm := models.InsightVm{Vcenter: vm.Vcenter,
			Datacenter: vm.Datacenter,
			Cluster:    vm.Cluster,
			Name:       vm.Name,
			Production: cl.TypeWork}
		err = o.Read(&insightVm, "Vcenter", "Datacenter", "Cluster", "Name")
		importInsightVM(&vm, &insightVm)

		CapacityGb := float64(0)
		FreeSpaceGb := float64(0)
		PercentUsed := float64(0)

		for _, dsk := range vm.Disk {
			CapacityGb += dsk.CapacityGb
			FreeSpaceGb += dsk.FreeSpaceGb
		}
		if CapacityGb > 0 {
			PercentUsed = (CapacityGb - FreeSpaceGb) / CapacityGb
		}
		if err != nil {
			insightVm.Time = time.Now().Local()
			num, err := o.Insert(&insightVm)
			if err != nil {
				log.Println("Ошибка вставки записи в таблицу insight_vm", err, num)
			}
		} else {
			num, err := o.Update(&insightVm)
			if err != nil {
				log.Println("Ошибка обновления записи в таблице insight_vm", err, num)
			}
		}

		for _, perf := range vm.Perf {
			if perf == nil {
				log.Println("Ошибка в метриках perf == nil")
				continue
			}
			insightPerf := models.InsightVmPerf{Vcenter: vm.Vcenter,
				Datacenter:      vm.Datacenter,
				Cluster:         vm.Cluster,
				Name:            vm.Name,
				HostName:        vm.HostName,
				DiskCommittedGB: insightVm.DiskCommittedGB,
				CapacityGb:      CapacityGb,
				FreeSpaceGb:     FreeSpaceGb,
				PercentUsed:     PercentUsed}

			importInsightVMPerf(perf, &insightPerf)

			num, err := o.Insert(&insightPerf)
			if err != nil {
				log.Println("Ошибка вставки записи в таблицу insight_vm_perf", err, num)
			}
		}
		//конец блока записи для Игоря Титова
		/*
			if vm.WorkStation { //Запись данных по рабочим станциям только для Титова Игоря
				continue
			}
		*/
		if vm.Template { //пропускаем шаблоны
			continue
		}
		//rec := models.VMRecord{ClusterId: clusterId, Date: time.Now().Local()} //создаем пустую запись по виртуальной машине и запоминаем привязку к кластеру
		rec := models.VMRecord{ClusterId: clusterId, Date: currentTime} //создаем пустую запись по виртуальной машине и запоминаем привязку к кластеру
		importVMRecord(&vm, &rec)                                       //копируем данные по виртуальной машине

		if rec.Cpu_usage_maximum == 0.0 && rec.Mem_usage_maximum == 0.0 { //пропускаем записи с пустыми метриками
			continue
		}

		if rec.Cpu_usage_maximum < rec.Cpu_usage_average {
			fmt.Println("rec.Cpu_usage_maximum < rec.Cpu_usage_average", vm.Name, vm.HostName)
		}
		if rec.Mem_usage_maximum < rec.Mem_usage_average {
			fmt.Println("rec.Mem_usage_maximum < rec.Mem_usage_average", vm.Name, vm.HostName)
		}
		if rec.Cpu_usage_maximum > 100.0 {
			fmt.Println("rec.Cpu_usage_maximum > 100.0", vm.Name, vm.HostName)
		}

		if rec.Cpu_usage_average > 100.0 {
			fmt.Println("rec.Cpu_usage_average > 100.0", vm.Name, vm.HostName)
		}

		if rec.Cpu_usage_minimum > 100.0 {
			fmt.Println("rec.Cpu_usage_minimum > 100.0", vm.Name, vm.HostName)
		}

		if rec.Mem_usage_maximum > 100.0 {
			fmt.Println("rec.Mem_usage_maximum > 100.0", vm.Name, vm.HostName)
		}

		if rec.Mem_usage_average > 100.0 {
			fmt.Println("rec.Mem_usage_average > 100.0", vm.Name, vm.HostName)
		}

		if rec.Mem_usage_minimum > 100.0 {
			fmt.Println("rec.Mem_usage_minimum > 100.0", vm.Name, vm.HostName)
		}

		vmSpr := models.VMSprRecord{Vcenter: vm.Vcenter, Vm_Uuid: vm.UUID}
		srvSpr := models.ServerSprRecords{}
		VMSprId := int64(0)
		serverId := int64(0)

		//поиск в справочнике виртуальных машин
		trn, _ := o.Begin()
		//defer trn.RollbackUnlessCommit()

		err = trn.Read(&vmSpr, "Vcenter", "Vm_Uuid")
		// обработка для виртуалок с одним полным именем и разными UUID
		if err != nil {
			vmSpr.KeyName = fmt.Sprintf("%s/%s/%s/%s", vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Name)
			err = trn.Read(&vmSpr, "KeyName")
		}
		//
		if err != nil { //Новая виртуальная машина
			CopyVmToSprVm(vm, &vmSpr)      //копируем данные в структуру записи справочника VM !!! нет привязки к справочнику серверов
			vmSpr.Production = cl.TypeWork //наследуем из clusterspr
			vmSpr.ClusterId = cl.ClusterID
			VMSprId, err = trn.Insert(&vmSpr) //вставляем новую запись в справочник
			if err != nil {                   //ошибка вставки новой записи, видимо она уже есть
				fmt.Println("Ошибка сохранения в справочник виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
			CopyVmToSprSrv(vm, &srvSpr) //копируем данные в структуру о сервере
			srvSpr.ClusterId = cl.ClusterID

			srvSpr.Type = 1
			srvSpr.Stat = 1
			srvSpr.Details = "Добавлен из Vsphere, требуется правка"
			srvSpr.VMSprId = VMSprId
			srvSpr.Time_Mfy = time.Now().Local() // время модификации
			serverId, err = trn.Insert(&srvSpr)
			if err != nil {
				fmt.Println("Ошибка вставки в справочник серверов: ", err)
				trn.Rollback()
				continue
			}
			vmSpr.Id = VMSprId
			err = trn.Read(&vmSpr, "Id") //читаем запись в справочнике по id сервера
			if err != nil {
				fmt.Println("Ошибка чтения из справочника виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
			vmSpr.ServerId = serverId
			vmSpr.Production = cl.TypeWork
			vmSpr.WorkStation = vm.WorkStation
			_, err = trn.Update(&vmSpr)
			if err != nil {
				fmt.Println("Ошибка обновления справочника виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
			trn.Commit()
		} else { //обновление существующей

			CopyVmToSprVm(vm, &vmSpr)      //копируем данные в структуру записи справочника VM !!! нет привязки к справочнику серверов
			vmSpr.Production = cl.TypeWork //наследуем из clusterspr
			vmSpr.TypeWork = cl.TypeWork
			vmSpr.ClusterId = cl.ClusterID
			_, err = trn.Update(&vmSpr) //обновляем запись в справочнике
			VMSprId = vmSpr.Id
			if err != nil { //ошибка обновления записи, видимо она уже есть
				fmt.Println("Ошибка обновления в справочнике виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
			serverId = vmSpr.ServerId
			srvSpr.Id = serverId
			err = trn.Read(&srvSpr, "Id")
			if err != nil {
				fmt.Println("Ошибка чтения в справочнике серверов: ", err, "по Id: ", serverId)
				trn.Rollback()
				continue
			}
			CopyVmToSprSrv(vm, &srvSpr) //копируем данные в структуру о сервере
			srvSpr.ClusterId = cl.ClusterID
			srvSpr.Type = 1
			srvSpr.VMSprId = VMSprId
			srvSpr.Time_Mfy = time.Now().Local() // время модификации
			_, err = trn.Update(&srvSpr)         //обновление справочника серверов

			if err != nil {
				fmt.Println("Ошибка вставки в справочник серверов: ", err)
				trn.Rollback()
				continue
			}
			vmSpr.Id = VMSprId
			err = trn.Read(&vmSpr, "Id") //читаем запись в справочнике по id сервера
			if err != nil {
				fmt.Println("Ошибка чтения из справочника виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
			vmSpr.ServerId = serverId
			vmSpr.Production = cl.TypeWork
			vmSpr.WorkStation = vm.WorkStation
			vmSpr.TypeWork = cl.TypeWork
			vmSpr.ClusterId = cl.ClusterID
			_, err = trn.Update(&vmSpr)
			if err != nil {
				fmt.Println("Ошибка обновления справочника виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
			trn.Commit()
		}

		vmSpr.Id = VMSprId
		err = o.Read(&vmSpr, "Id") //читаем запись в справочнике по id виртуальной машины

		if err != nil {
			fmt.Println("Ошибка поиска виртуальной машины с Id=", VMSprId, "в справочнике виртуальных машин: ", err)
			continue
		}
		srvSpr.Id = vmSpr.ServerId
		err = o.Read(&srvSpr, "Id") //читаем запись в справочнике по serverid сервера

		srvSpr.TagName = vm.TagName
		srvSpr.TagDescription = vm.TagDescription

		srvSpr.Vcenter = vm.Vcenter
		srvSpr.Datacenter = vm.Datacenter
		srvSpr.Cluster = vm.Cluster
		if err != nil {
			fmt.Println("Ошибка поиска сервера Id=", vmSpr.ServerId, "в справочнике серверов: ", err)
			srvSpr.VMSprId = VMSprId
			err = o.Read(&srvSpr, "VMSprId") //читаем запись в справочнике по serverid сервера
			if err != nil {
				fmt.Println("Ошибка поиска сервера VMSprId=", VMSprId, "в справочнике серверов: ", err)
			} else {
				o.Read(&vmSpr, "Id")
				vmSpr.ServerId = srvSpr.Id
				srvSpr.TypeWork = cl.TypeWork //запоминаем режим работы по кластеру
				o.Update(&vmSpr)              //обновляем ссылку в справочнике виртуальных машин
			}
		} else {
			srvSpr.VMSprId = VMSprId
			srvSpr.TypeWork = cl.TypeWork //запоминаем режим работы по кластеру
			o.Update(&srvSpr)             //обновляем ссылку в справочнике серверов
		}

		rec.Date = time.Now().Local()
		rec.ServerId = serverId        //ссылка на справочник серверов
		rec.VMsprId = VMSprId          //заполняем ссылку на справочник виртуальных машин
		vmRecordId, err := o.Insert(&rec) //добавляем запись по виртуальной машине в переменной vmRecordId запоминаем номер записи по виртуальной машине по нему будет строиться связь с зависимыми данными

		if err != nil {
			fmt.Println("Ошибка записи данных по виртуальной машине: ", err)
			continue
		}

		// отладка
		/*
			for _, perf := range vm.Perf {
				vmperf := models.VMPerfRecord{}
				ImportVMPerf(&vmperf, perf)
				vmperf.VMId = vmRecordId
				_, err := o.Insert(&vmperf)
				if err != nil {
					fmt.Println("Ошибка записи метрик по виртуальной машине: ", err)
					continue
				}
			}
		*/
		//
		if len(vm.UsageDatastore) > 0 { // проверяем есть ли данные по использованию хранилищ виртуальной машиной
			//вставить запись данных записей хранилища (дополнение к записям кластеров)
			for _, st := range vm.UsageDatastore {

				//                                         ds, ok := Infra.DS_maplist.Get(st.Datastore) // ищем хранилище в карте хранилищ
				ds, ok := grpcclient.GetDSData(st.Datastore)
				datastoreId := int64(0) // инициализируем ссылку на datastore
				if ds != nil && ok {
					usd := models.VMUsageDatastore{VMSprId: VMSprId, UID: strings.ToUpper(st.UID)}
					if err := o.Read(&usd, "VMSprId", "UID"); err == nil {
						//обновляем данные
						usd.Vcenter = st.Vcenter
						usd.Datacenter = st.Datacenter
						usd.Cluster = st.Cluster
						usd.Datastore = st.Datastore
						usd.StorageName = st.StorageName
						usd.CommittedGb = st.CommittedGb
						usd.UncommittedGb = st.UncommittedGb
						usd.UnsharedGb = st.UncommittedGb
						usd.PercentAllocateStorage = st.PercentAllocateStorage
						count, err := o.Update(&usd)
						if err != nil || count < 1 {
							fmt.Println("Ошибка обновления данных по хранилищам vm", vm.Name, err)
						}
					} else {

						//вставляем новую запись
						usd.VMSprId = VMSprId
						usd.UID = strings.ToUpper(st.UID)
						usd.Vcenter = st.Vcenter
						usd.Datacenter = st.Datacenter
						usd.Cluster = st.Cluster
						usd.Datastore = st.Datastore
						usd.StorageName = st.StorageName
						usd.CommittedGb = st.CommittedGb
						usd.UncommittedGb = st.UncommittedGb
						usd.UnsharedGb = st.UncommittedGb
						usd.PercentAllocateStorage = st.PercentAllocateStorage
						count, err := o.Insert(&usd)
						if err != nil || count < 1 {
							fmt.Println("Ошибка вставки данных по хранилищам vm", vm.Name, err)
						}
					}

					dsRecord := models.Datastore{ClusterId: clusterId,
						ShortName:   "", //временно
						StorageName: ds.StorageName,
						CapacityGb:  ds.CapacityGb,
						FreeSpaceGb: ds.FreeSpaceGb,
						PercentUsed: ds.PercentUsed}

					volume := models.StorageVolumes{}
					if ds.UID != "" { //если хранилище на СХД
						volume.UID = strings.ToUpper(ds.UID)
						err = o.Read(&volume, "UID") //Ищем запись в базе по UID (Предварительно все записи с СХД перенесены в базу и обновлены)
					} else {
						volume.VirtualName = ds.StorageName //если хранилище не на СХД ищем по имени хранилища в виртуальной структуре (могут различаться с именами LUN)
						err = o.Read(&volume, "VirtualName")
					}

					volume.VirtualName = ds.StorageName
					//volume.StorageName = ds.StorageName
					volume.Vcenter = vm.Vcenter
					volume.Datacenter = vm.Datacenter
					volume.Cluster = vm.Cluster

					//volume.Server, _ = SplitNameFQDN(vm.Name)

					if err == nil { //есть такая запись в списке томов
						dsRecord.StorageSystemsId = volume.StorageSystemsId //ссылка на хранилище в справочнике
						volume.UID = strings.ToUpper(ds.UID)
						//dsRecord.ShortName = volume.ShortName //наименование хранилища из справочника
						dsRecord.ShortName = volume.VirtualName //наименование хранилища из справочника !!!!!! проверить логику
						volume.Status = 1
						volume.CapacityGb = ds.CapacityGb
						volume.FreeSpaceGb = ds.FreeSpaceGb
						volume.PercentUsed = ds.PercentUsed
						volume.Id_User = int64(0)
						volume.Time_Mfy = time.Now().Local()
						_, err = o.Update(&volume)
						if err != nil {
							fmt.Println("Ошибка обновления записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
						}
					} else { //записи нет вставляем новую
						volume.Status = 1
						volume.UID = strings.ToUpper(ds.UID)
						volume.CapacityGb = ds.CapacityGb
						volume.FreeSpaceGb = ds.FreeSpaceGb
						volume.PercentUsed = ds.PercentUsed
						volume.Id_User = int64(0)
						volume.Time_Mfy = time.Now().Local()
						_, err = o.Insert(&volume)
						if err != nil {
							fmt.Println("Ошибка вставки записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
						}
					}
					qs := o.QueryTable("datastore")
					qs = qs.OrderBy("cluster_id")
					qs = qs.Filter("cluster_id__eq", clusterId)
					qs = qs.Filter("storage_name__eq", ds.StorageName)
					err = qs.One(&dsRecord)
					if err == nil {
						datastoreId = dsRecord.Id
					} else { //нужно вставить запись по хранилищу
						datastoreId, err = o.Insert(&dsRecord)
						if err != nil {
							fmt.Println("Ошибка вставки записи харнилища", err)
						}
					}
				} else {
					// Хранилище не найдено вставить обработку записи RDM дисков
					fmt.Println("Хранилище", st.Datastore, "не найдено") // добавить создание записи о хранилище
					dsRecord := models.Datastore{ClusterId: clusterId,
						ShortName:   strings.ToUpper(st.UID), //временно
						StorageName: st.StorageName,
						CapacityGb:  st.CommittedGb,
						FreeSpaceGb: 0,
						PercentUsed: 0}

					volume := models.StorageVolumes{}
					if st.UID != "" { //если хранилище на СХД
						strings.ToUpper(st.UID)
						err = o.Read(&volume, "UID") //Ищем запись в базе по UID (Предварительно все записи с СХД перенесены в Базу и обновлены)
					} else {
						volume.VirtualName = st.StorageName //если хранилище не на СХД ищем по имени хранилища в виртуальной структуре (могут различаться с именами LUN)
						err = o.Read(&volume, "VirtualName")
					}
					volume.VirtualName = st.StorageName
					//volume.StorageName = st.StorageName
					volume.Vcenter = vm.Vcenter
					volume.Datacenter = vm.Datacenter
					volume.Cluster = vm.Cluster

					volume.Server, _ = SplitNameFQDN(vm.Name)

					if err == nil { //есть такая запись в списке томов
						dsRecord.StorageSystemsId = volume.StorageSystemsId //ссылка на хранилище в справочнике
						volume.UID = strings.ToUpper(st.UID)
						//dsRecord.ShortName = volume.ShortName //наименование хранилища из справочника перетирается UID
						dsRecord.ShortName = volume.VirtualName //наименование хранилища из справочника перетирается UID
						volume.Status = 1
						volume.CapacityGb = st.CommittedGb
						volume.Id_User = int64(0)
						volume.Time_Mfy = time.Now().Local()
						_, err = o.Update(&volume)
						if err != nil {
							fmt.Println("Ошибка обновления записи тома:", volume.StorageName, volume.VirtualName, st.UID)
						}
					} else { //записи нет вставляем новую
						volume.Status = 1
						volume.UID = strings.ToUpper(st.UID)
						volume.CapacityGb = st.CommittedGb
						volume.Id_User = int64(0)
						volume.Time_Mfy = time.Now().Local()
						_, err = o.Insert(&volume)
						if err != nil {
							fmt.Println("Ошибка вставки записи тома:", volume.StorageName, volume.VirtualName, st.UID)
						}
					}
					qs := o.QueryTable("datastore")
					qs = qs.OrderBy("cluster_id")
					qs = qs.Filter("cluster_id__eq", clusterId)
					qs = qs.Filter("storage_name__eq", st.StorageName)
					err = qs.One(&dsRecord)
					if err == nil {
						datastoreId = dsRecord.Id
					} else { //нужно вставить запись по хранилищу
						datastoreId, err = o.Insert(&dsRecord)
						if err != nil {
							fmt.Println("Ошибка вставки записи харнилища", err)
						}
					}
				}

				usageDatastore := models.VMRecordUsageOnDatastore{VMRecordId: vmRecordId,
					DatastoreId:            datastoreId, //добавить ссылку на запись хранилища кластера
					CommittedGb:            st.CommittedGb,
					UncommittedGb:          st.UncommittedGb,
					UnsharedGb:             st.UnsharedGb,
					PercentAllocateStorage: st.PercentAllocateStorage}
				_, err := o.Insert(&usageDatastore)
				if err != nil {
					fmt.Println("Ошибка вставки записи по использованию хранилища", err)
				}
			}
		}

		/* отключена запись гостевых дисков
		if len(vm.Disk) > 0 {
			for _, disk := range vm.Disk {
				// TODO: truncated in screenshot — field name cut off at disk.Percent
				discRecord := models.VMRecordGuestDiskInfo{VMRecordId: vmRecordId, DiskPath: disk.DiskPath, CapacityGb: disk.CapacityGb, FreeSpaceGb: disk.FreeSpaceGb, PercentUsed: disk.PercentUsed}
				_, err := o.Insert(&discRecord)
				if err != nil {
					fmt.Println("Ошибка вставки записей по дискам гостевой системы", err)

					return
				}

				num, err = o.InsertMulti(len(disk_data), disk_data)
				if err != nil {
					fmt.Println(vm.Name, "Ошибка записи данных по дискам: записано ", num, "из", len(disk_data), err)
					return
				}
			}
		}
		*/
	}
}
