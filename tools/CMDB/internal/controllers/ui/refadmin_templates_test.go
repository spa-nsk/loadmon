package ui

import (
	"strings"
	"testing"

	"cmdb/internal/repo"
	"cmdb/internal/services/refadmin"
)

// Рендерит шаблоны справочников (E1.8) без БД на всех ~20 TableSpec, чтобы поймать ошибки
// шаблона (неверные имена полей/функций) для каждого типа поля — так же, как templates_test.go
// делает для карточки КЕ.

func TestTemplates_RefAdminIndex(t *testing.T) {
	dir := viewsDir(t)
	out := render(t, dir, "admin/ref_index.tpl", tplData{"Specs": refadmin.All})
	for _, s := range refadmin.All {
		if !strings.Contains(out, s.Title) {
			t.Errorf("ref_index.tpl: нет ссылки на %q", s.Title)
		}
	}
}

// fakeValuesFor — по одному правдоподобному текстовому значению на каждое нескрытое поле спецификации.
func fakeValuesFor(spec refadmin.TableSpec) map[string]string {
	v := map[string]string{}
	for _, f := range spec.Fields {
		switch f.Type {
		case refadmin.TypeBool:
			v[f.Name] = "true"
		case refadmin.TypeJSONB:
			v[f.Name] = `{"a":1}`
		case refadmin.TypeTextArr:
			v[f.Name] = "{a,b}"
		default:
			v[f.Name] = "x"
		}
	}
	return v
}

func fakeFKOptions(spec refadmin.TableSpec) map[string][]refadmin.FKOption {
	out := map[string][]refadmin.FKOption{}
	for _, f := range spec.Fields {
		if f.FK != nil {
			out[f.Name] = []refadmin.FKOption{{Value: "x", Label: "X"}}
		}
	}
	return out
}

// TestTemplates_RefAdminAllSpecs рендерит список и форму (создание и правка) для каждого из ~20
// справочников — ловит ошибки шаблона на каждом сочетании типов полей (bool/select/jsonb/text[]/date/…).
func TestTemplates_RefAdminAllSpecs(t *testing.T) {
	dir := viewsDir(t)
	for _, spec := range refadmin.All {
		t.Run(spec.Slug, func(t *testing.T) {
			values := fakeValuesFor(spec)
			fk := fakeFKOptions(spec)

			listData := tplData{"Spec": spec, "Rows": []repo.Row{values}, "Q": "", "Specs": refadmin.All, "Nav": "admin"}
			render(t, dir, "admin/ref_list.tpl", listData)
			render(t, dir, "admin/_ref_results.tpl", listData)

			create := tplData{"Spec": spec, "Values": map[string]string{}, "FKOptions": fk, "IsCreate": true, "Nav": "admin"}
			render(t, dir, "admin/ref_form.tpl", create)

			edit := tplData{"Spec": spec, "Values": values, "PK": "x", "FKOptions": fk, "IsCreate": false, "Nav": "admin", "FormError": "тестовая ошибка"}
			render(t, dir, "admin/ref_form.tpl", edit)
		})
	}
}

func TestTemplates_RefAdminEmptyResults(t *testing.T) {
	dir := viewsDir(t)
	spec, ok := refadmin.Spec("location")
	if !ok {
		t.Fatal("спецификация location не найдена")
	}
	out := render(t, dir, "admin/_ref_results.tpl", tplData{"Spec": spec, "Rows": []repo.Row{}})
	if !strings.Contains(out, "Ничего не найдено") {
		t.Error("_ref_results.tpl: нет сообщения о пустом результате")
	}
}
