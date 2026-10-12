// Package repo — слой доступа к PostgreSQL: инициализация пула (pgx через beego ORM) и
// raw-SQL репозитории для истории, AS OF, JSONB и диапазонов, которые не выразить ORM beego.
package repo

import (
	"context"
	"fmt"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует sql-драйвер "pgx"

	"cmdb/internal/platform/config"
)

const defaultAlias = "default"

var driverRegistered bool

// Init регистрирует пул соединений beego ORM (default) на pgx-драйвере по конфигурации из app.conf.
func Init(cfg config.DBConfig) error {
	if cfg.DSN == "" {
		return fmt.Errorf("repo: sqlconn is empty, проверьте app.conf/CMDB_DB_DSN")
	}
	// Повторный вызов (например, из нескольких тестов одного процесса) — не ошибка: beego не
	// разрешает регистрировать один alias дважды.
	if _, err := orm.GetDB(defaultAlias); err == nil {
		return nil
	}
	if !driverRegistered {
		if err := orm.RegisterDriver("pgx", orm.DRPostgres); err != nil {
			return fmt.Errorf("repo: register driver: %w", err)
		}
		driverRegistered = true
	}
	if err := orm.RegisterDataBase(
		defaultAlias, "pgx", cfg.DSN,
		orm.MaxOpenConnections(cfg.MaxOpenConns),
		orm.MaxIdleConnections(cfg.MaxIdleConns),
	); err != nil {
		return fmt.Errorf("repo: register database: %w", err)
	}
	return nil
}

// Ping проверяет соединение с БД коротким запросом.
func Ping(ctx context.Context) error {
	o := orm.NewOrm()
	var one int
	err := o.RawWithCtx(ctx, "SELECT 1").QueryRow(&one)
	if err != nil {
		return fmt.Errorf("repo: ping: %w", err)
	}
	return nil
}
