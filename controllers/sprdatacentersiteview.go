package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"loadmon/rackpower"
	"sort"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// dcViewRack - шкаф в просмотре площадки/ЦОД.
type dcViewRack struct {
	Id         int64   `json:"id"`
	Name       string  `json:"name"`
	UHeight    int     `json:"uHeight"`
	Servers    int     `json:"servers"`  // rack-серверы
	Blades     int     `json:"blades"`   // блейды в корзинах шкафа
	Chassis    int     `json:"chassis"`  // блейд-корзины
	Storages   int     `json:"storages"` // СХД (с размещением в этом шкафу)
	PowerAvgW  float64 `json:"powerAvgW"`
	PowerPeakW float64 `json:"powerPeakW"`
	PowerRated float64 `json:"powerRatedW"`
}

type dcViewData struct {
	SiteId     int64        `json:"siteId"`
	SiteName   string       `json:"siteName"`
	Racks      []dcViewRack `json:"racks"`
	PowerAvgW  float64      `json:"powerAvgW"`
	PowerPeakW float64      `json:"powerPeakW"`
}

// SprDatacenterSiteView - просмотр оборудования площадки/ЦОД: слева
// изометрическая проекция машинного зала со шкафами (подсказка при наведении:
// номер шкафа, число серверов/корзин/СХД, мощность), справа - выбранный шкаф
// крупно (существующий просмотр шкафа /sprsrv/rackcabinetview/:id во фрейме).
//
// Вызывается из контекстного меню /sprsrv/datacentersitelist.
//
// Маршрут: /sprsrv/datacentersiteview/:id
func (this *MainController) SprDatacenterSiteView() {
	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewLoadTable)
	this.activeContent("sprsrv/datacentersiteview")

	this.Data["Id"] = idStr
	this.Data["SiteName"] = ""
	this.Data["RackCount"] = 0
	this.Data["DCJSON"] = template.JS("null")

	flash := beego.NewFlash()
	defer func() {
		if len(flash.Data) > 0 {
			flash.Store(&this.Controller)
		}
	}()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор площадки: %s", idStr)
		return
	}
	o := orm.NewOrm()

	var site models.DatacenterSite
	if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", id).One(&site); err != nil {
		flash.Error("Площадка Id=%d не найдена: %v", id, err)
		return
	}
	this.Data["SiteName"] = site.Name

	var racks []models.RackCabinet
	if _, err := o.QueryTable(new(models.RackCabinet)).Filter("DatacenterSiteId", id).OrderBy("Name").All(&racks); err != nil {
		flash.Error("Шкафы не загружены: %v", err)
		return
	}
	data := dcViewData{SiteId: id, SiteName: site.Name, Racks: make([]dcViewRack, 0, len(racks))}
	this.Data["RackCount"] = len(racks)
	if len(racks) == 0 {
		flash.Notice("В этой площадке нет шкафов")
		return
	}

	ids := make([]int64, 0, len(racks))
	byId := make(map[int64]*dcViewRack, len(racks))
	for _, r := range racks {
		u := int(r.UHeight)
		if u <= 0 {
			u = 42
		}
		data.Racks = append(data.Racks, dcViewRack{Id: r.Id, Name: r.Name, UHeight: u})
		ids = append(ids, r.Id)
	}
	for i := range data.Racks {
		byId[data.Racks[i].Id] = &data.Racks[i]
	}

	// rack-серверы
	var hws []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).Filter("RackCabinetId__in", ids).Exclude("Active", 2).
		All(&hws, "Id", "RackCabinetId", "FormFactor", "BladeSystemId"); err == nil {
		for _, h := range hws {
			if h.FormFactor == 2 {
				if h.BladeSystemId > 0 {
					continue // блейды в корзинах считаются через корзину
				}
				// блейд без корзины - считается блейдом, а не rack-сервером
				if r := byId[h.RackCabinetId]; r != nil {
					r.Blades++
				}
				continue
			}
			if r := byId[h.RackCabinetId]; r != nil {
				r.Servers++
			}
		}
	}
	// корзины и блейды в них
	var chassis []models.BladeSystems
	chassisRack := map[int64]int64{}
	if _, err := o.QueryTable(new(models.BladeSystems)).Filter("RackCabinetId__in", ids).Exclude("Active", 2).
		All(&chassis, "Id", "RackCabinetId"); err == nil {
		chIds := make([]int64, 0, len(chassis))
		for _, c := range chassis {
			chassisRack[c.Id] = c.RackCabinetId
			chIds = append(chIds, c.Id)
			if r := byId[c.RackCabinetId]; r != nil {
				r.Chassis++
			}
		}
		if len(chIds) > 0 {
			var blades []models.ServerHardware
			if _, err := o.QueryTable(new(models.ServerHardware)).Filter("BladeSystemId__in", chIds).Exclude("Active", 2).
				All(&blades, "Id", "BladeSystemId"); err == nil {
				for _, b := range blades {
					if r := byId[chassisRack[b.BladeSystemId]]; r != nil {
						r.Blades++
					}
				}
			}
		}
	}
	// СХД: число различных систем с размещением в шкафу
	var pl []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("RackCabinetId__in", ids).
		All(&pl, "RackCabinetId", "StorageSystemsId"); err == nil {
		seen := map[[2]int64]bool{}
		for _, p := range pl {
			k := [2]int64{p.RackCabinetId, p.StorageSystemsId}
			if seen[k] {
				continue
			}
			seen[k] = true
			if r := byId[p.RackCabinetId]; r != nil {
				r.Storages++
			}
		}
	}

	// мощность - тот же расчёт, что в списках и просмотре шкафа
	for rid, p := range rackpower.RackCabinetPowerTotalsBatch(o, ids) {
		if r := byId[rid]; r != nil {
			r.PowerAvgW, r.PowerPeakW, r.PowerRated = p.Avg24hW, p.Peak24hW, p.RatedMaxW
		}
	}
	sp := rackpower.DatacenterSitePowerTotalsBatch(o, []int64{id})[id]
	data.PowerAvgW, data.PowerPeakW = sp.Avg24hW, sp.Peak24hW

	sort.SliceStable(data.Racks, func(i, j int) bool { return naturalLess(data.Racks[i].Name, data.Racks[j].Name) })

	b, jerr := json.Marshal(data)
	if jerr != nil {
		flash.Error("Ошибка подготовки данных: %v", jerr)
		return
	}
	this.Data["DCJSON"] = template.JS(b)
}

// naturalLess - сравнение имён с учётом чисел ("R2" < "R10").
func naturalLess(a, b string) bool {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		ca, cb := a[i], b[j]
		if ca >= '0' && ca <= '9' && cb >= '0' && cb <= '9' {
			si := i
			for i < len(a) && a[i] >= '0' && a[i] <= '9' {
				i++
			}
			sj := j
			for j < len(b) && b[j] >= '0' && b[j] <= '9' {
				j++
			}
			na, _ := strconv.Atoi(a[si:i])
			nb, _ := strconv.Atoi(b[sj:j])
			if na != nb {
				return na < nb
			}
			continue
		}
		if ca != cb {
			return ca < cb
		}
		i++
		j++
	}
	return len(a)-i < len(b)-j
}
