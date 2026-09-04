package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/markfarnan/go-canvas/canvas"
	"loadmon/infracanvas/payload"
)

type appState struct {
	data payload.CanvasData
	page int
}

var (
	state appState
	cvs   *canvas.Canvas2d
)

func main() {
	loadData()
	registerPageBridge()
	registerResizeBridge()
	registerHitTestBridge()
	registerClickBridge()

	doc := js.Global().Get("document")
	canvasEl := doc.Call("getElementById", "infra-canvas")
	if canvasEl.IsNull() || canvasEl.IsUndefined() {
		return
	}

	var err error
	cvs, err = canvas.NewCanvas2d(false)
	if err != nil {
		return
	}
	applyCanvasSize(canvasEl)
	state.page = 0
	cvs.Start(30, render)
	select {}
}

func applyCanvasSize(canvasEl js.Value) {
	w := canvasEl.Get("width").Int()
	h := canvasEl.Get("height").Int()
	if w <= 0 {
		w = 1280
	}
	if h <= 0 {
		h = 720
	}
	cvs.Set(canvasEl, w, h)
}

// loadData читает данные инфраструктуры из data-атрибута
// data-payload-b64 элемента #infra-canvas-data (см.
// server_web_new/views/admin/infrastructure.tpl) - значение приходит в
// base64, а не сырым JSON текстом внутри <script>, потому что html/template
// на Go-стороне (Beego) экранирует спецсимволы для HTML-контекста, а
// <script> - RAWTEXT-элемент, в котором браузер эти HTML-сущности не
// декодирует обратно; из-за этого раньше сюда приходил битый JSON (см.
// controllers/admininfrastructure.go). Ошибки декодирования/парсинга
// логируются в консоль браузера через console.error - раньше ошибка
// json.Unmarshal отбрасывалась молча (`_ = json.Unmarshal(...)`), из-за
// чего страница выглядела "рабочей", но без данных, без единой подсказки
// в консоли.
func loadData() {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", "infra-canvas-data")
	if el.IsNull() || el.IsUndefined() {
		return
	}
	b64 := el.Call("getAttribute", "data-payload-b64").String()
	if b64 == "" {
		return
	}
	raw := js.Global().Call("atob", b64).String()
	if raw == "" {
		return
	}
	if err := json.Unmarshal([]byte(raw), &state.data); err != nil {
		js.Global().Get("console").Call("error", fmt.Sprintf("infracanvas: ошибка разбора данных: %v", err))
	}
}

func registerPageBridge() {
	js.Global().Set("setInfraCanvasPage", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			state.page = args[0].Int()
		}
		return nil
	}))
}

func registerResizeBridge() {
	js.Global().Set("resizeInfraCanvas", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 2 || cvs == nil {
			return nil
		}
		doc := js.Global().Get("document")
		canvasEl := doc.Call("getElementById", "infra-canvas")
		if canvasEl.IsNull() {
			return nil
		}
		w := args[0].Int()
		h := args[1].Int()
		if w < 640 {
			w = 640
		}
		if h < 360 {
			h = 360
		}
		canvasEl.Set("width", w)
		canvasEl.Set("height", h)
		cvs.Set(canvasEl, w, h)
		return nil
	}))
}

func registerHitTestBridge() {
	js.Global().Set("infraCanvasHitTest", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 2 {
			return js.ValueOf(nil)
		}
		x := args[0].Float()
		y := args[1].Float()
		r := hitTest(x, y)
		if r == nil {
			return js.ValueOf(nil)
		}
		out, _ := json.Marshal(map[string]any{
			"title": r.Title,
			"lines": r.Lines,
		})
		return string(out)
	}))
}

func registerClickBridge() {
	doc := js.Global().Get("document")
	canvasEl := doc.Call("getElementById", "infra-canvas")
	if canvasEl.IsNull() {
		return
	}
	handler := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		ev := args[0]
		rect := canvasEl.Call("getBoundingClientRect")
		scaleX := float64(canvasEl.Get("width").Int()) / rect.Get("width").Float()
		scaleY := float64(canvasEl.Get("height").Int()) / rect.Get("height").Float()
		x := (ev.Get("clientX").Float() - rect.Get("left").Float()) * scaleX
		y := (ev.Get("clientY").Float() - rect.Get("top").Float()) * scaleY
		r := hitTest(x, y)
		showTooltip(ev, r)
		return nil
	})
	canvasEl.Call("addEventListener", "click", handler)
}

func showTooltip(ev js.Value, r *hitRegion) {
	doc := js.Global().Get("document")
	tip := doc.Call("getElementById", "infra-canvas-tooltip")
	if tip.IsNull() {
		return
	}
	if r == nil {
		tip.Get("classList").Call("add", "d-none")
		return
	}
	titleEl := tip.Call("querySelector", ".infra-tooltip-title")
	bodyEl := tip.Call("querySelector", ".infra-tooltip-body")
	if !titleEl.IsNull() {
		titleEl.Set("textContent", r.Title)
	}
	if !bodyEl.IsNull() {
		bodyEl.Set("innerHTML", "")
		for _, line := range r.Lines {
			if line == "" {
				continue
			}
			p := doc.Call("createElement", "div")
			p.Set("textContent", line)
			bodyEl.Call("appendChild", p)
		}
	}
	viewport := doc.Call("querySelector", ".infra-canvas-viewport")
	vRect := viewport.Call("getBoundingClientRect")
	left := ev.Get("clientX").Float() - vRect.Get("left").Float() + 12
	top := ev.Get("clientY").Float() - vRect.Get("top").Float() + 12
	tip.Set("style", "left:"+trimFloat(left)+"px;top:"+trimFloat(top)+"px;")
	tip.Get("classList").Call("remove", "d-none")
}

func trimFloat(v float64) string {
	return fmt.Sprintf("%.0f", v)
}

func render(gc *draw2dimg.GraphicContext) bool {
	clearHits()
	drawFrame(gc, cvs.Width(), cvs.Height())
	return true
}

func drawFrame(gc *draw2dimg.GraphicContext, width, height int) {
	clearBackground(gc, float64(width), float64(height))
	switch state.page {
	case 1:
		drawVirtualizationPage(gc, float64(width), float64(height), state.data.Virtualization)
	case 2:
		drawMonitoringPage(gc, float64(width), float64(height), state.data.Monitoring)
	case 3:
		drawClusterMapPage(gc, float64(width), float64(height), state.data.ClusterMap)
	default:
		drawHardwarePage(gc, float64(width), float64(height), state.data.Hardware)
	}
}
