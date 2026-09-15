package controllers

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"loadmon/config"
	models "loadmon/models"
	pk "loadmon/utilities/pbkdf2"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/validation"
	beego "github.com/beego/beego/v2/server/web"

	"github.com/twinj/uuid"
)

var (
	guestID     int
	guestIDLock sync.Mutex
)

// Login отображает форму входа и обрабатывает аутентификацию пользователя
// (в т.ч. поддерживает предварительно выставленный заголовок X-Loadmon-Auth от SSO)
// с сохранением параметров пользователя и фильтра в сессии.
func (this *MainController) Login() {

	m := make(map[string]interface{})

	back := strings.Replace(this.Ctx.Input.Param(":back"), "_", "/", -1) // заменяет подстроку URL l1_l2_l3 на l1/l2/l3
	back = strings.Replace(back, "=", " ", -1)
	log.Println("back is", back)

	h := this.Ctx.ResponseWriter.Header()

	// Доменный пользователь прошёл Kerberos/SPNEGO-аутентификацию, но ни
	// одна из его групп AD не найдена в config.ConfigData.Auth (см.
	// appHandler в server_web_new/main_web.go и ResolveDomainPrivilege в
	// config/config.go) - вместо формы логина и вместо создания сессии
	// показываем страницу с предложением подать заявку на доступ.
	if h.Get("X-Loadmon-Access-Denied") == "1" {
		this.activeContent("user/accessdenied")
		this.Data["AccessRequestURL"] = config.ConfigData.AccessRequestURL
		return
	}

	xAuth, ok := h["X-Loadmon-Auth"]
	if ok {
		if xAuth[0] != "" {
			auth, err := base64.StdEncoding.DecodeString(xAuth[0])
			fmt.Println("X-Loadmon-Auth", string(auth))
			if err == nil {
				str := strings.Split((string)(auth), ":")
				if len(str) == 5 {
					m["timestamp"] = time.Now()
					m["SprSearchString"] = "" //строка фильтра справочника
					m["SprSearchField"] = "1" //тип строки для фильтра справочника
					m["username"] = str[1]
					m["first"] = str[0]
					userid, err := strconv.ParseInt(str[2], 10, 64)
					_ = err
					m["user_id"] = int64(userid)

					privilage, err := strconv.ParseInt(str[3], 10, 32)
					_ = err
					m["privilage"] = int(privilage)

					podrazdelenie, err := strconv.ParseInt(str[4], 10, 32)
					_ = err
					m["podrazdelenie"] = int(podrazdelenie)
					to := time.Now()
					to = CorrectTimeFilter(to)
					from := to.AddDate(0, 0, -7)
					from = CorrectTimeFilter(from)
					f := models.LoadFilter{FromTime: from, ToTime: to, Map: m}
					this.SetSession("loadmon", f)
					RegisterRequest(this, 1) //авторизация
					this.Redirect("/"+back, 302)
					return

				}
			}
		}
	}

	this.activeContent("user/login")

	if this.Ctx.Input.Method() == "POST" {
		flash := beego.NewFlash()
		email := this.GetString("email")
		password := this.GetString("password")
		domain := this.GetString("domain") //гостевой вход если в значении "on"
		valid := validation.Validation{}
		valid.Required(password, "password")
		if domain == "on" {
			guestIDLock.Lock()
			guestID++
			user := models.AuthUser{First: "Гость-" + strconv.Itoa(guestID), Last: this.Ctx.Request.RemoteAddr}
			guestIDLock.Unlock()
			m["first"] = user.First
			m["last"] = user.Last
			m["username"] = "Просмотр"
			m["user_id"] = int64(1000 + guestID)
			m["privilage"] = user.Priv     // 0 - обычный пользователь 1 - разрешено редактировать справочник
			m["podrazdelenie"] = user.Podr // 0 - не указано 1 - ДАТИ 2 - СТО
			m["timestamp"] = time.Now()
			m["SprSearchString"] = "" //строка фильтра справочника
			m["SprSearchField"] = "1" //тип строки для фильтра справочника
		} else {
			// Локальный вход по логину/паролю запрещён, если конфигурация
			// требует доменную авторизацию (см. RequireLocalAuth в
			// controllers/main_controller.go) - гостевой вход по чекбоксу
			// "domain" выше и SSO-заголовок X-Loadmon-Auth в начале функции
			// это отдельные, не завязанные на AuthMode механизмы и здесь не
			// проверяются.
			if !this.RequireLocalAuth() {
				return
			}

			valid.Email(email, "email")
			if valid.HasErrors() {
				errormap := make(map[string]string)
				for _, err := range valid.Errors {
					errormap[err.Key] = err.Message
				}
				this.Data["Errors"] = errormap
				return
			}
			//******** Read password hash from database
			var x pk.PasswordHash

			x.Hash = make([]byte, 32)
			x.Salt = make([]byte, 16)

			o := orm.NewOrm()
			//o.Using("default")
			user := models.AuthUser{Email: email}
			err := o.Read(&user, "Email")
			if err == nil {
				if strings.Trim(user.Reg_key, " ") != "" {
					flash.Error("Учетная запись не проверена")
					flash.Store(&this.Controller)
					return
				}

				// scan in the password hash/salt
				if x.Hash, err = hex.DecodeString(user.Password[:64]); err != nil {
					log.Println("Ошибка:", err)
				}
				if x.Salt, err = hex.DecodeString(user.Password[64:]); err != nil {
					log.Println("Ошибка:", err)
				}
			} else {
				flash.Error("Нет такого пользователя/email")
				flash.Store(&this.Controller)
				return
			}

			//******** Compare submitted password with database
			if !pk.MatchPassword(password, &x) {
				flash.Error("Ошибочный пароль")
				flash.Store(&this.Controller)
				return
			}

			//******** Create session and go back to previous page

			m["first"] = user.First
			m["last"] = user.Last
			m["username"] = email
			m["user_id"] = int64(user.Id)
			m["privilage"] = user.Priv     // 0 - обычный пользователь 1 - разрешено редактировать справочник 3 - администратор
			m["podrazdelenie"] = user.Podr // 0 - не указано 1 - ДАТИ  2 - СТО
			m["timestamp"] = time.Now()
			m["SprSearchString"] = "" //строка фильтра справочника
			m["SprSearchField"] = "1" //тип строки для фильтра справочника
		}
		to := time.Now()
		to = CorrectTimeFilter(to)
		from := to.AddDate(0, 0, -7)
		from = CorrectTimeFilter(from)

		f := models.LoadFilter{FromTime: from, ToTime: to, Map: m}
		this.SetSession("loadmon", f)
		RegisterRequest(this, 1) //авторизация
		//		if back == "view" {
		//			back = "view/filter"
		//		}
		this.Redirect("/"+back, 302)
		//this.Redirect(back, 302)

	}

}

// Logout завершает сеанс пользователя (удаляет сессию) и перенаправляет
// на страницу входа.
func (this *MainController) Logout() {
	//this.activeContent("logout") // logout.tpl отсутствует
	if this != nil {
		RegisterRequest(this, 2) // выход
	}
	this.DelSession("loadmon")
	this.Redirect("/user/login/filter", 302)
}

type user1 struct {
	First    string `form:"first"`
	Last     string `form:"last"`
	Email    string `form:"email"`
	Password string `form:"password"`
	Confirm  string `form:"password2"`
}

// Register отображает форму регистрации нового пользователя и обрабатывает
// её отправку: создаёт учётную запись и отправляет письмо с подтверждением.
func (this *MainController) Register() {
	this.activeContent("user/register")
	RegisterRequest(this, LogTypeRegister) // регистрация нового пользователя - до создания сессии запись не пишется, RegisterRequest тихо возвращает ошибку

	// Самостоятельная регистрация полностью заблокирована при доменной
	// авторизации (см. RequireLocalAuth в controllers/main_controller.go) -
	// учётные записи в этом режиме заводятся не через models.AuthUser, а
	// приходят из домена (Kerberos/SPNEGO). Проверка до POST, чтобы форма
	// регистрации не работала вовсе (не только сохранение), а не только
	// после отправки.
	if !this.RequireLocalAuth() {
		return
	}

	if this.Ctx.Input.Method() == "POST" {
		flash := beego.NewFlash()
		u := user1{}
		if err := this.ParseForm(&u); err != nil {
			flash.Error("Ошибка распознавания формы")
			flash.Store(&this.Controller)
			return
		}

		this.Data["User"] = u

		valid := validation.Validation{}
		valid.Required(u.First, "first")
		valid.Required(u.Last, "last")
		valid.Email(u.Email, "email")
		valid.Required(u.Password, "password")
		valid.Required(u.Confirm, "password2")
		valid.Range(len(u.Password), 6, 30, "password")
		valid.Range(len(u.Confirm), 6, 30, "password2")

		if valid.HasErrors() {
			errormap := make(map[string]string)
			for _, err := range valid.Errors {
				errormap[err.Key] = err.Message
			}
			this.Data["Errors"] = errormap
			return
		}

		/*
			if b, _ := valid.Valid(&u); !b {
				this.Data["Errors"] = valid.ErrorsMap
				return
			}
		*/
		if u.Password != u.Confirm {
			flash.Error("Пароли не совпадают")
			flash.Store(&this.Controller)
			return
		}
		h := pk.HashPassword(u.Password)

		//******** Save user info to database
		o := orm.NewOrm()
		//o.Using("default")

		user := models.AuthUser{First: u.First, Last: u.Last, Email: u.Email}

		// Convert password hash to string
		user.Password = hex.EncodeToString(h.Hash) + hex.EncodeToString(h.Salt)

		// Add user to database with new uuid and send verification email
		key := uuid.NewV4()
		user.Reg_key = key.String()
		_, err := o.Insert(&user)
		if err != nil {
			flash.Error(u.Email + " уже зарегистрирован")
			flash.Store(&this.Controller)
			return
		}

		domainname := this.Data["domainname"]
		//		if !sendVerification(u.Email, key.String(), domainname.(string)) {
		//			flash.Error("Ошибка отправки письма для проверки")
		//			flash.Store(&this.Controller)
		//			return
		//		}
		sendVerification(u.Email, key.String(), domainname.(string))
		SaveRequestVerification(u.Email, key.String(), domainname.(string))
		log.Println("Строка для активации пользователя:", domainname.(string)+"/user/verify/"+key.String())
		//		flash.Notice("Ваша учетная запись создана. Вы должны активировать её по ссылке в письме.")
		flash.Notice("Ваша учетная запись создана. Вам отправлена ссылка для активации.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}
}

// переделать функцию
func SaveRequestVerification(email, u string, domainname string) bool {
	f, err := os.OpenFile("./request.rec", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Println(err)
		return false
	}
	defer f.Close()

	if _, err = f.WriteString("<" + email + ">" + domainname + "/user/verify/" + u + "\n"); err != nil {
		log.Println(err)
		return false
	}
	return true
}

func sendVerification(email, u string, domainname string) error {
	link := domainname + "/user/verify/" + u
	sender := "noreply@bcs.ru"
	host := "nsk-smtp.global.bcs"
	contentType := "Content-Type: text/plain" + "; charset=UTF-8"
	subject := "Активация учетной записи (мониторинг загрузки серверов)"

	// Connect to the remote SMTP server.
	c, err := smtp.Dial(host + ":25")
	if err != nil {
		return err
	}

	// Set the sender and recipient first
	if err := c.Mail(sender); err != nil {
		return err
	}
	if err := c.Rcpt(email); err != nil {
		return err
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		return err
	}

	mailmsg := "To: " + email + "\r\nFrom: " + "norepy" + " <" + sender +
		">\r\nSubject: " + subject + "\r\n" + contentType + "\r\n\r\n" +
		"Для активации учетной записи перейдите по ссылке : " + link

	_, err = fmt.Fprintf(wc, mailmsg)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		return err
	}
	return nil
}

// Verify подтверждает регистрацию пользователя по ключу подтверждения
// (uuid) из ссылки в письме.
func (this *MainController) Verify() {
	this.activeContent("user/verify")
	RegisterRequest(this, LogTypeRegister) // подтверждение регистрации - без сессии запись не пишется

	// Подтверждение регистрации заблокировано вместе с самой регистрацией
	// при доменной авторизации (см. RequireLocalAuth в
	// controllers/main_controller.go и аналогичную проверку в Register выше) -
	// раз новые локальные учётные записи не создаются, подтверждать нечего.
	if !this.RequireLocalAuth() {
		return
	}

	u := this.Ctx.Input.Param(":uuid")
	o := orm.NewOrm()
	//o.Using("default")
	user := models.AuthUser{Reg_key: u}
	err := o.Read(&user, "Reg_key")
	if err == nil {
		this.Data["Verified"] = 1
		user.Reg_key = ""
		if _, err := o.Update(&user); err != nil {
			delete(this.Data, "Verified")
		}
	}
}

type user2 struct {
	First   string `form:"first"`
	Last    string `form:"last"`
	Email   string `form:"email"`
	Current string `form:"current"`
}

// Profile отображает и обрабатывает форму редактирования профиля
// текущего пользователя (имя, фамилия, email, смена пароля).
func (this *MainController) Profile() {
	this.activeContent("user/profile")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeServicePage) // просмотр страницы профиля - отдельная запись типа "обновление профиля" пишется ниже только при успешном сохранении

	this.Data["Filter"] = sess.(models.LoadFilter)

	p := this.Data["Filter"]

	filter := p.(models.LoadFilter)
	m := filter.Map
	//	m := sess.(map[string]interface{})
	if strings.Contains(m["first"].(string), "Гость") {
		this.Redirect("/", 302)
		return
	}
	flash := beego.NewFlash()

	//******** Read password hash from database
	var x pk.PasswordHash

	x.Hash = make([]byte, 32)
	x.Salt = make([]byte, 16)

	o := orm.NewOrm()
	//o.Using("default")
	user := models.AuthUser{Email: m["username"].(string)}
	err := o.Read(&user, "Email")
	if err == nil {
		// scan in the password hash/salt
		if x.Hash, err = hex.DecodeString(user.Password[:64]); err != nil {
			log.Println("Ошибка:", err)
		}
		if x.Salt, err = hex.DecodeString(user.Password[64:]); err != nil {
			log.Println("Ошибка:", err)
		}
	} else {
		flash.Error("Внутренняя ошибка")
		flash.Store(&this.Controller)
		return
	}

	if this.Ctx.Input.Method() == "POST" {
		u := user2{}
		if err := this.ParseForm(&u); err != nil {
			flash.Error("Ошибка распознавания формы")
			flash.Store(&this.Controller)
			return
		}
		this.Data["User"] = u

		valid := validation.Validation{}

		valid.Required(u.First, "first")
		valid.Required(u.Last, "last")
		valid.Email(u.Email, "email")
		valid.Required(u.Current, "current")

		password := this.GetString("password")
		password2 := this.GetString("password2")

		valid.Range(len(password), 6, 30, "password")
		valid.Range(len(password2), 6, 30, "password2")

		if valid.HasErrors() {
			errormap := make(map[string]string)
			for _, err := range valid.Errors {
				errormap[err.Key] = err.Message
			}
			this.Data["Errors"] = errormap
			return
		}

		if password != "" {
			if password != password2 {
				flash.Error("Пароли не совпадают")
				flash.Store(&this.Controller)
				return
			}
			h := pk.HashPassword(password)

			// Convert password hash to string
			user.Password = hex.EncodeToString(h.Hash) + hex.EncodeToString(h.Salt)
		}

		//******** Compare submitted password with database
		if !pk.MatchPassword(u.Current, &x) {
			flash.Error("Ошибочный текущий пароль")
			flash.Store(&this.Controller)
			return
		}

		//******** Save user info to database
		user.First = u.First
		user.Last = u.Last
		user.Email = u.Email

		_, err := o.Update(&user)
		if err == nil {
			flash.Notice("Профиль обновлен")
			flash.Store(&this.Controller)
			m["username"] = u.Email

			RegisterRequest(this, 4) //обновление профиля

		} else {
			flash.Error("Внутренняя ошибка")
			flash.Store(&this.Controller)
			return
		}
	} else {
		this.Data["User"] = user
	}
}

// Remove отображает и обрабатывает удаление учётной записи текущего пользователя.
func (this *MainController) Remove() {
	this.activeContent("user/remove")

	// Проверка сессии централизована в AuthFilter (см. controllers/auth_filter.go).
	sess := this.GetSession("loadmon")
	RegisterRequest(this, LogTypeAdminAction) // административная операция - удаление учётной записи

	this.Data["Filter"] = sess.(models.LoadFilter)

	p := this.Data["Filter"]

	filter := p.(models.LoadFilter)
	m := filter.Map
	// m := sess.(map[string]interface{})

	if this.Ctx.Input.Method() == "POST" {
		current := this.GetString("current")
		valid := validation.Validation{}
		valid.Required(current, "current")
		if valid.HasErrors() {
			errormap := make(map[string]string)
			for _, err := range valid.Errors {
				errormap[err.Key] = err.Message
			}
			this.Data["Errors"] = errormap
			return
		}

		flash := beego.NewFlash()

		//******** Read password hash from database
		var x pk.PasswordHash

		x.Hash = make([]byte, 32)
		x.Salt = make([]byte, 16)

		o := orm.NewOrm()
		//o.Using("default")
		user := models.AuthUser{Email: m["username"].(string)}
		err := o.Read(&user, "Email")
		if err == nil {
			// scan in the password hash/salt
			if x.Hash, err = hex.DecodeString(user.Password[:64]); err != nil {
				log.Println("Ошибка:", err)
			}
			if x.Salt, err = hex.DecodeString(user.Password[64:]); err != nil {
				log.Println("Ошибка:", err)
			}
		} else {
			flash.Error("Внутренняя ошибка")
			flash.Store(&this.Controller)
			return
		}

		//******** Compare submitted password with database
		if !pk.MatchPassword(current, &x) {
			flash.Error("Ошибочный текущий пароль")
			flash.Store(&this.Controller)
			return
		}

		//******** Delete user record
		_, err = o.Delete(&user)
		if err == nil {
			flash.Notice("Ваша учетная запись удалена.")
			flash.Store(&this.Controller)
			this.DelSession("loadmon")
			this.Redirect("/notice", 302)
		} else {
			flash.Error("Внутренняя ошибка")
			flash.Store(&this.Controller)
			return
		}
	}
}

// Forgot отображает форму восстановления пароля и отправляет пользователю
// письмо со ссылкой для сброса пароля.
func (this *MainController) Forgot() {
	this.activeContent("user/forgot")
	RegisterRequest(this, LogTypePasswordReset) // сброс/восстановление пароля - без сессии запись не пишется

	if this.Ctx.Input.Method() == "POST" {
		email := this.GetString("email")
		valid := validation.Validation{}
		valid.Email(email, "email")
		if valid.HasErrors() {
			errormap := make(map[string]string)
			for _, err := range valid.Errors {
				errormap[err.Key] = err.Message
			}
			this.Data["Errors"] = errormap
			return
		}

		flash := beego.NewFlash()

		o := orm.NewOrm()
		//o.Using("default")
		user := models.AuthUser{Email: email}
		err := o.Read(&user, "Email")
		if err != nil {
			flash.Error("Нет такого пользователя/email")
			flash.Store(&this.Controller)
			return
		}

		u := uuid.NewV4()
		user.Reset_key = u.String()
		_, err = o.Update(&user)
		if err != nil {
			flash.Error("Внутренняя ошибка")
			flash.Store(&this.Controller)
			return
		}

		domainname := this.Data["domainname"]
		sendRequestReset(email, u.String(), domainname.(string))
		log.Println("Ссылка на сброс пароля пользователя: ", "http://"+domainname.(string)+"/user/reset/"+u.String())
		flash.Notice("Вам отправлена ссылка для сброса пароля. Проверьте вашу почту.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}
}

// изменить функцию
func sendRequestReset(email, u string, domainname string) error {

	link := domainname + "/user/reset/" + u
	sender := "noreply@bcs.ru"
	host := "nsk-smtp.global.bcs"
	contentType := "Content-Type: text/plain" + "; charset=UTF-8"
	subject := "Сброс пароля учетной записи (мониторинг загрузки серверов)"

	// Connect to the remote SMTP server.
	c, err := smtp.Dial(host + ":25")
	if err != nil {
		return err
	}

	// Set the sender and recipient first
	if err := c.Mail(sender); err != nil {
		return err
	}
	if err := c.Rcpt(email); err != nil {
		return err
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		return err
	}

	mailmsg := "To: " + email + "\r\nFrom: " + "noreply" + "<" + sender + ">\r\nSubject: " + subject + "\r\n" + contentType + "\r\n\r\n" +
		"Для сброса пароля учетной записи перейдите по ссылке : " + link

	_, err = fmt.Fprintf(wc, mailmsg)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		return err
	}
	return nil
}

// Reset отображает форму сброса пароля по ссылке из письма (uuid)
// и обрабатывает установку нового пароля.
func (this *MainController) Reset() {
	this.activeContent("user/reset")
	RegisterRequest(this, LogTypePasswordReset) // сброс/восстановление пароля - без сессии запись не пишется

	flash := beego.NewFlash()

	u := this.Ctx.Input.Param(":uuid")
	o := orm.NewOrm()
	//o.Using("default")
	user := models.AuthUser{Reset_key: u}
	err := o.Read(&user, "Reset_key")
	if err == nil {
		if this.Ctx.Input.Method() == "POST" {
			password := this.GetString("password")
			password2 := this.GetString("password2")
			valid := validation.Validation{}
			valid.MinSize(password, 6, "password")
			valid.Required(password2, "password2")
			if valid.HasErrors() {
				errormap := make(map[string]string)
				for _, err := range valid.Errors {
					errormap[err.Key] = err.Message
				}
				this.Data["Errors"] = errormap
				return
			}

			if password != password2 {
				flash.Error("Пароли не совпадают")
				flash.Store(&this.Controller)
				return
			}
			h := pk.HashPassword(password)

			// Convert password hash to string
			user.Password = hex.EncodeToString(h.Hash) + hex.EncodeToString(h.Salt)
			user.Reg_key = "" //ставим татус проверен
			user.Reset_key = ""
			if _, err := o.Update(&user); err != nil {
				flash.Error("Внутренняя ошибка")
				flash.Store(&this.Controller)
				return
			}
			flash.Notice("Пароль обновлен.")
			flash.Store(&this.Controller)
			this.Redirect("/notice", 302)
		}
	} else {
		flash.Notice("Неправильный идентификатор пользователя.")
		flash.Store(&this.Controller)
		this.Redirect("/notice", 302)
	}
}
