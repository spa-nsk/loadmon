package cmdb

import (
	"errors"
	"fmt"
	"loadmon/grpcclient"

	//"loadmon/tools"
	vsphere "loadmon/vspheremetrics"

	//"loadmon/itsystem"
	models "loadmon/models"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

var (
	nilPointerError string = "Ожидался указатель на переменную, отличный от nil"

// timeOutError string = "Превышено время ожидания"
)

// UpdateHostSprRecord создаёт либо обновляет в единой транзакции связанные записи
// справочника серверов (ServerSprRecords) и справочника оборудования (ServerHardware)
// для хоста ESXi, дополняя их данными из HP OneView при наличии соответствия по
// серийному номеру. Возвращает Id записи ServerSprRecords.
func UpdateHostSprRecord(hs vsphere.HostInfo, cl *vsphere.Cluster) (int64, error) {
	if cl == nil {
		return 0, errors.New(nilPointerError)
	}
	srvspr := models.ServerSprRecords{}
	hwspr := models.ServerHardware{}

	ov := oneview.GetGlobalOVInfrastructure()
	o := orm.NewOrm()

	trn, err := o.Begin()
	if err != nil {
		return 0, err
	}
	//defer trn.RollbackUnlessCommit()

	hw, sn, snEnc := GetHostData(hs.SerialNumber) //проверяем серийный номер в hp one view
	_ = snEnc
	_ = sn

	foundHWSpr := false
	if !foundHWSpr {
		if hw != nil {
			hwspr.SerialNum = sn
		} else {
			hwspr.SerialNum = hs.SerialNumber
		}
		err := trn.Read(&hwspr, "SerialNum")
		if err == nil {
			foundHWSpr = true
		}
	}

	if !foundHWSpr {
		hwspr.SerialNum = hs.SerialNumber // пытаемся найти по серийному номеру из host (возможно виртуальный серийный номер)
		err := trn.Read(&hwspr, "SerialNum")
		if err == nil {
			foundHWSpr = true
		}
	}

	foundSrvSpr := false
	if foundHWSpr { // если есть запись в ServerHardware и нет данных по ServerSprRecords
		srvspr.HWSprId = hwspr.Id
		err := models.ServerSprQS(trn).Filter("HWSprId", srvspr.HWSprId).Limit(1).One(&srvspr) // без записей Stat=666
		if err == nil {
			foundSrvSpr = true
			hwspr.ServerId = srvspr.Id
		}
	}

	hwId := hwspr.Id
	srvId := srvspr.Id

	CopyHostToSprSrv(hs, &srvspr)
	srvspr.ClusterId = cl.ClusterID
	srvspr.SourceUpdateType = models.SourceUpdateVMware
	CopyHostToSprHW(hs, &hwspr)
	hwspr.SourceUpdateType = models.SourceUpdateVMware

	if hw != nil {
		ImportHWSprRecord(trn, &hwspr, hw, ov)
		srvspr.SerialNumber = sn
		hwspr.SerialNum = sn
		hwspr.SourceUpdateType = models.SourceUpdateOneView // данные оборудования уточнены из HP OneView - более авторитетный источник, чем голый ESXi
	}

	if err == nil {
		var (
			errS, errH error
		)

		if foundHWSpr {
			_, errH = trn.Update(&hwspr)
			hwId = hwspr.Id
		} else {
			// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
			// (10002 - VMware/vSphere) для новой записи ServerHardware.
			hwspr.TimeCreate = time.Now().Local()
			hwspr.CreatorUser = 10002
			hwId, errH = trn.Insert(&hwspr)
		}

		srvspr.HWSprId = hwId

		if foundSrvSpr {
			srvspr.TypeWork = cl.TypeWork
			srvspr.Stat = 1
			_, errS = trn.Update(&srvspr)
			srvId = srvspr.Id
		} else {
			srvspr.Stat = 1
			srvspr.TypeWork = cl.TypeWork
			// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
			// (10002 - VMware/vSphere) для новой записи ServerSprRecords.
			srvspr.TimeCreate = time.Now().Local()
			srvspr.CreatorUser = 10002
			srvId, errS = trn.Insert(&srvspr)
		}
		hwspr.Id = hwId
		err2 := trn.Read(&hwspr, "Id")
		if err2 == nil {
			hwspr.ServerId = srvId
			_, err2 = trn.Update(&hwspr)
			if err2 != nil {
				trn.Rollback()
				return 0, err2
			}
			if errH == nil && errS == nil {
				trn.Commit()
				return srvId, nil
			} else {
				trn.Rollback()
				if errH != nil {
					return srvId, errH
				}
				if errS != nil {
					return srvId, errS
				}
			}
		} else {
			trn.Rollback()
			return 0, err2
		}
	}
	trn.Commit()
	return 0, err
}

// UpdateHostSprRecordOld — устаревшая версия UpdateHostSprRecord, обновляющая записи
// справочников ServerSprRecords и ServerHardware без учёта данных кластера.
func UpdateHostSprRecordOld(hs vsphere.HostInfo) (int64, error) {
	srvspr := models.ServerSprRecords{}

	ov := oneview.GetGlobalOVInfrastructure()

	o := orm.NewOrm()

	foundSrvSpr := false
	if !foundSrvSpr {
		srvspr.SerialNumber = hs.SerialNumber
		err := models.ServerSprQS(o).Filter("SerialNumber", srvspr.SerialNumber).Limit(1).One(&srvspr)
		if err == nil {
			foundSrvSpr = true
		}
	}
	if !foundSrvSpr {
		srvspr.Name = hs.Name
		err := models.ServerSprQS(o).Filter("Name", srvspr.Name).Limit(1).One(&srvspr) //пытамся найти запись в справочнике  по Name
		if err == nil {
			foundSrvSpr = true
		}
	}
	if !foundSrvSpr {
		srvspr.NameFQDN = hs.Name
		err := models.ServerSprQS(o).Filter("NameFQDN", srvspr.NameFQDN).Limit(1).One(&srvspr) //пытамся найти запись в справочнике по NameFQDN
		if err == nil {
			foundSrvSpr = true
		}
	}

	hw, sn, snEnc := GetHostData(hs.SerialNumber) //проверяем серийный номер в hp one view
	_ = snEnc

	CopyHostToSprSrv(hs, &srvspr)

	if hw != nil {
		srvspr.SerialNumber = sn
	} else {
		srvspr.SerialNumber = hs.SerialNumber
	}

	if !foundSrvSpr { //не нашлось в справочнике - нужно создать
		srvspr.Stat = 1
		if hw != nil {
			if hw.Identity.MpHostName == hs.Name {
				srvspr.Details = hw.Identity.SerialNumber + " Добавлен из Vsphere + hp One View, пребуется правка"
			}
		} else {
			srvspr.Details = hs.SerialNumber + " Добавлен из Vsphere, требуется правка "
		}
	} else {
		srvspr.Stat = 1
		if hw != nil {
			if hw.Identity.MpHostName == hs.Name {
				srvspr.SerialNumber = sn
			}
		} else {
			srvspr.SerialNumber = hs.SerialNumber
		}
	}

	if srvspr.SerialNumber != "" {
		hw2 := models.ServerHardware{SerialNum: srvspr.SerialNumber}
		err := o.Read(&hw2, "SerialNum")
		if err == nil {
			srvspr.HWSprId = hw2.Id
			if hw != nil {
				ImportHWSprRecord(o, &hw2, hw, ov)
				hw2.SourceUpdateType = models.SourceUpdateOneView
				num, err := o.Update(&hw2)
				if err != nil || num < 1 {
					fmt.Println("Ошибка обновления записи конфигурации сервера:", srvspr.Name, srvspr.NameFQDN, err)
				}
			}
		} else {
			if hw != nil {
				ImportHWSprRecord(o, &hw2, hw, ov)
				hw2.SourceUpdateType = models.SourceUpdateOneView
				if srvspr.Active != 0 {
					hw2.Active = srvspr.Active
				}
				hw2.Time_Mfy = time.Now().Local()
				// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
				// (10002 - VMware/vSphere) для новой записи ServerHardware.
				hw2.TimeCreate = time.Now().Local()
				hw2.CreatorUser = 10002
				HWSprId, err := o.Insert(&hw2)
				if err != nil {
					fmt.Println("Ошибка вставки записи конфигурации сервера:", srvspr.Name, srvspr.NameFQDN, err)
				} else {
					srvspr.HWSprId = HWSprId
				}
			}
		}
	}

	srvspr.Time_Mfy = time.Now().Local()
	srvspr.SourceUpdateType = models.SourceUpdateVMware

	if !foundSrvSpr {
		// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
		// (10002 - VMware/vSphere) для новой записи ServerSprRecords.
		srvspr.TimeCreate = time.Now().Local()
		srvspr.CreatorUser = 10002
		if Id, err := o.Insert(&srvspr); err != nil {
			fmt.Println("Ошибка вставки в справочник серверов", err)
			return 0, err
		} else {
			return Id, nil
		}
	} else {
		if _, err := o.Update(&srvspr); err != nil {
			fmt.Println("Ошибка обновления справочника серверов", err)
			return srvspr.Id, err
		} else {
			return srvspr.Id, nil
		}
	}
}

// SaveVMwareCMDBData выполняет полный цикл сохранения в CMDB данных, полученных из vSphere:
// перебирает все кластеры (кроме одиночных хостов) и для каждого вызывает SaveClusterCMDBData.
func SaveVMwareCMDBData() {
	fmtDate := "02.01.2006 15:04:05.000"
	currentTime := time.Now().Local() //запоминаем текущее время
	fmt.Println(currentTime.Format(fmtDate), "Запись данных кластеров, хостов и виртуальных машин в базу")

	clusterSpr, err := grpcclient.LoadClusterSpr("all", "all", "all")
	if err != nil || clusterSpr == nil {
		return
	}

	clList := grpcclient.LoadClusterList("all", "all")
	count := len(clList)

	for i, clShort := range clList {
		cl := grpcclient.GetClusterMedium(clShort.Vcenter, clShort.Datacenter, clShort.Name)

		notData := "нет данных"

		if cl.Datacenter == notData || cl.Name == notData || cl.HostFlag { //если не заполнены данные по датацентру, кластеру или это одиночный хост, то пропускаем запись 15.10.2025
			log.Println("Пропускаем одиночный хост или если данные не заполнены: ", i+1, "/", count, cl.Name)
			continue
		}

		log.Println("запись кластера: ", i+1, "/", count, cl.Name)
		SaveClusterCMDBData(cl, clusterSpr, currentTime)
	}

	fmt.Println(time.Now().Local().Format(fmtDate), "Запись данных завершена.")
}

// сохраняем данные кластера
func SaveClusterCMDBData(cl *vsphere.Cluster, clusterSpr map[string]*models.ClusterSprRecords, currentTime time.Time) {
	// поиск информации о кластере в справочнике кластеров
	clId := int64(0)
	//produtcion := int(0)
	cls, ok := clusterSpr[fmt.Sprintf("%s/%s/%s", cl.Vcenter, cl.Datacenter, cl.Name)]
	if ok { // кластер найден
		clId = cls.Id // запоминаем id код из справочника
		//produtcion = cls.Production
	} else {
		//rec := models.ClusterSprRecords{Vcenter: cl.Vcenter, Datacenter: cl.Datacenter, Name: cl.Name, Time_Mfy: time.Now().Local()}
		rec := models.ClusterSprRecords{Vcenter: cl.Vcenter, Datacenter: cl.Datacenter, Name: cl.Name, Time_Mfy: currentTime}
		status := 0
		clId, status = grpcclient.AddClusterSprRecord(&rec)
		if status != 1 { //ошибка вставки записи кластера в справочник
			return //пропускаем этот кластер
		}
	}
	clusterId, err := SaveClusterCMDBRecord(cl, clId, currentTime)
	if err != nil {
		return
	}
	SaveHostCMDBData(cl, clusterId, currentTime)
	//SaveVMCMDBData(cl, clusterId, currentTime, produtcion)
	SaveVMCMDBData(cl, clusterId, currentTime, cl.TypeWork)
}

// SaveClusterCMDBRecord предназначена для сохранения записи кластера в CMDB.
// В текущей реализации не выполняет действий (заглушка).
func SaveClusterCMDBRecord(cl *vsphere.Cluster, clId int64, currentTime time.Time) (int64, error) {
	return 0, nil
}

// GetHostData ищет данные оборудования хоста в HP OneView по серийному номеру (в том числе
// виртуальному) и возвращает найденную запись, фактический серийный номер и серийный номер
// корзины (enclosure), если применимо.
func GetHostData(serialNum string) (*oneview.Server, string, string) {
	var (
		hw *oneview.Server
		ok bool
	)

	sn := ""
	snEnc := ""

	hw, ok = grpcclient.GetServerHardwareByVirtualSN(serialNum)

	if !ok {
		hw, ok = grpcclient.GetServerHardwareBySN(serialNum)
	}
	if ok {
		sn = hw.Identity.SerialNumber
		s := strings.TrimSpace(hw.EnclosureInfo.SerialNumber) //если корзина реально есть
		if s != "" {
			snEnc = s
		}
	}
	return hw, sn, snEnc
}

// SaveHostCMDBData сохраняет в CMDB сведения обо всех хостах указанного кластера vSphere,
// дополняя их данными аппаратной конфигурации из HP OneView.
func SaveHostCMDBData(cl *vsphere.Cluster, clusterID int64, currentTime time.Time) {
	o := orm.NewOrm()

	hsKeys := grpcclient.LoadHostListKey(cl.Vcenter, cl.Datacenter, cl.Name, "all")
	for _, key := range hsKeys {
		//получить аппаратную конфигурацию хоста и серийный номер корзины, при наличие
		//log.Println("запись метрик хоста: ", cl.Name, host.Name)
		host := grpcclient.GetHostByKey(key)

		hw, serialNum, snEnc := GetHostData(host.SerialNumber)
		_ = hw

		if serialNum == "" {
			serialNum = host.SerialNumber
		}

		//cl.Hosts[i].EnclosureSerialNumber = snEnc
		host.EnclosureSerialNumber = snEnc
		//bladeSystemID, err := CheckEnclosureSprRecord(cl.Hosts[i])
		bladeSystemID, err := CheckEnclosureSprRecord(host)
		if err != nil {
			fmt.Println("Ошибка получения данных блэйд корзины", err)
		}

		//прочитать данные по справочнику серверов и внести найденный ServerId в hostRecord
		//var records []*models.ServerSprRecords

		serverId, err := UpdateHostSprRecord(*host, cl) //поиск в справочнике, при отсутствии вставка записи
		if err != nil {
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
				srvhw.SourceUpdateType = models.SourceUpdateVMware // поля выше берутся из host.* (vSphere), не из hw.* (HP OneView)
				// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
				// (10002 - VMware/vSphere) для новой записи ServerHardware.
				srvhw.TimeCreate = time.Now().Local()
				srvhw.CreatorUser = 10002
				if srvhw.SerialNum != "" {
					srvhw.Id, err = o.Insert(&srvhw) //вставляем запись и запоминаем значение Id для повторного чтения
					if err != nil {
						fmt.Println("Ошибка вставки записи о конфигурации сервера", host.Name, err) //дальнейшая обработка в следующем чтении
					}
				}
			}
		}

		serverSprRecord := models.ServerSprRecords{Id: serverId}

		//читаем запись в справочнике серверов (должна быть)
		if err := o.Read(&serverSprRecord, "Id"); err == nil && !models.IsServerSprDestroyed(&serverSprRecord) {
			//добавлено 27.02.2023

			CopyHostToSprSrv(*host, &serverSprRecord)      //формируем данные для обновления записи в справочнике серверов (включая, vcenter, datacenter, cluster)
			serverSprRecord.SerialNumber = srvhw.SerialNum //запоминаем серийный номер
			serverSprRecord.HWSprId = srvhw.Id

			serverSprRecord.Time_Mfy = time.Now().Local()
			serverSprRecord.SourceUpdateType = models.SourceUpdateVMware
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
						srvhw.ServerId = serverId
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
						fmt.Println("!!! На хосте", host.Name, "изменилось оборудование, новый инвентарный и серийный номера", srvhw.InventoryNum, host.SerialNumber, "\nОбор")
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
	}
}

// SaveVMCMDBData сохраняет в CMDB сведения обо всех виртуальных машинах указанного
// кластера vSphere.
func SaveVMCMDBData(cl *vsphere.Cluster, clusterId int64, currentTime time.Time, production int) {

	vml, err := grpcclient.GetVMListKey(cl.Vcenter, cl.Datacenter, cl.Name, "all")
	if err != nil {
		fmt.Println("Ошибка получения ключей виртуальных машин кластера: ", cl.Name, err)
		return
	}
	o := orm.NewOrm()

	// записываем данные по виртуальным машинам кластера
	for _, vmk := range vml.KeyRecs {
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
		/*
			//блок для записи данных для Игоря Титова
			insightVm := models.InsightVm{Vcenter: vm.Vcenter,
				Datacenter: vm.Datacenter,
				Cluster:    vm.Cluster,
				Name:       vm.Vm_Name,
				Production: production}
			err = o.Read(&insightVm, "Vcenter", "Datacenter", "Cluster", "Name")
			tools.ImportInsightVM(&vm, &insightVm)

			CapacityGb := float64(0)
			FreeSpaceGb := float64(0)
			PercentUsed := float64(0)

			for _, dsk := range vm.Vm_Disk {
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
				insightPerf := models.InsightVmPerf{Vcenter: vm.Vcenter,
					Datacenter:      vm.Datacenter,
					Cluster:         vm.Cluster,
					Name:            vm.Vm_Name,
					HostName:        vm.Vm_HostName,
					DiskCommittedGB: insightVm.DiskCommittedGB,
					CapacityGb:      CapacityGb,
					FreeSpaceGb:     FreeSpaceGb,
					PercentUsed:     PercentUsed}

				tools.ImportInsightVMPerf(perf, &insightPerf)

				num, err := o.Insert(&insightPerf)
				if err != nil {
					log.Println("Ошибка вставки записи в таблицу insight_vm_perf", err, num)
				}
			}
		*/
		//конец блока записи для Игоря Титова

		if vm.WorkStation { //Запись данных по рабочим станциям только для Титова Игоря
			continue
		}

		//поиск в справочнике виртуальных машин

		VMSprId := int64(0)
		serverId := int64(0)
		//vmspr := models.VMSprRecord{}
		srvspr := models.ServerSprRecords{}

		vmspr := models.VMSprRecord{
			Vcenter:    vm.Vcenter,
			Datacenter: vm.Datacenter,
			Cluster:    vm.Cluster,
			Vm_Name:    vm.Name}
		trn, _ := o.Begin()
		err = trn.Read(&vmspr, "Vcenter", "Datacenter", "Cluster", "Vm_Name")
		CopyVmToSprVm(vm, &vmspr)     //копируем данные в структуру записи справочника VM !!! нет привязки к справочнику серверов
		vmspr.Production = production //наследуем из clusterspr

		if err != nil {
			VMSprId, err = trn.Insert(&vmspr) //вставляем новую запись в справочник
			if err != nil {
				//ошибка вставки новой записи, видимо она уже есть
				fmt.Println("Ошибка сохранения в справочник виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
		} else {
			vmspr.ServerId = 0
			num, err := trn.Update(&vmspr) //вставляем новую запись в справочник
			if err != nil || num < 1 {     //ошибка вставки новой записи, видимо она уже есть
				fmt.Println("Ошибка сохранения в справочник виртуальных машин: ", err)
				trn.Rollback()
				continue
			}
			VMSprId = vmspr.Id
			//serverId = vmspr.ServerId
		}
		srvspr.Vcenter = vm.Vcenter
		srvspr.Datacenter = vm.Datacenter
		srvspr.Cluster = vm.Cluster
		srvspr.Name = vm.Name
		err = models.ServerSprQS(trn).Filter("Vcenter", srvspr.Vcenter).Filter("Datacenter", srvspr.Datacenter).Filter("Cluster", srvspr.Cluster).Filter("Name", srvspr.Name).Limit(1).One(&srvspr) // без записей Stat=666

		CopyVmToSprSrv(vm, &srvspr)
		srvspr.VMSprId = VMSprId
		srvspr.Time_Mfy = time.Now().Local() // время модификации
		srvspr.SourceUpdateType = models.SourceUpdateVMware
		if err != nil {
			srvspr.Type = 1
			srvspr.Stat = 1
			srvspr.Details = " Добавлен из Vsphere, пребуется правка"
			// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
			// (10002 - VMware/vSphere) для новой записи ServerSprRecords.
			srvspr.TimeCreate = time.Now().Local()
			srvspr.CreatorUser = 10002
			serverId, err = trn.Insert(&srvspr)
			if err != nil {
				fmt.Println("Ошибка вставки в справочник серверов: ", err)
				trn.Rollback()
				continue
			}
		} else {
			_, err = trn.Update(&srvspr)
			if err != nil {
				fmt.Println("Ошибка обновления в справочнике серверов: ", err)
				trn.Rollback()
				continue
			}
			serverId = srvspr.Id
		}

		vmspr.Id = VMSprId
		err = trn.Read(&vmspr, "Id") //читаем запись в справочнике по id сервера
		if err != nil {
			fmt.Println("Ошибка чтения из справочника виртуальных машин: ", err)
			trn.Rollback()
			continue

		}
		vmspr.ServerId = serverId
		vmspr.Production = production
		vmspr.WorkStation = vm.WorkStation
		num, err := trn.Update(&vmspr)
		if err != nil || num < 1 {
			fmt.Println("Ошибка обновления справочника виртуальных машин: ", err)
			trn.Rollback()
			continue
		}
		trn.Commit()

		if len(vm.UsageDatastore) > 0 { // проверяем есть ли данные по использованию хранилищ виртуальной машиной
			//вставить запись данных записей хранилища  (дополнение к записям кластеров)
			for _, st := range vm.UsageDatastore {
				//                              ds, ok := Infra.DS_maplist.Get(st.Datastore) // ищем хранилище в карте хранилищ
				ds, ok := grpcclient.GetDSData(st.Datastore)
				//datastoreId := int64(0) // инициализируем ссылку на datastore
				if ds != nil && ok {
					usd := models.VMUsageDatastore{VMSprId: VMSprId, UID: st.UID}
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
						usd.UID = st.UID
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
					volume := models.StorageVolumes{}
					if ds.UID != "" { //если хранилище на СХД
						volume.UID = ds.UID
						err = o.Read(&volume, "UID") //Ищем запись в базе по UID (Предварительно все записи с СХД перенесены в базу и обновлены)
					} else {
						volume.VirtualName = ds.StorageName //если хранилище не на СХД ищем по имени хранилища в виртуальной структуре (могут различаться с именами LUN)
						err = o.Read(&volume, "VirtualName")
					}

					volume.VirtualName = ds.StorageName
					volume.ShortName = ds.StorageName
					volume.Vcenter = vm.Vcenter
					volume.Datacenter = vm.Datacenter
					volume.Cluster = vm.Cluster

					if err == nil { //есть такая запись в списке томов
						volume.UID = ds.UID
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
						volume.UID = ds.UID
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

				} else {
					// Хранилище не найдено вставить обработку записи RDM дисков
					fmt.Println("Хранилище", st.Datastore, "не найдено") // добавить создание записи о хранилище

					volume := models.StorageVolumes{}
					if st.UID != "" { //если хранилище на СХД
						volume.UID = st.UID
						err = o.Read(&volume, "UID") //Ищем запись в базе по UID (Предварительно все записи с СХД перенесены в базу и обновлены)
					} else {
						volume.VirtualName = st.StorageName //если хранилище не на СХД ищем по имени хранилища в виртуальной структуре (могут различаться с именами LUN)
						err = o.Read(&volume, "VirtualName")
					}
					volume.VirtualName = st.StorageName
					volume.ShortName = st.StorageName
					volume.Vcenter = vm.Vcenter
					volume.Datacenter = vm.Datacenter
					volume.Cluster = vm.Cluster

					if err == nil { //есть такая запись в списке томов
						volume.UID = st.UID
						volume.Status = 1
						volume.CapacityGb = st.CommittedGb
						volume.Id_User = int64(0)
						volume.Time_Mfy = time.Now().Local()
						_, err = o.Update(&volume)
						if err != nil {
							fmt.Println("Ошибка обновления записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
						}
					} else { //записи нет вставляем новую
						volume.Status = 1
						volume.UID = st.UID
						volume.CapacityGb = st.CommittedGb
						volume.Id_User = int64(0)
						volume.Time_Mfy = time.Now().Local()
						_, err = o.Insert(&volume)
						if err != nil {
							fmt.Println("Ошибка вставки записи тома:", volume.StorageName, volume.VirtualName, ds.UID)
						}
					}
				}
			}
		}

		// запись гостевых дисков
		if len(vm.Disk) > 0 {
			disk_data := make([]models.VMGuestDiskInfo, 0)
			for _, disk := range vm.Disk {
				dsk := models.VMGuestDiskInfo{VMId: VMSprId, DiskPath: disk.DiskPath, CapacityGb: disk.CapacityGb, FreeSpaceGb: disk.FreeSpaceGb, PercentUsed: disk.PercentUsed}
				disk_data = append(disk_data, dsk)
			}

			num, err := o.Delete(&models.VMGuestDiskInfo{VMId: VMSprId}, "VMId")
			if err != nil {
				fmt.Println(vm.Name, "Ошибка очистки данных по дискам: ", num, err)
				return
			}

			num, err = o.InsertMulti(len(disk_data), disk_data)
			if err != nil {
				fmt.Println(vm.Name, "Ошибка записи данных по дискам: записано ", num, "из", len(disk_data), err)
				return
			}
		}
	}
}

//
