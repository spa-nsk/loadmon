// Команда seeddata — служебная CLI-утилита для наполнения тестовой SQLite-базы
// loadmon сгенерированными записями серверов: виртуальными машинами и
// физическими серверами (HPE ProLiant) со случайными характеристиками.
// Используется для локальной разработки и тестирования без подключения
// к реальным vCenter/оборудованию.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type appConfig struct {
	DBFile string `json:"DBFile"`
}

func resolveDBPath(configPath, dbOverride string) string {
	if dbOverride != "" {
		return dbOverride
	}
	b, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	var cfg appConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		log.Fatalf("config json: %v", err)
	}
	dbFile := cfg.DBFile
	if dbFile == "" {
		log.Fatal("DBFile не задан в config")
	}
	if !filepath.IsAbs(dbFile) {
		dbFile = filepath.Join(filepath.Dir(configPath), dbFile)
	}
	return dbFile
}

var namePrefixes = []string{
	"web", "app", "db", "srv", "mail", "api", "sql", "mon", "log", "cache",
	"proxy", "file", "batch", "test", "dev", "prod", "redis", "kafka", "nginx", "ldap",
}

type hpModel struct {
	Model      string
	PartNum    string
	FormFactor int
	CPUModel   string
	CPUCount   int
	CorePerCPU int
	CoreMHz    int
	MemoryMb   int
	SizeUnit   int
}

var hpModels = []hpModel{
	{Model: "ProLiant DL360 Gen10", PartNum: "867961-B21", FormFactor: 1, CPUModel: "Intel Xeon Gold 6248R", CPUCount: 2, CorePerCPU: 24, CoreMHz: 3000, MemoryMb: 262144, SizeUnit: 1},
	{Model: "ProLiant DL380 Gen10", PartNum: "868703-B21", FormFactor: 1, CPUModel: "Intel Xeon Silver 4214R", CPUCount: 2, CorePerCPU: 12, CoreMHz: 2400, MemoryMb: 393216, SizeUnit: 2},
	{Model: "ProLiant DL580 Gen10", PartNum: "869120-B21", FormFactor: 1, CPUModel: "Intel Xeon Platinum 8280", CPUCount: 4, CorePerCPU: 28, CoreMHz: 2700, MemoryMb: 1048576, SizeUnit: 4},
	{Model: "ProLiant BL460c Gen10", PartNum: "873940-B21", FormFactor: 2, CPUModel: "Intel Xeon Gold 6230", CPUCount: 2, CorePerCPU: 20, CoreMHz: 2100, MemoryMb: 131072, SizeUnit: 1},
}

func namesWithRange(count, from, to int) []string {
	names := make([]string, 0, count)
	for n := from; n <= to; n++ {
		for _, p := range namePrefixes {
			if len(names) >= count {
				return names
			}
			names = append(names, fmt.Sprintf("%s%02d", p, n))
		}
	}
	return names
}

func hpSerial(r *rand.Rand) string {
	letters := "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digits := "0123456789"
	b := make([]byte, 10)
	b[0] = letters[r.Intn(len(letters))]
	b[1] = letters[r.Intn(len(letters))]
	for i := 2; i < 10; i++ {
		if r.Intn(3) == 0 {
			b[i] = letters[r.Intn(len(letters))]
		} else {
			b[i] = digits[r.Intn(len(digits))]
		}
	}
	return string(b)
}

func insertVM(tx *sql.Tx, name string, now string) error {
	fqdn := name + ".spa-net"
	res, err := tx.Exec(`
		INSERT INTO server_spr_records (
			type, vcenter, datacenter, cluster, name, name_f_q_d_n,
			power_state, active, stat, time__mfy
		) VALUES (1, ?, ?, ?, ?, ?, 1, 1, 1, ?)`,
		"vcenter.spa-net", "Production", "Common", name, fqdn, now,
	)
	if err != nil {
		return fmt.Errorf("server_spr_records vm %s: %w", name, err)
	}
	serverID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	res, err = tx.Exec(`
		INSERT INTO vm_spr_record (
			server_id, vcenter, datacenter, cluster,
			vm__name, vm__host_name, vm__power_state,
			vm__num_c_p_u, vm__memory_size_g_b, vm__guest_full_name
		) VALUES (?, ?, ?, ?, ?, ?, 1, 2, 8, 'Microsoft Windows Server 2019')`,
		serverID, "vcenter.spa-net", "Production", "Common", name, fqdn,
	)
	if err != nil {
		return fmt.Errorf("vm_spr_record %s: %w", name, err)
	}
	vmID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE server_spr_records SET v_m_spr_id = ? WHERE id = ?`, vmID, serverID)
	if err != nil {
		return fmt.Errorf("link vm %s: %w", name, err)
	}
	return nil
}

func insertHW(tx *sql.Tx, name string, now string, r *rand.Rand) error {
	fqdn := name + ".spa-net"
	serial := hpSerial(r)
	m := hpModels[r.Intn(len(hpModels))]

	res, err := tx.Exec(`
		INSERT INTO server_spr_records (
			type, name, name_f_q_d_n, serial_number,
			power_state, active, stat, time__mfy
		) VALUES (0, ?, ?, ?, 1, 1, 1, ?)`,
		name, fqdn, serial, now,
	)
	if err != nil {
		return fmt.Errorf("server_spr_records hw %s: %w", name, err)
	}
	serverID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	res, err = tx.Exec(`
		INSERT INTO server_hardware (
			power_state, active, server_id, vendor, model, part_num, form_factor,
			serial_num, c_p_u_model, c_p_u_count, core_per_c_p_u, core_m_hz, memory_mb,
			size_unit, datacenter_name, rack_name, time__mfy, source_update
		) VALUES (1, 1, ?, 'HPE', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'DC-SPA', 'RACK-A01', ?, 'seeddata')`,
		serverID, m.Model, m.PartNum, m.FormFactor, serial, m.CPUModel,
		m.CPUCount, m.CorePerCPU, m.CoreMHz, m.MemoryMb, m.SizeUnit, now,
	)
	if err != nil {
		return fmt.Errorf("server_hardware %s: %w", name, err)
	}
	hwID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE server_spr_records SET h_w_spr_id = ? WHERE id = ?`, hwID, serverID)
	if err != nil {
		return fmt.Errorf("link hw %s: %w", name, err)
	}
	return nil
}

func main() {
	config := flag.String("config", "server_web_new/conf/app2.json", "путь к app2.json")
	dbPath := flag.String("db", "", "путь к sqlite (перекрывает config)")
	count := flag.Int("count", 100, "количество VM и HW записей каждого типа")
	flag.Parse()

	path := resolveDBPath(*config, *dbPath)
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	now := time.Now().Local().Format("2006-01-02 15:04:05")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	vmNames := namesWithRange(*count, 1, 5)
	hwNames := namesWithRange(*count, 6, 10)

	for _, name := range vmNames {
		if err := insertVM(tx, name, now); err != nil {
			log.Fatal(err)
		}
	}
	for _, name := range hwNames {
		if err := insertHW(tx, name, now, r); err != nil {
			log.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Добавлено %d VM (type=1) и %d HW (type=0) в %s\n", *count, *count, path)
}
