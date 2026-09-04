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
		},
	}

	hwNameByID := map[int64]string{}
	var sprRows []models.ServerSprRecords
	if _, err := o.QueryTable(new(models.ServerSprRecords)).
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
		hasPlacement := strings.TrimSpace(ch.DatacenterRoom) != "" || strings.TrimSpace(ch.RackNum) != ""
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

		r := ensureRack(ch.DatacenterRoom, ch.RackNum)
		sizeU := ch.SizeUnit
		if sizeU <= 0 {
			sizeU = 10
		}
		startU := ch.StartUnitPos
		if startU <= 0 {
			startU = parseStartUnit(ch.StartUnit)
		}
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
		if startU <= 0 {
			startU = parseStartUnit(hw.StartUnit)
		}
		r := ensureRack(hw.DatacenterRoom, hw.RackNum)
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
	storageItems := make([]payload.Storage, 0, len(storageRows))
	for _, st := range storageRows {
		diskMax := st.Size
		if diskMax <= 0 && st.Used > 0 {
			diskMax = st.Used
		}
		storageItems = append(storageItems, payload.Storage{
			ShortName:   st.ShortName,
			Vendor:      st.Vendor,
			Model:       st.Model,
			DiskCount:   st.Used,
			DiskMax:     diskMax,
			TotalSizeTB: st.TotalSizeRAW,
			UsedPercent: st.UsedPercent,
			Room:        st.DatacenterRoom,
			RackNum:     st.RackNum,
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
	vmCount, _ := o.QueryTable(new(models.ServerSprRecords)).
		Filter("Stat", 1).Filter("Active", 1).Filter("Type", 1).Count()
	hsCount, _ := o.QueryTable(new(models.ServerSprRecords)).
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
		StorageOnline:     onlineStorage,
		StorageOffline:    offlineStorage,
		AvgUsedPercent:    avgUsed,
	}

	return data, nil
}

func parseStartUnit(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ';' || r == '-'
	})
	if len(parts) == 0 {
		return 0
	}
	var n int
	fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &n)
	return n
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
