package locality

import (
	"regexp"
	"strconv"
	"strings"
)

// Location - разобранное значение поля "Местоположение".
type Location struct {
	Raw         string
	Description string // описание (всё, что не населённый пункт/улица/дом)
	Locality    string // населённый пункт как написан в файле
	Street      string
	House       string
	Floor       string
	Room        string // помещение (несколько сегментов склеены через " | ")
	Cabinet     string // шкаф из "Местоположения" (опционально)
	Pos         *Position
	Service     bool // служебная отметка (например "! Покинуло ФГ ..."), разбору не подлежит
}

// Address - адрес площадки: улица и дом через запятую.
func (l *Location) Address() string {
	switch {
	case l.Street != "" && l.House != "":
		return l.Street + ", " + l.House
	case l.Street != "":
		return l.Street
	default:
		return l.House
	}
}

// Position - положение в шкафу (юниты START..END).
type Position struct {
	Start, End int
}

func (p *Position) String() string {
	if p == nil {
		return ""
	}
	if p.Start == p.End {
		return strconv.Itoa(p.Start)
	}
	return strconv.Itoa(p.Start) + "-" + strconv.Itoa(p.End)
}

// Placement - разобранное поле "Размещение в шкафу/стойке".
type Placement struct {
	Raw     string
	AltName string // альтернативное наименование шкафа (часть до "/")
	Cabinet string
	Pos     *Position
}

var (
	reRange       = regexp.MustCompile(`^(\d+)\s*(?:-\s*(\d+))?$`)
	reUnit        = regexp.MustCompile(`(?i)^unit\s*(\d+)\s*(?:-\s*(\d+))?$`)
	reCabinet     = regexp.MustCompile(`(?i)^шкаф\s+(.+)$`)
	reFloor       = regexp.MustCompile(`^-?\d+(\.\s*\S.*)?$`)
	reHouseTail   = regexp.MustCompile(`^(.*?\S)\s+(\d[\d/\-]*[а-яa-zА-ЯA-Z]?(?:\s.*)?)$`)
	reParenPrefix = regexp.MustCompile(`^\(([^)]*)\)\s*(.+)$`)
	reHouseHead   = regexp.MustCompile(`^(?i:д\.?\s*)?\d`)
)

func parseRange(a, b string) *Position {
	s, _ := strconv.Atoi(a)
	e := s
	if b != "" {
		e, _ = strconv.Atoi(b)
	}
	return &Position{Start: s, End: e}
}

func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParsePlacement разбирает "<альт. наименование> / <шкаф> | START-END".
func ParsePlacement(s string) *Placement {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	pl := &Placement{Raw: s}
	segs := splitTrim(s, "|")
	left := ""
	if len(segs) > 0 {
		left = segs[0]
	}
	if len(segs) > 1 {
		if m := reRange.FindStringSubmatch(segs[1]); m != nil {
			pl.Pos = parseRange(m[1], m[2])
		} else if m := reUnit.FindStringSubmatch(segs[1]); m != nil {
			pl.Pos = parseRange(m[1], m[2])
		}
	}
	if i := strings.LastIndex(left, "/"); i >= 0 {
		pl.AltName = strings.TrimSpace(left[:i])
		pl.Cabinet = strings.TrimSpace(left[i+1:])
	} else {
		pl.Cabinet = left
	}
	return pl
}

// ParseLocation разбирает "описание, нас.пункт, улица, дом | этаж | помещение [| Шкаф X [| Unit N]]".
// resolve сообщает, является ли токен населённым пунктом.
func ParseLocation(s string, isLocality func(string) bool) *Location {
	s = strings.TrimSpace(s)
	loc := &Location{Raw: s}
	if s == "" {
		return loc
	}
	if strings.HasPrefix(s, "!") {
		loc.Service = true
		loc.Description = strings.TrimSpace(strings.TrimLeft(s, "! "))
		return loc
	}
	segs := splitTrim(s, "|")
	tokens := splitTrim(segs[0], ",")

	// населённый пункт - первый из первых трёх токенов, найденный в справочнике
	li := -1
	for i := 0; i < len(tokens) && i < 3; i++ {
		if isLocality(tokens[i]) {
			li = i
			break
		}
	}
	var desc []string
	rest := tokens
	if li >= 0 {
		desc = append(desc, tokens[:li]...)
		loc.Locality = tokens[li]
		rest = tokens[li+1:]
	} else if len(tokens) > 0 {
		// не найден в справочнике: считаем населённым пунктом первый токен,
		// если перед ним нет явного описания в скобках
		loc.Locality = tokens[0]
		rest = tokens[1:]
	}

	// улица и дом
	hi := -1
	for i, t := range rest {
		if reHouseHead.MatchString(t) {
			hi = i
			break
		}
	}
	switch {
	case hi >= 1:
		desc = append(desc, rest[:hi-1]...)
		loc.Street = rest[hi-1]
		loc.House = strings.Join(rest[hi:], ", ")
	case hi == 0:
		loc.House = strings.Join(rest, ", ")
	default:
		// дома отдельным токеном нет - пробуем "Менделеева 1" в последнем токене
		if n := len(rest); n > 0 {
			if m := reHouseTail.FindStringSubmatch(rest[n-1]); m != nil {
				desc = append(desc, rest[:n-1]...)
				loc.Street, loc.House = m[1], m[2]
			} else {
				// нет номера дома - последний токен считаем улицей/названием
				// здания (например "БЦ (на Советской)"), остальные - описание
				desc = append(desc, rest[:n-1]...)
				loc.Street = rest[n-1]
			}
		}
	}
	// "(ЦОД DSP) Шарикоподшипниковская улица" - текст в скобках перед улицей
	// относится к описанию
	if m := reParenPrefix.FindStringSubmatch(loc.Street); m != nil {
		desc = append(desc, m[1])
		loc.Street = m[2]
	}
	loc.Description = strings.Join(desc, ", ")

	// этаж | помещение | шкаф | unit
	var room []string
	for i, sg := range segs[1:] {
		switch {
		case reUnit.MatchString(sg):
			m := reUnit.FindStringSubmatch(sg)
			loc.Pos = parseRange(m[1], m[2])
		case reCabinet.MatchString(sg):
			loc.Cabinet = reCabinet.FindStringSubmatch(sg)[1]
		case i == 0 && reFloor.MatchString(sg):
			loc.Floor = sg
		default:
			if len(room) == 0 || room[len(room)-1] != sg {
				room = append(room, sg)
			}
		}
	}
	loc.Room = strings.Join(room, ", ")
	return loc
}
