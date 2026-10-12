package config

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/beego/beego/v2/client/orm"
)

// Перенос уникальности server_metric/server_metric_record на тройку с
// Source (2026-10-01, появление второго источника метрик - SCOM, см.
// models.MetricSourceSCOM и kontext/13-scom-metrics.md).
//
// Почему нужна отдельная миграция: beego RunSyncdb(force=false) на уже
// существующей таблице только ДОБАВЛЯЕТ недостающие колонки (колонка
// source появится сама, со значением DEFAULT 0 = Zabbix у всех старых
// строк), но не трогает ограничения. Ограничение UNIQUE beego создаёт
// внутри CREATE TABLE без имени (beego/v2/client/orm/ddl.go), поэтому:
//   - PostgreSQL даёт ему автоимя <таблица>_<колонки>_key - его можно
//     удалить и создать новое с тем же принципом именования;
//   - SQLite не умеет удалять ограничение таблицы - таблица пересоздаётся
//     по официальной схеме (CREATE новой -> копия данных -> DROP старой ->
//     RENAME -> восстановление индексов) в одной транзакции.
// Без этой миграции первая же запись SCOM с HostId, совпавшим с HostId
// Zabbix в ту же минуту, упала бы на старом UNIQUE (HostId, CollectedAt).
//
// Идемпотентна: если старого ограничения нет (свежая база или миграция уже
// выполнена) - ничего не делает. Вызывается из DBInit ПОСЛЕ RunSyncdb.

type uniqueMigration struct {
	table   string
	oldCols []string // старое ограничение
	newCols []string // новое ограничение
}

var serverMetricUniqueMigrations = []uniqueMigration{
	{table: "server_metric", oldCols: []string{"host_id", "collected_at"}, newCols: []string{"source", "host_id", "collected_at"}},
	{table: "server_metric_record", oldCols: []string{"host_id", "date"}, newCols: []string{"source", "host_id", "date"}},
}

// migrateServerMetricSourceUnique применяет serverMetricUniqueMigrations к
// обеим базам ("default" и "default-add") - модели регистрируются
// глобально, и RunSyncdb создаёт таблицы в каждой из них.
func migrateServerMetricSourceUnique(aliasDrivers map[string]string) {
	for alias, driver := range aliasDrivers {
		o := orm.NewOrmUsingDB(alias)
		for _, m := range serverMetricUniqueMigrations {
			var mErr error
			switch driver {
			case "postgres":
				mErr = migrateUniquePostgres(o, m)
			case "sqlite3":
				mErr = migrateUniqueSQLite(o, m)
			default:
				continue
			}
			if mErr != nil {
				log.Printf("migrateServerMetricSourceUnique(%s, %s): ОШИБКА: %v", alias, m.table, mErr)
			}
		}
	}
}

func migrateUniquePostgres(o orm.Ormer, m uniqueMigration) error {
	oldName := m.table + "_" + strings.Join(m.oldCols, "_") + "_key"
	newName := m.table + "_" + strings.Join(m.newCols, "_") + "_key"
	var cnt int
	if err := o.Raw("SELECT count(*) FROM pg_constraint WHERE conname = ?", oldName).QueryRow(&cnt); err != nil {
		return fmt.Errorf("проверка ограничения %s: %w", oldName, err)
	}
	if cnt == 0 {
		return nil // старого ограничения нет - свежая база или уже мигрировано
	}
	log.Printf("migrateServerMetricSourceUnique: %s - замена UNIQUE (%s) на (%s): перестройка индекса, ОДНОРАЗОВО, может занять минуты на больших таблицах", m.table, strings.Join(m.oldCols, ", "), strings.Join(m.newCols, ", "))
	return o.DoTx(func(_ context.Context, tx orm.TxOrmer) error {
		if _, err := tx.Raw(fmt.Sprintf(`ALTER TABLE "%s" DROP CONSTRAINT IF EXISTS "%s"`, m.table, oldName)).Exec(); err != nil {
			return err
		}
		_, err := tx.Raw(fmt.Sprintf(`ALTER TABLE "%s" ADD CONSTRAINT "%s" UNIQUE ("%s")`, m.table, newName, strings.Join(m.newCols, `", "`))).Exec()
		return err
	})
}

// uniqueClauseRe строит регулярное выражение для UNIQUE (col1, col2, ...)
// в тексте CREATE TABLE с любыми кавычками (` или ") и пробелами.
func uniqueClauseRe(cols []string) *regexp.Regexp {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = "[`\"]?" + regexp.QuoteMeta(c) + "[`\"]?"
	}
	return regexp.MustCompile(`(?i)UNIQUE\s*\(\s*` + strings.Join(parts, `\s*,\s*`) + `\s*\)`)
}

func migrateUniqueSQLite(o orm.Ormer, m uniqueMigration) error {
	var createSQL string
	if err := o.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", m.table).QueryRow(&createSQL); err != nil {
		return nil // таблицы нет - создаст RunSyncdb, мигрировать нечего
	}
	oldRe := uniqueClauseRe(m.oldCols)
	if !oldRe.MatchString(createSQL) {
		return nil // старого ограничения нет
	}
	if !strings.Contains(strings.ToLower(createSQL), "source") {
		return fmt.Errorf("колонка source ещё не добавлена (RunSyncdb не отработал?) - миграция отложена до следующего запуска")
	}

	tmp := m.table + "__mig"
	quoted := make([]string, len(m.newCols))
	for i, c := range m.newCols {
		quoted[i] = "`" + c + "`"
	}
	newSQL := oldRe.ReplaceAllString(createSQL, "UNIQUE ("+strings.Join(quoted, ", ")+")")
	nameRe := regexp.MustCompile("(?i)^\\s*CREATE\\s+TABLE\\s+(IF\\s+NOT\\s+EXISTS\\s+)?[`\"]?" + regexp.QuoteMeta(m.table) + "[`\"]?")
	if !nameRe.MatchString(newSQL) {
		return fmt.Errorf("не удалось разобрать CREATE TABLE: %s", createSQL)
	}
	newSQL = nameRe.ReplaceAllString(newSQL, "CREATE TABLE `"+tmp+"`")

	var indexSQL []string
	if _, err := o.Raw("SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND sql IS NOT NULL", m.table).QueryRows(&indexSQL); err != nil {
		return fmt.Errorf("чтение индексов: %w", err)
	}

	var rowCnt int64
	_ = o.Raw("SELECT count(*) FROM `" + m.table + "`").QueryRow(&rowCnt)
	log.Printf("migrateServerMetricSourceUnique: %s - пересоздание таблицы для UNIQUE (%s): копирование %d строк одной транзакцией, это ОДНОРАЗОВО, но может занять минуты - не прерывайте запуск", m.table, strings.Join(m.newCols, ", "), rowCnt)
	return o.DoTx(func(_ context.Context, tx orm.TxOrmer) error {
		stmts := []string{
			"DROP TABLE IF EXISTS `" + tmp + "`",
			newSQL,
			"INSERT INTO `" + tmp + "` SELECT * FROM `" + m.table + "`",
			"DROP TABLE `" + m.table + "`",
			"ALTER TABLE `" + tmp + "` RENAME TO `" + m.table + "`",
		}
		stmts = append(stmts, indexSQL...)
		for _, st := range stmts {
			if _, err := tx.Raw(st).Exec(); err != nil {
				return fmt.Errorf("%s: %w", st, err)
			}
		}
		return nil
	})
}
