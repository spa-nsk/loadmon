package main

import (
	"fmt"
	"image/color"

	"github.com/llgcode/draw2d/draw2dimg"
	"loadmon/infracanvas/payload"
)

func drawClusterMapPage(gc *draw2dimg.GraphicContext, width, height float64, cm payload.ClusterMapPage) {
	drawText(gc, 24, 34, "Карта кластеров vSphere", colTitle, 22)
	stats := fmt.Sprintf("vCenter: %d  •  Datacenter: %d  •  Кластеров: %d  •  Хостов: %d  •  ВМ: %d",
		cm.Stats.VcenterCount, cm.Stats.DatacenterCount, cm.Stats.ClusterCount,
		cm.Stats.HostCount, cm.Stats.VMCount)
	drawText(gc, 24, 56, stats, colMuted, 12)

	x := 20.0
	y := 72.0
	colW := 280.0
	maxBottom := height - 20

	for _, vc := range cm.Vcenters {
		for _, dc := range vc.Datacenters {
			boxH := estimateClusterGroupHeight(dc)
			if y+boxH > maxBottom {
				x += colW + 16
				y = 72
			}
			drawDatacenterClusterGroup(gc, x, y, colW, boxH, vc.Name, dc)
			y += boxH + 16
		}
	}
	if len(cm.Vcenters) == 0 {
		drawText(gc, 24, 90, "Нет активных кластеров в справочнике cluster_spr_records", colMuted, 12)
	}
}

func estimateClusterGroupHeight(dc payload.DatacenterGroup) float64 {
	h := 56.0 + float64(len(dc.Clusters))*36
	if h < 100 {
		h = 100
	}
	return h
}

func drawDatacenterClusterGroup(gc *draw2dimg.GraphicContext, x, y, w, h float64, vcenter string, dc payload.DatacenterGroup) {
	title := dc.Name
	if vcenter != "" {
		title = vcenter + " / " + dc.Name
	}
	drawPanel(gc, x, y, w, h, title)
	curY := y + 40
	for _, cl := range dc.Clusters {
		nodeH := 30.0
		fillRect(gc, x+10, curY, w-20, nodeH, color.RGBA{30, 58, 138, 255})
		strokeRect(gc, x+10, curY, w-20, nodeH, 1, colAccent)
		drawText(gc, x+18, curY+19, cl.Name, colTitle, 10)
		cnt := fmt.Sprintf("%d хост. • %d ВМ", cl.HostCount, cl.VMCount)
		drawText(gc, x+w-110, curY+19, cnt, colMuted, 9)

		twLabel := typeWorkLabel(cl.TypeWork)
		lines := []string{
			fmt.Sprintf("vCenter: %s", vcenter),
			fmt.Sprintf("Datacenter: %s", dc.Name),
			cnt,
		}
		if twLabel != "" {
			lines = append(lines, "Среда: "+twLabel)
		}
		addHit(x+10, curY, w-20, nodeH, cl.Name, lines...)

		curY += nodeH + 6
	}
}

func typeWorkLabel(t int) string {
	switch t {
	case 1:
		return "Продуктовая"
	case 2:
		return "Внутренний тест"
	case 3:
		return "Внешний тест"
	default:
		return ""
	}
}
