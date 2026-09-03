package controllers

import (
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// AdminDeactivatedServersList отображает список записей справочника серверов
// (models.ServerSprRecords), выведенных из учёта автоматической сверкой
// (Stat == 0 - см. kontext/05-conventions-and-gotchas.md, раздел
// "cmdb.markServerSprAbsent и родственный код" - записи, которые
// авто-обновление из VMware/OneView/Zabbix посчитало пропавшими из
// источника и пометило Stat=0/PowerState=2/Active=2).
//
// Назначение страницы - ручной разбор: часть таких записей выведена из
// учёта ОШИБОЧНО (временная недоступность источника на момент сверки,
// смена идентификатора и т.п.), и администратор должен иметь возможность
// вернуть их в работу — как по одной, так и группой. Сама простановка
// новых значений и связанных записей (VM/аппаратный сервер) выполняется в
// AdminDeactivatedServersRestore ниже, эта функция только показывает
// список и собирает выбор чекбоксами (тот же приём, что и в
// /sprsrv/view2 → /sprsrv/updateselectedbase, см.
// controllers/sprsrvview2.go и sprsrvupdateselectedbase.go).
//
// Доступ - только models.AuthUser.Priv == 3, как /admin/userlog и
// /admin/metricsexport.
//
// Маршрут: /admin/deactivatedservers.
func (this *MainController) AdminDeactivatedServersList() {

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/deactivatedserverslist")

	flash := beego.NewFlash()

	nameFilter := strings.TrimSpace(this.GetString("name"))
	typeFilter := this.GetString("type") // "" / "all" - без фильтра по типу
	this.Data["CurName"] = nameFilter
	this.Data["CurType"] = typeFilter

	o := orm.NewOrm()

	// Список серверов, отмеченных чекбоксами, и нажатие кнопки "Восстановить
	// выделенные" - собираем выбор и перенаправляем на форму восстановления
	// (аналог кнопки "Редактировать выделенные" на /sprsrv/view2, см.
	// controllers/sprsrvview2.go).
	if this.Ctx.Input.Method() == "POST" && this.GetString("RestoreSelectedBTN") != "" {
		var candidates []*models.ServerSprRecords
		o.QueryTable("server_spr_records").Filter("Stat", 0).All(&candidates)

		strList := ""
		for _, s := range candidates {
			if s == nil {
				continue
			}
			strId := strconv.FormatInt(s.Id, 10)
			if this.GetString("chSelection"+strId) == "on" {
				if strList == "" {
					strList = strId
				} else {
					strList += "-" + strId
				}
			}
		}
		if strList == "" {
			flash.Error("Выберите хотя бы один сервер для восстановления")
			flash.Store(&this.Controller)
		} else {
			this.Redirect("/admin/deactivatedservers/restore/"+strList, 302)
			return
		}
	}

	RegisterRequest(this, LogTypeAdminAction) // просмотр списка серверов, выведенных из учёта (администрирование)

	qs := o.QueryTable("server_spr_records").Filter("Stat", 0)
	if nameFilter != "" {
		qs = qs.Filter("Name__icontains", nameFilter)
	}
	if typeFilter != "" && typeFilter != "all" {
		if v, err := strconv.Atoi(typeFilter); err == nil {
			qs = qs.Filter("Type", v)
		}
	}
	qs = qs.OrderBy("-Time_Mfy")

	var all []*models.ServerSprRecords
	if _, err := qs.All(&all); err != nil {
		flash.Error("Ошибка чтения справочника серверов: %v", err)
		flash.Store(&this.Controller)
		all = nil
	}

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
	totalRecords := len(all)
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
	pageRows := make([]*models.ServerSprRecords, 0)
	if offset < totalRecords {
		pageRows = all[offset:end]
	}

	this.Data["Rows"] = pageRows
	this.Data["RowCount"] = totalRecords
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	if totalRecords == 0 {
		this.Data["PageFrom"] = 0
		this.Data["PageTo"] = 0
	} else {
		this.Data["PageFrom"] = offset + 1
		this.Data["PageTo"] = end
	}
}

// AdminDeactivatedServersRestore показывает форму (GET) и применяет (POST)
// возврат в работу выбранных записей models.ServerSprRecords, ранее
// выведенных из учёта (Stat == 0) - как по одной, так и группой
// (:selected - разделённые "-" id, тот же формат, что и в
// /sprsrv/updateselectedbase/:selected).
//
// Логика возврата - точное зеркало логики вывода из учёта
// (cmdb.markServerSprAbsent и родственные функции в cmdb/checkserversprexistence.go,
// cmdb/updatehwhostinfo.go и т.п. - см. kontext/05-conventions-and-gotchas.md):
// там при выводе из учёта одновременно проставляются Stat=0/PowerState=2/
// Active=2, здесь при возврате - наоборот, Stat выставляется в 1 ТОЛЬКО
// когда итоговые PowerState/Active после применения формы означают
// "включен и активен" (PowerState==1 && Active==1) - то есть недостаточно
// просто открыть форму, нужно явно подтвердить оба значения.
//
// SourceUpdateType самой записи ServerSprRecords этой формой НЕ
// редактируется (наследуется - остаётся тем, что было) - согласно общей
// договорённости не затирать источник записи в формах редактирования
// (см. kontext/05-conventions-and-gotchas.md, раздел SourceUpdateType).
// При каскадном обновлении связанных записей текущее значение
// SourceUpdateType записи ServerSprRecords переносится ("наследуется") на
// связанную models.ServerHardware, чтобы источник был согласован между
// связанными записями.
//
// Связанные записи, которые правятся при фактическом возврате в работу
// (Stat стал 1):
//   - models.VMSprRecord (если Type == 1 и VMSprId > 0) - Vm_PowerState
//     выставляется в соответствии с PowerState (это единственное поле
//     жизненного цикла VM в vm_spr_record - у неё нет Stat/Active).
//   - models.ServerHardware (если HWSprId > 0) - PowerState/Active
//     синхронизируются с ServerSprRecords, SourceUpdateType наследуется
//     от ServerSprRecords (см. выше).
//
// Доступ - только models.AuthUser.Priv == 3.
//
// Маршрут: /admin/deactivatedservers/restore/:selected.
func (this *MainController) AdminDeactivatedServersRestore() {

	selected := this.Ctx.Input.Param(":selected")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	RegisterRequest(this, LogTypeViewReference) // просмотр формы возврата в учёт - отдельная запись типа "изменение" пишется ниже при сохранении

	this.activeContent("admin/deactivatedserversrestore")
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
		flash.Error("Не выбрано ни одного сервера для восстановления")
		flash.Store(&this.Controller)
		this.Redirect("/admin/deactivatedservers", 302)
		return
	}

	o := orm.NewOrm()

	selectedRecords := make([]*models.ServerSprRecords, 0, len(ids))
	qs := o.QueryTable("server_spr_records").Filter("Id__in", ids).OrderBy("Name")
	qs.All(&selectedRecords)

	this.Data["SelectedRecords"] = selectedRecords
	this.Data["SelectedCount"] = len(selectedRecords)

	if this.Ctx.Input.Method() != "POST" {
		return
	}

	// PowerState/Active выбираются на форме явно (select без "не менять" по
	// умолчанию предзаполнен на "включен"/"активен" - см. шаблон), но 0
	// ("не указано"/"не менять") по-прежнему поддержан на случай, если
	// администратор хочет поменять только часть выбранных записей.
	powerState, _ := strconv.Atoi(this.GetString("PowerState"))
	active, _ := strconv.Atoi(this.GetString("Active"))

	if powerState == 0 && active == 0 {
		flash.Error("Не выбрано ни одного значения (PowerState/Active) - нечего применять")
		flash.Store(&this.Controller)
		return
	}

	var currentUserId int64
	if uid, ok := this.Data["UserId"].(int64); ok {
		currentUserId = uid
	}
	now := time.Now().Local()

	restored, updated := 0, 0
	var lastErr error

	for _, id := range ids {
		rec := models.ServerSprRecords{Id: id}
		if err := o.Read(&rec); err != nil {
			lastErr = err
			continue
		}

		if powerState != 0 {
			rec.PowerState = powerState
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

		wasDeactivated := rec.Stat == 0
		if rec.PowerState == 1 && rec.Active == 1 {
			rec.Stat = 1 // возврат в учёт - см. комментарий к функции выше
		}

		rec.Id_User = currentUserId
		rec.Time_Mfy = now
		// SourceUpdateType самой записи НЕ трогаем - наследуется как есть
		// (см. комментарий к функции выше и kontext/05-conventions-and-gotchas.md).

		if _, err := o.Update(&rec); err != nil {
			lastErr = err
			continue
		}
		updated++

		if wasDeactivated && rec.Stat == 1 {
			restored++

			if rec.Type == 1 && rec.VMSprId > 0 {
				vm := models.VMSprRecord{Id: rec.VMSprId}
				if err := o.Read(&vm); err == nil {
					vm.Vm_PowerState = rec.PowerState == 1
					o.Update(&vm, "Vm_PowerState")
				}
			}

			if rec.HWSprId > 0 {
				hw := models.ServerHardware{Id: rec.HWSprId}
				if err := o.Read(&hw); err == nil {
					prevHwActive := hw.Active
					hw.PowerState = rec.PowerState
					hw.Active = rec.Active
					hw.SourceUpdateType = rec.SourceUpdateType // наследуется от ServerSprRecords, см. комментарий к функции выше
					hw.Id_User = currentUserId
					hw.Time_Mfy = now
					if prevHwActive != hw.Active {
						switch hw.Active {
						case 1:
							hw.TimeActiveState = now
							hw.ActivatorUser = currentUserId
						case 2:
							hw.TimeDeActiveState = now
							hw.DeActivatorUser = currentUserId
						}
					}
					o.Update(&hw)
				}
			}
		}
	}

	if updated == 0 {
		flash.Error("Не удалось обновить ни одной записи: %v", lastErr)
		flash.Store(&this.Controller)
		return
	}

	if updated < len(ids) {
		flash.Error("Обновлено %d из %d записей (возвращено в учёт: %d), часть записей не удалось сохранить: %v", updated, len(ids), restored, lastErr)
	} else if restored < updated {
		flash.Notice("Обновлено записей: %d, возвращено в учёт (Stat=1): %d - у остальных PowerState/Active ещё не означают полный возврат в работу", updated, restored)
	} else {
		flash.Notice("Возвращено в учёт: %d", restored)
	}
	flash.Store(&this.Controller)
	RegisterRequest(this, LogTypeEditReference) // возврат в учёт ранее выведенных серверов (администрирование)

	this.Redirect("/admin/deactivatedservers", 302)
}
