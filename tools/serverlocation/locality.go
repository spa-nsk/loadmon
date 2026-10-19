package main

import (
	"strings"

	"loadmon/locality"
	"loadmon/models"
)

// Directory - справочник населённых пунктов, загруженный из файла ОКТМО.
type Directory struct {
	byName map[string][]*models.Locality
	cities []*models.Locality // только "г" - для поиска с опечаткой
}

// Match - результат поиска населённого пункта.
type Match struct {
	Loc        *models.Locality
	Candidates int  // сколько записей справочника имеют такое же название
	Fuzzy      bool // найден приблизительно (расхождение в 1 букву)
}

func norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.ReplaceAll(s, "ё", "е")
}

// LoadDirectory читает справочник из csv ОКТМО (без БД).
func LoadDirectory(path string) (*Directory, error) {
	list, err := locality.Parse(path)
	if err != nil {
		return nil, err
	}
	d := &Directory{byName: make(map[string][]*models.Locality, len(list))}
	for i := range list {
		l := &list[i]
		k := norm(l.Name)
		d.byName[k] = append(d.byName[k], l)
		if l.TypeShort == "г" {
			d.cities = append(d.cities, l)
		}
	}
	return d, nil
}

// Size возвращает число записей справочника.
func (d *Directory) Size() int {
	n := 0
	for _, v := range d.byName {
		n += len(v)
	}
	return n
}

// Find ищет населённый пункт по названию. При нескольких одноимённых
// предпочитается город ("г"), затем меньший код ОКТМО.
func (d *Directory) Find(name string, fuzzy bool) *Match {
	k := norm(name)
	if k == "" {
		return nil
	}
	if c := d.byName[k]; len(c) > 0 {
		return &Match{Loc: best(c), Candidates: len(c)}
	}
	if !fuzzy {
		return nil
	}
	var found []*models.Locality
	for _, l := range d.cities {
		if oneEdit(k, norm(l.Name)) {
			found = append(found, l)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return &Match{Loc: best(found), Candidates: len(found), Fuzzy: true}
}

func best(c []*models.Locality) *models.Locality {
	b := c[0]
	for _, l := range c[1:] {
		if (l.TypeShort == "г") != (b.TypeShort == "г") {
			if l.TypeShort == "г" {
				b = l
			}
			continue
		}
		if l.Oktmo < b.Oktmo {
			b = l
		}
	}
	return b
}

// oneEdit - расстояние Левенштейна между a и b ровно 1 (замена/вставка/удаление).
func oneEdit(a, b string) bool {
	ra, rb := []rune(a), []rune(b)
	if len(ra) < len(rb) {
		ra, rb = rb, ra
	}
	if len(ra)-len(rb) > 1 || len(rb) < 4 {
		return false
	}
	i := 0
	for i < len(rb) && ra[i] == rb[i] {
		i++
	}
	if i == len(rb) {
		return len(ra) != len(rb)
	}
	if len(ra) == len(rb) {
		return string(ra[i+1:]) == string(rb[i+1:])
	}
	return string(ra[i+1:]) == string(rb[i:])
}
