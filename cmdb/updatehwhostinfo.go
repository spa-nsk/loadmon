package cmdb

import (
	"fmt"
	models "loadmon/models"
	_ "loadmon/routers"
	vsphere "loadmon/vspheremetrics"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	oneview "github.com/spa-nsk/oneview2"
)

// resolveHWBySerial подбирает "настоящий" серийный номер оборудования для
// хоста vSphere hs и, если удалось найти соответствие в HP OneView, саму
// запись *oneview.Server. У некоторых blade-хостов ESXi видит не тот же
// серийный номер, что OneView (см. поле VirtualSerialNum в
// models.ServerHardware) - поэтому серийник ESXi сначала ищется среди
// "виртуальных" серийников OneView (ov.ServerByVirtualSN), и только если
// там не нашлось - используется как есть, в надежде что это уже "настоящий"
// серийник, известный OneView напрямую.
func resolveHWBySerial(ov *oneview.Infrastructure, hostSerial string) (serialNum string, hw *oneview.Server) {
	if h, ok := ov.ServerByVirtualSN[hostSerial]; ok {
		return h.Identity.SerialNumber, h
	}
	if h, ok := ov.ServerBySN[hostSerial]; ok {
		return h.Identity.SerialNumber, h
	}
	return hostSerial, nil
}

// insertNewHostSprRecord создаёт новую запись справочника серверов
// (models.ServerSprRecords) для хоста vSphere hs, не найденного ни по имени,
// ни по серийному номеру среди уже существующих записей. Запись сразу
// создаётся с корректными Type/SerialNumber/HWSprId/Vcenter/Datacenter/
// Cluster/ClusterId - раньше (см. историю этого файла) эти поля выставлял
// только последующий блок в UpdateHWHostInfo, а здесь стояла нерабочая
// проверка "if srvspr.SerialNumber != """ на заведомо пустом поле свежей
// структуры - функционально это маскировалось тем, что UpdateHWHostInfo
// всё равно перечитывал и дозаполнял запись сразу после вставки, но эта
// функция должна быть корректна сама по себе.
func insertNewHostSprRecord(hs vsphere.HostInfo, serialNum string, hw *oneview.Server) (int64, error) {
	srvspr := models.ServerSprRecords{}
	serverName := strings.TrimSpace(hs.Name)

	o := orm.NewOrm()

	srvspr.Name = serverName
	srvspr.NameFQDN = serverName
	srvspr.Type = 2 // host vmware - хост найден в закэшированном пуле vSphere (infra.HostMapList), значит сейчас это гипервизор
	srvspr.CPUCount = float64(hs.NumCPUPkgs)
	srvspr.CPULogical = float64(hs.NumCPUCores)
	srvspr.MemSize = float64(hs.MemorySizeGB)
	srvspr.Stat = 1 // используется - хост сейчас реально присутствует в vSphere

	if hw != nil && hw.Identity.MpHostName == hs.Name {
		srvspr.Details = serialNum + " Host Добавлен из Vsphere, требуется правка"
	} else {
		srvspr.Details = "Virtual SN: " + hs.SerialNumber + " Host Добавлен из Vsphere, требуется правка"
	}

	srvspr.Time_Mfy = time.Now().Local()
	srvspr.OS = hs.ProductFullName //копируем точную версию esxi

	srvspr.Vcenter = hs.Vcenter
	srvspr.Datacenter = hs.Datacenter
	srvspr.Cluster = hs.Cluster
	srvspr.ClusterId = hs.ClusterID
	srvspr.SerialNumber = serialNum
	srvspr.SourceUpdateType = models.SourceUpdateVMware

	if serialNum != "" {
		hw2 := models.ServerHardware{SerialNum: serialNum}
		if err := o.Read(&hw2, "SerialNum"); err == nil {
			srvspr.HWSprId = hw2.Id
		}
	}

	Id, err := o.Insert(&srvspr)
	if err != nil {
		fmt.Println("Ошибка записи справочника серверов", err)
		return 0, err
	}
	return Id, nil
}

// checkAndCorrectHost выбирает из нескольких найденных по имени записей
// (records) ту, что лучше всего соответствует хосту host (совпадение по
// NameFQDN/CPUCount/серийному номеру и по количеству заполненных вручную
// полей), и помечает остальные как дубликаты (Stat=666 - см. тот же
// сигнальный статус в cmdb/updatehwsprhp.go, за пределами документированных
// в models.ServerSprRecords.Stat значений 0-3, используется по всему
// проекту как временный маркер "дубликат, требует ручной проверки").
func checkAndCorrectHost(records []*models.ServerSprRecords, host vsphere.HostInfo) int64 {
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

	for i := range records {
		foundIndex = append(foundIndex, 0)

		hw := models.ServerHardware{}
		qs := o.QueryTable("server_hardware").OrderBy("ServerId").Filter("ServerId", records[i].Id)
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
		exclude := false
		for _, ex := range excludeIndex {
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
			srvspr.ClusterId = host.ClusterID

			if _, err := o.Update(&srvspr); err != nil {
				fmt.Println("Ошибка пометки дубликата записи хоста:", records[i].Id, err)
			}
		}
	}
	return records[idx].Id
}

// UpdateHWHostInfo синхронизирует конфигурации хостов-гипервизоров ESXi из
// закэшированного пула vSphere (vspheremetrics.GetGlobalVMInfrastructure().
// HostMapList) со справочником серверов server_spr_records: создаёт новые
// записи, устраняет дубликаты по имени/серийному номеру, связывает записи с
// данными оборудования из HP OneView.
//
// Эта функция - "прямое" направление синхронизации (vSphere -> ServerSprRecords):
// для каждого хоста, который СЕЙЧАС реально присутствует в пуле vSphere,
// гарантирует Type=2 и актуальные Vcenter/Datacenter/Cluster/ClusterId/OS.
// "Обратное" направление - обнаружение сервера, который раньше был
// гипервизором, но по данным HP OneView (без учёта его же устаревшего
// Type==2) больше не значится хостом в vSphere и должен быть понижен до
// Type=0 ("передача владения") - выполняет cmdb.ImportServerSprFromHW
// (cmdb/initrec.go), вызываемая из OneView-стороны синхронизации
// (cmdb.UpdateHwSprHPServers). Эта функция сама НИКОГДА не понижает Type -
// у неё единственная ответственность: "хост есть в vSphere -> запись должна
// это отражать". Не смешивать обе ответственности в одной функции, чтобы
// они не могли "тянуть" Type в разные стороны в рамках одного цикла синхронизации.
//
// ⚠️ Функция не вызывается ни из одного текущего рабочего цикла (см.
// закомментированный вызов в metrics_grpc/metrics_grpc.go) - подключение
// вызова делается отдельным осознанным шагом после проверки на реальных
// данных, а не автоматически при правке этого файла.
func UpdateHWHostInfo() {
	fmt.Println("Обновление конфигураций хостов")

	ov := oneview.GetGlobalOVInfrastructure()
	infra := vsphere.GetGlobalVMInfrastructure()
	if infra == nil {
		fmt.Println("UpdateHWHostInfo: инфраструктура vSphere не инициализирована")
		return
	}

	o := orm.NewOrm()

	for _, key := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(key)
		if !ok || hs == nil {
			continue
		}

		serialNum, hw := resolveHWBySerial(ov, hs.SerialNumber)
		if hw != nil {
			if s := strings.TrimSpace(hw.EnclosureInfo.SerialNumber); s != "" { //если корзина реально есть
				hs.EnclosureSerialNumber = s
			}
		}

		// упорядочиваем справочники Vcenter/Датацентр/Кластер (создаём при
		// отсутствии) - датацентр и кластер ищутся/создаются с учётом
		// Vcenter (а кластер - ещё и Datacenter), а не по одному Name, иначе
		// при совпадении имён датацентров/кластеров в разных vCenter можно
		// было бы найти/переиспользовать чужую запись.
		vc := models.VcenterSprRecords{Name: hs.Vcenter}
		if err := o.Read(&vc, "Name"); err != nil {
			vc.Id_User = 0
			vc.Time_Mfy = time.Now().Local()
			if _, err := o.Insert(&vc); err != nil {
				fmt.Println("Ошибка создания справочника vCenter:", hs.Vcenter, err)
			}
		}

		dc := models.DatacenterSprRecords{Name: hs.Datacenter, Vcenter: hs.Vcenter}
		if err := o.Read(&dc, "Name", "Vcenter"); err != nil {
			dc.Id_User = 0
			dc.Time_Mfy = time.Now().Local()
			if _, err := o.Insert(&dc); err != nil {
				fmt.Println("Ошибка создания справочника датацентра:", hs.Vcenter, hs.Datacenter, err)
			}
		}

		cl := models.ClusterSprRecords{Name: hs.Cluster, Datacenter: hs.Datacenter, Vcenter: hs.Vcenter}
		if err := o.Read(&cl, "Name", "Datacenter", "Vcenter"); err != nil {
			cl.Id_User = 0
			cl.Time_Mfy = time.Now().Local()
			if _, err := o.Insert(&cl); err != nil {
				fmt.Println("Ошибка создания справочника кластера:", hs.Vcenter, hs.Datacenter, hs.Cluster, err)
			}
		}

		shortName := strings.TrimSpace(strings.Split(hs.Name, ".")[0]) //краткое имя сервера без домена
		serverName := strings.TrimSpace(hs.Name)

		if shortName == "" || serverName == "" || shortName == "localhost" || serverName == "localhost" {
			fmt.Printf("Ошибка в данных гипервизора SN:%s shortName:%s serverName:%s\n", hs.SerialNumber, shortName, serverName)
			continue
		}

		// ищем существующие записи справочника серверов по краткому/полному
		// имени хоста (без учёта регистра); type != 1 - исключаем только
		// виртуальные машины (Type==1), физические серверы (0), гипервизоры
		// (2) и внешние облачные серверы (3) - в выборку, так как запись
		// могла уже существовать как обычный физический сервер (см.
		// комментарий к cmdb.ImportServerSprFromHW про смену роли
		// оборудования - тот же сервер физически мог сначала попасть в
		// справочник через HP OneView).
		var records []*models.ServerSprRecords
		qs := o.QueryTable("server_spr_records").OrderBy("name")
		cond := orm.NewCondition()
		cond1 := cond.And("Type__ne", 1).AndCond(cond.Or("Name__iexact", shortName).Or("Name__iexact", serverName).Or("NameFQDN__iexact", shortName).Or("NameFQDN__iexact", serverName))
		qs = qs.SetCond(cond1)
		num, err := qs.All(&records)
		if err != nil {
			fmt.Println("Ошибка поиска записи справочника серверов по имени:", serverName, err)
		}

		if num == 0 && serialNum != "" { //не нашли по имени - пробуем найти через серийный номер и справочник оборудования
			srvHW := models.ServerHardware{SerialNum: serialNum}
			if errh := o.Read(&srvHW, "SerialNum"); errh == nil {
				if hs.PowerState {
					srvHW.PowerState = 1
				} else {
					srvHW.PowerState = 2
				}
				srvHW.SourceUpdate = "Vmware " + hs.Vcenter + " " + hs.Datacenter + " " + hs.Cluster
				srvHW.SourceUpdateType = models.SourceUpdateVMware
				if _, err := o.Update(&srvHW); err != nil {
					fmt.Println("Ошибка обновления справочника оборудования по серийному номеру:", serialNum, err)
				}

				qs = o.QueryTable("server_spr_records").OrderBy("name")
				cond = orm.NewCondition().And("Id__eq", srvHW.ServerId)
				qs = qs.SetCond(cond)
				num, err = qs.All(&records)
				if err != nil {
					fmt.Println("Ошибка поиска записи справочника серверов по HWSprId:", srvHW.Id, err)
				}
			}
		}

		var serverId int64
		switch {
		case num == 1:
			serverId = records[0].Id
		case num > 1:
			serverId = checkAndCorrectHost(records, *hs)
		default: // num == 0
			serverId, err = insertNewHostSprRecord(*hs, serialNum, hw)
			if err != nil {
				continue // ошибка уже выведена в insertNewHostSprRecord
			}
		}

		srvspr := models.ServerSprRecords{Id: serverId}
		if err := o.Read(&srvspr, "Id"); err != nil {
			fmt.Println("Ошибка чтения записи справочника серверов:", serverId, err)
			continue
		}

		if hs.PowerState {
			srvspr.PowerState = 1
		} else {
			srvspr.PowerState = 2
		}

		srvspr.Name = shortName
		srvspr.NameFQDN = serverName
		srvspr.Type = 2 // host vmware - см. комментарий к функции про единственную ответственность этой стороны синхронизации
		srvspr.Stat = 1 // используется - хост реально присутствует в vSphere прямо сейчас (раньше здесь Stat не трогался при обновлении уже существующей записи)
		srvspr.CPUCount = float64(hs.NumCPUPkgs)
		srvspr.CPULogical = float64(hs.NumCPUCores)
		srvspr.MemSize = float64(hs.MemorySizeGB)

		srvspr.OS = hs.ProductFullName //копируем точную версию esxi

		srvspr.Vcenter = hs.Vcenter
		srvspr.Datacenter = hs.Datacenter
		srvspr.Cluster = hs.Cluster
		srvspr.ClusterId = hs.ClusterID
		srvspr.SerialNumber = serialNum

		srvHW := models.ServerHardware{SerialNum: serialNum}
		errHW := o.Read(&srvHW, "SerialNum")
		if errHW != nil {
			srvHW = models.ServerHardware{ServerId: srvspr.Id}
			errHW = o.Read(&srvHW, "ServerId")
		}
		if errHW == nil {
			srvspr.HWSprId = srvHW.Id

			if hs.PowerState {
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
				srvHW.SerialNum = serialNum
			}
			srvHW.SourceUpdate = "Vmware " + hs.Vcenter + " " + hs.Datacenter + " " + hs.Cluster
			srvHW.SourceUpdateType = models.SourceUpdateVMware
			if _, err := o.Update(&srvHW); err != nil {
				fmt.Println("Ошибка обновления справочника конфигураций:", hs.Name, err)
			}
		}

		srvspr.SourceUpdateType = models.SourceUpdateVMware
		if _, err := o.Update(&srvspr); err != nil {
			fmt.Println("Ошибка обновления записи справочника серверов:", hs.Name, err)
		}
	}
	fmt.Println("Обновление конфигураций хостов завершено")
}

// UpdateHWHostInfoInfra — самостоятельная переписанная версия
// UpdateHWHostInfo для использования в cmdb.UpdateServerInventory:
// принимает снапшот инфраструктуры vSphere (infra) явным параметром вместо
// того, чтобы брать его самостоятельно через
// vsphere.GetGlobalVMInfrastructure().
//
// Зачем: при вызове из cmdb.UpdateServerInventory ПОСЛЕ HP OneView-стороны
// синхронизации (UpdateHwSprHPServersInfra/ImportServerSprFromHWInfra)
// важно, чтобы обе стороны видели ОДНО и то же согласованное состояние
// vSphere в рамках одного цикла - иначе демоция роли гипервизора,
// принятая HP OneView-стороной по одному снапшоту, может быть немедленно
// отменена этой функцией, если она возьмёт себе более поздний снапшот
// самостоятельно (см. подробное объяснение сценария рассинхронизации в
// комментарии к ImportServerSprFromHWInfra, cmdb/initrec.go). Явная
// передача infra параметром устраняет этот риск.
//
// UpdateHWHostInfo выше НЕ вызывает эту функцию и не вызывается ею - это
// две независимые функции с одинаковой логикой (кроме единственного
// отличия - источника снапшота vSphere), чтобы существующий,
// пока закомментированный вызов UpdateHWHostInfo() в
// metrics_grpc/metrics_grpc.go остался абсолютно нетронутым.
func UpdateHWHostInfoInfra(infra *vsphere.VMInfrastructure) {
	fmt.Println("Обновление конфигураций хостов")

	ov := oneview.GetGlobalOVInfrastructure()
	if infra == nil {
		fmt.Println("UpdateHWHostInfo: инфраструктура vSphere не инициализирована")
		return
	}

	o := orm.NewOrm()

	for _, key := range infra.HostMapList.Keys() {
		hs, ok := infra.HostMapList.Get(key)
		if !ok || hs == nil {
			continue
		}

		serialNum, hw := resolveHWBySerial(ov, hs.SerialNumber)
		if hw != nil {
			if s := strings.TrimSpace(hw.EnclosureInfo.SerialNumber); s != "" { //если корзина реально есть
				hs.EnclosureSerialNumber = s
			}
		}

		vc := models.VcenterSprRecords{Name: hs.Vcenter}
		if err := o.Read(&vc, "Name"); err != nil {
			vc.Id_User = 0
			vc.Time_Mfy = time.Now().Local()
			if _, err := o.Insert(&vc); err != nil {
				fmt.Println("Ошибка создания справочника vCenter:", hs.Vcenter, err)
			}
		}

		dc := models.DatacenterSprRecords{Name: hs.Datacenter, Vcenter: hs.Vcenter}
		if err := o.Read(&dc, "Name", "Vcenter"); err != nil {
			dc.Id_User = 0
			dc.Time_Mfy = time.Now().Local()
			if _, err := o.Insert(&dc); err != nil {
				fmt.Println("Ошибка создания справочника датацентра:", hs.Vcenter, hs.Datacenter, err)
			}
		}

		cl := models.ClusterSprRecords{Name: hs.Cluster, Datacenter: hs.Datacenter, Vcenter: hs.Vcenter}
		if err := o.Read(&cl, "Name", "Datacenter", "Vcenter"); err != nil {
			cl.Id_User = 0
			cl.Time_Mfy = time.Now().Local()
			if _, err := o.Insert(&cl); err != nil {
				fmt.Println("Ошибка создания справочника кластера:", hs.Vcenter, hs.Datacenter, hs.Cluster, err)
			}
		}

		shortName := strings.TrimSpace(strings.Split(hs.Name, ".")[0]) //краткое имя сервера без домена
		serverName := strings.TrimSpace(hs.Name)

		if shortName == "" || serverName == "" || shortName == "localhost" || serverName == "localhost" {
			fmt.Printf("Ошибка в данных гипервизора SN:%s shortName:%s serverName:%s\n", hs.SerialNumber, shortName, serverName)
			continue
		}

		var records []*models.ServerSprRecords
		qs := o.QueryTable("server_spr_records").OrderBy("name")
		cond := orm.NewCondition()
		cond1 := cond.And("Type__ne", 1).AndCond(cond.Or("Name__iexact", shortName).Or("Name__iexact", serverName).Or("NameFQDN__iexact", shortName).Or("NameFQDN__iexact", serverName))
		qs = qs.SetCond(cond1)
		num, err := qs.All(&records)
		if err != nil {
			fmt.Println("Ошибка поиска записи справочника серверов по имени:", serverName, err)
		}

		if num == 0 && serialNum != "" { //не нашли по имени - пробуем найти через серийный номер и справочник оборудования
			srvHW := models.ServerHardware{SerialNum: serialNum}
			if errh := o.Read(&srvHW, "SerialNum"); errh == nil {
				if hs.PowerState {
					srvHW.PowerState = 1
				} else {
					srvHW.PowerState = 2
				}
				srvHW.SourceUpdate = "Vmware " + hs.Vcenter + " " + hs.Datacenter + " " + hs.Cluster
				srvHW.SourceUpdateType = models.SourceUpdateVMware
				if _, err := o.Update(&srvHW); err != nil {
					fmt.Println("Ошибка обновления справочника оборудования по серийному номеру:", serialNum, err)
				}

				qs = o.QueryTable("server_spr_records").OrderBy("name")
				cond = orm.NewCondition().And("Id__eq", srvHW.ServerId)
				qs = qs.SetCond(cond)
				num, err = qs.All(&records)
				if err != nil {
					fmt.Println("Ошибка поиска записи справочника серверов по HWSprId:", srvHW.Id, err)
				}
			}
		}

		var serverId int64
		switch {
		case num == 1:
			serverId = records[0].Id
		case num > 1:
			serverId = checkAndCorrectHost(records, *hs)
		default: // num == 0
			serverId, err = insertNewHostSprRecord(*hs, serialNum, hw)
			if err != nil {
				continue // ошибка уже выведена в insertNewHostSprRecord
			}
		}

		srvspr := models.ServerSprRecords{Id: serverId}
		if err := o.Read(&srvspr, "Id"); err != nil {
			fmt.Println("Ошибка чтения записи справочника серверов:", serverId, err)
			continue
		}

		if hs.PowerState {
			srvspr.PowerState = 1
		} else {
			srvspr.PowerState = 2
		}

		srvspr.Name = shortName
		srvspr.NameFQDN = serverName
		srvspr.Type = 2 // host vmware - см. комментарий к функции про единственную ответственность этой стороны синхронизации
		srvspr.Stat = 1 // используется - хост реально присутствует в vSphere прямо сейчас
		srvspr.CPUCount = float64(hs.NumCPUPkgs)
		srvspr.CPULogical = float64(hs.NumCPUCores)
		srvspr.MemSize = float64(hs.MemorySizeGB)

		srvspr.OS = hs.ProductFullName //копируем точную версию esxi

		srvspr.Vcenter = hs.Vcenter
		srvspr.Datacenter = hs.Datacenter
		srvspr.Cluster = hs.Cluster
		srvspr.ClusterId = hs.ClusterID
		srvspr.SerialNumber = serialNum

		srvHW := models.ServerHardware{SerialNum: serialNum}
		errHW := o.Read(&srvHW, "SerialNum")
		if errHW != nil {
			srvHW = models.ServerHardware{ServerId: srvspr.Id}
			errHW = o.Read(&srvHW, "ServerId")
		}
		if errHW == nil {
			srvspr.HWSprId = srvHW.Id

			if hs.PowerState {
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
				srvHW.SerialNum = serialNum
			}
			srvHW.SourceUpdate = "Vmware " + hs.Vcenter + " " + hs.Datacenter + " " + hs.Cluster
			srvHW.SourceUpdateType = models.SourceUpdateVMware
			if _, err := o.Update(&srvHW); err != nil {
				fmt.Println("Ошибка обновления справочника конфигураций:", hs.Name, err)
			}
		}

		srvspr.SourceUpdateType = models.SourceUpdateVMware
		if _, err := o.Update(&srvspr); err != nil {
			fmt.Println("Ошибка обновления записи справочника серверов:", hs.Name, err)
		}
	}
	fmt.Println("Обновление конфигураций хостов завершено")
}
