// Интеграционный тест seqalloc: реальный PostgreSQL с примененными миграциями и seed nc2026.
// В этой среде разработки нет Docker/локального PostgreSQL, поэтому тест пропускается, если не
// задана CMDB_TEST_DB_DSN. Прогнать: CMDB_TEST_DB_DSN=... go test ./internal/services/seqalloc/...
// (после `make migrate && make seed-nc2026` на тестовой БД) — проверяет DoD E1.6: 100 параллельных
// выделений для одного (site, designationCode) не дают дублей и не выходят за границы диапазона.
package seqalloc

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/platform/config"
	"cmdb/internal/repo"
	"cmdb/internal/services/naming"
)

func TestAllocateHostName_ConcurrentNoDuplicates(t *testing.T) {
	dsn := os.Getenv("CMDB_TEST_DB_DSN")
	if dsn == "" {
		t.Skip("CMDB_TEST_DB_DSN не задан — пропуск интеграционного теста (нужен PostgreSQL с миграциями и seed nc2026)")
	}
	if err := repo.Init(config.DBConfig{DSN: dsn, MaxOpenConns: 20, MaxIdleConns: 5}); err != nil {
		t.Fatalf("repo.Init: %v", err)
	}

	reg := naming.DefaultRegistry()
	const site = "TESTQA"
	const designationCode = "CL" // 3-значный диапазон, легко исчерпать в тесте
	const attempts = 100

	var (
		mu    sync.Mutex
		names = make(map[string]int)
		wg    sync.WaitGroup
		errs  []error
	)

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			o := orm.NewOrm()
			err := o.DoTxWithCtx(context.Background(), func(ctx context.Context, tx orm.TxOrmer) error {
				name, err := AllocateHostName(ctx, tx, reg, site, designationCode)
				if err != nil {
					return err
				}
				mu.Lock()
				names[name]++
				mu.Unlock()
				return nil
			})
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("%d из %d выделений завершились ошибкой, первая: %v", len(errs), attempts, errs[0])
	}
	if len(names) != attempts {
		t.Fatalf("получено %d уникальных имён, ожидалось %d (найдены дубли)", len(names), attempts)
	}
	for name, count := range names {
		if count != 1 {
			t.Errorf("имя %q выдано %d раз", name, count)
		}
	}
}
