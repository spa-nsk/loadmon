// E6.5/E6.6: закрытый учётный период отклоняет правки; администратор учёта может править с причиной.
package period

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/beego/beego/v2/client/orm"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/svcerr"
	"cmdb/internal/testutil"
)

// oldMonth — месяц в далёком прошлом, который никто не использует (тесты делят БД): каждый прогон
// берёт свой, чтобы закрытие не мешало соседним тестам и не зависело от их состояния.
func oldMonth() time.Time {
	n := time.Now().UnixNano()
	return time.Date(1950+int(n/12%60), time.Month(1+int(n%12)), 1, 0, 0, 0, 0, time.UTC)
}

func TestClosedPeriodRejectsEditsAndAdminMayBackdate(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	loc := testutil.NewLocation(t, db)
	res, err := ci.CreateServer(ctx, naming.DefaultRegistry(), ci.CreateServerInput{LocationCode: loc, Reason: "фикстура", UserID: 1})
	if err != nil {
		t.Fatal(err)
	}

	month := oldMonth()
	inMonth := month.AddDate(0, 0, 14)
	if err := Close(ctx, month, 1, "закрытие для теста"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM rpt.accounting_period WHERE period = $1`, month) })
	if err := Close(ctx, month, 1, ""); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Fatalf("повторное закрытие: err = %v, want ErrInvalidState", err)
	}
	if closed, err := IsClosed(ctx, inMonth); err != nil || !closed {
		t.Fatalf("IsClosed = %v, %v, want true", closed, err)
	}

	edit := func(meta changectx.Meta) error {
		_, err := changectx.Run(ctx, meta, func(ctx context.Context, tx orm.TxOrmer) error {
			_, err := tx.RawWithCtx(ctx, `UPDATE cmdb.ci SET description = 'правка', updated_at = now() WHERE id = ?`, res.CIID).Exec()
			return err
		})
		return err
	}

	// 1. обычная правка с датой учёта в закрытом месяце — отклонена
	if err := edit(changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "правка", EffectiveAt: &inMonth}); !errors.Is(err, svcerr.ErrPeriodClosed) {
		t.Fatalf("правка в закрытом периоде: err = %v, want ErrPeriodClosed", err)
	}
	// 2. администратор учёта без причины — отклонена
	if err := edit(changectx.Meta{Source: changectx.SourceImport, UserID: 1, EffectiveAt: &inMonth, AllowClosedPeriod: true}); !errors.Is(err, svcerr.ErrPeriodClosed) {
		t.Fatalf("правка администратора без причины: err = %v, want ErrPeriodClosed", err)
	}
	// 3. администратор учёта с причиной — проходит, change_set помечен задним числом
	if err := edit(changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "корректировка учёта", EffectiveAt: &inMonth, AllowClosedPeriod: true}); err != nil {
		t.Fatalf("правка администратора с причиной: %v", err)
	}
	var backdated bool
	db.QueryRow(`SELECT is_backdated FROM audit.change_set WHERE reason = 'корректировка учёта' ORDER BY id DESC LIMIT 1`).Scan(&backdated)
	if !backdated {
		t.Error("change_set правки закрытого периода должен быть помечен is_backdated")
	}

	// 4. отчёт о правках задним числом видит эту правку и отмечает закрытый период
	rep, err := BackdatedReport(ctx, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rep {
		if r.Reason == "корректировка учёта" && r.ClosedPeriod {
			found = true
		}
	}
	if !found {
		t.Errorf("отчёт о правках задним числом не содержит правку закрытого периода: %+v", rep)
	}

	// 5. после переоткрытия обычная правка проходит и тоже считается заднее-числовой (дата учёта в прошлом)
	if err := Reopen(ctx, month, 1, ""); !errors.Is(err, svcerr.ErrInvalidInput) {
		t.Fatalf("переоткрытие без причины: err = %v, want ErrInvalidInput", err)
	}
	if err := Reopen(ctx, month, 1, "ошибка в закрытии"); err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if err := edit(changectx.Meta{Source: changectx.SourceManual, UserID: 1, Reason: "правка после переоткрытия", EffectiveAt: &inMonth}); err != nil {
		t.Fatalf("правка в переоткрытом периоде: %v", err)
	}
	db.QueryRow(`SELECT is_backdated FROM audit.change_set WHERE reason = 'правка после переоткрытия' ORDER BY id DESC LIMIT 1`).Scan(&backdated)
	if !backdated {
		t.Error("дата учёта в прошлом должна ставить is_backdated автоматически")
	}
	if err := Reopen(ctx, month, 1, "ещё раз"); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Fatalf("переоткрытие открытого: err = %v, want ErrInvalidState", err)
	}
}

func TestCloseRejectsCurrentAndFutureMonth(t *testing.T) {
	testutil.DB(t)
	ctx := context.Background()
	if err := Close(ctx, time.Now(), 1, ""); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("закрытие текущего месяца: err = %v, want ErrInvalidState (закрыть можно только закончившийся)", err)
	}
	if err := Close(ctx, time.Now().AddDate(0, 2, 0), 1, ""); !errors.Is(err, svcerr.ErrInvalidState) {
		t.Errorf("закрытие будущего месяца: err = %v, want ErrInvalidState", err)
	}
}

func TestParseMonth(t *testing.T) {
	for _, in := range []string{"2026-03", "2026-03-17"} {
		m, err := ParseMonth(in)
		if err != nil || !m.Equal(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("ParseMonth(%q) = %v, %v", in, m, err)
		}
	}
	if _, err := ParseMonth("март"); err == nil {
		t.Error("ParseMonth(мусор) должен вернуть ошибку")
	}
}
