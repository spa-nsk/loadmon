// cmd/loadgen — генератор синтетических данных по профилю intent.md §9 «крупная организация»
// (plan.md E8.5): ~100 тыс. КЕ, ~250 тыс. компонентов, представительный объём истории — для
// проверки SLA (intent.md §10) и тюнинга индексов/секций на реалистичном объёме. НЕ предназначен
// для боевой БД — запускать только против отдельной базы (CMDB_DB_DSN), не общей dev/test.
//
// Данные пишутся через тот же internal/services/changectx.Run, что и настоящее приложение (один
// change_set на пакет, а не на каждую КЕ — так же, как настоящий импорт группирует много КЕ в
// один прогон, E4.8) — обычные (не через хитрые обходы) INSERT проходят через триггер
// hist.versioning() как боевые. История «за год» так получена быть не может (триггер всегда
// пишет sys_period/hist_recorded_at = now(), не подставить прошлую дату) — для нижнего пласта
// истории пишем в hist.ci/audit.change_log напрямую, в обход триггера, с явно задним числом,
// отдельным, ясно обозначенным шагом (см. generateHistory).
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/jackc/pgx/v5/stdlib"

	"cmdb/internal/platform/config"
	"cmdb/internal/platform/logging"
	"cmdb/internal/repo"
	"cmdb/internal/services/changectx"
)

const batchSize = 500

// classSpec — сколько КЕ класса генерировать; распределение — по intent.md §9 (крупная
// организация: 1500 физических серверов, из них ~400 гипервизоров, 12000 ВМ, 60 СХД, 40
// FC-коммутаторов, 1500 сетевых устройств, 3000 прочих, ~40 тыс. томов, ~3 тыс. датасторов,
// ~10 тыс. шар/qtree, ~2 тыс. БД, ~200 кластеров) — с округлениями, сумма близка к нижней границе
// заявленного диапазона 80–100 тыс. КЕ, не подогнана искусственно под ровно 100000.
var classSpecs = []classSpec{
	{"server", 1100, extServer},
	{"hypervisor_host", 400, extHypervisor},
	{"vm", 12000, extVM},
	{"storage_array", 60, extStorageArray},
	{"fc_switch", 40, extFCSwitch},
	{"net_switch", 900, extNetworkDevice},
	{"net_firewall", 200, extNetworkDevice},
	{"net_loadbalancer", 150, extNetworkDevice},
	{"net_wifi_ap", 150, extNetworkDevice},
	{"net_proxy", 50, extNetworkDevice},
	{"net_mgmt", 50, extNetworkDevice},
	{"cluster", 200, extCluster},
	{"datastore", 3000, extDatastore},
	{"volume", 45000, extVolume},
	{"share", 8000, nil},
	{"qtree", 2000, nil},
	{"db_instance", 2000, nil},
	{"tape_library", 30, nil},
	{"backup_server", 40, nil},
	{"facility_ups", 400, nil},
	{"facility_pdu", 800, nil},
	{"facility_cooling", 200, nil},
}

type classSpec struct {
	class string
	count int
	ext   extBuilder // nil — класс без таблицы-расширения в этом прогоне
}

// extBuilder строит SQL и аргументы вставки в таблицу-расширение для одного пакета уже созданных
// КЕ (ids — id этого пакета, в порядке вставки cmdb.ci).
type extBuilder func(ids []int64) (table string, cols []string, rows [][]any)

func main() {
	config.ApplyEnvOverrides()
	logging.Setup("warn")
	if err := repo.Init(config.LoadDBConfig()); err != nil {
		log.Fatalf("repo.Init: %v", err)
	}
	ctx := context.Background()
	if err := repo.Ping(ctx); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}

	rand.Seed(42) // воспроизводимость между прогонами — детерминированный набор для сравнения EXPLAIN

	fmt.Println("== cmdb/loadgen: генератор нагрузки по профилю intent.md §9 ==")
	locations := ensureLocations(ctx)
	fmt.Printf("площадки: %v\n", locations)

	ids := map[string][]int64{}
	compTotal, ifTotal := 0, 0
	for _, spec := range classSpecs {
		t0 := time.Now()
		classIDs, comps, ifaces := generateClass(ctx, spec, locations)
		ids[spec.class] = classIDs
		compTotal += comps
		ifTotal += ifaces
		fmt.Printf("%-18s %6d КЕ  (%6d компонентов, %6d интерфейсов)  %s\n", spec.class, len(classIDs), comps, ifaces, time.Since(t0).Round(time.Millisecond))
	}

	relTotal := generateRelations(ctx, ids)
	fmt.Printf("связей создано: %d\n", relTotal)

	histCI, histChangeLog := generateHistory(ctx, ids)
	fmt.Printf("история (в обход триггера, задним числом до 12 мес.): hist.ci=%d, audit.change_log=%d\n", histCI, histChangeLog)

	total := 0
	for _, v := range ids {
		total += len(v)
	}
	fmt.Printf("\nИТОГО: КЕ=%d, компонентов=%d, интерфейсов=%d, связей=%d, версий истории=%d\n",
		total, compTotal, ifTotal, relTotal, histCI)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// ensureLocations — несколько синтетических площадок (не боевые коды, LG-префикс — узнаваемо
// «нагрузочные», не спутать со справочником реальной организации).
func ensureLocations(ctx context.Context) []string {
	o := orm.NewOrm()
	if _, err := o.RawWithCtx(ctx, `INSERT INTO ref.country (country_code, name_en, name_ru) VALUES ('ZZ', 'Loadgen', 'Нагрузочный тест') ON CONFLICT DO NOTHING`).Exec(); err != nil {
		fatalf("ref.country: %v", err)
	}
	// location_code: CHECK ^[A-Z2-9]{3}(-[A-Z0-9]{1,3})?$ — цифры 0/1 запрещены (риск спутать с O/I).
	codes := []string{"LGA", "LGB", "LGC", "LGD", "LGE"}
	for _, c := range codes {
		if _, err := o.RawWithCtx(ctx, `
			INSERT INTO ref.location (location_code, location_name, location_name_lat, country_code, site_kind)
			VALUES (?, ?, ?, 'ZZ', 'dc') ON CONFLICT (location_code) DO NOTHING`,
			c, "Нагрузочная площадка "+c, c).Exec(); err != nil {
			fatalf("ref.location %s: %v", c, err)
		}
	}
	return codes
}

func randLocation(locs []string) string { return locs[rand.Intn(len(locs))] }

// generateClass создаёт КЕ класса пакетами batchSize, для каждого пакета — один change_set
// (internal/services/changectx.Run, тот же путь записи, что у настоящего приложения) и, если у
// класса есть extBuilder, компоненты/интерфейсы по классу.
func generateClass(ctx context.Context, spec classSpec, locs []string) (ids []int64, components, interfaces int) {
	remaining := spec.count
	seq := 0
	for remaining > 0 {
		n := min(batchSize, remaining)
		remaining -= n

		var batchIDs []int64
		_, err := changectx.Run(ctx, changectx.Meta{Source: changectx.SourceSystem, UserID: 0}, func(ctx context.Context, tx orm.TxOrmer) error {
			names := make([]string, n)
			for i := 0; i < n; i++ {
				seq++
				names[i] = fmt.Sprintf("LOADGEN-%s-%06d", strings.ToUpper(spec.class), seq)
			}
			batchIDs = bulkInsertCI(ctx, tx, spec.class, names, locs)

			if spec.ext != nil {
				table, cols, rows := spec.ext(batchIDs)
				bulkInsertExt(ctx, tx, table, cols, rows)
			}
			if compsPerCI(spec.class) > 0 {
				components += bulkInsertComponents(ctx, tx, spec.class, batchIDs)
			}
			if ifacesPerCI(spec.class) > 0 {
				interfaces += bulkInsertInterfaces(ctx, tx, spec.class, batchIDs)
			}
			return nil
		})
		if err != nil {
			fatalf("генерация %s: %v", spec.class, err)
		}
		ids = append(ids, batchIDs...)
	}
	return ids, components, interfaces
}

func bulkInsertCI(ctx context.Context, tx orm.TxOrmer, class string, names []string, locs []string) []int64 {
	var sb strings.Builder
	sb.WriteString(`INSERT INTO cmdb.ci (ci_class, name, naming_status, location_code, lifecycle_stage, criticality) VALUES `)
	args := make([]any, 0, len(names)*6)
	for i, name := range names {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("(?,?,?,?,?,?)")
		args = append(args, class, name, "exception", randLocation(locs), "Operational", []string{"low", "medium", "high"}[rand.Intn(3)])
	}
	sb.WriteString(" RETURNING id")

	var rows []orm.Params
	if _, err := tx.RawWithCtx(ctx, sb.String(), args...).Values(&rows); err != nil {
		fatalf("bulkInsertCI %s: %v", class, err)
	}
	ids := make([]int64, len(rows))
	for i, r := range rows {
		fmt.Sscanf(fmt.Sprint(r["id"]), "%d", &ids[i])
	}
	return ids
}

func bulkInsertExt(ctx context.Context, tx orm.TxOrmer, table string, cols []string, rows [][]any) {
	if len(rows) == 0 {
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "INSERT INTO cmdb.%s (%s) VALUES ", table, strings.Join(cols, ","))
	args := make([]any, 0, len(rows)*len(cols))
	ph := "(" + strings.TrimSuffix(strings.Repeat("?,", len(cols)), ",") + ")"
	for i, r := range rows {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(ph)
		args = append(args, r...)
	}
	if _, err := tx.RawWithCtx(ctx, sb.String(), args...).Exec(); err != nil {
		fatalf("bulkInsertExt %s: %v", table, err)
	}
}
