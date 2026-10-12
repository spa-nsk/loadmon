// cmd/migrate — обёртка над goose для миграций database/migrations (используется Makefile-целью migrate).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"

	"cmdb/internal/platform/config"
)

func main() {
	flag.Parse()
	command := "up"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}
	args := flag.Args()
	if len(args) > 1 {
		args = args[1:]
	} else {
		args = nil
	}

	dsn := config.LoadDBConfig().DSN
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "migrate: sqlconn пуст, проверьте conf/app.conf или CMDB_DB_DSN")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: set dialect: %v\n", err)
		os.Exit(1)
	}

	if err := goose.RunContext(context.Background(), command, db, "database/migrations", args...); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %s: %v\n", command, err)
		os.Exit(1)
	}
}
