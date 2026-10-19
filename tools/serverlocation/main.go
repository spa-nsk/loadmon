// Команда serverlocation разбирает из xlsx-выгрузки серверов поля
// "Местоположение" и "Размещение в шкафу/стойке" и выводит результат на экран
// в структурированном виде. База данных не нужна: справочник населённых
// пунктов читается прямо из файла ОКТМО (см. locality.Parse).
//
//	go run ./tools/serverlocation
//	go run ./tools/serverlocation -xlsx файл.xlsx -oktmo oktmo.csv -row 15
//	go run ./tools/serverlocation -problems      # только записи с замечаниями
//	go run ./tools/serverlocation -compact       # по одной строке на запись
//	go run ./tools/serverlocation -n 20          # первые 20 записей
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"loadmon/locality"

	"github.com/xuri/excelize/v2"
)

const (
	colID        = "Тех. номер IT"
	colStatus    = "Статус"
	colHost      = "Имя (hostname)"
	colLocation  = "Местоположение"
	colPlacement = "Размещение в шкафу/стойке"
)

// Record - итог разбора одной строки xlsx.
type Record struct {
	Row       int
	ID        string
	Status    string
	Host      string
	Loc       *locality.Location
	Place     *locality.Placement
	Match     *locality.Match
	CabSource string // название поля, из которого взят шкаф
	Cabinet   string
	AltName   string
	Pos       *locality.Position
	Notes     []string
}

func main() {
	xlsx := flag.String("xlsx", `Серверы_new_2026_10_19_clear.xlsx`, "xlsx-выгрузка серверов")
	oktmo := flag.String("oktmo", `oktmo_data_20261001.csv`, "файл ОКТМО со справочником населённых пунктов")
	sheet := flag.String("sheet", "export", "лист xlsx")
	row := flag.Int("row", 0, "показать только строку xlsx с этим номером")
	limit := flag.Int("n", 0, "ограничить число выводимых записей (0 - все)")
	problems := flag.Bool("problems", false, "выводить только записи с замечаниями")
	compact := flag.Bool("compact", false, "компактный вывод - одна строка на запись")
	flag.Parse()

	dir, err := LoadDirectory(*oktmo)
	if err != nil {
		fatal("справочник населённых пунктов: %v", err)
	}
	fmt.Printf("Справочник населённых пунктов: %d записей (%s)\n", dir.Size(), *oktmo)

	f, err := excelize.OpenFile(*xlsx)
	if err != nil {
		fatal("открытие %s: %v", *xlsx, err)
	}
	defer f.Close()
	rows, err := f.GetRows(*sheet)
	if err != nil || len(rows) == 0 {
		fatal("чтение листа %q: %v", *sheet, err)
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[strings.TrimSpace(h)] = i
	}
	for _, c := range []string{colLocation, colPlacement} {
		if _, ok := idx[c]; !ok {
			fatal("в файле нет колонки %q", c)
		}
	}
	cell := func(r []string, col string) string {
		if i, ok := idx[col]; ok && i < len(r) {
			return strings.TrimSpace(r[i])
		}
		return ""
	}

	cache := map[string]*locality.Match{}
	find := func(name string) *locality.Match {
		if m, ok := cache[name]; ok {
			return m
		}
		m := dir.Find(name, true)
		cache[name] = m
		return m
	}
	isLocality := func(name string) bool { return find(name) != nil }

	var recs []*Record
	for i, r := range rows[1:] {
		rec := &Record{
			Row: i + 2, ID: cell(r, colID), Status: cell(r, colStatus), Host: cell(r, colHost),
		}
		locRaw, plRaw := cell(r, colLocation), cell(r, colPlacement)
		if locRaw == "" && plRaw == "" {
			continue
		}
		rec.Loc = locality.ParseLocation(locRaw, isLocality)
		rec.Place = locality.ParsePlacement(plRaw)
		analyze(rec, find)
		recs = append(recs, rec)
	}

	shown := 0
	for _, rec := range recs {
		if *row != 0 && rec.Row != *row {
			continue
		}
		if *problems && len(rec.Notes) == 0 {
			continue
		}
		if *limit > 0 && shown >= *limit {
			break
		}
		shown++
		if *compact {
			printCompact(rec)
		} else {
			printFull(rec)
		}
	}
	printSummary(recs, shown)
}

// analyze определяет населённый пункт и итоговый шкаф/позицию (приоритет у
// поля "Размещение в шкафу/стойке", запасной вариант - "Местоположение").
func analyze(rec *Record, find func(string) *locality.Match) {
	loc := rec.Loc
	if loc.Service {
		rec.Notes = append(rec.Notes, "служебная отметка, разбору не подлежит")
		return
	}
	if loc.Locality != "" {
		rec.Match = find(loc.Locality)
		switch {
		case rec.Match == nil:
			rec.Notes = append(rec.Notes, fmt.Sprintf("населённый пункт %q не найден в справочнике", loc.Locality))
		case rec.Match.Fuzzy:
			rec.Notes = append(rec.Notes, fmt.Sprintf("населённый пункт %q найден приблизительно как %q", loc.Locality, rec.Match.Loc.Name))
		case rec.Match.Candidates > 1:
			rec.Notes = append(rec.Notes, fmt.Sprintf("название населённого пункта неоднозначно (%d одноимённых), выбран %s", rec.Match.Candidates, rec.Match.Loc.FullName))
		}
	} else if loc.Raw != "" {
		rec.Notes = append(rec.Notes, "в местоположении не выделен населённый пункт")
	}

	switch {
	case rec.Place != nil && rec.Place.Cabinet != "":
		rec.CabSource = colPlacement
		rec.Cabinet, rec.AltName, rec.Pos = rec.Place.Cabinet, rec.Place.AltName, rec.Place.Pos
		if loc.Cabinet != "" && !strings.EqualFold(loc.Cabinet, rec.Cabinet) {
			rec.Notes = append(rec.Notes, fmt.Sprintf("шкаф в местоположении (%s) отличается от поля размещения (%s)", loc.Cabinet, rec.Cabinet))
		}
		if rec.Pos == nil && loc.Pos != nil {
			rec.Pos = loc.Pos
		}
	case loc.Cabinet != "":
		rec.CabSource = colLocation
		rec.Cabinet, rec.Pos = loc.Cabinet, loc.Pos
	}
	if rec.Cabinet != "" && rec.Pos == nil {
		rec.Notes = append(rec.Notes, "позиция в шкафу не указана")
	}
	if rec.Pos != nil && rec.Pos.Start > rec.Pos.End {
		rec.Notes = append(rec.Notes, fmt.Sprintf("начальная позиция %d больше конечной %d", rec.Pos.Start, rec.Pos.End))
	}
}

func val(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func localityText(rec *Record) string {
	if rec.Loc == nil || rec.Loc.Locality == "" {
		return "—"
	}
	if rec.Match == nil {
		return rec.Loc.Locality + "  [код: не найден]"
	}
	l := rec.Match.Loc
	s := fmt.Sprintf("%s  [код ОКТМО: %s; %s; %s]", l.FullName, l.Oktmo, l.RegionName, l.Municipality)
	if rec.Match.Fuzzy {
		s = rec.Loc.Locality + " ≈ " + s
	}
	return s
}

func posText(p *locality.Position) string {
	if p == nil {
		return "—"
	}
	return fmt.Sprintf("%d–%d", p.Start, p.End)
}

func printFull(rec *Record) {
	title := rec.ID
	if rec.Host != "" {
		title += "  " + rec.Host
	}
	fmt.Printf("\n── строка %d  %s  (%s)\n", rec.Row, title, val(rec.Status))
	fmt.Printf("   Местоположение (исх.): %s\n", val(rec.Loc.Raw))
	if rec.Place != nil {
		fmt.Printf("   Размещение (исх.):     %s\n", rec.Place.Raw)
	}
	if rec.Loc.Service {
		fmt.Printf("   Отметка:               %s\n", rec.Loc.Description)
	} else {
		fmt.Printf("   Описание:              %s\n", val(rec.Loc.Description))
		fmt.Printf("   Населённый пункт:      %s\n", localityText(rec))
		fmt.Printf("   Улица:                 %s\n", val(rec.Loc.Street))
		fmt.Printf("   Дом:                   %s\n", val(rec.Loc.House))
		fmt.Printf("   Этаж:                  %s\n", val(rec.Loc.Floor))
		fmt.Printf("   Комната (помещение):   %s\n", val(rec.Loc.Room))
	}
	fmt.Printf("   Шкаф:                  %s", val(rec.Cabinet))
	if rec.CabSource != "" {
		fmt.Printf("   (из поля «%s»)", rec.CabSource)
	}
	fmt.Println()
	fmt.Printf("   Альт. наим. шкафа:     %s\n", val(rec.AltName))
	fmt.Printf("   Позиция в шкафу:       %s\n", posText(rec.Pos))
	for _, n := range rec.Notes {
		fmt.Printf("   ⚠ %s\n", n)
	}
}

func printCompact(rec *Record) {
	code, name := "-", rec.Loc.Locality
	if rec.Match != nil {
		code, name = rec.Match.Loc.Oktmo, rec.Match.Loc.FullName
	}
	fmt.Printf("%4d | %s | %s [%s] | ул: %s | дом: %s | этаж: %s | пом: %s | шкаф: %s | поз: %s", rec.Row, rec.ID, val(name), code,
		val(rec.Loc.Street), val(rec.Loc.House), val(rec.Loc.Floor), val(rec.Loc.Room), val(rec.Cabinet), posText(rec.Pos))
	if len(rec.Notes) > 0 {
		fmt.Printf(" | ⚠ %s", strings.Join(rec.Notes, "; "))
	}
	fmt.Println()
}

func printSummary(recs []*Record, shown int) {
	var found, fuzzy, notFound, withCab, fromPlace, fromLoc, withNotes int
	missing := map[string]int{}
	byLoc := map[string]int{}
	for _, r := range recs {
		switch {
		case r.Match != nil && r.Match.Fuzzy:
			fuzzy++
		case r.Match != nil:
			found++
		case r.Loc != nil && r.Loc.Locality != "":
			notFound++
			missing[r.Loc.Locality]++
		}
		if r.Match != nil {
			byLoc[r.Match.Loc.FullName+" ["+r.Match.Loc.Oktmo+"]"]++
		}
		if r.Cabinet != "" {
			withCab++
		}
		switch r.CabSource {
		case colPlacement:
			fromPlace++
		case colLocation:
			fromLoc++
		}
		if len(r.Notes) > 0 {
			withNotes++
		}
	}
	fmt.Printf("\n══ ИТОГО ══\n")
	fmt.Printf("записей с данными о местоположении/размещении: %d (выведено: %d)\n", len(recs), shown)
	fmt.Printf("  населённый пункт найден точно: %d, приблизительно: %d, не найден: %d\n", found, fuzzy, notFound)
	fmt.Printf("  со шкафом: %d (из «%s»: %d, из «%s»: %d)\n", withCab, colPlacement, fromPlace, colLocation, fromLoc)
	fmt.Printf("  с замечаниями: %d\n", withNotes)
	if len(missing) > 0 {
		fmt.Println("  не найденные населённые пункты:")
		for _, k := range sortedKeys(missing) {
			fmt.Printf("    %s — %d\n", k, missing[k])
		}
	}
	fmt.Println("  записей по населённым пунктам:")
	for _, k := range sortedKeys(byLoc) {
		fmt.Printf("    %s — %d\n", k, byLoc[k])
	}
}

func sortedKeys(m map[string]int) []string {
	k := make([]string, 0, len(m))
	for s := range m {
		k = append(k, s)
	}
	sort.Strings(k)
	return k
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "ошибка: "+format+"\n", a...)
	os.Exit(1)
}
