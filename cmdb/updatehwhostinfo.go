package cmdb

import (
	"fmt"
	models "loadmon/models"
	"loadmon/oneview"
	_ "loadmon/routers"
	vsphere "loadmon/vspheremetrics"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func insertNewHostSprRecord(hs vsphere.HostInfo) (int64, error) {
	srvspr := models.ServerSprRecords{}
	serverName := strings.TrimSpace(hs.Name)

	ov := oneview.GetGlobalOVInfrastructure()

	o := orm.NewOrm()
	//o.Using("default")

	//shortName := strings.Split(serverName, ".")[0]
	srvspr.Name = serverName
	srvspr.NameFQDN = serverName

	//
	// srvspr.Type = 2
	srvspr.CPUCount = float64(hs.NumCPUPkgs)
	srvspr.CPULogical = float64(hs.NumCPUCores)
	srvspr.MemSize = float64(hs.MemorySizeGB)
	srvspr.Stat = 1

	hw, ok := ov.ServerByVirtualSN[hs.SerialNumber]

	if ok {
		if hw.Base.Hostname == hs.Name {
			srvspr.Details = hw.Base.SerialNumber.String() + " Host Добавлен из Vsphere, требуется правка"
		}
	} else {
		srvspr.Details = "Virtual SN: " + hs.SerialNumber + " Host Добавлен из Vsphere, требуется правка"
	}

	srvspr.Time_Mfy = time.Now().Local()

	srvspr.OS = hs.ProductFullName //копируем точную версию esxi

	srvspr.Vcenter = hs.Vcenter
	srvspr.Datacenter = hs.Datacenter
	srvspr.Cluster = hs.Cluster

	if srvspr.SerialNumber != "" {
		srvspr.SerialNumber = string(hw.Base.SerialNumber)

		hw2 := models.ServerHardware{SerialNum: string(hw.Base.SerialNumber)}
		err := o.Read(&hw2, "SerialNum")
		if err == nil {
			srvspr.HWSprId = hw2.Id
		}
	}
	if Id, err := o.Insert(&srvspr); err != nil {
		fmt.Println("Ошибка записи справочника серверов", err)
		return 0, err
	} else {
		return Id, nil
	}
}

// внести корректировку по уникальности имён
func checkAndCorrectHost(records []*models.ServerSprRecords, host vsphere.HostInfo) int64 {
	//strings.EqualFold(string, string)
	count := len(records)
	if count == 0 {
		return 0
	}
	if count == 1 {
		return records[0].Id
	}
	var foundIndex []int
	var excludeIndex []int

	o := orm.NewOrm()
	//o.Using("default")

	for i := range records {
		foundIndex = append(foundIndex, 0)

		hw := models.ServerHardware{}
		qs := o.QueryTable("server_hardware").OrderBy("server_id").Filter("server_id", records[i].Id)
		err := qs.One(&hw)

		if !strings.EqualFold(records[i].NameFQDN, host.Name) || records[i].CPUCount != float64(host.NumCPUPkgs) || (err == nil && hw.SerialNum != host.SerialNumber) {
			excludeIndex = append(excludeIndex, i)
			continue
		}

		if records[i].Service != "" {
			foundIndex[i]++
		}
		if records[i].Owner != "" {
			foundIndex[i]++
		}
		if records[i].Allocation != "" {
			foundIndex[i]++
		}
		if records[i].Anotation != "" {
			foundIndex[i]++
		}
		if records[i].Role != "" {
			foundIndex[i]++
		}
		if records[i].Details != "" {
			foundIndex[i]++
		}
	}
	max := 0
	idx := 0
	for j := range foundIndex {
		if max < foundIndex[j] {
			max = foundIndex[j]
			idx = j
		}
	}

	for i := range records {
		if i == idx {
			continue
		}
		exclude := false                  //флаг исключения
		for _, ex := range excludeIndex { //проверяем текущий индекс в списке исключений
			if i == ex {
				exclude = true
			}
		}
		if exclude {
			continue
		}

		srvspr := models.ServerSprRecords{Id: records[i].Id}
		if err := o.Read(&srvspr, "Id"); err == nil {
			srvspr.Stat = 666
			srvspr.OS = host.ProductFullName //копируем точную версию esxi

			srvspr.Vcenter = host.Vcenter
			srvspr.Datacenter = host.Datacenter
			srvspr.Cluster = host.Cluster

			num, err := o.Update(&srvspr)
			_ = num
			_ = err
		}
	}
	return records[idx].Id
}

// UpdateHWHostInfo синхронизирует конфигурации хостов-гипервизоров ESXi из инфраструктуры
// vSphere со справочником server_spr_records: создаёт новые записи, устраняет дубликаты
// по имени и серийному номеру, связывает записи с данными оборудования из HP OneView.
func UpdateHWHostInfo() {
	fmt.Println("Обновление конфигураций хостов")
	//	LockBase()
	//	defer UnlockBase()
	ov := oneview.GetGlobalOVInfrastructure()
	infra := vsphere.GetGlobalVMInfrastructure()

	o := orm.NewOrm()
	//o.Using("default")

	for _, key := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(key)
		if !ok {
			continue
		}
		sn := ""
		hw, ok := ov.ServerByVirtualSN[hs.SerialNumber] //переделать на другой тип хэш таблицы
		if ok {
			sn = hw.Base.SerialNumber.String()
		} else {
			sn = hs.SerialNumber
		}
		hw, ok = ov.ServerBySN[sn]
		if ok {
			s := strings.TrimSpace(hw.EnclosureInfo.SerialNumber) //если корзина реально есть
			if s != "" {
				hs.EnclosureSerialNumber = s
			}
		}
		//		hostName := strings.Split(strings.ToLower(hs.Name), ".")
		vc := models.VcenterSprRecords{Name: hs.Vcenter}
		if err := o.Read(&vc, "Name"); err != nil {
			vc.Id_User = 0
			vc.Time_Mfy = time.Now().Local()
			o.Insert(&vc)
		}

		dc := models.DatacenterSprRecords{Name: hs.Datacenter, Vcenter: hs.Vcenter}
		if err := o.Read(&dc, "Name"); err != nil {
			dc.Id_User = 0
			dc.Time_Mfy = time.Now().Local()
			o.Insert(&dc)
		}

		cl := models.ClusterSprRecords{Name: hs.Cluster, Datacenter: hs.Datacenter, Vcenter: hs.Vcenter}
		if err := o.Read(&cl, "Name"); err != nil {
			cl.Id_User = 0
			cl.Time_Mfy = time.Now().Local()
			o.Insert(&cl)
		}

		var records []*models.ServerSprRecords

		shortName := strings.TrimSpace(strings.Split(hs.Name, ".")[0]) //получить краткое имя сервера без указания домена
		serverName := strings.TrimSpace(hs.Name)

		if shortName == "" || serverName == "" || shortName == "localhost" || serverName == "localhost" {
			fmt.Printf("Ошибка в данных гипервизора SN:%s shortName:%s serverName:%s", hs.SerialNumber, shortName, serverName)
			continue
		}

		qs := o.QueryTable("server_spr_records")
		qs = qs.OrderBy("name") //сортировка по кратким именам серверов
		cond := orm.NewCondition()

		cond1 := cond.And("type__ne", 1).AndCond(cond.Or("name__iexact", shortName).Or("name__iexact", serverName).Or("name_f_q_d_n__iexact", shortName).Or("name_f_q_d_n__iexact", serverName))
		qs = qs.SetCond(cond1) // type != 1 физические серверы и хосты + краткое имя без учета регистра выбираем все потому как могли появиться в базе по данным из HP OneView

		//		qs = qs.Filter("type__ne", 1).Filter("name__icontains", shortName) // type != 1 физические серверы и хосты + краткое имя без учета регистра выбираем все потому как
		num, err := qs.All(&records)
		_ = err
		serialNum := ""
		if num == 0 { //если не найден по имени пытаемся найти через серийный номер и базу оборудования
			srvHW := models.ServerHardware{}
			var errh error
			hwsrv, ok := ov.ServerByVirtualSN[hs.SerialNumber]
			if ok {
				serialNum = hwsrv.Base.SerialNumber.String()
			} else {
				serialNum = hs.SerialNumber
			}
			srvHW.SerialNum = serialNum
			errh = o.Read(&srvHW, "SerialNum")
			if errh == nil {

				if hs.PowerState /*|| hs.InMaintenanceMode*/ {
					srvHW.PowerState = 1
				} else {
					srvHW.PowerState = 2
				}
				srvHW.SourceUpdate = "Vmware " + hs.Vcenter + " " + hs.Datacenter + " " + hs.Cluster
				o.Update(&srvHW)

				qs = o.QueryTable("server_spr_records")
				qs = qs.OrderBy("name") //сортировка по кратким именам серверов
				cond = orm.NewCondition()
				cond = cond.And("id__eq", srvHW.ServerId)
				qs = qs.SetCond(cond) //
				num, err = qs.All(&records)
			}
		}

		serverId := int64(0)
		switch {
		case num == 1:
			serverId = records[0].Id //всё отлично одна запись в справочнике
		case num > 1:
			serverId = checkAndCorrectHost(records, *hs) //требуется коррекция с удалением лишних записей или корректировкой
		case num == 0:
			serverId, _ = insertNewHostSprRecord(*hs) //новая запись
		}

		srvspr := models.ServerSprRecords{Id: serverId}
		if err := o.Read(&srvspr, "Id"); err == nil {
			if hs.PowerState /*|| hs.InMaintenanceMode*/ {
				srvspr.PowerState = 1
			} else {
				srvspr.PowerState = 2
			}

			srvspr.Name = shortName
			srvspr.NameFQDN = serverName
			srvspr.Type = 2 //host vmware
			srvspr.CPUCount = float64(hs.NumCPUPkgs)
			srvspr.CPULogical = float64(hs.NumCPUCores)
			srvspr.MemSize = float64(hs.MemorySizeGB)

			srvspr.OS = hs.ProductFullName //копируем точную версию esxi

			srvspr.Vcenter = hs.Vcenter
			srvspr.Datacenter = hs.Datacenter
			srvspr.Cluster = hs.Cluster
			srvspr.SerialNumber = serialNum

			srvHW := models.ServerHardware{}
			/*
				if hs.SerialNumber != "" {
					srvHW.SerialNum = hs.SerialNumber
					err = o.Read(&srvHW, "SerialNum")
				} else {
					srvHW.ServerId = srvspr.Id
					err = o.Read(&srvHW, "ServerId")
				}
			*/
			srvHW.SerialNum = serialNum
			err = o.Read(&srvHW, "SerialNum")
			if err != nil {
				srvHW.ServerId = srvspr.Id
				err = o.Read(&srvHW, "ServerId")
			}
			if err == nil {
				srvspr.HWSprId = srvHW.Id

				if hs.PowerState /*|| hs.InMaintenanceMode*/ {
					srvHW.PowerState = 1
				} else {
					srvHW.PowerState = 2
				}

				if srvHW.Vendor == "" {
					srvHW.Vendor = hs.Vendor
				}
				if srvHW.Model == "" {
					srvHW.Model = hs.Model
				}
				if srvHW.CPUModel == "" {
					srvHW.CPUModel = hs.CPUModel
				}
				if srvHW.CPUCount == 0 {
					srvHW.CPUCount = int(hs.NumCPUPkgs)
				}
				if srvHW.CorePerCPU == 0 {
					srvHW.CorePerCPU = int(hs.NumCPUCores)
				}
				if srvHW.CoreMHz == 0 {
					srvHW.CoreMHz = int(hs.CPUMhz)
				}
				if srvHW.SerialNum == "" {
					//srvHW.SerialNum = hs.SerialNumber
					srvHW.SerialNum = serialNum
				}
				srvHW.SourceUpdate = "Vmware " + hs.Vcenter + " " + hs.Datacenter + " " + hs.Cluster
				_, err := o.Update(&srvHW)
				if err != nil {
					fmt.Println("Ошибка обновления справочника конфигураций", hs.Name, err)
				}
			}
			_, err = o.Update(&srvspr)

		}
	}
	fmt.Println("Обновление конфигураций хостов завершено")
}
