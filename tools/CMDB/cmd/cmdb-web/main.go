// cmdb-web — веб-интерфейс и REST API CMDB (beego v2).
package main

import (
	"context"
	"os"
	"time"

	web "github.com/beego/beego/v2/server/web"

	"cmdb/internal/filters"
	_ "cmdb/internal/importer/connectors/all" // виды коннекторов для «проверить подключение»
	"cmdb/internal/platform/config"
	"cmdb/internal/platform/logging"
	"cmdb/internal/repo"
	"cmdb/internal/routers"
	"cmdb/internal/services/auth"
	"cmdb/internal/services/quality"
)

func main() {
	config.ApplyEnvOverrides()
	logger := logging.Setup(config.LogLevel())

	if err := repo.Init(config.LoadDBConfig()); err != nil {
		logger.Error("db init failed", "error", err)
		panic(err)
	}

	// Аутентификация (E7.1). CMDB_AUTH_DISABLED=1 — режим разработки без входа (все запросы от dev-администратора);
	// в prod-профиле он запрещён: сервис не стартует, а не молча работает без защиты.
	disabled := os.Getenv("CMDB_AUTH_DISABLED") == "1"
	if disabled && config.RunMode() == "prod" {
		logger.Error("CMDB_AUTH_DISABLED=1 недопустимо в prod-профиле — запуск отменён")
		os.Exit(1)
	}
	svc := &auth.Service{LDAP: auth.LDAPFromEnv()}
	filters.SetAuth(svc, disabled)
	if disabled {
		logger.Warn("аутентификация ОТКЛЮЧЕНА (CMDB_AUTH_DISABLED=1): режим только для локальной разработки")
	} else if svc.LDAP == nil {
		logger.Info("LDAP не настроен (CMDB_LDAP_URL): вход только локальными учётными записями (создать: go run ./cmd/cmdb-user add …)")
	}
	go func() {
		for range time.Tick(time.Hour) {
			if n, err := svc.PurgeSessions(context.Background()); err != nil {
				logger.Error("session purge failed", "error", err)
			} else if n > 0 {
				logger.Info("expired sessions purged", "count", n)
			}
		}
	}()

	// Дашборд качества данных (E7.4): страница /quality и так пересчитывает и сохраняет снимок
	// на каждый заход, но без него история пуста до первого визита и рвётся в дни, когда никто не
	// заходил, — эта горутина досчитывает снимок раз в сутки независимо от посещений страницы.
	go func() {
		for range time.Tick(24 * time.Hour) {
			snap, err := quality.Compute(context.Background())
			if err != nil {
				logger.Error("quality snapshot compute failed", "error", err)
				continue
			}
			if err := quality.Save(context.Background(), snap); err != nil {
				logger.Error("quality snapshot save failed", "error", err)
			}
		}
	}()

	routers.Init()

	logger.Info("cmdb-web starting", "runmode", config.RunMode())
	web.Run()
}
