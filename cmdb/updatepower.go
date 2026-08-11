package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
	models "loadmon/models"
	"loadmon/oneview"
	_ "loadmon/routers"
	vsphere "loadmon/vspheremetrics"
	"log"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// UpdateClusterSprRecordsV2 актуализирует записи ClusterSprRecords, сверяя их с
// данными глобальной инфраструктуры vSphere (кэш карты кластеров) и обновляя
// идентификатор и статус кластера.
func UpdateClusterSprRecordsV2() {
	var records []*models.ClusterSprRecords

	infra := vsphere.GetGlobalVMInfrastructure()
	if infra == nil {
		log.Println("Ошибка сохранения данных по виртуальным машинам: Инфраструктура не инициализирована")
		return
	}
	o := orm.NewOrm()
	//o.Using("default")
	qs := o.QueryTable("cluster_spr_records")
	qs = qs.OrderBy("name") //сортировка по кратким именам серверов
	qs.All(&records)
	for _, clSpr := range records {
		clRec := models.ClusterSprRecords{Id: clSpr.Id}
		err := o.Read(&clRec, "Id")
		if err != nil { //возможно запись уже удалена
			continue
		}
		key := fmt.Sprintf("%s/%s/%s", clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
		cl, ok := infra.ClusterMapList.Get(key)
		if !ok || cl == nil {
			fmt.Println("Ошибка получения данных кластера", clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
			clRec.Stat = 0
			_, err = o.Update(&clRec)
			if err != nil {
				fmt.Println("Ошибка обновления данных кластера", clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
			}
			continue
		}
		clRec.ClusterId = cl.ClusterID
		clRec.Stat = 1
		_, err = o.Update(&clRec)
		if err != nil {
			fmt.Println("Ошибка обновления данных кластера", clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
		}
	}
}

// UpdateClusterSprRecords актуализирует записи ClusterSprRecords, запрашивая
// текущие данные кластера через grpcclient, и обновляет идентификатор и статус кластера.
func UpdateClusterSprRecords() {
	var records []*models.ClusterSprRecords

	infra := vsphere.GetGlobalVMInfrastructure()
	if infra == nil {
		log.Println("Ошибка сохранения данных по виртуальным машинам: Инфраструктура не инициализирована")
		return
	}
	o := orm.NewOrm()
	//o.Using("default")
	qs := o.QueryTable("cluster_spr_records")
	qs = qs.OrderBy("name") //сортировка по кратким именам серверов
	qs.All(&records)
	for _, clSpr := range records {
		clRec := models.ClusterSprRecords{Id: clSpr.Id}
		err := o.Read(&clRec, "Id")
		if err != nil { //возможно запись уже удалена
			continue
		}
		cl := grpcclient.GetClusterMedium(clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
		if cl == nil {
			fmt.Println("Ошибка получения данных кластера", clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
			clRec.Stat = 0
			_, err = o.Update(&clRec)
			if err != nil {
				fmt.Println("Ошибка обновления данных кластера", clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
			}
			continue
		}
		clRec.ClusterId = cl.ClusterID
		clRec.Stat = 1
		_, err = o.Update(&clRec)
		if err != nil {
			fmt.Println("Ошибка обновления данных кластера", clSpr.Vcenter, clSpr.Datacenter, clSpr.Name)
		}
	}
}

// UpdatePowerStateForServers обновляет состояние включения (PowerState) для всех
// серверов из server_spr_records в зависимости от их типа: физический сервер и его
// конфигурация опрашиваются через HP OneView, виртуальные машины сверяются с данными
// vSphere, гипервизоры проверяются через grpcclient и HP OneView.
func UpdatePowerStateForServers() {

	var records []*models.ServerSprRecords

	infra := vsphere.GetGlobalVMInfrastructure()
	if infra == nil {
		log.Println("Ошибка сохранения данных по виртуальным машинам: Инфраструктура не инициализирована")
		return
	}

	ov := oneview.GetGlobalOVInfrastructure()

	o := orm.NewOrm()
	//o.Using("default")
	qs := o.QueryTable("server_spr_records")
	qs = qs.OrderBy("name") //сортировка по кратким именам серверов
	qs.All(&records)

	for _, srv := range records {
		if srv.Stat == 666 { //пропускаем помеченные для удаления
			continue
		}
		trn, err := o.Begin()
		if err != nil {
			return
		}
		//defer trn.RollbackUnlessCommit()
		serverSpr := models.ServerSprRecords{Id: srv.Id}
		err = trn.Read(&serverSpr, "Id")
		serverSpr.PowerState = 2 //по умолчанию ставим статус выключен
		if err != nil {          //странно но могли удалить после чтения списка
			trn.Rollback()
			continue
		}
		switch srv.Type { //выбор по типу серера
		case 0: //аппаратный сервер
			hw := models.ServerHardware{Id: srv.HWSprId}
			err := trn.Read(&hw, "Id")
			if err == nil {
				hwsn, err := ov.FindServerHardwareSN(hw.SerialNum) //делаем запрос в HP OneView по серийному номеру сервера
				if err == nil {
					if hwsn.Base.PowerState == "On" { //сервер включен
						serverSpr.PowerState = 1
						hw.PowerState = 1
					} else {
						serverSpr.PowerState = 2
						hw.PowerState = 2
					}
				} else { //нет среди серверов в hp one view помечаем как выключенные до появления метрик из других систем
					serverSpr.PowerState = 2
					hw.PowerState = 2
				}
				if serverSpr.Stat == 0 {
					hw.Active = 2
				}
				_, err = trn.Update(&hw)
				if err != nil {
					fmt.Println("Ошибка обновления записи справочника виртуальных машин о статусе включения", err)
					trn.Rollback()
					continue
				}
			} else {
				serverSpr.PowerState = 2 //не найден значит ставим выключен
			}
		case 1: //виртуальная машина
			if srv.VMSprId == 0 {
				fmt.Println("Ошибка формата записи, нет ссылки на справочник виртуальных машин ServerSprRecords.Id=", srv.Id)
				trn.Rollback()
				continue
			}
			vmspr := models.VMSprRecord{Id: srv.VMSprId}
			err = trn.Read(&vmspr, "Id") //поиск vm по Id в справочнике виртуальных машин
			if err == nil {              //найден
				keyName := vmspr.Vcenter + "/" + vmspr.Datacenter + "/" + vmspr.Cluster + "/" + vmspr.Vm_Name
				vms, ok := infra.VMMapList.Get(keyName) //проверяем наличие виртуальной машины среди включенных
				if ok {
					if vms.PowerState {
						serverSpr.PowerState = 1
					} else {
						serverSpr.PowerState = 2
					}
					vmspr.Vm_PowerState = vms.PowerState

				} else { //если не найдена среди включенных ставим статус выключена в справочнике
					vmspr.Vm_PowerState = false
					serverSpr.PowerState = 2 //указываем явно
				}

				_, err = trn.Update(&vmspr)
				if err != nil {
					fmt.Println("Ошибка обновления записи справочника виртуальных машин о статусе включения", err)
					trn.Rollback()
					continue
				}
			} else {
				fmt.Println("Виртуальный сервер ", srv.NameFQDN, " не найден в справочнике виртуальных машин по Id", srv.VMSprId)
				serverSpr.PowerState = 2 //указываем явно
			}
			up := strings.ToUpper(serverSpr.Name)
			if strings.Contains(up, "_OFF") || strings.Contains(up, "_DEL") { //такие имена делают перед удалением
				serverSpr.PowerState = 2 //выключен
				serverSpr.Active = 2     //выведен из эксплуатации
			}
		case 2: //если это гипервизор vmware
			if srv.HWSprId == 0 {
				fmt.Println("Ошибка формата записи, нет ссылки на справочник конфигураций серверов ServerSprRecords.Id=", srv.Id)
				trn.Rollback()
				continue
			}

			powerState := false
			if srv.SerialNumber != "" {
				host := grpcclient.GetHostBySN(srv.SerialNumber)
				if host != nil {
					if host.PowerState {
						powerState = true
					}
				}
			}

			if powerState == false {
				hwsn, err := ov.FindServerHardwareSN(srv.SerialNumber) //поиск в HP OneView
				if err == nil {                                        //найден среди серверов в HP OneView
					if hwsn.Base.PowerState == "On" {
						powerState = true
					}
				}
			}

			if powerState {
				serverSpr.PowerState = 1
			} else {
				serverSpr.PowerState = 2
			}

			hw := models.ServerHardware{Id: srv.HWSprId}
			err = trn.Read(&hw, "Id") //проверяем в справочнике конфигураций серверов
			if err == nil {
				if powerState {
					hw.PowerState = 1
				} else {
					hw.PowerState = 2
				}
				_, err = trn.Update(&hw)
				if err != nil {
					fmt.Println("Ошибка обновления записи справочника конфигураций серверов о статусе включения", err)
					trn.Rollback()
					continue
				}
			} else {
				fmt.Println("Нарушение связности для ServerSprRecords.Id=", srv.Id, "нет связанной записи HWSprId=", srv.HWSprId)
			}
		}

		if serverSpr.Stat == 0 || serverSpr.Stat == 666 {
			serverSpr.Active = 2
		}
		_, err = trn.Update(&serverSpr)
		if err != nil {
			fmt.Println("Ошибка обновления записи справочника серверов о статусе включения", err)
			trn.Rollback()
			continue
		}
		trn.Commit()
	}
}
