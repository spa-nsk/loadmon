package cmdb

import (
	"errors"
	"fmt"
	models "loadmon/models"
	"loadmon/oneview"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ExtractServerNameILO предназначена для извлечения имени сервера из адреса iLO.
// В текущей реализации не выполняет действий (заглушка).
func ExtractServerNameILO(ilo string) {

}

func insertNewSprRecord(srv *oneview.ServerHardware) (int64, error) {
	srvspr := models.ServerSprRecords{}
	serverName := ""
	if strings.Contains(srv.Base.Name, srv.Base.SerialNumber.String()) {
		serverName = srv.Base.ServerName
	} else {
		if strings.Contains(srv.Base.Name, "ILO-") {
			temp := strings.Split(srv.Base.Name, "ILO-")
			if len(temp) == 2 {
				serverName = temp[1]
			} else {
				serverName = srv.Base.ServerName
			}
		} else if strings.Contains(srv.Base.Name, "ILO") {
			temp := strings.Split(srv.Base.Name, "ILO")
			if len(temp) == 2 {
				serverName = temp[1]
			} else {
				serverName = srv.Base.ServerName
			}
		} else if strings.Contains(srv.Base.Name, "ilo-") {
			temp := strings.Split(srv.Base.Name, "ilo-")
			if len(temp) == 2 {
				serverName = temp[1]
			} else {
				serverName = srv.Base.ServerName
			}
		} else if strings.Contains(srv.Base.Name, "ilo") {
			temp := strings.Split(srv.Base.Name, "ilo")
			if len(temp) == 2 {
				serverName = temp[1]
			} else {
				serverName = srv.Base.ServerName
			}
		} else if strings.Contains(srv.Base.Name, "iLO-") {
			temp := strings.Split(srv.Base.Name, "iLO-")
			if len(temp) == 2 {
				serverName = temp[1]
			} else {
				serverName = srv.Base.ServerName
			}
		} else if strings.Contains(srv.Base.Name, "iLO") {
			temp := strings.Split(srv.Base.Name, "iLO")
			if len(temp) == 2 {
				serverName = temp[1]
			} else {
				serverName = srv.Base.ServerName
			}
		} else {
			serverName = srv.Base.ServerName
		}
	}
	if serverName == "" {
		return 0, errors.New("Имя файла пустое")
	}
	o := orm.NewOrm()
	//o.Using("default")

	shortName := strings.Split(serverName, ".")[0]
	/*
		if srv.Base.ServerName != "" {
			srvspr.Name = srv.Base.ServerName
		} else {
			srvspr.Name = shortName
		}
	*/
	srvspr.Name = shortName
	srvspr.NameFQDN = serverName

	srvspr.NameFQDN = strings.TrimSpace(srvspr.NameFQDN)
	srvspr.Name = strings.TrimSpace(srvspr.Name)
	if srv.OperatingSystem != "" {
		srvspr.OS = srv.OperatingSystem
	}
	//
	srvspr.CPUCount = float64(srv.Base.ProcessorCount)
	srvspr.CPULogical = float64(srv.Base.ProcessorCount) * float64(srv.Base.ProcessorCoreCount)
	srvspr.MemSize = float64(srv.Base.MemoryMb) / 1024.0
	srvspr.Stat = 1
	srvspr.Details = srv.Base.SerialNumber.String() + " Добавлен из HP One, требуется правка"
	srvspr.Time_Mfy = time.Now().Local()

	if Id, err := o.Insert(&srvspr); err != nil {
		fmt.Println("Ошибка записи справочника серверов", err)
		return 0, err
	} else {
		return Id, nil
	}
}

func checkAndCorrect(records []*models.ServerSprRecords, serverName string, cpuCount int) int64 {
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

		if !strings.EqualFold(records[i].NameFQDN, serverName) || records[i].CPUCount != float64(cpuCount) {
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

// обновить справочник моделей производителя
func UpdateModelVendor(model *models.ModelVendor, srv *oneview.ServerHardware) int64 {
	if model == nil || srv == nil {
		return 0
	}
	m := models.ModelVendor{Model: srv.Base.Model}
	o := orm.NewOrm()
	err := o.Read(&m, "Model")
	if err == nil {
		*model = m
		return m.Code
	}
	pm := make([]*models.ModelVendor, 0)

	sql := "SELECT * FROM public.cmdb_model_vendor ORDER BY code DESC"
	qs := o.Raw(sql)
	num, err := qs.QueryRows(&pm)
	if err == nil && num > 0 {
		model.Code = pm[0].Code + 1
	} else {
		model.Code = 1
	}
	model.CategoryEquipment = 1 //сервер
	model.Model = srv.Base.Model
	model.SizeUnit = 1 //нужно указать реальные данные
	model.Vendor = "HPE"
	if strings.Contains(srv.Base.Name, "bay") {
		model.Type = 2 //лезвие
	} else {
		model.Type = 1 //обычный сервер
	}
	model.Id, err = o.Insert(model)
	if err != nil {
		return -1
	}
	return model.Code
}

// обновить справочник моделей-нормативов
func UpdateModelEquipment(norma *models.ModelEquipment, srv *oneview.ServerHardware, modelVendor int64) int64 {
	if norma == nil || srv == nil {
		return -1
	}

	diskMap := make(map[string]int64)

	for _, storage := range srv.Storage.Data {
		for _, disk := range storage.PhysicalDrives {
			rec := fmt.Sprintf("%.0fGb %s", disk.CapacityMiB/1000, disk.MediaType)
			count, ok := diskMap[rec]
			if ok {
				count++
			} else {
				count = 1
			}
			diskMap[rec] = count
		}
	}

	ssd := ""
	hdd := ""
	alternative := ""

	for disk, count := range diskMap {
		if strings.Contains(disk, "SSD") {
			ssd = fmt.Sprintf("%s %dx%s ", ssd, count, disk)
		} else if strings.Contains(disk, "HDD") {
			hdd = fmt.Sprintf("%s %dx%s ", hdd, count, disk)
		} else {
			alternative = fmt.Sprintf("%s %dx%s ", alternative, count, disk)
		}
	}

	key := fmt.Sprintf("%s %s %dx%s %dMb %s %s %s", srv.Base.Model, srv.Base.PartNumber, srv.Base.ProcessorCount, srv.Base.ProcessorType, srv.Base.MemoryMb, ssd, hdd, alternative)
	key = strings.TrimSpace(key)

	/*
		for _, storage := range srv.Storage.Data {
			for _, disk := range storage.PhysicalDrives {
				key += fmt.Sprintf("%.0fGb %s %s", disk.CapacityMiB/1000, disk.MediaType, disk.Model)
			}
		}
	*/

	nm := models.ModelEquipment{Name: key}

	o := orm.NewOrm()
	err := o.Read(&nm, "Name")
	if err == nil {
		*norma = nm
		return nm.Code
	}
	pm := make([]*models.ModelEquipment, 0)

	sql := "SELECT * FROM public.cmdb_model_equipment ORDER BY code DESC"
	qs := o.Raw(sql)
	num, err := qs.QueryRows(&pm)
	if err == nil && num > 0 {
		norma.Code = pm[0].Code + 1
	} else {
		norma.Code = 1
	}
	norma.ModelVendor = modelVendor
	norma.Name = key
	norma.PartNum = srv.Base.PartNumber
	norma.PartNumSale = ""
	norma.Money = 0
	norma.Currency = 0
	norma.MoneyInfo = ""
	norma.ChangeModelEquipment = 0
	norma.ChangeMoney = 0
	norma.ChangeCurrency = 0
	norma.ChangeMoneyInfo = ""
	norma.Detail = ""

	norma.Id, err = o.Insert(norma)
	if err != nil {
		return -1
	}
	return norma.Code
}

// обновить справочник процессоров
func UpdateCpuModel(model *models.ComponentsCPU, srv *oneview.ServerHardware) int64 {
	if model == nil || srv == nil {
		return 0
	}
	m := models.ComponentsCPU{ProcessorType: srv.Base.ProcessorType}
	o := orm.NewOrm()
	err := o.Read(&m, "ProcessorType")
	if err == nil {
		*model = m
		return m.Code
	}
	pm := make([]*models.ComponentsCPU, 0)

	sql := "SELECT * FROM cmdb_components_cpu ORDER BY code DESC"
	qs := o.Raw(sql)
	num, err := qs.QueryRows(&pm)
	if err == nil && num > 0 {
		model.Code = pm[0].Code + 1
	} else {
		model.Code = 1
	}
	model.ProcessorType = srv.Base.ProcessorType
	model.ProcessorCoreCount = srv.Base.ProcessorCoreCount
	model.ProcessorSpeedMhz = srv.Base.ProcessorSpeedMhz

	model.Id, err = o.Insert(model)
	if err != nil {
		return -1
	}
	return model.Code
}

// обновить справочник Цод
func UpdateDatacenter(dc *models.Datacenter, srv *oneview.ServerHardware) int64 {
	if dc == nil || srv == nil {
		return 0
	}
	if srv.Datacenter.Name == "" {
		return -2
	}
	m := models.Datacenter{Name: srv.Datacenter.Name}
	o := orm.NewOrm()
	err := o.Read(&m, "Name")
	if err == nil {
		*dc = m
		return m.Code
	}
	pm := make([]*models.Datacenter, 0)

	sql := "SELECT * FROM public.cmdb_datacenter ORDER BY code DESC"
	qs := o.Raw(sql)
	num, err := qs.QueryRows(&pm)
	if err == nil && num > 0 {
		dc.Code = pm[0].Code + 1
	} else {
		dc.Code = 1
	}
	dc.Width = int64(srv.Datacenter.Width)
	dc.Depth = int64(srv.Datacenter.Depth)
	dc.Name = srv.Datacenter.Name
	dc.Id, err = o.Insert(dc)
	if err != nil {
		return -1
	}
	return dc.Code
}

// обновить справочник Цод
func UpdateRack(rk *models.Rack, srv *oneview.ServerHardware) int64 {
	if rk == nil || srv == nil {
		return 0
	}
	if srv.Rack.Name == "" {
		return -2
	}
	m := models.Rack{Name: srv.Rack.Name}
	o := orm.NewOrm()
	err := o.Read(&m, "Name")
	if err == nil {
		*rk = m
		return m.Code
	}
	pm := make([]*models.Rack, 0)

	sql := "SELECT * FROM public.cmdb_rack ORDER BY code DESC"
	qs := o.Raw(sql)
	num, err := qs.QueryRows(&pm)
	if err == nil && num > 0 {
		rk.Code = pm[0].Code + 1
	} else {
		rk.Code = 1
	}
	rk.Width = int64(srv.Rack.Width)
	rk.Depth = int64(srv.Rack.Depth)
	rk.Height = int64(srv.Rack.Height)
	rk.UHeight = int64(srv.Rack.UHeight)
	rk.Name = srv.Rack.Name
	rk.Id, err = o.Insert(rk)
	if err != nil {
		return -1
	}
	return rk.Code
}

// UpdateRamModel ищет в справочнике моделей оперативной памяти (ComponentsRAM) запись,
// соответствующую характеристикам модуля памяти mem, и при отсутствии создаёт новую с
// очередным кодом модели. Возвращает код модели.
func UpdateRamModel(model *models.ComponentsRAM, srv *oneview.ServerHardware, mem *oneview.MemoryModule) int64 {
	if model == nil || srv == nil || mem == nil {
		return 0
	}
	key := fmt.Sprintf("%dGb %s %s %dMhz %s Rank Count %d", mem.CapacityMiB/1024, mem.BaseModuleType, mem.MemoryDeviceType, mem.OperatingSpeedMhz, strings.TrimSpace(mem.PartNumber), mem.RankCount)
	m := models.ComponentsRAM{Model: key}
	o := orm.NewOrm()
	err := o.Read(&m, "Model")
	if err == nil {
		*model = m
		return m.Code
	}
	pm := make([]*models.ComponentsRAM, 0)

	sql := "SELECT * FROM cmdb_components_ram ORDER BY code DESC"
	qs := o.Raw(sql)
	num, err := qs.QueryRows(&pm)
	if err == nil && num > 0 {
		model.Code = pm[0].Code + 1
	} else {
		model.Code = 1
	}

	model.Model = key
	model.CapacityMiB = mem.CapacityMiB
	model.BaseModuleType = mem.BaseModuleType
	model.MemoryDeviceType = mem.MemoryDeviceType
	model.OperatingSpeedMhz = mem.OperatingSpeedMhz
	model.Vendor = mem.Manufacturer
	model.PartNumber = mem.PartNumber
	model.PartNumberSale = ""
	model.RankCount = mem.RankCount
	model.Id, err = o.Insert(model)
	if err != nil {
		return -1
	}
	return model.Code
}

// UpdateCMDBHwSprHP обновляет справочники нормативных моделей оборудования (производитель,
// модель, ЦОД, стойка, процессор, память) в CMDB на основании данных всех серверов из
// HP OneView.
func UpdateCMDBHwSprHP() {
	fmt.Println("Обновление CMDB по данным из HP OneView")
	ov := oneview.GetGlobalOVInfrastructure()

	for _, srv := range ov.Servers { //проходим по списку всех серверов

		serverName := ""
		if strings.Contains(srv.Base.Name, srv.Base.SerialNumber.String()) {
			serverName = srv.Base.ServerName
		} else {
			if strings.Contains(srv.Base.Name, "ILO-") {
				temp := strings.Split(srv.Base.Name, "ILO-")
				if len(temp) == 2 {
					serverName = temp[1]
				} else {
					serverName = srv.Base.ServerName
				}
			} else if strings.Contains(srv.Base.Name, "ILO") {
				temp := strings.Split(srv.Base.Name, "ILO")
				if len(temp) == 2 {
					serverName = temp[1]
				} else {
					serverName = srv.Base.ServerName
				}
			} else if strings.Contains(srv.Base.Name, "ilo-") {
				temp := strings.Split(srv.Base.Name, "ilo-")
				if len(temp) == 2 {
					serverName = temp[1]
				} else {
					serverName = srv.Base.ServerName
				}
			} else if strings.Contains(srv.Base.Name, "ilo") {
				temp := strings.Split(srv.Base.Name, "ilo")
				if len(temp) == 2 {
					serverName = temp[1]
				} else {
					serverName = srv.Base.ServerName
				}
			} else if strings.Contains(srv.Base.Name, "iLO-") {
				temp := strings.Split(srv.Base.Name, "iLO-")
				if len(temp) == 2 {
					serverName = temp[1]
				} else {
					serverName = srv.Base.ServerName
				}
			} else if strings.Contains(srv.Base.Name, "iLO") {
				temp := strings.Split(srv.Base.Name, "iLO")
				if len(temp) == 2 {
					serverName = temp[1]
				} else {
					serverName = srv.Base.ServerName
				}
			} else {
				serverName = srv.Base.ServerName
			}
		}

		norm := models.ModelEquipment{}
		model := models.ModelVendor{}
		dc := models.Datacenter{}
		rk := models.Rack{}

		cu := models.ConfigurationUnit{SerialNum: srv.Base.SerialNumber.String()}
		//обновление и поиски модели производителя
		modelCode := UpdateModelVendor(&model, srv) //
		if modelCode < 0 {
			fmt.Println("Ошибка добавления модели оборудования: ", srv.Base.Model, "-", srv.Base.PartNumber)
			continue
		}
		//обновление и поиск модели-норматива
		modelEqCode := UpdateModelEquipment(&norm, srv, modelCode)
		if modelEqCode < 0 {
			fmt.Println("Ошибка добавления нормативной модели : ", srv.Base.Model, "-", srv.Base.PartNumber)
			continue
		}

		dcCode := UpdateDatacenter(&dc, srv)
		if modelEqCode < 0 {
			fmt.Println("Ошибка добавления датацентра : ", srv.Datacenter.Name)
		}

		rkCode := UpdateRack(&rk, srv)
		if modelEqCode < 0 {
			fmt.Println("Ошибка добавления шкафа : ", srv.Rack.Name)
		}

		id_me := int64(0)

		o := orm.NewOrm()
		//поиск конфигурационной единицы сервера
		err := o.Read(&cu, "SerialNum")

		cu.NameFQDN = serverName
		if dcCode > 0 {
			cu.Datacenter = dcCode
		}
		if rkCode > 0 {
			cu.Rack = rkCode
		}

		if err == nil { //найдена
			cu.ModelEquipment = modelEqCode
			o.Update(&cu)
			id_me = cu.Id
		} else { //заводим новую
			cu.ModelEquipment = modelEqCode
			id_me, err = o.Insert(&cu)
			if err != nil {
				fmt.Println("Ошибка добавления конфигурационной единицы: ", srv.Base.Model, "=", srv.Base.PartNumber)
			}
		}
		cpu := models.ComponentsCPU{}         //
		cpu_code := UpdateCpuModel(&cpu, srv) //поиск процессора в справочнике
		comp_cpu := models.Components{
			ConfigurationUnit: id_me,
			CategoryEquipment: 1,        //серверное
			Type:              1,        //процессор
			Component:         cpu_code} //модель процессора
		for i := 0; i < srv.Base.ProcessorCount; i++ {
			o.Insert(&comp_cpu) //записываем компоненты процессооры
		}

		for _, mem := range srv.Memory.Data { //проходим в цикле по всем модулям памяти
			ram := models.ComponentsRAM{}

			ram_code := UpdateRamModel(&ram, srv, &mem) //ищем модуль памяти в справочнике
			comp_ram := models.Components{
				ConfigurationUnit: id_me,
				CategoryEquipment: 1, //серверное
				Type:              2, //оперативная память
				Location:          mem.DeviceLocator,
				Component:         ram_code} //память из справочника
			//вставить запись компонент

			if _, err := o.Insert(&comp_ram); err != nil { //вставка записи о оперативной памяти
				fmt.Println("Ошибка записи данных о памяти", err)
			}
		}

	}
}

// проверить функцию или переписать
// oneview.GetHwserverNameFQDN(srv *ServerHardware) string
// oneview.GetHwserverName(srv *ServerHardware) string
// oneview.GetHwserverDomain(srv *ServerHardware) string
// cmdb.SplitNameFQDN(nameFQDN string) (string, string)

// UpdateHwSprHPBladeSystem синхронизирует справочник блейд-корзин (BladeSystems) и их
// компонентов с данными корзин, полученными из HP OneView.
func UpdateHwSprHPBladeSystem(ov *oneview.OVInfrastructure) {
	o := orm.NewOrm()
	for _, enc := range ov.AllEnclosureBySN { //проходим по списку всех корзин
		bls := models.BladeSystems{SerialNum: strings.TrimSpace(enc.SerialNumber)}

		if err := o.Read(&bls, "SerialNum"); err != nil { //ищем корзину в справочнике по серийному номеру
			//добавляем запись в справочник корзин, если она не нашлась
			bls.Name = enc.Name
			bls.RackNum = enc.RackName
			bls.SerialNum = enc.SerialNumber
			bls.PartNum = enc.PartNumber
			bls.Model = enc.EnclosureModel
			bls.Type = enc.EnclosureType
			bls.Size = enc.DeviceBayCount
			bls.Description = enc.Description.String()
			bls.Used = 0
			for _, bay := range enc.DeviceBays {
				if bay.DevicePresence == "Present" {
					bls.Used++
				}
			}
			bls.Time_Mfy = time.Now().Local()
			bls.Anotation = "Данные добавлены из HP One"
			if _, err := o.Insert(&bls); err != nil {
				fmt.Println("Ошибка записи данных в справочник корзин.", err)
			}

		} else { //корзина уже есть в справочнике
			//обновление записи в справочнике корзин
			bls.Name = enc.Name
			bls.RackNum = enc.RackName
			bls.SerialNum = enc.SerialNumber
			bls.PartNum = enc.PartNumber
			bls.Model = enc.EnclosureModel
			bls.Type = enc.EnclosureType
			bls.Size = enc.DeviceBayCount
			bls.Description = enc.Description.String()
			bls.Used = 0
			for _, bay := range enc.DeviceBays {
				if bay.DevicePresence == "Present" {
					bls.Used++
				}
			}

			if _, err := o.Update(&bls); err != nil {
				fmt.Println("Ошибка обновление данных конфигурации корзины", err)
			}
		}

		bls = models.BladeSystems{SerialNum: enc.SerialNumber}
		if err := o.Read(&bls, "SerialNum"); err == nil { //читаем запись о корзине по серийному номеру
			m := models.ServerHardwareComponents{BladeSystemId: bls.Id}
			if nDel, err := o.Delete(&m, "BladeSystemId"); err != nil { //удаляем записи о компонентах корзины из базы (для последующего обновления данных )
				fmt.Println("Ошибка удаления записей о компонентах корзины: ", bls.Name, err)
				_ = nDel
			}
		}

		//видимо ошибка в алгоритме компоненты уже удалены, а мы пытаемся искать interconnect похоже на дублирование на случай если удаление не прошло
		m := models.ServerHardwareComponents{BladeSystemId: bls.Id, Type: 6} //тип компонент 1 память, 2 диск, 3 сетевой контроллер, 4 FC контроллер, 5 RAID контроллер, 6 interconnect
		if err := o.Read(&m, "BladeSystemId", "Type"); err != nil {          //ничего не нашлось
			for _, inter := range enc.InterconnectBays { //проходим по всем interconnect в корзине
				if inter.InterconnectUri == "" { //пропускаем записи где нет ссылки на interconnect
					continue
				}
				/*
					if inter.InterconnectURI == "" {
						continue
					}
				*/
				iConnect := models.ServerHardwareComponents{ //формируем запись для вставки в базу
					BladeSystemId:   bls.Id,
					Type:            6, //interconnect
					Vendor:          bls.Vendor,
					Model:           inter.InterconnectModel,
					CapacityGb:      0,
					SerialNum:       inter.SerialNumber,
					Slot:            fmt.Sprintf("%d", inter.BayNumber),
					ContractDoc:     bls.ContractDoc,
					DateContractDoc: bls.DateContractDoc,
					Invoice:         bls.Invoice,
					DateInvoice:     bls.DateInvoice,
					ServiceContract: bls.ServiceContract,
					DateService:     bls.DateService,
					Time_Mfy:        time.Now().Local(),
					Id_User:         bls.Id_User}
				//вставить запись компонент
				if _, err := o.Insert(&iConnect); err != nil { //вставляем запись с информацией о interconnect
					fmt.Println("Ошибка записи данных интерконнекта", err)
				}
			}
		} else {
			//добавить обработку при наличие записей по компонентам корзины(если записи не удалились ранее)
		}

		m = models.ServerHardwareComponents{BladeSystemId: bls.Id, Type: 7}
		if err := o.Read(&m, "BladeSystemId", "Type"); err != nil { //пытаемся найти managment card blade
			for _, manager := range enc.ManagerBays {
				if manager.DevicePresence != "Present" {
					continue
				}
				iManager := models.ServerHardwareComponents{ //формируем запись managment card blade для вставки в базу
					BladeSystemId: bls.Id,
					Type:          7,
					Vendor:        bls.Vendor,
					//					Model:           fmt.Sprintf("Name: %-20s IP: %-20s Fw v%s %s", manager.FqdnHostName, manager.IPAddr
					Model:           fmt.Sprintf("Name: %-20s IP: %-20s Fw v%s %s", manager.Model, manager.IpAddress, manager.FwVersion, manager.FwBuildDate),
					CapacityGb:      0,
					SerialNum:       "",
					Slot:            fmt.Sprintf("%d", manager.BayNumber),
					ContractDoc:     bls.ContractDoc,
					DateContractDoc: bls.DateContractDoc,
					Invoice:         bls.Invoice,
					DateInvoice:     bls.DateInvoice,
					ServiceContract: bls.ServiceContract,
					DateService:     bls.DateService,
					Time_Mfy:        time.Now().Local(),
					Id_User:         bls.Id_User}
				//вставить запись компонент
				if _, err := o.Insert(&iManager); err != nil {
					fmt.Println("Ошибка записи данных о плате управления", err)
				}
			}
		} else {
			//добавить обработку при наличие записей по компонентам корзины
		}

		m = models.ServerHardwareComponents{BladeSystemId: bls.Id, Type: 8} //power supply
		if err := o.Read(&m, "BladeSystemId", "Type"); err != nil {
			for _, powerBay := range enc.PowerSupplyBays {
				if powerBay.DevicePresence != "Present" {
					continue
				}
				power := models.ServerHardwareComponents{ //формируем запись по блокам питания
					BladeSystemId:   bls.Id,
					Type:            8,
					Vendor:          bls.Vendor,
					Model:           powerBay.Model + " (" + powerBay.PartNumber + ") " + " " + powerBay.SparePartNumber + " " + powerBay.Status,
					CapacityGb:      0,
					SerialNum:       powerBay.SerialNumber,
					Slot:            fmt.Sprintf("%d", powerBay.BayNumber),
					ContractDoc:     bls.ContractDoc,
					DateContractDoc: bls.DateContractDoc,
					Invoice:         bls.Invoice,
					DateInvoice:     bls.DateInvoice,
					ServiceContract: bls.ServiceContract,
					DateService:     bls.DateService,
					Time_Mfy:        time.Now().Local(),
					Id_User:         bls.Id_User}
				//вставить запись компонент
				if _, err := o.Insert(&power); err != nil {
					fmt.Println("Ошибка записи данных о плате управления", err)
				}
			}
		} else {
			//добавить обработку при наличие записей по компонентам корзины
		}

		m = models.ServerHardwareComponents{BladeSystemId: bls.Id, Type: 9}
		if err := o.Read(&m, "BladeSystemId", "Type"); err != nil {
			for _, fanBay := range enc.FanBays {
				if fanBay.DevicePresence != "Present" {
					continue
				}
				fan := models.ServerHardwareComponents{
					BladeSystemId:   bls.Id,
					Type:            9,
					Vendor:          bls.Vendor,
					Model:           fanBay.Model + " (" + fanBay.PartNumber + ")" + " " + fanBay.SparePartNumber + " " + fanBay.Status,
					CapacityGb:      0,
					SerialNum:       "",
					Slot:            fmt.Sprintf("%d", fanBay.BayNumber),
					ContractDoc:     bls.ContractDoc,
					DateContractDoc: bls.DateContractDoc,
					Invoice:         bls.Invoice,
					DateInvoice:     bls.DateInvoice,
					ServiceContract: bls.ServiceContract,
					DateService:     bls.DateService,
					Time_Mfy:        time.Now().Local(),
					Id_User:         bls.Id_User}
				//вставить запись компонент
				if _, err := o.Insert(&fan); err != nil {
					fmt.Println("Ошибка записи данных о плате управления", err)
				}
			}
		} else {
			//добавить обработку при наличие записей по компонентам корзины
		}
	}
}

// UpdateHwSprHPServers синхронизирует справочники конфигураций оборудования
// (ServerHardware) и серверов (ServerSprRecords) с данными всех серверов, полученными
// из HP OneView.
func UpdateHwSprHPServers(ov *oneview.OVInfrastructure) {
	o := orm.NewOrm()
	for _, srv := range ov.Servers { //проходим по списку всех серверов

		hw := models.ServerHardware{SerialNum: string(srv.Base.SerialNumber)}
		srvspr := models.ServerSprRecords{}
		foundHwBySN := false
		foundSrvSpr := false
		hwId := int64(0)     //создание переменных
		serverId := int64(0) //

		trn, err := o.Begin()
		if err != nil {
			fmt.Println("Ошибка начала транзакции ", err)
			return
		}
		err = trn.Read(&hw, "SerialNum")
		if err == nil {
			foundHwBySN = true
		}

		ImportHWSprRecord(&hw, srv, ov) //заполняем данные в models.ServerHardware
		if foundHwBySN {
			//                 srvspr.Id = hw.ServerId
			//
			//                 err := trn.Read(&srvspr, "Id")
			//                 if err == nil {
			//                     foundSrvSpr = true
			//                     hw.ServerId = srvspr.Id
			//                     serverId = srvspr.Id
			//                 }
			srvspr.HWSprId = hw.Id

			err := trn.Read(&srvspr, "HWSprId")
			if err == nil {
				foundSrvSpr = true
				hw.ServerId = srvspr.Id
				serverId = srvspr.Id
			}
		}

		ImportServerSprFromHW(&srvspr, srv) //заполняем данные в models.ServerSprRecords
		srvspr.Name, _ = SplitNameFQDN(oneview.GetHWserverName(srv))
		srvspr.NameFQDN = oneview.GetHWserverNameFQDN(srv)
		serverName := srvspr.NameFQDN

		fqdn, ip, errn := GetFQDNNameAndIP(srvspr.NameFQDN)
		if errn == nil {
			if srvspr.Type != 2 {
				srvspr.IPAddress = ip
			}
			srvspr.NameFQDN = fqdn
			serverName = fqdn
			srvspr.Name, _ = SplitNameFQDN(fqdn)
		}

		if foundHwBySN {
			_, err = trn.Update(&hw)
			hwId = hw.Id
			if err != nil {
				fmt.Println("Ошибка обновления записи в ServerHardware", serverName, err)
				trn.Rollback()
				continue
			}
		} else {
			hwId, err = trn.Insert(&hw)
			if err != nil {
				fmt.Println("Ошибка вставки записи в ServerHardware", serverName, err)
				trn.Rollback()
				continue
			}
		}

		srvspr.HWSprId = hwId

		if foundSrvSpr {
			srvspr.Stat = 1
			_, err = trn.Update(&srvspr)
			if err != nil {
				fmt.Println("Ошибка обновления записи в ServerSprRecords", serverName, err)
				trn.Rollback()
				continue
			}
		} else {
			srvspr.Stat = 1
			serverId, err = trn.Insert(&srvspr)
			if err != nil {
				fmt.Println("Ошибка обновления записи в ServerSprRecords", serverName, err)
				trn.Rollback()
				continue
			}
		}
		err = trn.Read(&hw, "Id")
		if err != nil {
			fmt.Println("Ошибка чтения ServerHardware по id ", hw.Id, err)
			trn.Rollback()
			continue
		}

		hw.ServerId = serverId
		_, err = trn.Update(&hw)
		if err != nil {
			fmt.Println("Ошибка обновления записи в ServerHardware", serverName, err)
			trn.Rollback()
			continue
		}

		m := models.ServerHardwareComponents{ServerId: hw.Id, Type: 1}
		if nDel, err := trn.Delete(&m, "ServerId", "Type"); err != nil { //удаляем записи о оперативной памяти из базы (для последующего обновления данных )
			fmt.Println("Ошибка удаления записей о оперативной памяти сервера: ", serverName, hw.SerialNum, err)
			_ = nDel
		}

		m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 1} //проверяем остались ли данные по памяти
		if err := trn.Read(&m, "ServerId", "Type"); err != nil {      //нет записей по памяти
			for idx, mem := range srv.Memory.Data { //проходим в цикле по всем модулям памяти
				ram := models.ServerHardwareComponents{ //формируем данные для записи по памяти
					ServerId:        hw.Id,
					Type:            1,
					Vendor:          mem.Manufacturer,
					Model:           fmt.Sprintf("%.0fGb", float64(mem.CapacityMiB)/1024.0) + " " + mem.BaseModuleType + " " + mem.MemoryDeviceType + " " + strconv.Itoa(mem.OperatingSpeedMhz),
					CapacityGb:      float64(mem.CapacityMiB) / 1024.0,
					SerialNum:       hw.SerialNum + "-" + strconv.Itoa(idx),
					Slot:            mem.Name,
					ContractDoc:     hw.ContractDoc,
					DateContractDoc: hw.DateContractDoc,
					Invoice:         hw.Invoice,
					DateInvoice:     hw.DateInvoice,
					ServiceContract: hw.ServiceContract,
					DateService:     hw.DateService,
					Time_Mfy:        hw.Time_Mfy,
					Id_User:         hw.Id_User}
				//вставить запись компонент

				if _, err := trn.Insert(&ram); err != nil { //вставка записи о оперативной памяти
					fmt.Println("Ошибка записи данных о памяти", err)
				}
			}
		}

		m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 2}
		if nDel, err := trn.Delete(&m, "ServerId", "Type"); err != nil { //удаляем записи о дисках из базы (для последующего обновления данных )
			fmt.Println("Ошибка удаления записей о дисках сервера: ", serverName, hw.SerialNum, err)
			_ = nDel
		}

		m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 5}
		if nDel, err := trn.Delete(&m, "ServerId", "Type"); err != nil { //удаляем записи о контроллерах дисков из базы (для последующего обновления данных )
			fmt.Println("Ошибка удаления записей о дисках сервера: ", serverName, hw.SerialNum, err)
			_ = nDel
		}

		m = models.ServerHardwareComponents{ServerId: hw.Id, Type: 2}
		if err := trn.Read(&m, "ServerId", "Type"); err != nil { //проверяем есть ли записи по этому серверу
			for _, storage := range srv.Storage.Data { //записей нет, записываем данные по физическим дискам
				raid := models.ServerHardwareComponents{
					ServerId:        hw.Id,
					Type:            5,
					Vendor:          "HPE",
					Model:           storage.Model,
					CapacityGb:      float64(storage.CacheMemorySizeMiB) / 1024.0,
					SerialNum:       strings.TrimSpace(storage.SerialNumber),
					Slot:            storage.Location, //формат строки расположения "ControllerPort:Box:Bay"
					ContractDoc:     hw.ContractDoc,
					DateContractDoc: hw.DateContractDoc,
					Invoice:         hw.Invoice,
					DateInvoice:     hw.DateInvoice,
					ServiceContract: hw.ServiceContract,
					DateService:     hw.DateService,
					Time_Mfy:        hw.Time_Mfy,
					Id_User:         hw.Id_User}
				if str := strings.TrimSpace(storage.CacheModuleSerialNumber); str != "" {
					raid.SerialNum += " CacheModule " + str
				}
				//вставить запись компонент
				if _, err := trn.Insert(&raid); err != nil {
					fmt.Println("Ошибка записи данных о диске", err)
				}

				for _, disk := range storage.PhysicalDrives {
					dsk := models.ServerHardwareComponents{
						ServerId:        hw.Id,
						Type:            2,
						Vendor:          "HPE",
						Model:           fmt.Sprintf("%.0fGb", float64(disk.CapacityMiB)/1024.0) + " " + disk.MediaType + " " + strings.TrimSpace(disk.Model) + " (" + strings.TrimSpace(disk.SerialNumber) + ")",
						CapacityGb:      float64(disk.CapacityMiB) / 1000.0,
						SerialNum:       strings.TrimSpace(disk.SerialNumber),
						Slot:            disk.Location, //формат строки расположения "ControllerPort:Box:Bay"
						ContractDoc:     hw.ContractDoc,
						DateContractDoc: hw.DateContractDoc,
						Invoice:         hw.Invoice,
						DateInvoice:     hw.DateInvoice,
						ServiceContract: hw.ServiceContract,
						DateService:     hw.DateService,
						Time_Mfy:        hw.Time_Mfy,
						Id_User:         hw.Id_User}
					switch disk.MediaType {
					case "HDD":
						dsk.SubType = 2
					case "SSD":
						dsk.SubType = 1
					}
					//вставить запись компонент
					if _, err := trn.Insert(&dsk); err != nil {
						fmt.Println("Ошибка записи данных о диске", err)
					}
				}
			}
		}
		trn.Commit()
	}
}

// UpdateHwSprHP выполняет полное обновление справочника оборудования CMDB по данным
// HP OneView: последовательно синхронизирует блейд-корзины и серверы.
func UpdateHwSprHP() {
	fmt.Println("Обновление справочника оборудования по данным из HP OneView")
	ov := oneview.GetGlobalOVInfrastructure()
	UpdateHwSprHPBladeSystem(ov)
	UpdateHwSprHPServers(ov)
	fmt.Println("Обновление справочника оборудования завершено")
}
