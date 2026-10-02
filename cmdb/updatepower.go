package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
	models "loadmon/models"
	_ "loadmon/routers"
	vsphere "loadmon/vspheremetrics"
	"log"

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

// UpdatePowerStateForServers (раз в час, metrics_grpc) - применяет единое
// правило Stat/PowerState/Active ко всему справочнику серверов.
//
// С 2026-10-02 собственной логики здесь нет: прежний разбор по типам записи
// (опрос HP OneView по серийному номеру, VMMapList для ВМ, GetHostBySN для
// гипервизоров) заменён общей функцией ReconcileServerSprRecordsTypeAndState
// (cmdb/reconcileserversprtype.go), которая работает по тем же источникам
// плюс Zabbix/SCOM, учитывает достоверность источников и ручные записи и
// одинаково обновляет ServerSprRecords и связанный ServerHardware.
func UpdatePowerStateForServers() {
	ReconcileServerSprRecordsTypeAndState()
}
