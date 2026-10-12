// Package config читает конфигурацию beego (conf/app.conf) и переменные окружения.
//
// Beego v2 не подставляет переменные окружения внутрь app.conf, поэтому override
// делаем сами: значение из окружения (CMDB_*) побеждает значение из app.conf, если задано.
package config

import (
	"os"
	"strconv"

	beeweb "github.com/beego/beego/v2/server/web"
)

// DBConfig описывает параметры подключения к PostgreSQL для одного из бинарников.
type DBConfig struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

// ApplyEnvOverrides переопределяет значения app.conf переменными окружения. Вызывается
// в main() до routers.Init()/web.Run(), потому что beego читает httpport/runmode при старте.
func ApplyEnvOverrides() {
	if v := os.Getenv("CMDB_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			beeweb.BConfig.Listen.HTTPPort = port
		}
	}
	if v := os.Getenv("CMDB_RUNMODE"); v != "" {
		beeweb.BConfig.RunMode = v
	}
	if v := os.Getenv("CMDB_XSRFKEY"); v != "" {
		beeweb.BConfig.WebConfig.XSRFKey = v
	}
}

// LoadDBConfig читает sqlconn и лимиты пула из активного runmode-профиля app.conf,
// с возможностью переопределить их через CMDB_DB_DSN / CMDB_DB_MAX_OPEN / CMDB_DB_MAX_IDLE.
func LoadDBConfig() DBConfig {
	dsn, _ := beeweb.AppConfig.String("sqlconn")
	if v := os.Getenv("CMDB_DB_DSN"); v != "" {
		dsn = v
	}

	maxOpen := beeweb.AppConfig.DefaultInt("db_max_open_conns", 20)
	if v := os.Getenv("CMDB_DB_MAX_OPEN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxOpen = n
		}
	}

	maxIdle := beeweb.AppConfig.DefaultInt("db_max_idle_conns", 5)
	if v := os.Getenv("CMDB_DB_MAX_IDLE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			maxIdle = n
		}
	}

	return DBConfig{DSN: dsn, MaxOpenConns: maxOpen, MaxIdleConns: maxIdle}
}

// LogLevel возвращает уровень логирования (по умолчанию "info"), с переопределением CMDB_LOG_LEVEL.
func LogLevel() string {
	if v := os.Getenv("CMDB_LOG_LEVEL"); v != "" {
		return v
	}
	level := beeweb.AppConfig.DefaultString("log_level", "info")
	if level == "" {
		return "info"
	}
	return level
}

// RunMode возвращает активный профиль (dev/test/prod).
func RunMode() string {
	return beeweb.BConfig.RunMode
}
