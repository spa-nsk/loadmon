package controllers

import (
	"fmt"
	"loadmon/models"
	"regexp"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// LocalityFilter - фильтр списка /admin/localities (вхождение без учёта регистра).
type LocalityFilter struct {
	Name       string `form:"Name"`
	Oktmo      string `form:"Oktmo"`
	RegionCode string `form:"RegionCode"`
	RegionName string `form:"RegionName"`
}

var reOktmo = regexp.MustCompile(`^\d{11}$`)

// adminLocalitiesAllowed - справочник населённых пунктов доступен только
// администратору (Priv==3), как и сама загрузка из ОКТМО.
func (this *MainController) adminLocalitiesAllowed() bool {
	sess := this.GetSession("loadmon")
	f, ok := sess.(models.LoadFilter)
	if !ok {
		return false
	}
	priv, _ := f.Map["privilage"].(int)
	return priv == 3
}

// AdminLocalities - список справочника населённых пунктов (models.Locality)
// с фильтром и постраничной навигацией.
//
// Маршрут: /admin/localities
func (this *MainController) AdminLocalities() {
	if !this.adminLocalitiesAllowed() {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}
	RegisterRequest(this, LogTypeViewReference)
	this.activeContent("admin/localitylist")
	flash := beego.NewFlash()

	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	const keyFilter = "LocalityFilter"
	filter, ok := m[keyFilter].(*LocalityFilter)
	if !ok {
		filter = &LocalityFilter{}
		m[keyFilter] = filter
	}

	page, pageSize := 1, 50
	if this.Ctx.Input.Method() == "POST" {
		if err := this.ParseForm(filter); err != nil {
			flash.Error("Ошибка распознавания формы фильтра: %v", err)
			flash.Store(&this.Controller)
		}
		if p, err := strconv.Atoi(this.GetString("page")); err == nil && p > 0 {
			page = p
		}
		if s, err := strconv.Atoi(this.GetString("pagesize")); err == nil {
			switch s {
			case 25, 50, 100, 200:
				pageSize = s
			}
		}
	}
	this.Data[keyFilter] = filter

	qs := orm.NewOrm().QueryTable("cmdb_locality")
	if v := strings.TrimSpace(filter.Name); v != "" {
		qs = qs.Filter("Name__icontains", v)
	}
	if v := strings.TrimSpace(filter.Oktmo); v != "" {
		qs = qs.Filter("Oktmo__startswith", v)
	}
	if v := strings.TrimSpace(filter.RegionCode); v != "" {
		qs = qs.Filter("RegionCode", v)
	}
	if v := strings.TrimSpace(filter.RegionName); v != "" {
		qs = qs.Filter("RegionName__icontains", v)
	}

	total, err := qs.Count()
	if err != nil {
		flash.Error("Ошибка выборки данных (возможно, справочник ещё не загружен): %v", err)
		flash.Store(&this.Controller)
		return
	}
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages < 1 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * pageSize

	var list []models.Locality
	if _, err := qs.OrderBy("RegionCode", "Name", "Oktmo").Limit(pageSize, offset).All(&list); err != nil {
		flash.Error("Ошибка выборки данных: %v", err)
		flash.Store(&this.Controller)
		return
	}

	this.Data["LocalityList"] = list
	this.Data["LocalityCount"] = total
	this.Data["Page"] = page
	this.Data["PageSize"] = pageSize
	this.Data["TotalPages"] = totalPages
	this.Data["PageFrom"] = int64(0)
	if total > 0 {
		this.Data["PageFrom"] = int64(offset) + 1
	}
	pageTo := int64(offset + pageSize)
	if pageTo > total {
		pageTo = total
	}
	this.Data["PageTo"] = pageTo
}

// AdminLocalityEdit - просмотр/редактирование одной записи справочника
// населённых пунктов; Id=0 - добавление новой. Параметр формы Delete=1 -
// удаление записи (если на её код не ссылаются площадки/ЦОД).
//
// Маршрут: /admin/localityedit/:id
func (this *MainController) AdminLocalityEdit() {
	if !this.adminLocalitiesAllowed() {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}
	RegisterRequest(this, LogTypeViewReference)
	this.activeContent("admin/localityedit")
	flash := beego.NewFlash()

	idStr := this.Ctx.Input.Param(":id")
	this.Data["Id"] = idStr
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}
	isNew := id == 0
	this.Data["IsNew"] = isNew

	o := orm.NewOrm()
	var rec models.Locality
	if !isNew {
		if err := o.QueryTable("cmdb_locality").Filter("Id", id).One(&rec); err != nil {
			flash.Error("Запись с Id=%d не найдена: %v", id, err)
			flash.Store(&this.Controller)
			return
		}
	}

	if this.Ctx.Input.Method() == "POST" {
		if this.GetString("Delete") == "1" && !isNew {
			used, _ := o.QueryTable("cmdb_datacenter_site").Filter("LocalityCode", rec.Oktmo).Count()
			if used > 0 {
				flash.Error("Нельзя удалить: код %s указан у площадок/ЦОД (%d).", rec.Oktmo, used)
				flash.Store(&this.Controller)
				this.Data["Loc"] = rec
				return
			}
			if _, err := o.Delete(&rec); err != nil {
				flash.Error("Ошибка удаления: %v", err)
				flash.Store(&this.Controller)
				this.Data["Loc"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.Redirect("/admin/localities", 302)
			return
		}

		oldOktmo := rec.Oktmo
		rec.Oktmo = strings.TrimSpace(this.GetString("Oktmo"))
		rec.RegionCode = strings.TrimSpace(this.GetString("RegionCode"))
		rec.RegionName = strings.TrimSpace(this.GetString("RegionName"))
		rec.Municipality = strings.TrimSpace(this.GetString("Municipality"))
		rec.TypeShort = strings.TrimSpace(this.GetString("TypeShort"))
		rec.Name = strings.TrimSpace(this.GetString("Name"))
		rec.FullName = strings.TrimSpace(this.GetString("FullName"))
		if rec.FullName == "" && rec.Name != "" {
			rec.FullName = strings.TrimSpace(rec.TypeShort + " " + rec.Name)
		}
		if rec.RegionCode == "" && len(rec.Oktmo) >= 2 {
			rec.RegionCode = rec.Oktmo[:2]
		}

		var problem string
		switch {
		case !reOktmo.MatchString(rec.Oktmo):
			problem = "Код ОКТМО должен состоять ровно из 11 цифр."
		case rec.Name == "":
			problem = "Не заполнено наименование."
		case rec.RegionName == "":
			problem = "Не заполнен субъект РФ."
		}
		if problem == "" {
			q := o.QueryTable("cmdb_locality").Filter("Oktmo", rec.Oktmo)
			if !isNew {
				q = q.Exclude("Id", rec.Id)
			}
			if n, _ := q.Count(); n > 0 {
				problem = fmt.Sprintf("Код ОКТМО %s уже есть в справочнике.", rec.Oktmo)
			}
		}
		if problem == "" && !isNew && oldOktmo != rec.Oktmo {
			if n, _ := o.QueryTable("cmdb_datacenter_site").Filter("LocalityCode", oldOktmo).Count(); n > 0 {
				problem = fmt.Sprintf("Код %s указан у площадок/ЦОД (%d) - изменить его нельзя.", oldOktmo, n)
			}
		}
		if problem != "" {
			flash.Error("%s", problem)
			flash.Store(&this.Controller)
			this.Data["Loc"] = rec
			return
		}

		if isNew {
			if _, err := o.Insert(&rec); err != nil {
				flash.Error("Ошибка создания записи: %v", err)
				flash.Store(&this.Controller)
				this.Data["Loc"] = rec
				return
			}
			RegisterRequest(this, LogTypeEditReference)
			this.Redirect(fmt.Sprintf("/admin/localityedit/%d", rec.Id), 302)
			return
		}
		if _, err := o.Update(&rec); err != nil {
			flash.Error("Ошибка сохранения: %v", err)
			flash.Store(&this.Controller)
			this.Data["Loc"] = rec
			return
		}
		RegisterRequest(this, LogTypeEditReference)
		flash.Notice("Запись сохранена.")
		flash.Store(&this.Controller)
	}
	this.Data["Loc"] = rec
}
