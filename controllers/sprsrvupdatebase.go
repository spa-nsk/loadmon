package controllers

import (
	"fmt"
	"loadmon/auditlog"
	"loadmon/grpcclient"
	"loadmon/itsystem"
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// SprSrvUpdateBase отображает и сохраняет форму редактирования общих данных
// записи справочника серверов (models.ServerSprRecords). Значения для
// справочников (Type, TypeMicroservice, RoleType, TypeWork, CriticalLevel,
// Active, DPCOffice, Monitoring, Backuped) берутся из комментариев к полям
// структуры в models/models_add.go.
//
// SprSrvUpdateBaseByVMName находит запись справочника серверов
// (models.ServerSprRecords) по имени виртуальной машины (Name, Type=1) и
// переходит на её форму общих данных /sprsrv/updatebase/all/:id - аналог
// SprHwConfigEditByName для гипервизоров/физических серверов, но для ВМ,
// у которых нет собственной записи в models.ServerHardware, поэтому
// переход идёт сразу на её собственную ServerSprRecords.Id, а не на
// связанную запись оборудования.
//
// Если запись с таким именем не найдена, пользователь перенаправляется на
// общий список /sprsrv/view2/all/0 с сообщением в flash.
//
// Маршрут: /sprsrv/vmeditbyname/:Name
func (this *MainController) SprSrvUpdateBaseByVMName() {
	name := this.Ctx.Input.Param(":Name")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeServerDetails) // переход к деталям ВМ по имени

	flash := beego.NewFlash()

	o := orm.NewOrm()
	var rec models.ServerSprRecords
	if err := models.ServerSprQS(o).Filter("Name", name).Filter("Type", 1).One(&rec); err != nil {
		flash.Notice("Виртуальная машина с именем \"%s\" не найдена в справочнике серверов.", name)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.Redirect(fmt.Sprintf("/sprsrv/updatebase/all/%d", rec.Id), 302)
}

// Маршрут: /sprsrv/updatebase/:backmode/:id([0-9]+)
// :backmode - зарезервирован, текущее значение всегда "all"
// :id - models.ServerSprRecords.Id редактируемой записи
func (this *MainController) SprSrvUpdateBase() {
	sprSrvUpdateBasePage(this, "sprsrv/updatebase", this.Ctx.Input.Param(":backmode"), this.Ctx.Input.Param(":id"))
}

// SprSrvUpdateBaseTab - та же карточка справочника серверов, что
// SprSrvUpdateBase, но с формой на закладках (шаблон updatebase_tab.tpl).
// Отдельный маршрут для пробного UI, никуда в меню/ссылки не вставляется.
//
// Маршрут: /sprsrv/updatebase-tab/:id([0-9]+)
func (this *MainController) SprSrvUpdateBaseTab() {
	sprSrvUpdateBasePage(this, "sprsrv/updatebase_tab", "all", this.Ctx.Input.Param(":id"))
}

// sprSrvUpdateBasePage - общая логика GET/POST формы общих данных
// ServerSprRecords. tpl - имя шаблона (sprsrv/updatebase или
// sprsrv/updatebase_tab), backmode - для ссылки «Отмена» на view2.
func sprSrvUpdateBasePage(this *MainController, tpl, backmode, idStr string) {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки записи справочника - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent(tpl)
	this.Data["backmode"] = backmode

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/view2/"+backmode+"/0", 302)
		return
	}

	o := orm.NewOrm()

	record := models.ServerSprRecords{Id: id}
	if err := o.Read(&record); err != nil {
		flash.Error("Запись с id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/view2/"+backmode+"/0", 302)
		return
	}
	if models.IsServerSprDestroyed(&record) {
		flash.Error("Запись с id=%d помечена к уничтожению (Stat=%d) и не редактируется", id, models.ServerSprStatDestroy)
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/view2/"+backmode+"/0", 302)
		return
	}

	// before - снимок ДО правки формой ниже, нужен только для diff'а при
	// сохранении (см. auditlog.LogServerSprRecordChange, вызов перед
	// o.Update ниже) - история изменений карточки сервера
	// (models.ServerSprEventLog, добавлено 2026-09-04 по запросу "сделай
	// аналогичную историю изменений карточки серверов... с сохранением
	// даты изменений и авторов").
	before := record

	// справочник ИТ-систем для select ITSystemKey (тот же источник, что и в
	// фильтре /sprsrv/view2)
	this.Data["ITSystemSpr"] = itsystem.ImportITSystemSpr()

	// Дата создания/изменения и автор последних изменений - показываются в
	// карточке (updatebase.tpl) как read-only справка, по прямому запросу
	// пользователя. Id_User/CreatorUser - это Id пользователя
	// (models.AuthUser), сам шаблон к БД доступа не имеет, поэтому имя
	// резолвится здесь.
	this.Data["ModifiedByName"] = auditlog.ResolveUserDisplayName(o, record.Id_User)
	this.Data["CreatedByName"] = auditlog.ResolveUserDisplayName(o, record.CreatorUser)

	// Размещение (ЦОД/Шкаф из нового справочника RackCabinet/DatacenterSite) -
	// read-only справка в карточке, по запросу пользователя ("проверь
	// корректность отображения... при просмотре не отображалось" -
	// на этой форме размещение вообще не показывалось, в отличие от
	// hwconflist/hwconfedit/storagesystemsedit/view2, куда оно уже перенесено
	// с прежнего текстового DPCOffice). Само поле RackCabinetId живёт на
	// models.ServerHardware (для Type 0/2) или резолвится через хост (для
	// VM, Type 1) - на этой форме оно НЕ редактируется (редактирование
	// остаётся на /sprsrv/hwconfedit/:HWSprId, чтобы не дублировать
	// источник истины) - здесь только отображение со ссылкой на карточку
	// оборудования.
	// Для VM (Type=1) размещение раньше резолвилось ЗДЕСЬ ЖЕ, синхронно -
	// resolveVMPlacementText делает "живой" gRPC-запрос к vCenter
	// (grpcclient.GetVMByKeyDirect, см. её комментарий), который заметно
	// медленнее локального чтения и ощутимо тормозил открытие страницы -
	// по прямому запросу пользователя (2026-09-18, "это замедляет открытие
	// страницы, сделай запрос асинхронным... Пока запрос выполняется в поле
	// можно отображать что идет загрузка данных. Если страницу закроют до
	// получения ответа, то данные будут утрачены и это нормально") эта
	// ветка вынесена в отдельный AJAX-эндпоинт SprSrvVMPlacementTextAsync
	// (маршрут /sprsrv/vmplacementtext/:id, вызывается из JS в
	// updatebase.tpl ПОСЛЕ отрисовки страницы) - здесь для VM только
	// проставляется признак PlacementAsync, по которому шаблон рисует
	// плейсхолдер "загрузка..." вместо блокирующего ожидания результата.
	// Для физического сервера/гипервизора (Type 0/2) размещение
	// резолвится из уже загрученных в БД данных (без похода в сеть) -
	// там синхронный путь остаётся как был, тормозить нечему.
	this.Data["PlacementText"] = ""
	this.Data["PlacementHWSprId"] = int64(0)
	this.Data["PlacementAsync"] = false
	switch record.Type {
	case 0, 2:
		if record.HWSprId > 0 {
			this.Data["PlacementHWSprId"] = record.HWSprId
			this.Data["PlacementText"] = resolveServerPlacementText(o, record.HWSprId)
		}
	case 1:
		if record.VMSprId > 0 {
			this.Data["PlacementAsync"] = true
		}
	}

	if this.Ctx.Input.Method() == "POST" {

		if !this.RequireEditReference() {
			this.Data["record"] = record
			setUpdateBaseCardHeader(this, &record)
			return
		}

		form := models.ServerSprRecords{}
		if err := this.ParseForm(&form); err != nil {
			flash.Error("Ошибка распознавания формы: %v", err)
			flash.Store(&this.Controller)
			this.Data["record"] = record
			setUpdateBaseCardHeader(this, &record)
			return
		}

		// переносим в запись только поля, доступные для редактирования на этой
		// форме; служебные поля (Id, OldId, Stat, ссылки VMSprId/HWSprId/
		// ZabbixHostId, метрики CPU/Mem/Disk, время создания и т.п.) не трогаем
		record.Type = form.Type

		// Серийный номер имеет смысл только для физического сервера (0) и
		// гипервизора (2); vCenter/Датацентр/Кластер - только для виртуальной
		// машины (1) и гипервизора (2). Если для текущего Type поле
		// недоступно для редактирования (см. JS в updatebase.tpl), значение
		// из формы игнорируется и в записи остаётся то, что было в БД -
		// это одновременно и защита от прямого POST в обход интерфейса.
		allowSerialNumber := record.Type == 0 || record.Type == 2
		allowVCenterFields := record.Type == 1 || record.Type == 2

		if allowSerialNumber {
			record.SerialNumber = strings.TrimSpace(form.SerialNumber)
		}
		if allowVCenterFields {
			record.Vcenter = strings.TrimSpace(form.Vcenter)
			record.Datacenter = strings.TrimSpace(form.Datacenter)
			record.Cluster = strings.TrimSpace(form.Cluster)
		}

		record.TypeMicroservice = form.TypeMicroservice
		record.RoleType = form.RoleType
		record.TypeWork = form.TypeWork
		record.CriticalLevel = form.CriticalLevel

		record.Name = strings.TrimSpace(form.Name)
		record.NameFQDN = strings.TrimSpace(form.NameFQDN)
		record.NameInfo = strings.TrimSpace(form.NameInfo)
		record.IPAddress = strings.TrimSpace(form.IPAddress)
		record.MacAddress = strings.TrimSpace(form.MacAddress)
		record.Network = strings.TrimSpace(form.Network)
		record.NetworkName = strings.TrimSpace(form.NetworkName)

		// запоминаем прежнее значение ИТ-системы до перезаписи - нужно ниже,
		// чтобы понять, изменилось ли оно, и решить, нужна ли автоматическая
		// корректировка зависимых полей через correctServerSprRecord
		prevITSystemKey := record.ITSystemKey
		record.ITSystemKey = strings.TrimSpace(form.ITSystemKey)
		record.OS = strings.TrimSpace(form.OS)
		record.Owner = strings.TrimSpace(form.Owner)
		record.BusinessOwner = strings.TrimSpace(form.BusinessOwner)
		record.ITOwner = strings.TrimSpace(form.ITOwner)
		record.ProductOwner = strings.TrimSpace(form.ProductOwner)
		record.Support = strings.TrimSpace(form.Support)
		record.SupportTeam = strings.TrimSpace(form.SupportTeam)
		record.SystemType = strings.TrimSpace(form.SystemType)
		record.LicensingType = strings.TrimSpace(form.LicensingType)
		record.SolutionType = strings.TrimSpace(form.SolutionType)
		record.Users = strings.TrimSpace(form.Users)
		record.Allocation = strings.TrimSpace(form.Allocation)
		record.Anotation = strings.TrimSpace(form.Anotation)
		record.Role = strings.TrimSpace(form.Role)
		record.Details = strings.TrimSpace(form.Details)
		// record.DPCOffice больше НЕ читается из формы (2026-09-10, по
		// запросу пользователя "убери поле ввода ЦОД: адрес и площадка -
		// в этом блоке уже есть отображение правильного размещения") -
		// select с DPCOffice убран из updatebase.tpl, старое размещение
		// (ЦОД/Шкаф по новой схеме) показывается через PlacementText выше.
		// Если оставить `record.DPCOffice = form.DPCOffice` без изменений,
		// отсутствующее в форме поле дало бы form.DPCOffice==0 и затёрло бы
		// уже сохранённое значение нулём при каждом сохранении карточки -
		// то есть DPCOffice просто не трогается этой формой вовсе (поле
		// оставлено в модели/БД как есть, историческое значение сохраняется).
		record.SZHistory = strings.TrimSpace(form.SZHistory)
		record.Monitoring = form.Monitoring
		record.Backuped = form.Backuped
		record.MonitoringSZ = strings.TrimSpace(form.MonitoringSZ)

		// Если поле "ИТ система" изменилось - подтягиваем актуальные данные по
		// выбранной ИТ-системе из локального справочника (itsystem.GetITSystemRec,
		// наполняется itsystem/jira.go) и корректируем зависимые поля записи
		// (критичность, признаки мониторинга/резервного копирования,
		// ответственные, тип системы и т.п.) через correctServerSprRecord -
		// функция была объявлена ранее, но нигде не вызывалась. Значения из
		// itsysRec намеренно перезаписывают то, что пришло из формы для тех же
		// полей (BusinessOwner/ITOwner/Owner/ProductOwner/Support/SupportTeam/
		// SystemType/LicensingType/SolutionType/Users/Allocation/CriticalLevel/
		// Monitoring/Backuped выше в этой же функции) - источником истины для
		// них должна быть выбранная ИТ-система, а не то, что ранее было
		// сохранено/введено вручную. Если код не найден в справочнике - запись
		// сохраняется как есть, без корректировки (пользователь уже видит
		// подсказку "код не найден" в форме - см. updatebase.tpl).
		if record.ITSystemKey != "" && record.ITSystemKey != prevITSystemKey {
			if itsysRec, ok := itsystem.GetITSystemRec(record.ITSystemKey); ok {
				correctServerSprRecord(&record, &itsysRec)
			}
		}

		now := time.Now().Local()

		var currentUserId int64
		if uid, ok := this.Data["UserId"].(int64); ok {
			currentUserId = uid
		}

		// 2026-09-22: используем общий audit-helper (models.SetActiveWithAudit)
		// вместо ручного sравнения prevActive/switch - логика та же (менять
		// TimeActiveState/ActivatorUser или TimeDeActiveState/DeActivatorUser
		// только при фактической смене статуса), но теперь общая для всех
		// мест простановки Active по проекту. currentUserId - реальный Id
		// залогиненного пользователя, вносящего изменение вручную через форму.
		models.SetActiveWithAudit(&record.Active, &record.TimeActiveState, &record.ActivatorUser, &record.TimeDeActiveState, &record.DeActivatorUser, form.Active, currentUserId)

		if record.Monitoring == 1 && record.TimeActiveMonitoring.IsZero() {
			record.TimeActiveMonitoring = now
		}

		record.Id_User = currentUserId
		record.Time_Mfy = now

		if id == 0 { //для серверов созданных вручную (Id=0) SourceUpdateType всегда SourceUpdateManual, чтобы не затирать шаблон при ручном редактировании
			record.SourceUpdateType = models.SourceUpdateManual
		}

		if _, err := o.Update(&record); err != nil {
			flash.Error("Ошибка сохранения записи: %v", err)
			flash.Store(&this.Controller)
		} else {
			// История изменений карточки (models.ServerSprEventLog) -
			// пишется ПОСЛЕ успешного сохранения, чтобы не логировать
			// изменение, которое на самом деле не применилось. Уведомления
			// по email для этого журнала не отправляются (прямое указание
			// пользователя, см. cmdb/serversprchangelog.go).
			auditlog.LogServerSprRecordChange(o, &before, &record)
			this.Data["ModifiedByName"] = auditlog.ResolveUserDisplayName(o, record.Id_User)

			flash.Notice("Данные сервера \"%s\" сохранены", record.Name)
			flash.Store(&this.Controller)
			RegisterRequest(this, 6) //изменение данных справочника
		}
	}

	this.Data["record"] = record
	setUpdateBaseCardHeader(this, &record)
}

// setUpdateBaseCardHeader кладёт в Data имя для заголовка карточки
// (FQDN → Name → NameInfo) и краткую строку CPU/RAM/DISK — для HW и VM
// одинаково, из полей models.ServerSprRecords. Используется и на страницах
// метрик (zabbixperfperiod / vmperfperiod / vmperfday) + в PDF-экспорте.
func setUpdateBaseCardHeader(c *MainController, record *models.ServerSprRecords) {
	if c == nil {
		return
	}
	if record == nil {
		c.Data["ServerTitleName"] = ""
		c.Data["ServerResourcesText"] = ""
		return
	}
	c.Data["ServerTitleName"] = serverSprDisplayName(record)
	c.Data["ServerResourcesText"] = serverSprResourcesText(record)
}

// loadServerSprByID читает ServerSprRecords по Id (для шапки метрик Zabbix).
func loadServerSprByID(o orm.Ormer, id int64) *models.ServerSprRecords {
	if id <= 0 {
		return nil
	}
	rec := models.ServerSprRecords{Id: id}
	if err := o.Read(&rec); err != nil {
		return nil
	}
	if models.IsServerSprDestroyed(&rec) {
		return nil
	}
	return &rec
}

// loadServerSprByVMSprID читает ServerSprRecords Type=1 по ссылке на vm_spr_record.
func loadServerSprByVMSprID(o orm.Ormer, vmSprId int64) *models.ServerSprRecords {
	if vmSprId <= 0 {
		return nil
	}
	var rec models.ServerSprRecords
	if err := models.ServerSprQS(o).Filter("VMSprId", vmSprId).Filter("Type", 1).One(&rec); err != nil {
		return nil
	}
	return &rec
}

// serverSprDisplayName — предпочтительно FQDN, иначе короткое имя, иначе
// альтернативное (NameInfo).
func serverSprDisplayName(r *models.ServerSprRecords) string {
	if r == nil {
		return ""
	}
	if s := strings.TrimSpace(r.NameFQDN); s != "" {
		return s
	}
	if s := strings.TrimSpace(r.Name); s != "" {
		return s
	}
	return strings.TrimSpace(r.NameInfo)
}

// serverSprResourcesText — «CPU: N ядер / RAM: X ГБ / DISK: Y ГБ» для шапки
// карточки. Ядра: CPULogical (логич./ядра), иначе CPUCount; память и диск в ГБ
// (как заполняется в cmdb/initrec из VMware/OneView).
func serverSprResourcesText(r *models.ServerSprRecords) string {
	if r == nil {
		return ""
	}
	cpu := r.CPULogical
	if cpu <= 0 {
		cpu = r.CPUCount
	}
	return fmt.Sprintf("CPU: %s / RAM: %s / DISK: %s",
		formatCPUCores(cpu), formatSizeGB(r.MemSize), formatSizeGB(r.DiskSize))
}

func formatCPUCores(v float64) string {
	if v <= 0 {
		return "—"
	}
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f ядер", v)
	}
	return fmt.Sprintf("%.1f ядер", v)
}

func formatSizeGB(v float64) string {
	if v <= 0 {
		return "—"
	}
	if v == float64(int64(v)) {
		return fmt.Sprintf("%.0f ГБ", v)
	}
	return fmt.Sprintf("%.1f ГБ", v)
}

// resolveServerPlacementText возвращает read-only текст размещения
// "ЦОД / Шкаф" для аппаратного сервера/гипервизора по Id его записи
// models.ServerHardware (ServerSprRecords.HWSprId), либо "" если у него нет
// связанной ServerHardware или размещение (RackCabinetId) не проставлено.
// Используется только для отображения на /sprsrv/updatebase - тот же
// справочник RackCabinet/DatacenterSite, что и в hwconflist/view2, но здесь
// без редактирования (редактирование - на /sprsrv/hwconfedit/:HWSprId).
func resolveServerPlacementText(o orm.Ormer, hwSprId int64) string {
	if hwSprId <= 0 {
		return ""
	}
	var hw models.ServerHardware
	if err := o.QueryTable(new(models.ServerHardware)).Filter("Id", hwSprId).One(&hw, "Id", "RackCabinetId"); err != nil {
		return ""
	}
	return placementTextByCabinetId(o, hw.RackCabinetId)
}

// resolveVMPlacementText возвращает read-only текст размещения для
// виртуальной машины - у неё самой нет физического размещения, оно
// наследуется от хоста (гипервизора), на котором она выполняется В МОМЕНТ
// ЗАПРОСА страницы (2026-09-15, по прямому запросу пользователя "добавь...
// отображение размещения для виртуальных машин исходя из местоположения
// гипервизора, на котором она выполняется, в момент запроса").
//
// Раньше здесь использовалась локальная models.VMSprRecord.Vm_HostName
// (обновляется только очередным фоновым опросом vSphere, могла отставать) -
// и, что важнее, это ИМЯ ГОСТЕВОЙ ОС внутри VM, а НЕ хост ESXi (та же
// ошибка, что была найдена и исправлена пользователем в фильтре
// /sprsrv/view2, см. CHANGELOG (67); задел под неё - поле
// models.VMSprRecord.Vm_EsxHost - здесь намеренно не используется, т.к.
// пользователь прямо просит показывать состояние "в момент запроса", а не
// последнее известное из фонового опроса).
//
// Вместо этого делается ЖИВОЙ запрос к vCenter через
// grpcclient.GetVMByKeyDirect (см. её комментарий - отдельное govmomi-
// соединение, минуя закэшированный в памяти пул сборщика) по Vcenter/
// Datacenter/Cluster/Vm_Name записи справочника ВМ - получаем актуальный на
// сейчас VMPoint.EsxHost (имя ESXi-хоста), а дальше ищем размещение этого
// хоста как обычно - по его собственной записи ServerSprRecords (Type=2) и
// связанной ServerHardware.RackCabinetId. Прямой запрос к vCenter заметно
// медленнее кэшированного чтения, но карточка сервера открывается по одной
// записи за раз, а не массово - см. предупреждение в комментарии
// GetVMByKeyDirect о недопустимости массовых вызовов.
//
// ИЗМЕНЕНО 2026-09-18: этот "живой" запрос замедлял открытие страницы
// /sprsrv/updatebase, поэтому вызов вынесен из основного рендера страницы в
// отдельный AJAX-эндпоинт SprSrvVMPlacementTextAsync (см. её комментарий) -
// сама функция при этом не изменилась по сути, кроме возврата ВТОРЫМ
// значением placementHWSprId (Id ServerHardware хоста, нужен вызывающему
// коду для ссылки "изменить в карточке оборудования") - раньше это
// значение нигде не требовалось отдельно от текста, теперь требуется в
// JSON-ответе эндпоинта, и доставать его вторым отдельным вызовом означало
// бы задваивать самый медленный шаг (сам gRPC-запрос к vCenter), поэтому
// функция просто возвращает оба значения из одного прохода.
func resolveVMPlacementText(o orm.Ormer, vmSprId int64) (text string, placementHWSprId int64) {
	if vmSprId <= 0 {
		return "", 0
	}
	var vm models.VMSprRecord
	if err := o.QueryTable(new(models.VMSprRecord)).Filter("Id", vmSprId).
		One(&vm, "Vcenter", "Datacenter", "Cluster", "Vm_Name"); err != nil {
		return "", 0
	}
	if strings.TrimSpace(vm.Vcenter) == "" || strings.TrimSpace(vm.Vm_Name) == "" {
		return "", 0
	}

	live, err := grpcclient.GetVMByKeyDirect(vm.Vcenter, vm.Datacenter, vm.Cluster, vm.Vm_Name)
	if err != nil {
		return "не удалось определить (ошибка обращения к vCenter: " + err.Error() + ")", 0
	}
	hostName := strings.TrimSpace(live.EsxHost)
	if hostName == "" {
		return "", 0
	}

	var host models.ServerSprRecords
	cond := orm.NewCondition().
		And("Type", 2).
		AndCond(orm.NewCondition().Or("Name__iexact", hostName).Or("NameFQDN__iexact", hostName))
	if err := models.ServerSprQS(o).SetCond(models.ExcludeDestroyedCond(cond)).One(&host, "Id", "Name", "HWSprId"); err != nil {
		return "хост \"" + hostName + "\" (не найден в справочнике серверов)", 0
	}
	if host.HWSprId <= 0 {
		return "хост \"" + host.Name + "\" (размещение хоста не указано)", 0
	}
	placeText := resolveServerPlacementText(o, host.HWSprId)
	if placeText == "" {
		return "хост \"" + host.Name + "\" (размещение хоста не указано)", host.HWSprId
	}
	return placeText + " (унаследовано от хоста \"" + host.Name + "\")", host.HWSprId
}

// placementTextByCabinetId форматирует "ЦОД / Шкаф" по Id шкафа
// (models.RackCabinet.Id), либо "" если шкаф не указан/не найден.
func placementTextByCabinetId(o orm.Ormer, rackCabinetId int64) string {
	if rackCabinetId <= 0 {
		return ""
	}
	var rc models.RackCabinet
	if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", rackCabinetId).One(&rc, "Id", "Name", "DatacenterSiteId"); err != nil {
		return ""
	}
	siteName := ""
	if rc.DatacenterSiteId > 0 {
		var site models.DatacenterSite
		if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", rc.DatacenterSiteId).One(&site, "Name"); err == nil {
			siteName = site.Name
		}
	}
	if siteName == "" {
		return rc.Name
	}
	return siteName + " / " + rc.Name
}

func correctServerSprRecord(record *models.ServerSprRecords, itsysRec *models.ItSystemRecord) {
	switch itsysRec.LevelCriticality {
	case "Критически важные (Vital)":
		record.CriticalLevel = 1
	case "Критические для бизнеса (Critical)":
		record.CriticalLevel = 2
	case "Важные для бизнеса (Secondary)":
		record.CriticalLevel = 3
	case "Поддерживающие (Non-priority)":
		record.CriticalLevel = 4
	}

	if itsysRec.IsMonitored {
		record.Monitoring = 1
	} else {
		record.Monitoring = 2
	}

	if itsysRec.IsBackuped {
		record.Backuped = 1
	} else {
		record.Backuped = 2
	}

	objAttr, err := itsystem.GetITSystemObjectAttributes(itsysRec.Key)
	if err == nil {
		record.BusinessOwner = itsystem.GetAttributes(objAttr, "Business-owner")
		record.ITOwner = itsystem.GetAttributes(objAttr, "IT-owner")
		record.Owner = record.ITOwner
		record.ProductOwner = itsystem.GetAttributes(objAttr, "Product owners")
		record.Support = itsystem.GetAttributes(objAttr, "Support")
		record.SupportTeam = itsystem.GetAttributes(objAttr, "Support team")
		record.SystemType = itsystem.GetAttributes(objAttr, "System type")
		record.LicensingType = itsystem.GetAttributes(objAttr, "Licensing type")
		record.SolutionType = itsystem.GetAttributes(objAttr, "Solution type")
		record.Users = itsystem.GetAttributes(objAttr, "IT-System users")
		record.Allocation = itsystem.GetAttributes(objAttr, "Business column")
	}
}

// SprSrvVMPlacementTextAsync отдаёт JSON с текстом физического размещения
// виртуальной машины (унаследованного от хоста, на котором она выполняется
// В МОМЕНТ ЗАПРОСА - см. resolveVMPlacementText) - вынесено из
// SprSrvUpdateBase в отдельный AJAX-эндпоинт по прямому запросу
// пользователя (2026-09-18, "делается grpc запрос. Это замедляет открытие
// страницы, сделай запрос асинхронным и он может выполняться после
// открытия страницы и обновлять отображение по факту загрузки данных...
// Если страницу закроют до получения ответа, то данные будут утрачены и
// это нормально") - resolveVMPlacementText делает "живой" gRPC-запрос к
// vCenter (заметно медленнее локального чтения), поэтому страница
// /sprsrv/updatebase больше не ждёт его результата перед отдачей HTML:
// JS в updatebase.tpl вызывает этот маршрут ПОСЛЕ того, как страница уже
// отрисована и показана пользователю, и подставляет результат в поле
// "Физическое размещение" по готовности; специального сохранения/повтора
// на случай закрытия вкладки до ответа НЕ предусмотрено - результат
// используется только для отображения "на лету" и никуда не персистится,
// поэтому его потеря при закрытии страницы безвредна.
//
// Ответ: {"placementText":"...", "placementHWSprId":N} - placementText
// пустой, если размещение определить не удалось (та же логика, что была
// раньше синхронно в SprSrvUpdateBase).
//
// Маршрут: /sprsrv/vmplacementtext/:id (GET), :id - models.ServerSprRecords.Id
func (this *MainController) SprSrvVMPlacementTextAsync() {
	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Data["json"] = map[string]interface{}{"placementText": ""}
		this.ServeJSON()
		return
	}

	idStr := this.Ctx.Input.Param(":id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		this.Data["json"] = map[string]interface{}{"placementText": ""}
		this.ServeJSON()
		return
	}

	o := orm.NewOrm()
	var record models.ServerSprRecords
	if err := models.ServerSprQS(o).Filter("Id", id).One(&record, "Id", "Type", "VMSprId"); err != nil {
		this.Data["json"] = map[string]interface{}{"placementText": ""}
		this.ServeJSON()
		return
	}
	if record.Type != 1 || record.VMSprId <= 0 {
		this.Data["json"] = map[string]interface{}{"placementText": ""}
		this.ServeJSON()
		return
	}

	placementText, placementHWSprId := resolveVMPlacementText(o, record.VMSprId)

	this.Data["json"] = map[string]interface{}{
		"placementText":    placementText,
		"placementHWSprId": placementHWSprId,
	}
	this.ServeJSON()
}

// SprITSystemCorrectPreview отдаёт JSON с результатом correctServerSprRecord
// для переданного кода ИТ-системы - используется формой /sprsrv/updatebase
// (см. JS-обработчик выбора ИТ-системы в updatebase.tpl), чтобы показать
// пользователю пересчитанные поля (критичность, мониторинг, резервное
// копирование, ответственные, тип/лицензирование/вариант решения,
// пользователи, аллокация) сразу при выборе ИТ-системы в форме - до нажатия
// "Сохранить", а не только после него, когда те же поля пересчитывает сам
// SprSrvUpdateBase при сохранении. Логика намеренно та же самая
// (correctServerSprRecord вызывается на пустой записи, чтобы получить только
// то, что она сама выставляет, без примеси текущих значений формы).
//
// Ответ: {"found":false} - код не найден в справочнике (itsystem.GetITSystemRec);
// иначе {"found":true, "criticalLevel":N, "monitoring":N, "backuped":N,
// "businessOwner":"...", "itOwner":"...", "owner":"...", "productOwner":"...",
// "support":"...", "supportTeam":"...", "systemType":"...",
// "licensingType":"...", "solutionType":"...", "users":"...", "allocation":"..."}.
//
// Маршрут: /sprsrv/itsystemcorrectpreview/:key (GET)
func (this *MainController) SprITSystemCorrectPreview() {
	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Data["json"] = map[string]interface{}{"found": false, "error": "не авторизован"}
		this.ServeJSON()
		return
	}

	key := strings.TrimSpace(this.Ctx.Input.Param(":key"))
	if key == "" {
		this.Data["json"] = map[string]interface{}{"found": false}
		this.ServeJSON()
		return
	}

	itsysRec, ok := itsystem.GetITSystemRec(key)
	if !ok {
		this.Data["json"] = map[string]interface{}{"found": false}
		this.ServeJSON()
		return
	}

	var rec models.ServerSprRecords
	correctServerSprRecord(&rec, &itsysRec)

	this.Data["json"] = map[string]interface{}{
		"found":         true,
		"criticalLevel": rec.CriticalLevel,
		"monitoring":    rec.Monitoring,
		"backuped":      rec.Backuped,
		"businessOwner": rec.BusinessOwner,
		"itOwner":       rec.ITOwner,
		"owner":         rec.Owner,
		"productOwner":  rec.ProductOwner,
		"support":       rec.Support,
		"supportTeam":   rec.SupportTeam,
		"systemType":    rec.SystemType,
		"licensingType": rec.LicensingType,
		"solutionType":  rec.SolutionType,
		"users":         rec.Users,
		"allocation":    rec.Allocation,
	}
	this.ServeJSON()
}
