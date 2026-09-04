package main

import (
	"fmt"
	"image/color"

	"github.com/llgcode/draw2d/draw2dimg"
	"loadmon/infracanvas/payload"
)

var (
	colBg        = color.RGBA{15, 23, 42, 255}
	colPanel     = color.RGBA{30, 41, 59, 255}
	colPanelEdge = color.RGBA{71, 85, 105, 255}
	colTitle     = color.RGBA{226, 232, 240, 255}
	colText      = color.RGBA{203, 213, 225, 255}
	colMuted     = color.RGBA{148, 163, 184, 255}
	colAccent    = color.RGBA{56, 189, 248, 255}
	colBladeOn   = color.RGBA{34, 197, 94, 255}
	colBladeOff  = color.RGBA{100, 116, 139, 255}
	colEmpty     = color.RGBA{51, 65, 85, 255}
	colRackSrv   = color.RGBA{99, 102, 241, 255}
	colStorage   = color.RGBA{249, 115, 22, 255}
	colChassis   = color.RGBA{14, 165, 233, 255}
)

func clearBackground(gc *draw2dimg.GraphicContext, w, h float64) {
	gc.SetFillColor(colBg)
	gc.Clear()
	fillRect(gc, 0, 0, w, h, colBg)
}

func fillRect(gc *draw2dimg.GraphicContext, x, y, w, h float64, c color.Color) {
	gc.SetFillColor(c)
	gc.SetStrokeColor(c)
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+w, y)
	gc.LineTo(x+w, y+h)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.Fill()
}

func strokeRect(gc *draw2dimg.GraphicContext, x, y, w, h, sw float64, c color.Color) {
	gc.SetStrokeColor(c)
	gc.SetLineWidth(sw)
	gc.BeginPath()
	gc.MoveTo(x, y)
	gc.LineTo(x+w, y)
	gc.LineTo(x+w, y+h)
	gc.LineTo(x, y+h)
	gc.Close()
	gc.Stroke()
}

func drawText(gc *draw2dimg.GraphicContext, x, y float64, text string, c color.Color, size float64) {
	gc.SetFillColor(c)
	gc.SetFontSize(size)
	gc.FillStringAt(text, x, y)
}

func drawTextCenter(gc *draw2dimg.GraphicContext, cx, y float64, text string, c color.Color, size float64) {
	gc.SetFillColor(c)
	gc.SetFontSize(size)
	w := gc.CreateStringPath(text, 0, 0)
	gc.FillStringAt(text, cx-w/2, y)
}

func drawPanel(gc *draw2dimg.GraphicContext, x, y, w, h float64, title string) {
	fillRect(gc, x, y, w, h, colPanel)
	strokeRect(gc, x, y, w, h, 1.5, colPanelEdge)
	drawText(gc, x+12, y+22, title, colTitle, 14)
}

func drawHardwarePage(gc *draw2dimg.GraphicContext, width, height float64, hw payload.HardwarePage) {
	drawText(gc, 24, 34, "Аппаратная инфраструктура", colTitle, 22)
	stats := fmt.Sprintf("Корзин: %d  •  Блэйдов: %d  •  Rack: %d  •  СХД: %d",
		hw.Stats.BladeChassis, hw.Stats.BladeServers, hw.Stats.RackServers, hw.Stats.StorageUnits)
	drawText(gc, 24, 56, stats, colMuted, 12)

	x := 20.0
	y := 72.0
	maxBottom := height - 24

	// Стойки с корзинами и rack-серверами.
	for _, rack := range hw.Racks {
		rw := 360.0
		rh := estimateRackHeight(rack)
		if y+rh > maxBottom {
			x += rw + 24
			y = 72
		}
		drawRackPanel(gc, x, y, rw, rh, rack)
		y += rh + 20
	}

	// СХД отдельным блоком справа или снизу.
	sx := width - 340
	if sx < x+40 {
		sx = x + 40
	}
	sy := 72.0
	drawStoragePanel(gc, sx, sy, 320, height-96, hw.Storage)
}

func estimateRackHeight(rack payload.Rack) float64 {
	h := 80.0
	for _, ch := range rack.BladeChassis {
		rows := (ch.SlotCount + 1) / 2
		slotH := 16.0
		h += 48 + float64(rows)*slotH + 18
	}
	h += float64(len(rack.RackServers)) * 34
	if h < 180 {
		h = 180
	}
	return h
}

func drawRackPanel(gc *draw2dimg.GraphicContext, x, y, w, h float64, rack payload.Rack) {
	drawPanel(gc, x, y, w, h, rack.Label)
	innerX := x + 14
	innerY := y + 38

	// Рамка стойки.
	fillRect(gc, innerX, innerY, w-28, h-52, color.RGBA{17, 24, 39, 255})
	strokeRect(gc, innerX, innerY, w-28, h-52, 1, colPanelEdge)
	drawText(gc, innerX+8, innerY+16, "42U", colMuted, 10)

	curY := innerY + 28
	for _, ch := range rack.BladeChassis {
		curY = drawBladeChassis(gc, innerX+10, curY, w-48, ch)
		curY += 12
	}
	for _, srv := range rack.RackServers {
		drawRackServerBar(gc, innerX+10, curY, w-48, srv)
		curY += 30
	}
}

func drawBladeChassis(gc *draw2dimg.GraphicContext, x, y, maxW float64, ch payload.BladeChassis) float64 {
	boxH := 44.0 + float64((ch.SlotCount+1)/2)*16
	fillRect(gc, x, y, maxW, boxH, color.RGBA{8, 47, 73, 255})
	strokeRect(gc, x, y, maxW, boxH, 1.2, colChassis)
	drawText(gc, x+8, y+16, ch.Name, colTitle, 11)
	sub := ch.Model
	if ch.SerialNum != "" {
		sub += " • " + ch.SerialNum
	}
	drawText(gc, x+8, y+30, sub, colMuted, 9)

	slotW := (maxW - 24) / 2
	slotH := 14.0
	sx := x + 8
	sy := y + 38
	for i, slot := range ch.Slots {
		col := slotCol(i)
		row := i / 2
		cx := sx + float64(col)*slotW
		cy := sy + float64(row)*slotH
		slotW2 := slotW - 6
		slotH2 := slotH - 2
		if slot.Occupied {
			c := colBladeOff
			if slot.PowerOn {
				c = colBladeOn
			}
			fillRect(gc, cx, cy, slotW2, slotH2, c)
			label := fmt.Sprintf("%d", slot.Num)
			if slot.ServerName != "" {
				if len(slot.ServerName) > 14 {
					label = fmt.Sprintf("%d %s…", slot.Num, slot.ServerName[:12])
				} else {
					label = fmt.Sprintf("%d %s", slot.Num, slot.ServerName)
				}
			}
			drawText(gc, cx+3, cy+10, label, color.RGBA{15, 23, 42, 255}, 8)
			pwr := "Выключен"
			if slot.PowerOn {
				pwr = "Включен"
			}
			addHit(cx, cy, slotW2, slotH2, slot.ServerName,
				fmt.Sprintf("Слот %d • %s", slot.Num, ch.Name),
				"Модель: "+slot.Model,
				"Питание: "+pwr,
			)
		} else {
			fillRect(gc, cx, cy, slotW2, slotH2, colEmpty)
			strokeRect(gc, cx, cy, slotW2, slotH2, 0.8, colMuted)
			drawText(gc, cx+3, cy+10, fmt.Sprintf("%d", slot.Num), colMuted, 8)
			addHit(cx, cy, slotW2, slotH2, fmt.Sprintf("Слот %d (пусто)", slot.Num),
				ch.Name,
				"Слот свободен",
			)
		}
	}
	addHit(x, y, maxW, 36, ch.Name, ch.Model, "S/N: "+ch.SerialNum,
		fmt.Sprintf("Слотов: %d", ch.SlotCount))
	return y + boxH
}

func slotCol(i int) int {
	return i % 2
}

func drawRackServerBar(gc *draw2dimg.GraphicContext, x, y, w float64, srv payload.RackServer) {
	const barH = 24.0
	fillRect(gc, x, y, w, barH, color.RGBA{49, 46, 129, 255})
	strokeRect(gc, x, y, w, barH, 1, colRackSrv)
	label := srv.Name
	if srv.Model != "" {
		label += " • " + srv.Model
	}
	drawText(gc, x+8, y+16, label, colText, 10)
	uLabel := fmt.Sprintf("U%d×%dU", srv.StartUnit, srv.SizeUnit)
	drawText(gc, x+w-72, y+16, uLabel, colMuted, 9)
	pwr := "Выключен"
	if srv.PowerOn {
		pwr = "Включен"
	}
	addHit(x, y, w, barH, srv.Name,
		"Модель: "+srv.Model,
		"S/N: "+srv.SerialNum,
		uLabel,
		"Питание: "+pwr,
	)
}

func drawStoragePanel(gc *draw2dimg.GraphicContext, x, y, w, h float64, items []payload.Storage) {
	drawPanel(gc, x, y, w, h, "Системы хранения (СХД)")
	curY := y + 42
	for _, st := range items {
		if curY > y+h-40 {
			break
		}
		boxH := 72.0
		fillRect(gc, x+12, curY, w-24, boxH, color.RGBA{67, 20, 7, 255})
		strokeRect(gc, x+12, curY, w-24, boxH, 1.2, colStorage)
		drawText(gc, x+20, curY+18, st.ShortName, colTitle, 12)
		meta := fmt.Sprintf("%s %s", st.Vendor, st.Model)
		drawText(gc, x+20, curY+34, meta, colMuted, 9)
		disks := fmt.Sprintf("Диски: %d", st.DiskCount)
		if st.DiskMax > 0 {
			disks = fmt.Sprintf("Диски: %d / %d", st.DiskCount, st.DiskMax)
		}
		size := fmt.Sprintf("Объём: %.1f TB", st.TotalSizeTB)
		drawText(gc, x+20, curY+50, disks+"  •  "+size, colText, 10)
		if st.UsedPercent > 0 {
			drawUsageBar(gc, x+20, curY+58, w-56, st.UsedPercent)
		}
		pwr := "Выключена"
		if st.PowerOn {
			pwr = "Включена"
		}
		loc := st.Room
		if st.RackNum != "" {
			loc += ", стойка " + st.RackNum
		}
		addHit(x+12, curY, w-24, boxH, st.ShortName,
			meta, disks, size,
			fmt.Sprintf("Утилизация: %.1f%%", st.UsedPercent),
			"Питание: "+pwr,
			"Размещение: "+loc,
		)
		curY += boxH + 10
	}
	if len(items) == 0 {
		drawText(gc, x+20, curY+10, "Нет активных СХД", colMuted, 11)
	}
}

func drawUsageBar(gc *draw2dimg.GraphicContext, x, y, w float64, pct float64) {
	fillRect(gc, x, y, w, 6, colEmpty)
	if pct > 100 {
		pct = 100
	}
	fillRect(gc, x, y, w*pct/100, 6, colStorage)
}

func drawVirtualizationPage(gc *draw2dimg.GraphicContext, width, height float64, vp payload.VirtualPage) {
	drawText(gc, 24, 40, "Виртуализация (VMware)", colTitle, 22)
	drawPanel(gc, 24, 70, width-48, 120, "Сводка")
	drawText(gc, 44, 118, fmt.Sprintf("Виртуальные машины: %d", vp.VMCount), colText, 14)
	drawText(gc, 44, 142, fmt.Sprintf("Гипервизоры: %d", vp.HypervisorCount), colText, 14)
	drawText(gc, 44, 166, fmt.Sprintf("Кластеры в справочнике: %d", vp.ClusterCount), colText, 14)

	drawPanel(gc, 24, 210, width-48, height-240, "Топ кластеров по числу ВМ")
	y := 250.0
	for _, c := range vp.TopClusters {
		fillRect(gc, 44, y, width-120, 28, color.RGBA{30, 64, 175, 255})
		drawText(gc, 54, y+18, c.Name, colText, 11)
		drawText(gc, width-100, y+18, fmt.Sprintf("%d ВМ", c.Count), colAccent, 11)
		y += 36
	}
	if len(vp.TopClusters) == 0 {
		drawText(gc, 44, 250, "Нет данных по кластерам", colMuted, 12)
	}
}

func drawMonitoringPage(gc *draw2dimg.GraphicContext, width, height float64, mp payload.MonitoringPage) {
	drawText(gc, 24, 40, "Мониторинг (Zabbix / СХД)", colTitle, 22)
	drawPanel(gc, 24, 70, (width-60)/2, 150, "Zabbix")
	drawText(gc, 44, 118, fmt.Sprintf("Хостов в справочнике: %d", mp.ZabbixHostCount), colText, 13)
	drawText(gc, 44, 142, fmt.Sprintf("Связано с server_spr_records: %d", mp.LinkedServers), colText, 13)

	drawPanel(gc, 36+(width-60)/2, 70, (width-60)/2, 150, "СХД")
	drawText(gc, 56+(width-60)/2, 118, fmt.Sprintf("В сети: %d", mp.StorageOnline), colBladeOn, 13)
	drawText(gc, 56+(width-60)/2, 142, fmt.Sprintf("Выключено: %d", mp.StorageOffline), colBladeOff, 13)
	drawText(gc, 56+(width-60)/2, 166, fmt.Sprintf("Средняя утилизация: %.1f%%", mp.AvgUsedPercent), colAccent, 13)

	drawPanel(gc, 24, 240, width-48, height-280, "Состояние")
	drawText(gc, 44, 280, "Страница мониторинга дополняется метриками Zabbix и событиями СХД.", colMuted, 12)
	drawText(gc, 44, 304, "Данные берутся из zabbix_host, storage_systems и server_spr_records.", colMuted, 12)
}
