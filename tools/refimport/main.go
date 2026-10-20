// Команда refimport очищает тестовую базу из conf/app2.json и заливает в неё
// справочники из рабочей базы PostgreSQL (по умолчанию - база "loadmon").
//
// Источник: PostgreSQL, пользователь, пароль, хост, порт и sslmode берутся из
// conf/app2.json (как у текущей тестовой), имя базы - флаг -srcdb ("loadmon").
// Приёмник: PostgreSQL, база DBName из conf/app2.json (тот же сервер, схема должна уже существовать).
//
// Переносятся ТОЛЬКО справочники (список refTables ниже): серверы,
// конфигурации серверов, СХД, блейд-корзины, площадки/ЦОД/шкафы, модели,
// вендоры, сети, коммутаторы, vCenter/кластеры/ВМ, сервисы и т.п. Метрики,
// история мощности, журналы событий, архив сопоставлений Zabbix/SCOM и
// пользователи веб-интерфейса НЕ переносятся.
//
// Очистка: удаляются данные ВСЕХ таблиц приёмника, кроме users и
// loadmon_migration (учётные записи веб-интерфейса и отметки одноразовых
// миграций остаются). Схема приёмника не меняется. База "default-add" не
// затрагивается.
//
// По умолчанию - пробный прогон (только счётчики). Для применения -apply:
//
//	go run ./tools/refimport                 # пробный прогон
//	go run ./tools/refimport -apply          # очистка + импорт справочников
//	go run ./tools/refimport -srcdb loadmon -apply
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"loadmon/config"
	"loadmon/utilities"

	_ "github.com/lib/pq"
)

// refTables - справочники в порядке импорта (родители раньше потомков).
var refTables = []string{
	// площадки, ЦОД, шкафы
	"cmdb_locality",
	"cmdb_datacenter_site",
	"cmdb_rack_cabinet",
	// классификаторы оборудования
	"cmdb_model_vendor",
	"cmdb_model_equipment",
	"cmdb_component_type",
	"cmdb_components",
	"cmdb_components_cpu",
	"cmdb_components_ram",
	"cmdb_models_end_life",
	"cmdb_reason_end_life",
	"cmdb_category_equipment",
	"cmdb_server_type",
	"cmdb_operational_system",
	// сеть
	"network_spr_records",
	"switch_spr_records",
	"switch_network_link",
	"ip_address_spr_records",
	// блейд-корзины, серверы, конфигурации
	"blade_systems",
	"server_hardware",
	"server_hardware_components",
	"server_spr_records",
	"server_spr_records_arch",
	"server_network_interface",
	"spr_disk_info",
	"spr_cfo_allocation",
	"spr_busines_allocation",
	// СХД
	"san_systems",
	"storage_systems",
	"storage_system_placement",
	"storage_enclosure",
	"storage_disk",
	"storage_pool",
	"storage_volumes",
	// виртуализация (справочные записи)
	"vcenter_spr_records",
	"datacenter_spr_records",
	"cluster_spr_records",
	"vm_spr_record",
	"datastore",
	// персонал, сервисы, контрагенты
	"model_personal_equipment",
	"role_personal",
	"role_personal_item",
	"competence_center",
	"role_competence_center",
	"app_service",
	"service_member",
	"members_fg",
	"counterparty",
}

// keepTables - таблицы приёмника, которые не очищаются.
var keepTables = map[string]bool{
	"users":             true,
	"loadmon_migration": true,
	"sqlite_sequence":   true,
}

func q(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

func main() {
	srcDB := flag.String("srcdb", "loadmon", "имя базы PostgreSQL - источника справочников")
	apply := flag.Bool("apply", false, "применить (по умолчанию - пробный прогон)")
	flag.Parse()

	if _, err := config.ConfigData.LoadConfigData(); err != nil {
		log.Fatalf("refimport: конфигурация: %v", err)
	}
	cfg := &config.ConfigData
	if cfg.Password == "password" {
		cfg.Password = utilities.GetKeyData("work")
	}
	if cfg.DBName == *srcDB {
		log.Fatalf("refimport: источник и приёмник - одна и та же база %q, отмена", *srcDB)
	}
	dsn := func(db string) string {
		return fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s",
			cfg.UserName, cfg.Password, db, cfg.HostName, cfg.Port, cfg.SSLmode)
	}
	ctx := context.Background()
	dst, err := sql.Open("postgres", dsn(cfg.DBName))
	if err != nil {
		log.Fatalf("refimport: приёмник: %v", err)
	}
	defer dst.Close()
	if err := dst.PingContext(ctx); err != nil {
		log.Fatalf("refimport: нет подключения к приёмнику %s@%s:%d/%s: %v", cfg.UserName, cfg.HostName, cfg.Port, cfg.DBName, err)
	}

	src, err := sql.Open("postgres", dsn(*srcDB))
	if err != nil {
		log.Fatalf("refimport: источник: %v", err)
	}
	defer src.Close()
	if err := src.PingContext(ctx); err != nil {
		log.Fatalf("refimport: нет подключения к источнику %s@%s:%d/%s: %v", cfg.UserName, cfg.HostName, cfg.Port, *srcDB, err)
	}

	const isSqlite = false
	log.Printf("refimport: postgres %s:%d, источник %s -> приёмник %s", cfg.HostName, cfg.Port, *srcDB, cfg.DBName)

	// таблицы приёмника
	dstTables, err := listTables(ctx, dst, isSqlite)
	if err != nil {
		log.Fatalf("refimport: список таблиц приёмника: %v", err)
	}
	var clearList []string
	for _, t := range dstTables {
		if !keepTables[t] {
			clearList = append(clearList, t)
		}
	}

	// ---- пробный прогон / план ----
	dstSet := map[string]bool{}
	for _, t := range dstTables {
		dstSet[t] = true
	}
	var toImport []string
	for _, t := range refTables {
		var n int64
		if err := src.QueryRowContext(ctx, "SELECT count(*) FROM "+q(t)).Scan(&n); err != nil {
			log.Printf("  источник: %-30s пропуск (%v)", t, firstLine(err))
			continue
		}
		if !dstSet[t] && *apply {
			log.Printf("  приёмник: %-30s таблицы нет (схема не создана), пропуск", t)
			continue
		}
		log.Printf("  к импорту: %-30s %d строк", t, n)
		toImport = append(toImport, t)
	}
	if !*apply {
		var total int64
		for _, t := range clearList {
			var n int64
			if err := dst.QueryRowContext(ctx, "SELECT count(*) FROM "+q(t)).Scan(&n); err == nil {
				total += n
			}
		}
		log.Printf("refimport: ПРОБНЫЙ ПРОГОН. Будет очищено таблиц %d (строк %d), сохранены %v; импортировано таблиц %d. Для применения добавьте -apply",
			len(clearList), total, keys(keepTables), len(toImport))
		return
	}

	// ---- очистка + импорт в одной транзакции на одном соединении ----
	conn, err := dst.Conn(ctx)
	if err != nil {
		log.Fatalf("refimport: соединение: %v", err)
	}
	defer conn.Close()
	if isSqlite {
		if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
			log.Fatalf("refimport: %v", err)
		}
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("refimport: транзакция: %v", err)
	}
	fail := func(format string, a ...any) {
		_ = tx.Rollback()
		log.Fatalf("refimport: ОТКАТ, приёмник не изменён: "+format, a...)
	}

	t0 := time.Now()
	if isSqlite {
		for _, t := range clearList {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+q(t)); err != nil {
				fail("очистка %s: %v", t, err)
			}
			_, _ = tx.ExecContext(ctx, "DELETE FROM sqlite_sequence WHERE name=?", t)
		}
	} else if len(clearList) > 0 {
		qs := make([]string, len(clearList))
		for i, t := range clearList {
			qs[i] = q(t)
		}
		if _, err := tx.ExecContext(ctx, "TRUNCATE "+strings.Join(qs, ", ")+" RESTART IDENTITY"); err != nil {
			fail("очистка: %v", err)
		}
	}
	log.Printf("refimport: очищено таблиц %d", len(clearList))

	var totalRows int64
	for _, t := range toImport {
		n, err := copyTable(ctx, src, tx, t, isSqlite)
		if err != nil {
			fail("импорт %s: %v", t, err)
		}
		totalRows += n
		log.Printf("  %-30s %d строк", t, n)
	}
	if err := tx.Commit(); err != nil {
		log.Fatalf("refimport: commit: %v", err)
	}
	if !isSqlite {
		fixSequences(ctx, conn, toImport)
	}
	log.Printf("refimport: ГОТОВО. Таблиц %d, строк %d, за %s", len(toImport), totalRows, time.Since(t0).Round(time.Millisecond))
}

func firstLine(err error) string {
	return strings.SplitN(err.Error(), "\n", 2)[0]
}

func keys(m map[string]bool) []string {
	var r []string
	for k := range m {
		r = append(r, k)
	}
	return r
}

func listTables(ctx context.Context, db *sql.DB, isSqlite bool) ([]string, error) {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY 1"
	if isSqlite {
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY 1"
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var r []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		r = append(r, s)
	}
	return r, rows.Err()
}

func dstColumns(ctx context.Context, tx *sql.Tx, table string, isSqlite bool) (map[string]bool, error) {
	cols := map[string]bool{}
	if isSqlite {
		rows, err := tx.QueryContext(ctx, "PRAGMA table_info("+q(table)+")")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var cid, notnull, pk int
			var name, typ string
			var dflt any
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				return nil, err
			}
			cols[name] = true
		}
		return cols, rows.Err()
	}
	rows, err := tx.QueryContext(ctx, "SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=$1", table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		cols[s] = true
	}
	return cols, rows.Err()
}

// copyTable переносит строки таблицы по общим колонкам источника и приёмника.
func copyTable(ctx context.Context, src *sql.DB, tx *sql.Tx, table string, isSqlite bool) (int64, error) {
	dcols, err := dstColumns(ctx, tx, table, isSqlite)
	if err != nil {
		return 0, err
	}
	rows, err := src.QueryContext(ctx, "SELECT * FROM "+q(table))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	scols, err := rows.Columns()
	if err != nil {
		return 0, err
	}
	ctypes, err := rows.ColumnTypes()
	if err != nil {
		return 0, err
	}
	var idx []int // индексы общих колонок в строке источника
	var names []string
	for i, c := range scols {
		if dcols[c] {
			idx = append(idx, i)
			names = append(names, q(c))
		}
	}
	if len(idx) == 0 {
		return 0, fmt.Errorf("нет общих колонок")
	}
	ph := make([]string, len(idx))
	for i := range ph {
		if isSqlite {
			ph[i] = "?"
		} else {
			ph[i] = fmt.Sprintf("$%d", i+1)
		}
	}
	stmt, err := tx.PrepareContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", q(table), strings.Join(names, ","), strings.Join(ph, ",")))
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	vals := make([]any, len(scols))
	ptrs := make([]any, len(scols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	args := make([]any, len(idx))
	var n int64
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return n, err
		}
		for j, i := range idx {
			v := vals[i]
			// lib/pq отдаёт numeric и пр. как []byte; в текст, кроме bytea
			if b, ok := v.([]byte); ok && ctypes[i].DatabaseTypeName() != "BYTEA" {
				v = string(b)
			}
			args[j] = v
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return n, fmt.Errorf("строка %d: %w", n+1, err)
		}
		n++
	}
	return n, rows.Err()
}

// fixSequences выравнивает последовательности postgres после вставки с явными id.
func fixSequences(ctx context.Context, conn *sql.Conn, tables []string) {
	for _, t := range tables {
		_, err := conn.ExecContext(ctx, fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s','id'), COALESCE((SELECT max(id) FROM %s),0)+1, false)`, t, q(t)))
		if err != nil {
			log.Printf("refimport: последовательность %s: %v", t, firstLine(err))
		}
	}
}
