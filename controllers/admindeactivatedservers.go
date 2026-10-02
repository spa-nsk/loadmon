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
// (models.ServerSprRecords), выведенных из эксплуатации (Active == 2) -
// автоматически (единое правило Stat/PowerState/Active, сервер не найден ни
// в одном источнике: Stat=1/PowerState=2/Active=2, см.
// cmdb/reconcileserversprtype.go) или вручную. До 2026-10-02 страница
// отбирала Stat == 0: тогда автосверка помечала отсутствующие записи
// Stat=0, теперь Stat=0 ставит только человек ("историческая запись").
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
		models.ServerSprQS(o).Filter("Active", 2).All(&candidates)

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

	qs := models.ServerSprQS(o).Filter("Active", 2)
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
// выведенных из эксплуатации (Active == 2) - как по одной, так и группой
// (:selected - разделённые "-" id, тот же формат, что и в
// /sprsrv/updateselectedbase/:selected).
//
// Возврат в эксплуатацию (2026-10-02) - смена Active с 2 на 1 через форму.
// Stat=1 у таких записей уже стоит (единое правило ставит "не найденным"
// Stat=1/PowerState=2/Active=2); историческая запись (Stat=0), если её
// вернули в эксплуатацию, получает Stat=1. Учтите: если сервер не найден ни
// в одном источнике и не ручной, единое правило снова поставит Active=2.
//
// SourceUpdateType записи ServerSprRecords в этой форме РЕДАКТИРУЕМ (по
// прямой просьбе пользователя - обычная договорённость "формы
// редактирования не трогают SourceUpdateType", см.
// kontext/05-conventions-and-gotchas.md, здесь сознательно нарушена, так
// как страница именно для РУЧНОГО разбора ошибочных авто-обновлений, и
// администратору может быть нужно поправить и сам признак источника, не
// только PowerState/Active). Поле - как и PowerState/Active - работает по
// принципу "пусто = не менять": select на форме включает отдельный пункт
// "не менять" (пустое значение), собственно "не указано" (0) - это уже
// самостоятельный вариант выбора, а не отсутствие выбора. При каскадном
// обновлении связанных записей ИТОГОВОЕ (после применения формы) значение
// SourceUpdateType записи ServerSprRecords переносится ("наследуется") на
// связанную models.ServerHardware, чтобы источник был согласован между
// связанными записями - это касается только каскада на ServerHardware, у
// самой ServerSprRecords значение теперь может быть изменено явно.
//
// Связанные записи, которые правятся при фактическом возврате в работу
// (Active стал 1):
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
	qs := models.ServerSprQS(o).Filter("Id__in", ids).OrderBy("Name")
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
	//
	// SourceUpdateType - отдельный принцип "пусто = не менять": в отличие от
	// PowerState/Active (где 0 совмещает роль "не менять" и "не указано"),
	// у SourceUpdateType 0 - самостоятельное значение "не указано" (см.
	// models.SourceUpdateType* константы), поэтому "не менять" на форме -
	// это ПУСТАЯ строка, а не "0", и распознаётся отдельно (sourceUpdateTypeSet).
	powerState, _ := strconv.Atoi(this.GetString("PowerState"))
	active, _ := strconv.Atoi(this.GetString("Active"))
	sourceUpdateTypeStr := strings.TrimSpace(this.GetString("SourceUpdateType"))
	sourceUpdateTypeSet := sourceUpdateTypeStr != ""
	sourceUpdateType, _ := strconv.Atoi(sourceUpdateTypeStr)

	if powerState == 0 && active == 0 && !sourceUpdateTypeSet {
		flash.Error("Не выбрано ни одного значения (PowerState/Active/SourceUpdateType) - нечего применять")
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
		if models.IsServerSprDestroyed(&rec) {
			continue // помечена к уничтожению (Stat=666) - не восстанавливается и не изменяется
		}

		// Выведена из эксплуатации ДО применения формы (Active=2) - с
		// 2026-10-02 признак вывода - Active, а не Stat (см. комментарий к
		// AdminDeactivatedServersList).
		wasDeactivated := rec.Active == 2

		if powerState != 0 {
			rec.PowerState = powerState
		}
		if active != 0 {
			// 2026-09-22: audit-helper вместо ручного sравнения
			// prevActive/switch - currentUserId это реальный Id
			// залогиненного пользователя (массовая ручная операция
			// восстановления/деактивации).
			models.SetActiveWithAudit(&rec.Active, &rec.TimeActiveState, &rec.ActivatorUser, &rec.TimeDeActiveState, &rec.DeActivatorUser, active, currentUserId)
		}

		if wasDeactivated && rec.Active == 1 && rec.Stat == models.ServerSprStatHistory {
			rec.Stat = models.ServerSprStatUsed // возврат в работу исторической записи
		}

		if sourceUpdateTypeSet {
			rec.SourceUpdateType = sourceUpdateType
		}

		rec.Id_User = currentUserId
		rec.Time_Mfy = now

		if _, err := o.Update(&rec); err != nil {
			lastErr = err
			continue
		}
		updated++

		if wasDeactivated && rec.Active == 1 {
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
					hw.PowerState = rec.PowerState
					hw.SourceUpdateType = rec.SourceUpdateType // синхронизируется с итоговым (после формы) значением ServerSprRecords, см. комментарий к функции выше
					hw.Id_User = currentUserId
					hw.Time_Mfy = now
					// 2026-09-22: audit-helper вместо ручного sравнения
					// prevHwActive/switch - синхронизация статуса
					// ServerHardware со статусом связанной ServerSprRecords,
					// поэтому пользователь тот же (currentUserId).
					models.SetActiveWithAudit(&hw.Active, &hw.TimeActiveState, &hw.ActivatorUser, &hw.TimeDeActiveState, &hw.DeActivatorUser, rec.Active, currentUserId)
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
		flash.Notice("Обновлено записей: %d, возвращено в эксплуатацию (Active=1): %d", updated, restored)
	} else {
		flash.Notice("Возвращено в учёт: %d", restored)
	}
	flash.Store(&this.Controller)
	RegisterRequest(this, LogTypeEditReference) // возврат в учёт ранее выведенных серверов (администрирование)

	this.Redirect("/admin/deactivatedservers", 302)
}
