package main

type hitRegion struct {
	X, Y, W, H float64
	Title      string
	Lines      []string
}

var hitRegions []hitRegion

func clearHits() {
	hitRegions = hitRegions[:0]
}

func addHit(x, y, w, h float64, title string, lines ...string) {
	if w <= 0 || h <= 0 {
		return
	}
	hitRegions = append(hitRegions, hitRegion{
		X: x, Y: y, W: w, H: h,
		Title: title,
		Lines: lines,
	})
}

func hitTest(x, y float64) *hitRegion {
	for i := len(hitRegions) - 1; i >= 0; i-- {
		r := &hitRegions[i]
		if x >= r.X && x <= r.X+r.W && y >= r.Y && y <= r.Y+r.H {
			return r
		}
	}
	return nil
}
