package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/models"
	"loadmon/rackpower"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// rackViewLink - ссылка в подсказке оборудования (подробности/метрики) -
// URL-ы собираются здесь, на сервере, чтобы JS не знал схему маршрутов.
type rackViewLink struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// rackViewKV - строка "параметр: значение" в подсказке оборудования.
type rackViewKV struct {
	K string `json:"k"`
	V string `json:"v"`
}

// rackViewBlade - блейд-сервер в слоте корзины (ServerHardware, FormFactor=2).
type rackViewBlade struct {
	Slot  int            `json:"slot"`
	Name  string         `json:"name"`
	Model string         `json:"model"`
	Power int            `json:"power"` // 0 - не определено, 1 - включен, 2 - выключен
	Rows  []rackViewKV   `json:"rows"`
	Links []rackViewLink `json:"links"`
}

// rackViewItem - единица оборудования шкафа. Kind: server, chassis, storage,
// switch. StartU/SizeU == 0 - позиция в шкафу неизвестна (оборудование
// показывается списком "без позиции").
type rackViewItem struct {
	Kind      string          `json:"kind"`
	Id        int64           `json:"id"`
	Name      string          `json:"name"`
	Vendor    string          `json:"vendor"`
	Model     string          `json:"model"`
	StartU    int             `json:"startU"`
	SizeU     int             `json:"sizeU"`
	Power     int             `json:"power"` // 0 - не определено, 1 - включен, 2 - выключен
	Slots     int             `json:"slots,omitempty"`
	Rows      []rackViewKV    `json:"rows"`
	Links     []rackViewLink  `json:"links"`
	Blades    []rackViewBlade `json:"blades,omitempty"`
	SizeGuess bool            `json:"sizeGuess,omitempty"`
}

type rackViewData struct {
	CabinetId   int64          `json:"cabinetId"`
	CabinetName string         `json:"cabinetName"`
	Site        string         `json:"site"`
	UHeight     int            `json:"uHeight"`
	Items       []rackViewItem `json:"items"`
	Unplaced    []rackViewItem `json:"unplaced"`
	// Суммарная мощность шкафа, Вт: тот же расчёт, что и в списке шкафов
	// (rackpower.RackCabinetPowerVariantC - rack-серверы + корзины целиком, блейды не дублируются).
	PowerAvgW   float64 `json:"powerAvgW"`
	PowerPeakW  float64 `json:"powerPeakW"`
	PowerRatedW float64 `json:"powerRatedW"`
}

// rackGuessSizeU - высота сервера в юнитах по модели, если в справочнике
// SizeUnit не заполнен (по распространённым линейкам HPE/Dell/Lenovo).
func rackGuessSizeU(model string) int {
	m := strings.ToLower(model)
	for _, k := range []string{"dl360", "dl325", "dl20", "dl160", "dl120", "r640", "r650", "r440", "r450", "r340", "sr630", "sr635", "sr250", "sr530", "sd530"} {
		if strings.Contains(m, k) {
			return 1
		}
	}
	for _, k := range []string{"dl580", "dl560", "r840", "r940", "sr850", "sr950", "sr860"} {
		if strings.Contains(m, k) {
			return 4
		}
	}
	for _, k := range []string{"ml350", "ml110", "ml30"} {
		if strings.Contains(m, k) {
			return 5
		}
	}
	return 2
}

// rackViewPowerRows добавляет строки потребляемой мощности (Вт); если данных
// по мощности нет совсем - одну строку "нет данных".
func rackViewPowerRows(rows []rackViewKV, avg, peak, rated float64) []rackViewKV {
	if avg <= 0 && peak <= 0 && rated <= 0 {
		return append(rows, rackViewKV{K: "Мощность", V: "нет данных"})
	}
	if avg > 0 {
		rows = append(rows, rackViewKV{K: "Мощность, средняя 24ч", V: fmt.Sprintf("%.0f Вт", avg)})
	}
	if peak > 0 {
		rows = append(rows, rackViewKV{K: "Мощность, пик 24ч", V: fmt.Sprintf("%.0f Вт", peak)})
	}
	if rated > 0 {
		rows = append(rows, rackViewKV{K: "Мощность, паспорт", V: fmt.Sprintf("%.0f Вт", rated)})
	}
	return rows
}

func rackViewPower(state int) int {
	switch state {
	case 1:
		return 1
	case 2:
		return 2
	}
	return 0
}

func rackViewKVAdd(rows []rackViewKV, k, v string) []rackViewKV {
	v = strings.TrimSpace(v)
	if v == "" {
		return rows
	}
	return append(rows, rackViewKV{K: k, V: v})
}

// SprRackCabinetView отображает шкаф (models.RackCabinet) с установленным
// оборудованием на HTML5 canvas - серверы (models.ServerHardware), блейд-
// корзины (models.BladeSystems) с блейдами в слотах и СХД
// (models.StorageSystemPlacement/StorageSystems) на своих юнитах. Коммутаторы
// (models.SwitchSprRecords) позиции в шкафу не имеют - выводятся списком
// "без позиции" вместе с остальным оборудованием, у которого не указан юнит.
//
// Данные передаются в шаблон одним JSON (rackViewData) - рисование и
// подсказки целиком на стороне браузера, см.
// server_web_new/views/sprsrv/rackcabinetview.tpl.
//
// Вызывается по правому клику на /sprsrv/rackcabinetlist ("Вид шкафа").
//
// Маршрут: /sprsrv/rackcabinetview/:id
func (this *MainController) SprRackCabinetView() {

	idStr := this.Ctx.Input.Param(":id")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	RegisterRequest(this, LogTypeViewReference) // просмотр шкафа с оборудованием

	this.activeContent("sprsrv/rackcabinetview")
	this.Data["Embed"] = this.GetString("embed") == "1" // показ во фрейме просмотра ЦОД: без боковой панели

	this.Data["Id"] = idStr
	this.Data["CabinetName"] = ""
	this.Data["SiteName"] = ""
	this.Data["UHeight"] = 0
	this.Data["ItemCount"] = 0
	this.Data["RackJSON"] = template.JS("null")

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор шкафа: %s", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()

	var cab models.RackCabinet
	if err := o.QueryTable(new(models.RackCabinet)).Filter("Id", id).One(&cab); err != nil {
		flash.Error("Шкаф с Id=%d не найден: %v", id, err)
		flash.Store(&this.Controller)
		return
	}

	siteName := ""
	if cab.DatacenterSiteId > 0 {
		var site models.DatacenterSite
		if err := o.QueryTable(new(models.DatacenterSite)).Filter("Id", cab.DatacenterSiteId).One(&site); err == nil {
			siteName = site.Name
		}
	}

	uHeight := int(cab.UHeight)
	if uHeight <= 0 {
		uHeight = 42
	}

	data := rackViewData{
		CabinetId:   cab.Id,
		CabinetName: cab.Name,
		Site:        siteName,
		UHeight:     uHeight,
		Items:       []rackViewItem{},
		Unplaced:    []rackViewItem{},
	}

	add := func(it rackViewItem) {
		if it.StartU > 0 && it.SizeU > 0 {
			if it.StartU+it.SizeU-1 > data.UHeight {
				data.UHeight = it.StartU + it.SizeU - 1
			}
			data.Items = append(data.Items, it)
		} else {
			data.Unplaced = append(data.Unplaced, it)
		}
	}

	// --- блейд-корзины и блейды в них ---
	var chassisRows []models.BladeSystems
	if _, err := o.QueryTable(new(models.BladeSystems)).
		Filter("RackCabinetId", id).Exclude("Active", 2).OrderBy("StartUnitPos").All(&chassisRows); err != nil {
		flash.Notice("Корзины не загружены: %v", err)
	}
	chassisIds := make([]int64, 0, len(chassisRows))
	for _, ch := range chassisRows {
		chassisIds = append(chassisIds, ch.Id)
	}
	var bladeRows []models.ServerHardware
	if len(chassisIds) > 0 {
		if _, err := o.QueryTable(new(models.ServerHardware)).
			Filter("BladeSystemId__in", chassisIds).Exclude("Active", 2).OrderBy("BladeNum").All(&bladeRows); err != nil {
			flash.Notice("Блейд-серверы не загружены: %v", err)
		}
	}

	// --- серверы шкафа (rack) ---
	var hwRows []models.ServerHardware
	if _, err := o.QueryTable(new(models.ServerHardware)).
		Filter("RackCabinetId", id).Exclude("Active", 2).OrderBy("StartUnitPos").All(&hwRows); err != nil {
		flash.Notice("Серверы не загружены: %v", err)
	}
	chassisInRack := make(map[int64]bool, len(chassisRows))
	for _, ch := range chassisRows {
		chassisInRack[ch.Id] = true
	}
	// Блейд, который нельзя показать внутри корзины этого шкафа (нет корзины
	// или корзина в другом шкафу/выведена), не пропадает, а попадает в список
	// "без позиции" с пометкой - включенный блейд вне корзины это ошибка данных.
	bladeNote := map[int64]string{}
	rackServers := make([]models.ServerHardware, 0, len(hwRows))
	for _, hw := range hwRows {
		if hw.FormFactor == 2 {
			if hw.BladeSystemId > 0 && chassisInRack[hw.BladeSystemId] {
				continue // блейды рисуются внутри корзины
			}
			if hw.BladeSystemId > 0 && hw.Active == 1 {
				// работающий блейд размещается по корзине (его собственный
				// RackCabinetId не важен): в этом шкафу его нет, он виден в шкафу корзины
				continue
			}
			if hw.BladeSystemId > 0 {
				bladeNote[hw.Id] = "блейд-сервер: корзина (Id " + strconv.FormatInt(hw.BladeSystemId, 10) + ") не в этом шкафу или выведена из эксплуатации"
			} else {
				bladeNote[hw.Id] = "блейд-сервер без блейд-корзины"
			}
			if hw.Active == 1 {
				bladeNote[hw.Id] += " - ОШИБКА: включенный блейд должен находиться в корзине"
			}
		}
		rackServers = append(rackServers, hw)
	}

	// имена/IP/Zabbix из справочника серверов по ServerHardware.Id (HWSprId)
	hwIds := make([]int64, 0, len(rackServers)+len(bladeRows))
	for _, hw := range rackServers {
		hwIds = append(hwIds, hw.Id)
	}
	for _, hw := range bladeRows {
		hwIds = append(hwIds, hw.Id)
	}
	sprByHW := map[int64]models.ServerSprRecords{}
	if len(hwIds) > 0 {
		var sprRows []models.ServerSprRecords
		if _, err := models.ServerSprQS(o).Filter("Stat", 1).Filter("HWSprId__in", hwIds).All(&sprRows); err == nil {
			for _, s := range sprRows {
				if _, dup := sprByHW[s.HWSprId]; !dup {
					sprByHW[s.HWSprId] = s
				}
			}
		}
	}

	serverRowsLinks := func(hw models.ServerHardware) ([]rackViewKV, []rackViewLink, string) {
		spr, hasSpr := sprByHW[hw.Id]
		name := ""
		var rows []rackViewKV
		var links []rackViewLink
		if hasSpr && strings.TrimSpace(spr.Name) != "" {
			name = spr.Name
		}
		if name == "" {
			name = firstNonEmpty(hw.SerialNum, hw.Model, fmt.Sprintf("srv-%d", hw.Id))
		}
		if hasSpr {
			rows = rackViewKVAdd(rows, "FQDN", spr.NameFQDN)
			rows = rackViewKVAdd(rows, "IP сервера", spr.IPAddress)
			if spr.Type == 2 {
				rows = rackViewKVAdd(rows, "Роль", "Гипервизор VMware")
			}
		}
		rows = rackViewKVAdd(rows, "Серийный номер", hw.SerialNum)
		rows = rackViewKVAdd(rows, "Номер продукта", hw.PartNum)
		rows = rackViewKVAdd(rows, "Инвентарный номер", hw.InventoryNum)
		rows = rackViewKVAdd(rows, "IP iLO/MP", hw.MpIPAddress)
		if hw.CPUCount > 0 {
			rows = rackViewKVAdd(rows, "Процессоры", fmt.Sprintf("%d × %s (%d ядер)", hw.CPUCount, strings.TrimSpace(hw.CPUModel), hw.CorePerCPU))
		}
		if hw.MemoryMb > 0 {
			rows = rackViewKVAdd(rows, "Память", fmt.Sprintf("%d ГБ", hw.MemoryMb/1024))
		}
		rows = rackViewPowerRows(rows, hw.PowerAvg24hW, hw.PowerPeak24hW, hw.PowerRatedMaxW)
		rows = rackViewKVAdd(rows, "Состояние (health)", hw.Status)

		serverSprId := hw.ServerId
		if hasSpr && spr.Id > 0 {
			serverSprId = spr.Id
		}
		if serverSprId > 0 {
			links = append(links, rackViewLink{Title: "Общие данные по серверу", URL: fmt.Sprintf("/sprsrv/updatebase/all/%d", serverSprId)})
		}
		if serverSprId > 0 {
			links = append(links,
				rackViewLink{Title: "Мощность за 3 суток", URL: fmt.Sprintf("/sprsrv/powerrecent/server/%d", serverSprId)},
				rackViewLink{Title: "Мощность за период", URL: fmt.Sprintf("/sprsrv/powerview/server/%d", serverSprId)})
		}
		links = append(links, rackViewLink{Title: "Карточка оборудования", URL: fmt.Sprintf("/sprsrv/hwconfedit/%d", hw.Id)})
		if sn := strings.TrimSpace(hw.SerialNum); sn != "" {
			links = append(links, rackViewLink{Title: "Данные OneView", URL: "/sprsrv/srvhwviewbysn/" + sn})
		}
		if hasSpr && spr.ZabbixHostId > 0 {
			links = append(links, rackViewLink{Title: "Метрики производительности (Zabbix)", URL: fmt.Sprintf("/sprsrv/zabbixperfperiod/%d", spr.ZabbixHostId)})
		}
		return rows, links, name
	}

	for _, hw := range rackServers {
		rows, links, name := serverRowsLinks(hw)
		if note := bladeNote[hw.Id]; note != "" {
			rows = append([]rackViewKV{{K: "Внимание", V: note}}, rows...)
		}
		sizeU := hw.SizeUnit
		guess := false
		if sizeU <= 0 && hw.StartUnitPos > 0 {
			sizeU = rackGuessSizeU(hw.Model)
			guess = true
		}
		add(rackViewItem{
			Kind: "server", Id: hw.Id, Name: name, Vendor: hw.Vendor, Model: hw.Model,
			StartU: hw.StartUnitPos, SizeU: sizeU, Power: rackViewPower(hw.PowerState),
			Rows: rows, Links: links, SizeGuess: guess,
		})
	}

	bladesByChassis := map[int64][]models.ServerHardware{}
	for _, b := range bladeRows {
		bladesByChassis[b.BladeSystemId] = append(bladesByChassis[b.BladeSystemId], b)
	}
	for _, ch := range chassisRows {
		slots := ch.Size
		if slots <= 0 {
			slots = 16
		}
		sizeU := ch.SizeUnit
		if sizeU <= 0 && ch.EndUnitPos >= ch.StartUnitPos && ch.StartUnitPos > 0 {
			sizeU = ch.EndUnitPos - ch.StartUnitPos + 1
		}
		if sizeU <= 0 && ch.StartUnitPos > 0 {
			sizeU = 10
		}
		var rows []rackViewKV
		rows = rackViewKVAdd(rows, "Тип", ch.Type)
		rows = rackViewKVAdd(rows, "Серийный номер", ch.SerialNum)
		rows = rackViewKVAdd(rows, "Номер продукта", ch.PartNum)
		rows = rackViewKVAdd(rows, "Инвентарный номер", ch.InventoryNum)
		rows = rackViewKVAdd(rows, "Слотов (занято)", fmt.Sprintf("%d (%d)", slots, len(bladesByChassis[ch.Id])))
		rows = rackViewPowerRows(rows, ch.PowerAvg24hW, ch.PowerPeak24hW, ch.PowerRatedMaxW)
		links := []rackViewLink{
			{Title: "Серверы в корзине", URL: fmt.Sprintf("/sprsrv/bladesystemserverslist/%d", ch.Id)},
			{Title: "Мощность корзины за период (суточные, OneView)", URL: fmt.Sprintf("/sprsrv/powerview/blade/%d", ch.Id)},
			{Title: "Редактировать корзину", URL: fmt.Sprintf("/sprsrv/bladesystemsedit/%d", ch.Id)},
		}
		blades := make([]rackViewBlade, 0, len(bladesByChassis[ch.Id]))
		for _, b := range bladesByChassis[ch.Id] {
			brows, blinks, bname := serverRowsLinks(b)
			brows = rackViewKVAdd(brows, "Учёт в мощности шкафа", "не суммируется отдельно (учтена в мощности корзины)")
			blades = append(blades, rackViewBlade{
				Slot: b.BladeNum, Name: bname, Model: firstNonEmpty(b.Model, b.ShortModel),
				Power: rackViewPower(b.PowerState), Rows: brows, Links: blinks,
			})
		}
		add(rackViewItem{
			Kind: "chassis", Id: ch.Id, Name: firstNonEmpty(ch.Name, ch.SerialNum, fmt.Sprintf("chassis-%d", ch.Id)),
			Vendor: ch.Vendor, Model: firstNonEmpty(ch.Model, ch.Type), StartU: ch.StartUnitPos, SizeU: sizeU,
			Power: rackViewPower(ch.PowerState), Slots: slots, Rows: rows, Links: links, Blades: blades,
		})
	}

	// --- СХД: размещения в этом шкафу ---
	var placements []models.StorageSystemPlacement
	if _, err := o.QueryTable(new(models.StorageSystemPlacement)).
		Filter("RackCabinetId", id).OrderBy("StartUnitPos").All(&placements); err != nil {
		flash.Notice("Размещения СХД не загружены: %v", err)
	}
	if len(placements) > 0 {
		stIds := make([]int64, 0, len(placements))
		for _, p := range placements {
			stIds = append(stIds, p.StorageSystemsId)
		}
		stShare := rackpower.PlacementShare(o, stIds)
		stByID := map[int64]models.StorageSystems{}
		var stRows []models.StorageSystems
		if _, err := o.QueryTable(new(models.StorageSystems)).Filter("Id__in", stIds).All(&stRows); err == nil {
			for _, s := range stRows {
				stByID[s.Id] = s
			}
		}
		for _, p := range placements {
			st, ok := stByID[p.StorageSystemsId]
			if !ok || st.Status == 6 { // 6 - демонтирована
				continue
			}
			sizeU := p.SizeUnit
			if sizeU <= 0 && p.EndUnitPos >= p.StartUnitPos && p.StartUnitPos > 0 {
				sizeU = p.EndUnitPos - p.StartUnitPos + 1
			}
			guess := false
			if sizeU <= 0 && p.StartUnitPos > 0 {
				sizeU = 2
				guess = true
			}
			power := 2
			if st.Power == 1 {
				power = 1
			}
			var rows []rackViewKV
			rows = rackViewKVAdd(rows, "IP управления", st.IpAddress)
			rows = rackViewKVAdd(rows, "Серийный номер", st.SerialNum)
			rows = rackViewKVAdd(rows, "Номер продукта", st.PartNum)
			rows = rackViewKVAdd(rows, "Инвентарный номер", st.InventoryNum)
			rows = rackViewKVAdd(rows, "Прошивка", st.Firmware)
			if st.TotalSizeRAW > 0 {
				rows = rackViewKVAdd(rows, "Ёмкость RAW", fmt.Sprintf("%.1f", st.TotalSizeRAW))
				rows = rackViewKVAdd(rows, "Занято", fmt.Sprintf("%.1f %%", st.UsedPercent))
			}
			rows = rackViewPowerRows(rows, st.PowerAvg24hW, st.PowerPeak24hW, 0)
			if share := stShare(p); share < 0.999 && (st.PowerAvg24hW > 0 || st.PowerPeak24hW > 0) {
				// СХД в нескольких шкафах: в итог этого шкафа идёт только доля
				rows = append(rows, rackViewKV{K: "Доля в этом шкафу", V: fmt.Sprintf("%.0f %%: %.0f Вт (средняя), %.0f Вт (пик)",
					share*100, st.PowerAvg24hW*share, st.PowerPeak24hW*share)})
			}
			links := []rackViewLink{
				{Title: "Конфигурация СХД (полки, диски, пулы)", URL: fmt.Sprintf("/sprsrv/storagesystemconfigview/%d", st.Id)},
				{Title: "Метрики СХД за период", URL: fmt.Sprintf("/sprsrv/storagesystemmetricsview/%d", st.Id)},
				{Title: "Метрики СХД за 3 суток", URL: fmt.Sprintf("/sprsrv/storagesystemmetricsrecent/%d", st.Id)},
				{Title: "Мощность СХД за 3 суток", URL: fmt.Sprintf("/sprsrv/powerrecent/storage/%d", st.Id)},
				{Title: "Мощность СХД за период", URL: fmt.Sprintf("/sprsrv/powerview/storage/%d", st.Id)},
				{Title: "Редактировать СХД", URL: fmt.Sprintf("/sprsrv/storagesystemsedit/%d", st.Id)},
			}
			add(rackViewItem{
				Kind: "storage", Id: st.Id, Name: firstNonEmpty(st.ShortName, st.SerialNum, fmt.Sprintf("storage-%d", st.Id)),
				Vendor: st.Vendor, Model: st.Model, StartU: p.StartUnitPos, SizeU: sizeU, Power: power,
				Rows: rows, Links: links, SizeGuess: guess,
			})
		}
	}

	// --- коммутаторы (позиции в шкафу нет) ---
	var switches []models.SwitchSprRecords
	if _, err := o.QueryTable(new(models.SwitchSprRecords)).Filter("RackCabinetId", id).OrderBy("Name").All(&switches); err != nil {
		flash.Notice("Коммутаторы не загружены: %v", err)
	}
	for _, sw := range switches {
		var rows []rackViewKV
		rows = rackViewKVAdd(rows, "IP управления", sw.ManagementIp)
		rows = rackViewKVAdd(rows, "Серийный номер", sw.SerialNum)
		rows = rackViewKVAdd(rows, "Описание", sw.Annotation)
		add(rackViewItem{
			Kind: "switch", Id: sw.Id, Name: firstNonEmpty(sw.Name, fmt.Sprintf("switch-%d", sw.Id)),
			Vendor: sw.Vendor, Model: sw.Model, Rows: rows,
			Links: []rackViewLink{{Title: "Редактировать коммутатор", URL: fmt.Sprintf("/sprsrv/switchedit/%d", sw.Id)}},
		})
	}

	pw := rackpower.RackCabinetPowerVariantC(o, id)
	data.PowerAvgW, data.PowerPeakW, data.PowerRatedW = pw.Avg24hW, pw.Peak24hW, pw.RatedMaxW

	b, jerr := json.Marshal(data)
	if jerr != nil {
		flash.Error("Ошибка подготовки данных шкафа: %v", jerr)
		flash.Store(&this.Controller)
		return
	}
	if len(flash.Data) > 0 {
		flash.Store(&this.Controller)
	}

	this.Data["CabinetName"] = cab.Name
	this.Data["SiteName"] = siteName
	this.Data["UHeight"] = data.UHeight
	this.Data["ItemCount"] = len(data.Items) + len(data.Unplaced)
	this.Data["RackJSON"] = template.JS(b)
}
