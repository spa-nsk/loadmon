package main

import (
	"encoding/json"
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

	doc := js.Global().Get("document")
	canvasEl := doc.Call("getElementById", "infra-canvas")
	if canvasEl.IsNull() || canvasEl.IsUndefined() {
		return
	}
	width := canvasEl.Get("width").Int()
	height := canvasEl.Get("height").Int()
	if width <= 0 {
		width = 1280
	}
	if height <= 0 {
		height = 760
	}

	var err error
	cvs, err = canvas.NewCanvas2d(false)
	if err != nil {
		return
	}
	cvs.Set(canvasEl, width, height)
	state.page = 0
	cvs.Start(30, render)
	select {}
}

func loadData() {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", "infra-canvas-data")
	if el.IsNull() || el.IsUndefined() {
		return
	}
	raw := el.Get("textContent").String()
	if raw == "" {
		return
	}
	_ = json.Unmarshal([]byte(raw), &state.data)
}

func registerPageBridge() {
	js.Global().Set("setInfraCanvasPage", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			state.page = args[0].Int()
		}
		return nil
	}))
}

func render(gc *draw2dimg.GraphicContext) bool {
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
	default:
		drawHardwarePage(gc, float64(width), float64(height), state.data.Hardware)
	}
}
