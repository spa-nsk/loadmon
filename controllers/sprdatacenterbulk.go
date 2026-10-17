package controllers

import (
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// SprDatacenterSiteUpdateSelected - групповое редактирование записей
// справочника Площадок/ЦОД (models.DatacenterSite), по образцу
// /sprsrv/updateselectedbase (справочник серверов, 2026-10-02). Площадки,
// отмеченные чекбоксами в /sprsrv/datacentersitelist (кнопка
// "Редактировать выделенные"), передаются в URL как разделённые "-" id.
// Применяются только заполненные поля: пустая строка и "Не менять" в select
// оставляют значение у каждой записи прежним.
//
// Наименование/альтернативное наименование в групповое редактирование не
// включены - это уникальные для каждой площадки данные.
//
// Маршрут: /sprsrv/datacentersiteupdateselected/:selected
func (this *MainController) SprDatacenterSiteUpdateSelected() {

	selected := this.Ctx.Input.Param(":selected")

	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/datacentersiteupdateselected")
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
		flash.Error("Не выбрано ни одной площадки для группового редактирования")
		flash.Store(&this.Controller)
		this.Redirect("/sprsrv/datacentersitelist/0", 302)
		return
	}

	o := orm.NewOrm()

	selectedRecords := make([]models.DatacenterSite, 0, len(ids))
	if _, err := o.QueryTable("cmdb_datacenter_site").Filter("Id__in", ids).OrderBy("Name").All(&selectedRecords); err != nil {
		flash.Error("Ошибка выборки выбранных площадок: %v", err)
		flash.Store(&this.Controller)
	}
	this.Data["SelectedRecords"] = selectedRecords
	this.Data["SelectedCount"] = len(selectedRecords)

	if this.Ctx.Input.Method() != "POST" {
		return
	}
	if !this.RequireEditReference() {
		return
	}

	// IsDatacenter: "0" - не менять, "1" - ЦОД, "2" - не ЦОД (площадка)
	isDatacenter := strings.TrimSpace(this.GetString("IsDatacenter"))
	codeJira := strings.TrimSpace(this.GetString("CodeJira"))
	annotation := strings.TrimSpace(this.GetString("Annotation"))

	address := strings.TrimSpace(this.GetString("Address"))
	floor := strings.TrimSpace(this.GetString("Floor"))

	if (isDatacenter == "" || isDatacenter == "0") && codeJira == "" && annotation == "" && address == "" && floor == "" {
		flash.Error("Не заполнено ни одно поле - нечего применять к выбранным площадкам")
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
		rec := models.DatacenterSite{Id: id}
		if err := o.Read(&rec); err != nil {
			lastErr = err
			continue
		}
		switch isDatacenter {
		case "1":
			rec.IsDatacenter = true
		case "2":
			rec.IsDatacenter = false
		}
		if codeJira != "" {
			rec.CodeJira = codeJira
		}
		if annotation != "" {
			rec.Annotation = annotation
		}
		if address != "" {
			rec.Address = address
		}
		if floor != "" {
			rec.Floor = floor
		}
		rec.Id_User = currentUserId
		rec.Time_Mfy = now
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
		flash.Error("Обновлено %d из %d площадок, часть записей не удалось сохранить: %v", updated, len(ids), lastErr)
	} else {
		flash.Notice("Обновлено площадок: %d", updated)
	}
	flash.Store(&this.Controller)
	RegisterRequest(this, LogTypeEditReference)

	this.Redirect("/sprsrv/datacentersitelist/0", 302)
}
