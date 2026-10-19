package locality

import (
	"sort"
	"strings"

	"loadmon/models"

	"github.com/beego/beego/v2/client/orm"
)

// Match - результат поиска населённого пункта по названию.
type Match struct {
	Loc        *models.Locality
	Candidates int  // сколько записей справочника имеют такое же название
	Fuzzy      bool // найден приблизительно (расхождение в 1 букву)
}

// Norm приводит название к виду для сравнения (регистр, ё/е).
func Norm(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.ReplaceAll(s, "ё", "е")
}

// Best выбирает из одноимённых населённых пунктов город ("г"), затем меньший код ОКТМО.
func Best(c []*models.Locality) *models.Locality {
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

// OneEdit - расстояние Левенштейна между a и b ровно 1 (замена/вставка/удаление).
func OneEdit(a, b string) bool {
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

// Resolver ищет населённые пункты в загруженной в БД таблице cmdb_locality
// (результаты кэшируются на время жизни Resolver - создавайте по одному на
// импорт). Не потокобезопасен.
type Resolver struct {
	cache  map[string]*Match
	cities []*models.Locality // города - для поиска с опечаткой, загружаются лениво
	loaded bool
}

// NewResolver создаёт пустой Resolver.
func NewResolver() *Resolver {
	return &Resolver{cache: map[string]*Match{}}
}

// Find ищет населённый пункт по названию: точное совпадение (при нескольких
// одноимённых выбирается город), затем - город с расхождением в одну букву.
// Возвращает nil, если ничего не найдено.
func (r *Resolver) Find(name string) *Match {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	key := Norm(name)
	if m, ok := r.cache[key]; ok {
		return m
	}
	m := r.find(name, key)
	r.cache[key] = m
	return m
}

// IsLocality сообщает, найден ли населённый пункт с таким названием.
func (r *Resolver) IsLocality(name string) bool { return r.Find(name) != nil }

func (r *Resolver) find(name, key string) *Match {
	o := orm.NewOrm()
	var list []models.Locality
	qs := o.QueryTable("cmdb_locality")
	if _, err := qs.Filter("Name", name).Limit(500).All(&list); err != nil || len(list) == 0 {
		list = nil
		if _, err := o.QueryTable("cmdb_locality").Filter("Name__iexact", name).Limit(500).All(&list); err != nil {
			return nil
		}
	}
	if len(list) > 0 {
		ptrs := make([]*models.Locality, len(list))
		for i := range list {
			ptrs[i] = &list[i]
		}
		return &Match{Loc: Best(ptrs), Candidates: len(ptrs)}
	}

	if !r.loaded {
		r.loaded = true
		var cities []models.Locality
		if _, err := o.QueryTable("cmdb_locality").Filter("TypeShort", "г").Limit(-1).All(&cities); err == nil {
			for i := range cities {
				r.cities = append(r.cities, &cities[i])
			}
			sort.Slice(r.cities, func(i, j int) bool { return r.cities[i].Oktmo < r.cities[j].Oktmo })
		}
	}
	var found []*models.Locality
	for _, l := range r.cities {
		if OneEdit(key, Norm(l.Name)) {
			found = append(found, l)
		}
	}
	if len(found) == 0 {
		return nil
	}
	return &Match{Loc: Best(found), Candidates: len(found), Fuzzy: true}
}
