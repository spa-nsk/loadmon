// Command itiliumgetcu — прототип синхронизации CMDB из 1С:ITILIUM getCU.
// За один запуск скачивает и импортирует все поддерживаемые cuClass
// (порядок: корзины → блейды → rack → СХД → ленты → сеть).
//
//	go run ./tools/itiliumgetcu
//
// Параметры подключения берутся из app2.json (ConfigItilium); флаги
// переопределяют отдельные поля. Пароль: -password, ITILIUM_PASSWORD,
// ConfigItilium.Password или GetKeyData.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	"loadmon/config"
	"loadmon/itilium"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Порядок важен: корзины (159) до блейдов (142).
var syncCUClasses = []int{
	itilium.CUClassBladeEnclosures, // 159
	itilium.CUClassBlades,          // 142
	itilium.CUClassServers,         // 145
	itilium.CUClassStorage,         // 158
	itilium.CUClassTapeLibraries,   // 198
	itilium.CUClassNetwork,         // 146
}

func main() {
	// Временно отключаем системный http(s)_proxy (часто ломает NTLM к 1С).
	httpKey, httpsKey := "http_proxy", "https_proxy"
	httpEnv, httpsEnv := os.Getenv(httpKey), os.Getenv(httpsKey)
	defer os.Setenv(httpKey, httpEnv)
	defer os.Setenv(httpsKey, httpsEnv)
	os.Unsetenv(httpKey)
	os.Unsetenv(httpsKey)

	base := flag.String("base", "", "корень публикации 1С (пусто — ConfigItilium)")
	domain := flag.String("domain", "", "AD-домен")
	user := flag.String("user", "", "логин без домена")
	password := flag.String("password", "", "пароль")
	auth := flag.String("auth", "", "ntlm или basic")
	apiPrefix := flag.String("api-prefix", "", "префикс HTTP-сервиса")
	cuStatus := flag.String("cu-status", "", "cuStatus getCU (пусто — all / ConfigItilium)")
	insecure := flag.Bool("insecure", false, "пропуск проверки TLS")
	timeout := flag.Duration("timeout", 0, "таймаут одного HTTP-запроса (0 — из конфига / 2m)")
	flag.Parse()

	if err := initDBForImport(); err != nil {
		log.Fatal(err)
	}
	itilCfg := config.ConfigData.ConfigItilium
	applyItiliumConfig(&itilCfg, base, domain, user, password, auth, apiPrefix, insecure, timeout, cuStatus)

	if *base == "" {
		*base = "https://itsd.global.bcs/ITILIUM"
	}
	if *domain == "" {
		*domain = "itildomail"
	}
	if *user == "" {
		*user = "itiluser"
	}
	if *auth == "" {
		*auth = itilium.AuthNTLM
	}
	if *apiPrefix == "" {
		*apiPrefix = itilium.DefaultAPIPrefix
	}
	if *cuStatus == "" {
		*cuStatus = itilium.CUStatusAll
	}
	if *timeout <= 0 {
		*timeout = 2 * time.Minute
	}

	pass := resolveItiliumPassword(*password, *domain, *user, itilCfg)
	if pass == "" {
		log.Fatal("пароль не задан: -password, ITILIUM_PASSWORD, ConfigItilium.Password или GetKeyData")
	}

	c, err := itilium.NewClient(itilium.Config{
		Name: "getcu-sync", BaseURL: *base, Login: *user, Password: pass,
		Domain: *domain, AuthMode: *auth, APIPrefix: *apiPrefix,
		Timeout: *timeout, InsecureSkipVerify: *insecure,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	api := c.API()

	log.Printf("синхронизация getCU → БД: classes=%v status=%s", syncCUClasses, *cuStatus)
	for _, cuClass := range syncCUClasses {
		rows, err := fetchAllCU(ctx, api, cuClass, *cuStatus)
		if err != nil {
			log.Fatalf("getCU cuClass=%d: %v", cuClass, err)
		}
		runImportCU(rows, cuClass)
	}
	log.Printf("синхронизация завершена")
}

func fetchAllCU(ctx context.Context, api *itilium.APIClient, cuClass int, cuStatus string) ([]itilium.CURow, error) {
	className := itilium.CUClassName(cuClass)
	if className == "" {
		className = "?"
	}
	q := itilium.GetCUQuery{
		CUClass:  cuClass,
		CUStatus: cuStatus,
		Fields:   itilium.DefaultFieldsForClass(cuClass),
	}
	var all []itilium.CURow
	pages := 0
	for {
		pages++
		log.Printf("GET getCU cuClass=%d (%s) page=%d", cuClass, className, pages)
		resp, err := api.GetCU(ctx, q)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Rows...)
		log.Printf("  rows+=%d total=%d hasMore=%v", len(resp.Rows), len(all), resp.HasMore)
		next, ok := resp.NextPageQuery(q)
		if !ok {
			break
		}
		q = next
	}
	log.Printf("cuClass=%d (%s): загружено rows=%d pages=%d", cuClass, className, len(all), pages)
	return all, nil
}

func applyItiliumConfig(cfg *config.ItiliumConfig, base, domain, user, password, auth, apiPrefix *string, insecure *bool, timeout *time.Duration, cuStatus *string) {
	if cfg == nil {
		return
	}
	if cfg.BaseURL != "" && *base == "" {
		*base = cfg.BaseURL
	}
	if cfg.Domain != "" && *domain == "" {
		*domain = cfg.Domain
	}
	if cfg.Login != "" && *user == "" {
		*user = cfg.Login
	}
	if cfg.Password != "" && cfg.Password != "password" && *password == "" {
		*password = cfg.Password
	}
	if cfg.AuthMode != "" && *auth == "" {
		*auth = cfg.AuthMode
	}
	if cfg.APIPrefix != "" && *apiPrefix == "" {
		*apiPrefix = cfg.APIPrefix
	}
	if cfg.InsecureSkipVerify {
		*insecure = true
	}
	if cfg.TimeoutSec > 0 && *timeout <= 0 {
		*timeout = time.Duration(cfg.TimeoutSec) * time.Second
	}
	if cfg.CUStatus != "" && *cuStatus == "" {
		*cuStatus = cfg.CUStatus
	}
}
