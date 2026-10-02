package cmdb

import (
	"context"
	"fmt"
	"loadmon/grpcclient"
	vsphere "loadmon/vspheremetrics"

	//"loadmon/itsystem"
	models "loadmon/models"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

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

// SaveHostRecord сохраняет метрики производительности всех хостов указанного кластера
// vSphere, получая аппаратную конфигурацию и данные корзины при их наличии.
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

		// Stage 1 доработки хранения сетевых интерфейсов (см. CHANGELOG.md,
		// запись про ServerNetworkInterface): дополнительно, не заменяя
		// существующую сборку ";"-строк host.IPAddress/host.MacAddress
		// (заполняется в vspheremetrics/importdata.go как и раньше),
		// сохраняем структурированные данные из host.Interfaces
		// (hi.Config.Network.Vnic) в новую таблицу. MacAddress здесь есть
		// (в отличие от Zabbix), SubnetMask - тоже (источник "vmware_host_vnic").
		if serverId > 0 && len(host.Interfaces) > 0 {
			var ifaceRows []*models.ServerNetworkInterface
			for _, iface := range host.Interfaces {
				if iface.IPAddress == "" {
					continue
				}
				ver := int8(4)
				if strings.Contains(iface.IPAddress, ":") {
					ver = 6
				}
				ifaceRows = append(ifaceRows, &models.ServerNetworkInterface{
					ServerSprId: serverId,
					Source:      "vmware_host_vnic",
					MacAddress:  iface.MacAddress,
					IPAddress:   iface.IPAddress,
					IPVersion:   ver,
					SubnetMask:  iface.SubnetMask,
				})
			}
			if err := models.ReplaceServerNetworkInterfaces(context.Background(), o, serverId, "vmware_host_vnic", ifaceRows); err != nil {
				fmt.Println("Ошибка записи models.ServerNetworkInterface (vmware_host_vnic)", host.Name, err)
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

// SaveHostRecordOld — устаревшая версия SaveHostRecord для сохранения метрик
// производительности хостов кластера vSphere.
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
				srvhw.SourceUpdateType = models.SourceUpdateVMware // поля выше берутся из host.* (vSphere), не из hw.* (HP OneView)
				// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
				// (10002 - VMware/vSphere) для новой записи ServerHardware.
				srvhw.TimeCreate = time.Now().Local()
				srvhw.CreatorUser = 10002
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

// SaveVMRecord сохраняет метрики производительности всех виртуальных машин указанного
// кластера vSphere.
