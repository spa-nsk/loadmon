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

// hwConfVisualDisk - физический диск сервера (ServerHardwareComponents Type=2)
// для наведения/выбора на canvas /sprsrv/hwconfvisual.
type hwConfVisualDisk struct {
	Id          int64        `json:"id"`
	Index       int          `json:"index"` // порядковый индекс в корзине дисков (0-based)
	Slot        string       `json:"slot"`
	Name        string       `json:"name"`
	Vendor      string       `json:"vendor"`
	Model       string       `json:"model"`
	SerialNum   string       `json:"serialNum"`
	CapacityGb  float64      `json:"capacityGb"`
	SubType     int          `json:"subType"`
	SubTypeName string       `json:"subTypeName"`
	Rows        []rackViewKV `json:"rows"`
}

// hwConfVisualServer - данные лицевой панели сервера для отрисовки.
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
}

// hwConfVisualData - JSON для /sprsrv/hwconfvisual/:id.
type hwConfVisualData struct {
	Server hwConfVisualServer `json:"server"`
	Disks  []hwConfVisualDisk `json:"disks"`
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

// SprHwConfVisual - визуализация конфигурации сервера по вендору/модели
// (лицевая панель как в /sprsrv/rackcabinetview) с наведением на диски.
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

	const diskType = 2 // cmdb.ServerHWComponentDisk
	var diskComps []models.ServerHardwareComponents
	_, _ = o.QueryTable(new(models.ServerHardwareComponents)).
		Filter("ServerId", hw.Id).Filter("Type", diskType).
		OrderBy("Slot", "Id").All(&diskComps)

	disks := make([]hwConfVisualDisk, 0, len(diskComps))
	for i, c := range diskComps {
		dname := firstNonEmpty(c.Model, c.SerialNum, fmt.Sprintf("Диск %d", i+1))
		var drows []rackViewKV
		drows = rackViewKVAdd(drows, "Слот / расположение", c.Slot)
		drows = rackViewKVAdd(drows, "Производитель", c.Vendor)
		drows = rackViewKVAdd(drows, "Модель", c.Model)
		drows = rackViewKVAdd(drows, "Серийный номер", c.SerialNum)
		if c.CapacityGb > 0 {
			drows = rackViewKVAdd(drows, "Ёмкость", fmt.Sprintf("%.1f ГБ", c.CapacityGb))
		}
		if st := hwDiskSubTypeName(c.SubType); st != "" {
			drows = rackViewKVAdd(drows, "Тип носителя", st)
		}
		disks = append(disks, hwConfVisualDisk{
			Id: c.Id, Index: i, Slot: c.Slot, Name: dname,
			Vendor: c.Vendor, Model: c.Model, SerialNum: c.SerialNum,
			CapacityGb: c.CapacityGb, SubType: c.SubType, SubTypeName: hwDiskSubTypeName(c.SubType),
			Rows: drows,
		})
	}
	rows = rackViewKVAdd(rows, "Дисков", fmt.Sprintf("%d", len(disks)))

	payload, jerr := json.Marshal(hwConfVisualData{
		Server: hwConfVisualServer{
			Id: hw.Id, Name: name, Vendor: hw.Vendor, Model: firstNonEmpty(hw.Model, hw.ShortModel),
			SerialNum: hw.SerialNum, FormFactor: hw.FormFactor, SizeU: sizeU,
			Power: rackViewPower(hw.PowerState), Rows: rows, Links: links,
		},
		Disks: disks,
	})
	if jerr != nil {
		flash.Error("Ошибка сериализации данных визуализации: %v", jerr)
		flash.Store(&this.Controller)
		return
	}
	this.Data["VisualJSON"] = template.JS(payload)
	this.Data["DisksCount"] = len(disks)
}
