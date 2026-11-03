package controllers

import (
	"loadmon/config"
)

// GetSSO — единственная точка Kerberos/SPNEGO при доменной авторизации.
// Вызывается с authgate (fetch credentials:same-origin) пока крутится «дождь».
// spnegoFilter (BeforeExec на /sso) выставляет X-Loadmon-Auth / Access-Denied;
// здесь создаём сессию и отвечаем JSON для клиента.
func (this *MainController) GetSSO() {
	if !config.ConfigData.IsDomainAuth() {
		this.Ctx.Output.SetStatus(404)
		this.Data["json"] = map[string]interface{}{"ok": false, "error": "not domain auth"}
		_ = this.ServeJSON()
		return
	}

	// Повторный /sso при уже живой сессии (повторный fetch / гонка) — успех без Negotiate.
	if this.GetSession("loadmon") != nil {
		this.Data["json"] = map[string]interface{}{"ok": true}
		_ = this.ServeJSON()
		return
	}

	if applied, denied := this.applyDomainAuthFromHeader(); denied {
		this.Ctx.Output.SetStatus(403)
		this.Data["json"] = map[string]interface{}{
			"ok":               false,
			"denied":           true,
			"accessRequestURL": config.ConfigData.AccessRequestURL,
		}
		_ = this.ServeJSON()
		return
	} else if applied {
		this.Data["json"] = map[string]interface{}{"ok": true}
		_ = this.ServeJSON()
		return
	}

	// Нет заголовков после SPNEGO — не должны сюда попадать при штатном
	// Negotiate (фильтр уже отдал 401 или Reject). На всякий случай.
	this.Ctx.Output.SetStatus(401)
	this.Ctx.Output.Header("WWW-Authenticate", "Negotiate")
	this.Data["json"] = map[string]interface{}{"ok": false, "error": "negotiate required"}
	_ = this.ServeJSON()
}
