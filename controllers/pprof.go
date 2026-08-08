package controllers

import "net/http/pprof"

func (this *MainController) Prof() {
	pprof.Index(this.Ctx.ResponseWriter, this.Ctx.Request)

	this.Ctx.ResponseWriter.WriteHeader(200)
}

func (this *MainController) ProfCmdline() {
	pprof.Cmdline(this.Ctx.ResponseWriter, this.Ctx.Request)

	this.Ctx.ResponseWriter.WriteHeader(200)
}

func (this *MainController) ProfProfile() {
	pprof.Profile(this.Ctx.ResponseWriter, this.Ctx.Request)

	this.Ctx.ResponseWriter.WriteHeader(200)
}

func (this *MainController) ProfSymbol() {
	pprof.Symbol(this.Ctx.ResponseWriter, this.Ctx.Request)
	this.Ctx.ResponseWriter.WriteHeader(200)
}

func (this *MainController) ProfTrace() {
	pprof.Trace(this.Ctx.ResponseWriter, this.Ctx.Request)
}
