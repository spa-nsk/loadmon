// E6.3/E6.4/E6.6: сравнение срезов и выгрузка с манифестом на реальном PostgreSQL.
package baseline

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"cmdb/internal/services/ci"
	"cmdb/internal/testutil"
)

// twoStates готовит площадку: 3 КЕ, срез B1; затем правка КЕ A и новая КЕ D, срез B2. Возвращает id срезов,
// моменты и КЕ.
type fixture struct {
	loc    string
	a, d   int64
	b1, b2 int64
	t1, t2 time.Time
}

func prepare(t *testing.T) fixture {
	t.Helper()
	db := testutil.DB(t)
	ctx := context.Background()
	var f fixture
	f.loc = testutil.NewLocation(t, db)
	f.a = mkServer(t, f.loc)
	mkServer(t, f.loc)
	mkServer(t, f.loc)
	time.Sleep(50 * time.Millisecond)
	f.t1 = time.Now()
	time.Sleep(50 * time.Millisecond)

	var err error
	if f.b1, err = Create(ctx, "b1-"+testutil.Suffix(), "", f.t1, Scope{LocationCode: f.loc}, 1); err != nil {
		t.Fatal(err)
	}
	desc := "описание после первого среза"
	if _, err := ci.Update(ctx, ci.UpdateInput{CIID: f.a, ExpectedVersion: 1, Reason: "правка", UserID: 1, Description: &desc}); err != nil {
		t.Fatal(err)
	}
	f.d = mkServer(t, f.loc)
	time.Sleep(50 * time.Millisecond)
	f.t2 = time.Now()
	time.Sleep(50 * time.Millisecond)
	if f.b2, err = Create(ctx, "b2-"+testutil.Suffix(), "", f.t2, Scope{LocationCode: f.loc}, 1); err != nil {
		t.Fatal(err)
	}
	return f
}

func row(res Result, entity string) SummaryRow {
	for _, r := range res.Summary {
		if r.Entity == entity {
			return r
		}
	}
	return SummaryRow{}
}

func checkDiff(t *testing.T, what string, res Result, f fixture) {
	t.Helper()
	if r := row(res, "ci"); r.Added != 1 || r.Changed != 1 || r.Removed != 0 {
		t.Errorf("%s: ci = %+v, want добавлена D, изменена A, ничего не удалено", what, r)
	}
	if r := row(res, "ci_server"); r.Added != 1 || r.Removed != 0 {
		t.Errorf("%s: ci_server = %+v, want добавлена строка расширения D", what, r)
	}
	found := false
	for _, c := range res.Changes {
		if c.Entity == "ci" && c.Kind == "changed" && c.EntityID == f.a {
			for _, fd := range c.Fields {
				if fd.Name == "description" && fd.A == "" && fd.B == "описание после первого среза" {
					found = true
				}
				if noiseFields[fd.Name] {
					t.Errorf("%s: в диффе служебное поле %s", what, fd.Name)
				}
			}
			if c.CIName == "" {
				t.Errorf("%s: у изменения нет имени КЕ", what)
			}
		}
	}
	if !found {
		t.Errorf("%s: нет атрибутного диффа description у КЕ A: %+v", what, res.Changes)
	}
}

func TestCompare_BaselineBaselineDateDateAndCurrent(t *testing.T) {
	f := prepare(t)
	ctx := context.Background()

	bb, err := Compare(ctx, State{BaselineID: f.b1}, State{BaselineID: f.b2}, Scope{})
	if err != nil {
		t.Fatalf("срез↔срез: %v", err)
	}
	checkDiff(t, "срез↔срез", bb, f)

	dd, err := Compare(ctx, State{AsOf: &f.t1}, State{AsOf: &f.t2}, Scope{LocationCode: f.loc})
	if err != nil {
		t.Fatalf("дата↔дата: %v", err)
	}
	checkDiff(t, "дата↔дата", dd, f)
	if dd.Added != bb.Added || dd.Changed != bb.Changed || dd.Removed != bb.Removed {
		t.Errorf("дата↔дата (%d/%d/%d) должно совпасть со срез↔срез (%d/%d/%d): срез на момент = AS OF на момент",
			dd.Added, dd.Removed, dd.Changed, bb.Added, bb.Removed, bb.Changed)
	}

	bc, err := Compare(ctx, State{BaselineID: f.b1}, State{Current: true}, Scope{})
	if err != nil {
		t.Fatalf("срез↔текущее: %v", err)
	}
	checkDiff(t, "срез↔текущее", bc, f)

	same, err := Compare(ctx, State{BaselineID: f.b2}, State{Current: true}, Scope{})
	if err != nil || same.Added+same.Removed+same.Changed != 0 {
		t.Errorf("срез B2 против текущего (после B2 ничего не менялось) = %+v err=%v, want без различий", same, err)
	}

	if _, err := Compare(ctx, State{BaselineID: 999999999}, State{Current: true}, Scope{}); err == nil {
		t.Error("несуществующий срез должен дать ошибку")
	}
}

func TestParseState(t *testing.T) {
	for in, ok := range map[string]bool{"now": true, "b:12": true, "t:2026-09-01": true, "t:2026-09-01T10:00": true,
		"t:2026-09-01T10:00:00Z": true, "b:x": false, "t:мусор": false, "": false} {
		if _, err := ParseState(in); (err == nil) != ok {
			t.Errorf("ParseState(%q): err = %v, want ok=%v", in, err, ok)
		}
	}
}

func TestExport_ManifestAndFormats(t *testing.T) {
	f := prepare(t)
	ctx := context.Background()

	var buf bytes.Buffer
	if _, err := Export(ctx, f.b1, "json", &buf); err == nil {
		t.Fatal("выгрузка черновика должна быть отклонена")
	}
	if err := Freeze(ctx, f.b1, 1); err != nil {
		t.Fatal(err)
	}
	bl, _ := Get(ctx, f.b1)
	total, _ := strconv.Atoi(bl.ItemCount)

	// JSON: манифест + позиции с содержимым
	buf.Reset()
	m, err := Export(ctx, f.b1, "json", &buf)
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	var doc struct {
		Manifest Manifest `json:"manifest"`
		Items    []Item   `json:"items"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("json невалиден: %v\n%.300s", err, buf.String())
	}
	if doc.Manifest.Checksum != bl.Checksum || len(doc.Manifest.Checksum) != 64 || doc.Manifest.ItemCount != bl.ItemCount || m.Checksum != bl.Checksum {
		t.Errorf("манифест = %+v, want checksum/количество среза", doc.Manifest)
	}
	if doc.Manifest.Scope.LocationCode != f.loc || doc.Manifest.AsOf == "" || doc.Manifest.GeneratedAt == "" {
		t.Errorf("манифест не содержит фильтр/дату: %+v", doc.Manifest)
	}
	if len(doc.Items) != total {
		t.Fatalf("позиций в выгрузке %d, want %d (item_count среза)", len(doc.Items), total)
	}
	for _, it := range doc.Items {
		if it.Data == nil || it.RowHash == "" {
			t.Fatalf("позиция без содержимого: %+v", it)
		}
	}

	// CSV: строки-комментарии манифеста + заголовок + позиции
	buf.Reset()
	if _, err := Export(ctx, f.b1, "csv", &buf); err != nil {
		t.Fatalf("csv: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if !strings.HasPrefix(lines[0], "# baseline_id: ") || !strings.Contains(buf.String(), "# checksum_sha256: "+bl.Checksum) {
		t.Errorf("CSV без манифеста-комментариев:\n%.400s", buf.String())
	}
	hdr := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "entity,entity_id") {
			hdr = i
		}
	}
	if hdr < 0 || len(lines)-hdr-1 < total {
		t.Errorf("строк данных в CSV меньше %d", total)
	}

	// XLSX: два листа, данные = позиции
	buf.Reset()
	if _, err := Export(ctx, f.b1, "xlsx", &buf); err != nil {
		t.Fatalf("xlsx: %v", err)
	}
	x, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("xlsx не открывается: %v", err)
	}
	defer x.Close()
	rows, _ := x.GetRows("Данные")
	man, _ := x.GetRows("Манифест")
	if len(rows) != total+1 || len(man) < 5 {
		t.Errorf("xlsx: строк данных %d (want %d+заголовок), строк манифеста %d", len(rows), total, len(man))
	}

	if _, err := Export(ctx, f.b1, "pdf", &buf); err == nil {
		t.Error("неизвестный формат должен быть отклонён")
	}

	// отчёт сравнения в CSV
	res, _ := Compare(ctx, State{BaselineID: f.b1}, State{BaselineID: f.b2}, Scope{})
	buf.Reset()
	if err := CompareCSV(res, &buf); err != nil || !strings.Contains(buf.String(), "description") || !strings.Contains(buf.String(), "# добавлено / удалено / изменено") {
		t.Errorf("CSV сравнения = %v\n%.400s", err, buf.String())
	}
}
