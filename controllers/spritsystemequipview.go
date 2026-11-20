package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/itsystem"
	"loadmon/models"
	"sort"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// itSysEquipListItem — строка списка оборудования справа.
type itSysEquipListItem struct {
	Kind     string         `json:"kind"` // server|vm|storage|chassis
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

// itSysEquipRack — шкаф только с оборудованием ИТ-системы.
type itSysEquipRack struct {
	Id      int64          `json:"id"`
	Name    string         `json:"name"`
	UHeight int            `json:"uHeight"`
	Items   []rackViewItem `json:"items"`
}

// itSysEquipSite — площадка с задействованными шкафами.
type itSysEquipSite struct {
	Id   int64            `json:"id"`
	Name string           `json:"name"`
	Racks []itSysEquipRack `json:"racks"`
}

// itSysEquipViewData — JSON страницы визуализации ИТ-системы.
type itSysEquipViewData struct {
	SystemKey  string               `json:"systemKey"`
	SystemName string               `json:"systemName"`
	Sites      []itSysEquipSite     `json:"sites"`
	Equipment  []itSysEquipListItem `json:"equipment"`
	Stats      struct {
		Servers  int `json:"servers"`
		VMs      int `json:"vms"`
		Storages int `json:"storages"`
		Racks    int `json:"racks"`
		Sites    int `json:"sites"`
	} `json:"stats"`
}

// SprITSystemEquipView — визуализация оборудования ИТ-системы: площадки и
// шкафы только с задействованным оборудованием (серверы + СХД по LUN ВМ /
// по имени аппаратного сервера), список справа, карточка по наведению.
//
// Маршрут: /sprsrv/itsystemequipview
func (this *MainController) SprITSystemEquipView() {
	RegisterRequest(this, LogTypeViewLoadTable)
	this.activeContent("sprsrv/itsystemequipview")

	key := strings.TrimSpace(this.GetString("ITSystemKey"))
	if key == "" {
		key = strings.TrimSpace(this.GetString("itsystem"))
	}
	this.Data["ITSystemKey"] = key
	this.Data["ITSystemSpr"] = itsystem.ImportITSystemSpr()
	this.Data["ITJSON"] = template.JS("null")
	this.Data["HasSystem"] = key != ""

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

	data, err := buildITSystemEquipView(orm.NewOrm(), key, sysName)
	this.Data["Stats"] = data.Stats
	if err != nil {
		flash.Error("Ошибка построения визуализации: %v", err)
		return
	}
	if len(data.Equipment) == 0 {
		flash.Notice("Для ИТ-системы %s не найдено серверов/СХД в CMDB", key)
	}
	b, jerr := json.Marshal(data)
	if jerr != nil {
		flash.Error("Ошибка подготовки JSON: %v", jerr)
		return
	}
	this.Data["ITJSON"] = template.JS(b)
}

func buildITSystemEquipView(o orm.Ormer, key, sysName string) (itSysEquipViewData, error) {
	out := itSysEquipViewData{SystemKey: key, SystemName: sysName, Sites: []itSysEquipSite{}, Equipment: []itSysEquipListItem{}}

	var sprs []models.ServerSprRecords
	cond := orm.NewCondition().And("ITSystemKey", key).AndNot("Active", 2)
	if _, err := models.ServerSprQS(o).SetCond(models.ExcludeDestroyedCond(cond)).Limit(-1).
		All(&sprs, "Id", "Name", "NameFQDN", "Type", "VMSprId", "HWSprId", "IPAddress", "OS", "Role", "Active"); err != nil {
		return out, err
	}

	hwByID := map[int64]models.ServerHardware{}
	hwIDs := make([]int64, 0)
	vmSprIDs := make([]int64, 0)
	hwNames := map[string]bool{} // short lower names for LUN.Server match
	sprByHW := map[int64]models.ServerSprRecords{}
	listEquip := make([]itSysEquipListItem, 0, len(sprs))

	for _, s := range sprs {
		switch s.Type {
		case 1: // VM
			out.Stats.VMs++
			if s.VMSprId > 0 {
				vmSprIDs = append(vmSprIDs, s.VMSprId)
			}
			listEquip = append(listEquip, itSysEquipListItem{
				Kind: "vm", Id: s.Id, Name: firstNonEmpty(s.Name, s.NameFQDN, fmt.Sprintf("vm-%d", s.Id)),
				Subtitle: "Виртуальная машина",
				Rows: []rackViewKV{
					{K: "FQDN", V: s.NameFQDN},
					{K: "IP", V: s.IPAddress},
					{K: "ОС", V: s.OS},
					{K: "Роль", V: s.Role},
				},
				Links: []rackViewLink{
					{Title: "Карточка сервера", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", s.Id)},
				},
			})
		default: // 0 hardware, 2 hypervisor
			out.Stats.Servers++
			short := strings.ToLower(itSysShortHost(s.Name))
			if short != "" {
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
			item := itSysEquipListItem{
				Kind: "server", Id: s.Id, Name: firstNonEmpty(s.Name, s.NameFQDN, fmt.Sprintf("srv-%d", s.Id)),
				Subtitle: typeLabelITSys(s.Type),
				Rows: []rackViewKV{
					{K: "FQDN", V: s.NameFQDN},
					{K: "IP", V: s.IPAddress},
					{K: "ОС", V: s.OS},
					{K: "Роль", V: s.Role},
				},
				Links: []rackViewLink{
					{Title: "Общие данные по серверу", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", s.Id)},
				},
			}
			if hwID > 0 {
				item.Links = append(item.Links, rackViewLink{Title: "Карточка оборудования", URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", hwID)})
				hwIDs = append(hwIDs, hwID)
				sprByHW[hwID] = s
			}
			listEquip = append(listEquip, item)
		}
	}

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

	// СХД по LUN виртуальных машин ИТ-системы.
	stIDs := map[int64]bool{}
	stVia := map[int64]string{} // storageId → как нашли
	if len(vmSprIDs) > 0 {
		var usages []models.VMUsageDatastore
		if _, err := o.QueryTable(new(models.VMUsageDatastore)).Filter("VMSprId__in", vmSprIDs).Limit(-1).
			All(&usages, "VMSprId", "UID", "StorageName", "Datastore"); err == nil {
			uids := make([]string, 0)
			vnames := make([]string, 0)
			uidSet, vnSet := map[string]bool{}, map[string]bool{}
			for _, u := range usages {
				if uid := strings.TrimSpace(u.UID); uid != "" && !uidSet[uid] {
					uidSet[uid] = true
					uids = append(uids, uid)
				}
				sn := strings.TrimSpace(u.StorageName)
				if sn == "" {
					sn = strings.TrimSpace(u.Datastore)
				}
				if sn != "" && !vnSet[sn] {
					vnSet[sn] = true
					vnames = append(vnames, sn)
				}
			}
			vols := loadVolumesByUIDOrVirtualName(o, uids, vnames)
			for _, v := range vols {
				if v.StorageSystemsId <= 0 {
					continue
				}
				stIDs[v.StorageSystemsId] = true
				stVia[v.StorageSystemsId] = "LUN ВМ"
			}
		}
	}

	// СХД по связке LUN.Server ↔ имя аппаратного сервера ИТ-системы.
	if len(hwNames) > 0 {
		for name := range hwNames {
			if len(name) < 2 {
				continue
			}
			var vols []models.StorageVolumes
			if _, err := o.QueryTable(new(models.StorageVolumes)).Filter("Server__icontains", name).Limit(-1).
				All(&vols, "Id", "StorageSystemsId", "Server", "VirtualName", "UID"); err != nil {
				continue
			}
			for _, v := range vols {
				if v.StorageSystemsId <= 0 || !volumeMentionsServer(v.Server, hwNames) {
					continue
				}
				stIDs[v.StorageSystemsId] = true
				if stVia[v.StorageSystemsId] == "" {
					stVia[v.StorageSystemsId] = "LUN ↔ сервер"
				} else if !strings.Contains(stVia[v.StorageSystemsId], "сервер") {
					stVia[v.StorageSystemsId] += "; LUN ↔ сервер"
				}
			}
		}
	}

	stIDList := make([]int64, 0, len(stIDs))
	for id := range stIDs {
		stIDList = append(stIDList, id)
	}
	stByID := map[int64]models.StorageSystems{}
	if len(stIDList) > 0 {
		var sts []models.StorageSystems
		if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Id__in", stIDList).Limit(-1).All(&sts); err == nil {
			for _, st := range sts {
				if st.Status == 6 || models.ItiliumStatusHiddenFromPlans(st.ItiliumStatus) {
					continue
				}
				stByID[st.Id] = st
				out.Stats.Storages++
				listEquip = append(listEquip, itSysEquipListItem{
					Kind: "storage", Id: st.Id, Name: firstNonEmpty(st.ShortName, st.SerialNum, fmt.Sprintf("storage-%d", st.Id)),
					Subtitle: "СХД (" + stVia[st.Id] + ")",
					Rows: []rackViewKV{
						{K: "Вендор", V: st.Vendor},
						{K: "Модель", V: st.Model},
						{K: "SN", V: st.SerialNum},
						{K: "IP", V: st.IpAddress},
						{K: "Источник связи", V: stVia[st.Id]},
					},
					Links: []rackViewLink{
						{Title: "Редактировать СХД", URL: fmt.Sprintf("/sprsrv/storagesystemsedit-tab/%d", st.Id)},
						{Title: "Конфигурация СХД", URL: fmt.Sprintf("/sprsrv/storagesystemconfigview/%d", st.Id)},
					},
				})
			}
		}
	}

	// Размещения: серверы (через HW / корзину) и СХД.
	rackItems := map[int64][]rackViewItem{} // rackId → items
	rackMeta := map[int64]models.RackCabinet{}
	siteOnlyStorage := map[int64][]rackViewItem{} // siteId → unplaced storage items

	ensureRack := func(rackID int64) (models.RackCabinet, bool) {
		if rackID <= 0 {
			return models.RackCabinet{}, false
		}
		if rc, ok := rackMeta[rackID]; ok {
			return rc, true
		}
		var rc models.RackCabinet
		if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", rackID).One(&rc); err != nil {
			return models.RackCabinet{}, false
		}
		rackMeta[rackID] = rc
		return rc, true
	}

	resolveHWRackID := func(hw models.ServerHardware) int64 {
		if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
			var ch models.BladeSystems
			if err := o.QueryTable(new(models.BladeSystems)).Filter("Id", hw.BladeSystemId).One(&ch, "RackCabinetId"); err == nil && ch.RackCabinetId > 0 {
				return ch.RackCabinetId
			}
		}
		return hw.RackCabinetId
	}

	for hwID, hw := range hwByID {
		spr := sprByHW[hwID]
		rackID := resolveHWRackID(hw)
		rc, ok := ensureRack(rackID)
		if !ok {
			continue
		}
		rows, links, name := itSysServerCard(hw, spr)
		sizeU := hw.SizeUnit
		guess := false
		if sizeU <= 0 && hw.StartUnitPos > 0 {
			sizeU = rackGuessSizeU(hw.Model)
			guess = true
		}
		rackItems[rackID] = append(rackItems[rackID], rackViewItem{
			Kind: "server", Id: hw.Id, Name: name, Vendor: hw.Vendor, Model: hw.Model,
			StartU: hw.StartUnitPos, SizeU: sizeU, Power: rackViewPower(hw.PowerState),
			Rows: rows, Links: links, SizeGuess: guess,
		})
		for i := range listEquip {
			if listEquip[i].Kind == "server" && listEquip[i].Id == spr.Id {
				listEquip[i].RackId = rc.Id
				listEquip[i].RackName = rc.Name
				listEquip[i].SiteId = rc.DatacenterSiteId
				listEquip[i].Links = dedupeLinks(append(listEquip[i].Links,
					rackViewLink{Title: "Карточка оборудования", URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", hw.Id)}))
				break
			}
		}
	}

	if len(stByID) > 0 {
		stList := make([]int64, 0, len(stByID))
		for id := range stByID {
			stList = append(stList, id)
		}
		var placements []models.StorageSystemPlacement
		if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId__in", stList).Limit(-1).All(&placements); err == nil {
			for _, p := range placements {
				st, ok := stByID[p.StorageSystemsId]
				if !ok {
					continue
				}
				rows := []rackViewKV{
					{K: "Вендор", V: st.Vendor},
					{K: "Модель", V: st.Model},
					{K: "SN", V: st.SerialNum},
					{K: "Источник связи", V: stVia[st.Id]},
				}
				links := []rackViewLink{
					{Title: "Редактировать СХД", URL: fmt.Sprintf("/sprsrv/storagesystemsedit-tab/%d", st.Id)},
					{Title: "Конфигурация СХД", URL: fmt.Sprintf("/sprsrv/storagesystemconfigview/%d", st.Id)},
				}
				name := firstNonEmpty(st.ShortName, st.SerialNum, fmt.Sprintf("storage-%d", st.Id))
				sizeU := p.SizeUnit
				if sizeU <= 0 && p.EndUnitPos >= p.StartUnitPos && p.StartUnitPos > 0 {
					sizeU = p.EndUnitPos - p.StartUnitPos + 1
				}
				item := rackViewItem{
					Kind: "storage", Id: st.Id, Name: name, Vendor: st.Vendor, Model: st.Model,
					StartU: p.StartUnitPos, SizeU: sizeU, Power: 1,
					Rows: rows, Links: links,
				}
				if p.RackCabinetId > 0 {
					rc, ok := ensureRack(p.RackCabinetId)
					if !ok {
						continue
					}
					rackItems[p.RackCabinetId] = append(rackItems[p.RackCabinetId], item)
					for i := range listEquip {
						if listEquip[i].Kind == "storage" && listEquip[i].Id == st.Id {
							listEquip[i].RackId = rc.Id
							listEquip[i].RackName = rc.Name
							listEquip[i].SiteId = rc.DatacenterSiteId
							break
						}
					}
				} else if p.DatacenterSiteId > 0 {
					siteOnlyStorage[p.DatacenterSiteId] = append(siteOnlyStorage[p.DatacenterSiteId], item)
					for i := range listEquip {
						if listEquip[i].Kind == "storage" && listEquip[i].Id == st.Id {
							listEquip[i].SiteId = p.DatacenterSiteId
							listEquip[i].RackName = "(без шкафа)"
							break
						}
					}
				}
			}
		}
	}

	// Собрать площадки.
	siteIDs := map[int64]bool{}
	for _, rc := range rackMeta {
		if rc.DatacenterSiteId > 0 {
			siteIDs[rc.DatacenterSiteId] = true
		}
	}
	for sid := range siteOnlyStorage {
		siteIDs[sid] = true
	}
	sites := make([]models.DatacenterSite, 0, len(siteIDs))
	if len(siteIDs) > 0 {
		ids := make([]int64, 0, len(siteIDs))
		for id := range siteIDs {
			ids = append(ids, id)
		}
		_, _ = o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", ids).Limit(-1).All(&sites)
	}
	siteByID := map[int64]models.DatacenterSite{}
	for _, s := range sites {
		siteByID[s.Id] = s
	}

	// Дописать имена площадок в список.
	for i := range listEquip {
		if listEquip[i].SiteId > 0 {
			if s, ok := siteByID[listEquip[i].SiteId]; ok {
				listEquip[i].SiteName = s.Name
			}
		}
	}

	outSites := make([]itSysEquipSite, 0, len(siteIDs))
	for sid := range siteIDs {
		site := siteByID[sid]
		name := site.Name
		if name == "" {
			name = fmt.Sprintf("площадка-%d", sid)
		}
		es := itSysEquipSite{Id: sid, Name: name, Racks: []itSysEquipRack{}}
		// шкафы этой площадки с оборудованием
		rackIDs := make([]int64, 0)
		for rid, rc := range rackMeta {
			if rc.DatacenterSiteId == sid && len(rackItems[rid]) > 0 {
				rackIDs = append(rackIDs, rid)
			}
		}
		sort.Slice(rackIDs, func(i, j int) bool {
			return naturalLess(rackMeta[rackIDs[i]].Name, rackMeta[rackIDs[j]].Name)
		})
		for _, rid := range rackIDs {
			rc := rackMeta[rid]
			u := int(rc.UHeight)
			if u <= 0 {
				u = 42
			}
			items := rackItems[rid]
			sort.SliceStable(items, func(i, j int) bool {
				if items[i].StartU != items[j].StartU {
					return items[i].StartU > items[j].StartU
				}
				return items[i].Name < items[j].Name
			})
			es.Racks = append(es.Racks, itSysEquipRack{Id: rid, Name: rc.Name, UHeight: u, Items: items})
			out.Stats.Racks++
		}
		// СХД только на площадке — виртуальный «стеллаж»
		if extra := siteOnlyStorage[sid]; len(extra) > 0 {
			es.Racks = append(es.Racks, itSysEquipRack{
				Id: 0, Name: models.RackCabinetOutsideName, UHeight: 42, Items: extra,
			})
			out.Stats.Racks++
		}
		if len(es.Racks) > 0 {
			outSites = append(outSites, es)
		}
	}
	sort.Slice(outSites, func(i, j int) bool { return naturalLess(outSites[i].Name, outSites[j].Name) })

	sort.SliceStable(listEquip, func(i, j int) bool {
		ki, kj := kindOrderITSys(listEquip[i].Kind), kindOrderITSys(listEquip[j].Kind)
		if ki != kj {
			return ki < kj
		}
		return naturalLess(listEquip[i].Name, listEquip[j].Name)
	})

	out.Sites = outSites
	out.Equipment = listEquip
	out.Stats.Sites = len(outSites)
	return out, nil
}

func kindOrderITSys(k string) int {
	switch k {
	case "server":
		return 0
	case "chassis":
		return 1
	case "storage":
		return 2
	case "vm":
		return 3
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
	for i := 0; i < len(uids); i += chunk {
		end := i + chunk
		if end > len(uids) {
			end = len(uids)
		}
		var vols []models.StorageVolumes
		if _, err := o.QueryTable(new(models.StorageVolumes)).Filter("UID__in", uids[i:end]).Limit(-1).All(&vols); err == nil {
			appendUnique(vols)
		}
	}
	for i := 0; i < len(vnames); i += chunk {
		end := i + chunk
		if end > len(vnames) {
			end = len(vnames)
		}
		var vols []models.StorageVolumes
		if _, err := o.QueryTable(new(models.StorageVolumes)).Filter("VirtualName__in", vnames[i:end]).Limit(-1).All(&vols); err == nil {
			appendUnique(vols)
		}
	}
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

func dedupeLinks(in []rackViewLink) []rackViewLink {
	seen := map[string]bool{}
	out := make([]rackViewLink, 0, len(in))
	for _, l := range in {
		k := l.URL
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, l)
	}
	return out
}
