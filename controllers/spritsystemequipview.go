package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/config"
	"loadmon/itsystem"
	"loadmon/models"
	"loadmon/rackpower"
	"sort"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// itSysEquipListItem - строка списка оборудования справа от сцены.
// Key - собственный ключ строки; EqKey - ключ юнита/корзины на сцене
// (для блейда - ключ корзины, пусто у ВМ и оборудования без размещения).
type itSysEquipListItem struct {
	Key      string         `json:"key"`
	EqKey    string         `json:"eqKey,omitempty"`
	Kind     string         `json:"kind"` // server|vm|storage
	Id       int64          `json:"id"`
	Name     string         `json:"name"`
	Subtitle string         `json:"subtitle"`
	SiteId   int64          `json:"siteId"`
	SiteName string         `json:"siteName"`
	RackId   int64          `json:"rackId"`
	RackName string         `json:"rackName"`
	Rows     []rackViewKV   `json:"rows"`
	Links    []rackViewLink `json:"links"`
}

// itSysSiteInfo - площадка в переключателе площадок режима.
type itSysSiteInfo struct {
	Id      int64  `json:"id"`
	Name    string `json:"name"`
	Racks   int    `json:"racks"`
	Equip   int    `json:"equip"`
	Current bool   `json:"current"`
}

// itSysScene - данные режима «Оборудование ИТ-системы» внутри dcViewData.
type itSysScene struct {
	SystemKey   string               `json:"systemKey"`
	SystemName  string               `json:"systemName"`
	CurrentSite int64                `json:"currentSite"`
	Sites       []itSysSiteInfo      `json:"sites"`
	Equipment   []itSysEquipListItem `json:"equipment"`
	Stats       itSysStats           `json:"stats"`
}

type itSysStats struct {
	Servers  int `json:"servers"`
	VMs      int `json:"vms"`
	Storages int `json:"storages"`
	Racks    int `json:"racks"`
	Sites    int `json:"sites"`
}

// itSysBuild - результат разбора ИТ-системы: сцена на каждую площадку + общий список.
type itSysBuild struct {
	scene  itSysScene
	order  []int64                // площадки в порядке показа
	scenes map[int64]*dcViewData // siteId -> шкафы с оборудованием
}

// SprITSystemEquipView - «Оборудование ИТ-системы»: та же изометрическая 3D-сцена, что
// в «Оборудование площадки/ЦОД» (datacentersiteview.tpl), но показываются только шкафы
// с оборудованием выбранной ИТ-системы; юниты системы подсвечены, по наведению -
// расширенная карточка со ссылками на формы редактирования, справа - список оборудования.
// СХД определяются: для ВМ - по LUN, использованным ВМ; для аппаратных серверов - по
// связке LUN.Server с именем сервера.
//
// Маршрут: /sprsrv/itsystemequipview?ITSystemKey=...&site=<id площадки>
func (this *MainController) SprITSystemEquipView() {
	RegisterRequest(this, LogTypeViewLoadTable)
	this.activeContent("sprsrv/datacentersiteview")

	key := strings.TrimSpace(this.GetString("ITSystemKey"))
	this.Data["ITMode"] = true
	this.Data["ShowEquipFilters"] = false
	this.Data["ITSystemKey"] = key
	this.Data["ITSystemSpr"] = itsystem.ImportITSystemSpr()
	this.Data["HasSystem"] = key != ""
	this.Data["SystemName"] = ""
	this.Data["ITSites"] = []itSysSiteInfo{}
	this.Data["Id"] = "0"
	this.Data["SiteName"] = ""
	this.Data["RackCount"] = 0
	this.Data["DCJSON"] = template.JS("null")
	this.Data["PowerLight"] = config.ConfigData.PowerLightMode()

	flash := beego.NewFlash()
	defer func() {
		if len(flash.Data) > 0 {
			flash.Store(&this.Controller)
		}
	}()

	if key == "" {
		return
	}
	sysName := key
	if rec, ok := itsystem.GetITSystemRec(key); ok {
		sysName = firstNonEmpty(rec.Label, rec.FullName, rec.ObjectKey, key)
	}
	this.Data["SystemName"] = sysName

	b, err := buildITSystemEquipView(orm.NewOrm(), key, sysName)
	if err != nil {
		flash.Error("Ошибка построения визуализации: %v", err)
		return
	}
	if len(b.scene.Equipment) == 0 {
		flash.Notice("Для ИТ-системы %s не найдено серверов/СХД в CMDB", key)
	}

	cur := int64(0)
	if q, e := strconv.ParseInt(strings.TrimSpace(this.GetString("site")), 10, 64); e == nil && q > 0 {
		if _, ok := b.scenes[q]; ok {
			cur = q
		}
	}
	if cur == 0 && len(b.order) > 0 {
		cur = b.order[0]
	}

	scene := b.scene
	scene.CurrentSite = cur
	for i := range scene.Sites {
		scene.Sites[i].Current = scene.Sites[i].Id == cur
	}
	this.Data["ITSites"] = scene.Sites

	data := dcViewData{Racks: []dcViewRack{}}
	if s := b.scenes[cur]; s != nil {
		data = *s
		this.Data["SiteName"] = s.SiteName
		this.Data["RackCount"] = len(s.Racks)
		this.Data["Id"] = strconv.FormatInt(cur, 10)
	}
	data.IT = &scene

	js, jerr := json.Marshal(data)
	if jerr != nil {
		flash.Error("Ошибка подготовки данных: %v", jerr)
		return
	}
	this.Data["DCJSON"] = template.JS(js)
}

// rackAgg - накопитель оборудования ИТ-системы в одном шкафу.
type rackAgg struct {
	rc    models.RackCabinet
	eq    []rackViewItem
	shelf []dcShelfItem
}

func buildITSystemEquipView(o orm.Ormer, key, sysName string) (*itSysBuild, error) {
	out := &itSysBuild{
		scene:  itSysScene{SystemKey: key, SystemName: sysName, Sites: []itSysSiteInfo{}, Equipment: []itSysEquipListItem{}},
		scenes: map[int64]*dcViewData{},
	}

	var sprs []models.ServerSprRecords
	cond := orm.NewCondition().And("ITSystemKey", key).AndNot("Active", 2)
	if _, err := models.ServerSprQS(o).SetCond(models.ExcludeDestroyedCond(cond)).Limit(-1).
		All(&sprs, "Id", "Name", "NameFQDN", "Type", "VMSprId", "HWSprId", "IPAddress", "OS", "Role", "Active"); err != nil {
		return out, err
	}

	var (
		hwIDs     []int64
		vmSprIDs  []int64
		hwNames   = map[string]bool{}
		sprByHW   = map[int64]models.ServerSprRecords{}
		listEquip = make([]itSysEquipListItem, 0, len(sprs))
		listByKey = map[string]int{}
	)
	addList := func(it itSysEquipListItem) {
		listByKey[it.Key] = len(listEquip)
		listEquip = append(listEquip, it)
	}

	for _, s := range sprs {
		switch s.Type {
		case 1: // виртуальная машина
			out.scene.Stats.VMs++
			if s.VMSprId > 0 {
				vmSprIDs = append(vmSprIDs, s.VMSprId)
			}
			addList(itSysEquipListItem{
				Key: fmt.Sprintf("vm:%d", s.Id), Kind: "vm", Id: s.Id,
				Name:     firstNonEmpty(s.Name, s.NameFQDN, fmt.Sprintf("vm-%d", s.Id)),
				Subtitle: "Виртуальная машина",
				Rows:     itSysSprRows(s),
				Links:    []rackViewLink{{Title: "Карточка сервера", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", s.Id)}},
			})
		default: // 0 - аппаратный сервер, 2 - гипервизор
			out.scene.Stats.Servers++
			if short := strings.ToLower(itSysShortHost(s.Name)); short != "" {
				hwNames[short] = true
			}
			if fq := strings.ToLower(itSysShortHost(s.NameFQDN)); fq != "" {
				hwNames[fq] = true
			}
			hwID := s.HWSprId
			if hwID <= 0 {
				var hw models.ServerHardware
				if err := o.QueryTable(new(models.ServerHardware)).Filter("ServerId", s.Id).One(&hw, "Id"); err == nil {
					hwID = hw.Id
				}
			}
			it := itSysEquipListItem{
				Key: fmt.Sprintf("server:%d", s.Id), Kind: "server", Id: s.Id,
				Name:     firstNonEmpty(s.Name, s.NameFQDN, fmt.Sprintf("srv-%d", s.Id)),
				Subtitle: typeLabelITSys(s.Type),
				Rows:     itSysSprRows(s),
				Links:    []rackViewLink{{Title: "Общие данные по серверу", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", s.Id)}},
			}
			if hwID > 0 {
				it.Links = append(it.Links, rackViewLink{Title: "Карточка оборудования", URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", hwID)})
				hwIDs = append(hwIDs, hwID)
				sprByHW[hwID] = s
			}
			addList(it)
		}
	}

	hwByID := map[int64]models.ServerHardware{}
	if len(hwIDs) > 0 {
		var hws []models.ServerHardware
		if _, err := o.QueryTable(new(models.ServerHardware)).Filter("Id__in", hwIDs).
			Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).
			Limit(-1).All(&hws); err == nil {
			for _, h := range hws {
				hwByID[h.Id] = h
			}
		}
	}

	// ---- СХД: LUN, использованные ВМ ИТ-системы ----
	stVia := map[int64]string{}
	if len(vmSprIDs) > 0 {
		var usages []models.VMUsageDatastore
		if _, err := o.QueryTable(new(models.VMUsageDatastore)).Filter("VMSprId__in", vmSprIDs).Limit(-1).
			All(&usages, "VMSprId", "UID", "StorageName", "Datastore"); err == nil {
			var uids, vnames []string
			uidSet, vnSet := map[string]bool{}, map[string]bool{}
			for _, u := range usages {
				if uid := strings.TrimSpace(u.UID); uid != "" && !uidSet[uid] {
					uidSet[uid] = true
					uids = append(uids, uid)
				}
				sn := firstNonEmpty(strings.TrimSpace(u.StorageName), strings.TrimSpace(u.Datastore))
				if sn != "" && !vnSet[sn] {
					vnSet[sn] = true
					vnames = append(vnames, sn)
				}
			}
			for _, v := range loadVolumesByUIDOrVirtualName(o, uids, vnames) {
				if v.StorageSystemsId > 0 {
					stVia[v.StorageSystemsId] = "LUN виртуальных машин"
				}
			}
		}
	}
	// ---- СХД: LUN <-> имя аппаратного сервера ИТ-системы ----
	for name := range hwNames {
		if len(name) < 2 {
			continue
		}
		var vols []models.StorageVolumes
		if _, err := o.QueryTable(new(models.StorageVolumes)).Filter("Server__icontains", name).Limit(-1).
			All(&vols, "Id", "StorageSystemsId", "Server"); err != nil {
			continue
		}
		for _, v := range vols {
			if v.StorageSystemsId <= 0 || !volumeMentionsServer(v.Server, hwNames) {
				continue
			}
			switch cur := stVia[v.StorageSystemsId]; {
			case cur == "":
				stVia[v.StorageSystemsId] = "LUN сервера по имени"
			case !strings.Contains(cur, "по имени"):
				stVia[v.StorageSystemsId] = cur + "; LUN сервера по имени"
			}
		}
	}

	stByID := map[int64]models.StorageSystems{}
	if len(stVia) > 0 {
		ids := make([]int64, 0, len(stVia))
		for id := range stVia {
			ids = append(ids, id)
		}
		var sts []models.StorageSystems
		if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Id__in", ids).Limit(-1).All(&sts); err == nil {
			for _, st := range sts {
				if st.Status == 6 || models.ItiliumStatusHiddenFromPlans(st.ItiliumStatus) {
					continue
				}
				stByID[st.Id] = st
				out.scene.Stats.Storages++
				addList(itSysEquipListItem{
					Key: fmt.Sprintf("storage:%d", st.Id), Kind: "storage", Id: st.Id,
					Name:     firstNonEmpty(st.ShortName, st.SerialNum, fmt.Sprintf("storage-%d", st.Id)),
					Subtitle: "СХД",
					Rows:     itSysStorageRows(st, stVia[st.Id]),
					Links:    itSysStorageLinks(st),
				})
			}
		}
	}

	// ---- размещение: накопители по шкафам ----
	aggs := map[int64]*rackAgg{}
	siteOfRack := map[int64]int64{}
	ensureRack := func(rackID int64) *rackAgg {
		if rackID <= 0 {
			return nil
		}
		if a, ok := aggs[rackID]; ok {
			return a
		}
		var rc models.RackCabinet
		if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", rackID).One(&rc); err != nil {
			return nil
		}
		a := &rackAgg{rc: rc}
		aggs[rackID] = a
		siteOfRack[rackID] = rc.DatacenterSiteId
		return a
	}
	// служебный шкаф «вне шкафа» площадки (для СХД без шкафа; создаётся виртуально, если его нет)
	outsideRack := func(siteID int64) *rackAgg {
		var rc models.RackCabinet
		if err := o.QueryTable(new(models.RackCabinet)).Filter("DatacenterSiteId", siteID).
			Filter("Name", models.RackCabinetOutsideName).One(&rc); err == nil {
			return ensureRack(rc.Id)
		}
		id := -siteID // виртуальный стеллаж: отрицательный Id, чтобы не пересекаться с реальными шкафами
		if a, ok := aggs[id]; ok {
			return a
		}
		a := &rackAgg{rc: models.RackCabinet{Id: id, Name: models.RackCabinetOutsideName, DatacenterSiteId: siteID, UHeight: 42}}
		aggs[id] = a
		siteOfRack[id] = siteID
		return a
	}
	isOutside := func(a *rackAgg) bool { return a.rc.Name == models.RackCabinetOutsideName }

	// аппаратные серверы: блейды - в корзину, остальные - в свой шкаф
	type bladeRef struct {
		hw  models.ServerHardware
		spr models.ServerSprRecords
	}
	chassisBlades := map[int64][]bladeRef{}
	for hwID, hw := range hwByID {
		spr := sprByHW[hwID]
		if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
			chassisBlades[hw.BladeSystemId] = append(chassisBlades[hw.BladeSystemId], bladeRef{hw: hw, spr: spr})
			continue
		}
		agg := ensureRack(hw.RackCabinetId)
		if agg == nil {
			continue
		}
		rows, links, name := itSysServerCard(hw, spr)
		key := itSysServerKey(hw, spr)
		power := rackViewPower(hw.PowerState)
		if isOutside(agg) {
			vd := strings.TrimSpace(hw.Vendor + " " + firstNonEmpty(hw.Model, hw.ShortModel))
			agg.shelf = append(agg.shelf, dcShelfItem{Kind: "server", Id: hw.Id, Name: name, Vendor: vd, U: hw.SizeUnit, Power: power,
				Blades: []dcShelfBlade{}, Key: key, Rows: rows, Links: links})
		} else {
			sizeU, guess := hw.SizeUnit, false
			if sizeU <= 0 && hw.StartUnitPos > 0 {
				sizeU, guess = rackGuessSizeU(hw.Model), true
			}
			agg.eq = append(agg.eq, rackViewItem{
				Key: key, Kind: "server", Id: hw.Id, Name: name, Vendor: hw.Vendor, Model: hw.Model,
				StartU: hw.StartUnitPos, SizeU: sizeU, Power: power, Rows: rows, Links: links, SizeGuess: guess,
				Blades: []rackViewBlade{},
			})
		}
		if i, ok := listByKey[fmt.Sprintf("server:%d", spr.Id)]; ok && spr.Id > 0 {
			listEquip[i].EqKey = key
			listEquip[i].RackId = agg.rc.Id
			listEquip[i].RackName = agg.rc.Name
			listEquip[i].SiteId = agg.rc.DatacenterSiteId
		}
	}

	if len(chassisBlades) > 0 {
		chIDs := make([]int64, 0, len(chassisBlades))
		for id := range chassisBlades {
			chIDs = append(chIDs, id)
		}
		var chs []models.BladeSystems
		if _, err := o.QueryTable(new(models.BladeSystems)).Filter("Id__in", chIDs).
			Exclude("ItiliumStatus__in", itiliumStatusWrittenOff, itiliumStatusInTransit).Limit(-1).All(&chs); err == nil {
			for _, ch := range chs {
				agg := ensureRack(ch.RackCabinetId)
				if agg == nil {
					continue
				}
				blades := chassisBlades[ch.Id]
				sort.Slice(blades, func(i, j int) bool { return blades[i].hw.BladeNum < blades[j].hw.BladeNum })
				ckey := fmt.Sprintf("chassis:%d", ch.Id)
				name := firstNonEmpty(ch.Name, ch.SerialNum, fmt.Sprintf("chassis-%d", ch.Id))
				slots := ch.Size
				if slots <= 0 {
					slots = 16
				}
				var rows []rackViewKV
				rows = rackViewKVAdd(rows, "Тип", ch.Type)
				rows = rackViewKVAdd(rows, "Серийный номер", ch.SerialNum)
				rows = rackViewKVAdd(rows, "Слотов", strconv.Itoa(slots))
				rows = rackViewKVAdd(rows, "Блейдов ИТ-системы", strconv.Itoa(len(blades)))
				links := []rackViewLink{
					{Title: "Редактировать корзину", URL: fmt.Sprintf("/sprsrv/bladesystemsedit-tab/%d", ch.Id)},
					{Title: "Серверы в корзине", URL: fmt.Sprintf("/sprsrv/bladesystemserverslist/%d", ch.Id)},
				}
				rbl := make([]rackViewBlade, 0, len(blades))
				sbl := make([]dcShelfBlade, 0, len(blades))
				for _, b := range blades {
					bn := firstNonEmpty(b.spr.Name, b.hw.SerialNum, fmt.Sprintf("blade-%d", b.hw.Id))
					rows = rackViewKVAdd(rows, fmt.Sprintf("Блейд, слот %d", b.hw.BladeNum), bn)
					if b.spr.Id > 0 {
						links = append(links, rackViewLink{Title: "Сервер " + bn, URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", b.spr.Id)})
					}
					links = append(links, rackViewLink{Title: "Оборудование " + bn, URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", b.hw.Id)})
					rbl = append(rbl, rackViewBlade{Id: b.hw.Id, Slot: b.hw.BladeNum, Name: bn, Model: b.hw.Model, Power: rackViewPower(b.hw.PowerState)})
					sbl = append(sbl, dcShelfBlade{Slot: b.hw.BladeNum, Power: rackViewPower(b.hw.PowerState)})
					if i, ok := listByKey[fmt.Sprintf("server:%d", b.spr.Id)]; ok && b.spr.Id > 0 {
						listEquip[i].EqKey = ckey
						listEquip[i].RackId = agg.rc.Id
						listEquip[i].RackName = agg.rc.Name
						listEquip[i].SiteId = agg.rc.DatacenterSiteId
						listEquip[i].Subtitle = fmt.Sprintf("Блейд, слот %d, корзина %s", b.hw.BladeNum, name)
					}
				}
				vd := strings.TrimSpace(ch.Vendor + " " + firstNonEmpty(ch.Model, ch.Type))
				if isOutside(agg) {
					agg.shelf = append(agg.shelf, dcShelfItem{
						Kind: "chassis", Id: ch.Id, Name: name, Vendor: firstNonEmpty(vd, name), U: ch.SizeUnit,
						Power: rackViewPower(ch.PowerState), Slots: slots, Blades: sbl, Key: ckey, Rows: rows, Links: links,
					})
					continue
				}
				sizeU := ch.SizeUnit
				if sizeU <= 0 && ch.EndUnitPos >= ch.StartUnitPos && ch.StartUnitPos > 0 {
					sizeU = ch.EndUnitPos - ch.StartUnitPos + 1
				}
				agg.eq = append(agg.eq, rackViewItem{
					Key: ckey, Kind: "chassis", Id: ch.Id, Name: name, Vendor: ch.Vendor, Model: firstNonEmpty(ch.Model, ch.Type),
					StartU: ch.StartUnitPos, SizeU: sizeU, Power: rackViewPower(ch.PowerState), Slots: slots,
					Rows: rows, Links: links, Blades: rbl,
				})
			}
		}
	}

	// СХД: все размещения (шкаф / только площадка)
	if len(stByID) > 0 {
		stList := make([]int64, 0, len(stByID))
		for id := range stByID {
			stList = append(stList, id)
		}
		var placements []models.StorageSystemPlacement
		if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId__in", stList).
			Limit(-1).All(&placements); err == nil {
			for _, p := range placements {
				st, ok := stByID[p.StorageSystemsId]
				if !ok {
					continue
				}
				var agg *rackAgg
				switch {
				case p.RackCabinetId > 0:
					agg = ensureRack(p.RackCabinetId)
				case p.DatacenterSiteId > 0:
					agg = outsideRack(p.DatacenterSiteId)
				}
				if agg == nil {
					continue
				}
				name := firstNonEmpty(st.ShortName, st.SerialNum, fmt.Sprintf("storage-%d", st.Id))
				key := fmt.Sprintf("storage:%d", st.Id)
				power := 2
				if st.Power == 1 {
					power = 1
				}
				sizeU := p.SizeUnit
				if sizeU <= 0 && p.EndUnitPos >= p.StartUnitPos && p.StartUnitPos > 0 {
					sizeU = p.EndUnitPos - p.StartUnitPos + 1
				}
				if isOutside(agg) || p.RackCabinetId == 0 {
					agg.shelf = append(agg.shelf, dcShelfItem{
						Kind: "storage", Id: st.Id, Name: name,
						Vendor: strings.TrimSpace(st.Vendor + " " + st.Model), U: sizeU, Power: power,
						Blades: []dcShelfBlade{}, Key: key, Rows: itSysStorageRows(st, stVia[st.Id]), Links: itSysStorageLinks(st),
					})
				} else {
					guess := false
					if sizeU <= 0 && p.StartUnitPos > 0 {
						sizeU, guess = 2, true
					}
					agg.eq = append(agg.eq, rackViewItem{
						Key: key, Kind: "storage", Id: st.Id, Name: name, Vendor: st.Vendor, Model: st.Model,
						StartU: p.StartUnitPos, SizeU: sizeU, Power: power,
						Rows: itSysStorageRows(st, stVia[st.Id]), Links: itSysStorageLinks(st), SizeGuess: guess,
						Blades: []rackViewBlade{},
					})
				}
				if i, ok := listByKey[key]; ok && listEquip[i].RackId == 0 {
					listEquip[i].EqKey = key
					listEquip[i].RackId = agg.rc.Id
					listEquip[i].RackName = agg.rc.Name
					listEquip[i].SiteId = agg.rc.DatacenterSiteId
				}
			}
		}
	}

	// ---- площадки ----
	siteIDs := map[int64]bool{}
	for rid, a := range aggs {
		if len(a.eq) > 0 || len(a.shelf) > 0 {
			siteIDs[siteOfRack[rid]] = true
		}
	}
	siteName := map[int64]string{}
	if len(siteIDs) > 0 {
		ids := make([]int64, 0, len(siteIDs))
		for id := range siteIDs {
			ids = append(ids, id)
		}
		var sites []models.DatacenterSite
		if _, err := o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", ids).Limit(-1).All(&sites, "Id", "Name"); err == nil {
			for _, s := range sites {
				siteName[s.Id] = s.Name
			}
		}
	}
	for i := range listEquip {
		if listEquip[i].SiteId > 0 {
			listEquip[i].SiteName = siteName[listEquip[i].SiteId]
		}
	}

	// мощность реальных шкафов - для световых столбов, как в обычном просмотре площадки
	realRackIDs := make([]int64, 0, len(aggs))
	for rid, a := range aggs {
		if rid > 0 && (len(a.eq) > 0 || len(a.shelf) > 0) {
			realRackIDs = append(realRackIDs, rid)
		}
	}
	rackPower := rackpower.RackCabinetPowerTotalsBatch(o, realRackIDs) // пустой список -> пустая карта

	for sid := range siteIDs {
		name := firstNonEmpty(siteName[sid], fmt.Sprintf("площадка-%d", sid))
		sc := &dcViewData{SiteId: sid, SiteName: name, Racks: []dcViewRack{}}
		equip := 0
		for rid, a := range aggs {
			if siteOfRack[rid] != sid || (len(a.eq) == 0 && len(a.shelf) == 0) {
				continue
			}
			u := int(a.rc.UHeight)
			if u <= 0 {
				u = 42
			}
			placeUnpositioned(a.eq, u)
			r := dcViewRack{Id: rid, Name: a.rc.Name, UHeight: u, Eq: a.eq}
			for _, e := range a.eq {
				switch e.Kind {
				case "server":
					r.Servers++
				case "chassis":
					r.Chassis++
					r.Blades += len(e.Blades)
				case "storage":
					r.Storages++
				}
			}
			if len(a.shelf) > 0 {
				r.Shelf = true
				r.Items = a.shelf
				for _, it := range a.shelf {
					switch it.Kind {
					case "server":
						r.Servers++
					case "chassis":
						r.Chassis++
						r.Blades += len(it.Blades)
					case "storage":
						r.Storages++
					}
				}
				units := r.Servers*2 + r.Chassis*7 + r.Storages*2
				r.ShelfUnits = (units + 34) / 35
				if r.ShelfUnits < 1 {
					r.ShelfUnits = 1
				}
			}
			equip += len(a.eq) + len(a.shelf)
			if p, ok := rackPower[rid]; ok {
				r.PowerAvgW, r.PowerPeakW, r.PowerRated = p.Avg24hW, p.Peak24hW, p.RatedMaxW
				sc.PowerAvgW += p.Avg24hW
				sc.PowerPeakW += p.Peak24hW
			}
			sc.Racks = append(sc.Racks, r)
		}
		sort.SliceStable(sc.Racks, func(i, j int) bool { return naturalLess(sc.Racks[i].Name, sc.Racks[j].Name) })
		out.scenes[sid] = sc
		out.order = append(out.order, sid)
		out.scene.Sites = append(out.scene.Sites, itSysSiteInfo{Id: sid, Name: name, Racks: len(sc.Racks), Equip: equip})
		out.scene.Stats.Racks += len(sc.Racks)
	}
	sort.SliceStable(out.order, func(i, j int) bool { return naturalLess(siteName[out.order[i]], siteName[out.order[j]]) })
	sort.SliceStable(out.scene.Sites, func(i, j int) bool { return naturalLess(out.scene.Sites[i].Name, out.scene.Sites[j].Name) })
	out.scene.Stats.Sites = len(out.scene.Sites)

	sort.SliceStable(listEquip, func(i, j int) bool {
		ki, kj := kindOrderITSys(listEquip[i].Kind), kindOrderITSys(listEquip[j].Kind)
		if ki != kj {
			return ki < kj
		}
		return naturalLess(listEquip[i].Name, listEquip[j].Name)
	})
	out.scene.Equipment = listEquip
	return out, nil
}

// placeUnpositioned раскладывает оборудование без позиции (StartU=0) в свободные юниты шкафа
// сверху вниз - чтобы оно было видно на сцене; позиция помечается как оценочная.
func placeUnpositioned(eq []rackViewItem, uHeight int) {
	busy := make([]bool, uHeight+2)
	for _, e := range eq {
		if e.StartU <= 0 {
			continue
		}
		size := e.SizeU
		if size <= 0 {
			size = 1
		}
		for u := e.StartU; u < e.StartU+size && u <= uHeight; u++ {
			busy[u] = true
		}
	}
	for i := range eq {
		if eq[i].StartU > 0 {
			continue
		}
		size := eq[i].SizeU
		if size <= 0 {
			size = 2
			if eq[i].Kind == "chassis" {
				size = 10
			}
		}
		for top := uHeight; top-size+1 >= 1; top-- {
			free := true
			for u := top; u > top-size; u-- {
				if busy[u] {
					free = false
					break
				}
			}
			if !free {
				continue
			}
			for u := top; u > top-size; u-- {
				busy[u] = true
			}
			eq[i].StartU = top - size + 1
			eq[i].SizeU = size
			eq[i].SizeGuess = true
			eq[i].Rows = append([]rackViewKV{{K: "Позиция в шкафу", V: "не указана в CMDB (показано условно)"}}, eq[i].Rows...)
			break
		}
	}
}

func itSysServerKey(hw models.ServerHardware, spr models.ServerSprRecords) string {
	if spr.Id > 0 {
		return fmt.Sprintf("server:%d", spr.Id)
	}
	return fmt.Sprintf("hw:%d", hw.Id)
}

func itSysSprRows(s models.ServerSprRecords) []rackViewKV {
	var rows []rackViewKV
	rows = rackViewKVAdd(rows, "FQDN", s.NameFQDN)
	rows = rackViewKVAdd(rows, "IP", s.IPAddress)
	rows = rackViewKVAdd(rows, "ОС", s.OS)
	rows = rackViewKVAdd(rows, "Роль", s.Role)
	return rows
}

func itSysStorageRows(st models.StorageSystems, via string) []rackViewKV {
	var rows []rackViewKV
	rows = rackViewKVAdd(rows, "Вендор", st.Vendor)
	rows = rackViewKVAdd(rows, "Модель", st.Model)
	rows = rackViewKVAdd(rows, "Серийный номер", st.SerialNum)
	rows = rackViewKVAdd(rows, "IP управления", st.IpAddress)
	rows = rackViewKVAdd(rows, "Связь с ИТ-системой", via)
	return rows
}

func itSysStorageLinks(st models.StorageSystems) []rackViewLink {
	return []rackViewLink{
		{Title: "Редактировать СХД", URL: fmt.Sprintf("/sprsrv/storagesystemsedit-tab/%d", st.Id)},
		{Title: "Конфигурация СХД", URL: fmt.Sprintf("/sprsrv/storagesystemconfigview/%d", st.Id)},
		{Title: "Метрики СХД", URL: fmt.Sprintf("/sprsrv/storagesystemmetricsrecent/%d", st.Id)},
	}
}

func kindOrderITSys(k string) int {
	switch k {
	case "server":
		return 0
	case "storage":
		return 1
	case "vm":
		return 2
	default:
		return 9
	}
}

func typeLabelITSys(t int) string {
	switch t {
	case 1:
		return "Виртуальная машина"
	case 2:
		return "Гипервизор"
	default:
		return "Аппаратный сервер"
	}
}

func itSysShortHost(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if i := strings.IndexByte(name, '.'); i > 0 {
		return name[:i]
	}
	return name
}

func volumeMentionsServer(serverField string, names map[string]bool) bool {
	for _, part := range strings.Split(serverField, ";") {
		p := strings.ToLower(itSysShortHost(part))
		if p != "" && names[p] {
			return true
		}
	}
	return false
}

func loadVolumesByUIDOrVirtualName(o orm.Ormer, uids, vnames []string) []models.StorageVolumes {
	seen := map[int64]bool{}
	var out []models.StorageVolumes
	appendUnique := func(vols []models.StorageVolumes) {
		for _, v := range vols {
			if seen[v.Id] {
				continue
			}
			seen[v.Id] = true
			out = append(out, v)
		}
	}
	const chunk = 200
	load := func(field string, vals []string) {
		for i := 0; i < len(vals); i += chunk {
			end := i + chunk
			if end > len(vals) {
				end = len(vals)
			}
			var vols []models.StorageVolumes
			if _, err := o.QueryTable(new(models.StorageVolumes)).Filter(field+"__in", vals[i:end]).Limit(-1).All(&vols); err == nil {
				appendUnique(vols)
			}
		}
	}
	load("UID", uids)
	load("VirtualName", vnames)
	return out
}

func itSysServerCard(hw models.ServerHardware, spr models.ServerSprRecords) ([]rackViewKV, []rackViewLink, string) {
	name := firstNonEmpty(spr.Name, hw.SerialNum, hw.Model, fmt.Sprintf("srv-%d", hw.Id))
	var rows []rackViewKV
	rows = rackViewKVAdd(rows, "FQDN", spr.NameFQDN)
	rows = rackViewKVAdd(rows, "IP сервера", spr.IPAddress)
	rows = rackViewKVAdd(rows, "Серийный номер", hw.SerialNum)
	rows = rackViewKVAdd(rows, "Модель", hw.Model)
	rows = rackViewKVAdd(rows, "IP iLO/MP", hw.MpIPAddress)
	if spr.Type == 2 {
		rows = rackViewKVAdd(rows, "Роль", "Гипервизор VMware")
	}
	var links []rackViewLink
	if spr.Id > 0 {
		links = append(links, rackViewLink{Title: "Общие данные по серверу", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", spr.Id)})
	}
	links = append(links, rackViewLink{Title: "Карточка оборудования", URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", hw.Id)})
	return rows, links, name
}
