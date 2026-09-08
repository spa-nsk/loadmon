package controllers

import (
	"encoding/csv"
	"fmt"
	"loadmon/csvtools"
	"loadmon/grpcclient"
	"loadmon/itsystem"
	models "loadmon/models"
	vsphere "loadmon/vspheremetrics"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// newLightServerView строит "лёгкую" запись models.ServerSprView по одной
// записи справочника rec: заполняет только r.S и вычисляет
// PercentFilledField, без дополнительных запросов к БД (VM/Hardware
// остаются пустыми заглушками и заполняются позже - см. enrichServerViews).
func newLightServerView(rec *models.ServerSprRecords) models.ServerSprView {
	r := models.ServerSprView{}
	r.S = *rec
	r.VM = &models.VMSprRecordView{}
	r.Hardware = &models.ServerHardwareView{}
	CalcPercentFilled(&r)
	return r
}

// enrichServerViews догружает детали виртуальной машины (vm_spr_record) и
// аппаратной конфигурации (server_hardware) для каждой записи списка list -
// по два дополнительных запроса на запись. Вызывается точечно, только для
// того среза записей, который реально нужен на этом шаге (текущая страница
// при обычном просмотре, либо весь список непосредственно перед экспортом
// в CSV/XLSX) - а не для всей найденной фильтром выборки сразу, как было
// раньше. Именно повторение этих запросов для каждой найденной записи (а
// не только отображаемой странице) было основной причиной медленной
// загрузки /sprsrv/view2 при больших объёмах справочника.
func enrichServerViews(o orm.Ormer, list []models.ServerSprView) {
	for i := range list {
		rec := &list[i].S

		if rec.VMSprId != 0 && rec.Type == 1 {
			vm := models.VMSprRecordView{}
			qs := o.QueryTable("vm_spr_record").OrderBy("id").Filter("id", rec.VMSprId)
			if err := qs.One(&vm.VMSprRecord); err == nil {
				list[i].VM = &vm
			}
		}

		if rec.Type == 2 || rec.Type == 0 {
			hw := models.ServerHardwareView{}
			qs := o.QueryTable("server_hardware").OrderBy("ServerId").Filter("ServerId", rec.Id)
			if err := qs.One(&hw.H); err == nil {
				list[i].Hardware = &hw
			}
		}
	}
}

// SprSrvViewV2 отображает справочник серверов/ВМ (актуальная версия страницы
// просмотра): список с фильтрацией и постраничной навигацией, а также экспорт
// данных в CSV/Excel в зависимости от режима Mode.
func (this *MainController) SprSrvViewV2() {

	Mode := this.Ctx.Input.Param(":mode")
	Id := this.Ctx.Input.Param(":id")
	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	this.activeContent("sprsrv/view2")
	this.Data["ListId"] = Id
	this.Data["backmode"] = Mode

	var (
		records      []*models.ServerSprRecords
		recordsAdd   []*models.ServerSprRecords
		recordsVMAdd []*models.ServerSprRecords
		list         []models.ServerSprView
	)

	// indexSrv - множество Id уже добавленных в list записей: используется
	// для дедупликации, когда одна и та же запись server_spr_records может
	// быть найдена и основным фильтром, и расширенным поиском по кластерам
	// (recordsVMAdd). Дедупликация по Id (первичный ключ) проще и надёжнее,
	// чем прежний вариант по конкатенации Name/NameFQDN/IPAddress.
	indexSrv := make(map[int64]bool)

	//rawSelect := "SELECT B.* FROM server_spr_records B JOIN spr_cfo_allocation C ON B.id = C.server_id WHERE B.stat != '666' "
	rawVMSelect := "SELECT B.* FROM server_spr_records B JOIN vm_spr_record C ON B.v_m_spr_id = C.id WHERE B.stat != '666' "

	this.Data["ViewOn"] = 0
	this.Data["ExportCSV"] = 0
	exportcsv := false

	this.Data["ExportXLSX"] = 0
	exportxlsx := false

	var (
		//				sprITloaded chan bool
		// sprITloaded = make(chan bool, 1)
		itSystemSpr []models.ItSystemRecord
		err         error
	)
	//		go func() {
	//itSystemSpr, err = itsystem.GetAllItSystem()
	itSystemSpr = itsystem.ImportITSystemSpr()
	//				sprITloaded <- true
	//		}()

	this.Data["ITSystemSpr"] = itSystemSpr //

	o := orm.NewOrm()
	//o.Using("default")

	//o1 := orm.NewOrm()
	//o1.Using("vm")
	appservice := make([]models.AppService, 0)
	qs := o.QueryTable("app_service")
	qs.All(&appservice)

	this.Data["AppService"] = appservice

	qs = o.QueryTable("server_spr_records")
	qs = qs.OrderBy("active", "name")
	cond := orm.NewCondition()

	cond = cond.And("Stat__ne", 666)
	active := 0

	switch Mode {
	case "new":
		//cond = cond.And("active__eq", 0)
		active = 0
	case "use":
		//cond = cond.And("active__eq", 1)
		active = 1
	case "del":
		//cond = cond.And("active__eq", 2)
		active = 2
	case "mix":
		//cond = cond.And("active__ne", 2)
		active = 998
	case "all":
		active = 999
	default:
	}

	_ = active

	m := sess.(models.LoadFilter).Map
	filter := sess.(models.LoadFilter)
	this.Data["Filter"] = filter

	cfo, ok := m["FilterCFO"].(*models.SprCFOAllocation)
	if !ok {
		cfo = &models.SprCFOAllocation{}
		m["FilterCFO"] = cfo
	}

	this.Data["cfo"] = cfo

	record, ok := m["FilterServer"].(*models.ServerSprRecords)
	if !ok {
		record = &models.ServerSprRecords{
			Type:             999,
			TypeMicroservice: 999,
			TypeWork:         0,
			CriticalLevel:    0,
			PowerState:       1,
			Active:           active,
			DPCAddress:       0,
			DPCOffice:        0,
			ServiceId:        0,
		}
		m["FilterServer"] = record
	}

	this.Data["record"] = record

	clusterList, ok := m["ClusterList"].([]*vsphere.Cluster)
	if !ok {
		/*
			clusterList = grpcclient.LoadClusterListFull("all", "all", "all")
			m["ClusterList"] = clusterList
		*/
		clList := grpcclient.LoadClusterList("all", "all")

		for _, clShort := range clList {
			cl := grpcclient.GetClusterMedium(clShort.Vcenter, clShort.Datacenter, clShort.Name)
			if cl.HostFlag {
				continue
			}
			clusterList = append(clusterList, cl)
		}
		sort.Sort(vsphere.ClusterVcenter(clusterList))
		this.Data["ClusterListData"] = clusterList
		m["ClusterList"] = clusterList
	}

	this.Data["ClusterList"] = clusterList
	baseBtn := ""
	allocationBtn := ""

	flash := beego.NewFlash()
	isPost := this.Ctx.Input.Method() == "POST"
	if isPost {
		if err := this.ParseForm(record); err != nil {
			//beego.Error("Ошибка распознавания формы:", err)
			flash.Error("Ошибка распознавания формы %s", err)
			flash.Store(&this.Controller)
			return
		}

		// beego.ParseForm (beego/v2/server/web/context/form.go: formFirstValue)
		// для строковых полей молча пропускает присвоение, если значение поля
		// формы - пустая строка (formFirstValue возвращает ok=false для value
		// == ""). Из-за этого при очистке текстового поля фильтра и нажатии
		// "Просмотр" ParseForm никак не изменяет соответствующее поле record -
		// а record это не новый объект на каждый запрос, а указатель,
		// сохранённый в сессии (m["FilterServer"] выше), поэтому в нём остаётся
		// прежнее (непустое) значение с прошлого поиска, и очищенное поле
		// фильтра "само" заполняется обратно. Читаем эти строковые поля
		// вручную через this.GetString - он корректно возвращает "" для
		// пустого/отсутствующего поля - и присваиваем их record напрямую,
		// перекрывая то, что (не) сделал ParseForm.
		record.Name = this.GetString("Name")
		record.NameFQDN = this.GetString("NameFQDN")
		record.NameInfo = this.GetString("NameInfo")
		record.ITSystemKey = this.GetString("ITSystemKey")
		record.OS = this.GetString("OS")
		record.Role = this.GetString("Role")
		record.Anotation = this.GetString("Anotation")
		record.Network = this.GetString("Network")
		record.IPAddress = this.GetString("IPAddress")
		record.Owner = this.GetString("Owner")

		expBtn := this.GetString("ExportBTN")
		if expBtn == "Экспорт в CSV" {
			this.Data["ExportCSV"] = 1
			exportcsv = true
		} else {
			this.Data["ExportCSV"] = 0
		}

		expBtn2 := this.GetString("ExportBTN2")
		if expBtn2 == "Экспорт в XLSX" {
			this.Data["ExportXLSX"] = 1
			exportxlsx = true
		} else {
			this.Data["ExportXLSX"] = 0
		}

		this.Data["ViewOn"] = 1
		// запоминаем в сессии, что список уже был показан с текущим фильтром -
		// это позволяет при возврате обычным GET (после редактирования сервера,
		// просмотра деталей или группового редактирования) снова показать
		// список с учётом внесённых изменений без повторного нажатия "Просмотр"
		m["ViewOnFlag"] = true
		for idx, cl := range clusterList {
			clusterList[idx].ID = 0
			if this.GetString("ch"+cl.Name) == "on" {
				clusterList[idx].ID = 1
			}
		}
		baseBtn = this.GetString("SetBaseBTN")

		allocationBtn = this.GetString("SetAllocateBTN")

	}

	// Фильтр размещения "ЦОД -> Шкаф" (2026-09-08, по запросу пользователя
	// "замени в форме /sprsrv/view2 ... схему с выбором ЦОД и Шкафа вместо
	// ЦОД: адрес и площадка по старым данным") - хранится ОТДЕЛЬНО от record
	// (models.ServerSprRecords не имеет и не должна получать новое поле ради
	// одного фильтра списка - тот же принцип, что и HWConfigFilter в
	// sprhwconfig.go), значение - строка ("0"/"" - фильтр не применён,
	// аналогично остальным select-фильтрам этой страницы). Списки ЦОД/Шкаф
	// для select'ов (см. view2.tpl) грузятся безусловно (и на GET, и на
	// POST) - тот же приём, что в sprhwconfig.go:SprHwConfig.
	rackCabinetId, _ := m["FilterRackCabinetId"].(string)
	if isPost {
		rackCabinetId = this.GetString("RackCabinetId")
		m["FilterRackCabinetId"] = rackCabinetId
	}
	this.Data["RackCabinetId"] = rackCabinetId

	var dcSites []models.DatacenterSite
	o.QueryTable("cmdb_datacenter_site").OrderBy("Name").All(&dcSites)
	this.Data["DatacenterSites"] = dcSites

	var rackCabinets []models.RackCabinet
	o.QueryTable("cmdb_rack_cabinet").OrderBy("Name").All(&rackCabinets)
	this.Data["RackCabinets"] = rackCabinets

	id, err := strconv.Atoi(Id)
	if err != nil {
		//beego.Error("Недопустимый аргумент Id", err)
		flash.Error("Недопустимый аргумент Id", err)
		flash.Store(&this.Controller)
		return
	}

	if id != 0 {
		this.Data["ViewOn"] = 1
	}

	// обычный GET-переход (ссылка "Отмена", редирект после сохранения на
	// /sprsrv/updatebase, /sprsrv/updateselectedbase и т.п.) - если фильтр уже
	// применялся раньше в этой сессии, восстанавливаем список автоматически
	if !isPost && this.Data["ViewOn"] == 0 {
		if v, ok := m["ViewOnFlag"].(bool); ok && v {
			this.Data["ViewOn"] = 1
		}
	}

	if this.Data["ViewOn"] == 0 {
		return
	}
	//
	//	if this.GetString("add") == "add" {
	//		if strings.TrimSpace(record.Name) == "" {
	//			this.Redirect("/sprsrv/addserver"+"/"+Mode+"/"+ "new", 302)
	//		} else {
	//			this.Redirect("/sprsrv/addserver"+"/"+Mode+"/"+record.Name, 302)
	//		}
	//		return
	//	}
	//
	// состояние чекбоксов "Основной блок фильтра" / "Дополнение фильтра по
	// кластерам" тоже сохраняем в сессии, иначе при автоматическом
	// восстановлении списка на обычном GET (см. ViewOnFlag выше) они бы
	// читались как пустые (GetString на GET-запросе без тела формы) и фильтр
	// применялся бы не так, как был настроен пользователем
	baseFilter := false
	if isPost {
		baseFilter = this.GetString("chBaseFilter") == "on"
		m["ChBaseFilter"] = baseFilter
	} else if v, ok := m["ChBaseFilter"].(bool); ok {
		baseFilter = v
	}

	var num int64

	if baseFilter {

		switch record.Active {
		case 0:
			cond = cond.And("active", 0)
		case 1:
			cond = cond.And("active", 1)
		case 2:
			cond = cond.And("active", 2)
		case 998:
			cond = cond.And("Active__ne", 2)
		case 999:
		}

		record.Name = strings.TrimSpace(record.Name)

		if record.Name != "" {
			condName := orm.NewCondition()
			names := strings.Split(record.Name, ";")
			for m := range names {
				s := strings.TrimSpace(names[m])
				if s != "" {
					condName = condName.Or("Name__icontains", s)
				}
			}
			cond = cond.AndCond(condName)
		}

		record.NameInfo = strings.TrimSpace(record.NameInfo)

		if record.NameInfo != "" {
			condName := orm.NewCondition()
			names := strings.Split(record.NameInfo, ";")
			for m := range names {
				s := strings.TrimSpace(names[m])
				if s != "" {
					condName = condName.Or("NameInfo__icontains", s)
				}
			}
			cond = cond.AndCond(condName)
		}

		record.NameFQDN = strings.TrimSpace(record.NameFQDN)

		if record.NameFQDN != "" {
			condName := orm.NewCondition()
			names := strings.Split(record.NameFQDN, ";")
			for m := range names {
				s := strings.TrimSpace(names[m])
				if s != "" {
					condName = condName.Or("NameFQDN__icontains", s)
				}
			}
			cond = cond.AndCond(condName)
		}

		if record.ITSystemKey != "" {
			cond = cond.And("ITSystemKey", record.ITSystemKey)
		}

		// ИЗМЕНЕНО (2026-09-08, по запросу пользователя): фильтр по
		// record.DPCOffice (старый жёстко зашитый в HTML список "ЦОД: адрес
		// и площадка") заменён на фильтр по выбранному шкафу (RackCabinetId,
		// см. view2.tpl и загрузку rackCabinetId выше) - см.
		// placementServerSprIds ниже про правила определения размещения для
		// аппаратных серверов/гипервизоров, блейд-серверов (приоритет у
		// размещения самой корзины) и виртуальных машин (по хосту).
		if rackCabinetId != "" && rackCabinetId != "0" {
			if v, errRc := strconv.ParseInt(rackCabinetId, 10, 64); errRc == nil && v > 0 {
				placementIds := placementServerSprIds(o, v)
				if len(placementIds) == 0 {
					// ничего не размещено в выбранном шкафу - гарантированно
					// пустой результат вместо некорректного "Id__in" с пустым
					// списком (часть ORM-драйверов не умеет "IN ()").
					cond = cond.And("Id", int64(-1))
				} else {
					cond = cond.And("Id__in", placementIds)
				}
			}
		}
		if record.RoleType != 0 {
			cond = cond.And("RoleType", record.RoleType)
		}

		if record.Type == 4 {
			// Псевдо-тип "Аппаратный сервер в Zabbix" (значение 4 в форме фильтра
			// не хранится в БД) - реальный тип 0 (аппаратный сервер) + признак
			// регистрации в Zabbix (ZabbixHostId > 0).
			cond = cond.And("type", 0).And("ZabbixHostId__gt", 0)
		} else if record.Type != 999 {
			cond = cond.And("type", record.Type)
		}

		if record.TypeMicroservice != 999 {
			cond = cond.And("TypeMicroservice", record.TypeMicroservice)
		}

		if record.TypeWork != 0 {
			cond = cond.And("TypeWork", record.TypeWork)
		}

		if record.CriticalLevel != 0 {
			cond = cond.And("CriticalLevel", record.CriticalLevel)
		}
		//		if record.ServiceId != 0 {
		//			cond = cond.And("service_id", record.ServiceId)
		//		}

		if strings.TrimSpace(record.ITSystemKey) != "" {
			cond = cond.And("ITSystemKey", record.ITSystemKey)
		}

		//		if strings.TrimSpace(record.Service) != "" {
		//			cond = cond.And("service__icontains", record.Service)
		//		}
		if strings.TrimSpace(record.Anotation) != "" {
			cond = cond.And("Anotation__icontains", record.Anotation)
		}
		if strings.TrimSpace(record.Role) != "" {
			cond = cond.And("Role__icontains", record.Role)
		}

		record.Owner = strings.TrimSpace(record.Owner)

		if record.Owner != "" {
			condOwner := orm.NewCondition()
			owners := strings.Split(record.Owner, ";")
			for m := range owners {
				s := strings.TrimSpace(owners[m])
				if s != "" {
					condOwner = condOwner.Or("Owner__icontains", s)
				}
			}
			cond = cond.AndCond(condOwner)
		}
		/*
			if strings.TrimSpace(record.Allocation) != "" {
				condAllocation := orm.NewCondition()
				alloc := strings.Split(record.Allocation, ";")
				for m := range alloc {
					s := strings.TrimSpace(alloc[m])
					if s != "" {
						condAllocation = condAllocation.Or("allocation__icontains", s)
					}
				}
				cond = cond.AndCond(condAllocation)
			}
		*/
		//	if strings.TrimSpace(record.Component) != "" {
		//		cond = cond.And("component__icontains", record.Component)
		//	}
		record.OS = strings.TrimSpace(record.OS)
		if record.OS == "не указано" || record.OS == "-" { //поиск незаполненных
			cond = cond.And("OS__eq", "")
		} else if record.OS != "" {
			condName := orm.NewCondition()
			os := strings.Split(record.OS, ";")
			for m := range os {
				s := strings.TrimSpace(os[m])
				if s != "" {
					condName = condName.And("OS__icontains", s)
				}
			}
			cond = cond.AndCond(condName)
		}

		if record.PowerState != 0 {
			cond = cond.And("PowerState", record.PowerState)
		}

		if strings.TrimSpace(record.Network) != "" {
			cond = cond.And("Network__icontains", record.Network)
		}

		if strings.TrimSpace(record.NetworkName) != "" {
			cond = cond.And("NetworkName__icontains", record.NetworkName)
		}

		if strings.TrimSpace(record.IPAddress) != "" {
			cond = cond.And("IPAddress__icontains", record.IPAddress)
		}

		//		if strings.TrimSpace(record.IPSharedVIP) != "" {
		//			cond = cond.And("i_p_shared_v_i_p__icontains", record.IPSharedVIP)
		//		}

		//	cond = cond.AndCond(cond1)

		qs = qs.SetCond(cond)

		num, err = qs.All(&records)
	}
	//-------------------------------------------------------------------------
	//rawSelect := "SELECT B.* FROM server_spr_records B JOIN spr_cfo_allocation C ON B.id = C.server_id WHERE B.stst != '666' "
	//C.busines LIKE '%Розничный%'
	var (
		numAdd   int64
		errAdd   error
		numVMAdd int64
		errVMAdd error
	)
	/*
		if this.GetString("chAllocation") == "on" {
			cfo.Name = strings.TrimSpace(this.GetString("NameCFO"))
			cfo.Busines = strings.TrimSpace(this.GetString("BusinesCFO"))
			if cfo.Name != "" || cfo.Busines != "" {
				rawSelect += "AND ("
				if cfo.Name != "" {
					rawSelect += "lower(C.name) LIKE lower('" + cfo.Name + "')"
				}

				if cfo.Busines != "" {
					if cfo.Name != "" {
						rawSelect += " OR "
					}
					rawSelect += "lower(C.busines) LIKE lower('" + cfo.Busines + "') "
				}
				rawSelect += ")"
			}
			rawSelect += " ;"
			numAdd, errAdd = o.Raw(rawSelect).QueryRows(&recordsAdd)
		}
	*/
	chClusters := false
	if isPost {
		chClusters = this.GetString("chClusters") == "on"
		m["ChClusters"] = chClusters
	} else if v, ok := m["ChClusters"].(bool); ok {
		chClusters = v
	}
	if chClusters {
		found := false
		for idx, cl := range clusterList {
			if clusterList[idx].ID == 1 {
				if !found {
					rawVMSelect += "AND ("
					found = true
				} else {
					rawVMSelect += " OR "
				}
				rawVMSelect += "C.cluster ='" + cl.Name + "' "
			}
		}
		if found {
			rawVMSelect += ")"
			rawVMSelect += " ;"
			numVMAdd, errVMAdd = o.Raw(rawVMSelect).QueryRows(&recordsVMAdd)
		}
	}
	//----------------------------------------------------------------------------
	log.Println("Количество записей", len(records))

	// Построение list выполняется в лёгком режиме: без запросов CFO/Busines
	// (эти поля - срезы spr_cfo_allocation/spr_busines_allocation - нигде не
	// отображаются в шаблоне и не попадают в CSV/XLSX-экспорт, т.к.
	// csvtools.GetCSVHeader/GetCSVData пропускают поля-срезы; тратить на них
	// запрос на каждую запись не имеет смысла) и без запросов VM/Hardware
	// (server_id-детали подгружаются отдельно функцией enrichServerViews -
	// см. ниже - только для записей текущей страницы при обычном просмотре,
	// либо для всего списка непосредственно перед экспортом в CSV/XLSX).
	// Раньше эти 4 запроса на запись выполнялись для ВСЕХ найденных записей
	// ещё до пагинации - именно это было причиной медленной загрузки
	// /sprsrv/view2 при большом количестве серверов: пагинация лишь урезала
	// то, что отдавалось в шаблон, а не объём уже выполненной работы.
	if err != orm.ErrNoRows && num > 0 {
		list = make([]models.ServerSprView, 0, len(records))
		for k := range records {
			rec := records[k]
			if indexSrv[rec.Id] {
				continue
			}
			indexSrv[rec.Id] = true
			list = append(list, newLightServerView(rec))
		}
	}

	if errAdd == nil && numAdd > 0 { // добавляем из расширенного поиска
		for k := range recordsAdd {
			rec := recordsAdd[k]
			if indexSrv[rec.Id] {
				continue
			}
			indexSrv[rec.Id] = true
			list = append(list, newLightServerView(rec))
		}
	}

	if errVMAdd == nil && numVMAdd > 0 { //добавляем из расширенного поиска
		for k := range recordsVMAdd {
			rec := recordsVMAdd[k]
			if indexSrv[rec.Id] {
				continue
			}
			indexSrv[rec.Id] = true
			list = append(list, newLightServerView(rec))
		}
	}
	// Пагинация HTML-таблицы: отдаём в шаблон только текущую страницу
	// списка, а не весь результат фильтрации целиком. Это резко ускоряет
	// отрисовку при больших выборках (тысячи серверов) - сервер тратит
	// меньше времени на выполнение шаблона, а браузер - на построение DOM.
	// На экспорт в CSV/XLSX (ниже) пагинация не влияет - там используется
	// полный список list.
	pageSize, _ := strconv.Atoi(this.GetString("pagesize"))
	switch pageSize {
	case 25, 50, 100, 200:
	default:
		pageSize = 50
	}
	page, _ := strconv.Atoi(this.GetString("page"))
	if page < 1 {
		page = 1
	}
	totalRecords := len(list)
	totalPages := (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > totalRecords {
		end = totalRecords
	}
	pagedList := make([]models.ServerSprView, 0)
	if offset < end {
		pagedList = list[offset:end]
	}

	// Детали VM/Hardware (2 доп. запроса на запись) подгружаются только для
	// записей текущей страницы - не более PageSize (25/50/100/200) штук,
	// а не для всей найденной выборки. pagedList - это подсрез list с общим
	// backing-массивом, поэтому изменения видны и в list (важно для экспорта
	// ниже, если он запрошен для того же результата).
	enrichServerViews(o, pagedList)

	this.Data["records"] = pagedList
	this.Data["CountRecords"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = 0
	if totalRecords > 0 {
		this.Data["PageFrom"] = offset + 1
	}
	this.Data["PageTo"] = end

	if exportxlsx || exportcsv {
		// экспорт отдаёт весь найденный список (list), а не только текущую
		// страницу - поэтому здесь (и только здесь) детали VM/Hardware
		// догружаются для всех записей списка, а не только для pagedList
		// (для записей внутри диапазона текущей страницы это повторный, но
		// безвредный запрос - основная часть list детализируется впервые)
		enrichServerViews(o, list)
	}

	if exportxlsx {
		if len(list) > 0 {
			crt := time.Now().Local()
			short_name := fmt.Sprintf("%02d_%02d_%04d_%02d_%02d_%02d_%d.xlsx", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
			destination := "./reports/" + "Excel_" + short_name

			file := excelize.NewFile()
			if file == nil {
				flash.Error("Ошибка создания XLSX файла: " + destination)
				flash.Store(&this.Controller)
				return
			}

			header, _ := csvtools.GetCSVHeader(&list[0])
			sheet := "Список серверов"
			index, err := file.NewSheet(sheet)
			if err != nil {
				flash.Error("Ошибка закрытия XLSX файла: "+destination+" ", err)
				flash.Store(&this.Controller)
				file.Close()
				return
			}
			file.SetActiveSheet(index)

			//формируем заголовок
			cell, err := excelize.CoordinatesToCellName(1, 1)
			if err != nil {
				flash.Error("Ошибка преобразования координат XLSX файла: "+destination+" ", err)
				flash.Store(&this.Controller)
				file.Close()
				return
			}
			err = file.SetSheetRow(sheet, cell, &header)
			if err != nil {
				flash.Error("Ошибка записи заголовка XLSX файла: "+destination+" ", err)
				flash.Store(&this.Controller)
				file.Close()
				return
			}

			for j, rec := range list {
				if rec.S.Stat == 666 {
					continue
				}
				row, _ := csvtools.GetCSVData(&rec)
				//формируем строку j+1
				cell, err := excelize.CoordinatesToCellName(1, j+2)
				if err != nil {
					flash.Error("Ошибка преобразования координат XLSX файла: "+destination+" ", err)
					flash.Store(&this.Controller)
					file.Close()
					return
				}
				err = file.SetSheetRow(sheet, cell, &row)
				if err != nil {
					flash.Error("Ошибка записи строки XLSX файла: "+destination+" ", err)
					flash.Store(&this.Controller)
					file.Close()
					return
				}
			}

			if err := file.SaveAs(destination); err != nil {
				flash.Error("Ошибка сохранения XLSX файла: "+destination+" ", err)
				flash.Store(&this.Controller)
				file.Close()
				return
			}

			if err := file.Close(); err != nil {
				flash.Error("Ошибка закрытия XLSX файла: "+destination+" ", err)
				flash.Store(&this.Controller)
				return
			}

			fileInfo, err := os.Stat(destination)

			if err != nil {
				flash.Error("Ошибка получения данных XLSX файла: "+destination+" ", err)
				flash.Store(&this.Controller)
				return
			}

			this.Ctx.Output.Header("Content-Description", "File Transfer")
			this.Ctx.Output.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
			this.Ctx.Output.Header("Content-Disposition", "attachment; filename=\""+short_name+"\"")
			this.Ctx.Output.Header("Expires", "0")
			this.Ctx.Output.Header("Cache-Control", "must-revalidate")
			this.Ctx.Output.Header("Pragma", "public")
			this.Ctx.Output.Header("Content-Length", fmt.Sprint(fileInfo.Size()))

			http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)

		}
	}
	if exportcsv {
		if len(list) > 0 {
			crt := time.Now().Local()
			short_name := fmt.Sprintf("%02d_%02d_%04d_%02d_%02d_%02d_%d.csv", crt.Day(), crt.Month(), crt.Year(), crt.Hour(), crt.Minute(), crt.Second(), crt.UnixNano())
			destination := "./reports/" + "CSV_" + short_name

			csvFile, err := os.Create(destination)
			if err != nil {
				//beego.Error("Ошибка создания CSV файла: "+destination+" ", err)
				flash.Error("Ошибка создания CSV файла: "+destination+" ", err)
				flash.Store(&this.Controller)
				return
			}
			csvwriter := csv.NewWriter(csvFile)
			csvwriter.Comma = '|'
			header, _ := csvtools.GetCSVHeader(&list[0])
			_ = csvwriter.Write(header)

			for _, rec := range list {
				if rec.S.Stat == 666 {
					continue
				}
				data, _ := csvtools.GetCSVData(&rec)
				_ = csvwriter.Write(data)
			}
			csvwriter.Flush()
			csvFile.Close()

			this.Ctx.Output.Header("Content-Disposition", "attachment; "+"filename="+short_name)
			this.Ctx.Output.Header("Content-Description", "File Transfer")
			this.Ctx.Output.Header("Content-Type", "application/csv")
			this.Ctx.Output.Header("Content-Transfer-Encoding", "text")
			this.Ctx.Output.Header("Expires", "0")
			this.Ctx.Output.Header("Cache-Control", "must-revalidate")
			this.Ctx.Output.Header("Pragma", "public")

			http.ServeFile(this.Ctx.Output.Context.ResponseWriter, this.Ctx.Output.Context.Request, destination)

		}
	}

	if baseBtn == "Редактировать выделенные" {
		strList := ""
		for _, s := range list {
			strId := strconv.FormatInt(s.S.Id, 10)
			if this.GetString("chSelection"+strId) == "on" {
				if strList == "" {
					strList = strId
				} else {
					strList += ("-" + strId)
				}
			}
		}
		if strList != "" {
			this.Redirect("/sprsrv/updateselectedbase/"+strList, 302)
		} else {
			//beego.Error("Выберите хоть один сервер")
			flash.Error("Выберите хоть один сервер")
			flash.Store(&this.Controller)
			return
		}
	}
	if allocationBtn == "Аллокация" {

	}

	RegisterRequest(this, 5)

}

/*
Поля обязательные для заполнения
Аллокация        Аллокация (Подразделение), Может совмещать в себе несколько колонн
Тип среды       Продуктовая, ПреПрод, Внешний тест, Внутренний тест
Компонента      Краткое назначение сервера, текстовое поле (например: kafka, pSQL...)
APP/DB сервер  Тип сервера (ППО или БД)
Тип сервера     VM - виртуальный сервер, HW - железный сервер, MS - микросервисы
Состояние сервера    Активен, выведен из работы
Дата изменения статуса состояние сервера
Центральный мониторинг  Учет постановки системы в ЦМ (центральном мониторинге)
Дата изменения статуса
Ответственный   Владелец системы

*/
// CalcPercentFilled вычисляет процент заполненности карточки сервера rec
// (долю заполненных ключевых полей справочника) для отображения в интерфейсе.
func CalcPercentFilled(rec *models.ServerSprView) {
	count := 0.0
	if strings.TrimSpace(rec.S.Allocation) != "" {
		count++
	}
	if strings.TrimSpace(rec.S.Anotation) != "" {
		count++
	}
	if strings.TrimSpace(rec.S.Component) != "" {
		count++
	}
	if rec.S.CriticalLevel != 0 {
		count++
	}
	if rec.S.DPCAddress != 0 {
		count++
	}
	if rec.S.DPCOffice != 0 {
		count++
	}
	if strings.TrimSpace(rec.S.Details) != "" {
		count++
	}
	if strings.TrimSpace(rec.S.IPAddress) != "" || strings.TrimSpace(rec.S.IPSharedVIP) != "" {
		count++
	}
	if rec.S.Monitoring != 0 {
		count++
	}
	if strings.TrimSpace(rec.S.OS) != "" {
		count++
	}
	if strings.TrimSpace(rec.S.Owner) != "" {
		count++
	}
	if rec.S.TypeWork != 0 {
		count++
	}

	rec.PercentFilledField = count * 100 / 12
}

// placementServerSprIds возвращает Id записей models.ServerSprRecords
// (аппаратные серверы Type=0, гипервизоры Type=2 и виртуальные машины
// Type=1), физически размещённых в шкафу rackCabinetId - используется
// фильтром "ЦОД/Шкаф" на /sprsrv/view2 (2026-09-08, по запросу пользователя
// "замени в форме /sprsrv/view2 ... схему с выбором ЦОД и Шкафа вместо ЦОД:
// адрес и площадка по старым данным").
//
// Правила определения места размещения:
//   - аппаратные серверы и гипервизоры (не блейд) - по прямому
//     models.ServerHardware.RackCabinetId;
//   - блейд-серверы (FormFactor==2, BladeSystemId!=0) - В ПРИОРИТЕТЕ по
//     размещению самой блейд-корзины (models.BladeSystems.RackCabinetId, по
//     BladeSystemId), а не по собственному ServerHardware.RackCabinetId -
//     если у корзины размещение не указано (RackCabinetId==0) или сама
//     корзина не найдена, используется запасной вариант - собственный
//     RackCabinetId сервера (тот же приём, что уже применяется для
//     PowerState/Active блейд-корзин, см. cmdb/initrec.go);
//   - виртуальные машины - по размещению их хоста (гипервизора): среди уже
//     найденных выше хостов (Type==2) ищем совпадение имени с
//     models.VMSprRecord.Vm_HostName (сравнение без учёта регистра с
//     Name/NameFQDN хоста - прямой FK-связи VM->хост в схеме нет, только
//     сопоставление по имени, см. cmdb/initrec.go:CopySprVmToVm/CopyVmToSprVm),
//     затем берём все ServerSprRecords с Type==1, чей VMSprId ссылается на
//     найденную запись models.VMSprRecord.
func placementServerSprIds(o orm.Ormer, rackCabinetId int64) []int64 {
	// --- аппаратные серверы/гипервизоры - прямое совпадение RackCabinetId
	// (сюда попадают и блейд-сервера с собственным RackCabinetId, если у их
	// корзины размещение не указано - см. ниже) -------------------------
	var directHw []models.ServerHardware
	o.QueryTable("server_hardware").Filter("RackCabinetId", rackCabinetId).All(&directHw, "ServerId")

	matchedServerIds := map[int64]bool{}
	for _, hw := range directHw {
		if hw.ServerId != 0 {
			matchedServerIds[hw.ServerId] = true
		}
	}

	// --- блейд-серверы: размещение самой корзины в приоритете -----------
	var bladeHw []models.ServerHardware
	o.QueryTable("server_hardware").Filter("FormFactor", 2).Filter("BladeSystemId__gt", 0).All(&bladeHw, "ServerId", "BladeSystemId")
	if len(bladeHw) > 0 {
		bsIds := make([]int64, 0, len(bladeHw))
		seenBs := map[int64]bool{}
		for _, hw := range bladeHw {
			if !seenBs[hw.BladeSystemId] {
				seenBs[hw.BladeSystemId] = true
				bsIds = append(bsIds, hw.BladeSystemId)
			}
		}
		var chassis []models.BladeSystems
		o.QueryTable(new(models.BladeSystems)).Filter("Id__in", bsIds).All(&chassis, "Id", "RackCabinetId")
		chassisRackById := make(map[int64]int64, len(chassis))
		for _, c := range chassis {
			chassisRackById[c.Id] = c.RackCabinetId
		}
		for _, hw := range bladeHw {
			if hw.ServerId == 0 {
				continue
			}
			chassisRack, known := chassisRackById[hw.BladeSystemId]
			if !known || chassisRack == 0 {
				// корзина не найдена/её размещение не указано - оставляем
				// как определено выше по собственному RackCabinetId сервера
				// (уже в matchedServerIds, если совпал, иначе не совпал).
				continue
			}
			if chassisRack == rackCabinetId {
				matchedServerIds[hw.ServerId] = true
			} else {
				// корзина размещена в ДРУГОМ шкафу - у размещения корзины
				// приоритет, исключаем сервер, даже если его собственный
				// RackCabinetId совпал с искомым выше.
				delete(matchedServerIds, hw.ServerId)
			}
		}
	}

	result := make([]int64, 0, len(matchedServerIds))
	for id := range matchedServerIds {
		result = append(result, id)
	}
	if len(result) == 0 {
		return result
	}

	// --- виртуальные машины - по размещению хоста ------------------------
	var hosts []models.ServerSprRecords
	o.QueryTable("server_spr_records").Filter("Id__in", result).Filter("Type", 2).All(&hosts, "Id", "Name", "NameFQDN")
	if len(hosts) == 0 {
		return result
	}

	hostNames := map[string]bool{}
	for _, h := range hosts {
		if n := strings.ToLower(strings.TrimSpace(h.Name)); n != "" {
			hostNames[n] = true
		}
		if n := strings.ToLower(strings.TrimSpace(h.NameFQDN)); n != "" {
			hostNames[n] = true
		}
	}
	if len(hostNames) == 0 {
		return result
	}

	var vms []models.VMSprRecord
	o.QueryTable("vm_spr_record").All(&vms, "Id", "Vm_HostName")
	vmIds := make([]int64, 0)
	for _, vm := range vms {
		n := strings.ToLower(strings.TrimSpace(vm.Vm_HostName))
		if n != "" && hostNames[n] {
			vmIds = append(vmIds, vm.Id)
		}
	}
	if len(vmIds) == 0 {
		return result
	}

	var vmSpr []models.ServerSprRecords
	o.QueryTable("server_spr_records").Filter("Type", 1).Filter("VMSprId__in", vmIds).All(&vmSpr, "Id")
	for _, s := range vmSpr {
		result = append(result, s.Id)
	}

	return result
}
