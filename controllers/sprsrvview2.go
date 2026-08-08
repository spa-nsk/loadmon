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

func (this *MainController) SprSrvViewV2() {

	Mode := this.Ctx.Input.Param(":mode")
	Id := this.Ctx.Input.Param(":id")
	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_view2"+"_"+Mode+"_"+Id, 302)
		return
	}
	this.activeContent("sprsrv/view2")
	this.Data["ListId"] = Id
	this.Data["backmode"] = Mode

	var (
		records      []*models.ServerSprRecords
		recordsAdd   []*models.ServerSprRecords
		recordsVMAdd []*models.ServerSprRecords
		list         []models.ServerSprView
	)

	indexSrv := make(map[string]*models.ServerSprRecords)

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

	cond = cond.And("stat__ne", 666)
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
			cond = cond.And("active__ne", 2)
		case 999:
		}

		record.Name = strings.TrimSpace(record.Name)

		if record.Name != "" {
			condName := orm.NewCondition()
			names := strings.Split(record.Name, ";")
			for m := range names {
				s := strings.TrimSpace(names[m])
				if s != "" {
					condName = condName.Or("name__icontains", s)
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
					condName = condName.Or("name_info__icontains", s)
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
					condName = condName.Or("name_f_q_d_n__icontains", s)
				}
			}
			cond = cond.AndCond(condName)
		}

		if record.ITSystemKey != "" {
			cond = cond.And("i_t_system_key", record.ITSystemKey)
		}

		//         if record.DPCAddress != 0 {
		//             cond = cond.And("d_p_c_address", record.DPCAddress)
		//         }
		if record.DPCOffice != 0 {
			cond = cond.And("d_p_c_office", record.DPCOffice)
		}
		if record.RoleType != 0 {
			cond = cond.And("role_type", record.RoleType)
		}

		if record.Type != 999 {
			cond = cond.And("type", record.Type)
		}

		if record.TypeMicroservice != 999 {
			cond = cond.And("type_microservice", record.TypeMicroservice)
		}

		if record.TypeWork != 0 {
			cond = cond.And("type_work", record.TypeWork)
		}

		if record.CriticalLevel != 0 {
			cond = cond.And("critical_level", record.CriticalLevel)
		}
		//		if record.ServiceId != 0 {
		//			cond = cond.And("service_id", record.ServiceId)
		//		}

		if strings.TrimSpace(record.ITSystemKey) != "" {
			cond = cond.And("i_t_system_key", record.ITSystemKey)
		}

		//		if strings.TrimSpace(record.Service) != "" {
		//			cond = cond.And("service__icontains", record.Service)
		//		}
		if strings.TrimSpace(record.Anotation) != "" {
			cond = cond.And("anotation__icontains", record.Anotation)
		}
		if strings.TrimSpace(record.Role) != "" {
			cond = cond.And("role__icontains", record.Role)
		}

		record.Owner = strings.TrimSpace(record.Owner)

		if record.Owner != "" {
			condOwner := orm.NewCondition()
			owners := strings.Split(record.Owner, ";")
			for m := range owners {
				s := strings.TrimSpace(owners[m])
				if s != "" {
					condOwner = condOwner.Or("owner__icontains", s)
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
			cond = cond.And("os__eq", "")
		} else if record.OS != "" {
			condName := orm.NewCondition()
			os := strings.Split(record.OS, ";")
			for m := range os {
				s := strings.TrimSpace(os[m])
				if s != "" {
					condName = condName.And("os__icontains", s)
				}
			}
			cond = cond.AndCond(condName)
		}

		if record.PowerState != 0 {
			cond = cond.And("power_state", record.PowerState)
		}

		if strings.TrimSpace(record.Network) != "" {
			cond = cond.And("network__icontains", record.Network)
		}

		if strings.TrimSpace(record.NetworkName) != "" {
			cond = cond.And("network_name__icontains", record.NetworkName)
		}

		if strings.TrimSpace(record.IPAddress) != "" {
			cond = cond.And("i_p_address__icontains", record.IPAddress)
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

	if err != orm.ErrNoRows && num > 0 {
		list = make([]models.ServerSprView, 0)
		for k, rec := range records {
			indexSrv[rec.Name+"/"+rec.NameFQDN+"/"+rec.IPAddress] = records[k]
			r := models.ServerSprView{}
			r.S = *rec

			qs = o.QueryTable("spr_cfo_allocation")
			qs = qs.OrderBy("server_id")
			qs = qs.Filter("server_id", rec.Id)
			qs.All(&r.CFO)

			qs = o.QueryTable("spr_busines_allocation")
			qs = qs.OrderBy("server_id")
			qs = qs.Filter("server_id", rec.Id)
			qs.All(&r.Busines)
			if rec.VMSprId != 0 && rec.Type == 1 {
				vm := models.VMSprRecordView{}
				qs = o.QueryTable("vm_spr_record").OrderBy("id").Filter("id", rec.VMSprId)
				err := qs.One(&vm.VMSprRecord)
				if err == nil {
					r.VM = &vm
				} else {
					//				r.VM = nil
					r.VM = &models.VMSprRecordView{}
				}

			} else {
				//			r.VM = nil
				r.VM = &models.VMSprRecordView{}
			}
			if rec.Type == 2 || rec.Type == 0 {
				hw := models.ServerHardwareView{}
				qs = o.QueryTable("server_hardware").OrderBy("server_id").Filter("server_id", rec.Id)
				err := qs.One(&hw.H)
				if err == nil {
					r.Hardware = &hw
				} else {
					r.Hardware = &models.ServerHardwareView{}
				}
			} else {
				r.Hardware = &models.ServerHardwareView{}
			}
			CalcPercentFilled(&r)
			list = append(list, r)
		}
	}

	if errAdd == nil && numAdd > 0 { // добавляем из расширенного поиска
		for k, rec := range recordsAdd {
			_, ok := indexSrv[rec.Name+"/"+rec.NameFQDN+"/"+rec.IPAddress]
			if ok {
				continue
			}
			indexSrv[rec.Name+"/"+rec.NameFQDN+"/"+rec.IPAddress] = recordsAdd[k]
			r := models.ServerSprView{}
			r.S = *rec

			qs = o.QueryTable("spr_cfo_allocation")
			qs = qs.OrderBy("server_id")
			qs = qs.Filter("server_id", rec.Id)
			qs.All(&r.CFO)

			qs = o.QueryTable("spr_busines_allocation")
			qs = qs.OrderBy("server_id")
			qs = qs.Filter("server_id", rec.Id)
			qs.All(&r.Busines)
			if rec.VMSprId != 0 && rec.Type == 1 {
				vm := models.VMSprRecordView{}
				qs = o.QueryTable("vm_spr_record").OrderBy("id").Filter("id", rec.VMSprId)
				err := qs.One(&vm.VMSprRecord)
				if err == nil {
					r.VM = &vm
				} else {
					r.VM = &models.VMSprRecordView{}
				}
			} else {
				r.VM = &models.VMSprRecordView{}
			}
			if rec.Type == 2 || rec.Type == 0 {
				hw := models.ServerHardwareView{}
				qs = o.QueryTable("server_hardware").OrderBy("server_id").Filter("server_id", rec.Id)
				err := qs.One(&hw.H)
				if err == nil {
					r.Hardware = &hw
				} else {
					r.Hardware = &models.ServerHardwareView{}
				}
			} else {
				r.Hardware = &models.ServerHardwareView{}
			}
			CalcPercentFilled(&r)
			list = append(list, r)
		}
	}

	if errVMAdd == nil && numVMAdd > 0 { //добавляем из расширенного поиска
		for k, rec := range recordsVMAdd {
			_, ok := indexSrv[rec.Name+"/"+rec.NameFQDN+"/"+rec.IPAddress]
			if ok {
				continue
			}
			indexSrv[rec.Name+"/"+rec.NameFQDN+"/"+rec.IPAddress] = recordsVMAdd[k]
			r := models.ServerSprView{}
			r.S = *rec
			qs = o.QueryTable("spr_cfo_allocation")
			qs = qs.OrderBy("server_id")
			qs = qs.Filter("server_id", rec.Id)
			qs.All(&r.CFO)

			qs = o.QueryTable("spr_busines_allocation")
			qs = qs.OrderBy("server_id")
			qs = qs.Filter("server_id", rec.Id)
			qs.All(&r.Busines)
			if rec.VMSprId != 0 && rec.Type == 1 {
				vm := models.VMSprRecordView{}
				qs = o.QueryTable("vm_spr_record").OrderBy("id").Filter("id", rec.VMSprId)
				err := qs.One(&vm.VMSprRecord)
				if err == nil {
					r.VM = &vm
				} else {
					r.VM = &models.VMSprRecordView{}
				}

			} else {
				r.VM = &models.VMSprRecordView{}
			}
			if rec.Type == 2 || rec.Type == 0 {
				hw := models.ServerHardwareView{}
				qs = o.QueryTable("server_hardware").OrderBy("server_id").Filter("server_id", rec.Id)
				err := qs.One(&hw.H)
				if err == nil {
					r.Hardware = &hw
				} else {
					r.Hardware = &models.ServerHardwareView{}
				}
			} else {
				r.Hardware = &models.ServerHardwareView{}
			}
			CalcPercentFilled(&r)
			list = append(list, r)
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
			csvwriter.Comma = ';'
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
