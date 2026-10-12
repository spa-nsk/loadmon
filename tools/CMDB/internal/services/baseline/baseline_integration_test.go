// E6.2/E6.6: реальный PostgreSQL. Срез на момент X совпадает с AS OF X, замороженный срез неизменяем
// (триггеры БД) и проверяем на подмену истории.
package baseline

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"cmdb/internal/services/ci"
	"cmdb/internal/services/naming"
	"cmdb/internal/testutil"
)

func mkServer(t *testing.T, loc string) int64 {
	t.Helper()
	res, err := ci.CreateServer(context.Background(), naming.DefaultRegistry(), ci.CreateServerInput{
		LocationCode: loc, Reason: "фикстура среза", UserID: 1,
	})
	if err != nil {
		t.Fatalf("CreateServer: %v", err)
	}
	return res.CIID
}

// itemSet читает позиции среза по сущности как множество "id:версия".
func itemSet(t *testing.T, db *sql.DB, baselineID int64, entity string) map[string]bool {
	t.Helper()
	rows, err := db.Query(`SELECT entity_id, row_version FROM rpt.baseline_item WHERE baseline_id = $1 AND entity = $2`, baselineID, entity)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id int64
		var v int
		rows.Scan(&id, &v)
		out[fmt.Sprintf("%d:%d", id, v)] = true
	}
	return out
}

func asOfSet(t *testing.T, db *sql.DB, ts time.Time, loc string) map[string]bool {
	t.Helper()
	rows, err := db.Query(`SELECT id, row_version FROM cmdb.ci_asof($1) WHERE location_code = $2`, ts, loc)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id int64
		var v int
		rows.Scan(&id, &v)
		out[fmt.Sprintf("%d:%d", id, v)] = true
	}
	return out
}

func TestBaseline_MatchesAsOfAndIsImmutable(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	loc := testutil.NewLocation(t, db)

	a, b, c := mkServer(t, loc), mkServer(t, loc), mkServer(t, loc)
	time.Sleep(50 * time.Millisecond)
	t1 := time.Now()
	time.Sleep(50 * time.Millisecond)

	name := "срез-" + testutil.Suffix()
	id, err := Create(ctx, name, "тест", t1, Scope{LocationCode: loc}, 1)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// после среза: правим A и добавляем D — срез на t1 этого видеть не должен
	desc := "изменено после среза"
	if _, err := ci.Update(ctx, ci.UpdateInput{CIID: a, ExpectedVersion: 1, Reason: "правка после среза", UserID: 1, Description: &desc}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	mkServer(t, loc)

	got, want := itemSet(t, db, id, "ci"), asOfSet(t, db, t1, loc)
	if len(got) != 3 || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("позиции ci среза = %v, AS OF t1 = %v — должны совпадать (3 КЕ: %d, %d, %d)", got, want, a, b, c)
	}
	// ext-строки этих КЕ тоже в срезе
	if n := len(itemSet(t, db, id, "ci_server")); n != 3 {
		t.Errorf("позиций ci_server = %d, want 3", n)
	}

	bl, err := Get(ctx, id)
	if err != nil || bl.Status != "draft" || bl.ItemCount == "0" {
		t.Fatalf("срез = %+v err=%v", bl, err)
	}
	if err := Freeze(ctx, id, 1); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	bl, _ = Get(ctx, id)
	if bl.Status != "frozen" || len(bl.Checksum) != 64 {
		t.Fatalf("после заморозки: %+v (checksum — SHA-256, 64 hex)", bl)
	}
	if err := Freeze(ctx, id, 1); err == nil {
		t.Error("повторная заморозка должна вернуть ошибку")
	}

	v, err := Verify(ctx, id)
	if err != nil || !v.OK {
		t.Fatalf("Verify целого среза = %+v err=%v, want OK (версия A ушла в hist — хеш содержимого прежний)", v, err)
	}

	// неизменяемость на уровне БД
	if _, err := db.Exec(`DELETE FROM rpt.baseline_item WHERE baseline_id = $1`, id); err == nil {
		t.Error("удаление позиций замороженного среза должно быть запрещено триггером")
	}
	if _, err := db.Exec(`UPDATE rpt.baseline SET name = 'другое' WHERE id = $1`, id); err == nil {
		t.Error("переименование замороженного среза должно быть запрещено триггером")
	}
	if _, err := db.Exec(`UPDATE rpt.baseline SET checksum = 'x' WHERE id = $1`, id); err == nil {
		t.Error("подмена контрольной суммы должна быть запрещена триггером")
	}
	if err := Delete(ctx, id, 1); err == nil {
		t.Error("замороженный срез удалять нельзя")
	}

	// подмена истории мимо приложения (владелец БД может править hist) обнаруживается проверкой
	if _, err := db.Exec(`UPDATE hist.ci SET description = 'подмена' WHERE id = $1 AND row_version = 1`, a); err != nil {
		t.Fatalf("подготовка подмены: %v", err)
	}
	v, err = Verify(ctx, id)
	if err != nil || v.OK || len(v.Problems) == 0 {
		t.Fatalf("Verify после подмены истории = %+v err=%v, want проблемы", v, err)
	}

	if err := Approve(ctx, id, 1); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if bl, _ = Get(ctx, id); bl.Status != "approved" {
		t.Errorf("статус = %s, want approved", bl.Status)
	}
	if err := Approve(ctx, id, 1); err == nil {
		t.Error("повторное утверждение должно вернуть ошибку")
	}
}

func TestBaseline_DraftDeleteAndValidation(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	loc := testutil.NewLocation(t, db)
	mkServer(t, loc)
	time.Sleep(30 * time.Millisecond)

	name := "черновик-" + testutil.Suffix()
	id, err := Create(ctx, name, "", time.Now(), Scope{LocationCode: loc}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Create(ctx, name, "", time.Now(), Scope{}, 1); err == nil {
		t.Error("повторное имя среза должно быть отклонено")
	}
	if _, err := Create(ctx, "", "", time.Now(), Scope{}, 1); err == nil {
		t.Error("пустое имя среза должно быть отклонено")
	}
	if _, err := Create(ctx, "будущее-"+testutil.Suffix(), "", time.Now().Add(time.Hour), Scope{}, 1); err == nil {
		t.Error("срез на будущий момент должен быть отклонён")
	}
	if _, err := Verify(ctx, id); err == nil {
		t.Error("Verify черновика должен вернуть ошибку")
	}
	if err := Approve(ctx, id, 1); err == nil {
		t.Error("нельзя утвердить незамороженный срез")
	}
	if err := Delete(ctx, id, 1); err != nil {
		t.Fatalf("удаление черновика: %v", err)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM rpt.baseline_item WHERE baseline_id = $1`, id).Scan(&n)
	if n != 0 {
		t.Errorf("позиции удалённого черновика остались: %d", n)
	}
}

// Область среза: фильтр по классу отсекает прочие КЕ.
func TestBaseline_ScopeFilterByClass(t *testing.T) {
	db := testutil.DB(t)
	ctx := context.Background()
	loc := testutil.NewLocation(t, db)
	mkServer(t, loc)
	time.Sleep(30 * time.Millisecond)

	id, err := Create(ctx, "класс-"+testutil.Suffix(), "", time.Now(), Scope{LocationCode: loc, Classes: []string{"vm"}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(itemSet(t, db, id, "ci")); n != 0 {
		t.Errorf("срез по классу vm в площадке с одними серверами содержит %d КЕ, want 0", n)
	}
}
