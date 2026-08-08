package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	//"loadmon/cmdb"
	"loadmon/config"
	"loadmon/itsystem"
	"loadmon/models"
	_ "loadmon/routers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"runtime"
	"sync"
	"syscall"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
	begocont "github.com/beego/beego/v2/server/web/context"

	// "github.com/beego/beego/v2/server/web/context"
	"github.com/gorilla/sessions"
	"github.com/jcmturner/goidentity/v6"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/jcmturner/gokrb5/v8/service"
	"github.com/jcmturner/gokrb5/v8/spnego"

	_ "github.com/lib/pq"
	// sqlite3-драйвер регистрируется в пакете config (config/dbinit.go)
)

// var password string = ""
var baseLock sync.RWMutex
var baseLocked bool = false
var kt *keytab.Keytab

func IsBaseLocked() bool {
	baseLock.RLock()
	defer baseLock.RUnlock()
	return baseLocked
}

func LockBase() {
	baseLock.Lock()
	defer baseLock.Unlock()
	baseLocked = true
}

func UnlockBase() {
	baseLock.Lock()
	defer baseLock.Unlock()
	baseLocked = false
}

type SessionMgr struct {
	store      sessions.Store
	cookieName string
}

func init() {
	config.ConfigData.LoadConfigData()
	config.ConfigData.InitParams()

	// Режим авторизации задаётся параметром AuthMode в conf/app2.json:
	// "domain" - Kerberos/SPNEGO против Active Directory (поведение по умолчанию, в т.ч. если параметр не указан)
	// "local"  - логин/пароль по локальной таблице models.AuthUser (см. controllers/user2.go), без Kerberos
	if config.ConfigData.IsDomainAuth() {
		// 1. Получение пути к keytab из переменной окружения или использование дефолтного
		// ВАЖНО: В продакшене не используйте хардкод путей. Используйте секреты или переменные окружения.
		ktPath := os.Getenv("KEYTAB_PATH")
		if ktPath == "" {
			// Для примера используем тестовый путь, но в реальности укажите реальный путь
			// Например: "C:/win/keytabs/server.keytab" или "/etc/krb5.keytab"
			ktPath = "./svc-loadmon-krb-01.keytab"
		}

		// Загрузка Keytab
		var err error
		kt, err = keytab.Load(ktPath)
		if err != nil {
			log.Fatalf("Ошибка загрузки keytab: %v. Проверьте путь и права доступа. Если Kerberos-авторизация не нужна, укажите \"AuthMode\": \"local\" в conf/app2.json.", err)
		}

		// Настройка логгера Beego
		logs.SetLogger(logs.AdapterFile, `{"filename": "beego_krb5.log", "level": 7, "maxlines": 0}`)
		//logs.SetDefaultLogger(logs.NewLogger(1000))

		// Регистрация фильтров Beego
		// Мы создаем фильтр, который обрабатывает SPNEGO аутентификацию
		//web.Filter("krb5_auth", spnegoFilter)
		beego.InsertFilter("/*", beego.BeforeExec, spnegoFilter)
		// Регистрация контроллера
		//web.Router("/", &AuthController{})
		log.Println("Авторизация: режим domain (Kerberos/SPNEGO)")
	} else {
		log.Println("Авторизация: режим local (логин/пароль)")
	}
}

// spnegoFilter - Middleware для обработки аутентификации
func spnegoFilter(ctx *begocont.Context) {

	session, err := ctx.Session() //("loadmon", f) это работало
	if err == nil {
		s := session.Get(context.Background(), "loadmon")
		if s != nil {
			return
		}
	}

	authHeader := ctx.Input.Header("Authorization")
	if authHeader == "" {
		ctx.Output.Header("WWW-Authenticate", "Negotiate")
		ctx.ResponseWriter.WriteHeader(401)
		ctx.ResponseWriter.Write([]byte("401 Unauthorized\n"))
		return
	}
	//fmt.Println("Authorization: ", authHeader)

	// Создаем логгер для service (используем стандартный Go log или beego logger)

	//l := log.New(ctx.ResponseWriter, "GOKRB5 Service: ", log.Ldate|log.Ltime|log.Lshortfile)

	l := log.New(os.Stdout, "GOKRB5 Service: ", log.Ldate|log.Ltime|log.Lshortfile)
	// Создаем объект service с логгером
	// Важно: мы передаем *kt, но spnego требует *keytab.Keytab
	// В данном случае мы передаем указатель на глобальную переменную

	// Создаем прокси для http.ResponseWriter и Request
	// Beego context уже содержит оригинальные http.ResponseWriter и Request
	// Нам нужно передать их в spnego

	// Простой способ: создать обертку, так как spnego ожидает стандартные http handlers
	// Но так как мы внутри фильтра, мы можем вызвать spnego напрямую, передав контекст

	// SPNEGO требует http.Handler. Мы создадим временный handler, который вызывает appHandler
	// Но так как мы внутри фильтра, мы можем просто передать вызов дальше, если аутентификация прошла,
	// или завершить ответ, если нет.

	// Лучший подход для Beego: использовать spnego как middleware, который выполняет аутентификацию
	// и сохраняет данные в контекст.

	// Создаем тестовый request/response для вызова SPNEGO, если он не работает с фильтром напрямую.
	// Однако, spnego.SPNEGOKRB5Authenticate возвращает http.Handler.
	// Мы можем использовать его как фильтр.

	// Создаем "пустой" handler для передачи в spnego, который просто вызовет appHandler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appHandler(w, r)
	})

	// Обертываем в SPNEGO
	snegoHandler := spnego.SPNEGOKRB5Authenticate(handler, kt, service.Logger(l))

	// Вызываем обработчик SPNEGO
	snegoHandler.ServeHTTP(ctx.ResponseWriter, ctx.Request)

	// После вызова SPNEGO, если аутентификация прошла успешно, SPNEGO уже отправил ответ или установил контекст.
	// Если аутентификация прошла и был вызван appHandler, то он сам отправит ответ.
	// Если аутентификация не прошла, SPNEGO вернет 401.
	// В этом случае мы не должны вызывать ctx.Next(), так как ответ уже отправлен.

	// Важно: Если SPNEGO вернул 401, он завершил запрос.
	// Если он вернул 200 и вызвал appHandler, то appHandler сам завершил запрос.
	// В Beego v2 Filter API, если вы не вызываете ctx.Next(), запрос считается обработанным.
	// SPNEGO делает это сам, поэтому мы просто возвращаемся.
	//key := "github.com/jcmturner/gokrb5/v8/sessionCredentials"
	//cr, err := sessionMgr.Get(ctx.Request, key)

}

// AppHandler - основная логика отображения информации о пользователе
func appHandler(w http.ResponseWriter, r *http.Request) {

	creds := goidentity.FromHTTPRequestContext(r)

	if creds != nil && creds.Authenticated() {
		m := make(map[string]interface{})

		m["timestamp"] = time.Now()
		m["SprSearchString"] = "" //строка фильтра справочника
		m["SprSearchField"] = "1" //тип строки для фильтра справочника
		email := fmt.Sprintf("%s@%s", creds.UserName(), creds.Domain())
		m["username"] = email
		m["first"] = creds.DisplayName()

		o := orm.NewOrm()
		if ADCredsJSON, ok := creds.Attributes()[credentials.AttributeKeyADCredentials]; ok {
			ADCreds, ok := ADCredsJSON.(credentials.ADCredentials)
			if ok {
				name := strings.SplitN(ADCreds.FullName, " ", 3)
				m["first"] = name[0]
				m["last"] = name[1]
				email := fmt.Sprintf("%s@%s", ADCreds.EffectiveName, ADCreds.LogonDomainName)
				m["username"] = email
				users := make([]models.AuthUser, 0)
				qs := o.QueryTable("users").OrderBy("email").Filter("email__icontains", ADCreds.EffectiveName)
				fmt.Println("Поиск пользователя: ", ADCreds.EffectiveName, " в базе пользователей")
				qs.All(&users)
				if len(users) == 1 {
					fmt.Println("В локальной базе пользователей найден: ", users[0].Email, name)
					m["user_id"] = int64(users[0].Id)
					m["privilage"] = users[0].Priv     // 0 - обычный пользователь 1 - разрешено редактировать справочник
					m["podrazdelenie"] = users[0].Podr // 0 - не указано 1 - ДАТИ 2 - СТО
				} else {
					fmt.Println("В локальной базе пользователей не найден: ", ADCreds.EffectiveName)
					m["user_id"] = 0
					m["privilage"] = 0
					m["podrazdelenie"] = 0
				}

				auth := fmt.Sprintf("%s:%s:%d:%d:%d", ADCreds.FullName, ADCreds.EffectiveName, m["user_id"], m["privilage"], m["podrazdelenie"])
				token := base64.StdEncoding.EncodeToString([]byte(auth))
				w.Header().Set("X-Loadmon-Auth", token)
			}
		}
	}
}

func NewSessionMgr(cookieName string) SessionMgr {

	skey := []byte("loadmon")

	return SessionMgr{
		store:      sessions.NewCookieStore(skey),
		cookieName: cookieName,
	}
}

func (smgr SessionMgr) Get(r *http.Request, k string) ([]byte, error) {
	//func (smgr SessionMgr) Get(r *http.Request, k string) (any, error) {
	s, err := smgr.store.Get(r, smgr.cookieName)
	if err != nil {
		return nil, err
	}

	if s == nil {
		return nil, errors.New("nil session")
	}

	b, ok := s.Values[k].([]byte)
	//b, ok := s.Values[k].(any)
	if !ok {
		return nil, fmt.Errorf("could not get bytes held in session at key %s", k)
	}

	return b, nil
}

func (smgr SessionMgr) New(w http.ResponseWriter, r *http.Request, k string, v []byte) error {
	//func (smgr SessionMgr) New(w http.ResponseWriter, r *http.Request, k string, v any) error {
	s, err := smgr.store.New(r, smgr.cookieName)
	if err != nil {
		return fmt.Errorf("could not create new session: %v", err)
	}

	s.Values[k] = v

	return s.Save(r, w)

}

func main() {

	//beego.InsertFilter("*", beego.BeforeRouter, KerberosFilter)

	config.DBInit()
	/*
		if len(os.Args) > 1 {
			if os.Args[1] == "convert" {
				fmt.Println("Конвертация данных начата")
				cmdb.ConvertDatabase("default", "default-add")
				fmt.Println("Конвертация данных завершена")
			}
		}
	*/
	httpKey := config.ConfigData.HttpKey
	httpsKey := config.ConfigData.HttpsKey
	httpEnv := os.Getenv(httpKey)
	defer os.Setenv(httpKey, httpEnv)

	httpsEnv := os.Getenv(httpsKey)
	defer os.Setenv(httpsKey, httpsEnv)

	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	s1 := os.Getenv(httpKey)
	s2 := os.Getenv(httpsKey)

	_ = s1
	_ = s2

	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("Используются : ", runtime.NumCPU(), " CPU")

	go itsystem.LoadITSystemSpr()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			<-sigs
			log.Println("Получен сигнал на завершение loadmon")
			if err := beego.BeeApp.Server.Shutdown(context.Background()); err != nil {
				// Error from closing listeners, or context timeout:
				log.Printf("Ошибка выключения сервера HTTP: %v", err)
			}
		}
	}()

	if config.ConfigData.WorkMode == "web" {
		log.Println("Подключение к API по адресу:", config.ConfigData.GRPCCliApiAddr)
		log.Println("Подключение к DB по адресу:", config.ConfigData.HostName, ":", config.ConfigData.Port, config.ConfigData.DriverName, config.ConfigData.DBName, config.ConfigData.UserName)
		if beego.BConfig.RunMode == "dev" {
			beego.BConfig.WebConfig.DirectoryIndex = true
			beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
		}

		beego.Run()
	} else {
		fmt.Println("Попытка запуска web сервиса из папки с конфигурацией: ", config.ConfigData.WorkMode)
	}
}

type PageData struct {
	Title  string
	User   string
	Domain string
	Groups []string
	AD     *credentials.ADCredentials
}

// Фильтр-функция для проверки Kerberos-токена
var KerberosFilter = func(ctx *begocont.Context) {
	// Здесь инициализируется контекст клиента/сервера gokrb5,
	// разбираются заголовки запроса и проверяется подлинность.
	// Примерная логика: извлечь заголовок Authorization,
	// обработать SPNEGO токен.

	authHeader := ctx.Input.Header("Authorization")
	if authHeader == "" {
		ctx.Output.Header("WWW-Authenticate", "Negotiate")
		ctx.ResponseWriter.WriteHeader(401)
		ctx.ResponseWriter.Write([]byte("401 Unauthorized\n"))
		return
	}

	fmt.Println("Получен токен")
	s := strings.SplitN(ctx.Input.Header("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != "Negotiate" {
		//ошибка не тот запрос
		ctx.Output.Header("WWW-Authenticate", "Negotiate")
		ctx.ResponseWriter.WriteHeader(401)
		ctx.ResponseWriter.Write([]byte("401 Unauthorized\n"))
		return
	}

	fmt.Println("Token: ", s[1])
	tokenBytes, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		fmt.Println("Ошибка преобразования из формата base64")
		return
	}
	_ = tokenBytes
	var token spnego.SPNEGOToken
	err = token.Unmarshal(tokenBytes)
	if err != nil {
		fmt.Println("Ошибка Unmarshal")
		return
	}
	ok, t, err := spnego.UnmarshalNegToken(tokenBytes)
	if ok {
		cc, ok2 := ((t).(spnego.NegTokenInit))
		if ok2 {
			_ = cc
		}
	}

	i := token.Init
	r := token.Resp
	c := token.NegTokenInit.Context()

	_ = i
	_ = r
	_ = c
	if c != nil {
		if id, ok := c.Value(goidentity.CTXKey).(goidentity.Identity); ok {
			d := PageData{
				Title:  "Kerberos SSO App",
				User:   id.UserName(),
				Domain: id.Domain(),
				Groups: id.AuthzAttributes(),
				AD:     getADCredentials(id),
			}
			_ = d
		}
	}
	id := goidentity.FromHTTPRequestContext(ctx.Request)
	if id != nil {
		data := PageData{
			Title:  "Kerberos SSO App",
			User:   id.UserName(),
			Domain: id.Domain(),
			Groups: id.AuthzAttributes(),
			AD:     getADCredentials(id),
		}
		_ = data
	}
	//spnegoToken, err := spnego.ParseToken(tokenBytes)
	//if err != nil {
	//	fmt.Errorf("ошибка парсинга SPNEGO: %w", err)
	//	return
	//}

	//fmt.Printf("SPNEGO Mechanism: %s\n", spnegoToken.MechType.String())

	// Проверяем, что это Kerberos (OID 1.2.840.113554.1.2.2)
	//if !spnegoToken.MechType.Equal(types.K) { //KRB5_MechType
	//	fmt.Errorf("ошибка: механизм не Kerberos. Получен: %s", spnegoToken.MechType.String())
	//	return
	//}

	// Обработка SPNEGO token через gokrb5 ...
	//Декодирование Base64: Полученная строка превращается в бинарные данные.
	//Распаковка SPNEGO (RFC 4178): Бинарные данные - это контейнер SPNEGO. Он содержит заголовок и, собственно, сам протокол аутентификации (Kerberos V5 или NTLM).
	//Распаковка GSSAPI (RFC 2743 / RFC 1964): Внутри SPNEGO находится GSSAPI токен (в данном случае Kerberos AP-REQ).
	//Распаковка Kerberos (RFC 4120): Из GSSAPI извлекается сам AP-REQ (Application Protocol Request).

}

func getADCredentials(id goidentity.Identity) *credentials.ADCredentials {
	if id == nil {
		return nil
	}

	v, ok := id.Attributes()[credentials.AttributeKeyADCredentials]
	if !ok {
		return nil
	}

	ad, ok := v.(credentials.ADCredentials)
	if !ok {
		return nil
	}

	return &ad
}
