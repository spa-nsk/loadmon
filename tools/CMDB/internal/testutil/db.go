// Package testutil — общие помощники интеграционных тестов. Тесты, которым нужен PostgreSQL с
// миграциями и seed nc2026, вызывают DB(t): без CMDB_TEST_DB_DSN тест пропускается.
package testutil

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"cmdb/internal/platform/config"
	"cmdb/internal/repo"
)

// Тестовая площадка: сжатый код для имён (TESTQA-S-…) и полный код локации (FK cmdb.ci.location_code).
const (
	TestSite     = "TSTQA1" // = location_code_c для TestLocation (SITE строится из площадки)
	TestLocation = "TST-QA1"
)

// DB инициализирует пул приложения (repo.Init) и возвращает отдельное *sql.DB для проверочных
// запросов теста. Гарантирует наличие тестовой локации.
func DB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("CMDB_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("CMDB_TEST_DB_DSN не задан — пропуск интеграционного теста (нужен PostgreSQL с миграциями и seed nc2026)")
	}
	if err := repo.Init(config.DBConfig{DSN: dsn, MaxOpenConns: 20, MaxIdleConns: 5}); err != nil {
		t.Fatalf("repo.Init: %v", err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// ref.* не версионируется, поэтому ChangeContext не нужен; идемпотентно.
	stmts := []string{
		`INSERT INTO ref.country (country_code, name_en, name_ru) VALUES ('ZZ', 'Test', 'Тест') ON CONFLICT DO NOTHING`,
		`INSERT INTO ref.location (location_code, location_name, location_name_lat, country_code, site_kind)
		 VALUES ('` + TestLocation + `', 'Тестовая площадка', 'Test site', 'ZZ', 'dc') ON CONFLICT DO NOTHING`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("подготовка тестовых данных: %v", err)
		}
	}
	return db
}

// Suffix возвращает строку, уникальную для запуска, — для имён, VRF и прочего, что должно быть
// уникальным в общей тестовой базе между повторными прогонами.
func Suffix() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatInt(int64(rand.Intn(1296)), 36)
}

// NewLocation создаёт новую площадку со случайным кодом (для тестов уникальности «в пределах площадки»).
func NewLocation(t *testing.T, db *sql.DB) string {
	t.Helper()
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	for range 20 {
		code := string([]byte{letters[rand.Intn(len(letters))], letters[rand.Intn(len(letters))], letters[rand.Intn(len(letters))]}) +
			"-" + string(letters[rand.Intn(len(letters))]) + strconv.Itoa(rand.Intn(10))
		res, err := db.Exec(`INSERT INTO ref.location (location_code, location_name, location_name_lat, country_code, site_kind)
			VALUES ($1, 'Тестовая площадка', 'Test site', 'ZZ', 'dc') ON CONFLICT DO NOTHING`, code)
		if err != nil {
			t.Fatalf("NewLocation: %v", err)
		}
		if n, _ := res.RowsAffected(); n == 1 {
			return code
		}
	}
	t.Fatal("NewLocation: не удалось подобрать свободный код")
	return ""
}

// RandomWWN возвращает случайный WWN в виде 50:01:xx:xx:xx:xx:xx:xx.
func RandomWWN() string {
	return fmt.Sprintf("50:01:%02x:%02x:%02x:%02x:%02x:%02x",
		rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256))
}
