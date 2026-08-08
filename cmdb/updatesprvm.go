package cmdb

import (
	"fmt"
	"loadmon/grpcclient"
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
		exclude := false                //флаг исключения
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
			qs := o.QueryTable("server_spr_records")
			qs = qs.OrderBy("v_m_spr_id")
			qs = qs.Filter("VMSprId", records[i].Id)
			num, err := qs.All(&list)
			for _, srv := range list {
				r := models.ServerSprRecords{Id: srv.Id}
				o.Read(&r, "Id")
				r.VMSprId = 0
				r.Stat = 6666
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
		exclude := false                //флаг исключения
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
			num, err := o.Update(&srvspr)
			_ = num
			_ = err
		}
	}
	return records[idx].Id
}

func UpdateSprVMv2() {
	fmt.Println("Обновление справочника виртуальных машин")
	//LockBase()
	// UnlockBase()

	Infra := vsphere.GetGlobalVMInfrastructure()
	if Infra == nil {
		log.Println("Ошибка сохранения данных по виртуальным машинам: Инфраструктура не инициализирована")
		return
	}
	//currentTime := time.Now().In(LocalTimezone)
	o := orm.NewOrm()
	//o.Using("default")
	var vmRecs []*models.VMSprRecord
	qs := o.QueryTable("vm_spr_record")
	//  qs = qs.Filter("Vm_PowerState", true)
	vmCount, vmErr := qs.All(&vmRecs)
	_ = vmCount
	_ = vmErr
	for _, rec := range vmRecs {
		key := fmt.Sprintf("%s/%s/%s/%s", rec.Vcenter, rec.Datacenter, rec.Cluster, rec.Vm_Name)
		//vm, ok := Infra.VMMapList.Get(key)
		vm, err2 := grpcclient.GetVM(key)

		data := models.VMSprRecord{Id: rec.Id}
		err := o.Read(&data, "Id")

		srv := models.ServerSprRecords{Id: data.ServerId}
		err3 := o.Read(&srv, "Id")

		//if ok {
		if err2 == nil {
			if err == nil {
				data.Vm_IpAddress = vm.IPAddress
				data.Vm_Network = vm.Network
				data.Vm_PowerState = vm.PowerState
				o.Update(&data)
				fmt.Println("Обновлены данные:", rec.Vcenter, rec.Datacenter, rec.Cluster, rec.Vm_Name, data.Vm_Network)
				if err3 == nil {
					srv.IPAddress = vm.IPAddress
					srv.Network = vm.Network
					if vm.PowerState {
						srv.PowerState = 1
						srv.Active = 1
					} else {
						srv.PowerState = 2
						srv.Active = 2
					}
					o.Update(&srv)
				}
			}
		} else {
			data.Vm_PowerState = false
			o.Update(&data)
			if err3 == nil {
				srv.PowerState = 2
				srv.Active = 2
				o.Update(&srv)
			}
		}
	}
}

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

		qs = o.QueryTable("server_spr_records")
		qs = qs.OrderBy("name") //сортировка по кратким именам серверов
		//		qs = qs.Filter("type__eq", 1).Filter("v_m_spr_id__eq", VMSprId).Filter("name__icontains", vmName) // type == 1 виртуальные серверы + краткое имя без учета регистра

		cond := orm.NewCondition()
		cond1 := cond.And("type__eq", 1).And("v_m_spr_id__eq", VMSprId).AndCond(cond.Or("name__iexact", vmName).Or("name__iexact", vmHostName).Or("name_f_q_d_n__iexact", vmName).Or("name_f_q_d_n__iexact", vmHostName))
		//cond1 := cond.And("type__eq", 1).AndCond(cond.And("name__iexact", vmName).Or("name__iexact", vmHostName).Or("name_f_q_d_n__iexact", vmName).Or("name_f_q_d_n__iexact", vmHostName))
		qs = qs.SetCond(cond1) // type == 1 виртуальные серверы + краткое имя без учета регистра

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

func UpdateVMMove() {
	fmt.Println("Корректировка данных после переносов виртуальных машин")
	//LockBase()
	//defer UnlockBase()

	infra := vsphere.GetGlobalVMInfrastructure()
	if infra == nil {
		log.Println("Ошибка сохранения данных по виртуальным машинам: Инфраструктура не инициализирована")
		return
	}

	/*	vmlist, err := infra.GetVMList()
		if err != nil {
			log.Println("Ошибка сохранения данных по виртуальным машинам: ", err)
			return
		}
		//currentTime := time.Now().In(LocalTimezone)
	*/
	var (
		records []*models.ServerSprRecords
		srvlist map[string]*models.ServerSprRecords
	)
	srvlist = make(map[string]*models.ServerSprRecords)

	o := orm.NewOrm()
	//o.Using("default")

	qs := o.QueryTable("server_spr_records")
	qs = qs.OrderBy("Type", "Name").Filter("Type", 1).Filter("Stat__ne", 666).Filter("Stat__ne", 3).Filter("OldId", 0) //выбираем VM
	srvCount, srvErr := qs.All(&records)
	_ = srvCount
	_ = srvErr

	for _, srv := range records {

		if srv.Stat == 666 { //пропускаем так как формально удалён
			continue
		}

		if srv.Stat == 3 { //пропускаем так как флаг архивного, после переименования
			continue
		}

		if srv.OldId > 0 { //пропускаем так как уже обрабатывался и помечена ссылка на архив
			continue
		}
		if srv.VMSprId == 0 {
			//добавить обработку
			continue
		}
		/*
			if !vm.Vm_PowerState { //игнорируем выключенные VM
				continue
			}
		*/

		var (
			//vmRecs []*models.VMSprRecord
			vm  models.VMSprRecord
			vm2 models.VMSprRecord
		)
		vm.Id = srv.VMSprId
		err := o.Read(&vm, "Id")
		if err != nil {
			fmt.Println("Ошибка: обрыв связи", srv.Id, " с vm_spr_record", srv.VMSprId)
			continue
		}

		keyName := vm.Vcenter + "/" + vm.Datacenter + "/" + vm.Cluster + "/" + vm.Vm_Name
		//		okvm := infra.VM_list[keyName]
		_, okvm := infra.VMMapList.Get(keyName)
		if s, ok := srvlist[vm.Vcenter+"/"+vm.Vm_Name+"/"+vm.Vm_Uuid]; ok {
			vm2.Id = s.VMSprId
			err := o.Read(&vm2, "Id")
			if err != nil {
				fmt.Println("Ошибка: обрыв связи ", s.Id, " с vm_spr_record", s.VMSprId)
				continue
			}
			if s.PowerState == 1 && !okvm { // считаем сервер с id == s.id активным, а srv.Id архивным

				fmt.Println(" on", s.Id, vm2.Id, s.Time_Mfy.Format("02.01.2006"), vm2.Vcenter, vm2.Datacenter, vm2.Cluster, vm2.Vm_Name, vm2.Vm_HostName, vm2.Vm_Uuid)
				fmt.Println("off", srv.Id, vm.Id, srv.Time_Mfy.Format("02.01.2006"), vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Vm_Name, vm.Vm_HostName, vm.Vm_Uuid)
				srv2 := models.ServerSprRecords{Id: s.Id}
				o.Read(&srv2, "Id")           //запоминаем ссылку на архивную запись
				RestoreSprData(&srv2, *srv)   //копируем данные из архивной записи, если не заполнены
				srv2.OldId = srv.Id           //
				o.Update(&srv2)               //
				srv2.Id = srv.Id              //
				o.Read(&srv2, "Id")           //
				srv2.Stat = 3                 //архивная запись
				srv2.PowerState = 2           //
				srv2.Active = 2               //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

			} else if s.PowerState == 2 && okvm { //считаем сервер srv.Id активным, а s.Id архивным
				fmt.Println("on", srv.Id, vm.Id, srv.Time_Mfy.Format("01.02.2006"), vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Vm_Name, vm.Vm_HostName, vm.Vm_Uuid)
				fmt.Println("off", s.Id, vm2.Id, s.Time_Mfy.Format("01.02.2006"), vm2.Vcenter, vm2.Datacenter, vm2.Cluster, vm2.Vm_Name, vm2.Vm_HostName, vm2.Vm_Uuid)
				srv2 := models.ServerSprRecords{Id: srv.Id}
				o.Read(&srv2, "Id")         //запоминаем ссылку на архивную запись
				RestoreSprData(&srv2, *s)   //копируем данные из архивной записи, если не заполнены
				srv2.OldId = s.Id           //
				o.Update(&srv2)             //
				srv2.Id = s.Id              //
				o.Read(&srv2, "Id")         //
				srv2.Stat = 3               //архивная запись
				srv2.PowerState = 2         //
				srv2.Active = 2             //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

			} else { //оба сервера считаем архивными
				fmt.Println("off", srv.Id, vm.Id, srv.Time_Mfy.Format("01.02.2006"), vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Vm_Name, vm.Vm_HostName, vm.Vm_Uuid)
				fmt.Println("off", s.Id, vm2.Id, s.Time_Mfy.Format("01.02.2006"), vm2.Vcenter, vm2.Datacenter, vm2.Cluster, vm2.Vm_Name, vm2.Vm_HostName, vm2.Vm_Uuid)
				srv2 := models.ServerSprRecords{Id: srv.Id}
				o.Read(&srv2, "Id") //
				srv2.Stat = 3       //архивная запись
				srv2.PowerState = 2 //
				srv2.Active = 2     //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

				srv2.Id = s.Id
				o.Read(&srv2, "Id") //
				srv2.Stat = 3       //архивная запись
				srv2.PowerState = 2 //
				srv2.Active = 2     //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

			}
			continue
		} else {
			p := *srv
			srvlist[vm.Vcenter+"/"+vm.Vm_Name+"/"+vm.Vm_Uuid] = &p
			if okvm {
				p.PowerState = 1
			} else {
				p.PowerState = 2
			}
		}
	}
}

func UpdateVMMoveRename() {
	fmt.Println("Корректировка данных после переносов и переименований виртуальных машин")
	//LockBase()
	//	defer UnlockBase()

	infra := vsphere.GetGlobalVMInfrastructure()
	if infra == nil {
		log.Println("Ошибка сохранения данных по виртуальным машинам: Инфраструктура не инициализирована")
		return
	}

	/*	vmlist, err := infra.GetVMList()
		if err != nil {
			log.Println("Ошибка сохранения данных по виртуальным машинам: ", err)
			return
		}
		//currentTime := time.Now().In(LocalTimezone)
	*/
	var (
		records []*models.ServerSprRecords
		srvlist map[string]*models.ServerSprRecords
	)
	srvlist = make(map[string]*models.ServerSprRecords)

	o := orm.NewOrm()
	//o.Using("default")

	qs := o.QueryTable("server_spr_records")
	qs = qs.OrderBy("Type", "Name").Filter("Type", 1).Filter("Stat__ne", 666).Filter("Stat__ne", 3).Filter("OldId", 0) //выбираем VM
	srvCount, srvErr := qs.All(&records)
	_ = srvCount
	_ = srvErr

	for _, srv := range records {

		if srv.Stat == 666 { //пропускаем так как формально удалён
			continue
		}

		if srv.Stat == 3 { //пропускаем так как флаг архивного, после переименования
			continue
		}

		if srv.OldId > 0 { //пропускаем так как уже обрабатывался и помечена ссылка на архив
			continue
		}
		if srv.VMSprId == 0 {
			//добавить обработку
			continue
		}
		/*
			if !vm.Vm_PowerState { //игнорируем выключенные VM
				continue
			}
		*/

		var (
			//vmRecs []*models.VMSprRecord
			vm  models.VMSprRecord
			vm2 models.VMSprRecord
		)
		vm.Id = srv.VMSprId
		err := o.Read(&vm, "Id")
		if err != nil {
			fmt.Println("Ошибка: обрыв связи", srv.Id, " с vm_spr_record", srv.VMSprId) //нет записи в справочнике виртуальных машин
			continue
		}

		keyName := vm.Vcenter + "/" + vm.Datacenter + "/" + vm.Cluster + "/" + vm.Vm_Name
		//		_, okvm := infra.VM_list[keyName] //проверяем наличие виртуальной машины среди включенных
		_, okvm := infra.VMMapList.Get(keyName) //проверяем наличие виртуальной машины среди включенных

		if s, ok := srvlist[vm.Vcenter+"/"+vm.Vm_Uuid]; ok { //проверяем встречалась ли vm с этим uuid (было переименование или перенос)
			vm2.Id = s.VMSprId
			err := o.Read(&vm2, "Id") //проверяем наличие vm в справочнике
			if err != nil {
				fmt.Println("Ошибка: обрыв связи ", s.Id, " с vm_spr_record", s.VMSprId)
				continue
			}
			if s.PowerState == 1 && !okvm { // считаем сервер c id == s.id активным, а srv.Id архивным

				fmt.Println(" on", s.Id, vm2.Id, s.Time_Mfy.Format("01.02.2006"), vm2.Vcenter, vm2.Datacenter, vm2.Cluster, vm2.Vm_Name, vm2.Vm_HostName, vm2.Vm_Uuid)
				fmt.Println("off", srv.Id, vm.Id, srv.Time_Mfy.Format("01.02.2006"), vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Vm_Name, vm.Vm_HostName, vm.Vm_Uuid)
				srv2 := models.ServerSprRecords{Id: s.Id}
				o.Read(&srv2, "Id")           //запоминаем ссылку на архивную запись
				RestoreSprData(&srv2, *srv)   //копируем данные из архивной записи, если не заполнены
				srv2.OldId = srv.Id           //
				o.Update(&srv2)               //
				srv2.Id = srv.Id              //
				o.Read(&srv2, "Id")           //
				srv2.Stat = 3                 //архивная запись
				srv2.PowerState = 2           //
				srv2.Active = 2               //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

			} else if s.PowerState == 2 && okvm { //считаем сервер srv.Id активным, а s.Id архивным
				fmt.Println(" on", srv.Id, vm.Id, srv.Time_Mfy.Format("01.02.2006"), vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Vm_Name, vm.Vm_HostName, vm.Vm_Uuid)
				fmt.Println("off", s.Id, vm2.Id, s.Time_Mfy.Format("01.02.2006"), vm2.Vcenter, vm2.Datacenter, vm2.Cluster, vm2.Vm_Name, vm2.Vm_HostName, vm2.Vm_Uuid)
				srv2 := models.ServerSprRecords{Id: srv.Id}
				o.Read(&srv2, "Id")         //запоминаем ссылку на архивную запись
				RestoreSprData(&srv2, *s)   //копируем данные из архивной записи, если не заполнены
				srv2.OldId = s.Id           //
				o.Update(&srv2)             //
				srv2.Id = s.Id              //
				o.Read(&srv2, "Id")         //
				srv2.Stat = 3               //архивная запись
				srv2.PowerState = 2         //
				srv2.Active = 2             //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

			} else { //оба сервера считаем архивными
				fmt.Println("off", srv.Id, vm.Id, srv.Time_Mfy.Format("01.02.2006"), vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Vm_Name, vm.Vm_HostName, vm.Vm_Uuid)
				fmt.Println("off", s.Id, vm2.Id, s.Time_Mfy.Format("01.02.2006"), vm2.Vcenter, vm2.Datacenter, vm2.Cluster, vm2.Vm_Name, vm2.Vm_HostName, vm2.Vm_Uuid)
				srv2 := models.ServerSprRecords{Id: srv.Id}
				o.Read(&srv2, "Id") //
				srv2.Stat = 3       //архивная запись
				srv2.PowerState = 2 //
				srv2.Active = 2     //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

				srv2.Id = s.Id
				o.Read(&srv2, "Id") //
				srv2.Stat = 3       //архивная запись
				srv2.PowerState = 2 //
				srv2.Active = 2     //выведен из эксплуатации
				srv2.TimeDeActiveState = time.Now().Local()
				o.Update(&srv2) //

			}
			continue
		} else {
			p := *srv
			srvlist[vm.Vcenter+"/"+vm.Vm_Uuid] = &p
			if okvm {
				p.PowerState = 1
			} else {
				p.PowerState = 2
			}
		}
	}
}
