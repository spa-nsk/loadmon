package main

import (

	"loadmon/locality"
	"loadmon/models"
)

// Directory - справочник населённых пунктов, загруженный из файла ОКТМО.
type Directory struct {
	byName map[string][]*models.Locality
	cities []*models.Locality // только "г" - для поиска с опечаткой
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
		k := locality.Norm(l.Name)
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
func (d *Directory) Find(name string, fuzzy bool) *locality.Match {
	k := locality.Norm(name)
	if k == "" {
		return nil
	}
	if c := d.byName[k]; len(c) > 0 {
		return &locality.Match{Loc: locality.Best(c), Candidates: len(c)}
	}
	if !fuzzy {
		return nil
	}
	var found []*models.Locality
	for _, l := range d.cities {
		if locality.OneEdit(k, locality.Norm(l.Name)) {
			found = append(found, l)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return &locality.Match{Loc: locality.Best(found), Candidates: len(found), Fuzzy: true}
}
