package cmdb

import (
	"errors"
	"fmt"
	models "loadmon/models"
	vsphere "loadmon/vspheremetrics"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// ExtractServerNameILO предназначена для извлечения имени сервера из адреса iLO.
// В текущей реализации не выполняет действий (заглушка).
func ExtractServerNameILO(ilo string) {

}

// isDatabaseLockedErr определяет, является ли ошибка временной блокировкой
// sqlite (SQLITE_BUSY, драйвер mattn/go-sqlite3 возвращает текст "database
// is locked") - в отличие от прочих ошибок ORM, такую операцию имеет смысл
// повторить через небольшую паузу, а не сразу считать сбоем записи.
func isDatabaseLockedErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "database is busy")
}

// retryOnDatabaseLocked повторяет fn до maxAttempts раз с небольшой паузой
// между попытками, если очередная ошибка - "database is locked" (sqlite
// SQLITE_BUSY). Заведено по жалобе на массовые "Ошибка обновления записи в
// ServerHardware ... database is locked" при синхронизации с HP OneView -
// несколько независимых горутин в metrics_grpc/metrics_grpc.go пишут в один
// sqlite-файл одновременно (см. config/dbinit.go - там же добавлен
// _busy_timeout и ограничение пула соединений до 1, это первый уровень
// защиты; retryOnDatabaseLocked - второй, на случай, если пауза
// busy_timeout всё равно будет исчерпана под нагрузкой). Возвращает
// последнюю полученную ошибку, если все попытки исчерпаны.
func retryOnDatabaseLocked(maxAttempts int, fn func() error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = fn()
		if err == nil || !isDatabaseLockedErr(err) {
			return err
		}
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}
	}
	return err
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

	// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
	// (10003 - HPE OneView) для новой записи ServerSprRecords.
	srvspr.TimeCreate = time.Now().Local()
	srvspr.CreatorUser = 10003

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

// ⚠️ УДАЛЕНО (2026-09-06): UpdateCMDBHwSprHP - обновляла старый бюджетный
// нормативный контур (ModelEquipment/ModelVendor/ConfigurationUnit/
// Components + models.Datacenter/models.Rack, ныне тоже удалённые) на
// основании данных из HP OneView. Функция нигде не вызывалась (мёртвый
// код), а её единственные потребители Datacenter/Rack теперь наполняются
// из Итиллиум, а не из OneView (см. itilliumimport/importrackservers.go) -
// см. историю удаления в CHANGELOG.md. UpdateModelVendor/
// UpdateModelEquipment/UpdateCpuModel/UpdateRamModel оставлены нетронутыми
// (не входят в область этой правки), но с удалением UpdateCMDBHwSprHP
// вызывающих их мест больше не осталось.

// проверить функцию или переписать
// oneview.GetHwserverNameFQDN(srv *ServerHardware) string
// oneview.GetHwserverName(srv *ServerHardware) string
// oneview.GetHwserverDomain(srv *ServerHardware) string
// cmdb.SplitNameFQDN(nameFQDN string) (string, string)

// UpdateHwSprHPBladeSystem синхронизирует справочник блейд-корзин (BladeSystems) и их
// компонентов с данными корзин, полученными из HP OneView.
//
// ИСПРАВЛЕНО 2026-09-04 (была пустым no-op стабом после миграции на
// oneview2, см. историю этого комментария в git/CHANGELOG) - реализация
// вынесена в cmdb/syncbladesystemcomponents.go:SyncBladeSystemsFromOneView
// (отдельный файл, так как потребовалось собственное подключение к HP
// OneView через низкоуровневый Client.ListEnclosures - Infrastructure сама
// по себе хранит только серверы, не корзины целиком). См. подробный
// комментарий там же, включая явное предупреждение о неподтверждённых
// именах ключей сырых JSON-полей interconnect/manager/power/fan bay.
func UpdateHwSprHPBladeSystem(ov *oneview.Infrastructure) {
	SyncBladeSystemsFromOneView(ov)
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

		ImportHWSprRecord(trn, &hw, srv, ov) //заполняем данные в models.ServerHardware
		hw.SourceUpdateType = models.SourceUpdateOneView
		// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
		// (10003 - HPE OneView) для новой записи ServerHardware (если foundHwBySN
		// окажется false и ниже сработает Insert, а не Update).
		hw.TimeCreate = time.Now().Local()
		hw.CreatorUser = 10003
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
		srvspr.SourceUpdateType = models.SourceUpdateOneView
		// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
		// (10003 - HPE OneView) для новой записи ServerSprRecords (если ниже
		// сработает Insert, а не Update).
		srvspr.TimeCreate = time.Now().Local()
		srvspr.CreatorUser = 10003
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
			err = retryOnDatabaseLocked(5, func() error {
				_, e := trn.Update(&hw)
				return e
			})
			hwId = hw.Id
			if err != nil {
				fmt.Println("Ошибка обновления записи в ServerHardware", serverName, err)
				trn.Rollback()
				continue
			}
		} else {
			err = retryOnDatabaseLocked(5, func() error {
				var e error
				hwId, e = trn.Insert(&hw)
				return e
			})
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
				err = retryOnDatabaseLocked(5, func() error {
					_, e := trn.Update(&srvspr)
					return e
				})
				if err != nil {
					fmt.Println("Ошибка обновления записи в ServerSprRecords", serverName, err)
					trn.Rollback()
					continue
				}
			} else {
				srvspr.Stat = 1
				err = retryOnDatabaseLocked(5, func() error {
					var e error
					serverId, e = trn.Insert(&srvspr)
					return e
				})
				if err != nil {
					fmt.Println("Ошибка обновления записи в ServerSprRecords", serverName, err)
					trn.Rollback()
					continue
				}
			}
		} else {
			// Имя сервера не определено (oneview.GetHWserverName/GetHWserverNameFQDN
			// возвращают "" - как правило, у только что добавленного в OneView
			// оборудования, на которое ещё не установлена ОС: srv.Identity.ServerName
			// у него пусто, а особые случаи "имя через серийный номер"/"ILO-префикс"
			// не сработали). Раньше это молча пропускалось без единой строки в
			// логах - запись ServerHardware для сервера создавалась/обновлялась
			// (см. выше), а связанная ServerSprRecords - нет, из-за чего новый
			// физический сервер не появлялся в справочнике серверов и это было
			// не отличить от прочих временных сетевых сбоев. См.
			// kontext/05-conventions-and-gotchas.md.
			fmt.Println("Пропуск ServerSprRecords: не удалось определить имя сервера из OneView (S/N",
				hw.SerialNum, ", вероятно, только что добавлен и ещё не имеет ОС/hostname)")
		}
		err = trn.Read(&hw, "Id")
		if err != nil {
			fmt.Println("Ошибка чтения ServerHardware по id ", hw.Id, err)
			trn.Rollback()
			continue
		}

		hw.ServerId = serverId
		err = retryOnDatabaseLocked(5, func() error {
			_, e := trn.Update(&hw)
			return e
		})
		if err != nil {
			fmt.Println("Ошибка обновления записи в ServerHardware", serverName, err)
			trn.Rollback()
			continue
		}

		// Полная пересборка всех компонентов сервера - вынесена в
		// cmdb/syncserverhardwarecomponents.go (2026-09-04), см. комментарий
		// там.
		SyncServerHardwareComponents(trn, &hw, srv)
		trn.Commit()
	}
}

// UpdateHwSprHPServersInfra — самостоятельная переписанная версия
// UpdateHwSprHPServers для использования в cmdb.UpdateServerInventory:
// принимает снапшот инфраструктуры vSphere (infra) явным параметром и
// передаёт его в ImportServerSprFromHWInfra вместо того, чтобы та брала
// его самостоятельно на каждой итерации цикла по ov.Servers через
// vsphere.GetGlobalVMInfrastructure().
//
// Зачем: при последовательном вызове HP OneView-стороны синхронизации
// (эта функция) и VMware-стороны (cmdb.UpdateHWHostInfoInfra) в рамках
// ОДНОГО цикла cmdb.UpdateServerInventory важно, чтобы обе стороны видели
// ОДНО и то же согласованное состояние vSphere - иначе решение о демоции
// роли гипервизора, принятое здесь по одному снапшоту, и последующее
// решение UpdateHWHostInfoInfra о промоции/подтверждении роли, принятое по
// другому (более позднему) снапшоту, могут разойтись в рамках одного и
// того же цикла синхронизации (подробности - в комментарии к
// ImportServerSprFromHWInfra, cmdb/initrec.go).
//
// UpdateHwSprHPServers выше НЕ вызывает эту функцию и не вызывается ею -
// это две независимые функции с одинаковой логикой (кроме единственного
// отличия - источника снапшота vSphere), чтобы существующий вызов
// UpdateHwSprHPServers(ov) из UpdateHwSprHP() (metrics_grpc/metrics_grpc.go)
// остался абсолютно нетронутым.
func UpdateHwSprHPServersInfra(ov *oneview.Infrastructure, infra *vsphere.VMInfrastructure) {
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

		ImportHWSprRecord(trn, &hw, srv, ov) //заполняем данные в models.ServerHardware
		hw.SourceUpdateType = models.SourceUpdateOneView
		// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
		// (10003 - HPE OneView) для новой записи ServerHardware (если foundHwBySN
		// окажется false и ниже сработает Insert, а не Update).
		hw.TimeCreate = time.Now().Local()
		hw.CreatorUser = 10003
		if foundHwBySN {
			srvspr.HWSprId = hw.Id

			err := trn.Read(&srvspr, "HWSprId")
			if err == nil {
				foundSrvSpr = true
				hw.ServerId = srvspr.Id
				serverId = srvspr.Id
			}
		}

		ImportServerSprFromHWInfra(&srvspr, srv, infra) //заполняем данные в models.ServerSprRecords, снапшот vSphere передан явным параметром
		srvspr.SourceUpdateType = models.SourceUpdateOneView
		// 2026-09-14, по запросу пользователя - фиксируем время создания и источник
		// (10003 - HPE OneView) для новой записи ServerSprRecords (если ниже
		// сработает Insert, а не Update).
		srvspr.TimeCreate = time.Now().Local()
		srvspr.CreatorUser = 10003
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
			err = retryOnDatabaseLocked(5, func() error {
				_, e := trn.Update(&hw)
				return e
			})
			hwId = hw.Id
			if err != nil {
				fmt.Println("Ошибка обновления записи в ServerHardware", serverName, err)
				trn.Rollback()
				continue
			}
		} else {
			err = retryOnDatabaseLocked(5, func() error {
				var e error
				hwId, e = trn.Insert(&hw)
				return e
			})
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
				err = retryOnDatabaseLocked(5, func() error {
					_, e := trn.Update(&srvspr)
					return e
				})
				if err != nil {
					fmt.Println("Ошибка обновления записи в ServerSprRecords", serverName, err)
					trn.Rollback()
					continue
				}
			} else {
				srvspr.Stat = 1
				err = retryOnDatabaseLocked(5, func() error {
					var e error
					serverId, e = trn.Insert(&srvspr)
					return e
				})
				if err != nil {
					fmt.Println("Ошибка обновления записи в ServerSprRecords", serverName, err)
					trn.Rollback()
					continue
				}
			}
		} else {
			// Имя сервера не определено (oneview.GetHWserverName/GetHWserverNameFQDN
			// возвращают "" - как правило, у только что добавленного в OneView
			// оборудования, на которое ещё не установлена ОС: srv.Identity.ServerName
			// у него пусто, а особые случаи "имя через серийный номер"/"ILO-префикс"
			// не сработали). Раньше это молча пропускалось без единой строки в
			// логах - запись ServerHardware для сервера создавалась/обновлялась
			// (см. выше), а связанная ServerSprRecords - нет, из-за чего новый
			// физический сервер не появлялся в справочнике серверов и это было
			// не отличить от прочих временных сетевых сбоев. См.
			// kontext/05-conventions-and-gotchas.md.
			fmt.Println("Пропуск ServerSprRecords: не удалось определить имя сервера из OneView (S/N",
				hw.SerialNum, ", вероятно, только что добавлен и ещё не имеет ОС/hostname)")
		}
		err = trn.Read(&hw, "Id")
		if err != nil {
			fmt.Println("Ошибка чтения ServerHardware по id ", hw.Id, err)
			trn.Rollback()
			continue
		}

		hw.ServerId = serverId
		err = retryOnDatabaseLocked(5, func() error {
			_, e := trn.Update(&hw)
			return e
		})
		if err != nil {
			fmt.Println("Ошибка обновления записи в ServerHardware", serverName, err)
			trn.Rollback()
			continue
		}

		// Полная пересборка всех компонентов сервера - вынесена в
		// cmdb/syncserverhardwarecomponents.go (2026-09-04), см. комментарий
		// там.
		SyncServerHardwareComponents(trn, &hw, srv)
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
