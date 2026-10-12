package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/beego/beego/v2/client/orm"
)

// ciColumns — точный список колонок cmdb.ci (миграция 0007_cmdb_ci.sql), нужен для явной вставки
// в hist.ci в обход триггера (см. комментарий пакета в main.go): для КЕ, кроме id, uid и служебных
// hist_*, состав колонок у cmdb.ci и hist.ci одинаков (hist.ci создана как `LIKE cmdb.ci`).
const ciColumns = `id, uid, ci_class, name, source_name, naming_status, naming_profile_code, naming_rule_code,
	location_code, designation_code, sequence_number, company_code, directory_code, environment_code,
	lifecycle_stage, lifecycle_status, criticality, ownership, service_provider, kii_object, kii_category,
	model_id, serial_number, asset_tag, site_place_id, rack_position, owner_person_id, support_group_id,
	description, attrs, first_seen_at, last_seen_at, verified_at, verification_status,
	created_at, created_by, updated_at, updated_by, change_set_id, row_version, sys_period`

const monthsBack = 12

// ensurePastPartitions — секции hist.ci/audit.change_log за прошлые monthsBack месяцев: без них
// вставка всё равно сработает (уйдёт в DEFAULT), но порезка диапазона в запросах «на дату» будет
// хуже — ops.ensure_partitions (миграция 0027) создаёт только текущий период и вперёд, для
// прошлого нужен отдельный, такой же по духу, но не переиспользующий её код (разная семантика
// «вперёд от сейчас» vs «назад от сейчас»).
func ensurePastPartitions(ctx context.Context) {
	o := orm.NewOrm()
	for i := 1; i <= monthsBack; i++ {
		start := time.Now().AddDate(0, -i, 0)
		start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0)
		for _, t := range []struct{ schema, table string }{{"hist", "ci"}, {"audit", "change_log"}} {
			name := fmt.Sprintf("%s_%s", t.table, start.Format("200601"))
			var exists int
			o.RawWithCtx(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=? AND c.relname=?`, t.schema, name).QueryRow(&exists)
			if exists > 0 {
				continue
			}
			// Границы секции — литералами в тексте DDL, не bind-параметрами: PostgreSQL считает
			// границы PARTITION OF ... FOR VALUES константным выражением, разбираемым при
			// планировании DDL, а не значением, подставляемым на этапе выполнения — с ? здесь
			// beego/pgx отвечали «mismatched param and argument count» (поймано этим прогоном).
			sql := fmt.Sprintf(`CREATE TABLE %s.%s PARTITION OF %s.%s FOR VALUES FROM ('%s') TO ('%s')`,
				t.schema, name, t.schema, t.table, start.Format(time.RFC3339), end.Format(time.RFC3339))
			if _, err := o.RawWithCtx(ctx, sql).Exec(); err != nil {
				fatalf("ensurePastPartitions %s.%s: %v", t.schema, name, err)
			}
		}
	}
}

// generateHistory фабрикует представительный объём истории за monthsBack месяцев напрямую в
// hist.ci/audit.change_log, В ОБХОД триггера hist.versioning() (тот всегда пишет now() — задним
// числом им не воспользоваться, см. комментарий пакета) — явно синтетические строки, не реальный
// журнал аудита. subsetFrac — доля КЕ, получающих историю (не все — «~2% КЕ меняются в день» из
// intent.md §8 за год даёт куда больше 100% с повторами, здесь — количество версий на КЕ, не доля).
func generateHistory(ctx context.Context, ids map[string][]int64) (histRows, changeLogRows int) {
	ensurePastPartitions(ctx)

	var all []int64
	for _, v := range ids {
		all = append(all, v...)
	}
	rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	const subsetFrac = 0.25
	subset := all[:int(float64(len(all))*subsetFrac)]

	o := orm.NewOrm()
	// один синтетический change_set на "день" истории — не на каждую версию, тот же принцип
	// группировки, что у настоящего импорта (E4.8: много строк в одном прогоне — один change_set).
	changeSetByDay := map[string]int64{}
	getChangeSet := func(day time.Time) int64 {
		key := day.Format("2006-01-02")
		if id, ok := changeSetByDay[key]; ok {
			return id
		}
		var id int64
		if err := o.RawWithCtx(ctx, `
			INSERT INTO audit.change_set (source_kind, effective_at, started_at, committed_at, is_backdated, reason)
			VALUES ('system', ?, ?, ?, true, 'cmdb/loadgen: синтетическая история нагрузочного теста')
			RETURNING id`, day, day, day).QueryRow(&id); err != nil {
			fatalf("audit.change_set (история): %v", err)
		}
		changeSetByDay[key] = id
		return id
	}

	for _, ciID := range subset {
		versions := 1 + rand.Intn(3)
		for v := 0; v < versions; v++ {
			daysAgo := rand.Intn(monthsBack * 30)
			at := time.Now().AddDate(0, 0, -daysAgo)
			csID := getChangeSet(at)

			// hist.ci: копия текущей строки cmdb.ci с задним числом hist_recorded_at/sys_period —
			// содержимое не обязано быть буквально прежним значением (это нагрузочные данные, не
			// настоящий аудит), важен объём и форма строки для проверки партиционирования/запросов.
			if _, err := o.RawWithCtx(ctx, `
				INSERT INTO hist.ci (`+ciColumns+`, hist_op, hist_recorded_at)
				SELECT `+ciColumns+`, 'U', ? FROM cmdb.ci WHERE id = ?`, at, ciID).Exec(); err != nil {
				fatalf("hist.ci (история): %v", err)
			}
			histRows++

			if _, err := o.RawWithCtx(ctx, `
				INSERT INTO audit.change_log (change_set_id, entity, entity_id, ci_id, op, attr_name, old_value, new_value, at)
				VALUES (?, 'ci', ?, ?, 'U', 'criticality', '"medium"'::jsonb, '"high"'::jsonb, ?)`,
				csID, ciID, ciID, at).Exec(); err != nil {
				fatalf("audit.change_log (история): %v", err)
			}
			changeLogRows++
		}
		if histRows%20000 == 0 {
			fmt.Printf("  история: %d строк hist.ci...\n", histRows)
		}
	}
	return histRows, changeLogRows
}
