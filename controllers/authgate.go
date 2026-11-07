package controllers

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/models"
)

// authGateMinMs — минимум показа дождя от старта SSO (перекрывает авторизацию,
// не добавляется после уже успешного входа).
const authGateMinMs = 4000

// defaultAuthReturnURL - куда вести после входа, если цель не задана или небезопасна.
const defaultAuthReturnURL = "/sprsrv/view2/all/0"

// sanitizeReturnURL - только относительный путь приложения (защита от open redirect
// и от «кривых» next вроде //evil или /user/login/...).
func sanitizeReturnURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "/" || raw == "/login" {
		return defaultAuthReturnURL
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return defaultAuthReturnURL
	}
	if strings.HasPrefix(raw, "/user/login") {
		return defaultAuthReturnURL
	}
	return raw
}

// requestReturnURL собирает path[+?query] исходного запроса для возврата после входа.
func requestReturnURL(path, rawQuery string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if rawQuery != "" {
		return path + "?" + rawQuery
	}
	return path
}

// showAuthGate показывает полноэкранную визуализацию входа (Matrix rain).
// При доменной авторизации страница сразу стартует fetch(/sso): дождь идёт
// параллельно Negotiate, переход на target — когда SSO готов и прошло ≥authGateMinMs
// с момента открытия (не дополнительная пауза после входа).
// При локальном входе сессия уже создана — сразу уходим на target (minMs=0).
func (this *MainController) showAuthGate(target string) {
	target = sanitizeReturnURL(target)
	this.Layout = ""
	this.TplName = "user/authgate.tpl"
	this.Data["AuthGateTarget"] = target
	this.Data["StaticVer"] = currentStaticVersion()
	domain := config.ConfigData.IsDomainAuth()
	this.Data["AuthGateSSO"] = domain
	if domain {
		this.Data["AuthGateMinMs"] = authGateMinMs
	} else {
		this.Data["AuthGateMinMs"] = 0
	}
}

// applyDomainAuthFromHeader читает X-Loadmon-Auth / X-Loadmon-Access-Denied,
// выставленные spnegoFilter/appHandler (server_web_new/main_web.go), и при
// успехе создаёт сессию loadmon. Возвращает:
//
//	applied=true  - сессия создана
//	denied=true   - доменный пользователь без прав (нужна страница accessdenied)
//	оба false     - заголовков нет, сессию не трогали
func (this *MainController) applyDomainAuthFromHeader() (applied, denied bool) {
	h := this.Ctx.ResponseWriter.Header()
	if h.Get("X-Loadmon-Access-Denied") == "1" {
		return false, true
	}
	xAuth, ok := h["X-Loadmon-Auth"]
	if !ok || len(xAuth) == 0 || xAuth[0] == "" {
		return false, false
	}
	auth, err := base64.StdEncoding.DecodeString(xAuth[0])
	if err != nil {
		return false, false
	}
	str := strings.Split(string(auth), ":")
	if len(str) != 5 {
		return false, false
	}
	m := make(map[string]interface{})
	m["timestamp"] = time.Now()
	m["SprSearchString"] = ""
	m["SprSearchField"] = "1"
	m["username"] = str[1]
	m["first"] = str[0]
	userid, _ := strconv.ParseInt(str[2], 10, 64)
	m["user_id"] = userid
	privilage, _ := strconv.ParseInt(str[3], 10, 32)
	m["privilage"] = int(privilage)
	podrazdelenie, _ := strconv.ParseInt(str[4], 10, 32)
	m["podrazdelenie"] = int(podrazdelenie)
	to := CorrectTimeFilter(time.Now())
	from := CorrectTimeFilter(to.AddDate(0, 0, -7))
	this.SetSession("loadmon", models.LoadFilter{FromTime: from, ToTime: to, Map: m})
	RegisterRequest(this, 1)
	fmt.Println("X-Loadmon-Auth applied for", str[1])
	return true, false
}

// authGateTargetFromBack собирает целевой URL из параметра :back логина
// (сегменты через "_" — устаревшая схема; предпочтителен query next).
func authGateTargetFromBack(back string) string {
	back = strings.ReplaceAll(back, "_", "/")
	back = strings.ReplaceAll(back, "=", " ")
	back = strings.TrimSpace(back)
	if back == "" || back == "filter" || back == "back" {
		return defaultAuthReturnURL
	}
	if !strings.HasPrefix(back, "/") {
		back = "/" + back
	}
	return sanitizeReturnURL(back)
}

// resolveLoginReturnURL - цель после входа: ?next=... (предпочтительно) или :back.
func (this *MainController) resolveLoginReturnURL() string {
	if next := strings.TrimSpace(this.GetString("next")); next != "" {
		return sanitizeReturnURL(next)
	}
	return authGateTargetFromBack(this.Ctx.Input.Param(":back"))
}

// LoginBlockedRedirect - при доменной авторизации форма /user/login и /login
// не используются: редирект на "/?next=..." (SPNEGO + экран входа), с сохранением
// исходной цели. Возвращает true, если редирект уже выполнен.
func (this *MainController) LoginBlockedRedirect() bool {
	if config.ConfigData.IsLocalAuth() {
		return false
	}
	target := this.resolveLoginReturnURL()
	this.Redirect("/?next="+url.QueryEscape(target), 302)
	return true
}
