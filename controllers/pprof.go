package controllers

import "net/http/pprof"

// Prof отдаёт индексную страницу профилировщика net/http/pprof.
func (this *MainController) Prof() {
	pprof.Index(this.Ctx.ResponseWriter, this.Ctx.Request)

	this.Ctx.ResponseWriter.WriteHeader(200)
}

// ProfCmdline отдаёт аргументы командной строки процесса (pprof/cmdline).
func (this *MainController) ProfCmdline() {
	pprof.Cmdline(this.Ctx.ResponseWriter, this.Ctx.Request)

	this.Ctx.ResponseWriter.WriteHeader(200)
}

// ProfProfile отдаёт CPU-профиль процесса (pprof/profile).
func (this *MainController) ProfProfile() {
	pprof.Profile(this.Ctx.ResponseWriter, this.Ctx.Request)

	this.Ctx.ResponseWriter.WriteHeader(200)
}

// ProfSymbol сопоставляет адреса программы с именами функций (pprof/symbol).
func (this *MainController) ProfSymbol() {
	pprof.Symbol(this.Ctx.ResponseWriter, this.Ctx.Request)
	this.Ctx.ResponseWriter.WriteHeader(200)
}

// ProfTrace отдаёт трассировку выполнения процесса (pprof/trace).
func (this *MainController) ProfTrace() {
	pprof.Trace(this.Ctx.ResponseWriter, this.Ctx.Request)
}
