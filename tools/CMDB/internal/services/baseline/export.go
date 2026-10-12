package baseline

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/xuri/excelize/v2"

	"cmdb/internal/repo"
	"cmdb/internal/services/svcerr"
)

// Manifest — паспорт выгрузки среза (plan.md E6.4): что выгружено, за какой момент и по какому фильтру,
// сколько позиций и контрольная сумма для проверки целостности при аудите.
type Manifest struct {
	BaselineID  string         `json:"baseline_id"`
	Name        string         `json:"name"`
	Status      string         `json:"status"`
	AsOf        string         `json:"as_of_utc"`
	Scope       Scope          `json:"scope_filter"`
	ItemCount   string         `json:"item_count"`
	ByEntity    map[string]int `json:"items_by_entity"`
	Checksum    string         `json:"checksum_sha256"`
	FrozenAt    string         `json:"frozen_at_utc,omitempty"`
	ApprovedAt  string         `json:"approved_at_utc,omitempty"`
	GeneratedAt string         `json:"generated_at_utc"`
}

// Item — позиция выгрузки: ссылка на версию строки и её содержимое.
type Item struct {
	Entity     string         `json:"entity"`
	EntityID   string         `json:"entity_id"`
	CIID       string         `json:"ci_id"`
	CIName     string         `json:"ci_name"`
	RowVersion string         `json:"row_version"`
	RowHash    string         `json:"row_hash"`
	Data       map[string]any `json:"data"`
}

// Formats — поддерживаемые форматы выгрузки.
var Formats = []string{"csv", "xlsx", "json"}

func manifestFor(ctx context.Context, b Baseline) (Manifest, error) {
	by, err := Count(ctx, mustID(b.ID))
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{BaselineID: b.ID, Name: b.Name, Status: b.Status, AsOf: b.AsOf, Scope: b.Scope(), ItemCount: b.ItemCount, ByEntity: by,
		Checksum: b.Checksum, FrozenAt: b.FrozenAt, ApprovedAt: b.ApprovedAt, GeneratedAt: time.Now().UTC().Format("2006-01-02 15:04:05")}, nil
}

func mustID(s string) int64 { id, _ := strconv.ParseInt(s, 10, 64); return id }

// eachItem обходит позиции среза с содержимым версий порциями (keyset по entity_id) — выгрузка
// не держит в памяти весь срез (до ~500 тыс. позиций, intent §9).
func eachItem(ctx context.Context, id int64, fn func(Item) error) error {
	for _, entity := range Entities() {
		pk, _, _, _ := entityInfo(entity)
		var last int64 = -1
		for {
			rows, err := repo.QueryRows(ctx, fmt.Sprintf(`
				SELECT i.entity_id::text AS entity_id, COALESCE(i.ci_id::text, '') AS ci_id, i.row_version::text AS row_version, i.row_hash,
				       COALESCE((SELECT name FROM cmdb.ci WHERE id = i.ci_id), '') AS ci_name, x.j::text AS data
				FROM rpt.baseline_item i
				JOIN LATERAL (
					SELECT j FROM (
						SELECT to_jsonb(t) - 'sys_period' AS j FROM cmdb.%[1]s t WHERE t.%[2]s = i.entity_id AND t.row_version = i.row_version
						UNION ALL
						SELECT to_jsonb(h)%[3]s FROM hist.%[1]s h WHERE h.%[2]s = i.entity_id AND h.row_version = i.row_version
					) v LIMIT 1) x ON true
				WHERE i.baseline_id = ? AND i.entity = ? AND i.entity_id > ?
				ORDER BY i.entity_id LIMIT 5000`, entity, pk, histCols), id, entity, last)
			if err != nil {
				return fmt.Errorf("baseline: выгрузка %s: %w", entity, err)
			}
			for _, r := range rows {
				it := Item{Entity: entity, EntityID: r["entity_id"], CIID: r["ci_id"], CIName: r["ci_name"], RowVersion: r["row_version"], RowHash: r["row_hash"]}
				if err := json.Unmarshal([]byte(r["data"]), &it.Data); err != nil {
					return err
				}
				if err := fn(it); err != nil {
					return err
				}
				last, _ = strconv.ParseInt(r["entity_id"], 10, 64)
			}
			if len(rows) < 5000 {
				break
			}
		}
	}
	return nil
}

// Export пишет срез в w в формате format (csv | xlsx | json) с манифестом и возвращает манифест.
// Выгружать можно только замороженный или утверждённый срез — у черновика нет контрольной суммы,
// а «доказательная база для аудита» не должна меняться. CSV: манифест — строки-комментарии «# ключ: значение».
func Export(ctx context.Context, id int64, format string, w io.Writer) (Manifest, error) {
	b, err := Get(ctx, id)
	if err != nil {
		return Manifest{}, err
	}
	if b.Status == "draft" {
		return Manifest{}, fmt.Errorf("%w: черновик выгружать нельзя — заморозьте срез", svcerr.ErrInvalidState)
	}
	m, err := manifestFor(ctx, b)
	if err != nil {
		return m, err
	}
	switch format {
	case "json":
		return m, exportJSON(ctx, id, m, w)
	case "csv":
		return m, exportCSV(ctx, id, m, w)
	case "xlsx":
		return m, exportXLSX(ctx, id, m, w)
	}
	return m, fmt.Errorf("%w: формат %q (csv, xlsx, json)", svcerr.ErrInvalidInput, format)
}

func manifestPairs(m Manifest) [][2]string {
	scope, _ := json.Marshal(m.Scope)
	by, _ := json.Marshal(m.ByEntity)
	return [][2]string{{"baseline_id", m.BaselineID}, {"name", m.Name}, {"status", m.Status}, {"as_of_utc", m.AsOf}, {"scope_filter", string(scope)},
		{"item_count", m.ItemCount}, {"items_by_entity", string(by)}, {"checksum_sha256", m.Checksum}, {"frozen_at_utc", m.FrozenAt},
		{"approved_at_utc", m.ApprovedAt}, {"generated_at_utc", m.GeneratedAt}}
}

var itemHeader = []string{"entity", "entity_id", "ci_id", "ci_name", "row_version", "row_hash", "data_json"}

func exportJSON(ctx context.Context, id int64, m Manifest, w io.Writer) error {
	mb, _ := json.MarshalIndent(m, "  ", "  ")
	if _, err := fmt.Fprintf(w, "{\n  \"manifest\": %s,\n  \"items\": [", mb); err != nil {
		return err
	}
	first := true
	if err := eachItem(ctx, id, func(it Item) error {
		b, err := json.Marshal(it)
		if err != nil {
			return err
		}
		sep := ",\n    "
		if first {
			sep, first = "\n    ", false
		}
		_, err = fmt.Fprintf(w, "%s%s", sep, b)
		return err
	}); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n  ]\n}\n")
	return err
}

func itemRow(it Item) []string {
	data, _ := json.Marshal(it.Data)
	return []string{it.Entity, it.EntityID, it.CIID, it.CIName, it.RowVersion, it.RowHash, string(data)}
}

func exportCSV(ctx context.Context, id int64, m Manifest, w io.Writer) error {
	for _, p := range manifestPairs(m) {
		if _, err := fmt.Fprintf(w, "# %s: %s\n", p[0], p[1]); err != nil {
			return err
		}
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(itemHeader); err != nil {
		return err
	}
	if err := eachItem(ctx, id, func(it Item) error { return cw.Write(itemRow(it)) }); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func exportXLSX(ctx context.Context, id int64, m Manifest, w io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()
	const sheetM, sheetD = "Манифест", "Данные"
	f.SetSheetName("Sheet1", sheetM)
	for i, p := range manifestPairs(m) {
		_ = f.SetSheetRow(sheetM, "A"+strconv.Itoa(i+1), &[]any{p[0], p[1]})
	}
	if _, err := f.NewSheet(sheetD); err != nil {
		return err
	}
	sw, err := f.NewStreamWriter(sheetD)
	if err != nil {
		return err
	}
	hdr := make([]any, len(itemHeader))
	for i, h := range itemHeader {
		hdr[i] = h
	}
	if err := sw.SetRow("A1", hdr); err != nil {
		return err
	}
	row := 2
	if err := eachItem(ctx, id, func(it Item) error {
		r := itemRow(it)
		cells := make([]any, len(r))
		for i, v := range r {
			if len(v) > 32000 { // предел ячейки Excel — 32 767 символов
				v = v[:32000]
			}
			cells[i] = v
		}
		cell, _ := excelize.CoordinatesToCellName(1, row)
		row++
		return sw.SetRow(cell, cells)
	}); err != nil {
		return err
	}
	if err := sw.Flush(); err != nil {
		return err
	}
	return f.Write(w)
}

// CompareCSV пишет отчёт сравнения в CSV: по строке на каждое различие поля (для added/removed — одна строка).
func CompareCSV(res Result, w io.Writer) error {
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"# A", res.LabelA})
	_ = cw.Write([]string{"# B", res.LabelB})
	_ = cw.Write([]string{"# добавлено / удалено / изменено", fmt.Sprintf("%d / %d / %d", res.Added, res.Removed, res.Changed)})
	if res.Truncated {
		_ = cw.Write([]string{"# внимание", fmt.Sprintf("подробно показаны первые %d различий, счётчики полные", MaxDetails)})
	}
	_ = cw.Write([]string{"entity", "entity_id", "ci_id", "ci_name", "kind", "field", "a", "b"})
	for _, c := range res.Changes {
		base := []string{c.Entity, strconv.FormatInt(c.EntityID, 10), strconv.FormatInt(c.CIID, 10), c.CIName, c.Kind}
		if len(c.Fields) == 0 {
			_ = cw.Write(append(base, "", "", ""))
			continue
		}
		for _, f := range c.Fields {
			_ = cw.Write(append(append([]string{}, base...), f.Name, f.A, f.B))
		}
	}
	cw.Flush()
	return cw.Error()
}
