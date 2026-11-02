package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/models"
	"loadmon/rackpower"
	"sort"
	"strconv"
	"strings"

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
	Tapes      int     `json:"tapes"`    // ленточные библиотеки
	Network    int     `json:"network"`  // сетевое оборудование (коммутаторы, маршрутизаторы, ...)
	PowerAvgW  float64 `json:"powerAvgW"`
	PowerPeakW float64 `json:"powerPeakW"`
	PowerRated float64 `json:"powerRatedW"`
	// Shelf - служебный шкаф "(размещение вне шкафа)": рисуется как стеллаж справа от
	// группы шкафов; ShelfUnits - число секций (примерная оценка по количеству оборудования).
	Shelf      bool          `json:"shelf,omitempty"`
	ShelfUnits int           `json:"shelfUnits,omitempty"`
	Items      []dcShelfItem `json:"items,omitempty"` // оборудование на стеллаже (для 3D-отрисовки)
}

// dcShelfBlade - занятый слот блейд-корзины на стеллаже (для отрисовки лицевой панели).
type dcShelfBlade struct {
	Slot  int `json:"s"` // номер слота (BladeNum, обычно 1..)
	Power int `json:"p"` // 0/1/2
}

// dcShelfItem - единица оборудования на стеллаже: тип (server/chassis/storage/switch),
// "вендор модель" (для выбора оформления), высота в юнитах (0 - не указана), питание (0/1/2).
// Для корзин дополнительно: Id/Name/Slots/Blades - иначе на полке рисовался пустой контур без идентификации.
type dcShelfItem struct {
	Kind   string         `json:"k"`
	Id     int64          `json:"id,omitempty"`
	Name   string         `json:"nm,omitempty"`
	Vendor string         `json:"vd"`
	U      int            `json:"u"`
	Power  int            `json:"p"`
	Slots  int            `json:"slots,omitempty"` // число слотов корзины
	Blades []dcShelfBlade `json:"blades"`          // всегда массив (даже []); иначе фронт рисовал пустой контур
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

	this.Data["ShowEquipFilters"] = false
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
	fillDatacenterSiteViewPage(this, orm.NewOrm(), id, flash)
}

// SprDatacenterEquipView - визуализация оборудования площадки/ЦОД с фильтром
// «Населённый пункт» + «Площадка/ЦОД» (как в справочнике площадок/шкафов).
// Та же сцена, что /sprsrv/datacentersiteview/:id.
//
// Маршрут: /sprsrv/datacenterequipview/:id  (0 = площадка ещё не выбрана)
func (this *MainController) SprDatacenterEquipView() {
	idStr := this.Ctx.Input.Param(":id")

	RegisterRequest(this, LogTypeViewLoadTable)
	this.activeContent("sprsrv/datacentersiteview")

	this.Data["ShowEquipFilters"] = true
	this.Data["Id"] = idStr
	this.Data["SiteName"] = ""
	this.Data["RackCount"] = 0
	this.Data["DCJSON"] = template.JS("null")
	this.Data["FilterLocalityName"] = ""
	this.Data["FilterLocalityCode"] = ""

	flash := beego.NewFlash()
	defer func() {
		if len(flash.Data) > 0 {
			flash.Store(&this.Controller)
		}
	}()

	o := orm.NewOrm()
	var sites []models.DatacenterSite
	_, _ = o.QueryTable(new(models.DatacenterSite)).OrderBy("Name").Limit(-1).All(&sites, "Id", "Name", "LocalityCode")
	this.Data["DatacenterSites"] = sites

	localityCode := strings.TrimSpace(this.GetString("locality"))
	if localityCode == "" {
		localityCode = strings.TrimSpace(this.GetString("LocalityCode"))
	}
	this.Data["FilterLocalityCode"] = localityCode
	if localityCode != "" {
		this.Data["FilterLocalityName"] = localityNamesByCode(o, []models.DatacenterSite{{LocalityCode: localityCode}})[localityCode]
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id < 0 {
		flash.Error("Некорректный идентификатор площадки: %s", idStr)
		return
	}
	if id == 0 {
		return
	}
	// если площадка выбрана, а населённый пункт в query пуст - подставить из записи
	var site models.DatacenterSite
	if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", id).One(&site, "Id", "Name", "LocalityCode"); err == nil {
		if localityCode == "" && site.LocalityCode != "" {
			this.Data["FilterLocalityCode"] = site.LocalityCode
			this.Data["FilterLocalityName"] = localityNamesByCode(o, []models.DatacenterSite{site})[site.LocalityCode]
		}
	}
	fillDatacenterSiteViewPage(this, o, id, flash)
}

// fillDatacenterSiteViewPage заполняет SiteName / RackCount / DCJSON для
// просмотра оборудования выбранной площадки (общая логика siteview и equipview).
func fillDatacenterSiteViewPage(this *MainController, o orm.Ormer, id int64, flash *beego.FlashData) {
	var site models.DatacenterSite
	if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", id).One(&site); err != nil {
		flash.Error("Площадка Id=%d не найдена: %v", id, err)
		return
	}
	this.Data["SiteName"] = site.Name
	this.Data["Id"] = strconv.FormatInt(id, 10)

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

	// rack-серверы (списанные и «в пути» не показываем; выведенные и прочие статусы - да)
	var hws []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).Filter("RackCabinetId__in", ids).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
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
	// корзины и блейды в них (списанные и «в пути» корзины не считаем)
	var chassis []models.BladeSystems
	chassisRack := map[int64]int64{}
	if _, err := o.QueryTable(new(models.BladeSystems)).Filter("RackCabinetId__in", ids).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
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
			if _, err := o.QueryTable(new(models.ServerHardware)).Filter("BladeSystemId__in", chIds).
				Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
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
		// как в просмотре шкафа: СХД без записи или демонтированные (Status=6) не считаются
		stIds := make([]int64, 0, len(pl))
		for _, p := range pl {
			stIds = append(stIds, p.StorageSystemsId)
		}
		stOK := map[int64]bool{}
		var stRows []models.StorageSystems
		if len(stIds) > 0 {
			if _, e := o.QueryTable(new(models.StorageSystems)).Filter("Id__in", stIds).Limit(-1).All(&stRows, "Id", "Status", "ItiliumStatus"); e == nil {
				for _, st := range stRows {
					stOK[st.Id] = st.Status != 6 && !models.ItiliumStatusHiddenFromPlans(st.ItiliumStatus)
				}
			}
		}
		seen := map[[2]int64]bool{}
		for _, p := range pl {
			if !stOK[p.StorageSystemsId] {
				continue
			}
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

	// ленточные библиотеки и сетевое оборудование (списанные и «в пути» не считаем)
	var tapes []models.TapeLibrary
	if _, err := o.QueryTable(new(models.TapeLibrary)).Filter("RackCabinetId__in", ids).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
		Limit(-1).All(&tapes, "Id", "RackCabinetId"); err == nil {
		for _, t := range tapes {
			if r := byId[t.RackCabinetId]; r != nil {
				r.Tapes++
			}
		}
	}
	var nets []models.SwitchSprRecords
	if _, err := o.QueryTable(new(models.SwitchSprRecords)).Filter("RackCabinetId__in", ids).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
		Limit(-1).All(&nets, "Id", "RackCabinetId"); err == nil {
		for _, n := range nets {
			if r := byId[n.RackCabinetId]; r != nil {
				r.Network++
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

	// стеллаж вне шкафов: секция вмещает ~5 полок по ~7U (сервер 2U, корзина до 7U, СХД 2U)
	for i := range data.Racks {
		r := &data.Racks[i]
		if r.Name != models.RackCabinetOutsideName {
			continue
		}
		r.Shelf = true
		r.Items = shelfItems(o, r.Id)
		u := r.Servers*2 + r.Chassis*7 + r.Storages*2 + r.Tapes*6 + r.Network
		r.ShelfUnits = (u + 34) / 35 // 5 полок * ~7U
		if r.ShelfUnits < 1 {
			r.ShelfUnits = 1
		}
	}

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

// shelfItems собирает оборудование служебного шкафа "вне шкафа" (стеллаж) для 3D-отрисовки.
func shelfItems(o orm.Ormer, rackId int64) []dcShelfItem {
	items := []dcShelfItem{}
	var hws []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).Filter("RackCabinetId", rackId).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
		Limit(-1).All(&hws, "Id", "Vendor", "Model", "ShortModel", "SizeUnit", "PowerState", "FormFactor", "BladeSystemId"); err == nil {
		for _, h := range hws {
			if h.FormFactor == 2 && h.BladeSystemId > 0 {
				continue
			}
			vd := strings.TrimSpace(h.Vendor + " " + firstNonEmpty(h.Model, h.ShortModel))
			items = append(items, dcShelfItem{
				Kind: "server", Id: h.Id, Name: firstNonEmpty(vd, fmt.Sprintf("server-%d", h.Id)),
				Vendor: vd, U: h.SizeUnit, Power: h.PowerState,
			})
		}
	}
	var chassis []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).Filter("RackCabinetId", rackId).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
		Limit(-1).All(&chassis, "Id", "Name", "Vendor", "Model", "Type", "Size", "SizeUnit", "SerialNum", "PowerState"); err == nil {
		chIds := make([]int64, 0, len(chassis))
		for _, c := range chassis {
			chIds = append(chIds, c.Id)
		}
		bladesByCh := map[int64][]dcShelfBlade{}
		if len(chIds) > 0 {
			var blades []models.ServerHardware
			if _, err := o.QueryTable(new(models.ServerHardware)).Filter("BladeSystemId__in", chIds).
				Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
				Limit(-1).All(&blades, "Id", "BladeSystemId", "BladeNum", "PowerState"); err == nil {
				for _, b := range blades {
					bladesByCh[b.BladeSystemId] = append(bladesByCh[b.BladeSystemId], dcShelfBlade{
						Slot: b.BladeNum, Power: rackViewPower(b.PowerState),
					})
				}
			}
		}
		for _, c := range chassis {
			vd := strings.TrimSpace(c.Vendor + " " + firstNonEmpty(c.Model, c.Type))
			name := firstNonEmpty(c.Name, vd, c.SerialNum, fmt.Sprintf("chassis-%d", c.Id))
			if vd == "" {
				vd = name
			}
			slots := c.Size
			if slots <= 0 {
				slots = 16
			}
			sizeU := c.SizeUnit
			if sizeU <= 0 {
				sizeU = 10
			}
			bls := bladesByCh[c.Id]
			if bls == nil {
				bls = []dcShelfBlade{}
			}
			items = append(items, dcShelfItem{
				Kind: "chassis", Id: c.Id, Name: name, Vendor: vd,
				U: sizeU, Power: c.PowerState, Slots: slots, Blades: bls,
			})
		}
	}
	var pl []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("RackCabinetId", rackId).
		Limit(-1).All(&pl, "StorageSystemsId", "SizeUnit"); err == nil && len(pl) > 0 {
		ids := make([]int64, 0, len(pl))
		for _, p := range pl {
			ids = append(ids, p.StorageSystemsId)
		}
		byId := map[int64]models.StorageSystems{}
		var sts []models.StorageSystems
		if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Id__in", ids).Limit(-1).
			All(&sts, "Id", "Vendor", "Model", "ShortName", "Status", "ItiliumStatus", "Power"); err == nil {
			for _, st := range sts {
				byId[st.Id] = st
			}
		}
		for _, p := range pl {
			st, ok := byId[p.StorageSystemsId]
			if !ok || st.Status == 6 || models.ItiliumStatusHiddenFromPlans(st.ItiliumStatus) {
				continue
			}
			pw := 2
			if st.Power == 1 {
				pw = 1
			}
			vd := strings.TrimSpace(st.Vendor + " " + st.Model)
			items = append(items, dcShelfItem{
				Kind: "storage", Id: st.Id, Name: firstNonEmpty(st.ShortName, vd, fmt.Sprintf("storage-%d", st.Id)),
				Vendor: vd, U: p.SizeUnit, Power: pw,
			})
		}
	}
	var sws []models.SwitchSprRecords
	if _, err := o.QueryTable(new(models.SwitchSprRecords)).Filter("RackCabinetId", rackId).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
		Limit(-1).All(&sws, "Id", "Vendor", "Model", "Name", "SizeUnit", "ItiliumStatus"); err == nil {
		for _, w := range sws {
			u := w.SizeUnit
			if u <= 0 {
				u = 1
			}
			vd := strings.TrimSpace(w.Vendor + " " + w.Model + " " + w.Name)
			items = append(items, dcShelfItem{
				Kind: "switch", Id: w.Id, Name: firstNonEmpty(w.Name, vd, fmt.Sprintf("switch-%d", w.Id)),
				Vendor: vd, U: u, Power: itiliumPower(w.ItiliumStatus),
			})
		}
	}
	var tps []models.TapeLibrary
	if _, err := o.QueryTable(new(models.TapeLibrary)).Filter("RackCabinetId", rackId).
		Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
		Limit(-1).All(&tps, "Id", "Name", "Model", "SizeUnit", "ItiliumStatus"); err == nil {
		for _, t := range tps {
			vd := "HPE " + t.Model
			items = append(items, dcShelfItem{
				Kind: "tape", Id: t.Id, Name: firstNonEmpty(t.Name, vd, fmt.Sprintf("tape-%d", t.Id)),
				Vendor: vd, U: t.SizeUnit, Power: itiliumPower(t.ItiliumStatus),
			})
		}
	}
	return items
}
