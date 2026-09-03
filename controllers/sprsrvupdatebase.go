package controllers

import (
	"fmt"
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
	if err := o.QueryTable("server_spr_records").Filter("Name", name).Filter("Type", 1).One(&rec); err != nil {
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

	backmode := this.Ctx.Input.Param(":backmode")
	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр карточки записи справочника - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("sprsrv/updatebase")
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

	// справочник ИТ-систем для select ITSystemKey (тот же источник, что и в
	// фильтре /sprsrv/view2)
	this.Data["ITSystemSpr"] = itsystem.ImportITSystemSpr()

	if this.Ctx.Input.Method() == "POST" {

		if !this.RequireEditReference() {
			this.Data["record"] = record
			return
		}

		form := models.ServerSprRecords{}
		if err := this.ParseForm(&form); err != nil {
			flash.Error("Ошибка распознавания формы: %v", err)
			flash.Store(&this.Controller)
			this.Data["record"] = record
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
		record.DPCOffice = form.DPCOffice
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

		// record.Active на этот момент ещё хранит значение из БД (form.Active
		// в него не копировался) - используем это, чтобы понять, изменился ли
		// статус активности, и только тогда проставить служебные поля
		prevActive := record.Active
		record.Active = form.Active

		if prevActive != record.Active {
			switch record.Active {
			case 1:
				record.TimeActiveState = now
				record.ActivatorUser = currentUserId
			case 2:
				record.TimeDeActiveState = now
				record.DeActivatorUser = currentUserId
			}
		}

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
			flash.Notice("Данные сервера \"%s\" сохранены", record.Name)
			flash.Store(&this.Controller)
			RegisterRequest(this, 6) //изменение данных справочника
		}
	}

	this.Data["record"] = record
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
