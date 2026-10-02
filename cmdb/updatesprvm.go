package cmdb

import (
	"fmt"
	models "loadmon/models"
	_ "loadmon/routers"
	vsphere "loadmon/vspheremetrics"
	"log"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

func checkAndCorrectVmSpr(records []*models.VMSprRecord, vm vsphere.VMPoint) int64 {
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

	for i := range records {
		foundIndex = append(foundIndex, 0)
		if records[i].Vcenter != vm.Vcenter || records[i].Datacenter != vm.Datacenter || records[i].Cluster != vm.Cluster {
			excludeIndex = append(excludeIndex, i)
			continue
		}

		if records[i].Vm_Annotation != "" {
			foundIndex[i]++
		}
		if records[i].Vm_IpAddress != "" {
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

	o := orm.NewOrm()
	//o.Using("default")

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
		vmspr := models.VMSprRecord{Id: records[i].Id}
		if err := o.Read(&vmspr, "Id"); err == nil {
			var list []*models.ServerSprRecords
			qs := models.ServerSprQS(o)
			qs = qs.OrderBy("VMSprId")
			qs = qs.Filter("VMSprId", records[i].Id)
			num, err := qs.All(&list)
			for _, srv := range list {
				r := models.ServerSprRecords{Id: srv.Id}
				o.Read(&r, "Id")
				r.VMSprId = 0
				r.Stat = models.ServerSprStatDestroy // было 6666 - опечатка, значение вне всех фильтров
				num, err := o.Update(&r)
				_ = num
				_ = err
			}
			num, err = o.Delete(&vmspr, "Id")
			_ = num
			_ = err
		}
	}
	return records[idx].Id
}

func checkAndCorrectVM(records []*models.ServerSprRecords, vm vsphere.VMPoint) int64 {
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

	for i, _ := range records {
		foundIndex = append(foundIndex, 0)
		if records[i].VMSprId > 0 {
			vmspr := models.VMSprRecord{Id: records[i].VMSprId}
			err := o.Read(&vmspr, "Id") //поиск vm по Id
			if err == nil {
				if vmspr.Vcenter != vm.Vcenter || vmspr.Datacenter != vm.Datacenter || vmspr.Cluster != vm.Cluster {
					excludeIndex = append(excludeIndex, i)
				}
			}
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
			srvspr.Stat = models.ServerSprStatDestroy
			num, err := o.Update(&srvspr)
			_ = num
			_ = err
		}
	}
	return records[idx].Id
}

// UpdateSprVM обновляет справочник виртуальных машин (vm_spr_record) и связанные с ним
// записи серверов на основании данных глобальной инфраструктуры vSphere.
func UpdateSprVM() {
	fmt.Println("Обновление справочника виртуальных машин")
	//LockBase()
	//defer UnlockBase()

	Infra := vsphere.GetGlobalVMInfrastructure()
	if Infra == nil {
		log.Println("Ошибка сохранения данных по виртуальным машинам: Инфраструктура не инициализирована")
		return
	}
	//currentTime := time.Now().In(LocalTimezone)
	o := orm.NewOrm()
	//o.Using("default")

	for _, key := range Infra.VMMapList.Keys() {
		vm, ok := Infra.VMMapList.Get(key)
		if !ok {
			continue
		}

		/*
			if !vm.Vm_PowerState { //игнорируем выключенные VM
				continue
			}
		*/
		if vm.WorkStation { //пропускаем рабочие станции windows
			continue
		}
		if vm.Cluster == "не в кластере" && vm.Datacenter == "нет данных" { //пропускаем vm без заполненных данных (возникает при длительной потере связи с vcenter)
			continue
		}

		if strings.Contains(vm.GuestID, "windows") { //игнорируем рабочие станции windows
			if !strings.Contains(vm.GuestID, "Server") {
				continue
			}
		}
		var vmRecs []*models.VMSprRecord
		qs := o.QueryTable("vm_spr_record")
		qs = qs.Filter("Vcenter", vm.Vcenter).Filter("Datacenter", vm.Datacenter).Filter("Cluster", vm.Cluster).Filter("Vm_Name", vm.Name)
		vmCount, vmErr := qs.All(&vmRecs)
		_ = vmCount
		_ = vmErr

		id := int64(0)
		if vmCount > 1 {
			id = checkAndCorrectVmSpr(vmRecs, *vm)
		} else if vmCount == 1 {
			id = vmRecs[0].Id
		}

		vmspr := models.VMSprRecord{Id: id}
		VMSprId := int64(0)
		//		o.Using("vm")

		err := o.Read(&vmspr, "Id") //поиск vm по Id
		if err == nil && vmCount > 0 {
			VMSprId = vmspr.Id //запомнить id
			vm.ID = vmspr.Id   //внести id по справочнику !!!!!!!!
			CopyVmToSprVm(*vm, &vmspr)
			vmspr.Id = VMSprId
			num, erru := o.Update(&vmspr)
			if erru != nil {
				fmt.Println("Ошибка обновления в справочнике виртуальных машин: ", vm.HostName, erru, num)
				continue
			}
		} else {
			CopyVmToSprVm(*vm, &vmspr)
			VMSprId, err = o.Insert(&vmspr)
			if err != nil {
				fmt.Println("Ошибка сохранения в справочнике виртуальных машин: ", err)
				continue
			}
			vm.ID = VMSprId //внести id по справочнику !!!!!!!!
		}

		var records []*models.ServerSprRecords

		vmName := vm.Name
		vmHostName := vm.HostName

		if vm.HostName == "" {
			vmHostName = vm.Name
		}

		if vmName == "" || vmHostName == "" || vmName == "localhost" || vmHostName == "localhost" {
			fmt.Printf("Ошибка в имени VM vmName: %s vmHostName: %s", vmName, vmHostName)
			continue
		}

		qs = models.ServerSprQS(o)
		qs = qs.OrderBy("name") //сортировка по кратким именам серверов
		//		qs = qs.Filter("type__eq", 1).Filter("VMSprId__eq", VMSprId).Filter("name__icontains", vmName) // type == 1 виртуальные серверы + краткое имя без учета регистра

		cond := orm.NewCondition()
		cond1 := cond.And("Type__eq", 1).And("VMSprId__eq", VMSprId).AndCond(cond.Or("Name__iexact", vmName).Or("Name__iexact", vmHostName).Or("NameFQDN__iexact", vmName).Or("NameFQDN__iexact", vmHostName))
		//cond1 := cond.And("type__eq", 1).AndCond(cond.And("name__iexact", vmName).Or("name__iexact", vmHostName).Or("NameFQDN__iexact", vmName).Or("NameFQDN__iexact", vmHostName))
		qs = qs.SetCond(models.ExcludeDestroyedCond(cond1)) // type == 1 виртуальные серверы + краткое имя без учета регистра

		num, err := qs.All(&records)
		_ = err

		serverId := int64(0)
		srvspr := models.ServerSprRecords{}

		switch {
		case num == 1:
			serverId = records[0].Id //всё отлично одна запись в справочнике
		case num > 1:
			serverId = checkAndCorrectVM(records, *vm) //требуется коррекция с удалением лишних записей или корректировкой
		case num == 0: //требуется вставка записи
			CopyVmToSprSrv(*vm, &srvspr)
			srvspr.Type = 1
			srvspr.Stat = 1
			srvspr.Details = srvspr.Details + " Добавлен из Vsphere, пребуется правка"
			srvspr.VMSprId = VMSprId
			srvspr.Time_Mfy = time.Now() // время модификации
			srvspr.SourceUpdateType = models.SourceUpdateVMware
			// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
			// (10002 - VMware/vSphere) для новой записи ServerSprRecords.
			srvspr.TimeCreate = time.Now().Local()
			srvspr.CreatorUser = 10002
			serverId, err = o.Insert(&srvspr)
			if err != nil {
				fmt.Println("Ошибка записи в справочник виртуальных машин: ", err)
				continue
			} else {
				vmu := models.VMSprRecord{Id: VMSprId}
				o.Read(&vmu, "Id")
				vmu.ServerId = serverId //запомнить ссылку на сервер
				o.Update(&vmu)
			}
		}

		//      Vm_HostName := strings.Split(strings.ToLower(vm.Vm_HostName), ".") 19.01.2023 приведение баз к виду именования как в vcenter
		//      srvspr := models.ServerSprRecords{Name: Vm_HostName[0]}
		//      srvspr := models.ServerSprRecords{NameFQDN: vm.Vm_HostName}

		srvspr.Id = serverId

		// if err := o.Read(&srvspr, "NameFQDN"); err == nil {
		if err := o.Read(&srvspr, "Id"); err == nil {
			UpdateVmToSprSrv(*vm, &srvspr) //обновить параметры из vm
			srvspr.Type = 1
			srvspr.Name = vm.Name
			srvspr.VMSprId = VMSprId
			srvspr.Time_Mfy = time.Now() // время модификации
			srvspr.SourceUpdateType = models.SourceUpdateVMware
			if _, err := o.Update(&srvspr); err != nil {
				fmt.Println("Ошибка привязки виртуальной машины к справочнику серверов: ", srvspr.Name, err)
			} else {
				vmu := models.VMSprRecord{Id: VMSprId}
				o.Read(&vmu, "Id")
				vmu.ServerId = serverId //запомнить ссылку на сервер
				o.Update(&vmu)
			}
		} else {
			_ = err
		}
	}
}
