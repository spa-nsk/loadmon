package controllers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"loadmon/models"
	"strconv"
	"strings"

	"github.com/beego/beego/v2/client/orm"
	beego "github.com/beego/beego/v2/server/web"
)

// hwConfVisualComp - компонент сервера (ServerHardwareComponents) для 3D-вида
// /sprsrv/hwconfvisual. Kind: disk, cpu, ram, nic, fc, raid, pci, psu, fan.
type hwConfVisualComp struct {
	Id          int64        `json:"id"`
	Kind        string       `json:"kind"`
	Index       int          `json:"index"` // порядковый индекс среди компонентов этого вида (0-based)
	Slot        string       `json:"slot"`
	Name        string       `json:"name"`
	Vendor      string       `json:"vendor"`
	Model       string       `json:"model"`
	SerialNum   string       `json:"serialNum"`
	CapacityGb  float64      `json:"capacityGb"`
	SubType     int          `json:"subType"`
	SubTypeName string       `json:"subTypeName"`
	Role        string       `json:"role"`     // диски: Data | Spare | Raw | Failed | Cache | ""
	RoleName    string       `json:"roleName"` // человекочитаемое назначение диска
	Status      string       `json:"status"`   // здоровье по OneView ("OK", "Warning", ...) либо ""
	Rows        []rackViewKV `json:"rows"`
}

// hwConfVisualServer - данные сервера для 3D-отрисовки.
type hwConfVisualServer struct {
	Id         int64          `json:"id"`
	Name       string         `json:"name"`
	Vendor     string         `json:"vendor"`
	Model      string         `json:"model"`
	SerialNum  string         `json:"serialNum"`
	FormFactor int            `json:"formFactor"` // 1 rack, 2 blade
	SizeU      int            `json:"sizeU"`
	Power      int            `json:"power"`
	Rows       []rackViewKV   `json:"rows"`
	Links      []rackViewLink `json:"links"`
	CPUCount   int            `json:"cpuCount"`
	MemoryGb   int            `json:"memoryGb"`
	MgmtIP     string         `json:"mgmtIp"`
	MgmtModel  string         `json:"mgmtModel"`
	RatedW     float64        `json:"ratedW"`
	MgmtRows   []rackViewKV   `json:"mgmtRows"`  // карточка контроллера управления (iLO/iDRAC/...)
	PowerRows  []rackViewKV   `json:"powerRows"` // сводка по питанию сервера (для БП/вентиляторов без собственных данных)
}

// hwConfVisualData - JSON для /sprsrv/hwconfvisual/:id.
type hwConfVisualData struct {
	Server     hwConfVisualServer `json:"server"`
	Components []hwConfVisualComp `json:"components"`
}

// hwConfVisualKinds - типы компонентов сервера, выводимые в 3D-виде.
var hwConfVisualKinds = []struct {
	Type  int
	Kind  string
	Title string
}{
	{2, "disk", "Диск"},
	{11, "cpu", "Процессор"},
	{1, "ram", "Модуль памяти"},
	{3, "nic", "Сетевой адаптер"},
	{4, "fc", "FC-адаптер"},
	{5, "raid", "RAID-контроллер"},
	{9, "pci", "PCI-устройство"},
	{12, "psu", "Блок питания"},
	{13, "fan", "Вентилятор"},
}

func hwDiskSubTypeName(sub int) string {
	switch sub {
	case 1:
		return "SSD"
	case 2:
		return "SAS"
	case 3:
		return "SATA"
	}
	return ""
}

// hwTypicalFanCount — сколько вентиляторов обычно у rack-сервера данного U /
// семейства. OneView server-hardware вентиляторы не отдаёт; оценка нужна, чтобы
// 3D-вид сверху не был пустым. Точные S/N и health — только из iLO Redfish.
func hwTypicalFanCount(sizeU int, model string) int {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "dl360"), strings.Contains(m, "r640"), strings.Contains(m, "r650"):
		return 7
	case strings.Contains(m, "dl380"), strings.Contains(m, "dl385"), strings.Contains(m, "r740"),
		strings.Contains(m, "r750"):
		return 6
	case sizeU == 1:
		return 7
	case sizeU <= 0, sizeU >= 2:
		return 6
	}
	return 6
}

// hwDiskRoleName - подпись назначения диска (models.ServerHardwareComponents.Role).
func hwDiskRoleName(role string) string {
	switch role {
	case "Data":
		return "Рабочий (в массиве)"
	case "Spare":
		return "Горячий резерв (hot spare)"
	case "Raw":
		return "Не в массиве (не сконфигурирован)"
	case "Failed":
		return "Отказавший"
	case "Cache":
		return "Кэш-диск"
	}
	return role
}

// SprHwConfVisual - 3D-визуализация конфигурации сервера по вендору/модели
// (HPE, Dell, Supermicro, DEPO и др.): лицевая панель с дисками, тыльная
// сторона (БП, адаптеры, порты управления) и вид сверху со снятым кожухом
// (вентиляторы, процессоры, память).
//
// Маршрут: /sprsrv/hwconfvisual/:id([0-9]+)
func (this *MainController) SprHwConfVisual() {
	idStr := this.Ctx.Input.Param(":id")
	RegisterRequest(this, LogTypeViewReference)

	this.activeContent("sprsrv/hwconfvisual")
	this.Data["Id"] = idStr
	this.Data["VisualJSON"] = template.JS("null")

	flash := beego.NewFlash()

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		flash.Error("Некорректный идентификатор записи: %s", idStr)
		flash.Store(&this.Controller)
		return
	}

	o := orm.NewOrm()
	var hw models.ServerHardware
	if err := o.QueryTable("server_hardware").Filter("Id", id).One(&hw); err != nil {
		flash.Error("Конфигурация сервера с Id=%d не найдена: %v", id, err)
		flash.Store(&this.Controller)
		return
	}
	this.Data["HW"] = hw

	name := firstNonEmpty(hw.SerialNum, hw.Model, fmt.Sprintf("srv-%d", hw.Id))
	var spr models.ServerSprRecords
	hasSpr := false
	if hw.ServerId > 0 {
		if err := models.ServerSprQS(o).Filter("Id", hw.ServerId).One(&spr); err == nil {
			hasSpr = true
			if strings.TrimSpace(spr.Name) != "" {
				name = spr.Name
			}
		}
	}
	if !hasSpr {
		_ = models.ServerSprQS(o).Filter("Stat", 1).Filter("HWSprId", hw.Id).One(&spr)
		if spr.Id > 0 {
			hasSpr = true
			if strings.TrimSpace(spr.Name) != "" {
				name = spr.Name
			}
		}
	}
	this.Data["ServerName"] = name

	sizeU := hw.SizeUnit
	if sizeU <= 0 {
		sizeU = rackGuessSizeU(hw.Model)
	}
	if sizeU <= 0 {
		if hw.FormFactor == 2 {
			sizeU = 1
		} else {
			sizeU = 2
		}
	}

	var rows []rackViewKV
	rows = rackViewKVAdd(rows, "Производитель", hw.Vendor)
	rows = rackViewKVAdd(rows, "Модель", hw.Model)
	rows = rackViewKVAdd(rows, "Короткая модель", hw.ShortModel)
	rows = rackViewKVAdd(rows, "Серийный номер", hw.SerialNum)
	rows = rackViewKVAdd(rows, "Номер продукта", hw.PartNum)
	rows = rackViewKVAdd(rows, "Инвентарный номер", hw.InventoryNum)
	if hw.FormFactor == 2 {
		rows = rackViewKVAdd(rows, "Форм-фактор", "Блейд")
		if hw.BladeNum > 0 {
			rows = rackViewKVAdd(rows, "Слот в корзине", fmt.Sprintf("%d", hw.BladeNum))
		}
	} else if hw.FormFactor == 1 {
		rows = rackViewKVAdd(rows, "Форм-фактор", "Rack")
	}
	if hw.CPUCount > 0 {
		rows = rackViewKVAdd(rows, "Процессоры", fmt.Sprintf("%d × %s (%d ядер @ %d MHz)", hw.CPUCount, strings.TrimSpace(hw.CPUModel), hw.CorePerCPU, hw.CoreMHz))
	}
	if hw.MemoryMb > 0 {
		rows = rackViewKVAdd(rows, "Память", fmt.Sprintf("%d ГБ", hw.MemoryMb/1024))
	}
	rows = rackViewKVAdd(rows, "IP iLO/MP", hw.MpIPAddress)
	rows = rackViewPowerRows(rows, hw.PowerAvg24hW, hw.PowerPeak24hW, hw.PowerRatedMaxW)
	rows = rackViewKVAdd(rows, "Состояние (health)", hw.Status)

	links := []rackViewLink{
		{Title: "Карточка конфигурации", URL: fmt.Sprintf("/sprsrv/hwconfedit-tab/%d", hw.Id)},
	}
	serverSprId := hw.ServerId
	if hasSpr && spr.Id > 0 {
		serverSprId = spr.Id
	}
	if serverSprId > 0 {
		links = append(links, rackViewLink{Title: "Общие данные по серверу", URL: fmt.Sprintf("/sprsrv/updatebase-tab/%d", serverSprId)})
	}
	if sn := strings.TrimSpace(hw.SerialNum); sn != "" {
		links = append(links, rackViewLink{Title: "Данные OneView", URL: "/sprsrv/srvhwviewbysn/" + sn})
	}

	// Компоненты сервера (по ServerId = ServerHardware.Id).
	comps := make([]hwConfVisualComp, 0, 64)
	countByKind := map[string]int{}
	for _, kd := range hwConfVisualKinds {
		var list []models.ServerHardwareComponents
		_, _ = o.QueryTable(new(models.ServerHardwareComponents)).
			Filter("ServerId", hw.Id).Filter("Type", kd.Type).
			OrderBy("Slot", "Id").All(&list)
		for i, c := range list {
			cname := firstNonEmpty(c.Model, c.SerialNum, fmt.Sprintf("%s %d", kd.Title, i+1))
			var crow []rackViewKV
			crow = rackViewKVAdd(crow, "Слот / расположение", c.Slot)
			crow = rackViewKVAdd(crow, "Производитель", c.Vendor)
			crow = rackViewKVAdd(crow, "Модель", c.Model)
			crow = rackViewKVAdd(crow, "Серийный номер", c.SerialNum)
			if c.CapacityGb > 0 {
				if kd.Kind == "psu" {
					crow = rackViewKVAdd(crow, "Мощность (номинал)", fmt.Sprintf("%.0f Вт", c.CapacityGb))
				} else {
					crow = rackViewKVAdd(crow, "Ёмкость", fmt.Sprintf("%.1f ГБ", c.CapacityGb))
				}
			}
			stName, roleName := "", ""
			if kd.Kind == "disk" {
				stName = hwDiskSubTypeName(c.SubType)
				if stName != "" {
					crow = rackViewKVAdd(crow, "Тип носителя", stName)
				}
				if c.Role != "" {
					roleName = hwDiskRoleName(c.Role)
					crow = rackViewKVAdd(crow, "Назначение", roleName)
				}
				if c.Role == "Spare" {
					countByKind["disk-spare"]++
				}
			}
			crow = rackViewKVAdd(crow, "Состояние (health)", c.Status)
			comps = append(comps, hwConfVisualComp{
				Id: c.Id, Kind: kd.Kind, Index: i, Slot: c.Slot, Name: cname,
				Vendor: c.Vendor, Model: c.Model, SerialNum: c.SerialNum,
				CapacityGb: c.CapacityGb, SubType: c.SubType, SubTypeName: stName,
				Role: c.Role, RoleName: roleName, Status: c.Status,
				Rows: crow,
			})
			countByKind[kd.Kind]++
		}
	}
	// Процессоры не синхронизированы как компоненты - показываем по данным сервера.
	if countByKind["cpu"] == 0 && hw.CPUCount > 0 {
		for i := 0; i < hw.CPUCount; i++ {
			var crow []rackViewKV
			crow = rackViewKVAdd(crow, "Сокет", fmt.Sprintf("%d", i+1))
			crow = rackViewKVAdd(crow, "Модель", strings.TrimSpace(hw.CPUModel))
			if hw.CorePerCPU > 0 {
				crow = rackViewKVAdd(crow, "Ядер", fmt.Sprintf("%d", hw.CorePerCPU))
			}
			if hw.CoreMHz > 0 {
				crow = rackViewKVAdd(crow, "Частота", fmt.Sprintf("%d MHz", hw.CoreMHz))
			}
			comps = append(comps, hwConfVisualComp{
				Kind: "cpu", Index: i, Slot: fmt.Sprintf("Сокет %d", i+1),
				Name: firstNonEmpty(strings.TrimSpace(hw.CPUModel), fmt.Sprintf("Процессор %d", i+1)),
				Model: strings.TrimSpace(hw.CPUModel), Rows: crow,
			})
			countByKind["cpu"]++
		}
	}
	// Вентиляторы rack-серверов OneView не отдаёт (только enclosure у блейд-корзин).
	// Для 3D подставляем типовое число по U/модели — с пометкой «оценка», чтобы
	// на виде сверху были не пустые заглушки без пояснения.
	fanEstimated := false
	if countByKind["fan"] == 0 && hw.FormFactor != 2 {
		nFan := hwTypicalFanCount(sizeU, hw.Model)
		for i := 0; i < nFan; i++ {
			var crow []rackViewKV
			crow = rackViewKVAdd(crow, "Позиция", fmt.Sprintf("%d", i+1))
			crow = rackViewKVAdd(crow, "Источник данных", "оценка по форм-фактору (OneView не отдаёт вентиляторы rack-серверов)")
			crow = rackViewKVAdd(crow, "Как получить точные данные", "iLO Redfish: /redfish/v1/Chassis/1/Thermal")
			crow = rackViewKVAdd(crow, "Сервер", firstNonEmpty(hw.Model, hw.ShortModel))
			comps = append(comps, hwConfVisualComp{
				Kind: "fan", Index: i, Slot: fmt.Sprintf("Fan %d", i+1),
				Name: fmt.Sprintf("Вентилятор %d (оценка)", i+1),
				Model: fmt.Sprintf("типовая конфигурация %dU", sizeU),
				Rows: crow,
			})
			countByKind["fan"]++
		}
		fanEstimated = nFan > 0
	}
	rows = rackViewKVAdd(rows, "Дисков", fmt.Sprintf("%d", countByKind["disk"]))
	if n := countByKind["disk-spare"]; n > 0 {
		rows = rackViewKVAdd(rows, "Из них горячий резерв", fmt.Sprintf("%d", n))
	}
	if n := countByKind["ram"]; n > 0 {
		rows = rackViewKVAdd(rows, "Модулей памяти", fmt.Sprintf("%d", n))
	}
	if n := countByKind["nic"] + countByKind["fc"] + countByKind["raid"] + countByKind["pci"]; n > 0 {
		rows = rackViewKVAdd(rows, "Адаптеров/портов (NIC, FC, RAID, PCI)", fmt.Sprintf("%d", n))
	}
	if n := countByKind["pci"]; n > 0 {
		rows = rackViewKVAdd(rows, "PCI-устройств (Devices)", fmt.Sprintf("%d", n))
	}
	if n := countByKind["fan"]; n > 0 {
		label := "Вентиляторов"
		if fanEstimated {
			label = "Вентиляторов (оценка по U)"
		}
		rows = rackViewKVAdd(rows, label, fmt.Sprintf("%d", n))
	}

	// Контроллер управления (iLO/iDRAC/...): всё, что OneView отдаёт о нём в server-hardware.
	var mgmtRows []rackViewKV
	mgmtRows = rackViewKVAdd(mgmtRows, "IP адрес", hw.MpIPAddress)
	mgmtRows = rackViewKVAdd(mgmtRows, "Модель контроллера", hw.MpModel)
	mgmtRows = rackViewKVAdd(mgmtRows, "Прошивка контроллера", hw.MpFirmwareVersion)
	mgmtRows = rackViewKVAdd(mgmtRows, "Версия BIOS/ROM", hw.RomVersion)
	// Питание сервера (замеры OneView) - показывается у БП/вентиляторов, для которых нет собственных данных.
	var powerRows []rackViewKV
	if hw.PowerAvg24hW > 0 {
		powerRows = rackViewKVAdd(powerRows, "Мощность сервера, средняя 24ч", fmt.Sprintf("%.0f Вт", hw.PowerAvg24hW))
	}
	if hw.PowerPeak24hW > 0 {
		powerRows = rackViewKVAdd(powerRows, "Мощность сервера, пик 24ч", fmt.Sprintf("%.0f Вт", hw.PowerPeak24hW))
	}
	if hw.PowerRatedMaxW > 0 {
		powerRows = rackViewKVAdd(powerRows, "Мощность сервера, паспорт", fmt.Sprintf("%.0f Вт", hw.PowerRatedMaxW))
	}

	payload, jerr := json.Marshal(hwConfVisualData{
		Server: hwConfVisualServer{
			Id: hw.Id, Name: name, Vendor: hw.Vendor, Model: firstNonEmpty(hw.Model, hw.ShortModel),
			SerialNum: hw.SerialNum, FormFactor: hw.FormFactor, SizeU: sizeU,
			Power: rackViewPower(hw.PowerState), Rows: rows, Links: links,
			CPUCount: hw.CPUCount, MemoryGb: hw.MemoryMb / 1024,
			MgmtIP: hw.MpIPAddress, MgmtModel: hw.MpModel, RatedW: hw.PowerRatedMaxW,
			MgmtRows: mgmtRows, PowerRows: powerRows,
		},
		Components: comps,
	})
	if jerr != nil {
		flash.Error("Ошибка сериализации данных визуализации: %v", jerr)
		flash.Store(&this.Controller)
		return
	}
	this.Data["VisualJSON"] = template.JS(payload)
	this.Data["DisksCount"] = countByKind["disk"]
}
