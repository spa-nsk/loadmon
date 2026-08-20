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

// SprAddServer отображает и сохраняет форму создания новой записи справочника
// серверов (models.ServerSprRecords). Использует тот же шаблон и тот же набор
// редактируемых полей, что и SprSrvUpdateBase, отличие - вставка новой записи
// (o.Insert) вместо обновления существующей, плюс служебные поля создания
// (Stat, TimeCreate, CreatorUser).
//
// Маршрут: /sprsrv/addserver/:backmode/:name
// :backmode - зарезервирован, текущее значение всегда "all"
// :name - зарезервирован, текущее значение всегда "new"
func (this *MainController) SprAddServer() {

	backmode := this.Ctx.Input.Param(":backmode")
	name := this.Ctx.Input.Param(":name")

	sess := this.GetSession("loadmon")
	if sess == nil {
		this.Redirect("/user/login/sprsrv_addserver_"+backmode+"_"+name, 302)
		return
	}
	RegisterRequest(this, LogTypeViewReference) // просмотр формы добавления записи - отдельная запись типа "изменение" пишется ниже при сохранении

	// используем тот же шаблон, что и форма редактирования - набор полей
	// идентичен, отличается только heading (см. .IsNew) и действие при отправке
	this.activeContent("sprsrv/updatebase")
	this.Data["backmode"] = backmode
	this.Data["IsNew"] = true

	flash := beego.NewFlash()

	// значения по умолчанию для новой записи
	record := models.ServerSprRecords{
		Type:   0,
		Active: 1,
	}

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

		// тот же набор редактируемых полей, что и в SprSrvUpdateBase
		record.Type = form.Type

		// см. пояснение в controllers/sprsrvupdatebase.go: серийный номер -
		// только для физического сервера (0) и гипервизора (2), vCenter/
		// Датацентр/Кластер - только для виртуальной машины (1) и
		// гипервизора (2). Для новой записи "недоступное" поле просто
		// остаётся пустым.
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
		record.Active = form.Active

		record.Name = strings.TrimSpace(form.Name)
		record.NameFQDN = strings.TrimSpace(form.NameFQDN)
		record.NameInfo = strings.TrimSpace(form.NameInfo)
		record.IPAddress = strings.TrimSpace(form.IPAddress)
		record.MacAddress = strings.TrimSpace(form.MacAddress)
		record.Network = strings.TrimSpace(form.Network)
		record.NetworkName = strings.TrimSpace(form.NetworkName)
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

		if record.Name == "" {
			flash.Error("Не заполнено обязательное поле \"Имя сервера\"")
			flash.Store(&this.Controller)
			this.Data["record"] = record
			return
		}

		now := time.Now().Local()

		var currentUserId int64
		if uid, ok := this.Data["UserId"].(int64); ok {
			currentUserId = uid
		}

		// служебные поля новой записи
		record.Stat = 1 // 1 - используется
		record.TimeCreate = now
		record.CreatorUser = currentUserId
		record.Id_User = currentUserId
		record.Time_Mfy = now

		switch record.Active {
		case 1:
			record.TimeActiveState = now
			record.ActivatorUser = currentUserId
		case 2:
			record.TimeDeActiveState = now
			record.DeActivatorUser = currentUserId
		}

		if record.Monitoring == 1 {
			record.TimeActiveMonitoring = now
		}

		o := orm.NewOrm()
		newId, err := o.Insert(&record)
		if err != nil {
			flash.Error("Ошибка создания записи: %v", err)
			flash.Store(&this.Controller)
			this.Data["record"] = record
			return
		}
		record.Id = newId

		flash.Notice("Сервер \"%s\" добавлен", record.Name)
		flash.Store(&this.Controller)
		RegisterRequest(this, 6) //изменение данных справочника

		this.Redirect("/sprsrv/updatebase/"+backmode+"/"+strconv.FormatInt(newId, 10), 302)
		return
	}

	this.Data["record"] = record
}
