package itilium

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// Справочник полей КЕ по cuClass.
// Источник сейчас — статический JSON (выгрузка «Список полей КЕ.xls»).
// В перспективе тот же контракт FieldCatalog будет закрывать HTTP-endpoint
// 1С (динамический перечень полей); клиентский код должен брать fields=
// через FieldCodes/DefaultFieldsForClass, а не хардкодить списки.

//go:embed cu_fields_catalog.json
var cuFieldsCatalogJSON []byte

// CUFieldDef — одно поле КЕ: код для fields= и человекочитаемый label.
type CUFieldDef struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// CUClassFields — класс КЕ и его поля.
type CUClassFields struct {
	Code   int          `json:"code"`
	Name   string       `json:"name"`
	Fields []CUFieldDef `json:"fields"`
}

// FieldCatalog — источник справочника полей (static JSON или будущий API).
type FieldCatalog interface {
	Classes() []CUClassFields
	Fields(cuClass int) []CUFieldDef
	FieldCodes(cuClass int) []string
	ClassName(cuClass int) string
}

type catalogFile struct {
	Source  string          `json:"source"`
	Classes []CUClassFields `json:"classes"`
}

type staticFieldCatalog struct {
	source  string
	byCode  map[int]CUClassFields
	ordered []CUClassFields
}

var (
	defaultCatalog     FieldCatalog
	defaultCatalogOnce sync.Once
	defaultCatalogErr  error
)

// DefaultFieldCatalog возвращает встроенный справочник (из xls-выгрузки).
func DefaultFieldCatalog() FieldCatalog {
	defaultCatalogOnce.Do(func() {
		defaultCatalog, defaultCatalogErr = LoadFieldCatalogJSON(cuFieldsCatalogJSON)
	})
	if defaultCatalogErr != nil {
		// не должно случаться с embed; пустой каталог безопаснее panic в init
		return &staticFieldCatalog{byCode: map[int]CUClassFields{}}
	}
	return defaultCatalog
}

// LoadFieldCatalogJSON разбирает JSON формата {source, classes:[…]}.
func LoadFieldCatalogJSON(data []byte) (FieldCatalog, error) {
	var f catalogFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("itilium: field catalog: %w", err)
	}
	c := &staticFieldCatalog{
		source: f.Source,
		byCode: make(map[int]CUClassFields, len(f.Classes)),
	}
	for _, cl := range f.Classes {
		c.byCode[cl.Code] = cl
		c.ordered = append(c.ordered, cl)
	}
	sort.SliceStable(c.ordered, func(i, j int) bool { return c.ordered[i].Code < c.ordered[j].Code })
	return c, nil
}

func (c *staticFieldCatalog) Classes() []CUClassFields {
	out := make([]CUClassFields, len(c.ordered))
	copy(out, c.ordered)
	return out
}

func (c *staticFieldCatalog) Fields(cuClass int) []CUFieldDef {
	cl, ok := c.byCode[cuClass]
	if !ok {
		return nil
	}
	out := make([]CUFieldDef, len(cl.Fields))
	copy(out, cl.Fields)
	return out
}

func (c *staticFieldCatalog) FieldCodes(cuClass int) []string {
	fs := c.Fields(cuClass)
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		if f.Code != "" {
			out = append(out, f.Code)
		}
	}
	return out
}

func (c *staticFieldCatalog) ClassName(cuClass int) string {
	if cl, ok := c.byCode[cuClass]; ok {
		return cl.Name
	}
	return ""
}

// SupplementalCUFields — поля, которые встречаются в ответах getCU / нужны
// импорту, но отсутствуют в статическом «Список полей КЕ.xls» (например
// linksUnit, modelVendS, units). При появлении динамического endpoint
// их можно убрать, если API начнёт отдавать полный перечень.
var SupplementalCUFields = map[int][]string{
	CUClassServers: {
		"linksUnit", "modelVendS", "units",
		"statusHistory", "statusHistoryYY",
		"currency", "currencyA", "dateChPriA", "dateChPric",
		"isActual", "priceA", "priceCommA", "priceComme", "priceGPL",
	},
}

// DefaultFieldsForClass — codes для query fields= : каталог + supplemental.
// Базовые code/status/name/… getCU часто отдаёт и без fields; здесь явный
// полный набор для импорта.
func DefaultFieldsForClass(cuClass int) []string {
	return DefaultFieldsForClassFrom(DefaultFieldCatalog(), cuClass)
}

// DefaultFieldsForClassFrom — то же с произвольным каталогом (тесты / mock API).
func DefaultFieldsForClassFrom(cat FieldCatalog, cuClass int) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(codes ...string) {
		for _, c := range codes {
			if c == "" {
				continue
			}
			if _, ok := seen[c]; ok {
				continue
			}
			seen[c] = struct{}{}
			out = append(out, c)
		}
	}
	// Базовые идентификаторы/статус — всегда полезны в fields=.
	add("code", "status", "name", "description", "author")
	if cat != nil {
		add(cat.FieldCodes(cuClass)...)
	}
	add(SupplementalCUFields[cuClass]...)
	return out
}
