package controllers

import (
	"loadmon/itsystem"
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// SprSrvUpdateSelectedBase - групповое редактирование записей справочника
// серверов (models.ServerSprRecords). Список серверов, отмеченных чекбоксами
// в /sprsrv/view2 (кнопка "Редактировать выделенные"), передаётся в URL как
// разделённые "-" id. На форме заполняются только те поля, которые нужно
// изменить у всех выбранных серверов; пустая строка и значение "не указано"
// в select считаются "не менять это поле" и не применяются.
//
// Идентификационные поля (Type, SerialNumber, Vcenter/Datacenter/Cluster,
// Name, NameFQDN, IP/MAC/Network) в групповое редактирование не включены -
// это уникальные для каждого сервера данные, которые не имеет смысла
// затирать одним и тем же значением у всех выбранных записей разом.
//
// Маршрут: /sprsrv/updateselectedbase/:selected
func (this *MainController) SprSrvUpdateSelectedBase() {

	selected := this.Ctx.Input.Param(":selected")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр формы группового редактирования - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("sprsrv/updateselectedbase")
	this.Data["selected"] = selected

	flash := beego.NewFlash()

	ids := make([]int64, 0)
	for _, s := range strings.Split(selected, "-") {
		id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		flash.Error("Не выбрано ни одного сервера для группового редактирования")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	o := orm.NewOrm()

	// список выбранных серверов - показываем, что именно будет изменено
	selectedRecords := make([]models.ServerSprRecords, 0, len(ids))
	qs := o.QueryTable("server_spr_records")
	qs = qs.Filter("Id__in", ids).OrderBy("Name")
	qs.All(&selectedRecords)

	this.Data["SelectedRecords"] = selectedRecords
	this.Data["SelectedCount"] = len(selectedRecords)
	this.Data["ITSystemSpr"] = itsystem.ImportITSystemSpr()

	if this.Ctx.Input.Method() == "POST" {

		if !this.RequireEditReference() {
			return
		}

		form := models.ServerSprRecords{}
		if err := this.ParseForm(&form); err != nil {
			flash.Error("Ошибка распознавания формы: %v", err)
			flash.Store(&this.Controller)
			return
		}

		trim := strings.TrimSpace

		typeMicroservice := form.TypeMicroservice // 0 = не менять
		roleType := form.RoleType                 // 0 = не менять
		typeWork := form.TypeWork                 // 0 = не менять
		criticalLevel := form.CriticalLevel       // 0 = не менять
		active := form.Active                     // 0 = не менять (значения 1/2)
		dpcOffice := form.DPCOffice               // 0 = не менять
		monitoring := form.Monitoring             // 0 = не менять
		backuped := form.Backuped                 // 0 = не менять

		itSystemKey := trim(form.ITSystemKey)
		osName := trim(form.OS)
		owner := trim(form.Owner)
		businessOwner := trim(form.BusinessOwner)
		itOwner := trim(form.ITOwner)
		productOwner := trim(form.ProductOwner)
		support := trim(form.Support)
		supportTeam := trim(form.SupportTeam)
		systemType := trim(form.SystemType)
		licensingType := trim(form.LicensingType)
		solutionType := trim(form.SolutionType)
		users := trim(form.Users)
		allocation := trim(form.Allocation)
		anotation := trim(form.Anotation)
		role := trim(form.Role)
		details := trim(form.Details)
		szHistory := trim(form.SZHistory)
		monitoringSZ := trim(form.MonitoringSZ)
		networkName := trim(form.NetworkName)

		// проверка заполненности: пустые строки и "не указано" (0) в select
		// считаются пустыми и не должны приводить к "успешному" сохранению,
		// если реально нечего применять
		anyFilled := typeMicroservice != 0 || roleType != 0 || typeWork != 0 || criticalLevel != 0 ||
			active != 0 || dpcOffice != 0 || monitoring != 0 || backuped != 0 ||
			itSystemKey != "" || osName != "" || owner != "" || businessOwner != "" || itOwner != "" ||
			productOwner != "" || support != "" || supportTeam != "" || systemType != "" ||
			licensingType != "" || solutionType != "" || users != "" || allocation != "" ||
			anotation != "" || role != "" || details != "" || szHistory != "" || monitoringSZ != "" ||
			networkName != ""

		if !anyFilled {
			flash.Error("Не заполнено ни одно поле - нечего применять к выбранным серверам")
			flash.Store(&this.Controller)
			return
		}

		var currentUserId int64
		if uid, ok := this.Data["UserId"].(int64); ok {
			currentUserId = uid
		}
		now := time.Now().Local()

		updated := 0
		var lastErr error

		for _, id := range ids {
			rec := models.ServerSprRecords{Id: id}
			if err := o.Read(&rec); err != nil {
				lastErr = err
				continue
			}

			if typeMicroservice != 0 {
				rec.TypeMicroservice = typeMicroservice
			}
			if roleType != 0 {
				rec.RoleType = roleType
			}
			if typeWork != 0 {
				rec.TypeWork = typeWork
			}
			if criticalLevel != 0 {
				rec.CriticalLevel = criticalLevel
			}
			if active != 0 {
				prevActive := rec.Active
				rec.Active = active
				if prevActive != rec.Active {
					switch rec.Active {
					case 1:
						rec.TimeActiveState = now
						rec.ActivatorUser = currentUserId
					case 2:
						rec.TimeDeActiveState = now
						rec.DeActivatorUser = currentUserId
					}
				}
			}
			if itSystemKey != "" {
				rec.ITSystemKey = itSystemKey
			}
			if osName != "" {
				rec.OS = osName
			}
			if owner != "" {
				rec.Owner = owner
			}
			if businessOwner != "" {
				rec.BusinessOwner = businessOwner
			}
			if itOwner != "" {
				rec.ITOwner = itOwner
			}
			if productOwner != "" {
				rec.ProductOwner = productOwner
			}
			if support != "" {
				rec.Support = support
			}
			if supportTeam != "" {
				rec.SupportTeam = supportTeam
			}
			if systemType != "" {
				rec.SystemType = systemType
			}
			if licensingType != "" {
				rec.LicensingType = licensingType
			}
			if solutionType != "" {
				rec.SolutionType = solutionType
			}
			if users != "" {
				rec.Users = users
			}
			if allocation != "" {
				rec.Allocation = allocation
			}
			if anotation != "" {
				rec.Anotation = anotation
			}
			if role != "" {
				rec.Role = role
			}
			if details != "" {
				rec.Details = details
			}
			if dpcOffice != 0 {
				rec.DPCOffice = dpcOffice
			}
			if szHistory != "" {
				rec.SZHistory = szHistory
			}
			if monitoring != 0 {
				rec.Monitoring = monitoring
				if monitoring == 1 && rec.TimeActiveMonitoring.IsZero() {
					rec.TimeActiveMonitoring = now
				}
			}
			if backuped != 0 {
				rec.Backuped = backuped
			}
			if monitoringSZ != "" {
				rec.MonitoringSZ = monitoringSZ
			}
			if networkName != "" {
				rec.NetworkName = networkName
			}

			rec.Id_User = currentUserId
			rec.Time_Mfy = now
			// SourceUpdateType намеренно НЕ трогаем здесь - см.
			// kontext/05-conventions-and-gotchas.md (раздел SourceUpdateType,
			// уточнение "массовые формы не проставляют"). Поле помечает
			// записи, внесённые вручную и отсутствующие в
			// VMware/OneView/Zabbix, чтобы код автообновления из этих систем
			// их игнорировал и не портил (например, не проставлял Stat=0/
			// PowerState=2/Active=2 записи, которой в источнике нет).
			// Массовое редактирование через эту форму задевает лишь узкий
			// набор полей у уже существующих записей произвольного
			// происхождения - простановка Manual здесь могла бы ошибочно
			// "рассекретить" от автообновления запись, которая на самом
			// деле числится в одной из автоматических систем.

			if _, err := o.Update(&rec); err != nil {
				lastErr = err
				continue
			}
			updated++
		}

		if updated == 0 {
			flash.Error("Не удалось обновить ни одной записи: %v", lastErr)
			flash.Store(&this.Controller)
			return
		}

		if updated < len(ids) {
			flash.Error("Обновлено %d из %d серверов, часть записей не удалось сохранить: %v", updated, len(ids), lastErr)
		} else {
			flash.Notice("Обновлено серверов: %d", updated)
		}
		flash.Store(&this.Controller)
		RegisterRequest(this, 6) //изменение данных справочника

		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}
}
