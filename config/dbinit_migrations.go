package config

import (
	"fmt"
	"log"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// Одноразовые миграции данных: каждая выполняется ровно один раз на базе,
// факт выполнения записывается в таблицу loadmon_migration (создаётся при
// первом запуске). В отличие от идемпотентных проверок при каждом старте
// (см. dbinit_servermetric.go), здесь важно именно "один раз": например,
// перевод Stat=0 -> 1 нельзя повторять при каждом запуске, иначе он
// затирал бы исторические записи, которые человек пометил Stat=0 позже.

type dataMigration struct {
	name string
	sql  string
}

var dataMigrations = []dataMigration{
	{
		// 2026-10-02: Stat=0 теперь ставит только человек ("историческая
		// запись"); выведенные автоматикой записи (Stat=0, Active=2) по
		// единому правилу должны быть Stat=1, PowerState=2, Active=2.
		name: "2026-10-02-serverspr-stat0-active2-to-stat1",
		sql:  "UPDATE server_spr_records SET stat = 1 WHERE stat = 0 AND active = 2",
	},
	{
		// 2026-10-02: опечатка в cmdb/updatesprvm.go:checkAndCorrectVmSpr
		// ставила Stat=6666 вместо 666 ("к уничтожению").
		name: "2026-10-02-serverspr-stat6666-to-666",
		sql:  "UPDATE server_spr_records SET stat = 666 WHERE stat = 6666",
	},
}

// runDataMigrations выполняет ещё не применённые миграции на алиасе alias.
func runDataMigrations(alias string) {
	o := orm.NewOrmUsingDB(alias)
	if _, err := o.Raw("CREATE TABLE IF NOT EXISTS loadmon_migration (name VARCHAR(128) PRIMARY KEY, applied_at VARCHAR(32))").Exec(); err != nil {
		log.Printf("runDataMigrations(%s): создание loadmon_migration: %v", alias, err)
		return
	}
	for _, m := range dataMigrations {
		var cnt int
		if err := o.Raw("SELECT count(*) FROM loadmon_migration WHERE name = ?", m.name).QueryRow(&cnt); err != nil {
			log.Printf("runDataMigrations(%s): %s: %v", alias, m.name, err)
			continue
		}
		if cnt > 0 {
			continue
		}
		res, err := o.Raw(m.sql).Exec()
		if err != nil {
			log.Printf("runDataMigrations(%s): %s: ОШИБКА: %v", alias, m.name, err)
			continue
		}
		n, _ := res.RowsAffected()
		if _, err := o.Raw("INSERT INTO loadmon_migration (name, applied_at) VALUES (?, ?)", m.name, time.Now().Format(time.RFC3339)).Exec(); err != nil {
			log.Printf("runDataMigrations(%s): %s: запись факта выполнения: %v", alias, m.name, err)
			continue
		}
		log.Println(fmt.Sprintf("runDataMigrations(%s): %s выполнена, строк: %d", alias, m.name, n))
	}
}
