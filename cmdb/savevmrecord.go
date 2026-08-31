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
			srvSpr.SourceUpdateType = models.SourceUpdateVMware
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
			srvSpr.SourceUpdateType = models.SourceUpdateVMware
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
		rec.ServerId = serverId           //ссылка на справочник серверов
		rec.VMSprId = VMSprId             //заполняем ссылку на справочник виртуальных машин
		vmRecordId, err := o.Insert(&rec) //добавляем запись по виртуальной машине в переменной vmRecordId запоминаем номер записи по виртуальной машине по нему будет строиться связь с зависимыми данными

		if err != nil {
			fmt.Println("Ошибка записи данных по виртуальной машине: ", err)
			continue
		}

		// Детальные ("сырые", по каждому замеру vSphere) метрики загрузки ВМ -
		// таблица vm_perf (models.VMPerfRecord), в отличие от только что
		// вставленной агрегированной суточной записи rec (vm_record). Раньше
		// эта вставка была отключена (закомментирована как отладочная) -
		// таблица vm_perf не заполнялась вовсе. Включено по требованию:
		// хранить детальные метрики только для серверов, рабочие станции
		// Windows (vm.WorkStation - поле уже по своему определению означает
		// именно "рабочая станция Windows", см. комментарий к полю в
		// vspheremetrics/datatypes.go) в детальную запись не включать - для
		// них остаётся только суточная агрегированная запись rec выше.
		// Хранение ограничено 3 сутками - см. ClearOldMetric
		// (cmdb/savemetric_orchestration.go).
		if !vm.WorkStation {
			for _, perf := range vm.Perf {
				vmperf := models.VMPerfRecord{}
				ImportVMPerf(&vmperf, perf)
				vmperf.VMId = vmRecordId
				if _, err := o.Insert(&vmperf); err != nil {
					fmt.Println("Ошибка записи детальных метрик по виртуальной машине: ", vm.Name, err)
					continue
				}
			}
		}
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
					qs = qs.OrderBy("ClusterId")
					qs = qs.Filter("ClusterId__eq", clusterId)
					qs = qs.Filter("StorageName__eq", ds.StorageName)
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
					qs = qs.OrderBy("ClusterId")
					qs = qs.Filter("ClusterId__eq", clusterId)
					qs = qs.Filter("StorageName__eq", st.StorageName)
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
