module loadmon/infracanvas

go 1.25.0

require (
	github.com/llgcode/draw2d v0.0.0-20260422081035-c4331ac66734
	github.com/markfarnan/go-canvas v0.0.0-20200722235510-6971ccd00770
	loadmon/infracanvas/payload v0.0.0
)

require (
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	golang.org/x/image v0.36.0 // indirect
)

replace loadmon/infracanvas/payload => ./payload
