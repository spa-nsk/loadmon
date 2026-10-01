package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	models "loadmon/models"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/client/orm"
)

// ---- JSON-модель для новой canvas-страницы "Аппаратная инфраструктура" ----
//
// В отличие от старого infracanvas (controllers/infrastructure_canvas.go),
// который группирует стойки по текстовым полям DatacenterRoom/RackNum, эта
// страница строит иерархию по актуальным справочникам
// DatacenterSite -> RackCabinet: FK RackCabinetId у ServerHardware/
// BladeSystems (один шкаф на запись), и через отдельную таблицу связи
// models.StorageSystemPlacement у StorageSystems (2026-09-09 - одна СХД
// может занимать НЕСКОЛЬКО шкафов одновременно, см. её комментарий в
// models_add.go) - наполнение ЦОД/Шкафов идёт из Итиллиум (см.
// kontext/03-data-model.md).

// HWInfraDisk - упрощённое представление "заполнения дисками" СХД для
// визуализации (без выгрузки каждого физического диска - это агрегат
// Size/Used из справочника СХД, отдельные тома/LUN - другая сущность).
type HWInfraStorage struct {
	Name        string  `json:"name"`
	Vendor      string  `json:"vendor"`
	Model       string  `json:"model"`
	DiskUsed    int     `json:"diskUsed"`
	DiskMax     int     `json:"diskMax"`
	TotalSizeTB float64 `json:"totalSizeTb"`
	UsedPercent float64 `json:"usedPercent"`
	PowerOn     bool    `json:"powerOn"`
	StartUnit   int     `json:"startUnit"`
	SizeUnit    int     `json:"sizeUnit"`
}

type HWInfraServer struct {
	Name       string `json:"name"`
	Vendor     string `json:"vendor"`
	Model      string `json:"model"`
	SerialNum  string `json:"serialNum"`
	StartUnit  int    `json:"startUnit"`
	SizeUnit   int    `json:"sizeUnit"`
	PowerOn    bool   `json:"powerOn"`
	Hypervisor bool   `json:"hypervisor"`
	CPUModel   string `json:"cpuModel"`
	CPUCount   int    `json:"cpuCount"`
	CorePerCPU int    `json:"corePerCpu"`
	MemoryMb   int    `json:"memoryMb"`
}

type HWInfraSlot struct {
	Num        int    `json:"num"`
	Occupied   bool   `json:"occupied"`
	ServerName string `json:"serverName,omitempty"`
	Model      string `json:"model,omitempty"`
	PowerOn    bool   `json:"powerOn,omitempty"`
	Hypervisor bool   `json:"hypervisor,omitempty"`
}

type HWInfraChassis struct {
	Name      string        `json:"name"`
	Model     string        `json:"model"`
	SerialNum string        `json:"serialNum"`
	StartUnit int           `json:"startUnit"`
	SizeUnit  int           `json:"sizeUnit"`
	SlotCount int           `json:"slotCount"`
	Used      int           `json:"used"`
	Slots     []HWInfraSlot `json:"slots"`
}

type HWInfraRack struct {
	Name       string           `json:"name"`
	AltName    string           `json:"altName,omitempty"`
	Model      string           `json:"model,omitempty"`
	UHeight    int              `json:"uHeight"`
	Annotation string           `json:"annotation,omitempty"`
	Storage    []HWInfraStorage `json:"storage"`
	Servers    []HWInfraServer  `json:"servers"`
	Chassis    []HWInfraChassis `json:"chassis"`
}

type HWInfraDatacenter struct {
	Name         string        `json:"name"`
	AltName      string        `json:"altName,omitempty"`
	Address      string        `json:"address,omitempty"` // из DatacenterSite.Annotation
	RackCount    int           `json:"rackCount"`
	ChassisCount int           `json:"chassisCount"`
	ServerCount  int           `json:"serverCount"` // rack + blade суммарно
	HVCount      int           `json:"hvCount"`     // из них гипервизоры (ESXi)
	StorageCount int           `json:"storageCount"`
	Racks        []HWInfraRack `json:"racks"`
}

type HWInfraData struct {
	Datacenters []HWInfraDatacenter `json:"datacenters"`
}

func buildHWInfraData() (HWInfraData, error) {
	o := orm.NewOrm()
	var data HWInfraData

	var sites []models.DatacenterSite
	if _, err := o.QueryTable(new(models.DatacenterSite)).All(&sites); err != nil {
		return data, err
	}
	var cabinets []models.RackCabinet
	if _, err := o.QueryTable(new(models.RackCabinet)).All(&cabinets); err != nil {
		return data, err
	}
	var hwRows []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).Filter("Active", 1).All(&hwRows); err != nil {
		return data, err
	}
	var chassisRows []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).All(&chassisRows); err != nil {
		return data, err
	}
	var storageRows []models.StorageSystems
	if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Status", 1).All(&storageRows); err != nil {
		return data, err
	}
	// Размещение СХД (2026-09-09) - отдельная таблица связи "многие-ко-
	// многим" models.StorageSystemPlacement (одна СХД может занимать
	// несколько шкафов), а не поле прямо на StorageSystems - см. её
	// комментарий в models_add.go. Грузится списком под все активные СХД
	// сразу, группировка по StorageSystemsId - ниже.
	storageIds := make([]int64, 0, len(storageRows))
	for _, st := range storageRows {
		storageIds = append(storageIds, st.Id)
	}
	placementsByStorage := map[int64][]models.StorageSystemPlacement{}
	if len(storageIds) > 0 {
		var placements []models.StorageSystemPlacement
		if _, err := o.QueryTable(new(models.StorageSystemPlacement)).Filter("StorageSystemsId__in", storageIds).All(&placements); err == nil {
			for _, p := range placements {
				placementsByStorage[p.StorageSystemsId] = append(placementsByStorage[p.StorageSystemsId], p)
			}
		}
	}
	var sprRows []models.ServerSprRecords
	if _, err := o.QueryTable(new(models.ServerSprRecords)).
		Filter("Stat", 1).Filter("Active", 1).All(&sprRows); err != nil {
		return data, err
	}

	// HWSprId -> (имя из справочника серверов, признак гипервизора Type=2).
	hwName := map[int64]string{}
	hwIsHV := map[int64]bool{}
	for _, s := range sprRows {
		if s.HWSprId <= 0 {
			continue
		}
		if strings.TrimSpace(s.Name) != "" {
			hwName[s.HWSprId] = s.Name
		}
		if s.Type == 2 {
			hwIsHV[s.HWSprId] = true
		}
	}

	cabinetByID := map[int64]*models.RackCabinet{}
	for i := range cabinets {
		cabinetByID[cabinets[i].Id] = &cabinets[i]
	}
	siteByID := map[int64]*models.DatacenterSite{}
	for i := range sites {
		siteByID[sites[i].Id] = &sites[i]
	}

	type rackAgg struct {
		cabinet *models.RackCabinet
		storage []HWInfraStorage
		servers []HWInfraServer
		chassis []HWInfraChassis
	}
	// siteID(0 = "не определён") -> rackCabinetID(0 = "без шкафа") -> агрегат
	bySite := map[int64]map[int64]*rackAgg{}
	ensureRack := func(siteID, cabID int64) *rackAgg {
		if _, ok := bySite[siteID]; !ok {
			bySite[siteID] = map[int64]*rackAgg{}
		}
		if r, ok := bySite[siteID][cabID]; ok {
			return r
		}
		r := &rackAgg{cabinet: cabinetByID[cabID]}
		bySite[siteID][cabID] = r
		return r
	}
	siteOfCabinet := func(cabID int64) int64 {
		if cabID <= 0 {
			return 0
		}
		if c, ok := cabinetByID[cabID]; ok {
			return c.DatacenterSiteId
		}
		return 0
	}

	bladesByChassis := map[int64][]models.ServerHardware{}
	var rackServers []models.ServerHardware
	for _, hw := range hwRows {
		if hw.FormFactor == 2 && hw.BladeSystemId > 0 {
			bladesByChassis[hw.BladeSystemId] = append(bladesByChassis[hw.BladeSystemId], hw)
			continue
		}
		rackServers = append(rackServers, hw)
	}

	for _, hw := range rackServers {
		name := hwName[hw.Id]
		if name == "" {
			name = firstNonEmpty(hw.SerialNum, hw.Model, fmt.Sprintf("srv-%d", hw.Id))
		}
		sizeU := hw.SizeUnit
		if sizeU <= 0 {
			sizeU = 2
		}
		startU := hw.StartUnitPos
		siteID := siteOfCabinet(hw.RackCabinetId)
		r := ensureRack(siteID, hw.RackCabinetId)
		r.servers = append(r.servers, HWInfraServer{
			Name: name, Vendor: hw.Vendor, Model: hw.Model, SerialNum: hw.SerialNum,
			StartUnit: startU, SizeUnit: sizeU, PowerOn: hw.PowerState == 1,
			Hypervisor: hwIsHV[hw.Id], CPUModel: hw.CPUModel, CPUCount: hw.CPUCount,
			CorePerCPU: hw.CorePerCPU, MemoryMb: hw.MemoryMb,
		})
	}

	for _, ch := range chassisRows {
		blades := bladesByChassis[ch.Id]
		if len(blades) == 0 && ch.RackCabinetId <= 0 {
			continue
		}
		slotCount := ch.Size
		if slotCount <= 0 {
			slotCount = 16
		}
		occupied := map[int]models.ServerHardware{}
		for _, b := range blades {
			if b.BladeNum > 0 {
				occupied[b.BladeNum] = b
			}
		}
		slots := make([]HWInfraSlot, 0, slotCount)
		for n := 1; n <= slotCount; n++ {
			slot := HWInfraSlot{Num: n}
			if hw, ok := occupied[n]; ok {
				slot.Occupied = true
				slot.Model = hw.Model
				slot.PowerOn = hw.PowerState == 1
				slot.Hypervisor = hwIsHV[hw.Id]
				if name := hwName[hw.Id]; name != "" {
					slot.ServerName = name
				} else if hw.SerialNum != "" {
					slot.ServerName = hw.SerialNum
				} else {
					slot.ServerName = fmt.Sprintf("blade-%d", n)
				}
			}
			slots = append(slots, slot)
		}
		sizeU := ch.SizeUnit
		if sizeU <= 0 {
			sizeU = 10
		}
		// ИЗМЕНЕНО (2026-09-08): models.BladeSystems лишился текстового
		// StartUnit - StartUnitPos (число) остался единственным источником.
		startU := ch.StartUnitPos
		siteID := siteOfCabinet(ch.RackCabinetId)
		r := ensureRack(siteID, ch.RackCabinetId)
		r.chassis = append(r.chassis, HWInfraChassis{
			Name:      firstNonEmpty(ch.Name, ch.SerialNum, fmt.Sprintf("chassis-%d", ch.Id)),
			Model:     ch.Model, SerialNum: ch.SerialNum,
			StartUnit: startU, SizeUnit: sizeU, SlotCount: slotCount, Used: len(occupied),
			Slots: slots,
		})
	}

	// СХД: размещение определяется через models.StorageSystemPlacement
	// (2026-09-09, см. комментарий выше и в models_add.go) - ОДНА СХД может
	// иметь НЕСКОЛЬКО записей размещения (несколько шкафов) - в этом случае
	// она добавляется в КАЖДЫЙ соответствующий шкаф на канве отдельным
	// элементом (с одними и теми же Name/Vendor/Model/... - юниты у каждого
	// элемента свои, из своей записи StorageSystemPlacement). Если у СХД
	// нет ни одной записи размещения - она показывается один раз без
	// привязки к шкафу (siteID=0, cabID=0, тот же приём, что раньше был
	// единственным путём).
	for _, st := range storageRows {
		diskMax := st.Size
		if diskMax <= 0 && st.Used > 0 {
			diskMax = st.Used
		}
		baseItem := HWInfraStorage{
			Name: st.ShortName, Vendor: st.Vendor, Model: st.Model,
			DiskUsed: st.Used, DiskMax: diskMax, TotalSizeTB: st.TotalSizeRAW,
			UsedPercent: st.UsedPercent, PowerOn: st.Power == 1,
		}

		placements := placementsByStorage[st.Id]
		if len(placements) == 0 {
			item := baseItem
			item.SizeUnit = 4
			r := ensureRack(0, 0)
			r.storage = append(r.storage, item)
			continue
		}
		for _, p := range placements {
			item := baseItem
			item.StartUnit = p.StartUnitPos
			item.SizeUnit = p.SizeUnit
			if item.SizeUnit <= 0 {
				item.SizeUnit = 4
			}
			siteID := siteOfCabinet(p.RackCabinetId)
			r := ensureRack(siteID, p.RackCabinetId)
			r.storage = append(r.storage, item)
		}
	}

	// Сборка итоговой структуры.
	siteIDs := make([]int64, 0, len(bySite))
	for id := range bySite {
		siteIDs = append(siteIDs, id)
	}
	sort.Slice(siteIDs, func(i, j int) bool { return siteIDs[i] < siteIDs[j] })

	for _, siteID := range siteIDs {
		dc := HWInfraDatacenter{}
		if site, ok := siteByID[siteID]; ok {
			dc.Name = site.Name
			dc.AltName = site.AltName
			dc.Address = site.Annotation
		} else {
			dc.Name = "Без привязки к ЦОД"
		}

		cabIDs := make([]int64, 0, len(bySite[siteID]))
		for id := range bySite[siteID] {
			cabIDs = append(cabIDs, id)
		}
		sort.Slice(cabIDs, func(i, j int) bool { return cabIDs[i] < cabIDs[j] })

		for _, cabID := range cabIDs {
			agg := bySite[siteID][cabID]
			rack := HWInfraRack{}
			if agg.cabinet != nil {
				rack.Name = agg.cabinet.Name
				rack.AltName = agg.cabinet.AltName
				rack.Model = agg.cabinet.Model
				rack.UHeight = int(agg.cabinet.UHeight)
				rack.Annotation = agg.cabinet.Annotation
			} else {
				rack.Name = "Без привязки к шкафу"
			}
			if rack.UHeight <= 0 {
				rack.UHeight = 42
			}
			sort.Slice(agg.servers, func(i, j int) bool { return agg.servers[i].StartUnit < agg.servers[j].StartUnit })
			sort.Slice(agg.chassis, func(i, j int) bool { return agg.chassis[i].Name < agg.chassis[j].Name })
			sort.Slice(agg.storage, func(i, j int) bool { return agg.storage[i].Name < agg.storage[j].Name })
			rack.Servers = agg.servers
			rack.Chassis = agg.chassis
			rack.Storage = agg.storage

			dc.RackCount++
			dc.ChassisCount += len(rack.Chassis)
			dc.StorageCount += len(rack.Storage)
			dc.ServerCount += len(rack.Servers)
			for _, c := range rack.Chassis {
				dc.ServerCount += c.Used
			}
			for _, s := range rack.Servers {
				if s.Hypervisor {
					dc.HVCount++
				}
			}
			for _, c := range rack.Chassis {
				for _, sl := range c.Slots {
					if sl.Occupied && sl.Hypervisor {
						dc.HVCount++
					}
				}
			}
			dc.Racks = append(dc.Racks, rack)
		}
		data.Datacenters = append(data.Datacenters, dc)
	}

	return data, nil
}

// AdminHwInfra отображает новую canvas-страницу "Аппаратная инфраструктура":
// обзор ЦОД -> шкафы -> СХД/серверы/блейд-корзины с занятыми слотами.
// Реализована обычным JS+canvas (без WASM, в отличие от /admin/infrastructure)
// - см. kontext/09-canvas-best-practices.md и kontext/08-svg-infra-icon-library.md.
//
// Маршрут: /admin/hwinfra
func (this *MainController) AdminHwInfra() {
	sess := this.GetSession("loadmon")
	m := sess.(models.LoadFilter).Map
	priv, _ := m["privilage"].(int)
	if priv != 3 {
		this.Redirect("/sprsrv/view2/all/0", 302)
		return
	}

	this.activeContent("admin/hwinfra")
	flash := beego.NewFlash()
	RegisterRequest(this, LogTypeAdminAction)

	data, err := buildHWInfraData()
	if err != nil {
		log.Printf("AdminHwInfra: build data: %v", err)
		flash.Error("Не удалось загрузить данные инфраструктуры: %v", err)
		this.Data["HWInfraJSON"] = base64.StdEncoding.EncodeToString([]byte("{}"))
	} else {
		raw, err := json.Marshal(data)
		if err != nil {
			log.Printf("AdminHwInfra: marshal: %v", err)
			flash.Error("Ошибка сериализации данных инфраструктуры")
			this.Data["HWInfraJSON"] = base64.StdEncoding.EncodeToString([]byte("{}"))
		} else {
			this.Data["HWInfraJSON"] = base64.StdEncoding.EncodeToString(raw)
		}
	}

	flash.Store(&this.Controller)
	this.TplName = "admin/hwinfra.tpl"
}
