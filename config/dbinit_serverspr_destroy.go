package config

import (
	"fmt"
	"log"

	"github.com/beego/beego/v2/client/orm"

	"loadmon/models"
)

// Защита записей server_spr_records, помеченных к уничтожению
// (Stat = models.ServerSprStatDestroy = 666), на уровне БД.
//
// Правило владельца проекта (2026-10-02): "из state=666 нельзя переводить в
// другие. Это должно пропускаться во всех запросах. Это записи помеченные
// для уничтожения". В коде такие записи исключаются из выборок
// (models.ServerSprQS / ExcludeDestroyedCond), но в проекте много старых
// путей, читающих запись по произвольному ключу (Name, HWSprId, ...) и
// сразу сохраняющих её. Триггер гарантирует правило независимо от того,
// каким путём (ORM, сырой SQL, будущий код) пришло обновление: UPDATE
// строки, у которой УЖЕ stat = 666, молча пропускается - строка остаётся
// без изменений (SQLite: RAISE(IGNORE), PostgreSQL: BEFORE-триггер
// возвращает NULL). Перевод В 666 (дедупликация) не ограничивается, DELETE
// тоже (это и есть "уничтожение").
//
// Пропущенный UPDATE возвращает 0 затронутых строк без ошибки - вызывающий
// код не падает и транзакции не откатываются.
//
// Идемпотентно, вызывается из DBInit после RunSyncdb для обоих алиасов.

const destroyTriggerName = "server_spr_records_keep_destroy"

func ensureServerSprDestroyTrigger(aliasDrivers map[string]string) {
	for alias, driver := range aliasDrivers {
		o := orm.NewOrmUsingDB(alias)
		var stmts []string
		switch driver {
		case "sqlite3":
			stmts = []string{
				fmt.Sprintf("DROP TRIGGER IF EXISTS %s", destroyTriggerName),
				fmt.Sprintf(`CREATE TRIGGER %s BEFORE UPDATE ON server_spr_records
FOR EACH ROW WHEN OLD.stat = %d
BEGIN
	SELECT RAISE(IGNORE);
END`, destroyTriggerName, models.ServerSprStatDestroy),
			}
		case "postgres":
			stmts = []string{
				fmt.Sprintf(`CREATE OR REPLACE FUNCTION %s_fn() RETURNS trigger AS $$
BEGIN
	IF OLD.stat = %d THEN
		RETURN NULL;
	END IF;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql`, destroyTriggerName, models.ServerSprStatDestroy),
				fmt.Sprintf("DROP TRIGGER IF EXISTS %s ON server_spr_records", destroyTriggerName),
				fmt.Sprintf("CREATE TRIGGER %s BEFORE UPDATE ON server_spr_records FOR EACH ROW EXECUTE FUNCTION %s_fn()", destroyTriggerName, destroyTriggerName),
			}
		default:
			continue
		}
		for _, st := range stmts {
			if _, err := o.Raw(st).Exec(); err != nil {
				log.Printf("ensureServerSprDestroyTrigger(%s): %v", alias, err)
				break
			}
		}
	}
}
