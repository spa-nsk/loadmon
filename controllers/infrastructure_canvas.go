package controllers

import (
	"fmt"
	"sort"
	"strings"

	models "loadmon/models"
	"loadmon/infracanvas/payload"

	"github.com/beego/beego/v2/client/orm"
)

func buildInfrastructureCanvasData() (payload.CanvasData, error) {
	o := orm.NewOrm()
	data := payload.CanvasData{
		Pages: []payload.PageMeta{
			{ID: "hardware", Title: "Аппаратная часть"},
			{ID: "virtualization", Title: "Виртуализация"},
			{ID: "monitoring", Title: "Мониторинг"},
			{ID: "clusterMap", Title: "Кластеры vSphere"},
		},
	}

	hwNameByID := map[int64]string{}
	var sprRows []models.ServerSprRecords
	if _, err := models.ServerSprQS(o).
		Filter("Stat", 1).Filter("Active", 1).All(&sprRows); err != nil {
		return data, err
	}
	for _, s := range sprRows {
		if s.HWSprId > 0 && strings.TrimSpace(s.Name) != "" {
			hwNameByID[s.HWSprId] = s.Name
		}
	}

	// Активные blade/rack серверы из server_hardware (OneView/CMDB).
	var hwRows []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).
		Filter("Active", 1).All(&hwRows); err != nil {
		return data, err
	}

	bladesByChassis := map[int64][]models.ServerHardware{}
	var rackServers []models.ServerHardware
	for _, hw := range hwRows {
		if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
			bladesByChassis[hw.BladeSystemId] = append(bladesByChassis[hw.BladeSystemId], hw)
			continue
		}
		if hw.FormFactor == 1 {
			rackServers = append(rackServers, hw)
		}
	}

	// Блэйд-корзины.
	var chassisRows []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).All(&chassisRows); err != nil {
		return data, err
	}

	// ИЗМЕНЕНО (2026-09-08): models.BladeSystems лишился текстовых
	// DatacenterRoom/RackNum - расположение корзины теперь определяется
	// только через RackCabinetId (FK на models.RackCabinet), поэтому для
	// группировки по стойке/серверной здесь строится собственный лookup
	// RackCabinetId -> (имя серверной, имя шкафа). models.ServerHardware
	// лишился тех же текстовых полей позже (2026-10-01) - тот же lookup
	// используется и для него (см. ниже), StorageSystems размещается через
	// отдельную таблицу-связку models.StorageSystemPlacement.
	type cabinetPlacement struct {
		room, rack string
	}
	cabinetById := map[int64]cabinetPlacement{}
	{
		cabinetIds := make([]int64, 0, len(chassisRows)+len(rackServers))
		for _, ch := range chassisRows {
			if ch.RackCabinetId > 0 {
				cabinetIds = append(cabinetIds, ch.RackCabinetId)
			}
		}
		// ДОБАВЛЕНО (2026-09-09, по запросу пользователя "реализуй
		// аналогичную обработку... и других где отображается
		// ServerHardware"): rack-серверы (не блейд) раньше группировались
		// на канвасе ТОЛЬКО по легаси-текстовым hw.DatacenterRoom/hw.RackNum
		// (см. fallback ниже), хотя у ServerHardware уже есть актуальный FK
		// RackCabinetId - используем тот же lookup RackCabinetId->(ЦОД,
		// Шкаф), что и для блейд-корзин выше, чтобы канвас не расходился с
		// /sprsrv/hwconflist/hwconfedit, которые давно на новой схеме.
		for _, hw := range rackServers {
			if hw.RackCabinetId > 0 {
				cabinetIds = append(cabinetIds, hw.RackCabinetId)
			}
		}
		if len(cabinetIds) > 0 {
			var cabinets []models.RackCabinet
			if _, err := o.QueryTable(new(models.RackCabinet)).Filter("Id__in", cabinetIds).All(&cabinets); err == nil {
				siteIds := make([]int64, 0, len(cabinets))
				for _, c := range cabinets {
					if c.DatacenterSiteId > 0 {
						siteIds = append(siteIds, c.DatacenterSiteId)
					}
				}
				siteNameById := map[int64]string{}
				if len(siteIds) > 0 {
					var sites []models.DatacenterSite
					if _, err := o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", siteIds).All(&sites); err == nil {
						for _, s := range sites {
							siteNameById[s.Id] = s.Name
						}
					}
				}
				for _, c := range cabinets {
					cabinetById[c.Id] = cabinetPlacement{room: siteNameById[c.DatacenterSiteId], rack: c.Name}
				}
			}
		}
	}

	type rackKey struct {
		room, rack string
	}
	racks := map[rackKey]*payload.Rack{}

	ensureRack := func(room, rack string) *payload.Rack {
		if strings.TrimSpace(room) == "" {
			room = "Без серверной"
		}
		if strings.TrimSpace(rack) == "" {
			rack = "—"
		}
		k := rackKey{room: room, rack: rack}
		if r, ok := racks[k]; ok {
			return r
		}
		r := &payload.Rack{
			Room:    room,
			RackNum: rack,
			Label:   fmt.Sprintf("%s / стойка %s", room, rack),
		}
		racks[k] = r
		return r
	}

	activeChassis := 0
	activeBlades := 0
	for _, ch := range chassisRows {
		blades := bladesByChassis[ch.Id]
		placement := cabinetById[ch.RackCabinetId]
		hasPlacement := ch.RackCabinetId > 0 && (strings.TrimSpace(placement.room) != "" || strings.TrimSpace(placement.rack) != "")
		if len(blades) == 0 && !hasPlacement {
			continue
		}
		activeChassis++

		slotCount := ch.Size
		if slotCount <= 0 {
			slotCount = 16
		}
		occupied := map[int]models.ServerHardware{}
		for _, b := range blades {
			if b.BladeNum > 0 {
				occupied[b.BladeNum] = b
			}
			activeBlades++
		}
		slots := make([]payload.BladeSlot, 0, slotCount)
		for n := 1; n <= slotCount; n++ {
			slot := payload.BladeSlot{Num: n}
			if hw, ok := occupied[n]; ok {
				slot.Occupied = true
				slot.Model = hw.Model
				slot.PowerOn = hw.PowerState == 1
				if name := hwNameByID[hw.Id]; name != "" {
					slot.ServerName = name
				} else if hw.SerialNum != "" {
					slot.ServerName = hw.SerialNum
				} else {
					slot.ServerName = fmt.Sprintf("blade-%d", n)
				}
			}
			slots = append(slots, slot)
		}

		r := ensureRack(placement.room, placement.rack)
		sizeU := ch.SizeUnit
		if sizeU <= 0 {
			sizeU = 10
		}
		// ИЗМЕНЕНО (2026-09-08): models.BladeSystems лишился текстового
		// StartUnit - StartUnitPos (число) остался единственным источником.
		startU := ch.StartUnitPos
		r.BladeChassis = append(r.BladeChassis, payload.BladeChassis{
			ID:        ch.Id,
			Name:      firstNonEmpty(ch.Name, ch.SerialNum, fmt.Sprintf("chassis-%d", ch.Id)),
			Model:     ch.Model,
			SerialNum: ch.SerialNum,
			SlotCount: slotCount,
			StartUnit: startU,
			SizeUnit:  sizeU,
			Slots:     slots,
		})
	}

	activeRackSrv := 0
	for _, hw := range rackServers {
		activeRackSrv++
		name := hwNameByID[hw.Id]
		if name == "" {
			name = firstNonEmpty(hw.SerialNum, hw.Model, fmt.Sprintf("srv-%d", hw.Id))
		}
		sizeU := hw.SizeUnit
		if sizeU <= 0 {
			sizeU = 2
		}
		startU := hw.StartUnitPos
		// Размещение - в приоритете новая схема (RackCabinetId->RackCabinet/
		// DatacenterSite, тот же lookup cabinetById, что и у блейд-корзин
		// выше); если RackCabinetId не проставлен (старые данные, ещё не
		// мигрированные через /sprsrv/hwconfedit) - легаси текстовые поля
		// DatacenterRoom/RackNum удалены из ServerHardware (2026-10-01,
		// атавизм), поэтому и "серверная", и "стойка" остаются пустыми -
		// такой сервер попадёт в общий бакет "без привязки к шкафу", но не
		// пропадёт с канваса молча.
		placement := cabinetById[hw.RackCabinetId]
		room, rack := "", ""
		if hw.RackCabinetId > 0 && (strings.TrimSpace(placement.room) != "" || strings.TrimSpace(placement.rack) != "") {
			room, rack = placement.room, placement.rack
		}
		r := ensureRack(room, rack)
		r.RackServers = append(r.RackServers, payload.RackServer{
			Name:      name,
			Model:     hw.Model,
			SerialNum: hw.SerialNum,
			StartUnit: startU,
			SizeUnit:  sizeU,
			PowerOn:   hw.PowerState == 1,
		})
	}

	// Активные СХД (Status=1).
	var storageRows []models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).
		Filter("Status", 1).All(&storageRows); err != nil {
		return data, err
	}
	sort.Slice(storageRows, func(i, j int) bool {
		return storageRows[i].ShortName < storageRows[j].ShortName
	})

	// ИЗМЕНЕНО (2026-09-09, по запросу пользователя "СХД может размещаться в
	// нескольких шкафах и схема со ссылкой StorageSystems.RackCabinetId не
	// позволяет реализовать учет размещения"): расположение СХД теперь
	// определяется через отдельную таблицу связи models.
	// StorageSystemPlacement (см. её комментарий в models_add.go) - ОДНА
	// СХД может иметь НЕСКОЛЬКО записей размещения. Этот легаси-канвас
	// (payload.Storage) - плоский список сбоку от стоек (в отличие от
	// /admin/hwinfra, где СХД физически повторяется В КАЖДОМ шкафу на
	// canvas, см. controllers/adminhwinfra.go), поэтому здесь все
	// размещения одной СХД просто склеиваются в Room/RackNum через "; " -
	// довычисляются недостающие записи cabinetById для шкафов, на которые
	// ссылаются размещения СХД (RackCabinetId у блейд-корзин и у размещений
	// СХД может пересекаться, но необязательно совпадать полностью, поэтому
	// нельзя просто переиспользовать карту как есть без дозагрузки).
	storageIds := make([]int64, 0, len(storageRows))
	for _, st := range storageRows {
		storageIds = append(storageIds, st.Id)
	}
	placementsByStorage := map[int64][]models.StorageSystemPlacement{}
	if len(storageIds) > 0 {
		var placements []models.StorageSystemPlacement
		if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId__in", storageIds).OrderBy("-IsPrimary", "Id").All(&placements); err == nil {
			for _, p := range placements {
				placementsByStorage[p.StorageSystemsId] = append(placementsByStorage[p.StorageSystemsId], p)
			}
		}
	}
	{
		missingCabinetIds := make([]int64, 0)
		for _, placements := range placementsByStorage {
			for _, p := range placements {
				if p.RackCabinetId > 0 {
					if _, ok := cabinetById[p.RackCabinetId]; !ok {
						missingCabinetIds = append(missingCabinetIds, p.RackCabinetId)
					}
				}
			}
		}
		if len(missingCabinetIds) > 0 {
			var cabinets []models.RackCabinet
			if _, err := o.QueryTable(new(models.RackCabinet)).Filter("Id__in", missingCabinetIds).All(&cabinets); err == nil {
				siteIds := make([]int64, 0, len(cabinets))
				for _, c := range cabinets {
					if c.DatacenterSiteId > 0 {
						siteIds = append(siteIds, c.DatacenterSiteId)
					}
				}
				siteNameById := map[int64]string{}
				if len(siteIds) > 0 {
					var sites []models.DatacenterSite
					if _, err := o.QueryTable(new(models.DatacenterSite)).Filter("Id__in", siteIds).All(&sites); err == nil {
						for _, s := range sites {
							siteNameById[s.Id] = s.Name
						}
					}
				}
				for _, c := range cabinets {
					cabinetById[c.Id] = cabinetPlacement{room: siteNameById[c.DatacenterSiteId], rack: c.Name}
				}
			}
		}
	}

	storageItems := make([]payload.Storage, 0, len(storageRows))
	for _, st := range storageRows {
		diskMax := st.Size
		if diskMax <= 0 && st.Used > 0 {
			diskMax = st.Used
		}
		var rooms, racks []string
		for _, p := range placementsByStorage[st.Id] {
			placement := cabinetById[p.RackCabinetId]
			if placement.room != "" {
				rooms = append(rooms, placement.room)
			}
			if placement.rack != "" {
				racks = append(racks, placement.rack)
			}
		}
		storageItems = append(storageItems, payload.Storage{
			ShortName:   st.ShortName,
			Vendor:      st.Vendor,
			Model:       st.Model,
			DiskCount:   st.Used,
			DiskMax:     diskMax,
			TotalSizeTB: st.TotalSizeRAW,
			UsedPercent: st.UsedPercent,
			Room:        strings.Join(rooms, "; "),
			RackNum:     strings.Join(racks, "; "),
			PowerOn:     st.Power == 1,
		})
	}

	rackList := make([]payload.Rack, 0, len(racks))
	for _, r := range racks {
		sort.Slice(r.BladeChassis, func(i, j int) bool {
			return r.BladeChassis[i].Name < r.BladeChassis[j].Name
		})
		sort.Slice(r.RackServers, func(i, j int) bool {
			return r.RackServers[i].StartUnit < r.RackServers[j].StartUnit
		})
		rackList = append(rackList, *r)
	}
	sort.Slice(rackList, func(i, j int) bool {
		if rackList[i].Room == rackList[j].Room {
			return rackList[i].RackNum < rackList[j].RackNum
		}
		return rackList[i].Room < rackList[j].Room
	})

	data.Hardware = payload.HardwarePage{
		Racks:   rackList,
		Storage: storageItems,
		Stats: payload.HWStats{
			BladeChassis: activeChassis,
			BladeServers: activeBlades,
			RackServers:  activeRackSrv,
			StorageUnits: len(storageItems),
		},
	}

	// Виртуализация: активные VM/гипервизоры и кластеры.
	vmCount, _ := models.ServerSprQS(o).
		Filter("Stat", 1).Filter("Active", 1).Filter("Type", 1).Count()
	hsCount, _ := models.ServerSprQS(o).
		Filter("Stat", 1).Filter("Active", 1).Filter("Type", 2).Count()
	clusterCount, _ := o.QueryTable(new(models.ClusterSprRecords)).
		Filter("Stat", 1).Count()

	clusterNames := map[string]int{}
	for _, s := range sprRows {
		if s.Type != 1 || strings.TrimSpace(s.Cluster) == "" {
			continue
		}
		clusterNames[s.Cluster]++
	}
	topClusters := make([]payload.Named, 0, len(clusterNames))
	for name, cnt := range clusterNames {
		topClusters = append(topClusters, payload.Named{Name: name, Count: int(cnt)})
	}
	sort.Slice(topClusters, func(i, j int) bool {
		if topClusters[i].Count == topClusters[j].Count {
			return topClusters[i].Name < topClusters[j].Name
		}
		return topClusters[i].Count > topClusters[j].Count
	})
	if len(topClusters) > 8 {
		topClusters = topClusters[:8]
	}

	data.Virtualization = payload.VirtualPage{
		VMCount:         int(vmCount),
		HypervisorCount: int(hsCount),
		ClusterCount:    int(clusterCount),
		TopClusters:     topClusters,
	}

	// Мониторинг: Zabbix и СХД.
	zabbixCount, _ := o.QueryTable(new(models.ZabbixHost)).Count()
	linkedZabbix, _ := o.QueryTable(new(models.ZabbixHost)).Filter("ServerId__gt", 0).Count()
	onlineStorage := 0
	offlineStorage := 0
	var usedSum float64
	for _, st := range storageRows {
		if st.Power == 1 {
			onlineStorage++
		} else {
			offlineStorage++
		}
		usedSum += st.UsedPercent
	}
	avgUsed := 0.0
	if len(storageRows) > 0 {
		avgUsed = usedSum / float64(len(storageRows))
	}
	data.Monitoring = payload.MonitoringPage{
		ZabbixHostCount: int(zabbixCount),
		LinkedServers:   int(linkedZabbix),
		StorageOnline:   onlineStorage,
		StorageOffline:  offlineStorage,
		AvgUsedPercent:  avgUsed,
	}

	data.ClusterMap = buildClusterMapPage(o, sprRows)

	return data, nil
}

func buildClusterMapPage(o orm.Ormer, sprRows []models.ServerSprRecords) payload.ClusterMapPage {
	var clusterRows []models.ClusterSprRecords
	_, _ = o.QueryTable(new(models.ClusterSprRecords)).Filter("Stat", 1).All(&clusterRows)

	type clusterKey struct {
		vc, dc, name string
	}
	hostCount := map[clusterKey]int{}
	vmCount := map[clusterKey]int{}
	typeWorkByKey := map[clusterKey]int{}

	for _, s := range sprRows {
		if strings.TrimSpace(s.Cluster) == "" {
			continue
		}
		k := clusterKey{
			vc:   strings.TrimSpace(s.Vcenter),
			dc:   strings.TrimSpace(s.Datacenter),
			name: strings.TrimSpace(s.Cluster),
		}
		switch s.Type {
		case 1:
			vmCount[k]++
		case 2:
			hostCount[k]++
		}
	}

	vcMap := map[string]map[string][]payload.ClusterNode{}
	dcSet := map[string]struct{}{}
	for _, cl := range clusterRows {
		k := clusterKey{
			vc:   strings.TrimSpace(cl.Vcenter),
			dc:   strings.TrimSpace(cl.Datacenter),
			name: strings.TrimSpace(cl.Name),
		}
		if k.name == "" {
			continue
		}
		if k.vc == "" {
			k.vc = "—"
		}
		if k.dc == "" {
			k.dc = "—"
		}
		typeWorkByKey[k] = cl.TypeWork
		if _, ok := vcMap[k.vc]; !ok {
			vcMap[k.vc] = map[string][]payload.ClusterNode{}
		}
		node := payload.ClusterNode{
			Name:      k.name,
			VMCount:   vmCount[k],
			HostCount: hostCount[k],
			TypeWork:  cl.TypeWork,
		}
		vcMap[k.vc][k.dc] = append(vcMap[k.vc][k.dc], node)
		dcSet[k.vc+"/"+k.dc] = struct{}{}
	}

	// Кластеры из sprRows, которых нет в справочнике.
	seen := map[clusterKey]bool{}
	for _, cl := range clusterRows {
		seen[clusterKey{cl.Vcenter, cl.Datacenter, cl.Name}] = true
	}
	for k, vms := range vmCount {
		if seen[k] {
			continue
		}
		vc := k.vc
		if vc == "" {
			vc = "—"
		}
		dc := k.dc
		if dc == "" {
			dc = "—"
		}
		if _, ok := vcMap[vc]; !ok {
			vcMap[vc] = map[string][]payload.ClusterNode{}
		}
		vcMap[vc][dc] = append(vcMap[vc][dc], payload.ClusterNode{
			Name:      k.name,
			VMCount:   vms,
			HostCount: hostCount[k],
			TypeWork:  typeWorkByKey[k],
		})
		dcSet[vc+"/"+dc] = struct{}{}
	}

	vcenters := make([]payload.VcenterGroup, 0, len(vcMap))
	totalClusters := 0
	totalHosts := 0
	totalVMs := 0
	vcNames := make([]string, 0, len(vcMap))
	for name := range vcMap {
		vcNames = append(vcNames, name)
	}
	sort.Strings(vcNames)
	for _, vcName := range vcNames {
		dcMap := vcMap[vcName]
		dcNames := make([]string, 0, len(dcMap))
		for dcName := range dcMap {
			dcNames = append(dcNames, dcName)
		}
		sort.Strings(dcNames)
		dcs := make([]payload.DatacenterGroup, 0, len(dcNames))
		for _, dcName := range dcNames {
			clusters := dcMap[dcName]
			sort.Slice(clusters, func(i, j int) bool {
				return clusters[i].Name < clusters[j].Name
			})
			totalClusters += len(clusters)
			for _, c := range clusters {
				totalHosts += c.HostCount
				totalVMs += c.VMCount
			}
			dcs = append(dcs, payload.DatacenterGroup{Name: dcName, Clusters: clusters})
		}
		vcenters = append(vcenters, payload.VcenterGroup{Name: vcName, Datacenters: dcs})
	}

	return payload.ClusterMapPage{
		Vcenters: vcenters,
		Stats: payload.ClusterStats{
			VcenterCount:    len(vcenters),
			DatacenterCount: len(dcSet),
			ClusterCount:    totalClusters,
			HostCount:       totalHosts,
			VMCount:         totalVMs,
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
