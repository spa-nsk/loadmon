package controllers

import (
	"fmt"
	"loadmon/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// NetworkEquipmentTypeRow - строка списка типов: тип + число устройств этого типа.
type NetworkEquipmentTypeRow struct {
	models.NetworkEquipmentType
	Count int64
}

// SprNetworkEquipmentTypes - список справочника типов сетевого оборудования
// (models.NetworkEquipmentType) с числом устройств каждого типа.
// Маршрут: /sprsrv/networkequipmenttypelist/:id
func (this *MainController) SprNetworkEquipmentTypes() {
	RegisterRequest(this, LogTypeViewReference)
	this.activeContent("sprsrv/networkequipmenttypelist")
	this.Data["ListId"] = this.Ctx.Input.Param(":id")
	flash := beego.NewFlash()

	o := orm.NewOrm()
	var types []models.NetworkEquipmentType
	if _, err := o.QueryTable(new(models.NetworkEquipmentType)).OrderBy("Name").All(&types); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}
	counts := map[int64]int64{}
	var raws []orm.Params
	if _, err := o.Raw("SELECT type_id AS tid, COUNT(*) AS cnt FROM switch_spr_records GROUP BY type_id").Values(&raws); err == nil {
		for _, r := range raws {
			tid, _ := strconv.ParseInt(fmt.Sprint(r["tid"]), 10, 64)
			cnt, _ := strconv.ParseInt(fmt.Sprint(r["cnt"]), 10, 64)
			counts[tid] = cnt
		}
	}
	rows := make([]NetworkEquipmentTypeRow, 0, len(types))
	for _, t := range types {
		rows = append(rows, NetworkEquipmentTypeRow{NetworkEquipmentType: t, Count: counts[t.Id]})
	}
	this.Data["TypeList"] = rows
	this.Data["TypeCount"] = len(rows)
}

// SprNetworkEquipmentTypeEdit - просмотр/редактирование/добавление типа сетевого
// оборудования (Id=0 - новый). Имя типа должно быть уникальным.
// Маршрут: /sprsrv/networkequipmenttypeedit/:id
func (this *MainController) SprNetworkEquipmentTypeEdit() {
	idStr := this.Ctx.Input.Param(":id")
	RegisterRequest(this, LogTypeViewReference)
	this.activeContent("sprsrv/networkequipmenttypeedit")
	this.Data["Id"] = idStr
	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}
	isNew := id == 0
	this.Data["IsNew"] = isNew

	o := orm.NewOrm()
	var rec models.NetworkEquipmentType
	if !isNew {
		if err := o.QueryTable(new(models.NetworkEquipmentType)).Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if !this.RequireEditReference() {
			this.Data["TP"] = rec
			return
		}
		name := strings.TrimSpace(this.GetString("Name"))
		rec.Annotation = this.GetString("Annotation")
		if name == "" {
			rec.Name = name
			flash.Error("Наименование типа не может быть пустым.")
			flash.Store(&this.Controller)
			this.Data["TP"] = rec
			return
		}
		dup := o.QueryTable(new(models.NetworkEquipmentType)).Filter("Name__iexact", name).Exclude("Id", rec.Id).Exist()
		rec.Name = name
		if dup {
			flash.Error("Тип с наименованием %q уже существует.", name)
			flash.Store(&this.Controller)
			this.Data["TP"] = rec
			return
		}
		if uid, ok := this.Data["UserId"].(int64); ok {
			rec.Id_User = uid
		}
		rec.Time_Mfy = time.Now().Local()

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["TP"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.Redirect(fmt.Sprintf("/sprsrv/networkequipmenttypeedit/%d", rec.Id), 302)
			return
		}
		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["TP"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference)
		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}

	if !isNew {
		this.Data["EquipmentCount"], _ = o.QueryTable("switch_spr_records").Filter("TypeId", rec.Id).Count()
	}
	this.Data["TP"] = rec
}
