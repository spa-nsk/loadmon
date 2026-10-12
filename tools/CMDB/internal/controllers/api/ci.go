package api

import (
	"fmt"
	"strings"

	"cmdb/internal/services/changectx"
	"cmdb/internal/services/ci"
	"cmdb/internal/services/component"
	"cmdb/internal/services/history"
	"cmdb/internal/services/naming"
	"cmdb/internal/services/relation"
)

var registry = naming.DefaultRegistry()

// CIController — /api/v1/cis: список, карточка, создание, правка, смена стадии ЖЦ, компоненты,
// связи, история одной КЕ. Права и AS OF — как у internal/controllers/ui.CIController (тот же
// RBAC-фильтр и filters.AsOf), но ответы JSON и без HTMX.
type CIController struct {
	BaseController
}

func (c *CIController) listFilter() ci.ListFilter {
	f := ci.ListFilter{
		Class: c.GetString("class"), Location: c.GetString("location"), Env: c.GetString("env"),
		Stage: c.GetString("stage"), NamingStatus: c.GetString("naming"), Q: c.GetString("q"),
		Sort: c.GetString("sort", "name"), Desc: c.GetString("dir") == "desc",
	}
	f.Page, _ = c.GetInt("page", 1)
	f.PerPage, _ = c.GetInt("per_page", ci.PerPageOptions[0])
	valid := false
	for _, p := range ci.PerPageOptions {
		valid = valid || p == f.PerPage
	}
	if !valid {
		f.PerPage = ci.PerPageOptions[0]
	}
	return f
}

// List — GET /api/v1/cis: страница КЕ по фильтру, тот же набор параметров, что и у UI-списка
// (class, location, env, stage, naming, q, sort, dir, page, per_page).
func (c *CIController) List() {
	res, err := ci.List(c.ctx(), c.listFilter())
	c.must(err)
	c.writeJSON(200, map[string]any{"rows": res.Rows, "total": res.Total, "page": res.Page, "per_page": res.PerPage, "pages": res.Pages})
}

// Get — GET /api/v1/cis/{id}: карточка КЕ; с ?asof= — состояние на дату (internal/filters.AsOf).
func (c *CIController) Get() {
	d, err := ci.Get(c.ctx(), c.id())
	c.must(err)
	c.writeJSON(200, map[string]any{"ci": d.CI, "class_name": d.ClassName, "ext": d.Ext})
}

// createRequest — тело POST /api/v1/cis.
type createRequest struct {
	Class           string         `json:"class"`
	LocationCode    string         `json:"location_code"`
	CompanyCode     string         `json:"company_code"`
	EnvironmentCode string         `json:"environment_code"`
	SerialNumber    string         `json:"serial_number"`
	Description     string         `json:"description"`
	Criticality     string         `json:"criticality"`
	Attrs           map[string]any `json:"attrs"`
	Reason          string         `json:"reason"`
}

// attrsToStrings приводит типизированные значения JSON к строкам для ci.ClassSpec.ParseAttrs —
// той же проверки (обязательность, формат по типу поля), что у формы создания в UI, только
// значения приходят типизированными (bool/число), а не текстом из <input>.
func attrsToStrings(m map[string]any) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if v == nil {
			continue
		}
		out[k] = fmt.Sprint(v)
	}
	return out
}

// Create — POST /api/v1/cis: создаёт КЕ с именем по профилю именования (как ci.Create у UI),
// но записывает изменение с source=api (plan.md E7.3), не manual.
func (c *CIController) Create() {
	var req createRequest
	c.decodeJSON(&req)

	spec := ci.SpecFor(req.Class)
	if spec == nil {
		c.badRequest(fmt.Sprintf("неизвестный или не поддерживающий создание класс %q", req.Class))
	}
	c.requireCIWrite(spec.Class)
	if strings.TrimSpace(req.LocationCode) == "" {
		c.badRequest("не указана площадка (location_code)")
	}
	attrs, fieldErrs := spec.ParseAttrs(attrsToStrings(req.Attrs))
	if len(fieldErrs) > 0 {
		c.writeJSON(422, map[string]any{"error": "некорректные атрибуты класса", "fields": fieldErrs})
		c.StopRun()
	}

	res, err := ci.Create(c.ctx(), registry, ci.CreateInput{
		Class: spec.Class, LocationCode: req.LocationCode, CompanyCode: req.CompanyCode,
		EnvironmentCode: req.EnvironmentCode, SerialNumber: strings.TrimSpace(req.SerialNumber),
		Description: strings.TrimSpace(req.Description), Criticality: req.Criticality,
		Attrs: attrs, Reason: req.Reason, UserID: c.userID(), Source: changectx.SourceAPI,
	})
	c.must(err)
	c.writeJSON(201, map[string]any{"id": res.CIID, "name": res.Name, "change_set_id": res.ChangeSetID})
}

// updateRequest — тело PATCH /api/v1/cis/{id}. Указанное в JSON поле (даже null) меняется;
// отсутствующее — не трогается (различие держится на разборе тела в map, см. Update).
type updateRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	Reason          string `json:"reason"`
}

// Update — PATCH /api/v1/cis/{id}: правка изменяемых атрибутов КЕ с оптимистической блокировкой
// (row_version — expected_version). Присутствие поля в JSON — само по себе команда «изменить»
// (как «отмеченный чекбоксом» input у формы UI): чтобы сбросить значение в NULL, передайте null
// или "" — не отсутствие поля.
func (c *CIController) Update() {
	id := c.id()
	d, err := ci.Get(c.ctx(), id)
	c.must(err)
	c.requireCIWrite(d.CI["ci_class"])

	var body map[string]any
	c.decodeJSON(&body)

	in := ci.UpdateInput{CIID: id, Source: changectx.SourceAPI}
	if v, ok := body["expected_version"]; ok {
		in.ExpectedVersion = int(toFloat(v))
	}
	in.Reason, _ = body["reason"].(string)

	strField := func(key string) *string {
		v, ok := body[key]
		if !ok {
			return nil
		}
		s := ""
		if v != nil {
			s = fmt.Sprint(v)
		}
		return &s
	}
	in.Description = strField("description")
	in.Criticality = strField("criticality")
	in.EnvironmentCode = strField("environment_code")
	in.SerialNumber = strField("serial_number")
	in.AssetTag = strField("asset_tag")
	in.RackPosition = strField("rack_position")
	in.Ownership = strField("ownership")
	in.ServiceProvider = strField("service_provider")
	in.KIICategory = strField("kii_category")
	in.LifecycleStatus = strField("lifecycle_status")
	if v, ok := body["kii_object"]; ok {
		b, _ := v.(bool)
		in.KIIObject = &b
	}

	res, err := ci.Update(c.ctx(), in)
	c.must(err)
	c.writeJSON(200, map[string]any{"row_version": res.RowVersion, "change_set_id": res.ChangeSetID})
}

// toFloat читает число из значения, декодированного encoding/json (float64), не паникуя на прочих типах.
func toFloat(v any) float64 {
	f, _ := v.(float64)
	return f
}

// lifecycleRequest — тело POST /api/v1/cis/{id}/lifecycle.
type lifecycleRequest struct {
	ExpectedVersion int    `json:"expected_version"`
	ToStage         string `json:"to_stage"`
	Status          string `json:"status"`
	Reason          string `json:"reason"`
}

// Lifecycle — POST /api/v1/cis/{id}/lifecycle: переход стадии ЖЦ по матрице (ci.CanTransition).
func (c *CIController) Lifecycle() {
	id := c.id()
	d, err := ci.Get(c.ctx(), id)
	c.must(err)
	c.requireCIWrite(d.CI["ci_class"])

	var req lifecycleRequest
	c.decodeJSON(&req)
	csID, err := ci.ChangeLifecycle(c.ctx(), ci.ChangeLifecycleInput{
		CIID: id, ExpectedVersion: req.ExpectedVersion, ToStage: req.ToStage,
		Status: strings.TrimSpace(req.Status), Reason: req.Reason, UserID: c.userID(), Source: changectx.SourceAPI,
	})
	c.must(err)
	c.writeJSON(200, map[string]any{"change_set_id": csID})
}

// History — GET /api/v1/cis/{id}/history: последние change set, затронувшие КЕ (E3.7); ?limit=.
func (c *CIController) History() {
	limit, _ := c.GetInt("limit", 50)
	sets, err := history.ForCI(c.ctx(), c.id(), limit)
	c.must(err)
	c.writeJSON(200, map[string]any{"change_sets": sets})
}

// Components — GET /api/v1/cis/{id}/components: компоненты КЕ деревом (?asof= — на дату).
func (c *CIController) Components() {
	items, err := component.ListForCI(c.ctx(), c.id())
	c.must(err)
	out := make([]map[string]any, len(items))
	for i, it := range items {
		row := map[string]any{"depth": it.Depth}
		for k, v := range it.Row {
			row[k] = v
		}
		out[i] = row
	}
	c.writeJSON(200, map[string]any{"components": out})
}

// addComponentRequest — тело POST /api/v1/cis/{id}/components.
type addComponentRequest struct {
	ParentComponentID *int64   `json:"parent_component_id"`
	ComponentType     string   `json:"component_type"`
	Slot              string   `json:"slot"`
	Manufacturer      string   `json:"manufacturer"`
	PartNumber        string   `json:"part_number"`
	SerialNumber      string   `json:"serial_number"`
	Firmware          string   `json:"firmware"`
	CapacityValue     *float64 `json:"capacity_value"`
	CapacityUnit      string   `json:"capacity_unit"`
	Speed             string   `json:"speed"`
	Reason            string   `json:"reason"`
}

// AddComponent — POST /api/v1/cis/{id}/components: установка компонента (status=installed).
func (c *CIController) AddComponent() {
	id := c.id()
	d, err := ci.Get(c.ctx(), id)
	c.must(err)
	c.requireCIWrite(d.CI["ci_class"])

	var req addComponentRequest
	c.decodeJSON(&req)
	if strings.TrimSpace(req.ComponentType) == "" {
		c.badRequest("не указан тип компонента (component_type)")
	}
	compID, csID, err := component.Add(c.ctx(), component.AddInput{
		CIID: id, ParentComponentID: req.ParentComponentID, ComponentType: req.ComponentType, Slot: req.Slot,
		Spec: component.Spec{Manufacturer: req.Manufacturer, PartNumber: req.PartNumber, SerialNumber: req.SerialNumber,
			Firmware: req.Firmware, CapacityValue: req.CapacityValue, CapacityUnit: req.CapacityUnit, Speed: req.Speed},
		Reason: req.Reason, UserID: c.userID(), Source: changectx.SourceAPI,
	})
	c.must(err)
	c.writeJSON(201, map[string]any{"id": compID, "change_set_id": csID})
}

// Relations — GET /api/v1/cis/{id}/relations: связи КЕ в обе стороны, действующие на ?asof= (сейчас — по умолчанию).
func (c *CIController) Relations() {
	id := c.id()
	rels, err := relation.ListForCI(c.ctx(), id, nil)
	c.must(err)
	c.writeJSON(200, map[string]any{"relations": rels})
}

// addRelationRequest — тело POST /api/v1/cis/{id}/relations; CIID в адресе — источник связи (src).
type addRelationRequest struct {
	RelType string         `json:"rel_type"`
	DstCIID int64          `json:"dst_ci_id"`
	Attrs   map[string]any `json:"attrs"`
	Reason  string         `json:"reason"`
}

// AddRelation — POST /api/v1/cis/{id}/relations: создаёт связь src(id) --rel_type--> dst_ci_id.
func (c *CIController) AddRelation() {
	id := c.id()
	d, err := ci.Get(c.ctx(), id)
	c.must(err)
	c.requireCIWrite(d.CI["ci_class"])

	var req addRelationRequest
	c.decodeJSON(&req)
	if strings.TrimSpace(req.RelType) == "" || req.DstCIID == 0 {
		c.badRequest("нужны rel_type и dst_ci_id")
	}
	relID, csID, err := relation.Create(c.ctx(), relation.CreateInput{
		RelType: req.RelType, SrcCIID: id, DstCIID: req.DstCIID, Attrs: req.Attrs,
		Reason: req.Reason, UserID: c.userID(), Source: changectx.SourceAPI,
	})
	c.must(err)
	c.writeJSON(201, map[string]any{"id": relID, "change_set_id": csID})
}
