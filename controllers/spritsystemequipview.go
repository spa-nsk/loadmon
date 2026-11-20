package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/config"
	"loadmon/grpcclient"
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
	Kind     string         `json:"kind"` // server|host|vm|storage
	Role     string         `json:"role,omitempty"`
	Cluster  string         `json:"cluster,omitempty"`
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

// itSysSiteInfo - площадка на сцене режима (все площадки показываются одновременно).
type itSysSiteInfo struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Racks int    `json:"racks"`
	Equip int    `json:"equip"`
}

// itSysScene - данные режима «Оборудование ИТ-системы» внутри dcViewData.
type itSysScene struct {
	SystemKey  string               `json:"systemKey"`
	SystemName string               `json:"systemName"`
	Sites      []itSysSiteInfo      `json:"sites"`
	Equipment  []itSysEquipListItem `json:"equipment"`
	Stats      itSysStats           `json:"stats"`
}

type itSysStats struct {
	Servers  int `json:"servers"`
	Hosts    int `json:"hosts"` // хосты ESXi (хосты размещения ВМ и остальные хосты их кластеров)
	VMs      int `json:"vms"`
	Clusters int `json:"clusters"`
	Storages int `json:"storages"`
	Racks    int `json:"racks"`
	Sites    int `json:"sites"`
}

// itSysBuild - результат разбора ИТ-системы: шкафы всех площадок + общий список.
type itSysBuild struct {
	scene itSysScene
	racks []dcViewRack // шкафы всех площадок (сгруппированы по площадкам, у шкафа SiteId)
	data  dcViewData
}

// itHost - хост ESXi: относится к ИТ-системе, является хостом размещения её ВМ или
// просто членом кластера, в котором работают ВМ системы.
type itHost struct {
	spr     models.ServerSprRecords
	inSys   bool     // сам записан в ИТ-систему
	cluster string   // имя кластера
	vms     []string // имена ВМ ИТ-системы, которые сейчас на этом хосте
}

// itHostRole: "host" - на хосте ВМ ИТ-системы; "" - гипервизор самой системы; "cluster" - прочие хосты кластера.
func itHostRole(h *itHost) string {
	switch {
	case h == nil:
		return ""
	case len(h.vms) > 0:
		return "host"
	case h.inSys:
		return ""
	default:
		return "cluster"
	}
}

func itHostRoleLabel(h *itHost) string {
	switch itHostRole(h) {
	case "host":
		return "Хост размещения ВМ ИТ-системы"
	case "cluster":
		return "Хост кластера (ВМ ИТ-системы сейчас нет)"
	default:
		return "Гипервизор ИТ-системы"
	}
}

func itHostRows(h *itHost) []rackViewKV {
	var rows []rackViewKV
	rows = rackViewKVAdd(rows, "Роль", itHostRoleLabel(h))
	rows = rackViewKVAdd(rows, "Кластер", h.cluster)
	rows = rackViewKVAdd(rows, "vCenter", h.spr.Vcenter)
	if n := len(h.vms); n > 0 {
		names := append([]string(nil), h.vms...)
		sort.Strings(names)
		more := ""
		if len(names) > 12 {
			names, more = names[:12], fmt.Sprintf(" … (+%d)", len(h.vms)-12)
		}
		rows = rackViewKVAdd(rows, "ВМ ИТ-системы на хосте", fmt.Sprintf("%d: %s%s", n, strings.Join(names, ", "), more))
	}
	return rows
}

// SprITSystemEquipView - «Оборудование ИТ-системы»: та же изометрическая 3D-сцена, что
// в «Оборудование площадки/ЦОД» (datacentersiteview.tpl), но показываются только шкафы
// с оборудованием выбранной ИТ-системы; юниты системы подсвечены, по наведению -
// расширенная карточка со ссылками на формы редактирования, справа - список оборудования.
// СХД определяются: для ВМ - по LUN, использованным ВМ; для аппаратных серверов - по
// связке LUN.Server с именем сервера.
//
// Все площадки с оборудованием системы показываются на одной сцене (подписаны, связаны
// линиями связи); для ВМ показываются хосты ESXi и все хосты их кластеров.
//
// Маршрут: /sprsrv/itsystemequipview?ITSystemKey=...[&focus=<ключ оборудования>]
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

	scene := b.scene
	data := b.data
	data.IT = &scene
	this.Data["RackCount"] = len(data.Racks)

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
		scene: itSysScene{SystemKey: key, SystemName: sysName, Sites: []itSysSiteInfo{}, Equipment: []itSysEquipListItem{}},
		data:  dcViewData{Racks: []dcViewRack{}},
	}

	var sprs []models.ServerSprRecords
	cond := orm.NewCondition().And("ITSystemKey", key).AndNot("Active", 2)
	if _, err := models.ServerSprQS(o).SetCond(models.ExcludeDestroyedCond(cond)).Limit(-1).
		All(&sprs, itSysSprCols...); err != nil {
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

	// addServerItem - строка списка и привязка к ServerHardware для сервера/хоста (ServerSprRecords Type 0/2)
	addServerItem := func(s models.ServerSprRecords, kind, subtitle string, extra []rackViewKV) {
		hwID := s.HWSprId
		if hwID <= 0 {
			var hw models.ServerHardware
			if err := o.QueryTable(new(models.ServerHardware)).Filter("ServerId", s.Id).One(&hw, "Id"); err == nil {
				hwID = hw.Id
			}
		}
		it := itSysEquipListItem{
			Key: fmt.Sprintf("server:%d", s.Id), Kind: kind, Id: s.Id,
			Name:     firstNonEmpty(s.Name, s.NameFQDN, fmt.Sprintf("srv-%d", s.Id)),
			Subtitle: subtitle,
			Rows:     append(extra, itSysSprRows(s)...),
			Links:    []rackViewLink{{Title: "Общие данные по серверу", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", s.Id)}},
		}
		if hwID > 0 {
			it.Links = append(it.Links, rackViewLink{Title: "Карточка оборудования", URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", hwID)})
			hwIDs = append(hwIDs, hwID)
			sprByHW[hwID] = s
		}
		addList(it)
	}

	type vmRef struct {
		spr models.ServerSprRecords
		key string // ключ строки списка
	}
	var vmRefs []vmRef
	hosts := map[int64]*itHost{} // ServerSprRecords.Id хоста -> хост
	for _, s := range sprs {
		switch s.Type {
		case 1: // виртуальная машина
			out.scene.Stats.VMs++
			if s.VMSprId > 0 {
				vmSprIDs = append(vmSprIDs, s.VMSprId)
			}
			k := fmt.Sprintf("vm:%d", s.Id)
			vmRefs = append(vmRefs, vmRef{spr: s, key: k})
			addList(itSysEquipListItem{
				Key: k, Kind: "vm", Id: s.Id,
				Name:     firstNonEmpty(s.Name, s.NameFQDN, fmt.Sprintf("vm-%d", s.Id)),
				Subtitle: "Виртуальная машина",
				Rows:     itSysSprRows(s),
				Links:    []rackViewLink{{Title: "Карточка сервера", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", s.Id)}},
			})
		case 2: // хост ESXi, записанный в ИТ-систему
			hosts[s.Id] = &itHost{spr: s, inSys: true, cluster: s.Cluster}
			if short := strings.ToLower(itSysShortHost(s.Name)); short != "" {
				hwNames[short] = true
			}
			if fq := strings.ToLower(itSysShortHost(s.NameFQDN)); fq != "" {
				hwNames[fq] = true
			}
		default: // 0 - аппаратный сервер
			out.scene.Stats.Servers++
			if short := strings.ToLower(itSysShortHost(s.Name)); short != "" {
				hwNames[short] = true
			}
			if fq := strings.ToLower(itSysShortHost(s.NameFQDN)); fq != "" {
				hwNames[fq] = true
			}
			addServerItem(s, "server", typeLabelITSys(s.Type), nil)
		}
	}

	// ---- ВМ -> хост ESXi -> все хосты кластера ----
	// Хост ВМ - тот же алгоритм, что в карточке сервера (resolveVMPlacementText /sprsrv/updatebase):
	// актуальное имя ESXi (EsxHost) -> ServerSprRecords Type=2 по Name/NameFQDN (__iexact).
	// Vm_EsxHost из справочника здесь только запасной вариант: в карточке его намеренно не
	// используют (может отставать от vCenter). Для пакета ВМ: сначала один LoadVMList на
	// кластер (как фильтр размещения view2), затем GetVMByKeyDirect по оставшимся ВМ.
	type clKey struct{ vc, id string }
	clusters := map[clKey]string{}                // кластер -> имя
	vmRecByKey := map[string]models.VMSprRecord{} // ключ строки ВМ -> запись справочника ВМ
	esxByKey := map[string]string{}               // ключ строки ВМ -> имя хоста ESXi (живое или запасное)
	if len(vmRefs) > 0 {
		recCols := []string{"Id", "ServerId", "Vm_Name", "KeyName", "Vm_EsxHost", "Vcenter", "Datacenter", "Cluster", "ClusterId"}
		byID := map[int64]models.VMSprRecord{}
		if len(vmSprIDs) > 0 {
			var vrs []models.VMSprRecord
			if _, err := o.QueryTable(new(models.VMSprRecord)).Filter("Id__in", vmSprIDs).Limit(-1).All(&vrs, recCols...); err == nil {
				for _, v := range vrs {
					byID[v.Id] = v
				}
			}
		}
		byServer := map[int64]models.VMSprRecord{} // запасной путь: ВМ без VMSprId - по ссылке VMSprRecord.ServerId
		var noLink []int64
		for _, r := range vmRefs {
			if _, ok := byID[r.spr.VMSprId]; !ok {
				noLink = append(noLink, r.spr.Id)
			}
		}
		if len(noLink) > 0 {
			var vrs []models.VMSprRecord
			if _, err := o.QueryTable(new(models.VMSprRecord)).Filter("ServerId__in", noLink).Limit(-1).All(&vrs, recCols...); err == nil {
				for _, v := range vrs {
					byServer[v.ServerId] = v
				}
			}
		}
		for _, r := range vmRefs {
			v, ok := byID[r.spr.VMSprId]
			if !ok {
				v, ok = byServer[r.spr.Id]
			}
			if !ok {
				continue
			}
			vmRecByKey[r.key] = v
		}
		// 1) живой список ВМ кластера (EsxHost «сейчас»); индекс по KeyName и по имени ВМ
		type liveKey struct{ vc, dc, cl string }
		liveDone := map[liveKey]map[string]string{} // нормализованный ключ -> EsxHost
		liveIndex := func(lk liveKey) map[string]string {
			if m, ok := liveDone[lk]; ok {
				return m
			}
			m := map[string]string{}
			put := func(k, host string) {
				k, host = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(host)
				if k == "" || host == "" {
					return
				}
				m[k] = host
			}
			for _, p := range grpcclient.LoadVMList(lk.vc, lk.dc, lk.cl, "") {
				if p == nil {
					continue
				}
				h := strings.TrimSpace(p.EsxHost)
				put(p.KeyName, h)
				put(p.Name, h)
				if kn := strings.TrimSpace(p.KeyName); kn != "" {
					if i := strings.LastIndexByte(kn, '/'); i >= 0 {
						put(kn[i+1:], h) // последний сегмент KeyName = имя ВМ
					}
				}
			}
			liveDone[lk] = m
			return m
		}
		for _, r := range vmRefs {
			v, ok := vmRecByKey[r.key]
			if !ok || strings.TrimSpace(v.Vcenter) == "" || strings.TrimSpace(v.Cluster) == "" {
				continue
			}
			lk := liveKey{strings.TrimSpace(v.Vcenter), strings.TrimSpace(v.Datacenter), strings.TrimSpace(v.Cluster)}
			m := liveIndex(lk)
			for _, cand := range []string{v.KeyName, v.Vm_Name, r.spr.Name, r.spr.NameFQDN} {
				if h := m[strings.ToLower(strings.TrimSpace(cand))]; h != "" {
					esxByKey[r.key] = h
					break
				}
			}
		}
		// 2) прямой запрос к vCenter - ровно как resolveVMPlacementText в карточке сервера
		direct := 0
		for _, r := range vmRefs {
			v, ok := vmRecByKey[r.key]
			if !ok || esxByKey[r.key] != "" || direct >= itSysDirectVMLimit ||
				strings.TrimSpace(v.Vcenter) == "" || strings.TrimSpace(v.Vm_Name) == "" {
				continue
			}
			direct++
			if p, err := grpcclient.GetVMByKeyDirect(v.Vcenter, v.Datacenter, v.Cluster, v.Vm_Name); err == nil {
				if h := strings.TrimSpace(p.EsxHost); h != "" {
					esxByKey[r.key] = h
				}
			}
		}
		// 3) запасной вариант - последнее известное из справочника (карточка его не использует)
		for _, r := range vmRefs {
			if esxByKey[r.key] != "" {
				continue
			}
			if v, ok := vmRecByKey[r.key]; ok {
				esxByKey[r.key] = strings.TrimSpace(v.Vm_EsxHost)
			}
		}
	}
	// кластеры ВМ и гипервизоров системы; затем - все хосты этих кластеров
	for _, v := range vmRecByKey {
		if v.ClusterId != "" {
			clusters[clKey{v.Vcenter, v.ClusterId}] = firstNonEmpty(v.Cluster, v.ClusterId)
		}
	}
	for _, h := range hosts {
		if h.spr.ClusterId != "" {
			clusters[clKey{h.spr.Vcenter, h.spr.ClusterId}] = firstNonEmpty(h.spr.Cluster, h.spr.ClusterId)
		}
	}
	clusterLoaded := map[clKey]bool{}
	loadClusterMembers := func() {
		for ck, cname := range clusters { // оборудование всего кластера
			if clusterLoaded[ck] || ck.vc == "" || ck.id == "" {
				continue
			}
			clusterLoaded[ck] = true
			cc := orm.NewCondition().And("Type", 2).And("Vcenter", ck.vc).And("ClusterId", ck.id).AndNot("Active", 2)
			var members []models.ServerSprRecords
			if _, err := models.ServerSprQS(o).SetCond(models.ExcludeDestroyedCond(cc)).Limit(-1).All(&members, itSysSprCols...); err != nil {
				continue
			}
			for _, m := range members {
				if h, ok := hosts[m.Id]; ok {
					h.cluster = firstNonEmpty(h.cluster, cname)
					continue
				}
				hosts[m.Id] = &itHost{spr: m, cluster: firstNonEmpty(m.Cluster, cname)}
			}
		}
	}
	loadClusterMembers()
	hostByName := map[string]*itHost{}
	hostByShort := map[string]*itHost{} // короткое имя (до первой точки); при неоднозначности - nil
	indexHosts := func() {
		clear(hostByName)
		clear(hostByShort)
		for _, h := range hosts {
			for _, n := range []string{h.spr.Name, h.spr.NameFQDN} {
				n = strings.ToLower(strings.TrimSpace(n))
				if n == "" {
					continue
				}
				hostByName[n] = h
				sn := itSysShortHost(n)
				if old, ok := hostByShort[sn]; ok && old != h {
					hostByShort[sn] = nil
				} else {
					hostByShort[sn] = h
				}
			}
		}
	}
	indexHosts()
	lookupHost := func(esx string) *itHost {
		esx = strings.ToLower(strings.TrimSpace(esx))
		if esx == "" {
			return nil
		}
		if h := hostByName[esx]; h != nil {
			return h
		}
		return hostByShort[itSysShortHost(esx)]
	}
	// findHostSpr - поиск гипервизора по имени, как в resolveVMPlacementText (карточка сервера).
	findHostSpr := func(esx string, preferVC string) (models.ServerSprRecords, bool) {
		esx = strings.TrimSpace(esx)
		if esx == "" {
			return models.ServerSprRecords{}, false
		}
		c := orm.NewCondition().And("Type", 2).
			AndCond(orm.NewCondition().Or("Name__iexact", esx).Or("NameFQDN__iexact", esx))
		var found []models.ServerSprRecords
		if _, err := models.ServerSprQS(o).SetCond(models.ExcludeDestroyedCond(c)).Limit(5).All(&found, itSysSprCols...); err != nil || len(found) == 0 {
			// запасной путь: короткое имя (до точки) - EsxHost иногда без домена, NameFQDN - с доменом
			if sn := itSysShortHost(esx); sn != "" && !strings.EqualFold(sn, esx) {
				c2 := orm.NewCondition().And("Type", 2).
					AndCond(orm.NewCondition().Or("Name__iexact", sn).Or("NameFQDN__istartswith", sn+"."))
				if _, err := models.ServerSprQS(o).SetCond(models.ExcludeDestroyedCond(c2)).Limit(5).All(&found, itSysSprCols...); err != nil || len(found) == 0 {
					return models.ServerSprRecords{}, false
				}
			} else {
				return models.ServerSprRecords{}, false
			}
		}
		pick := found[0]
		if preferVC != "" {
			for _, f := range found {
				if f.Vcenter == preferVC {
					pick = f
					break
				}
			}
		}
		return pick, true
	}
	// хост ВМ не найден среди уже загруженных - ищем в справочнике серверов (как в карточке)
	for _, r := range vmRefs {
		esx := esxByKey[r.key]
		if esx == "" || lookupHost(esx) != nil {
			continue
		}
		pick, ok := findHostSpr(esx, vmRecByKey[r.key].Vcenter)
		if !ok {
			continue
		}
		if _, dup := hosts[pick.Id]; !dup {
			hosts[pick.Id] = &itHost{spr: pick, cluster: firstNonEmpty(pick.Cluster, vmRecByKey[r.key].Cluster)}
			if pick.ClusterId != "" {
				clusters[clKey{pick.Vcenter, pick.ClusterId}] = firstNonEmpty(pick.Cluster, pick.ClusterId)
			}
		}
		indexHosts()
	}
	loadClusterMembers() // кластеры вновь найденных хостов
	indexHosts()
	vmHost := map[string]*itHost{} // ключ строки ВМ -> хост
	for _, r := range vmRefs {
		if h := lookupHost(esxByKey[r.key]); h != nil {
			h.vms = append(h.vms, firstNonEmpty(r.spr.Name, r.spr.NameFQDN))
			vmHost[r.key] = h
		}
	}
	hostIDs := make([]int64, 0, len(hosts))
	for id := range hosts {
		hostIDs = append(hostIDs, id)
	}
	sort.Slice(hostIDs, func(i, j int) bool { return hostIDs[i] < hostIDs[j] })
	clusterSeen := map[string]bool{}
	for _, id := range hostIDs {
		h := hosts[id]
		out.scene.Stats.Hosts++
		if h.cluster != "" {
			clusterSeen[h.cluster] = true
		}
		sub := itHostRoleLabel(h)
		if h.cluster != "" {
			sub += " · кластер " + h.cluster
		}
		addServerItem(h.spr, "host", sub, itHostRows(h))
		listEquip[len(listEquip)-1].Cluster = h.cluster
		listEquip[len(listEquip)-1].Role = itHostRole(h)
	}
	out.scene.Stats.Clusters = len(clusterSeen)

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
	// siteByCluster - площадка, уже занятая хостами того же кластера (для хостов без шкафа)
	siteByCluster := map[string]int64{}
	placeServerOnAgg := func(agg *rackAgg, hw models.ServerHardware, spr models.ServerSprRecords) {
		if agg == nil {
			return
		}
		rows, links, name := itSysServerCard(hw, spr)
		role := ""
		if h := hosts[spr.Id]; h != nil && spr.Id > 0 {
			rows = append(itHostRows(h), rows...)
			role = itHostRole(h)
		}
		key := itSysServerKey(hw, spr)
		power := rackViewPower(hw.PowerState)
		if isOutside(agg) {
			vd := strings.TrimSpace(hw.Vendor + " " + firstNonEmpty(hw.Model, hw.ShortModel))
			agg.shelf = append(agg.shelf, dcShelfItem{Kind: "server", Id: hw.Id, Name: name, Vendor: vd, U: hw.SizeUnit, Power: power,
				Blades: []dcShelfBlade{}, Key: key, Role: role, Rows: rows, Links: links})
		} else {
			sizeU, guess := hw.SizeUnit, false
			if sizeU <= 0 && hw.StartUnitPos > 0 {
				sizeU, guess = rackGuessSizeU(hw.Model), true
			}
			agg.eq = append(agg.eq, rackViewItem{
				Key: key, Role: role, Kind: "server", Id: hw.Id, Name: name, Vendor: hw.Vendor, Model: hw.Model,
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
	for hwID, hw := range hwByID {
		spr := sprByHW[hwID]
		if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
			chassisBlades[hw.BladeSystemId] = append(chassisBlades[hw.BladeSystemId], bladeRef{hw: hw, spr: spr})
			continue
		}
		agg := ensureRack(hw.RackCabinetId)
		if agg == nil {
			// хост/сервер без шкафа: на стеллаж площадки, где уже есть оборудование того же кластера
			if h := hosts[spr.Id]; h != nil {
				if sid := siteByCluster[h.cluster]; sid > 0 {
					agg = outsideRack(sid)
				}
			}
		}
		if agg == nil {
			continue
		}
		if hw.RackCabinetId > 0 {
			if h := hosts[spr.Id]; h != nil && h.cluster != "" && agg.rc.DatacenterSiteId > 0 {
				siteByCluster[h.cluster] = agg.rc.DatacenterSiteId
			}
		}
		placeServerOnAgg(agg, hw, spr)
	}
	// второй проход: хосты без шкафа, у которых площадка кластера стала известна только после первого прохода
	for hwID, hw := range hwByID {
		spr := sprByHW[hwID]
		if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
			continue
		}
		if hw.RackCabinetId > 0 {
			continue
		}
		if i, ok := listByKey[fmt.Sprintf("server:%d", spr.Id)]; ok && listEquip[i].RackId != 0 {
			continue
		}
		h := hosts[spr.Id]
		if h == nil || h.cluster == "" || siteByCluster[h.cluster] == 0 {
			continue
		}
		placeServerOnAgg(outsideRack(siteByCluster[h.cluster]), hw, spr)
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
				anyHost, anySys, anyCluster := false, false, false
				for _, b := range blades {
					bn := firstNonEmpty(b.spr.Name, b.hw.SerialNum, fmt.Sprintf("blade-%d", b.hw.Id))
					bdesc := bn
					if h := hosts[b.spr.Id]; h != nil && b.spr.Id > 0 {
						switch itHostRole(h) {
						case "host":
							anyHost = true
							bdesc += fmt.Sprintf(" (хост, ВМ ИТ-системы: %d)", len(h.vms))
						case "cluster":
							anyCluster = true
							bdesc += " (хост кластера " + h.cluster + ")"
						default:
							anySys = true
						}
					} else {
						anySys = true
					}
					rows = rackViewKVAdd(rows, fmt.Sprintf("Блейд, слот %d", b.hw.BladeNum), bdesc)
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
						listEquip[i].Subtitle += fmt.Sprintf(" · блейд, слот %d, корзина %s", b.hw.BladeNum, name)
					}
				}
				chRole := ""
				switch {
				case anyHost:
					chRole = "host"
				case anyCluster && !anySys:
					chRole = "cluster"
				}
				vd := strings.TrimSpace(ch.Vendor + " " + firstNonEmpty(ch.Model, ch.Type))
				if isOutside(agg) {
					agg.shelf = append(agg.shelf, dcShelfItem{
						Kind: "chassis", Id: ch.Id, Name: name, Vendor: firstNonEmpty(vd, name), U: ch.SizeUnit,
						Power: rackViewPower(ch.PowerState), Slots: slots, Blades: sbl, Key: ckey, Role: chRole, Rows: rows, Links: links,
					})
					continue
				}
				sizeU := ch.SizeUnit
				if sizeU <= 0 && ch.EndUnitPos >= ch.StartUnitPos && ch.StartUnitPos > 0 {
					sizeU = ch.EndUnitPos - ch.StartUnitPos + 1
				}
				agg.eq = append(agg.eq, rackViewItem{
					Key: ckey, Role: chRole, Kind: "chassis", Id: ch.Id, Name: name, Vendor: ch.Vendor, Model: firstNonEmpty(ch.Model, ch.Type),
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
	// ВМ - в списке/на сцене привязываются к хосту, на котором размещены сейчас
	// (тот же смысл, что «унаследовано от хоста» в resolveVMPlacementText).
	for _, r := range vmRefs {
		i, ok := listByKey[r.key]
		if !ok {
			continue
		}
		vr := vmRecByKey[r.key]
		li := &listEquip[i]
		var vrows []rackViewKV
		h := vmHost[r.key]
		if h != nil {
			hn := firstNonEmpty(h.spr.Name, h.spr.NameFQDN)
			vrows = rackViewKVAdd(vrows, "Хост размещения", hn)
			vrows = rackViewKVAdd(vrows, "Кластер", firstNonEmpty(h.cluster, vr.Cluster))
			li.Cluster = firstNonEmpty(h.cluster, vr.Cluster)
			li.Subtitle = "Виртуальная машина · хост " + hn
			hk := fmt.Sprintf("server:%d", h.spr.Id)
			if hi, ok2 := listByKey[hk]; ok2 {
				host := &listEquip[hi]
				if host.EqKey == "" {
					host.EqKey = host.Key // хост без шкафа: всё равно есть ключ для подсветки списка
				}
				li.EqKey = host.EqKey
				li.RackId, li.RackName, li.SiteId = host.RackId, host.RackName, host.SiteId
				if host.RackId == 0 {
					li.Subtitle += " (размещение хоста в шкафу не указано)"
					vrows = rackViewKVAdd(vrows, "Размещение", "унаследовано от хоста, шкаф хоста не указан")
				} else {
					place := strings.TrimSpace(firstNonEmpty(siteName[host.SiteId], "") + " / " + host.RackName)
					place = strings.TrimPrefix(strings.TrimSpace(place), "/")
					place = strings.TrimSpace(place)
					if place != "" {
						vrows = rackViewKVAdd(vrows, "Размещение", place+" (унаследовано от хоста)")
					}
				}
			} else {
				li.EqKey = hk
			}
		} else {
			esx := firstNonEmpty(esxByKey[r.key], "не определён")
			vrows = rackViewKVAdd(vrows, "Хост размещения", esx)
			vrows = rackViewKVAdd(vrows, "Кластер", vr.Cluster)
			if esxByKey[r.key] != "" {
				li.Subtitle = "Виртуальная машина · хост \"" + esx + "\" (не найден в справочнике серверов)"
			} else {
				li.Subtitle = "Виртуальная машина · хост не определён"
			}
		}
		li.Rows = append(vrows, li.Rows...)
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

	siteOrder := make([]int64, 0, len(siteIDs))
	for sid := range siteIDs {
		siteOrder = append(siteOrder, sid)
	}
	sort.SliceStable(siteOrder, func(i, j int) bool {
		a, b := siteName[siteOrder[i]], siteName[siteOrder[j]]
		if a != b {
			return naturalLess(a, b)
		}
		return siteOrder[i] < siteOrder[j]
	})
	for _, sid := range siteOrder {
		name := firstNonEmpty(siteName[sid], fmt.Sprintf("площадка-%d", sid))
		sc := &dcViewData{Racks: []dcViewRack{}}
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
			r := dcViewRack{Id: rid, Name: a.rc.Name, UHeight: u, Eq: a.eq, SiteId: sid}
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
		out.data.Racks = append(out.data.Racks, sc.Racks...)
		out.data.PowerAvgW += sc.PowerAvgW
		out.data.PowerPeakW += sc.PowerPeakW
		out.scene.Sites = append(out.scene.Sites, itSysSiteInfo{Id: sid, Name: name, Racks: len(sc.Racks), Equip: equip})
		out.scene.Stats.Racks += len(sc.Racks)
	}
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

// itSysDirectVMLimit - сколько ВМ за открытие страницы можно доопределять прямым
// запросом к vCenter (grpcclient.GetVMByKeyDirect медленный; остальные ВМ
// определяются живым списком кластера LoadVMList и запасным Vm_EsxHost).
const itSysDirectVMLimit = 80

// itSysSprCols - загружаемые колонки ServerSprRecords для режима «Оборудование ИТ-системы».
var itSysSprCols = []string{"Id", "Name", "NameFQDN", "Type", "VMSprId", "HWSprId", "IPAddress", "OS", "Role", "Active",
	"Vcenter", "Datacenter", "Cluster", "ClusterId"}

func kindOrderITSys(k string) int {
	switch k {
	case "server":
		return 0
	case "host":
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
