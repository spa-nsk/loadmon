package cmdb

import (
	"loadmon/grpcclient"
	"loadmon/models"
	"log"

	"github.com/beego/beego/v2/client/orm"
)

// CheckServerSprRecordsExistence вызывается после сохранения данных по
// кластерам/хостам/виртуальным машинам (см. SaveMetricData,
// cmdb/savemetric_orchestration.go - вызывается сразу после цикла по
// кластерам, то есть уже после SaveVMRecord по всем кластерам) и приводит
// в порядок статусы справочника серверов (models.ServerSprRecords):
//
//  1. Записи с Stat==3 ("архивная запись после переименования" - см.
//     комментарий к полю Stat в models.ServerSprRecords) переводятся в
//     Stat=0 ("историческая запись") одним пакетным UPDATE по всей
//     таблице, без ограничения по типу записи.
//  2. Для виртуальных машин (Type==1) и гипервизоров (Type==2) проверяется
//     фактическое наличие объекта в VMware (через grpcclient - те же
//     функции, что использует остальной веб-слой для чтения live-данных
//     vSphere, см. kontext/01-architecture.md):
//     - ВМ ищется по Vcenter/Datacenter/Cluster/Vm_Name из связанной
//     записи models.VMSprRecord (ServerSprRecords.VMSprId -> VMSprRecord.Id);
//     - гипервизор ищется по Vcenter/Datacenter/Cluster/Name самой записи
//     ServerSprRecords (для хостов отдельного справочника, аналогичного
//     VMSprRecord, в проекте нет).
//     Если объект не найден в VMware (переименован, удалён, перемещён вне
//     наблюдаемых кластеров и т.п.), запись помечается как выведенная из
//     эксплуатации: Stat=0, PowerState=2 (выключен), Active=2 (выведено из
//     эксплуатации); для ВМ дополнительно в связанной записи
//     models.VMSprRecord выставляется Vm_PowerState=false.
//
// Проверяются только записи с Active==1 (уже активные) - записи, ранее
// выведенные из эксплуатации, повторно не запрашиваются у vSphere на
// каждый прогон (это были бы бесполезные gRPC-запросы: если ВМ пропала
// один раз, то за минуты между прогонами SaveMetricData она не появится
// снова - повторное появление обрабатывается обычным путём восстановления
// записи через SaveVMRecord при следующем реальном обнаружении ВМ в
// vSphere).
func CheckServerSprRecordsExistence() {
	o := orm.NewOrm()

	if _, err := o.QueryTable("server_spr_records").Filter("Stat", 3).Update(orm.Params{"Stat": 0}); err != nil {
		log.Println("CheckServerSprRecordsExistence: ошибка сброса Stat==3 -> 0:", err)
	}

	var servers []*models.ServerSprRecords
	_, err := o.QueryTable("server_spr_records").
		Filter("Type__in", 1, 2).
		Filter("Active", 1).
		All(&servers)
	if err != nil {
		log.Println("CheckServerSprRecordsExistence: ошибка чтения справочника серверов:", err)
		return
	}

	for _, srv := range servers {
		if srv == nil {
			continue
		}
		switch srv.Type {
		case 1:
			checkVMExistence(o, srv)
		case 2:
			checkHypervisorExistence(o, srv)
		}
	}
}

// checkVMExistence проверяет наличие в VMware виртуальной машины,
// связанной со srv (Type == 1), по Vcenter/Datacenter/Cluster/Vm_Name из
// models.VMSprRecord - если ВМ не найдена, помечает srv и связанную
// запись VMSprRecord как выведенные из эксплуатации (см. комментарий к
// CheckServerSprRecordsExistence выше).
func checkVMExistence(o orm.Ormer, srv *models.ServerSprRecords) {
	if srv.VMSprId <= 0 {
		return
	}
	vmSpr := models.VMSprRecord{Id: srv.VMSprId}
	if err := o.Read(&vmSpr); err != nil {
		return // нет связанной записи справочника ВМ - проверять нечего
	}
	if vmSpr.Vcenter == "" || vmSpr.Cluster == "" || vmSpr.Vm_Name == "" {
		return // недостаточно данных для поиска в VMware
	}

	// Намеренно GetVMByKeyDirect, а не GetVMByKey - запрос идёт напрямую в
	// VMware, минуя закэшированный в памяти пул данных сборщика (который
	// мог ещё не заметить удаление/переименование ВМ до следующего опроса
	// по тикеру) - иначе можно было бы ошибочно пометить ещё существующую
	// ВМ как выведенную из эксплуатации только из-за отставания кэша.
	if _, err := grpcclient.GetVMByKeyDirect(vmSpr.Vcenter, vmSpr.Datacenter, vmSpr.Cluster, vmSpr.Vm_Name); err == nil {
		return // ВМ найдена в VMware - запись актуальна, ничего не меняем
	}

	srv.Stat = 0
	srv.PowerState = 2
	srv.Active = 2
	if _, err := o.Update(srv, "Stat", "PowerState", "Active"); err != nil {
		log.Println("CheckServerSprRecordsExistence: ошибка обновления записи ВМ", srv.Name, err)
	}

	vmSpr.Vm_PowerState = false
	if _, err := o.Update(&vmSpr, "Vm_PowerState"); err != nil {
		log.Println("CheckServerSprRecordsExistence: ошибка обновления Vm_PowerState для", vmSpr.Vm_Name, err)
	}
}

// checkHypervisorExistence проверяет наличие в VMware хоста (гипервизора),
// соответствующего srv (Type == 2), по Vcenter/Datacenter/Cluster/Name
// самой записи ServerSprRecords - если хост не найден, помечает srv как
// выведенный из эксплуатации (см. комментарий к
// CheckServerSprRecordsExistence выше).
func checkHypervisorExistence(o orm.Ormer, srv *models.ServerSprRecords) {
	if srv.Vcenter == "" || srv.Cluster == "" || srv.Name == "" {
		return // недостаточно данных для поиска в VMware
	}

	// Намеренно GetHostByKeyDirect, а не LoadHostListFull - запрос идёт
	// напрямую в VMware, минуя закэшированный в памяти пул данных сборщика
	// (см. аналогичный комментарий выше в checkVMExistence).
	if _, err := grpcclient.GetHostByKeyDirect(srv.Vcenter, srv.Datacenter, srv.Cluster, srv.Name); err == nil {
		return // хост найден в VMware - запись актуальна, ничего не меняем
	}

	srv.Stat = 0
	srv.PowerState = 2
	srv.Active = 2
	if _, err := o.Update(srv, "Stat", "PowerState", "Active"); err != nil {
		log.Println("CheckServerSprRecordsExistence: ошибка обновления записи гипервизора", srv.Name, err)
	}
}
