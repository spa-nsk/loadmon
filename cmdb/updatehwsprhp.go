package cmdb

import (
	"errors"
	"fmt"
	models "loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// ExtractServerNameILO предназначена для извлечения имени сервера из адреса iLO.
// В текущей реализации не выполняет действий (заглушка).
func ExtractServerNameILO(ilo string) {

}

func insertNewSprRecord(srv *oneview.Server) (int64, error) {
	srvspr := models.ServerSprRecords{}
	// Примечание про oneview2: разбор служебного имени (Identity.Name) для
	// получения "человекочитаемого" имени сервера теперь инкапсулирован в
	// oneview.GetHWserverName - используем его вместо повторения логики.
	serverName := oneview.GetHWserverName(srv)
	if serverName == "" {
		return 0, errors.New("Имя файла пустое")
	}
	o := orm.NewOrm()
	//o.Using("default")

	shortName := strings.Split(serverName, ".")[0]
	srvspr.Name = shortName
	srvspr.NameFQDN = serverName

	srvspr.NameFQDN = strings.TrimSpace(srvspr.NameFQDN)
	srvspr.Name = strings.TrimSpace(srvspr.Name)
	if srv.Identity.OperatingSystem != "" {
		srvspr.OS = srv.Identity.OperatingSystem
	}
	//
	srvspr.CPUCount = float64(len(srv.Processors))
	if len(srv.Processors) > 0 {
		srvspr.CPULogical = float64(len(srv.Processors)) * float64(srv.Processors[0].TotalCores)
	}
	srvspr.MemSize = float64(oneview.TotalMemoryMb(srv)) / 1024.0
	srvspr.Stat = 1
	srvspr.Details = srv.Identity.SerialNumber + " Добавлен из HP One, требуется правка"
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
func UpdateModelVendor(model *models.ModelVendor, srv *oneview.Server) int64 {
	if model == nil || srv == nil {
		return 0
	}
	m := models.ModelVendor{Model: srv.Identity.Model}
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
	model.Model = srv.Identity.Model
	model.SizeUnit = 1 //нужно указать реальные данные
	model.Vendor = "HPE"
	if strings.Contains(srv.Identity.Name, "bay") {
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
func UpdateModelEquipment(norma *models.ModelEquipment, srv *oneview.Server, modelVendor int64) int64 {
	if norma == nil || srv == nil {
		return -1
	}

	diskMap := make(map[string]int64)

	for _, storage := range srv.Controllers {
		for _, disk := range storage.Drives {
			rec := fmt.Sprintf("%.0fGb %s", float64(disk.CapacityMiB)/1000, disk.MediaType)
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

	cpuModel := ""
	if len(srv.Processors) > 0 {
		cpuModel = srv.Processors[0].Model
	}
	key := fmt.Sprintf("%s %s %dx%s %dMb %s %s %s", srv.Identity.Model, srv.Identity.PartNumber, len(srv.Processors), cpuModel, oneview.TotalMemoryMb(srv), ssd, hdd, alternative)
	key = strings.TrimSpace(key)

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
	norma.PartNum = srv.Identity.PartNumber
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
func UpdateCpuModel(model *models.ComponentsCPU, srv *oneview.Server) int64 {
	if model == nil || srv == nil || len(srv.Processors) == 0 {
		return 0
	}
	p := srv.Processors[0]
	m := models.ComponentsCPU{ProcessorType: p.Model}
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
	model.ProcessorType = p.Model
	model.ProcessorCoreCount = p.TotalCores
	speed := p.OperatingSpeedMHz
	if speed == 0 {
		speed = p.MaxSpeedMHz
	}
	model.ProcessorSpeedMhz = speed

	model.Id, err = o.Insert(model)
	if err != nil {
		return -1
	}
	return model.Code
}

// обновить справочник Цод
func UpdateDatacenter(dc *models.Datacenter, srv *oneview.Server) int64 {
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
	// Примечание про oneview2: DatacenterInfo не хранит Depth (в отличие от
	// старого пакета) - оставляем как есть (нулевым), если запись новая.
	dc.Name = srv.Datacenter.Name
	dc.Id, err = o.Insert(dc)
	if err != nil {
		return -1
	}
	return dc.Code
}

// обновить справочник Цод
func UpdateRack(rk *models.Rack, srv *oneview.Server) int64 {
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
	// Примечание про oneview2: RackInfo не хранит Width/Depth/Height (в
	// отличие от старого пакета) - есть только UHeight, поэтому rk.Width/
	// rk.Depth/rk.Height остаются нулевыми для новых записей.
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
func UpdateRamModel(model *models.ComponentsRAM, srv *oneview.Server, mem *oneview.MemoryModule) int64 {
	if model == nil || srv == nil || mem == nil {
		return 0
	}
	key := fmt.Sprintf("%dGb %s %s %dMhz %s Rank Count %d", mem.CapacityMiB/1024, mem.BaseModuleType, mem.MemoryDeviceType, mem.OperatingSpeedMHz, strings.TrimSpace(mem.PartNumber), mem.RankCount)
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
	model.OperatingSpeedMhz = mem.OperatingSpeedMHz
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

		// Примечание про oneview2: разбор служебного имени вынесен в
		// oneview.GetHWserverName (см. также insertNewSprRecord выше).
		serverName := oneview.GetHWserverName(srv)

		norm := models.ModelEquipment{}
		model := models.ModelVendor{}
		dc := models.Datacenter{}
		rk := models.Rack{}

		cu := models.ConfigurationUnit{SerialNum: srv.Identity.SerialNumber}
		//обновление и поиски модели производителя
		modelCode := UpdateModelVendor(&model, srv) //
		if modelCode < 0 {
			fmt.Println("Ошибка добавления модели оборудования: ", srv.Identity.Model, "-", srv.Identity.PartNumber)
			continue
		}
		//обновление и поиск модели-норматива
		modelEqCode := UpdateModelEquipment(&norm, srv, modelCode)
		if modelEqCode < 0 {
			fmt.Println("Ошибка добавления нормативной модели : ", srv.Identity.Model, "-", srv.Identity.PartNumber)
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
				fmt.Println("Ошибка добавления конфигурационной единицы: ", srv.Identity.Model, "=", srv.Identity.PartNumber)
			}
		}
		cpu := models.ComponentsCPU{}         //
		cpu_code := UpdateCpuModel(&cpu, srv) //поиск процессора в справочнике
		comp_cpu := models.Components{
			ConfigurationUnit: id_me,
			CategoryEquipment: 1,        //серверное
			Type:              1,        //процессор
			Component:         cpu_code} //модель процессора
		for i := 0; i < len(srv.Processors); i++ {
			o.Insert(&comp_cpu) //записываем компоненты процессооры
		}

		for _, mem := range srv.Memory { //проходим в цикле по всем модулям памяти
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
//
// Примечание про oneview2: старая версия строила этот справочник по глобальному
// списку ov.AllEnclosureBySN, где каждая корзина (Enclosure) содержала типизированные
// списки DeviceBays/InterconnectBays/ManagerBays/PowerSupplyBays/FanBays с детальными
// полями (модель, серийный номер, прошивка и т.д.). В oneview2 такого глобального
// индекса корзин нет вовсе (см. loadmon/oneview2/global.go - Infrastructure хранит
// только серверы), а низкоуровневый клиентский API (loadmon/oneview2/resources.go,
// Client.ListEnclosures) возвращает Enclosure, в котором эти списки - это
// []map[string]any (сырой JSON), без типизированных полей вроде InterconnectUri/
// FqdnHostName/BayNumber. Восстановление этой функциональности потребовало бы
// самостоятельного разбора сырого JSON по каждому типу отсека - отдельная задача,
// выходящая за рамки миграции на новые типы. Пока функция ничего не делает; данные
// по компонентам блейд-корзин (interconnect/менеджмент/питание/охлаждение) в CMDB
// не обновляются.
func UpdateHwSprHPBladeSystem(ov *oneview.Infrastructure) {
	_ = ov
}

// UpdateHwSprHPServers синхронизирует справочники конфигураций оборудования
// (ServerHardware) и серверов (ServerSprRecords) с данными всех серверов, полученными
// из HP OneView.
func UpdateHwSprHPServers(ov *oneview.Infrastructure) {
	o := orm.NewOrm()
	for _, srv := range ov.Servers { //проходим по списку всех серверов

		hw := models.ServerHardware{SerialNum: srv.Identity.SerialNumber}
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
		if srvspr.Name != "" && srvspr.NameFQDN != "" { //если имя сервера не определено то не обновляем и не создаём новый
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
			for idx, mem := range srv.Memory { //проходим в цикле по всем модулям памяти
				ram := models.ServerHardwareComponents{ //формируем данные для записи по памяти
					ServerId:        hw.Id,
					Type:            1,
					Vendor:          mem.Manufacturer,
					Model:           fmt.Sprintf("%.0fGb", float64(mem.CapacityMiB)/1024.0) + " " + mem.BaseModuleType + " " + mem.MemoryDeviceType + " " + strconv.Itoa(mem.OperatingSpeedMHz),
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
			for _, storage := range srv.Controllers { //записей нет, записываем данные по физическим дискам
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
				// Примечание про oneview2: у StorageController нет отдельного
				// серийного номера кэш-модуля (в отличие от старого
				// CacheModuleSerialNumber).
				//вставить запись компонент
				if _, err := trn.Insert(&raid); err != nil {
					fmt.Println("Ошибка записи данных о диске", err)
				}

				for _, disk := range storage.Drives {
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
