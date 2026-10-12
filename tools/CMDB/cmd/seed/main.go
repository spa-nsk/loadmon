// cmd/seed — выполняет идемпотентные SQL-файлы из database/seed/<profile>/ в одной транзакции
// (используется как `go run ./cmd/seed nc2026` / `go run ./cmd/seed guide2019`, см. plan.md E1.3–E1.4).
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	_ "github.com/jackc/pgx/v5/stdlib"

	"cmdb/internal/platform/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: seed <profile>  (например: seed nc2026)")
		os.Exit(1)
	}
	profile := os.Args[1]
	dir := filepath.Join("database", "seed", profile)

	files, err := seedFiles(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "seed: в %s не найдено .sql файлов\n", dir)
		os.Exit(1)
	}

	dsn := config.LoadDBConfig().DSN
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "seed: sqlconn пуст, проверьте conf/app.conf или CMDB_DB_DSN")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed: open db: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "seed: begin tx: %v\n", err)
		os.Exit(1)
	}

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			_ = tx.Rollback()
			fmt.Fprintf(os.Stderr, "seed: read %s: %v\n", f, err)
			os.Exit(1)
		}
		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			fmt.Fprintf(os.Stderr, "seed: exec %s: %v\n", f, err)
			os.Exit(1)
		}
		fmt.Printf("seed: applied %s\n", f)
	}

	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "seed: commit: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("seed: profile %q applied (%d files)\n", profile, len(files))
}

func seedFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	return files, nil
}
