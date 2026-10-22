package controllers

import (
	"encoding/json"
	"html/template"
	"loadmon/models"
	"loadmon/rackpower"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// siteMapSite - площадка/ЦОД (одна "комната" на этаже здания) на карте.
// На карте показываются только площадки с признаком ЦОД.
type siteMapSite struct {
	Id          int64         `json:"id"`
	Name        string        `json:"name"`
	Room        string        `json:"room"` // текст выноски: комната из описания ("комната: ..."), иначе наименование
	Floor       int           `json:"floor"`
	Racks       []*dcViewRack `json:"racks"`
	Servers     int           `json:"servers"`
	Blades      int           `json:"blades"`
	Chassis     int           `json:"chassis"`
	Storages    int           `json:"storages"`
	Switches    int           `json:"switches"`
	PowerAvgW   float64       `json:"powerAvgW"`
	PowerPeakW  float64       `json:"powerPeakW"`
	PowerRatedW float64       `json:"powerRatedW"`
}

type siteMapFloor struct {
	Floor int           `json:"floor"`
	Sites []siteMapSite `json:"sites"`
}

type siteMapBuilding struct {
	Address string         `json:"address"`
	Floors  []siteMapFloor `json:"floors"`
}

type siteMapCity struct {
	Name      string            `json:"name"`
	Buildings []siteMapBuilding `json:"buildings"`
}

var (
	siteMapRoomRe  = regexp.MustCompile(`(?i)комната:\s*(.*)$`)
	siteMapFloorRe = regexp.MustCompile(`-?\d+`)
	siteMapCityRe  = regexp.MustCompile(`(?i)^(г\.?|город)\s+`)
)

// siteMapFloorNum - номер этажа из поля Floor: пусто/не распознано = 1,
// "подвал"/"цоколь" = 0, иначе первое целое число в строке.
func siteMapFloorNum(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 1
	}
	if strings.Contains(s, "подвал") || strings.Contains(s, "цоколь") || strings.Contains(s, "basement") {
		return 0
	}
	if m := siteMapFloorRe.FindString(s); m != "" {
		if n, err := strconv.Atoi(m); err == nil {
			return n
		}
	}
	return 1
}

// siteMapRoom - код комнаты из описания ("... комната: <код>").
func siteMapRoom(annotation string) string {
	if m := siteMapRoomRe.FindStringSubmatch(annotation); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// siteMapAddress - адрес площадки: поле Address, иначе наименование без
// хвостового кода комнаты (как в sql/datacenter_site_address_floor.sql).
func siteMapAddress(s models.DatacenterSite, room string) string {
	if a := strings.TrimSpace(s.Address); a != "" {
		return a
	}
	nm := strings.TrimSpace(s.Name)
	if room != "" && strings.HasSuffix(nm, " "+room) {
		nm = strings.TrimSpace(strings.TrimSuffix(nm, room))
	}
	if nm == "" {
		return "Без адреса"
	}
	return nm
}

// siteMapCityOf - город = часть адреса до первой запятой.
func siteMapCityOf(addr string) string {
	i := strings.Index(addr, ",")
	if i <= 0 {
		return "Без города"
	}
	c := strings.TrimSpace(siteMapCityRe.ReplaceAllString(strings.TrimSpace(addr[:i]), ""))
	if c == "" {
		return "Без города"
	}
	return c
}

// SprDatacenterSiteMap - визуализация всех площадок/ЦОД: города -> здания
// (адреса) -> этажи -> комнаты (площадки). Данные считаются одним проходом и
// отдаются в страницу как JSON; рисование - на canvas в шаблоне.
//
// Маршрут: /sprsrv/datacentersitemap
func (this *MainController) SprDatacenterSiteMap() {
	RegisterRequest(this, LogTypeViewLoadTable)
	this.activeContent("sprsrv/datacentersitemap")
	this.Data["MapJSON"] = template.JS("null")

	flash := beego.NewFlash()
	defer func() {
		if len(flash.Data) > 0 {
			flash.Store(&this.Controller)
		}
	}()

	o := orm.NewOrm()
	var sites []models.DatacenterSite
	if _, err := o.QueryTable(new(models.DatacenterSite)).OrderBy("Name").Limit(-1).All(&sites); err != nil {
		flash.Error("Площадки не загружены: %v", err)
		return
	}

	siteIds := make([]int64, 0, len(sites))
	bySite := make(map[int64]*siteMapSite, len(sites))
	items := make([]*siteMapSite, 0, len(sites))
	addrOf := make(map[int64]string, len(sites))
	cityOf := make(map[int64]string, len(sites)) // город по LocalityCode (справочник населённых пунктов)
	cityNames := localityCityNames(o, sites)
	for _, s := range sites {
		if !s.IsDatacenter {
			continue
		}
		room := siteMapRoom(s.Annotation)
		addr := siteMapAddress(s, room)
		if city, ok := cityNames[s.LocalityCode]; ok {
			// новая схема: город - из справочника, адрес - улица и дом
			cityOf[s.Id] = city
			addr = strings.TrimSpace(s.Address)
			if addr == "" {
				addr = "Без адреса"
			}
		}
		if r := strings.TrimSpace(s.Room); r != "" {
			room = r
		}
		if room == "" {
			room = strings.TrimSpace(s.Name)
		}
		it := &siteMapSite{Id: s.Id, Name: s.Name, Room: room, Floor: siteMapFloorNum(s.Floor), Racks: []*dcViewRack{}}
		items = append(items, it)
		bySite[s.Id] = it
		addrOf[s.Id] = addr
		siteIds = append(siteIds, s.Id)
	}

	// шкафы
	rackSite := map[int64]int64{}
	rackBy := map[int64]*dcViewRack{}
	rackIds := []int64{}
	var racks []models.RackCabinet
	if _, err := o.QueryTable(new(models.RackCabinet)).OrderBy("Name").Limit(-1).All(&racks); err == nil {
		for _, r := range racks {
			s := bySite[r.DatacenterSiteId]
			if s == nil {
				continue
			}
			u := int(r.UHeight)
			if u <= 0 {
				u = 42
			}
			rk := &dcViewRack{Id: r.Id, Name: r.Name, UHeight: u}
			s.Racks = append(s.Racks, rk)
			rackSite[r.Id] = r.DatacenterSiteId
			rackBy[r.Id] = rk
			rackIds = append(rackIds, r.Id)
		}
		for _, s := range items {
			rs := s.Racks
			sort.SliceStable(rs, func(i, j int) bool { return naturalLess(rs[i].Name, rs[j].Name) })
		}
	}

	// rack-серверы
	var hws []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).Filter("RackCabinetId__gt", 0).Exclude("Active", 2).
		Limit(-1).All(&hws, "Id", "RackCabinetId", "FormFactor", "BladeSystemId"); err == nil {
		for _, h := range hws {
			if h.FormFactor == 2 {
				if h.BladeSystemId > 0 {
					continue // блейды в корзинах считаются через корзину
				}
				// блейд без корзины - считается блейдом, а не rack-сервером
				if r := rackBy[h.RackCabinetId]; r != nil {
					r.Blades++
				}
				continue
			}
			if r := rackBy[h.RackCabinetId]; r != nil {
				r.Servers++
			}
		}
	}
	// корзины и блейды
	chassisRack := map[int64]int64{}
	var chassis []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).Filter("RackCabinetId__gt", 0).Exclude("Active", 2).
		Limit(-1).All(&chassis, "Id", "RackCabinetId"); err == nil {
		for _, c := range chassis {
			chassisRack[c.Id] = c.RackCabinetId
			if r := rackBy[c.RackCabinetId]; r != nil {
				r.Chassis++
			}
		}
		var blades []models.ServerHardware
		if _, err := o.QueryTable(new(models.ServerHardware)).Filter("BladeSystemId__gt", 0).Exclude("Active", 2).
			Limit(-1).All(&blades, "Id", "BladeSystemId"); err == nil {
			for _, b := range blades {
				if r := rackBy[chassisRack[b.BladeSystemId]]; r != nil {
					r.Blades++
				}
			}
		}
	}
	// СХД: на шкаф - различные системы; на площадку - различные системы площадки
	var pl []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("RackCabinetId__gt", 0).
		Limit(-1).All(&pl, "RackCabinetId", "StorageSystemsId"); err == nil {
		// как в просмотре шкафа: СХД без записи или демонтированные (Status=6) не считаются
		stIds := make([]int64, 0, len(pl))
		for _, p := range pl {
			stIds = append(stIds, p.StorageSystemsId)
		}
		stOK := map[int64]bool{}
		var stRows []models.StorageSystems
		if len(stIds) > 0 {
			if _, e := o.QueryTable(new(models.StorageSystems)).Filter("Id__in", stIds).Limit(-1).All(&stRows, "Id", "Status"); e == nil {
				for _, st := range stRows {
					stOK[st.Id] = st.Status != 6
				}
			}
		}
		seenR := map[[2]int64]bool{}
		seenS := map[[2]int64]bool{}
		for _, p := range pl {
			r := rackBy[p.RackCabinetId]
			if r == nil || !stOK[p.StorageSystemsId] {
				continue
			}
			if k := [2]int64{p.RackCabinetId, p.StorageSystemsId}; !seenR[k] {
				seenR[k] = true
				r.Storages++
			}
			sid := rackSite[p.RackCabinetId]
			if k := [2]int64{sid, p.StorageSystemsId}; !seenS[k] {
				seenS[k] = true
				bySite[sid].Storages++
			}
		}
	}
	// мощность по шкафам
	for rid, p := range rackpower.RackCabinetPowerTotalsBatch(o, rackIds) {
		if r := rackBy[rid]; r != nil {
			r.PowerAvgW, r.PowerPeakW, r.PowerRated = p.Avg24hW, p.Peak24hW, p.RatedMaxW
		}
	}
	// итоги площадки
	for _, s := range items {
		for _, r := range s.Racks {
			s.Servers += r.Servers
			s.Blades += r.Blades
			s.Chassis += r.Chassis
		}
	}
	// коммутаторы
	var sws []models.SwitchSprRecords
	if _, err := o.QueryTable(new(models.SwitchSprRecords)).Limit(-1).All(&sws, "Id", "DatacenterSiteId", "RackCabinetId"); err == nil {
		for _, w := range sws {
			sid := w.DatacenterSiteId
			if sid <= 0 {
				sid = rackSite[w.RackCabinetId]
			}
			if s := bySite[sid]; s != nil {
				s.Switches++
			}
		}
	}
	// мощность площадки
	for sid, p := range rackpower.DatacenterSitePowerTotalsBatch(o, siteIds) {
		if s := bySite[sid]; s != nil {
			s.PowerAvgW, s.PowerPeakW, s.PowerRatedW = p.Avg24hW, p.Peak24hW, p.RatedMaxW
		}
	}

	// группировка город -> адрес -> этаж
	cityB := map[string]map[string]map[int][]siteMapSite{}
	for _, it := range items {
		addr := addrOf[it.Id]
		city, ok := cityOf[it.Id]
		if !ok {
			city = siteMapCityOf(addr)
		}
		if cityB[city] == nil {
			cityB[city] = map[string]map[int][]siteMapSite{}
		}
		if cityB[city][addr] == nil {
			cityB[city][addr] = map[int][]siteMapSite{}
		}
		cityB[city][addr][it.Floor] = append(cityB[city][addr][it.Floor], *it)
	}

	cities := make([]siteMapCity, 0, len(cityB))
	for cn, addrs := range cityB {
		c := siteMapCity{Name: cn}
		for an, floors := range addrs {
			b := siteMapBuilding{Address: an}
			for fn, ss := range floors {
				sort.SliceStable(ss, func(i, j int) bool { return naturalLess(ss[i].Room, ss[j].Room) })
				b.Floors = append(b.Floors, siteMapFloor{Floor: fn, Sites: ss})
			}
			sort.Slice(b.Floors, func(i, j int) bool { return b.Floors[i].Floor < b.Floors[j].Floor })
			c.Buildings = append(c.Buildings, b)
		}
		sort.Slice(c.Buildings, func(i, j int) bool { return c.Buildings[i].Address < c.Buildings[j].Address })
		cities = append(cities, c)
	}
	sort.Slice(cities, func(i, j int) bool {
		// "Без города" - в конец
		ai, aj := cities[i].Name == "Без города", cities[j].Name == "Без города"
		if ai != aj {
			return aj
		}
		return cities[i].Name < cities[j].Name
	})

	b, err := json.Marshal(cities)
	if err != nil {
		flash.Error("Ошибка подготовки данных: %v", err)
		return
	}
	this.Data["MapJSON"] = template.JS(b)
	this.Data["SiteCount"] = len(items)
}

// localityCityNames возвращает названия населённых пунктов (без типа) по кодам
// ОКТМО из DatacenterSite.LocalityCode.
func localityCityNames(o orm.Ormer, sites []models.DatacenterSite) map[string]string {
	res := map[string]string{}
	seen := map[string]bool{}
	codes := make([]string, 0)
	for _, s := range sites {
		if s.LocalityCode != "" && !seen[s.LocalityCode] {
			seen[s.LocalityCode] = true
			codes = append(codes, s.LocalityCode)
		}
	}
	if len(codes) == 0 {
		return res
	}
	var locs []models.Locality
	if _, err := o.QueryTable("cmdb_locality").Filter("Oktmo__in", codes).All(&locs); err != nil {
		return res
	}
	for _, l := range locs {
		res[l.Oktmo] = l.Name
	}
	return res
}
