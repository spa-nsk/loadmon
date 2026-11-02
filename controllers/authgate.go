package controllers

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/models"
)

// authGateMinMs - минимальная длительность экрана «цифрового дождя» при входе.
const authGateMinMs = 15000

// showAuthGate показывает полноэкранную визуализацию входа (Matrix rain)
// и через AuthGateMinMs перенаправляет браузер на target (клиентский JS).
// Вызывается сразу после успешного создания сессии, до перехода в приложение.
func (this *MainController) showAuthGate(target string) {
	target = strings.TrimSpace(target)
	if target == "" || target == "/" {
		target = "/sprsrv/view2/all/0"
	}
	if !strings.HasPrefix(target, "/") {
		target = "/" + target
	}
	// не уводить обратно на экраны входа - защита от зацикливания
	if strings.HasPrefix(target, "/user/login") || target == "/login" {
		target = "/sprsrv/view2/all/0"
	}
	this.Layout = ""
	this.TplName = "user/authgate.tpl"
	this.Data["AuthGateTarget"] = target
	this.Data["AuthGateMinMs"] = authGateMinMs
	this.Data["StaticVer"] = staticAssetVersion
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
// (сегменты через "_") или query next.
func authGateTargetFromBack(back string) string {
	back = strings.ReplaceAll(back, "_", "/")
	back = strings.ReplaceAll(back, "=", " ")
	back = strings.TrimSpace(back)
	if back == "" || back == "filter" {
		return "/sprsrv/view2/all/0"
	}
	if !strings.HasPrefix(back, "/") {
		back = "/" + back
	}
	return back
}

// LoginBlockedRedirect - при доменной авторизации форма /user/login и /login
// не используются: редирект на "/" (там SPNEGO + экран входа). Возвращает
// true, если редирект уже выполнен.
func (this *MainController) LoginBlockedRedirect() bool {
	if config.ConfigData.IsLocalAuth() {
		return false
	}
	this.Redirect("/", 302)
	return true
}
